---
title: CI/CD Server
subtitle: Distributed job dispatch with atkins --login
layout: page
---

Atkins can attach to a CI/CD server. Once a machine is logged in, `atkins` stops running the pipeline itself: it records the run as a job, prints one URL, and exits. An agent checks the repository out somewhere else and runs it there.

The design keeps atkins in charge. The server is a queue and a ledger; it does not decide what a pipeline does. Three things travel with every run — the git repository, the directory inside its work tree, and the atkins command — which is everything another machine needs to reproduce it.

```text
laptop                     server (:3200)            agent (container)
------                     --------------            -----------------
atkins --login  ─────────► /api/user/login
atkins          ─────────► /api/dispatch  ──┐
   prints job URL, exits                    │  job queued
                                            ▼
                           /api/job/claim ◄──── poll
                                            ──► clone into
                                                /app/data/repos/<slug>
                           /api/job/{id}/log ◄─ streamed output
                           /api/job/{id}/status ◄ terminal state
browser ──► /job/{ULID}
```

## Trying it

The repository ships a throwaway instance: a server and one agent, on port 3200.

```bash
atkins up                                 # build and start
atkins --register http://localhost:3200   # first account, becomes admin
atkins                                    # prints http://localhost:3200/job/<ULID>
atkins down
```

Nothing is mounted from the host. The agent clones the repository your checkout reports, exactly as another machine would, so a job runs against what the remote actually has rather than against your working tree. State lives in the containers and goes away with them.

Settings for the instance live in `.env.docker`, shared by both services. Two of them are worth replacing before it outlives an afternoon: `ATKINS_SIGNING_KEY`, because whoever knows it can mint any token, and `ATKINS_AGENT_TOKEN`, because whoever knows it can join the queue and read the deploy keys.

## Logging in

```bash
atkins --login https://ci.example.com
```

Atkins prompts for an email and password, exchanges them for an access token and a refresh token, and writes them to `~/.atkins/credentials.json` (mode `0600`). The file holds one credential per server, so a machine can attach to several instances.

To create an account:

```bash
atkins --register https://ci.example.com
```

Registration prompts for a username, an email and a password, then logs the new account in. The **first** human account on a fresh instance is always allowed and becomes an admin, so a new server can be bootstrapped without touching its database. Later registrations need `registration.open`. Agents do not count as humans, so an agent starting first neither takes that slot nor closes the door behind it.

To detach a machine:

```bash
atkins --logout
```

Logout revokes the session on the server and removes the local credential. It succeeds locally even when the server is unreachable.

### Non-interactive login

Set the credentials in the environment and no prompt is shown. This is how a container or a provisioning script attaches itself:

| Variable          | Purpose                     |
|-------------------|-----------------------------|
| `ATKINS_EMAIL`    | Email, skips the prompt     |
| `ATKINS_USERNAME` | Username, `--register` only |
| `ATKINS_PASSWORD` | Password, skips the prompt  |

## What a run dispatches

Once logged in, `atkins` posts to `/api/dispatch` and prints the job URL:

| Field               | Value                                               |
|---------------------|-----------------------------------------------------|
| `repository`        | `origin` remote URL, ref and default branch         |
| `working_directory` | Pipeline directory, relative to the repository root |
| `command`           | The atkins invocation, e.g. `atkins test:build`     |
| `parent_id`         | `ATKINS_JOB_ID`, when a job dispatches further work |
| `labels`            | Which agents may run it                             |
| `clone_depth`       | History the agent's work tree needs; 0 is all of it |

The server normalizes the remote URL into a `host/owner/name` slug, so `git@github.com:you/repo.git` and `https://github.com/you/repo` are one repository. Repositories are created on first sight; nothing has to be registered up front.

The ref atkins sends is the **commit you are on**, not the branch you are on. A dispatched run belongs to the code in front of you, and a branch name would let the agent build whatever that branch had moved to by the time it claimed the job. The consequence is worth knowing: dispatching a commit you have not pushed fails, naming the ref it could not find, rather than quietly building the branch tip instead.

