//go:build js && wasm

package app

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/router"
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
	NameInput              string
	ToneInput              string
	ThinkingEnabledInput   bool
	ThinkingEffortInput    string
	SystemPromptInput      string
	UserMemories           []editableUserMemory
	LocaleInput            string
	ThinkingEnabled        bool
	ThinkingEffort         string
	SidebarOpen            bool
	ExpandedThoughts       map[string]bool
	ThreadCostSummary      threadCostSummary
	ThinkingSupported      bool
	MarkdownWorkerFallback bool
	CanvasSession          canvasSessionState
	CanvasOnlyRoute        bool
}

func deriveAppViewState(currentState appState, userName string, sidebarOpen bool, threadSummary threadCostSummary, canvasOnlyRoute bool) appViewState {
	return appViewState{
		CurrentPath:            router.GetCurrentPath(),
		GRPCReady:              currentState.GRPCReady,
		AuthResolved:           currentState.AuthResolved,
		Authenticated:          currentState.Authenticated,
		AuthMode:               currentState.AuthMode,
		AuthError:              currentState.AuthError,
		AuthSubmitting:         currentState.AuthSubmitting,
		AuthEmail:              currentState.AuthEmail,
		AuthPassword:           currentState.AuthPassword,
		AuthDisplayName:        currentState.AuthDisplayName,
		SessionEmail:           currentState.SessionEmail,
		Messages:               currentState.Messages,
		InputText:              currentState.InputText,
		IsStreaming:            currentState.Streaming,
		EditIdx:                currentState.EditIdx,
		EditText:               currentState.EditText,
		ModelOptions:           currentState.ModelOptions,
		DefaultModelID:         currentState.DefaultModel,
		SelectedModel:          currentState.SelectedModel,
		ConversationList:       currentState.ConversationList,
		ActiveConversationID:   currentState.ActiveConvID,
		DeleteTarget:           currentState.DeleteTarget,
		UserName:               userName,
		UserInitials:           displayNameInitials(userName),
		ShowSettingsModal:      currentState.ShowNameModal,
		ActiveSettingsSection:  currentState.ActiveSettingsSection,
		NameInput:              currentState.NameInput,
		ToneInput:              currentState.ToneInput,
		ThinkingEnabledInput:   currentState.ThinkingEnabledInput,
		ThinkingEffortInput:    currentState.ThinkingEffortInput,
		SystemPromptInput:      currentState.SystemPromptInput,
		UserMemories:           currentState.UserMemories,
		LocaleInput:            normalizeChatLocaleID(currentState.LocaleInput),
		ThinkingEnabled:        currentState.SelectedThinkingEnabled,
		ThinkingEffort:         currentState.SelectedThinkingEffort,
		SidebarOpen:            sidebarOpen,
		ExpandedThoughts:       currentState.ExpandedThoughtSections,
		ThreadCostSummary:      threadSummary,
		ThinkingSupported:      modelSupportsThinking(currentState.SelectedModel, currentState.ModelOptions, currentState.DefaultModel),
		MarkdownWorkerFallback: currentState.MarkdownWorkerFallback,
		CanvasSession:          currentState.CanvasSession,
		CanvasOnlyRoute:        canvasOnlyRoute,
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
	ConversationList     conversationListController
	ChatStream           chatStreamController
	ProfileSettings      profileSettingsController
	ModelPreferences     modelPreferencesController
	AuthSession          authSessionController
	QuoteSelection       quoteSelectionController
	TTSAudio             ttsAudioController
	ScrollMemory         threadScrollMemory
	CanvasWorkspace      canvasWorkspaceController
}

func renderAppShell(props appShellProps) ui.Node {
	content := renderWorkspaceShell(props)
	if !props.View.GRPCReady || !props.View.AuthResolved {
		content = renderAuthLoadingShell(props.Intl, props.View)
	} else if !props.View.Authenticated {
		if isLandingRoute(props.View.CurrentPath) {
			content = renderLandingShell(props.Intl, props.View, props.AuthSession)
		} else {
			content = renderAuthShell(props.Intl, props.View, props.AuthSession)
		}
	}
	return Div(
		Tag("style", Text(chatWizardStyles)),
		Div(
			FromProps(Props{Raw: map[string]interface{}{
				"dir":                 string(props.Intl.Direction()),
				"lang":                props.Intl.Locale(),
				"data-current-locale": props.Intl.Locale(),
			}}),
			Class("flex h-screen w-screen overflow-hidden bg-[#212121] text-white"),
			OnMouseUp(props.QuoteSelection.HandleSelectionMouse),
			content,
		),
	)
}

func renderWorkspaceShell(props appShellProps) ui.Node {
	if props.View.CanvasOnlyRoute {
		return Div(
			Class("flex h-full w-full min-w-0 min-h-0"),
			canvasWorkspacePane(props.Intl, props.View.CanvasSession, props.CanvasWorkspace, true),
		)
	}
	return Fragment(
		sidebar(
			props.View.ConversationList,
			props.View.ActiveConversationID,
			props.View.IsStreaming,
			props.View.UserName,
			props.View.UserInitials,
			props.View.SidebarOpen,
			props.ResetChat,
			props.ConversationList.Load,
			props.ConversationList.RequestDelete,
			props.ProfileSettings.Open,
			props.ToggleSidebar,
		),
		mainPanel(
			props.View.Messages,
			props.View.IsStreaming,
			props.View.MarkdownWorkerFallback,
			props.View.InputText,
			props.ChatStream.HandleInput,
			props.ChatStream.HandleKey,
			props.ChatStream.Send,
			props.View.EditIdx,
			props.View.EditText,
			props.ChatStream.StartEdit,
			props.ChatStream.CancelEdit,
			props.ChatStream.HandleEditChange,
			props.ChatStream.SubmitEdit,
			props.ChatStream.HandleEditKey,
			props.ChatStream.Fork,
			props.CanvasWorkspace.OpenFromMessage,
			props.ToggleThoughtSection,
			props.View.ModelOptions,
			props.View.DefaultModelID,
			props.View.ThreadCostSummary,
			props.View.SelectedModel,
			props.ModelPreferences.SetProvider,
			props.ModelPreferences.SetModel,
			props.View.ThinkingEnabled,
			props.View.ThinkingEffort,
			props.View.ThinkingSupported,
			props.ModelPreferences.SetThinkingMode,
			props.View.UserInitials,
			props.View.SidebarOpen,
			props.ToggleSidebar,
			props.View.ExpandedThoughts,
			props.TTSAudio,
			props.RequestSpeechUpgrade,
			props.ScrollMemory,
			props.View.CanvasSession,
			props.CanvasWorkspace,
		),
		If(props.View.CanvasSession.Active && props.View.CanvasSession.LayoutMode == canvasLayoutOverlay,
			Div(Class("fixed inset-0 z-40 flex min-h-0 min-w-0 bg-black/72 backdrop-blur-md overlay-in"),
				canvasWorkspacePane(props.Intl, props.View.CanvasSession, props.CanvasWorkspace, false),
			),
		),
		quoteSelectionPrompt(props.QuoteSelection.State, props.QuoteSelection.QuoteSelectedText, props.QuoteSelection.StopPromptMouseUp),
		renderDeleteConversationModal(props.Intl, props.View, props.StopBubble, props.ConversationList),
		renderSpeechUpgradeModal(props.Intl, props.ShowSpeechModal, props.SpeechModalError, props.StopBubble, props.CancelSpeechModal, props.ConfirmSpeechModal),
		renderSettingsModal(props.Intl, props.View, props.StopBubble, props.ProfileSettings, props.AuthSession),
	)
}

func renderDeleteConversationModal(intl i18n.Runtime, view appViewState, stopBubble ui.Handler, conversationList conversationListController) ui.Node {
	if view.DeleteTarget <= 0 {
		return nil
	}
	return Div(
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm overlay-in"),
		OnClick(conversationList.CancelDelete),
		Div(
			Class("bg-[#2f2f2f] border border-white/10 rounded-2xl p-6 max-w-sm w-full mx-4 flex flex-col gap-4 modal-in"),
			OnClick(stopBubble),
			P(Class("text-white font-semibold text-base"), Text(intl.T(chatI18nNamespace, "modal.deleteTitle"))),
			P(Class("text-white/60 text-sm"), Text(intl.T(chatI18nNamespace, "modal.deleteBody"))),
			Div(Class("flex gap-3 justify-end"),
				Button(
					Class("px-4 py-2 text-sm rounded-lg bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
					OnClick(conversationList.CancelDelete),
					Text(intl.T(chatI18nNamespace, "message.cancel")),
				),
				Button(
					Class("px-4 py-2 text-sm rounded-lg bg-red-600 text-white hover:bg-red-500 transition-colors font-medium"),
					OnClick(conversationList.ConfirmDelete),
					Text(intl.T(chatI18nNamespace, "sidebar.deleteConversation")),
				),
			),
		),
	)
}

func renderSettingsModal(intl i18n.Runtime, view appViewState, stopBubble ui.Handler, profileSettings profileSettingsController, auth authSessionController) ui.Node {
	if !view.ShowSettingsModal {
		return nil
	}
	activeSection := normalizeSettingsSectionID(view.ActiveSettingsSection)
	if activeSection == "" {
		activeSection = defaultSettingsSectionID
	}
	currentThinkingMode := "off"
	if view.ThinkingEnabledInput {
		currentThinkingMode = normalizeSelectedThinkingEffort(view.ThinkingEffortInput)
	}
	return Div(
		Class("fixed inset-0 z-50 flex items-stretch justify-center bg-black/60 p-3 backdrop-blur-sm overlay-in sm:p-6"),
		OnClick(profileSettings.Close),
		Div(
			Class("modal-in flex h-full max-h-[92vh] w-full max-w-6xl flex-col overflow-hidden rounded-[1.75rem] border border-white/10 bg-[#2f2f2f] shadow-[0_28px_90px_rgba(0,0,0,0.45)]"),
			OnClick(stopBubble),
			Div(Class("flex items-center justify-between gap-4 border-b border-white/10 px-5 py-4 sm:px-6"),
				Div(Class("flex min-w-0 items-center gap-3"),
					Div(Class("flex h-10 w-10 items-center justify-center rounded-2xl border border-white/10 bg-white/6 text-lg"), Text("\u2699\ufe0f")),
					Div(Class("min-w-0"),
						P(Class("text-base font-semibold text-white sm:text-lg"), Text(intl.T(chatI18nNamespace, "modal.settingsTitle"))),
						P(Class("text-sm text-white/45"), Text(view.SessionEmail)),
					),
				),
				Button(
					Class("rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/70 transition-colors hover:bg-white/10 hover:text-white"),
					OnClick(profileSettings.Close),
					Text(intl.T(chatI18nNamespace, "message.cancel")),
				),
			),
			Div(Class("flex min-h-0 flex-1 flex-col gap-4 p-4 lg:flex-row lg:gap-5 lg:p-5"),
				Div(Class("min-h-0 overflow-hidden rounded-[1.5rem] border border-white/10 bg-black/18 lg:basis-[20%] lg:max-w-[20%]"),
					Div(Class("chat-scrollbar flex h-full min-h-0 flex-col overflow-y-auto p-3"),
						Div(Class("rounded-[1.2rem] border border-white/8 bg-white/[0.03] px-4 py-3"),
							P(Class("text-[11px] font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.settingsTitle"))),
							P(Class("mt-2 text-sm font-medium text-white"), Text(view.SessionEmail)),
							P(Class("mt-1 text-xs leading-6 text-white/42"), Text("Navigation")),
						),
						Div(Class("mt-3 flex flex-col gap-2"),
							renderSettingsNavItem(intl, activeSection, settingsSectionProfile, intl.T(chatI18nNamespace, "modal.displayName"), intl.T(chatI18nNamespace, "modal.displayNamePlaceholder"), profileSettings.NavigateSection),
							renderSettingsNavItem(intl, activeSection, settingsSectionTone, intl.T(chatI18nNamespace, "modal.aiTone"), toneLabel(intl, view.ToneInput), profileSettings.NavigateSection),
							renderSettingsNavItem(intl, activeSection, settingsSectionPrompt, intl.T(chatI18nNamespace, "modal.systemPrompt"), intl.T(chatI18nNamespace, "modal.systemPromptHelp"), profileSettings.NavigateSection),
							renderSettingsNavItem(intl, activeSection, settingsSectionIntelligence, intl.T(chatI18nNamespace, "modal.intelligence"), thinkingEffortLabel(intl, currentThinkingMode), profileSettings.NavigateSection),
							renderSettingsNavItem(intl, activeSection, settingsSectionMemories, intl.T(chatI18nNamespace, "modal.memories"), intl.T(chatI18nNamespace, "modal.memoriesHelp"), profileSettings.NavigateSection),
							renderSettingsNavItem(intl, activeSection, settingsSectionLanguage, intl.T(chatI18nNamespace, "modal.language"), localeLabel(view.LocaleInput), profileSettings.NavigateSection),
						),
					),
				),
				Div(Class("flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.5rem] border border-white/10 bg-black/12 lg:basis-[80%] lg:max-w-[80%]"),
					Div(Class("border-b border-white/10 bg-black/10 px-5 py-4"),
						P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(settingsSectionEyebrow(activeSection))),
						P(Class("mt-2 text-xl font-semibold tracking-tight text-white"), Text(settingsSectionTitle(intl, activeSection))),
						P(Class("mt-2 max-w-2xl text-sm leading-7 text-white/55"), Text(settingsSectionDescription(intl, view, activeSection, currentThinkingMode))),
					),
					Div(Class("chat-scrollbar flex min-h-0 flex-1 overflow-y-auto"),
						Div(Class("flex w-full flex-col gap-4 p-4 sm:p-5"),
							renderActiveSettingsPane(intl, view, activeSection, currentThinkingMode, profileSettings),
						),
					),
					Div(Class("flex flex-col gap-3 border-t border-white/10 bg-black/10 px-4 py-4 sm:flex-row sm:items-center sm:justify-end sm:px-5"),
						If(view.Authenticated,
							Button(
								Class("rounded-lg bg-red-500/15 px-4 py-2 text-sm text-red-200 transition-colors hover:bg-red-500/25 sm:mr-auto"),
								OnClick(auth.Logout),
								Text(intl.T(chatI18nNamespace, "auth.logout")),
							),
						),
						Button(
							Class("rounded-lg bg-white/10 px-4 py-2 text-sm text-white/70 transition-colors hover:bg-white/20"),
							OnClick(profileSettings.Close),
							Text(intl.T(chatI18nNamespace, "message.cancel")),
						),
						Button(
							Class("rounded-lg bg-white px-4 py-2 text-sm font-medium text-black transition-colors hover:bg-white/90"),
							OnClick(profileSettings.Save),
							Text(intl.T(chatI18nNamespace, "modal.save")),
						),
					),
				),
			),
		),
	)
}

