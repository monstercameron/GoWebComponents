//go:build js && wasm

package app

import (
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/interop"
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

	completedMarkdownSignature := ui.UseMemo(func() string {
		return completedAssistantMessagesMarkdownSignature(currentState.Messages)
	}, currentState.Messages)

	threadCostSummary := ui.UseMemo(func() threadCostSummary {
		return deriveThreadCostSummary(currentState.Messages)
	}, currentState.Messages)

	scrollMemory := useThreadScrollMemory(currentState.ActiveConvID, len(currentState.Messages))
	modelCatalogState := modelCatalog{DefaultModel: currentState.DefaultModel, Models: currentState.ModelOptions}

	ttsAudio := useTTSAudio(currentState.ActiveConvID, modelCatalogState, chatClientRef)
	handleAuthFailure := func(err error) bool {
		return handleUnauthenticatedRPC(app, userNameState, err)
	}
	modelPreferences := useModelPreferences(app, chatClientRef, handleAuthFailure)

	profileSettings := useProfileSettings(intl, app, userNameState, chatClientRef, userNameFetchedAt, modelPreferences.SelectedToneCache, modelPreferences.SelectedThinkingEnabledCache, modelPreferences.SelectedThinkingEffortCache, handleAuthFailure)

	conversationList := useConversationList(app, chatClientRef, convListFetchedAt, scrollMemory, func() {
		modelPreferences.ResetDraftModel()
	}, func(publicID string) {
		path := chatThreadPath(publicID)
		if router.GetCurrentPath() != path {
			nav.Navigate(path)
		}
	}, handleAuthFailure)

	authSession := useAuthSession(app, userNameState, chatClientRef, func() {
		conversationList.Refresh(true)
		profileSettings.Refresh(true)
	}, func() {
		modelPreferences.ResetDraftModel()
	})

	chatStream := useChatStream(intl, app, chatClientRef, scrollMemory, func() {
		modelPreferences.ResetDraftModel()
	}, func(newConvID int64) {
		conversationList.Refresh(true)
	}, func(publicID string) {
		path := chatThreadPath(publicID)
		if router.GetCurrentPath() != path {
			nav.Navigate(path)
		}
	}, handleAuthFailure)
	quoteSelection := useQuoteSelection(app)
	canvasWorkspace := useCanvasWorkspace(app, nav, threadRoutePublicID, canvasRouteID)

	useAppRuntime(
		app,
		chatClientRef,
		markdownWorkerRef,
		markdownRenderInFlight,
		markdownRenderVersion,
		completedMarkdownSignature,
		conversationList.Refresh,
		profileSettings.Refresh,
	)

	ui.UseEffect(func() func() {
		if !currentState.Authenticated || !currentState.GRPCReady || currentState.Streaming {
			return nil
		}
		switch {
		case threadRoutePublicID != "" && currentState.ActiveConvPublicID != threadRoutePublicID:
			conversationList.ResolveRoute(threadRoutePublicID)
		case threadRoutePublicID == "" && currentState.ActiveConvID > 0:
			scrollMemory.CancelPendingPersist()
			scrollMemory.PersistNow(currentState.ActiveConvID)
			scrollMemory.PrepareRestore(0)
			app.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
			app.Dispatch(appAction{Type: appActionSetInputText, InputText: ""})
			app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
			app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
			app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
			modelPreferences.ResetDraftModel()
		}
		return nil
	}, currentState.Authenticated, currentState.GRPCReady, currentState.Streaming, currentState.ActiveConvID, currentState.ActiveConvPublicID, threadRoutePublicID, currentState.ConversationList)

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
		if !currentState.Authenticated || currentState.ActiveConvPublicID == "" {
			return nil
		}
		expectedPath := chatThreadPath(currentState.ActiveConvPublicID)
		if strings.TrimSpace(canvasRouteID) != "" && currentState.CanvasSession.Active {
			expectedPath = chatCanvasPath(currentState.ActiveConvPublicID, currentState.CanvasSession.ArtifactID)
		}
		if router.GetCurrentPath() != expectedPath {
			nav.Replace(expectedPath)
		}
		return nil
	}, currentState.Authenticated, currentState.ActiveConvPublicID, currentState.CanvasSession.Active, currentState.CanvasSession.ArtifactID, canvasRouteID)

	view := deriveAppViewState(currentState, userName, sidebarOpen, threadCostSummary, strings.TrimSpace(canvasRouteID) != "")

	return renderAppShell(appShellProps{
		Intl:                 intl,
		View:                 view,
		ResetChat:            newChatHandler(app, scrollMemory, modelPreferences.ResetDraftModel, func() { nav.Navigate(chatRouteRoot) }),
		ToggleSidebar:        toggleSidebarHandler(sidebarOpenState),
		ToggleThoughtSection: toggleThoughtSectionHandler(app),
		StopBubble:           ui.UseEvent(func(e ui.Event) { e.StopPropagation() }),
		ConversationList:     conversationList,
		ChatStream:           chatStream,
		ProfileSettings:      profileSettings,
		ModelPreferences:     modelPreferences,
		AuthSession:          authSession,
		QuoteSelection:       quoteSelection,
		TTSAudio:             ttsAudio,
		CanvasWorkspace:      canvasWorkspace,
	})
}

// --- entry point -------------------------------------------------------------

func Run() {
	r := router.NewRouter(router.RouterOptions{DefaultRoute: chatRouteRoot})
	r.Register(chatRouteRoot, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(chatRouteThreadPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register(chatRouteCanvasPattern, func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Register("*", func(router.Attrs) *router.Element {
		return ui.CreateElement(chatWizardRoot)
	})
	r.Mount(appSelector)
	utils.WaitForever()
}
