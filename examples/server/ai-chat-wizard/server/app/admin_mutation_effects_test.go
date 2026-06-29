package app

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const parseTestAuthPassword = "StrongPass123!"

// parseBuildTokenAuthContext builds one metadata-authenticated test context for one token.
func parseBuildTokenAuthContext(parseToken string) metadata.MD {
	parseAuthMD := metadata.Pairs(authMetadataKey, "Bearer "+parseToken)
	return parseAuthMD
}

// TestAdminMutationDisableAndRestoreEnforcesImmediateAuthEffects verifies user disable/restore applies auth-block and session-revocation side effects.
func TestAdminMutationDisableAndRestoreEnforcesImmediateAuthEffects(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("mutation-superuser@example.com", parseTestAuthPassword, "Mutation Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseTargetUser, parseErr := parseServer.authManager.parseSignup("mutation-target@example.com", parseTestAuthPassword, "Mutation Target")
	if parseErr != nil {
		parseT.Fatalf("parseSignup target: %v", parseErr)
	}
	parseTargetWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseTargetUser.ID, "ws-mutation-target")
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "mutation-target-api-key-1",
		WorkspaceID: parseTargetWorkspaceID,
		UserID:      parseTargetUser.ID,
		Label:       "Mutation Target Key",
		KeyPrefix:   "mtk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey target baseline: %v", parseErr)
	}

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-admin-mutation-superuser-exec"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)

	parseTargetToken, parseErr := parseServer.authManager.issueToken(parseTargetUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken target: %v", parseErr)
	}
	parseTargetCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-admin-mutation-target"),
		parseBuildTokenAuthContext(parseTargetToken),
	)
	if _, parseErr = parseServer.GetSession(parseTargetCtx, &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("GetSession baseline target: %v", parseErr)
	}

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "security incident response"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction disable user: %v", parseErr)
	}
	if _, parseErr = parseServer.GetSession(parseTargetCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetSession disabled target status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr = parseServer.authManager.parseLogin(parseTargetUser.Email, parseTestAuthPassword); !errors.Is(parseErr, errInvalidCredentials) {
		parseT.Fatalf("parseLogin disabled target error=%v want=%v", parseErr, errInvalidCredentials)
	}
	parseAPIKeyRows, parseErr := parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys after disable: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) == "" {
		parseT.Fatalf("expected target API key revoked on disable, rows=%+v", parseAPIKeyRows)
	}

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationRestoreUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "incident resolved"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction restore user: %v", parseErr)
	}
	if _, parseErr = parseServer.authManager.parseLogin(parseTargetUser.Email, parseTestAuthPassword); parseErr != nil {
		parseT.Fatalf("parseLogin restored target: %v", parseErr)
	}
	if _, parseErr = parseServer.GetSession(parseTargetCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetSession old target token after restore status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	parseAPIKeyRows, parseErr = parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys after restore: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) == "" {
		parseT.Fatalf("expected target API key to remain revoked after restore, rows=%+v", parseAPIKeyRows)
	}
	parseAuditRows, parseErr := parseStore.parseListAuditLogs(20)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs user mutation: %v", parseErr)
	}
	parseMutationAuditCount := 0
	for _, parseAuditRow := range parseAuditRows {
		if parseAuditRow.ActorUserID != parseSuperuser.ID {
			continue
		}
		if parseAuditRow.EventType == "admin.mutation.action" {
			parseMutationAuditCount++
		}
	}
	if parseMutationAuditCount < 2 {
		parseT.Fatalf("expected at least two user-mutation audit rows, got %d rows=%+v", parseMutationAuditCount, parseAuditRows)
	}
}

// TestAdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects verifies workspace suspend/restore applies member auth-block and session-revocation side effects.
func TestAdminMutationSuspendAndRestoreWorkspaceEnforcesImmediateAuthEffects(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("workspace-superuser@example.com", parseTestAuthPassword, "Workspace Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseWorkspaceAdmin, parseErr := parseServer.authManager.parseSignup("workspace-admin@example.com", parseTestAuthPassword, "Workspace Admin")
	if parseErr != nil {
		parseT.Fatalf("parseSignup workspace admin: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-mutation-suspend")
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "workspace-admin-api-key-1",
		WorkspaceID: parseWorkspaceID,
		UserID:      parseWorkspaceAdmin.ID,
		Label:       "Workspace Admin Key",
		KeyPrefix:   "wak_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey workspace baseline: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Workspace Suspend Webhook",
		TargetURL:   "https://example.com/suspend-webhook",
		SecretHash:  "hash",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint workspace baseline: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       "workspace-suspend-job-1",
		JobType:      backgroundJobTypeWeeklySummary,
		QueueKey:     fmt.Sprintf("workspace:%d", parseWorkspaceID),
		Status:       "pending",
		AttemptCount: 0,
		MaxAttempts:  3,
		PayloadJSON:  `{}`,
		RunAfter:     "2026-03-28T00:00:00Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertBackgroundJob workspace baseline: %v", parseErr)
	}

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-admin-workspace-superuser-exec"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)

	parseWorkspaceAdminToken, parseErr := parseServer.authManager.issueToken(parseWorkspaceAdmin)
	if parseErr != nil {
		parseT.Fatalf("issueToken workspace admin: %v", parseErr)
	}
	parseWorkspaceAdminCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-admin-workspace-admin"),
		parseBuildTokenAuthContext(parseWorkspaceAdminToken),
	)
	if _, parseErr = parseServer.ListAdminUsers(parseWorkspaceAdminCtx, &chatpb.ListAdminUsersRequest{Limit: 5}); parseErr != nil {
		parseT.Fatalf("ListAdminUsers workspace-admin baseline: %v", parseErr)
	}

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationSuspendWorkspace,
		parseAdminMutationTarget{workspaceID: parseWorkspaceID, isConfirmed: true, reason: "billing hold"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction suspend workspace: %v", parseErr)
	}
	parseWorkspaceRow, isParseFound, parseErr := parseStore.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceByID suspended: %v", parseErr)
	}
	if !isParseFound || parseWorkspaceRow.Status != "suspended" {
		parseT.Fatalf("expected suspended workspace status, found=%v row=%+v", isParseFound, parseWorkspaceRow)
	}
	parseAPIKeyRows, parseErr := parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys suspended: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) == "" {
		parseT.Fatalf("expected suspended workspace API keys revoked, rows=%+v", parseAPIKeyRows)
	}
	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints suspended: %v", parseErr)
	}
	if len(parseWebhookRows) == 0 || parseWebhookRows[0].IsEnabled {
		parseT.Fatalf("expected suspended workspace webhooks disabled, rows=%+v", parseWebhookRows)
	}
	parseBackgroundJobRows, parseErr := parseStore.parseListBackgroundJobs(10)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs suspended: %v", parseErr)
	}
	if len(parseBackgroundJobRows) == 0 || parseBackgroundJobRows[0].Status != "failed" || !strings.Contains(parseBackgroundJobRows[0].ErrorMessage, "billing hold") {
		parseT.Fatalf("expected suspended workspace jobs suppressed, rows=%+v", parseBackgroundJobRows)
	}
	if _, parseErr = parseServer.GetSession(parseWorkspaceAdminCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetSession suspended workspace admin status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr = parseServer.authManager.parseLogin(parseWorkspaceAdmin.Email, parseTestAuthPassword); !errors.Is(parseErr, errInvalidCredentials) {
		parseT.Fatalf("parseLogin suspended workspace admin error=%v want=%v", parseErr, errInvalidCredentials)
	}
	if _, parseErr = parseServer.ListAdminUsers(parseWorkspaceAdminCtx, &chatpb.ListAdminUsersRequest{Limit: 5}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("ListAdminUsers suspended workspace admin status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationRestoreWorkspace,
		parseAdminMutationTarget{workspaceID: parseWorkspaceID, isConfirmed: true, reason: "billing reinstated"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction restore workspace: %v", parseErr)
	}
	parseWorkspaceRow, isParseFound, parseErr = parseStore.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceByID restored: %v", parseErr)
	}
	if !isParseFound || parseWorkspaceRow.Status != "active" {
		parseT.Fatalf("expected active workspace status after restore, found=%v row=%+v", isParseFound, parseWorkspaceRow)
	}
	if _, parseErr = parseServer.authManager.parseLogin(parseWorkspaceAdmin.Email, parseTestAuthPassword); parseErr != nil {
		parseT.Fatalf("parseLogin restored workspace admin: %v", parseErr)
	}
	if _, parseErr = parseServer.GetSession(parseWorkspaceAdminCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetSession old workspace-admin token after restore status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	parseAuditRows, parseErr := parseStore.parseListAuditLogs(20)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs workspace mutation: %v", parseErr)
	}
	parseMutationAuditCount := 0
	for _, parseAuditRow := range parseAuditRows {
		if parseAuditRow.ActorUserID != parseSuperuser.ID {
			continue
		}
		if parseAuditRow.EventType == "admin.mutation.action" {
			parseMutationAuditCount++
		}
	}
	if parseMutationAuditCount < 2 {
		parseT.Fatalf("expected at least two workspace-mutation audit rows, got %d rows=%+v", parseMutationAuditCount, parseAuditRows)
	}
}

// TestRequireAdminMutationConfirmation verifies destructive mutation confirmation and reason requirements.
func TestRequireAdminMutationConfirmation(parseT *testing.T) {
	if _, parseErr := parseRequireAdminMutationConfirmation(parseAdminMutationAction("unknown"), true, "ok"); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported action status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr := parseRequireAdminMutationConfirmation(parseAdminMutationDisableUser, false, "ok"); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing confirmation status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr := parseRequireAdminMutationConfirmation(parseAdminMutationDisableUser, true, "   "); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing reason status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	parseReason, parseErr := parseRequireAdminMutationConfirmation(parseAdminMutationRestoreWorkspace, true, "restored after validation")
	if parseErr != nil {
		parseT.Fatalf("parseRequireAdminMutationConfirmation valid: %v", parseErr)
	}
	if parseReason != "restored after validation" {
		parseT.Fatalf("expected trimmed reason, got %q", parseReason)
	}
}

// TestAdminMutationRestoreUserPreservesNonDisableAuthBlocks verifies restore clears only the user-disable auth block and keeps independent auth blocks in place.
func TestAdminMutationRestoreUserPreservesNonDisableAuthBlocks(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("restore-superuser@example.com", parseTestAuthPassword, "Restore Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseTargetUser, parseErr := parseServer.authManager.parseSignup("restore-target@example.com", parseTestAuthPassword, "Restore Target")
	if parseErr != nil {
		parseT.Fatalf("parseSignup target: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	if parseErr = parseStore.parseUpsertUserAuthBlock(parseTargetUser.ID, "risk.manual", "risk-engine", "manual risk hold"); parseErr != nil {
		parseT.Fatalf("parseUpsertUserAuthBlock manual: %v", parseErr)
	}

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-restore-user-superuser"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "manual disable"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction disable: %v", parseErr)
	}
	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationRestoreUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "manual restore"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction restore: %v", parseErr)
	}

	parseBlockCount, parseErr := parseStore.parseCountUserAuthBlocksByUser(parseTargetUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCountUserAuthBlocksByUser: %v", parseErr)
	}
	if parseBlockCount != 1 {
		parseT.Fatalf("expected exactly one remaining auth block (manual risk hold), got %d", parseBlockCount)
	}
	if _, parseErr = parseServer.authManager.parseLogin(parseTargetUser.Email, parseTestAuthPassword); !errors.Is(parseErr, errInvalidCredentials) {
		parseT.Fatalf("parseLogin should remain blocked by manual auth block, got %v", parseErr)
	}
}

// TestAdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides verifies workspace restore keeps revoked API keys and disabled webhooks unchanged.
func TestAdminMutationRestoreWorkspaceDoesNotReopenRevokedOrDisabledOverrides(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("restore-workspace-superuser@example.com", parseTestAuthPassword, "Workspace Restore Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseWorkspaceAdmin, parseErr := parseServer.authManager.parseSignup("restore-workspace-admin@example.com", parseTestAuthPassword, "Workspace Restore Admin")
	if parseErr != nil {
		parseT.Fatalf("parseSignup workspace admin: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-restore-override")

	parseAPIKeyID := "restore-api-key-1"
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       parseAPIKeyID,
		WorkspaceID: parseWorkspaceID,
		UserID:      parseWorkspaceAdmin.ID,
		Label:       "Restore API Key",
		KeyPrefix:   "rk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey: %v", parseErr)
	}
	if parseErr = parseStore.parseRevokeAPIKey(parseAPIKeyID); parseErr != nil {
		parseT.Fatalf("parseRevokeAPIKey: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Restore Webhook",
		TargetURL:   "https://example.com/restore-webhook",
		SecretHash:  "hash",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   false,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint disabled: %v", parseErr)
	}

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-restore-workspace-superuser"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)
	parseWorkspaceAdminToken, parseErr := parseServer.authManager.issueToken(parseWorkspaceAdmin)
	if parseErr != nil {
		parseT.Fatalf("issueToken workspace admin: %v", parseErr)
	}
	parseWorkspaceAdminCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-restore-workspace-admin"),
		parseBuildTokenAuthContext(parseWorkspaceAdminToken),
	)

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationSuspendWorkspace,
		parseAdminMutationTarget{workspaceID: parseWorkspaceID, isConfirmed: true, reason: "suspend for restore verification"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction suspend: %v", parseErr)
	}
	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationRestoreWorkspace,
		parseAdminMutationTarget{workspaceID: parseWorkspaceID, isConfirmed: true, reason: "restore for override verification"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction restore: %v", parseErr)
	}

	parseAPIKeyRows, parseErr := parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) == "" {
		parseT.Fatalf("expected API key to remain revoked after restore, rows=%+v", parseAPIKeyRows)
	}

	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints: %v", parseErr)
	}
	if len(parseWebhookRows) == 0 || parseWebhookRows[0].IsEnabled {
		parseT.Fatalf("expected webhook endpoint to remain disabled after restore, rows=%+v", parseWebhookRows)
	}

	if _, parseErr = parseServer.GetSession(parseWorkspaceAdminCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected old workspace-admin session to remain revoked after restore, got %v", status.Code(parseErr))
	}
	if _, parseErr = parseServer.authManager.parseLogin(parseWorkspaceAdmin.Email, parseTestAuthPassword); parseErr != nil {
		parseT.Fatalf("parseLogin restored workspace admin: %v", parseErr)
	}
}

