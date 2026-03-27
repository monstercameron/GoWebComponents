package app

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var errStoreSuperuserScopeMissing = errors.New("store superuser scope missing")

type parseSURoleWrite struct {
	RoleKey     string
	Label       string
	Description string
	IsSystem    bool
	IsEnabled   bool
	Permissions []parseSURolePermissionRow
}

type parseSURoleRow struct {
	RoleKey     string
	Label       string
	Description string
	IsSystem    bool
	IsEnabled   bool
	CreatedAt   string
	UpdatedAt   string
}

type parseSURolePermissionRow struct {
	RoleKey         string
	PermissionKey   string
	PermissionValue string
	UpdatedAt       string
}

type parseSUUserRoleRow struct {
	UserID           int64
	RoleKey          string
	AssignedByUserID int64
	CreatedAt        string
}

type parseSiteConfigWrite struct {
	ConfigKey       string
	ConfigValue     string
	ValueType       string
	Description     string
	UpdatedByUserID int64
}

type parseSiteConfigRow struct {
	ConfigKey       string
	ConfigValue     string
	ValueType       string
	Description     string
	UpdatedByUserID int64
	UpdatedAt       string
}

type parseFeatureFlagWrite struct {
	FlagKey         string
	Description     string
	IsEnabled       bool
	RolloutPercent  int64
	AudienceJSON    string
	PayloadJSON     string
	UpdatedByUserID int64
}

type parseFeatureFlagRow struct {
	FlagKey         string
	Description     string
	IsEnabled       bool
	RolloutPercent  int64
	AudienceJSON    string
	PayloadJSON     string
	UpdatedByUserID int64
	UpdatedAt       string
}

type parseWorkspaceWrite struct {
	WorkspaceKey string
	Slug         string
	Name         string
	PlanCode     string
	Status       string
	OwnerUserID  int64
	SettingsJSON string
}

type parseWorkspaceRow struct {
	ID           int64
	WorkspaceKey string
	Slug         string
	Name         string
	PlanCode     string
	Status       string
	OwnerUserID  int64
	SettingsJSON string
	CreatedAt    string
	UpdatedAt    string
}

type parseWorkspaceMembershipWrite struct {
	WorkspaceID     int64
	UserID          int64
	RoleKey         string
	Status          string
	InvitedByUserID int64
}

type parseWorkspaceMembershipRow struct {
	ID              int64
	WorkspaceID     int64
	UserID          int64
	RoleKey         string
	Status          string
	InvitedByUserID int64
	CreatedAt       string
	UpdatedAt       string
}

type parseAPIKeyWrite struct {
	KeyID       string
	WorkspaceID int64
	UserID      int64
	Label       string
	KeyPrefix   string
	SecretHash  string
	ScopesJSON  string
}

type parseAPIKeyRow struct {
	ID          int64
	KeyID       string
	WorkspaceID int64
	UserID      int64
	Label       string
	KeyPrefix   string
	SecretHash  string
	ScopesJSON  string
	LastUsedAt  string
	RevokedAt   string
	CreatedAt   string
}

type parseWebhookEndpointWrite struct {
	WorkspaceID    int64
	Label          string
	TargetURL      string
	SecretHash     string
	EventsJSON     string
	IsEnabled      bool
	LastDeliveryAt string
	FailureCount   int64
}

type parseWebhookEndpointRow struct {
	ID             int64
	WorkspaceID    int64
	Label          string
	TargetURL      string
	SecretHash     string
	EventsJSON     string
	IsEnabled      bool
	LastDeliveryAt string
	FailureCount   int64
	CreatedAt      string
	UpdatedAt      string
}

type parseAuditLogWrite struct {
	ActorUserID int64
	WorkspaceID int64
	EventType   string
	TargetType  string
	TargetID    string
	Summary     string
	PayloadJSON string
}

type parseAuditLogRow struct {
	ID          int64
	ActorUserID int64
	WorkspaceID int64
	EventType   string
	TargetType  string
	TargetID    string
	Summary     string
	PayloadJSON string
	CreatedAt   string
}

type parseSupportTicketWrite struct {
	TicketKey      string
	WorkspaceID    int64
	UserID         int64
	Status         string
	Priority       string
	Subject        string
	Body           string
	AssigneeUserID int64
	ResolutionNote string
}

