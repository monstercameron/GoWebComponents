package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

type stubAddr string

func (a stubAddr) Network() string { return "test" }

func (a stubAddr) String() string { return string(a) }

type fakeProvider struct {
	id                    string
	defaultModel          string
	supportedModels       map[string]provider.ModelCapabilities
	streamChat            func(context.Context, provider.ChatRequest, func(provider.ChatEvent) error) (provider.ChatResult, error)
	generateTitle         func(context.Context, provider.TitleRequest) (string, error)
	extractUserMemories   func(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error)
	synthesizeSpeech      func(context.Context, provider.SpeechRequest, func(provider.SpeechChunk) error) (provider.SpeechResult, error)
	modelOptions          []provider.ModelOption
	available             bool
	lastGenerateTitleReq  provider.TitleRequest
	lastMemoryExtractReq  provider.MemoryExtractionRequest
	lastSynthesizeReq     provider.SpeechRequest
	lastStreamChatRequest provider.ChatRequest
}

func (p *fakeProvider) ID() string { return p.id }

func (p *fakeProvider) Available() bool { return p != nil && p.available }

func (p *fakeProvider) Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		ID:                 p.id,
		Label:              "Fake",
		AuthConfigured:     p.available,
		Available:          p.available,
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   true,
	}
}

func (p *fakeProvider) DefaultModel() string { return p.defaultModel }

func (p *fakeProvider) SupportsModel(model string) bool {
	_, ok := p.supportedModels[normalizeSelectedModelID(model)]
	return ok
}

func (p *fakeProvider) ModelOptions() []provider.ModelOption { return p.modelOptions }

func (p *fakeProvider) ModelMetadata(model string) (provider.ModelMetadata, bool) {
	capabilities := p.Capabilities(model)
	for _, option := range p.modelOptions {
		if option.ID != normalizeSelectedModelID(model) {
			continue
		}
		return provider.ModelMetadata{
			ID:                 option.ID,
			DisplayName:        option.Label,
			Description:        option.Note,
			ProviderID:         p.id,
			ProviderLabel:      "Fake",
			ProviderFamily:     "fake",
			Capabilities:       capabilities,
			StreamingSupported: true,
			ReasoningSupported: capabilities.SupportsThinking,
			ToolUseSupported:   true,
			OnboardingReady:    true,
		}, true
	}
	return provider.ModelMetadata{}, false
}

func (p *fakeProvider) Capabilities(model string) provider.ModelCapabilities {
	if capabilities, ok := p.supportedModels[normalizeSelectedModelID(model)]; ok {
		return capabilities
	}
	return provider.ModelCapabilities{ProviderID: p.id, ProviderLabel: p.id}
}

func (p *fakeProvider) Health() provider.ProviderHealth {
	status := provider.ProviderHealthUnavailable
	if p.Available() {
		status = provider.ProviderHealthUnknown
	}
	return provider.ProviderHealth{ProviderID: p.id, Status: status}
}

func (p *fakeProvider) CurrentRateLimits() provider.RateLimitSnapshot {
	return provider.RateLimitSnapshot{}
}

func (p *fakeProvider) StreamChat(ctx context.Context, req provider.ChatRequest, emit func(provider.ChatEvent) error) (provider.ChatResult, error) {
	p.lastStreamChatRequest = req
	return p.streamChat(ctx, req, emit)
}

func (p *fakeProvider) GenerateTitle(ctx context.Context, req provider.TitleRequest) (string, error) {
	p.lastGenerateTitleReq = req
	return p.generateTitle(ctx, req)
}

func (p *fakeProvider) ExtractUserMemories(ctx context.Context, req provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
	p.lastMemoryExtractReq = req
	return p.extractUserMemories(ctx, req)
}

func (p *fakeProvider) SynthesizeSpeech(ctx context.Context, req provider.SpeechRequest, emit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
	p.lastSynthesizeReq = req
	return p.synthesizeSpeech(ctx, req, emit)
}

type fakeChatSendStream struct {
	ctx    context.Context
	chunks []*chatpb.ChatChunk
}

func (s *fakeChatSendStream) Send(chunk *chatpb.ChatChunk) error {
	copyChunk := *chunk
	s.chunks = append(s.chunks, &copyChunk)
	return nil
}

func (s *fakeChatSendStream) SetHeader(metadata.MD) error { return nil }

func (s *fakeChatSendStream) SendHeader(metadata.MD) error { return nil }

func (s *fakeChatSendStream) SetTrailer(metadata.MD) {}

func (s *fakeChatSendStream) Context() context.Context { return s.ctx }

func (s *fakeChatSendStream) SendMsg(any) error { return nil }

func (s *fakeChatSendStream) RecvMsg(any) error { return nil }

type fakeSpeechStream struct {
	ctx    context.Context
	chunks []*chatpb.SynthesizeSpeechChunk
}

func (s *fakeSpeechStream) Send(chunk *chatpb.SynthesizeSpeechChunk) error {
	copyChunk := *chunk
	s.chunks = append(s.chunks, &copyChunk)
	return nil
}

func (s *fakeSpeechStream) SetHeader(metadata.MD) error { return nil }

func (s *fakeSpeechStream) SendHeader(metadata.MD) error { return nil }

func (s *fakeSpeechStream) SetTrailer(metadata.MD) {}

func (s *fakeSpeechStream) Context() context.Context { return s.ctx }

func (s *fakeSpeechStream) SendMsg(any) error { return nil }

func (s *fakeSpeechStream) RecvMsg(any) error { return nil }

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := openChatStore(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatalf("openChatStore: %v", err)
	}
	t.Cleanup(store.close)
	return store
}

func newAuthenticatedContext(peerName string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{Addr: stubAddr(peerName)})
}

func bindAuthUser(server *chatServer, peerName string, userID int64, email string) context.Context {
	ctx := newAuthenticatedContext(peerName)
	server.bindAuthenticatedPeer(peerName, authUser{ID: userID, Email: email})
	return ctx
}

func mustCreateUser(t *testing.T, store *Store, email string) authUser {
	t.Helper()
	userID, err := store.createUser(email, "hash", "Demo User")
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}
	return authUser{ID: userID, Email: normalizeAuthEmail(email)}
}

func newFakeProvider() *fakeProvider {
	capabilities := provider.ModelCapabilities{
		ProviderID:       "fake",
		ProviderLabel:    "Fake",
		SupportsThinking: true,
		SupportsSpeech:   true,
	}
	return &fakeProvider{
		id:           "fake",
		defaultModel: modelGPT54Mini,
		supportedModels: map[string]provider.ModelCapabilities{
			modelGPT54Mini: capabilities,
			modelGPT54:     capabilities,
		},
		modelOptions: []provider.ModelOption{{
			ID:           modelGPT54Mini,
			Label:        "GPT-5.4 mini",
			Note:         "Fast",
			Capabilities: capabilities,
		}},
		available: true,
		extractUserMemories: func(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
			return nil, nil
		},
	}
}

func newFakeChatServer(store *Store, fake *fakeProvider) *chatServer {
	server := &chatServer{
		providerRegistry: provider.NewRegistry(fake),
		defaultModel:     modelGPT54Mini,
		store:            store,
		logger:           newTestLogger(),
		sessions:         map[string]*sessionState{},
		authUsers:        map[string]authUser{},
	}
	return server
}

func verifyEmptyAndStringWrappersCompile(_ *emptypb.Empty, _ *wrapperspb.StringValue) {}

var _ net.Addr = stubAddr("")
