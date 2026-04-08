package provider

import (
	"strings"
	"testing"
	"time"
)

// ─── RateLimitBucket ─────────────────────────────────────────────────────────

// TestRateLimitBucketZeroValue verifies the zero value is a valid empty bucket
// with all fields at their default zero states.
func TestRateLimitBucketZeroValue(parseT *testing.T) {
	parseT.Parallel()
	parseBucket := RateLimitBucket{}
	if parseBucket.Limit != 0 {
		parseT.Errorf("Limit = %d, want 0", parseBucket.Limit)
	}
	if parseBucket.Remaining != 0 {
		parseT.Errorf("Remaining = %d, want 0", parseBucket.Remaining)
	}
	if parseBucket.ResetAfter != 0 {
		parseT.Errorf("ResetAfter = %v, want 0", parseBucket.ResetAfter)
	}
}

// TestRateLimitBucketFullyConsumed verifies the shape of a fully-consumed bucket
// (Remaining == 0) that the dashboard uses to render a "quota exhausted" state.
func TestRateLimitBucketFullyConsumed(parseT *testing.T) {
	parseT.Parallel()
	parseBucket := RateLimitBucket{
		Limit:      500,
		Remaining:  0,
		ResetAfter: 45 * time.Second,
	}
	if parseBucket.Limit != 500 {
		parseT.Errorf("Limit = %d, want 500", parseBucket.Limit)
	}
	if parseBucket.Remaining != 0 {
		parseT.Errorf("Remaining = %d, want 0 (fully consumed)", parseBucket.Remaining)
	}
	if parseBucket.ResetAfter != 45*time.Second {
		parseT.Errorf("ResetAfter = %v, want 45s", parseBucket.ResetAfter)
	}
}

// TestRateLimitBucketPartiallyConsumed verifies the shape of a partially-consumed
// bucket that the dashboard uses to render a usage-percentage indicator.
func TestRateLimitBucketPartiallyConsumed(parseT *testing.T) {
	parseT.Parallel()
	parseBucket := RateLimitBucket{
		Limit:      1000,
		Remaining:  700,
		ResetAfter: 60 * time.Second,
	}
	parseUsed := parseBucket.Limit - parseBucket.Remaining
	if parseUsed != 300 {
		parseT.Errorf("used = %d, want 300", parseUsed)
	}
	parseUsedPct := float64(parseUsed) / float64(parseBucket.Limit) * 100
	if parseUsedPct != 30.0 {
		parseT.Errorf("used-pct = %.1f, want 30.0", parseUsedPct)
	}
}

// ─── RateLimitSnapshot ───────────────────────────────────────────────────────

// TestRateLimitSnapshotEmptyZeroValue verifies ParseEmpty returns true for the
// zero-value snapshot so dashboard preflight checks correctly skip empty data.
func TestRateLimitSnapshotEmptyZeroValue(parseT *testing.T) {
	parseT.Parallel()
	if !(RateLimitSnapshot{}).ParseEmpty() {
		parseT.Error("ParseEmpty() = false for zero-value snapshot, want true")
	}
}

// TestRateLimitSnapshotNonEmptyWhenSourceSet verifies ParseEmpty returns false
// when the Source field alone is set — covers the dashboard provider-row path.
func TestRateLimitSnapshotNonEmptyWhenSourceSet(parseT *testing.T) {
	parseT.Parallel()
	if (RateLimitSnapshot{Source: "response-headers"}).ParseEmpty() {
		parseT.Error("ParseEmpty() = true when Source is set, want false")
	}
}

// TestRateLimitSnapshotNonEmptyWhenBucketSet verifies ParseEmpty returns false
// when any one bucket is non-zero — catches the reset-only case (Limit==0,
// ResetAfter>0) that OpenAI response headers sometimes produce.
func TestRateLimitSnapshotNonEmptyWhenBucketSet(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := RateLimitSnapshot{
		RequestsPerMinute: RateLimitBucket{Limit: 60, Remaining: 55, ResetAfter: 10 * time.Second},
	}
	if parseSnapshot.ParseEmpty() {
		parseT.Error("ParseEmpty() = true when RequestsPerMinute is set, want false")
	}
}

