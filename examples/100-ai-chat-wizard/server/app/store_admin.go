package app

import (
	"strconv"
	"strings"
	"time"
)

type parseAdminDashboardSummaryRow struct {
	TotalUsers                  int64
	TotalConversations          int64
	TotalMessages               int64
	WindowNewUsers              int64
	WindowNewConversations      int64
	WindowNewMessages           int64
	WindowUsageEvents           int64
	WindowTotalCostUSD          float64
	WindowPromptTokens          int64
	WindowCompletionTokens      int64
	WindowActiveUsers           int64
	WindowActiveConversations   int64
	WindowActiveClients         int64
	WindowCompletedEvents       int64
	WindowFailedEvents          int64
	WindowBillingEvents         int64
	WindowOpenInvoices          int64
	WindowOpenDunningEvents     int64
	OpenIncidents               int64
	OpenSupportTickets          int64
	ActiveExperiments           int64
	UnhealthyExperiments        int64
	WindowExperimentAssignments int64
}

type parseAdminDashboardDailyUsageRow struct {
	UsageDay            string
	UsageEventCount     int64
	TotalCostUSD        float64
	PromptTokens        int64
	CompletionTokens    int64
	ActiveUsers         int64
	ActiveConversations int64
	ActiveClients       int64
}

type parseAdminDashboardProviderUsageRow struct {
	ProviderID          string
	UsageEventCount     int64
	TotalCostUSD        float64
	PromptTokens        int64
	CompletionTokens    int64
	ActiveUsers         int64
	CompletedEventCount int64
	FailedEventCount    int64
}

type parseAdminDashboardModelUsageRow struct {
	ProviderID          string
	ModelID             string
	UsageEventCount     int64
	TotalCostUSD        float64
	PromptTokens        int64
	CompletionTokens    int64
	ActiveUsers         int64
	CompletedEventCount int64
	FailedEventCount    int64
}

type parseAdminDashboardUserUsageRow struct {
	UserID            int64
	Email             string
	DisplayName       string
	ConversationCount int64
	MessageCount      int64
	UsageEventCount   int64
	TotalCostUSD      float64
	PromptTokens      int64
	CompletionTokens  int64
	LastSeenAt        string
}

type parseAdminUsageEventRow struct {
	EventID                 string
	UserID                  int64
	Email                   string
	DisplayName             string
	ConversationID          int64
	ConversationPublicID    string
	ConversationTitle       string
	ProviderID              string
	ModelID                 string
	PromptTokens            int64
	CompletionTokens        int64
	UsageSource             string
	ProviderRequestID       string
	InputCostPerMillionUSD  float64
	OutputCostPerMillionUSD float64
	PricingCurrency         string
	InputCostUSD            float64
	OutputCostUSD           float64
	TotalCostUSD            float64
	ClientID                string
	TraceID                 string
	SpanID                  string
	TraceState              string
	Status                  string
	ErrorMessage            string
	CreatedAt               string
}

type parseAdminUserRow struct {
	UserID             int64
	Email              string
	DisplayName        string
	CreatedAt          string
	ConversationCount  int64
	MessageCount       int64
	UsageEventCount    int64
	TotalCostUSD       float64
	LastSeenAt         string
	WorkspaceCount     int64
	MemoryCount        int64
	OpenSupportCount   int64
	ActiveSubCount     int64
	TokenVersion       int64
	SupportMsgCount    int64
	ActiveSessionCount int64
}

type parseAdminConversationRow struct {
	ConversationID  int64
	PublicID        string
	UserID          int64
	Email           string
	DisplayName     string
	StartedAt       string
	Preview         string
	MessageCount    int64
	UsageEventCount int64
	TotalCostUSD    float64
	LastActivityAt  string
}

const parseAdminControlScanLimit int64 = 5000

// parseGetAdminDashboardSummary returns global and windowed counts for the admin dashboard.
func (parseS *Store) parseGetAdminDashboardSummary(parseSince string) (parseAdminDashboardSummaryRow, error) {
	parseRow := parseS.db.QueryRow(
		parseS.queries.getAdminDashboardSummary,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
		parseSince,
	)
	var parseSummary parseAdminDashboardSummaryRow
	if parseErr := parseRow.Scan(
		&parseSummary.TotalUsers,
		&parseSummary.TotalConversations,
		&parseSummary.TotalMessages,
		&parseSummary.WindowNewUsers,
		&parseSummary.WindowNewConversations,
		&parseSummary.WindowNewMessages,
		&parseSummary.WindowUsageEvents,
		&parseSummary.WindowTotalCostUSD,
		&parseSummary.WindowPromptTokens,
		&parseSummary.WindowCompletionTokens,
		&parseSummary.WindowActiveUsers,
		&parseSummary.WindowActiveConversations,
		&parseSummary.WindowActiveClients,
		&parseSummary.WindowCompletedEvents,
		&parseSummary.WindowFailedEvents,
		&parseSummary.WindowBillingEvents,
		&parseSummary.WindowOpenInvoices,
		&parseSummary.WindowOpenDunningEvents,
		&parseSummary.OpenIncidents,
		&parseSummary.OpenSupportTickets,
		&parseSummary.ActiveExperiments,
		&parseSummary.UnhealthyExperiments,
		&parseSummary.WindowExperimentAssignments,
	); parseErr != nil {
		return parseAdminDashboardSummaryRow{}, parseErr
	}
	return parseSummary, nil
}

// parseListAdminDashboardDailyUsage returns a daily usage series for the dashboard lookback window.
func (parseS *Store) parseListAdminDashboardDailyUsage(parseSince string) ([]parseAdminDashboardDailyUsageRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminDashboardDailyUsage, parseSince)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseDailyRows := make([]parseAdminDashboardDailyUsageRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminDashboardDailyUsageRow
		if parseErr2 := parseRows.Scan(
			&parseRow.UsageDay,
			&parseRow.UsageEventCount,
			&parseRow.TotalCostUSD,
			&parseRow.PromptTokens,
			&parseRow.CompletionTokens,
			&parseRow.ActiveUsers,
			&parseRow.ActiveConversations,
			&parseRow.ActiveClients,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseDailyRows = append(parseDailyRows, parseRow)
	}
	return parseDailyRows, parseRows.Err()
}

