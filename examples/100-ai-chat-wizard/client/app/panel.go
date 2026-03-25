//go:build js && wasm

package app

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func mainPanel(msgs []message, isStreaming bool, useMarkdownFallback bool, inputVal string, onInput, onKey, onSend ui.Handler,
	editIdx int, editText string, startEdit, cancelEdit, handleEditChange, submitEdit, handleEditKey, doFork, openCanvas, toggleThoughtSection ui.Handler,
	modelOptions []modelOption, defaultModelID string, threadCostSummary threadCostSummary,
	curModel string, setProvider, setModel ui.Handler, thinkingEnabled bool, thinkingEffort string, thinkingSupported bool, setThinkingMode ui.Handler, userInitials string, sidebarOpen bool, onToggleSidebar ui.Handler, expandedThoughtSections map[string]bool, ttsAudio ttsAudioController, canvasSession canvasSessionState, canvas canvasWorkspaceController) ui.Node {
	intl := i18n.UseI18n()
	providerOptions := providerOptionsForModels(modelOptions)
	activeProvider := providerForModel(curModel, modelOptions, defaultModelID)
	visibleModelOptions := modelsForProvider(modelOptions, activeProvider.ID)
	currentThinkingMode := "off"
	if thinkingEnabled && thinkingSupported {
		currentThinkingMode = normalizeSelectedThinkingEffort(thinkingEffort)
	}
	splitActive := canvasSession.Active && canvasSession.LayoutMode == canvasLayoutSplit
	leftStyle := map[string]string{}
	if splitActive {
		leftStyle["flex"] = fmt.Sprintf("0 0 %.2f%%", canvasSession.SplitRatio*100)
		leftStyle["minWidth"] = "20%"
		leftStyle["maxWidth"] = "80%"
	}
	return Div(
		Class("flex flex-col flex-1 min-w-0 h-full"),
		renderMobileControlBar(intl, providerOptions, activeProvider, visibleModelOptions, curModel, currentThinkingMode, isStreaming, thinkingSupported, setProvider, setModel, setThinkingMode),
		renderDesktopControlBar(intl, providerOptions, activeProvider, visibleModelOptions, curModel, currentThinkingMode, isStreaming, thinkingSupported, sidebarOpen, onToggleSidebar, setProvider, setModel, setThinkingMode),
		Div(Class("flex flex-1 min-h-0 min-w-0"),
			Div(
				FromProps(Props{Style: leftStyle}),
				Class("flex min-h-0 min-w-0 flex-1 flex-col"),
				messageList(messageListProps{
					Intl:                    intl,
					Messages:                msgs,
					IsStreaming:             isStreaming,
					UseMarkdownFallback:     useMarkdownFallback,
					EditIdx:                 editIdx,
					EditText:                editText,
					StartEdit:               startEdit,
					CancelEdit:              cancelEdit,
					HandleEditChange:        handleEditChange,
					SubmitEdit:              submitEdit,
					HandleEditKey:           handleEditKey,
					OnFork:                  doFork,
					OnOpenCanvas:            openCanvas,
					ToggleThoughtSection:    toggleThoughtSection,
					ModelOptions:            modelOptions,
					DefaultModelID:          defaultModelID,
					ThreadCostSummary:       threadCostSummary,
					UserInitials:            userInitials,
					ExpandedThoughtSections: expandedThoughtSections,
					TTSAudio:                ttsAudio,
				}),
				inputArea(composerProps{
					Intl:              intl,
					Value:             inputVal,
					Disabled:          isStreaming,
					ThreadCostSummary: threadCostSummary,
					OnInput:           onInput,
					OnKeyDown:         onKey,
					OnSend:            onSend,
				}),
			),
			If(splitActive,
				Div(
					ID(idCanvasSplitHandle),
					Class("hidden xl:flex w-3 shrink-0 cursor-col-resize items-center justify-center bg-[linear-gradient(180deg,rgba(255,255,255,0.03),rgba(255,255,255,0.01))] hover:bg-white/8 transition-colors"),
					TabIndex(0),
					Role("separator"),
					FromProps(Props{Aria: map[string]string{
						"orientation": "vertical",
						"label":       "Resize chat and canvas panes",
					}}),
					OnClick(canvas.StartSplitDrag),
					OnKeyDown(canvas.HandleSplitKey),
					Div(Class("h-16 w-1 rounded-full bg-white/10")),
				),
			),
			If(splitActive,
				Div(
					Class("min-h-0 min-w-0 flex-1"),
					FromProps(Props{Style: map[string]string{
						"flex":     fmt.Sprintf("0 0 %.2f%%", (1-canvasSession.SplitRatio)*100),
						"minWidth": "20%",
						"maxWidth": "80%",
					}}),
					canvasWorkspacePane(intl, canvasSession, canvas, false),
				),
			),
		),
	)
}

