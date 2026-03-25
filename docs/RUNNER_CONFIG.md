# Runner Config Schema

`gwc-runner.json` is the single external runner-configuration contract for:

- the Go launcher under `tools/gwc`
- the launcher-owned browser and test flows that resolve workspace paths through the shared runner config
- the nested livereload server under `tools/livereload`

Use [gwc-runner.example.json](examples/gwc-runner.example.json) as the canonical example. Relative paths are resolved from the directory that contains the config file, whether the file is discovered as `gwc-runner.json`, pointed to by `GWC_RUNNER_CONFIG`, or loaded from the home-level fallback at `.gwc/runner.json`.

The canonical `paths` schema is:

- `generatedProjectRoot`: default output location for generated starter apps.
- `artifactRoot`: root directory for launcher-owned artifacts under a per-workspace namespace.
- `workspaceBuildRoot`: shared workspace build root for repo-owned build outputs.
- `wasmExecJS`: override for `wasm_exec.js`.
- `goWasmExec`: override for the js/wasm test executor.
- `browserWorkspace`: browser-test workspace for launcher browser lanes (`playwrightgo` under this workspace, or `test/playwrightgo` when using a repo root workspace).
- `livereloadWorkspace`: nested livereload workspace used by `gwc dev`.
- `livereloadClientScript`: optional override for replacing the embedded livereload client served by `gwc dev`.

Schema ownership rules:

- Add new runner-owned path fields in the shared Go `tools/runnerconfig` package first.
- Keep launcher-owned consumers aligned with the same field names and relative-path semantics.
- Validate the canonical example from each consumer surface so schema drift is caught by tests before release.

Ownership guidance:

- Put organization-wide enforcement, trusted executable roots, inherited environment policy, and company-standard runner paths in a centrally managed config distributed through `GWC_RUNNER_CONFIG` or a shared home-level `.gwc/runner.json`.
- Put project-local workspace layout choices in checked-in `gwc-runner.json` only when the whole repository needs the same browser workspace, livereload workspace, or artifact policy for every contributor.
- Keep short-lived experimentation, one-off output destinations, and local debugging needs on explicit command flags instead of committing them into runner config.
- Prefer the org-managed layer for security and compliance rules, the project layer for reproducible repository layout, and CLI flags for temporary operator intent.
