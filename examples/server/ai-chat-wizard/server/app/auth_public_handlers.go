package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const parsePublicPasswordResetOutcomeAccepted = "accepted"
const parsePublicPasswordResetOutcomePasswordUpdated = "password_updated"
const parsePublicPasswordResetOutcomeExpiredToken = "expired_token"
const parsePublicPasswordResetOutcomeInvalidToken = "invalid_token"
const parsePublicPasswordResetOutcomeReplayDenied = "replay_denied"
const parsePublicPasswordResetOutcomeWeakPassword = "weak_password"
const parsePublicPasswordResetOutcomeInvalidRequest = "invalid_request"
const parsePublicPasswordResetOutcomeUnavailable = "unavailable"

const parsePublicSignupVerificationOutcomeAccepted = "accepted"
const parsePublicSignupVerificationOutcomeVerified = "verified"
const parsePublicSignupVerificationOutcomeExpiredToken = "expired_token"
const parsePublicSignupVerificationOutcomeReplayDenied = "replay_denied"
const parsePublicSignupVerificationOutcomeInvalidToken = "invalid_token"
const parsePublicSignupVerificationOutcomeInvalidRequest = "invalid_request"
const parsePublicSignupVerificationOutcomeUnavailable = "unavailable"

const parsePublicSignupVerificationRedirectTarget = "/app?auth_verified=1"

type parsePublicPasswordResetRequestBody struct {
	ParseEmail string `json:"email"`
}

type parsePublicSignupVerificationRequestBody struct {
	ParseEmail string `json:"email"`
}

type parsePublicPasswordResetConsumeBody struct {
	ParseToken       string `json:"token"`
	ParsePassword    string `json:"password"`
	ParseNewPassword string `json:"new_password"`
}

type parsePublicSignupVerificationConsumeBody struct {
	ParseToken string `json:"token"`
}

type parsePublicPasswordResetResponse struct {
	IsParseOK       bool   `json:"ok"`
	ParseOutcome    string `json:"outcome"`
	ParseMessage    string `json:"message,omitempty"`
	ParseResetToken string `json:"reset_token,omitempty"`
}

type parsePublicSignupVerificationResponse struct {
	IsParseOK              bool   `json:"ok"`
	ParseOutcome           string `json:"outcome"`
	ParseMessage           string `json:"message,omitempty"`
	ParseVerificationToken string `json:"verification_token,omitempty"`
	ParseRedirectTarget    string `json:"redirect_target,omitempty"`
}

// parseHandlePublicAuthSessionSync serves one same-origin endpoint that mirrors a valid bearer token into the auth cookie.
func (parseS *chatServer) parseHandlePublicAuthSessionSync(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method == http.MethodDelete {
		if parseS != nil && parseS.authManager != nil {
			parseS.authManager.clearAuthCookie(parseW, parseR)
		}
		parseWritePublicNoContent(parseW)
		return
	}
	if parseR.Method != http.MethodPost {
		http.Error(parseW, "use POST to sync auth session", http.StatusMethodNotAllowed)
		return
	}
	if parseS == nil || parseS.authManager == nil {
		http.Error(parseW, "auth unavailable", http.StatusServiceUnavailable)
		return
	}
	_, parseToken, parseOk := parseS.authManager.parseAuthenticatedUserFromAuthorizationRequest(parseR)
	if !parseOk || strings.TrimSpace(parseToken) == "" {
		parseS.authManager.clearAuthCookie(parseW, parseR)
		http.Error(parseW, "authentication required", http.StatusUnauthorized)
		return
	}
	parseS.authManager.setAuthCookie(parseW, parseR, parseToken)
	parseWritePublicNoContent(parseW)
}

