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
3. Users can run a thread in â€œask with docsâ€ mode and receive answers grounded in retrieved passages.
4. Responses show which files and chunks were used so the result is verifiable.
5. The backend supports document ingestion, vector search or equivalent retrieval, reindexing, and permission-aware access.

### Skills

1. Users or admins can define reusable â€œskillsâ€ that package prompts, tool availability, docs scope, and execution rules.
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

## Server-Owned I18n Migration Story

1. The server becomes the source of truth for user-facing copy instead of the client embedding one giant translation file.
2. Catalog delivery is layered so future locale, site, tenant, and experiment overrides can exist without another client rewrite.
3. The boot shell injects critical first-route namespaces so public pages and auth do not flash untranslated copy.
4. The client keeps the current i18n runtime and rendering model, but loads versioned catalogs from the server.
5. Catalogs are split by namespace so marketing, auth, chat, settings, billing, and dashboard copy can evolve independently.
6. Cache, fallback, and validation rules make locale loads resilient when the bridge, network, or publish state is bad.
7. The architecture leaves room for future DB- or CMS-backed copy management without changing the client contract.

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

- [x] Add one browser-level regression that verifies boot-injected `marketing` and `auth` catalogs render correctly on first paint for `/home`, `/pricing`, `/signup`, and `/` without raw keys, untranslated flashes, or a visible second copy swap after hydration.
	Added `TestExample100BootCatalogFirstPaintRegression` (`test/playwrightgo/examples/example100_boot_catalog_first_paint_regression_test.go`) to cover `/home`, `/pricing`, `/signup`, and `/` first-paint route hydration with selector-readiness, raw i18n-key/template-marker guards, and post-boot copy-stability checks under browser runtime error guardrails.
- [x] Add focused regressions for stub-removal parity so provider-backed memory extraction behaves consistently across OpenAI, Anthropic, and Cerebras paths, and so server-tool policy or execution failures surface one sane operator-visible error instead of hanging or silently no-oping.
	Added cross-provider memory extraction parity coverage (`TestMemoryExtractionParityAcrossProviders` in `server/provider/provider_memory_parity_test.go`) and one server-tool start-failure regression (`TestRunServerToolStartFailureEmitsOneRuntimeError` in `server/app/server_tool_stub_test.go`) so normalized memory-candidate contracts and operator-visible `start_failed`/`policy_denied` failure paths are asserted without silent hangs.
- [x] Update browser-smoke and happy-path QA coverage to use one stable seeded customer/admin credential pair, specifically `customer@email.com / password` and `admin@email.com / password`, so manual and automated QA stop depending on legacy `demo@example.com` fixtures.
	Updated `TestExample100RouteSmokePricingAuthDashboard` and `TestExample100AuthenticatedHappyPath` to use the stable seeded credential pair (`customer@email.com / password`), align auth entry with `/login`, and keep happy-path replay deterministic by seeding billing-customer scope in `cmd/seed-test-db` so `chat.send.enabled` entitlement resolution is present.
- [ ] Add one auth-entry regression matrix that proves local email/password login still works for quick QA after Google and external-identity work lands, covering signup, login, refresh, logout, password reset, and first-chat happy path with `customer@email.com / password`.
- [ ] Add one coexistence regression matrix for password plus external login so the same account can survive link-or-login decisions without duplicate users, accidental account merges, or broken session restoration across refresh and reopen flows.
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
	Added `TestExample100AdminJourneyRegression` (`test/playwrightgo/examples/example100_admin_journey_test.go`) to cover homepage boot, role-elevated admin-journey login (`admin@email.com` + seeded `su` role), dashboard deep-link entry visibility (`/app/dashboard`), slice route navigation (`/app/dashboard/usage`, `/app/admin/users`), back+refresh stability, and role-scoped dashboard/slice RPC checks (`GetAdminDashboard`, `ListAdminUsers`, `ListAdminConversations`) using the authenticated browser token.
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
	Added `TestExample100AuthenticatedHappyPath` to seed a deterministic DB, sign in as `customer@email.com`, create/send in a new thread, wait for streamed assistant output, refresh and assert the same `/app/thread/:publicID`, then reopen the thread route and verify prompt persistence.
- [x] Add browser-level smoke coverage for the pricing, auth, and dashboard routes.
	Added `TestExample100RouteSmokePricingAuthDashboard` to smoke `/pricing#faq`, unauthenticated `/app` auth inputs, and authenticated `/app/settings?panel=settings-profile` with zero browser console/page errors.
- [x] Add one browser-level pricing regression that covers `/pricing` load, plan-card copy, pricing FAQ, contact CTA targets, and the absence of any stale `free`, `unlimited`, or `no token caps` language once the usage-based pricing model lands.
	Added `TestExample100PricingRegression` (`test/playwrightgo/examples/example100_pricing_regression_test.go`) to validate `/pricing` route load, usage-based plan/FAQ/contact copy, and stale-language bans (`free`, `unlimited`, `no token caps`) with console/page runtime guardrails.
- [x] Add one browser-level billing-summary regression that covers the customer-facing billing/settings surfaces and verifies platform fee, raw usage cost, service premium, and total are shown with the same labels and formulas as the backend billing model.
	Added `TestExample100BillingSummaryRegression` (`test/playwrightgo/examples/example100_billing_summary_regression_test.go`) to authenticate via the current auth-entry route, load `/app/settings?panel=settings-billing`, and assert billing labels plus formula parity (`platform fee + usage + service premium = total`, `service premium = usage * premium%`) with nonzero platform-fee/premium policy visibility and browser runtime error guards.
- [x] Trace and fix the mismatch where the model select shows `GPT-5.4 - Best` while message bubbles report `GPT-5.4 mini`.
	Stream completion now normalizes the assistant-reported reply model, syncs `SelectedModel` to that canonical model when it differs, and persists the synced value via `SetSelectedModel` so picker state stays aligned with message metadata after first reply.
- [x] Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.
	`ParseScrollToBottom` now performs a delayed settle pass that snaps the message list to exact bottom after the initial smooth jump, and `TestExample100ScrollToBottomButton` was added to assert bottom-alignment after clicking `#scroll-to-bottom-btn`.
- [x] Update the app version number and ensure the displayed version string is sourced consistently.
	Version is now `v2026.03.27.2` from one shared source (`internal/buildinfo.GetBuildAppVersion`), with both the client badge (`sidebar`) and server boot-shell version label rendering from that same source.

### Agent 2: Security, auth, and billing enforcement

- [x] Add authz rules for server-owned catalogs so public namespaces like `marketing` and `auth` stay public, authenticated namespaces like `chat` and `settings` require session context, and admin namespaces fail closed outside the correct role boundary.
	Added typed catalog namespace read-authz rules in `server/app/catalog_namespace_authz.go`: public (`marketing`/`auth`) remains open, authenticated (`chat`/`settings`/`billing`) requires session auth, and admin namespaces are mapped to dashboard surface slice scopes so workspace-admin callers are restricted to allowed surfaces while superuser-only surfaces fail closed. Added coverage in `TestCatalogNamespaceReadScopeBoundaries` and `TestResolveCatalogNamespaceClass` (`server/app/catalog_namespace_authz_test.go`).
- [x] Add validation and safety guards so server-published strings cannot ship with malformed placeholders, missing template variables, or unauthorized override-layer writes.
	Added catalog publish guards in `server/app/catalog_publish_guard.go`: placeholder syntax validation (`parseValidateCatalogMessageTemplateSet`), base-layer placeholder parity checks (`parseValidateCatalogMessageTemplateParity`), and override-layer write authz (`parseRequireCatalogOverrideWriteScope`) that fail closed for immutable layers and unauthorized tenant/site writes. Added coverage and one micro-benchmark in `server/app/catalog_publish_guard_test.go`.
- [x] Replace the current server-tool mutation stubs with real security rules so `SetServerToolPolicy` and `RunServerTool` stay superuser-only, require fresh session plus explicit confirmation for dangerous changes, enforce one command whitelist plus argument policy, and fail closed with auditable denial reasons.
	Replaced the `Unimplemented` server-tool mutation stubs in `server/app/server_tool_stub.go` with real fail-closed authz/policy enforcement: `SetServerToolPolicy` now validates policy shape, requires explicit metadata confirmation for dangerous expansions, persists policy into `site_config`, and emits audit events for both denied and applied changes; `RunServerTool` now enforces start-frame, whitelist/prefix, argument regex, cwd policy, timeout caps, fresh-session superuser auth, emits typed policy/runtime error events, and writes auditable denial/runtime-block audit rows. Added focused tests + micro benches in `server/app/server_tool_stub_test.go` and policy helpers in `server/app/server_tool_security.go`.
- [x] Eliminate the remaining fail-open entitlement-store fallback in the send gate so missing billing state or missing store wiring blocks paid actions explicitly instead of silently allowing sends when runtime billing state is unavailable.
	Updated `parseRequireUserEntitlement` and `parseRequireUsageBudget` in `server/app/authz_entitlement.go` to fail closed with explicit `Unavailable` errors when billing/store wiring is missing, instead of silently allowing sends. Added focused coverage in `server/app/authz_entitlement_test.go` (`TestRequireUserEntitlementFailsClosedWhenStoreUnavailable`, `TestRequireUsageBudgetFailsClosedWhenStoreUnavailable`) and re-ran entitlement/usage decision tests.
- [x] Define one canonical external-identity security model so password, Google OIDC, generic OIDC, and later SAML are all login methods attached to one user account instead of separate account systems.
	Documented the canonical model in `README.md` under `Auth Architecture End State`: one `users` account with many linked login methods (`password`, `google_oidc`, workspace `oidc`, later `saml`), one shared session/token lifecycle, workspace-level auth policy resolution, and role-based post-login authz boundaries.
