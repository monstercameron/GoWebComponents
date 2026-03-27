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