// parseHandlePublicPasswordResetRequest serves one public endpoint for password-reset token requests.
func (parseS *chatServer) parseHandlePublicPasswordResetRequest(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseWritePublicPasswordResetResponse(parseW, http.StatusMethodNotAllowed, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Use POST for password-reset requests.",
		})
		return
	}
	if parseS == nil || parseS.authManager == nil {
		parseWritePublicPasswordResetResponse(parseW, http.StatusServiceUnavailable, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeUnavailable,
			ParseMessage: "Password recovery is temporarily unavailable.",
		})
		return
	}
	parseRequestBody, parseErr := parseDecodePublicPasswordResetRequestBody(parseR)
	if parseErr != nil {
		parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Enter a valid email to continue.",
		})
		return
	}
	parseEmail := parseNormalizeAuthEmail(parseRequestBody.ParseEmail)
	if parseEmail == "" {
		parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Enter a valid email to continue.",
		})
		return
	}
	parseRequestMetadata := parseResolveAuthMetadataFromRequest(parseR)
	parseResetToken, parseErr := parseS.authManager.parseBeginPasswordResetToken(parseEmail, parseRequestMetadata.IPAddress)
	if parseErr != nil {
		parseStatusCode := http.StatusBadRequest
		parseOutcome := parsePublicPasswordResetOutcomeInvalidRequest
		parseMessage := "Password recovery request was rejected."
		if errors.Is(parseErr, errPasswordRecoveryUnavailable) {
			parseStatusCode = http.StatusServiceUnavailable
			parseOutcome = parsePublicPasswordResetOutcomeUnavailable
			parseMessage = "Password recovery is temporarily unavailable."
		}
		parseWritePublicPasswordResetResponse(parseW, parseStatusCode, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parseOutcome,
			ParseMessage: parseMessage,
		})
		return
	}
	parseResponse := parsePublicPasswordResetResponse{
		IsParseOK:    true,
		ParseOutcome: parsePublicPasswordResetOutcomeAccepted,
		ParseMessage: "If an account exists, a reset link is ready.",
	}
	if parseS.parseShouldExposePasswordResetTokenForQA() && strings.TrimSpace(parseResetToken) != "" {
		parseResponse.ParseResetToken = strings.TrimSpace(parseResetToken)
	}
	parseWritePublicPasswordResetResponse(parseW, http.StatusOK, parseResponse)
}

// parseHandlePublicPasswordResetConsume serves one public endpoint for reset-token password updates.
func (parseS *chatServer) parseHandlePublicPasswordResetConsume(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseWritePublicPasswordResetResponse(parseW, http.StatusMethodNotAllowed, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Use POST for password updates.",
		})
		return
	}
	if parseS == nil || parseS.authManager == nil {
		parseWritePublicPasswordResetResponse(parseW, http.StatusServiceUnavailable, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeUnavailable,
			ParseMessage: "Password recovery is temporarily unavailable.",
		})
		return
	}
	parseRequestBody, parseErr := parseDecodePublicPasswordResetConsumeBody(parseR)
	if parseErr != nil {
		parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Enter a reset token and new password.",
		})
		return
	}
	parseToken := strings.TrimSpace(parseRequestBody.ParseToken)
	parsePassword := strings.TrimSpace(parseRequestBody.ParsePassword)
	if parsePassword == "" {
		parsePassword = strings.TrimSpace(parseRequestBody.ParseNewPassword)
	}
	if parseToken == "" || parsePassword == "" {
		parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Enter a reset token and new password.",
		})
		return
	}
	parseErr = parseS.authManager.parseCompletePasswordResetWithToken(parseToken, parsePassword)
	if parseErr != nil {
		if errors.Is(parseErr, errPasswordRecoveryUnavailable) {
			parseWritePublicPasswordResetResponse(parseW, http.StatusServiceUnavailable, parsePublicPasswordResetResponse{
				IsParseOK:    false,
				ParseOutcome: parsePublicPasswordResetOutcomeUnavailable,
				ParseMessage: "Password recovery is temporarily unavailable.",
			})
			return
		}
		if errors.Is(parseErr, errInvalidCredentials) {
			parseOutcome := parseS.parseResolvePublicPasswordResetConsumeOutcome(parseToken)
			parseMessage := "Reset token is invalid. Request a new link and try again."
			if parseOutcome == parsePublicPasswordResetOutcomeExpiredToken {
				parseMessage = "Reset token expired. Request a new link and try again."
			}
			if parseOutcome == parsePublicPasswordResetOutcomeReplayDenied {
				parseMessage = "Reset token was already used. Request a new link and try again."
			}
			parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
				IsParseOK:    false,
				ParseOutcome: parseOutcome,
				ParseMessage: parseMessage,
			})
			return
		}
		if strings.Contains(strings.ToLower(parseErr.Error()), "at least 8 characters") {
			parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
				IsParseOK:    false,
				ParseOutcome: parsePublicPasswordResetOutcomeWeakPassword,
				ParseMessage: "Password must be at least 8 characters.",
			})
			return
		}
		parseWritePublicPasswordResetResponse(parseW, http.StatusBadRequest, parsePublicPasswordResetResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicPasswordResetOutcomeInvalidRequest,
			ParseMessage: "Password update failed.",
		})
		return
	}
	parseWritePublicPasswordResetResponse(parseW, http.StatusOK, parsePublicPasswordResetResponse{
		IsParseOK:    true,
		ParseOutcome: parsePublicPasswordResetOutcomePasswordUpdated,
		ParseMessage: "Password updated successfully.",
	})
}

