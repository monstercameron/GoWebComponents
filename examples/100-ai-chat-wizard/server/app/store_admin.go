package app

type parseAdminDashboardSummaryRow struct {
	TotalUsers                int64
	TotalConversations        int64
	TotalMessages             int64
	WindowNewUsers            int64
	WindowNewConversations    int64
	WindowNewMessages         int64
	WindowUsageEvents         int64
	WindowTotalCostUSD        float64
	WindowPromptTokens        int64
	WindowCompletionTokens    int64
	WindowActiveUsers         int64
	WindowActiveConversations int64
	WindowActiveClients       int64
	WindowCompletedEvents     int64
	WindowFailedEvents        int64
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
	UserID            int64
	Email             string
	DisplayName       string
	CreatedAt         string
	ConversationCount int64
	MessageCount      int64
	UsageEventCount   int64
	TotalCostUSD      float64
	LastSeenAt        string
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
