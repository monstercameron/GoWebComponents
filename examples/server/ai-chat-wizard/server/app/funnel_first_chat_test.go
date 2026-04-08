package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestResolveFirstChatFunnelSessionKeyFromRequest verifies request-scoped funnel session resolution for nil, client-identity, and auth-cookie inputs.
func TestResolveFirstChatFunnelSessionKeyFromRequest(parseT *testing.T) {
	if parseSessionKey := parseResolveFirstChatFunnelSessionKeyFromRequest(nil, nil); parseSessionKey != "" {
		parseT.Fatalf("nil request session key = %q, want empty", parseSessionKey)
	}

	parseClientIdentity := "11111111-1111-1111-1111-111111111111"
	parseIdentityRequest := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseIdentityRequest.Header.Set(clientMetadataKey, parseClientIdentity)
	if parseSessionKey := parseResolveFirstChatFunnelSessionKeyFromRequest(parseIdentityRequest, nil); parseSessionKey != parseClientIdentity {
		parseT.Fatalf("client identity fallback session key = %q, want %q", parseSessionKey, parseClientIdentity)
	}

	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("funnel-session@example.com", "password123", "Funnel Session")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}
	_, parseClaims, parseErr := parseAuth.parseTokenWithMetadata(parseToken, parseResolveAuthMetadataFromRequest(parseIdentityRequest))
	if parseErr != nil {
		parseT.Fatalf("parseTokenWithMetadata: %v", parseErr)
	}
	if parseClaims.SessionID == "" {
		parseT.Fatalf("expected issued auth token to contain session id, claims=%+v", parseClaims)
	}

	parseCookieRequest := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseCookieRequest.Header.Set(clientMetadataKey, "22222222-2222-2222-2222-222222222222")
	parseCookieRequest.AddCookie(&http.Cookie{Name: authCookieName, Value: parseToken})
	parseSessionKey := parseResolveFirstChatFunnelSessionKeyFromRequest(parseCookieRequest, parseAuth)
	if parseSessionKey != parseClaims.SessionID {
		parseT.Fatalf("auth-cookie session key = %q, want %q", parseSessionKey, parseClaims.SessionID)
	}
}

// TestTrackFirstChatFunnelStepWithRequest verifies HTTP-path funnel tracking writes one analytics row with request-derived session identity.
func TestTrackFirstChatFunnelStepWithRequest(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("funnel-track@example.com", "password123", "Funnel Track")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUser.ID, "ws-funnel-track")

	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseAuth

	parseClientIdentity := "33333333-3333-3333-3333-333333333333"
	parseRequest := httptest.NewRequest(http.MethodGet, "http://example.com/pricing", nil)
	parseRequest.Header.Set(clientMetadataKey, parseClientIdentity)

	parseServer.parseTrackFirstChatFunnelStepWithRequest(parseRequest, parseUser.ID, parseFirstChatStepLandingViewed, map[string]any{
		"route": "/pricing",
	})

	parseAnalyticsRows, parseErr := parseStore.parseListProductAnalyticsEvents(10)
	if parseErr != nil {
		parseT.Fatalf("parseListProductAnalyticsEvents: %v", parseErr)
	}
	if len(parseAnalyticsRows) == 0 {
		parseT.Fatal("expected one first-chat funnel analytics event row")
	}
	parseAnalyticsRow := parseAnalyticsRows[0]
	if parseAnalyticsRow.UserID != parseUser.ID || parseAnalyticsRow.FunnelKey != parseFirstChatFunnelKey || parseAnalyticsRow.StepKey != parseFirstChatStepLandingViewed {
		parseT.Fatalf("unexpected analytics row shape: %+v", parseAnalyticsRow)
	}
	if parseAnalyticsRow.SessionKey != parseClientIdentity {
		parseT.Fatalf("analytics session key = %q, want %q", parseAnalyticsRow.SessionKey, parseClientIdentity)
	}
}
