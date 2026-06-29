package provider

// ModelPricing stores token pricing as USD per 1M tokens.
type ModelPricing struct {
	InputPerMillionUSD  float64
	OutputPerMillionUSD float64
	Currency            string
}

// CostEstimate is the reusable provider-agnostic result of one pricing
// calculation.
type CostEstimate struct {
	InputCostUSD  float64
	OutputCostUSD float64
	TotalCostUSD  float64
}

// ParseEstimateCost calculates approximate request cost from token counts and model pricing metadata.
func ParseEstimateCost(parsePromptTokens, parseCompletionTokens int64, parsePricing ModelPricing) CostEstimate {
	parseInputCost := (float64(parsePromptTokens) / 1_000_000) * parsePricing.InputPerMillionUSD
	parseOutputCost := (float64(parseCompletionTokens) / 1_000_000) * parsePricing.OutputPerMillionUSD
	return CostEstimate{
		InputCostUSD:  parseInputCost,
		OutputCostUSD: parseOutputCost,
		TotalCostUSD:  parseInputCost + parseOutputCost,
	}
}
