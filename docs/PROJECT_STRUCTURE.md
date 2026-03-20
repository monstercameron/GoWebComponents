# Recommended Project Structure

This is the recommended layout for a production GoWebComponents app.

The goal is to keep framework usage predictable without forcing app code to live in the same repository shape as the framework itself.

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

## Responsibility Split

- `cmd/web/main.go` owns process startup, environment wiring, and HTTP server boot.
- `internal/app/` owns the app-specific route tree, SSR handlers, loaders, mutations, and integration glue.
- `internal/app/pages/` or `internal/app/components/` holds reusable UI composition for the app.
- `internal/app/services/` and `internal/app/domain/` hold business logic that should not depend on the browser runtime.
- `web/static/` holds generated or served assets, including the wasm bundle, CSS, and images.
- `web/templates/` holds the shell HTML or SSR template fragments if the app keeps them on disk.
- `migrations/` holds database schema changes when the app has persistence.
- `test/` or a sibling test directory holds browser, integration, and regression tests.

## Why This Layout

- It keeps the Go entrypoint obvious.
- It separates framework integration from domain logic.
- It makes SSR, hydration, forms, and asset serving easier to reason about in larger apps.
- It gives teams a stable place to add route modules and page components without inventing a new folder convention for every feature.

## Conventions

- Keep framework code imported from public packages such as `ui`, `html`, `state`, `fetch`, `router`, and `devtools`.
- Avoid putting app-specific logic into the framework repository root.
- Keep browser-only behavior behind browser entrypoints or wasm build tags.
- Prefer feature-local folders inside `internal/app/` once the app grows beyond a few files.

## Notes

- This layout is a recommendation, not a compiler-enforced rule.
- Starter apps should use the same high-level separation so upgrade guidance stays predictable.
