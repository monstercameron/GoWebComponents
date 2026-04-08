package app

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parsePrivilegedRoleSuperuser = "superuser"
const parsePrivilegedRoleWorkspaceAdmin = "workspace_admin"

type parsePrivilegedAuthPolicy struct {
	IsParseSuperuserExternalLoginAllowed      bool
	IsParseWorkspaceAdminExternalLoginAllowed bool
	IsParsePasswordBreakglassRequired         bool
}

type parsePrivilegedAuthAttempt struct {
	ParseRoleKey    string
	ParseAuthMethod string
}

type parsePrivilegedAuthDecision struct {
	IsParseAllowed bool
	ParseReason    string
}

// parseBuildDefaultPrivilegedAuthPolicy returns the canonical privileged-auth policy defaults.
func parseBuildDefaultPrivilegedAuthPolicy() parsePrivilegedAuthPolicy {
	return parsePrivilegedAuthPolicy{
		IsParseSuperuserExternalLoginAllowed:      true,
		IsParseWorkspaceAdminExternalLoginAllowed: true,
		IsParsePasswordBreakglassRequired:         true,
	}
}

// parseAuthorizePrivilegedLoginMethod enforces privileged account login-method policy for superuser and workspace-admin roles.
func parseAuthorizePrivilegedLoginMethod(parsePolicy parsePrivilegedAuthPolicy, parseAttempt parsePrivilegedAuthAttempt) (parsePrivilegedAuthDecision, error) {
	parseRoleKey := parseNormalizePrivilegedRole(parseAttempt.ParseRoleKey)
	if parseRoleKey == "" {
		return parsePrivilegedAuthDecision{}, status.Error(codes.InvalidArgument, "privileged role key is required")
	}
	parseAuthMethod := parseNormalizePrivilegedAuthMethod(parseAttempt.ParseAuthMethod)

	switch parseRoleKey {
	case parsePrivilegedRoleSuperuser:
		if parseAuthMethod == parseWorkspaceAuthMethodPassword {
			return parsePrivilegedAuthDecision{IsParseAllowed: true, ParseReason: "superuser_password_allowed"}, nil
		}
		if !parsePolicy.IsParseSuperuserExternalLoginAllowed {
			return parsePrivilegedAuthDecision{IsParseAllowed: false, ParseReason: "superuser_external_login_blocked"}, status.Error(codes.PermissionDenied, "superuser policy blocks external login")
		}
		return parsePrivilegedAuthDecision{IsParseAllowed: true, ParseReason: "superuser_external_login_allowed"}, nil
	case parsePrivilegedRoleWorkspaceAdmin:
		if parseAuthMethod == parseWorkspaceAuthMethodPassword {
			return parsePrivilegedAuthDecision{IsParseAllowed: true, ParseReason: "workspace_admin_password_allowed"}, nil
		}
		if !parsePolicy.IsParseWorkspaceAdminExternalLoginAllowed {
			return parsePrivilegedAuthDecision{IsParseAllowed: false, ParseReason: "workspace_admin_external_login_blocked"}, status.Error(codes.PermissionDenied, "workspace-admin policy blocks external login")
		}
		return parsePrivilegedAuthDecision{IsParseAllowed: true, ParseReason: "workspace_admin_external_login_allowed"}, nil
	default:
		return parsePrivilegedAuthDecision{}, status.Error(codes.InvalidArgument, "unsupported privileged role")
	}
}

// parseAuthorizePrivilegedMutationSession enforces fresh-session and break-glass policy for sensitive privileged mutations.
func parseAuthorizePrivilegedMutationSession(parsePolicy parsePrivilegedAuthPolicy, parseAttempt parsePrivilegedAuthAttempt, isParseFreshSession bool) (parsePrivilegedAuthDecision, error) {
	parseDecision, parseErr := parseAuthorizePrivilegedLoginMethod(parsePolicy, parseAttempt)
	if parseErr != nil {
		return parseDecision, parseErr
	}
	if !isParseFreshSession {
		return parsePrivilegedAuthDecision{IsParseAllowed: false, ParseReason: "privileged_session_stale"}, status.Error(codes.Unauthenticated, "privileged re-authentication required")
	}
	parseAuthMethod := parseNormalizePrivilegedAuthMethod(parseAttempt.ParseAuthMethod)
	if parsePolicy.IsParsePasswordBreakglassRequired && parseAuthMethod != parseWorkspaceAuthMethodPassword {
		return parsePrivilegedAuthDecision{IsParseAllowed: false, ParseReason: "privileged_password_breakglass_required"}, status.Error(codes.Unauthenticated, "privileged mutation requires local password re-authentication")
	}
	return parsePrivilegedAuthDecision{IsParseAllowed: true, ParseReason: "privileged_mutation_session_allowed"}, nil
}

// parseNormalizePrivilegedRole normalizes one privileged role key into one canonical value.
func parseNormalizePrivilegedRole(parseRoleKey string) string {
	switch strings.TrimSpace(strings.ToLower(parseRoleKey)) {
	case parsePrivilegedRoleSuperuser, parsePrivilegedRoleWorkspaceAdmin:
		return strings.TrimSpace(strings.ToLower(parseRoleKey))
	default:
		return ""
	}
}

// parseNormalizePrivilegedAuthMethod normalizes one privileged-auth method and falls back to password for legacy sessions.
func parseNormalizePrivilegedAuthMethod(parseAuthMethod string) string {
	parseAuthMethod = parseNormalizeWorkspaceAuthMethod(parseAuthMethod)
	if parseAuthMethod == "" {
		return parseWorkspaceAuthMethodPassword
	}
	return parseAuthMethod
}
