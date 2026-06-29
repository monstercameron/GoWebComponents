package app

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
)

// TestAuthManagerSessionValidationFailureLogging verifies session-validation failure logs include typed context fields.
func TestAuthManagerSessionValidationFailureLogging(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	var parseLogOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseLogOutput, serverServiceName)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseLogger)

	parseAuth.parseLogSessionValidationFailure(" session lookup failed ", &authClaims{
		UserID:       42,
		SessionID:    "sess-42",
		TokenVersion: 3,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: "jti-42",
		},
	}, errors.New("db unavailable"))

	parseLogLine := strings.TrimSpace(parseLogOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one session-validation warning log line")
	}
	if !strings.Contains(parseLogLine, `"message":"auth: session validation failed"`) {
		parseT.Fatalf("expected auth session validation log message, got %q", parseLogLine)
	}
	if !strings.Contains(parseLogLine, `"reason":"session lookup failed"`) {
		parseT.Fatalf("expected reason field in log line, got %q", parseLogLine)
	}
	if !strings.Contains(parseLogLine, `"session_id":"sess-42"`) || !strings.Contains(parseLogLine, `"jti":"jti-42"`) {
		parseT.Fatalf("expected session and jti context in log line, got %q", parseLogLine)
	}
	if !strings.Contains(parseLogLine, `"token_version":3`) || !strings.Contains(parseLogLine, `"error":"db unavailable"`) {
		parseT.Fatalf("expected token_version and error fields in log line, got %q", parseLogLine)
	}
}

func TestAuthManagerSignupLoginAndTokenRoundTrip(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())

	parseUser, parseErr := parseAuth.parseSignup(" Test@Example.com ", "password123", "")
	if parseErr != nil {
		parseT.Fatalf("signup: %v", parseErr)
	}
	if parseUser.Email != "test@example.com" {
		parseT.Fatalf("normalized email mismatch: %q", parseUser.Email)
	}

	parseRecord, parseErr := store.getUserAuthByEmail("test@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail: %v", parseErr)
	}
	if bcrypt.CompareHashAndPassword([]byte(parseRecord.PasswordHash), []byte("password123")) != nil {
		parseT.Fatal("stored password hash does not match original password")
	}

	parseLoggedIn, parseErr := parseAuth.parseLogin("TEST@example.com", "password123")
	if parseErr != nil {
		parseT.Fatalf("login: %v", parseErr)
	}
	if parseLoggedIn.ID != parseUser.ID {
		parseT.Fatalf("login user mismatch: got %d want %d", parseLoggedIn.ID, parseUser.ID)
	}

	parseToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}
	parseParsed, parseErr := parseAuth.parseToken(parseToken)
	if parseErr != nil {
		parseT.Fatalf("parseToken: %v", parseErr)
	}
	if parseParsed != parseUser {
		parseT.Fatalf("parsed user mismatch: got %+v want %+v", parseParsed, parseUser)
	}
}

