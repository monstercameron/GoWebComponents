# Example 100 Changelog

## Checkpoints

### 2026-03-28 00:10 America/New_York

- completed todo: Expand the example-100 backlog and docs for chat capability planning, admin workflow gaps, testing stories, and repo cleanup guidance.
- files changed: `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `Get-Content examples/100-ai-chat-wizard/TODO.md`; `Get-Content examples/100-ai-chat-wizard/CHANGELOG.md`
- result: Passed. The backlog now includes future chat capability stories (calendar/email hooks, web search, shareable chats, scheduled jobs, image upload, ask-with-docs, skills, workflows, code interpreter), clearer admin/operator workflow coverage, explicit bug-fix/testing stories, and concrete repo-cleanup/doc-refresh instructions.
- residual risk: These changes only improve planning and documentation shape; the newly added capability stories are not yet decomposed into implementation slices across backend, authz, runtime, and UI.
- next suggested todo: Choose which future chat capabilities should move from the planning section into the active agent backlog.

### 2026-03-27 23:21 America/New_York

- completed todo: Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage.
- files changed: `examples/100-ai-chat-wizard/MANUAL_SMOKE.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. `MANUAL_SMOKE.md` now includes a journey-to-coverage matrix across browser specs, focused package tests, and manual smoke groups.
- residual risk: Bug-report templates and broader repo-layout documentation tasks remain open in Agent 4.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:20 America/New_York

- completed todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.
- files changed: `examples/100-ai-chat-wizard/server/app/job_flows.go`, `examples/100-ai-chat-wizard/server/app/job_flows_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run TestHandleBackgroundJobsLifecycle -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle)" -count=1`
- result: Passed. Server-side job-flow helpers now enqueue and dispatch typed weekly-summary, dunning-retry, retention-purge, and health-score-refresh jobs with persisted running/completed/failed or retry-to-pending state transitions.
- residual risk: Dispatch remains callback-driven and synchronous in-process; a persistent scheduler loop and distributed worker coordination are still pending.
- next suggested todo: Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.

### 2026-03-27 23:19 America/New_York

- completed todo: Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate; refresh `MANUAL_SMOKE.md` so manual verification covers landing routes, auth, first chat, settings, admin dashboard entry, and key operator actions.
- files changed: `examples/100-ai-chat-wizard/MANUAL_SMOKE.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. `MANUAL_SMOKE.md` now defines release-gate checklist groups, concrete visitor/first-chat/admin flow steps, operator-action guidance, and optional focused automation commands.
- residual risk: Testing-story matrix and bug-report template docs are still open in Agent 4.
- next suggested todo: Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage.

### 2026-03-27 23:18 America/New_York

- completed todo: Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.
- files changed: `examples/100-ai-chat-wizard/server/app/authz_entitlement.go`, `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/server/app/store.go`, `examples/100-ai-chat-wizard/server/app/queries.go`, `examples/100-ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/100-ai-chat-wizard/sql/store/sum_usage_tokens_since.sql`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/100-ai-chat-wizard/server/app -run "Test(RequireUserEntitlement|RequireUsageBudget|SendStreamsThoughtsAndPersistsConversation|SendPersistsUsageTraceMetadata|SendUsesBillingPlanDefaultModel|SendRejectsConversationOwnedByAnotherUser|SendRejectsMissingAuthenticatedUserBeforeProviderWork)$" -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/100-ai-chat-wizard/server/app -run "^$" -bench "BenchmarkRequireUsageBudget$" -benchtime=200x`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/100-ai-chat-wizard/server/app -count=1`
- result: Passed. `parseRequireUsageBudget` now enforces monthly token limits from billing access control, applies per-user send-rate and concurrency limits, and returns a lease release function that `Send` defers for correct stream-lifetime concurrency tracking.
- residual risk: Rate and concurrency accounting is process-local in-memory state today; limits reset on server restart and are not yet distributed across multiple server instances.
- next suggested todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.

### 2026-03-27 23:17 America/New_York

- completed todo: Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps.
- files changed: `examples/100-ai-chat-wizard/FLOWS.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now defines an admin-dashboard incident workflow that captures role scope, entry path, slice-load failures, mutation-state evidence, and guard-safe validation steps.
- residual risk: Manual smoke matrix/test-story/bug-template docs are still open in Agent 4.
- next suggested todo: Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate.

