package app

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestChatServerPreferenceAndConversationRPCs(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "rpc@example.com")
	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "user", "Hi", "", 0, 0); err != nil {
		t.Fatalf("saveConversationMessage user: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "assistant", "Hello", modelGPT54Mini, 10, 5); err != nil {
		t.Fatalf("saveConversationMessage assistant: %v", err)
	}

	server := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        store,
		logger:       newTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	ctx := bindAuthUser(server, "peer-rpc", user.ID, user.Email)

	if _, err := server.SetUserName(ctx, &chatpb.SetUserNameRequest{Name: "Updated"}); err != nil {
		t.Fatalf("SetUserName: %v", err)
	}
	nameResp, err := server.GetUserName(ctx, &chatpb.GetUserNameRequest{})
	if err != nil || nameResp.GetName() != "Updated" {
		t.Fatalf("GetUserName: resp=%+v err=%v", nameResp, err)
	}

	if _, err := server.SetSelectedTone(ctx, wrapperspb.String("professional")); err != nil {
		t.Fatalf("SetSelectedTone: %v", err)
	}
	if _, err := server.SetSelectedThinkingEnabled(ctx, wrapperspb.Bool(false)); err != nil {
		t.Fatalf("SetSelectedThinkingEnabled: %v", err)
	}
	if _, err := server.SetSelectedThinkingEffort(ctx, wrapperspb.String("high")); err != nil {
		t.Fatalf("SetSelectedThinkingEffort: %v", err)
	}
	if _, err := server.SetCustomSystemPrompt(ctx, wrapperspb.String("Address me as Captain.")); err != nil {
		t.Fatalf("SetCustomSystemPrompt: %v", err)
	}

	toneResp, err := server.GetSelectedTone(ctx, &emptypb.Empty{})
	if err != nil || toneResp.GetValue() != "professional" {
		t.Fatalf("GetSelectedTone: resp=%+v err=%v", toneResp, err)
	}
	enabledResp, err := server.GetSelectedThinkingEnabled(ctx, &emptypb.Empty{})
	if err != nil || enabledResp.GetValue() {
		t.Fatalf("GetSelectedThinkingEnabled: resp=%+v err=%v", enabledResp, err)
	}
	effortResp, err := server.GetSelectedThinkingEffort(ctx, &emptypb.Empty{})
	if err != nil || effortResp.GetValue() != "high" {
		t.Fatalf("GetSelectedThinkingEffort: resp=%+v err=%v", effortResp, err)
	}
	systemPromptResp, err := server.GetCustomSystemPrompt(ctx, &emptypb.Empty{})
	if err != nil || systemPromptResp.GetValue() != "Address me as Captain." {
		t.Fatalf("GetCustomSystemPrompt: resp=%+v err=%v", systemPromptResp, err)
	}

	listResp, err := server.ListConversations(ctx, &chatpb.ListConversationsRequest{})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(listResp.Conversations) != 1 {
		t.Fatalf("expected one conversation, got %d", len(listResp.Conversations))
	}
	loadResp, err := server.LoadConversation(ctx, &chatpb.LoadConversationRequest{Id: conversationID})
	if err != nil {
		t.Fatalf("LoadConversation: %v", err)
	}
	if len(loadResp.Messages) != 2 {
		t.Fatalf("expected two loaded messages, got %d", len(loadResp.Messages))
	}

	if _, err := server.DeleteConversation(ctx, &chatpb.DeleteConversationRequest{Id: conversationID}); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	remaining, err := server.ListConversations(ctx, &chatpb.ListConversationsRequest{})
	if err != nil {
		t.Fatalf("ListConversations after delete: %v", err)
	}
	if len(remaining.Conversations) != 0 {
		t.Fatalf("expected zero conversations after delete, got %d", len(remaining.Conversations))
	}

	server.unbindAuthenticatedPeer("peer-rpc")
}

