# Agentic Live Bridge

The GWC live bridge lets an agent drive a real browser session during local
development or headless CI. It is a development and dogfood surface, not a
production API.

Use it when:

- a browser-only failure needs a live wasm session instead of static source
  inspection
- an agent should query rendered UI by role, label, text, id, or stable ref
- a CI dogfood test should prove MCP tool -> hub -> wasm -> ack round trips
- a reload needs to preserve session lineage while an agent keeps verifying

Do not use it when:

- the app is built for release or served to end users
- a normal component, hook, router, SSR, or Playwright test can prove the same
  behavior more directly
- the command would require unredacted secrets, cookies, provider keys, or raw
  customer payloads in a snapshot

## Mental Model

```text
MCP client or gwc CLI
        |
        v
gwc mcp / gwc sessions / gwc snapshot / gwc query / mutating commands
        |
        v
loopback agent hub with ephemeral token
        |
        v
wasm app opened with ?gwc-dev=agent and built with the gwcagent tag
```

The hub owns session registration, token checks, session selection, and command
relay. The wasm app owns the actual runtime state and acks with the post-command
state version.

## Tool Catalog

- `gwc sessions`: list live sessions
- `gwc snapshot`: read a redacted runtime snapshot
- `gwc query`: find nodes by semantic selector and return stable refs
- `gwc set-atom`: set an atom value through the registered payload path
- `gwc emit`: invoke a node event handler by ref
- `gwc publish`: publish a topic event
- `gwc navigate`: drive router navigation
- `gwc snapshot-diff`: compare two snapshots by stable ref

Each CLI command has a matching MCP tool named with a `gwc_` prefix, such as
`gwc_snapshot` or `gwc_set_atom`.

## Headless E2E Recipe

Run the native bridge lane first:

```powershell
go run ./tools/gwc test -lane agent -json
```

Run the headless dogfood lane when the Playwright-Go dogfood test is present:

```powershell
go run ./tools/gwc test -lane agent-browser -json
```

The dogfood test should launch the ai-chat-wizard app through `gwc dev`, open it
with `?gwc-dev=agent`, connect to the real hub with the minted token, query the
composer, set the model atom, emit send, wait for settlement, snapshot the
thread, and assert the message appears.

It should also prove:

- the same app without `?gwc-dev=agent` opens no socket and still serves normally
- a livereload-triggered reload registers a linked successor session
- the hub tears down browser/server processes and releases ports at the end

## Security Boundary

Treat the bridge as CDP-equivalent. A leaked token gives the holder full control
over that app session.

Required controls:

- loopback-only hub bind
- one ephemeral token per run
- token required on every frame and HTTP route
- `gwcagent` tag only for development and dogfood builds
- release-profile string and symbol scans must find no bridge
- snapshots, logs, diagnostics, and crash paths are redacted before leaving the
  page or hub
- bounded buffers report dropped entries

Review the repository threat model before adding bridge verbs:
`security/agent-bridge-threat-model.md`.
