# Authenticated Shell

The authenticated shell is the runtime container for `/app`, `/app/thread/:publicID`, settings, dashboard routes, and canvas routes. It starts after auth resolves and the client has enough runtime state to render private surfaces.

## Route Ownership

`client/app/routes.go` names the route contract. `client/app/app.go` registers every route with the history router and routes them all through `parseChatWizardRoot`. `renderWorkspaceShell` in `client/app/app_shell.go` then chooses the visible private surface from `appViewState.CurrentPath`.

The route owner is the client shell, not an individual page file. Settings, dashboard, chat, and canvas are route-specific regions inside one app state graph. This is why refresh and in-app navigation must preserve authenticated shell state instead of tearing down to a public/auth shell during transient bridge loss.

## Shell Composition

```text
appState
  |
  v
parseDeriveAppViewState
  |
  |-- auth and role flags
  |-- active conversation and route ids
  |-- model, tone, thinking, TTS preferences
  |-- settings inputs and account summary
  |-- dashboard data and admin tools data
  |-- canvas session state
  v
renderWorkspaceShell
```

The shell composes persistent chrome plus route surfaces: sidebar, top controls, thread panel, composer, settings panels, dashboard, and canvas. `appViewState` is the read model for rendering; actions dispatch back into `appState` through typed app actions and controllers.

## Persistent Preferences

Preferences have different owners:

| Preference | Owner | Reason |
|---|---|---|
| Auth token | localStorage plus server session validation | Browser persistence with server authority. |
| Post-login route intent | localStorage, consumed once | Allows auth handoff without keeping stale redirects. |
| Selected model, tone, thinking | RPC-backed profile state plus local rendering state | User preference should survive sessions and refresh. |
| Sidebar/layout/scroll memory | localStorage or in-memory browser state | Device/layout convenience, not durable product data. |
| Settings/profile/memory rows | SQLite-backed RPCs | Account data must follow the authenticated user. |

Account-scoped data must reset on account switch or logout. Device-level preferences can remain local only when they do not reveal or reuse another account's product data.

## Model Selection

Model selection starts from the catalog returned by typed RPCs such as `ListModelOptions`, then resolves provider/model choices in the client preference controller. The visible provider filter is client-derived from catalog metadata, while final send-time provider and model capability checks happen on the server through the provider registry. The client can suggest a model; the server decides whether it is available, entitled, and capability-compatible.

## Cross-tab Sync

The shell uses browser storage events for cross-tab preference sync where local state is the right owner, such as sidebar/layout style state. Server-owned preferences still need RPC reads/writes so a second tab or later login can observe persisted account state. The rule is to synchronize UI convenience locally, but not to invent a local-only source of truth for account, billing, auth, or admin state.

## Worker Usage

The background worker under `client/backgroundworker/` runs WASM tasks for markdown/render metadata and artifact-oriented processing. The main shell sends render work to the worker and stores the result in caches keyed by message so thread rendering does not block the main UI path. If the worker is unavailable, the app should degrade through visible diagnostics rather than corrupting thread state.

## gRPC Tunnel Lifecycle

The gRPC tunnel is carried by WebSocket at `/socket` through GoGRPCBridge. Client runtime state tracks whether the bridge is ready. Auth/session bootstrap, model catalog loads, conversation list fetches, settings saves, dashboard loads, and chat sends all depend on typed RPCs over this tunnel.

Reconnect behavior is intentionally visible. During transient tunnel loss, the app should keep the authenticated route and shell composition stable, surface actionable reconnect state, and retry route-owned data where appropriate. It should not collapse back to public/auth shells unless auth itself is invalid or revoked.
