package app

import (
	"context"
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseWorkspaceAuthMethodPassword = "password"
const parseWorkspaceAuthMethodGoogleOIDC = "google_oidc"
const parseWorkspaceAuthMethodOIDC = "oidc"
const parseWorkspaceAuthMethodSAML = "saml"

type parseWorkspaceAuthPolicy struct {
	IsParsePasswordAllowed            bool
	IsParseExternalLoginAllowed       bool
	IsParseWorkspaceSSORequired       bool
	ParseWorkspaceRequiredProviderKey string
	IsParseJITProvisioningAllowed     bool
	IsParseLocalPasswordQAModeAllowed bool
}

type parseWorkspaceAuthAttempt struct {
	ParseLoginMethod           string
	IsParseLocalPasswordQAPath bool
}

type parseWorkspaceAuthDecision struct {
	IsParseAllowed bool
	ParseReason    string
}

type parseWorkspaceAuthPolicyResolution struct {
	IsParsePasswordAllowed            bool
	IsParseExternalLoginAllowed       bool
	IsParseExternalLoginOptional      bool
	IsParseWorkspaceSSORequired       bool
	ParseWorkspaceRequiredProviderKey string
	IsParseJITProvisioningAllowed     bool
	IsParseLocalPasswordQAModeAllowed bool
	ParsePolicySource                 string
}

// parseBuildDefaultWorkspaceAuthPolicy returns one default workspace-login policy when no workspace override row exists.
func parseBuildDefaultWorkspaceAuthPolicy() parseWorkspaceAuthPolicy {
	return parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:            true,
		IsParseExternalLoginAllowed:       false,
		IsParseWorkspaceSSORequired:       false,
		ParseWorkspaceRequiredProviderKey: "",
		IsParseJITProvisioningAllowed:     false,
		IsParseLocalPasswordQAModeAllowed: false,
	}
}

// parseResolveWorkspaceAuthPolicy resolves one typed workspace login policy contract from persisted workspace policy and SSO config rows.
func (parseS *chatServer) parseResolveWorkspaceAuthPolicy(parseCtx context.Context, parseWorkspaceID int64) (parseWorkspaceAuthPolicyResolution, error) {
	_ = parseCtx
	if parseS == nil || parseS.store == nil {
		return parseWorkspaceAuthPolicyResolution{}, status.Error(codes.Unavailable, "workspace auth policy store unavailable")
	}
	parsePolicy := parseBuildDefaultWorkspaceAuthPolicy()
	parsePolicySource := "default"
	if parseWorkspaceID > 0 {
		parsePolicyRow, isParsePolicyFound, parseErr := parseS.store.parseGetWorkspaceAuthPolicyByWorkspace(parseWorkspaceID)
		if parseErr != nil {
			return parseWorkspaceAuthPolicyResolution{}, status.Errorf(codes.Internal, "workspace auth policy lookup failed: %v", parseErr)
		}
		if isParsePolicyFound {
			parsePolicySource = "workspace"
			parsePolicy = parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:            parsePolicyRow.IsPasswordAllowed,
				IsParseExternalLoginAllowed:       parsePolicyRow.IsExternalLoginAllowed,
				IsParseWorkspaceSSORequired:       parsePolicyRow.IsSSORequired,
				ParseWorkspaceRequiredProviderKey: parsePolicyRow.RequiredProviderKey,
				IsParseJITProvisioningAllowed:     parsePolicyRow.IsJITProvisioningAllowed,
				IsParseLocalPasswordQAModeAllowed: parsePolicyRow.IsLocalPasswordQAAllowed,
			}
		}
	}
	parseRequiredProviderKey := parseNormalizeWorkspaceAuthMethod(parsePolicy.ParseWorkspaceRequiredProviderKey)
	if parseRequiredProviderKey == "" && parsePolicy.IsParseWorkspaceSSORequired && parseWorkspaceID > 0 {
		parseResolvedProviderKey, parseErr := parseS.parseResolveWorkspaceRequiredProviderKeyFromSSOConfigs(parseWorkspaceID)
		if parseErr != nil {
			return parseWorkspaceAuthPolicyResolution{}, parseErr
		}
		parseRequiredProviderKey = parseResolvedProviderKey
	}
	parseResolution := parseWorkspaceAuthPolicyResolution{
		IsParsePasswordAllowed:            parsePolicy.IsParsePasswordAllowed,
		IsParseExternalLoginAllowed:       parsePolicy.IsParseExternalLoginAllowed,
		IsParseExternalLoginOptional:      parsePolicy.IsParseExternalLoginAllowed && !parsePolicy.IsParseWorkspaceSSORequired,
		IsParseWorkspaceSSORequired:       parsePolicy.IsParseWorkspaceSSORequired,
		ParseWorkspaceRequiredProviderKey: parseRequiredProviderKey,
		IsParseJITProvisioningAllowed:     parsePolicy.IsParseJITProvisioningAllowed,
		IsParseLocalPasswordQAModeAllowed: parsePolicy.IsParseLocalPasswordQAModeAllowed,
		ParsePolicySource:                 parsePolicySource,
	}
	if _, parseErr := parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:            parseResolution.IsParsePasswordAllowed,
		IsParseExternalLoginAllowed:       parseResolution.IsParseExternalLoginAllowed,
		IsParseWorkspaceSSORequired:       parseResolution.IsParseWorkspaceSSORequired,
		ParseWorkspaceRequiredProviderKey: parseResolution.ParseWorkspaceRequiredProviderKey,
		IsParseJITProvisioningAllowed:     parseResolution.IsParseJITProvisioningAllowed,
		IsParseLocalPasswordQAModeAllowed: parseResolution.IsParseLocalPasswordQAModeAllowed,
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodPassword}); parseErr != nil && status.Code(parseErr) == codes.FailedPrecondition {
		return parseWorkspaceAuthPolicyResolution{}, parseErr
	}
	return parseResolution, nil
}

