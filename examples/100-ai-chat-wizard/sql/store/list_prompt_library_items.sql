SELECT id, workspace_id, user_id, item_key, title, category, prompt_text, tags_json, is_public, use_count, created_at, updated_at
FROM prompt_library_items
ORDER BY id DESC
LIMIT ?;
