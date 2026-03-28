package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestModelOptionAndSelectedModelRPCs(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "models@example.com")
	parseMustAssignBillingPlan(parseT, store, parseUser.ID, "team")
	parseFake := parseNewFakeProvider()
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-models", parseUser.ID, parseUser.Email)

	parseListResp, parseErr := parseServer.ListModelOptions(parseCtx, &chatpb.ListModelOptionsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListModelOptions: %v", parseErr)
	}
	if parseListResp.GetDefaultModel() != modelGPT54Mini || len(parseListResp.Models) != 1 {
		parseT.Fatalf("unexpected model options response: %+v", parseListResp)
	}

	if _, parseErr2 := parseServer.SetSelectedModel(parseCtx, wrapperspb.String(modelGPT54)); parseErr2 != nil {
		parseT.Fatalf("SetSelectedModel: %v", parseErr2)
	}
	parseSelectedModel, parseErr := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseSelectedModel.GetValue() != modelGPT54 {
		parseT.Fatalf("GetSelectedModel: resp=%+v err=%v", parseSelectedModel, parseErr)
	}

	if _, parseErr3 := parseServer.SetSelectedModel(parseCtx, wrapperspb.String("missing-model")); status.Code(parseErr3) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for unsupported model, got %v", status.Code(parseErr3))
	}
	parseServer.parseUnbindAuthenticatedPeer("peer-models")
}

// TestSetSelectedModelRespectsBillingPlanModelAccess verifies policy denies disabled plan models.
func TestSetSelectedModelRespectsBillingPlanModelAccess(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "models-denied@example.com")
	parseMustAssignBillingPlan(parseT, store, parseUser.ID, "pro")
	if _, parseErr := store.db.Exec(`UPDATE billing_plan_model_access SET is_enabled = 0 WHERE plan_code = 'pro' AND model_id = ?`, modelGPT54); parseErr != nil {
		parseT.Fatalf("disable pro gpt-5.4: %v", parseErr)
	}
	parseServer := parseNewFakeChatServer(store, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-models-denied", parseUser.ID, parseUser.Email)

	if _, parseErr := parseServer.SetSelectedModel(parseCtx, wrapperspb.String(modelGPT54)); status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("expected failed precondition for plan-denied model, got %v", status.Code(parseErr))
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-models-denied")
}

func TestGetSelectedModelRepairsBlankPreferenceUsingFirstCatalogModel(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "model-repair@example.com")
	parseFake := parseNewFakeProvider()
	parseFake.modelOptions = []provider.ModelOption{
		{
			ID:           modelGPT54,
			Label:        "GPT-5.4",
			Note:         "Best",
			Capabilities: parseFake.supportedModels[modelGPT54],
		},
		{
			ID:           modelGPT54Mini,
			Label:        "GPT-5.4 mini",
			Note:         "Fast",
			Capabilities: parseFake.supportedModels[modelGPT54Mini],
		},
	}
	parseFake.defaultModel = modelGPT54Mini
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-model-repair", parseUser.ID, parseUser.Email)

	if parseErr := store.setSelectedModel(parseUser.ID, ""); parseErr != nil {
		parseT.Fatalf("seed blank selected model: %v", parseErr)
	}
	parseSelectedModel, parseErr2 := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{})
	if parseErr2 != nil {
		parseT.Fatalf("GetSelectedModel repair: %v", parseErr2)
	}
	if parseSelectedModel.GetValue() != modelGPT54 {
		parseT.Fatalf("expected first catalog model fallback %q, got %q", modelGPT54, parseSelectedModel.GetValue())
	}
	parsePersistedModel, parseErr2 := store.getSelectedModel(parseUser.ID, "")
	if parseErr2 != nil {
		parseT.Fatalf("store.getSelectedModel after repair: %v", parseErr2)
	}
	if parsePersistedModel != modelGPT54 {
		parseT.Fatalf("expected repaired persisted model %q, got %q", modelGPT54, parsePersistedModel)
	}
	parseServer.parseUnbindAuthenticatedPeer("peer-model-repair")
}

