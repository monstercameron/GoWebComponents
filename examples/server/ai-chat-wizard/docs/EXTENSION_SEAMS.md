# Extension Seams

Use this chapter when adding new functionality without breaking Example 100's teaching value. The safest changes extend an existing pattern instead of creating a new one-off path.

## Add A Provider

1. Implement the provider interface under `server/provider`.
2. Normalize model metadata, capabilities, pricing, rate-limit state, and health snapshot behavior.
3. Add model-catalog rows or catalog source support.
4. Apply traceability headers for upstream HTTP calls.
5. Ensure unsupported capabilities return typed provider errors.
6. Add parity coverage for streaming, memory extraction where relevant, TTS where relevant, and admin provider snapshots.

The client should not learn provider-specific streaming protocols. It should receive normalized model options and `ChatChunk` streams.

## Add A Dashboard Slice

1. Add a route constant in `client/app/routes.go`.
2. Register the route in the router.
3. Extend `dashboard_shell.go` to dispatch the slice.
4. Add a typed controller/data load in the client.
5. Add proto messages and RPCs for the slice.
6. Enforce scope in `server/app/admin_scope.go`.
7. Back reads with typed SQL/store helpers.
8. Pair summary cards with a table, drawer, or detail route.
9. Add mutation confirmation, side effects, and audit logging if the slice can change state.

Workspace-admin slices must filter rows to the caller's scope. Platform-wide slices must require superuser scope.

## Add A Route

1. Name the route in `client/app/routes.go`.
2. Register it in `client/app/app.go`.
3. Decide whether direct loads need server shell delivery in `server/app/server.go`.
4. Add route-owned state load and empty/loading/error branches.
5. Preserve auth handoff and post-login route normalization if the route is private.
6. Add browser regression coverage for direct load, refresh, in-app navigation, and back/forward behavior.

Routes should not bypass the single-shell pattern unless the file is truly a static asset or health endpoint.

## Add A Worker Task

1. Define the task payload and result type near existing worker render types.
2. Add worker-side handling in `client/backgroundworker`.
3. Add main-thread controller code that can tolerate worker unavailability.
4. Cache results by stable route/message/artifact keys when recomputation would be disruptive.
5. Add diagnostics for invalid input, task failure, and worker lifecycle failure.

Worker tasks should improve responsiveness or isolation. They should not own durable product state.

## Add A Settings Section

1. Add route/query state through the settings route helpers.
2. Add a typed load and save RPC when the setting is account or workspace data.
3. Keep local-only state only for device/layout preferences.
4. Render explicit loading, denied, dirty, saved, and failed states.
5. Reset account-scoped state on logout/account switch.
6. Add focused tests for persistence, validation, stale-state handling, and auth failure.

## Add An RPC

1. Define request/response messages in `proto/chat.proto`.
2. Regenerate proto stubs.
3. Add server implementation with session, authz, validation, store access, and customer-safe error behavior.
4. Add client call site and route-owned state update.
5. Add logs with request/correlation context where the RPC can fail operationally.
6. Add server tests for success, auth failure, validation failure, scope failure, store failure, and any policy branch.

New RPCs should preserve typed boundaries. Do not tunnel arbitrary JSON around the proto contract just to move faster.

## Keep The Teaching Value

Every extension should make the system easier to reason about:

- reuse the shell route model
- reuse typed RPC boundaries
- reuse provider normalization
- reuse dashboard slice patterns
- reuse store/query separation
- reuse customer-safe error and operator diagnostic conventions
- document shipped behavior separately from planned future state
