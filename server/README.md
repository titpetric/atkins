# server

The atkins CI/CD server: a `platform.Module` that authenticates atkins
clients and records the jobs they dispatch.

See [docs/content/usage/ci-cd.md](../docs/content/usage/ci-cd.md) for the
user-facing guide. This file covers the layout and the decisions behind
it.

## Layout

```text
server/
├── server.go        module lifecycle, route assembly, background sweeps
├── options.go       options, built from the config package
├── api/             JSON handlers: decode, validate, map status, encode
│   ├── user.go      register, login, refresh, logout, whoami
│   ├── dispatch.go  dispatch and the job queue
│   ├── agent.go     enrolment, policy, deploy keys
│   └── admin.go     users, allowlist, settings, keys
│   └── artefact.go  upload, list and download job outputs
├── auth/            HS256 access tokens
├── blob/            artefact bytes, behind a Put/Open/Remove interface
├── model/           generated records (types.mig.go) + domain helpers
├── schema/          append-only migrations, one file per table
├── storage/         SQL, via titpetric/pdo generic methods
└── web/             /job/{ULID} and / pages, embedded templates
```

`model` represents persisted data, not request payloads. Request and
response types live beside the handlers that speak them.

`storage` owns SQL, transactions and invariants. `api` calls storage and
never issues SQL itself.

## Schema

`schema/*.up.sql` is the source of truth. One file owns one table, and
files are append-only once applied anywhere: a later change belongs in a
new `*.up.sql`.

Regenerate `model/types.mig.go` and `schema/docs` after any change:

```sh
atkins "$HOME/.atkins/skills/schema.yml" -w ./server migrate
```

That applies the migrations to a disposable database, runs `mig lint`,
writes `schema/docs` and `schema/schema.yml`, then runs `mig gen`.
Generated files say `DO NOT EDIT`: change SQL and regenerate.

| Table             | Holds                                                |
|-------------------|------------------------------------------------------|
| `user`            | Accounts, bcrypt password, admin/active/agent flags  |
| `session`         | One CLI login, with its refresh token                |
| `repository`      | A git repository, identified by its normalized slug  |
| `job`             | One dispatched run, its lease and its outcome        |
| `job_log`         | Output chunks, sequenced per job                     |
| `job_artefact`    | Files a job produced, and where their bytes are      |
| `repository_rule` | Allowlist patterns                                   |
| `setting`         | Runtime configuration an admin can change            |
| `ssh_key`         | Deploy keys for clone and fetch                      |

Tables are singular, columns are `snake_case`, IDs are ULIDs in
`CHAR(26)` generated in Go. There are no SQL foreign keys: modules share
identity values without coupling migrations or connections.

## Database access

