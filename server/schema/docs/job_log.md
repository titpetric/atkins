# Job Log

Job Log.

| Name       | Type     | Key | Comment    |
|------------|----------|-----|------------|
| id         | char(26) | PRI | ID         |
| job_id     | char(26) | MUL | Job ID     |
| seq        | bigint   | MUL | Seq        |
| stream     | varchar  |     | Stream     |
| content    | varchar  |     | Content    |
| created_at | datetime |     | Created At |
