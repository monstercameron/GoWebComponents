DELETE FROM billing_plan_entitlements
WHERE plan_code = ?
  AND entitlement_key = ?;