### 2026-03-27 23:16 America/New_York

- completed todo: Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues.
- files changed: `examples/100-ai-chat-wizard/FLOWS.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now includes a first-chat incident workflow with explicit auth, model-bootstrap, send/stream, and route-normalization evidence capture steps.
- residual risk: Admin-dashboard bug-fix workflow documentation and several release-gate/checklist docs remain open.
- next suggested todo: Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps.

### 2026-03-27 23:15 America/New_York

- completed todo: Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code.
- files changed: `examples/100-ai-chat-wizard/FLOWS.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now includes a concrete public-route incident workflow with route, hydration, router-state, and diagnostics capture steps before patching.
- residual risk: First-chat and admin-dashboard bug-fix workflow docs are still open and should mirror this level of capture specificity.
- next suggested todo: Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues.

### 2026-03-27 23:14 America/New_York

- completed todo: Add customer-facing notification flows backed by `notification_outbox`.
- files changed: `examples/100-ai-chat-wizard/server/app/queries.go`, `examples/100-ai-chat-wizard/server/app/store_superuser.go`, `examples/100-ai-chat-wizard/server/app/notification_flow.go`, `examples/100-ai-chat-wizard/server/app/notification_flow_test.go`, `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/sql/store/list_notification_outbox_pending.sql`, `examples/100-ai-chat-wizard/sql/store/update_notification_outbox_status.sql`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "Test(DispatchNotificationOutboxPendingLifecycle|StoreSuperuserControlPlaneLifecycle)$" -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle)" -count=1`
- result: Passed. Pending notification rows now dispatch through a typed flow and transition to `sent`/`failed` status while future-scheduled rows remain pending.
- residual risk: Delivery is currently callback-driven dispatch plumbing; production channel adapters and scheduled job orchestration remain pending.
- next suggested todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.

### 2026-03-27 23:13 America/New_York

- completed todo: Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.
- files changed: `examples/100-ai-chat-wizard/client/app/scroll_memory.go`, `test/playwrightgo/examples/example100_scroll_to_bottom_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -v` (blocked by pre-existing server compile failure: `server.go:578 assignment mismatch: 2 variables but parseS.parseRequireUsageBudget returns 1 value`); `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -c`; `go run ./tools/gwc test -lane wasm -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard`
- result: The jump-to-bottom action now issues a smooth scroll followed by a timed settle snap to exact bottom, with cancel-safe timers integrated into scroll-memory lifecycle paths. New browser regression coverage was added for the button behavior and compiles under the Playwright lane.
- residual risk: Live execution of the new browser regression is currently blocked by an unrelated in-progress server compile break in the worktree (`parseRequireUsageBudget` call signature mismatch).
- next suggested todo: Update the app version number and ensure the displayed version string is sourced consistently.

### 2026-03-27 23:14 America/New_York

- completed todo: Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer.
- files changed: `examples/100-ai-chat-wizard/README.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. README now includes a runtime-pieces table plus interaction sequence that ties boot shell, client, worker, tunnel, server handlers, store, and provider layers together.
- residual risk: Several remaining Agent 4 workflow docs and repo-layout cleanup tasks are still open.
- next suggested todo: Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code.

### 2026-03-27 23:13 America/New_York

- completed todo: Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior.
- files changed: `examples/100-ai-chat-wizard/README.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. README now documents concrete route classes, current admin route reality, static asset paths, and shell-vs-SPA routing semantics.
- residual risk: The dedicated runtime-components explainer and remaining Agent 4 docs hygiene tasks are still pending.
- next suggested todo: Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer.

### 2026-03-27 23:11 America/New_York

