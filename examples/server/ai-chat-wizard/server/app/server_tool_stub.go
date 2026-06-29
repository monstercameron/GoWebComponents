package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetServerToolPolicy returns the active server-tool policy snapshot for superusers.
func (parseS *chatServer) GetServerToolPolicy(parseCtx context.Context, parseReq *chatpb.GetServerToolPolicyRequest) (*chatpb.GetServerToolPolicyResponse, error) {
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
		return nil, parseErr
	}
	if parseReq == nil {
		parseReq = &chatpb.GetServerToolPolicyRequest{}
	}

	parsePolicy, parseErr := parseResolveServerToolPolicy(parseS.store)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "GetServerToolPolicy policy parse failed: %v", parseErr)
	}
	parseS.logger.Info(
		"rpc.GetServerToolPolicy: complete",
		slog.String("rpc", "GetServerToolPolicy"),
		slog.String("policy.source", parsePolicy.GetSource()),
		slog.Bool("policy.is_enabled", parsePolicy.GetIsEnabled()),
		slog.Int("policy.approved_tools", len(parsePolicy.GetApprovedTools())),
	)
	return parsePolicy, nil
}

// SetServerToolPolicy validates and applies one server-tool policy snapshot.
func (parseS *chatServer) SetServerToolPolicy(parseCtx context.Context, parseReq *chatpb.SetServerToolPolicyRequest) (*chatpb.SetServerToolPolicyResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "server_tool_policy.set")
	if parseErr != nil {
		return nil, parseErr
	}
	parseCurrentPolicy, parseErr := parseResolveServerToolPolicy(parseS.store)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "resolve current server tool policy: %v", parseErr)
	}
	if parseErr = parseValidateSetServerToolPolicyRequest(parseReq); parseErr != nil {
		parseS.parseTrackAdminAuditEvent(
			parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
			"admin.superuser.server_tool_policy.denied",
			"server_tool_policy",
			"global",
			parseErr.Error(),
			"{}",
			0,
		)
		return nil, parseErr
	}
	if parseHasServerToolPolicyDangerousChange(parseCurrentPolicy, parseReq) && !parseResolveServerToolDangerousChangeConfirmation(parseCtx) {
		parseErr = status.Error(codes.InvalidArgument, "dangerous server tool policy change requires explicit confirmation")
		parseS.parseTrackAdminAuditEvent(
			parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
			"admin.superuser.server_tool_policy.denied",
			"server_tool_policy",
			"global",
			parseErr.Error(),
			"{}",
			0,
		)
		return nil, parseErr
	}
	parseAppliedPolicy, parseErr := parseStoreServerToolPolicy(parseS.store, parseSuperuserUserID, parseReq)
	if parseErr != nil {
		parseS.parseTrackAdminAuditEvent(
			parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
			"admin.superuser.server_tool_policy.denied",
			"server_tool_policy",
			"global",
			parseErr.Error(),
			"{}",
			0,
		)
		return nil, status.Errorf(codes.Internal, "store server tool policy: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.server_tool_policy.set",
		"server_tool_policy",
		"global",
		"Server tool policy updated",
		"{}",
		0,
	)
	return &chatpb.SetServerToolPolicyResponse{
		IsApplied:         true,
		UpdatedAt:         strings.TrimSpace(parseAppliedPolicy.GetUpdatedAt()),
		IsEnabled:         parseAppliedPolicy.GetIsEnabled(),
		MaxSessionSeconds: parseAppliedPolicy.GetMaxSessionSeconds(),
		MaxOutputBytes:    parseAppliedPolicy.GetMaxOutputBytes(),
		ApprovedTools:     parseAppliedPolicy.GetApprovedTools(),
		Source:            parseAppliedPolicy.GetSource(),
	}, nil
}

