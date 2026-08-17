package stream_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/server/stream"
)

// wait is how long a test gives a channel before deciding nothing is
// coming. Everything here is in-process, so this is a deadlock detector
// rather than a timing assumption.
const wait = 2 * time.Second

func TestWatcherGetsWhatIsPublished(t *testing.T) {
	hub := stream.New()

	watcher, cancel := hub.Watch("job")
	defer cancel()

	hub.Publish("job", stream.Chunk{Seq: 0, Content: "hello"})

	select {
	case chunk := <-watcher:
		assert.Equal(t, "hello", chunk.Content)
	case <-time.After(wait):
		t.Fatal("published chunk never arrived")
	}
}

// Most jobs are watched by nobody, and publishing for one of those must
// not leave a channel behind for every job the instance ever ran.
func TestPublishingToNobodyAllocatesNothing(t *testing.T) {
	hub := stream.New()

	hub.Publish("job", stream.Chunk{Content: "hello"})

	assert.Equal(t, 0, hub.Sweep(0))
	assert.Equal(t, 0, hub.Sweep(-time.Second))
}

// A watcher that stops reading must not slow down the agent posting
// output. It is dropped instead, and the browser replays from the
// stored rows when it reconnects.
func TestASlowWatcherIsDroppedRatherThanWaitedFor(t *testing.T) {
	hub := stream.New()

	watcher, cancel := hub.Watch("job")
	defer cancel()

	// Far past the buffer, and nothing is reading.
	for i := range 4096 {
		hub.Publish("job", stream.Chunk{Seq: int64(i), Content: "x"})
	}

	// Drain to the close: what matters is that publishing returned at
	// all, and that the watcher was ended rather than left half fed.
	closed := false
	for range 4096 {
		if _, open := <-watcher; !open {
			closed = true
			break
		}
	}
	assert.True(t, closed, "a watcher that stopped reading was not dropped")
}

func TestCancelUnsubscribesAndIsSafeTwice(t *testing.T) {
	hub := stream.New()

	watcher, cancel := hub.Watch("job")
	cancel()

	_, open := <-watcher
	assert.False(t, open, "cancel should close the watcher")

	// The handler defers this, and the job settling may have closed the
	// channel first.
	assert.NotPanics(t, cancel)
}

func TestCloseEndsEveryWatcher(t *testing.T) {
	hub := stream.New()

	first, cancelFirst := hub.Watch("job")
	second, cancelSecond := hub.Watch("job")
	defer cancelFirst()
	defer cancelSecond()

	hub.Close("job")

	_, open := <-first
	assert.False(t, open)
	_, open = <-second
	assert.False(t, open)

	// Closing a job that has already been closed, or one that never had
	// a channel, is what a second status report and a cancel look like.
	assert.NotPanics(t, func() {
		hub.Close("job")
		hub.Close("never-seen")
	})
}

func TestReceiveCollectsWhatWasSent(t *testing.T) {
	hub := stream.New()

	hub.Send("job", []byte("ls"))
	hub.Send("job", []byte(" -la\r"))

	assert.Equal(t, "ls -la\r", string(hub.Receive(t.Context(), "job", wait)))

	// The queue is emptied by collecting it, so the next poll waits
	// rather than handing the same keystrokes over twice.
	assert.Empty(t, hub.Receive(t.Context(), "job", time.Millisecond))
}

// The agent polls in a loop, and the loop only works if a keystroke that
// lands mid-wait wakes it.
func TestReceiveWakesOnAKeystroke(t *testing.T) {
	hub := stream.New()

	go func() {
		time.Sleep(20 * time.Millisecond)
		hub.Send("job", []byte("q"))
	}()

	started := time.Now()
	collected := hub.Receive(t.Context(), "job", wait)

	assert.Equal(t, "q", string(collected))
	assert.Less(t, time.Since(started), wait, "the poll waited out its timeout instead of waking")
}

func TestReceiveReturnsWhenTheJobSettles(t *testing.T) {
	hub := stream.New()

	// Create the channel first, so the close below is the one the
	// waiter is sitting on.
	require.Empty(t, hub.Receive(t.Context(), "job", time.Millisecond))

	go func() {
		time.Sleep(20 * time.Millisecond)
		hub.Close("job")
	}()

	started := time.Now()
	hub.Receive(t.Context(), "job", wait)

	assert.Less(t, time.Since(started), wait, "a settled job left its agent waiting")
}

func TestReceiveReturnsWhenTheRequestGoesAway(t *testing.T) {
	hub := stream.New()

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	assert.Empty(t, hub.Receive(ctx, "job", wait))
	assert.Less(t, time.Since(started), wait)
}

// The queue is fed by an HTTP endpoint. A script pointed at it should
// fill a bounded buffer and stop.
func TestInputIsBounded(t *testing.T) {
	hub := stream.New()

	flood := make([]byte, 1<<20)
	for range 4 {
		hub.Send("job", flood)
	}

	collected := hub.Receive(t.Context(), "job", time.Millisecond)
	assert.NotEmpty(t, collected)
	assert.Less(t, len(collected), len(flood), "the input queue grew past its bound")
}

// Sweep is the backstop for the channels no handler settles: a job
// reclaimed in bulk, a terminal opened on a job that had already
// finished.
func TestSweepForgetsChannelsNobodyIsUsing(t *testing.T) {
	hub := stream.New()

	hub.Send("abandoned", []byte("x"))

	// Nothing is old enough yet.
	assert.Equal(t, 0, hub.Sweep(time.Hour))

	assert.Equal(t, 1, hub.Sweep(time.Nanosecond))
	assert.Equal(t, 0, hub.Sweep(time.Nanosecond))
}

// A build that prints nothing for an hour is still a build somebody is
// watching, and sweeping it would close their terminal.
func TestSweepKeepsAChannelSomebodyIsWatching(t *testing.T) {
	hub := stream.New()

	watcher, cancel := hub.Watch("job")
	defer cancel()

	assert.Equal(t, 0, hub.Sweep(time.Nanosecond))

	select {
	case _, open := <-watcher:
		assert.True(t, open, "a watched channel was swept")
	default:
	}
}

// Zero and negative disable the sweep, which is what a module with no
// interval configured passes.
func TestSweepIsDisabledByANonPositiveIdle(t *testing.T) {
	hub := stream.New()

	hub.Send("job", []byte("x"))

	assert.Equal(t, 0, hub.Sweep(0))
	assert.Equal(t, "x", string(hub.Receive(t.Context(), "job", time.Millisecond)))
}
