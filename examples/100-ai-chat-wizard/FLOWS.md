# Example 100 Product Flows

## Visit-To-First-Chat (Current Runtime)

This flow describes the current public-to-authenticated chat journey in product terms, with concrete route, default-state, and backend-call checkpoints.

| Stage | Route entry | Product surface and defaults | Backend calls and dependencies |
| --- | --- | --- | --- |
| 1. Public entry | `/`, `/home`, `/capabilities`, `/pricing`, `/signup` | Public marketing shell renders. Auth starts in login mode by default; `/signup` forces signup mode. | Static shell delivery only (`/chat-bootstrap.js`, `/app/chat.wasm`, `/worker/background-worker.wasm`). |
| 2. Auth submit | `/` or `/signup` | Auth form requires email and password (signup also supports display name). | `Signup` or `Login` RPC returns auth token and profile identity. |
| 3. Auth resolve and runtime connect | `/app` (post-auth redirect) | Boot shell transitions into the authenticated app shell once gRPC and worker runtime become ready. | gRPC tunnel connects to `/socket`; `GetSession` validates stored token; `RefreshSession` runs on activity/focus/visibility to keep long-lived sessions valid. |
| 4. Authenticated shell bootstrap | `/app` or `/app/thread/:publicID` | Initial defaults before server hydration: model empty (resolved by catalog), tone `balanced`, thinking enabled `true`, thinking effort `medium`, TTS provider `openai`. | `ListModelOptions`, `GetSelectedModel`, `GetSelectedTone`, `GetSelectedThinkingEnabled`, `GetSelectedThinkingEffort`, `GetCustomSystemPrompt`, `GetUserName`, `ListUserMemories`, and `ListConversations`. |
| 5. Thread route resolution | `/app/thread/:publicID` | If the route targets an existing thread, the shell loads that conversation and normalizes state. | `ResolveConversationRoute` then `LoadConversation` (when accessible). |
| 6. First prompt send and stream | `/app` or active thread route | User submits first prompt from composer; assistant stream starts immediately in-thread. | `Send` streaming RPC with `history`, `message`, `model`, `tone`, `thinking_enabled`, `thinking_effort`, and `conversation_id`. Server creates or resumes conversation rows, saves messages, and writes metering records to `usage_events`. |
| 7. Reply completion and canonical route | `/app/thread/:publicID` | On stream completion, active thread becomes canonical and route normalizes to the stable public thread id. | Final `ChatChunk` carries metering fields (`prompt_tokens`, `completion_tokens`, `provider_id`, `total_cost_usd`, `usage_source`, `usage_event_id`, `usage_persisted`) for billing traceability. |
| 8. Reopen after refresh | `/app` or `/app/thread/:publicID` | Rehydrated shell restores authenticated state, conversations, and active-thread context. | `GetSession` re-check plus the same bootstrap reads (`ListConversations`, settings/profile/model reads, and `LoadConversation` for routed thread). |

## Token Traceability Notes (Billing)

- Every successful streamed completion can emit one immutable `usage_event_id` in the final `ChatChunk`.
- The same completion carries provider, prompt-token, completion-token, and total-cost fields needed to attribute cost by provider and model.
- The canonical server ledger is the `usage_events` table; client-visible chunk fields are the transport handoff for observability and UX.

## Admin-Dashboard Journey (Current Runtime)

This flow documents how operators enter and use admin data surfaces today, and where role-split behavior is expected next.

