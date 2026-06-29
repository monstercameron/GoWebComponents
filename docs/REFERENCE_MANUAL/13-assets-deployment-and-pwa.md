# 13 Assets Deployment And PWA

Use this chapter when you need a deployable asset graph, browser-compatible wasm delivery, installability, service-worker coordination, or offline-capable release planning.

It is the right chapter for:

- manifest-backed asset URLs and hashed deploy outputs
- `gwc release` artifacts and `wasm-release-manifest.json`
- `pwa.Manifest`, `pwa.MarshalManifestJSON(...)`, and installability wiring
- `pwa.RegisterServiceWorker(...)`, release-scoped asset plans, Cache Storage plans, and diagnostics snapshots
- browser-support boundaries that affect deployment, offline behavior, and installability

Use another chapter instead when:

- you need the launcher commands in depth first: go to [02 GWC Workflows](02-gwc-workflows.md)
- you need request-time rendering and hydration lifecycle first: go to [09 SSR And Hydration](09-ssr-and-hydration.md)
- you need offline mutation ownership and replay semantics first: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need cross-tab channels or worker capability gates first: go to [10 Browser Interop And Workers](10-browser-interop-and-workers.md)

## Overview

Deployment is deliberately split across three ownership layers:

1. application code refers to stable logical assets and decides what the user experience should do offline
2. `gwc release` emits the wasm artifact, compression sidecars, and `wasm-release-manifest.json`
3. the `pwa` package turns that release record into typed manifest, service-worker, cache, and diagnostics helpers without hiding rollout policy inside the rendering runtime

Keep these boundaries clear:

- assets are app-owned deployment inputs, not hidden runtime magic
- the service-worker script is application code even when registration and planning helpers come from `pwa`
- installability is explicit and browser-owned; the framework reports readiness, but it does not force prompts
- offline reads, offline writes, and release invalidation are related but distinct problems

## Deployment Targets

Use one of these deployment shapes on purpose:

- request-time SSR: the server renders HTML, resolves logical assets through one lookup layer, and serves short-lived documents plus immutable hashed assets
- static export: prerender once, copy hashed assets beside exported HTML, and host the output on any static host
- installable shell: ship a manifest, register a service worker explicitly, and keep scope plus update behavior visible in app code
- offline-first app shell: combine service-worker shell caching, reconstructible read caches, and durable mutation replay without confusing any one layer for the source of truth

`examples/server/static-export-site` is the current static-host reference. The `97-pwa-*` examples are the current installability and offline reference slice.

## Stability Note

These are the main `Supported companion` deployment and PWA surfaces:

- `pwa.Manifest`, `Manifest.Validate()`, `MarshalManifestJSON(...)`, and `MarshalManifestJSONIndented(...)`
- `pwa.ObserveInstallability(...)`, `InstallabilityManager.State()`, `Prompt(...)`, and `Subscribe(...)`
- `pwa.RegisterServiceWorker(...)`, `ServiceWorkerRegistration.Snapshot()`, `Update(...)`, `SkipWaiting(...)`, `ReloadOnControllerChange()`, and `RegisterSync(...)`
- `pwa.ParseWasmReleaseManifestJSON(...)`, `WasmReleaseManifest.Revision()`, `BuildServiceWorkerAssetPlan(...)`, and `BuildCacheStoragePlan(...)`
- `pwa.OpenCacheStorageManager()`, `CacheStorageManager.Sync(...)`, `Inspect(...)`, and `InspectDiagnostics(...)`
- `pwa.BuildMutationQueueDiagnosticsSource(...)` when a durable mutation queue should appear in the broader PWA diagnostics view

Important app-owned boundaries:

- the service-worker file itself
- the asset-manifest schema used by SSR, prerender, or static HTML
- cache headers, CDN invalidation, HTTPS, and host-level content types
- offline route-opening policy, purge rules, and user-visible update prompting

