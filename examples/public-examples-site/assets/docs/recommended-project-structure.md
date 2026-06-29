# Recommended Project Structure

This is the recommended layout for a production GoWebComponents app.

The goal is to keep framework usage predictable without forcing app code to live in the same repository shape as the framework itself.

## At A Glance

- Keep one obvious application entrypoint and one obvious place for app-owned integration code.
- Import only public packages such as `ui`, `html`, `html/shorthand`, `state`, `fetch`, `router`, `head`, `i18n`, `interop`, `logging`, `pwa`, and `devtools` from application code.
- Treat `internal/` in this repo as framework implementation detail, not as application scaffolding to copy into your own app.
- Use `go run ./tools/gwc dev -app .\path\to\main.go` and `go run ./tools/gwc build -app .\path\to\main.go` as the standard standalone wasm inner loop.
- Split browser entrypoints, server entrypoints, generated assets, and domain code early so SSR, hydration, and deployment concerns do not bleed across the whole app.

## Quick Structure Chooser

Use this layout when the app is:

- standalone wasm only: keep one client entrypoint under `cmd/web` or `app/`, plus feature folders and a generated asset output directory
- request-time SSR or hybrid: add a server entrypoint, keep browser bootstrap separate, and give route, loader, and HTML shell code a stable home under the app integration layer
- domain-heavy: keep business logic under `internal/` or another app-private package tree that does not depend on browser-only code
- growing beyond a few screens: prefer feature-local folders over one large `components/` or `pages/` dumping ground

## Canonical Layout

```text
your-app/
  cmd/
    web/
      main.go
  internal/
    app/
      app.go
      server.go
      routes.go
      middleware.go
      bootstrap.go
      pages/
      components/
      services/
      domain/
      data/
  web/
    static/
      css/
      images/
      wasm/
    templates/
  scripts/
  migrations/
  test/
```

## Current Recommended Shape

That layout is still valid, but the current repo direction suggests a slightly more explicit split for apps that expect SSR, hydration, or multiple deployment modes:

```text
your-app/
  cmd/
    web/
      main.go
    server/
      main.go
  internal/
    app/
      routes/
      pages/
      features/
      loaders/
      forms/
      shell/
      bootstrap/
    domain/
    services/
    data/
    config/
  web/
    static/
      css/
      images/
      icons/
      wasm/
    templates/
    manifest/
  test/
  scripts/
```

Use the extra split when:

- one entrypoint owns browser-only bootstrap and another owns native server startup
- the app emits hashed wasm or manifest-backed assets
- route loaders, forms, or auth rules are large enough to deserve their own home
- teams need a stable place for app shell, head metadata, and bootstrap payload wiring

## Responsibility Split

- `cmd/web/main.go` owns process startup, environment wiring, and HTTP server boot.
- `internal/app/` owns the app-specific route tree, SSR handlers, loaders, mutations, and integration glue.
- `internal/app/pages/` or `internal/app/components/` holds reusable UI composition for the app.
- `internal/app/services/` and `internal/app/domain/` hold business logic that should not depend on the browser runtime.
- `web/static/` holds generated or served assets, including the wasm bundle, CSS, and images.
- `web/templates/` holds the shell HTML or SSR template fragments if the app keeps them on disk.
- `migrations/` holds database schema changes when the app has persistence.
- `test/` or a sibling test directory holds browser, integration, and regression tests.

## Public API Boundary

Application code should treat this repo's public packages as the supported boundary.

Good imports for app code:

- `ui`, `html`, `html/shorthand`
- `state`, `fetch`, `router`, `head`
- `i18n`, `interop`, `logging`, `pwa`, `devtools`

Avoid depending on:

- `internal/*`
- example app source as if it were a stable framework package
- repo-local test helpers, benchmarks, or experimental scaffolding as app architecture guidance

## Why This Layout

- It keeps the Go entrypoint obvious.
- It separates framework integration from domain logic.
- It makes SSR, hydration, forms, and asset serving easier to reason about in larger apps.
- It gives teams a stable place to add route modules and page components without inventing a new folder convention for every feature.

## Workflow Notes

Recommended inner loop:

- standalone wasm app: `go run ./tools/gwc dev -app .\cmd\web\main.go`
- standalone wasm production-oriented build: `go run ./tools/gwc build -app .\cmd\web\main.go -profile development`
- native server entrypoint: ordinary `go run ./cmd/server`

Recommended output separation:

- generated wasm and hashed artifacts under a served static directory
- short-lived HTML or manifest entrypoints separate from immutable assets
- test flows under `test/` or example-local test folders rather than mixed into app runtime packages

## Conventions

- Keep framework code imported from public packages such as `ui`, `html`, `state`, `fetch`, `router`, and `devtools`.
- Avoid putting app-specific logic into the framework repository root.
- Keep browser-only behavior behind browser entrypoints or wasm build tags.
- Prefer feature-local folders inside `internal/app/` once the app grows beyond a few files.

Additional current conventions:

- keep bootstrap payload generation and route shell composition near the server or shell layer, not scattered across feature code
- keep auth, persistence, and external service adapters outside reusable UI packages
- keep manifest, service-worker, and offline support files near deployment assets rather than inside view-component folders

## Notes

- This layout is a recommendation, not a compiler-enforced rule.
- Starter apps should use the same high-level separation so upgrade guidance stays predictable.

## Review Checklist

- is there one obvious app entrypoint for wasm and, when needed, one obvious server entrypoint for SSR or native serving
- are public framework imports separated cleanly from domain, service, and storage code
- are generated assets, HTML shells, and deployment manifests kept outside feature-component folders
- does the structure scale from a small standalone app to a routed SSR or hybrid app without reorganizing the whole repository
