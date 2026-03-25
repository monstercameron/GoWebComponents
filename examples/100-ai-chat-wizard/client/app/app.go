//go:build js && wasm

package app

import (
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// App renders the chat wizard application shell.
func ParseApp() ui.Node {
	parseIntl := i18n.UseI18n()
	parseApp := ui.UseReducer(parseReduceAppState, parseInitialAppState())
	parseCurrentState := parseApp.Get()

	parseMarkdownRenderVersion := ui.UseState(0)
	parseMarkdownRenderTick := parseMarkdownRenderVersion.Get()
	parseUserNameState := state.UseAtom("chat-wizard:user-name", "User")
	parseSidebarOpenState := state.UseAtom("chat-wizard:sidebar-open", true)
	parseTtsProviderState := state.UseAtom("chat-wizard:tts-provider", defaultTTSProvider)
	parseStoredTTSProvider := parseResolveTTSProviderID(parseTtsProviderState.Get())
	parseUserName := parseUserNameState.Get()
	parseSidebarOpen := parseSidebarOpenState.Get()
	parseNav := router.UseNavigate()
	parseThreadRoutePublicID := parseCurrentThreadRoutePublicID()
	parseCanvasRouteID := parseCurrentCanvasRouteID()

	parseChatClientRef := ui.UseRef[chatpb.ChatServiceClient](nil)
	parseMarkdownWorkerRef := ui.UseRef[*interop.Worker](nil)
	parseMarkdownRenderInFlight := ui.UseRef(map[string]bool{})

	parseUserNameFetchedAt := ui.UseRef(time.Time{})
	parseConvListFetchedAt := ui.UseRef(time.Time{})
	parsePendingRootRouteWarningConvID := ui.UseRef(int64(0))
	parsePendingThreadRouteResolution := ui.UseRef("")
	parseLastRouteMismatchWarning := ui.UseRef("")
	parseLastRouteNormalizationWarning := ui.UseRef("")
	parseSettingsReturnRouteRef := ui.UseRef("")
	parseCurrentPath := router.GetCurrentPath()
	parseSettingsPanelRouteID := parseCurrentSettingsPanelRouteID()
	parseDeferredMessages := ui.UseDeferredValue(parseCurrentState.Messages)

	parseCompletedMarkdownSignature := ui.UseMemo(func() string {
		return parseCompletedAssistantMessagesMarkdownSignature(parseDeferredMessages)
	}, parseDeferredMessages)

	parseThreadCostSummary := ui.UseMemo(func() threadCostSummary {
		return parseDeriveThreadCostSummary(parseDeferredMessages, parseCurrentState.ParseModelOptions)
	}, parseDeferredMessages, parseCurrentState.ParseModelOptions)

	parseScrollMemory := parseUseThreadScrollMemory(parseCurrentState.ActiveConvID, len(parseCurrentState.Messages))
	parseModelCatalogState := modelCatalog{DefaultModel: parseCurrentState.ParseDefaultModel, Models: parseCurrentState.ParseModelOptions}

	parseTtsAudio := parseUseTTSAudio(parseCurrentState.ActiveConvID, parseModelCatalogState, parseChatClientRef, parseCurrentState.SelectedTTSProvider)
	parseRedirectToAuthLanding := func() {
		parseNav.Replace(authLandingRoute)
	}
	handleAuthFailure := func(parseErr error) bool {
		if !handleUnauthenticatedRPC(parseApp, parseUserNameState, parseErr) {
			return false
		}
		parseRedirectToAuthLanding()
		return true
	}
	parseModelPreferences := parseUseModelPreferences(parseApp, parseChatClientRef, handleAuthFailure)
	parseSpeechModalOpen := ui.UseState(false)
	parseSpeechModalError := ui.UseState("")

	parseProfileSettings := parseUseProfileSettings(parseIntl, parseApp, parseUserNameState, parseTtsProviderState, parseChatClientRef, parseNav, parseCurrentPath, parseSettingsPanelRouteID, parseSettingsReturnRouteRef, parseUserNameFetchedAt, parseModelPreferences.SelectedToneCache, parseModelPreferences.SelectedThinkingEnabledCache, parseModelPreferences.SelectedThinkingEffortCache, handleAuthFailure)

	parseConversationList := parseUseConversationList(parseApp, parseChatClientRef, parseConvListFetchedAt, parseScrollMemory, func(parsePublicID2 string) {
		parsePath := parseChatThreadPath(parsePublicID2)
		if router.GetCurrentPath() != parsePath {
			parseNav.Navigate(parsePath)
		}
	}, handleAuthFailure)
	parseAccountCostSummary := parseUseAccountCostSummary(parseCurrentState, parseChatClientRef, handleAuthFailure)

	parseAuthSession := parseUseAuthSession(parseApp, parseUserNameState, parseChatClientRef, func() {
		if !isChatRoute(router.GetCurrentPath()) {
			parseNav.Replace(chatRouteRoot)
		}
		parseConversationList.Refresh(true)
		parseProfileSettings.Refresh(true)
	}, func() {
		parseRedirectToAuthLanding()
	})

	parseChatStream := parseUseChatStream(parseIntl, parseApp, parseChatClientRef, parseScrollMemory, nil, func(parseNewConvID int64) {
		parseConversationList.Refresh(true)
	}, func(parsePublicID3 string) {
		parsePath2 := parseChatThreadPath(parsePublicID3)
		if router.GetCurrentPath() != parsePath2 {
			parseNav.Navigate(parsePath2)
		}
	}, handleAuthFailure)
	parseQuoteSelection := parseUseQuoteSelection(parseApp)
	parseCanvasWorkspace := parseUseCanvasWorkspace(parseApp, parseNav, parseSidebarOpenState, parseThreadRoutePublicID, parseCanvasRouteID)
	parseRequestSpeechUpgrade := func() {
		parseSpeechModalError.Set("")
		parseSpeechModalOpen.Set(true)
	}
	parseCancelSpeechUpgrade := ui.UseEvent(func() {
		parseSpeechModalError.Set("")
		parseSpeechModalOpen.Set(false)
	})
	parseConfirmSpeechUpgrade := ui.UseEvent(func() {
		parseCurrentState2 := parseApp.Get()
		if !parseTtsProviderSupportsSpeech(ttsProviderOpenAI, parseCurrentState2.ParseModelOptions, parseCurrentState2.ParseDefaultModel) {
			parseSpeechModalError.Set(parseIntl.T(chatI18nNamespace, "modal.speechProviderUnavailable"))
			return
		}
		parseTtsProviderState.Set(ttsProviderOpenAI)
		parseApp.Dispatch(appAction{Type: appActionSetSelectedTTSProvider, SelectedTTSProvider: ttsProviderOpenAI})
		parseApp.Dispatch(appAction{Type: appActionSetTTSProviderInput, TTSProviderInput: ttsProviderOpenAI})
		parseSpeechModalError.Set("")
		parseSpeechModalOpen.Set(false)
	})

	ui.UseEffect(func() func() {
		if parseStoredTTSProvider == parseCurrentState.SelectedTTSProvider {
			return nil
		}
		parseApp.Dispatch(appAction{Type: appActionSetSelectedTTSProvider, SelectedTTSProvider: parseStoredTTSProvider})
		if !parseCurrentState.ShowNameModal {
			parseApp.Dispatch(appAction{Type: appActionSetTTSProviderInput, TTSProviderInput: parseStoredTTSProvider})
		}
		return nil
	}, parseStoredTTSProvider, parseCurrentState.SelectedTTSProvider, parseCurrentState.ShowNameModal)

	parseUseAppRuntime(
		parseApp,
		parseChatClientRef,
		parseMarkdownWorkerRef,
		parseMarkdownRenderInFlight,
		parseMarkdownRenderVersion,
		parseMarkdownRenderTick,
		parseCompletedMarkdownSignature,
		parseConversationList.Refresh,
		parseProfileSettings.Refresh,
	)

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady || len(parseCurrentState.ParseModelOptions) > 0 {
			return nil
		}
		if parseModelPreferences.RefreshCatalog != nil {
			parseModelPreferences.RefreshCatalog()
		}
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, len(parseCurrentState.ParseModelOptions))

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady || parseCurrentState.Streaming || isSettingsRoute(parseCurrentPath) {
			parsePendingThreadRouteResolution.Set("")
			parseLastRouteMismatchWarning.Set("")
			return nil
		}
		switch {
		case shouldResolveConversationRoute(parseThreadRoutePublicID, parseCurrentState.ActiveConvPublicID):
			if parseMismatchKey := parseRouteSyncMismatchKey(parseThreadRoutePublicID, parseCurrentState.ActiveConvPublicID); parseMismatchKey != "" && parseLastRouteMismatchWarning.Get() != parseMismatchKey {
				chatLog.Warn("route mismatch detected; resolving requested thread route", logging.Fields{
					"requested_public_id": parseThreadRoutePublicID,
					"active_public_id":    parseCurrentState.ActiveConvPublicID,
					"active_conv_id":      parseCurrentState.ActiveConvID,
					"conversation_list":   len(parseCurrentState.ConversationList),
					"path":                parseCurrentPath,
				})
				parseLastRouteMismatchWarning.Set(parseMismatchKey)
			}
			if parsePendingThreadRouteResolution.Get() == parseThreadRoutePublicID {
				return nil
			}
			parsePendingThreadRouteResolution.Set(parseThreadRoutePublicID)
			parsePendingRootRouteWarningConvID.Set(0)
			parseConversationList.ResolveRoute(parseThreadRoutePublicID)
		case shouldResetDraftForRootRoute(parseThreadRoutePublicID, parseCurrentState.ActiveConvID, parseCurrentState.ActiveConvPublicID):
			parsePendingThreadRouteResolution.Set("")
			parseLastRouteMismatchWarning.Set("")
			parsePendingRootRouteWarningConvID.Set(0)
			parseScrollMemory.CancelPendingPersist()
			parseScrollMemory.ParsePersistNow(parseCurrentState.ActiveConvID)
			parseScrollMemory.ParsePrepareRestore(0)
			parseApp.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
			parseApp.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
			parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
			parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
			parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		case shouldWarnPendingRootRoute(parseThreadRoutePublicID, parseCurrentState.ActiveConvID, parseCurrentState.ActiveConvPublicID):
			parsePendingThreadRouteResolution.Set("")
			parseLastRouteMismatchWarning.Set("")
			if parsePendingRootRouteWarningConvID.Get() != parseCurrentState.ActiveConvID {
				chatLog.Warn("route sync deferred for unresolved conversation public id", logging.Fields{
					"conv_id":           parseCurrentState.ActiveConvID,
					"conversation_list": len(parseCurrentState.ConversationList),
				})
				parsePendingRootRouteWarningConvID.Set(parseCurrentState.ActiveConvID)
			}
		default:
			parsePendingThreadRouteResolution.Set("")
			parseLastRouteMismatchWarning.Set("")
			parsePendingRootRouteWarningConvID.Set(0)
		}
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, parseCurrentState.Streaming, parseCurrentState.ActiveConvID, parseCurrentState.ActiveConvPublicID, parseThreadRoutePublicID, parseCurrentState.ConversationList, parseCurrentPath)

	ui.UseEffect(func() func() {
		if parseCurrentState.ActiveConvID <= 0 {
			return nil
		}
		parsePublicID := parseSummaryPublicIDForID(parseCurrentState.ConversationList, parseCurrentState.ActiveConvID)
		if parsePublicID != "" && parsePublicID != parseCurrentState.ActiveConvPublicID {
			parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: parseCurrentState.ActiveConvID, ActiveConvPublicID: parsePublicID})
		}
		return nil
	}, parseCurrentState.ActiveConvID, parseCurrentState.ActiveConvPublicID, parseCurrentState.ConversationList)

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || isSettingsRoute(parseCurrentPath) {
			parseLastRouteNormalizationWarning.Set("")
			return nil
		}
		if !shouldNormalizeActiveConversationRoute(parseThreadRoutePublicID, parseCurrentState.ActiveConvPublicID) {
			parseMismatchKey2 := parseRouteSyncMismatchKey(parseThreadRoutePublicID, parseCurrentState.ActiveConvPublicID)
			if parseMismatchKey2 != "" && parseLastRouteNormalizationWarning.Get() != parseMismatchKey2 {
				chatLog.Warn("suppressed route normalization while requested thread differs from active conversation", logging.Fields{
					"requested_public_id": parseThreadRoutePublicID,
					"active_public_id":    parseCurrentState.ActiveConvPublicID,
					"active_conv_id":      parseCurrentState.ActiveConvID,
					"path":                parseCurrentPath,
				})
				parseLastRouteNormalizationWarning.Set(parseMismatchKey2)
			}
			if parseMismatchKey2 == "" {
				parseLastRouteNormalizationWarning.Set("")
			}
			return nil
		}
		parseLastRouteNormalizationWarning.Set("")
		parseExpectedPath := parseChatThreadPath(parseCurrentState.ActiveConvPublicID)
		if strings.TrimSpace(parseCanvasRouteID) != "" && parseCurrentState.CanvasSession.Active {
			parseExpectedPath = parseChatCanvasPath(parseCurrentState.ActiveConvPublicID, parseCurrentState.CanvasSession.ArtifactID)
		}
		if router.GetCurrentPath() != parseExpectedPath {
			parseNav.Replace(parseExpectedPath)
		}
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.ActiveConvPublicID, parseCurrentState.CanvasSession.Active, parseCurrentState.CanvasSession.ArtifactID, parseCanvasRouteID, parseThreadRoutePublicID, parseCurrentPath)

	parseView := parseDeriveAppViewState(parseCurrentState, parseUserName, parseSidebarOpen, parseThreadCostSummary, parseAccountCostSummary, strings.TrimSpace(parseCanvasRouteID) != "")

	return renderAppShell(appShellProps{
		Intl:                 parseIntl,
		View:                 parseView,
		ResetChat:            parseNewChatHandler(parseApp, parseScrollMemory, func() { parseNav.Navigate(chatRouteRoot) }),
		ToggleSidebar:        parseToggleSidebarHandler(parseSidebarOpenState),
		ToggleThoughtSection: parseToggleThoughtSectionHandler(parseApp),
		RequestSpeechUpgrade: parseRequestSpeechUpgrade,
		ShowSpeechModal:      parseSpeechModalOpen.Get(),
		SpeechModalError:     parseSpeechModalError.Get(),
		ConfirmSpeechModal:   parseConfirmSpeechUpgrade,
		CancelSpeechModal:    parseCancelSpeechUpgrade,
		StopBubble:           ui.UseEvent(func(parseE ui.Event) { parseE.StopPropagation() }),
		ConversationList:     parseConversationList,
		ChatStream:           parseChatStream,
		ProfileSettings:      parseProfileSettings,
		ModelPreferences:     parseModelPreferences,
		AuthSession:          parseAuthSession,
		QuoteSelection:       parseQuoteSelection,
		TTSAudio:             parseTtsAudio,
		ScrollMemory:         parseScrollMemory,
		CanvasWorkspace:      parseCanvasWorkspace,
	})
}

// --- entry point -------------------------------------------------------------

func ParseRun() {
	parseR := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: chatRouteRoot})
	parseR.Register(authLandingRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(marketingHomeRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(marketingCapabilitiesRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(marketingPricingRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(chatRouteRoot, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(chatRouteThreadPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(chatRouteCanvasPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register(settingsRoutePath, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Register("*", func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot)
	})
	parseR.Mount(appSelector)
	utils.WaitForever()
}