## Release Manifest, Cache Plans, And Inspection

The repo now treats the emitted release and cache-planning artifacts as the authoritative handoff between build, offline delivery, and inspection.

That means:

- `wasm-release-manifest.json` is the normalized source for service-worker asset planning, cache-storage planning, and higher-level asset inspection
- `BuildServiceWorkerAssetPlan(...)` and `BuildCacheStoragePlan(...)` are the typed contracts; do not rebuild cache policy from ad hoc DOM or network scraping
- diagnostics and devtools should inspect manifest, registration, and cache-plan state first, then compare that with actual browser cache residency
- if a future asset-management or cache-policy plugin exists, it should consume these typed plans and diagnostics instead of inferring release behavior from file-layout conventions alone

## Minimal Example

Start with the smallest installable baseline: create a valid manifest and register the service worker from explicit client bootstrap code.

```go gwc:build
package deploy

import (
	"context"
	"os"

	"github.com/monstercameron/GoWebComponents/v4/pwa"
)

// buildInstallableManifest returns the app-owned manifest that will be linked from HTML.
func buildInstallableManifest() pwa.Manifest {
	return pwa.Manifest{
		ID:              "/app/",
		Name:            "Atlas Workspace",
		ShortName:       "Atlas",
		Description:     "Installable workspace shell for Atlas.",
		StartURL:        "/app/",
		Scope:           "/app/",
		Display:         pwa.ManifestDisplayStandalone,
		ThemeColor:      "#0f172a",
		BackgroundColor: "#08111d",
		Icons: []pwa.ManifestImage{
			{Src: "/static/icons/app-192.png", Sizes: "192x192", Type: "image/png"},
			{Src: "/static/icons/app-512.png", Sizes: "512x512", Type: "image/png"},
		},
	}
}

// storeInstallableManifest writes a validated manifest JSON file for the deploy output.
func storeInstallableManifest(getPath string) error {
	getManifestJSON, getErr := pwa.MarshalManifestJSONIndented(buildInstallableManifest(), "", "  ")
	if getErr != nil {
		return getErr
	}
	return os.WriteFile(getPath, getManifestJSON, 0o644)
}

// registerClientServiceWorker wires the service worker explicitly during browser bootstrap.
func registerClientServiceWorker(getCtx context.Context) (pwa.ServiceWorkerSubscription, error) {
	getRegistration, getErr := pwa.RegisterServiceWorker(getCtx, pwa.ServiceWorkerOptions{
		URL:   "/app/sw.js",
		Scope: "/app/",
	})
	if getErr != nil {
		return pwa.ServiceWorkerSubscription{}, getErr
	}

	// Keep release rollout visible by reloading only after the new controller actually takes over.
	return getRegistration.ReloadOnControllerChange()
}
```

Why this is the right first deployment shape:

- manifest generation stays typed and validated in Go
- service-worker ownership stays explicit in client bootstrap
- the runtime does not pretend every app is installable or offline-ready by default

## Production-Shaped Example

For a real release, treat `wasm-release-manifest.json` as the shared record between `gwc release`, service-worker precache planning, Cache Storage coordination, and diagnostics.

