package app

import (
	"context"
	"log/slog"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildBillingCustomerEntry maps one billing-customer row into protobuf form.
func parseBuildBillingCustomerEntry(parseRow parseBillingCustomerRow) *chatpb.BillingCustomerEntry {
	return &chatpb.BillingCustomerEntry{
		Id:                   parseRow.ID,
		UserId:               parseRow.UserID,
		ProviderId:           parseRow.ProviderID,
		ProviderCustomerId:   parseRow.ProviderCustomerID,
		Email:                parseRow.BillingEmail,
		DisplayName:          parseRow.BillingName,
		Currency:             parseRow.DefaultCurrency,
		TaxExemptStatus:      parseRow.TaxExemptStatus,
		ExternalMetadataJson: parseRow.ExternalMetadataJSON,
		CreatedAt:            parseRow.CreatedAt,
		UpdatedAt:            parseRow.UpdatedAt,
	}
}

// parseBuildBillingSubscriptionEntry maps one billing-subscription row into protobuf form.
func parseBuildBillingSubscriptionEntry(parseRow parseBillingSubscriptionRow) *chatpb.BillingSubscriptionEntry {
	return &chatpb.BillingSubscriptionEntry{
		Id:                     parseRow.ID,
		CustomerId:             parseRow.CustomerID,
		ProviderId:             parseRow.ProviderID,
		ProviderSubscriptionId: parseRow.ProviderSubscriptionID,
		PlanCode:               parseRow.PlanCode,
		PriceCode:              parseRow.PriceCode,
		Status:                 parseRow.Status,
		BillingInterval:        parseRow.BillingInterval,
		Quantity:               parseRow.Quantity,
		CurrentPeriodStart:     parseRow.CurrentPeriodStart,
		CurrentPeriodEnd:       parseRow.CurrentPeriodEnd,
		IsCancelAtPeriodEnd:    parseRow.IsCancelAtPeriodEnd,
		CanceledAt:             parseRow.CanceledAt,
		TrialEndsAt:            parseRow.TrialEndsAt,
		AccessExpiresAt:        parseRow.AccessExpiresAt,
		CreatedAt:              parseRow.CreatedAt,
		UpdatedAt:              parseRow.UpdatedAt,
	}
}

// parseBuildBillingInvoiceEntry maps one billing-invoice row into protobuf form.
func parseBuildBillingInvoiceEntry(parseRow parseBillingInvoiceRow) *chatpb.BillingInvoiceEntry {
	return &chatpb.BillingInvoiceEntry{
		Id:                   parseRow.ID,
		CustomerId:           parseRow.CustomerID,
		SubscriptionId:       parseRow.SubscriptionID,
		ProviderId:           parseRow.ProviderID,
		ProviderInvoiceId:    parseRow.ProviderInvoiceID,
		Status:               parseRow.Status,
		Currency:             parseRow.Currency,
		SubtotalCents:        parseRow.SubtotalCents,
		TaxCents:             parseRow.TaxCents,
		DiscountCents:        parseRow.DiscountCents,
		TotalCents:           parseRow.TotalCents,
		AmountDueCents:       parseRow.AmountDueCents,
		AmountPaidCents:      parseRow.AmountPaidCents,
		PeriodStart:          parseRow.PeriodStart,
		PeriodEnd:            parseRow.PeriodEnd,
		DueAt:                parseRow.DueAt,
		PaidAt:               parseRow.PaidAt,
		HostedInvoiceUrl:     parseRow.HostedInvoiceURL,
		ExternalMetadataJson: parseRow.ExternalMetadataJSON,
		CreatedAt:            parseRow.CreatedAt,
		UpdatedAt:            parseRow.UpdatedAt,
	}
}

// parseBuildBillingInvoiceLineItemEntry maps one invoice-line-item row into protobuf form.
func parseBuildBillingInvoiceLineItemEntry(parseRow parseBillingInvoiceLineItemRow) *chatpb.BillingInvoiceLineItemEntry {
	return &chatpb.BillingInvoiceLineItemEntry{
		Id:              parseRow.ID,
		InvoiceId:       parseRow.InvoiceID,
		UsageEventId:    strings.TrimSpace(parseRow.UsageEventID),
		LineType:        parseRow.LineType,
		Description:     parseRow.Description,
		Quantity:        parseRow.Quantity,
		UnitAmountCents: parseRow.UnitAmountCents,
		AmountCents:     parseRow.AmountCents,
		Currency:        parseRow.Currency,
		PeriodStart:     parseRow.PeriodStart,
		PeriodEnd:       parseRow.PeriodEnd,
		CreatedAt:       parseRow.CreatedAt,
	}
}

// parseBuildProductAnalyticsEventEntry maps one product-analytics row into protobuf form.
func parseBuildProductAnalyticsEventEntry(parseRow parseProductAnalyticsEventRow) *chatpb.ProductAnalyticsEventEntry {
	return &chatpb.ProductAnalyticsEventEntry{
		Id:             parseRow.ID,
		WorkspaceId:    parseRow.WorkspaceID,
		UserId:         parseRow.UserID,
		SessionKey:     parseRow.SessionKey,
		EventName:      parseRow.EventName,
		FunnelKey:      parseRow.FunnelKey,
		StepKey:        parseRow.StepKey,
		ExperimentKey:  parseRow.ExperimentKey,
		VariantKey:     parseRow.VariantKey,
		EventPropsJson: parseRow.EventPropsJSON,
		CreatedAt:      parseRow.CreatedAt,
	}
}

// parseBuildSubscriptionChurnFeedbackEntry maps one churn-feedback row into protobuf form.
func parseBuildSubscriptionChurnFeedbackEntry(parseRow parseSubscriptionChurnFeedbackRow) *chatpb.SubscriptionChurnFeedbackEntry {
	return &chatpb.SubscriptionChurnFeedbackEntry{
		Id:               parseRow.ID,
		CustomerId:       parseRow.CustomerID,
		SubscriptionId:   parseRow.SubscriptionID,
		WorkspaceId:      parseRow.WorkspaceID,
		ReasonKey:        parseRow.ReasonKey,
		Detail:           parseRow.Detail,
		RecoveryOfferKey: parseRow.RecoveryOfferKey,
		CreatedAt:        parseRow.CreatedAt,
	}
}

// GetAdminBusinessDrilldown returns one typed business drill-down payload for one admin-target user.
func (parseS *chatServer) GetAdminBusinessDrilldown(parseCtx context.Context, parseReq *chatpb.GetAdminBusinessDrilldownRequest) (*chatpb.GetAdminBusinessDrilldownResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminBusinessDrilldown"))
	var parseUserID int64
	var parseLimit int32
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseLimit = parseReq.GetLimit()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.business.drilldown")
	if parseErr != nil {
		return nil, parseErr
	}

	parseResponse := &chatpb.GetAdminBusinessDrilldownResponse{
		Subscriptions:          make([]*chatpb.BillingSubscriptionEntry, 0),
		Invoices:               make([]*chatpb.BillingInvoiceEntry, 0),
		InvoiceLineItems:       make([]*chatpb.BillingInvoiceLineItemEntry, 0),
		AccessOverrides:        make([]*chatpb.AdminBillingAccessOverrideEntry, 0),
		BillingEvents:          make([]*chatpb.AdminBillingEventEntry, 0),
		UsageEvents:            make([]*chatpb.AdminUsageEvent, 0),
		ProductAnalyticsEvents: make([]*chatpb.ProductAnalyticsEventEntry, 0),
		ChurnFeedback:          make([]*chatpb.SubscriptionChurnFeedbackEntry, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetAdminBusinessDrilldown: store unavailable")
		return parseResponse, nil
	}

	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminBusinessDrilldown: drill-down fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int64("target_user_id", parseUserID),
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"user",
		"business",
		"Business drill-down viewed",
		"{}",
		parseUserID,
	)

	parseUserRow, hasParseUserRow, parseErr := parseS.store.parseGetAdminUserSummaryByUserID(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get admin user summary: %v", parseErr)
	}
	if !hasParseUserRow {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	parseResponse.User = parseBuildAdminUserSummary(parseUserRow)

	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEventsByUser(parseUserID, "", int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list usage events by user: %v", parseErr)
	}
	for _, parseUsageRow := range parseUsageRows {
		parseResponse.UsageEvents = append(parseResponse.UsageEvents, parseBuildAdminUsageEvent(parseUsageRow))
	}

	parseAnalyticsRows, parseErr := parseS.store.parseListProductAnalyticsEventsByUser(parseUserID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list product analytics by user: %v", parseErr)
	}
	for _, parseAnalyticsRow := range parseAnalyticsRows {
		parseResponse.ProductAnalyticsEvents = append(parseResponse.ProductAnalyticsEvents, parseBuildProductAnalyticsEventEntry(parseAnalyticsRow))
	}

	parseCustomerRow, hasParseCustomerRow, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get billing customer by user: %v", parseErr)
	}
	if !hasParseCustomerRow {
		parseLogger.Info("rpc.GetAdminBusinessDrilldown: no billing customer for target user", slog.Int64("target_user_id", parseUserID))
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

	parseAccessOverrideRows, parseErr := parseS.store.parseListBillingAccessOverridesByCustomer(parseCustomerRow.ID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing access overrides by customer: %v", parseErr)
	}
	for _, parseAccessOverrideRow := range parseAccessOverrideRows {
		parseResponse.AccessOverrides = append(parseResponse.AccessOverrides, parseBuildAdminBillingAccessOverrideEntry(parseAccessOverrideRow))
	}

	parseBillingEventRows, parseErr := parseS.store.parseListBillingEventsByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing events by customer: %v", parseErr)
	}
	for _, parseBillingEventRow := range parseBillingEventRows {
		parseResponse.BillingEvents = append(parseResponse.BillingEvents, parseBuildAdminBillingEventEntry(parseBillingEventRow))
	}

	parseChurnRows, parseErr := parseS.store.parseListSubscriptionChurnFeedbackByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list churn feedback by customer: %v", parseErr)
	}
	for _, parseChurnRow := range parseChurnRows {
		parseResponse.ChurnFeedback = append(parseResponse.ChurnFeedback, parseBuildSubscriptionChurnFeedbackEntry(parseChurnRow))
	}

	parseLogger.Info(
		"rpc.GetAdminBusinessDrilldown: complete",
		slog.Int64("target_user_id", parseUserID),
		slog.Int("subscriptions", len(parseResponse.Subscriptions)),
		slog.Int("invoices", len(parseResponse.Invoices)),
		slog.Int("invoice_line_items", len(parseResponse.InvoiceLineItems)),
		slog.Int("access_overrides", len(parseResponse.AccessOverrides)),
		slog.Int("billing_events", len(parseResponse.BillingEvents)),
		slog.Int("usage_events", len(parseResponse.UsageEvents)),
		slog.Int("product_analytics_events", len(parseResponse.ProductAnalyticsEvents)),
		slog.Int("churn_feedback", len(parseResponse.ChurnFeedback)),
	)
	return parseResponse, nil
}
