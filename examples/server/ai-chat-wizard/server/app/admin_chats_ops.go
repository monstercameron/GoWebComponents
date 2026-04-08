package app

import (
	"context"
	"log/slog"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildOnboardingTemplateEntry maps one onboarding-template row into protobuf form.
func parseBuildOnboardingTemplateEntry(parseRow parseOnboardingTemplateRow) *chatpb.OnboardingTemplateEntry {
	return &chatpb.OnboardingTemplateEntry{
		Id:            parseRow.ID,
		TemplateKey:   parseRow.TemplateKey,
		Title:         parseRow.Title,
		Category:      parseRow.Category,
		PromptText:    parseRow.PromptText,
		ChecklistJson: parseRow.ChecklistJSON,
		IsDefault:     parseRow.IsDefault,
		SortOrder:     parseRow.SortOrder,
		UpdatedAt:     parseRow.UpdatedAt,
	}
}

// parseBuildUserActivationMilestoneEntry maps one user-activation milestone row into protobuf form.
func parseBuildUserActivationMilestoneEntry(parseRow parseUserActivationMilestoneRow) *chatpb.UserActivationMilestoneEntry {
	return &chatpb.UserActivationMilestoneEntry{
		Id:           parseRow.ID,
		UserId:       parseRow.UserID,
		MilestoneKey: parseRow.MilestoneKey,
		Status:       parseRow.Status,
		AchievedAt:   parseRow.AchievedAt,
		MetadataJson: parseRow.MetadataJSON,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseBuildSavedWorkflowEntry maps one saved-workflow row into protobuf form.
func parseBuildSavedWorkflowEntry(parseRow parseSavedWorkflowRow) *chatpb.SavedWorkflowEntry {
	return &chatpb.SavedWorkflowEntry{
		Id:           parseRow.ID,
		WorkspaceId:  parseRow.WorkspaceID,
		UserId:       parseRow.UserID,
		WorkflowKey:  parseRow.WorkflowKey,
		Name:         parseRow.Name,
		Description:  parseRow.Description,
		WorkflowJson: parseRow.WorkflowJSON,
		IsPublic:     parseRow.IsPublic,
		CreatedAt:    parseRow.CreatedAt,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseBuildPromptLibraryItemEntry maps one prompt-library row into protobuf form.
func parseBuildPromptLibraryItemEntry(parseRow parsePromptLibraryItemRow) *chatpb.PromptLibraryItemEntry {
	return &chatpb.PromptLibraryItemEntry{
		Id:          parseRow.ID,
		WorkspaceId: parseRow.WorkspaceID,
		UserId:      parseRow.UserID,
		ItemKey:     parseRow.ItemKey,
		Title:       parseRow.Title,
		Category:    parseRow.Category,
		PromptText:  parseRow.PromptText,
		TagsJson:    parseRow.TagsJSON,
		IsPublic:    parseRow.IsPublic,
		UseCount:    parseRow.UseCount,
		CreatedAt:   parseRow.CreatedAt,
		UpdatedAt:   parseRow.UpdatedAt,
	}
}

// parseBuildWeeklyValueSummaryEntry maps one weekly-value summary row into protobuf form.
func parseBuildWeeklyValueSummaryEntry(parseRow parseWeeklyValueSummaryRow) *chatpb.WeeklyValueSummaryEntry {
	return &chatpb.WeeklyValueSummaryEntry{
		Id:          parseRow.ID,
		WorkspaceId: parseRow.WorkspaceID,
		UserId:      parseRow.UserID,
		SummaryWeek: parseRow.SummaryWeek,
		SummaryText: parseRow.SummaryText,
		MetricsJson: parseRow.MetricsJSON,
		SentAt:      parseRow.SentAt,
		CreatedAt:   parseRow.CreatedAt,
	}
}

// parseFilterActivationMilestoneRowsByScope filters milestone rows by one scoped user-id set.
func parseFilterActivationMilestoneRowsByScope(parseRows []parseUserActivationMilestoneRow, parseUserIDs map[int64]struct{}) []parseUserActivationMilestoneRow {
	parseFilteredRows := make([]parseUserActivationMilestoneRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseUser := parseUserIDs[parseRow.UserID]; !hasParseUser {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterSavedWorkflowRowsByScope filters saved-workflow rows by one scoped workspace-id set.
func parseFilterSavedWorkflowRowsByScope(parseRows []parseSavedWorkflowRow, parseWorkspaceIDs map[int64]struct{}) []parseSavedWorkflowRow {
	parseFilteredRows := make([]parseSavedWorkflowRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterPromptLibraryRowsByScope filters prompt-library rows by one scoped workspace-id set.
func parseFilterPromptLibraryRowsByScope(parseRows []parsePromptLibraryItemRow, parseWorkspaceIDs map[int64]struct{}) []parsePromptLibraryItemRow {
	parseFilteredRows := make([]parsePromptLibraryItemRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterWeeklyValueRowsByScope filters weekly-value rows by one scoped workspace-id set.
func parseFilterWeeklyValueRowsByScope(parseRows []parseWeeklyValueSummaryRow, parseWorkspaceIDs map[int64]struct{}) []parseWeeklyValueSummaryRow {
	parseFilteredRows := make([]parseWeeklyValueSummaryRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterProductAnalyticsRowsByScope filters product-analytics rows by one scoped workspace-id set.
func parseFilterProductAnalyticsRowsByScope(parseRows []parseProductAnalyticsEventRow, parseWorkspaceIDs map[int64]struct{}) []parseProductAnalyticsEventRow {
	parseFilteredRows := make([]parseProductAnalyticsEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminChatsDrilldownSummary derives one chats drill-down summary from typed row slices.
func parseBuildAdminChatsDrilldownSummary(
	parseConversationRows []parseAdminConversationRow,
	parseUsageRows []parseAdminUsageEventRow,
	parseTemplateRows []parseOnboardingTemplateRow,
	parseMilestoneRows []parseUserActivationMilestoneRow,
	parseWorkflowRows []parseSavedWorkflowRow,
	parsePromptRows []parsePromptLibraryItemRow,
	parseWeeklyRows []parseWeeklyValueSummaryRow,
	parseAnalyticsRows []parseProductAnalyticsEventRow,
) *chatpb.AdminChatsDrilldownSummary {
	parseSummary := &chatpb.AdminChatsDrilldownSummary{
		ConversationCount:          int64(len(parseConversationRows)),
		UsageEventCount:            int64(len(parseUsageRows)),
		OnboardingTemplateCount:    int64(len(parseTemplateRows)),
		ActivationMilestoneCount:   int64(len(parseMilestoneRows)),
		SavedWorkflowCount:         int64(len(parseWorkflowRows)),
		PromptLibraryItemCount:     int64(len(parsePromptRows)),
		WeeklyValueSummaryCount:    int64(len(parseWeeklyRows)),
		ProductAnalyticsEventCount: int64(len(parseAnalyticsRows)),
	}
	for _, parseConversationRow := range parseConversationRows {
		parseSummary.MessageCount += parseConversationRow.MessageCount
	}
	for _, parseUsageRow := range parseUsageRows {
		parseSummary.TotalCostUsd += parseUsageRow.TotalCostUSD
		switch parseUsageRow.Status {
		case "completed":
			parseSummary.CompletedReplyCount++
		case "failed":
			parseSummary.FailedReplyCount++
		}
	}
	return parseSummary
}

// GetAdminChatsDrilldown returns one typed chats drill-down snapshot for dashboard chats surfaces.
func (parseS *chatServer) GetAdminChatsDrilldown(parseCtx context.Context, parseReq *chatpb.GetAdminChatsDrilldownRequest) (*chatpb.GetAdminChatsDrilldownResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminChatsDrilldown"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.conversations")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLookbackDays int32
	var parseLimit int32
	if parseReq != nil {
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminChatsDrilldown: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"chats",
		"Admin chats slice viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"slice",
		"chats",
		"Admin chats drill-down viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseResponse := &chatpb.GetAdminChatsDrilldownResponse{
		Summary: &chatpb.AdminChatsDrilldownSummary{},
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminChatsDrilldown: store unavailable",
			slog.String("next_action", "restore store availability before retrying chats drill-down"),
		)
		return parseResponse, nil
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseConversationRows, parseErr := parseS.store.parseListAdminConversations(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: conversation query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats conversations: %v", parseErr)
	}
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats usage events: %v", parseErr)
	}
	parseTemplateRows, parseErr := parseS.store.parseListOnboardingTemplates(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: onboarding-template query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats onboarding templates: %v", parseErr)
	}
	parseMilestoneRows, parseErr := parseS.store.parseListUserActivationMilestones(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: milestone query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats activation milestones: %v", parseErr)
	}
	parseWorkflowRows, parseErr := parseS.store.parseListSavedWorkflows(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: workflow query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats saved workflows: %v", parseErr)
	}
	parsePromptRows, parseErr := parseS.store.parseListPromptLibraryItems(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: prompt-library query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats prompt library items: %v", parseErr)
	}
	parseWeeklyRows, parseErr := parseS.store.parseListWeeklyValueSummaries(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: weekly-value query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats weekly value summaries: %v", parseErr)
	}
	parseAnalyticsRows, parseErr := parseS.store.parseListProductAnalyticsEvents(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsDrilldown: analytics query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chats analytics events: %v", parseErr)
	}

	if !parseScope.isPlatformScope {
		parseConversationRows = parseFilterAdminConversationRowsByScope(parseConversationRows, parseScope.userIDs)
		parseUsageRows = parseFilterAdminUsageRowsByScope(parseUsageRows, parseScope.userIDs)
		parseMilestoneRows = parseFilterActivationMilestoneRowsByScope(parseMilestoneRows, parseScope.userIDs)
		parseWorkflowRows = parseFilterSavedWorkflowRowsByScope(parseWorkflowRows, parseScope.workspaceIDs)
		parsePromptRows = parseFilterPromptLibraryRowsByScope(parsePromptRows, parseScope.workspaceIDs)
		parseWeeklyRows = parseFilterWeeklyValueRowsByScope(parseWeeklyRows, parseScope.workspaceIDs)
		parseAnalyticsRows = parseFilterProductAnalyticsRowsByScope(parseAnalyticsRows, parseScope.workspaceIDs)
	}

	parseResponse.Summary = parseBuildAdminChatsDrilldownSummary(
		parseConversationRows,
		parseUsageRows,
		parseTemplateRows,
		parseMilestoneRows,
		parseWorkflowRows,
		parsePromptRows,
		parseWeeklyRows,
		parseAnalyticsRows,
	)

	parseConversationRows = parseApplyAdminSliceWindow(parseConversationRows, 0, parseLimit)
	parseUsageRows = parseApplyAdminSliceWindow(parseUsageRows, 0, parseLimit)
	parseTemplateRows = parseApplyAdminSliceWindow(parseTemplateRows, 0, parseLimit)
	parseMilestoneRows = parseApplyAdminSliceWindow(parseMilestoneRows, 0, parseLimit)
	parseWorkflowRows = parseApplyAdminSliceWindow(parseWorkflowRows, 0, parseLimit)
	parsePromptRows = parseApplyAdminSliceWindow(parsePromptRows, 0, parseLimit)
	parseWeeklyRows = parseApplyAdminSliceWindow(parseWeeklyRows, 0, parseLimit)
	parseAnalyticsRows = parseApplyAdminSliceWindow(parseAnalyticsRows, 0, parseLimit)

	parseResponse.Conversations = make([]*chatpb.AdminConversationSummary, 0, len(parseConversationRows))
	for _, parseConversationRow := range parseConversationRows {
		parseResponse.Conversations = append(parseResponse.Conversations, parseRedactAdminConversationSummaryByScope(parseScope, parseBuildAdminConversationSummary(parseConversationRow)))
	}
	parseResponse.UsageEvents = make([]*chatpb.AdminUsageEvent, 0, len(parseUsageRows))
	for _, parseUsageRow := range parseUsageRows {
		parseResponse.UsageEvents = append(parseResponse.UsageEvents, parseRedactAdminUsageEventByScope(parseScope, parseBuildAdminUsageEvent(parseUsageRow)))
	}
	parseResponse.OnboardingTemplates = make([]*chatpb.OnboardingTemplateEntry, 0, len(parseTemplateRows))
	for _, parseTemplateRow := range parseTemplateRows {
		parseResponse.OnboardingTemplates = append(parseResponse.OnboardingTemplates, parseBuildOnboardingTemplateEntry(parseTemplateRow))
	}
	parseResponse.ActivationMilestones = make([]*chatpb.UserActivationMilestoneEntry, 0, len(parseMilestoneRows))
	for _, parseMilestoneRow := range parseMilestoneRows {
		parseResponse.ActivationMilestones = append(parseResponse.ActivationMilestones, parseBuildUserActivationMilestoneEntry(parseMilestoneRow))
	}
	parseResponse.SavedWorkflows = make([]*chatpb.SavedWorkflowEntry, 0, len(parseWorkflowRows))
	for _, parseWorkflowRow := range parseWorkflowRows {
		parseResponse.SavedWorkflows = append(parseResponse.SavedWorkflows, parseBuildSavedWorkflowEntry(parseWorkflowRow))
	}
	parseResponse.PromptLibraryItems = make([]*chatpb.PromptLibraryItemEntry, 0, len(parsePromptRows))
	for _, parsePromptRow := range parsePromptRows {
		parseResponse.PromptLibraryItems = append(parseResponse.PromptLibraryItems, parseBuildPromptLibraryItemEntry(parsePromptRow))
	}
	parseResponse.WeeklyValueSummaries = make([]*chatpb.WeeklyValueSummaryEntry, 0, len(parseWeeklyRows))
	for _, parseWeeklyRow := range parseWeeklyRows {
		parseResponse.WeeklyValueSummaries = append(parseResponse.WeeklyValueSummaries, parseBuildWeeklyValueSummaryEntry(parseWeeklyRow))
	}
	parseResponse.ProductAnalyticsEvents = make([]*chatpb.ProductAnalyticsEventEntry, 0, len(parseAnalyticsRows))
	for _, parseAnalyticsRow := range parseAnalyticsRows {
		parseResponse.ProductAnalyticsEvents = append(parseResponse.ProductAnalyticsEvents, parseBuildProductAnalyticsEventEntry(parseAnalyticsRow))
	}
	parseLogger.Info(
		"rpc.GetAdminChatsDrilldown: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int("conversations", len(parseResponse.Conversations)),
		slog.Int("usage_events", len(parseResponse.UsageEvents)),
		slog.Int("onboarding_templates", len(parseResponse.OnboardingTemplates)),
		slog.Int("activation_milestones", len(parseResponse.ActivationMilestones)),
		slog.Int("saved_workflows", len(parseResponse.SavedWorkflows)),
		slog.Int("prompt_library_items", len(parseResponse.PromptLibraryItems)),
		slog.Int("weekly_value_summaries", len(parseResponse.WeeklyValueSummaries)),
		slog.Int("product_analytics_events", len(parseResponse.ProductAnalyticsEvents)),
		slog.Int64("conversation_count", parseResponse.Summary.GetConversationCount()),
	)
	return parseResponse, nil
}
