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
                           /api/job/{id}/artefact ◄ files it produced
                           /api/job/{id}/status ◄ terminal state
browser ──► /job/{ULID}?t=…
```

## Trying it

The repository ships a throwaway instance: a server and one agent, on port 3200.

```bash
atkins up                                 # build and start
atkins --register http://localhost:3200   # first account, becomes admin
atkins                                    # prints http://localhost:3200/job/<ULID>?t=<token>
atkins down
```

Nothing is mounted from the host. The agent clones the repository your checkout reports, exactly as another machine would, so a job runs against what the remote actually has rather than against your working tree. State lives in the containers and goes away with them.

Settings for the instance live in two files: `.env.docker`, which both services load, and `.env.docker.server`, which the server alone loads. The split keeps `ATKINS_SIGNING_KEY` in the server's process, where tokens are issued, while `ATKINS_AGENT_TOKEN` reaches both services, as the agent presents it on enrolment and the server verifies it.

Both values ship as development placeholders. For an instance that outlives the trial, generate replacements with `openssl rand -hex 32`, keep them in the deployment's secret store, and pass them to the services through the environment rather than through a committed file. `ATKINS_SIGNING_KEY` signs every token the server issues, admin tokens among them, and rotating it invalidates every issued token and browser session. `ATKINS_AGENT_TOKEN` admits an agent to the queue and to the deploy keys, so it deserves the same handling and a rotation whenever an agent host is decommissioned. The agent keeps the token to itself; see [what a job receives](#environment-exported-to-a-job).

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
| `artefacts`         | Globs the agent collects when the command exits     |

The server normalizes the remote URL into a `host/owner/name` slug, so `git@github.com:you/repo.git` and `https://github.com/you/repo` are one repository. Repositories are created on first sight; nothing has to be registered up front.

The ref atkins sends is the **commit you are on**, not the branch you are on. A dispatched run belongs to the code in front of you, and a branch name would let the agent build whatever that branch had moved to by the time it claimed the job. The consequence is worth knowing: dispatching a commit you have not pushed fails, naming the ref it could not find, rather than quietly building the branch tip instead.

The response carries the job's ID and, while the instance keeps jobs private, a `view_token` that opens its page in a browser. `atkins` prints the two as one URL; a script driving `/api/dispatch` or a trigger with `curl` builds it the same way. Losing that line is not losing the link: `GET /api/job/{id}` and `GET /api/job` return the same `view_token`, plus a ready-made `url`, to any caller allowed to read the job.

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
| `ATKINS_ARTEFACTS`     | Directory whose contents are kept           |
| `ATKINS_REPOSITORY`    | Repository slug                             |
| `ATKINS_REF`           | The ref that was checked out                |
| `ATKINS_COMMIT_SHA`    | The commit that ref resolved to             |
| `ATKINS_REVISION`      | The same commit, under its older name       |
| `ATKINS_BRANCH`        | Set only when the ref named a branch        |
| `ATKINS_SERVER`        | The server a child job is dispatched to     |
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

### The agent's own settings

The table above lists the whole of the `ATKINS_*` a command sees. The agent sanitizes its own environment before it starts the command: every `ATKINS_*` and `PLATFORM_*` variable in the agent's process is removed, and the job's variables are then set by name.

The filter protects the credentials the agent runs with. `ATKINS_AGENT_TOKEN` admits its holder to the queue, to jobs dispatched for other repositories, and to the deploy keys with their private halves, which keeps a job to the repository it was dispatched for. `ATKINS_SIGNING_KEY` signs every token the server issues, and an installation with the server and the agent on one host has both values in one environment. `PLATFORM_*` is removed alongside them, as a database URL carries its own password.

The rest of the environment reaches the command unchanged — `PATH`, and whatever else the agent was started with — which is how a job finds its tooling. A value a job needs is best given a name outside the `ATKINS_` namespace, and secrets a pipeline needs belong in the job's own configuration rather than in the agent's environment.

## Triggering without a checkout

A cron, a webhook receiver or another job can queue work for a repository the server already knows, without a checkout of its own. This is the "POST a job name to a project" trigger; `/admin/repository` puts a form on it for the times a person is doing the queuing:

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

## Artefacts

