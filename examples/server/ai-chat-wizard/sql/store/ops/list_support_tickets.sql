SELECT id, ticket_key, workspace_id, user_id, status, priority, subject, body, assignee_user_id, resolution_note, created_at, updated_at
FROM support_tickets
ORDER BY id DESC
LIMIT ?;
