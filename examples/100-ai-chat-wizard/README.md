# 100 - AI Chat Wizard

A streaming AI chat workspace built with **Go WASM + gRPC + GoGRPCBridge + provider runtime**.

The browser client and server are both written in Go. The chat UI and background render worker compile to
WASM, and chat messages stream over a gRPC tunnel carried by WebSocket (`/socket`). The server can resolve
models from OpenAI, Anthropic, and Cerebras catalogs through one runtime provider registry.

---

## Why this example matters for GWC

Example 100 is the **flagship integration test** for the GoWebComponents framework. It is not primarily a SaaS demo — it exists to verify that every major framework pattern works together in one coherent, full-stack application.

The specific GWC capabilities it demonstrates:

| Pattern | Where it lives |
|---|---|
| **Single-shell routed SPA** | `client/app/app.go` + `client/app/routes.go` — one WASM app, multiple distinct route surfaces |
| **Server-rendered public routes** | `server/app/server.go` — shell-first HTML delivery for `/`, `/home`, `/pricing`, `/signup` |
| **WASM-authenticated workspace shell** | `client/app/app_shell.go` — session-gated workspace, settings, and dashboard surfaces |
| **Typed gRPC bridge** | `client/app/grpc*.go`, `server/app/server.go` — typed RPCs over GoGRPCBridge WebSocket tunnel |
| **Worker-backed rendering** | `client/backgroundworker/` — markdown and render metadata tasks offloaded to a background WASM worker |
| **Cross-tab preference sync** | `client/app/prefs*.go`, `state/` — sidebar state, theme, and model selection persist and sync across tabs |
| **Streaming progressive render** | `client/app/thread*.go` — server-streamed `ChatChunk` deltas applied incrementally to the thread view |
| **Route-scoped async state** | `client/app/settings_*.go`, `client/app/dashboard_shell.go` — per-route data that loads on activation and redraws on RPC update |

If a refactor silently breaks any row in this table, the example has stopped being a useful framework signal — regardless of whether product tests still pass.

---

## Start here

These are the four highest-signal files. Read them in order before exploring anything else.

### 1. `client/app/app.go` — top-level route dispatcher
**Teaches:** how GWC routes WASM UI output based on URL path. `ParseApp` owns the single render path from "URL changed" to "correct surface rendered". Every public, auth, and workspace view forks from this one function. Understanding it is the prerequisite for understanding anything else.

### 2. `client/app/app_shell.go` — authenticated workspace composition
**Teaches:** multi-region UI composition under one stateful shell. `renderWorkspaceShell` assembles the sidebar, thread panel, settings modal, and canvas surface as a layered set of GWC nodes driven by one `appViewState`. Shows how large UI surfaces stay composable without a global state hack.

### 3. `client/app/routes.go` + `server/app/server.go` — full routing contract
**Teaches:** the split between client-side route selection and server-side shell delivery. The client defines what path maps to what view; the server defines which request paths get the SPA shell vs direct static responses. Reading both together is the only way to understand why public routes, auth routes, and app routes behave differently despite sharing one HTML document.

### 4. `client/app/dashboard_shell.go` — representative complex surface
**Teaches:** a complete async-loading surface pattern: RPC fetch on activation, typed view-model derivation, multi-panel tile grid, and role-gated content — all in one self-contained file. Once you understand dashboard, settings panels and admin surfaces follow the same structure.

---

## Architecture

```
Browser (Go WASM, port 8095)                     Go Server (in-process gRPC)
+--------------------------------------------+    +---------------------------------------------+
| App shell routes: / /home /pricing /app... |    | HTTP mux                                     |
|  - Auth + chat UI state                     |    |  - GET /, /home, /pricing, /app/* -> shell  |
|  - Composer -> ChatService.Send() stream    |<-->|  - GET /app/chat.wasm, /worker/*.wasm       |
|                                            WS/gRPC| - GET /chat-bootstrap.js, /healthz           |
| Background worker (WASM)                     |    |  - /socket -> GoGRPCBridge tunnel            |
|  - Markdown/render metadata tasks            |    |                                             |
+--------------------------------------------+    | gRPC ChatService                             |
                                                  |  - Auth/session RPCs (Login/GetSession/...)  |
                                                  |  - Chat RPCs (Send/List/Load/Resolve)        |
                                                  |  - Admin/superuser RPCs                      |
                                                  |                                             |
                                                  | Store + provider runtime                      |
                                                  |  - SQLite tables (auth, chat, usage, ops)    |
                                                  |  - Provider registry + model catalog          |
                                                  +---------------------------------------------+
```

**Data flow**

1. Visitor lands on a public route (`/`, `/home`, `/pricing`, `/signup`) and bootstraps the shell.
2. User authenticates via `Login`/`Signup`; client persists token and validates with `GetSession`.
3. Client opens the gRPC tunnel at `/socket`; runtime marks `grpc ready` and worker readiness.
4. Authenticated shell hydrates profile/settings/catalog/conversation state from typed RPCs.
5. User sends a prompt; WASM client calls `ChatService.Send` with history, model, tone, and thinking settings.
6. Server resolves provider/model, streams `ChatChunk` deltas, and persists usage and conversation updates.
7. Client renders streamed output incrementally, normalizes canonical thread route, and keeps state resumable.

---
## Prerequisites

| Tool | Purpose |
|---|---|
| Go ≥ 1.25 | Build server and WASM client |
| `protoc` | Proto compiler (generate Go stubs) |
| `protoc-gen-go` | Go protobuf plugin |
| `protoc-gen-go-grpc` | Go gRPC plugin |
| Tailwind CSS CLI | Built via `go run ./tools/gwc tailwind` (cached under `third_party/tailwindcss/bin`) |

The `third_party/GoGRPCBridge` git submodule must be initialised:

```powershell
git submodule update --init --recursive
```
## Quick start

### 1. Initialize submodules (first clone only)

```powershell
git submodule update --init --recursive
```

### 2. Pick one shared local DB path

The server and `cmd/seed-test-db` must use the same `CHAT_DB_PATH`.

```powershell
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
```

### 3. Build both WASM artifacts

```powershell
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard\client -out .\examples\100-ai-chat-wizard\bin\client\app\chat.wasm -json
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\backgroundworker\main.go -root .\examples\100-ai-chat-wizard\client\backgroundworker -out .\examples\100-ai-chat-wizard\bin\client\worker\background-worker.wasm -json
```

