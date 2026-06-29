package app

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type parseAuthIdentityWrite struct {
	UserID          int64
	ProviderKey     string
	ProviderType    string
	ProviderSubject string
	Email           string
	IsEmailVerified bool
	ProfileJSON     string
	LastLoginAt     string
}

type parseAuthIdentityRow struct {
	ID              int64
	UserID          int64
	ProviderKey     string
	ProviderType    string
	ProviderSubject string
	Email           string
	IsEmailVerified bool
	ProfileJSON     string
	LastLoginAt     string
	CreatedAt       string
	UpdatedAt       string
}

type parseAuthOIDCStateWrite struct {
	ProviderKey     string
	WorkspaceID     int64
	SessionKey      string
	StateTokenHash  string
	NonceTokenHash  string
	ReturnToURL     string
	ExpectedSubject string
	ExpiresAt       string
	ConsumedAt      string
	CreatedByUserID int64
}

type parseAuthOIDCStateRow struct {
	ID              int64
	ProviderKey     string
	WorkspaceID     int64
	SessionKey      string
	StateTokenHash  string
	NonceTokenHash  string
	ReturnToURL     string
	ExpectedSubject string
	ExpiresAt       string
	ConsumedAt      string
	CreatedByUserID int64
	CreatedAt       string
	UpdatedAt       string
}

type parseWorkspaceAuthPolicyWrite struct {
	WorkspaceID              int64
	IsPasswordAllowed        bool
	IsExternalLoginAllowed   bool
	IsSSORequired            bool
	RequiredProviderKey      string
	IsJITProvisioningAllowed bool
	IsLocalPasswordQAAllowed bool
	UpdatedByUserID          int64
}

type parseWorkspaceAuthPolicyRow struct {
	WorkspaceID              int64
	IsPasswordAllowed        bool
	IsExternalLoginAllowed   bool
	IsSSORequired            bool
	RequiredProviderKey      string
	IsJITProvisioningAllowed bool
	IsLocalPasswordQAAllowed bool
	UpdatedByUserID          int64
	UpdatedAt                string
}

// parseNormalizeAuthProviderKey normalizes one provider key into one canonical external-auth value.
func parseNormalizeAuthProviderKey(parseProviderKey string) string {
	return parseNormalizeExternalIdentityProviderKey(parseProviderKey)
}

// parseNormalizeAuthProviderType normalizes one provider type with provider-key fallback.
func parseNormalizeAuthProviderType(parseProviderType string, parseProviderKey string) string {
	parseProviderType = strings.TrimSpace(strings.ToLower(parseProviderType))
	switch parseProviderType {
	case "oidc", "saml":
		return parseProviderType
	}
	if parseProviderKey == parseWorkspaceAuthMethodSAML {
		return "saml"
	}
	return "oidc"
}