func renderSpeechUpgradeModal(intl i18n.Runtime, show bool, errorText string, stopBubble ui.Handler, cancel ui.Handler, confirm ui.Handler) ui.Node {
	if !show {
		return nil
	}
	return Div(
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm overlay-in"),
		OnClick(cancel),
		Div(
			Class("bg-[#2f2f2f] border border-white/10 rounded-2xl p-6 max-w-md w-full mx-4 flex flex-col gap-4 modal-in"),
			OnClick(stopBubble),
			P(Class("text-white font-semibold text-base"), Text(intl.T(chatI18nNamespace, "modal.speechProviderTitle"))),
			P(Class("text-white/60 text-sm leading-6"), Text(intl.T(chatI18nNamespace, "modal.speechProviderBody"))),
			If(strings.TrimSpace(errorText) != "",
				Div(Class("rounded-xl border border-[#f59e0b]/24 bg-[#f59e0b]/10 px-3 py-2 text-xs text-[#ffd7a3]"), Text(errorText)),
			),
			Div(Class("flex gap-3 justify-end"),
				Button(
					Class("px-4 py-2 text-sm rounded-lg bg-white/10 text-white/70 hover:bg-white/20 transition-colors"),
					OnClick(cancel),
					Text(intl.T(chatI18nNamespace, "message.cancel")),
				),
				Button(
					Class("px-4 py-2 text-sm rounded-lg bg-[#19c37d] text-[#052516] hover:bg-[#31de90] transition-colors font-medium"),
					OnClick(confirm),
					Text(intl.T(chatI18nNamespace, "modal.speechProviderConfirm")),
				),
			),
		),
	)
}

