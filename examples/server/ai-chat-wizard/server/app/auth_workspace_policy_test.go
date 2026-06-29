package app

import (
	"strings"
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

// TestAuthorizeWorkspaceSSORegressionMatrix verifies optional-vs-required workspace SSO outcomes and local QA password fallback behavior.
func TestAuthorizeWorkspaceSSORegressionMatrix(parseT *testing.T) {
	parseCases := []struct {
		parseLabel          string
		parsePolicy         parseWorkspaceAuthPolicy
		parseAttempt        parseWorkspaceAuthAttempt
		isParseAllowed      bool
		parseReason         string
		parseErrorCode      codes.Code
		parseExpectedAction string
	}{
		{
			parseLabel: "optional sso allows local password login",
			parsePolicy: parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:      true,
				IsParseExternalLoginAllowed: true,
				IsParseWorkspaceSSORequired: false,
			},
			parseAttempt:        parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodPassword},
			isParseAllowed:      true,
			parseReason:         "password_allowed",
			parseErrorCode:      codes.OK,
			parseExpectedAction: "allow_local_password",
		},
		{
			parseLabel: "optional sso allows external login",
			parsePolicy: parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:      true,
				IsParseExternalLoginAllowed: true,
				IsParseWorkspaceSSORequired: false,
			},
			parseAttempt:        parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodGoogleOIDC},
			isParseAllowed:      true,
			parseReason:         "external_login_allowed",
			parseErrorCode:      codes.OK,
			parseExpectedAction: "allow_external_login",
		},
		{
			parseLabel: "required sso denies password and signals sso redirect path",
			parsePolicy: parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:      false,
				IsParseExternalLoginAllowed: true,
				IsParseWorkspaceSSORequired: true,
			},
			parseAttempt:        parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodPassword},
			isParseAllowed:      false,
			parseReason:         "workspace_sso_required",
			parseErrorCode:      codes.PermissionDenied,
			parseExpectedAction: "redirect_workspace_sso_entry",
		},
		{
			parseLabel: "required sso allows local qa password path when policy enables it",
			parsePolicy: parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:            false,
				IsParseExternalLoginAllowed:       true,
				IsParseWorkspaceSSORequired:       true,
				IsParseLocalPasswordQAModeAllowed: true,
			},
			parseAttempt: parseWorkspaceAuthAttempt{
				ParseLoginMethod:           parseWorkspaceAuthMethodPassword,
				IsParseLocalPasswordQAPath: true,
			},
			isParseAllowed:      true,
			parseReason:         "qa_local_password_path_allowed",
			parseErrorCode:      codes.OK,
			parseExpectedAction: "allow_local_password_qa_path",
		},
		{
			parseLabel: "required sso allows matching required provider",
			parsePolicy: parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:            false,
				IsParseExternalLoginAllowed:       true,
				IsParseWorkspaceSSORequired:       true,
				ParseWorkspaceRequiredProviderKey: parseWorkspaceAuthMethodOIDC,
			},
			parseAttempt:        parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodOIDC},
			isParseAllowed:      true,
			parseReason:         "external_login_allowed",
			parseErrorCode:      codes.OK,
			parseExpectedAction: "allow_external_login",
		},
		{
			parseLabel: "required sso denies mismatched provider",
			parsePolicy: parseWorkspaceAuthPolicy{
				IsParsePasswordAllowed:            false,
				IsParseExternalLoginAllowed:       true,
				IsParseWorkspaceSSORequired:       true,
				ParseWorkspaceRequiredProviderKey: parseWorkspaceAuthMethodOIDC,
			},
			parseAttempt:        parseWorkspaceAuthAttempt{ParseLoginMethod: parseWorkspaceAuthMethodGoogleOIDC},
			isParseAllowed:      false,
			parseReason:         "workspace_required_provider_mismatch",
			parseErrorCode:      codes.PermissionDenied,
			parseExpectedAction: "deny_provider_mismatch",
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.parseLabel, func(parseT *testing.T) {
			parseDecision, parseErr := parseAuthorizeWorkspaceLoginMethod(parseCase.parsePolicy, parseCase.parseAttempt)
			if parseDecision.IsParseAllowed != parseCase.isParseAllowed {
				parseT.Fatalf("allowed=%t want=%t decision=%+v err=%v", parseDecision.IsParseAllowed, parseCase.isParseAllowed, parseDecision, parseErr)
			}
			if strings.TrimSpace(parseDecision.ParseReason) != parseCase.parseReason {
				parseT.Fatalf("reason=%q want=%q decision=%+v err=%v", parseDecision.ParseReason, parseCase.parseReason, parseDecision, parseErr)
			}
			if parseStatusCode := status.Code(parseErr); parseStatusCode != parseCase.parseErrorCode {
				parseT.Fatalf("status=%v want=%v decision=%+v err=%v", parseStatusCode, parseCase.parseErrorCode, parseDecision, parseErr)
			}
			parseAction := "allow_local_password"
			switch parseDecision.ParseReason {
			case "external_login_allowed":
				parseAction = "allow_external_login"
			case "workspace_sso_required":
				parseAction = "redirect_workspace_sso_entry"
			case "workspace_required_provider_mismatch":
				parseAction = "deny_provider_mismatch"
			case "qa_local_password_path_allowed":
				parseAction = "allow_local_password_qa_path"
			}
			if parseAction != parseCase.parseExpectedAction {
				parseT.Fatalf("action=%q want=%q decision=%+v err=%v", parseAction, parseCase.parseExpectedAction, parseDecision, parseErr)
			}
		})
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
