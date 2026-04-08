package app

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type parseSuperuserBillingPlanOverageWrite struct {
	PlanCode          string
	MeterKey          string
	IncludedUnits     int64
	SoftLimitUnits    int64
	HardLimitUnits    int64
	OverageUnitSize   int64
	OveragePriceCents int64
	BillingInterval   string
}

type parseSuperuserBillingQuotaPolicyWrite struct {
	PlanCode        string
	QuotaKey        string
	SoftLimitValue  int64
	HardLimitValue  int64
	ResetInterval   string
	EnforcementMode string
}

type parseSuperuserBillingUpgradeTriggerWrite struct {
	PlanCode         string
	TriggerKey       string
	ThresholdPercent int64
	UpgradePlanCode  string
	Message          string
	CTALabel         string
	CTAURL           string
	IsEnabled        bool
}

type parseSuperuserBillingDunningEventWrite struct {
	ID             int64
	CustomerID     int64
	SubscriptionID int64
	InvoiceID      int64
	Status         string
	AttemptCount   int64
	FailureReason  string
	NextAttemptAt  string
	ResolvedAt     string
}

type parseSuperuserBillingPlanWrite struct {
	PlanCode                string
	PlanName                string
	PlanRank                int64
	IsActive                bool
	MonthlyBaseCents        int64
	MonthlyPlatformFeeCents int64
	YearlyBaseCents         int64
	UsagePremiumBasisPoints int64
	IncludedTokensMonthly   int64
	IncludedSeats           int64
	MinSeats                int64
	WorkspaceMode           string
	MaxSeats                int64
	SupportsPriority        bool
	SupportsCollaboration   bool
	SupportsWorkspaceAdmin  bool
	SupportsTeamWorkspace   bool
	SupportsSSO             bool
}

type parseSuperuserBillingPlanEntitlementWrite struct {
	PlanCode         string
	EntitlementKey   string
	EntitlementValue string
}