- completed todo: Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly.
- files changed: `examples/100-ai-chat-wizard/README.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. Quick start now uses current `gwc build` and managed `gwc examples ... cmd\server` lifecycle commands, seeded auth credentials, stub-provider mode, and focused browser verification tests.
- residual risk: Later README sections still contain older layout references that are tracked by separate Agent 4 documentation/layout cleanup todos.
- next suggested todo: Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior.

### 2026-03-27 23:10 America/New_York

- completed todo: Add webhook retry and delivery history plumbing backed by `webhook_deliveries`.
- files changed: `examples/100-ai-chat-wizard/server/app/queries.go`, `examples/100-ai-chat-wizard/server/app/store_superuser.go`, `examples/100-ai-chat-wizard/server/app/superuser_control_test.go`, `examples/100-ai-chat-wizard/sql/store/list_webhook_deliveries_pending_retry.sql`, `examples/100-ai-chat-wizard/sql/store/update_webhook_delivery_attempt.sql`, `examples/100-ai-chat-wizard/sql/store/update_webhook_delivery_delivered.sql`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth)" -count=1`
- result: Passed. Webhook delivery history now supports pending-retry lookup plus explicit failed-attempt and delivered-state updates.
- residual risk: Retry plumbing is wired at store/query level, but background-job orchestration for automatic retries is still pending.
- next suggested todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.

### 2026-03-27 23:09 America/New_York

- completed todo: Trace and fix the mismatch where the model select shows `GPT-5.4 - Best` while message bubbles report `GPT-5.4 mini`.
- files changed: `examples/100-ai-chat-wizard/client/app/stream.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`; `go run ./tools/gwc test -lane wasm -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard`
- result: Passed. Reply finalization now normalizes and adopts the assistant-reported model ID, then persists that model selection so the toolbar picker stays aligned with assistant message model metadata after first-send completion.
- residual risk: The route-smoke and happy-path suites validate send/stream completion and zero runtime errors, but there is still no dedicated assertion that compares the exact visible picker label text against the assistant metadata row text.
- next suggested todo: Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.

### 2026-03-27 23:07 America/New_York

- completed todo: Wire store/query funcs for onboarding templates, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
- files changed: `examples/100-ai-chat-wizard/server/app/queries.go`, `examples/100-ai-chat-wizard/server/app/store_growth_ops.go`, `examples/100-ai-chat-wizard/server/app/superuser_control_test.go`, `examples/100-ai-chat-wizard/sql/store/upsert_onboarding_template.sql`, `examples/100-ai-chat-wizard/sql/store/list_onboarding_templates.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_user_activation_milestone.sql`, `examples/100-ai-chat-wizard/sql/store/list_user_activation_milestones.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_saved_workflow.sql`, `examples/100-ai-chat-wizard/sql/store/list_saved_workflows.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_prompt_library_item.sql`, `examples/100-ai-chat-wizard/sql/store/list_prompt_library_items.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_weekly_value_summary.sql`, `examples/100-ai-chat-wizard/sql/store/list_weekly_value_summaries.sql`, `examples/100-ai-chat-wizard/sql/store/create_product_analytics_event.sql`, `examples/100-ai-chat-wizard/sql/store/list_product_analytics_events.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_experiment_assignment.sql`, `examples/100-ai-chat-wizard/sql/store/list_experiment_assignments.sql`, `examples/100-ai-chat-wizard/sql/store/create_subscription_churn_feedback.sql`, `examples/100-ai-chat-wizard/sql/store/list_subscription_churn_feedback.sql`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot)$" -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|Auth)" -count=1`
- result: Passed. Typed store/query wiring now covers onboarding, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
- residual risk: These tables are now wired at store/query level with lifecycle coverage, but dedicated admin and `su` RPC CRUD surfaces are still pending.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-27 23:06 America/New_York

- completed todo: Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.
- files changed: `examples/100-ai-chat-wizard/server/app/authz_entitlement.go`, `examples/100-ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/server/app/rpc_test.go`, `examples/100-ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/100-ai-chat-wizard/server/app/benchmark_sla_test.go`, `examples/100-ai-chat-wizard/server/app/benchmark_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/100-ai-chat-wizard/server/app -run "Test(RequireUserEntitlement|SendAndSpeechNegativeBranches|SendStreamsThoughtsAndPersistsConversation|SendPersistsUsageTraceMetadata|SendRejectsConversationOwnedByAnotherUser|SendRejectsMissingAuthenticatedUserBeforeProviderWork)$" -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/100-ai-chat-wizard/server/app -run "TestSendSLASweep$" -count=1`
- result: Passed. Entitlement checks now deny missing effective access by default while preserving existing Send-path error boundaries and benchmark/test expectations via explicit plan seeding.
- residual risk: Quota/rate/concurrency enforcement is still pending and remains the final open Agent 2 item.
- next suggested todo: Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.

### 2026-03-27 23:06 America/New_York

- completed todo: Refresh `README.md` so it reads like a polished entry point for example 100, preserving the existing ASCII architecture diagram and enhancing it to better represent the current server, client, worker, routing, and gRPC flow.
- files changed: `examples/100-ai-chat-wizard/README.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. README now has a cleaner product entry, an updated architecture diagram aligned to `/socket` + worker/runtime behavior, and an explicit current data-flow summary.
- residual risk: Quick-start and route/runtime explainer sections are still pending as separate Agent 4 doc todos.
- next suggested todo: Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly.