func TestChatServerRejectsUnauthenticatedRPCs(t *testing.T) {
	server := &chatServer{logger: newTestLogger(), sessions: map[string]*sessionState{}, authUsers: map[string]authUser{}}
	ctx := context.Background()

	_, err := server.GetUserName(ctx, &chatpb.GetUserNameRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated status, got %v", status.Code(err))
	}

	_, err = server.SetUserName(newAuthenticatedContext("orphan-peer"), &chatpb.SetUserNameRequest{Name: "User"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated status for unbound peer, got %v", status.Code(err))
	}
}

func TestChatServerUserMemoryRPCs(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "memories@example.com")
	server := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        store,
		logger:       newTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	ctx := bindAuthUser(server, "peer-memories", user.ID, user.Email)

	if _, err := server.UpsertUserMemory(ctx, &chatpb.UpsertUserMemoryRequest{
		Memory: &chatpb.UserMemory{
			Category:     "Preference",
			Summary:      "Prefers concise answers",
			Detail:       "Usually wants a short final response.",
			RubricReason: "Stable response preference.",
		},
	}); err != nil {
		t.Fatalf("UpsertUserMemory create: %v", err)
	}

	listResp, err := server.ListUserMemories(ctx, &chatpb.ListUserMemoriesRequest{})
	if err != nil {
		t.Fatalf("ListUserMemories after create: %v", err)
	}
	if len(listResp.GetMemories()) != 1 {
		t.Fatalf("expected one stored memory, got %d", len(listResp.GetMemories()))
	}
	created := listResp.GetMemories()[0]
	if created.GetKey() == "" {
		t.Fatal("expected created memory key to be derived")
	}
	if created.GetCategory() != "preference" {
		t.Fatalf("expected normalized category, got %q", created.GetCategory())
	}
	if created.GetSummary() != "Prefers concise answers" {
		t.Fatalf("unexpected created summary: %q", created.GetSummary())
	}

	if _, err := server.UpsertUserMemory(ctx, &chatpb.UpsertUserMemoryRequest{
		Memory: &chatpb.UserMemory{
			Key:          created.GetKey(),
			Category:     created.GetCategory(),
			Summary:      created.GetSummary(),
			Detail:       "Wants short answers unless more depth is requested.",
			RubricReason: "User consistently asks for brevity.",
		},
	}); err != nil {
		t.Fatalf("UpsertUserMemory update: %v", err)
	}

	updatedResp, err := server.ListUserMemories(ctx, &chatpb.ListUserMemoriesRequest{})
	if err != nil {
		t.Fatalf("ListUserMemories after update: %v", err)
	}
	if len(updatedResp.GetMemories()) != 1 {
		t.Fatalf("expected one stored memory after update, got %d", len(updatedResp.GetMemories()))
	}
	updated := updatedResp.GetMemories()[0]
	if updated.GetDetail() != "Wants short answers unless more depth is requested." {
		t.Fatalf("unexpected updated detail: %q", updated.GetDetail())
	}
	if updated.GetRubricReason() != "User consistently asks for brevity." {
		t.Fatalf("unexpected updated rubric reason: %q", updated.GetRubricReason())
	}

	if _, err := server.DeleteUserMemory(ctx, &chatpb.DeleteUserMemoryRequest{Key: created.GetKey()}); err != nil {
		t.Fatalf("DeleteUserMemory: %v", err)
	}

	finalResp, err := server.ListUserMemories(ctx, &chatpb.ListUserMemoriesRequest{})
	if err != nil {
		t.Fatalf("ListUserMemories after delete: %v", err)
	}
	if len(finalResp.GetMemories()) != 0 {
		t.Fatalf("expected zero stored memories after delete, got %d", len(finalResp.GetMemories()))
	}

	server.unbindAuthenticatedPeer("peer-memories")
}

