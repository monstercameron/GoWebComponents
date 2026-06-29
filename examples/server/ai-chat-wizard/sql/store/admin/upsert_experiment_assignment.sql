INSERT INTO experiment_assignments (
    experiment_key,
    workspace_id,
    user_id,
    variant_key,
    assigned_at
)
SELECT ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM experiments WHERE experiment_key = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(experiment_key, workspace_id, user_id) DO UPDATE SET
    variant_key = excluded.variant_key,
    assigned_at = excluded.assigned_at;