### 2026-03-27 23:04 America/New_York

- completed todo: Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control.
- files changed: `examples/100-ai-chat-wizard/FLOWS.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. Admin operational workflow documentation now defines actor intent, dependency tables, current-vs-target RPC surfaces, and guardrails for destructive changes.
- residual risk: Most workflow mutation RPCs described here are still intentionally pending and remain tracked in Agent 2 and Agent 3 todos.
- next suggested todo: Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.

### 2026-03-27 23:03 America/New_York

- completed todo: Add browser-level smoke coverage for the pricing, auth, and dashboard routes.
- files changed: `test/playwrightgo/examples/example100_route_smoke_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100RouteSmokePricingAuthDashboard -v`
- result: Passed. New route smoke validates `/pricing#faq`, unauthenticated `/app` auth form rendering, and authenticated `/app/settings?panel=settings-profile` in one browser run with `status=200` and zero console/page errors.
- residual risk: This smoke confirms route availability and core shell mounts, but it does not yet assert model-label consistency or scroll-control behavior.
- next suggested todo: Trace and fix the model-label mismatch where the picker shows `GPT-5.4 · Best` while message bubbles show `GPT-5.4 mini`.

### 2026-03-27 23:02 America/New_York

- completed todo: Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions.
- files changed: `examples/100-ai-chat-wizard/FLOWS.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. Admin dashboard journey documentation now captures route entry, current superuser gating, data-slice dependencies, and operator action checkpoints.
- residual risk: Admin operational workflow stories (disable/restore/suspend/billing/support/incident) still need explicit product-flow documentation in Agent 4.
- next suggested todo: Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control.

### 2026-03-27 23:01 America/New_York

- completed todo: Document the visit-to-first-chat flow in product terms, including which routes, panels, defaults, and backend calls participate at each stage.
- files changed: `examples/100-ai-chat-wizard/FLOWS.md`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\100-ai-chat-wizard -ext .md`
- result: Passed. Visit-to-first-chat flow documentation now exists with concrete routes, defaults, and RPC dependencies, including token-cost trace fields.
- residual risk: Admin dashboard journey and admin operational workflow documentation are still pending under Agent 4.
- next suggested todo: Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions.

### 2026-03-27 23:00 America/New_York

- completed todo: Verify the authenticated happy path end-to-end (login, create thread, send message, stream reply, refresh, reopen thread).
- files changed: `test/playwrightgo/examples/example100_authenticated_happy_path_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`
- result: Passed. New browser test seeds a temporary DB, authenticates as `demo@example.com`, creates a new chat thread, sends and streams a reply, reloads while preserving `/app/thread/:publicID`, and reopens the same thread from the sidebar with prompt text intact and zero console/page errors.
- residual risk: The flow still logs route-resolution warnings right after first-send (`reply completed before conversation route resolved`), so route-sync diagnostics remain noisy even when the happy path succeeds.
- next suggested todo: Add browser-level smoke coverage for pricing, auth, and dashboard routes.

### 2026-03-27 22:59 America/New_York

