package app

import (
	"context"
	"log/slog"
	"math"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseClampCustomerBillingPreviewLineLimit bounds invoice-breakdown line limits to one safe positive range.
func parseClampCustomerBillingPreviewLineLimit(parseRequested int32) int64 {
	if parseRequested <= 0 {
		return 250
	}
	if parseRequested > 2000 {
		return 2000
	}
	return int64(parseRequested)
}

// parseResolveCustomerBillingPlan selects one plan row matching the first available subscription plan code.
func parseResolveCustomerBillingPlan(parsePlanRows []parseBillingPlanRow, parseSubscriptionRows []parseBillingSubscriptionRow) *parseBillingPlanRow {
	parsePlanByCode := make(map[string]parseBillingPlanRow, len(parsePlanRows))
	for _, parsePlanRow := range parsePlanRows {
		parsePlanCode := strings.TrimSpace(strings.ToLower(parsePlanRow.PlanCode))
		if parsePlanCode == "" {
			continue
		}
		parsePlanByCode[parsePlanCode] = parsePlanRow
	}
	for _, parseSubscriptionRow := range parseSubscriptionRows {
		parsePlanCode := strings.TrimSpace(strings.ToLower(parseSubscriptionRow.PlanCode))
		if parsePlanCode == "" {
			continue
		}
		parsePlanRow, isParseFound := parsePlanByCode[parsePlanCode]
		if !isParseFound {
			continue
		}
		parsePlanCopy := parsePlanRow
		return &parsePlanCopy
	}
	return nil
}

// parseBuildCustomerBillingPreview computes one typed preview contract from plan metadata, invoice lines, and usage rows.
func parseBuildCustomerBillingPreview(parsePlan *parseBillingPlanRow, parseLatestInvoice parseBillingInvoiceRow, isHasLatestInvoice bool, parseLatestLineItems []parseBillingInvoiceLineItemRow, parseUsageRows []parseUsageEventRow) (*chatpb.CustomerBillingPreviewBreakdown, error) {
	parseCurrency := "USD"
	if isHasLatestInvoice {
		parseInvoiceCurrency := strings.ToUpper(strings.TrimSpace(parseLatestInvoice.Currency))
		if parseInvoiceCurrency != "" {
			parseCurrency = parseInvoiceCurrency
		}
	}

	parsePreview := &chatpb.CustomerBillingPreviewBreakdown{
		Currency: parseCurrency,
		Source:   "computed",
	}
	isHasClassifiedLineItems := false
	for _, parseLineItem := range parseLatestLineItems {
		parseClassifiedLineType, parseErr := parseResolveUsageBasedBillingLineType(parseLineItem.LineType)
		if parseErr != nil {
			continue
		}
		isHasClassifiedLineItems = true
		switch parseClassifiedLineType {
		case parseBillingLineTypePlatformFee:
			parsePreview.MonthlyPlatformFeeCents += parseLineItem.AmountCents
		case parseBillingLineTypeUsageCost:
			parsePreview.RawModelUsageCents += parseLineItem.AmountCents
		case parseBillingLineTypeServicePremium:
			parsePreview.ServicePremiumCents += parseLineItem.AmountCents
		}
	}
	if parsePreview.RawModelUsageCents <= 0 {
		for _, parseUsageRow := range parseUsageRows {
			parsePreview.RawModelUsageCents += int64(math.Round(parseUsageRow.TotalCostUSD * 100))
		}
	}

	if parsePlan != nil {
		parsePreview.PlanCode = parsePlan.PlanCode
		parsePreview.UsagePremiumBasisPoints = parsePlan.UsagePremiumBasisPoints
		if parsePreview.UsagePremiumBasisPoints < 0 {
			parsePreview.UsagePremiumBasisPoints = 0
		}
		if parsePreview.MonthlyPlatformFeeCents <= 0 {
			parsePreview.MonthlyPlatformFeeCents = parsePlan.MonthlyPlatformFeeCents
			if parsePreview.MonthlyPlatformFeeCents <= 0 {
				parsePreview.MonthlyPlatformFeeCents = parsePlan.MonthlyBaseCents
			}
		}
	}

	if isHasClassifiedLineItems {
		parsePreview.Source = "invoice_line_items"
		parsePreview.TotalCents = parsePreview.MonthlyPlatformFeeCents + parsePreview.RawModelUsageCents + parsePreview.ServicePremiumCents
		if parsePreview.TotalCents <= 0 && isHasLatestInvoice && parseLatestInvoice.TotalCents > 0 {
			parsePreview.TotalCents = parseLatestInvoice.TotalCents
		}
		return parsePreview, nil
	}

	parseFormulaPreview, parseErr := parseBuildUsageBasedBillingPreview(parsePreview.MonthlyPlatformFeeCents, parsePreview.RawModelUsageCents, parsePreview.UsagePremiumBasisPoints)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePreview.ServicePremiumCents = parseFormulaPreview.servicePremiumCents
	parsePreview.TotalCents = parseFormulaPreview.totalCents
	if parsePreview.TotalCents <= 0 && isHasLatestInvoice && parseLatestInvoice.TotalCents > 0 {
		parsePreview.TotalCents = parseLatestInvoice.TotalCents
	}
	return parsePreview, nil
}

// parseBuildCustomerInvoiceBreakdownLine maps one invoice line item into one typed customer invoice-breakdown row.
func parseBuildCustomerInvoiceBreakdownLine(parseInvoiceRow parseBillingInvoiceRow, parseLineItemRow parseBillingInvoiceLineItemRow) *chatpb.CustomerInvoiceBreakdownLine {
	parseClassifiedLineType := "other"
	if parseResolvedLineType, parseErr := parseResolveUsageBasedBillingLineType(parseLineItemRow.LineType); parseErr == nil {
		parseClassifiedLineType = parseResolvedLineType
	}
	return &chatpb.CustomerInvoiceBreakdownLine{
		InvoiceId:          parseInvoiceRow.ID,
		InvoiceStatus:      parseInvoiceRow.Status,
		Currency:           parseLineItemRow.Currency,
		LineType:           parseLineItemRow.LineType,
		ClassifiedLineType: parseClassifiedLineType,
		Description:        parseLineItemRow.Description,
		Quantity:           parseLineItemRow.Quantity,
		UnitAmountCents:    parseLineItemRow.UnitAmountCents,
		AmountCents:        parseLineItemRow.AmountCents,
		PeriodStart:        parseLineItemRow.PeriodStart,
		PeriodEnd:          parseLineItemRow.PeriodEnd,
		CreatedAt:          parseLineItemRow.CreatedAt,
	}
}

// GetCustomerBillingPreview returns one typed customer billing preview for platform fee, usage cost, service premium, and total.
func (parseS *chatServer) GetCustomerBillingPreview(parseCtx context.Context, parseReq *chatpb.GetCustomerBillingPreviewRequest) (*chatpb.GetCustomerBillingPreviewResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetCustomerBillingPreview"))
	if parseReq == nil {
		parseReq = &chatpb.GetCustomerBillingPreviewRequest{}
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.GetCustomerBillingPreviewResponse{
		Preview: &chatpb.CustomerBillingPreviewBreakdown{
			Currency: "USD",
			Source:   "none",
		},
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetCustomerBillingPreview: store unavailable")
		return parseResponse, nil
	}

	parseCustomerRow, isHasCustomer, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get billing customer by user: %v", parseErr)
	}
	if !isHasCustomer {
		return parseResponse, nil
	}

	parseInvoiceLimit := parseClampCustomerBillingLimit(parseReq.GetInvoiceLimit(), 12)
	parseUsageLimit := parseClampCustomerBillingLimit(parseReq.GetUsageLimit(), 250)
	parseSubscriptionRows, parseErr := parseS.store.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, parseInvoiceLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing subscriptions by customer: %v", parseErr)
	}
	parsePlanRows, parseErr := parseS.store.parseListBillingPlans(200)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing plans: %v", parseErr)
	}
	parseSelectedPlan := parseResolveCustomerBillingPlan(parsePlanRows, parseSubscriptionRows)

	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, parseInvoiceLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing invoices by customer: %v", parseErr)
	}
	var parseLatestInvoice parseBillingInvoiceRow
	isHasLatestInvoice := false
	parseLatestLineItems := make([]parseBillingInvoiceLineItemRow, 0)
	if len(parseInvoiceRows) > 0 {
		parseLatestInvoice = parseInvoiceRows[0]
		isHasLatestInvoice = true
		parseLatestLineItems, parseErr = parseS.store.parseListBillingInvoiceLineItems(parseLatestInvoice.ID, parseCustomerRow.ID)
		if parseErr != nil {
			return nil, status.Errorf(codes.Internal, "list billing invoice line items: %v", parseErr)
		}
	}

	parseUsageRows, parseErr := parseS.store.parseListUsageEvents(parseUserID, parseUsageLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list usage events: %v", parseErr)
	}
	parsePreview, parseErr := parseBuildCustomerBillingPreview(parseSelectedPlan, parseLatestInvoice, isHasLatestInvoice, parseLatestLineItems, parseUsageRows)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "build customer billing preview: %v", parseErr)
	}
	parseResponse.Preview = parsePreview
	return parseResponse, nil
}

