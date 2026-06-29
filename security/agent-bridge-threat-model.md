# Agent Bridge Threat Model And Checklist

The GWC live agent bridge is a local development and CI-dogfood control plane.
It lets an MCP client or CLI command inspect and mutate a running wasm app
through the agent hub, so treat it as CDP-equivalent: a token leak is full app
control for that dev session.

## Scope

In scope:

- `gwc dev` and livereload-served pages that opt into agent mode
- the local agent hub HTTP and WebSocket routes under `/__gwc-agent` and
  `/gwc-agent`
- live bridge tools exposed by `gwc mcp`, including sessions, snapshot, query,
  set-atom, emit, publish, navigate, and snapshot-diff
- headless Playwright-Go dogfood flows that connect an app to a real hub

Out of scope:

- production or release-profile runtime control
- remote multi-tenant agent service hosting
- browser extension or CDP security beyond the local process boundary

## Assets

- app state reachable through atoms, fiber refs, events, and router state
- bridge token minted for one dev or CI run
- hub session metadata, command history, snapshots, logs, and diagnostics
- browser-local credentials, cookies, localStorage, sessionStorage, and IndexedDB
  that the app can already access
- CI secrets in the process environment and checkout

## Trust Boundaries

- Browser page to hub: local WebSocket/HTTP. The hub must require the run token
  on every agent route and reject foreign origins.
- CLI/MCP client to hub: local HTTP. The caller is trusted only if it has the
  ephemeral token and can reach the loopback bind.
- Hub to app session: per-session ordered command relay. Frames for one session
  must never reach another.
- Release artifacts: production and release profiles must not contain the bridge
  runtime, bootstrap script, token injection, or command registration.

## Required Controls

- Bind the hub to loopback only. Non-localhost binds are a security finding.
- Mint a high-entropy token per run. Do not check it into files, logs, docs, or
  fixtures.
- Require the token for every sessions, command, and WebSocket route, not only
  the initial hello.
- Accept agent mode only when the page was launched with `?gwc-dev=agent` and
  the wasm build includes the explicit `gwcagent` tag.
- Keep the non-agent path normal: the same app without `?gwc-dev=agent` opens no
  agent socket and serves normally.
- Redact snapshots, logs, diagnostics, and crash reports before they cross the
  hub boundary.
- Keep command payloads schema-shaped and fail closed on unknown commands,
  malformed JSON, stale refs, and type mismatches.
- Refuse release-profile bridge artifacts with a string and symbol scan.
- Bound logs, diagnostics, events, command history, snapshots, and crash reports;
  report dropped records.

## Checklist

Before merging live bridge changes:

- `go run ./tools/gwc test -lane agent -json` passes or the failing package is
  explained.
- `go run ./tools/gwc test -lane agent-browser -json` passes when the dogfood
  test exists, or reports an explicit skipped lane on checkouts that do not yet
  include it.
- The headless recipe launches the app through `gwc dev`, opens the page with
  `?gwc-dev=agent`, connects with the run token, and drives at least one real
  query/read plus one mutating command.
- The no-query-param path is tested: no socket opens and the app remains usable.
- A reload links successor sessions instead of losing the chain.
- Logs and snapshots are reviewed for secrets, cookies, bearer tokens,
  passwords, provider keys, and raw customer content.
- CI uses an ephemeral token and tears down browsers, servers, and temp
  databases after the lane.

## Incident Response

If a bridge token is exposed, assume the holder can inspect and mutate that app
session. Stop the dev server or CI job, rotate any app credentials used in that
session, delete captured snapshots/logs if they may contain sensitive data, and
re-run the security checklist before re-enabling the bridge.