func TestAuthManagerPersistsSessionAndRevocation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("session-user@example.com", "password123", "Session User")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}

	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-agent", "relaydesk-e2e/1.0",
		"x-forwarded-for", "203.0.113.10",
	))
	parseToken, parseErr := parseAuth.issueTokenForContext(parseCtx, parseUser, "")
	if parseErr != nil {
		parseT.Fatalf("issueTokenForContext: %v", parseErr)
	}

	parseParsedToken, parseErr := jwt.ParseWithClaims(parseToken, &authClaims{}, func(parseToken *jwt.Token) (any, error) {
		return parseAuth.secret, nil
	})
	if parseErr != nil {
		parseT.Fatalf("jwt parse claims: %v", parseErr)
	}
	parseClaims, parseOk := parseParsedToken.Claims.(*authClaims)
	if !parseOk || !parseParsedToken.Valid {
		parseT.Fatalf("expected valid auth claims, got %#v", parseParsedToken.Claims)
	}
	if parseClaims.TokenVersion != 1 || parseClaims.SessionID == "" {
		parseT.Fatalf("expected session-backed claims, got %+v", parseClaims)
	}
	if parseClaims.ID != parseClaims.SessionID {
		parseT.Fatalf("expected jti to match session id, got jti=%q sid=%q", parseClaims.ID, parseClaims.SessionID)
	}

	parseSession, isParseFound, parseErr := parseStore.parseGetAuthSessionBySessionID(parseClaims.SessionID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthSessionBySessionID: %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatalf("expected auth session row for %q", parseClaims.SessionID)
	}
	if parseSession.UserID != parseUser.ID || parseSession.TokenVersion != 1 {
		parseT.Fatalf("unexpected auth session row: %+v", parseSession)
	}
	if parseSession.UserAgent != "relaydesk-e2e/1.0" || parseSession.IPAddress != "203.0.113.10" {
		parseT.Fatalf("expected request metadata capture, got %+v", parseSession)
	}

	parseAuthCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseToken))
	if parseResolved, parseOk2 := parseAuth.parseAuthenticatedUserFromContext(parseAuthCtx); !parseOk2 || parseResolved.ID != parseUser.ID {
		parseT.Fatalf("expected authenticated user from context, got user=%+v ok=%v", parseResolved, parseOk2)
	}

	if parseErr2 := parseStore.parseRevokeAuthSession(parseClaims.SessionID); parseErr2 != nil {
		parseT.Fatalf("parseRevokeAuthSession: %v", parseErr2)
	}
	if _, parseOk2 := parseAuth.parseAuthenticatedUserFromContext(parseAuthCtx); parseOk2 {
		parseT.Fatal("expected revoked session to fail authentication")
	}
}

func TestAuthManagerTokenKidRotationPolicy(parseT *testing.T) {
	parseT.Setenv("CHAT_AUTH_SIGNING_KEYS", "legacy=legacy-secret,rotated=rotated-secret")
	parseT.Setenv("CHAT_AUTH_ACTIVE_KID", "rotated")

	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("legacy-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("kid-policy@example.com", "password123", "Kid Policy")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseIssuedToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}

	parseParser := jwt.Parser{}
	parseUnverifiedToken, _, parseErr := parseParser.ParseUnverified(parseIssuedToken, &authClaims{})
	if parseErr != nil {
		parseT.Fatalf("ParseUnverified: %v", parseErr)
	}
	parseIssuedTokenKeyID, parseOk := parseUnverifiedToken.Header["kid"].(string)
	if !parseOk || strings.TrimSpace(parseIssuedTokenKeyID) != "rotated" {
		parseT.Fatalf("expected issued token kid=rotated, got %#v", parseUnverifiedToken.Header["kid"])
	}

	parseParsedToken, parseErr := jwt.ParseWithClaims(parseIssuedToken, &authClaims{}, func(parseToken *jwt.Token) (any, error) {
		return parseAuth.secret, nil
	})
	if parseErr != nil {
		parseT.Fatalf("ParseWithClaims issued token: %v", parseErr)
	}
	parseClaims, parseOk := parseParsedToken.Claims.(*authClaims)
	if !parseOk || !parseParsedToken.Valid {
		parseT.Fatalf("expected valid issued claims, got %#v", parseParsedToken.Claims)
	}

	parseLegacyToken := jwt.NewWithClaims(jwt.SigningMethodHS256, *parseClaims)
	delete(parseLegacyToken.Header, "kid")
	parseLegacyTokenString, parseErr := parseLegacyToken.SignedString(parseAuth.verifyKeys["legacy"])
	if parseErr != nil {
		parseT.Fatalf("SignedString legacy token: %v", parseErr)
	}
	if _, parseErr2 := parseAuth.parseToken(parseLegacyTokenString); parseErr2 != nil {
		parseT.Fatalf("expected kid-less legacy token to validate during rotation window, got %v", parseErr2)
	}

	parseUnknownKidToken := jwt.NewWithClaims(jwt.SigningMethodHS256, *parseClaims)
	parseUnknownKidToken.Header["kid"] = "missing"
	parseUnknownKidTokenString, parseErr := parseUnknownKidToken.SignedString([]byte("missing-secret"))
	if parseErr != nil {
		parseT.Fatalf("SignedString unknown kid token: %v", parseErr)
	}
	if _, parseErr2 := parseAuth.parseToken(parseUnknownKidTokenString); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected unknown kid token to fail with invalid credentials, got %v", parseErr2)
	}
}

