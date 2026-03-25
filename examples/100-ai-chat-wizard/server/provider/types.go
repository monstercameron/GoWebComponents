package provider

// ProviderInfo describes one configured provider in a provider-agnostic way.
//
// This is the static/operator-facing view used by config, onboarding, and
// dashboard surfaces. Runtime health and rate-limit state are kept separate.
type ProviderInfo struct {
	ID                 string
	Label              string
	BaseURL            string
	AuthConfigured     bool
	Available          bool
	StreamingSupported bool
	ReasoningSupported bool
	ToolUseSupported   bool
	Notes              []string
}

// ModelMetadata is the richer catalog entry for one provider-backed model.
//
// ModelOption remains the compact picker shape used by the current client.
// ModelMetadata adds pricing, limits, compatibility notes, and onboarding
// status so providers can plug into future dashboard/config surfaces cleanly.
type ModelMetadata struct {
	ID                        string
	DisplayName               string
	Description               string
	ProviderID                string
	ProviderLabel             string
	ProviderFamily            string
	Capabilities              ModelCapabilities
	StreamingSupported        bool
	ReasoningSupported        bool
	ToolUseSupported          bool
	StructuredOutputSupport   string
	MaxContextTokens          int64
	MaxOutputTokens           int64
	ThroughputTokensPerSecond float64
	OnboardingReady           bool
	CompatibilityNotes        []string
	Pricing                   ModelPricing
}

// ModelOptionFromMetadata collapses a rich catalog entry into the compact UI
// shape used by the current example picker.
func ParseModelOptionFromMetadata(parseMetadata ModelMetadata, parseNote string) ModelOption {
	return ModelOption{
		ID:           parseMetadata.ParseID,
		Label:        parseMetadata.DisplayName,
		Note:         parseNote,
		Capabilities: parseMetadata.ParseCapabilities,
		Pricing:      parseMetadata.ParsePricing,
	}
}
