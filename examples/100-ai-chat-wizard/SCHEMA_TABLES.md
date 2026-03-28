# Example 100 Schema Tables

This document summarizes the tables defined in `sql/store/schema.sql`.

## Auth and Identity

### `users`
Purpose: Primary account record for login-backed users.
Columns: `id`, `email`, `password_hash`, `created_at`

### `auth_sessions`
Purpose: Durable server-side session ledger for refresh, revocation, device tracking, and expiry.
Columns: `id`, `user_id`, `session_id`, `token_version`, `refresh_token_hash`, `user_agent`, `ip_address`, `last_seen_at`, `expires_at`, `revoked_at`, `created_at`, `updated_at`

### `auth_token_versions`
Purpose: Per-user token generation counter for global session invalidation.
Columns: `user_id`, `token_version`, `updated_at`

### `password_reset_tokens`
Purpose: One-time password reset token storage with expiry and consumption state.
Columns: `id`, `user_id`, `email`, `token_hash`, `status`, `requested_by_ip`, `expires_at`, `consumed_at`, `created_at`

### `email_verification_tokens`
Purpose: One-time email verification token storage with expiry and verification state.
Columns: `id`, `user_id`, `email`, `token_hash`, `status`, `expires_at`, `verified_at`, `created_at`

### `user_profile`
Purpose: Per-user product preferences and saved model/tone settings.
Columns: `user_id`, `name`, `updated_at`, `selected_model`, `selected_tone`, `selected_thinking_enabled`, `selected_thinking_effort`, `selected_system_prompt`

### `user_memory`
Purpose: Long-lived structured memory items extracted from chat history.
Columns: `id`, `user_id`, `memory_key`, `category`, `summary`, `detail`, `source_message`, `usefulness_score`, `confidence_score`, `rubric_reason`, `created_at`, `updated_at`

## Chat and Usage

### `conversations`
Purpose: Conversation threads owned by a user.
Columns: `id`, `user_id`, `public_id`, `started_at`, `title`

### `messages`
Purpose: Stored conversation messages and model token counts.
Columns: `id`, `conversation_id`, `role`, `content`, `model_id`, `prompt_tokens`, `completion_tokens`, `saved_at`

### `usage_events`
Purpose: Immutable metering ledger for provider/model usage, tracing, and cost.
Columns: `id`, `event_id`, `user_id`, `conversation_id`, `provider_id`, `model_id`, `prompt_tokens`, `completion_tokens`, `usage_source`, `provider_request_id`, `input_cost_per_million_usd`, `output_cost_per_million_usd`, `pricing_currency`, `input_cost_usd`, `output_cost_usd`, `total_cost_usd`, `client_id`, `trace_id`, `span_id`, `trace_state`, `status`, `error_message`, `created_at`

## Billing and Pricing

### `billing_plans`
Purpose: Plan catalog with packaging, base price, seat, and capability flags.
Columns: `plan_code`, `plan_name`, `plan_rank`, `is_active`, `monthly_base_cents`, `yearly_base_cents`, `included_tokens_monthly`, `included_seats`, `max_seats`, `supports_priority`, `supports_team_workspace`, `supports_sso`, `created_at`, `updated_at`

### `billing_plan_entitlements`
Purpose: Key/value entitlement map attached to each billing plan.
Columns: `plan_code`, `entitlement_key`, `entitlement_value`, `updated_at`

### `billing_plan_model_access`
Purpose: Model allowlist and defaults by billing plan.
Columns: `plan_code`, `model_id`, `is_default`, `is_enabled`, `updated_at`

### `billing_customers`
Purpose: Billing-system customer identity linked to one app user.
Columns: `id`, `user_id`, `provider_id`, `provider_customer_id`, `billing_email`, `billing_name`, `billing_country`, `billing_region`, `default_currency`, `tax_exempt_status`, `external_metadata_json`, `created_at`, `updated_at`

### `billing_subscriptions`
Purpose: Subscription state, plan assignment, renewal window, and cancellation status.
Columns: `id`, `customer_id`, `provider_id`, `provider_subscription_id`, `plan_code`, `price_code`, `status`, `billing_interval`, `quantity`, `current_period_start`, `current_period_end`, `cancel_at_period_end`, `canceled_at`, `trial_ends_at`, `access_expires_at`, `created_at`, `updated_at`

