-- job_log: Output captured from a job, appended in chunks by the agent that ran it
CREATE TABLE IF NOT EXISTS job_log (
    id CHAR(26) PRIMARY KEY NOT NULL,
    job_id CHAR(26) NOT NULL,
    seq INTEGER NOT NULL DEFAULT 0,
    stream TEXT NOT NULL DEFAULT 'output' CHECK (stream IN ('output', 'error')),
    content TEXT NOT NULL DEFAULT '',
    created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_job_log_job_id_seq ON job_log(job_id, seq);
