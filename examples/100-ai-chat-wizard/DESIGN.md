# RelayDesk Design Intent

This document captures the current intended product surface for example 100.
It replaces the older page-by-page visual overhaul spec and aligns design intent
to the shipped runtime, route model, and operator workflows.

## Product intent

RelayDesk is a shell-first chat workspace where:

- visitors can evaluate value on public routes,
- users can authenticate and send first chat quickly,
- returning users can reopen prior threads reliably,
- operators can inspect platform health and admin slices with strict authz.

The design goal is predictable task completion over ornamental UI.

## Primary journeys

### Visitor to first chat

1. Land on public route (`/`, `/home`, `/capabilities`, `/pricing`, `/signup`).
2. Enter auth path (login default, signup when requested).
3. Boot authenticated shell at `/app`.
4. Send first prompt and see stream start immediately.
5. Normalize to canonical `/app/thread/:publicID`.
6. Refresh and reopen without losing route-state continuity.

### Returning user

1. Re-enter `/app` with persisted session.
2. Rehydrate profile/settings/catalog/conversation list.
3. Continue recent thread or start a new one without picker drift.

### Admin and superuser journey

1. Authenticate into app shell.
2. Resolve role and scope.
3. Enter dashboard-capable surface.
4. Inspect summary slices and drill into detail.
5. Execute high-impact actions with confirmation and audit trail.

## IA and route model

### Current route classes

- Public marketing and auth-entry routes: `/`, `/home`, `/capabilities`, `/pricing`, `/signup`.
- Authenticated chat routes: `/app`, `/app/thread/:publicID`, `/app/thread/:publicID/canvas/:canvasID`.
- Settings surface: `/app/settings?panel=settings-*`.
- Runtime assets and transport endpoints: `/chat-bootstrap.js`, `/app/chat.wasm`, `/worker/background-worker.wasm`, `/healthz`, `/socket`.

### Current admin IA state

- Admin and control-plane data access is RPC-gated.
- Superuser checks are enforced for admin analytics/control-plane RPCs.
- Dedicated dashboard browser-route IA is still in-progress.

### Intended admin IA direction

- Keep normal user chat flow uncluttered.
- Add a clear but scoped admin entry affordance for eligible roles.
- Split data and actions by role:
  - normal user: no admin surfaces,
  - workspace admin: workspace-scoped slices,
  - superuser: platform-scoped slices and control-plane access.

## Surface definitions

### Chat workspace

- Left: thread list and route-stable thread reopen.
- Center: streamed conversation with bottom anchoring and jump-to-bottom behavior.
- Top controls: provider, model, and intelligence settings with stable persistence.
- Composer: first-send path with clear loading/error states and entitlement feedback.

### Settings

- Settings remain in-app under `/app/settings`.
- Panels include profile, tone, prompt, intelligence, speech, memories, and language.
- Design priority is scanability and low-friction edits over deep nested controls.

### Dashboard and operations

- Dashboard home should summarize operational health before drill-down.
- Slice views should support search, filter, sort, and pagination state continuity.
- Mutations (disable/restore/suspend/override/rollback) require confirmation and reason capture.
- Denied/expired auth must fail closed and return to safe non-privileged state.

### Core dashboard surfaces

- `Business`: revenue, plan mix, churn, invoice health, overage/upgrade/dunning controls, and account-level commercial risk.
  Not in `Business`: per-thread chat debugging, provider incident triage, or low-level webhook/job troubleshooting.
- `Customers`: user/workspace health, lifecycle, entitlements, billing/support context, session/access state, and operator account actions.
  Not in `Customers`: global pricing policy authoring, provider routing configuration, or site-level reliability policy.
- `Chats`: first-chat funnel, send/stream health, thread/message quality, memory/onboarding/workflow/skill defaults, and chat-surface settings.
  Not in `Chats`: subscription ledger policy, provider quota routing controls, or infra incident command.
- `Providers`: provider/model availability, latency/error/cost trend analysis, routing/fallback controls, model visibility, and guardrail limits.
  Not in `Providers`: customer-profile moderation workflows, support-ticket handling, or direct subscription plan edits.
- `Ops`: site health, incidents/SLOs, background jobs, notification/webhook reliability, feature flags, retention/integration settings, and audit signals.
  Not in `Ops`: business KPI interpretation, pricing package design, or per-customer relationship management.