Delegation degrades to a local run rather than to an error. If the machine isn't logged in, isn't inside a git repository, or the server can't be reached, the pipeline runs here as it always did — with the reason on stderr.

To run locally on purpose:

```bash
atkins --local              # this run only
ATKINS_NO_DISPATCH=1 atkins # same, for a script
```

Setting `client.dispatch: false` in the configuration turns delegation off for good.

### Environment exported to a job

An agent publishes these to the command it runs:

| Variable               | Value                                       |
|------------------------|---------------------------------------------|
| `ATKINS_JOB_ID`        | This job                                    |
| `ATKINS_PARENT_JOB_ID` | The job that dispatched it, if any          |
| `ATKINS_ROOT_JOB_ID`   | Top of the job tree, stable across nesting  |
| `ATKINS_JOB_PARAMS`    | The JSON object the job was dispatched with |
| `ATKINS_WORKSPACE`     | The checkout the job runs in                |
| `ATKINS_REPOSITORY`    | Repository slug                             |
| `ATKINS_REF`           | The ref that was checked out                |
| `ATKINS_COMMIT_SHA`    | The commit that ref resolved to             |
| `ATKINS_REVISION`      | The same commit, under its older name       |
| `ATKINS_BRANCH`        | Set only when the ref named a branch        |
| `CI`                   | `true`                                      |

It also sets `ATKINS_NO_DISPATCH=1`. Without it, the atkins the agent runs would see the agent's own credentials, hand the work straight back to the server, and nothing would ever execute. A pipeline that genuinely wants to queue child work clears it:

```yaml
jobs:
  analyze:
    desc: "Fan out one job per tag"
    vars:
      tags: $(git tag --list)
    steps:
      - for: tag in tags
        env:
          vars:
            ATKINS_NO_DISPATCH: ""
        run: atkins analyzeTag
```

Children read `ATKINS_JOB_ID` and are recorded under it. Nesting is bounded by `job.max_depth` (3 by default), so a pipeline that dispatches itself cannot run away with the queue.

## Triggering without a checkout

A cron, a webhook receiver or another job can queue work for a repository the server already knows, without a checkout of its own. This is the "POST a job name to a project" trigger:

```bash
REPO=$(curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository" |
  jq -r '.[] | select(.slug == "github.com/titpetric/atkins") | .id')

curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
  -d '{"job": "analyze"}'
```

`job` becomes `atkins analyze`, so a trigger payload stays a name rather than a shell string somebody has to quote. `command` overrides the whole invocation when a bare job name isn't enough. With no `ref`, the agent resolves the repository's default branch when it runs the job — what a nightly wants.

`params` is how a dispatching job hands each child its work. It reaches the job as `ATKINS_JOB_PARAMS`:

```bash
for tag in $(git tag --list); do
  curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
    -d "$(jq -n --arg tag "$tag" --arg parent "$ATKINS_JOB_ID" \
      '{job: "analyzeTag", parent_id: $parent, ref: $tag, clone_depth: 1, params: {tag: $tag}}')"
done
```

`ref` is what the child checks out; `params` is what it is told. Here they carry the same tag, and the child gets one commit of history rather than the whole repository.

Each child records `parent_id`, shares the root job's ID, and counts against `job.max_depth`.

## Refs, tags and clone depth

A job says what to check out in one field. `ref` takes whatever git takes:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
  -d '{"job": "release", "ref": "v1.2.3", "clone_depth": 1}'
