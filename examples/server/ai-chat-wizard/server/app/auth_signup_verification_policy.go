package app

import (
	"errors"
	"log/slog"
	"strings"
	"time"
)

const emailVerificationResendThrottleWindow = 1 * time.Hour
const emailVerificationResendThrottleMaxPerEmail = int64(3)

const parseSignupVerificationStatusVerified = "verified"
const parseSignupVerificationStatusPending = "pending"
const parseSignupVerificationStatusExpired = "expired"
const parseSignupVerificationStatusUnknown = "unknown"

const parseSignupVerificationAuditEventRequest = "auth.signup_verification.request"
const parseSignupVerificationAuditEventConsume = "auth.signup_verification.consume"

const parseSignupVerificationAuditOutcomeRequested = "requested"
const parseSignupVerificationAuditOutcomeThrottled = "throttled"
const parseSignupVerificationAuditOutcomeMissingUser = "missing_user"
const parseSignupVerificationAuditOutcomeAlreadyVerified = "already_verified"
const parseSignupVerificationAuditOutcomePolicyDenied = "policy_denied"
const parseSignupVerificationAuditOutcomeVerified = "verified"
const parseSignupVerificationAuditOutcomeExpired = "expired"
const parseSignupVerificationAuditOutcomeReplayDenied = "replay_denied"
const parseSignupVerificationAuditOutcomeInvalidToken = "invalid_token"

var errSignupVerificationUnavailable = errors.New("signup verification unavailable")

type parseSignupVerificationPolicy struct {
	isParseSignInAllowedBeforeVerification     bool
	isParseResendAllowedBeforeVerification     bool
	isParseProtectedActionBlockedUntilVerified bool
}

// parseResolveSignupVerificationPolicy returns the canonical signup-verification policy contract for password accounts.
func parseResolveSignupVerificationPolicy() parseSignupVerificationPolicy {
	return parseSignupVerificationPolicy{
		isParseSignInAllowedBeforeVerification:     true,
		isParseResendAllowedBeforeVerification:     true,
		isParseProtectedActionBlockedUntilVerified: false,
	}
}

// parseLogSignupVerificationAuditEvent emits one structured signup-verification audit event for operator diagnostics.
func (parseA *authManager) parseLogSignupVerificationAuditEvent(parseEvent string, parseOutcome string, parseUserID int64, parseEmail string) {
	if parseA == nil || parseA.logger == nil {
		return
	}
	parseA.logger.Info(
		"auth signup verification audit",
		slog.String("event", strings.TrimSpace(parseEvent)),
		slog.String("outcome", strings.TrimSpace(parseOutcome)),
		slog.Int64("user_id", parseUserID),
		slog.String("email_hash", parseBuildAuthAuditEmailHash(parseEmail)),
	)
}

// parseResolveSignupVerificationStatus resolves one normalized signup-verification status for one email.
func (parseA *authManager) parseResolveSignupVerificationStatus(parseEmail string, parseNow time.Time) (string, error) {
	if parseA == nil || parseA.store == nil {
		return parseSignupVerificationStatusUnknown, errSignupVerificationUnavailable
	}
	parseStatus, parseErr := parseA.store.parseResolveEmailVerificationStatusByEmail(parseEmail, parseNow)
	if parseErr != nil {
		return parseSignupVerificationStatusUnknown, parseErr
	}
	parseStatus = strings.TrimSpace(strings.ToLower(parseStatus))
	switch parseStatus {
	case parseSignupVerificationStatusVerified, parseSignupVerificationStatusPending, parseSignupVerificationStatusExpired:
		return parseStatus, nil
	default:
		return parseSignupVerificationStatusUnknown, nil
	}
}

// parseIsSignupVerificationResendThrottled reports whether verification resend requests exceed per-email limits.
func (parseA *authManager) parseIsSignupVerificationResendThrottled(parseEmail string, parseNow time.Time) (bool, error) {
	if parseA == nil || parseA.store == nil {
		return false, errSignupVerificationUnavailable
	}
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseSince := parseNow.UTC().Add(-1 * emailVerificationResendThrottleWindow)
	parseRequestCount, parseErr := parseA.store.parseCountEmailVerificationTokenRequestsByEmailSince(parseEmail, parseSince)
	if parseErr != nil {
		return false, parseErr
	}
	return parseRequestCount >= emailVerificationResendThrottleMaxPerEmail, nil
}

// parseResolveSignupVerificationConsumeOutcome classifies one failed verification consume attempt into one stable outcome.
func (parseA *authManager) parseResolveSignupVerificationConsumeOutcome(parseTokenHash string, parseNow time.Time) string {
	if parseA == nil || parseA.store == nil {
		return parseSignupVerificationAuditOutcomeInvalidToken
	}
	parseTokenRow, isParseTokenFound, parseErr := parseA.store.parseGetEmailVerificationTokenByHash(parseTokenHash)
	if parseErr != nil || !isParseTokenFound {
		return parseSignupVerificationAuditOutcomeInvalidToken
	}
	if !strings.EqualFold(strings.TrimSpace(parseTokenRow.Status), "pending") {
		return parseSignupVerificationAuditOutcomeReplayDenied
	}
	parseExpiresAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseTokenRow.ExpiresAt))
	if parseErr != nil {
		return parseSignupVerificationAuditOutcomeInvalidToken
	}
	if parseNow.UTC().After(parseExpiresAt.UTC()) {
		return parseSignupVerificationAuditOutcomeExpired
	}
	return parseSignupVerificationAuditOutcomeReplayDenied
}

