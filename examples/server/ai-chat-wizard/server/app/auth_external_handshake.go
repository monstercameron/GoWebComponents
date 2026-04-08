package app

import (
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseExternalHandshakeStart struct {
	ParseProviderKey string
	ParseReturnToURL string
	ParseSessionKey  string
}

type parseExternalHandshakeState struct {
	ParseProviderKey     string
	ParseStateToken      string
	ParseNonceToken      string
	ParseReturnToURL     string
	ParseSessionKey      string
	ParseBoundSubject    string
	ParseExpiresAt       time.Time
	ParseConsumedAt      time.Time
	IsParseReplayBlocked bool
}

type parseExternalHandshakeCallback struct {
	ParseProviderKey string
	ParseStateToken  string
	ParseNonceToken  string
	ParseReturnToURL string
	ParseSessionKey  string
	ParseSubject     string
	ParseNow         time.Time
}

type parseExternalHandshakeDecision struct {
	IsParseAllowed   bool
	ParseReason      string
	ParseReturnToURL string
}

// parseAuthorizeExternalHandshakeStart validates one external-auth handshake start request.
func parseAuthorizeExternalHandshakeStart(parseStart parseExternalHandshakeStart) (parseExternalHandshakeDecision, error) {
	parseProviderKey := parseNormalizeExternalIdentityProviderKey(parseStart.ParseProviderKey)
	if parseProviderKey == "" {
		return parseExternalHandshakeDecision{}, status.Error(codes.InvalidArgument, "external auth provider key is required")
	}
	parseSessionKey := strings.TrimSpace(parseStart.ParseSessionKey)
	if parseSessionKey == "" {
		return parseExternalHandshakeDecision{}, status.Error(codes.InvalidArgument, "external auth session key is required")
	}
	parseReturnToURL, parseErr := parseValidateExternalHandshakeReturnToURL(parseStart.ParseReturnToURL)
	if parseErr != nil {
		return parseExternalHandshakeDecision{}, parseErr
	}
	return parseExternalHandshakeDecision{
		IsParseAllowed:   true,
		ParseReason:      "start_allowed",
		ParseReturnToURL: parseReturnToURL,
	}, nil
}

// parseAuthorizeExternalHandshakeCallback enforces callback state, nonce, replay, return-to, and provider-subject binding checks.
func parseAuthorizeExternalHandshakeCallback(parseState parseExternalHandshakeState, parseCallback parseExternalHandshakeCallback) (parseExternalHandshakeDecision, error) {
	parseProviderKey := parseNormalizeExternalIdentityProviderKey(parseCallback.ParseProviderKey)
	if parseProviderKey == "" {
		return parseExternalHandshakeDecision{}, status.Error(codes.InvalidArgument, "external auth callback provider key is required")
	}
	parseStateProviderKey := parseNormalizeExternalIdentityProviderKey(parseState.ParseProviderKey)
	if parseStateProviderKey == "" || parseStateProviderKey != parseProviderKey {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "provider_mismatch"}, status.Error(codes.PermissionDenied, "external auth callback provider mismatch")
	}
	parseStateToken := strings.TrimSpace(parseState.ParseStateToken)
	parseCallbackStateToken := strings.TrimSpace(parseCallback.ParseStateToken)
	if parseStateToken == "" || parseCallbackStateToken == "" || parseStateToken != parseCallbackStateToken {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "state_mismatch"}, status.Error(codes.PermissionDenied, "external auth callback state mismatch")
	}
	parseNonceToken := strings.TrimSpace(parseState.ParseNonceToken)
	parseCallbackNonceToken := strings.TrimSpace(parseCallback.ParseNonceToken)
	if parseNonceToken == "" || parseCallbackNonceToken == "" || parseNonceToken != parseCallbackNonceToken {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "nonce_mismatch"}, status.Error(codes.PermissionDenied, "external auth callback nonce mismatch")
	}
	parseSessionKey := strings.TrimSpace(parseState.ParseSessionKey)
	parseCallbackSessionKey := strings.TrimSpace(parseCallback.ParseSessionKey)
	if parseSessionKey == "" || parseCallbackSessionKey == "" || parseSessionKey != parseCallbackSessionKey {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "session_mismatch"}, status.Error(codes.PermissionDenied, "external auth callback session mismatch")
	}
	parseStateReturnToURL, parseErr := parseValidateExternalHandshakeReturnToURL(parseState.ParseReturnToURL)
	if parseErr != nil {
		return parseExternalHandshakeDecision{}, parseErr
	}
	parseCallbackReturnToURL, parseErr := parseValidateExternalHandshakeReturnToURL(parseCallback.ParseReturnToURL)
	if parseErr != nil {
		return parseExternalHandshakeDecision{}, parseErr
	}
	if parseStateReturnToURL != parseCallbackReturnToURL {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "return_to_mismatch"}, status.Error(codes.PermissionDenied, "external auth callback return-to mismatch")
	}
	if parseState.IsParseReplayBlocked && !parseState.ParseConsumedAt.IsZero() {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "callback_replay_detected"}, status.Error(codes.PermissionDenied, "external auth callback replay detected")
	}
	parseCallbackNow := parseCallback.ParseNow.UTC()
	if parseCallbackNow.IsZero() {
		parseCallbackNow = time.Now().UTC()
	}
	if parseState.ParseExpiresAt.IsZero() || parseCallbackNow.After(parseState.ParseExpiresAt.UTC()) {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "state_expired"}, status.Error(codes.PermissionDenied, "external auth callback state expired")
	}
	parseSubject := strings.TrimSpace(parseCallback.ParseSubject)
	if parseSubject == "" {
		return parseExternalHandshakeDecision{}, status.Error(codes.InvalidArgument, "external auth callback subject is required")
	}
	parseBoundSubject := strings.TrimSpace(parseState.ParseBoundSubject)
	if parseBoundSubject != "" && parseBoundSubject != parseSubject {
		return parseExternalHandshakeDecision{IsParseAllowed: false, ParseReason: "provider_subject_mismatch"}, status.Error(codes.PermissionDenied, "external auth callback provider subject mismatch")
	}
	return parseExternalHandshakeDecision{
		IsParseAllowed:   true,
		ParseReason:      "callback_allowed",
		ParseReturnToURL: parseStateReturnToURL,
	}, nil
}

// parseValidateExternalHandshakeReturnToURL validates one callback return-to URL and blocks open-redirect forms.
func parseValidateExternalHandshakeReturnToURL(parseReturnToURL string) (string, error) {
	parseReturnToURL = strings.TrimSpace(parseReturnToURL)
	if parseReturnToURL == "" {
		return "/", nil
	}
	if !strings.HasPrefix(parseReturnToURL, "/") || strings.HasPrefix(parseReturnToURL, "//") {
		return "", status.Error(codes.InvalidArgument, "external auth return-to must be an absolute app path")
	}
	if strings.Contains(parseReturnToURL, "://") || strings.Contains(parseReturnToURL, "\\") {
		return "", status.Error(codes.InvalidArgument, "external auth return-to contains unsafe redirect syntax")
	}
	return parseReturnToURL, nil
}