- completed todo: Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.
- files changed: `examples/100-ai-chat-wizard/proto/chat.proto`, `examples/100-ai-chat-wizard/proto/chat.pb.go`, `examples/100-ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/100-ai-chat-wizard/server/app/superuser_control.go`, `examples/100-ai-chat-wizard/server/app/superuser_control_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/100-ai-chat-wizard`); `go test ./examples/100-ai-chat-wizard/proto -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|Auth)" -count=1`
- result: Passed. The superuser control-plane snapshot now includes typed operational rows for auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox rows, and background jobs.
- residual risk: Snapshot coverage is now extended, but dedicated typed mutating `su` CRUD RPCs for pricing and reliability control tables are still pending.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-27 22:58 America/New_York

- completed todo: Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.
- files changed: `examples/100-ai-chat-wizard/server/app/auth_service.go`, `examples/100-ai-chat-wizard/server/app/auth_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "Test(AuthManager(SignupLoginAndTokenRoundTrip|NegativePaths|PersistsSessionAndRevocation|TokenKidRotationPolicy)|ChatServerAuthRPCs)$" -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreAuthSessionLifecycle|StoreSuperuserControlPlaneLifecycle)$" -count=1`
- result: Passed. JWT validation now enforces `jti` and session-version checks while using `kid`-aware signing and verification with rotation-key fallback for legacy kid-less tokens.
- residual risk: Deny-by-default entitlement enforcement and usage-budget quota enforcement are still open and tracked in the next Agent 2 todos.
- next suggested todo: Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.

### 2026-03-27 22:54 America/New_York

- completed todo: Add typed store/query funcs for the new operational tables: auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs.
- files changed: `examples/100-ai-chat-wizard/server/app/queries.go`, `examples/100-ai-chat-wizard/server/app/store_superuser.go`, `examples/100-ai-chat-wizard/server/app/store_auth.go`, `examples/100-ai-chat-wizard/server/app/auth_service.go`, `examples/100-ai-chat-wizard/server/app/superuser_control_test.go`, `examples/100-ai-chat-wizard/sql/store/upsert_workspace_invitation.sql`, `examples/100-ai-chat-wizard/sql/store/list_workspace_invitations.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_webhook_delivery.sql`, `examples/100-ai-chat-wizard/sql/store/list_webhook_deliveries.sql`, `examples/100-ai-chat-wizard/sql/store/create_support_ticket_message.sql`, `examples/100-ai-chat-wizard/sql/store/list_support_ticket_messages.sql`, `examples/100-ai-chat-wizard/sql/store/create_incident_update.sql`, `examples/100-ai-chat-wizard/sql/store/list_incident_updates.sql`, `examples/100-ai-chat-wizard/sql/store/create_notification_outbox.sql`, `examples/100-ai-chat-wizard/sql/store/list_notification_outbox.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_background_job.sql`, `examples/100-ai-chat-wizard/sql/store/list_background_jobs.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_auth_session.sql`, `examples/100-ai-chat-wizard/sql/store/get_auth_session_by_session_id.sql`, `examples/100-ai-chat-wizard/sql/store/list_auth_sessions.sql`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run TestGetSuperuserControlPlaneReturnsSnapshot -count=1`; `go test ./examples/100-ai-chat-wizard/server/app -run TestAuth -count=1`
- result: Passed. The new operational-table store/query surfaces are wired and covered by focused lifecycle, superuser snapshot, and auth tests.
- residual risk: These tables are now typed in store/query layers, but dedicated `su` RPC surfaces for them are still pending.
- next suggested todo: Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.

### 2026-03-27 22:53 America/New_York

- completed todo: Persist and enforce server-side auth sessions using `auth_sessions` and `auth_token_versions`.
- files changed: `examples/100-ai-chat-wizard/server/app/auth_service.go`, `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/server/app/store_auth.go`, `examples/100-ai-chat-wizard/server/app/store_superuser.go`, `examples/100-ai-chat-wizard/server/app/queries.go`, `examples/100-ai-chat-wizard/server/app/auth_test.go`, `examples/100-ai-chat-wizard/server/app/store_test.go`, `examples/100-ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/100-ai-chat-wizard/sql/store/upsert_auth_session.sql`, `examples/100-ai-chat-wizard/sql/store/list_auth_sessions.sql`, `examples/100-ai-chat-wizard/sql/store/get_auth_session_by_session_id.sql`, `examples/100-ai-chat-wizard/sql/store/get_auth_token_version.sql`, `examples/100-ai-chat-wizard/sql/store/upsert_auth_token_version.sql`, `examples/100-ai-chat-wizard/sql/store/touch_auth_session_last_seen.sql`, `examples/100-ai-chat-wizard/sql/store/revoke_auth_session.sql`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "Test(StoreAuthSessionLifecycle|AuthManagerPersistsSessionAndRevocation|ChatServerAuthRPCs|StoreSuperuserControlPlaneLifecycle)$" -count=1`
- result: Passed. Auth tokens are now backed by durable server sessions and token-version rows, and revoked or expired sessions are denied.
- residual risk: Key rotation (`kid`) and explicit per-token `jti` lifecycle policy are still pending and tracked by the next Agent 2 todo.
- next suggested todo: Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.

