AGENTS.md

Skills
Skill files live in agents/.
Load a skill before acting on any task that matches its description.

agents/design.skills  frontend UI, landing pages, components, visual design

Naming
Use verbSubject[Object].

Verbs
get set store cache clear render build handle filter format parse apply reset

Subjects
Use the owning domain:
sidebar composer thread model conv toolbar user canvas

Rules
- All new functions and variables start with a verb.
- Subject names the domain, not the data type.
- Add Object only when needed.
- Booleans start with is, has, can, or should.
- Every function needs a GoDoc comment.
- First word of each GoDoc comment must be the function name.
- Complex blocks need intent comments, not mechanics comments.

Examples
renderSidebar
renderSidebarConvRow
renderComposerInputArea
getModelLabel
storeConvList
cacheScrollPosition
handleUserDeleteConv
clearThreadMessages
buildToolbarSelectOption
isStreaming
hasExactCost

GWC
Run from repo root:
go run ./tools/gwc

Main commands
doctor bootstrap start examples dev serve build release test verify files import tailwind bench wasm dashboard seed

Primary docs
docs/REFERENCE_MANUAL/02-gwc-workflows.md
docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md
docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md
docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md
tools/README.md

Help
go run ./tools/gwc -h
go run ./tools/gwc <command> -h
go run ./tools/gwc wasm measure -h
go run ./tools/gwc wasm compare -h
go run ./tools/gwc bench -h

Flags (terse)
doctor: -json -host -port -audit(-policy/-baseline/-suppress/-write-baseline)
test: -lane -app/-main -root -json
build: -app/-main -root -profile -out/-output -json
dev: -app/-main -root -html/-index -wasm/-output -host -port -dry-run -json
serve: -root -host -port -index -wasm-file -wasm-route -fixture-json
release: -app/-main -root -out-dir -compression -post-link-opt -validate-smoke -json
verify: -app/-main -root -skip-tests -audit(-policy/-min-severity/-baseline/-suppress/-write-baseline) -json
examples/start/bootstrap: examples(-host,-port,-export-static-catalog | <server-path> <action> | <action> -path <server-path>; action=start|status|stop|restart) start(-mode,-init-git,-skip-*) bootstrap(-examples,-host,-port)
files/import/tailwind: files(-root,-ext,-exclude-dir,-json) import(-src,-out,-json) tailwind(-root,-input,-output,-manifest,-skip-manifest,-json)
bench/wasm/dashboard/seed: bench(-root,-lane,-bench,-count,-parallel,-out,-reference,-json) wasm(measure|compare|compare-compression|compare-cache|compare-toolchain; use subcommand -h) dashboard(-root,-status-url,-json) seed(-root,-command,-db-path,-json)

Typical usage
go run ./tools/gwc doctor
go run ./tools/gwc test -lane unit -lane wasm -lane hydration -lane browser
go run ./tools/gwc build -app .\examples\public\counter\main.go -root .\examples\public\counter
go run ./tools/gwc verify -app .\examples\public\counter\main.go -root .\examples\public\counter
go run ./tools/gwc examples
go run ./tools/gwc .\examples\server\ai-chat-wizard\cmd\server start -json
go run ./tools/gwc .\examples\server\ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\server\ai-chat-wizard\cmd\server status -json
go run ./tools/gwc serve -root .\examples -port 8090
go run ./tools/gwc release -app .\examples\public\counter\main.go -root .\examples\public\counter

Todo execution
Do one todo at a time.

Per todo
1. Pick one unchecked item.
2. Read only needed files.
3. Make the smallest correct change.
4. Run the minimum validation.
5. Update todo status and notes.
6. Write a checkpoint.
7. Wait 10 seconds if supported, else continue.

Batching
Do not batch unrelated todos.
Combine only if:
- same subsection
- second is required for the first
- diff stays small
- same validation covers both

Loop
do one todo
validate
checkpoint
wait if supported
continue

Checkpoint
- completed todo
- files changed
- validation run
- result
- residual risk
- next suggested todo

Resume
Reopen the todo list, find the next unchecked item, continue from the last checkpoint, do not redo completed work.

Stop
Stop when no unchecked todos remain.

Priority
Controlled sequential progress beats throughput.

