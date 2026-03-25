# GWC Layout

`tools/gwc` stays as one Go command package so the launcher can keep package-private helpers and tests without adding unnecessary export surface. The large `main.go` file has been split so that command ownership is easier to find:

- `main.go`: process entrypoint, command dispatch, shared launcher wiring, and the remaining cross-command helpers.
- `bench.go`: benchmark discovery, execution, scoring, and JSON report generation.
- `dev.go`: `gwc dev` flag parsing, plan resolution, and livereload handoff.
- `doctor.go`: `gwc doctor` checks, audit flow, and report rendering.
- `examples.go`: `gwc examples` catalog generation and example shell rendering.
- `files.go`: project file inventory and JSON reporting.
- `import.go`: static HTML/JSX/TSX import pipeline.
- `serve.go`: static fixture serving and runtime asset delivery.
- `start.go`: scaffold metadata, `gwc start`, and scaffold generation helpers.
- `*_tui.go`: Bubble Tea terminal UIs used by `start`, `dev`, and dashboard flows.
- `enterprise_*.go`: enterprise runner config and plugin/runtime integration helpers.

When adding a new `gwc` feature, prefer a clearly named file next to the owning command instead of growing `main.go`. If the command needs a dedicated workflow or data model, keep its types and helpers in that same file unless they are truly shared across multiple commands.
