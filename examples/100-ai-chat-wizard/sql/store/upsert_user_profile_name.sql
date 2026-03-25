INSERT INTO user_profile (user_id, name, updated_at) VALUES (?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET name=excluded.name, updated_at=excluded.updated_at
