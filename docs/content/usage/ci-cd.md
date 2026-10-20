---
title: CI/CD Server
subtitle: Distributed job dispatch with atkins --login
layout: page
---

Atkins can attach to a CI/CD server. Logging in changes what atkins **records** and leaves the place a run happens alone: `atkins` still runs the pipeline on the machine you typed it on, and the server receives the job an agent would have produced: the same command, checkout, output and exit code. `atkins --dispatch` is the opt-in that sends a run to an agent.

The design keeps atkins in charge. The server is a queue and a ledger; it does not decide what a pipeline does. Three things travel with every run: the git repository, the directory inside its work tree, and the atkins command. Another machine needs exactly these three to reproduce the run.

```text
laptop                     server (:3200)            agent (container)
------                     --------------            -----------------
atkins --login  ─────────► /api/user/login

atkins          ─────────► /api/dispatch  (agent: this machine)
   runs here                              │  job starts running
                           /api/job/{id}/log ◄─ the run's transcript
                           /api/job/{id}/status ◄ terminal state

atkins --dispatch ───────► /api/dispatch  ──┐
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
atkins                                    # runs here, recorded at http://localhost:3200/job/<ULID>?t=<token>
atkins --dispatch                         # queues it for the agent instead
atkins down
```

The containers mount nothing from the host. The agent clones the repository your checkout reports, exactly as another machine would, so a job runs against what the remote actually has, and your working tree plays no part. State lives in the containers and goes away with them.

Settings for the instance live in two files: `.env.docker`, which both services load, and `.env.docker.server`, which the server alone loads. The split keeps `ATKINS_SIGNING_KEY` in the server's process, where the server issues tokens, while `ATKINS_AGENT_TOKEN` reaches both services, as the agent presents it on enrolment and the server verifies it.

Both values ship as development placeholders. For an instance that outlives the trial, generate replacements with `openssl rand -hex 32`, keep them in the deployment's secret store, and pass them to the services through the environment, never through a committed file. `ATKINS_SIGNING_KEY` signs every token the server issues, admin tokens among them, and rotating it invalidates every issued token and browser session. `ATKINS_AGENT_TOKEN` admits an agent to the queue and to the deploy keys, so it deserves the same handling and a rotation whenever you decommission an agent host. The agent keeps the token to itself; see [what a job receives](#environment-exported-to-a-job).

## Logging in

```bash
atkins --login https://ci.example.com
```

Atkins prompts for an email and password, exchanges them for an access token and a refresh token, and writes them to `~/.atkins/credentials.json` (mode `0600`). The file holds one credential per server, so a machine can attach to more than one instance.

To create an account:

```bash
atkins --register https://ci.example.com
```

Registration prompts for a username, an email and a password, then logs the new account in. The **first** human account on a fresh instance is always allowed and becomes an admin, so you can bootstrap a new server without touching its database. Later registrations need `registration.open`. Agents do not count as humans, so an agent starting first neither takes that slot nor closes the door behind it.

To detach a machine:

```bash
atkins --logout
```

Logout revokes the session on the server and removes the local credential. It succeeds locally even when the server is unreachable.

### Non-interactive login

Set the credentials in the environment and atkins shows no prompt. A container or a provisioning script attaches itself this way:

| Variable          | Purpose                     |
|-------------------|-----------------------------|
| `ATKINS_EMAIL`    | Email, skips the prompt     |
| `ATKINS_USERNAME` | Username, `--register` only |
| `ATKINS_PASSWORD` | Password, skips the prompt  |

## What a run records

A logged-in machine posts every run to `/api/dispatch` and goes on to run the pipeline itself. The server creates the job **running** and leases it to this machine, skipping the queue, and the same terminal that draws the tree reports the outcome:

```bash
$ atkins test
recording: http://localhost:3200/job/01J8…?t=…
Atkins project tests, build
└─ test:simple - Simple go test with gotestsum ✓
```