```

| `ref`              | Resolves to                                  |
|--------------------|----------------------------------------------|
| *(empty)*          | The repository's default branch, at run time |
| `v1.2.3`           | That tag, then a branch of the same name     |
| `main`             | That branch                                  |
| `4f2a1c…`          | That commit, abbreviated or in full          |
| `refs/tags/v1.2.3` | Exactly that ref, and nothing else           |
| `refs/pull/7/head` | Anything else the remote carries             |

A bare name is looked for as a tag first and then as a branch, which is git's own order. A fully qualified `refs/...` name is taken at its word.

**A ref that does not resolve fails the job, naming the ref.** There is no fallback to the default branch: a typo in a tag name should not produce a green run of the wrong code.

**The commit is recorded.** Before it runs anything the agent reports what it actually checked out, and the job page shows the ref and the commit side by side. A tag moves; a job that remembered only `v1.2.3` could not be reproduced after it did. `ATKINS_COMMIT_SHA` carries the same value into the pipeline, so an artefact can be labelled with the commit even when the job named a branch.

A retry repeats the **ref**, not the commit: a retried job pointed at a branch asks for whatever that branch holds now. Put a commit in the ref to get the same code twice.

### Clone depth

`clone_depth` limits the history the job's work tree carries. `1` is one commit, `0` — the default — is all of it.

Depth applies to the **work tree**, never to the agent's repository cache. The cache is a full mirror shared by every job for that repository, and it stays that way: a shallow cache would have to be deepened the first time a job named an older tag, and deepening repeatedly costs more than never having discarded the objects. The work tree is thrown away when the job ends, so limiting it costs nothing later.

What a shallow work tree buys is a small `.git`: it matters when the job packages the tree, copies it into a docker build context, or uploads it. It does not make the clone faster — a full work tree is hardlinked out of the cache and is very nearly free, while a shallow one has to be fetched. Ask for depth when the size of `.git` matters to the job, not to save the agent time.

A job with `clone_depth: 1` has no history and no tags: `git describe`, `git log` past the tip and `git merge-base` will not work, even for the tag the job itself named. `ATKINS_REF` carries that name into the job, which is what a build usually wanted `git describe` for. Fan-outs over tags are the case that wants a depth — each child builds one commit and never looks behind it.

## Retrying and cancelling

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/retry"
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/cancel"
```

A retry is a **new job** built from the finished one, not a reset: the previous attempt keeps its output and its outcome, which is the point of looking at a failure after retrying it. A retried child stays under the job that dispatched it. Only finished jobs can be retried — cancel a running one first.

## The job page

`/job/{ULID}` is the URL atkins prints. It shows the status, the command, the repository, the ref the job asked for and the commit the agent resolved it to, timing, the exit code, and the output the agent captured, and it refreshes itself until the job settles. `/` lists recent jobs.

The pages are readable without a session: a ULID is not enumerable in practice, and the point of printing a URL in a terminal is that pasting it into a browser works. Put the server behind your own auth if the output is sensitive.

## Configuration

Two layers. `.atkins/config.yml` is the source of truth, and `ATKINS_*` variables overlay individual fields on top of it — an unset or empty variable leaves the configured value alone. Every field is checked on load and an empty one takes its default from the document built into the binary.

```bash
atkins --config
```

opens a menu over the project document, creating it from the built-in defaults if it doesn't exist. Fields currently overridden by the environment are marked, so an edit that won't take effect says so.

```yaml
client:
  server: https://ci.example.com
  labels: [linux, amd64]
  dispatch: true

server:
  addr: ":3200"
  database: sqlite://file:atkins.db
  signing_key: ""
  agent_token: ""

agent:
  id: ""
  token: ""
  data_dir: /app/data
  prefer_https: false
```

## Running the server

```bash
atkins server --signing-key "$(openssl rand -hex 32)" \
              --agent-token "$(openssl rand -hex 32)"
```

| Flag                   | Config                      | Environment                 |
|------------------------|-----------------------------|-----------------------------|
| `--addr`               | `server.addr`               | `PLATFORM_SERVER_ADDR`      |
| `--database`           | `server.database`           | `PLATFORM_DB_DEFAULT`       |
| `--signing-key`        | `server.signing_key`        | `ATKINS_SIGNING_KEY`        |
| `--agent-token`        | `server.agent_token`        | `ATKINS_AGENT_TOKEN`        |
| `--allow-registration` | `server.allow_registration` | `ATKINS_ALLOW_REGISTRATION` |

