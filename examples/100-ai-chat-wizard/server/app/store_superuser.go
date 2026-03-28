package app

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errStoreSuperuserScopeMissing = errors.New("store superuser scope missing")

const parseWorkspaceJobSuppressionErrorPrefix = "workspace suspended:"

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

type parseWorkspaceInvitationWrite struct {
	WorkspaceID         int64
	Email               string
	RoleKey             string
	InvitationTokenHash string
	InvitedByUserID     int64
	Status              string
	ExpiresAt           string
	AcceptedAt          string
}

type parseWorkspaceInvitationRow struct {
	ID                  int64
	WorkspaceID         int64
	Email               string
	RoleKey             string
	InvitationTokenHash string
	InvitedByUserID     int64
	Status              string
	ExpiresAt           string
	AcceptedAt          string
	CreatedAt           string
	UpdatedAt           string
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

type parseWebhookDeliveryWrite struct {
	EndpointID         int64
	EventType          string
	DeliveryKey        string
	RequestHeadersJSON string
	RequestBodyJSON    string
	ResponseStatus     int64
	ResponseBody       string
	AttemptCount       int64
	DeliveredAt        string
	FailedAt           string
	NextRetryAt        string
}

type parseWebhookDeliveryRow struct {
	ID                 int64
	EndpointID         int64
	EventType          string
	DeliveryKey        string
	RequestHeadersJSON string
	RequestBodyJSON    string
	ResponseStatus     int64
	ResponseBody       string
	AttemptCount       int64
	DeliveredAt        string
	FailedAt           string
	NextRetryAt        string
	CreatedAt          string
	UpdatedAt          string
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

type parseSupportTicketMessageWrite struct {
	TicketID     int64
	AuthorUserID int64
	MessageType  string
	Body         string
	IsInternal   bool
}

type parseSupportTicketMessageRow struct {
	ID           int64
	TicketID     int64
	AuthorUserID int64
	MessageType  string
	Body         string
	IsInternal   bool
	CreatedAt    string
	UpdatedAt    string
}

type parseIncidentUpdateWrite struct {
	IncidentID      int64
	Status          string
	Message         string
	IsPublic        bool
	PublishedAt     string
	CreatedByUserID int64
}

type parseIncidentUpdateRow struct {
	ID              int64
	IncidentID      int64
	Status          string
	Message         string
	IsPublic        bool
	PublishedAt     string
	CreatedByUserID int64
	CreatedAt       string
}

type parseNotificationOutboxWrite struct {
	WorkspaceID     int64
	UserID          int64
	NotificationKey string
	ChannelKey      string
	TemplateKey     string
	Status          string
	Subject         string
	BodyText        string
	PayloadJSON     string
	DedupeKey       string
	ScheduledAt     string
	SentAt          string
	FailedAt        string
	ErrorMessage    string
}

type parseNotificationOutboxRow struct {
	ID              int64
	WorkspaceID     int64
	UserID          int64
	NotificationKey string
	ChannelKey      string
	TemplateKey     string
	Status          string
	Subject         string
	BodyText        string
	PayloadJSON     string
	DedupeKey       string
	ScheduledAt     string
	SentAt          string
	FailedAt        string
	ErrorMessage    string
	CreatedAt       string
	UpdatedAt       string
}

type parseBackgroundJobWrite struct {
	JobKey       string
	JobType      string
	QueueKey     string
	Status       string
	AttemptCount int64
	MaxAttempts  int64
	PayloadJSON  string
	RunAfter     string
	StartedAt    string
	FinishedAt   string
	ErrorMessage string
}

type parseBackgroundJobRow struct {
	ID           int64
	JobKey       string
	JobType      string
	QueueKey     string
	Status       string
	AttemptCount int64
	MaxAttempts  int64
	PayloadJSON  string
	RunAfter     string
	StartedAt    string
	FinishedAt   string
	ErrorMessage string
	CreatedAt    string
	UpdatedAt    string
}

type parseWorkspaceSuspendOperationalResult struct {
	RevokedAPIKeys           int64
	DisabledWebhookEndpoints int64
	SuppressedBackgroundJobs int64
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

type parseBillingPlanOverageRow struct {
	ID                int64
	PlanCode          string
	MeterKey          string
	IncludedUnits     int64
	SoftLimitUnits    int64
	HardLimitUnits    int64
	OverageUnitSize   int64
	OveragePriceCents int64
	BillingInterval   string
	UpdatedAt         string
}

type parseBillingQuotaPolicyRow struct {
	ID              int64
	PlanCode        string
	QuotaKey        string
	SoftLimitValue  int64
	HardLimitValue  int64
	ResetInterval   string
	EnforcementMode string
	UpdatedAt       string
}

type parseBillingUpgradeTriggerRow struct {
	ID               int64
	PlanCode         string
	TriggerKey       string
	ThresholdPercent int64
	UpgradePlanCode  string
	Message          string
	CTALabel         string
	CTAURL           string
	IsEnabled        bool
	UpdatedAt        string
}

type parseIncidentRow struct {
	ID            int64
	IncidentKey   string
	SLOKey        string
	Severity      string
	Status        string
	Title         string
	Summary       string
	StartedAt     string
	ResolvedAt    string
	PostmortemURL string
	UpdatedAt     string
}

type parseWorkspaceCostGuardrailRow struct {
	ID                     int64
	WorkspaceID            int64
	GuardrailKey           string
	DailyBudgetCents       int64
	MonthlyBudgetCents     int64
	MaxCostPerRequestCents int64
	AlertThresholdPercent  int64
	ActionMode             string
	UpdatedAt              string
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

// parseGetWorkspaceByID resolves one workspace row by id.
func (parseS *Store) parseGetWorkspaceByID(parseWorkspaceID int64) (parseWorkspaceRow, bool, error) {
	if parseWorkspaceID <= 0 {
		return parseWorkspaceRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getWorkspaceByID, parseWorkspaceID)
	var parseWorkspace parseWorkspaceRow
	if parseErr := parseRow.Scan(
		&parseWorkspace.ID,
		&parseWorkspace.WorkspaceKey,
		&parseWorkspace.Slug,
		&parseWorkspace.Name,
		&parseWorkspace.PlanCode,
		&parseWorkspace.Status,
		&parseWorkspace.OwnerUserID,
		&parseWorkspace.SettingsJSON,
		&parseWorkspace.CreatedAt,
		&parseWorkspace.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseWorkspaceRow{}, false, nil
		}
		return parseWorkspaceRow{}, false, parseErr
	}
	return parseWorkspace, true, nil
}

// parseSetWorkspaceStatusByID updates one workspace status by id while preserving existing workspace fields.
func (parseS *Store) parseSetWorkspaceStatusByID(parseWorkspaceID int64, parseStatus string) error {
	parseWorkspace, isParseFound, parseErr := parseS.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		return parseErr
	}
	if !isParseFound {
		return errStoreSuperuserScopeMissing
	}
	parseWorkspace.Status = strings.TrimSpace(parseStatus)
	if parseWorkspace.Status == "" {
		parseWorkspace.Status = "active"
	}
	return parseS.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: parseWorkspace.WorkspaceKey,
		Slug:         parseWorkspace.Slug,
		Name:         parseWorkspace.Name,
		PlanCode:     parseWorkspace.PlanCode,
		Status:       parseWorkspace.Status,
		OwnerUserID:  parseWorkspace.OwnerUserID,
		SettingsJSON: parseWorkspace.SettingsJSON,
	})
}

