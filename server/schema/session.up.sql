-- session: A login issued to one atkins CLI installation; holds the refresh token
CREATE TABLE IF NOT EXISTS session (
    id CHAR(26) PRIMARY KEY NOT NULL,
    user_id CHAR(26) NOT NULL,
    refresh_token TEXT NOT NULL,
    hostname TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    remote_addr TEXT NOT NULL DEFAULT '',
    last_seen_at DATETIME,
    expires_at DATETIME,
    revoked_at DATETIME,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_session_refresh_token ON session(refresh_token);
CREATE INDEX IF NOT EXISTS idx_session_user_id ON session(user_id);
CREATE INDEX IF NOT EXISTS idx_session_expires_at ON session(expires_at);
