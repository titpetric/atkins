-- setting: Server configuration that can be changed without a restart
CREATE TABLE IF NOT EXISTS setting (
    name TEXT PRIMARY KEY NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    updated_by_user_id CHAR(26) NOT NULL DEFAULT '',
    created_at DATETIME,
    updated_at DATETIME
);
