// Package stream carries a running job's terminal between the agent
// that runs it and the browsers watching it.
//
// Two directions, and they are not symmetric.
//
// Output already has a home: every chunk an agent posts is stored, and
// the job page renders the stored rows. What the stored rows cannot do
// is arrive — a page reading them polls, and a terminal that repaints
// every three seconds is not a terminal. So a chunk is published here on
// its way past, and a watcher gets it as it lands. Nothing here is the
// record of what a job printed; that is the database, and a watcher that
// falls behind is dropped and re-reads it rather than being waited for.
//
// Input has no home at all. Keystrokes are worth nothing a second after
// they were typed and worth nothing to anybody but the process they were
// typed at, so they are queued in memory for the agent's next poll and
// forgotten. A job nobody is typing at never allocates a queue.
//
// The whole thing is per-process on purpose. Two servers behind one load
// balancer would need the agent and the browser to land on the same one;
// the queue is the sort of state that belongs in whatever those two
// deployments already share, and pretending otherwise here would be a
// distributed system hiding in a map.
package stream

import (
	"context"
	"sync"
	"time"
)

// Chunk is one piece of a job's output, with the sequence number the
// stored row carries.
//
// The sequence is what makes a reconnect clean: a watcher replays the
// stored rows, learns the last sequence it saw, and ignores anything at
// or below it from the live feed. Without it, subscribing and then
// reading the table would either duplicate the chunks written in between
// or lose them, depending on which order the two happened in.
//
// The tags are not decoration: a chunk is marshalled straight into a
// server-sent event and read by the terminal page, so these names are
// the wire format that page is written against.
type Chunk struct {
	Seq     int64  `json:"seq"`
	Stream  string `json:"stream"`
	Content string `json:"content"`
}

// watcherBuffer is how far behind a browser may fall before it is
// dropped and told to start again.
//
// It is generous — a build printing steadily fills it in seconds only if
// nothing is draining it — because dropping a watcher costs a full
// replay. It is bounded at all because the alternative is an agent's log
// POST blocking on a browser somebody closed the laptop lid on.
const watcherBuffer = 512

// maxInputQueue bounds the keystrokes held for one job.
//
// A terminal's input is a person typing, so this is enormous by that
// measure and small by every other. It exists because the queue is fed
// by an HTTP endpoint: a script pointed at it should fill a bounded
// buffer and stop, not the server's memory.
const maxInputQueue = 64 * 1024

// Hub owns the live channels, one per job that has one.
type Hub struct {
	mu   sync.Mutex
	jobs map[string]*channel
}

// New returns an empty Hub.
func New() *Hub {
	return &Hub{jobs: map[string]*channel{}}
}

// channel is one job's live state.
type channel struct {
	mu sync.Mutex

	// watchers are the browsers following the output. The value is the
	// buffered channel each one reads.
	watchers map[chan Chunk]struct{}

	// input is what has been typed and not yet collected by the agent.
	input []byte

	// arrived is closed and replaced whenever input lands, which is how
	// a polling agent waits without spinning.
	arrived chan struct{}

	// done is closed when the job settles, so watchers and pollers stop
	// rather than hold a request open on a job that will never speak
	// again.
	done chan struct{}

	// touched is when this channel last carried anything. Sweep reads it
	// to find the ones nobody settled.
	touched time.Time
}

// Publish hands a chunk of output to everyone watching a job.
//
// It never blocks and it never fails. A watcher whose buffer is full is
// disconnected instead of waited for: the chunk is already in the
// database, and the browser's reconnect replays it. The agent posting
// output must not be slowed by a page nobody is looking at.
func (h *Hub) Publish(jobID string, chunk Chunk) {
	channel := h.existing(jobID)
	if channel == nil {
		return
	}

	channel.mu.Lock()
	defer channel.mu.Unlock()

	channel.touched = time.Now()

	for watcher := range channel.watchers {
		select {
		case watcher <- chunk:
		default:
			// Too far behind to catch up. Closing ends its request; the
			// browser reconnects and replays from the stored rows.
			delete(channel.watchers, watcher)
			close(watcher)
		}
	}
}

// Watch subscribes to a job's output.
//
// The returned channel closing is the caller's only stop signal, and it
// covers every way watching ends: the job settled, the watcher fell too
// far behind to catch up, or the hub swept a channel nobody was using.
// The cancel function unsubscribes and must be called; a watcher that is
// never cancelled is a chunk written to nobody on every publish.
func (h *Hub) Watch(jobID string) (<-chan Chunk, func()) {
	channel := h.channel(jobID)
	watcher := make(chan Chunk, watcherBuffer)

	channel.mu.Lock()
	channel.watchers[watcher] = struct{}{}
	channel.touched = time.Now()
	channel.mu.Unlock()

	return watcher, func() {
		channel.mu.Lock()
		defer channel.mu.Unlock()

		if _, live := channel.watchers[watcher]; live {
			delete(channel.watchers, watcher)
			close(watcher)
		}
	}
}

