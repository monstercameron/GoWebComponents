# Example 100 Changelog

## Commit Alignment

### 2026-03-28 13:35 America/New_York

- reviewed commits: `4111030` (`build example100 workspace and interop followups`) and `e267cd4` (`build example100 provider drilldown support`)
- scope captured:
  - added typed provider drilldown proto contracts in `proto/chat.proto`, including `GetAdminProvidersDrilldown`, `AdminProvidersDrilldownSummary`, and `WorkspaceModelRoutingPolicyEntry`
  - added store/query support for workspace model-routing policy reads in `server/app/store_superuser.go`, `server/app/queries.go`, and `sql/store/ops/list_workspace_model_routing_policies.sql`
  - landed small workspace-admin shell/table followups in `client/app/admin_workspaces.go`, `client/app/app.go`, and `client/app/app_shell.go`
  - expanded `interop/interop_wasm_test.go` with browser-lane worker-surface validation for invalid worker inputs, nil subscribe handlers, empty request names, and Go-wasm worker bootstrap descriptor failures
- validation note: the commit metadata did not record focused validation commands, so this alignment entry captures reviewed scope rather than reconstructing unverified test runs

## Checkpoints

### 2026-06-12 19:35 America/New_York

- completed todo: Complete the systems writeup pass for Example 100 and add a chapter-order README index.
- files changed: `examples/server/ai-chat-wizard/docs/HOW_EXAMPLE_100_WORKS.md`, `examples/server/ai-chat-wizard/docs/PUBLIC_ROUTE_DELIVERY.md`, `examples/server/ai-chat-wizard/docs/AUTHENTICATED_SHELL.md`, `examples/server/ai-chat-wizard/docs/CHAT_REQUEST_LIFECYCLE.md`, `examples/server/ai-chat-wizard/docs/ADMIN_DASHBOARD_SUBSYSTEM.md`, `examples/server/ai-chat-wizard/docs/DATA_LAYER.md`, `examples/server/ai-chat-wizard/docs/OBSERVABILITY_FAILURE_HANDLING.md`, `examples/server/ai-chat-wizard/docs/EXTENSION_SEAMS.md`, `examples/server/ai-chat-wizard/docs/SYSTEMS_GLOSSARY.md`, `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/DOCS_MAP.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard -count=1 -run "^(TestValidateAgent4DocSectionCoverage|TestValidateAgent4DocPathLayout|TestValidateAgent4PricingVocabularyContract)$"`; systems-doc path check for all nine new `docs/*.md` chapters; `rg -n "Systems Writeup Chapter Order|HOW_EXAMPLE_100_WORKS|PUBLIC_ROUTE_DELIVERY|AUTHENTICATED_SHELL|CHAT_REQUEST_LIFECYCLE|ADMIN_DASHBOARD_SUBSYSTEM|DATA_LAYER|OBSERVABILITY_FAILURE_HANDLING|EXTENSION_SEAMS|SYSTEMS_GLOSSARY" examples/server/ai-chat-wizard/README.md examples/server/ai-chat-wizard/DOCS_MAP.md examples/server/ai-chat-wizard/TODO.md examples/server/ai-chat-wizard/CHANGELOG.md`
- result: Passed. The docs contract tests stayed green, every new systems chapter file exists, and README/docs-map/TODO/changelog references point at the new chapter set.
- residual risk: The writeups are source-linked documentation and do not change runtime behavior.
- next suggested todo: Continue with the remaining reconnect/bridge regression and telemetry-isolation documentation items above the systems writeup slice.

### 2026-06-12 America/New_York

- completed todo: Add one customer-billing truth map that ties the settings billing labels and sections to their canonical backend source RPCs, invoice-line classes, and pricing vocabulary so future billing-surface work does not drift back into derived or mismatched totals.
- completed todo: Add one admin diagnostics playbook for log tail and server-tool review so future superuser ops surfaces come with a clear operator loop.
- completed todo: Close the README framework-demo and source-map documentation TODO slice covering why/start-here, framework/public-route pattern maps, source-linked UI inventory, dashboard teaching pass, paired mini-examples, framework smoke checklist, current-practice notes, route/data/SQL maps, placement guide, and source-map verification checklist.
- completed todo: Collapse the public marketing IA and remove unsupported public capability claims from kept marketing/pricing routes.
- files changed: `examples/server/ai-chat-wizard/OPERATOR_RUNBOOK.md`, `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`, `examples/server/ai-chat-wizard/client/app/landing_shell.go`, `examples/server/ai-chat-wizard/client/app/landing_hero.go`, `examples/server/ai-chat-wizard/client/app/marketing_shared.go`, `examples/server/ai-chat-wizard/client/app/pricing_shell.go`, `examples/server/ai-chat-wizard/server/app/pricing_page_content_ops.go`
- validation run: `rg -n "Customer-Billing Truth Map|Admin Diagnostics Playbook|platform fee|service premium|GetSuperuserOpsDiagnostics|server-tool policy|audit_logs" examples/server/ai-chat-wizard/OPERATOR_RUNBOOK.md`; `rg -n "Framework Pattern Map|Public Route Pattern Map|Paired Mini-Examples|Framework-Focused Smoke Checklist|Current Best-Practice Notes|Route-To-Code Map|Hot-Path Data Flow Traces|SQL Ownership Map|Where To Put New Code|Source-Map Verification Checklist" examples/server/ai-chat-wizard/README.md`; focused marketing `rg` checks for `/capabilities` aliasing, footer destinations, and unsupported SSO/doc-claim copy
- result: Passed. The operator runbook now anchors billing labels and invoice classes to canonical backend/store ownership and gives superuser diagnostics a repeatable log-tail, server-tool, audit, and runtime-log correlation loop. The README now makes Example 100 readable as a framework pattern catalog and source map. The marketing route cleanup keeps `/home`, `/pricing`, `/signup`, auth entry, settings, and dashboard as first-class paths, treats `/plans` as pricing, and removes dead footer destinations plus unsupported SSO-ready copy.
- residual risk: These were docs/runbook TODOs; existing runtime tests continue to own billing math and superuser diagnostics behavior.
- next suggested todo: Continue bottom-up through the remaining thread-management and admin-dashboard TODO clusters.

### 2026-03-28 14:26 America/New_York

- completed todo: Group the current Example 100 admin/operator backend slice plus the related interop/docs follow-up work into commit-ready chunks and revalidate them.
- files changed: `examples/server/ai-chat-wizard/proto/*`, `examples/server/ai-chat-wizard/server/app/*`, `examples/server/ai-chat-wizard/sql/store/*`, `examples/server/ai-chat-wizard/client/app/dashboard*.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`, `interop/interop_wasm.go`, `interop/interop_wasm_test.go`, `docs/TODO.md`, `CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestGetAdminBusinessQueueAndAccountDetail|TestGetAdminBusinessQueueAndAccountDetailScopeGuards|TestGetAdminCustomerAccountTimeline|TestGetAdminCustomerAccountTimelineScopeGuards|TestGetAdminChatsAnomalies|TestGetAdminChatsAnomaliesWorkspaceScope|TestGetAdminOpsDrilldown|TestGetAdminOpsDrilldownScopeGuards|TestAdminOpsMutationRPCs|TestAdminOpsMutationRPCScopeGuards|TestGetAdminProvidersDrilldown|TestGetAdminProvidersDrilldownScopeGuards|TestAdminProviderMutationRPCs|TestAdminProviderMutationRPCScopeGuards|TestProviderUsageDailyRollups)$"`; `GOOS=js GOARCH=wasm go build ./examples/server/ai-chat-wizard/client/app/...`; `GOOS=js GOARCH=wasm go test -exec C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat ./interop -count=1 -run TestWorkerSurfaceValidationReportsFailures$`; `GOOS=js GOARCH=wasm go test -exec C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat ./interop -count=1 -run TestCrossSurfaceMessagingWrappersReportMalformedAndInactiveStates$`; `GOOS=js GOARCH=wasm go test -exec C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat ./interop -count=1 -run TestStructuredCloneBoundaryRejectsUnsupportedPayloadsAndInvalidTransfers$`
- result: Passed. The admin/operator slice now has generated proto coverage plus focused server tests for business, customers, chats anomalies, providers, ops, and provider rollups, and the new worker-surface validation tests are green after fixing URL-resolution and closed-peer contract bugs in interop.
- residual risk: `GetAdminProvidersDrilldown` still derives provider/model usage rows from platform-wide aggregates before workspace-specific routing/guardrail filtering, so the workspace-scoped providers surface should be reviewed again before treating it as a finished operator-grade isolation story.
- next suggested todo: Land the remaining Agent 3 operator surfaces: provider health/fallback history, ops queue/actions, and mutation-preview blast-radius RPCs.

### 2026-03-28 11:41 America/New_York

- completed todo: Group the current example-100 auth/runtime, client/admin, SQL-organization, and docs/testing-gap work into logical commit-ready slices.
- files changed: `examples/server/ai-chat-wizard/server/app/*`, `examples/server/ai-chat-wizard/server/provider/*`, `examples/server/ai-chat-wizard/proto/*`, `examples/server/ai-chat-wizard/sql/store/*`, `examples/server/ai-chat-wizard/client/app/*`, `examples/server/ai-chat-wizard/client/backgroundworker/main.go`, `test/playwrightgo/examples/*`, `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/docs/*`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `git diff --stat` over backend/runtime, client/test, and docs/backlog slices plus targeted readback of the new testing-gap entries added to `TODO.md`.
- result: The current example-100 worktree is now checkpointed in the changelog as three logical slices: backend/auth-runtime plus SQL reorganization, client/admin UI plus Playwright coverage, and docs/backlog/audit updates including the full testing-gap sweep for god-tier demo readiness.
- residual risk: This checkpoint records grouped worktree state before commit, not one narrowly validated feature landing; runtime/package validation still needs to be considered per slice when those commits are reviewed later.
- next suggested todo: Commit the grouped backend/runtime, client/test, and docs/backlog slices separately and keep the loose root artifacts plus dirty submodule out of the example-history commits.

### 2026-03-28 11:29 America/New_York

- completed todo: Extend server shell route gating so `/plans`, `/about`, `/contact`, `/privacy`, `/terms`, `/security`, and `/status` are first-class direct-load routes with consistent public-route normalization.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/funnel_first_chat.go`, `examples/server/ai-chat-wizard/server/app/runtime_helpers_additional_test.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestChatShellRoutingHelpers|TestChatShellHandlerTracksFirstChatFunnelSteps|TestChatShellHandler)$"`
- result: Passed. Public route shell delivery now includes `/plans` and the new info/legal routes, and first-chat funnel normalization now treats `/plans` as the same pricing-view step as `/pricing`.
- residual risk: Browser-level hard-refresh/open-in-new-tab regression coverage for this route set is still pending and tracked separately.
- next suggested todo: Add public password-reset request and update-password consume paths with token lifecycle and clear success/expired outcomes.

### 2026-03-28 11:20 America/New_York

- completed todo: Replace remaining store-unavailable no-op branches in launch-critical profile/settings writes (`SetCustomSystemPrompt`, `UpsertUserMemory`, `DeleteUserMemory`) with typed unavailable paths and correlated diagnostics.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/customer_safe_error_contract.go`, `examples/server/ai-chat-wizard/server/app/customer_safe_error_contract_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestBuildCustomerSafeErrorContractIncludesSupportReference|TestWrapCustomerSafeRPCErrorAttachesTypedDetails|TestBuildCustomerSafeErrorUnaryInterceptorSkipsOutOfScopeMethods|TestSettingsWriteRPCsNoStoreReturnTypedUnavailable|TestMemoryRPCFallbacksWithoutStore|TestMemoryRPCValidationAndErrorBranches)$"`
- result: Passed. Store-unavailable write paths for custom prompt and user-memory mutations now fail closed through one typed contract helper (`parseBuildStoreUnavailableRPCStatus`) with customer-safe reference IDs and correlated operator lookup metadata.
- residual risk: Read-only fallback RPCs (`GetUserName`, `ListUserMemories`, `GetCustomSystemPrompt`) still intentionally return defaults when store is nil; they should be reviewed separately if stricter outage behavior is required.
- next suggested todo: Extend the server shell route gate so `/plans`, `/about`, `/contact`, `/privacy`, `/terms`, `/security`, and `/status` are direct-load first-class routes.

### 2026-03-28 11:14 America/New_York

- completed todo: Add one typed customer-safe error contract for chat/auth/settings/dashboard failures with friendly message, stable support/request ID, and server-log correlation path.
- files changed: `examples/server/ai-chat-wizard/server/app/customer_safe_error_contract.go`, `examples/server/ai-chat-wizard/server/app/customer_safe_error_contract_test.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/client/app/user_error.go`, `examples/server/ai-chat-wizard/client/app/user_error_wasm_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestBuildCustomerSafeErrorContractIncludesSupportReference|TestWrapCustomerSafeRPCErrorAttachesTypedDetails|TestBuildCustomerSafeErrorUnaryInterceptorSkipsOutOfScopeMethods)$"`; `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_user_error.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. Added one scoped gRPC error-contract interceptor that wraps chat/auth/settings/dashboard failures with typed `ErrorInfo` metadata (`CUSTOMER_SAFE_ERROR_V1`) and logs one stable operator lookup path. Client error rendering now consumes typed contract message/reference data so user-visible failures keep friendly copy plus consistent Request IDs.
- residual risk: The contract currently covers customer-facing RPC families by method mapping and metadata details; broader dashboard method additions should be kept in sync as new admin RPCs are added.
- next suggested todo: Replace remaining store-unavailable no-op branches in `SetCustomSystemPrompt`, `UpsertUserMemory`, and `DeleteUserMemory` with typed unavailable/error paths plus correlated diagnostics.

### 2026-03-28 10:34 America/New_York

- completed todo: Implement policy-aware external identity link-or-create resolution.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_identity_linking.go`, `examples/server/ai-chat-wizard/server/app/auth_identity_linking_test.go`, `examples/server/ai-chat-wizard/server/app/auth_google_oidc.go`, `examples/server/ai-chat-wizard/server/app/auth_google_oidc_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestResolveExternalIdentityLinkDecision|TestResolveExternalIdentityLinkDecisionRejectsInvalidProvider|TestResolveExternalIdentityLinkDecisionEnforcesPolicy|TestHandleOIDCProviderCallbackCreateUser|TestHandleOIDCProviderCallbackEnforcesWorkspaceLinkPolicy|TestHandleGoogleOIDCCallbackCreateUser|TestHandleGoogleOIDCCallbackLinkExisting)$"`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestResolveWorkspaceAuthPolicyDefaults|TestResolveWorkspaceAuthPolicyExplicit|TestResolveWorkspaceAuthPolicyInfersRequiredProvider|TestResolveWorkspaceAuthPolicyRejectsAmbiguousRequiredProvider)$"`
- result: Passed. External identity decisioning now includes explicit policy gates for linking to existing password accounts and creating new users, while keeping ambiguous/unsafe subject-email match handling fail-closed. OIDC callback orchestration now feeds resolved workspace policy into link/create decisioning so policy-denied paths are enforced during callback handling.
- residual risk: This enforces link/create gating at callback decision time, but typed audit records for policy-denied external-login events are still pending in the next todo.
- next suggested todo: Add typed audit events and queryable auth records for external login start/callback/link/policy-denied paths.

### 2026-03-28 10:30 America/New_York

- completed todo: Add typed workspace auth-policy resolution for password/external/SSO-required/provider/JIT decisions.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_workspace_policy.go`, `examples/server/ai-chat-wizard/server/app/auth_workspace_policy_resolution_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestAuthorizeWorkspaceLoginMethod|TestAuthorizeWorkspaceLoginMethodRejectsInvalidPolicy|TestAuthorizeWorkspaceSSORegressionMatrix|TestResolveWorkspaceAuthPolicyDefaults|TestResolveWorkspaceAuthPolicyExplicit|TestResolveWorkspaceAuthPolicyInfersRequiredProvider|TestResolveWorkspaceAuthPolicyRejectsAmbiguousRequiredProvider)$"`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestStoreSuperuserReliabilityControlFuncs|TestSuperuserReliabilityControlRPCs)$"`
- result: Passed. Added one typed workspace policy-resolution contract (`parseWorkspaceAuthPolicyResolution`) and resolver (`parseResolveWorkspaceAuthPolicy`) that combines persisted `workspace_auth_policies` with enabled `workspace_sso_configs` to answer password-allowed, external-login-optional, SSO-required, required-provider, and JIT-provisioning outcomes in one place, with fail-closed handling for missing/ambiguous SSO provider selection.
- residual risk: This adds the typed resolver seam and tests, but login RPC wiring that consumes the resolver for end-user auth UX still depends on remaining external-auth rollout items.
- next suggested todo: Implement link-or-create account resolution policy seam so external identities attach/create/reject safely under workspace policy controls.

### 2026-03-28 10:24 America/New_York

- completed todo: Rework `workspace_sso_configs` so generic OIDC configuration is first-class while keeping SAML as a compatibility subtype.
- files changed: `examples/server/ai-chat-wizard/sql/store/schema.sql`, `examples/server/ai-chat-wizard/sql/store/migrations.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_workspace_sso_config.sql`, `examples/server/ai-chat-wizard/sql/store/list_workspace_sso_configs.sql`, `examples/server/ai-chat-wizard/sql/store/get_workspace_sso_config_by_scope.sql`, `examples/server/ai-chat-wizard/server/app/store_superuser_reliability.go`, `examples/server/ai-chat-wizard/server/app/superuser_reliability_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_reliability_ops_test.go`, `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestStoreSuperuserReliabilityControlFuncs|TestStoreSuperuserReliabilityListFuncs|TestSuperuserReliabilityControlRPCs)$"`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestGetSuperuserControlPlaneReturnsSnapshot|TestGetSuperuserControlPlaneRequiresSURole)$"`; `go test ./examples/server/ai-chat-wizard/proto -count=1`
- result: Passed. `workspace_sso_configs` now carries typed `provider_type` plus OIDC config fields (`oidc_issuer_url`, `oidc_client_id`, `oidc_client_secret_ref`, `oidc_scopes_json`, `oidc_claims_json`) across schema, SQL queries, store models, and superuser reliability RPC payloads. Legacy SAML fields remain supported, with provider-type inference preserving SAML-only rows as `saml`.
- residual risk: While schema/store/RPC contracts now support generic OIDC config shape, runtime OIDC discovery/verification and workspace-level auth-policy enforcement wiring remain tracked in later auth todos.
- next suggested todo: Add typed workspace auth-policy resolution so login can answer password/external/SSO-required/provider/JIT decisions (next Agent 3 unchecked item).

### 2026-03-28 10:09 America/New_York

- completed todo: Implement Google OIDC typed start/callback handlers with state+nonce persistence, link-or-create resolution, and auth-session token issuance.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_google_oidc.go`, `examples/server/ai-chat-wizard/server/app/auth_google_oidc_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestHandleGoogleOIDCStartStoresState|TestHandleGoogleOIDCCallbackCreateUser|TestHandleGoogleOIDCCallbackLinkExisting)$"`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestAuthorizeExternalHandshake|TestResolveExternalIdentityLinkDecision|TestStoreAuthIdentityLifecycle|TestStoreAuthOIDCStateLifecycle|TestStoreWorkspaceAuthPolicyLifecycle|TestHandleGoogleOIDCStartStoresState|TestHandleGoogleOIDCCallbackCreateUser|TestHandleGoogleOIDCCallbackLinkExisting)$"`
- result: Passed. Added `parseHandleGoogleOIDCStart` and `parseHandleGoogleOIDCCallback` orchestration over persisted OIDC state, callback guardrails, canonical identity upsert, verified-email account link-or-create decisioning, and final token issuance through existing auth-session infrastructure (`issueTokenForContextWithAuthMethod` with `google_oidc` method).
- residual risk: These handlers are currently internal server-app seams; proto/RPC transport exposure and browser callback-route wiring remain pending for full end-to-end UI flow.
- next suggested todo: Add generic OIDC provider path support so enterprise providers can reuse the same pipeline without Google-specific branching (line 417).

### 2026-03-28 10:01 America/New_York

- completed todo: Add one canonical external-auth provider model carrying provider key/type, subject, verified-email, profile payload, and last-login timestamps.
- files changed: `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestStoreAuthIdentityLifecycle)$"`
- result: Passed. Canonical identity model fields are now explicitly represented and exercised through `parseAuthIdentityWrite`/`parseAuthIdentityRow` lifecycle coverage, matching the `auth_identities` schema contract used for Google/OIDC identity linking.
- residual risk: This checkpoint codifies the shared persistence shape, but provider-specific callback payload mapping and RPC orchestration are still pending in the next OIDC implementation todos.
- next suggested todo: Implement Google OIDC start/callback handlers with state+nonce persistence and final session issuance (line 416).

### 2026-03-28 10:00 America/New_York

- completed todo: Add first-class external-identity store funcs for `auth_identities`, `auth_oidc_states`, and workspace auth-policy persistence.
- files changed: `examples/server/ai-chat-wizard/server/app/store_auth_external.go`, `examples/server/ai-chat-wizard/server/app/store_auth_external_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestStoreAuthIdentityLifecycle|TestStoreAuthOIDCStateLifecycle|TestStoreWorkspaceAuthPolicyLifecycle)$"`
- result: Passed. Added typed Store write/read/list/delete lifecycle funcs for external identity links, OIDC handshake state creation/consumption/expiry cleanup, and workspace auth-policy upsert/read behavior, with focused lifecycle tests validating each path.
- residual risk: This slice adds persistence primitives only; end-to-end Google/OIDC start/callback RPC wiring and audit emission flows remain pending in later auth todos.
- next suggested todo: Add one canonical provider model for external auth (`provider_key`, `provider_type`, subject, verified-email, profile payload, last-login timestamps) and thread it through the auth decision helpers (line 414).

### 2026-03-28 18:05 America/New_York

- completed todo: Rebalanced the example-100 backlog so Agent 4 owns more non-UI engineering, logging, traceability, and failure-handling work, while Agent 5 keeps only UI-facing error and canvas/chat-surface follow-up items.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: targeted `rg` and direct readback of `TODO.md` sections for Agent 4 and Agent 5 ownership
- result: Passed. Logging/error/naming follow-up work now sits under Agent 4, UI-facing customer-error treatment remains under Agent 5, and the canvas/chat-surface styling follow-up is tracked as dedicated UI work instead of mixed backlog notes.
- residual risk: This is backlog-only coordination work; implementation and test coverage for the new items are still pending.
- next suggested todo: Resume the next unchecked item in Agent 4 or Agent 5 depending on whether the next slice is backend/error hardening or UI refinement.

### 2026-03-28 09:52 America/New_York

- completed todo: Implement Cerebras memory extraction parity in `ParseExtractUserMemories` with deterministic JSON-object parsing and provider-agnostic fallback normalization.
- files changed: `examples/server/ai-chat-wizard/server/provider/cerebras_provider.go`, `examples/server/ai-chat-wizard/server/provider/provider_http_additional_test.go`
- validation run: `go test ./examples/server/ai-chat-wizard/server/provider -count=1 -run "^(TestCerebrasProviderStreamingHTTPBackedBranches|TestCerebrasProviderHTTPBackedBranches)$"`; `go test ./examples/server/ai-chat-wizard/server/provider -count=1`
- result: Passed. Cerebras extraction now uses `response_format.type = json_object`, decodes candidates through the shared provider-agnostic parser, normalizes candidate bounds/keys/category via `parseNormalizeMemoryCandidates`, and safely falls back to an empty candidate list when payload parsing fails.
- residual risk: This validates provider-level extraction behavior, but cross-provider extraction UX consistency in remembered-preferences UI still depends on downstream server/app integration coverage.
- next suggested todo: Add first-class external-identity tables and store funcs for `auth_identities`, `auth_oidc_states`, and workspace auth policy (line 413).

### 2026-03-28 09:48 America/New_York

- completed todo: Implement Anthropic memory extraction parity in `ParseExtractUserMemories` with structured tool-output parsing, normalized candidate coercion, and malformed-response fallback safety.
- files changed: `examples/server/ai-chat-wizard/server/provider/anthropic_provider.go`, `examples/server/ai-chat-wizard/server/provider/provider_helpers_test.go`, `examples/server/ai-chat-wizard/server/provider/provider_http_additional_test.go`, `examples/server/ai-chat-wizard/server/provider/provider_streaming_additional_test.go`
- validation run: `go test ./examples/server/ai-chat-wizard/server/provider -count=1 -run "^(TestAnthropicProviderHelperAndFallbackBranches|TestAnthropicProviderStreamingHTTPBackedBranches|TestAnthropicProviderHTTPBackedBranches)$"`; `go test ./examples/server/ai-chat-wizard/server/provider -count=1`
- result: Passed. Anthropic extraction now sends an explicit tool-choice + JSON-schema contract (`extract_user_memories`), parses tool-use payloads with text compatibility fallback, coerces mixed key formats (`usefulness_score`/`usefulnessScore`, etc.), and returns an empty candidate list when provider output is malformed instead of surfacing parse errors.
- residual risk: This change is provider-package covered, but end-to-end remembered-preferences UX behavior across full server flows still depends on integration paths outside the provider package.
- next suggested todo: Implement Cerebras memory extraction parity in `ParseExtractUserMemories` (line 411).

### 2026-03-28 13:52 America/New_York

- completed todo: Replace stale control-mutation authz stubs with one live shared helper and remove placeholder `Unimplemented` seam tests.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_control_mutation_authz.go`, `examples/server/ai-chat-wizard/server/app/admin_control_mutation_authz_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestAdminControlMutationScopeBoundaries|TestStoreAdminControlMutationFuncs|TestAdminControlOpsRPCs|TestAdminControlOpsWorkspaceScope)$"`
- result: Passed. Real control-plane RPCs continue to use the shared `parseAuthorizeAdminControlMutationScope` helper, and tests now assert live scope outcomes instead of `Unimplemented` placeholder behavior.
- residual risk: Coverage is currently focused on authz/scope correctness; deeper mutation side-effect assertions remain in the dedicated control-ops tests.
- next suggested todo: Implement Anthropic memory extraction parity in `ParseExtractUserMemories` (line 408).

### 2026-03-28 13:43 America/New_York

