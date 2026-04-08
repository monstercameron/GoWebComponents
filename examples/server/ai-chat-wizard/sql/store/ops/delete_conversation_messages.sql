DELETE FROM messages WHERE conversation_id IN (SELECT id FROM conversations WHERE id = ? AND user_id = ?)
