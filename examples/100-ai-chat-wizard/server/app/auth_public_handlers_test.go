package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHandlePublicPasswordResetRequestReturnsLocalTokenForQA verifies development mode surfaces reset tokens for local QA.
func TestHandlePublicPasswordResetRequestReturnsLocalTokenForQA(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	if _, parseErr := parseAuth.parseSignup("qa-reset@example.com", "password123", "QA Reset"); parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseServer := &chatServer{authManager: parseAuth, runtimeEnvironment: "development"}
	parseRequestBody := bytes.NewBufferString(`{"email":"qa-reset@example.com"}`)
	parseRequest := httptest.NewRequest(http.MethodPost, "/api/public/auth/password-reset/request", parseRequestBody)
	parseRequest.Header.Set("Content-Type", "application/json")
	parseRequest.Header.Set("X-Forwarded-For", "198.51.100.8")
	parseResponse := httptest.NewRecorder()

	parseServer.parseHandlePublicPasswordResetRequest(parseResponse, parseRequest)

	if parseResponse.Code != http.StatusOK {
		parseT.Fatalf("status code = %d, want %d", parseResponse.Code, http.StatusOK)
	}
	parsePayload := parseDecodePublicPasswordResetResponse(parseT, parseResponse)
	if !parsePayload.IsParseOK || parsePayload.ParseOutcome != parsePublicPasswordResetOutcomeAccepted {
		parseT.Fatalf("unexpected request response: %+v", parsePayload)
	}
	if parsePayload.ParseResetToken == "" {
		parseT.Fatalf("expected local reset token for QA, got %+v", parsePayload)
	}
}

// TestHandlePublicPasswordResetConsumeSuccess verifies valid tokens update the password through the public consume endpoint.
func TestHandlePublicPasswordResetConsumeSuccess(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("consume-success@example.com", "password123", "Consume Success")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseResetToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "203.0.113.42")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken: %v", parseErr)
	}
	parseServer := &chatServer{authManager: parseAuth, runtimeEnvironment: "development"}
	parseRequestBody := bytes.NewBufferString(`{"token":"` + parseResetToken + `","password":"password456"}`)
	parseRequest := httptest.NewRequest(http.MethodPost, "/api/public/auth/password-reset/consume", parseRequestBody)
	parseRequest.Header.Set("Content-Type", "application/json")
	parseResponse := httptest.NewRecorder()

	parseServer.parseHandlePublicPasswordResetConsume(parseResponse, parseRequest)

	if parseResponse.Code != http.StatusOK {
		parseT.Fatalf("status code = %d, want %d", parseResponse.Code, http.StatusOK)
	}
	parsePayload := parseDecodePublicPasswordResetResponse(parseT, parseResponse)
	if !parsePayload.IsParseOK || parsePayload.ParseOutcome != parsePublicPasswordResetOutcomePasswordUpdated {
		parseT.Fatalf("unexpected consume response: %+v", parsePayload)
	}
	if _, parseErr = parseAuth.parseLogin(parseUser.Email, "password456"); parseErr != nil {
		parseT.Fatalf("parseLogin(new password): %v", parseErr)
	}
	if _, parseErr = parseAuth.parseLogin(parseUser.Email, "password123"); !errors.Is(parseErr, errInvalidCredentials) {
		parseT.Fatalf("expected original password to fail, got %v", parseErr)
	}
}

