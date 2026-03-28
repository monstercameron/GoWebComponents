SELECT
    u.id,
    u.email,
    COALESCE(NULLIF(TRIM(up.name), ''), 'User') AS display_name,
    u.created_at,
    (SELECT COUNT(*) FROM conversations c WHERE c.user_id = u.id) AS conversation_count,
    (SELECT COUNT(*)
     FROM messages m
     JOIN conversations c ON c.id = m.conversation_id
     WHERE c.user_id = u.id) AS message_count,
    (SELECT COUNT(*) FROM usage_events ue WHERE ue.user_id = u.id) AS usage_event_count,
    COALESCE((SELECT SUM(ue.total_cost_usd) FROM usage_events ue WHERE ue.user_id = u.id), 0) AS total_cost_usd,
    COALESCE((SELECT MAX(ue.created_at) FROM usage_events ue WHERE ue.user_id = u.id), '') AS last_seen_at,
    (SELECT COUNT(*) FROM workspace_memberships wm WHERE wm.user_id = u.id) AS workspace_count,
    (SELECT COUNT(*) FROM user_memory um WHERE um.user_id = u.id) AS memory_count,
    (SELECT COUNT(*)
     FROM support_tickets st
     WHERE st.user_id = u.id
       AND LOWER(TRIM(st.status)) NOT IN ('resolved', 'closed')) AS open_support_ticket_count,
    (SELECT COUNT(*)
     FROM billing_subscriptions bs
     JOIN billing_customers bc ON bc.id = bs.customer_id
     WHERE bc.user_id = u.id
       AND (TRIM(bs.status) = '' OR LOWER(TRIM(bs.status)) IN ('active', 'trialing', 'past_due'))) AS active_subscription_count,
    COALESCE((SELECT atv.token_version FROM auth_token_versions atv WHERE atv.user_id = u.id), 0) AS token_version,
    (SELECT COUNT(*)
     FROM support_ticket_messages stm
     JOIN support_tickets st ON st.id = stm.ticket_id
     WHERE st.user_id = u.id) AS support_message_count,
    (SELECT COUNT(*)
     FROM auth_sessions ases
     WHERE ases.user_id = u.id
       AND TRIM(COALESCE(ases.revoked_at, '')) = '') AS active_session_count
FROM users u
LEFT JOIN user_profile up ON up.user_id = u.id
ORDER BY u.id DESC
LIMIT ?;
