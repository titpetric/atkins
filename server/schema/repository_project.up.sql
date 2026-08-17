-- Give a repository the handful of fields that make it a project.
--
-- A repository row used to be something the server discovered: a slug it
-- derived the first time somebody dispatched a run from a checkout. The
-- setup flow adds the other direction — a person names a project and
-- pastes a clone URL — and that person has things to say the slug cannot
-- carry.
--
-- `name` is what a human calls it. The slug stays the identity, because
-- two people naming the same remote differently must not produce two
-- repositories.
--
-- `command`, `ref` and `working_directory` are the defaults a run starts
-- from: the pipeline arguments, which ref to build, and where in the work
-- tree the pipeline file lives. They are defaults rather than settings —
-- a job records what it was actually dispatched with.
--
-- `pipeline` is the cached `atkins --list --json` of the project, and
-- `pipeline_job_id` the job that produced it. The tree is read from an
-- artefact of that job the first time it is needed and kept here after,
-- so the page survives the artefact being swept by retention.
ALTER TABLE repository ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE repository ADD COLUMN command TEXT NOT NULL DEFAULT '';
ALTER TABLE repository ADD COLUMN ref TEXT NOT NULL DEFAULT '';
ALTER TABLE repository ADD COLUMN working_directory TEXT NOT NULL DEFAULT '';
ALTER TABLE repository ADD COLUMN pipeline_job_id CHAR(26) NOT NULL DEFAULT '';
ALTER TABLE repository ADD COLUMN pipeline TEXT NOT NULL DEFAULT '';
ALTER TABLE repository ADD COLUMN pipeline_at DATETIME;
