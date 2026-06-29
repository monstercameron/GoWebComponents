# Admin Dashboard Subsystem

The admin/dashboard subsystem is the operator surface for workspace and platform health. It is intentionally built as a set of repeated slice patterns rather than one bespoke dashboard page.

## Role Resolution

```text
GetSession role summary
  |
  |-- CanAccessAdmin
  |-- IsSuperuser
  v
client shell entry visibility
  |
  v
server-side parseRequireAdminAccessScope
  |-- platform scope for superuser
  |-- workspace scope for owner/admin/workspace_admin
  |-- denied for normal users
```

The client can hide dashboard entry buttons, but the server owns the actual authorization decision. `server/app/admin_scope.go` resolves admin scope from the authenticated session and emits structured allow/deny diagnostics. Slice-specific scope checks decide whether the caller can reach customers, chats, providers, ops, or business surfaces.

## Workspace Admin vs Superuser

Workspace-admin surfaces are scoped views over a workspace and its users. They are appropriate for customer, workspace, support, and conversation operations within that workspace boundary.

Superuser surfaces are platform-wide control planes. Business, provider, ops, pricing, feature flags, incident, and cross-workspace controls require superuser scope because they can affect many customers or the whole runtime.

This distinction exists in both the UI and server. The UI labels route availability and limits links. The server filters rows by scope and rejects disallowed surfaces even when a caller crafts an RPC manually.

## Shared Slice Pattern

```text
dashboard route
  |
  v
client/app/dashboard_shell.go
  |-- pick slice from CurrentPath
  |-- render shared top bar and wrapper
  |-- load typed data through controllers
  v
server typed RPC
  |-- require admin scope
  |-- run typed query/read helper
  |-- map rows to proto response
  v
summary cards + trend blocks + tables + detail routes
```

The home route summarizes account/admin context and links to slices. Business, Customers, Chats, Providers, and Ops use the same basic structure: shared route wrapper, scoped async data load, compact summary, drill-down table, and explicit empty/loading/error states.

## Typed RPC Boundaries

Admin surfaces should not scrape client state or reuse customer chat RPCs. They call typed admin RPCs with explicit filters, limits, sort keys, route context, and target identifiers. Server responses map SQL/store rows into proto view models so the UI can render without knowing schema details.

This boundary is why dashboard code can preserve filters and time ranges during drill-down navigation: the route owns filter state, the RPC owns typed data shape, and the server owns scope enforcement.

## Drill-down Philosophy

A dashboard widget should lead to evidence. Summary cards and trend blocks are useful only when paired with a table, detail drawer, or route that explains what changed. The existing dashboard work follows this by pairing cards with customer, conversation, usage, support, incident, provider, or ops detail reads.

Drill-down routes must preserve list context. Back links should return to the same slice, filters, and time range rather than resetting the operator's investigation.

## Mutation Paths

Mutations are more guarded than reads:

1. Parse the target and action into a typed mutation intent.
2. Resolve admin scope and mutation-specific authorization.
3. Require confirmation and non-empty reason for destructive actions.
4. Apply operational effects, such as revoking sessions or suspending workspace access.
5. Write an audit row with request and correlation metadata.
6. Emit structured success or rollback-required diagnostics.

`server/app/admin_mutation_effects.go` is the representative pattern. Provider, ops, chat, billing, and customer mutations should follow the same confirmation, authorization, side-effect, and audit shape.

## Operator Diagnostics

The subsystem logs role resolution, slice denials, dashboard bootstrap, list/detail loads, mutation submit/confirm/success, and rollback-required failures. Logs should include admin slice, target scope, admin user id where available, request id, correlation id, and next-action hints. Customer-safe UI copy should not expose raw secrets, upstream payloads, or internal stack detail.