### `billing_invoices`
Purpose: Invoice ledger with totals, due dates, payment state, and hosted invoice URL.
Columns: `id`, `customer_id`, `subscription_id`, `provider_id`, `provider_invoice_id`, `status`, `currency`, `subtotal_cents`, `tax_cents`, `discount_cents`, `total_cents`, `amount_due_cents`, `amount_paid_cents`, `period_start`, `period_end`, `due_at`, `paid_at`, `hosted_invoice_url`, `external_metadata_json`, `created_at`, `updated_at`

### `billing_invoice_line_items`
Purpose: Invoice rows for usage, seats, overages, or fixed charges.
Columns: `id`, `invoice_id`, `usage_event_id`, `line_type`, `description`, `quantity`, `unit_amount_cents`, `amount_cents`, `currency`, `period_start`, `period_end`, `created_at`

### `billing_access_overrides`
Purpose: Per-customer manual entitlement overrides outside the base plan.
Columns: `id`, `customer_id`, `override_key`, `override_value`, `reason`, `is_enabled`, `starts_at`, `ends_at`, `actor_user_id`, `created_at`, `updated_at`

### `billing_events`
Purpose: Immutable billing audit/event stream for customer, subscription, and invoice changes.
Columns: `id`, `customer_id`, `subscription_id`, `invoice_id`, `event_type`, `event_source`, `event_summary`, `event_payload_json`, `actor_user_id`, `created_at`

### `billing_plan_overages`
Purpose: Overage pricing rules by meter for each plan.
Columns: `id`, `plan_code`, `meter_key`, `included_units`, `soft_limit_units`, `hard_limit_units`, `overage_unit_size`, `overage_price_cents`, `billing_interval`, `updated_at`

### `billing_quota_policies`
Purpose: Enforceable quota policies and reset/enforcement behavior per plan.
Columns: `id`, `plan_code`, `quota_key`, `soft_limit_value`, `hard_limit_value`, `reset_interval`, `enforcement_mode`, `updated_at`

### `billing_upgrade_triggers`
Purpose: In-product upgrade thresholds, CTA copy, and destination plan mapping.
Columns: `id`, `plan_code`, `trigger_key`, `threshold_percent`, `upgrade_plan_code`, `message`, `cta_label`, `cta_url`, `is_enabled`, `updated_at`

### `billing_dunning_events`
Purpose: Failed-payment recovery tracking with retry scheduling and resolution state.
Columns: `id`, `customer_id`, `subscription_id`, `invoice_id`, `status`, `attempt_count`, `failure_reason`, `next_attempt_at`, `resolved_at`, `created_at`, `updated_at`

### `subscription_churn_feedback`
Purpose: Captured cancellation reasons and win-back offer metadata.
Columns: `id`, `customer_id`, `subscription_id`, `workspace_id`, `reason_key`, `detail`, `recovery_offer_key`, `created_at`

## Control Plane and Admin

### `su_roles`
Purpose: Superuser/admin/customer role definitions.
Columns: `role_key`, `label`, `description`, `is_system`, `is_enabled`, `created_at`, `updated_at`

### `su_role_permissions`
Purpose: Permission key/value map for each superuser role.
Columns: `role_key`, `permission_key`, `permission_value`, `updated_at`

### `su_user_roles`
Purpose: User-to-role assignments for superuser control-plane access.
Columns: `id`, `user_id`, `role_key`, `assigned_by_user_id`, `created_at`

### `site_config`
Purpose: Global site-wide key/value configuration.
Columns: `config_key`, `config_value`, `value_type`, `description`, `updated_by_user_id`, `updated_at`

### `feature_flags`
Purpose: Feature rollout definitions with audience and payload metadata.
Columns: `flag_key`, `description`, `is_enabled`, `rollout_percent`, `audience_json`, `payload_json`, `updated_by_user_id`, `updated_at`

### `workspaces`
Purpose: Team/account container for plan, ownership, and workspace settings.
Columns: `id`, `workspace_key`, `slug`, `name`, `plan_code`, `status`, `owner_user_id`, `settings_json`, `created_at`, `updated_at`

### `workspace_memberships`
Purpose: User membership rows for workspace roles and status.
Columns: `id`, `workspace_id`, `user_id`, `role_key`, `status`, `invited_by_user_id`, `created_at`, `updated_at`

