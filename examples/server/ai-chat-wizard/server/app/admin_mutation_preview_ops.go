package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminBusinessMutationImpactSummary derives one business mutation-impact summary from preview counts.
func parseBuildAdminBusinessMutationImpactSummary(parsePreview parseAdminBusinessMutationPreviewRow) string {
	parseSummaryParts := make([]string, 0, 3)
	if parsePreview.OpenInvoiceCount > 0 || parsePreview.OpenDunningEventCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf("Billing collections currently involve %d open invoices and %d open dunning events.", parsePreview.OpenInvoiceCount, parsePreview.OpenDunningEventCount),
		)
	}
	if parsePreview.RecentFailedPaymentEventCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf("Recent failed-payment activity includes %d events in the selected lookback window.", parsePreview.RecentFailedPaymentEventCount),
		)
	}
	if parsePreview.RecentUsageEventCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf("Recent usage spans %d metered events across affected accounts.", parsePreview.RecentUsageEventCount),
		)
	}
	if len(parseSummaryParts) == 0 {
		return "No elevated billing-side blast-radius signals were detected in the selected lookback window."
	}
	return strings.Join(parseSummaryParts, " ")
}

// parseBuildAdminProviderMutationImpactSummary derives one provider mutation-impact summary from preview counts.
func parseBuildAdminProviderMutationImpactSummary(parsePreview parseAdminProviderMutationPreviewRow) string {
	parseSummaryParts := make([]string, 0, 3)
	if parsePreview.WorkspaceCount > 0 || parsePreview.RoutingPolicyCount > 0 || parsePreview.GuardrailCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf(
				"Provider controls currently affect %d workspaces, %d routing policies, and %d cost guardrails.",
				parsePreview.WorkspaceCount,
				parsePreview.RoutingPolicyCount,
				parsePreview.GuardrailCount,
			),
		)
	}
	if parsePreview.RecentFailedUsageEventCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf("Recent provider/model failures include %d failed usage events in scope.", parsePreview.RecentFailedUsageEventCount),
		)
	}
	if parsePreview.RecentUsageEventCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf("Recent provider/model throughput includes %d usage events in the selected lookback window.", parsePreview.RecentUsageEventCount),
		)
	}
	if len(parseSummaryParts) == 0 {
		return "No elevated provider-side blast-radius signals were detected in the selected lookback window."
	}
	return strings.Join(parseSummaryParts, " ")
}

// parseBuildAdminOpsMutationImpactSummary derives one ops mutation-impact summary from preview counts.
func parseBuildAdminOpsMutationImpactSummary(parsePreview parseAdminOpsMutationPreviewRow) string {
	parseSummaryParts := make([]string, 0, 3)
	if parsePreview.FailedBackgroundJobCount > 0 || parsePreview.FailedNotificationCount > 0 || parsePreview.FailedWebhookDeliveryCount > 0 {
		parseSummaryParts = append(
			parseSummaryParts,
			fmt.Sprintf(
				"Current failed queues include %d background jobs, %d notifications, and %d webhook deliveries.",
				parsePreview.FailedBackgroundJobCount,
				parsePreview.FailedNotificationCount,
				parsePreview.FailedWebhookDeliveryCount,
			),
		)
	}
	if parsePreview.OpenIncidentCount > 0 {
		parseSummaryParts = append(parseSummaryParts, fmt.Sprintf("There are %d open incidents that may be affected by control-plane changes.", parsePreview.OpenIncidentCount))
	}
	if parsePreview.RecentAdminActionCount > 0 {
		parseSummaryParts = append(parseSummaryParts, fmt.Sprintf("Recent ops interventions include %d admin actions in the selected lookback window.", parsePreview.RecentAdminActionCount))
	}
	if len(parseSummaryParts) == 0 {
		return "No elevated ops-side blast-radius signals were detected in the selected lookback window."
	}
	return strings.Join(parseSummaryParts, " ")
}

// parseBuildAdminBusinessMutationPreviewEntry maps one business mutation-preview row into protobuf form.
func parseBuildAdminBusinessMutationPreviewEntry(parseRow parseAdminBusinessMutationPreviewRow) *chatpb.AdminBusinessMutationPreview {
	return &chatpb.AdminBusinessMutationPreview{
		AffectedCustomerCount:         parseRow.CustomerCount,
		AffectedSubscriptionCount:     parseRow.SubscriptionCount,
		OpenInvoiceCount:              parseRow.OpenInvoiceCount,
		OpenDunningEventCount:         parseRow.OpenDunningEventCount,
		RecentFailedPaymentEventCount: parseRow.RecentFailedPaymentEventCount,
		RecentUsageEventCount:         parseRow.RecentUsageEventCount,
		RecentUsageCostUsd:            parseRow.RecentUsageCostUSD,
		LikelyDownstreamImpact:        parseBuildAdminBusinessMutationImpactSummary(parseRow),
	}
}

