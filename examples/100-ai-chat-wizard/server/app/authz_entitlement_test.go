package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireUserEntitlementDenyByDefault(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "deny-by-default@example.com")

	parseErr := parseServer.parseRequireUserEntitlement(parseUser.ID, billingEntitlementChatSendEnabled)
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected missing entitlement to deny, got code=%v err=%v", status.Code(parseErr), parseErr)
	}
}

func TestRequireUserEntitlementAllowsEffectiveAccess(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "entitled@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")

	if parseErr := parseServer.parseRequireUserEntitlement(parseUser.ID, billingEntitlementChatSendEnabled); parseErr != nil {
		parseT.Fatalf("expected active entitlement to pass, got %v", parseErr)
	}
}
