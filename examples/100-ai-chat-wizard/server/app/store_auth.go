package app

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type parseAuthSessionWrite struct {
	UserID           int64
	SessionID        string
	TokenVersion     int64
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	LastSeenAt       string
	ExpiresAt        string
	RevokedAt        string
}

type parseAuthSessionRow struct {
	ID               int64
	UserID           int64
	SessionID        string
	TokenVersion     int64
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	LastSeenAt       string
	ExpiresAt        string
	RevokedAt        string
	CreatedAt        string
	UpdatedAt        string
}

type parseUserAccessStateWrite struct {
	UserID           int64
	Status           string
	Reason           string
	DisabledByUserID int64
	DisabledAt       string
}

type parseUserAccessStateRow struct {
	UserID           int64
	Status           string
	Reason           string
	DisabledByUserID int64
	DisabledAt       string
	UpdatedAt        string
}

type parseEmailVerificationTokenWrite struct {
	UserID    int64
	Email     string
	TokenHash string
	ExpiresAt string
}

type parseEmailVerificationTokenRow struct {
	ID         int64
	UserID     int64
	Email      string
	Status     string
	ExpiresAt  string
	VerifiedAt string
	CreatedAt  string
}

type parsePasswordResetTokenWrite struct {
	UserID        int64
	Email         string
	TokenHash     string
	RequestedByIP string
	ExpiresAt     string
}

type parsePasswordResetTokenRow struct {
	ID            int64
	UserID        int64
	Email         string
	Status        string
	RequestedByIP string
	ExpiresAt     string
	ConsumedAt    string
	CreatedAt     string
}

// parseNormalizeUserAccessStateStatus normalizes one user access-state status value.
func parseNormalizeUserAccessStateStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "disabled":
		return "disabled"
	default:
		return "active"
	}
}

// parseUpsertUserAccessState persists one explicit user access-state override.
func (parseS *Store) parseUpsertUserAccessState(parseWrite parseUserAccessStateWrite) error {
	if parseWrite.UserID <= 0 {
		return errors.New("upsert user access state: user id is required")
	}
	parseStatus := parseNormalizeUserAccessStateStatus(parseWrite.Status)
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseDisabledAt := strings.TrimSpace(parseWrite.DisabledAt)
	if parseStatus == "disabled" && parseDisabledAt == "" {
		parseDisabledAt = parseNow
	}
	if parseStatus != "disabled" {
		parseDisabledAt = ""
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertUserAccessState,
		parseWrite.UserID,
		parseStatus,
		strings.TrimSpace(parseWrite.Reason),
		parseWrite.DisabledByUserID,
		parseDisabledAt,
		parseNow,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreUserMissing
	}
	return nil
}

// parseGetUserAccessState returns one persisted access-state row when present.
func (parseS *Store) parseGetUserAccessState(parseUserID int64) (parseUserAccessStateRow, bool, error) {
	if parseUserID <= 0 {
		return parseUserAccessStateRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getUserAccessState, parseUserID)
	var parseState parseUserAccessStateRow
	if parseErr := parseRow.Scan(
		&parseState.UserID,
		&parseState.Status,
		&parseState.Reason,
		&parseState.DisabledByUserID,
		&parseState.DisabledAt,
		&parseState.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseUserAccessStateRow{}, false, nil
		}
		return parseUserAccessStateRow{}, false, parseErr
	}
	parseState.Status = parseNormalizeUserAccessStateStatus(parseState.Status)
	return parseState, true, nil
}

// parseIsUserAccessDisabled reports whether one user has one persisted disabled access state.
func (parseS *Store) parseIsUserAccessDisabled(parseUserID int64) (bool, error) {
	parseState, isParseFound, parseErr := parseS.parseGetUserAccessState(parseUserID)
	if parseErr != nil || !isParseFound {
		return false, parseErr
	}
	return parseState.Status == "disabled", nil
}

