package app

import (
	"context"
	"log/slog"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetServerToolPolicy returns a placeholder command-policy response while persistence wiring is pending.
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

// SetServerToolPolicy validates access and marks policy-write support as pending.
func (parseS *chatServer) SetServerToolPolicy(parseCtx context.Context, parseReq *chatpb.SetServerToolPolicyRequest) (*chatpb.SetServerToolPolicyResponse, error) {
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
		return nil, parseErr
	}
	if parseReq == nil {
		parseReq = &chatpb.SetServerToolPolicyRequest{}
	}
	return nil, status.Error(codes.Unimplemented, "SetServerToolPolicy stub: persistence not implemented yet")
}

// RunServerTool validates access and marks terminal session streaming as pending.
func (parseS *chatServer) RunServerTool(parseStream grpc.BidiStreamingServer[chatpb.RunServerToolRequest, chatpb.RunServerToolEvent]) error {
	if parseStream == nil {
		return status.Error(codes.InvalidArgument, "RunServerTool stream is required")
	}
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseStream.Context()); parseErr != nil {
		return parseErr
	}
	return status.Error(codes.Unimplemented, "RunServerTool stub: terminal execution not implemented yet")
}