// TestAdminMutationRestoreUserCanExplicitlyRestoreAPIKeys verifies user-restore options can explicitly re-enable user API keys.
func TestAdminMutationRestoreUserCanExplicitlyRestoreAPIKeys(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("restore-user-key-superuser@example.com", parseTestAuthPassword, "Restore User Key Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseTargetUser, parseErr := parseServer.authManager.parseSignup("restore-user-key-target@example.com", parseTestAuthPassword, "Restore User Key Target")
	if parseErr != nil {
		parseT.Fatalf("parseSignup target: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseTargetUser.ID, "ws-restore-user-keys")
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "restore-user-key-1",
		WorkspaceID: parseWorkspaceID,
		UserID:      parseTargetUser.ID,
		Label:       "Restore User Key",
		KeyPrefix:   "ruk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey baseline: %v", parseErr)
	}

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-restore-user-key-superuser"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "disable before explicit restore"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction disable: %v", parseErr)
	}
	parseAPIKeyRows, parseErr := parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys after disable: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) == "" {
		parseT.Fatalf("expected revoked API key after disable, rows=%+v", parseAPIKeyRows)
	}

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationRestoreUser,
		parseAdminMutationTarget{
			userID:               parseTargetUser.ID,
			isConfirmed:          true,
			reason:               "explicitly restore user API keys",
			isRestoreUserAPIKeys: true,
		},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction restore with API key option: %v", parseErr)
	}
	parseAPIKeyRows, parseErr = parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys after restore: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) != "" {
		parseT.Fatalf("expected restored API key after explicit restore option, rows=%+v", parseAPIKeyRows)
	}
}

