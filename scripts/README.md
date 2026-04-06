# Scripts

Location: `scripts/`

This folder is for repo-level helper scripts that do not belong in a Go package.

## File Layout

- `bootstrap-gogrpcbridge.ps1`: helper for bootstrapping the `third_party/GoGRPCBridge` submodule and related local setup

## Placement Rules

- Put cross-platform or user-facing automation in `tools/gwc/` first.
- Keep `scripts/` for one-off repo helpers, bootstrap steps, or maintenance tasks that do not justify a first-class Go command.
- If a script becomes part of the normal developer workflow, promote it into `tools/` and leave only compatibility shims here.
