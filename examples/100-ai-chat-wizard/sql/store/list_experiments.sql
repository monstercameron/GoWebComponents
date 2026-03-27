SELECT id, experiment_key, name, status, variants_json, audience_json, start_at, end_at, updated_at
FROM experiments
ORDER BY id DESC
LIMIT ?;
