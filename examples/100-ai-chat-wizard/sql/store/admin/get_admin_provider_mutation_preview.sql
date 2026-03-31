SELECT
    (SELECT COUNT(*)
     FROM workspaces w
     WHERE (? <= 0 OR w.id = ?)
    ) AS workspace_count,
    (SELECT COUNT(*)
     FROM workspace_model_routing_policies wmrp
     WHERE (? <= 0 OR wmrp.workspace_id = ?)
       AND (? = '' OR LOWER(TRIM(wmrp.default_model_id)) = LOWER(TRIM(?)) OR LOWER(TRIM(wmrp.fallback_model_id)) = LOWER(TRIM(?)))
    ) AS routing_policy_count,
    (SELECT COUNT(*)
     FROM workspace_cost_guardrails wcg
     WHERE (? <= 0 OR wcg.workspace_id = ?)
    ) AS guardrail_count,
    (SELECT COUNT(*)
     FROM usage_events ue
     WHERE ue.created_at >= ?
       AND (? = '' OR LOWER(TRIM(ue.provider_id)) = LOWER(TRIM(?)))
       AND (? = '' OR LOWER(TRIM(ue.model_id)) = LOWER(TRIM(?)))
    ) AS recent_usage_event_count,
    (SELECT COUNT(*)
     FROM usage_events ue
     WHERE ue.created_at >= ?
       AND LOWER(TRIM(ue.status)) = 'failed'
       AND (? = '' OR LOWER(TRIM(ue.provider_id)) = LOWER(TRIM(?)))
       AND (? = '' OR LOWER(TRIM(ue.model_id)) = LOWER(TRIM(?)))
    ) AS recent_failed_usage_event_count,
    (SELECT COALESCE(SUM(ue.total_cost_usd), 0)
     FROM usage_events ue
     WHERE ue.created_at >= ?
       AND (? = '' OR LOWER(TRIM(ue.provider_id)) = LOWER(TRIM(?)))
       AND (? = '' OR LOWER(TRIM(ue.model_id)) = LOWER(TRIM(?)))
    ) AS recent_usage_cost_usd;
