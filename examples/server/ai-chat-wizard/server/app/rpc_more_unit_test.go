package app

import (
	"context"
	"errors"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type failingSpeechStream struct {
	ctx context.Context
}

func (parseS *failingSpeechStream) Send(*chatpb.SynthesizeSpeechChunk) error {
	return errors.New("stream send failed")
}
func (parseS *failingSpeechStream) SetHeader(metadata.MD) error  { return nil }
func (parseS *failingSpeechStream) SendHeader(metadata.MD) error { return nil }
func (parseS *failingSpeechStream) SetTrailer(metadata.MD)       {}
func (parseS *failingSpeechStream) Context() context.Context     { return parseS.ctx }
func (parseS *failingSpeechStream) SendMsg(any) error            { return nil }
func (parseS *failingSpeechStream) RecvMsg(any) error            { return nil }

func TestGenerateAndSaveConversationTitleBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "title-branches@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}

	parseNilSafeServer := &chatServer{logger: parseNewTestLogger()}
	parseNilSafeServer.parseGenerateAndSaveConversationTitle(parseUser.ID, parseConversationID, modelGPT54Mini, "user", "assistant")

	parseMissingProviderServer := &chatServer{store: store, logger: parseNewTestLogger()}
	parseMissingProviderServer.parseGenerateAndSaveConversationTitle(parseUser.ID, parseConversationID, modelGPT54Mini, "user", "assistant")

	parseFake := parseNewFakeProvider()
	parseFake.generateTitle = func(_ context.Context, parseReq provider.TitleRequest) (string, error) {
		if len(parseReq.Prompt) == 0 {
			parseT.Fatal("expected generate title prompt to be populated")
		}
		if len(parseReq.Prompt) > 900 {
			parseT.Fatalf("expected title prompt truncation to keep prompt compact, got len=%d", len(parseReq.Prompt))
		}
		return "", nil
	}
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseServer.parseGenerateAndSaveConversationTitle(parseUser.ID, parseConversationID, modelGPT54Mini, string(make([]byte, 600)), string(make([]byte, 600)))

	parseConversations, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 1 && len(parseConversations) != 0 {
		parseT.Fatalf("unexpected conversations after empty title branch: %+v", parseConversations)
	}

	parseErroringProvider := parseNewFakeProvider()
	parseErroringProvider.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) {
		return "", errors.New("title failure")
	}
	parseNewFakeChatServer(store, parseErroringProvider).parseGenerateAndSaveConversationTitle(parseUser.ID, parseConversationID, modelGPT54Mini, "user", "assistant")

	store.parseClose()
	parseNewFakeChatServer(store, parseFake).parseGenerateAndSaveConversationTitle(parseUser.ID, parseConversationID, modelGPT54Mini, "user", "assistant")
}

func TestListAndLoadConversationBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "list-load@example.com")
	parseOther := parseMustCreateUser(parseT, store, "list-load-other@example.com")
	parseMustEnsureWorkspaceMembership(parseT, store, parseUser.ID, "ws-list-load-owner")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	parseLongTitle := "This is a deliberately very long conversation title that should be truncated in summaries"
	if parseErr2 := store.parseSaveConversationTitle(parseUser.ID, parseConversationID, parseLongTitle); parseErr2 != nil {
		parseT.Fatalf("saveConversationTitle: %v", parseErr2)
	}
	if parseErr3 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "ASSISTANT", "hello", modelGPT54Mini, 5, 7); parseErr3 != nil {
		parseT.Fatalf("saveConversationMessage: %v", parseErr3)
	}

	parseServer := &chatServer{defaultModel: modelGPT54Mini, store: store, logger: parseNewTestLogger(), sessions: map[string]*sessionState{}, authUsers: map[string]authUser{}}
	parseCtx := parseBindAuthUser(parseServer, "peer-list-load", parseUser.ID, parseUser.Email)

	parseListResp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations: %v", parseErr)
	}
	if len(parseListResp.GetConversations()) != 1 {
		parseT.Fatalf("expected one conversation summary, got %+v", parseListResp)
	}
	if parsePreview := parseListResp.GetConversations()[0].GetPreview(); len(parsePreview) != 63 || parsePreview[len(parsePreview)-3:] != "..." {
		parseT.Fatalf("expected truncated preview with ellipsis, got %q", parsePreview)
	}

	parsePublicID := parseListResp.GetConversations()[0].GetPublicId()
	if parsePublicID == "" {
		parseT.Fatal("expected conversation summary public_id")
	}

	parseRouteResp, parseErr := parseServer.ResolveConversationRoute(parseCtx, &chatpb.ResolveConversationRouteRequest{PublicId: parsePublicID})
	if parseErr != nil {
		parseT.Fatalf("ResolveConversationRoute owner: %v", parseErr)
	}
	if !parseRouteResp.GetAccessible() || parseRouteResp.GetId() != parseConversationID {
		parseT.Fatalf("unexpected owner route response: %+v", parseRouteResp)
	}

	parseOtherCtx := parseBindAuthUser(parseServer, "peer-list-load-other", parseOther.ID, parseOther.Email)
	parseOtherRouteResp, parseErr := parseServer.ResolveConversationRoute(parseOtherCtx, &chatpb.ResolveConversationRouteRequest{PublicId: parsePublicID})
	if parseErr != nil {
		parseT.Fatalf("ResolveConversationRoute other: %v", parseErr)
	}
	if parseOtherRouteResp.GetAccessible() || parseOtherRouteResp.GetId() != 0 {
		parseT.Fatalf("expected inaccessible route for non-owner, got %+v", parseOtherRouteResp)
	}

	parseLoadResp, parseErr := parseServer.LoadConversation(parseCtx, &chatpb.LoadConversationRequest{Id: parseConversationID})
	if parseErr != nil {
		parseT.Fatalf("LoadConversation: %v", parseErr)
	}
	if len(parseLoadResp.GetMessages()) != 1 || parseLoadResp.GetMessages()[0].GetModelId() != modelGPT54Mini {
		parseT.Fatalf("unexpected load conversation response: %+v", parseLoadResp)
	}
	if parseLoadResp.GetMessages()[0].GetRole() != "assistant" {
		parseT.Fatalf("expected normalized assistant role, got %q", parseLoadResp.GetMessages()[0].GetRole())
	}
	parseAnalyticsRows, parseErr := store.parseListProductAnalyticsEvents(100)
	if parseErr != nil {
		parseT.Fatalf("parseListProductAnalyticsEvents: %v", parseErr)
	}
	isParseHasThreadReopenedEvent := false
	for _, parseAnalyticsRow := range parseAnalyticsRows {
		if parseAnalyticsRow.UserID != parseUser.ID || parseAnalyticsRow.FunnelKey != parseFirstChatFunnelKey {
			continue
		}
		if parseAnalyticsRow.StepKey == parseFirstChatStepThreadReopened {
			isParseHasThreadReopenedEvent = true
			break
		}
	}
	if !isParseHasThreadReopenedEvent {
		parseT.Fatalf("expected thread reopened analytics event, got rows=%+v", parseAnalyticsRows)
	}

	store.parseClose()
	if _, parseErr4 := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{}); status.Code(parseErr4) != codes.Internal {
		parseT.Fatalf("expected internal ListConversations error after store close, got %v", status.Code(parseErr4))
	}
	if _, parseErr5 := parseServer.ResolveConversationRoute(parseCtx, &chatpb.ResolveConversationRouteRequest{PublicId: parsePublicID}); status.Code(parseErr5) != codes.Internal {
		parseT.Fatalf("expected internal ResolveConversationRoute error after store close, got %v", status.Code(parseErr5))
	}
	if _, parseErr6 := parseServer.LoadConversation(parseCtx, &chatpb.LoadConversationRequest{Id: parseConversationID}); status.Code(parseErr6) != codes.Internal {
		parseT.Fatalf("expected internal LoadConversation error after store close, got %v", status.Code(parseErr6))
	}
}

