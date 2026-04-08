SELECT id, workspace_id, user_id, notification_key, channel_key, template_key, status, subject, body_text, payload_json, dedupe_key, scheduled_at, sent_at, failed_at, error_message, created_at, updated_at
FROM notification_outbox
WHERE status = 'pending'
  AND (TRIM(COALESCE(scheduled_at, '')) = '' OR scheduled_at <= ?)
ORDER BY id ASC
LIMIT ?;