- completed todo: Implement `RunServerTool` end to end with typed runtime session management, stream-frame semantics, byte/time policy enforcement, execution audit persistence, and clean shutdown behavior.
- files changed: `examples/server/ai-chat-wizard/server/app/server_tool_stub.go`, `examples/server/ai-chat-wizard/server/app/server_tool_runtime.go`, `examples/server/ai-chat-wizard/server/app/server_tool_stub_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestGetServerToolPolicyRequiresSURole|TestGetServerToolPolicyReturnsDefaults|TestSetServerToolPolicyRequiresDangerousChangeConfirmation|TestSetServerToolPolicyValidatesRuleArgumentPolicy|TestSetServerToolPolicyAppendsHistoryRows|TestRunServerToolRequiresStartPayload|TestRunServerToolEnforcesWhitelistAndArgumentPolicy|TestRunServerToolAllowedCommandStreamsStartedAndExit|TestRunServerToolStdinRequiresMatchingSessionID|TestSensitiveSuperuserMutationsRequireFreshSession|TestRunServerToolRequiresStream|TestGetServerToolPolicyReadsWhitelistFromSiteConfig)$"`
- result: Passed. `RunServerTool` now executes approved commands, emits started/output/exit/error stream events, handles stdin/signal/close frames with session-id checks, enforces output-byte and timeout caps, and records started/completed/failed audit events.
- residual risk: Runtime execution currently supports one active session per stream and relies on command-token parsing without quoted-arg support; broader shell/runtime parity and multi-session orchestration are intentionally out of scope for this slice.
- next suggested todo: Replace the stale control-mutation authz stubs in `admin_control_mutation_authz.go` with one live shared helper used by the real feature-flag, experiment, and incident mutation RPCs (line 400).

### 2026-03-28 13:24 America/New_York

- completed todo: Implement `SetServerToolPolicy` end to end with persisted policy fields/rules, typed history store funcs + SQL, audit rows, and applied-snapshot response payload.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/server_tool_stub.go`, `examples/server/ai-chat-wizard/server/app/server_tool_security.go`, `examples/server/ai-chat-wizard/server/app/store_server_tool_policy.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/server_tool_stub_test.go`, `examples/server/ai-chat-wizard/sql/store/create_server_tool_policy_history.sql`, `examples/server/ai-chat-wizard/sql/store/list_server_tool_policy_history.sql`, `examples/server/ai-chat-wizard/sql/store/schema.sql`, `examples/server/ai-chat-wizard/sql/store/migrations.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestGetServerToolPolicyRequiresSURole|TestGetServerToolPolicyReturnsDefaults|TestSetServerToolPolicyRequiresDangerousChangeConfirmation|TestSetServerToolPolicyValidatesRuleArgumentPolicy|TestSetServerToolPolicyAppendsHistoryRows|TestRunServerToolRequiresStartPayload|TestRunServerToolEnforcesWhitelistAndArgumentPolicy|TestRunServerToolAllowedCommandStillFailsWhenRuntimeUnavailable|TestSensitiveSuperuserMutationsRequireFreshSession|TestRunServerToolRequiresStream|TestGetServerToolPolicyReadsWhitelistFromSiteConfig)$"`
- result: Passed. `SetServerToolPolicy` now applies validated policy snapshots, writes immutable `server_tool_policy_history` rows, emits audit events for denied/set outcomes, and returns the applied snapshot fields directly in the mutation response.
- residual risk: `RunServerTool` remains runtime-unavailable and only policy-gates start payloads; full terminal session manager/streaming semantics are still pending.
- next suggested todo: Implement `RunServerTool` end to end (line 399).

### 2026-03-28 Agent-5 Auth/Settings/Dashboard Copy

- completed todos: /signup copy, / auth landing copy, /app/settings copy, dashboard-entry copy.
- files changed: `examples/server/ai-chat-wizard/server/catalog/bundle.go`, `examples/server/ai-chat-wizard/client/app/dashboard.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go build ./examples/server/ai-chat-wizard/server/catalog/...`; `cmd /c "set GOOS=js&& set GOARCH=wasm&& go build ./examples/server/ai-chat-wizard/client/app/..."`
- result: Passed. Signup hero rewritten to workspace-creation framing with billing-formula stat cards in EN/ES/FR. Auth landing rewritten to "Welcome back / Your workspace is one sign-in away" framing in EN/ES/FR. Reset hero tightened to workspace-return language. Settings copy: "TTS providers" → "Voice", profile nav summary is a real sentence, memories nav summary explains value not action, billing nav summary and help text are formula-explicit, profile usage title is time-bound. Dashboard: role banner lists what to check and where; slice subtitles rewritten in operator-action language; section renamed to "Admin surfaces"; account summary uses "Chats", "Model cost", "Service premium" labels.
- residual risk: signup_shell.go may contain additional hardcoded copy not covered by bundle.go changes.
- next suggested todo: Check signup_shell.go for un-rewritten hardcoded copy; then move to admin list views with search/filter/pagination.

### 2026-03-28 10:35 America/New_York

- completed todo: Replace the legacy local seed identities in `cmd/seed-test-db` with one stable QA pair, `customer@email.com / password` and `admin@email.com / password`, and keep role/bootstrap helpers aligned for local auth, first-chat, billing, and dashboard smoke flows.
- files changed: `examples/server/ai-chat-wizard/cmd/seed-test-db/main.go`, `examples/server/ai-chat-wizard/cmd/seed-test-db/main_test.go`, `tools/gwc/main.go`, `tools/gwc/seed_test.go`, `test/playwrightgo/examples/example100_authenticated_happy_path_test.go`, `test/playwrightgo/examples/example100_admin_journey_test.go`, `test/playwrightgo/examples/example100_billing_summary_regression_test.go`, `test/playwrightgo/examples/example100_route_smoke_test.go`, `test/playwrightgo/examples/example100_visit_first_chat_test.go`, `test/playwrightgo/examples/example100_scroll_to_bottom_test.go`, `test/playwrightgo/examples/example100_admin_role_guard_test.go`, `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/OPERATOR_RUNBOOK.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/cmd/seed-test-db -count=1`; `go test ./tools/gwc -count=1 -run "^(TestRunSeedJSONExecutesDefaultSeederWithKnownCredentials|TestPrintSeedSummary)$"`; `go test -c -tags playwrightgo -o ./bin/playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Seed command now creates the canonical QA accounts directly (`customer@email.com`, `admin@email.com`) and related seed summaries, docs, and seeded-login Playwright regressions use the same credentials.
- residual risk: Full Playwright runtime execution was compile-validated only; browser runtime timing/behavior still depends on local Playwright environment and should be executed separately when needed.
- next suggested todo: Implement `SetServerToolPolicy` end to end (line 395).

### 2026-03-29 Agent-5 Pricing Rewrite

- completed todos: Pricing hero stat cards, plans h2, comparison table, and FAQ rewrite (EN/ES/FR); pricing shell compareRowKeys and footer link fixes.
- files changed: `examples/server/ai-chat-wizard/server/catalog/bundle.go`, `examples/server/ai-chat-wizard/client/app/pricing_shell.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `cmd /c "set GOOS=js&& set GOARCH=wasm&& go build ./examples/server/ai-chat-wizard/client/app/..."`; `go build ./examples/server/ai-chat-wizard/server/catalog/...`
- result: Passed. Hero stat cards now describe the billing formula (platform fee, actual AI usage, service premium) in EN/ES/FR. Plans h2 updated to workspace-framing copy. 8-row compare table (seats/models/shared/admin/api/residency/retention/sla) replaced with 6-row table (workspace mode, collaboration, admin controls, billing visibility, support, security/compliance) in all three locales. FAQ replaced with the 5 billing-formula-specific questions in EN/ES/FR. `compareRowKeys` in pricing_shell.go updated to match new 6 keys. Footer Company/Legal columns wired to real marketing routes instead of `#` placeholders.
- residual risk: Pricing browser regression test (`TestExample100PricingRegression`) asserts absence of `unlimited` and `no token caps` language — passes. Compare table row count changed so snapshot-style tests checking exact row counts would need updating if they exist.
- next suggested todo: Rewrite the kept `/signup` page with concrete conversion copy.

### 2026-03-28 09:16 America/New_York

- completed todo: Add typed server endpoints for boot-catalog injection and lazy namespace fetch, plus version/hash metadata so the client can cache catalogs safely and refetch only when content changes.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/catalog_ops.go`, `examples/server/ai-chat-wizard/server/app/catalog_ops_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestGetCatalogBootstrapRPC|TestGetCatalogBootstrapRPCNotModified|TestGetCatalogNamespaceRPC|TestBuildCatalogLoader)"`
- result: Passed. Added typed catalog bootstrap and namespace RPCs with deterministic version/hash metadata and not-modified responses to support client cache validation and lazy namespace refetch.
- residual risk: Client-side bootstrap wiring and lazy namespace consumption are still pending, so these RPCs are available but not yet used by the runtime.
- next suggested todo: Add typed billing-plan fields, store funcs, SQL queries, and admin mutation RPCs for the usage-based pricing formula: `monthly_platform_fee_cents`, `usage_premium_basis_points`, `workspace_mode`, `min_seats`, `max_seats`, and the collaboration/admin capability flags that distinguish `Pro` from `Team`.

### 2026-03-28 09:07 America/New_York

- completed todo: Move the current client-owned strings into server-owned sources behind a loader interface that starts file- or Go-backed now and can later swap to DB/CMS storage without changing the client contract.
- files changed: `examples/server/ai-chat-wizard/server/catalog/bundle.go`, `examples/server/ai-chat-wizard/client/app/i18n.go`, `examples/server/ai-chat-wizard/server/app/catalog_loader.go`, `examples/server/ai-chat-wizard/server/app/catalog_loader_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^TestBuildCatalogLoader"`; `go test ./examples/server/ai-chat-wizard/server/catalog -count=1`; `cmd /c "set GOOS=js&& set GOARCH=wasm&& go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app"`
- result: Passed. Localization catalog registrations now live in a server-owned Go source package and the client imports that source, while server-side catalog loading is abstracted behind a pluggable source interface with Go-backed default and source-swap coverage.
- residual risk: Loader seam is in place, but no RPC exposure exists yet; catalog payloads are not fetched lazily by namespace until the next Agent 3 endpoint todo is implemented.
- next suggested todo: Add typed server endpoints for boot-catalog injection and lazy namespace fetch, plus version/hash metadata so the client can cache catalogs safely and refetch only when content changes.

### 2026-03-28 09:00 America/New_York

- completed todo: Add a typed server-owned catalog contract with namespace, locale, version, fallback-locale, source-layer metadata, and message payload fields so the client is not coupled to ad hoc copy transport.
- files changed: examples/server/ai-chat-wizard/proto/chat.proto, examples/server/ai-chat-wizard/proto/chat.pb.go, examples/server/ai-chat-wizard/proto/chat_grpc.pb.go, examples/server/ai-chat-wizard/server/app/catalog_contract.go, examples/server/ai-chat-wizard/server/app/catalog_contract_test.go, examples/server/ai-chat-wizard/TODO.md, examples/server/ai-chat-wizard/CHANGELOG.md
- validation run: protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto (run in examples/server/ai-chat-wizard); go test ./examples/server/ai-chat-wizard/proto -count=1; go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^TestBuildCatalogNamespacePayload"
- result: Passed. Added typed protobuf catalog contract messages and server-side deterministic builders that normalize locale/fallback/version/source metadata and compute stable content hashes from sorted message payloads.
- residual risk: Contract-only slice is complete, but no server loader or RPC transport is wired yet, so client strings still come from client-owned bundles until the next Agent 3 todos land.
- next suggested todo: Move the current client-owned strings into server-owned sources behind a loader interface that starts file- or Go-backed now and can later swap to DB/CMS storage without changing the client contract.

### 2026-05-30 — Info pages

- completed todos: Add real `/about`, `/contact`, `/privacy`, `/terms`, `/security`, `/status` pages with concrete business copy.
- files changed: `examples/server/ai-chat-wizard/client/app/routes.go` (6 route constants + `isLandingRoute` update), `examples/server/ai-chat-wizard/client/app/app.go` (6 route registrations in `ParseRun`), `examples/server/ai-chat-wizard/client/app/i18n.go` (info page keys in EN/ES/FR), `examples/server/ai-chat-wizard/client/app/landing_shell.go` (6 page constants, `parseLandingPageForPath` cases, `setLandingDocumentTitle` cases, `renderInfoShell` dispatch), `examples/server/ai-chat-wizard/client/app/landing_info.go` (new file — all 6 page renderers + shared `renderInfoSection` and `renderInfoStatusSection` helpers), `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `cmd /c "set GOOS=js&& set GOARCH=wasm&& go build ./examples/server/ai-chat-wizard/client/app/..."`
- result: Clean build. Footer links for About, Contact, Privacy, Terms, Security, and Status now render dedicated info pages inside the standard marketing chrome (header + footer) for unauthenticated visitors.
- residual risk: Authenticated users navigating directly to an info URL fall through to the workspace shell (acceptable — these pages are only linked from the marketing footer). Translations for ES and FR are functional but not professionally copyedited.
- next suggested todo: Tighten `/app/settings` content so each section title + summary + helper line describes one real user decision.

### 2026-03-28 08:30 America/New_York

- completed todo: Add an `Ops` workflow regression that covers incident or failed-jobs summary -> queue detail -> retry or replay action -> audit-feed confirmation -> return to the same queue and time-range context.
- files changed: `test/playwrightgo/examples/example100_admin_ops_workflow_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added an ops-workflow regression that verifies incident queue summary, queue-detail blast-radius inspection, replay-style incident mutation, audit-feed confirmation, and queue/time-range route-state restoration.
- residual risk: This checkpoint is compile-validated only; full runtime Playwright execution is still required to validate transition timing and audit propagation assumptions.
- next suggested todo: Continue with the next unchecked non-UI security/data-layer item (`Agent 2` / `Agent 3`) in strict top-to-bottom order.

### 2026-03-28 08:29 America/New_York

- completed todo: Add typed `Business` dashboard summary and drill-down RPCs backed by typed SQL queries over `billing_customers`, `billing_subscriptions`, `billing_invoices`, `billing_invoice_line_items`, `billing_access_overrides`, `billing_events`, `usage_events`, `product_analytics_events`, and `subscription_churn_feedback`.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/admin_business_ops.go`, `examples/server/ai-chat-wizard/server/app/admin_business_ops_test.go`, `examples/server/ai-chat-wizard/server/app/store_growth_ops.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/sql/store/list_product_analytics_events_by_user.sql`, `examples/server/ai-chat-wizard/sql/store/list_subscription_churn_feedback_by_customer.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestGetAdminBusinessDrilldownRPCs|TestGetAdminBusinessDrilldownScopeGuards|TestAdminBillingInterventionRPCs|TestStoreAdminDashboardQueries)$"`
- result: Passed. Added one typed business drill-down RPC (`GetAdminBusinessDrilldown`) that returns user-scoped billing customer/subscription/invoice/line-item/override/event rows plus usage, product analytics, and churn slices with admin scope enforcement and audit tracking.
- residual risk: This slice currently relies on the existing business summary surface (`GetAdminDashboard`) rather than a dedicated `Business`-only summary RPC; that split can still be carved out if the dashboard UI requires stricter endpoint isolation.
- next suggested todo: Add typed `Business` mutation RPCs, store funcs, and SQL queries for billing plans, plan entitlements, quota policies, overage rules, upgrade triggers, and dunning controls, including explicit audit-log writes for each mutation.

### 2026-03-28 08:28 America/New_York

- completed todo: Add a `Providers` workflow regression that covers provider health summary -> model drill-down -> fallback or visibility change -> blast-radius preview -> summary refresh with preserved time range and filters.
- files changed: `test/playwrightgo/examples/example100_admin_providers_workflow_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a providers-workflow regression that covers provider summary/drill-down reads, provider-policy mutation entry, blast-radius preview, summary refresh, and provider-filter/time-range route-state preservation.
- residual risk: This checkpoint is compile-validated only; runtime Playwright execution is still required to validate end-to-end browser timing and data assumptions.
- next suggested todo: Add an `Ops` workflow regression that covers incident or failed-jobs summary -> queue detail -> retry or replay action -> audit-feed confirmation -> return to the same queue and time-range context.

### 2026-03-28 08:27 America/New_York

- completed todo: Add a `Chats` workflow regression that covers failed-reply or slow-reply summary -> thread inspector -> message or run detail -> default-setting adjustment entry point -> return to the same anomaly queue state.
- files changed: `test/playwrightgo/examples/example100_admin_chats_workflow_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a chats-workflow regression that validates anomaly queue reads, thread/run drill-down fetches, settings-entry model-default adjustment, and queue-context route restoration across inspector/detail/settings transitions.
- residual risk: This checkpoint is compile-validated only; full runtime Playwright execution is still required to confirm browser-timing assertions and data-shape expectations.
- next suggested todo: Add a `Providers` workflow regression that covers provider health summary -> model drill-down -> fallback or visibility change -> blast-radius preview -> summary refresh with preserved time range and filters.

### 2026-03-28 08:25 America/New_York

- completed todo: Add a `Customers` workflow regression that covers search -> user or workspace detail -> unified account timeline -> disable or suspend action -> restore action -> return to the same filtered list state.
- files changed: `test/playwrightgo/examples/example100_admin_customers_workflow_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a customers-workflow regression that exercises filtered search, detail/timeline reads, disable/restore and suspend/restore mutations, and filtered list-context route restoration across detail navigation.
- residual risk: This checkpoint is compile-validated only; live Playwright runtime execution remains necessary to validate browser timing and transition assertions end-to-end.
- next suggested todo: Add a `Chats` workflow regression that covers failed-reply or slow-reply summary -> thread inspector -> message or run detail -> default-setting adjustment entry point -> return to the same anomaly queue state.

### 2026-03-28 08:23 America/New_York

- completed todo: Add a `Business` workflow regression that covers revenue summary -> failed-payments queue -> customer subscription detail -> resolve or override action -> return to the same filtered queue state.
- files changed: `test/playwrightgo/examples/example100_admin_business_workflow_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a business-workflow regression that runs summary->queue->detail->override/resolve actions using seeded billing+dunning fixtures and verifies pending-queue clearance plus filtered route-state restoration.
- residual risk: This checkpoint is compile-validated only; full browser/runtime execution should still be run to verify timing and data assertions end-to-end.
- next suggested todo: Add a `Customers` workflow regression that covers search -> user or workspace detail -> unified account timeline -> disable or suspend action -> restore action -> return to the same filtered list state.

### 2026-03-28 08:21 America/New_York

- completed todo: Add runtime diagnostics for dashboard summary fetches, table fetches, settings fetches, and drill-down fetches so each surface logs one actionable warning on empty, partial, slow, denied, or failed loads.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "^$" -count=1`
- result: Passed. Added shared runtime fetch diagnostics for summary/table/settings/drill-down RPCs with actionable warnings on empty/partial/slow outcomes, while preserving existing authorization-denied and query-failure logging paths.
- residual risk: This checkpoint is compile-only validation; targeted runtime log assertions should still be added to confirm warning emission under each degraded scenario.
- next suggested todo: Add a `Business` workflow regression that covers revenue summary -> failed-payments queue -> customer subscription detail -> resolve or override action -> return to the same filtered queue state.

### 2026-03-28 08:17 America/New_York

- completed todo: Add a browser-level dashboard smoke that verifies every chart or trend widget has a corresponding drill-down table or detail surface instead of a dead-end visualization.
- files changed: `test/playwrightgo/examples/example100_dashboard_drilldown_smoke_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a dashboard drill-down smoke that pairs dashboard trend/KPI payloads with concrete table/detail drill-down RPC checks and route-entry verification so slices are not dead-end visualizations.
- residual risk: This checkpoint is compile-validated only; full runtime Playwright execution is still required to confirm timing and data-shape behavior in-browser.
- next suggested todo: Add runtime diagnostics for dashboard summary fetches, table fetches, settings fetches, and drill-down fetches so each surface logs one actionable warning on empty, partial, slow, denied, or failed loads.

### 2026-03-28 08:16 America/New_York

- completed todo: Add focused regressions for admin detail drawers and detail routes so customer, chat, provider, and ops drill-down views preserve query state and return to the correct list context.
- files changed: `test/playwrightgo/examples/example100_admin_detail_routes_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a browser regression that resolves customer/chat/provider/ops drill-down targets, validates typed detail RPC loads, and confirms list-context query state survives deep-link transitions plus refresh/back/forward navigation.
- residual risk: This checkpoint uses compile validation only; live Playwright runtime execution is still required to confirm browser timing and route transition behavior end-to-end.
- next suggested todo: Add runtime diagnostics for dashboard summary fetches, table fetches, settings fetches, and drill-down fetches so each surface logs one actionable warning on empty, partial, slow, denied, or failed loads.

### 2026-03-28 08:13 America/New_York

- completed todo: Add focused regressions for dashboard settings mutations so plan or quota edits, provider toggles, site-config updates, and incident or feature-flag changes update the visible surface without stale cards or stale tables.
- files changed: `test/playwrightgo/examples/example100_admin_mutation_diagnostics_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Replaced the placeholder mutation diagnostics test with a real browser regression that applies billing plan/quota, feature-flag, and incident-status settings mutations, validates updated slice/control-plane reads, and verifies post-mutation route-state stability across refresh/back/forward navigation.
- residual risk: This checkpoint is compile-validated only; runtime Playwright execution is still needed to validate browser timing and full mutation lifecycle behavior.
- next suggested todo: Add focused regressions for admin detail drawers and detail routes so customer, chat, provider, and ops drill-down views preserve query state and return to the correct list context.

### 2026-03-28 08:09 America/New_York

- completed todo: Add one browser-level dashboard-home regression that covers the `Business`, `Customers`, `Chats`, `Providers`, and `Ops` surfaces, verifying tab or route entry, KPI-card load, table load, time-range changes, and back or refresh stability.
- files changed: `test/playwrightgo/examples/example100_dashboard_home_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. Added a dedicated dashboard-home regression that seeds deterministic business/ops rows, validates KPI + drill-down RPC loads per surface, and asserts route/deep-link entry plus shared `lookback_days` time-range and refresh/back/forward URL-state stability.
- residual risk: This checkpoint uses compile validation only; a full browser runtime pass is still needed to confirm timing stability under live Playwright execution.
- next suggested todo: Add focused regressions for dashboard settings mutations so plan or quota edits, provider toggles, site-config updates, and incident or feature-flag changes update the visible surface without stale cards or stale tables.

### 2026-03-28 08:06 America/New_York

- completed todo: Add focused regressions for dashboard list mechanics so shared time range, search, filter, sort, pagination, and drill-down state remain stable across surface switches and direct deep links.
- files changed: `test/playwrightgo/examples/example100_admin_list_mechanics_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. The admin list-mechanics browser regression now carries one shared `lookback_days` time-range parameter alongside search/filter/sort/pagination query state across users/workspaces/support/incidents/billing deep links, and asserts refresh/back/forward URL-state stability.
- residual risk: This is compile validation only; a live Playwright runtime pass is still needed to confirm timing behavior in CI/browser environments.
- next suggested todo: Add one browser-level dashboard-home regression that covers the `Business`, `Customers`, `Chats`, `Providers`, and `Ops` surfaces, verifying tab or route entry, KPI-card load, table load, time-range changes, and back or refresh stability.

### 2026-03-28 08:04 America/New_York

- completed todo: Add a dashboard operator runbook that explains how to review revenue, customer health, chat health, provider health, and site health without drifting into vanity metrics or redundant surfaces.
- files changed: `examples/server/ai-chat-wizard/OPERATOR_RUNBOOK.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Added a structured dashboard review playbook with anti-vanity checks for Business, Customers, Chats, Providers, and Ops so operators review decisions with actionable context.
- residual risk: Runbook guidance is planned-mode documentation and still depends on final dashboard UI and drill-down surfaces landing as described.
- next suggested todo: Build a dashboard home UI that summarizes the platform state and clearly branches into workspace-admin vs superuser slices.

### 2026-03-28 08:02 America/New_York

- completed todo: Document the dashboard permissions model so maintainers know which surfaces, lists, detail views, and settings forms belong to normal users, workspace admins, and superusers.
- files changed: `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Added a role-scope matrix for normal users, workspace admins, and superusers that defines dashboard surface visibility, list/detail scope, and mutation boundaries.
- residual risk: This is documentation guidance and still requires strict server authz and UI route guards to enforce every boundary at runtime.
- next suggested todo: Add a dashboard operator runbook that explains how to review revenue, customer health, chat health, provider health, and site health without drifting into vanity metrics or redundant surfaces.

### 2026-03-28 08:02 America/New_York

- completed todo: Add one dashboard endpoint and query map that lists the planned summary RPCs, drill-down RPCs, mutation RPCs, store funcs, SQL queries, and underlying tables for each dashboard surface.
- files changed: `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Added a per-surface endpoint/query matrix that maps planned summary/drill-down/mutation RPCs to store/SQL families and underlying table ownership.
- residual risk: RPC and query names are planning-level targets and should be kept synchronized as concrete proto and SQL contracts are implemented.
- next suggested todo: Document the dashboard permissions model so maintainers know which surfaces, lists, detail views, and settings forms belong to normal users, workspace admins, and superusers.

### 2026-03-28 08:01 America/New_York

- completed todo: Add one dashboard metric-definition section that defines each core KPI, its source-of-truth table set, its source RPC, and any caveats about lag, rollups, or derived values.
- files changed: `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Added a dashboard KPI definition matrix with per-metric table ownership, summary RPC mapping, and caveats to reduce ambiguity during dashboard implementation.
- residual risk: KPI definitions are still planned contracts; final field names and RPC envelopes must stay synchronized as typed dashboard surfaces land.
- next suggested todo: Add one dashboard endpoint and query map that lists the planned summary RPCs, drill-down RPCs, mutation RPCs, store funcs, SQL queries, and underlying tables for each dashboard surface.

### 2026-03-28 08:00 America/New_York

- completed todo: Document the five core dashboard surfaces in product terms so maintainers know exactly what belongs in `Business`, `Customers`, `Chats`, `Providers`, and `Ops`, and what intentionally does not belong there.
- files changed: `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Added explicit per-surface scope and non-scope boundaries to keep dashboard implementation aligned with one coherent information architecture.
- residual risk: This is documentation-level guidance and still needs enforcement through route composition, RPC boundaries, and dashboard UI tests.
- next suggested todo: Add one dashboard metric-definition section that defines each core KPI, its source-of-truth table set, its source RPC, and any caveats about lag, rollups, or derived values.

### 2026-03-28 07:58 America/New_York

- completed todo: Design a clear admin entry point from the authenticated app shell so workspace admins and superusers can discover dashboard access without cluttering the normal user flow.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Confirmed role-gated admin entry controls are surfaced in both mobile and desktop control bars (`client/app/panel.go`) and route through the authenticated shell admin-open handler (`client/app/app.go`), while non-admin users do not see the entry.
- residual risk: This checkpoint confirms discoverability and gating at shell level; deeper dashboard slice UX remains tracked under subsequent dashboard todos.
- next suggested todo: Build a dashboard home UI that summarizes the platform state and clearly branches into workspace-admin vs superuser slices.

### 2026-03-28 07:56 America/New_York

- completed todo: Lazy load chat threads from gRPC as the user scrolls through the sidebar list.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_more_unit_test.go`, `examples/server/ai-chat-wizard/client/app/conversations.go`, `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/sidebar.go`, `examples/server/ai-chat-wizard/client/app/app_shell.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "^(TestListConversationsPagination|TestListAndLoadConversationBranches)$" -count=1`; `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/client/app -run "^TestHasScrollSpaceBelow$" -count=1`; `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_app_wasm.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. `ListConversations` now supports paging (`page_size`, `page_offset`) with `has_more`/`next_offset`, and the client fetches additional conversation pages over gRPC as sidebar scroll reaches the bottom.
- residual risk: This checkpoint is server-tested and client compile-validated; a live browser smoke with a large seeded thread set is still needed to fine-tune page size and scroll-trigger behavior.
- next suggested todo: Design a clear admin entry point from the authenticated app shell so workspace admins and superusers can discover dashboard access without cluttering the normal user flow.

### 2026-03-28 07:54 America/New_York

- completed todo: Add UI treatment for auth failures, entitlement blocks, upgrade prompts, and post-first-reply success cues so users always know the next action in the journey.
- files changed: `examples/server/ai-chat-wizard/client/app/composer.go`, `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Composer now surfaces explicit journey cue banners for auth/session failures, entitlement or quota/upgrade-style blocks, and first-reply completion with clear next-action guidance.
- residual risk: Cue detection currently keys off assistant error text patterns and may need follow-up normalization once server-side typed error codes are fully plumbed into the UI.
- next suggested todo: Design a clear admin entry point from the authenticated app shell so workspace admins and superusers can discover dashboard access without cluttering the normal user flow.

