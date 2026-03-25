//go:build js && wasm

package app

import (
	"context"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/logging"
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
func useConversationList(
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	convListFetchedAt ui.Ref[time.Time],
	scrollMemory threadScrollMemory,
	onResetDraftModel func(),
	onNavigateToConversation func(string),
	handleAuthFailure func(error) bool,
) conversationListController {
	refresh := func(force bool) {
		if !app.Get().Authenticated {
			return
		}
		if !force && time.Since(convListFetchedAt.Get()) < convListTTL {
			return
		}
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		go func() {
			resp, err := client.ListConversations(context.Background(), &chatpb.ListConversationsRequest{})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("list conversations failed", logging.Fields{"error": err})
				return
			}
			summaries := make([]convSummary, 0, len(resp.Conversations))
			for _, conversation := range resp.Conversations {
				summaries = append(summaries, convSummary{
					ID:        conversation.Id,
					PublicID:  conversation.GetPublicId(),
					StartedAt: conversation.StartedAt,
					Preview:   conversation.Preview,
				})
			}
			app.Dispatch(appAction{Type: appActionSetConversationList, ConversationList: summaries})
			convListFetchedAt.Set(time.Now())
			if force {
				chatLog.Info("conv list", logging.Fields{"count": len(summaries)})
			}
		}()
	}

	loadByID := func(id int64) {
		currentState := app.Get()
		if currentState.Streaming || id <= 0 {
			return
		}
		if summary, ok := findConversationSummaryByID(currentState.ConversationList, id); ok && onNavigateToConversation != nil {
			onNavigateToConversation(summary.PublicID)
		}
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		scrollMemory.CancelPendingPersist()
		scrollMemory.PersistNow(currentState.ActiveConvID)
		scrollMemory.PrepareRestore(id)
		chatLog.Info("load conv", logging.Fields{"conv_id": id})
		go func() {
			resp, err := client.LoadConversation(context.Background(), &chatpb.LoadConversationRequest{Id: id})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("load conversation failed", logging.Fields{"error": err})
				return
			}
			loaded := make([]message, 0, len(resp.Messages))
			for _, msg := range resp.Messages {
				loaded = append(loaded, message{
					Role:             msg.Role,
					Content:          msg.Content,
					ModelID:          msg.GetModelId(),
					PromptTokens:     int(msg.GetPromptTokens()),
					CompletionTokens: int(msg.GetCompletionTokens()),
					Tokens:           int(msg.GetCompletionTokens()),
				})
			}
			state := app.Get()
			app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: selectedModelForConversation(loaded, state.ModelOptions, state.DefaultModel)})
			app.Dispatch(appAction{Type: appActionSetMessages, Messages: loaded})
			app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: id, ActiveConvPublicID: summaryPublicIDForID(state.ConversationList, id)})
			app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
			app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
			chatLog.Info("conv loaded", logging.Fields{"conv_id": id, "messages": len(loaded)})
		}()
	}

	load := ui.UseEvent(func(e ui.Event) {
		id, ok := eventDatasetInt64(e, dataConvID)
		if !ok || id <= 0 {
			return
		}
		loadByID(id)
	})

	resolveRoute := func(publicID string) {
		currentState := app.Get()
		if currentState.Streaming {
			return
		}
		publicID = strings.TrimSpace(publicID)
		if publicID == "" {
			return
		}
		if summary, ok := findConversationSummaryByPublicID(currentState.ConversationList, publicID); ok {
			loadByID(summary.ID)
			return
		}
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		go func() {
			resp, err := client.ResolveConversationRoute(context.Background(), &chatpb.ResolveConversationRouteRequest{PublicId: publicID})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("resolve conversation route failed", logging.Fields{"error": err, "public_id": publicID})
				return
			}
			if !resp.GetAccessible() || resp.GetId() <= 0 {
				chatLog.Warn("conversation route inaccessible", logging.Fields{"public_id": publicID})
				if onNavigateToConversation != nil {
					onNavigateToConversation("")
				}
				return
			}
			loadByID(resp.GetId())
		}()
	}

	requestDelete := ui.UseEvent(func(e ui.Event) {
		e.StopPropagation()
		id, ok := eventDatasetInt64(e, dataConvID)
		if !ok || id <= 0 {
			return
		}
		chatLog.Info("delete conv", logging.Fields{"conv_id": id})
		app.Dispatch(appAction{Type: appActionSetDeleteTarget, DeleteTarget: id})
	})

	cancelDelete := ui.UseEvent(func() {
		app.Dispatch(appAction{Type: appActionSetDeleteTarget, DeleteTarget: 0})
	})

	confirmDelete := ui.UseEvent(func() {
		id := app.Get().DeleteTarget
		if id <= 0 {
			return
		}
		app.Dispatch(appAction{Type: appActionSetDeleteTarget, DeleteTarget: 0})
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		go func() {
			_, err := client.DeleteConversation(context.Background(), &chatpb.DeleteConversationRequest{Id: id})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("delete conversation failed", logging.Fields{"error": err})
				return
			}
			if app.Get().ActiveConvID == id {
				scrollMemory.CancelPendingPersist()
				scrollMemory.PersistNow(id)
				scrollMemory.PrepareRestore(0)
				app.Dispatch(appAction{Type: appActionSetMessages, Messages: []message{}})
				app.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: 0, ActiveConvPublicID: ""})
				app.Dispatch(appAction{Type: appActionSetEditIdx, EditIdx: -1})
				app.Dispatch(appAction{Type: appActionSetEditText, EditText: ""})
				if onResetDraftModel != nil {
					onResetDraftModel()
				}
				if onNavigateToConversation != nil {
					onNavigateToConversation("")
				}
			}
			chatLog.Info("conv deleted", logging.Fields{"conv_id": id})
			refresh(true)
		}()
	})

	return conversationListController{
		Refresh:       refresh,
		LoadByID:      loadByID,
		ResolveRoute:  resolveRoute,
		Load:          load,
		RequestDelete: requestDelete,
		CancelDelete:  cancelDelete,
		ConfirmDelete: confirmDelete,
	}
}
