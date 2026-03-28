//go:build js && wasm

package app

import (
	"context"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parseUseAccountCostSummary(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseMarkdownWorkerRef ui.Ref[*interop.Worker],
	parseMarkdownWorkerPoolRef ui.Ref[*interop.WorkerPool],
	handleAuthFailure func(error) bool,
) accountCostSummary {
	parseInitial := parseDeriveAccountCostSummary(nil, parseConfiguredUsagePremiumPercent(), parseConfiguredPlatformFeeUSD(), 0)
	parseSummaryState := ui.UseState(parseInitial)
	parseRequestSeq := ui.UseRef(uint64(0))
	parseHasWorkerRequester := parseMarkdownWorkerRef.Get() != nil || parseMarkdownWorkerPoolRef.Get() != nil

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

		parseConversations := append([]convSummary(nil), parseCurrentState.ConversationList...)
		parseModels := append([]modelOption(nil), parseCurrentState.ModelOptions...)
		parseWorkerRequester, _ := parseResolveBackgroundRenderRequester(parseMarkdownWorkerRef, parseMarkdownWorkerPoolRef)
		parseNextSeq := parseRequestSeq.Get() + 1
		parseRequestSeq.Set(parseNextSeq)

		go func(parseSeq uint64, parseRows []convSummary, parseAvailableModels []modelOption, parsePremiumPct float64, parsePlatformFeeUSD float64, parseRequester interop.WorkerRequester) {
			if len(parseAvailableModels) == 0 {
				if parseRequestSeq.Get() == parseSeq {
					parseSummaryState.Set(parseDeriveAccountCostSummary(nil, parsePremiumPct, parsePlatformFeeUSD, len(parseRows)))
				}
				return
			}
			parseThreadSummaries := make([]threadCostSummary, 0, len(parseRows))
			parseFailedLookups := 0
			for _, parseRow := range parseRows {
				if parseRow.ID <= 0 {
					parseFailedLookups++
					continue
				}
				parseResp, parseErr := parseClient.LoadConversation(context.Background(), &chatpb.LoadConversationRequest{Id: parseRow.ID})
				if parseErr != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr) {
						return
					}
					parseFailedLookups++
					chatLog.Warn("account cost refresh: conversation load failed", logging.Fields{"conv_id": parseRow.ID, "error": parseErr})
					continue
				}
				parseLoadedMessages := make([]message, 0, len(parseResp.Messages))
				for _, parseCurrentMessage := range parseResp.Messages {
					parseLoadedMessages = append(parseLoadedMessages, message{
						Role:             parseCurrentMessage.Role,
						Content:          parseCurrentMessage.Content,
						ModelID:          parseCurrentMessage.GetModelId(),
						PromptTokens:     int(parseCurrentMessage.GetPromptTokens()),
						CompletionTokens: int(parseCurrentMessage.GetCompletionTokens()),
					})
				}
				if parseRequester != nil {
					parseSummary, parseWorkerErr := parseRequestWorkerThreadCostSummary(context.Background(), parseRequester, parseSeq, parseLoadedMessages, parseAvailableModels)
					if parseWorkerErr == nil {
						parseThreadSummaries = append(parseThreadSummaries, parseSummary)
						continue
					}
					chatLog.Warn("account cost refresh: worker thread cost summary failed; using sync fallback", logging.Fields{"conv_id": parseRow.ID, "error": parseWorkerErr})
				}
				parseThreadSummaries = append(parseThreadSummaries, parseDeriveThreadCostSummary(parseLoadedMessages, parseAvailableModels))
			}
			if parseRequestSeq.Get() != parseSeq {
				return
			}
			parseSummary := parseDeriveAccountCostSummary(parseThreadSummaries, parsePremiumPct, parsePlatformFeeUSD, parseFailedLookups)
			parseSummaryState.Set(parseSummary)
			chatLog.Info("account cost refreshed", logging.Fields{
				"thread_count":          parseSummary.ThreadCount,
				"exact_thread_costs":    parseSummary.ExactThreadCostCount,
				"failed_thread_lookups": parseSummary.FailedThreadLookups,
				"platform_fee_usd":      parseSummary.PlatformFee,
				"usage_cost_usd":        parseSummary.UsageCost,
				"premium_pct":           parseSummary.PremiumPercent,
				"total_cost_usd":        parseSummary.TotalCost,
			})
		}(parseNextSeq, parseConversations, parseModels, parsePremiumPercent, parsePlatformFee, parseWorkerRequester)

		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, parseCurrentState.ConversationList, parseCurrentState.ModelOptions, parseHasWorkerRequester)

	return parseSummaryState.Get()
}
