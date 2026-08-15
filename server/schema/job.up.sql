-- job: One atkins invocation recorded by /api/dispatch, claimable by an agent
CREATE TABLE IF NOT EXISTS job (
    id CHAR(26) PRIMARY KEY NOT NULL,
    parent_id CHAR(26) NOT NULL DEFAULT '',
    root_id CHAR(26) NOT NULL DEFAULT '',
    depth INTEGER NOT NULL DEFAULT 0,
    repository_id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    working_directory TEXT NOT NULL DEFAULT '',
    command TEXT NOT NULL,
    branch TEXT NOT NULL DEFAULT '',
    revision TEXT NOT NULL DEFAULT '',
    labels TEXT NOT NULL DEFAULT '',
    params TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'passed', 'failed', 'timeout', 'cancelled')),
    exit_code INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    lease_expires_at DATETIME,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_job_status_created_at ON job(status, created_at);
CREATE INDEX IF NOT EXISTS idx_job_repository_id ON job(repository_id);
CREATE INDEX IF NOT EXISTS idx_job_user_id ON job(user_id);
CREATE INDEX IF NOT EXISTS idx_job_parent_id ON job(parent_id);
CREATE INDEX IF NOT EXISTS idx_job_root_id ON job(root_id);
CREATE INDEX IF NOT EXISTS idx_job_lease_expires_at ON job(lease_expires_at);