// parseBuildAdminProviderMutationPreviewEntry maps one provider mutation-preview row into protobuf form.
func parseBuildAdminProviderMutationPreviewEntry(parseRow parseAdminProviderMutationPreviewRow) *chatpb.AdminProviderMutationPreview {
	return &chatpb.AdminProviderMutationPreview{
		AffectedWorkspaceCount:      parseRow.WorkspaceCount,
		RoutingPolicyCount:          parseRow.RoutingPolicyCount,
		CostGuardrailCount:          parseRow.GuardrailCount,
		RecentUsageEventCount:       parseRow.RecentUsageEventCount,
		RecentFailedUsageEventCount: parseRow.RecentFailedUsageEventCount,
		RecentUsageCostUsd:          parseRow.RecentUsageCostUSD,
		LikelyDownstreamImpact:      parseBuildAdminProviderMutationImpactSummary(parseRow),
	}
}

// parseBuildAdminOpsMutationPreviewEntry maps one ops mutation-preview row into protobuf form.
func parseBuildAdminOpsMutationPreviewEntry(parseRow parseAdminOpsMutationPreviewRow) *chatpb.AdminOpsMutationPreview {
	return &chatpb.AdminOpsMutationPreview{
		FailedBackgroundJobCount:   parseRow.FailedBackgroundJobCount,
		FailedNotificationCount:    parseRow.FailedNotificationCount,
		FailedWebhookDeliveryCount: parseRow.FailedWebhookDeliveryCount,
		OpenIncidentCount:          parseRow.OpenIncidentCount,
		RecentAdminActionCount:     parseRow.RecentAdminActionCount,
		LikelyDownstreamImpact:     parseBuildAdminOpsMutationImpactSummary(parseRow),
	}
}

// GetAdminBusinessMutationPreview returns one typed business blast-radius preview for high-risk mutation planning.
func (parseS *chatServer) GetAdminBusinessMutationPreview(parseCtx context.Context, parseReq *chatpb.GetAdminBusinessMutationPreviewRequest) (*chatpb.GetAdminBusinessMutationPreviewResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminBusinessMutationPreview"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.business.mutation_preview")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseUserID int64
	var parseLookbackDays int32
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseLookbackDays = parseReq.GetLookbackDays()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"slice",
		"business-mutation-preview",
		"Admin business mutation preview viewed",
		"{}",
		parseUserID,
	)
	parseResponse := &chatpb.GetAdminBusinessMutationPreviewResponse{
		Preview: &chatpb.AdminBusinessMutationPreview{},
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetAdminBusinessMutationPreview: store unavailable")
		return parseResponse, nil
	}
	parsePreviewRow, parseErr := parseS.store.parseGetAdminBusinessMutationPreview(parseUserID, parseSince)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get business mutation preview: %v", parseErr)
	}
	parseResponse.Preview = parseBuildAdminBusinessMutationPreviewEntry(parsePreviewRow)
	return parseResponse, nil
}

// GetAdminProviderMutationPreview returns one typed provider blast-radius preview for high-risk mutation planning.
func (parseS *chatServer) GetAdminProviderMutationPreview(parseCtx context.Context, parseReq *chatpb.GetAdminProviderMutationPreviewRequest) (*chatpb.GetAdminProviderMutationPreviewResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminProviderMutationPreview"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.providers.mutation_preview")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseProviderID string
	var parseModelID string
	var parseLookbackDays int32
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseProviderID = strings.TrimSpace(parseReq.GetProviderId())
		parseModelID = strings.TrimSpace(parseReq.GetModelId())
		parseLookbackDays = parseReq.GetLookbackDays()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"slice",
		"provider-mutation-preview",
		"Admin provider mutation preview viewed",
		"{}",
		parseWorkspaceID,
	)
	parseResponse := &chatpb.GetAdminProviderMutationPreviewResponse{
		Preview: &chatpb.AdminProviderMutationPreview{},
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetAdminProviderMutationPreview: store unavailable")
		return parseResponse, nil
	}
	parsePreviewRow, parseErr := parseS.store.parseGetAdminProviderMutationPreview(parseWorkspaceID, parseProviderID, parseModelID, parseSince)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get provider mutation preview: %v", parseErr)
	}
	parseResponse.Preview = parseBuildAdminProviderMutationPreviewEntry(parsePreviewRow)
	return parseResponse, nil
}

// GetAdminOpsMutationPreview returns one typed ops blast-radius preview for high-risk mutation planning.
func (parseS *chatServer) GetAdminOpsMutationPreview(parseCtx context.Context, parseReq *chatpb.GetAdminOpsMutationPreviewRequest) (*chatpb.GetAdminOpsMutationPreviewResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminOpsMutationPreview"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.ops.mutation_preview")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseLookbackDays int32
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseLookbackDays = parseReq.GetLookbackDays()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"slice",
		"ops-mutation-preview",
		"Admin ops mutation preview viewed",
		"{}",
		parseWorkspaceID,
	)
	parseResponse := &chatpb.GetAdminOpsMutationPreviewResponse{
		Preview: &chatpb.AdminOpsMutationPreview{},
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetAdminOpsMutationPreview: store unavailable")
		return parseResponse, nil
	}
	parsePreviewRow, parseErr := parseS.store.parseGetAdminOpsMutationPreview(parseWorkspaceID, parseSince)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get ops mutation preview: %v", parseErr)
	}
	parseResponse.Preview = parseBuildAdminOpsMutationPreviewEntry(parsePreviewRow)
	return parseResponse, nil
}