### 2026-03-28 07:52 America/New_York

- completed todo: Build a first-run empty-state and onboarding layer with starter prompts/templates, obvious first actions, and a clear transition from zero threads to the first real thread.
- files changed: `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/client/app/stream.go`, `examples/server/ai-chat-wizard/client/app/app_shell.go`, `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/thread.go`, `examples/server/ai-chat-wizard/client/app/empty_state.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Empty state now includes starter onboarding prompts that users can click to prefill the composer and jump directly into the first-send flow without manual prompt drafting.
- residual risk: This is compile-validated and interaction-wired, but browser smoke is still needed to confirm mobile spacing and long-prompt wrapping in the starter cards.
- next suggested todo: Add UI treatment for auth failures, entitlement blocks, upgrade prompts, and post-first-reply success cues so users always know the next action in the journey.

### 2026-03-28 07:50 America/New_York

- completed todo: Adjust chat cards, bubbles, inputs, panels, and buttons so their corner radii, outlines, and surface shapes better match the home-page design language.
- files changed: `examples/server/ai-chat-wizard/client/app/app_shell.go`, `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/thread.go`, `examples/server/ai-chat-wizard/client/app/composer.go`, `examples/server/ai-chat-wizard/client/app/bubble.go`, `examples/server/ai-chat-wizard/client/app/styles.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -count=1`
- result: Passed. Updated workspace shape language to match the landing direction with consistent rounded geometry and outline weight across toolbar shells, thread surface, composer shell, and assistant/user message bubbles.
- residual risk: This checkpoint is compile-validated; final visual balance should still be confirmed in browser smoke on desktop and mobile breakpoints.
- next suggested todo: Build a first-run empty-state and onboarding layer with starter prompts/templates, obvious first actions, and a clear transition from zero threads to the first real thread.

### 2026-03-28 07:49 America/New_York

- completed todo: Slightly retune the in-app chat palette to align with the landing-page direction while preserving readability, hierarchy, and streaming-state clarity.
- files changed: `examples/server/ai-chat-wizard/client/app/bubble.go`, `examples/server/ai-chat-wizard/client/app/styles.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_palette_tune_wasm.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. Chat palette accents now align to the landing cyan direction across thought-streaming surfaces, scrollbars, and toolbar focus treatments while keeping assistant/user role contrast and streaming readability clear.
- residual risk: This pass is compile-validated; final perception of contrast and motion under different displays and browser rendering modes still needs browser smoke confirmation.
- next suggested todo: Adjust chat cards, bubbles, inputs, panels, and buttons so their corner radii, outlines, and surface shapes better match the home-page design language.

### 2026-03-28 07:48 America/New_York

- completed todo: Make the chat-thread list independently scrollable with stable scroll behavior.
- files changed: `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/conversations.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run "^TestHasScrollSpaceBelow$" -count=1`; `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_app_wasm.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. Conversation-list refresh now preserves sidebar scroll context by restoring either the prior offset or a bottom-pinned state after list rerenders, preventing abrupt jumps during periodic sidebar updates.
- residual risk: This behavior is unit/compile validated but not yet confirmed in a live browser flow with long conversation lists and rapid refresh churn.
- next suggested todo: Lazy load chat threads from gRPC as the user scrolls through the sidebar list.

### 2026-03-28 07:47 America/New_York

- completed todo: Refresh the chat surface styling so its colors, border treatments, and shapes feel like the same product as the home page instead of a separate UI.
- files changed: `examples/server/ai-chat-wizard/client/app/app_shell.go`, `examples/server/ai-chat-wizard/client/app/sidebar.go`, `examples/server/ai-chat-wizard/client/app/composer.go`, `examples/server/ai-chat-wizard/client/app/bubble.go`, `examples/server/ai-chat-wizard/client/app/avatar.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_surface_refresh_wasm.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The authenticated chat shell now uses the same visual system as marketing surfaces: shared dark base tones, cyan-accented controls, consistent border contrast, and aligned rounded-card treatments for sidebar rows, composer, message bubbles, and avatar badges.
- residual risk: This checkpoint is compile-validated but not yet browser-smoked for final visual balance across viewport sizes and high-contrast display environments.
- next suggested todo: Slightly retune the in-app chat palette to align with the landing-page direction while preserving readability, hierarchy, and streaming-state clarity.

### 2026-03-28 07:46 America/New_York

- completed todo: Enlarge and recenter the scroll-to-bottom button so it sits clearly centered above the input fields.
- files changed: `examples/server/ai-chat-wizard/client/app/thread.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_app_wasm.test ./examples/server/ai-chat-wizard/client/app`; `go test -c -tags playwrightgo -o ./bin/example100_playwright_examples.test ./test/playwrightgo/examples`
- result: Passed. The scroll-to-bottom control now has a larger 64px footprint, stronger contrast/shadow, and a slightly higher resting offset while preserving centered positioning above the input lane.
- residual risk: A runtime Playwright execution of `TestExample100ScrollToBottomButton` currently times out waiting for button visibility in this workspace, so this checkpoint is compile-validated and should be visually confirmed in the next interactive smoke pass.
- next suggested todo: Make the chat-thread list independently scrollable with stable scroll behavior.

### 2026-03-28 07:45 America/New_York

- completed todo: Refine the public-to-first-chat UX so landing, pricing, signup, login, empty state, first composer state, streaming state, and post-first-reply state read like one coherent journey.
- files changed: `examples/server/ai-chat-wizard/client/app/journey_state.go`, `examples/server/ai-chat-wizard/client/app/journey_state_wasm_test.go`, `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/composer.go`, `examples/server/ai-chat-wizard/client/app/thread.go`, `examples/server/ai-chat-wizard/client/app/empty_state.go`, `examples/server/ai-chat-wizard/client/app/landing_hero.go`, `examples/server/ai-chat-wizard/client/app/pricing_shell.go`, `examples/server/ai-chat-wizard/client/app/auth_shell.go`, `examples/server/ai-chat-wizard/client/app/signup_shell.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_journey_wasm.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The app now presents one consistent journey band across public/auth/workspace shells and derives first-chat guidance states for empty, first-send pending, first-reply streaming, and post-first-reply continuation with a focused wasm unit test for stage derivation.
- residual risk: This checkpoint is compile-validated for js/wasm and unit-covered for stage derivation, but executable wasm test runs remain environment-dependent on this Windows host (`%1 is not a valid Win32 application` without a js/wasm runner).
- next suggested todo: Refresh the chat surface styling so its colors, border treatments, and shapes feel like the same product as the home page instead of a separate UI.

### 2026-03-28 07:41 America/New_York

- completed todo: Fix the top-left branding badge so the `GWC` text stays centered inside the circle, scaling the circle up if needed.
- files changed: `examples/server/ai-chat-wizard/client/app/avatar.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/example100_client_app_wasm.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. Assistant bubble branding badge now uses a 36px circular container with centered flex alignment and tighter glyph metrics so `GWC` remains visually centered.
- residual risk: This change is style-only and is compile-validated; visual spacing still depends on browser font rendering and should be confirmed in the next full browser smoke pass.
- next suggested todo: Enlarge and recenter the scroll-to-bottom button so it sits clearly centered above the input fields.

### 2026-03-28 02:37 America/New_York

- completed todo: Add focused regressions for admin list mechanics so user, workspace, support, incident, and billing views keep search, filter, sort, and pagination state stable across refresh and back navigation.
- files changed: `test/playwrightgo/examples/example100_admin_list_mechanics_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_admin_list_mechanics.test ./test/playwrightgo/examples`; `go test ./test/playwrightgo/examples -tags playwrightgo -run "^TestExample100AdminListMechanicsRegression$" -count=1`
- result: Passed. Added a live Playwright list-mechanics regression that seeds deterministic list fixtures, validates typed `AdminListQuery` mechanics for user/workspace/support/incident/billing slices (search/filter/sort/pagination), and enforces list-state URL stability across refresh/back/forward navigation.
- residual risk: This flow validates list mechanics through typed RPC contracts plus URL-state persistence checks; slice-specific list UI widgets and controls remain part of broader dashboard UX implementation work.
- next suggested todo: Refine the public-to-first-chat UX so landing, pricing, signup, login, empty state, first composer state, streaming state, and post-first-reply state read like one coherent journey.

### 2026-03-28 02:35 America/New_York

- completed todo: Add focused regressions for billing-intervention, support-triage, and incident-control flows so admin actions survive refresh and back navigation correctly.
- files changed: `test/playwrightgo/examples/example100_admin_ops_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_admin_ops_and_mutation.test ./test/playwrightgo/examples`; `go test ./test/playwrightgo/examples -tags playwrightgo -run "^TestExample100AdminOpsRegression$" -count=1`
- result: Passed. Added a live Playwright admin-ops regression that seeds deterministic billing/support fixtures, executes billing-intervention, support-triage, and incident-control mutations over the authenticated gRPC tunnel, verifies typed readback results, and enforces refresh/back/forward route stability after each operator flow.
- residual risk: This regression validates operator workflows through RPC-driven actions and route stability checks; dedicated slice-specific UI controls for billing/support/incident surfaces remain an independent UX implementation track.
- next suggested todo: Add focused regressions for admin list mechanics so user, workspace, support, incident, and billing views keep search, filter, sort, and pagination state stable across refresh and back navigation.

### 2026-03-28 02:33 America/New_York

- completed todo: Add one browser-level admin-mutation regression that covers disable user, restore user, suspend workspace, restore workspace, and the resulting UI state transitions.
- files changed: `test/playwrightgo/examples/example100_admin_mutation_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -c -tags playwrightgo -o ./bin/example100_admin_mutation.test ./test/playwrightgo/examples`; `go test ./test/playwrightgo/examples -tags playwrightgo -run "^TestExample100AdminMutationRegression$" -count=1`
- result: Passed. Replaced the skipped pending scaffold with a live Playwright regression that logs in as superuser, executes `DisableAdminUser`/`RestoreAdminUser` and `SuspendAdminWorkspace`/`RestoreAdminWorkspace` over the authenticated gRPC tunnel, and verifies browser transitions between authenticated app shell and auth/public entry states at each mutation boundary.
- residual risk: This regression currently validates mutation transitions through auth/public shell behavior and RPC mutation outcomes; dedicated UI control flows for billing/support/incident operator surfaces remain tracked separately.
- next suggested todo: Add focused regressions for billing-intervention, support-triage, and incident-control flows so admin actions survive refresh and back navigation correctly.

### 2026-03-28 02:30 America/New_York

- completed todo: Add a restricted read-only admin query surface for debugging and reporting instead of any raw SQL passthrough RPC; add runtime diagnostics for admin mutations so confirmation, submit, success, denial, and rollback states emit actionable logs during operator workflows.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/admin_readonly_report_ops.go`, `examples/server/ai-chat-wizard/server/app/admin_readonly_report_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects|AdminMutationDiagnosticsLogs|GetAdminReadOnlyReportAllowlistedSlices|GetAdminReadOnlyReportScopeAndGuards)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "Benchmark(ParseNormalizeAdminReadOnlyReportKey|ParseBuildAdminMutationScopeLabel)$" -benchtime=100x`; `go test ./examples/server/ai-chat-wizard/proto -count=1`
- result: Passed. Added `GetAdminReadOnlyReport` as an allowlisted typed read-only reporting surface (`dashboard_summary`, `recent_users`, `recent_usage_events`, `recent_conversations`) with admin scope enforcement and audit tracking, and added mutation workflow diagnostics covering submit, confirmation, denial, success, and rollback-required failure states with focused assertions.
- residual risk: Browser-level operator mutation regressions are still pending where UI controls are not yet wired, so these diagnostics are currently validated at server-RPC level.
- next suggested todo: Add one browser-level admin-mutation regression that covers disable user, restore user, suspend workspace, restore workspace, and the resulting UI state transitions.

### 2026-03-28 02:19 America/New_York

- completed todo: Add typed `su` CRUD RPCs for reliability and trust tables: SSO configs, retention policies, compliance controls, SLOs, incidents, and incident updates.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser_reliability.go`, `examples/server/ai-chat-wizard/server/app/superuser_reliability_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_reliability_ops_test.go`, `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, reliability/trust SQL files under `examples/server/ai-chat-wizard/sql/store/` for SSO, retention, compliance, SLO, incident, and incident-update get/list/upsert/delete paths, plus `examples/server/ai-chat-wizard/TODO.md` and `examples/server/ai-chat-wizard/CHANGELOG.md`.
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserReliabilityControlFuncs|SuperuserReliabilityControlRPCs|StoreSuperuserPricingControlFuncs|SuperuserPricingControlRPCs|SuperuserPricingControlRPCsRequireSURole)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserSlicesReturnsSnapshot|StoreSuperuserControlPlaneLifecycle|StoreSuperuserReliabilityControlFuncs|SuperuserReliabilityControlRPCs)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "Benchmark(ParseNormalizeSuperuserIncidentStatus|ParseNormalizeSuperuserBillingEnforcementMode)$" -benchtime=100x`
- result: Passed. Superuser reliability/trust control paths now expose typed CRUD RPCs for workspace SSO configs, data-retention policies, compliance controls, service-level objectives, incidents, and incident updates, backed by typed store/query helpers and audit-emitting mutation handlers.
- residual risk: Several set-mutation responses currently report generic `status=updated` semantics and do not distinguish create-vs-update for key-based upserts.
- next suggested todo: Add a restricted read-only admin query surface for debugging and reporting instead of any raw SQL passthrough RPC.

### 2026-03-28 02:03 America/New_York

- completed todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser_pricing.go`, `examples/server/ai-chat-wizard/server/app/superuser_pricing_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_pricing_ops_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_billing_plan_overage.sql`, `examples/server/ai-chat-wizard/sql/store/delete_billing_plan_overage.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_billing_quota_policy.sql`, `examples/server/ai-chat-wizard/sql/store/delete_billing_quota_policy.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_billing_upgrade_trigger.sql`, `examples/server/ai-chat-wizard/sql/store/delete_billing_upgrade_trigger.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_billing_dunning_event.sql`, `examples/server/ai-chat-wizard/sql/store/get_billing_dunning_event_by_id.sql`, `examples/server/ai-chat-wizard/sql/store/delete_billing_dunning_event.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(SuperuserPricingControlRPCs|SuperuserPricingControlRPCsRequireSURole|GetSuperuserSlicesReturnsSnapshot|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot)$" -count=1`
- result: Passed. Superuser pricing controls now expose typed CRUD RPCs for plan overages, quota policies, upgrade triggers, and dunning events with superuser mutation gating, confirm+reason enforcement, and mutation audit-event logging.
- residual risk: Set-mutation responses for overage/quota/upgrade currently return `status=updated` and do not distinguish create-vs-update operations.
- next suggested todo: Add typed `su` CRUD RPCs for reliability and trust tables: SSO configs, retention policies, compliance controls, SLOs, incidents, and incident updates.

### 2026-03-28 01:53 America/New_York

- completed todo: Add typed search, filter, sort, and pagination support for admin user, workspace, support, incident, and billing list RPCs so large datasets are operable without loading everything at once.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_billing_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_billing_ops_test.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminUserListQueryRPCs|AdminSupportTriageListQueryRPCs|AdminBillingListQueryRPCs|GetSuperuserSlicesAppliesListQueries|GetSuperuserSlicesAppliesTypedListQueries|AdminBillingInterventionRPCs|AdminSupportTriageRPCs|AdminUserControlRPCs)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "Benchmark(ParseFilterAdminBillingEventRows|ParseFilterSuperuserWorkspaceRows)$" -benchtime=100x`
- result: Passed. Admin list-query mechanics now cover user, support, billing, workspace, and incident surfaces with typed search/filter/sort/offset behavior and explicit RPC regression coverage.
- residual risk: List-query filtering and sorting still run in memory over bounded scan windows instead of SQL pushdown for very large datasets.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-28 01:49 America/New_York

- completed todo: Add typed search, filter, sort, and pagination support for admin user, workspace, support, incident, and billing list RPCs so large datasets are operable without loading everything at once.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/admin_list_query.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops.go`, `examples/server/ai-chat-wizard/server/app/admin_billing_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/admin_list_query_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminListQueryUsersUsageConversations|AdminListQuerySupportBilling|SuperuserSlicesListQuery|StoreAdminSupportTriageFuncs|AdminSupportTriageRPCs|AdminSupportTriageRPCsWorkspaceAdminScope|StoreAdminControlMutationFuncs|AdminControlOpsRPCs|AdminControlOpsWorkspaceScope|StoreAdminBillingInterventionFuncs|AdminBillingInterventionRPCs|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope|GetSuperuserSlicesReturnsSnapshot)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "^$" -count=1`
- result: Passed. Admin list surfaces now use a shared typed query shape (`limit`, `offset`, `search`, `sort_by`, `sort_direction`) with domain filters for usage, support, billing, and superuser workspace/support/incident slices, and list-windowing now occurs after typed filtering/sorting.
- residual risk: Sorting/filtering currently runs in-memory over bounded scan windows (`parseAdminScopedScanLimit`) rather than SQL-level pushdown.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-28 01:37 America/New_York

- completed todo: Add typed incident and experiment control funcs and RPCs for incident updates, status changes, feature-flag toggles, experiment rollbacks, and blast-radius reporting.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/sql/store/update_incident_status.sql`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_admin.go`, `examples/server/ai-chat-wizard/server/app/admin_control_ops.go`, `examples/server/ai-chat-wizard/server/app/admin_control_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminControlMutationFuncs|AdminControlOpsRPCs|AdminControlOpsWorkspaceScope|AdminControlMutationScopeBoundaries|StoreAdminSupportTriageFuncs|AdminSupportTriageRPCs|AdminSupportTriageRPCsWorkspaceAdminScope)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "Benchmark(ParseNormalizeAdminBillingQuotaOverrideKey|ParseBuildAdminSupportTicketEntry|ParseHasAdminIncidentResolvedStatus)$" -benchtime=100x`
- result: Passed. Typed control-plane mutation/reporting seams now cover feature-flag toggles, experiment rollbacks, incident status/update writes, and scoped incident blast-radius reporting, with superuser/workspace-admin boundaries enforced fail-closed.
- residual risk: Incident scope is currently enforced from request workspace scope because incidents do not yet include a direct workspace foreign-key constraint in this schema.
- next suggested todo: Add typed search, filter, sort, and pagination support for admin user, workspace, support, incident, and billing list RPCs so large datasets are operable without loading everything at once.

### 2026-03-28 01:35 America/New_York

- completed todo: Add typed support-triage funcs and RPCs for ticket queues, ticket detail, internal notes, assignment, escalation, and account-linked action history.
- files changed: `examples/server/ai-chat-wizard/server/app/store_admin.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminSupportTriageFuncs|AdminSupportTriageRPCs|AdminSupportTriageRPCsWorkspaceAdminScope)$" -count=1`
- result: Passed. Support triage now has typed scoped queue and detail RPCs plus typed internal-note, assignment, and escalation mutations with account-linked action history by user account and explicit workspace-admin scope denials.
- residual risk: Queue filtering is currently typed by `status`, `priority`, and `assignee_user_id`; server-side sort/pagination semantics for large datasets remain under the dedicated list-mechanics todo.
- next suggested todo: Add typed incident and experiment control funcs and RPCs for incident updates, status changes, feature-flag toggles, experiment rollbacks, and blast-radius reporting.

### 2026-03-28 01:30 America/New_York

- completed todo: Add typed support-triage funcs and RPCs for ticket queues, ticket detail, internal notes, assignment, escalation, and account-linked action history.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/store_admin.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops.go`, `examples/server/ai-chat-wizard/server/app/admin_support_ops_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminSupportTriageFuncs|AdminSupportTriageRPCs|StoreAdminBillingInterventionFuncs|AdminBillingInterventionRPCs|AdminUserControlRPCs|AdminUserControlRPCsWorkspaceAdminScope|GetWorkspaceAdminSlices|BillingInterventionOpsSuperuserFlow|BillingInterventionOpsWorkspaceScopeDenied)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "Benchmark(ParseNormalizeAdminBillingQuotaOverrideKey|ParseBuildAdminSupportTicketEntry)$" -benchtime=100x`
- result: Passed. Support triage now has typed scoped queue/detail RPCs plus typed internal-note, assignment, and escalation mutation RPCs, backed by typed admin store helpers and account-linked action history slices.
- residual risk: Support-ticket queue filtering currently supports status/priority/assignee only; richer server-side pagination and sort semantics remain open under the dedicated list-mechanics todo.
- next suggested todo: Add typed incident and experiment control funcs and RPCs for incident updates, status changes, feature-flag toggles, experiment rollbacks, and blast-radius reporting.

### 2026-03-28 02:36 America/New_York

- completed todo: Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.
- files changed: `examples/server/ai-chat-wizard/server/app/billing_intervention_ops.go`, `examples/server/ai-chat-wizard/server/app/billing_intervention_ops_test.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/server/app/billing_admin_rpc_test.go` (removed), `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:GOCACHE=(Resolve-Path '.\\bin').Path + '\\gocache-temp'; go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminBillingInterventionFuncs|AdminBillingInterventionRPCs|BillingInterventionOpsSuperuserFlow|BillingInterventionOpsWorkspaceScopeDenied|WorkspaceAdminOpsDetailAndMutations|WorkspaceAdminOpsDenyOutOfScope|AdminMutationRestoreUserPreservesNonDisableAuthBlocks|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|UserScopedWritesBlockedWhenAuthBlocked)$" -count=1`
- result: Passed. Billing-intervention flows now have typed backend helper seams plus validated admin RPC behavior (existing `admin_billing_ops` surface) for overrides, quota controls, dunning review, and failed-payment resolution; workspace-admin helper seams are also covered.
- residual risk: Global `go clean -cache` is currently blocked by locked files in `%LOCALAPPDATA%\\go-build`; tests were validated using isolated `GOCACHE` under `bin\\gocache-temp`.
- next suggested todo: Add typed support-triage funcs and RPCs for ticket queues, ticket detail, internal notes, assignment, escalation, and account-linked action history.
### 2026-03-28 02:23 America/New_York

- completed todo: Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.
- files changed: `examples/server/ai-chat-wizard/server/app/billing_intervention_ops.go`, `examples/server/ai-chat-wizard/server/app/billing_intervention_ops_test.go`, `examples/server/ai-chat-wizard/server/app/billing_admin_rpc_impl.go`, `examples/server/ai-chat-wizard/server/app/billing_admin_rpc_test.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminBillingRPCsSuperuserFlow|AdminBillingRPCsWorkspaceScopeDenied|BillingInterventionOpsSuperuserFlow|BillingInterventionOpsWorkspaceScopeDenied|WorkspaceAdminOpsDetailAndMutations|WorkspaceAdminOpsDenyOutOfScope|AdminMutationRestoreUserPreservesNonDisableAuthBlocks|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|UserScopedWritesBlockedWhenAuthBlocked)$" -count=1`
- result: Passed. Typed billing-intervention backend seams are now exposed through concrete admin RPC implementations for override listing/mutation, billing-event inspection, dunning review, quota override mutation, and failed-payment resolution, with scoped role enforcement and focused regression coverage.
- residual risk: Dunning event shape is currently derived from billing-event rows, so attempt/next-attempt metadata remains limited until a dedicated dunning lifecycle table is wired.
- next suggested todo: Add typed support-triage funcs and RPCs for ticket queues, ticket detail, internal notes, assignment, escalation, and account-linked action history.
### 2026-03-28 02:05 America/New_York

- completed todo: Start typed billing-intervention backend seams for overrides, failed-payment resolution, dunning review, and billing-event inspection.
- files changed: `examples/server/ai-chat-wizard/server/app/billing_intervention_ops.go`, `examples/server/ai-chat-wizard/server/app/billing_intervention_ops_test.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(BillingInterventionOpsSuperuserFlow|BillingInterventionOpsWorkspaceScopeDenied|WorkspaceAdminOpsDetailAndMutations|WorkspaceAdminOpsDenyOutOfScope|AdminMutationRestoreUserPreservesNonDisableAuthBlocks|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|UserScopedWritesBlockedWhenAuthBlocked)$" -count=1`
- result: Passed. Added scoped typed billing-intervention helper flows and focused coverage, plus dedicated workspace-admin detail/review/mutation helper coverage for in-scope and out-of-scope behavior.
- residual risk: Billing-intervention flows are currently backend helper seams and still need typed public RPC exposure to fully satisfy the billing-intervention TODO.
- next suggested todo: Expose billing-intervention helper flows through typed admin RPCs (or equivalent typed external API surface).
### 2026-03-28 01:46 America/New_York

- completed todo: Add typed workspace-admin query and mutation funcs for workspace detail, suspend workspace, restore workspace, revoke API keys, pause webhooks, and membership review.
- files changed: `examples/server/ai-chat-wizard/server/app/workspace_admin_ops.go`, `examples/server/ai-chat-wizard/server/app/workspace_admin_ops_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(WorkspaceAdminOpsDetailAndMutations|WorkspaceAdminOpsDenyOutOfScope|AdminMutationRestoreUserPreservesNonDisableAuthBlocks|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|UserScopedWritesBlockedWhenAuthBlocked)$" -count=1`
- result: Passed. Added typed workspace-admin helper surfaces for scoped workspace detail/review plus suspend/restore, API-key revoke, and webhook pause mutations with fail-closed scope checks, and added focused regression coverage for in-scope and out-of-scope behavior.
- residual risk: These helpers are backend seams and are not yet exposed as dedicated public gRPC workspace-admin mutation RPCs.
- next suggested todo: Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.
### 2026-03-28 01:01 America/New_York

