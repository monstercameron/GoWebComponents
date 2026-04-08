//go:build js && wasm

package app

import "testing"

// TestParseShouldRefreshWorkerDerivedSignature verifies redundant worker-derived refreshes are skipped when the applied signature already matches.
func TestParseShouldRefreshWorkerDerivedSignature(parseT *testing.T) {
	if parseShouldRefreshWorkerDerivedSignature("sig-1", "sig-1") {
		parseT.Fatal("expected identical signatures to skip worker-derived refresh")
	}
	if !parseShouldRefreshWorkerDerivedSignature("sig-2", "sig-1") {
		parseT.Fatal("expected changed signatures to refresh worker-derived state")
	}
}

// TestParseThreadCostSummarySignatureIgnoresUnusedModels verifies thread-cost signatures do not churn when unrelated model pricing rows change.
func TestParseThreadCostSummarySignatureIgnoresUnusedModels(parseT *testing.T) {
	parseMessages := []message{
		{Role: roleAssistant, Content: "answer", ModelID: "gpt-5.4", PromptTokens: 10, CompletionTokens: 20},
	}
	parseBaseModels := []modelOption{
		{ID: "gpt-5.4", Pricing: modelPricing{InputDollarsPerMillion: 1.25, OutputDollarsPerMillion: 10, Currency: "USD"}},
	}
	parseExpandedModels := append(append([]modelOption(nil), parseBaseModels...),
		modelOption{ID: "unused-model", Pricing: modelPricing{InputDollarsPerMillion: 99, OutputDollarsPerMillion: 199, Currency: "USD"}},
	)

	parseBaseSignature := parseThreadCostSummarySignature(parseMessages, parseBaseModels)
	parseExpandedSignature := parseThreadCostSummarySignature(parseMessages, parseExpandedModels)
	if parseBaseSignature != parseExpandedSignature {
		parseT.Fatalf("expected unused models to leave thread-cost signature unchanged, got %q vs %q", parseBaseSignature, parseExpandedSignature)
	}
}

// TestParseBuildWorkerThreadCostSummaryRequestFiltersUnusedPayload verifies the worker cost-summary request excludes irrelevant messages and unused model rows.
func TestParseBuildWorkerThreadCostSummaryRequestFiltersUnusedPayload(parseT *testing.T) {
	parseMessages := []message{
		{Role: roleUser, Content: "user prompt"},
		{Role: roleAssistant, Content: "assistant answer", ModelID: "gpt-5.4", PromptTokens: 10, CompletionTokens: 20},
		{Role: roleAssistant, Pending: true, Content: "pending", ModelID: "gpt-5.4-mini"},
		{Role: roleAssistant, Content: "   ", Thought: "thought-only", ModelID: "gpt-5.4-nano"},
	}
	parseModels := []modelOption{
		{ID: "gpt-5.4", Pricing: modelPricing{InputDollarsPerMillion: 1.25, OutputDollarsPerMillion: 10, Currency: "USD"}},
		{ID: "gpt-5.4-mini", Pricing: modelPricing{InputDollarsPerMillion: 0.25, OutputDollarsPerMillion: 2, Currency: "USD"}},
		{ID: "gpt-5.4-nano", Pricing: modelPricing{InputDollarsPerMillion: 0.05, OutputDollarsPerMillion: 0.4, Currency: "USD"}},
	}

	parseRequest := parseBuildWorkerThreadCostSummaryRequest(7, parseMessages, parseModels)
	if len(parseRequest.GetMessage) != 1 {
		parseT.Fatalf("expected 1 relevant assistant cost message, got %d", len(parseRequest.GetMessage))
	}
	if parseRequest.GetMessage[0].GetMessageIndex != 1 {
		parseT.Fatalf("expected assistant message index 1, got %d", parseRequest.GetMessage[0].GetMessageIndex)
	}
	if len(parseRequest.GetModel) != 1 || string(parseRequest.GetModel[0].GetModelIDBytes) != "gpt-5.4" {
		parseT.Fatalf("expected only used model pricing row, got %+v", parseRequest.GetModel)
	}
}

