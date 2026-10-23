-- command: One command a run executed, with the normalized form the statistics group by
CREATE TABLE IF NOT EXISTS command (
    id CHAR(26) PRIMARY KEY NOT NULL,
    -- run.id of the pipeline node of the invocation that ran this
    invocation_id CHAR(26) NOT NULL,
    -- run.id of the node that ran this, empty when the event names none
    parent_id CHAR(26) NOT NULL DEFAULT '',
    -- Where the command came from: a step, a $() substitution, a variable
    kind TEXT NOT NULL DEFAULT 'step' CHECK (kind IN ('step', 'substitution', 'interpolation')),
    -- The command as the shell received it
    command TEXT NOT NULL DEFAULT '',
    -- The command with its arguments dropped, its flags sorted and its values removed, so every `go test -v` counts as one
    normalized TEXT NOT NULL DEFAULT '',
    -- The first word of the normalized form, the tool
    tool TEXT NOT NULL DEFAULT '',
    -- The directory the command ran in
    dir TEXT NOT NULL DEFAULT '',
    -- The code the process exited with
    exit_code INTEGER NOT NULL DEFAULT 0,
    -- What went wrong, empty when nothing did
    error TEXT NOT NULL DEFAULT '',
    -- How long the command took
    duration_ms INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_command_invocation_id ON command(invocation_id);
CREATE INDEX IF NOT EXISTS idx_command_parent_id ON command(parent_id);
CREATE INDEX IF NOT EXISTS idx_command_normalized ON command(normalized);
CREATE INDEX IF NOT EXISTS idx_command_tool ON command(tool);
CREATE INDEX IF NOT EXISTS idx_command_started_at ON command(started_at);
