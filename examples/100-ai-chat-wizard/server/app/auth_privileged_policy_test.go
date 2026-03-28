package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAuthorizePrivilegedLoginMethod verifies privileged-role login-method policy boundaries.
func TestAuthorizePrivilegedLoginMethod(parseT *testing.T) {
	parsePolicy := parseBuildDefaultPrivilegedAuthPolicy()
	parseDecision, parseErr := parseAuthorizePrivilegedLoginMethod(parsePolicy, parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleSuperuser,
		ParseAuthMethod: parseWorkspaceAuthMethodGoogleOIDC,
	})
	if parseErr != nil {
		parseT.Fatalf("superuser external login allowed: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseReason != "superuser_external_login_allowed" {
		parseT.Fatalf("unexpected superuser external decision: %+v", parseDecision)
	}

	parsePolicy.IsParseSuperuserExternalLoginAllowed = false
	if _, parseErr = parseAuthorizePrivilegedLoginMethod(parsePolicy, parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleSuperuser,
		ParseAuthMethod: parseWorkspaceAuthMethodGoogleOIDC,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("superuser external blocked status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	parsePolicy = parseBuildDefaultPrivilegedAuthPolicy()
	parsePolicy.IsParseWorkspaceAdminExternalLoginAllowed = false
	if _, parseErr = parseAuthorizePrivilegedLoginMethod(parsePolicy, parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleWorkspaceAdmin,
		ParseAuthMethod: parseWorkspaceAuthMethodOIDC,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin external blocked status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestAuthorizePrivilegedMutationSession verifies fresh-session and break-glass behavior for privileged mutation sessions.
func TestAuthorizePrivilegedMutationSession(parseT *testing.T) {
	parsePolicy := parseBuildDefaultPrivilegedAuthPolicy()
	parseDecision, parseErr := parseAuthorizePrivilegedMutationSession(parsePolicy, parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleSuperuser,
		ParseAuthMethod: parseWorkspaceAuthMethodPassword,
	}, true)
	if parseErr != nil {
		parseT.Fatalf("superuser password mutation session allowed: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseReason != "privileged_mutation_session_allowed" {
		parseT.Fatalf("unexpected password mutation decision: %+v", parseDecision)
	}

	if _, parseErr = parseAuthorizePrivilegedMutationSession(parsePolicy, parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleSuperuser,
		ParseAuthMethod: parseWorkspaceAuthMethodGoogleOIDC,
	}, true); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("external mutation session status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}

	if _, parseErr = parseAuthorizePrivilegedMutationSession(parsePolicy, parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleSuperuser,
		ParseAuthMethod: parseWorkspaceAuthMethodPassword,
	}, false); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("stale mutation session status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
}

// BenchmarkAuthorizePrivilegedMutationSession measures privileged mutation-session policy overhead.
func BenchmarkAuthorizePrivilegedMutationSession(parseB *testing.B) {
	parsePolicy := parseBuildDefaultPrivilegedAuthPolicy()
	parseAttempt := parsePrivilegedAuthAttempt{
		ParseRoleKey:    parsePrivilegedRoleSuperuser,
		ParseAuthMethod: parseWorkspaceAuthMethodPassword,
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDecision, parseErr := parseAuthorizePrivilegedMutationSession(parsePolicy, parseAttempt, true)
		if parseErr != nil {
			parseB.Fatalf("parseAuthorizePrivilegedMutationSession: %v", parseErr)
		}
		if !parseDecision.IsParseAllowed {
			parseB.Fatalf("unexpected denied decision: %+v", parseDecision)
		}
	}
}
