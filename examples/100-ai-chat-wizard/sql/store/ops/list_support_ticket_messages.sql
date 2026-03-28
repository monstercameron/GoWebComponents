SELECT id, ticket_id, author_user_id, message_type, body, is_internal, created_at, updated_at
FROM support_ticket_messages
ORDER BY id DESC
LIMIT ?;
