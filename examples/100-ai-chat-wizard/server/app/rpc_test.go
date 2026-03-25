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

func TestChatServerPreferenceAndConversationRPCs(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "rpc@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ParseID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseSaveConversationMessage(parseUser.ParseID, parseConversationID, "user", "Hi", "", 0, 0); parseErr2 != nil {
		parseT.Fatalf("saveConversationMessage user: %v", parseErr2)
	}
	if parseErr3 := store.parseSaveConversationMessage(parseUser.ParseID, parseConversationID, "assistant", "Hello", modelGPT54Mini, 10, 5); parseErr3 != nil {
		parseT.Fatalf("saveConversationMessage assistant: %v", parseErr3)
	}

	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        store,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-rpc", parseUser.ParseID, parseUser.Email)

	if _, parseErr4 := parseServer.SetUserName(parseCtx, &chatpb.SetUserNameRequest{Name: "Updated"}); parseErr4 != nil {
		parseT.Fatalf("SetUserName: %v", parseErr4)
	}
	parseNameResp, parseErr := parseServer.GetUserName(parseCtx, &chatpb.GetUserNameRequest{})
	if parseErr != nil || parseNameResp.GetName() != "Updated" {
		parseT.Fatalf("GetUserName: resp=%+v err=%v", parseNameResp, parseErr)
	}

	if _, parseErr5 := parseServer.SetSelectedTone(parseCtx, wrapperspb.String("professional")); parseErr5 != nil {
		parseT.Fatalf("SetSelectedTone: %v", parseErr5)
	}
	if _, parseErr6 := parseServer.SetSelectedThinkingEnabled(parseCtx, wrapperspb.Bool(false)); parseErr6 != nil {
		parseT.Fatalf("SetSelectedThinkingEnabled: %v", parseErr6)
	}
	if _, parseErr7 := parseServer.SetSelectedThinkingEffort(parseCtx, wrapperspb.String("high")); parseErr7 != nil {
		parseT.Fatalf("SetSelectedThinkingEffort: %v", parseErr7)
	}
	if _, parseErr8 := parseServer.SetCustomSystemPrompt(parseCtx, wrapperspb.String("Address me as Captain.")); parseErr8 != nil {
		parseT.Fatalf("SetCustomSystemPrompt: %v", parseErr8)
	}

	parseToneResp, parseErr := parseServer.GetSelectedTone(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseToneResp.GetValue() != "professional" {
		parseT.Fatalf("GetSelectedTone: resp=%+v err=%v", parseToneResp, parseErr)
	}
	parseEnabledResp, parseErr := parseServer.GetSelectedThinkingEnabled(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseEnabledResp.GetValue() {
		parseT.Fatalf("GetSelectedThinkingEnabled: resp=%+v err=%v", parseEnabledResp, parseErr)
	}
	parseEffortResp, parseErr := parseServer.GetSelectedThinkingEffort(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseEffortResp.GetValue() != "high" {
		parseT.Fatalf("GetSelectedThinkingEffort: resp=%+v err=%v", parseEffortResp, parseErr)
	}
	parseSystemPromptResp, parseErr := parseServer.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseSystemPromptResp.GetValue() != "Address me as Captain." {
		parseT.Fatalf("GetCustomSystemPrompt: resp=%+v err=%v", parseSystemPromptResp, parseErr)
	}

	parseListResp, parseErr := parseServer.ParseListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations: %v", parseErr)
	}
	if len(parseListResp.Conversations) != 1 {
		parseT.Fatalf("expected one conversation, got %d", len(parseListResp.Conversations))
	}
	parseLoadResp, parseErr := parseServer.ParseLoadConversation(parseCtx, &chatpb.LoadConversationRequest{Id: parseConversationID})
	if parseErr != nil {
		parseT.Fatalf("LoadConversation: %v", parseErr)
	}
	if len(parseLoadResp.Messages) != 2 {
		parseT.Fatalf("expected two loaded messages, got %d", len(parseLoadResp.Messages))
	}

	if _, parseErr9 := parseServer.ParseDeleteConversation(parseCtx, &chatpb.DeleteConversationRequest{Id: parseConversationID}); parseErr9 != nil {
		parseT.Fatalf("DeleteConversation: %v", parseErr9)
	}
	parseRemaining, parseErr := parseServer.ParseListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations after delete: %v", parseErr)
	}
	if len(parseRemaining.Conversations) != 0 {
		parseT.Fatalf("expected zero conversations after delete, got %d", len(parseRemaining.Conversations))
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-rpc")
}

func TestChatServerRejectsUnauthenticatedRPCs(parseT *testing.T) {
	parseServer := &chatServer{logger: parseNewTestLogger(), sessions: map[string]*sessionState{}, authUsers: map[string]authUser{}}
	parseCtx := context.Background()

	_, parseErr := parseServer.GetUserName(parseCtx, &chatpb.GetUserNameRequest{})
	if status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected unauthenticated status, got %v", status.Code(parseErr))
	}

	_, parseErr = parseServer.SetUserName(parseNewAuthenticatedContext("orphan-peer"), &chatpb.SetUserNameRequest{Name: "User"})
	if status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected unauthenticated status for unbound peer, got %v", status.Code(parseErr))
	}
}

