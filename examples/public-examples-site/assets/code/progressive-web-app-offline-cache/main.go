//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/fetch"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/pwa"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func offlineCachePlan() pwa.CacheStoragePlan {
	parsePlan, _ := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{
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
	return parsePlan
}

func offlineSummary(parseSnapshot pwa.DiagnosticsSnapshot) string {
	return fmt.Sprintf("cache entries=%d | queued=%d | retrying=%d | dead=%d | storage=%s %.0f%% used", parseSnapshot.CacheStorage.EntryCount, parseSnapshot.OfflineQueue.QueuedEntries, parseSnapshot.OfflineQueue.RetryingEntries, parseSnapshot.OfflineQueue.DeadEntries, parseSnapshot.Storage.Pressure, parseSnapshot.Storage.UsageRatio*100)
}

func describeOfflineError(parsePrefix string, parseErr error) string {
	if parseErr == nil {
		return parsePrefix
	}
	return fmt.Sprintf("%s: %v", parsePrefix, parseErr)
}

func offlineDiagnosticsExample() ui.Node {
	parsePlan := offlineCachePlan()
	cacheManagerRef := ui.UseRef[*pwa.CacheStorageManager](nil)
	cacheManagerStartedRef := ui.UseRef(false)
	parseQueueRef := ui.UseRef[*fetch.MutationQueue](nil)
	parseQueueStartedRef := ui.UseRef(false)
	parseRegistrationRef := ui.UseRef[*pwa.ServiceWorkerRegistration](nil)
	parseServiceWorkerStartedRef := ui.UseRef(false)
	cacheStatus := ui.UseState("Offline cache is idle.")
	parseQueueStatus := ui.UseState("No queued writes yet.")
	parseReplayStatus := ui.UseState("No replay attempted yet.")
	parseBackgroundSyncStatus := ui.UseState("Background replay scheduling idle.")
	parseConflictStatus := ui.UseState("No conflict resolution attempted yet.")
	parseServiceWorkerStatus := ui.UseState("Service worker registration pending.")
	parseDiagnosticsPreview := ui.UseState("Click Inspect diagnostics to capture a structured PWA snapshot.")
	parseDiagnosticsSummary := ui.UseState("No snapshot captured yet.")
	parseEnsureServiceWorkerRegistration := func() (*pwa.ServiceWorkerRegistration, error) {
		if parseRegistration := parseRegistrationRef.Get(); parseRegistration != nil {
			return parseRegistration, nil
		}
		parseRegistration2, parseErr := pwa.RegisterServiceWorker(context.Background(), pwa.ServiceWorkerOptions{
			URL:   "/97-pwa-offline-cache/sw.js",
			Scope: "/97-pwa-offline-cache/",
		})
		if parseErr != nil {
			return nil, parseErr
		}
		parseRegistrationRef.Set(&parseRegistration2)
		parseCapabilities := parseRegistration2.BackgroundSyncCapabilities()
		if parseCapabilities.OneShot {
			parseServiceWorkerStatus.Set("Service worker registered for offline shell fallback. Background Sync is available.")
		} else {
			parseServiceWorkerStatus.Set("Service worker registered for offline shell fallback. Background Sync is unavailable, so manual replay remains the fallback.")
		}
		return &parseRegistration2, nil
	}

	ui.UseEffect(func() func() {
		if cacheManagerRef.Get() == nil && !cacheManagerStartedRef.Get() {
			cacheManagerStartedRef.Set(true)
			go func() {
				parseManager, parseErr2 := pwa.OpenCacheStorageManager()
				if parseErr2 != nil {
					cacheStatus.Set(describeOfflineError("Cache Storage manager unavailable", parseErr2))
					return
				}
				cacheManagerRef.Set(&parseManager)
				cacheStatus.Set("Cache Storage manager ready.")
			}()
		}
		if parseQueueRef.Get() == nil && !parseQueueStartedRef.Get() {
			parseQueueStartedRef.Set(true)
			go func() {
				parseQueue, parseErr3 := fetch.OpenMutationQueue(fetch.MutationQueueOptions{DeleteOnCorruption: false})
				if parseErr3 != nil {
					parseQueueStatus.Set(describeOfflineError("Offline mutation queue unavailable", parseErr3))
					return
				}
				parseQueueRef.Set(&parseQueue)
				parseQueueStatus.Set("Offline mutation queue ready.")
			}()
		}
		return nil
	}, "offline-cache-initializers")

	ui.UseEffect(func() func() {
		if parseRegistrationRef.Get() == nil && !parseServiceWorkerStartedRef.Get() {
			parseServiceWorkerStartedRef.Set(true)
			parseServiceWorkerStatus.Set("Registering service worker...")
			go func() {
				var parseErr4 error
				for parseAttempt := 0; parseAttempt < 2; parseAttempt++ {
					_, parseErr4 = parseEnsureServiceWorkerRegistration()
					if parseErr4 == nil {
						break
					}
					if parseAttempt == 0 {
						parseServiceWorkerStatus.Set("Retrying service worker registration...")
						time.Sleep(250 * time.Millisecond)
					}
				}
				if parseErr4 != nil {
					parseServiceWorkerStatus.Set(describeOfflineError("Service worker registration failed", parseErr4))
					return
				}
			}()
		}
		return nil
	}, "offline-cache-service-worker")

	parseWarmOfflineCache := ui.UseEvent(func() {
		parseManager2 := cacheManagerRef.Get()
		if parseManager2 == nil {
			cacheStatus.Set("Cache Storage manager is not ready yet.")
			return
		}
		cacheStatus.Set("Warming offline cache...")
		go func() {
			parseSnapshot, parseErr5 := parseManager2.Sync(context.Background(), parsePlan)
			if parseErr5 != nil {
				cacheStatus.Set(describeOfflineError("Cache warmup failed", parseErr5))
				return
			}
			cacheStatus.Set(fmt.Sprintf("Cached %d release entries into %s.", parseSnapshot.EntryCount, parseSnapshot.CacheName))
		}()
	})
	parseQueueOfflineWrite := ui.UseEvent(func() {
		parseQueue2 := parseQueueRef.Get()
		if parseQueue2 == nil {
			parseQueueStatus.Set("Offline queue is not ready yet.")
			return
		}
		parseQueueStatus.Set("Queueing offline write...")
		go func() {
			parseEntry, parseErr6 := parseQueue2.Enqueue(fetch.MutationDraft{URL: "/api/offline-demo", Method: "POST", Kind: "demo.sync", Metadata: map[string]string{"source": "offline-cache-example"}})
			if parseErr6 != nil {
				parseQueueStatus.Set(describeOfflineError("Queue write failed", parseErr6))
				return
			}
			parseQueueStatus.Set(fmt.Sprintf("Queued offline write %s with state=%s.", parseEntry.ID, parseEntry.State))
		}()
	})
	parseQueueConflictingWrite := ui.UseEvent(func() {
		parseQueue3 := parseQueueRef.Get()
		if parseQueue3 == nil {
			parseConflictStatus.Set("Offline queue is not ready yet.")
			return
		}
		parseConflictStatus.Set("Queueing a conflict demo write...")
		go func() {
			parseEntry2, parseErr7 := parseQueue3.Enqueue(fetch.MutationDraft{
				URL:      "/api/offline-demo",
				Method:   "POST",
				Kind:     "demo.conflict",
				DedupKey: "conflict:offline-cache-example",
				Metadata: map[string]string{"source": "offline-cache-example", "revision": "local-1"},
			})
			if parseErr7 != nil {
				parseConflictStatus.Set(describeOfflineError("Queue conflict write failed", parseErr7))
				return
			}
			parseConflictStatus.Set(fmt.Sprintf("Queued conflict demo write %s at revision %s.", parseEntry2.ID, parseEntry2.Metadata["revision"]))
		}()
	})
	parseReplayQueuedWrites := ui.UseEvent(func() {
		parseQueue4 := parseQueueRef.Get()
		if parseQueue4 == nil {
			parseReplayStatus.Set("Offline queue is not ready yet.")
			return
		}
		parseReplayStatus.Set("Replaying queued writes...")
		go func() {
			parseReport, parseErr8 := parseQueue4.Replay(context.Background(), func(parseCtx context.Context, parseMutation fetch.QueuedMutation) error {
				_ = parseCtx
				_ = parseMutation
				return nil
			})
			if parseErr8 != nil {
				parseReplayStatus.Set(describeOfflineError("Queued write replay failed", parseErr8))
				return
			}
			parseReplayStatus.Set(fmt.Sprintf("Replay succeeded=%d retried=%d dead=%d remaining=%d.", parseReport.Succeeded, parseReport.Retried, parseReport.DeadLetters, parseReport.Remaining))
		}()
	})
	parseScheduleBackgroundReplay := ui.UseEvent(func() {
		parseBackgroundSyncStatus.Set("Scheduling background replay...")
		go func() {
			parseRegistration3, parseErr9 := parseEnsureServiceWorkerRegistration()
			if parseErr9 != nil {
				parseBackgroundSyncStatus.Set("Background Sync unavailable; use Replay queued writes as the fallback.")
				return
			}
			parseCapabilities2 := parseRegistration3.BackgroundSyncCapabilities()
			if !parseCapabilities2.OneShot {
				parseBackgroundSyncStatus.Set("Background Sync unavailable; use Replay queued writes as the fallback.")
				return
			}
			if parseErr10 := parseRegistration3.RegisterSync(context.Background(), "gwc-offline-demo-replay"); parseErr10 != nil {
				parseBackgroundSyncStatus.Set(describeOfflineError("Background replay scheduling failed", parseErr10))
				return
			}
			parseBackgroundSyncStatus.Set("Background replay sync registered with tag gwc-offline-demo-replay.")
		}()
	})
	parseReplayWithConflictPolicy := ui.UseEvent(func() {
		parseQueue5 := parseQueueRef.Get()
		if parseQueue5 == nil {
			parseConflictStatus.Set("Offline queue is not ready yet.")
			return
		}
		parseConflictStatus.Set("Replaying queued writes with conflict policy...")
		go func() {
			parseReport2, parseErr11 := parseQueue5.ReplayWithOptions(context.Background(), func(parseCtx2 context.Context, parseMutation2 fetch.QueuedMutation) error {
				_ = parseCtx2
				if parseMutation2.Kind == "demo.conflict" && parseMutation2.Metadata["resolved"] != "server-v2" {
					return fetch.NewMutationConflict(errors.New("etag mismatch"), fetch.MutationConflict{
						Code:          "etag_mismatch",
						Message:       "server revision is newer",
						LocalVersion:  parseMutation2.Metadata["revision"],
						RemoteVersion: "server-v2",
					})
				}
				return nil
			}, fetch.MutationReplayOptions{ConflictHandler: func(parseCtx3 context.Context, parseMutation3 fetch.QueuedMutation, parseConflict fetch.MutationConflict) (fetch.MutationConflictResolution, error) {
				_ = parseCtx3
				return fetch.MutationConflictResolution{
					Action:  fetch.MutationResolutionReplace,
					Message: fmt.Sprintf("Rebased %s onto %s.", parseMutation3.ID, parseConflict.RemoteVersion),
					Draft: fetch.MutationDraft{
						Metadata: map[string]string{"source": "offline-cache-example", "revision": parseConflict.RemoteVersion, "resolved": parseConflict.RemoteVersion},
						Body:     map[string]any{"resolution": "rebased", "remoteRevision": parseConflict.RemoteVersion},
					},
				}, nil
			}})
			if parseErr11 != nil {
				parseConflictStatus.Set(describeOfflineError("Conflict-aware replay failed", parseErr11))
				return
			}
			if parseReport2.Resolved > 0 {
				parseConflictStatus.Set(fmt.Sprintf("Resolved %d conflict and re-queued it for a follow-up replay. Remaining=%d.", parseReport2.Resolved, parseReport2.Remaining))
				return
			}
			parseConflictStatus.Set(fmt.Sprintf("Conflict-aware replay succeeded=%d dead=%d remaining=%d.", parseReport2.Succeeded, parseReport2.DeadLetters, parseReport2.Remaining))
		}()
	})
	parseInspectDiagnostics := ui.UseEvent(func() {
		parseOptions := pwa.DiagnosticsOptions{}
		if parseManager3 := cacheManagerRef.Get(); parseManager3 != nil {
			parseOptions.CacheStorage = parseManager3
			parseOptions.CacheStoragePlan = &parsePlan
		}
		if parseQueue6 := parseQueueRef.Get(); parseQueue6 != nil {
			parseOptions.OfflineQueue = pwa.BuildMutationQueueDiagnosticsSource(parseQueue6)
		}
		if parseRegistration4 := parseRegistrationRef.Get(); parseRegistration4 != nil {
			parseOptions.ServiceWorker = parseRegistration4
		}
		parseDiagnosticsPreview.Set("Capturing diagnostics snapshot...")
		go func() {
			parseSnapshot2, parseErr12 := pwa.InspectDiagnostics(context.Background(), parseOptions)
			if parseErr12 != nil {
				parseDiagnosticsPreview.Set(describeOfflineError("Diagnostics snapshot failed", parseErr12))
				return
			}
			parsePreviewLines := []string{
				fmt.Sprintf("manifest valid: %t", parseSnapshot2.Manifest.Valid),
				fmt.Sprintf("cache entries: %d", parseSnapshot2.CacheStorage.EntryCount),
				fmt.Sprintf("queued writes: %d", parseSnapshot2.OfflineQueue.TotalEntries),
				fmt.Sprintf("storage pressure: %s", parseSnapshot2.Storage.Pressure),
				fmt.Sprintf("indexedDB bytes: %d", parseSnapshot2.Storage.IndexedDBBytes),
				fmt.Sprintf("cache storage bytes: %d", parseSnapshot2.Storage.CacheStorageBytes),
			}
			if parseSnapshot2.ServiceWorker.Scope != "" {
				parsePreviewLines = append(parsePreviewLines, "service worker scope: "+parseSnapshot2.ServiceWorker.Scope)
			}
			isParseCapabilitiesAvailable := false
			if parseRegistration5 := parseRegistrationRef.Get(); parseRegistration5 != nil {
				isParseCapabilitiesAvailable = parseRegistration5.BackgroundSyncCapabilities().OneShot
			}
			parsePreviewLines = append(parsePreviewLines, fmt.Sprintf("background sync available: %t", isParseCapabilitiesAvailable))
			parseDiagnosticsPreview.Set(strings.Join(parsePreviewLines, "\n"))
			parseDiagnosticsSummary.Set(offlineSummary(parseSnapshot2))
		}()
	})

	return shared.ExamplePage(
		"PWA offline cache",
		"pwa.BuildCacheStoragePlan, pwa.OpenCacheStorageManager, pwa.InspectDiagnostics",
		"Warm a versioned offline shell, enqueue durable writes, schedule background replay where supported, and resolve replay conflicts through one structured diagnostics snapshot.",
		shared.ExamplePanel("Offline shell controls",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example keeps service-worker ownership explicit. Cache Storage warming, background sync scheduling, durable mutation replay, conflict policy, and diagnostics inspection are separate buttons so the deployment steps stay reviewable.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Warm offline cache", parseWarmOfflineCache),
				shared.ExampleButton("Queue offline write", parseQueueOfflineWrite),
				shared.ExampleButton("Queue conflicting write", parseQueueConflictingWrite),
				shared.ExampleButton("Replay queued writes", parseReplayQueuedWrites),
				shared.ExampleButton("Replay with conflict policy", parseReplayWithConflictPolicy),
				shared.ExampleButton("Schedule background replay", parseScheduleBackgroundReplay),
				shared.ExampleButton("Inspect diagnostics", parseInspectDiagnostics),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-300", ID: "offline-cache-status"}, html.Text(cacheStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-queue-status"}, html.Text(parseQueueStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-replay-status"}, html.Text(parseReplayStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-background-sync-status"}, html.Text(parseBackgroundSyncStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-conflict-status"}, html.Text(parseConflictStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-sw-status"}, html.Text(parseServiceWorkerStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "offline-diagnostics-summary"}, html.Text(parseDiagnosticsSummary.Get())),
		),
		shared.ExamplePanel("Structured diagnostics output",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The diagnostics helper merges Cache Storage inspection, offline queue state, service-worker lifecycle, and browser storage estimates into one typed snapshot.")),
			html.Pre(html.Props{Class: "mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "offline-diagnostics-preview"}, html.Text(parseDiagnosticsPreview.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(offlineDiagnosticsExample))
	exampleboot.WaitExampleRuntime()
}
