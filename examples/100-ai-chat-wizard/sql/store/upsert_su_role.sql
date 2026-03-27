INSERT INTO su_roles (
    role_key,
    label,
    description,
    is_system,
    is_enabled,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(role_key) DO UPDATE SET
    label = excluded.label,
    description = excluded.description,
    is_system = excluded.is_system,
    is_enabled = excluded.is_enabled,
    updated_at = excluded.updated_at;
