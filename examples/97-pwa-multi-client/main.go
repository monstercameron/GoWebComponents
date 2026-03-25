//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/pwa"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	multiClientPWAChannelName     = "example:pwa:multi-client"
	multiClientPWASyncTopic       = "pwa:sync"
	multiClientPWAInvalidateTopic = "pwa:shell"
)

type multiClientPWAPeer struct {
	Identity interop.ClientIdentity
	State    string
	Summary  string
	LastSeen time.Time
}

func newMultiClientPWAIdentity() interop.ClientIdentity {
	return interop.ClientIdentity{
		ID:      fmt.Sprintf("pwa-tab-%d", time.Now().UTC().UnixNano()),
		App:     "examples/pwa-multi-client",
		Surface: "pwa-tab",
		Role:    "client",
		Version: "v1",
	}
}

func multiClientPWAManifest() pwa.Manifest {
	return pwa.Manifest{
		ID:              "/97-pwa-multi-client/",
		Name:            "GoWebComponents PWA Multi-Client Demo",
		ShortName:       "PWA Peers",
		Description:     "Cross-tab inter-wasm coordination layered onto a PWA shell with explicit service-worker and cache ownership.",
		StartURL:        "/97-pwa-multi-client/pwa-multi-client.html",
		Scope:           "/97-pwa-multi-client/",
		Display:         "standalone",
		ThemeColor:      "#0f172a",
		BackgroundColor: "#08111d",
		Icons: []pwa.ManifestImage{
			{Src: "/static/images/favicon/android-chrome-192x192.png", Sizes: "192x192", Type: "image/png"},
			{Src: "/static/images/favicon/android-chrome-512x512.png", Sizes: "512x512", Type: "image/png"},
		},
	}
}

func multiClientPWACachePlan() pwa.CacheStoragePlan {
	parsePlan, _ := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{
		CacheName:        "pwa-multi-client-demo-v1",
		ManifestRevision: "demo-v1",
		WasmURL:          "/static/bin/pwa-multi-client.wasm",
		ShellURLs:        []string{"/97-pwa-multi-client/pwa-multi-client.html", "/97-pwa-multi-client/offline.html"},
		ImmutableURLs: []string{
			"/static/css/tailwind.css",
			"/static/css/example-shell.css",
			"/static/script/wasm_exec.js",
		},
	}, pwa.CacheStoragePlanOptions{CachePrefix: "pwa-multi-client-demo-"})
	return parsePlan
}

func cloneMultiClientPWAPeers(parsePrevious map[string]multiClientPWAPeer) map[string]multiClientPWAPeer {
	parseNext := make(map[string]multiClientPWAPeer, len(parsePrevious))
	for parseKey, parseValue := range parsePrevious {
		parseNext[parseKey] = parseValue
	}
	return parseNext
}

func describeMultiClientPWAError(parsePrefix string, parseErr error) string {
	if parseErr == nil {
		return parsePrefix
	}
	if parseCode, parseOk := interop.CodeOf(parseErr); parseOk {
		return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr)
	}
	return fmt.Sprintf("%s: %v", parsePrefix, parseErr)
}

func formatMultiClientPWAPayload(parsePayload any) string {
	parseText := strings.TrimSpace(fmt.Sprintf("%v", parsePayload))
	if parseText == "" || parseText == "<nil>" {
		return "No payload"
	}
	return parseText
}

func renderMultiClientPWAPeers(parsePeers map[string]multiClientPWAPeer) []ui.Node {
	if len(parsePeers) == 0 {
		return []ui.Node{
			html.Li(html.Props{Class: "rounded-2xl border border-dashed border-white/10 bg-slate-950/35 px-4 py-3 text-sm leading-7 text-slate-400"}, html.Text("Open this page in a second tab to watch another wasm client appear here through the cross-tab multi-client channel.")),
		}
	}
	parseKeys := make([]string, 0, len(parsePeers))
	for parseKey := range parsePeers {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseNodes := make([]ui.Node, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parsePeer := parsePeers[parseKey2]
		parseLabel := fmt.Sprintf("%s | %s | %s | last seen %s", parsePeer.Identity.Surface, parsePeer.State, parsePeer.Summary, parsePeer.LastSeen.Local().Format("15:04:05"))
		parseNodes = append(parseNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseLabel)),
		)
	}
	return parseNodes
}

func multiClientPWASummary(parseSnapshot pwa.DiagnosticsSnapshot, parsePeerCount int) string {
	return fmt.Sprintf("peers=%d | cache entries=%d | queued=%d | storage=%s", parsePeerCount, parseSnapshot.CacheStorage.EntryCount, parseSnapshot.OfflineQueue.TotalEntries, parseSnapshot.Storage.Pressure)
}