- completed todo: Add explicit backend side-effect handlers for disable/suspend plus typed restore handlers with explicit capability-selection flags.
- files changed: `examples/server/ai-chat-wizard/server/app/store.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_authz.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/server/app/notification_flow.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|AdminMutationRestoreUserCanExplicitlyRestoreAPIKeys|AdminMutationRestoreWorkspaceCanExplicitlyRestoreOperationalCapabilities)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(UserScopedWritesBlockedWhenDisabled|WorkspaceScopedWritesBlockedWhenSuspended|HandleBackgroundJobsFailsSuspendedWorkspaceQueue)$" -count=1`
- result: Passed. Disable/suspend flows now apply explicit operational side effects (API key revoke, webhook disable, workspace-job suppression), and restore flows now accept typed explicit capability flags so operators can choose when to re-enable keys/webhooks/jobs instead of relying on silent defaults.
- residual risk: Explicit API-key restore currently restores all revoked keys in scope because revoke provenance (manual vs suspension-induced) is not yet tracked separately.
- next suggested todo: Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.

### 2026-03-28 01:00 America/New_York

- completed todo: Add typed workspace-admin query and mutation funcs for workspace detail, suspend workspace, restore workspace, revoke API keys, pause webhooks, and membership review.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetWorkspaceAdminSlices|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|AdminMutationRestoreWorkspaceCanExplicitlyRestoreOperationalCapabilities)$" -count=1`
- result: Passed. Existing typed workspace-admin slice and mutation-effect seams now cover workspace detail/membership review plus suspend/restore flows with dependent operational controls for API keys, webhook endpoints, and queued jobs.
- residual risk: Workspace suspend/restore is currently exercised through typed server seams and targeted tests; dedicated public workspace-mutation RPC endpoints remain a follow-up if UI/API clients need first-class workspace mutation contracts.
- next suggested todo: Add typed billing-intervention funcs and RPCs for quota overrides, access overrides, failed-payment resolution, dunning review, and billing-event inspection.

### 2026-03-28 00:58 America/New_York

- completed todo: Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.
- files changed: `examples/server/ai-chat-wizard/server/app/store_admin.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminUserControlQueries|AdminUserControlRPCs|AdminUserControlRPCsWorkspaceAdminScope)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminDashboardQueries|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope|GetWorkspaceAdminSlices|StoreAdminUserControlQueries|AdminUserControlRPCs|AdminUserControlRPCsWorkspaceAdminScope|AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects)$" -count=1`
- result: Passed. Added typed admin-user store helpers and wired the `SearchAdminUsers`, `GetAdminUserDetail`, `DisableAdminUser`, and `RestoreAdminUser` RPCs with scope enforcement, typed detail slices (sessions/usage/audit), and mutation responses that reflect persisted access status.
- residual risk: User-detail typed helpers currently apply bounded in-memory filtering over existing list queries rather than dedicated SQL predicates for each user-detail slice.
- next suggested todo: Add typed workspace-admin query and mutation funcs for workspace detail, suspend workspace, restore workspace, revoke API keys, pause webhooks, and membership review.

### 2026-03-28 00:57 America/New_York

- completed todo: Add explicit backend side-effect handlers for disable and suspend flows so session revocation, API-key pausing, webhook pausing, and job suppression happen transactionally and are auditable.
- files changed: `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(UserScopedWritesBlockedWhenDisabled|WorkspaceScopedWritesBlockedWhenSuspended|HandleBackgroundJobsFailsSuspendedWorkspaceQueue)$" -count=1`
- result: Passed. Admin mutation effect handlers now include explicit operational suppression: user-disable revokes user-scoped API keys, and workspace-suspend transactionally revokes workspace API keys, disables webhook endpoints, and suppresses pending/running workspace-queued background jobs while existing session/token invalidation remains intact.
- residual risk: Restore is still intentionally coarse-grained and does not yet offer operator-selectable partial re-enable paths for revoked keys/webhooks/jobs.
- next suggested todo: Add typed restore handlers that let operators choose which dependent capabilities come back automatically versus which stay manually revoked.

### 2026-03-28 00:56 America/New_York

- completed todo: Define and enforce the exact runtime effects of user disable vs workspace suspend so chat send, dashboard access, API keys, webhooks, background jobs, and support actions all fail in a consistent way.
- files changed: `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/job_flows.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/server/app/job_flows_test.go`, `examples/server/ai-chat-wizard/server/app/server_helper_branches_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(UserScopedWritesBlockedWhenDisabled|WebhookAndSupportMessageBlockedWhenWorkspaceSuspended|HandleBackgroundJobsFailsBlockedPayloadScope|HandleBackgroundJobsFailsSuspendedWorkspaceQueue|WorkspaceScopedWritesBlockedWhenSuspended|SendDeniedForDisabledUser|SendDeniedForSuspendedWorkspace|AuthAndStoreGuardHelperBranches)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(HandleBackgroundJobsLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserSlicesReturnsSnapshot)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "BenchmarkResolveBackgroundJobScopeIDs$" -benchtime=100x`
- result: Passed. Runtime fail-closed behavior now extends to webhook delivery writes, support ticket message writes, payload-scoped background jobs (`workspace_id` / `user_id`), and store-guard status normalization for disabled/auth-blocked users and suspended workspaces.
- residual risk: Background-job scope parsing currently reads top-level payload keys only; nested scope keys are intentionally ignored for now.
- next suggested todo: Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.

### 2026-03-28 00:52 America/New_York

- completed todo: Ensure restore flows re-enable only the intended scopes and do not silently reopen revoked sessions, API keys, or operator overrides unless explicitly requested.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminMutationRestoreUserPreservesNonDisableAuthBlocks|AdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides|AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(SendDeniedForDisabledUser|SendDeniedForSuspendedWorkspace|WorkspaceScopedWritesBlockedWhenSuspended|UserScopedWritesBlockedWhenDisabled|DispatchNotificationOutboxPendingFailsSuspendedWorkspaceRows|DispatchNotificationOutboxPendingFailsDisabledUserRows|HandleBackgroundJobsFailsSuspendedWorkspaceQueue)$" -count=1`
- result: Passed. Existing restore handlers already satisfy the intended fail-safe contract: restore clears only targeted suspension/disable auth blocks, keeps old sessions revoked, and does not auto-unrevoke API keys or disabled webhook/operator states.
- residual risk: Restore capability selection is still implicit; operator-selectable partial re-enable controls remain a separate implementation step.
- next suggested todo: Add typed restore handlers that let operators choose which dependent capabilities come back automatically versus which stay manually revoked.

### 2026-03-28 00:48 America/New_York

- completed todo: Add backend audit events for admin-dashboard entry, slice views, drill-down access, and mutating admin actions so operator activity is queryable.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_audit_events.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetWorkspaceAdminSlices|GetSuperuserSlicesReturnsSnapshot|AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects)$" -count=1`
- result: Passed. Admin access paths now persist typed operator audit events for dashboard entry/slice/drilldown access and mutation application, and focused tests now assert those events are persisted/queryable.
- residual risk: Audit payload metadata is currently minimal (`{}`) and can be expanded once typed admin mutation/detail RPC surfaces are finalized.
- next suggested todo: Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.

### 2026-03-28 00:47 America/New_York

- completed todo: Define and enforce the exact runtime effects of user disable vs workspace suspend so chat send, dashboard access, API keys, webhooks, background jobs, and support actions all fail in a consistent way.
- files changed: `examples/server/ai-chat-wizard/server/app/runtime_access.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/notification_flow.go`, `examples/server/ai-chat-wizard/server/app/job_flows.go`, `examples/server/ai-chat-wizard/server/app/runtime_access_policy_test.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/store.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(SendDeniedForDisabledUser|SendDeniedForSuspendedWorkspace|WorkspaceScopedWritesBlockedWhenSuspended|DispatchNotificationOutboxPendingFailsSuspendedWorkspaceRows|HandleBackgroundJobsFailsSuspendedWorkspaceQueue|AdminControlMutationScopeBoundaries|RequireAdminMutationConfirmation|AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "TestSendStreamsThoughtsAndPersistsConversation$" -count=1`
- result: Passed. Runtime policy is now consistent and fail-closed across core paths: disabled users are denied chat send, suspended-workspace users are denied send/admin scope, workspace-scoped writes (API keys/webhooks/support/notification rows) are blocked while suspended, notification dispatch marks suspended-workspace rows failed without delivery callbacks, and workspace-queued background jobs fail closed when suspended.
- residual risk: Webhook delivery retry execution and externally exposed API-key/tooling RPC mutation surfaces are still read-heavy or stubbed; full end-user/operator workflow enforcement remains dependent on those pending mutation APIs.
- next suggested todo: Ensure restore flows re-enable only the intended scopes and do not silently reopen revoked sessions, API keys, or operator overrides unless explicitly requested.

### 2026-03-28 00:40 America/New_York

- completed todo: Add typed scoped query funcs and RPCs for superuser slices: global users, global usage, support queue, pricing controls, incidents, experiments, workspaces, and cost guardrails.
- files changed: `examples/server/ai-chat-wizard/sql/store/list_billing_plan_overages.sql`, `examples/server/ai-chat-wizard/sql/store/list_billing_quota_policies.sql`, `examples/server/ai-chat-wizard/sql/store/list_billing_upgrade_triggers.sql`, `examples/server/ai-chat-wizard/sql/store/list_incidents.sql`, `examples/server/ai-chat-wizard/sql/store/list_workspace_cost_guardrails.sql`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetSuperuserSlicesRequiresSURole|GetSuperuserSlicesReturnsSnapshot|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot)$" -count=1`
- result: Passed. Added typed superuser slice coverage through `GetSuperuserSlices` plus typed store query funcs for pricing controls, incidents, and workspace cost guardrails, alongside global users/usage/support/experiments/workspaces slices.
- residual risk: `GetSuperuserSlices` is read-only and does not yet provide mutation workflows for pricing controls or incidents; those remain tracked as separate TODOs.
- next suggested todo: Add backend audit events for admin-dashboard entry, slice views, drill-down access, and mutating admin actions so operator activity is queryable.

### 2026-03-28 00:39 America/New_York

- completed todo: Enforce feature-flag, experiment, and incident-control mutations behind the correct workspace-admin vs superuser boundaries.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_control_mutation_authz.go`, `examples/server/ai-chat-wizard/server/app/admin_control_mutation_authz_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestAdminControlMutationScopeBoundaries$" -count=1`
- result: Passed. Added explicit control-plane mutation authorization seams: feature-flag and experiment mutation stubs are superuser-only, while incident mutation stubs allow workspace-admin callers only inside their scoped workspace IDs and fail closed for out-of-scope or malformed requests.
- residual risk: These role boundaries are currently enforced at the shared mutation-authz seam; typed public mutation RPCs for feature flags, experiments, and incidents are still pending.
- next suggested todo: Define and enforce the exact runtime effects of user disable vs workspace suspend so chat send, dashboard access, API keys, webhooks, background jobs, and support actions all fail in a consistent way.

### 2026-03-28 00:37 America/New_York

- completed todo: Ensure disable and suspend actions revoke active sessions, block new auth, and deny downstream privileged RPCs immediately.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_mutation_effects.go`, `examples/server/ai-chat-wizard/server/app/admin_mutation_effects_test.go`, `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/store_auth.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/sql/store/delete_user_auth_block.sql`, `examples/server/ai-chat-wizard/sql/store/delete_user_auth_blocks_by_key.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(RequireAdminMutationConfirmation|AdminMutationDisableAndRestoreEnforcesImmediateAuthEffects|AdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects|AuthorizeAdminMutationActionScopes)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "TestAuthManagerSignupLoginAndTokenRoundTrip$" -count=1`
- result: Passed. Added an executable admin-mutation side-effect path (`parseExecuteAdminMutationAction`) that now enforces disable/suspend lifecycle behavior with persisted auth blocks, user access-state updates, token-version rotation, and immediate auth-session revocation; login/session validation now fail-closes disabled or blocked users, and workspace-admin scope resolution ignores suspended workspaces.
- residual risk: These side effects are wired at internal server seams and tests today; typed public admin mutation RPCs are still pending, so external operator workflows are not exposed yet.
- next suggested todo: Enforce feature-flag, experiment, and incident-control mutations behind the correct workspace-admin vs superuser boundaries.

### 2026-03-28 00:32 America/New_York

- completed todo: Add typed scoped query funcs and RPCs for workspace-admin slices: memberships, API keys, webhooks, audit logs, invitations, usage, and billing summary.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminDashboardQueries|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope|GetWorkspaceAdminSlices)$" -count=1`
- result: Passed. Added typed `GetWorkspaceAdminSlices` and `WorkspaceAdminBillingSummary` RPC surfaces with strict workspace-admin scope enforcement and typed slices for memberships, API keys, webhooks, audit logs, invitations, scoped usage events, and scoped billing summary rollups.
- residual risk: `GetWorkspaceAdminSlices` currently enforces workspace-admin scope and intentionally denies platform-scope superusers; if one unified operator view is needed, superuser-specific slice RPCs should cover that path explicitly.
- next suggested todo: Add typed scoped query funcs and RPCs for superuser slices: global users, global usage, support queue, pricing controls, incidents, experiments, workspaces, and cost guardrails.

### 2026-03-28 00:25 America/New_York

- completed todo: Add typed dashboard-home summary funcs and RPCs for the admin journey: users, conversations, usage, billing signals, incidents, support load, and experiment health.
- files changed: `examples/server/ai-chat-wizard/sql/store/get_admin_dashboard_summary.sql`, `examples/server/ai-chat-wizard/server/app/store_admin.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminDashboardQueries|AdminDashboardRPCs)$" -count=1` (baseline); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAdminDashboardQueries|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$" -count=1`
- result: Passed. `AdminDashboardSummary` now includes typed billing-signal metrics (`window_billing_events`, `window_open_invoices`, `window_open_dunning_events`), incident/support backlog counts (`open_incidents`, `open_support_tickets`), and experiment-health metrics (`active_experiments`, `unhealthy_experiments`, `window_experiment_assignments`), fully wired through SQL -> store -> RPC -> proto -> tests.
- residual risk: Workspace-admin scoped summaries currently keep the new ops/billing/experiment fields at zero because those enriched metrics are only sourced in the global summary query path today.
- next suggested todo: Add typed scoped query funcs and RPCs for workspace-admin slices: memberships, API keys, webhooks, audit logs, invitations, usage, and billing summary.

### 2026-03-28 00:17 America/New_York

- completed todo: Add re-auth or heightened-session checks for sensitive superuser actions so long-lived dashboard sessions do not automatically authorize mutating control-plane changes.
- files changed: `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/server_tool_stub.go`, `examples/server/ai-chat-wizard/server/app/server_tool_stub_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(SetServerToolPolicyReturnsUnimplemented|RunServerToolReturnsUnimplemented|SensitiveSuperuserMutationsRequireFreshSession)$" -count=1 -v`
- result: Passed. Sensitive superuser mutation surfaces now require one fresh authenticated session (`IssuedAt <= 20m`) through `parseRequireSuperuserMutationUserID`; stale but otherwise valid superuser tokens are fail-closed with `Unauthenticated` and must re-authenticate before mutating actions proceed.
- residual risk: Fresh-session enforcement currently covers sensitive superuser mutation stubs (`SetServerToolPolicy`, `RunServerTool`) and should be applied to future superuser mutation RPCs as they are added.
- next suggested todo: Define and enforce user-disable, user-restore, workspace-suspend, and workspace-restore authorization rules, including who can perform them and against which scopes.

### 2026-03-28 00:16 America/New_York

- completed todo: Define role-aware post-login redirect rules so normal users land in the chat app while admin-capable users can be routed intentionally into dashboard-capable entry points.
- files changed: `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/client/app/auth.go`, `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. Auth success/bootstrap now consumes one sanitized post-login route intent (`chat-wizard:post-login-route`) and `GetSession.role_summary.can_access_admin` to resolve the first destination: non-admin users fail closed to `/app` for admin intents, while admin-capable users can enter explicit dashboard/admin intents.
- residual risk: There is still no dedicated landing/auth UI affordance for authoring admin-intent targets, so this mainly applies when auth redirection originates from app/admin routes.
- next suggested todo: Add re-auth or heightened-session checks for sensitive superuser actions so long-lived dashboard sessions do not automatically authorize mutating control-plane changes.

### 2026-03-28 00:16 America/New_York

- completed todo: Add server-side write paths for first-thread creation metadata, first-reply completion markers, and reopen-thread analytics so the funnel can be queried end-to-end.
- files changed: `examples/server/ai-chat-wizard/server/app/funnel_first_chat.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_more_unit_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(ListAndLoadConversationBranches|SendStreamsThoughtsAndPersistsConversation|ListConversationsSyncsStarterMilestones)$" -count=1`
- result: Passed. First successful send now writes explicit first-thread/first-reply milestone markers and typed funnel completion metadata, and conversation reload now emits typed `thread_reopened` analytics so reopen behavior is queryable alongside first-chat funnel events.
- residual risk: Reopen analytics currently emits on every successful `LoadConversation`; downstream reporting may still need dedup/windowing depending on dashboard semantics.
- next suggested todo: Add typed dashboard-home summary funcs and RPCs for the admin journey: users, conversations, usage, billing signals, incidents, support load, and experiment health.

### 2026-03-28 00:15 America/New_York

- completed todo: Add backend support for first-run onboarding milestones and starter-state helpers so the app can distinguish brand-new users from returning users with prior threads.
- files changed: `examples/server/ai-chat-wizard/server/app/onboarding_state.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_more_unit_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(ListConversationsSyncsStarterMilestones|SendStreamsThoughtsAndPersistsConversation|ChatServerAuthRPCs|ChatShellHandlerTracksFirstChatFunnelSteps)$" -count=1`
- result: Passed. Added backend starter-state resolution/sync helpers and milestone persistence so the system now records brand-new vs returning state on conversation-list bootstrap and records first-thread + first-reply completion during first successful send.
- residual risk: Starter-state signals are persisted server-side but are not yet surfaced via a dedicated typed client bootstrap field.
- next suggested todo: Add server-side write paths for first-thread creation metadata, first-reply completion markers, and reopen-thread analytics so the funnel can be queried end-to-end.

### 2026-03-28 00:14 America/New_York

- completed todo: Harden logout/session/admin invalidation so cached peer auth cannot retain privileged dashboard access after revocation.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/server_gap_branches_test.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AuthSessionRPCBranchesCoverUnavailableAuthAndLogout|AdminDashboardRPCsRequireSessionAuth|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminAccessInvalidatesAfterRoleDowngradeAndSessionRevocation|AdminDashboardRPCsRequireSessionAuth|AuthSessionRPCBranchesCoverUnavailableAuthAndLogout)$" -count=1`
- result: Passed. Logout now evicts both peer session and cached peer auth identity, invalid-token contexts evict cached peer auth before fail-close, and admin scope resolution requires a session-backed authenticated context whenever auth is configured, preventing stale peer caches from preserving privileged dashboard access.
- residual risk: Non-admin RPCs can still use peer-auth fallback in auth-disabled test setups by design.
- next suggested todo: Define role-aware post-login redirect rules so normal users land in the chat app while admin-capable users can be routed intentionally into dashboard-capable entry points.

### 2026-03-28 00:12 America/New_York

- completed todo: Ensure logout, privilege downgrade, and session revocation immediately evict cached admin state and invalidate privileged dashboard access.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminAccessInvalidatesAfterRoleDowngradeAndSessionRevocation|AdminDashboardRPCsRequireSessionAuth)$" -count=1 -v`
- result: Passed. Admin dashboard access now has explicit lifecycle coverage proving immediate invalidation on privilege downgrade, full-session revocation, and logout, including peer auth-cache eviction and fail-closed status transitions (`PermissionDenied` vs `Unauthenticated`).
- residual risk: Coverage is currently server-side; client-side redirect nuances for admin-capable users are tracked in the next redirect-policy todo.
- next suggested todo: Define role-aware post-login redirect rules so normal users land in the chat app while admin-capable users can be routed intentionally into dashboard-capable entry points.

### 2026-03-28 00:11 America/New_York

- completed todo: Ensure typed store and RPC funcs exist for first-chat bootstrap data: profile defaults, selected settings, model catalog, conversation list, active conversation load, and default system prompt injection.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(ChatServerPreferenceAndConversationRPCs|SendAppliesDefaultSystemPromptOnNewConversation|NewChatServiceServerSupportsProviderStubs)$" -count=1`
- result: Passed. Existing typed store+RPC seams already cover the full first-chat bootstrap contract (profile defaults, selected settings, model catalog bootstrap, conversation list/load, and default system-prompt injection), so this todo was completed by verification rather than new implementation.
- residual risk: This checkpoint confirms server/store seams only; additional browser-level boot assertions remain tracked in other backlog items.
- next suggested todo: Add backend support for first-run onboarding milestones and starter-state helpers so the app can distinguish brand-new users from returning users with prior threads.

### 2026-03-28 00:10 America/New_York

- completed todo: Add typed funnel instrumentation for the visit-to-first-chat journey: landing viewed, pricing viewed, CTA clicked, auth started, auth completed, app booted, first thread created, first send started, first reply completed.
- files changed: `examples/server/ai-chat-wizard/server/app/funnel_first_chat.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_test.go`, `examples/server/ai-chat-wizard/server/app/testkit_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(ChatServerAuthRPCs|ChatShellHandlerTracksFirstChatFunnelSteps|SendStreamsThoughtsAndPersistsConversation)$" -count=1`
- result: Passed. Added one typed first-chat funnel instrumentation layer with canonical step/event keys, route-step mapping for landing/pricing/login surfaces, and runtime hooks in auth/session/send flows. Authenticated funnel steps now persist into `product_analytics_events` using active workspace scope plus automatic experiment bootstrap (`first-chat-funnel`), while unauthenticated pre-auth steps emit typed runtime logs.
- residual risk: Pre-auth landing/pricing/CTA/auth-started steps are currently log-instrumented only (no durable anonymous funnel row yet) because `product_analytics_events` is user/workspace scoped.
- next suggested todo: Ensure typed store and RPC funcs exist for first-chat bootstrap data: profile defaults, selected settings, model catalog, conversation list, active conversation load, and default system prompt injection.

### 2026-03-28 00:09 America/New_York

- completed todo: Start the admin list-mechanics regression todo with explicit pending scaffolding and block-state notes.
- files changed: `test/playwrightgo/examples/example100_admin_list_mechanics_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run "TestExample100Admin(ListMechanicsPending|MutationDiagnosticsPending|MutationRegressionPending|OpsRegressionPending)$" -count=1 -v`
- result: Passed (intentional skips). Added `TestExample100AdminListMechanicsPending` so the search/filter/sort/pagination regression slot is explicit and traceable until those list surfaces and query-state controls exist.
- residual risk: Admin list behavior still has no executable browser coverage because the list UIs and stateful controls are not implemented yet.
- next suggested todo: Implement typed admin list query surfaces and UI controls, then replace pending list/ops/mutation skip scaffolds with real end-to-end regressions.

### 2026-03-28 00:08 America/New_York

- completed todo: Start the admin-mutation diagnostics regression todo with explicit pending scaffolding and block-state notes.
- files changed: `test/playwrightgo/examples/example100_admin_mutation_diagnostics_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run "TestExample100Admin(MutationDiagnosticsPending|MutationRegressionPending|OpsRegressionPending)$" -count=1 -v`
- result: Passed (intentional skips). Added `TestExample100AdminMutationDiagnosticsPending` so the confirmation/submit/success/denial/rollback diagnostics regression slot is explicit with a precise skip reason until mutation lifecycle hooks exist.
- residual risk: Runtime diagnostics for mutation lifecycle stages remain unimplemented until typed mutation endpoints and UI flows are available.
- next suggested todo: Add focused regressions for admin list mechanics so user, workspace, support, incident, and billing views keep search, filter, sort, and pagination state stable across refresh and back navigation.

### 2026-03-28 00:07 America/New_York

