# Session

Session.

| Name          | Type     | Key | Comment       |
|---------------|----------|-----|---------------|
| id            | char(26) | PRI | ID            |
| user_id       | char(26) | MUL | User ID       |
| refresh_token | varchar  | MUL | Refresh Token |
| hostname      | varchar  |     | Hostname      |
| user_agent    | varchar  |     | User Agent    |
| remote_addr   | varchar  |     | Remote Addr   |
| last_seen_at  | datetime |     | Last Seen At  |
| expires_at    | datetime | MUL | Expires At    |
| revoked_at    | datetime |     | Revoked At    |
| created_at    | datetime |     | Created At    |
| updated_at    | datetime |     | Updated At    |
