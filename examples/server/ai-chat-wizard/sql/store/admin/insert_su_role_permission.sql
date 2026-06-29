INSERT INTO su_role_permissions (
    role_key,
    permission_key,
    permission_value,
    updated_at
)
VALUES (?, ?, ?, ?)
ON CONFLICT(role_key, permission_key) DO UPDATE SET
    permission_value = excluded.permission_value,
    updated_at = excluded.updated_at;
