package app

import (
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseWorkspaceProvisioningSourceInvite = "invite"
const parseWorkspaceProvisioningSourceDomainJIT = "domain_jit"
const parseWorkspaceProvisioningSourceExplicitJIT = "explicit_jit"

type parseWorkspaceProvisioningAssignmentRequest struct {
	ParseProviderKey               string
	ParseEmail                     string
	IsParseEmailVerified           bool
	ParseRequestedWorkspaceID      int64
	IsParseJITProvisioningAllowed  bool
	ParseInviteWorkspaceID         int64
	ParseInviteRoleKey             string
	ParseInviteEmail               string
	IsParseInvitePending           bool
	IsParseInviteExpired           bool
	ParseDomainMatchedWorkspaceIDs []int64
}

type parseWorkspaceProvisioningAssignmentDecision struct {
	IsParseAllowed      bool
	ParseReason         string
	ParseWorkspaceID    int64
	ParseMembershipRole string
	ParseAssignmentFrom string
}

// parseAuthorizeWorkspaceProvisioningAssignment enforces invite/domain/JIT workspace-assignment policy for external sign-in.
func parseAuthorizeWorkspaceProvisioningAssignment(parseReq parseWorkspaceProvisioningAssignmentRequest) (parseWorkspaceProvisioningAssignmentDecision, error) {
	if parseNormalizeExternalIdentityProviderKey(parseReq.ParseProviderKey) == "" {
		return parseWorkspaceProvisioningAssignmentDecision{}, status.Error(codes.InvalidArgument, "workspace assignment provider key is required")
	}
	parseEmail := parseNormalizeAuthEmail(parseReq.ParseEmail)
	if parseEmail == "" {
		return parseWorkspaceProvisioningAssignmentDecision{}, status.Error(codes.InvalidArgument, "workspace assignment email is required")
	}
	if !parseReq.IsParseEmailVerified {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "email_unverified"}, status.Error(codes.PermissionDenied, "workspace assignment requires verified email")
	}
	parseRequestedWorkspaceID := parseReq.ParseRequestedWorkspaceID
	parseInviteWorkspaceID := parseReq.ParseInviteWorkspaceID
	if parseInviteWorkspaceID > 0 {
		return parseAuthorizeWorkspaceProvisioningInviteAssignment(parseReq, parseEmail, parseRequestedWorkspaceID, parseInviteWorkspaceID)
	}
	parseDomainWorkspaceIDs := parseNormalizeWorkspaceProvisioningWorkspaceIDs(parseReq.ParseDomainMatchedWorkspaceIDs)
	if len(parseDomainWorkspaceIDs) > 0 {
		return parseAuthorizeWorkspaceProvisioningDomainAssignment(parseReq, parseRequestedWorkspaceID, parseDomainWorkspaceIDs)
	}
	if !parseReq.IsParseJITProvisioningAllowed {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "jit_provisioning_disabled"}, status.Error(codes.PermissionDenied, "workspace assignment requires invite or allowed domain")
	}
	if parseRequestedWorkspaceID <= 0 {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "jit_workspace_required"}, status.Error(codes.PermissionDenied, "workspace assignment target is required when no invite/domain match exists")
	}
	return parseWorkspaceProvisioningAssignmentDecision{
		IsParseAllowed:      true,
		ParseReason:         "explicit_jit_allowed",
		ParseWorkspaceID:    parseRequestedWorkspaceID,
		ParseMembershipRole: "member",
		ParseAssignmentFrom: parseWorkspaceProvisioningSourceExplicitJIT,
	}, nil
}

