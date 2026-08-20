// Package server is the atkins CI/CD server: a platform.Module that
// authenticates atkins clients and records the jobs they dispatch.
//
// The shape of the system is deliberately small. `atkins --login
// https://domain` stores a credential. From then on every atkins run
// posts to /api/dispatch with three things: which git repository, which
// directory inside its work tree, and the atkins command. The server
// writes a job row and hands back an ID; atkins manages its own job
// dispatch from there, and agents claim queued work over /api/job/claim.
//
// Mount it in a platform server:
//
//	svc := platform.New(platform.NewOptions())
//	svc.Register(server.NewModule(server.NewOptions()))
//	svc.Start(ctx)
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/titpetric/oida"
	"github.com/titpetric/platform"

	"github.com/titpetric/atkins/server/api"
	"github.com/titpetric/atkins/server/blob"
	"github.com/titpetric/atkins/server/model"
	"github.com/titpetric/atkins/server/schema"
	"github.com/titpetric/atkins/server/storage"
	"github.com/titpetric/atkins/server/web"
)

// Name is the module name, and the name of the database connection the
// module uses (PLATFORM_DB_ATKINS).
const Name = storage.ConnectionName

// Module implements the platform module contract for the atkins server.
type Module struct {
	platform.UnimplementedModule

	opts *Options
	api  *api.Handlers
	web  *web.Handlers

	jobs      *storage.JobStorage
	artefacts *storage.JobArtefactStorage
	settings  *storage.SettingStorage

	// tracer records the background sweeps. It is the platform's
	// recorder when the host enabled telemetry, and nil otherwise;
	// every oida entry point tolerates a nil one.
	tracer *oida.Tracer

	// cancel stops the background sweeps: expired agent leases, and
	// retention. done waits for both.
	cancel context.CancelFunc
	done   sync.WaitGroup
}

// Verify contract.
var _ platform.Module = (*Module)(nil)

// NewModule returns the atkins server module. A nil opts selects
// NewOptions, which reads the environment.
func NewModule(opts *Options) *Module {
	if opts == nil {
		opts = NewOptions()
	}
	return &Module{opts: opts}
}

// Name returns the module name.
func (m *Module) Name() string {
	return Name
}

// Start connects the database, applies migrations and wires handlers.
//
// A missing signing key is fatal rather than defaulted: a server that
// signs tokens with a well-known key is a server anyone can mint an
// admin token for.
func (m *Module) Start(ctx context.Context) error {
	if m.opts.SigningKey == "" {
		return errors.New("atkins server: " + EnvSigningKey + " is required to sign access tokens")
	}

	m.tracer = telemetryTracer(ctx)

	db, err := storage.DB(ctx, m.opts.Connection)
	if err != nil {
		return err
	}

	if err := storage.Migrate(ctx, db, schema.Migrations()); err != nil {
		return err
	}

	settings := storage.NewSettingStorage(db)
	if err := settings.Load(ctx); err != nil {
		return err
	}
	m.settings = settings

	// Artefact bytes live on a disk rather than in the database. A root
	// the server cannot create is fatal here, instead of a surprise on
	// the first job that produces a file.
	artefactDir := m.opts.ArtefactDir
	if artefactDir == "" {
		artefactDir = DefaultArtefactDir
	}
	blobs := blob.NewDir(artefactDir)
	if err := blobs.Prepare(); err != nil {
		return fmt.Errorf("atkins server: artefact directory %q: %w", artefactDir, err)
	}

	// The settings store goes in rather than two values read out of it:
	// `job.max_depth` and `job.lease_ttl` are runtime configuration, and
	// resolving them here would mean an admin's change waited for a
	// restart while the API reported it as already in force.
	m.jobs = storage.NewJobStorage(db, settings, m.opts.MaxJobDepth, m.opts.LeaseTTL)
	m.artefacts = storage.NewJobArtefactStorage(db, blobs)
	jobLogs := storage.NewJobLogStorage(db)
	repositories := storage.NewRepositoryStorage(db)
	users := storage.NewUserStorage(db)
	sessions := storage.NewSessionStorage(db, m.opts.SessionTTL)
	rules := storage.NewRepositoryRuleStorage(db)
	sshKeys := storage.NewSSHKeyStorage(db)

	m.api = api.NewHandlers(api.Options{
		SigningKey:            m.opts.SigningKey,
		TokenTTL:              m.opts.TokenTTL,
		AllowRegistration:     m.opts.AllowRegistration,
		AgentToken:            m.opts.AgentToken,
		UserStorage:           users,
		SessionStorage:        sessions,
		RepositoryStorage:     repositories,
		JobStorage:            m.jobs,
		JobLogStorage:         jobLogs,
		JobArtefactStorage:    m.artefacts,
		RepositoryRuleStorage: rules,
		SettingStorage:        settings,
		SSHKeyStorage:         sshKeys,
	})

	// The pages read the same storage the API does. They share the
	// signing key too: the session cookie is signed with it, so there is
	// one secret to configure and rotating it signs the browser out
	// along with every issued token.
	m.web, err = web.NewHandlers(web.Options{
		JobStorage:            m.jobs,
		JobLogStorage:         jobLogs,
		JobArtefactStorage:    m.artefacts,
		RepositoryStorage:     repositories,
		UserStorage:           users,
		SessionStorage:        sessions,
		RepositoryRuleStorage: rules,
		SettingStorage:        settings,
		SSHKeyStorage:         sshKeys,
		SigningKey:            m.opts.SigningKey,
	})
	if err != nil {
		return err
	}

	// The sweeps outlive the start context; they are bound to the
	// module lifecycle and cancelled from Stop.
	sweeps, cancel := context.WithCancel(context.WithoutCancel(ctx))
	m.cancel = cancel

	m.startReclaim(sweeps)
	m.startRetention(sweeps)

	return nil
}