// parseResolveWorkspaceRequiredProviderKeyFromSSOConfigs resolves one required provider key from enabled workspace SSO configs.
func (parseS *chatServer) parseResolveWorkspaceRequiredProviderKeyFromSSOConfigs(parseWorkspaceID int64) (string, error) {
	parseConfigRows, parseErr := parseS.store.parseListWorkspaceSSOConfigs(1000)
	if parseErr != nil {
		return "", status.Errorf(codes.Internal, "workspace sso config lookup failed: %v", parseErr)
	}
	parseProviderKeys := make([]string, 0, 2)
	for _, parseConfigRow := range parseConfigRows {
		if parseConfigRow.WorkspaceID != parseWorkspaceID || !parseConfigRow.IsEnabled {
			continue
		}
		parseAuthMethod := parseResolveWorkspaceAuthMethodFromProviderConfig(parseConfigRow.ProviderKey, parseConfigRow.ProviderType)
		if parseAuthMethod == "" || slices.Contains(parseProviderKeys, parseAuthMethod) {
			continue
		}
		parseProviderKeys = append(parseProviderKeys, parseAuthMethod)
	}
	switch len(parseProviderKeys) {
	case 1:
		return parseProviderKeys[0], nil
	case 0:
		return "", status.Error(codes.FailedPrecondition, "workspace auth policy requires one enabled sso provider")
	default:
		return "", status.Error(codes.FailedPrecondition, "workspace auth policy requires explicit provider when multiple sso providers are enabled")
	}
}

// parseResolveWorkspaceAuthMethodFromProviderConfig maps one provider-key/type pair into one workspace auth method key.
func parseResolveWorkspaceAuthMethodFromProviderConfig(parseProviderKey, parseProviderType string) string {
	if parseAuthMethod := parseNormalizeWorkspaceAuthMethod(parseProviderKey); parseAuthMethod != "" && parseAuthMethod != parseWorkspaceAuthMethodPassword {
		return parseAuthMethod
	}
	switch strings.TrimSpace(strings.ToLower(parseProviderType)) {
	case parseWorkspaceAuthMethodSAML:
		return parseWorkspaceAuthMethodSAML
	case parseWorkspaceAuthMethodOIDC:
		return parseWorkspaceAuthMethodOIDC
	default:
		return ""
	}
}

