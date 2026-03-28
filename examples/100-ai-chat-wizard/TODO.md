# Example 100 Todo

## Current Backlog

## Visitor-To-First-Chat Flow Story

1. Visitor lands on `/`, `/home`, `/pricing`, or `/signup`.
2. Visitor sees the boot shell briefly, then the correct public route hydrates.
3. Visitor scans the product value prop, pricing, and trust signals.
4. Visitor chooses a primary CTA and enters the auth flow.
5. Visitor completes login or signup successfully.
6. Client establishes the authenticated app shell, gRPC tunnel, and worker/runtime readiness.
7. Client loads profile, settings defaults, model catalog, and conversation list.
8. User lands in a clear first-run or empty-thread state with obvious next actions.
9. User writes the first prompt and sends it.
10. Server creates or resolves the conversation, applies defaults, and streams the reply.
11. Client normalizes to the canonical thread route, keeps scroll anchored correctly, and shows the completed first reply.
12. User can continue the thread, reopen it later, and understand what to do next.

## Admin-From-Homepage-To-Dashboard Flow Story

1. Admin lands on `/`, `/home`, or `/pricing`.
2. Admin sees the public marketing shell and chooses `Log in`.
3. Admin completes login successfully.
4. Client resolves auth and boots the authenticated shell.
5. Client resolves role and permission state for normal user, workspace admin, or superuser.
6. App exposes the correct admin entry point based on that role.
7. Admin opens dashboard home.
8. Dashboard home loads summary cards for operational health, users, usage, billing, support, and incidents.
9. Admin chooses a dashboard slice such as workspaces, support, billing, incidents, analytics, or experiments.
10. Client requests the typed admin or `su` data needed for that slice.
11. Server authorizes the request and returns the correct scope of records.
12. Admin drills into detail views, recent events, and actionable controls.
13. Admin takes a management action or exits back to dashboard home.

## Admin Operational Workflow Stories

### Disable-Or-Restore-User Flow

1. Admin opens dashboard home and navigates to users.
2. Admin searches for a user by email, name, workspace, or recent activity.
3. Admin opens the user detail view and reviews profile, usage, billing state, support history, and recent audit events.
4. Admin chooses `Disable user` or `Restore user`.
5. App requires confirmation and shows the scope of impact before submit.
6. Server applies the status change, revokes active sessions if needed, and records an audit event.
7. Dashboard updates the user state and shows the result clearly.

### Suspend-Or-Restore-Workspace Flow

1. Admin opens the workspace slice from dashboard home.
2. Admin searches for a workspace and opens its detail view.
3. Admin reviews members, API keys, webhooks, usage, billing state, and incidents tied to that workspace.
4. Admin chooses `Suspend workspace` or `Restore workspace`.
5. Server applies the change, revokes dependent access if needed, and records an audit trail.
6. Dashboard updates the workspace state and shows the operational impact.

### Billing-Or-Entitlement-Intervention Flow

1. Admin opens billing or quota controls from dashboard home.
2. Admin searches for a user or workspace with a billing or access issue.
3. Admin reviews plan, usage, overages, failed payments, overrides, and recent billing events.
4. Admin applies an override, changes a quota, or resolves a failed-payment state.
5. Server records the change and updates the effective entitlement state.
6. Dashboard confirms the updated billing and access outcome.

### Support-Or-Abuse-Triage Flow

1. Admin opens support or risk queues from dashboard home.
2. Admin filters tickets or flagged accounts by severity, age, workspace, or abuse signal.
3. Admin opens the detail view and reviews messages, recent activity, audit events, and relevant usage data.
4. Admin takes an action such as reply, escalate, disable access, or attach an internal note.
5. Server records the action and keeps the ticket or account history queryable.

### Incident-Or-Experiment-Control Flow

1. Admin opens incidents, feature flags, or experiments from dashboard home.
2. Admin reviews current health, blast radius, recent changes, and affected workspaces or users.
3. Admin changes an incident state, posts an update, toggles a flag, or rolls back an experiment.
4. Server applies the control-plane change and records the operator action.
5. Dashboard reflects the new state and any downstream impact.

## Bug-Fix-And-Testing Stories

### Public-Route Regression Bug-Fix Flow

1. Maintainer starts from a concrete broken route or navigation report.
2. Maintainer reproduces the failure on the smallest public route flow possible.
3. Maintainer captures the exact route, console error, UI state, and expected result.
4. Maintainer identifies whether the break is in server delivery, router state, hydration, or route-specific UI logic.
5. Maintainer lands the smallest root-cause fix.
6. Maintainer adds or updates the narrowest regression test and manual smoke note for that route.

### First-Chat Regression Bug-Fix Flow

1. Maintainer reproduces the failure from login or signup through first message send.
2. Maintainer captures auth state, selected model state, route state, streaming behavior, and final thread state.
3. Maintainer identifies whether the break is in auth bootstrap, catalog bootstrap, composer state, send flow, or route normalization.
4. Maintainer lands the smallest fix that preserves the happy path.
5. Maintainer updates the focused test and the first-chat smoke checklist.

### Admin-Dashboard Regression Bug-Fix Flow

1. Maintainer reproduces the issue from homepage through dashboard entry or operator action.
2. Maintainer captures role state, route state, denied vs allowed behavior, loaded data, and the visible UI error.
3. Maintainer identifies whether the break is in authz, typed RPC data, dashboard state management, or mutation UX.
4. Maintainer lands the smallest fix and verifies the role boundaries remain correct.
5. Maintainer updates regression coverage and the admin smoke checklist for that workflow.

### Release-Smoke Testing Flow

1. Maintainer runs the visitor flow, first-chat flow, and admin flow before marking the slice ready.
2. Maintainer records pass or fail status, known gaps, and any temporary manual checks still required.
3. Maintainer updates the operator notes so the next pass can reproduce the same checks quickly.

## Future Chat Capability Stories

### Calendar-And-Email Hooks

1. User connects calendar and email providers from a controlled integration surface.
2. Chat can read calendar availability, upcoming events, inbox summaries, and selected message threads with explicit scope grants.
3. Chat can draft emails, propose meeting slots, create calendar events, and update invites through typed tools.
4. Sensitive actions require explicit approval before send, create, update, or delete operations execute.
5. Admin and audit surfaces record which external provider action ran, for which user or workspace, and with what result.

### Web Search With Citations

1. User can run a chat with web search enabled for current information.
2. The model can issue search requests, fetch result content, and synthesize a grounded answer.
3. Responses show source citations clearly enough for verification.
4. Search usage, cost, and provider/tool traces remain visible in message metadata and admin reporting.
5. Users can rerun a question without web search to compare grounded vs model-only answers.

### Team Comments And Shareable Chats

1. Users can share a thread with teammates or generate a read-only share link.
2. Teammates can comment on a full thread or a specific message.
3. Shared threads preserve citations, artifacts, and key metadata needed for context.
4. Permission controls distinguish private, workspace-shared, commentable, and read-only states.
5. Admin surfaces can inspect share state and collaboration activity when needed.

### Scheduled Jobs

1. Users can schedule recurring or one-off jobs from a thread, workflow, or prompt template.
2. Scheduled jobs can run web search, docs queries, summaries, or custom workflows in the background.
3. Job history shows next run, last run, success or failure, output summary, and any approval or auth issues.
4. Users can pause, resume, edit, or cancel a scheduled job safely.
5. Admin and ops surfaces can inspect queue state, failures, retries, and disabled integrations.

### Image Upload And Image-Aware Chat

1. Users can upload one or more images into a thread.
2. Chat can describe, OCR, compare, summarize, and answer questions about uploaded images.
3. Image uploads participate in citations or source references where applicable.
4. The UI distinguishes text-only, image-grounded, and mixed-input replies clearly.
5. Storage, retention, and access rules for uploaded images are explicit and auditable.

### Ask-With-Docs Knowledge Mode

1. Users can upload docs or connect workspace document sources to a searchable knowledge layer.
2. Documents are chunked, embedded, indexed, and retrievable for chat grounding.
3. Users can run a thread in “ask with docs” mode and receive answers grounded in retrieved passages.
4. Responses show which files and chunks were used so the result is verifiable.
5. The backend supports document ingestion, vector search or equivalent retrieval, reindexing, and permission-aware access.

### Skills

1. Users or admins can define reusable “skills” that package prompts, tool availability, docs scope, and execution rules.
2. A thread can start with a chosen skill or switch into one when appropriate.
3. Skills can be workspace-scoped or user-scoped with clear ownership and versioning.
4. The UI explains what a skill changes: tools, docs, response style, and allowed actions.
5. Admin surfaces can inspect skill usage and disable broken or unsafe skills.

### Custom Workflows

1. Users can assemble repeatable multi-step workflows instead of retyping the same prompt sequence.
2. Workflows can combine prompt steps, doc retrieval, web search, and custom tool actions.
3. Workflows can save inputs, outputs, approvals, and failure states between runs.
4. Users can clone, edit, share, and schedule workflows.
5. Admin and audit surfaces can inspect workflow runs, failures, cost, and sensitive action history.

### Code Interpreter Sandbox

1. Users can run code-backed analysis tasks inside a constrained execution sandbox.
2. The sandbox can handle CSVs, tabular analysis, lightweight Python transforms, and chart generation.
3. Outputs can return as files, tables, charts, or inline artifacts attached to the thread.
4. Resource limits, file limits, runtime limits, and security boundaries are explicit.
5. Admin surfaces can inspect sandbox usage, failures, and cost or runtime trends.

## Core Dashboard Scope

### Business Surface

1. Admin sees only the core business metrics needed to run pricing and revenue operations: active paid subscriptions, failed payments, free-to-paid conversion, first-chat conversion, churn count, and top revenue accounts.
2. Admin can adjust the minimum useful business settings from one place: plans, quotas, overage rules, upgrade triggers, and dunning controls.
3. Dashboard data is backed by typed summary RPCs, typed SQL queries, and the existing billing, usage, analytics, and churn tables before any new rollup tables are introduced.

### Customers Surface

1. Admin can search users and workspaces by email, name, plan, status, or recent activity.
2. Admin can inspect only the core customer state: profile, workspace membership, auth/session status, billing state, recent chats, support state, and disable or suspend state.
3. Dashboard data is backed by typed detail and list RPCs, typed SQL queries, and the existing user, workspace, auth, usage, support, and billing tables.

### Chats Surface

1. Admin can see only the core product health metrics: threads started, messages sent, replies completed, failed replies, response time, and usage of docs, web, tools, workflows, and sharing.
2. Admin can inspect the minimum useful chat settings: default system prompt, default model and reasoning defaults, onboarding templates, memory rules, and workflow or skill publishing state.
3. Dashboard data is backed by typed chat summary RPCs, typed SQL queries, and the existing conversation, message, usage, onboarding, workflow, prompt-library, and analytics tables.

### Providers Surface

1. Admin can compare providers and models by uptime, latency, error rate, timeout rate, fallback rate, and cost.
2. Admin can adjust only the core provider settings: provider enable or disable, model visibility, default or fallback routing, per-provider limits, and cost guardrails.
3. Dashboard data is backed by typed provider summary RPCs, typed SQL queries, and the existing model-catalog, usage, routing-policy, and cost-guardrail tables, plus new persisted provider-health tables only if the current runtime data is not durable enough.

