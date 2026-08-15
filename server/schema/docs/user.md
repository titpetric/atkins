# User

User.

| Name       | Type     | Key | Comment    |
|------------|----------|-----|------------|
| id         | char(26) | PRI | ID         |
| email      | varchar  | MUL | Email      |
| username   | varchar  | MUL | Username   |
| full_name  | varchar  |     | Full Name  |
| password   | varchar  |     | Password   |
| is_admin   | boolean  |     | Is Admin   |
| is_active  | boolean  |     | Is Active  |
| created_at | datetime |     | Created At |
| updated_at | datetime |     | Updated At |
| deleted_at | datetime | MUL | Deleted At |
| is_agent   | boolean  | MUL | Is Agent   |
