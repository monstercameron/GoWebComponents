package app

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseRequireChatRuntimeAccess enforces user-disable and workspace-suspend runtime policy for chat send operations.
func (parseS *chatServer) parseRequireChatRuntimeAccess(parseUserID int64) error {
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		return nil
	}
	isParseDisabled, parseErr := parseS.store.parseIsUserAccessDisabled(parseUserID)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "user access-state lookup failed: %v", parseErr)
	}
	if isParseDisabled {
		return status.Error(codes.PermissionDenied, "user account is disabled")
	}

	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByUser(parseUserID)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "workspace membership lookup failed: %v", parseErr)
	}
	isParseHasActiveMembership := false
	isParseHasOperationalWorkspace := false
	for _, parseMembershipRow := range parseMembershipRows {
		if !parseIsWorkspaceMembershipActive(parseMembershipRow.Status) {
			continue
		}
		isParseHasActiveMembership = true
		parseWorkspaceRow, isParseFound, parseWorkspaceErr := parseS.store.parseGetWorkspaceByID(parseMembershipRow.WorkspaceID)
		if parseWorkspaceErr != nil {
			return status.Errorf(codes.Internal, "workspace status lookup failed: %v", parseWorkspaceErr)
		}
		if !isParseFound {
			continue
		}
		if parseIsWorkspaceOperationalStatus(parseWorkspaceRow.Status) {
			isParseHasOperationalWorkspace = true
			break
		}
	}
	if isParseHasActiveMembership && !isParseHasOperationalWorkspace {
		return status.Error(codes.PermissionDenied, "workspace is suspended")
	}
	return nil
}
