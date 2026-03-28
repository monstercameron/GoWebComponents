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

### Agent 1: Runtime, routing, and regression coverage

- [ ] Add one browser-level visit-to-first-chat regression that covers landing load, pricing/auth navigation, login/signup handoff, authenticated shell boot, first send, streamed reply, and canonical thread-route normalization.
- [ ] Add runtime diagnostics for the visit-to-first-chat path so boot, gRPC connect, worker readiness, model-catalog load, conversation bootstrap, and first-send failures log one actionable warning or error each.
- [ ] Add focused regression coverage for empty-state boot, first-thread creation, scroll anchoring during first reply, and reopen-after-refresh behavior.
- [ ] Add one browser-level admin journey regression that covers homepage load, login, role resolution, admin-entry visibility, dashboard-home load, slice navigation, and back/refresh behavior.
- [ ] Add runtime diagnostics for the admin dashboard journey so role resolution, admin route entry, dashboard bootstrap, slice fetches, and unauthorized transitions emit actionable logs.
- [ ] Add focused regressions for workspace-admin vs superuser route guards, direct deep links into dashboard slices, and empty/error dashboard states.
- [ ] Add one browser-level admin-mutation regression that covers disable user, restore user, suspend workspace, restore workspace, and the resulting UI state transitions.
- [ ] Add focused regressions for billing-intervention, support-triage, and incident-control flows so admin actions survive refresh and back navigation correctly.
- [ ] Add runtime diagnostics for admin mutations so confirmation, submit, success, denial, and rollback states emit actionable logs during operator workflows.
- [ ] Add focused regressions for admin list mechanics so user, workspace, support, incident, and billing views keep search, filter, sort, and pagination state stable across refresh and back navigation.
- [x] Run a real browser smoke against `http://127.0.0.1:8095/` and capture any remaining startup console/runtime errors.
	Executed a live Playwright smoke against `http://127.0.0.1:8095/` with `status=200`, title `RelayDesk - AI Chat Workspace`, and zero console/page errors.
- [x] Add a focused end-to-end regression for server start, WASM shell boot, and background worker boot.
	Added `TestExample100StartupBoot` under `test/playwrightgo/examples` to assert health check, boot-shell removal, `chat.wasm` and `background-worker.wasm` responses, `grpc ready`, `worker ready`, and worker-pool startup logs with zero console/page errors.
- [x] Verify the authenticated happy path end-to-end: login, create thread, send message, stream reply, refresh, reopen thread.
	Added `TestExample100AuthenticatedHappyPath` to seed a deterministic DB, sign in as `demo@example.com`, create/send in a new thread, wait for streamed assistant output, refresh and assert the same `/app/thread/:publicID`, then reopen the thread from the sidebar and verify prompt persistence.
- [x] Add browser-level smoke coverage for the pricing, auth, and dashboard routes.
	Added `TestExample100RouteSmokePricingAuthDashboard` to smoke `/pricing#faq`, unauthenticated `/app` auth inputs, and authenticated `/app/settings?panel=settings-profile` with zero browser console/page errors.
- [x] Trace and fix the mismatch where the model select shows `GPT-5.4 - Best` while message bubbles report `GPT-5.4 mini`.
	Stream completion now normalizes the assistant-reported reply model, syncs `SelectedModel` to that canonical model when it differs, and persists the synced value via `SetSelectedModel` so picker state stays aligned with message metadata after first reply.
- [x] Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.
	`ParseScrollToBottom` now performs a delayed settle pass that snaps the message list to exact bottom after the initial smooth jump, and `TestExample100ScrollToBottomButton` was added to assert bottom-alignment after clicking `#scroll-to-bottom-btn`.
- [ ] Update the app version number and ensure the displayed version string is sourced consistently.

### Agent 2: Security, auth, and billing enforcement

- [x] Wire the visit-to-first-chat auth path to durable server-side sessions so login/signup, refresh, and reopen-thread flows all use persisted `auth_sessions` and `auth_token_versions`.
	Server-issued auth tokens now carry durable `sid`/`ver` claims backed by `auth_sessions` and `auth_token_versions`, with persisted-session validation and logout revocation covering login/signup, refresh, and reopen flows.
