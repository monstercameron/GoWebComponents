SELECT
    id,
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
FROM auth_identities
WHERE user_id = ?
ORDER BY id DESC;