func multiClientPWARoot() ui.Node {
	parseSelf := ui.UseState(newMultiClientPWAIdentity())
	parseManifest := multiClientPWAManifest()
	cachePlan := multiClientPWACachePlan()
	parseTransportStatus := ui.UseState("pending")
	parseServiceWorkerStatus := ui.UseState("Registering service worker...")
	cacheStatus := ui.UseState("Offline shell is idle.")
	parseLastSent := ui.UseState("No multi-client traffic sent yet.")
	parseLastReceived := ui.UseState("No multi-client traffic received yet.")
	parseDiagnosticsSummary := ui.UseState("No diagnostics snapshot captured yet.")
	parseDiagnosticsPreview := ui.UseState("Click Inspect diagnostics after warming the shell or exchanging peer traffic.")
	parsePeers := ui.UseState(map[string]multiClientPWAPeer{})
	parseChannelRef := ui.UseRef(interop.CrossTabChannel{})
	parseChannelCancelRef := ui.UseRef((func())(nil))
	cacheManagerRef := ui.UseRef[*pwa.CacheStorageManager](nil)
	parseRegistrationRef := ui.UseRef[*pwa.ServiceWorkerRegistration](nil)
	cacheManagerStartedRef := ui.UseRef(false)
	parseServiceWorkerStartedRef := ui.UseRef(false)

	parseUpsertPeer := func(parseIdentity interop.ClientIdentity, parseState string, parseSummary string) {
		if strings.TrimSpace(parseIdentity.ID) == "" || parseIdentity.ID == parseSelf.Get().ID {
			return
		}
		parseNext := cloneMultiClientPWAPeers(parsePeers.Get())
		parseEntry := parseNext[parseIdentity.ID]
		parseEntry.Identity = parseIdentity
		parseEntry.State = parseState
		parseEntry.Summary = parseSummary
		parseEntry.LastSeen = time.Now().UTC()
		parseNext[parseIdentity.ID] = parseEntry
		parsePeers.Set(parseNext)
	}

	ui.UseEffect(func() func() {
		if cacheManagerRef.Get() == nil && !cacheManagerStartedRef.Get() {
			cacheManagerStartedRef.Set(true)
			go func() {
				parseManager, parseErr := pwa.OpenCacheStorageManager()
				if parseErr != nil {
					cacheStatus.Set(describeMultiClientPWAError("Cache Storage manager unavailable", parseErr))
					return
				}
				cacheManagerRef.Set(&parseManager)
				cacheStatus.Set("Cache Storage manager ready.")
			}()
		}
		return nil
	}, "multi-client-pwa-cache-manager")

	ui.UseEffect(func() func() {
		if parseRegistrationRef.Get() == nil && !parseServiceWorkerStartedRef.Get() {
			parseServiceWorkerStartedRef.Set(true)
			go func() {
				parseRegistration, parseErr2 := pwa.RegisterServiceWorker(context.Background(), pwa.ServiceWorkerOptions{
					URL:   "/97-pwa-multi-client/sw.js",
					Scope: "/97-pwa-multi-client/",
				})
				if parseErr2 != nil {
					parseServiceWorkerStatus.Set(describeMultiClientPWAError("Service worker registration failed", parseErr2))
					return
				}
				parseRegistrationRef.Set(&parseRegistration)
				parseServiceWorkerStatus.Set("Service worker registered for the multi-client PWA shell.")
			}()
		}
		return nil
	}, "multi-client-pwa-service-worker")

	ui.UseEffect(func() func() {
		parseChannel, parseErr3 := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: multiClientPWAChannelName})
		if parseErr3 != nil {
			parseTransportStatus.Set(describeMultiClientPWAError("Cross-tab channel unavailable", parseErr3))
			return nil
		}
		parseChannelRef.Set(parseChannel)
		parseTransportStatus.Set(parseChannel.Transport())

		parseSubscription, parseErr3 := interop.SubscribeClientMessages(parseChannel, func(parseMessage interop.ClientMessage, parseReceiveErr error) {
			if parseReceiveErr != nil {
				parseLastReceived.Set(describeMultiClientPWAError("Peer message failed", parseReceiveErr))
				return
			}
			if parseMessage.Source.ID == parseSelf.Get().ID {
				return
			}
			switch parseMessage.Kind {
			case interop.ClientHello:
				parseUpsertPeer(parseMessage.Source, "ready", "hello received")
				parseLastReceived.Set(fmt.Sprintf("Peer hello from %s over %s.", parseMessage.Source.ID, parseChannel.Transport()))
			case interop.ClientEvent:
				if parseMessage.Topic != multiClientPWASyncTopic {
					return
				}
				parseUpsertPeer(parseMessage.Source, "event", formatMultiClientPWAPayload(parseMessage.Payload))
				parseLastReceived.Set(fmt.Sprintf("Received sync event from %s: %s", parseMessage.Source.ID, formatMultiClientPWAPayload(parseMessage.Payload)))
			case interop.ClientInvalidate:
				if parseMessage.Topic != multiClientPWAInvalidateTopic {
					return
				}
				parseUpsertPeer(parseMessage.Source, "invalidate", parseMessage.Revision)
				parseLastReceived.Set(fmt.Sprintf("Received cache invalidation for %s rev %s.", parseMessage.Topic, parseMessage.Revision))
			case interop.ClientGoodbye:
				parseNext2 := cloneMultiClientPWAPeers(parsePeers.Get())
				parseEntry2 := parseNext2[parseMessage.Source.ID]
				parseEntry2.Identity = parseMessage.Source
				parseEntry2.State = "disconnected"
				parseEntry2.Summary = "goodbye received"
				parseEntry2.LastSeen = time.Now().UTC()
				parseNext2[parseMessage.Source.ID] = parseEntry2
				parsePeers.Set(parseNext2)
				parseLastReceived.Set(fmt.Sprintf("Peer goodbye from %s.", parseMessage.Source.ID))
			}
		})
		if parseErr3 != nil {
			parseTransportStatus.Set(describeMultiClientPWAError("Cross-tab subscription failed", parseErr3))
			_ = parseChannel.Close()
			parseChannelRef.Set(interop.CrossTabChannel{})
			return nil
		}
		parseChannelCancelRef.Set(parseSubscription.Cancel)

		if parseErr4 := interop.PublishClientHello(parseChannel, parseSelf.Get()); parseErr4 != nil {
			parseLastSent.Set(describeMultiClientPWAError("Initial hello failed", parseErr4))
		} else {
			parseLastSent.Set("Published initial multi-client hello.")
		}

		return func() {
			if parseCancel := parseChannelCancelRef.Get(); parseCancel != nil {
				parseCancel()
				parseChannelCancelRef.Set(nil)
			}
			_ = interop.PublishClientGoodbye(parseChannel, parseSelf.Get())
			_ = parseChannel.Close()
			parseChannelRef.Set(interop.CrossTabChannel{})
		}
	}, true)

	parseAnnouncePeer := ui.UseEvent(func() {
		parseChannel2 := parseChannelRef.Get()
		if parseChannel2.Name() == "" {
			parseLastSent.Set("Cross-tab channel is not ready yet.")
			return
		}
		if parseErr5 := interop.PublishClientHello(parseChannel2, parseSelf.Get()); parseErr5 != nil {
			parseLastSent.Set(describeMultiClientPWAError("Peer hello failed", parseErr5))
			return
		}
		parseLastSent.Set("Re-announced peer hello to the multi-client channel.")
	})

	parseBroadcastSyncEvent := ui.UseEvent(func() {
		parseChannel3 := parseChannelRef.Get()
		if parseChannel3.Name() == "" {
			parseLastSent.Set("Cross-tab channel is not ready yet.")
			return
		}
		parsePayload := map[string]string{
			"status":    "shell-ready",
			"transport": parseTransportStatus.Get(),
			"sender":    parseSelf.Get().ID,
		}
		if parseErr6 := interop.PublishClientEvent(parseChannel3, multiClientPWASyncTopic, parseSelf.Get(), parsePayload); parseErr6 != nil {
			parseLastSent.Set(describeMultiClientPWAError("Sync event failed", parseErr6))
			return
		}
		parseLastSent.Set("Broadcast sync event across the inter-wasm peer channel.")
	})

	parseBroadcastCacheInvalidation := ui.UseEvent(func() {
		parseChannel4 := parseChannelRef.Get()
		if parseChannel4.Name() == "" {
			parseLastSent.Set("Cross-tab channel is not ready yet.")
			return
		}
		if parseErr7 := interop.PublishClientInvalidation(parseChannel4, multiClientPWAInvalidateTopic, parseSelf.Get(), cachePlan.ManifestRevision); parseErr7 != nil {
			parseLastSent.Set(describeMultiClientPWAError("Cache invalidation failed", parseErr7))
			return
		}
		parseLastSent.Set(fmt.Sprintf("Broadcast cache invalidation rev %s.", cachePlan.ManifestRevision))
	})

	parseWarmOfflineShell := ui.UseEvent(func() {
		parseManager2 := cacheManagerRef.Get()
		if parseManager2 == nil {
			cacheStatus.Set("Cache Storage manager is not ready yet.")
			return
		}
		cacheStatus.Set("Warming multi-client offline shell...")
		go func() {
			parseSnapshot, parseErr8 := parseManager2.Sync(context.Background(), cachePlan)
			if parseErr8 != nil {
				cacheStatus.Set(describeMultiClientPWAError("Shell warmup failed", parseErr8))
				return
			}
			cacheStatus.Set(fmt.Sprintf("Cached %d entries into %s.", parseSnapshot.EntryCount, parseSnapshot.CacheName))
		}()
	})

	parseInspectDiagnostics := ui.UseEvent(func() {
		parseOptions := pwa.DiagnosticsOptions{Manifest: &parseManifest}
		if parseManager3 := cacheManagerRef.Get(); parseManager3 != nil {
			parseOptions.CacheStorage = parseManager3
			parseOptions.CacheStoragePlan = &cachePlan
		}
		if parseRegistration2 := parseRegistrationRef.Get(); parseRegistration2 != nil {
			parseOptions.ServiceWorker = parseRegistration2
		}
		parseDiagnosticsPreview.Set("Capturing diagnostics snapshot...")
		go func() {
			parseSnapshot2, parseErr9 := pwa.InspectDiagnostics(context.Background(), parseOptions)
			if parseErr9 != nil {
				parseDiagnosticsPreview.Set(describeMultiClientPWAError("Diagnostics snapshot failed", parseErr9))
				return
			}
			parseLines := []string{
				fmt.Sprintf("manifest valid: %t", parseSnapshot2.Manifest.Valid),
				fmt.Sprintf("cache entries: %d", parseSnapshot2.CacheStorage.EntryCount),
				fmt.Sprintf("storage pressure: %s", parseSnapshot2.Storage.Pressure),
			}
			if parseSnapshot2.ServiceWorker.Scope != "" {
				parseLines = append(parseLines, "service worker scope: "+parseSnapshot2.ServiceWorker.Scope)
			}
			parseDiagnosticsPreview.Set(strings.Join(parseLines, "\n"))
			parseDiagnosticsSummary.Set(multiClientPWASummary(parseSnapshot2, len(parsePeers.Get())))
		}()
	})

	return shared.ExamplePage(
		"PWA multi-client coordination",
		"pwa.RegisterServiceWorker",
		"Run two wasm tabs under one PWA-shaped shell, exchange multi-client peer messages, and inspect the same service-worker and Cache Storage state that coordinates the offline shell.",
		shared.ExamplePanel("PWA and peer controls",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example combines the multi-client cross-tab message helpers with explicit PWA ownership. Each tab is its own wasm client, but they still coordinate shell revision and peer state through a typed browser-local channel.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Announce peer", parseAnnouncePeer),
				shared.ExampleButton("Broadcast sync event", parseBroadcastSyncEvent),
				shared.ExampleButton("Broadcast cache invalidation", parseBroadcastCacheInvalidation),
				shared.ExampleButton("Warm offline shell", parseWarmOfflineShell),
				shared.ExampleButton("Inspect diagnostics", parseInspectDiagnostics),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-3 md:grid-cols-3"},
				shared.ExampleStat("Transport", parseTransportStatus.Get()),
				shared.ExampleStat("Peer count", fmt.Sprintf("%d", len(parsePeers.Get()))),
				shared.ExampleStat("Shell revision", cachePlan.ManifestRevision),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-300", ID: "multi-client-pwa-sw-status"}, html.Text(parseServiceWorkerStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-cache-status"}, html.Text(cacheStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-last-sent"}, html.Text(parseLastSent.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-last-received"}, html.Text(parseLastReceived.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-diagnostics-summary"}, html.Text(parseDiagnosticsSummary.Get())),
		),
		shared.ExamplePanel("Discovered peers",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Open a second tab of this same page. Each tab publishes a multi-client hello and receives typed event or invalidation messages from the other wasm runtime.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3", ID: "multi-client-pwa-peer-list"}, renderMultiClientPWAPeers(parsePeers.Get())...),
		),
		shared.ExamplePanel("Structured diagnostics output",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The diagnostics helper stays separate from peer transport. The page combines them at the app layer so PWA shell state and inter-wasm coordination remain reviewable instead of implicit.")),
			html.Pre(html.Props{Class: "mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "multi-client-pwa-diagnostics-preview"}, html.Text(parseDiagnosticsPreview.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiClientPWARoot), "#app")
	select {}
}