// parseAuthorizeWorkspaceProvisioningInviteAssignment enforces invite-based workspace assignment boundaries.
func parseAuthorizeWorkspaceProvisioningInviteAssignment(parseReq parseWorkspaceProvisioningAssignmentRequest, parseEmail string, parseRequestedWorkspaceID int64, parseInviteWorkspaceID int64) (parseWorkspaceProvisioningAssignmentDecision, error) {
	if !parseReq.IsParseInvitePending || parseReq.IsParseInviteExpired {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "invite_invalid_state"}, status.Error(codes.PermissionDenied, "workspace invitation is not active")
	}
	parseInviteEmail := parseNormalizeAuthEmail(parseReq.ParseInviteEmail)
	if parseInviteEmail == "" || parseInviteEmail != parseEmail {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "invite_email_mismatch"}, status.Error(codes.PermissionDenied, "workspace invitation email does not match authenticated identity")
	}
	if parseRequestedWorkspaceID > 0 && parseRequestedWorkspaceID != parseInviteWorkspaceID {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "invite_workspace_mismatch"}, status.Error(codes.PermissionDenied, "workspace invitation target mismatch")
	}
	parseRoleKey := strings.TrimSpace(strings.ToLower(parseReq.ParseInviteRoleKey))
	if parseRoleKey == "" {
		parseRoleKey = "member"
	}
	return parseWorkspaceProvisioningAssignmentDecision{
		IsParseAllowed:      true,
		ParseReason:         "invite_assignment_allowed",
		ParseWorkspaceID:    parseInviteWorkspaceID,
		ParseMembershipRole: parseRoleKey,
		ParseAssignmentFrom: parseWorkspaceProvisioningSourceInvite,
	}, nil
}

// parseAuthorizeWorkspaceProvisioningDomainAssignment enforces domain-match workspace assignment and ambiguity rules.
func parseAuthorizeWorkspaceProvisioningDomainAssignment(parseReq parseWorkspaceProvisioningAssignmentRequest, parseRequestedWorkspaceID int64, parseDomainWorkspaceIDs []int64) (parseWorkspaceProvisioningAssignmentDecision, error) {
	if parseRequestedWorkspaceID > 0 && !slices.Contains(parseDomainWorkspaceIDs, parseRequestedWorkspaceID) {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "requested_workspace_not_domain_matched"}, status.Error(codes.PermissionDenied, "requested workspace is not allowed by domain assignment")
	}
	if parseRequestedWorkspaceID == 0 && len(parseDomainWorkspaceIDs) > 1 {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "domain_workspace_ambiguous"}, status.Error(codes.PermissionDenied, "workspace assignment is ambiguous for this domain")
	}
	if !parseReq.IsParseJITProvisioningAllowed {
		return parseWorkspaceProvisioningAssignmentDecision{IsParseAllowed: false, ParseReason: "domain_jit_disabled"}, status.Error(codes.PermissionDenied, "workspace domain assignment requires explicit provisioning policy")
	}
	parseTargetWorkspaceID := parseRequestedWorkspaceID
	if parseTargetWorkspaceID == 0 {
		parseTargetWorkspaceID = parseDomainWorkspaceIDs[0]
	}
	return parseWorkspaceProvisioningAssignmentDecision{
		IsParseAllowed:      true,
		ParseReason:         "domain_assignment_allowed",
		ParseWorkspaceID:    parseTargetWorkspaceID,
		ParseMembershipRole: "member",
		ParseAssignmentFrom: parseWorkspaceProvisioningSourceDomainJIT,
	}, nil
}

// parseNormalizeWorkspaceProvisioningWorkspaceIDs normalizes and deduplicates workspace ids for domain-assignment checks.
func parseNormalizeWorkspaceProvisioningWorkspaceIDs(parseWorkspaceIDs []int64) []int64 {
	parseNormalizedWorkspaceIDs := make([]int64, 0, len(parseWorkspaceIDs))
	for _, parseWorkspaceID := range parseWorkspaceIDs {
		if parseWorkspaceID <= 0 {
			continue
		}
		if slices.Contains(parseNormalizedWorkspaceIDs, parseWorkspaceID) {
			continue
		}
		parseNormalizedWorkspaceIDs = append(parseNormalizedWorkspaceIDs, parseWorkspaceID)
	}
	return parseNormalizedWorkspaceIDs
}
