# GWC Port Todo (Codex CLI)

This list tracks migration of remaining shell-script features into `go run ./tools/gwc ...`.

## Todo

- [x] Kickoff port: add `gwc bench compare`, add simple benchmark score graphs, and add signed-out top "Open chat" entry in RelayDesk auth shell.
- [x] Port the legacy bench-runtime wrappers into `gwc` as a raw benchmark capture mode.
- [x] Port the legacy wasm measurement wrappers into `gwc` as a wasm measurement command.
- [x] Port the legacy manifest-comparison wrappers into `gwc` as a manifest comparison command with thresholds.
- [x] Port the legacy compression-comparison wrappers into `gwc` as a compression variant comparison command.
- [x] Port the legacy cache-topology wrappers into `gwc` as a cache-topology comparison command.
- [x] Port the legacy toolchain-comparison wrappers into `gwc` as a toolchain comparison command.
- [x] Update docs to make `gwc` the primary interface and mark wrappers deprecated.
- [x] Add custom `-binary-name` support to `gwc wasm compare-toolchain` for full wrapper parity.
- [x] Add strict byte-level parity checks for optimized wasm compression variants.
- [x] Add positional baseline/candidate shortcuts for `gwc bench compare`.

## Checkpoints

### 2026-03-25 12:15 -04:00

- completed todo: Kickoff port: add `gwc bench compare`, add simple benchmark score graphs, and add signed-out top "Open chat" entry in RelayDesk auth shell.
- files changed: `tools/gwc/bench.go`, `tools/gwc/bench_test.go`, `tools/gwc/main.go`, `examples/100-ai-chat-wizard/client/app/auth_shell.go`, `examples/100-ai-chat-wizard/client/app/i18n.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunBenchmarkCompare|TestBenchmarkScoreGraph|TestRunBenchmarkWritesJSONReport" -count=1`; `GOOS=js GOARCH=wasm go test -c -o ./bin/examples-100-ai-chat-wizard-client-app.test ./examples/100-ai-chat-wizard/client/app`
- result: Passed. `gwc bench compare` now covers benchstat compare parity, `gwc bench` now emits simple inline score bars, and the signed-out auth shell now exposes a top "Open chat" action.
- residual risk: `gwc bench compare` currently requires explicit `-baseline` and `-candidate` flags (it does not yet support positional args), and there is no browser-level visual regression for the new auth-shell top action.
- next suggested todo: Port the legacy bench-runtime wrappers into `gwc` as a raw benchmark capture mode.

### 2026-03-25 12:21 -04:00

- completed todo: Port the legacy bench-runtime wrappers into `gwc` as a raw benchmark capture mode.
- files changed: `tools/gwc/bench.go`, `tools/gwc/bench_test.go`, `tools/gwc/main.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunBenchmarkCapture|TestRunBenchmarkCompare|TestBenchmarkScoreGraph|TestRunBenchmarkWritesJSONReport" -count=1`
- result: Passed. `gwc bench capture` now runs raw `go test -bench` samples and writes timestamped output files with optional `-exec`, matching the old script surface.
- residual risk: The command currently keeps script-style explicit flags and does not yet accept positional baseline/candidate shortcuts for compare mode.
- next suggested todo: Port the legacy wasm measurement wrappers into `gwc` as a wasm measurement command.

### 2026-03-25 12:24 -04:00

- completed todo: Port the legacy wasm measurement wrappers into `gwc` as a wasm measurement command.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/main.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmMeasure|TestRunWasmRejectsUnknownSubcommand" -count=1`
- result: Passed. `gwc wasm measure` now builds wasm artifacts, captures phase timings, emits gzip/brotli sidecars, and writes a structured measurement manifest.
- residual risk: Toolchain/version capture and build execution currently assume the configured Go executable is directly invocable without shell wrappers.
- next suggested todo: Port the legacy manifest-comparison wrappers into `gwc` as a manifest comparison command with thresholds.

### 2026-03-25 12:28 -04:00

- completed todo: Port the legacy manifest-comparison wrappers into `gwc` as a manifest comparison command with thresholds.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmMeasure|TestRunWasmCompare|TestRunWasmRejectsUnknownSubcommand" -count=1`
- result: Passed. `gwc wasm compare` now flattens numeric metrics, applies timing/size/other thresholds, emits structured summaries, and exits non-zero when regressions exceed thresholds.
- residual risk: Metric categorization still relies on path-name heuristics (`*_ms`, `*bytes`) and can misclassify unusual manifest shapes.
- next suggested todo: Port the legacy compression-comparison wrappers into `gwc` as a compression variant comparison command.

### 2026-03-25 12:35 -04:00

