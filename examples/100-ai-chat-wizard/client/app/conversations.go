//go:build js && wasm

package app

import (
	"context"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// conversationListController groups the sidebar conversation RPC workflow.
//
// The app shell only needs the resulting handlers and the explicit refresh
// entrypoint; the gRPC list/load/delete details stay local to this hook.
type conversationListController struct {
	Refresh       func(bool)
	LoadByID      func(int64)
	ResolveRoute  func(string)
	Load          ui.Handler
	RequestDelete ui.Handler
	CancelDelete  ui.Handler
	ConfirmDelete ui.Handler
}

// useConversationList hides the conversation-list CRUD flow behind one
// feature-specific hook so App() is not interleaving sidebar RPC details with
// unrelated render and streaming logic.
func parseUseConversationList(
	parseApp ui.Reducer[appState, appAction],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseConvListFetchedAt ui.Ref[time.Time],
	parseScrollMemory threadScrollMemory,
	parseOnNavigateToConversation func(string),
	handleAuthFailure func(error) bool,
) conversationListController {
	parseLoadRequestSeq := ui.UseRef(uint64(0))
	parseResolveRouteSeq := ui.UseRef(uint64(0))

	parseRefresh := func(isForce bool) {
		if !parseApp.Get().Authenticated {
			return
		}
		if !isForce && time.Since(parseConvListFetchedAt.Get()) < convListTTL {
			return
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return
		}
		go func() {
			parseResp, parseErr := parseClient.ListConversations(context.Background(), &chatpb.ListConversationsRequest{})
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				chatLog.Error("list conversations failed", logging.Fields{"error": parseErr})
				return
			}
			parseSummaries := make([]convSummary, 0, len(parseResp.Conversations))
			for _, parseConversation := range parseResp.Conversations {
				parseSummaries = append(parseSummaries, convSummary{
					ID:        parseConversation.Id,
					PublicID:  parseConversation.GetPublicId(),
					StartedAt: parseConversation.StartedAt,
					Preview:   parseConversation.Preview,
				})
			}
			parseApp.Dispatch(appAction{Type: appActionSetConversationList, ConversationList: parseSummaries})
			parseConvListFetchedAt.Set(time.Now())
			if isForce {
				chatLog.Info("conv list", logging.Fields{"count": len(parseSummaries)})
			}
		}()
	}

	parseLoadByID := func(parseId4 int64) {
		parseCurrentState := parseApp.Get()
		if parseCurrentState.Streaming || parseId4 <= 0 {
			return
		}
		parseRequestSeq := parseLoadRequestSeq.Get() + 1
		parseLoadRequestSeq.Set(parseRequestSeq)
		if parseSummary, parseOk := parseFindConversationSummaryByID(parseCurrentState.ConversationList, parseId4); parseOk && parseOnNavigateToConversation != nil {
			parseOnNavigateToConversation(parseSummary.PublicID)
		}
		parseClient2 := parseChatClientRef.Get()
		if parseClient2 == nil {
			return
		}
		parseScrollMemory.CancelPendingPersist()
		parseScrollMemory.ParsePersistNow(parseCurrentState.ActiveConvID)
		parseScrollMemory.ParsePrepareRestore(parseId4)
		chatLog.Info("load conv", logging.Fields{"conv_id": parseId4})
		go func(parseLoadSeq uint64, parseConversationID int64) {
			parseResp2, parseErr2 := parseClient2.LoadConversation(context.Background(), &chatpb.LoadConversationRequest{Id: parseId4})
			if parseErr2 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr2) {
					return
				}
				chatLog.Error("load conversation failed", logging.Fields{"error": parseErr2})
				return
			}
			if parseLoadRequestSeq.Get() != parseLoadSeq {
				chatLog.Warn("ignored stale conversation load", logging.Fields{
					"conv_id":        parseConversationID,
					"request_seq":    parseLoadSeq,
					"latest_seq":     parseLoadRequestSeq.Get(),
					"current_path":   router.GetCurrentPath(),
					"active_conv_id": parseApp.Get().ActiveConvID,
				})
				return
			}
			parseLoaded := make([]message, 0, len(parseResp2.Messages))
			for _, parseMsg := range parseResp2.Messages {
				parseLoaded = append(parseLoaded, message{
					Role:             parseMsg.Role,
					Content:          parseMsg.Content,
					ModelID:          parseMsg.GetModelId(),
					PromptTokens:     int(parseMsg.GetPromptTokens()),
					CompletionTokens: int(parseMsg.GetCompletionTokens()),
					Tokens:           int(parseMsg.GetCompletionTokens()),
				})
			}
			parseState := parseApp.Get()
			parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseSelectedModelForConversation(parseLoaded, parseState.ModelOptions, parseState.SelectedModel)})
			parseApp.Dispatch(appAction{Type: appActionSetMessages, Messages: parseLoaded})
			parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: parseConversationID, ActiveConvPublicID: parseSummaryPublicIDForID(parseState.ConversationList, parseConversationID)})
			parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
			parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
			chatLog.Info("conv loaded", logging.Fields{"conv_id": parseConversationID, "messages": len(parseLoaded)})
		}(parseRequestSeq, parseId4)
	}

	parseLoad := ui.UseEvent(func(parseE ui.Event) {
		parseId, parseOk2 := parseEventDatasetInt64(parseE, dataConvID)
		if !parseOk2 || parseId <= 0 {
			return
		}
		parseLoadByID(parseId)
	})

	parseResolveRoute := func(parsePublicID string) {
		parseCurrentState2 := parseApp.Get()
		if parseCurrentState2.Streaming {
			return
		}
		parsePublicID = strings.TrimSpace(parsePublicID)
		if parsePublicID == "" {
			return
		}
		parseRequestSeq2 := parseResolveRouteSeq.Get() + 1
		parseResolveRouteSeq.Set(parseRequestSeq2)
		if parseSummary2, parseOk3 := parseFindConversationSummaryByPublicID(parseCurrentState2.ConversationList, parsePublicID); parseOk3 {
			parseLoadByID(parseSummary2.ID)
			return
		}
		parseClient3 := parseChatClientRef.Get()
		if parseClient3 == nil {
			return
		}
		go func(parseResolveSeq uint64, parseRequestedPublicID string) {
			parseResp3, parseErr3 := parseClient3.ResolveConversationRoute(context.Background(), &chatpb.ResolveConversationRouteRequest{PublicId: parsePublicID})
			if parseErr3 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr3) {
					return
				}
				chatLog.Error("resolve conversation route failed", logging.Fields{"error": parseErr3, "public_id": parsePublicID})
				return
			}
			if parseResolveRouteSeq.Get() != parseResolveSeq {
				chatLog.Warn("ignored stale conversation route resolution", logging.Fields{
					"public_id":      parseRequestedPublicID,
					"request_seq":    parseResolveSeq,
					"latest_seq":     parseResolveRouteSeq.Get(),
					"current_path":   router.GetCurrentPath(),
					"active_conv_id": parseApp.Get().ActiveConvID,
				})
				return
			}
			if parseThreadRoutePublicIDFromPath(router.GetCurrentPath()) != parseRequestedPublicID {
				chatLog.Warn("ignored route resolution for inactive path", logging.Fields{
					"public_id":        parseRequestedPublicID,
					"current_path":     router.GetCurrentPath(),
					"current_route_id": parseThreadRoutePublicIDFromPath(router.GetCurrentPath()),
					"active_public_id": parseApp.Get().ActiveConvPublicID,
					"active_conv_id":   parseApp.Get().ActiveConvID,
				})
				return
			}
			if !parseResp3.GetAccessible() || parseResp3.GetId() <= 0 {
				chatLog.Warn("conversation route inaccessible", logging.Fields{"public_id": parsePublicID})
				if parseOnNavigateToConversation != nil {
					parseOnNavigateToConversation("")
				}
				return
			}
			parseLoadByID(parseResp3.GetId())
		}(parseRequestSeq2, parsePublicID)
	}

	parseRequestDelete := ui.UseEvent(func(parseE2 ui.Event) {
		parseE2.StopPropagation()
		parseId2, parseOk4 := parseEventDatasetInt64(parseE2, dataConvID)
		if !parseOk4 || parseId2 <= 0 {
			return
		}
		chatLog.Info("delete conv", logging.Fields{"conv_id": parseId2})
		parseApp.Dispatch(appAction{Type: appActionSetDeleteTarget, DeleteTarget: parseId2})
	})

	parseCancelDelete := ui.UseEvent(func() {
		parseApp.Dispatch(appAction{Type: appActionSetDeleteTarget, DeleteTarget: 0})
	})

	parseConfirmDelete := ui.UseEvent(func() {
		parseId3 := parseApp.Get().DeleteTarget
		if parseId3 <= 0 {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionSetDeleteTarget, DeleteTarget: 0})
		parseClient4 := parseChatClientRef.Get()
		if parseClient4 == nil {
			return
		}
		go func() {
			_, parseErr4 := parseClient4.DeleteConversation(context.Background(), &chatpb.DeleteConversationRequest{Id: parseId3})
			if parseErr4 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr4) {
					return
				}
				chatLog.Error("delete conversation failed", logging.Fields{"error": parseErr4})
				return
			}
			if parseApp.Get().ActiveConvID == parseId3 {
				parseScrollMemory.CancelPendingPersist()
				parseScrollMemory.ParsePersistNow(parseId3)
				parseScrollMemory.ParsePrepareRestore(0)
				parseApp.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
				parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
				parseApp.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
				parseApp.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
				if parseOnNavigateToConversation != nil {
					parseOnNavigateToConversation("")
				}
			}
			chatLog.Info("conv deleted", logging.Fields{"conv_id": parseId3})
			parseRefresh(true)
		}()
	})

	return conversationListController{
		Refresh:       parseRefresh,
		LoadByID:      parseLoadByID,
		ResolveRoute:  parseResolveRoute,
		Load:          parseLoad,
		RequestDelete: parseRequestDelete,
		CancelDelete:  parseCancelDelete,
		ConfirmDelete: parseConfirmDelete,
	}
}
