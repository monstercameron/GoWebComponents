# Manual Smoke For Example 100

This is the human release-gate checklist for example 100.

## Preconditions

From repo root, prepare one shared local DB and build both WASM artifacts:

```powershell
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard\client -out .\examples\100-ai-chat-wizard\bin\client\app\chat.wasm -json
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\backgroundworker\main.go -root .\examples\100-ai-chat-wizard\client\backgroundworker -out .\examples\100-ai-chat-wizard\bin\client\worker\background-worker.wasm -json
go run ./examples/100-ai-chat-wizard/cmd/seed-test-db
```

Start the managed example server:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server start -json
```

## Seeded Credentials

- customer account: `customer@email.com / password`
- admin account: `admin@email.com / password`
- seed scope note: `cmd/seed-test-db` does not grant `su_user_roles`; admin checks require an explicit local `su` grant for superuser-only surfaces.

## Release Gate Checklist

Mark each group `PASS`, `FAIL`, or `N/A`.

| Group | Status | Notes |
|---|---|---|
| Visitor flow |  |  |
| First-chat flow |  |  |
| Admin dashboard flow |  |  |
| Admin mutation flow |  |  |

## 1) Visitor Flow (Unauthenticated)

1. Open `http://127.0.0.1:8095/`.
2. Confirm boot shell appears briefly, then public shell mounts.
3. Visit `/home`, `/pricing`, and `/signup`.
4. Confirm no redirect loops and no console/page errors.
5. Confirm pricing/auth CTAs navigate to expected auth entry points.

## 2) First-Chat Flow (Demo User)

1. Log in with `customer@email.com / password`.
2. Confirm authenticated app shell mounts and sidebar loads.
3. Create a new thread and send one prompt.
4. Confirm assistant reply streams and finishes without runtime errors.
5. Refresh the page; confirm route remains `/app/thread/:publicID` and thread content reopens.
6. Open settings (`/app/settings?panel=settings-profile`) and verify profile controls render.

## 3) Admin Dashboard Flow (Admin User)

1. Log out and log in as `admin@email.com / password`.
2. Confirm authenticated shell boot succeeds without auth loops.
3. Confirm dashboard-capable access checks behave as expected for admin role in this build.
4. Capture any denied/empty/error states with exact route + console/server log evidence.

Notes:

- Example 100 exposes dedicated dashboard routes at `/app/dashboard` and `/app/dashboard/*`.
- Admin/superuser data access is still enforced by role-gated RPC boundaries; capture denied states with route and log evidence.
- For explicit superuser browser checks, grant `su` in the local DB first (see `OPERATOR_RUNBOOK.md`, section `6`).

## 4) Admin Mutation Flow

1. Exercise available high-impact admin actions in this revision (disable/restore/suspend/billing/support/incident controls).
2. For each action, capture confirmation state, submit result, and post-action refresh behavior.
3. Verify no stale privileged state remains after role/session changes.
4. Confirm logs include enough detail to reconstruct actor, target, and outcome.

If mutation controls are not implemented in this revision, mark `N/A` and record that as a release caveat.

