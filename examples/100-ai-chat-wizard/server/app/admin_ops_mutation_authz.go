package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminOpsMutationAction string

const (
	parseAdminOpsMutationSetSiteConfig         parseAdminOpsMutationAction = "ops.site_config.set"
	parseAdminOpsMutationSetFeatureFlag        parseAdminOpsMutationAction = "ops.feature_flag.set"
	parseAdminOpsMutationSetRetentionPolicy    parseAdminOpsMutationAction = "ops.retention_policy.set"
	parseAdminOpsMutationSetWebhookBehavior    parseAdminOpsMutationAction = "ops.webhook_behavior.set"
	parseAdminOpsMutationSetIntegrationSetting parseAdminOpsMutationAction = "ops.integration_setting.set"
)

// parseAuthorizeAdminOpsMutationScope enforces one consistent superuser gate for ops settings mutation actions.
func (parseS *chatServer) parseAuthorizeAdminOpsMutationScope(parseCtx context.Context, parseAction parseAdminOpsMutationAction) (parseAdminAccessScope, error) {
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	switch parseAction {
	case parseAdminOpsMutationSetSiteConfig,
		parseAdminOpsMutationSetFeatureFlag,
		parseAdminOpsMutationSetRetentionPolicy,
		parseAdminOpsMutationSetWebhookBehavior,
		parseAdminOpsMutationSetIntegrationSetting:
		if parseScope.isPlatformScope {
			return parseScope, nil
		}
		return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "%s requires superuser role", strings.TrimSpace(string(parseAction)))
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported ops mutation action: %s", strings.TrimSpace(string(parseAction)))
	}
}