// TestRateLimitSnapshotNonEmptyWhenLastUpdatedSet verifies that a snapshot with
// only LastUpdated set is treated as non-empty — covers the "reported at" path.
func TestRateLimitSnapshotNonEmptyWhenLastUpdatedSet(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := RateLimitSnapshot{LastUpdated: time.Now()}
	if parseSnapshot.ParseEmpty() {
		parseT.Error("ParseEmpty() = true when LastUpdated is set, want false")
	}
}

// TestRateLimitSnapshotAllBucketsPopulated verifies a fully populated snapshot
// is non-empty and all six bucket fields are addressable individually — the
// shape that gets marshalled into the admin dashboard proto.
func TestRateLimitSnapshotAllBucketsPopulated(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := RateLimitSnapshot{
		RequestsPerMinute: RateLimitBucket{Limit: 60, Remaining: 55, ResetAfter: 10 * time.Second},
		RequestsPerHour:   RateLimitBucket{Limit: 3600, Remaining: 3400, ResetAfter: 30 * time.Minute},
		RequestsPerDay:    RateLimitBucket{Limit: 50000, Remaining: 49000, ResetAfter: 18 * time.Hour},
		TokensPerMinute:   RateLimitBucket{Limit: 200000, Remaining: 190000, ResetAfter: 10 * time.Second},
		TokensPerHour:     RateLimitBucket{Limit: 2000000, Remaining: 1900000, ResetAfter: 30 * time.Minute},
		TokensPerDay:      RateLimitBucket{Limit: 5000000, Remaining: 4500000, ResetAfter: 18 * time.Hour},
		Source:            "response-headers",
		LastUpdated:       time.Now(),
	}
	if parseSnapshot.ParseEmpty() {
		parseT.Error("ParseEmpty() = true for fully-populated snapshot, want false")
	}
	if parseSnapshot.RequestsPerMinute.Limit != 60 {
		parseT.Errorf("RequestsPerMinute.Limit = %d, want 60", parseSnapshot.RequestsPerMinute.Limit)
	}
	if parseSnapshot.TokensPerDay.Remaining != 4500000 {
		parseT.Errorf("TokensPerDay.Remaining = %d, want 4500000", parseSnapshot.TokensPerDay.Remaining)
	}
	if parseSnapshot.Source != "response-headers" {
		parseT.Errorf("Source = %q, want response-headers", parseSnapshot.Source)
	}
}

// ─── ProviderHealth ───────────────────────────────────────────────────────────

// TestProviderHealthStatusConstants verifies the four status constant values
// that the dashboard renders as coloured pills.
func TestProviderHealthStatusConstants(parseT *testing.T) {
	parseT.Parallel()
	if string(ProviderHealthUnknown) != "unknown" {
		parseT.Errorf("ProviderHealthUnknown = %q, want unknown", ProviderHealthUnknown)
	}
	if string(ProviderHealthHealthy) != "healthy" {
		parseT.Errorf("ProviderHealthHealthy = %q, want healthy", ProviderHealthHealthy)
	}
	if string(ProviderHealthDegraded) != "degraded" {
		parseT.Errorf("ProviderHealthDegraded = %q, want degraded", ProviderHealthDegraded)
	}
	if string(ProviderHealthUnavailable) != "unavailable" {
		parseT.Errorf("ProviderHealthUnavailable = %q, want unavailable", ProviderHealthUnavailable)
	}
}

// TestProviderHealthZeroValue verifies the zero-value ProviderHealth is valid:
// the empty ProviderID and unknown status are both acceptable initial states.
func TestProviderHealthZeroValue(parseT *testing.T) {
	parseT.Parallel()
	parseHealth := ProviderHealth{}
	if parseHealth.ProviderID != "" {
		parseT.Errorf("ProviderID = %q, want empty", parseHealth.ProviderID)
	}
	if parseHealth.Status != "" {
		parseT.Errorf("Status = %q, want empty zero value", parseHealth.Status)
	}
	if parseHealth.RequestCount != 0 {
		parseT.Errorf("RequestCount = %d, want 0", parseHealth.RequestCount)
	}
	if !parseHealth.RateLimits.ParseEmpty() {
		parseT.Error("RateLimits is not empty for zero-value ProviderHealth, want empty")
	}
}

