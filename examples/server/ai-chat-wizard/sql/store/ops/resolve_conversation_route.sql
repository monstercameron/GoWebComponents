SELECT id, public_id
FROM conversations
WHERE user_id = ?
  AND public_id = ?
LIMIT 1