func TestAuthManagerNegativePaths(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())
	parseDevFallback := parseNewAuthManager(" ", store, parseNewTestLogger())
	parseNoStoreAuth := parseNewAuthManager("test-secret", nil, parseNewTestLogger())
	if len(parseDevFallback.secret) == 0 {
		parseT.Fatal("expected fallback auth secret to be populated")
	}
	if parseDefaultDisplayNameFromEmail("   ") != "User" {
		parseT.Fatal("expected blank email to fall back to default display name")
	}
	if _, parseErr := parseNoStoreAuth.parseSignup("nostore@example.com", "password123", ""); parseErr == nil {
		parseT.Fatal("expected signup to fail when store is unavailable")
	}
	if _, parseErr2 := parseNoStoreAuth.parseLogin("nostore@example.com", "password123"); parseErr2 == nil {
		parseT.Fatal("expected login to fail when store is unavailable")
	}

	if _, parseErr3 := parseAuth.parseSignup("", "password123", ""); parseErr3 == nil {
		parseT.Fatal("expected signup to reject missing email")
	}
	if _, parseErr4 := parseAuth.parseSignup("a@example.com", "short", ""); parseErr4 == nil {
		parseT.Fatal("expected signup to reject short password")
	}

	if _, parseErr5 := parseAuth.parseSignup("user@example.com", "password123", "User"); parseErr5 != nil {
		parseT.Fatalf("initial signup: %v", parseErr5)
	}
	if _, parseErr6 := parseAuth.parseSignup("user@example.com", "password123", "User"); !errors.Is(parseErr6, errUserAlreadyExists) {
		parseT.Fatalf("expected duplicate signup to return errUserAlreadyExists, got %v", parseErr6)
	}

	if _, parseErr7 := parseAuth.parseLogin("user@example.com", "wrong-password"); !errors.Is(parseErr7, errInvalidCredentials) {
		parseT.Fatalf("expected invalid credentials for wrong password, got %v", parseErr7)
	}
	if _, parseErr8 := parseAuth.parseToken("not-a-jwt"); parseErr8 == nil {
		parseT.Fatal("expected parseToken to reject invalid JWT")
	}
	parseNonHMACToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": 1, "email": "user@example.com"})
	parseTokenString, parseErr9 := parseNonHMACToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if parseErr9 != nil {
		parseT.Fatalf("SignedString none token: %v", parseErr9)
	}
	if _, parseErr10 := parseAuth.parseToken(parseTokenString); parseErr10 == nil {
		parseT.Fatal("expected parseToken to reject unexpected signing method")
	}
	if _, parseErr11 := parseAuth.parseToken(""); !errors.Is(parseErr11, errInvalidCredentials) {
		parseT.Fatalf("expected empty token to return invalid credentials, got %v", parseErr11)
	}
}