func TestChatServerUserMemoryRPCs(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "memories@example.com")
	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        store,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-memories", parseUser.ParseID, parseUser.Email)

	if _, parseErr := parseServer.ParseUpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{
		Memory: &chatpb.UserMemory{
			Category:     "Preference",
			Summary:      "Prefers concise answers",
			Detail:       "Usually wants a short final response.",
			RubricReason: "Stable response preference.",
		},
	}); parseErr != nil {
		parseT.Fatalf("UpsertUserMemory create: %v", parseErr)
	}

	parseListResp, parseErr2 := parseServer.ParseListUserMemories(parseCtx, &chatpb.ListUserMemoriesRequest{})
	if parseErr2 != nil {
		parseT.Fatalf("ListUserMemories after create: %v", parseErr2)
	}
	if len(parseListResp.GetMemories()) != 1 {
		parseT.Fatalf("expected one stored memory, got %d", len(parseListResp.GetMemories()))
	}
	parseCreated := parseListResp.GetMemories()[0]
	if parseCreated.GetKey() == "" {
		parseT.Fatal("expected created memory key to be derived")
	}
	if parseCreated.GetCategory() != "preference" {
		parseT.Fatalf("expected normalized category, got %q", parseCreated.GetCategory())
	}
	if parseCreated.GetSummary() != "Prefers concise answers" {
		parseT.Fatalf("unexpected created summary: %q", parseCreated.GetSummary())
	}

	if _, parseErr3 := parseServer.ParseUpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{
		Memory: &chatpb.UserMemory{
			Key:          parseCreated.GetKey(),
			Category:     parseCreated.GetCategory(),
			Summary:      parseCreated.GetSummary(),
			Detail:       "Wants short answers unless more depth is requested.",
			RubricReason: "User consistently asks for brevity.",
		},
	}); parseErr3 != nil {
		parseT.Fatalf("UpsertUserMemory update: %v", parseErr3)
	}

	parseUpdatedResp, parseErr2 := parseServer.ParseListUserMemories(parseCtx, &chatpb.ListUserMemoriesRequest{})
	if parseErr2 != nil {
		parseT.Fatalf("ListUserMemories after update: %v", parseErr2)
	}
	if len(parseUpdatedResp.GetMemories()) != 1 {
		parseT.Fatalf("expected one stored memory after update, got %d", len(parseUpdatedResp.GetMemories()))
	}
	parseUpdated := parseUpdatedResp.GetMemories()[0]
	if parseUpdated.GetDetail() != "Wants short answers unless more depth is requested." {
		parseT.Fatalf("unexpected updated detail: %q", parseUpdated.GetDetail())
	}
	if parseUpdated.GetRubricReason() != "User consistently asks for brevity." {
		parseT.Fatalf("unexpected updated rubric reason: %q", parseUpdated.GetRubricReason())
	}

	if _, parseErr4 := parseServer.ParseDeleteUserMemory(parseCtx, &chatpb.DeleteUserMemoryRequest{Key: parseCreated.GetKey()}); parseErr4 != nil {
		parseT.Fatalf("DeleteUserMemory: %v", parseErr4)
	}

	parseFinalResp, parseErr2 := parseServer.ParseListUserMemories(parseCtx, &chatpb.ListUserMemoriesRequest{})
	if parseErr2 != nil {
		parseT.Fatalf("ListUserMemories after delete: %v", parseErr2)
	}
	if len(parseFinalResp.GetMemories()) != 0 {
		parseT.Fatalf("expected zero stored memories after delete, got %d", len(parseFinalResp.GetMemories()))
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-memories")
}

