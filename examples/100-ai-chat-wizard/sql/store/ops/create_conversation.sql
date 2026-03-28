INSERT INTO conversations (user_id, public_id, started_at)
SELECT ?, ?, ?
WHERE EXISTS(SELECT 1 FROM users WHERE id = ?)
