-- Add the agent role to user.
--
-- An agent is a user that runs jobs rather than dispatching them. The
-- distinction matters because agents are handed things humans are not:
-- the job queue, and the ssh keys used to clone private repositories.
--
-- Agents enrol with a shared token rather than a password, so is_agent
-- cannot be reached by ordinary registration.
ALTER TABLE user ADD COLUMN is_agent BOOLEAN NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_user_is_agent ON user(is_agent);
