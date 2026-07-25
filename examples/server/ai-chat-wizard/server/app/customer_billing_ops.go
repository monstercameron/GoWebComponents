package app

import (
	"context"
	"log/slog"
	"math"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseClampCustomerBillingLimit bounds one customer-billing request limit to a safe positive range.
func parseClampCustomerBillingLimit(parseRequested int32, parseDefault int64) int64 {
	if parseDefault <= 0 {
		parseDefault = 25
	}
	if parseRequested <= 0 {
		return parseDefault
	}
	if parseRequested > 250 {
		return 250
	}
	return int64(parseRequested)
}

// parseBuildCustomerBillingUsageSummary aggregates one usage-event slice into one typed customer usage summary.
func parseBuildCustomerBillingUsageSummary(parseRows []parseUsageEventRow) *chatpb.CustomerBillingUsageSummary {
	parseSummary := &chatpb.CustomerBillingUsageSummary{EventCount: int64(len(parseRows))}
	for _, parseRow := range parseRows {
		parseSummary.PromptTokens += parseRow.PromptTokens
		parseSummary.CompletionTokens += parseRow.CompletionTokens
		parseSummary.UsageCostUsd += parseRow.TotalCostUSD

		parseCreatedAt := strings.TrimSpace(parseRow.CreatedAt)
		if parseCreatedAt == "" {
			continue
		}
		if parseSummary.WindowStart == "" || parseCreatedAt < parseSummary.WindowStart {
			parseSummary.WindowStart = parseCreatedAt
		}
		if parseSummary.WindowEnd == "" || parseCreatedAt > parseSummary.WindowEnd {
			parseSummary.WindowEnd = parseCreatedAt
		}
	}
	return parseSummary
}

// parseBuildCustomerBillingTotals resolves one customer totals contract from plan metadata, invoice line items, and usage rows.
func parseBuildCustomerBillingTotals(parseDefaultCurrency string, parsePlan *parseBillingPlanRow, parseLatestInvoice parseBillingInvoiceRow, isHasLatestInvoice bool, parseLatestLineItems []parseBillingInvoiceLineItemRow, parseUsageSummary *chatpb.CustomerBillingUsageSummary) *chatpb.CustomerBillingTotals {
	parseCurrency := strings.ToUpper(strings.TrimSpace(parseDefaultCurrency))
	if parseCurrency == "" && isHasLatestInvoice {
		parseCurrency = strings.ToUpper(strings.TrimSpace(parseLatestInvoice.Currency))
	}
	if parseCurrency == "" {
		parseCurrency = "USD"
	}
	parseTotals := &chatpb.CustomerBillingTotals{Currency: parseCurrency}

	isHasLineBreakdown := false
	for _, parseLineItem := range parseLatestLineItems {
		parseLineType, parseErr := parseResolveUsageBasedBillingLineType(parseLineItem.LineType)
		if parseErr != nil {
			continue
		}
		isHasLineBreakdown = true
		switch parseLineType {
		case parseBillingLineTypePlatformFee:
			parseTotals.PlatformFeeCents += parseLineItem.AmountCents
		case parseBillingLineTypeUsageCost:
			parseTotals.UsageCostCents += parseLineItem.AmountCents
		case parseBillingLineTypeServicePremium:
			parseTotals.ServicePremiumCents += parseLineItem.AmountCents
		}
	}

	if !isHasLineBreakdown && parsePlan != nil {
		parseTotals.PlatformFeeCents = parsePlan.MonthlyPlatformFeeCents
		if parseTotals.PlatformFeeCents <= 0 {
			parseTotals.PlatformFeeCents = parsePlan.MonthlyBaseCents
		}
	}
	if parseTotals.UsageCostCents <= 0 && parseUsageSummary != nil && parseUsageSummary.GetUsageCostUsd() > 0 {
		parseTotals.UsageCostCents = int64(math.Round(parseUsageSummary.GetUsageCostUsd() * 100))
	}
	if parseTotals.TotalCents <= 0 {
		parseTotals.TotalCents = parseTotals.PlatformFeeCents + parseTotals.UsageCostCents + parseTotals.ServicePremiumCents
	}
	if parseTotals.TotalCents <= 0 && isHasLatestInvoice && parseLatestInvoice.TotalCents > 0 {
		parseTotals.TotalCents = parseLatestInvoice.TotalCents
	}
	return parseTotals
}

// GetCustomerBillingSummary returns one typed customer billing snapshot for the settings billing panel.
func (parseS *chatServer) GetCustomerBillingSummary(parseCtx context.Context, parseReq *chatpb.GetCustomerBillingSummaryRequest) (*chatpb.GetCustomerBillingSummaryResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetCustomerBillingSummary"))
	if parseReq == nil {
		parseReq = &chatpb.GetCustomerBillingSummaryRequest{}
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.GetCustomerBillingSummaryResponse{
		Subscriptions:          make([]*chatpb.BillingSubscriptionEntry, 0),
		Plans:                  make([]*chatpb.BillingPlanEntry, 0),
		UsageSummary:           &chatpb.CustomerBillingUsageSummary{},
		Totals:                 &chatpb.CustomerBillingTotals{Currency: "USD"},
		RecentInvoices:         make([]*chatpb.BillingInvoiceEntry, 0),
		RecentInvoiceLineItems: make([]*chatpb.BillingInvoiceLineItemEntry, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetCustomerBillingSummary: store unavailable")
		return parseResponse, nil
	}

	parseCustomerRow, isHasCustomer, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get billing customer by user: %v", parseErr)
	}
	if !isHasCustomer {
		return parseResponse, nil
	}
	parseResponse.Customer = parseBuildBillingCustomerEntry(parseCustomerRow)

	parseInvoiceLimit := parseClampCustomerBillingLimit(parseReq.GetInvoiceLimit(), 12)
	parseUsageLimit := parseClampCustomerBillingLimit(parseReq.GetUsageLimit(), 250)

	parseSubscriptionRows, parseErr := parseS.store.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, parseInvoiceLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing subscriptions by customer: %v", parseErr)
	}
	parsePlanCodeSet := map[string]struct{}{}
	for _, parseSubscriptionRow := range parseSubscriptionRows {
		parseResponse.Subscriptions = append(parseResponse.Subscriptions, parseBuildBillingSubscriptionEntry(parseSubscriptionRow))
		parsePlanCode := strings.TrimSpace(strings.ToLower(parseSubscriptionRow.PlanCode))
		if parsePlanCode == "" {
			continue
		}
		parsePlanCodeSet[parsePlanCode] = struct{}{}
	}

	parsePlanRows, parseErr := parseS.store.parseListBillingPlans(200)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing plans: %v", parseErr)
	}
	var parseActivePlan *parseBillingPlanRow
	for _, parsePlanRow := range parsePlanRows {
		parsePlanCode := strings.TrimSpace(strings.ToLower(parsePlanRow.PlanCode))
		if _, isHasPlan := parsePlanCodeSet[parsePlanCode]; !isHasPlan {
			continue
		}
		parsePlanRowCopy := parsePlanRow
		if parseActivePlan == nil {
			parseActivePlan = &parsePlanRowCopy
		}
		parseResponse.Plans = append(parseResponse.Plans, parseBuildSuperuserBillingPlanEntry(parsePlanRow))
	}

	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, parseInvoiceLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing invoices by customer: %v", parseErr)
	}
	var parseLatestInvoice parseBillingInvoiceRow
	isHasLatestInvoice := false
	parseLatestLineItems := make([]parseBillingInvoiceLineItemRow, 0)
	for parseInvoiceIndex, parseInvoiceRow := range parseInvoiceRows {
		parseResponse.RecentInvoices = append(parseResponse.RecentInvoices, parseBuildBillingInvoiceEntry(parseInvoiceRow))
		parseLineItemRows, parseErr2 := parseS.store.parseListBillingInvoiceLineItems(parseInvoiceRow.ID, parseCustomerRow.ID)
		if parseErr2 != nil {
			return nil, status.Errorf(codes.Internal, "list billing invoice line items: %v", parseErr2)
		}
		for _, parseLineItemRow := range parseLineItemRows {
			parseResponse.RecentInvoiceLineItems = append(parseResponse.RecentInvoiceLineItems, parseBuildBillingInvoiceLineItemEntry(parseLineItemRow))
		}
		if parseInvoiceIndex == 0 {
			parseLatestInvoice = parseInvoiceRow
			isHasLatestInvoice = true
			parseLatestLineItems = append(parseLatestLineItems[:0], parseLineItemRows...)
		}
	}

	parseUsageRows, parseErr := parseS.store.parseListUsageEvents(parseUserID, parseUsageLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list usage events: %v", parseErr)
	}
	parseResponse.UsageSummary = parseBuildCustomerBillingUsageSummary(parseUsageRows)
	parseResponse.Totals = parseBuildCustomerBillingTotals(
		parseCustomerRow.DefaultCurrency,
		parseActivePlan,
		parseLatestInvoice,
		isHasLatestInvoice,
		parseLatestLineItems,
		parseResponse.GetUsageSummary(),
	)
	return parseResponse, nil
}

// ListCustomerInvoiceHistory returns typed customer invoice and line-item history rows.
func (parseS *chatServer) ListCustomerInvoiceHistory(parseCtx context.Context, parseReq *chatpb.ListCustomerInvoiceHistoryRequest) (*chatpb.ListCustomerInvoiceHistoryResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListCustomerInvoiceHistory"))
	if parseReq == nil {
		parseReq = &chatpb.ListCustomerInvoiceHistoryRequest{}
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.ListCustomerInvoiceHistoryResponse{
		Invoices:         make([]*chatpb.BillingInvoiceEntry, 0),
		InvoiceLineItems: make([]*chatpb.BillingInvoiceLineItemEntry, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListCustomerInvoiceHistory: store unavailable")
		return parseResponse, nil
	}

	parseCustomerRow, isHasCustomer, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get billing customer by user: %v", parseErr)
	}
	if !isHasCustomer {
		return parseResponse, nil
	}

	parseLimit := parseClampCustomerBillingLimit(parseReq.GetLimit(), 24)
	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, parseLimit)
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
	return parseResponse, nil
}
