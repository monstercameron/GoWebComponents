//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
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
	SidebarOpen            bool
	ExpandedThoughts       map[string]bool
	ThoughtCacheByMessage  map[int]renderWorkerThoughtCacheEntry
	CanvasCacheByMessage   map[int]renderWorkerCanvasCacheEntry
	ThreadCostSummary      threadCostSummary
	AccountCostSummary     accountCostSummary
	ThinkingSupported      bool
	MarkdownWorkerFallback bool
	CanvasSession          canvasSessionState
	CanvasOnlyRoute        bool
	AdminDashboardData     adminDashboardData
}

func parseDeriveAppViewState(parseCurrentState appState, parseCurrentPath string, parseUserName string, isSidebarOpen bool, parseThoughtCacheByMessage map[int]renderWorkerThoughtCacheEntry, parseCanvasCacheByMessage map[int]renderWorkerCanvasCacheEntry, parseThreadSummary threadCostSummary, parseAccountSummary accountCostSummary, isCanvasOnlyRoute bool, parseAdminDashboard adminDashboardData) appViewState {
	return appViewState{
		CurrentPath:            parseCurrentPath,
		GRPCReady:              parseCurrentState.GRPCReady,
		AuthResolved:           parseCurrentState.AuthResolved,
		Authenticated:          parseCurrentState.Authenticated,
		CanAccessAdmin:         parseCurrentState.CanAccessAdmin,
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
		SidebarOpen:            isSidebarOpen,
		ExpandedThoughts:       parseCurrentState.ExpandedThoughtSections,
		ThoughtCacheByMessage:  parseThoughtCacheByMessage,
		CanvasCacheByMessage:   parseCanvasCacheByMessage,
		ThreadCostSummary:      parseThreadSummary,
		AccountCostSummary:     parseAccountSummary,
		ThinkingSupported:      parseModelSupportsThinking(parseCurrentState.SelectedModel, parseCurrentState.ModelOptions, parseCurrentState.DefaultModel),
		MarkdownWorkerFallback: parseCurrentState.MarkdownWorkerFallback,
		CanvasSession:          parseCurrentState.CanvasSession,
		CanvasOnlyRoute:        isCanvasOnlyRoute,
		AdminDashboardData:     parseAdminDashboard,
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
}

// shouldRenderLandingShellEarly returns whether a public landing route should bypass the auth loading shell.
func shouldRenderLandingShellEarly(parseView appViewState) bool {
	return isLandingRoute(parseView.CurrentPath) && !parseView.Authenticated
}

func renderAppShell(parseProps appShellProps) ui.Node {
	parseContent := renderWorkspaceShell(parseProps)
	isWorkspace := true
	if shouldRenderLandingShellEarly(parseProps.View) {
		parseContent = renderLandingShell(parseProps.Intl, parseProps.View, parseProps.AuthSession)
		isWorkspace = false
	} else if !parseProps.View.GRPCReady || !parseProps.View.AuthResolved {
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
		Div(
			FromProps(Props{Raw: map[string]interface{}{
				"dir":                 string(parseProps.Intl.Direction()),
				"lang":                parseProps.Intl.Locale(),
				"data-current-locale": parseProps.Intl.Locale(),
			}}),
			Class(parseOuterClass),
			OnMouseUp(parseProps.QuoteSelection.HandleSelectionMouse),
			parseContent,
		),
	)
}

func renderWorkspaceShell(parseProps appShellProps) ui.Node {
	if parseProps.View.CanvasOnlyRoute {
		return Div(
			Class("flex h-full w-full min-w-0 min-h-0"),
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
		parseProps.ConversationList.Load,
		parseProps.ConversationList.RequestDelete,
		parseProps.ProfileSettings.Open,
		parseProps.ToggleSidebar,
		parseProps.View.CanAccessAdmin,
		parseProps.OpenAdminDashboard,
	)
	if isDashboardRoute(parseProps.View.CurrentPath) {
		return Fragment(
			parseSidebarNode,
			renderDashboardHome(parseProps.Intl, parseProps.View, parseProps.OpenAdminDashboard, parseProps.AdminCustomers, parseProps.AdminWorkspaces),
		)
	}
	return Fragment(
		parseSidebarNode,
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
		),
		If(parseProps.View.CanvasSession.Active && parseProps.View.CanvasSession.LayoutMode == canvasLayoutOverlay,
			Div(Class("fixed inset-0 z-40 flex min-h-0 min-w-0 bg-black/72 backdrop-blur-md overlay-in"),
				canvasWorkspacePane(parseProps.Intl, parseProps.View.CanvasSession, parseProps.CanvasWorkspace, false),
			),
		),
		parseQuoteSelectionPrompt(parseProps.QuoteSelection.State, parseProps.QuoteSelection.QuoteSelectedText, parseProps.QuoteSelection.StopPromptMouseUp),
		renderDeleteConversationModal(parseProps.Intl, parseProps.View, parseProps.StopBubble, parseProps.ConversationList),
		renderSpeechUpgradeModal(parseProps.Intl, parseProps.ShowSpeechModal, parseProps.SpeechModalError, parseProps.StopBubble, parseProps.CancelSpeechModal, parseProps.ConfirmSpeechModal),
		renderSettingsModal(parseProps.Intl, parseProps.View, parseProps.StopBubble, parseProps.ProfileSettings, parseProps.AuthSession),
	)
}

func renderDeleteConversationModal(parseIntl i18n.Runtime, parseView appViewState, parseStopBubble ui.Handler, parseConversationList conversationListController) ui.Node {
	if parseView.DeleteTarget <= 0 {
		return nil
	}
	return Div(
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm overlay-in"),
		OnClick(parseConversationList.CancelDelete),
		Div(
			Class("bg-[#111118] border border-white/[0.08] rounded-[1.35rem] p-6 max-w-sm w-full mx-4 flex flex-col gap-4 modal-in"),
			OnClick(parseStopBubble),
			P(Class("text-white font-semibold text-base"), Text(parseIntl.T(chatI18nNamespace, "modal.deleteTitle"))),
			P(Class("text-white/60 text-sm"), Text(parseIntl.T(chatI18nNamespace, "modal.deleteBody"))),
			Div(Class("flex gap-3 justify-end"),
				Button(
					Class("px-4 py-2 text-sm rounded-xl bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
					OnClick(parseConversationList.CancelDelete),
					Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
				),
				Button(
					Class("px-4 py-2 text-sm rounded-xl bg-red-600 text-white hover:bg-red-500 transition-colors font-medium"),
					OnClick(parseConversationList.ConfirmDelete),
					Text(parseIntl.T(chatI18nNamespace, "sidebar.deleteConversation")),
				),
			),
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
		Class("fixed inset-0 z-50 flex items-stretch justify-center bg-black/60 p-3 backdrop-blur-sm overlay-in sm:p-6"),
		OnClick(parseProfileSettings.Close),
		Div(
			Class("modal-in flex h-full max-h-[92vh] w-full max-w-6xl flex-col overflow-hidden rounded-[1.75rem] border border-white/10 bg-[#2f2f2f] shadow-[0_28px_90px_rgba(0,0,0,0.45)]"),
			OnClick(parseStopBubble),
			Div(Class("flex items-center justify-between gap-4 border-b border-white/10 px-5 py-4 sm:px-6"),
				Div(Class("flex min-w-0 items-center gap-3"),
					Div(Class("flex h-10 w-10 items-center justify-center rounded-2xl border border-white/10 bg-white/6 text-lg"), Text("\u2699\ufe0f")),
					Div(Class("min-w-0"),
						P(Class("text-base font-semibold text-white sm:text-lg"), Text(parseIntl.T(chatI18nNamespace, "modal.settingsTitle"))),
						P(Class("text-sm text-white/45"), Text(parseView.SessionEmail)),
					),
				),
				Button(
					Class("flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-white/5 text-white/70 transition-colors hover:bg-white/10 hover:text-white"),
					FromProps(Props{Aria: map[string]string{"label": parseIntl.T(chatI18nNamespace, "message.cancel")}}),
					OnClick(parseProfileSettings.Close),
					Span(Class("text-lg leading-none"), Text("\u00d7")),
				),
			),
			Div(Class("flex min-h-0 flex-1 flex-col gap-4 p-4 lg:flex-row lg:gap-5 lg:p-5"),
				Div(Class("min-h-0 overflow-hidden rounded-[1.5rem] border border-white/10 bg-black/18 lg:basis-[20%] lg:max-w-[20%]"),
					Div(Class("chat-scrollbar flex h-full min-h-0 flex-col overflow-y-auto p-3"),
						Div(Class("rounded-[1.2rem] border border-white/8 bg-white/[0.03] px-4 py-3"),
							P(Class("text-[11px] font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.settingsTitle"))),
							P(Class("mt-2 text-sm font-medium text-white"), Text(parseView.SessionEmail)),
							P(Class("mt-1 text-xs leading-6 text-white/42"), Text("Navigation")),
						),
						Div(Class("mt-3 flex flex-col gap-2"),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionProfile, parseIntl.T(chatI18nNamespace, "modal.displayName"), parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder"), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionTone, parseIntl.T(chatI18nNamespace, "modal.aiTone"), parseToneLabel(parseIntl, parseView.ToneInput), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionPrompt, parseIntl.T(chatI18nNamespace, "modal.systemPrompt"), parseSystemPromptNavSummary(parseIntl, parseView.SystemPromptInput), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionIntelligence, parseIntl.T(chatI18nNamespace, "modal.intelligence"), parseThinkingEffortLabel(parseIntl, parseCurrentThinkingMode), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionSpeech, parseIntl.T(chatI18nNamespace, "modal.ttsProviders"), parseTtsProviderLabel(parseView.SelectedTTSProvider), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionMemories, parseIntl.T(chatI18nNamespace, "modal.memories"), parseIntl.T(chatI18nNamespace, "modal.memoriesHelp"), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionLanguage, parseIntl.T(chatI18nNamespace, "modal.language"), parseLocaleLabel(parseView.LocaleInput), parseProfileSettings.NavigateSection),
							renderSettingsNavItem(parseIntl, parseActiveSection, settingsSectionBilling, parseIntl.T(chatI18nNamespace, "modal.billingTitle"), parseIntl.T(chatI18nNamespace, "modal.billingNavSummary"), parseProfileSettings.NavigateSection),
						),
					),
				),
				Div(Class("flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.5rem] border border-white/10 bg-black/12 lg:basis-[80%] lg:max-w-[80%]"),
					Div(Class("border-b border-white/10 bg-black/10 px-5 py-4"),
						P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(settingsSectionEyebrow(parseActiveSection))),
						P(Class("mt-2 text-xl font-semibold tracking-tight text-white"), Text(settingsSectionTitle(parseIntl, parseActiveSection))),
						P(Class("mt-2 max-w-2xl text-sm leading-7 text-white/55"), Text(settingsSectionDescription(parseIntl, parseView, parseActiveSection, parseCurrentThinkingMode))),
					),
					Div(Class("chat-scrollbar flex min-h-0 flex-1 overflow-y-auto"),
						Div(Class("flex w-full flex-col gap-4 p-4 sm:p-5"),
							renderActiveSettingsPane(parseIntl, parseView, parseActiveSection, parseCurrentThinkingMode, parseProfileSettings),
						),
					),
					Div(Class("flex flex-col gap-3 border-t border-white/10 bg-black/10 px-4 py-4 sm:flex-row sm:items-center sm:justify-end sm:px-5"),
						If(strings.TrimSpace(parseView.SettingsError) != "",
							Div(
								ID("settings-save-error"),
								Class("rounded-xl border border-[#f59e0b]/24 bg-[#f59e0b]/10 px-3 py-2 text-xs text-[#ffd7a3] sm:mr-auto"),
								Text(parseView.SettingsError),
							),
						),
						If(parseView.Authenticated,
							Button(
								Class("rounded-lg bg-red-500/15 px-4 py-2 text-sm text-red-200 transition-colors hover:bg-red-500/25 sm:mr-auto"),
								OnClick(parseAuth.Logout),
								Text(parseIntl.T(chatI18nNamespace, "auth.logout")),
							),
						),
						Button(
							Class("rounded-lg bg-white/10 px-4 py-2 text-sm text-white/70 transition-colors hover:bg-white/20"),
							OnClick(parseProfileSettings.Close),
							Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							Class("rounded-lg bg-white px-4 py-2 text-sm font-medium text-black transition-colors hover:bg-white/90"),
							OnClick(parseProfileSettings.Save),
							Text(parseIntl.T(chatI18nNamespace, "modal.save")),
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
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm overlay-in"),
		OnClick(parseCancel),
		Div(
			Class("bg-[#111118] border border-white/[0.08] rounded-[1.35rem] p-6 max-w-md w-full mx-4 flex flex-col gap-4 modal-in"),
			OnClick(parseStopBubble),
			P(Class("text-white font-semibold text-base"), Text(parseIntl.T(chatI18nNamespace, "modal.speechProviderTitle"))),
			P(Class("text-white/60 text-sm leading-6"), Text(parseIntl.T(chatI18nNamespace, "modal.speechProviderBody"))),
			If(strings.TrimSpace(parseErrorText) != "",
				Div(Class("rounded-xl border border-[#f59e0b]/24 bg-[#f59e0b]/10 px-3 py-2 text-xs text-[#ffd7a3]"), Text(parseErrorText)),
			),
			Div(Class("flex gap-3 justify-end"),
				Button(
					Class("px-4 py-2 text-sm rounded-xl bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
					OnClick(parseCancel),
					Text(parseIntl.T(chatI18nNamespace, "message.cancel")),
				),
				Button(
					Class("px-4 py-2 text-sm rounded-xl bg-[#00d9ff] text-[#05111d] hover:bg-[#33e3ff] transition-colors font-medium"),
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
		Class(ClassNames(
			"rounded-[1.2rem] border px-4 py-3 text-left transition-colors",
			When(parseActiveSection == parseSectionID, "border-[#8df5cf]/40 bg-[#8df5cf]/10 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]"),
			When(parseActiveSection != parseSectionID, "border-white/10 bg-white/[0.03] hover:bg-white/8"),
		)),
		P(Class("text-[11px] font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseTitle)),
		P(Class(ClassNames(
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
			Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.aiTone"))),
			P(Class("mt-1 text-sm text-white/55"), Text(parseToneDescription(parseIntl, parseView.ToneInput))),
			Div(Class("mt-4 flex flex-col gap-2"),
				Map(availableTones, func(parseOption toneOption) ui.Node {
					isActive := parseView.ToneInput == parseOption.ID
					return Button(
						Class(ClassNames(
							"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
							When(isActive, "border-white/30 bg-white/15 text-white"),
							When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
						)),
						Data(dataTone, parseOption.ID),
						OnClick(parseProfileSettings.HandleToneChange),
						Span(Class("font-medium"), Text(parseToneLabel(parseIntl, parseOption.ID))),
						Span(Class(ClassNames(
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
			Class("flex flex-col gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			Tag("textarea",
				Class("min-h-[10rem] w-full resize-y rounded-xl border border-white/20 bg-[#3a3a3a] px-4 py-3 text-sm text-white placeholder:text-white/40 focus:outline-none"),
				Placeholder(parseIntl.T(chatI18nNamespace, "modal.systemPromptPlaceholder")),
				Value(parseView.SystemPromptInput),
				OnInput(parseProfileSettings.HandleSystemPrompt),
			),
			P(
				Class("whitespace-pre-wrap rounded-xl border border-white/10 bg-[#2d2d2d] px-3 py-2 font-mono text-[11px] leading-relaxed text-white/60"),
				Text(parseIntl.T(chatI18nNamespace, "modal.systemPromptTemplate")),
			),
		)
	case settingsSectionIntelligence:
		return Div(
			ID(settingsSectionIntelligence),
			Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligence"))),
			P(Class("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceHelp"))),
			Div(Class("mt-4 flex flex-col gap-2"),
				Map(availableThinkingEfforts, func(parseOption2 thinkingEffortOption) ui.Node {
					isActive := parseCurrentThinkingMode == parseOption2.ID
					return Button(
						Class(ClassNames(
							"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
							When(isActive, "border-white/30 bg-white/15 text-white"),
							When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							When(!parseView.ThinkingSupported, "cursor-not-allowed opacity-50 hover:bg-[#3a3a3a] hover:text-white/60"),
						)),
						DisabledIf(!parseView.ThinkingSupported),
						Data(dataThinkingEffort, parseOption2.ID),
						OnClick(parseProfileSettings.HandleThinkingMode),
						Span(Class("font-medium"), Text(parseThinkingEffortLabel(parseIntl, parseOption2.ID))),
					)
				}),
			),
			If(!parseView.ThinkingSupported,
				P(Class("mt-3 text-xs leading-relaxed text-white/40"), Text(parseIntl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
			),
		)
	case settingsSectionSpeech:
		parseProviderOptions := parseTtsProviderOptionsForModels(parseView.ModelOptions, parseView.DefaultModelID)
		parseActiveProvider := parseResolveTTSProviderID(parseView.TTSProviderInput)
		return Div(
			ID(settingsSectionSpeech),
			Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
			P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.ttsProviders"))),
			P(Class("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.ttsProvidersHelp"))),
			Div(Class("mt-4 flex flex-col gap-2"),
				Map(parseProviderOptions, func(parseOption3 ttsProviderOption) ui.Node {
					isActive := parseActiveProvider == parseOption3.ID
					parseAvailabilityText := parseModelLabelForID(parseOption3.ResolvedModel, parseView.ModelOptions)
					if !parseOption3.Available {
						parseAvailabilityText = parseIntl.T(chatI18nNamespace, "modal.ttsProviderUnavailable")
					}
					return Button(
						Class(ClassNames(
							"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
							When(isActive, "border-white/30 bg-white/15 text-white"),
							When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							When(!parseOption3.Available, "cursor-not-allowed opacity-50 hover:bg-[#3a3a3a] hover:text-white/60"),
						)),
						DisabledIf(!parseOption3.Available),
						Data(dataTTSProvider, parseOption3.ID),
						OnClick(parseProfileSettings.HandleTTSProvider),
						Span(Class("font-medium"), Text(parseOption3.Label)),
						Span(Class(ClassNames(
							"text-xs",
							When(isActive, "text-white/60"),
							When(!isActive, "text-white/35"),
						)), Text(parseAvailabilityText)),
					)
				}),
			),
		)
	case settingsSectionMemories:
		return Div(
			ID(settingsSectionMemories),
			Class("flex flex-col gap-4"),
			Div(Class("flex items-center justify-between gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				Div(Class("min-w-0"),
					P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.memories"))),
					P(Class("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.memoriesHelp"))),
				),
				Button(
					Class("rounded-lg bg-white/10 px-3 py-1.5 text-xs text-white/80 transition-colors hover:bg-white/20"),
					OnClick(parseProfileSettings.AddMemory),
					Text(parseIntl.T(chatI18nNamespace, "modal.memoryAdd")),
				),
			),
			If(len(parseView.UserMemories) == 0,
				Div(Class("rounded-[1.4rem] border border-dashed border-white/10 bg-white/[0.03] px-4 py-4 text-sm text-white/40"), Text(parseIntl.T(chatI18nNamespace, "modal.memoriesEmpty"))),
			),
			Fragment(renderEditableUserMemories(parseIntl, parseView.UserMemories, parseProfileSettings)),
		)
	case settingsSectionLanguage:
		return Div(
			ID(settingsSectionLanguage),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.language"))),
				P(Class("mt-1 text-sm text-white/55"), Text(parseLocaleLabel(parseView.LocaleInput))),
				Div(Class("mt-4 flex flex-col gap-2"),
					Map(availableLocales, func(parseOption4 localeOption) ui.Node {
						isActive := parseView.LocaleInput == parseOption4.ID
						return Button(
							Class(ClassNames(
								"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
								When(isActive, "border-white/30 bg-white/15 text-white"),
								When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							)),
							Data(dataLocale, parseOption4.ID),
							OnClick(parseProfileSettings.HandleLocaleChange),
							Span(Class("font-medium"), Text(parseLocaleLabel(parseOption4.ID))),
						)
					}),
				),
			),
		)
	case settingsSectionBilling:
		parseHasUsage := parseView.AccountCostSummary.HasAnyExactCosts
		parseHasBreakdown := parseView.AccountCostSummary.PlatformFee > 0 || parseHasUsage
		parseCoverage := parseBillingCoverageText(parseView.AccountCostSummary)
		return Div(
			ID(settingsSectionBilling),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPlanLabel"))),
				P(Class("mt-3 text-lg font-semibold text-white"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPlanValue"))),
				P(Class("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.billingHelp"))),
			),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Usage breakdown")),
				If(!parseHasUsage,
					P(Class("mt-3 text-sm text-white/45"), Text(parseIntl.T(chatI18nNamespace, "modal.billingNoUsage"))),
				),
				If(parseHasBreakdown,
					Div(Class("mt-3 flex flex-col gap-2"),
						Div(Class("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(Class("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPlatformFee"))),
							P(ID(idBillingPlatformFeeValue), Class("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.PlatformFee))),
						),
						Div(Class("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(Class("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.billingUsageCost"))),
							P(ID(idBillingUsageValue), Class("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.UsageCost))),
						),
						Div(Class("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(Class("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.billingPremiumCost"))),
							P(ID(idBillingPremiumValue), Class("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.PremiumCost))),
						),
						Div(Class("flex items-center justify-between rounded-xl border border-[#8df5cf]/20 bg-[#8df5cf]/[0.04] px-4 py-2.5"),
							P(Class("text-sm font-medium text-white/80"), Text(parseIntl.T(chatI18nNamespace, "modal.billingTotalCost"))),
							P(ID(idBillingTotalValue), Class("text-sm font-semibold text-[#8df5cf]"), Text(formatCostUSD(parseView.AccountCostSummary.TotalCost))),
						),
					),
				),
			),
			If(parseCoverage != "",
				Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
					P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.billingCoverageLabel"))),
					P(Class("mt-2 text-sm text-white/70"), Text(parseCoverage)),
					If(!parseView.AccountCostSummary.AllThreadCostsExact,
						P(Class("mt-1 text-xs text-white/40"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsagePartial"))),
					),
				),
			),
		)
	default:
		return Div(
			ID(settingsSectionProfile),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.displayName"))),
				P(Class("mt-1 text-sm text-white/55"), Text(parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder"))),
				Input(
					ID(idNameInput),
					Type("text"),
					Class("mt-4 w-full rounded-xl border border-white/20 bg-[#3a3a3a] px-4 py-3 text-sm text-white placeholder:text-white/40 focus:outline-none"),
					Placeholder(parseIntl.T(chatI18nNamespace, "modal.displayNamePlaceholder")),
					Value(parseView.NameInput),
					OnInput(parseProfileSettings.HandleNameInput),
					OnKeyDown(parseProfileSettings.HandleNameKey),
				),
			),
			If(strings.TrimSpace(parseView.SessionEmail) != "",
				Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
					P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.profileEmailLabel"))),
					P(Class("mt-2 text-sm text-white/70"), Text(parseView.SessionEmail)),
				),
			),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsageTitle"))),
				If(!parseView.AccountCostSummary.HasAnyExactCosts,
					P(Class("mt-2 text-sm text-white/45"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsageLoading"))),
				),
				If(parseView.AccountCostSummary.HasAnyExactCosts,
					Div(Class("mt-3 flex flex-col gap-2"),
						Div(Class("flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.03] px-4 py-2.5"),
							P(Class("text-sm text-white/70"), Text(parseIntl.T(chatI18nNamespace, "modal.profileUsageTotalSpend"))),
							Span(Class("text-sm font-medium text-white"), Text(formatCostUSD(parseView.AccountCostSummary.TotalCost))),
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