// parseListAdminDashboardProviderUsage returns provider leaderboard rows for the dashboard lookback window.
func (parseS *Store) parseListAdminDashboardProviderUsage(parseSince string, parseLimit int64) ([]parseAdminDashboardProviderUsageRow, error) {
	if parseLimit <= 0 {
		parseLimit = 10
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminDashboardProviderUsage, parseSince, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseProviderRows := make([]parseAdminDashboardProviderUsageRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminDashboardProviderUsageRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ProviderID,
			&parseRow.UsageEventCount,
			&parseRow.TotalCostUSD,
			&parseRow.PromptTokens,
			&parseRow.CompletionTokens,
			&parseRow.ActiveUsers,
			&parseRow.CompletedEventCount,
			&parseRow.FailedEventCount,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseProviderRows = append(parseProviderRows, parseRow)
	}
	return parseProviderRows, parseRows.Err()
}

// parseListAdminDashboardModelUsage returns model leaderboard rows for the dashboard lookback window.
func (parseS *Store) parseListAdminDashboardModelUsage(parseSince string, parseLimit int64) ([]parseAdminDashboardModelUsageRow, error) {
	if parseLimit <= 0 {
		parseLimit = 10
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminDashboardModelUsage, parseSince, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseModelRows := make([]parseAdminDashboardModelUsageRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminDashboardModelUsageRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ProviderID,
			&parseRow.ModelID,
			&parseRow.UsageEventCount,
			&parseRow.TotalCostUSD,
			&parseRow.PromptTokens,
			&parseRow.CompletionTokens,
			&parseRow.ActiveUsers,
			&parseRow.CompletedEventCount,
			&parseRow.FailedEventCount,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseModelRows = append(parseModelRows, parseRow)
	}
	return parseModelRows, parseRows.Err()
}

// parseListAdminDashboardUserUsage returns top-spending user rows for the dashboard lookback window.
func (parseS *Store) parseListAdminDashboardUserUsage(parseSince string, parseLimit int64) ([]parseAdminDashboardUserUsageRow, error) {
	if parseLimit <= 0 {
		parseLimit = 10
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminDashboardUserUsage, parseSince, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseUserRows := make([]parseAdminDashboardUserUsageRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminDashboardUserUsageRow
		if parseErr2 := parseRows.Scan(
			&parseRow.UserID,
			&parseRow.Email,
			&parseRow.DisplayName,
			&parseRow.ConversationCount,
			&parseRow.MessageCount,
			&parseRow.UsageEventCount,
			&parseRow.TotalCostUSD,
			&parseRow.PromptTokens,
			&parseRow.CompletionTokens,
			&parseRow.LastSeenAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseUserRows = append(parseUserRows, parseRow)
	}
	return parseUserRows, parseRows.Err()
}

// parseListAdminUsageEvents returns recent global usage ledger rows for the dashboard.
func (parseS *Store) parseListAdminUsageEvents(parseSince string, parseLimit int64) ([]parseAdminUsageEventRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminUsageEvents, parseSince, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseUsageRows := make([]parseAdminUsageEventRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminUsageEventRow
		if parseErr2 := parseRows.Scan(
			&parseRow.EventID,
			&parseRow.UserID,
			&parseRow.Email,
			&parseRow.DisplayName,
			&parseRow.ConversationID,
			&parseRow.ConversationPublicID,
			&parseRow.ConversationTitle,
			&parseRow.ProviderID,
			&parseRow.ModelID,
			&parseRow.PromptTokens,
			&parseRow.CompletionTokens,
			&parseRow.UsageSource,
			&parseRow.ProviderRequestID,
			&parseRow.InputCostPerMillionUSD,
			&parseRow.OutputCostPerMillionUSD,
			&parseRow.PricingCurrency,
			&parseRow.InputCostUSD,
			&parseRow.OutputCostUSD,
			&parseRow.TotalCostUSD,
			&parseRow.ClientID,
			&parseRow.TraceID,
			&parseRow.SpanID,
			&parseRow.TraceState,
			&parseRow.Status,
			&parseRow.ErrorMessage,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseUsageRows = append(parseUsageRows, parseRow)
	}
	return parseUsageRows, parseRows.Err()
}

// parseListAdminUsers returns recent users with aggregate activity and spend.
func (parseS *Store) parseListAdminUsers(parseLimit int64) ([]parseAdminUserRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminUsers, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseUserRows := make([]parseAdminUserRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminUserRow
		if parseErr2 := parseRows.Scan(
			&parseRow.UserID,
			&parseRow.Email,
			&parseRow.DisplayName,
			&parseRow.CreatedAt,
			&parseRow.ConversationCount,
			&parseRow.MessageCount,
			&parseRow.UsageEventCount,
			&parseRow.TotalCostUSD,
			&parseRow.LastSeenAt,
			&parseRow.WorkspaceCount,
			&parseRow.MemoryCount,
			&parseRow.OpenSupportCount,
			&parseRow.ActiveSubCount,
			&parseRow.TokenVersion,
			&parseRow.SupportMsgCount,
			&parseRow.ActiveSessionCount,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseUserRows = append(parseUserRows, parseRow)
	}
	return parseUserRows, parseRows.Err()
}

// parseListAdminConversations returns recent conversations with owner and cost rollups.
func (parseS *Store) parseListAdminConversations(parseLimit int64) ([]parseAdminConversationRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listAdminConversations, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseConversationRows := make([]parseAdminConversationRow, 0)
	for parseRows.Next() {
		var parseRow parseAdminConversationRow
		if parseErr2 := parseRows.Scan(
			&parseRow.ConversationID,
			&parseRow.PublicID,
			&parseRow.UserID,
			&parseRow.Email,
			&parseRow.DisplayName,
			&parseRow.StartedAt,
			&parseRow.Preview,
			&parseRow.MessageCount,
			&parseRow.UsageEventCount,
			&parseRow.TotalCostUSD,
			&parseRow.LastActivityAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseConversationRows = append(parseConversationRows, parseRow)
	}
	return parseConversationRows, parseRows.Err()
}

// parseSearchAdminUsers returns admin user rows matching one case-insensitive query.
func (parseS *Store) parseSearchAdminUsers(parseQuery string, parseLimit int64) ([]parseAdminUserRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseNeedle := strings.ToLower(strings.TrimSpace(parseQuery))
	parseRows, parseErr := parseS.parseListAdminUsers(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseMatchedRows := make([]parseAdminUserRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseNeedle == "" {
			parseMatchedRows = append(parseMatchedRows, parseRow)
		} else {
			parseUserIDValue := strconv.FormatInt(parseRow.UserID, 10)
			parseEmailValue := strings.ToLower(strings.TrimSpace(parseRow.Email))
			parseDisplayNameValue := strings.ToLower(strings.TrimSpace(parseRow.DisplayName))
			if strings.Contains(parseUserIDValue, parseNeedle) || strings.Contains(parseEmailValue, parseNeedle) || strings.Contains(parseDisplayNameValue, parseNeedle) {
				parseMatchedRows = append(parseMatchedRows, parseRow)
			}
		}
		if int64(len(parseMatchedRows)) >= parseLimit {
			break
		}
	}
	return parseMatchedRows, nil
}

// parseGetAdminUserSummaryByUserID returns one admin user summary row by user id.
func (parseS *Store) parseGetAdminUserSummaryByUserID(parseUserID int64) (parseAdminUserRow, bool, error) {
	if parseUserID <= 0 {
		return parseAdminUserRow{}, false, nil
	}
	parseRows, parseErr := parseS.parseListAdminUsers(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseAdminUserRow{}, false, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.UserID != parseUserID {
			continue
		}
		return parseRow, true, nil
	}
	return parseAdminUserRow{}, false, nil
}

// parseListAdminAuthSessionsByUser returns recent auth sessions for one user id.
func (parseS *Store) parseListAdminAuthSessionsByUser(parseUserID int64, parseLimit int64) ([]parseAuthSessionRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListAuthSessions(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseSessionRows := make([]parseAuthSessionRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.UserID != parseUserID {
			continue
		}
		parseSessionRows = append(parseSessionRows, parseRow)
		if int64(len(parseSessionRows)) >= parseLimit {
			break
		}
	}
	return parseSessionRows, nil
}

// parseListAdminWorkspaceMembershipsByUser returns workspace-membership rows for one user id.
func (parseS *Store) parseListAdminWorkspaceMembershipsByUser(parseUserID int64, parseLimit int64) ([]parseWorkspaceMembershipRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListWorkspaceMembershipsByUser(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if int64(len(parseRows)) > parseLimit {
		return parseRows[:parseLimit], nil
	}
	return parseRows, nil
}

// parseListAdminWorkspacesByUser returns workspace rows linked to one user membership set.
func (parseS *Store) parseListAdminWorkspacesByUser(parseUserID int64, parseLimit int64) ([]parseWorkspaceRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseMembershipRows, parseErr := parseS.parseListWorkspaceMembershipsByUser(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	parseWorkspaceRows := make([]parseWorkspaceRow, 0, len(parseMembershipRows))
	parseSeenWorkspaceID := make(map[int64]struct{}, len(parseMembershipRows))
	for _, parseMembershipRow := range parseMembershipRows {
		if _, hasParseWorkspaceID := parseSeenWorkspaceID[parseMembershipRow.WorkspaceID]; hasParseWorkspaceID {
			continue
		}
		parseWorkspaceRow, hasParseWorkspaceRow, parseWorkspaceErr := parseS.parseGetWorkspaceByID(parseMembershipRow.WorkspaceID)
		if parseWorkspaceErr != nil {
			return nil, parseWorkspaceErr
		}
		if !hasParseWorkspaceRow {
			continue
		}
		parseSeenWorkspaceID[parseMembershipRow.WorkspaceID] = struct{}{}
		parseWorkspaceRows = append(parseWorkspaceRows, parseWorkspaceRow)
		if int64(len(parseWorkspaceRows)) >= parseLimit {
			break
		}
	}
	return parseWorkspaceRows, nil
}

// parseListAdminUserMemoriesByUser returns memory rows for one user id.
func (parseS *Store) parseListAdminUserMemoriesByUser(parseUserID int64, parseLimit int64) ([]userMemoryRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListUserMemories(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if int64(len(parseRows)) > parseLimit {
		return parseRows[:parseLimit], nil
	}
	return parseRows, nil
}

// parseListAdminSupportTicketsByUser returns support-ticket rows for one user id.
func (parseS *Store) parseListAdminSupportTicketsByUser(parseUserID int64, parseLimit int64) ([]parseSupportTicketRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListSupportTickets(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseTicketRows := make([]parseSupportTicketRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.UserID != parseUserID {
			continue
		}
		parseTicketRows = append(parseTicketRows, parseRow)
		if int64(len(parseTicketRows)) >= parseLimit {
			break
		}
	}
	return parseTicketRows, nil
}

// parseListAdminSupportTicketMessagesByUser returns support-ticket message rows linked to one user's tickets.
func (parseS *Store) parseListAdminSupportTicketMessagesByUser(parseUserID int64, parseLimit int64) ([]parseSupportTicketMessageRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseTicketRows, parseErr := parseS.parseListAdminSupportTicketsByUser(parseUserID, parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	if len(parseTicketRows) == 0 {
		return nil, nil
	}
	parseTicketIDSet := make(map[int64]struct{}, len(parseTicketRows))
	for _, parseTicketRow := range parseTicketRows {
		parseTicketIDSet[parseTicketRow.ID] = struct{}{}
	}
	parseRows, parseErr := parseS.parseListSupportTicketMessages(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseMessageRows := make([]parseSupportTicketMessageRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseTicket := parseTicketIDSet[parseRow.TicketID]; !hasParseTicket {
			continue
		}
		parseMessageRows = append(parseMessageRows, parseRow)
		if int64(len(parseMessageRows)) >= parseLimit {
			break
		}
	}
	return parseMessageRows, nil
}

// parseListAdminBillingSubscriptionsByUser returns billing-subscription rows for one user id.
func (parseS *Store) parseListAdminBillingSubscriptionsByUser(parseUserID int64, parseLimit int64) ([]parseBillingSubscriptionRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseCustomerRow, hasParseCustomerRow, parseErr := parseS.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasParseCustomerRow || parseCustomerRow.ID <= 0 {
		return nil, nil
	}
	return parseS.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, parseLimit)
}

// parseGetAdminAuthTokenVersionByUser returns one auth token version value for one user id.
func (parseS *Store) parseGetAdminAuthTokenVersionByUser(parseUserID int64) (int64, error) {
	if parseUserID <= 0 {
		return 0, nil
	}
	return parseS.parseGetAuthTokenVersion(parseUserID)
}

// parseListAdminUsageEventsByUser returns recent admin-usage rows for one user id in one lookback window.
func (parseS *Store) parseListAdminUsageEventsByUser(parseUserID int64, parseSince string, parseLimit int64) ([]parseAdminUsageEventRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListAdminUsageEvents(parseSince, parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseUsageRows := make([]parseAdminUsageEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.UserID != parseUserID {
			continue
		}
		parseUsageRows = append(parseUsageRows, parseRow)
		if int64(len(parseUsageRows)) >= parseLimit {
			break
		}
	}
	return parseUsageRows, nil
}

// parseListAdminAuditLogsByUser returns recent audit rows where one user is the actor or typed target.
func (parseS *Store) parseListAdminAuditLogsByUser(parseUserID int64, parseLimit int64) ([]parseAuditLogRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseTargetID := strconv.FormatInt(parseUserID, 10)
	parseRows, parseErr := parseS.parseListAuditLogs(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseAuditRows := make([]parseAuditLogRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		hasParseActorMatch := parseRow.ActorUserID == parseUserID
		hasParseTargetMatch := strings.EqualFold(strings.TrimSpace(parseRow.TargetType), "user") && strings.TrimSpace(parseRow.TargetID) == parseTargetID
		if !hasParseActorMatch && !hasParseTargetMatch {
			continue
		}
		parseAuditRows = append(parseAuditRows, parseRow)
		if int64(len(parseAuditRows)) >= parseLimit {
			break
		}
	}
	return parseAuditRows, nil
}

// parseGetAdminWorkspaceByWorkspaceID returns one workspace row by workspace id.
func (parseS *Store) parseGetAdminWorkspaceByWorkspaceID(parseWorkspaceID int64) (parseWorkspaceRow, bool, error) {
	return parseS.parseGetWorkspaceByID(parseWorkspaceID)
}

// parseListAdminWorkspaceMembershipsByWorkspace returns membership rows for one workspace id.
func (parseS *Store) parseListAdminWorkspaceMembershipsByWorkspace(parseWorkspaceID int64, parseLimit int64) ([]parseWorkspaceMembershipRow, error) {
	if parseWorkspaceID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	if int64(len(parseRows)) > parseLimit {
		return parseRows[:parseLimit], nil
	}
	return parseRows, nil
}

// parseListAdminWorkspaceInvitationsByWorkspace returns invitation rows for one workspace id.
func (parseS *Store) parseListAdminWorkspaceInvitationsByWorkspace(parseWorkspaceID int64, parseLimit int64) ([]parseWorkspaceInvitationRow, error) {
	if parseWorkspaceID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListWorkspaceInvitations(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseInvitationRows := make([]parseWorkspaceInvitationRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseInvitationRows = append(parseInvitationRows, parseRow)
		if int64(len(parseInvitationRows)) >= parseLimit {
			break
		}
	}
	return parseInvitationRows, nil
}

// parseListAdminAPIKeysByWorkspace returns API-key rows for one workspace id.
func (parseS *Store) parseListAdminAPIKeysByWorkspace(parseWorkspaceID int64, parseLimit int64) ([]parseAPIKeyRow, error) {
	if parseWorkspaceID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListAPIKeys(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseKeyRows := make([]parseAPIKeyRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseKeyRows = append(parseKeyRows, parseRow)
		if int64(len(parseKeyRows)) >= parseLimit {
			break
		}
	}
	return parseKeyRows, nil
}

// parseGetAdminAPIKeyByKeyID returns one API-key row by key id.
func (parseS *Store) parseGetAdminAPIKeyByKeyID(parseKeyID string) (parseAPIKeyRow, bool, error) {
	parseKeyID = strings.TrimSpace(parseKeyID)
	if parseKeyID == "" {
		return parseAPIKeyRow{}, false, nil
	}
	parseRows, parseErr := parseS.parseListAPIKeys(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseAPIKeyRow{}, false, parseErr
	}
	for _, parseRow := range parseRows {
		if strings.TrimSpace(parseRow.KeyID) != parseKeyID {
			continue
		}
		return parseRow, true, nil
	}
	return parseAPIKeyRow{}, false, nil
}

// parseListAdminWebhookEndpointsByWorkspace returns webhook endpoint rows for one workspace id.
func (parseS *Store) parseListAdminWebhookEndpointsByWorkspace(parseWorkspaceID int64, parseLimit int64) ([]parseWebhookEndpointRow, error) {
	if parseWorkspaceID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListWebhookEndpoints(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseEndpointRows := make([]parseWebhookEndpointRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseEndpointRows = append(parseEndpointRows, parseRow)
		if int64(len(parseEndpointRows)) >= parseLimit {
			break
		}
	}
	return parseEndpointRows, nil
}

// parseGetAdminWebhookEndpointByID returns one webhook endpoint row by numeric endpoint id.
func (parseS *Store) parseGetAdminWebhookEndpointByID(parseEndpointID int64) (parseWebhookEndpointRow, bool, error) {
	if parseEndpointID <= 0 {
		return parseWebhookEndpointRow{}, false, nil
	}
	parseRows, parseErr := parseS.parseListWebhookEndpoints(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseWebhookEndpointRow{}, false, parseErr
	}
	for _, parseRow := range parseRows {
		if parseRow.ID != parseEndpointID {
			continue
		}
		return parseRow, true, nil
	}
	return parseWebhookEndpointRow{}, false, nil
}

// parseListAdminAuditLogsByWorkspace returns audit rows for one workspace id.
func (parseS *Store) parseListAdminAuditLogsByWorkspace(parseWorkspaceID int64, parseLimit int64) ([]parseAuditLogRow, error) {
	if parseWorkspaceID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.parseListAuditLogs(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseAuditRows := make([]parseAuditLogRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseAuditRows = append(parseAuditRows, parseRow)
		if int64(len(parseAuditRows)) >= parseLimit {
			break
		}
	}
	return parseAuditRows, nil
}

// parseGetAdminBillingCustomerByUserID resolves one billing customer row by one admin-target user id.
func (parseS *Store) parseGetAdminBillingCustomerByUserID(parseUserID int64) (parseBillingCustomerRow, bool, error) {
	if parseUserID <= 0 {
		return parseBillingCustomerRow{}, false, nil
	}
	return parseS.parseGetBillingCustomerByUser(parseUserID)
}

// parseListAdminBillingEventsByUser returns one billing-event inspection slice for one admin-target user id.
func (parseS *Store) parseListAdminBillingEventsByUser(parseUserID int64, parseLimit int64) ([]parseBillingEventRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseCustomer, hasParseCustomer, parseErr := parseS.parseGetAdminBillingCustomerByUserID(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasParseCustomer || parseCustomer.ID <= 0 {
		return nil, nil
	}
	return parseS.parseListBillingEventsByCustomer(parseCustomer.ID, parseLimit)
}

// parseListAdminBillingDunningEventsByUser returns one dunning-review slice for one admin-target user id.
func (parseS *Store) parseListAdminBillingDunningEventsByUser(parseUserID int64, parseLimit int64) ([]parseBillingDunningEventRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseCustomer, hasParseCustomer, parseErr := parseS.parseGetAdminBillingCustomerByUserID(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasParseCustomer || parseCustomer.ID <= 0 {
		return nil, nil
	}
	return parseS.parseListBillingDunningEventsByCustomer(parseCustomer.ID, parseLimit)
}

// parseListAdminBillingAccessOverridesByUser returns one access-override slice for one admin-target user id.
func (parseS *Store) parseListAdminBillingAccessOverridesByUser(parseUserID int64) ([]parseBillingAccessOverrideRow, error) {
	if parseUserID <= 0 {
		return nil, nil
	}
	parseCustomer, hasParseCustomer, parseErr := parseS.parseGetAdminBillingCustomerByUserID(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasParseCustomer || parseCustomer.ID <= 0 {
		return nil, nil
	}
	return parseS.parseListBillingAccessOverridesByCustomer(parseCustomer.ID)
}

// parseSetAdminBillingAccessOverrideByUser stores one typed access override for one admin-target user id.
func (parseS *Store) parseSetAdminBillingAccessOverrideByUser(parseUserID int64, parseWrite parseBillingAccessOverrideWrite) (parseBillingAccessOverrideRow, error) {
	return parseS.parseStoreAdminBillingOverrideByUser(parseUserID, parseWrite, "billing.access_override.updated")
}

// parseSetAdminBillingQuotaOverrideByUser stores one typed quota override for one admin-target user id.
func (parseS *Store) parseSetAdminBillingQuotaOverrideByUser(parseUserID int64, parseWrite parseBillingAccessOverrideWrite) (parseBillingAccessOverrideRow, error) {
	parseWrite.OverrideKey = parseNormalizeAdminBillingQuotaOverrideKey(parseWrite.OverrideKey)
	return parseS.parseStoreAdminBillingOverrideByUser(parseUserID, parseWrite, "billing.quota_override.updated")
}

// parseStoreAdminBillingOverrideByUser persists one admin billing override and appends one immutable billing event.
func (parseS *Store) parseStoreAdminBillingOverrideByUser(parseUserID int64, parseWrite parseBillingAccessOverrideWrite, parseEventType string) (parseBillingAccessOverrideRow, error) {
	if parseUserID <= 0 {
		return parseBillingAccessOverrideRow{}, errStoreUserMissing
	}
	parseCustomer, hasParseCustomer, parseErr := parseS.parseGetAdminBillingCustomerByUserID(parseUserID)
	if parseErr != nil {
		return parseBillingAccessOverrideRow{}, parseErr
	}
	if !hasParseCustomer || parseCustomer.ID <= 0 {
		return parseBillingAccessOverrideRow{}, errStoreBillingCustomerMissing
	}
	parseWrite.CustomerID = parseCustomer.ID
	parseWrite.OverrideKey = strings.TrimSpace(parseWrite.OverrideKey)
	if parseWrite.OverrideKey == "" {
		return parseBillingAccessOverrideRow{}, errStoreBillingScopeMissing
	}
	if parseErr = parseValidateUsageBasedBillingOverrideWrite(parseWrite.OverrideKey, parseWrite.OverrideValue); parseErr != nil {
		return parseBillingAccessOverrideRow{}, parseErr
	}
	if parseErr = parseS.parseUpsertBillingAccessOverride(parseWrite); parseErr != nil {
		return parseBillingAccessOverrideRow{}, parseErr
	}
	parseEventSummary := strings.TrimSpace(parseWrite.Reason)
	if parseEventSummary == "" {
		parseEventSummary = "Admin billing override updated"
	}
	parseSubscriptionID := int64(0)
	parseInvoiceID := int64(0)
	parseInvoiceRows, parseErr := parseS.parseListBillingInvoicesByCustomer(parseCustomer.ID, 1)
	if parseErr != nil {
		return parseBillingAccessOverrideRow{}, parseErr
	}
	if len(parseInvoiceRows) > 0 {
		parseInvoiceID = parseInvoiceRows[0].ID
		parseSubscriptionID = parseInvoiceRows[0].SubscriptionID
	}
	if parseSubscriptionID <= 0 {
		parseSubscriptionRows, parseErr := parseS.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 1)
		if parseErr != nil {
			return parseBillingAccessOverrideRow{}, parseErr
		}
		if len(parseSubscriptionRows) > 0 {
			parseSubscriptionID = parseSubscriptionRows[0].ID
		}
	}
	if parseSubscriptionID > 0 {
		if _, parseErr = parseS.parseCreateBillingEvent(parseBillingEventWrite{
			CustomerID:     parseCustomer.ID,
			SubscriptionID: parseSubscriptionID,
			InvoiceID:      parseInvoiceID,
			EventType:      strings.TrimSpace(parseEventType),
			EventSource:    "admin",
			EventSummary:   parseEventSummary,
			ActorUserID:    parseWrite.ActorUserID,
		}); parseErr != nil {
			return parseBillingAccessOverrideRow{}, parseErr
		}
	}
	parseOverrideRows, parseErr := parseS.parseListBillingAccessOverridesByCustomer(parseCustomer.ID)
	if parseErr != nil {
		return parseBillingAccessOverrideRow{}, parseErr
	}
	for _, parseOverrideRow := range parseOverrideRows {
		if strings.EqualFold(strings.TrimSpace(parseOverrideRow.OverrideKey), parseWrite.OverrideKey) {
			return parseOverrideRow, nil
		}
	}
	return parseBillingAccessOverrideRow{}, errStoreBillingScopeMissing
}

// parseResolveAdminBillingFailedPaymentByUser resolves one failed-payment state by setting invoice paid and closing related dunning rows.
func (parseS *Store) parseResolveAdminBillingFailedPaymentByUser(parseUserID, parseInvoiceID, parseActorUserID int64, parseReason string) (parseBillingInvoiceRow, error) {
	if parseUserID <= 0 {
		return parseBillingInvoiceRow{}, errStoreUserMissing
	}
	if parseInvoiceID <= 0 {
		return parseBillingInvoiceRow{}, errStoreBillingInvoiceMissing
	}
	parseCustomer, hasParseCustomer, parseErr := parseS.parseGetAdminBillingCustomerByUserID(parseUserID)
	if parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	if !hasParseCustomer || parseCustomer.ID <= 0 {
		return parseBillingInvoiceRow{}, errStoreBillingCustomerMissing
	}
	parseInvoiceRows, parseErr := parseS.parseListBillingInvoicesByCustomer(parseCustomer.ID, parseAdminControlScanLimit)
	if parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	parseTargetInvoice := parseBillingInvoiceRow{}
	hasParseTargetInvoice := false
	for _, parseInvoiceRow := range parseInvoiceRows {
		if parseInvoiceRow.ID != parseInvoiceID {
			continue
		}
		parseTargetInvoice = parseInvoiceRow
		hasParseTargetInvoice = true
		break
	}
	if !hasParseTargetInvoice {
		return parseBillingInvoiceRow{}, errStoreBillingInvoiceMissing
	}
	parseResolvedAt := time.Now().UTC().Format(time.RFC3339)
	if parseErr = parseS.parseSetBillingInvoiceResolution(parseTargetInvoice.ID, parseCustomer.ID, "paid", parseResolvedAt); parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	parseDunningRows, parseErr := parseS.parseListBillingDunningEventsByCustomer(parseCustomer.ID, parseAdminControlScanLimit)
	if parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	for _, parseDunningRow := range parseDunningRows {
		if parseDunningRow.InvoiceID != parseTargetInvoice.ID {
			continue
		}
		if parseIsAdminDunningTerminalStatus(parseDunningRow.Status) {
			continue
		}
		if parseErr = parseS.parseSetBillingDunningEventResolved(parseDunningRow.ID, parseCustomer.ID, "resolved", parseResolvedAt); parseErr != nil {
			return parseBillingInvoiceRow{}, parseErr
		}
	}
	parseEventSummary := strings.TrimSpace(parseReason)
	if parseEventSummary == "" {
		parseEventSummary = "Admin resolved failed payment"
	}
	if _, parseErr = parseS.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseCustomer.ID,
		SubscriptionID: parseTargetInvoice.SubscriptionID,
		InvoiceID:      parseTargetInvoice.ID,
		EventType:      "invoice.payment_resolved",
		EventSource:    "admin",
		EventSummary:   parseEventSummary,
		ActorUserID:    parseActorUserID,
	}); parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	parseResolvedInvoiceRows, parseErr := parseS.parseListBillingInvoicesByCustomer(parseCustomer.ID, parseAdminControlScanLimit)
	if parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	for _, parseResolvedInvoiceRow := range parseResolvedInvoiceRows {
		if parseResolvedInvoiceRow.ID != parseTargetInvoice.ID {
			continue
		}
		return parseResolvedInvoiceRow, nil
	}
	parseTargetInvoice.Status = "paid"
	parseTargetInvoice.PaidAt = parseResolvedAt
	parseTargetInvoice.UpdatedAt = parseResolvedAt
	return parseTargetInvoice, nil
}

// parseNormalizeAdminBillingQuotaOverrideKey normalizes one operator-provided quota override key to the quota namespace.
func parseNormalizeAdminBillingQuotaOverrideKey(parseOverrideKey string) string {
	parseOverrideKey = strings.TrimSpace(strings.ToLower(parseOverrideKey))
	if parseOverrideKey == "" {
		return "quota.default"
	}
	if strings.HasPrefix(parseOverrideKey, "quota.") {
		return parseOverrideKey
	}
	return "quota." + parseOverrideKey
}

// parseIsAdminDunningTerminalStatus reports whether one dunning status is already in a terminal state.
func parseIsAdminDunningTerminalStatus(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "resolved", "closed", "succeeded":
		return true
	default:
		return false
	}
}

// parseListAdminSupportTickets returns one typed support-ticket queue slice.
func (parseS *Store) parseListAdminSupportTickets(parseLimit int64) ([]parseSupportTicketRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	return parseS.parseListSupportTickets(parseLimit)
}

// parseGetAdminSupportTicketByID returns one support-ticket row by numeric id.
func (parseS *Store) parseGetAdminSupportTicketByID(parseTicketID int64) (parseSupportTicketRow, bool, error) {
	if parseTicketID <= 0 {
		return parseSupportTicketRow{}, false, nil
	}
	parseTicketRows, parseErr := parseS.parseListSupportTickets(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseSupportTicketRow{}, false, parseErr
	}
	for _, parseTicketRow := range parseTicketRows {
		if parseTicketRow.ID != parseTicketID {
			continue
		}
		return parseTicketRow, true, nil
	}
	return parseSupportTicketRow{}, false, nil
}

// parseListAdminSupportTicketMessagesByTicketID returns support-ticket messages for one ticket id newest-first.
func (parseS *Store) parseListAdminSupportTicketMessagesByTicketID(parseTicketID, parseLimit int64) ([]parseSupportTicketMessageRow, error) {
	if parseTicketID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseMessageRows, parseErr := parseS.parseListSupportTicketMessages(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseFilteredRows := make([]parseSupportTicketMessageRow, 0, len(parseMessageRows))
	for _, parseMessageRow := range parseMessageRows {
		if parseMessageRow.TicketID != parseTicketID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseMessageRow)
		if int64(len(parseFilteredRows)) >= parseLimit {
			break
		}
	}
	return parseFilteredRows, nil
}

// parseListAdminSupportAccountActionHistoryByUser returns one typed account-linked action history slice for one user id.
func (parseS *Store) parseListAdminSupportAccountActionHistoryByUser(parseUserID, parseLimit int64) ([]parseAuditLogRow, error) {
	return parseS.parseListAdminAuditLogsByUser(parseUserID, parseLimit)
}

// parseListAdminSupportAccountActionHistoryByTicketID returns one account-linked action history slice for one support ticket id.
func (parseS *Store) parseListAdminSupportAccountActionHistoryByTicketID(parseTicketID, parseLimit int64) ([]parseAuditLogRow, error) {
	if parseTicketID <= 0 {
		return nil, nil
	}
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseTicketRow, hasParseTicket, parseErr := parseS.parseGetAdminSupportTicketByID(parseTicketID)
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasParseTicket {
		return nil, nil
	}
	parseTicketTargetID := strconv.FormatInt(parseTicketID, 10)
	parseUserTargetID := strconv.FormatInt(parseTicketRow.UserID, 10)
	parseAuditRows, parseErr := parseS.parseListAuditLogs(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseLinkedRows := make([]parseAuditLogRow, 0, len(parseAuditRows))
	for _, parseAuditRow := range parseAuditRows {
		parseTargetType := strings.TrimSpace(strings.ToLower(parseAuditRow.TargetType))
		parseTargetID := strings.TrimSpace(parseAuditRow.TargetID)
		hasParseTicketTarget := parseTargetType == "support_ticket" && parseTargetID == parseTicketTargetID
		hasParseUserTarget := parseTargetType == "user" && parseTargetID == parseUserTargetID
		hasParseUserActor := parseAuditRow.ActorUserID > 0 && parseAuditRow.ActorUserID == parseTicketRow.UserID
		if !hasParseTicketTarget && !hasParseUserTarget && !hasParseUserActor {
			continue
		}
		parseLinkedRows = append(parseLinkedRows, parseAuditRow)
		if int64(len(parseLinkedRows)) >= parseLimit {
			break
		}
	}
	return parseLinkedRows, nil
}

// parseStoreAdminSupportTicketAssignment stores one support-ticket assignment update by ticket id.
func (parseS *Store) parseStoreAdminSupportTicketAssignment(parseTicketID, parseAssigneeUserID int64) (parseSupportTicketRow, error) {
	parseTicketRow, hasParseTicket, parseErr := parseS.parseGetAdminSupportTicketByID(parseTicketID)
	if parseErr != nil {
		return parseSupportTicketRow{}, parseErr
	}
	if !hasParseTicket {
		return parseSupportTicketRow{}, errStoreSuperuserScopeMissing
	}
	parseTicketRow.AssigneeUserID = parseAssigneeUserID
	return parseS.parseStoreAdminSupportTicketUpsert(parseTicketRow)
}

// parseStoreAdminSupportTicketEscalation stores one support-ticket escalation update by ticket id.
func (parseS *Store) parseStoreAdminSupportTicketEscalation(parseTicketID int64, parsePriority, parseStatus, parseResolutionNote string) (parseSupportTicketRow, error) {
	parseTicketRow, hasParseTicket, parseErr := parseS.parseGetAdminSupportTicketByID(parseTicketID)
	if parseErr != nil {
		return parseSupportTicketRow{}, parseErr
	}
	if !hasParseTicket {
		return parseSupportTicketRow{}, errStoreSuperuserScopeMissing
	}
	parsePriority = strings.TrimSpace(strings.ToLower(parsePriority))
	if parsePriority == "" {
		parsePriority = "urgent"
	}
	parseStatus = strings.TrimSpace(strings.ToLower(parseStatus))
	if parseStatus == "" {
		parseStatus = "escalated"
	}
	parseTicketRow.Priority = parsePriority
	parseTicketRow.Status = parseStatus
	parseResolutionNote = strings.TrimSpace(parseResolutionNote)
	if parseResolutionNote != "" {
		parseTicketRow.ResolutionNote = parseResolutionNote
	}
	return parseS.parseStoreAdminSupportTicketUpsert(parseTicketRow)
}

// parseStoreAdminSupportTicketUpsert writes one updated support-ticket row and reads it back by id.
func (parseS *Store) parseStoreAdminSupportTicketUpsert(parseTicketRow parseSupportTicketRow) (parseSupportTicketRow, error) {
	if parseErr := parseS.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      parseTicketRow.TicketKey,
		WorkspaceID:    parseTicketRow.WorkspaceID,
		UserID:         parseTicketRow.UserID,
		Status:         parseTicketRow.Status,
		Priority:       parseTicketRow.Priority,
		Subject:        parseTicketRow.Subject,
		Body:           parseTicketRow.Body,
		AssigneeUserID: parseTicketRow.AssigneeUserID,
		ResolutionNote: parseTicketRow.ResolutionNote,
	}); parseErr != nil {
		return parseSupportTicketRow{}, parseErr
	}
	parseUpdatedTicketRow, hasParseTicket, parseErr := parseS.parseGetAdminSupportTicketByID(parseTicketRow.ID)
	if parseErr != nil {
		return parseSupportTicketRow{}, parseErr
	}
	if !hasParseTicket {
		return parseSupportTicketRow{}, errStoreSuperuserScopeMissing
	}
	return parseUpdatedTicketRow, nil
}

// parseGetAdminFeatureFlagByKey returns one feature-flag row by flag key.
func (parseS *Store) parseGetAdminFeatureFlagByKey(parseFlagKey string) (parseFeatureFlagRow, bool, error) {
	parseFlagKey = strings.TrimSpace(strings.ToLower(parseFlagKey))
	if parseFlagKey == "" {
		return parseFeatureFlagRow{}, false, nil
	}
	parseFlagRows, parseErr := parseS.parseListFeatureFlags()
	if parseErr != nil {
		return parseFeatureFlagRow{}, false, parseErr
	}
	for _, parseFlagRow := range parseFlagRows {
		if strings.TrimSpace(strings.ToLower(parseFlagRow.FlagKey)) != parseFlagKey {
			continue
		}
		return parseFlagRow, true, nil
	}
	return parseFeatureFlagRow{}, false, nil
}

// parseStoreAdminFeatureFlagToggle stores one typed feature-flag control mutation.
func (parseS *Store) parseStoreAdminFeatureFlagToggle(parseFlagKey string, isParseEnabled bool, parseRolloutPercent int64, parseAudienceJSON, parsePayloadJSON, parseReason string, parseActorUserID int64) (parseFeatureFlagRow, error) {
	parseFlagRow, hasParseFlag, parseErr := parseS.parseGetAdminFeatureFlagByKey(parseFlagKey)
	if parseErr != nil {
		return parseFeatureFlagRow{}, parseErr
	}
	parseDescription := strings.TrimSpace(parseReason)
	if hasParseFlag && strings.TrimSpace(parseFlagRow.Description) != "" {
		parseDescription = strings.TrimSpace(parseFlagRow.Description)
	}
	if parseDescription == "" {
		parseDescription = "Admin feature-flag control update"
	}
	if parseRolloutPercent <= 0 && hasParseFlag {
		parseRolloutPercent = parseFlagRow.RolloutPercent
	}
	if parseErr = parseS.parseUpsertFeatureFlag(parseFeatureFlagWrite{
		FlagKey:         strings.TrimSpace(parseFlagKey),
		Description:     parseDescription,
		IsEnabled:       isParseEnabled,
		RolloutPercent:  parseRolloutPercent,
		AudienceJSON:    strings.TrimSpace(parseAudienceJSON),
		PayloadJSON:     strings.TrimSpace(parsePayloadJSON),
		UpdatedByUserID: parseActorUserID,
	}); parseErr != nil {
		return parseFeatureFlagRow{}, parseErr
	}
	parseStoredFlagRow, hasParseStoredFlag, parseErr := parseS.parseGetAdminFeatureFlagByKey(parseFlagKey)
	if parseErr != nil {
		return parseFeatureFlagRow{}, parseErr
	}
	if !hasParseStoredFlag {
		return parseFeatureFlagRow{}, errStoreSuperuserScopeMissing
	}
	return parseStoredFlagRow, nil
}

// parseGetAdminExperimentByKey returns one experiment row by experiment key.
func (parseS *Store) parseGetAdminExperimentByKey(parseExperimentKey string) (parseExperimentRow, bool, error) {
	parseExperimentKey = strings.TrimSpace(strings.ToLower(parseExperimentKey))
	if parseExperimentKey == "" {
		return parseExperimentRow{}, false, nil
	}
	parseExperimentRows, parseErr := parseS.parseListExperiments(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseExperimentRow{}, false, parseErr
	}
	for _, parseExperimentRow := range parseExperimentRows {
		if strings.TrimSpace(strings.ToLower(parseExperimentRow.ExperimentKey)) != parseExperimentKey {
			continue
		}
		return parseExperimentRow, true, nil
	}
	return parseExperimentRow{}, false, nil
}

// parseStoreAdminExperimentRollback stores one typed experiment rollback mutation.
func (parseS *Store) parseStoreAdminExperimentRollback(parseExperimentKey, parseReason string) (parseExperimentRow, error) {
	parseExperimentRow, hasParseExperiment, parseErr := parseS.parseGetAdminExperimentByKey(parseExperimentKey)
	if parseErr != nil {
		return parseExperimentRow, parseErr
	}
	if !hasParseExperiment {
		return parseExperimentRow, errStoreSuperuserScopeMissing
	}
	parseEndAt := strings.TrimSpace(parseExperimentRow.EndAt)
	if parseEndAt == "" {
		parseEndAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseName := strings.TrimSpace(parseExperimentRow.Name)
	if parseName == "" {
		parseName = strings.TrimSpace(parseReason)
	}
	if parseErr = parseS.parseUpsertExperiment(parseExperimentWrite{
		ExperimentKey: parseExperimentRow.ExperimentKey,
		Name:          parseName,
		Status:        "rolled_back",
		VariantsJSON:  parseExperimentRow.VariantsJSON,
		AudienceJSON:  parseExperimentRow.AudienceJSON,
		StartAt:       parseExperimentRow.StartAt,
		EndAt:         parseEndAt,
	}); parseErr != nil {
		return parseExperimentRow, parseErr
	}
	parseStoredExperimentRow, hasParseStoredExperiment, parseErr := parseS.parseGetAdminExperimentByKey(parseExperimentRow.ExperimentKey)
	if parseErr != nil {
		return parseExperimentRow, parseErr
	}
	if !hasParseStoredExperiment {
		return parseExperimentRow, errStoreSuperuserScopeMissing
	}
	return parseStoredExperimentRow, nil
}

// parseGetAdminIncidentByID returns one incident row by numeric incident id.
func (parseS *Store) parseGetAdminIncidentByID(parseIncidentID int64) (parseIncidentRow, bool, error) {
	if parseIncidentID <= 0 {
		return parseIncidentRow{}, false, nil
	}
	parseIncidentRows, parseErr := parseS.parseListIncidents(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseIncidentRow{}, false, parseErr
	}
	for _, parseIncidentRow := range parseIncidentRows {
		if parseIncidentRow.ID != parseIncidentID {
			continue
		}
		return parseIncidentRow, true, nil
	}
	return parseIncidentRow{}, false, nil
}

// parseSetAdminIncidentStatus updates one incident status and resolved timestamp by incident id.
func (parseS *Store) parseSetAdminIncidentStatus(parseIncidentID int64, parseStatus string) error {
	if parseIncidentID <= 0 {
		return errStoreSuperuserScopeMissing
	}
	parseStatus = strings.TrimSpace(strings.ToLower(parseStatus))
	if parseStatus == "" {
		parseStatus = "investigating"
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResolvedAt := ""
	if parseHasAdminIncidentResolvedStatus(parseStatus) {
		parseResolvedAt = parseNow
	}
	parseResult, parseErr := parseS.db.Exec(parseS.queries.updateIncidentStatus, parseStatus, parseResolvedAt, parseNow, parseIncidentID)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreSuperuserScopeMissing
	}
	return nil
}

// parseStoreAdminIncidentStatusUpdate stores one typed incident status change with one incident-update row.
func (parseS *Store) parseStoreAdminIncidentStatusUpdate(parseIncidentID int64, parseStatus, parseMessage string, isParsePublic bool, parseReason string, parseActorUserID int64) (parseIncidentRow, parseIncidentUpdateRow, error) {
	parseIncidentRow, hasParseIncident, parseErr := parseS.parseGetAdminIncidentByID(parseIncidentID)
	if parseErr != nil {
		return parseIncidentRow, parseIncidentUpdateRow{}, parseErr
	}
	if !hasParseIncident {
		return parseIncidentRow, parseIncidentUpdateRow{}, errStoreSuperuserScopeMissing
	}
	parseStatus = strings.TrimSpace(strings.ToLower(parseStatus))
	if parseStatus == "" {
		parseStatus = strings.TrimSpace(strings.ToLower(parseIncidentRow.Status))
	}
	if parseStatus == "" {
		parseStatus = "investigating"
	}
	if parseErr = parseS.parseSetAdminIncidentStatus(parseIncidentID, parseStatus); parseErr != nil {
		return parseIncidentRow, parseIncidentUpdateRow{}, parseErr
	}
	parseMessage = strings.TrimSpace(parseMessage)
	if parseMessage == "" {
		parseMessage = strings.TrimSpace(parseReason)
	}
	if parseMessage == "" {
		parseMessage = "Admin incident status updated"
	}
	parsePublishedAt := ""
	if isParsePublic {
		parsePublishedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseUpdateID, parseErr := parseS.parseCreateIncidentUpdate(parseIncidentUpdateWrite{
		IncidentID:      parseIncidentID,
		Status:          parseStatus,
		Message:         parseMessage,
		IsPublic:        isParsePublic,
		PublishedAt:     parsePublishedAt,
		CreatedByUserID: parseActorUserID,
	})
	if parseErr != nil {
		return parseIncidentRow, parseIncidentUpdateRow{}, parseErr
	}
	parseStoredIncidentRow, hasParseStoredIncident, parseErr := parseS.parseGetAdminIncidentByID(parseIncidentID)
	if parseErr != nil {
		return parseIncidentRow, parseIncidentUpdateRow{}, parseErr
	}
	if !hasParseStoredIncident {
		return parseIncidentRow, parseIncidentUpdateRow{}, errStoreSuperuserScopeMissing
	}
	parseUpdateRows, parseErr := parseS.parseListIncidentUpdates(parseAdminControlScanLimit)
	if parseErr != nil {
		return parseIncidentRow, parseIncidentUpdateRow{}, parseErr
	}
	for _, parseUpdateRow := range parseUpdateRows {
		if parseUpdateRow.ID != parseUpdateID {
			continue
		}
		return parseStoredIncidentRow, parseUpdateRow, nil
	}
	return parseStoredIncidentRow, parseIncidentUpdateRow{}, errStoreSuperuserScopeMissing
}

// parseHasAdminIncidentResolvedStatus reports whether one incident status should set resolved_at.
func parseHasAdminIncidentResolvedStatus(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "resolved", "closed":
		return true
	default:
		return false
	}
}
