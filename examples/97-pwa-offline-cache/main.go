//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/pwa"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func offlineCachePlan() pwa.CacheStoragePlan {
	plan, _ := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{
		CacheName:        "pwa-offline-cache-demo-v1",
		ManifestRevision: "demo-v1",
		WasmURL:          "/static/bin/pwa-offline-cache.wasm",
		ShellURLs:        []string{"/97-pwa-offline-cache/offline-cache.html", "/97-pwa-offline-cache/offline.html"},
		ImmutableURLs: []string{
			"/static/css/tailwind.css",
			"/static/css/example-shell.css",
			"/static/script/wasm_exec.js",
		},
	}, pwa.CacheStoragePlanOptions{CachePrefix: "pwa-offline-cache-demo-"})
	return plan
}

func offlineSummary(snapshot pwa.DiagnosticsSnapshot) string {
	return fmt.Sprintf("cache entries=%d | queued=%d | retrying=%d | dead=%d | storage=%s %.0f%% used", snapshot.CacheStorage.EntryCount, snapshot.OfflineQueue.QueuedEntries, snapshot.OfflineQueue.RetryingEntries, snapshot.OfflineQueue.DeadEntries, snapshot.Storage.Pressure, snapshot.Storage.UsageRatio*100)
}

func describeOfflineError(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

func offlineDiagnosticsExample() ui.Node {
	plan := offlineCachePlan()
	cacheManagerRef := ui.UseRef[*pwa.CacheStorageManager](nil)
	cacheManagerStartedRef := ui.UseRef(false)
	queueRef := ui.UseRef[*fetch.MutationQueue](nil)
	queueStartedRef := ui.UseRef(false)
	registrationRef := ui.UseRef[*pwa.ServiceWorkerRegistration](nil)
	serviceWorkerStartedRef := ui.UseRef(false)
	cacheStatus := ui.UseState("Offline cache is idle.")
	queueStatus := ui.UseState("No queued writes yet.")
	replayStatus := ui.UseState("No replay attempted yet.")
	backgroundSyncStatus := ui.UseState("Background replay scheduling idle.")
	conflictStatus := ui.UseState("No conflict resolution attempted yet.")
	serviceWorkerStatus := ui.UseState("Service worker registration pending.")
	diagnosticsPreview := ui.UseState("Click Inspect diagnostics to capture a structured PWA snapshot.")
	diagnosticsSummary := ui.UseState("No snapshot captured yet.")
	ensureServiceWorkerRegistration := func() (*pwa.ServiceWorkerRegistration, error) {
		if registration := registrationRef.Get(); registration != nil {
			return registration, nil
		}
		registration, err := pwa.RegisterServiceWorker(context.Background(), pwa.ServiceWorkerOptions{
			URL:   "/97-pwa-offline-cache/sw.js",
			Scope: "/97-pwa-offline-cache/",
		})
		if err != nil {
			return nil, err
		}
		registrationRef.Set(&registration)
		capabilities := registration.BackgroundSyncCapabilities()
		if capabilities.OneShot {
			serviceWorkerStatus.Set("Service worker registered for offline shell fallback. Background Sync is available.")
		} else {
			serviceWorkerStatus.Set("Service worker registered for offline shell fallback. Background Sync is unavailable, so manual replay remains the fallback.")
		}
		return &registration, nil
	}

	ui.UseEffect(func() func() {
		if cacheManagerRef.Get() == nil && !cacheManagerStartedRef.Get() {
			cacheManagerStartedRef.Set(true)
			go func() {
				manager, err := pwa.OpenCacheStorageManager()
				if err != nil {
					cacheStatus.Set(describeOfflineError("Cache Storage manager unavailable", err))
					return
				}
				cacheManagerRef.Set(&manager)
				cacheStatus.Set("Cache Storage manager ready.")
			}()
		}
		if queueRef.Get() == nil && !queueStartedRef.Get() {
			queueStartedRef.Set(true)
			go func() {
				queue, err := fetch.OpenMutationQueue(fetch.MutationQueueOptions{DeleteOnCorruption: false})
				if err != nil {
					queueStatus.Set(describeOfflineError("Offline mutation queue unavailable", err))
					return
				}
				queueRef.Set(&queue)
				queueStatus.Set("Offline mutation queue ready.")
			}()
		}
		return nil
	}, "offline-cache-initializers")

	ui.UseEffect(func() func() {
		if registrationRef.Get() == nil && !serviceWorkerStartedRef.Get() {
			serviceWorkerStartedRef.Set(true)
			serviceWorkerStatus.Set("Registering service worker...")
			go func() {
				var err error
				for attempt := 0; attempt < 2; attempt++ {
					_, err = ensureServiceWorkerRegistration()
					if err == nil {
						break
					}
					if attempt == 0 {
						serviceWorkerStatus.Set("Retrying service worker registration...")
						time.Sleep(250 * time.Millisecond)
					}
				}
				if err != nil {
					serviceWorkerStatus.Set(describeOfflineError("Service worker registration failed", err))
					return
				}
			}()
		}
		return nil
	}, "offline-cache-service-worker")

	warmOfflineCache := ui.UseEvent(func() {
		manager := cacheManagerRef.Get()
		if manager == nil {
			cacheStatus.Set("Cache Storage manager is not ready yet.")
			return
		}
		cacheStatus.Set("Warming offline cache...")
		go func() {
			snapshot, err := manager.Sync(context.Background(), plan)
			if err != nil {
				cacheStatus.Set(describeOfflineError("Cache warmup failed", err))
				return
			}
			cacheStatus.Set(fmt.Sprintf("Cached %d release entries into %s.", snapshot.EntryCount, snapshot.CacheName))
		}()
	})
	queueOfflineWrite := ui.UseEvent(func() {
		queue := queueRef.Get()
		if queue == nil {
			queueStatus.Set("Offline queue is not ready yet.")
			return
		}
		queueStatus.Set("Queueing offline write...")
		go func() {
			entry, err := queue.Enqueue(fetch.MutationDraft{URL: "/api/offline-demo", Method: "POST", Kind: "demo.sync", Metadata: map[string]string{"source": "offline-cache-example"}})
			if err != nil {
				queueStatus.Set(describeOfflineError("Queue write failed", err))
				return
			}
			queueStatus.Set(fmt.Sprintf("Queued offline write %s with state=%s.", entry.ID, entry.State))
		}()
	})
	queueConflictingWrite := ui.UseEvent(func() {
		queue := queueRef.Get()
		if queue == nil {
			conflictStatus.Set("Offline queue is not ready yet.")
			return
		}
		conflictStatus.Set("Queueing a conflict demo write...")
		go func() {
			entry, err := queue.Enqueue(fetch.MutationDraft{
				URL:      "/api/offline-demo",
				Method:   "POST",
				Kind:     "demo.conflict",
				DedupKey: "conflict:offline-cache-example",
				Metadata: map[string]string{"source": "offline-cache-example", "revision": "local-1"},
			})
			if err != nil {
				conflictStatus.Set(describeOfflineError("Queue conflict write failed", err))
				return
			}
			conflictStatus.Set(fmt.Sprintf("Queued conflict demo write %s at revision %s.", entry.ID, entry.Metadata["revision"]))
		}()
	})
	replayQueuedWrites := ui.UseEvent(func() {
		queue := queueRef.Get()
		if queue == nil {
			replayStatus.Set("Offline queue is not ready yet.")
			return
		}
		replayStatus.Set("Replaying queued writes...")
		go func() {
			report, err := queue.Replay(context.Background(), func(ctx context.Context, mutation fetch.QueuedMutation) error {
				_ = ctx
				_ = mutation
				return nil
			})
			if err != nil {
				replayStatus.Set(describeOfflineError("Queued write replay failed", err))
				return
			}
			replayStatus.Set(fmt.Sprintf("Replay succeeded=%d retried=%d dead=%d remaining=%d.", report.Succeeded, report.Retried, report.DeadLetters, report.Remaining))
		}()
	})
	scheduleBackgroundReplay := ui.UseEvent(func() {
		backgroundSyncStatus.Set("Scheduling background replay...")
		go func() {
			registration, err := ensureServiceWorkerRegistration()
			if err != nil {
				backgroundSyncStatus.Set("Background Sync unavailable; use Replay queued writes as the fallback.")
				return
			}
			capabilities := registration.BackgroundSyncCapabilities()
			if !capabilities.OneShot {
				backgroundSyncStatus.Set("Background Sync unavailable; use Replay queued writes as the fallback.")
				return
			}
			if err := registration.RegisterSync(context.Background(), "gwc-offline-demo-replay"); err != nil {
				backgroundSyncStatus.Set(describeOfflineError("Background replay scheduling failed", err))
				return
			}
			backgroundSyncStatus.Set("Background replay sync registered with tag gwc-offline-demo-replay.")
		}()
	})
	replayWithConflictPolicy := ui.UseEvent(func() {
		queue := queueRef.Get()
		if queue == nil {
			conflictStatus.Set("Offline queue is not ready yet.")
			return
		}
		conflictStatus.Set("Replaying queued writes with conflict policy...")
		go func() {
			report, err := queue.ReplayWithOptions(context.Background(), func(ctx context.Context, mutation fetch.QueuedMutation) error {
				_ = ctx
				if mutation.Kind == "demo.conflict" && mutation.Metadata["resolved"] != "server-v2" {
					return fetch.NewMutationConflict(errors.New("etag mismatch"), fetch.MutationConflict{
						Code:          "etag_mismatch",
						Message:       "server revision is newer",
						LocalVersion:  mutation.Metadata["revision"],
						RemoteVersion: "server-v2",
					})
				}
				return nil
			}, fetch.MutationReplayOptions{ConflictHandler: func(ctx context.Context, mutation fetch.QueuedMutation, conflict fetch.MutationConflict) (fetch.MutationConflictResolution, error) {
				_ = ctx
				return fetch.MutationConflictResolution{
					Action:  fetch.MutationResolutionReplace,
					Message: fmt.Sprintf("Rebased %s onto %s.", mutation.ID, conflict.RemoteVersion),
					Draft: fetch.MutationDraft{
						Metadata: map[string]string{"source": "offline-cache-example", "revision": conflict.RemoteVersion, "resolved": conflict.RemoteVersion},
						Body:     map[string]any{"resolution": "rebased", "remoteRevision": conflict.RemoteVersion},
					},
				}, nil
			}})
			if err != nil {
				conflictStatus.Set(describeOfflineError("Conflict-aware replay failed", err))
				return
			}
			if report.Resolved > 0 {
				conflictStatus.Set(fmt.Sprintf("Resolved %d conflict and re-queued it for a follow-up replay. Remaining=%d.", report.Resolved, report.Remaining))
				return
			}
			conflictStatus.Set(fmt.Sprintf("Conflict-aware replay succeeded=%d dead=%d remaining=%d.", report.Succeeded, report.DeadLetters, report.Remaining))
		}()
	})
	inspectDiagnostics := ui.UseEvent(func() {
		options := pwa.DiagnosticsOptions{}
		if manager := cacheManagerRef.Get(); manager != nil {
			options.CacheStorage = manager
			options.CacheStoragePlan = &plan
		}
		if queue := queueRef.Get(); queue != nil {
			options.OfflineQueue = pwa.MutationQueueDiagnosticsSource(queue)
		}
		if registration := registrationRef.Get(); registration != nil {
			options.ServiceWorker = registration
		}
		diagnosticsPreview.Set("Capturing diagnostics snapshot...")
		go func() {
			snapshot, err := pwa.InspectDiagnostics(context.Background(), options)
			if err != nil {
				diagnosticsPreview.Set(describeOfflineError("Diagnostics snapshot failed", err))
				return
			}
			previewLines := []string{
				fmt.Sprintf("manifest valid: %t", snapshot.Manifest.Valid),
				fmt.Sprintf("cache entries: %d", snapshot.CacheStorage.EntryCount),
				fmt.Sprintf("queued writes: %d", snapshot.OfflineQueue.TotalEntries),
				fmt.Sprintf("storage pressure: %s", snapshot.Storage.Pressure),
				fmt.Sprintf("indexedDB bytes: %d", snapshot.Storage.IndexedDBBytes),
				fmt.Sprintf("cache storage bytes: %d", snapshot.Storage.CacheStorageBytes),
			}
			if snapshot.ServiceWorker.Scope != "" {
				previewLines = append(previewLines, "service worker scope: "+snapshot.ServiceWorker.Scope)
			}
			capabilitiesAvailable := false
			if registration := registrationRef.Get(); registration != nil {
				capabilitiesAvailable = registration.BackgroundSyncCapabilities().OneShot
			}
			previewLines = append(previewLines, fmt.Sprintf("background sync available: %t", capabilitiesAvailable))
			diagnosticsPreview.Set(strings.Join(previewLines, "\n"))
			diagnosticsSummary.Set(offlineSummary(snapshot))
		}()
	})

	return shared.ExamplePage(
		"PWA offline cache",
		"pwa.BuildCacheStoragePlan, pwa.OpenCacheStorageManager, pwa.InspectDiagnostics",
		"Warm a versioned offline shell, enqueue durable writes, schedule background replay where supported, and resolve replay conflicts through one structured diagnostics snapshot.",
		shared.ExamplePanel("Offline shell controls",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example keeps service-worker ownership explicit. Cache Storage warming, background sync scheduling, durable mutation replay, conflict policy, and diagnostics inspection are separate buttons so the deployment steps stay reviewable.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Warm offline cache", warmOfflineCache),
				shared.ExampleButton("Queue offline write", queueOfflineWrite),
				shared.ExampleButton("Queue conflicting write", queueConflictingWrite),
				shared.ExampleButton("Replay queued writes", replayQueuedWrites),
				shared.ExampleButton("Replay with conflict policy", replayWithConflictPolicy),
				shared.ExampleButton("Schedule background replay", scheduleBackgroundReplay),
				shared.ExampleButton("Inspect diagnostics", inspectDiagnostics),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-300", ID: "offline-cache-status"}, html.Text(cacheStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-queue-status"}, html.Text(queueStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-replay-status"}, html.Text(replayStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-background-sync-status"}, html.Text(backgroundSyncStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-conflict-status"}, html.Text(conflictStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-sw-status"}, html.Text(serviceWorkerStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-diagnostics-summary"}, html.Text(diagnosticsSummary.Get())),
		),
		shared.ExamplePanel("Structured diagnostics output",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The diagnostics helper merges Cache Storage inspection, offline queue state, service-worker lifecycle, and browser storage estimates into one typed snapshot.")),
			html.Pre(html.Props{Class: "mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "offline-diagnostics-preview"}, html.Text(diagnosticsPreview.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(offlineDiagnosticsExample), "#app")
	select {}
}
