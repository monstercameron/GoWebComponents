//go:build js && wasm

package app

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

type canvasWorkspaceController struct {
	OpenFromMessage  ui.Handler
	Close            ui.Handler
	Reload           ui.Handler
	RefreshPreview   ui.Handler
	ToggleOverlay    ui.Handler
	OpenCanvasOnly   ui.Handler
	BackToThread     ui.Handler
	ToggleConsole    ui.Handler
	ClearConsole     ui.Handler
	ChangeFocus      ui.Handler
	HandleFocusDraft ui.Handler
	ApplyFocusDraft  ui.Handler
	ResetOriginal    ui.Handler
	RevertLastPatch  ui.Handler
	CopyCurrentCode  ui.Handler
	StartSplitDrag   ui.Handler
	HandleSplitKey   ui.Handler
}

func useCanvasWorkspace(
	app ui.Reducer[appState, appAction],
	nav router.Navigator,
	sidebarOpenState state.Atom[bool],
	currentThreadPublicID string,
	currentCanvasRouteID string,
) canvasWorkspaceController {
	splitLoaded := ui.UseRef(false)
	beginSplitDrag := func() {
		window := js.Global().Get("window")
		if !window.Truthy() || window.Get("addEventListener").Type() != js.TypeFunction {
			return
		}
		moveListener := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				return nil
			}
			width := window.Get("innerWidth").Float()
			if width <= 0 {
				return nil
			}
			ratio := args[0].Get("clientX").Float() / width
			if ratio < canvasSplitMin {
				ratio = canvasSplitMin
			}
			if ratio > canvasSplitMax {
				ratio = canvasSplitMax
			}
			app.Dispatch(appAction{Type: appActionSetCanvasSplitRatio, CanvasSplitRatio: ratio})
			return nil
		})
		var upListener js.Func
		upListener = js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			window.Call("removeEventListener", "mousemove", moveListener)
			window.Call("removeEventListener", "mouseup", upListener)
			moveListener.Release()
			upListener.Release()
			return nil
		})
		window.Call("addEventListener", "mousemove", moveListener)
		window.Call("addEventListener", "mouseup", upListener)
	}

	ui.UseEffect(func() func() {
		if splitLoaded.Get() {
			return nil
		}
		splitLoaded.Set(true)
		storage, err := interop.GetLocalStorage()
		if err != nil {
			return nil
		}
		value, ok, err := storage.GetItem(storageKeyCanvasSplit)
		if err != nil || !ok {
			return nil
		}
		ratio, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil && ratio >= canvasSplitMin && ratio <= canvasSplitMax {
			app.Dispatch(appAction{Type: appActionSetCanvasSplitRatio, CanvasSplitRatio: ratio})
		}
		return nil
	}, app.Get().CanvasSession.SplitRatio)

	ui.UseEffect(func() func() {
		ratio := app.Get().CanvasSession.SplitRatio
		if ratio < canvasSplitMin || ratio > canvasSplitMax {
			return nil
		}
		storage, err := interop.GetLocalStorage()
		if err != nil {
			return nil
		}
		_ = storage.SetItem(storageKeyCanvasSplit, fmt.Sprintf("%.4f", ratio))
		return nil
	}, app.Get().CanvasSession.SplitRatio)

	ui.UseEffect(func() func() {
		canvasID := strings.TrimSpace(currentCanvasRouteID)
		if canvasID == "" {
			return nil
		}
		artifact, ok := findCanvasArtifact(app.Get().Messages, canvasID)
		if !ok {
			return nil
		}
		session := app.Get().CanvasSession
		if session.Active && session.ArtifactID == canvasID {
			return nil
		}
		sidebarOpenState.Set(false)
		app.Dispatch(appAction{Type: appActionOpenCanvasSession, CanvasArtifact: artifact})
		return nil
	}, currentCanvasRouteID, app.Get().Messages)

	ui.UseEffect(func() func() {
		window := js.Global().Get("window")
		if !window.Truthy() || window.Get("addEventListener").Type() != js.TypeFunction {
			return nil
		}
		listener := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				return nil
			}
			data := args[0].Get("data")
			if !data.Truthy() || !data.Get("__gwcCanvas").Truthy() || !data.Get("__gwcCanvas").Bool() {
				return nil
			}
			sessionID := strings.TrimSpace(data.Get("sessionID").String())
			if sessionID == "" || sessionID != app.Get().CanvasSession.SessionID {
				return nil
			}
			renderVersion := data.Get("renderVersion").Int()
			kind := strings.TrimSpace(data.Get("kind").String())
			payload := data.Get("payload")
			switch kind {
			case "console":
				app.Dispatch(appAction{Type: appActionCanvasAppendConsole, CanvasConsoleEntry: canvasConsoleEntry{
					Level:         strings.TrimSpace(payload.Get("level").String()),
					Message:       strings.TrimSpace(payload.Get("message").String()),
					RenderVersion: renderVersion,
				}})
			case "runtime_error":
				app.Dispatch(appAction{Type: appActionCanvasAppendConsole, CanvasConsoleEntry: canvasConsoleEntry{
					Level:         "error",
					Message:       strings.TrimSpace(payload.Get("message").String()),
					Detail:        strings.TrimSpace(payload.Get("detail").String()),
					RenderVersion: renderVersion,
				}})
				app.Dispatch(appAction{Type: appActionCanvasSetStatus, CanvasPreviewStatus: canvasPreviewRuntimeErr, CanvasRuntimeStatus: "runtime error"})
			case "status":
				app.Dispatch(appAction{Type: appActionCanvasSetStatus,
					CanvasPreviewStatus: strings.TrimSpace(payload.Get("previewStatus").String()),
					CanvasRuntimeStatus: strings.TrimSpace(payload.Get("runtimeStatus").String()),
				})
			}
			return nil
		})
		window.Call("addEventListener", "message", listener)
		return func() {
			window.Call("removeEventListener", "message", listener)
			listener.Release()
		}
	}, app.Get().CanvasSession.SessionID)

	ui.UseEffect(func() func() {
		if !app.Get().CanvasSession.Active || app.Get().CanvasSession.LayoutMode != canvasLayoutSplit {
			return nil
		}
		document := js.Global().Get("document")
		if !document.Truthy() || document.Get("getElementById").Type() != js.TypeFunction {
			return nil
		}
		handle := document.Call("getElementById", idCanvasSplitHandle)
		if !handle.Truthy() || handle.Get("addEventListener").Type() != js.TypeFunction {
			return nil
		}
		listener := js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			beginSplitDrag()
			return nil
		})
		handle.Call("addEventListener", "mousedown", listener)
		return func() {
			handle.Call("removeEventListener", "mousedown", listener)
			listener.Release()
		}
	}, app.Get().CanvasSession.Active, app.Get().CanvasSession.LayoutMode)

	openFromMessage := ui.UseEvent(func(e ui.Event) {
		artifactID := eventDatasetValue(e, dataCanvasID)
		if artifactID == "" {
			return
		}
		artifact, ok := findCanvasArtifact(app.Get().Messages, artifactID)
		if !ok {
			return
		}
		if app.Get().CanvasSession.Active {
			app.Dispatch(appAction{Type: appActionApplyCanvasArtifact, CanvasArtifact: artifact})
			return
		}
		sidebarOpenState.Set(false)
		app.Dispatch(appAction{Type: appActionOpenCanvasSession, CanvasArtifact: artifact})
	})

	closeCanvas := ui.UseEvent(func() {
		if strings.TrimSpace(currentCanvasRouteID) != "" {
			if strings.TrimSpace(currentThreadPublicID) != "" {
				nav.Navigate(chatThreadPath(currentThreadPublicID))
			} else {
				nav.Navigate(chatRouteRoot)
			}
		}
		app.Dispatch(appAction{Type: appActionCloseCanvasSession})
	})

	refreshPreview := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		app.Dispatch(appAction{Type: appActionCanvasRefreshPreview})
	})

	reloadCanvas := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		app.Dispatch(appAction{Type: appActionCanvasClearConsole})
		app.Dispatch(appAction{Type: appActionCanvasRefreshPreview})
	})

	toggleOverlay := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		layout := canvasLayoutOverlay
		if app.Get().CanvasSession.LayoutMode == canvasLayoutOverlay {
			layout = canvasLayoutSplit
		}
		app.Dispatch(appAction{Type: appActionSetCanvasLayoutMode, CanvasLayoutMode: layout})
	})

	openCanvasOnly := ui.UseEvent(func() {
		session := app.Get().CanvasSession
		if !session.Active || strings.TrimSpace(currentThreadPublicID) == "" {
			return
		}
		nav.Navigate(chatCanvasPath(currentThreadPublicID, session.ArtifactID))
	})

	backToThread := ui.UseEvent(func() {
		if strings.TrimSpace(currentThreadPublicID) == "" {
			nav.Navigate(chatRouteRoot)
			return
		}
		nav.Navigate(chatThreadPath(currentThreadPublicID))
	})

	toggleConsole := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		app.Dispatch(appAction{Type: appActionCanvasToggleConsole})
	})

	clearConsole := ui.UseEvent(func() {
		app.Dispatch(appAction{Type: appActionCanvasClearConsole})
	})

	changeFocus := ui.UseEvent(func(e ui.Event) {
		session := app.Get().CanvasSession
		if !session.Active {
			return
		}
		rawIndex := eventDatasetValue(e, dataCanvasFocus)
		index, err := strconv.Atoi(rawIndex)
		if err != nil || index < 0 || index >= len(session.FocusOptions) {
			return
		}
		app.Dispatch(appAction{Type: appActionSetCanvasFocus, CanvasFocus: session.FocusOptions[index]})
	})

	handleFocusDraft := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetCanvasFocusDraft, CanvasFocusDraft: e.GetValue()})
	})

	applyFocusDraft := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		app.Dispatch(appAction{Type: appActionApplyCanvasFocusDraft})
	})

	resetOriginal := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		app.Dispatch(appAction{Type: appActionCanvasResetOriginal})
	})

	revertLastPatch := ui.UseEvent(func() {
		if !app.Get().CanvasSession.Active {
			return
		}
		app.Dispatch(appAction{Type: appActionCanvasRevertLastPatch})
	})

	copyCurrentCode := ui.UseEvent(func() {
		copyToClipboard(app.Get().CanvasSession.CurrentSource)
	})

	startSplitDrag := ui.UseEvent(func(e ui.Event) {
		if !app.Get().CanvasSession.Active {
			return
		}
		beginSplitDrag()
		e.PreventDefault()
	})

	handleSplitKey := ui.UseEvent(func(e ui.Event) {
		key := e.JSValue().Get("key").String()
		ratio := app.Get().CanvasSession.SplitRatio
		switch key {
		case "ArrowLeft":
			ratio -= 0.04
		case "ArrowRight":
			ratio += 0.04
		default:
			return
		}
		if ratio < canvasSplitMin {
			ratio = canvasSplitMin
		}
		if ratio > canvasSplitMax {
			ratio = canvasSplitMax
		}
		app.Dispatch(appAction{Type: appActionSetCanvasSplitRatio, CanvasSplitRatio: ratio})
		e.PreventDefault()
	})

	return canvasWorkspaceController{
		OpenFromMessage:  openFromMessage,
		Close:            closeCanvas,
		Reload:           reloadCanvas,
		RefreshPreview:   refreshPreview,
		ToggleOverlay:    toggleOverlay,
		OpenCanvasOnly:   openCanvasOnly,
		BackToThread:     backToThread,
		ToggleConsole:    toggleConsole,
		ClearConsole:     clearConsole,
		ChangeFocus:      changeFocus,
		HandleFocusDraft: handleFocusDraft,
		ApplyFocusDraft:  applyFocusDraft,
		ResetOriginal:    resetOriginal,
		RevertLastPatch:  revertLastPatch,
		CopyCurrentCode:  copyCurrentCode,
		StartSplitDrag:   startSplitDrag,
		HandleSplitKey:   handleSplitKey,
	}
}

