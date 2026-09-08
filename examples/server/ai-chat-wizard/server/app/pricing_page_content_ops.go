package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildPricingPageSeedPlanRows returns one canonical server-seeded pricing-plan content snapshot.
func parseBuildPricingPageSeedPlanRows() []parseBillingPlanRow {
	return []parseBillingPlanRow{
		{
			PlanCode:                "pro",
			PlanName:                "Pro",
			IsActive:                true,
			MonthlyPlatformFeeCents: 2900,
			MonthlyBaseCents:        2900,
			UsagePremiumBasisPoints: 1200,
		},
		{
			PlanCode:                "team",
			PlanName:                "Team",
			IsActive:                true,
			MonthlyPlatformFeeCents: 9900,
			MonthlyBaseCents:        9900,
			UsagePremiumBasisPoints: 1000,
		},
		{
			PlanCode:                "enterprise",
			PlanName:                "Enterprise",
			IsActive:                true,
			MonthlyPlatformFeeCents: 0,
			MonthlyBaseCents:        0,
			UsagePremiumBasisPoints: 0,
		},
	}
}

// parseResolvePricingPagePlanDescription resolves one stable public pricing description by plan code.
func parseResolvePricingPagePlanDescription(parsePlanCode string) string {
	switch strings.TrimSpace(strings.ToLower(parsePlanCode)) {
	case "pro":
		return "Solo workspace with one operator seat, clear platform fee, and full core chat workflows."
	case "team":
		return "Shared workspace with collaboration and workspace-admin controls for growing teams."
	case "enterprise":
		return "Contract-gated workspace with security review, procurement support, and governed admin controls."
	default:
		return "Usage-based workspace plan."
	}
}

// parseBuildPricingPagePlanContentEntry maps one billing-plan row into one typed pricing-page plan entry.
func parseBuildPricingPagePlanContentEntry(parseRow parseBillingPlanRow) *chatpb.PricingPagePlanContentEntry {
	parseMonthlyPlatformFeeCents := parseRow.MonthlyPlatformFeeCents
	if parseMonthlyPlatformFeeCents <= 0 {
		parseMonthlyPlatformFeeCents = parseRow.MonthlyBaseCents
	}
	return &chatpb.PricingPagePlanContentEntry{
		PlanCode:                parseRow.PlanCode,
		PlanName:                parseRow.PlanName,
		PlanDescription:         parseResolvePricingPagePlanDescription(parseRow.PlanCode),
		MonthlyPlatformFeeCents: parseMonthlyPlatformFeeCents,
		UsagePremiumBasisPoints: parseRow.UsagePremiumBasisPoints,
	}
}

// parseBuildPricingPagePlanContentEntries filters one billing-plan slice down to active plans for public pricing content.
func parseBuildPricingPagePlanContentEntries(parseRows []parseBillingPlanRow) []*chatpb.PricingPagePlanContentEntry {
	parseEntries := make([]*chatpb.PricingPagePlanContentEntry, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseRow.IsActive {
			continue
		}
		parseEntries = append(parseEntries, parseBuildPricingPagePlanContentEntry(parseRow))
	}
	return parseEntries
}

// GetPricingPageContent returns one typed pricing-page content source from billing-plan metadata.
func (parseS *chatServer) GetPricingPageContent(parseCtx context.Context, parseReq *chatpb.GetPricingPageContentRequest) (*chatpb.GetPricingPageContentResponse, error) {
	_ = parseCtx
	_ = parseReq
	parseResponse := &chatpb.GetPricingPageContentResponse{
		Plans:  parseBuildPricingPagePlanContentEntries(parseBuildPricingPageSeedPlanRows()),
		Source: "seeded",
	}
	if parseS == nil || parseS.store == nil {
		return parseResponse, nil
	}
	parseRows, parseErr := parseS.store.parseListBillingPlans(200)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list billing plans: %v", parseErr)
	}
	parseEntries := parseBuildPricingPagePlanContentEntries(parseRows)
	if len(parseEntries) == 0 {
		return parseResponse, nil
	}
	return &chatpb.GetPricingPageContentResponse{
		Plans:  parseEntries,
		Source: "billing_plans",
	}, nil
}
