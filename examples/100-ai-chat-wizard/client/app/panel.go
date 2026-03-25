//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func mainPanel(msgs []message, isStreaming bool, useMarkdownFallback bool, inputVal string, onInput, onKey, onSend ui.Handler,
	editIdx int, editText string, startEdit, cancelEdit, handleEditChange, submitEdit, handleEditKey, doFork, openCanvas, toggleThoughtSection ui.Handler,
	modelOptions []modelOption, defaultModelID string, threadCostSummary threadCostSummary,
	curModel string, setProvider, setModel ui.Handler, thinkingEnabled bool, thinkingEffort string, thinkingSupported bool, setThinkingMode ui.Handler, userInitials string, sidebarOpen bool, onToggleSidebar ui.Handler, expandedThoughtSections map[string]bool, ttsAudio ttsAudioController, onSpeechUpgrade func(), scrollMemory threadScrollMemory, canvasSession canvasSessionState, canvas canvasWorkspaceController) ui.Node {
	intl := i18n.UseI18n()
	scrollToBottom := ui.UseEvent(func() {
		scrollMemory.ScrollToBottom()
	})
	currentThinkingMode := "off"
	if thinkingEnabled && thinkingSupported {
		currentThinkingMode = normalizeSelectedThinkingEffort(thinkingEffort)
	}
	requiredCapability := ""
	if currentThinkingMode != "off" {
		requiredCapability = "thinking"
	}
	capabilityScopedModels := filterModelsByCapability(modelOptions, requiredCapability)
	providerOptions := providerOptionsForModels(capabilityScopedModels)
	activeProvider := providerForModel(curModel, capabilityScopedModels, defaultModelID)
	if activeProvider.ID == "" && len(providerOptions) > 0 {
		activeProvider = providerOptions[0]
	}
	visibleModelOptions := modelsForProvider(capabilityScopedModels, activeProvider.ID)
	displayModel := curModel
	if len(visibleModelOptions) > 0 {
		displayModel = normalizeSelectedModelID(curModel, visibleModelOptions, defaultModelForProvider(activeProvider.ID, capabilityScopedModels, defaultModelID))
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
		renderMobileControlBar(intl, providerOptions, activeProvider, visibleModelOptions, displayModel, currentThinkingMode, requiredCapability, isStreaming, thinkingSupported, setProvider, setModel, setThinkingMode),
		renderDesktopControlBar(intl, providerOptions, activeProvider, visibleModelOptions, displayModel, currentThinkingMode, requiredCapability, isStreaming, thinkingSupported, sidebarOpen, onToggleSidebar, setProvider, setModel, setThinkingMode),
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
					OnSpeechUpgrade:         onSpeechUpgrade,
					ShowScrollToBottom:      scrollMemory.ShowScrollToBottom(),
					ScrollToBottom:          scrollToBottom,
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

func renderMobileControlBar(intl i18n.Runtime, providerOptions []providerOption, activeProvider providerOption, visibleModelOptions []modelOption, curModel, currentThinkingMode, requiredCapability string, isStreaming, thinkingSupported bool, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(Class("md:hidden flex flex-col border-b border-white/10 bg-[#212121]/70 backdrop-blur-md sticky top-0 z-10"),
		Div(Class("flex items-center gap-2 px-4 py-2"),
			Div(Class("h-7 w-7 rounded-full bg-gradient-to-br from-[#19c37d] to-[#0ea47e] flex items-center justify-center text-sm"),
				Text(assistantBadgeText),
			),
			Span(Class("font-semibold text-sm flex-1 min-w-0 truncate"), Text(appBrandName)),
			Span(Class("text-[10px] text-white/30 uppercase tracking-[0.18em] shrink-0"), Text(appVersion)),
		),
		Div(Class("grid grid-cols-2 gap-2 px-3 pb-3"),
			renderToolbarSelect(
				"col-span-1",
				intl.T(chatI18nNamespace, "controls.provider"),
				activeProvider.ID,
				isStreaming,
				setProvider,
				Map(providerOptions, func(option providerOption) ui.Node {
					return renderToolbarOption(option.ID, option.Label)
				}),
			),
			renderToolbarSelect(
				"col-span-2",
				intl.T(chatI18nNamespace, "controls.model"),
				curModel,
				isStreaming,
				setModel,
				Map(visibleModelOptions, func(option modelOption) ui.Node {
					return renderToolbarOption(option.ID, toolbarModelLabel(option))
				}),
			),
			renderToolbarSelect(
				"col-span-1",
				intl.T(chatI18nNamespace, "controls.intelligence"),
				currentThinkingMode,
				isStreaming || !thinkingSupported,
				setThinkingMode,
				Map(availableThinkingEfforts, func(option thinkingEffortOption) ui.Node {
					return renderToolbarOption(option.ID, thinkingEffortLabel(intl, option.ID))
				}),
			),
			If(!thinkingSupported,
				P(Class("col-span-2 px-1 text-[11px] leading-relaxed text-white/35"), Text(intl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
			If(requiredCapability == "thinking",
				P(Class("col-span-2 px-1 text-[11px] leading-relaxed text-[#8df5cf]"), Text(intl.T(chatI18nNamespace, "controls.capabilityThinkingFilter"))),
			),
		),
	)
}

func renderDesktopControlBar(intl i18n.Runtime, providerOptions []providerOption, activeProvider providerOption, visibleModelOptions []modelOption, curModel, currentThinkingMode, requiredCapability string, isStreaming, thinkingSupported, sidebarOpen bool, onToggleSidebar, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
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
		Div(Class("flex min-w-0 flex-1 items-center gap-2"),
			renderToolbarSelect(
				"flex-[0_0_12rem]",
				intl.T(chatI18nNamespace, "controls.provider"),
				activeProvider.ID,
				isStreaming,
				setProvider,
				Map(providerOptions, func(option providerOption) ui.Node {
					return renderToolbarOption(option.ID, option.Label)
				}),
			),
			renderToolbarSelect(
				"min-w-0 flex-[1_1_24rem]",
				intl.T(chatI18nNamespace, "controls.model"),
				curModel,
				isStreaming,
				setModel,
				Map(visibleModelOptions, func(option modelOption) ui.Node {
					return renderToolbarOption(option.ID, toolbarModelLabel(option))
				}),
			),
			renderToolbarSelect(
				"flex-[0_0_12rem]",
				intl.T(chatI18nNamespace, "controls.intelligence"),
				currentThinkingMode,
				isStreaming || !thinkingSupported,
				setThinkingMode,
				Map(availableThinkingEfforts, func(option thinkingEffortOption) ui.Node {
					return renderToolbarOption(option.ID, thinkingEffortLabel(intl, option.ID))
				}),
			),
			If(!thinkingSupported,
				Span(Class("shrink-0 text-[11px] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
			If(requiredCapability == "thinking",
				Span(Class("shrink-0 text-[11px] text-[#8df5cf]"), Text(intl.T(chatI18nNamespace, "controls.capabilityThinkingFilter"))),
			),
		),
	)
}

func renderToolbarSelect(containerClass, label, value string, disabled bool, onChange ui.Handler, options []ui.Node) ui.Node {
	return Label(Class(ClassNames(
		"flex min-w-0 items-center gap-2 rounded-2xl border border-white/8 bg-white/[0.03] px-3 py-2",
		containerClass,
	)),
		Span(Class("control-group-label shrink-0 min-w-[5.4rem]"), Text(label)),
		Select(
			Value(value),
			DisabledIf(disabled),
			OnChange(onChange),
			Class(ClassNames(
				"toolbar-select h-9 min-w-0 flex-1 rounded-xl border border-white/10 bg-[#3a3a3a] px-3 text-sm text-white outline-none",
				When(disabled, "cursor-not-allowed opacity-60"),
			)),
			options,
		),
	)
}

func renderToolbarOption(value, label string) ui.Node {
	return Option(
		Class("toolbar-select-option"),
		Value(value),
		Text(label),
	)
}

func toolbarModelLabel(option modelOption) string {
	if strings.TrimSpace(option.Note) == "" {
		return option.Label
	}
	return option.Label + " \u00b7 " + option.Note
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