func TestSendStreamsThoughtsAndPersistsConversation(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "send@example.com")
	parseFake := parseNewFakeProvider()
	parseFake.streamChat = func(_ context.Context, parseReq2 provider.ChatRequest, parseEmit func(provider.ChatEvent) error) (provider.ChatResult, error) {
		if parseErr := parseEmit(provider.ChatEvent{ThoughtDelta: "Thinking..."}); parseErr != nil {
			return provider.ChatResult{}, parseErr
		}
		if parseErr2 := parseEmit(provider.ChatEvent{ThoughtDone: true}); parseErr2 != nil {
			return provider.ChatResult{}, parseErr2
		}
		if parseErr3 := parseEmit(provider.ChatEvent{TextDelta: "Hello"}); parseErr3 != nil {
			return provider.ChatResult{}, parseErr3
		}
		if parseErr4 := parseEmit(provider.ChatEvent{TextDelta: " world"}); parseErr4 != nil {
			return provider.ChatResult{}, parseErr4
		}
		return provider.ChatResult{Model: modelGPT54, PromptTokens: 21, CompletionTokens: 9}, nil
	}
	parseFake.generateTitle = func(_ context.Context, parseReq3 provider.TitleRequest) (string, error) {
		return "Generated title", nil
	}
	parseFake.extractUserMemories = func(_ context.Context, parseReq4 provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
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
	parseServer := parseNewFakeChatServer(store, parseFake)
	if parseErr5 := store.setSelectedSystemPrompt(parseUser.ParseID, "Be extra terse."); parseErr5 != nil {
		parseT.Fatalf("setSelectedSystemPrompt: %v", parseErr5)
	}
	if parseErr6 := store.parseUpsertUserMemory(parseUser.ParseID, userMemoryRow{
		Key:             "pref-editor",
		Category:        "preference",
		Summary:         "Prefers Neovim",
		Detail:          "Uses it for coding",
		SourceMessage:   "I use Neovim for coding.",
		UsefulnessScore: 88,
		ConfidenceScore: 0.95,
		RubricReason:    "Stable tooling preference",
	}); parseErr6 != nil {
		parseT.Fatalf("upsertUserMemory seed: %v", parseErr6)
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-send", parseUser.ParseID, parseUser.Email)
	parseStream := &fakeChatSendStream{ctx: parseCtx}

	parseReq := &chatpb.SendRequest{
		History: []*chatpb.ChatMessage{{Role: "USER", Content: "Earlier"}},
		Message: "Current question",
		Tone:    "professional",
		Model:   modelGPT54Mini,
	}
	if parseErr7 := parseServer.ParseSend(parseReq, parseStream); parseErr7 != nil {
		parseT.Fatalf("Send: %v", parseErr7)
	}
	if len(parseStream.chunks) != 5 {
		parseT.Fatalf("expected 5 streamed chunks, got %d", len(parseStream.chunks))
	}
	if parseStream.chunks[0].GetModel() != thoughtChunkModelPrefix+"Thinking..." {
		parseT.Fatalf("unexpected thought chunk: %+v", parseStream.chunks[0])
	}
	if !parseStream.chunks[4].GetDone() || parseStream.chunks[4].GetModel() != modelGPT54 {
		parseT.Fatalf("unexpected terminal chunk: %+v", parseStream.chunks[4])
	}

	parseConversations, parseErr8 := store.parseListConversations(parseUser.ParseID)
	if parseErr8 != nil {
		parseT.Fatalf("listConversations: %v", parseErr8)
	}
	if len(parseConversations) != 1 {
		parseT.Fatalf("expected one persisted conversation, got %d", len(parseConversations))
	}
	parseMessages, parseErr8 := store.parseLoadConversation(parseUser.ParseID, parseConversations[0].ParseID)
	if parseErr8 != nil {
		parseT.Fatalf("loadConversation: %v", parseErr8)
	}
	if len(parseMessages) != 3 {
		parseT.Fatalf("expected history + user + assistant messages, got %d", len(parseMessages))
	}
	if parseMessages[2].Content != "Hello world" || parseMessages[2].ModelID != modelGPT54 {
		parseT.Fatalf("unexpected assistant message: %+v", parseMessages[2])
	}
	if !strings.Contains(parseFake.lastStreamChatRequest.SystemPrompt, "Be extra terse.") {
		parseT.Fatalf("expected custom system prompt in chat request, got %q", parseFake.lastStreamChatRequest.SystemPrompt)
	}
	if !strings.Contains(parseFake.lastStreamChatRequest.SystemPrompt, "Prefers Neovim") {
		parseT.Fatalf("expected stored user memory in chat request, got %q", parseFake.lastStreamChatRequest.SystemPrompt)
	}
	if len(parseFake.lastStreamChatRequest.History) != 1 || parseFake.lastStreamChatRequest.History[0].Role != "user" {
		parseT.Fatalf("expected normalized history role in provider request, got %+v", parseFake.lastStreamChatRequest.History)
	}

	parseDeadline := time.Now().Add(2 * time.Second)
	for {
		parseUpdated, parseListErr := store.parseListConversations(parseUser.ParseID)
		if parseListErr != nil {
			parseT.Fatalf("listConversations title poll: %v", parseListErr)
		}
		if len(parseUpdated) == 1 && parseUpdated[0].Preview == "Generated title" {
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatal("timed out waiting for generated title")
		}
		time.Sleep(20 * time.Millisecond)
	}

	parseMemoryDeadline := time.Now().Add(2 * time.Second)
	for {
		parseMemories, parseListErr2 := store.parseListUserMemories(parseUser.ParseID)
		if parseListErr2 != nil {
			parseT.Fatalf("listUserMemories: %v", parseListErr2)
		}
		for _, parseMemory := range parseMemories {
			if parseMemory.Summary == "Prefers Neovim" {
				return
			}
		}
		if time.Now().After(parseMemoryDeadline) {
			parseT.Fatal("timed out waiting for extracted user memory")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSendRejectsConversationOwnedByAnotherUser(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, store, "owner@example.com")
	parseOther := parseMustCreateUser(parseT, store, "other@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseOwner.ParseID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	parseFake := parseNewFakeProvider()
	parseFake.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		return provider.ChatResult{}, nil
	}
	parseFake.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) {
		return "", nil
	}
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-permission", parseOther.ParseID, parseOther.Email)
	parseStream := &fakeChatSendStream{ctx: parseCtx}

	parseErr = parseServer.ParseSend(&chatpb.SendRequest{ConversationId: parseConversationID, Message: "Blocked"}, parseStream)
	if status.Code(parseErr) != codes.NotFound {
		parseT.Fatalf("expected not found, got %v", status.Code(parseErr))
	}
}

func TestSendRejectsMissingAuthenticatedUserBeforeProviderWork(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseFake := parseNewFakeProvider()
	parseFake.streamChat = func(_ context.Context, _ provider.ChatRequest, _ func(provider.ChatEvent) error) (provider.ChatResult, error) {
		parseT.Fatal("StreamChat should not be called for a missing authenticated user")
		return provider.ChatResult{}, nil
	}
	parseFake.generateTitle = func(_ context.Context, _ provider.TitleRequest) (string, error) {
		return "", nil
	}
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-missing-user", 999999, "ghost@example.com")
	parseStream := &fakeChatSendStream{ctx: parseCtx}

	parseErr := parseServer.ParseSend(&chatpb.SendRequest{Message: "hello"}, parseStream)
	if status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected unauthenticated, got %v", status.Code(parseErr))
	}
}

func TestSynthesizeSpeechStreamsChunks(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "speech@example.com")
	parseFake := parseNewFakeProvider()
	parseFake.synthesizeSpeech = func(_ context.Context, parseReq provider.SpeechRequest, parseEmit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
		if parseErr := parseEmit(provider.SpeechChunk{AudioChunk: []byte("abc"), MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: parseReq.Text}); parseErr != nil {
			return provider.SpeechResult{}, parseErr
		}
		if parseErr2 := parseEmit(provider.SpeechChunk{Done: true, MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: parseReq.Text}); parseErr2 != nil {
			return provider.SpeechResult{}, parseErr2
		}
		return provider.SpeechResult{MimeType: "audio/mpeg", Model: modelGPT54Mini, Voice: "sage", Script: parseReq.Text}, nil
	}
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-speech", parseUser.ParseID, parseUser.Email)
	parseStream := &fakeSpeechStream{ctx: parseCtx}

	parseErr3 := parseServer.ParseSynthesizeSpeech(&chatpb.SynthesizeSpeechRequest{Text: "Hello [site](https://example.com) ```code```", Model: modelGPT54Mini}, parseStream)
	if parseErr3 != nil {
		parseT.Fatalf("SynthesizeSpeech: %v", parseErr3)
	}
	if len(parseStream.chunks) != 2 {
		parseT.Fatalf("expected two speech chunks, got %d", len(parseStream.chunks))
	}
	if parseStream.chunks[0].GetScript() != "Hello site" {
		parseT.Fatalf("unexpected sanitized script: %q", parseStream.chunks[0].GetScript())
	}
	if !parseStream.chunks[1].GetDone() {
		parseT.Fatalf("expected terminal speech chunk, got %+v", parseStream.chunks[1])
	}
}