- [ ] Add auth-state enforcement for the public-to-app transition so blocked, expired, revoked, or malformed sessions fail into a sane auth route instead of a half-booted app shell.
- [ ] Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.
- [ ] Enforce dashboard access by resolved role so normal users cannot reach admin surfaces, workspace admins get workspace-scoped data, and superusers get platform-scoped data.
- [ ] Add server-side checks for admin deep links, slice RPCs, and detail views so unauthorized dashboard routes fail closed and redirect or render a sane denied state.
- [ ] Define session and auth-expiry behavior for long-lived dashboard tabs so expired admin sessions fall back safely without leaving stale privileged UI visible.
- [ ] Add a typed auth bootstrap contract that returns session status, role summary, and expiry information needed for safe app-shell and dashboard entry decisions.
- [ ] Enforce persisted token lifecycles for signup verification, password reset, and update-password flows using the new token tables instead of ad hoc state.
- [ ] Ensure logout, privilege downgrade, and session revocation immediately evict cached admin state and invalidate privileged dashboard access.
- [ ] Define role-aware post-login redirect rules so normal users land in the chat app while admin-capable users can be routed intentionally into dashboard-capable entry points.
- [ ] Add re-auth or heightened-session checks for sensitive superuser actions so long-lived dashboard sessions do not automatically authorize mutating control-plane changes.
- [ ] Define and enforce user-disable, user-restore, workspace-suspend, and workspace-restore authorization rules, including who can perform them and against which scopes.
- [ ] Ensure disable and suspend actions revoke active sessions, block new auth, and deny downstream privileged RPCs immediately.
- [ ] Add confirmation and reason requirements for destructive admin actions so high-impact changes are auditable and harder to trigger accidentally.
- [ ] Enforce feature-flag, experiment, and incident-control mutations behind the correct workspace-admin vs superuser boundaries.
- [ ] Define and enforce the exact runtime effects of user disable vs workspace suspend so chat send, dashboard access, API keys, webhooks, background jobs, and support actions all fail in a consistent way.
- [ ] Ensure restore flows re-enable only the intended scopes and do not silently reopen revoked sessions, API keys, or operator overrides unless explicitly requested.
- [x] Persist and enforce server-side auth sessions using `auth_sessions` and `auth_token_versions`.
	Server-issued auth tokens now carry durable session and version claims backed by `auth_sessions` and `auth_token_versions`, with active-session checks enforced during token validation and session revocation on logout.
- [x] Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.
	Tokens now stamp `jti` to match durable session IDs, session-version checks are enforced on every validation, and `kid`-aware verification supports active-key signing plus configured rotation keys (`CHAT_AUTH_SIGNING_KEYS` / `CHAT_AUTH_ACTIVE_KID`).
- [x] Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.
	`parseRequireUserEntitlement` now fails closed when the effective entitlement key is absent, and Send-path tests/benchmarks now seed explicit billing plans where non-entitlement outcomes are expected.
- [x] Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.
	`parseRequireUsageBudget` now enforces monthly token caps from `usage.monthly_token_limit`, plus per-user send-rate and concurrency leases (`usage.sends_per_minute`, `usage.concurrent_sends`) with stream-lifetime release semantics in `Send`.

### Agent 3: Control plane, ops, and backend workflows

- [ ] Add typed funnel instrumentation for the visit-to-first-chat journey: landing viewed, pricing viewed, CTA clicked, auth started, auth completed, app booted, first thread created, first send started, first reply completed.
- [ ] Ensure typed store and RPC funcs exist for first-chat bootstrap data: profile defaults, selected settings, model catalog, conversation list, active conversation load, and default system prompt injection.
- [ ] Add backend support for first-run onboarding milestones and starter-state helpers so the app can distinguish brand-new users from returning users with prior threads.
- [ ] Add server-side write paths for first-thread creation metadata, first-reply completion markers, and reopen-thread analytics so the funnel can be queried end-to-end.
- [ ] Add typed dashboard-home summary funcs and RPCs for the admin journey: users, conversations, usage, billing signals, incidents, support load, and experiment health.
- [ ] Add typed scoped query funcs and RPCs for workspace-admin slices: memberships, API keys, webhooks, audit logs, invitations, usage, and billing summary.
- [ ] Add typed scoped query funcs and RPCs for superuser slices: global users, global usage, support queue, pricing controls, incidents, experiments, workspaces, and cost guardrails.
- [ ] Add backend audit events for admin-dashboard entry, slice views, drill-down access, and mutating admin actions so operator activity is queryable.
- [ ] Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.
- [ ] Add typed workspace-admin query and mutation funcs for workspace detail, suspend workspace, restore workspace, revoke API keys, pause webhooks, and membership review.
- [ ] Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.
- [ ] Add typed support-triage funcs and RPCs for ticket queues, ticket detail, internal notes, assignment, escalation, and account-linked action history.
- [ ] Add typed incident and experiment control funcs and RPCs for incident updates, status changes, feature-flag toggles, experiment rollbacks, and blast-radius reporting.
- [ ] Add typed search, filter, sort, and pagination support for admin user, workspace, support, incident, and billing list RPCs so large datasets are operable without loading everything at once.
- [ ] Add explicit backend side-effect handlers for disable and suspend flows so session revocation, API-key pausing, webhook pausing, and job suppression happen transactionally and are auditable.
- [ ] Add typed restore handlers that let operators choose which dependent capabilities come back automatically versus which stay manually revoked.
- [x] Add typed store/query funcs for the new operational tables: auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs.
	Added typed SQL query files, query-loader wiring, and typed store methods for workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs, with auth-session list coverage integrated into the existing auth-session store path.
