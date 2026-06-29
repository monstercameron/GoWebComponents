//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
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
	parseTheme := ui.UseState("light")
	parseSession := ui.UseState("signed-in")
	cacheStatus := ui.UseState("products:list is fresh.")
	cacheRevision := ui.UseState(1)
	parseDraft := ui.UseState("Open this page in a second tab, then broadcast a new draft.")
	parseDraftVersion := ui.UseState(1)
	parseThemeTransport := ui.UseState("pending")
	parseAuthTransport := ui.UseState("pending")
	cacheTransport := ui.UseState("pending")
	parseDraftTransport := ui.UseState("pending")
	parseLogs := ui.UseState([]string{"Open this example in two tabs to watch cross-tab updates arrive."})

	parseAppendLog := func(parseLine string) {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			return
		}
		parseLogs.Update(func(parsePrevious []string) []string {
			parseNext := append([]string{parseTrimmed}, parsePrevious...)
			if len(parseNext) > 8 {
				parseNext = parseNext[:8]
			}
			return parseNext
		})
	}

	parseDescribeError := func(parsePrefix string, parseErr8 error) string {
		if parseErr8 == nil {
			return parsePrefix
		}
		if parseCode, parseOk := interop.CodeOf(parseErr8); parseOk {
			return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr8)
		}
		return fmt.Sprintf("%s: %v", parsePrefix, parseErr8)
	}

	parsePublish := func(parseName string, parsePayload any, parseAfter func(interop.CrossTabChannel)) {
		parseChannel, parseErr := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: parseName})
		if parseErr != nil {
			parseAppendLog(parseDescribeError("Cross-tab channel unavailable", parseErr))
			return
		}
		defer parseChannel.Close()
		if parseErr2 := parseChannel.Publish(parsePayload); parseErr2 != nil {
			parseAppendLog(parseDescribeError("Cross-tab publish failed", parseErr2))
			return
		}
		if parseAfter != nil {
			parseAfter(parseChannel)
		}
	}

	setDraft := ui.UseEvent(func(parseE ui.Event) {
		parseDraft.Set(parseE.GetValue())
	})
	parseBroadcastLight := ui.UseEvent(func() {
		parsePublish(themeChannelName, themeSignal{Theme: "light"}, func(parseChannel2 interop.CrossTabChannel) {
			parseTheme.Set("light")
			parseThemeTransport.Set(parseChannel2.Transport())
			parseAppendLog("Sent theme update: light via " + parseChannel2.Transport())
		})
	})
	parseBroadcastDark := ui.UseEvent(func() {
		parsePublish(themeChannelName, themeSignal{Theme: "dark"}, func(parseChannel3 interop.CrossTabChannel) {
			parseTheme.Set("dark")
			parseThemeTransport.Set(parseChannel3.Transport())
			parseAppendLog("Sent theme update: dark via " + parseChannel3.Transport())
		})
	})
	parseBroadcastLogout := ui.UseEvent(func() {
		parsePublish(authChannelName, authSignal{Status: "signed-out", Reason: "Operator signed out in another tab."}, func(parseChannel4 interop.CrossTabChannel) {
			parseSession.Set("signed-out")
			parseAuthTransport.Set(parseChannel4.Transport())
			parseAppendLog("Broadcast a signed-out auth hint via " + parseChannel4.Transport())
		})
	})
	parseInvalidateCache := ui.UseEvent(func() {
		parseNextRevision := cacheRevision.Get() + 1
		parsePublish(cacheChannelName, cacheSignal{Key: "products:list", Revision: parseNextRevision}, func(parseChannel5 interop.CrossTabChannel) {
			cacheRevision.Set(parseNextRevision)
			cacheStatus.Set(fmt.Sprintf("Published invalidation for products:list rev %d.", parseNextRevision))
			cacheTransport.Set(parseChannel5.Transport())
			parseAppendLog(fmt.Sprintf("Broadcast cache invalidation rev %d via %s", parseNextRevision, parseChannel5.Transport()))
		})
	})
	parseBroadcastDraft := ui.UseEvent(func() {
		parseNextVersion := parseDraftVersion.Get() + 1
		parsePublish(draftChannelName, draftSignal{
			Draft:   parseDraft.Get(),
			Version: parseNextVersion,
			Editor:  "local tab",
		}, func(parseChannel6 interop.CrossTabChannel) {
			parseDraftVersion.Set(parseNextVersion)
			parseDraftTransport.Set(parseChannel6.Transport())
			parseAppendLog(fmt.Sprintf("Broadcast draft version %d via %s", parseNextVersion, parseChannel6.Transport()))
		})
	})

	ui.UseEffect(func() func() {
		type channelBinding struct {
			channel interop.CrossTabChannel
			cancel  func()
		}
		parseBindings := make([]channelBinding, 0, 4)

		parseOpenTheme, parseErr3 := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: themeChannelName})
		if parseErr3 != nil {
			parseAppendLog(parseDescribeError("Theme channel unavailable", parseErr3))
		} else {
			parseThemeTransport.Set(parseOpenTheme.Transport())
			parseSubscription, parseErr4 := interop.SubscribeDecodedCrossTab[themeSignal](parseOpenTheme, func(parseMessage interop.DecodedCrossTabEnvelope[themeSignal], parseErr9 error) {
				if parseErr9 != nil {
					parseAppendLog(parseDescribeError("Theme sync failed", parseErr9))
					return
				}
				parseTheme.Set(strings.TrimSpace(parseMessage.Payload.Theme))
				parseAppendLog(fmt.Sprintf("Received theme %q from %s.", parseMessage.Payload.Theme, parseMessage.Source))
			})
			if parseErr4 != nil {
				parseAppendLog(parseDescribeError("Theme subscription failed", parseErr4))
				parseOpenTheme.Close()
			} else {
				parseBindings = append(parseBindings, channelBinding{channel: parseOpenTheme, cancel: parseSubscription.Cancel})
			}
		}

		parseOpenAuth, parseErr3 := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: authChannelName})
		if parseErr3 != nil {
			parseAppendLog(parseDescribeError("Auth channel unavailable", parseErr3))
		} else {
			parseAuthTransport.Set(parseOpenAuth.Transport())
			parseSubscription2, parseErr5 := interop.SubscribeDecodedCrossTab[authSignal](parseOpenAuth, func(parseMessage2 interop.DecodedCrossTabEnvelope[authSignal], parseErr10 error) {
				if parseErr10 != nil {
					parseAppendLog(parseDescribeError("Auth sync failed", parseErr10))
					return
				}
				parseSession.Set(parseMessage2.Payload.Status)
				parseAppendLog(fmt.Sprintf("Received auth signal %q: %s", parseMessage2.Payload.Status, parseMessage2.Payload.Reason))
			})
			if parseErr5 != nil {
				parseAppendLog(parseDescribeError("Auth subscription failed", parseErr5))
				parseOpenAuth.Close()
			} else {
				parseBindings = append(parseBindings, channelBinding{channel: parseOpenAuth, cancel: parseSubscription2.Cancel})
			}
		}

		parseOpenCache, parseErr3 := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: cacheChannelName})
		if parseErr3 != nil {
			parseAppendLog(parseDescribeError("Cache channel unavailable", parseErr3))
		} else {
			cacheTransport.Set(parseOpenCache.Transport())
			parseSubscription3, parseErr6 := interop.SubscribeDecodedCrossTab[cacheSignal](parseOpenCache, func(parseMessage3 interop.DecodedCrossTabEnvelope[cacheSignal], parseErr11 error) {
				if parseErr11 != nil {
					parseAppendLog(parseDescribeError("Cache sync failed", parseErr11))
					return
				}
				cacheRevision.Set(parseMessage3.Payload.Revision)
				cacheStatus.Set(fmt.Sprintf("Received invalidation for %s rev %d. Revalidate authoritative data now.", parseMessage3.Payload.Key, parseMessage3.Payload.Revision))
				parseAppendLog(fmt.Sprintf("Received cache invalidation for %s rev %d.", parseMessage3.Payload.Key, parseMessage3.Payload.Revision))
			})
			if parseErr6 != nil {
				parseAppendLog(parseDescribeError("Cache subscription failed", parseErr6))
				parseOpenCache.Close()
			} else {
				parseBindings = append(parseBindings, channelBinding{channel: parseOpenCache, cancel: parseSubscription3.Cancel})
			}
		}

		parseOpenDraft, parseErr3 := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: draftChannelName})
		if parseErr3 != nil {
			parseAppendLog(parseDescribeError("Draft channel unavailable", parseErr3))
		} else {
			parseDraftTransport.Set(parseOpenDraft.Transport())
			parseSubscription4, parseErr7 := interop.SubscribeDecodedCrossTab[draftSignal](parseOpenDraft, func(parseMessage4 interop.DecodedCrossTabEnvelope[draftSignal], parseErr12 error) {
				if parseErr12 != nil {
					parseAppendLog(parseDescribeError("Draft sync failed", parseErr12))
					return
				}
				if parseMessage4.Payload.Version >= parseDraftVersion.Get() {
					parseDraft.Set(parseMessage4.Payload.Draft)
					parseDraftVersion.Set(parseMessage4.Payload.Version)
				}
				parseAppendLog(fmt.Sprintf("Received draft version %d from %s.", parseMessage4.Payload.Version, parseMessage4.Payload.Editor))
			})
			if parseErr7 != nil {
				parseAppendLog(parseDescribeError("Draft subscription failed", parseErr7))
				parseOpenDraft.Close()
			} else {
				parseBindings = append(parseBindings, channelBinding{channel: parseOpenDraft, cancel: parseSubscription4.Cancel})
			}
		}

		return func() {
			for parseI := len(parseBindings) - 1; parseI >= 0; parseI-- {
				if parseBindings[parseI].cancel != nil {
					parseBindings[parseI].cancel()
				}
				_ = parseBindings[parseI].channel.Close()
			}
		}
	}, true)

	parseLogNodes := make([]ui.Node, 0, len(parseLogs.Get()))
	for _, parseEntry := range parseLogs.Get() {
		parseLogNodes = append(parseLogNodes,
			html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm leading-7 text-slate-300"}, html.Text(parseEntry)),
		)
	}

	return shared.ExamplePage(
		"Cross-Tab Sync",
		"interop.OpenCrossTabChannel",
		"Open this example in two tabs to broadcast theme changes, logout hints, cache invalidations, and draft updates through the public cross-tab channel helpers.",
		shared.ExamplePanel("Cross-tab state hints",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Each concern uses its own named channel instead of one global bus. The sender updates its own local state immediately, while the second tab receives the typed signal through SubscribeDecodedCrossTab(...).")),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Theme", parseTheme.Get()),
				shared.ExampleStat("Session", parseSession.Get()),
				shared.ExampleStat("Cache rev", fmt.Sprintf("%d", cacheRevision.Get())),
				shared.ExampleStat("Draft version", fmt.Sprintf("%d", parseDraftVersion.Get())),
			),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Theme: light", parseBroadcastLight),
				shared.ExampleButton("Theme: dark", parseBroadcastDark),
				shared.ExampleButton("Broadcast logout", parseBroadcastLogout),
				shared.ExampleButton("Invalidate cache", parseInvalidateCache),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(cacheStatus.Get())),
		),
		shared.ExamplePanel("Draft sync",
			html.Label(html.Props{For: "cross-tab-draft", Class: "text-sm font-semibold text-slate-200"}, html.Text("Shared draft")),
			html.Textarea(html.Props{
				ID:          "cross-tab-draft",
				Rows:        7,
				Value:       parseDraft.Get(),
				OnInput:     setDraft,
				Placeholder: "Edit this draft in one tab, then broadcast it to the other.",
				Class:       "mt-2 w-full rounded-[1.35rem] border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
			}),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Broadcast draft", parseBroadcastDraft),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Draft updates carry a version field so receivers can ignore obviously older state after their own restore or local edits.")),
		),
		shared.ExamplePanel("Diagnostics",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Theme transport", parseThemeTransport.Get()),
				shared.ExampleStat("Auth transport", parseAuthTransport.Get()),
				shared.ExampleStat("Cache transport", cacheTransport.Get()),
				shared.ExampleStat("Draft transport", parseDraftTransport.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The diagnostics log shows what arrived from another tab and which transport the browser resolved. BroadcastChannel is preferred, and localStorage storage events are the fallback.")),
			html.Ul(html.Props{Class: "mt-5 grid gap-3"}, parseLogNodes...),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(crossTabExample))
	exampleboot.WaitExampleRuntime()
}