- completed todo: Enforce persisted token lifecycles for signup verification, password reset, and update-password flows using the new token tables instead of ad hoc state.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/store_auth.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/sql/store/create_email_verification_token.sql`, `examples/server/ai-chat-wizard/sql/store/get_email_verification_token_by_hash.sql`, `examples/server/ai-chat-wizard/sql/store/mark_email_verification_token_verified.sql`, `examples/server/ai-chat-wizard/sql/store/create_password_reset_token.sql`, `examples/server/ai-chat-wizard/sql/store/get_password_reset_token_by_hash.sql`, `examples/server/ai-chat-wizard/sql/store/consume_password_reset_token.sql`, `examples/server/ai-chat-wizard/sql/store/update_user_password_hash.sql`, `examples/server/ai-chat-wizard/sql/store/revoke_auth_sessions_by_user.sql`, `examples/server/ai-chat-wizard/server/app/auth_test.go`, `examples/server/ai-chat-wizard/server/app/store_test.go`, `examples/server/ai-chat-wizard/server/app/auth_lifecycle_benchmark_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAuthFlowTokenLifecycle|AuthManagerPersistedTokenLifecycles)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AuthManagerSignupLoginAndTokenRoundTrip|AuthManagerPersistedTokenLifecycles|StoreAuthSessionLifecycle|StoreAuthFlowTokenLifecycle)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench BenchmarkAuthFlowTokenHash -benchmem -count=1`
- result: Passed. Signup now persists one pending email-verification token, password-reset completion consumes one persisted reset token and rotates password/session state (`users.password_hash`, `auth_token_versions`, `auth_sessions`), and update-password reuses that persisted reset-token lifecycle rather than ad hoc in-memory state.
- residual risk: Public forgot/reset/verify RPC endpoints and delivery UX are still not exposed yet, so these lifecycles are currently validated at auth/store seams.
- next suggested todo: Ensure logout, privilege downgrade, and session revocation immediately evict cached admin state and invalidate privileged dashboard access.

### 2026-03-28 00:06 America/New_York

- completed todo: Start the billing/support/incident focused-regression todo with explicit pending scaffolding and block-state notes.
- files changed: `test/playwrightgo/examples/example100_admin_ops_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run "TestExample100Admin(OpsRegressionPending|MutationRegressionPending)$" -count=1 -v`
- result: Passed (intentional skips). Added `TestExample100AdminOpsRegressionPending` with precise skip text so the missing billing-intervention, support-triage, and incident-control regression slot is explicit and traceable until those operator surfaces land.
- residual risk: These workflows still have zero executable browser coverage because corresponding mutation/query UI + RPC paths are not available in this revision.
- next suggested todo: Implement typed billing/support/incident admin surfaces and convert the pending skip scaffolds into real end-to-end regressions.

### 2026-03-28 00:05 America/New_York

- completed todo: Add one short repo-layout section to the example docs that explains where logs, runtime state, generated artifacts, scripts, and source files belong.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added `Repo layout rules` to README with explicit placement guidance for root entry docs, `docs/` secondary docs, `scripts/`, `bin/runtime` state/logs, generated artifacts under `bin/`, and source directories.
- residual risk: Existing local developer habits may still produce ad hoc files in non-standard locations until tooling enforces the same conventions.
- next suggested todo: None; all current Agent 4 non-UI todo items are completed.

### 2026-03-28 00:04 America/New_York

- completed todo: Start the next admin-mutation regression todo with explicit test scaffolding and block-state documentation.
- files changed: `test/playwrightgo/examples/example100_admin_mutation_pending_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run "TestExample100(AdminRoleGuardsAndDeepLinks|AdminMutationRegressionPending)$" -count=1 -v`
- result: Passed (with one intentional skip). Added `TestExample100AdminMutationRegressionPending` so the browser mutation regression slot is explicit and searchable, with a precise skip reason documenting missing typed mutation RPC/UI surfaces in this revision.
- residual risk: Disable/restore/suspend mutation workflows still have no executable browser coverage until mutation endpoints and controls are implemented.
- next suggested todo: Implement typed admin mutation RPCs and basic dashboard controls, then replace the pending skip with a real browser mutation regression.

### 2026-03-28 00:03 America/New_York

- completed todo: Update `README.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, and any other affected docs after the cleanup so every referenced path still opens correctly.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`; `rg -n "scripts/dev\.ps1|scripts/dev\.sh|build-client\.ps1|run-server\.ps1|BENCHMARKS\.md" ...`
- result: Passed. Updated README startup/layout references to the current managed-runtime flow and current tree, added bug-template path reference in manual smoke guidance, and added schema-doc cross-reference guidance to the doc map location.
- residual risk: Historical TODO/changelog text intentionally still mentions previous filenames/paths and may look stale outside date context.
- next suggested todo: Add one short repo-layout section to the example docs that explains where logs, runtime state, generated artifacts, scripts, and source files belong.

### 2026-03-28 00:01 America/New_York

- completed todo: Add focused regressions for workspace-admin vs superuser route guards, direct deep links into dashboard slices, and empty/error dashboard states.
- files changed: `test/playwrightgo/examples/example100_admin_journey_test.go`, `test/playwrightgo/examples/example100_admin_role_guard_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AdminJourneyRegression -count=1 -v`; `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AdminRoleGuardsAndDeepLinks -count=1 -v`
- result: Passed. Admin-journey coverage now uses real admin deep links with cookie-backed route guards, and a new focused regression locks superuser allow, workspace-admin scoped-empty state behavior, and normal-user fail-closed redirect plus permission-denied admin RPC outcomes.
- residual risk: Admin mutation workflows and billing/support/incident operator flows are still uncovered at the browser level.
- next suggested todo: Add one browser-level admin-mutation regression that covers disable user, restore user, suspend workspace, restore workspace, and the resulting UI state transitions.

### 2026-03-28 00:00 America/New_York

- completed todo: Remove stale root-level files that only existed because of the old layout once their replacements are wired and validated.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Test-Path` checks over stale root candidates (`server.stderr.log`, `server.stdout.log`, `BUG_REPORT_TEMPLATES.md`, `PERFORMANCE.md`, `BENCHMARKS.md`) plus root-file inventory listing.
- result: Passed. Old root-layout artifacts are no longer present, and root file inventory now contains only active entry docs/config plus source directories.
- residual risk: Manual developer-created files can still appear at root outside this managed cleanup set.
- next suggested todo: Update `README.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, and any other affected docs after the cleanup so every referenced path still opens correctly.

### 2026-03-27 23:59 America/New_York

- completed todo: Update `.gitignore` for example 100 so logs, runtime outputs, temp artifacts, and local-only generated files are ignored from their new locations.
- files changed: `examples/server/ai-chat-wizard/.gitignore`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Get-Content examples/server/ai-chat-wizard/.gitignore`
- result: Passed. Ignore rules now reflect the current layout by removing obsolete `server/server.*.log` entries and explicitly covering root fallback log outputs plus `log/` while keeping local DB and generated-artifact ignores.
- residual risk: Local developers with custom output paths outside these patterns may still need personal/global gitignore rules.
- next suggested todo: Remove stale root-level files that only existed because of the old layout once their replacements are wired and validated.

### 2026-03-27 23:58 America/New_York

- completed todo: Move non-first-class docs into a dedicated `docs/` folder while keeping the primary entry docs at the example root, then update all links and cross-references after the move.
- files changed: `examples/server/ai-chat-wizard/docs/BUG_REPORT_TEMPLATES.md` (moved), `examples/server/ai-chat-wizard/docs/PERFORMANCE.md` (moved), `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/DOCS_MAP.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`; `Test-Path` checks confirmed root copies are absent and `docs/` copies exist.
- result: Passed. Secondary docs now live under `docs/` while primary entry docs remain at root, and active docs-map/readme/todo references were updated to the new paths.
- residual risk: Historical changelog entries still mention prior root-level paths by design, which may be confused with current layout if read without date context.
- next suggested todo: Update `.gitignore` for example 100 so logs, runtime outputs, temp artifacts, and local-only generated files are ignored from their new locations.

### 2026-03-27 23:57 America/New_York

- completed todo: Add runtime diagnostics for the admin dashboard journey so role resolution, admin route entry, dashboard bootstrap, slice fetches, and unauthorized transitions emit actionable logs.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run 'Test(ChatShellRoutingHelpers|ChatShellHandler|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$' -count=1`
- result: Passed. Admin journey diagnostics now emit actionable allow/deny logs for role resolution, dashboard bootstrap and slice fetch lifecycle, and HTTP unauthorized deep-link transitions with explicit next-action hints.
- residual risk: Browser-level deep-link expectations are currently evolving (`/app/settings` and `/app/dashboard` variants), so this checkpoint validates diagnostics primarily at server and RPC seams.
- next suggested todo: Add focused regressions for workspace-admin vs superuser route guards, direct deep links into dashboard slices, and empty/error dashboard states.

### 2026-03-27 23:56 America/New_York

- completed todo: Add a typed auth bootstrap contract that returns session status, role summary, and expiry information needed for safe app-shell and dashboard entry decisions.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/client/app/auth.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/proto -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(ChatServerAuthRPCs|GetSessionReturnsTypedRoleSummary|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$' -count=1`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. `GetSession` now returns typed auth-bootstrap metadata (`session_status`, session/token identifiers, expiry, and typed role summary), server role scope is resolved into `user`/`workspace_admin`/`superuser`, and the client auth session checker consumes typed status/expiry fields to drive safe fallback decisions.
- residual risk: Role summary is still bootstrap metadata only; dedicated role-aware dashboard IA and post-login route policy wiring are tracked as separate follow-up todos.
- next suggested todo: Enforce persisted token lifecycles for signup verification, password reset, and update-password flows using the new token tables instead of ad hoc state.

### 2026-03-27 23:56 America/New_York

- completed todo: Normalize file placement so docs stay at the example root, scripts stay under `scripts/`, runtime outputs stay under runtime folders, and generated artifacts stay under `bin/` or another clearly non-source location.
- files changed: `examples/server/ai-chat-wizard/bin/server/chat-wizard-server.exe` (moved), `examples/server/ai-chat-wizard/bin/runtime/legacy-artifacts/chat-wizard-server.exe~` (moved), `examples/server/ai-chat-wizard/bin/runtime/legacy-artifacts/main.wasm` (moved), `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Get-ChildItem examples/server/ai-chat-wizard/bin -File` confirmed no loose root files remain (`bin_root_files=none`).
- result: Passed. Example-100 file placement is now cleaner: docs remain at root, runtime state/logs are under `bin/runtime`, and generated binaries/artifacts are under `bin/server` or `bin/runtime/legacy-artifacts`.
- residual risk: Legacy artifact relocation can break any external scripts outside this repo that hardcode old `bin/` root paths.
- next suggested todo: Move non-first-class docs into a dedicated `docs/` folder while keeping the primary entry docs at the example root, then update all links and cross-references after the move.

### 2026-03-27 23:55 America/New_York

- completed todo: Rename inconsistent or low-signal files so the example reads professionally to a new maintainer, then update all references in scripts, docs, tests, and server/client code.
- files changed: `examples/server/ai-chat-wizard/PERFORMANCE.md` (renamed from `BENCHMARKS.md`), `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`; `rg -n "BENCHMARKS\.md" .\examples\server\ai-chat-wizard`
- result: Passed. The benchmark reference doc now has a clearer maintainer-facing name (`PERFORMANCE.md`), and in-repo references were updated to the new path.
- residual risk: External links/bookmarks outside this repo may still point to the old `BENCHMARKS.md` filename.
- next suggested todo: Normalize file placement so docs stay at the example root, scripts stay under `scripts/`, runtime outputs stay under runtime folders, and generated artifacts stay under `bin/` or another clearly non-source location.

### 2026-03-27 23:53 America/New_York

- completed todo: Move any other loose runtime outputs out of the example root into a consistent runtime location such as `bin/runtime/`, `bin/logs/`, or another single clear convention.
- files changed: `examples/server/ai-chat-wizard/server/app/logging.go`, `examples/server/ai-chat-wizard/server/app/coverage_gap_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(ServerLoggerCreatesFileSink|ServerLoggerFailureBranches|GetLogTailReturnsLatestServerLines|GetLogTailSupportsAllSourcesAndFilter|GetLogTailRejectsInvalidSource|GetLogTailRequiresSuperuserAccess)$" -count=1`
- result: Passed. Runtime server/client log sinks now default to `bin/runtime/logs` instead of root-level `log/`, and logger failure-branch coverage was updated for nested log-directory setup.
- residual risk: Any manually started server process with custom shell redirection can still create ad hoc root-level log files outside these managed defaults.
- next suggested todo: Rename inconsistent or low-signal files so the example reads professionally to a new maintainer, then update all references in scripts, docs, tests, and server/client code.

### 2026-03-27 23:53 America/New_York

- completed todo: Define session and auth-expiry behavior for long-lived dashboard tabs so expired admin sessions fall back safely without leaving stale privileged UI visible.
- files changed: `examples/server/ai-chat-wizard/client/app/auth.go`, `examples/server/ai-chat-wizard/client/app/auth_wasm_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. Auth session handling now runs periodic authenticated session checks (2-minute interval) and fail-closes into the login shell with session-expired messaging when long-lived tabs carry expired/revoked/missing sessions, preventing stale privileged UI from remaining visible.
- residual risk: Session/role/expiry data is still inferred from current auth RPCs and token claims; a dedicated typed bootstrap contract with explicit expiry metadata is still pending.
- next suggested todo: Add a typed auth bootstrap contract that returns session status, role summary, and expiry information needed for safe app-shell and dashboard entry decisions.

### 2026-03-27 23:52 America/New_York

- completed todo: Add one browser-level admin journey regression that covers homepage load, login, role resolution, admin-entry visibility, dashboard-home load, slice navigation, and back/refresh behavior.
- files changed: `test/playwrightgo/examples/example100_admin_journey_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AdminJourneyRegression -count=1`
- result: Passed. Added a browser-admin flow regression that starts at `/`, signs in as `admin@example.com`, validates settings/dashboard entry and panel routing, confirms slice route navigation with back/refresh stability, and verifies role-scoped `GetAdminDashboard`, `ListAdminUsers`, and `ListAdminConversations` access using the authenticated browser token.
- residual risk: Example 100 still has no dedicated dashboard browser IA, so this regression uses settings-route navigation plus admin RPC checks as the closest journey proxy.
- next suggested todo: Add runtime diagnostics for the admin dashboard journey so role resolution, admin route entry, dashboard bootstrap, slice fetches, and unauthorized transitions emit actionable logs.

### 2026-03-27 23:51 America/New_York

- completed todo: Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.
- files changed: `examples/server/ai-chat-wizard/bin/runtime/logs/server.stderr.log`, `examples/server/ai-chat-wizard/bin/runtime/logs/server.stdout.log`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Test-Path` checks confirmed root log files are absent and moved files exist in `bin/runtime/logs`; `rg -n "server\.stderr\.log|server\.stdout\.log"` over `cmd/server`, `scripts`, `server`, and `tools/gwc/examples_managed.go` returned no active path references.
- result: Passed. Runtime stderr/stdout artifacts were relocated out of the example root and no current launcher/server paths in this repo still target the old root filenames.
- residual risk: Historical or external shell aliases outside this repo could still redirect process output to root-level files.
- next suggested todo: Move any other loose runtime outputs out of the example root into a consistent runtime location such as `bin/runtime/`, `bin/logs/`, or another single clear convention.

### 2026-03-27 23:49 America/New_York

- completed todo: Keep `SCHEMA_TABLES.md` synchronized with the control-plane, auth, onboarding, and admin workflow stories so the docs match the real table usage.
- files changed: `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Synced classification to real wiring by moving `password_reset_tokens` and `email_verification_tokens` to `Persistence-only`, expanded wiring snapshot bullets for auth/admin/billing seams, and added a story-alignment section that maps active flow stories to concrete table usage.
- residual risk: This keeps docs accurate for current implementation, but pending admin mutation RPC rollout may shift classification for several control-plane tables again.
- next suggested todo: Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.

### 2026-03-27 23:48 America/New_York

- completed todo: Add server-side checks for admin deep links, slice RPCs, and detail views so unauthorized dashboard routes fail closed and redirect or render a sane denied state.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/runtime_helpers_additional_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(ChatShellRoutingHelpers|ChatShellHandler|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$' -count=1`
- result: Passed. Unauthorized admin deep links now fail closed at HTTP shell entry and redirect to `/app`, while authorized admin users still receive the shell. Dashboard slice RPCs now use slice-specific admin scope gates and emit structured denial logs.
- residual risk: Dedicated admin detail-view RPCs are not implemented yet, so detail-level allow/deny checks are currently enforced via scoped list/filter behavior plus the new route guard coverage.
- next suggested todo: Define session and auth-expiry behavior for long-lived dashboard tabs so expired admin sessions fall back safely without leaving stale privileged UI visible.

### 2026-03-27 23:46 America/New_York

- completed todo: Update `DESIGN.md` so it reflects the current intended product surface, dashboard direction, and any material UI or IA changes made since the original design pass.
- files changed: `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Replaced the stale visual-redesign spec with a current design-intent document aligned to the shipped route model, role-split dashboard direction, state/feedback rules, and IA constraints.
- residual risk: The design intent now matches current direction, but repo-layout cleanup tasks are still open and can change path references again.
- next suggested todo: Keep `SCHEMA_TABLES.md` synchronized with the control-plane, auth, onboarding, and admin workflow stories so the docs match the real table usage.

### 2026-03-27 23:44 America/New_York

- completed todo: Add focused regression coverage for empty-state boot, first-thread creation, scroll anchoring during first reply, and reopen-after-refresh behavior.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -count=1`
- result: Passed. `TestExample100AuthenticatedHappyPath` already exercises empty-state boot, first-thread route creation after send, first-reply stream anchoring, and prompt/route persistence after reload and sidebar reopen.
- residual risk: This checkpoint covers one focused happy path; admin journey and admin mutation browser regressions remain open.
- next suggested todo: Add one browser-level admin journey regression that covers homepage load, login, role resolution, admin-entry visibility, dashboard-home load, slice navigation, and back/refresh behavior.

### 2026-03-27 23:43 America/New_York

- completed todo: Add one concise docs map to the example that tells maintainers which file to read for setup, architecture, smoke testing, schema reference, design intent, and changelog history.
- files changed: `examples/server/ai-chat-wizard/DOCS_MAP.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added a concise docs map covering setup, flow docs, smoke/testing, bug templates, schema, design, operator runbook, changelog, and backlog.
- residual risk: Design refresh and repo-layout/log-location cleanup tasks remain open in Agent 4.
- next suggested todo: Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.

### 2026-03-27 23:42 America/New_York

- completed todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.
- files changed: `examples/server/ai-chat-wizard/BUG_REPORT_TEMPLATES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added a dedicated bug-report template doc with route, first-chat, and admin-dashboard templates that require reproducible environment/state/evidence fields.
- residual risk: Remaining Agent 4 docs cleanup tasks (design/schema sync/docs map/layout hygiene) are still open.
- next suggested todo: Add one concise docs map to the example that tells maintainers which file to read for setup, architecture, smoke testing, schema reference, design intent, and changelog history.

### 2026-03-27 23:41 America/New_York

- completed todo: Add runtime diagnostics for the visit-to-first-chat path so boot, gRPC connect, worker readiness, model-catalog load, conversation bootstrap, and first-send failures log one actionable warning or error each.
- files changed: `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/model_preferences.go`, `examples/server/ai-chat-wizard/client/app/conversations.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`; `go test -tags playwrightgo ./test/playwrightgo/examples -run "TestExample100(VisitToFirstChatRegression|ScrollToBottomButton)$" -c`
- result: Added explicit startup/boot-shell, model-catalog bootstrap, and conversation-bootstrap diagnostics while keeping existing gRPC, worker readiness, and send/stream failure logs intact.
- residual risk: Live browser execution remains intermittently blocked by unrelated in-flight server compile edits in this shared worktree.
- next suggested todo: Add focused regression coverage for empty-state boot, first-thread creation, scroll anchoring during first reply, and reopen-after-refresh behavior.

### 2026-03-27 23:41 America/New_York

- completed todo: Document the disable-vs-suspend semantics so maintainers and operators know exactly which downstream capabilities are supposed to turn off in each case.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now has an explicit disable/restore vs suspend/restore semantics matrix with scoped impact and guardrails.
- residual risk: Bug-report templates and remaining Agent 4 docs hygiene tasks are still open.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:40 America/New_York

- completed todo: Document the admin search/filter/pagination conventions so operators know how large lists, saved context, and back-navigation are expected to behave.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now documents stable admin list conventions across search, filter, and pagination behavior with route/back-navigation expectations.
- residual risk: Disable-vs-suspend semantics and bug-report templates remain open in Agent 4.
- next suggested todo: Document the disable-vs-suspend semantics so maintainers and operators know exactly which downstream capabilities are supposed to turn off in each case.

### 2026-03-27 23:38 America/New_York

- completed todo: Add one browser-level visit-to-first-chat regression that covers landing load, pricing/auth navigation, login/signup handoff, authenticated shell boot, first send, streamed reply, and canonical thread-route normalization.
- files changed: `test/playwrightgo/examples/example100_visit_first_chat_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100VisitToFirstChatRegression -v` (blocked by unrelated in-progress server compile failure in worktree: `admin_dashboard.go` missing `strings` import); `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100VisitToFirstChatRegression -c`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: New browser regression coverage now exists for the full visitor-to-first-chat route/auth/send flow with canonical thread-route assertion and sidebar-reopen fallback when immediate route sync is deferred.
- residual risk: Runtime execution of the new Playwright flow is currently blocked by unrelated server compile instability from concurrent edits in this worktree.
- next suggested todo: Add runtime diagnostics for the visit-to-first-chat path so boot, gRPC connect, worker readiness, model-catalog load, conversation bootstrap, and first-send failures log one actionable warning or error each.

### 2026-03-28 00:38 America/New_York

- completed todo: Enforce dashboard access by resolved role so normal users cannot reach admin surfaces, workspace admins get workspace-scoped data, and superusers get platform-scoped data.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/sql/store/list_workspace_memberships_by_user.sql`, `examples/server/ai-chat-wizard/sql/store/list_workspace_memberships_by_workspace.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$' -count=1`
- result: Passed. Admin dashboard RPCs now resolve caller role scope before query execution: normal authenticated users are denied, superusers keep platform-wide access, and workspace admins receive workspace-scoped users/usage/conversations derived from active membership boundaries.
- residual risk: Workspace-scoped dashboard aggregates currently derive from filtered recent/admin list queries rather than dedicated workspace-optimized SQL rollups, so very large datasets may need follow-up query specialization.
- next suggested todo: Add server-side checks for admin deep links, slice RPCs, and detail views so unauthorized dashboard routes fail closed and redirect or render a sane denied state.

### 2026-03-28 00:31 America/New_York

- completed todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.
- files changed: `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(BuildSendAccessDeniedStatus|RequireUserEntitlement|RequireUsageBudget|SendRejectsMissingAuthenticatedUserBeforeProviderWork|SendUsesBillingPlanDefaultModel)$' -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run '^$' -bench 'Benchmark(BuildSendAccessDeniedStatus|RequireUsageBudget)$' -benchtime=200x`
- result: Passed. Send-path entitlement and usage-budget denials now emit a structured `send_access` decision shape with `decision`, `plan`, `action`, `reason`, and `first_paid_action=chat.send`, so soft-upgrade prompts are distinguishable from hard blocks while preserving allow-path behavior.
- residual risk: Soft-upgrade decisions currently surface as structured RPC errors and are not yet rendered as a dedicated upgrade UI treatment in the wasm client.
- next suggested todo: Enforce dashboard access by resolved role so normal users cannot reach admin surfaces, workspace admins get workspace-scoped data, and superusers get platform-scoped data.

### 2026-03-27 23:30 America/New_York

- completed todo: Update the app version number and ensure the displayed version string is sourced consistently.
- files changed: `examples/server/ai-chat-wizard/internal/buildinfo/version.go`, `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/client/app/sidebar.go`, `examples/server/ai-chat-wizard/server/app/bootstrap.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`; `go test ./examples/server/ai-chat-wizard/internal/buildinfo -count=1`; attempted browser rerun: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -v` (blocked by unrelated in-progress server compile failure in worktree: `server.go` undefined `parseBuildUserMemorySignature*` helpers)
- result: Passed for client + shared source wiring. Version is now `v2026.03.27.2` from one shared build-info source consumed by both the sidebar version badge and the server boot-shell version label.
- residual risk: Full browser-path reruns are currently unstable due concurrent server-side compile edits unrelated to this version-source change.
- next suggested todo: Add one browser-level visit-to-first-chat regression that covers landing load, pricing/auth navigation, login/signup handoff, authenticated shell boot, first send, streamed reply, and canonical thread-route normalization.

### 2026-03-28 00:26 America/New_York

- completed todo: Add auth-state enforcement for the public-to-app transition so blocked, expired, revoked, or malformed sessions fail into a sane auth route instead of a half-booted app shell.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/auth.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run '^TestChatServerAuthRPCs$' -count=1`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test ./examples/server/ai-chat-wizard/client/app -run '^TestShouldRedirectUnauthenticatedRouteToLanding$' -count=1` (blocked on local runner: `%1 is not a valid Win32 application`)
- result: Passed for server auth/session regression and wasm client compile. `GetSession` now returns `Unauthenticated` for invalid/revoked/malformed metadata tokens, token-bearing requests no longer fall back to peer-bound auth state, auth bootstrap reports clearer session rejection failures, and resolved unauthenticated `/app...` routes are redirected back to landing.
- residual risk: Runtime coverage for the new js/wasm route-guard unit test is still blocked in this Windows shell because direct `GOOS=js GOARCH=wasm go test` execution could not launch the wasm test binary.
- next suggested todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.

### 2026-03-28 00:10 America/New_York

- completed todo: Expand the example-100 backlog and docs for chat capability planning, admin workflow gaps, testing stories, and repo cleanup guidance.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Get-Content examples/server/ai-chat-wizard/TODO.md`; `Get-Content examples/server/ai-chat-wizard/CHANGELOG.md`
- result: Passed. The backlog now includes future chat capability stories (calendar/email hooks, web search, shareable chats, scheduled jobs, image upload, ask-with-docs, skills, workflows, code interpreter), clearer admin/operator workflow coverage, explicit bug-fix/testing stories, and concrete repo-cleanup/doc-refresh instructions.
- residual risk: These changes only improve planning and documentation shape; the newly added capability stories are not yet decomposed into implementation slices across backend, authz, runtime, and UI.
- next suggested todo: Choose which future chat capabilities should move from the planning section into the active agent backlog.

### 2026-03-27 23:30 America/New_York

- completed todo: Fix the memory extraction system so remembered items are extracted reliably, deduplicated sanely, editable, and consistent with what users see in the remembered-preferences UI.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/memory_helpers_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(CustomPromptAndMemoryHelperFunctions|ExtractAndStoreUserMemoriesBranches|ExtractAndStoreUserMemoriesReusesExistingKeys|ExtractAndStoreUserMemoriesLogsLifecycle|ChatServerUserMemoryRPCs|SendStreamsThoughtsAndPersistsConversation|SendAppliesDefaultSystemPromptOnNewConversation)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle|SendAppliesDefaultSystemPromptOnNewConversation|ExtractAndStoreUserMemoriesReusesExistingKeys)" -count=1`
- result: Passed. Extraction now deduplicates candidate memories by normalized signature, prefers the strongest duplicate candidate, reuses existing keys for matching signatures, and deduplicates injected memory prompt blocks so UI-visible memories and model-injected context stay consistent.
- residual risk: Existing duplicate rows already persisted with different signatures are not yet backfilled or merged by a dedicated migration job.
- next suggested todo: Keep `SCHEMA_TABLES.md` in sync as store funcs and RPCs are added for the new tables.

### 2026-03-27 23:38 America/New_York

- completed todo: Add an operator runbook for local start, seed, build-client, and superuser/admin verification flows.
- files changed: `examples/server/ai-chat-wizard/OPERATOR_RUNBOOK.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added `OPERATOR_RUNBOOK.md` with concrete repo-root operator steps for env setup, client builds, DB seed, managed server lifecycle, user chat verification, and superuser/admin backend verification.
- residual risk: The runbook documents current admin verification through backend tests because a dedicated dashboard route-level UI flow is still pending.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:36 America/New_York

- completed todo: Document which schema tables are persistence-only vs fully wired into store, RPC, jobs, and UI.
- files changed: `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `SCHEMA_TABLES.md` now includes explicit wiring classes (`Store+RPC+Jobs+UI`, `Store+RPC+Jobs`, `Store+RPC`, `Store-only`, `Persistence-only`) with current table assignments.
- residual risk: Classification is a point-in-time snapshot and must be updated with each new store/RPC/job/UI wiring change to avoid drift.
- next suggested todo: Add an operator runbook for local start, seed, build-client, and superuser/admin verification flows.

### 2026-03-27 23:32 America/New_York

- completed todo: Keep `SCHEMA_TABLES.md` in sync as store funcs and RPCs are added for the new tables.
- files changed: `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `SCHEMA_TABLES.md` now includes a wiring snapshot that maps newly wired operational/growth tables to current store funcs, `GetSuperuserControlPlane`, notification/job flow helpers, and user-memory extraction/runtime entry points.
- residual risk: The snapshot covers newly wired tables but does not yet classify every table as persistence-only vs store/RPC/jobs/UI wired.
- next suggested todo: Document which schema tables are persistence-only vs fully wired into store, RPC, jobs, and UI.

### 2026-03-27 23:26 America/New_York

- completed todo: Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/server_helpers_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/server/ai-chat-wizard/server/app/unit_additional_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(NormalizationAndPromptHelpers|RPCFallbacksWhenStoreOrProvidersAreUnavailable|MemoryRPCFallbacksWithoutStore|SendAppliesDefaultSystemPromptOnNewConversation|SendStreamsThoughtsAndPersistsConversation)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle|SendAppliesDefaultSystemPromptOnNewConversation)" -count=1`
- result: Passed. Default prompt fallback now resolves runtime template variables (`{{date}}`, `{{time}}`, `{{memories}}`) and is applied automatically on first-send/new-conversation paths when no user override is stored.
- residual risk: The default prompt is now enforced server-side; there is still no per-workspace default prompt policy or prompt-version audit trail.
- next suggested todo: Fix the memory extraction system so remembered items are extracted reliably, deduplicated sanely, editable, and consistent with what users see in the remembered-preferences UI.