### `workspace_invitations`
Purpose: Pending or accepted workspace invites addressed to email recipients.
Columns: `id`, `workspace_id`, `email`, `role_key`, `invitation_token_hash`, `invited_by_user_id`, `status`, `expires_at`, `accepted_at`, `created_at`, `updated_at`

### `api_keys`
Purpose: Customer or workspace-scoped API credential records.
Columns: `id`, `key_id`, `workspace_id`, `user_id`, `label`, `key_prefix`, `secret_hash`, `scopes_json`, `last_used_at`, `revoked_at`, `created_at`

### `webhook_endpoints`
Purpose: Registered webhook destinations and subscription settings.
Columns: `id`, `workspace_id`, `label`, `target_url`, `secret_hash`, `events_json`, `is_enabled`, `last_delivery_at`, `failure_count`, `created_at`, `updated_at`

### `webhook_deliveries`
Purpose: Delivery attempt log for webhook payloads, responses, and retries.
Columns: `id`, `endpoint_id`, `event_type`, `delivery_key`, `request_headers_json`, `request_body_json`, `response_status`, `response_body`, `attempt_count`, `delivered_at`, `failed_at`, `next_retry_at`, `created_at`, `updated_at`

### `audit_logs`
Purpose: Immutable audit trail for administrative and workspace actions.
Columns: `id`, `actor_user_id`, `workspace_id`, `event_type`, `target_type`, `target_id`, `summary`, `payload_json`, `created_at`

### `support_tickets`
Purpose: Top-level support cases tied to a user and workspace.
Columns: `id`, `ticket_key`, `workspace_id`, `user_id`, `status`, `priority`, `subject`, `body`, `assignee_user_id`, `resolution_note`, `created_at`, `updated_at`

### `support_ticket_messages`
Purpose: Threaded replies and internal notes attached to support tickets.
Columns: `id`, `ticket_id`, `author_user_id`, `message_type`, `body`, `is_internal`, `created_at`, `updated_at`

### `notification_outbox`
Purpose: Durable outbound notification queue for email or in-product delivery.
Columns: `id`, `workspace_id`, `user_id`, `notification_key`, `channel_key`, `template_key`, `status`, `subject`, `body_text`, `payload_json`, `dedupe_key`, `scheduled_at`, `sent_at`, `failed_at`, `error_message`, `created_at`, `updated_at`

### `background_jobs`
Purpose: Scheduled or retryable background job queue for async operational work.
Columns: `id`, `job_key`, `job_type`, `queue_key`, `status`, `attempt_count`, `max_attempts`, `payload_json`, `run_after`, `started_at`, `finished_at`, `error_message`, `created_at`, `updated_at`

## Product Growth and Collaboration

### `experiments`
Purpose: A/B test definitions and audience/variant config.
Columns: `id`, `experiment_key`, `name`, `status`, `variants_json`, `audience_json`, `start_at`, `end_at`, `updated_at`

### `onboarding_templates`
Purpose: Starter templates and guided onboarding prompt content.
Columns: `id`, `template_key`, `title`, `category`, `prompt_text`, `checklist_json`, `is_default`, `sort_order`, `updated_at`

### `user_activation_milestones`
Purpose: First-value and onboarding milestone completion tracking per user.
Columns: `id`, `user_id`, `milestone_key`, `status`, `achieved_at`, `metadata_json`, `updated_at`

### `saved_workflows`
Purpose: Reusable saved workflows owned by a user or workspace.
Columns: `id`, `workspace_id`, `user_id`, `workflow_key`, `name`, `description`, `workflow_json`, `is_public`, `created_at`, `updated_at`

### `prompt_library_items`
Purpose: Shared or private prompt library entries with tags and usage counts.
Columns: `id`, `workspace_id`, `user_id`, `item_key`, `title`, `category`, `prompt_text`, `tags_json`, `is_public`, `use_count`, `created_at`, `updated_at`

### `workspace_asset_shares`
Purpose: Share records for cross-user or cross-workspace access to saved assets.
Columns: `id`, `workspace_id`, `asset_type`, `asset_id`, `target_workspace_id`, `target_user_id`, `shared_by_user_id`, `permission_key`, `created_at`

