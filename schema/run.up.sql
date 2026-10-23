-- run: One node of what atkins ran, the pipeline, a job or a step, with the time it took including everything under it
CREATE TABLE IF NOT EXISTS run (
    id CHAR(26) PRIMARY KEY NOT NULL,
    -- run.id of the pipeline node this belongs to, the same value for every node of one invocation
    invocation_id CHAR(26) NOT NULL,
    -- run.id of the node above this one, empty for the pipeline node
    parent_id CHAR(26) NOT NULL DEFAULT '',
    -- What the node is
    kind TEXT NOT NULL DEFAULT 'step' CHECK (kind IN ('pipeline', 'job', 'step')),
    -- The node as the tree shows it: a job name, a step label, the pipeline name
    name TEXT NOT NULL DEFAULT '',
    -- The pipeline the invocation ran, as its name: block spells it
    pipeline TEXT NOT NULL DEFAULT '',
    -- The pipeline file, as the command line named or discovered it
    file TEXT NOT NULL DEFAULT '',
    -- The directory atkins ran in
    dir TEXT NOT NULL DEFAULT '',
    -- The go module the directory belongs to, empty outside one
    module TEXT NOT NULL DEFAULT '',
    -- The repository the remote URL names, owner/name
    repository TEXT NOT NULL DEFAULT '',
    -- The branch checked out when the run started
    branch TEXT NOT NULL DEFAULT '',
    -- The commit checked out when the run started
    commit_sha TEXT NOT NULL DEFAULT '',
    -- How the node ended
    result TEXT NOT NULL DEFAULT 'pass' CHECK (result IN ('pass', 'fail', 'skipped')),
    -- What went wrong, empty when nothing did
    error TEXT NOT NULL DEFAULT '',
    -- Steps counted under this node
    steps INTEGER NOT NULL DEFAULT 0,
    -- How long the node took, everything under it included
    duration_ms INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_run_invocation_id ON run(invocation_id);
CREATE INDEX IF NOT EXISTS idx_run_parent_id ON run(parent_id);
CREATE INDEX IF NOT EXISTS idx_run_kind_name ON run(kind, name);
CREATE INDEX IF NOT EXISTS idx_run_started_at ON run(started_at);