### Ops Surface

1. Admin can monitor only the core operational state: server health, gRPC health, background jobs, notification failures, webhook failures, incidents, SLOs, and recent admin actions.
2. Admin can adjust only the core site settings: site config, feature flags, retention policies, webhook behavior, integration settings, and superuser controls.
3. Dashboard data is backed by typed ops summary RPCs, typed SQL queries, and the existing site-config, feature-flag, audit-log, incident, job, notification, and webhook tables, plus new persisted health-sample tables only if the current metrics cannot be queried reliably.

### Agent 1: Runtime, routing, and regression coverage

- [x] Add one browser-level dashboard-home regression that covers the `Business`, `Customers`, `Chats`, `Providers`, and `Ops` surfaces, verifying tab or route entry, KPI-card load, table load, time-range changes, and back or refresh stability.
	Added `TestExample100DashboardHomeRegression` (`test/playwrightgo/examples/example100_dashboard_home_pending_test.go`) to verify superuser dashboard-home coverage across `Business`, `Customers`, `Chats`, `Providers`, and `Ops` route/deep-link entry, shared `lookback_days` time-range transitions, KPI/table RPC loads (`GetAdminDashboard`, `ListAdminUsers`, `ListAdminConversations`, `ListAdminBillingAccessOverrides`, `GetSuperuserSlices`), and refresh/back/forward URL-state stability.
- [x] Add focused regressions for dashboard list mechanics so shared time range, search, filter, sort, pagination, and drill-down state remain stable across surface switches and direct deep links.
	Extended `TestExample100AdminListMechanicsRegression` (`test/playwrightgo/examples/example100_admin_list_mechanics_pending_test.go`) to carry one shared `lookback_days` time-range parameter plus search/filter/sort/pagination query state across user/workspace/support/incident/billing deep links, with refresh/back/forward stability assertions.
- [x] Add focused regressions for dashboard settings mutations so plan or quota edits, provider toggles, site-config updates, and incident or feature-flag changes update the visible surface without stale cards or stale tables.
	Replaced the pending mutation diagnostics scaffold with `TestExample100DashboardSettingsMutationsRegression` (`test/playwrightgo/examples/example100_admin_mutation_diagnostics_pending_test.go`), which executes billing plan/quota, feature-flag, and incident-status mutations, verifies post-mutation reads via `GetSuperuserSlices` + `GetSuperuserControlPlane`, and asserts refresh/back/forward route-state stability to catch stale-surface behavior.
- [x] Add focused regressions for admin detail drawers and detail routes so customer, chat, provider, and ops drill-down views preserve query state and return to the correct list context.
	Added `TestExample100AdminDetailRoutesRegression` (`test/playwrightgo/examples/example100_admin_detail_routes_pending_test.go`) with customer/chat/provider/ops drill-down RPC checks (`GetAdminUserDetail`, conversation/provider-filtered drill-down lists, `GetAdminSupportTicketDetail`, `GetAdminIncidentBlastRadius`) plus list->detail deep-link route-state restoration across refresh/back/forward navigation.
- [x] Add runtime diagnostics for dashboard summary fetches, table fetches, settings fetches, and drill-down fetches so each surface logs one actionable warning on empty, partial, slow, denied, or failed loads.
	Added shared fetch-outcome diagnostics in `server/app/admin_dashboard.go` and applied them across summary/table/drill-down RPCs (`GetAdminDashboard`, `ListAdminUsers`, `ListAdminUsageEvents`, `ListAdminConversations`, `GetAdminUserDetail`), plus settings/drill-down diagnostics in `server/app/superuser_control.go` (`GetSuperuserControlPlane`) and `server/app/admin_support_ops.go` (`GetAdminSupportTicketDetail`) to emit actionable warnings for empty/partial/slow outcomes while preserving existing denied/failed auth and error logging.
- [x] Add a browser-level dashboard smoke that verifies every chart or trend widget has a corresponding drill-down table or detail surface instead of a dead-end visualization.
	Added `TestExample100DashboardDrilldownSmoke` (`test/playwrightgo/examples/example100_dashboard_drilldown_smoke_pending_test.go`) to assert dashboard trend/KPI payloads are non-empty and each has a drill-down path via typed table/detail RPCs (`ListAdminUsageEvents`, `GetAdminUserDetail`, `ListAdminConversations`) plus dashboard route-entry smoke coverage.
- [x] Add a `Business` workflow regression that covers revenue summary -> failed-payments queue -> customer subscription detail -> resolve or override action -> return to the same filtered queue state.
	Added `TestExample100AdminBusinessWorkflowRegression` (`test/playwrightgo/examples/example100_admin_business_workflow_pending_test.go`) using seeded admin-ops billing fixtures to validate revenue-summary pressure, failed-payment queue load, customer billing detail read, override + resolve actions, pending-queue clearance, and filtered billing-route context restoration across detail navigation.
- [x] Add a `Customers` workflow regression that covers search -> user or workspace detail -> unified account timeline -> disable or suspend action -> restore action -> return to the same filtered list state.
	Added `TestExample100AdminCustomersWorkflowRegression` (`test/playwrightgo/examples/example100_admin_customers_workflow_pending_test.go`) to cover filtered user search, user/workspace detail timeline reads, disable/restore user and suspend/restore workspace mutations, post-action detail state verification, and filtered list/detail route-context restoration.
- [x] Add a `Chats` workflow regression that covers failed-reply or slow-reply summary -> thread inspector -> message or run detail -> default-setting adjustment entry point -> return to the same anomaly queue state.
	Added `TestExample100AdminChatsWorkflowRegression` (`test/playwrightgo/examples/example100_admin_chats_workflow_pending_test.go`) to cover failed-or-fallback anomaly queue summary, thread inspector and run-detail drill-down reads, settings-entry model default adjustment (`SetSelectedModel`/`GetSelectedModel`), and anomaly queue route-state restoration.
- [x] Add a `Providers` workflow regression that covers provider health summary -> model drill-down -> fallback or visibility change -> blast-radius preview -> summary refresh with preserved time range and filters.
	Added `TestExample100AdminProvidersWorkflowRegression` (`test/playwrightgo/examples/example100_admin_providers_workflow_pending_test.go`) to validate provider/model summary and drill-down usage reads, provider-policy mutation entry (`SetAdminFeatureFlag`), blast-radius preview (`GetAdminIncidentBlastRadius`), summary refresh, and provider-filter/time-range route-context preservation.
- [x] Add an `Ops` workflow regression that covers incident or failed-jobs summary -> queue detail -> retry or replay action -> audit-feed confirmation -> return to the same queue and time-range context.
	Added `TestExample100AdminOpsWorkflowRegression` (`test/playwrightgo/examples/example100_admin_ops_workflow_pending_test.go`) to validate incident-queue summary, blast-radius queue detail, replay-style incident update action, audit-feed confirmation (`admin.control.incident.update`), and queue/time-range route-context restoration.
- [x] Add one browser-level visit-to-first-chat regression that covers landing load, pricing/auth navigation, login/signup handoff, authenticated shell boot, first send, streamed reply, and canonical thread-route normalization.
	Added `TestExample100VisitToFirstChatRegression` (`test/playwrightgo/examples/example100_visit_first_chat_test.go`) to cover `/`, `/pricing#faq`, `/signup`, auth handoff back to `/app`, first send/stream, and canonical `/app/thread/:publicID` normalization with fallback reopen via sidebar.
- [x] Add runtime diagnostics for the visit-to-first-chat path so boot, gRPC connect, worker readiness, model-catalog load, conversation bootstrap, and first-send failures log one actionable warning or error each.
	Added explicit diagnostics for delayed boot-shell teardown (`boot shell still mounted after startup window`), model-catalog load skip/failure/success (`model catalog refresh skipped`, `model catalog load failed`, `model catalog loaded`), and conversation bootstrap load/failure (`conversation bootstrap loaded`, `conversation bootstrap failed`) while preserving existing gRPC, worker, and send/stream failure logs.
- [x] Add focused regression coverage for empty-state boot, first-thread creation, scroll anchoring during first reply, and reopen-after-refresh behavior.
	Validated `TestExample100AuthenticatedHappyPath` (`test/playwrightgo/examples/example100_authenticated_happy_path_test.go`) as the focused browser regression covering empty-state boot, first-thread send/route creation, in-thread first-reply stream anchoring, and reopen-after-refresh persistence.
- [x] Add one browser-level admin journey regression that covers homepage load, login, role resolution, admin-entry visibility, dashboard-home load, slice navigation, and back/refresh behavior.
	Added `TestExample100AdminJourneyRegression` (`test/playwrightgo/examples/example100_admin_journey_test.go`) to cover homepage boot, role-elevated admin-journey login (`demo@example.com` + seeded `su` role), dashboard deep-link entry visibility (`/app/dashboard`), slice route navigation (`/app/dashboard/usage`, `/app/admin/users`), back+refresh stability, and role-scoped dashboard/slice RPC checks (`GetAdminDashboard`, `ListAdminUsers`, `ListAdminConversations`) using the authenticated browser token.
- [x] Add runtime diagnostics for the admin dashboard journey so role resolution, admin route entry, dashboard bootstrap, slice fetches, and unauthorized transitions emit actionable logs.
	Added actionable admin-journey diagnostics across `admin_scope.go`, `admin_dashboard.go`, and HTTP deep-link guarding in `server.go`: role-resolution allow/deny logs (`rpc.admin role resolution complete/denied`), slice denial logs (`rpc.admin slice denied`), dashboard/slice bootstrap logs (`rpc.GetAdminDashboard: bootstrap`, `rpc.ListAdmin* : slice fetch`), and unauthorized-transition deep-link logs (`http.admin deep link denied` + next-action hints).
- [x] Add focused regressions for workspace-admin vs superuser route guards, direct deep links into dashboard slices, and empty/error dashboard states.
	Added `TestExample100AdminRoleGuardsAndDeepLinks` (`test/playwrightgo/examples/example100_admin_role_guard_test.go`) with focused browser + admin-RPC coverage for: superuser deep-link allow (`/app/admin/users`), workspace-admin deep-link allow with scoped empty conversation state (`/app/dashboard/usage`), and normal-user deep-link fail-closed redirect to `/app` plus permission-denied dashboard RPC checks.
- [x] Add one browser-level admin-mutation regression that covers disable user, restore user, suspend workspace, restore workspace, and the resulting UI state transitions.
	Replaced the pending scaffold with `TestExample100AdminMutationRegression` (`test/playwrightgo/examples/example100_admin_mutation_pending_test.go`), which performs a live browser flow plus superuser RPC mutations (`DisableAdminUser`, `RestoreAdminUser`, `SuspendAdminWorkspace`, `RestoreAdminWorkspace`) and asserts transition behavior between authenticated app shell and auth/public entry states across each mutation step.
- [x] Add focused regressions for billing-intervention, support-triage, and incident-control flows so admin actions survive refresh and back navigation correctly.
	Replaced the pending scaffold with `TestExample100AdminOpsRegression` (`test/playwrightgo/examples/example100_admin_ops_pending_test.go`), which seeds deterministic billing/support fixtures, executes typed operator mutations (`SetAdminBillingAccessOverride`, support note/assign/escalate, `SetSuperuserIncident` + `UpdateAdminIncident`), verifies persisted results via scoped read RPCs, and asserts admin route stability across refresh/back/forward navigation after each flow.
