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
	"log"
	"sync"
	"time"

	"github.com/titpetric/platform"
	"github.com/titpetric/platform/pkg/telemetry"

	"github.com/titpetric/atkins/server/api"
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

	jobs *storage.JobStorage

	// reclaim is the background sweep of expired agent leases.
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

	// A setting overrides the corresponding start-up flag, so an admin
	// can change these without a restart.
	maxDepth := m.opts.MaxJobDepth
	if configured := settings.Int(model.SettingJobMaxDepth); configured > 0 {
		maxDepth = configured
	}
	leaseTTL := m.opts.LeaseTTL
	if configured := settings.Duration(model.SettingJobLeaseTTL); configured > 0 {
		leaseTTL = configured
	}

	m.jobs = storage.NewJobStorage(db, maxDepth, leaseTTL)
	jobLogs := storage.NewJobLogStorage(db)
	repositories := storage.NewRepositoryStorage(db)

	m.api = api.NewHandlers(api.Options{
		SigningKey:            m.opts.SigningKey,
		TokenTTL:              m.opts.TokenTTL,
		AllowRegistration:     m.opts.AllowRegistration,
		AgentToken:            m.opts.AgentToken,
		UserStorage:           storage.NewUserStorage(db),
		SessionStorage:        storage.NewSessionStorage(db, m.opts.SessionTTL),
		RepositoryStorage:     repositories,
		JobStorage:            m.jobs,
		JobLogStorage:         jobLogs,
		RepositoryRuleStorage: storage.NewRepositoryRuleStorage(db),
		SettingStorage:        settings,
		SSHKeyStorage:         storage.NewSSHKeyStorage(db),
	})

	m.web, err = web.NewHandlers(web.Options{
		JobStorage:        m.jobs,
		JobLogStorage:     jobLogs,
		RepositoryStorage: repositories,
	})
	if err != nil {
		return err
	}

	m.startReclaim(ctx)

	return nil
}

// Mount registers the API and page routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
	m.api.Mount(r)
	m.web.Mount(r)
	return nil
}

// Stop halts the lease sweep and waits for it to finish.
func (m *Module) Stop(context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.done.Wait()
	return nil
}

// startReclaim sweeps jobs whose agent lease has lapsed back out of the
// running state. Without it, an agent that dies mid-job strands that
// job as running forever.
func (m *Module) startReclaim(ctx context.Context) {
	if m.opts.ReclaimInterval <= 0 {
		return
	}

	// The sweep outlives the start context; it is bound to the module
	// lifecycle and cancelled from Stop.
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	m.cancel = cancel

	m.done.Add(1)
	go func() {
		defer m.done.Done()

		ticker := time.NewTicker(m.opts.ReclaimInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reclaimed, err := m.jobs.ReclaimExpired(ctx)
				if err != nil {
					telemetry.CaptureError(ctx, err)
					continue
				}
				if reclaimed > 0 {
					log.Printf("[atkins] reclaimed %d job(s) from expired agent leases", reclaimed)
				}
			}
		}
	}()
}