// parseIsWorkspaceOperationalStatus reports whether one workspace status allows runtime operations.
func parseIsWorkspaceOperationalStatus(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "", "active":
		return true
	default:
		return false
	}
}

// parseRequireWorkspaceOperational requires one existing active workspace before write-side operations proceed.
func (parseS *Store) parseRequireWorkspaceOperational(parseWorkspaceID int64) error {
	if parseWorkspaceID <= 0 {
		return nil
	}
	parseWorkspace, isParseFound, parseErr := parseS.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		return parseErr
	}
	if !isParseFound {
		return errStoreSuperuserScopeMissing
	}
	if !parseIsWorkspaceOperationalStatus(parseWorkspace.Status) {
		return errStoreWorkspaceSuspended
	}
	return nil
}

// parseRequireUserOperational requires one existing runtime-operational user before user-scoped write-side operations proceed.
func (parseS *Store) parseRequireUserOperational(parseUserID int64) error {
	if parseUserID <= 0 {
		return nil
	}
	isParseDisabled, parseErr := parseS.parseIsUserAccessDisabled(parseUserID)
	if parseErr != nil {
		return parseErr
	}
	if isParseDisabled {
		return errStoreUserDisabled
	}
	parseBlockCount, parseErr := parseS.parseCountUserAuthBlocksByUser(parseUserID)
	if parseErr != nil {
		return parseErr
	}
	if parseBlockCount > 0 {
		return errStoreUserAuthBlocked
	}
	return nil
}

// parseGetWebhookEndpointWorkspaceID resolves one webhook endpoint id into its workspace scope.
func (parseS *Store) parseGetWebhookEndpointWorkspaceID(parseEndpointID int64) (int64, bool, error) {
	if parseEndpointID <= 0 {
		return 0, false, nil
	}
	parseRow := parseS.db.QueryRow("SELECT workspace_id FROM webhook_endpoints WHERE id = ?", parseEndpointID)
	var parseWorkspaceID int64
	if parseErr := parseRow.Scan(&parseWorkspaceID); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, parseErr
	}
	return parseWorkspaceID, true, nil
}

// parseGetSupportTicketScopeByID resolves one support-ticket id into workspace and user scope values.
func (parseS *Store) parseGetSupportTicketScopeByID(parseTicketID int64) (int64, int64, bool, error) {
	if parseTicketID <= 0 {
		return 0, 0, false, nil
	}
	parseRow := parseS.db.QueryRow("SELECT workspace_id, user_id FROM support_tickets WHERE id = ?", parseTicketID)
	var parseWorkspaceID sql.NullInt64
	var parseUserID sql.NullInt64
	if parseErr := parseRow.Scan(&parseWorkspaceID, &parseUserID); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return 0, 0, false, nil
		}
		return 0, 0, false, parseErr
	}
	return parseWorkspaceID.Int64, parseUserID.Int64, true, nil
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

// parseListWorkspaceMembershipsByUser lists all memberships for one user newest-first.
func (parseS *Store) parseListWorkspaceMembershipsByUser(parseUserID int64) ([]parseWorkspaceMembershipRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaceMembershipsByUser, parseUserID)
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