func canvasWorkspacePane(intl i18n.Runtime, session canvasSessionState, controller canvasWorkspaceController, canvasOnly bool) ui.Node {
	if !session.Active {
		return nil
	}
	document := buildCanvasRuntimeDocument(session.CurrentSource, session.SessionID, session.ArtifactID, session.LatestRenderedVersion)
	menuButtonClass := "rounded-md border border-white/10 px-2.5 py-1 text-[11px] text-white/72 hover:bg-white/10 transition-colors whitespace-nowrap"
	metaBadgeClass := "shrink-0 rounded-full border border-white/10 bg-white/4 px-2 py-0.5 text-[10px] uppercase tracking-[0.12em] text-white/45"
	return Div(
		ID(idCanvasWorkspace),
		Class("flex h-full min-h-0 min-w-0 flex-col border-l border-white/8 bg-[#101010]"),
		Div(Class("flex items-center gap-2 border-b border-white/8 px-3 py-2"),
			If(canvasOnly,
				Button(
					Class(menuButtonClass),
					OnClick(controller.BackToThread),
					Text("Back to thread"),
				),
			),
			Div(Class("min-w-0 flex-1 flex items-center gap-2 overflow-hidden"),
				Div(Class("truncate text-sm font-semibold text-white/92"), Text(session.CurrentFileID)),
				Span(Class(metaBadgeClass), Text(session.PreviewStatus)),
				Span(Class(metaBadgeClass), Text(session.RuntimeStatus)),
				If(session.Dirty,
					Span(Class("shrink-0 rounded-full border border-[#19c37d]/25 bg-[#19c37d]/12 px-2 py-0.5 text-[10px] uppercase tracking-[0.12em] text-[#9af7d0]/72"), Text("dirty")),
				),
			),
			Div(Class("chat-scrollbar flex items-center gap-1 overflow-x-auto whitespace-nowrap"),
				Button(Class(menuButtonClass), OnClick(controller.Reload), Text("Reload")),
				Button(Class(menuButtonClass), OnClick(controller.RefreshPreview), Text("Refresh")),
				Button(Class(menuButtonClass), OnClick(controller.ToggleConsole), Text("Console")),
				If(!canvasOnly,
					Button(Class(menuButtonClass), OnClick(controller.ToggleOverlay), Text("Overlay")),
				),
				If(!canvasOnly,
					Button(Class(menuButtonClass), OnClick(controller.OpenCanvasOnly), Text("Canvas only")),
				),
				Button(Class(menuButtonClass), OnClick(controller.Close), Text("Close")),
			),
		),
		Div(Class("grid min-h-0 flex-1 grid-cols-1 xl:grid-cols-[minmax(16rem,0.58fr)_minmax(0,1.42fr)]"),
			Div(Class("chat-scrollbar flex min-h-0 flex-col overflow-y-auto border-b border-white/8 xl:border-b-0 xl:border-r"),
				Div(Class("border-b border-white/8 px-3 py-2"),
					Div(Class("flex flex-wrap items-center gap-2"),
						Map(session.FocusOptions, func(region canvasFocusRegion) ui.Node {
							index := canvasFocusIndex(session.FocusOptions, region)
							active := canvasFocusSameLineRange(region, session.FocusedRegion)
							return Button(
								Class(ClassNames(
									"rounded-full border px-3 py-1.5 text-xs transition-colors",
									When(active, "border-[#19c37d]/35 bg-[#19c37d]/12 text-[#d8fff1]"),
									When(!active, "border-white/10 text-white/55 hover:bg-white/8 hover:text-white/85"),
								)),
								Data(dataCanvasFocus, strconv.Itoa(index)),
								OnClick(controller.ChangeFocus),
								Text(canvasFocusPillLabel(region)),
							)
						}),
					),
					Div(Class("mt-3 flex flex-wrap items-center gap-2 text-xs text-white/45"),
						Span(Textf("file: %s", session.CurrentFileID)),
						Span(Textf("focus: %s", canvasFocusDisplayLabel(session.FocusedRegion))),
						Span(Textf("lines: %d-%d", session.FocusedRegion.StartLine, session.FocusedRegion.EndLine)),
					),
				),
				Div(Class("grid gap-2 px-3 py-3"),
					Div(Class("rounded-[1.2rem] border border-[#19c37d]/16 bg-[linear-gradient(180deg,rgba(17,34,28,0.72),rgba(10,19,16,0.78))] p-3"),
						Div(Class("mb-2 flex items-center justify-between gap-3"),
							Div(
								Class("text-xs uppercase tracking-[0.18em] text-[#9af7d0]/62"),
								Textf("Focused region: %s", canvasFocusDisplayLabel(session.FocusedRegion)),
							),
							Div(Class("flex items-center gap-2"),
								Button(Class(menuButtonClass), OnClick(controller.ApplyFocusDraft), Text("Apply patch")),
								Button(Class(menuButtonClass), OnClick(controller.ResetOriginal), Text("Reset")),
								Button(Class(menuButtonClass), OnClick(controller.RevertLastPatch), Text("Revert")),
							),
						),
						Tag("textarea",
							ID(idCanvasFocusEditor),
							Class("chat-scrollbar min-h-[10rem] w-full resize-y rounded-2xl border border-white/8 bg-black/20 px-4 py-3 font-mono text-[0.92rem] leading-6 text-white/90 focus:outline-none"),
							Value(session.FocusDraft),
							OnInput(controller.HandleFocusDraft),
						),
						Div(Class("mt-3 flex flex-wrap items-center gap-2"),
							Button(Class(menuButtonClass), OnClick(controller.CopyCurrentCode), Text("Copy code")),
							Span(Class("text-xs text-white/40"), Textf("patches: %d", len(session.PatchHistory))),
						),
					),
					Tag("details",
						Class("rounded-[1.25rem] border border-white/8 bg-black/16 p-3"),
						Tag("summary",
							Class("text-xs uppercase tracking-[0.18em] text-white/45"),
							Text("Current file"),
						),
						Div(Class("mt-2"),
							canvasSourceView(session.CurrentSource, session.FocusedRegion),
						),
					),
					If(len(session.PatchHistory) > 0,
						Tag("details",
							Class("rounded-[1.25rem] border border-white/8 bg-black/16 p-3"),
							Tag("summary",
								Class("text-xs uppercase tracking-[0.18em] text-white/45"),
								Text("Patch history"),
							),
							Div(Class("mt-2 flex flex-col gap-2"),
								Map(session.PatchHistory, func(record canvasPatchRecord) ui.Node {
									return Div(Class("rounded-2xl border border-white/8 bg-white/3 px-3 py-2"),
										Div(Class("text-sm text-white/85"), Text(record.Summary)),
										Div(Class("mt-1 text-xs text-white/42"), Textf("%s lines %d-%d", record.Type, record.StartLine, record.EndLine)),
									)
								}),
							),
						),
					),
				),
			),
			Div(Class("flex min-h-0 flex-col bg-[#0b0b0b]"),
				Div(Class("flex items-center justify-between gap-3 border-b border-white/8 px-3 py-2"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-white/38"), Text("Preview")),
					Div(Class("text-xs text-white/45"), Textf("render v%d", session.LatestRenderedVersion)),
				),
				Div(Class("min-h-0 flex-1 p-2"),
					Tag("iframe",
						ID(idCanvasFrame),
						Class("h-full min-h-[34rem] w-full rounded-[1rem] border border-black/20 bg-white shadow-[0_24px_70px_rgba(0,0,0,0.28)]"),
						FromProps(Props{Raw: map[string]interface{}{
							"sandbox": "allow-scripts",
							"srcdoc":  document,
							"title":   intl.T(chatI18nNamespace, "canvas.title"),
						}}),
					),
				),
			),
		),
		If(session.ConsoleOpen,
			Div(
				ID(idCanvasConsole),
				Class("chat-scrollbar max-h-[10rem] overflow-y-auto border-t border-white/8 bg-black/36 px-3 py-2"),
				Div(Class("mb-3 flex items-center justify-between gap-3"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-white/38"), Text("Console")),
					Button(Class(menuButtonClass), OnClick(controller.ClearConsole), Text("Clear logs")),
				),
				If(len(session.ConsoleEntries) == 0,
					Div(Class("text-sm text-white/35"), Text("No console output yet.")),
				),
				Div(Class("flex flex-col gap-2"),
					Map(session.ConsoleEntries, func(entry canvasConsoleEntry) ui.Node {
						return Div(Class("rounded-2xl border border-white/8 bg-white/3 px-3 py-2 font-mono text-xs leading-5"),
							Div(Class("flex items-center gap-2"),
								Span(Class("uppercase tracking-[0.18em] text-white/38"), Text(entry.Level)),
								Span(Class("text-white/25"), Textf("render %d", entry.RenderVersion)),
							),
							Div(Class("mt-1 whitespace-pre-wrap text-white/82"), Text(entry.Message)),
							If(strings.TrimSpace(entry.Detail) != "",
								Div(Class("mt-1 whitespace-pre-wrap text-[#ffbdbd]/78"), Text(entry.Detail)),
							),
						)
					}),
				),
			),
		),
	)
}

func canvasSourceView(source string, focus canvasFocusRegion) ui.Node {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	rows := make([]ui.Node, 0, len(lines))
	for idx, line := range lines {
		lineNumber := idx + 1
		highlighted := lineNumber >= focus.StartLine && lineNumber <= focus.EndLine
		rows = append(rows, Div(
			Class(ClassNames(
				"grid grid-cols-[3.5rem_minmax(0,1fr)] gap-3 px-3 py-0.5",
				When(highlighted, "bg-[#19c37d]/10"),
			)),
			Span(Class("select-none text-right text-white/26"), Text(strconv.Itoa(lineNumber))),
			Span(Class("whitespace-pre-wrap break-words text-white/84"), Text(line)),
		))
	}
	rowChildren := make([]interface{}, 0, len(rows)+1)
	rowChildren = append(rowChildren, Class("min-w-full font-mono text-[0.82rem] leading-6"))
	for _, row := range rows {
		rowChildren = append(rowChildren, row)
	}
	return Div(Class("chat-scrollbar max-h-[22rem] overflow-auto rounded-2xl border border-white/8 bg-[#0d0d0d]"),
		Div(rowChildren...),
	)
}

func canvasFocusIndex(options []canvasFocusRegion, target canvasFocusRegion) int {
	for idx, option := range options {
		if canvasFocusSameLineRange(option, target) {
			return idx
		}
	}
	return 0
}

func canvasFocusSameLineRange(a, b canvasFocusRegion) bool {
	return a.StartLine == b.StartLine && a.EndLine == b.EndLine && a.Label == b.Label
}

func canvasFocusPillLabel(region canvasFocusRegion) string {
	label := strings.TrimSpace(region.Label)
	if label != "" {
		return label
	}
	if region.Kind == "file" {
		return "Full file"
	}
	return fmt.Sprintf("%s %d-%d", region.Kind, region.StartLine, region.EndLine)
}
