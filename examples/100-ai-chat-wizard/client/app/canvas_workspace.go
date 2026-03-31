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

func parseUseCanvasWorkspace(
	parseApp ui.Reducer[appState, appAction],
	parseNav router.Navigator,
	parseSidebarOpenState state.Atom[bool],
	parseCurrentThreadPublicID string,
	parseCurrentCanvasRouteID string,
) canvasWorkspaceController {
	parseSplitLoaded := ui.UseRef(false)
	parseBeginSplitDrag := func() {
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() || parseWindow.Get("addEventListener").Type() != js.TypeFunction {
			return
		}
		parseMoveListener := js.FuncOf(func(_ js.Value, parseArgs []js.Value) interface{} {
			if len(parseArgs) == 0 {
				return nil
			}
			parseWidth := parseWindow.Get("innerWidth").Float()
			if parseWidth <= 0 {
				return nil
			}
			parseRatio := parseArgs[0].Get("clientX").Float() / parseWidth
			if parseRatio < canvasSplitMin {
				parseRatio = canvasSplitMin
			}
			if parseRatio > canvasSplitMax {
				parseRatio = canvasSplitMax
			}
			parseApp.Dispatch(appAction{Type: appActionSetCanvasSplitRatio, CanvasSplitRatio: parseRatio})
			return nil
		})
		var parseUpListener js.Func
		parseUpListener = js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			parseWindow.Call("removeEventListener", "mousemove", parseMoveListener)
			parseWindow.Call("removeEventListener", "mouseup", parseUpListener)
			parseMoveListener.Release()
			parseUpListener.Release()
			return nil
		})
		parseWindow.Call("addEventListener", "mousemove", parseMoveListener)
		parseWindow.Call("addEventListener", "mouseup", parseUpListener)
	}

	ui.UseEffect(func() func() {
		if parseSplitLoaded.Get() {
			return nil
		}
		parseSplitLoaded.Set(true)
		parseStorage, parseErr := interop.GetLocalStorage()
		if parseErr != nil {
			return nil
		}
		parseValue, parseOk, parseErr := parseStorage.GetItem(storageKeyCanvasSplit)
		if parseErr != nil || !parseOk {
			return nil
		}
		parseRatio2, parseErr := strconv.ParseFloat(strings.TrimSpace(parseValue), 64)
		if parseErr == nil && parseRatio2 >= canvasSplitMin && parseRatio2 <= canvasSplitMax {
			parseApp.Dispatch(appAction{Type: appActionSetCanvasSplitRatio, CanvasSplitRatio: parseRatio2})
		}
		return nil
	}, parseApp.Get().CanvasSession.SplitRatio)

	ui.UseEffect(func() func() {
		parseRatio3 := parseApp.Get().CanvasSession.SplitRatio
		if parseRatio3 < canvasSplitMin || parseRatio3 > canvasSplitMax {
			return nil
		}
		parseStorage2, parseErr2 := interop.GetLocalStorage()
		if parseErr2 != nil {
			return nil
		}
		_ = parseStorage2.SetItem(storageKeyCanvasSplit, fmt.Sprintf("%.4f", parseRatio3))
		return nil
	}, parseApp.Get().CanvasSession.SplitRatio)

	ui.UseEffect(func() func() {
		parseCanvasID := strings.TrimSpace(parseCurrentCanvasRouteID)
		if parseCanvasID == "" {
			return nil
		}
		parseArtifact, parseOk2 := parseFindCanvasArtifact(parseApp.Get().Messages, parseCanvasID)
		if !parseOk2 {
			return nil
		}
		parseSession := parseApp.Get().CanvasSession
		if parseSession.Active && parseSession.ArtifactID == parseCanvasID {
			return nil
		}
		parseSidebarOpenState.Set(false)
		parseApp.Dispatch(appAction{Type: appActionOpenCanvasSession, CanvasArtifact: parseArtifact})
		return nil
	}, parseCurrentCanvasRouteID, parseApp.Get().Messages)

	ui.UseEffect(func() func() {
		parseWindow2 := js.Global().Get("window")
		if !parseWindow2.Truthy() || parseWindow2.Get("addEventListener").Type() != js.TypeFunction {
			return nil
		}
		parseListener := js.FuncOf(func(_ js.Value, parseArgs2 []js.Value) interface{} {
			if len(parseArgs2) == 0 {
				return nil
			}
			parseData := parseArgs2[0].Get("data")
			if !parseData.Truthy() || !parseData.Get("__gwcCanvas").Truthy() || !parseData.Get("__gwcCanvas").Bool() {
				return nil
			}
			parseSessionID := strings.TrimSpace(parseData.Get("sessionID").String())
			if parseSessionID == "" || parseSessionID != parseApp.Get().CanvasSession.SessionID {
				return nil
			}
			renderVersion := parseData.Get("renderVersion").Int()
			parseKind := strings.TrimSpace(parseData.Get("kind").String())
			parsePayload := parseData.Get("payload")
			switch parseKind {
			case "console":
				parseApp.Dispatch(appAction{Type: appActionCanvasAppendConsole, CanvasConsoleEntry: canvasConsoleEntry{
					Level:         strings.TrimSpace(parsePayload.Get("level").String()),
					Message:       strings.TrimSpace(parsePayload.Get("message").String()),
					RenderVersion: renderVersion,
				}})
			case "runtime_error":
				parseApp.Dispatch(appAction{Type: appActionCanvasAppendConsole, CanvasConsoleEntry: canvasConsoleEntry{
					Level:         "error",
					Message:       strings.TrimSpace(parsePayload.Get("message").String()),
					Detail:        strings.TrimSpace(parsePayload.Get("detail").String()),
					RenderVersion: renderVersion,
				}})
				parseApp.Dispatch(appAction{Type: appActionCanvasSetStatus, CanvasPreviewStatus: canvasPreviewRuntimeErr, CanvasRuntimeStatus: "runtime error"})
			case "status":
				parseApp.Dispatch(appAction{Type: appActionCanvasSetStatus,
					CanvasPreviewStatus: strings.TrimSpace(parsePayload.Get("previewStatus").String()),
					CanvasRuntimeStatus: strings.TrimSpace(parsePayload.Get("runtimeStatus").String()),
				})
			}
			return nil
		})
		parseWindow2.Call("addEventListener", "message", parseListener)
		return func() {
			parseWindow2.Call("removeEventListener", "message", parseListener)
			parseListener.Release()
		}
	}, parseApp.Get().CanvasSession.SessionID)

	ui.UseEffect(func() func() {
		if !parseApp.Get().CanvasSession.Active || parseApp.Get().CanvasSession.LayoutMode != canvasLayoutSplit {
			return nil
		}
		parseDocument := js.Global().Get("document")
		if !parseDocument.Truthy() || parseDocument.Get("getElementById").Type() != js.TypeFunction {
			return nil
		}
		handle := parseDocument.Call("getElementById", idCanvasSplitHandle)
		if !handle.Truthy() || handle.Get("addEventListener").Type() != js.TypeFunction {
			return nil
		}
		parseListener2 := js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			parseBeginSplitDrag()
			return nil
		})
		handle.Call("addEventListener", "mousedown", parseListener2)
		return func() {
			handle.Call("removeEventListener", "mousedown", parseListener2)
			parseListener2.Release()
		}
	}, parseApp.Get().CanvasSession.Active, parseApp.Get().CanvasSession.LayoutMode)

	parseOpenFromMessage := ui.UseEvent(func(parseE ui.Event) {
		parseArtifactID := parseEventDatasetValue(parseE, dataCanvasID)
		if parseArtifactID == "" {
			return
		}
		parseArtifact2, parseOk3 := parseFindCanvasArtifact(parseApp.Get().Messages, parseArtifactID)
		if !parseOk3 {
			return
		}
		if parseApp.Get().CanvasSession.Active {
			parseApp.Dispatch(appAction{Type: appActionApplyCanvasArtifact, CanvasArtifact: parseArtifact2})
			return
		}
		parseSidebarOpenState.Set(false)
		parseApp.Dispatch(appAction{Type: appActionOpenCanvasSession, CanvasArtifact: parseArtifact2})
	})

	parseCloseCanvas := ui.UseEvent(func() {
		if strings.TrimSpace(parseCurrentCanvasRouteID) != "" {
			if strings.TrimSpace(parseCurrentThreadPublicID) != "" {
				parseNav.Navigate(parseChatThreadPath(parseCurrentThreadPublicID))
			} else {
				parseNav.Navigate(chatRouteRoot)
			}
		}
		parseApp.Dispatch(appAction{Type: appActionCloseCanvasSession})
	})

	parseRefreshPreview := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionCanvasRefreshPreview})
	})

	parseReloadCanvas := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionCanvasClearConsole})
		parseApp.Dispatch(appAction{Type: appActionCanvasRefreshPreview})
	})

	parseToggleOverlay := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseLayout := canvasLayoutOverlay
		if parseApp.Get().CanvasSession.LayoutMode == canvasLayoutOverlay {
			parseLayout = canvasLayoutSplit
		}
		parseApp.Dispatch(appAction{Type: appActionSetCanvasLayoutMode, CanvasLayoutMode: parseLayout})
	})

	parseOpenCanvasOnly := ui.UseEvent(func() {
		parseSession2 := parseApp.Get().CanvasSession
		if !parseSession2.Active || strings.TrimSpace(parseCurrentThreadPublicID) == "" {
			return
		}
		parseNav.Navigate(parseChatCanvasPath(parseCurrentThreadPublicID, parseSession2.ArtifactID))
	})

	parseBackToThread := ui.UseEvent(func() {
		if strings.TrimSpace(parseCurrentThreadPublicID) == "" {
			parseNav.Navigate(chatRouteRoot)
			return
		}
		parseNav.Navigate(parseChatThreadPath(parseCurrentThreadPublicID))
	})

	parseToggleConsole := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionCanvasToggleConsole})
	})

	clearConsole := ui.UseEvent(func() {
		parseApp.Dispatch(appAction{Type: appActionCanvasClearConsole})
	})

	parseChangeFocus := ui.UseEvent(func(parseE2 ui.Event) {
		parseSession3 := parseApp.Get().CanvasSession
		if !parseSession3.Active {
			return
		}
		parseRawIndex := parseEventDatasetValue(parseE2, dataCanvasFocus)
		parseIndex, parseErr3 := strconv.Atoi(parseRawIndex)
		if parseErr3 != nil || parseIndex < 0 || parseIndex >= len(parseSession3.FocusOptions) {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionSetCanvasFocus, CanvasFocus: parseSession3.FocusOptions[parseIndex]})
	})

	handleFocusDraft := ui.UseEvent(func(parseE3 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetCanvasFocusDraft, CanvasFocusDraft: parseE3.GetValue()})
	})

	applyFocusDraft := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionApplyCanvasFocusDraft})
	})

	resetOriginal := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionCanvasResetOriginal})
	})

	parseRevertLastPatch := ui.UseEvent(func() {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionCanvasRevertLastPatch})
	})

	parseCopyCurrentCode := ui.UseEvent(func() {
		parseCopyToClipboard(parseApp.Get().CanvasSession.CurrentSource)
	})

	parseStartSplitDrag := ui.UseEvent(func(parseE4 ui.Event) {
		if !parseApp.Get().CanvasSession.Active {
			return
		}
		parseBeginSplitDrag()
		parseE4.PreventDefault()
	})

	handleSplitKey := ui.UseEvent(func(parseE5 ui.Event) {
		parseKey := parseE5.JSValue().Get("key").String()
		parseRatio4 := parseApp.Get().CanvasSession.SplitRatio
		switch parseKey {
		case "ArrowLeft":
			parseRatio4 -= 0.04
		case "ArrowRight":
			parseRatio4 += 0.04
		default:
			return
		}
		if parseRatio4 < canvasSplitMin {
			parseRatio4 = canvasSplitMin
		}
		if parseRatio4 > canvasSplitMax {
			parseRatio4 = canvasSplitMax
		}
		parseApp.Dispatch(appAction{Type: appActionSetCanvasSplitRatio, CanvasSplitRatio: parseRatio4})
		parseE5.PreventDefault()
	})

	return canvasWorkspaceController{
		OpenFromMessage:  parseOpenFromMessage,
		Close:            parseCloseCanvas,
		Reload:           parseReloadCanvas,
		RefreshPreview:   parseRefreshPreview,
		ToggleOverlay:    parseToggleOverlay,
		OpenCanvasOnly:   parseOpenCanvasOnly,
		BackToThread:     parseBackToThread,
		ToggleConsole:    parseToggleConsole,
		ClearConsole:     clearConsole,
		ChangeFocus:      parseChangeFocus,
		HandleFocusDraft: handleFocusDraft,
		ApplyFocusDraft:  applyFocusDraft,
		ResetOriginal:    resetOriginal,
		RevertLastPatch:  parseRevertLastPatch,
		CopyCurrentCode:  parseCopyCurrentCode,
		StartSplitDrag:   parseStartSplitDrag,
		HandleSplitKey:   handleSplitKey,
	}
}