// parseCountWorkspaceMembershipsByUser counts membership rows for one user id.
func (parseS *Store) parseCountWorkspaceMembershipsByUser(parseUserID int64) (int64, error) {
	if parseUserID <= 0 {
		return 0, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.countWorkspaceMembershipsByUser, parseUserID)
	var parseCount int64
	if parseErr := parseRow.Scan(&parseCount); parseErr != nil {
		return 0, parseErr
	}
	return parseCount, nil
}

// parseCountActiveWorkspaceMembershipsByUser counts active memberships in active workspaces for one user id.
func (parseS *Store) parseCountActiveWorkspaceMembershipsByUser(parseUserID int64) (int64, error) {
	if parseUserID <= 0 {
		return 0, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.countActiveWorkspaceMembershipsByUser, parseUserID)
	var parseCount int64
	if parseErr := parseRow.Scan(&parseCount); parseErr != nil {
		return 0, parseErr
	}
	return parseCount, nil
}

// parseListWorkspaceMembershipsByWorkspace lists all memberships for one workspace newest-first.
func (parseS *Store) parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID int64) ([]parseWorkspaceMembershipRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaceMembershipsByWorkspace, parseWorkspaceID)
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

// parseListAuthSessions lists auth sessions newest-first.
func (parseS *Store) parseListAuthSessions(parseLimit int64) ([]parseAuthSessionRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAuthSessions, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseSessionRows := make([]parseAuthSessionRow, 0)
	for parseRows.Next() {
		var parseRow parseAuthSessionRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.UserID,
			&parseRow.SessionID,
			&parseRow.TokenVersion,
			&parseRow.RefreshTokenHash,
			&parseRow.UserAgent,
			&parseRow.IPAddress,
			&parseRow.LastSeenAt,
			&parseRow.ExpiresAt,
			&parseRow.RevokedAt,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseSessionRows = append(parseSessionRows, parseRow)
	}
	return parseSessionRows, parseRows.Err()
}

