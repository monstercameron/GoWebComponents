# Example 100 Operator Runbook

This runbook defines the standard local operator flow for Example 100:
- build client artifacts
- seed a shared local DB
- start or restart the managed server
- verify chat login and first-send behavior
- verify superuser/admin backend surfaces

## 1) Environment Setup

Run from repo root:

```powershell
cd C:\Users\Cam\Desktop\GoWebComponents
```

Use one shared DB path for both server and seeding:

```powershell
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/chat_history.db"
```

Provider mode:

```powershell
# Local stubs (recommended for deterministic local ops):
$env:CHAT_PROVIDER_STUBS = "all"

# Optional real providers:
# $env:OPENAI_API_KEY = "sk-..."
# $env:ANTHROPIC_API_KEY = "sk-ant-..."
# $env:CEREBRAS_API_KEY = "csk-..."
```

## 2) Build Client Artifacts

```powershell
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard\client -out .\examples\100-ai-chat-wizard\bin\client\app\chat.wasm -json
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\backgroundworker\main.go -root .\examples\100-ai-chat-wizard\client\backgroundworker -out .\examples\100-ai-chat-wizard\bin\client\worker\background-worker.wasm -json
```

## 3) Seed Accounts and Baseline Data

```powershell
go run ./examples/100-ai-chat-wizard/cmd/seed-test-db
```

The managed `chat-wizard` start path now seeds the target `CHAT_DB_PATH` automatically when the local runtime DB is missing or empty, but the explicit seed step remains the cleanest way to reset local auth and demo data.

Seeded users:
- `customer@email.com / password`
- `admin@email.com / password`

Seed scope:
- Creates auth + profile rows for one customer login and one admin login (`customer@email.com`, `admin@email.com`).
- Seeds three demo conversations/messages for the customer login (`customer@email.com`).
- Does not grant any `su_user_roles` entries; admin checks require an explicit local `su` grant for superuser-only surfaces.

## 4) Start Managed Server

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server start -json
```

Lifecycle:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server stop -json
```

Health check:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8095/healthz
```

## 5) Verify User Flow (Chat Path)

1. Open `http://127.0.0.1:8095/`.
2. Log in as `customer@email.com`.
3. Create a new thread and send one prompt.
4. Confirm streamed reply completion.
5. Refresh and verify route remains `/app/thread/:publicID`.

