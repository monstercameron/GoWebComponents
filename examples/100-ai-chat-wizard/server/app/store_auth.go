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