- [x] Define and enforce account-linking rules for external identities: verified-email match requirements, when an existing password user can link Google/OIDC on first login, when linking must be blocked, and how duplicate-account creation is prevented.
	Added a shared account-linking policy resolver seam in `server/app/auth_identity_linking.go` (`parseResolveExternalIdentityLinkDecision`) that fail-closes invalid/ambiguous provider-subject and email-match cases, requires verified email for first-link decisions, allows one password-user link bootstrap on verified single-email match, and blocks duplicate-account/conflict scenarios. Added focused rule coverage + one micro benchmark in `server/app/auth_identity_linking_test.go`.
- [x] Define and enforce workspace auth-policy rules so each workspace can express `password allowed`, `password blocked`, `external login allowed`, and `workspace SSO required`, while keeping one explicit local password path available for quick QA/dev testing.
	Added a strict workspace auth-policy resolver seam in `server/app/auth_workspace_policy.go` (`parseAuthorizeWorkspaceLoginMethod`) that enforces password-vs-external-vs-SSO-required rules, required-provider matching, and one explicit local QA password path override. Added focused rule tests + one micro benchmark in `server/app/auth_workspace_policy_test.go`.
- [x] Define and enforce how superuser/admin accounts authenticate under the new model, including whether privileged accounts may use social login, whether local password remains the break-glass path, and how fresh-session checks interact with external identities.
	Added canonical privileged-auth policy seams in `server/app/auth_privileged_policy.go`: social/external login can remain enabled for superuser and workspace-admin identities, but sensitive privileged mutations now require a fresh session plus local-password break-glass re-auth (`parseAuthorizePrivilegedMutationSession`). Wired this into `parseRequireSuperuserMutationUserID` (`server/app/superuser_control.go`) using JWT `amr` claims from `auth_service.go`, added focused policy tests + micro benchmark (`server/app/auth_privileged_policy_test.go`), and extended mutation freshness coverage for external-auth sessions in `server/app/server_tool_stub_test.go`.
- [x] Research the final superuser/admin external-auth policy before implementation so the repo has one explicit answer for whether privileged accounts may use Google/OIDC, whether local password stays the mandatory break-glass path, and how fresh-session re-auth should work after external login.
	Documented the resolved privileged external-auth policy in `README.md` under `Privileged External-Auth Decision (Resolved)`, including one explicit role/method matrix and mutation/fresh-session behavior. The documented decision is aligned with implemented policy seams in `server/app/auth_privileged_policy.go`, `server/app/superuser_control.go`, and JWT `amr` handling in `server/app/auth_service.go`.
- [x] Add OIDC/SAML handshake security rules for state, nonce, callback replay protection, return-to validation, and provider-subject binding so Google/OIDC/SSO login cannot be forged or replayed.
	Added fail-closed external-auth handshake guard seams in `server/app/auth_external_handshake.go`: start-path validation (`parseAuthorizeExternalHandshakeStart`) and callback validation (`parseAuthorizeExternalHandshakeCallback`) now enforce provider normalization, state+nonce matching, session-key binding, replay-block (`consumed_at`), expiration checks, strict return-to open-redirect validation, and optional provider-subject binding. Added focused rule tests and one micro benchmark in `server/app/auth_external_handshake_test.go`.
- [x] Add explicit authz rules for just-in-time provisioning and workspace membership assignment so domain-based SSO login, invites, and first-time enterprise sign-in cannot silently attach a user to the wrong workspace.
	Added explicit workspace-assignment authz resolver seams in `server/app/auth_workspace_provisioning.go`: `parseAuthorizeWorkspaceProvisioningAssignment` now fail-closes unverified-email assignment attempts, invite email/workspace mismatches, ambiguous multi-workspace domain matches, requested-workspace/domain mismatches, and JIT-disabled first-sign-in flows, while returning one typed assignment source (`invite`, `domain_jit`, `explicit_jit`) when allowed. Added focused regression coverage + one micro benchmark in `server/app/auth_workspace_provisioning_test.go`.
- [x] Define and enforce the role and scope matrix for the five dashboard surfaces so normal users see none of them, workspace admins only see scoped customer and workspace data, and superusers see platform-wide business, provider, and ops controls.
	Added explicit admin-surface matrix enforcement in `server/app/admin_scope.go`: workspace-admin callers are now fail-closed to scoped `customers/chats` slices while `business/providers/ops` slices require superuser scope; added focused verification in `TestAdminDashboardSurfaceRoleScopeMatrix` (`server/app/admin_dashboard_test.go`) plus regression checks for workspace-admin support/billing/workspace slice access.
- [x] Add server-side authz for `Business` settings mutations so plan edits, quota-policy changes, overage-rule changes, upgrade-trigger changes, and dunning changes require the correct superuser scope and fresh-session checks where needed.
	`Set/DeleteSuperuserBilling*` mutation RPCs are now superuser-gated with session freshness (`parseRequireSuperuserMutationUserID`) and explicit confirm+reason guards (`parseRequireSuperuserBillingMutationConfirmation`) across plan overages, quota policies, upgrade triggers, and dunning controls; added focused stale-session regression in `TestSuperuserBusinessMutationsRequireFreshSession` (`server/app/superuser_pricing_ops_test.go`).
- [x] Add server-side rules for the usage-based billing model so plan changes, pricing previews, invoice generation, and admin overrides all respect `platform fee + usage cost + service premium` instead of any stale flat-rate or unlimited assumptions.
	Added usage-based billing guard seams in `server/app/billing_formula_guard.go`: plan-write validation (`parseValidateUsageBasedBillingPlanWrite`), formula preview math (`parseBuildUsageBasedBillingPreview`), invoice line-type/math validation (`parseValidateUsageBasedBillingLineItemWrite`), and override validation (`parseValidateUsageBasedBillingOverrideWrite`). Wired them into plan mutations (`store_superuser_pricing.go`, `superuser_pricing_ops.go`), invoice line persistence (`store_billing.go`), and admin override writes (`admin_billing_ops.go`, `store_admin.go`), and added focused coverage + micro-benchmark in `server/app/billing_formula_guard_test.go`.
- [x] Add server-side rules that enforce the emotional plan boundary correctly: `Pro` stays single-operator/personal-workspace scoped, `Team` unlocks shared workspace/admin/collaboration capabilities, and `Enterprise` remains contract-gated.
	Added canonical plan-boundary validators in `server/app/billing_formula_guard.go` and wired them into pricing writes (`store_superuser_pricing.go`, `superuser_pricing_ops.go`): `Pro` now fails closed if it enables multi-seat/team/SSO traits, `Team` requires collaboration-capable seat ranges, and `Enterprise` remains contract-sized + SSO-capable. Entitlement writes now enforce matching boundary rules for `workspace.multi_user.enabled` and `sso.enabled`; coverage added in `server/app/billing_formula_guard_test.go`.
- [x] Add server-side authz for `Customers` mutations so user disable or restore, workspace suspend or restore, entitlement overrides, and billing overrides respect workspace scope, platform scope, and reason plus confirmation requirements.
	Customer mutation authz is now enforced across disable/restore and suspend/restore (`server/app/admin_mutation_authz.go`, `server/app/admin_mutation_effects.go`) plus billing overrides (`server/app/admin_billing_ops.go`) with scoped caller checks, explicit confirmation, and required reasons; coverage includes `TestAuthorizeAdminMutationActionScopes`, `TestAdminMutationDisableAndRestoreEnforcesImmediateAuthEffects`, `TestAdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects`, and `TestAdminBillingOverrideMutationsRequireReasonAndConfirmation` (`server/app/*_test.go`).
- [x] Add server-side authz for `Chats` settings mutations so system-default prompt changes, model-default changes, memory-rule changes, onboarding-template changes, and workflow or skill publishing changes respect the correct operator boundary.
	Added explicit chat-settings mutation authz seams in `server/app/admin_chat_mutation_authz.go`: system-default prompt/model/memory mutations are superuser-only, while onboarding-template/workflow/skill mutations are workspace-scoped (or superuser) and fail closed when workspace scope is missing/out-of-scope; coverage in `TestAdminChatMutationScopeBoundaries` (`server/app/admin_chat_mutation_authz_test.go`).
- [x] Add server-side authz for `Providers` settings mutations so provider enable or disable, model visibility, fallback routing, per-provider limits, and cost-guardrail changes fail closed outside superuser scope.
	Added provider mutation authz seam `parseAuthorizeAdminProviderMutationScope` in `server/app/admin_provider_mutation_authz.go` that fail-closes provider enable/disable, model visibility, fallback routing, provider limits, and cost-guardrail actions outside platform superuser scope; coverage in `TestAdminProviderMutationScopeBoundaries` (`server/app/admin_provider_mutation_authz_test.go`).
- [x] Add server-side authz for `Ops` settings mutations so site-config changes, feature-flag changes, retention-policy changes, webhook-behavior changes, and integration-setting changes are gated and audited consistently.
	Added unified ops-mutation authz seam `parseAuthorizeAdminOpsMutationScope` in `server/app/admin_ops_mutation_authz.go` with superuser-only gates for site-config, feature-flag, retention-policy, webhook-behavior, and integration-setting actions; existing control-plane mutation RPCs continue to emit audit events and boundary coverage now includes `TestAdminOpsMutationScopeBoundaries` plus `TestAdminControlMutationScopeBoundaries` (`server/app/*_test.go`).
- [x] Add privacy and redaction rules for dashboard detail RPCs so customer chats, support data, billing records, and provider error payloads expose only the minimum operator-visible fields for each role.
	Added shared workspace-scope redaction helpers in `server/app/admin_redaction.go` and wired them into admin dashboard/support/billing responses: workspace-admin views now redact conversation previews, provider request/trace diagnostics and raw error payloads in usage rows, session identifiers/IPs, support ticket/message bodies (internal message content), audit payload JSON, and billing event payload JSON; coverage is in updated workspace-scope tests including `TestAdminDashboardRPCsWorkspaceAdminScope`, `TestAdminUserControlRPCsWorkspaceAdminScope`, `TestAdminSupportTriageRPCsWorkspaceAdminScope`, and `TestAdminBillingEventsWorkspaceScopeRedactsPayload`.
