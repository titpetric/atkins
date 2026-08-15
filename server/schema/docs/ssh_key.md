# SSH Key

SSH Key.

| Name               | Type     | Key | Comment            |
|--------------------|----------|-----|--------------------|
| id                 | char(26) | PRI | ID                 |
| name               | varchar  | MUL | Name               |
| host               | varchar  | MUL | Host               |
| private_key        | varchar  |     | Private Key        |
| public_key         | varchar  |     | Public Key         |
| fingerprint        | varchar  |     | Fingerprint        |
| known_hosts        | varchar  |     | Known Hosts        |
| is_active          | boolean  |     | Is Active          |
| created_by_user_id | char(26) | MUL | Created By User ID |
| last_used_at       | datetime |     | Last Used At       |
| created_at         | datetime |     | Created At         |
| updated_at         | datetime |     | Updated At         |
| deleted_at         | datetime | MUL | Deleted At         |
