# Internal Platform

Location: `internal/platform/`

This folder holds the runtime-facing platform adapters behind the public `ui` API.

## Folder Layout

- `jsdom/`: browser-backed `js/wasm` adapter implementations and wasm-side contract tests
- `mockdom/`: native mock DOM and scheduler used by runtime tests

## Start Here

- Read `jsdom/` when you are changing real browser DOM, browser state, or scheduler integration.
- Read `mockdom/` when you are changing native test behavior, selector lookup, or runtime fixture expectations.
- Keep platform contracts aligned with `internal/runtime/interfaces.go`.