The signing key is required: a server signing tokens with a known key is one anyone can mint an admin token for. Rotating it invalidates every issued token, which is the intended way to log everyone out.

Migrations run on start. The database can be sqlite, MySQL or PostgreSQL:

```bash
atkins server --database "mysql://user:pass@tcp(db:3306)/atkins"
```

## Running an agent

```bash
atkins worker --server https://ci.example.com --token "$ATKINS_AGENT_TOKEN"
```

Agents don't have passwords. One shared enrolment token, given to the agent, is traded for the same rotating credentials a human holds — so a fleet carries one secret rather than a password per worker. Enrolling twice with the same `--agent-id` returns the same account, so a restarted agent keeps its identity and its job history.

For each claimed job the agent:

1. mirrors the repository into `<data_dir>/repos/<slug>.git`, or fetches it when already cached;
2. resolves the job's ref against that mirror, once, into a commit — a tag that moves mid-job cannot split the run across two commits;
3. builds a work tree in `<data_dir>/work/<job-id>` at that commit, shallow if the job asked for a depth, and reports the ref and commit back to the server;
4. runs the command in the job's working directory;
5. streams the output back as it goes;
6. reports `passed`, `failed` or `timeout`, and removes the work tree.

The work tree is checked out detached. A job builds one commit; leaving a branch checked out would suggest it has somewhere to push it back to.

The lease is renewed every 30 seconds. A job whose lease lapses is swept back out of `running` and marked `timeout`, so a worker that disappears mid-job doesn't strand its work.

Labels filter the queue. A job with no labels runs anywhere; a job requiring `linux,arm64` only lands on an agent advertising both:

```bash
atkins worker --labels linux,arm64,docker
```

## Repository allowlist

By default any repository a logged-in user dispatches will be built. To restrict that, switch the policy and write rules:

```bash
# Only repositories matching a rule may be built.
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"allowlist"}' "$SERVER/api/admin/setting/repository.policy"

curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"pattern":"github.com/titpetric/*"}' "$SERVER/api/admin/repository"
```

Patterns match the repository slug. `*` matches within a path segment, `**` across segments:

| Pattern                       | Matches                       |
|-------------------------------|-------------------------------|
| `github.com/titpetric/atkins` | that one repository           |
| `github.com/titpetric/*`      | every repository of one owner |
| `github.com/**`               | every repository on one host  |
| `**`                          | everything                    |

With the policy on and no rules, nothing runs. That is the point of turning it on.

The rule is enforced twice. The server refuses the dispatch with a 403, and the agent checks again before cloning — because a job can outlive the rule that admitted it, and a repository removed from the allowlist must not be built by a job that was queued while it was still listed. An agent that cannot read the policy refuses to run: a server outage must not turn an allowlist into an open door.

## Deploy keys

An agent clones public repositories anonymously. For private ones, give the server a key:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/admin/ssh-key" \
  -d "$(jq -n --arg key "$(cat ~/.ssh/id_ed25519)" \
    '{name: "github", host: "github.com", private_key: $key}')"
```

The public half and a SHA256 fingerprint are derived on create, so a key that doesn't parse is refused there and then rather than failing a job later. Private material is never returned to an admin listing — only to an enrolled agent, which writes the keys to `<data_dir>/ssh` at `0600` and offers them to git with `IdentitiesOnly=yes`. The directory is rebuilt on each refresh, so a key deleted on the server stops working within the minute. Pin host keys with `known_hosts` on the key; without it the agent trusts on first use rather than prompting, which would hang a job.

A container with no key agent can side-step ssh entirely for public repositories:

```bash
atkins worker --token "$TOKEN"   # with agent.prefer_https: true
```

which rewrites `git@host:owner/repo.git` to `https://host/owner/repo.git` before cloning.

## Settings

Runtime configuration an admin can change without a restart:

