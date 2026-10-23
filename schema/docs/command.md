# Command

Command.

| Name          | Type     | Key | Comment       |
|---------------|----------|-----|---------------|
| id            | char(26) | PRI | ID            |
| invocation_id | char(26) | MUL | Invocation ID |
| parent_id     | char(26) | MUL | Parent ID     |
| kind          | varchar  |     | Kind          |
| command       | varchar  |     | Command       |
| normalized    | varchar  | MUL | Normalized    |
| tool          | varchar  | MUL | Tool          |
| dir           | varchar  |     | Dir           |
| exit_code     | bigint   |     | Exit Code     |
| error         | varchar  |     | Error         |
| duration_ms   | bigint   |     | Duration Ms   |
| started_at    | datetime | MUL | Started At    |
| created_at    | datetime |     | Created At    |