## Optional Focused Automation (Fast Spot Checks)

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100StartupBoot -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100RouteSmokePricingAuthDashboard -v
```

## Testing Story Matrix

| Journey | Browser coverage | Focused package coverage | Manual smoke coverage |
|---|---|---|---|
| Visitor route hydration (`/`, `/home`, `/pricing`, `/signup`) | `TestExample100RouteSmokePricingAuthDashboard` | `server/app` route-shell helper tests (`shouldServeClientShell`, rewrite helpers) | Section `1) Visitor Flow (Unauthenticated)` |
| Authenticated startup and worker boot | `TestExample100StartupBoot` | `server/app/startup_helpers_test.go`, runtime/helper coverage in client/app wasm tests | Preconditions + Section `2) First-Chat Flow` |
| Login -> first send -> stream -> refresh/reopen | `TestExample100AuthenticatedHappyPath` | `server/app/rpc_test.go`, auth/session + send-path unit tests | Section `2) First-Chat Flow (Demo User)` |
| Admin dashboard entry and role-gated access | `TestExample100RouteSmokePricingAuthDashboard` (auth route/shell check) | `server/app/admin_dashboard_test.go`, `server/app/log_tail_test.go`, `server/app/superuser_control_test.go` | Section `3) Admin Dashboard Flow (Admin User)` |
| Admin mutation workflows (disable/restore/suspend/billing/support/incident) | Pending dedicated browser mutation specs | Pending typed mutation RPC/store tests by workflow | Section `4) Admin Mutation Flow` (`N/A` allowed until implemented) |

## Testing Gap Matrix (Shipped Areas)

| Feature area | Package-test coverage | Playwright coverage | Manual smoke coverage | Uncovered risk / likely bug-hunt seam |
|---|---|---|---|---|
| Public route model + shell delivery | Route-shell helpers and startup runtime helpers in `server/app` tests | `TestExample100RouteSmokePricingAuthDashboard`, `TestExample100StartupBoot` | Section `1) Visitor Flow` | Route aliases and deep-link edge cases can regress without direct browser assertions per alias path. |
| Auth/session lifecycle | Auth manager/session tests (`auth_*`, startup auth checks, session revocation paths) | Authenticated happy-path route smoke only | Sections `2)` and `3)` | Password-reset/email-verification browser coverage is thin; token/role edge cases can slip through. |
| First-chat send + token/cost transport | Send-path and RPC tests in `server/app`; usage-event persistence tests | `TestExample100AuthenticatedHappyPath` | Section `2)` | Provider-degraded streaming and retry paths are still under-covered in browser tests. |
| Settings/profile/billing account panes | Focused settings/package tests where present | No broad settings-focused end-to-end suite yet | Section `2)` includes spot-check only | UI-state and formula-parity regressions in settings billing panes may not be caught early. |
| Admin role gating + dashboard entry | `admin_scope`, `admin_dashboard`, `superuser_control`, `log_tail` tests | Route smoke includes dashboard entry checks | Section `3)` | Workspace-admin vs superuser route/state transitions still depend heavily on package-level verification. |
| Admin customers/chats scoped reads | Admin list/detail/store tests (`ListAdminUsers`, `ListAdminUsageEvents`, `ListAdminConversations`, detail scope tests) | Dedicated browser specs exist but naming/status drift remains (`*_pending_test.go`) | Sections `3)` and `4)` | Discoverability drift in pending-vs-real Playwright files can hide true browser coverage gaps. |
| Admin mutation paths (billing/support/control) | Several typed mutation/store tests exist by module | Partial browser mutation coverage; not comprehensive across all high-risk actions | Section `4)` (`N/A` allowed) | Confirmation/reason/audit + rollback UX gaps can miss blast-radius regressions. |
| Provider mode behavior (stub vs real) | Provider/runtime helper tests and stub-path package tests | Primarily stub-backed browser flows | Preconditions + Sections `2)` and `3)` | Production-only failures (key rotation, provider degradation, upstream API variance) need real-provider spot checks. |

## Playwright Naming / Status Audit

Audit result (current tree): `test/playwrightgo/examples` still has `*_pending_test.go` files that now execute real regression coverage and should be recategorized as active tests for discoverability.

### Classified as already-real regressions (rename planned)

- `example100_admin_business_workflow_pending_test.go` -> `example100_admin_business_workflow_test.go`
- `example100_admin_chats_workflow_pending_test.go` -> `example100_admin_chats_workflow_test.go`
- `example100_admin_customers_workflow_pending_test.go` -> `example100_admin_customers_workflow_test.go`
- `example100_admin_detail_routes_pending_test.go` -> `example100_admin_detail_routes_test.go`
- `example100_admin_list_mechanics_pending_test.go` -> `example100_admin_list_mechanics_test.go`
- `example100_admin_mutation_diagnostics_pending_test.go` -> `example100_admin_mutation_diagnostics_test.go`
- `example100_admin_mutation_pending_test.go` -> `example100_admin_mutation_test.go`
- `example100_admin_ops_pending_test.go` -> `example100_admin_ops_test.go`
- `example100_admin_ops_workflow_pending_test.go` -> `example100_admin_ops_workflow_test.go`
- `example100_admin_providers_workflow_pending_test.go` -> `example100_admin_providers_workflow_test.go`
- `example100_dashboard_drilldown_smoke_pending_test.go` -> `example100_dashboard_drilldown_smoke_test.go`
- `example100_dashboard_home_pending_test.go` -> `example100_dashboard_home_test.go`

### Classified as truly pending

- None in the current inventory.

Recategorization rule going forward:

- If coverage is real and runnable, do not keep `_pending_test.go`.
- If intentionally pending, keep `_pending_test.go` only with an explicit `t.Skip` + `PENDING:` reason and unblock criteria in file header comments.

## External Auth Manual Smoke Checklist (Planned Rollout)

Run this checklist once Google/OIDC callbacks are wired.

1. Customer password login still works (`customer@email.com / password`) and reaches first-chat send.
2. Admin password login still works (`admin@email.com / password`) and reaches dashboard entry checks.
3. Account linking path: existing password user logs in through external provider and links to the same user without duplicate account creation.
4. Workspace-required SSO denial: password login is denied for an SSO-required workspace with a clear policy message.
5. Enterprise callback success path: valid callback state/nonce creates or links session and lands on expected post-auth route.
6. Enterprise callback failure path: invalid/expired state, denied consent, or callback error lands on explicit auth failure messaging.

## Coverage Gap Notes and Bug-Search Checklists

### Password reset and update-password browser gap

Current status:
- Token lifecycle behavior is package-tested in auth/service/store seams.
- Browser-level request/consume/expired-token/post-reset-login flow is not clearly covered by one coherent Playwright spec.

Bug-search checklist:
1. Request reset for an existing email and verify neutral response behavior for unknown emails.
2. Consume a valid reset token and verify login works with new password.
3. Consume an expired/invalid token and verify clear failure UI without partial session state.
4. Verify old password no longer works after successful reset.
5. Verify reset flow preserves safe redirect behavior (no open redirect).

### Email verification lifecycle browser gap

Current status:
- Verification token persistence and service behavior are package-tested.
- Browser verification states (`verify`, `already used`, `expired`, resend-adjacent) lack explicit end-to-end coverage.

Bug-search checklist:
1. Verify a fresh token transitions account to verified state.
2. Re-open the same token and confirm already-used behavior is explicit and non-destructive.
3. Use an expired token and verify failure guidance plus resend direction.
4. Confirm post-verify login/session behavior matches verified-account expectations.

### Dashboard route smoke gap

Current status:
- Route smoke names emphasize pricing/auth/settings entry.
- There is no single named smoke gate that always exercises `/app/dashboard` plus at least one `/app/dashboard/*` slice route in the same run.

Bug-search checklist:
1. Add dashboard root route assertion to baseline route smoke.
2. Add one deep-link slice assertion (`/app/dashboard/customers` or equivalent).
3. Validate refresh/back/forward stability on dashboard routes.
4. Capture denied-vs-allowed behavior for admin vs superuser paths.

### Workspace-admin browser-coverage gap

Current status:
- Workspace-admin scope and denial boundaries are package-tested.
- Browser coverage is fragmented; no coherent workspace-admin operator journey covers scoped list, detail, empty-state, and mutation confirmations end to end.

Bug-search checklist:
1. Workspace-admin login -> scoped dashboard entry.
2. Scoped `Customers`/`Chats` list + detail navigation with preserved filters.
3. Empty-state behavior for unsupported/unavailable slices.
4. Confirmation behavior on allowed scoped mutations (or explicit denial if not enabled).
5. Attempt superuser-only slice and confirm deny behavior is clear.

### Stub-vs-real-provider coverage gap

Current status:
- Most browser smoke is validated with `CHAT_PROVIDER_STUBS=all`.
- Real-provider behavior is only spot-checked and can diverge in production-only conditions.

Concrete stub-heavy seams that still need real-provider checks:
- startup provider readiness and key-validation paths
- send-stream behavior under degraded upstream responses
- billing totals under real token usage payloads
- provider health/rate-limit/fallback telemetry fidelity

Bug-search checklist:
1. Run baseline smoke under stubs.
2. Re-run first-chat and one dashboard provider slice with one real provider key enabled.
3. Inject degraded provider behavior (timeout/429/invalid upstream payload) and verify operator-visible error handling.
4. Compare usage/billing totals between stub and real-provider runs for formula drift.

### Diagnostics-assertion coverage review

Current status:
- Many tests assert behavior and status codes.
- Fewer tests assert emitted warnings/operator hints at key failure seams.

High-value seams for diagnostics assertions:
- boot/hydration warning emission
- auth bootstrap failure hints
- dashboard partial-load warnings
- mutation audit confirmation logs
- retry/replay failure diagnostics

### Known untested or lightly tested seams

- intentionally stubbed server-tools RPC surface (`SetServerToolPolicy`, `RunServerTool`)
- tenant/site override readiness for server-owned catalog overlays
- cache invalidation after catalog/settings updates
- performance-threshold drift and slow-path warnings
- route or mutation seams covered mostly by package tests, not coherent browser journeys

### Recurring Release Bug-Hunt Checklist

Rotate through this matrix before calling example 100 stable:

1. Routes: `/`, `/home`, `/pricing`, `/signup`, `/app`, `/app/settings`, `/app/dashboard`, one dashboard slice deep link.
2. Flows: signup/login, first chat send, refresh/reopen, admin dashboard entry, one high-impact mutation.
3. Roles: normal user, workspace admin, superuser.
4. Env modes: `CHAT_PROVIDER_STUBS=all`, one real-provider key mode, production-like auth-secret mode.
5. Failure injections: expired session token, denied role access, provider timeout/error, partial dashboard data, callback state/nonce failure (when external auth is enabled).

## Evidence To Capture On Failure

- failing route and exact timestamp
- browser console/page errors
- relevant server log lines
- screenshot or short screen recording
- expected vs actual behavior summary
- bug template used (`docs/BUG_REPORT_TEMPLATES.md`)
