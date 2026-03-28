//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parseMainPanel(parseMsgs []message, isStreaming bool, isUseMarkdownFallback bool, parseInputVal string, parseOnInput, parseOnKey, parseOnSend ui.Handler,
	parseEditIdx int, parseEditText string, parseStartEdit, parseCancelEdit, handleEditChange, parseSubmitEdit, handleEditKey, parseDoFork, parseOpenCanvas, parseToggleThoughtSection ui.Handler,
	parseModelOptions []modelOption, parseDefaultModelID string, parseThreadCostSummary threadCostSummary, parseAccountCostSummary accountCostSummary,
	parseCurModel string, setProvider, setModel ui.Handler, isThinkingEnabled bool, parseThinkingEffort string, isThinkingSupported bool, setThinkingMode ui.Handler, parseUserInitials string, isSidebarOpen bool, parseOnToggleSidebar ui.Handler, parseExpandedThoughtSections map[string]bool, parseThoughtCacheByMessage map[int]renderWorkerThoughtCacheEntry, parseCanvasCacheByMessage map[int]renderWorkerCanvasCacheEntry, parseTtsAudio ttsAudioController, parseOnSpeechUpgrade func(), parseScrollMemory threadScrollMemory, parseCanvasSession canvasSessionState, parseCanvas canvasWorkspaceController) ui.Node {
	parseIntl := i18n.UseI18n()
	parseScrollToBottom := ui.UseEvent(func() {
		parseScrollMemory.ParseScrollToBottom()
	})
	parseCurrentThinkingMode := "off"
	if isThinkingEnabled && isThinkingSupported {
		parseCurrentThinkingMode = parseNormalizeSelectedThinkingEffort(parseThinkingEffort)
	}
	parseRequiredCapability := ""
	if parseCurrentThinkingMode != "off" {
		parseRequiredCapability = "thinking"
	}
	parseCapabilityScopedModels := filterModelsByCapability(parseModelOptions, parseRequiredCapability)
	parseProviderOptions := parseProviderOptionsForModels(parseCapabilityScopedModels)
	parseActiveProvider := parseProviderForModel(parseCurModel, parseCapabilityScopedModels, parseDefaultModelID)
	if parseActiveProvider.ID == "" && len(parseProviderOptions) > 0 {
		parseActiveProvider = parseProviderOptions[0]
	}
	parseVisibleModelOptions := parseModelsForProvider(parseCapabilityScopedModels, parseActiveProvider.ID)
	parseDisplayModel := parseCurModel
	if len(parseVisibleModelOptions) > 0 {
		parseDisplayModel = parseNormalizeSelectedModelID(parseCurModel, parseVisibleModelOptions, parseDefaultModelForProvider(parseActiveProvider.ID, parseCapabilityScopedModels, parseDefaultModelID))
	}
	isParseSplitActive := parseCanvasSession.Active && parseCanvasSession.LayoutMode == canvasLayoutSplit
	parseLeftStyle := map[string]string{}
	if isParseSplitActive {
		parseLeftStyle["flex"] = fmt.Sprintf("0 0 %.2f%%", parseCanvasSession.SplitRatio*100)
		parseLeftStyle["minWidth"] = "20%"
		parseLeftStyle["maxWidth"] = "80%"
	}
	return Div(
		Class("flex flex-col flex-1 min-w-0 h-full"),
		renderMobileControlBar(parseIntl, parseProviderOptions, parseActiveProvider, parseVisibleModelOptions, parseDisplayModel, parseCurrentThinkingMode, parseRequiredCapability, isStreaming, isThinkingSupported, setProvider, setModel, setThinkingMode),
		renderDesktopControlBar(parseIntl, parseProviderOptions, parseActiveProvider, parseVisibleModelOptions, parseDisplayModel, parseCurrentThinkingMode, parseRequiredCapability, isStreaming, isThinkingSupported, isSidebarOpen, parseOnToggleSidebar, setProvider, setModel, setThinkingMode),
		Div(Class("flex flex-1 min-h-0 min-w-0"),
			Div(
				FromProps(Props{Style: parseLeftStyle}),
				Class("flex min-h-0 min-w-0 flex-1 flex-col"),
				parseMessageList(messageListProps{
					Intl:                    parseIntl,
					Messages:                parseMsgs,
					IsStreaming:             isStreaming,
					UseMarkdownFallback:     isUseMarkdownFallback,
					EditIdx:                 parseEditIdx,
					EditText:                parseEditText,
					StartEdit:               parseStartEdit,
					CancelEdit:              parseCancelEdit,
					HandleEditChange:        handleEditChange,
					SubmitEdit:              parseSubmitEdit,
					HandleEditKey:           handleEditKey,
					OnFork:                  parseDoFork,
					OnOpenCanvas:            parseOpenCanvas,
					ToggleThoughtSection:    parseToggleThoughtSection,
					ModelOptions:            parseModelOptions,
					DefaultModelID:          parseDefaultModelID,
					ThreadCostSummary:       parseThreadCostSummary,
					UserInitials:            parseUserInitials,
					ExpandedThoughtSections: parseExpandedThoughtSections,
					ThoughtCacheByMessage:   parseThoughtCacheByMessage,
					CanvasCacheByMessage:    parseCanvasCacheByMessage,
					TTSAudio:                parseTtsAudio,
					OnSpeechUpgrade:         parseOnSpeechUpgrade,
					ShowScrollToBottom:      parseScrollMemory.ParseShowScrollToBottom(),
					ScrollToBottom:          parseScrollToBottom,
				}),
				parseInputArea(composerProps{
					Intl:               parseIntl,
					Value:              parseInputVal,
					Disabled:           isStreaming,
					ThreadCostSummary:  parseThreadCostSummary,
					AccountCostSummary: parseAccountCostSummary,
					OnInput:            parseOnInput,
					OnKeyDown:          parseOnKey,
					OnSend:             parseOnSend,
				}),
			),
			If(isParseSplitActive,
				Div(
					ID(idCanvasSplitHandle),
					Class("hidden xl:flex w-3 shrink-0 cursor-col-resize items-center justify-center bg-[linear-gradient(180deg,rgba(255,255,255,0.03),rgba(255,255,255,0.01))] hover:bg-white/8 transition-colors"),
					TabIndex(0),
					Role("separator"),
					FromProps(Props{Aria: map[string]string{
						"orientation": "vertical",
						"label":       "Resize chat and canvas panes",
					}}),
					OnClick(parseCanvas.StartSplitDrag),
					OnKeyDown(parseCanvas.HandleSplitKey),
					Div(Class("h-16 w-1 rounded-full bg-white/10")),
				),
			),
			If(isParseSplitActive,
				Div(
					Class("min-h-0 min-w-0 flex-1"),
					FromProps(Props{Style: map[string]string{
						"flex":     fmt.Sprintf("0 0 %.2f%%", (1-parseCanvasSession.SplitRatio)*100),
						"minWidth": "20%",
						"maxWidth": "80%",
					}}),
					canvasWorkspacePane(parseIntl, parseCanvasSession, parseCanvas, false),
				),
			),
		),
	)
}