### Dashboard KPI definitions (planned)

| KPI | Source tables | Source RPC | Caveats |
| --- | --- | --- | --- |
| MRR trend (`Business`) | `billing_subscriptions`, `billing_invoices`, `billing_invoice_line_items` | `GetBusinessDashboardSummary` | Invoice posting lag and proration can temporarily skew period totals. |
| Churn and downgrade risk (`Business`) | `subscription_churn_feedback`, `billing_events`, `billing_access_overrides` | `GetBusinessDashboardSummary` | Voluntary churn reason quality depends on user-provided feedback coverage. |
| Active customers/workspaces (`Customers`) | `users`, `workspaces`, `workspace_memberships`, `auth_sessions` | `GetCustomersDashboardSummary` | Session freshness windows can overcount recently inactive users. |
| Customer health and intervention load (`Customers`) | `support_tickets`, `support_ticket_messages`, `usage_events`, `billing_subscriptions` | `GetCustomersDashboardSummary` | Health score is derived and should be shown with component signal breakdown. |
| First-chat conversion (`Chats`) | `product_analytics_events`, `conversations`, `messages`, `user_activation_milestones` | `GetChatsDashboardSummary` | Funnel steps can be delayed when analytics writes are batched. |
| Reply quality and latency (`Chats`) | `messages`, `usage_events` | `GetChatsDashboardSummary` | Tail latency should be percentile-based, not average-only. |
| Provider reliability (`Providers`) | `usage_events`, `model_catalog`, provider-health tables | `GetProvidersDashboardSummary` | Provider outage windows may need rollup smoothing for noisy short spikes. |
| Provider cost per token (`Providers`) | `usage_events`, `workspace_model_routing_policies`, `workspace_cost_guardrails` | `GetProvidersDashboardSummary` | Token-cost attribution may differ by model pricing snapshot timing. |
| Incident and SLO posture (`Ops`) | `incidents`, `incident_updates`, `service_level_objectives` | `GetOpsDashboardSummary` | SLO burn calculations should include window length in UI labels. |
| Async delivery reliability (`Ops`) | `background_jobs`, `notification_outbox`, `webhook_deliveries`, `webhook_endpoints` | `GetOpsDashboardSummary` | Retry-heavy periods can mask first-attempt failure rates unless split out. |

### Dashboard endpoint and query map (planned)

