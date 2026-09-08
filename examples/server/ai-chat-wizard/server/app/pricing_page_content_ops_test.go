package app

import (
	"context"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
)

// parseFindPricingPagePlanContentEntry resolves one pricing-page plan entry by plan code.
func parseFindPricingPagePlanContentEntry(parseEntries []*chatpb.PricingPagePlanContentEntry, parsePlanCode string) (*chatpb.PricingPagePlanContentEntry, bool) {
	parsePlanCode = parseNormalizeSUKey(parsePlanCode)
	for _, parseEntry := range parseEntries {
		if parseNormalizeSUKey(parseEntry.GetPlanCode()) == parsePlanCode {
			return parseEntry, true
		}
	}
	return nil, false
}

// TestGetPricingPageContent falls back to seeded pricing content when no store is wired.
func TestGetPricingPageContent(parseT *testing.T) {
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, nil, parseNewTestLogger(), "all")
	parseResp, parseErr := parseServer.GetPricingPageContent(context.Background(), &chatpb.GetPricingPageContentRequest{})
	if parseErr != nil {
		parseT.Fatalf("GetPricingPageContent: %v", parseErr)
	}
	if parseResp.GetSource() != "seeded" {
		parseT.Fatalf("expected seeded source, got %+v", parseResp)
	}
	if len(parseResp.GetPlans()) < 3 {
		parseT.Fatalf("expected seeded plans, got %+v", parseResp.GetPlans())
	}
	parseProPlan, isHasProPlan := parseFindPricingPagePlanContentEntry(parseResp.GetPlans(), "pro")
	if !isHasProPlan || parseProPlan.GetMonthlyPlatformFeeCents() <= 0 || parseProPlan.GetUsagePremiumBasisPoints() <= 0 || parseProPlan.GetPlanDescription() == "" {
		parseT.Fatalf("expected seeded pro plan pricing copy, got %+v", parseProPlan)
	}
}

// TestGetPricingPageContentFromBillingPlans returns active billing-plan rows as one typed pricing content source.
func TestGetPricingPageContentFromBillingPlans(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	if _, parseErr := parseStore.parseUpsertSuperuserBillingPlan(parseSuperuserBillingPlanWrite{
		PlanCode:                "team",
		PlanName:                "Team",
		PlanRank:                20,
		IsActive:                true,
		MonthlyBaseCents:        10900,
		MonthlyPlatformFeeCents: 10900,
		YearlyBaseCents:         109000,
		UsagePremiumBasisPoints: 900,
		IncludedTokensMonthly:   20000000,
		IncludedSeats:           3,
		MinSeats:                3,
		WorkspaceMode:           "team",
		MaxSeats:                50,
		SupportsPriority:        true,
		SupportsCollaboration:   true,
		SupportsWorkspaceAdmin:  true,
		SupportsTeamWorkspace:   true,
		SupportsSSO:             false,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserBillingPlan team: %v", parseErr)
	}

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseResp, parseErr := parseServer.GetPricingPageContent(context.Background(), &chatpb.GetPricingPageContentRequest{})
	if parseErr != nil {
		parseT.Fatalf("GetPricingPageContent: %v", parseErr)
	}
	if parseResp.GetSource() != "billing_plans" {
		parseT.Fatalf("expected billing_plans source, got %+v", parseResp)
	}
	parseTeamPlan, isHasTeamPlan := parseFindPricingPagePlanContentEntry(parseResp.GetPlans(), "team")
	if !isHasTeamPlan {
		parseT.Fatalf("expected team plan content, got %+v", parseResp.GetPlans())
	}
	if parseTeamPlan.GetMonthlyPlatformFeeCents() != 10900 || parseTeamPlan.GetUsagePremiumBasisPoints() != 900 || parseTeamPlan.GetPlanDescription() == "" {
		parseT.Fatalf("unexpected team pricing plan content: %+v", parseTeamPlan)
	}
}