```go gwc:build
package release

import (
	"context"
	"os"

	"github.com/monstercameron/GoWebComponents/v4/fetch"
	"github.com/monstercameron/GoWebComponents/v4/pwa"
)

// loadReleaseCachePlan converts one emitted release manifest into service-worker and Cache Storage plans.
func loadReleaseCachePlan(getManifestPath string) (pwa.ServiceWorkerAssetPlan, pwa.CacheStoragePlan, error) {
	getManifestJSON, getErr := os.ReadFile(getManifestPath)
	if getErr != nil {
		return pwa.ServiceWorkerAssetPlan{}, pwa.CacheStoragePlan{}, getErr
	}

	getReleaseManifest, getErr := pwa.ParseWasmReleaseManifestJSON(getManifestJSON)
	if getErr != nil {
		return pwa.ServiceWorkerAssetPlan{}, pwa.CacheStoragePlan{}, getErr
	}

	getAssetPlan, getErr := pwa.BuildServiceWorkerAssetPlan(getReleaseManifest, pwa.ServiceWorkerAssetPlanOptions{
		BaseURL:     "/app",
		CachePrefix: "atlas-shell",
		ShellURLs:   []string{"/", "/offline.html"},
		ImmutableURLs: []string{
			"/static/wasm_exec.js",
			"/static/app.css",
		},
	})
	if getErr != nil {
		return pwa.ServiceWorkerAssetPlan{}, pwa.CacheStoragePlan{}, getErr
	}

	getCachePlan, getErr := pwa.BuildCacheStoragePlan(getAssetPlan, pwa.CacheStoragePlanOptions{
		ScriptURLs: []string{"/app/static/wasm_exec.js"},
		StyleURLs:  []string{"/app/static/app.css"},
		MediaURLs:  []string{"/app/static/media/hero-home.svg"},
	})
	if getErr != nil {
		return pwa.ServiceWorkerAssetPlan{}, pwa.CacheStoragePlan{}, getErr
	}

	return getAssetPlan, getCachePlan, nil
}

// syncReleaseCache applies the current release plan to Cache Storage before the user goes offline.
func syncReleaseCache(getCtx context.Context, getPlan pwa.CacheStoragePlan) (pwa.CacheStorageSnapshot, error) {
	getManager, getErr := pwa.OpenCacheStorageManager()
	if getErr != nil {
		return pwa.CacheStorageSnapshot{}, getErr
	}
	return getManager.Sync(getCtx, getPlan)
}

// inspectReleasePWA collects one structured offline and installability snapshot for support workflows.
func inspectReleasePWA(getCtx context.Context, getManifest pwa.Manifest, getRegistration pwa.ServiceWorkerRegistration, getPlan pwa.CacheStoragePlan, getQueue *fetch.MutationQueue) (pwa.DiagnosticsSnapshot, error) {
	getManager, getErr := pwa.OpenCacheStorageManager()
	if getErr != nil {
		return pwa.DiagnosticsSnapshot{}, getErr
	}

	getOptions := pwa.DiagnosticsOptions{
		Manifest:         &getManifest,
		ServiceWorker:    &getRegistration,
		CacheStorage:     &getManager,
		CacheStoragePlan: &getPlan,
	}
	if getQueue != nil {
		// Fold durable write state into the same snapshot without making Cache Storage the source of truth for writes.
		getOptions.OfflineQueue = pwa.BuildMutationQueueDiagnosticsSource(getQueue)
	}

	return pwa.InspectDiagnostics(getCtx, getOptions)
}
```

Why this is the production baseline:

- the release manifest becomes the handoff between build output and offline planning
- the cache namespace is tied to a release revision instead of guessed ad hoc
- diagnostics explain manifest validity, service-worker state, cache residency, queue pressure, and storage pressure together

## Scale-Up Example

For static-hosted or exported docs and marketing sites, keep logical asset names in app code and resolve hashed output URLs through one manifest-backed lookup layer.