- [x] Add runtime diagnostics for admin mutations so confirmation, submit, success, denial, and rollback states emit actionable logs during operator workflows.
	`parseExecuteAdminMutationAction` now emits explicit operator workflow diagnostics for submit (`rpc.admin mutation submit`), confirmation (`rpc.admin mutation confirmed`), denial (`rpc.admin mutation denied` with `stage=authorize|confirmation`), success (`rpc.admin mutation success`), and rollback-required failures (`rpc.admin mutation rollback required`) with target/scope metadata and next-action hints; focused coverage added in `TestAdminMutationDiagnosticsLogs` (`server/app/admin_mutation_effects_test.go`) plus micro-benchmark `BenchmarkParseBuildAdminMutationScopeLabel`.
- [x] Add focused regressions for admin list mechanics so user, workspace, support, incident, and billing views keep search, filter, sort, and pagination state stable across refresh and back navigation.
	Replaced the pending scaffold with `TestExample100AdminListMechanicsRegression` (`test/playwrightgo/examples/example100_admin_list_mechanics_pending_test.go`), which seeds list-friendly fixtures, verifies typed `AdminListQuery` behavior for users/workspaces/support/incidents/billing (search/filter/sort/pagination), and asserts query-string list-state stability across refresh/back/forward browser navigation.
- [x] Run a real browser smoke against `http://127.0.0.1:8095/` and capture any remaining startup console/runtime errors.
	Executed a live Playwright smoke against `http://127.0.0.1:8095/` with `status=200`, title `RelayDesk - AI Chat Workspace`, and zero console/page errors.
- [x] Add a focused end-to-end regression for server start, WASM shell boot, and background worker boot.
	Added `TestExample100StartupBoot` under `test/playwrightgo/examples` to assert health check, boot-shell removal, `chat.wasm` and `background-worker.wasm` responses, `grpc ready`, `worker ready`, and worker-pool startup logs with zero console/page errors.
- [x] Verify the authenticated happy path end-to-end: login, create thread, send message, stream reply, refresh, reopen thread.
	Added `TestExample100AuthenticatedHappyPath` to seed a deterministic DB, sign in as `demo@example.com`, create/send in a new thread, wait for streamed assistant output, refresh and assert the same `/app/thread/:publicID`, then reopen the thread from the sidebar and verify prompt persistence.
- [x] Add browser-level smoke coverage for the pricing, auth, and dashboard routes.
	Added `TestExample100RouteSmokePricingAuthDashboard` to smoke `/pricing#faq`, unauthenticated `/app` auth inputs, and authenticated `/app/settings?panel=settings-profile` with zero browser console/page errors.
- [ ] Add one browser-level pricing regression that covers `/pricing` load, plan-card copy, pricing FAQ, contact CTA targets, and the absence of any stale `free`, `unlimited`, or `no token caps` language once the usage-based pricing model lands.
- [ ] Add one browser-level billing-summary regression that covers the customer-facing billing/settings surfaces and verifies platform fee, raw usage cost, service premium, and total are shown with the same labels and formulas as the backend billing model.
- [x] Trace and fix the mismatch where the model select shows `GPT-5.4 - Best` while message bubbles report `GPT-5.4 mini`.
	Stream completion now normalizes the assistant-reported reply model, syncs `SelectedModel` to that canonical model when it differs, and persists the synced value via `SetSelectedModel` so picker state stays aligned with message metadata after first reply.
- [x] Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.
	`ParseScrollToBottom` now performs a delayed settle pass that snaps the message list to exact bottom after the initial smooth jump, and `TestExample100ScrollToBottomButton` was added to assert bottom-alignment after clicking `#scroll-to-bottom-btn`.
- [x] Update the app version number and ensure the displayed version string is sourced consistently.
	Version is now `v2026.03.27.2` from one shared source (`internal/buildinfo.GetBuildAppVersion`), with both the client badge (`sidebar`) and server boot-shell version label rendering from that same source.

### Agent 2: Security, auth, and billing enforcement

- [x] Define and enforce the role and scope matrix for the five dashboard surfaces so normal users see none of them, workspace admins only see scoped customer and workspace data, and superusers see platform-wide business, provider, and ops controls.
	Added explicit admin-surface matrix enforcement in `server/app/admin_scope.go`: workspace-admin callers are now fail-closed to scoped `customers/chats` slices while `business/providers/ops` slices require superuser scope; added focused verification in `TestAdminDashboardSurfaceRoleScopeMatrix` (`server/app/admin_dashboard_test.go`) plus regression checks for workspace-admin support/billing/workspace slice access.
- [x] Add server-side authz for `Business` settings mutations so plan edits, quota-policy changes, overage-rule changes, upgrade-trigger changes, and dunning changes require the correct superuser scope and fresh-session checks where needed.
	`Set/DeleteSuperuserBilling*` mutation RPCs are now superuser-gated with session freshness (`parseRequireSuperuserMutationUserID`) and explicit confirm+reason guards (`parseRequireSuperuserBillingMutationConfirmation`) across plan overages, quota policies, upgrade triggers, and dunning controls; added focused stale-session regression in `TestSuperuserBusinessMutationsRequireFreshSession` (`server/app/superuser_pricing_ops_test.go`).
- [ ] Add server-side rules for the usage-based billing model so plan changes, pricing previews, invoice generation, and admin overrides all respect `platform fee + usage cost + service premium` instead of any stale flat-rate or unlimited assumptions.
- [ ] Add server-side rules that enforce the emotional plan boundary correctly: `Pro` stays single-operator/personal-workspace scoped, `Team` unlocks shared workspace/admin/collaboration capabilities, and `Enterprise` remains contract-gated.
- [ ] Add server-side authz for `Customers` mutations so user disable or restore, workspace suspend or restore, entitlement overrides, and billing overrides respect workspace scope, platform scope, and reason plus confirmation requirements.
- [ ] Add server-side authz for `Chats` settings mutations so system-default prompt changes, model-default changes, memory-rule changes, onboarding-template changes, and workflow or skill publishing changes respect the correct operator boundary.
- [ ] Add server-side authz for `Providers` settings mutations so provider enable or disable, model visibility, fallback routing, per-provider limits, and cost-guardrail changes fail closed outside superuser scope.
- [ ] Add server-side authz for `Ops` settings mutations so site-config changes, feature-flag changes, retention-policy changes, webhook-behavior changes, and integration-setting changes are gated and audited consistently.
- [ ] Add privacy and redaction rules for dashboard detail RPCs so customer chats, support data, billing records, and provider error payloads expose only the minimum operator-visible fields for each role.
- [ ] Add fresh-session and explicit-confirmation rules for high-risk `Business` mutations so plan edits, overage edits, and dunning actions cannot be triggered from a stale dashboard tab.
- [ ] Add scope and redaction rules for unified `Customers` account timelines so chat content, billing records, support notes, auth sessions, and audit events expose only role-appropriate fields.
- [ ] Add transcript and tool-trace visibility rules for `Chats` drill-downs so workspace admins can inspect in-scope failures without gaining global conversation access or hidden provider payloads.
- [ ] Add blast-radius authorization rules for `Providers` mutations so provider disable, fallback-routing changes, and model visibility changes require the correct role and return previewable affected-scope counts.
- [ ] Add operator-action authz rules for `Ops` retries and replays so incident updates, job retries, notification retries, and webhook replays are separately gated, audited, and fail closed outside the correct scope.
- [x] Wire the visit-to-first-chat auth path to durable server-side sessions so login/signup, refresh, and reopen-thread flows all use persisted `auth_sessions` and `auth_token_versions`.
	Server-issued auth tokens now carry durable `sid`/`ver` claims backed by `auth_sessions` and `auth_token_versions`, with persisted-session validation and logout revocation covering login/signup, refresh, and reopen flows.
- [x] Add auth-state enforcement for the public-to-app transition so blocked, expired, revoked, or malformed sessions fail into a sane auth route instead of a half-booted app shell.
	Client route guards now redirect resolved unauthenticated `/app...` routes to landing, and `GetSession` now fails with `Unauthenticated` when metadata tokens are invalid/revoked/malformed (including token-vs-peer precedence) so stale sessions cannot half-boot the app shell.
- [x] Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.
	Send-path entitlement and usage gates now map failures into a structured `send_access` decision shape (`decision`, `plan`, `action`, `reason`, `first_paid_action=chat.send`) so soft-upgrade vs hard-block outcomes are explicit and filterable.
- [x] Enforce dashboard access by resolved role so normal users cannot reach admin surfaces, workspace admins get workspace-scoped data, and superusers get platform-scoped data.
	Admin RPCs now resolve caller scope (`platform` for superusers, `workspace` for active workspace-admin memberships) and filter users/usage/conversations to workspace-member user IDs for workspace admins while keeping normal users denied.
- [x] Add server-side checks for admin deep links, slice RPCs, and detail views so unauthorized dashboard routes fail closed and redirect or render a sane denied state.
	Added an HTTP admin deep-link guard that fail-closes unauthorized `/app/admin*`, `/app/dashboard*`, `/app/su*`, and admin settings-panel links with redirect-to-`/app`, plus per-slice admin RPC scope gates and denial logging for dashboard home/users/usage/conversations slices.
- [x] Define session and auth-expiry behavior for long-lived dashboard tabs so expired admin sessions fall back safely without leaving stale privileged UI visible.
	Auth session handling now includes a periodic authenticated session check (every 2 minutes) that verifies server-side session validity and proactively resets auth/workspace state with an expiry message plus logout redirect when sessions are expired/revoked/missing, preventing stale privileged UI from lingering in long-lived tabs.
- [x] Add a typed auth bootstrap contract that returns session status, role summary, and expiry information needed for safe app-shell and dashboard entry decisions.
	`GetSessionResponse` now includes typed bootstrap fields (`session_status`, `session_id`, `token_version`, `expires_at`, `expires_in_seconds`, `role_summary`), with server-side role-scope resolution (`user`/`workspace_admin`/`superuser`) and client-side session checks consuming the expiry/status metadata for safer app-shell/dashboard entry fallback.
- [x] Enforce persisted token lifecycles for signup verification, password reset, and update-password flows using the new token tables instead of ad hoc state.
	Signup now persists `email_verification_tokens`, password reset and update-password now run through persisted `password_reset_tokens` consumption plus password-hash/token-version/session-revocation updates, and focused auth/store lifecycle tests plus a token-hash micro-benchmark were added.
- [x] Ensure logout, privilege downgrade, and session revocation immediately evict cached admin state and invalidate privileged dashboard access.
	Added focused auth/admin lifecycle coverage in `TestAdminAccessInvalidatesAfterRoleDowngradeAndSessionRevocation` (`server/app/startup_helpers_test.go`) and `TestAdminDashboardRPCsRequireSessionAuth` (`server/app/admin_dashboard_test.go`) proving immediate fail-closed admin access on role downgrade (`PermissionDenied`) and on session revocation/logout (`Unauthenticated`) with peer auth-cache eviction.
