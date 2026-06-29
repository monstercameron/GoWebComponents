INSERT INTO billing_customers (
    user_id,
    provider_id,
    provider_customer_id,
    billing_email,
    billing_name,
    billing_country,
    billing_region,
    default_currency,
    tax_exempt_status,
    external_metadata_json,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM users WHERE id = ?)
ON CONFLICT(user_id) DO UPDATE SET
    provider_id = excluded.provider_id,
    provider_customer_id = excluded.provider_customer_id,
    billing_email = excluded.billing_email,
    billing_name = excluded.billing_name,
    billing_country = excluded.billing_country,
    billing_region = excluded.billing_region,
    default_currency = excluded.default_currency,
    tax_exempt_status = excluded.tax_exempt_status,
    external_metadata_json = excluded.external_metadata_json,
    updated_at = excluded.updated_at;
