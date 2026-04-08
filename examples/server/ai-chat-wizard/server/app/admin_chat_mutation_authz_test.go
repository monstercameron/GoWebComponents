package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminChatMutationScopeBoundaries verifies chat settings mutation authz boundaries for superuser and workspace-admin callers.
func TestAdminChatMutationScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-chat-authz-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chat-authz")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-chat-authz-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseOtherWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-chat-authz")

	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseBobCtx, parseAdminChatMutationSetSystemDefaultPrompt, parseAdminChatMutationTarget{}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin system-default prompt status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseAliceCtx, parseAdminChatMutationSetSystemDefaultPrompt, parseAdminChatMutationTarget{}); parseErr != nil {
		parseT.Fatalf("superuser system-default prompt expected allow: %v", parseErr)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseBobCtx, parseAdminChatMutationSetOnboardingTemplate, parseAdminChatMutationTarget{
		workspaceID: parseBobWorkspaceID,
	}); parseErr != nil {
		parseT.Fatalf("workspace-admin onboarding template expected in-scope allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseBobCtx, parseAdminChatMutationPublishWorkflow, parseAdminChatMutationTarget{
		workspaceID: parseOtherWorkspaceID,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin workflow publish out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseBobCtx, parseAdminChatMutationPublishSkill, parseAdminChatMutationTarget{}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("workspace-admin skill publish missing workspace id status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseAliceCtx, parseAdminChatMutationPublishWorkflow, parseAdminChatMutationTarget{
		workspaceID: parseOtherWorkspaceID,
	}); parseErr != nil {
		parseT.Fatalf("superuser workflow publish expected allow: %v", parseErr)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminChatMutationScope(parseAliceCtx, parseAdminChatMutationAction("chat.unknown"), parseAdminChatMutationTarget{}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported chat mutation status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