// TestAuthManagerPersistedTokenLifecycles verifies signup verification, reset-password, and update-password use persisted token tables.
func TestAuthManagerPersistedTokenLifecycles(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("lifecycle@example.com", "password123", "Lifecycle")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}

	var parseVerificationTokenHash string
	if parseErr2 := parseStore.db.QueryRow(
		`SELECT token_hash FROM email_verification_tokens WHERE user_id = ? ORDER BY id DESC LIMIT 1`,
		parseUser.ID,
	).Scan(&parseVerificationTokenHash); parseErr2 != nil {
		parseT.Fatalf("select email verification token hash: %v", parseErr2)
	}
	parseVerificationRow, hasParseVerificationRow, parseErr := parseStore.parseConsumeEmailVerificationToken(parseVerificationTokenHash, time.Now().UTC())
	if parseErr != nil {
		parseT.Fatalf("parseConsumeEmailVerificationToken: %v", parseErr)
	}
	if !hasParseVerificationRow || parseVerificationRow.UserID != parseUser.ID || parseVerificationRow.Status != "verified" {
		parseT.Fatalf("unexpected verification row: has=%v row=%+v", hasParseVerificationRow, parseVerificationRow)
	}
	if _, hasParseSecondVerificationRow, parseErr2 := parseStore.parseConsumeEmailVerificationToken(parseVerificationTokenHash, time.Now().UTC()); parseErr2 != nil {
		parseT.Fatalf("parseConsumeEmailVerificationToken second consume: %v", parseErr2)
	} else if hasParseSecondVerificationRow {
		parseT.Fatal("expected verification token to be one-time")
	}

	parseInitialToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}
	parseParsedToken, parseErr := jwt.ParseWithClaims(parseInitialToken, &authClaims{}, func(parseToken *jwt.Token) (any, error) {
		return parseAuth.secret, nil
	})
	if parseErr != nil {
		parseT.Fatalf("jwt ParseWithClaims: %v", parseErr)
	}
	parseClaims, parseOk := parseParsedToken.Claims.(*authClaims)
	if !parseOk || !parseParsedToken.Valid {
		parseT.Fatalf("expected valid auth claims, got %#v", parseParsedToken.Claims)
	}

	parseUnknownResetToken, parseErr := parseAuth.parseBeginPasswordResetToken("missing@example.com", "203.0.113.1")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken missing user: %v", parseErr)
	}
	if strings.TrimSpace(parseUnknownResetToken) != "" {
		parseT.Fatalf("expected unknown-email reset flow to return empty token, got %q", parseUnknownResetToken)
	}

	parseResetToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "203.0.113.2")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken: %v", parseErr)
	}
	if strings.TrimSpace(parseResetToken) == "" {
		parseT.Fatal("expected non-empty reset token")
	}
	if parseErr2 := parseAuth.parseCompletePasswordResetWithToken(parseResetToken, "password456"); parseErr2 != nil {
		parseT.Fatalf("parseCompletePasswordResetWithToken: %v", parseErr2)
	}
	if _, parseErr2 := parseAuth.parseLogin(parseUser.Email, "password123"); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected old password login failure after reset, got %v", parseErr2)
	}
	if _, parseErr2 := parseAuth.parseLogin(parseUser.Email, "password456"); parseErr2 != nil {
		parseT.Fatalf("expected new password login success after reset, got %v", parseErr2)
	}
	if _, parseErr2 := parseAuth.parseToken(parseInitialToken); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected pre-reset token revocation, got %v", parseErr2)
	}
	if parseErr2 := parseAuth.parseCompletePasswordResetWithToken(parseResetToken, "password789"); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected one-time reset token consume, got %v", parseErr2)
	}
	parseTokenVersion, parseErr := parseStore.parseGetAuthTokenVersion(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthTokenVersion: %v", parseErr)
	}
	if parseTokenVersion != 2 {
		parseT.Fatalf("expected token version 2 after reset completion, got %d", parseTokenVersion)
	}
	parseSessionRow, hasParseSessionRow, parseErr := parseStore.parseGetAuthSessionBySessionID(parseClaims.SessionID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthSessionBySessionID: %v", parseErr)
	}
	if !hasParseSessionRow || strings.TrimSpace(parseSessionRow.RevokedAt) == "" {
		parseT.Fatalf("expected reset flow to revoke active sessions, got has=%v row=%+v", hasParseSessionRow, parseSessionRow)
	}

	if parseErr2 := parseAuth.parseUpdatePasswordWithCurrentPassword(parseUser.Email, "password456", "password789"); parseErr2 != nil {
		parseT.Fatalf("parseUpdatePasswordWithCurrentPassword: %v", parseErr2)
	}
	if _, parseErr2 := parseAuth.parseLogin(parseUser.Email, "password456"); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected previous password login failure after update-password, got %v", parseErr2)
	}
	if _, parseErr2 := parseAuth.parseLogin(parseUser.Email, "password789"); parseErr2 != nil {
		parseT.Fatalf("expected updated password login success, got %v", parseErr2)
	}
	if parseErr2 := parseAuth.parseUpdatePasswordWithCurrentPassword(parseUser.Email, "wrong-current", "password999"); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected update-password to require valid current password, got %v", parseErr2)
	}

	var parseConsumedSelfServiceResets int64
	if parseErr2 := parseStore.db.QueryRow(
		`SELECT COUNT(1) FROM password_reset_tokens WHERE user_id = ? AND requested_by_ip = 'self-service' AND status = 'consumed'`,
		parseUser.ID,
	).Scan(&parseConsumedSelfServiceResets); parseErr2 != nil {
		parseT.Fatalf("count self-service reset rows: %v", parseErr2)
	}
	if parseConsumedSelfServiceResets < 1 {
		parseT.Fatalf("expected at least one consumed self-service reset token row, got %d", parseConsumedSelfServiceResets)
	}
}

