SELECT m.role, m.content, m.model_id, m.prompt_tokens, m.completion_tokens
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
WHERE m.conversation_id = ? AND c.user_id = ?
ORDER BY m.id
