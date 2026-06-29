package app

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAuthorizeExternalHandshakeStart verifies provider, session, and return-to checks for handshake start.
func TestAuthorizeExternalHandshakeStart(parseT *testing.T) {
	parseDecision, parseErr := parseAuthorizeExternalHandshakeStart(parseExternalHandshakeStart{
		ParseProviderKey: "google_oidc",
		ParseReturnToURL: "/app/settings?panel=security",
		ParseSessionKey:  "browser-session-1",
	})
	if parseErr != nil {
		parseT.Fatalf("parseAuthorizeExternalHandshakeStart allowed: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseReason != "start_allowed" {
		parseT.Fatalf("unexpected start decision: %+v", parseDecision)
	}

	if _, parseErr = parseAuthorizeExternalHandshakeStart(parseExternalHandshakeStart{
		ParseProviderKey: "google_oidc",
		ParseReturnToURL: "https://evil.example.com/phish",
		ParseSessionKey:  "browser-session-1",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsafe return-to status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestAuthorizeExternalHandshakeCallback verifies callback security checks across replay, binding, and expiry boundaries.
func TestAuthorizeExternalHandshakeCallback(parseT *testing.T) {
	parseNow := time.Now().UTC()
	parseState := parseExternalHandshakeState{
		ParseProviderKey:     "google_oidc",
		ParseStateToken:      "state-123",
		ParseNonceToken:      "nonce-123",
		ParseReturnToURL:     "/app",
		ParseSessionKey:      "session-123",
		ParseBoundSubject:    "subject-123",
		ParseExpiresAt:       parseNow.Add(5 * time.Minute),
		IsParseReplayBlocked: true,
	}
	parseDecision, parseErr := parseAuthorizeExternalHandshakeCallback(parseState, parseExternalHandshakeCallback{
		ParseProviderKey: "google_oidc",
		ParseStateToken:  "state-123",
		ParseNonceToken:  "nonce-123",
		ParseReturnToURL: "/app",
		ParseSessionKey:  "session-123",
		ParseSubject:     "subject-123",
		ParseNow:         parseNow,
	})
	if parseErr != nil {
		parseT.Fatalf("parseAuthorizeExternalHandshakeCallback allowed: %v", parseErr)
	}
	if !parseDecision.IsParseAllowed || parseDecision.ParseReason != "callback_allowed" {
		parseT.Fatalf("unexpected callback decision: %+v", parseDecision)
	}

	parseReplayState := parseState
	parseReplayState.ParseConsumedAt = parseNow
	if _, parseErr = parseAuthorizeExternalHandshakeCallback(parseReplayState, parseExternalHandshakeCallback{
		ParseProviderKey: "google_oidc",
		ParseStateToken:  "state-123",
		ParseNonceToken:  "nonce-123",
		ParseReturnToURL: "/app",
		ParseSessionKey:  "session-123",
		ParseSubject:     "subject-123",
		ParseNow:         parseNow,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("replay callback status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseAuthorizeExternalHandshakeCallback(parseState, parseExternalHandshakeCallback{
		ParseProviderKey: "google_oidc",
		ParseStateToken:  "state-123",
		ParseNonceToken:  "bad-nonce",
		ParseReturnToURL: "/app",
		ParseSessionKey:  "session-123",
		ParseSubject:     "subject-123",
		ParseNow:         parseNow,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("nonce mismatch status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseAuthorizeExternalHandshakeCallback(parseState, parseExternalHandshakeCallback{
		ParseProviderKey: "google_oidc",
		ParseStateToken:  "state-123",
		ParseNonceToken:  "nonce-123",
		ParseReturnToURL: "/app",
		ParseSessionKey:  "session-123",
		ParseSubject:     "different-subject",
		ParseNow:         parseNow,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("provider-subject mismatch status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseAuthorizeExternalHandshakeCallback(parseState, parseExternalHandshakeCallback{
		ParseProviderKey: "google_oidc",
		ParseStateToken:  "state-123",
		ParseNonceToken:  "nonce-123",
		ParseReturnToURL: "/app",
		ParseSessionKey:  "session-123",
		ParseSubject:     "subject-123",
		ParseNow:         parseNow.Add(10 * time.Minute),
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expired state status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// BenchmarkAuthorizeExternalHandshakeCallback measures callback security-check overhead for the allow path.
func BenchmarkAuthorizeExternalHandshakeCallback(parseB *testing.B) {
	parseNow := time.Now().UTC()
	parseState := parseExternalHandshakeState{
		ParseProviderKey:     "google_oidc",
		ParseStateToken:      "state-bench",
		ParseNonceToken:      "nonce-bench",
		ParseReturnToURL:     "/app/dashboard",
		ParseSessionKey:      "session-bench",
		ParseBoundSubject:    "subject-bench",
		ParseExpiresAt:       parseNow.Add(5 * time.Minute),
		IsParseReplayBlocked: true,
	}
	parseCallback := parseExternalHandshakeCallback{
		ParseProviderKey: "google_oidc",
		ParseStateToken:  "state-bench",
		ParseNonceToken:  "nonce-bench",
		ParseReturnToURL: "/app/dashboard",
		ParseSessionKey:  "session-bench",
		ParseSubject:     "subject-bench",
		ParseNow:         parseNow,
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDecision, parseErr := parseAuthorizeExternalHandshakeCallback(parseState, parseCallback)
		if parseErr != nil {
			parseB.Fatalf("parseAuthorizeExternalHandshakeCallback: %v", parseErr)
		}
		if !parseDecision.IsParseAllowed {
			parseB.Fatalf("unexpected denied decision: %+v", parseDecision)
		}
	}
}