// parseHandlePublicSignupVerificationResend serves one public endpoint for signup-verification resend requests.
func (parseS *chatServer) parseHandlePublicSignupVerificationResend(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusMethodNotAllowed, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Use POST for signup-verification resend.",
		})
		return
	}
	if parseS == nil || parseS.authManager == nil {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusServiceUnavailable, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeUnavailable,
			ParseMessage: "Signup verification is temporarily unavailable.",
		})
		return
	}
	parseRequestBody, parseErr := parseDecodePublicSignupVerificationRequestBody(parseR)
	if parseErr != nil {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusBadRequest, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Enter a valid email to continue.",
		})
		return
	}
	parseEmail := parseNormalizeAuthEmail(parseRequestBody.ParseEmail)
	if parseEmail == "" {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusBadRequest, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Enter a valid email to continue.",
		})
		return
	}
	parseVerificationToken, parseErr := parseS.authManager.parseResendSignupVerificationToken(parseEmail)
	if parseErr != nil {
		parseStatusCode := http.StatusBadRequest
		parseOutcome := parsePublicSignupVerificationOutcomeInvalidRequest
		parseMessage := "Signup verification request was rejected."
		if errors.Is(parseErr, errSignupVerificationUnavailable) {
			parseStatusCode = http.StatusServiceUnavailable
			parseOutcome = parsePublicSignupVerificationOutcomeUnavailable
			parseMessage = "Signup verification is temporarily unavailable."
		}
		parseWritePublicSignupVerificationResponse(parseW, parseStatusCode, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parseOutcome,
			ParseMessage: parseMessage,
		})
		return
	}
	parseResponse := parsePublicSignupVerificationResponse{
		IsParseOK:    true,
		ParseOutcome: parsePublicSignupVerificationOutcomeAccepted,
		ParseMessage: "If an account exists and needs verification, a verification link is ready.",
	}
	if parseS.parseShouldExposePasswordResetTokenForQA() && strings.TrimSpace(parseVerificationToken) != "" {
		parseResponse.ParseVerificationToken = strings.TrimSpace(parseVerificationToken)
	}
	parseWritePublicSignupVerificationResponse(parseW, http.StatusOK, parseResponse)
}