| Surface | Summary RPCs | Drill-down RPCs | Mutation RPCs | Store funcs / SQL families | Primary tables |
| --- | --- | --- | --- | --- | --- |
| `Business` | `GetBusinessDashboardSummary` | `ListBusinessRevenueSeries`, `ListBusinessTopAccounts`, `ListBusinessDunningQueue` | `Set/DeleteBusinessPlan`, `Set/DeleteBusinessQuotaPolicy`, `Set/DeleteBusinessOverageRule`, `Set/DeleteBusinessUpgradeTrigger`, `Set/DeleteBusinessDunningRule` | `parseGetBusiness*`, `parseListBusiness*`, `parseStoreBusiness*`; `sql/store/get_business_*.sql`, `list_business_*.sql`, `upsert_business_*.sql`, `delete_business_*.sql` | `billing_customers`, `billing_subscriptions`, `billing_invoices`, `billing_invoice_line_items`, `billing_events`, `billing_access_overrides`, `subscription_churn_feedback`, `usage_events` |
| `Customers` | `GetCustomersDashboardSummary` | `ListCustomers`, `GetCustomerDetail`, `ListCustomerSessions`, `ListCustomerUsage`, `ListCustomerSupport` | `Disable/RestoreAdminUser`, `Suspend/RestoreAdminWorkspace`, `SetAdminBillingAccessOverride`, `SetAdminBillingQuotaOverride`, support-action mutations | `parseSearchAdminUsers`, `parseGetAdminUserSummary*`, `parseListAdmin*`; `sql/store/get_admin_*.sql`, `list_admin_*.sql`, `upsert_user_access_*.sql` | `users`, `user_profile`, `user_memory`, `workspaces`, `workspace_memberships`, `auth_sessions`, `auth_token_versions`, `support_tickets`, `support_ticket_messages`, `usage_events`, `audit_logs` |
| `Chats` | `GetChatsDashboardSummary` | `ListChatHealthThreads`, `ListChatLatencySeries`, `ListChatFailures`, `GetChatSettingState` | `SetSystemDefaultPrompt`, `SetModelDefaults`, `SetMemoryRules`, `SetOnboardingTemplate`, `PublishWorkflow`, `PublishSkill` | `parseGetChats*`, `parseListChats*`, `parseStoreChatSettings*`; `sql/store/get_chats_*.sql`, `list_chats_*.sql`, `upsert_chat_settings_*.sql` | `conversations`, `messages`, `usage_events`, `onboarding_templates`, `user_activation_milestones`, `saved_workflows`, `prompt_library_items`, `weekly_value_summaries`, `product_analytics_events` |
| `Providers` | `GetProvidersDashboardSummary` | `ListProviderHealth`, `ListProviderCostSeries`, `ListProviderRoutingPolicies`, `ListProviderGuardrails` | `SetProviderEnabled`, `SetModelVisibility`, `SetProviderFallbackPolicy`, `SetProviderLimit`, `SetWorkspaceCostGuardrail` | `parseGetProvider*`, `parseListProvider*`, `parseStoreProvider*`; `sql/store/get_provider_*.sql`, `list_provider_*.sql`, `upsert_provider_*.sql` | `model_catalog`, `usage_events`, `workspace_model_routing_policies`, `workspace_cost_guardrails`, provider-health/rate-limit tables |
| `Ops` | `GetOpsDashboardSummary` | `ListOpsIncidents`, `ListOpsJobFailures`, `ListOpsWebhookFailures`, `ListOpsAuditFeed` | `SetSiteConfig`, `SetFeatureFlag`, `SetRetentionPolicy`, `SetWebhookBehavior`, `SetIntegrationConfig` | `parseGetOps*`, `parseListOps*`, `parseStoreOps*`; `sql/store/get_ops_*.sql`, `list_ops_*.sql`, `upsert_ops_*.sql` | `site_config`, `feature_flags`, `audit_logs`, `service_level_objectives`, `incidents`, `incident_updates`, `background_jobs`, `notification_outbox`, `webhook_endpoints`, `webhook_deliveries` |

### Dashboard data-source map

- `Business` surface:
  - KPI cards (`MRR trend`, `churn risk`, `active customers`, `failed-payment count`, `upgrade pressure`) -> `GetBusinessDashboardSummary`, `parseGetBusinessSummary` families, `get_business_summary.sql` + related `list_business_*.sql` reads (`billing_subscriptions`, `billing_invoices`, `billing_invoice_line_items`, `subscription_churn_feedback`, `billing_access_overrides`, `billing_events`, `billing_dunning_queue`), plus customer-level rollups in `billing_customers`.
  - Trend blocks (`MRR trend`, `charge failure trend`, `plan migration trend`) -> `ListBusinessRevenueSeries`, `parseListBusinessSeries` + `list_business_revenue_series.sql` / `list_business_plan_mix_series.sql`.
  - Lists (`top accounts`, `dunning queue`, `failed-payment cases`) -> `ListBusinessTopAccounts`, `ListBusinessDunningQueue`, `parseListBusiness*` + `list_business_top_accounts.sql` / `list_business_dunning_queue.sql`.
  - Settings forms (`plan`, `quota`, `overage`, `upgrade trigger`, `dunning rule`) -> `Set/DeleteBusinessPlan`, `Set/DeleteBusinessQuotaPolicy`, `Set/DeleteBusinessOverageRule`, `Set/DeleteBusinessUpgradeTrigger`, `Set/DeleteBusinessDunningRule`, `parseStoreBusiness*` + `upsert_business_*.sql` / `delete_business_*.sql`.

