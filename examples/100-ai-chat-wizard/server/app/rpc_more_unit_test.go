package app

import (
	"context"
	"errors"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type failingSpeechStream struct {
	ctx context.Context
}

func (s *failingSpeechStream) Send(*chatpb.SynthesizeSpeechChunk) error {
	return errors.New("stream send failed")
}
func (s *failingSpeechStream) SetHeader(metadata.MD) error  { return nil }
func (s *failingSpeechStream) SendHeader(metadata.MD) error { return nil }
func (s *failingSpeechStream) SetTrailer(metadata.MD)       {}
func (s *failingSpeechStream) Context() context.Context     { return s.ctx }
func (s *failingSpeechStream) SendMsg(any) error            { return nil }
func (s *failingSpeechStream) RecvMsg(any) error            { return nil }

func TestGenerateAndSaveConversationTitleBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "title-branches@example.com")
	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}

	nilSafeServer := &chatServer{logger: newTestLogger()}
	nilSafeServer.generateAndSaveConversationTitle(user.ID, conversationID, modelGPT54Mini, "user", "assistant")

	missingProviderServer := &chatServer{store: store, logger: newTestLogger()}
	missingProviderServer.generateAndSaveConversationTitle(user.ID, conversationID, modelGPT54Mini, "user", "assistant")

	fake := newFakeProvider()
	fake.generateTitle = func(_ context.Context, req provider.TitleRequest) (string, error) {
		if len(req.Prompt) == 0 {
			t.Fatal("expected generate title prompt to be populated")
		}
		if len(req.Prompt) > 900 {
			t.Fatalf("expected title prompt truncation to keep prompt compact, got len=%d", len(req.Prompt))
		}
		return "", nil
	}
	server := newFakeChatServer(store, fake)
	server.generateAndSaveConversationTitle(user.ID, conversationID, modelGPT54Mini, string(make([]byte, 600)), string(make([]byte, 600)))

	conversations, err := store.listConversations(user.ID)
	if err != nil {
		t.Fatalf("listConversations: %v", err)
	}
	if len(conversations) != 1 && len(conversations) != 0 {
		t.Fatalf("unexpected conversations after empty title branch: %+v", conversations)
	}

	erroringProvider := newFakeProvider()
	erroringProvider.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) {
		return "", errors.New("title failure")
	}
	newFakeChatServer(store, erroringProvider).generateAndSaveConversationTitle(user.ID, conversationID, modelGPT54Mini, "user", "assistant")

	store.close()
	newFakeChatServer(store, fake).generateAndSaveConversationTitle(user.ID, conversationID, modelGPT54Mini, "user", "assistant")
}

func TestListAndLoadConversationBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "list-load@example.com")
	other := mustCreateUser(t, store, "list-load-other@example.com")
	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	longTitle := "This is a deliberately very long conversation title that should be truncated in summaries"
	if err := store.saveConversationTitle(user.ID, conversationID, longTitle); err != nil {
		t.Fatalf("saveConversationTitle: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "ASSISTANT", "hello", modelGPT54Mini, 5, 7); err != nil {
		t.Fatalf("saveConversationMessage: %v", err)
	}

	server := &chatServer{defaultModel: modelGPT54Mini, store: store, logger: newTestLogger(), sessions: map[string]*sessionState{}, authUsers: map[string]authUser{}}
	ctx := bindAuthUser(server, "peer-list-load", user.ID, user.Email)

	listResp, err := server.ListConversations(ctx, &chatpb.ListConversationsRequest{})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(listResp.GetConversations()) != 1 {
		t.Fatalf("expected one conversation summary, got %+v", listResp)
	}
	if preview := listResp.GetConversations()[0].GetPreview(); len(preview) != 63 || preview[len(preview)-3:] != "…" {
		t.Fatalf("expected truncated preview with ellipsis, got %q", preview)
	}

	publicID := listResp.GetConversations()[0].GetPublicId()
	if publicID == "" {
		t.Fatal("expected conversation summary public_id")
	}

	routeResp, err := server.ResolveConversationRoute(ctx, &chatpb.ResolveConversationRouteRequest{PublicId: publicID})
	if err != nil {
		t.Fatalf("ResolveConversationRoute owner: %v", err)
	}
	if !routeResp.GetAccessible() || routeResp.GetId() != conversationID {
		t.Fatalf("unexpected owner route response: %+v", routeResp)
	}

	otherCtx := bindAuthUser(server, "peer-list-load-other", other.ID, other.Email)
	otherRouteResp, err := server.ResolveConversationRoute(otherCtx, &chatpb.ResolveConversationRouteRequest{PublicId: publicID})
	if err != nil {
		t.Fatalf("ResolveConversationRoute other: %v", err)
	}
	if otherRouteResp.GetAccessible() || otherRouteResp.GetId() != 0 {
		t.Fatalf("expected inaccessible route for non-owner, got %+v", otherRouteResp)
	}

	loadResp, err := server.LoadConversation(ctx, &chatpb.LoadConversationRequest{Id: conversationID})
	if err != nil {
		t.Fatalf("LoadConversation: %v", err)
	}
	if len(loadResp.GetMessages()) != 1 || loadResp.GetMessages()[0].GetModelId() != modelGPT54Mini {
		t.Fatalf("unexpected load conversation response: %+v", loadResp)
	}
	if loadResp.GetMessages()[0].GetRole() != "assistant" {
		t.Fatalf("expected normalized assistant role, got %q", loadResp.GetMessages()[0].GetRole())
	}

	store.close()
	if _, err := server.ListConversations(ctx, &chatpb.ListConversationsRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal ListConversations error after store close, got %v", status.Code(err))
	}
	if _, err := server.ResolveConversationRoute(ctx, &chatpb.ResolveConversationRouteRequest{PublicId: publicID}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal ResolveConversationRoute error after store close, got %v", status.Code(err))
	}
	if _, err := server.LoadConversation(ctx, &chatpb.LoadConversationRequest{Id: conversationID}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal LoadConversation error after store close, got %v", status.Code(err))
	}
}

func TestSynthesizeSpeechAdditionalBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "speech-branches@example.com")

	noProviderServer := &chatServer{logger: newTestLogger(), store: store, sessions: map[string]*sessionState{}, authUsers: map[string]authUser{}}
	noProviderCtx := bindAuthUser(noProviderServer, "peer-no-provider-speech", user.ID, user.Email)
	if err := noProviderServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello"}, &fakeSpeechStream{ctx: noProviderCtx}); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected unavailable when provider registry is nil, got %v", status.Code(err))
	}

	unsupportedModelServer := newFakeChatServer(store, newFakeProvider())
	unsupportedCtx := bindAuthUser(unsupportedModelServer, "peer-unsupported-model-speech", user.ID, user.Email)
	if err := unsupportedModelServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: "missing-model"}, &fakeSpeechStream{ctx: unsupportedCtx}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for unsupported speech model, got %v", status.Code(err))
	}

	emptyAudioProvider := newFakeProvider()
	emptyAudioProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, emit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		return provider.SpeechResult{MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: "hello"}, emit(provider.SpeechChunk{Done: true})
	}
	emptyAudioServer := newFakeChatServer(store, emptyAudioProvider)
	emptyAudioCtx := bindAuthUser(emptyAudioServer, "peer-empty-audio-speech", user.ID, user.Email)
	if err := emptyAudioServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: emptyAudioCtx}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for empty synthesized audio, got %v", status.Code(err))
	}

	statusErrProvider := newFakeProvider()
	statusErrProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, _ func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		return provider.SpeechResult{}, status.Error(codes.Unavailable, "provider busy")
	}
	statusErrServer := newFakeChatServer(store, statusErrProvider)
	statusErrCtx := bindAuthUser(statusErrServer, "peer-status-err-speech", user.ID, user.Email)
	if err := statusErrServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: modelGPT54Mini}, &fakeSpeechStream{ctx: statusErrCtx}); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected provider gRPC status to pass through, got %v", status.Code(err))
	}

	streamFailProvider := newFakeProvider()
	streamFailProvider.synthesizeSpeech = func(_ context.Context, _ provider.SpeechRequest, emit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		if err := emit(provider.SpeechChunk{AudioChunk: []byte("abc")}); err != nil {
			return provider.SpeechResult{}, err
		}
		return provider.SpeechResult{MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: "hello"}, nil
	}
	streamFailServer := newFakeChatServer(store, streamFailProvider)
	streamFailCtx := bindAuthUser(streamFailServer, "peer-stream-fail-speech", user.ID, user.Email)
	if err := streamFailServer.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "hello", Model: modelGPT54Mini}, &failingSpeechStream{ctx: streamFailCtx}); status.Code(err) != codes.Canceled {
		t.Fatalf("expected canceled status for downstream stream send failure, got %v", status.Code(err))
	}
}