What lands on the job page is the finished tree, the commit it ran against, the exit code the shell saw, and how long it took. Atkins records a run stopped with ctrl-C as cancelled. A run that dies with the terminal stops renewing its lease, and the server reclaims it as a timeout, the same way it reclaims a lost agent's job.

The transcript arrives in one piece when the run finishes, so a page opened mid-run says which machine runs what and fills in at the end. A local run collects no artefacts: nothing here has an `ATKINS_ARTEFACTS` directory to sweep.

To run without recording anything:

```bash
atkins --local                 # this run only
```

Setting `client.record: false` in the configuration turns recording off for good, and a machine that never logged in has nothing to record to.

## What a run dispatches

`atkins --dispatch` sends the run to an agent, skips running it here, and prints the job URL:

| Field               | Value                                                        |
|---------------------|--------------------------------------------------------------|
| `repository`        | `origin` remote URL, ref and default branch                  |
| `working_directory` | Pipeline directory, relative to the repository root          |
| `command`           | The atkins invocation, e.g. `atkins test:build`              |
| `parent_id`         | `ATKINS_JOB_ID`, when a job dispatches further work          |
| `labels`            | Which agents may run it                                      |
| `clone_depth`       | History the agent's work tree needs; 0 keeps the full history |
| `artefacts`         | Globs the agent collects when the command exits              |
| `agent`             | Set only by a recorded local run: the machine running it now |

The server normalizes the remote URL into a `host/owner/name` slug, so `git@github.com:you/repo.git` and `https://github.com/you/repo` are one repository. The server creates repositories on first sight; nothing needs registering up front.

The ref atkins sends is the **commit you are on**. A dispatched run belongs to the code in front of you, and a branch name would let the agent build whatever that branch had moved to by the time it claimed the job.

That is why `--dispatch` inspects the work tree before it queues anything:

```text
$ atkins --dispatch test
ERROR: the working tree has uncommitted changes: an agent would build 41c9d0e2a7b1, which does not have them

$ atkins --dispatch test
ERROR: HEAD has not been pushed to a remote: an agent clones the repository, and 41c9d0e2a7b1 is only on this machine
```

Atkins refuses both here, while committing and pushing still lie in front of you, and spares you a job that dies in `git checkout` minutes later. The test looks at the commit and ignores the branch: a clone can reach any commit a remote has, whether the local branch tracks anything or not.

The response carries the job's ID and, while the instance keeps jobs private, a `view_token` that opens its page in a browser. `atkins` prints the two as one URL; a script driving `/api/dispatch` or a trigger with `curl` builds it the same way. Losing that line is not losing the link: `GET /api/job/{id}` and `GET /api/job` return the same `view_token`, plus a ready-made `url`, to any caller allowed to read the job.

`--dispatch` fails outright and never degrades. A machine that isn't logged in, isn't inside a git repository or can't reach the server cannot carry out the hand-over, and quietly running the job here would leave a person watching a queue that holds nothing. Recording works the opposite way: recording is bookkeeping, so it drops out silently and never costs the run it describes.

Setting `client.dispatch: true` in the configuration makes dispatch the default for a machine, as if every run carried the flag.

`ATKINS_NO_DISPATCH=1` outranks both, and an agent sets it for the command it runs. Without it the atkins the agent runs would send the work straight back to the server, and a job recorded as `atkins --dispatch test` would bounce forever; the guard keeps that job on the agent that claimed it.

### Environment exported to a job

An agent publishes these to the command it runs:

| Variable               | Value                                       |
|------------------------|---------------------------------------------|
| `ATKINS_JOB_ID`        | This job                                    |
| `ATKINS_PARENT_JOB_ID` | The job that dispatched it, if any          |
| `ATKINS_ROOT_JOB_ID`   | Top of the job tree, stable across nesting  |
| `ATKINS_JOB_PARAMS`    | The JSON object from the dispatch request   |
| `ATKINS_WORKSPACE`     | The checkout the job runs in                |
| `ATKINS_ARTEFACTS`     | Directory whose contents the agent uploads  |
| `ATKINS_REPOSITORY`    | Repository slug                             |
| `ATKINS_REF`           | The ref the agent checked out               |
| `ATKINS_COMMIT_SHA`    | The commit that ref resolved to             |
| `ATKINS_REVISION`      | The same commit, under its older name       |
| `ATKINS_BRANCH`        | Set only when the ref named a branch        |
| `ATKINS_SERVER`        | The server that receives child jobs         |
| `CI`                   | `true`                                      |

It also sets `ATKINS_NO_DISPATCH=1`. Without it, the atkins the agent runs would see the agent's own credentials, send the work straight back to the server, and nothing would ever execute. A pipeline that needs to queue child work clears it:

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

Children read `ATKINS_JOB_ID`, and the server records them under it. `job.max_depth` (3 by default) bounds nesting, so a pipeline that dispatches itself cannot run away with the queue.

### The agent's own settings

The table above lists the whole of the `ATKINS_*` a command sees. The agent sanitizes its own environment before it starts the command: it removes every `ATKINS_*` and `PLATFORM_*` variable in its process, then sets the job's variables by name.

The filter protects the credentials the agent runs with. `ATKINS_AGENT_TOKEN` admits its holder to the queue, to jobs for other repositories, and to the deploy keys with their private halves, so removing it keeps a job inside its own repository. `ATKINS_SIGNING_KEY` signs every token the server issues, and an installation with the server and the agent on one host has both values in one environment. The agent removes `PLATFORM_*` alongside them, as a database URL carries its own password.

The rest of the environment reaches the command unchanged: `PATH`, and anything else the agent's own environment holds at start. A job finds its tooling this way. Name a value a job needs outside the `ATKINS_` namespace, and put secrets a pipeline needs in the job's own configuration, away from the agent's environment.

## Triggering without a checkout

A cron, a webhook receiver or another job can queue work for a repository the server already knows, without a checkout of its own. This is the "POST a job name to a project" trigger; `/admin/repository` puts a form on it for the times a person is doing the queuing:

```bash
REPO=$(curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository" |
  jq -r '.[] | select(.slug == "github.com/titpetric/atkins") | .id')

curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
  -d '{"job": "analyze"}'
```

`job` becomes `atkins analyze`, so a trigger payload stays a name and never turns into a shell string somebody has to quote. `command` overrides the whole invocation when a bare job name isn't enough. With no `ref`, the agent resolves the repository's default branch when it runs the job, which suits a nightly.

`params` carries each child's work from a dispatching job. It reaches the job as `ATKINS_JOB_PARAMS`:

```bash
for tag in $(git tag --list); do
  curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
    -d "$(jq -n --arg tag "$tag" --arg parent "$ATKINS_JOB_ID" \
      '{job: "analyzeTag", parent_id: $parent, ref: $tag, clone_depth: 1, params: {tag: $tag}}')"
done
```

`ref` is what the child checks out; `params` is the input the child reads. Here they carry the same tag, and the child gets one commit of history, never the whole repository.

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

Atkins looks up a bare name as a tag first and then as a branch, which is git's own order. A fully qualified `refs/...` name means exactly that ref.

**A ref that does not resolve fails the job, naming the ref.** Nothing falls back to the default branch: a typo in a tag name should not produce a green run of the wrong code.

**The job records the commit.** Before it runs anything the agent reports what it actually checked out, and the job page shows the ref and the commit side by side. A tag moves; nobody could reproduce a job that remembered only `v1.2.3` after its tag moved. `ATKINS_COMMIT_SHA` carries the same value into the pipeline, so an artefact can carry the commit in its label even when the job named a branch.

A retry repeats the **ref**, and the commit can differ: a retried job pointed at a branch builds whatever that branch holds now. Put a commit in the ref to get the same code twice.