### 2026-03-27 23:21 America/New_York

- completed todo: Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage.
- files changed: `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `MANUAL_SMOKE.md` now includes a journey-to-coverage matrix across browser specs, focused package tests, and manual smoke groups.
- residual risk: Bug-report templates and broader repo-layout documentation tasks remain open in Agent 4.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:20 America/New_York

- completed todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.
- files changed: `examples/server/ai-chat-wizard/server/app/job_flows.go`, `examples/server/ai-chat-wizard/server/app/job_flows_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestHandleBackgroundJobsLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle)" -count=1`
- result: Passed. Server-side job-flow helpers now enqueue and dispatch typed weekly-summary, dunning-retry, retention-purge, and health-score-refresh jobs with persisted running/completed/failed or retry-to-pending state transitions.
- residual risk: Dispatch remains callback-driven and synchronous in-process; a persistent scheduler loop and distributed worker coordination are still pending.
- next suggested todo: Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.

### 2026-03-27 23:19 America/New_York

- completed todo: Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate; refresh `MANUAL_SMOKE.md` so manual verification covers landing routes, auth, first chat, settings, admin dashboard entry, and key operator actions.
- files changed: `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `MANUAL_SMOKE.md` now defines release-gate checklist groups, concrete visitor/first-chat/admin flow steps, operator-action guidance, and optional focused automation commands.
- residual risk: Testing-story matrix and bug-report template docs are still open in Agent 4.
- next suggested todo: Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage.

### 2026-03-27 23:18 America/New_York

- completed todo: Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.
- files changed: `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/store.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/server/ai-chat-wizard/sql/store/sum_usage_tokens_since.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "Test(RequireUserEntitlement|RequireUsageBudget|SendStreamsThoughtsAndPersistsConversation|SendPersistsUsageTraceMetadata|SendUsesBillingPlanDefaultModel|SendRejectsConversationOwnedByAnotherUser|SendRejectsMissingAuthenticatedUserBeforeProviderWork)$" -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "^$" -bench "BenchmarkRequireUsageBudget$" -benchtime=200x`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -count=1`
- result: Passed. `parseRequireUsageBudget` now enforces monthly token limits from billing access control, applies per-user send-rate and concurrency limits, and returns a lease release function that `Send` defers for correct stream-lifetime concurrency tracking.
- residual risk: Rate and concurrency accounting is process-local in-memory state today; limits reset on server restart and are not yet distributed across multiple server instances.
- next suggested todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.

### 2026-03-27 23:17 America/New_York

- completed todo: Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now defines an admin-dashboard incident workflow that captures role scope, entry path, slice-load failures, mutation-state evidence, and guard-safe validation steps.
- residual risk: Manual smoke matrix/test-story/bug-template docs are still open in Agent 4.
- next suggested todo: Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate.

### 2026-03-27 23:16 America/New_York

- completed todo: Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now includes a first-chat incident workflow with explicit auth, model-bootstrap, send/stream, and route-normalization evidence capture steps.
- residual risk: Admin-dashboard bug-fix workflow documentation and several release-gate/checklist docs remain open.
- next suggested todo: Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps.

### 2026-03-27 23:15 America/New_York

- completed todo: Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now includes a concrete public-route incident workflow with route, hydration, router-state, and diagnostics capture steps before patching.
- residual risk: First-chat and admin-dashboard bug-fix workflow docs are still open and should mirror this level of capture specificity.
- next suggested todo: Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues.

### 2026-03-27 23:14 America/New_York

- completed todo: Add customer-facing notification flows backed by `notification_outbox`.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/notification_flow.go`, `examples/server/ai-chat-wizard/server/app/notification_flow_test.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/sql/store/list_notification_outbox_pending.sql`, `examples/server/ai-chat-wizard/sql/store/update_notification_outbox_status.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(DispatchNotificationOutboxPendingLifecycle|StoreSuperuserControlPlaneLifecycle)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle)" -count=1`
- result: Passed. Pending notification rows now dispatch through a typed flow and transition to `sent`/`failed` status while future-scheduled rows remain pending.
- residual risk: Delivery is currently callback-driven dispatch plumbing; production channel adapters and scheduled job orchestration remain pending.
- next suggested todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.

### 2026-03-27 23:13 America/New_York

- completed todo: Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.
- files changed: `examples/server/ai-chat-wizard/client/app/scroll_memory.go`, `test/playwrightgo/examples/example100_scroll_to_bottom_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -v` (blocked by pre-existing server compile failure: `server.go:578 assignment mismatch: 2 variables but parseS.parseRequireUsageBudget returns 1 value`); `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -c`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: The jump-to-bottom action now issues a smooth scroll followed by a timed settle snap to exact bottom, with cancel-safe timers integrated into scroll-memory lifecycle paths. New browser regression coverage was added for the button behavior and compiles under the Playwright lane.
- residual risk: Live execution of the new browser regression is currently blocked by an unrelated in-progress server compile break in the worktree (`parseRequireUsageBudget` call signature mismatch).
- next suggested todo: Update the app version number and ensure the displayed version string is sourced consistently.

### 2026-03-27 23:14 America/New_York

- completed todo: Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. README now includes a runtime-pieces table plus interaction sequence that ties boot shell, client, worker, tunnel, server handlers, store, and provider layers together.
- residual risk: Several remaining Agent 4 workflow docs and repo-layout cleanup tasks are still open.
- next suggested todo: Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code.

### 2026-03-27 23:13 America/New_York

- completed todo: Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. README now documents concrete route classes, current admin route reality, static asset paths, and shell-vs-SPA routing semantics.
- residual risk: The dedicated runtime-components explainer and remaining Agent 4 docs hygiene tasks are still pending.
- next suggested todo: Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer.

### 2026-03-27 23:11 America/New_York

- completed todo: Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Quick start now uses current `gwc build` and managed `gwc examples ... cmd\server` lifecycle commands, seeded auth credentials, stub-provider mode, and focused browser verification tests.
- residual risk: Later README sections still contain older layout references that are tracked by separate Agent 4 documentation/layout cleanup todos.
- next suggested todo: Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior.

### 2026-03-27 23:10 America/New_York

- completed todo: Add webhook retry and delivery history plumbing backed by `webhook_deliveries`.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/sql/store/list_webhook_deliveries_pending_retry.sql`, `examples/server/ai-chat-wizard/sql/store/update_webhook_delivery_attempt.sql`, `examples/server/ai-chat-wizard/sql/store/update_webhook_delivery_delivered.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth)" -count=1`
- result: Passed. Webhook delivery history now supports pending-retry lookup plus explicit failed-attempt and delivered-state updates.
- residual risk: Retry plumbing is wired at store/query level, but background-job orchestration for automatic retries is still pending.
- next suggested todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.

### 2026-03-27 23:09 America/New_York

- completed todo: Trace and fix the mismatch where the model select shows `GPT-5.4 - Best` while message bubbles report `GPT-5.4 mini`.
- files changed: `examples/server/ai-chat-wizard/client/app/stream.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. Reply finalization now normalizes and adopts the assistant-reported model ID, then persists that model selection so the toolbar picker stays aligned with assistant message model metadata after first-send completion.
- residual risk: The route-smoke and happy-path suites validate send/stream completion and zero runtime errors, but there is still no dedicated assertion that compares the exact visible picker label text against the assistant metadata row text.
- next suggested todo: Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.

### 2026-03-27 23:07 America/New_York

- completed todo: Wire store/query funcs for onboarding templates, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_growth_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_onboarding_template.sql`, `examples/server/ai-chat-wizard/sql/store/list_onboarding_templates.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_user_activation_milestone.sql`, `examples/server/ai-chat-wizard/sql/store/list_user_activation_milestones.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_saved_workflow.sql`, `examples/server/ai-chat-wizard/sql/store/list_saved_workflows.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_prompt_library_item.sql`, `examples/server/ai-chat-wizard/sql/store/list_prompt_library_items.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_weekly_value_summary.sql`, `examples/server/ai-chat-wizard/sql/store/list_weekly_value_summaries.sql`, `examples/server/ai-chat-wizard/sql/store/create_product_analytics_event.sql`, `examples/server/ai-chat-wizard/sql/store/list_product_analytics_events.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_experiment_assignment.sql`, `examples/server/ai-chat-wizard/sql/store/list_experiment_assignments.sql`, `examples/server/ai-chat-wizard/sql/store/create_subscription_churn_feedback.sql`, `examples/server/ai-chat-wizard/sql/store/list_subscription_churn_feedback.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|Auth)" -count=1`
- result: Passed. Typed store/query wiring now covers onboarding, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
- residual risk: These tables are now wired at store/query level with lifecycle coverage, but dedicated admin and `su` RPC CRUD surfaces are still pending.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-27 23:06 America/New_York

- completed todo: Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.
- files changed: `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/server/ai-chat-wizard/server/app/benchmark_sla_test.go`, `examples/server/ai-chat-wizard/server/app/benchmark_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "Test(RequireUserEntitlement|SendAndSpeechNegativeBranches|SendStreamsThoughtsAndPersistsConversation|SendPersistsUsageTraceMetadata|SendRejectsConversationOwnedByAnotherUser|SendRejectsMissingAuthenticatedUserBeforeProviderWork)$" -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "TestSendSLASweep$" -count=1`
- result: Passed. Entitlement checks now deny missing effective access by default while preserving existing Send-path error boundaries and benchmark/test expectations via explicit plan seeding.
- residual risk: Quota/rate/concurrency enforcement is still pending and remains the final open Agent 2 item.
- next suggested todo: Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.

### 2026-03-27 23:06 America/New_York

- completed todo: Refresh `README.md` so it reads like a polished entry point for example 100, preserving the existing ASCII architecture diagram and enhancing it to better represent the current server, client, worker, routing, and gRPC flow.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. README now has a cleaner product entry, an updated architecture diagram aligned to `/socket` + worker/runtime behavior, and an explicit current data-flow summary.
- residual risk: Quick-start and route/runtime explainer sections are still pending as separate Agent 4 doc todos.
- next suggested todo: Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly.

### 2026-03-27 23:04 America/New_York

- completed todo: Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Admin operational workflow documentation now defines actor intent, dependency tables, current-vs-target RPC surfaces, and guardrails for destructive changes.
- residual risk: Most workflow mutation RPCs described here are still intentionally pending and remain tracked in Agent 2 and Agent 3 todos.
- next suggested todo: Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.

### 2026-03-27 23:03 America/New_York

- completed todo: Add browser-level smoke coverage for the pricing, auth, and dashboard routes.
- files changed: `test/playwrightgo/examples/example100_route_smoke_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100RouteSmokePricingAuthDashboard -v`
- result: Passed. New route smoke validates `/pricing#faq`, unauthenticated `/app` auth form rendering, and authenticated `/app/settings?panel=settings-profile` in one browser run with `status=200` and zero console/page errors.
- residual risk: This smoke confirms route availability and core shell mounts, but it does not yet assert model-label consistency or scroll-control behavior.
- next suggested todo: Trace and fix the model-label mismatch where the picker shows `GPT-5.4 � Best` while message bubbles show `GPT-5.4 mini`.

### 2026-03-27 23:02 America/New_York

- completed todo: Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Admin dashboard journey documentation now captures route entry, current superuser gating, data-slice dependencies, and operator action checkpoints.
- residual risk: Admin operational workflow stories (disable/restore/suspend/billing/support/incident) still need explicit product-flow documentation in Agent 4.
- next suggested todo: Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control.

### 2026-03-27 23:01 America/New_York

- completed todo: Document the visit-to-first-chat flow in product terms, including which routes, panels, defaults, and backend calls participate at each stage.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Visit-to-first-chat flow documentation now exists with concrete routes, defaults, and RPC dependencies, including token-cost trace fields.
- residual risk: Admin dashboard journey and admin operational workflow documentation are still pending under Agent 4.
- next suggested todo: Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions.

### 2026-03-27 23:00 America/New_York

- completed todo: Verify the authenticated happy path end-to-end (login, create thread, send message, stream reply, refresh, reopen thread).
- files changed: `test/playwrightgo/examples/example100_authenticated_happy_path_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`
- result: Passed. New browser test seeds a temporary DB, authenticates as `demo@example.com`, creates a new chat thread, sends and streams a reply, reloads while preserving `/app/thread/:publicID`, and reopens the same thread from the sidebar with prompt text intact and zero console/page errors.
- residual risk: The flow still logs route-resolution warnings right after first-send (`reply completed before conversation route resolved`), so route-sync diagnostics remain noisy even when the happy path succeeds.
- next suggested todo: Add browser-level smoke coverage for pricing, auth, and dashboard routes.

### 2026-03-27 22:59 America/New_York

- completed todo: Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|Auth)" -count=1`
- result: Passed. The superuser control-plane snapshot now includes typed operational rows for auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox rows, and background jobs.
- residual risk: Snapshot coverage is now extended, but dedicated typed mutating `su` CRUD RPCs for pricing and reliability control tables are still pending.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-27 22:58 America/New_York

- completed todo: Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/auth_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AuthManager(SignupLoginAndTokenRoundTrip|NegativePaths|PersistsSessionAndRevocation|TokenKidRotationPolicy)|ChatServerAuthRPCs)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAuthSessionLifecycle|StoreSuperuserControlPlaneLifecycle)$" -count=1`
- result: Passed. JWT validation now enforces `jti` and session-version checks while using `kid`-aware signing and verification with rotation-key fallback for legacy kid-less tokens.
- residual risk: Deny-by-default entitlement enforcement and usage-budget quota enforcement are still open and tracked in the next Agent 2 todos.
- next suggested todo: Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.

### 2026-03-27 22:54 America/New_York

- completed todo: Add typed store/query funcs for the new operational tables: auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/store_auth.go`, `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_workspace_invitation.sql`, `examples/server/ai-chat-wizard/sql/store/list_workspace_invitations.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_webhook_delivery.sql`, `examples/server/ai-chat-wizard/sql/store/list_webhook_deliveries.sql`, `examples/server/ai-chat-wizard/sql/store/create_support_ticket_message.sql`, `examples/server/ai-chat-wizard/sql/store/list_support_ticket_messages.sql`, `examples/server/ai-chat-wizard/sql/store/create_incident_update.sql`, `examples/server/ai-chat-wizard/sql/store/list_incident_updates.sql`, `examples/server/ai-chat-wizard/sql/store/create_notification_outbox.sql`, `examples/server/ai-chat-wizard/sql/store/list_notification_outbox.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_background_job.sql`, `examples/server/ai-chat-wizard/sql/store/list_background_jobs.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_auth_session.sql`, `examples/server/ai-chat-wizard/sql/store/get_auth_session_by_session_id.sql`, `examples/server/ai-chat-wizard/sql/store/list_auth_sessions.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run TestGetSuperuserControlPlaneReturnsSnapshot -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run TestAuth -count=1`
- result: Passed. The new operational-table store/query surfaces are wired and covered by focused lifecycle, superuser snapshot, and auth tests.
- residual risk: These tables are now typed in store/query layers, but dedicated `su` RPC surfaces for them are still pending.
- next suggested todo: Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.

### 2026-03-27 22:53 America/New_York

- completed todo: Persist and enforce server-side auth sessions using `auth_sessions` and `auth_token_versions`.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/store_auth.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/auth_test.go`, `examples/server/ai-chat-wizard/server/app/store_test.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_auth_session.sql`, `examples/server/ai-chat-wizard/sql/store/list_auth_sessions.sql`, `examples/server/ai-chat-wizard/sql/store/get_auth_session_by_session_id.sql`, `examples/server/ai-chat-wizard/sql/store/get_auth_token_version.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_auth_token_version.sql`, `examples/server/ai-chat-wizard/sql/store/touch_auth_session_last_seen.sql`, `examples/server/ai-chat-wizard/sql/store/revoke_auth_session.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAuthSessionLifecycle|AuthManagerPersistsSessionAndRevocation|ChatServerAuthRPCs|StoreSuperuserControlPlaneLifecycle)$" -count=1`
- result: Passed. Auth tokens are now backed by durable server sessions and token-version rows, and revoked or expired sessions are denied.
- residual risk: Key rotation (`kid`) and explicit per-token `jti` lifecycle policy are still pending and tracked by the next Agent 2 todo.
- next suggested todo: Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.

### 2026-03-27 22:51 America/New_York

- completed todo: Add a focused end-to-end regression for server start, WASM shell boot, and background worker boot.
- files changed: `test/playwrightgo/examples/example100_startup_boot_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100StartupBoot -v`
- result: Passed. New browser regression confirms `status=200`, boot-shell removal after mount, `/app/chat.wasm` + `/worker/background-worker.wasm` responses, `grpc ready`, `worker ready`, and `background render worker pool ready` startup logs with zero console/page errors.
- residual risk: This checkpoint validates startup/runtime boot only; authenticated message-send flows and route-specific UX paths are still covered by separate todos.
- next suggested todo: Verify the authenticated happy path end-to-end (login, create thread, send, stream, refresh, reopen).

### 2026-03-27 22:43 America/New_York

- completed todo: Run live browser startup smoke against `http://127.0.0.1:8095/` and capture startup console/runtime errors.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./bin/example100_startup_smoke.go` (temporary local helper removed after execution)
- result: Passed. Startup route `/` returned `200` with page title `RelayDesk � AI Chat Workspace`, `console_error_count=0`, and `page_error_count=0`.
- residual risk: This checkpoint covers only initial load at `/`; route-specific startup regressions (pricing/auth/dashboard) still require dedicated browser assertions.
- next suggested todo: Add a focused end-to-end regression that proves server start, WASM shell boot, and background worker boot.

### 2026-03-27 20:30 America/New_York

- completed todo: Repair server policy/model regression paths so `server/app` tests pass again.
- files changed: `examples/server/ai-chat-wizard/server/app/server_tool_policy.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|GetSelectedModelRepairsUnsupportedPreference|GetServerToolPolicyReadsWhitelistFromSiteConfig|ProviderStubRuntimeSupportsCrossProviderSelection)"`; `go test ./examples/server/ai-chat-wizard/server/app`; `go test ./examples/server/ai-chat-wizard/server/...`
- result: Passed. Whitelist policy JSON now parses correctly and model-selection regression coverage is green.
- residual risk: Entitlement deny-by-default, usage-budget quotas, and token revocation/key-rotation remain intentionally open TODOs and are not implemented in this checkpoint.
- next suggested todo: Implement deny-by-default entitlement gating once guaranteed entitlement bootstrap rows exist for all active users.

### 2026-03-27 20:09 America/New_York

- completed todo: Gate admin analytics and diagnostics RPCs behind superuser authorization.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/log_tail.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/server/app/log_tail_test.go`, `examples/server/ai-chat-wizard/server/app/testkit_test.go`, `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetLogTail)"`
- result: Blocked by a pre-existing compile failure in `server.go` (`undefined: parseSelectedModel`), unrelated to this authz-gating patch.
- residual risk: Authorization behavior is updated in code, but test confirmation is pending until the pre-existing server compile issue is resolved.
- next suggested todo: Add auth-secret startup validation and entitlement gate seams with TODO stubs for quota enforcement.

### 2026-03-27 20:09 America/New_York

- completed todo: Add production auth-secret validation and wire entitlement budget gate seams for `Send`.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetLogTail)"`
- result: Blocked by the same pre-existing compile failure in `server.go` (`undefined: parseSelectedModel`) before targeted test execution.
- residual risk: Entitlement and revocation seams are currently fail-open stubs for missing billing bootstrap and missing revocation backend; they are intentionally documented as TODO for follow-up implementation.
- next suggested todo: Implement deny-by-default entitlement behavior once user-to-plan bootstrap rows are guaranteed for all users.

### 2026-03-25 02:31 America/New_York

- completed todo: Define an AI provider companion package pattern for LLM-backed GWC applications.
- files changed: `docs/ECOSYSTEM.md`, `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The documented companion-package boundary matches the live catalog RPC path and the current wasm client surface.
- residual risk: The ecosystem guidance now captures the package boundary, but only RelayDesk currently validates the pattern, so promotion beyond `Experimental` would still require a second production-shaped app.
- next suggested todo: None in the current example-100 provider-switching slice.

### 2026-03-25 02:28 America/New_York

- completed todo: Define the SQL-backed model catalog pattern for runtime provider and model discovery.
- files changed: `examples/server/ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`
- result: Passed. The documented SQL-backed catalog path matches the live startup and RPC behavior exercised by the server tests.
- residual risk: The README now defines the recommended bootstrap payload and freshness policy, but RelayDesk still relies on authenticated RPC revalidation rather than shipping the initial catalog through SSR bootstrap today.
- next suggested todo: Define an AI provider companion package pattern for LLM-backed GWC applications.

### 2026-03-25 02:14 America/New_York

- completed todo: Promote RelayDesk as the reference implementation for runtime AI provider switching.
- files changed: `examples/server/ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `npx playwright test --config=playwright.chat-wizard.config.ts --grep "provider and model selection sync across open tabs"`
- result: Passed. The example README now explicitly positions RelayDesk as the runtime provider-switching reference app and points maintainers to the focused browser regression that proves the shipped flow.
- residual risk: The reference example now documents the shipped switching path clearly, but capability-aware filtering beyond provider membership is still not implemented in the UI.
- next suggested todo: Add capability-aware model filtering and picker messaging so RelayDesk can demonstrate why a provider or model disappears when a workflow requires a missing capability.

### 2026-03-25 01:18 America/New_York

- completed todo: Keep provider/model/intelligence selection stable across new chats and reconnects.
- files changed: `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/helpers_wasm_test.go`, `examples/server/ai-chat-wizard/client/app/model_preferences.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestGetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|TestRPCFallbacksWhenStoreOrProvidersAreUnavailable"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go build ./examples/server/ai-chat-wizard/client/...`
- result: Passed. New-chat and reconnect flows now recover and persist provider/model state instead of leaving the picker blank.
- residual risk: This is covered by unit tests and compile checks, but there is still no browser-level end-to-end regression that exercises the full picker flow through a real page reload.
- next suggested todo: Add a Playwright regression that selects a non-default provider/model, reloads, starts a new chat, and verifies the same provider/model remains selected with a non-blank intelligence mode.

### 2026-03-24 22:12 America/New_York

- completed todo: Prevent newly created threads from being cleared when the first assistant response finishes.
- files changed: `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run TestShouldResetDraftForRootRoute`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The regression guard behaves correctly and the js/wasm app package still compiles.
- residual risk: This flow still lacks a browser-level end-to-end regression that drives a real send on a fresh draft thread.
- next suggested todo: Add an example-100 integration regression that seeds a fake provider response and verifies the browser stays on `/thread/:publicID` after the first reply.

### 2026-03-24 22:14 America/New_York

- completed todo: Add diagnostics for unresolved thread-route state after a fresh reply.
- files changed: `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/stream.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run 'TestShould(ResetDraftForRootRoute|WarnPendingRootRoute)$'`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The warning guard condition is locked in and the js/wasm app package still compiles.
- residual risk: The warnings improve diagnosis but they do not replace a browser-level regression that exercises a real first-message send path.
- next suggested todo: Add an end-to-end regression with a fake provider that verifies the app stays in the new thread after the first streamed reply and asserts the warning does not appear in the healthy path.

### 2026-03-24 22:19 America/New_York

- completed todo: Flatten the provider, model, and intelligence controls so they use width more efficiently.
- files changed: `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The example-100 client app still compiles for js/wasm after the control-bar layout change.
- residual risk: This is a visual adjustment only; there is still no browser-level layout regression covering narrow widths and the inline control row.
- next suggested todo: Add a Playwright visual/layout smoke for the compact control bar at desktop and mobile widths.

### 2026-03-24 22:24 America/New_York

- completed todo: Add hover and press animations to the toolbar selects.
- files changed: `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/styles.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The client app still compiles for js/wasm after the animated select styling pass.
- residual risk: Native option hover and press styling remain browser-dependent, so the select element motion is reliable but option-row animation fidelity will vary by platform.
- next suggested todo: Add a browser smoke that exercises the animated control bar in Chromium and confirms hover, focus, and press states remain readable.

### 2026-03-24 22:31 America/New_York

- completed todo: Add a floating down-arrow when more thread content is available below.
- files changed: `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/app_shell.go`, `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/scroll_memory.go`, `examples/server/ai-chat-wizard/client/app/scroll_visibility.go`, `examples/server/ai-chat-wizard/client/app/scroll_visibility_test.go`, `examples/server/ai-chat-wizard/client/app/thread.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run TestHasScrollSpaceBelow`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The visibility helper is locked in and the js/wasm client app still compiles with the floating jump-to-bottom control.
- residual risk: Browser-level placement and overlap behavior still need a visual smoke pass, especially with split canvas mode and long threads.
- next suggested todo: Add a Playwright scroll smoke that verifies the button appears when the user scrolls up and jumps back to the latest message when clicked.

### 2026-03-24 22:37 America/New_York

- completed todo: Rebrand the example-100 product surface away from the framework name.
- files changed: `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/server/app/bootstrap.go`, `examples/tests/100-ai-chat-wizard.spec.ts`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The visible app brand and server-rendered page title now use the new product name, and the client app still compiles for js/wasm.
- residual risk: This updates the primary visible branding, but supporting copy such as the empty-state headline still reads like an experiment rather than a polished product surface.
- next suggested todo: Refresh the empty-state and onboarding copy so the rest of the home screen matches the new product branding.

### 2026-03-28 10:57 America/New_York