// parseEnsureAuthTokenVersion ensures one auth token version row exists for one user and returns the current version.
func (parseS *Store) parseEnsureAuthTokenVersion(parseUserID int64) (int64, error) {
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertAuthTokenVersion,
		parseUserID,
		1,
		parseNow,
	); parseErr != nil {
		return 0, parseErr
	}
	return parseS.parseGetAuthTokenVersion(parseUserID)
}

// parseGetAuthTokenVersion returns one user's current token version when present.
func (parseS *Store) parseGetAuthTokenVersion(parseUserID int64) (int64, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getAuthTokenVersion, parseUserID)
	var parseTokenVersion int64
	if parseErr := parseRow.Scan(&parseTokenVersion); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, parseErr
	}
	return parseTokenVersion, nil
}

// parseUpsertUserAuthBlock persists one active auth block for one user.
func (parseS *Store) parseUpsertUserAuthBlock(parseUserID int64, parseBlockKey, parseBlockSource, parseBlockReason string) error {
	parseBlockKey = strings.TrimSpace(parseBlockKey)
	if parseUserID <= 0 || parseBlockKey == "" {
		return errors.New("upsert user auth block: user id and block key are required")
	}
	parseBlockSource = strings.TrimSpace(parseBlockSource)
	if parseBlockSource == "" {
		parseBlockSource = parseBlockKey
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertUserAuthBlock,
		parseUserID,
		parseBlockKey,
		parseBlockSource,
		strings.TrimSpace(parseBlockReason),
		parseNow,
		parseNow,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreUserMissing
	}
	return nil
}

// parseCountUserAuthBlocksByUser returns the number of active auth blocks for one user.
func (parseS *Store) parseCountUserAuthBlocksByUser(parseUserID int64) (int64, error) {
	if parseUserID <= 0 {
		return 0, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.countUserAuthBlocksByUser, parseUserID)
	var parseBlockCount int64
	if parseErr := parseRow.Scan(&parseBlockCount); parseErr != nil {
		return 0, parseErr
	}
	return parseBlockCount, nil
}

// parseDeleteUserAuthBlock clears one auth block key for one user.
func (parseS *Store) parseDeleteUserAuthBlock(parseUserID int64, parseBlockKey string) error {
	parseBlockKey = strings.TrimSpace(parseBlockKey)
	if parseUserID <= 0 || parseBlockKey == "" {
		return nil
	}
	_, parseErr := parseS.db.Exec(parseS.queries.deleteUserAuthBlock, parseUserID, parseBlockKey)
	return parseErr
}

// parseDeleteUserAuthBlocksByKey clears one auth block key across all users.
func (parseS *Store) parseDeleteUserAuthBlocksByKey(parseBlockKey string) error {
	parseBlockKey = strings.TrimSpace(parseBlockKey)
	if parseBlockKey == "" {
		return nil
	}
	_, parseErr := parseS.db.Exec(parseS.queries.deleteUserAuthBlocksByKey, parseBlockKey)
	return parseErr
}

