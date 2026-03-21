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
	plan, _ := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{
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
	return plan
}

func cloneMultiClientPWAPeers(previous map[string]multiClientPWAPeer) map[string]multiClientPWAPeer {
	next := make(map[string]multiClientPWAPeer, len(previous))
	for key, value := range previous {
		next[key] = value
	}
	return next
}

func describeMultiClientPWAError(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	if code, ok := interop.CodeOf(err); ok {
		return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

func formatMultiClientPWAPayload(payload any) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", payload))
	if text == "" || text == "<nil>" {
		return "No payload"
	}
	return text
}

func renderMultiClientPWAPeers(peers map[string]multiClientPWAPeer) []ui.Node {
	if len(peers) == 0 {
		return []ui.Node{
			html.Li(html.Props{Class: "rounded-2xl border border-dashed border-white/10 bg-slate-950/35 px-4 py-3 text-sm leading-7 text-slate-400"}, html.Text("Open this page in a second tab to watch another wasm client appear here through the cross-tab multi-client channel.")),
		}
	}
	keys := make([]string, 0, len(peers))
	for key := range peers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	nodes := make([]ui.Node, 0, len(keys))
	for _, key := range keys {
		peer := peers[key]
		label := fmt.Sprintf("%s | %s | %s | last seen %s", peer.Identity.Surface, peer.State, peer.Summary, peer.LastSeen.Local().Format("15:04:05"))
		nodes = append(nodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(label)),
		)
	}
	return nodes
}

func multiClientPWASummary(snapshot pwa.DiagnosticsSnapshot, peerCount int) string {
	return fmt.Sprintf("peers=%d | cache entries=%d | queued=%d | storage=%s", peerCount, snapshot.CacheStorage.EntryCount, snapshot.OfflineQueue.TotalEntries, snapshot.Storage.Pressure)
}