// RunServerTool executes one policy-authorized server tool session with bidirectional stream control frames.
func (parseS *chatServer) RunServerTool(parseStream grpc.BidiStreamingServer[chatpb.RunServerToolRequest, chatpb.RunServerToolEvent]) error {
	if parseStream == nil {
		return status.Error(codes.InvalidArgument, "RunServerTool stream is required")
	}
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseStream.Context(), "server_tool.run")
	if parseErr != nil {
		return parseErr
	}
	parseStartRequest, parseErr := parseStream.Recv()
	if parseErr != nil {
		return status.Error(codes.InvalidArgument, "run server tool start payload is required")
	}
	parseStart := parseStartRequest.GetStart()
	if parseStart == nil {
		return status.Error(codes.InvalidArgument, "run server tool start payload is required")
	}
	parsePolicy, parseErr := parseResolveServerToolPolicy(parseS.store)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "resolve server tool policy: %v", parseErr)
	}
	if parseErr = parseAuthorizeRunServerToolStart(parsePolicy, parseStart); parseErr != nil {
		parseS.parseTrackAdminAuditEvent(
			parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
			"admin.superuser.server_tool.run.denied",
			"server_tool",
			strings.TrimSpace(parseStart.GetSessionId()),
			parseErr.Error(),
			"{}",
			0,
		)
		parseEmitErr := parseStream.Send(&chatpb.RunServerToolEvent{
			Payload: &chatpb.RunServerToolEvent_Error{
				Error: &chatpb.RunServerToolError{
					SessionId: strings.TrimSpace(parseStart.GetSessionId()),
					Code:      "policy_denied",
					Message:   parseErr.Error(),
				},
			},
		})
		if parseEmitErr != nil {
			return status.Errorf(codes.PermissionDenied, "run server tool denied: %v (emit=%v)", parseErr, parseEmitErr)
		}
		return parseErr
	}
	parseSender := &parseServerToolEventSender{parseStream: parseStream}
	parseSession, parseErr := parseStartServerToolRuntimeSession(parseStream.Context(), parseSender, parseStart, parsePolicy)
	if parseErr != nil {
		parseS.parseTrackAdminAuditEvent(
			parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
			"admin.superuser.server_tool.run.denied",
			"server_tool",
			strings.TrimSpace(parseStart.GetSessionId()),
			parseErr.Error(),
			"{}",
			0,
		)
		_ = parseSender.parseSendErrorEvent(strings.TrimSpace(parseStart.GetSessionId()), "start_failed", parseErr.Error())
		return status.Errorf(codes.FailedPrecondition, "run server tool start failed: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.server_tool.run.started",
		"server_tool",
		parseSession.parseSessionID,
		"Server tool execution started",
		"{}",
		0,
	)
	parseRecvRequestCh := make(chan *chatpb.RunServerToolRequest, 1)
	parseRecvErrCh := make(chan error, 1)
	go func() {
		for {
			parseRequest, parseRecvErr := parseStream.Recv()
			if parseRecvErr != nil {
				parseRecvErrCh <- parseRecvErr
				close(parseRecvRequestCh)
				return
			}
			parseRecvRequestCh <- parseRequest
		}
	}()
	for {
		select {
		case parseRuntimeResult := <-parseSession.parseDone:
			if parseSendErr := parseSender.parseSendExitEvent(parseSession.parseSessionID, parseRuntimeResult); parseSendErr != nil {
				return parseSendErr
			}
			parseEventType := "admin.superuser.server_tool.run.completed"
			parseSummary := "Server tool execution completed"
			if parseRuntimeResult.ParseExitCode != 0 || strings.TrimSpace(parseRuntimeResult.ParseErrorText) != "" {
				parseEventType = "admin.superuser.server_tool.run.failed"
				parseSummary = strings.TrimSpace(parseRuntimeResult.ParseErrorText)
				if parseSummary == "" {
					parseSummary = "Server tool execution failed"
				}
			}
			parseS.parseTrackAdminAuditEvent(
				parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
				parseEventType,
				"server_tool",
				parseSession.parseSessionID,
				parseSummary,
				"{}",
				0,
			)
			return nil
		case parseRequest, parseOpen := <-parseRecvRequestCh:
			if !parseOpen {
				parseRecvRequestCh = nil
				continue
			}
			if parseRequest == nil {
				continue
			}
			switch parsePayload := parseRequest.Payload.(type) {
			case *chatpb.RunServerToolRequest_Stdin:
				if parsePayload.Stdin == nil {
					parseSession.parseEmitRuntimeError("invalid_request", "run server tool stdin payload is required")
					parseSession.parseCancel()
					continue
				}
				if parseStdinErr := parseSession.parseWriteInputChunk(parsePayload.Stdin.GetSessionId(), parsePayload.Stdin.GetChunk(), parsePayload.Stdin.GetClose()); parseStdinErr != nil {
					parseSession.parseEmitRuntimeError("stdin_failed", parseStdinErr.Error())
					parseSession.parseCancel()
				}
			case *chatpb.RunServerToolRequest_Signal:
				if parsePayload.Signal == nil {
					parseSession.parseEmitRuntimeError("invalid_request", "run server tool signal payload is required")
					parseSession.parseCancel()
					continue
				}
				if parseSignalErr := parseSession.parseSendSignal(parsePayload.Signal.GetSessionId(), parsePayload.Signal.GetSignal()); parseSignalErr != nil {
					parseSession.parseEmitRuntimeError("signal_failed", parseSignalErr.Error())
					parseSession.parseCancel()
				}
			case *chatpb.RunServerToolRequest_Close:
				if parsePayload.Close == nil {
					parseSession.parseEmitRuntimeError("invalid_request", "run server tool close payload is required")
					parseSession.parseCancel()
					continue
				}
				if parseCloseErr := parseSession.parseCloseSession(parsePayload.Close.GetSessionId()); parseCloseErr != nil {
					parseSession.parseEmitRuntimeError("close_failed", parseCloseErr.Error())
					parseSession.parseCancel()
				}
			default:
				parseSession.parseEmitRuntimeError("invalid_request", "run server tool payload type is unsupported")
				parseSession.parseCancel()
			}
		case parseRecvErr := <-parseRecvErrCh:
			if errors.Is(parseRecvErr, io.EOF) {
				parseRecvErrCh = nil
				continue
			}
			parseSession.parseEmitRuntimeError("stream_receive_failed", parseRecvErr.Error())
			parseSession.parseCancel()
			parseRecvErrCh = nil
		case <-parseStream.Context().Done():
			parseSession.parseEmitRuntimeError("stream_context_canceled", "run server tool stream context canceled")
			parseSession.parseCancel()
		}
	}
}