type parseSupportTicketRow struct {
	ID             int64
	TicketKey      string
	WorkspaceID    int64
	UserID         int64
	Status         string
	Priority       string
	Subject        string
	Body           string
	AssigneeUserID int64
	ResolutionNote string
	CreatedAt      string
	UpdatedAt      string
}

type parseExperimentWrite struct {
	ExperimentKey string
	Name          string
	Status        string
	VariantsJSON  string
	AudienceJSON  string
	StartAt       string
	EndAt         string
}

type parseExperimentRow struct {
	ID            int64
	ExperimentKey string
	Name          string
	Status        string
	VariantsJSON  string
	AudienceJSON  string
	StartAt       string
	EndAt         string
	UpdatedAt     string
}

// parseUpsertSURole persists one superuser role and its permission set.
func (parseS *Store) parseUpsertSURole(parseWrite parseSURoleWrite) error {
	parseRoleKey := parseNormalizeSURoleKey(parseWrite.RoleKey)
	if parseRoleKey == "" {
		return errors.New("upsert su role: role key is required")
	}
	parseLabel := strings.TrimSpace(parseWrite.Label)
	if parseLabel == "" {
		parseLabel = parseRoleKey
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseTx, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parseErr
	}
	defer func() {
		_ = parseTx.Rollback()
	}()

	if _, parseErr2 := parseTx.Exec(
		parseS.queries.upsertSURole,
		parseRoleKey,
		parseLabel,
		strings.TrimSpace(parseWrite.Description),
		parseBuildBillingFlagValue(parseWrite.IsSystem),
		parseBuildBillingFlagValue(parseWrite.IsEnabled),
		parseNow,
		parseNow,
	); parseErr2 != nil {
		return parseErr2
	}
	if _, parseErr3 := parseTx.Exec(parseS.queries.deleteSURolePermissions, parseRoleKey); parseErr3 != nil {
		return parseErr3
	}
	for _, parsePermission := range parseWrite.Permissions {
		parsePermissionKey := parseNormalizeSUKey(parsePermission.PermissionKey)
		if parsePermissionKey == "" {
			continue
		}
		if _, parseErr4 := parseTx.Exec(
			parseS.queries.insertSURolePermission,
			parseRoleKey,
			parsePermissionKey,
			parseNormalizeSUValue(parsePermission.PermissionValue, "allow"),
			parseNow,
		); parseErr4 != nil {
			return parseErr4
		}
	}
	return parseTx.Commit()
}

// parseListSURoles lists configured superuser roles.
func (parseS *Store) parseListSURoles() ([]parseSURoleRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSURoles)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseRoleRows := make([]parseSURoleRow, 0)
	for parseRows.Next() {
		var parseRow parseSURoleRow
		var parseIsSystem int64
		var parseIsEnabled int64
		if parseErr2 := parseRows.Scan(
			&parseRow.RoleKey,
			&parseRow.Label,
			&parseRow.Description,
			&parseIsSystem,
			&parseIsEnabled,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsSystem = parseIsSystem != 0
		parseRow.IsEnabled = parseIsEnabled != 0
		parseRoleRows = append(parseRoleRows, parseRow)
	}
	return parseRoleRows, parseRows.Err()
}

// parseListSURolePermissions lists all role permission rows.
func (parseS *Store) parseListSURolePermissions() ([]parseSURolePermissionRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSURolePermissions)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parsePermissionRows := make([]parseSURolePermissionRow, 0)
	for parseRows.Next() {
		var parseRow parseSURolePermissionRow
		if parseErr2 := parseRows.Scan(
			&parseRow.RoleKey,
			&parseRow.PermissionKey,
			&parseRow.PermissionValue,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parsePermissionRows = append(parsePermissionRows, parseRow)
	}
	return parsePermissionRows, parseRows.Err()
}