- [x] Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.
	`GetSuperuserControlPlane` now returns typed rows for auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs via extended protobuf response fields.
- [ ] Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.
- [ ] Add typed `su` CRUD RPCs for reliability and trust tables: SSO configs, retention policies, compliance controls, SLOs, incidents, and incident updates.
- [ ] Add a restricted read-only admin query surface for debugging and reporting instead of any raw SQL passthrough RPC.
- [x] Wire store/query funcs for onboarding templates, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
	Added typed SQL files, query-loader entries, and typed store methods for all listed growth tables, with focused lifecycle test coverage proving insert/upsert + list behavior in the server app.
- [x] Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.
	Added typed enqueue + dispatch flows for weekly summaries, dunning retries, retention purges, and health-score refresh jobs, including retry/backoff status transitions with focused lifecycle coverage.
- [x] Add customer-facing notification flows backed by `notification_outbox`.
	Added pending-queue notification dispatch plumbing with typed pending-list + status-update store funcs and a server-side dispatch helper that marks rows as `sent` or `failed` while preserving scheduled future rows.
- [x] Add webhook retry and delivery history plumbing backed by `webhook_deliveries`.
	Added pending-retry listing plus failed-attempt and successful-delivery update paths over `webhook_deliveries`, with focused lifecycle coverage proving retry rows can be scheduled, retried, and cleared on success.
- [ ] Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.
- [ ] Fix the memory extraction system so remembered items are extracted reliably, deduplicated sanely, editable, and consistent with what users see in the remembered-preferences UI.
- [ ] Keep `SCHEMA_TABLES.md` in sync as store funcs and RPCs are added for the new tables.
- [ ] Document which schema tables are persistence-only vs fully wired into store, RPC, jobs, and UI.
- [ ] Add an operator runbook for local start, seed, build-client, and superuser/admin verification flows.

### Agent 4: Docs, flow definition, and operator guidance

