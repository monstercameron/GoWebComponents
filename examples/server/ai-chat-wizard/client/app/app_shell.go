//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/i18n"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/logging"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// appViewState is the render-only snapshot consumed by the shell helpers.
//
// Keeping this separate from App() makes the top-level component read as hook
// composition followed by one render call, rather than a long list of local
// aliases mixed into the UI tree.
type appViewState struct {
	CurrentPath            string
	GRPCReady              bool
	AuthResolved           bool
	Authenticated          bool
	CanAccessAdmin         bool
	IsSuperuser            bool
	AuthMode               string
	AuthError              string
	AuthSubmitting         bool
	AuthEmail              string
	AuthPassword           string
	AuthDisplayName        string
	SessionEmail           string
	Messages               []message
	InputText              string
	IsStreaming            bool
	EditIdx                int
	EditText               string
	ModelOptions           []modelOption
	DefaultModelID         string
	SelectedModel          string
	ConversationList       []convSummary
	ActiveConversationID   int64
	DeleteTarget           int64
	UserName               string
	UserInitials           string
	ShowSettingsModal      bool
	ActiveSettingsSection  string
	SettingsError          string
	NameInput              string
	ToneInput              string
	ThinkingEnabledInput   bool
	ThinkingEffortInput    string
	TTSProviderInput       string
	SystemPromptInput      string
	UserMemories           []editableUserMemory
	LocaleInput            string
	ThinkingEnabled        bool
	ThinkingEffort         string
	SelectedTTSProvider    string
	BridgeState            bridgeState
	BridgeReason           string
	SidebarOpen            bool
	ExpandedThoughts       map[string]bool
	ThoughtCacheByMessage  map[int]renderWorkerThoughtCacheEntry
	CanvasCacheByMessage   map[int]renderWorkerCanvasCacheEntry
	ThreadCostSummary      threadCostSummary
	AccountCostSummary     accountCostSummary
	ThinkingSupported      bool
	MarkdownWorkerFallback bool
	CatalogServerSynced    bool
	CanvasSession          canvasSessionState
	CanvasOnlyRoute        bool
	AdminDashboardData     adminDashboardData
	AdminServerTools       adminServerToolsData
	AdminOperations        adminOperationsData
}

func parseDeriveAppViewState(parseCurrentState appState, parseCurrentPath string, parseUserName string, isSidebarOpen bool, parseThoughtCacheByMessage map[int]renderWorkerThoughtCacheEntry, parseCanvasCacheByMessage map[int]renderWorkerCanvasCacheEntry, parseThreadSummary threadCostSummary, parseAccountSummary accountCostSummary, isCanvasOnlyRoute bool, parseAdminDashboard adminDashboardData, isCatalogServerSynced bool, parseAdminServerTools adminServerToolsData, parseAdminOperations adminOperationsData) appViewState {
	return appViewState{
		CurrentPath:            parseCurrentPath,
		GRPCReady:              parseCurrentState.GRPCReady,
		AuthResolved:           parseCurrentState.AuthResolved,
		Authenticated:          parseCurrentState.Authenticated,
		CanAccessAdmin:         parseCurrentState.CanAccessAdmin,
		IsSuperuser:            parseCurrentState.IsSuperuser,
		AuthMode:               parseCurrentState.AuthMode,
		AuthError:              parseCurrentState.AuthError,
		AuthSubmitting:         parseCurrentState.AuthSubmitting,
		AuthEmail:              parseCurrentState.AuthEmail,
		AuthPassword:           parseCurrentState.AuthPassword,
		AuthDisplayName:        parseCurrentState.AuthDisplayName,
		SessionEmail:           parseCurrentState.SessionEmail,
		Messages:               parseCurrentState.Messages,
		InputText:              parseCurrentState.InputText,
		IsStreaming:            parseCurrentState.Streaming,
		EditIdx:                parseCurrentState.EditIdx,
		EditText:               parseCurrentState.EditText,
		ModelOptions:           parseCurrentState.ModelOptions,
		DefaultModelID:         parseCurrentState.DefaultModel,
		SelectedModel:          parseCurrentState.SelectedModel,
		ConversationList:       parseCurrentState.ConversationList,
		ActiveConversationID:   parseCurrentState.ActiveConvID,
		DeleteTarget:           parseCurrentState.DeleteTarget,
		UserName:               parseUserName,
		UserInitials:           parseDisplayNameInitials(parseUserName),
		ShowSettingsModal:      parseCurrentState.ShowNameModal,
		ActiveSettingsSection:  parseCurrentState.ActiveSettingsSection,
		SettingsError:          parseCurrentState.SettingsError,
		NameInput:              parseCurrentState.NameInput,
		ToneInput:              parseCurrentState.ToneInput,
		ThinkingEnabledInput:   parseCurrentState.ThinkingEnabledInput,
		ThinkingEffortInput:    parseCurrentState.ThinkingEffortInput,
		TTSProviderInput:       parseResolveTTSProviderID(parseCurrentState.TTSProviderInput),
		SystemPromptInput:      parseCurrentState.SystemPromptInput,
		UserMemories:           parseCurrentState.UserMemories,
		LocaleInput:            parseNormalizeChatLocaleID(parseCurrentState.LocaleInput),
		ThinkingEnabled:        parseCurrentState.SelectedThinkingEnabled,
		ThinkingEffort:         parseCurrentState.SelectedThinkingEffort,
		SelectedTTSProvider:    parseResolveTTSProviderID(parseCurrentState.SelectedTTSProvider),
		BridgeState:            parseCurrentState.BridgeState,
		BridgeReason:           parseCurrentState.BridgeReason,
		SidebarOpen:            isSidebarOpen,
		ExpandedThoughts:       parseCurrentState.ExpandedThoughtSections,
		ThoughtCacheByMessage:  parseThoughtCacheByMessage,
		CanvasCacheByMessage:   parseCanvasCacheByMessage,
		ThreadCostSummary:      parseThreadSummary,
		AccountCostSummary:     parseAccountSummary,
		ThinkingSupported:      parseModelSupportsThinking(parseCurrentState.SelectedModel, parseCurrentState.ModelOptions, parseCurrentState.DefaultModel),
		MarkdownWorkerFallback: parseCurrentState.MarkdownWorkerFallback,
		CatalogServerSynced:    isCatalogServerSynced,
		CanvasSession:          parseCurrentState.CanvasSession,
		CanvasOnlyRoute:        isCanvasOnlyRoute,
		AdminDashboardData:     parseAdminDashboard,
		AdminServerTools:       parseAdminServerTools,
		AdminOperations:        parseAdminOperations,
	}
}