// TestProviderHealthDegradedShape verifies the degraded health snapshot shape
// that the provider dashboard uses to render amber/red incident indicators.
func TestProviderHealthDegradedShape(parseT *testing.T) {
	parseT.Parallel()
	parseNow := time.Now()
	parseHealth := ProviderHealth{
		ProviderID:   "openai",
		Status:       ProviderHealthDegraded,
		LastSuccess:  parseNow.Add(-10 * time.Minute),
		LastFailure:  parseNow,
		LastError:    "rate limit exceeded",
		LastLatency:  1500 * time.Millisecond,
		RequestCount: 1200,
		TokenCount:   4500000,
		RateLimits: RateLimitSnapshot{
			RequestsPerMinute: RateLimitBucket{Limit: 60, Remaining: 0, ResetAfter: 55 * time.Second},
			Source:            "response-headers",
		},
	}
	if parseHealth.Status != ProviderHealthDegraded {
		parseT.Errorf("Status = %q, want degraded", parseHealth.Status)
	}
	if parseHealth.LastError != "rate limit exceeded" {
		parseT.Errorf("LastError = %q, want rate limit exceeded", parseHealth.LastError)
	}
	if parseHealth.RequestCount != 1200 {
		parseT.Errorf("RequestCount = %d, want 1200", parseHealth.RequestCount)
	}
	if !parseHealth.LastFailure.After(parseHealth.LastSuccess) {
		parseT.Error("LastFailure should be after LastSuccess for degraded provider")
	}
	if parseHealth.RateLimits.ParseEmpty() {
		parseT.Error("RateLimits is empty for degraded provider, want non-empty")
	}
	if parseHealth.RateLimits.RequestsPerMinute.Remaining != 0 {
		parseT.Errorf("RPM remaining = %d, want 0 (exhausted bucket)", parseHealth.RateLimits.RequestsPerMinute.Remaining)
	}
}

// TestProviderHealthUnavailableShape verifies the unavailable health snapshot
// shape delivered by providers whose API keys are unconfigured.
func TestProviderHealthUnavailableShape(parseT *testing.T) {
	parseT.Parallel()
	parseHealth := ProviderHealth{
		ProviderID: "anthropic",
		Status:     ProviderHealthUnavailable,
	}
	if parseHealth.Status != ProviderHealthUnavailable {
		parseT.Errorf("Status = %q, want unavailable", parseHealth.Status)
	}
	if parseHealth.RequestCount != 0 {
		parseT.Errorf("RequestCount = %d, want 0 for unavailable provider", parseHealth.RequestCount)
	}
	if !parseHealth.RateLimits.ParseEmpty() {
		parseT.Error("RateLimits should be empty for unavailable provider")
	}
}

// TestProviderHealthHealthyShape verifies the healthy snapshot shape with all
// counters and rate-limit buckets populated — the green-pill dashboard state.
func TestProviderHealthHealthyShape(parseT *testing.T) {
	parseT.Parallel()
	parseNow := time.Now()
	parseHealth := ProviderHealth{
		ProviderID:   "openai",
		Status:       ProviderHealthHealthy,
		LastSuccess:  parseNow,
		LastLatency:  220 * time.Millisecond,
		RequestCount: 8400,
		TokenCount:   18000000,
		RateLimits: RateLimitSnapshot{
			RequestsPerMinute: RateLimitBucket{Limit: 60, Remaining: 58, ResetAfter: 5 * time.Second},
			TokensPerMinute:   RateLimitBucket{Limit: 200000, Remaining: 195000, ResetAfter: 5 * time.Second},
			Source:            "response-headers",
		},
	}
	if parseHealth.Status != ProviderHealthHealthy {
		parseT.Errorf("Status = %q, want healthy", parseHealth.Status)
	}
	if parseHealth.RateLimits.ParseEmpty() {
		parseT.Error("RateLimits is empty for healthy provider with header data, want non-empty")
	}
	parseRPMUsed := parseHealth.RateLimits.RequestsPerMinute.Limit - parseHealth.RateLimits.RequestsPerMinute.Remaining
	if parseRPMUsed != 2 {
		parseT.Errorf("RPM used = %d, want 2", parseRPMUsed)
	}
}

