package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestHandleGoogleOIDCStartStoresState verifies handshake-start validation and persisted state/nonce hashes.
func TestHandleGoogleOIDCStartStoresState(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "google-start-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "google-start-workspace")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseStartResponse, parseErr := parseServer.parseHandleGoogleOIDCStart(parseGoogleOIDCStartRequest{
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app/auth/callback",
		ParseSessionKey:      "google-start-session",
		ParseExpectedSubject: "google-subject-start",
		ParseCreatedByUserID: parseUser.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleGoogleOIDCStart: %v", parseErr)
	}
	if parseStartResponse.ParseProviderKey != parseGoogleOIDCProviderKey || parseStartResponse.ParseStateToken == "" || parseStartResponse.ParseNonceToken == "" || parseStartResponse.ParseSessionKey == "" {
		parseT.Fatalf("unexpected google oidc start response: %+v", parseStartResponse)
	}
	parseStateRow, isParseFound, parseErr := parseStore.parseGetAuthOIDCStateByStateTokenHash(parseBuildAuthFlowTokenHash(parseStartResponse.ParseStateToken))
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthOIDCStateByStateTokenHash: %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatal("expected stored google oidc state row")
	}
	if parseStateRow.ProviderKey != parseGoogleOIDCProviderKey || parseStateRow.WorkspaceID != parseWorkspaceID || parseStateRow.SessionKey != parseStartResponse.ParseSessionKey || parseStateRow.NonceTokenHash != parseBuildAuthFlowTokenHash(parseStartResponse.ParseNonceToken) {
		parseT.Fatalf("unexpected persisted google oidc state row: %+v", parseStateRow)
	}
}

