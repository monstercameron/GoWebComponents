# Router Blues

This note tracks the recurring route-sync failures in Example 100 where chat state, dashboard state, and browser paths drift out of agreement.

## Current Rule

Conversation route normalization must only run on true chat surfaces:

- `/app`
- `/app/thread/:publicID`
- `/app/thread/:publicID/canvas/:canvasID`

It must not run on:

- `/app/dashboard`
- `/app/dashboard/*`
- `/app/settings`
- public landing routes

## Known Failure Mode

When one conversation is active and the user navigates to an admin dashboard route, any "active conversation route normalization" that treats the dashboard like `/app` will immediately replace the dashboard URL with the active thread URL.

That shows up as:

- dashboard page flashes briefly
- browser returns to the chat thread
- admin route looks unreachable even though router registration is correct

## Guardrails

- Route-sync helpers must take the current path, not just the active conversation state.
- Dashboard routes are independent top-level app surfaces, not chat-root aliases.
- "Reset draft on root route" logic must only run on the exact `/app` route.
- "Normalize active conversation route" logic must only run on `/app/thread/*` and `/app`.

## Recent Fix

The current fix scopes conversation normalization and root-route draft reset so they ignore dashboard routes entirely. See:

- `client/app/route_sync.go`
- `client/app/app.go`
- `client/app/route_sync_test.go`

## Dashboard Tile Investigation Trap

Another failure mode looked like a router/runtime regression but was actually a stale served client artifact.

Observed behavior:

- source code in `client/app/dashboard_shell.go` rendered dashboard tiles as `button` elements with router-driven `OnClick(...)`
- live browser DOM still showed `<a href="/app/dashboard/...">` tiles
- clicking those tiles triggered full browser navigation, app restart, and fallback to `/app`

What made this confusing:

- browser cache was disabled during QA, so this did not look like a normal cache problem
- server HTML was not the source of the bad anchor markup; the stale markup came from the executing client bundle
- raw `chat.wasm` had been rebuilt, but `bin/client/app/chat.wasm.br` was older and the server preferred the Brotli sidecar by default

Why this belongs in the router note:

- from the browser, the symptom is indistinguishable from "framework router failed and browser navigation won"
- the UI looked like the component tree was still generating anchors even though the checked-in source had already been changed
- this kind of asset-freshness drift can mask real router bugs or send investigation down the wrong layer

Research follow-up for core/framework work:

- add a framework-level story for SPA-critical navigation controls so browser regressions assert host tag type plus no-reload route transitions
- add better runtime/build diagnostics for served wasm freshness, especially when `.wasm.br` is older than `.wasm`
- consider whether the dev/runtime infrastructure should fail closed or warn loudly when compressed sidecars are stale relative to raw artifacts
- keep investigating whether router-managed navigation should expose stronger debug markers so "SPA transition" versus "full document navigation" is easier to prove during QA

Example 100 mitigation:

- server now skips stale Brotli sidecars and falls back to raw `.wasm`
- focused Playwright coverage was added for dashboard-home tile navigation so this exact regression has one browser-level repro path