func multiClientPWARoot() ui.Node {
	self := ui.UseState(newMultiClientPWAIdentity())
	manifest := multiClientPWAManifest()
	cachePlan := multiClientPWACachePlan()
	transportStatus := ui.UseState("pending")
	serviceWorkerStatus := ui.UseState("Registering service worker...")
	cacheStatus := ui.UseState("Offline shell is idle.")
	lastSent := ui.UseState("No multi-client traffic sent yet.")
	lastReceived := ui.UseState("No multi-client traffic received yet.")
	diagnosticsSummary := ui.UseState("No diagnostics snapshot captured yet.")
	diagnosticsPreview := ui.UseState("Click Inspect diagnostics after warming the shell or exchanging peer traffic.")
	peers := ui.UseState(map[string]multiClientPWAPeer{})
	channelRef := ui.UseRef(interop.CrossTabChannel{})
	channelCancelRef := ui.UseRef((func())(nil))
	cacheManagerRef := ui.UseRef[*pwa.CacheStorageManager](nil)
	registrationRef := ui.UseRef[*pwa.ServiceWorkerRegistration](nil)
	cacheManagerStartedRef := ui.UseRef(false)
	serviceWorkerStartedRef := ui.UseRef(false)

	upsertPeer := func(identity interop.ClientIdentity, state string, summary string) {
		if strings.TrimSpace(identity.ID) == "" || identity.ID == self.Get().ID {
			return
		}
		next := cloneMultiClientPWAPeers(peers.Get())
		entry := next[identity.ID]
		entry.Identity = identity
		entry.State = state
		entry.Summary = summary
		entry.LastSeen = time.Now().UTC()
		next[identity.ID] = entry
		peers.Set(next)
	}

	ui.UseEffect(func() func() {
		if cacheManagerRef.Get() == nil && !cacheManagerStartedRef.Get() {
			cacheManagerStartedRef.Set(true)
			go func() {
				manager, err := pwa.OpenCacheStorageManager()
				if err != nil {
					cacheStatus.Set(describeMultiClientPWAError("Cache Storage manager unavailable", err))
					return
				}
				cacheManagerRef.Set(&manager)
				cacheStatus.Set("Cache Storage manager ready.")
			}()
		}
		return nil
	}, "multi-client-pwa-cache-manager")

	ui.UseEffect(func() func() {
		if registrationRef.Get() == nil && !serviceWorkerStartedRef.Get() {
			serviceWorkerStartedRef.Set(true)
			go func() {
				registration, err := pwa.RegisterServiceWorker(context.Background(), pwa.ServiceWorkerOptions{
					URL:   "/97-pwa-multi-client/sw.js",
					Scope: "/97-pwa-multi-client/",
				})
				if err != nil {
					serviceWorkerStatus.Set(describeMultiClientPWAError("Service worker registration failed", err))
					return
				}
				registrationRef.Set(&registration)
				serviceWorkerStatus.Set("Service worker registered for the multi-client PWA shell.")
			}()
		}
		return nil
	}, "multi-client-pwa-service-worker")

	ui.UseEffect(func() func() {
		channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: multiClientPWAChannelName})
		if err != nil {
			transportStatus.Set(describeMultiClientPWAError("Cross-tab channel unavailable", err))
			return nil
		}
		channelRef.Set(channel)
		transportStatus.Set(channel.Transport())

		subscription, err := interop.SubscribeClientMessages(channel, func(message interop.ClientMessage, receiveErr error) {
			if receiveErr != nil {
				lastReceived.Set(describeMultiClientPWAError("Peer message failed", receiveErr))
				return
			}
			if message.Source.ID == self.Get().ID {
				return
			}
			switch message.Kind {
			case interop.ClientHello:
				upsertPeer(message.Source, "ready", "hello received")
				lastReceived.Set(fmt.Sprintf("Peer hello from %s over %s.", message.Source.ID, channel.Transport()))
			case interop.ClientEvent:
				if message.Topic != multiClientPWASyncTopic {
					return
				}
				upsertPeer(message.Source, "event", formatMultiClientPWAPayload(message.Payload))
				lastReceived.Set(fmt.Sprintf("Received sync event from %s: %s", message.Source.ID, formatMultiClientPWAPayload(message.Payload)))
			case interop.ClientInvalidate:
				if message.Topic != multiClientPWAInvalidateTopic {
					return
				}
				upsertPeer(message.Source, "invalidate", message.Revision)
				lastReceived.Set(fmt.Sprintf("Received cache invalidation for %s rev %s.", message.Topic, message.Revision))
			case interop.ClientGoodbye:
				next := cloneMultiClientPWAPeers(peers.Get())
				entry := next[message.Source.ID]
				entry.Identity = message.Source
				entry.State = "disconnected"
				entry.Summary = "goodbye received"
				entry.LastSeen = time.Now().UTC()
				next[message.Source.ID] = entry
				peers.Set(next)
				lastReceived.Set(fmt.Sprintf("Peer goodbye from %s.", message.Source.ID))
			}
		})
		if err != nil {
			transportStatus.Set(describeMultiClientPWAError("Cross-tab subscription failed", err))
			_ = channel.Close()
			channelRef.Set(interop.CrossTabChannel{})
			return nil
		}
		channelCancelRef.Set(subscription.Cancel)

		if err := interop.PublishClientHello(channel, self.Get()); err != nil {
			lastSent.Set(describeMultiClientPWAError("Initial hello failed", err))
		} else {
			lastSent.Set("Published initial multi-client hello.")
		}

		return func() {
			if cancel := channelCancelRef.Get(); cancel != nil {
				cancel()
				channelCancelRef.Set(nil)
			}
			_ = interop.PublishClientGoodbye(channel, self.Get())
			_ = channel.Close()
			channelRef.Set(interop.CrossTabChannel{})
		}
	}, true)

	announcePeer := ui.UseEvent(func() {
		channel := channelRef.Get()
		if channel.Name() == "" {
			lastSent.Set("Cross-tab channel is not ready yet.")
			return
		}
		if err := interop.PublishClientHello(channel, self.Get()); err != nil {
			lastSent.Set(describeMultiClientPWAError("Peer hello failed", err))
			return
		}
		lastSent.Set("Re-announced peer hello to the multi-client channel.")
	})

	broadcastSyncEvent := ui.UseEvent(func() {
		channel := channelRef.Get()
		if channel.Name() == "" {
			lastSent.Set("Cross-tab channel is not ready yet.")
			return
		}
		payload := map[string]string{
			"status":    "shell-ready",
			"transport": transportStatus.Get(),
			"sender":    self.Get().ID,
		}
		if err := interop.PublishClientEvent(channel, multiClientPWASyncTopic, self.Get(), payload); err != nil {
			lastSent.Set(describeMultiClientPWAError("Sync event failed", err))
			return
		}
		lastSent.Set("Broadcast sync event across the inter-wasm peer channel.")
	})

	broadcastCacheInvalidation := ui.UseEvent(func() {
		channel := channelRef.Get()
		if channel.Name() == "" {
			lastSent.Set("Cross-tab channel is not ready yet.")
			return
		}
		if err := interop.PublishClientInvalidation(channel, multiClientPWAInvalidateTopic, self.Get(), cachePlan.ManifestRevision); err != nil {
			lastSent.Set(describeMultiClientPWAError("Cache invalidation failed", err))
			return
		}
		lastSent.Set(fmt.Sprintf("Broadcast cache invalidation rev %s.", cachePlan.ManifestRevision))
	})

	warmOfflineShell := ui.UseEvent(func() {
		manager := cacheManagerRef.Get()
		if manager == nil {
			cacheStatus.Set("Cache Storage manager is not ready yet.")
			return
		}
		cacheStatus.Set("Warming multi-client offline shell...")
		go func() {
			snapshot, err := manager.Sync(context.Background(), cachePlan)
			if err != nil {
				cacheStatus.Set(describeMultiClientPWAError("Shell warmup failed", err))
				return
			}
			cacheStatus.Set(fmt.Sprintf("Cached %d entries into %s.", snapshot.EntryCount, snapshot.CacheName))
		}()
	})

	inspectDiagnostics := ui.UseEvent(func() {
		options := pwa.DiagnosticsOptions{Manifest: &manifest}
		if manager := cacheManagerRef.Get(); manager != nil {
			options.CacheStorage = manager
			options.CacheStoragePlan = &cachePlan
		}
		if registration := registrationRef.Get(); registration != nil {
			options.ServiceWorker = registration
		}
		diagnosticsPreview.Set("Capturing diagnostics snapshot...")
		go func() {
			snapshot, err := pwa.InspectDiagnostics(context.Background(), options)
			if err != nil {
				diagnosticsPreview.Set(describeMultiClientPWAError("Diagnostics snapshot failed", err))
				return
			}
			lines := []string{
				fmt.Sprintf("manifest valid: %t", snapshot.Manifest.Valid),
				fmt.Sprintf("cache entries: %d", snapshot.CacheStorage.EntryCount),
				fmt.Sprintf("storage pressure: %s", snapshot.Storage.Pressure),
			}
			if snapshot.ServiceWorker.Scope != "" {
				lines = append(lines, "service worker scope: "+snapshot.ServiceWorker.Scope)
			}
			diagnosticsPreview.Set(strings.Join(lines, "\n"))
			diagnosticsSummary.Set(multiClientPWASummary(snapshot, len(peers.Get())))
		}()
	})

	return shared.ExamplePage(
		"PWA multi-client coordination",
		"pwa.RegisterServiceWorker",
		"Run two wasm tabs under one PWA-shaped shell, exchange multi-client peer messages, and inspect the same service-worker and Cache Storage state that coordinates the offline shell.",
		shared.ExamplePanel("PWA and peer controls",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example combines the multi-client cross-tab message helpers with explicit PWA ownership. Each tab is its own wasm client, but they still coordinate shell revision and peer state through a typed browser-local channel.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Announce peer", announcePeer),
				shared.ExampleButton("Broadcast sync event", broadcastSyncEvent),
				shared.ExampleButton("Broadcast cache invalidation", broadcastCacheInvalidation),
				shared.ExampleButton("Warm offline shell", warmOfflineShell),
				shared.ExampleButton("Inspect diagnostics", inspectDiagnostics),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-3 md:grid-cols-3"},
				shared.ExampleStat("Transport", transportStatus.Get()),
				shared.ExampleStat("Peer count", fmt.Sprintf("%d", len(peers.Get()))),
				shared.ExampleStat("Shell revision", cachePlan.ManifestRevision),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-300", ID: "multi-client-pwa-sw-status"}, html.Text(serviceWorkerStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-cache-status"}, html.Text(cacheStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-last-sent"}, html.Text(lastSent.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-last-received"}, html.Text(lastReceived.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "multi-client-pwa-diagnostics-summary"}, html.Text(diagnosticsSummary.Get())),
		),
		shared.ExamplePanel("Discovered peers",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Open a second tab of this same page. Each tab publishes a multi-client hello and receives typed event or invalidation messages from the other wasm runtime.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3", ID: "multi-client-pwa-peer-list"}, renderMultiClientPWAPeers(peers.Get())...),
		),
		shared.ExamplePanel("Structured diagnostics output",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The diagnostics helper stays separate from peer transport. The page combines them at the app layer so PWA shell state and inter-wasm coordination remain reviewable instead of implicit.")),
			html.Pre(html.Props{Class: "mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "multi-client-pwa-diagnostics-preview"}, html.Text(diagnosticsPreview.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(multiClientPWARoot), "#app")
	select {}
}