### `weekly_value_summaries`
Purpose: Weekly user/workspace value recap history and send timestamps.
Columns: `id`, `workspace_id`, `user_id`, `summary_week`, `summary_text`, `metrics_json`, `sent_at`, `created_at`

### `product_analytics_events`
Purpose: Product funnel and experiment instrumentation events.
Columns: `id`, `workspace_id`, `user_id`, `session_key`, `event_name`, `funnel_key`, `step_key`, `experiment_key`, `variant_key`, `event_props_json`, `created_at`

### `experiment_assignments`
Purpose: Persisted user/workspace assignment to one experiment variant.
Columns: `id`, `experiment_key`, `workspace_id`, `user_id`, `variant_key`, `assigned_at`

### `account_health_scores`
Purpose: Derived customer health snapshots for success or churn-risk workflows.
Columns: `id`, `workspace_id`, `score_value`, `risk_level`, `signals_json`, `owner_user_id`, `scored_at`

### `customer_success_playbooks`
Purpose: Reusable success motions segmented by customer type or lifecycle stage.
Columns: `id`, `playbook_key`, `title`, `segment_key`, `steps_json`, `is_active`, `updated_at`

### `integration_connections`
Purpose: Workspace-level external integration connections and configuration.
Columns: `id`, `workspace_id`, `provider_key`, `external_account_id`, `status`, `scopes_json`, `config_json`, `connected_at`, `updated_at`

### `partner_referrals`
Purpose: Referral or partner-attributed workspace/customer acquisition records.
Columns: `id`, `partner_key`, `workspace_id`, `user_id`, `referral_code`, `status`, `revenue_share_bps`, `notes`, `created_at`, `updated_at`

## Enterprise, Compliance, and Reliability

### `workspace_sso_configs`
Purpose: Per-workspace SSO/SAML configuration and domain mapping.
Columns: `id`, `workspace_id`, `provider_key`, `saml_entrypoint`, `saml_issuer`, `saml_certificate_pem`, `domains_json`, `is_enabled`, `updated_at`

### `data_retention_policies`
Purpose: Retention and purge rules for workspace data scopes.
Columns: `id`, `workspace_id`, `scope_key`, `retention_days`, `purge_mode`, `legal_hold_json`, `updated_at`

### `compliance_controls`
Purpose: Workspace compliance control tracking, ownership, and evidence links.
Columns: `id`, `workspace_id`, `control_key`, `framework_key`, `status`, `owner_user_id`, `evidence_url`, `reviewed_at`, `updated_at`

### `service_level_objectives`
Purpose: SLO targets, error budgets, and status page linkage for major services.
Columns: `id`, `slo_key`, `service_name`, `objective_percent`, `window_days`, `error_budget_minutes`, `status_page_url`, `updated_at`

### `incidents`
Purpose: Incident records tied to service objectives and postmortem links.
Columns: `id`, `incident_key`, `slo_key`, `severity`, `status`, `title`, `summary`, `started_at`, `resolved_at`, `postmortem_url`, `updated_at`

### `incident_updates`
Purpose: Timeline/status updates published against an incident.
Columns: `id`, `incident_id`, `status`, `message`, `is_public`, `published_at`, `created_by_user_id`, `created_at`

### `workspace_model_routing_policies`
Purpose: Per-workspace model routing, fallback, approval, and cost rules.
Columns: `id`, `workspace_id`, `policy_key`, `default_model_id`, `fallback_model_id`, `max_input_cost_per_million_usd`, `max_output_cost_per_million_usd`, `requires_approval`, `rules_json`, `updated_at`

### `workspace_cost_guardrails`
Purpose: Per-workspace budget and per-request cost protection settings.
Columns: `id`, `workspace_id`, `guardrail_key`, `daily_budget_cents`, `monthly_budget_cents`, `max_cost_per_request_cents`, `alert_threshold_percent`, `action_mode`, `updated_at`

## Model Catalog

### `model_catalog`
Purpose: Available model metadata, pricing, capability, and product role flags.
Columns: `id`, `provider_id`, `provider_label`, `label`, `note`, `description`, `supports_thinking`, `supports_speech`, `input_cost_per_million_usd`, `output_cost_per_million_usd`, `pricing_currency`, `max_output_tokens`, `throughput_tokens_per_second`, `onboarding_ready`, `is_default`, `use_for_title_generation`, `use_for_memory_extraction`, `sort_order`
