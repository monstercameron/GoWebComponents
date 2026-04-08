SELECT role_key, label, description, is_system, is_enabled, created_at, updated_at
FROM su_roles
ORDER BY is_system DESC, role_key ASC;
