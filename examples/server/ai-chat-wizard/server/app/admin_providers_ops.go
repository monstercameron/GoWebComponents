package app

import (
	"context"
	"log/slog"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminProviderUsageEntry maps one provider-usage aggregate row into protobuf form.
func parseBuildAdminProviderUsageEntry(parseRow parseAdminDashboardProviderUsageRow) *chatpb.AdminProviderUsage {
	return &chatpb.AdminProviderUsage{
		ProviderId:          parseRow.ProviderID,
		UsageEventCount:     parseRow.UsageEventCount,
		TotalCostUsd:        parseRow.TotalCostUSD,
		PromptTokens:        parseRow.PromptTokens,
		CompletionTokens:    parseRow.CompletionTokens,
		ActiveUsers:         parseRow.ActiveUsers,
		CompletedEventCount: parseRow.CompletedEventCount,
		FailedEventCount:    parseRow.FailedEventCount,
	}
}

// parseBuildAdminModelUsageEntry maps one model-usage aggregate row into protobuf form.
func parseBuildAdminModelUsageEntry(parseRow parseAdminDashboardModelUsageRow) *chatpb.AdminModelUsage {
	return &chatpb.AdminModelUsage{
		ProviderId:          parseRow.ProviderID,
		ModelId:             parseRow.ModelID,
		UsageEventCount:     parseRow.UsageEventCount,
		TotalCostUsd:        parseRow.TotalCostUSD,
		PromptTokens:        parseRow.PromptTokens,
		CompletionTokens:    parseRow.CompletionTokens,
		ActiveUsers:         parseRow.ActiveUsers,
		CompletedEventCount: parseRow.CompletedEventCount,
		FailedEventCount:    parseRow.FailedEventCount,
	}
}

// parseBuildProviderModelCatalogOption maps one model-catalog row into protobuf form.
func parseBuildProviderModelCatalogOption(parseRow modelCatalogRow) *chatpb.ModelOption {
	parseOption := parseRow.parseOption()
	return &chatpb.ModelOption{
		Id:    parseOption.ID,
		Label: parseOption.Label,
		Note:  parseOption.Note,
		Capabilities: &chatpb.ModelCapabilities{
			SupportsThinking: parseOption.Capabilities.SupportsThinking,
			SupportsSpeech:   parseOption.Capabilities.SupportsSpeech,
			ProviderId:       parseOption.Capabilities.ProviderID,
			ProviderLabel:    parseOption.Capabilities.ProviderLabel,
		},
		Pricing: &chatpb.ModelPricing{
			InputCostPerMillionUsd:  parseOption.Pricing.InputPerMillionUSD,
			OutputCostPerMillionUsd: parseOption.Pricing.OutputPerMillionUSD,
			Currency:                parseOption.Pricing.Currency,
		},
	}
}

// parseBuildWorkspaceModelRoutingPolicyEntry maps one workspace model-routing policy row into protobuf form.
func parseBuildWorkspaceModelRoutingPolicyEntry(parseRow parseWorkspaceModelRoutingPolicyRow) *chatpb.WorkspaceModelRoutingPolicyEntry {
	return &chatpb.WorkspaceModelRoutingPolicyEntry{
		Id:                         parseRow.ID,
		WorkspaceId:                parseRow.WorkspaceID,
		PolicyKey:                  parseRow.PolicyKey,
		DefaultModelId:             parseRow.DefaultModelID,
		FallbackModelId:            parseRow.FallbackModelID,
		MaxInputCostPerMillionUsd:  parseRow.MaxInputCostPerMillionUSD,
		MaxOutputCostPerMillionUsd: parseRow.MaxOutputCostPerMillionUSD,
		RequiresApproval:           parseRow.RequiresApproval,
		RulesJson:                  parseRow.RulesJSON,
		UpdatedAt:                  parseRow.UpdatedAt,
	}
}

// parseBuildWorkspaceCostGuardrailEntry maps one workspace cost-guardrail row into protobuf form.
func parseBuildWorkspaceCostGuardrailEntry(parseRow parseWorkspaceCostGuardrailRow) *chatpb.WorkspaceCostGuardrailEntry {
	return &chatpb.WorkspaceCostGuardrailEntry{
		Id:                     parseRow.ID,
		WorkspaceId:            parseRow.WorkspaceID,
		GuardrailKey:           parseRow.GuardrailKey,
		DailyBudgetCents:       parseRow.DailyBudgetCents,
		MonthlyBudgetCents:     parseRow.MonthlyBudgetCents,
		MaxCostPerRequestCents: parseRow.MaxCostPerRequestCents,
		AlertThresholdPercent:  parseRow.AlertThresholdPercent,
		ActionMode:             parseRow.ActionMode,
		UpdatedAt:              parseRow.UpdatedAt,
	}
}

// parseFilterWorkspaceModelRoutingPolicyRowsByWorkspaceID filters model-routing policy rows by one workspace id.
func parseFilterWorkspaceModelRoutingPolicyRowsByWorkspaceID(parseRows []parseWorkspaceModelRoutingPolicyRow, parseWorkspaceID int64) []parseWorkspaceModelRoutingPolicyRow {
	if parseWorkspaceID <= 0 {
		return parseRows
	}
	parseFilteredRows := make([]parseWorkspaceModelRoutingPolicyRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterWorkspaceCostGuardrailRowsByWorkspaceID filters cost-guardrail rows by one workspace id.
func parseFilterWorkspaceCostGuardrailRowsByWorkspaceID(parseRows []parseWorkspaceCostGuardrailRow, parseWorkspaceID int64) []parseWorkspaceCostGuardrailRow {
	if parseWorkspaceID <= 0 {
		return parseRows
	}
	parseFilteredRows := make([]parseWorkspaceCostGuardrailRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminProvidersDrilldownSummary derives one providers drill-down summary from typed provider slices.
func parseBuildAdminProvidersDrilldownSummary(
	parseProviderRows []parseAdminDashboardProviderUsageRow,
	parseModelRows []parseAdminDashboardModelUsageRow,
	parseUsageRows []parseAdminUsageEventRow,
	parseCatalogRows []modelCatalogRow,
	parseRoutingRows []parseWorkspaceModelRoutingPolicyRow,
	parseGuardrailRows []parseWorkspaceCostGuardrailRow,
	parseProviderSnapshotCount int,
) *chatpb.AdminProvidersDrilldownSummary {
	parseSummary := &chatpb.AdminProvidersDrilldownSummary{
		ProviderUsageCount:    int64(len(parseProviderRows)),
		ModelUsageCount:       int64(len(parseModelRows)),
		UsageEventCount:       int64(len(parseUsageRows)),
		ModelCatalogCount:     int64(len(parseCatalogRows)),
		RoutingPolicyCount:    int64(len(parseRoutingRows)),
		CostGuardrailCount:    int64(len(parseGuardrailRows)),
		ProviderSnapshotCount: int64(parseProviderSnapshotCount),
	}
	for _, parseProviderRow := range parseProviderRows {
		parseSummary.CompletedEventCount += parseProviderRow.CompletedEventCount
		parseSummary.FailedEventCount += parseProviderRow.FailedEventCount
		parseSummary.TotalCostUsd += parseProviderRow.TotalCostUSD
	}
	return parseSummary
}

// parseApplyProviderUsageDailyRollups updates provider summary totals from one provider rollup slice.
func parseApplyProviderUsageDailyRollups(parseSummary *chatpb.AdminProvidersDrilldownSummary, parseRollupRows []parseProviderUsageDailyRollupRow) {
	if parseSummary == nil || len(parseRollupRows) == 0 {
		return
	}
	parseSummary.ProviderUsageCount = int64(len(parseRollupRows))
	parseSummary.UsageEventCount = 0
	parseSummary.CompletedEventCount = 0
	parseSummary.FailedEventCount = 0
	parseSummary.TotalCostUsd = 0
	for _, parseRollupRow := range parseRollupRows {
		parseSummary.UsageEventCount += parseRollupRow.UsageEventCount
		parseSummary.CompletedEventCount += parseRollupRow.CompletedEventCount
		parseSummary.FailedEventCount += parseRollupRow.FailedEventCount
		parseSummary.TotalCostUsd += parseRollupRow.TotalCostUSD
	}
}

// GetAdminProvidersDrilldown returns one typed providers drill-down snapshot for dashboard provider surfaces.
func (parseS *chatServer) GetAdminProvidersDrilldown(parseCtx context.Context, parseReq *chatpb.GetAdminProvidersDrilldownRequest) (*chatpb.GetAdminProvidersDrilldownResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminProvidersDrilldown"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.providers")
	if parseErr != nil {
		return nil, parseErr
	}

	var parseLookbackDays int32
	var parseLimit int32
	var parseWorkspaceID int64
	if parseReq != nil {
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
		parseWorkspaceID = parseReq.GetWorkspaceId()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminProvidersDrilldown: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int64("workspace_id", parseWorkspaceID),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"providers",
		"Admin providers slice viewed",
		"{}",
		parseWorkspaceID,
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"slice",
		"providers",
		"Admin providers drill-down viewed",
		"{}",
		parseWorkspaceID,
	)

	parseResponse := &chatpb.GetAdminProvidersDrilldownResponse{
		Summary:           &chatpb.AdminProvidersDrilldownSummary{},
		ProviderSnapshots: parseBuildAdminProviderSnapshots(parseS.providerRegistry),
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminProvidersDrilldown: store unavailable",
			slog.String("next_action", "restore store availability before retrying providers drill-down"),
		)
		parseResponse.Summary.ProviderSnapshotCount = int64(len(parseResponse.GetProviderSnapshots()))
		return parseResponse, nil
	}

	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseWindowLimit := int64(parseLimit)
	if parseWorkspaceID > 0 {
		parseWindowLimit = parseAdminScopedScanLimit
	}
	parseProviderRows, parseErr := parseS.store.parseListAdminDashboardProviderUsage(parseSince, int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminProvidersDrilldown: provider usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list provider usage: %v", parseErr)
	}
	parseModelRows, parseErr := parseS.store.parseListAdminDashboardModelUsage(parseSince, int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminProvidersDrilldown: model usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list model usage: %v", parseErr)
	}
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminProvidersDrilldown: usage-event query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list provider usage events: %v", parseErr)
	}
	parseCatalogRows, parseErr := parseS.store.parseListModelCatalog()
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminProvidersDrilldown: model-catalog query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list provider model catalog: %v", parseErr)
	}
	parseRoutingRows, parseErr := parseS.store.parseListWorkspaceModelRoutingPolicies(parseWindowLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminProvidersDrilldown: workspace routing query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list workspace routing policies: %v", parseErr)
	}
	parseGuardrailRows, parseErr := parseS.store.parseListWorkspaceCostGuardrails(parseWindowLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminProvidersDrilldown: workspace guardrail query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list workspace cost guardrails: %v", parseErr)
	}

	parseRoutingRows = parseFilterWorkspaceModelRoutingPolicyRowsByWorkspaceID(parseRoutingRows, parseWorkspaceID)
	parseGuardrailRows = parseFilterWorkspaceCostGuardrailRowsByWorkspaceID(parseGuardrailRows, parseWorkspaceID)
	parseResponse.Summary = parseBuildAdminProvidersDrilldownSummary(
		parseProviderRows,
		parseModelRows,
		parseUsageRows,
		parseCatalogRows,
		parseRoutingRows,
		parseGuardrailRows,
		len(parseResponse.GetProviderSnapshots()),
	)
	if parseWorkspaceID <= 0 && parseLookbackDays >= 120 {
		if parseErr = parseS.store.parseRefreshProviderUsageDailyRollups(parseSince); parseErr != nil {
			parseLogger.Warn("rpc.GetAdminProvidersDrilldown: provider rollup refresh failed", slog.String("error", parseErr.Error()))
		} else {
			parseRollupRows, parseRollupErr := parseS.store.parseListProviderUsageDailyRollups(parseSince, parseWindowLimit)
			if parseRollupErr != nil {
				parseLogger.Warn("rpc.GetAdminProvidersDrilldown: provider rollup list failed", slog.String("error", parseRollupErr.Error()))
			} else {
				parseApplyProviderUsageDailyRollups(parseResponse.Summary, parseRollupRows)
			}
		}
	}

	parseProviderRows = parseApplyAdminSliceWindow(parseProviderRows, 0, parseLimit)
	parseModelRows = parseApplyAdminSliceWindow(parseModelRows, 0, parseLimit)
	parseUsageRows = parseApplyAdminSliceWindow(parseUsageRows, 0, parseLimit)
	parseCatalogRows = parseApplyAdminSliceWindow(parseCatalogRows, 0, parseLimit)
	parseRoutingRows = parseApplyAdminSliceWindow(parseRoutingRows, 0, parseLimit)
	parseGuardrailRows = parseApplyAdminSliceWindow(parseGuardrailRows, 0, parseLimit)
	parseResponse.ProviderSnapshots = parseApplyAdminSliceWindow(parseResponse.ProviderSnapshots, 0, parseLimit)

	parseResponse.Providers = make([]*chatpb.AdminProviderUsage, 0, len(parseProviderRows))
	for _, parseProviderRow := range parseProviderRows {
		parseResponse.Providers = append(parseResponse.Providers, parseBuildAdminProviderUsageEntry(parseProviderRow))
	}
	parseResponse.Models = make([]*chatpb.AdminModelUsage, 0, len(parseModelRows))
	for _, parseModelRow := range parseModelRows {
		parseResponse.Models = append(parseResponse.Models, parseBuildAdminModelUsageEntry(parseModelRow))
	}
	parseResponse.UsageEvents = make([]*chatpb.AdminUsageEvent, 0, len(parseUsageRows))
	for _, parseUsageRow := range parseUsageRows {
		parseResponse.UsageEvents = append(parseResponse.UsageEvents, parseRedactAdminUsageEventByScope(parseScope, parseBuildAdminUsageEvent(parseUsageRow)))
	}
	parseResponse.ModelCatalog = make([]*chatpb.ModelOption, 0, len(parseCatalogRows))
	for _, parseCatalogRow := range parseCatalogRows {
		parseResponse.ModelCatalog = append(parseResponse.ModelCatalog, parseBuildProviderModelCatalogOption(parseCatalogRow))
	}
	parseResponse.WorkspaceModelRoutingPolicies = make([]*chatpb.WorkspaceModelRoutingPolicyEntry, 0, len(parseRoutingRows))
	for _, parseRoutingRow := range parseRoutingRows {
		parseResponse.WorkspaceModelRoutingPolicies = append(parseResponse.WorkspaceModelRoutingPolicies, parseBuildWorkspaceModelRoutingPolicyEntry(parseRoutingRow))
	}
	parseResponse.WorkspaceCostGuardrails = make([]*chatpb.WorkspaceCostGuardrailEntry, 0, len(parseGuardrailRows))
	for _, parseGuardrailRow := range parseGuardrailRows {
		parseResponse.WorkspaceCostGuardrails = append(parseResponse.WorkspaceCostGuardrails, parseBuildWorkspaceCostGuardrailEntry(parseGuardrailRow))
	}

	parseLogger.Info(
		"rpc.GetAdminProvidersDrilldown: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int("providers", len(parseResponse.Providers)),
		slog.Int("models", len(parseResponse.Models)),
		slog.Int("usage_events", len(parseResponse.UsageEvents)),
		slog.Int("model_catalog", len(parseResponse.ModelCatalog)),
		slog.Int("routing_policies", len(parseResponse.WorkspaceModelRoutingPolicies)),
		slog.Int("cost_guardrails", len(parseResponse.WorkspaceCostGuardrails)),
		slog.Int("provider_snapshots", len(parseResponse.ProviderSnapshots)),
	)
	return parseResponse, nil
}