// parseUpsertSuperuserBillingPlan stores one billing plan row keyed by plan code and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserBillingPlan(parseWrite parseSuperuserBillingPlanWrite) (parseBillingPlanRow, error) {
	if parseErr := parseValidateUsageBasedBillingPlanWrite(parseWrite); parseErr != nil {
		return parseBillingPlanRow{}, parseErr
	}
	if parseErr := parseValidatePlanBoundaryPlanWrite(parseWrite); parseErr != nil {
		return parseBillingPlanRow{}, parseErr
	}
	parsePlanCode := parseNormalizeSuperuserBillingPlanCode(parseWrite.PlanCode)
	if parsePlanCode == "" {
		return parseBillingPlanRow{}, errors.New("upsert superuser billing plan: plan code is required")
	}
	parsePlanName := strings.TrimSpace(parseWrite.PlanName)
	if parsePlanName == "" {
		parsePlanName = parsePlanCode
	}
	parsePlanRank := parseWrite.PlanRank
	if parsePlanRank < 0 {
		parsePlanRank = 0
	}
	parseMonthlyPlatformFeeCents := parseWrite.MonthlyPlatformFeeCents
	if parseMonthlyPlatformFeeCents <= 0 {
		parseMonthlyPlatformFeeCents = parseWrite.MonthlyBaseCents
	}
	parseMonthlyBaseCents := parseWrite.MonthlyBaseCents
	if parseMonthlyBaseCents <= 0 {
		parseMonthlyBaseCents = parseMonthlyPlatformFeeCents
	}
	parseSupportsCollaboration := parseWrite.SupportsCollaboration || parseWrite.SupportsTeamWorkspace
	parseSupportsWorkspaceAdmin := parseWrite.SupportsWorkspaceAdmin || parseSupportsCollaboration
	parseSupportsTeamWorkspace := parseWrite.SupportsTeamWorkspace || parseSupportsCollaboration
	parseIncludedSeats := parseWrite.IncludedSeats
	if parseIncludedSeats <= 0 {
		parseIncludedSeats = parseWrite.MinSeats
	}
	if parseIncludedSeats <= 0 {
		parseIncludedSeats = 1
	}
	parseMinSeats := parseWrite.MinSeats
	if parseMinSeats <= 0 {
		parseMinSeats = parseIncludedSeats
	}
	if parseMinSeats <= 0 {
		parseMinSeats = 1
	}
	if parseIncludedSeats < parseMinSeats {
		parseIncludedSeats = parseMinSeats
	}
	parseMaxSeats := parseWrite.MaxSeats
	if parseMaxSeats <= 0 {
		parseMaxSeats = parseMinSeats
	}
	if parseMaxSeats < parseMinSeats {
		parseMaxSeats = parseMinSeats
	}
	parseWorkspaceMode := strings.TrimSpace(strings.ToLower(parseWrite.WorkspaceMode))
	if parseWorkspaceMode == "" {
		switch parsePlanCode {
		case "pro":
			parseWorkspaceMode = "single"
		case "team":
			parseWorkspaceMode = "team"
		case "enterprise":
			parseWorkspaceMode = "enterprise"
		default:
			if parseSupportsCollaboration {
				parseWorkspaceMode = "team"
			} else {
				parseWorkspaceMode = "single"
			}
		}
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingPlan,
		parsePlanCode,
		parsePlanName,
		parsePlanRank,
		parseBuildBillingFlagValue(parseWrite.IsActive),
		parseMonthlyBaseCents,
		parseMonthlyPlatformFeeCents,
		parseWrite.YearlyBaseCents,
		parseWrite.UsagePremiumBasisPoints,
		parseWrite.IncludedTokensMonthly,
		parseIncludedSeats,
		parseMinSeats,
		parseWorkspaceMode,
		parseMaxSeats,
		parseBuildBillingFlagValue(parseWrite.SupportsPriority),
		parseBuildBillingFlagValue(parseSupportsCollaboration),
		parseBuildBillingFlagValue(parseSupportsWorkspaceAdmin),
		parseBuildBillingFlagValue(parseSupportsTeamWorkspace),
		parseBuildBillingFlagValue(parseWrite.SupportsSSO),
		parseNow,
		parseNow,
	); parseErr != nil {
		return parseBillingPlanRow{}, parseErr
	}
	parseRows, parseErr := parseS.parseListBillingPlans(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseBillingPlanRow{}, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode {
			return parseRow, nil
		}
	}
	return parseBillingPlanRow{}, errStoreSuperuserScopeMissing
}