The server serves Brotli sidecars when present, but raw `.wasm` artifacts are enough for local development.

### 4. Seed local login accounts

```powershell
go run ./examples/100-ai-chat-wizard/cmd/seed-test-db
```

Seeded credentials:

- `customer@email.com / password`
- `admin@email.com / password`

What `cmd/seed-test-db` creates today:

- two auth users (`customer@email.com`, `admin@email.com`)
- two user-profile rows with default model/tone/thinking preferences
- three demo conversations and messages for `customer@email.com`
- schema defaults (including `su_roles` definitions) from `sql/store/schema.sql`

What it does not create:

- no `su_user_roles` grants (so `admin@email.com` is not automatically superuser)
- no seeded workspace-admin membership graph for dashboard mutations
- no seeded superuser control-plane mutations or override history

### 5. Choose provider mode

Use real provider keys:

```powershell
$env:OPENAI_API_KEY = "sk-..."
$env:ANTHROPIC_API_KEY = "sk-ant-..."
$env:CEREBRAS_API_KEY = "csk-..."
```

Or run fully local provider stubs:

```powershell
$env:CHAT_PROVIDER_STUBS = "all"
```

### 6. Start the managed example server

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server start -json
```

Useful lifecycle commands:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server stop -json
```

Open `http://127.0.0.1:8095/`.

### 7. Verify auth and first chat

1. Log in with `customer@email.com / password`.
2. Create a new thread and send one prompt.
3. Refresh and confirm the thread reopens at `/app/thread/:publicID`.

