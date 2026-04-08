SELECT id, customer_id, subscription_id, workspace_id, reason_key, detail, recovery_offer_key, created_at
FROM subscription_churn_feedback
WHERE customer_id = ?
ORDER BY id DESC
LIMIT ?;