```go gwc:build
package export

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/head"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// getExportedAssetManifest returns the logical-to-emitted asset map for the current export.
func getExportedAssetManifest() map[string]string {
	return map[string]string{
		"site-css":   "/static/site-export.4f3a2b1c.css",
		"hero-home":  "/static/media/hero-home.2f71c1a0.svg",
		"panel-home": "/static/media/panel-home.614ca941.svg",
	}
}

// resolveExportedAssetURL resolves a stable asset key to the deploy URL used by exported HTML.
func resolveExportedAssetURL(getKey string) string {
	if getURL, getOK := getExportedAssetManifest()[getKey]; getOK {
		return getURL
	}
	return ""
}

// renderExportedLandingPage emits prerender-friendly HTML that never guesses hashed filenames.
func renderExportedLandingPage() (string, error) {
	getHeadMarkup, getErr := head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:       "Static export home",
			Description: "Manifest-backed asset resolution for a static host.",
		},
		ResourceHints: []head.ResourceHint{
			{Rel: "preload", Href: resolveExportedAssetURL("site-css"), As: "style"},
			{Rel: "preload", Href: resolveExportedAssetURL("hero-home"), As: "image"},
		},
	})
	if getErr != nil {
		return "", getErr
	}

	getBodyMarkup, getErr := ui.RenderToString(
		html.Main(html.Props{},
			html.Section(html.Props{Class: "shell"},
				html.H1(html.Props{}, html.Text("Static export home")),
				html.A(html.Props{Href: "/pricing/"}, html.Text("Pricing")),
				html.Img(html.Props{
					Src: resolveExportedAssetURL("hero-home"),
					Alt: "Static export home hero",
					Raw: map[string]interface{}{
						"loading":  "eager",
						"decoding": "async",
						"width":    "1280",
						"height":   "720",
					},
				}),
			),
		),
	)
	if getErr != nil {
		return "", getErr
	}

	return fmt.Sprintf("<!doctype html><html><head>%s<link rel=\"stylesheet\" href=\"%s\"></head><body>%s</body></html>", getHeadMarkup, resolveExportedAssetURL("site-css"), getBodyMarkup), nil
}
```

How this scales:

- SSR and prerender can share one logical asset lookup model
- route templates never hard-code build output filenames
- static hosts can cache hashed assets aggressively while documents stay short-lived

## Asset And Release Workflow

Use this release flow as the default baseline:

1. build or release the wasm app through `gwc`, not a one-off shell script
2. publish the wasm binary, `wasm_exec.js`, CSS, and other immutable assets together
3. keep HTML documents, manifests, and equivalent entrypoints short-lived
4. let one release manifest or asset lookup layer resolve the final URLs
5. only then derive service-worker and Cache Storage plans from the same release record

The critical rule is consistency: do not let HTML point at one asset graph while the service worker warms another.

## Reproducible Builds

Source-archive provenance (`git archive` + SHA-256 + build-provenance attestation) is already
part of the release flow. The remaining step to a fully verifiable release is **byte-identical
binary reproducibility**: the same source must compile to the same bytes on any machine. The
recipe is exact:

- **`-trimpath`** — strip local filesystem paths from the binary, so it doesn't encode the
  builder's directory layout.
- **`-buildvcs=false`** — keep VCS stamping out of the binary (provenance is recorded
  separately, not embedded), so an otherwise-identical build isn't perturbed by git state.
- **`SOURCE_DATE_EPOCH`** — pin any embedded timestamps to a fixed commit time rather than
  "now".
- **A CI-pinned `toolchain`** — the `go` directive in `go.mod` (and a pinned toolchain line)
  fixes the exact compiler, since output is only reproducible within one toolchain version.

```sh
SOURCE_DATE_EPOCH=$(git log -1 --format=%ct) \
  GOFLAGS='-trimpath -buildvcs=false' \
  GOOS=js GOARCH=wasm go build -o app.wasm ./cmd/app
```

**Verify** in CI by building twice from a clean checkout and byte-comparing (`sha256sum`) the
two outputs; a mismatch fails the release. Reproducibility plus the existing provenance
attestation means a consumer can independently rebuild the exact published artifact and
confirm it matches.

## Browser And Hosting Boundaries

Deployment still has browser and host requirements:

- service-worker registration and installability require HTTPS outside local development
- the emitted wasm binary and `wasm_exec.js` must come from the same Go toolchain
- current evergreen browsers are the intended baseline
- advanced features such as Background Sync, BroadcastChannel, SharedArrayBuffer, and worker shared memory remain capability-gated
- Mobile Safari validation is release-relevant for installability, storage pressure, viewport behavior, and offline flows