func renderSettingsNavItem(intl i18n.Runtime, activeSection, sectionID, title, summary string, onNavigate ui.Handler) ui.Node {
	return A(
		Href(buildSettingsRoute(sectionID)),
		Data(dataSettingsSection, sectionID),
		OnClick(onNavigate),
		Class(ClassNames(
			"rounded-[1.2rem] border px-4 py-3 text-left transition-colors",
			When(activeSection == sectionID, "border-[#8df5cf]/40 bg-[#8df5cf]/10 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]"),
			When(activeSection != sectionID, "border-white/10 bg-white/[0.03] hover:bg-white/8"),
		)),
		P(Class("text-[11px] font-medium uppercase tracking-[0.22em] text-white/35"), Text(title)),
		P(Class(ClassNames(
			"mt-1 text-sm leading-6",
			When(activeSection == sectionID, "text-white"),
			When(activeSection != sectionID, "text-white/74"),
		)), Text(summary)),
	)
}

func renderActiveSettingsPane(intl i18n.Runtime, view appViewState, activeSection, currentThinkingMode string, profileSettings profileSettingsController) ui.Node {
	switch activeSection {
	case settingsSectionTone:
		return Div(
			ID(settingsSectionTone),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.aiTone"))),
				P(Class("mt-1 text-sm text-white/55"), Text(toneDescription(intl, view.ToneInput))),
				Div(Class("mt-4 flex flex-col gap-2"),
					Map(availableTones, func(option toneOption) ui.Node {
						isActive := view.ToneInput == option.ID
						return Button(
							Class(ClassNames(
								"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
								When(isActive, "border-white/30 bg-white/15 text-white"),
								When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							)),
							Data(dataTone, option.ID),
							OnClick(profileSettings.HandleToneChange),
							Span(Class("font-medium"), Text(toneLabel(intl, option.ID))),
							Span(Class(ClassNames(
								"text-xs",
								When(isActive, "text-white/60"),
								When(!isActive, "text-white/30"),
							)), Text(toneDescription(intl, option.ID))),
						)
					}),
				),
			),
		)
	case settingsSectionPrompt:
		return Div(
			ID(settingsSectionPrompt),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.systemPrompt"))),
				P(Class("mt-1 text-sm text-white/55"), Text(intl.T(chatI18nNamespace, "modal.systemPromptHelp"))),
				Tag("textarea",
					Class("mt-4 min-h-[18rem] w-full resize-y rounded-xl border border-white/20 bg-[#3a3a3a] px-4 py-3 text-sm text-white placeholder:text-white/40 focus:outline-none"),
					Placeholder(intl.T(chatI18nNamespace, "modal.systemPromptPlaceholder")),
					Value(view.SystemPromptInput),
					OnInput(profileSettings.HandleSystemPrompt),
				),
				P(
					Class("mt-3 whitespace-pre-wrap rounded-xl border border-white/10 bg-[#2d2d2d] px-3 py-2 font-mono text-[11px] leading-relaxed text-white/60"),
					Text(intl.T(chatI18nNamespace, "modal.systemPromptTemplate")),
				),
			),
		)
	case settingsSectionIntelligence:
		return Div(
			ID(settingsSectionIntelligence),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.intelligence"))),
				P(Class("mt-1 text-sm text-white/55"), Text(intl.T(chatI18nNamespace, "modal.intelligenceHelp"))),
				Div(Class("mt-4 flex flex-col gap-2"),
					Map(availableThinkingEfforts, func(option thinkingEffortOption) ui.Node {
						isActive := currentThinkingMode == option.ID
						return Button(
							Class(ClassNames(
								"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
								When(isActive, "border-white/30 bg-white/15 text-white"),
								When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
								When(!view.ThinkingSupported, "cursor-not-allowed opacity-50 hover:bg-[#3a3a3a] hover:text-white/60"),
							)),
							DisabledIf(!view.ThinkingSupported),
							Data(dataThinkingEffort, option.ID),
							OnClick(profileSettings.HandleThinkingMode),
							Span(Class("font-medium"), Text(thinkingEffortLabel(intl, option.ID))),
						)
					}),
				),
				If(!view.ThinkingSupported,
					P(Class("mt-3 text-xs leading-relaxed text-white/40"), Text(intl.T(chatI18nNamespace, "modal.intelligenceUnavailable"))),
				),
			),
		)
	case settingsSectionMemories:
		return Div(
			ID(settingsSectionMemories),
			Class("flex flex-col gap-4"),
			Div(Class("flex items-center justify-between gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				Div(Class("min-w-0"),
					P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.memories"))),
					P(Class("mt-1 text-sm text-white/55"), Text(intl.T(chatI18nNamespace, "modal.memoriesHelp"))),
				),
				Button(
					Class("rounded-lg bg-white/10 px-3 py-1.5 text-xs text-white/80 transition-colors hover:bg-white/20"),
					OnClick(profileSettings.AddMemory),
					Text(intl.T(chatI18nNamespace, "modal.memoryAdd")),
				),
			),
			If(len(view.UserMemories) == 0,
				Div(Class("rounded-[1.4rem] border border-dashed border-white/10 bg-white/[0.03] px-4 py-4 text-sm text-white/40"), Text(intl.T(chatI18nNamespace, "modal.memoriesEmpty"))),
			),
			Fragment(renderEditableUserMemories(intl, view.UserMemories, profileSettings)),
		)
	case settingsSectionLanguage:
		return Div(
			ID(settingsSectionLanguage),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.language"))),
				P(Class("mt-1 text-sm text-white/55"), Text(localeLabel(view.LocaleInput))),
				Div(Class("mt-4 flex flex-col gap-2"),
					Map(availableLocales, func(option localeOption) ui.Node {
						isActive := view.LocaleInput == option.ID
						return Button(
							Class(ClassNames(
								"flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left text-sm transition-colors",
								When(isActive, "border-white/30 bg-white/15 text-white"),
								When(!isActive, "border-white/10 bg-[#3a3a3a] text-white/60 hover:bg-white/10 hover:text-white/90"),
							)),
							Data(dataLocale, option.ID),
							OnClick(profileSettings.HandleLocaleChange),
							Span(Class("font-medium"), Text(localeLabel(option.ID))),
						)
					}),
				),
			),
		)
	default:
		return Div(
			ID(settingsSectionProfile),
			Class("flex flex-col gap-4"),
			Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
				P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(intl.T(chatI18nNamespace, "modal.displayName"))),
				P(Class("mt-1 text-sm text-white/55"), Text(intl.T(chatI18nNamespace, "modal.displayNamePlaceholder"))),
				Input(
					ID(idNameInput),
					Type("text"),
					Class("mt-4 w-full rounded-xl border border-white/20 bg-[#3a3a3a] px-4 py-3 text-sm text-white placeholder:text-white/40 focus:outline-none"),
					Placeholder(intl.T(chatI18nNamespace, "modal.displayNamePlaceholder")),
					Value(view.NameInput),
					OnInput(profileSettings.HandleNameInput),
					OnKeyDown(profileSettings.HandleNameKey),
				),
			),
		)
	}
}

