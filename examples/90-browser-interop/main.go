//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type demoPulse struct {
	Message string `json:"message"`
	Source  string `json:"source"`
	Count   int    `json:"count"`
}

const browserInteropLazyModuleSpecifier = "/static/modules/browser-interop-lazy-module.js"

func browserInteropExample() ui.Node {
	draft := ui.UseState("Ship the browser bridge.")
	savedDraft := ui.UseState("No stored draft yet.")
	savedScheme := ui.UseState("unset")
	storageStatus := ui.UseState("Storage has not been read yet.")
	clipboardStatus := ui.UseState("Clipboard idle.")
	colorScheme := ui.UseState("unknown")
	hostLookup := ui.UseState("Waiting for document lookup.")
	panelSize := ui.UseState("Waiting for resize observer.")
	eventStatus := ui.UseState("Waiting for interop-demo events.")
	eventCount := ui.UseState(0)
	eventSource := ui.UseState("none")
	moduleStatus := ui.UseState("Lazy module has not been imported yet.")
	moduleResult := ui.UseState("No module exports have been read yet.")
	moduleLoadCount := ui.UseState(0)

	describeError := func(prefix string, err error) string {
		if err == nil {
			return prefix
		}
		if code, ok := interop.CodeOf(err); ok {
			return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
		}
		return fmt.Sprintf("%s: %v", prefix, err)
	}

	loadSnapshot := func(storage interop.Storage) {
		values, err := storage.GetMany("interop-demo:draft", "interop-demo:scheme")
		if err != nil {
			storageStatus.Set(describeError("Storage read failed", err))
			return
		}
		if value, ok := values["interop-demo:draft"]; ok {
			savedDraft.Set(value)
		} else {
			savedDraft.Set("No stored draft yet.")
		}
		if value, ok := values["interop-demo:scheme"]; ok {
			savedScheme.Set(value)
		} else {
			savedScheme.Set("unset")
		}
		storageStatus.Set(fmt.Sprintf("Loaded %d keys through storage.GetMany(...).", len(values)))
	}

	updateDraft := ui.UseEvent(func(e ui.Event) {
		draft.Set(e.GetValue())
	})

	loadLazyModule := ui.UseEvent(func() {
		moduleStatus.Set("Lazy module import requested.")
		moduleResult.Set("Waiting for the module exports.")
		moduleLoadCount.Update(func(previous int) int { return previous + 1 })
	})

	ui.UseEffect(func() func() {
		if moduleLoadCount.Get() == 0 {
			return nil
		}

		moduleStatus.Set("Importing a lazy module through interop.ImportModule(...).")

		module, err := interop.ImportModule(context.Background(), browserInteropLazyModuleSpecifier)
		if err != nil {
			moduleStatus.Set(describeError("Lazy module import failed", err))
			return nil
		}
		defer module.Dispose()

		name, err := module.Value(context.Background(), "bundleName")
		if err != nil {
			moduleStatus.Set(describeError("Lazy module bundle name lookup failed", err))
			return nil
		}

		helperLabel, err := module.Call(context.Background(), "formatLabel", draft.Get())
		if err != nil {
			moduleStatus.Set(describeError("Lazy module helper call failed", err))
			return nil
		}

		defaultLabel, err := module.CallDefault(context.Background(), draft.Get())
		if err != nil {
			moduleStatus.Set(describeError("Lazy module default export failed", err))
			return nil
		}

		moduleStatus.Set(fmt.Sprintf("Imported %s through interop.ImportModule(...).", name))
		moduleResult.Set(fmt.Sprintf("%s | %s", helperLabel, defaultLabel))
		return nil
	}, moduleLoadCount.Get())

	saveStorage := ui.UseEvent(func() {
		storage, err := interop.LocalStorage()
		if err != nil {
			storageStatus.Set(describeError("LocalStorage unavailable", err))
			return
		}
		if err := storage.SetItem("interop-demo:draft", draft.Get()); err != nil {
			storageStatus.Set(describeError("Failed to store draft", err))
			return
		}
		if err := storage.SetItem("interop-demo:scheme", colorScheme.Get()); err != nil {
			storageStatus.Set(describeError("Failed to store media state", err))
			return
		}
		loadSnapshot(storage)
	})

	loadStorage := ui.UseEvent(func() {
		storage, err := interop.LocalStorage()
		if err != nil {
			storageStatus.Set(describeError("LocalStorage unavailable", err))
			return
		}
		loadSnapshot(storage)
	})

	copyDraft := ui.UseEvent(func() {
		clipboard, err := interop.NavigatorClipboard()
		if err != nil {
			clipboardStatus.Set(describeError("Clipboard unavailable", err))
			return
		}
		if err := clipboard.WriteText(context.Background(), draft.Get()); err != nil {
			clipboardStatus.Set(describeError("Clipboard write failed", err))
			return
		}
		clipboardStatus.Set("Copied the current draft through NavigatorClipboard().")
	})

	readClipboard := ui.UseEvent(func() {
		clipboard, err := interop.NavigatorClipboard()
		if err != nil {
			clipboardStatus.Set(describeError("Clipboard unavailable", err))
			return
		}
		value, err := clipboard.ReadText(context.Background())
		if err != nil {
			clipboardStatus.Set(describeError("Clipboard read failed", err))
			return
		}
		clipboardStatus.Set(fmt.Sprintf("Clipboard now contains %q.", value))
	})

	dispatchPulse := ui.UseEvent(func() {
		target, err := interop.DocumentEvents()
		if err != nil {
			eventStatus.Set(describeError("Document event target unavailable", err))
			return
		}
		next := eventCount.Get() + 1
		if err := target.Dispatch("interop-demo", demoPulse{
			Message: draft.Get(),
			Source:  "Go button",
			Count:   next,
		}); err != nil {
			eventStatus.Set(describeError("Custom event dispatch failed", err))
			return
		}
	})

	ui.UseEffect(func() func() {
		cleanups := make([]func(), 0, 3)

		storage, err := interop.LocalStorage()
		if err != nil {
			storageStatus.Set(describeError("LocalStorage unavailable", err))
		} else {
			loadSnapshot(storage)
		}

		media, err := interop.MatchMedia("(prefers-color-scheme: dark)")
		if err != nil {
			colorScheme.Set("unavailable")
		} else {
			if media.Matches() {
				colorScheme.Set("dark")
			} else {
				colorScheme.Set("light")
			}
			subscription, err := media.Subscribe(func(event interop.MediaQueryEvent) {
				if event.Matches {
					colorScheme.Set("dark")
					return
				}
				colorScheme.Set("light")
			})
			if err == nil {
				cleanups = append(cleanups, subscription.Cancel)
			}
		}

		document, err := interop.CurrentDocument()
		if err != nil {
			hostLookup.Set(describeError("Document lookup unavailable", err))
		} else {
			elements, err := document.ElementsByID("interop-demo-panel", "interop-demo-status")
			if err != nil {
				hostLookup.Set(describeError("Grouped host lookup failed", err))
			} else {
				hostLookup.Set(fmt.Sprintf("Resolved %d hosts through document.ElementsByID(...).", len(elements)))
				if panel, ok := elements["interop-demo-panel"]; ok {
					if rect, err := panel.BoundingClientRect(); err == nil {
						panelSize.Set(fmt.Sprintf("%.0f x %.0f", rect.Width, rect.Height))
					}
					subscription, err := panel.ObserveResize(func(entry interop.ResizeEntry) {
						panelSize.Set(fmt.Sprintf("%.0f x %.0f", entry.ContentRect.Width, entry.ContentRect.Height))
					})
					if err == nil {
						cleanups = append(cleanups, subscription.Cancel)
					}
				}
			}
		}

		target, err := interop.DocumentEvents()
		if err != nil {
			eventStatus.Set(describeError("Document event target unavailable", err))
		} else {
			subscription, err := interop.SubscribeDecoded(target, "interop-demo", func(event interop.DecodedCustomEvent[demoPulse], err error) {
				if err != nil {
					eventStatus.Set(describeError("Typed custom-event decode failed", err))
					return
				}
				eventCount.Set(event.Detail.Count)
				eventSource.Set(event.Detail.Source)
				eventStatus.Set(fmt.Sprintf("Received %q through interop.SubscribeDecoded(...).", event.Detail.Message))
			})
			if err == nil {
				cleanups = append(cleanups, subscription.Cancel)
			}
		}

		return func() {
			for i := len(cleanups) - 1; i >= 0; i-- {
				cleanups[i]()
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
				Value:       draft.Get(),
				OnInput:     updateDraft,
				Placeholder: "Type a browser-side draft",
				Class:       "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
			}),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Save to storage", saveStorage),
				shared.ExampleButton("Load snapshot", loadStorage),
				shared.ExampleButton("Copy draft", copyDraft),
				shared.ExampleButton("Read clipboard", readClipboard),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Saved draft", savedDraft.Get()),
				shared.ExampleStat("Saved scheme", savedScheme.Get()),
			),
			html.P(html.Props{ID: "interop-demo-status", Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(storageStatus.Get())),
			html.P(html.Props{Class: "mt-2 text-sm leading-7 text-slate-300"}, html.Text(clipboardStatus.Get())),
		),
		shared.ExamplePanel("Observers and DOM lookup",
			html.Div(html.Props{ID: "interop-demo-panel", Class: "mt-3 rounded-[1.25rem] border border-cyan-300/20 bg-cyan-500/5 p-5"},
				html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text("This panel is resolved through document.ElementsByID(...) and measured through ObserveResize(...). Resize the browser or change the catalog width to watch the live measurement update.")),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Color scheme", colorScheme.Get()),
				shared.ExampleStat("Panel size", panelSize.Get()),
				shared.ExampleStat("Host lookup", hostLookup.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The example subscribes to prefers-color-scheme through MatchMedia(...) and keeps browser handles local to the effect that owns them.")),
		),
		shared.ExamplePanel("Typed custom events",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Dispatch a document-level CustomEvent through interop.DocumentEvents() and decode it back into a typed Go payload through interop.SubscribeDecoded(...).")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Dispatch pulse", dispatchPulse),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Event count", fmt.Sprintf("%d", eventCount.Get())),
				shared.ExampleStat("Event source", eventSource.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(eventStatus.Get())),
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
				shared.ExampleButton("Load lazy module", loadLazyModule),
			),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Module status", moduleStatus.Get()),
				shared.ExampleStat("Module result", moduleResult.Get()),
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
	ui.Render(ui.CreateElement(browserInteropExample), "#app")
	select {}
}
