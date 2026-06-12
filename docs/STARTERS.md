# Starter Templates

GoWebComponents ships starter templates through `gwc start`. They are
scaffolded apps, not framework rules: generate one, keep what helps, and edit
the result like normal application code.

Run the interactive generator from the repo root:

```powershell
go run ./tools/gwc start
```

For an existing project that only needs launcher metadata, `gwc init -preset
<key>` records the same preset family in `gwc-start.json`.

## Gallery

| Preset key | Starter | Use it when | Included capability set |
| --- | --- | --- | --- |
| `minimal-client` | Minimal Client App | you need the smallest browser-mounted wasm app | `ui`, `html`, dev profile, browser mount |
| `routed-spa` | Routed SPA | you already know the app has multiple screens | `router`, browser tests, dev profile |
| `ssr-app` | SSR App | request-time HTML and hydration matter from day one | `router`, SSR, hydration, release profile |
| `reference-app` | Reference App | you want a broad baseline that demonstrates the main public packages | routing, forms, fetch, state, browser tests |
| `dashboard-app` | Operations Dashboard | internal tools, admin consoles, and reporting surfaces | routing, fetch, state, forms, browser tests, release profile |
| `marketing-site` | Marketing Site | public landing pages, campaigns, and SEO-sensitive launches | routing, SSR, hydration, browser tests, release profile |
| `content-blog` | Content Blog | docs, changelogs, editorial content, or blog-style sites | routing, SSR, hydration, fetch placeholders, release profile |
| `authed-app-shell` | Authed App Shell | SaaS-style shells with sign-in state, guarded screens, and settings flows | routing, forms, fetch, state, browser tests, release profile |

## Generated Contract

Every starter writes:

- `main.go` with a non-empty mounted tree and feature-specific placeholders
- `index.html` with the wasm boot path
- `gwc-start.json` with launcher metadata
- `FEATURE_MATRIX.md` with the selected capability set
- `starter_test.go` with scaffold integrity checks
- `.github/workflows/ci.yml` with `go test ./...` and a wasm build

The repository also has a dedicated starter-templates CI lane that runs the
same scaffold path for every default preset, then executes generated tests and
`gwc build` against the current checkout.

## Selection Rule

Pick the starter whose first day looks most like your app. Do not choose the
largest starter for optional future needs. It is cheaper to add `router`,
`fetch`, `state`, or SSR later than to delete architecture that did not earn
its place.
