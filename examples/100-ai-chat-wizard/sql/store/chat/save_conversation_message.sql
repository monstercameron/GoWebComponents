INSERT INTO messages (conversation_id, role, content, model_id, prompt_tokens, completion_tokens, saved_at)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS(SELECT 1 FROM conversations WHERE id = ? AND user_id = ?)
