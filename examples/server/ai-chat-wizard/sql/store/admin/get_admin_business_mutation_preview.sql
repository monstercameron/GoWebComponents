SELECT
    (SELECT COUNT(*)
     FROM billing_customers bc
     WHERE (? <= 0 OR bc.user_id = ?)
    ) AS customer_count,
    (SELECT COUNT(*)
     FROM billing_subscriptions bs
     JOIN billing_customers bc ON bc.id = bs.customer_id
     WHERE (? <= 0 OR bc.user_id = ?)
    ) AS subscription_count,
    (SELECT COUNT(*)
     FROM billing_invoices bi
     JOIN billing_customers bc ON bc.id = bi.customer_id
     WHERE (? <= 0 OR bc.user_id = ?)
       AND LOWER(TRIM(bi.status)) IN ('open', 'draft', 'past_due')
    ) AS open_invoice_count,
    (SELECT COUNT(*)
     FROM billing_dunning_events bde
     JOIN billing_customers bc ON bc.id = bde.customer_id
     WHERE (? <= 0 OR bc.user_id = ?)
       AND LOWER(TRIM(bde.status)) NOT IN ('resolved', 'closed')
    ) AS open_dunning_event_count,
    (SELECT COUNT(*)
     FROM billing_events be
     JOIN billing_customers bc ON bc.id = be.customer_id
     WHERE (? <= 0 OR bc.user_id = ?)
       AND be.created_at >= ?
       AND LOWER(TRIM(be.event_type)) IN ('invoice.payment_failed', 'payment_failed')
    ) AS recent_failed_payment_event_count,
    (SELECT COUNT(*)
     FROM usage_events ue
     WHERE (? <= 0 OR ue.user_id = ?)
       AND ue.created_at >= ?
    ) AS recent_usage_event_count,
    (SELECT COALESCE(SUM(ue.total_cost_usd), 0)
     FROM usage_events ue
     WHERE (? <= 0 OR ue.user_id = ?)
       AND ue.created_at >= ?
    ) AS recent_usage_cost_usd;
