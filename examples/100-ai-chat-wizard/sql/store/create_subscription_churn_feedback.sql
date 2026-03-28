INSERT INTO subscription_churn_feedback (
    customer_id,
    subscription_id,
    workspace_id,
    reason_key,
    detail,
    recovery_offer_key,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM billing_customers WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM billing_subscriptions WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?));
