package app

import (
	"context"
	"database/sql"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestLoadStoreQueriesAndBestEffortStatements(t *testing.T) {
	queries, err := loadStoreQueries()
	if err != nil {
		t.Fatalf("loadStoreQueries: %v", err)
	}
	if queries.schema == "" || queries.createUser == "" || queries.upsertUserMemory == "" {
		t.Fatalf("expected key SQL queries to be loaded, got %+v", queries)
	}

	db, err := sql.Open("sqlite3", "file:unit-test-best-effort?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	runBestEffortStatements(db, "CREATE TABLE example(value TEXT); invalid sql; INSERT INTO example(value) VALUES ('ok')")
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM example").Scan(&count); err != nil {
		t.Fatalf("QueryRow count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected best-effort statements to continue after invalid SQL, got count=%d", count)
	}
}

func TestNewChatServiceServerFallbackModel(t *testing.T) {
	server := newChatServiceServer("test-key", "", "", "missing-model", nil, newTestLogger())
	if server == nil {
		t.Fatal("expected newChatServiceServer to return a server")
	}
	if server.defaultModel != modelGPT54Mini {
		t.Fatalf("expected fallback default model %q, got %q", modelGPT54Mini, server.defaultModel)
	}
	if server.providerRegistry == nil {
		t.Fatal("expected provider registry to be initialized")
	}
	if cap(server.memoryExtractionSlots) != 2 {
		t.Fatalf("expected memory extraction slot capacity 2, got %d", cap(server.memoryExtractionSlots))
	}

	normalizedOnly := newChatServiceServer("", "", "", "gpt-5.4-2026-03-17", nil, newTestLogger())
	if normalizedOnly.defaultModel != modelGPT54 {
		t.Fatalf("expected normalized default model %q, got %q", modelGPT54, normalizedOnly.defaultModel)
	}
}

func TestMemoryRPCValidationAndErrorBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "memory-rpc-errors@example.com")
	server := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        store,
		logger:       newTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	ctx := bindAuthUser(server, "peer-memory-rpc-errors", user.ID, user.Email)

	if _, err := server.UpsertUserMemory(ctx, &chatpb.UpsertUserMemoryRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for missing memory, got %v", status.Code(err))
	}
	if _, err := server.UpsertUserMemory(ctx, &chatpb.UpsertUserMemoryRequest{Memory: &chatpb.UserMemory{Summary: "   "}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for blank memory summary, got %v", status.Code(err))
	}
	if _, err := server.DeleteUserMemory(ctx, &chatpb.DeleteUserMemoryRequest{Key: "   "}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for blank memory key, got %v", status.Code(err))
	}

	store.close()

	if _, err := server.ListUserMemories(ctx, &chatpb.ListUserMemoriesRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for closed-store ListUserMemories, got %v", status.Code(err))
	}
	if _, err := server.UpsertUserMemory(ctx, &chatpb.UpsertUserMemoryRequest{Memory: &chatpb.UserMemory{Summary: "Stored summary"}}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for closed-store UpsertUserMemory, got %v", status.Code(err))
	}
	if _, err := server.DeleteUserMemory(ctx, &chatpb.DeleteUserMemoryRequest{Key: "memory-key"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for closed-store DeleteUserMemory, got %v", status.Code(err))
	}
	if _, err := server.SetCustomSystemPrompt(ctx, wrapperspb.String("Prompt")); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for closed-store SetCustomSystemPrompt, got %v", status.Code(err))
	}
	if _, err := server.GetCustomSystemPrompt(ctx, &emptypb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for closed-store GetCustomSystemPrompt, got %v", status.Code(err))
	}

	server.unbindAuthenticatedPeer("peer-memory-rpc-errors")
}

func TestMemoryRPCFallbacksWithoutStore(t *testing.T) {
	server := &chatServer{
		defaultModel: modelGPT54Mini,
		logger:       newTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	ctx := bindAuthUser(server, "peer-memory-fallbacks", 7, "memory-fallbacks@example.com")

	listResp, err := server.ListUserMemories(ctx, &chatpb.ListUserMemoriesRequest{})
	if err != nil || len(listResp.GetMemories()) != 0 {
		t.Fatalf("expected empty memory fallback response, resp=%+v err=%v", listResp, err)
	}
	if _, err := server.UpsertUserMemory(ctx, &chatpb.UpsertUserMemoryRequest{}); err != nil {
		t.Fatalf("expected no-store UpsertUserMemory to no-op, got %v", err)
	}
	if _, err := server.DeleteUserMemory(ctx, &chatpb.DeleteUserMemoryRequest{}); err != nil {
		t.Fatalf("expected no-store DeleteUserMemory to no-op, got %v", err)
	}
	customPrompt, err := server.GetCustomSystemPrompt(ctx, &emptypb.Empty{})
	if err != nil || customPrompt.GetValue() != "" {
		t.Fatalf("expected empty custom prompt fallback, resp=%+v err=%v", customPrompt, err)
	}
	if _, err := server.SetCustomSystemPrompt(ctx, wrapperspb.String("ignored")); err != nil {
		t.Fatalf("expected no-store SetCustomSystemPrompt to no-op, got %v", err)
	}

	server.unbindAuthenticatedPeer("peer-memory-fallbacks")
	if _, err := server.ListUserMemories(context.Background(), &chatpb.ListUserMemoriesRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated ListUserMemories after unbind, got %v", status.Code(err))
	}
}
