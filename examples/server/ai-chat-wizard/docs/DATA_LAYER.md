# Data Layer

The data layer is the SQLite-backed runtime truth for Example 100. It is designed to be readable as a teaching schema, not hidden behind a generic ORM.

## Schema Ownership

`sql/store/schema.sql` owns durable table definitions. The major groups are:

| Group | Tables and purpose |
|---|---|
| Auth | `users`, `auth_sessions`, `auth_token_versions`, `auth_identities`, `auth_oidc_states`, password reset and email verification tables. |
| Workspace | `workspaces`, `workspace_memberships`, invitations, SSO config and auth policy tables. |
| Chat | `conversations`, `messages`, `usage_events`, prompt/library/workflow tables, onboarding and activation tables. |
| Billing | plans, entitlements, model access, customers, subscriptions, invoices, line items, overrides, quota and dunning tables. |
| Admin/Ops | superuser roles, site config, feature flags, server tool policy history, incidents, SLOs, background jobs, notifications, webhooks. |
| Support/Audit | `support_tickets`, support messages, `audit_logs`, compliance and retention records. |
| Provider reporting | model catalog, provider usage rollups, routing policy, and cost guardrail tables. |

The schema should describe durable ownership and relationships. It should not become the only place where product policy lives.

## Query Loading

SQL statements live under `sql/store/` and are loaded by Go helpers through the example's SQL file loader. Query files are grouped by operational concern, for example `ops`, `billing`, and `admin` reads/writes. This keeps SQL reviewable and lets tests point at concrete query files.

The server maps SQL results into typed runtime structs, then into proto responses. Client code should not know table names.

## Store Boundaries

Store helpers are responsible for persistence mechanics: running a query, scanning rows, normalizing nullable fields, and returning typed Go values. Service and RPC handlers are responsible for caller validation, authorization, product policy, and customer-safe error mapping.

Keep these boundaries clear:

| Concern | Belongs in |
|---|---|
| Unique indexes, foreign keys, durable columns | `schema.sql` |
| Select/insert/update shape | `sql/store/*.sql` |
| Row scanning and transactional persistence | `server/app/store*.go` helpers |
| Authz, entitlement, provider capability, mutation confirmation | Go service/RPC policy code |
| UI labels, dashboard grouping, route state | Client app code |

## Migration Expectations

Example 100 uses a local SQLite database. Schema changes must be coordinated with seed data, store helpers, tests, and docs. A new table is not complete until it has a clear owner, indexes for expected reads, query files or helpers, seed data when needed for local QA, and a documented policy boundary.

For destructive or shape-changing migrations, preserve local operator instructions in README/runbook material so maintainers know whether to recreate the demo DB or run a migration path.

## Seed Data

`cmd/seed-test-db` creates deterministic local accounts:

| Account | Purpose |
|---|---|
| `customer@email.com / password` | Fast customer login and first-chat/browser QA path. |
| `admin@email.com / password` | Admin/superuser journey coverage and dashboard testing. |

The seed command also writes profile preferences, billing customer records, and realistic conversation history. That lets browser tests, manual smoke, billing summaries, and dashboard reads run against coherent product data instead of empty tables.

## Application Policy vs Persistence

Persistence records facts: a user exists, a session expires, a conversation has messages, a usage event has provider/model/cost metadata, an audit row records an operator action.

Application policy decides what those facts mean: whether password login is allowed under workspace SSO policy, whether a user can send with a model, whether a workspace admin can view a row, whether a provider supports thinking, or whether an admin mutation requires confirmation. Keep policy in Go so it can be unit tested, logged, and changed without hiding behavior inside ad hoc SQL branches.