func settingsSectionTitle(intl i18n.Runtime, activeSection string) string {
	switch activeSection {
	case settingsSectionTone:
		return intl.T(chatI18nNamespace, "modal.aiTone")
	case settingsSectionPrompt:
		return intl.T(chatI18nNamespace, "modal.systemPrompt")
	case settingsSectionIntelligence:
		return intl.T(chatI18nNamespace, "modal.intelligence")
	case settingsSectionMemories:
		return intl.T(chatI18nNamespace, "modal.memories")
	case settingsSectionLanguage:
		return intl.T(chatI18nNamespace, "modal.language")
	default:
		return intl.T(chatI18nNamespace, "modal.displayName")
	}
}

func settingsSectionDescription(intl i18n.Runtime, view appViewState, activeSection, currentThinkingMode string) string {
	switch activeSection {
	case settingsSectionTone:
		return toneDescription(intl, view.ToneInput)
	case settingsSectionPrompt:
		return intl.T(chatI18nNamespace, "modal.systemPromptHelp")
	case settingsSectionIntelligence:
		if !view.ThinkingSupported {
			return intl.T(chatI18nNamespace, "modal.intelligenceUnavailable")
		}
		return thinkingEffortLabel(intl, currentThinkingMode)
	case settingsSectionMemories:
		return intl.T(chatI18nNamespace, "modal.memoriesHelp")
	case settingsSectionLanguage:
		return localeLabel(view.LocaleInput)
	default:
		return intl.T(chatI18nNamespace, "modal.displayNamePlaceholder")
	}
}

