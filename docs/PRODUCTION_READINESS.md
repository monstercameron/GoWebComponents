# Production Readiness Checklist

A gated list of steps to take a GoWebComponents app from "works on my machine"
to "ready to deploy." Each step names the `gwc` command or framework API that
satisfies it, or marks it as a tracked backlog item where automation does not
yet exist.

## Build & artifacts

- [ ] **Release build.** Produce an optimized artifact with the release
  profile: `go run ./tools/gwc release -app ./app/main.go -out-dir ./bin/release`.
  For the smallest leaf-app binaries, the `tinygo` profile is available:
  `gwc build -app ./app/main.go -profile tinygo`.
- [ ] **Compression sidecars.** Emit precompressed assets for your CDN/origin:
  `gwc release ... -compression gzip+brotli`. Confirm the server serves the
  `.gz` / `.br` sidecars with correct `Content-Encoding`.
- [ ] **Release manifest.** The release manifest records per-asset SHA-256.
  Keep it with the deploy so you can verify what shipped.

## Correctness gates

- [ ] **Verify audit.** Run the CI-style gate locally before shipping:
  `gwc verify -app ./app/main.go -root ./app -audit -audit-min-severity error`.
- [ ] **Test lanes.** Run `gwc test -lane unit -lane wasm -lane hydration -lane browser`.
- [ ] **Hydration parity.** If the app uses SSR, confirm the first client
  render matches server markup (no `GWC-HYDRATION-*` diagnostics — see the
  [error-code reference](REFERENCE_MANUAL/error-codes.md)).

## Runtime resilience

- [ ] **Crash containment is on by default.** Verify panics surface as
  structured console diagnostics (not a dead page) in a staging build. Use
  `ui.SafeGo(...)` for app goroutines.
- [ ] **Error transport.** Decide where crash reports go in production. The
  console report is always emitted; wiring an `OnReport` transport to your
  telemetry is app-owned.

## Offline / PWA (if applicable)

- [ ] **Service worker + manifest.** Register via
  `pwa.RegisterServiceWorker(...)` and build a cache plan with
  `pwa.BuildCacheStoragePlan(...)`. Validate the manifest with the `pwa`
  installability helpers.
- [ ] **Offline mutation replay.** If you use `fetch.MutationQueue`, exercise
  the offline→reconnect replay path before relying on it.

## Accessibility & i18n

- [ ] **Reduced-motion / color-scheme.** Respect user preferences via
  `ui.UsePrefersReducedMotion()` / `ui.UsePrefersColorScheme()`.
- [ ] **Focus & announcements.** For overlays and route changes, use the
  `ui` focus-trap, announcer, and composite-navigation primitives.
- [ ] **Locale completeness.** If localized, confirm every `T(...)` key has a
  translation in each shipped locale.

## Security

- [ ] **Untrusted HTML.** Never insert untrusted HTML as markup. Render
  user-supplied markdown through `html.RenderMarkdown` (URL-scheme allowlisted)
  and untrusted HTML through the `sanitize` package.
- [ ] **Dependency scan.** Run `govulncheck ./...` and address findings. See
  [SECURITY.md](../SECURITY.md).
- [ ] **Dev-only surfaces off.** Ensure live-reload and devtools panels are not
  shipped in the production build.

## Performance (partly manual today)

- [ ] **Measure startup & size.** Use `gwc bench` and
  `gwc wasm measure -package ./app` to record wasm size and route-startup
  numbers. Compare against your budget.
- [ ] **Budget gate.** Backlog: an automated perf-budget CI gate that fails on
  regression is not yet wired — track route-startup/size manually for now.

## Backlog (not yet automated — track manually)

These go-live concerns are in the project backlog and do not yet have a `gwc`
command behind them:

- CSP nonce threading and SRI emission on generated shells.
- SBOM emission and root-module gosec/govulncheck in CI.
- Visual-regression and perf-budget CI gates.
- Canary / gradual wasm rollout with instant rollback.

Until these land, satisfy them with your own deployment tooling and CI.
