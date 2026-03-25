package provider

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCatalogBuildsDefaultsAndCopiesOptions(t *testing.T) {
	normalized := normalizeCatalog("stub", "Stub", Catalog{
		Models: []ModelMetadata{
			{
				ID:          " model-a ",
				DisplayName: "Model A",
				Pricing:     ModelPricing{InputPerMillionUSD: 1.25},
			},
			{
				ID:            "model-b",
				DisplayName:   "Model B",
				ProviderLabel: "Custom Stub",
				Capabilities:  ModelCapabilities{ProviderID: "existing-provider", SupportsThinking: true},
			},
			{
				ID: "   ",
			},
		},
	})
	if normalized.DefaultModel != "model-a" || normalized.TitleModel != "model-a" {
		t.Fatalf("normalizeCatalog() defaults = %+v", normalized)
	}
	if len(normalized.Models) != 2 || len(normalized.Options) != 2 {
		t.Fatalf("normalizeCatalog() models/options = %+v", normalized)
	}
	if normalized.Models[0].ProviderID != "stub" || normalized.Models[0].ProviderLabel != "Stub" || normalized.Models[0].Capabilities.ProviderID != "stub" || normalized.Models[0].Capabilities.ProviderLabel != "Stub" {
		t.Fatalf("normalizeCatalog() first model normalization = %+v", normalized.Models[0])
	}
	if normalized.Models[1].ProviderLabel != "Custom Stub" || normalized.Models[1].Capabilities.ProviderID != "existing-provider" || normalized.Models[1].Capabilities.ProviderLabel != "Custom Stub" {
		t.Fatalf("normalizeCatalog() second model normalization = %+v", normalized.Models[1])
	}
	if normalized.Options[0].ID != "model-a" || normalized.Options[0].Label != "Model A" || normalized.Options[0].Pricing.InputPerMillionUSD != 1.25 {
		t.Fatalf("normalizeCatalog() derived options = %+v", normalized.Options)
	}

	withExplicitOptions := normalizeCatalog("stub", "Stub", Catalog{
		Models: []ModelMetadata{{ID: "model-a", DisplayName: "Model A"}},
		Options: []ModelOption{
			{ID: " model-x ", Label: "Model X"},
			{ID: " ", Label: "Skip"},
		},
		DefaultModel: " model-a ",
		TitleModel:   " model-x ",
	})
	if withExplicitOptions.DefaultModel != "model-a" || withExplicitOptions.TitleModel != "model-x" {
		t.Fatalf("normalizeCatalog(explicit defaults) = %+v", withExplicitOptions)
	}
	if len(withExplicitOptions.Options) != 1 || withExplicitOptions.Options[0].ID != "model-x" || withExplicitOptions.Options[0].Capabilities.ProviderID != "stub" || withExplicitOptions.Options[0].Capabilities.ProviderLabel != "Stub" {
		t.Fatalf("normalizeCatalog(explicit options) = %+v", withExplicitOptions.Options)
	}
}

func TestCatalogAndRegistryAdditionalErrorBranches(t *testing.T) {
	catalog := Catalog{
		Models: []ModelMetadata{{ID: "model-a", DisplayName: "Model A"}},
		Options: []ModelOption{
			{ID: "model-a", Label: "Model A"},
		},
	}
	if !catalog.SupportsModel(" MODEL-A ") || catalog.SupportsModel("missing") {
		t.Fatalf("Catalog.SupportsModel() behavior mismatch")
	}
	options := catalog.ModelOptions()
	options[0].Label = "mutated"
	if catalog.Options[0].Label != "Model A" {
		t.Fatalf("Catalog.ModelOptions() did not clone options: %+v", catalog.Options)
	}

	if got := (*Registry)(nil).DefaultModel(); got != "" {
		t.Fatalf("(*Registry)(nil).DefaultModel() = %q, want empty", got)
	}
	if got := (*Registry)(nil).ModelOptions(); got != nil {
		t.Fatalf("(*Registry)(nil).ModelOptions() = %#v, want nil", got)
	}
	if got := (*Registry)(nil).ProviderInfos(); got != nil {
		t.Fatalf("(*Registry)(nil).ProviderInfos() = %#v, want nil", got)
	}
	if got := (*Registry)(nil).HealthSnapshots(); got != nil {
		t.Fatalf("(*Registry)(nil).HealthSnapshots() = %#v, want nil", got)
	}

	registry := NewRegistry(&stubProvider{
		id:           "meta",
		available:    true,
		defaultModel: "model-a",
		models:       map[string]ModelCapabilities{"model-a": {ProviderID: "meta", ProviderLabel: "Meta"}},
		metadata:     map[string]ModelMetadata{},
	})
	if _, resolved, err := registry.ModelMetadata("model-a"); err == nil || resolved != "model-a" || !strings.Contains(err.Error(), "metadata unavailable") {
		t.Fatalf("Registry.ModelMetadata() error = %v resolved=%q, want metadata unavailable", err, resolved)
	}
	if _, _, err := registry.Pricing("model-a"); err == nil || !strings.Contains(err.Error(), "metadata unavailable") {
		t.Fatalf("Registry.Pricing() error = %v, want metadata unavailable", err)
	}

	var nilErr *NormalizedError
	if err := nilErr.Unwrap(); err != nil {
		t.Fatalf("(*NormalizedError)(nil).Unwrap() = %v, want nil", err)
	}
	rootErr := errors.New("boom")
	withProviderNoModel := (&NormalizedError{ProviderID: "openai", Message: "failed", Err: rootErr}).Error()
	if !strings.Contains(withProviderNoModel, "openai provider error: failed") {
		t.Fatalf("NormalizedError without model = %q", withProviderNoModel)
	}
}