// TestAuthManagerPasswordRecoveryRequestThrottleAndAudit verifies password-reset request throttling and audit visibility.
func TestAuthManagerPasswordRecoveryRequestThrottleAndAudit(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	var parseLogOutput bytes.Buffer
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewOTELLogger(&parseLogOutput, serverServiceName))
	parseUser, parseErr := parseAuth.parseSignup("throttle@example.com", "password123", "Throttle")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}

	for parseRequestIndex := range passwordResetRequestThrottleMaxPerEmail {
		parseResetToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "198.51.100.33")
		if parseErr != nil {
			parseT.Fatalf("parseBeginPasswordResetToken(request=%d): %v", parseRequestIndex, parseErr)
		}
		if strings.TrimSpace(parseResetToken) == "" {
			parseT.Fatalf("expected reset token for non-throttled request #%d", parseRequestIndex)
		}
	}
	parseThrottledToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "198.51.100.33")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken(throttled): %v", parseErr)
	}
	if strings.TrimSpace(parseThrottledToken) != "" {
		parseT.Fatalf("expected throttled request to suppress token emission, got %q", parseThrottledToken)
	}

	parseLogs := parseLogOutput.String()
	if !strings.Contains(parseLogs, `"event":"auth.recovery.request"`) || !strings.Contains(parseLogs, `"outcome":"throttled"`) {
		parseT.Fatalf("expected throttled recovery request audit log, got %q", parseLogs)
	}
}

// TestAuthManagerPasswordRecoveryConsumeAuditOutcomes verifies consume-path audit outcomes for consumed, expired, and replay-denied cases.
func TestAuthManagerPasswordRecoveryConsumeAuditOutcomes(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	var parseLogOutput bytes.Buffer
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewOTELLogger(&parseLogOutput, serverServiceName))
	parseUser, parseErr := parseAuth.parseSignup("consume@example.com", "password123", "Consume")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}

	parseExpiredToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "203.0.113.90")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken(expired setup): %v", parseErr)
	}
	parseExpiredTokenHash := parseBuildAuthFlowTokenHash(parseExpiredToken)
	if _, parseErr2 := parseStore.db.Exec(
		`UPDATE password_reset_tokens SET expires_at = ? WHERE token_hash = ?`,
		time.Now().UTC().Add(-1*time.Minute).Format(time.RFC3339),
		parseExpiredTokenHash,
	); parseErr2 != nil {
		parseT.Fatalf("expire token setup: %v", parseErr2)
	}
	if parseErr2 := parseAuth.parseCompletePasswordResetWithToken(parseExpiredToken, "password456"); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected expired token consume to fail with invalid credentials, got %v", parseErr2)
	}

	parseReplayToken, parseErr := parseAuth.parseBeginPasswordResetToken(parseUser.Email, "203.0.113.91")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken(replay setup): %v", parseErr)
	}
	if parseErr2 := parseAuth.parseCompletePasswordResetWithToken(parseReplayToken, "password456"); parseErr2 != nil {
		parseT.Fatalf("parseCompletePasswordResetWithToken(first consume): %v", parseErr2)
	}
	if parseErr2 := parseAuth.parseCompletePasswordResetWithToken(parseReplayToken, "password789"); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected replay consume to fail with invalid credentials, got %v", parseErr2)
	}

	parseLogs := parseLogOutput.String()
	if !strings.Contains(parseLogs, `"event":"auth.recovery.consume"`) {
		parseT.Fatalf("expected recovery consume audit logs, got %q", parseLogs)
	}
	if !strings.Contains(parseLogs, `"outcome":"expired"`) {
		parseT.Fatalf("expected expired consume audit outcome, got %q", parseLogs)
	}
	if !strings.Contains(parseLogs, `"outcome":"replay_denied"`) {
		parseT.Fatalf("expected replay-denied consume audit outcome, got %q", parseLogs)
	}
	if !strings.Contains(parseLogs, `"outcome":"consumed"`) {
		parseT.Fatalf("expected successful consume audit outcome, got %q", parseLogs)
	}
}