// TestAdminMutationRestoreWorkspaceCanExplicitlyRestoreOperationalCapabilities verifies workspace-restore options can explicitly re-enable API keys, webhooks, and suppressed jobs.
func TestAdminMutationRestoreWorkspaceCanExplicitlyRestoreOperationalCapabilities(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("restore-workspace-option-superuser@example.com", parseTestAuthPassword, "Restore Workspace Option Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseWorkspaceAdmin, parseErr := parseServer.authManager.parseSignup("restore-workspace-option-admin@example.com", parseTestAuthPassword, "Restore Workspace Option Admin")
	if parseErr != nil {
		parseT.Fatalf("parseSignup workspace admin: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-restore-workspace-options")
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "restore-workspace-key-1",
		WorkspaceID: parseWorkspaceID,
		UserID:      parseWorkspaceAdmin.ID,
		Label:       "Restore Workspace Key",
		KeyPrefix:   "rwk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey baseline: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Restore Workspace Webhook",
		TargetURL:   "https://example.com/restore-workspace-webhook",
		SecretHash:  "hash",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint baseline: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       "restore-workspace-job-1",
		JobType:      backgroundJobTypeWeeklySummary,
		QueueKey:     fmt.Sprintf("workspace:%d", parseWorkspaceID),
		Status:       "pending",
		AttemptCount: 0,
		MaxAttempts:  3,
		PayloadJSON:  `{}`,
		RunAfter:     "2026-03-28T00:00:00Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertBackgroundJob baseline: %v", parseErr)
	}

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-restore-workspace-option-superuser"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationSuspendWorkspace,
		parseAdminMutationTarget{workspaceID: parseWorkspaceID, isConfirmed: true, reason: "suspend before explicit restore options"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction suspend: %v", parseErr)
	}
	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationRestoreWorkspace,
		parseAdminMutationTarget{
			workspaceID:               parseWorkspaceID,
			isConfirmed:               true,
			reason:                    "explicit restore workspace capabilities",
			isRestoreWorkspaceAPIKeys: true,
			isRestoreWebhookEndpoints: true,
			isRestoreBackgroundJobs:   true,
		},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction restore with options: %v", parseErr)
	}

	parseAPIKeyRows, parseErr := parseStore.parseListAPIKeys(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAPIKeys: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 || strings.TrimSpace(parseAPIKeyRows[0].RevokedAt) != "" {
		parseT.Fatalf("expected workspace API keys restored when requested, rows=%+v", parseAPIKeyRows)
	}
	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints: %v", parseErr)
	}
	if len(parseWebhookRows) == 0 || !parseWebhookRows[0].IsEnabled {
		parseT.Fatalf("expected workspace webhooks re-enabled when requested, rows=%+v", parseWebhookRows)
	}
	parseBackgroundJobRows, parseErr := parseStore.parseListBackgroundJobs(10)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs: %v", parseErr)
	}
	if len(parseBackgroundJobRows) == 0 || parseBackgroundJobRows[0].Status != "pending" || strings.TrimSpace(parseBackgroundJobRows[0].ErrorMessage) != "" {
		parseT.Fatalf("expected workspace jobs restored to pending when requested, rows=%+v", parseBackgroundJobRows)
	}
}

