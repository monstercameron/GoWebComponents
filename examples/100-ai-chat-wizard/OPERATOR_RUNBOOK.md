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
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
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

Seeded users:
- `demo@example.com / password123`
- `admin@example.com / password`

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
2. Log in as `demo@example.com`.
3. Create a new thread and send one prompt.
4. Confirm streamed reply completion.
5. Refresh and verify route remains `/app/thread/:publicID`.

Focused automated check:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v
```

## 6) Verify Superuser/Admin Backend Flow

There is no dedicated dashboard route yet; current admin verification is RPC/back-end focused.

1. Log in as `admin@example.com` to verify auth path.
2. Run focused server tests that exercise admin/superuser typed surfaces:

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run "Test(GetSuperuserControlPlaneReturnsExtendedOperationalSnapshot|AdminDashboardRPCs)" -count=1
```

3. Validate no authz regression in core admin gates:

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run "Test(AuthManagerPersistsSessionAndRevocation|GetSessionRejectsMalformedMetadataToken)" -count=1
```

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

## 8) Operator Exit Checklist

- Server status is healthy or intentionally stopped.
- `CHAT_DB_PATH` value used for both seed and server was consistent.
- Chat happy path passed (manual or browser test).
- Superuser/admin backend checks passed.