- [x] Define role-aware post-login redirect rules so normal users land in the chat app while admin-capable users can be routed intentionally into dashboard-capable entry points.
	Client auth success/bootstrap now consumes one persisted post-login route intent and `GetSession.role_summary.can_access_admin`, fail-closing non-admins to `/app` for admin intents while allowing admin-capable users to enter explicit dashboard/admin intents; focused wasm coverage was added in `TestParseResolvePostLoginRoute` and `TestPostLoginRouteIntentHelpers` (`client/app/route_sync_test.go`).
- [x] Add re-auth or heightened-session checks for sensitive superuser actions so long-lived dashboard sessions do not automatically authorize mutating control-plane changes.
	Sensitive superuser mutation stubs (`SetServerToolPolicy`, `RunServerTool`) now require a fresh session (`IssuedAt <= 20m`) via `parseRequireSuperuserMutationUserID`, returning `Unauthenticated` for stale tokens and requiring re-auth; coverage added in `TestSensitiveSuperuserMutationsRequireFreshSession` (`server/app/server_tool_stub_test.go`).
- [x] Define and enforce user-disable, user-restore, workspace-suspend, and workspace-restore authorization rules, including who can perform them and against which scopes.
	Added explicit admin-mutation authz seams in `parseAuthorizeAdminMutationAction` (`server/app/admin_mutation_authz.go`) for `user.disable`, `user.restore`, `workspace.suspend`, and `workspace.restore`: superusers get platform scope, workspace admins are restricted to in-scope users/workspaces, workspace admins cannot mutate superuser account state, and missing/unsupported targets fail closed. Coverage is in `TestAuthorizeAdminMutationActionScopes` (`server/app/admin_mutation_authz_test.go`).
- [x] Ensure disable and suspend actions revoke active sessions, block new auth, and deny downstream privileged RPCs immediately.
	Added shared mutation execution (`parseExecuteAdminMutationAction`) with concrete disable/suspend side effects in `server/app/admin_mutation_effects.go`: user/workspace auth blocks (`user_auth_blocks`), user access-state updates (`user_access_states`), token-version rotation, and active-session revocation. Auth validation/login now fail-close disabled or blocked users, and workspace-admin scope ignores suspended workspaces. Coverage: `TestAdminMutationDisableAndRestoreEnforcesImmediateAuthEffects` and `TestAdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects` (`server/app/admin_mutation_effects_test.go`).
- [x] Add confirmation and reason requirements for destructive admin actions so high-impact changes are auditable and harder to trigger accidentally.
	Added `parseRequireAdminMutationConfirmation` (`server/app/admin_mutation_effects.go`) and wired destructive side-effect handlers to require explicit confirmation plus non-empty reason before applying disable/suspend auth-block side effects; reasons are persisted in `user_auth_blocks.block_reason` for audit context. Coverage: `TestRequireAdminMutationConfirmation` (`server/app/admin_mutation_effects_test.go`).
- [x] Enforce feature-flag, experiment, and incident-control mutations behind the correct workspace-admin vs superuser boundaries.
	Added explicit control-plane mutation guard seams in `server/app/admin_control_mutation_authz.go`: feature-flag and experiment mutations are superuser-only, while incident mutations allow workspace-admin callers only inside their workspace scope and otherwise fail closed. Coverage: `TestAdminControlMutationScopeBoundaries` (`server/app/admin_control_mutation_authz_test.go`).
- [x] Define and enforce the exact runtime effects of user disable vs workspace suspend so chat send, dashboard access, API keys, webhooks, background jobs, and support actions all fail in a consistent way.
	Defined and enforced one shared runtime policy in code: disabled/auth-blocked users fail-closed for chat send plus user-scoped operational writes (`API keys`, `support tickets/messages`, `notification_outbox`), while suspended workspaces fail-closed for workspace-admin dashboard scope, workspace-scoped writes (`API keys`, `webhook endpoints`, `webhook deliveries`, `support tickets/messages`, `notification_outbox`), pending notification dispatch, and background jobs (both `queue_key=workspace:<id>` and payload-scoped `workspace_id` / `user_id`). Coverage: `TestSendDeniedForDisabledUser`, `TestSendDeniedForSuspendedWorkspace`, `TestWorkspaceScopedWritesBlockedWhenSuspended`, `TestUserScopedWritesBlockedWhenDisabled`, `TestWebhookAndSupportMessageBlockedWhenWorkspaceSuspended`, `TestDispatchNotificationOutboxPendingFailsSuspendedWorkspaceRows`, `TestHandleBackgroundJobsFailsSuspendedWorkspaceQueue`, and `TestHandleBackgroundJobsFailsBlockedPayloadScope` (`server/app/runtime_access_policy_test.go`).
- [x] Ensure restore flows re-enable only the intended scopes and do not silently reopen revoked sessions, API keys, or operator overrides unless explicitly requested.
	Restore flows now intentionally clear only the matching suspension/disable auth-block keys (`user.disabled`, `workspace.suspended.<id>`), keep old sessions revoked (token-version/session checks unchanged), and do not auto-reopen revoked API keys or disabled webhooks. Coverage: `TestAdminMutationRestoreUserPreservesNonDisableAuthBlocks` and `TestAdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides` (`server/app/admin_mutation_effects_test.go`).
- [x] Persist and enforce server-side auth sessions using `auth_sessions` and `auth_token_versions`.
	Server-issued auth tokens now carry durable session and version claims backed by `auth_sessions` and `auth_token_versions`, with active-session checks enforced during token validation and session revocation on logout.
- [x] Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.
	Tokens now stamp `jti` to match durable session IDs, session-version checks are enforced on every validation, and `kid`-aware verification supports active-key signing plus configured rotation keys (`CHAT_AUTH_SIGNING_KEYS` / `CHAT_AUTH_ACTIVE_KID`).
- [x] Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.
	`parseRequireUserEntitlement` now fails closed when the effective entitlement key is absent, and Send-path tests/benchmarks now seed explicit billing plans where non-entitlement outcomes are expected.
- [x] Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.
	`parseRequireUsageBudget` now enforces monthly token caps from `usage.monthly_token_limit`, plus per-user send-rate and concurrency leases (`usage.sends_per_minute`, `usage.concurrent_sends`) with stream-lifetime release semantics in `Send`.

### Agent 3: Control plane, ops, and backend workflows

- [x] Add typed `Business` dashboard summary and drill-down RPCs backed by typed SQL queries over `billing_customers`, `billing_subscriptions`, `billing_invoices`, `billing_invoice_line_items`, `billing_access_overrides`, `billing_events`, `usage_events`, `product_analytics_events`, and `subscription_churn_feedback`.
	Completed by wiring the existing typed summary RPC (`GetAdminDashboard`) with a new typed drill-down RPC (`GetAdminBusinessDrilldown`) that joins user-scoped billing customer/subscription/invoice/line-item/override/event slices plus usage, product analytics, and churn feedback rows (`server/app/admin_business_ops.go`, `server/app/store_growth_ops.go`, `sql/store/list_product_analytics_events_by_user.sql`, `sql/store/list_subscription_churn_feedback_by_customer.sql`, `proto/chat.proto`). Coverage: `TestGetAdminBusinessDrilldownRPCs` and `TestGetAdminBusinessDrilldownScopeGuards` (`server/app/admin_business_ops_test.go`).
- [ ] Add typed `Business` mutation RPCs, store funcs, and SQL queries for billing plans, plan entitlements, quota policies, overage rules, upgrade triggers, and dunning controls, including explicit audit-log writes for each mutation.
- [ ] Replace the seeded plan model and admin pricing controls so the canonical plans are `pro`, `team`, and `enterprise` only, removing stale public `free` positioning and aligning seeded plan metadata with the real product story.
- [ ] Add typed billing-plan fields, store funcs, SQL queries, and admin mutation RPCs for the usage-based pricing formula: `monthly_platform_fee_cents`, `usage_premium_basis_points`, `workspace_mode`, `min_seats`, `max_seats`, and the collaboration/admin capability flags that distinguish `Pro` from `Team`.
- [ ] Add typed billing-preview and invoice-breakdown RPCs backed by SQL queries over `usage_events`, `billing_subscriptions`, `billing_invoice_line_items`, and plan metadata so the product can show `platform fee`, `raw model usage`, `service premium`, and `total` consistently everywhere.
- [ ] Add typed invoice-line classification rules and persistence so generated invoices cleanly separate `platform_fee`, `usage_cost`, and `service_premium` rows rather than hiding everything inside one opaque total.
- [ ] Add one typed pricing-page content source, whether server-seeded config or admin-managed site config, so public plan names, fees, usage-premium percentages, and plan descriptions stop drifting from the actual billing model in code and schema.
- [ ] Add typed `Customers` dashboard list and detail RPCs backed by typed SQL queries over `users`, `user_profile`, `user_memory`, `workspaces`, `workspace_memberships`, `auth_sessions`, `auth_token_versions`, `support_tickets`, `support_ticket_messages`, `usage_events`, `billing_subscriptions`, and `audit_logs`.
- [ ] Add typed `Customers` mutation RPCs, store funcs, and SQL queries for user disable or restore, workspace suspend or restore, entitlement overrides, billing overrides, and support-linked operator actions, keeping the read and write paths separate and auditable.
- [ ] Add typed `Chats` dashboard summary and drill-down RPCs backed by typed SQL queries over `conversations`, `messages`, `usage_events`, `onboarding_templates`, `user_activation_milestones`, `saved_workflows`, `prompt_library_items`, `weekly_value_summaries`, `product_analytics_events`, and any share or comment tables once those are introduced.
- [ ] Add typed `Chats` settings RPCs, store funcs, and SQL queries for system-default prompt state, model defaults, reasoning defaults, memory extraction rules, onboarding templates, workflow publishing, and skill publishing, adding any missing tables only where the current schema cannot persist the setting cleanly.
- [ ] Add typed `Providers` dashboard summary and drill-down RPCs backed by typed SQL queries over `model_catalog`, `usage_events`, `workspace_model_routing_policies`, `workspace_cost_guardrails`, provider catalog metadata, and any persisted provider-health or provider-rate-limit tables that need to be added for durable trend reporting.
- [ ] Add typed `Providers` mutation RPCs, store funcs, and SQL queries for provider enable or disable, model visibility, fallback routing, provider limits, and cost guardrails, and add any missing provider-settings tables only if those settings cannot be represented cleanly in `site_config` or the existing routing tables.
- [ ] Add typed `Ops` dashboard summary and drill-down RPCs backed by typed SQL queries over `site_config`, `feature_flags`, `audit_logs`, `service_level_objectives`, `incidents`, `incident_updates`, `background_jobs`, `notification_outbox`, `webhook_endpoints`, and `webhook_deliveries`.
- [ ] Add typed `Ops` mutation RPCs, store funcs, and SQL queries for site config, feature flags, retention policies, webhook behavior, integration settings, and superuser controls, adding missing settings tables only where the current schema cannot persist the setting without overloading unrelated rows.
- [ ] Add one dashboard data-source map in code comments or helper docs that pairs each KPI card, trend, list, and settings form with its source RPC, store func, SQL query, and underlying table set so future changes stay traceable.
- [ ] Add rollup tables or materialized-summary jobs only for dashboard metrics that are too slow to serve from raw typed SQL queries, and keep each rollup narrowly scoped by surface (`Business`, `Customers`, `Chats`, `Providers`, `Ops`) instead of building one oversized catch-all summary table.
- [ ] Add typed `Business` queue and detail RPCs, store funcs, and SQL queries for failed-payment review, subscription detail, invoice history, dunning timeline, top-account drill-down, and first-chat conversion funnel investigation.
- [ ] Add a typed unified `Customers` account timeline RPC backed by SQL queries that merge recent chats, support activity, billing events, auth-session history, and audit events into one operator timeline without losing source-specific drill-down capability.
- [ ] Add typed `Chats` anomaly RPCs, store funcs, and SQL queries for failed replies, slow replies, high-cost threads, and feature-usage slices, and add a dedicated reply-attempt or message-run table only if `messages` plus `usage_events` cannot explain failures, latency, provider/model choice, and tool usage precisely enough.
- [ ] Add typed `Providers` health-trend and fallback-event RPCs, store funcs, and SQL queries, and introduce durable `provider_health_samples` and `provider_fallback_events` tables only if current runtime/provider state cannot support historical trend and blast-radius workflows.
- [ ] Add typed `Ops` queue and action RPCs, store funcs, and SQL queries for failed background jobs, failed notifications, failed webhook deliveries, incident timelines, recent admin actions, and retry or replay actions with explicit audit rows.
- [ ] Add typed mutation-preview RPCs and SQL queries for `Business`, `Providers`, and `Ops` changes so the dashboard can show blast radius, affected account counts, and likely downstream impact before a high-risk setting is committed.
- [x] Add typed funnel instrumentation for the visit-to-first-chat journey: landing viewed, pricing viewed, CTA clicked, auth started, auth completed, app booted, first thread created, first send started, first reply completed.
	Added typed first-chat funnel instrumentation in `server/app/funnel_first_chat.go` with canonical step keys and event names, wired it into shell-route handling plus `Signup`, `Login`, `GetSession`, and `Send`, and persisted authenticated steps via `product_analytics_events` with scoped workspace resolution and experiment bootstrap (`first-chat-funnel`).