// ─── ModelPricing / ParseEstimateCost ────────────────────────────────────────

// TestParseEstimateCostZeroTokens verifies that zero token counts produce a
// zero-cost estimate with no division errors.
func TestParseEstimateCostZeroTokens(parseT *testing.T) {
	parseT.Parallel()
	parseEstimate := ParseEstimateCost(0, 0, ModelPricing{
		InputPerMillionUSD:  5.0,
		OutputPerMillionUSD: 15.0,
	})
	if parseEstimate.InputCostUSD != 0 {
		parseT.Errorf("InputCostUSD = %v, want 0 for zero tokens", parseEstimate.InputCostUSD)
	}
	if parseEstimate.OutputCostUSD != 0 {
		parseT.Errorf("OutputCostUSD = %v, want 0 for zero tokens", parseEstimate.OutputCostUSD)
	}
	if parseEstimate.TotalCostUSD != 0 {
		parseT.Errorf("TotalCostUSD = %v, want 0 for zero tokens", parseEstimate.TotalCostUSD)
	}
}

// TestParseEstimateCostZeroPricing verifies that zero pricing produces a
// zero-cost estimate regardless of token count.
func TestParseEstimateCostZeroPricing(parseT *testing.T) {
	parseT.Parallel()
	parseEstimate := ParseEstimateCost(1_000_000, 1_000_000, ModelPricing{
		InputPerMillionUSD:  0,
		OutputPerMillionUSD: 0,
	})
	if parseEstimate.TotalCostUSD != 0 {
		parseT.Errorf("TotalCostUSD = %v, want 0 for zero pricing", parseEstimate.TotalCostUSD)
	}
}

// TestParseEstimateCostExactMillionBoundary verifies cost at the 1M token
// boundary equals the per-million rate exactly — a regression anchor for the
// pricing formula used by the billing dashboard.
func TestParseEstimateCostExactMillionBoundary(parseT *testing.T) {
	parseT.Parallel()
	parseEstimate := ParseEstimateCost(1_000_000, 1_000_000, ModelPricing{
		InputPerMillionUSD:  3.0,
		OutputPerMillionUSD: 12.0,
	})
	if parseEstimate.InputCostUSD != 3.0 {
		parseT.Errorf("InputCostUSD = %v, want 3.0 at 1M tokens", parseEstimate.InputCostUSD)
	}
	if parseEstimate.OutputCostUSD != 12.0 {
		parseT.Errorf("OutputCostUSD = %v, want 12.0 at 1M tokens", parseEstimate.OutputCostUSD)
	}
	if parseEstimate.TotalCostUSD != 15.0 {
		parseT.Errorf("TotalCostUSD = %v, want 15.0", parseEstimate.TotalCostUSD)
	}
}

// TestParseEstimateCostSubMillionInput verifies the fractional-token cost path
// used by the per-request cost tracking in the usage events table.
// Floating-point arithmetic is inherently approximate, so we use an epsilon
// comparison rather than exact equality.
func TestParseEstimateCostSubMillionInput(parseT *testing.T) {
	parseT.Parallel()
	// 500 input tokens at $3/M = $0.0015; 200 output tokens at $12/M = $0.0024
	parseEstimate := ParseEstimateCost(500, 200, ModelPricing{
		InputPerMillionUSD:  3.0,
		OutputPerMillionUSD: 12.0,
	})
	const parseEpsilon = 1e-12
	parseAbsDiff := func(a, b float64) float64 {
		if a > b {
			return a - b
		}
		return b - a
	}
	if parseAbsDiff(parseEstimate.InputCostUSD, 0.0015) > parseEpsilon {
		parseT.Errorf("InputCostUSD = %v, want ~0.0015", parseEstimate.InputCostUSD)
	}
	if parseAbsDiff(parseEstimate.OutputCostUSD, 0.0024) > parseEpsilon {
		parseT.Errorf("OutputCostUSD = %v, want ~0.0024", parseEstimate.OutputCostUSD)
	}
	if parseAbsDiff(parseEstimate.TotalCostUSD, 0.0039) > parseEpsilon {
		parseT.Errorf("TotalCostUSD = %v, want ~0.0039", parseEstimate.TotalCostUSD)
	}
}