Bug fix workflow
1. Reproduce with the smallest reliable command, test, or browser flow.
2. Capture the exact failure.
3. Find root cause before editing.
4. Prefer the smallest root-cause fix.
5. Preserve public behavior unless the bug requires change.
6. Keep useful diagnostics.
7. Validate narrowly first, widen only if needed.

Implementation rules
- Do not guess when the repo can be inspected.
- Do not hide panics or errors just to pass tests.
- Do not remove useful debug detail without a good reason.
- Do not touch unrelated files or formatting.
- Respect nearby user changes.

Validation rules
- Prefer focused package tests, targeted Playwright specs, or the smallest reproducible command.
- For browser issues, capture the real console or page error.
- For wasm or cross-compilation, clear stale env vars first.
- Confirm the bug is fixed and diagnostics are clearer.
- Ad hoc binaries from repo root go under ./bin.

Examples
go test -c -o ./bin/<name>.test <package>
go build -o ./bin/<name>.exe <package>

Final report
- root cause
- what changed
- validation run
- remaining risk or follow-up

================================================================
BUILDING APPS WITH GOWEBCOMPONENTS
================================================================
Everything below is what an agent needs to author working apps. It is the
programming model; the sections above are the workflow and naming rules. Both
apply at once.

Module path
github.com/monstercameron/GoWebComponents
Import public packages by that path, e.g.
  "github.com/monstercameron/GoWebComponents/v5/ui"
  "github.com/monstercameron/GoWebComponents/v5/html"
  . "github.com/monstercameron/GoWebComponents/v5/html/shorthand"   // dot-import sugar

Mental model
React-style components + hooks on a fiber runtime, written in Go, compiled to
js/wasm. A component is a Go function that returns a `ui.Node`. Hooks give it
local state, effects, and async. You build the DOM with typed `html` builders
(or the `html/shorthand` dot-import sugar) and mount with `ui.Render`. Crash
containment is on by default - a panic in one component is isolated, not a
white screen.

A component
- A component is `func(props P) ui.Node` (props may be omitted: `func() ui.Node`).
- Call hooks ONLY at the top level of the component body - never in a
  conditional, loop, goroutine, or nested closure. (`gwc lint` has a
  `gwc-hooks` pass that enforces this; dev/test builds also panic on a
  cross-goroutine hook call.)
- Wrap it with `ui.CreateElement(Component)` or
  `ui.CreateElement(Component, props)` to get a mountable node.
- Return built nodes; do not touch the DOM directly.

Minimal app (entry file MUST be js/wasm - see build constraint)

  //go:build js && wasm

  package main

  import (
    . "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
    "github.com/monstercameron/GoWebComponents/v5/ui"
  )

  // App renders the root component.
  func App() ui.Node {
    parseCount := ui.UseState(0)
    parseInc := ui.UseEvent(func() {
      parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
    })
    return Div(Class("p-4"),
      Div(Class("text-2xl"), Textf("count: %d", parseCount.Get())),
      Button(OnClick(parseInc), Class("mt-2 rounded bg-cyan-500 px-3 py-1"), Text("+")),
    )
  }

  func main() {
    ui.Render(ui.CreateElement(App), "#app")
    select {}   // keep the wasm runtime alive
  }

The canonical, fuller reference is examples/public/counter/main.go (it uses the
example boot harness `exampleboot.RenderExampleRoot` instead of bare
`ui.Render`; for a standalone app use `ui.Render(node, "#app")` plus `select {}`).

Build constraint (non-negotiable)
- The app entry (`package main` with `main()`) and any file using browser APIs
  is `//go:build js && wasm`. Build with GOOS=js GOARCH=wasm (gwc does this).
- gopls hides js/wasm files unless GOOS=js/GOARCH=wasm is set; the committed
  `.vscode/settings.json` does this. Clear stale GOOS/GOARCH before native work.

wasm / native split (how the framework stays buildable on both targets)
- Browser-only code lives in `*_wasm.go` (`//go:build js && wasm`).
- A matching `*_native.go` stub (`//go:build !js || !wasm`) provides the same
  exported signatures and returns an "unavailable" result so the package still
  compiles, vets, and unit-tests on the host. Native tests assert the
  unavailability via `interop.IsCode(err, interop.CodeUnavailable)`.
- When you add a browser capability, add BOTH files. Pure-Go logic (parsing,
  serialization, formatting) goes in a tag-free file so it is unit-testable on
  the host and identical on both targets.