| Setting             | Default | Purpose                                       |
|---------------------|---------|-----------------------------------------------|
| `repository.policy` | `open`  | `open`, or `allowlist` to gate on rules       |
| `registration.open` | `false` | Let anyone register                           |
| `job.max_depth`     | `3`     | How deep a job may dispatch children          |
| `job.lease_ttl`     | `15m`   | How long an agent may hold a job              |
| `job.retention`     | `0`     | How long finished jobs are kept; 0 is forever |

```bash
curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/admin/setting" |
  jq -r '.[] | "\(.name)\t\(.value)\tdefault=\(.is_default)"'
```

## Users and roles

Three flags, because there are three things to decide:

- **`is_admin`** — the `/api/admin/*` surface. The API refuses to remove the last active admin, so an instance can't be locked out.
- **`is_agent`** — may claim jobs, report status, append output, and read deploy keys. Reachable only by enrolment, never by registration.
- **`is_active`** — login and every authenticated call. Deactivating takes effect immediately, not when the token expires.

```bash
curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/admin/user" |
  jq -r '.[] | "\(.username)\tadmin=\(.is_admin)\tagent=\(.is_agent)\tactive=\(.is_active)"'

curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"is_admin":true}' "$SERVER/api/admin/user/$USER_ID"
```

## API

| Endpoint                       | Method          | Who    | Purpose                                |
|--------------------------------|-----------------|--------|----------------------------------------|
| `/api/user/register`           | POST            | anyone | Create an account, returns tokens      |
| `/api/user/login`              | POST            | anyone | Email and password for tokens          |
| `/api/user/refreshToken`       | POST            | anyone | Rotate the refresh token               |
| `/api/user/logout`             | POST            | user   | Revoke the session                     |
| `/api/user/whoami`             | GET             | user   | Describe the authenticated user        |
| `/api/dispatch`                | POST            | user   | Record a job, returns its ID           |
| `/api/repository`              | GET             | user   | List known repositories                |
| `/api/repository/{id}/trigger` | POST            | user   | Queue a job by name, with params       |
| `/api/job`                     | GET             | user   | List jobs                              |
| `/api/job/{id}`                | GET             | user   | Read one job                           |
| `/api/job/{id}/log`            | GET             | user   | Read captured output                   |
| `/api/job/{id}/retry`          | POST            | user   | Queue a copy of a finished job         |
| `/api/job/{id}/cancel`         | POST            | user   | Settle an unfinished job               |
| `/api/job/claim`               | POST            | agent  | Lease the oldest pending job           |
| `/api/job/{id}/status`         | POST            | agent  | Settle a job                           |
| `/api/job/{id}/checkout`       | POST            | agent  | Record the ref and commit it built     |
| `/api/job/{id}/heartbeat`      | POST            | agent  | Extend the lease                       |
| `/api/job/{id}/log`            | POST            | agent  | Append output                          |
| `/api/agent/enrol`             | POST            | token  | Trade the shared token for credentials |
| `/api/agent/policy`            | GET             | agent  | The repository policy to enforce       |
| `/api/agent/ssh-key`           | GET             | agent  | Deploy keys, with private material     |
| `/api/admin/user[/{id}]`       | GET/POST        | admin  | List accounts, change flags            |
| `/api/admin/repository[/{id}]` | GET/POST/DELETE | admin  | Manage allowlist rules                 |
| `/api/admin/setting[/{name}]`  | GET/POST/DELETE | admin  | Read and change settings               |
| `/api/admin/ssh-key[/{id}]`    | GET/POST/DELETE | admin  | Manage deploy keys                     |

Authentication is `Authorization: Bearer <token>`. Access tokens live an hour and carry the session they came from, so logout takes effect immediately rather than when the token expires. Refresh tokens are single-use and rotate on every refresh, which makes a leaked one detectable: the legitimate client's next refresh fails.

## See Also

- [CLI Flags](./cli-flags) - Command-line reference
- [Automation (JSON/YAML)](./automation) - Machine-readable run output
- [Migrating from GitHub Actions](../migrating/migration-from-github-actions) - What carries over and what doesn't