// TestSignupVerificationPolicyContract verifies explicit signup-verification policy defaults.
func TestSignupVerificationPolicyContract(parseT *testing.T) {
	parsePolicy := parseResolveSignupVerificationPolicy()
	if !parsePolicy.isParseSignInAllowedBeforeVerification {
		parseT.Fatal("expected policy to allow sign-in before verification")
	}
	if !parsePolicy.isParseResendAllowedBeforeVerification {
		parseT.Fatal("expected policy to allow verification resend")
	}
	if parsePolicy.isParseProtectedActionBlockedUntilVerified {
		parseT.Fatal("expected policy to keep protected-action blocking disabled by default")
	}
}

// TestAuthManagerSignupVerificationLifecycle verifies resend/consume policy behavior and fail-closed replay handling.
func TestAuthManagerSignupVerificationLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	var parseLogOutput bytes.Buffer
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewOTELLogger(&parseLogOutput, serverServiceName))
	parseUser, parseErr := parseAuth.parseSignup("verification@example.com", "password123", "Verification")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	if _, parseErr2 := parseAuth.parseLogin(parseUser.Email, "password123"); parseErr2 != nil {
		parseT.Fatalf("expected unverified login to follow policy and succeed, got %v", parseErr2)
	}
	isParseBlocked, parseErr := parseAuth.parseShouldBlockProtectedActionPendingVerification(parseUser.Email, time.Now().UTC())
	if parseErr != nil {
		parseT.Fatalf("parseShouldBlockProtectedActionPendingVerification: %v", parseErr)
	}
	if isParseBlocked {
		parseT.Fatal("expected protected-action check to follow policy and remain unblocked")
	}

	parseMissingResendToken, parseErr := parseAuth.parseResendSignupVerificationToken("missing@example.com")
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken(missing): %v", parseErr)
	}
	if strings.TrimSpace(parseMissingResendToken) != "" {
		parseT.Fatalf("expected missing-user resend to suppress token emission, got %q", parseMissingResendToken)
	}

	parseVerificationToken, parseErr := parseAuth.parseResendSignupVerificationToken(parseUser.Email)
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken(user): %v", parseErr)
	}
	if strings.TrimSpace(parseVerificationToken) == "" {
		parseT.Fatal("expected non-empty resend token for unverified user")
	}
	if parseErr = parseAuth.parseCompleteSignupVerificationWithToken(parseVerificationToken); parseErr != nil {
		parseT.Fatalf("parseCompleteSignupVerificationWithToken(first): %v", parseErr)
	}
	if parseErr = parseAuth.parseCompleteSignupVerificationWithToken(parseVerificationToken); !errors.Is(parseErr, errInvalidCredentials) {
		parseT.Fatalf("expected one-time verification consume replay denial, got %v", parseErr)
	}
	parsePostVerifyResendToken, parseErr := parseAuth.parseResendSignupVerificationToken(parseUser.Email)
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken(post-verify): %v", parseErr)
	}
	if strings.TrimSpace(parsePostVerifyResendToken) != "" {
		parseT.Fatalf("expected already-verified resend to suppress token emission, got %q", parsePostVerifyResendToken)
	}

	parseLogs := parseLogOutput.String()
	if !strings.Contains(parseLogs, `"event":"auth.signup_verification.request"`) || !strings.Contains(parseLogs, `"outcome":"requested"`) {
		parseT.Fatalf("expected signup verification request audit logs, got %q", parseLogs)
	}
	if !strings.Contains(parseLogs, `"event":"auth.signup_verification.consume"`) || !strings.Contains(parseLogs, `"outcome":"verified"`) {
		parseT.Fatalf("expected verification consume success audit logs, got %q", parseLogs)
	}
	if !strings.Contains(parseLogs, `"outcome":"replay_denied"`) || !strings.Contains(parseLogs, `"outcome":"already_verified"`) {
		parseT.Fatalf("expected replay/already-verified audit outcomes, got %q", parseLogs)
	}
}