### Clone depth

`clone_depth` limits the history the job's work tree carries. `1` is one commit, and `0`, the default, is the full history.

Depth applies to the **work tree**, never to the agent's repository cache. The cache is a full mirror shared by every job for that repository, and it stays that way: a shallow cache would need deepening the first time a job named an older tag, and deepening repeatedly costs more than never discarding the objects. The agent throws the work tree away when the job ends, so limiting it costs nothing later.

What a shallow work tree buys is a small `.git`: it matters when the job packages the tree, copies it into a docker build context, or uploads it. It does not make the clone faster: the agent hardlinks a full work tree out of the cache at nearly no cost, while it has to fetch a shallow one. Set a depth when the size of `.git` matters to the job; a depth saves the agent no time.

A job with `clone_depth: 1` has no history and no tags: `git describe`, `git log` past the tip and `git merge-base` will not work, even for the tag the job itself named. `ATKINS_REF` carries that name into the job, and that name covers what a build typically needs `git describe` for. Fan-outs over tags are the case that suits a depth: each child builds one commit and never looks behind it.

## Artefacts

A job that produces a file (a scan, a coverage report, a built binary) can push it back to the server and attach it to the job. This is the "collect the outputs and data mine them centrally" half of a CI system: the agent's disk goes away with the work tree, and the server keeps what mattered.

A job can say what to keep two ways, and use both.

**Copy it into `$ATKINS_ARTEFACTS`.** The agent creates that directory before the command runs, and uploads whatever is in it afterwards, named by its path within the directory:

```yaml
jobs:
  scan:
    steps:
      - ./.atkins/bin/scan-repository > scan.json
      - cp scan.json "$ATKINS_ARTEFACTS/"
```

Nothing needs a declaration anywhere, no schema changes with your pipeline, and it works for any command in any language: a job says what to keep by putting it somewhere.

**Declare a glob when you dispatch or trigger.** For a pipeline you leave unedited, one that writes `coverage.json` where it writes it, the job can name what to pick up:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/repository/$REPO/trigger" \
  -d '{"job": "test", "artefacts": ["coverage.json", "reports/**/*.json"]}'
```

Patterns are relative to the directory the job ran in. `*` matches within a path segment and `**` across segments, the same as the repository allowlist patterns. `.git` is never collected.

Collection runs **after the command exits, whatever it exited with**, a timeout included. The artefacts of a failure are often the ones worth having.

The agent walks the checkout and matches what it finds; the filesystem never sees the pattern. A pattern can only ever select among files already inside the checkout, the agent does not follow symlinks, and both the server at job creation and the agent drop a path that attempts to leave, such as `../../etc/passwd`.

Reading them back:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/artefact" |
  jq -r '.[] | "\(.path)\t\(.size)\t\(.checksum)"'

curl -sSL -H "Authorization: Bearer $TOKEN" -O "$SERVER/api/job/$JOB/artefact/$ARTEFACT"
```

The job page lists them with download links, so the URL atkins printed is also where the files are.

**Where the bytes live.** The database records what an artefact is (job, path, size, media type, SHA256), and the file itself goes under `server.artefact_dir` as `<job-id>/<artefact-id>`. That is a directory an operator can back up, rsync or mount on a volume. The server streams the upload straight to it and hashes it on the way past, so the checksum describes the bytes the server received, independent of the agent's claim. The server refuses an upload that arrives short of its declared checksum.

**Limits.** `artefact.max_size` bounds one file and `artefact.max_count` bounds how many one job may keep. Both are settings and not constants, so stopping a full disk takes one `curl` call and no redeploy. Re-uploading a path replaces the file at that path.

