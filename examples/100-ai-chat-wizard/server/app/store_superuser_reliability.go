package app

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type parseWorkspaceSSOConfigRow struct {
	ID                  int64
	WorkspaceID         int64
	ProviderKey         string
	ProviderType        string
	OIDCIssuerURL       string
	OIDCClientID        string
	OIDCClientSecretRef string
	OIDCScopesJSON      string
	OIDCClaimsJSON      string
	SAMLEntrypoint      string
	SAMLIssuer          string
	SAMLCertificatePEM  string
	DomainsJSON         string
	IsEnabled           bool
	UpdatedAt           string
}

type parseDataRetentionPolicyRow struct {
	ID            int64
	WorkspaceID   int64
	ScopeKey      string
	RetentionDays int64
	PurgeMode     string
	LegalHoldJSON string
	UpdatedAt     string
}

type parseComplianceControlRow struct {
	ID           int64
	WorkspaceID  int64
	ControlKey   string
	FrameworkKey string
	Status       string
	OwnerUserID  int64
	EvidenceURL  string
	ReviewedAt   string
	UpdatedAt    string
}

type parseServiceLevelObjectiveRow struct {
	ID                 int64
	SLOKey             string
	ServiceName        string
	ObjectivePercent   float64
	WindowDays         int64
	ErrorBudgetMinutes int64
	StatusPageURL      string
	UpdatedAt          string
}

type parseSuperuserWorkspaceSSOConfigWrite struct {
	WorkspaceID         int64
	ProviderKey         string
	ProviderType        string
	OIDCIssuerURL       string
	OIDCClientID        string
	OIDCClientSecretRef string
	OIDCScopesJSON      string
	OIDCClaimsJSON      string
	SAMLEntrypoint      string
	SAMLIssuer          string
	SAMLCertificatePEM  string
	DomainsJSON         string
	IsEnabled           bool
}

type parseSuperuserDataRetentionPolicyWrite struct {
	WorkspaceID   int64
	ScopeKey      string
	RetentionDays int64
	PurgeMode     string
	LegalHoldJSON string
}

type parseSuperuserComplianceControlWrite struct {
	WorkspaceID  int64
	ControlKey   string
	FrameworkKey string
	Status       string
	OwnerUserID  int64
	EvidenceURL  string
	ReviewedAt   string
}

type parseSuperuserServiceLevelObjectiveWrite struct {
	SLOKey             string
	ServiceName        string
	ObjectivePercent   float64
	WindowDays         int64
	ErrorBudgetMinutes int64
	StatusPageURL      string
}

type parseSuperuserIncidentWrite struct {
	IncidentKey   string
	SLOKey        string
	Severity      string
	Status        string
	Title         string
	Summary       string
	StartedAt     string
	ResolvedAt    string
	PostmortemURL string
}

type parseSuperuserIncidentUpdateWrite struct {
	ID              int64
	IncidentID      int64
	Status          string
	Message         string
	IsPublic        bool
	PublishedAt     string
	CreatedByUserID int64
	CreatedAt       string
}

// parseListWorkspaceSSOConfigs lists workspace SSO configs newest-first.
func (parseS *Store) parseListWorkspaceSSOConfigs(parseLimit int64) ([]parseWorkspaceSSOConfigRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaceSSOConfigs, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseConfigRows := make([]parseWorkspaceSSOConfigRow, 0)
	for parseRows.Next() {
		var parseRow parseWorkspaceSSOConfigRow
		var parseIsEnabled int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.ProviderKey,
			&parseRow.ProviderType,
			&parseRow.OIDCIssuerURL,
			&parseRow.OIDCClientID,
			&parseRow.OIDCClientSecretRef,
			&parseRow.OIDCScopesJSON,
			&parseRow.OIDCClaimsJSON,
			&parseRow.SAMLEntrypoint,
			&parseRow.SAMLIssuer,
			&parseRow.SAMLCertificatePEM,
			&parseRow.DomainsJSON,
			&parseIsEnabled,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsEnabled = parseIsEnabled != 0
		parseConfigRows = append(parseConfigRows, parseRow)
	}
	return parseConfigRows, parseRows.Err()
}