Queries go through [`titpetric/pdo`](https://github.com/titpetric/pdo).
Its generic methods scan straight into the generated model types:

```go
query := `SELECT * FROM ` + model.UserTable + ` WHERE id = ? AND deleted_at IS NULL`
return client(s.db).Get[model.User](ctx, query, id)
```

A `*pdo.PDO` is request-scoped and not safe for concurrent use, so every
storage method allocates one over the shared `*sqlx.DB` pool. Storage
functions are transaction-agnostic: the same call joins an open
transaction or runs standalone, so `Claim` composes without a second
code path.

The pool comes from the platform's connection provider, named `atkins`
(`PLATFORM_DB_ATKINS`) and falling back to `PLATFORM_DB_DEFAULT`.
`Options.Connection` overrides the name, which is how the module tests
give each case its own database: the platform caches pools by name for
the life of the process.

## Artefacts

A job's output splits in two. `job_artefact` holds what the file is —
job, path, size, media type, SHA256 — and `blob` holds the bytes,
addressed by the `storage_key` on the row.

The split is what keeps an object store backend to one interface: `Put`,
`Open`, `Remove` over a key namespace, with no SQL and no handler near
it. `blob.Dir` writes under a root directory, which is the right first
backend and probably the right one for most installs — the database is
already on a disk, artefacts are written once and read rarely, and a
directory is something an operator can back up without asking anybody
for a bucket.

`JobArtefactStorage` owns both halves so they cannot drift: bytes are
written before the row exists, and removed if the insert fails; a
retention sweep removes bytes first and then soft deletes the row, so
the failure mode is a row that reads as "swept" rather than a file
nothing will ever clean up.

The count and size limits are settings rather than constants, because
the moment they matter is the moment a disk is filling, and that is the
worst moment to need a redeploy.

## Lifecycle

`Start` connects, migrates, wires storage and handlers, prepares the
artefact root, and starts two background sweeps. `Mount` registers
routes. `Stop` cancels both and waits for them.

A root the server cannot create is fatal at start-up rather than a
surprise on the first job that produces a file.

The lease sweep is the reason an agent can die without stranding a job:
a `running` job whose `lease_expires_at` has passed is moved to
`timeout` and can be retried. It carries the artefact retention pass
too, which reads its setting on every tick rather than at start-up.

The retention sweep applies `job.retention` and `job.log_retention`. It
is a second ticker rather than another statement in the first, because
the two have nothing in common but a timer: reclaiming is one cheap
UPDATE that has to happen within a lease of an agent dying, and
retention walks two tables about as often as a log file is worth
rotating. Its cadence is `Options.RetentionInterval`, a start-up flag,
while the windows are settings — the cadence is a property of the
machine, the windows are policy.

`JobStorage.Purge` deletes in bounded batches and stops after
`DefaultRetentionBatches` of them, reporting `Partial` so the next tick
knows there is more. A first sweep of an instance that has been running
for a year is therefore a series of small deletes rather than one that
holds `job_log` for minutes. The batch for expired output is selected
from `job_log` joined to `job`, not from `job`: selecting jobs and
deleting their logs would keep re-reading jobs whose output was already
gone, and a large backlog would never be worked through.

Neither window touches a job that has not settled. A pending job is not
old, it is waiting.

## Authentication and roles

Access tokens are short-lived HS256 JWTs carrying `user_id`,
`session_id` and a `jti`. Refresh tokens are 32 random bytes stored on
the session row and rotated on every use.

Binding the access token to a session is what makes logout immediate:
the middleware rejects a token whose session is revoked, rather than
waiting for the token to expire. It also avoids keeping a revocation
list of every access token ever issued.

Three flags on `user`, rather than a role table, because there are three
things to decide:

- `is_admin` gates `/api/admin/*`. The API refuses to remove the last
  active admin, so an instance cannot be locked out of itself.
- `is_agent` gates the queue and the deploy keys. It is reachable only
  through `/api/agent/enrol` with the shared token, never through
  registration, so a self-registered account cannot promote itself into
  reading private keys.
- `is_active` gates every authenticated call.

Agents are excluded from the human user count. An agent enrolling before
any human therefore neither claims the bootstrap admin slot nor closes
registration behind it.

## Policy

`repository.policy` is `open` or `allowlist`. Under `allowlist`, a
dispatch is refused unless an active `repository_rule` pattern matches
the repository slug.

The rule is enforced in two places on purpose. The server refuses the
dispatch, and the agent checks again before cloning: a job can outlive
the rule that admitted it, and a repository removed from the allowlist
must not be built by a job queued while it was still listed. An agent
that cannot read the policy refuses to run, so a server outage does not
turn an allowlist into an open door.

Settings are typed and validated against a registry in
`model/setting.go`, cached in memory by `storage.SettingStorage` because
they are read on the dispatch path, and fall back to a registry default
so the table is empty on a fresh instance.

## Visibility

`job.visibility` is `private` or `public`, and it governs two surfaces.

Over the API, `private` narrows job reads to the caller: their own jobs,
and everything under a job tree they started. The tree half is not
decoration — a pipeline that clears `ATKINS_NO_DISPATCH` queues its
children under the agent's credentials, so scoping on `user_id` alone
would hide a fan-out from the person who started it. Admins and agents
are exempt: an admin is what the flag is for, and an agent works the
whole queue. A job the caller may not read is a 404, because "not
yours" and "not here" must look the same.

The pages have no session to check, so `private` checks a per-job token
in the URL instead: `auth.JWT.ViewToken` is an HMAC of the job ID under
the signing key. Deriving it rather than storing it means no column to
migrate, nothing extra in a database dump, and one way to revoke every
outstanding link — rotate the signing key, which already revokes every
access token. The token rides in the URL because the requirement it has
to keep meeting is that atkins prints one line a human pastes into a
browser.

Under `private` the index page lists nothing: no token can scope a
listing and there is no session to scope it by, so `GET /api/job` is
the listing. The page still answers — it is the front door, and the
compose health check probes it.

## Deploy keys

`ssh_key` rows carry the private half, and it leaves this package only
through `GET /api/agent/ssh-key`. Admin listings are projected to
`api.SSHKeyView`, which has no private field at all — the projection is
the guard, not a `json:"-"` tag somebody can forget to add.

Keys are parsed on create so a bad paste is a 400 rather than a failed
job an hour later, and the public half and SHA256 fingerprint are
derived rather than asked for.