// TestHandlePublicPasswordResetConsumeExpiredTokenOutcome verifies expired tokens return one explicit expired outcome.
func TestHandlePublicPasswordResetConsumeExpiredTokenOutcome(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("consume-expired@example.com", "password123", "Consume Expired")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseResetToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "203.0.113.77")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken: %v", parseErr)
	}
	parseTokenHash := parseBuildAuthFlowTokenHash(parseResetToken)
	_, parseErr = parseStore.db.Exec(
		"UPDATE password_reset_tokens SET expires_at = ? WHERE token_hash = ?",
		time.Now().UTC().Add(-1*time.Minute).Format(time.RFC3339),
		parseTokenHash,
	)
	if parseErr != nil {
		parseT.Fatalf("expire reset token: %v", parseErr)
	}
	parseServer := &chatServer{authManager: parseAuth, runtimeEnvironment: "development"}
	parseRequestBody := bytes.NewBufferString(`{"token":"` + parseResetToken + `","password":"password456"}`)
	parseRequest := httptest.NewRequest(http.MethodPost, "/api/public/auth/password-reset/consume", parseRequestBody)
	parseRequest.Header.Set("Content-Type", "application/json")
	parseResponse := httptest.NewRecorder()

	parseServer.parseHandlePublicPasswordResetConsume(parseResponse, parseRequest)

	if parseResponse.Code != http.StatusBadRequest {
		parseT.Fatalf("status code = %d, want %d", parseResponse.Code, http.StatusBadRequest)
	}
	parsePayload := parseDecodePublicPasswordResetResponse(parseT, parseResponse)
	if parsePayload.IsParseOK || parsePayload.ParseOutcome != parsePublicPasswordResetOutcomeExpiredToken {
		parseT.Fatalf("expected expired-token outcome, got %+v", parsePayload)
	}
}

// TestHandlePublicSignupVerificationResendReturnsLocalTokenForQA verifies development mode surfaces signup-verification tokens for local QA.
func TestHandlePublicSignupVerificationResendReturnsLocalTokenForQA(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	if _, parseErr := parseAuth.parseSignup("qa-verify@example.com", "password123", "QA Verify"); parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseServer := &chatServer{authManager: parseAuth, runtimeEnvironment: "development"}
	parseRequestBody := bytes.NewBufferString(`{"email":"qa-verify@example.com"}`)
	parseRequest := httptest.NewRequest(http.MethodPost, "/api/public/auth/signup-verification/resend", parseRequestBody)
	parseRequest.Header.Set("Content-Type", "application/json")
	parseResponse := httptest.NewRecorder()

	parseServer.parseHandlePublicSignupVerificationResend(parseResponse, parseRequest)

	if parseResponse.Code != http.StatusOK {
		parseT.Fatalf("status code = %d, want %d", parseResponse.Code, http.StatusOK)
	}
	parsePayload := parseDecodePublicSignupVerificationResponse(parseT, parseResponse)
	if !parsePayload.IsParseOK || parsePayload.ParseOutcome != parsePublicSignupVerificationOutcomeAccepted {
		parseT.Fatalf("unexpected resend response: %+v", parsePayload)
	}
	if parsePayload.ParseVerificationToken == "" {
		parseT.Fatalf("expected local verification token for QA, got %+v", parsePayload)
	}
}

// TestHandlePublicSignupVerificationConsumeSuccess verifies valid verification tokens transition users into verified state and return a redirect target.
func TestHandlePublicSignupVerificationConsumeSuccess(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("verify-success@example.com", "password123", "Verify Success")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseVerificationToken, parseErr := parseAuth.parseResendSignupVerificationToken(parseUser.Email)
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken: %v", parseErr)
	}
	if parseVerificationToken == "" {
		parseT.Fatal("expected non-empty verification token")
	}
	parseServer := &chatServer{authManager: parseAuth, runtimeEnvironment: "development"}
	parseRequestBody := bytes.NewBufferString(`{"token":"` + parseVerificationToken + `"}`)
	parseRequest := httptest.NewRequest(http.MethodPost, "/api/public/auth/signup-verification/consume", parseRequestBody)
	parseRequest.Header.Set("Content-Type", "application/json")
	parseResponse := httptest.NewRecorder()

	parseServer.parseHandlePublicSignupVerificationConsume(parseResponse, parseRequest)

	if parseResponse.Code != http.StatusOK {
		parseT.Fatalf("status code = %d, want %d", parseResponse.Code, http.StatusOK)
	}
	parsePayload := parseDecodePublicSignupVerificationResponse(parseT, parseResponse)
	if !parsePayload.IsParseOK || parsePayload.ParseOutcome != parsePublicSignupVerificationOutcomeVerified {
		parseT.Fatalf("unexpected consume response: %+v", parsePayload)
	}
	if parsePayload.ParseRedirectTarget != parsePublicSignupVerificationRedirectTarget {
		parseT.Fatalf("redirect target = %q, want %q", parsePayload.ParseRedirectTarget, parsePublicSignupVerificationRedirectTarget)
	}
	parseStatus, parseErr := parseAuth.parseResolveSignupVerificationStatus(parseUser.Email, time.Now().UTC())
	if parseErr != nil {
		parseT.Fatalf("parseResolveSignupVerificationStatus: %v", parseErr)
	}
	if parseStatus != parseSignupVerificationStatusVerified {
		parseT.Fatalf("verification status = %q, want %q", parseStatus, parseSignupVerificationStatusVerified)
	}
}

