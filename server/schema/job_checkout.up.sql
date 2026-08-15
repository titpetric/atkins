-- Replace branch and revision on job with an explicit checkout model.
--
-- `ref` is the request, named the way git names things: a branch, a tag,
-- a commit sha, or a fully qualified refname. Carrying `branch` and
-- `revision` side by side meant two columns for one decision, and a tag
-- had to be smuggled through the branch field and hope the agent's
-- fallback chain caught it.
--
-- `commit_sha` is the answer rather than the question: the commit the
-- agent actually checked out. A tag moves, so a job that records only
-- "v1.2.3" cannot be reproduced later.
--
-- `clone_depth` limits the history of the job's work tree; 0 is the
-- whole history. It is deliberately not called `depth`, which already
-- means how deeply this job nests under its parent.
ALTER TABLE job ADD COLUMN ref TEXT NOT NULL DEFAULT '';
ALTER TABLE job ADD COLUMN commit_sha TEXT NOT NULL DEFAULT '';
ALTER TABLE job ADD COLUMN clone_depth INTEGER NOT NULL DEFAULT 0;

-- Carry the old columns over: a pinned revision was the more specific of
-- the two, so it wins where a row has both.
UPDATE job SET ref = revision WHERE ref = '' AND revision <> '';
UPDATE job SET ref = branch WHERE ref = '' AND branch <> '';
UPDATE job SET commit_sha = revision WHERE commit_sha = '' AND length(revision) = 40;

ALTER TABLE job DROP COLUMN branch;
ALTER TABLE job DROP COLUMN revision;

CREATE INDEX IF NOT EXISTS idx_job_commit_sha ON job(commit_sha);