A job that produces a file — a scan, a coverage report, a built binary — can push it back to the server and have it attached to the job. This is the "collect the outputs and data mine them centrally" half of a CI system: the agent's disk is thrown away with the work tree, and the server keeps what mattered.

There are two ways to say what to keep, and a job can use both.

**Copy it into `$ATKINS_ARTEFACTS`.** The agent creates that directory before the command runs, and uploads whatever is in it afterwards, named by its path within the directory:

```yaml
jobs:
  scan:
    steps:
      - ./.atkins/bin/scan-repository > scan.json
      - cp scan.json "$ATKINS_ARTEFACTS/"
```

Nothing has to be declared anywhere, no schema changes with your pipeline, and it works for any command in any language: a job says what it wants kept by putting it somewhere.

**Declare a glob when you dispatch or trigger.** For a pipeline you'd rather not edit — one that writes `coverage.json` where it writes it — the job can say what to pick up:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
  -d '{"job": "test", "artefacts": ["coverage.json", "reports/**/*.json"]}'
```

Patterns are relative to the directory the job ran in. `*` matches within a path segment and `**` across segments, the same as the repository allowlist patterns. `.git` is never collected.

Collection runs **after the command exits, whatever it exited with** — including a timeout. The artefacts of a failure are usually the ones worth having.

The agent walks the checkout and matches what it finds, rather than handing the pattern to the filesystem: a pattern can only ever select among files that are already inside the checkout, symlinks are not followed, and a path that tries to leave — `../../etc/passwd` — is dropped when the job is created and again by the agent.

Reading them back:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/artefact" |
  jq -r '.[] | "\(.path)\t\(.size)\t\(.checksum)"'

curl -sSL -H "Authorization: Bearer $TOKEN" -O "$SERVER/api/job/$JOB/artefact/$ARTEFACT"
```

The job page lists them with download links, so the URL atkins printed is also where the files are.

**Where the bytes live.** The database records what an artefact is — job, path, size, media type, SHA256 — and the file itself goes under `server.artefact_dir` as `<job-id>/<artefact-id>`. That is a directory an operator can back up, rsync or mount on a volume. The upload is streamed straight to it and hashed on the way past, so the checksum is what the server received rather than what the agent claimed; an upload that arrives short of its declared checksum is refused.

**Limits.** `artefact.max_size` bounds one file and `artefact.max_count` bounds how many one job may keep, both settings rather than constants, so a full disk is a `curl` away from being stopped rather than a redeploy. Re-uploading a path replaces it instead of adding to it.

**Retention.** `artefact.retention` is how long the bytes are kept; `0` follows `job.retention`. The sweep runs on the same ticker as the lease reclaim, removes the file and marks the row deleted — so the record that a 40MB `scan.json` was produced and swept survives, while the 40MB does not. Artefacts are the thing that fills a disk, and they have their own setting because keeping a year of job history while keeping a week of files is a normal thing to want; the reverse never is.

## Retrying and cancelling

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/retry"
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/cancel"
```

A retry is a **new job** built from the finished one, not a reset: the previous attempt keeps its output and its outcome, which is the point of looking at a failure after retrying it. A retried child stays under the job that dispatched it. Only finished jobs can be retried — cancel a running one first.

Cancelling settles the record and stops the command. The agent finds out at its next heartbeat, which the server refuses once the job is no longer leased to it: the agent kills the command's process group, notes on the page that the job was taken back, and goes to look for the next one. It reports no outcome of its own — a cancelled job stays cancelled — and it does not upload the artefacts collected so far, because a page that says `cancelled` should not go on growing files. Worst case the build runs for one heartbeat interval (`agent.heartbeat_interval`, 30s by default) after the click.

## The job page

`/job/{ULID}?t={token}` is the URL atkins prints. It shows the status, the command, the repository, the ref the job asked for and the commit the agent resolved it to, timing, the exit code, the output the agent captured and the artefacts it uploaded, and it refreshes itself until the job settles.

The output keeps its colour. atkins colours its tree whether or not anything is watching, so the page translates the SGR escape sequences into spans it styles, and drops the ones that move a cursor or clear a line — a document cannot honour those, and printing them as text is how a three-line tree turns into noise. What is stored is untouched: `GET /api/job/{id}/log` returns the bytes the command wrote, escape sequences and all, for a script that would rather strip or re-colour them itself.

The page has no session to check — the browser reading it never logged in — so what a private instance checks instead is the token in the URL. It is an HMAC of the job ID under the server's signing key: nothing is stored, nothing extra leaks from a database dump, and rotating the signing key invalidates every outstanding link along with every token. Paste the whole line and the job opens; trim the query string and you get a 403 saying so. Links between jobs on the page — the parent, and the children a job dispatched — carry the token for the job they point at, so following the tree keeps working. So do the artefact download links: a file a job produced is reachable on exactly the terms its page is, no wider and no narrower.

`/` lists recent jobs. On a private instance the session decides what it lists: signed in, it is your own runs, with each link carrying the view token for the job it points at; signed out, it lists nothing and says where to sign in, because there is no token that could scope a listing and no session to scope it by. The page answers either way — it is the front door, and a health check probing it should find a server. `GET /api/job` is the same listing for a bearer token, scoped the same way.

Both of those relax under `job.visibility`:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"public"}' "$SERVER/api/admin/setting/job.visibility"
```