**Retention.** `artefact.retention` sets how long the server keeps the bytes; `0` follows `job.retention`. The sweep runs on the same ticker as the lease reclaim, removes the file and marks the row deleted, so the record of an artefact such as `scan.json` survives its sweep while the bytes go. Artefacts are the thing that fills a disk, and they have their own setting because keeping a year of job history alongside a week of files is a normal combination; the reverse never is.

## Retrying and cancelling

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/retry"
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/job/$JOB/cancel"
```

A retry creates a **new job** from the finished one and leaves the previous attempt alone: it keeps its output and its outcome, so you can still look at a failure after retrying it. A retried child stays under the job that dispatched it. The server retries only finished jobs; cancel a running one first.

Cancelling settles the record and stops the command. The agent finds out at its next heartbeat, which the server refuses once it has taken the job's lease back: the agent kills the command's process group, notes on the page that the server took the job back, and goes to look for the next one. It reports no outcome of its own, so a cancelled job stays cancelled, and it skips uploading the artefacts collected so far, because a page that says `cancelled` should not go on growing files. In the worst case the build runs for one heartbeat interval (`agent.heartbeat_interval`, 30s by default) after the click.

## The job page

`/job/{ULID}?t={token}` is the URL atkins prints. It shows the status, the command, the repository, the ref the job names and the commit the agent resolved it to, timing, the exit code, the output the agent captured and the artefacts it uploaded, and it refreshes itself until the job settles.

The output keeps its colour. atkins colours its tree whether anything watches or not, so the page translates the SGR escape sequences into spans it styles, and drops the ones that move a cursor or clear a line: a document cannot honour those, and printing them as text turns a three-line tree into noise. The stored output stays untouched: `GET /api/job/{id}/log` returns the bytes the command wrote, escape sequences and all, for a script that strips or re-colours them itself.

The page has no session to check, since the browser reading it never logged in, so a private instance checks the token in the URL. The token is an HMAC of the job ID under the server's signing key: the server stores nothing, a database dump leaks nothing extra, and rotating the signing key invalidates every outstanding link along with every token. Paste the whole line and the job opens; trim the query string and you get a 403 saying so. Links between jobs on the page (the parent, and the children a job dispatched) carry the token for the job they point at, so following the tree keeps working. The artefact download links carry it too: a file a job produced is reachable on exactly the terms its page is, no wider and no narrower.

`/` lists recent jobs. On a private instance the session decides what it lists: signed in, the page lists your own runs, with each link carrying the view token for the job it points at; signed out, it lists nothing and says where to sign in, because no token could scope a listing and no session exists to scope it by. The page answers either way: the page is the front door, and a health check probing it should find a server. `GET /api/job` is the same listing for a bearer token, scoped the same way.

Both of those relax under `job.visibility`:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"public"}' "$SERVER/api/admin/setting/job.visibility"
```

A public instance suits a single team: every authenticated user reads every job over the API, the job page opens for anyone holding the URL, `/` lists everything, and the server issues no token, since a secret in a URL that guards nothing is a habit worth not forming. An admin makes this choice, and a server starts private, because job output routinely contains things nobody intends to publish.

Whichever way the setting goes, the API scoping follows the same rule:

| Caller        | Sees                                                     |
|---------------|----------------------------------------------------------|
| admin         | every job                                                |
| agent         | every job; a worker operates on the whole queue          |
| user, private | their own jobs, and everything under a tree they started |
| user, public  | every job                                                |

"Everything under a tree they started" matters because a pipeline that clears `ATKINS_NO_DISPATCH` queues its children under the *agent's* credentials. Scoping on the dispatching user alone would hide a fan-out from the person who started it, so the root of the job tree decides too.

The server reports a job somebody may not read as `404`, not `403`: "not yours" and "not here" have to look the same, or the endpoint tells a stranger which job IDs exist. The same check governs `retry` and `cancel`: a job you may not read is one you may not stop.

## Retention

An instance that runs for a year grows `job` and `job_log` without bound, and `job_log` is the one that hurts: it holds every line every build printed. Two windows bound it, kept separate because output stops being interesting long before an outcome does:

```bash
# Keep captured output for a week.
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"168h"}' "$SERVER/api/admin/setting/job.log_retention"

# Keep the job records themselves for a year.
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"8760h"}' "$SERVER/api/admin/setting/job.retention"
```

`job.log_retention` deletes the output of a finished job and leaves the job: the page still says what ran, when, and whether it passed. `job.retention` deletes the record, and takes the output with it. `0` on either keeps that half forever, and `job.retention` defaults to `0`.

The server measures both windows from when a job **finished**. The sweep never touches a job that has not settled, no matter how long ago it entered the queue: a pending job waits and does not age, and a running job whose agent died is the lease sweep's problem.

Deletion happens in bounded batches, up to 500 rows at a time and 20 batches per pass, and a pass that runs out of batches leaves the rest for the next one. So the first sweep of an instance that has accumulated rows for a year becomes a series of small deletes spread over a few passes, and no single delete holds a table for minutes. The log says when a pass had more to do:

```text
[atkins] retention removed 0 job(s) and 10000 output row(s), more to come
```

`server.retention_interval` (default `1h`) sets how often the server looks. This start-up setting sits outside the runtime settings: the windows are policy, the cadence is a property of the machine. `0` turns the sweep off entirely.

Artefact downloads from the page (`/job/{ULID}/artefact/{ULID}`) follow the same terms, because a browser has no bearer token to present and a download link that fails is no link at all. `GET /api/job/{id}/artefact/{id}` is the authenticated door, and scripts should use that one. The server sends a file a job produced as an attachment with `X-Content-Type-Options: nosniff`, so an artefact named `report.html` downloads as a file to save and never runs in the server's origin.

## The admin pages

Everything under `/admin` is the operator's face on `/api/admin/*`, and it needs a signed-in administrator:

| Page                | What it does                                                                  |
|---------------------|-------------------------------------------------------------------------------|
| `/admin/repository` | What the server has seen, with each repository's last job, and a trigger form |
| `/admin/allowlist`  | List, add, enable, disable and remove rules, with the policy in force         |
| `/admin/setting`    | Every setting with its effective value, default, and any override in force    |
| `/admin/user`       | Accounts and their `is_admin` / `is_active` / `is_agent` flags                |
| `/admin/ssh-key`    | Deploy keys with fingerprints: add, deactivate, remove                        |

Sign in at `/login` with the same account `atkins --login` uses; the web has no separate password. The first account on a fresh instance becomes an administrator, so `atkins --register <server>` is how you get in.

### The session

Signing in creates a session in the same `session` table the CLI uses and sets one cookie naming it. The cookie is the session id followed by an HMAC of it under the server signing key, so nobody can forge it without the key, and the cookie carries `HttpOnly`, `SameSite=Lax`, and `Secure` when the request arrived over TLS (directly, or through a proxy that sets `X-Forwarded-Proto`).

The browser's session has nothing special about it: signing out revokes it, it expires with `server.session_ttl`, and rotating `ATKINS_SIGNING_KEY` invalidates it along with every issued access token.

Forms carry a CSRF token (an HMAC of the session id, scoped so it differs from the cookie value), and the server refuses a cross-origin post outright. A form left open long enough for the session to change shows "this form has expired; reload the page and try again".

The pages are plain HTML: a form post and a redirect. No JavaScript, no framework, no CDN.

