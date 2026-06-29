package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminChatMutationAction string

const (
	parseAdminChatMutationSetSystemDefaultPrompt parseAdminChatMutationAction = "chat.system_default_prompt.set"
	parseAdminChatMutationSetModelDefault        parseAdminChatMutationAction = "chat.model_default.set"
	parseAdminChatMutationSetMemoryRule          parseAdminChatMutationAction = "chat.memory_rule.set"
	parseAdminChatMutationSetOnboardingTemplate  parseAdminChatMutationAction = "chat.onboarding_template.set"
	parseAdminChatMutationPublishWorkflow        parseAdminChatMutationAction = "chat.workflow.publish"
	parseAdminChatMutationPublishSkill           parseAdminChatMutationAction = "chat.skill.publish"
)

type parseAdminChatMutationTarget struct {
	workspaceID int64
}

// parseAuthorizeAdminChatMutationScope enforces role/scope boundaries for one chat settings mutation action.
func (parseS *chatServer) parseAuthorizeAdminChatMutationScope(parseCtx context.Context, parseAction parseAdminChatMutationAction, parseTarget parseAdminChatMutationTarget) (parseAdminAccessScope, error) {
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	switch parseAction {
	case parseAdminChatMutationSetSystemDefaultPrompt, parseAdminChatMutationSetModelDefault, parseAdminChatMutationSetMemoryRule:
		return parseAuthorizeAdminPlatformChatMutationScope(parseScope, parseAction)
	case parseAdminChatMutationSetOnboardingTemplate, parseAdminChatMutationPublishWorkflow, parseAdminChatMutationPublishSkill:
		return parseAuthorizeAdminWorkspaceChatMutationScope(parseScope, parseAction, parseTarget.workspaceID)
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported chat mutation action: %s", strings.TrimSpace(string(parseAction)))
	}
}

// parseAuthorizeAdminPlatformChatMutationScope restricts platform-wide chat defaults to superuser scope.
func parseAuthorizeAdminPlatformChatMutationScope(parseScope parseAdminAccessScope, parseAction parseAdminChatMutationAction) (parseAdminAccessScope, error) {
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "%s requires superuser role", strings.TrimSpace(string(parseAction)))
}

// parseAuthorizeAdminWorkspaceChatMutationScope gates workspace-scoped chat content mutations by workspace-admin scope membership.
func parseAuthorizeAdminWorkspaceChatMutationScope(parseScope parseAdminAccessScope, parseAction parseAdminChatMutationAction, parseWorkspaceID int64) (parseAdminAccessScope, error) {
	if parseWorkspaceID <= 0 {
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "%s requires workspace id", strings.TrimSpace(string(parseAction)))
	}
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	if _, hasParseWorkspace := parseScope.workspaceIDs[parseWorkspaceID]; !hasParseWorkspace {
		return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "%s target workspace outside scope", strings.TrimSpace(string(parseAction)))
	}
	return parseScope, nil
}
