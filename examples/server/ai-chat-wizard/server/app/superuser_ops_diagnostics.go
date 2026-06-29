package app

import (
	"context"
	"log/slog"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseClampSuperuserOpsExecutionLimit bounds one diagnostics execution limit to a safe range.
func parseClampSuperuserOpsExecutionLimit(parseRequested int32) int64 {
	if parseRequested <= 0 {
		return 25
	}
	if parseRequested > 200 {
		return 200
	}
	return int64(parseRequested)
}

// parseHasServerToolExecutionAuditEvent reports whether one audit event type belongs to server-tool execution outcomes.
func parseHasServerToolExecutionAuditEvent(parseEventType string) bool {
	parseEventType = strings.TrimSpace(strings.ToLower(parseEventType))
	return strings.HasPrefix(parseEventType, "admin.superuser.server_tool.run.")
}

// parseBuildAuditLogEntry maps one audit-log row into protobuf form.
func parseBuildAuditLogEntry(parseRow parseAuditLogRow) *chatpb.AuditLogEntry {
	return &chatpb.AuditLogEntry{
		Id:          parseRow.ID,
		ActorUserId: parseRow.ActorUserID,
		WorkspaceId: parseRow.WorkspaceID,
		EventType:   parseRow.EventType,
		TargetType:  parseRow.TargetType,
		TargetId:    parseRow.TargetID,
		Summary:     parseRow.Summary,
		PayloadJson: parseRow.PayloadJSON,
		CreatedAt:   parseRow.CreatedAt,
	}
}

// parseBuildServerToolExecutionAuditEntries filters recent audit rows down to server-tool execution outcomes.
func parseBuildServerToolExecutionAuditEntries(parseRows []parseAuditLogRow, parseLimit int64) []*chatpb.AuditLogEntry {
	parseEntries := make([]*chatpb.AuditLogEntry, 0, parseLimit)
	for _, parseRow := range parseRows {
		if !parseHasServerToolExecutionAuditEvent(parseRow.EventType) {
			continue
		}
		parseEntries = append(parseEntries, parseBuildAuditLogEntry(parseRow))
		if int64(len(parseEntries)) >= parseLimit {
			break
		}
	}
	return parseEntries
}

// GetSuperuserOpsDiagnostics bundles log-tail rows, server-tool policy, and recent server-tool execution outcomes.
func (parseS *chatServer) GetSuperuserOpsDiagnostics(parseCtx context.Context, parseReq *chatpb.GetSuperuserOpsDiagnosticsRequest) (*chatpb.GetSuperuserOpsDiagnosticsResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseReq == nil {
		parseReq = &chatpb.GetSuperuserOpsDiagnosticsRequest{}
	}

	parseLogTailResp, parseErr := parseS.GetLogTail(parseCtx, &chatpb.GetLogTailRequest{
		Source:   strings.TrimSpace(parseReq.GetLogSource()),
		MaxLines: parseReq.GetLogMaxLines(),
	})
	if parseErr != nil {
		return nil, parseErr
	}
	parsePolicyResp, parseErr := parseResolveServerToolPolicy(parseS.store)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "resolve server tool policy: %v", parseErr)
	}

	parseExecutionLimit := parseClampSuperuserOpsExecutionLimit(parseReq.GetExecutionLimit())
	parseExecutionRows := make([]*chatpb.AuditLogEntry, 0)
	if parseS.store != nil {
		parseAuditRows, parseErr2 := parseS.store.parseListAuditLogs(parseExecutionLimit * 8)
		if parseErr2 != nil {
			return nil, status.Errorf(codes.Internal, "list audit logs: %v", parseErr2)
		}
		parseExecutionRows = parseBuildServerToolExecutionAuditEntries(parseAuditRows, parseExecutionLimit)
	}

	parseS.logger.Info(
		"rpc.GetSuperuserOpsDiagnostics: complete",
		slog.Int64("superuser_user_id", parseSuperuserUserID),
		slog.Int("log_tail_lines", len(parseLogTailResp.GetEntries())),
		slog.Int("execution_rows", len(parseExecutionRows)),
		slog.String("policy_source", parsePolicyResp.GetSource()),
	)
	return &chatpb.GetSuperuserOpsDiagnosticsResponse{
		Diagnostics: &chatpb.SuperuserOpsDiagnosticsSlice{
			LogTail:                    parseLogTailResp.GetEntries(),
			ServerToolPolicy:           parsePolicyResp,
			RecentServerToolExecutions: parseExecutionRows,
		},
	}, nil
}