// TestParseBuildWorkerRenderSignatureRequestFiltersToAssistantMessages verifies render-signature requests exclude non-assistant and pending rows while keeping assistant metadata inputs.
func TestParseBuildWorkerRenderSignatureRequestFiltersToAssistantMessages(parseT *testing.T) {
	parseMessages := []message{
		{Role: roleUser, Content: "user prompt"},
		{Role: roleAssistant, Content: "assistant answer", Thought: "reasoning", ModelID: "gpt-5.4", PromptTokens: 10, CompletionTokens: 20},
		{Role: roleAssistant, Content: "", Thought: "thought-only", ModelID: "gpt-5.4-mini"},
		{Role: roleAssistant, Pending: true, Content: "pending", ModelID: "gpt-5.4-nano"},
	}
	parseModels := []modelOption{
		{ID: "gpt-5.4", Pricing: modelPricing{InputDollarsPerMillion: 1.25, OutputDollarsPerMillion: 10, Currency: "USD"}},
		{ID: "gpt-5.4-mini", Pricing: modelPricing{InputDollarsPerMillion: 0.25, OutputDollarsPerMillion: 2, Currency: "USD"}},
		{ID: "gpt-5.4-nano", Pricing: modelPricing{InputDollarsPerMillion: 0.05, OutputDollarsPerMillion: 0.4, Currency: "USD"}},
	}

	parseRequest := parseBuildWorkerRenderSignatureRequest(9, parseMessages, parseModels)
	if len(parseRequest.GetMessage) != 2 {
		parseT.Fatalf("expected 2 completed assistant signature messages, got %d", len(parseRequest.GetMessage))
	}
	if parseRequest.GetMessage[0].GetMessageIndex != 1 || parseRequest.GetMessage[1].GetMessageIndex != 2 {
		parseT.Fatalf("unexpected assistant signature message indexes: %+v", parseRequest.GetMessage)
	}
	if len(parseRequest.GetModel) != 1 || string(parseRequest.GetModel[0].GetModelIDBytes) != "gpt-5.4" {
		parseT.Fatalf("expected only model row used by content-bearing assistant messages, got %+v", parseRequest.GetModel)
	}
}

// TestParseBuildAssistantMessageMetadataDeltaRetainsUnchangedEntries verifies unchanged assistant metadata cache entries are retained and only changed messages are requested.
func TestParseBuildAssistantMessageMetadataDeltaRetainsUnchangedEntries(parseT *testing.T) {
	parseMessages := []message{
		{Role: roleAssistant, Content: "alpha", Thought: "reason alpha"},
		{Role: roleAssistant, Content: "beta", Thought: "reason beta updated"},
		{Role: roleUser, Content: "ignored"},
	}
	parseThoughtCacheByMessage := map[int]renderWorkerThoughtCacheEntry{
		0: {GetThoughtText: "reason alpha", GetSection: []thoughtSection{{Key: "0", Heading: "Plan", Body: "alpha"}}},
		1: {GetThoughtText: "reason beta", GetSection: []thoughtSection{{Key: "1", Heading: "Plan", Body: "beta"}}},
	}
	parseCanvasCacheByMessage := map[int]renderWorkerCanvasCacheEntry{
		0: {GetContentText: "alpha", GetArtifact: []canvasArtifact{{ID: "a", MessageIndex: 0, Label: "A"}}},
		1: {GetContentText: "beta", GetArtifact: []canvasArtifact{{ID: "b", MessageIndex: 1, Label: "B"}}},
	}

	parseItems, parseRetainedThoughtCacheByMessage, parseRetainedCanvasCacheByMessage := parseBuildAssistantMessageMetadataDelta(
		parseMessages,
		parseThoughtCacheByMessage,
		parseCanvasCacheByMessage,
	)

	if len(parseItems) != 1 {
		parseT.Fatalf("expected 1 changed metadata request item, got %d", len(parseItems))
	}
	if parseItems[0].GetMessageIndex != 1 {
		parseT.Fatalf("expected changed message index 1, got %d", parseItems[0].GetMessageIndex)
	}
	if len(parseRetainedThoughtCacheByMessage) != 1 || len(parseRetainedCanvasCacheByMessage) != 1 {
		parseT.Fatalf("expected only unchanged entry to be retained, got thought=%d canvas=%d", len(parseRetainedThoughtCacheByMessage), len(parseRetainedCanvasCacheByMessage))
	}
	if parseRetainedThoughtCacheByMessage[0].GetThoughtText != "reason alpha" {
		parseT.Fatalf("expected message 0 thought cache to be retained, got %q", parseRetainedThoughtCacheByMessage[0].GetThoughtText)
	}
	if parseRetainedCanvasCacheByMessage[0].GetContentText != "alpha" {
		parseT.Fatalf("expected message 0 canvas cache to be retained, got %q", parseRetainedCanvasCacheByMessage[0].GetContentText)
	}
}

// TestParseBuildAssistantMessageMetadataTextByIndexPrefersLocalText verifies metadata lookup reuses local text fields instead of rebuilding strings from bytes when available.
func TestParseBuildAssistantMessageMetadataTextByIndexPrefersLocalText(parseT *testing.T) {
	parseItems := []renderWorkerMessageMetadataMessageRequest{
		{
			GetMessageIndex: 7,
			GetContentBytes: []byte("fallback content"),
			GetThoughtBytes: []byte("fallback thought"),
			GetContentText:  "local content",
			GetThoughtText:  "local thought",
		},
		{
			GetMessageIndex: 9,
			GetContentBytes: []byte("byte content"),
			GetThoughtBytes: []byte("byte thought"),
		},
	}

	parseTextByIndex := parseBuildAssistantMessageMetadataTextByIndex(parseItems)
	if parseTextByIndex[7].GetContentText != "local content" || parseTextByIndex[7].GetThoughtText != "local thought" {
		parseT.Fatalf("expected local metadata text to win, got %+v", parseTextByIndex[7])
	}
	if parseTextByIndex[9].GetContentText != "byte content" || parseTextByIndex[9].GetThoughtText != "byte thought" {
		parseT.Fatalf("expected byte metadata fallback text, got %+v", parseTextByIndex[9])
	}
}