| Stage | Route entry points | Role split behavior | Data dependencies and backend calls | Expected operator action |
| --- | --- | --- | --- | --- |
| 1. Public entry | `/`, `/home`, `/pricing` | All users start as public visitors. | Static shell delivery only. | Navigate to login. |
| 2. Auth handoff | `/` or `/signup` -> `/app` | Authenticated identity is established; dashboard-capable users can navigate into dedicated `/app/dashboard*` routes. | `Login` or `Signup`, then `GetSession` during shell boot. | Enter authenticated shell. |
| 3. Admin eligibility resolution | `/app` | Current enforced split is API-level: workspace-admin and superuser scope are resolved at runtime, with superuser required for global and high-risk surfaces. | `parseRequireAdminSliceScope` enforces slice-level scope; superuser guard remains on `GetLogTail`, `GetSuperuserControlPlane`, and `GetSuperuserSlices`. | Confirm whether the actor should be scoped workspace admin or platform superuser before evaluating expected access. |
| 4. Dashboard-home data fetch | `/app/dashboard` and slice routes under `/app/dashboard/*`. | Superuser only. | `GetAdminDashboard` returns summary, usage series, provider/model/user leaderboards, recent usage events, recent users, recent conversations, and provider health/rate-limit snapshots. Backed by `usage_events`, `users`, `conversations`, `messages`, and provider runtime health. | Check global health, usage, and risk indicators before acting. |
| 5. Slice drill-down | Same admin surface | Workspace-admin has scoped `Customers`/`Chats` drill-down access; superuser has full cross-surface access. | `ListAdminUsers`, `ListAdminUsageEvents`, and `ListAdminConversations` provide scoped list/detail reads; superuser-only control-plane context remains in `GetSuperuserControlPlane`/`GetSuperuserSlices`/`GetLogTail`. | Drill into scoped detail first, then escalate to superuser-only surfaces when platform-wide context is required. |
| 6. Ongoing observability | Same admin surface | Superuser only. | `GetLogTail` for server/client diagnostics, plus repeated dashboard/list RPC refreshes as needed. | Verify impact and monitor for regressions after operator decisions. |

### Role-Split Gap (Explicit)

- Product target: normal user, workspace admin, and superuser should have distinct dashboard entry points and scoped data.
- Current implementation in example 100: workspace-admin scoped slice access exists for `Customers`/`Chats`, while global home/business/providers/ops and superuser control-plane endpoints remain superuser-gated.

## Admin Operational Workflows (Product Definition)

These workflows define intended operator behavior and data contracts. Example 100 currently exposes read-heavy admin/superuser RPCs; most mutating admin RPCs in this section are still pending implementation.

| Workflow | Entry and actor | Required data dependencies | Backend calls (current and target) | Expected result |
| --- | --- | --- | --- | --- |
| Disable or restore user | Superuser enters admin surface from `/app` | `users`, `auth_sessions`, `auth_token_versions`, `usage_events`, `billing_*`, `support_tickets`, `audit_logs` | Current: `GetSuperuserControlPlane`, `ListAdminUsers`, `ListAdminUsageEvents` for context reads. Target: typed user mutation RPCs for disable/restore + session/token revocation writes. | User access state changes with immediate session invalidation and auditable reason trail. |
| Suspend or restore workspace | Workspace admin or superuser opens workspace slice | `workspaces`, `workspace_memberships`, `api_keys`, `webhook_endpoints`, `webhook_deliveries`, `audit_logs`, `billing_*` | Current: `GetSuperuserControlPlane` for workspace-scoped context reads. Target: typed workspace mutation RPCs for suspend/restore, key revocation, webhook pause/resume. | Workspace state flips safely, dependent access paths are constrained, and impact is visible. |
| Billing or entitlement intervention | Billing-capable operator enters billing controls | `billing_customers`, `billing_subscriptions`, `billing_invoices`, `billing_events`, `billing_access_overrides`, `billing_quota_policies`, `billing_upgrade_triggers`, `billing_dunning_events` | Current: aggregate signals via admin dashboard + superuser snapshot. Target: typed billing override/quota/dunning mutation RPCs. | Access and quota outcomes match operator intent, and customer billing state is reconciled. |
| Support or abuse triage | Support/risk operator opens queue and account detail | `support_tickets`, `support_ticket_messages`, `audit_logs`, `usage_events`, `users`, `workspaces` | Current: support rows visible in `GetSuperuserControlPlane`; usage context via list RPCs. Target: typed support assignment/note/escalation/account-action RPCs. | Ticket/account timeline stays queryable, with clear owner, action history, and escalation state. |
| Incident or experiment control | Reliability/product operator enters incident/flag/experiment controls | `incidents`, `incident_updates`, `feature_flags`, `experiments`, `audit_logs`, `site_config` | Current: incident/experiment read context in `GetSuperuserControlPlane`. Target: typed control-plane mutation RPCs for incident updates, flag toggles, and rollback actions. | Operational state changes are explicit, reversible, and linked to blast-radius and audit records. |

### Workflow Guardrails

- Every destructive or high-impact action should require explicit confirmation and a reason payload.
- Authorization should fail closed at RPC boundaries, not only in UI route guards.
- Mutation success criteria includes session/cache invalidation where privilege or access scope changes.
- Every workflow should emit immutable audit records tied to actor, scope, target, and timestamp.

