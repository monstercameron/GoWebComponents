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

- demo account: `demo@example.com / password123`
- admin dev account: `admin@example.com / password`

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

1. Log in with `demo@example.com / password123`.
2. Confirm authenticated app shell mounts and sidebar loads.
3. Create a new thread and send one prompt.
4. Confirm assistant reply streams and finishes without runtime errors.
5. Refresh the page; confirm route remains `/app/thread/:publicID` and thread content reopens.
6. Open settings (`/app/settings?panel=settings-profile`) and verify profile controls render.

## 3) Admin Dashboard Flow (Admin User)

1. Log out and log in as `admin@example.com / password`.
2. Confirm authenticated shell boot succeeds without auth loops.
3. Confirm dashboard-capable access checks behave as expected for admin role in this build.
4. Capture any denied/empty/error states with exact route + console/server log evidence.

Notes:

- Example 100 currently has superuser-gated admin RPC surfaces; dedicated dashboard route IA may still be in progress.
- If admin route UI is unavailable in this revision, mark this group `N/A` with explicit reason.

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

## Evidence To Capture On Failure

- failing route and exact timestamp
- browser console/page errors
- relevant server log lines
- screenshot or short screen recording
- expected vs actual behavior summary