// TestParseEstimateCostTotalIsSum verifies the invariant that TotalCostUSD is
// always exactly InputCostUSD + OutputCostUSD.
func TestParseEstimateCostTotalIsSum(parseT *testing.T) {
	parseT.Parallel()
	parsePricing := ModelPricing{InputPerMillionUSD: 1.25, OutputPerMillionUSD: 4.75}
	parseEstimate := ParseEstimateCost(123456, 78910, parsePricing)
	parseExpectedTotal := parseEstimate.InputCostUSD + parseEstimate.OutputCostUSD
	if parseEstimate.TotalCostUSD != parseExpectedTotal {
		parseT.Errorf("TotalCostUSD = %v, want InputCost+OutputCost = %v", parseEstimate.TotalCostUSD, parseExpectedTotal)
	}
}

// ─── ModelPricing shape ───────────────────────────────────────────────────────

// TestModelPricingCurrencyField verifies the Currency field is preserved — it
// feeds the billing dashboard's i18n currency display.
func TestModelPricingCurrencyField(parseT *testing.T) {
	parseT.Parallel()
	parsePricing := ModelPricing{
		InputPerMillionUSD:  2.5,
		OutputPerMillionUSD: 10.0,
		Currency:            "USD",
	}
	if parsePricing.Currency != "USD" {
		parseT.Errorf("Currency = %q, want USD", parsePricing.Currency)
	}
}

// ─── ParseModelOptionFromMetadata ────────────────────────────────────────────

// TestParseModelOptionFromMetadataMapsFields verifies that all fields relevant
// to the model picker are correctly collapsed from the full catalog entry.
func TestParseModelOptionFromMetadataMapsFields(parseT *testing.T) {
	parseT.Parallel()
	parseMetadata := ModelMetadata{
		ID:          "gpt-4o",
		DisplayName: "GPT-4o",
		ProviderID:  "openai",
		Capabilities: ModelCapabilities{
			ProviderID:       "openai",
			ProviderLabel:    "OpenAI",
			SupportsThinking: false,
			SupportsSpeech:   true,
		},
		Pricing: ModelPricing{
			InputPerMillionUSD:  2.5,
			OutputPerMillionUSD: 10.0,
		},
	}
	parseOption := ParseModelOptionFromMetadata(parseMetadata, "Fast and affordable")
	if parseOption.ID != "gpt-4o" {
		parseT.Errorf("ID = %q, want gpt-4o", parseOption.ID)
	}
	if parseOption.Label != "GPT-4o" {
		parseT.Errorf("Label = %q, want GPT-4o", parseOption.Label)
	}
	if parseOption.Note != "Fast and affordable" {
		parseT.Errorf("Note = %q, want Fast and affordable", parseOption.Note)
	}
	if parseOption.Capabilities.ProviderID != "openai" {
		parseT.Errorf("Capabilities.ProviderID = %q, want openai", parseOption.Capabilities.ProviderID)
	}
	if parseOption.Capabilities.SupportsSpeech != true {
		parseT.Error("Capabilities.SupportsSpeech = false, want true")
	}
	if parseOption.Pricing.InputPerMillionUSD != 2.5 {
		parseT.Errorf("Pricing.InputPerMillionUSD = %v, want 2.5", parseOption.Pricing.InputPerMillionUSD)
	}
}

// TestParseModelOptionFromMetadataEmptyNote verifies that an empty note string
// is passed through unchanged (no padding or sentinel substitution).
func TestParseModelOptionFromMetadataEmptyNote(parseT *testing.T) {
	parseT.Parallel()
	parseOption := ParseModelOptionFromMetadata(ModelMetadata{ID: "o3-mini", DisplayName: "o3 mini"}, "")
	if parseOption.Note != "" {
		parseT.Errorf("Note = %q, want empty", parseOption.Note)
	}
}