type appShellProps struct {
	Intl                 i18n.Runtime
	View                 appViewState
	ResetChat            ui.Handler
	ToggleSidebar        ui.Handler
	ToggleThoughtSection ui.Handler
	RequestSpeechUpgrade func()
	ShowSpeechModal      bool
	SpeechModalError     string
	ConfirmSpeechModal   ui.Handler
	CancelSpeechModal    ui.Handler
	StopBubble           ui.Handler
	OpenAdminDashboard   ui.Handler
	ConversationList     conversationListController
	ChatStream           chatStreamController
	ProfileSettings      profileSettingsController
	ModelPreferences     modelPreferencesController
	AuthSession          authSessionController
	QuoteSelection       quoteSelectionController
	TTSAudio             ttsAudioController
	ScrollMemory         threadScrollMemory
	CanvasWorkspace      canvasWorkspaceController
	AdminCustomers       adminCustomersController
	AdminWorkspaces      adminWorkspacesController
	AdminOperations      adminOperationsController
}

// shouldRenderLandingShellEarly returns whether a public landing route should bypass the auth loading shell.
func shouldRenderLandingShellEarly(parseView appViewState) bool {
	return isLandingRoute(parseView.CurrentPath) && !parseView.Authenticated
}

// shouldRenderAuthLoadingShell returns whether the app should still show the boot/auth loading shell instead of the resolved route UI.
func shouldRenderAuthLoadingShell(parseView appViewState) bool {
	return !parseView.AuthResolved
}

func renderAppShell(parseProps appShellProps) ui.Node {
	// Publish the active locale for non-hook helpers (browser-ICU cost
	// formatting) before any of this render's children format values.
	parseCurrentUILocale = parseProps.Intl.Locale()
	// Honor the OS reduced-motion preference live (ui.UsePrefersReducedMotion
	// re-renders on preference flips); the .reduce-motion class disables the
	// marketing and streaming animations in styles.go.
	isReduceMotion := ui.UsePrefersReducedMotion()
	// Screen-reader announcement when a streamed assistant reply finishes:
	// visually the stream just stops, so without this there is no non-visual
	// completion signal.
	parseAnnouncer := ui.UseAnnouncer()
	wasStreamingRef := ui.UseRef(false)
	ui.UseEffect(func() func() {
		if wasStreamingRef.Get() && !parseProps.View.IsStreaming {
			parseAnnouncer.Polite(parseProps.Intl.T(chatI18nNamespace, "thread.replyComplete"))
		}
		wasStreamingRef.Set(parseProps.View.IsStreaming)
		return nil
	}, parseProps.View.IsStreaming)
	// The browser tab title follows the active conversation
	// (interop.Document.SetTitle); falls back to the brand title on
	// non-thread surfaces. Keyed on the resolved string so the effect only
	// fires when the visible title actually changes.
	parseTabTitle := appBrandName + " – AI Chat Workspace"
	for _, parseSummary := range parseProps.View.ConversationList {
		if parseSummary.ID == parseProps.View.ActiveConversationID && strings.TrimSpace(parseSummary.Preview) != "" {
			parseTabTitle = strings.TrimSpace(parseSummary.Preview) + " — " + appBrandName
			break
		}
	}
	ui.UseEffect(func() func() {
		if parseDocument, parseErr := interop.GetDocument(); parseErr == nil {
			_ = parseDocument.SetTitle(parseTabTitle)
		}
		return nil
	}, parseTabTitle)
	// UI density is a persisted preference (ui.UsePersistedState): it survives
	// reloads via localStorage and syncs across tabs through the storage event.
	parseDensity := ui.UsePersistedState[string]("chatwizard.ui.density", "comfortable", ui.PersistLocal)
	isCompactDensity := parseDensity.Get() == "compact"
	parseContent := renderWorkspaceShell(parseProps)
	isWorkspace := true
	if shouldRenderLandingShellEarly(parseProps.View) {
		parseContent = renderLandingShell(parseProps.Intl, parseProps.View, parseProps.AuthSession)
		isWorkspace = false
	} else if shouldRenderAuthLoadingShell(parseProps.View) {
		parseContent = ui.Component(renderAuthLoadingShell, authLoadingShellProps{View: parseProps.View})
		isWorkspace = false
	} else if !parseProps.View.Authenticated {
		isWorkspace = false
		if isLandingRoute(parseProps.View.CurrentPath) {
			parseContent = renderLandingShell(parseProps.Intl, parseProps.View, parseProps.AuthSession)
		} else {
			parseContent = renderAuthShell(parseProps.Intl, parseProps.View, parseProps.AuthSession)
		}
	}
	// Workspace needs a fixed full-screen viewport with overflow clipped because
	// scrolling is managed internally per panel. Landing and auth pages are
	// standard document-flow pages that must be able to scroll freely.
	parseOuterClass := "chat-shell flex h-screen w-screen overflow-hidden text-white"
	if !isWorkspace {
		parseOuterClass = "h-screen w-full overflow-x-hidden overflow-y-auto text-white"
	}
	return Div(
		Tag("style", Text(chatWizardStyles)),
		Tag("style", Text(chatWizardMotionStyles)),
		Div(
			FromProps(Props{Raw: map[string]interface{}{
				"dir":                 string(parseProps.Intl.Direction()),
				"lang":                parseProps.Intl.Locale(),
				"data-current-locale": parseProps.Intl.Locale(),
			}}),
			ClassStr(ClassNames(
				parseOuterClass,
				When(isReduceMotion, "reduce-motion"),
				When(isCompactDensity, "density-compact"),
			)),
			parseContent,
		),
		If(isWorkspace, renderDensityToggle(parseProps.Intl, isCompactDensity, func() {
			if isCompactDensity {
				parseDensity.Set("comfortable")
				return
			}
			parseDensity.Set("compact")
		})),
		parseAnnouncer.Region(),
		renderDevToolsOverlay(parseProps.View),
	)
}

// renderDensityToggle is the persisted-density control: a quiet fixed chip in
// the bottom-right corner of the workspace.
func renderDensityToggle(parseIntl i18n.Runtime, isCompact bool, parseOnToggle func()) ui.Node {
	parseLabel := parseIntl.T(chatI18nNamespace, "shell.densityCompact")
	if isCompact {
		parseLabel = parseIntl.T(chatI18nNamespace, "shell.densityComfortable")
	}
	return Button(
		ID("density-toggle"),
		ClassStr("fixed bottom-3 right-3 z-40 rounded-full border border-white/[0.08] bg-[#13122080] px-3 py-1.5 text-[10px] font-medium uppercase tracking-[0.14em] text-white/35 backdrop-blur-md transition-colors hover:border-[#8e7bff]/40 hover:text-white/70"),
		FromProps(Props{Aria: map[string]string{"label": parseLabel}}),
		OnClick(parseOnToggle),
		Text(parseLabel),
	)
}

