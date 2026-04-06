# Third Party

Location: `third_party/`

This directory contains pinned external code or tool payloads that the repo depends on for development workflows.

## Current Layout

- `GoGRPCBridge/`: git submodule used by the example and tooling stack for typed browser RPC transport
- `tailwindcss/`: cached Tailwind CLI payload used by repo tooling
- `_shared/`: shared external data files consumed by local tooling

## How To Use It

- treat these directories as owned by their upstream projects or by explicit repo tooling contracts
- avoid editing submodule contents casually from the main repo workflow
- prefer the upstream or canonical docs inside each dependency when you need implementation details

Useful entry docs:

- [GoGRPCBridge/README.md](GoGRPCBridge/README.md)
- [GoGRPCBridge/docs/core/GOGRPCBRIDGE_SUBMODULE_LIFECYCLE.md](GoGRPCBridge/docs/core/GOGRPCBRIDGE_SUBMODULE_LIFECYCLE.md)
- [../tools/README.md](../tools/README.md)

## Maintenance Rule

Anything added under `third_party/` should have one clear reason:

- submodule
- vendored build tool payload
- shared external dataset used by tooling

If it does not fit one of those, it probably belongs somewhere else.
