//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

type demoPulse struct {
	Message string `json:"message"`
	Source  string `json:"source"`
	Count   int    `json:"count"`
}

const browserInteropLazyModuleSpecifier = "/static/modules/browser-interop-lazy-module.js"

func browserInteropExample() ui.Node {
	parseDraft := ui.UseState("Ship the browser bridge.")
	parseSavedDraft := ui.UseState("No stored draft yet.")
	parseSavedScheme := ui.UseState("unset")
	parseStorageStatus := ui.UseState("Storage has not been read yet.")
	parseClipboardStatus := ui.UseState("Clipboard idle.")
	parseColorScheme := ui.UseState("unknown")
	parseHostLookup := ui.UseState("Waiting for document lookup.")
	parsePanelSize := ui.UseState("Waiting for resize observer.")
	parseEventStatus := ui.UseState("Waiting for interop-demo events.")
	parseEventCount := ui.UseState(0)
	parseEventSource := ui.UseState("none")
	parseModuleStatus := ui.UseState("Lazy module has not been imported yet.")
	parseModuleResult := ui.UseState("No module exports have been read yet.")
	parseModuleLoadCount := ui.UseState(0)

	parseDescribeError := func(parsePrefix string, parseErr18 error) string {
		if parseErr18 == nil {
			return parsePrefix
		}
		if parseCode, parseOk := interop.CodeOf(parseErr18); parseOk {
			return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr18)
		}
		return fmt.Sprintf("%s: %v", parsePrefix, parseErr18)
	}

	parseLoadSnapshot := func(parseStorage4 interop.Storage) {
		parseValues, parseErr := parseStorage4.GetMany("interop-demo:draft", "interop-demo:scheme")
		if parseErr != nil {
			parseStorageStatus.Set(parseDescribeError("Storage read failed", parseErr))
			return
		}
		if parseValue, parseOk2 := parseValues["interop-demo:draft"]; parseOk2 {
			parseSavedDraft.Set(parseValue)
		} else {
			parseSavedDraft.Set("No stored draft yet.")
		}
		if parseValue2, parseOk3 := parseValues["interop-demo:scheme"]; parseOk3 {
			parseSavedScheme.Set(parseValue2)
		} else {
			parseSavedScheme.Set("unset")
		}
		parseStorageStatus.Set(fmt.Sprintf("Loaded %d keys through storage.GetMany(...).", len(parseValues)))
	}

	parseUpdateDraft := ui.UseEvent(func(parseE ui.Event) {
		parseDraft.Set(parseE.GetValue())
	})

	parseLoadLazyModule := ui.UseEvent(func() {
		parseModuleStatus.Set("Lazy module import requested.")
		parseModuleResult.Set("Waiting for the module exports.")
		parseModuleLoadCount.Update(func(parsePrevious int) int { return parsePrevious + 1 })
	})

	ui.UseEffect(func() func() {
		if parseModuleLoadCount.Get() == 0 {
			return nil
		}

		parseModuleStatus.Set("Importing a lazy module through interop.ImportModule(...).")

		parseModule, parseErr2 := interop.ImportModule(context.Background(), browserInteropLazyModuleSpecifier)
		if parseErr2 != nil {
			parseModuleStatus.Set(parseDescribeError("Lazy module import failed", parseErr2))
			return nil
		}
		defer parseModule.Dispose()

		parseName, parseErr2 := parseModule.Value(context.Background(), "bundleName")
		if parseErr2 != nil {
			parseModuleStatus.Set(parseDescribeError("Lazy module bundle name lookup failed", parseErr2))
			return nil
		}

		parseHelperLabel, parseErr2 := parseModule.Call(context.Background(), "formatLabel", parseDraft.Get())
		if parseErr2 != nil {
			parseModuleStatus.Set(parseDescribeError("Lazy module helper call failed", parseErr2))
			return nil
		}

		parseDefaultLabel, parseErr2 := parseModule.CallDefault(context.Background(), parseDraft.Get())
		if parseErr2 != nil {
			parseModuleStatus.Set(parseDescribeError("Lazy module default export failed", parseErr2))
			return nil
		}

		parseModuleStatus.Set(fmt.Sprintf("Imported %s through interop.ImportModule(...).", parseName))
		parseModuleResult.Set(fmt.Sprintf("%s | %s", parseHelperLabel, parseDefaultLabel))
		return nil
	}, parseModuleLoadCount.Get())

	parseSaveStorage := ui.UseEvent(func() {
		parseStorage, parseErr3 := interop.GetLocalStorage()
		if parseErr3 != nil {
			parseStorageStatus.Set(parseDescribeError("LocalStorage unavailable", parseErr3))
			return
		}
		if parseErr4 := parseStorage.SetItem("interop-demo:draft", parseDraft.Get()); parseErr4 != nil {
			parseStorageStatus.Set(parseDescribeError("Failed to store draft", parseErr4))
			return
		}
		if parseErr5 := parseStorage.SetItem("interop-demo:scheme", parseColorScheme.Get()); parseErr5 != nil {
			parseStorageStatus.Set(parseDescribeError("Failed to store media state", parseErr5))
			return
		}
		parseLoadSnapshot(parseStorage)
	})

	parseLoadStorage := ui.UseEvent(func() {
		parseStorage2, parseErr6 := interop.GetLocalStorage()
		if parseErr6 != nil {
			parseStorageStatus.Set(parseDescribeError("LocalStorage unavailable", parseErr6))
			return
		}
		parseLoadSnapshot(parseStorage2)
	})

	parseCopyDraft := ui.UseEvent(func() {
		parseClipboard, parseErr7 := interop.GetClipboard()
		if parseErr7 != nil {
			parseClipboardStatus.Set(parseDescribeError("Clipboard unavailable", parseErr7))
			return
		}
		if parseErr8 := parseClipboard.WriteText(context.Background(), parseDraft.Get()); parseErr8 != nil {
			parseClipboardStatus.Set(parseDescribeError("Clipboard write failed", parseErr8))
			return
		}
		parseClipboardStatus.Set("Copied the current draft through GetClipboard().")
	})

	parseReadClipboard := ui.UseEvent(func() {
		parseClipboard2, parseErr9 := interop.GetClipboard()
		if parseErr9 != nil {
			parseClipboardStatus.Set(parseDescribeError("Clipboard unavailable", parseErr9))
			return
		}
		parseValue3, parseErr9 := parseClipboard2.ReadText(context.Background())
		if parseErr9 != nil {
			parseClipboardStatus.Set(parseDescribeError("Clipboard read failed", parseErr9))
			return
		}
		parseClipboardStatus.Set(fmt.Sprintf("Clipboard now contains %q.", parseValue3))
	})

	parseDispatchPulse := ui.UseEvent(func() {
		parseTarget, parseErr10 := interop.GetDocumentEvents()
		if parseErr10 != nil {
			parseEventStatus.Set(parseDescribeError("Document event target unavailable", parseErr10))
			return
		}
		parseNext := parseEventCount.Get() + 1
		if parseErr11 := parseTarget.Dispatch("interop-demo", demoPulse{
			Message: parseDraft.Get(),
			Source:  "Go button",
			Count:   parseNext,
		}); parseErr11 != nil {
			parseEventStatus.Set(parseDescribeError("Custom event dispatch failed", parseErr11))
			return
		}
	})

	ui.UseEffect(func() func() {
		parseCleanups := make([]func(), 0, 3)

		parseStorage3, parseErr12 := interop.GetLocalStorage()
		if parseErr12 != nil {
			parseStorageStatus.Set(parseDescribeError("LocalStorage unavailable", parseErr12))
		} else {
			parseLoadSnapshot(parseStorage3)
		}

		parseMedia, parseErr12 := interop.GetMediaQuery("(prefers-color-scheme: dark)")
		if parseErr12 != nil {
			parseColorScheme.Set("unavailable")
		} else {
			if parseMedia.Matches() {
				parseColorScheme.Set("dark")
			} else {
				parseColorScheme.Set("light")
			}
			parseSubscription, parseErr13 := parseMedia.Subscribe(func(parseEvent interop.MediaQueryEvent) {
				if parseEvent.Matches {
					parseColorScheme.Set("dark")
					return
				}
				parseColorScheme.Set("light")
			})
			if parseErr13 == nil {
				parseCleanups = append(parseCleanups, parseSubscription.Cancel)
			}
		}

		parseDocument, parseErr12 := interop.GetDocument()
		if parseErr12 != nil {
			parseHostLookup.Set(parseDescribeError("Document lookup unavailable", parseErr12))
		} else {
			parseElements, parseErr14 := parseDocument.ElementsByID("interop-demo-panel", "interop-demo-status")
			if parseErr14 != nil {
				parseHostLookup.Set(parseDescribeError("Grouped host lookup failed", parseErr14))
			} else {
				parseHostLookup.Set(fmt.Sprintf("Resolved %d hosts through document.ElementsByID(...).", len(parseElements)))
				if parsePanel, parseOk4 := parseElements["interop-demo-panel"]; parseOk4 {
					if parseRect, parseErr15 := parsePanel.BoundingClientRect(); parseErr15 == nil {
						parsePanelSize.Set(fmt.Sprintf("%.0f x %.0f", parseRect.Width, parseRect.Height))
					}
					parseSubscription2, parseErr16 := parsePanel.ObserveResize(func(parseEntry interop.ResizeEntry) {
						parsePanelSize.Set(fmt.Sprintf("%.0f x %.0f", parseEntry.ContentRect.Width, parseEntry.ContentRect.Height))
					})
					if parseErr16 == nil {
						parseCleanups = append(parseCleanups, parseSubscription2.Cancel)
					}
				}
			}
		}

		parseTarget2, parseErr12 := interop.GetDocumentEvents()
		if parseErr12 != nil {
			parseEventStatus.Set(parseDescribeError("Document event target unavailable", parseErr12))
		} else {
			parseSubscription3, parseErr17 := interop.SubscribeDecoded(parseTarget2, "interop-demo", func(parseEvent2 interop.DecodedCustomEvent[demoPulse], parseErr19 error) {
				if parseErr19 != nil {
					parseEventStatus.Set(parseDescribeError("Typed custom-event decode failed", parseErr19))
					return
				}
				parseEventCount.Set(parseEvent2.Detail.Count)
				parseEventSource.Set(parseEvent2.Detail.Source)
				parseEventStatus.Set(fmt.Sprintf("Received %q through interop.SubscribeDecoded(...).", parseEvent2.Detail.Message))
			})
			if parseErr17 == nil {
				parseCleanups = append(parseCleanups, parseSubscription3.Cancel)
			}
		}

		return func() {
			for parseI := len(parseCleanups) - 1; parseI >= 0; parseI-- {
				parseCleanups[parseI]()
			}
		}
	}, true)

	return shared.ExamplePage(
		"Browser Interop",
		"interop storage, clipboard, observers, and custom events",
		"Use the public interop package to reach browser APIs without scattering raw syscall/js code through app components.",
		shared.ExamplePanel("Storage and clipboard",
			html.Label(html.Props{For: "interop-draft", Class: "text-sm font-semibold text-slate-200"}, html.Text("Interop draft")),
			html.Input(html.Props{
				ID:          "interop-draft",
				Value:       parseDraft.Get(),
				OnInput:     parseUpdateDraft,
				Placeholder: "Type a browser-side draft",
				Class:       "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
			}),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Save to storage", parseSaveStorage),
				shared.ExampleButton("Load snapshot", parseLoadStorage),
				shared.ExampleButton("Copy draft", parseCopyDraft),
				shared.ExampleButton("Read clipboard", parseReadClipboard),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Saved draft", parseSavedDraft.Get()),
				shared.ExampleStat("Saved scheme", parseSavedScheme.Get()),
			),
			html.P(html.Props{ID: "interop-demo-status", Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseStorageStatus.Get())),
			html.P(html.Props{Class: "mt-2 text-sm leading-7 text-slate-300"}, html.Text(parseClipboardStatus.Get())),
		),
		shared.ExamplePanel("Observers and DOM lookup",
			html.Div(html.Props{ID: "interop-demo-panel", Class: "mt-3 rounded-[1.25rem] border border-cyan-300/20 bg-cyan-500/5 p-5"},
				html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text("This panel is resolved through document.ElementsByID(...) and measured through ObserveResize(...). Resize the browser or change the catalog width to watch the live measurement update.")),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Color scheme", parseColorScheme.Get()),
				shared.ExampleStat("Panel size", parsePanelSize.Get()),
				shared.ExampleStat("Host lookup", parseHostLookup.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The example subscribes to prefers-color-scheme through MatchMedia(...) and keeps browser handles local to the effect that owns them.")),
		),
		shared.ExamplePanel("Typed custom events",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Dispatch a document-level CustomEvent through interop.DocumentEvents() and decode it back into a typed Go payload through interop.SubscribeDecoded(...).")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Dispatch pulse", parseDispatchPulse),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Event count", fmt.Sprintf("%d", parseEventCount.Get())),
				shared.ExampleStat("Event source", parseEventSource.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseEventStatus.Get())),
			shared.ExampleCode(
				`storage.GetMany("interop-demo:draft", "interop-demo:scheme")`,
				`clipboard.WriteText(context.Background(), draft)`,
				`document.ElementsByID("interop-demo-panel", "interop-demo-status")`,
				`interop.SubscribeDecoded[demoPulse](target, "interop-demo", handler)`,
			),
		),
		shared.ExamplePanel("Lazy module loading",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Import a JavaScript helper only when the user asks for it, then call its named and default exports through interop.ImportModule(...).")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Load lazy module", parseLoadLazyModule),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Module status", parseModuleStatus.Get()),
				shared.ExampleStat("Module result", parseModuleResult.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The loader resolves the module specifier through the browser bridge, reads a named export, calls the default export, and disposes the handle once the work is done.")),
			shared.ExampleCode(
				`interop.ImportModule(context.Background(), "/static/modules/browser-interop-lazy-module.js")`,
				`module.Value(context.Background(), "bundleName")`,
				`module.Call(context.Background(), "formatLabel", draft)`,
				`module.Dispose()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(browserInteropExample))
	exampleboot.WaitExampleRuntime()
}
