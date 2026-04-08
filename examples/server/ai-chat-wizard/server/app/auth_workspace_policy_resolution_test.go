package app

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestResolveWorkspaceAuthPolicyDefaults verifies default policy resolution when no workspace auth-policy row exists.
func TestResolveWorkspaceAuthPolicyDefaults(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "workspace-auth-defaults@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "workspace-auth-defaults")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseResolution, parseErr := parseServer.parseResolveWorkspaceAuthPolicy(context.Background(), parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseResolveWorkspaceAuthPolicy(defaults): %v", parseErr)
	}
	if !parseResolution.IsParsePasswordAllowed || parseResolution.IsParseExternalLoginAllowed || parseResolution.IsParseWorkspaceSSORequired || parseResolution.ParseWorkspaceRequiredProviderKey != "" || parseResolution.IsParseJITProvisioningAllowed || parseResolution.ParsePolicySource != "default" {
		parseT.Fatalf("unexpected default workspace auth resolution: %+v", parseResolution)
	}
}

// TestResolveWorkspaceAuthPolicyExplicit verifies explicit workspace auth-policy rows map to the typed resolution contract.
func TestResolveWorkspaceAuthPolicyExplicit(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "workspace-auth-explicit@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "workspace-auth-explicit")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	if parseErr := parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        false,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            true,
		RequiredProviderKey:      parseWorkspaceAuthMethodOIDC,
		IsJITProvisioningAllowed: true,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceAuthPolicy(explicit): %v", parseErr)
	}

	parseResolution, parseErr := parseServer.parseResolveWorkspaceAuthPolicy(context.Background(), parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseResolveWorkspaceAuthPolicy(explicit): %v", parseErr)
	}
	if parseResolution.IsParsePasswordAllowed || !parseResolution.IsParseExternalLoginAllowed || parseResolution.IsParseExternalLoginOptional || !parseResolution.IsParseWorkspaceSSORequired || parseResolution.ParseWorkspaceRequiredProviderKey != parseWorkspaceAuthMethodOIDC || !parseResolution.IsParseJITProvisioningAllowed || parseResolution.ParsePolicySource != "workspace" {
		parseT.Fatalf("unexpected explicit workspace auth resolution: %+v", parseResolution)
	}
}

// TestResolveWorkspaceAuthPolicyInfersRequiredProvider verifies required-provider inference from one enabled workspace SSO config when policy requires SSO.
func TestResolveWorkspaceAuthPolicyInfersRequiredProvider(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "workspace-auth-infer@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "workspace-auth-infer")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	if parseErr := parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        false,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            true,
		RequiredProviderKey:      "",
		IsJITProvisioningAllowed: true,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceAuthPolicy(infer): %v", parseErr)
	}
	if _, parseErr := parseStore.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:         parseWorkspaceID,
		ProviderKey:         "okta",
		ProviderType:        "oidc",
		OIDCIssuerURL:       "https://idp.example.com",
		OIDCClientID:        "relaydesk-client",
		OIDCClientSecretRef: "vault://relaydesk/oidc/client-secret",
		OIDCScopesJSON:      `["openid","email","profile"]`,
		OIDCClaimsJSON:      `{"email":"email","subject":"sub"}`,
		DomainsJSON:         `["example.com"]`,
		IsEnabled:           true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserWorkspaceSSOConfig(oidc): %v", parseErr)
	}

	parseResolution, parseErr := parseServer.parseResolveWorkspaceAuthPolicy(context.Background(), parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseResolveWorkspaceAuthPolicy(infer): %v", parseErr)
	}
	if parseResolution.ParseWorkspaceRequiredProviderKey != parseWorkspaceAuthMethodOIDC {
		parseT.Fatalf("expected inferred required provider %q, got %+v", parseWorkspaceAuthMethodOIDC, parseResolution)
	}
}

// TestResolveWorkspaceAuthPolicyRejectsAmbiguousRequiredProvider verifies ambiguous enabled SSO providers fail closed when required provider is unset.
func TestResolveWorkspaceAuthPolicyRejectsAmbiguousRequiredProvider(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "workspace-auth-ambiguous@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "workspace-auth-ambiguous")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	if parseErr := parseStore.parseUpsertWorkspaceAuthPolicy(parseWorkspaceAuthPolicyWrite{
		WorkspaceID:              parseWorkspaceID,
		IsPasswordAllowed:        false,
		IsExternalLoginAllowed:   true,
		IsSSORequired:            true,
		RequiredProviderKey:      "",
		IsJITProvisioningAllowed: false,
		IsLocalPasswordQAAllowed: false,
		UpdatedByUserID:          parseUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceAuthPolicy(ambiguous): %v", parseErr)
	}
	if _, parseErr := parseStore.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:         parseWorkspaceID,
		ProviderKey:         "oidc",
		ProviderType:        "oidc",
		OIDCIssuerURL:       "https://idp.example.com",
		OIDCClientID:        "relaydesk-client",
		OIDCClientSecretRef: "vault://relaydesk/oidc/client-secret",
		OIDCScopesJSON:      `["openid","email"]`,
		OIDCClaimsJSON:      `{"email":"email","subject":"sub"}`,
		IsEnabled:           true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserWorkspaceSSOConfig(oidc): %v", parseErr)
	}
	if _, parseErr := parseStore.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:        parseWorkspaceID,
		ProviderKey:        "saml",
		ProviderType:       "saml",
		SAMLEntrypoint:     "https://idp.example.com/saml",
		SAMLIssuer:         "relaydesk",
		SAMLCertificatePEM: "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----",
		IsEnabled:          true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserWorkspaceSSOConfig(saml): %v", parseErr)
	}

	if _, parseErr := parseServer.parseResolveWorkspaceAuthPolicy(context.Background(), parseWorkspaceID); status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("expected ambiguous provider resolution to fail with failed precondition, got %v", status.Code(parseErr))
	}
}