// parseUpsertAuthSession writes one auth session row and keeps the session active.
func (parseS *Store) parseUpsertAuthSession(parseWrite parseAuthSessionWrite) error {
	parseSessionID := strings.TrimSpace(parseWrite.SessionID)
	if parseWrite.UserID <= 0 || parseSessionID == "" {
		return errors.New("upsert auth session: user and session are required")
	}
	if parseWrite.TokenVersion <= 0 {
		return errors.New("upsert auth session: token version must be positive")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseLastSeenAt := strings.TrimSpace(parseWrite.LastSeenAt)
	if parseLastSeenAt == "" {
		parseLastSeenAt = parseNow
	}
	parseExpiresAt := strings.TrimSpace(parseWrite.ExpiresAt)
	if parseExpiresAt == "" {
		parseExpiresAt = time.Now().UTC().Add(authTokenTTL).Format(time.RFC3339)
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertAuthSession,
		parseWrite.UserID,
		parseSessionID,
		parseWrite.TokenVersion,
		strings.TrimSpace(parseWrite.RefreshTokenHash),
		strings.TrimSpace(parseWrite.UserAgent),
		strings.TrimSpace(parseWrite.IPAddress),
		parseLastSeenAt,
		parseExpiresAt,
		strings.TrimSpace(parseWrite.RevokedAt),
		parseNow,
		parseNow,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreUserMissing
	}
	return nil
}

// parseGetAuthSessionBySessionID resolves one auth session row by stable session identifier.
func (parseS *Store) parseGetAuthSessionBySessionID(parseSessionID string) (parseAuthSessionRow, bool, error) {
	parseSessionID = strings.TrimSpace(parseSessionID)
	if parseSessionID == "" {
		return parseAuthSessionRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getAuthSessionBySessionID, parseSessionID)
	var parseSession parseAuthSessionRow
	if parseErr := parseRow.Scan(
		&parseSession.ID,
		&parseSession.UserID,
		&parseSession.SessionID,
		&parseSession.TokenVersion,
		&parseSession.RefreshTokenHash,
		&parseSession.UserAgent,
		&parseSession.IPAddress,
		&parseSession.LastSeenAt,
		&parseSession.ExpiresAt,
		&parseSession.RevokedAt,
		&parseSession.CreatedAt,
		&parseSession.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseAuthSessionRow{}, false, nil
		}
		return parseAuthSessionRow{}, false, parseErr
	}
	return parseSession, true, nil
}

// parseTouchAuthSessionLastSeen updates one session's observed activity and expiry window.
func (parseS *Store) parseTouchAuthSessionLastSeen(parseSessionID, parseUserAgent, parseIPAddress string, parseExpiresAt time.Time) error {
	parseSessionID = strings.TrimSpace(parseSessionID)
	if parseSessionID == "" {
		return errors.New("touch auth session: session id is required")
	}
	parseNow := time.Now().UTC()
	_, parseErr := parseS.db.Exec(
		parseS.queries.touchAuthSessionLastSeen,
		strings.TrimSpace(parseUserAgent),
		strings.TrimSpace(parseIPAddress),
		parseNow.Format(time.RFC3339),
		parseExpiresAt.UTC().Format(time.RFC3339),
		parseNow.Format(time.RFC3339),
		parseSessionID,
	)
	return parseErr
}