func renderWorkspaceShell(parseProps appShellProps) ui.Node {
	if parseProps.View.CanvasOnlyRoute {
		return Div(
			ClassStr("flex h-full w-full min-w-0 min-h-0"),
			canvasWorkspacePane(parseProps.Intl, parseProps.View.CanvasSession, parseProps.CanvasWorkspace, true),
		)
	}
	parseSidebarNode := parseSidebar(
		parseProps.View.ConversationList,
		parseProps.View.ActiveConversationID,
		parseProps.View.IsStreaming,
		parseProps.View.UserName,
		parseProps.View.UserInitials,
		parseProps.View.SidebarOpen,
		parseProps.ResetChat,
		parseProps.ConversationList,
		parseProps.ProfileSettings.Open,
		parseProps.ToggleSidebar,
		parseProps.View.CanAccessAdmin,
		parseProps.OpenAdminDashboard,
	)
	if isDashboardRoute(parseProps.View.CurrentPath) {
		return Fragment(
			parseSidebarNode,
			renderBridgeChurnBanner(parseProps.View),
			renderDashboardHome(parseProps.Intl, parseProps.View, parseProps.OpenAdminDashboard, parseProps.AdminCustomers, parseProps.AdminWorkspaces, parseProps.AdminOperations),
		)
	}
	return Fragment(
		parseSidebarNode,
		renderBridgeChurnBanner(parseProps.View),
		parseMainPanel(
			parseProps.View.Messages,
			parseProps.View.IsStreaming,
			parseProps.View.MarkdownWorkerFallback,
			parseProps.View.InputText,
			parseProps.ChatStream.HandleInput,
			parseProps.ChatStream.HandleKey,
			parseProps.ChatStream.Send,
			parseProps.ChatStream.ApplyStarterPrompt,
			parseProps.View.CanAccessAdmin,
			parseProps.OpenAdminDashboard,
			parseProps.View.EditIdx,
			parseProps.View.EditText,
			parseProps.ChatStream.StartEdit,
			parseProps.ChatStream.CancelEdit,
			parseProps.ChatStream.HandleEditChange,
			parseProps.ChatStream.SubmitEdit,
			parseProps.ChatStream.HandleEditKey,
			parseProps.ChatStream.Fork,
			parseProps.CanvasWorkspace.OpenFromMessage,
			parseProps.ToggleThoughtSection,
			parseProps.View.ModelOptions,
			parseProps.View.DefaultModelID,
			parseProps.View.ThreadCostSummary,
			parseProps.View.AccountCostSummary,
			parseProps.View.SelectedModel,
			parseProps.ModelPreferences.SetProvider,
			parseProps.ModelPreferences.SetModel,
			parseProps.View.ThinkingEnabled,
			parseProps.View.ThinkingEffort,
			parseProps.View.ThinkingSupported,
			parseProps.ModelPreferences.SetThinkingMode,
			parseProps.View.UserInitials,
			parseProps.View.SidebarOpen,
			parseProps.ToggleSidebar,
			parseProps.View.ExpandedThoughts,
			parseProps.View.ThoughtCacheByMessage,
			parseProps.View.CanvasCacheByMessage,
			parseProps.TTSAudio,
			parseProps.RequestSpeechUpgrade,
			parseProps.ScrollMemory,
			parseProps.View.CanvasSession,
			parseProps.CanvasWorkspace,
			parseProps.QuoteSelection.HandleSelectionMouse,
		),
		If(parseProps.View.CanvasSession.Active && parseProps.View.CanvasSession.LayoutMode == canvasLayoutOverlay,
			Div(ClassStr("fixed inset-0 z-40 flex min-h-0 min-w-0 bg-black/72 backdrop-blur-md overlay-in"),
				canvasWorkspacePane(parseProps.Intl, parseProps.View.CanvasSession, parseProps.CanvasWorkspace, false),
			),
		),
		parseQuoteSelectionPrompt(parseProps.QuoteSelection.State, parseProps.QuoteSelection.QuoteSelectedText, parseProps.QuoteSelection.StopPromptMouseUp),
		renderDeleteConversationModal(parseProps.Intl, parseProps.View, parseProps.StopBubble, parseProps.ConversationList),
		renderSpeechUpgradeModal(parseProps.Intl, parseProps.ShowSpeechModal, parseProps.SpeechModalError, parseProps.StopBubble, parseProps.CancelSpeechModal, parseProps.ConfirmSpeechModal),
		renderSettingsModal(parseProps.Intl, parseProps.View, parseProps.StopBubble, parseProps.ProfileSettings, parseProps.AuthSession),
		renderSettingsSaveErrorToast(parseProps.View, parseProps.ProfileSettings.DismissError),
	)
}

func renderBridgeChurnBanner(parseView appViewState) ui.Node {
	parseLabel := parseBridgeStatusLabel(parseView.BridgeState)
	if parseLabel == "" || !parseView.Authenticated {
		return nil
	}
	return Div(
		ClassStr("fixed left-1/2 top-3 z-40 flex -translate-x-1/2 items-center gap-2 rounded-full border border-[#f6b84b]/25 bg-[#171100]/90 px-3 py-1.5 text-xs text-[#ffd7a3] shadow-[0_8px_24px_rgba(0,0,0,0.28)] backdrop-blur-md"),
		FromProps(Props{Raw: map[string]interface{}{"role": "status"}}),
		Span(ClassStr("h-1.5 w-1.5 rounded-full bg-[#f6b84b]"), FromProps(Props{Aria: map[string]string{"hidden": "true"}})),
		Span(Text(parseLabel)),
	)
}