A public instance is the single-team one: every authenticated user reads every job over the API, the job page opens for anyone holding the URL, `/` lists everything, and no token is issued — a secret in a URL that guards nothing is a habit worth not forming. It is a choice an admin makes, not the state a server starts in, because job output routinely contains things nobody meant to publish.

Whichever way it is set, the API scoping follows the same rule:

| Caller        | Sees                                                     |
|---------------|----------------------------------------------------------|
| admin         | every job                                                |
| agent         | every job — a worker operates on the whole queue         |
| user, private | their own jobs, and everything under a tree they started |
| user, public  | every job                                                |

"Everything under a tree they started" matters because a pipeline that clears `ATKINS_NO_DISPATCH` queues its children under the *agent's* credentials. Scoping on the dispatching user alone would hide a fan-out from the person who started it, so the root of the job tree decides too.

A job somebody may not read is reported as `404`, not `403`: "not yours" and "not here" have to look the same, or the endpoint tells a stranger which job IDs exist. The same check governs `retry` and `cancel` — a job you may not read is one you may not stop.

## Retention

An instance that runs for a year grows `job` and `job_log` without bound, and `job_log` is the one that hurts: it holds every line every build printed. Two windows bound it, and they are separate on purpose — output stops being interesting long before an outcome does:

```bash
# Keep captured output for a week.
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"168h"}' "$SERVER/api/admin/setting/job.log_retention"

# Keep the job records themselves for a year.
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"8760h"}' "$SERVER/api/admin/setting/job.retention"
```

`job.log_retention` deletes the output of a finished job and leaves the job: the page still says what ran, when, and whether it passed. `job.retention` deletes the record, and takes the output with it. `0` on either keeps that half forever, which is what `job.retention` defaults to.

Both windows are measured from when a job **finished**. A job that has not settled is never swept, however long ago it was queued — a pending job is not old, it is waiting, and a running job whose agent died is the lease sweep's problem.

Deletion happens in bounded batches, up to 500 rows at a time and 20 batches per pass, and a pass that runs out of batches leaves the rest for the next one. The first sweep of an instance that has been accumulating for a year is therefore a series of small deletes spread over a few passes rather than one that holds a table for minutes. The log says when a pass had more to do:

```text
[atkins] retention removed 0 job(s) and 10000 output row(s), more to come
```

How often the server looks is `server.retention_interval` (default `1h`), a start-up setting rather than a runtime one: the windows are policy, the cadence is a property of the machine. `0` turns the sweep off entirely.

Artefact downloads from the page — `/job/{ULID}/artefact/{ULID}` — are on the same terms, because a browser has no bearer token to offer and a download link that fails is not a link. `GET /api/job/{id}/artefact/{id}` is the authenticated door, and it is the one a script should use. A file a job produced is served as an attachment with `X-Content-Type-Options: nosniff`, so an artefact named `report.html` is something to save rather than something that runs in the server's origin.

## The admin pages

Everything under `/admin` is the operator's face on `/api/admin/*`, and it needs an administrator to be signed in:

| Page                | What it does                                                                  |
|---------------------|-------------------------------------------------------------------------------|
| `/admin/repository` | What the server has seen, with each repository's last job, and a trigger form |
| `/admin/allowlist`  | List, add, enable, disable and remove rules, with the policy in force         |
| `/admin/setting`    | Every setting with its effective value, default, and whether it is overridden |
| `/admin/user`       | Accounts and their `is_admin` / `is_active` / `is_agent` flags                |
| `/admin/ssh-key`    | Deploy keys with fingerprints: add, deactivate, remove                        |

Sign in at `/login` with the same account `atkins --login` uses — there is no separate web password. The first account on a fresh instance becomes an administrator, so `atkins --register <server>` is how you get in.

### The session

Signing in creates a session in the same `session` table the CLI uses and sets one cookie naming it. The cookie is the session id followed by an HMAC of it under the server signing key, so it cannot be forged without the key, and it is `HttpOnly`, `SameSite=Lax`, and `Secure` when the request arrived over TLS (directly, or through a proxy that sets `X-Forwarded-Proto`).

Nothing about the browser's session is special: it is revoked by signing out, it expires with `server.session_ttl`, and rotating `ATKINS_SIGNING_KEY` invalidates it along with every issued access token.

Forms carry a CSRF token — an HMAC of the session id, scoped so it is not the cookie value — and a cross-origin post is refused outright. A form that has been open long enough for the session to change comes back with "this form has expired; reload the page and try again".

The pages are plain HTML: a form post and a redirect. There is no JavaScript, no framework and no CDN.

