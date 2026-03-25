//go:build js && wasm

package app

import (
	"context"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

func useAccountCostSummary(
	currentState appState,
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) accountCostSummary {
	initial := deriveAccountCostSummary(nil, configuredUsagePremiumPercent(), 0)
	summaryState := ui.UseState(initial)
	requestSeq := ui.UseRef(uint64(0))

	ui.UseEffect(func() func() {
		premiumPercent := configuredUsagePremiumPercent()
		if !currentState.Authenticated || !currentState.GRPCReady {
			summaryState.Set(deriveAccountCostSummary(nil, premiumPercent, 0))
			return nil
		}
		client := chatClientRef.Get()
		if client == nil {
			summaryState.Set(deriveAccountCostSummary(nil, premiumPercent, 0))
			return nil
		}

		conversations := append([]convSummary(nil), currentState.ConversationList...)
		models := append([]modelOption(nil), currentState.ModelOptions...)
		nextSeq := requestSeq.Get() + 1
		requestSeq.Set(nextSeq)

		go func(seq uint64, rows []convSummary, availableModels []modelOption, premiumPct float64) {
			if len(availableModels) == 0 {
				if requestSeq.Get() == seq {
					summaryState.Set(deriveAccountCostSummary(nil, premiumPct, len(rows)))
				}
				return
			}
			threadSummaries := make([]threadCostSummary, 0, len(rows))
			failedLookups := 0
			for _, row := range rows {
				if row.ID <= 0 {
					failedLookups++
					continue
				}
				resp, err := client.LoadConversation(context.Background(), &chatpb.LoadConversationRequest{Id: row.ID})
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					failedLookups++
					chatLog.Warn("account cost refresh: conversation load failed", logging.Fields{"conv_id": row.ID, "error": err})
					continue
				}
				loadedMessages := make([]message, 0, len(resp.Messages))
				for _, currentMessage := range resp.Messages {
					loadedMessages = append(loadedMessages, message{
						Role:             currentMessage.Role,
						Content:          currentMessage.Content,
						ModelID:          currentMessage.GetModelId(),
						PromptTokens:     int(currentMessage.GetPromptTokens()),
						CompletionTokens: int(currentMessage.GetCompletionTokens()),
					})
				}
				threadSummaries = append(threadSummaries, deriveThreadCostSummary(loadedMessages, availableModels))
			}
			if requestSeq.Get() != seq {
				return
			}
			summary := deriveAccountCostSummary(threadSummaries, premiumPct, failedLookups)
			summaryState.Set(summary)
			chatLog.Info("account cost refreshed", logging.Fields{
				"thread_count":          summary.ThreadCount,
				"exact_thread_costs":    summary.ExactThreadCostCount,
				"failed_thread_lookups": summary.FailedThreadLookups,
				"usage_cost_usd":        summary.UsageCost,
				"premium_pct":           summary.PremiumPercent,
				"total_cost_usd":        summary.TotalCost,
			})
		}(nextSeq, conversations, models, premiumPercent)

		return nil
	}, currentState.Authenticated, currentState.GRPCReady, currentState.ConversationList, currentState.ModelOptions)

	return summaryState.Get()
}