### 2026-03-27 22:51 America/New_York

- completed todo: Add a focused end-to-end regression for server start, WASM shell boot, and background worker boot.
- files changed: `test/playwrightgo/examples/example100_startup_boot_test.go`, `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100StartupBoot -v`
- result: Passed. New browser regression confirms `status=200`, boot-shell removal after mount, `/app/chat.wasm` + `/worker/background-worker.wasm` responses, `grpc ready`, `worker ready`, and `background render worker pool ready` startup logs with zero console/page errors.
- residual risk: This checkpoint validates startup/runtime boot only; authenticated message-send flows and route-specific UX paths are still covered by separate todos.
- next suggested todo: Verify the authenticated happy path end-to-end (login, create thread, send, stream, refresh, reopen).

### 2026-03-27 22:43 America/New_York

- completed todo: Run live browser startup smoke against `http://127.0.0.1:8095/` and capture startup console/runtime errors.
- files changed: `examples/100-ai-chat-wizard/TODO.md`, `examples/100-ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./bin/example100_startup_smoke.go` (temporary local helper removed after execution)
- result: Passed. Startup route `/` returned `200` with page title `RelayDesk – AI Chat Workspace`, `console_error_count=0`, and `page_error_count=0`.
- residual risk: This checkpoint covers only initial load at `/`; route-specific startup regressions (pricing/auth/dashboard) still require dedicated browser assertions.
- next suggested todo: Add a focused end-to-end regression that proves server start, WASM shell boot, and background worker boot.

### 2026-03-27 20:30 America/New_York

- completed todo: Repair server policy/model regression paths so `server/app` tests pass again.
- files changed: `examples/100-ai-chat-wizard/server/app/server_tool_policy.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "Test(GetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|GetSelectedModelRepairsUnsupportedPreference|GetServerToolPolicyReadsWhitelistFromSiteConfig|ProviderStubRuntimeSupportsCrossProviderSelection)"`; `go test ./examples/100-ai-chat-wizard/server/app`; `go test ./examples/100-ai-chat-wizard/server/...`
- result: Passed. Whitelist policy JSON now parses correctly and model-selection regression coverage is green.
- residual risk: Entitlement deny-by-default, usage-budget quotas, and token revocation/key-rotation remain intentionally open TODOs and are not implemented in this checkpoint.
- next suggested todo: Implement deny-by-default entitlement gating once guaranteed entitlement bootstrap rows exist for all active users.

### 2026-03-27 20:09 America/New_York

- completed todo: Gate admin analytics and diagnostics RPCs behind superuser authorization.
- files changed: `examples/100-ai-chat-wizard/server/app/admin_dashboard.go`, `examples/100-ai-chat-wizard/server/app/log_tail.go`, `examples/100-ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/100-ai-chat-wizard/server/app/log_tail_test.go`, `examples/100-ai-chat-wizard/server/app/testkit_test.go`, `examples/100-ai-chat-wizard/proto/chat.proto`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetLogTail)"`
- result: Blocked by a pre-existing compile failure in `server.go` (`undefined: parseSelectedModel`), unrelated to this authz-gating patch.
- residual risk: Authorization behavior is updated in code, but test confirmation is pending until the pre-existing server compile issue is resolved.
- next suggested todo: Add auth-secret startup validation and entitlement gate seams with TODO stubs for quota enforcement.

