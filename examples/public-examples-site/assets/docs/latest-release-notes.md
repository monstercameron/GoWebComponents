# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v4.0.0 - 2026-06-28

The current latest changelog section is `v4.0.0 - 2026-06-28` — the V4 release.

**Added (highlights):**

- **Whole-stack, edge-portable deployment** — `wholestack.Handler` composes one
  `http.Handler` serving the embedded wasm bundle *and* the app's `//gwc:server`
  functions (SPA fallback included); the V4 server packages compile to `GOOS=wasip1`
  so the stack runs in edge WASI runtimes. One `go build` is the whole app.
- **`//gwc:server` server functions** — type-safe `func(ctx, Req) (Resp, error)` that
  runs only on the server, with `gwc server gen` generating browser stubs + server
  wiring, fully testable with `httptest`.
- **Local-first sync** — `localfirst` LWW-Register CRDT engine + `PresenceSet`
  awareness; durable offline replay that converges over the real `serverfn` transport.
- **Query/data layer** — the `query` package (keyed cache, dedup, SWR, optimistic
  mutations with rollback) plus `query.MutateAsync` and `ui.UseQuery`/`ui.UseMutation`.
- **Agent-native generative UI** — `agentui` typed renderable schema validated against
  a component allow-list (`Registry.Catalog` introspection for MCP tools), safe by
  construction.
- **Shared validation** — the dependency-free `validate` package (server == client via
  wasm), `ui.Form.ValidateStruct()`, and `router.DecodeQuery`/`EncodeQuery`.
- **Fine-grained reactivity** — `state.Signal`/`NewSignal`/`NewComputed`, two-way
  binding via `html.BindTo`/`BindFunc`, and the one-line `ui.Run` entrypoint.
- **Time travel & devtools** — `timetravel` snapshot engine + in-app dogfooded panels
  (`timetravel/devpanel`, `workbench/gallery`, `query/devtools`, `localfirst/facepile`),
  `ui.UseInspect`, and `ui/erroroverlay.ErrorOverlay`.
- **Tooling** — `gwc add` headless component registry, `gwc i18n gen`/`routes gen`
  typed codegen, `gwc supplychain`, `gwc vuln`, `gwc buildreport`, `gwc llms`, and a
  VS Code extension surfacing `gwc lint` diagnostics. Public-API baseline tests pin the
  V4 packages' exported surface.

It is checked by the blocking `tools/changelogcheck` release gate before a versioned
release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
