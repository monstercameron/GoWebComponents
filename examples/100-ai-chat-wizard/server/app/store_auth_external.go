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
	WorkspaceID                  int64
	IsPasswordAllowed            bool
	IsExternalLoginAllowed       bool
	IsSSORequired                bool
	RequiredProviderKey          string
	IsJITProvisioningAllowed     bool
	IsLocalPasswordQAModeAllowed bool
	UpdatedByUserID              int64
}

type parseWorkspaceAuthPolicyRow struct {
	WorkspaceID                  int64
	IsPasswordAllowed            bool
	IsExternalLoginAllowed       bool
	IsSSORequired                bool
	RequiredProviderKey          string
	IsJITProvisioningAllowed     bool
	IsLocalPasswordQAModeAllowed bool
	UpdatedByUserID              int64
	UpdatedAt                    string
}

// parseBoolToInt64 converts one boolean flag into one sqlite-compatible integer value.
func parseBoolToInt64(isParseEnabled bool) int64 {
	if isParseEnabled {
		return 1
	}
	return 0
}

// parseNormalizeAuthIdentityProviderType normalizes one identity provider type with provider-key fallback.
func parseNormalizeAuthIdentityProviderType(parseProviderType string, parseProviderKey string) string {
	parseProviderType = strings.TrimSpace(strings.ToLower(parseProviderType))
	switch parseProviderType {
	case "oidc", "saml":
		return parseProviderType
	}
	if parseNormalizeExternalIdentityProviderKey(parseProviderKey) == parseWorkspaceAuthMethodSAML {
		return "saml"
	}
	return "oidc"
}

// parseUpsertAuthIdentity persists one external identity row and returns the stored identity snapshot.
func (parseS *Store) parseUpsertAuthIdentity(parseWrite parseAuthIdentityWrite) (parseAuthIdentityRow, error) {
	if parseWrite.UserID <= 0 {
		return parseAuthIdentityRow{}, errors.New("upsert auth identity: user id is required")
	}
	parseProviderKey := parseNormalizeExternalIdentityProviderKey(parseWrite.ProviderKey)
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
		parseNormalizeAuthIdentityProviderType(parseWrite.ProviderType, parseProviderKey),
		parseProviderSubject,
		parseNormalizeAuthEmail(parseWrite.Email),
		parseBoolToInt64(parseWrite.IsEmailVerified),
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
	parseRow, isParseFound, parseErr := parseS.parseGetAuthIdentityByProviderSubject(parseProviderKey, parseProviderSubject)
	if parseErr != nil {
		return parseAuthIdentityRow{}, parseErr
	}
	if !isParseFound {
		return parseAuthIdentityRow{}, errors.New("upsert auth identity: persisted row missing")
	}
	return parseRow, nil
}

// parseGetAuthIdentityByProviderSubject returns one identity row by provider+subject when present.
func (parseS *Store) parseGetAuthIdentityByProviderSubject(parseProviderKey string, parseProviderSubject string) (parseAuthIdentityRow, bool, error) {
	parseProviderKey = parseNormalizeExternalIdentityProviderKey(parseProviderKey)
	parseProviderSubject = strings.TrimSpace(parseProviderSubject)
	if parseProviderKey == "" || parseProviderSubject == "" {
		return parseAuthIdentityRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getAuthIdentityByProviderSubject, parseProviderKey, parseProviderSubject)
	var parseIdentity parseAuthIdentityRow
	var parseIsEmailVerified int64
	if parseErr := parseRow.Scan(
		&parseIdentity.ID,
		&parseIdentity.UserID,
		&parseIdentity.ProviderKey,
		&parseIdentity.ProviderType,
		&parseIdentity.ProviderSubject,
		&parseIdentity.Email,
		&parseIsEmailVerified,
		&parseIdentity.ProfileJSON,
		&parseIdentity.LastLoginAt,
		&parseIdentity.CreatedAt,
		&parseIdentity.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseAuthIdentityRow{}, false, nil
		}
		return parseAuthIdentityRow{}, false, parseErr
	}
	parseIdentity.IsEmailVerified = parseIsEmailVerified != 0
	return parseIdentity, true, nil
}

