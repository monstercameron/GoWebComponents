package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAuthorizeWorkspaceLoginMethod verifies workspace auth-policy boundaries for password, external login, and SSO-required states.
func TestAuthorizeWorkspaceLoginMethod(parseT *testing.T) {
	parseAllowPasswordDecision, parseErr := parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:      true,
		IsParseExternalLoginAllowed: true,
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodPassword})
	if parseErr != nil || !parseAllowPasswordDecision.IsParseAllowed {
		parseT.Fatalf("password allowed decision=%+v err=%v", parseAllowPasswordDecision, parseErr)
	}

	parseAllowExternalDecision, parseErr := parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:      false,
		IsParseExternalLoginAllowed: true,
		IsParseWorkspaceSSORequired: true,
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodGoogleOIDC})
	if parseErr != nil || !parseAllowExternalDecision.IsParseAllowed {
		parseT.Fatalf("external allowed decision=%+v err=%v", parseAllowExternalDecision, parseErr)
	}

	if _, parseErr = parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:      false,
		IsParseExternalLoginAllowed: true,
		IsParseWorkspaceSSORequired: true,
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodPassword}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("password blocked by sso-required status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	parseAllowQADecision, parseErr := parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:            false,
		IsParseExternalLoginAllowed:       true,
		IsParseWorkspaceSSORequired:       true,
		IsParseLocalPasswordQAModeAllowed: true,
	}, parseWorkspaceAuthAttempt{
		ParseLoginMethod:           parseWorkspaceAuthMethodPassword,
		IsParseLocalPasswordQAPath: true,
	})
	if parseErr != nil || !parseAllowQADecision.IsParseAllowed || parseAllowQADecision.ParseReason != "qa_local_password_path_allowed" {
		parseT.Fatalf("qa local password path decision=%+v err=%v", parseAllowQADecision, parseErr)
	}

	if _, parseErr = parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:            true,
		IsParseExternalLoginAllowed:       true,
		ParseWorkspaceRequiredProviderKey: parseWorkspaceAuthMethodOIDC,
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodGoogleOIDC}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("required provider mismatch status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestAuthorizeWorkspaceLoginMethodRejectsInvalidPolicy verifies malformed workspace auth policies fail closed.
func TestAuthorizeWorkspaceLoginMethodRejectsInvalidPolicy(parseT *testing.T) {
	if _, parseErr := parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:      false,
		IsParseExternalLoginAllowed: false,
		IsParseWorkspaceSSORequired: true,
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodOIDC}); status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("invalid sso policy status code=%v want=%v", status.Code(parseErr), codes.FailedPrecondition)
	}
	if _, parseErr := parseAuthorizeWorkspaceLoginMethod(parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:            true,
		IsParseExternalLoginAllowed:       true,
		ParseWorkspaceRequiredProviderKey: "password",
	}, parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodPassword}); status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("invalid required provider status code=%v want=%v", status.Code(parseErr), codes.FailedPrecondition)
	}
}

// BenchmarkAuthorizeWorkspaceLoginMethod measures workspace auth-policy decision overhead for common external-login allow paths.
func BenchmarkAuthorizeWorkspaceLoginMethod(parseB *testing.B) {
	parsePolicy := parseWorkspaceAuthPolicy{
		IsParsePasswordAllowed:      false,
		IsParseExternalLoginAllowed: true,
		IsParseWorkspaceSSORequired: true,
	}
	parseAttempt := parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodGoogleOIDC}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDecision, parseErr := parseAuthorizeWorkspaceLoginMethod(parsePolicy, parseAttempt)
		if parseErr != nil {
			parseB.Fatalf("parseAuthorizeWorkspaceLoginMethod: %v", parseErr)
		}
		if !parseDecision.IsParseAllowed {
			parseB.Fatalf("unexpected denied decision: %+v", parseDecision)
		}
	}
}
