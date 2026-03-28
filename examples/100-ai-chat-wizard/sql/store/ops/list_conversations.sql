SELECT c.id, c.public_id, c.started_at,
       COALESCE(
           c.title,
           (SELECT content FROM messages
            WHERE conversation_id = c.id AND role = 'user'
            ORDER BY id LIMIT 1),
           ''
       ) AS preview
FROM conversations c
WHERE c.user_id = ?
ORDER BY c.id DESC
