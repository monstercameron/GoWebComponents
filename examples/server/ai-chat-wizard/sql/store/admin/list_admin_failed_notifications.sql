SELECT
    id,
    workspace_id,
    user_id,
    notification_key,
    channel_key,
    template_key,
    status,
    subject,
    body_text,
    payload_json,
    dedupe_key,
    scheduled_at,
    sent_at,
    failed_at,
    error_message,
    created_at,
    updated_at
FROM notification_outbox
WHERE updated_at >= ?
  AND (
      LOWER(TRIM(status)) IN ('failed', 'error')
      OR TRIM(COALESCE(failed_at, '')) <> ''
  )
ORDER BY updated_at DESC, id DESC
LIMIT ?;