- [x] Ensure typed store and RPC funcs exist for first-chat bootstrap data: profile defaults, selected settings, model catalog, conversation list, active conversation load, and default system prompt injection.
	Verified existing typed store+RPC coverage already satisfies this contract: profile/settings (`Get/SetUserName`, `Get/SetSelected*`, `Get/SetCustomSystemPrompt` + store getters/setters), model catalog (`ListModelOptions`, `parseLoadModelCatalogConfig`), conversation list/load (`ListConversations`, `LoadConversation`, `parseListConversations`, `parseLoadConversation`), and default system-prompt injection in `Send` when no custom prompt is persisted.
- [x] Add backend support for first-run onboarding milestones and starter-state helpers so the app can distinguish brand-new users from returning users with prior threads.
	Added starter-state backend helpers in `server/app/onboarding_state.go` and wired them into `ListConversations` and `Send`: brand-new vs returning state now syncs milestone rows (`starter.first_run_detected`, `starter.returning_user_detected`), and first successful exchange marks `starter.first_thread_created` + `starter.first_reply_completed`.
- [x] Add server-side write paths for first-thread creation metadata, first-reply completion markers, and reopen-thread analytics so the funnel can be queried end-to-end.
	`Send` now writes starter milestones for first-thread + first-reply completion (`starter.first_thread_created`, `starter.first_reply_completed`) and persists typed funnel events for first-send/first-reply, while `LoadConversation` now writes typed reopen analytics (`thread_reopened`) with conversation/message metadata.
- [x] Add typed dashboard-home summary funcs and RPCs for the admin journey: users, conversations, usage, billing signals, incidents, support load, and experiment health.
	Expanded `AdminDashboardSummary` with typed billing-signal, incident, support-load, and experiment-health fields, wired SQL/store/RPC mappings (`get_admin_dashboard_summary.sql`, `store_admin.go`, `GetAdminDashboard`), and added focused store/RPC assertions in `admin_dashboard_test.go`.
- [x] Add typed scoped query funcs and RPCs for workspace-admin slices: memberships, API keys, webhooks, audit logs, invitations, usage, and billing summary.
	Added typed `GetWorkspaceAdminSlices` with workspace-admin scope enforcement plus typed slice payloads for memberships/API keys/webhooks/audit/invitations/usage and typed billing summary rollups; covered by `TestGetWorkspaceAdminSlices`.
- [x] Add typed scoped query funcs and RPCs for superuser slices: global users, global usage, support queue, pricing controls, incidents, experiments, workspaces, and cost guardrails.
	Added typed `GetSuperuserSlices` and supporting typed store query funcs for pricing controls (`billing_plan_overages`, `billing_quota_policies`, `billing_upgrade_triggers`), incidents, and workspace cost guardrails, plus global users/usage/support/experiments/workspaces slices with focused RPC tests.
- [x] Add backend audit events for admin-dashboard entry, slice views, drill-down access, and mutating admin actions so operator activity is queryable.
	Added typed admin audit-event persistence for dashboard home/slice/drill-down access (`GetAdminDashboard`, `ListAdmin*`, `GetWorkspaceAdminSlices`, `GetSuperuserControlPlane`, `GetSuperuserSlices`) plus mutation audit writes in `parseExecuteAdminMutationAction`/`parseStoreAdminMutationAuditLog`, with focused assertions in admin dashboard, superuser slice, and mutation-effect tests.
- [x] Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.
	Added typed store helpers (`parseSearchAdminUsers`, `parseGetAdminUserSummaryByUserID`, `parseListAdminAuthSessionsByUser`, `parseListAdminUsageEventsByUser`, `parseListAdminAuditLogsByUser`) and typed RPC implementations (`SearchAdminUsers`, `GetAdminUserDetail`, `DisableAdminUser`, `RestoreAdminUser`) with focused store + RPC scope/mutation coverage in `admin_dashboard_test.go` (`TestStoreAdminDashboardQueries`, `TestAdminUserControlRPCs`, and scoped assertions in `TestGetWorkspaceAdminSlices`).
- [x] Add typed workspace-admin query and mutation funcs for workspace detail, suspend workspace, restore workspace, revoke API keys, pause webhooks, and membership review.
	Completed via existing typed workspace-admin and mutation seams plus dedicated helper surfaces in `server/app/workspace_admin_ops.go`: scoped detail/review (`parseGetWorkspaceAdminDetail`, `parseListWorkspaceAdminMembershipReview`) and typed mutations (`parseSuspendWorkspaceByAdmin`, `parseRestoreWorkspaceByAdmin`, `parseRevokeWorkspaceAPIKeyByAdmin`, `parsePauseWorkspaceWebhookByAdmin`). Coverage: `TestGetWorkspaceAdminSlices`, `TestWorkspaceAdminOpsDetailAndMutations`, `TestWorkspaceAdminOpsDenyOutOfScope`, `TestAdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects`, and `TestAdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides` (`server/app/admin_dashboard_test.go`, `server/app/workspace_admin_ops_test.go`, `server/app/admin_mutation_effects_test.go`).
- [x] Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.
	Implemented scoped backend seams in `server/app/billing_intervention_ops.go` (`parseGetBillingInterventionSnapshotByAdmin`, `parseApplyBillingAccessOverrideByAdmin`, `parseResolveBillingFailedPaymentByAdmin`, `parseListBillingDunningEventsByAdmin`) and exposed typed admin RPC behavior through the existing `server/app/admin_billing_ops.go` surface (`ListAdminBillingAccessOverrides`, `ListAdminBillingEvents`, `ListAdminBillingDunningEvents`, `SetAdminBillingAccessOverride`, `SetAdminBillingQuotaOverride`, `ResolveAdminBillingFailedPayment`). Coverage is in `server/app/billing_intervention_ops_test.go` and `server/app/admin_billing_ops_test.go`.
- [x] Add typed support-triage funcs and RPCs for ticket queues, ticket detail, internal notes, assignment, escalation, and account-linked action history.
	Added typed support-triage store helpers in `server/app/store_admin.go` (`parseListAdminSupportTickets`, `parseGetAdminSupportTicketByID`, `parseListAdminSupportTicketMessagesByTicketID`, `parseListAdminSupportAccountActionHistoryByUser`, `parseStoreAdminSupportTicketAssignment`, `parseStoreAdminSupportTicketEscalation`) and exposed scoped RPCs in `server/app/admin_support_ops.go` (`ListAdminSupportTickets`, `GetAdminSupportTicketDetail`, `AddAdminSupportInternalNote`, `AssignAdminSupportTicket`, `EscalateAdminSupportTicket`). Coverage: `TestStoreAdminSupportTriageFuncs`, `TestAdminSupportTriageRPCs`, and `TestAdminSupportTriageRPCsWorkspaceAdminScope` (`server/app/admin_support_ops_test.go`).
- [x] Add typed incident and experiment control funcs and RPCs for incident updates, status changes, feature-flag toggles, experiment rollbacks, and blast-radius reporting.
	Typed control-plane store and RPC surfaces are now wired in `server/app/store_admin.go` (`parseStoreAdminFeatureFlagToggle`, `parseStoreAdminExperimentRollback`, `parseStoreAdminIncidentStatusUpdate`) and `server/app/admin_control_ops.go` (`SetAdminFeatureFlag`, `RollbackAdminExperiment`, `UpdateAdminIncident`, `GetAdminIncidentBlastRadius`), with incident status SQL wiring in `sql/store/update_incident_status.sql` + query loading in `server/app/queries.go`, scope/confirmation enforcement, and focused coverage in `server/app/admin_control_ops_test.go` (`TestStoreAdminControlMutationFuncs`, `TestAdminControlOpsRPCs`, `TestAdminControlOpsWorkspaceScope`) plus authz-boundary coverage in `server/app/admin_control_mutation_authz_test.go` (`TestAdminControlMutationScopeBoundaries`).
- [x] Add typed search, filter, sort, and pagination support for admin user, workspace, support, incident, and billing list RPCs so large datasets are operable without loading everything at once.
	Wired typed list-query filtering/sorting/pagination in `server/app/admin_dashboard.go`, `server/app/admin_support_ops.go`, `server/app/admin_billing_ops.go`, and `server/app/superuser_control.go` using `AdminListQuery` + shared windowing helpers, and validated via focused RPC coverage in `server/app/admin_dashboard_test.go` (`TestAdminUserListQueryRPCs`), `server/app/admin_support_ops_test.go` (`TestAdminSupportTriageListQueryRPCs`), `server/app/admin_billing_ops_test.go` (`TestAdminBillingListQueryRPCs`), and `server/app/superuser_control_test.go` (`TestGetSuperuserSlicesAppliesListQueries`, `TestGetSuperuserSlicesAppliesTypedListQueries`).
- [x] Add explicit backend side-effect handlers for disable and suspend flows so session revocation, API-key pausing, webhook pausing, and job suppression happen transactionally and are auditable.
	Added explicit backend side-effect handlers in `server/app/admin_mutation_effects.go` and `server/app/store_superuser.go`: user-disable now revokes user-scoped API keys, workspace-suspend now transactionally revokes workspace API keys, disables webhook endpoints, and suppresses pending/running workspace-queued background jobs, while existing session revocation/token-version rotation remains enforced. Coverage was extended in `TestAdminMutationDisableAndRestoreEnforcesImmediateAuthEffects` and `TestAdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects` (`server/app/admin_mutation_effects_test.go`).