// parseRevokeAuthSession marks one auth session row as revoked.
func (parseS *Store) parseRevokeAuthSession(parseSessionID string) error {
	parseSessionID = strings.TrimSpace(parseSessionID)
	if parseSessionID == "" {
		return nil
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	_, parseErr := parseS.db.Exec(
		parseS.queries.revokeAuthSession,
		parseNow,
		parseNow,
		parseSessionID,
	)
	return parseErr
}

// parseRevokeAuthSessionsByUser marks all active sessions revoked for one user id.
func (parseS *Store) parseRevokeAuthSessionsByUser(parseUserID int64) error {
	if parseUserID <= 0 {
		return nil
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	_, parseErr := parseS.db.Exec(
		parseS.queries.revokeAuthSessionsByUser,
		parseNow,
		parseNow,
		parseUserID,
	)
	return parseErr
}

// parseUpdateUserPasswordHash updates one stored password hash for one user id.
func (parseS *Store) parseUpdateUserPasswordHash(parseUserID int64, parsePasswordHash string) error {
	if parseUserID <= 0 {
		return errors.New("update user password hash: user id is required")
	}
	parsePasswordHash = strings.TrimSpace(parsePasswordHash)
	if parsePasswordHash == "" {
		return errors.New("update user password hash: password hash is required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.updateUserPasswordHash,
		parsePasswordHash,
		parseUserID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreUserMissing
	}
	return nil
}

// parseIncrementAuthTokenVersion increments and persists one auth token version for one user id.
func (parseS *Store) parseIncrementAuthTokenVersion(parseUserID int64) (int64, error) {
	parseCurrentVersion, parseErr := parseS.parseEnsureAuthTokenVersion(parseUserID)
	if parseErr != nil {
		return 0, parseErr
	}
	parseNextVersion := parseCurrentVersion + 1
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr2 := parseS.db.Exec(
		parseS.queries.upsertAuthTokenVersion,
		parseUserID,
		parseNextVersion,
		parseNow,
	); parseErr2 != nil {
		return 0, parseErr2
	}
	return parseNextVersion, nil
}

// parseCreateEmailVerificationToken inserts one pending email-verification token row.
func (parseS *Store) parseCreateEmailVerificationToken(parseWrite parseEmailVerificationTokenWrite) (int64, error) {
	parseEmail := parseNormalizeAuthEmail(parseWrite.Email)
	parseTokenHash := strings.TrimSpace(parseWrite.TokenHash)
	parseExpiresAt := strings.TrimSpace(parseWrite.ExpiresAt)
	if parseWrite.UserID <= 0 || parseEmail == "" || parseTokenHash == "" || parseExpiresAt == "" {
		return 0, errors.New("create email verification token: user, email, token hash, and expires at are required")
	}
	parseCreatedAt := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createEmailVerificationToken,
		parseWrite.UserID,
		parseEmail,
		parseTokenHash,
		"pending",
		parseExpiresAt,
		"",
		parseCreatedAt,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreUserMissing
	}
	return parseResult.LastInsertId()
}

// parseConsumeEmailVerificationToken verifies one pending, unexpired email-verification token hash exactly once.
func (parseS *Store) parseConsumeEmailVerificationToken(parseTokenHash string, parseNow time.Time) (parseEmailVerificationTokenRow, bool, error) {
	parseTokenHash = strings.TrimSpace(parseTokenHash)
	if parseTokenHash == "" {
		return parseEmailVerificationTokenRow{}, false, nil
	}
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseTx, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parseEmailVerificationTokenRow{}, false, parseErr
	}
	defer func() {
		_ = parseTx.Rollback()
	}()

	parseRow := parseTx.QueryRow(parseS.queries.getEmailVerificationTokenByHash, parseTokenHash)
	var parseTokenRow parseEmailVerificationTokenRow
	if parseErr2 := parseRow.Scan(
		&parseTokenRow.ID,
		&parseTokenRow.UserID,
		&parseTokenRow.Email,
		&parseTokenRow.Status,
		&parseTokenRow.ExpiresAt,
		&parseTokenRow.VerifiedAt,
		&parseTokenRow.CreatedAt,
	); parseErr2 != nil {
		if errors.Is(parseErr2, sql.ErrNoRows) {
			return parseEmailVerificationTokenRow{}, false, nil
		}
		return parseEmailVerificationTokenRow{}, false, parseErr2
	}
	if !strings.EqualFold(strings.TrimSpace(parseTokenRow.Status), "pending") {
		return parseEmailVerificationTokenRow{}, false, nil
	}
	parseExpiresAt, parseErr2 := time.Parse(time.RFC3339, strings.TrimSpace(parseTokenRow.ExpiresAt))
	if parseErr2 != nil {
		return parseEmailVerificationTokenRow{}, false, parseErr2
	}
	if parseNow.UTC().After(parseExpiresAt.UTC()) {
		return parseEmailVerificationTokenRow{}, false, nil
	}
	parseVerifiedAt := parseNow.UTC().Format(time.RFC3339)
	parseResult, parseErr2 := parseTx.Exec(parseS.queries.markEmailVerificationTokenVerified, parseVerifiedAt, parseTokenRow.ID)
	if parseErr2 != nil {
		return parseEmailVerificationTokenRow{}, false, parseErr2
	}
	if parseRowsAffected, parseErr3 := parseResult.RowsAffected(); parseErr3 == nil && parseRowsAffected == 0 {
		return parseEmailVerificationTokenRow{}, false, nil
	}
	if parseErr3 := parseTx.Commit(); parseErr3 != nil {
		return parseEmailVerificationTokenRow{}, false, parseErr3
	}
	parseTokenRow.Status = "verified"
	parseTokenRow.VerifiedAt = parseVerifiedAt
	return parseTokenRow, true, nil
}

// parseCreatePasswordResetToken inserts one pending password-reset token row.
func (parseS *Store) parseCreatePasswordResetToken(parseWrite parsePasswordResetTokenWrite) (int64, error) {
	parseEmail := parseNormalizeAuthEmail(parseWrite.Email)
	parseTokenHash := strings.TrimSpace(parseWrite.TokenHash)
	parseExpiresAt := strings.TrimSpace(parseWrite.ExpiresAt)
	if parseWrite.UserID <= 0 || parseEmail == "" || parseTokenHash == "" || parseExpiresAt == "" {
		return 0, errors.New("create password reset token: user, email, token hash, and expires at are required")
	}
	parseCreatedAt := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createPasswordResetToken,
		parseWrite.UserID,
		parseEmail,
		parseTokenHash,
		"pending",
		strings.TrimSpace(parseWrite.RequestedByIP),
		parseExpiresAt,
		"",
		parseCreatedAt,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreUserMissing
	}
	return parseResult.LastInsertId()
}