## Dashboard Day-In-The-Life Workflows

This section defines the expected queue-to-detail-to-action loop for each dashboard surface so operators can make decisions with evidence instead of acting from summary cards only.

| Surface | Queue or summary entry point | Required detail inspection | Typical operator action | Evidence required before mutate |
| --- | --- | --- | --- | --- |
| `Business` | Revenue summary, failed-payments queue, overage and churn signals | Customer subscription/invoice timeline, billing events, dunning state, recent usage trend | Resolve failed payment, set billing override, adjust pricing/plan controls | Platform fee + usage + service premium breakdown, affected customer count, invoice-state impact, rollback path |
| `Customers` | Search/list by email, workspace key, plan, or account status | User timeline (sessions, usage, support, billing) and workspace timeline (members, keys, webhooks, audit) | Disable/restore user, suspend/restore workspace, support-linked action | Scope check (workspace vs platform), account/workspace blast radius, reason capture, recovery path |
| `Chats` | Failed or slow reply queue, thread-volume and latency anomalies | Thread inspector, event/run detail, model/provider metadata, recent anomaly pattern | Adjust defaults entry point, triage provider/model failure class, route to support/ops | Failure trend confidence, sample event IDs, affected thread count, verification query for post-action state |
| `Providers` | Provider/model health summary and cost/routing trend tables | Provider-scoped usage events, model-level drilldown, rate-limit/health snapshots, guardrail state | Toggle provider/model visibility policy, update fallback/routing policy, tighten guardrails | Affected scope count, fallback path viability, projected cost/latency impact, rollback steps |
| `Ops` | Incident queue, failed jobs, notification/webhook reliability summary | Incident detail, blast-radius preview, queue detail rows, recent audit actions | Retry/replay/incident status update, flag/experiment rollback entry | Blast radius preview, queue-size impact, audit note, clear post-action validation query |

### Dashboard Decision Points

1. Confirm the route/query state captures the exact queue context you intend to operate on.
2. Open detail from that queue context and verify identifiers match the summary signal.
3. Validate blast radius and affected-record counts before any mutating action.
4. Capture one explicit reason and expected outcome before submit.
5. Return to the same filtered queue and confirm the action produced the expected state change.

## Public-Route Bug-Fix Workflow

Use this sequence before editing code when a public-route regression is reported.

1. Reproduce on one concrete path (`/`, `/home`, `/capabilities`, `/pricing`, `/signup`, or `/app` when unauthenticated).
2. Record request outcome (`status`, final URL after redirects, and whether shell HTML loaded).
3. Capture hydration evidence:
   - boot shell visible vs hidden
   - whether app root mounted
   - whether expected route view rendered
4. Capture router-state evidence:
   - browser location path/query/hash
   - expected route classification (public, app, settings, asset)
   - any route normalization or redirect behavior
5. Capture runtime diagnostics before edits:
   - browser console errors/warnings
   - page errors/unhandled rejections
   - server log lines around request and startup
6. Isolate root cause bucket before patching: shell-delivery mismatch, route classification logic, auth gate behavior, or hydration/runtime boot failure.
7. Apply smallest root-cause fix and confirm the same route reproducer now passes.
8. Lock regression with the narrowest route-level browser test and update manual smoke notes if behavior changed.

## First-Chat Bug-Fix Workflow

Use this sequence when regressions happen between auth completion and first streamed assistant reply.

1. Reproduce with one deterministic account and path (`/` or `/signup` -> `/app`).
2. Capture auth-bootstrap state before send:
   - token present vs missing
   - `GetSession` success/failure behavior
   - authenticated shell ready vs half-booted shell
3. Capture model/bootstrap state before send:
   - selected model/tone/thinking values
   - `ListModelOptions` readiness and default-model resolution
   - conversation list readiness (`ListConversations`)
4. Capture send-flow inputs and outcomes:
   - composer input state and submit event
   - `Send` request context (conversation id, model, tone, thinking flags)
   - first stream chunk arrival timing and any stream errors
5. Capture route and thread normalization state:
   - active route before send vs after reply
   - resolved `conversation_id`/`public_id` mapping
   - whether URL normalizes to `/app/thread/:publicID`