### 8. Run focused browser verification

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100StartupBoot -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100RouteSmokePricingAuthDashboard -v
```

Superuser-only verification path:

- `admin@email.com` can validate auth and denied-role behavior by default.
- For allowed superuser browser checks, grant `su` to that user in the same `CHAT_DB_PATH` database (see `OPERATOR_RUNBOOK.md`, section `6`) and restart the managed server.

---

## Route model

The server serves one shell-first SPA surface, and the client router decides which public/auth/chat view to render.

| Route class | Paths | Behavior |
|---|---|---|
| Public landing routes | `/`, `/home`, `/capabilities`, `/pricing`, `/signup` | Server returns the chat shell bootstrap HTML; client renders marketing/auth-facing views for unauthenticated users. |
| Auth entry | `/` (login default), `/signup` (signup mode) | Auth form submits to gRPC (`Login`/`Signup`), then authenticated sessions pivot into `/app`. |
| App root and thread routes | `/app`, `/app/thread/:publicID`, `/app/thread/:publicID/canvas/:canvasID` | Authenticated shell routes for conversation list, active thread replay, streaming replies, and canvas artifacts. |
| Settings routes | `/app/settings?panel=settings-*` | Still part of the SPA app shell; query `panel` selects profile/tone/prompt/intelligence/speech/memories/language section. |
| Legacy entry aliases | `/login`, `/logout`, `/thread/:legacyID` | Server still serves shell for compatibility; client normalizes into current `/app` route model. |
| Admin/dashboard surfaces | `/app/dashboard`, `/app/dashboard/business`, `/app/dashboard/customers`, `/app/dashboard/chats`, `/app/dashboard/providers`, `/app/dashboard/ops` | Dedicated dashboard routes are part of the SPA route model; server still enforces role gates at RPC boundaries for admin/superuser data access. |
| Static/runtime assets | `/chat-bootstrap.js`, `/app/chat.wasm`, `/worker/background-worker.wasm`, `/static/*`, `/healthz`, `/socket` | Not SPA routes; served directly by HTTP mux or gRPC bridge (`/socket`). |

### SPA vs server-shell behavior

- For route-like paths (public routes plus `/app*` paths without file extensions), the server returns the same shell document.
- The client-side router selects the active view and can normalize thread routes after conversation resolution.
- Asset paths (WASM, JS, CSS, images, static files) bypass SPA shelling and are served directly.
- Legacy `/chat.wasm` and `/background-worker.wasm` requests are rewritten to `/app/chat.wasm` and `/worker/background-worker.wasm`.

### Authenticated shell UI region → GWC pattern map

This table maps the visible regions of the `/app` workspace shell to the GWC patterns and packages each one exercises. Use it as a reading guide when exploring `client/app/app_shell.go`.

| UI region | Primary files | GWC pattern demonstrated |
|---|---|---|
| **Top control bar** | `client/app/canvas_top_bar.go`, `app_shell.go` | `ui` composition + route-aware conditional rendering; control state derived from `appViewState` without prop drilling |
| **Sidebar** | `client/app/sidebar.go` | Async conversation list pagination, scroll-position memory (`cacheScrollPosition`), reactive open/closed toggle via `state.Atom` |
| **Thread body** | `client/app/thread*.go` | Streaming partial render — `ChatChunk` deltas applied incrementally; background-worker markdown task dispatch over worker message bridge |
| **Composer** | `client/app/composer.go` | Controlled text input with handler composition; disabled-during-stream state; starter-prompt injection |
| **Settings modal** | `client/app/settings_*.go`, `app_shell.go` | Route-scoped panel selection via `?panel=` query param; persisted preference RPCs (`UpdateProfile`); cross-tab preference sync via `state` package |
| **Canvas surface** | `client/app/canvas*.go` | Conditionally-mounted sub-route (`/canvas/:canvasID`); `iframe` sandboxing for untrusted HTML artifact rendering |
| **Dashboard tiles** | `client/app/dashboard_shell.go` | Multi-surface async-loading pattern: RPC on activation, typed view-model derivation, role-gated content, tile grid |
| **Admin drawers** | `client/app/admin_*.go` | Shared drawer/table composition reused across five slice surfaces; mutation banner pattern (confirm → submit → receipt) |

### Public-route delivery and hydration model

Public routes (`/`, `/home`, `/pricing`, `/signup`, `/security`, `/privacy`, `/terms`, `/status`, `/about`, `/contact`) are delivered as one HTML shell document and hydrated by the same WASM binary as the authenticated app. The ownership split:

| Responsibility | Owner | File(s) |
|---|---|---|
| Shell HTML document | Server | `server/app/server.go` — `GET /home`, `/pricing`, etc. return the bootstrap shell |
| Bootstrap loader | Server-owned JS | `chat-bootstrap.js` — shows loading state, fetches `app/chat.wasm`, hides on mount |
| i18n bundle delivery | Server (with embedded fallback) | `server/app/i18n*.go`; client embeds catalog via `i18n/bundle/bundle.go` as compile-time fallback |
| Route selection once mounted | WASM client | `client/app/app.go` `ParseApp` — reads `window.location.pathname` to select landing vs auth vs workspace shell |
| Public-route rendering | WASM client | `client/app/landing_shell.go`, `pricing_shell.go`, `signup_shell.go`, `about_shell.go`, etc. |
| Journey progress band | WASM client | `client/app/journey.go` — shared multi-step progress indicator shown on all public marketing routes |
| Marketing footer / header | WASM client | `client/app/marketing_footer.go` — shared across all public routes via `renderMarketingFooter` |

Key points for framework readers:
- There is **no separate SSR build**. The server delivers a static shell; the WASM client owns all rendering including the public marketing pages.
- **Route normalization happens client-side**: the server maps `/`, `/home`, `/pricing` etc. to the same shell document; the WASM client decides what to render based on the path.
- **i18n** is embedded at WASM compile time today (*planned*: server-delivered lazy namespace fetch to reduce WASM binary size).

---

## Pricing philosophy and billing vocabulary

### Plan boundary

- `Pro`: one serious operator, personal workspace scope, individual execution depth.
- `Team`: shared workspace with collaboration and admin relief for coordinated operators.
- `Enterprise`: contract/security/compliance path with negotiated controls and procurement flow.

### Billing model

Monthly bill must be represented as:

`platform fee + usage + service premium`

- `platform fee`: fixed recurring subscription charge for the selected plan/workspace tier.
- `usage`: raw provider model usage cost from metered token/tool consumption.
- `service premium`: platform premium multiplier or basis-point markup applied to raw usage.

### Customer-facing vocabulary contract

Use these terms consistently in docs and UI:
- `platform fee`
- `usage`
- `service premium`
- `total`

Avoid stale pricing language in customer copy:
- `free`
- `unlimited`
- `all models included`
- `no token caps`

---

## Runtime pieces

This is the current runtime chain from first paint to streamed reply.

| Runtime piece | Responsibility | Depends on |
|---|---|---|
| Boot shell (`/chat-bootstrap.js`) | Shows startup state, loads `chat.wasm`, then hides once the app mounts. | HTTP shell route, WASM artifacts, `wasm_exec.js`. |
| WASM client (`client/main.go`, `client/app/*`) | Owns router state, auth/session UX, thread state, composer send, stream rendering, and settings panels. | gRPC tunnel readiness, model/profile/conversation RPCs. |
| Background worker (`client/backgroundworker/main.go`) | Offloads markdown/render metadata tasks and async render helpers. | Worker WASM artifact and worker message bridge. |
| gRPC tunnel (`/socket`) | Carries unary + streaming RPCs over WebSocket between browser and in-process gRPC server. | GoGRPCBridge handler, client dial/reconnect loop. |
| Server HTTP handlers (`server/app/server.go`) | Serves shell routes, static assets, health probe, wasm_exec, and tunnel endpoint. | Runtime config, static dirs, bridge handler. |
| Chat/auth/admin RPC handlers | Implement auth/session, chat send/list/load, model/profile preferences, and admin/superuser surfaces. | Store layer, auth manager, provider registry. |
| Store layer (SQLite) | Persists auth sessions, conversations/messages, usage events, and control-plane records. | SQL query files under `sql/store`, DB path/config. |
| Provider layer (`server/provider/*`) | Resolves provider/model runtime, streams completions, and reports health/rate-limit metadata. | API keys or `CHAT_PROVIDER_STUBS`, model catalog rows. |

### Interaction sequence

1. Shell HTML and bootstrap JS load, then fetch `app/chat.wasm`.
2. Client mounts, starts worker lifecycle, and opens gRPC tunnel to `/socket`.
3. Auth/session bootstrap resolves (`GetSession`, optional `RefreshSession`), then app state hydrates (`ListModelOptions`, profile/settings, conversations).
4. Composer submit calls `Send`; server resolves provider/model, streams `ChatChunk` deltas, and writes usage + conversation state.
5. Client applies streamed deltas, updates canonical thread route, and keeps session/thread state resumable across reloads.

---

## Local Truth vs Planned Future State

This section separates what is already real in example 100 from what is still roadmap material.

| Area | Local truth (implemented now) | Planned future state |
|---|---|---|
| Admin surfaces | Dedicated dashboard routes exist (`/app/dashboard` + slice routes). Core admin/superuser reads and diagnostics are wired with role guards and audit events. | Denser end-to-end operator journeys with fully completed UI + mutation loops for every slice. |
| Workspace-admin scope | Workspace-admin scope resolution exists and can read scoped `Customers`/`Chats` slices with superuser-only surfaces denied. | Broader workspace-admin control plane (full action coverage, richer scoped queues/details, fewer read-heavy gaps). |
| Superuser scope | Superuser-only RPCs (`GetAdminDashboard`, `GetSuperuserControlPlane`, `GetSuperuserSlices`, `GetLogTail`) are enforced and test-covered. | Full production-grade operator workflows with complete preview/rollback UX and exhaustive browser regressions. |
| Server-owned i18n | Architecture and namespace ownership are documented; client still ships broad embedded catalogs while migration is in progress. | Server-delivered layered catalogs (boot + lazy namespace fetch), minimal embedded fallback strings, and publish/override workflows. |

---

## Stub-Removal Status

This status table tracks placeholder seams so docs do not claim behavior is still stubbed once code goes live.

| Seam | Status | Current truth |
|---|---|---|
| `server tools` (`SetServerToolPolicy`, `RunServerTool`) | Remaining placeholder | Authz + fresh-session checks exist, but both RPCs still return `Unimplemented` from `server/app/server_tool_stub.go`. |
| `provider memory extraction` | Partial live conversion | Runtime extraction pipeline is live in `server/app/server.go`; `OpenAIProvider` and stub provider return extraction results, while Anthropic/Cerebras extraction paths still report not implemented. |
| `entitlement fail-open guard` | Remaining placeholder | Send entitlement gate seam exists, but `parseRequireUserEntitlement` still has a documented fail-open fallback when billing state is unavailable (`server/app/authz_entitlement.go`). |
| `control-mutation authz helper` | Partial live conversion | Live feature-flag/experiment/incident mutation RPCs are implemented in `server/app/admin_control_ops.go`, but legacy helper stubs in `server/app/admin_control_mutation_authz.go` still return `Unimplemented`. |

---

## Provider Memory Extraction Parity Contract

Intended cross-provider contract for `ParseExtractUserMemories`:

| Contract element | Required behavior |
|---|---|
| Candidate shape | Each candidate follows `provider.UserMemoryCandidate`: `key`, `category`, `summary`, `detail`, `usefulness_score` (0-100), `confidence_score` (0-1), `rubric_reason`. |
| Score semantics | Server clamps scores and only persists candidates meeting current thresholds (`usefulness >= 60`, `confidence >= 0.55`) after dedupe and normalization. |
| Malformed-response fallback | OpenAI path uses strict JSON schema first, then fallback object extraction (`parseExtractJSONObject`) before failing. Parse failures are logged and do not block chat reply flow. |
| Provider unsupported behavior | Providers that cannot extract memories (currently Anthropic/Cerebras) return a clear `not implemented` error; runtime logs `memory extraction failed` and continues without saving memories. |
| User-visible expectation | Memory extraction is asynchronous best-effort enrichment. Chat replies continue even when extraction is skipped, queue-limited, unavailable, or parse-failed. |

Current provider status snapshot:

- OpenAI: live extraction path with strict schema + fallback parsing.
- Anthropic: extraction endpoint not implemented.
- Cerebras: extraction endpoint not implemented.
- Stub provider: deterministic no-op extraction (`nil` candidates, no error) for local flows.

---

## Release Notes and Upgrade Caveats (Placeholder Removal)

Use this section when landing placeholder-removal changes so local/dev assumptions and tests are updated in the same release.

### Entitlement fail-open fallback removal

- Current placeholder: `parseRequireUserEntitlement` allows fail-open when billing state is unavailable.
- When removed: send/authz paths become fail-closed if billing access control cannot be resolved.
- Local/dev impact: seeded/local runs that previously sent chat without full billing state may start returning deny/precondition errors.
- Test impact: update tests that currently assume send continues during missing billing state; seed explicit entitlement rows and assert deny/upgrade responses where appropriate.

### Legacy control-mutation authz stub removal

- Current placeholder: legacy helper stubs in `admin_control_mutation_authz.go` return `Unimplemented`.
- When removed: those seams should route through live shared authz + mutation behavior, matching real control-plane RPC expectations.
- Local/dev impact: callers relying on `Unimplemented` as expected behavior must switch to real allow/deny + persistence assertions.
- Test impact: retire stub-only assertions in `admin_control_mutation_authz_test.go` and migrate to live mutation coverage patterns used by `admin_control_ops_test.go`.

Release checklist for these removals:

1. Update this section with exact rollout date/commit and changed behavior.
2. Update manual smoke and operator runbook checks for new deny/error expectations.
3. Replace obsolete stub-only tests with real allow/deny/outcome assertions.

---

## Auth Architecture End State

Target model for external auth work:

1. One user account per person in `users`, with many linked login methods (`password`, `google_oidc`, generic workspace `oidc`, later `saml`) attached as identities, not separate account systems.
2. Workspace-level auth policy decides which login methods are allowed (`password allowed/blocked`, `external allowed`, `sso required`) for the selected workspace context.
3. Password and external providers share one session/token issuance path (`auth_sessions` plus token version checks), so refresh/revoke/audit behavior stays consistent across methods.
4. Authz remains role-based after login (normal user, workspace admin, superuser); login method does not bypass role scope.

### External Auth Product Policy Decisions

| Policy area | Decision |
|---|---|
| Same-email account auto-linking | Auto-link only when external provider email is verified and exactly matches one existing local user with no conflicting existing external identity for that provider+subject. |
| Workspace-required SSO vs password | If workspace policy is `sso required`, password login is denied for that workspace context. Password can still be allowed for workspaces that do not require SSO. |
| Multi-workspace policy conflicts | Resolve policy against the target workspace selected at login entry. If the workspace requires SSO, enforce SSO for that entry even if another workspace allows password. |
| Superuser break-glass path | Local password remains required for superuser break-glass access in local/dev and emergency recovery flows; external login is optional and never the only superuser path. |

### Privileged External-Auth Decision (Resolved)

This is the explicit repo policy for privileged identities:

| Privileged role | Password login | External login (`google_oidc` / `oidc` / `saml`) | Sensitive mutation gate | Fresh-session rule |
|---|---|---|---|---|
| Superuser | allowed and required as break-glass | allowed for normal session access | must re-auth with local password before sensitive superuser mutations | mutation session must be within `superuserMutationSessionMaxAge` |
| Workspace admin | allowed | allowed when workspace policy allows external | role/scope authz still enforced; no login-method bypass | standard session validity rules apply |

Implementation references:
- `server/app/auth_privileged_policy.go` (`parseAuthorizePrivilegedLoginMethod`, `parseAuthorizePrivilegedMutationSession`)
- `server/app/superuser_control.go` (`parseRequireSuperuserMutationUserID`)
- `server/app/auth_service.go` JWT `amr` claim used for privileged mutation policy checks

### Local Password Rollout Guardrail

Google/OIDC rollout must preserve local email/password as a first-class path for:

- quick QA (`customer@email.com / password`, `admin@email.com / password`)
- seeded local testing on one shared `CHAT_DB_PATH`
- offline development without mandatory external-provider dependencies

Any auth change that breaks local password login in this example is a regression unless explicitly gated by a documented workspace policy test case.

### Auth Provider Capability Matrix

| Provider path | Current status | Target users | Required backend/runtime pieces | Required UI/runtime pieces |
|---|---|---|---|---|
| Password | fully wired | local QA, general users, superuser break-glass | user/password auth store, session issuance/revoke, reset/verify token lifecycle | login/signup/reset/update forms, session restore/logout |
| Google OIDC | planned | consumer/prosumer users that prefer social login | start/callback handlers, state+nonce persistence, verified-email extraction, link-or-create, shared session issuance | login entry option, callback route handling, clear deny/error states |
| Generic workspace OIDC | config-only | enterprise/workspace SSO | provider config + policy resolution, workspace callback binding, JIT membership rules | workspace SSO entry, workspace-aware callback and policy messaging |
| SAML (future) | config-only (storage only) | enterprise IdP programs | canonical provider model compatibility, assertion/subject validation, policy enforcement | enterprise entry surfaces and callback/failure messaging |

### External-Auth Data Model Comparison

`auth_identities` + `auth_oidc_states` + `workspace_auth_policies` are the current runtime seam that is most migration-resilient for Google/OIDC/SAML direction:

- Shape A (runtime identity-linking seam):
  - `auth_identities`: canonical person-level identity row with provider key/type, provider subject, verified email, and shared session target.
  - `auth_oidc_states`: callback state/nonce persistence, expected-subject guard, return URL, and consumed tracking.
  - `workspace_auth_policies`: per-workspace policy gate for local password, external login, required-provider enforcement, and JIT membership behavior.
  - Shape A keeps account identity and workspace policy close together so provider-specific runtimes can plug in without changing downstream authorization contracts.

- Shape B (enterprise SSO config bucket):
  - `workspace_sso_configs`: per-workspace SSO config rows with provider key, SAML fields, domains, and `is_enabled`.
  - Shape B is clean as long-term enterprise config storage and avoids adding provider runtime assumptions into the identity tables.

This split lowers remigration risk when Google OIDC, generic OIDC, and SAML runtime paths land:

- `Google OIDC` and `generic workspace OIDC` map to Shape A via state+nonce and identity/provider-subject rows.
- `SAML` starts in Shape B as config-only metadata, then can move to active runtime with one dedicated callback/auth pipeline migration path.
- shared identity resolution and workspace policy logic stay stable across current implementation and future enterprise provider additions.

### Memory Extraction Provider Parity Decision (Resolved)

Final implementation decision for remembered-preferences extraction:

| Provider | Structured output approach | Parse fallback approach | Shared parity contract |
|---|---|---|---|
| OpenAI | Responses API with strict JSON-schema output | Extract JSON object from text if wrapper noise appears, then parse | `[]UserMemoryCandidate` normalized by category/score/key |
| Anthropic | Messages API tool-use schema (`extract_user_memories`) with explicit tool choice | Fallback text payload parse for compatibility; malformed payload resolves to empty candidates | Same normalized candidate contract |
| Cerebras | Chat Completions with `response_format.type = json_object` plus deterministic prompt contract | Deterministic strict parse, then JSON-object extraction fallback before surfacing parse failure | Same normalized candidate contract |

Implementation invariants:
- provider output must converge to one typed `UserMemoryCandidate` shape with stable field names (`key`, `category`, `summary`, `detail`, `usefulness_score`, `confidence_score`, `rubric_reason`)
- normalization/clamping and key derivation are shared (`parseNormalizeMemoryCandidates`) so providers differ only in transport/runtime semantics
- provider-specific extraction failures must not silently change stored schema or field meaning

### Marketed Capability vs Implemented Capability (Auth/Trust)

This note prevents docs and marketing copy from overstating auth/trust support before runtime exists.

| Capability area | Marketed capability statement (allowed wording) | Implemented capability (current truth) | Risk if overstated |
|---|---|---|---|
| Google login | "Google login is planned for this example." | Not wired end to end yet: no live Google start/callback runtime in example 100 server handlers. | Users/operators expect a button/path that cannot complete login. |
| Enterprise SSO | "Enterprise SSO configuration groundwork exists; runtime flow is not complete." | `workspace_sso_configs` and related policy direction exist, but generic OIDC/SAML runtime login flow is not fully implemented. | Enterprise trial flows fail at callback/policy steps and appear broken. |
| Account linking | "Account linking rules are defined as target behavior." | Policy/design intent documented, but full provider identity linking and conflict resolution are not yet fully live. | Duplicate or ambiguous account assumptions leak into support docs. |
| Trust controls (state/nonce/replay, policy denials, audit detail) | "Trust controls are partially implemented and expanding." | Password/session/authz controls are live; external-auth state/nonce/replay/return-to/provider-subject policy seams are implemented, while full external-auth start/callback runtime and complete auth audit trails remain pending. | Security posture appears stronger in docs than in executable runtime. |

Operator documentation rule:
- When a feature is not wired end to end, label it `planned` or `config-only` and avoid "available now" language.

---

### Paid Launch Truth Table

This matrix is the release gate for public surface claims on `/home`, `/pricing`, `/signup`, and auth entry.

| Surface | External claim | Status | Current truth |
|---|---|---|---|
| `/home` | Faster answers, fewer repeat questions, and one workspace for support, ops, and customer-facing teams. | shipped | The chat workspace, model routing, and public landing shell are live. |
| `/home` | Internal docs, document Q&A, and team knowledge base. | future | The example does not ship live docs search or a knowledge-base runtime yet. |
| `/home` | Enterprise-ready controls such as SSO, retention controls, and SLA guarantees. | config-only | Workspace auth-policy and SSO configuration storage exist, but the enterprise login runtime is still incomplete. |
| `/pricing` | Seat-based pricing with a clear platform fee and usage-based billing. | shipped | The pricing and billing model is usage-based and surfaced in the app. |
| `/pricing` | All models included. | config-only | Model availability is driven by the server catalog and provider configuration, not an unconditional blanket guarantee. |
| `/pricing` | Enterprise review, procurement, SSO, and contract controls. | future | Those are still launch-direction claims rather than fully delivered runtime guarantees. |
| `/signup` | Create a workspace and start solo before adding a team. | shipped | Local workspace signup and the quick-QA password path are live. |
| `/signup` | Team onboarding and collaboration expansion. | future | Team collaboration exists in the product direction, but it is not a launch guarantee for the public signup flow. |
| Auth entry | Local email/password login, password reset, and refresh-safe session restore. | shipped | Password auth, token lifecycle, and session persistence are live. |
| Auth entry | Google login and workspace SSO entry paths. | config-only | External-auth policy/config storage exists, but the public entry copy must not imply universal runtime availability. |

---

### Pre-Launch Product Review Checklist

Use this checklist before approving public copy changes for example 100.

- Block release if `/home`, `/pricing`, `/signup`, or auth entry claims unsupported auth behavior such as Google login or enterprise SSO without a shipped or config-backed row in the paid-launch truth table.
- Block release if marketing copy claims live docs search, document Q&A, or team knowledge base behavior that is not backed by the current runtime.
- Block release if provider-brand or model-family copy does not match the live server catalog and the provider labels surfaced by the runtime.
- Block release if any public capability claim lacks a `shipped`, `config-only`, or `future` status in the paid-launch truth table.

---

### Demo-Readiness Code Review Checklist

Use this checklist when reviewing code changes for demo polish instead of prototype behavior.

- Verify names follow the repo rules: verb-first helpers, domain subjects, and boolean prefixes where needed.
- Verify functions stay small enough to scan quickly and that larger workflows are split into clear helpers.
- Verify files live in the right domain area and avoid one-off historical dumping grounds.
- Verify GoDoc and intent comments explain why the code exists, not just what the next line does.
- Verify diagnostics are clear, customer-safe where needed, and actionable for operators.
- Verify dead code, stale branches, and placeholder behavior are removed instead of left behind.
- Verify server, transport, store, provider, and client layers agree on names, payload shapes, and error behavior.

---

### Variable-Name Quality Review Checklist

Use this checklist when reviewing example 100 hot paths so name cleanup stays deliberate instead of turning into style churn.

- Flag locals that hide their subject behind vague nouns like `data`, `info`, `result`, `state`, or `item` when a domain name is available.
- Prefer verb-first helper names and domain subjects so auth, dashboard, billing, chat send, and logging paths read like actions instead of buckets.
- Keep booleans explicit with `is`, `has`, `can`, or `should` so guard branches stay readable at call sites.
- Rename helper inputs only when the new name clarifies the owned domain, the direction of the value, or the boundary being crossed.
- Leave stable protocol, schema, and API field names alone unless the code is wrapping them in a clearer domain-specific alias.
- Review the surrounding call chain before renaming so the new name matches the actual responsibility instead of one local line of code.
- Prefer one small name cleanup per review pass when the code path is already changing for another reason.

---

## SQL-backed model catalog pattern

RelayDesk keeps provider and model discovery in SQLite instead of hard-coding model enums into the WASM client.

- [`sql/store/schema.sql`](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/sql/store/schema.sql) defines `model_catalog` as the source of truth for provider ID, display name, pricing, throughput, onboarding state, and capability flags such as thinking and speech support.
- [`server/app/model_catalog_store.go`](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/server/app/model_catalog_store.go) loads those rows into per-provider catalogs plus the global default, title-generation, and memory-extraction model picks during server startup.
- [`server/app/server.go`](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/server/app/server.go) exposes the runtime query contract through `ListModelOptions`, so the authenticated shell can ask the server which providers and models are currently live.

That pattern means a provider rollout is usually a data change, not a WASM rebuild: add or update rows in `model_catalog`, restart the server, and the next `ListModelOptions` refresh advertises the new catalog to connected clients.

For apps that want a zero-RPC first paint, use `ui.SSRBootstrap.Data` to ship the same catalog shape the RPC already returns:

```json
{
  "modelCatalog": {
    "defaultModel": "gpt-5.4-mini",
    "generatedAt": "2026-03-25T02:28:00-04:00",
    "models": [
      {
        "id": "gpt-5.4-mini",
        "label": "GPT-5.4 mini",
        "providerId": "openai",
        "providerLabel": "OpenAI",
        "supportsThinking": true,
        "supportsSpeech": true,
        "pricing": {
          "inputCostPerMillionUsd": 0.25,
          "outputCostPerMillionUsd": 2.0,
          "currency": "USD"
        }
      }
    ]
  }
}
```

RelayDesk treats that bootstrap payload, or the cached `chat-wizard:model-catalog` local-storage entry, as last-known-good UI state only. The authenticated shell still revalidates through `ListModelOptions` when the gRPC session comes up so long-lived tabs converge back to the server-owned catalog without a full page reload.

### Optional: one-command managed server start

Start the default managed profile directly:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server start -json
```

Then inspect or restart as needed:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server stop -json
```

---

## Source-linked UI inventory

Each row maps a visible UI surface to its primary source files. Use this as a navigation index when reading or modifying a specific region.

| Surface | Primary client files | Primary server files | SQL / data |
|---|---|---|---|
| Marketing shell | `client/app/landing_shell.go`, `client/app/landing_sections.go` | `server/app/server.go` (static route) | — |
| Auth shell | `client/app/auth_shell.go`, `client/app/auth.go`, `client/app/auth_session.go` | `server/app/auth_service.go`, `server/app/auth_workspace_policy.go`, `server/app/auth_google_oidc.go` | `sql/store/store_auth.go` |
| Workspace entrypoint | `client/app/app.go`, `client/app/app_shell.go`, `client/app/routes.go` | `server/app/server.go` boot RPC | — |
| Chat thread | `client/app/thread.go`, `client/app/thread_view.go`, `client/app/panel.go`, `client/app/stream.go` | `server/app/tunnel_handler.go`, `server/app/authz_entitlement.go` | `sql/store/*.sql` chat history |
| Composer | `client/app/composer.go`, `client/app/model_picker.go`, `client/app/model_prefs.go` | `server/app/grpc_server.go` send RPC | — |
| Canvas pane | `client/app/canvas.go`, `client/app/canvas_workspace.go`, `client/app/canvas_patch.go` | `server/app/server.go` | — |
| Settings panel | `client/app/settings_route.go`, `client/app/settings_profile.go`, `client/app/settings_security.go`, `client/app/account_costs.go` | `server/app/auth_service.go`, `server/app/billing_formula_guard.go` | `sql/store/store_billing.go` |
| Admin dashboard | `client/app/dashboard_shell.go`, `client/app/admin_data.go`, `client/app/admin_customers.go`, `client/app/admin_providers.go` | `server/app/admin_dashboard.go`, `server/app/admin_list_query.go`, `server/app/admin_billing_ops.go` | `sql/store/` admin query files |
| Admin mutations | `client/app/admin_ops.go`, `client/app/admin_server_tools.go` | `server/app/admin_mutation_authz.go`, `server/app/admin_mutation_effects.go`, `server/app/admin_control_ops.go` | `sql/store/` ops query files |
| Worker markdown | `client/backgroundworker/worker.go`, `client/app/worker_render_types.go` | — | — |

---

## File layout

```text
examples/100-ai-chat-wizard/
+-- docs/
|   +-- BUG_REPORT_TEMPLATES.md
|   +-- PERFORMANCE.md
+-- bin/
|   +-- client/
|   |   +-- app/chat.wasm
|   |   +-- worker/background-worker.wasm
|   +-- runtime/
|   |   +-- chat_history.db
|   |   +-- logs/server.stderr.log
|   |   +-- logs/server.stdout.log
|   |   +-- legacy-artifacts/
|   +-- server/
|       +-- chat-wizard.exe
|       +-- chat-wizard-server.exe
+-- scripts/                    # reserved for example-local helpers
+-- client/
|   +-- main.go
|   +-- app/
|   +-- backgroundworker/
+-- cmd/
|   +-- server/
|   +-- seed-test-db/
+-- proto/
|   +-- chat.proto
|   +-- chat.pb.go
|   +-- chat_grpc.pb.go
+-- server/
|   +-- app/
|   +-- provider/
+-- sql/
+-- testdata/
+-- README.md
+-- FLOWS.md
+-- MANUAL_SMOKE.md
+-- SCHEMA_TABLES.md
+-- DESIGN.md
+-- DOCS_MAP.md
+-- OPERATOR_RUNBOOK.md
+-- TODO.md
+-- CHANGELOG.md
```

### Walkthrough Path Map

If you want the cleanest end-to-end explanation of the example, start here:

- Routing and boot: `client/main.go`, `client/app/routes.go`, `client/app/app_shell.go`, and `client/app/route_sync.go` show how the shell decides which surface to render and how it keeps URL state stable.
- Auth: `client/app/auth.go`, `client/app/auth_shell.go`, `server/app/auth_service.go`, `server/app/auth_workspace_policy.go`, `server/app/auth_identity_linking.go`, and `server/app/auth_google_oidc.go` show login, policy, linking, and external-auth flow.
- Chat streaming: `client/app/thread.go`, `client/app/stream.go`, `client/app/panel.go`, `server/app/server.go`, and `server/app/tunnel_handler.go` show send, stream, reconnect, and thread-state handling.
- Billing: `client/app/settings_route.go`, `client/app/account_costs.go`, `server/app/billing_formula_guard.go`, `server/app/store_billing.go`, and `server/app/superuser_pricing_ops.go` show usage formulas, plan state, and operator pricing controls.
- Dashboard reads: `client/app/dashboard.go`, `client/app/admin_data.go`, `server/app/admin_dashboard.go`, `server/app/admin_list_query.go`, `server/app/admin_business_ops.go`, and `server/app/admin_billing_ops.go` show the operator read path.
- Admin mutations: `server/app/admin_mutation_authz.go`, `server/app/admin_mutation_effects.go`, `server/app/admin_control_ops.go`, `server/app/admin_provider_mutation_authz.go`, `server/app/admin_ops_action_authz.go`, and `server/app/admin_chat_mutation_authz.go` show the guarded mutation path.

### Dashboard slice pattern — one structure reused five times

The admin dashboard has five slices (Business, Customers, Chats, Providers, Ops). Each follows the same four-step structure. Once you can read one, you can read all five.

| Step | Client | Server |
|---|---|---|
| 1. Route activation triggers data load | `dashboard_shell.go` dispatches the admin-data RPC on mount | `admin_dashboard.go` gate-checks role before running the query |
| 2. Typed view-model derivation | `admin_data.go` maps the proto response into slice-specific view structs | `admin_list_query.go` builds paginated, typed results per slice |
| 3. Tile grid render | `dashboard_shell.go` (overview) + `admin_customers.go`, `admin_providers.go`, etc. render the per-slice grid | — |
| 4. Role gate | `dashboard_shell.go` checks `parseView.CanAccessAdmin` / `parseView.IsSuperuser` before rendering admin-only sections | `admin_mutation_authz.go` + `admin_ops_action_authz.go` enforce before every mutating RPC |

**What this teaches for GWC:** route-scoped async state where each tab activates its own data fetch without blocking other tabs, and role-gated regions that are controlled at both the UI node level and the server RPC level. The same shape applies to settings panels: each settings section fetches its own data on activation and submits changes through a typed RPC.

The short teaching version is:
- `client/main.go` starts the shell.
- `server/app/server.go` owns the gRPC entrypoint and routes.
- `client/app/*` owns the browser story.
- `server/app/*` owns the runtime truth, policy, and data access.

### Repo Structure Map

This is the purpose-first view of example 100:

| Purpose | Primary folders | Representative files |
|---|---|---|
| Auth | `client/app/`, `server/app/`, `sql/store/` | `client/app/auth.go`, `client/app/auth_shell.go`, `server/app/auth_service.go`, `server/app/auth_workspace_policy.go`, `server/app/auth_identity_linking.go`, `server/app/auth_google_oidc.go`, `server/app/store_auth.go`, `server/app/store_auth_external.go` |
| Chat | `client/app/`, `server/app/` | `client/app/thread.go`, `client/app/stream.go`, `client/app/composer.go`, `client/app/panel.go`, `server/app/funnel_first_chat.go`, `server/app/tunnel_handler.go`, `server/app/authz_entitlement.go` |
| Dashboard | `client/app/`, `server/app/` | `client/app/dashboard.go`, `client/app/admin_data.go`, `client/app/admin_customers.go`, `server/app/admin_dashboard.go`, `server/app/admin_list_query.go`, `server/app/admin_business_ops.go`, `server/app/admin_billing_ops.go` |
| Canvas | `client/app/` | `client/app/canvas.go`, `client/app/canvas_workspace.go`, `client/app/panel.go`, `client/app/worker_render_types.go` |
| Billing | `client/app/`, `server/app/`, `sql/store/` | `client/app/account_costs.go`, `server/app/billing_formula_guard.go`, `server/app/store_billing.go`, `server/app/superuser_pricing_ops.go`, `sql/store/*.sql` billing queries |
| SQL | `sql/store/` | `sql/store/schema.sql`, query files grouped by auth, billing, admin, growth, and ops concerns |
| Runtime assets | `bin/` | `bin/runtime/`, `bin/client/`, `bin/server/`, managed logs and generated artifacts |
| Docs | repo root + `docs/` | `README.md`, `FLOWS.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, `DESIGN.md`, `DOCS_MAP.md`, `OPERATOR_RUNBOOK.md`, `docs/BUG_REPORT_TEMPLATES.md`, `docs/PERFORMANCE.md` |

### Comment And GoDoc Style Checklist

Use this style guide when adding or polishing comments in example 100.

- Keep GoDoc short, direct, and purpose-first.
- Start each GoDoc comment with the function name it documents.
- Use package docs to explain why the package exists and what it owns.
- Use file headers only when they add useful orientation for a reader.
- Use intent comments for tricky blocks, invariants, and cross-layer boundaries.
- Avoid mechanics comments that just repeat the next line of code.
- Keep the tone consistent across client, server, store, provider, and SQL-adjacent helpers.

## Repo layout rules

- Keep primary entry docs at root (`README.md`, `FLOWS.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, `TODO.md`, `CHANGELOG.md`).
- Keep secondary/supporting docs under `docs/` (for example `docs/BUG_REPORT_TEMPLATES.md`, `docs/PERFORMANCE.md`).
- Keep runtime state and logs under `bin/runtime/` (`chat_history.db`, `logs/`, managed state files).
- Keep generated binaries and build artifacts under `bin/` subfolders (`bin/client`, `bin/server`, `bin/runtime/legacy-artifacts`).
- Keep helper scripts under `scripts/` only; avoid scattering executable helpers at root.
- Keep source code under `client/`, `server/`, `cmd/`, `proto/`, `sql/`, and `internal/`.

---

## Regenerating proto stubs

```powershell
cd examples/100-ai-chat-wizard
protoc --go_out=. --go_opt=paths=source_relative `
       --go-grpc_out=. --go-grpc_opt=paths=source_relative `
       proto/chat.proto
```

RelayDesk is the repo's reference implementation for runtime AI provider switching. It demonstrates:

- a server-backed runtime model catalog loaded into the authenticated shell at startup
- provider and model switching without a page reload
- per-user selected-model persistence through the chat RPC surface
- live cross-tab synchronization of the active provider/model selection
- local stub-provider workflows so provider switching stays testable without real upstream keys

---

## Benchmarks

Use the server benchmark suite to get a quick read on store cost and synthetic active-chat pressure by clients-per-core:

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run '^$' -bench 'Benchmark(StoreCorePaths|SendClientsPerCore)' -benchmem
```

`BenchmarkSendClientsPerCore` uses a synthetic provider and reports `clients/core` sub-benchmarks at `1`, `2`, and `4`.

For an SLA sweep that pins the server to `1..8` cores and increases concurrent clients until the p95 request latency breaches the target:

```powershell
$env:CHAT_WIZARD_BENCH_SLA_MS = "100"
go test ./examples/100-ai-chat-wizard/server/app -run TestSendSLASweep -v
```

Optional knobs:
- `CHAT_WIZARD_BENCH_MAX_CORES` default `8`
- `CHAT_WIZARD_BENCH_MAX_CLIENTS` default `64`
- `CHAT_WIZARD_BENCH_BURST_RUNS` default `3`
- `CHAT_WIZARD_BENCH_PREDICT_CORES` default `32`

The sweep logs:
- measured `max_clients_under_sla` for each core count
- `clients_per_core`
- `scaling_vs_1_core`
- cumulative rollup totals across core levels
- a linear-regression projection for the requested prediction core count using uncensored cumulative rollup points

For a fuller explanation of the output fields, interpretation, and latest measured sample data, see [PERFORMANCE.md](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/docs/PERFORMANCE.md).

---

## Key packages

| Package | Role |
|---|---|
| `github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel` | WebSocket↔gRPC tunnel (server + WASM client) |
| `github.com/monstercameron/GoWebComponents/ui` | Hooks-based WASM UI (state, effects, events) |
| `github.com/monstercameron/GoWebComponents/html` | Typed HTML node builders |
| `google.golang.org/grpc` | gRPC runtime |
| `google.golang.org/protobuf` | Protobuf serialisation |

The GoGRPCBridge module lives at `third_party/GoGRPCBridge` (git submodule) and is
referenced via a `replace` directive in the root `go.mod`.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `127.0.0.1:8095` | Server listen address |
| `CHAT_ENV` | `development` | Runtime environment label. `production` enforces stricter auth-secret startup validation. |
| `CHAT_AUTH_SECRET` | _(empty)_ | Primary JWT signing secret. Required when `CHAT_ENV=production` unless insecure fallback is explicitly enabled. |
| `CHAT_ALLOW_INSECURE_AUTH_FALLBACK` | `false` | When truthy (`1/true/yes/on`), permits startup with development fallback auth secret even without `CHAT_AUTH_SECRET`. |
| `CHAT_AUTH_SIGNING_KEYS` | _(optional)_ | Comma-separated key ring (`kid=secret,kid2=secret2`) used for JWT verification/rotation in addition to the primary secret. |
| `CHAT_AUTH_ACTIVE_KID` | `primary` | Active JWT signing key ID. If missing or unknown, server falls back to the first available configured key ID. |
| `CHAT_MODEL` | _(optional)_ | Default model override for runtime startup. |
| `OPENAI_MODEL` | _(optional)_ | Legacy fallback default model used only when `CHAT_MODEL` is empty. |
| `OPENAI_API_KEY` | _(optional)_ | OpenAI provider key. Required unless `CHAT_PROVIDER_STUBS` includes `openai`/`all`. |
| `ANTHROPIC_API_KEY` | _(optional)_ | Anthropic provider key. Required unless `CHAT_PROVIDER_STUBS` includes `anthropic`/`all`. |
| `CEREBRAS_API_KEY` | _(optional)_ | Cerebras provider key. Required unless `CHAT_PROVIDER_STUBS` includes `cerebras`/`all`. |
| `CHAT_PROVIDER_STUBS` | _(empty)_ | Comma-separated stub provider list (`openai,anthropic,cerebras`) or `all` for full local stubs. |
| `CHAT_DB_PATH` | `examples/100-ai-chat-wizard/bin/runtime/chat_history.db` | SQLite database path for auth and conversation persistence |
| `CHAT_USAGE_PREMIUM_PERCENT` | `5` | Service-premium percent added on top of raw usage costs. Invalid/negative values fall back to default; values above `1000` are clamped. |

Usage-premium behavior:

- Parsed once at server startup and applied to billing total calculations.
- Exposed to the boot payload as `window.__relaydesk_usage_premium_percent` for client-side display parity.

---

## What this example demonstrates

- **gRPC over WebSocket in the browser** — no HTTP polling, no custom wire format
- **Server-streaming RPC** — tokens flow from OpenAI → gRPC server → browser in real time
- **Go WASM UI** — the entire frontend is Go; no JavaScript application code
- **Hooks pattern for async state** — `UseState`, `UseEffect`, `UseRef`, goroutines
- **Companion submodule pattern** — `GoGRPCBridge` consumed via `third_party/` + `go.mod replace`

