# 102 Static Export Site

This example is the current first-party static export and asset-delivery reference.

It prerenders several routes to files with `prerender.Export(...)`:

- `/`
- `/pricing`
- `/docs/getting-started`

Run it from the repo root:

```powershell
go run ./examples/server/static-export-site
```

That writes a `dist/` tree under `examples/server/static-export-site/dist`.

Then serve the generated files from any static host or local file server, for example:

```powershell
python -m http.server 8088 --directory examples/server/static-export-site/dist
```

The key point is that the generated site does not need a custom Go request-time server. The example uses the same `ui.RenderToString(...)` primitives as SSR, but the final deployment shape is static files only.

It now also demonstrates:

- manifest-backed hashed asset URLs copied into `dist/static/`
- route-scoped preload and prefetch hints emitted through `head.Render(...)`
- responsive hero image markup with `srcset` and `sizes`
- lazy-loaded secondary media in the exported route HTML
