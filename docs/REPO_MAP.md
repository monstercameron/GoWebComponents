# Repository Map

Use this file when you want to know where to read first, not where every file happens to live.

## Fast Paths

### I want to use the library

Start here:

1. [START_HERE.md](START_HERE.md)
2. [../ui/README.md](../ui/README.md)
3. [../html/README.md](../html/README.md)
4. [../state/README.md](../state/README.md)
5. [../fetch/README.md](../fetch/README.md)
6. [../router/README.md](../router/README.md)

Then use:

- [WORKFLOWS.md](WORKFLOWS.md) for task-oriented guidance
- [REFERENCE_MAP.md](REFERENCE_MAP.md) for concept-to-doc and concept-to-example lookup
- [../examples/README.md](../examples/README.md) to find runnable examples by API

### I want to debug or extend the framework

Start here:

1. [../internal/README.md](../internal/README.md)
2. [../internal/runtime/README.md](../internal/runtime/README.md)
3. [../internal/platform/README.md](../internal/platform/README.md)
4. [../internal/runtime2/README.md](../internal/runtime2/README.md)

Use these when the question is narrower:

- rendering and hydration: [HYDRATION.md](HYDRATION.md), [SCHEDULING.md](SCHEDULING.md)
- worker-backed runtime work: [MULTITHREADED_RUNTIME.md](MULTITHREADED_RUNTIME.md), [MULTITHREADED_RUNTIME_TODO.md](MULTITHREADED_RUNTIME_TODO.md), [PARALLEL_REGION_AUTHORING.md](PARALLEL_REGION_AUTHORING.md)
- production guardrails: [PRODUCTION_CORRECTNESS.md](PRODUCTION_CORRECTNESS.md)

### I want to work on tooling

Start here:

1. [../tools/README.md](../tools/README.md)
2. [GWC.md](GWC.md)
3. [../tools/gwc/docs/README.md](../tools/gwc/docs/README.md)

Then branch by sub-area:

- launcher and workflows: `tools/gwc/`
- live reload and dev server plumbing: `tools/livereload/`
- config resolution: `tools/runnerconfig/`
- docs ingestion helpers: `tools/doc_ingest/`

### I want to browse examples

Start here:

1. [../examples/README.md](../examples/README.md)
2. `go run ./tools/gwc examples`

Best first examples:

- beginner app shape: `examples/01-counter`
- routing and SSR: `examples/18-ssr-server-routing`
- plugin host: `examples/99-plugin-host`
- worker regions and runtime2: `examples/108-parallel-region-basic`, `examples/109-parallel-region-grid`, `examples/110-parallel-region-diagnostics`
- production-shaped apps: `examples/86-atlas-commerce-os`, `examples/100-ai-chat-wizard`

### I want to run tests or verify behavior

Start here:

1. [../test/README.md](../test/README.md)
2. [TESTING.md](TESTING.md)
3. [../tools/README.md](../tools/README.md)

Quick split:

- package tests: `go test ./...`
- launcher-owned validation lanes: `go run ./tools/gwc test ...`
- browser harness: `test/`
- example-specific browser checks: `test/playwrightgo/examples`

## Top-Level Directory Map

- `ui/`, `html/`, `state/`, `fetch/`, `router/`: primary public package surface
- `devtools/`, `head/`, `hotreload/`, `i18n/`, `interop/`, `logging/`, `plugin/`, `prerender/`, `pwa/`, `virtualization/`, `utils/`: companion packages and browser/runtime support
- `internal/`: framework internals; do not start here unless you are changing the framework itself
- `tools/`: launcher and repo automation
- `examples/`: examples, showcase apps, and reference applications
- `test/`, `testkit/`: validation harnesses and reusable test helpers
- `docs/`: prose docs, workflows, design notes, and backlog tracking
- `agents/`, `scripts/`: local contributor and automation helpers
- `third_party/`: pinned external repos, vendored tool payloads, and shared external data
- `bin/`: ignored local runtime outputs produced by launcher-managed servers and build flows
- `log/`: ignored local logs kept out of source control

## Suggested First 30 Minutes

If you are new to the repo:

1. Read [../README.md](../README.md).
2. Read [START_HERE.md](START_HERE.md).
3. Open the package README that matches your immediate job.
4. Use [../examples/README.md](../examples/README.md) to find the smallest runnable reference.
5. Use [REFERENCE_MAP.md](REFERENCE_MAP.md) only after you know the feature area.

## Maintenance Rule

If a top-level directory becomes important for everyday work, it should either:

- have its own README, or
- be linked from one of the entry docs above

The goal is that a contributor should be able to answer "where do I start?" in under a minute.
