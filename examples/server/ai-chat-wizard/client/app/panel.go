//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parseMainPanel(parseMsgs []message, isStreaming bool, isUseMarkdownFallback bool, parseInputVal string, parseOnInput, parseOnKey, parseOnSend, parseApplyStarterPrompt ui.Handler, isParseCanAccessAdmin bool, parseOpenAdminEntry ui.Handler,
	parseEditIdx int, parseEditText string, parseStartEdit, parseCancelEdit, handleEditChange, parseSubmitEdit, handleEditKey, parseDoFork, parseOpenCanvas, parseToggleThoughtSection ui.Handler,
	parseModelOptions []modelOption, parseDefaultModelID string, parseThreadCostSummary threadCostSummary, parseAccountCostSummary accountCostSummary,
	parseCurModel string, setProvider, setModel ui.Handler, isThinkingEnabled bool, parseThinkingEffort string, isThinkingSupported bool, setThinkingMode ui.Handler, parseUserInitials string, isSidebarOpen bool, parseOnToggleSidebar ui.Handler, parseExpandedThoughtSections map[string]bool, parseThoughtCacheByMessage map[int]renderWorkerThoughtCacheEntry, parseCanvasCacheByMessage map[int]renderWorkerCanvasCacheEntry, parseTtsAudio ttsAudioController, parseOnSpeechUpgrade func(), parseScrollMemory threadScrollMemory, parseCanvasSession canvasSessionState, parseCanvas canvasWorkspaceController, parseHandleSelectionMouse ui.Handler) ui.Node {
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
	parseJourneyState := parseBuildChatJourneyState(parseMsgs, isStreaming)
	parseCueTone := ""
	parseCueTitle := ""
	parseCueBody := ""
	parseLatestAssistantText := ""
	for parseIndex := len(parseMsgs) - 1; parseIndex >= 0; parseIndex-- {
		parseMessage := parseMsgs[parseIndex]
		if parseMessage.Role != roleAssistant {
			continue
		}
		parseLatestAssistantText = strings.TrimSpace(parseMessage.Content)
		if parseLatestAssistantText != "" {
			break
		}
	}
	parseLatestAssistantLower := strings.ToLower(parseLatestAssistantText)
	switch {
	case strings.Contains(parseLatestAssistantLower, "unauthenticated"),
		strings.Contains(parseLatestAssistantLower, "permission denied"),
		strings.Contains(parseLatestAssistantLower, "login"):
		parseCueTone = "warning"
		parseCueTitle = "Session check required"
		parseCueBody = "Sign in again to restore full chat access, then retry your message."
	case strings.Contains(parseLatestAssistantLower, "upgrade"),
		strings.Contains(parseLatestAssistantLower, "entitlement"),
		strings.Contains(parseLatestAssistantLower, "quota"),
		strings.Contains(parseLatestAssistantLower, "billing"):
		parseCueTone = "upgrade"
		parseCueTitle = "Access is plan-gated"
		parseCueBody = "Review plan or quota settings before retrying this request."
	}
	return Div(
		ClassStr("flex flex-col flex-1 min-w-0 h-full"),
		renderMobileControlBar(parseIntl, parseProviderOptions, parseActiveProvider, parseVisibleModelOptions, parseDisplayModel, parseCurrentThinkingMode, parseRequiredCapability, isStreaming, isThinkingSupported, isParseCanAccessAdmin, parseOpenAdminEntry, setProvider, setModel, setThinkingMode),
		renderDesktopControlBar(parseIntl, parseProviderOptions, parseActiveProvider, parseVisibleModelOptions, parseDisplayModel, parseCurrentThinkingMode, parseRequiredCapability, isStreaming, isThinkingSupported, isSidebarOpen, isParseCanAccessAdmin, parseOnToggleSidebar, parseOpenAdminEntry, setProvider, setModel, setThinkingMode),
		Div(ClassStr("flex flex-1 min-h-0 min-w-0"),
			Div(
				FromProps(Props{Style: parseLeftStyle}),
				ClassStr("flex min-h-0 min-w-0 flex-1 flex-col"),
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
					ApplyStarterPrompt:      parseApplyStarterPrompt,
					Journey:                 parseJourneyState,
					HandleSelectionMouse:    parseHandleSelectionMouse,
				}),
				parseInputArea(composerProps{
					Intl:               parseIntl,
					Value:              parseInputVal,
					Disabled:           isStreaming,
					ThreadCostSummary:  parseThreadCostSummary,
					AccountCostSummary: parseAccountCostSummary,
					CueTone:            parseCueTone,
					CueTitle:           parseCueTitle,
					CueBody:            parseCueBody,
					OnInput:            parseOnInput,
					OnKeyDown:          parseOnKey,
					OnSend:             parseOnSend,
					Journey:            parseJourneyState,
				}),
			),
			If(isParseSplitActive,
				Div(
					ID(idCanvasSplitHandle),
					ClassStr("hidden xl:flex w-3 shrink-0 cursor-col-resize items-center justify-center bg-[linear-gradient(180deg,rgba(255,255,255,0.03),rgba(255,255,255,0.01))] hover:bg-white/8 transition-colors"),
					TabIndex(0),
					Role("separator"),
					FromProps(Props{Aria: map[string]string{
						"orientation": "vertical",
						"label":       "Resize chat and canvas panes",
					}}),
					OnMouseDown(parseCanvas.StartSplitDrag),
					OnKeyDown(parseCanvas.HandleSplitKey),
					Div(ClassStr("h-16 w-1 rounded-full bg-white/10")),
				),
			),
			If(isParseSplitActive,
				Div(
					ClassStr("min-h-0 min-w-0 flex-1"),
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

func renderMobileControlBar(parseIntl i18n.Runtime, parseProviderOptions []providerOption, parseActiveProvider providerOption, parseVisibleModelOptions []modelOption, parseCurModel, parseCurrentThinkingMode, parseRequiredCapability string, isStreaming, isThinkingSupported, isParseCanAccessAdmin bool, parseOpenAdminEntry, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(ClassStr("chat-toolbar-shell md:hidden sticky top-0 z-10 flex flex-col border-b backdrop-blur-sm"),
		Div(ClassStr("flex items-center gap-3 px-4 py-3"),
			Img(
				Src(brandLogoURL),
				Attr("alt", appBrandName),
				ClassStr("h-12 w-auto shrink-0 object-contain"),
			),
			If(isParseCanAccessAdmin,
				Button(
					ClassStr("rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[10px] font-semibold uppercase tracking-[0.14em] text-white/50 transition-colors hover:bg-white/8 hover:text-white/70"),
					OnClick(parseOpenAdminEntry),
					Text("Admin"),
				),
			),
		),
		Div(ClassStr("grid grid-cols-2 gap-2 px-3 pb-3"),
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
			Div(ClassStr("group relative col-span-1"),
				renderToolbarSelect(
					"col-span-1 w-full",
					parseIntl.T(chatI18nNamespace, "controls.intelligence"),
					parseCurrentThinkingMode,
					isStreaming || !isThinkingSupported,
					setThinkingMode,
					Map(availableThinkingEfforts, func(parseOption3 thinkingEffortOption) ui.Node {
						return renderToolbarOption(parseOption3.ID, parseThinkingEffortLabel(parseIntl, parseOption3.ID))
					}),
				),
				If(parseRequiredCapability == "thinking",
					Span(ClassStr("pointer-events-none absolute -top-8 left-1/2 z-20 -translate-x-1/2 whitespace-nowrap rounded-lg border border-[#8df5cf]/20 bg-[#0a1f0f]/95 px-2.5 py-1 text-[11px] text-[#8df5cf] opacity-0 shadow-sm transition-opacity group-hover:opacity-100"),
						Text(parseIntl.T(chatI18nNamespace, "controls.capabilityThinkingFilter")),
					),
				),
			),
			If(!isThinkingSupported,
				P(ClassStr("col-span-2 px-1 text-[11px] leading-relaxed text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
		),
	)
}

func renderDesktopControlBar(parseIntl i18n.Runtime, parseProviderOptions []providerOption, parseActiveProvider providerOption, parseVisibleModelOptions []modelOption, parseCurModel, parseCurrentThinkingMode, parseRequiredCapability string, isStreaming, isThinkingSupported, isSidebarOpen, isParseCanAccessAdmin bool, parseOnToggleSidebar, parseOpenAdminEntry, setProvider, setModel, setThinkingMode ui.Handler) ui.Node {
	return Div(ClassStr("chat-toolbar-shell hidden md:flex sticky top-0 z-10 items-center gap-3 border-b px-3 py-2 backdrop-blur-sm"),
		Button(
			ClassStr(ClassNames(
				"shrink-0 p-1.5 rounded-xl text-white/40 hover:text-white hover:bg-white/10 transition-all duration-200 ease-out",
				When(isSidebarOpen, "pointer-events-none opacity-0 -translate-x-1 scale-95"),
				When(!isSidebarOpen, "opacity-100 translate-x-0 scale-100"),
			)),
			OnClick(parseOnToggleSidebar),
			parseSidebarToggleIcon(!isSidebarOpen),
		),
		Div(ClassStr("flex shrink-0 items-center gap-3"),
			Img(
				Src(brandLogoURL),
				Attr("alt", appBrandName),
				ClassStr("h-12 w-auto shrink-0 object-contain"),
			),
		),
		Div(ClassStr("flex min-w-0 flex-1 items-center justify-end gap-2"),
			If(isParseCanAccessAdmin,
				Button(
					ClassStr("h-8 shrink-0 rounded-full border border-white/10 bg-white/5 px-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-white/50 transition-colors hover:bg-white/8 hover:text-white/70"),
					OnClick(parseOpenAdminEntry),
					Text("Admin dashboard"),
				),
			),
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
			Div(ClassStr("group relative flex-[0_0_12rem]"),
				renderToolbarSelect(
					"w-full",
					parseIntl.T(chatI18nNamespace, "controls.intelligence"),
					parseCurrentThinkingMode,
					isStreaming || !isThinkingSupported,
					setThinkingMode,
					Map(availableThinkingEfforts, func(parseOption3 thinkingEffortOption) ui.Node {
						return renderToolbarOption(parseOption3.ID, parseThinkingEffortLabel(parseIntl, parseOption3.ID))
					}),
				),
				If(parseRequiredCapability == "thinking",
					Span(ClassStr("pointer-events-none absolute -top-8 left-1/2 z-20 -translate-x-1/2 whitespace-nowrap rounded-lg border border-[#8df5cf]/20 bg-[#0a1f0f]/95 px-2.5 py-1 text-[11px] text-[#8df5cf] opacity-0 shadow-sm transition-opacity group-hover:opacity-100"),
						Text(parseIntl.T(chatI18nNamespace, "controls.capabilityThinkingFilter")),
					),
				),
			),
			If(!isThinkingSupported,
				Span(ClassStr("shrink-0 text-[11px] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
		),
	)
}

func renderToolbarSelect(parseContainerClass, parseLabel, parseValue string, isDisabled bool, parseOnChange ui.Handler, parseOptions []ui.Node) ui.Node {
	return Label(ClassStr(ClassNames(
		"flex min-w-0 items-center gap-2 rounded-2xl border border-[#88ffd8]/20 bg-[#10101c]/70 px-2.5 py-1.5",
		parseContainerClass,
	)),
		Span(ClassStr("control-group-label shrink-0 min-w-[5.4rem]"), Text(parseLabel)),
		Select(
			Value(parseValue),
			DisabledIf(isDisabled),
			OnChange(parseOnChange),
			ClassStr(ClassNames(
				"toolbar-select h-8 min-w-0 flex-1 rounded-[0.95rem] border border-[#8fffd8]/22 bg-[#13141f]/84 px-2.5 text-sm text-[#e6f8ff] outline-none",
				When(isDisabled, "cursor-not-allowed opacity-60"),
			)),
			parseOptions,
		),
	)
}

func renderToolbarOption(parseValue, parseLabel string) ui.Node {
	return Option(
		ClassStr("toolbar-select-option"),
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