Building the DOM
- Typed builders in `html`: `Div`, `Span`, `Button`, `H1..H6`, `P`, `Ul`, `Li`,
  `Input`, `Form`, `Img`, ... plus `Text(s)`, `Textf(format, ...)`, `Fragment()`.
- Sugar via `. "html/shorthand"` dot-import: `Class(...)`, `ClassNames(...)`
  (compose Tailwind class strings), `When(cond, class)`, `IfElse(cond, a, b)`,
  `OnClick(handler)`, `Props{...}`. The counter example shows the idiom.
- Props are attributes/handlers; children are trailing varargs.
- Styling is Tailwind; `gwc tailwind` builds the shared CSS + class manifest.

Hooks (package ui) - the surface you will use most
State / lifecycle:
  UseState[T](initial) State[T]            // .Get() .Set(v) .Update(fn)
  UseReducer[S,A](reducer, initial)        // Redux-style
  UseRef[T](initial) Ref[T]                // mutable box, no re-render
  UseEffect(func() func(), deps...)        // returns a cleanup func; deps gate it
  UseMemo[T](compute, deps...) T
  UseCallback[T](fn, deps...) T
  UseEvent(fn) Handler                     // stable event handler (use for OnClick)
  UseContext[T](*Context[T]) T
  UseId() string                           // stable unique id (labels/aria)
  UsePrevious[T](v) Previous[T]
Async / data:
  UseTask[T](run func(ctx) (T,error)) Task[T]
  UseChannel[T](<-chan T) Channel[T]
  UseLazyNode(loader, deps...)             // code-split / defer a subtree
  UseWorkerTask[...]                       // offload to a web worker
Perf / scheduling:
  UseTransition(), UseDeferredValue[T], UseDebounced[T], UseThrottled[T]
Accessibility:
  UseAnnouncer(), UseFocusTrap(opts), UseFocusManager(),
  UseCompositeNavigation(items, opts), UseOverlayStack(opts)
Preferences / animation:
  UsePrefersReducedMotion() bool, UsePrefersColorScheme() ColorScheme,
  UseSpring(target, anim.SpringConfig) float64
Persistence:
  UsePersistedState[T](key, initial, area)  // localStorage/sessionStorage write-through + cross-tab

Beyond local state (other packages)
- state  : `UseAtom[T](id, initial) Atom[T]` - app-wide shared reactive state
           (fan-out to every reader). Atoms are app-global by id.
- events : typed in-app pub/sub. `events.UseTopic[T](topic, handler, opts...)`,
           or non-hook `events.Subscribe[T]` / `events.Publish[T]`. Exactly-once
           fan-out, lifecycle-tied unsubscribe.
- router : `router.NewHashRouter()` / `NewHistoryRouter()`,
           `router.RegisterRoute(path, component)`, `router.Navigate(path)`,
           `router.Current()`.
- fetch  : data hooks - `UseQuery`/`UseInfiniteQuery`/`LoadQuery`, cache with
           tags (`InvalidateQueryTag`), `UseWebSocket`/`UseEventSource`,
           `MutationQueue` (offline replay), and resilience (`RetryPolicy`,
           `CircuitBreaker`, `ExecuteWithPolicy[T]`). Cache keys are app-global;
           scope with `fetch.ScopeCacheKey(...)`.
- flags  : feature flags + weighted experiments; `flags.RemoteProvider` for
           remote config + kill switches.
- i18n   : `FormatNumber`/`FormatDate`/`FormatRelativeTime`/`FormatList`, plural
           rules, `T(namespace, key)` messages. Browser ICU bridge via interop.
- interop: browser APIs from Go (cookies, crypto.subtle, Intl, media queries,
           rAF, document events, workers) - always a `_wasm.go` + `_native.go`
           pair returning `CodeUnavailable` on the host.
- pwa    : service worker registration, install observation, offline diagnostics.
- anim   : pure-Go `Spring` (presets GentleSpring/WobblySpring/StiffSpring),
           easings, FLIP. Drive UI with `ui.UseSpring`.
- sanitize: `Sanitize(html, ...Policy)` for untrusted HTML (there is no raw
           innerHTML sink in the render path by design - never reintroduce one).