func TestSendStreamsThoughtsAndPersistsConversation(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "send@example.com")
	fake := newFakeProvider()
	fake.streamChat = func(_ context.Context, req provider.ChatRequest, emit func(provider.ChatEvent) error) (provider.ChatResult, error) {
		if err := emit(provider.ChatEvent{ThoughtDelta: "Thinking..."}); err != nil {
			return provider.ChatResult{}, err
		}
		if err := emit(provider.ChatEvent{ThoughtDone: true}); err != nil {
			return provider.ChatResult{}, err
		}
		if err := emit(provider.ChatEvent{TextDelta: "Hello"}); err != nil {
			return provider.ChatResult{}, err
		}
		if err := emit(provider.ChatEvent{TextDelta: " world"}); err != nil {
			return provider.ChatResult{}, err
		}
		return provider.ChatResult{Model: modelGPT54, PromptTokens: 21, CompletionTokens: 9}, nil
	}
	fake.generateTitle = func(_ context.Context, req provider.TitleRequest) (string, error) {
		return "Generated title", nil
	}
	fake.extractUserMemories = func(_ context.Context, req provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		return []provider.UserMemoryCandidate{{
			Key:             "preference-editor",
			Category:        "preference",
			Summary:         "Prefers Neovim",
			Detail:          "Uses Neovim for most coding work",
			UsefulnessScore: 84,
			ConfidenceScore: 0.92,
			RubricReason:    "Stable tooling preference",
		}}, nil
	}
	server := newFakeChatServer(store, fake)
	if err := store.setSelectedSystemPrompt(user.ID, "Be extra terse."); err != nil {
		t.Fatalf("setSelectedSystemPrompt: %v", err)
	}
	if err := store.upsertUserMemory(user.ID, userMemoryRow{
		Key:             "pref-editor",
		Category:        "preference",
		Summary:         "Prefers Neovim",
		Detail:          "Uses it for coding",
		SourceMessage:   "I use Neovim for coding.",
		UsefulnessScore: 88,
		ConfidenceScore: 0.95,
		RubricReason:    "Stable tooling preference",
	}); err != nil {
		t.Fatalf("upsertUserMemory seed: %v", err)
	}
	ctx := bindAuthUser(server, "peer-send", user.ID, user.Email)
	stream := &fakeChatSendStream{ctx: ctx}

	req := &chatpb.SendRequest{
		History: []*chatpb.ChatMessage{{Role: "user", Content: "Earlier"}},
		Message: "Current question",
		Tone:    "professional",
		Model:   modelGPT54Mini,
	}
	if err := server.Send(req, stream); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(stream.chunks) != 5 {
		t.Fatalf("expected 5 streamed chunks, got %d", len(stream.chunks))
	}
	if stream.chunks[0].GetModel() != thoughtChunkModelPrefix+"Thinking..." {
		t.Fatalf("unexpected thought chunk: %+v", stream.chunks[0])
	}
	if !stream.chunks[4].GetDone() || stream.chunks[4].GetModel() != modelGPT54 {
		t.Fatalf("unexpected terminal chunk: %+v", stream.chunks[4])
	}

	conversations, err := store.listConversations(user.ID)
	if err != nil {
		t.Fatalf("listConversations: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("expected one persisted conversation, got %d", len(conversations))
	}
	messages, err := store.loadConversation(user.ID, conversations[0].ID)
	if err != nil {
		t.Fatalf("loadConversation: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected history + user + assistant messages, got %d", len(messages))
	}
	if messages[2].Content != "Hello world" || messages[2].ModelID != modelGPT54 {
		t.Fatalf("unexpected assistant message: %+v", messages[2])
	}
	if !strings.Contains(fake.lastStreamChatRequest.SystemPrompt, "Be extra terse.") {
		t.Fatalf("expected custom system prompt in chat request, got %q", fake.lastStreamChatRequest.SystemPrompt)
	}
	if !strings.Contains(fake.lastStreamChatRequest.SystemPrompt, "Prefers Neovim") {
		t.Fatalf("expected stored user memory in chat request, got %q", fake.lastStreamChatRequest.SystemPrompt)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		updated, listErr := store.listConversations(user.ID)
		if listErr != nil {
			t.Fatalf("listConversations title poll: %v", listErr)
		}
		if len(updated) == 1 && updated[0].Preview == "Generated title" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for generated title")
		}
		time.Sleep(20 * time.Millisecond)
	}

	memoryDeadline := time.Now().Add(2 * time.Second)
	for {
		memories, listErr := store.listUserMemories(user.ID)
		if listErr != nil {
			t.Fatalf("listUserMemories: %v", listErr)
		}
		for _, memory := range memories {
			if memory.Summary == "Prefers Neovim" {
				return
			}
		}
		if time.Now().After(memoryDeadline) {
			t.Fatal("timed out waiting for extracted user memory")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSendRejectsConversationOwnedByAnotherUser(t *testing.T) {
	store := newTestStore(t)
	owner := mustCreateUser(t, store, "owner@example.com")
	other := mustCreateUser(t, store, "other@example.com")
	conversationID, err := store.createConversation(owner.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	fake := newFakeProvider()
	fake.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		return provider.ChatResult{}, nil
	}
	fake.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) {
		return "", nil
	}
	server := newFakeChatServer(store, fake)
	ctx := bindAuthUser(server, "peer-permission", other.ID, other.Email)
	stream := &fakeChatSendStream{ctx: ctx}

	err = server.Send(&chatpb.SendRequest{ConversationId: conversationID, Message: "Blocked"}, stream)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", status.Code(err))
	}
}