- `Customers` surface:
  - KPI cards (`active users`, `active workspaces`, `disable/suspend posture`, `support backlog`) -> `GetCustomersDashboardSummary`, `parseGetCustomersSummary` families, `get_customers_summary.sql` + list helpers (`list_admin_users.sql`, `list_admin_support_tickets.sql`), tables `users`, `workspaces`, `workspace_memberships`, `auth_sessions`, `support_tickets`, `support_ticket_messages`.
  - Trend blocks (`usage trend`, `support age trend`) -> `GetCustomersDashboardSummary`, `parseGetCustomersSummary`, `list_customer_usage_trends.sql` / `list_support_age_series.sql` (where available).
  - Lists (`user list`, `workspace list`, `support queue`, `sessions`, `billing overrides`) -> `ListCustomers`, `ListCustomerSessions`, `ListCustomerUsage`, `ListCustomerSupport`, `parseListAdmin*`, `list_admin_users.sql` + `list_admin_support_tickets.sql` + `list_auth_sessions.sql`; tables in section summary.
  - Settings forms (`billing access override`, `quota override`, workspace suspension`) -> `SetAdminBillingAccessOverride`, `SetAdminBillingQuotaOverride`, `SetAdminWorkspaceSuspend`, `SetAdminWorkspaceRestore`, `parseStoreCustomerSettings*` + `upsert_user_access_overrides.sql` / `upsert_workspace_admin_actions.sql`.

- `Chats` surface:
  - KPI cards (`first-chat conversion`, `reply success`, `reply latency`, `memory quality`, `model fallback rate`) -> `GetChatsDashboardSummary`, `parseGetChatsSummary` families, `get_chats_summary.sql`; tables `product_analytics_events`, `conversations`, `messages`, `usage_events`, `user_activation_milestones`.
  - Trend blocks (`completion rate trend`, `latency trend`, `failure trend`) -> `ListChatHealthThreads`, `ListChatLatencySeries`, `ListChatFailures`, `parseListChat*` + `list_chat_health_threads.sql` / `list_chat_latency_series.sql` / `list_chat_failures.sql`.
  - Lists (`thread health list`, `failure detail list`) -> same drill-down RPC families and list SQL (`list_chats_threads.sql`, `list_message_replies.sql`, `list_chat_failures.sql`).
  - Settings forms (`system prompt`, `model defaults`, `memory rules`, `onboarding template`, `workflow publish`, `skill publish`) -> `SetSystemDefaultPrompt`, `SetModelDefaults`, `SetMemoryRules`, `SetOnboardingTemplate`, `PublishWorkflow`, `PublishSkill`, `parseStoreChatSettings*` + `upsert_system_prompt*.sql`, `upsert_chat_model_defaults.sql`, `upsert_memory_rules.sql`.

- `Providers` surface:
  - KPI cards (`provider reliability`, `cost per token`, `fallback rate`, `routing compliance`) -> `GetProvidersDashboardSummary`, `parseGetProvidersSummary`, `get_provider_summary.sql`; tables `usage_events`, `model_catalog`, `workspace_model_routing_policies`, `workspace_cost_guardrails`.
  - Trend blocks (`latency trend`, `error trend`, `cost trend`) -> `ListProviderHealth`, `ListProviderCostSeries`, `ListProviderRoutingPolicies`, `parseListProvider*` + `list_provider_health.sql` / `list_provider_cost_series.sql`.
  - Lists (`routing policy list`, `guardrail list`, `fallback event list`) -> `ListProviderRoutingPolicies`, `ListProviderGuardrails`, `parseListProvider*` + `list_provider_routing_policies.sql`, `list_provider_guardrails.sql`, `list_provider_fallback_events.sql` (if enabled).
  - Settings forms (`enable/disable provider`, `model visibility`, `fallback policy`, `provider/workspace limit`, `cost guardrail`) -> `SetProviderEnabled`, `SetModelVisibility`, `SetProviderFallbackPolicy`, `SetProviderLimit`, `SetWorkspaceCostGuardrail`, `parseStoreProvider*` + `upsert_provider_enabled.sql`, `upsert_model_visibility.sql`, `upsert_provider_limits.sql`.

- `Ops` surface:
  - KPI cards (`incident health`, `job backlog`, `webhook reliability`, `notification reliability`, `audit risk`) -> `GetOpsDashboardSummary`, `parseGetOpsSummary`, `get_ops_summary.sql`; tables `incidents`, `incident_updates`, `background_jobs`, `notification_outbox`, `webhook_deliveries`, `audit_logs`.
  - Trend blocks (`incident trend`, `job failure trend`, `webhook retry trend`) -> `ListOpsIncidents`, `ListOpsJobFailures`, `ListOpsWebhookFailures`, `parseListOps*` + `list_ops_incidents.sql`, `list_ops_job_failures.sql`, `list_ops_webhook_failures.sql`.
  - Lists (`audit feed`, `job queue`, `webhook incident context`) -> `ListOpsAuditFeed`, `parseListOps*` + `list_ops_audit_feed.sql`, `list_ops_jobs.sql`, `list_ops_webhook_failures.sql`.
  - Settings forms (`site config`, `feature flags`, `retention`, `webhook behavior`, `integration config`) -> `SetSiteConfig`, `SetFeatureFlag`, `SetRetentionPolicy`, `SetWebhookBehavior`, `SetIntegrationConfig`, `parseStoreOps*` + `upsert_site_config*.sql`, `upsert_feature_flags.sql`, `upsert_retention_policy.sql`, `upsert_webhook_behavior.sql`.

### Dashboard permissions model (planned)

| Role | Surface access | List/detail scope | Settings/mutation scope |
| --- | --- | --- | --- |
| Normal user | No dashboard surfaces | No admin list or detail access | No dashboard settings/mutations |
| Workspace admin | `Customers` (workspace-scoped), selected `Chats` operational views for their workspace | Only users/workspaces/tickets/chats tied to active admin workspace memberships | May run workspace-scoped customer/support operations (for example disable non-superuser users in-scope, suspend in-scope workspaces, support escalation), never global pricing/provider/site controls |
| Superuser | Full `Business`, `Customers`, `Chats`, `Providers`, `Ops` surfaces | Platform-wide lists and detail views with role-appropriate redaction | Full mutation rights for pricing, provider, and ops controls with confirmation/reason/fresh-session requirements for sensitive actions |

- Sensitive controls (pricing policy changes, provider routing limits, site config, retention, incident state transitions) require explicit confirmation and audit logging even for superusers.
- Workspace-admin access must fail closed when workspace membership is missing, disabled, or suspended.
- Detail views must apply minimum-field redaction by role even when list access is allowed.

### Dashboard permissions model

| Role | Visible surfaces | Allowed list/detail scope | Allowed settings and mutations |
| --- | --- | --- | --- |
| Normal user | No dashboard/admin surfaces. | None. | None. |
| Workspace admin | Workspace-scoped customer/chat/workspace slices only. No platform-wide `Business`, `Providers`, or global `Ops` surfaces. | Workspace/member records tied to authorized workspace ids only; cross-workspace rows must be denied or redacted. | Workspace-scoped operational actions only (for example user/workspace interventions within scope), with required confirmation/reason payloads. |
| Superuser | Full `Business`, `Customers`, `Chats`, `Providers`, and `Ops` surfaces. | Platform-wide list/detail access with privacy/redaction rules for sensitive payload fields. | Full control-plane and policy mutation access, with fail-closed authz and immutable audit logging. |

Permission enforcement rules:
- UI discoverability must match role scope, but RPC authz is the source of truth and must fail closed on deep links.
- Detail drawers/routes must re-check scope before returning payloads.
- High-impact mutations require confirmation + reason capture and must emit audit events.

## State and feedback rules

- Route, auth, and role state should always have explicit loading, empty, denied, and error variants.
- First-chat path must expose actionable diagnostics for bootstrap, send, and stream failures.
- Admin actions must surface pending, success, denial, and rollback outcomes explicitly.
- Token/cost provenance should remain traceable from stream completion metadata into usage ledger records.

## Design constraints

- Preserve shell-first SPA behavior with server-delivered bootstrap and WASM client routing.
- Keep docs, flows, smoke checklists, and schema wiring references synchronized.
- Avoid introducing IA that conflicts with existing RPC authz boundaries.
- Favor incremental, test-backed UI and IA changes over broad unvalidated redesign sweeps.

## Material changes from original design pass

- Reframed from a visual-style rewrite spec to a product-surface and IA intent document.
- Added explicit route-class and role-split guidance tied to the current runtime.
- Added dashboard-direction guidance based on shipped superuser authz enforcement.
- Added continuity requirements for list state, mutation confirmation, and fail-closed auth behavior.
- Added diagnostics and billing-traceability expectations as first-class UX requirements.

## Related docs

- `README.md` for setup, architecture, and route/runtime reference.
- `FLOWS.md` for user/admin journey definitions and bug-fix workflows.
- `MANUAL_SMOKE.md` for release-gate verification.
- `SCHEMA_TABLES.md` for table wiring status and integration class.
- `TODO.md` for active implementation backlog.