// parseAuthorizeWorkspaceLoginMethod resolves one workspace auth-policy decision for one login method attempt.
func parseAuthorizeWorkspaceLoginMethod(parsePolicy parseWorkspaceAuthPolicy, parseAttempt parseWorkspaceAuthAttempt) (parseWorkspaceAuthDecision, error) {
	parseLoginMethod := parseNormalizeWorkspaceAuthMethod(parseAttempt.ParseLoginMethod)
	if parseLoginMethod == "" {
		return parseWorkspaceAuthDecision{}, status.Error(codes.InvalidArgument, "workspace auth login method is required")
	}
	parseRequiredProviderKey := parseNormalizeWorkspaceAuthMethod(parsePolicy.ParseWorkspaceRequiredProviderKey)
	if parsePolicy.IsParseWorkspaceSSORequired && !parsePolicy.IsParseExternalLoginAllowed {
		return parseWorkspaceAuthDecision{}, status.Error(codes.FailedPrecondition, "workspace auth policy invalid: sso required requires external login allowed")
	}
	if parseRequiredProviderKey != "" && parseRequiredProviderKey == parseWorkspaceAuthMethodPassword {
		return parseWorkspaceAuthDecision{}, status.Error(codes.FailedPrecondition, "workspace auth policy invalid: required provider cannot be password")
	}
	if parseRequiredProviderKey != "" && !parsePolicy.IsParseExternalLoginAllowed {
		return parseWorkspaceAuthDecision{}, status.Error(codes.FailedPrecondition, "workspace auth policy invalid: required provider requires external login allowed")
	}

	if parseLoginMethod == parseWorkspaceAuthMethodPassword {
		if parsePolicy.IsParsePasswordAllowed {
			return parseWorkspaceAuthDecision{IsParseAllowed: true, ParseReason: "password_allowed"}, nil
		}
		if parseAttempt.IsParseLocalPasswordQAPath && parsePolicy.IsParseLocalPasswordQAModeAllowed {
			return parseWorkspaceAuthDecision{IsParseAllowed: true, ParseReason: "qa_local_password_path_allowed"}, nil
		}
		if parsePolicy.IsParseWorkspaceSSORequired {
			return parseWorkspaceAuthDecision{IsParseAllowed: false, ParseReason: "workspace_sso_required"}, status.Error(codes.PermissionDenied, "workspace requires sso login")
		}
		return parseWorkspaceAuthDecision{IsParseAllowed: false, ParseReason: "password_blocked"}, status.Error(codes.PermissionDenied, "workspace policy blocks password login")
	}

	if !parsePolicy.IsParseExternalLoginAllowed {
		return parseWorkspaceAuthDecision{IsParseAllowed: false, ParseReason: "external_login_blocked"}, status.Error(codes.PermissionDenied, "workspace policy blocks external login")
	}
	if parseRequiredProviderKey != "" && parseLoginMethod != parseRequiredProviderKey {
		return parseWorkspaceAuthDecision{IsParseAllowed: false, ParseReason: "workspace_required_provider_mismatch"}, status.Errorf(codes.PermissionDenied, "workspace requires external provider %s", parseRequiredProviderKey)
	}
	return parseWorkspaceAuthDecision{IsParseAllowed: true, ParseReason: "external_login_allowed"}, nil
}

// parseNormalizeWorkspaceAuthMethod normalizes one workspace login method key into one supported value.
func parseNormalizeWorkspaceAuthMethod(parseLoginMethod string) string {
	switch strings.TrimSpace(strings.ToLower(parseLoginMethod)) {
	case parseWorkspaceAuthMethodPassword, parseWorkspaceAuthMethodGoogleOIDC, parseWorkspaceAuthMethodOIDC, parseWorkspaceAuthMethodSAML:
		return strings.TrimSpace(strings.ToLower(parseLoginMethod))
	default:
		return ""
	}
}
