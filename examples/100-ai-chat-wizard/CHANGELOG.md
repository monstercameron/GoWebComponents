# Example 100 Changelog

## Checkpoints

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