- [x] Add fresh-session and explicit-confirmation rules for high-risk `Business` mutations so plan edits, overage edits, and dunning actions cannot be triggered from a stale dashboard tab.
	High-risk superuser business mutations (`Set/DeleteSuperuserBillingPlanOverage`, `Set/DeleteSuperuserBillingQuotaPolicy`, `Set/DeleteSuperuserBillingUpgradeTrigger`, `Set/DeleteSuperuserBillingDunningEvent`) now require both fresh session auth (`parseRequireSuperuserMutationUserID`) and explicit confirm+reason (`parseRequireSuperuserBillingMutationConfirmation`), with stale-session regression coverage in `TestSuperuserBusinessMutationsRequireFreshSession` and confirm/role coverage in `TestSuperuserPricingControlRPCs` (`server/app/superuser_pricing_ops_test.go`).
- [x] Add scope and redaction rules for unified `Customers` account timelines so chat content, billing records, support notes, auth sessions, and audit events expose only role-appropriate fields.
	Added typed unified timeline scope contract in `server/app/admin_customer_timeline_scope.go` (`parseAuthorizeAdminCustomerTimelineScope`) covering chat content, billing records, support notes, auth sessions, and audit events with fail-closed target-user scope checks and explicit redaction requirement signaling for workspace-admin callers; coverage in `TestAdminCustomerTimelineScopeRules` (`server/app/admin_customer_timeline_scope_test.go`) and runtime redaction is wired through shared helpers in `server/app/admin_redaction.go`.
- [x] Add transcript and tool-trace visibility rules for `Chats` drill-downs so workspace admins can inspect in-scope failures without gaining global conversation access or hidden provider payloads.
	Added typed chat drill-down scope contract in `server/app/admin_chat_drilldown_scope.go` (`parseAuthorizeAdminChatDrilldownScope`) so superusers can view full transcript/tool-trace payloads while workspace-admin callers are constrained to in-scope users with transcript redaction and tool-trace payload suppression; verified in `TestAdminChatDrilldownScopeRules` (`server/app/admin_chat_drilldown_scope_test.go`) and enforced redaction for usage trace diagnostics via `server/app/admin_redaction.go`.
- [x] Add blast-radius authorization rules for `Providers` mutations so provider disable, fallback-routing changes, and model visibility changes require the correct role and return previewable affected-scope counts.
	Extended provider mutation authz in `server/app/admin_provider_mutation_authz.go` with `parseAuthorizeAdminProviderMutationWithBlastRadius`, which enforces superuser-only provider mutations and returns previewable affected workspace/user counts from current store scope; verified in `TestAdminProviderMutationScopeBoundaries` and `TestAdminProviderMutationBlastRadiusPreview` (`server/app/admin_provider_mutation_authz_test.go`).
- [x] Add operator-action authz rules for `Ops` retries and replays so incident updates, job retries, notification retries, and webhook replays are separately gated, audited, and fail closed outside the correct scope.
	Added typed ops action authz/audit seam in `server/app/admin_ops_action_authz.go`: incident updates now support workspace-admin in-scope gating while job retries, notification retries, and webhook replays are superuser-only; included explicit audit event-key mapping (`parseResolveAdminOpsActionAuditEventType`) and fail-closed unsupported-action behavior with coverage in `TestAdminOpsActionScopeBoundaries` and `TestResolveAdminOpsActionAuditEventType` (`server/app/admin_ops_action_authz_test.go`).
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

- [x] Add a typed server-owned catalog contract with namespace, locale, version, fallback-locale, source-layer metadata, and message payload fields so the client is not coupled to ad hoc copy transport.
	Added typed catalog contract protobuf messages (`CatalogSourceMetadata`, `CatalogMessagePayload`, `CatalogNamespacePayload`) plus deterministic server-side contract builders/hash normalization in `server/app/catalog_contract.go`. Coverage: `TestBuildCatalogNamespacePayloadNormalizesContract` and `TestBuildCatalogNamespacePayloadContentHashStable` (`server/app/catalog_contract_test.go`).
- [x] Move the current client-owned strings into server-owned sources behind a loader interface that starts file- or Go-backed now and can later swap to DB/CMS storage without changing the client contract.
	Moved the localization bundle registrations out of `client/app/i18n.go` into a new server-owned source package (`server/catalog/bundle.go`) and switched the client runtime to consume `catalog.BuildBundle()`. Added a pluggable server-side loader seam in `server/app/catalog_loader.go` (`parseCatalogSource` + Go-backed default + swappable source stub coverage) so future file/DB/CMS sources can slot in without changing the catalog contract.
- [x] Add typed server endpoints for boot-catalog injection and lazy namespace fetch, plus version/hash metadata so the client can cache catalogs safely and refetch only when content changes.
	Added typed catalog RPCs in protobuf and server app (`GetCatalogBootstrap`, `GetCatalogNamespace`) with version/hash cache metadata (`bundle_version`, `bundle_hash`, per-namespace `content_hash`) and `is_not_modified` short-circuit responses for known version/hash requests. Coverage: `TestGetCatalogBootstrapRPC`, `TestGetCatalogBootstrapRPCNotModified`, and `TestGetCatalogNamespaceRPC` (`server/app/catalog_ops_test.go`).
- [x] Replace the legacy local seed identities in `cmd/seed-test-db` with one stable QA pair, `customer@email.com / password` and `admin@email.com / password`, and keep any required role-grant/bootstrap helpers aligned so local auth, billing, first-chat, and dashboard smoke flows all start from the same known accounts.
	Updated `cmd/seed-test-db` to seed `customer@email.com` and `admin@email.com` with password `password`, aligned `tools/gwc seed` credential summaries, and switched seeded-credential Playwright regressions (auth/first-chat/billing/route/admin flows) to the same QA pair. Coverage: `go test ./examples/100-ai-chat-wizard/cmd/seed-test-db -count=1`, `go test ./tools/gwc -count=1 -run "^(TestRunSeedJSONExecutesDefaultSeederWithKnownCredentials|TestPrintSeedSummary)$"`, and compile check `go test -c -tags playwrightgo -o ./bin/playwright_examples.test ./test/playwrightgo/examples`.
- [x] Implement `SetServerToolPolicy` end to end: persist policy fields and approved tool rules, add typed store funcs and SQL for create/read/update history, emit audit rows, and return the applied policy snapshot instead of `Unimplemented`.
	Implemented full `SetServerToolPolicy` mutation flow with request validation + dangerous-change confirmation + site-config persistence + immutable history snapshots (`server_tool_policy_history`) and audit rows. Added typed store funcs and SQL (`create/list_server_tool_policy_history.sql`, `store_server_tool_policy.go`) and extended the RPC response to return the applied policy snapshot fields (`is_enabled`, `max_session_seconds`, `max_output_bytes`, `approved_tools`, `source`). Coverage: `go test ./examples/100-ai-chat-wizard/proto -count=1` and `go test ./examples/100-ai-chat-wizard/server/app -count=1 -run "^(TestGetServerToolPolicyRequiresSURole|TestGetServerToolPolicyReturnsDefaults|TestSetServerToolPolicyRequiresDangerousChangeConfirmation|TestSetServerToolPolicyValidatesRuleArgumentPolicy|TestSetServerToolPolicyAppendsHistoryRows|TestRunServerToolRequiresStartPayload|TestRunServerToolEnforcesWhitelistAndArgumentPolicy|TestRunServerToolAllowedCommandStreamsStartedAndExit|TestSensitiveSuperuserMutationsRequireFreshSession|TestRunServerToolRequiresStream|TestGetServerToolPolicyReadsWhitelistFromSiteConfig)$"`.
- [x] Implement `RunServerTool` end to end: add a typed terminal-session manager, enforce start/stdin/signal/close semantics from the existing proto, stream stdout/stderr/exit events, enforce byte/time/session caps from policy, persist execution audit state, and shut down cleanly on disconnect or policy violation.
	Implemented a typed runtime session manager (`server_tool_runtime.go`) and wired `RunServerTool` to execute policy-authorized commands with started/stdout/stderr/exit/error events, stdin/signal/close control frames, output-byte cap enforcement, timeout-based cancellation, and disconnect/context shutdown handling. Added audit-state events for started/completed/failed runs and focused stream-behavior coverage (`TestRunServerToolAllowedCommandStreamsStartedAndExit`, `TestRunServerToolStdinRequiresMatchingSessionID`) in `server_tool_stub_test.go`.
- [x] Replace the stale control-mutation authz stubs in `admin_control_mutation_authz.go` with one live shared helper used by the real feature-flag, experiment, and incident mutation RPCs, then remove the now-misleading `Unimplemented` stub behavior and tests that only verify the placeholder seam.
	Removed the stale `parseExecute*ControlMutation` `Unimplemented` helpers from `admin_control_mutation_authz.go` and kept `parseAuthorizeAdminControlMutationScope` as the shared live authz boundary used by `SetAdminFeatureFlag`, `RollbackAdminExperiment`, `UpdateAdminIncident`, and `GetAdminIncidentBlastRadius`. Reworked `admin_control_mutation_authz_test.go` to validate real scope outcomes directly (superuser platform scope, workspace-admin incident scope, and fail-closed cases) instead of asserting placeholder `Unimplemented` behavior.
- [x] Implement Anthropic memory extraction so `ParseExtractUserMemories` uses the provider’s structured-output/tooling path to return the same normalized candidate shape as OpenAI, including stable score/category parsing and safe fallback behavior when the provider response is malformed.
	`AnthropicProvider.ParseExtractUserMemories` now uses an explicit tool-choice + JSON-schema tool contract (`extract_user_memories`) with tool-first parsing and text compatibility fallback (`server/provider/anthropic_provider.go`), normalizes category/score/key bounds via `parseNormalizeMemoryCandidates`, and safely falls back to an empty candidate list when provider output is malformed. Added HTTP-backed request/response coverage for tool schema + malformed fallback (`server/provider/provider_http_additional_test.go`) and a focused micro benchmark (`server/provider/anthropic_memory_benchmark_test.go`). Also added JSON tags on `provider.UserMemoryCandidate` (`server/provider/provider.go`) so snake_case extraction payloads map to stable `usefulness_score` / `confidence_score` fields correctly.
