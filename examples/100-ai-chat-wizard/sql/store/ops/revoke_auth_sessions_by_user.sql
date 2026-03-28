UPDATE auth_sessions
SET
    revoked_at = ?,
    updated_at = ?
WHERE user_id = ?
  AND TRIM(COALESCE(revoked_at, '')) = '';
