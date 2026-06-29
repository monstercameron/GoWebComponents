INSERT INTO su_user_roles (
    user_id,
    role_key,
    assigned_by_user_id,
    created_at
)
SELECT ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM users WHERE id = ?)
  AND EXISTS (SELECT 1 FROM su_roles WHERE role_key = ?)
ON CONFLICT(user_id, role_key) DO UPDATE SET
    assigned_by_user_id = excluded.assigned_by_user_id;