SSR / hydration (server-side rendering)
- `ui.RenderToString(node) (string, error)` - render a tree to HTML on the host
  (no browser). This is also the headless oracle for tests.
- `ui.RenderToStream` / `RenderToStreamObserved` - streaming SSR; emits a shell
  before async `AsyncBoundary` content, flushes replacements as they resolve.
- Hydration attaches the wasm runtime to server HTML; bootstrap sidecars carry
  versioned payloads. Static islands hydrate selectively.
- Async data: wrap suspending work in `AsyncBoundary` with `ui.SuspendUntil` /
  `ui.Await`; SSR renders the fallback, the client retries on resolve.

Run / build / verify an app (from repo root)
  go run ./tools/gwc dev    -app .\path\to\main.go             # build+serve+livereload
  go run ./tools/gwc build  -app .\path\to\main.go -root .\path
  go run ./tools/gwc verify -app .\path\to\main.go -root .\path  # tests + CI wasm build
  go run ./tools/gwc release -app .\path\to\main.go -root .\path
  go run ./tools/gwc test -lane unit -lane wasm -lane hydration -lane browser
  go run ./tools/gwc lint                                       # golangci-lint + gwc-hooks
Browser tests live under test/playwrightgo/ (build tag `playwrightgo`); run with
  go test -tags playwrightgo ./test/playwrightgo/...

Hard rules (these cause real bugs if broken)
- Hooks at the top level of the component only - never conditional/loop/goroutine/closure.
- All local variables are `parse`-prefixed (see Conventions); exported names are
  ordinary Go; functions are `verbSubject`; GoDoc first word = the symbol name.
- Add the `_native.go` stub whenever you add a `_wasm.go` browser file.
- Do not introduce a raw-HTML sink (innerHTML/outerHTML) on the render path;
  use text/attribute APIs or the `sanitize` package. A guard test enforces this.
- Do not hide panics to pass tests; crash containment surfaces them deliberately.
- Keep handlers stable with `UseEvent`; do not allocate a new closure as a DOM
  handler on every render where it matters.

More docs
- docs/REFERENCE_MANUAL/  (workflows, devtools/testing, assets/PWA, boundaries)
- docs/CONVENTIONS.md     (the naming/`parse`-prefix contract in full)
- docs/capabilities + docs/errorcodes (capability matrix + error codes)
- README.md               (overview + quick start)
- examples/public/        (runnable single-file apps; counter is the canonical one)

================================================================
THE AGENTIC SDLC WITH GWC
================================================================
`gwc` (go run ./tools/gwc <command>) is the toolchain. Treat building an app as
a loop, not a line: orient -> plan -> scaffold -> implement -> build -> verify
-> review -> diagnose -> release -> observe -> back to plan. Below, each phase
maps to the command to reach for. "Today" = available now. "Planned" = speced
in todos.md (Agentic gwc section), NOT yet built - do not assume it exists; use
the Today column until it lands.

Phase        Today (use now)                          Planned (roadmap)
orient       inspect (routes/deps/ownership),         richer semantic ranking
             doctor, model, search, explain,
             README + docs/
plan         inspect (dependency report),             richer multi-symbol impact
             inspect --impact
scaffold     start (TUI), init, import,               starter expansion coverage
             scaffold --no-input --json
implement    edit files directly (Go), mutate,        broader codemod recipes
             click/type/press/hover/scroll (drive
             the running app over CDP)
build        build, wasm, release                     (build --json envelope)
verify       test (-lane unit/race/wasm/              single acceptance gate
             hydration/browser/agent/
             agent-browser/release), verify,
             verify --agent, render, probe, bench,
             lint, screenshot + screenshot-diff
             (visual regression)
review       lint / review, doctor -audit,            richer fixes-as-data
             check --json
diagnose     doctor, dev (livereload + doctor-        deeper hydration/commit
             on-failure), dev --agent, observe        trace coverage
             --agent, sessions/snapshot/query,
             snapshot-diff, test output, browser
             (headed window), screenshot, console,
             network, dom, eval (live CDP proxy)
migrate      migrate (-apply safe rewrites), mutate   arbitrary-safe ops
release      release, deploy, prerender/export        (canary rollout)
observe      observe, observe --agent,                queryable RUM/crash/replay
             logging/diagnostics packages,
             devtools panels

