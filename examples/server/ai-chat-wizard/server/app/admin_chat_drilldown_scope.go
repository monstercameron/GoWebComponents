package app

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminChatDrilldownVisibility struct {
	isTranscriptRedacted    bool
	canViewToolTracePayload bool
}

// parseAuthorizeAdminChatDrilldownScope enforces chat drill-down scope and resolves transcript/tool-trace visibility rules per caller role.
func (parseS *chatServer) parseAuthorizeAdminChatDrilldownScope(parseCtx context.Context, parseTargetUserID int64) (parseAdminAccessScope, parseAdminChatDrilldownVisibility, error) {
	if parseTargetUserID <= 0 {
		return parseAdminAccessScope{}, parseAdminChatDrilldownVisibility{}, status.Error(codes.InvalidArgument, "target user id is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.chats.drilldown")
	if parseErr != nil {
		return parseAdminAccessScope{}, parseAdminChatDrilldownVisibility{}, parseErr
	}
	if parseScope.isPlatformScope {
		return parseScope, parseAdminChatDrilldownVisibility{
			isTranscriptRedacted:    false,
			canViewToolTracePayload: true,
		}, nil
	}
	if _, hasParseUser := parseScope.userIDs[parseTargetUserID]; !hasParseUser {
		return parseAdminAccessScope{}, parseAdminChatDrilldownVisibility{}, status.Error(codes.PermissionDenied, "target user outside workspace-admin scope")
	}
	return parseScope, parseAdminChatDrilldownVisibility{
		isTranscriptRedacted:    true,
		canViewToolTracePayload: false,
	}, nil
}