Treat these as deployment rules, not optional polish:

- verify the service-worker scope actually matches the routed app base path
- serve wasm with the expected content type and compression behavior for the chosen host
- keep immutable hashed assets on long-lived caching
- keep HTML, manifests, and other mutable entrypoints on short-lived or revalidated caching
- validate that CDN and reverse-proxy behavior do not cache `sw.js` or offline fallback documents incorrectly

## Build Profiles Code Splitting And Experimental Output Modes

Treat build profiles and output modes as explicit release policy:

- development builds optimize for iteration and diagnostics
- debug builds preserve local source paths and record `gcflags` for browser
  crash/DevTools correlation
- CI builds prove the app still compiles and boots under release-like settings
- benchmark builds record the exact flags and compression context used
- release builds optimize for deployable artifact shape and reproducible manifests
- TinyGo builds are explicit constrained-profile experiments for leaf apps that
  can compile under TinyGo's `wasm` target

Keep these related but distinct:

- code splitting is an application-owned loading strategy, not a hidden framework default
- prerender is build-time HTML generation, not request-time SSR and not a synonym for installability
- build experiments should preserve the same release-manifest and asset-lookup contracts before they are trusted in production
- source-debug artifacts should be built with `gwc build -profile debug` or
  `gwc release -profile debug`; do not infer debug flags from an ad hoc
  `go build` command that is not recorded in the release summary
- TinyGo is an opt-in compiler profile, not a silent fallback; use
  `go run ./tools/gwc build -app .\main.go -profile tinygo` only when CI proves
  the app's runtime and package dependencies are TinyGo-compatible

When in doubt, prove the output through `gwc build`, `gwc release`, and the `gwc wasm ...` comparison tools instead of hand-assembling release conclusions from one ad hoc `go build`.

## Runtime Configuration And Browser Support

Startup configuration that crosses into the browser should follow the same rules as bootstrap data:

- only public, non-secret values
- typed ownership in app code
- environment layering and secret resolution on the server
- capability gates for browser-specific features such as Background Sync, BroadcastChannel, shared memory, or install prompts

That keeps deployment predictable across evergreen browsers, constrained devices, and static or SSR hosts.

## API Families

| Family | Primary APIs | Stability | Use this when | Do not use this when | Notes |
| --- | --- | --- | --- | --- | --- |
| Manifest generation | `pwa.Manifest`, `Validate`, `MarshalManifestJSON`, `MarshalManifestJSONIndented` | `Supported companion` | you need typed manifest generation and validation in Go | the app still has no manifest link in HTML or no installability goal | the manifest stays app-owned even though the helper is typed |
| Installability observation | `pwa.ObserveInstallability`, `State`, `Prompt`, `Subscribe` | `Supported companion` | the app should surface readiness, reasons, or an explicit install button | you expect the framework to auto-prompt or guess hidden browser heuristics | browsers decide prompt timing and secure-context rules |
| Service-worker lifecycle | `pwa.RegisterServiceWorker`, `Snapshot`, `Update`, `SkipWaiting`, `ReloadOnControllerChange`, `RegisterSync` | `Supported companion` | the app wants explicit registration and release-aware update handling | you want a built-in generated service worker | registration is first-party, but the worker file is yours |
| Release-manifest planning | `pwa.ParseWasmReleaseManifestJSON`, `Revision`, `BuildServiceWorkerAssetPlan` | `Supported companion` | service-worker and asset invalidation should follow the emitted release record | you are still hand-assembling cache names or URLs per deploy | this is the bridge from `gwc release` into offline planning |
| Cache Storage coordination | `pwa.BuildCacheStoragePlan`, `pwa.OpenCacheStorageManager`, `Sync`, `Inspect` | `Supported companion` | the app needs a typed plan for shell and immutable asset caching | you are trying to use Cache Storage as the source of truth for mutable data | shell HTML and immutable assets belong here; correctness-sensitive API data usually does not |
| Diagnostics | `pwa.InspectDiagnostics`, `pwa.BuildMutationQueueDiagnosticsSource` | `Supported companion` | support or devtools need one structured PWA snapshot | you only need one small point-in-time cache or queue read | combine this with app-owned support flows, not raw console scraping |