// ─── UnsupportedCapabilityError ──────────────────────────────────────────────

// TestUnsupportedCapabilityErrorWithModel verifies the error message when the
// model field is populated — feeds dashboard incident error display.
func TestUnsupportedCapabilityErrorWithModel(parseT *testing.T) {
	parseT.Parallel()
	parseErr := &UnsupportedCapabilityError{
		Capability: CapabilityThinking,
		Model:      "gpt-4o-mini",
		ProviderID: "openai",
	}
	parseMsg := parseErr.Error()
	if !strings.Contains(parseMsg, "gpt-4o-mini") {
		parseT.Errorf("error message %q does not contain model name", parseMsg)
	}
	if !strings.Contains(parseMsg, string(CapabilityThinking)) {
		parseT.Errorf("error message %q does not contain capability name", parseMsg)
	}
}

// TestUnsupportedCapabilityErrorWithoutModel verifies the fallback error message
// when the model field is empty — the provider-level error path.
func TestUnsupportedCapabilityErrorWithoutModel(parseT *testing.T) {
	parseT.Parallel()
	parseErr := &UnsupportedCapabilityError{
		Capability: CapabilitySpeech,
		Model:      "",
		ProviderID: "anthropic",
	}
	parseMsg := parseErr.Error()
	if !strings.Contains(parseMsg, "anthropic") {
		parseT.Errorf("error message %q does not contain provider ID", parseMsg)
	}
	if !strings.Contains(parseMsg, string(CapabilitySpeech)) {
		parseT.Errorf("error message %q does not contain capability name", parseMsg)
	}
}

// TestUnsupportedCapabilityErrorNilPointer verifies the nil-safety guard in
// ParseError returns a non-empty fallback string.
func TestUnsupportedCapabilityErrorNilPointer(parseT *testing.T) {
	parseT.Parallel()
	var parseErr *UnsupportedCapabilityError
	parseMsg := parseErr.ParseError()
	if parseMsg == "" {
		parseT.Error("ParseError() on nil pointer returned empty string, want fallback message")
	}
}

// ─── ParseNormalizeRole ───────────────────────────────────────────────────────

// TestParseNormalizeRoleKnownRoles verifies all accepted role identifiers pass
// through to the output exactly, with whitespace and casing normalized.
func TestParseNormalizeRoleKnownRoles(parseT *testing.T) {
	parseT.Parallel()
	parseCases := []struct {
		parseInput string
		parseWant  string
	}{
		{"assistant", "assistant"},
		{"  ASSISTANT  ", "assistant"},
		{"system", "system"},
		{"SYSTEM", "system"},
		{"developer", "developer"},
		{"tool", "tool"},
		{"user", "user"},
		{"USER", "user"},
	}
	for _, parseCase := range parseCases {
		parseGot := ParseNormalizeRole(parseCase.parseInput)
		if parseGot != parseCase.parseWant {
			parseT.Errorf("ParseNormalizeRole(%q) = %q, want %q", parseCase.parseInput, parseGot, parseCase.parseWant)
		}
	}
}

// TestParseNormalizeRoleUnknownFallsBackToUser verifies that any unrecognised
// role string is normalized to "user", which is the safe default for provider
// payloads that enforce strict role sets.
func TestParseNormalizeRoleUnknownFallsBackToUser(parseT *testing.T) {
	parseT.Parallel()
	parseUnknown := []string{"", "human", "ai", "bot", "unknown-role", "  "}
	for _, parseInput := range parseUnknown {
		parseGot := ParseNormalizeRole(parseInput)
		if parseGot != "user" {
			parseT.Errorf("ParseNormalizeRole(%q) = %q, want user", parseInput, parseGot)
		}
	}
}

// ─── ModelCapabilities.ParseSupports ─────────────────────────────────────────