// parseHandlePublicSignupVerificationConsume serves one public endpoint for signup-verification token consumption.
func (parseS *chatServer) parseHandlePublicSignupVerificationConsume(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusMethodNotAllowed, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Use POST for signup-verification completion.",
		})
		return
	}
	if parseS == nil || parseS.authManager == nil {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusServiceUnavailable, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeUnavailable,
			ParseMessage: "Signup verification is temporarily unavailable.",
		})
		return
	}
	parseRequestBody, parseErr := parseDecodePublicSignupVerificationConsumeBody(parseR)
	if parseErr != nil {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusBadRequest, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Enter a verification token to continue.",
		})
		return
	}
	parseVerificationToken := strings.TrimSpace(parseRequestBody.ParseToken)
	if parseVerificationToken == "" {
		parseWritePublicSignupVerificationResponse(parseW, http.StatusBadRequest, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Enter a verification token to continue.",
		})
		return
	}
	parseErr = parseS.authManager.parseCompleteSignupVerificationWithToken(parseVerificationToken)
	if parseErr != nil {
		if errors.Is(parseErr, errSignupVerificationUnavailable) {
			parseWritePublicSignupVerificationResponse(parseW, http.StatusServiceUnavailable, parsePublicSignupVerificationResponse{
				IsParseOK:    false,
				ParseOutcome: parsePublicSignupVerificationOutcomeUnavailable,
				ParseMessage: "Signup verification is temporarily unavailable.",
			})
			return
		}
		if errors.Is(parseErr, errInvalidCredentials) {
			parseOutcome := parseS.parseResolvePublicSignupVerificationConsumeOutcome(parseVerificationToken)
			parseMessage := "Verification token is invalid. Request a new link and try again."
			if parseOutcome == parsePublicSignupVerificationOutcomeExpiredToken {
				parseMessage = "Verification token expired. Request a new link and try again."
			}
			if parseOutcome == parsePublicSignupVerificationOutcomeReplayDenied {
				parseMessage = "Verification token was already used. Request a new link and try again."
			}
			parseWritePublicSignupVerificationResponse(parseW, http.StatusBadRequest, parsePublicSignupVerificationResponse{
				IsParseOK:    false,
				ParseOutcome: parseOutcome,
				ParseMessage: parseMessage,
			})
			return
		}
		parseWritePublicSignupVerificationResponse(parseW, http.StatusBadRequest, parsePublicSignupVerificationResponse{
			IsParseOK:    false,
			ParseOutcome: parsePublicSignupVerificationOutcomeInvalidRequest,
			ParseMessage: "Signup verification failed.",
		})
		return
	}
	parseWritePublicSignupVerificationResponse(parseW, http.StatusOK, parsePublicSignupVerificationResponse{
		IsParseOK:           true,
		ParseOutcome:        parsePublicSignupVerificationOutcomeVerified,
		ParseMessage:        "Email verified successfully.",
		ParseRedirectTarget: parsePublicSignupVerificationRedirectTarget,
	})
}

// parseShouldExposePasswordResetTokenForQA reports whether reset tokens can be surfaced for local QA.
func (parseS *chatServer) parseShouldExposePasswordResetTokenForQA() bool {
	if parseS == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(parseS.runtimeEnvironment)) {
	case "", "development", "dev", "test", "local":
		return true
	default:
		return false
	}
}

// parseResolvePublicSignupVerificationConsumeOutcome maps one signup-verification consume outcome into one public API outcome value.
func (parseS *chatServer) parseResolvePublicSignupVerificationConsumeOutcome(parseVerificationToken string) string {
	if parseS == nil || parseS.authManager == nil {
		return parsePublicSignupVerificationOutcomeInvalidToken
	}
	parseTokenHash := parseBuildAuthFlowTokenHash(parseVerificationToken)
	parseOutcome := parseS.authManager.parseResolveSignupVerificationConsumeOutcome(parseTokenHash, time.Now().UTC())
	switch parseOutcome {
	case parseSignupVerificationAuditOutcomeExpired:
		return parsePublicSignupVerificationOutcomeExpiredToken
	case parseSignupVerificationAuditOutcomeReplayDenied:
		return parsePublicSignupVerificationOutcomeReplayDenied
	default:
		return parsePublicSignupVerificationOutcomeInvalidToken
	}
}

// parseResolvePublicPasswordResetConsumeOutcome maps one auth consume outcome into one public API outcome value.
func (parseS *chatServer) parseResolvePublicPasswordResetConsumeOutcome(parseResetToken string) string {
	if parseS == nil || parseS.authManager == nil {
		return parsePublicPasswordResetOutcomeInvalidToken
	}
	parseTokenHash := parseBuildAuthFlowTokenHash(parseResetToken)
	parseOutcome := parseS.authManager.parseResolvePasswordResetConsumeOutcome(parseTokenHash, time.Now().UTC())
	switch parseOutcome {
	case parsePasswordRecoveryAuditOutcomeExpired:
		return parsePublicPasswordResetOutcomeExpiredToken
	case parsePasswordRecoveryAuditOutcomeReplayDenied:
		return parsePublicPasswordResetOutcomeReplayDenied
	default:
		return parsePublicPasswordResetOutcomeInvalidToken
	}
}