// parseListAuthIdentitiesByUser returns identity rows for one user newest-first.
func (parseS *Store) parseListAuthIdentitiesByUser(parseUserID int64) ([]parseAuthIdentityRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAuthIdentitiesByUser, parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseIdentityRows := make([]parseAuthIdentityRow, 0)
	for parseRows.Next() {
		var parseIdentity parseAuthIdentityRow
		var parseIsEmailVerified int64
		if parseErr2 := parseRows.Scan(
			&parseIdentity.ID,
			&parseIdentity.UserID,
			&parseIdentity.ProviderKey,
			&parseIdentity.ProviderType,
			&parseIdentity.ProviderSubject,
			&parseIdentity.Email,
			&parseIsEmailVerified,
			&parseIdentity.ProfileJSON,
			&parseIdentity.LastLoginAt,
			&parseIdentity.CreatedAt,
			&parseIdentity.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseIdentity.IsEmailVerified = parseIsEmailVerified != 0
		parseIdentityRows = append(parseIdentityRows, parseIdentity)
	}
	return parseIdentityRows, parseRows.Err()
}

// parseDeleteAuthIdentityByScope deletes one provider identity for one user scope.
func (parseS *Store) parseDeleteAuthIdentityByScope(parseUserID int64, parseProviderKey string) error {
	parseProviderKey = parseNormalizeExternalIdentityProviderKey(parseProviderKey)
	if parseUserID <= 0 || parseProviderKey == "" {
		return nil
	}
	_, parseErr := parseS.db.Exec(parseS.queries.deleteAuthIdentityByScope, parseUserID, parseProviderKey)
	return parseErr
}

// parseCreateAuthOIDCState persists one OIDC handshake state row and returns the stored snapshot.
func (parseS *Store) parseCreateAuthOIDCState(parseWrite parseAuthOIDCStateWrite) (parseAuthOIDCStateRow, error) {
	parseProviderKey := parseNormalizeExternalIdentityProviderKey(parseWrite.ProviderKey)
	if parseProviderKey == "" {
		return parseAuthOIDCStateRow{}, errors.New("create auth oidc state: provider key is required")
	}
	parseSessionKey := strings.TrimSpace(parseWrite.SessionKey)
	parseStateTokenHash := strings.TrimSpace(parseWrite.StateTokenHash)
	parseNonceTokenHash := strings.TrimSpace(parseWrite.NonceTokenHash)
	parseExpiresAt := strings.TrimSpace(parseWrite.ExpiresAt)
	if parseSessionKey == "" || parseStateTokenHash == "" || parseNonceTokenHash == "" || parseExpiresAt == "" {
		return parseAuthOIDCStateRow{}, errors.New("create auth oidc state: session key, state token hash, nonce token hash, and expires at are required")
	}
	parseReturnToURL := strings.TrimSpace(parseWrite.ReturnToURL)
	if parseReturnToURL == "" {
		parseReturnToURL = "/"
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createAuthOIDCState,
		parseProviderKey,
		parseWrite.WorkspaceID,
		parseSessionKey,
		parseStateTokenHash,
		parseNonceTokenHash,
		parseReturnToURL,
		strings.TrimSpace(parseWrite.ExpectedSubject),
		parseExpiresAt,
		"",
		parseWrite.CreatedByUserID,
		parseNow,
		parseNow,
		parseWrite.WorkspaceID,
		parseWrite.WorkspaceID,
		parseWrite.CreatedByUserID,
		parseWrite.CreatedByUserID,
	)
	if parseErr != nil {
		return parseAuthOIDCStateRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseAuthOIDCStateRow{}, errStoreSuperuserScopeMissing
	}
	parseRow, isParseFound, parseErr := parseS.parseGetAuthOIDCStateByStateTokenHash(parseStateTokenHash)
	if parseErr != nil {
		return parseAuthOIDCStateRow{}, parseErr
	}
	if !isParseFound {
		return parseAuthOIDCStateRow{}, errors.New("create auth oidc state: persisted row missing")
	}
	return parseRow, nil
}

// parseGetAuthOIDCStateByStateTokenHash resolves one OIDC state row by state token hash.
func (parseS *Store) parseGetAuthOIDCStateByStateTokenHash(parseStateTokenHash string) (parseAuthOIDCStateRow, bool, error) {
	parseStateTokenHash = strings.TrimSpace(parseStateTokenHash)
	if parseStateTokenHash == "" {
		return parseAuthOIDCStateRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getAuthOIDCStateByStateTokenHash, parseStateTokenHash)
	var parseState parseAuthOIDCStateRow
	if parseErr := parseRow.Scan(
		&parseState.ID,
		&parseState.ProviderKey,
		&parseState.WorkspaceID,
		&parseState.SessionKey,
		&parseState.StateTokenHash,
		&parseState.NonceTokenHash,
		&parseState.ReturnToURL,
		&parseState.ExpectedSubject,
		&parseState.ExpiresAt,
		&parseState.ConsumedAt,
		&parseState.CreatedByUserID,
		&parseState.CreatedAt,
		&parseState.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseAuthOIDCStateRow{}, false, nil
		}
		return parseAuthOIDCStateRow{}, false, parseErr
	}
	return parseState, true, nil
}

