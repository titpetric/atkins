-- repository: A git repository seen by the server, identified by its normalized remote URL
CREATE TABLE IF NOT EXISTS repository (
    id CHAR(26) PRIMARY KEY NOT NULL,
    slug TEXT NOT NULL,
    remote_url TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT '',
    created_by_user_id CHAR(26) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_repository_slug ON repository(slug);
CREATE INDEX IF NOT EXISTS idx_repository_created_by_user_id ON repository(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_repository_deleted_at ON repository(deleted_at);