// parseGetWorkspaceSSOConfigByScope resolves one workspace SSO config by workspace+provider.
func (parseS *Store) parseGetWorkspaceSSOConfigByScope(parseWorkspaceID int64, parseProviderKey string) (parseWorkspaceSSOConfigRow, bool, error) {
	parseProviderKey = parseNormalizeSUKey(parseProviderKey)
	if parseWorkspaceID <= 0 || parseProviderKey == "" {
		return parseWorkspaceSSOConfigRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getWorkspaceSSOConfigByScope, parseWorkspaceID, parseProviderKey)
	var parseConfigRow parseWorkspaceSSOConfigRow
	var parseIsEnabled int64
	if parseErr := parseRow.Scan(
		&parseConfigRow.ID,
		&parseConfigRow.WorkspaceID,
		&parseConfigRow.ProviderKey,
		&parseConfigRow.ProviderType,
		&parseConfigRow.OIDCIssuerURL,
		&parseConfigRow.OIDCClientID,
		&parseConfigRow.OIDCClientSecretRef,
		&parseConfigRow.OIDCScopesJSON,
		&parseConfigRow.OIDCClaimsJSON,
		&parseConfigRow.SAMLEntrypoint,
		&parseConfigRow.SAMLIssuer,
		&parseConfigRow.SAMLCertificatePEM,
		&parseConfigRow.DomainsJSON,
		&parseIsEnabled,
		&parseConfigRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseWorkspaceSSOConfigRow{}, false, nil
		}
		return parseWorkspaceSSOConfigRow{}, false, parseErr
	}
	parseConfigRow.IsEnabled = parseIsEnabled != 0
	return parseConfigRow, true, nil
}

// parseUpsertSuperuserWorkspaceSSOConfig stores one workspace SSO config by workspace+provider and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserWorkspaceSSOConfig(parseWrite parseSuperuserWorkspaceSSOConfigWrite) (parseWorkspaceSSOConfigRow, error) {
	parseProviderKey := parseNormalizeSUKey(parseWrite.ProviderKey)
	if parseWrite.WorkspaceID <= 0 || parseProviderKey == "" {
		return parseWorkspaceSSOConfigRow{}, errors.New("upsert superuser workspace sso config: workspace id and provider key are required")
	}
	parseProviderType := parseNormalizeWorkspaceSSOProviderType(
		parseWrite.ProviderType,
		parseProviderKey,
		parseWrite.SAMLEntrypoint,
		parseWrite.SAMLIssuer,
		parseWrite.SAMLCertificatePEM,
	)
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWorkspaceSSOConfig,
		parseWrite.WorkspaceID,
		parseProviderKey,
		parseProviderType,
		strings.TrimSpace(parseWrite.OIDCIssuerURL),
		strings.TrimSpace(parseWrite.OIDCClientID),
		strings.TrimSpace(parseWrite.OIDCClientSecretRef),
		parseNormalizeSUJSONArray(parseWrite.OIDCScopesJSON),
		parseNormalizeBillingJSON(parseWrite.OIDCClaimsJSON),
		strings.TrimSpace(parseWrite.SAMLEntrypoint),
		strings.TrimSpace(parseWrite.SAMLIssuer),
		strings.TrimSpace(parseWrite.SAMLCertificatePEM),
		parseNormalizeSUJSONArray(parseWrite.DomainsJSON),
		parseBuildBillingFlagValue(parseWrite.IsEnabled),
		parseNow,
		parseWrite.WorkspaceID,
	)
	if parseErr != nil {
		return parseWorkspaceSSOConfigRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseWorkspaceSSOConfigRow{}, errStoreSuperuserScopeMissing
	}
	parseConfigRow, hasParseConfigRow, parseErr := parseS.parseGetWorkspaceSSOConfigByScope(parseWrite.WorkspaceID, parseProviderKey)
	if parseErr != nil {
		return parseWorkspaceSSOConfigRow{}, parseErr
	}
	if !hasParseConfigRow {
		return parseWorkspaceSSOConfigRow{}, errStoreSuperuserScopeMissing
	}
	return parseConfigRow, nil
}