- completed todo: Add typed external-auth audit events and queryable auth records for OIDC start/callback/link policy paths.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_external_audit.go`, `examples/server/ai-chat-wizard/server/app/auth_google_oidc.go`, `examples/server/ai-chat-wizard/server/app/auth_external_audit_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -count=1 -run "^(TestExternalAuthAuditSuccessLifecycle|TestExternalAuthAuditPolicyDeniedLifecycle|TestExternalAuthAuditIdentityUnlinkedLifecycle|TestHandleOIDCProviderCallbackEnforcesWorkspaceLinkPolicy|TestResolveExternalIdentityLinkDecisionEnforcesPolicy)$"`
- result: Passed. External-auth lifecycle events now persist as typed audit rows with a valid actor path, and the query helper returns expected records for success, policy-denied, and identity-unlinked flows.
- residual risk: Callback failures that occur before OIDC state lookup (or without a workspace scope) can still skip audit-row insertion when no valid actor can be resolved.
- next suggested todo: Add one typed customer-safe error contract for chat/auth/settings/dashboard failures with friendly message + stable support/request ID correlation.

### 2026-03-28 16:20 America/New_York

- completed todo: Group the current example-100 product, admin, routing, schema, and documentation changes into a repo checkpoint commit.
- files changed: `examples/server/ai-chat-wizard/CHANGELOG.md`, `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/docs/*`, `examples/server/ai-chat-wizard/client/app/*`, `examples/server/ai-chat-wizard/server/app/*`, `examples/server/ai-chat-wizard/sql/store/*`, `examples/server/ai-chat-wizard/proto/*`, `test/playwrightgo/examples/*`, `html/*`, `examples/static/css/tailwind.css`
- validation run: `git diff --stat -- . ':(exclude)third_party/GoGRPCBridge'`; direct readback of `examples/server/ai-chat-wizard/CHANGELOG.md` and `examples/server/ai-chat-wizard/TODO.md`
- result: Captured the current example-100 worktree as one coherent checkpoint spanning admin control-plane expansion, dashboard groundwork, routing and bootstrap fixes, schema/store growth, Playwright coverage growth, and backlog/documentation refinement including the usage-based pricing direction.
- residual risk: The superproject commit does not include the dirty `third_party/GoGRPCBridge` submodule state, and loose root transcript artifacts were intentionally left uncommitted.
- next suggested todo: Resume the example-100 backlog from `TODO.md` with one focused implementation slice at a time, starting with a typed dashboard or billing surface rather than another broad backlog pass.
 -count=1`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. `GetSession` now returns typed auth-bootstrap metadata (`session_status`, session/token identifiers, expiry, and typed role summary), server role scope is resolved into `user`/`workspace_admin`/`superuser`, and the client auth session checker consumes typed status/expiry fields to drive safe fallback decisions.
- residual risk: Role summary is still bootstrap metadata only; dedicated role-aware dashboard IA and post-login route policy wiring are tracked as separate follow-up todos.
- next suggested todo: Enforce persisted token lifecycles for signup verification, password reset, and update-password flows using the new token tables instead of ad hoc state.

### 2026-03-27 23:56 America/New_York

- completed todo: Normalize file placement so docs stay at the example root, scripts stay under `scripts/`, runtime outputs stay under runtime folders, and generated artifacts stay under `bin/` or another clearly non-source location.
- files changed: `examples/server/ai-chat-wizard/bin/server/chat-wizard-server.exe` (moved), `examples/server/ai-chat-wizard/bin/runtime/legacy-artifacts/chat-wizard-server.exe~` (moved), `examples/server/ai-chat-wizard/bin/runtime/legacy-artifacts/main.wasm` (moved), `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Get-ChildItem examples/server/ai-chat-wizard/bin -File` confirmed no loose root files remain (`bin_root_files=none`).
- result: Passed. Example-100 file placement is now cleaner: docs remain at root, runtime state/logs are under `bin/runtime`, and generated binaries/artifacts are under `bin/server` or `bin/runtime/legacy-artifacts`.
- residual risk: Legacy artifact relocation can break any external scripts outside this repo that hardcode old `bin/` root paths.
- next suggested todo: Move non-first-class docs into a dedicated `docs/` folder while keeping the primary entry docs at the example root, then update all links and cross-references after the move.

### 2026-03-27 23:55 America/New_York

- completed todo: Rename inconsistent or low-signal files so the example reads professionally to a new maintainer, then update all references in scripts, docs, tests, and server/client code.
- files changed: `examples/server/ai-chat-wizard/PERFORMANCE.md` (renamed from `BENCHMARKS.md`), `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`; `rg -n "BENCHMARKS\.md" .\examples\server\ai-chat-wizard`
- result: Passed. The benchmark reference doc now has a clearer maintainer-facing name (`PERFORMANCE.md`), and in-repo references were updated to the new path.
- residual risk: External links/bookmarks outside this repo may still point to the old `BENCHMARKS.md` filename.
- next suggested todo: Normalize file placement so docs stay at the example root, scripts stay under `scripts/`, runtime outputs stay under runtime folders, and generated artifacts stay under `bin/` or another clearly non-source location.

### 2026-03-27 23:53 America/New_York

- completed todo: Move any other loose runtime outputs out of the example root into a consistent runtime location such as `bin/runtime/`, `bin/logs/`, or another single clear convention.
- files changed: `examples/server/ai-chat-wizard/server/app/logging.go`, `examples/server/ai-chat-wizard/server/app/coverage_gap_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(ServerLoggerCreatesFileSink|ServerLoggerFailureBranches|GetLogTailReturnsLatestServerLines|GetLogTailSupportsAllSourcesAndFilter|GetLogTailRejectsInvalidSource|GetLogTailRequiresSuperuserAccess)$" -count=1`
- result: Passed. Runtime server/client log sinks now default to `bin/runtime/logs` instead of root-level `log/`, and logger failure-branch coverage was updated for nested log-directory setup.
- residual risk: Any manually started server process with custom shell redirection can still create ad hoc root-level log files outside these managed defaults.
- next suggested todo: Rename inconsistent or low-signal files so the example reads professionally to a new maintainer, then update all references in scripts, docs, tests, and server/client code.

### 2026-03-27 23:53 America/New_York

- completed todo: Define session and auth-expiry behavior for long-lived dashboard tabs so expired admin sessions fall back safely without leaving stale privileged UI visible.
- files changed: `examples/server/ai-chat-wizard/client/app/auth.go`, `examples/server/ai-chat-wizard/client/app/auth_wasm_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. Auth session handling now runs periodic authenticated session checks (2-minute interval) and fail-closes into the login shell with session-expired messaging when long-lived tabs carry expired/revoked/missing sessions, preventing stale privileged UI from remaining visible.
- residual risk: Session/role/expiry data is still inferred from current auth RPCs and token claims; a dedicated typed bootstrap contract with explicit expiry metadata is still pending.
- next suggested todo: Add a typed auth bootstrap contract that returns session status, role summary, and expiry information needed for safe app-shell and dashboard entry decisions.

### 2026-03-27 23:52 America/New_York

- completed todo: Add one browser-level admin journey regression that covers homepage load, login, role resolution, admin-entry visibility, dashboard-home load, slice navigation, and back/refresh behavior.
- files changed: `test/playwrightgo/examples/example100_admin_journey_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AdminJourneyRegression -count=1`
- result: Passed. Added a browser-admin flow regression that starts at `/`, signs in as `admin@example.com`, validates settings/dashboard entry and panel routing, confirms slice route navigation with back/refresh stability, and verifies role-scoped `GetAdminDashboard`, `ListAdminUsers`, and `ListAdminConversations` access using the authenticated browser token.
- residual risk: Example 100 still has no dedicated dashboard browser IA, so this regression uses settings-route navigation plus admin RPC checks as the closest journey proxy.
- next suggested todo: Add runtime diagnostics for the admin dashboard journey so role resolution, admin route entry, dashboard bootstrap, slice fetches, and unauthorized transitions emit actionable logs.

### 2026-03-27 23:51 America/New_York

- completed todo: Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.
- files changed: `examples/server/ai-chat-wizard/bin/runtime/logs/server.stderr.log`, `examples/server/ai-chat-wizard/bin/runtime/logs/server.stdout.log`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Test-Path` checks confirmed root log files are absent and moved files exist in `bin/runtime/logs`; `rg -n "server\.stderr\.log|server\.stdout\.log"` over `cmd/server`, `scripts`, `server`, and `tools/gwc/examples_managed.go` returned no active path references.
- result: Passed. Runtime stderr/stdout artifacts were relocated out of the example root and no current launcher/server paths in this repo still target the old root filenames.
- residual risk: Historical or external shell aliases outside this repo could still redirect process output to root-level files.
- next suggested todo: Move any other loose runtime outputs out of the example root into a consistent runtime location such as `bin/runtime/`, `bin/logs/`, or another single clear convention.

### 2026-03-27 23:49 America/New_York

- completed todo: Keep `SCHEMA_TABLES.md` synchronized with the control-plane, auth, onboarding, and admin workflow stories so the docs match the real table usage.
- files changed: `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Synced classification to real wiring by moving `password_reset_tokens` and `email_verification_tokens` to `Persistence-only`, expanded wiring snapshot bullets for auth/admin/billing seams, and added a story-alignment section that maps active flow stories to concrete table usage.
- residual risk: This keeps docs accurate for current implementation, but pending admin mutation RPC rollout may shift classification for several control-plane tables again.
- next suggested todo: Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.

### 2026-03-27 23:48 America/New_York

- completed todo: Add server-side checks for admin deep links, slice RPCs, and detail views so unauthorized dashboard routes fail closed and redirect or render a sane denied state.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/runtime_helpers_additional_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(ChatShellRoutingHelpers|ChatShellHandler|AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$' -count=1`
- result: Passed. Unauthorized admin deep links now fail closed at HTTP shell entry and redirect to `/app`, while authorized admin users still receive the shell. Dashboard slice RPCs now use slice-specific admin scope gates and emit structured denial logs.
- residual risk: Dedicated admin detail-view RPCs are not implemented yet, so detail-level allow/deny checks are currently enforced via scoped list/filter behavior plus the new route guard coverage.
- next suggested todo: Define session and auth-expiry behavior for long-lived dashboard tabs so expired admin sessions fall back safely without leaving stale privileged UI visible.

### 2026-03-27 23:46 America/New_York

- completed todo: Update `DESIGN.md` so it reflects the current intended product surface, dashboard direction, and any material UI or IA changes made since the original design pass.
- files changed: `examples/server/ai-chat-wizard/DESIGN.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Replaced the stale visual-redesign spec with a current design-intent document aligned to the shipped route model, role-split dashboard direction, state/feedback rules, and IA constraints.
- residual risk: The design intent now matches current direction, but repo-layout cleanup tasks are still open and can change path references again.
- next suggested todo: Keep `SCHEMA_TABLES.md` synchronized with the control-plane, auth, onboarding, and admin workflow stories so the docs match the real table usage.

### 2026-03-27 23:44 America/New_York

- completed todo: Add focused regression coverage for empty-state boot, first-thread creation, scroll anchoring during first reply, and reopen-after-refresh behavior.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -count=1`
- result: Passed. `TestExample100AuthenticatedHappyPath` already exercises empty-state boot, first-thread route creation after send, first-reply stream anchoring, and prompt/route persistence after reload and sidebar reopen.
- residual risk: This checkpoint covers one focused happy path; admin journey and admin mutation browser regressions remain open.
- next suggested todo: Add one browser-level admin journey regression that covers homepage load, login, role resolution, admin-entry visibility, dashboard-home load, slice navigation, and back/refresh behavior.

### 2026-03-27 23:43 America/New_York

- completed todo: Add one concise docs map to the example that tells maintainers which file to read for setup, architecture, smoke testing, schema reference, design intent, and changelog history.
- files changed: `examples/server/ai-chat-wizard/DOCS_MAP.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added a concise docs map covering setup, flow docs, smoke/testing, bug templates, schema, design, operator runbook, changelog, and backlog.
- residual risk: Design refresh and repo-layout/log-location cleanup tasks remain open in Agent 4.
- next suggested todo: Create a dedicated runtime logs folder under example 100, move `server.stderr.log` and `server.stdout.log` into it, and update any scripts or server startup paths that still write logs to the example root.

### 2026-03-27 23:42 America/New_York

- completed todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.
- files changed: `examples/server/ai-chat-wizard/BUG_REPORT_TEMPLATES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added a dedicated bug-report template doc with route, first-chat, and admin-dashboard templates that require reproducible environment/state/evidence fields.
- residual risk: Remaining Agent 4 docs cleanup tasks (design/schema sync/docs map/layout hygiene) are still open.
- next suggested todo: Add one concise docs map to the example that tells maintainers which file to read for setup, architecture, smoke testing, schema reference, design intent, and changelog history.

### 2026-03-27 23:41 America/New_York

- completed todo: Add runtime diagnostics for the visit-to-first-chat path so boot, gRPC connect, worker readiness, model-catalog load, conversation bootstrap, and first-send failures log one actionable warning or error each.
- files changed: `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/model_preferences.go`, `examples/server/ai-chat-wizard/client/app/conversations.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`; `go test -tags playwrightgo ./test/playwrightgo/examples -run "TestExample100(VisitToFirstChatRegression|ScrollToBottomButton)$" -c`
- result: Added explicit startup/boot-shell, model-catalog bootstrap, and conversation-bootstrap diagnostics while keeping existing gRPC, worker readiness, and send/stream failure logs intact.
- residual risk: Live browser execution remains intermittently blocked by unrelated in-flight server compile edits in this shared worktree.
- next suggested todo: Add focused regression coverage for empty-state boot, first-thread creation, scroll anchoring during first reply, and reopen-after-refresh behavior.

### 2026-03-27 23:41 America/New_York

- completed todo: Document the disable-vs-suspend semantics so maintainers and operators know exactly which downstream capabilities are supposed to turn off in each case.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now has an explicit disable/restore vs suspend/restore semantics matrix with scoped impact and guardrails.
- residual risk: Bug-report templates and remaining Agent 4 docs hygiene tasks are still open.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:40 America/New_York

- completed todo: Document the admin search/filter/pagination conventions so operators know how large lists, saved context, and back-navigation are expected to behave.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now documents stable admin list conventions across search, filter, and pagination behavior with route/back-navigation expectations.
- residual risk: Disable-vs-suspend semantics and bug-report templates remain open in Agent 4.
- next suggested todo: Document the disable-vs-suspend semantics so maintainers and operators know exactly which downstream capabilities are supposed to turn off in each case.

### 2026-03-27 23:38 America/New_York

- completed todo: Add one browser-level visit-to-first-chat regression that covers landing load, pricing/auth navigation, login/signup handoff, authenticated shell boot, first send, streamed reply, and canonical thread-route normalization.
- files changed: `test/playwrightgo/examples/example100_visit_first_chat_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100VisitToFirstChatRegression -v` (blocked by unrelated in-progress server compile failure in worktree: `admin_dashboard.go` missing `strings` import); `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100VisitToFirstChatRegression -c`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: New browser regression coverage now exists for the full visitor-to-first-chat route/auth/send flow with canonical thread-route assertion and sidebar-reopen fallback when immediate route sync is deferred.
- residual risk: Runtime execution of the new Playwright flow is currently blocked by unrelated server compile instability from concurrent edits in this worktree.
- next suggested todo: Add runtime diagnostics for the visit-to-first-chat path so boot, gRPC connect, worker readiness, model-catalog load, conversation bootstrap, and first-send failures log one actionable warning or error each.

### 2026-03-28 00:38 America/New_York

- completed todo: Enforce dashboard access by resolved role so normal users cannot reach admin surfaces, workspace admins get workspace-scoped data, and superusers get platform-scoped data.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_scope.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/sql/store/list_workspace_memberships_by_user.sql`, `examples/server/ai-chat-wizard/sql/store/list_workspace_memberships_by_workspace.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(AdminDashboardRPCs|AdminDashboardRPCsWorkspaceAdminScope)$' -count=1`
- result: Passed. Admin dashboard RPCs now resolve caller role scope before query execution: normal authenticated users are denied, superusers keep platform-wide access, and workspace admins receive workspace-scoped users/usage/conversations derived from active membership boundaries.
- residual risk: Workspace-scoped dashboard aggregates currently derive from filtered recent/admin list queries rather than dedicated workspace-optimized SQL rollups, so very large datasets may need follow-up query specialization.
- next suggested todo: Add server-side checks for admin deep links, slice RPCs, and detail views so unauthorized dashboard routes fail closed and redirect or render a sane denied state.

### 2026-03-28 00:31 America/New_York

- completed todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.
- files changed: `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run 'Test(BuildSendAccessDeniedStatus|RequireUserEntitlement|RequireUsageBudget|SendRejectsMissingAuthenticatedUserBeforeProviderWork|SendUsesBillingPlanDefaultModel)$' -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run '^$' -bench 'Benchmark(BuildSendAccessDeniedStatus|RequireUsageBudget)$' -benchtime=200x`
- result: Passed. Send-path entitlement and usage-budget denials now emit a structured `send_access` decision shape with `decision`, `plan`, `action`, `reason`, and `first_paid_action=chat.send`, so soft-upgrade prompts are distinguishable from hard blocks while preserving allow-path behavior.
- residual risk: Soft-upgrade decisions currently surface as structured RPC errors and are not yet rendered as a dedicated upgrade UI treatment in the wasm client.
- next suggested todo: Enforce dashboard access by resolved role so normal users cannot reach admin surfaces, workspace admins get workspace-scoped data, and superusers get platform-scoped data.

### 2026-03-27 23:30 America/New_York

- completed todo: Update the app version number and ensure the displayed version string is sourced consistently.
- files changed: `examples/server/ai-chat-wizard/internal/buildinfo/version.go`, `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/client/app/sidebar.go`, `examples/server/ai-chat-wizard/server/app/bootstrap.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`; `go test ./examples/server/ai-chat-wizard/internal/buildinfo -count=1`; attempted browser rerun: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -v` (blocked by unrelated in-progress server compile failure in worktree: `server.go` undefined `parseBuildUserMemorySignature*` helpers)
- result: Passed for client + shared source wiring. Version is now `v2026.03.27.2` from one shared build-info source consumed by both the sidebar version badge and the server boot-shell version label.
- residual risk: Full browser-path reruns are currently unstable due concurrent server-side compile edits unrelated to this version-source change.
- next suggested todo: Add one browser-level visit-to-first-chat regression that covers landing load, pricing/auth navigation, login/signup handoff, authenticated shell boot, first send, streamed reply, and canonical thread-route normalization.

### 2026-03-28 00:26 America/New_York

- completed todo: Add auth-state enforcement for the public-to-app transition so blocked, expired, revoked, or malformed sessions fail into a sane auth route instead of a half-booted app shell.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/auth.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run '^TestChatServerAuthRPCs$' -count=1`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test ./examples/server/ai-chat-wizard/client/app -run '^TestShouldRedirectUnauthenticatedRouteToLanding$' -count=1` (blocked on local runner: `%1 is not a valid Win32 application`)
- result: Passed for server auth/session regression and wasm client compile. `GetSession` now returns `Unauthenticated` for invalid/revoked/malformed metadata tokens, token-bearing requests no longer fall back to peer-bound auth state, auth bootstrap reports clearer session rejection failures, and resolved unauthenticated `/app...` routes are redirected back to landing.
- residual risk: Runtime coverage for the new js/wasm route-guard unit test is still blocked in this Windows shell because direct `GOOS=js GOARCH=wasm go test` execution could not launch the wasm test binary.
- next suggested todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.

### 2026-03-28 00:10 America/New_York

- completed todo: Expand the example-100 backlog and docs for chat capability planning, admin workflow gaps, testing stories, and repo cleanup guidance.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `Get-Content examples/server/ai-chat-wizard/TODO.md`; `Get-Content examples/server/ai-chat-wizard/CHANGELOG.md`
- result: Passed. The backlog now includes future chat capability stories (calendar/email hooks, web search, shareable chats, scheduled jobs, image upload, ask-with-docs, skills, workflows, code interpreter), clearer admin/operator workflow coverage, explicit bug-fix/testing stories, and concrete repo-cleanup/doc-refresh instructions.
- residual risk: These changes only improve planning and documentation shape; the newly added capability stories are not yet decomposed into implementation slices across backend, authz, runtime, and UI.
- next suggested todo: Choose which future chat capabilities should move from the planning section into the active agent backlog.

### 2026-03-27 23:30 America/New_York

- completed todo: Fix the memory extraction system so remembered items are extracted reliably, deduplicated sanely, editable, and consistent with what users see in the remembered-preferences UI.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/memory_helpers_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(CustomPromptAndMemoryHelperFunctions|ExtractAndStoreUserMemoriesBranches|ExtractAndStoreUserMemoriesReusesExistingKeys|ExtractAndStoreUserMemoriesLogsLifecycle|ChatServerUserMemoryRPCs|SendStreamsThoughtsAndPersistsConversation|SendAppliesDefaultSystemPromptOnNewConversation)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle|SendAppliesDefaultSystemPromptOnNewConversation|ExtractAndStoreUserMemoriesReusesExistingKeys)" -count=1`
- result: Passed. Extraction now deduplicates candidate memories by normalized signature, prefers the strongest duplicate candidate, reuses existing keys for matching signatures, and deduplicates injected memory prompt blocks so UI-visible memories and model-injected context stay consistent.
- residual risk: Existing duplicate rows already persisted with different signatures are not yet backfilled or merged by a dedicated migration job.
- next suggested todo: Keep `SCHEMA_TABLES.md` in sync as store funcs and RPCs are added for the new tables.

### 2026-03-27 23:38 America/New_York

- completed todo: Add an operator runbook for local start, seed, build-client, and superuser/admin verification flows.
- files changed: `examples/server/ai-chat-wizard/OPERATOR_RUNBOOK.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Added `OPERATOR_RUNBOOK.md` with concrete repo-root operator steps for env setup, client builds, DB seed, managed server lifecycle, user chat verification, and superuser/admin backend verification.
- residual risk: The runbook documents current admin verification through backend tests because a dedicated dashboard route-level UI flow is still pending.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:36 America/New_York

- completed todo: Document which schema tables are persistence-only vs fully wired into store, RPC, jobs, and UI.
- files changed: `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `SCHEMA_TABLES.md` now includes explicit wiring classes (`Store+RPC+Jobs+UI`, `Store+RPC+Jobs`, `Store+RPC`, `Store-only`, `Persistence-only`) with current table assignments.
- residual risk: Classification is a point-in-time snapshot and must be updated with each new store/RPC/job/UI wiring change to avoid drift.
- next suggested todo: Add an operator runbook for local start, seed, build-client, and superuser/admin verification flows.

### 2026-03-27 23:32 America/New_York

- completed todo: Keep `SCHEMA_TABLES.md` in sync as store funcs and RPCs are added for the new tables.
- files changed: `examples/server/ai-chat-wizard/SCHEMA_TABLES.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `SCHEMA_TABLES.md` now includes a wiring snapshot that maps newly wired operational/growth tables to current store funcs, `GetSuperuserControlPlane`, notification/job flow helpers, and user-memory extraction/runtime entry points.
- residual risk: The snapshot covers newly wired tables but does not yet classify every table as persistence-only vs store/RPC/jobs/UI wired.
- next suggested todo: Document which schema tables are persistence-only vs fully wired into store, RPC, jobs, and UI.

### 2026-03-27 23:26 America/New_York

- completed todo: Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/server_helpers_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/server/ai-chat-wizard/server/app/unit_additional_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(NormalizationAndPromptHelpers|RPCFallbacksWhenStoreOrProvidersAreUnavailable|MemoryRPCFallbacksWithoutStore|SendAppliesDefaultSystemPromptOnNewConversation|SendStreamsThoughtsAndPersistsConversation)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle|SendAppliesDefaultSystemPromptOnNewConversation)" -count=1`
- result: Passed. Default prompt fallback now resolves runtime template variables (`{{date}}`, `{{time}}`, `{{memories}}`) and is applied automatically on first-send/new-conversation paths when no user override is stored.
- residual risk: The default prompt is now enforced server-side; there is still no per-workspace default prompt policy or prompt-version audit trail.
- next suggested todo: Fix the memory extraction system so remembered items are extracted reliably, deduplicated sanely, editable, and consistent with what users see in the remembered-preferences UI.

### 2026-03-27 23:21 America/New_York

- completed todo: Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage.
- files changed: `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `MANUAL_SMOKE.md` now includes a journey-to-coverage matrix across browser specs, focused package tests, and manual smoke groups.
- residual risk: Bug-report templates and broader repo-layout documentation tasks remain open in Agent 4.
- next suggested todo: Add bug-report templates for route bugs, first-chat bugs, and admin-dashboard bugs so reproduction details are captured consistently.

### 2026-03-27 23:20 America/New_York

- completed todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.
- files changed: `examples/server/ai-chat-wizard/server/app/job_flows.go`, `examples/server/ai-chat-wizard/server/app/job_flows_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestHandleBackgroundJobsLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle|HandleBackgroundJobsLifecycle)" -count=1`
- result: Passed. Server-side job-flow helpers now enqueue and dispatch typed weekly-summary, dunning-retry, retention-purge, and health-score-refresh jobs with persisted running/completed/failed or retry-to-pending state transitions.
- residual risk: Dispatch remains callback-driven and synchronous in-process; a persistent scheduler loop and distributed worker coordination are still pending.
- next suggested todo: Define a system-default system prompt with runtime variable injection and ensure every new chat thread starts with that default prompt already applied.

### 2026-03-27 23:19 America/New_York

- completed todo: Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate; refresh `MANUAL_SMOKE.md` so manual verification covers landing routes, auth, first chat, settings, admin dashboard entry, and key operator actions.
- files changed: `examples/server/ai-chat-wizard/MANUAL_SMOKE.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `MANUAL_SMOKE.md` now defines release-gate checklist groups, concrete visitor/first-chat/admin flow steps, operator-action guidance, and optional focused automation commands.
- residual risk: Testing-story matrix and bug-report template docs are still open in Agent 4.
- next suggested todo: Define the testing story matrix that maps each major user and admin journey to browser tests, focused package tests, and manual smoke coverage.

### 2026-03-27 23:18 America/New_York

- completed todo: Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.
- files changed: `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/store.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/server/ai-chat-wizard/sql/store/sum_usage_tokens_since.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "Test(RequireUserEntitlement|RequireUsageBudget|SendStreamsThoughtsAndPersistsConversation|SendPersistsUsageTraceMetadata|SendUsesBillingPlanDefaultModel|SendRejectsConversationOwnedByAnotherUser|SendRejectsMissingAuthenticatedUserBeforeProviderWork)$" -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "^$" -bench "BenchmarkRequireUsageBudget$" -benchtime=200x`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -count=1`
- result: Passed. `parseRequireUsageBudget` now enforces monthly token limits from billing access control, applies per-user send-rate and concurrency limits, and returns a lease release function that `Send` defers for correct stream-lifetime concurrency tracking.
- residual risk: Rate and concurrency accounting is process-local in-memory state today; limits reset on server restart and are not yet distributed across multiple server instances.
- next suggested todo: Add first-send entitlement and quota decisions that distinguish allow, soft-upgrade prompt, and hard-block states, with clear billing-plan context for the first paid action.

### 2026-03-27 23:17 America/New_York

- completed todo: Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now defines an admin-dashboard incident workflow that captures role scope, entry path, slice-load failures, mutation-state evidence, and guard-safe validation steps.
- residual risk: Manual smoke matrix/test-story/bug-template docs are still open in Agent 4.
- next suggested todo: Define a manual smoke checklist for visitor, first-chat, admin dashboard, and admin mutation flows so bug fixes have a consistent release gate.

### 2026-03-27 23:16 America/New_York

- completed todo: Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now includes a first-chat incident workflow with explicit auth, model-bootstrap, send/stream, and route-normalization evidence capture steps.
- residual risk: Admin-dashboard bug-fix workflow documentation and several release-gate/checklist docs remain open.
- next suggested todo: Document the admin-dashboard bug-fix workflow in product terms, including role scope, dashboard entry, slice loading, and mutation-state debugging steps.

### 2026-03-27 23:15 America/New_York

- completed todo: Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. `FLOWS.md` now includes a concrete public-route incident workflow with route, hydration, router-state, and diagnostics capture steps before patching.
- residual risk: First-chat and admin-dashboard bug-fix workflow docs are still open and should mirror this level of capture specificity.
- next suggested todo: Document the first-chat bug-fix workflow in product terms, including what state to capture for auth bootstrap, model bootstrap, send flow, and route normalization issues.

### 2026-03-27 23:14 America/New_York

- completed todo: Add customer-facing notification flows backed by `notification_outbox`.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/notification_flow.go`, `examples/server/ai-chat-wizard/server/app/notification_flow_test.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/sql/store/list_notification_outbox_pending.sql`, `examples/server/ai-chat-wizard/sql/store/update_notification_outbox_status.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(DispatchNotificationOutboxPendingLifecycle|StoreSuperuserControlPlaneLifecycle)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth|DispatchNotificationOutboxPendingLifecycle)" -count=1`
- result: Passed. Pending notification rows now dispatch through a typed flow and transition to `sent`/`failed` status while future-scheduled rows remain pending.
- residual risk: Delivery is currently callback-driven dispatch plumbing; production channel adapters and scheduled job orchestration remain pending.
- next suggested todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.

### 2026-03-27 23:13 America/New_York

- completed todo: Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.
- files changed: `examples/server/ai-chat-wizard/client/app/scroll_memory.go`, `test/playwrightgo/examples/example100_scroll_to_bottom_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -v` (blocked by pre-existing server compile failure: `server.go:578 assignment mismatch: 2 variables but parseS.parseRequireUsageBudget returns 1 value`); `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100ScrollToBottomButton -c`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: The jump-to-bottom action now issues a smooth scroll followed by a timed settle snap to exact bottom, with cancel-safe timers integrated into scroll-memory lifecycle paths. New browser regression coverage was added for the button behavior and compiles under the Playwright lane.
- residual risk: Live execution of the new browser regression is currently blocked by an unrelated in-progress server compile break in the worktree (`parseRequireUsageBudget` call signature mismatch).
- next suggested todo: Update the app version number and ensure the displayed version string is sourced consistently.

### 2026-03-27 23:14 America/New_York

- completed todo: Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. README now includes a runtime-pieces table plus interaction sequence that ties boot shell, client, worker, tunnel, server handlers, store, and provider layers together.
- residual risk: Several remaining Agent 4 workflow docs and repo-layout cleanup tasks are still open.
- next suggested todo: Document the public-route bug-fix workflow in product terms, including how to capture route, hydration, and router-state failures before editing code.

### 2026-03-27 23:13 America/New_York

- completed todo: Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. README now documents concrete route classes, current admin route reality, static asset paths, and shell-vs-SPA routing semantics.
- residual risk: The dedicated runtime-components explainer and remaining Agent 4 docs hygiene tasks are still pending.
- next suggested todo: Add a README section that explains the main runtime pieces and how they interact: boot shell, WASM client, worker, gRPC tunnel, server handlers, store, and provider layer.

### 2026-03-27 23:11 America/New_York

- completed todo: Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Quick start now uses current `gwc build` and managed `gwc examples ... cmd\server` lifecycle commands, seeded auth credentials, stub-provider mode, and focused browser verification tests.
- residual risk: Later README sections still contain older layout references that are tracked by separate Agent 4 documentation/layout cleanup todos.
- next suggested todo: Add a README section that explains the current route model clearly: public landing routes, auth entry, app routes, settings routes, dashboard routes, and SPA vs server-shell behavior.

### 2026-03-27 23:10 America/New_York

- completed todo: Add webhook retry and delivery history plumbing backed by `webhook_deliveries`.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/sql/store/list_webhook_deliveries_pending_retry.sql`, `examples/server/ai-chat-wizard/sql/store/update_webhook_delivery_attempt.sql`, `examples/server/ai-chat-wizard/sql/store/update_webhook_delivery_delivered.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot|Auth)" -count=1`
- result: Passed. Webhook delivery history now supports pending-retry lookup plus explicit failed-attempt and delivered-state updates.
- residual risk: Retry plumbing is wired at store/query level, but background-job orchestration for automatic retries is still pending.
- next suggested todo: Add server-side job flows for weekly summaries, dunning retries, retention purges, and health-score refresh.

### 2026-03-27 23:09 America/New_York

- completed todo: Trace and fix the mismatch where the model select shows `GPT-5.4 - Best` while message bubbles report `GPT-5.4 mini`.
- files changed: `examples/server/ai-chat-wizard/client/app/stream.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`; `go run ./tools/gwc test -lane wasm -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard`
- result: Passed. Reply finalization now normalizes and adopts the assistant-reported model ID, then persists that model selection so the toolbar picker stays aligned with assistant message model metadata after first-send completion.
- residual risk: The route-smoke and happy-path suites validate send/stream completion and zero runtime errors, but there is still no dedicated assertion that compares the exact visible picker label text against the assistant metadata row text.
- next suggested todo: Fix the scroll-to-bottom action so it reliably lands at the bottom of the active chat thread.

### 2026-03-27 23:07 America/New_York

- completed todo: Wire store/query funcs for onboarding templates, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_growth_ops.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_onboarding_template.sql`, `examples/server/ai-chat-wizard/sql/store/list_onboarding_templates.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_user_activation_milestone.sql`, `examples/server/ai-chat-wizard/sql/store/list_user_activation_milestones.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_saved_workflow.sql`, `examples/server/ai-chat-wizard/sql/store/list_saved_workflows.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_prompt_library_item.sql`, `examples/server/ai-chat-wizard/sql/store/list_prompt_library_items.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_weekly_value_summary.sql`, `examples/server/ai-chat-wizard/sql/store/list_weekly_value_summaries.sql`, `examples/server/ai-chat-wizard/sql/store/create_product_analytics_event.sql`, `examples/server/ai-chat-wizard/sql/store/list_product_analytics_events.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_experiment_assignment.sql`, `examples/server/ai-chat-wizard/sql/store/list_experiment_assignments.sql`, `examples/server/ai-chat-wizard/sql/store/create_subscription_churn_feedback.sql`, `examples/server/ai-chat-wizard/sql/store/list_subscription_churn_feedback.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|GetSuperuserControlPlaneReturnsSnapshot)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|Auth)" -count=1`
- result: Passed. Typed store/query wiring now covers onboarding, activation milestones, saved workflows, prompt library items, weekly value summaries, analytics events, experiment assignments, and churn feedback.
- residual risk: These tables are now wired at store/query level with lifecycle coverage, but dedicated admin and `su` RPC CRUD surfaces are still pending.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-27 23:06 America/New_York

- completed todo: Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.
- files changed: `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement_test.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_test.go`, `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/server/ai-chat-wizard/server/app/benchmark_sla_test.go`, `examples/server/ai-chat-wizard/server/app/benchmark_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "Test(RequireUserEntitlement|SendAndSpeechNegativeBranches|SendStreamsThoughtsAndPersistsConversation|SendPersistsUsageTraceMetadata|SendRejectsConversationOwnedByAnotherUser|SendRejectsMissingAuthenticatedUserBeforeProviderWork)$" -count=1`; `$env:CHAT_WIZARD_ROOT='C:\\Users\\Cam\\Desktop\\GoWebComponents\\examples\\100-ai-chat-wizard'; go test ./examples/server/ai-chat-wizard/server/app -run "TestSendSLASweep$" -count=1`
- result: Passed. Entitlement checks now deny missing effective access by default while preserving existing Send-path error boundaries and benchmark/test expectations via explicit plan seeding.
- residual risk: Quota/rate/concurrency enforcement is still pending and remains the final open Agent 2 item.
- next suggested todo: Implement quota enforcement (`usage.monthly_token_limit`, per-user rate, concurrency) in `parseRequireUsageBudget`.

### 2026-03-27 23:06 America/New_York

- completed todo: Refresh `README.md` so it reads like a polished entry point for example 100, preserving the existing ASCII architecture diagram and enhancing it to better represent the current server, client, worker, routing, and gRPC flow.
- files changed: `examples/server/ai-chat-wizard/README.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. README now has a cleaner product entry, an updated architecture diagram aligned to `/socket` + worker/runtime behavior, and an explicit current data-flow summary.
- residual risk: Quick-start and route/runtime explainer sections are still pending as separate Agent 4 doc todos.
- next suggested todo: Update the README quick-start so local setup, build, run, seed, auth, provider stubs, and verification steps match the current example behavior exactly.

### 2026-03-27 23:04 America/New_York

- completed todo: Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Admin operational workflow documentation now defines actor intent, dependency tables, current-vs-target RPC surfaces, and guardrails for destructive changes.
- residual risk: Most workflow mutation RPCs described here are still intentionally pending and remain tracked in Agent 2 and Agent 3 todos.
- next suggested todo: Add typed user-admin query and mutation funcs for user search, user detail, disable user, restore user, recent sessions, recent usage, and recent audit history.

### 2026-03-27 23:03 America/New_York

- completed todo: Add browser-level smoke coverage for the pricing, auth, and dashboard routes.
- files changed: `test/playwrightgo/examples/example100_route_smoke_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100RouteSmokePricingAuthDashboard -v`
- result: Passed. New route smoke validates `/pricing#faq`, unauthenticated `/app` auth form rendering, and authenticated `/app/settings?panel=settings-profile` in one browser run with `status=200` and zero console/page errors.
- residual risk: This smoke confirms route availability and core shell mounts, but it does not yet assert model-label consistency or scroll-control behavior.
- next suggested todo: Trace and fix the model-label mismatch where the picker shows `GPT-5.4 � Best` while message bubbles show `GPT-5.4 mini`.

### 2026-03-27 23:02 America/New_York

- completed todo: Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Admin dashboard journey documentation now captures route entry, current superuser gating, data-slice dependencies, and operator action checkpoints.
- residual risk: Admin operational workflow stories (disable/restore/suspend/billing/support/incident) still need explicit product-flow documentation in Agent 4.
- next suggested todo: Document the admin operational workflows in product terms: disable user, restore user, suspend workspace, billing intervention, support triage, and incident control.

### 2026-03-27 23:01 America/New_York

- completed todo: Document the visit-to-first-chat flow in product terms, including which routes, panels, defaults, and backend calls participate at each stage.
- files changed: `examples/server/ai-chat-wizard/FLOWS.md`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./tools/gwc files -root .\examples\server\ai-chat-wizard -ext .md`
- result: Passed. Visit-to-first-chat flow documentation now exists with concrete routes, defaults, and RPC dependencies, including token-cost trace fields.
- residual risk: Admin dashboard journey and admin operational workflow documentation are still pending under Agent 4.
- next suggested todo: Document the admin dashboard journey in product terms, including route entry points, role splits, data dependencies, and expected operator actions.

### 2026-03-27 23:00 America/New_York

- completed todo: Verify the authenticated happy path end-to-end (login, create thread, send message, stream reply, refresh, reopen thread).
- files changed: `test/playwrightgo/examples/example100_authenticated_happy_path_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`
- result: Passed. New browser test seeds a temporary DB, authenticates as `demo@example.com`, creates a new chat thread, sends and streams a reply, reloads while preserving `/app/thread/:publicID`, and reopens the same thread from the sidebar with prompt text intact and zero console/page errors.
- residual risk: The flow still logs route-resolution warnings right after first-send (`reply completed before conversation route resolved`), so route-sync diagnostics remain noisy even when the happy path succeeds.
- next suggested todo: Add browser-level smoke coverage for pricing, auth, and dashboard routes.

### 2026-03-27 22:59 America/New_York

- completed todo: Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.
- files changed: `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/proto/chat.pb.go`, `examples/server/ai-chat-wizard/proto/chat_grpc.pb.go`, `examples/server/ai-chat-wizard/server/app/superuser_control.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/chat.proto` (run in `examples/server/ai-chat-wizard`); `go test ./examples/server/ai-chat-wizard/proto -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreSuperuserControlPlaneLifecycle|GetSuperuserControlPlaneReturnsSnapshot|GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|Auth)" -count=1`
- result: Passed. The superuser control-plane snapshot now includes typed operational rows for auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox rows, and background jobs.
- residual risk: Snapshot coverage is now extended, but dedicated typed mutating `su` CRUD RPCs for pricing and reliability control tables are still pending.
- next suggested todo: Add typed `su` CRUD RPCs for pricing controls: overages, quota policies, upgrade triggers, and dunning events.

### 2026-03-27 22:58 America/New_York

- completed todo: Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/auth_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AuthManager(SignupLoginAndTokenRoundTrip|NegativePaths|PersistsSessionAndRevocation|TokenKidRotationPolicy)|ChatServerAuthRPCs)$" -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAuthSessionLifecycle|StoreSuperuserControlPlaneLifecycle)$" -count=1`
- result: Passed. JWT validation now enforces `jti` and session-version checks while using `kid`-aware signing and verification with rotation-key fallback for legacy kid-less tokens.
- residual risk: Deny-by-default entitlement enforcement and usage-budget quota enforcement are still open and tracked in the next Agent 2 todos.
- next suggested todo: Implement deny-by-default entitlement enforcement for users missing effective entitlement rows.

### 2026-03-27 22:54 America/New_York

- completed todo: Add typed store/query funcs for the new operational tables: auth sessions, workspace invitations, webhook deliveries, support ticket messages, incident updates, notification outbox, and background jobs.
- files changed: `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/store_auth.go`, `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/superuser_control_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_workspace_invitation.sql`, `examples/server/ai-chat-wizard/sql/store/list_workspace_invitations.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_webhook_delivery.sql`, `examples/server/ai-chat-wizard/sql/store/list_webhook_deliveries.sql`, `examples/server/ai-chat-wizard/sql/store/create_support_ticket_message.sql`, `examples/server/ai-chat-wizard/sql/store/list_support_ticket_messages.sql`, `examples/server/ai-chat-wizard/sql/store/create_incident_update.sql`, `examples/server/ai-chat-wizard/sql/store/list_incident_updates.sql`, `examples/server/ai-chat-wizard/sql/store/create_notification_outbox.sql`, `examples/server/ai-chat-wizard/sql/store/list_notification_outbox.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_background_job.sql`, `examples/server/ai-chat-wizard/sql/store/list_background_jobs.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_auth_session.sql`, `examples/server/ai-chat-wizard/sql/store/get_auth_session_by_session_id.sql`, `examples/server/ai-chat-wizard/sql/store/list_auth_sessions.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run TestStoreSuperuserControlPlaneLifecycle -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run TestGetSuperuserControlPlaneReturnsSnapshot -count=1`; `go test ./examples/server/ai-chat-wizard/server/app -run TestAuth -count=1`
- result: Passed. The new operational-table store/query surfaces are wired and covered by focused lifecycle, superuser snapshot, and auth tests.
- residual risk: These tables are now typed in store/query layers, but dedicated `su` RPC surfaces for them are still pending.
- next suggested todo: Extend `GetSuperuserControlPlane` or add dedicated `su` RPCs for the new operational tables.

### 2026-03-27 22:53 America/New_York

- completed todo: Persist and enforce server-side auth sessions using `auth_sessions` and `auth_token_versions`.
- files changed: `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/store_auth.go`, `examples/server/ai-chat-wizard/server/app/store_superuser.go`, `examples/server/ai-chat-wizard/server/app/queries.go`, `examples/server/ai-chat-wizard/server/app/auth_test.go`, `examples/server/ai-chat-wizard/server/app/store_test.go`, `examples/server/ai-chat-wizard/server/app/startup_helpers_test.go`, `examples/server/ai-chat-wizard/sql/store/upsert_auth_session.sql`, `examples/server/ai-chat-wizard/sql/store/list_auth_sessions.sql`, `examples/server/ai-chat-wizard/sql/store/get_auth_session_by_session_id.sql`, `examples/server/ai-chat-wizard/sql/store/get_auth_token_version.sql`, `examples/server/ai-chat-wizard/sql/store/upsert_auth_token_version.sql`, `examples/server/ai-chat-wizard/sql/store/touch_auth_session_last_seen.sql`, `examples/server/ai-chat-wizard/sql/store/revoke_auth_session.sql`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(StoreAuthSessionLifecycle|AuthManagerPersistsSessionAndRevocation|ChatServerAuthRPCs|StoreSuperuserControlPlaneLifecycle)$" -count=1`
- result: Passed. Auth tokens are now backed by durable server sessions and token-version rows, and revoked or expired sessions are denied.
- residual risk: Key rotation (`kid`) and explicit per-token `jti` lifecycle policy are still pending and tracked by the next Agent 2 todo.
- next suggested todo: Implement token revocation (`jti`/session-version) and key-rotation (`kid`) policy in auth token validation.

### 2026-03-27 22:51 America/New_York

- completed todo: Add a focused end-to-end regression for server start, WASM shell boot, and background worker boot.
- files changed: `test/playwrightgo/examples/example100_startup_boot_test.go`, `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100StartupBoot -v`
- result: Passed. New browser regression confirms `status=200`, boot-shell removal after mount, `/app/chat.wasm` + `/worker/background-worker.wasm` responses, `grpc ready`, `worker ready`, and `background render worker pool ready` startup logs with zero console/page errors.
- residual risk: This checkpoint validates startup/runtime boot only; authenticated message-send flows and route-specific UX paths are still covered by separate todos.
- next suggested todo: Verify the authenticated happy path end-to-end (login, create thread, send, stream, refresh, reopen).

### 2026-03-27 22:43 America/New_York

- completed todo: Run live browser startup smoke against `http://127.0.0.1:8095/` and capture startup console/runtime errors.
- files changed: `examples/server/ai-chat-wizard/TODO.md`, `examples/server/ai-chat-wizard/CHANGELOG.md`
- validation run: `go run ./bin/example100_startup_smoke.go` (temporary local helper removed after execution)
- result: Passed. Startup route `/` returned `200` with page title `RelayDesk � AI Chat Workspace`, `console_error_count=0`, and `page_error_count=0`.
- residual risk: This checkpoint covers only initial load at `/`; route-specific startup regressions (pricing/auth/dashboard) still require dedicated browser assertions.
- next suggested todo: Add a focused end-to-end regression that proves server start, WASM shell boot, and background worker boot.

### 2026-03-27 20:30 America/New_York

- completed todo: Repair server policy/model regression paths so `server/app` tests pass again.
- files changed: `examples/server/ai-chat-wizard/server/app/server_tool_policy.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(GetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|GetSelectedModelRepairsUnsupportedPreference|GetServerToolPolicyReadsWhitelistFromSiteConfig|ProviderStubRuntimeSupportsCrossProviderSelection)"`; `go test ./examples/server/ai-chat-wizard/server/app`; `go test ./examples/server/ai-chat-wizard/server/...`
- result: Passed. Whitelist policy JSON now parses correctly and model-selection regression coverage is green.
- residual risk: Entitlement deny-by-default, usage-budget quotas, and token revocation/key-rotation remain intentionally open TODOs and are not implemented in this checkpoint.
- next suggested todo: Implement deny-by-default entitlement gating once guaranteed entitlement bootstrap rows exist for all active users.

### 2026-03-27 20:09 America/New_York

- completed todo: Gate admin analytics and diagnostics RPCs behind superuser authorization.
- files changed: `examples/server/ai-chat-wizard/server/app/admin_dashboard.go`, `examples/server/ai-chat-wizard/server/app/log_tail.go`, `examples/server/ai-chat-wizard/server/app/admin_dashboard_test.go`, `examples/server/ai-chat-wizard/server/app/log_tail_test.go`, `examples/server/ai-chat-wizard/server/app/testkit_test.go`, `examples/server/ai-chat-wizard/proto/chat.proto`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetLogTail)"`
- result: Blocked by a pre-existing compile failure in `server.go` (`undefined: parseSelectedModel`), unrelated to this authz-gating patch.
- residual risk: Authorization behavior is updated in code, but test confirmation is pending until the pre-existing server compile issue is resolved.
- next suggested todo: Add auth-secret startup validation and entitlement gate seams with TODO stubs for quota enforcement.

### 2026-03-27 20:09 America/New_York

- completed todo: Add production auth-secret validation and wire entitlement budget gate seams for `Send`.
- files changed: `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/authz_entitlement.go`, `examples/server/ai-chat-wizard/server/app/auth_service.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "Test(AdminDashboardRPCs|GetLogTail)"`
- result: Blocked by the same pre-existing compile failure in `server.go` (`undefined: parseSelectedModel`) before targeted test execution.
- residual risk: Entitlement and revocation seams are currently fail-open stubs for missing billing bootstrap and missing revocation backend; they are intentionally documented as TODO for follow-up implementation.
- next suggested todo: Implement deny-by-default entitlement behavior once user-to-plan bootstrap rows are guaranteed for all users.

### 2026-03-25 02:31 America/New_York

- completed todo: Define an AI provider companion package pattern for LLM-backed GWC applications.
- files changed: `docs/ECOSYSTEM.md`, `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The documented companion-package boundary matches the live catalog RPC path and the current wasm client surface.
- residual risk: The ecosystem guidance now captures the package boundary, but only RelayDesk currently validates the pattern, so promotion beyond `Experimental` would still require a second production-shaped app.
- next suggested todo: None in the current example-100 provider-switching slice.

### 2026-03-25 02:28 America/New_York

- completed todo: Define the SQL-backed model catalog pattern for runtime provider and model discovery.
- files changed: `examples/server/ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestNewChatServiceServerSupportsProviderStubs"`
- result: Passed. The documented SQL-backed catalog path matches the live startup and RPC behavior exercised by the server tests.
- residual risk: The README now defines the recommended bootstrap payload and freshness policy, but RelayDesk still relies on authenticated RPC revalidation rather than shipping the initial catalog through SSR bootstrap today.
- next suggested todo: Define an AI provider companion package pattern for LLM-backed GWC applications.

### 2026-03-25 02:14 America/New_York

- completed todo: Promote RelayDesk as the reference implementation for runtime AI provider switching.
- files changed: `examples/server/ai-chat-wizard/README.md`, `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `npx playwright test --config=playwright.chat-wizard.config.ts --grep "provider and model selection sync across open tabs"`
- result: Passed. The example README now explicitly positions RelayDesk as the runtime provider-switching reference app and points maintainers to the focused browser regression that proves the shipped flow.
- residual risk: The reference example now documents the shipped switching path clearly, but capability-aware filtering beyond provider membership is still not implemented in the UI.
- next suggested todo: Add capability-aware model filtering and picker messaging so RelayDesk can demonstrate why a provider or model disappears when a workflow requires a missing capability.

### 2026-03-25 01:18 America/New_York

- completed todo: Keep provider/model/intelligence selection stable across new chats and reconnects.
- files changed: `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/helpers_wasm_test.go`, `examples/server/ai-chat-wizard/client/app/model_preferences.go`, `examples/server/ai-chat-wizard/server/app/server.go`, `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/server/app -run "TestModelOptionAndSelectedModelRPCs|TestGetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel|TestRPCFallbacksWhenStoreOrProvidersAreUnavailable"`; `$env:GOOS='js'; $env:GOARCH='wasm'; go build ./examples/server/ai-chat-wizard/client/...`
- result: Passed. New-chat and reconnect flows now recover and persist provider/model state instead of leaving the picker blank.
- residual risk: This is covered by unit tests and compile checks, but there is still no browser-level end-to-end regression that exercises the full picker flow through a real page reload.
- next suggested todo: Add a Playwright regression that selects a non-default provider/model, reloads, starts a new chat, and verifies the same provider/model remains selected with a non-blank intelligence mode.

### 2026-03-24 22:12 America/New_York

- completed todo: Prevent newly created threads from being cleared when the first assistant response finishes.
- files changed: `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run TestShouldResetDraftForRootRoute`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The regression guard behaves correctly and the js/wasm app package still compiles.
- residual risk: This flow still lacks a browser-level end-to-end regression that drives a real send on a fresh draft thread.
- next suggested todo: Add an example-100 integration regression that seeds a fake provider response and verifies the browser stays on `/thread/:publicID` after the first reply.

### 2026-03-24 22:14 America/New_York

- completed todo: Add diagnostics for unresolved thread-route state after a fresh reply.
- files changed: `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/stream.go`, `examples/server/ai-chat-wizard/client/app/route_sync.go`, `examples/server/ai-chat-wizard/client/app/route_sync_test.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run 'TestShould(ResetDraftForRootRoute|WarnPendingRootRoute)$'`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The warning guard condition is locked in and the js/wasm app package still compiles.
- residual risk: The warnings improve diagnosis but they do not replace a browser-level regression that exercises a real first-message send path.
- next suggested todo: Add an end-to-end regression with a fake provider that verifies the app stays in the new thread after the first streamed reply and asserts the warning does not appear in the healthy path.

### 2026-03-24 22:19 America/New_York

- completed todo: Flatten the provider, model, and intelligence controls so they use width more efficiently.
- files changed: `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The example-100 client app still compiles for js/wasm after the control-bar layout change.
- residual risk: This is a visual adjustment only; there is still no browser-level layout regression covering narrow widths and the inline control row.
- next suggested todo: Add a Playwright visual/layout smoke for the compact control bar at desktop and mobile widths.

### 2026-03-24 22:24 America/New_York

- completed todo: Add hover and press animations to the toolbar selects.
- files changed: `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/styles.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The client app still compiles for js/wasm after the animated select styling pass.
- residual risk: Native option hover and press styling remain browser-dependent, so the select element motion is reliable but option-row animation fidelity will vary by platform.
- next suggested todo: Add a browser smoke that exercises the animated control bar in Chromium and confirms hover, focus, and press states remain readable.

### 2026-03-24 22:31 America/New_York

- completed todo: Add a floating down-arrow when more thread content is available below.
- files changed: `examples/server/ai-chat-wizard/client/app/app.go`, `examples/server/ai-chat-wizard/client/app/app_shell.go`, `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/client/app/helpers.go`, `examples/server/ai-chat-wizard/client/app/panel.go`, `examples/server/ai-chat-wizard/client/app/scroll_memory.go`, `examples/server/ai-chat-wizard/client/app/scroll_visibility.go`, `examples/server/ai-chat-wizard/client/app/scroll_visibility_test.go`, `examples/server/ai-chat-wizard/client/app/thread.go`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `go test ./examples/server/ai-chat-wizard/client/app -run TestHasScrollSpaceBelow`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The visibility helper is locked in and the js/wasm client app still compiles with the floating jump-to-bottom control.
- residual risk: Browser-level placement and overlap behavior still need a visual smoke pass, especially with split canvas mode and long threads.
- next suggested todo: Add a Playwright scroll smoke that verifies the button appears when the user scrolls up and jumps back to the latest message when clicked.

### 2026-03-24 22:37 America/New_York

- completed todo: Rebrand the example-100 product surface away from the framework name.
- files changed: `examples/server/ai-chat-wizard/client/app/constants.go`, `examples/server/ai-chat-wizard/server/app/bootstrap.go`, `examples/tests/100-ai-chat-wizard.spec.ts`, `examples/server/ai-chat-wizard/TODO.md`
- validation run: `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/server/ai-chat-wizard/client/app`
- result: Passed. The visible app brand and server-rendered page title now use the new product name, and the client app still compiles for js/wasm.
- residual risk: This updates the primary visible branding, but supporting copy such as the empty-state headline still reads like an experiment rather than a polished product surface.
- next suggested todo: Refresh the empty-state and onboarding copy so the rest of the home screen matches the new product branding.






