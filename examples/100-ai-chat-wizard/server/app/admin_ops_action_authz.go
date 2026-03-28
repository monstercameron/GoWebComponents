package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminOpsAction string

const (
	parseAdminOpsActionIncidentUpdate    parseAdminOpsAction = "ops.incident.update"
	parseAdminOpsActionBackgroundJobRetry parseAdminOpsAction = "ops.background_job.retry"
	parseAdminOpsActionNotificationRetry parseAdminOpsAction = "ops.notification.retry"
	parseAdminOpsActionWebhookReplay     parseAdminOpsAction = "ops.webhook.replay"
)

// parseAuthorizeAdminOpsActionScope enforces one fail-closed authz contract for ops retries/replays and incident updates.
func (parseS *chatServer) parseAuthorizeAdminOpsActionScope(parseCtx context.Context, parseAction parseAdminOpsAction, parseWorkspaceID int64) (parseAdminAccessScope, error) {
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	switch parseAction {
	case parseAdminOpsActionIncidentUpdate:
		if parseScope.isPlatformScope {
			return parseScope, nil
		}
		if parseWorkspaceID <= 0 {
			return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "%s requires workspace id for workspace-admin scope", strings.TrimSpace(string(parseAction)))
		}
		if _, hasParseWorkspace := parseScope.workspaceIDs[parseWorkspaceID]; !hasParseWorkspace {
			return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "%s workspace outside scope", strings.TrimSpace(string(parseAction)))
		}
		return parseScope, nil
	case parseAdminOpsActionBackgroundJobRetry, parseAdminOpsActionNotificationRetry, parseAdminOpsActionWebhookReplay:
		if parseScope.isPlatformScope {
			return parseScope, nil
		}
		return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "%s requires superuser role", strings.TrimSpace(string(parseAction)))
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported ops action: %s", strings.TrimSpace(string(parseAction)))
	}
}

// parseTrackAdminOpsActionAudit emits one typed audit event record for one authorized ops action.
func (parseS *chatServer) parseTrackAdminOpsActionAudit(parseScope parseAdminAccessScope, parseAction parseAdminOpsAction, parseTargetType string, parseTargetID string, parseSummary string, parseWorkspaceID int64) {
	if parseS == nil {
		return
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		parseResolveAdminOpsActionAuditEventType(parseAction),
		strings.TrimSpace(parseTargetType),
		strings.TrimSpace(parseTargetID),
		strings.TrimSpace(parseSummary),
		"{}",
		parseWorkspaceID,
	)
}

// parseResolveAdminOpsActionAuditEventType resolves one stable audit event key per ops action.
func parseResolveAdminOpsActionAuditEventType(parseAction parseAdminOpsAction) string {
	switch parseAction {
	case parseAdminOpsActionIncidentUpdate:
		return "admin.ops.incident.update"
	case parseAdminOpsActionBackgroundJobRetry:
		return "admin.ops.background_job.retry"
	case parseAdminOpsActionNotificationRetry:
		return "admin.ops.notification.retry"
	case parseAdminOpsActionWebhookReplay:
		return "admin.ops.webhook.replay"
	default:
		return "admin.ops.unknown"
	}
}

