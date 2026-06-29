# Production Readiness Checklist

A gated list of steps to take a GoWebComponents app from "works on my machine"
to "ready to deploy." This document is guidance for application teams, not the
repository backlog.

## Build & Artifacts

- **Release build.** Produce an optimized artifact with the release profile:
  `go run ./tools/gwc release -app ./app/main.go -out-dir ./bin/release`. For
  the smallest leaf-app binaries, the `tinygo` profile is available:
  `gwc build -app ./app/main.go -profile tinygo`.
- **Compression sidecars.** Emit precompressed assets for your CDN/origin:
  `gwc release ... -compression gzip+brotli`. Confirm the server serves the
  `.gz` and `.br` sidecars with correct `Content-Encoding`.
- **Release manifest.** Keep the release manifest with the deploy so you can
  verify the per-asset SHA-256 records for what shipped.

## Correctness Gates

- **Verify audit.** Run the CI-style gate locally before shipping:
  `gwc verify -app ./app/main.go -root ./app -audit -audit-min-severity error`.
- **Test lanes.** Run
  `gwc test -lane unit -lane wasm -lane hydration -lane browser`.
- **Hydration parity.** If the app uses SSR, confirm the first client render
  matches server markup without `GWC-HYDRATION-*` diagnostics. See
  [the error-code reference](REFERENCE_MANUAL/error-codes.md).

## Runtime Resilience

- **Crash containment is on by default.** Verify panics surface as structured
  console diagnostics in a staging build instead of leaving a dead page. Use
  `ui.SafeGo(...)` for app goroutines.
- **Error transport.** Decide where crash reports go in production. The console
  report is always emitted; wiring an `OnReport` transport to your telemetry is
  app-owned.

## Offline / PWA

Use this section only when the app ships offline or installability features.

- **Service worker and manifest.** Register via
  `pwa.RegisterServiceWorker(...)` and build a cache plan with
  `pwa.BuildCacheStoragePlan(...)`. Validate the manifest with the PWA
  installability helpers.
- **Offline mutation replay.** If you use `fetch.MutationQueue`, exercise the
  offline-to-reconnect replay path before relying on it.

## Accessibility & I18n

- **Reduced-motion and color-scheme.** Respect user preferences via
  `ui.UsePrefersReducedMotion()` and `ui.UsePrefersColorScheme()`.
- **Focus and announcements.** For overlays and route changes, use the `ui`
  focus-trap, announcer, and composite-navigation primitives.
- **Locale completeness.** If localized, confirm every `T(...)` key has a
  translation in each shipped locale.

## Security

- **Untrusted HTML.** Never insert untrusted HTML as markup. Render
  user-supplied markdown through `html.RenderMarkdown` and untrusted HTML
  through the `sanitize` package.
- **Dependency scan.** Run `govulncheck ./...` and address findings. See
  [SECURITY.md](../SECURITY.md).
- **Dev-only surfaces off.** Ensure live reload, the agent bridge, and devtools
  panels are not shipped in the production build.
- **Release security artifacts.** The repository backlog has completed CSP
  nonce threading, SRI emission, SBOM generation, and root-module security scan
  work; verify your deployment path preserves those generated artifacts.

## Performance

- **Measure startup and size.** Use `gwc bench` and
  `gwc wasm measure -package ./app` to record wasm size and route-startup
  numbers. Compare them against your budget.
- **Budget gate.** The repository backlog has completed the perf-budget gate
  work. Make sure your app or CI adopts the generated budget policy instead of
  treating the numbers as advisory only.
- **Visual regression.** The repository backlog has completed the visual
  regression lane. Add app-specific route screenshots or assertions before
  relying on it for release signoff.

## Rollout

- **Canary and rollback.** The repository backlog has completed the canary /
  gradual wasm rollout work. Confirm your deployment adapter can serve the
  selected release and roll back quickly if startup, hydration, or runtime
  diagnostics regress.
- **Operational record.** Preserve the release manifest, SBOM, benchmark output,
  audit output, and any screenshot or browser-lane artifacts with the release
  candidate so production findings can be traced back to the shipped build.