- [x] Document the visit-to-first-chat flow in product terms, including which routes, panels, defaults, and backend calls participate at each stage. (`FLOWS.md`, Visit-To-First-Chat section)
- [x] Refresh `README.md` so it reads like a polished entry point for example 100, preserving the existing ASCII architecture diagram and enhancing it to better represent the current server, client, worker, routing, and gRPC flow. (`README.md`, intro + Architecture + Data flow)
- [x] Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly. (`README.md`, Quick start)
- [x] Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior. (`README.md`, Route model)
- [x] Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer. (`README.md`, Runtime pieces)
- [x] Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions. (`FLOWS.md`, Admin-Dashboard Journey section)
- [x] Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control. (`FLOWS.md`, Admin Operational Workflows section)
- [ ] Document the admin search/filter/pagination conventions so operators know how large lists, saved context, and back-navigation are expected to behave.
- [ ] Document the disable-vs-suspend semantics so maintainers and operators know exactly which downstream capabilities are supposed to turn off in each case.
- [x] Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code. (`FLOWS.md`, Public-Route Bug-Fix Workflow)
- [x] Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues. (`FLOWS.md`, First-Chat Bug-Fix Workflow)
- [x] Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps. (`FLOWS.md`, Admin-Dashboard Bug-Fix Workflow)
- [x] Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate. (`MANUAL_SMOKE.md`, Release Gate Checklist)
- [x] Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage. (`MANUAL_SMOKE.md`, Testing Story Matrix)
- [ ] Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.
- [x] Refresh `MANUAL_SMOKE.md` so the manual verification flows cover landing routes, auth, first chat, settings, admin dashboard entry, and key operator actions. (`MANUAL_SMOKE.md`, flow sections 1-4)
- [ ] Update `DESIGN.md` so it reflects the current intended product surface, dashboard direction, and any material UI or IA changes made since the original design pass.
- [ ] Keep `SCHEMA_TABLES.md` synchronized with the control-plane, auth, onboarding, and admin workflow stories so the docs match the real table usage.
- [ ] Add one concise docs map to the example that tells maintainers which file to read for setup, architecture, smoke testing, schema reference, design intent, and changelog history.
- [ ] Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.
- [ ] Move any other loose runtime outputs out of the example root into a consistent runtime location such as `bin/runtime/`, `bin/logs/`, or another single clear convention.
- [ ] Rename inconsistent or low-signal files so the example reads professionally to a new maintainer, then update all references in scripts, docs, tests, and server/client code.
- [ ] Normalize file placement so docs stay at the example root, scripts stay under `scripts/`, runtime outputs stay under runtime folders, and generated artifacts stay under `bin/` or another clearly non-source location.
- [ ] Move non-first-class docs into a dedicated `docs/` folder while keeping the primary entry docs at the example root, then update all links and cross-references after the move.
- [ ] Update `.gitignore` for example 100 so logs, runtime outputs, temp artifacts, and local-only generated files are ignored from their new locations.
- [ ] Remove stale root-level files that only existed because of the old layout once their replacements are wired and validated.
- [ ] Update `README.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, and any other affected docs after the cleanup so every referenced path still opens correctly.
- [ ] Add one short repo-layout section to the example docs that explains where logs, runtime state, generated artifacts, scripts, and source files belong.

### Agent 5: UI surface and dashboard UX implementation

- [ ] Refine the public-to-first-chat UX so landing, pricing, signup, login, empty state, first composer state, streaming state, and post-first-reply state read like one coherent journey.
- [ ] Refresh the chat surface styling so its colors, border treatments, and shapes feel like the same product as the home page instead of a separate UI.
- [ ] Slightly retune the in-app chat palette to align with the landing-page direction while preserving readability, hierarchy, and streaming-state clarity.
- [ ] Adjust chat cards, bubbles, inputs, panels, and buttons so their corner radii, outlines, and surface shapes better match the home-page design language.
- [ ] Build a first-run empty-state and onboarding layer with starter prompts/templates, obvious first actions, and a clear transition from zero threads to the first real thread.
- [ ] Add UI treatment for auth failures, entitlement blocks, upgrade prompts, and post-first-reply success cues so users always know the next action in the journey.
- [ ] Design a clear admin entry point from the authenticated app shell so workspace admins and superusers can discover dashboard access without cluttering the normal user flow.
- [ ] Build a dashboard home UI that summarizes the platform state and clearly branches into workspace-admin vs superuser slices.
- [ ] Build slice UIs for the admin journey in dependency order: dashboard home, workspaces, support, billing and quotas, incidents and SLOs, analytics, and experiments.
- [ ] Add clear empty, loading, denied, and error states for every dashboard surface so admins always know what the system is doing and what action to take next.
- [ ] Build a user-detail admin UI with disable or restore actions, recent activity context, billing context, support context, and clear impact messaging.
- [ ] Build a workspace-detail admin UI with suspend or restore actions, member context, API key and webhook context, and operational-impact messaging.
- [ ] Build billing-intervention UI flows for overrides, failed-payment review, quota review, and entitlement troubleshooting.
- [ ] Build support-triage UI flows for queue review, ticket detail, internal notes, escalation, and linked account actions.
- [ ] Build incident, feature-flag, and experiment control UIs with explicit state-change affordances and clear blast-radius feedback.
- [ ] Build admin list views with search, filters, sort controls, pagination, and persistent query state for users, workspaces, support queues, billing views, and incidents.
- [ ] Build confirmation modals for disable, restore, suspend, override, and rollback actions with reason capture, scope-of-impact copy, and clear success or failure feedback.
- [ ] Show the operational state of disabled users and suspended workspaces clearly across detail views, lists, and related action surfaces so operators can see what is active vs blocked at a glance.
- [ ] Make the first settings option a simple profile page with display-name editing, account total/cost summary placement, and the core user fields and preferences surfaced cleanly.
- [ ] Add a dedicated billing menu to settings for plan, usage, invoice, and account-total visibility.
- [ ] Flatten the AI tone widget by one level so the control is less nested and faster to scan.
- [ ] Flatten the reasoning menu by one level and explain that it sets the default reasoning level used when switching providers or models.
- [ ] Move the "Reasoning mode is on, so only reasoning-capable models are shown" message out of the main layout and show it as a hover overlay on the enabled/disabled reasoning controls.
- [ ] Flatten the TTS provider control by one level and add a toggle for cross-provider override behavior.
- [ ] Normalize the left-pane system-prompt panel height so it matches neighboring panels and reduce the header copy to a simple heading.
- [ ] Redesign the remembered-preferences form to be more compact and scrollable, with per-memory edit and delete icon actions.
- [ ] Fix the top-left branding badge so the `GWC` text stays centered inside the circle, scaling the circle up if needed.
- [ ] Build a superuser dashboard UI for pricing controls, workspaces, support tickets, incidents, and cost guardrails.
- [ ] Build a workspace admin UI for API keys, webhook endpoints, invitations, and audit logs.
- [ ] Enlarge and recenter the scroll-to-bottom button so it sits clearly centered above the input fields.
- [ ] Add chat-thread folders so users can organize conversations into named groups.
- [ ] Support drag-and-drop of chat threads into folders.
- [ ] Support drag-to-reorder for chat threads and folders in the sidebar.
- [ ] Make the chat-thread list independently scrollable with stable scroll behavior.
- [ ] Lazy load chat threads from gRPC as the user scrolls through the sidebar list.

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