// Mount registers the API and page routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
	m.api.Mount(r)
	m.web.Mount(r)
	return nil
}

// Stop halts the background sweeps and waits for them to finish.
func (m *Module) Stop(context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.done.Wait()
	return nil
}

// startReclaim runs the two periodic sweeps: expired agent leases, and
// artefacts past their retention.
//
// They share a ticker because they are the same kind of work — a small
// amount of tidying on a timer — and because a server with one
// background goroutine is easier to reason about than one with two.
func (m *Module) startReclaim(ctx context.Context) {
	m.sweep(ctx, "atkins.reclaim", m.opts.ReclaimInterval, func(ctx context.Context) error {
		// Deferred rather than sequential: a reclaim that fails is no
		// reason to leave expired bytes on the disk for another tick.
		defer m.pruneArtefacts(ctx)

		reclaimed, err := m.jobs.ReclaimExpired(ctx)
		if err != nil {
			return err
		}
		if reclaimed > 0 {
			log.Printf("[atkins] reclaimed %d job(s) from expired agent leases", reclaimed)
		}
		return nil
	})
}

// startRetention applies job.retention and job.log_retention on a
// ticker of its own.
//
// It is a separate sweep from the lease reclaim rather than another
// statement inside it, because the two have nothing in common but a
// timer: reclaiming is one cheap UPDATE that has to happen within a
// lease of the agent dying, while retention walks two tables and is
// worth doing about as often as a log file is worth rotating.
//
// The windows are read from the settings on every pass, so an admin
// changing them takes effect at the next tick rather than at the next
// restart. Passing zero for both makes the pass a no-op, which is what
// an instance that keeps everything forever wants.
func (m *Module) startRetention(ctx context.Context) {
	m.sweep(ctx, "atkins.retention", m.opts.RetentionInterval, func(ctx context.Context) error {
		result, err := m.jobs.Purge(ctx, storage.RetentionRequest{
			Jobs: m.settings.Duration(model.SettingJobRetention),
			Logs: m.settings.Duration(model.SettingJobLogRetention),
		})

		// A pass that failed halfway still deleted what it deleted, so
		// report the count before the error.
		if !result.Empty() {
			log.Printf("[atkins] retention removed %d job(s) and %d output row(s)%s",
				result.Jobs, result.Logs, partially(result.Partial))
		}

		return err
	})
}

// partially annotates a retention pass that stopped short of the end of
// its backlog, so an operator watching the log can tell a server that
// is catching up from one that is done.
func partially(partial bool) string {
	if partial {
		return ", more to come"
	}
	return ""
}

// sweep runs work named name on a ticker until the module stops. A
// non-positive interval disables it, which is how the module tests keep
// background writes out of their assertions.
//
// Each tick runs inside a trace of its own. A sweep does not arrive over
// the network, so nothing upstream opens one, and without it the storage
// spans underneath would have nothing to record onto. The error a tick
// returns is recorded on that trace and goes nowhere else: a ticker has
// no caller to return it to.
func (m *Module) sweep(ctx context.Context, name string, interval time.Duration, work func(context.Context) error) {
	if interval <= 0 {
		return
	}

	m.done.Add(1)
	go func() {
		defer m.done.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = m.tracer.Observe(ctx, name, work)
			}
		}
	}()
}

// telemetryTracer returns the recorder the platform hosting the module
// registered, or nil when the host records nothing. The module records
// onto the host's tracer rather than the process wide default, so a test
// starting two servers keeps their traces apart.
func telemetryTracer(ctx context.Context) *oida.Tracer {
	host := platform.FromContext(ctx)
	if host == nil {
		return nil
	}

	var module *platform.TelemetryModule
	if !host.Find(&module) {
		return nil
	}
	return module.Tracer()
}

// pruneArtefacts drops artefact bytes that have outlived their
// retention.
//
// Retention is read on every pass rather than captured at start-up,
// because it is a setting an admin changes when a disk fills up, and
// that is exactly the moment a restart is least welcome.
//
// `artefact.retention` falls back to `job.retention`: an artefact
// belongs to a job and should not outlive it. It exists separately
// because bytes are the expensive half — keeping the ledger of what ran
// for a year while dropping the files after a week is a normal thing to
// want, and the reverse never is.
func (m *Module) pruneArtefacts(ctx context.Context) {
	retention := m.settings.Duration(model.SettingArtefactRetention)
	if retention <= 0 {
		retention = m.settings.Duration(model.SettingJobRetention)
	}
	if retention <= 0 {
		return
	}

	swept, err := m.artefacts.PruneExpired(ctx, time.Now().Add(-retention))
	if err != nil {
		oida.RecordError(ctx, err)
		return
	}
	if swept > 0 {
		log.Printf("[atkins] swept %d artefact(s) older than %s", swept, retention)
	}
}
