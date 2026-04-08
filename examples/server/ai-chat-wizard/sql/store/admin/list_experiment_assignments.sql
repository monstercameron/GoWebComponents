SELECT id, experiment_key, workspace_id, user_id, variant_key, assigned_at
FROM experiment_assignments
ORDER BY id DESC
LIMIT ?;
