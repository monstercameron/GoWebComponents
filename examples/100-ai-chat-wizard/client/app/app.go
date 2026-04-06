//go:build js && wasm

package app

import (
	"context"
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

// ParseApp renders the chat wizard application shell.
func ParseApp(parseProps chatWizardRouteProps) ui.Node {
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
	// Fetch the server-owned catalog bootstrap once gRPC is ready and merge it into
	// chatWizardBundle so server-managed copy replaces the client emergency fallback.
	parseUseCatalogServerSync(parseCurrentState, parseChatClientRef, parseIntl.Locale())
	parseMarkdownWorkerRef := ui.UseRef[*interop.Worker](nil)
	parseMarkdownWorkerPoolRef := ui.UseRef[*interop.WorkerPool](nil)
	parseMarkdownRenderInFlight := ui.UseRef(map[string]bool{})
	parseThoughtCacheByMessageState := ui.UseState(map[int]renderWorkerThoughtCacheEntry{})
	parseCanvasCacheByMessageState := ui.UseState(map[int]renderWorkerCanvasCacheEntry{})
	parseThreadCostSummaryState := ui.UseState(threadCostSummary{AssistantMessageCosts: map[int]assistantMessageCost{}})
	parseThreadCostSummarySignatureState := ui.UseState("")

	parseUserNameFetchedAt := ui.UseRef(time.Time{})
	parseConvListFetchedAt := ui.UseRef(time.Time{})
	parsePendingRootRouteWarningConvID := ui.UseRef(int64(0))
	parsePendingThreadRouteResolution := ui.UseRef("")
	parseLastRouteMismatchWarning := ui.UseRef("")
	parseLastRouteNormalizationWarning := ui.UseRef("")
	parseSettingsReturnRouteRef := ui.UseRef("")
	parseCurrentPath := strings.TrimSpace(parseProps.CurrentPath)
	if parseCurrentPath == "" {
		parseCurrentPath = router.GetCurrentPath()
	}
	parseUsePricingFragmentScroll(parseCurrentPath)
	parseSettingsPanelRouteID := parseCurrentSettingsPanelRouteID()
	parseDeferredMessages := ui.UseDeferredValue(parseCurrentState.Messages)
	parseRenderSignatureState := ui.UseState(parseBuildRenderSignatureState(parseDeferredMessages, parseCurrentState.ModelOptions))
	parseRenderSignatureGenerationRef := ui.UseRef(uint64(0))
	parseHasWorkerRequester := parseMarkdownWorkerRef.Get() != nil || parseMarkdownWorkerPoolRef.Get() != nil

	ui.UseEffect(func() func() {
		if parseCurrentState.MarkdownWorkerFallback || !parseHasWorkerRequester {
			parseRenderSignatureState.Set(parseBuildRenderSignatureState(parseDeferredMessages, parseCurrentState.ModelOptions))
			return nil
		}
		parseRequester, _ := parseResolveBackgroundRenderRequester(parseMarkdownWorkerRef, parseMarkdownWorkerPoolRef)
		if parseRequester == nil {
			parseRenderSignatureState.Set(parseBuildRenderSignatureState(parseDeferredMessages, parseCurrentState.ModelOptions))
			return nil
		}
		parseGeneration := parseRenderSignatureGenerationRef.Get() + 1
		parseRenderSignatureGenerationRef.Set(parseGeneration)
		// App state updates replace slice snapshots, so this render's message/model
		// views are safe to hand to the async worker request without another clone.
		parseMessages := parseDeferredMessages
		parseModels := parseCurrentState.ModelOptions
		go func(parseExpectedGeneration uint64, parseWorkerMessages []message, parseWorkerModels []modelOption) {
			parseWorkerSignatures, parseErr := parseRequestWorkerRenderSignatures(context.Background(), parseRequester, parseExpectedGeneration, parseWorkerMessages, parseWorkerModels)
			if parseErr != nil {
				chatLog.Warn("render signature worker request failed; using synchronous fallback", logging.Fields{"error": parseErr})
				parseWorkerSignatures = parseBuildRenderSignatureState(parseWorkerMessages, parseWorkerModels)
			}
			if parseRenderSignatureGenerationRef.Get() != parseExpectedGeneration {
				return
			}
			parseRenderSignatureState.Set(parseWorkerSignatures)
		}(parseGeneration, parseMessages, parseModels)
		return nil
	}, parseCurrentState.MarkdownWorkerFallback, parseDeferredMessages, parseCurrentState.ModelOptions, parseHasWorkerRequester)

	parseRenderSignatures := parseRenderSignatureState.Get()
	parseCompletedMarkdownSignature := parseRenderSignatures.GetCompletedMarkdownSignature
	parseAssistantMetadataSignature := parseRenderSignatures.GetAssistantMetadataSignature
	parseThreadCostSignature := parseRenderSignatures.GetThreadCostSignature

	parseThreadCostSummary := parseThreadCostSummaryState.Get()
	if parseThreadCostSummary.AssistantMessageCosts == nil {
		parseThreadCostSummary.AssistantMessageCosts = map[int]assistantMessageCost{}
	}
	if parseThreadCostSummarySignatureState.Get() != parseThreadCostSignature {
		parseThreadCostSummary = threadCostSummary{AssistantMessageCosts: map[int]assistantMessageCost{}}
	}
	if parseCurrentState.MarkdownWorkerFallback {
		parseThreadCostSummary = parseDeriveThreadCostSummary(parseDeferredMessages, parseCurrentState.ModelOptions)
	}

	parseScrollMemory := parseUseThreadScrollMemory(parseCurrentState.ActiveConvID, len(parseCurrentState.Messages))
	parseModelCatalogState := modelCatalog{DefaultModel: parseCurrentState.DefaultModel, Models: parseCurrentState.ModelOptions}

	parseTtsAudio := parseUseTTSAudio(parseCurrentState.ActiveConvID, parseModelCatalogState, parseChatClientRef, parseCurrentState.SelectedTTSProvider)
	parseRedirectToAuthLanding := func(parseIntentPath string) {
		parseStorePostLoginRouteIntent(parseIntentPath)
		parseNav.Replace(authLoginRoute)
	}
	handleAuthFailure := func(parseErr error) bool {
		if !handleUnauthenticatedRPC(parseApp, parseUserNameState, parseErr) {
			return false
		}
		parseRedirectToAuthLanding(strings.TrimSpace(router.GetCurrentPath()))
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
	parseAccountCostSummary := parseUseAccountCostSummary(parseCurrentState, parseChatClientRef, parseMarkdownWorkerRef, parseMarkdownWorkerPoolRef, handleAuthFailure)
	parseAdminDashboard := parseUseAdminDashboard(parseCurrentState, parseChatClientRef, handleAuthFailure)
	parseAdminCustomers := parseUseAdminCustomers(parseCurrentState, parseAdminDashboard, parseChatClientRef, handleAuthFailure)
	parseAdminWorkspaces := parseUseAdminWorkspaces(parseCurrentState, parseAdminDashboard, parseChatClientRef, handleAuthFailure)

	parseAuthSession := parseUseAuthSession(parseApp, parseUserNameState, parseChatClientRef, func(parseSession *chatpb.GetSessionResponse) {
		var parseRoleSummary *chatpb.AuthRoleSummary
		if parseSession != nil {
			parseRoleSummary = parseSession.GetRoleSummary()
		}
		parseApp.Dispatch(appAction{Type: appActionSetCanAccessAdmin, CanAccessAdmin: parseCanAccessAdminFromRoleSummary(parseRoleSummary)})
		parseApp.Dispatch(appAction{Type: appActionSetIsSuperuser, IsSuperuser: parseRoleSummary != nil && parseRoleSummary.GetIsSuperuser()})
		parseCurrentAuthPath := strings.TrimSpace(router.GetCurrentPath())
		parseIntentPath := parseConsumePostLoginRouteIntent()
		parsePostLoginPath := parseResolvePostLoginRoute(parseIntentPath, parseRoleSummary)
		if parseIntentPath != "" || !isChatRoute(parseCurrentAuthPath) {
			if shouldNavigateLandingRoute(parseCurrentAuthPath, parsePostLoginPath) {
				parseNav.Replace(parsePostLoginPath)
			}
		}
		parseConversationList.Refresh(true)
		parseProfileSettings.Refresh(true)
	}, func() {
		parseRedirectToAuthLanding(strings.TrimSpace(router.GetCurrentPath()))
	})

	ui.UseEffect(func() func() {
		parseTimer, parseErr := interop.ScheduleTimeout(6*time.Second, func() {
			if !parseHasBootShell() {
				return
			}
			parseCurrent := parseApp.Get()
			chatLog.Warn("boot shell still mounted after startup window", logging.Fields{
				"path":          strings.TrimSpace(router.GetCurrentPath()),
				"grpc_ready":    parseCurrent.GRPCReady,
				"auth_resolved": parseCurrent.AuthResolved,
				"authenticated": parseCurrent.Authenticated,
			})
		})
		if parseErr != nil {
			return nil
		}
		return func() {
			_ = parseTimer.Cancel()
		}
	}, true)

	ui.UseEffect(func() func() {
		if !shouldRedirectUnauthenticatedRouteToLanding(parseCurrentPath, parseCurrentState.AuthResolved, parseCurrentState.Authenticated) {
			return nil
		}
		if !shouldNavigateLandingRoute(strings.TrimSpace(router.GetCurrentPath()), authLoginRoute) {
			return nil
		}
		chatLog.Warn("auth route guard redirected unauthenticated app route", logging.Fields{"path": parseCurrentPath})
		parseRedirectToAuthLanding(parseCurrentPath)
		return nil
	}, parseCurrentPath, parseCurrentState.AuthResolved, parseCurrentState.Authenticated)

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
		if !parseTtsProviderSupportsSpeech(ttsProviderOpenAI, parseCurrentState2.ModelOptions, parseCurrentState2.DefaultModel) {
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

	// Sync auth mode with the dedicated /signup route so the form always submits as signup.
	ui.UseEffect(func() func() {
		if parseCurrentPath == marketingSignupRoute && parseApp.Get().AuthMode != authModeSignup {
			parseApp.Dispatch(appAction{Type: appActionSetAuthMode, AuthMode: authModeSignup})
		}
		return nil
	}, parseCurrentPath)

	parseUseAppRuntime(
		parseApp,
		parseChatClientRef,
		parseMarkdownWorkerRef,
		parseMarkdownWorkerPoolRef,
		parseMarkdownRenderInFlight,
		parseMarkdownRenderVersion,
		parseMarkdownRenderTick,
		parseCompletedMarkdownSignature,
		parseAssistantMetadataSignature,
		parseThreadCostSignature,
		parseThoughtCacheByMessageState,
		parseCanvasCacheByMessageState,
		parseThreadCostSummaryState,
		parseThreadCostSummarySignatureState,
		parseConversationList.Refresh,
		parseProfileSettings.Refresh,
	)

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady || len(parseCurrentState.ModelOptions) > 0 {
			return nil
		}
		if parseModelPreferences.RefreshCatalog != nil {
			parseModelPreferences.RefreshCatalog()
		}
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, len(parseCurrentState.ModelOptions))

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
		case shouldResetDraftForRootRoute(parseCurrentPath, parseThreadRoutePublicID, parseCurrentState.ActiveConvID, parseCurrentState.ActiveConvPublicID):
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
		case shouldWarnPendingRootRoute(parseCurrentPath, parseThreadRoutePublicID, parseCurrentState.ActiveConvID, parseCurrentState.ActiveConvPublicID):
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
		if !shouldNormalizeActiveConversationRoute(parseCurrentPath, parseThreadRoutePublicID, parseCurrentState.ActiveConvPublicID) {
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

	parseAdminServerTools := parseUseAdminServerTools(parseCurrentState, parseChatClientRef, handleAuthFailure)
	parseView := parseDeriveAppViewState(parseCurrentState, parseCurrentPath, parseUserName, parseSidebarOpen, parseThoughtCacheByMessageState.Get(), parseCanvasCacheByMessageState.Get(), parseThreadCostSummary, parseAccountCostSummary, strings.TrimSpace(parseCanvasRouteID) != "", parseAdminDashboard, parseUseCatalogServerSynced(), parseAdminServerTools)
	parseOpenAdminDashboard := ui.UseEvent(func() {
		if !parseApp.Get().CanAccessAdmin {
			return
		}
		parseNav.Navigate(chatRouteDashboardHome)
	})

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
		OpenAdminDashboard:   parseOpenAdminDashboard,
		ConversationList:     parseConversationList,
		ChatStream:           parseChatStream,
		ProfileSettings:      parseProfileSettings,
		ModelPreferences:     parseModelPreferences,
		AuthSession:          parseAuthSession,
		QuoteSelection:       parseQuoteSelection,
		TTSAudio:             parseTtsAudio,
		ScrollMemory:         parseScrollMemory,
		CanvasWorkspace:      parseCanvasWorkspace,
		AdminCustomers:       parseAdminCustomers,
		AdminWorkspaces:      parseAdminWorkspaces,
	})
}

// --- entry point -------------------------------------------------------------

// ParseRun starts the chat wizard client entrypoint.
func ParseRun() {
	parseRegisterRuntime2Regions()
	parseR := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: chatRouteRoot})
	parseR.Register(authLandingRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(authLoginRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingHomeRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingCapabilitiesRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingPricingRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingAboutRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingContactRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingPrivacyRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingTermsRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingSecurityRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(marketingStatusRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteRoot, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteThreadPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteCanvasPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(settingsRoutePath, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteDashboardHome, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteDashboardBusiness, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteDashboardCustomers, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteDashboardChats, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteDashboardProviders, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register(chatRouteDashboardOps, func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Register("*", func(router.Attrs) *router.Element {
		return ui.CreateElement(parseChatWizardRoot, buildAppRouteProps())
	})
	parseR.Mount(appSelector)
	utils.WaitForever()
}
