# Runner Config

Location: `tools/runnerconfig/`

This package owns loading and resolving `gwc-runner.json` configuration used by launcher workflows.

## File Layout

- `schema.go`: config model and public shape
- `defaults.go`: default values applied when config is omitted
- `load.go`: file loading and decode flow
- `resolver.go`: root-relative path resolution and normalization
- `core.go`: shared helpers used by load and resolve paths

## When To Change It

- Edit this package when `gwc` needs a new runner-level setting or a new defaulting rule.
- Keep path normalization here instead of duplicating it inside `tools/gwc/`.