### 2026-03-27 20:09 America/New_York

- completed todo: Add production auth-secret validation and wire entitlement budget gate seams for `Send`.
- files changed: `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/server/app/authz_entitlement.go`, `examples/100-ai-chat-wizard/server/app/auth_service.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetLogTail)"`
- result: Blocked by the same pre-existing compile failure in `server.go` (`undefined: parseSelectedModel`) before targeted test execution.
- residual risk: Entitlement and revocation seams are currently fail-open stubs for missing billing bootstrap and missing revocation backend; they are intentionally documented as TODO for follow-up implementation.
- next suggested todo: Implement deny-by-default entitlement behavior once user-to-plan bootstrap rows are guaranteed for all users.

### 2026-03-25 02:31 America/New_York

- completed todo: Define an AI provider companion package pattern for LLM-backed GWC applications.
- files changed: `docs/ECOSYSTEM.md`, `docs/TODO.md`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The documented companion-package boundary matches the live catalog RPC path and the current wasm client surface.
- residual risk: The ecosystem guidance now captures the package boundary, but only RelayDesk currently validates the pattern, so promotion beyond `Experimental` would still require a second production-shaped app.
- next suggested todo: None in the current example-100 provider-switching slice.

### 2026-03-25 02:28 America/New_York

- completed todo: Define the SQL-backed model catalog pattern for runtime provider and model discovery.
- files changed: `examples/100-ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`
- result: Passed. The documented SQL-backed catalog path matches the live startup and RPC behavior exercised by the server tests.
- residual risk: The README now defines the recommended bootstrap payload and freshness policy, but RelayDesk still relies on authenticated RPC revalidation rather than shipping the initial catalog through SSR bootstrap today.
- next suggested todo: Define an AI provider companion package pattern for LLM-backed GWC applications.

### 2026-03-25 02:14 America/New_York

- completed todo: Promote RelayDesk as the reference implementation for runtime AI provider switching.
- files changed: `examples/100-ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `npx playwright test --config=playwright.chat-wizard.config.ts --grep "provider and model selection sync across open tabs"`
- result: Passed. The example README now explicitly positions RelayDesk as the runtime provider-switching reference app and points maintainers to the focused browser regression that proves the shipped flow.
- residual risk: The reference example now documents the shipped switching path clearly, but capability-aware filtering beyond provider membership is still not implemented in the UI.
- next suggested todo: Add capability-aware model filtering and picker messaging so RelayDesk can demonstrate why a provider or model disappears when a workflow requires a missing capability.

### 2026-03-25 01:18 America/New_York

- completed todo: Keep provider/model/intelligence selection stable across new chats and reconnects.
- files changed: `examples/100-ai-chat-wizard/client/app/helpers.go`, `examples/100-ai-chat-wizard/client/app/helpers_wasm_test.go`, `examples/100-ai-chat-wizard/client/app/model_preferences.go`, `examples/100-ai-chat-wizard/server/app/server.go`, `examples/100-ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestGetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|TestRPCFallbacksWhenStoreOrProvidersAreUnavailable"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go build ./examples/100-ai-chat-wizard/client/...`
- result: Passed. New-chat and reconnect flows now recover and persist provider/model state instead of leaving the picker blank.
- residual risk: This is covered by unit tests and compile checks, but there is still no browser-level end-to-end regression that exercises the full picker flow through a real page reload.
- next suggested todo: Add a Playwright regression that selects a non-default provider/model, reloads, starts a new chat, and verifies the same provider/model remains selected with a non-blank intelligence mode.

### 2026-03-24 22:12 America/New_York