// parseConsumeAuthOIDCStateByStateTokenHash marks one OIDC state row consumed once and reports whether a row changed.
func (parseS *Store) parseConsumeAuthOIDCStateByStateTokenHash(parseStateTokenHash string, parseConsumedAt time.Time) (bool, error) {
	parseStateTokenHash = strings.TrimSpace(parseStateTokenHash)
	if parseStateTokenHash == "" {
		return false, nil
	}
	if parseConsumedAt.IsZero() {
		parseConsumedAt = time.Now().UTC()
	}
	parseConsumedAtText := parseConsumedAt.UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.consumeAuthOIDCStateByStateTokenHash,
		parseConsumedAtText,
		parseConsumedAtText,
		parseStateTokenHash,
	)
	if parseErr != nil {
		return false, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr != nil {
		return false, parseErr
	}
	return parseRowsAffected > 0, nil
}

// parseDeleteExpiredAuthOIDCStates deletes expired OIDC state rows and returns the deleted row count.
func (parseS *Store) parseDeleteExpiredAuthOIDCStates(parseBefore time.Time) (int64, error) {
	if parseBefore.IsZero() {
		parseBefore = time.Now().UTC()
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteExpiredAuthOIDCStates, parseBefore.UTC().Format(time.RFC3339))
	if parseErr != nil {
		return 0, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr != nil {
		return 0, parseErr
	}
	return parseRowsAffected, nil
}

// parseUpsertWorkspaceAuthPolicy persists one workspace auth-policy row and returns the stored snapshot.
func (parseS *Store) parseUpsertWorkspaceAuthPolicy(parseWrite parseWorkspaceAuthPolicyWrite) (parseWorkspaceAuthPolicyRow, error) {
	if parseWrite.WorkspaceID <= 0 {
		return parseWorkspaceAuthPolicyRow{}, errors.New("upsert workspace auth policy: workspace id is required")
	}
	parseRequiredProviderKey := parseNormalizeExternalIdentityProviderKey(parseWrite.RequiredProviderKey)
	if parseRequiredProviderKey != "" && !parseWrite.IsExternalLoginAllowed {
		return parseWorkspaceAuthPolicyRow{}, errors.New("upsert workspace auth policy: required provider requires external login allowed")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWorkspaceAuthPolicy,
		parseWrite.WorkspaceID,
		parseBoolToInt64(parseWrite.IsPasswordAllowed),
		parseBoolToInt64(parseWrite.IsExternalLoginAllowed),
		parseBoolToInt64(parseWrite.IsSSORequired),
		parseRequiredProviderKey,
		parseBoolToInt64(parseWrite.IsJITProvisioningAllowed),
		parseBoolToInt64(parseWrite.IsLocalPasswordQAModeAllowed),
		parseWrite.UpdatedByUserID,
		parseNow,
		parseWrite.WorkspaceID,
	)
	if parseErr != nil {
		return parseWorkspaceAuthPolicyRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseWorkspaceAuthPolicyRow{}, errStoreSuperuserScopeMissing
	}
	parseRow, isParseFound, parseErr := parseS.parseGetWorkspaceAuthPolicyByWorkspace(parseWrite.WorkspaceID)
	if parseErr != nil {
		return parseWorkspaceAuthPolicyRow{}, parseErr
	}
	if !isParseFound {
		return parseWorkspaceAuthPolicyRow{}, errors.New("upsert workspace auth policy: persisted row missing")
	}
	return parseRow, nil
}

// parseGetWorkspaceAuthPolicyByWorkspace returns one workspace auth-policy row when present.
func (parseS *Store) parseGetWorkspaceAuthPolicyByWorkspace(parseWorkspaceID int64) (parseWorkspaceAuthPolicyRow, bool, error) {
	if parseWorkspaceID <= 0 {
		return parseWorkspaceAuthPolicyRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getWorkspaceAuthPolicyByWorkspace, parseWorkspaceID)
	var parsePolicy parseWorkspaceAuthPolicyRow
	var parseIsPasswordAllowed int64
	var parseIsExternalLoginAllowed int64
	var parseIsSSORequired int64
	var parseIsJITProvisioningAllowed int64
	var parseIsLocalPasswordQAModeAllowed int64
	if parseErr := parseRow.Scan(
		&parsePolicy.WorkspaceID,
		&parseIsPasswordAllowed,
		&parseIsExternalLoginAllowed,
		&parseIsSSORequired,
		&parsePolicy.RequiredProviderKey,
		&parseIsJITProvisioningAllowed,
		&parseIsLocalPasswordQAModeAllowed,
		&parsePolicy.UpdatedByUserID,
		&parsePolicy.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseWorkspaceAuthPolicyRow{}, false, nil
		}
		return parseWorkspaceAuthPolicyRow{}, false, parseErr
	}
	parsePolicy.IsPasswordAllowed = parseIsPasswordAllowed != 0
	parsePolicy.IsExternalLoginAllowed = parseIsExternalLoginAllowed != 0
	parsePolicy.IsSSORequired = parseIsSSORequired != 0
	parsePolicy.IsJITProvisioningAllowed = parseIsJITProvisioningAllowed != 0
	parsePolicy.IsLocalPasswordQAModeAllowed = parseIsLocalPasswordQAModeAllowed != 0
	return parsePolicy, true, nil
}