func renderMobileControlBar(parseIntl i18n.Runtime, parseProviderOptions []providerOption, parseActiveProvider providerOption, parseVisibleModelOptions []modelOption, parseCurModel, parseCurrentThinkingMode, parseRequiredCapability string, isStreaming, isThinkingSupported bool, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(Class("md:hidden flex flex-col border-b border-white/10 bg-[#212121]/70 backdrop-blur-md sticky top-0 z-10"),
		Div(Class("flex items-center gap-2 px-4 py-2"),
			Img(
				Src(brandChatIconURL),
				Attr("alt", appBrandName),
				Class("h-7 w-7 shrink-0 rounded-full object-cover"),
			),
			Span(Class("font-semibold text-sm flex-1 min-w-0 truncate"), Text(appBrandName)),
			Span(Class("text-[10px] text-white/30 uppercase tracking-[0.18em] shrink-0"), Text(appVersion)),
		),
		Div(Class("grid grid-cols-2 gap-2 px-3 pb-3"),
			renderToolbarSelect(
				"col-span-1",
				parseIntl.T(chatI18nNamespace, "controls.provider"),
				parseActiveProvider.ID,
				isStreaming,
				setProvider,
				Map(parseProviderOptions, func(parseOption providerOption) ui.Node {
					return renderToolbarOption(parseOption.ID, parseOption.Label)
				}),
			),
			renderToolbarSelect(
				"col-span-2",
				parseIntl.T(chatI18nNamespace, "controls.model"),
				parseCurModel,
				isStreaming,
				setModel,
				Map(parseVisibleModelOptions, func(parseOption2 modelOption) ui.Node {
					return renderToolbarOption(parseOption2.ID, parseToolbarModelLabel(parseOption2))
				}),
			),
			renderToolbarSelect(
				"col-span-1",
				parseIntl.T(chatI18nNamespace, "controls.intelligence"),
				parseCurrentThinkingMode,
				isStreaming || !isThinkingSupported,
				setThinkingMode,
				Map(availableThinkingEfforts, func(parseOption3 thinkingEffortOption) ui.Node {
					return renderToolbarOption(parseOption3.ID, parseThinkingEffortLabel(parseIntl, parseOption3.ID))
				}),
			),
			If(!isThinkingSupported,
				P(Class("col-span-2 px-1 text-[11px] leading-relaxed text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
			If(parseRequiredCapability == "thinking",
				P(Class("col-span-2 px-1 text-[11px] leading-relaxed text-[#8df5cf]"), Text(parseIntl.T(chatI18nNamespace, "controls.capabilityThinkingFilter"))),
			),
		),
	)
}

func renderDesktopControlBar(parseIntl i18n.Runtime, parseProviderOptions []providerOption, parseActiveProvider providerOption, parseVisibleModelOptions []modelOption, parseCurModel, parseCurrentThinkingMode, parseRequiredCapability string, isStreaming, isThinkingSupported, isSidebarOpen bool, parseOnToggleSidebar, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(Class("hidden md:flex items-center gap-2 px-3 py-1 border-b border-white/5 bg-[#212121]/70 backdrop-blur-md sticky top-0 z-10"),
		Button(
			Class(ClassNames(
				"p-1.5 rounded-lg text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out",
				When(isSidebarOpen, "pointer-events-none opacity-0 -translate-x-1 scale-95"),
				When(!isSidebarOpen, "opacity-100 translate-x-0 scale-100"),
			)),
			OnClick(parseOnToggleSidebar),
			parseSidebarToggleIcon(!isSidebarOpen),
		),
		Div(Class("flex min-w-0 flex-1 items-center gap-2"),
			renderToolbarSelect(
				"flex-[0_0_12rem]",
				parseIntl.T(chatI18nNamespace, "controls.provider"),
				parseActiveProvider.ID,
				isStreaming,
				setProvider,
				Map(parseProviderOptions, func(parseOption providerOption) ui.Node {
					return renderToolbarOption(parseOption.ID, parseOption.Label)
				}),
			),
			renderToolbarSelect(
				"min-w-0 flex-[1_1_24rem]",
				parseIntl.T(chatI18nNamespace, "controls.model"),
				parseCurModel,
				isStreaming,
				setModel,
				Map(parseVisibleModelOptions, func(parseOption2 modelOption) ui.Node {
					return renderToolbarOption(parseOption2.ID, parseToolbarModelLabel(parseOption2))
				}),
			),
			renderToolbarSelect(
				"flex-[0_0_12rem]",
				parseIntl.T(chatI18nNamespace, "controls.intelligence"),
				parseCurrentThinkingMode,
				isStreaming || !isThinkingSupported,
				setThinkingMode,
				Map(availableThinkingEfforts, func(parseOption3 thinkingEffortOption) ui.Node {
					return renderToolbarOption(parseOption3.ID, parseThinkingEffortLabel(parseIntl, parseOption3.ID))
				}),
			),
			If(!isThinkingSupported,
				Span(Class("shrink-0 text-[11px] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
			If(parseRequiredCapability == "thinking",
				Span(Class("shrink-0 text-[11px] text-[#8df5cf]"), Text(parseIntl.T(chatI18nNamespace, "controls.capabilityThinkingFilter"))),
			),
		),
	)
}

func renderToolbarSelect(parseContainerClass, parseLabel, parseValue string, isDisabled bool, parseOnChange ui.Handler, parseOptions []ui.Node) ui.Node {
	return Label(Class(ClassNames(
		"flex min-w-0 items-center gap-2 rounded-xl border border-white/5 bg-white/[0.02] px-2.5 py-1",
		parseContainerClass,
	)),
		Span(Class("control-group-label shrink-0 min-w-[5.4rem]"), Text(parseLabel)),
		Select(
			Value(parseValue),
			DisabledIf(isDisabled),
			OnChange(parseOnChange),
			Class(ClassNames(
				"toolbar-select h-7 min-w-0 flex-1 rounded-lg border border-white/8 bg-[#2f2f2f] px-2 text-sm text-white/80 outline-none",
				When(isDisabled, "cursor-not-allowed opacity-60"),
			)),
			parseOptions,
		),
	)
}

func renderToolbarOption(parseValue, parseLabel string) ui.Node {
	return Option(
		Class("toolbar-select-option"),
		Value(parseValue),
		Text(parseLabel),
	)
}

func parseToolbarModelLabel(parseOption modelOption) string {
	if strings.TrimSpace(parseOption.Note) == "" {
		return parseOption.Label
	}
	return parseOption.Label + " \u00b7 " + parseOption.Note
}