// parseDeleteSuperuserWorkspaceSSOConfig deletes one workspace SSO config row by workspace+provider.
func (parseS *Store) parseDeleteSuperuserWorkspaceSSOConfig(parseWorkspaceID int64, parseProviderKey string) error {
	parseProviderKey = parseNormalizeSUKey(parseProviderKey)
	if parseWorkspaceID <= 0 || parseProviderKey == "" {
		return errors.New("delete superuser workspace sso config: workspace id and provider key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteWorkspaceSSOConfig, parseWorkspaceID, parseProviderKey)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseNormalizeWorkspaceSSOProviderType normalizes one SSO provider type with compatibility inference for legacy SAML-only rows.
func parseNormalizeWorkspaceSSOProviderType(parseProviderType, parseProviderKey, parseSAMLEntrypoint, parseSAMLIssuer, parseSAMLCertificatePEM string) string {
	switch strings.TrimSpace(strings.ToLower(parseProviderType)) {
	case "oidc", "saml":
		return strings.TrimSpace(strings.ToLower(parseProviderType))
	}
	if parseNormalizeExternalIdentityProviderKey(parseProviderKey) == parseWorkspaceAuthMethodSAML {
		return "saml"
	}
	if strings.TrimSpace(parseSAMLEntrypoint) != "" || strings.TrimSpace(parseSAMLIssuer) != "" || strings.TrimSpace(parseSAMLCertificatePEM) != "" {
		return "saml"
	}
	return "oidc"
}

// parseListDataRetentionPolicies lists data-retention policies newest-first.
func (parseS *Store) parseListDataRetentionPolicies(parseLimit int64) ([]parseDataRetentionPolicyRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listDataRetentionPolicies, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parsePolicyRows := make([]parseDataRetentionPolicyRow, 0)
	for parseRows.Next() {
		var parseRow parseDataRetentionPolicyRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.ScopeKey,
			&parseRow.RetentionDays,
			&parseRow.PurgeMode,
			&parseRow.LegalHoldJSON,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parsePolicyRows = append(parsePolicyRows, parseRow)
	}
	return parsePolicyRows, parseRows.Err()
}

// parseGetDataRetentionPolicyByScope resolves one retention policy row by workspace+scope.
func (parseS *Store) parseGetDataRetentionPolicyByScope(parseWorkspaceID int64, parseScopeKey string) (parseDataRetentionPolicyRow, bool, error) {
	parseScopeKey = parseNormalizeSUKey(parseScopeKey)
	if parseWorkspaceID <= 0 || parseScopeKey == "" {
		return parseDataRetentionPolicyRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getDataRetentionPolicyByScope, parseWorkspaceID, parseScopeKey)
	var parsePolicyRow parseDataRetentionPolicyRow
	if parseErr := parseRow.Scan(
		&parsePolicyRow.ID,
		&parsePolicyRow.WorkspaceID,
		&parsePolicyRow.ScopeKey,
		&parsePolicyRow.RetentionDays,
		&parsePolicyRow.PurgeMode,
		&parsePolicyRow.LegalHoldJSON,
		&parsePolicyRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseDataRetentionPolicyRow{}, false, nil
		}
		return parseDataRetentionPolicyRow{}, false, parseErr
	}
	return parsePolicyRow, true, nil
}

// parseUpsertSuperuserDataRetentionPolicy stores one data-retention policy by workspace+scope and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserDataRetentionPolicy(parseWrite parseSuperuserDataRetentionPolicyWrite) (parseDataRetentionPolicyRow, error) {
	parseScopeKey := parseNormalizeSUKey(parseWrite.ScopeKey)
	if parseWrite.WorkspaceID <= 0 || parseScopeKey == "" {
		return parseDataRetentionPolicyRow{}, errors.New("upsert superuser data retention policy: workspace id and scope key are required")
	}
	parseRetentionDays := parseWrite.RetentionDays
	if parseRetentionDays < 0 {
		parseRetentionDays = 0
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertDataRetentionPolicy,
		parseWrite.WorkspaceID,
		parseScopeKey,
		parseRetentionDays,
		parseNormalizeSuperuserPurgeMode(parseWrite.PurgeMode),
		parseNormalizeBillingJSON(parseWrite.LegalHoldJSON),
		parseNow,
		parseWrite.WorkspaceID,
	)
	if parseErr != nil {
		return parseDataRetentionPolicyRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseDataRetentionPolicyRow{}, errStoreSuperuserScopeMissing
	}
	parsePolicyRow, hasParsePolicyRow, parseErr := parseS.parseGetDataRetentionPolicyByScope(parseWrite.WorkspaceID, parseScopeKey)
	if parseErr != nil {
		return parseDataRetentionPolicyRow{}, parseErr
	}
	if !hasParsePolicyRow {
		return parseDataRetentionPolicyRow{}, errStoreSuperuserScopeMissing
	}
	return parsePolicyRow, nil
}

// parseDeleteSuperuserDataRetentionPolicy deletes one data-retention policy by workspace+scope.
func (parseS *Store) parseDeleteSuperuserDataRetentionPolicy(parseWorkspaceID int64, parseScopeKey string) error {
	parseScopeKey = parseNormalizeSUKey(parseScopeKey)
	if parseWorkspaceID <= 0 || parseScopeKey == "" {
		return errors.New("delete superuser data retention policy: workspace id and scope key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteDataRetentionPolicy, parseWorkspaceID, parseScopeKey)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListComplianceControls lists compliance-control rows newest-first.
func (parseS *Store) parseListComplianceControls(parseLimit int64) ([]parseComplianceControlRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listComplianceControls, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseControlRows := make([]parseComplianceControlRow, 0)
	for parseRows.Next() {
		var parseRow parseComplianceControlRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.ControlKey,
			&parseRow.FrameworkKey,
			&parseRow.Status,
			&parseRow.OwnerUserID,
			&parseRow.EvidenceURL,
			&parseRow.ReviewedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseControlRows = append(parseControlRows, parseRow)
	}
	return parseControlRows, parseRows.Err()
}

// parseGetComplianceControlByScope resolves one compliance-control row by workspace+control+framework.
func (parseS *Store) parseGetComplianceControlByScope(parseWorkspaceID int64, parseControlKey string, parseFrameworkKey string) (parseComplianceControlRow, bool, error) {
	parseControlKey = parseNormalizeSUKey(parseControlKey)
	parseFrameworkKey = parseNormalizeSUKey(parseFrameworkKey)
	if parseWorkspaceID <= 0 || parseControlKey == "" || parseFrameworkKey == "" {
		return parseComplianceControlRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getComplianceControlByScope, parseWorkspaceID, parseControlKey, parseFrameworkKey)
	var parseControlRow parseComplianceControlRow
	if parseErr := parseRow.Scan(
		&parseControlRow.ID,
		&parseControlRow.WorkspaceID,
		&parseControlRow.ControlKey,
		&parseControlRow.FrameworkKey,
		&parseControlRow.Status,
		&parseControlRow.OwnerUserID,
		&parseControlRow.EvidenceURL,
		&parseControlRow.ReviewedAt,
		&parseControlRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseComplianceControlRow{}, false, nil
		}
		return parseComplianceControlRow{}, false, parseErr
	}
	return parseControlRow, true, nil
}

// parseUpsertSuperuserComplianceControl stores one compliance-control row by workspace+control+framework and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserComplianceControl(parseWrite parseSuperuserComplianceControlWrite) (parseComplianceControlRow, error) {
	parseControlKey := parseNormalizeSUKey(parseWrite.ControlKey)
	parseFrameworkKey := parseNormalizeSUKey(parseWrite.FrameworkKey)
	if parseWrite.WorkspaceID <= 0 || parseControlKey == "" || parseFrameworkKey == "" {
		return parseComplianceControlRow{}, errors.New("upsert superuser compliance control: workspace id, control key, and framework key are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertComplianceControl,
		parseWrite.WorkspaceID,
		parseControlKey,
		parseFrameworkKey,
		parseNormalizeSuperuserComplianceStatus(parseWrite.Status),
		parseWrite.OwnerUserID,
		strings.TrimSpace(parseWrite.EvidenceURL),
		strings.TrimSpace(parseWrite.ReviewedAt),
		parseNow,
		parseWrite.WorkspaceID,
		parseWrite.OwnerUserID,
		parseWrite.OwnerUserID,
	)
	if parseErr != nil {
		return parseComplianceControlRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseComplianceControlRow{}, errStoreSuperuserScopeMissing
	}
	parseControlRow, hasParseControlRow, parseErr := parseS.parseGetComplianceControlByScope(parseWrite.WorkspaceID, parseControlKey, parseFrameworkKey)
	if parseErr != nil {
		return parseComplianceControlRow{}, parseErr
	}
	if !hasParseControlRow {
		return parseComplianceControlRow{}, errStoreSuperuserScopeMissing
	}
	return parseControlRow, nil
}

// parseDeleteSuperuserComplianceControl deletes one compliance-control row by workspace+control+framework.
func (parseS *Store) parseDeleteSuperuserComplianceControl(parseWorkspaceID int64, parseControlKey string, parseFrameworkKey string) error {
	parseControlKey = parseNormalizeSUKey(parseControlKey)
	parseFrameworkKey = parseNormalizeSUKey(parseFrameworkKey)
	if parseWorkspaceID <= 0 || parseControlKey == "" || parseFrameworkKey == "" {
		return errors.New("delete superuser compliance control: workspace id, control key, and framework key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteComplianceControl, parseWorkspaceID, parseControlKey, parseFrameworkKey)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListServiceLevelObjectives lists SLO rows newest-first.
func (parseS *Store) parseListServiceLevelObjectives(parseLimit int64) ([]parseServiceLevelObjectiveRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listServiceLevelObjectives, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseSLORows := make([]parseServiceLevelObjectiveRow, 0)
	for parseRows.Next() {
		var parseRow parseServiceLevelObjectiveRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.SLOKey,
			&parseRow.ServiceName,
			&parseRow.ObjectivePercent,
			&parseRow.WindowDays,
			&parseRow.ErrorBudgetMinutes,
			&parseRow.StatusPageURL,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseSLORows = append(parseSLORows, parseRow)
	}
	return parseSLORows, parseRows.Err()
}

// parseGetServiceLevelObjectiveByKey resolves one SLO row by key.
func (parseS *Store) parseGetServiceLevelObjectiveByKey(parseSLOKey string) (parseServiceLevelObjectiveRow, bool, error) {
	parseSLOKey = parseNormalizeSUKey(parseSLOKey)
	if parseSLOKey == "" {
		return parseServiceLevelObjectiveRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getServiceLevelObjectiveByKey, parseSLOKey)
	var parseSLORow parseServiceLevelObjectiveRow
	if parseErr := parseRow.Scan(
		&parseSLORow.ID,
		&parseSLORow.SLOKey,
		&parseSLORow.ServiceName,
		&parseSLORow.ObjectivePercent,
		&parseSLORow.WindowDays,
		&parseSLORow.ErrorBudgetMinutes,
		&parseSLORow.StatusPageURL,
		&parseSLORow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseServiceLevelObjectiveRow{}, false, nil
		}
		return parseServiceLevelObjectiveRow{}, false, parseErr
	}
	return parseSLORow, true, nil
}

// parseUpsertSuperuserServiceLevelObjective stores one SLO row by key and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserServiceLevelObjective(parseWrite parseSuperuserServiceLevelObjectiveWrite) (parseServiceLevelObjectiveRow, error) {
	parseSLOKey := parseNormalizeSUKey(parseWrite.SLOKey)
	if parseSLOKey == "" {
		return parseServiceLevelObjectiveRow{}, errors.New("upsert superuser service level objective: slo key is required")
	}
	parseWindowDays := parseWrite.WindowDays
	if parseWindowDays <= 0 {
		parseWindowDays = 30
	}
	parseErrorBudgetMinutes := parseWrite.ErrorBudgetMinutes
	if parseErrorBudgetMinutes < 0 {
		parseErrorBudgetMinutes = 0
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertServiceLevelObjective,
		parseSLOKey,
		parseNormalizeSUValue(parseWrite.ServiceName, parseSLOKey),
		parseClampSuperuserObjectivePercent(parseWrite.ObjectivePercent),
		parseWindowDays,
		parseErrorBudgetMinutes,
		strings.TrimSpace(parseWrite.StatusPageURL),
		parseNow,
	); parseErr != nil {
		return parseServiceLevelObjectiveRow{}, parseErr
	}
	parseSLORow, hasParseSLORow, parseErr := parseS.parseGetServiceLevelObjectiveByKey(parseSLOKey)
	if parseErr != nil {
		return parseServiceLevelObjectiveRow{}, parseErr
	}
	if !hasParseSLORow {
		return parseServiceLevelObjectiveRow{}, errStoreSuperuserScopeMissing
	}
	return parseSLORow, nil
}

// parseDeleteSuperuserServiceLevelObjective deletes one SLO row by key.
func (parseS *Store) parseDeleteSuperuserServiceLevelObjective(parseSLOKey string) error {
	parseSLOKey = parseNormalizeSUKey(parseSLOKey)
	if parseSLOKey == "" {
		return errors.New("delete superuser service level objective: slo key is required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteServiceLevelObjective, parseSLOKey)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseGetIncidentByKey resolves one incident row by incident key.
func (parseS *Store) parseGetIncidentByKey(parseIncidentKey string) (parseIncidentRow, bool, error) {
	parseIncidentKey = parseNormalizeSUKey(parseIncidentKey)
	if parseIncidentKey == "" {
		return parseIncidentRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getIncidentByKey, parseIncidentKey)
	var parseIncident parseIncidentRow
	if parseErr := parseRow.Scan(
		&parseIncident.ID,
		&parseIncident.IncidentKey,
		&parseIncident.SLOKey,
		&parseIncident.Severity,
		&parseIncident.Status,
		&parseIncident.Title,
		&parseIncident.Summary,
		&parseIncident.StartedAt,
		&parseIncident.ResolvedAt,
		&parseIncident.PostmortemURL,
		&parseIncident.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseIncidentRow{}, false, nil
		}
		return parseIncidentRow{}, false, parseErr
	}
	return parseIncident, true, nil
}

// parseUpsertSuperuserIncident stores one incident row by incident key and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserIncident(parseWrite parseSuperuserIncidentWrite) (parseIncidentRow, error) {
	parseIncidentKey := parseNormalizeSUKey(parseWrite.IncidentKey)
	parseSLOKey := parseNormalizeSUKey(parseWrite.SLOKey)
	if parseIncidentKey == "" || parseSLOKey == "" {
		return parseIncidentRow{}, errors.New("upsert superuser incident: incident key and slo key are required")
	}
	parseStatus := parseNormalizeSuperuserIncidentStatus(parseWrite.Status)
	parseResolvedAt := strings.TrimSpace(parseWrite.ResolvedAt)
	if parseResolvedAt == "" && parseHasAdminIncidentResolvedStatus(parseStatus) {
		parseResolvedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertIncident,
		parseIncidentKey,
		parseSLOKey,
		parseNormalizeSuperuserIncidentSeverity(parseWrite.Severity),
		parseStatus,
		parseNormalizeSUValue(strings.TrimSpace(parseWrite.Title), parseIncidentKey),
		strings.TrimSpace(parseWrite.Summary),
		parseNormalizeSUValue(strings.TrimSpace(parseWrite.StartedAt), parseNow),
		parseResolvedAt,
		strings.TrimSpace(parseWrite.PostmortemURL),
		parseNow,
		parseSLOKey,
	)
	if parseErr != nil {
		return parseIncidentRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseIncidentRow{}, errStoreSuperuserScopeMissing
	}
	parseIncidentResultRow, hasParseIncidentRow, parseErr := parseS.parseGetIncidentByKey(parseIncidentKey)
	if parseErr != nil {
		return parseIncidentResultRow, parseErr
	}
	if !hasParseIncidentRow {
		return parseIncidentResultRow, errStoreSuperuserScopeMissing
	}
	return parseIncidentResultRow, nil
}

// parseDeleteSuperuserIncident deletes one incident row by incident key.
func (parseS *Store) parseDeleteSuperuserIncident(parseIncidentKey string) error {
	parseIncidentKey = parseNormalizeSUKey(parseIncidentKey)
	if parseIncidentKey == "" {
		return errors.New("delete superuser incident: incident key is required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteIncident, parseIncidentKey)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseGetIncidentUpdateByID resolves one incident-update row by id.
func (parseS *Store) parseGetIncidentUpdateByID(parseIncidentUpdateID int64) (parseIncidentUpdateRow, bool, error) {
	if parseIncidentUpdateID <= 0 {
		return parseIncidentUpdateRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getIncidentUpdateByID, parseIncidentUpdateID)
	var parseUpdateRow parseIncidentUpdateRow
	var parseIsPublic int64
	if parseErr := parseRow.Scan(
		&parseUpdateRow.ID,
		&parseUpdateRow.IncidentID,
		&parseUpdateRow.Status,
		&parseUpdateRow.Message,
		&parseIsPublic,
		&parseUpdateRow.PublishedAt,
		&parseUpdateRow.CreatedByUserID,
		&parseUpdateRow.CreatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseIncidentUpdateRow{}, false, nil
		}
		return parseIncidentUpdateRow{}, false, parseErr
	}
	parseUpdateRow.IsPublic = parseIsPublic != 0
	return parseUpdateRow, true, nil
}

// parseUpsertSuperuserIncidentUpdate stores one incident-update row by id and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserIncidentUpdate(parseWrite parseSuperuserIncidentUpdateWrite) (parseIncidentUpdateRow, error) {
	if parseWrite.IncidentID <= 0 || parseWrite.CreatedByUserID <= 0 {
		return parseIncidentUpdateRow{}, errors.New("upsert superuser incident update: incident id and created-by user id are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parsePublishedAt := strings.TrimSpace(parseWrite.PublishedAt)
	if parsePublishedAt == "" && parseWrite.IsPublic {
		parsePublishedAt = parseNow
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertIncidentUpdate,
		parseWrite.ID,
		parseWrite.IncidentID,
		parseNormalizeSuperuserIncidentStatus(parseWrite.Status),
		parseNormalizeSUValue(strings.TrimSpace(parseWrite.Message), "Incident update"),
		parseBuildBillingFlagValue(parseWrite.IsPublic),
		parsePublishedAt,
		parseWrite.CreatedByUserID,
		parseNormalizeSUValue(strings.TrimSpace(parseWrite.CreatedAt), parseNow),
		parseWrite.IncidentID,
		parseWrite.CreatedByUserID,
		parseWrite.CreatedByUserID,
	)
	if parseErr != nil {
		return parseIncidentUpdateRow{}, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return parseIncidentUpdateRow{}, errStoreSuperuserScopeMissing
	}
	parseIncidentUpdateID := parseWrite.ID
	if parseIncidentUpdateID <= 0 {
		parseIncidentUpdateID, parseErr = parseResult.LastInsertId()
		if parseErr != nil {
			return parseIncidentUpdateRow{}, parseErr
		}
	}
	parseUpdateRow, hasParseUpdateRow, parseErr := parseS.parseGetIncidentUpdateByID(parseIncidentUpdateID)
	if parseErr != nil {
		return parseIncidentUpdateRow{}, parseErr
	}
	if !hasParseUpdateRow {
		return parseIncidentUpdateRow{}, errStoreSuperuserScopeMissing
	}
	return parseUpdateRow, nil
}

// parseDeleteSuperuserIncidentUpdate deletes one incident-update row by id.
func (parseS *Store) parseDeleteSuperuserIncidentUpdate(parseIncidentUpdateID int64) error {
	if parseIncidentUpdateID <= 0 {
		return errors.New("delete superuser incident update: incident update id is required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteIncidentUpdate, parseIncidentUpdateID)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseNormalizeSuperuserPurgeMode normalizes one retention purge mode for superuser policy controls.
func parseNormalizeSuperuserPurgeMode(parseMode string) string {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "delete", "archive", "anonymize":
		return strings.TrimSpace(strings.ToLower(parseMode))
	default:
		return "delete"
	}
}

// parseNormalizeSuperuserComplianceStatus normalizes one compliance status for superuser controls.
func parseNormalizeSuperuserComplianceStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "planned", "in_progress", "implemented", "verified", "waived":
		return strings.TrimSpace(strings.ToLower(parseStatus))
	default:
		return "planned"
	}
}

// parseClampSuperuserObjectivePercent clamps one SLO target percentage into inclusive 0-100 bounds.
func parseClampSuperuserObjectivePercent(parseObjective float64) float64 {
	if parseObjective < 0 {
		return 0
	}
	if parseObjective > 100 {
		return 100
	}
	return parseObjective
}

// parseNormalizeSuperuserIncidentSeverity normalizes one incident severity value.
func parseNormalizeSuperuserIncidentSeverity(parseSeverity string) string {
	switch strings.TrimSpace(strings.ToLower(parseSeverity)) {
	case "minor", "major", "critical":
		return strings.TrimSpace(strings.ToLower(parseSeverity))
	default:
		return "minor"
	}
}

// parseNormalizeSuperuserIncidentStatus normalizes one incident or incident-update lifecycle status.
func parseNormalizeSuperuserIncidentStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "open", "investigating", "identified", "mitigated", "monitoring", "resolved", "closed":
		return strings.TrimSpace(strings.ToLower(parseStatus))
	default:
		return "investigating"
	}
}
