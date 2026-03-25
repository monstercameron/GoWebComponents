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

// EstimateCost calculates approximate request cost from token counts and model
// pricing metadata.
func EstimateCost(promptTokens, completionTokens int64, pricing ModelPricing) CostEstimate {
	inputCost := (float64(promptTokens) / 1_000_000) * pricing.InputPerMillionUSD
	outputCost := (float64(completionTokens) / 1_000_000) * pricing.OutputPerMillionUSD
	return CostEstimate{
		InputCostUSD:  inputCost,
		OutputCostUSD: outputCost,
		TotalCostUSD:  inputCost + outputCost,
	}
}
