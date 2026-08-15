# Repository

Repository.

| Name               | Type     | Key | Comment            |
|--------------------|----------|-----|--------------------|
| id                 | char(26) | PRI | ID                 |
| slug               | varchar  | MUL | Slug               |
| remote_url         | varchar  |     | Remote URL         |
| default_branch     | varchar  |     | Default Branch     |
| created_by_user_id | char(26) | MUL | Created By User ID |
| is_active          | boolean  |     | Is Active          |
| created_at         | datetime |     | Created At         |
| updated_at         | datetime |     | Updated At         |
| deleted_at         | datetime | MUL | Deleted At         |