func TestHTTPIntegrationProtectsAndServesChatShell(parseT *testing.T) {
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/chat-bootstrap.js", parseServeChatBootstrapJS)
	parseMux.Handle("/", http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path == "/" {
			parseServeChatShell(parseW, parseR)
			return
		}
		http.NotFound(parseW, parseR)
	}))

	parseTestServer := httptest.NewServer(parseMux)
	defer parseTestServer.Close()

	parseJar, parseErr := cookiejar.New(nil)
	if parseErr != nil {
		parseT.Fatalf("cookiejar.New: %v", parseErr)
	}
	parseClient := &http.Client{
		Jar: parseJar,
		CheckRedirect: func(parseReq *http.Request, parseVia []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	parseResp, parseErr := parseClient.Get(parseTestServer.URL + "/")
	if parseErr != nil {
		parseT.Fatalf("GET shell: %v", parseErr)
	}
	parseBody, parseErr := io.ReadAll(parseResp.Body)
	if parseErr != nil {
		parseT.Fatalf("ReadAll shell body: %v", parseErr)
	}
	parseShellBody := string(parseBody)
	if !strings.Contains(parseShellBody, `id="boot-shell"`) || !strings.Contains(parseShellBody, "chat-bootstrap.js") {
		parseT.Fatal("expected shell response to include boot shell content")
	}

	parseBootstrapResp, parseErr := parseClient.Get(parseTestServer.URL + "/chat-bootstrap.js")
	if parseErr != nil {
		parseT.Fatalf("GET bootstrap: %v", parseErr)
	}
	parseBootstrapBody, parseErr := io.ReadAll(parseBootstrapResp.Body)
	if parseErr != nil {
		parseT.Fatalf("ReadAll bootstrap body: %v", parseErr)
	}
	if !strings.Contains(string(parseBootstrapBody), "loadChatWasm") {
		parseT.Fatal("expected bootstrap JS endpoint to return chat bootstrap source")
	}
}