// parseUpsertSUUserRole grants one role to one user.
func (parseS *Store) parseUpsertSUUserRole(parseUserID int64, parseRoleKey string, parseAssignedByUserID int64) error {
	parseRoleKey = parseNormalizeSURoleKey(parseRoleKey)
	if parseUserID <= 0 || parseRoleKey == "" {
		return errors.New("upsert su user role: user id and role key are required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertSUUserRole,
		parseUserID,
		parseRoleKey,
		parseAssignedByUserID,
		time.Now().UTC().Format(time.RFC3339),
		parseUserID,
		parseRoleKey,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListSUUserRoles lists all user-to-role grants.
func (parseS *Store) parseListSUUserRoles() ([]parseSUUserRoleRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSUUserRoles)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseUserRoleRows := make([]parseSUUserRoleRow, 0)
	for parseRows.Next() {
		var parseRow parseSUUserRoleRow
		if parseErr2 := parseRows.Scan(
			&parseRow.UserID,
			&parseRow.RoleKey,
			&parseRow.AssignedByUserID,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseUserRoleRows = append(parseUserRoleRows, parseRow)
	}
	return parseUserRoleRows, parseRows.Err()
}

// parseUserHasSURole reports whether one authenticated user currently holds the su role.
func (parseS *Store) parseUserHasSURole(parseUserID int64) (bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.userHasSURole, parseUserID)
	var parseExists int64
	if parseErr := parseRow.Scan(&parseExists); parseErr != nil {
		return false, parseErr
	}
	return parseExists != 0, nil
}

// parseUpsertSiteConfig persists one site configuration entry.
func (parseS *Store) parseUpsertSiteConfig(parseWrite parseSiteConfigWrite) error {
	if parseNormalizeSUKey(parseWrite.ConfigKey) == "" {
		return errors.New("upsert site config: config key is required")
	}
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertSiteConfig,
		parseNormalizeSUKey(parseWrite.ConfigKey),
		strings.TrimSpace(parseWrite.ConfigValue),
		parseNormalizeSUValue(parseWrite.ValueType, "string"),
		strings.TrimSpace(parseWrite.Description),
		parseWrite.UpdatedByUserID,
		time.Now().UTC().Format(time.RFC3339),
	)
	return parseErr
}

// parseListSiteConfigs lists site configuration entries.
func (parseS *Store) parseListSiteConfigs() ([]parseSiteConfigRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSiteConfigs)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseConfigRows := make([]parseSiteConfigRow, 0)
	for parseRows.Next() {
		var parseRow parseSiteConfigRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ConfigKey,
			&parseRow.ConfigValue,
			&parseRow.ValueType,
			&parseRow.Description,
			&parseRow.UpdatedByUserID,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseConfigRows = append(parseConfigRows, parseRow)
	}
	return parseConfigRows, parseRows.Err()
}

// parseUpsertFeatureFlag persists one feature flag definition.
func (parseS *Store) parseUpsertFeatureFlag(parseWrite parseFeatureFlagWrite) error {
	parseFlagKey := parseNormalizeSUKey(parseWrite.FlagKey)
	if parseFlagKey == "" {
		return errors.New("upsert feature flag: flag key is required")
	}
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertFeatureFlag,
		parseFlagKey,
		strings.TrimSpace(parseWrite.Description),
		parseBuildBillingFlagValue(parseWrite.IsEnabled),
		parseClampSURolloutPercent(parseWrite.RolloutPercent),
		parseNormalizeBillingJSON(parseWrite.AudienceJSON),
		parseNormalizeBillingJSON(parseWrite.PayloadJSON),
		parseWrite.UpdatedByUserID,
		time.Now().UTC().Format(time.RFC3339),
	)
	return parseErr
}

// parseListFeatureFlags lists feature flags.
func (parseS *Store) parseListFeatureFlags() ([]parseFeatureFlagRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listFeatureFlags)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseFlagRows := make([]parseFeatureFlagRow, 0)
	for parseRows.Next() {
		var parseRow parseFeatureFlagRow
		var parseIsEnabled int64
		if parseErr2 := parseRows.Scan(
			&parseRow.FlagKey,
			&parseRow.Description,
			&parseIsEnabled,
			&parseRow.RolloutPercent,
			&parseRow.AudienceJSON,
			&parseRow.PayloadJSON,
			&parseRow.UpdatedByUserID,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsEnabled = parseIsEnabled != 0
		parseFlagRows = append(parseFlagRows, parseRow)
	}
	return parseFlagRows, parseRows.Err()
}

