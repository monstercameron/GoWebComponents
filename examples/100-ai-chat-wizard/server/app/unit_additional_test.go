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

func TestLoadStoreQueriesAndBestEffortStatements(parseT *testing.T) {
	parseQueries, parseErr := parseLoadStoreQueries()
	if parseErr != nil {
		parseT.Fatalf("loadStoreQueries: %v", parseErr)
	}
	if parseQueries.schema == "" || parseQueries.parseCreateUser == "" || parseQueries.parseUpsertUserMemory == "" {
		parseT.Fatalf("expected key SQL queries to be loaded, got %+v", parseQueries)
	}

	parseDb, parseErr := sql.Open("sqlite3", "file:unit-test-best-effort?mode=memory&cache=shared")
	if parseErr != nil {
		parseT.Fatalf("sql.Open: %v", parseErr)
	}
	defer parseDb.Close()

	parseRunBestEffortStatements(parseDb, "CREATE TABLE example(value TEXT); invalid sql; INSERT INTO example(value) VALUES ('ok')")
	var parseCount int
	if parseErr2 := parseDb.QueryRow("SELECT COUNT(*) FROM example").Scan(&parseCount); parseErr2 != nil {
		parseT.Fatalf("QueryRow count: %v", parseErr2)
	}
	if parseCount != 1 {
		parseT.Fatalf("expected best-effort statements to continue after invalid SQL, got count=%d", parseCount)
	}
}

func TestNewChatServiceServerFallbackModel(parseT *testing.T) {
	parseServer := parseNewChatServiceServer("test-key", "", "", "missing-model", nil, parseNewTestLogger())
	if parseServer == nil {
		parseT.Fatal("expected newChatServiceServer to return a server")
	}
	if parseServer.defaultModel != modelGPT54Mini {
		parseT.Fatalf("expected fallback default model %q, got %q", modelGPT54Mini, parseServer.defaultModel)
	}
	if parseServer.providerRegistry == nil {
		parseT.Fatal("expected provider registry to be initialized")
	}
	if cap(parseServer.memoryExtractionSlots) != 2 {
		parseT.Fatalf("expected memory extraction slot capacity 2, got %d", cap(parseServer.memoryExtractionSlots))
	}

	parseNormalizedOnly := parseNewChatServiceServer("", "", "", "gpt-5.4-2026-03-17", nil, parseNewTestLogger())
	if parseNormalizedOnly.defaultModel != modelGPT54 {
		parseT.Fatalf("expected normalized default model %q, got %q", modelGPT54, parseNormalizedOnly.defaultModel)
	}
}

func TestMemoryRPCValidationAndErrorBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "memory-rpc-errors@example.com")
	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		store:        store,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-memory-rpc-errors", parseUser.ParseID, parseUser.Email)

	if _, parseErr := parseServer.ParseUpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for missing memory, got %v", status.Code(parseErr))
	}
	if _, parseErr2 := parseServer.ParseUpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{Memory: &chatpb.UserMemory{Summary: "   "}}); status.Code(parseErr2) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for blank memory summary, got %v", status.Code(parseErr2))
	}
	if _, parseErr3 := parseServer.ParseDeleteUserMemory(parseCtx, &chatpb.DeleteUserMemoryRequest{Key: "   "}); status.Code(parseErr3) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for blank memory key, got %v", status.Code(parseErr3))
	}

	store.parseClose()

	if _, parseErr4 := parseServer.ParseListUserMemories(parseCtx, &chatpb.ListUserMemoriesRequest{}); status.Code(parseErr4) != codes.Internal {
		parseT.Fatalf("expected internal error for closed-store ListUserMemories, got %v", status.Code(parseErr4))
	}
	if _, parseErr5 := parseServer.ParseUpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{Memory: &chatpb.UserMemory{Summary: "Stored summary"}}); status.Code(parseErr5) != codes.Internal {
		parseT.Fatalf("expected internal error for closed-store UpsertUserMemory, got %v", status.Code(parseErr5))
	}
	if _, parseErr6 := parseServer.ParseDeleteUserMemory(parseCtx, &chatpb.DeleteUserMemoryRequest{Key: "memory-key"}); status.Code(parseErr6) != codes.Internal {
		parseT.Fatalf("expected internal error for closed-store DeleteUserMemory, got %v", status.Code(parseErr6))
	}
	if _, parseErr7 := parseServer.SetCustomSystemPrompt(parseCtx, wrapperspb.String("Prompt")); status.Code(parseErr7) != codes.Internal {
		parseT.Fatalf("expected internal error for closed-store SetCustomSystemPrompt, got %v", status.Code(parseErr7))
	}
	if _, parseErr8 := parseServer.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{}); status.Code(parseErr8) != codes.Internal {
		parseT.Fatalf("expected internal error for closed-store GetCustomSystemPrompt, got %v", status.Code(parseErr8))
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-memory-rpc-errors")
}

func TestMemoryRPCFallbacksWithoutStore(parseT *testing.T) {
	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-memory-fallbacks", 7, "memory-fallbacks@example.com")

	parseListResp, parseErr := parseServer.ParseListUserMemories(parseCtx, &chatpb.ListUserMemoriesRequest{})
	if parseErr != nil || len(parseListResp.GetMemories()) != 0 {
		parseT.Fatalf("expected empty memory fallback response, resp=%+v err=%v", parseListResp, parseErr)
	}
	if _, parseErr2 := parseServer.ParseUpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{}); parseErr2 != nil {
		parseT.Fatalf("expected no-store UpsertUserMemory to no-op, got %v", parseErr2)
	}
	if _, parseErr3 := parseServer.ParseDeleteUserMemory(parseCtx, &chatpb.DeleteUserMemoryRequest{}); parseErr3 != nil {
		parseT.Fatalf("expected no-store DeleteUserMemory to no-op, got %v", parseErr3)
	}
	parseCustomPrompt, parseErr := parseServer.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{})
	if parseErr != nil || parseCustomPrompt.GetValue() != "" {
		parseT.Fatalf("expected empty custom prompt fallback, resp=%+v err=%v", parseCustomPrompt, parseErr)
	}
	if _, parseErr4 := parseServer.SetCustomSystemPrompt(parseCtx, wrapperspb.String("ignored")); parseErr4 != nil {
		parseT.Fatalf("expected no-store SetCustomSystemPrompt to no-op, got %v", parseErr4)
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-memory-fallbacks")
	if _, parseErr5 := parseServer.ParseListUserMemories(context.Background(), &chatpb.ListUserMemoriesRequest{}); status.Code(parseErr5) != codes.Unauthenticated {
		parseT.Fatalf("expected unauthenticated ListUserMemories after unbind, got %v", status.Code(parseErr5))
	}
}
