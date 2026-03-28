package app

import (
	"errors"
	"strings"
	"time"
)

type parseOnboardingTemplateWrite struct {
	TemplateKey   string
	Title         string
	Category      string
	PromptText    string
	ChecklistJSON string
	IsDefault     bool
	SortOrder     int64
}

type parseOnboardingTemplateRow struct {
	ID            int64
	TemplateKey   string
	Title         string
	Category      string
	PromptText    string
	ChecklistJSON string
	IsDefault     bool
	SortOrder     int64
	UpdatedAt     string
}

type parseUserActivationMilestoneWrite struct {
	UserID       int64
	MilestoneKey string
	Status       string
	AchievedAt   string
	MetadataJSON string
}

type parseUserActivationMilestoneRow struct {
	ID           int64
	UserID       int64
	MilestoneKey string
	Status       string
	AchievedAt   string
	MetadataJSON string
	UpdatedAt    string
}

type parseSavedWorkflowWrite struct {
	WorkspaceID  int64
	UserID       int64
	WorkflowKey  string
	Name         string
	Description  string
	WorkflowJSON string
	IsPublic     bool
}

type parseSavedWorkflowRow struct {
	ID           int64
	WorkspaceID  int64
	UserID       int64
	WorkflowKey  string
	Name         string
	Description  string
	WorkflowJSON string
	IsPublic     bool
	CreatedAt    string
	UpdatedAt    string
}

type parsePromptLibraryItemWrite struct {
	WorkspaceID int64
	UserID      int64
	ItemKey     string
	Title       string
	Category    string
	PromptText  string
	TagsJSON    string
	IsPublic    bool
	UseCount    int64
}

type parsePromptLibraryItemRow struct {
	ID          int64
	WorkspaceID int64
	UserID      int64
	ItemKey     string
	Title       string
	Category    string
	PromptText  string
	TagsJSON    string
	IsPublic    bool
	UseCount    int64
	CreatedAt   string
	UpdatedAt   string
}

type parseWeeklyValueSummaryWrite struct {
	WorkspaceID int64
	UserID      int64
	SummaryWeek string
	SummaryText string
	MetricsJSON string
	SentAt      string
}

type parseWeeklyValueSummaryRow struct {
	ID          int64
	WorkspaceID int64
	UserID      int64
	SummaryWeek string
	SummaryText string
	MetricsJSON string
	SentAt      string
	CreatedAt   string
}

type parseProductAnalyticsEventWrite struct {
	WorkspaceID    int64
	UserID         int64
	SessionKey     string
	EventName      string
	FunnelKey      string
	StepKey        string
	ExperimentKey  string
	VariantKey     string
	EventPropsJSON string
}

type parseProductAnalyticsEventRow struct {
	ID             int64
	WorkspaceID    int64
	UserID         int64
	SessionKey     string
	EventName      string
	FunnelKey      string
	StepKey        string
	ExperimentKey  string
	VariantKey     string
	EventPropsJSON string
	CreatedAt      string
}

type parseExperimentAssignmentWrite struct {
	ExperimentKey string
	WorkspaceID   int64
	UserID        int64
	VariantKey    string
	AssignedAt    string
}

type parseExperimentAssignmentRow struct {
	ID            int64
	ExperimentKey string
	WorkspaceID   int64
	UserID        int64
	VariantKey    string
	AssignedAt    string
}

type parseSubscriptionChurnFeedbackWrite struct {
	CustomerID       int64
	SubscriptionID   int64
	WorkspaceID      int64
	ReasonKey        string
	Detail           string
	RecoveryOfferKey string
}

type parseSubscriptionChurnFeedbackRow struct {
	ID               int64
	CustomerID       int64
	SubscriptionID   int64
	WorkspaceID      int64
	ReasonKey        string
	Detail           string
	RecoveryOfferKey string
	CreatedAt        string
}