// TestHandleGoogleOIDCStartRequiresScopedWorkspace verifies start rejects missing workspace/creator scope.
func TestHandleGoogleOIDCStartRequiresScopedWorkspace(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	if _, parseErr := parseServer.parseHandleGoogleOIDCStart(parseGoogleOIDCStartRequest{
		ParseReturnToURL: "/app",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected missing scope to fail with invalid argument, got %v", status.Code(parseErr))
	}
}

// TestHandleGoogleOIDCCallbackCreateUser verifies create-user callback path and auth session issuance.
func TestHandleGoogleOIDCCallbackCreateUser(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseWorkspaceOwner := parseMustCreateUser(parseT, parseStore, "google-callback-create-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceOwner.ID, "google-callback-create-workspace")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseStartResponse, parseErr := parseServer.parseHandleGoogleOIDCStart(parseGoogleOIDCStartRequest{
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app",
		ParseSessionKey:      "google-callback-create-session",
		ParseCreatedByUserID: parseWorkspaceOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleGoogleOIDCStart: %v", parseErr)
	}
	parseCallbackResponse, parseErr := parseServer.parseHandleGoogleOIDCCallback(parseNewAuthenticatedContext("google-callback-create-peer"), parseGoogleOIDCCallbackRequest{
		ParseStateToken:      parseStartResponse.ParseStateToken,
		ParseNonceToken:      parseStartResponse.ParseNonceToken,
		ParseReturnToURL:     parseStartResponse.ParseReturnToURL,
		ParseSessionKey:      parseStartResponse.ParseSessionKey,
		ParseProviderSub:     "google-subject-create-1",
		ParseVerifiedEmail:   "google-create-user@example.com",
		ParseDisplayName:     "Google Create User",
		ParseProfileJSON:     `{"name":"Google Create User"}`,
		IsParseEmailVerified: true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleGoogleOIDCCallback(create): %v", parseErr)
	}
	if parseCallbackResponse.ParseDecision != parseExternalIdentityLinkActionCreateUser || !parseCallbackResponse.IsParseUserCreated || parseCallbackResponse.ParseUser.ID <= 0 {
		parseT.Fatalf("unexpected callback create response: %+v", parseCallbackResponse)
	}
	if parseCallbackResponse.ParseUser.Email != "google-create-user@example.com" {
		parseT.Fatalf("unexpected callback user email: %+v", parseCallbackResponse.ParseUser)
	}
	parseParsedTokenUser, parseParsedClaims, parseErr := parseServer.authManager.parseTokenWithMetadata(parseCallbackResponse.ParseAuthToken, parseAuthRequestMetadata{})
	if parseErr != nil {
		parseT.Fatalf("parseTokenWithMetadata: %v", parseErr)
	}
	if parseParsedTokenUser.ID != parseCallbackResponse.ParseUser.ID || parseParsedClaims.AuthMethod != parseWorkspaceAuthMethodGoogleOIDC {
		parseT.Fatalf("unexpected parsed token data: user=%+v claims=%+v", parseParsedTokenUser, parseParsedClaims)
	}
	parseIdentityRow, isParseFound, parseErr := parseStore.parseGetAuthIdentityByProviderSubject(parseGoogleOIDCProviderKey, "google-subject-create-1")
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthIdentityByProviderSubject: %v", parseErr)
	}
	if !isParseFound || parseIdentityRow.UserID != parseCallbackResponse.ParseUser.ID {
		parseT.Fatalf("unexpected identity row after callback create: row=%+v found=%v", parseIdentityRow, isParseFound)
	}
	parseSessionRow, isParseFound, parseErr := parseStore.parseGetAuthSessionBySessionID(parseStartResponse.ParseSessionKey)
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthSessionBySessionID: %v", parseErr)
	}
	if !isParseFound || parseSessionRow.UserID != parseCallbackResponse.ParseUser.ID {
		parseT.Fatalf("unexpected auth session row after callback create: row=%+v found=%v", parseSessionRow, isParseFound)
	}
	if _, parseErr = parseServer.parseHandleGoogleOIDCCallback(parseNewAuthenticatedContext("google-callback-create-peer-2"), parseGoogleOIDCCallbackRequest{
		ParseStateToken:      parseStartResponse.ParseStateToken,
		ParseNonceToken:      parseStartResponse.ParseNonceToken,
		ParseReturnToURL:     parseStartResponse.ParseReturnToURL,
		ParseSessionKey:      parseStartResponse.ParseSessionKey,
		ParseProviderSub:     "google-subject-create-1",
		ParseVerifiedEmail:   "google-create-user@example.com",
		IsParseEmailVerified: true,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected callback replay to fail with permission denied, got %v", status.Code(parseErr))
	}
}

// TestHandleGoogleOIDCCallbackLinkExisting verifies verified-email link-to-existing behavior.
func TestHandleGoogleOIDCCallbackLinkExisting(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseExistingUser := parseMustCreateUser(parseT, parseStore, "google-link-user@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseExistingUser.ID, "google-callback-link-workspace")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseStartResponse, parseErr := parseServer.parseHandleGoogleOIDCStart(parseGoogleOIDCStartRequest{
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app",
		ParseSessionKey:      "google-callback-link-session",
		ParseCreatedByUserID: parseExistingUser.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleGoogleOIDCStart: %v", parseErr)
	}
	parseCallbackResponse, parseErr := parseServer.parseHandleGoogleOIDCCallback(parseNewAuthenticatedContext("google-callback-link-peer"), parseGoogleOIDCCallbackRequest{
		ParseStateToken:      parseStartResponse.ParseStateToken,
		ParseNonceToken:      parseStartResponse.ParseNonceToken,
		ParseReturnToURL:     parseStartResponse.ParseReturnToURL,
		ParseSessionKey:      parseStartResponse.ParseSessionKey,
		ParseProviderSub:     "google-subject-link-1",
		ParseVerifiedEmail:   "google-link-user@example.com",
		ParseProfileJSON:     `{"name":"Google Link User"}`,
		IsParseEmailVerified: true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleGoogleOIDCCallback(link): %v", parseErr)
	}
	if parseCallbackResponse.ParseDecision != parseExternalIdentityLinkActionLinkExisting || !parseCallbackResponse.IsParseUserLinked || parseCallbackResponse.ParseUser.ID != parseExistingUser.ID {
		parseT.Fatalf("unexpected callback link response: %+v", parseCallbackResponse)
	}
	parseIdentityRow, isParseFound, parseErr := parseStore.parseGetAuthIdentityByProviderSubject(parseGoogleOIDCProviderKey, "google-subject-link-1")
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthIdentityByProviderSubject: %v", parseErr)
	}
	if !isParseFound || parseIdentityRow.UserID != parseExistingUser.ID {
		parseT.Fatalf("expected linked identity row for existing user %d, got row=%+v found=%v", parseExistingUser.ID, parseIdentityRow, isParseFound)
	}
}

// TestHandleOIDCProviderStartStoresState verifies generic OIDC start persists state for provider key `oidc`.
func TestHandleOIDCProviderStartStoresState(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "oidc-start-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "oidc-start-workspace")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseStartResponse, parseErr := parseServer.parseHandleOIDCProviderStart(parseOIDCStartRequest{
		ParseProviderKey:     parseWorkspaceAuthMethodOIDC,
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app/auth/callback",
		ParseSessionKey:      "oidc-start-session",
		ParseExpectedSubject: "oidc-subject-start",
		ParseCreatedByUserID: parseUser.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderStart: %v", parseErr)
	}
	if parseStartResponse.ParseProviderKey != parseWorkspaceAuthMethodOIDC {
		parseT.Fatalf("expected provider key oidc, got %q", parseStartResponse.ParseProviderKey)
	}
	parseStateRow, isParseFound, parseErr := parseStore.parseGetAuthOIDCStateByStateTokenHash(parseBuildAuthFlowTokenHash(parseStartResponse.ParseStateToken))
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthOIDCStateByStateTokenHash: %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatal("expected stored oidc state row")
	}
	if parseStateRow.ProviderKey != parseWorkspaceAuthMethodOIDC || parseStateRow.WorkspaceID != parseWorkspaceID {
		parseT.Fatalf("unexpected persisted oidc state row: %+v", parseStateRow)
	}
}

