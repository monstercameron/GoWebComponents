package app

import (
	"context"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseHasCustomerInvoiceBreakdownClassification reports whether one invoice-breakdown slice contains one classified line type.
func parseHasCustomerInvoiceBreakdownClassification(parseLines []*chatpb.CustomerInvoiceBreakdownLine, parseClassifiedLineType string) bool {
	parseClassifiedLineType = parseNormalizeSUKey(parseClassifiedLineType)
	for _, parseLine := range parseLines {
		if parseNormalizeSUKey(parseLine.GetClassifiedLineType()) == parseClassifiedLineType {
			return true
		}
	}
	return false
}

// TestGetCustomerBillingPreview returns one normalized platform-fee + usage + service-premium preview.
func TestGetCustomerBillingPreview(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "customer-billing-preview@example.com")
	parseCustomerRow, _, parseInvoiceRow := parseSeedCustomerBillingSnapshot(parseT, parseStore, parseUser.ID)

	parseLineItems := []parseBillingInvoiceLineItemWrite{
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "platform_fee", Description: "Platform fee", Quantity: 1, UnitAmountCents: 9900, AmountCents: 9900, Currency: "usd"},
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "usage_cost", Description: "Raw usage", Quantity: 1, UnitAmountCents: 800, AmountCents: 800, Currency: "usd"},
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "service_premium", Description: "Service premium", Quantity: 1, UnitAmountCents: 80, AmountCents: 80, Currency: "usd"},
	}
	for _, parseLineItemWrite := range parseLineItems {
		if _, parseErr := parseStore.parseCreateBillingInvoiceLineItem(parseLineItemWrite); parseErr != nil {
			parseT.Fatalf("parseCreateBillingInvoiceLineItem: %v", parseErr)
		}
	}

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseCtx := parseBindAuthUser(parseServer, "peer-customer-billing-preview", parseUser.ID, parseUser.Email)
	parseResp, parseErr := parseServer.GetCustomerBillingPreview(parseCtx, &chatpb.GetCustomerBillingPreviewRequest{
		InvoiceLimit: 10,
		UsageLimit:   10,
	})
	if parseErr != nil {
		parseT.Fatalf("GetCustomerBillingPreview: %v", parseErr)
	}
	parsePreview := parseResp.GetPreview()
	if parsePreview.GetPlanCode() != "team" || parsePreview.GetMonthlyPlatformFeeCents() != 9900 || parsePreview.GetRawModelUsageCents() != 800 || parsePreview.GetServicePremiumCents() != 80 || parsePreview.GetTotalCents() != 10780 || parsePreview.GetUsagePremiumBasisPoints() <= 0 || parsePreview.GetSource() != "invoice_line_items" {
		parseT.Fatalf("unexpected preview payload: %+v", parsePreview)
	}
}

// TestGetCustomerInvoiceBreakdown returns classified invoice rows for one customer scope.
func TestGetCustomerInvoiceBreakdown(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "customer-invoice-breakdown@example.com")
	parseCustomerRow, _, parseInvoiceRow := parseSeedCustomerBillingSnapshot(parseT, parseStore, parseUser.ID)

	parseLineItems := []parseBillingInvoiceLineItemWrite{
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "platform_fee", Description: "Platform fee", Quantity: 1, UnitAmountCents: 9900, AmountCents: 9900, Currency: "usd"},
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "usage_cost", Description: "Raw usage", Quantity: 1, UnitAmountCents: 700, AmountCents: 700, Currency: "usd"},
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "service_premium", Description: "Service premium", Quantity: 1, UnitAmountCents: 70, AmountCents: 70, Currency: "usd"},
	}
	for _, parseLineItemWrite := range parseLineItems {
		if _, parseErr := parseStore.parseCreateBillingInvoiceLineItem(parseLineItemWrite); parseErr != nil {
			parseT.Fatalf("parseCreateBillingInvoiceLineItem: %v", parseErr)
		}
	}

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseCtx := parseBindAuthUser(parseServer, "peer-customer-invoice-breakdown", parseUser.ID, parseUser.Email)
	parseResp, parseErr := parseServer.GetCustomerInvoiceBreakdown(parseCtx, &chatpb.GetCustomerInvoiceBreakdownRequest{
		InvoiceLimit: 10,
		LineLimit:    20,
	})
	if parseErr != nil {
		parseT.Fatalf("GetCustomerInvoiceBreakdown: %v", parseErr)
	}
	parseLines := parseResp.GetLines()
	if len(parseLines) != 3 {
		parseT.Fatalf("expected three invoice breakdown lines, got %+v", parseLines)
	}
	if !parseHasCustomerInvoiceBreakdownClassification(parseLines, parseBillingLineTypePlatformFee) || !parseHasCustomerInvoiceBreakdownClassification(parseLines, parseBillingLineTypeUsageCost) || !parseHasCustomerInvoiceBreakdownClassification(parseLines, parseBillingLineTypeServicePremium) {
		parseT.Fatalf("expected classified platform/usage/premium rows, got %+v", parseLines)
	}
}

// TestCustomerBillingPreviewRPCsRequireAuthentication verifies preview and invoice-breakdown RPCs deny unauthenticated callers.
func TestCustomerBillingPreviewRPCsRequireAuthentication(parseT *testing.T) {
	parseServer := &chatServer{logger: parseNewTestLogger()}
	if _, parseErr := parseServer.GetCustomerBillingPreview(context.Background(), &chatpb.GetCustomerBillingPreviewRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetCustomerBillingPreview status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr := parseServer.GetCustomerInvoiceBreakdown(context.Background(), &chatpb.GetCustomerInvoiceBreakdownRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetCustomerInvoiceBreakdown status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
}