func renderDeleteConversationModal(parseIntl i18n.Runtime, parseView appViewState, parseStopBubble ui.Handler, parseConversationList conversationListController) ui.Node {
	if parseView.DeleteTarget <= 0 {
		return nil
	}
	return Div(
		ClassStr("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm overlay-in"),
		OnClick(parseConversationList.CancelDelete),
		Div(
			ClassStr("bg-[#13131e] border border-white/[0.08] rounded-[1.35rem] p-6 max-w-sm w-full mx-4 flex flex-col gap-4 modal-in"),
			OnClick(parseStopBubble),
			P(ClassStr("text-white font-semibold text-base"), Text(parseIntl.T(chatI18nNamespace, "modal.deleteTitle"))),
			P(ClassStr("text-white/60 text-sm"), Text(parseIntl.T(chatI18nNamespace, "modal.deleteBody"))),
			Div(ClassStr("flex gap-3 justify-end"),
				Button(
					ClassStr("px-4 py-2 text-sm rounded-xl bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
					OnClick(parseConversationList.CancelDelete),
					Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
				),
				Button(
					ClassStr("px-4 py-2 text-sm rounded-xl bg-red-600 text-white hover:bg-red-500 transition-colors font-medium"),
					OnClick(parseConversationList.ConfirmDelete),
					Text(parseIntl.T(chatI18nNamespace, "sidebar.deleteConversation")),
				),
			),
		),
	)
}

// renderSettingsSaveErrorToast renders a fixed amber toast when a settings save fails
// after the modal has already closed, so the user still sees the error.
func renderSettingsSaveErrorToast(parseView appViewState, parseDismiss ui.Handler) ui.Node {
	if parseView.ShowSettingsModal || strings.TrimSpace(parseView.SettingsError) == "" {
		return nil
	}
	return Div(
		ClassStr("fixed bottom-4 left-1/2 z-[60] flex -translate-x-1/2 items-start gap-3 rounded-2xl border border-[#f59e0b]/24 bg-[#1a1200] px-4 py-3 shadow-[0_8px_28px_rgba(0,0,0,0.4)] backdrop-blur-md modal-in max-w-sm w-[calc(100%-2rem)]"),
		Span(ClassStr("flex flex-col flex-1 text-sm text-[#ffd7a3] leading-5"),
			Text(parseUserErrorMessage(parseView.SettingsError)),
			renderSupportIDChip(parseUserErrorRequestID(parseView.SettingsError)),
		),
		Button(
			ClassStr("ml-1 flex-none text-[#ffd7a3]/60 hover:text-[#ffd7a3] transition-colors text-base leading-none"),
			OnClick(parseDismiss),
			Text("\u00d7"),
		),
	)
}

func renderSettingsModal(parseIntl i18n.Runtime, parseView appViewState, parseStopBubble ui.Handler, parseProfileSettings profileSettingsController, parseAuth authSessionController) ui.Node {
	if !parseView.ShowSettingsModal {
		return nil
	}
	parseActiveSection := parseNormalizeSettingsSectionID(parseView.ActiveSettingsSection)
	if parseActiveSection == "" {
		parseActiveSection = defaultSettingsSectionID
	}
	parseCurrentThinkingMode := "off"
	if parseView.ThinkingEnabledInput {
		parseCurrentThinkingMode = parseNormalizeSelectedThinkingEffort(parseView.ThinkingEffortInput)
	}
	return Div(
		ClassStr("fixed inset-0 z-50 flex items-stretch justify-center bg-black/60 p-3 backdrop-blur-sm overlay-in sm:p-6"),
		OnClick(parseProfileSettings.Close),
		Div(
			ClassStr("modal-in flex h-full max-h-[92vh] w-full max-w-6xl flex-col overflow-hidden rounded-[1.75rem] border border-white/10 bg-[#2f2f2f] shadow-[0_28px_90px_rgba(0,0,0,0.45)]"),
			OnClick(parseStopBubble),
			Div(ClassStr("flex items-center justify-between gap-4 border-b border-white/10 px-5 py-4 sm:px-6"),
				Div(ClassStr("flex min-w-0 items-center gap-3"),
					Div(ClassStr("flex h-10 w-10 items-center justify-center rounded-2xl border border-white/10 bg-white/6 text-lg"), Text("\u2699\ufe0f")),
					Div(ClassStr("min-w-0"),
						P(ClassStr("text-base font-semibold text-white sm:text-lg"), Text(parseIntl.T(chatI18nNamespace, "modal.settingsTitle"))),
						P(ClassStr("text-sm text-white/45"), Text(parseView.SessionEmail)),
					),
				),
				Button(
					ClassStr("flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-white/5 text-white/70 transition-colors hover:bg-white/10 hover:text-white"),
					FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "message.cancel")}}),
					OnClick(parseProfileSettings.Close),
					Span(ClassStr("text-lg leading-none"), Text("\u00d7")),
				),
			),
			Div(ClassStr("flex min-h-0 flex-1 flex-col gap-4 p-4 lg:flex-row lg:gap-5 lg:p-5"),
				Div(ClassStr("min-h-0 overflow-hidden rounded-[1.5rem] border border-white/10 bg-black/18 lg:basis-[20%] lg:max-w-[20%]"),
					Div(ClassStr("chat-scrollbar flex h-full min-h-0 flex-col overflow-y-auto p-3"),
						Div(ClassStr("rounded-[1.2rem] border border-white/8 bg-white/[0.03] px-4 py-3"),
							P(ClassStr("text-[11px] font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.settingsTitle"))),
							P(ClassStr("mt-2 text-sm font-medium text-white"), Text(parseView.SessionEmail)),
							P(ClassStr("mt-1 text-xs leading-6 text-white/42"), Text("Navigation")),
						),
						Div(ClassStr("mt-3 flex flex-col gap-2"),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionProfile, parseIntl.T(chatI18nNamespace, "modal.displayName"), parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder"), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionTone, parseIntl.T(chatI18nNamespace, "modal.aiTone"), parseToneLabel(parseIntl, parseView.ToneInput), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionPrompt, parseIntl.T(chatI18nNamespace, "modal.systemPrompt"), parseSystemPromptNavSummary(parseIntl, parseView.SystemPromptInput), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionIntelligence, parseIntl.T(chatI18nNamespace, "modal.intelligence"), parseThinkingEffortLabel(parseIntl, parseCurrentThinkingMode), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionSpeech, parseIntl.T(chatI18nNamespace, "modal.ttsProviders"), parseTtsProviderLabel(parseView.SelectedTTSProvider), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionMemories, parseIntl.T(chatI18nNamespace, "modal.memories"), parseIntl.T(chatI18nNamespace, "modal.memoriesHelp"), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionLanguage, parseIntl.T(chatI18nNamespace, "modal.language"), parseLocaleLabel(parseView.LocaleInput), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionBilling, parseIntl.T(chatI18nNamespace, "modal.billingTitle"), parseIntl.T(chatI18nNamespace, "modal.billingNavSummary"), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionSecurity, parseIntl.T(chatI18nNamespace, "modal.securityTitle"), parseIntl.T(chatI18nNamespace, "modal.securityNavSummary"), parseProfileSettings.NavigateSection),
						),
					),
				),
				Div(ClassStr("flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.5rem] border border-white/10 bg-black/12 lg:basis-[80%] lg:max-w-[80%]"),
					Div(ClassStr("border-b border-white/10 bg-black/10 px-5 py-4"),
						P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(settingsSectionEyebrow(parseActiveSection))),
						P(ClassStr("mt-2 text-xl font-semibold tracking-tight text-white"), Text(settingsSectionTitle(parseIntl, parseActiveSection))),
						P(ClassStr("mt-2 max-w-2xl text-sm leading-7 text-white/55"), Text(settingsSectionDescription(parseIntl, parseView, parseActiveSection, parseCurrentThinkingMode))),
					),
					Div(ClassStr("chat-scrollbar flex min-h-0 flex-1 overflow-y-auto"),
						Div(ClassStr("flex w-full flex-col gap-4 p-4 sm:p-5"),
							renderActiveSettingsPane(parseIntl, parseView, parseActiveSection, parseCurrentThinkingMode, parseProfileSettings),
						),
					),
					Div(ClassStr("flex flex-col gap-3 border-t border-white/10 bg-black/10 px-4 py-4 sm:flex-row sm:items-center sm:justify-end sm:px-5"),
						If(strings.TrimSpace(parseView.SettingsError) != "",
							Div(
								ID("settings-save-error"),
								ClassStr("rounded-xl border border-[#f59e0b]/24 bg-[#f59e0b]/10 px-3 py-2 text-xs text-[#ffd7a3] sm:mr-auto"),
								Text(parseView.SettingsError),
							),
						),
						If(parseBridgeStatusLabel(parseView.BridgeState) != "",
							Div(
								ClassStr("rounded-xl border border-[#f6b84b]/20 bg-[#f6b84b]/10 px-3 py-2 text-xs text-[#ffd7a3] sm:mr-auto"),
								FromProps(Props{Raw: map[string]interface{}{"role": "status"}}),
								Text(parseBridgeStatusLabel(parseView.BridgeState)+". Settings will close and sync in the background when possible."),
							),
						),
						If(parseView.Authenticated,
							Button(
								ClassStr("rounded-lg bg-red-500/15 px-4 py-2 text-sm text-red-200 transition-colors hover:bg-red-500/25 sm:mr-auto"),
								OnClick(parseAuth.Logout),
								Text(parseIntl.T(chatI18nNamespace, "auth.logout")),
							),
						),
						Button(
							ClassStr("rounded-lg bg-white/10 px-4 py-2 text-sm text-white/70 transition-colors hover:bg-white/20"),
							OnClick(parseProfileSettings.Close),
							Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							ClassStr("rounded-lg bg-white px-4 py-2 text-sm font-medium text-black transition-colors hover:bg-white/90"),
							OnClick(parseProfileSettings.Save),
							If(parseBridgeStatusLabel(parseView.BridgeState) == "",
								Text(parseIntl.T(chatI18nNamespace, "modal.save")),
							),
							If(parseBridgeStatusLabel(parseView.BridgeState) != "",
								Text("Save locally"),
							),
						),
					),
				),
			),
		),
	)
}

