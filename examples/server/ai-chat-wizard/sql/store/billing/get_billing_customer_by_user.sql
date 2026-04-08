SELECT
    id,
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
FROM billing_customers
WHERE user_id = ?;
