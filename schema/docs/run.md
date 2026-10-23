# Run

Run.

| Name          | Type     | Key | Comment       |
|---------------|----------|-----|---------------|
| id            | char(26) | PRI | ID            |
| invocation_id | char(26) | MUL | Invocation ID |
| parent_id     | char(26) | MUL | Parent ID     |
| kind          | varchar  | MUL | Kind          |
| name          | varchar  | MUL | Name          |
| pipeline      | varchar  |     | Pipeline      |
| file          | varchar  |     | File          |
| dir           | varchar  |     | Dir           |
| module        | varchar  |     | Module        |
| repository    | varchar  |     | Repository    |
| branch        | varchar  |     | Branch        |
| commit_sha    | varchar  |     | Commit Sha    |
| result        | varchar  |     | Result        |
| error         | varchar  |     | Error         |
| steps         | bigint   |     | Steps         |
| duration_ms   | bigint   |     | Duration Ms   |
| started_at    | datetime | MUL | Started At    |
| finished_at   | datetime |     | Finished At   |
| created_at    | datetime |     | Created At    |