// parseUpsertOnboardingTemplate persists one onboarding-template row.
func (parseS *Store) parseUpsertOnboardingTemplate(parseWrite parseOnboardingTemplateWrite) error {
	parseTemplateKey := parseNormalizeSUKey(parseWrite.TemplateKey)
	if parseTemplateKey == "" {
		return errors.New("upsert onboarding template: template key is required")
	}
	parseTitle := strings.TrimSpace(parseWrite.Title)
	if parseTitle == "" {
		parseTitle = parseTemplateKey
	}
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertOnboardingTemplate,
		parseTemplateKey,
		parseTitle,
		strings.TrimSpace(parseWrite.Category),
		strings.TrimSpace(parseWrite.PromptText),
		parseNormalizeSUJSONArray(parseWrite.ChecklistJSON),
		parseBuildBillingFlagValue(parseWrite.IsDefault),
		parseWrite.SortOrder,
		time.Now().UTC().Format(time.RFC3339),
	)
	return parseErr
}

// parseListOnboardingTemplates lists onboarding templates in display order.
func (parseS *Store) parseListOnboardingTemplates(parseLimit int64) ([]parseOnboardingTemplateRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listOnboardingTemplates, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseTemplateRows := make([]parseOnboardingTemplateRow, 0)
	for parseRows.Next() {
		var parseRow parseOnboardingTemplateRow
		var parseIsDefault int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.TemplateKey,
			&parseRow.Title,
			&parseRow.Category,
			&parseRow.PromptText,
			&parseRow.ChecklistJSON,
			&parseIsDefault,
			&parseRow.SortOrder,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsDefault = parseIsDefault != 0
		parseTemplateRows = append(parseTemplateRows, parseRow)
	}
	return parseTemplateRows, parseRows.Err()
}

