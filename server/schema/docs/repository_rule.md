# Repository Rule

Repository Rule.

| Name               | Type     | Key | Comment            |
|--------------------|----------|-----|--------------------|
| id                 | char(26) | PRI | ID                 |
| pattern            | varchar  | MUL | Pattern            |
| description        | varchar  |     | Description        |
| is_active          | boolean  | MUL | Is Active          |
| created_by_user_id | char(26) | MUL | Created By User ID |
| created_at         | datetime |     | Created At         |
| updated_at         | datetime |     | Updated At         |
| deleted_at         | datetime | MUL | Deleted At         |