6. Capture token-cost and provider evidence from final stream chunk:
   - `provider_id`, `usage_event_id`, prompt/completion tokens, total cost fields
7. Identify failing seam before patching: auth/session invalidation, catalog/bootstrap drift, send RPC failure, stream consumption failure, or route-sync mismatch.
8. Ship smallest fix, then verify login -> first send -> streamed reply -> refresh/reopen with targeted browser coverage.

## Admin-Dashboard Bug-Fix Workflow

Use this sequence when admin access, dashboard data slices, or operator actions regress.

1. Reproduce with explicit actor role context (normal user, workspace admin, superuser).
2. Capture dashboard entry conditions:
   - starting route and auth state
   - expected admin entry affordance present/absent
   - resulting route/state after attempted entry
3. Capture role-scope evidence before edits:
   - authenticated user id/email
   - role grants (`su_user_roles` and any workspace-scoped role context)
   - expected allowed vs denied surfaces for that role
4. Capture slice-load behavior:
   - which dashboard slice request failed or returned unexpected shape
   - RPC status code (`Unauthenticated`, `PermissionDenied`, `Internal`, etc.)
   - empty/error state payload details
5. Capture mutation-state evidence (when applicable):
   - confirmation payload, reason fields, and target identifiers
   - submit result and rollback/error state
   - post-mutation data refresh and access-state invalidation behavior
6. Collect diagnostics from both sides:
   - browser console/page errors
   - server logs for admin RPC guard and store-query failures
   - audit/event evidence if mutation succeeded
7. Classify root cause before patching: role guard mismatch, route-entry mismatch, slice query defect, mutation contract defect, or stale client state.
8. Ship smallest role-safe fix and re-run role-specific regressions for denied and allowed paths.

## Admin List Conventions

Use these conventions for operator-facing admin lists (users, workspaces, tickets, usage events, and similar large slices).

### Search

- Search should be explicit and stable: one clear query input, deterministic matching order, and predictable empty-state messaging.
- Supported searchable identifiers should be documented per slice (for example: email, workspace key, ticket key, conversation public id).
- Search queries should be reflected in route/query params when practical so reload/back preserves operator intent.

### Filter

- Filters should use bounded enums or typed ranges, not free-form text where cardinality is known.
- Active filters should be visibly listed and individually clearable.
- Clearing filters must not clear an active search term unless explicitly requested.
- Denied/unsupported filter combinations should fail with actionable operator feedback, not silent no-op behavior.

### Pagination

- List pagination must be deterministic (stable sort + explicit cursor/page boundary) so operators can safely go back and forward.
- Pagination state (page/cursor/page-size) should remain recoverable across refresh/back for long triage sessions.
- Empty pages after deletions or filter narrowing should auto-recover to the nearest valid page boundary.
- List response contracts should always return enough metadata for the UI to render previous/next affordances safely.

## Disable vs Suspend Semantics

Use these distinctions consistently so operator actions have predictable blast radius.

| Action | Primary target | Auth impact | Chat/runtime impact | API/webhook impact | Billing/support impact |
|---|---|---|---|---|---|
| Disable user | One user identity | Revoke active user sessions; block new login for that user. | User cannot create/send/load personal conversations while disabled. | User-scoped API credentials and delegated access should be treated as inactive. | Preserve billing/support history; future support actions should show account as disabled. |
| Restore user | One user identity | Allow new login; issue fresh sessions under current token-version policy. | User regains normal conversation access under current entitlement rules. | User-scoped API credentials/access can be re-enabled per policy. | Billing/support history remains intact; restore state should be auditable. |
| Suspend workspace | One workspace boundary | Workspace-member auth may still exist, but workspace-scoped actions are denied. | Workspace-scoped chats/automation should fail closed until restored. | Workspace API keys/webhooks should stop processing deliveries and new calls. | Workspace billing/support records remain readable; workspace status marked suspended. |
| Restore workspace | One workspace boundary | Workspace-scoped authorization resumes per current membership roles. | Workspace chat/automation surfaces become available again. | Workspace keys/webhooks can resume according to restore policy. | Billing/support continuity preserved; restore event must be auditable. |

### Guardrail summary

- `disable` is identity-scoped; `suspend` is workspace-scoped.
- Both actions should emit immutable audit records with actor, reason, target, and timestamp.
- Restore flows should not silently clear historical billing/support/usage evidence.