- [x] Implement Cerebras memory extraction so `ParseExtractUserMemories` produces the same normalized candidate contract as OpenAI, either through JSON-schema prompting on the current API surface or through one deterministic parser/fallback path that keeps memory extraction provider-agnostic.
	Implemented deterministic Cerebras memory extraction in `server/provider/cerebras_provider.go`: `ParseExtractUserMemories` now issues a JSON-object constrained chat-completion request (`response_format.type = json_object`), parses and fallback-extracts one `{"memories":[...]}` payload (`parseResolveCerebrasMemoryCandidates`), and returns normalized candidates through the shared `parseNormalizeMemoryCandidates` contract. Updated streaming helper expectations for the no-network blank-input path in `server/provider/provider_streaming_additional_test.go`, added HTTP-backed extraction coverage (request shape + parse failure fallback) in `server/provider/provider_http_additional_test.go`, and added one micro benchmark in `server/provider/cerebras_memory_benchmark_test.go`.
- [ ] Add first-class external-identity tables and store funcs, including at minimum `auth_identities` for provider-linked identities, `auth_oidc_states` for login handshakes, and one auth-policy table or equivalent config source for password/external-login/SSO requirements by workspace.
- [ ] Add one canonical provider model for external auth with `provider_key`, `provider_type`, subject identifier, verified email, profile payload, and last-login timestamps so Google login and enterprise OIDC reuse the same persistence contract.
- [ ] Implement Google OIDC as the first external provider with typed start/callback handlers, state+nonce persistence, verified-email extraction, account link-or-create logic, and final session issuance through the existing auth-session/token path.
- [ ] Add one generic OIDC provider path for workspace auth so enterprise providers can be configured without hardcoding Google-specific behavior into the core auth pipeline.
- [ ] Rework or extend `workspace_sso_configs` so the data model can support generic OIDC configuration cleanly, and treat the current SAML fields as either one provider subtype or a later-phase compatibility path instead of the only enterprise shape.
- [ ] Add typed workspace auth-policy resolution so login can answer: is password allowed here, is external login optional, is workspace SSO required, which provider should be used, and should just-in-time membership creation be allowed.
- [ ] Implement link-or-create account resolution so an external identity can attach to an existing password user safely when policy allows it, create a new user when appropriate, and reject ambiguous or unsafe matches.
- [ ] Add typed audit events and queryable auth records for external login start, callback success, callback failure, identity linked, identity unlinked, policy-denied login, and workspace SSO enforcement decisions.
- [ ] Keep local email/password fully operational as a first-class auth path for quick QA and local operator testing, including stable seed accounts, normal session issuance, password reset, and no mandatory external-provider dependency in local development.
- [ ] Add one typed customer-safe error contract for chat/auth/settings/dashboard failures that includes a friendly user message, a stable support or request ID derived from existing request/correlation metadata, and one server-log correlation path so support can look up the failure quickly without exposing raw internals to customers.
- [x] Add typed `Business` dashboard summary and drill-down RPCs backed by typed SQL queries over `billing_customers`, `billing_subscriptions`, `billing_invoices`, `billing_invoice_line_items`, `billing_access_overrides`, `billing_events`, `usage_events`, `product_analytics_events`, and `subscription_churn_feedback`.
	Completed by wiring the existing typed summary RPC (`GetAdminDashboard`) with a new typed drill-down RPC (`GetAdminBusinessDrilldown`) that joins user-scoped billing customer/subscription/invoice/line-item/override/event slices plus usage, product analytics, and churn feedback rows (`server/app/admin_business_ops.go`, `server/app/store_growth_ops.go`, `sql/store/list_product_analytics_events_by_user.sql`, `sql/store/list_subscription_churn_feedback_by_customer.sql`, `proto/chat.proto`). Coverage: `TestGetAdminBusinessDrilldownRPCs` and `TestGetAdminBusinessDrilldownScopeGuards` (`server/app/admin_business_ops_test.go`).
- [x] Add typed `Business` mutation RPCs, store funcs, and SQL queries for billing plans, plan entitlements, quota policies, overage rules, upgrade triggers, and dunning controls, including explicit audit-log writes for each mutation.
	Completed by adding typed superuser billing-plan and billing-plan-entitlement mutation RPCs plus store/query SQL seams (`Set/DeleteSuperuserBillingPlan`, `Set/DeleteSuperuserBillingPlanEntitlement`, `list/upsert/delete_billing_plan*.sql`) and explicit admin audit events for each new mutation; existing typed quota/overage/upgrade/dunning mutation seams remain active and audited in `superuser_pricing_ops.go`. Coverage: `TestStoreSuperuserPricingControlFuncs`, `TestSuperuserPricingControlRPCs`, and `TestSuperuserBusinessMutationsRequireFreshSession` (`server/app/superuser_pricing_ops_test.go`).
- [x] Replace the seeded plan model and admin pricing controls so the canonical plans are `pro`, `team`, and `enterprise` only, removing stale public `free` positioning and aligning seeded plan metadata with the real product story.
	Completed by removing `free` from seeded billing plans/entitlements/model-access/overage/quota/upgrade controls in both schema and migrations, switching fallback/default plan resolution to `pro` (`list_billing_effective_*`, workspace defaults, admin audit bootstrap workspace), and keeping legacy test call sites stable via a seed-plan alias helper that maps `free -> pro` in test fixtures (`testkit_test.go`, `benchmark_test.go`).
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

### Agent 4: Docs, test engineering, flow definition, and operator guidance

- [x] Fix the route-model docs so `README.md`, `FLOWS.md`, `MANUAL_SMOKE.md`, and `OPERATOR_RUNBOOK.md` all reflect the current browser truth: dedicated `/app/dashboard` and `/app/dashboard/*` routes exist now, and the old â€œno dedicated dashboard route yetâ€ wording is stale.
	Checkpoint: completed route-model wording alignment across `README.md`, `FLOWS.md`, `MANUAL_SMOKE.md`, and `OPERATOR_RUNBOOK.md`; validated stale wording removal and `/app/dashboard*` presence with targeted `rg` checks; result matches current route truth; residual risk is broader terminology drift in other docs; next suggested todo is seeded-account/superuser local-path clarification.
- [x] Fix the local operator docs around seeded accounts and role grants so they stop implying `admin@example.com` is automatically a working superuser; document exactly what `cmd/seed-test-db` creates today, what it does not create, and the current local path for verifying superuser-only flows.
	Checkpoint: documented seed-create vs seed-do-not-create scope in `README.md` and `OPERATOR_RUNBOOK.md`, added superuser-verification path and optional local SQL role-grant flow, and aligned `MANUAL_SMOKE.md`; validated with targeted `rg` checks for role-grant wording and stale-route wording; result clarifies real local authz behavior; residual risk is deeper role matrix drift outside these docs; next suggested todo is expanding the environment-variable runtime contract.
- [x] Expand the environment-variable reference to cover the real runtime contract used by the server: `CHAT_ENV`, `CHAT_AUTH_SECRET`, `CHAT_ALLOW_INSECURE_AUTH_FALLBACK`, `CHAT_AUTH_SIGNING_KEYS`, `CHAT_AUTH_ACTIVE_KID`, `CHAT_MODEL`, `OPENAI_MODEL`, provider keys/stubs, `CHAT_DB_PATH`, and usage-premium behavior.
	Checkpoint: expanded `README.md` environment-variable contract to match runtime parsing and auth/provider behavior, including signing-key rotation fields and usage-premium semantics; validated key coverage with `rg` against docs and runtime env usage; result gives one operator-facing source of truth; residual risk is unrelated server/app compile failures blocking package-test validation in current worktree; next suggested todo is adding startup-and-troubleshooting guidance.
- [x] Add one focused startup-and-troubleshooting guide for common local failures: missing submodule, wrong `CHAT_DB_PATH`, missing auth secret in production mode, missing provider keys without stubs, stale WASM artifacts, and tunnel/health-check failures.
	Checkpoint: added `OPERATOR_RUNBOOK.md` section `Startup and Troubleshooting Guide` with failure->confirm->fix entries for submodule, DB path, auth secret policy, provider keys/stubs, stale WASM, and tunnel/health checks plus quick probe commands; validated with targeted `rg` checks; result gives one focused local failure playbook; residual risk is environment-specific failure signatures not covered in table; next suggested todo is adding an operator diagnostics evidence guide.
- [x] Add one operator diagnostics guide that tells maintainers where to look for evidence during failures: browser console, `gwc ... status -json`, `/healthz`, runtime logs under `bin/runtime/logs/`, and the server/admin diagnostic log lines already emitted by auth, bootstrap, dashboard, and mutation flows.
	Checkpoint: added `OPERATOR_RUNBOOK.md` section `Operator Diagnostics Guide` with an evidence matrix and fast triage commands covering browser console, managed status, `/healthz`, runtime logs, and auth/bootstrap/dashboard/mutation log prefixes; validated with targeted `rg` checks; result gives maintainers one diagnostics-first flow; residual risk is log-prefix drift if RPC names change; next suggested todo is documenting the role model and verification matrix.
- [x] Document the current role model and verification matrix in plain operator language: normal user vs workspace admin vs superuser, which routes and RPCs each role can reach today, and which parts of the dashboard are still partially implemented or read-heavy.
	Checkpoint: added `OPERATOR_RUNBOOK.md` section `Role Model and Verification Matrix` with per-role route/RPC reach and explicit read-heavy/partial dashboard status notes; validated matrix anchors with targeted `rg` checks; result gives operators one current authz truth table; residual risk is behavior drift as pending admin routes/mutations land; next suggested todo is adding a local-truth vs planned-future-state section.