// GetCustomerInvoiceBreakdown returns classified customer invoice line rows for billing details and drill-down surfaces.
func (parseS *chatServer) GetCustomerInvoiceBreakdown(parseCtx context.Context, parseReq *chatpb.GetCustomerInvoiceBreakdownRequest) (*chatpb.GetCustomerInvoiceBreakdownResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetCustomerInvoiceBreakdown"))
	if parseReq == nil {
		parseReq = &chatpb.GetCustomerInvoiceBreakdownRequest{}
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.GetCustomerInvoiceBreakdownResponse{
		Lines: make([]*chatpb.CustomerInvoiceBreakdownLine, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetCustomerInvoiceBreakdown: store unavailable")
		return parseResponse, nil
	}

	parseCustomerRow, isHasCustomer, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get billing customer by user: %v", parseErr)
	}
	if !isHasCustomer {
		return parseResponse, nil
	}

	parseInvoiceLimit := parseClampCustomerBillingLimit(parseReq.GetInvoiceLimit(), 12)
	parseLineLimit := parseClampCustomerBillingPreviewLineLimit(parseReq.GetLineLimit())
	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, parseInvoiceLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing invoices by customer: %v", parseErr)
	}

	for _, parseInvoiceRow := range parseInvoiceRows {
		parseLineItemRows, parseErr2 := parseS.store.parseListBillingInvoiceLineItems(parseInvoiceRow.ID, parseCustomerRow.ID)
		if parseErr2 != nil {
			return nil, status.Errorf(codes.Internal, "list billing invoice line items: %v", parseErr2)
		}
		for _, parseLineItemRow := range parseLineItemRows {
			parseResponse.Lines = append(parseResponse.Lines, parseBuildCustomerInvoiceBreakdownLine(parseInvoiceRow, parseLineItemRow))
			if int64(len(parseResponse.Lines)) >= parseLineLimit {
				return parseResponse, nil
			}
		}
	}
	return parseResponse, nil
}