// parseUpsertAuthIdentity persists one external identity row keyed by provider subject.
func (parseS *Store) parseUpsertAuthIdentity(parseWrite parseAuthIdentityWrite) (parseAuthIdentityRow, error) {
	if parseWrite.UserID <= 0 {
		return parseAuthIdentityRow{}, errors.New("upsert auth identity: user id is required")
	}
	parseProviderKey := parseNormalizeAuthProviderKey(parseWrite.ProviderKey)
	if parseProviderKey == "" {
		return parseAuthIdentityRow{}, errors.New("upsert auth identity: provider key is required")
	}
	parseProviderSubject := strings.TrimSpace(parseWrite.ProviderSubject)
	if parseProviderSubject == "" {
		return parseAuthIdentityRow{}, errors.New("upsert auth identity: provider subject is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseProfileJSON := strings.TrimSpace(parseWrite.ProfileJSON)
	if parseProfileJSON == "" {
		parseProfileJSON = "{}"
	}
	parseLastLoginAt := strings.TrimSpace(parseWrite.LastLoginAt)
	if parseLastLoginAt == "" {
		parseLastLoginAt = parseNow
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertAuthIdentity,
		parseWrite.UserID,
		parseProviderKey,
		parseNormalizeAuthProviderType(parseWrite.ProviderType, parseProviderKey),
		parseProviderSubject,
		parseNormalizeAuthEmail(parseWrite.Email),
		parseBuildBillingFlagValue(parseWrite.IsEmailVerified),
		parseProfileJSON,
		parseLastLoginAt,
		parseNow,
		parseNow,
		parseWrite.UserID,
	)
	if parseErr != nil {
		return parseAuthIdentityRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseAuthIdentityRow{}, errStoreUserMissing
	}
	parseIdentityRow, isParseFound, parseErr := parseS.parseGetAuthIdentityByProviderSubject(parseProviderKey, parseProviderSubject)
	if parseErr != nil {
		return parseAuthIdentityRow{}, parseErr
	}
	if !isParseFound {
		return parseAuthIdentityRow{}, errors.New("upsert auth identity: identity row missing after write")
	}
	return parseIdentityRow, nil
}

// parseGetAuthIdentityByProviderSubject returns one external identity row for one provider subject pair.
func (parseS *Store) parseGetAuthIdentityByProviderSubject(parseProviderKey, parseProviderSubject string) (parseAuthIdentityRow, bool, error) {
	parseProviderKey = parseNormalizeAuthProviderKey(parseProviderKey)
	parseProviderSubject = strings.TrimSpace(parseProviderSubject)
	if parseProviderKey == "" || parseProviderSubject == "" {
		return parseAuthIdentityRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getAuthIdentityByProviderSubject, parseProviderKey, parseProviderSubject)
	var parseIdentityRow parseAuthIdentityRow
	var parseIsEmailVerified int64
	if parseErr := parseRow.Scan(
		&parseIdentityRow.ID,
		&parseIdentityRow.UserID,
		&parseIdentityRow.ProviderKey,
		&parseIdentityRow.ProviderType,
		&parseIdentityRow.ProviderSubject,
		&parseIdentityRow.Email,
		&parseIsEmailVerified,
		&parseIdentityRow.ProfileJSON,
		&parseIdentityRow.LastLoginAt,
		&parseIdentityRow.CreatedAt,
		&parseIdentityRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseAuthIdentityRow{}, false, nil
		}
		return parseAuthIdentityRow{}, false, parseErr
	}
	parseIdentityRow.IsEmailVerified = parseIsEmailVerified != 0
	return parseIdentityRow, true, nil
}

// parseListAuthIdentitiesByUser lists external identity rows for one user.
func (parseS *Store) parseListAuthIdentitiesByUser(parseUserID int64) ([]parseAuthIdentityRow, error) {
	if parseUserID <= 0 {
		return []parseAuthIdentityRow{}, nil
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAuthIdentitiesByUser, parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()
	parseIdentityRows := make([]parseAuthIdentityRow, 0, 4)
	for parseRows.Next() {
		var parseIdentityRow parseAuthIdentityRow
		var parseIsEmailVerified int64
		if parseErr2 := parseRows.Scan(
			&parseIdentityRow.ID,
			&parseIdentityRow.UserID,
			&parseIdentityRow.ProviderKey,
			&parseIdentityRow.ProviderType,
			&parseIdentityRow.ProviderSubject,
			&parseIdentityRow.Email,
			&parseIsEmailVerified,
			&parseIdentityRow.ProfileJSON,
			&parseIdentityRow.LastLoginAt,
			&parseIdentityRow.CreatedAt,
			&parseIdentityRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseIdentityRow.IsEmailVerified = parseIsEmailVerified != 0
		parseIdentityRows = append(parseIdentityRows, parseIdentityRow)
	}
	if parseErr2 := parseRows.Err(); parseErr2 != nil {
		return nil, parseErr2
	}
	return parseIdentityRows, nil
}

// parseDeleteAuthIdentityByScope deletes one provider identity binding for one user.
func (parseS *Store) parseDeleteAuthIdentityByScope(parseUserID int64, parseProviderKey string) error {
	parseProviderKey = parseNormalizeAuthProviderKey(parseProviderKey)
	if parseUserID <= 0 || parseProviderKey == "" {
		return nil
	}
	_, parseErr := parseS.db.Exec(parseS.queries.deleteAuthIdentityByScope, parseUserID, parseProviderKey)
	return parseErr
}

// parseCreateAuthOIDCState inserts one external-auth handshake state row.
func (parseS *Store) parseCreateAuthOIDCState(parseWrite parseAuthOIDCStateWrite) (parseAuthOIDCStateRow, error) {
	parseProviderKey := parseNormalizeAuthProviderKey(parseWrite.ProviderKey)
	parseSessionKey := strings.TrimSpace(parseWrite.SessionKey)
	parseStateTokenHash := strings.TrimSpace(parseWrite.StateTokenHash)
	parseNonceTokenHash := strings.TrimSpace(parseWrite.NonceTokenHash)
	parseExpiresAt := strings.TrimSpace(parseWrite.ExpiresAt)
	if parseProviderKey == "" || parseSessionKey == "" || parseStateTokenHash == "" || parseNonceTokenHash == "" || parseExpiresAt == "" {
		return parseAuthOIDCStateRow{}, errors.New("create auth oidc state: provider key, session key, state hash, nonce hash, and expires at are required")
	}
	parseWorkspaceID := max(parseWrite.WorkspaceID, 0)
	parseCreatedByUserID := max(parseWrite.CreatedByUserID, 0)
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseReturnToURL := strings.TrimSpace(parseWrite.ReturnToURL)
	if parseReturnToURL == "" {
		parseReturnToURL = "/"
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createAuthOIDCState,
		parseProviderKey,
		parseWorkspaceID,
		parseSessionKey,
		parseStateTokenHash,
		parseNonceTokenHash,
		parseReturnToURL,
		strings.TrimSpace(parseWrite.ExpectedSubject),
		parseExpiresAt,
		strings.TrimSpace(parseWrite.ConsumedAt),
		parseCreatedByUserID,
		parseNow,
		parseNow,
		parseWorkspaceID,
		parseWorkspaceID,
		parseCreatedByUserID,
		parseCreatedByUserID,
	)
	if parseErr != nil {
		return parseAuthOIDCStateRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseAuthOIDCStateRow{}, errStoreSuperuserScopeMissing
	}
	parseStateRow, isParseFound, parseErr := parseS.parseGetAuthOIDCStateByStateTokenHash(parseStateTokenHash)
	if parseErr != nil {
		return parseAuthOIDCStateRow{}, parseErr
	}
	if !isParseFound {
		return parseAuthOIDCStateRow{}, errors.New("create auth oidc state: row missing after write")
	}
	return parseStateRow, nil
}

// parseGetAuthOIDCStateByStateTokenHash returns one external-auth handshake state row by state token hash.
func (parseS *Store) parseGetAuthOIDCStateByStateTokenHash(parseStateTokenHash string) (parseAuthOIDCStateRow, bool, error) {
	parseStateTokenHash = strings.TrimSpace(parseStateTokenHash)
	if parseStateTokenHash == "" {
		return parseAuthOIDCStateRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getAuthOIDCStateByStateTokenHash, parseStateTokenHash)
	var parseStateRow parseAuthOIDCStateRow
	if parseErr := parseRow.Scan(
		&parseStateRow.ID,
		&parseStateRow.ProviderKey,
		&parseStateRow.WorkspaceID,
		&parseStateRow.SessionKey,
		&parseStateRow.StateTokenHash,
		&parseStateRow.NonceTokenHash,
		&parseStateRow.ReturnToURL,
		&parseStateRow.ExpectedSubject,
		&parseStateRow.ExpiresAt,
		&parseStateRow.ConsumedAt,
		&parseStateRow.CreatedByUserID,
		&parseStateRow.CreatedAt,
		&parseStateRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseAuthOIDCStateRow{}, false, nil
		}
		return parseAuthOIDCStateRow{}, false, parseErr
	}
	return parseStateRow, true, nil
}

// parseConsumeAuthOIDCStateByStateTokenHash marks one external-auth handshake state as consumed exactly once.
func (parseS *Store) parseConsumeAuthOIDCStateByStateTokenHash(parseStateTokenHash string, parseConsumedAt time.Time) (bool, error) {
	parseStateTokenHash = strings.TrimSpace(parseStateTokenHash)
	if parseStateTokenHash == "" {
		return false, nil
	}
	if parseConsumedAt.IsZero() {
		parseConsumedAt = time.Now().UTC()
	}
	parseConsumedAtValue := parseConsumedAt.UTC().Format(time.RFC3339)
	parseUpdatedAt := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.consumeAuthOIDCStateByStateTokenHash,
		parseConsumedAtValue,
		parseUpdatedAt,
		parseStateTokenHash,
	)
	if parseErr != nil {
		return false, parseErr
	}
	parseRowsAffected, parseErr2 := parseResult.RowsAffected()
	if parseErr2 != nil {
		return false, parseErr2
	}
	return parseRowsAffected > 0, nil
}

// parseDeleteExpiredAuthOIDCStates deletes expired external-auth handshake states and returns the number of rows removed.
func (parseS *Store) parseDeleteExpiredAuthOIDCStates(parseNow time.Time) (int64, error) {
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteExpiredAuthOIDCStates, parseNow.UTC().Format(time.RFC3339))
	if parseErr != nil {
		return 0, parseErr
	}
	parseRowsAffected, parseErr2 := parseResult.RowsAffected()
	if parseErr2 != nil {
		return 0, parseErr2
	}
	return parseRowsAffected, nil
}

