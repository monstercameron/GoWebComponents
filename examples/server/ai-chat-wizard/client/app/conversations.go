//go:build js && wasm

package app

import (
	"context"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/logging"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// conversationListController groups the sidebar conversation RPC workflow.
//
// The app shell only needs the resulting handlers and the explicit refresh
// entrypoint; the gRPC list/load/delete details stay local to this hook.
type conversationListController struct {
	Refresh           func(bool)
	LoadByID          func(int64)
	ResolveRoute      func(string)
	Load              ui.Handler
	LoadMore          ui.Handler
	RequestDelete     ui.Handler
	CancelDelete      ui.Handler
	ConfirmDelete     ui.Handler
	Organization      sidebarOrganizationState
	RenameTargetID    int64
	RenameDraft       string
	CreateFolder      ui.Handler
	ChangeFolderName  ui.Handler
	StartRename       ui.Handler
	ChangeRenameDraft ui.Handler
	SaveRename        ui.Handler
	HandleRenameKey   ui.Handler
	CancelRename      ui.Handler
	DragStart         ui.Handler
	DragOver          ui.Handler
	DropThread        ui.Handler
	DropFolder        ui.Handler
	DropUnfiled       ui.Handler
	DragEnd           ui.Handler
}

const parseSidebarConversationPageSize int32 = 40

// parseMergeConversationPage appends one paged response while avoiding duplicate rows.
func parseMergeConversationPage(parseExisting, parseIncoming []convSummary, isAppend bool) []convSummary {
	if !isAppend {
		return parseIncoming
	}
	if len(parseExisting) == 0 {
		return parseIncoming
	}
	parseMerged := make([]convSummary, 0, len(parseExisting)+len(parseIncoming))
	parseMerged = append(parseMerged, parseExisting...)
	parseSeenConversationID := make(map[int64]struct{}, len(parseExisting))
	for _, parseSummary := range parseExisting {
		parseSeenConversationID[parseSummary.ID] = struct{}{}
	}
	for _, parseSummary2 := range parseIncoming {
		if _, hasSeenConversationID := parseSeenConversationID[parseSummary2.ID]; hasSeenConversationID {
			continue
		}
		parseMerged = append(parseMerged, parseSummary2)
	}
	return parseMerged
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
	parseListPageRequestSeq := ui.UseRef(uint64(0))
	parseNextConversationOffset := ui.UseRef(int32(0))
	hasConversationPageMore := ui.UseRef(false)
	isConversationPageLoading := ui.UseRef(false)
	parseConversationBootstrapLogged := ui.UseRef(false)
	parseSidebarOrganization := ui.UsePersistedState[sidebarOrganizationState](storageKeySidebarOrganization, sidebarOrganizationState{ThreadFolders: map[int64]string{}}, ui.PersistLocal)
	parseRenameTargetID := ui.UseState(int64(0))
	parseRenameDraft := ui.UseState("")
	parseSidebarDragKind := ui.UseRef("")
	parseSidebarDragConvID := ui.UseRef(int64(0))
	parseSidebarDragFolderID := ui.UseRef("")

	parseFetchConversationPage := func(parseOffset int32, isAppend bool) {
		if !parseApp.Get().Authenticated {
			return
		}
		if isConversationPageLoading.Get() {
			return
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return
		}
		isConversationPageLoading.Set(true)
		parseRequestSeq := parseListPageRequestSeq.Get() + 1
		parseListPageRequestSeq.Set(parseRequestSeq)
		parseSidebarScrollTop, hasSidebarScrollTop := parseSidebarListScrollTop()
		isSidebarPinnedBottom := parseSidebarListIsAtScrollBottom()
		go func(parsePageSeq uint64, parsePageOffset int32, isAppendPage bool, parsePrevSidebarScrollTop float64, hasPrevSidebarScrollTop, isPrevSidebarPinnedBottom bool) {
			parseResp, parseErr := parseClient.ListConversations(context.Background(), &chatpb.ListConversationsRequest{
				PageSize:   parseSidebarConversationPageSize,
				PageOffset: parsePageOffset,
			})
			if parseErr != nil {
				isConversationPageLoading.Set(false)
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				chatLog.Error("conversation page load failed", logging.Fields{
					"error":         parseErr,
					"grpc_ready":    parseApp.Get().GRPCReady,
					"authenticated": parseApp.Get().Authenticated,
					"offset":        parsePageOffset,
					"is_append":     isAppendPage,
				})
				return
			}
			if parseListPageRequestSeq.Get() != parsePageSeq {
				isConversationPageLoading.Set(false)
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
			parseExistingSummaries := []convSummary(nil)
			if isAppendPage {
				parseExistingSummaries = parseApp.Get().ConversationList
			}
			parseMergedSummaries := parseMergeConversationPage(parseExistingSummaries, parseSummaries, isAppendPage)
			parseSidebarOrganization.Set(parseSanitizeSidebarOrganization(parseSidebarOrganization.Get(), parseMergedSummaries))
			parseApp.Dispatch(appAction{Type: appActionSetConversationList, ConversationList: parseMergedSummaries})
			parseNextConversationOffset.Set(parseResp.GetNextOffset())
			hasConversationPageMore.Set(parseResp.GetHasMore())
			parseConvListFetchedAt.Set(time.Now())
			parseRestoreSidebarListScroll(parsePrevSidebarScrollTop, hasPrevSidebarScrollTop, isPrevSidebarPinnedBottom)
			if !parseConversationBootstrapLogged.Get() {
				parseConversationBootstrapLogged.Set(true)
				chatLog.Info("conversation bootstrap loaded", logging.Fields{
					"count":    len(parseMergedSummaries),
					"has_more": parseResp.GetHasMore(),
				})
				isConversationPageLoading.Set(false)
				return
			}
			if isAppendPage {
				chatLog.Info("conversation page loaded", logging.Fields{
					"count":       len(parseSummaries),
					"total_count": len(parseMergedSummaries),
					"next_offset": parseResp.GetNextOffset(),
					"has_more":    parseResp.GetHasMore(),
				})
			}
			isConversationPageLoading.Set(false)
		}(parseRequestSeq, parseOffset, isAppend, parseSidebarScrollTop, hasSidebarScrollTop, isSidebarPinnedBottom)
	}

	parseRefresh := func(isForce bool) {
		if !parseApp.Get().Authenticated {
			return
		}
		if !isForce && time.Since(parseConvListFetchedAt.Get()) < convListTTL {
			return
		}
		parseNextConversationOffset.Set(0)
		hasConversationPageMore.Set(false)
		parseFetchConversationPage(0, false)
		if isForce {
			chatLog.Info("conv list refresh requested", logging.Fields{"mode": "page-reset"})
		}
	}

	parseTryLoadMore := func() {
		if !hasConversationPageMore.Get() || isConversationPageLoading.Get() {
			return
		}
		if !parseSidebarListIsAtScrollBottom() {
			return
		}
		parseFetchConversationPage(parseNextConversationOffset.Get(), true)
	}

	parseLoadMore := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseTryLoadMore()
	})

	isConversationListAuthenticated := parseApp.Get().Authenticated
	parseConversationListLength := len(parseApp.Get().ConversationList)
	ui.UseEffect(func() func() {
		if !isConversationListAuthenticated {
			return nil
		}
		parseDoc, parseErr := interop.GetDocument()
		if parseErr != nil {
			return nil
		}
		parseSidebarListElement, hasSidebarListElement, parseErr := parseDoc.ElementByID(idConvList)
		if parseErr != nil || !hasSidebarListElement {
			return nil
		}
		parseScrollSub, parseErr := parseSidebarListElement.Listen("scroll", func(_ interop.BrowserEvent) {
			parseTryLoadMore()
		})
		if parseErr != nil {
			return nil
		}
		return func() {
			parseScrollSub.Cancel()
		}
	}, isConversationListAuthenticated, parseConversationListLength)

	parseLoadConversationByID := func(parseId4 int64, parseKnownPublicID ...string) {
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
			parseActiveConversationPublicID := parseSummaryPublicIDForID(parseState.ConversationList, parseConversationID)
			if parseActiveConversationPublicID == "" && len(parseKnownPublicID) > 0 {
				parseActiveConversationPublicID = strings.TrimSpace(parseKnownPublicID[0])
			}
			parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseSelectedModelForConversation(parseLoaded, parseState.ModelOptions, parseState.SelectedModel)})
			parseApp.Dispatch(appAction{Type: appActionSetMessages, Messages: parseLoaded})
			parseApp.Dispatch(appAction{Type: appActionSetActiveConvID, ActiveConvID: parseConversationID, ActiveConvPublicID: parseActiveConversationPublicID})
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
		parseLoadConversationByID(parseId)
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
			parseLoadConversationByID(parseSummary2.ID, parseSummary2.PublicID)
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
			parseLoadConversationByID(parseResp3.GetId(), parseResp3.GetPublicId())
		}(parseRequestSeq2, parsePublicID)
	}

	parseLoadByID := func(parseConversationID int64) {
		parseLoadConversationByID(parseConversationID)
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

	parseCreateFolder := ui.UseEvent(func() {
		parseCurrentOrg := parseSidebarOrganization.Get()
		parseSidebarOrganization.Set(parseSidebarCreateFolder(parseCurrentOrg, ""))
	})

	parseChangeFolderName := ui.UseEvent(func(parseE ui.Event) {
		parseFolderID := parseEventDatasetValue(parseE, dataSidebarFolderID)
		if parseFolderID == "" {
			return
		}
		parseSidebarOrganization.Set(parseSidebarRenameFolder(parseSidebarOrganization.Get(), parseFolderID, parseE.GetValue()))
	})

	parseStartRename := ui.UseEvent(func(parseE ui.Event) {
		parseE.StopPropagation()
		parseID, parseOk := parseEventDatasetInt64(parseE, dataConvID)
		if !parseOk || parseID <= 0 {
			return
		}
		parsePreview := ""
		if parseSummary, parseFound := parseFindConversationSummaryByID(parseApp.Get().ConversationList, parseID); parseFound {
			parsePreview = parseSummary.Preview
		}
		parseRenameTargetID.Set(parseID)
		parseRenameDraft.Set(strings.TrimSpace(parsePreview))
	})

	parseChangeRenameDraft := ui.UseEvent(func(parseE ui.Event) {
		parseRenameDraft.Set(parseE.GetValue())
	})

	parseCancelRenameNow := func() {
		parseRenameTargetID.Set(0)
		parseRenameDraft.Set("")
	}

	parseCancelRename := ui.UseEvent(func(parseE ui.Event) {
		parseE.StopPropagation()
		parseCancelRenameNow()
	})

	parseSaveRenameNow := func() {
		parseID := parseRenameTargetID.Get()
		parseTitle := strings.TrimSpace(parseRenameDraft.Get())
		if parseID <= 0 {
			return
		}
		if parseTitle == "" {
			parseCancelRenameNow()
			return
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return
		}
		parsePreviousList := append([]convSummary(nil), parseApp.Get().ConversationList...)
		parseOptimisticList := parseReplaceConversationPreview(parsePreviousList, parseID, parseTitle)
		parseApp.Dispatch(appAction{Type: appActionSetConversationList, ConversationList: parseOptimisticList})
		parseRenameTargetID.Set(0)
		parseRenameDraft.Set("")
		go func(parseConversationID int64, parseConversationTitle string, parseRollback []convSummary) {
			parseResp, parseErr := parseClient.RenameConversation(context.Background(), &chatpb.RenameConversationRequest{
				Id:    parseConversationID,
				Title: parseConversationTitle,
			})
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				chatLog.Error("rename conversation failed", logging.Fields{"error": parseErr, "conv_id": parseConversationID})
				parseApp.Dispatch(appAction{Type: appActionSetConversationList, ConversationList: parseRollback})
				return
			}
			if parseResp.GetConversation() == nil {
				return
			}
			parseApp.Dispatch(appAction{Type: appActionSetConversationList, ConversationList: parseReplaceConversationPreview(parseApp.Get().ConversationList, parseConversationID, parseResp.GetConversation().GetPreview())})
		}(parseID, parseTitle, parsePreviousList)
	}

	parseSaveRename := ui.UseEvent(func(parseE ui.Event) {
		parseE.StopPropagation()
		parseSaveRenameNow()
	})

	parseHandleRenameKey := ui.UseEvent(func(parseE ui.Event) {
		switch parseE.GetKey() {
		case "Enter":
			parseE.PreventDefault()
			parseSaveRenameNow()
		case "Escape":
			parseE.PreventDefault()
			parseCancelRenameNow()
		}
	})

	parseDragStart := ui.UseEvent(func(parseE ui.Event) {
		parseSidebarDragKind.Set(parseEventDatasetValue(parseE, dataSidebarDragKind))
		if parseID, parseOk := parseEventDatasetInt64(parseE, dataConvID); parseOk {
			parseSidebarDragConvID.Set(parseID)
		} else {
			parseSidebarDragConvID.Set(0)
		}
		parseSidebarDragFolderID.Set(parseEventDatasetValue(parseE, dataSidebarFolderID))
	})

	parseDragOver := ui.UseEvent(func(parseE ui.Event) {
		parseE.PreventDefault()
	})

	parseDropThread := ui.UseEvent(func(parseE ui.Event) {
		parseE.PreventDefault()
		parseKind := parseSidebarDragKind.Get()
		parseTargetThreadID, parseOk := parseEventDatasetInt64(parseE, dataConvID)
		if !parseOk || parseTargetThreadID <= 0 {
			return
		}
		parseTargetFolderID := parseEventDatasetValue(parseE, dataSidebarFolderID)
		if parseKind == sidebarDragKindThread {
			parseThreadID := parseSidebarDragConvID.Get()
			parseSidebarOrganization.Set(parseSidebarReorderThread(parseSidebarOrganization.Get(), parseApp.Get().ConversationList, parseThreadID, parseTargetThreadID, parseTargetFolderID))
		}
	})

	parseDropFolder := ui.UseEvent(func(parseE ui.Event) {
		parseE.PreventDefault()
		parseTargetFolderID := parseEventDatasetValue(parseE, dataSidebarFolderID)
		if parseTargetFolderID == "" {
			return
		}
		switch parseSidebarDragKind.Get() {
		case sidebarDragKindThread:
			parseThreadID := parseSidebarDragConvID.Get()
			parseSidebarOrganization.Set(parseSidebarMoveThreadToFolder(parseSidebarOrganization.Get(), parseApp.Get().ConversationList, parseThreadID, parseTargetFolderID))
		case sidebarDragKindFolder:
			parseSidebarOrganization.Set(parseSidebarReorderFolder(parseSidebarOrganization.Get(), parseSidebarDragFolderID.Get(), parseTargetFolderID))
		}
	})

	parseDropUnfiled := ui.UseEvent(func(parseE ui.Event) {
		parseE.PreventDefault()
		if parseSidebarDragKind.Get() != sidebarDragKindThread {
			return
		}
		parseThreadID := parseSidebarDragConvID.Get()
		parseSidebarOrganization.Set(parseSidebarMoveThreadToFolder(parseSidebarOrganization.Get(), parseApp.Get().ConversationList, parseThreadID, ""))
	})

	parseDragEnd := ui.UseEvent(func() {
		parseSidebarDragKind.Set("")
		parseSidebarDragConvID.Set(0)
		parseSidebarDragFolderID.Set("")
	})

	return conversationListController{
		Refresh:           parseRefresh,
		LoadByID:          parseLoadByID,
		ResolveRoute:      parseResolveRoute,
		Load:              parseLoad,
		LoadMore:          parseLoadMore,
		RequestDelete:     parseRequestDelete,
		CancelDelete:      parseCancelDelete,
		ConfirmDelete:     parseConfirmDelete,
		Organization:      parseSanitizeSidebarOrganization(parseSidebarOrganization.Get(), parseApp.Get().ConversationList),
		RenameTargetID:    parseRenameTargetID.Get(),
		RenameDraft:       parseRenameDraft.Get(),
		CreateFolder:      parseCreateFolder,
		ChangeFolderName:  parseChangeFolderName,
		StartRename:       parseStartRename,
		ChangeRenameDraft: parseChangeRenameDraft,
		SaveRename:        parseSaveRename,
		HandleRenameKey:   parseHandleRenameKey,
		CancelRename:      parseCancelRename,
		DragStart:         parseDragStart,
		DragOver:          parseDragOver,
		DropThread:        parseDropThread,
		DropFolder:        parseDropFolder,
		DropUnfiled:       parseDropUnfiled,
		DragEnd:           parseDragEnd,
	}
}
