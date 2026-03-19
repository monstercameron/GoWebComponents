//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	themeChannelName = "example:cross-tab:theme"
	authChannelName  = "example:cross-tab:auth"
	cacheChannelName = "example:cross-tab:cache"
	draftChannelName = "example:cross-tab:draft"
)

type themeSignal struct {
	Theme string `json:"theme"`
}

type authSignal struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type cacheSignal struct {
	Key      string `json:"key"`
	Revision int    `json:"revision"`
}

type draftSignal struct {
	Draft   string `json:"draft"`
	Version int    `json:"version"`
	Editor  string `json:"editor"`
}

func crossTabExample() ui.Node {
	theme := ui.UseState("light")
	session := ui.UseState("signed-in")
	cacheStatus := ui.UseState("products:list is fresh.")
	cacheRevision := ui.UseState(1)
	draft := ui.UseState("Open this page in a second tab, then broadcast a new draft.")
	draftVersion := ui.UseState(1)
	themeTransport := ui.UseState("pending")
	authTransport := ui.UseState("pending")
	cacheTransport := ui.UseState("pending")
	draftTransport := ui.UseState("pending")
	logs := ui.UseState([]string{"Open this example in two tabs to watch cross-tab updates arrive."})

	appendLog := func(line string) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return
		}
		logs.Update(func(previous []string) []string {
			next := append([]string{trimmed}, previous...)
			if len(next) > 8 {
				next = next[:8]
			}
			return next
		})
	}

	describeError := func(prefix string, err error) string {
		if err == nil {
			return prefix
		}
		if code, ok := interop.CodeOf(err); ok {
			return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
		}
		return fmt.Sprintf("%s: %v", prefix, err)
	}

	publish := func(name string, payload any, after func(interop.CrossTabChannel)) {
		channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: name})
		if err != nil {
			appendLog(describeError("Cross-tab channel unavailable", err))
			return
		}
		defer channel.Close()
		if err := channel.Publish(payload); err != nil {
			appendLog(describeError("Cross-tab publish failed", err))
			return
		}
		if after != nil {
			after(channel)
		}
	}

	setDraft := ui.UseEvent(func(e ui.Event) {
		draft.Set(e.GetValue())
	})
	broadcastLight := ui.UseEvent(func() {
		publish(themeChannelName, themeSignal{Theme: "light"}, func(channel interop.CrossTabChannel) {
			theme.Set("light")
			themeTransport.Set(channel.Transport())
			appendLog("Sent theme update: light via " + channel.Transport())
		})
	})
	broadcastDark := ui.UseEvent(func() {
		publish(themeChannelName, themeSignal{Theme: "dark"}, func(channel interop.CrossTabChannel) {
			theme.Set("dark")
			themeTransport.Set(channel.Transport())
			appendLog("Sent theme update: dark via " + channel.Transport())
		})
	})
	broadcastLogout := ui.UseEvent(func() {
		publish(authChannelName, authSignal{Status: "signed-out", Reason: "Operator signed out in another tab."}, func(channel interop.CrossTabChannel) {
			session.Set("signed-out")
			authTransport.Set(channel.Transport())
			appendLog("Broadcast a signed-out auth hint via " + channel.Transport())
		})
	})
	invalidateCache := ui.UseEvent(func() {
		nextRevision := cacheRevision.Get() + 1
		publish(cacheChannelName, cacheSignal{Key: "products:list", Revision: nextRevision}, func(channel interop.CrossTabChannel) {
			cacheRevision.Set(nextRevision)
			cacheStatus.Set(fmt.Sprintf("Published invalidation for products:list rev %d.", nextRevision))
			cacheTransport.Set(channel.Transport())
			appendLog(fmt.Sprintf("Broadcast cache invalidation rev %d via %s", nextRevision, channel.Transport()))
		})
	})
	broadcastDraft := ui.UseEvent(func() {
		nextVersion := draftVersion.Get() + 1
		publish(draftChannelName, draftSignal{
			Draft:   draft.Get(),
			Version: nextVersion,
			Editor:  "local tab",
		}, func(channel interop.CrossTabChannel) {
			draftVersion.Set(nextVersion)
			draftTransport.Set(channel.Transport())
			appendLog(fmt.Sprintf("Broadcast draft version %d via %s", nextVersion, channel.Transport()))
		})
	})

	ui.UseEffect(func() func() {
		type channelBinding struct {
			channel interop.CrossTabChannel
			cancel  func()
		}
		bindings := make([]channelBinding, 0, 4)

		openTheme, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: themeChannelName})
		if err != nil {
			appendLog(describeError("Theme channel unavailable", err))
		} else {
			themeTransport.Set(openTheme.Transport())
			subscription, err := interop.SubscribeDecodedCrossTab[themeSignal](openTheme, func(message interop.DecodedCrossTabEnvelope[themeSignal], err error) {
				if err != nil {
					appendLog(describeError("Theme sync failed", err))
					return
				}
				theme.Set(strings.TrimSpace(message.Payload.Theme))
				appendLog(fmt.Sprintf("Received theme %q from %s.", message.Payload.Theme, message.Source))
			})
			if err != nil {
				appendLog(describeError("Theme subscription failed", err))
				openTheme.Close()
			} else {
				bindings = append(bindings, channelBinding{channel: openTheme, cancel: subscription.Cancel})
			}
		}

		openAuth, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: authChannelName})
		if err != nil {
			appendLog(describeError("Auth channel unavailable", err))
		} else {
			authTransport.Set(openAuth.Transport())
			subscription, err := interop.SubscribeDecodedCrossTab[authSignal](openAuth, func(message interop.DecodedCrossTabEnvelope[authSignal], err error) {
				if err != nil {
					appendLog(describeError("Auth sync failed", err))
					return
				}
				session.Set(message.Payload.Status)
				appendLog(fmt.Sprintf("Received auth signal %q: %s", message.Payload.Status, message.Payload.Reason))
			})
			if err != nil {
				appendLog(describeError("Auth subscription failed", err))
				openAuth.Close()
			} else {
				bindings = append(bindings, channelBinding{channel: openAuth, cancel: subscription.Cancel})
			}
		}

		openCache, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: cacheChannelName})
		if err != nil {
			appendLog(describeError("Cache channel unavailable", err))
		} else {
			cacheTransport.Set(openCache.Transport())
			subscription, err := interop.SubscribeDecodedCrossTab[cacheSignal](openCache, func(message interop.DecodedCrossTabEnvelope[cacheSignal], err error) {
				if err != nil {
					appendLog(describeError("Cache sync failed", err))
					return
				}
				cacheRevision.Set(message.Payload.Revision)
				cacheStatus.Set(fmt.Sprintf("Received invalidation for %s rev %d. Revalidate authoritative data now.", message.Payload.Key, message.Payload.Revision))
				appendLog(fmt.Sprintf("Received cache invalidation for %s rev %d.", message.Payload.Key, message.Payload.Revision))
			})
			if err != nil {
				appendLog(describeError("Cache subscription failed", err))
				openCache.Close()
			} else {
				bindings = append(bindings, channelBinding{channel: openCache, cancel: subscription.Cancel})
			}
		}

		openDraft, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: draftChannelName})
		if err != nil {
			appendLog(describeError("Draft channel unavailable", err))
		} else {
			draftTransport.Set(openDraft.Transport())
			subscription, err := interop.SubscribeDecodedCrossTab[draftSignal](openDraft, func(message interop.DecodedCrossTabEnvelope[draftSignal], err error) {
				if err != nil {
					appendLog(describeError("Draft sync failed", err))
					return
				}
				if message.Payload.Version >= draftVersion.Get() {
					draft.Set(message.Payload.Draft)
					draftVersion.Set(message.Payload.Version)
				}
				appendLog(fmt.Sprintf("Received draft version %d from %s.", message.Payload.Version, message.Payload.Editor))
			})
			if err != nil {
				appendLog(describeError("Draft subscription failed", err))
				openDraft.Close()
			} else {
				bindings = append(bindings, channelBinding{channel: openDraft, cancel: subscription.Cancel})
			}
		}

		return func() {
			for i := len(bindings) - 1; i >= 0; i-- {
				if bindings[i].cancel != nil {
					bindings[i].cancel()
				}
				_ = bindings[i].channel.Close()
			}
		}
	}, true)

	logNodes := make([]ui.Node, 0, len(logs.Get()))
	for _, entry := range logs.Get() {
		logNodes = append(logNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(entry)),
		)
	}

	return shared.ExamplePage(
		"Cross-Tab Sync",
		"interop.OpenCrossTabChannel",
		"Open this example in two tabs to broadcast theme changes, logout hints, cache invalidations, and draft updates through the public cross-tab channel helpers.",
		shared.ExamplePanel("Cross-tab state hints",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Each concern uses its own named channel instead of one global bus. The sender updates its own local state immediately, while the second tab receives the typed signal through SubscribeDecodedCrossTab(...).")),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Theme", theme.Get()),
				shared.ExampleStat("Session", session.Get()),
				shared.ExampleStat("Cache rev", fmt.Sprintf("%d", cacheRevision.Get())),
				shared.ExampleStat("Draft version", fmt.Sprintf("%d", draftVersion.Get())),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Theme: light", broadcastLight),
				shared.ExampleButton("Theme: dark", broadcastDark),
				shared.ExampleButton("Broadcast logout", broadcastLogout),
				shared.ExampleButton("Invalidate cache", invalidateCache),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(cacheStatus.Get())),
		),
		shared.ExamplePanel("Draft sync",
			html.Label(html.Props{For: "cross-tab-draft", Class: "text-sm font-semibold text-slate-200"}, html.Text("Shared draft")),
			html.Textarea(html.Props{
				ID:          "cross-tab-draft",
				Rows:        7,
				Value:       draft.Get(),
				OnInput:     setDraft,
				Placeholder: "Edit this draft in one tab, then broadcast it to the other.",
				Class:       "mt-2 w-full rounded-[1.35rem] border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
			}),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Broadcast draft", broadcastDraft),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Draft updates carry a version field so receivers can ignore obviously older state after their own restore or local edits.")),
		),
		shared.ExamplePanel("Diagnostics",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Theme transport", themeTransport.Get()),
				shared.ExampleStat("Auth transport", authTransport.Get()),
				shared.ExampleStat("Cache transport", cacheTransport.Get()),
				shared.ExampleStat("Draft transport", draftTransport.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The diagnostics log shows what arrived from another tab and which transport the browser resolved. BroadcastChannel is preferred, and localStorage storage events are the fallback.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, logNodes...),
		),
		shared.ExamplePanel("Integration shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Use one named channel per concern, restore local bootstrap or persisted state first, then open the channel and ignore incoming messages that are older than the local revision when conflict risk is real.")),
			shared.ExampleCode(
				`channel, _ := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: "draft:checkout"})`,
				`sub, _ := interop.SubscribeDecodedCrossTab[draftSignal](channel, func(message interop.DecodedCrossTabEnvelope[draftSignal], err error) { ... })`,
				`_ = channel.Publish(draftSignal{Draft: draft, Version: nextVersion})`,
				`defer sub.Cancel(); defer channel.Close()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(crossTabExample), "#app")
	select {}
}
