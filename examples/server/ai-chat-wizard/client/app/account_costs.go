//go:build js && wasm

package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/logging"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func parseUseAccountCostSummary(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseMarkdownWorkerRef ui.Ref[*interop.Worker],
	parseMarkdownWorkerPoolRef ui.Ref[*interop.WorkerPool],
	handleAuthFailure func(error) bool,
) accountCostSummary {
	_ = parseMarkdownWorkerRef
	_ = parseMarkdownWorkerPoolRef
	parseInitial := parseDeriveAccountCostSummary(nil, parseConfiguredUsagePremiumPercent(), parseConfiguredPlatformFeeUSD(), 0)
	parseSummaryState := ui.UseState(parseInitial)
	parseRequestSeq := ui.UseRef(uint64(0))

	ui.UseEffect(func() func() {
		parsePremiumPercent := parseConfiguredUsagePremiumPercent()
		parsePlatformFee := parseConfiguredPlatformFeeUSD()
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady {
			parseSummaryState.Set(parseDeriveAccountCostSummary(nil, parsePremiumPercent, parsePlatformFee, 0))
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			parseSummaryState.Set(parseDeriveAccountCostSummary(nil, parsePremiumPercent, parsePlatformFee, 0))
			return nil
		}
		parseNextSeq := parseRequestSeq.Get() + 1
		parseRequestSeq.Set(parseNextSeq)

		go func(parseSeq uint64, parsePremiumPct float64, parsePlatformFeeUSD float64) {
			parseResp, parseErr := parseClient.GetCustomerBillingSummary(context.Background(), &chatpb.GetCustomerBillingSummaryRequest{
				InvoiceLimit: 12,
				UsageLimit:   250,
			})
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				chatLog.Warn("account cost refresh: customer billing summary failed", logging.Fields{"error": parseErr})
				if parseRequestSeq.Get() == parseSeq {
					parseSummaryState.Set(parseDeriveAccountCostSummary(nil, parsePremiumPct, parsePlatformFeeUSD, 0))
				}
				return
			}
			if parseRequestSeq.Get() != parseSeq {
				return
			}
			parseSummary := parseBuildAccountCostSummaryFromBillingSummary(parseResp, parsePremiumPct, parsePlatformFeeUSD)
			parseSummaryState.Set(parseSummary)
			chatLog.Info("account cost refreshed", logging.Fields{
				"usage_events":     parseSummary.ThreadCount,
				"platform_fee_usd": parseSummary.PlatformFee,
				"usage_cost_usd":   parseSummary.UsageCost,
				"premium_pct":      parseSummary.PremiumPercent,
				"total_cost_usd":   parseSummary.TotalCost,
			})
		}(parseNextSeq, parsePremiumPercent, parsePlatformFee)

		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady)

	return parseSummaryState.Get()
}

// parseBuildAccountCostSummaryFromBillingSummary maps one typed customer billing summary into the settings billing view model.
func parseBuildAccountCostSummaryFromBillingSummary(parseResp *chatpb.GetCustomerBillingSummaryResponse, parseDefaultPremiumPercent float64, _ float64) accountCostSummary {
	parseSummary := parseDeriveAccountCostSummary(nil, parseDefaultPremiumPercent, 0, 0)
	if parseResp == nil {
		return parseSummary
	}
	parseTotals := parseResp.GetTotals()
	if parseTotals != nil {
		parseSummary.PlatformFee = float64(parseTotals.GetPlatformFeeCents()) / 100.0
		parseSummary.UsageCost = float64(parseTotals.GetUsageCostCents()) / 100.0
		parseSummary.PremiumCost = float64(parseTotals.GetServicePremiumCents()) / 100.0
		parseSummary.TotalCost = float64(parseTotals.GetTotalCents()) / 100.0
		if parseSummary.TotalCost <= 0 {
			parseSummary.TotalCost = parseSummary.PlatformFee + parseSummary.UsageCost + parseSummary.PremiumCost
		}
	}
	parseUsageSummary := parseResp.GetUsageSummary()
	if parseUsageSummary != nil {
		parseSummary.ThreadCount = int(parseUsageSummary.GetEventCount())
		parseSummary.ExactThreadCostCount = parseSummary.ThreadCount
	}
	parseSummary.HasAnyExactCosts = parseSummary.TotalCost > 0 || parseSummary.UsageCost > 0 || parseSummary.PremiumCost > 0
	parseSummary.AllThreadCostsExact = true
	parseSummary.HasCoverageGaps = false
	parseSummary.FailedThreadLookups = 0
	if parseSummary.UsageCost > 0 {
		parseSummary.PremiumPercent = (parseSummary.PremiumCost / parseSummary.UsageCost) * 100
	}
	// Extract plan name from server-provided plans, then subscription as fallback.
	if parsePlans := parseResp.GetPlans(); len(parsePlans) > 0 {
		if parsePlanName := strings.TrimSpace(parsePlans[0].GetPlanName()); parsePlanName != "" {
			parseSummary.PlanLabel = parsePlanName
		}
	}
	if parseSummary.PlanLabel == "" {
		if parseSubs := parseResp.GetSubscriptions(); len(parseSubs) > 0 {
			if parsePlanCode := strings.TrimSpace(parseSubs[0].GetPlanCode()); parsePlanCode != "" {
				parseSummary.PlanLabel = parsePlanCode
			}
		}
	}
	// Extract recent invoices.
	for _, parseInv := range parseResp.GetRecentInvoices() {
		parseSummary.Invoices = append(parseSummary.Invoices, billingInvoiceRow{
			PeriodStart: parseInv.GetPeriodStart(),
			PeriodEnd:   parseInv.GetPeriodEnd(),
			TotalCents:  parseInv.GetTotalCents(),
			Status:      parseInv.GetStatus(),
		})
	}
	return parseSummary
}
