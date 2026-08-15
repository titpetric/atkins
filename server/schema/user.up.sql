-- user: A person who authenticates against the CI/CD server with `atkins --login`
CREATE TABLE IF NOT EXISTS user (
    id CHAR(26) PRIMARY KEY NOT NULL,
    email TEXT NOT NULL,
    username TEXT NOT NULL,
    full_name TEXT NOT NULL DEFAULT '',
    password TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_email ON user(email);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_username ON user(username);
CREATE INDEX IF NOT EXISTS idx_user_deleted_at ON user(deleted_at);
