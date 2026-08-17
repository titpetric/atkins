# Package ./server/stream

```go
import (
	"github.com/titpetric/atkins/server/stream"
}
```

Package stream carries a running job's terminal between the agent
that runs it and the browsers watching it.

Two directions, and they are not symmetric.

Output already has a home: every chunk an agent posts is stored, and
the job page renders the stored rows. What the stored rows cannot do
is arrive — a page reading them polls, and a terminal that repaints
every three seconds is not a terminal. So a chunk is published here on
its way past, and a watcher gets it as it lands. Nothing here is the
record of what a job printed; that is the database, and a watcher that
falls behind is dropped and re-reads it rather than being waited for.

Input has no home at all. Keystrokes are worth nothing a second after
they were typed and worth nothing to anybody but the process they were
typed at, so they are queued in memory for the agent's next poll and
forgotten. A job nobody is typing at never allocates a queue.

The whole thing is per-process on purpose. Two servers behind one load
balancer would need the agent and the browser to land on the same one;
the queue is the sort of state that belongs in whatever those two
deployments already share, and pretending otherwise here would be a
distributed system hiding in a map.

## Types

```go
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
```

```go
// Hub owns the live channels, one per job that has one.
type Hub struct {
	mu   sync.Mutex
	jobs map[string]*channel
}
```

## Function symbols

- `func New () *Hub`
- `func (*Hub) Close (jobID string)`
- `func (*Hub) Publish (jobID string, chunk Chunk)`
- `func (*Hub) Receive (ctx context.Context, jobID string, timeout time.Duration) []byte`
- `func (*Hub) Send (jobID string, input []byte)`
- `func (*Hub) Sweep (idle time.Duration) int`
- `func (*Hub) Watch (jobID string) (<-chan Chunk, func())`

### New

New returns an empty Hub.

```go
func New() *Hub
```

### Close

Close ends a job's live channel: watchers are told to stop, pollers
return, and the state is forgotten.

It is called when a job settles. Every channel the hub holds is
reachable from a job row, and a job row always settles — an agent that
dies takes its lease with it and the reclaim sweep finishes the job —
so there is no channel that outlives the process for want of a caller.

```go
func (*Hub) Close(jobID string)
```

### Publish

Publish hands a chunk of output to everyone watching a job.

It never blocks and it never fails. A watcher whose buffer is full is
disconnected instead of waited for: the chunk is already in the
database, and the browser's reconnect replays it. The agent posting
output must not be slowed by a page nobody is looking at.

```go
func (*Hub) Publish(jobID string, chunk Chunk)
```

### Receive

Receive collects the input queued for a job, waiting up to timeout for
some to arrive.

An agent calls this in a loop while it holds a job. Returning nothing
is the normal outcome — a build nobody is typing at — and the agent
simply calls again. The wait is what keeps that loop from being a
busy poll while still bounding how long one request is held open.

```go
func (*Hub) Receive(ctx context.Context, jobID string, timeout time.Duration) []byte
```

### Send

Send queues keystrokes for the agent running a job.

Input past the queue bound is dropped rather than blocking the
browser: the agent collects continuously, so a full queue means
nothing is running to collect it, and holding the request open would
only move the problem.

```go
func (*Hub) Send(jobID string, input []byte)
```

### Sweep

Sweep closes the channels nobody is using any more, and returns how
many it closed.

Close on a settled job is the ordinary way a channel ends, and it
covers the jobs that settle through a handler. It does not cover the
rest: a job reclaimed in bulk when its agent died, a browser that
opened a terminal on a job which settled a moment later, a poll for a
job id that never existed. Each of those leaves a small struct in the
map and no caller who knows to remove it.

So the hub is swept on a timer instead, the way the leases are. A
channel with a watcher on it is never swept however quiet it is —
that is a build printing nothing, not a channel nobody wants — which
leaves exactly the ones with no reader and nothing to read.

```go
func (*Hub) Sweep(idle time.Duration) int
```

### Watch

Watch subscribes to a job's output.

The returned channel closing is the caller's only stop signal, and it
covers every way watching ends: the job settled, the watcher fell too
far behind to catch up, or the hub swept a channel nobody was using.
The cancel function unsubscribes and must be called; a watcher that is
never cancelled is a chunk written to nobody on every publish.

```go
func (*Hub) Watch(jobID string) (<-chan Chunk, func())
```
