# How Example 100 Works

Example 100 is a full-stack teaching example for GoWebComponents. It uses one Go WASM browser application, a Go server, a WebSocket-carried gRPC tunnel, a background WASM worker, SQLite-backed stores, and provider adapters for OpenAI, Anthropic, Cerebras, and the local stub provider. The important idea is not that every product feature is finished. The important idea is that public delivery, authentication, chat streaming, admin operations, data ownership, and diagnostics all run through one coherent system.

Read this chapter first when you want the whole story before opening source files. The deeper chapters are:

1. [Public Route Delivery](PUBLIC_ROUTE_DELIVERY.md)
2. [Authenticated Shell](AUTHENTICATED_SHELL.md)
3. [Chat Request Lifecycle](CHAT_REQUEST_LIFECYCLE.md)
4. [Admin Dashboard Subsystem](ADMIN_DASHBOARD_SUBSYSTEM.md)
5. [Data Layer](DATA_LAYER.md)
6. [Observability And Failure Handling](OBSERVABILITY_FAILURE_HANDLING.md)
7. [Extension Seams](EXTENSION_SEAMS.md)
8. [Systems Glossary](SYSTEMS_GLOSSARY.md)

## System Shape

```text
Browser request
  |
  v
server/app/server.go
  |-- public and app routes return the shell HTML
  |-- static WASM and bootstrap assets are served from bin/client
  |-- /socket upgrades into the GoGRPCBridge tunnel
  v
client/main.go -> client/app/app.go
  |-- router resolves public, auth, app, thread, settings, dashboard, and canvas paths
  |-- auth bootstrap validates local token through GetSession
  |-- app shell composes sidebar, thread, settings, dashboard, and canvas regions
  v
typed RPCs + background worker + provider runtime
  |-- ChatService.Send streams ChatChunk events
  |-- markdown and artifact rendering move through client/backgroundworker
  |-- server/provider resolves provider, model, capabilities, and health snapshots
  v
SQLite store and operational records
  |-- users, sessions, conversations, messages, usage_events
  |-- workspaces, billing, support, incidents, audit_logs
```

The server deliberately keeps shell delivery boring: direct loads for public routes and authenticated app routes return the same client shell. The client then decides which surface to render by reading the current route and the resolved auth state. That makes refresh, deep links, and in-app route changes exercise the same code path instead of building separate public and private applications.

## Public Entry

```text
GET /, /home, /pricing, /signup, /login
  |
  v
server shell response
  |
  v
boot payload with route-critical copy and bootstrap state
  |
  v
WASM router renders marketing, auth, or redirect handoff
```

Public routes are first-class because they must work before a session exists. `client/app/routes.go` names the public paths, `server/app/server.go` serves the shell for direct loads, and `client/app/app.go` routes everything back through `parseChatWizardRoot`. The boot shell exists so the page can show stable public/auth content while the WASM app and gRPC tunnel are still coming online.

The i18n work is shaped around that first paint. Boot-critical namespaces such as `marketing` and `auth` are available early, while route-specific namespaces can load later. The client still has fallback strings for safety while the server-owned catalog migration continues, as documented in [I18N Architecture](I18N_ARCHITECTURE.md).

## Authenticated Runtime

```text
localStorage auth-token
  |
  v
GetSession / RefreshSession
  |
  v
appViewState
  |-- Authenticated
  |-- GRPCReady
  |-- CanAccessAdmin / IsSuperuser
  |-- selected model, tone, thinking settings
  |-- conversation, settings, dashboard, canvas state
```

Once auth resolves, `renderWorkspaceShell` in `client/app/app_shell.go` owns the major UI regions. The shell does not hand ownership to the server after boot. It composes the route-specific view using local app state plus typed RPC data. Settings, billing, dashboard slices, thread view, and canvas are route-owned surfaces inside the same authenticated shell.

Persistent preferences live in a mix of SQLite-backed profile settings, local storage, and cross-tab storage events. The selected model and thinking preferences are loaded through typed RPCs when available, then mirrored locally so route refresh and tab reopen are stable. Scroll memory and layout preferences are intentionally local; account data is reset on auth changes.

## Chat Flow

