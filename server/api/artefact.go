package api

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/storage"
)

// HeaderArtefactChecksum carries the SHA256 the agent computed while
// reading the file. It is optional; when present the server compares it
// against what actually arrived, so a truncated upload is a 400 rather
// than an artefact nobody can use.
const HeaderArtefactChecksum = "X-Atkins-Checksum"

// ArtefactView is how an artefact is described over the API.
//
// It is a projection rather than the row: `storage_key` says where the
// bytes sit on the server, and that is the server's business. The rule
// is the same one SSHKeyView follows — the projection is the guard, not
// a `json:"-"` somebody has to remember to add to a new column.
type ArtefactView struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Checksum    string `json:"checksum"`
	AgentID     string `json:"agent_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`

	// URL is where the bytes are, so a caller that listed artefacts
	// does not have to know how to build the path.
	URL string `json:"url"`
}

// artefactView projects a stored artefact.
func artefactView(artefact model.JobArtefact) ArtefactView {
	view := ArtefactView{
		ID:          artefact.ID,
		JobID:       artefact.JobID,
		Path:        artefact.Path,
		Size:        artefact.Size,
		ContentType: artefact.ContentType,
		Checksum:    artefact.Checksum,
		AgentID:     artefact.AgentID,
		URL:         "/api/job/" + artefact.JobID + "/artefact/" + artefact.ID,
	}
	if artefact.CreatedAt != nil {
		view.CreatedAt = artefact.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return view
}

// UploadArtefact stores a file a job produced.
//
// The body is the file itself rather than a multipart form: an artefact
// is bytes, and a raw body streams from the agent's disk to the
// server's without either side building an envelope around it. The
// name, the media type and the checksum are small enough to be
// metadata, and travel as the `path` query parameter, `Content-Type`
// and `X-Atkins-Checksum`.
func (s *Handlers) UploadArtefact(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.uploadArtefact(w, r))
}

func (s *Handlers) uploadArtefact(w http.ResponseWriter, r *http.Request) error {
	// Artefacts are pushed by the agent that ran the job, exactly like
	// its output.
	agent, err := s.requireAgent(r)
	if err != nil {
		return err
	}

	jobID := platform.URLParam(r, "jobID")
	if _, err := s.jobs.Get(r.Context(), jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return requestError(http.StatusNotFound, errors.New("job not found"))
		}
		return err
	}

	maxSize := s.settings.Bytes(model.SettingArtefactMaxSize)

	// The store stops reading at the limit on its own. This caps what
	// the server will take off the socket at all, so the remainder of
	// an oversized upload is not politely drained after the refusal.
	// A limit of zero is "no limit", which an admin can choose.
	var body io.Reader = r.Body
	if maxSize > 0 {
		body = http.MaxBytesReader(w, r.Body, maxSize+1)
	}

	artefact, err := s.artefacts.Create(r.Context(), storage.ArtefactRequest{
		JobID:       jobID,
		AgentID:     agent.Username,
		Path:        platform.QueryParam(r, "path"),
		ContentType: r.Header.Get("Content-Type"),
		Checksum:    r.Header.Get(HeaderArtefactChecksum),
		Content:     body,
		MaxSize:     maxSize,
		MaxCount:    s.settings.Int(model.SettingArtefactMaxCount),
	})
	if err != nil {
		return mapArtefactError(err)
	}

	platform.JSON(w, r, http.StatusCreated, artefactView(*artefact))
	return nil
}

// ListArtefacts returns the artefacts a job produced.
func (s *Handlers) ListArtefacts(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.listArtefacts(w, r))
}

func (s *Handlers) listArtefacts(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.authenticateUser(r); err != nil {
		return err
	}

	artefacts, err := s.artefacts.List(r.Context(), platform.URLParam(r, "jobID"))
	if err != nil {
		return err
	}

	views := make([]ArtefactView, 0, len(artefacts))
	for _, artefact := range artefacts {
		views = append(views, artefactView(artefact))
	}

	platform.JSON(w, r, http.StatusOK, views)
	return nil
}

// DownloadArtefact writes the bytes of one artefact.
//
// This is the authenticated way to read an artefact, and the one a
// script should use. The job page has an unauthenticated route of its
// own, on the same terms as the output it shows.
func (s *Handlers) DownloadArtefact(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, s.downloadArtefact(w, r))
}

func (s *Handlers) downloadArtefact(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.authenticateUser(r); err != nil {
		return err
	}

	artefact, err := s.artefacts.Get(r.Context(),
		platform.URLParam(r, "jobID"), platform.URLParam(r, "artefactID"))
	if err != nil {
		return mapArtefactError(err)
	}

	contents, err := s.artefacts.Open(r.Context(), artefact)
	if err != nil {
		return mapArtefactError(err)
	}
	defer contents.Close()

	WriteArtefact(w, artefact, contents)
	return nil
}

// WriteArtefact streams an artefact as a download.
//
// The web package serves the same bytes from the job page, and a
// difference between the two responses would only ever be a bug.
func WriteArtefact(w http.ResponseWriter, artefact *model.JobArtefact, contents io.Reader) {
	w.Header().Set("Content-Type", artefact.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(artefact.Size, 10))
	// A job's output is not to be trusted as a page: an artefact named
	// report.html is a file to save, never something to run in the
	// server's origin.
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(downloadName(artefact.Path)))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	_, _ = io.Copy(w, contents)
}

// downloadName reduces a stored path to the file name a browser should
// save it as.
func downloadName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// mapArtefactError turns an artefact failure into a status.
//
// Oversize is 413 because the request was too big to accept; too many
// is 422 because the request was fine and the job was not, which is how
// the runaway-nesting limit already reports itself.
func mapArtefactError(err error) error {
	switch {
	case errors.Is(err, model.ErrInvalidArtefactPath):
		return requestError(http.StatusBadRequest, err)
	case errors.Is(err, model.ErrChecksumMismatch):
		return requestError(http.StatusBadRequest, err)
	case errors.Is(err, model.ErrArtefactTooLarge):
		return requestError(http.StatusRequestEntityTooLarge, err)
	case errors.Is(err, model.ErrTooManyArtefacts):
		return requestError(http.StatusUnprocessableEntity, err)
	case errors.Is(err, model.ErrArtefactNotFound):
		return requestError(http.StatusNotFound, err)
	}
	return err
}
