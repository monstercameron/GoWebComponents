SELECT
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(*) FROM conversations) AS total_conversations,
    (SELECT COUNT(*) FROM messages) AS total_messages,
    (SELECT COUNT(*) FROM users WHERE created_at >= ?) AS window_new_users,
    (SELECT COUNT(*) FROM conversations WHERE started_at >= ?) AS window_new_conversations,
    (SELECT COUNT(*) FROM messages WHERE saved_at >= ?) AS window_new_messages,
    (SELECT COUNT(*) FROM usage_events WHERE created_at >= ?) AS window_usage_events,
    COALESCE((SELECT SUM(total_cost_usd) FROM usage_events WHERE created_at >= ?), 0) AS window_total_cost_usd,
    COALESCE((SELECT SUM(prompt_tokens) FROM usage_events WHERE created_at >= ?), 0) AS window_prompt_tokens,
    COALESCE((SELECT SUM(completion_tokens) FROM usage_events WHERE created_at >= ?), 0) AS window_completion_tokens,
    (SELECT COUNT(DISTINCT user_id) FROM usage_events WHERE created_at >= ?) AS window_active_users,
    (SELECT COUNT(DISTINCT conversation_id) FROM usage_events WHERE created_at >= ?) AS window_active_conversations,
    (SELECT COUNT(DISTINCT client_id)
     FROM usage_events
     WHERE created_at >= ?
       AND TRIM(COALESCE(client_id, '')) <> '') AS window_active_clients,
    (SELECT COUNT(*)
     FROM usage_events
     WHERE created_at >= ?
       AND status = 'completed') AS window_completed_events,
    (SELECT COUNT(*)
     FROM usage_events
     WHERE created_at >= ?
       AND status <> 'completed') AS window_failed_events,
    (SELECT COUNT(*)
     FROM billing_events
     WHERE created_at >= ?) AS window_billing_events,
    (SELECT COUNT(*)
     FROM billing_invoices
     WHERE created_at >= ?
       AND LOWER(status) NOT IN ('paid', 'void')) AS window_open_invoices,
    (SELECT COUNT(*)
     FROM billing_dunning_events
     WHERE created_at >= ?
       AND LOWER(status) NOT IN ('resolved', 'closed', 'succeeded')) AS window_open_dunning_events,
    (SELECT COUNT(*)
     FROM incidents
     WHERE LOWER(status) NOT IN ('resolved', 'closed')) AS open_incidents,
    (SELECT COUNT(*)
     FROM support_tickets
     WHERE LOWER(status) NOT IN ('resolved', 'closed')) AS open_support_tickets,
    (SELECT COUNT(*)
     FROM experiments
     WHERE LOWER(status) IN ('active', 'running', 'live')) AS active_experiments,
    (SELECT COUNT(*)
     FROM experiments
     WHERE LOWER(status) IN ('paused', 'error', 'failed', 'rolled_back', 'stopped')) AS unhealthy_experiments,
    (SELECT COUNT(*)
     FROM experiment_assignments
     WHERE assigned_at >= ?) AS window_experiment_assignments;
