-- ssh_key: Deploy keys agents use to clone and fetch private repositories
CREATE TABLE IF NOT EXISTS ssh_key (
    id CHAR(26) PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    -- host scopes the key to one git host, e.g. github.com. Empty means
    -- the key is offered for any host.
    host TEXT NOT NULL DEFAULT '',
    private_key TEXT NOT NULL,
    public_key TEXT NOT NULL DEFAULT '',
    fingerprint TEXT NOT NULL DEFAULT '',
    known_hosts TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_by_user_id CHAR(26) NOT NULL,
    last_used_at DATETIME,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_ssh_key_name ON ssh_key(name);
CREATE INDEX IF NOT EXISTS idx_ssh_key_host ON ssh_key(host);
CREATE INDEX IF NOT EXISTS idx_ssh_key_deleted_at ON ssh_key(deleted_at);