func TestRPCFallbacksWhenStoreOrProvidersAreUnavailable(parseT *testing.T) {
	parseServer := &chatServer{
		defaultModel:     modelGPT54Mini,
		logger:           parseNewTestLogger(),
		providerRegistry: nil,
		store:            nil,
		sessions:         map[string]*sessionState{},
		authUsers:        map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-fallbacks", 99, "fallbacks@example.com")

	parseNameResp, parseErr := parseServer.GetUserName(parseCtx, &chatpb.GetUserNameRequest{})
	if parseErr != nil || parseNameResp.GetName() != "User" {
		parseT.Fatalf("GetUserName fallback: resp=%+v err=%v", parseNameResp, parseErr)
	}
	parseToneResp, parseErr := parseServer.GetSelectedTone(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseToneResp.GetValue() != defaultToneID {
		parseT.Fatalf("GetSelectedTone fallback: resp=%+v err=%v", parseToneResp, parseErr)
	}
	parseModelResp, parseErr := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseModelResp.GetValue() != modelGPT54Mini {
		parseT.Fatalf("GetSelectedModel fallback: resp=%+v err=%v", parseModelResp, parseErr)
	}
	parseSystemPromptResp, parseErr := parseServer.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseSystemPromptResp.GetValue() != defaultCustomSystemPromptTemplate {
		parseT.Fatalf("GetCustomSystemPrompt fallback: resp=%+v err=%v", parseSystemPromptResp, parseErr)
	}
	parseListResp, parseErr := parseServer.ListModelOptions(parseCtx, &chatpb.ListModelOptionsRequest{})
	if parseErr != nil || len(parseListResp.Models) != 0 {
		parseT.Fatalf("ListModelOptions fallback: resp=%+v err=%v", parseListResp, parseErr)
	}
	if _, parseErr2 := parseServer.SetSelectedTone(parseCtx, wrapperspb.String("professional")); parseErr2 != nil {
		parseT.Fatalf("SetSelectedTone no-store should no-op, got %v", parseErr2)
	}
	if _, parseErr3 := parseServer.SetSelectedThinkingEnabled(parseCtx, wrapperspb.Bool(true)); parseErr3 != nil {
		parseT.Fatalf("SetSelectedThinkingEnabled no-store should no-op, got %v", parseErr3)
	}
	if _, parseErr4 := parseServer.SetSelectedThinkingEffort(parseCtx, wrapperspb.String("low")); parseErr4 != nil {
		parseT.Fatalf("SetSelectedThinkingEffort no-store should no-op, got %v", parseErr4)
	}
	if _, parseErr5 := parseServer.SetSelectedModel(parseCtx, wrapperspb.String(modelGPT54Mini)); parseErr5 != nil {
		parseT.Fatalf("SetSelectedModel without registry/store should no-op, got %v", parseErr5)
	}
	if _, parseErr6 := parseServer.SetCustomSystemPrompt(parseCtx, wrapperspb.String("Stay concise.")); parseErr6 != nil {
		parseT.Fatalf("SetCustomSystemPrompt without store should no-op, got %v", parseErr6)
	}
	parseServer.parseUnbindAuthenticatedPeer("peer-fallbacks")
}

func TestSendAndSpeechNegativeBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "negative@example.com")
	parseMustAssignBillingPlan(parseT, store, parseUser.ID, "free")

	parseProviderWithoutThinking := parseNewFakeProvider()
	parseProviderWithoutThinking.supportedModels[modelGPT54Mini] = provider.ModelCapabilities{
		ProviderID:       "fake",
		ProviderLabel:    "Fake",
		SupportsThinking: false,
		SupportsSpeech:   true,
	}
	parseProviderWithoutThinking.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		return provider.ChatResult{}, nil
	}
	parseProviderWithoutThinking.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) { return "", nil }
	parseServer := parseNewFakeChatServer(store, parseProviderWithoutThinking)
	parseCtx := parseBindAuthUser(parseServer, "peer-negative-send", parseUser.ID, parseUser.Email)
	parseErr := parseServer.Send(&chatpb.SendRequest{Message: "Need thinking", Model: modelGPT54Mini, ThinkingEnabled: true}, &fakeChatSendStream{ctx: parseCtx})
	if status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("expected failed precondition for unsupported thinking, got %v", status.Code(parseErr))
	}

	parseErroringProvider := parseNewFakeProvider()
	parseErroringProvider.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		return provider.ChatResult{}, errors.New("boom")
	}
	parseErroringProvider.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) { return "", nil }
	parseErroringServer := parseNewFakeChatServer(store, parseErroringProvider)
	parseErroringCtx := parseBindAuthUser(parseErroringServer, "peer-error-send", parseUser.ID, parseUser.Email)
	parseErroringStream := &fakeChatSendStream{ctx: parseErroringCtx}
	parseErr = parseErroringServer.Send(&chatpb.SendRequest{Message: "Trigger error", Model: modelGPT54Mini}, parseErroringStream)
	if parseErr != nil {
		parseT.Fatalf("expected provider failure to stream an error chunk, got %v", parseErr)
	}
	if len(parseErroringStream.chunks) == 0 {
		parseT.Fatal("expected provider failure to produce at least one stream chunk")
	}
	parseLastChunk := parseErroringStream.chunks[len(parseErroringStream.chunks)-1]
	if !parseLastChunk.GetDone() {
		parseT.Fatalf("expected final error chunk with done=true, got %+v", parseLastChunk)
	}
	if !strings.Contains(parseLastChunk.GetError(), "boom") {
		parseT.Fatalf("expected error chunk to include provider failure details, got %q", parseLastChunk.GetError())
	}
	if parseLastChunk.GetConversationId() <= 0 {
		parseT.Fatalf("expected error chunk to include persisted conversation id, got %+v", parseLastChunk)
	}
	if parseLastChunk.GetUsageEventId() == "" || !parseLastChunk.GetUsagePersisted() {
		parseT.Fatalf("expected error chunk to include persisted usage metadata, got %+v", parseLastChunk)
	}
	if parseLastChunk.GetProviderId() != "fake" || parseLastChunk.GetUsageSource() != provider.UsageSourceMissing {
		parseT.Fatalf("unexpected provider/source on error chunk: %+v", parseLastChunk)
	}
	parseUsageEvents, parseUsageErr := store.parseListUsageEvents(parseUser.ID, 10)
	if parseUsageErr != nil {
		parseT.Fatalf("parseListUsageEvents after provider error: %v", parseUsageErr)
	}
	if len(parseUsageEvents) != 1 {
		parseT.Fatalf("expected one usage event after provider error, got %d", len(parseUsageEvents))
	}
	if parseUsageEvents[0].EventID != parseLastChunk.GetUsageEventId() {
		parseT.Fatalf("usage event id mismatch: row=%q chunk=%q", parseUsageEvents[0].EventID, parseLastChunk.GetUsageEventId())
	}
	if parseUsageEvents[0].Status != "failed" || parseUsageEvents[0].ErrorMessage == "" {
		parseT.Fatalf("expected failed usage event with error details, got %+v", parseUsageEvents[0])
	}
	if parseUsageEvents[0].ProviderID != "fake" || parseUsageEvents[0].ModelID != modelGPT54Mini {
		parseT.Fatalf("unexpected failed usage event provider/model: %+v", parseUsageEvents[0])
	}

	parseSpeechProvider := parseNewFakeProvider()
	parseSpeechProvider.supportedModels[modelGPT54Mini] = provider.ModelCapabilities{
		ProviderID:       "fake",
		ProviderLabel:    "Fake",
		SupportsThinking: true,
		SupportsSpeech:   false,
	}
	parseSpeechProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, _ func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		return provider.SpeechResult{}, nil
	}
	parseSpeechServer := parseNewFakeChatServer(store, parseSpeechProvider)
	parseSpeechCtx := parseBindAuthUser(parseSpeechServer, "peer-negative-speech", parseUser.ID, parseUser.Email)
	parseErr = parseSpeechServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "```code```", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: parseSpeechCtx})
	if status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for unspeakable text, got %v", status.Code(parseErr))
	}
	parseErr = parseSpeechServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "Hello", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: parseSpeechCtx})
	if status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("expected failed precondition for unsupported speech, got %v", status.Code(parseErr))
	}
	parseServer.parseUnbindAuthenticatedPeer("peer-negative-send")
	parseErroringServer.parseUnbindAuthenticatedPeer("peer-error-send")
	parseSpeechServer.parseUnbindAuthenticatedPeer("peer-negative-speech")
}

func TestPreferenceAndConversationRPCErrorBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "rpc-errors@example.com")
	parseServer := parseNewFakeChatServer(store, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-rpc-errors", parseUser.ID, parseUser.Email)

	if _, parseErr := parseServer.SetUserName(parseCtx, &chatpb.SetUserNameRequest{Name: "   "}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for blank name, got %v", status.Code(parseErr))
	}

	store.parseClose()

	if _, parseErr2 := parseServer.DeleteConversation(parseCtx, &chatpb.DeleteConversationRequest{Id: 123}); status.Code(parseErr2) != codes.Internal {
		parseT.Fatalf("expected internal delete error after store close, got %v", status.Code(parseErr2))
	}
	if _, parseErr3 := parseServer.SetUserName(parseCtx, &chatpb.SetUserNameRequest{Name: "Cam"}); status.Code(parseErr3) != codes.Internal {
		parseT.Fatalf("expected internal set user name error after store close, got %v", status.Code(parseErr3))
	}
	if _, parseErr4 := parseServer.GetSelectedThinkingEnabled(parseCtx, &emptypb.Empty{}); status.Code(parseErr4) != codes.Internal {
		parseT.Fatalf("expected internal thinking enabled error after store close, got %v", status.Code(parseErr4))
	}
	if _, parseErr5 := parseServer.GetSelectedThinkingEffort(parseCtx, &emptypb.Empty{}); status.Code(parseErr5) != codes.Internal {
		parseT.Fatalf("expected internal thinking effort error after store close, got %v", status.Code(parseErr5))
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-rpc-errors")
}