// parseUpsertUserActivationMilestone persists one user-activation milestone row.
func (parseS *Store) parseUpsertUserActivationMilestone(parseWrite parseUserActivationMilestoneWrite) error {
	if parseWrite.UserID <= 0 {
		return errors.New("upsert user activation milestone: user id is required")
	}
	parseMilestoneKey := parseNormalizeSUKey(parseWrite.MilestoneKey)
	if parseMilestoneKey == "" {
		return errors.New("upsert user activation milestone: milestone key is required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertUserActivationMilestone,
		parseWrite.UserID,
		parseMilestoneKey,
		parseNormalizeSUValue(parseWrite.Status, "pending"),
		strings.TrimSpace(parseWrite.AchievedAt),
		parseNormalizeBillingJSON(parseWrite.MetadataJSON),
		time.Now().UTC().Format(time.RFC3339),
		parseWrite.UserID,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return errStoreUserMissing
	}
	return nil
}

// parseListUserActivationMilestones lists user-activation milestones newest-first.
func (parseS *Store) parseListUserActivationMilestones(parseLimit int64) ([]parseUserActivationMilestoneRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listUserActivationMilestones, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseMilestoneRows := make([]parseUserActivationMilestoneRow, 0)
	for parseRows.Next() {
		var parseRow parseUserActivationMilestoneRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.UserID,
			&parseRow.MilestoneKey,
			&parseRow.Status,
			&parseRow.AchievedAt,
			&parseRow.MetadataJSON,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseMilestoneRows = append(parseMilestoneRows, parseRow)
	}
	return parseMilestoneRows, parseRows.Err()
}

// parseUpsertSavedWorkflow persists one saved-workflow row.
func (parseS *Store) parseUpsertSavedWorkflow(parseWrite parseSavedWorkflowWrite) error {
	parseWorkflowKey := parseNormalizeSUKey(parseWrite.WorkflowKey)
	if parseWorkflowKey == "" {
		return errors.New("upsert saved workflow: workflow key is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertSavedWorkflow,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		parseWorkflowKey,
		strings.TrimSpace(parseWrite.Name),
		strings.TrimSpace(parseWrite.Description),
		parseNormalizeBillingJSON(parseWrite.WorkflowJSON),
		parseBuildBillingFlagValue(parseWrite.IsPublic),
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

// parseListSavedWorkflows lists saved workflows newest-first.
func (parseS *Store) parseListSavedWorkflows(parseLimit int64) ([]parseSavedWorkflowRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSavedWorkflows, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseWorkflowRows := make([]parseSavedWorkflowRow, 0)
	for parseRows.Next() {
		var parseRow parseSavedWorkflowRow
		var parseIsPublic int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.WorkflowKey,
			&parseRow.Name,
			&parseRow.Description,
			&parseRow.WorkflowJSON,
			&parseIsPublic,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsPublic = parseIsPublic != 0
		parseWorkflowRows = append(parseWorkflowRows, parseRow)
	}
	return parseWorkflowRows, parseRows.Err()
}

// parseUpsertPromptLibraryItem persists one prompt-library item row.
func (parseS *Store) parseUpsertPromptLibraryItem(parseWrite parsePromptLibraryItemWrite) error {
	parseItemKey := parseNormalizeSUKey(parseWrite.ItemKey)
	if parseItemKey == "" {
		return errors.New("upsert prompt library item: item key is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertPromptLibraryItem,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		parseItemKey,
		strings.TrimSpace(parseWrite.Title),
		strings.TrimSpace(parseWrite.Category),
		strings.TrimSpace(parseWrite.PromptText),
		parseNormalizeSUJSONArray(parseWrite.TagsJSON),
		parseBuildBillingFlagValue(parseWrite.IsPublic),
		parseWrite.UseCount,
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

// parseListPromptLibraryItems lists prompt-library rows newest-first.
func (parseS *Store) parseListPromptLibraryItems(parseLimit int64) ([]parsePromptLibraryItemRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listPromptLibraryItems, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseItemRows := make([]parsePromptLibraryItemRow, 0)
	for parseRows.Next() {
		var parseRow parsePromptLibraryItemRow
		var parseIsPublic int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.ItemKey,
			&parseRow.Title,
			&parseRow.Category,
			&parseRow.PromptText,
			&parseRow.TagsJSON,
			&parseIsPublic,
			&parseRow.UseCount,
			&parseRow.CreatedAt,
			&parseRow.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsPublic = parseIsPublic != 0
		parseItemRows = append(parseItemRows, parseRow)
	}
	return parseItemRows, parseRows.Err()
}

// parseUpsertWeeklyValueSummary persists one weekly-value summary row.
func (parseS *Store) parseUpsertWeeklyValueSummary(parseWrite parseWeeklyValueSummaryWrite) error {
	if strings.TrimSpace(parseWrite.SummaryWeek) == "" {
		return errors.New("upsert weekly value summary: summary week is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertWeeklyValueSummary,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		strings.TrimSpace(parseWrite.SummaryWeek),
		strings.TrimSpace(parseWrite.SummaryText),
		parseNormalizeBillingJSON(parseWrite.MetricsJSON),
		strings.TrimSpace(parseWrite.SentAt),
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

// parseListWeeklyValueSummaries lists weekly-value summaries newest-first.
func (parseS *Store) parseListWeeklyValueSummaries(parseLimit int64) ([]parseWeeklyValueSummaryRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listWeeklyValueSummaries, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseSummaryRows := make([]parseWeeklyValueSummaryRow, 0)
	for parseRows.Next() {
		var parseRow parseWeeklyValueSummaryRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.SummaryWeek,
			&parseRow.SummaryText,
			&parseRow.MetricsJSON,
			&parseRow.SentAt,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseSummaryRows = append(parseSummaryRows, parseRow)
	}
	return parseSummaryRows, parseRows.Err()
}

// parseCreateProductAnalyticsEvent appends one product-analytics event row.
func (parseS *Store) parseCreateProductAnalyticsEvent(parseWrite parseProductAnalyticsEventWrite) (int64, error) {
	if strings.TrimSpace(parseWrite.EventName) == "" {
		return 0, errors.New("create product analytics event: event name is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseExperimentKey := strings.TrimSpace(parseWrite.ExperimentKey)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createProductAnalyticsEvent,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		strings.TrimSpace(parseWrite.SessionKey),
		strings.TrimSpace(parseWrite.EventName),
		strings.TrimSpace(parseWrite.FunnelKey),
		strings.TrimSpace(parseWrite.StepKey),
		parseExperimentKey,
		strings.TrimSpace(parseWrite.VariantKey),
		parseNormalizeBillingJSON(parseWrite.EventPropsJSON),
		parseNow,
		parseWrite.WorkspaceID,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		parseWrite.UserID,
		parseExperimentKey,
		parseExperimentKey,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseRowsAffected, parseErr2 := parseResult.RowsAffected(); parseErr2 == nil && parseRowsAffected == 0 {
		return 0, errStoreSuperuserScopeMissing
	}
	return parseResult.LastInsertId()
}

// parseListProductAnalyticsEvents lists product-analytics events newest-first.
func (parseS *Store) parseListProductAnalyticsEvents(parseLimit int64) ([]parseProductAnalyticsEventRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listProductAnalyticsEvents, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseEventRows := make([]parseProductAnalyticsEventRow, 0)
	for parseRows.Next() {
		var parseRow parseProductAnalyticsEventRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.SessionKey,
			&parseRow.EventName,
			&parseRow.FunnelKey,
			&parseRow.StepKey,
			&parseRow.ExperimentKey,
			&parseRow.VariantKey,
			&parseRow.EventPropsJSON,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseEventRows = append(parseEventRows, parseRow)
	}
	return parseEventRows, parseRows.Err()
}

// parseUpsertExperimentAssignment persists one experiment-assignment row.
func (parseS *Store) parseUpsertExperimentAssignment(parseWrite parseExperimentAssignmentWrite) error {
	parseExperimentKey := parseNormalizeSUKey(parseWrite.ExperimentKey)
	if parseExperimentKey == "" {
		return errors.New("upsert experiment assignment: experiment key is required")
	}
	parseAssignedAt := strings.TrimSpace(parseWrite.AssignedAt)
	if parseAssignedAt == "" {
		parseAssignedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertExperimentAssignment,
		parseExperimentKey,
		parseWrite.WorkspaceID,
		parseWrite.UserID,
		strings.TrimSpace(parseWrite.VariantKey),
		parseAssignedAt,
		parseExperimentKey,
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

// parseListExperimentAssignments lists experiment assignments newest-first.
func (parseS *Store) parseListExperimentAssignments(parseLimit int64) ([]parseExperimentAssignmentRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listExperimentAssignments, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseAssignmentRows := make([]parseExperimentAssignmentRow, 0)
	for parseRows.Next() {
		var parseRow parseExperimentAssignmentRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.ExperimentKey,
			&parseRow.WorkspaceID,
			&parseRow.UserID,
			&parseRow.VariantKey,
			&parseRow.AssignedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseAssignmentRows = append(parseAssignmentRows, parseRow)
	}
	return parseAssignmentRows, parseRows.Err()
}

// parseCreateSubscriptionChurnFeedback appends one subscription-churn feedback row.
func (parseS *Store) parseCreateSubscriptionChurnFeedback(parseWrite parseSubscriptionChurnFeedbackWrite) (int64, error) {
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createSubscriptionChurnFeedback,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		parseWrite.WorkspaceID,
		parseNormalizeSUValue(parseWrite.ReasonKey, "unknown"),
		strings.TrimSpace(parseWrite.Detail),
		strings.TrimSpace(parseWrite.RecoveryOfferKey),
		time.Now().UTC().Format(time.RFC3339),
		parseWrite.CustomerID,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		parseWrite.SubscriptionID,
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

// parseListSubscriptionChurnFeedback lists subscription-churn feedback rows newest-first.
func (parseS *Store) parseListSubscriptionChurnFeedback(parseLimit int64) ([]parseSubscriptionChurnFeedbackRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listSubscriptionChurnFeedback, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseFeedbackRows := make([]parseSubscriptionChurnFeedbackRow, 0)
	for parseRows.Next() {
		var parseRow parseSubscriptionChurnFeedbackRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.CustomerID,
			&parseRow.SubscriptionID,
			&parseRow.WorkspaceID,
			&parseRow.ReasonKey,
			&parseRow.Detail,
			&parseRow.RecoveryOfferKey,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseFeedbackRows = append(parseFeedbackRows, parseRow)
	}
	return parseFeedbackRows, parseRows.Err()
}