- [x] Add one â€œlocal truth vs planned future stateâ€ section so maintainers can distinguish what is already real in example 100 from what is still roadmap material across admin surfaces, workspace-admin flows, and server-owned i18n.
	Checkpoint: added `README.md` section `Local Truth vs Planned Future State` with explicit now-vs-next mapping for admin surfaces, workspace-admin scope, superuser scope, and server-owned i18n migration; validated anchors with targeted `rg` checks; result gives maintainers a clear reality-vs-roadmap lens; residual risk is drift as roadmap items land quickly; next suggested todo is terminology/stale-wording audit normalization.
- [x] Audit the example docs for stale route names, stale page names, and stale pricing/admin wording after the recent dashboard, pricing-model, and public-IA changes, then normalize terminology across README, flows, smoke docs, and runbook.
	Checkpoint: normalized stale wording across `FLOWS.md` and `MANUAL_SMOKE.md` (workspace-admin scope language, dashboard route wording, and seeded-admin label) and re-verified consistent admin/pricing vocabulary across `README.md`, `FLOWS.md`, `MANUAL_SMOKE.md`, and `OPERATOR_RUNBOOK.md`; validated via targeted `rg` stale-term sweeps; result aligns docs with current dashboard/authz truth; residual risk is future terminology drift as IA work continues; next suggested todo is testing-gap matrix documentation.
- [x] Replace all local QA-account references in `README.md`, `MANUAL_SMOKE.md`, `OPERATOR_RUNBOOK.md`, and any example-100 flow docs so they consistently use `customer@email.com / password` for customer checks and `admin@email.com / password` for admin checks, with one explicit note about any extra local superuser grant step.
	Checkpoint: normalized QA credential references across `README.md`, `MANUAL_SMOKE.md`, and `OPERATOR_RUNBOOK.md` to `customer@email.com / password` and `admin@email.com / password`, while preserving local seed-equivalent mapping and explicit superuser-grant notes; validated via targeted `rg` sweeps; result makes operator credentials consistent across docs; residual risk is future docs reintroducing legacy sample accounts; next suggested todo is the testing-gap matrix.
- [x] Create one explicit testing-gap matrix for example 100 that maps each shipped feature area to current package tests, Playwright coverage, manual smoke coverage, and uncovered risk so maintainers can see where bug-searching should focus next.
	Checkpoint: added `MANUAL_SMOKE.md` section `Testing Gap Matrix (Shipped Areas)` with feature-area mapping across package tests, Playwright coverage, manual smoke coverage, and uncovered risk seams; validated table anchors and risk-column presence via `rg`; result gives maintainers a prioritized bug-hunt lens; residual risk is stale test-name drift as suites evolve; next suggested todo is Playwright naming/status drift audit planning.
- [ ] Add focused regressions for lazy namespace loading so opening settings, billing, and dashboard routes fetches the required server catalogs once, preserves route/query state, and survives refresh with the expected cached version.
- [ ] Add one auth-entry regression matrix that proves local email/password login still works for quick QA after Google and external-identity work lands, covering signup, login, refresh, logout, password reset, and first-chat happy path with `customer@email.com / password`.
- [ ] Add one workspace-SSO regression matrix that covers optional SSO vs required SSO, showing correct redirects or denies for enterprise users while keeping the explicit local password QA/dev path working whenever policy allows it.
- [ ] Add one callback-route regression suite for Google/OIDC flows so invalid state, expired nonce, denied consent, callback error, and successful callback completion all land on sane auth screens without broken history or half-booted app state.
- [x] Audit the Playwright suite for naming/status drift so files still ending in `_pending_test.go` are classified as truly pending vs already-real regressions, then document the rename or recategorization plan to make bug coverage discoverable.
- [x] Add one stub-removal status section that lists the remaining intentional implementation placeholders in example 100, marks which ones were converted to live behavior (`server tools`, `provider memory extraction`, `entitlement fail-open guard`, `control-mutation authz helper`), and keeps docs from claiming placeholder behavior after the code is real.
	Checkpoint: added `README.md` section `Stub-Removal Status` with seam-by-seam status for server tools, provider memory extraction, entitlement fail-open guard, and control-mutation authz helper, including explicit current-truth file anchors; validated section presence and seam coverage via `rg`; result prevents stale placeholder claims in operator docs; residual risk is status drift if code paths change without doc updates; next suggested todo is the server-tools operator verification checklist.
- [x] Add one stub-removal status section that lists the remaining intentional implementation placeholders in example 100, marks which ones were converted to live behavior (`server tools`, `provider memory extraction`, `entitlement fail-open guard`, `control-mutation authz helper`), and keeps docs from claiming placeholder behavior after the code is real.
	Checkpoint: duplicate checklist entry closed using the same `README.md` `Stub-Removal Status` section added above; no additional file edits required beyond existing seam-status table; validated duplicate closure in `TODO.md`; result removes redundant open work; residual risk is future duplicate todo drift; next suggested todo is the server-tools operator verification checklist.
- [x] Add one operator verification checklist for the server-tools rollout covering policy read/write, whitelist enforcement, fresh-session re-auth behavior, stdout/stderr/exit streaming, audit-log evidence, and shutdown-on-disconnect so maintainers can verify the replacement is safe before calling it ready.
	Checkpoint: added `OPERATOR_RUNBOOK.md` section `Server-Tools Rollout Verification Checklist` with six required checks (policy gates, whitelist enforcement, fresh-session re-auth, stream lifecycle, audit evidence, disconnect cleanup) plus minimum pre-ready pass criteria; validated keyword coverage with targeted `rg`; result gives maintainers one concrete go/no-go list for server-tools rollout; residual risk is checklist drift when RPC contracts evolve; next suggested todo is provider-memory extraction parity contract notes.
- [x] Add one provider-memory extraction parity note that documents the intended shared contract across OpenAI, Anthropic, and Cerebras: candidate shape, score semantics, malformed-response fallback behavior, and what users should expect when one provider cannot extract stable memories from a message.
	Checkpoint: validated existing `README.md` section `Provider Memory Extraction Parity Contract` for candidate shape, score semantics, malformed fallback, unsupported-provider behavior, and user-visible expectations; validation run was targeted `rg` for section and contract terms; result confirms parity contract is now documented; residual risk is provider implementation drift without doc refresh; next suggested todo is release-caveat alignment.
- [x] Add one release-note and upgrade-caveat section for removing placeholder behavior, specifically the entitlement fail-open fallback and the old control-mutation authz stubs, so maintainers know which local/dev assumptions and old tests need to change when those paths go live.
	Checkpoint: validated existing `README.md` section `Release Notes and Upgrade Caveats (Placeholder Removal)` for entitlement fail-open and control-mutation authz caveats plus release checklist; validation run was targeted `rg` for both placeholder seams and checklist text; result documents migration impacts for local/dev/tests; residual risk is rollout-date drift if commits are not recorded; next suggested todo is auth-architecture end-state note.
- [x] Add one auth-architecture note that explains the intended end state plainly: one user account, many linked login methods, workspace-level auth policy, and one shared session/token layer reused by password, Google, OIDC, and later SAML.
	Checkpoint: added `README.md` section `Auth Architecture End State` with one-account/multi-identity model, workspace policy gate, and shared session-token path; validation run was targeted `rg` for end-state anchors; result gives a canonical auth target model; residual risk is runtime details changing before external auth lands; next suggested todo is explicit external-auth policy decisions.
- [x] Document the exact product-policy decisions for external auth before implementation proceeds: whether same-email accounts auto-link, whether workspace-required SSO blocks password, how multi-workspace users behave when policies conflict, and what the break-glass path is for superusers.
	Checkpoint: added `README.md` table `External Auth Product Policy Decisions` covering same-email auto-linking, required-SSO password denial, multi-workspace conflict resolution, and superuser break-glass policy; validation run was targeted `rg` for each policy row; result locks policy assumptions before backend implementation; residual risk is future policy reversals requiring coordinated test updates; next suggested todo is preserving local password rollout guardrail.
- [x] Add one rollout note that explicitly preserves local email/password for quick QA, seeded local testing, and offline development even after Google/OIDC work lands, so no future pass accidentally removes the simplest operator path.
	Checkpoint: added `README.md` subsection `Local Password Rollout Guardrail` with explicit QA/seeded/offline requirements and regression rule for password path removal; validation run was targeted `rg` for guardrail terms and seeded credential references; result protects fast local operator path through external-auth rollout; residual risk is enforcement depends on future test coverage; next suggested todo is local external-auth runbook split.
- [x] Add one operator runbook section for local external-auth testing that distinguishes `quick QA path = email/password` from `provider-integrated path = Google/OIDC`, including which env vars, callback URLs, and local test accounts are required for each.
	Checkpoint: added `OPERATOR_RUNBOOK.md` section `13) Local External-Auth Testing (Planned Rollout)` with separate quick-QA and provider-integrated paths, env var sets, callback URL checks, and required local accounts; validation run was targeted `rg` for both path labels and callback/env anchors; result gives operators a clear split test workflow; residual risk is env-var naming may change when handlers are implemented; next suggested todo is auth-provider capability matrix.
- [x] Add one auth-provider capability matrix covering password, Google OIDC, generic workspace OIDC, and future SAML, including current status (`planned`, `config-only`, `fully wired`), target users, and required backend/UI/runtime pieces.
	Checkpoint: added `README.md` section `Auth Provider Capability Matrix` with rows for password, Google OIDC, generic workspace OIDC, and SAML plus status/target-user/backend-ui requirements; validation run was targeted `rg` on matrix rows and status labels; result makes provider readiness and dependency scope explicit; residual risk is status drift as implementation lands; next suggested todo is external-auth manual-smoke checklist.
- [x] Add one research note that compares `marketed capability` vs `implemented capability` for Google login, enterprise SSO, and broader trust/auth features so docs stop overstating support before the runtime exists.
	Checkpoint: added `README.md` subsection `Marketed Capability vs Implemented Capability (Auth/Trust)` with side-by-side truth table for Google login, enterprise SSO, account linking, and trust-control coverage; validation run was targeted `rg` for section title and capability rows; result aligns claims with current runtime reality and adds wording guardrails; residual risk is drift if runtime lands before docs update; next suggested todo is continuing next unchecked item outside Agent 4.
