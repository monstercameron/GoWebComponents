package app

import (
	"context"
	"errors"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestModelOptionAndSelectedModelRPCs(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "models@example.com")
	fake := newFakeProvider()
	server := newFakeChatServer(store, fake)
	ctx := bindAuthUser(server, "peer-models", user.ID, user.Email)

	listResp, err := server.ListModelOptions(ctx, &chatpb.ListModelOptionsRequest{})
	if err != nil {
		t.Fatalf("ListModelOptions: %v", err)
	}
	if listResp.GetDefaultModel() != modelGPT54Mini || len(listResp.Models) != 1 {
		t.Fatalf("unexpected model options response: %+v", listResp)
	}

	if _, err := server.SetSelectedModel(ctx, wrapperspb.String(modelGPT54)); err != nil {
		t.Fatalf("SetSelectedModel: %v", err)
	}
	selectedModel, err := server.GetSelectedModel(ctx, &emptypb.Empty{})
	if err != nil || selectedModel.GetValue() != modelGPT54 {
		t.Fatalf("GetSelectedModel: resp=%+v err=%v", selectedModel, err)
	}

	if _, err := server.SetSelectedModel(ctx, wrapperspb.String("missing-model")); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for unsupported model, got %v", status.Code(err))
	}
	server.unbindAuthenticatedPeer("peer-models")
}

func TestRPCFallbacksWhenStoreOrProvidersAreUnavailable(t *testing.T) {
	server := &chatServer{
		defaultModel:     modelGPT54Mini,
		logger:           newTestLogger(),
		providerRegistry: nil,
		store:            nil,
		sessions:         map[string]*sessionState{},
		authUsers:        map[string]authUser{},
	}
	ctx := bindAuthUser(server, "peer-fallbacks", 99, "fallbacks@example.com")

	nameResp, err := server.GetUserName(ctx, &chatpb.GetUserNameRequest{})
	if err != nil || nameResp.GetName() != "User" {
		t.Fatalf("GetUserName fallback: resp=%+v err=%v", nameResp, err)
	}
	toneResp, err := server.GetSelectedTone(ctx, &emptypb.Empty{})
	if err != nil || toneResp.GetValue() != defaultToneID {
		t.Fatalf("GetSelectedTone fallback: resp=%+v err=%v", toneResp, err)
	}
	modelResp, err := server.GetSelectedModel(ctx, &emptypb.Empty{})
	if err != nil || modelResp.GetValue() != modelGPT54Mini {
		t.Fatalf("GetSelectedModel fallback: resp=%+v err=%v", modelResp, err)
	}
	systemPromptResp, err := server.GetCustomSystemPrompt(ctx, &emptypb.Empty{})
	if err != nil || systemPromptResp.GetValue() != "" {
		t.Fatalf("GetCustomSystemPrompt fallback: resp=%+v err=%v", systemPromptResp, err)
	}
	listResp, err := server.ListModelOptions(ctx, &chatpb.ListModelOptionsRequest{})
	if err != nil || len(listResp.Models) != 0 {
		t.Fatalf("ListModelOptions fallback: resp=%+v err=%v", listResp, err)
	}
	if _, err := server.SetSelectedTone(ctx, wrapperspb.String("professional")); err != nil {
		t.Fatalf("SetSelectedTone no-store should no-op, got %v", err)
	}
	if _, err := server.SetSelectedThinkingEnabled(ctx, wrapperspb.Bool(true)); err != nil {
		t.Fatalf("SetSelectedThinkingEnabled no-store should no-op, got %v", err)
	}
	if _, err := server.SetSelectedThinkingEffort(ctx, wrapperspb.String("low")); err != nil {
		t.Fatalf("SetSelectedThinkingEffort no-store should no-op, got %v", err)
	}
	if _, err := server.SetSelectedModel(ctx, wrapperspb.String(modelGPT54Mini)); err != nil {
		t.Fatalf("SetSelectedModel without registry/store should no-op, got %v", err)
	}
	if _, err := server.SetCustomSystemPrompt(ctx, wrapperspb.String("Stay concise.")); err != nil {
		t.Fatalf("SetCustomSystemPrompt without store should no-op, got %v", err)
	}
	server.unbindAuthenticatedPeer("peer-fallbacks")
}

func TestSendAndSpeechNegativeBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "negative@example.com")

	providerWithoutThinking := newFakeProvider()
	providerWithoutThinking.supportedModels[modelGPT54Mini] = provider.ModelCapabilities{
		ProviderID:       "fake",
		ProviderLabel:    "Fake",
		SupportsThinking: false,
		SupportsSpeech:   true,
	}
	providerWithoutThinking.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		return provider.ChatResult{}, nil
	}
	providerWithoutThinking.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) { return "", nil }
	server := newFakeChatServer(store, providerWithoutThinking)
	ctx := bindAuthUser(server, "peer-negative-send", user.ID, user.Email)
	err := server.Send(&chatpb.SendRequest{Message: "Need thinking", Model: modelGPT54Mini, ThinkingEnabled: true}, &fakeChatSendStream{ctx: ctx})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition for unsupported thinking, got %v", status.Code(err))
	}

	erroringProvider := newFakeProvider()
	erroringProvider.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		return provider.ChatResult{}, errors.New("boom")
	}
	erroringProvider.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) { return "", nil }
	erroringServer := newFakeChatServer(store, erroringProvider)
	erroringCtx := bindAuthUser(erroringServer, "peer-error-send", user.ID, user.Email)
	err = erroringServer.Send(&chatpb.SendRequest{Message: "Trigger error", Model: modelGPT54Mini}, &fakeChatSendStream{ctx: erroringCtx})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error from provider failure, got %v", status.Code(err))
	}

	speechProvider := newFakeProvider()
	speechProvider.supportedModels[modelGPT54Mini] = provider.ModelCapabilities{
		ProviderID:       "fake",
		ProviderLabel:    "Fake",
		SupportsThinking: true,
		SupportsSpeech:   false,
	}
	speechProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, _ func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		return provider.SpeechResult{}, nil
	}
	speechServer := newFakeChatServer(store, speechProvider)
	speechCtx := bindAuthUser(speechServer, "peer-negative-speech", user.ID, user.Email)
	err = speechServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "```code```", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: speechCtx})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for unspeakable text, got %v", status.Code(err))
	}
	err = speechServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "Hello", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: speechCtx})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition for unsupported speech, got %v", status.Code(err))
	}
	server.unbindAuthenticatedPeer("peer-negative-send")
	erroringServer.unbindAuthenticatedPeer("peer-error-send")
	speechServer.unbindAuthenticatedPeer("peer-negative-speech")
}

func TestPreferenceAndConversationRPCErrorBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "rpc-errors@example.com")
	server := newFakeChatServer(store, newFakeProvider())
	ctx := bindAuthUser(server, "peer-rpc-errors", user.ID, user.Email)

	if _, err := server.SetUserName(ctx, &chatpb.SetUserNameRequest{Name: "   "}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for blank name, got %v", status.Code(err))
	}

	store.close()

	if _, err := server.DeleteConversation(ctx, &chatpb.DeleteConversationRequest{Id: 123}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal delete error after store close, got %v", status.Code(err))
	}
	if _, err := server.SetUserName(ctx, &chatpb.SetUserNameRequest{Name: "Cam"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal set user name error after store close, got %v", status.Code(err))
	}
	if _, err := server.GetSelectedThinkingEnabled(ctx, &emptypb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal thinking enabled error after store close, got %v", status.Code(err))
	}
	if _, err := server.GetSelectedThinkingEffort(ctx, &emptypb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal thinking effort error after store close, got %v", status.Code(err))
	}

	server.unbindAuthenticatedPeer("peer-rpc-errors")
}
