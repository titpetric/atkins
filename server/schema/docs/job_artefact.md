# Job Artefact

Job Artefact.

| Name         | Type     | Key | Comment      |
|--------------|----------|-----|--------------|
| id           | char(26) | PRI | ID           |
| job_id       | char(26) | MUL | Job ID       |
| path         | varchar  | MUL | Path         |
| storage_key  | varchar  |     | Storage Key  |
| size         | bigint   |     | Size         |
| content_type | varchar  |     | Content Type |
| checksum     | char(64) |     | Checksum     |
| agent_id     | varchar  | MUL | Agent ID     |
| created_at   | datetime | MUL | Created At   |
| deleted_at   | datetime | MUL | Deleted At   |