```text
composer draft
  |
  v
ChatService.Send request
  |-- current messages
  |-- selected provider/model settings
  |-- tone and thinking options
  v
server app
  |-- require session and entitlement
  |-- create or resolve conversation
  |-- resolve provider/model capability
  |-- stream provider events as ChatChunk
  |-- persist messages and usage
  v
client thread view
  |-- append user turn
  |-- incrementally render assistant deltas
  |-- normalize /app/thread/:publicID
  |-- refresh conversation list for replay
```

The chat path is the central runtime test. `client/app/stream.go` sends the request, applies streaming deltas, records time-to-first-token style timings, and handles route normalization after the server resolves the conversation. `server/app/server.go` owns the RPC implementation, while `server/provider` owns model/provider specifics. Provider adapters normalize upstream differences into provider events so the UI can focus on progressive rendering rather than OpenAI, Anthropic, or Cerebras wire formats.

Failure handling is part of the product surface. Empty drafts are ignored, concurrent sends are blocked, unavailable gRPC state shows a reconnect path, provider failure produces one customer-safe assistant error, and server-side diagnostics keep request and correlation metadata available for operator lookup.

## Admin And Operator Flow

```text
authenticated user
  |
  v
role summary from auth bootstrap
  |-- normal user: no dashboard entry
  |-- workspace admin: scoped customer/workspace surfaces
  |-- superuser: platform-wide business/provider/ops surfaces
  v
dashboard route
  |-- shared shell
  |-- typed slice RPC
  |-- list/detail/drill-down data
  |-- guarded mutation RPC with audit record
```

The dashboard subsystem teaches route-scoped async state and server-enforced role scope. The UI can hide admin entry points, but the server remains authoritative through `parseRequireAdminAccessScope`, `parseRequireAdminSliceScope`, and mutation-specific authorization checks. Workspace admins get scoped operational views. Superusers can reach platform-level surfaces such as business, providers, and ops.

Each slice follows the same pattern: route activates, typed RPC loads data, a view model renders summary cards plus tables or detail panes, and mutations require explicit confirmation and audit logging. That structure is repeated so a new dashboard slice can be added without inventing a new admin architecture.

## Data And Policy

```text
sql/store/schema.sql
  |
  |-- tables define durable ownership
  |-- sql/store/ops/*.sql and grouped query files define data access
  v
server/app store helpers
  |
  |-- map query rows into runtime structs
  |-- keep policy in Go services and authz helpers
  v
RPC handlers
  |
  |-- validate caller
  |-- apply product policy
  |-- return proto-shaped responses
```

SQLite is the durable teaching store. The schema contains auth, sessions, conversations, messages, usage, billing, workspaces, support, incidents, audit logs, prompt/workflow tables, and provider rollup tables. SQL files are loaded by the application, but policy does not belong in raw SQL alone. Entitlements, auth method policy, admin role scope, provider capability routing, and customer-safe error mapping live in Go service code where they can be tested directly.

Seed data is intentionally product-like. `cmd/seed-test-db` creates deterministic `customer@email.com / password` and `admin@email.com / password` accounts, billing records, profile preferences, and example conversations so browser and server tests can exercise real flows.

## Observability

```text
client event or RPC
  |
  |-- request id
  |-- correlation id
  |-- route, provider, admin slice, or target scope
  v
server/client logs + audit rows + usage rows
  |
  v
customer-safe copy with support reference
```

Example 100 treats failure visibility as a system requirement. Client logs cover boot, route normalization, gRPC readiness, reconnects, catalog loads, stream failures, and dashboard loads. Server logs cover auth role resolution, admin slice denial, provider failures, mutation submit/confirm/success/failure, and customer-safe error wrapping. Audit rows preserve operator actions, and usage rows preserve provider/model/cost facts.

Customer-facing copy should stay calm and safe. Raw provider payloads, tokens, cookies, API keys, and secrets should stay out of UI text. Operators should still get enough structured context to diagnose the event by request ID, correlation ID, target scope, provider, or admin slice.

## Growth Model

Example 100 grows by extending repeated patterns, not by adding one-off flows. Add a provider by implementing the provider interface, model catalog metadata, capability checks, traceability middleware, and dashboard snapshot behavior. Add a route by naming it in `routes.go`, serving it in the shell route gate when direct loads matter, and giving it a route-owned state load. Add an admin slice by reusing the dashboard shell, typed RPC boundaries, scoped queries, detail/drill-down pattern, and mutation audit path.

That repetition is the teaching value. A reader should be able to learn one slice, one route, one RPC, or one store helper and predict the next one.