- [x] Research how to bring Anthropic and Cerebras memory extraction to parity with OpenAI, including whether each provider supports reliable structured output strongly enough or whether one provider-agnostic extraction fallback is required for consistent remembered-preferences behavior.
	Documented the resolved provider parity strategy in `README.md` under `Memory Extraction Provider Parity Decision (Resolved)`: OpenAI uses strict JSON-schema responses, Anthropic uses strict tool-use schema with text compatibility fallback, and Cerebras uses JSON-object constrained chat completions with deterministic parser fallback; all converge through the same normalized `UserMemoryCandidate` contract and shared normalization helpers.
- [ ] Research the external-auth data model before implementation and compare at least two schema shapes for identity-linking plus workspace auth policy so Google login, generic OIDC, local password, and future SAML do not force a second migration later.
- [ ] Defer full SAML runtime until after Google and generic OIDC are stable, but add one concrete backend migration note or TODO marker showing which current `workspace_sso_configs` fields are enterprise-config storage only versus actual active login runtime.
- [ ] Add one browser-level regression that forces auth, chat send, settings save, and dashboard-load failures and verifies the customer sees calm actionable copy plus a visible support or request ID instead of raw gRPC descriptions, stack traces, or internal error strings.
- [ ] Add one browser-level regression that exercises client-side error-boundary and fallback states for route boot, panel fetch, and mutation submit paths and verifies each fallback still emits one correlated diagnostic log entry instead of failing silently or blanking the screen.
- [ ] Add one explicit log-redaction policy and enforcement seam for auth secrets, passwords, API keys, bearer tokens, cookies, provider credentials, webhook secrets, raw provider payloads, and internal note bodies so none of those values can leak into customer-visible errors or operator logs by accident.
- [ ] Add one support-ID exposure policy that defines which identifiers may be shown to customers, which stay operator-only, and how request IDs, correlation IDs, audit IDs, and session IDs are mapped so traceability improves without exposing sensitive internals.
- [ ] Add fail-closed sanitization for validation, authz, and upstream-provider errors so public responses never leak raw SQL errors, provider payloads, role-scope internals, filesystem paths, or environment-derived secrets even when the server logs keep the original failure details.
- [ ] Add one traceability contract for privileged and customer-impacting flows so auth, chat send, settings save, dashboard reads, and admin mutations all carry request/correlation metadata, actor scope, and target scope through logs and audit events consistently.
- [ ] Audit variable-name quality in auth, billing, admin, runtime, and logging hot paths and rename low-signal locals like `data`, `value`, `item`, `resp`, `tmp`, and ambiguous `err` chains to domain-specific names that follow the repo naming rules without changing behavior.
- [ ] Add one shared error-envelope builder for HTTP, gRPC, worker, and background-job failures so every emitted error can carry a stable public code, friendly message key, support/request ID, severity, and trace metadata without every handler rebuilding that contract ad hoc.
- [ ] Add one shared log-field builder for route, RPC, mutation, and provider paths so request ID, correlation ID, actor user ID, workspace ID, target ID, route or RPC name, and action name are attached consistently to actionable logs instead of drifting by call site.
- [ ] Add one secret-scrubbing helper used by server logs, diagnostic payload logs, and customer-safe error responses so sensitive headers, tokens, passwords, API keys, cookies, provider payload fragments, and file paths are redacted in one place before any message is emitted.
- [ ] Add correlation propagation through HTTP entry, gRPC handlers, provider calls, background jobs, webhook delivery, and notification dispatch so a single support ID or request ID can be followed end to end during incident investigation.
- [ ] Add explicit error logging hooks for every top-level boundary that can swallow failures, including HTTP handlers, streaming RPC loops, background-job dispatch, webhook delivery, worker bootstrap, and any client-fallback boundary bridge, so failures never degrade into silent drops.
- [ ] Add one variable-name quality review checklist for example 100 hot paths so maintainers can systematically clean ambiguous locals and helper names in auth, dashboard, billing, chat send, and logging code without turning it into random style churn.
- [ ] Add one error-boundary inventory that lists every top-level failure boundary in the client, server, worker, stream, job, and webhook paths and marks whether it logs, whether it shows customer-safe copy, and whether it currently drops trace context.
- [ ] Add one operator log taxonomy note that defines severity levels, required fields, redaction expectations, and when a failure should emit `info`, `warn`, or `error` so new diagnostics stay consistent instead of ad hoc.
- [ ] Add one verification matrix for traceability and secrecy that maps customer-visible error IDs to operator logs, audit rows, and runtime logs while confirming that passwords, bearer tokens, API keys, cookies, provider credentials, and raw upstream payloads never appear in customer-facing copy.
- [ ] Add one manual bug-hunt checklist that intentionally triggers auth, chat-send, settings-save, dashboard-load, background-job, webhook, and provider failures and verifies four things each time: friendly customer copy, visible support ID, matching operator log trail, and no secret leakage.
- [x] Add one manual-smoke checklist for external auth covering customer login, account linking, workspace-required SSO denial, enterprise callback success/failure, and confirmation that `customer@email.com / password` and `admin@email.com / password` still work end to end for quick QA.
	Checkpoint: added `MANUAL_SMOKE.md` section `External Auth Manual Smoke Checklist (Planned Rollout)` with customer/admin password checks, account-linking validation, required-SSO denial check, and callback success/failure checks; validation run was targeted `rg` for all five checklist themes; result gives one explicit external-auth smoke gate; residual risk is checklist remains planned until callback routes are wired; next suggested todo is password-reset browser gap note.
