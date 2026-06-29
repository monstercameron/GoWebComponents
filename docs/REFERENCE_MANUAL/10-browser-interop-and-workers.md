# 10 Browser Interop And Workers

Use this chapter when your app needs browser APIs, worker-backed background compute, or browser-local transport between tabs and windows.

It is the right chapter for:

- `interop` wrappers around storage, clipboard, media queries, DOM lookup, resize observation, and custom events
- `interop.ImportModule(...)` for lazy JavaScript helpers that stay outside the normal Go render tree
- `interop.OpenWorker(...)`, `interop.OpenGoWASMWorker(...)`, `ui.UseWorkerTask(...)`, and `interop.OpenWorkerPool(...)`
- cross-tab coordination through `interop.OpenCrossTabChannel(...)`
- popup and opener coordination through `interop.OpenSecondaryWindowChannel(...)` and `interop.OpenWindowOpenerChannel(...)`
- typed peer messaging through the client-message helpers

Use another chapter instead when:

- you need the normal component and hook model first: go to [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
- you need route-owned data loading and cache ownership first: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need route registration and guard semantics first: go to [08 Routing](08-routing.md)
- you need service workers, installability, or offline shell packaging: go to [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## Overview

`interop` is the supported browser boundary for GoWebComponents.

The main rules are:

- prefer `interop` over raw `syscall/js` when the public package already covers the browser capability
- keep ownership local so the feature that opens a handle also closes or cancels it
- convert browser values into Go values early instead of storing long-lived browser objects in shared state
- choose the smallest primitive that matches the problem:
  - one component and one DOM tree: storage, document, media-query, or custom-event helpers
  - one CPU-heavy feature: worker helpers
  - multiple tabs with no direct window handle: cross-tab channel
  - opener and popup with targeted messages: window channel
  - multiple sovereign clients with explicit identity and capability negotiation: client-message helpers

One boundary matters throughout this chapter: browser interop is a runtime bridge, not a hidden distributed system. The framework gives you typed transport and lifecycle helpers. Your app still owns merge rules, authority, and cleanup.

## Stability Note

Most direct browser wrappers are `Stable`:

- `GetLocalStorage`, `GetSessionStorage`, `GetClipboard`, `GetDocument`, `GetDocumentEvents`, `GetWindowLocation`, `GetWindowHistory`, and `GetMediaQuery`
- `SubscribeDecoded(...)` for typed custom events
- `ImportModule(...)` plus `Module.Call(...)`, `Module.CallDefault(...)`, `Module.Value(...)`, and `Module.Dispose()`
- `OpenWorker(...)`, `OpenGoWASMWorker(...)`, `OpenMessageChannel()`, `SubscribeDecodedWorker(...)`, and `RequestWorkerDecoded(...)`
- `OpenCrossTabChannel(...)`, `SubscribeDecodedCrossTab(...)`, `OpenSecondaryWindowChannel(...)`, `OpenWindowOpenerChannel(...)`, and `SubscribeDecodedWindow(...)`
- common popup-signal helpers such as `PublishLogout(...)`, `PublishSessionExpired(...)`, `PublishRouteFocus(...)`, `PublishSelection(...)`, and `PublishIntent(...)`

Advanced or more change-sensitive surfaces:

- `OpenWorkerPool(...)`, `GetSharedMemorySupport()`, and `OpenSharedBuffer(...)` are advanced public worker surfaces and should be justified by measurable workload pressure
- the client-message helpers in `interop` are the current experimental public slice for multi-client coordination across sovereign browser surfaces
- service-worker ownership is intentionally separate under `pwa`, not hidden behind the worker APIs in this chapter

Older docs may still mention `CurrentDocument()`, `DocumentEvents()`, or `WindowOpenerChannel(...)`. The current exported names are `GetDocument()`, `GetDocumentEvents()`, and `OpenWindowOpenerChannel(...)`.

## Internal Interposer Boundary

The framework now also uses an internal interposer layer when first-party plugins such as `devtools` need deep visibility into browser and worker state.

That internal layer exists so the framework can normalize:

- `runtime` versus `runtime2`
- DOM adapter differences
- event, worker, and transport details
- route, fetch, asset, and security observations that originate in browser-owned code

For app authors, the rule is still the same:

- use the public `interop`, `ui`, `router`, `fetch`, and `pwa` APIs directly
- do not depend on `internal/pluginruntime` or any interposer package
- treat worker transport details such as structured-clone, binary, or shared-buffer paths as implementation details unless a public API documents them

This boundary matters most for debugging and devtools:

- devtools can inspect DOM, event, worker, and runtime2 state without freezing those implementation details into public application APIs
- apps still own normal browser interaction code through the supported public packages

## Minimal Example

Start with one component that keeps interop local: load a draft from storage, copy it to the clipboard, and listen for a typed custom event on the document event target.

```go
package main

import (
	"context"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

type workspaceSavedPulse struct {
	Revision int    `json:"revision"`
	Message  string `json:"message"`
}

// renderInteropQuickstart keeps storage, clipboard, and document-event ownership inside one component.
func renderInteropQuickstart() ui.Node {
	getDraft := ui.UseState("Ship the browser boundary cleanly.")
	getStatus := ui.UseState("Waiting for saved pulse.")

	handleUserInput := ui.UseEvent(func(getE ui.Event) {
		getDraft.Set(getE.GetValue())
	})
	handleUserCopy := ui.UseEvent(func() {
		getClipboard, getErr := interop.GetClipboard()
		if getErr != nil {
			getStatus.Set(getErr.Error())
			return
		}
		if getErr = getClipboard.WriteText(context.Background(), getDraft.Get()); getErr != nil {
			getStatus.Set(getErr.Error())
			return
		}
		getStatus.Set("Copied the draft through interop.GetClipboard().")
	})
	handleUserAnnounceSave := ui.UseEvent(func() {
		getEvents, getErr := interop.GetDocumentEvents()
		if getErr != nil {
			getStatus.Set(getErr.Error())
			return
		}
		getErr = getEvents.Dispatch("workspace-saved", workspaceSavedPulse{
			Revision: 1,
			Message:  getDraft.Get(),
		})
		if getErr != nil {
			getStatus.Set(getErr.Error())
		}
	})

	ui.UseEffect(func() func() {
		// Read once on mount, then subscribe to typed document-level events.
		if getStorage, getErr := interop.GetLocalStorage(); getErr == nil {
			if getSaved, getOk, getReadErr := getStorage.GetItem("workspace:draft"); getReadErr == nil && getOk {
				getDraft.Set(getSaved)
			}
		}

		getEvents, getErr := interop.GetDocumentEvents()
		if getErr != nil {
			getStatus.Set(getErr.Error())
			return nil
		}
		getSubscription, getErr := interop.SubscribeDecoded[workspaceSavedPulse](getEvents, "workspace-saved", func(getMessage interop.DecodedCustomEvent[workspaceSavedPulse], getDecodeErr error) {
			if getDecodeErr != nil {
				getStatus.Set(getDecodeErr.Error())
				return
			}
			getStatus.Set(getMessage.Detail.Message)
		})
		if getErr != nil {
			getStatus.Set(getErr.Error())
			return nil
		}
		return getSubscription.Cancel
	}, true)

	return h.Main(
		h.Class("space-y-4 p-6"),
		h.Input(h.Value(getDraft.Get()), h.OnInput(handleUserInput)),
		h.Div(
			h.Class("flex gap-3"),
			h.Button(h.Type("button"), h.OnClick(handleUserCopy), "Copy draft"),
			h.Button(h.Type("button"), h.OnClick(handleUserAnnounceSave), "Dispatch saved pulse"),
		),
		h.P(getStatus.Get()),
	)
}
```

Why this is the right first interop step:

- the browser handles stay inside one owner
- cleanup stays explicit through the returned subscription cancel function
- the payload that crosses the boundary is small and typed

For cross-root or plugin-host communication, use document-level custom events
as the narrow bridge between otherwise independent trees. A plugin host panel,
an exported custom element, and a normal GoWebComponents root should agree on a
small JSON-shaped payload, dispatch it through `GetDocumentEvents().Dispatch`,
and subscribe with `SubscribeDecoded[T]` inside an effect or equivalent owner.
The cleanup function returned by `SubscribeDecoded[T]` must run when that root
or panel unmounts so the browser listener and its underlying `js.Func` are
released before another host instance mounts.

## Production-Shaped Example

When a feature needs measurement, observation, and lazy JavaScript interop, keep those concerns behind one feature-level component instead of scattering browser calls through many handlers.

```go
package charts

import (
	"context"
	"fmt"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderUsageChartHost measures the host, watches it for resize, and lazily hands chart work to a JS module.
func renderUsageChartHost() ui.Node {
	getHostSize := ui.UseState("Measuring host...")
	getModuleStatus := ui.UseState("Chart helper not loaded yet.")
	getReloadCount := ui.UseState(0)

	handleUserLoadChart := ui.UseEvent(func() {
		getReloadCount.Update(func(getPrevious int) int { return getPrevious + 1 })
	})

	ui.UseEffect(func() func() {
		getDocument, getErr := interop.GetDocument()
		if getErr != nil {
			getHostSize.Set(getErr.Error())
			return nil
		}

		getElements, getErr := getDocument.ElementsByID("usage-chart-host")
		if getErr != nil {
			getHostSize.Set(getErr.Error())
			return nil
		}
		getHost, getOk := getElements["usage-chart-host"]
		if !getOk {
			getHostSize.Set("Chart host is missing from the committed DOM.")
			return nil
		}

		// Keep browser callbacks narrow and store only the measured snapshot.
		if getRect, getErr := getHost.BoundingClientRect(); getErr == nil {
			getHostSize.Set(fmt.Sprintf("%.0f x %.0f", getRect.Width, getRect.Height))
		}
		getResizeSub, getErr := getHost.ObserveResize(func(getEntry interop.ResizeEntry) {
			getHostSize.Set(fmt.Sprintf("%.0f x %.0f", getEntry.ContentRect.Width, getEntry.ContentRect.Height))
		})
		if getErr != nil {
			getHostSize.Set(getErr.Error())
			return nil
		}
		return getResizeSub.Cancel
	}, true)

	ui.UseEffect(func() func() {
		if getReloadCount.Get() == 0 {
			return nil
		}

		getModule, getErr := interop.ImportModule(context.Background(), "/static/modules/usage-chart.mjs")
		if getErr != nil {
			getModuleStatus.Set(getErr.Error())
			return nil
		}

		// One imported module handle can serve repeated calls while the feature stays mounted.
		_, getErr = getModule.CallDefault(context.Background(), map[string]any{
			"hostID": "usage-chart-host",
			"series": []int{12, 18, 22, 27},
		})
		if getErr != nil {
			getModuleStatus.Set(getErr.Error())
			_ = getModule.Dispose()
			return nil
		}

		getModuleStatus.Set("Chart helper loaded through interop.ImportModule(...).")
		return func() {
			_ = getModule.Dispose()
		}
	}, getReloadCount.Get())

	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		h.Div(h.ID("usage-chart-host"), h.Class("min-h-48 rounded-2xl border border-dashed border-slate-300")),
		h.P(h.Textf("Host size: %s", getHostSize.Get())),
		h.P(getModuleStatus.Get()),
		h.Button(h.Type("button"), h.OnClick(handleUserLoadChart), "Load chart helper"),
	)
}
```

Why this is the production-shaped baseline:

- DOM lookup happens after mount, not ahead of render
- resize observation replaces polling
- module ownership stays local and explicit through `Dispose()`
- the component stores measurements and results, not raw browser internals

## Scale-Up Example

In a larger app, separate browser-local transport by authority boundary: use a worker for CPU-heavy jobs, a cross-tab channel for peer hints, and a popup channel only for targeted opener or popup workflows.

```go
package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
)

type searchProgress struct {
	Percent int    `json:"percent"`
	Stage   string `json:"stage"`
}

type searchResult struct {
	Count int `json:"count"`
}

// buildWorkspaceRuntime opens the app-owned browser coordinators for one workspace shell.
func buildWorkspaceRuntime(getCtx context.Context, getPeerID string) (func(), error) {
	getSearchPool, getErr := interop.OpenWorkerPool(getCtx, interop.WorkerPoolOptions{
		Size:       2,
		QueueLimit: 4,
		OpenWorker: func(getOpenCtx context.Context) (interop.Worker, error) {
			return interop.OpenWorker(getOpenCtx, interop.WorkerOptions{
				URL:   "/workers/search.mjs",
				Name:  "workspace-search",
				Type:  "module",
				Ready: true,
			})
		},
	})
	if getErr != nil {
		return nil, getErr
	}

	getPeers, getErr := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{
		Name: "workspace:presence",
	})
	if getErr != nil {
		_ = getSearchPool.Close()
		return nil, getErr
	}

	getPeerSub, getErr := interop.SubscribeClientMessages(getPeers, func(getMessage interop.ClientMessage, getDecodeErr error) {
		if getDecodeErr != nil || getMessage.Source.ID == getPeerID {
			return
		}
		if getMessage.Kind == interop.ClientInvalidate && getMessage.Topic == "catalog" {
			fmt.Printf("peer invalidated catalog revision %s\n", getMessage.Revision)
		}
	})
	if getErr != nil {
		_ = getPeers.Close()
		_ = getSearchPool.Close()
		return nil, getErr
	}

	getIdentity := interop.ClientIdentity{
		ID:      getPeerID,
		App:     "atlas",
		Surface: "workspace-tab",
		Role:    "workspace",
		Version: "v1",
	}
	if getErr = interop.PublishClientHello(getPeers, getIdentity); getErr != nil {
		_ = getPeerSub.Cancel()
		_ = getPeers.Close()
		_ = getSearchPool.Close()
		return nil, getErr
	}

	go func() {
		_, _ = getSearchPool.Request(getCtx, "build-index", map[string]any{
			"query": "atlas",
		}, func(getProgress interop.WorkerMessage, getProgressErr error) {
			if getProgressErr == nil && getProgress.Phase == "progress" {
				fmt.Printf("worker progress: %v\n", getProgress.Payload)
			}
		})
	}()

	return func() {
		_ = interop.PublishClientGoodbye(getPeers, getIdentity)
		_ = getPeerSub.Cancel()
		_ = getPeers.Close()

		getDrainCtx, getCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer getCancel()
		_ = getSearchPool.Drain(getDrainCtx)
		_ = getSearchPool.Close()
	}, nil
}
```

How this scales:

- worker pools belong to app-owned services or route shells, not random leaf components
- cross-tab channels should carry small invalidation, presence, or draft hints instead of whole-state replication
- popup channels should remain targeted and authority-aware instead of pretending every browser surface is a peer
- client-message helpers give explicit identity and topic contracts when a plain payload channel is no longer enough

## Browser API Families

| Family | Primary APIs | Stability | Use this when | Do not use this when | Notes |
| --- | --- | --- | --- | --- | --- |
| Storage and clipboard | `GetLocalStorage`, `GetSessionStorage`, `GetClipboard` | `Stable` | you need browser-owned persistence or copy/paste inside the current tab | the server owns the source of truth or the data is sensitive enough that browser persistence is the wrong layer | handle `interop.CodeUnavailable` on non-browser paths |
| DOM lookup and observers | `GetDocument`, `ElementByID`, `ElementsByID`, `BoundingClientRect`, `ObserveResize`, `ObserveIntersection`, `WrapElement` | `Stable` | a mounted feature needs measurement, focus, or host-element attachment | you can express the behavior declaratively without imperative lookup | `WrapElement(jsValue)` bridges a ref node to the Element surface; reacquire handles after remounts |
| Promise bridge | `Value.Await(ctx)`, `Value.AwaitCall(ctx, method, …)` | `Stable` | resolve a JS Promise into Go with context cancellation and managed `js.Func` lifetime | the value is already synchronous | collapses the `then`/`catch`/`Release` idiom; call from a goroutine, not a render |
| Crypto and notifications | `GenerateAESKey`, `Encrypt`, `Decrypt`, `NewEncryptedStore`, `RequestNotificationPermission`, `PostNotification`, `NotificationPermissionState` | `Stable` | you need AES-GCM-at-rest or browser notifications without hand-rolling promise chains | a server owns crypto, or notifications are unwanted | all return `CodeUnavailable` off-browser |
| Event bridge | `GetDocumentEvents`, `GetWindowEvents`, `Dispatch`, `SubscribeDecoded` | `Stable` | the current page needs typed browser-local event fanout | the event is really route data, server data, or global app state | keep event payloads JSON-shaped |
| Media and timers | `GetMediaQuery`, `ScheduleTimeout`, `ScheduleInterval` | `Stable` | the browser owns the signal and you need a supported wrapper | a normal hook or render-driven state change is enough | prefer observers over polling where possible |
| Dynamic modules | `ImportModule`, `Module.Call`, `Module.CallDefault`, `Module.Value`, `Module.Dispose` | `Stable` | a JS helper should load lazily or stay outside the Go bundle | the behavior belongs in normal Go UI code | dispose imported modules when the feature unmounts |
| Direct workers | `OpenWorker`, `OpenGoWASMWorker`, `RequestWorkerDecoded`, `SubscribeDecodedWorker` | `Stable` | a browser-only CPU-heavy job should leave the UI thread | goroutines inside the current wasm runtime are enough | workers are explicit compute helpers, not a hidden scheduler |
| Worker composition | `OpenMessageChannel`, `PostPorts`, `SubscribeDecodedMessagePort`, `OpenWorkerPool` | `Advanced public` | one feature needs richer worker topology or bounded job fanout | one `ui.UseWorkerTask(...)` or one direct worker is enough | reserve pools for measured workload pressure |
| Shared memory | `GetSharedMemorySupport`, `OpenSharedBuffer` | `Advanced public` | cross-origin-isolated worker flows benefit from shared buffers or atomics | normal message-passing already meets the workload needs | advanced path; gate on capability checks |
| Cross-tab | `OpenCrossTabChannel`, `SubscribeDecodedCrossTab`, `PublishClientHello`, `PublishClientInvalidation` | `Stable` | separate tabs need presence, invalidation, or small state hints | you need targeted popup control or whole-state replication | `BroadcastChannel` is preferred; storage fallback exists |
| Multi-window | `OpenSecondaryWindowChannel`, `OpenWindowOpenerChannel`, `SubscribeSurfaceSignals`, `PublishRouteFocus`, `PublishIntent` | `Stable` | an opener and popup need targeted same-origin collaboration | there is no direct opener relationship | keep one surface authoritative |

## Multi-Surface And Multi-Client Rules

When the same product spans tabs, popups, windows, or several sovereign browser clients:

- choose one authoritative writer for each data class
- publish presence, invalidation, intent, or focus signals instead of whole-state replication
- carry explicit peer identity, surface kind, revision, and capability metadata once coordination becomes product-critical
- degrade cleanly when one tab, popup, or client disappears

That keeps cross-surface coordination understandable:

- cross-tab for peer hints without direct handles
- popup or opener channels for targeted same-origin collaboration
- multi-client helpers when several independent browser clients need explicit identity and topic contracts

## Lanes Versus Pools

Prefer direct worker lanes when:

- worker count is fixed for the active mode
- the app can map work deterministically to workers
- one request per lane per batch is the natural shape
- dispatch overhead matters enough to measure

Prefer `OpenWorkerPool(...)` when:

- requests arrive irregularly or burstily
- many callers share one worker fleet
- queue limits, backpressure, or worker replacement belong in one shared service

Use the smallest shape that works. Most features should start with one direct worker or `ui.UseWorkerTask(...)` and earn pools or multi-lane fanout through measurement.

## Design Notes And Boundaries

- `interop` is intentionally the supported browser boundary, not a browser-agnostic abstraction layer. Browser-only paths should stay browser-only.
- Workers in this repo are app-owned background compute. They are not a replacement for route loaders, server functions, or ordinary goroutines.
- Cross-tab transport is a hint and invalidation mechanism, not an automatic replicated-state system. Carry revision metadata when conflict matters.
- Popup coordination is narrower than peer-tab coordination. One surface should own lifecycle, canonical writes, and close or focus control.
- Client-message helpers are the current bridge for sovereign browser clients. They do not promise durable replay, total ordering across peers, or automatic authority resolution.
- Shared memory is optional and capability-gated. If `CanUseSharedMemory` is false, stay on the normal request or message-port path.
- Service workers remain in the `pwa` boundary because installability, caching, and update lifecycle are operational concerns, not just compute-worker concerns.

## Common Failure Modes

- caching raw `Element` handles across remounts and then reading stale DOM state
- leaving resize, event, or media-query subscriptions active after the feature unmounts
- importing one JS module per interaction instead of reusing one handle while the feature stays mounted
- pushing full app objects through cross-tab channels instead of publishing revisioned invalidation or intent messages
- treating popup loss as a fatal runtime error instead of an expected disconnected state
- opening worker pools before there is evidence that one worker or `ui.UseWorkerTask(...)` is insufficient
- assuming shared memory exists without checking `GetSharedMemorySupport()`
- using interop helpers from SSR or native paths without handling `interop.CodeUnavailable`

## Validation

Use the smallest relevant checks for the interop slice you touched:

```powershell
go run ./tools/gwc build -app .\examples\public\browser-interop\main.go -root .\examples\public\browser-interop
go run ./tools/gwc build -app .\examples\public\worker-text-index\main.go -root .\examples\public\worker-text-index
go run ./tools/gwc build -app .\examples\public\cross-tab-sync\main.go -root .\examples\public\cross-tab-sync
go run ./tools/gwc build -app .\examples\public\multi-window-console\main.go -root .\examples\public\multi-window-console
go test ./interop ./ui
```

For shared-memory or channel-transport debugging, confirm the actual resolved runtime behavior in the browser:

- inspect `channel.Transport()` to see whether cross-tab messaging resolved to `broadcast-channel` or `storage-event`
- inspect `GetSharedMemorySupport()` before enabling `OpenSharedBuffer(...)`
- treat popup `Closed()` state and publish failures as ordinary degraded-state signals

## Keeping The Program Alive (`interop.KeepAlive`)

A wasm `main` must not return, or the Go runtime exits and the app's event handlers stop firing.
`interop.KeepAlive()` blocks the calling goroutine forever (`select {}`). `ui.Run(...)` calls it for
you, so most apps never call it directly; reach for it only when `main` mounts manually and must
stay alive after doing its own setup work.

## Topic Pagination
Topic 10 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [09 SSR And Hydration](09-ssr-and-hydration.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