// parseDecodePublicPasswordResetRequestBody decodes one password-reset request payload.
func parseDecodePublicPasswordResetRequestBody(parseR *http.Request) (parsePublicPasswordResetRequestBody, error) {
	var parseRequestBody parsePublicPasswordResetRequestBody
	parseDecoder := json.NewDecoder(parseR.Body)
	parseDecoder.DisallowUnknownFields()
	if parseErr := parseDecoder.Decode(&parseRequestBody); parseErr != nil {
		return parsePublicPasswordResetRequestBody{}, parseErr
	}
	return parseRequestBody, nil
}

// parseDecodePublicSignupVerificationRequestBody decodes one signup-verification resend request payload.
func parseDecodePublicSignupVerificationRequestBody(parseR *http.Request) (parsePublicSignupVerificationRequestBody, error) {
	var parseRequestBody parsePublicSignupVerificationRequestBody
	parseDecoder := json.NewDecoder(parseR.Body)
	parseDecoder.DisallowUnknownFields()
	if parseErr := parseDecoder.Decode(&parseRequestBody); parseErr != nil {
		return parsePublicSignupVerificationRequestBody{}, parseErr
	}
	return parseRequestBody, nil
}

// parseDecodePublicSignupVerificationConsumeBody decodes one signup-verification consume payload.
func parseDecodePublicSignupVerificationConsumeBody(parseR *http.Request) (parsePublicSignupVerificationConsumeBody, error) {
	var parseRequestBody parsePublicSignupVerificationConsumeBody
	parseDecoder := json.NewDecoder(parseR.Body)
	parseDecoder.DisallowUnknownFields()
	if parseErr := parseDecoder.Decode(&parseRequestBody); parseErr != nil {
		return parsePublicSignupVerificationConsumeBody{}, parseErr
	}
	return parseRequestBody, nil
}

// parseDecodePublicPasswordResetConsumeBody decodes one password-reset consume payload.
func parseDecodePublicPasswordResetConsumeBody(parseR *http.Request) (parsePublicPasswordResetConsumeBody, error) {
	var parseRequestBody parsePublicPasswordResetConsumeBody
	parseDecoder := json.NewDecoder(parseR.Body)
	parseDecoder.DisallowUnknownFields()
	if parseErr := parseDecoder.Decode(&parseRequestBody); parseErr != nil {
		return parsePublicPasswordResetConsumeBody{}, parseErr
	}
	return parseRequestBody, nil
}

// parseWritePublicPasswordResetResponse writes one JSON response contract for public password-reset endpoints.
func parseWritePublicPasswordResetResponse(parseW http.ResponseWriter, parseStatusCode int, parseResponse parsePublicPasswordResetResponse) {
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseW.Header().Set("Cache-Control", "no-store")
	parseW.WriteHeader(parseStatusCode)
	_ = json.NewEncoder(parseW).Encode(parseResponse)
}

// parseWritePublicSignupVerificationResponse writes one JSON response contract for public signup-verification endpoints.
func parseWritePublicSignupVerificationResponse(parseW http.ResponseWriter, parseStatusCode int, parseResponse parsePublicSignupVerificationResponse) {
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseW.Header().Set("Cache-Control", "no-store")
	parseW.WriteHeader(parseStatusCode)
	_ = json.NewEncoder(parseW).Encode(parseResponse)
}

// parseWritePublicNoContent writes one no-store empty response for same-origin auth session sync endpoints.
func parseWritePublicNoContent(parseW http.ResponseWriter) {
	parseW.Header().Set("Cache-Control", "no-store")
	parseW.WriteHeader(http.StatusNoContent)
}