// TestAdminMutationDiagnosticsLogs verifies mutation submit, denial, confirmation, success, and rollback diagnostics are emitted.
func TestAdminMutationDiagnosticsLogs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSuperuser, parseErr := parseServer.authManager.parseSignup("mutation-diagnostics-superuser@example.com", parseTestAuthPassword, "Mutation Diagnostics Superuser")
	if parseErr != nil {
		parseT.Fatalf("parseSignup superuser: %v", parseErr)
	}
	parseTargetUser, parseErr := parseServer.authManager.parseSignup("mutation-diagnostics-target@example.com", parseTestAuthPassword, "Mutation Diagnostics Target")
	if parseErr != nil {
		parseT.Fatalf("parseSignup target: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseTargetUser.ID, "ws-mutation-diagnostics-target")

	parseSuperuserToken, parseErr := parseServer.authManager.issueToken(parseSuperuser)
	if parseErr != nil {
		parseT.Fatalf("issueToken superuser: %v", parseErr)
	}
	parseSuperuserCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-admin-mutation-diagnostics-superuser"),
		parseBuildTokenAuthContext(parseSuperuserToken),
	)

	parseNonAdmin, parseErr := parseServer.authManager.parseSignup("mutation-diagnostics-nonadmin@example.com", parseTestAuthPassword, "Mutation Diagnostics Nonadmin")
	if parseErr != nil {
		parseT.Fatalf("parseSignup nonadmin: %v", parseErr)
	}
	parseNonAdminToken, parseErr := parseServer.authManager.issueToken(parseNonAdmin)
	if parseErr != nil {
		parseT.Fatalf("issueToken nonadmin: %v", parseErr)
	}
	parseNonAdminCtx := metadata.NewIncomingContext(
		parseNewAuthenticatedContext("peer-admin-mutation-diagnostics-nonadmin"),
		parseBuildTokenAuthContext(parseNonAdminToken),
	)

	var parseLogs bytes.Buffer
	parseServer.logger = slog.New(slog.NewTextHandler(&parseLogs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseNonAdminCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "nonadmin should be denied"},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parseExecuteAdminMutationAction nonadmin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: false, reason: "missing confirm"},
	); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("parseExecuteAdminMutationAction missing confirm status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseTargetUser.ID, isConfirmed: true, reason: "suspicious account activity"},
	); parseErr != nil {
		parseT.Fatalf("parseExecuteAdminMutationAction success: %v", parseErr)
	}
	if parseErr = parseServer.parseExecuteAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationSuspendWorkspace,
		parseAdminMutationTarget{workspaceID: 999999, isConfirmed: true, reason: "missing workspace should trigger rollback diagnostics"},
	); status.Code(parseErr) != codes.Internal {
		parseT.Fatalf("parseExecuteAdminMutationAction rollback-path status code=%v want=%v", status.Code(parseErr), codes.Internal)
	}

	parseLogOutput := parseLogs.String()
	for _, parsePattern := range []string{
		"rpc.admin mutation submit",
		"rpc.admin mutation denied",
		"stage=authorize",
		"stage=confirmation",
		"rpc.admin mutation confirmed",
		"rpc.admin mutation success",
		"rpc.admin mutation rollback required",
	} {
		if !strings.Contains(parseLogOutput, parsePattern) {
			parseT.Fatalf("expected mutation diagnostic log pattern %q, got logs=%q", parsePattern, parseLogOutput)
		}
	}
}

// BenchmarkParseBuildAdminMutationScopeLabel reports micro-benchmark throughput for mutation scope-label diagnostics.
func BenchmarkParseBuildAdminMutationScopeLabel(parseB *testing.B) {
	parseScope := parseAdminAccessScope{isPlatformScope: true}
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseBuildAdminMutationScopeLabel(parseScope)
	}
}