// parseUpsertWorkspaceAuthPolicy upserts one workspace login policy row.
func (parseS *Store) parseUpsertWorkspaceAuthPolicy(parseWrite parseWorkspaceAuthPolicyWrite) error {
	if parseWrite.WorkspaceID <= 0 {
		return errors.New("upsert workspace auth policy: workspace id is required")
	}
	parseRequiredProviderKeyRaw := strings.TrimSpace(parseWrite.RequiredProviderKey)
	parseRequiredProviderKey := parseNormalizeAuthProviderKey(parseRequiredProviderKeyRaw)
	if parseRequiredProviderKeyRaw != "" && parseRequiredProviderKey == "" {
		return errors.New("upsert workspace auth policy: required provider key is invalid")
	}
	if parseRequiredProviderKey != "" && !parseWrite.IsExternalLoginAllowed {
		return errors.New("upsert workspace auth policy: required provider requires external login allowed")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWorkspaceAuthPolicy,
		parseWrite.WorkspaceID,
		parseBuildBillingFlagValue(parseWrite.IsPasswordAllowed),
		parseBuildBillingFlagValue(parseWrite.IsExternalLoginAllowed),
		parseBuildBillingFlagValue(parseWrite.IsSSORequired),
		parseRequiredProviderKey,
		parseBuildBillingFlagValue(parseWrite.IsJITProvisioningAllowed),
		parseBuildBillingFlagValue(parseWrite.IsLocalPasswordQAAllowed),
		parseWrite.UpdatedByUserID,
		parseNow,
		parseWrite.WorkspaceID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseGetWorkspaceAuthPolicyByWorkspace returns one persisted workspace auth policy row when present.
func (parseS *Store) parseGetWorkspaceAuthPolicyByWorkspace(parseWorkspaceID int64) (parseWorkspaceAuthPolicyRow, bool, error) {
	if parseWorkspaceID <= 0 {
		return parseWorkspaceAuthPolicyRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getWorkspaceAuthPolicyByWorkspace, parseWorkspaceID)
	var parsePolicyRow parseWorkspaceAuthPolicyRow
	var parseIsPasswordAllowed int64
	var parseIsExternalLoginAllowed int64
	var parseIsSSORequired int64
	var parseIsJITProvisioningAllowed int64
	var parseIsLocalPasswordQAAllowed int64
	if parseErr := parseRow.Scan(
		&parsePolicyRow.WorkspaceID,
		&parseIsPasswordAllowed,
		&parseIsExternalLoginAllowed,
		&parseIsSSORequired,
		&parsePolicyRow.RequiredProviderKey,
		&parseIsJITProvisioningAllowed,
		&parseIsLocalPasswordQAAllowed,
		&parsePolicyRow.UpdatedByUserID,
		&parsePolicyRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseWorkspaceAuthPolicyRow{}, false, nil
		}
		return parseWorkspaceAuthPolicyRow{}, false, parseErr
	}
	parsePolicyRow.IsPasswordAllowed = parseIsPasswordAllowed != 0
	parsePolicyRow.IsExternalLoginAllowed = parseIsExternalLoginAllowed != 0
	parsePolicyRow.IsSSORequired = parseIsSSORequired != 0
	parsePolicyRow.IsJITProvisioningAllowed = parseIsJITProvisioningAllowed != 0
	parsePolicyRow.IsLocalPasswordQAAllowed = parseIsLocalPasswordQAAllowed != 0
	return parsePolicyRow, true, nil
}