The markup lives in `server/web/*.templ`, and [templ](https://templ.guide) compiles it to Go. The repository commits the generated `*_templ.go` files next to their sources, so building the server needs nothing beyond the Go toolchain. A template that does not compile fails the build, so no page breaks the first time somebody opens it, and templ escapes an interpolated value for the context it lands in without anyone having to remember to do it. Regenerate after editing a `.templ` with `atkins templ`; `atkins fmt` already runs it first, so the usual formatting pass before a commit is enough.

## Configuration

Two layers. `.atkins/config.yml` is the source of truth, and `ATKINS_*` variables overlay individual fields on top of it; an unset or empty variable leaves the configured value alone. Atkins checks every field on load, and an empty one takes its default from the document built into the binary.

```bash
atkins --config
```

opens a menu over the project document, creating it from the built-in defaults if it doesn't exist. The menu marks fields that the environment overrides, so an edit that won't take effect says so.

```yaml
client:
  server: https://ci.example.com
  labels: [linux, amd64]
  dispatch: false
  record: true

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

A server needs the signing key: a server signing tokens with a known key is one anyone can mint an admin token for. Rotating it invalidates every issued token, which is the intended way to log everyone out.

Migrations run on start. The binary registers one sql driver, modernc's pure Go sqlite, so the DSN scheme has to be `sqlite`. That keeps `CGO_ENABLED=0` for the release build, and the platform's connection provider receives only that driver:

```bash
atkins server --database "sqlite://file:/var/lib/atkins/atkins.db"
```

## Running an agent

```bash
atkins worker --server https://ci.example.com --token "$ATKINS_AGENT_TOKEN"
```

Agents don't have passwords. The agent trades one shared enrolment token for the same rotating credentials a human holds, so a fleet carries one secret and no password per worker. Enrolling twice with the same `--agent-id` returns the same account, so a restarted agent keeps its identity and its job history.

For each claimed job the agent:

1. mirrors the repository into `<data_dir>/repos/<slug>.git`, or fetches it when already cached;
2. resolves the job's ref against that mirror, once, into a commit, so a tag that moves mid-job cannot split the run across two commits;
3. builds a work tree in `<data_dir>/work/<job-id>` at that commit, shallow if the job sets a depth, and reports the ref and commit back to the server;
4. runs the command in the job's working directory;
5. streams the output back as it goes;
6. uploads the artefacts the job asked to keep, whatever the exit code;
7. reports `passed`, `failed` or `timeout`, and removes the work tree.

The agent checks the work tree out detached. A job builds one commit; leaving a branch checked out would suggest it has somewhere to push it back to.

The agent renews the lease on every heartbeat (`agent.heartbeat_interval`). The server sweeps a job whose lease lapses back out of `running` and marks it `timeout`, so a worker that disappears mid-job doesn't strand its work.

Labels filter the queue. A job with no labels runs anywhere; a job requiring `linux,arm64` only lands on an agent advertising both:

```bash
atkins worker --labels linux,arm64,docker
```

## Repository allowlist

By default the server builds any repository a logged-in user dispatches. To restrict that, switch the policy and write rules: `/admin/allowlist` does both, and says out loud when the combination stops everything. Over the API:

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

With the policy on and no rules, nothing runs.

Atkins enforces the rule twice. The server refuses the dispatch with a 403, and the agent checks again before cloning, because a job can outlive the rule that admitted it, and a job queued under an old rule must not build a repository the allowlist has since dropped. An agent that cannot read the policy runs nothing: a server outage must not turn an allowlist into an open door.

## Deploy keys

An agent clones public repositories anonymously. For private ones, add a key to the server through `/admin/ssh-key`, or post it:

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/api/admin/ssh-key" \
  -d "$(jq -n --arg key "$(cat ~/.ssh/id_ed25519)" \
    '{name: "github", host: "github.com", private_key: $key}')"
```

The server derives the public half and a SHA256 fingerprint on create, so it refuses a key that doesn't parse at that moment and no job fails on it later. An admin listing never returns private material; only an enrolled agent receives it, writes the keys to `<data_dir>/ssh` at `0600` and passes them to git with `IdentitiesOnly=yes`. The agent rebuilds the directory on each refresh, so a key deleted on the server stops working within the minute. Pin host keys with `known_hosts` on the key; without it the agent trusts on first use and never prompts, since a prompt would hang a job.

A container with no key agent can side-step ssh entirely for public repositories:

```bash
atkins worker --token "$TOKEN"   # with agent.prefer_https: true
```

which rewrites `git@host:owner/repo.git` to `https://host/owner/repo.git` before cloning.

## Settings

Runtime configuration an admin can change without a restart. `/admin/setting` renders the table below from the registry itself (kind, default and accepted values), so a setting added to the server appears there without a template change:

| Setting              | Default   | Purpose                                                       |
|----------------------|-----------|---------------------------------------------------------------|
| `repository.policy`  | `open`    | `open`, or `allowlist` to gate on rules                       |
| `registration.open`  | `false`   | Let anyone register                                           |
| `job.max_depth`      | `3`       | How deep a job may dispatch children                          |
| `job.lease_ttl`      | `15m`     | How long an agent may hold a job                              |
| `job.retention`      | `0`       | How long the server keeps a finished job's record; 0 is forever |
| `job.log_retention`  | `720h`    | How long the server keeps a finished job's output; 0 is forever |
| `job.visibility`     | `private` | `private` scopes jobs to who dispatched them; `public` shares |
| `artefact.max_size`  | `32MB`    | Largest single artefact an agent may upload                   |
| `artefact.max_count` | `50`      | How many artefacts one job may keep                           |
| `artefact.retention` | `0`       | How long the server keeps artefact bytes; 0 follows the job   |

```bash
curl -sS -H "Authorization: Bearer $TOKEN" "$SERVER/api/admin/setting" |
  jq -r '.[] | "\(.name)\t\(.value)\tdefault=\(.is_default)"'
```

## Users and roles

Three flags, because there are three things to decide:

- **`is_admin`**: the `/api/admin/*` surface. The API rejects removing the last active admin, so an instance can't lock itself out.
- **`is_agent`**: may claim jobs, report status, append output, and read deploy keys. Reachable only by enrolment, never by registration.
- **`is_active`**: login and every authenticated call. Deactivating takes effect immediately, not when the token expires.

`/admin/user` toggles all three. It also greys out the last active admin's admin and active buttons: the server rejects removing the last one, and a warning before the click beats an error after it. Over the API:

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
| `/api/job/{id}/status`         | POST            | runner | Settle a job                           |
| `/api/job/{id}/checkout`       | POST            | runner | Record the ref and commit it built     |
| `/api/job/{id}/heartbeat`      | POST            | runner | Extend the lease                       |
| `/api/job/{id}/log`            | POST            | runner | Append output                          |
| `/api/job/{id}/artefact`       | POST            | agent  | Upload a file, `?path=` names it       |
| `/api/agent/enrol`             | POST            | token  | Trade the shared token for credentials |
| `/api/agent/policy`            | GET             | agent  | The repository policy to enforce       |
| `/api/agent/ssh-key`           | GET             | agent  | Deploy keys, with private material     |
| `/api/admin/user[/{id}]`       | GET/POST        | admin  | List accounts, change flags            |
| `/api/admin/repository[/{id}]` | GET/POST/DELETE | admin  | Manage allowlist rules                 |
| `/api/admin/setting[/{name}]`  | GET/POST/DELETE | admin  | Read and change settings               |
| `/api/admin/ssh-key[/{id}]`    | GET/POST/DELETE | admin  | Manage deploy keys                     |

Authentication is `Authorization: Bearer <token>`. Access tokens live an hour and carry the session they came from, so logout takes effect immediately, even while an access token has time left. Refresh tokens are single-use and rotate on every refresh, which makes a leaked one detectable: the legitimate client's next refresh fails.

**runner** in the table is whoever runs the job: an agent, for anything it claimed, or the owner of a job already running, which covers the machine recording a local run. The server demands ownership outright and ignores readability, so making an instance public does not let every user settle everybody else's runs. The server also refuses a *pending* job: nobody has started it, so a report on it would be fiction, and `/cancel` is the way to stop it.

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
