package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAuthorizeWorkspaceProvisioningAssignment verifies invite/domain/JIT workspace assignment boundaries.
func TestAuthorizeWorkspaceProvisioningAssignment(parseT *testing.T) {
	parseDecision, parseErr := parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:      "google_oidc",
		ParseEmail:            "invitee@example.com",
		IsParseEmailVerified:  true,
		ParseInviteWorkspaceID: 77,
		ParseInviteRoleKey:    "admin",
		ParseInviteEmail:      "invitee@example.com",
		IsParseInvitePending:  true,
	})
	if parseErr != nil {
		parseT.Fatalf("invite assignment allow: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseWorkspaceID != 77 || parseDecision.ParseMembershipRole != "admin" || parseDecision.ParseAssignmentFrom != parseWorkspaceProvisioningSourceInvite {
		parseT.Fatalf("unexpected invite decision: %+v", parseDecision)
	}

	if _, parseErr = parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:      "google_oidc",
		ParseEmail:            "invitee@example.com",
		IsParseEmailVerified:  true,
		ParseInviteWorkspaceID: 77,
		ParseInviteEmail:      "other@example.com",
		IsParseInvitePending:  true,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("invite email mismatch status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:      "oidc",
		ParseEmail:            "member@company.com",
		IsParseEmailVerified:  true,
		IsParseJITProvisioningAllowed: true,
		ParseDomainMatchedWorkspaceIDs: []int64{21, 22},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("domain ambiguous status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	parseDecision, parseErr = parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:      "oidc",
		ParseEmail:            "member@company.com",
		IsParseEmailVerified:  true,
		IsParseJITProvisioningAllowed: true,
		ParseDomainMatchedWorkspaceIDs: []int64{21},
	})
	if parseErr != nil {
		parseT.Fatalf("domain assignment allow: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseWorkspaceID != 21 || parseDecision.ParseAssignmentFrom != parseWorkspaceProvisioningSourceDomainJIT {
		parseT.Fatalf("unexpected domain assignment decision: %+v", parseDecision)
	}

	if _, parseErr = parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:      "oidc",
		ParseEmail:            "member@company.com",
		IsParseEmailVerified:  false,
		IsParseJITProvisioningAllowed: true,
		ParseDomainMatchedWorkspaceIDs: []int64{21},
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("unverified email status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:              "saml",
		ParseEmail:                    "new-user@enterprise.com",
		IsParseEmailVerified:          true,
		IsParseJITProvisioningAllowed: false,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("jit disabled status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	parseDecision, parseErr = parseAuthorizeWorkspaceProvisioningAssignment(parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:              "saml",
		ParseEmail:                    "new-user@enterprise.com",
		IsParseEmailVerified:          true,
		IsParseJITProvisioningAllowed: true,
		ParseRequestedWorkspaceID:     901,
	})
	if parseErr != nil {
		parseT.Fatalf("explicit jit assignment allow: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseWorkspaceID != 901 || parseDecision.ParseAssignmentFrom != parseWorkspaceProvisioningSourceExplicitJIT {
		parseT.Fatalf("unexpected explicit jit decision: %+v", parseDecision)
	}
}

// BenchmarkAuthorizeWorkspaceProvisioningAssignment measures workspace-assignment policy overhead for the domain allow path.
func BenchmarkAuthorizeWorkspaceProvisioningAssignment(parseB *testing.B) {
	parseReq := parseWorkspaceProvisioningAssignmentRequest{
		ParseProviderKey:               "oidc",
		ParseEmail:                     "member@company.com",
		IsParseEmailVerified:           true,
		IsParseJITProvisioningAllowed:  true,
		ParseDomainMatchedWorkspaceIDs: []int64{21},
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDecision, parseErr := parseAuthorizeWorkspaceProvisioningAssignment(parseReq)
		if parseErr != nil {
			parseB.Fatalf("parseAuthorizeWorkspaceProvisioningAssignment: %v", parseErr)
		}
		if !parseDecision.IsParseAllowed {
			parseB.Fatalf("unexpected denied decision: %+v", parseDecision)
		}
	}
}