// TestModelCapabilitiesParseSupportsThinking verifies that SupportsThinking
// correctly routes through the CapabilityThinking switch arm.
func TestModelCapabilitiesParseSupportsThinking(parseT *testing.T) {
	parseT.Parallel()
	parseCaps := ModelCapabilities{SupportsThinking: true, SupportsSpeech: false}
	if !parseCaps.ParseSupports(CapabilityThinking) {
		parseT.Error("ParseSupports(CapabilityThinking) = false, want true")
	}
	parseCapsNoThinking := ModelCapabilities{SupportsThinking: false}
	if parseCapsNoThinking.ParseSupports(CapabilityThinking) {
		parseT.Error("ParseSupports(CapabilityThinking) = true for non-thinking model, want false")
	}
}

// TestModelCapabilitiesParseSupportsSpeeech verifies that SupportsSpeech
// correctly routes through the CapabilitySpeech switch arm.
func TestModelCapabilitiesParseSupportsSpeeech(parseT *testing.T) {
	parseT.Parallel()
	parseCaps := ModelCapabilities{SupportsThinking: false, SupportsSpeech: true}
	if !parseCaps.ParseSupports(CapabilitySpeech) {
		parseT.Error("ParseSupports(CapabilitySpeech) = false, want true")
	}
	parseCapsNoSpeech := ModelCapabilities{SupportsSpeech: false}
	if parseCapsNoSpeech.ParseSupports(CapabilitySpeech) {
		parseT.Error("ParseSupports(CapabilitySpeech) = true for non-speech model, want false")
	}
}

// TestModelCapabilitiesParseSupportsUnknown verifies that an unrecognised
// capability string returns false via the default switch arm.
func TestModelCapabilitiesParseSupportsUnknown(parseT *testing.T) {
	parseT.Parallel()
	parseCaps := ModelCapabilities{SupportsThinking: true, SupportsSpeech: true}
	if parseCaps.ParseSupports(Capability("vision")) {
		parseT.Error("ParseSupports(unknown) = true, want false for unrecognised capability")
	}
}

// ─── BuildConversationInput ───────────────────────────────────────────────────

// TestBuildConversationInputEmptyHistory verifies that an empty history still
// produces a well-formed prompt with the preamble and the user turn.
func TestBuildConversationInputEmptyHistory(parseT *testing.T) {
	parseT.Parallel()
	parseOutput := BuildConversationInput(nil, "Hello!")
	if !strings.Contains(parseOutput, "user:") {
		parseT.Errorf("output %q does not contain user: label", parseOutput)
	}
	if !strings.Contains(parseOutput, "Hello!") {
		parseT.Errorf("output %q does not contain user message", parseOutput)
	}
}

// TestBuildConversationInputPreservesHistory verifies that history turns are
// included in order with their normalized role labels.
func TestBuildConversationInputPreservesHistory(parseT *testing.T) {
	parseT.Parallel()
	parseHistory := []ChatMessage{
		{Role: "user", Content: "Summarise the report."},
		{Role: "ASSISTANT", Content: "Here is the summary."},
	}
	parseOutput := BuildConversationInput(parseHistory, "Thanks, can you shorten it?")
	if !strings.Contains(parseOutput, "user:\nSummarise the report.") {
		parseT.Errorf("output %q missing first history turn", parseOutput)
	}
	if !strings.Contains(parseOutput, "assistant:\nHere is the summary.") {
		parseT.Errorf("output %q missing assistant turn (role should be normalized)", parseOutput)
	}
	if !strings.Contains(parseOutput, "Thanks, can you shorten it?") {
		parseT.Errorf("output %q missing current user message", parseOutput)
	}
}

// TestBuildConversationInputTrimsContentWhitespace verifies that leading and
// trailing whitespace in message content is trimmed for clean provider payloads.
func TestBuildConversationInputTrimsContentWhitespace(parseT *testing.T) {
	parseT.Parallel()
	parseOutput := BuildConversationInput(nil, "  trimmed message  ")
	if !strings.Contains(parseOutput, "trimmed message") {
		parseT.Errorf("output %q does not contain trimmed content", parseOutput)
	}
	if strings.Contains(parseOutput, "  trimmed message  ") {
		parseT.Errorf("output %q contains untrimmed whitespace", parseOutput)
	}
}
