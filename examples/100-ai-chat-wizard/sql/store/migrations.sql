ALTER TABLE conversations ADD COLUMN title TEXT;
ALTER TABLE conversations ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversations ADD COLUMN public_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN model_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN prompt_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN completion_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_profile ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_profile ADD COLUMN selected_model TEXT NOT NULL DEFAULT '';
ALTER TABLE user_profile ADD COLUMN selected_tone TEXT NOT NULL DEFAULT '';
ALTER TABLE user_profile ADD COLUMN selected_thinking_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE user_profile ADD COLUMN selected_thinking_effort TEXT NOT NULL DEFAULT 'medium';
ALTER TABLE user_profile ADD COLUMN selected_system_prompt TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profile_user_id ON user_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id, id);
CREATE TABLE IF NOT EXISTS user_memory (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL REFERENCES users(id),
    memory_key        TEXT NOT NULL,
    category          TEXT NOT NULL,
    summary           TEXT NOT NULL,
    detail            TEXT NOT NULL DEFAULT '',
    source_message    TEXT NOT NULL DEFAULT '',
    usefulness_score  INTEGER NOT NULL DEFAULT 0,
    confidence_score  REAL NOT NULL DEFAULT 0,
    rubric_reason     TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    UNIQUE(user_id, memory_key)
);
CREATE INDEX IF NOT EXISTS idx_user_memory_user_id ON user_memory(user_id, usefulness_score DESC, updated_at DESC);
CREATE TABLE IF NOT EXISTS usage_events (
    id                          INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id                    TEXT NOT NULL,
    user_id                     INTEGER NOT NULL REFERENCES users(id),
    conversation_id             INTEGER NOT NULL REFERENCES conversations(id),
    provider_id                 TEXT NOT NULL,
    model_id                    TEXT NOT NULL,
    prompt_tokens               INTEGER NOT NULL DEFAULT 0,
    completion_tokens           INTEGER NOT NULL DEFAULT 0,
    usage_source                TEXT NOT NULL DEFAULT 'missing',
    provider_request_id         TEXT NOT NULL DEFAULT '',
    input_cost_per_million_usd  REAL NOT NULL DEFAULT 0,
    output_cost_per_million_usd REAL NOT NULL DEFAULT 0,
    pricing_currency            TEXT NOT NULL DEFAULT 'USD',
    input_cost_usd              REAL NOT NULL DEFAULT 0,
    output_cost_usd             REAL NOT NULL DEFAULT 0,
    total_cost_usd              REAL NOT NULL DEFAULT 0,
    client_id                   TEXT NOT NULL DEFAULT '',
    trace_id                    TEXT NOT NULL DEFAULT '',
    span_id                     TEXT NOT NULL DEFAULT '',
    trace_state                 TEXT NOT NULL DEFAULT '',
    status                      TEXT NOT NULL DEFAULT 'completed',
    error_message               TEXT NOT NULL DEFAULT '',
    created_at                  TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_events_event_id ON usage_events(event_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_user_id ON usage_events(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_usage_events_conversation_id ON usage_events(conversation_id, id DESC);
CREATE TABLE IF NOT EXISTS billing_plans (
    plan_code                    TEXT PRIMARY KEY,
    plan_name                    TEXT NOT NULL,
    plan_rank                    INTEGER NOT NULL DEFAULT 0,
    is_active                    INTEGER NOT NULL DEFAULT 1,
    monthly_base_cents           INTEGER NOT NULL DEFAULT 0,
    yearly_base_cents            INTEGER NOT NULL DEFAULT 0,
    included_tokens_monthly      INTEGER NOT NULL DEFAULT 0,
    included_seats               INTEGER NOT NULL DEFAULT 1,
    max_seats                    INTEGER NOT NULL DEFAULT 1,
    supports_priority            INTEGER NOT NULL DEFAULT 0,
    supports_team_workspace      INTEGER NOT NULL DEFAULT 0,
    supports_sso                 INTEGER NOT NULL DEFAULT 0,
    created_at                   TEXT NOT NULL DEFAULT '',
    updated_at                   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_billing_plans_rank ON billing_plans(plan_rank, plan_code);
CREATE TABLE IF NOT EXISTS billing_plan_entitlements (
    plan_code                    TEXT NOT NULL REFERENCES billing_plans(plan_code),
    entitlement_key              TEXT NOT NULL,
    entitlement_value            TEXT NOT NULL,
    updated_at                   TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (plan_code, entitlement_key)
);
CREATE TABLE IF NOT EXISTS billing_plan_model_access (
    plan_code                    TEXT NOT NULL REFERENCES billing_plans(plan_code),
    model_id                     TEXT NOT NULL,
    is_default                   INTEGER NOT NULL DEFAULT 0,
    is_enabled                   INTEGER NOT NULL DEFAULT 1,
    updated_at                   TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (plan_code, model_id)
);
CREATE INDEX IF NOT EXISTS idx_billing_plan_model_access_plan ON billing_plan_model_access(plan_code, is_default DESC, model_id);
CREATE TABLE IF NOT EXISTS billing_customers (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                      INTEGER NOT NULL UNIQUE REFERENCES users(id),
    provider_id                  TEXT NOT NULL DEFAULT '',
    provider_customer_id         TEXT NOT NULL DEFAULT '',
    billing_email                TEXT NOT NULL DEFAULT '',
    billing_name                 TEXT NOT NULL DEFAULT '',
    billing_country              TEXT NOT NULL DEFAULT '',
    billing_region               TEXT NOT NULL DEFAULT '',
    default_currency             TEXT NOT NULL DEFAULT 'USD',
    tax_exempt_status            TEXT NOT NULL DEFAULT 'none',
    external_metadata_json       TEXT NOT NULL DEFAULT '{}',
    created_at                   TEXT NOT NULL DEFAULT '',
    updated_at                   TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_customers_provider_customer_id ON billing_customers(provider_customer_id);
CREATE INDEX IF NOT EXISTS idx_billing_customers_user_id ON billing_customers(user_id);
CREATE TABLE IF NOT EXISTS billing_subscriptions (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                  INTEGER NOT NULL REFERENCES billing_customers(id),
    provider_id                  TEXT NOT NULL DEFAULT '',
    provider_subscription_id     TEXT NOT NULL DEFAULT '',
    plan_code                    TEXT NOT NULL DEFAULT 'free',
    price_code                   TEXT NOT NULL DEFAULT '',
    status                       TEXT NOT NULL DEFAULT 'active',
    billing_interval             TEXT NOT NULL DEFAULT 'month',
    quantity                     INTEGER NOT NULL DEFAULT 1,
    current_period_start         TEXT NOT NULL DEFAULT '',
    current_period_end           TEXT NOT NULL DEFAULT '',
    cancel_at_period_end         INTEGER NOT NULL DEFAULT 0,
    canceled_at                  TEXT NOT NULL DEFAULT '',
    trial_ends_at                TEXT NOT NULL DEFAULT '',
    access_expires_at            TEXT NOT NULL DEFAULT '',
    created_at                   TEXT NOT NULL DEFAULT '',
    updated_at                   TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_subscriptions_provider_subscription_id ON billing_subscriptions(provider_subscription_id);
CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_customer_id ON billing_subscriptions(customer_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_status ON billing_subscriptions(status, current_period_end);
CREATE TABLE IF NOT EXISTS billing_invoices (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                  INTEGER NOT NULL REFERENCES billing_customers(id),
    subscription_id              INTEGER NOT NULL DEFAULT 0 REFERENCES billing_subscriptions(id),
    provider_id                  TEXT NOT NULL DEFAULT '',
    provider_invoice_id          TEXT NOT NULL DEFAULT '',
    status                       TEXT NOT NULL DEFAULT 'draft',
    currency                     TEXT NOT NULL DEFAULT 'USD',
    subtotal_cents               INTEGER NOT NULL DEFAULT 0,
    tax_cents                    INTEGER NOT NULL DEFAULT 0,
    discount_cents               INTEGER NOT NULL DEFAULT 0,
    total_cents                  INTEGER NOT NULL DEFAULT 0,
    amount_due_cents             INTEGER NOT NULL DEFAULT 0,
    amount_paid_cents            INTEGER NOT NULL DEFAULT 0,
    period_start                 TEXT NOT NULL DEFAULT '',
    period_end                   TEXT NOT NULL DEFAULT '',
    due_at                       TEXT NOT NULL DEFAULT '',
    paid_at                      TEXT NOT NULL DEFAULT '',
    hosted_invoice_url           TEXT NOT NULL DEFAULT '',
    external_metadata_json       TEXT NOT NULL DEFAULT '{}',
    created_at                   TEXT NOT NULL DEFAULT '',
    updated_at                   TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_invoices_provider_invoice_id ON billing_invoices(provider_invoice_id);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_customer_id ON billing_invoices(customer_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_status ON billing_invoices(status, due_at);
CREATE TABLE IF NOT EXISTS billing_invoice_line_items (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id                   INTEGER NOT NULL REFERENCES billing_invoices(id),
    usage_event_id               TEXT NOT NULL DEFAULT '',
    line_type                    TEXT NOT NULL DEFAULT 'usage',
    description                  TEXT NOT NULL DEFAULT '',
    quantity                     INTEGER NOT NULL DEFAULT 1,
    unit_amount_cents            INTEGER NOT NULL DEFAULT 0,
    amount_cents                 INTEGER NOT NULL DEFAULT 0,
    currency                     TEXT NOT NULL DEFAULT 'USD',
    period_start                 TEXT NOT NULL DEFAULT '',
    period_end                   TEXT NOT NULL DEFAULT '',
    created_at                   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_billing_invoice_line_items_invoice_id ON billing_invoice_line_items(invoice_id, id ASC);
CREATE INDEX IF NOT EXISTS idx_billing_invoice_line_items_usage_event_id ON billing_invoice_line_items(usage_event_id);
CREATE TABLE IF NOT EXISTS billing_access_overrides (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                  INTEGER NOT NULL REFERENCES billing_customers(id),
    override_key                 TEXT NOT NULL,
    override_value               TEXT NOT NULL,
    reason                       TEXT NOT NULL DEFAULT '',
    is_enabled                   INTEGER NOT NULL DEFAULT 1,
    starts_at                    TEXT NOT NULL DEFAULT '',
    ends_at                      TEXT NOT NULL DEFAULT '',
    actor_user_id                INTEGER NOT NULL DEFAULT 0,
    created_at                   TEXT NOT NULL DEFAULT '',
    updated_at                   TEXT NOT NULL DEFAULT '',
    UNIQUE(customer_id, override_key)
);
CREATE INDEX IF NOT EXISTS idx_billing_access_overrides_customer_id ON billing_access_overrides(customer_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS billing_events (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                  INTEGER NOT NULL REFERENCES billing_customers(id),
    subscription_id              INTEGER NOT NULL DEFAULT 0 REFERENCES billing_subscriptions(id),
    invoice_id                   INTEGER NOT NULL DEFAULT 0 REFERENCES billing_invoices(id),
    event_type                   TEXT NOT NULL,
    event_source                 TEXT NOT NULL DEFAULT 'system',
    event_summary                TEXT NOT NULL DEFAULT '',
    event_payload_json           TEXT NOT NULL DEFAULT '{}',
    actor_user_id                INTEGER NOT NULL DEFAULT 0,
    created_at                   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_billing_events_customer_id ON billing_events(customer_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_billing_events_subscription_id ON billing_events(subscription_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_billing_events_invoice_id ON billing_events(invoice_id, id DESC);
CREATE TABLE IF NOT EXISTS auth_sessions (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                    INTEGER NOT NULL REFERENCES users(id),
    session_id                 TEXT NOT NULL UNIQUE,
    token_version              INTEGER NOT NULL DEFAULT 1,
    refresh_token_hash         TEXT NOT NULL DEFAULT '',
    user_agent                 TEXT NOT NULL DEFAULT '',
    ip_address                 TEXT NOT NULL DEFAULT '',
    last_seen_at               TEXT NOT NULL DEFAULT '',
    expires_at                 TEXT NOT NULL DEFAULT '',
    revoked_at                 TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at, id DESC);
CREATE TABLE IF NOT EXISTS auth_token_versions (
    user_id                    INTEGER PRIMARY KEY REFERENCES users(id),
    token_version              INTEGER NOT NULL DEFAULT 1,
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                    INTEGER NOT NULL REFERENCES users(id),
    email                      TEXT NOT NULL COLLATE NOCASE,
    token_hash                 TEXT NOT NULL UNIQUE,
    status                     TEXT NOT NULL DEFAULT 'pending',
    requested_by_ip            TEXT NOT NULL DEFAULT '',
    expires_at                 TEXT NOT NULL DEFAULT '',
    consumed_at                TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_status ON password_reset_tokens(status, expires_at);
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                    INTEGER NOT NULL REFERENCES users(id),
    email                      TEXT NOT NULL COLLATE NOCASE,
    token_hash                 TEXT NOT NULL UNIQUE,
    status                     TEXT NOT NULL DEFAULT 'pending',
    expires_at                 TEXT NOT NULL DEFAULT '',
    verified_at                TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_id ON email_verification_tokens(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_status ON email_verification_tokens(status, expires_at);
INSERT OR IGNORE INTO billing_plans (
    plan_code, plan_name, plan_rank, is_active,
    monthly_base_cents, yearly_base_cents,
    included_tokens_monthly, included_seats, max_seats,
    supports_priority, supports_team_workspace, supports_sso,
    created_at, updated_at
) VALUES
    ('free', 'Free', 10, 1, 0, 0, 500000, 1, 1, 0, 0, 0, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z'),
    ('pro', 'Pro', 20, 1, 2900, 29000, 5000000, 1, 1, 1, 0, 0, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z'),
    ('team', 'Team', 30, 1, 9900, 99000, 20000000, 3, 50, 1, 1, 0, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z'),
    ('enterprise', 'Enterprise', 40, 1, 0, 0, 0, 10, 500, 1, 1, 1, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z');
INSERT OR IGNORE INTO billing_plan_entitlements (
    plan_code, entitlement_key, entitlement_value, updated_at
) VALUES
    ('free', 'chat.send.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('free', 'memory.editor.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('free', 'usage.monthly_token_limit', '500000', '1970-01-01T00:00:00Z'),
    ('pro', 'chat.send.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('pro', 'memory.editor.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('pro', 'usage.monthly_token_limit', '5000000', '1970-01-01T00:00:00Z'),
    ('pro', 'tts.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('team', 'chat.send.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('team', 'memory.editor.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('team', 'usage.monthly_token_limit', '20000000', '1970-01-01T00:00:00Z'),
    ('team', 'tts.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('team', 'workspace.multi_user.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('enterprise', 'chat.send.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('enterprise', 'memory.editor.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('enterprise', 'usage.monthly_token_limit', 'unlimited', '1970-01-01T00:00:00Z'),
    ('enterprise', 'tts.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('enterprise', 'workspace.multi_user.enabled', 'true', '1970-01-01T00:00:00Z'),
    ('enterprise', 'sso.enabled', 'true', '1970-01-01T00:00:00Z');
INSERT OR IGNORE INTO billing_plan_model_access (
    plan_code, model_id, is_default, is_enabled, updated_at
) VALUES
    ('free', 'gpt-5.4-mini', 1, 1, '1970-01-01T00:00:00Z'),
    ('free', 'gpt-5.4', 0, 1, '1970-01-01T00:00:00Z'),
    ('pro', 'gpt-5.4-mini', 1, 1, '1970-01-01T00:00:00Z'),
    ('pro', 'gpt-5.4', 0, 1, '1970-01-01T00:00:00Z'),
    ('team', 'gpt-5.4', 1, 1, '1970-01-01T00:00:00Z'),
    ('team', 'gpt-5.4-mini', 0, 1, '1970-01-01T00:00:00Z'),
    ('enterprise', 'gpt-5.4', 1, 1, '1970-01-01T00:00:00Z'),
    ('enterprise', 'gpt-5.4-mini', 0, 1, '1970-01-01T00:00:00Z');
CREATE TABLE IF NOT EXISTS su_roles (
    role_key                   TEXT PRIMARY KEY,
    label                      TEXT NOT NULL,
    description                TEXT NOT NULL DEFAULT '',
    is_system                  INTEGER NOT NULL DEFAULT 0,
    is_enabled                 INTEGER NOT NULL DEFAULT 1,
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS su_role_permissions (
    role_key                   TEXT NOT NULL REFERENCES su_roles(role_key),
    permission_key             TEXT NOT NULL,
    permission_value           TEXT NOT NULL DEFAULT 'allow',
    updated_at                 TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (role_key, permission_key)
);
CREATE TABLE IF NOT EXISTS su_user_roles (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                    INTEGER NOT NULL REFERENCES users(id),
    role_key                   TEXT NOT NULL REFERENCES su_roles(role_key),
    assigned_by_user_id        INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(user_id, role_key)
);
CREATE INDEX IF NOT EXISTS idx_su_user_roles_user_id ON su_user_roles(user_id, id DESC);
CREATE TABLE IF NOT EXISTS site_config (
    config_key                 TEXT PRIMARY KEY,
    config_value               TEXT NOT NULL DEFAULT '',
    value_type                 TEXT NOT NULL DEFAULT 'string',
    description                TEXT NOT NULL DEFAULT '',
    updated_by_user_id         INTEGER NOT NULL DEFAULT 0,
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS feature_flags (
    flag_key                   TEXT PRIMARY KEY,
    description                TEXT NOT NULL DEFAULT '',
    is_enabled                 INTEGER NOT NULL DEFAULT 0,
    rollout_percent            INTEGER NOT NULL DEFAULT 0,
    audience_json              TEXT NOT NULL DEFAULT '{}',
    payload_json               TEXT NOT NULL DEFAULT '{}',
    updated_by_user_id         INTEGER NOT NULL DEFAULT 0,
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS workspaces (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_key              TEXT NOT NULL UNIQUE,
    slug                       TEXT NOT NULL UNIQUE,
    name                       TEXT NOT NULL DEFAULT '',
    plan_code                  TEXT NOT NULL DEFAULT 'free',
    status                     TEXT NOT NULL DEFAULT 'active',
    owner_user_id              INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    settings_json              TEXT NOT NULL DEFAULT '{}',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_workspaces_owner_user_id ON workspaces(owner_user_id, id DESC);
CREATE TABLE IF NOT EXISTS workspace_memberships (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL REFERENCES users(id),
    role_key                   TEXT NOT NULL DEFAULT 'member',
    status                     TEXT NOT NULL DEFAULT 'active',
    invited_by_user_id         INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_memberships_workspace_id ON workspace_memberships(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_workspace_memberships_user_id ON workspace_memberships(user_id, id DESC);
CREATE TABLE IF NOT EXISTS workspace_invitations (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    email                      TEXT NOT NULL COLLATE NOCASE,
    role_key                   TEXT NOT NULL DEFAULT 'member',
    invitation_token_hash      TEXT NOT NULL UNIQUE,
    invited_by_user_id         INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    status                     TEXT NOT NULL DEFAULT 'pending',
    expires_at                 TEXT NOT NULL DEFAULT '',
    accepted_at                TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_workspace_id ON workspace_invitations(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_email ON workspace_invitations(email, id DESC);
CREATE TABLE IF NOT EXISTS api_keys (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    key_id                     TEXT NOT NULL UNIQUE,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    label                      TEXT NOT NULL DEFAULT '',
    key_prefix                 TEXT NOT NULL DEFAULT '',
    secret_hash                TEXT NOT NULL DEFAULT '',
    scopes_json                TEXT NOT NULL DEFAULT '[]',
    last_used_at               TEXT NOT NULL DEFAULT '',
    revoked_at                 TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_api_keys_workspace_id ON api_keys(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id, id DESC);
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    label                      TEXT NOT NULL DEFAULT '',
    target_url                 TEXT NOT NULL DEFAULT '',
    secret_hash                TEXT NOT NULL DEFAULT '',
    events_json                TEXT NOT NULL DEFAULT '[]',
    is_enabled                 INTEGER NOT NULL DEFAULT 1,
    last_delivery_at           TEXT NOT NULL DEFAULT '',
    failure_count              INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, target_url)
);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_workspace_id ON webhook_endpoints(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    endpoint_id                INTEGER NOT NULL REFERENCES webhook_endpoints(id),
    event_type                 TEXT NOT NULL DEFAULT '',
    delivery_key               TEXT NOT NULL UNIQUE,
    request_headers_json       TEXT NOT NULL DEFAULT '{}',
    request_body_json          TEXT NOT NULL DEFAULT '{}',
    response_status            INTEGER NOT NULL DEFAULT 0,
    response_body              TEXT NOT NULL DEFAULT '',
    attempt_count              INTEGER NOT NULL DEFAULT 0,
    delivered_at               TEXT NOT NULL DEFAULT '',
    failed_at                  TEXT NOT NULL DEFAULT '',
    next_retry_at              TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint_id ON webhook_deliveries(endpoint_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_next_retry_at ON webhook_deliveries(next_retry_at, id DESC);
CREATE TABLE IF NOT EXISTS audit_logs (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_user_id              INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    event_type                 TEXT NOT NULL DEFAULT '',
    target_type                TEXT NOT NULL DEFAULT '',
    target_id                  TEXT NOT NULL DEFAULT '',
    summary                    TEXT NOT NULL DEFAULT '',
    payload_json               TEXT NOT NULL DEFAULT '{}',
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_workspace_id ON audit_logs(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_user_id ON audit_logs(actor_user_id, id DESC);
CREATE TABLE IF NOT EXISTS support_tickets (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_key                 TEXT NOT NULL UNIQUE,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    status                     TEXT NOT NULL DEFAULT 'open',
    priority                   TEXT NOT NULL DEFAULT 'normal',
    subject                    TEXT NOT NULL DEFAULT '',
    body                       TEXT NOT NULL DEFAULT '',
    assignee_user_id           INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    resolution_note            TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_support_tickets_workspace_id ON support_tickets(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_id ON support_tickets(user_id, id DESC);
CREATE TABLE IF NOT EXISTS support_ticket_messages (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id                  INTEGER NOT NULL REFERENCES support_tickets(id),
    author_user_id             INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    message_type               TEXT NOT NULL DEFAULT 'reply',
    body                       TEXT NOT NULL DEFAULT '',
    is_internal                INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_ticket_id ON support_ticket_messages(ticket_id, id ASC);
CREATE TABLE IF NOT EXISTS experiments (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    experiment_key             TEXT NOT NULL UNIQUE,
    name                       TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL DEFAULT 'draft',
    variants_json              TEXT NOT NULL DEFAULT '[]',
    audience_json              TEXT NOT NULL DEFAULT '{}',
    start_at                   TEXT NOT NULL DEFAULT '',
    end_at                     TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS billing_plan_overages (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_code                  TEXT NOT NULL REFERENCES billing_plans(plan_code),
    meter_key                  TEXT NOT NULL DEFAULT '',
    included_units             INTEGER NOT NULL DEFAULT 0,
    soft_limit_units           INTEGER NOT NULL DEFAULT 0,
    hard_limit_units           INTEGER NOT NULL DEFAULT 0,
    overage_unit_size          INTEGER NOT NULL DEFAULT 1,
    overage_price_cents        INTEGER NOT NULL DEFAULT 0,
    billing_interval           TEXT NOT NULL DEFAULT 'monthly',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(plan_code, meter_key)
);
CREATE INDEX IF NOT EXISTS idx_billing_plan_overages_plan_code ON billing_plan_overages(plan_code, id DESC);
CREATE TABLE IF NOT EXISTS billing_quota_policies (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_code                  TEXT NOT NULL REFERENCES billing_plans(plan_code),
    quota_key                  TEXT NOT NULL DEFAULT '',
    soft_limit_value           INTEGER NOT NULL DEFAULT 0,
    hard_limit_value           INTEGER NOT NULL DEFAULT 0,
    reset_interval             TEXT NOT NULL DEFAULT 'monthly',
    enforcement_mode           TEXT NOT NULL DEFAULT 'block',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(plan_code, quota_key)
);
CREATE INDEX IF NOT EXISTS idx_billing_quota_policies_plan_code ON billing_quota_policies(plan_code, id DESC);
CREATE TABLE IF NOT EXISTS billing_upgrade_triggers (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_code                  TEXT NOT NULL REFERENCES billing_plans(plan_code),
    trigger_key                TEXT NOT NULL DEFAULT '',
    threshold_percent          INTEGER NOT NULL DEFAULT 0,
    upgrade_plan_code          TEXT NOT NULL DEFAULT '',
    message                    TEXT NOT NULL DEFAULT '',
    cta_label                  TEXT NOT NULL DEFAULT '',
    cta_url                    TEXT NOT NULL DEFAULT '',
    is_enabled                 INTEGER NOT NULL DEFAULT 1,
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(plan_code, trigger_key)
);
CREATE INDEX IF NOT EXISTS idx_billing_upgrade_triggers_plan_code ON billing_upgrade_triggers(plan_code, id DESC);
CREATE TABLE IF NOT EXISTS billing_dunning_events (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                INTEGER NOT NULL DEFAULT 0 REFERENCES billing_customers(id),
    subscription_id            INTEGER NOT NULL DEFAULT 0 REFERENCES billing_subscriptions(id),
    invoice_id                 INTEGER NOT NULL DEFAULT 0 REFERENCES billing_invoices(id),
    status                     TEXT NOT NULL DEFAULT 'pending',
    attempt_count              INTEGER NOT NULL DEFAULT 0,
    failure_reason             TEXT NOT NULL DEFAULT '',
    next_attempt_at            TEXT NOT NULL DEFAULT '',
    resolved_at                TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_billing_dunning_events_customer_id ON billing_dunning_events(customer_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_billing_dunning_events_subscription_id ON billing_dunning_events(subscription_id, id DESC);
CREATE TABLE IF NOT EXISTS onboarding_templates (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    template_key               TEXT NOT NULL UNIQUE,
    title                      TEXT NOT NULL DEFAULT '',
    category                   TEXT NOT NULL DEFAULT '',
    prompt_text                TEXT NOT NULL DEFAULT '',
    checklist_json             TEXT NOT NULL DEFAULT '[]',
    is_default                 INTEGER NOT NULL DEFAULT 0,
    sort_order                 INTEGER NOT NULL DEFAULT 0,
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS user_activation_milestones (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                    INTEGER NOT NULL REFERENCES users(id),
    milestone_key              TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL DEFAULT 'pending',
    achieved_at                TEXT NOT NULL DEFAULT '',
    metadata_json              TEXT NOT NULL DEFAULT '{}',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(user_id, milestone_key)
);
CREATE INDEX IF NOT EXISTS idx_user_activation_milestones_user_id ON user_activation_milestones(user_id, id DESC);
CREATE TABLE IF NOT EXISTS saved_workflows (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    workflow_key               TEXT NOT NULL UNIQUE,
    name                       TEXT NOT NULL DEFAULT '',
    description                TEXT NOT NULL DEFAULT '',
    workflow_json              TEXT NOT NULL DEFAULT '{}',
    is_public                  INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_saved_workflows_workspace_id ON saved_workflows(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_saved_workflows_user_id ON saved_workflows(user_id, id DESC);
CREATE TABLE IF NOT EXISTS prompt_library_items (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    item_key                   TEXT NOT NULL DEFAULT '',
    title                      TEXT NOT NULL DEFAULT '',
    category                   TEXT NOT NULL DEFAULT '',
    prompt_text                TEXT NOT NULL DEFAULT '',
    tags_json                  TEXT NOT NULL DEFAULT '[]',
    is_public                  INTEGER NOT NULL DEFAULT 0,
    use_count                  INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, item_key)
);
CREATE INDEX IF NOT EXISTS idx_prompt_library_items_workspace_id ON prompt_library_items(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_library_items_user_id ON prompt_library_items(user_id, id DESC);
CREATE TABLE IF NOT EXISTS workspace_asset_shares (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    asset_type                 TEXT NOT NULL DEFAULT '',
    asset_id                   TEXT NOT NULL DEFAULT '',
    target_workspace_id        INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    target_user_id             INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    shared_by_user_id          INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    permission_key             TEXT NOT NULL DEFAULT 'view',
    created_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, asset_type, asset_id, target_workspace_id, target_user_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_asset_shares_workspace_id ON workspace_asset_shares(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS weekly_value_summaries (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    summary_week               TEXT NOT NULL DEFAULT '',
    summary_text               TEXT NOT NULL DEFAULT '',
    metrics_json               TEXT NOT NULL DEFAULT '{}',
    sent_at                    TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, user_id, summary_week)
);
CREATE INDEX IF NOT EXISTS idx_weekly_value_summaries_workspace_id ON weekly_value_summaries(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_weekly_value_summaries_user_id ON weekly_value_summaries(user_id, id DESC);
CREATE TABLE IF NOT EXISTS product_analytics_events (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    session_key                TEXT NOT NULL DEFAULT '',
    event_name                 TEXT NOT NULL DEFAULT '',
    funnel_key                 TEXT NOT NULL DEFAULT '',
    step_key                   TEXT NOT NULL DEFAULT '',
    experiment_key             TEXT NOT NULL DEFAULT '' REFERENCES experiments(experiment_key),
    variant_key                TEXT NOT NULL DEFAULT '',
    event_props_json           TEXT NOT NULL DEFAULT '{}',
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_product_analytics_events_workspace_id ON product_analytics_events(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_product_analytics_events_user_id ON product_analytics_events(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_product_analytics_events_experiment_key ON product_analytics_events(experiment_key, id DESC);
CREATE TABLE IF NOT EXISTS experiment_assignments (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    experiment_key             TEXT NOT NULL REFERENCES experiments(experiment_key),
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    variant_key                TEXT NOT NULL DEFAULT '',
    assigned_at                TEXT NOT NULL DEFAULT '',
    UNIQUE(experiment_key, workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_experiment_assignments_workspace_id ON experiment_assignments(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_experiment_assignments_user_id ON experiment_assignments(user_id, id DESC);
CREATE TABLE IF NOT EXISTS subscription_churn_feedback (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id                INTEGER NOT NULL DEFAULT 0 REFERENCES billing_customers(id),
    subscription_id            INTEGER NOT NULL DEFAULT 0 REFERENCES billing_subscriptions(id),
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    reason_key                 TEXT NOT NULL DEFAULT '',
    detail                     TEXT NOT NULL DEFAULT '',
    recovery_offer_key         TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_subscription_churn_feedback_customer_id ON subscription_churn_feedback(customer_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_subscription_churn_feedback_subscription_id ON subscription_churn_feedback(subscription_id, id DESC);
CREATE TABLE IF NOT EXISTS workspace_sso_configs (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    provider_key               TEXT NOT NULL DEFAULT '',
    saml_entrypoint            TEXT NOT NULL DEFAULT '',
    saml_issuer                TEXT NOT NULL DEFAULT '',
    saml_certificate_pem       TEXT NOT NULL DEFAULT '',
    domains_json               TEXT NOT NULL DEFAULT '[]',
    is_enabled                 INTEGER NOT NULL DEFAULT 0,
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, provider_key)
);
CREATE INDEX IF NOT EXISTS idx_workspace_sso_configs_workspace_id ON workspace_sso_configs(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS data_retention_policies (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    scope_key                  TEXT NOT NULL DEFAULT '',
    retention_days             INTEGER NOT NULL DEFAULT 0,
    purge_mode                 TEXT NOT NULL DEFAULT 'delete',
    legal_hold_json            TEXT NOT NULL DEFAULT '{}',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, scope_key)
);
CREATE INDEX IF NOT EXISTS idx_data_retention_policies_workspace_id ON data_retention_policies(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS compliance_controls (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    control_key                TEXT NOT NULL DEFAULT '',
    framework_key              TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL DEFAULT 'planned',
    owner_user_id              INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    evidence_url               TEXT NOT NULL DEFAULT '',
    reviewed_at                TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, control_key, framework_key)
);
CREATE INDEX IF NOT EXISTS idx_compliance_controls_workspace_id ON compliance_controls(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS service_level_objectives (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    slo_key                    TEXT NOT NULL UNIQUE,
    service_name               TEXT NOT NULL DEFAULT '',
    objective_percent          REAL NOT NULL DEFAULT 99.9,
    window_days                INTEGER NOT NULL DEFAULT 30,
    error_budget_minutes       INTEGER NOT NULL DEFAULT 0,
    status_page_url            TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS incidents (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_key               TEXT NOT NULL UNIQUE,
    slo_key                    TEXT NOT NULL DEFAULT '' REFERENCES service_level_objectives(slo_key),
    severity                   TEXT NOT NULL DEFAULT 'minor',
    status                     TEXT NOT NULL DEFAULT 'open',
    title                      TEXT NOT NULL DEFAULT '',
    summary                    TEXT NOT NULL DEFAULT '',
    started_at                 TEXT NOT NULL DEFAULT '',
    resolved_at                TEXT NOT NULL DEFAULT '',
    postmortem_url             TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_incidents_slo_key ON incidents(slo_key, id DESC);
CREATE TABLE IF NOT EXISTS incident_updates (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id                INTEGER NOT NULL REFERENCES incidents(id),
    status                     TEXT NOT NULL DEFAULT 'investigating',
    message                    TEXT NOT NULL DEFAULT '',
    is_public                  INTEGER NOT NULL DEFAULT 1,
    published_at               TEXT NOT NULL DEFAULT '',
    created_by_user_id         INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    created_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_incident_updates_incident_id ON incident_updates(incident_id, id ASC);
CREATE TABLE IF NOT EXISTS account_health_scores (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    score_value                INTEGER NOT NULL DEFAULT 0,
    risk_level                 TEXT NOT NULL DEFAULT 'unknown',
    signals_json               TEXT NOT NULL DEFAULT '{}',
    owner_user_id              INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    scored_at                  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_account_health_scores_workspace_id ON account_health_scores(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS customer_success_playbooks (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    playbook_key               TEXT NOT NULL UNIQUE,
    title                      TEXT NOT NULL DEFAULT '',
    segment_key                TEXT NOT NULL DEFAULT '',
    steps_json                 TEXT NOT NULL DEFAULT '[]',
    is_active                  INTEGER NOT NULL DEFAULT 1,
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS notification_outbox (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    notification_key           TEXT NOT NULL DEFAULT '',
    channel_key                TEXT NOT NULL DEFAULT 'email',
    template_key               TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL DEFAULT 'pending',
    subject                    TEXT NOT NULL DEFAULT '',
    body_text                  TEXT NOT NULL DEFAULT '',
    payload_json               TEXT NOT NULL DEFAULT '{}',
    dedupe_key                 TEXT NOT NULL DEFAULT '',
    scheduled_at               TEXT NOT NULL DEFAULT '',
    sent_at                    TEXT NOT NULL DEFAULT '',
    failed_at                  TEXT NOT NULL DEFAULT '',
    error_message              TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_workspace_id ON notification_outbox(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_user_id ON notification_outbox(user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_status ON notification_outbox(status, scheduled_at);
CREATE TABLE IF NOT EXISTS background_jobs (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_key                    TEXT NOT NULL UNIQUE,
    job_type                   TEXT NOT NULL DEFAULT '',
    queue_key                  TEXT NOT NULL DEFAULT 'default',
    status                     TEXT NOT NULL DEFAULT 'pending',
    attempt_count              INTEGER NOT NULL DEFAULT 0,
    max_attempts               INTEGER NOT NULL DEFAULT 1,
    payload_json               TEXT NOT NULL DEFAULT '{}',
    run_after                  TEXT NOT NULL DEFAULT '',
    started_at                 TEXT NOT NULL DEFAULT '',
    finished_at                TEXT NOT NULL DEFAULT '',
    error_message              TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_background_jobs_status ON background_jobs(status, run_after, id DESC);
CREATE TABLE IF NOT EXISTS integration_connections (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    provider_key               TEXT NOT NULL DEFAULT '',
    external_account_id        TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL DEFAULT 'connected',
    scopes_json                TEXT NOT NULL DEFAULT '[]',
    config_json                TEXT NOT NULL DEFAULT '{}',
    connected_at               TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, provider_key, external_account_id)
);
CREATE INDEX IF NOT EXISTS idx_integration_connections_workspace_id ON integration_connections(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS partner_referrals (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    partner_key                TEXT NOT NULL DEFAULT '',
    workspace_id               INTEGER NOT NULL DEFAULT 0 REFERENCES workspaces(id),
    user_id                    INTEGER NOT NULL DEFAULT 0 REFERENCES users(id),
    referral_code              TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL DEFAULT 'pending',
    revenue_share_bps          INTEGER NOT NULL DEFAULT 0,
    notes                      TEXT NOT NULL DEFAULT '',
    created_at                 TEXT NOT NULL DEFAULT '',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(partner_key, referral_code)
);
CREATE INDEX IF NOT EXISTS idx_partner_referrals_workspace_id ON partner_referrals(workspace_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_partner_referrals_user_id ON partner_referrals(user_id, id DESC);
CREATE TABLE IF NOT EXISTS workspace_model_routing_policies (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    policy_key                 TEXT NOT NULL DEFAULT '',
    default_model_id           TEXT NOT NULL DEFAULT '' REFERENCES model_catalog(id),
    fallback_model_id          TEXT NOT NULL DEFAULT '' REFERENCES model_catalog(id),
    max_input_cost_per_million_usd REAL NOT NULL DEFAULT 0,
    max_output_cost_per_million_usd REAL NOT NULL DEFAULT 0,
    requires_approval          INTEGER NOT NULL DEFAULT 0,
    rules_json                 TEXT NOT NULL DEFAULT '{}',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, policy_key)
);
CREATE INDEX IF NOT EXISTS idx_workspace_model_routing_policies_workspace_id ON workspace_model_routing_policies(workspace_id, id DESC);
CREATE TABLE IF NOT EXISTS workspace_cost_guardrails (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id               INTEGER NOT NULL REFERENCES workspaces(id),
    guardrail_key              TEXT NOT NULL DEFAULT '',
    daily_budget_cents         INTEGER NOT NULL DEFAULT 0,
    monthly_budget_cents       INTEGER NOT NULL DEFAULT 0,
    max_cost_per_request_cents INTEGER NOT NULL DEFAULT 0,
    alert_threshold_percent    INTEGER NOT NULL DEFAULT 80,
    action_mode                TEXT NOT NULL DEFAULT 'notify',
    updated_at                 TEXT NOT NULL DEFAULT '',
    UNIQUE(workspace_id, guardrail_key)
);
CREATE INDEX IF NOT EXISTS idx_workspace_cost_guardrails_workspace_id ON workspace_cost_guardrails(workspace_id, id DESC);
INSERT OR IGNORE INTO su_roles (
    role_key, label, description, is_system, is_enabled, created_at, updated_at
) VALUES
    ('su', 'Superuser', 'Full control plane access across the application.', 1, 1, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z'),
    ('admin', 'Admin', 'Operational admin access without superuser ownership.', 1, 1, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z'),
    ('customer', 'Customer', 'Default end-user role for paid or free chat customers.', 1, 1, '1970-01-01T00:00:00Z', '1970-01-01T00:00:00Z');
INSERT OR IGNORE INTO su_role_permissions (
    role_key, permission_key, permission_value, updated_at
) VALUES
    ('su', 'control_plane.*', 'allow', '1970-01-01T00:00:00Z'),
    ('admin', 'control_plane.read', 'allow', '1970-01-01T00:00:00Z'),
    ('admin', 'workspace.manage', 'allow', '1970-01-01T00:00:00Z'),
    ('customer', 'chat.use', 'allow', '1970-01-01T00:00:00Z');
INSERT OR IGNORE INTO billing_plan_overages (
    plan_code, meter_key, included_units, soft_limit_units, hard_limit_units,
    overage_unit_size, overage_price_cents, billing_interval, updated_at
) VALUES
    ('free', 'monthly_tokens', 500000, 400000, 500000, 100000, 0, 'monthly', '1970-01-01T00:00:00Z'),
    ('pro', 'monthly_tokens', 5000000, 4500000, 6000000, 1000000, 500, 'monthly', '1970-01-01T00:00:00Z'),
    ('team', 'monthly_tokens', 20000000, 18000000, 25000000, 1000000, 400, 'monthly', '1970-01-01T00:00:00Z'),
    ('enterprise', 'monthly_tokens', 0, 0, 0, 1000000, 0, 'monthly', '1970-01-01T00:00:00Z');
INSERT OR IGNORE INTO billing_quota_policies (
    plan_code, quota_key, soft_limit_value, hard_limit_value, reset_interval, enforcement_mode, updated_at
) VALUES
    ('free', 'workspace.seats', 1, 1, 'monthly', 'block', '1970-01-01T00:00:00Z'),
    ('pro', 'workspace.seats', 1, 1, 'monthly', 'block', '1970-01-01T00:00:00Z'),
    ('team', 'workspace.seats', 3, 50, 'monthly', 'block', '1970-01-01T00:00:00Z'),
    ('enterprise', 'workspace.seats', 10, 500, 'monthly', 'review', '1970-01-01T00:00:00Z'),
    ('free', 'usage.monthly_tokens', 400000, 500000, 'monthly', 'block', '1970-01-01T00:00:00Z'),
    ('pro', 'usage.monthly_tokens', 4500000, 6000000, 'monthly', 'bill_overage', '1970-01-01T00:00:00Z'),
    ('team', 'usage.monthly_tokens', 18000000, 25000000, 'monthly', 'bill_overage', '1970-01-01T00:00:00Z'),
    ('enterprise', 'usage.monthly_tokens', 0, 0, 'monthly', 'contract', '1970-01-01T00:00:00Z');
INSERT OR IGNORE INTO billing_upgrade_triggers (
    plan_code, trigger_key, threshold_percent, upgrade_plan_code, message, cta_label, cta_url, is_enabled, updated_at
) VALUES
    ('free', 'monthly_tokens_80', 80, 'pro', 'You are approaching the Free plan token cap.', 'Upgrade to Pro', '/pricing', 1, '1970-01-01T00:00:00Z'),
    ('pro', 'monthly_tokens_90', 90, 'team', 'Your Pro workspace is close to its monthly token limit.', 'Upgrade to Team', '/pricing', 1, '1970-01-01T00:00:00Z'),
    ('team', 'monthly_tokens_90', 90, 'enterprise', 'Your Team workspace is nearing its monthly token limit.', 'Contact Sales', '/enterprise', 1, '1970-01-01T00:00:00Z');
