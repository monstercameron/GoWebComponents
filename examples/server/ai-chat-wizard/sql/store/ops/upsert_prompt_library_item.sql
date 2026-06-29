INSERT INTO prompt_library_items (
    workspace_id,
    user_id,
    item_key,
    title,
    category,
    prompt_text,
    tags_json,
    is_public,
    use_count,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(workspace_id, item_key) DO UPDATE SET
    user_id = excluded.user_id,
    title = excluded.title,
    category = excluded.category,
    prompt_text = excluded.prompt_text,
    tags_json = excluded.tags_json,
    is_public = excluded.is_public,
    use_count = excluded.use_count,
    updated_at = excluded.updated_at;
