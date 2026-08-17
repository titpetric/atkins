package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/stream"
)

// The job page renders what a command printed. It cannot render what a
// command is doing, because atkins draws its tree by moving the cursor
// and clearing lines, and a document has no cursor: the page drops those
// sequences and prints the rest, which is the right answer for something
// read after the fact and the wrong one for something watched.
//
// So there is a second page with a terminal emulator on it. The
// sequences go through untouched and are interpreted by something that
// understands them, the tree redraws in place, and a running job looks
// the way it looks in a shell.
//
// Output arrives by server-sent events, which is the whole of what this
// needs: one direction, text, and a browser that reconnects by itself.
// Input goes back the other way as ordinary posts, because keystrokes
// are rare and small and a request each costs nothing. Between them they
// do what a websocket would have done here, without the dependency or
// the proxy configuration.

// terminalPage is the view model for the terminal.
type terminalPage struct {
	Job        *model.Job
	Repository *model.Repository

	// Link is the job's document page, which is where somebody goes to
	// read the run afterwards, download its artefacts, or retry it.
	Link string

	// Stream and Input are where this page's script talks to, tokens and
	// all. They are built here rather than in the markup because whether
	// a link carries a token is a runtime setting.
	Stream string
	Input  string

	// Interactive reports that the job's command reads a terminal, which
	// is what decides whether this one types back.
	Interactive bool
}

// Terminal renders the live view of one job.
func (h *Handlers) Terminal(w http.ResponseWriter, r *http.Request) {
	job, err := h.viewableJob(w, r)
	if err != nil {
		return
	}

	page := &terminalPage{
		Job:         job,
		Link:        h.links().Job(job.ID),
		Stream:      h.links().withToken("/job/"+job.ID+"/stream", job.ID),
		Input:       h.links().withToken("/job/"+job.ID+"/input", job.ID),
		Interactive: job.Interactive,
	}

	if repository, err := h.repositories.Get(r.Context(), job.RepositoryID); err == nil {
		page.Repository = repository
	}

	h.render(w, r, terminalView(page))
}

// streamKeepalive is how often a comment is written down an idle stream.
//
// It exists for the hops in between: a proxy with an idle read timeout
// closes a connection nothing has been written to, and a build that
// prints nothing for a minute is a normal build. A comment line is
// ignored by the client and resets everybody's timer.
const streamKeepalive = 20 * time.Second

// StreamJob sends a job's output as it arrives.
//
// The stored rows are replayed first, so a browser that connects late —
// or reconnects — sees the run from the beginning rather than from
// whenever it happened to arrive. The sequence numbers make the join
// clean: the live feed is subscribed to before the table is read, and
// anything it offers at or below the last replayed sequence is a chunk
// the replay already covered.
func (h *Handlers) StreamJob(w http.ResponseWriter, r *http.Request) {
	job, err := h.viewableJob(w, r)
	if err != nil {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.fail(w, r, http.StatusInternalServerError, errors.New("this server cannot stream"))
		return
	}

	// Subscribe before reading the table. The other order loses whatever
	// is written in between; this one duplicates it, and duplicates are
	// what the sequence number is for.
	var (
		live   <-chan stream.Chunk
		cancel = func() {}
	)
	if h.live != nil && !job.IsTerminal() {
		live, cancel = h.live.Watch(job.ID)
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	// A stream a proxy has decided to buffer is a stream that arrives all
	// at once at the end, which is the one thing this page must not do.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	replayed := int64(-1)
	if entries, err := h.jobLogs.List(r.Context(), job.ID); err == nil {
		for _, entry := range entries {
			if !writeChunk(w, stream.Chunk{Seq: entry.Seq, Stream: entry.Stream, Content: entry.Content}) {
				return
			}
			replayed = entry.Seq
		}
		flusher.Flush()
	}

	// A job that has already settled has nothing more to say. Telling
	// the client so is what stops it reconnecting for ever.
	if live == nil {
		writeEvent(w, "end", job.Status)
		flusher.Flush()
		return
	}

	keepalive := time.NewTicker(streamKeepalive)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case chunk, open := <-live:
			if !open {
				// The job settled, or this watcher fell behind. Either
				// way the client reconnects and finds out which.
				writeEvent(w, "end", "")
				flusher.Flush()
				return
			}
			if chunk.Seq <= replayed {
				continue
			}
			if !writeChunk(w, chunk) {
				return
			}
			replayed = chunk.Seq
			flusher.Flush()

		case <-keepalive.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// maxInputBody bounds one post of keystrokes. A person types; this is
// the size at which something else is.
const maxInputBody = 8 * 1024

// InputJob queues what somebody typed at a running job.
//
// It answers 204 whatever happens to the bytes. The queue is bounded and
// the agent may have stopped collecting a moment ago, and a terminal
// that popped up an error every time a keystroke arrived a moment too
// late would be unusable; what a person sees is the same thing they see
// in a real terminal, which is that the character did not echo.
func (h *Handlers) InputJob(w http.ResponseWriter, r *http.Request) {
	job, err := h.viewableJob(w, r)
	if err != nil {
		return
	}

	// A job that never asked for stdin does not get any. The terminal
	// hides its keyboard for one, and this is the check behind that.
	if !job.Interactive || job.IsTerminal() || h.live == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// The token in the URL is what authorises this, and a cross-origin
	// post cannot read it out of a link somebody else was given. The
	// origin check is the same one the forms use, for the same reason.
	if err := h.sameOrigin(r); err != nil {
		h.fail(w, r, http.StatusForbidden, err)
		return
	}

	typed, err := io.ReadAll(io.LimitReader(r.Body, maxInputBody))
	if err != nil {
		h.fail(w, r, http.StatusBadRequest, err)
		return
	}

	h.live.Send(job.ID, typed)

	w.WriteHeader(http.StatusNoContent)
}

// viewableJob loads a job for a page gated on the view token, reporting
// the failure itself.
//
// It is the job page's own check, factored out because the terminal, the
// stream and the input endpoint are the same page and have to agree
// about who may see it. The token is checked before the job is loaded,
// so a wrong one discloses nothing about whether the job exists.
func (h *Handlers) viewableJob(w http.ResponseWriter, r *http.Request) (*model.Job, error) {
	jobID := platform.URLParam(r, "jobID")

	if !h.public() && !h.tokens.ValidViewToken(jobID, platform.QueryParam(r, ViewTokenParam)) {
		err := errors.New("this job link is missing its access token; use the whole URL atkins printed")
		h.fail(w, r, http.StatusForbidden, err)
		return nil, err
	}

	job, err := h.jobs.Get(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.New("no such job")
			h.fail(w, r, http.StatusNotFound, err)
			return nil, err
		}
		h.fail(w, r, http.StatusInternalServerError, err)
		return nil, err
	}

	return job, nil
}

// writeChunk sends one piece of output as an event, reporting whether
// the client is still there.
func writeChunk(w http.ResponseWriter, chunk stream.Chunk) bool {
	encoded, err := json.Marshal(chunk)
	if err != nil {
		return true
	}

	return writeEvent(w, "output", string(encoded))
}

// writeEvent writes one server-sent event.
//
// The payload is JSON on a single line, which sidesteps the format's one
// sharp edge: a data field is newline-delimited, so anything containing
// a newline — which is all of a build's output — would otherwise have to
// be split across fields and rejoined by the client.
func writeEvent(w http.ResponseWriter, name, payload string) bool {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload)
	return err == nil
}
