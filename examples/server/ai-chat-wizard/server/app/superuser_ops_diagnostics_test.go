package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetSuperuserOpsDiagnostics returns one typed diagnostics slice for superuser ops workflows.
func TestGetSuperuserOpsDiagnostics(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSuperuser := parseMustCreateUser(parseT, parseStore, "ops-diagnostics-superuser@example.com")
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseSuperuser.ID, "ws-ops-diagnostics")

	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseSuperuser.ID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "admin.superuser.server_tool_policy.set",
		TargetType:  "server_tool_policy",
		TargetID:    "global",
		Summary:     "policy updated",
		PayloadJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog policy row: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseSuperuser.ID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "admin.superuser.server_tool.run.completed",
		TargetType:  "server_tool",
		TargetID:    "session-ops-diagnostics",
		Summary:     "run completed",
		PayloadJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog run row: %v", parseErr)
	}

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseCtx := parseBindAuthUser(parseServer, "peer-ops-diagnostics-superuser", parseSuperuser.ID, parseSuperuser.Email)
	parseResp, parseErr := parseServer.GetSuperuserOpsDiagnostics(parseCtx, &chatpb.GetSuperuserOpsDiagnosticsRequest{
		LogSource:      "server",
		LogMaxLines:    5,
		ExecutionLimit: 5,
	})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserOpsDiagnostics: %v", parseErr)
	}
	if parseResp.GetDiagnostics() == nil {
		parseT.Fatalf("expected diagnostics payload, got %+v", parseResp)
	}
	if parseResp.GetDiagnostics().GetServerToolPolicy() == nil {
		parseT.Fatalf("expected server tool policy payload, got %+v", parseResp.GetDiagnostics())
	}
	parseExecutionRows := parseResp.GetDiagnostics().GetRecentServerToolExecutions()
	if len(parseExecutionRows) != 1 {
		parseT.Fatalf("expected one filtered server-tool execution row, got %+v", parseExecutionRows)
	}
	if parseExecutionRows[0].GetEventType() != "admin.superuser.server_tool.run.completed" {
		parseT.Fatalf("unexpected server-tool execution event type: %+v", parseExecutionRows[0])
	}
}

// TestGetSuperuserOpsDiagnosticsRequiresSURole verifies diagnostics slices deny non-superuser callers.
func TestGetSuperuserOpsDiagnosticsRequiresSURole(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "ops-diagnostics-user@example.com")
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseCtx := parseBindAuthUser(parseServer, "peer-ops-diagnostics-user", parseUser.ID, parseUser.Email)
	if _, parseErr := parseServer.GetSuperuserOpsDiagnostics(parseCtx, &chatpb.GetSuperuserOpsDiagnosticsRequest{}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetSuperuserOpsDiagnostics status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