// parseUpsertWorkspaceInvitation persists one workspace invitation row.
func (parseS *Store) parseUpsertWorkspaceInvitation(parseWrite parseWorkspaceInvitationWrite) error {
	if parseWrite.WorkspaceID <= 0 || strings.TrimSpace(parseWrite.Email) == "" || strings.TrimSpace(parseWrite.InvitationTokenHash) == "" || strings.TrimSpace(parseWrite.ExpiresAt) == "" {
		return errors.New("upsert workspace invitation: workspace id, email, invitation token hash, and expires at are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWorkspaceInvitation,
		parseWrite.WorkspaceID,
		parseNormalizeAuthEmail(parseWrite.Email),
		parseNormalizeSUValue(parseWrite.RoleKey, "member"),
		strings.TrimSpace(parseWrite.InvitationTokenHash),
		parseWrite.InvitedByUserID,
		parseNormalizeSUValue(parseWrite.Status, "pending"),
		strings.TrimSpace(parseWrite.ExpiresAt),
		strings.TrimSpace(parseWrite.AcceptedAt),
		parseNow,
		parseNow,
		parseWrite.WorkspaceID,
		parseWrite.InvitedByUserID,
		parseWrite.InvitedByUserID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListWorkspaceInvitations lists workspace invitations newest-first.
func (parseS *Store) parseListWorkspaceInvitations(parseLimit int64) ([]parseWorkspaceInvitationRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaceInvitations, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseInvitationRows := make([]parseWorkspaceInvitationRow, 0)
	for parseRows.Next() {
		var parseRow parseWorkspaceInvitationRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.Email,
			&parseRow.RoleKey,
			&parseRow.InvitationTokenHash,
			&parseRow.InvitedByUserID,
			&parseRow.Status,
			&parseRow.ExpiresAt,
			&parseRow.AcceptedAt,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseInvitationRows = append(parseInvitationRows, parseRow)
	}
	return parseInvitationRows, parseRows.Err()
}

// parseCreateAPIKey persists one API key metadata row.
func (parseS *Store) parseCreateAPIKey(parseWrite parseAPIKeyWrite) (int64, error) {
	if strings.TrimSpace(parseWrite.KeyID) == "" || parseWrite.WorkspaceID <= 0 || parseWrite.UserID <= 0 {
		return 0, errors.New("create api key: key id, workspace id, and user id are required")
	}
	if parseErr := parseS.parseRequireUserOperational(parseWrite.UserID); parseErr != nil {
		return 0, parseErr
	}
	if parseErr := parseS.parseRequireWorkspaceOperational(parseWrite.WorkspaceID); parseErr != nil {
		return 0, parseErr
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

// parseRevokeAPIKeysByUser marks one user's active API keys revoked and returns the number of updated rows.
func (parseS *Store) parseRevokeAPIKeysByUser(parseUserID int64) (int64, error) {
	if parseUserID <= 0 {
		return 0, nil
	}
	parseResult, parseErr := parseS.db.Exec(
		`UPDATE api_keys
		 SET revoked_at = ?
		 WHERE user_id = ?
		   AND COALESCE(TRIM(revoked_at), '') = ''`,
		time.Now().UTC().Format(time.RFC3339),
		parseUserID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr != nil {
		return 0, parseErr
	}
	return parseRowsAffected, nil
}

// parseBuildWorkspaceJobSuppressionError builds one stable suspension marker used to suppress and optionally restore workspace jobs.
func parseBuildWorkspaceJobSuppressionError(parseReason string) string {
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		parseReason = "workspace suspended"
	}
	return parseWorkspaceJobSuppressionErrorPrefix + " " + parseReason
}

// parseApplyWorkspaceSuspendOperationalEffects applies API-key, webhook, and background-job suppression for one suspended workspace transactionally.
func (parseS *Store) parseApplyWorkspaceSuspendOperationalEffects(parseWorkspaceID int64, parseReason string) (parseWorkspaceSuspendOperationalResult, error) {
	if parseWorkspaceID <= 0 {
		return parseWorkspaceSuspendOperationalResult{}, errors.New("apply workspace suspend operational effects: workspace id is required")
	}
	parseQueueKey := fmt.Sprintf("workspace:%d", parseWorkspaceID)
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		parseReason = "workspace suspended"
	}
	parseSuppressionError := parseBuildWorkspaceJobSuppressionError(parseReason)

	parseTX, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	defer func() {
		if parseErr != nil {
			_ = parseTX.Rollback()
		}
	}()

	parseResult := parseWorkspaceSuspendOperationalResult{}
	parseAPIKeyUpdateResult, parseErr := parseTX.Exec(
		`UPDATE api_keys
		 SET revoked_at = ?
		 WHERE workspace_id = ?
		   AND COALESCE(TRIM(revoked_at), '') = ''`,
		parseNow,
		parseWorkspaceID,
	)
	if parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	if parseResult.RevokedAPIKeys, parseErr = parseAPIKeyUpdateResult.RowsAffected(); parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}

	parseWebhookUpdateResult, parseErr := parseTX.Exec(
		`UPDATE webhook_endpoints
		 SET is_enabled = 0,
		     updated_at = ?
		 WHERE workspace_id = ?
		   AND is_enabled <> 0`,
		parseNow,
		parseWorkspaceID,
	)
	if parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	if parseResult.DisabledWebhookEndpoints, parseErr = parseWebhookUpdateResult.RowsAffected(); parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}

	parseJobUpdateResult, parseErr := parseTX.Exec(
		`UPDATE background_jobs
		 SET status = 'failed',
		     finished_at = ?,
		     error_message = ?,
		     updated_at = ?
		 WHERE queue_key = ?
		   AND status IN ('pending', 'running')`,
		parseNow,
		parseSuppressionError,
		parseNow,
		parseQueueKey,
	)
	if parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	if parseResult.SuppressedBackgroundJobs, parseErr = parseJobUpdateResult.RowsAffected(); parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}

	if parseErr = parseTX.Commit(); parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	return parseResult, nil
}

// parseRestoreAPIKeysByUser clears revocation timestamps for one user's API keys and returns the number of updated rows.
func (parseS *Store) parseRestoreAPIKeysByUser(parseUserID int64) (int64, error) {
	if parseUserID <= 0 {
		return 0, nil
	}
	parseResult, parseErr := parseS.db.Exec(
		`UPDATE api_keys
		 SET revoked_at = ''
		 WHERE user_id = ?
		   AND COALESCE(TRIM(revoked_at), '') <> ''`,
		parseUserID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr != nil {
		return 0, parseErr
	}
	return parseRowsAffected, nil
}

// parseRestoreWorkspaceOperationalEffects restores selected workspace-scoped capabilities and returns one count summary.
func (parseS *Store) parseRestoreWorkspaceOperationalEffects(parseWorkspaceID int64, isParseRestoreAPIKeys bool, isParseRestoreWebhooks bool, isParseRestoreJobs bool) (parseWorkspaceSuspendOperationalResult, error) {
	if parseWorkspaceID <= 0 {
		return parseWorkspaceSuspendOperationalResult{}, errors.New("restore workspace operational effects: workspace id is required")
	}
	parseQueueKey := fmt.Sprintf("workspace:%d", parseWorkspaceID)
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult := parseWorkspaceSuspendOperationalResult{}

	parseTX, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	defer func() {
		if parseErr != nil {
			_ = parseTX.Rollback()
		}
	}()

	if isParseRestoreAPIKeys {
		parseAPIKeyUpdateResult, parseErr2 := parseTX.Exec(
			`UPDATE api_keys
			 SET revoked_at = ''
			 WHERE workspace_id = ?
			   AND COALESCE(TRIM(revoked_at), '') <> ''`,
			parseWorkspaceID,
		)
		if parseErr2 != nil {
			parseErr = parseErr2
			return parseWorkspaceSuspendOperationalResult{}, parseErr
		}
		parseResult.RevokedAPIKeys, parseErr = parseAPIKeyUpdateResult.RowsAffected()
		if parseErr != nil {
			return parseWorkspaceSuspendOperationalResult{}, parseErr
		}
	}

	if isParseRestoreWebhooks {
		parseWebhookUpdateResult, parseErr2 := parseTX.Exec(
			`UPDATE webhook_endpoints
			 SET is_enabled = 1,
			     updated_at = ?
			 WHERE workspace_id = ?
			   AND is_enabled = 0`,
			parseNow,
			parseWorkspaceID,
		)
		if parseErr2 != nil {
			parseErr = parseErr2
			return parseWorkspaceSuspendOperationalResult{}, parseErr
		}
		parseResult.DisabledWebhookEndpoints, parseErr = parseWebhookUpdateResult.RowsAffected()
		if parseErr != nil {
			return parseWorkspaceSuspendOperationalResult{}, parseErr
		}
	}

	if isParseRestoreJobs {
		parseJobUpdateResult, parseErr2 := parseTX.Exec(
			`UPDATE background_jobs
			 SET status = 'pending',
			     finished_at = '',
			     error_message = '',
			     updated_at = ?
			 WHERE queue_key = ?
			   AND status = 'failed'
			   AND error_message LIKE ?`,
			parseNow,
			parseQueueKey,
			parseWorkspaceJobSuppressionErrorPrefix+"%",
		)
		if parseErr2 != nil {
			parseErr = parseErr2
			return parseWorkspaceSuspendOperationalResult{}, parseErr
		}
		parseResult.SuppressedBackgroundJobs, parseErr = parseJobUpdateResult.RowsAffected()
		if parseErr != nil {
			return parseWorkspaceSuspendOperationalResult{}, parseErr
		}
	}

	if parseErr = parseTX.Commit(); parseErr != nil {
		return parseWorkspaceSuspendOperationalResult{}, parseErr
	}
	return parseResult, nil
}

// parseUpsertWebhookEndpoint persists one webhook endpoint row.
func (parseS *Store) parseUpsertWebhookEndpoint(parseWrite parseWebhookEndpointWrite) error {
	if parseWrite.WorkspaceID <= 0 || strings.TrimSpace(parseWrite.TargetURL) == "" {
		return errors.New("upsert webhook endpoint: workspace id and target url are required")
	}
	if parseErr := parseS.parseRequireWorkspaceOperational(parseWrite.WorkspaceID); parseErr != nil {
		return parseErr
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

// parseUpsertWebhookDelivery persists one webhook delivery row.
func (parseS *Store) parseUpsertWebhookDelivery(parseWrite parseWebhookDeliveryWrite) error {
	if parseWrite.EndpointID <= 0 || strings.TrimSpace(parseWrite.DeliveryKey) == "" {
		return errors.New("upsert webhook delivery: endpoint id and delivery key are required")
	}
	parseWorkspaceID, isParseFound, parseErr := parseS.parseGetWebhookEndpointWorkspaceID(parseWrite.EndpointID)
	if parseErr != nil {
		return parseErr
	}
	if !isParseFound {
		return errStoreSuperuserScopeMissing
	}
	if parseErr = parseS.parseRequireWorkspaceOperational(parseWorkspaceID); parseErr != nil {
		return parseErr
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWebhookDelivery,
		parseWrite.EndpointID,
		strings.TrimSpace(parseWrite.EventType),
		strings.TrimSpace(parseWrite.DeliveryKey),
		parseNormalizeBillingJSON(parseWrite.RequestHeadersJSON),
		parseNormalizeBillingJSON(parseWrite.RequestBodyJSON),
		parseWrite.ResponseStatus,
		strings.TrimSpace(parseWrite.ResponseBody),
		parseWrite.AttemptCount,
		strings.TrimSpace(parseWrite.DeliveredAt),
		strings.TrimSpace(parseWrite.FailedAt),
		strings.TrimSpace(parseWrite.NextRetryAt),
		parseNow,
		parseNow,
		parseWrite.EndpointID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseListWebhookDeliveries lists webhook deliveries newest-first.
func (parseS *Store) parseListWebhookDeliveries(parseLimit int64) ([]parseWebhookDeliveryRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWebhookDeliveries, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseDeliveryRows := make([]parseWebhookDeliveryRow, 0)
	for parseRows.Next() {
		var parseRow parseWebhookDeliveryRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.EndpointID,
			&parseRow.EventType,
			&parseRow.DeliveryKey,
			&parseRow.RequestHeadersJSON,
			&parseRow.RequestBodyJSON,
			&parseRow.ResponseStatus,
			&parseRow.ResponseBody,
			&parseRow.AttemptCount,
			&parseRow.DeliveredAt,
			&parseRow.FailedAt,
			&parseRow.NextRetryAt,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseDeliveryRows = append(parseDeliveryRows, parseRow)
	}
	return parseDeliveryRows, parseRows.Err()
}

// parseListWebhookDeliveriesPendingRetry lists webhook deliveries due for retry oldest-first by retry time.
func (parseS *Store) parseListWebhookDeliveriesPendingRetry(parseNow string, parseLimit int64) ([]parseWebhookDeliveryRow, error) {
	if strings.TrimSpace(parseNow) == "" {
		return nil, errors.New("list webhook deliveries pending retry: now timestamp is required")
	}
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWebhookDeliveriesPendingRetry, strings.TrimSpace(parseNow), parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseDeliveryRows := make([]parseWebhookDeliveryRow, 0)
	for parseRows.Next() {
		var parseRow parseWebhookDeliveryRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.EndpointID,
			&parseRow.EventType,
			&parseRow.DeliveryKey,
			&parseRow.RequestHeadersJSON,
			&parseRow.RequestBodyJSON,
			&parseRow.ResponseStatus,
			&parseRow.ResponseBody,
			&parseRow.AttemptCount,
			&parseRow.DeliveredAt,
			&parseRow.FailedAt,
			&parseRow.NextRetryAt,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseDeliveryRows = append(parseDeliveryRows, parseRow)
	}
	return parseDeliveryRows, parseRows.Err()
}

// parseUpdateWebhookDeliveryAttempt records one failed webhook attempt and next retry schedule.
func (parseS *Store) parseUpdateWebhookDeliveryAttempt(parseDeliveryKey string, parseResponseStatus int64, parseResponseBody string, parseAttemptCount int64, parseFailedAt string, parseNextRetryAt string) error {
	parseDeliveryKey = strings.TrimSpace(parseDeliveryKey)
	if parseDeliveryKey == "" {
		return errors.New("update webhook delivery attempt: delivery key is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	_, parseErr := parseS.db.Exec(
		parseS.queries.updateWebhookDeliveryAttempt,
		parseResponseStatus,
		strings.TrimSpace(parseResponseBody),
		parseAttemptCount,
		strings.TrimSpace(parseFailedAt),
		strings.TrimSpace(parseNextRetryAt),
		parseNow,
		parseDeliveryKey,
	)
	return parseErr
}

// parseUpdateWebhookDeliveryDelivered records one successful webhook delivery and clears retry state.
func (parseS *Store) parseUpdateWebhookDeliveryDelivered(parseDeliveryKey string, parseResponseStatus int64, parseResponseBody string, parseAttemptCount int64, parseDeliveredAt string) error {
	parseDeliveryKey = strings.TrimSpace(parseDeliveryKey)
	if parseDeliveryKey == "" {
		return errors.New("update webhook delivery delivered: delivery key is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	_, parseErr := parseS.db.Exec(
		parseS.queries.updateWebhookDeliveryDelivered,
		parseResponseStatus,
		strings.TrimSpace(parseResponseBody),
		parseAttemptCount,
		strings.TrimSpace(parseDeliveredAt),
		parseNow,
		parseDeliveryKey,
	)
	return parseErr
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
	if parseErr := parseS.parseRequireUserOperational(parseWrite.UserID); parseErr != nil {
		return parseErr
	}
	if parseErr := parseS.parseRequireWorkspaceOperational(parseWrite.WorkspaceID); parseErr != nil {
		return parseErr
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

// parseCreateSupportTicketMessage appends one support-ticket message row.
func (parseS *Store) parseCreateSupportTicketMessage(parseWrite parseSupportTicketMessageWrite) (int64, error) {
	if parseWrite.TicketID <= 0 {
		return 0, errors.New("create support ticket message: ticket id is required")
	}
	if parseErr := parseS.parseRequireUserOperational(parseWrite.AuthorUserID); parseErr != nil {
		return 0, parseErr
	}
	parseWorkspaceID, parseTicketUserID, isParseFound, parseErr := parseS.parseGetSupportTicketScopeByID(parseWrite.TicketID)
	if parseErr != nil {
		return 0, parseErr
	}
	if !isParseFound {
		return 0, errStoreSuperuserScopeMissing
	}
	if parseErr = parseS.parseRequireWorkspaceOperational(parseWorkspaceID); parseErr != nil {
		return 0, parseErr
	}
	if parseErr = parseS.parseRequireUserOperational(parseTicketUserID); parseErr != nil {
		return 0, parseErr
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createSupportTicketMessage,
		parseWrite.TicketID,
		parseWrite.AuthorUserID,
		parseNormalizeSUValue(parseWrite.MessageType, "reply"),
		strings.TrimSpace(parseWrite.Body),
		parseBuildBillingFlagValue(parseWrite.IsInternal),
		parseNow,
		parseNow,
		parseWrite.TicketID,
		parseWrite.AuthorUserID,
		parseWrite.AuthorUserID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreSuperuserScopeMissing
	}
	return parseResult.LastInsertId()
}

// parseListSupportTicketMessages lists support-ticket messages newest-first.
func (parseS *Store) parseListSupportTicketMessages(parseLimit int64) ([]parseSupportTicketMessageRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSupportTicketMessages, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseMessageRows := make([]parseSupportTicketMessageRow, 0)
	for parseRows.Next() {
		var parseRow parseSupportTicketMessageRow
		var parseIsInternal int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.TicketID,
			&parseRow.AuthorUserID,
			&parseRow.MessageType,
			&parseRow.Body,
			&parseIsInternal,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsInternal = parseIsInternal != 0
		parseMessageRows = append(parseMessageRows, parseRow)
	}
	return parseMessageRows, parseRows.Err()
}

// parseCreateIncidentUpdate appends one incident update row.
func (parseS *Store) parseCreateIncidentUpdate(parseWrite parseIncidentUpdateWrite) (int64, error) {
	if parseWrite.IncidentID <= 0 {
		return 0, errors.New("create incident update: incident id is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createIncidentUpdate,
		parseWrite.IncidentID,
		parseNormalizeSUValue(parseWrite.Status, "investigating"),
		strings.TrimSpace(parseWrite.Message),
		parseBuildBillingFlagValue(parseWrite.IsPublic),
		strings.TrimSpace(parseWrite.PublishedAt),
		parseWrite.CreatedByUserID,
		parseNow,
		parseWrite.IncidentID,
		parseWrite.CreatedByUserID,
		parseWrite.CreatedByUserID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreSuperuserScopeMissing
	}
	return parseResult.LastInsertId()
}

// parseListIncidentUpdates lists incident updates newest-first.
func (parseS *Store) parseListIncidentUpdates(parseLimit int64) ([]parseIncidentUpdateRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listIncidentUpdates, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseUpdateRows := make([]parseIncidentUpdateRow, 0)
	for parseRows.Next() {
		var parseRow parseIncidentUpdateRow
		var parseIsPublic int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.IncidentID,
			&parseRow.Status,
			&parseRow.Message,
			&parseIsPublic,
			&parseRow.PublishedAt,
			&parseRow.CreatedByUserID,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsPublic = parseIsPublic != 0
		parseUpdateRows = append(parseUpdateRows, parseRow)
	}
	return parseUpdateRows, parseRows.Err()
}

// parseCreateNotificationOutbox appends one notification-outbox row.
func (parseS *Store) parseCreateNotificationOutbox(parseWrite parseNotificationOutboxWrite) (int64, error) {
	if strings.TrimSpace(parseWrite.NotificationKey) == "" {
		return 0, errors.New("create notification outbox: notification key is required")
	}
	if parseErr := parseS.parseRequireUserOperational(parseWrite.UserID); parseErr != nil {
		return 0, parseErr
	}
	if parseErr := parseS.parseRequireWorkspaceOperational(parseWrite.WorkspaceID); parseErr != nil {
		return 0, parseErr
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createNotificationOutbox,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		strings.TrimSpace(parseWrite.NotificationKey),
		parseNormalizeSUValue(parseWrite.ChannelKey, "email"),
		strings.TrimSpace(parseWrite.TemplateKey),
		parseNormalizeSUValue(parseWrite.Status, "pending"),
		strings.TrimSpace(parseWrite.Subject),
		strings.TrimSpace(parseWrite.BodyText),
		parseNormalizeBillingJSON(parseWrite.PayloadJSON),
		strings.TrimSpace(parseWrite.DedupeKey),
		strings.TrimSpace(parseWrite.ScheduledAt),
		strings.TrimSpace(parseWrite.SentAt),
		strings.TrimSpace(parseWrite.FailedAt),
		strings.TrimSpace(parseWrite.ErrorMessage),
		parseNow,
		parseNow,
		parseWrite.WorkspaceID,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
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

// parseListNotificationOutbox lists notification-outbox rows newest-first.
func (parseS *Store) parseListNotificationOutbox(parseLimit int64) ([]parseNotificationOutboxRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listNotificationOutbox, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseOutboxRows := make([]parseNotificationOutboxRow, 0)
	for parseRows.Next() {
		var parseRow parseNotificationOutboxRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.NotificationKey,
			&parseRow.ChannelKey,
			&parseRow.TemplateKey,
			&parseRow.Status,
			&parseRow.Subject,
			&parseRow.BodyText,
			&parseRow.PayloadJSON,
			&parseRow.DedupeKey,
			&parseRow.ScheduledAt,
			&parseRow.SentAt,
			&parseRow.FailedAt,
			&parseRow.ErrorMessage,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseOutboxRows = append(parseOutboxRows, parseRow)
	}
	return parseOutboxRows, parseRows.Err()
}

// parseListNotificationOutboxPending lists pending notification rows that are ready for dispatch.
func (parseS *Store) parseListNotificationOutboxPending(parseNow string, parseLimit int64) ([]parseNotificationOutboxRow, error) {
	parseNow = strings.TrimSpace(parseNow)
	if parseNow == "" {
		return nil, errors.New("list notification outbox pending: now timestamp is required")
	}
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listNotificationOutboxPending, parseNow, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseOutboxRows := make([]parseNotificationOutboxRow, 0)
	for parseRows.Next() {
		var parseRow parseNotificationOutboxRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.NotificationKey,
			&parseRow.ChannelKey,
			&parseRow.TemplateKey,
			&parseRow.Status,
			&parseRow.Subject,
			&parseRow.BodyText,
			&parseRow.PayloadJSON,
			&parseRow.DedupeKey,
			&parseRow.ScheduledAt,
			&parseRow.SentAt,
			&parseRow.FailedAt,
			&parseRow.ErrorMessage,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseOutboxRows = append(parseOutboxRows, parseRow)
	}
	return parseOutboxRows, parseRows.Err()
}

// parseUpdateNotificationOutboxStatus updates one notification row delivery status and delivery timestamps.
func (parseS *Store) parseUpdateNotificationOutboxStatus(parseNotificationID int64, parseStatus, parseSentAt, parseFailedAt, parseErrorMessage string) error {
	if parseNotificationID <= 0 {
		return errors.New("update notification outbox status: notification id is required")
	}
	_, parseErr := parseS.db.Exec(
		parseS.queries.updateNotificationOutboxStatus,
		parseNormalizeSUValue(parseStatus, "pending"),
		strings.TrimSpace(parseSentAt),
		strings.TrimSpace(parseFailedAt),
		strings.TrimSpace(parseErrorMessage),
		time.Now().UTC().Format(time.RFC3339),
		parseNotificationID,
	)
	return parseErr
}

// parseUpsertBackgroundJob persists one background-job row.
func (parseS *Store) parseUpsertBackgroundJob(parseWrite parseBackgroundJobWrite) error {
	if strings.TrimSpace(parseWrite.JobKey) == "" || strings.TrimSpace(parseWrite.JobType) == "" {
		return errors.New("upsert background job: job key and job type are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertBackgroundJob,
		parseNormalizeSUKey(parseWrite.JobKey),
		strings.TrimSpace(parseWrite.JobType),
		parseNormalizeSUValue(parseWrite.QueueKey, "default"),
		parseNormalizeSUValue(parseWrite.Status, "pending"),
		parseWrite.AttemptCount,
		parseWrite.MaxAttempts,
		parseNormalizeBillingJSON(parseWrite.PayloadJSON),
		strings.TrimSpace(parseWrite.RunAfter),
		strings.TrimSpace(parseWrite.StartedAt),
		strings.TrimSpace(parseWrite.FinishedAt),
		strings.TrimSpace(parseWrite.ErrorMessage),
		parseNow,
		parseNow,
	)
	return parseErr
}

// parseListBackgroundJobs lists background jobs newest-first.
func (parseS *Store) parseListBackgroundJobs(parseLimit int64) ([]parseBackgroundJobRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBackgroundJobs, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseJobRows := make([]parseBackgroundJobRow, 0)
	for parseRows.Next() {
		var parseRow parseBackgroundJobRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.JobKey,
			&parseRow.JobType,
			&parseRow.QueueKey,
			&parseRow.Status,
			&parseRow.AttemptCount,
			&parseRow.MaxAttempts,
			&parseRow.PayloadJSON,
			&parseRow.RunAfter,
			&parseRow.StartedAt,
			&parseRow.FinishedAt,
			&parseRow.ErrorMessage,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseJobRows = append(parseJobRows, parseRow)
	}
	return parseJobRows, parseRows.Err()
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

// parseListBillingPlanOverages lists pricing overage controls newest-first.
func (parseS *Store) parseListBillingPlanOverages(parseLimit int64) ([]parseBillingPlanOverageRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingPlanOverages, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseOverageRows := make([]parseBillingPlanOverageRow, 0)
	for parseRows.Next() {
		var parseRow parseBillingPlanOverageRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.PlanCode,
			&parseRow.MeterKey,
			&parseRow.IncludedUnits,
			&parseRow.SoftLimitUnits,
			&parseRow.HardLimitUnits,
			&parseRow.OverageUnitSize,
			&parseRow.OveragePriceCents,
			&parseRow.BillingInterval,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseOverageRows = append(parseOverageRows, parseRow)
	}
	return parseOverageRows, parseRows.Err()
}

// parseListBillingQuotaPolicies lists pricing quota policies newest-first.
func (parseS *Store) parseListBillingQuotaPolicies(parseLimit int64) ([]parseBillingQuotaPolicyRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingQuotaPolicies, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parsePolicyRows := make([]parseBillingQuotaPolicyRow, 0)
	for parseRows.Next() {
		var parseRow parseBillingQuotaPolicyRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.PlanCode,
			&parseRow.QuotaKey,
			&parseRow.SoftLimitValue,
			&parseRow.HardLimitValue,
			&parseRow.ResetInterval,
			&parseRow.EnforcementMode,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parsePolicyRows = append(parsePolicyRows, parseRow)
	}
	return parsePolicyRows, parseRows.Err()
}

// parseListBillingUpgradeTriggers lists pricing upgrade-trigger controls newest-first.
func (parseS *Store) parseListBillingUpgradeTriggers(parseLimit int64) ([]parseBillingUpgradeTriggerRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingUpgradeTriggers, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseTriggerRows := make([]parseBillingUpgradeTriggerRow, 0)
	for parseRows.Next() {
		var parseRow parseBillingUpgradeTriggerRow
		var parseIsEnabled int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.PlanCode,
			&parseRow.TriggerKey,
			&parseRow.ThresholdPercent,
			&parseRow.UpgradePlanCode,
			&parseRow.Message,
			&parseRow.CTALabel,
			&parseRow.CTAURL,
			&parseIsEnabled,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsEnabled = parseIsEnabled != 0
		parseTriggerRows = append(parseTriggerRows, parseRow)
	}
	return parseTriggerRows, parseRows.Err()
}

// parseListIncidents lists incident rows newest-first.
func (parseS *Store) parseListIncidents(parseLimit int64) ([]parseIncidentRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listIncidents, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseIncidentRows := make([]parseIncidentRow, 0)
	for parseRows.Next() {
		var parseRow parseIncidentRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.IncidentKey,
			&parseRow.SLOKey,
			&parseRow.Severity,
			&parseRow.Status,
			&parseRow.Title,
			&parseRow.Summary,
			&parseRow.StartedAt,
			&parseRow.ResolvedAt,
			&parseRow.PostmortemURL,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseIncidentRows = append(parseIncidentRows, parseRow)
	}
	return parseIncidentRows, parseRows.Err()
}

// parseListWorkspaceCostGuardrails lists workspace cost guardrail rows newest-first.
func (parseS *Store) parseListWorkspaceCostGuardrails(parseLimit int64) ([]parseWorkspaceCostGuardrailRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWorkspaceCostGuardrails, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseGuardrailRows := make([]parseWorkspaceCostGuardrailRow, 0)
	for parseRows.Next() {
		var parseRow parseWorkspaceCostGuardrailRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.GuardrailKey,
			&parseRow.DailyBudgetCents,
			&parseRow.MonthlyBudgetCents,
			&parseRow.MaxCostPerRequestCents,
			&parseRow.AlertThresholdPercent,
			&parseRow.ActionMode,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseGuardrailRows = append(parseGuardrailRows, parseRow)
	}
	return parseGuardrailRows, parseRows.Err()
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