// parseUpsertWorkspace persists one workspace row.
func (parseS *Store) parseUpsertWorkspace(parseWrite parseWorkspaceWrite) error {
	parseWorkspaceKey := parseNormalizeSUKey(parseWrite.WorkspaceKey)
	parseSlug := parseNormalizeSUKey(parseWrite.Slug)
	if parseWorkspaceKey == "" || parseSlug == "" || parseWrite.OwnerUserID <= 0 {
		return errors.New("upsert workspace: workspace key, slug, and owner user id are required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWorkspace,
		parseWorkspaceKey,
		parseSlug,
		strings.TrimSpace(parseWrite.Name),
		parseNormalizeSUValue(parseWrite.PlanCode, "free"),
		parseNormalizeSUValue(parseWrite.Status, "active"),
		parseWrite.OwnerUserID,
		parseNormalizeBillingJSON(parseWrite.SettingsJSON),
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		parseWrite.OwnerUserID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreUserMissing
	}
	return nil
}

// parseListWorkspaces lists workspaces newest-first.
func (parseS *Store) parseListWorkspaces(parseLimit int64) ([]parseWorkspaceRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaces, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseWorkspaceRows := make([]parseWorkspaceRow, 0)
	for parseRows.Next() {
		var parseRow parseWorkspaceRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceKey,
			&parseRow.Slug,
			&parseRow.Name,
			&parseRow.PlanCode,
			&parseRow.Status,
			&parseRow.OwnerUserID,
			&parseRow.SettingsJSON,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseWorkspaceRows = append(parseWorkspaceRows, parseRow)
	}
	return parseWorkspaceRows, parseRows.Err()
}

// parseUpsertWorkspaceMembership persists one workspace membership row.
func (parseS *Store) parseUpsertWorkspaceMembership(parseWrite parseWorkspaceMembershipWrite) error {
	if parseWrite.WorkspaceID <= 0 || parseWrite.UserID <= 0 {
		return errors.New("upsert workspace membership: workspace id and user id are required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWorkspaceMembership,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		parseNormalizeSUValue(parseWrite.RoleKey, "member"),
		parseNormalizeSUValue(parseWrite.Status, "active"),
		parseWrite.InvitedByUserID,
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		parseWrite.WorkspaceID,
		parseWrite.UserID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListWorkspaceMemberships lists memberships newest-first.
func (parseS *Store) parseListWorkspaceMemberships(parseLimit int64) ([]parseWorkspaceMembershipRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaceMemberships, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseMembershipRows := make([]parseWorkspaceMembershipRow, 0)
	for parseRows.Next() {
		var parseRow parseWorkspaceMembershipRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.RoleKey,
			&parseRow.Status,
			&parseRow.InvitedByUserID,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseMembershipRows = append(parseMembershipRows, parseRow)
	}
	return parseMembershipRows, parseRows.Err()
}

// parseCreateAPIKey persists one API key metadata row.
func (parseS *Store) parseCreateAPIKey(parseWrite parseAPIKeyWrite) (int64, error) {
	if strings.TrimSpace(parseWrite.KeyID) == "" || parseWrite.WorkspaceID <= 0 || parseWrite.UserID <= 0 {
		return 0, errors.New("create api key: key id, workspace id, and user id are required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createAPIKey,
		strings.TrimSpace(parseWrite.KeyID),
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		strings.TrimSpace(parseWrite.Label),
		strings.TrimSpace(parseWrite.KeyPrefix),
		strings.TrimSpace(parseWrite.SecretHash),
		parseNormalizeSUJSONArray(parseWrite.ScopesJSON),
		time.Now().UTC().Format(time.RFC3339),
		parseWrite.WorkspaceID,
		parseWrite.UserID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreSuperuserScopeMissing
	}
	return parseResult.LastInsertId()
}

// parseListAPIKeys lists API key metadata rows newest-first.
func (parseS *Store) parseListAPIKeys(parseLimit int64) ([]parseAPIKeyRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAPIKeys, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseKeyRows := make([]parseAPIKeyRow, 0)
	for parseRows.Next() {
		var parseRow parseAPIKeyRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.KeyID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.Label,
			&parseRow.KeyPrefix,
			&parseRow.SecretHash,
			&parseRow.ScopesJSON,
			&parseRow.LastUsedAt,
			&parseRow.RevokedAt,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseKeyRows = append(parseKeyRows, parseRow)
	}
	return parseKeyRows, parseRows.Err()
}

// parseRevokeAPIKey marks one API key as revoked.
func (parseS *Store) parseRevokeAPIKey(parseKeyID string) error {
	if strings.TrimSpace(parseKeyID) == "" {
		return errors.New("revoke api key: key id is required")
	}
	_, parseErr := parseS.db.Exec(parseS.queries.revokeAPIKey, time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(parseKeyID))
	return parseErr
}

// parseUpsertWebhookEndpoint persists one webhook endpoint row.
func (parseS *Store) parseUpsertWebhookEndpoint(parseWrite parseWebhookEndpointWrite) error {
	if parseWrite.WorkspaceID <= 0 || strings.TrimSpace(parseWrite.TargetURL) == "" {
		return errors.New("upsert webhook endpoint: workspace id and target url are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWebhookEndpoint,
		parseWrite.WorkspaceID,
		strings.TrimSpace(parseWrite.Label),
		strings.TrimSpace(parseWrite.TargetURL),
		strings.TrimSpace(parseWrite.SecretHash),
		parseNormalizeSUJSONArray(parseWrite.EventsJSON),
		parseBuildBillingFlagValue(parseWrite.IsEnabled),
		strings.TrimSpace(parseWrite.LastDeliveryAt),
		parseWrite.FailureCount,
		parseNow,
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

// parseListWebhookEndpoints lists webhook endpoints newest-first.
func (parseS *Store) parseListWebhookEndpoints(parseLimit int64) ([]parseWebhookEndpointRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWebhookEndpoints, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseEndpointRows := make([]parseWebhookEndpointRow, 0)
	for parseRows.Next() {
		var parseRow parseWebhookEndpointRow
		var parseIsEnabled int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.Label,
			&parseRow.TargetURL,
			&parseRow.SecretHash,
			&parseRow.EventsJSON,
			&parseIsEnabled,
			&parseRow.LastDeliveryAt,
			&parseRow.FailureCount,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsEnabled = parseIsEnabled != 0
		parseEndpointRows = append(parseEndpointRows, parseRow)
	}
	return parseEndpointRows, parseRows.Err()
}

// parseCreateAuditLog appends one immutable audit row.
func (parseS *Store) parseCreateAuditLog(parseWrite parseAuditLogWrite) (int64, error) {
	if strings.TrimSpace(parseWrite.EventType) == "" {
		return 0, errors.New("create audit log: event type is required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createAuditLog,
		parseWrite.ActorUserID,
		parseWrite.WorkspaceID,
		strings.TrimSpace(parseWrite.EventType),
		strings.TrimSpace(parseWrite.TargetType),
		strings.TrimSpace(parseWrite.TargetID),
		strings.TrimSpace(parseWrite.Summary),
		parseNormalizeBillingJSON(parseWrite.PayloadJSON),
		time.Now().UTC().Format(time.RFC3339),
		parseWrite.ActorUserID,
		parseWrite.ActorUserID,
		parseWrite.WorkspaceID,
		parseWrite.WorkspaceID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreSuperuserScopeMissing
	}
	return parseResult.LastInsertId()
}

// parseListAuditLogs lists audit logs newest-first.
func (parseS *Store) parseListAuditLogs(parseLimit int64) ([]parseAuditLogRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAuditLogs, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseAuditRows := make([]parseAuditLogRow, 0)
	for parseRows.Next() {
		var parseRow parseAuditLogRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.ActorUserID,
			&parseRow.WorkspaceID,
			&parseRow.EventType,
			&parseRow.TargetType,
			&parseRow.TargetID,
			&parseRow.Summary,
			&parseRow.PayloadJSON,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseAuditRows = append(parseAuditRows, parseRow)
	}
	return parseAuditRows, parseRows.Err()
}

// parseUpsertSupportTicket persists one support ticket row.
func (parseS *Store) parseUpsertSupportTicket(parseWrite parseSupportTicketWrite) error {
	parseTicketKey := parseNormalizeSUKey(parseWrite.TicketKey)
	if parseTicketKey == "" {
		return errors.New("upsert support ticket: ticket key is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertSupportTicket,
		parseTicketKey,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		parseNormalizeSUValue(parseWrite.Status, "open"),
		parseNormalizeSUValue(parseWrite.Priority, "normal"),
		strings.TrimSpace(parseWrite.Subject),
		strings.TrimSpace(parseWrite.Body),
		parseWrite.AssigneeUserID,
		strings.TrimSpace(parseWrite.ResolutionNote),
		parseNow,
		parseNow,
		parseWrite.WorkspaceID,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		parseWrite.UserID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListSupportTickets lists support tickets newest-first.
func (parseS *Store) parseListSupportTickets(parseLimit int64) ([]parseSupportTicketRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSupportTickets, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseTicketRows := make([]parseSupportTicketRow, 0)
	for parseRows.Next() {
		var parseRow parseSupportTicketRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.TicketKey,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.Status,
			&parseRow.Priority,
			&parseRow.Subject,
			&parseRow.Body,
			&parseRow.AssigneeUserID,
			&parseRow.ResolutionNote,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseTicketRows = append(parseTicketRows, parseRow)
	}
	return parseTicketRows, parseRows.Err()
}

// parseUpsertExperiment persists one experiment row.
func (parseS *Store) parseUpsertExperiment(parseWrite parseExperimentWrite) error {
	parseExperimentKey := parseNormalizeSUKey(parseWrite.ExperimentKey)
	if parseExperimentKey == "" {
		return errors.New("upsert experiment: experiment key is required")
	}
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertExperiment,
		parseExperimentKey,
		strings.TrimSpace(parseWrite.Name),
		parseNormalizeSUValue(parseWrite.Status, "draft"),
		parseNormalizeSUJSONArray(parseWrite.VariantsJSON),
		parseNormalizeBillingJSON(parseWrite.AudienceJSON),
		strings.TrimSpace(parseWrite.StartAt),
		strings.TrimSpace(parseWrite.EndAt),
		time.Now().UTC().Format(time.RFC3339),
	)
	return parseErr
}

// parseListExperiments lists experiments newest-first.
func (parseS *Store) parseListExperiments(parseLimit int64) ([]parseExperimentRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listExperiments, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseExperimentRows := make([]parseExperimentRow, 0)
	for parseRows.Next() {
		var parseRow parseExperimentRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.ExperimentKey,
			&parseRow.Name,
			&parseRow.Status,
			&parseRow.VariantsJSON,
			&parseRow.AudienceJSON,
			&parseRow.StartAt,
			&parseRow.EndAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseExperimentRows = append(parseExperimentRows, parseRow)
	}
	return parseExperimentRows, parseRows.Err()
}

// parseNormalizeSURoleKey normalizes one role key for superuser control-plane records.
func parseNormalizeSURoleKey(parseValue string) string {
	return parseNormalizeSUKey(parseValue)
}

// parseNormalizeSUKey normalizes one slug-like key for superuser control-plane records.
func parseNormalizeSUKey(parseValue string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseValue))
	parseNormalized = strings.ReplaceAll(parseNormalized, " ", "-")
	return parseNormalized
}

// parseNormalizeSUValue returns one fallback-backed normalized string value.
func parseNormalizeSUValue(parseValue string, parseFallback string) string {
	parseNormalized := strings.TrimSpace(parseValue)
	if parseNormalized == "" {
		return parseFallback
	}
	return parseNormalized
}

// parseNormalizeSUJSONArray ensures optional JSON arrays always persist as array literals.
func parseNormalizeSUJSONArray(parseValue string) string {
	parseNormalized := strings.TrimSpace(parseValue)
	if parseNormalized == "" {
		return "[]"
	}
	return parseNormalized
}

// parseClampSURolloutPercent clamps one rollout percentage into 0-100 bounds.
func parseClampSURolloutPercent(parseValue int64) int64 {
	if parseValue < 0 {
		return 0
	}
	if parseValue > 100 {
		return 100
	}
	return parseValue
}
