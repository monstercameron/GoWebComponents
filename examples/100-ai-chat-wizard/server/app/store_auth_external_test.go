package app

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestStoreAuthIdentityLifecycle verifies auth identity upsert/read/list/delete behavior.
func TestStoreAuthIdentityLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "identity-owner@example.com")
	parseOtherUser := parseMustCreateUser(parseT, parseStore, "identity-owner-2@example.com")
	parseLoginAt := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)

	parseIdentityRow, parseErr := parseStore.parseUpsertAuthIdentity(parseAuthIdentityWrite{
		UserID:          parseUser.ID,
		ProviderKey:     "google_oidc",
		ProviderType:    "oidc",
		ProviderSubject: "google-subject-123",
		Email:           "Identity.Owner@example.com",
		IsEmailVerified: true,
		ProfileJSON:     `{"name":"Identity Owner"}`,
		LastLoginAt:     parseLoginAt,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertAuthIdentity(create): %v", parseErr)
	}
	if parseIdentityRow.UserID != parseUser.ID || parseIdentityRow.ProviderKey != "google_oidc" || !parseIdentityRow.IsEmailVerified {
		parseT.Fatalf("unexpected identity row after create: %+v", parseIdentityRow)
	}

	parseFetchedIdentityRow, isParseFound, parseErr := parseStore.parseGetAuthIdentityByProviderSubject("google_oidc", "google-subject-123")
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthIdentityByProviderSubject: %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatal("expected auth identity row to exist")
	}
	if parseFetchedIdentityRow.Email != "identity.owner@example.com" || parseFetchedIdentityRow.ProviderType != "oidc" {
		parseT.Fatalf("unexpected fetched identity row: %+v", parseFetchedIdentityRow)
	}

	parseIdentityRows, parseErr := parseStore.parseListAuthIdentitiesByUser(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListAuthIdentitiesByUser(before upsert update): %v", parseErr)
	}
	if len(parseIdentityRows) != 1 {
		parseT.Fatalf("expected one identity row, got %d", len(parseIdentityRows))
	}

	parseUpdatedIdentityRow, parseErr := parseStore.parseUpsertAuthIdentity(parseAuthIdentityWrite{
		UserID:          parseOtherUser.ID,
		ProviderKey:     "google_oidc",
		ProviderType:    "oidc",
		ProviderSubject: "google-subject-123",
		Email:           "identity-owner-2@example.com",
		IsEmailVerified: false,
		ProfileJSON:     `{"name":"Identity Owner 2"}`,
		LastLoginAt:     time.Now().UTC().Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertAuthIdentity(update): %v", parseErr)
	}
	if parseUpdatedIdentityRow.UserID != parseOtherUser.ID || parseUpdatedIdentityRow.IsEmailVerified {
		parseT.Fatalf("expected subject upsert to move to second user and clear verification, got %+v", parseUpdatedIdentityRow)
	}

	if parseErr = parseStore.parseDeleteAuthIdentityByScope(parseOtherUser.ID, "google_oidc"); parseErr != nil {
		parseT.Fatalf("parseDeleteAuthIdentityByScope: %v", parseErr)
	}
	parseIdentityRows, parseErr = parseStore.parseListAuthIdentitiesByUser(parseOtherUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListAuthIdentitiesByUser(after delete): %v", parseErr)
	}
	if len(parseIdentityRows) != 0 {
		parseT.Fatalf("expected zero identity rows after delete, got %+v", parseIdentityRows)
	}

	if _, parseErr = parseStore.parseUpsertAuthIdentity(parseAuthIdentityWrite{
		UserID:          parseUser.ID,
		ProviderKey:     "saml",
		ProviderType:    "invalid-provider-type",
		ProviderSubject: "saml-subject-123",
		Email:           "identity-owner@example.com",
		IsEmailVerified: true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertAuthIdentity(saml fallback): %v", parseErr)
	}
	if parseFetchedIdentityRow, isParseFound, parseErr = parseStore.parseGetAuthIdentityByProviderSubject("saml", "saml-subject-123"); parseErr != nil {
		parseT.Fatalf("parseGetAuthIdentityByProviderSubject(saml fallback): %v", parseErr)
	} else if !isParseFound {
		parseT.Fatal("expected saml auth identity row to exist")
	} else if parseFetchedIdentityRow.ProviderType != "saml" {
		parseT.Fatalf("expected provider type fallback to saml, got %+v", parseFetchedIdentityRow)
	}

	if _, parseErr = parseStore.parseUpsertAuthIdentity(parseAuthIdentityWrite{
		UserID:          parseUser.ID,
		ProviderKey:     "github_oidc",
		ProviderType:    "oidc",
		ProviderSubject: "github-subject-123",
	}); parseErr == nil || !strings.Contains(parseErr.Error(), "provider key is required") {
		parseT.Fatalf("expected invalid provider key validation error, got %v", parseErr)
	}
}

// TestStoreAuthOIDCStateLifecycle verifies OIDC state create/read/consume/expire lifecycle behavior.
func TestStoreAuthOIDCStateLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "oidc-state-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "oidc-state-workspace")

	parseStateRow, parseErr := parseStore.parseCreateAuthOIDCState(parseAuthOIDCStateWrite{
		ProviderKey:     "google_oidc",
		WorkspaceID:     parseWorkspaceID,
		SessionKey:      "session-key-1",
		StateTokenHash:  "state-hash-1",
		NonceTokenHash:  "nonce-hash-1",
		ReturnToURL:     "/app/auth/callback",
		ExpectedSubject: "sub-123",
		ExpiresAt:       time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
		CreatedByUserID: parseUser.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseCreateAuthOIDCState: %v", parseErr)
	}
	if parseStateRow.ProviderKey != "google_oidc" || parseStateRow.WorkspaceID != parseWorkspaceID {
		parseT.Fatalf("unexpected created oidc state row: %+v", parseStateRow)
	}

	parseFetchedStateRow, isParseFound, parseErr := parseStore.parseGetAuthOIDCStateByStateTokenHash("state-hash-1")
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthOIDCStateByStateTokenHash: %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatal("expected oidc state row to exist")
	}
	if strings.TrimSpace(parseFetchedStateRow.ConsumedAt) != "" {
		parseT.Fatalf("expected initial consumed_at to be empty, got %q", parseFetchedStateRow.ConsumedAt)
	}

	isParseConsumed, parseErr := parseStore.parseConsumeAuthOIDCStateByStateTokenHash("state-hash-1", time.Now().UTC())
	if parseErr != nil {
		parseT.Fatalf("parseConsumeAuthOIDCStateByStateTokenHash(first): %v", parseErr)
	}
	if !isParseConsumed {
		parseT.Fatal("expected first state consume attempt to succeed")
	}
	isParseConsumed, parseErr = parseStore.parseConsumeAuthOIDCStateByStateTokenHash("state-hash-1", time.Now().UTC())
	if parseErr != nil {
		parseT.Fatalf("parseConsumeAuthOIDCStateByStateTokenHash(second): %v", parseErr)
	}
	if isParseConsumed {
		parseT.Fatal("expected second state consume attempt to no-op")
	}

	if _, parseErr = parseStore.parseCreateAuthOIDCState(parseAuthOIDCStateWrite{
		ProviderKey:     "oidc",
		WorkspaceID:     parseWorkspaceID,
		SessionKey:      "session-key-expired",
		StateTokenHash:  "state-hash-expired",
		NonceTokenHash:  "nonce-hash-expired",
		ReturnToURL:     "/app",
		ExpiresAt:       time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339),
		CreatedByUserID: parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuthOIDCState(expired seed): %v", parseErr)
	}
	parseDeletedRows, parseErr := parseStore.parseDeleteExpiredAuthOIDCStates(time.Now().UTC())
	if parseErr != nil {
		parseT.Fatalf("parseDeleteExpiredAuthOIDCStates: %v", parseErr)
	}
	if parseDeletedRows <= 0 {
		parseT.Fatalf("expected at least one expired state row deletion, got %d", parseDeletedRows)
	}
	if _, isParseFound, parseErr = parseStore.parseGetAuthOIDCStateByStateTokenHash("state-hash-expired"); parseErr != nil {
		parseT.Fatalf("parseGetAuthOIDCStateByStateTokenHash(expired): %v", parseErr)
	} else if isParseFound {
		parseT.Fatal("expected expired state row to be deleted")
	}

	if _, parseErr = parseStore.parseCreateAuthOIDCState(parseAuthOIDCStateWrite{
		ProviderKey:     "oidc",
		WorkspaceID:     parseWorkspaceID + 999,
		SessionKey:      "missing-workspace-session",
		StateTokenHash:  "missing-workspace-state",
		NonceTokenHash:  "missing-workspace-nonce",
		ReturnToURL:     "/app",
		ExpiresAt:       time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		CreatedByUserID: parseUser.ID,
	}); !errors.Is(parseErr, errStoreSuperuserScopeMissing) {
		parseT.Fatalf("expected errStoreSuperuserScopeMissing, got %v", parseErr)
	}
}

