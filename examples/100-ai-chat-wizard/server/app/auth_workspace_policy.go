package app

import (
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
