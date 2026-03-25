package provider

import (
	"strings"
	"testing"
)

func TestRegistryHelpersExposeAggregatedMetadata(parseT *testing.T) {
	parseCapA := ModelCapabilities{ProviderID: "stub-a", ProviderLabel: "Stub A", SupportsThinking: true}
	parseCapB := ModelCapabilities{ProviderID: "stub-b", ProviderLabel: "Stub B", SupportsSpeech: true}
	parseProviderA := &stubProvider{
		id:           "stub-a",
		available:    true,
		defaultModel: "model-a",
		models:       map[string]ModelCapabilities{"model-a": parseCapA},
		options:      []ModelOption{{ID: "model-a", Label: "Model A", Capabilities: parseCapA}},
		info:         ProviderInfo{ID: "stub-a", Label: "Stub A", Available: true},
		metadata: map[string]ModelMetadata{
			"model-a": {ID: "model-a", ProviderID: "stub-a", Pricing: ModelPricing{InputPerMillionUSD: 1.5}},
		},
	}
	parseProviderB := &stubProvider{
		id:           "stub-b",
		available:    true,
		defaultModel: "model-b",
		models:       map[string]ModelCapabilities{"model-b": parseCapB},
		options:      []ModelOption{{ID: "model-b", Label: "Model B", Capabilities: parseCapB}},
		info:         ProviderInfo{ID: "stub-b", Label: "Stub B", Available: true},
		metadata: map[string]ModelMetadata{
			"model-b": {ID: "model-b", ProviderID: "stub-b", Pricing: ModelPricing{OutputPerMillionUSD: 2.5}},
		},
	}
	parseRegistry := ParseNewRegistry(nil, &stubProvider{id: "unavailable", available: false}, parseProviderA, parseProviderB)

	if parseGot := parseRegistry.ParseDefaultModel(); parseGot != "model-a" {
		parseT.Fatalf("DefaultModel() = %q, want model-a", parseGot)
	}
	if parseOptions := parseRegistry.ParseModelOptions(); len(parseOptions) != 2 {
		parseT.Fatalf("ModelOptions() len = %d, want 2", len(parseOptions))
	}
	if parseInfos := parseRegistry.ParseProviderInfos(); len(parseInfos) != 2 || parseInfos[0].ParseID != "stub-a" || parseInfos[1].ParseID != "stub-b" {
		parseT.Fatalf("ProviderInfos() = %+v, want both providers", parseInfos)
	}
	if parseHealth := parseRegistry.ParseHealthSnapshots(); len(parseHealth) != 2 || parseHealth[0].ProviderID != "stub-a" || parseHealth[1].ProviderID != "stub-b" {
		parseT.Fatalf("HealthSnapshots() = %+v, want both providers", parseHealth)
	}

	parsePricing, parseResolvedModel, parseErr := parseRegistry.ParsePricing("model-b")
	if parseErr != nil {
		parseT.Fatalf("Pricing(): %v", parseErr)
	}
	if parseResolvedModel != "model-b" || parsePricing.OutputPerMillionUSD != 2.5 {
		parseT.Fatalf("Pricing() = %+v resolved=%q", parsePricing, parseResolvedModel)
	}

	parseCapabilities, parseResolvedModel, parseErr := parseRegistry.ParseCapabilities("model-a")
	if parseErr != nil {
		parseT.Fatalf("Capabilities(): %v", parseErr)
	}
	if parseResolvedModel != "model-a" || !parseCapabilities.SupportsThinking {
		parseT.Fatalf("Capabilities() = %+v resolved=%q", parseCapabilities, parseResolvedModel)
	}

	parseMetadata, parseResolvedModel, parseErr := parseRegistry.ParseModelMetadata("model-a")
	if parseErr != nil {
		parseT.Fatalf("ModelMetadata(): %v", parseErr)
	}
	if parseMetadata.ProviderID != "stub-a" || parseResolvedModel != "model-a" {
		parseT.Fatalf("ModelMetadata() = %+v resolved=%q", parseMetadata, parseResolvedModel)
	}
}

func TestRegistryRequireCapabilityAndErrorStrings(parseT *testing.T) {
	parseRegistry := ParseNewRegistry(&stubProvider{
		id:           "speechy",
		available:    true,
		defaultModel: "model-s",
		models: map[string]ModelCapabilities{
			"model-s": {ProviderID: "speechy", ProviderLabel: "Speechy", SupportsSpeech: true},
		},
	})

	parseProvider, parseResolvedModel, parseCaps, parseErr := parseRegistry.ParseRequireCapability("model-s", CapabilitySpeech)
	if parseErr != nil {
		parseT.Fatalf("RequireCapability(): %v", parseErr)
	}
	if parseProvider.ParseID() != "speechy" || parseResolvedModel != "model-s" || !parseCaps.SupportsSpeech {
		parseT.Fatalf("unexpected RequireCapability result: provider=%v model=%q caps=%+v", parseProvider, parseResolvedModel, parseCaps)
	}

	if !parseCaps.ParseSupports(CapabilitySpeech) || parseCaps.ParseSupports(CapabilityThinking) || parseCaps.ParseSupports(Capability("vision")) {
		parseT.Fatalf("unexpected Supports() behavior: %+v", parseCaps)
	}

	parseErrText := (&UnsupportedCapabilityError{Capability: CapabilitySpeech, ProviderID: "speechy"}).ParseError()
	if !strings.Contains(parseErrText, `provider "speechy" does not support speech`) {
		parseT.Fatalf("unexpected provider capability error text: %q", parseErrText)
	}
}
