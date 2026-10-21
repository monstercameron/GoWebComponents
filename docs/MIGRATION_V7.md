# Migrating an application from GoWebComponents v6 to v7

This guide covers changes that require action when upgrading a v6 application to
the v7 release candidate. The v7 module path is
`github.com/monstercameron/GoWebComponents/v7`; use an explicit RC requirement
until the stable version is published.

## Module, toolchain, and imports

Change the module requirement and every Go import from `/v6` to `/v7`:

```diff
-require github.com/monstercameron/GoWebComponents/v6 v6.0.0
+require github.com/monstercameron/GoWebComponents/v7 v7.0.0-rc.1
```

After the stable tag is published, replace `v7.0.0-rc.1` with `v7.0.0`.

Set `go 1.27.0` (and use Go 1.27.3 or newer). Run the matching `gofmt`; an older
`gofmt` can reject generic methods used by v7.

The root library module no longer carries the CLI, Playwright, OpenTelemetry SDK
test, or dependency-heavy example graph. Those areas are nested modules:

| Area | Module directory |
| --- | --- |
| `gwc` CLI and tool tests | `tools/` |
| Playwright browser suite | `test/playwrightgo/` |
| OTel SDK integration tests | `telemetry/otel/` |
| Example applications with heavy dependencies | each example's directory |

From the repository root, run the CLI through its module. Arguments are resolved
relative to `tools/`, so an application in the repository examples directory is
written with `../examples` paths:

```powershell
go -C tools run ./gwc verify -app ../examples/public/counter/main.go -root ../examples/public/counter
go -C tools run ./gwc build -app ../examples/public/counter/main.go -root ../examples/public/counter
```

For a consumer project, install or invoke the v7 CLI according to the project's
own tool setup; do not assume that root-module `go run ./tools/gwc` resolves the
same way after the split.

## Required source changes

### Typed input and change handlers

`html.OnInput` and `html.OnChange` now accept only the supported callback shapes.
An untyped value, `nil`, or a function with another signature no longer compiles.
Existing `ui.Handler` values use the explicit adapter:

```diff
-html.OnInput(anyValue)
+html.OnInput(func() { submit() })
```

```go
handler := ui.UseEvent(func(parseEvent ui.Event) { submit() })
option := html.OnInputHandler(handler)
```

The same forms apply to `OnChange` and `OnChangeHandler`. `OnClick`, `OnSubmit`,
and `OnKeyDown` retain their `any` callback form.

### Markdown engine and options

Markdown rendering is now opt-in. Register the bundled Goldmark engine once:

```go
import _ "github.com/monstercameron/GoWebComponents/v7/html/markdown"
```

Alternatively, call `markdown.Enable()` during startup. Without registration,
`html.Markdown` and `html.RenderMarkdown` render escaped source text in a marker
div and report `GWC-UI-MARKDOWN-ENGINE-MISSING` in development.

`html.Markdown` now accepts `...html.MarkdownOption`, so migrate option slices:

```diff
-opts := []html.MarkdownRenderOptions{{LinkRel: "nofollow"}}
-html.Markdown(source, opts...)
+opts := []html.MarkdownOption{html.MarkdownRenderOptions{LinkRel: "nofollow"}}
+html.Markdown(source, opts...)
```

For URL policy, use `html.WithURLSchemes`. An explicit empty call denies every
schemed URL; omitting it keeps the default `http`, `https`, and `mailto` list.
`RenderMarkdown` still accepts a single `MarkdownRenderOptions` value; pass that
value first when refining the configuration.

### Dot imports and the Slot attribute

New generated shorthand names can collide with declarations in code that
dot-imports `html/shorthand`. Rename the local declaration or switch that import
to a named import. The attribute builder is `html.SlotAttr`; `shorthand.Slot`
remains the named-slot node helper.

### Router options and deep links

Router constructors now prefer functional `RouterOption` and `RouteOption` values:

```go
router.NewHistoryRouter(
    router.WithBasePath("/app/"),
    router.WithDefaultRoute("/home"),
)
```

The existing `RouterOptions` and route option structs still satisfy those
interfaces. For a history router, serve deep-link requests through:

```go
http.Handle("/app/", router.SPAFallbackHandler(indexHTML, staticDir, "/app/"))
```

The new `UseParams`, `UseQuery`, `UseSearchParams`, and `UseRouteData` hooks
subscribe to route changes. `Params.As[T]` and `Query.Decode[T]` provide typed
decoding with an error instead of the older silent `Bool`/`Int` result.

## Behavior and data migrations

- **Retry and breaker defaults:** zero `RetryPolicy.MaxDelay` means 5 seconds,
  zero `Multiplier` means 2.0, zero `BreakerConfig.FailureThreshold` means 5,
  and zero `HalfOpenMaxCalls` means one probe. Set explicit values if the v6
  zero-value behavior was intentional (`Multiplier: 1` gives constant delay).
- **Tracing:** wasm fetch/XHR requests to the page origin may include a generated
  `traceparent`. Add cross-origin destinations to `Options.TracePropagationOrigins`
  and allow that header in CORS. Header-exact tests must account for it.
- **Realtime reconnects:** `MinStableDuration` defaults to one second. A negative
  value restores the v6 reset-on-open behavior.
- **`kvstate` tables:** identifiers containing replaced characters are now
  hash-suffixed (FNV-1a, eight hex digits). Rename or copy existing rows once;
  v7 does not perform this migration automatically.
- **PWA canaries:** the first deployment after the bucketing fix can move clients
  between cohorts once.
- **Projection:** a failed `Post` is now a transport failure and rolls back
  optimistic state.
- **CSS and SSR:** the first `css.Inject` body for an ID wins. SSR style
  declarations that could escape the attribute are dropped after key kebab-casing.

No deprecated stable symbol is removed by the documented v7 changes. The v7
major bump is still required because the module path, Go floor, dependency graph,
and the source-compatible-but-compile-breaking APIs above require consumer edits.

## Validation checklist

Before upgrading a consumer or opening the v7 release change:

1. Run `go mod tidy` with Go 1.27.3 and confirm imports resolve to `/v7`.
2. Build and test the root library, then test each nested module used by the app.
3. Run `go -C tools run ./gwc verify` (and use `../`-relative app paths).
4. If the app uses Markdown, execute a browser or rendered-output test with the
   engine registration present.
5. Exercise history-router deep links, typed form handlers, and any wasm fetch
   header assertions.
6. Check persisted `kvstate` tables and perform the one-time rename/copy before
   deploying a build that expects the new identifiers.
7. Run the app's native, wasm, and browser lanes; keep generated binaries and
   nested-module build output out of the root module's tracked files.

The release gate also includes the API-baseline checks and the nested-module
checks described in `VERSIONING.md`. This guide intentionally does not claim
that those release checks have passed until the v7 release run records them.