func settingsSectionEyebrow(activeSection string) string {
	switch activeSection {
	case settingsSectionTone:
		return "Tone"
	case settingsSectionPrompt:
		return "Prompt"
	case settingsSectionIntelligence:
		return "Intelligence"
	case settingsSectionMemories:
		return "Memories"
	case settingsSectionLanguage:
		return "Language"
	default:
		return "Profile"
	}
}

func newChatHandler(app ui.Reducer[appState, appAction], scrollMemory threadScrollMemory, onNavigateRoot func()) ui.Handler {
	return ui.UseEvent(func() {
		currentState := app.Get()
		if currentState.Streaming {
			return
		}
		scrollMemory.CancelPendingPersist()
		scrollMemory.PersistNow(currentState.ActiveConvID)
		scrollMemory.PrepareRestore(0)
		chatLog.Info("new chat", logging.Fields{"prev_conv_id": currentState.ActiveConvID, "messages": len(currentState.Messages)})
		app.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
		app.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
		app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
		app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
		app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		if onNavigateRoot != nil {
			onNavigateRoot()
		}
	})
}

func toggleSidebarHandler(sidebarOpenState state.Atom[bool]) ui.Handler {
	return ui.UseEvent(func() {
		sidebarOpenState.Set(!sidebarOpenState.Get())
	})
}

func toggleThoughtSectionHandler(app ui.Reducer[appState, appAction]) ui.Handler {
	return ui.UseEvent(func(e ui.Event) {
		sectionKey := strings.TrimSpace(e.JSValue().Get("currentTarget").Get("dataset").Get(dataThoughtSection).String())
		if sectionKey == "" {
			return
		}
		app.Dispatch(appAction{Type: appActionToggleThoughtSection, ThoughtSectionKey: sectionKey})
	})
}