func TestSendRejectsMissingAuthenticatedUserBeforeProviderWork(t *testing.T) {
	store := newTestStore(t)
	fake := newFakeProvider()
	fake.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		t.Fatal("StreamChat should not be called for a missing authenticated user")
		return provider.ChatResult{}, nil
	}
	fake.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) {
		return "", nil
	}
	server := newFakeChatServer(store, fake)
	ctx := bindAuthUser(server, "peer-missing-user", 999999, "ghost@example.com")
	stream := &fakeChatSendStream{ctx: ctx}

	err := server.Send(&chatpb.SendRequest{Message: "hello"}, stream)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", status.Code(err))
	}
}

func TestSynthesizeSpeechStreamsChunks(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "speech@example.com")
	fake := newFakeProvider()
	fake.synthesizeSpeech = func(_ context.Context, req provider.SpeechRequest, emit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		if err := emit(provider.SpeechChunk{AudioChunk: []byte("abc"), MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: req.Text}); err != nil {
			return provider.SpeechResult{}, err
		}
		if err := emit(provider.SpeechChunk{Done: true, MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: req.Text}); err != nil {
			return provider.SpeechResult{}, err
		}
		return provider.SpeechResult{MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: req.Text}, nil
	}
	server := newFakeChatServer(store, fake)
	ctx := bindAuthUser(server, "peer-speech", user.ID, user.Email)
	stream := &fakeSpeechStream{ctx: ctx}

	err := server.SynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "Hello [site](https://example.com) ```code```", Model: modelGPT54Mini}, stream)
	if err != nil {
		t.Fatalf("SynthesizeSpeech: %v", err)
	}
	if len(stream.chunks) != 2 {
		t.Fatalf("expected two speech chunks, got %d", len(stream.chunks))
	}
	if stream.chunks[0].GetScript() != "Hello site" {
		t.Fatalf("unexpected sanitized script: %q", stream.chunks[0].GetScript())
	}
	if !stream.chunks[1].GetDone() {
		t.Fatalf("expected terminal speech chunk, got %+v", stream.chunks[1])
	}
}

func TestHTTPIntegrationProtectsAndServesChatShell(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat-bootstrap.js", serveChatBootstrapJS)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			serveChatShell(w, r)
			return
		}
		http.NotFound(w, r)
	}))

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("GET shell: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll shell body: %v", err)
	}
	if !strings.Contains(string(body), "Preparing chat runtime") {
		t.Fatal("expected shell response to include boot shell content")
	}

	bootstrapResp, err := client.Get(testServer.URL + "/chat-bootstrap.js")
	if err != nil {
		t.Fatalf("GET bootstrap: %v", err)
	}
	bootstrapBody, err := io.ReadAll(bootstrapResp.Body)
	if err != nil {
		t.Fatalf("ReadAll bootstrap body: %v", err)
	}
	if !strings.Contains(string(bootstrapBody), "loadChatWasm") {
		t.Fatal("expected bootstrap JS endpoint to return chat bootstrap source")
	}
}
