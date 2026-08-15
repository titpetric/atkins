-- repository_rule: Slug patterns naming which repositories agents may build
CREATE TABLE IF NOT EXISTS repository_rule (
    id CHAR(26) PRIMARY KEY NOT NULL,
    pattern TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_by_user_id CHAR(26) NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_repository_rule_pattern ON repository_rule(pattern);
CREATE INDEX IF NOT EXISTS idx_repository_rule_is_active ON repository_rule(is_active);
CREATE INDEX IF NOT EXISTS idx_repository_rule_deleted_at ON repository_rule(deleted_at);
