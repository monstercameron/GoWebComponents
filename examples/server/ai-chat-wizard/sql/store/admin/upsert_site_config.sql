INSERT INTO site_config (
    config_key,
    config_value,
    value_type,
    description,
    updated_by_user_id,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(config_key) DO UPDATE SET
    config_value = excluded.config_value,
    value_type = excluded.value_type,
    description = excluded.description,
    updated_by_user_id = excluded.updated_by_user_id,
    updated_at = excluded.updated_at;
