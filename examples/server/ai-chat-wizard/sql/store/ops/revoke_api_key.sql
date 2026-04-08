UPDATE api_keys
SET revoked_at = ?
WHERE key_id = ?;