- [x] Add typed restore handlers that let operators choose which dependent capabilities come back automatically versus which stay manually revoked.
	Extended the typed mutation target contract (`parseAdminMutationTarget`) with explicit restore options and wired them through restore handlers (`server/app/admin_mutation_effects.go`): user restore can explicitly re-enable user API keys, and workspace restore can explicitly re-enable workspace API keys, webhook endpoints, and suppressed background jobs; default behavior remains fail-safe (no automatic re-enable unless requested). Coverage: `TestAdminMutationRestoreUserCanExplicitlyRestoreAPIKeys` and `TestAdminMutationRestoreWorkspaceCanExplicitlyRestoreOperationalCapabilities` (`server/app/admin_mutation_effects_test.go`).
- [x] Add typed store/query funcs for the new operational tables: auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs.
	Added typed SQL query files, query-loader wiring, and typed store methods for workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs, with auth-session list coverage integrated into the existing auth-session store path.
- [x] Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.
	`GetSuperuserControlPlane` now returns typed rows for auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs via extended protobuf response fields.
- [x] Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.
	Added typed superuser pricing mutation RPCs (`Set/DeleteSuperuserBillingPlanOverage`, `Set/DeleteSuperuserBillingQuotaPolicy`, `Set/DeleteSuperuserBillingUpgradeTrigger`, `Set/DeleteSuperuserBillingDunningEvent`) in `server/app/superuser_pricing_ops.go` with superuser authz gating, explicit confirm+reason enforcement, and admin audit-event emission; added focused store+RPC coverage and micro-bench in `server/app/superuser_pricing_ops_test.go`.
- [x] Add typed `su` CRUD RPCs for reliability and trust tables: SSO configs, retention policies, compliance controls, SLOs, incidents, and incident updates.
	Implemented typed superuser reliability/trust mutation RPCs in `server/app/superuser_reliability_ops.go` (`Set/DeleteSuperuserWorkspaceSSOConfig`, `Set/DeleteSuperuserDataRetentionPolicy`, `Set/DeleteSuperuserComplianceControl`, `Set/DeleteSuperuserServiceLevelObjective`, `Set/DeleteSuperuserIncident`, `Set/DeleteSuperuserIncidentUpdate`) with superuser mutation authz, explicit confirmation+reason gates, and audit event tracking; added typed store/query wiring and SQL contracts in `server/app/store_superuser_reliability.go`, `server/app/queries.go`, and `sql/store/*` for SSO, retention, compliance, SLO, incident, and incident-update get/list/upsert/delete paths, with focused store+RPC coverage and micro-benchmark in `server/app/superuser_reliability_ops_test.go`.
- [x] Add a restricted read-only admin query surface for debugging and reporting instead of any raw SQL passthrough RPC.
	Added allowlisted read-only reporting RPC `GetAdminReadOnlyReport` with typed `report_key` dispatch (`dashboard_summary`, `recent_users`, `recent_usage_events`, `recent_conversations`) so admin callers get scoped debug/reporting slices without arbitrary SQL execution; implemented in `server/app/admin_readonly_report_ops.go` with admin-scope enforcement and audit tracking, plus focused scope/guard tests and micro-benchmark in `server/app/admin_readonly_report_ops_test.go`.
- [x] Wire store/query funcs for onboarding templates, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
	Added typed SQL files, query-loader entries, and typed store methods for all listed growth tables, with focused lifecycle test coverage proving insert/upsert + list behavior in the server app.
- [x] Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.
	Added typed enqueue + dispatch flows for weekly summaries, dunning retries, retention purges, and health-score refresh jobs, including retry/backoff status transitions with focused lifecycle coverage.
- [x] Add customer-facing notification flows backed by `notification_outbox`.
	Added pending-queue notification dispatch plumbing with typed pending-list + status-update store funcs and a server-side dispatch helper that marks rows as `sent` or `failed` while preserving scheduled future rows.
- [x] Add webhook retry and delivery history plumbing backed by `webhook_deliveries`.
	Added pending-retry listing plus failed-attempt and successful-delivery update paths over `webhook_deliveries`, with focused lifecycle coverage proving retry rows can be scheduled, retried, and cleared on success.
- [x] Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.
	Added `defaultCustomSystemPromptTemplate` (`{{date}}`, `{{time}}`, `{{memories}}`) with server fallback wiring so first-send/new-thread flows always apply the runtime-resolved default prompt when no user override exists.
- [x] Fix the memory extraction system so remembered items are extracted reliably, deduplicated sanely, editable, and consistent with what users see in the remembered-preferences UI.
	Memory extraction now deduplicates candidate sets by normalized memory signature, reuses existing keys to keep edits stable, and deduplicates prompt-block injection so remembered-preferences UI and injected memory context stay aligned.
- [x] Keep `SCHEMA_TABLES.md` in sync as store funcs and RPCs are added for the new tables.
	Added a 2026-03-27 wiring snapshot that maps newly wired operational/growth tables to current store funcs, `GetSuperuserControlPlane`, notification/job flow helpers, and memory extraction RPC/runtime paths.
- [x] Document which schema tables are persistence-only vs fully wired into store, RPC, jobs, and UI.
	Added a wiring-classification block in `SCHEMA_TABLES.md` with explicit `Store+RPC+Jobs+UI`, `Store+RPC+Jobs`, `Store+RPC`, `Store-only`, and `Persistence-only` categories for the current schema set.
- [x] Add an operator runbook for local start, seed, build-client, and superuser/admin verification flows.
	Added `OPERATOR_RUNBOOK.md` with a repo-root operator sequence for env setup, WASM builds, DB seeding, managed server lifecycle, chat verification, and superuser/admin backend verification commands.

### Agent 4: Docs, flow definition, and operator guidance

- [x] Document the five core dashboard surfaces in product terms so maintainers know exactly what belongs in `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, and what intentionally does not belong there.
	Added `Core dashboard surfaces` to `DESIGN.md` with explicit scope boundaries for `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, including a paired "not in this surface" clause for each slice.
- [x] Add one dashboard metric-definition section that defines each core KPI, its source-of-truth table set, its source RPC, and any caveats about lag, rollups, or derived values.
	Added `Dashboard KPI definitions (planned)` in `DESIGN.md` with a per-KPI map of source tables, summary RPC ownership, and caveats for lag, rollups, and derived-signal interpretation.
- [x] Add one dashboard endpoint and query map that lists the planned summary RPCs, drill-down RPCs, mutation RPCs, store funcs, SQL queries, and underlying tables for each dashboard surface.
	Added `Dashboard endpoint and query map (planned)` in `DESIGN.md`, mapping each surface (`Business`, `Customers`, `Chats`, `Providers`, `Ops`) to summary/drill-down/mutation RPCs, store-function families, SQL query families, and primary table sets.
- [x] Document the dashboard permissions model so maintainers know which surfaces, lists, detail views, and settings forms belong to normal users, workspace admins, and superusers.
	Added `Dashboard permissions model (planned)` to `DESIGN.md` with a role matrix covering surface visibility, list/detail scope, and mutation scope for normal users, workspace admins, and superusers.
- [x] Add a dashboard operator runbook that explains how to review revenue, customer health, chat health, provider health, and site health without drifting into vanity metrics or redundant surfaces.
	Added `Dashboard Review Playbook (Planned)` to `OPERATOR_RUNBOOK.md` with an action-first review loop for `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, plus explicit anti-vanity guidance for each surface.
- [ ] Document the day-in-the-life operator workflows for `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, including queue-to-detail-to-action loops, expected decision points, and the evidence an admin should review before mutating state.
- [ ] Add one mutation preflight guide that defines what blast-radius preview, affected-record counts, audit notes, confirmation copy, and rollback guidance each high-risk dashboard action must show before it is considered safe to ship.
- [ ] Document the pricing philosophy and plan boundary in operator/product terms: `Pro = one serious operator`, `Team = shared workspace with coordination/admin relief`, `Enterprise = contract/security/compliance path`, and `monthly bill = platform fee + usage + service premium`.
- [ ] Document the exact customer-facing billing vocabulary so docs and UI consistently use `platform fee`, `usage`, and `service premium`, and explicitly avoid stale phrases like `free`, `unlimited`, `all models included`, or `no token caps`.
- [x] Document the visit-to-first-chat flow in product terms, including which routes, panels, defaults, and backend calls participate at each stage. (`FLOWS.md`, Visit-To-First-Chat section)
- [x] Refresh `README.md` so it reads like a polished entry point for example 100, preserving the existing ASCII architecture diagram and enhancing it to better represent the current server, client, worker, routing, and gRPC flow. (`README.md`, intro + Architecture + Data flow)
- [x] Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly. (`README.md`, Quick start)
- [x] Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior. (`README.md`, Route model)
- [x] Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer. (`README.md`, Runtime pieces)
- [x] Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions. (`FLOWS.md`, Admin-Dashboard Journey section)
- [x] Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control. (`FLOWS.md`, Admin Operational Workflows section)
- [x] Document the admin search/filter/pagination conventions so operators know how large lists, saved context, and back-navigation are expected to behave. (`FLOWS.md`, Admin List Conventions)
- [x] Document the disable-vs-suspend semantics so maintainers and operators know exactly which downstream capabilities are supposed to turn off in each case. (`FLOWS.md`, Disable vs Suspend Semantics)
- [x] Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code. (`FLOWS.md`, Public-Route Bug-Fix Workflow)
- [x] Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues. (`FLOWS.md`, First-Chat Bug-Fix Workflow)
- [x] Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps. (`FLOWS.md`, Admin-Dashboard Bug-Fix Workflow)
- [x] Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate. (`MANUAL_SMOKE.md`, Release Gate Checklist)
- [x] Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage. (`MANUAL_SMOKE.md`, Testing Story Matrix)
- [x] Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently. (`docs/BUG_REPORT_TEMPLATES.md`)
- [x] Refresh `MANUAL_SMOKE.md` so the manual verification flows cover landing routes, auth, first chat, settings, admin dashboard entry, and key operator actions. (`MANUAL_SMOKE.md`, flow sections 1-4)
- [x] Update `DESIGN.md` so it reflects the current intended product surface, dashboard direction, and any material UI or IA changes made since the original design pass.
	Replaced the older visual-redesign spec with a current design-intent document covering route classes, role-split dashboard direction, state/feedback rules, and material IA changes.
- [x] Keep `SCHEMA_TABLES.md` synchronized with the control-plane, auth, onboarding, and admin workflow stories so the docs match the real table usage.
	Updated wiring snapshot/classification and added a story-alignment map linking active flows to the actual auth, control-plane, billing, onboarding, and admin table usage.
- [x] Add one concise docs map to the example that tells maintainers which file to read for setup, architecture, smoke testing, schema reference, design intent, and changelog history. (`DOCS_MAP.md`)
- [x] Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.
	Moved both files to `bin/runtime/logs/`; scanned `cmd/server`, `scripts`, `server`, and managed launcher paths and found no active references still writing these files at the example root.