// TestHandleOIDCProviderCallbackCreateUser verifies generic OIDC callback create-user path and auth method mapping.
func TestHandleOIDCProviderCallbackCreateUser(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseWorkspaceOwner := parseMustCreateUser(parseT, parseStore, "oidc-callback-create-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceOwner.ID, "oidc-callback-create-workspace")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseStartResponse, parseErr := parseServer.parseHandleOIDCProviderStart(parseOIDCStartRequest{
		ParseProviderKey:     parseWorkspaceAuthMethodOIDC,
		ParseWorkspaceID:     parseWorkspaceID,
		ParseReturnToURL:     "/app",
		ParseSessionKey:      "oidc-callback-create-session",
		ParseCreatedByUserID: parseWorkspaceOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderStart: %v", parseErr)
	}
	parseCallbackResponse, parseErr := parseServer.parseHandleOIDCProviderCallback(parseNewAuthenticatedContext("oidc-callback-create-peer"), parseOIDCCallbackRequest{
		ParseProviderKey:   parseWorkspaceAuthMethodOIDC,
		ParseStateToken:    parseStartResponse.ParseStateToken,
		ParseNonceToken:    parseStartResponse.ParseNonceToken,
		ParseReturnToURL:   parseStartResponse.ParseReturnToURL,
		ParseSessionKey:    parseStartResponse.ParseSessionKey,
		ParseProviderSub:   "oidc-subject-create-1",
		ParseVerifiedEmail: "oidc-create-user@example.com",
		ParseDisplayName:   "OIDC Create User",
		ParseProfileJSON:   `{"name":"OIDC Create User"}`,
		IsEmailVerified:    true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderCallback(create): %v", parseErr)
	}
	if parseCallbackResponse.ParseDecision != parseExternalIdentityLinkActionCreateUser || !parseCallbackResponse.IsParseUserCreated || parseCallbackResponse.ParseUser.ID <= 0 {
		parseT.Fatalf("unexpected oidc callback create response: %+v", parseCallbackResponse)
	}
	parseParsedTokenUser, parseParsedClaims, parseErr := parseServer.authManager.parseTokenWithMetadata(parseCallbackResponse.ParseAuthToken, parseAuthRequestMetadata{})
	if parseErr != nil {
		parseT.Fatalf("parseTokenWithMetadata: %v", parseErr)
	}
	if parseParsedTokenUser.ID != parseCallbackResponse.ParseUser.ID || parseParsedClaims.AuthMethod != parseWorkspaceAuthMethodOIDC {
		parseT.Fatalf("unexpected oidc parsed token data: user=%+v claims=%+v", parseParsedTokenUser, parseParsedClaims)
	}
	parseIdentityRow, isParseFound, parseErr := parseStore.parseGetAuthIdentityByProviderSubject(parseWorkspaceAuthMethodOIDC, "oidc-subject-create-1")
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthIdentityByProviderSubject: %v", parseErr)
	}
	if !isParseFound || parseIdentityRow.UserID != parseCallbackResponse.ParseUser.ID {
		parseT.Fatalf("unexpected oidc identity row after callback create: row=%+v found=%v", parseIdentityRow, isParseFound)
	}
}

// TestHandleOIDCProviderCallbackEnforcesWorkspaceLinkPolicy verifies workspace policy can deny linking external identities to password-auth accounts.
func TestHandleOIDCProviderCallbackEnforcesWorkspaceLinkPolicy(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseWorkspaceOwner := parseMustCreateUser(parseT, parseStore, "oidc-policy-owner@example.com")
	parseExistingPasswordUser := parseMustCreateUser(parseT, parseStore, "oidc-policy-target@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceOwner.ID, "oidc-policy-workspace")
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
		ParseSessionKey:      "oidc-policy-link-session",
		ParseCreatedByUserID: parseWorkspaceOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleOIDCProviderStart: %v", parseErr)
	}
	if _, parseErr = parseServer.parseHandleOIDCProviderCallback(parseNewAuthenticatedContext("oidc-policy-link-peer"), parseOIDCCallbackRequest{
		ParseProviderKey:   parseWorkspaceAuthMethodOIDC,
		ParseStateToken:    parseStartResponse.ParseStateToken,
		ParseNonceToken:    parseStartResponse.ParseNonceToken,
		ParseReturnToURL:   parseStartResponse.ParseReturnToURL,
		ParseSessionKey:    parseStartResponse.ParseSessionKey,
		ParseProviderSub:   "oidc-policy-link-subject-1",
		ParseVerifiedEmail: parseExistingPasswordUser.Email,
		IsEmailVerified:    true,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected workspace policy to deny password-account link, got %v", status.Code(parseErr))
	}
}

// BenchmarkParseResolveExternalDisplayName measures display-name fallback normalization for external-auth user creation.
func BenchmarkParseResolveExternalDisplayName(parseB *testing.B) {
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseResolveExternalDisplayName("", "benchmark-user@example.com")
	}
}