Today's commands (one line each)
  doctor    check toolchains, runtime assets, project signals, optional audit
  inspect   route / dependency / ownership / file-type reports
  model     emit a static component manifest for agent planning
  search    search exported APIs by intent using the static manifest
  explain   resolve diagnostic error codes and framework capabilities
  start     scaffold TUI (presets); init = non-interactive project init
  import    convert a static HTML/JSX file into an inspectable GWC project
  scaffold  generate components, hooks, examples, or starter apps without prompts
  mutate    apply safe AST-backed source mutations with dry-run and JSON diff output
  dev       build app -> serve -> livereload (auto-runs doctor on failure)
  build     build a js/wasm app with an explicit profile
  wasm      wasm build experiments: measure / compare / compare-compression / ...
  test      run lanes: unit, race, wasm, hydration, browser, agent, agent-browser, release
  verify    app-local Go tests + a CI-profile wasm build; --agent emits NDJSON
  render    render a component through the headless SSR oracle
  probe     run a browser-oracle probe for a URL or example target
  check     run agent-shaped diagnostics across tests and source conventions
  bench     discover + run native/wasm benchmarks, compare with benchstat
  lint      golangci-lint + built-in gwc-hooks rules (review is an alias)
  migrate   compatibility findings + safe parser-backed rewrites (-apply)
  prerender static export (route HTML + wasm + manifest); export is an alias
  release   package a js/wasm release with manifest + compressed sidecars
  mcp       serve JSON-capable gwc commands as local stdio MCP tools
  sessions  list live local agent-bridge sessions
  snapshot  read a redacted live agent-bridge runtime snapshot
  query     find live agent-bridge nodes by semantic selector
  set-atom  set a live agent-bridge atom value
  emit      invoke a live agent-bridge node event handler by ref
  publish   publish a live agent-bridge topic event
  navigate  drive live agent-bridge router navigation
  snapshot-diff compare two agent-bridge snapshots by stable ref
  browser   open a headed (visible) Chromium window on a dev URL (-tags playwrightgo)
  screenshot capture a PNG of a running app (full page / selector / -cdp attach)
  screenshot-diff diff two PNGs for visual regression (default build, no browser)
  click/type/press/hover/scroll  real DOM input over CDP (-tags playwrightgo)
  console   capture browser console messages + uncaught JS errors over CDP
  network   capture browser requests/responses/failures over CDP
  dom       read the real rendered DOM (text/HTML/attrs) for a selector over CDP
  eval      evaluate read-oriented JS in the live page over CDP (dual-use)
  deploy    package validated release artifacts through deploy adapters
  tailwind  build shared Tailwind CSS + class manifest
  seed      provision local dev identities + fixture data
  examples  serve the catalog or manage example servers (start/status/stop/restart)
  dashboard monitor livereload clients + AI provider config
  env       print launcher-relevant environment variables
Every command supports `-h`; many support `-json` (see the Flags section above).
Run from repo root: `go run ./tools/gwc <command> -h`.

Recommended loop for building an app (with today's tools)
1. orient   - `inspect` the target area; read the nearest example + docs. Do not
              guess when the repo can be inspected.
2. plan     - scope the change; check the dependency report for blast radius.
3. change   - make the smallest correct edit (one todo at a time; see the
              workflow rules above). Add the `_native.go` stub with any
              `_wasm.go`.
4. build    - `gwc build -app ... -root ...` (clear stale GOOS/GOARCH first).
5. verify   - run the SMALLEST lane that proves it: `gwc verify` for tests+CI
              build, or `gwc test -lane wasm`/`-lane browser` for the relevant
              surface. For browser behavior, `go test -tags playwrightgo
              ./test/playwrightgo/...`. Capture the real console/page error on
              failure - never a paraphrase.
6. review   - `gwc lint` (conventions + gwc-hooks). Fix, re-run.
7. checkpoint - record completed todo, files changed, validation run, result,
              residual risk, next todo. Then continue.
Claim done only with evidence (a passing lane / clean lint / a rendered
oracle), never on assumption. A green `verify` + clean `lint` is the current
definition-of-done; use `verify --agent` when the loop needs NDJSON events and
trace summaries.

MCP
`gwc mcp` exposes JSON-capable commands as local stdio MCP tools so an agent can
call them natively instead of shelling out. It is a local developer surface; a
shipped HTTP+auth MCP service remains future productization work.
