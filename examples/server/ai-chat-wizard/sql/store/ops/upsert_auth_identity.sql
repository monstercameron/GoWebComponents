INSERT INTO auth_identities (
    user_id,
    provider_key,
    provider_type,
    provider_subject,
    email,
    is_email_verified,
    profile_json,
    last_login_at,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM users WHERE id = ?)
ON CONFLICT(provider_key, provider_subject) DO UPDATE SET
    user_id = excluded.user_id,
    provider_type = excluded.provider_type,
    email = excluded.email,
    is_email_verified = excluded.is_email_verified,
    profile_json = excluded.profile_json,
    last_login_at = excluded.last_login_at,
    updated_at = excluded.updated_at;
