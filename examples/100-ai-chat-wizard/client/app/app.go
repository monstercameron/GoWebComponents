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
func App() ui.Node {
	intl := i18n.UseI18n()
	app := ui.UseReducer(reduceAppState, initialAppState())
	currentState := app.Get()

	markdownRenderVersion := ui.UseState(0)
	markdownRenderTick := markdownRenderVersion.Get()
	userNameState := state.UseAtom("chat-wizard:user-name", "User")
	sidebarOpenState := state.UseAtom("chat-wizard:sidebar-open", true)
	userName := userNameState.Get()
	sidebarOpen := sidebarOpenState.Get()
	nav := router.UseNavigate()
	threadRoutePublicID := currentThreadRoutePublicID()
	canvasRouteID := currentCanvasRouteID()

	chatClientRef := ui.UseRef[chatpb.ChatServiceClient](nil)
	markdownWorkerRef := ui.UseRef[*interop.Worker](nil)
	markdownRenderInFlight := ui.UseRef(map[string]bool{})

	userNameFetchedAt := ui.UseRef(time.Time{})
	convListFetchedAt := ui.UseRef(time.Time{})
	pendingRootRouteWarningConvID := ui.UseRef(int64(0))
	pendingThreadRouteResolution := ui.UseRef("")
	lastRouteMismatchWarning := ui.UseRef("")
	lastRouteNormalizationWarning := ui.UseRef("")
	settingsReturnRouteRef := ui.UseRef("")
	currentPath := router.GetCurrentPath()
	settingsPanelRouteID := currentSettingsPanelRouteID()

	completedMarkdownSignature := ui.UseMemo(func() string {
		return completedAssistantMessagesMarkdownSignature(currentState.Messages)
	}, currentState.Messages)

	threadCostSummary := ui.UseMemo(func() threadCostSummary {
		return deriveThreadCostSummary(currentState.Messages, currentState.ModelOptions)
	}, currentState.Messages, currentState.ModelOptions)

	scrollMemory := useThreadScrollMemory(currentState.ActiveConvID, len(currentState.Messages))
	modelCatalogState := modelCatalog{DefaultModel: currentState.DefaultModel, Models: currentState.ModelOptions}

	ttsAudio := useTTSAudio(currentState.ActiveConvID, modelCatalogState, chatClientRef)
	redirectToAuthLanding := func() {
		nav.Replace(authLandingRoute)
	}
	handleAuthFailure := func(err error) bool {
		if !handleUnauthenticatedRPC(app, userNameState, err) {
			return false
		}
		redirectToAuthLanding()
		return true
	}
	modelPreferences := useModelPreferences(app, chatClientRef, handleAuthFailure)
	speechModalOpen := ui.UseState(false)
	speechModalError := ui.UseState("")

	profileSettings := useProfileSettings(intl, app, userNameState, chatClientRef, nav, currentPath, settingsPanelRouteID, settingsReturnRouteRef, userNameFetchedAt, modelPreferences.SelectedToneCache, modelPreferences.SelectedThinkingEnabledCache, modelPreferences.SelectedThinkingEffortCache, handleAuthFailure)

	conversationList := useConversationList(app, chatClientRef, convListFetchedAt, scrollMemory, func(publicID string) {
		path := chatThreadPath(publicID)
		if router.GetCurrentPath() != path {
			nav.Navigate(path)
		}
	}, handleAuthFailure)

	authSession := useAuthSession(app, userNameState, chatClientRef, func() {
		if !isChatRoute(router.GetCurrentPath()) {
			nav.Replace(chatRouteRoot)
		}
		conversationList.Refresh(true)
		profileSettings.Refresh(true)
	}, func() {
		redirectToAuthLanding()
	})

	chatStream := useChatStream(intl, app, chatClientRef, scrollMemory, nil, func(newConvID int64) {
		conversationList.Refresh(true)
	}, func(publicID string) {
		path := chatThreadPath(publicID)
		if router.GetCurrentPath() != path {
			nav.Navigate(path)
		}
	}, handleAuthFailure)
	quoteSelection := useQuoteSelection(app)
	canvasWorkspace := useCanvasWorkspace(app, nav, sidebarOpenState, threadRoutePublicID, canvasRouteID)
	requestSpeechUpgrade := func() {
		speechModalError.Set("")
		speechModalOpen.Set(true)
	}
	cancelSpeechUpgrade := ui.UseEvent(func() {
		speechModalError.Set("")
		speechModalOpen.Set(false)
	})
	confirmSpeechUpgrade := ui.UseEvent(func() {
		if modelPreferences.SwitchToSpeechProvider != nil && modelPreferences.SwitchToSpeechProvider() {
			speechModalError.Set("")
			speechModalOpen.Set(false)
			return
		}
		speechModalError.Set(intl.T(chatI18nNamespace, "modal.speechProviderUnavailable"))
	})

	useAppRuntime(
		app,
		chatClientRef,
		markdownWorkerRef,
		markdownRenderInFlight,
		markdownRenderVersion,
		markdownRenderTick,
		completedMarkdownSignature,
		conversationList.Refresh,
		profileSettings.Refresh,
	)

	ui.UseEffect(func() func() {
		if !currentState.Authenticated || !currentState.GRPCReady || currentState.Streaming || isSettingsRoute(currentPath) {
			pendingThreadRouteResolution.Set("")
			lastRouteMismatchWarning.Set("")
			return nil
		}
		switch {
		case shouldResolveConversationRoute(threadRoutePublicID, currentState.ActiveConvPublicID):
			if mismatchKey := routeSyncMismatchKey(threadRoutePublicID, currentState.ActiveConvPublicID); mismatchKey != "" && lastRouteMismatchWarning.Get() != mismatchKey {
				chatLog.Warn("route mismatch detected; resolving requested thread route", logging.Fields{
					"requested_public_id": threadRoutePublicID,
					"active_public_id":    currentState.ActiveConvPublicID,
					"active_conv_id":      currentState.ActiveConvID,
					"conversation_list":   len(currentState.ConversationList),
					"path":                currentPath,
				})
				lastRouteMismatchWarning.Set(mismatchKey)
			}
			if pendingThreadRouteResolution.Get() == threadRoutePublicID {
				return nil
			}
			pendingThreadRouteResolution.Set(threadRoutePublicID)
			pendingRootRouteWarningConvID.Set(0)
			conversationList.ResolveRoute(threadRoutePublicID)
		case shouldResetDraftForRootRoute(threadRoutePublicID, currentState.ActiveConvID, currentState.ActiveConvPublicID):
			pendingThreadRouteResolution.Set("")
			lastRouteMismatchWarning.Set("")
			pendingRootRouteWarningConvID.Set(0)
			scrollMemory.CancelPendingPersist()
			scrollMemory.PersistNow(currentState.ActiveConvID)
			scrollMemory.PrepareRestore(0)
			app.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
			app.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
			app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
			app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
			app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
		case shouldWarnPendingRootRoute(threadRoutePublicID, currentState.ActiveConvID, currentState.ActiveConvPublicID):
			pendingThreadRouteResolution.Set("")
			lastRouteMismatchWarning.Set("")
			if pendingRootRouteWarningConvID.Get() != currentState.ActiveConvID {
				chatLog.Warn("route sync deferred for unresolved conversation public id", logging.Fields{
					"conv_id":           currentState.ActiveConvID,
					"conversation_list": len(currentState.ConversationList),
				})
				pendingRootRouteWarningConvID.Set(currentState.ActiveConvID)
			}
		default:
			pendingThreadRouteResolution.Set("")
			lastRouteMismatchWarning.Set("")
			pendingRootRouteWarningConvID.Set(0)
		}
		return nil
	}, currentState.Authenticated, currentState.GRPCReady, currentState.Streaming, currentState.ActiveConvID, currentState.ActiveConvPublicID, threadRoutePublicID, currentState.ConversationList, currentPath)

	ui.UseEffect(func() func() {
		if currentState.ActiveConvID <= 0 {
			return nil
		}
		publicID := summaryPublicIDForID(currentState.ConversationList, currentState.ActiveConvID)
		if publicID != "" && publicID != currentState.ActiveConvPublicID {
			app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: currentState.ActiveConvID, ActiveConvPublicID: publicID})
		}
		return nil
	}, currentState.ActiveConvID, currentState.ActiveConvPublicID, currentState.ConversationList)

	ui.UseEffect(func() func() {
		if !currentState.Authenticated || isSettingsRoute(currentPath) {
			lastRouteNormalizationWarning.Set("")
			return nil
		}
		if !shouldNormalizeActiveConversationRoute(threadRoutePublicID, currentState.ActiveConvPublicID) {
			mismatchKey := routeSyncMismatchKey(threadRoutePublicID, currentState.ActiveConvPublicID)
			if mismatchKey != "" && lastRouteNormalizationWarning.Get() != mismatchKey {
				chatLog.Warn("suppressed route normalization while requested thread differs from active conversation", logging.Fields{
					"requested_public_id": threadRoutePublicID,
					"active_public_id":    currentState.ActiveConvPublicID,
					"active_conv_id":      currentState.ActiveConvID,
					"path":                currentPath,
				})
				lastRouteNormalizationWarning.Set(mismatchKey)
			}
			if mismatchKey == "" {
				lastRouteNormalizationWarning.Set("")
			}
			return nil
		}
		lastRouteNormalizationWarning.Set("")
		expectedPath := chatThreadPath(currentState.ActiveConvPublicID)
		if strings.TrimSpace(canvasRouteID) != "" && currentState.CanvasSession.Active {
			expectedPath = chatCanvasPath(currentState.ActiveConvPublicID, currentState.CanvasSession.ArtifactID)
		}
		if router.GetCurrentPath() != expectedPath {
			nav.Replace(expectedPath)
		}
		return nil
	}, currentState.Authenticated, currentState.ActiveConvPublicID, currentState.CanvasSession.Active, currentState.CanvasSession.ArtifactID, canvasRouteID, threadRoutePublicID, currentPath)

	view := deriveAppViewState(currentState, userName, sidebarOpen, threadCostSummary, strings.TrimSpace(canvasRouteID) != "")

	return renderAppShell(appShellProps{
		Intl:                 intl,
		View:                 view,
		ResetChat:            newChatHandler(app, scrollMemory, func() { nav.Navigate(chatRouteRoot) }),
		ToggleSidebar:        toggleSidebarHandler(sidebarOpenState),
		ToggleThoughtSection: toggleThoughtSectionHandler(app),
		RequestSpeechUpgrade: requestSpeechUpgrade,
		ShowSpeechModal:      speechModalOpen.Get(),
		SpeechModalError:     speechModalError.Get(),
		ConfirmSpeechModal:   confirmSpeechUpgrade,
		CancelSpeechModal:    cancelSpeechUpgrade,
		StopBubble:           ui.UseEvent(func(e ui.Event) { e.StopPropagation() }),
		ConversationList:     conversationList,
		ChatStream:           chatStream,
		ProfileSettings:      profileSettings,
		ModelPreferences:     modelPreferences,
		AuthSession:          authSession,
		QuoteSelection:       quoteSelection,
		TTSAudio:             ttsAudio,
		ScrollMemory:         scrollMemory,
		CanvasWorkspace:      canvasWorkspace,
	})
}

// --- entry point -------------------------------------------------------------

func Run() {
	r := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: chatRouteRoot})
	r.Register(authLandingRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(marketingHomeRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(marketingCapabilitiesRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(marketingPricingRoute, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(chatRouteRoot, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(chatRouteThreadPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(chatRouteCanvasPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(settingsRoutePath, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register("*", func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Mount(appSelector)
	utils.WaitForever()
}