The markup lives in `server/web/*.templ` and is compiled to Go by [templ](https://templ.guide). The generated `*_templ.go` files are committed next to their sources, so building the server needs nothing beyond the Go toolchain — a template that does not compile is a build failure rather than a page that breaks the first time somebody opens it, and an interpolated value is escaped for the context it lands in without anyone having to remember to do it. Regenerate after editing a `.templ` with `atkins templ`; `atkins fmt` already runs it first, so the usual formatting pass before a commit is enough.

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
  artefact_dir: artefacts
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
| `--artefact-dir`       | `server.artefact_dir`       | `ATKINS_ARTEFACT_DIR`       |
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
6. uploads the artefacts the job asked to keep, whatever the exit code;
7. reports `passed`, `failed` or `timeout`, and removes the work tree.

The work tree is checked out detached. A job builds one commit; leaving a branch checked out would suggest it has somewhere to push it back to.

The lease is renewed every 30 seconds. A job whose lease lapses is swept back out of `running` and marked `timeout`, so a worker that disappears mid-job doesn't strand its work.

Labels filter the queue. A job with no labels runs anywhere; a job requiring `linux,arm64` only lands on an agent advertising both:

```bash
atkins worker --labels linux,arm64,docker
```

## Repository allowlist

By default any repository a logged-in user dispatches will be built. To restrict that, switch the policy and write rules — `/admin/allowlist` does both, and says out loud when the combination stops everything. Over the API:

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

An agent clones public repositories anonymously. For private ones, give the server a key — paste it into `/admin/ssh-key`, or post it:

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

Runtime configuration an admin can change without a restart. `/admin/setting` renders the table below from the registry itself — kind, default and accepted values — so a setting added to the server appears there without a template change:

| Setting              | Default   | Purpose                                                       |
|----------------------|-----------|---------------------------------------------------------------|
| `repository.policy`  | `open`    | `open`, or `allowlist` to gate on rules                       |
| `registration.open`  | `false`   | Let anyone register                                           |
| `job.max_depth`      | `3`       | How deep a job may dispatch children                          |
| `job.lease_ttl`      | `15m`     | How long an agent may hold a job                              |
| `job.retention`      | `0`       | How long a finished job's record is kept; 0 is forever        |
| `job.log_retention`  | `720h`    | How long a finished job's output is kept; 0 is forever        |
| `job.visibility`     | `private` | `private` scopes jobs to who dispatched them; `public` shares |
| `artefact.max_size`  | `32MB`    | Largest single artefact an agent may upload                   |
| `artefact.max_count` | `50`      | How many artefacts one job may keep                           |
| `artefact.retention` | `0`       | How long artefact bytes are kept; 0 follows the job           |

```bash
curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/admin/setting" |
  jq -r '.[] | "\(.name)\t\(.value)\tdefault=\(.is_default)"'
```

## Users and roles

Three flags, because there are three things to decide:

- **`is_admin`** — the `/api/admin/*` surface. The API refuses to remove the last active admin, so an instance can't be locked out.
- **`is_agent`** — may claim jobs, report status, append output, and read deploy keys. Reachable only by enrolment, never by registration.
- **`is_active`** — login and every authenticated call. Deactivating takes effect immediately, not when the token expires.

`/admin/user` toggles all three. It also greys out the last active admin's admin and active buttons, because the server refuses to remove the last one and being told before the click beats being told after it. Over the API:

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
| `/api/job`                     | GET             | user   | List jobs the caller may see           |
| `/api/job/{id}`                | GET             | user   | Read one job the caller may see        |
| `/api/job/{id}/log`            | GET             | user   | Read captured output                   |
| `/api/job/{id}/artefact`       | GET             | user   | List the files a job produced          |
| `/api/job/{id}/artefact/{id}`  | GET             | user   | Download one artefact                  |
| `/api/job/{id}/retry`          | POST            | user   | Queue a copy of a finished job         |
| `/api/job/{id}/cancel`         | POST            | user   | Settle an unfinished job               |
| `/api/job/claim`               | POST            | agent  | Lease the oldest pending job           |
| `/api/job/{id}/status`         | POST            | agent  | Settle a job                           |
| `/api/job/{id}/checkout`       | POST            | agent  | Record the ref and commit it built     |
| `/api/job/{id}/heartbeat`      | POST            | agent  | Extend the lease                       |
| `/api/job/{id}/log`            | POST            | agent  | Append output                          |
| `/api/job/{id}/artefact`       | POST            | agent  | Upload a file, `?path=` names it       |
| `/api/agent/enrol`             | POST            | token  | Trade the shared token for credentials |
| `/api/agent/policy`            | GET             | agent  | The repository policy to enforce       |
| `/api/agent/ssh-key`           | GET             | agent  | Deploy keys, with private material     |
| `/api/admin/user[/{id}]`       | GET/POST        | admin  | List accounts, change flags            |
| `/api/admin/repository[/{id}]` | GET/POST/DELETE | admin  | Manage allowlist rules                 |
| `/api/admin/setting[/{name}]`  | GET/POST/DELETE | admin  | Read and change settings               |
| `/api/admin/ssh-key[/{id}]`    | GET/POST/DELETE | admin  | Manage deploy keys                     |

Authentication is `Authorization: Bearer <token>`. Access tokens live an hour and carry the session they came from, so logout takes effect immediately rather than when the token expires. Refresh tokens are single-use and rotate on every refresh, which makes a leaked one detectable: the legitimate client's next refresh fails.

## Pages

| Page                             | Method   | Who    | Purpose                              |
|----------------------------------|----------|--------|--------------------------------------|
| `/`                              | GET      | anyone | Recent jobs                          |
| `/job/{ULID}`                    | GET      | anyone | One run, and its output              |
| `/login`                         | GET/POST | anyone | Sign in, setting the session cookie  |
| `/logout`                        | POST     | user   | Revoke the session, clear the cookie |
| `/admin/repository`              | GET      | admin  | Repositories and their last job      |
| `/admin/repository/{id}/trigger` | POST     | admin  | Queue a job by name                  |
| `/admin/allowlist[/{id}]`        | GET/POST | admin  | Manage allowlist rules               |
| `/admin/setting`                 | GET/POST | admin  | Read and change settings             |
| `/admin/user[/{id}]`             | GET/POST | admin  | Accounts and flags                   |
| `/admin/ssh-key[/{id}]`          | GET/POST | admin  | Manage deploy keys                   |

The pages authenticate with the session cookie, not the bearer token, and every post carries a CSRF token from the page it belongs to.

## See Also

- [CLI Flags](./cli-flags) - Command-line reference
- [Automation (JSON/YAML)](./automation) - Machine-readable run output
- [Migrating from GitHub Actions](../migrating/migration-from-github-actions) - What carries over and what doesn't
