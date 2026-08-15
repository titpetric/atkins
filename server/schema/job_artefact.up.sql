-- job_artefact: A file a job produced, uploaded by the agent that ran it
CREATE TABLE IF NOT EXISTS job_artefact (
    id CHAR(26) PRIMARY KEY NOT NULL,
    job_id CHAR(26) NOT NULL,
    -- path is the name the pipeline gave the file, relative to the
    -- directory the job ran in. It is what a person recognizes the
    -- artefact by, and never where the bytes actually are.
    path TEXT NOT NULL,
    -- storage_key addresses the bytes in the blob store. Holding it
    -- apart from path is what lets a filesystem root be swapped for an
    -- object store later without reshaping this table or the API.
    storage_key TEXT NOT NULL DEFAULT '',
    size INTEGER NOT NULL DEFAULT 0,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    -- checksum is the SHA256 of the bytes, computed while they were
    -- written. A truncated upload is detectable rather than silently
    -- half a file.
    checksum CHAR(64) NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME,
    -- deleted_at marks an artefact whose bytes retention has swept.
    -- The row stays behind on purpose: it is a few dozen bytes, it
    -- records that the file existed and how big it was, and it is what
    -- makes the removal auditable rather than silent.
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_job_artefact_job_id_path ON job_artefact(job_id, path);
CREATE INDEX IF NOT EXISTS idx_job_artefact_created_at ON job_artefact(created_at);
CREATE INDEX IF NOT EXISTS idx_job_artefact_deleted_at ON job_artefact(deleted_at);