func renderMobileControlBar(intl i18n.Runtime, providerOptions []providerOption, activeProvider providerOption, visibleModelOptions []modelOption, curModel, currentThinkingMode string, isStreaming, thinkingSupported bool, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(Class("md:hidden flex flex-col border-b border-white/10 bg-[#212121]/70 backdrop-blur-md sticky top-0 z-10"),
		Div(Class("flex items-center gap-2 px-4 py-2"),
			Div(Class("h-7 w-7 rounded-full bg-gradient-to-br from-[#19c37d] to-[#0ea47e] flex items-center justify-center text-sm"),
				Text(assistantBadgeText),
			),
			Span(Class("font-semibold text-sm flex-1 min-w-0 truncate"), Text(appBrandName)),
			Span(Class("text-[10px] text-white/30 uppercase tracking-[0.18em] shrink-0"), Text(appVersion)),
		),
		renderCompactControlGroup(intl.T(chatI18nNamespace, "controls.provider"), isStreaming, providerOptions, func(option providerOption) ui.Node {
			return Button(
				Class(controlChipClass(activeProvider.ID == option.ID, isStreaming, false, true)),
				DisabledIf(isStreaming),
				OnClick(setProvider),
				Data(dataProvider, option.ID),
				Text(option.Label),
			)
		}),
		renderCompactControlGroup(intl.T(chatI18nNamespace, "controls.model"), isStreaming, visibleModelOptions, func(option modelOption) ui.Node {
			return Button(
				Class(controlChipClass(curModel == option.ID, isStreaming, false, false)),
				DisabledIf(isStreaming),
				OnClick(setModel),
				Data(dataModel, option.ID),
				Text(option.Label),
			)
		}),
		renderCompactControlGroup(intl.T(chatI18nNamespace, "controls.intelligence"), isStreaming || !thinkingSupported, availableThinkingEfforts, func(option thinkingEffortOption) ui.Node {
			return Button(
				Class(controlChipClass(currentThinkingMode == option.ID, isStreaming, !thinkingSupported, true)),
				DisabledIf(isStreaming || !thinkingSupported),
				OnClick(setThinkingMode),
				Data(dataThinkingEffort, option.ID),
				Text(thinkingEffortLabel(intl, option.ID)),
			)
		}),
	)
}

func renderDesktopControlBar(intl i18n.Runtime, providerOptions []providerOption, activeProvider providerOption, visibleModelOptions []modelOption, curModel, currentThinkingMode string, isStreaming, thinkingSupported, sidebarOpen bool, onToggleSidebar, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(Class("hidden md:flex items-center gap-2 px-3 py-2 border-b border-white/5 bg-[#212121]/70 backdrop-blur-md sticky top-0 z-10"),
		Button(
			Class(ClassNames(
				"p-1.5 rounded-lg text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out",
				When(sidebarOpen, "pointer-events-none opacity-0 -translate-x-1 scale-95"),
				When(!sidebarOpen, "opacity-100 translate-x-0 scale-100"),
			)),
			OnClick(onToggleSidebar),
			sidebarToggleIcon(!sidebarOpen),
		),
		renderDesktopControlGroup(intl.T(chatI18nNamespace, "controls.provider"), isStreaming, providerOptions, func(option providerOption) ui.Node {
			return Button(
				Class(controlChipClass(activeProvider.ID == option.ID, isStreaming, false, true)),
				DisabledIf(isStreaming),
				OnClick(setProvider),
				Data(dataProvider, option.ID),
				Text(option.Label),
			)
		}),
		renderDesktopControlGroup(intl.T(chatI18nNamespace, "controls.model"), isStreaming, visibleModelOptions, func(option modelOption) ui.Node {
			return Button(
				Class(controlChipClass(curModel == option.ID, isStreaming, false, false)),
				DisabledIf(isStreaming),
				OnClick(setModel),
				Data(dataModel, option.ID),
				Text(option.Label),
			)
		}),
		renderDesktopControlGroup(intl.T(chatI18nNamespace, "controls.intelligence"), isStreaming || !thinkingSupported, availableThinkingEfforts, func(option thinkingEffortOption) ui.Node {
			return Button(
				Class(controlChipClass(currentThinkingMode == option.ID, isStreaming, !thinkingSupported, true)),
				DisabledIf(isStreaming || !thinkingSupported),
				OnClick(setThinkingMode),
				Data(dataThinkingEffort, option.ID),
				Text(thinkingEffortLabel(intl, option.ID)),
			)
		}),
	)
}

func renderCompactControlGroup[T any](label string, disabled bool, options []T, render func(T) ui.Node) ui.Node {
	return Div(Class("control-group-mobile px-3 pb-2"),
		Div(Class("control-group-label px-1 pb-1"), Text(label)),
		Div(Class(ClassNames(
			"control-group-shell flex flex-wrap items-center gap-1 rounded-2xl px-1.5 py-1.5",
			When(disabled, "opacity-70"),
		)),
			Map(options, render),
		),
	)
}

func renderDesktopControlGroup[T any](label string, disabled bool, options []T, render func(T) ui.Node) ui.Node {
	return Div(Class(ClassNames(
		"control-group-card flex items-center gap-2 rounded-2xl px-2 py-1.5",
		When(disabled, "opacity-70"),
	)),
		Span(Class("control-group-label shrink-0"), Text(label)),
		Div(Class("control-group-shell flex items-center gap-1 rounded-full px-1 py-1"),
			Map(options, render),
		),
	)
}

func controlChipClass(active, streaming, unsupported, accent bool) string {
	return ClassNames(
		"control-chip px-3 py-1.5 rounded-full text-xs font-medium",
		When(active && accent, "control-chip-active-accent"),
		When(active && !accent, "control-chip-active-default"),
		When(!active, "control-chip-idle"),
		When(streaming || unsupported, "cursor-not-allowed opacity-50"),
	)
}