- completed todo: Port the legacy compression-comparison wrappers into `gwc` as a compression variant comparison command.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmCompareCompression|TestRunWasmMeasure|TestRunWasmCompare|TestRunWasmRejectsUnknownSubcommand" -count=1`
- result: Passed. `gwc wasm compare-compression` now builds plain/stripped/compressed variants, conditionally generates optimized variants when `wasm-opt` is available, and writes a structured comparison summary.
- residual risk: Optimized variant parity currently depends on filesystem-level output transforms and does not yet assert byte-for-byte parity with shell-wrapper artifacts.
- next suggested todo: Port the legacy cache-topology wrappers into `gwc` as a cache-topology comparison command.

### 2026-03-25 12:39 -04:00

- completed todo: Port the legacy cache-topology wrappers into `gwc` as a cache-topology comparison command.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmCompareCache|TestRunWasmCompareCompression|TestRunWasmMeasure|TestRunWasmCompare|TestRunWasmRejectsUnknownSubcommand" -count=1`
- result: Passed. `gwc wasm compare-cache` now runs shared-cache, isolated-cache, and CI-style cache variants, captures small-edit rebuild behavior, and writes a structured cache comparison summary.
- residual risk: Variant failures currently report summary status and return an overall error, which is stricter than one of the legacy wrapper flows.
- next suggested todo: Port the legacy toolchain-comparison wrappers into `gwc` as a toolchain comparison command.

### 2026-03-25 12:42 -04:00

- completed todo: Port the legacy toolchain-comparison wrappers into `gwc` as a toolchain comparison command.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmCompareToolchain|TestRunWasmCompareCache|TestRunWasmCompareCompression|TestRunWasmMeasure|TestRunWasmCompare|TestRunWasmRejectsUnknownSubcommand" -count=1`
- result: Passed. `gwc wasm compare-toolchain` now measures baseline/candidate toolchains, runs manifest threshold comparison, writes toolchain summaries, and propagates regression exit semantics.
- residual risk: The port currently defaults toolchain build output to `app.wasm`; wrapper parity for a custom binary name in this command is not yet exposed.
- next suggested todo: Update docs to make `gwc` the primary interface and mark wrappers deprecated.

### 2026-03-25 12:44 -04:00

- completed todo: Update docs to make `gwc` the primary interface and mark wrappers deprecated.
- files changed: `docs/GWC.md`, `docs/BUILD_EXPERIMENTS.md`, `docs/TROUBLESHOOTING.md`, `tools/README.md`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmCompareToolchain|TestRunWasmCompareCache|TestRunWasmCompareCompression|TestRunWasmMeasure|TestRunWasmCompare|TestRunWasmRejectsUnknownSubcommand|TestRunBenchmarkCapture|TestRunBenchmarkCompare" -count=1`
- result: Passed. Launcher docs now present `gwc wasm` as the primary experiment interface and explicitly mark legacy `tools/*.ps1` and `tools/*.sh` wrappers as deprecated compatibility shims.
- residual risk: Wrapper scripts still exist in-tree for compatibility and may remain referenced by local-only contributor notes outside the tracked docs set.
- next suggested todo: none.

### 2026-03-25 12:52 -04:00

- completed todo: Add custom `-binary-name` support to `gwc wasm compare-toolchain` for full wrapper parity.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmCompareToolchain" -count=1`
- result: Passed. `gwc wasm compare-toolchain` now accepts `-binary-name` and uses that artifact name for baseline/candidate measurement outputs.
- residual risk: Toolchain comparison summary still references manifest paths only and does not expose artifact filenames directly.
- next suggested todo: Add strict byte-level parity checks for optimized wasm compression variants.

### 2026-03-25 12:54 -04:00

- completed todo: Add strict byte-level parity checks for optimized wasm compression variants.
- files changed: `tools/gwc/wasm.go`, `tools/gwc/wasm_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunWasmCompareCompression" -count=1`
- result: Passed. Optimized compression flows now enforce byte-level raw-to-delivery wasm parity checks and expose parity details in the summary payload.
- residual risk: Parity checks currently validate optimized raw and optimized delivery wasm artifacts, but do not enforce parity expectations for compressed sidecars by design.
- next suggested todo: Add positional baseline/candidate shortcuts for `gwc bench compare`.

### 2026-03-25 12:55 -04:00

- completed todo: Add positional baseline/candidate shortcuts for `gwc bench compare`.
- files changed: `tools/gwc/bench.go`, `tools/gwc/bench_test.go`, `tools/gwc/CODEX_PORT_TODO.md`
- validation run: `go test ./tools/gwc -run "TestRunBenchmarkCompare" -count=1`
- result: Passed. `gwc bench compare` now accepts positional `baseline candidate` paths and one-path positional fallback when one flag is already provided.
- residual risk: Ambiguous combinations with both paths already supplied via flags and extra positional args now fail fast, which is intentionally stricter than silent fallback behavior.
- next suggested todo: none.
