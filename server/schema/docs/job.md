# Job

Job.

| Name              | Type     | Key | Comment           |
|-------------------|----------|-----|-------------------|
| id                | char(26) | PRI | ID                |
| parent_id         | char(26) | MUL | Parent ID         |
| root_id           | char(26) | MUL | Root ID           |
| depth             | bigint   |     | Depth             |
| repository_id     | char(26) | MUL | Repository ID     |
| user_id           | char(26) | MUL | User ID           |
| working_directory | varchar  |     | Working Directory |
| command           | varchar  |     | Command           |
| labels            | varchar  |     | Labels            |
| params            | varchar  |     | Params            |
| status            | varchar  | MUL | Status            |
| exit_code         | bigint   |     | Exit Code         |
| error             | varchar  |     | Error             |
| agent_id          | varchar  | MUL | Agent ID          |
| lease_expires_at  | datetime | MUL | Lease Expires At  |
| started_at        | datetime |     | Started At        |
| finished_at       | datetime |     | Finished At       |
| created_at        | datetime | MUL | Created At        |
| updated_at        | datetime |     | Updated At        |
| artefact_paths    | varchar  |     | Artefact Paths    |
| ref               | varchar  |     | Ref               |
| commit_sha        | varchar  | MUL | Commit Sha        |
| clone_depth       | bigint   |     | Clone Depth       |
| interactive       | boolean  |     | Interactive       |