// parseAuthorizeLoginSignupVerificationPolicy enforces login behavior for verified vs unverified password accounts.
func (parseA *authManager) parseAuthorizeLoginSignupVerificationPolicy(parseEmail string, parseNow time.Time) (bool, error) {
	parsePolicy := parseResolveSignupVerificationPolicy()
	parseStatus, parseErr := parseA.parseResolveSignupVerificationStatus(parseEmail, parseNow)
	if parseErr != nil {
		return false, parseErr
	}
	if parseStatus != parseSignupVerificationStatusVerified && !parsePolicy.isParseSignInAllowedBeforeVerification {
		return false, nil
	}
	return true, nil
}

// parseShouldBlockProtectedActionPendingVerification returns whether protected actions should be blocked for one account.
func (parseA *authManager) parseShouldBlockProtectedActionPendingVerification(parseEmail string, parseNow time.Time) (bool, error) {
	parsePolicy := parseResolveSignupVerificationPolicy()
	if !parsePolicy.isParseProtectedActionBlockedUntilVerified {
		return false, nil
	}
	parseStatus, parseErr := parseA.parseResolveSignupVerificationStatus(parseEmail, parseNow)
	if parseErr != nil {
		return true, parseErr
	}
	return parseStatus != parseSignupVerificationStatusVerified, nil
}

// parseResendSignupVerificationToken issues a fresh verification token when resend policy allows it.
func (parseA *authManager) parseResendSignupVerificationToken(parseEmail string) (string, error) {
	if parseA == nil || parseA.store == nil {
		return "", errSignupVerificationUnavailable
	}
	parseEmail = parseNormalizeAuthEmail(parseEmail)
	if parseEmail == "" {
		return "", errInvalidCredentials
	}
	parsePolicy := parseResolveSignupVerificationPolicy()
	if !parsePolicy.isParseResendAllowedBeforeVerification {
		parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventRequest, parseSignupVerificationAuditOutcomePolicyDenied, 0, parseEmail)
		return "", nil
	}
	parseNow := time.Now().UTC()
	isParseThrottled, parseErr := parseA.parseIsSignupVerificationResendThrottled(parseEmail, parseNow)
	if parseErr != nil {
		return "", parseErr
	}
	if isParseThrottled {
		parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventRequest, parseSignupVerificationAuditOutcomeThrottled, 0, parseEmail)
		return "", nil
	}
	parseUserRecord, parseErr := parseA.store.getUserAuthByEmail(parseEmail)
	if parseErr != nil {
		parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventRequest, parseSignupVerificationAuditOutcomeMissingUser, 0, parseEmail)
		return "", nil
	}
	parseStatus, parseErr := parseA.parseResolveSignupVerificationStatus(parseUserRecord.Email, parseNow)
	if parseErr != nil {
		return "", parseErr
	}
	if parseStatus == parseSignupVerificationStatusVerified {
		parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventRequest, parseSignupVerificationAuditOutcomeAlreadyVerified, parseUserRecord.ID, parseUserRecord.Email)
		return "", nil
	}
	parseVerificationToken, parseErr := parseA.parseIssueEmailVerificationTokenForUser(authUser{
		ID:    parseUserRecord.ID,
		Email: parseUserRecord.Email,
	})
	if parseErr != nil {
		return "", errSignupVerificationUnavailable
	}
	parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventRequest, parseSignupVerificationAuditOutcomeRequested, parseUserRecord.ID, parseUserRecord.Email)
	return parseVerificationToken, nil
}

// parseCompleteSignupVerificationWithToken consumes one verification token and marks that token lifecycle as verified.
func (parseA *authManager) parseCompleteSignupVerificationWithToken(parseVerificationToken string) error {
	if parseA == nil || parseA.store == nil {
		return errSignupVerificationUnavailable
	}
	parseVerificationToken = strings.TrimSpace(parseVerificationToken)
	if parseVerificationToken == "" {
		return errInvalidCredentials
	}
	parseTokenHash := parseBuildAuthFlowTokenHash(parseVerificationToken)
	parseNow := time.Now().UTC()
	parseTokenRow, isParseConsumed, parseErr := parseA.store.parseConsumeEmailVerificationToken(parseTokenHash, parseNow)
	if parseErr != nil {
		return errSignupVerificationUnavailable
	}
	if !isParseConsumed || parseTokenRow.UserID <= 0 {
		parseOutcome := parseA.parseResolveSignupVerificationConsumeOutcome(parseTokenHash, parseNow)
		parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventConsume, parseOutcome, 0, "")
		return errInvalidCredentials
	}
	parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventConsume, parseSignupVerificationAuditOutcomeVerified, parseTokenRow.UserID, parseTokenRow.Email)
	return nil
}