## Design Notes And Boundaries

The design intent is integration, not hidden platform ownership.

Important design notes from the source docs:

- GWC does not ship a built-in service-worker runtime
- GWC does not assume every app should be installable or offline-capable
- release invalidation should happen by release revision or hashed asset graph, not by guessing which files changed
- reconstructible read caches and durable write queues should stay separate
- offline route-opening policy belongs to the application because correctness varies by route family

That split is what keeps the PWA helpers useful at scale:

- the package gives you typed manifest, registration, planning, and diagnostics helpers
- the app keeps control over rollout, purge, fallback, and trust-boundary behavior

## Common Failure Modes

- manifest JSON is valid, but the HTML entrypoint never links it, so installability never becomes eligible
- service-worker scope does not match the routed base path, so the worker never controls the intended shell
- HTML is cached immutably while hashed assets are updated, which mixes old documents with a new asset graph
- the service worker warms shell or immutable assets, but the app treats stale API data as equally safe offline
- Background Sync is assumed to exist everywhere instead of falling back to explicit replay
- cache names are hand-written per release instead of derived from `wasm-release-manifest.json`
- Mobile Safari or constrained-device startup is skipped during release validation, so storage or startup regressions escape

## Validation

Use the smallest commands that prove the deployment slice you changed:

- installability and service-worker wiring:
  `go run ./tools/gwc serve -root .\\examples\\public\\pwa-installability -port 8097`
- offline cache, queue, and diagnostics helpers:
  `go run ./tools/gwc serve -root .\\examples\\public\\pwa-offline-cache -port 8098`
- release artifact and manifest emission:
  `go run ./tools/gwc release -app .\\examples\\public\\pwa-offline-cache\\main.go -root .\\examples\\public\\pwa-offline-cache -out-dir .\\bin\\pwa-offline-cache -validate-smoke`
- static export output shape:
  `go run ./examples/server/static-export-site`

When the deployment change is browser-sensitive, add at least one evergreen desktop pass and one Mobile Safari or constrained-device pass before calling the change production-ready.

## Whole-Stack One-Binary Deployment (`wholestack`)

`wholestack` composes a single `http.Handler` that serves the embedded wasm bundle (index +
`.wasm` + `wasm_exec.js`) AND the app's `//gwc:server` functions, with SPA fallback so client-routed
paths deep-link to the shell. One `go build` is the entire app — no Node, no separate static host, no
reverse proxy — and it runs anywhere `net/http` runs, including edge WASI runtimes (the V4 server
packages compile to `GOOS=wasip1`).

```go
//go:embed dist
var assets embed.FS

func main() {
	sub, _ := fs.Sub(assets, "dist")
	log.Fatal(wholestack.ListenAndServe(":8080", wholestack.Options{
		Assets:            sub,
		RegisterServerFns: api.RegisterServerFunctions, // generated by `gwc server gen`
		// IndexFile defaults to index.html; set DisableSPAFallback to opt out of deep-link rewrites
	}))
}
```

`wholestack.Handler(opts)` returns the composed handler if you need to mount it yourself or wrap it
in middleware. Server functions are served under `wholestack.ServerFnPrefix` (`/_gwc/fn/`); every
other non-asset path falls back to the SPA index unless `DisableSPAFallback` is set. See
[07 Data Loading And Mutations](07-data-loading-and-mutations.md) for the `//gwc:server` contract.

## Topic Pagination
Topic 13 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [14 Scaling Large Codebases](14-scaling-large-codebases.md)
