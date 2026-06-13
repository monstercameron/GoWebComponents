# Systems Glossary

This appendix defines repo-local terms used throughout Example 100.

| Term | Meaning |
|---|---|
| Shell | The single Go WASM application container that renders public, auth, authenticated workspace, settings, dashboard, thread, and canvas surfaces based on route and state. |
| Public shell | The same app shell when it is serving unauthenticated marketing/auth routes such as `/`, `/home`, `/pricing`, `/signup`, and `/login`. |
| Authenticated shell | The private workspace runtime rendered after session validation, including sidebar, thread, composer, settings, dashboard, and canvas regions. |
| Boot shell | The temporary first-render state shown while WASM, auth resolution, catalogs, and runtime readiness are still settling. |
| Bootstrap | Initial data or state required before a route can safely render, such as auth session, boot i18n catalogs, model catalog, or conversation route resolution. |
| Hydration | Browser-side WASM taking over the shell response and rendering the correct route surface. |
| Slice | A dashboard section with its own route, data load, summaries, tables, detail views, and scope rules. Business, Customers, Chats, Providers, and Ops are slices. |
| Shared slice pattern | The repeated dashboard structure: route activation, typed RPC, server scope check, view-model mapping, summary plus drill-down render, and explicit loading/error states. |
| Worker task | A background WASM operation used for markdown/render metadata or artifact processing so the main shell stays responsive. |
| Provider snapshot | A normalized provider health, model, capability, quota, and status view consumed by admin/provider surfaces. |
| Provider registry | Server runtime object that resolves provider ids, model ids, capabilities, health, and stream adapters behind one interface. |
| Model catalog | The selectable provider/model metadata exposed to the client and admin surfaces, backed by configured providers and SQL catalog rows. |
| Control plane | Superuser/admin surfaces and RPCs that change operational state, such as provider toggles, feature flags, pricing, incidents, site config, and user/workspace status. |
| Scoped admin view | A dashboard/admin read path filtered to the caller's role and workspace scope. Workspace admins see only scoped data; superusers can see platform-wide data. |
| Public conversation route | The replayable thread URL `/app/thread/:publicID`, where `publicID` hides the internal database id. |
| Route normalization | Client behavior that replaces draft or mismatched routes with the canonical route once the server resolves a conversation public id. |
| Post-login route intent | A safe private route saved before authentication and consumed once after successful login/signup/session bootstrap. |
| Customer-safe error | Friendly UI error copy plus support/request reference, with sensitive internal/provider detail kept in logs. |
| Operator-safe diagnostic | Structured log or audit metadata useful for debugging without leaking raw secrets to customer-facing surfaces. |
| Support ID | A stable customer-visible reference that lets operators find the matching server/client event. |
| Request ID | Identifier for one request/RPC or failure event. |
| Correlation ID | Identifier that links multiple events across browser, bridge, server, and provider calls. |
| Bridge | The GoGRPCBridge WebSocket tunnel at `/socket` that carries typed gRPC traffic between the WASM client and Go server. |
| Tunnel lifecycle | The connect, ready, disconnect, reconnect, and degraded states of the bridge. |
| Server-owned catalog | I18n copy served by backend/catalog infrastructure rather than embedded permanently in client code. |
| Route-owned async state | Data loaded because a route or panel became active, such as settings, billing, dashboard slice data, or conversation detail. |
| Durable product state | Data that must survive refresh/session changes and therefore belongs in SQLite-backed stores, not only in localStorage or in-memory app state. |