// TestAuthManagerSignupVerificationThrottleAndExpiry verifies resend throttling and expired-token fail-closed behavior.
func TestAuthManagerSignupVerificationThrottleAndExpiry(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	var parseLogOutput bytes.Buffer
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewOTELLogger(&parseLogOutput, serverServiceName))
	parseUser, parseErr := parseAuth.parseSignup("verify-throttle@example.com", "password123", "VerifyThrottle")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}

	for parseRequestIndex := range emailVerificationResendThrottleMaxPerEmail - 1 {
		parseResendToken, parseErr := parseAuth.parseResendSignupVerificationToken(parseUser.Email)
		if parseErr != nil {
			parseT.Fatalf("parseResendSignupVerificationToken(request=%d): %v", parseRequestIndex, parseErr)
		}
		if strings.TrimSpace(parseResendToken) == "" {
			parseT.Fatalf("expected resend token for non-throttled request #%d", parseRequestIndex)
		}
	}
	parseThrottledResendToken, parseErr := parseAuth.parseResendSignupVerificationToken(parseUser.Email)
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken(throttled): %v", parseErr)
	}
	if strings.TrimSpace(parseThrottledResendToken) != "" {
		parseT.Fatalf("expected throttled resend to suppress token emission, got %q", parseThrottledResendToken)
	}

	parseExpiredToken, parseErr := parseAuth.parseResendSignupVerificationToken("verify-expired@example.com")
	if parseErr == nil && strings.TrimSpace(parseExpiredToken) != "" {
		parseT.Fatalf("expected missing-user resend to suppress token emission, got %q", parseExpiredToken)
	}
	parseExpiredToken, parseErr = parseAuth.parseResendSignupVerificationToken(parseUser.Email)
	if parseErr != nil {
		parseT.Fatalf("parseResendSignupVerificationToken(expired setup): %v", parseErr)
	}
	if strings.TrimSpace(parseExpiredToken) != "" {
		parseT.Fatalf("expected throttled resend during expired setup to suppress token emission, got %q", parseExpiredToken)
	}

	parseSeedToken, parseErr := parseAuth.parseIssueEmailVerificationTokenForUser(parseUser)
	if parseErr != nil {
		parseT.Fatalf("parseIssueEmailVerificationTokenForUser(expired seed): %v", parseErr)
	}
	parseSeedTokenHash := parseBuildAuthFlowTokenHash(parseSeedToken)
	if _, parseErr2 := parseStore.db.Exec(
		`UPDATE email_verification_tokens SET expires_at = ? WHERE token_hash = ?`,
		time.Now().UTC().Add(-1*time.Minute).Format(time.RFC3339),
		parseSeedTokenHash,
	); parseErr2 != nil {
		parseT.Fatalf("expire verification token setup: %v", parseErr2)
	}
	if parseErr2 := parseAuth.parseCompleteSignupVerificationWithToken(parseSeedToken); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected expired verification consume to fail closed, got %v", parseErr2)
	}

	parseLogs := parseLogOutput.String()
	if !strings.Contains(parseLogs, `"outcome":"throttled"`) {
		parseT.Fatalf("expected resend throttled audit outcome, got %q", parseLogs)
	}
	if !strings.Contains(parseLogs, `"event":"auth.signup_verification.consume"`) || !strings.Contains(parseLogs, `"outcome":"expired"`) {
		parseT.Fatalf("expected expired verification consume audit outcome, got %q", parseLogs)
	}
}

func TestRequestUsesHTTPSAndCookieHelpers(parseT *testing.T) {
	parseAuth := parseNewAuthManager("test-secret", parseNewTestStore(parseT), parseNewTestLogger())

	parsePlainReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if parseRequestUsesHTTPS(nil) {
		parseT.Fatal("nil request should not be treated as HTTPS")
	}
	if parseRequestUsesHTTPS(parsePlainReq) {
		parseT.Fatal("plain request should not be treated as HTTPS")
	}

	parseForwardedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseForwardedReq.Header.Set("X-Forwarded-Proto", "https")
	if !parseRequestUsesHTTPS(parseForwardedReq) {
		parseT.Fatal("forwarded https request should be treated as HTTPS")
	}

	parseTlsReq := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	parseTlsReq.TLS = &tls.ConnectionState{}
	if !parseRequestUsesHTTPS(parseTlsReq) {
		parseT.Fatal("TLS request should be treated as HTTPS")
	}

	parseWriter := httptest.NewRecorder()
	parseAuth.setAuthCookie(parseWriter, parseForwardedReq, "token-value")
	parseResp := parseWriter.Result()
	if len(parseResp.Cookies()) != 1 {
		parseT.Fatalf("expected one cookie, got %d", len(parseResp.Cookies()))
	}
	parseCookie := parseResp.Cookies()[0]
	if !parseCookie.HttpOnly || !parseCookie.Secure || parseCookie.Value != "token-value" {
		parseT.Fatalf("unexpected auth cookie: %+v", parseCookie)
	}

	clearWriter := httptest.NewRecorder()
	parseAuth.clearAuthCookie(clearWriter, parsePlainReq)
	parseClearedCookie := clearWriter.Result().Cookies()[0]
	if parseClearedCookie.MaxAge != -1 {
		parseT.Fatalf("expected cleared cookie MaxAge -1, got %d", parseClearedCookie.MaxAge)
	}
	if parseClearedCookie.Value != "" {
		parseT.Fatalf("expected cleared cookie value to be empty, got %q", parseClearedCookie.Value)
	}
}