Focused automated check:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v
```

## 6) Verify Superuser/Admin Backend Flow

Example 100 has dedicated dashboard routes at `/app/dashboard` and `/app/dashboard/*`; admin verification should include both route entry and RPC/back-end gating checks.

1. Log in as `admin@email.com` to verify auth path and capture expected denied states on superuser-only slices.
2. Verify deterministic superuser-only behavior with focused package tests (tests grant `su` roles internally):

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run "Test(GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|AdminDashboardRPCs)" -count=1
```

3. Validate no authz regression in core admin gates:

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run "Test(AuthManagerPersistsSessionAndRevocation|GetSessionRejectsMalformedMetadataToken)" -count=1
```

4. Optional local browser path for superuser route checks:

```sql
INSERT INTO su_user_roles (user_id, role_key, assigned_by_user_id, created_at)
SELECT u.id, 'su', u.id, strftime('%Y-%m-%dT%H:%M:%fZ','now')
FROM users u
WHERE u.email = 'admin@email.com'
ON CONFLICT(user_id, role_key) DO UPDATE
SET assigned_by_user_id = excluded.assigned_by_user_id;
```

After applying the grant in the same `CHAT_DB_PATH` database, restart the managed server and re-check `/app/dashboard` access with `admin@email.com`.

## 7) Dashboard Review Playbook (Planned)

Use this loop once dashboard surfaces are enabled. Goal: actionability over vanity.

1. `Business` review:
   - Check MRR trend, churn signals, failed-payment queue, and overage/upgrade pressure.
   - Do not stop at topline revenue alone; always pair with dunning/churn and plan-mix context.
2. `Customers` review:
   - Check active-user/workspace health, support backlog age, and high-risk accounts.
   - Prioritize accounts with both usage drop and support/billing stress, not just ticket volume.
3. `Chats` review:
   - Check first-chat conversion, reply latency/failure trends, and onboarding/template outcomes.
   - Avoid vanity prompt counts without completion quality and error-rate context.
4. `Providers` review:
   - Check provider/model reliability, cost-per-token shifts, routing fallback pressure, and guardrail hits.
   - Correlate outages/cost spikes with routing policy changes before mutating limits.
5. `Ops` review:
   - Check incident/SLO posture, job queue health, webhook delivery reliability, and audit anomalies.
   - Pair chart spikes with drill-down tables before declaring regression or recovery.

## 8) Mutation Preflight Guide

Use this preflight before any high-risk dashboard mutation (billing controls, provider policy changes, suspend/disable actions, incident actions).

1. Blast radius preview:
   - Confirm impacted workspace/user/queue counts.
   - Confirm whether the action is workspace-scoped or platform-scoped.
2. Affected-record counts:
   - Capture the exact count and IDs sampled from the current filtered view.
   - Reject mutation if target scope cannot be resolved deterministically.
3. Audit note quality:
   - Record why the action is needed now.
   - Include ticket/incident/reference identifiers and expected outcome.
4. Confirmation copy:
   - Explicitly state what will be disabled, suspended, rerouted, retried, or restored.
   - Explicitly state what is not affected to reduce accidental broad changes.
5. Rollback guidance:
   - Confirm the inverse action exists and is authorized.
   - Confirm one post-action verification query and one rollback verification query.

Mutation is considered safe to ship only when all five checks pass in the same operator session.

## 9) Startup and Troubleshooting Guide

Use this focused checklist for the most common local startup/runtime failures.

| Failure | How to confirm | Fix |
|---|---|---|
| Missing `third_party/GoGRPCBridge` submodule | Build/start fails with missing package/module paths under `third_party/GoGRPCBridge`. | Run `git submodule update --init --recursive` from repo root, then rerun build/start. |
| Wrong `CHAT_DB_PATH` between seed and server | Login users are missing or server behavior does not match seeded data. | Export one path once, rerun `cmd/seed-test-db`, and start server with the same `CHAT_DB_PATH`. |
| Missing auth secret in production mode | Startup exits with `CHAT_AUTH_SECRET is required when CHAT_ENV=production`. | Set `CHAT_AUTH_SECRET`, or for local-only development set `CHAT_ALLOW_INSECURE_AUTH_FALLBACK=true`. |
| Missing provider keys without stubs | Send path returns `Unavailable`; runtime logs warn that provider key is missing. | Set provider API keys, or set `CHAT_PROVIDER_STUBS=all` (or specific providers) for local deterministic runs. |
| Stale WASM artifacts | Browser boot fails or shows old UI/runtime behavior after server restart. | Rebuild both WASM outputs (`client/main.go` and `client/backgroundworker/main.go`) and restart the managed server. |
| Tunnel or health-check failures | `gwc ... status -json` is unhealthy, `/healthz` fails, or `/socket` cannot establish gRPC tunnel. | Check managed server status, validate `LISTEN_ADDR`, verify `/healthz`, then restart server and re-check browser console/runtime logs. |

Quick probes:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8095/healthz
```

## 10) Operator Diagnostics Guide

Use this evidence path during incidents before changing code.

| Evidence source | Command or location | What to look for |
|---|---|---|
| Browser console | DevTools `Console` + `Network` on failing route | Route mismatch, hydration errors, gRPC tunnel failures, auth/session errors, and request status for `/socket` + RPC calls. |
| Managed server status | `go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json` | Process state, expected port/listen status, and whether managed runtime reports healthy state. |
| Health endpoint | `Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8095/healthz` | Non-200 responses or timeouts indicating server/tunnel startup failure. |
| Runtime logs | `examples/100-ai-chat-wizard/bin/runtime/logs/server.stdout.log` and `.../server.stderr.log` | Startup validation failures, provider/auth warnings, RPC error lines, and panic traces. |
| Admin diagnostics lines | `auth`, `bootstrap`, `dashboard`, and mutation prefixes in server logs | `auth:` validation/session failures, `rpc.GetAdminDashboard: bootstrap` flow markers, `admin.dashboard.*` audit/event context, and `rpc.*: mutation failed` operator-action failures. |

Fast log triage:

```powershell
Get-Content .\examples\100-ai-chat-wizard\bin\runtime\logs\server.stdout.log -Tail 200
Get-Content .\examples\100-ai-chat-wizard\bin\runtime\logs\server.stderr.log -Tail 200
Get-Content .\examples\100-ai-chat-wizard\bin\runtime\logs\server.stdout.log -Tail 400 | Select-String -Pattern "auth:|rpc.GetAdminDashboard|admin\\.dashboard|mutation failed|bootstrap|store unavailable"
```

## 11) Role Model and Verification Matrix

Current role model is resolved from `GetSession` role summary + admin scope checks (`superuser` role grants and active workspace-admin membership scope).

| Role | Route reach today | RPC reach today | Verification focus |
|---|---|---|---|
| Normal user | Public routes (`/`, `/home`, `/pricing`, `/signup`) and standard app routes (`/app`, `/app/thread/*`, `/app/settings*`). Admin deep links (`/app/dashboard*`, `/admin*`, `/su*`) are denied/redirected. | Standard auth/chat/settings RPCs. Admin/dashboard and superuser RPCs fail closed (`PermissionDenied`/`Unauthenticated`). | Confirm normal chat/settings still work and admin deep links do not grant data access. |
| Workspace admin (active workspace role: `owner`/`admin`/`workspace_admin`) | Can enter dashboard deep-link namespace (`/app/dashboard*`) through admin scope checks. | Scoped Customers/Chats surfaces only: user/workspace/support/billing/usage/conversation slices and scoped detail calls. Superuser-only surfaces (`Business`, `Providers`, `Ops`, home-level global scope) remain denied. | Verify scoped list/detail reads and explicit denial on superuser-only slices. |
| Superuser (`su_user_roles.role_key='su'`) | Full dashboard deep-link access (`/app/dashboard` and `/app/dashboard/*`). | Full admin/superuser read surfaces plus superuser-only endpoints (`GetAdminDashboard`, `GetSuperuserControlPlane`, `GetSuperuserSlices`, `GetLogTail`) and role-gated mutation surfaces. | Verify global dashboard snapshot, control-plane reads, log-tail access, and role-gated mutation behavior. |

Current implementation status:

- Dashboard/operator surfaces are still read-heavy overall.
- Some typed admin mutations exist (for example billing overrides and selected control-plane actions), but significant workspace-admin and operator workflows remain partial, stubbed, or pending full browser-first UX wiring.
- Treat denied states as expected unless the role/surface pair is explicitly listed as allowed above.

## 12) Server-Tools Rollout Verification Checklist

Use this checklist before calling the server-tools replacement safe and ready.

| Check | Verification target | Evidence to capture |
|---|---|---|
| Policy read/write gates | Superuser-only reads/writes for server-tool policy; non-superuser denied | RPC status codes for allow vs deny paths, plus role context used in test session |
| Whitelist + argument enforcement | Only approved commands and allowed arg shapes execute | One allowed sample, one blocked sample, and denial reason text from RPC/log output |
| Fresh-session re-auth | Sensitive tool mutations require recent auth/session | One fresh-session success and one stale-session `Unauthenticated` failure with re-auth step |
| Stream lifecycle (`stdout`/`stderr`/`exit`) | Tool run stream emits expected output channels and final exit metadata | Captured event sequence showing stdout, stderr, exit code, and terminal close |
| Audit-log evidence | Each policy/tool action writes actor + action + target + result | Query/read of resulting audit rows with IDs/timestamps linked to executed action |
| Shutdown-on-disconnect safety | Server tool process/stream stops when client disconnects | Disconnect reproducer plus log evidence showing cleanup and no orphan process |

Minimum pre-ready pass:

1. Run package tests covering server-tool authz and lifecycle seams.
2. Run one end-to-end manual tool invocation with allow/deny paths.
3. Capture runtime log snippets and audit rows for all six checks above.
4. Record unresolved gaps as release caveats if any check is `FAIL` or `N/A`.

## 13) Local External-Auth Testing (Planned Rollout)

Use this split path to avoid losing the fastest local verification loop while external auth is landing.

### Quick QA path (email/password)

Purpose: deterministic local verification without external providers.

Required env:

```powershell
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/chat_history.db"
$env:CHAT_PROVIDER_STUBS = "all"
```

Required accounts:
- `customer@email.com / password`
- `admin@email.com / password`

Required checks:
1. `Login` for customer and admin succeeds.
2. Session refresh/logout behaves normally.
3. First-chat flow and dashboard-entry checks still pass.

### Provider-integrated path (Google/OIDC)

Purpose: callback, state/nonce, and link-or-login behavior checks once provider handlers are wired.

Required env (target):

```powershell
$env:CHAT_AUTH_SECRET = "<dev-secret>"
$env:CHAT_AUTH_GOOGLE_CLIENT_ID = "<client-id>"
$env:CHAT_AUTH_GOOGLE_CLIENT_SECRET = "<client-secret>"
$env:CHAT_AUTH_GOOGLE_REDIRECT_URL = "http://127.0.0.1:8095/auth/google/callback"
```

Generic workspace OIDC provider config (target) must include:
- issuer URL
- client ID/client secret
- callback URL
- workspace policy link (`password allowed/blocked`, `external allowed`, `sso required`)

Callback URL checklist:
- Local dev callback host/port exactly matches provider configuration.
- Callback route returns to a valid in-app auth entry route.
- Invalid state/nonce and denied consent produce explicit auth failure screens.

Policy split to enforce in testing:
- Keep quick QA password path operational in local/dev.
- Validate provider path separately with explicit provider-enabled config.

## 14) Operator Exit Checklist

- Server status is healthy or intentionally stopped.
- `CHAT_DB_PATH` value used for both seed and server was consistent.
- Chat happy path passed (manual or browser test).
- Superuser/admin backend checks passed.
