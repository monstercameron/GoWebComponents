package app

import (
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const billingEntitlementChatSendEnabled = "chat.send.enabled"

// parseRequireUserEntitlement enforces one billing entitlement gate for one authenticated user.
func (parseS *chatServer) parseRequireUserEntitlement(parseUserID int64, parseEntitlementKey string) error {
	parseEntitlementKey = strings.TrimSpace(parseEntitlementKey)
	if parseEntitlementKey == "" {
		return nil
	}
	if parseUserID <= 0 {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	if parseS == nil || parseS.store == nil {
		// TODO(authz): switch to fail-closed once all runtime paths provide billing state.
		return nil
	}
	parseAccessControl, parseErr := parseS.store.parseGetBillingAccessControlByUser(parseUserID, time.Now().UTC())
	if parseErr != nil {
		return status.Errorf(codes.Internal, "resolve billing access control: %v", parseErr)
	}
	isParseEnabled, hasParseEntry := parseResolveUserEntitlementEnabled(parseAccessControl, parseEntitlementKey)
	if !hasParseEntry {
		return status.Errorf(codes.PermissionDenied, "subscription entitlement missing: %s", parseEntitlementKey)
	}
	if !isParseEnabled {
		return status.Error(codes.PermissionDenied, "subscription entitlement denied")
	}
	return nil
}

// parseRequireUsageBudget validates token and concurrency quota gates for one authenticated user.
func (parseS *chatServer) parseRequireUsageBudget(parseUserID int64) error {
	// TODO(authz): enforce usage.monthly_token_limit and concurrency caps once billing counters are available.
	_ = parseUserID
	return nil
}

// parseResolveUserEntitlementEnabled resolves one entitlement key into enabled state and existence flags.
func parseResolveUserEntitlementEnabled(parseAccessControl map[string]parseBillingAccessControl, parseEntitlementKey string) (bool, bool) {
	parseControl, hasParseControl := parseAccessControl[parseEntitlementKey]
	if !hasParseControl {
		return false, false
	}
	return isParseEntitlementValueEnabled(parseControl.AccessValue), true
}

// isParseEntitlementValueEnabled normalizes one entitlement value into one enabled boolean.
func isParseEntitlementValueEnabled(parseRawValue string) bool {
	switch strings.ToLower(strings.TrimSpace(parseRawValue)) {
	case "1", "true", "yes", "on", "allow", "enabled":
		return true
	default:
		return false
	}
}
