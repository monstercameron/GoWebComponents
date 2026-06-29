INSERT INTO experiments (
    experiment_key,
    name,
    status,
    variants_json,
    audience_json,
    start_at,
    end_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(experiment_key) DO UPDATE SET
    name = excluded.name,
    status = excluded.status,
    variants_json = excluded.variants_json,
    audience_json = excluded.audience_json,
    start_at = excluded.start_at,
    end_at = excluded.end_at,
    updated_at = excluded.updated_at;