// TestStoreWorkspaceAuthPolicyLifecycle verifies workspace auth-policy upsert and read behavior.
func TestStoreWorkspaceAuthPolicyLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "workspace-policy-owner@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "workspace-auth-policy")

	parseErr := parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        true,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            false,
		RequiredProviderKey:      "google_oidc",
		IsJITProvisioningAllowed: true,
		IsLocalPasswordQAAllowed: true,
		UpdatedByUserID:          parseUser.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceAuthPolicy(create): %v", parseErr)
	}
	parsePolicyRow, isParseFound, parseErr := parseStore.parseGetWorkspaceAuthPolicyByWorkspace(parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceAuthPolicyByWorkspace(create): %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatal("expected workspace auth policy row to exist")
	}
	if !parsePolicyRow.IsPasswordAllowed || !parsePolicyRow.IsExternalLoginAllowed || parsePolicyRow.IsSSORequired || parsePolicyRow.RequiredProviderKey != "google_oidc" {
		parseT.Fatalf("unexpected workspace auth policy row after create: %+v", parsePolicyRow)
	}

	parseErr = parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        false,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            true,
		RequiredProviderKey:      "oidc",
		IsJITProvisioningAllowed: false,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceAuthPolicy(update): %v", parseErr)
	}
	parsePolicyRow, isParseFound, parseErr = parseStore.parseGetWorkspaceAuthPolicyByWorkspace(parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceAuthPolicyByWorkspace(update): %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatal("expected workspace auth policy row after update")
	}
	if parsePolicyRow.IsPasswordAllowed || !parsePolicyRow.IsExternalLoginAllowed || !parsePolicyRow.IsSSORequired || parsePolicyRow.RequiredProviderKey != "oidc" {
		parseT.Fatalf("unexpected workspace auth policy row after update: %+v", parsePolicyRow)
	}

	if _, isParseFound, parseErr = parseStore.parseGetWorkspaceAuthPolicyByWorkspace(999999); parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceAuthPolicyByWorkspace(missing): %v", parseErr)
	} else if isParseFound {
		parseT.Fatal("expected missing workspace auth policy lookup to return not found")
	}

	parseErr = parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        false,
		IsExternalLoginAllowed:   false,
		IsSSORequired:            true,
		RequiredProviderKey:      "google_oidc",
		IsJITProvisioningAllowed: true,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "required provider requires external login allowed") {
		parseT.Fatalf("expected required-provider validation error, got %v", parseErr)
	}

	parseErr = parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        true,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            false,
		RequiredProviderKey:      "github_oidc",
		IsJITProvisioningAllowed: false,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "required provider key is invalid") {
		parseT.Fatalf("expected invalid required-provider validation error, got %v", parseErr)
	}

	parseErr = parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID + 777,
		IsPasswordAllowed:        true,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            false,
		RequiredProviderKey:      "",
		IsJITProvisioningAllowed: false,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	})
	if !errors.Is(parseErr, errStoreSuperuserScopeMissing) {
		parseT.Fatalf("expected errStoreSuperuserScopeMissing for missing workspace, got %v", parseErr)
	}
}

// BenchmarkParseNormalizeAuthProviderType measures canonical provider-type normalization costs.
func BenchmarkParseNormalizeAuthProviderType(parseB *testing.B) {
	for parseIdx := 0; parseIdx < parseB.N; parseIdx++ {
		_ = parseNormalizeAuthProviderType("oidc", "google_oidc")
		_ = parseNormalizeAuthProviderType("invalid", "saml")
		_ = parseNormalizeAuthProviderType("", "oidc")
	}
}
