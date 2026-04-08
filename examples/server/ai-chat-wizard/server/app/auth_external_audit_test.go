package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestExternalAuthAuditSuccessLifecycle verifies login-start and callback-success audit events are queryable for one workspace.
func TestExternalAuthAuditSuccessLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseWorkspaceOwner := parseMustCreateUser(parseT, parseStore, "external-audit-success-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceOwner.ID, "external-audit-success")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseStartResponse, parseErr := parseServer.parseHandleOIDCProviderStart(parseOIDCStartRequest{
		ParseProviderKey:     parseWorkspaceAuthMethodOIDC,
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app",
		ParseSessionKey:      "external-audit-success-session",
		ParseCreatedByUserID: parseWorkspaceOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderStart: %v", parseErr)
	}
	if _, parseErr = parseServer.parseHandleOIDCProviderCallback(parseNewAuthenticatedContext("external-audit-success-peer"), parseOIDCCallbackRequest{
		ParseProviderKey:   parseWorkspaceAuthMethodOIDC,
		ParseStateToken:    parseStartResponse.ParseStateToken,
		ParseNonceToken:    parseStartResponse.ParseNonceToken,
		ParseReturnToURL:   parseStartResponse.ParseReturnToURL,
		ParseSessionKey:    parseStartResponse.ParseSessionKey,
		ParseProviderSub:   "external-audit-success-subject-1",
		ParseVerifiedEmail: "external-audit-success-user@example.com",
		IsEmailVerified:    true,
	}); parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderCallback(success): %v", parseErr)
	}

	parseAuditRows, parseErr := parseStore.parseListExternalAuthAuditLogs(parseWorkspaceID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListExternalAuthAuditLogs(success): %v", parseErr)
	}
	if !parseHasExternalAuthEventType(parseAuditRows, parseExternalAuthAuditEventLoginStart) {
		parseT.Fatalf("expected login_start audit event, rows=%+v", parseAuditRows)
	}
	if !parseHasExternalAuthEventType(parseAuditRows, parseExternalAuthAuditEventCallbackSuccess) {
		parseT.Fatalf("expected callback_success audit event, rows=%+v", parseAuditRows)
	}
}

// TestExternalAuthAuditPolicyDeniedLifecycle verifies policy-denied callback paths emit queryable failure and enforcement audit events.
func TestExternalAuthAuditPolicyDeniedLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseWorkspaceOwner := parseMustCreateUser(parseT, parseStore, "external-audit-policy-owner@example.com")
	parseExistingPasswordUser := parseMustCreateUser(parseT, parseStore, "external-audit-policy-target@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceOwner.ID, "external-audit-policy")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	if parseErr := parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        false,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            true,
		RequiredProviderKey:      parseWorkspaceAuthMethodOIDC,
		IsJITProvisioningAllowed: true,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseWorkspaceOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceAuthPolicy: %v", parseErr)
	}
	parseStartResponse, parseErr := parseServer.parseHandleOIDCProviderStart(parseOIDCStartRequest{
		ParseProviderKey:     parseWorkspaceAuthMethodOIDC,
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app",
		ParseSessionKey:      "external-audit-policy-session",
		ParseCreatedByUserID: parseWorkspaceOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderStart: %v", parseErr)
	}
	_, parseErr = parseServer.parseHandleOIDCProviderCallback(parseNewAuthenticatedContext("external-audit-policy-peer"), parseOIDCCallbackRequest{
		ParseProviderKey:   parseWorkspaceAuthMethodOIDC,
		ParseStateToken:    parseStartResponse.ParseStateToken,
		ParseNonceToken:    parseStartResponse.ParseNonceToken,
		ParseReturnToURL:   parseStartResponse.ParseReturnToURL,
		ParseSessionKey:    parseStartResponse.ParseSessionKey,
		ParseProviderSub:   "external-audit-policy-subject-1",
		ParseVerifiedEmail: parseExistingPasswordUser.Email,
		IsEmailVerified:    true,
	})
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected policy denied callback, got %v", status.Code(parseErr))
	}

	parseAuditRows, parseErr := parseStore.parseListExternalAuthAuditLogs(0, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListExternalAuthAuditLogs(policy denied): %v", parseErr)
	}
	if !parseHasExternalAuthEventType(parseAuditRows, parseExternalAuthAuditEventWorkspaceSSOEnforcement) {
		parseT.Fatalf("expected workspace_sso_enforcement audit event, rows=%+v", parseAuditRows)
	}
	if !parseHasExternalAuthEventType(parseAuditRows, parseExternalAuthAuditEventPolicyDeniedLogin) {
		parseT.Fatalf("expected policy_denied_login audit event, rows=%+v", parseAuditRows)
	}
	if !parseHasExternalAuthEventType(parseAuditRows, parseExternalAuthAuditEventCallbackFailure) {
		parseT.Fatalf("expected callback_failure audit event, rows=%+v", parseAuditRows)
	}
}

// TestExternalAuthAuditIdentityUnlinkedLifecycle verifies identity-unlinked events are queryable from the external-auth audit stream.
func TestExternalAuthAuditIdentityUnlinkedLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseWorkspaceOwner := parseMustCreateUser(parseT, parseStore, "external-audit-unlink-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceOwner.ID, "external-audit-unlink")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseServer.parseTrackExternalAuthIdentityUnlinked(parseWorkspaceOwner.ID, parseWorkspaceID, parseWorkspaceOwner.ID, parseWorkspaceAuthMethodOIDC)

	parseAuditRows, parseErr := parseStore.parseListExternalAuthAuditLogs(parseWorkspaceID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListExternalAuthAuditLogs(identity unlinked): %v", parseErr)
	}
	if !parseHasExternalAuthEventType(parseAuditRows, parseExternalAuthAuditEventIdentityUnlinked) {
		parseT.Fatalf("expected identity_unlinked audit event, rows=%+v", parseAuditRows)
	}
}

// parseHasExternalAuthEventType reports whether one external-auth audit row slice contains one expected event type.
func parseHasExternalAuthEventType(parseRows []parseAuditLogRow, parseEventType string) bool {
	for _, parseRow := range parseRows {
		if parseRow.EventType == parseEventType {
			return true
		}
	}
	return false
}
