package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminCustomerTimelineSource string

const (
	parseAdminCustomerTimelineSourceChatContent  parseAdminCustomerTimelineSource = "chat_content"
	parseAdminCustomerTimelineSourceBilling      parseAdminCustomerTimelineSource = "billing_record"
	parseAdminCustomerTimelineSourceSupport      parseAdminCustomerTimelineSource = "support_note"
	parseAdminCustomerTimelineSourceAuthSession  parseAdminCustomerTimelineSource = "auth_session"
	parseAdminCustomerTimelineSourceAuditEvent   parseAdminCustomerTimelineSource = "audit_event"
)

// parseAuthorizeAdminCustomerTimelineScope enforces caller scope for one customer timeline source and returns whether payload redaction is required.
func (parseS *chatServer) parseAuthorizeAdminCustomerTimelineScope(parseCtx context.Context, parseTargetUserID int64, parseSource parseAdminCustomerTimelineSource) (parseAdminAccessScope, bool, error) {
	if parseTargetUserID <= 0 {
		return parseAdminAccessScope{}, false, status.Error(codes.InvalidArgument, "target user id is required")
	}
	if !parseIsAdminCustomerTimelineSourceSupported(parseSource) {
		return parseAdminAccessScope{}, false, status.Errorf(codes.InvalidArgument, "unsupported customer timeline source: %s", strings.TrimSpace(string(parseSource)))
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.customers.timeline")
	if parseErr != nil {
		return parseAdminAccessScope{}, false, parseErr
	}
	if parseScope.isPlatformScope {
		return parseScope, false, nil
	}
	if _, hasParseUser := parseScope.userIDs[parseTargetUserID]; !hasParseUser {
		return parseAdminAccessScope{}, false, status.Error(codes.PermissionDenied, "target user outside workspace-admin scope")
	}
	return parseScope, true, nil
}

// parseIsAdminCustomerTimelineSourceSupported reports whether one customer timeline source key is recognized.
func parseIsAdminCustomerTimelineSourceSupported(parseSource parseAdminCustomerTimelineSource) bool {
	switch parseSource {
	case parseAdminCustomerTimelineSourceChatContent,
		parseAdminCustomerTimelineSourceBilling,
		parseAdminCustomerTimelineSourceSupport,
		parseAdminCustomerTimelineSourceAuthSession,
		parseAdminCustomerTimelineSourceAuditEvent:
		return true
	default:
		return false
	}
}