func canvasWorkspacePane(parseIntl i18n.Runtime, parseSession canvasSessionState, parseController canvasWorkspaceController, isCanvasOnly bool) ui.Node {
	if !parseSession.Active {
		return nil
	}
	parseDocument := buildCanvasRuntimeDocument(parseSession.CurrentSource, parseSession.SessionID, parseSession.ArtifactID, parseSession.LatestRenderedVersion)
	parseMenuButtonClass := "rounded-md border border-white/10 px-2.5 py-1 text-[11px] text-white/72 hover:bg-white/10 transition-colors whitespace-nowrap"
	return Div(
		ID(idCanvasWorkspace),
		Class("flex h-full min-h-0 min-w-0 flex-col border-l border-white/8 bg-[#101010]"),
		// content toolbar — filename dominant, all dev tools hidden behind a ··· icon
		Div(Class("flex items-center gap-2 border-b border-white/8 px-3 py-2"),
			If(isCanvasOnly,
				Button(
					Class(parseMenuButtonClass),
					OnClick(parseController.BackToThread),
					Attr("title", "Back to thread"),
					Text("← Thread"),
				),
			),
			Div(Class("min-w-0 flex-1 flex items-center gap-2 overflow-hidden"),
				Div(Class("truncate text-sm font-semibold text-white/92"), Text(parseSession.CurrentFileID)),
				If(parseSession.Dirty,
					Span(Class("shrink-0 rounded-full border border-[#19c37d]/25 bg-[#19c37d]/12 px-2 py-0.5 text-[10px] uppercase tracking-[0.12em] text-[#9af7d0]/72"), Text("edited")),
				),
			),
			// ··· icon — collapses all dev actions so the toolbar feels like a document bar
			Tag("details",
				Class("relative shrink-0"),
				Tag("summary",
					Class("list-none cursor-pointer rounded-md border border-white/10 px-2 py-1 text-[13px] leading-none text-white/50 hover:bg-white/10 hover:text-white/80 transition-colors select-none"),
					Attr("title", "Canvas tools"),
					Text("···"),
				),
				Div(Class("absolute right-0 top-full z-20 mt-1 flex flex-col gap-1 rounded-xl border border-white/10 bg-[#181818] p-2 shadow-xl"),
					Div(Class("mb-1 px-1 text-[10px] uppercase tracking-[0.18em] text-white/30"), Text("Canvas controls")),
					Button(Class(parseMenuButtonClass+" w-full text-left"), OnClick(parseController.Reload), Text("Reload")),
					Button(Class(parseMenuButtonClass+" w-full text-left"), OnClick(parseController.RefreshPreview), Text("Refresh preview")),
					Button(Class(parseMenuButtonClass+" w-full text-left"), OnClick(parseController.ToggleConsole), Text("Toggle console")),
					If(!isCanvasOnly,
						Button(Class(parseMenuButtonClass+" w-full text-left"), OnClick(parseController.ToggleOverlay), Text("Overlay mode")),
					),
					If(!isCanvasOnly,
						Button(Class(parseMenuButtonClass+" w-full text-left"), OnClick(parseController.OpenCanvasOnly), Text("Canvas only")),
					),
				),
			),
			// ✕ icon close — no label, stays visually minimal
			Button(
				Class("shrink-0 rounded-md border border-white/10 px-2 py-1 text-[13px] leading-none text-white/50 hover:bg-white/10 hover:text-white/80 transition-colors"),
				Attr("title", "Close canvas"),
				OnClick(parseController.Close),
				Text("✕"),
			),
		),
		// body: preview is the primary/wider column; editor sidebar is secondary
		Div(Class("grid min-h-0 flex-1 grid-cols-1 xl:grid-cols-[minmax(14rem,0.36fr)_minmax(0,1fr)]"),
			Div(Class("chat-scrollbar flex min-h-0 flex-col overflow-y-auto border-b border-white/8 xl:border-b-0 xl:border-r"),
				Div(Class("border-b border-white/8 px-3 py-2"),
					Div(Class("flex flex-wrap items-center gap-2"),
						Map(parseSession.FocusOptions, func(parseRegion canvasFocusRegion) ui.Node {
							parseIndex := canvasFocusIndex(parseSession.FocusOptions, parseRegion)
							parseActive := canvasFocusSameLineRange(parseRegion, parseSession.FocusedRegion)
							return Button(
								Class(ClassNames(
									"rounded-full border px-3 py-1.5 text-xs transition-colors",
									When(parseActive, "border-[#19c37d]/35 bg-[#19c37d]/12 text-[#d8fff1]"),
									When(!parseActive, "border-white/10 text-white/55 hover:bg-white/8 hover:text-white/85"),
								)),
								Data(dataCanvasFocus, strconv.Itoa(parseIndex)),
								OnClick(parseController.ChangeFocus),
								Text(canvasFocusPillLabel(parseRegion)),
							)
						}),
					),
					Div(Class("mt-3 flex flex-wrap items-center gap-2 text-xs text-white/45"),
						Span(Textf("file: %s", parseSession.CurrentFileID)),
						Span(Textf("focus: %s", canvasFocusDisplayLabel(parseSession.FocusedRegion))),
						Span(Textf("lines: %d-%d", parseSession.FocusedRegion.StartLine, parseSession.FocusedRegion.EndLine)),
					),
				),
				Div(Class("grid gap-2 px-3 py-3"),
					Div(Class("rounded-[1.2rem] border border-[#19c37d]/16 bg-[linear-gradient(180deg,rgba(17,34,28,0.72),rgba(10,19,16,0.78))] p-3"),
						Div(Class("mb-2 flex items-center justify-between gap-3"),
							Div(
								Class("text-xs uppercase tracking-[0.18em] text-[#9af7d0]/62"),
								Textf("Focused region: %s", canvasFocusDisplayLabel(parseSession.FocusedRegion)),
							),
							Div(Class("flex items-center gap-2"),
								Button(Class(parseMenuButtonClass), OnClick(parseController.ApplyFocusDraft), Text("Apply patch")),
								Button(Class(parseMenuButtonClass), OnClick(parseController.ResetOriginal), Text("Reset")),
								Button(Class(parseMenuButtonClass), OnClick(parseController.RevertLastPatch), Text("Revert")),
							),
						),
						Tag("textarea",
							ID(idCanvasFocusEditor),
							Class("chat-scrollbar min-h-[10rem] w-full resize-y rounded-2xl border border-white/8 bg-black/20 px-4 py-3 font-mono text-[0.92rem] leading-6 text-white/90 focus:outline-none"),
							Value(parseSession.FocusDraft),
							OnInput(parseController.HandleFocusDraft),
						),
						Div(Class("mt-3 flex flex-wrap items-center gap-2"),
							Button(Class(parseMenuButtonClass), OnClick(parseController.CopyCurrentCode), Text("Copy code")),
							Span(Class("text-xs text-white/40"), Textf("patches: %d", len(parseSession.PatchHistory))),
						),
					),
					Tag("details",
						Class("rounded-[1.25rem] border border-white/8 bg-black/16 p-3"),
						Tag("summary",
							Class("text-xs uppercase tracking-[0.18em] text-white/45"),
							Text("Current file"),
						),
						Div(Class("mt-2"),
							canvasSourceView(parseSession.CurrentSource, parseSession.FocusedRegion),
						),
					),
					If(len(parseSession.PatchHistory) > 0,
						Tag("details",
							Class("rounded-[1.25rem] border border-white/8 bg-black/16 p-3"),
							Tag("summary",
								Class("text-xs uppercase tracking-[0.18em] text-white/45"),
								Text("Patch history"),
							),
							Div(Class("mt-2 flex flex-col gap-2"),
								Map(parseSession.PatchHistory, func(parseRecord canvasPatchRecord) ui.Node {
									return Div(Class("rounded-2xl border border-white/8 bg-white/3 px-3 py-2"),
										Div(Class("text-sm text-white/85"), Text(parseRecord.Summary)),
										Div(Class("mt-1 text-xs text-white/42"), Textf("%s lines %d-%d", parseRecord.Type, parseRecord.StartLine, parseRecord.EndLine)),
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
					Div(Class("text-xs text-white/45"), Textf("render v%d", parseSession.LatestRenderedVersion)),
				),
				Div(Class("min-h-0 flex-1 p-2"),
					Tag("iframe",
						ID(idCanvasFrame),
						Class("h-full min-h-[34rem] w-full rounded-[1rem] border border-black/20 bg-white shadow-[0_24px_70px_rgba(0,0,0,0.28)]"),
						FromProps(Props{Raw: map[string]interface{}{
							"sandbox": "allow-scripts",
							"srcdoc":  parseDocument,
							"title":   parseIntl.T(chatI18nNamespace, "canvas.title"),
						}}),
					),
				),
			),
		),
		If(parseSession.ConsoleOpen,
			Div(
				ID(idCanvasConsole),
				Class("chat-scrollbar max-h-[10rem] overflow-y-auto border-t border-white/8 bg-black/36 px-3 py-2"),
				Div(Class("mb-3 flex items-center justify-between gap-3"),
					Div(Class("text-xs uppercase tracking-[0.18em] text-white/38"), Text("Console")),
					Button(Class(parseMenuButtonClass), OnClick(parseController.ClearConsole), Text("Clear logs")),
				),
				If(len(parseSession.ConsoleEntries) == 0,
					Div(Class("text-sm text-white/35"), Text("No console output yet.")),
				),
				Div(Class("flex flex-col gap-2"),
					Map(parseSession.ConsoleEntries, func(parseEntry canvasConsoleEntry) ui.Node {
						return Div(Class("rounded-2xl border border-white/8 bg-white/3 px-3 py-2 font-mono text-xs leading-5"),
							Div(Class("flex items-center gap-2"),
								Span(Class("uppercase tracking-[0.18em] text-white/38"), Text(parseEntry.Level)),
								Span(Class("text-white/25"), Textf("render %d", parseEntry.RenderVersion)),
							),
							Div(Class("mt-1 whitespace-pre-wrap text-white/82"), Text(parseEntry.Message)),
							If(strings.TrimSpace(parseEntry.Detail) != "",
								Div(Class("mt-1 whitespace-pre-wrap text-[#ffbdbd]/78"), Text(parseEntry.Detail)),
							),
						)
					}),
				),
			),
		),
	)
}

func canvasSourceView(parseSource string, parseFocus canvasFocusRegion) ui.Node {
	parseLines := strings.Split(strings.ReplaceAll(parseSource, "\r\n", "\n"), "\n")
	if len(parseLines) == 0 {
		parseLines = []string{""}
	}
	parseRows := make([]ui.Node, 0, len(parseLines))
	for parseIdx, parseLine := range parseLines {
		parseLineNumber := parseIdx + 1
		isParseHighlighted := parseLineNumber >= parseFocus.StartLine && parseLineNumber <= parseFocus.EndLine
		parseRows = append(parseRows, Div(
			Class(ClassNames(
				"grid grid-cols-[3.5rem_minmax(0,1fr)] gap-3 px-3 py-0.5",
				When(isParseHighlighted, "bg-[#19c37d]/10"),
			)),
			Span(Class("select-none text-right text-white/26"), Text(strconv.Itoa(parseLineNumber))),
			Span(Class("whitespace-pre-wrap break-words text-white/84"), Text(parseLine)),
		))
	}
	parseRowChildren := make([]interface{}, 0, len(parseRows)+1)
	parseRowChildren = append(parseRowChildren, Class("min-w-full font-mono text-[0.82rem] leading-6"))
	for _, parseRow := range parseRows {
		parseRowChildren = append(parseRowChildren, parseRow)
	}
	return Div(Class("chat-scrollbar max-h-[22rem] overflow-auto rounded-2xl border border-white/8 bg-[#0d0d0d]"),
		Div(parseRowChildren...),
	)
}

func canvasFocusIndex(parseOptions []canvasFocusRegion, parseTarget canvasFocusRegion) int {
	for parseIdx, parseOption := range parseOptions {
		if canvasFocusSameLineRange(parseOption, parseTarget) {
			return parseIdx
		}
	}
	return 0
}

func canvasFocusSameLineRange(parseA, parseB canvasFocusRegion) bool {
	return parseA.StartLine == parseB.StartLine && parseA.EndLine == parseB.EndLine && parseA.Label == parseB.Label
}

func canvasFocusPillLabel(parseRegion canvasFocusRegion) string {
	parseLabel := strings.TrimSpace(parseRegion.Label)
	if parseLabel != "" {
		return parseLabel
	}
	if parseRegion.Kind == "file" {
		return "Full file"
	}
	return fmt.Sprintf("%s %d-%d", parseRegion.Kind, parseRegion.StartLine, parseRegion.EndLine)
}