func TestListConversationsPagination(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "list-pagination@example.com")
	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        parseStore,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-list-pagination", parseUser.ID, parseUser.Email)

	for parseIndex := range 5 {
		if _, parseErr := parseStore.parseCreateConversation(parseUser.ID); parseErr != nil {
			parseT.Fatalf("parseCreateConversation(%d): %v", parseIndex, parseErr)
		}
	}

	parseAllResp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations all: %v", parseErr)
	}
	if len(parseAllResp.GetConversations()) != 5 {
		parseT.Fatalf("expected full list length=5, got %d", len(parseAllResp.GetConversations()))
	}
	if parseAllResp.GetHasMore() {
		parseT.Fatalf("expected has_more=false for full-list mode, got true")
	}

	parsePage1Resp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{PageSize: 2, PageOffset: 0})
	if parseErr != nil {
		parseT.Fatalf("ListConversations page1: %v", parseErr)
	}
	if len(parsePage1Resp.GetConversations()) != 2 || !parsePage1Resp.GetHasMore() || parsePage1Resp.GetNextOffset() != 2 {
		parseT.Fatalf("unexpected page1 response: %+v", parsePage1Resp)
	}

	parsePage2Resp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{PageSize: 2, PageOffset: 2})
	if parseErr != nil {
		parseT.Fatalf("ListConversations page2: %v", parseErr)
	}
	if len(parsePage2Resp.GetConversations()) != 2 || !parsePage2Resp.GetHasMore() || parsePage2Resp.GetNextOffset() != 4 {
		parseT.Fatalf("unexpected page2 response: %+v", parsePage2Resp)
	}

	parsePage3Resp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{PageSize: 2, PageOffset: 4})
	if parseErr != nil {
		parseT.Fatalf("ListConversations page3: %v", parseErr)
	}
	if len(parsePage3Resp.GetConversations()) != 1 || parsePage3Resp.GetHasMore() || parsePage3Resp.GetNextOffset() != 5 {
		parseT.Fatalf("unexpected page3 response: %+v", parsePage3Resp)
	}
}

func TestListConversationsSyncsStarterMilestones(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "starter-milestones@example.com")
	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        parseStore,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-starter-milestones", parseUser.ID, parseUser.Email)

	parseListResp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations initial: %v", parseErr)
	}
	if len(parseListResp.GetConversations()) != 0 {
		parseT.Fatalf("expected empty conversations for brand-new user, got %+v", parseListResp.GetConversations())
	}
	parseMilestoneRows, parseErr := parseStore.parseListUserActivationMilestones(200)
	if parseErr != nil {
		parseT.Fatalf("parseListUserActivationMilestones initial: %v", parseErr)
	}
	isParseHasFirstRunMilestone := false
	for _, parseMilestoneRow := range parseMilestoneRows {
		if parseMilestoneRow.UserID != parseUser.ID {
			continue
		}
		if parseMilestoneRow.MilestoneKey == parseStarterMilestoneFirstRunDetected && parseMilestoneRow.Status == "completed" {
			isParseHasFirstRunMilestone = true
			break
		}
	}
	if !isParseHasFirstRunMilestone {
		parseT.Fatalf("expected %q milestone for new user, got rows=%+v", parseStarterMilestoneFirstRunDetected, parseMilestoneRows)
	}

	if _, parseErr2 := parseStore.parseCreateConversation(parseUser.ID); parseErr2 != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr2)
	}
	parseListResp2, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations returning: %v", parseErr)
	}
	if len(parseListResp2.GetConversations()) != 1 {
		parseT.Fatalf("expected one conversation for returning user, got %+v", parseListResp2.GetConversations())
	}
	parseMilestoneRows2, parseErr := parseStore.parseListUserActivationMilestones(200)
	if parseErr != nil {
		parseT.Fatalf("parseListUserActivationMilestones returning: %v", parseErr)
	}
	isParseHasReturningMilestone := false
	for _, parseMilestoneRow := range parseMilestoneRows2 {
		if parseMilestoneRow.UserID != parseUser.ID {
			continue
		}
		if parseMilestoneRow.MilestoneKey == parseStarterMilestoneReturningUserDetected && parseMilestoneRow.Status == "completed" {
			isParseHasReturningMilestone = true
			break
		}
	}
	if !isParseHasReturningMilestone {
		parseT.Fatalf("expected %q milestone for returning user, got rows=%+v", parseStarterMilestoneReturningUserDetected, parseMilestoneRows2)
	}
}

func TestSynthesizeSpeechAdditionalBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "speech-branches@example.com")

	parseNoProviderServer := &chatServer{logger: parseNewTestLogger(), store: store, sessions: map[string]*sessionState{}, authUsers: map[string]authUser{}}
	parseNoProviderCtx := parseBindAuthUser(parseNoProviderServer, "peer-no-provider-speech", parseUser.ID, parseUser.Email)
	if parseErr := parseNoProviderServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello"}, &fakeSpeechStream{ctx: parseNoProviderCtx}); status.Code(parseErr) != codes.Unavailable {
		parseT.Fatalf("expected unavailable when provider registry is nil, got %v", status.Code(parseErr))
	}

	parseUnsupportedModelServer := parseNewFakeChatServer(store, parseNewFakeProvider())
	parseUnsupportedCtx := parseBindAuthUser(parseUnsupportedModelServer, "peer-unsupported-model-speech", parseUser.ID, parseUser.Email)
	if parseErr2 := parseUnsupportedModelServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: "missing-model"}, &fakeSpeechStream{ctx: parseUnsupportedCtx}); status.Code(parseErr2) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for unsupported speech model, got %v", status.Code(parseErr2))
	}

	parseEmptyAudioProvider := parseNewFakeProvider()
	parseEmptyAudioProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, parseEmit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		return provider.SpeechResult{MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: "hello"}, parseEmit(provider.SpeechChunk{Done: true})
	}
	parseEmptyAudioServer := parseNewFakeChatServer(store, parseEmptyAudioProvider)
	parseEmptyAudioCtx := parseBindAuthUser(parseEmptyAudioServer, "peer-empty-audio-speech", parseUser.ID, parseUser.Email)
	if parseErr3 := parseEmptyAudioServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: parseEmptyAudioCtx}); status.Code(parseErr3) != codes.Internal {
		parseT.Fatalf("expected internal error for empty synthesized audio, got %v", status.Code(parseErr3))
	}

	parseStatusErrProvider := parseNewFakeProvider()
	parseStatusErrProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, _ func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		return provider.SpeechResult{}, status.Error(codes.Unavailable, "provider busy")
	}
	parseStatusErrServer := parseNewFakeChatServer(store, parseStatusErrProvider)
	parseStatusErrCtx := parseBindAuthUser(parseStatusErrServer, "peer-status-err-speech", parseUser.ID, parseUser.Email)
	if parseErr4 := parseStatusErrServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: parseStatusErrCtx}); status.Code(parseErr4) != codes.Unavailable {
		parseT.Fatalf("expected provider gRPC status to pass through, got %v", status.Code(parseErr4))
	}

	parseStreamFailProvider := parseNewFakeProvider()
	parseStreamFailProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, parseEmit2 func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		if parseErr5 := parseEmit2(provider.SpeechChunk{AudioChunk: []byte("abc")}); parseErr5 != nil {
			return provider.SpeechResult{}, parseErr5
		}
		return provider.SpeechResult{MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: "hello"}, nil
	}
	parseStreamFailServer := parseNewFakeChatServer(store, parseStreamFailProvider)
	parseStreamFailCtx := parseBindAuthUser(parseStreamFailServer, "peer-stream-fail-speech", parseUser.ID, parseUser.Email)
	if parseErr6 := parseStreamFailServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: modelGPT54Mini}, &failingSpeechStream{ctx: parseStreamFailCtx}); status.Code(parseErr6) != codes.Canceled {
		parseT.Fatalf("expected canceled status for downstream stream send failure, got %v", status.Code(parseErr6))
	}
}