- completed todo: Prevent newly created threads from being cleared when the first assistant response finishes.
- files changed: `examples/100-ai-chat-wizard/client/app/app.go`, `examples/100-ai-chat-wizard/client/app/route_sync.go`, `examples/100-ai-chat-wizard/client/app/route_sync_test.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/client/app -run TestShouldResetDraftForRootRoute`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The regression guard behaves correctly and the js/wasm app package still compiles.
- residual risk: This flow still lacks a browser-level end-to-end regression that drives a real send on a fresh draft thread.
- next suggested todo: Add an example-100 integration regression that seeds a fake provider response and verifies the browser stays on `/thread/:publicID` after the first reply.

### 2026-03-24 22:14 America/New_York

- completed todo: Add diagnostics for unresolved thread-route state after a fresh reply.
- files changed: `examples/100-ai-chat-wizard/client/app/app.go`, `examples/100-ai-chat-wizard/client/app/stream.go`, `examples/100-ai-chat-wizard/client/app/route_sync.go`, `examples/100-ai-chat-wizard/client/app/route_sync_test.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/client/app -run 'TestShould(ResetDraftForRootRoute|WarnPendingRootRoute)$'`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The warning guard condition is locked in and the js/wasm app package still compiles.
- residual risk: The warnings improve diagnosis but they do not replace a browser-level regression that exercises a real first-message send path.
- next suggested todo: Add an end-to-end regression with a fake provider that verifies the app stays in the new thread after the first streamed reply and asserts the warning does not appear in the healthy path.

### 2026-03-24 22:19 America/New_York

- completed todo: Flatten the provider, model, and intelligence controls so they use width more efficiently.
- files changed: `examples/100-ai-chat-wizard/client/app/panel.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The example-100 client app still compiles for js/wasm after the control-bar layout change.
- residual risk: This is a visual adjustment only; there is still no browser-level layout regression covering narrow widths and the inline control row.
- next suggested todo: Add a Playwright visual/layout smoke for the compact control bar at desktop and mobile widths.

### 2026-03-24 22:24 America/New_York

- completed todo: Add hover and press animations to the toolbar selects.
- files changed: `examples/100-ai-chat-wizard/client/app/panel.go`, `examples/100-ai-chat-wizard/client/app/styles.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The client app still compiles for js/wasm after the animated select styling pass.
- residual risk: Native option hover and press styling remain browser-dependent, so the select element motion is reliable but option-row animation fidelity will vary by platform.
- next suggested todo: Add a browser smoke that exercises the animated control bar in Chromium and confirms hover, focus, and press states remain readable.

### 2026-03-24 22:31 America/New_York

- completed todo: Add a floating down-arrow when more thread content is available below.
- files changed: `examples/100-ai-chat-wizard/client/app/app.go`, `examples/100-ai-chat-wizard/client/app/app_shell.go`, `examples/100-ai-chat-wizard/client/app/constants.go`, `examples/100-ai-chat-wizard/client/app/helpers.go`, `examples/100-ai-chat-wizard/client/app/panel.go`, `examples/100-ai-chat-wizard/client/app/scroll_memory.go`, `examples/100-ai-chat-wizard/client/app/scroll_visibility.go`, `examples/100-ai-chat-wizard/client/app/scroll_visibility_test.go`, `examples/100-ai-chat-wizard/client/app/thread.go`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/100-ai-chat-wizard/client/app -run TestHasScrollSpaceBelow`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The visibility helper is locked in and the js/wasm client app still compiles with the floating jump-to-bottom control.
- residual risk: Browser-level placement and overlap behavior still need a visual smoke pass, especially with split canvas mode and long threads.
- next suggested todo: Add a Playwright scroll smoke that verifies the button appears when the user scrolls up and jumps back to the latest message when clicked.

### 2026-03-24 22:37 America/New_York

- completed todo: Rebrand the example-100 product surface away from the framework name.
- files changed: `examples/100-ai-chat-wizard/client/app/constants.go`, `examples/100-ai-chat-wizard/server/app/bootstrap.go`, `examples/tests/100-ai-chat-wizard.spec.ts`, `examples/100-ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. The visible app brand and server-rendered page title now use the new product name, and the client app still compiles for js/wasm.
- residual risk: This updates the primary visible branding, but supporting copy such as the empty-state headline still reads like an experiment rather than a polished product surface.
- next suggested todo: Refresh the empty-state and onboarding copy so the rest of the home screen matches the new product branding.
