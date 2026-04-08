SELECT
    (SELECT COUNT(*)
     FROM background_jobs bj
     WHERE bj.updated_at >= ?
       AND LOWER(TRIM(bj.status)) IN ('failed', 'error')
    ) AS failed_background_job_count,
    (SELECT COUNT(*)
     FROM notification_outbox no
     WHERE no.updated_at >= ?
       AND (
           LOWER(TRIM(no.status)) IN ('failed', 'error')
           OR TRIM(COALESCE(no.failed_at, '')) <> ''
       )
       AND (? <= 0 OR no.workspace_id = ?)
    ) AS failed_notification_count,
    (SELECT COUNT(*)
     FROM webhook_deliveries wd
     JOIN webhook_endpoints we ON we.id = wd.endpoint_id
     WHERE wd.updated_at >= ?
       AND TRIM(COALESCE(wd.failed_at, '')) <> ''
       AND TRIM(COALESCE(wd.delivered_at, '')) = ''
       AND (? <= 0 OR we.workspace_id = ?)
    ) AS failed_webhook_delivery_count,
    (SELECT COUNT(*)
     FROM incidents i
     WHERE LOWER(TRIM(i.status)) = 'open'
    ) AS open_incident_count,
    (SELECT COUNT(*)
     FROM audit_logs al
     WHERE al.created_at >= ?
       AND (
           LOWER(TRIM(al.event_type)) LIKE 'admin.ops.%'
           OR LOWER(TRIM(al.event_type)) = 'admin.incident.updated'
       )
       AND (? <= 0 OR al.workspace_id = ?)
    ) AS recent_admin_action_count;
