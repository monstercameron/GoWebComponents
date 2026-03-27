package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetLogTailReturnsLatestServerLines verifies that server log tailing returns the newest entries.
func TestGetLogTailReturnsLatestServerLines(parseT *testing.T) {
	parseRestoreWorkingDir := parseChangeWorkingDirectory(parseT, parseT.TempDir())
	defer parseRestoreWorkingDir()

	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseUser := parseMustCreateUser(parseT, parseStore, "logs@example.com")
	parseCtx := parseBindAuthUser(parseServer, "peer-log-tail", parseUser.ID, parseUser.Email)

	parseServerLogPath := filepath.Join(serverLogDir, serverLogFilename)
	parseWriteLogLines(parseT, parseServerLogPath, []string{
		`{"timestamp":"2026-03-27T14:10:00Z","message":"server line 1"}`,
		`{"timestamp":"2026-03-27T14:10:01Z","message":"server line 2"}`,
		`{"timestamp":"2026-03-27T14:10:02Z","message":"server line 3"}`,
	})

	parseResp, parseErr := parseServer.GetLogTail(parseCtx, &chatpb.GetLogTailRequest{
		Source:   logTailSourceServer,
		MaxLines: 2,
	})
	if parseErr != nil {
		parseT.Fatalf("GetLogTail: %v", parseErr)
	}
	if parseResp.GetSource() != logTailSourceServer {
		parseT.Fatalf("source = %q, want %q", parseResp.GetSource(), logTailSourceServer)
	}
	if parseResp.GetReturnedLines() != 2 || len(parseResp.GetEntries()) != 2 {
		parseT.Fatalf("returned lines = %d/%d, want 2", parseResp.GetReturnedLines(), len(parseResp.GetEntries()))
	}
	if !strings.Contains(parseResp.GetEntries()[0].GetLine(), "server line 2") {
		parseT.Fatalf("first line = %q, want server line 2", parseResp.GetEntries()[0].GetLine())
	}
	if !strings.Contains(parseResp.GetEntries()[1].GetLine(), "server line 3") {
		parseT.Fatalf("second line = %q, want server line 3", parseResp.GetEntries()[1].GetLine())
	}
}

// TestGetLogTailSupportsAllSourcesAndFilter verifies multi-source tailing, filtering, and truncation behavior.
func TestGetLogTailSupportsAllSourcesAndFilter(parseT *testing.T) {
	parseRestoreWorkingDir := parseChangeWorkingDirectory(parseT, parseT.TempDir())
	defer parseRestoreWorkingDir()

	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseUser := parseMustCreateUser(parseT, parseStore, "logs-all@example.com")
	parseCtx := parseBindAuthUser(parseServer, "peer-log-tail-all", parseUser.ID, parseUser.Email)

	parseWriteLogLines(parseT, filepath.Join(serverLogDir, serverLogFilename), []string{
		`{"timestamp":"2026-03-27T14:10:00Z","message":"INFO startup"}`,
		`{"timestamp":"2026-03-27T14:10:01Z","message":"ERROR server timeout"}`,
	})
	parseWriteLogLines(parseT, filepath.Join(serverLogDir, clientLogFilename), []string{
		`{"timestamp":"2026-03-27T14:10:02Z","message":"WARN reconnecting"}`,
		`{"timestamp":"2026-03-27T14:10:03Z","message":"error client stream broken"}`,
		`{"timestamp":"2026-03-27T14:10:04Z","message":"error ui panic boundary"}`,
	})

	parseResp, parseErr := parseServer.GetLogTail(parseCtx, &chatpb.GetLogTailRequest{
		Source:   logTailSourceAll,
		MaxLines: 2,
		Contains: "error",
	})
	if parseErr != nil {
		parseT.Fatalf("GetLogTail: %v", parseErr)
	}
	if !parseResp.GetTruncated() {
		parseT.Fatalf("truncated = false, want true")
	}
	if len(parseResp.GetEntries()) != 2 {
		parseT.Fatalf("entries = %d, want 2", len(parseResp.GetEntries()))
	}
	for _, parseEntry := range parseResp.GetEntries() {
		if !strings.Contains(strings.ToLower(parseEntry.GetLine()), "error") {
			parseT.Fatalf("entry line = %q, want contains error", parseEntry.GetLine())
		}
		if parseEntry.GetSource() == "" {
			parseT.Fatalf("entry source should not be blank: %+v", parseEntry)
		}
	}
}

// TestGetLogTailRejectsInvalidSource verifies argument validation for source selectors.
func TestGetLogTailRejectsInvalidSource(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseUser := parseMustCreateUser(parseT, parseStore, "logs-invalid@example.com")
	parseCtx := parseBindAuthUser(parseServer, "peer-log-tail-invalid", parseUser.ID, parseUser.Email)

	_, parseErr := parseServer.GetLogTail(parseCtx, &chatpb.GetLogTailRequest{Source: "runtime"})
	if status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("status code = %v, want %v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestGetLogTailRequiresAuthentication verifies diagnostics log access is auth-gated.
func TestGetLogTailRequiresAuthentication(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	_, parseErr := parseServer.GetLogTail(context.Background(), &chatpb.GetLogTailRequest{})
	if status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("status code = %v, want %v", status.Code(parseErr), codes.Unauthenticated)
	}
}

// parseChangeWorkingDirectory switches into one directory for one test and returns a restore function.
func parseChangeWorkingDirectory(parseT *testing.T, parseNextWorkingDirectory string) func() {
	parseT.Helper()
	parseCurrentWorkingDirectory, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	if parseErr2 := os.Chdir(parseNextWorkingDirectory); parseErr2 != nil {
		parseT.Fatalf("Chdir(%q): %v", parseNextWorkingDirectory, parseErr2)
	}
	return func() {
		parseT.Helper()
		if parseErr3 := os.Chdir(parseCurrentWorkingDirectory); parseErr3 != nil {
			parseT.Fatalf("restore cwd: %v", parseErr3)
		}
	}
}

// parseWriteLogLines writes one newline-delimited log file with deterministic line order.
func parseWriteLogLines(parseT *testing.T, parseFilePath string, parseLines []string) {
	parseT.Helper()
	if parseErr := os.MkdirAll(filepath.Dir(parseFilePath), 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(%q): %v", filepath.Dir(parseFilePath), parseErr)
	}
	parseBody := strings.Join(parseLines, "\n") + "\n"
	if parseErr2 := os.WriteFile(parseFilePath, []byte(parseBody), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile(%q): %v", parseFilePath, parseErr2)
	}
}