func TestAuthenticatedHandlers(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("demo@example.com", "password123", "Demo")
	if parseErr != nil {
		parseT.Fatalf("signup: %v", parseErr)
	}
	parseToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}

	parsePageHandler := parseAuth.parseRequireAuthenticatedPage(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusNoContent)
	}))

	parseUnauthorizedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseUnauthorizedWriter := httptest.NewRecorder()
	parsePageHandler.ServeHTTP(parseUnauthorizedWriter, parseUnauthorizedReq)
	if parseUnauthorizedWriter.Code != http.StatusSeeOther {
		parseT.Fatalf("expected redirect for unauthenticated page request, got %d", parseUnauthorizedWriter.Code)
	}
	if parseLocation := parseUnauthorizedWriter.Result().Header.Get("Location"); parseLocation != "/app" {
		parseT.Fatalf("unexpected redirect location: %q", parseLocation)
	}

	parseAuthorizedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseAuthorizedReq.AddCookie(&http.Cookie{Name: authCookieName, Value: parseToken})
	parseAuthorizedWriter := httptest.NewRecorder()
	parsePageHandler.ServeHTTP(parseAuthorizedWriter, parseAuthorizedReq)
	if parseAuthorizedWriter.Code != http.StatusNoContent {
		parseT.Fatalf("expected protected page handler to run, got %d", parseAuthorizedWriter.Code)
	}

	isParseTunnelCalled := false
	isParseCallbackCalled := false
	parseTunnelHandler := parseAuth.parseRequireAuthenticatedTunnel(func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
		isParseTunnelCalled = true
		parseW2.WriteHeader(http.StatusAccepted)
	}, func(parseR3 *http.Request, parseUser2 authUser) {
		isParseCallbackCalled = parseUser2.ID == parseUser.ID && parseR3.URL.Path == "/grpc"
	})

	parseUnauthorizedTunnelWriter := httptest.NewRecorder()
	parseAuth.parseRequireAuthenticatedTunnel(func(parseW3 http.ResponseWriter, parseR4 *http.Request) {
		parseW3.WriteHeader(http.StatusNoContent)
	}, nil)(parseUnauthorizedTunnelWriter, httptest.NewRequest(http.MethodGet, "http://example.com/grpc", nil))
	if parseUnauthorizedTunnelWriter.Code != http.StatusUnauthorized {
		parseT.Fatalf("expected unauthorized tunnel response, got %d", parseUnauthorizedTunnelWriter.Code)
	}

	parseTunnelReq := httptest.NewRequest(http.MethodGet, "http://example.com/grpc", nil)
	parseTunnelReq.AddCookie(&http.Cookie{Name: authCookieName, Value: parseToken})
	parseTunnelWriter := httptest.NewRecorder()
	parseTunnelHandler(parseTunnelWriter, parseTunnelReq)
	if !isParseTunnelCalled {
		parseT.Fatal("expected authenticated tunnel handler to run")
	}
	if !isParseCallbackCalled {
		parseT.Fatal("expected authenticated tunnel callback to run")
	}
	if parseTunnelWriter.Code != http.StatusAccepted {
		parseT.Fatalf("expected authenticated tunnel response, got %d", parseTunnelWriter.Code)
	}
}
