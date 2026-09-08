package app

import (
	"context"
	"log/slog"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminUserUsageEntry maps one top-account usage row into protobuf form.
func parseBuildAdminUserUsageEntry(parseRow parseAdminDashboardUserUsageRow) *chatpb.AdminUserUsage {
	return &chatpb.AdminUserUsage{
		UserId:            parseRow.UserID,
		Email:             parseRow.Email,
		DisplayName:       parseRow.DisplayName,
		ConversationCount: parseRow.ConversationCount,
		MessageCount:      parseRow.MessageCount,
		UsageEventCount:   parseRow.UsageEventCount,
		TotalCostUsd:      parseRow.TotalCostUSD,
		PromptTokens:      parseRow.PromptTokens,
		CompletionTokens:  parseRow.CompletionTokens,
		LastSeenAt:        parseRow.LastSeenAt,
	}
}

// parseHasFirstChatFunnelEvent reports whether one analytics row belongs to first-chat funnel investigations.
func parseHasFirstChatFunnelEvent(parseRow parseProductAnalyticsEventRow) bool {
	return strings.EqualFold(strings.TrimSpace(parseRow.FunnelKey), "visit_to_first_chat")
}

// parseFilterProductAnalyticsRowsBySince filters analytics rows to one lookback boundary.
func parseFilterProductAnalyticsRowsBySince(parseRows []parseProductAnalyticsEventRow, parseSince string) []parseProductAnalyticsEventRow {
	parseSince = strings.TrimSpace(parseSince)
	if parseSince == "" {
		return parseRows
	}
	parseFilteredRows := make([]parseProductAnalyticsEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if strings.TrimSpace(parseRow.CreatedAt) < parseSince {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterProductAnalyticsRowsByFirstChatFunnel filters analytics rows to first-chat funnel events.
func parseFilterProductAnalyticsRowsByFirstChatFunnel(parseRows []parseProductAnalyticsEventRow) []parseProductAnalyticsEventRow {
	parseFilteredRows := make([]parseProductAnalyticsEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasFirstChatFunnelEvent(parseRow) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminBusinessQueueSummary derives one business queue summary from failed-payment, dunning, top-account, and funnel slices.
func parseBuildAdminBusinessQueueSummary(
	parseFailedPaymentRows []parseBillingEventRow,
	parseDunningRows []parseBillingDunningEventRow,
	parseTopAccountRows []parseAdminDashboardUserUsageRow,
	parseFunnelRows []parseProductAnalyticsEventRow,
) *chatpb.AdminBusinessQueueSummary {
	parseSummary := &chatpb.AdminBusinessQueueSummary{
		FailedPaymentEventCount:   int64(len(parseFailedPaymentRows)),
		DunningEventCount:         int64(len(parseDunningRows)),
		TopAccountCount:           int64(len(parseTopAccountRows)),
		FirstChatFunnelEventCount: int64(len(parseFunnelRows)),
	}
	for _, parseDunningRow := range parseDunningRows {
		if strings.EqualFold(strings.TrimSpace(parseDunningRow.Status), "resolved") {
			continue
		}
		parseSummary.OpenDunningEventCount++
	}
	return parseSummary
}

// GetAdminBusinessQueue returns typed failed-payment, dunning, top-account, and funnel queue slices.
func (parseS *chatServer) GetAdminBusinessQueue(parseCtx context.Context, parseReq *chatpb.GetAdminBusinessQueueRequest) (*chatpb.GetAdminBusinessQueueResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminBusinessQueue"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.business.queue")
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
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"business-queue",
		"Admin business queue viewed",
		"{}",
		0,
	)
	parseResponse := &chatpb.GetAdminBusinessQueueResponse{
		Summary: &chatpb.AdminBusinessQueueSummary{},
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetAdminBusinessQueue: store unavailable")
		return parseResponse, nil
	}

	parseFailedPaymentRows, parseErr := parseS.store.parseListAdminFailedPaymentBillingEvents(parseSince, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list business failed-payment events: %v", parseErr)
	}
	parseDunningRows, parseErr := parseS.store.parseListAdminBillingDunningTimeline(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list business dunning timeline: %v", parseErr)
	}
	parseTopAccountRows, parseErr := parseS.store.parseListAdminDashboardUserUsage(parseSince, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list business top accounts: %v", parseErr)
	}
	parseAnalyticsRows, parseErr := parseS.store.parseListProductAnalyticsEvents(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list business analytics events: %v", parseErr)
	}
	parseAnalyticsRows = parseFilterProductAnalyticsRowsBySince(parseAnalyticsRows, parseSince)
	parseAnalyticsRows = parseFilterProductAnalyticsRowsByFirstChatFunnel(parseAnalyticsRows)
	parseResponse.Summary = parseBuildAdminBusinessQueueSummary(parseFailedPaymentRows, parseDunningRows, parseTopAccountRows, parseAnalyticsRows)

	parseFailedPaymentRows = parseApplyAdminSliceWindow(parseFailedPaymentRows, 0, parseLimit)
	parseDunningRows = parseApplyAdminSliceWindow(parseDunningRows, 0, parseLimit)
	parseTopAccountRows = parseApplyAdminSliceWindow(parseTopAccountRows, 0, parseLimit)
	parseAnalyticsRows = parseApplyAdminSliceWindow(parseAnalyticsRows, 0, parseLimit)

	parseResponse.FailedPaymentEvents = make([]*chatpb.AdminBillingEventEntry, 0, len(parseFailedPaymentRows))
	for _, parseFailedPaymentRow := range parseFailedPaymentRows {
		parseResponse.FailedPaymentEvents = append(parseResponse.FailedPaymentEvents, parseBuildAdminBillingEventEntry(parseFailedPaymentRow))
	}
	parseResponse.DunningEvents = make([]*chatpb.AdminBillingDunningEventEntry, 0, len(parseDunningRows))
	for _, parseDunningRow := range parseDunningRows {
		parseResponse.DunningEvents = append(parseResponse.DunningEvents, parseBuildAdminBillingDunningEventEntry(parseDunningRow))
	}
	parseResponse.TopAccounts = make([]*chatpb.AdminUserUsage, 0, len(parseTopAccountRows))
	for _, parseTopAccountRow := range parseTopAccountRows {
		parseResponse.TopAccounts = append(parseResponse.TopAccounts, parseBuildAdminUserUsageEntry(parseTopAccountRow))
	}
	parseResponse.FirstChatFunnelEvents = make([]*chatpb.ProductAnalyticsEventEntry, 0, len(parseAnalyticsRows))
	for _, parseAnalyticsRow := range parseAnalyticsRows {
		parseResponse.FirstChatFunnelEvents = append(parseResponse.FirstChatFunnelEvents, parseBuildProductAnalyticsEventEntry(parseAnalyticsRow))
	}
	return parseResponse, nil
}

// GetAdminBusinessAccountDetail returns typed business account drill-down rows for one target user.
func (parseS *chatServer) GetAdminBusinessAccountDetail(parseCtx context.Context, parseReq *chatpb.GetAdminBusinessAccountDetailRequest) (*chatpb.GetAdminBusinessAccountDetailResponse, error) {
	var parseUserID int64
	var parseLookbackDays int32
	var parseLimit int32
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.business.account_detail")
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.GetAdminBusinessAccountDetailResponse{
		Subscriptions:         make([]*chatpb.BillingSubscriptionEntry, 0),
		Invoices:              make([]*chatpb.BillingInvoiceEntry, 0),
		InvoiceLineItems:      make([]*chatpb.BillingInvoiceLineItemEntry, 0),
		BillingEvents:         make([]*chatpb.AdminBillingEventEntry, 0),
		DunningEvents:         make([]*chatpb.AdminBillingDunningEventEntry, 0),
		FirstChatFunnelEvents: make([]*chatpb.ProductAnalyticsEventEntry, 0),
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"user",
		"business-account-detail",
		"Admin business account detail viewed",
		"{}",
		parseUserID,
	)
	if parseS.store == nil {
		return parseResponse, nil
	}

	parseUserRow, hasParseUserRow, parseErr := parseS.store.parseGetAdminUserSummaryByUserID(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get admin user summary: %v", parseErr)
	}
	if !hasParseUserRow {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	parseResponse.User = parseBuildAdminUserSummary(parseUserRow)

	parseCustomerRow, hasParseCustomerRow, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get billing customer by user: %v", parseErr)
	}
	if !hasParseCustomerRow {
		return parseResponse, nil
	}
	parseResponse.Customer = parseBuildBillingCustomerEntry(parseCustomerRow)

	parseSubscriptionRows, parseErr := parseS.store.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing subscriptions by customer: %v", parseErr)
	}
	for _, parseSubscriptionRow := range parseSubscriptionRows {
		parseResponse.Subscriptions = append(parseResponse.Subscriptions, parseBuildBillingSubscriptionEntry(parseSubscriptionRow))
	}
	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing invoices by customer: %v", parseErr)
	}
	for _, parseInvoiceRow := range parseInvoiceRows {
		parseResponse.Invoices = append(parseResponse.Invoices, parseBuildBillingInvoiceEntry(parseInvoiceRow))
		parseLineItemRows, parseErr2 := parseS.store.parseListBillingInvoiceLineItems(parseInvoiceRow.ID, parseCustomerRow.ID)
		if parseErr2 != nil {
			return nil, status.Errorf(codes.Internal, "list billing invoice line items: %v", parseErr2)
		}
		for _, parseLineItemRow := range parseLineItemRows {
			parseResponse.InvoiceLineItems = append(parseResponse.InvoiceLineItems, parseBuildBillingInvoiceLineItemEntry(parseLineItemRow))
		}
	}
	parseBillingEventRows, parseErr := parseS.store.parseListBillingEventsByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing events by customer: %v", parseErr)
	}
	for _, parseBillingEventRow := range parseBillingEventRows {
		parseResponse.BillingEvents = append(parseResponse.BillingEvents, parseBuildAdminBillingEventEntry(parseBillingEventRow))
	}
	parseDunningRows, parseErr := parseS.store.parseListBillingDunningEventsByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing dunning events by customer: %v", parseErr)
	}
	for _, parseDunningRow := range parseDunningRows {
		parseResponse.DunningEvents = append(parseResponse.DunningEvents, parseBuildAdminBillingDunningEventEntry(parseDunningRow))
	}
	parseAnalyticsRows, parseErr := parseS.store.parseListProductAnalyticsEventsByUser(parseUserID, parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list product analytics by user: %v", parseErr)
	}
	parseAnalyticsRows = parseFilterProductAnalyticsRowsBySince(parseAnalyticsRows, parseSince)
	parseAnalyticsRows = parseFilterProductAnalyticsRowsByFirstChatFunnel(parseAnalyticsRows)
	parseAnalyticsRows = parseApplyAdminSliceWindow(parseAnalyticsRows, 0, parseLimit)
	for _, parseAnalyticsRow := range parseAnalyticsRows {
		parseResponse.FirstChatFunnelEvents = append(parseResponse.FirstChatFunnelEvents, parseBuildProductAnalyticsEventEntry(parseAnalyticsRow))
	}
	return parseResponse, nil
}
