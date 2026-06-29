INSERT INTO support_tickets (
    ticket_key,
    workspace_id,
    user_id,
    status,
    priority,
    subject,
    body,
    assignee_user_id,
    resolution_note,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(ticket_key) DO UPDATE SET
    workspace_id = excluded.workspace_id,
    user_id = excluded.user_id,
    status = excluded.status,
    priority = excluded.priority,
    subject = excluded.subject,
    body = excluded.body,
    assignee_user_id = excluded.assignee_user_id,
    resolution_note = excluded.resolution_note,
    updated_at = excluded.updated_at;
