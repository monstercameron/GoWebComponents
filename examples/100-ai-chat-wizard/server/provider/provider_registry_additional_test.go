package provider

import (
	"strings"
	"testing"
)

func TestRegistryHelpersExposeAggregatedMetadata(t *testing.T) {
	capA := ModelCapabilities{ProviderID: "stub-a", ProviderLabel: "Stub A", SupportsThinking: true}
	capB := ModelCapabilities{ProviderID: "stub-b", ProviderLabel: "Stub B", SupportsSpeech: true}
	providerA := &stubProvider{
		id:           "stub-a",
		available:    true,
		defaultModel: "model-a",
		models:       map[string]ModelCapabilities{"model-a": capA},
		options:      []ModelOption{{ID: "model-a", Label: "Model A", Capabilities: capA}},
		info:         ProviderInfo{ID: "stub-a", Label: "Stub A", Available: true},
		metadata: map[string]ModelMetadata{
			"model-a": {ID: "model-a", ProviderID: "stub-a", Pricing: ModelPricing{InputPerMillionUSD: 1.5}},
		},
	}
	providerB := &stubProvider{
		id:           "stub-b",
		available:    true,
		defaultModel: "model-b",
		models:       map[string]ModelCapabilities{"model-b": capB},
		options:      []ModelOption{{ID: "model-b", Label: "Model B", Capabilities: capB}},
		info:         ProviderInfo{ID: "stub-b", Label: "Stub B", Available: true},
		metadata: map[string]ModelMetadata{
			"model-b": {ID: "model-b", ProviderID: "stub-b", Pricing: ModelPricing{OutputPerMillionUSD: 2.5}},
		},
	}
	registry := NewRegistry(nil, &stubProvider{id: "unavailable", available: false}, providerA, providerB)

	if got := registry.DefaultModel(); got != "model-a" {
		t.Fatalf("DefaultModel() = %q, want model-a", got)
	}
	if options := registry.ModelOptions(); len(options) != 2 {
		t.Fatalf("ModelOptions() len = %d, want 2", len(options))
	}
	if infos := registry.ProviderInfos(); len(infos) != 2 || infos[0].ID != "stub-a" || infos[1].ID != "stub-b" {
		t.Fatalf("ProviderInfos() = %+v, want both providers", infos)
	}
	if health := registry.HealthSnapshots(); len(health) != 2 || health[0].ProviderID != "stub-a" || health[1].ProviderID != "stub-b" {
		t.Fatalf("HealthSnapshots() = %+v, want both providers", health)
	}

	pricing, resolvedModel, err := registry.Pricing("model-b")
	if err != nil {
		t.Fatalf("Pricing(): %v", err)
	}
	if resolvedModel != "model-b" || pricing.OutputPerMillionUSD != 2.5 {
		t.Fatalf("Pricing() = %+v resolved=%q", pricing, resolvedModel)
	}

	capabilities, resolvedModel, err := registry.Capabilities("model-a")
	if err != nil {
		t.Fatalf("Capabilities(): %v", err)
	}
	if resolvedModel != "model-a" || !capabilities.SupportsThinking {
		t.Fatalf("Capabilities() = %+v resolved=%q", capabilities, resolvedModel)
	}

	metadata, resolvedModel, err := registry.ModelMetadata("model-a")
	if err != nil {
		t.Fatalf("ModelMetadata(): %v", err)
	}
	if metadata.ProviderID != "stub-a" || resolvedModel != "model-a" {
		t.Fatalf("ModelMetadata() = %+v resolved=%q", metadata, resolvedModel)
	}
}

func TestRegistryRequireCapabilityAndErrorStrings(t *testing.T) {
	registry := NewRegistry(&stubProvider{
		id:           "speechy",
		available:    true,
		defaultModel: "model-s",
		models: map[string]ModelCapabilities{
			"model-s": {ProviderID: "speechy", ProviderLabel: "Speechy", SupportsSpeech: true},
		},
	})

	provider, resolvedModel, caps, err := registry.RequireCapability("model-s", CapabilitySpeech)
	if err != nil {
		t.Fatalf("RequireCapability(): %v", err)
	}
	if provider.ID() != "speechy" || resolvedModel != "model-s" || !caps.SupportsSpeech {
		t.Fatalf("unexpected RequireCapability result: provider=%v model=%q caps=%+v", provider, resolvedModel, caps)
	}

	if !caps.Supports(CapabilitySpeech) || caps.Supports(CapabilityThinking) || caps.Supports(Capability("vision")) {
		t.Fatalf("unexpected Supports() behavior: %+v", caps)
	}

	errText := (&UnsupportedCapabilityError{Capability: CapabilitySpeech, ProviderID: "speechy"}).Error()
	if !strings.Contains(errText, `provider "speechy" does not support speech`) {
		t.Fatalf("unexpected provider capability error text: %q", errText)
	}
}
