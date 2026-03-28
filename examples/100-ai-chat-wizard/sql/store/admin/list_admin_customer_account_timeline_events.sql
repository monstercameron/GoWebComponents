SELECT
    timeline.timeline_id,
    timeline.source,
    timeline.event_type,
    timeline.user_id,
    timeline.workspace_id,
    timeline.conversation_id,
    timeline.support_ticket_id,
    timeline.billing_customer_id,
    timeline.auth_session_id,
    timeline.audit_log_id,
    timeline.summary,
    timeline.detail_json,
    timeline.created_at
FROM (
    SELECT
        printf('chat:%d', c.id) AS timeline_id,
        'chat_content' AS source,
        'conversation_started' AS event_type,
        c.user_id AS user_id,
        0 AS workspace_id,
        c.id AS conversation_id,
        0 AS support_ticket_id,
        0 AS billing_customer_id,
        0 AS auth_session_id,
        0 AS audit_log_id,
        COALESCE(
            NULLIF(TRIM(c.title), ''),
            (SELECT content
             FROM messages m
             WHERE m.conversation_id = c.id
               AND LOWER(TRIM(m.role)) = 'user'
             ORDER BY m.id ASC
             LIMIT 1),
            'Conversation started'
        ) AS summary,
        '{}' AS detail_json,
        c.started_at AS created_at
    FROM conversations c
    WHERE c.user_id = ?
      AND c.started_at >= ?

    UNION ALL

    SELECT
        printf('support_ticket:%d', st.id) AS timeline_id,
        'support_note' AS source,
        'support_ticket_updated' AS event_type,
        st.user_id AS user_id,
        st.workspace_id AS workspace_id,
        0 AS conversation_id,
        st.id AS support_ticket_id,
        0 AS billing_customer_id,
        0 AS auth_session_id,
        0 AS audit_log_id,
        COALESCE(NULLIF(TRIM(st.subject), ''), 'Support ticket updated') AS summary,
        '{}' AS detail_json,
        st.updated_at AS created_at
    FROM support_tickets st
    WHERE st.user_id = ?
      AND st.updated_at >= ?

    UNION ALL

    SELECT
        printf('support_message:%d', stm.id) AS timeline_id,
        'support_note' AS source,
        'support_message_added' AS event_type,
        st.user_id AS user_id,
        st.workspace_id AS workspace_id,
        0 AS conversation_id,
        st.id AS support_ticket_id,
        0 AS billing_customer_id,
        0 AS auth_session_id,
        0 AS audit_log_id,
        COALESCE(NULLIF(TRIM(stm.message_type), ''), 'Support message added') AS summary,
        '{}' AS detail_json,
        stm.created_at AS created_at
    FROM support_ticket_messages stm
    JOIN support_tickets st ON st.id = stm.ticket_id
    WHERE st.user_id = ?
      AND stm.created_at >= ?

    UNION ALL

    SELECT
        printf('billing:%d', be.id) AS timeline_id,
        'billing_record' AS source,
        COALESCE(NULLIF(TRIM(be.event_type), ''), 'billing_event') AS event_type,
        bc.user_id AS user_id,
        0 AS workspace_id,
        0 AS conversation_id,
        0 AS support_ticket_id,
        bc.id AS billing_customer_id,
        0 AS auth_session_id,
        0 AS audit_log_id,
        COALESCE(NULLIF(TRIM(be.event_summary), ''), 'Billing event') AS summary,
        '{}' AS detail_json,
        be.created_at AS created_at
    FROM billing_events be
    JOIN billing_customers bc ON bc.id = be.customer_id
    WHERE bc.user_id = ?
      AND be.created_at >= ?

    UNION ALL

    SELECT
        printf('auth_session:%d', ases.id) AS timeline_id,
        'auth_session' AS source,
        CASE
            WHEN TRIM(COALESCE(ases.revoked_at, '')) <> '' THEN 'session_revoked'
            ELSE 'session_issued'
        END AS event_type,
        ases.user_id AS user_id,
        0 AS workspace_id,
        0 AS conversation_id,
        0 AS support_ticket_id,
        0 AS billing_customer_id,
        ases.id AS auth_session_id,
        0 AS audit_log_id,
        CASE
            WHEN TRIM(COALESCE(ases.revoked_at, '')) <> '' THEN 'Session revoked'
            ELSE 'Session issued'
        END AS summary,
        '{}' AS detail_json,
        ases.created_at AS created_at
    FROM auth_sessions ases
    WHERE ases.user_id = ?
      AND ases.created_at >= ?

    UNION ALL

    SELECT
        printf('audit:%d', al.id) AS timeline_id,
        'audit_event' AS source,
        COALESCE(NULLIF(TRIM(al.event_type), ''), 'audit_event') AS event_type,
        ? AS user_id,
        al.workspace_id AS workspace_id,
        0 AS conversation_id,
        0 AS support_ticket_id,
        0 AS billing_customer_id,
        0 AS auth_session_id,
        al.id AS audit_log_id,
        COALESCE(NULLIF(TRIM(al.summary), ''), 'Audit event') AS summary,
        '{}' AS detail_json,
        al.created_at AS created_at
    FROM audit_logs al
    WHERE (
            al.actor_user_id = ?
            OR (
                LOWER(TRIM(al.target_type)) = 'user'
                AND TRIM(al.target_id) = CAST(? AS TEXT)
            )
        )
      AND al.created_at >= ?
) AS timeline
ORDER BY timeline.created_at DESC, timeline.timeline_id DESC
LIMIT ?;