// TestHandlePublicSignupVerificationConsumeExpiredTokenOutcome verifies expired verification tokens return one explicit expired outcome.
func TestHandlePublicSignupVerificationConsumeExpiredTokenOutcome(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("verify-expired@example.com", "password123", "Verify Expired")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseVerificationToken, parseErr := parseAuth.parseResendSignupVerificationToken(parseUser.Email)
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken: %v", parseErr)
	}
	if parseVerificationToken == "" {
		parseT.Fatal("expected non-empty verification token")
	}
	parseTokenHash := parseBuildAuthFlowTokenHash(parseVerificationToken)
	_, parseErr = parseStore.db.Exec(
		"UPDATE email_verification_tokens SET expires_at = ? WHERE token_hash = ?",
		time.Now().UTC().Add(-1*time.Minute).Format(time.RFC3339),
		parseTokenHash,
	)
	if parseErr != nil {
		parseT.Fatalf("expire verification token: %v", parseErr)
	}
	parseServer := &chatServer{authManager: parseAuth, runtimeEnvironment: "development"}
	parseRequestBody := bytes.NewBufferString(`{"token":"` + parseVerificationToken + `"}`)
	parseRequest := httptest.NewRequest(http.MethodPost, "/api/public/auth/signup-verification/consume", parseRequestBody)
	parseRequest.Header.Set("Content-Type", "application/json")
	parseResponse := httptest.NewRecorder()

	parseServer.parseHandlePublicSignupVerificationConsume(parseResponse, parseRequest)

	if parseResponse.Code != http.StatusBadRequest {
		parseT.Fatalf("status code = %d, want %d", parseResponse.Code, http.StatusBadRequest)
	}
	parsePayload := parseDecodePublicSignupVerificationResponse(parseT, parseResponse)
	if parsePayload.IsParseOK || parsePayload.ParseOutcome != parsePublicSignupVerificationOutcomeExpiredToken {
		parseT.Fatalf("expected expired-token outcome, got %+v", parsePayload)
	}
}

// parseDecodePublicSignupVerificationResponse decodes one JSON response for public signup-verification handler tests.
func parseDecodePublicSignupVerificationResponse(parseT *testing.T, parseResponse *httptest.ResponseRecorder) parsePublicSignupVerificationResponse {
	parseT.Helper()
	var parsePayload parsePublicSignupVerificationResponse
	if parseErr := json.NewDecoder(parseResponse.Body).Decode(&parsePayload); parseErr != nil {
		parseT.Fatalf("decode response body: %v", parseErr)
	}
	return parsePayload
}

// parseDecodePublicPasswordResetResponse decodes one JSON response for public password-reset handler tests.
func parseDecodePublicPasswordResetResponse(parseT *testing.T, parseResponse *httptest.ResponseRecorder) parsePublicPasswordResetResponse {
	parseT.Helper()
	var parsePayload parsePublicPasswordResetResponse
	if parseErr := json.NewDecoder(parseResponse.Body).Decode(&parsePayload); parseErr != nil {
		parseT.Fatalf("decode response body: %v", parseErr)
	}
	return parsePayload
}