// parseConsumePasswordResetToken consumes one pending, unexpired password-reset token hash exactly once.
func (parseS *Store) parseConsumePasswordResetToken(parseTokenHash string, parseNow time.Time) (parsePasswordResetTokenRow, bool, error) {
	parseTokenHash = strings.TrimSpace(parseTokenHash)
	if parseTokenHash == "" {
		return parsePasswordResetTokenRow{}, false, nil
	}
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseTx, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parsePasswordResetTokenRow{}, false, parseErr
	}
	defer func() {
		_ = parseTx.Rollback()
	}()

	parseRow := parseTx.QueryRow(parseS.queries.getPasswordResetTokenByHash, parseTokenHash)
	var parseTokenRow parsePasswordResetTokenRow
	if parseErr2 := parseRow.Scan(
		&parseTokenRow.ID,
		&parseTokenRow.UserID,
		&parseTokenRow.Email,
		&parseTokenRow.Status,
		&parseTokenRow.RequestedByIP,
		&parseTokenRow.ExpiresAt,
		&parseTokenRow.ConsumedAt,
		&parseTokenRow.CreatedAt,
	); parseErr2 != nil {
		if errors.Is(parseErr2, sql.ErrNoRows) {
			return parsePasswordResetTokenRow{}, false, nil
		}
		return parsePasswordResetTokenRow{}, false, parseErr2
	}
	if !strings.EqualFold(strings.TrimSpace(parseTokenRow.Status), "pending") {
		return parsePasswordResetTokenRow{}, false, nil
	}
	parseExpiresAt, parseErr2 := time.Parse(time.RFC3339, strings.TrimSpace(parseTokenRow.ExpiresAt))
	if parseErr2 != nil {
		return parsePasswordResetTokenRow{}, false, parseErr2
	}
	if parseNow.UTC().After(parseExpiresAt.UTC()) {
		return parsePasswordResetTokenRow{}, false, nil
	}
	parseConsumedAt := parseNow.UTC().Format(time.RFC3339)
	parseResult, parseErr2 := parseTx.Exec(parseS.queries.consumePasswordResetToken, parseConsumedAt, parseTokenRow.ID)
	if parseErr2 != nil {
		return parsePasswordResetTokenRow{}, false, parseErr2
	}
	if parseRowsAffected, parseErr3 := parseResult.RowsAffected(); parseErr3 == nil && parseRowsAffected == 0 {
		return parsePasswordResetTokenRow{}, false, nil
	}
	if parseErr3 := parseTx.Commit(); parseErr3 != nil {
		return parsePasswordResetTokenRow{}, false, parseErr3
	}
	parseTokenRow.Status = "consumed"
	parseTokenRow.ConsumedAt = parseConsumedAt
	return parseTokenRow, true, nil
}