func renderSpeechUpgradeModal(parseIntl i18n.Runtime, isShow bool, parseErrorText string, parseStopBubble ui.Handler, parseCancel ui.Handler, parseConfirm ui.Handler) ui.Node {
	if !isShow {
		return nil
	}
	return Div(
		ClassStr("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm overlay-in"),
		OnClick(parseCancel),
		Div(
			ClassStr("bg-[#13131e] border border-white/[0.08] rounded-[1.35rem] p-6 max-w-md w-full mx-4 flex flex-col gap-4 modal-in"),
			OnClick(parseStopBubble),
			P(ClassStr("text-white font-semibold text-base"), Text(parseIntl.T(chatI18nNamespace, "modal.speechProviderTitle"))),
			P(ClassStr("text-white/60 text-sm leading-6"), Text(parseIntl.T(chatI18nNamespace, "modal.speechProviderBody"))),
			If(strings.TrimSpace(parseErrorText) != "",
				Div(ClassStr("rounded-xl border border-[#f59e0b]/24 bg-[#f59e0b]/10 px-3 py-2 text-xs text-[#ffd7a3]"), Text(parseErrorText)),
			),
			Div(ClassStr("flex gap-3 justify-end"),
				Button(
					ClassStr("px-4 py-2 text-sm rounded-xl bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
					OnClick(parseCancel),
					Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
				),
				Button(
					ClassStr("px-4 py-2 text-sm rounded-xl bg-[#8e7bff] text-[#0a0a14] hover:bg-[#a99bff] transition-colors font-medium"),
					OnClick(parseConfirm),
					Text(parseIntl.T(chatI18nNamespace, "modal.speechProviderConfirm")),
				),
			),
		),
	)
}

func renderSettingsNavItem(parseIntl i18n.Runtime, parseActiveSection, parseSectionID, parseTitle, parseSummary string, parseOnNavigate ui.Handler) ui.Node {
	_ = parseIntl
	return A(
		Href(buildSettingsRoute(parseSectionID)),
		Data(dataSettingsSection, parseSectionID),
		OnClick(parseOnNavigate),
		ClassStr(ClassNames(
			"rounded-[1.2rem] border px-4 py-3 text-left transition-colors",
			When(parseActiveSection == parseSectionID, "border-[#8df5cf]/40 bg-[#8df5cf]/10 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]"),
			When(parseActiveSection != parseSectionID, "border-white/10 bg-white/[0.03] hover:bg-white/8"),
		)),
		P(ClassStr("text-[11px] font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseTitle)),
		P(ClassStr(ClassNames(
			"mt-1 text-sm leading-6",
			When(parseActiveSection == parseSectionID, "text-white"),
			When(parseActiveSection != parseSectionID, "text-white/74"),
		)), Text(parseSummary)),
	)
}

func renderActiveSettingsPane(parseIntl i18n.Runtime, parseView appViewState, parseActiveSection, parseCurrentThinkingMode string, parseProfileSettings profileSettingsController) ui.Node {
	switch parseActiveSection {
	case settingsSectionTone:
		return Div(
			ID(settingsSectionTone),
			ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.aiTone"))),
			P(ClassStr("mt-1 text-sm text-white/55"), Text(parseToneDescription(parseIntl, parseView.ToneInput))),
			Div(ClassStr("mt-4 flex flex-col gap-2"),
				Map(availableTones, func(parseOption toneOption) ui.Node {
					isActive := parseView.ToneInput == parseOption.ID
					return Button(
						ClassStr(ClassNames(
							"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
							When(isActive, "border-white/30 bg-white/15 text-white"),
							When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
						)),
						Data(dataTone, parseOption.ID),
						OnClick(parseProfileSettings.HandleToneChange),
						Span(ClassStr("font-medium"), Text(parseToneLabel(parseIntl, parseOption.ID))),
						Span(ClassStr(ClassNames(
							"text-xs",
							When(isActive, "text-white/60"),
							When(!isActive, "text-white/30"),
						)), Text(parseToneDescription(parseIntl, parseOption.ID))),
					)
				}),
			),
		)
	case settingsSectionPrompt:
		return Div(
			ID(settingsSectionPrompt),
			ClassStr("flex flex-col gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			Tag("textarea",
				ClassStr("min-h-[10rem] w-full resize-y rounded-xl border border-white/20 bg-[#3a3a3a] px-4 py-3 text-sm text-white placeholder:text-white/40 focus:outline-none"),
				Placeholder(parseIntl.T(chatI18nNamespace, "modal.systemPromptPlaceholder")),
				Value(parseView.SystemPromptInput),
				OnInput(parseProfileSettings.HandleSystemPrompt),
			),
			P(
				ClassStr("whitespace-pre-wrap rounded-xl border border-white/10 bg-[#2d2d2d] px-3 py-2 font-mono text-[11px] leading-relaxed text-white/60"),
				Text(parseIntl.T(chatI18nNamespace, "modal.systemPromptTemplate")),
			),
		)
	case settingsSectionIntelligence:
		return Div(
			ID(settingsSectionIntelligence),
			ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligence"))),
			P(ClassStr("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceHelp"))),
			Div(ClassStr("mt-4 flex flex-col gap-2"),
				Map(availableThinkingEfforts, func(parseOption2 thinkingEffortOption) ui.Node {
					isActive := parseCurrentThinkingMode == parseOption2.ID
					return Button(
						ClassStr(ClassNames(
							"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
							When(isActive, "border-white/30 bg-white/15 text-white"),
							When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							When(!parseView.ThinkingSupported, "cursor-not-allowed opacity-50 hover:bg-[#3a3a3a] hover:text-white/60"),
						)),
						DisabledIf(!parseView.ThinkingSupported),
						Data(dataThinkingEffort, parseOption2.ID),
						OnClick(parseProfileSettings.HandleThinkingMode),
						Span(ClassStr("font-medium"), Text(parseThinkingEffortLabel(parseIntl, parseOption2.ID))),
					)
				}),
			),
			If(!parseView.ThinkingSupported,
				P(ClassStr("mt-3 text-xs leading-relaxed text-white/40"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
		)
	case settingsSectionSpeech:
		parseProviderOptions := parseTtsProviderOptionsForModels(parseView.ModelOptions, parseView.DefaultModelID)
		parseActiveProvider := parseResolveTTSProviderID(parseView.TTSProviderInput)
		return Div(
			ID(settingsSectionSpeech),
			ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.ttsProviders"))),
			P(ClassStr("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.ttsProvidersHelp"))),
			Div(ClassStr("mt-4 flex flex-col gap-2"),
				Map(parseProviderOptions, func(parseOption3 ttsProviderOption) ui.Node {
					isActive := parseActiveProvider == parseOption3.ID
					parseAvailabilityText := parseModelLabelForID(parseOption3.ResolvedModel, parseView.ModelOptions)
					if !parseOption3.Available {
						parseAvailabilityText = parseIntl.T(chatI18nNamespace, "modal.ttsProviderUnavailable")
					}
					return Button(
						ClassStr(ClassNames(
							"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
							When(isActive, "border-white/30 bg-white/15 text-white"),
							When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							When(!parseOption3.Available, "cursor-not-allowed opacity-50 hover:bg-[#3a3a3a] hover:text-white/60"),
						)),
						DisabledIf(!parseOption3.Available),
						Data(dataTTSProvider, parseOption3.ID),
						OnClick(parseProfileSettings.HandleTTSProvider),
						Span(ClassStr("font-medium"), Text(parseOption3.Label)),
						Span(ClassStr(ClassNames(
							"text-xs",
							When(isActive, "text-white/60"),
							When(!isActive, "text-white/35"),
						)), Text(parseAvailabilityText)),
					)
				}),
			),
		)
	case settingsSectionMemories:
		// Count auto-extracted (non-managed) session memories for extraction parity feedback.
		parseSessionMemCount := 0
		for _, parseM := range parseView.UserMemories {
			if !isManagedUserNameMemory(parseM) {
				parseSessionMemCount++
			}
		}
		isParseExtractionActive := parseSessionMemCount > 0
		return Div(
			ID(settingsSectionMemories),
			ClassStr("flex flex-col gap-4"),
			Div(ClassStr("flex items-center justify-between gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				Div(ClassStr("min-w-0"),
					P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.memories"))),
					P(ClassStr("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.memoriesHelp"))),
				),
				Button(
					ClassStr("rounded-lg bg-white/10 px-3 py-1.5 text-xs text-white/80 transition-colors hover:bg-white/20"),
					OnClick(parseProfileSettings.AddMemory),
					Text(parseIntl.T(chatI18nNamespace, "modal.memoryAdd")),
				),
			),
			// Extraction status row — visible parity signal showing whether the AI is producing memories.
			Div(ClassStr("flex items-start justify-between gap-3 rounded-[1.4rem] border border-white/8 bg-white/[0.02] px-4 py-3"),
				Div(ClassStr("min-w-0 flex-1"),
					If(isParseExtractionActive,
						P(ClassStr("text-sm text-white/65"), Text("Session memories are being extracted from your chats.")),
					),
					If(!isParseExtractionActive,
						P(ClassStr("text-sm text-white/45"), Text("No session memories yet — the AI will build these automatically after your chats.")),
					),
					P(ClassStr("mt-1 text-xs text-white/30"), Text("Memory extraction depends on your current AI model. Some models may not yet support automatic extraction.")),
				),
				If(isParseExtractionActive,
					Span(ClassStr("shrink-0 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[0.6rem] font-semibold uppercase tracking-wide text-emerald-400/80"), Text("Active")),
				),
				If(!isParseExtractionActive,
					Span(ClassStr("shrink-0 rounded-full bg-amber-500/10 px-2 py-0.5 text-[0.6rem] font-semibold uppercase tracking-wide text-amber-400/60"), Text("Pending")),
				),
			),
			If(len(parseView.UserMemories) == 0,
				Div(ClassStr("rounded-[1.4rem] border border-dashed border-white/10 bg-white/[0.03] px-4 py-4 text-sm text-white/40"), Text(parseIntl.T(chatI18nNamespace, "modal.memoriesEmpty"))),
			),
			Fragment(renderEditableUserMemories(parseIntl, parseView.UserMemories, parseProfileSettings)),
		)
	case settingsSectionLanguage:
		return Div(
			ID(settingsSectionLanguage),
			ClassStr("flex flex-col gap-4"),
			Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.language"))),
				P(ClassStr("mt-1 text-sm text-white/55"), Text(parseLocaleLabel(parseView.LocaleInput))),
				Div(ClassStr("mt-4 flex flex-col gap-2"),
					Map(availableLocales, func(parseOption4 localeOption) ui.Node {
						isActive := parseView.LocaleInput == parseOption4.ID
						return Button(
							ClassStr(ClassNames(
								"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
								When(isActive, "border-white/30 bg-white/15 text-white"),
								When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							)),
							Data(dataLocale, parseOption4.ID),
							OnClick(parseProfileSettings.HandleLocaleChange),
							Span(ClassStr("font-medium"), Text(parseLocaleLabel(parseOption4.ID))),
						)
					}),
				),
			),
		)
	case settingsSectionBilling:
		parseHasUsage := parseView.AccountCostSummary.HasAnyExactCosts
		parseHasBreakdown := parseView.AccountCostSummary.PlatformFee > 0 || parseHasUsage
		parseCoverage := parseBillingCoverageText(parseView.AccountCostSummary)
		parsePlanName := parseView.AccountCostSummary.PlanLabel
		if parsePlanName == "" {
			parsePlanName = parseIntl.T(chatI18nNamespace, "modal.billingPlanValue")
		}
		return Div(
			ID(settingsSectionBilling),
			ClassStr("flex flex-col gap-4"),
			Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPlanLabel"))),
				P(ClassStr("mt-3 text-lg font-semibold text-white"), Text(parsePlanName)),
				P(ClassStr("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.billingHelp"))),
			),
			Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Usage breakdown")),
				If(!parseHasUsage,
					P(ClassStr("mt-3 text-sm text-white/45"), Text(parseIntl.T(chatI18nNamespace, "modal.billingNoUsage"))),
				),
				If(parseHasBreakdown,
					Div(ClassStr("mt-3 flex flex-col gap-2"),
						Div(ClassStr("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(ClassStr("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPlatformFee"))),
							P(ID(idBillingPlatformFeeValue), ClassStr("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.PlatformFee))),
						),
						Div(ClassStr("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(ClassStr("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.billingUsageCost"))),
							P(ID(idBillingUsageValue), ClassStr("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.UsageCost))),
						),
						Div(ClassStr("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(ClassStr("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPremiumCost"))),
							P(ID(idBillingPremiumValue), ClassStr("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.PremiumCost))),
						),
						Div(ClassStr("flex items-center justify-between rounded-xl border border-[#8df5cf]/20 bg-[#8df5cf]/[0.04] px-4 py-2.5"),
							P(ClassStr("text-sm font-medium text-white/80"), Text(parseIntl.T(chatI18nNamespace, "modal.billingTotalCost"))),
							P(ID(idBillingTotalValue), ClassStr("text-sm font-semibold text-[#8df5cf]"), Text(formatCostUSD(parseView.AccountCostSummary.TotalCost))),
						),
					),
				),
			),
			If(len(parseView.AccountCostSummary.Invoices) > 0,
				Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
					P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Recent invoices")),
					Div(ClassStr("mt-3 flex flex-col gap-2"),
						Map(parseView.AccountCostSummary.Invoices, func(parseInv billingInvoiceRow) ui.Node {
							parsePeriodLabel := parseInv.PeriodStart
							if len(parsePeriodLabel) > 10 {
								parsePeriodLabel = parsePeriodLabel[:10]
							}
							if parsePeriodLabel == "" {
								parsePeriodLabel = "Invoice"
							}
							isParseUnpaid := parseInv.Status == "open" || parseInv.Status == "past_due"
							return Div(
								ClassStr("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
								P(ClassStr("text-sm text-white/70"), Text(parsePeriodLabel)),
								Div(ClassStr("flex items-center gap-2"),
									If(isParseUnpaid,
										Span(ClassStr("text-xs text-amber-400 px-1.5 py-0.5 rounded bg-amber-900/30"), Text(parseInv.Status)),
									),
									P(ClassStr("text-sm font-medium text-white"), Text(formatCostUSD(float64(parseInv.TotalCents)/100.0))),
								),
							)
						}),
					),
				),
			),
			If(parseCoverage != "",
				Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
					P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.billingCoverageLabel"))),
					P(ClassStr("mt-2 text-sm text-white/70"), Text(parseCoverage)),
					If(!parseView.AccountCostSummary.AllThreadCostsExact,
						P(ClassStr("mt-1 text-xs text-white/40"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsagePartial"))),
					),
				),
			),
		)
	case settingsSectionSecurity:
		return renderSecuritySettingsPane(parseIntl, parseView)
	default:
		return Div(
			ID(settingsSectionProfile),
			ClassStr("flex flex-col gap-4"),
			Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.displayName"))),
				P(ClassStr("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder"))),
				Input(
					ID(idNameInput),
					Type("text"),
					ClassStr("mt-4 w-full rounded-xl border border-white/20 bg-[#3a3a3a] px-4 py-3 text-sm text-white placeholder:text-white/40 focus:outline-none"),
					Placeholder(parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder")),
					Value(parseView.NameInput),
					OnInput(parseProfileSettings.HandleNameInput),
					OnKeyDown(parseProfileSettings.HandleNameKey),
				),
			),
			If(strings.TrimSpace(parseView.SessionEmail) != "",
				Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
					P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.profileEmailLabel"))),
					P(ClassStr("mt-2 text-sm text-white/70"), Text(parseView.SessionEmail)),
				),
			),
			Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsageTitle"))),
				If(!parseView.AccountCostSummary.HasAnyExactCosts,
					P(ClassStr("mt-2 text-sm text-white/45"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsageLoading"))),
				),
				If(parseView.AccountCostSummary.HasAnyExactCosts,
					Div(ClassStr("mt-3 flex flex-col gap-2"),
						Div(ClassStr("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(ClassStr("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsageTotalSpend"))),
							Span(ClassStr("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.TotalCost))),
						),
					),
				),
			),
		)
	}
}

func settingsSectionTitle(parseIntl i18n.Runtime, parseActiveSection string) string {
	switch parseActiveSection {
	case settingsSectionTone:
		return parseIntl.T(chatI18nNamespace, "modal.aiTone")
	case settingsSectionPrompt:
		return parseIntl.T(chatI18nNamespace, "modal.systemPrompt")
	case settingsSectionIntelligence:
		return parseIntl.T(chatI18nNamespace, "modal.intelligence")
	case settingsSectionSpeech:
		return parseIntl.T(chatI18nNamespace, "modal.ttsProviders")
	case settingsSectionMemories:
		return parseIntl.T(chatI18nNamespace, "modal.memories")
	case settingsSectionLanguage:
		return parseIntl.T(chatI18nNamespace, "modal.language")
	case settingsSectionBilling:
		return parseIntl.T(chatI18nNamespace, "modal.billingTitle")
	case settingsSectionSecurity:
		return parseIntl.T(chatI18nNamespace, "modal.securityTitle")
	default:
		return parseIntl.T(chatI18nNamespace, "modal.displayName")
	}
}

func settingsSectionDescription(parseIntl i18n.Runtime, parseView appViewState, parseActiveSection, parseCurrentThinkingMode string) string {
	switch parseActiveSection {
	case settingsSectionTone:
		return parseToneDescription(parseIntl, parseView.ToneInput)
	case settingsSectionPrompt:
		return parseIntl.T(chatI18nNamespace, "modal.systemPromptHelp")
	case settingsSectionIntelligence:
		if !parseView.ThinkingSupported {
			return parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable")
		}
		return parseThinkingEffortLabel(parseIntl, parseCurrentThinkingMode)
	case settingsSectionSpeech:
		return parseIntl.T(chatI18nNamespace, "modal.ttsProvidersHelp")
	case settingsSectionMemories:
		return parseIntl.T(chatI18nNamespace, "modal.memoriesHelp")
	case settingsSectionLanguage:
		return parseLocaleLabel(parseView.LocaleInput)
	case settingsSectionBilling:
		return parseIntl.T(chatI18nNamespace, "modal.billingHelp")
	case settingsSectionSecurity:
		return parseIntl.T(chatI18nNamespace, "modal.securityHelp")
	default:
		return parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder")
	}
}

func settingsSectionEyebrow(parseActiveSection string) string {
	switch parseActiveSection {
	case settingsSectionTone:
		return "Tone"
	case settingsSectionPrompt:
		return "Prompt"
	case settingsSectionIntelligence:
		return "Intelligence"
	case settingsSectionSpeech:
		return "Speech"
	case settingsSectionMemories:
		return "Memories"
	case settingsSectionLanguage:
		return "Language"
	case settingsSectionBilling:
		return "Billing"
	case settingsSectionSecurity:
		return "Security"
	default:
		return "Profile"
	}
}

func parseNewChatHandler(parseApp ui.Reducer[appState, appAction], parseScrollMemory threadScrollMemory, parseOnNavigateRoot func()) ui.Handler {
	return ui.UseEvent(func() {
		parseCurrentState := parseApp.Get()
		if parseCurrentState.Streaming {
			return
		}
		parseScrollMemory.CancelPendingPersist()
		parseScrollMemory.ParsePersistNow(parseCurrentState.ActiveConvID)
		parseScrollMemory.ParsePrepareRestore(0)
		chatLog.Info("new chat", logging.Fields{"prev_conv_id": parseCurrentState.ActiveConvID, "messages": len(parseCurrentState.Messages)})
		parseApp.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
		parseApp.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
		parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
		parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		if parseOnNavigateRoot != nil {
			parseOnNavigateRoot()
		}
	})
}

func parseToggleSidebarHandler(parseSidebarOpenState state.Atom[bool]) ui.Handler {
	return ui.UseEvent(func() {
		parseSidebarOpenState.Set(!parseSidebarOpenState.Get())
	})
}

func parseToggleThoughtSectionHandler(parseApp ui.Reducer[appState, appAction]) ui.Handler {
	return ui.UseEvent(func(parseE ui.Event) {
		parseSectionKey := strings.TrimSpace(parseE.JSValue().Get("currentTarget").Get("dataset").Get(dataThoughtSection).String())
		if parseSectionKey == "" {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionToggleThoughtSection, ThoughtSectionKey: parseSectionKey})
	})
}

// renderSecuritySettingsPane renders the linked-login-methods surface in the Security settings section.
// It is display-only — no proto RPC for mutating linked providers exists yet.
func renderSecuritySettingsPane(parseIntl i18n.Runtime, parseView appViewState) ui.Node {
	c := chatI18nNamespace
	// Determine whether the account has a password set from the session email presence
	// (password login is always available when the user authenticated with email).
	isParsePasswordLinked := strings.TrimSpace(parseView.SessionEmail) != ""

	return Div(
		ID(settingsSectionSecurity),
		ClassStr("flex flex-col gap-4"),
		// linked methods header card
		Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(c, "modal.securityLinkedMethodsLabel"))),
			P(ClassStr("mt-1 text-sm text-white/55"), Text(parseIntl.T(c, "modal.securityLinkedMethodsHelp"))),
		),
		// password row
		Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			Div(ClassStr("flex items-center justify-between gap-3"),
				Div(
					P(ClassStr("text-sm font-medium text-white"), Text(parseIntl.T(c, "modal.securityPasswordMethod"))),
					P(ClassStr("mt-1 text-xs text-white/45"), Text(parseView.SessionEmail)),
				),
				If(isParsePasswordLinked,
					Span(ClassStr("shrink-0 rounded-full border border-emerald-400/30 bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-300"), Text(parseIntl.T(c, "modal.securityLinked"))),
				),
				If(!isParsePasswordLinked,
					Span(ClassStr("shrink-0 rounded-full border border-white/15 bg-white/5 px-2.5 py-0.5 text-xs font-medium text-white/40"), Text(parseIntl.T(c, "modal.securityNotLinked"))),
				),
			),
		),
		// Google row (always shown — not yet linked, no RPC to link/unlink)
		Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			Div(ClassStr("flex items-center justify-between gap-3"),
				Div(
					P(ClassStr("text-sm font-medium text-white"), Text(parseIntl.T(c, "modal.securityGoogleMethod"))),
					P(ClassStr("mt-1 text-xs text-white/45"), Text(parseIntl.T(c, "modal.securityGoogleHelp"))),
				),
				Span(ClassStr("shrink-0 rounded-full border border-white/15 bg-white/5 px-2.5 py-0.5 text-xs font-medium text-white/40"), Text(parseIntl.T(c, "modal.securityNotLinked"))),
			),
		),
		// Enterprise SSO row (always shown — availability depends on workspace plan)
		Div(ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			Div(ClassStr("flex items-center justify-between gap-3"),
				Div(
					P(ClassStr("text-sm font-medium text-white"), Text(parseIntl.T(c, "modal.securitySSOMethod"))),
					P(ClassStr("mt-1 text-xs text-white/45"), Text(parseIntl.T(c, "modal.securitySSOHelp"))),
				),
				Span(ClassStr("shrink-0 rounded-full border border-white/10 bg-white/[0.03] px-2.5 py-0.5 text-xs font-medium text-white/30"), Text(parseIntl.T(c, "modal.securityUnavailable"))),
			),
		),
		// informational note about local password always being available for testing
		Div(ClassStr("rounded-[1.4rem] border border-amber-400/15 bg-amber-500/[0.05] p-4 sm:p-5"),
			P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-amber-300/60"), Text(parseIntl.T(c, "modal.securityNoteLabel"))),
			P(ClassStr("mt-2 text-sm leading-6 text-white/45"), Text(parseIntl.T(c, "modal.securityNoteBody"))),
		),
	)
}
