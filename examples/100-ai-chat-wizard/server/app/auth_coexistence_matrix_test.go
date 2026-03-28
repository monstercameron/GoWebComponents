package app

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// parseCountUsersByEmail returns the persisted user count for one normalized email.
func parseCountUsersByEmail(parseT *testing.T, parseStore *Store, parseEmail string) int64 {
	parseT.Helper()
	parseRow := parseStore.db.QueryRow(`SELECT COUNT(1) FROM users WHERE email = ?`, parseNormalizeAuthEmail(parseEmail))
	var parseUserCount int64
	if parseErr := parseRow.Scan(&parseUserCount); parseErr != nil {
		parseT.Fatalf("count users by email: %v", parseErr)
	}
	return parseUserCount
}

// parseBuildAuthContextWithToken wraps one bearer token into incoming gRPC metadata.
func parseBuildAuthContextWithToken(parseToken string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseToken))
}

// TestExternalIdentityCoexistenceMatrix verifies password+external link/login decisions and session continuity without duplicate-account drift.
func TestExternalIdentityCoexistenceMatrix(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parsePasswordUser, parseErr := parseAuth.parseSignup("customer@email.com", "password123", "Customer")
	if parseErr != nil {
		parseT.Fatalf("parseSignup password user: %v", parseErr)
	}
	if parseUserCount := parseCountUsersByEmail(parseT, parseStore, "customer@email.com"); parseUserCount != 1 {
		parseT.Fatalf("seed user count=%d want=1", parseUserCount)
	}

	parseLinkDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-1",
		ParseVerifiedEmail:   "customer@email.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{
				ParseUserID:                parsePasswordUser.ID,
				IsParsePasswordAuthEnabled: true,
			},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("link-existing decision: %v", parseErr)
	}
	if parseLinkDecision.ParseAction != parseExternalIdentityLinkActionLinkExisting || parseLinkDecision.ParseUserID != parsePasswordUser.ID || !parseLinkDecision.IsParsePasswordBootstrapLink {
		parseT.Fatalf("unexpected link-existing decision: %+v", parseLinkDecision)
	}

	if _, parseErr = parseStore.parseUpsertAuthIdentity(parseAuthIdentityWrite{
		UserID:          parsePasswordUser.ID,
		ProviderKey:     "google_oidc",
		ProviderType:    "oidc",
		ProviderSubject: "google-subject-1",
		Email:           "customer@email.com",
		IsEmailVerified: true,
		ProfileJSON:     `{"name":"Customer"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertAuthIdentity: %v", parseErr)
	}

	parseLoginDecision, parseErr := parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:          "google_oidc",
		ParseProviderSubject:      "google-subject-1",
		ParseSubjectLinkedUserIDs: []int64{parsePasswordUser.ID},
	})
	if parseErr != nil {
		parseT.Fatalf("login-existing decision: %v", parseErr)
	}
	if parseLoginDecision.ParseAction != parseExternalIdentityLinkActionLoginExisting || parseLoginDecision.ParseUserID != parsePasswordUser.ID {
		parseT.Fatalf("unexpected login-existing decision: %+v", parseLoginDecision)
	}
	if parseUserCount := parseCountUsersByEmail(parseT, parseStore, "customer@email.com"); parseUserCount != 1 {
		parseT.Fatalf("post-link user count=%d want=1", parseUserCount)
	}

	if _, parseErr = parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-ambiguous",
		ParseVerifiedEmail:   "customer@email.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{ParseUserID: parsePasswordUser.ID},
			{ParseUserID: parsePasswordUser.ID + 100},
		},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("ambiguous email match status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseResolveExternalIdentityLinkDecision(parseExternalIdentityLinkRequest{
		ParseProviderKey:     "google_oidc",
		ParseProviderSubject: "google-subject-conflict",
		ParseVerifiedEmail:   "customer@email.com",
		IsParseEmailVerified: true,
		ParseEmailMatchedUsers: []parseExternalIdentityEmailMatch{
			{
				ParseUserID:                 parsePasswordUser.ID,
				IsParsePasswordAuthEnabled:  true,
				ParseProviderLinkedSubjects: []string{"other-google-subject"},
			},
		},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("provider subject conflict status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	parseExternalToken, parseErr := parseAuth.issueTokenForContextWithAuthMethod(context.Background(), authUser{
		ID:    parsePasswordUser.ID,
		Email: parsePasswordUser.Email,
	}, "", parseWorkspaceAuthMethodGoogleOIDC)
	if parseErr != nil {
		parseT.Fatalf("issue external token: %v", parseErr)
	}
	parseAuthCtx := parseBuildAuthContextWithToken(parseExternalToken)
	parseSessionUser1, parseSessionClaims1, parseSessionOK1 := parseAuth.parseAuthenticatedSessionFromContext(parseAuthCtx)
	if !parseSessionOK1 {
		parseT.Fatal("first external session validation failed")
	}
	parseSessionUser2, parseSessionClaims2, parseSessionOK2 := parseAuth.parseAuthenticatedSessionFromContext(parseBuildAuthContextWithToken(parseExternalToken))
	if !parseSessionOK2 {
		parseT.Fatal("second external session validation failed")
	}
	if parseSessionUser1.ID != parsePasswordUser.ID || parseSessionUser2.ID != parsePasswordUser.ID {
		parseT.Fatalf("unexpected session users: first=%+v second=%+v want-user-id=%d", parseSessionUser1, parseSessionUser2, parsePasswordUser.ID)
	}
	if parseSessionClaims1.SessionID == "" || parseSessionClaims1.SessionID != parseSessionClaims2.SessionID {
		parseT.Fatalf("session id continuity failed across refresh/reopen checks: first=%q second=%q", parseSessionClaims1.SessionID, parseSessionClaims2.SessionID)
	}
}