// Send queues keystrokes for the agent running a job.
//
// Input past the queue bound is dropped rather than blocking the
// browser: the agent collects continuously, so a full queue means
// nothing is running to collect it, and holding the request open would
// only move the problem.
func (h *Hub) Send(jobID string, input []byte) {
	if len(input) == 0 {
		return
	}

	channel := h.channel(jobID)

	channel.mu.Lock()
	defer channel.mu.Unlock()

	channel.touched = time.Now()

	if room := maxInputQueue - len(channel.input); room < len(input) {
		if room <= 0 {
			return
		}
		input = input[:room]
	}
	channel.input = append(channel.input, input...)

	// Wake whoever is polling. The channel is replaced rather than
	// reused so the next waiter gets a fresh one to block on.
	close(channel.arrived)
	channel.arrived = make(chan struct{})
}

// Receive collects the input queued for a job, waiting up to timeout for
// some to arrive.
//
// An agent calls this in a loop while it holds a job. Returning nothing
// is the normal outcome — a build nobody is typing at — and the agent
// simply calls again. The wait is what keeps that loop from being a
// busy poll while still bounding how long one request is held open.
func (h *Hub) Receive(ctx context.Context, jobID string, timeout time.Duration) []byte {
	channel := h.channel(jobID)

	if input := channel.take(); len(input) > 0 {
		return input
	}

	channel.mu.Lock()
	arrived := channel.arrived
	channel.touched = time.Now()
	channel.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-arrived:
		return channel.take()
	case <-channel.done:
		return channel.take()
	case <-ctx.Done():
		return nil
	case <-timer.C:
		return nil
	}
}

// Close ends a job's live channel: watchers are told to stop, pollers
// return, and the state is forgotten.
//
// It is called when a job settles. Every channel the hub holds is
// reachable from a job row, and a job row always settles — an agent that
// dies takes its lease with it and the reclaim sweep finishes the job —
// so there is no channel that outlives the process for want of a caller.
func (h *Hub) Close(jobID string) {
	h.mu.Lock()
	channel, found := h.jobs[jobID]
	delete(h.jobs, jobID)
	h.mu.Unlock()

	if !found {
		return
	}

	channel.mu.Lock()
	defer channel.mu.Unlock()

	for watcher := range channel.watchers {
		delete(channel.watchers, watcher)
		close(watcher)
	}
	close(channel.done)
}

// Sweep closes the channels nobody is using any more, and returns how
// many it closed.
//
// Close on a settled job is the ordinary way a channel ends, and it
// covers the jobs that settle through a handler. It does not cover the
// rest: a job reclaimed in bulk when its agent died, a browser that
// opened a terminal on a job which settled a moment later, a poll for a
// job id that never existed. Each of those leaves a small struct in the
// map and no caller who knows to remove it.
//
// So the hub is swept on a timer instead, the way the leases are. A
// channel with a watcher on it is never swept however quiet it is —
// that is a build printing nothing, not a channel nobody wants — which
// leaves exactly the ones with no reader and nothing to read.
func (h *Hub) Sweep(idle time.Duration) int {
	if idle <= 0 {
		return 0
	}

	deadline := time.Now().Add(-idle)
	swept := 0

	h.mu.Lock()
	stale := make([]*channel, 0, len(h.jobs))
	for jobID, candidate := range h.jobs {
		candidate.mu.Lock()
		quiet := len(candidate.watchers) == 0 && candidate.touched.Before(deadline)
		candidate.mu.Unlock()

		if quiet {
			stale = append(stale, candidate)
			delete(h.jobs, jobID)
			swept++
		}
	}
	h.mu.Unlock()

	for _, candidate := range stale {
		close(candidate.done)
	}

	return swept
}

// take empties the input queue.
func (c *channel) take() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.input) == 0 {
		return nil
	}

	input := c.input
	c.input = nil

	return input
}

// channel returns a job's channel, creating it on first use.
func (h *Hub) channel(jobID string) *channel {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, found := h.jobs[jobID]; found {
		return existing
	}

	created := &channel{
		watchers: map[chan Chunk]struct{}{},
		arrived:  make(chan struct{}),
		done:     make(chan struct{}),
		touched:  time.Now(),
	}
	h.jobs[jobID] = created

	return created
}

// existing returns a job's channel without creating one.
//
// Publish uses it so an agent posting output for a job nobody is
// watching — which is most jobs — allocates nothing.
func (h *Hub) existing(jobID string) *channel {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.jobs[jobID]
}