- [x] Add one explicit coverage-gap note and bug-search checklist for password-reset and update-password browser flows, because server and store lifecycle coverage exists but the current Playwright surface does not clearly exercise request, token-consume, expired-token, and post-reset login behavior end to end.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Password reset and update-password browser gap` with current-state note and request/consume/expired/post-reset bug-search steps; validation run was targeted `rg` for reset-token lifecycle checklist terms; result isolates a high-risk auth browser gap; residual risk is no dedicated playwright spec yet; next suggested todo is email-verification lifecycle gap note.
- [x] Add one explicit coverage-gap note and bug-search checklist for the email-verification lifecycle, because token persistence and auth-service behavior are package-tested but browser verification of verify, already-used, expired, and resend-adjacent states is still missing.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Email verification lifecycle browser gap` with explicit state coverage (`verify`, `already used`, `expired`, resend guidance) and bug-search steps; validation run was targeted `rg` for lifecycle-state keywords; result documents missing browser path coverage clearly; residual risk is resend flow remains manually checked until automated; next suggested todo is dashboard-route smoke gap note.
- [x] Document the dashboard-route smoke gap precisely: the current route smoke covers pricing, auth, and settings entry, but it does not clearly smoke `/app/dashboard` and `/app/dashboard/*`, so dashboard-route breakage can still slip past the named smoke suite.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Dashboard route smoke gap` naming the specific missing `/app/dashboard` and `/app/dashboard/*` gate coverage and checklist for adding route/deep-link assertions; validation run was targeted `rg` for explicit route strings; result closes ambiguity around dashboard smoke scope; residual risk is implementation work still needed in smoke test suite; next suggested todo is workspace-admin browser gap note.
- [x] Document the workspace-admin browser-coverage gap separately from superuser coverage, including which scoped dashboard slices, empty states, support flows, and mutation confirmations are package-tested or partially browser-tested but not yet covered by one coherent workspace-admin operator journey.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Workspace-admin browser-coverage gap` with scoped-slice/empty-state/mutation-confirmation journey checklist and superuser separation; validation run was targeted `rg` for workspace-admin journey anchors; result separates workspace-admin risk from superuser-only coverage; residual risk is no full workspace-admin e2e journey spec yet; next suggested todo is stub-vs-real-provider coverage gap note.
- [x] Document the stub-vs-real-provider coverage gap with concrete examples: which startup, send, billing-total, provider-health, and degraded-provider flows are mostly validated under `CHAT_PROVIDER_STUBS=all`, and which still need real-provider spot checks to catch production-only failures.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Stub-vs-real-provider coverage gap` with concrete stub-heavy seams (startup, send, billing totals, provider health, degraded upstream behavior) and spot-check sequence; validation run was targeted `rg` for `CHAT_PROVIDER_STUBS=all` and seam keywords; result clarifies production-only risk areas; residual risk is real-provider test cost and flake; next suggested todo is diagnostics-assertion coverage review.
- [x] Add one diagnostics-assertion coverage review that calls out the exact seams where tests currently verify behavior but not emitted warnings or operator hints, especially boot or hydration warnings, auth bootstrap failures, dashboard partial-load warnings, mutation audit confirmations, and retry or replay failures.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Diagnostics-assertion coverage review` listing boot/hydration, auth bootstrap, dashboard partial-load, mutation audit, and retry/replay diagnostic assertion gaps; validation run was targeted `rg` for each seam label; result identifies where behavior-only tests need operator-hint assertions; residual risk is assertion adoption depends on future test changes; next suggested todo is known lightly-tested seams list.
- [x] Add one "known untested or lightly tested seams" section with concrete entries for the intentionally stubbed server-tool surface, tenant/site override readiness, cache invalidation after catalog or settings changes, performance-threshold drift, and any route or mutation path still relying mostly on package tests instead of browser verification.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Known untested or lightly tested seams` with concrete entries for server-tools stubs, override readiness, cache invalidation, perf-threshold drift, and package-test-heavy route/mutation seams; validation run was targeted `rg` across all listed seam labels; result gives one consolidated residual-risk list; residual risk is list requires periodic pruning as seams close; next suggested todo is recurring bug-hunt checklist.
- [x] Add a recurring bug-hunt checklist for example 100 releases that tells maintainers which routes, flows, roles, env modes, and failure injections to rotate through before calling the example stable.
	Checkpoint: added `MANUAL_SMOKE.md` subsection `Recurring Release Bug-Hunt Checklist` with route/flow/role/env/failure-injection rotation matrix; validation run was targeted `rg` for each checklist dimension; result creates a reusable pre-release bug-hunt loop; residual risk is checklist completeness depends on new feature additions; next suggested todo is layered catalog precedence contract docs.
- [x] Document the target server-owned i18n architecture in product/runtime terms: what the server owns, what the client still owns, what stays embedded as emergency fallback, and how the boot path changes.
	Added `docs/I18N_ARCHITECTURE.md` section `Target server-owned i18n architecture` with explicit server/client/fallback ownership and updated boot-path sequence.
- [x] Add one namespace ownership map for `marketing`, `auth`, `chat`, `settings`, `billing`, and `dashboard`, including who owns each namespace and where new keys should be added.
	Added `Namespace ownership map` to `docs/I18N_ARCHITECTURE.md` with owner matrix and explicit key-placement rules by namespace.
- [x] Document the layered catalog precedence contract: app default -> locale translation -> environment/site override -> tenant override -> experiment/campaign override.
	Checkpoint: added `docs/I18N_ARCHITECTURE.md` section `3) Layered catalog precedence contract` with fixed precedence order and conflict-resolution rules; validation run was targeted `rg` for all five precedence layers; result provides one canonical overlay order contract; residual risk is runtime must enforce same precedence; next suggested todo is incremental migration plan.
- [x] Add one migration plan that explains how to move keys out of `client/app/i18n.go` incrementally, route by route, without breaking current rendering behavior.
	Checkpoint: added `docs/I18N_ARCHITECTURE.md` section `4) Incremental migration plan from client/app/i18n.go` with phased route-by-route migration and fallback-reduction rule; validation run was targeted `rg` for phase anchors and route-by-route rule; result defines safe migration sequencing; residual risk is migration timing tied to future implementation bandwidth; next suggested todo is first-paint requirements contract.
- [x] Document first-paint requirements for boot-injected namespaces so landing/auth copy never visibly swaps after hydration unless the locale actually changes.
	Checkpoint: added `docs/I18N_ARCHITECTURE.md` section `5) First-paint requirements for boot-injected namespaces` with no-raw-key/no-double-swap requirements and locale-change exception; validation run was targeted `rg` for first-paint guardrails and route list; result makes hydration behavior expectations explicit; residual risk is browser assertions still need enforcement tests; next suggested todo is cache/fallback contract.
- [x] Document the cache and fallback contract so maintainers know when the client should use server-fresh catalogs, cached catalogs, or a minimal built-in English fallback.
	Checkpoint: added `docs/I18N_ARCHITECTURE.md` section `6) Cache and fallback contract` with server-fresh/cached/fallback selection order and cache-key invalidation rules; validation run was targeted `rg` for source-selection and invalidation terms; result defines deterministic catalog-source behavior; residual risk is cache schema/version handling must match runtime implementation; next suggested todo is placeholder/interpolation policy.
- [x] Add a placeholder and interpolation policy that defines variable syntax, required placeholder parity across locales, and what validation must fail the build or publish.
	Checkpoint: added `docs/I18N_ARCHITECTURE.md` section `7) Placeholder and interpolation policy` specifying `{{variable_name}}`, locale placeholder parity, and build/publish failure gates; validation run was targeted `rg` for syntax/parity/validation rules; result gives one enforceable interpolation contract; residual risk is tooling must implement the documented validation checks; next suggested todo is future extensibility note.
- [x] Add a future extensibility note covering tenant overrides, white-label copy, pricing-copy experiments, temporary incident banners, and eventual CMS/admin-managed catalog publishing.
	Checkpoint: added `docs/I18N_ARCHITECTURE.md` section `8) Future extensibility note` covering tenant overrides, white-label packs, pricing experiments, incident banners, and CMS publishing path; validation run was targeted `rg` for each extensibility item; result documents forward-compatible goals without changing current client contract; residual risk is implementation sequencing across teams; next suggested todo is to continue next unchecked Agent 4 item.
- [x] Document the five core dashboard surfaces in product terms so maintainers know exactly what belongs in `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, and what intentionally does not belong there.
	Added `Core dashboard surfaces` to `DESIGN.md` with explicit scope boundaries for `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, including a paired "not in this surface" clause for each slice.
- [x] Add one dashboard metric-definition section that defines each core KPI, its source-of-truth table set, its source RPC, and any caveats about lag, rollups, or derived values.
	Added `Dashboard KPI definitions (planned)` in `DESIGN.md` with a per-KPI map of source tables, summary RPC ownership, and caveats for lag, rollups, and derived-signal interpretation.
- [x] Add one dashboard endpoint and query map that lists the planned summary RPCs, drill-down RPCs, mutation RPCs, store funcs, SQL queries, and underlying tables for each dashboard surface.
	Added `Dashboard endpoint and query map (planned)` in `DESIGN.md`, mapping each surface (`Business`, `Customers`, `Chats`, `Providers`, `Ops`) to summary/drill-down/mutation RPCs, store-function families, SQL query families, and primary table sets.
- [ ] Add one dashboard data-source map in code comments or helper docs that pairs each KPI card, trend, list, and settings form with its source RPC, store func, SQL query, and underlying table set so future changes stay traceable.
- [x] Document the dashboard permissions model so maintainers know which surfaces, lists, detail views, and settings forms belong to normal users, workspace admins, and superusers.
	Added `Dashboard permissions model (planned)` to `DESIGN.md` with a role matrix covering surface visibility, list/detail scope, and mutation scope for normal users, workspace admins, and superusers.
- [x] Add a dashboard operator runbook that explains how to review revenue, customer health, chat health, provider health, and site health without drifting into vanity metrics or redundant surfaces.
	Added `Dashboard Review Playbook (Planned)` to `OPERATOR_RUNBOOK.md` with an action-first review loop for `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, plus explicit anti-vanity guidance for each surface.
- [x] Document the day-in-the-life operator workflows for `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, including queue-to-detail-to-action loops, expected decision points, and the evidence an admin should review before mutating state.
	Added `Dashboard Day-In-The-Life Workflows` to `FLOWS.md` with one queue->detail->action evidence loop per surface plus explicit decision-point checkpoints.
- [x] Add one mutation preflight guide that defines what blast-radius preview, affected-record counts, audit notes, confirmation copy, and rollback guidance each high-risk dashboard action must show before it is considered safe to ship.
	Added `Mutation Preflight Guide` to `OPERATOR_RUNBOOK.md` defining required blast-radius, affected-record, audit-note, confirmation-copy, and rollback checks before high-risk mutations.
- [x] Document the pricing philosophy and plan boundary in operator/product terms: `Pro = one serious operator`, `Team = shared workspace with coordination/admin relief`, `Enterprise = contract/security/compliance path`, and `monthly bill = platform fee + usage + service premium`.
	Added `Pricing philosophy and billing vocabulary` to `README.md` with the `Pro`/`Team`/`Enterprise` boundary and explicit monthly bill formula contract.
- [x] Document the exact customer-facing billing vocabulary so docs and UI consistently use `platform fee`, `usage`, and `service premium`, and explicitly avoid stale phrases like `free`, `unlimited`, `all models included`, or `no token caps`.
	Added a customer-facing vocabulary contract in `README.md` that standardizes `platform fee`/`usage`/`service premium` and explicitly marks stale phrases as prohibited.
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

- [ ] Replace the client-only full-catalog wiring with a server-catalog integration layer so the existing `i18n.Provider` can render boot-injected namespaces first and merge lazy-loaded namespaces without changing overall UI composition.
- [ ] Update landing, auth, settings, billing, and dashboard surfaces so each route declares the namespaces it needs and shows a clean fallback state while a non-critical namespace is still loading.
- [ ] Reduce the embedded client catalog down to the smallest emergency fallback strings needed for catastrophic catalog-fetch failure while keeping normal rendered copy sourced from the server path.
- [ ] Add one real superuser server-tools surface under the admin or ops area so operators can review the active tool policy, edit the approved tool list and safety limits, and inspect recent execution outcomes instead of relying on a hidden RPC only.
- [ ] Add visible memory-extraction parity feedback in settings or remembered-preferences surfaces so provider-specific extraction failures, disabled extraction, and successful memory-candidate application are understandable to users rather than silently diverging by provider.
- [ ] Add one auth-entry UI that can present password login, Google login, and workspace SSO entry without turning the page into auth clutter, while keeping the plain email/password path obvious and always usable for quick QA/local testing.
- [ ] Add one external-auth callback and failure UI treatment so consent denial, expired state, mismatched workspace policy, and required-SSO messages render clearly instead of dumping users back into a generic login error.
- [ ] Add one account-linking and linked-login-methods surface in profile or security settings so users can see whether password, Google, or workspace identity is attached to their account, and so QA can confirm local password remains enabled after external providers are linked.
- [ ] Add customer-facing error treatment across auth, chat send, settings save, and dashboard actions so failures show a calm friendly message plus a visible support ID or request ID that support can use to find the matching server logs.
- [ ] Replace raw auth failure text in the client with calm customer-facing copy that distinguishes bad credentials, expired session, denied SSO policy, and temporary server issues without surfacing raw backend details.
- [ ] Replace raw chat-send, settings-save, and dashboard-load error text in the client with reusable customer-safe patterns that show what failed, what the user can try next, and the visible support/request ID without duplicating the full operator log message.
- [ ] Add one compact support-ID presentation pattern that is visible enough for QA and support handoff, copyable on desktop and mobile, and reused consistently across auth, chat, settings, and dashboard error surfaces.
- [ ] Ensure customer-facing error surfaces never render raw tokens, emails from hidden scopes, provider names from restricted contexts, stack traces, SQL/provider error bodies, or internal route/RPC identifiers even when fallback UI is forced from unusual failure paths.
- [ ] Research and mock the leanest auth-entry UI direction for password + Google + workspace SSO so the page stays clear for normal users while still exposing enterprise entry paths when needed.
- [ ] Research the smallest replacement for the current 7-step journey band on home/auth/chat surfaces so the product can keep any useful first-run guidance without the heavy progress-strip treatment.
- [x] Refine the public-to-first-chat UX so landing, pricing, signup, login, empty state, first composer state, streaming state, and post-first-reply state read like one coherent journey.
	Added one shared journey band across landing/pricing/signup/login/workspace plus state-derived first-chat guidance in empty and composer surfaces for first prompt, first-reply streaming, and post-first-reply continuation.
- [x] Refresh the chat surface styling so its colors, border treatments, and shapes feel like the same product as the home page instead of a separate UI.
	Retuned workspace base/background, sidebar cards, composer shell, send CTA, assistant badge, and message/user bubble surfaces to the marketing palette (`#050508`/`#111118` + cyan accents) with consistent border density and shape language.
- [ ] Reduce the chat-surface flashiness by roughly 50% so gradients, glow accents, highlight borders, and animated emphasis no longer dominate the reading experience.
- [ ] Flatten the chat interface styling so cards, bubbles, composer, and sidebar surfaces rely more on quiet contrast and spacing than on layered shine, heavy depth, or decorative visual effects.
- [ ] Tone down distracting branded treatments in the workspace shell so the app feels calmer during long chat sessions while still reading as the same product family as the home page.
- [ ] Increase branded icon sizing across the chat interface by about 150% where marks currently read too small, and rebalance spacing so the larger icons feel intentional rather than cramped.
- [ ] Make the chat-to-canvas split pane left-to-right resizable so users can drag the divider and give the canvas more or less room without losing the chat context.
- [ ] Constrain the resizable chat/canvas split with sane bounds so neither side collapses into nonsense, using roughly `20% / 80% / 20%` minimum-maximum behavior for the two panes.
- [ ] Rework the canvas surface so the canvas content area is the primary focus and occupies most of the available panel instead of competing with dev-oriented controls.
- [ ] Remove or hide the current dev-tool clutter from the canvas area and replace it with one simple top bar that only contains a few compact dropdown controls until the canvas feature has a clearer product definition.
- [ ] Simplify the canvas top bar so it feels like a lightweight content toolbar rather than an internal debugging surface, keeping only the minimum useful controls and leaving room for the actual canvas output.
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
- [x] Tweak the `/home` hero copy with concrete product language.
	Example headline: `AI chat for teams that need faster answers and fewer repeat questions.`
	Example subhead: `RelayDesk turns internal docs, shared prompts, and model routing into one reliable workspace for support, ops, and customer-facing teams.`
	Example primary CTA: `Create your workspace`
	Example secondary CTA: `See pricing`
- [x] Tweak the `/home` supporting sections so they answer who it is for and why it is useful.
	Example section heading: `Built for support, ops, and internal knowledge teams`
	Example body: `Use RelayDesk to answer repeat questions faster, standardize team responses, search company knowledge, and keep admins in control of cost, access, and model behavior.`
	Example capability bullets: `Ask across docs`, `Route to the right model`, `Save reusable workflows`, `Review usage and admin controls`
- [x] Tweak the `/home` proof, metric, and trust copy so it reads like a credible business.
	Example trust band: `Flexible model routing`, `Admin controls`, `Usage tracking`, `Workspace-ready billing`
	Example proof copy: `Designed for teams that need grounded answers, cleaner handoffs, and clear visibility into usage.`
	Example metric-label replacements: change decorative labels into `Workspace controls`, `Model flexibility`, and `Operational visibility`.
- [x] Tweak the `/home` section and footer copy so the whole page reads like one coherent SaaS narrative.
	Example footer group labels: `Product`, `Company`, `Legal`
	Example footer links to keep: `Home`, `Pricing`, `Sign up`, `About`, `Contact`, `Privacy`, `Terms`, `Security`, `Status`
	Example copy rule: no `framework`, `demo`, `lab`, or placeholder-style wording anywhere on kept pages.
- [x] Rewrite the kept `/pricing` page with concrete plan-selection copy.
	Rewrote FAQ questions in EN/ES/FR `bundle.go` to the five billing-formula questions: platform fee, how usage is billed, what the service premium is, when to choose Team over Starter, and how to contact sales.
- [x] Rewrite the kept `/pricing` plan cards so they sell the emotional boundary clearly.
	Example `Pro` card framing: `For one operator who wants a reliable AI workspace without team overhead.`
	Example `Team` card framing: `For teams that need shared workflows, admin visibility, and less chaos.`
	Example `Enterprise` card framing: `For organizations that need security review, procurement, SSO, and contract controls.`
- [x] Rewrite the kept `/pricing` numbers and labels so the formula is explicit.
	Rewrote hero stat cards in EN/ES/FR `bundle.go` to `Monthly platform fee`, `Actual AI usage`, and `Service premium` with formula-explicit body copy on each card. Updated plans h2 to workspace-framing language in all three locales.
- [x] Rewrite the kept `/pricing` comparison table so it compares `workspace mode`, `collaboration`, `admin controls`, `billing visibility`, `support`, and `security/compliance path` instead of implying the main difference is just more usage.
	Replaced 8-row seats/models/shared/admin/api/residency/retention/sla table with 6-row workspace-mode/collaboration/admin/billing-visibility/support/compliance table in EN/ES/FR `bundle.go`. Updated `compareRowKeys` slice in `pricing_shell.go` and fixed footer Company/Legal columns to use real marketing routes instead of `#` placeholders.
- [x] Rewrite the kept `/signup` page with concrete conversion copy.
	Rewrote `auth.signupBadge`, `auth.signupHeroTitle`, `auth.signupHeroBody`, and all three stat cards in EN/ES/FR `bundle.go` to billing-formula-grounded copy. Signup hero now reads: "Create your RelayDesk workspace." / "Start as one serious operator. Add your team when coordination and admin control matter. Billing stays clear from day one." Stat cards rewritten to clear platform fee, visible usage, and workspace-ready framing. `auth.signupFormSub` and `auth.signupFormBody` updated to workspace language in all three locales.
- [x] Rewrite the kept `/` auth landing with concrete login-first copy.
	Rewrote `auth.loginBadge`, `auth.loginHeroTitle`, `auth.loginHeroBody`, and all three login stat cards in EN/ES/FR `bundle.go`. Login page now leads with "Welcome back." / "Sign in to continue your chats, settings, and workspace activity." Stat cards rewritten to "Your chats", "Your settings", "Your workspace". Also tightened `auth.resetHeroTitle` and `auth.resetHeroBody` in all three locales to workspace-return framing.
- [x] Add a real `/about` page with concrete business copy.
	Example headline: `About RelayDesk`
	Example body: `RelayDesk helps teams turn AI chat into reliable day-to-day work by combining grounded answers, reusable workflows, and practical admin controls in one workspace.`
	Example CTA: `View pricing` and `Create your workspace`
- [x] Add a real `/contact` page with concrete business copy.
	Example headline: `Contact RelayDesk`
	Example body: `Talk to us about sales, support, onboarding, or enterprise requirements.`
	Example sections: `Sales`, `Support`, `General`, `Response times`
	Example expectation copy: `We aim to respond to product and sales questions within one business day.`
- [x] Add a real `/privacy` page with concrete top-level summary copy before the legal detail.
	Example intro: `Privacy at RelayDesk`
	Example summary bullets: `What data we collect`, `How we use it`, `How long we retain it`, `How to contact us about privacy requests`
- [x] Add a real `/terms` page with concrete top-level summary copy before the legal detail.
	Example intro: `RelayDesk Terms`
	Example summary bullets: `Account responsibilities`, `Acceptable use`, `Billing and renewals`, `Service availability`, `Contact information`
- [x] Add a real `/security` page with concrete trust copy.
	Example headline: `Security at RelayDesk`
	Example body: `RelayDesk is built for teams that need clear access controls, auditable operations, and predictable handling of workspace data.`
	Example sections: `Access control`, `Data handling`, `Retention`, `Operational safeguards`, `Contact`
- [x] Add a real `/status` page or status surface with concrete operational copy.
	Example headline: `RelayDesk Status`
	Example body: `Track current platform health, recent incidents, and service updates in one place.`
	Example sections: `Current status`, `Active incidents`, `Recent history`, `Planned maintenance`
- [x] Tighten the kept `/app/settings` content so each section title, summary, and helper line explains one real user decision instead of repeating vague product language, with profile and billing reading like first-class account surfaces.
	Rewrote settings copy in EN/ES/FR `bundle.go`: `modal.displayNamePlaceholder` → "Your name in chats and workspace" (nav summary + placeholder); `modal.ttsProviders` → "Voice" (dropped TTS jargon); `modal.ttsProvidersHelp` → "Select which service reads assistant replies aloud."; `modal.memoriesHelp` → "What the assistant remembers about you across sessions."; `modal.profileUsageTitle` → "This month's usage"; `modal.billingHelp` → formula-explicit sentence; `modal.billingNavSummary` → "Your current plan and this month's spend".
- [x] Tighten the kept dashboard-entry content so it explains the five admin surfaces in operator language, shows where to start when something is wrong, and avoids filler copy or placeholder-card language.
	Rewrote `parseDashboardRoleDescription` (admin: now lists which slice to check for what problem), `parseDashboardSlices` subtitles (each now names what to act on, not just what's there), section label "Dashboard slices" → "Admin surfaces", account summary section label → "Your account this period", and account card labels "Conversations" → "Chats", "Base cost" → "Model cost", "Premium" → "Service premium".
- [x] Upgrade the footer into a real business footer by wiring it only to first-class destinations: `/home`, `/pricing`, `/signup`, `/about`, `/contact`, `/privacy`, `/terms`, `/security`, and `/status`, while dropping low-signal filler links like `Careers`, `Customers`, and `Documentation` until those pages actually exist.
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
	Role-gated admin entry buttons are now present in both mobile and desktop control bars (`client/app/panel.go`) and route through the authenticated shellâ€™s `OpenAdminDashboard` handler (`client/app/app.go`) so non-admin users never see admin affordances.
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
	Changed `bottom-7` â†’ `bottom-24` so the button floats well above the composer, and `h-16 w-16 text-[1.25rem]` â†’ `h-[4.5rem] w-[4.5rem] text-[1.5rem]` for a larger, more visible target.
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