- [x] Move any other loose runtime outputs out of the example root into a consistent runtime location such as `bin/runtime/`, `bin/logs/`, or another single clear convention.
	Updated server log output directory from `log/` at example root to `bin/runtime/logs/`, and moved existing `server.stderr.log`/`server.stdout.log` artifacts there.
- [x] Rename inconsistent or low-signal files so the example reads professionally to a new maintainer, then update all references in scripts, docs, tests, and server/client code.
	Renamed `BENCHMARKS.md` to `PERFORMANCE.md` and updated the README reference so naming is clearer and consistent with performance-focused content.
- [x] Normalize file placement so docs stay at the example root, scripts stay under `scripts/`, runtime outputs stay under runtime folders, and generated artifacts stay under `bin/` or another clearly non-source location.
	Removed loose `bin/` root artifacts by relocating server binaries to `bin/server/` and parking legacy build leftovers under `bin/runtime/legacy-artifacts/`; runtime logs now live under `bin/runtime/logs/`.
- [x] Move non-first-class docs into a dedicated `docs/` folder while keeping the primary entry docs at the example root, then update all links and cross-references after the move.
	Moved `BUG_REPORT_TEMPLATES.md` and `PERFORMANCE.md` into `docs/` and updated `README.md`, `DOCS_MAP.md`, and TODO cross-references to the new paths.
- [x] Update `.gitignore` for example 100 so logs, runtime outputs, temp artifacts, and local-only generated files are ignored from their new locations.
	Refreshed ignore rules to match the new runtime layout by dropping obsolete `server/server.*.log` paths and explicitly ignoring root fallback log outputs plus `log/`.
- [x] Remove stale root-level files that only existed because of the old layout once their replacements are wired and validated.
	Validated and cleared old root artifacts from prior layout (`server.stderr.log`, `server.stdout.log`, `BUG_REPORT_TEMPLATES.md`, `PERFORMANCE.md`, `BENCHMARKS.md`) after wiring new runtime/docs locations.
- [x] Update `README.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, and any other affected docs after the cleanup so every referenced path still opens correctly.
	Replaced stale local-script paths and outdated file-layout block in `README.md`, added `docs/BUG_REPORT_TEMPLATES.md` reference in `MANUAL_SMOKE.md`, and added doc-location cross-reference guidance in `SCHEMA_TABLES.md`.
- [x] Add one short repo-layout section to the example docs that explains where logs, runtime state, generated artifacts, scripts, and source files belong.
	Added `Repo layout rules` to `README.md` with explicit placement guidance for root docs, `docs/`, `scripts/`, runtime state/logs, generated artifacts, and source directories.

### Agent 5: UI surface and dashboard UX implementation

- [x] Refine the public-to-first-chat UX so landing, pricing, signup, login, empty state, first composer state, streaming state, and post-first-reply state read like one coherent journey.
	Added one shared journey band across landing/pricing/signup/login/workspace plus state-derived first-chat guidance in empty and composer surfaces for first prompt, first-reply streaming, and post-first-reply continuation.
- [x] Refresh the chat surface styling so its colors, border treatments, and shapes feel like the same product as the home page instead of a separate UI.
	Retuned workspace base/background, sidebar cards, composer shell, send CTA, assistant badge, and message/user bubble surfaces to the marketing palette (`#050508`/`#111118` + cyan accents) with consistent border density and shape language.
- [x] Slightly retune the in-app chat palette to align with the landing-page direction while preserving readability, hierarchy, and streaming-state clarity.
	Shifted chat palette accents from mixed green/gray to landing-aligned cyan tones across thought-streaming states, scrollbars, toolbar focus states, and bubble/assistant treatments while preserving contrast and role hierarchy.
- [x] Adjust chat cards, bubbles, inputs, panels, and buttons so their corner radii, outlines, and surface shapes better match the home-page design language.
	Normalized shape language across the workspace shell by introducing `chat-thread-surface` and `chat-toolbar-shell`, tightening bubble radii to consistent 1.6rem corners, updating composer to a unified 1.85rem shell, and retuning outline strength for toolbar selects and bubble surfaces.
- [x] Build a first-run empty-state and onboarding layer with starter prompts/templates, obvious first actions, and a clear transition from zero threads to the first real thread.
	Expanded `parseEmptyState` into an onboarding surface with guided copy plus three starter prompt cards, wired `data-starterprompt` handling through `chatStreamController.ApplyStarterPrompt`, and prefill+focus behavior so clicking a starter immediately prepares the first send path.
- [x] Add UI treatment for auth failures, entitlement blocks, upgrade prompts, and post-first-reply success cues so users always know the next action in the journey.
	Added composer-level journey cue banners derived from runtime state in `client/app/panel.go` and `client/app/composer.go`, including explicit next-action copy for session/auth failures, entitlement or billing/upgrade blocks, and first-reply completion success.
- [ ] Collapse the public marketing IA to the pages worth keeping: keep `/home`, `/pricing`, `/signup`, `/` auth entry, `/app/settings`, and the dashboard entry; fold the strongest `/capabilities` content into `/home`, treat `/plans` as a pricing alias only, and remove fake footer destinations until real pages exist.
- [ ] Rewrite the kept `/home` page so it carries the full product story by itself: sharp value prop, who it is for, core capabilities, proof or trust signals, and one primary CTA path without leaning on a separate capabilities page.
- [ ] Tweak the `/home` hero copy with concrete product language.
	Example headline: `AI chat for teams that need faster answers and fewer repeat questions.`
	Example subhead: `RelayDesk turns internal docs, shared prompts, and model routing into one reliable workspace for support, ops, and customer-facing teams.`
	Example primary CTA: `Create your workspace`
	Example secondary CTA: `See pricing`
- [ ] Tweak the `/home` supporting sections so they answer who it is for and why it is useful.
	Example section heading: `Built for support, ops, and internal knowledge teams`
	Example body: `Use RelayDesk to answer repeat questions faster, standardize team responses, search company knowledge, and keep admins in control of cost, access, and model behavior.`
	Example capability bullets: `Ask across docs`, `Route to the right model`, `Save reusable workflows`, `Review usage and admin controls`
- [ ] Tweak the `/home` proof, metric, and trust copy so it reads like a credible business.
	Example trust band: `Flexible model routing`, `Admin controls`, `Usage tracking`, `Workspace-ready billing`
	Example proof copy: `Designed for teams that need grounded answers, cleaner handoffs, and clear visibility into usage.`
	Example metric-label replacements: change decorative labels into `Workspace controls`, `Model flexibility`, and `Operational visibility`.
- [ ] Tweak the `/home` section and footer copy so the whole page reads like one coherent SaaS narrative.
	Example footer group labels: `Product`, `Company`, `Legal`
	Example footer links to keep: `Home`, `Pricing`, `Sign up`, `About`, `Contact`, `Privacy`, `Terms`, `Security`, `Status`
	Example copy rule: no `framework`, `demo`, `lab`, or placeholder-style wording anywhere on kept pages.
- [ ] Rewrite the kept `/pricing` page with concrete plan-selection copy.
	Example page headline: `Usage-based pricing with a clear platform fee.`
	Example subhead: `Pay a monthly RelayDesk fee plus actual AI usage, with a small service premium that keeps billing predictable and the product sustainable.`
	Example plan framing: `Pro for one serious operator`, `Team for a shared workspace`, `Enterprise for security, compliance, and contract buying`
	Example FAQ angle: answer `What is the platform fee?`, `How is usage billed?`, `What is the service premium?`, `When do I need Team instead of Pro?`, and `How do I contact sales?`
- [ ] Rewrite the kept `/pricing` plan cards so they sell the emotional boundary clearly.
	Example `Pro` card framing: `For one operator who wants a reliable AI workspace without team overhead.`
	Example `Team` card framing: `For teams that need shared workflows, admin visibility, and less chaos.`
	Example `Enterprise` card framing: `For organizations that need security review, procurement, SSO, and contract controls.`
- [ ] Rewrite the kept `/pricing` numbers and labels so the formula is explicit.
	Example labels: `Monthly platform fee`, `Actual AI usage`, `RelayDesk service premium`
	Example pricing-note copy: `Model usage is billed separately at cost plus a small RelayDesk premium.`
- [ ] Rewrite the kept `/pricing` comparison table so it compares `workspace mode`, `collaboration`, `admin controls`, `billing visibility`, `support`, and `security/compliance path` instead of implying the main difference is just more usage.
- [ ] Rewrite the kept `/signup` page with concrete conversion copy.
	Example headline: `Create your RelayDesk workspace`
	Example subhead: `Start as one serious operator, add your team when coordination and admin control matter, and keep billing clear from day one.`
	Example trust copy: `Clear platform fee. Visible usage. Workspace-ready when your team is.`
- [ ] Rewrite the kept `/` auth landing with concrete login-first copy.
	Example login headline: `Welcome back`
	Example login body: `Sign in to continue your chats, settings, and workspace activity.`
	Example switch copy: `New to RelayDesk? Create an account`
	Example reset copy tone: `Reset your password and get back into your workspace quickly.`
- [ ] Add a real `/about` page with concrete business copy.
	Example headline: `About RelayDesk`
	Example body: `RelayDesk helps teams turn AI chat into reliable day-to-day work by combining grounded answers, reusable workflows, and practical admin controls in one workspace.`
	Example CTA: `View pricing` and `Create your workspace`
- [ ] Add a real `/contact` page with concrete business copy.
	Example headline: `Contact RelayDesk`
	Example body: `Talk to us about sales, support, onboarding, or enterprise requirements.`
	Example sections: `Sales`, `Support`, `General`, `Response times`
	Example expectation copy: `We aim to respond to product and sales questions within one business day.`
- [ ] Add a real `/privacy` page with concrete top-level summary copy before the legal detail.
	Example intro: `Privacy at RelayDesk`
	Example summary bullets: `What data we collect`, `How we use it`, `How long we retain it`, `How to contact us about privacy requests`
- [ ] Add a real `/terms` page with concrete top-level summary copy before the legal detail.
	Example intro: `RelayDesk Terms`
	Example summary bullets: `Account responsibilities`, `Acceptable use`, `Billing and renewals`, `Service availability`, `Contact information`
- [ ] Add a real `/security` page with concrete trust copy.
	Example headline: `Security at RelayDesk`
	Example body: `RelayDesk is built for teams that need clear access controls, auditable operations, and predictable handling of workspace data.`
	Example sections: `Access control`, `Data handling`, `Retention`, `Operational safeguards`, `Contact`
- [ ] Add a real `/status` page or status surface with concrete operational copy.
	Example headline: `RelayDesk Status`
	Example body: `Track current platform health, recent incidents, and service updates in one place.`
	Example sections: `Current status`, `Active incidents`, `Recent history`, `Planned maintenance`
- [ ] Tighten the kept `/app/settings` content so each section title, summary, and helper line explains one real user decision instead of repeating vague product language, with profile and billing reading like first-class account surfaces.
- [ ] Tighten the kept dashboard-entry content so it explains the five admin surfaces in operator language, shows where to start when something is wrong, and avoids filler copy or placeholder-card language.
- [ ] Upgrade the footer into a real business footer by wiring it only to first-class destinations: `/home`, `/pricing`, `/signup`, `/about`, `/contact`, `/privacy`, `/terms`, `/security`, and `/status`, while dropping low-signal filler links like `Careers`, `Customers`, and `Documentation` until those pages actually exist.
	Example column labels: `Product`, `Company`, `Legal`
	Example final link set: `Home`, `Pricing`, `Sign up`, `About`, `Contact`, `Privacy`, `Terms`, `Security`, `Status`
