INSERT INTO support_ticket_messages (
    ticket_id,
    author_user_id,
    message_type,
    body,
    is_internal,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM support_tickets WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?));