// parseDeleteSuperuserBillingPlan deletes one billing plan row keyed by plan code.
func (parseS *Store) parseDeleteSuperuserBillingPlan(parsePlanCode string) error {
	parsePlanCode = parseNormalizeSuperuserBillingPlanCode(parsePlanCode)
	if parsePlanCode == "" {
		return errors.New("delete superuser billing plan: plan code is required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteBillingPlan, parsePlanCode)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseUpsertSuperuserBillingPlanEntitlement stores one billing-plan entitlement row and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserBillingPlanEntitlement(parseWrite parseSuperuserBillingPlanEntitlementWrite) (parseBillingPlanEntitlementRow, error) {
	if parseErr := parseValidatePlanBoundaryEntitlementWrite(parseWrite); parseErr != nil {
		return parseBillingPlanEntitlementRow{}, parseErr
	}
	parsePlanCode := parseNormalizeSuperuserBillingPlanCode(parseWrite.PlanCode)
	parseEntitlementKey := strings.TrimSpace(parseWrite.EntitlementKey)
	if parsePlanCode == "" || parseEntitlementKey == "" {
		return parseBillingPlanEntitlementRow{}, errors.New("upsert superuser billing plan entitlement: plan code and entitlement key are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingPlanEntitlement,
		parsePlanCode,
		parseEntitlementKey,
		strings.TrimSpace(parseWrite.EntitlementValue),
		parseNow,
	); parseErr != nil {
		return parseBillingPlanEntitlementRow{}, parseErr
	}
	parseRows, parseErr := parseS.parseListBillingPlanEntitlements(parsePlanCode)
	if parseErr != nil {
		return parseBillingPlanEntitlementRow{}, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.EntitlementKey == parseEntitlementKey {
			return parseRow, nil
		}
	}
	return parseBillingPlanEntitlementRow{}, errStoreSuperuserScopeMissing
}

// parseDeleteSuperuserBillingPlanEntitlement deletes one billing-plan entitlement row keyed by plan code and entitlement key.
func (parseS *Store) parseDeleteSuperuserBillingPlanEntitlement(parsePlanCode string, parseEntitlementKey string) error {
	parsePlanCode = parseNormalizeSuperuserBillingPlanCode(parsePlanCode)
	parseEntitlementKey = strings.TrimSpace(parseEntitlementKey)
	if parsePlanCode == "" || parseEntitlementKey == "" {
		return errors.New("delete superuser billing plan entitlement: plan code and entitlement key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteBillingPlanEntitlement, parsePlanCode, parseEntitlementKey)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseUpsertSuperuserBillingPlanOverage stores one plan overage control row keyed by plan+meter and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserBillingPlanOverage(parseWrite parseSuperuserBillingPlanOverageWrite) (parseBillingPlanOverageRow, error) {
	parsePlanCode := strings.TrimSpace(parseWrite.PlanCode)
	parseMeterKey := strings.TrimSpace(parseWrite.MeterKey)
	if parsePlanCode == "" || parseMeterKey == "" {
		return parseBillingPlanOverageRow{}, errors.New("upsert superuser billing plan overage: plan code and meter key are required")
	}
	parseOverageUnitSize := parseWrite.OverageUnitSize
	if parseOverageUnitSize <= 0 {
		parseOverageUnitSize = 1
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingPlanOverage,
		parsePlanCode,
		parseMeterKey,
		parseWrite.IncludedUnits,
		parseWrite.SoftLimitUnits,
		parseWrite.HardLimitUnits,
		parseOverageUnitSize,
		parseWrite.OveragePriceCents,
		parseNormalizeBillingInterval(parseWrite.BillingInterval),
		parseNow,
	); parseErr != nil {
		return parseBillingPlanOverageRow{}, parseErr
	}
	parseRows, parseErr := parseS.parseListBillingPlanOverages(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseBillingPlanOverageRow{}, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode && parseRow.MeterKey == parseMeterKey {
			return parseRow, nil
		}
	}
	return parseBillingPlanOverageRow{}, errStoreSuperuserScopeMissing
}

// parseDeleteSuperuserBillingPlanOverage deletes one plan overage control row keyed by plan+meter.
func (parseS *Store) parseDeleteSuperuserBillingPlanOverage(parsePlanCode string, parseMeterKey string) error {
	parsePlanCode = strings.TrimSpace(parsePlanCode)
	parseMeterKey = strings.TrimSpace(parseMeterKey)
	if parsePlanCode == "" || parseMeterKey == "" {
		return errors.New("delete superuser billing plan overage: plan code and meter key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteBillingPlanOverage, parsePlanCode, parseMeterKey)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseUpsertSuperuserBillingQuotaPolicy stores one quota policy control row keyed by plan+quota and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserBillingQuotaPolicy(parseWrite parseSuperuserBillingQuotaPolicyWrite) (parseBillingQuotaPolicyRow, error) {
	parsePlanCode := strings.TrimSpace(parseWrite.PlanCode)
	parseQuotaKey := strings.TrimSpace(parseWrite.QuotaKey)
	if parsePlanCode == "" || parseQuotaKey == "" {
		return parseBillingQuotaPolicyRow{}, errors.New("upsert superuser billing quota policy: plan code and quota key are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingQuotaPolicy,
		parsePlanCode,
		parseQuotaKey,
		parseWrite.SoftLimitValue,
		parseWrite.HardLimitValue,
		parseNormalizeBillingInterval(parseWrite.ResetInterval),
		parseNormalizeSuperuserBillingEnforcementMode(parseWrite.EnforcementMode),
		parseNow,
	); parseErr != nil {
		return parseBillingQuotaPolicyRow{}, parseErr
	}
	parseRows, parseErr := parseS.parseListBillingQuotaPolicies(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseBillingQuotaPolicyRow{}, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode && parseRow.QuotaKey == parseQuotaKey {
			return parseRow, nil
		}
	}
	return parseBillingQuotaPolicyRow{}, errStoreSuperuserScopeMissing
}

// parseDeleteSuperuserBillingQuotaPolicy deletes one quota policy control row keyed by plan+quota.
func (parseS *Store) parseDeleteSuperuserBillingQuotaPolicy(parsePlanCode string, parseQuotaKey string) error {
	parsePlanCode = strings.TrimSpace(parsePlanCode)
	parseQuotaKey = strings.TrimSpace(parseQuotaKey)
	if parsePlanCode == "" || parseQuotaKey == "" {
		return errors.New("delete superuser billing quota policy: plan code and quota key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteBillingQuotaPolicy, parsePlanCode, parseQuotaKey)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseUpsertSuperuserBillingUpgradeTrigger stores one upgrade trigger control row keyed by plan+trigger and returns the persisted row.
func (parseS *Store) parseUpsertSuperuserBillingUpgradeTrigger(parseWrite parseSuperuserBillingUpgradeTriggerWrite) (parseBillingUpgradeTriggerRow, error) {
	parsePlanCode := strings.TrimSpace(parseWrite.PlanCode)
	parseTriggerKey := strings.TrimSpace(parseWrite.TriggerKey)
	if parsePlanCode == "" || parseTriggerKey == "" {
		return parseBillingUpgradeTriggerRow{}, errors.New("upsert superuser billing upgrade trigger: plan code and trigger key are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingUpgradeTrigger,
		parsePlanCode,
		parseTriggerKey,
		parseClampSURolloutPercent(parseWrite.ThresholdPercent),
		strings.TrimSpace(parseWrite.UpgradePlanCode),
		strings.TrimSpace(parseWrite.Message),
		strings.TrimSpace(parseWrite.CTALabel),
		strings.TrimSpace(parseWrite.CTAURL),
		parseBuildBillingFlagValue(parseWrite.IsEnabled),
		parseNow,
	); parseErr != nil {
		return parseBillingUpgradeTriggerRow{}, parseErr
	}
	parseRows, parseErr := parseS.parseListBillingUpgradeTriggers(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseBillingUpgradeTriggerRow{}, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode && parseRow.TriggerKey == parseTriggerKey {
			return parseRow, nil
		}
	}
	return parseBillingUpgradeTriggerRow{}, errStoreSuperuserScopeMissing
}

// parseDeleteSuperuserBillingUpgradeTrigger deletes one upgrade trigger control row keyed by plan+trigger.
func (parseS *Store) parseDeleteSuperuserBillingUpgradeTrigger(parsePlanCode string, parseTriggerKey string) error {
	parsePlanCode = strings.TrimSpace(parsePlanCode)
	parseTriggerKey = strings.TrimSpace(parseTriggerKey)
	if parsePlanCode == "" || parseTriggerKey == "" {
		return errors.New("delete superuser billing upgrade trigger: plan code and trigger key are required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteBillingUpgradeTrigger, parsePlanCode, parseTriggerKey)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseUpsertSuperuserBillingDunningEvent stores one dunning-event row by id (update) or as a new row (create).
func (parseS *Store) parseUpsertSuperuserBillingDunningEvent(parseWrite parseSuperuserBillingDunningEventWrite) (parseBillingDunningEventRow, error) {
	if parseWrite.CustomerID <= 0 || parseWrite.SubscriptionID <= 0 || parseWrite.InvoiceID <= 0 {
		return parseBillingDunningEventRow{}, errors.New("upsert superuser billing dunning event: customer id, subscription id, and invoice id are required")
	}
	parseAttemptCount := parseWrite.AttemptCount
	if parseAttemptCount < 0 {
		parseAttemptCount = 0
	}
	parseStatus := parseNormalizeSuperuserDunningStatus(parseWrite.Status)
	parseResolvedAt := strings.TrimSpace(parseWrite.ResolvedAt)
	if parseResolvedAt == "" && parseStatus == "resolved" {
		parseResolvedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingDunningEvent,
		parseWrite.ID,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		parseWrite.InvoiceID,
		parseStatus,
		parseAttemptCount,
		strings.TrimSpace(parseWrite.FailureReason),
		strings.TrimSpace(parseWrite.NextAttemptAt),
		parseResolvedAt,
		parseNow,
		parseNow,
	)
	if parseErr != nil {
		return parseBillingDunningEventRow{}, parseErr
	}
	parseDunningEventID := parseWrite.ID
	if parseDunningEventID <= 0 {
		parseDunningEventID, parseErr = parseResult.LastInsertId()
		if parseErr != nil {
			return parseBillingDunningEventRow{}, parseErr
		}
	}
	parseRow, hasParseRow, parseErr := parseS.parseGetBillingDunningEventByID(parseDunningEventID)
	if parseErr != nil {
		return parseBillingDunningEventRow{}, parseErr
	}
	if !hasParseRow {
		return parseBillingDunningEventRow{}, errStoreSuperuserScopeMissing
	}
	return parseRow, nil
}

// parseGetBillingDunningEventByID resolves one dunning-event row by id.
func (parseS *Store) parseGetBillingDunningEventByID(parseDunningEventID int64) (parseBillingDunningEventRow, bool, error) {
	if parseDunningEventID <= 0 {
		return parseBillingDunningEventRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.getBillingDunningEventByID, parseDunningEventID)
	var parseDunningRow parseBillingDunningEventRow
	if parseErr := parseRow.Scan(
		&parseDunningRow.ID,
		&parseDunningRow.CustomerID,
		&parseDunningRow.SubscriptionID,
		&parseDunningRow.InvoiceID,
		&parseDunningRow.Status,
		&parseDunningRow.AttemptCount,
		&parseDunningRow.FailureReason,
		&parseDunningRow.NextAttemptAt,
		&parseDunningRow.ResolvedAt,
		&parseDunningRow.CreatedAt,
		&parseDunningRow.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseBillingDunningEventRow{}, false, nil
		}
		return parseBillingDunningEventRow{}, false, parseErr
	}
	return parseDunningRow, true, nil
}

// parseDeleteSuperuserBillingDunningEvent deletes one dunning-event row by id.
func (parseS *Store) parseDeleteSuperuserBillingDunningEvent(parseDunningEventID int64) error {
	if parseDunningEventID <= 0 {
		return errors.New("delete superuser billing dunning event: dunning event id is required")
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.deleteBillingDunningEvent, parseDunningEventID)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseNormalizeSuperuserBillingEnforcementMode normalizes one quota enforcement mode for superuser pricing controls.
func parseNormalizeSuperuserBillingEnforcementMode(parseMode string) string {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "warn", "track", "block":
		return strings.TrimSpace(strings.ToLower(parseMode))
	default:
		return "block"
	}
}

// parseNormalizeSuperuserDunningStatus normalizes one dunning status for superuser dunning-event controls.
func parseNormalizeSuperuserDunningStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "pending", "retrying", "resolved", "failed":
		return strings.TrimSpace(strings.ToLower(parseStatus))
	default:
		return "pending"
	}
}

// parseNormalizeSuperuserBillingPlanCode normalizes one superuser billing plan code.
func parseNormalizeSuperuserBillingPlanCode(parsePlanCode string) string {
	return strings.TrimSpace(strings.ToLower(parsePlanCode))
}