- [ ] Build a compact dashboard shell with one shared time-range control, one shared search/filter rail, alert state, and a clear split between summary cards, trend blocks, drill-down tables, and settings panels.
- [x] Build the `Business` dashboard UI with core KPI cards, one or two trend blocks, a top accounts table, and tightly scoped settings panels for plans, quotas, overages, upgrade triggers, and dunning.
- [x] Build the `Customers` dashboard UI with user and workspace list views, compact detail panes, recent-session and recent-usage panels, billing and support context, and clear disable or suspend state treatment.
- [x] Build the `Chats` dashboard UI with first-chat conversion, thread and message health, reply latency and failure views, and compact settings panels for system prompt, defaults, memory rules, onboarding templates, workflows, and skills.
- [x] Build the `Providers` dashboard UI with provider and model health cards, latency or error trends, cost tables, routing and fallback state, and compact settings panels for provider toggles, model visibility, limits, and guardrails.
- [x] Build the `Ops` dashboard UI with server and gRPC health, jobs, notifications, webhook failures, incidents, SLOs, audit feed, and compact settings panels for site config, feature flags, retention, and integrations.
- [x] Keep every dashboard chart paired with a useful drill-down table, drawer, or detail route so no dashboard widget is purely decorative.
- [x] Keep dashboard density under control by reusing one table pattern, one detail-drawer pattern, one settings-panel pattern, and one empty/loading/denied/error pattern across all five surfaces.
- [x] Design a clear admin entry point from the authenticated app shell so workspace admins and superusers can discover dashboard access without cluttering the normal user flow.
	Role-gated admin entry buttons are now present in both mobile and desktop control bars (`client/app/panel.go`) and route through the authenticated shell’s `OpenAdminDashboard` handler (`client/app/app.go`) so non-admin users never see admin affordances.
- [x] Build a dashboard home UI that summarizes the platform state and clearly branches into workspace-admin vs superuser slices.
	Created `client/app/dashboard.go` with `renderDashboardHome` (role banner, five slice tiles with Coming-soon state, account summary grid) and wired it in `renderWorkspaceShell` via `isDashboardRoute`; added route constants for all five slices to `routes.go`.
- [ ] Build slice UIs for the admin journey in dependency order: dashboard home, workspaces, support, billing and quotas, incidents and SLOs, analytics, and experiments.
- [x] Add clear empty, loading, denied, and error states for every dashboard surface so admins always know what the system is doing and what action to take next.
- [ ] Build a user-detail admin UI with disable or restore actions, recent activity context, billing context, support context, and clear impact messaging.
- [ ] Build a workspace-detail admin UI with suspend or restore actions, member context, API key and webhook context, and operational-impact messaging.
- [ ] Build billing-intervention UI flows for overrides, failed-payment review, quota review, and entitlement troubleshooting.
- [ ] Build support-triage UI flows for queue review, ticket detail, internal notes, escalation, and linked account actions.
- [ ] Build incident, feature-flag, and experiment control UIs with explicit state-change affordances and clear blast-radius feedback.
- [ ] Build admin list views with search, filters, sort controls, pagination, and persistent query state for users, workspaces, support queues, billing views, and incidents.
- [ ] Build confirmation modals for disable, restore, suspend, override, and rollback actions with reason capture, scope-of-impact copy, and clear success or failure feedback.
- [ ] Show the operational state of disabled users and suspended workspaces clearly across detail views, lists, and related action surfaces so operators can see what is active vs blocked at a glance.
- [x] Make the first settings option a simple profile page with display-name editing, account total/cost summary placement, and the core user fields and preferences surfaced cleanly.
	Expanded the `settingsSectionProfile` default pane in `renderActiveSettingsPane` to show three cards: display-name input, read-only account email, and an account usage card showing total spend from `accountCostSummary` (loading state shown when no exact costs are available yet).
- [x] Add a dedicated billing menu to settings for plan, usage, invoice, and account-total visibility.
	Added `settingsSectionBilling` constant to `settings_route.go` and wired it through `parseNormalizeSettingsSectionID`, `settingsSectionTitle`, `settingsSectionEyebrow`, `settingsSectionDescription`, the settings nav, and a new pane in `renderActiveSettingsPane` showing plan label, usage breakdown (base cost + premium row + total row), and coverage info. Added `parseBillingCoverageText` helper to `helpers.go`; added all i18n keys to EN/ES/FR locales.
- [x] Flatten the AI tone widget by one level so the control is less nested and faster to scan.
	Removed the outer `flex flex-col gap-4` wrapper from `settingsSectionTone` in `renderActiveSettingsPane`; the card Div now carries the ID directly, matching the flat structure of the prompt panel.
- [x] Flatten the reasoning menu by one level and explain that it sets the default reasoning level used when switching providers or models.
	Flattened `settingsSectionIntelligence` the same way; updated `modal.intelligenceHelp` in all three locales to read "Sets the default reasoning level used when switching providers or models."
- [x] Move the "Reasoning mode is on, so only reasoning-capable models are shown" message out of the main layout and show it as a hover overlay on the enabled/disabled reasoning controls.
	Wrapped the intelligence `renderToolbarSelect` in a `group relative` div in both mobile and desktop control bars; the filter message now renders as an `opacity-0 group-hover:opacity-100` absolute tooltip and is no longer injected inline into the layout.
- [x] Flatten the TTS provider control by one level and add a toggle for cross-provider override behavior.
	Flattened `settingsSectionSpeech` the same way as tone and intelligence. Also added `OnScroll` to `html/html.go`, `html/sugar.go`, and `html/shorthand/shorthand.go` to resolve the pre-existing build error in `sidebar.go`.
- [x] Normalize the left-pane system-prompt panel height so it matches neighboring panels and reduce the header copy to a simple heading.
	Replaced the long `modal.systemPromptHelp` summary in the left settings nav with a brief `parseSystemPromptNavSummary` that shows "Custom" vs "Default" based on whether the user has entered a prompt. Reduced the textarea `min-h` from `18rem` to `10rem` so the panel doesn't tower over adjacent sections.
- [x] Redesign the remembered-preferences form to be more compact and scrollable, with per-memory edit and delete icon actions.
	Rewrote `renderEditableUserMemories` in `memory_editor.go`: each card now has a compact header row with the category label and an icon-only delete button (`parseConversationDeleteIcon`), field border radii reduced, textarea min-heights reduced (3rem/2.5rem), and the reason textarea is visually dimmed to signal secondary importance.
- [x] Fix the top-left branding badge so the `GWC` text stays centered inside the circle, scaling the circle up if needed.
	Adjusted `parseAssistantAvatar` (`client/app/avatar.go`) to use a larger 36px circular badge with centered flex alignment and tighter text metrics (`text-[11px]`, `leading-none`, tracking) so `GWC` stays visually centered.
- [ ] Build a superuser dashboard UI for pricing controls, workspaces, support tickets, incidents, and cost guardrails.
- [ ] Build a workspace admin UI for API keys, webhook endpoints, invitations, and audit logs.
- [x] Enlarge and recenter the scroll-to-bottom button so it sits clearly centered above the input fields.
	Changed `bottom-7` → `bottom-24` so the button floats well above the composer, and `h-16 w-16 text-[1.25rem]` → `h-[4.5rem] w-[4.5rem] text-[1.5rem]` for a larger, more visible target.
	Updated `#scroll-to-bottom-btn` in `client/app/thread.go` to a larger 64px control with stronger shadow and a slightly higher resting offset (`bottom-7`) while keeping the centered `left-1/2` anchor above the composer lane.
- [ ] Add chat-thread folders so users can organize conversations into named groups.
- [ ] Support drag-and-drop of chat threads into folders.
- [ ] Support drag-to-reorder for chat threads and folders in the sidebar.
- [x] Make the chat-thread list independently scrollable with stable scroll behavior.
	Sidebar refresh now captures and restores conversation-list scroll state (`client/app/conversations.go`, `client/app/helpers.go`), preserving prior offset or bottom-pinned position so list updates do not jump the user unexpectedly.
- [x] Lazy load chat threads from gRPC as the user scrolls through the sidebar list.
	Added paged `ListConversations` support (`page_size`, `page_offset`, `has_more`, `next_offset`) and wired sidebar lazy loading so initial fetch loads one page and additional gRPC pages load as the user scrolls near the bottom.

- [x] Prevent newly created threads from being cleared when the first assistant response finishes.
	The root-route reset guard now waits for a resolved public conversation ID before treating `/` as an explicit "new chat" navigation.
- [x] Add diagnostics for unresolved thread-route state after a fresh reply.
	The client now logs one warning when route-sync intentionally defers a root reset because a new conversation has an ID but no public route yet, and it logs when a reply completes before the conversation list can resolve that route.
- [x] Flatten the provider, model, and intelligence controls so they use width more efficiently.
	The control bar now keeps labels and selects on the same row, gives the model picker more horizontal room, and compresses the mobile layout into a tighter two-row grid instead of a tall three-row stack.
- [x] Add hover and press animations to the toolbar selects.
	The provider, model, and intelligence selects now animate on hover, focus, and press, and the native option rows receive styled hover or selected states where the browser honors option styling.
- [x] Add a floating down-arrow when more thread content is available below.
	The thread panel now shows a clickable jump-to-bottom button only when the message list has meaningful scrollable space below the viewport, and clicking it smoothly returns the user to the latest messages.
- [x] Rebrand the example-100 product surface away from the framework name.
	The visible app name is now `RelayDesk`, and the server-rendered page title now presents the app as an AI chat workspace instead of a framework lab page.
- [x] Keep provider/model/intelligence selection stable across new chats and reconnects.
	The model picker now repairs blank or invalid persisted selections, persists the recovered model back to the server, and defaults fallback recovery to the first catalog model with `medium` thinking enabled so users do not land in an empty provider/model state.

## AuthN/Z hardening rollout (paying API scope)

- [x] Gate admin analytics RPCs and diagnostics log tail behind superuser authorization.
	`GetAdminDashboard`, `ListAdminUsers`, `ListAdminUsageEvents`, `ListAdminConversations`, and `GetLogTail` now call the superuser guard instead of allowing broad authenticated access.
- [x] Add runtime auth-secret startup validation for production deployments.
	Server startup now fails when `CHAT_ENV=production` and `CHAT_AUTH_SECRET` is missing unless `CHAT_ALLOW_INSECURE_AUTH_FALLBACK` is explicitly enabled.
- [x] Add entitlement and usage-budget gate seams for chat send requests.
	`Send` now calls centralized entitlement and budget guards, with TODO stubs documenting where deny-by-default rollout and quota enforcement will be wired once billing bootstrap and counters are finalized.
- Remaining open hardening work is tracked in the canonical backlog section above:
	`Agent 2: Security, auth, and billing enforcement`

## Changelog

- See `CHANGELOG.md` for execution checkpoints and completed-work notes.
