package provider

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCatalogBuildsDefaultsAndCopiesOptions(parseT *testing.T) {
	parseNormalized := parseNormalizeCatalog("stub", "Stub", Catalog{
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
	if parseNormalized.DefaultModel != "model-a" || parseNormalized.TitleModel != "model-a" {
		parseT.Fatalf("normalizeCatalog() defaults = %+v", parseNormalized)
	}
	if len(parseNormalized.Models) != 2 || len(parseNormalized.Options) != 2 {
		parseT.Fatalf("normalizeCatalog() models/options = %+v", parseNormalized)
	}
	if parseNormalized.Models[0].ProviderID != "stub" || parseNormalized.Models[0].ProviderLabel != "Stub" || parseNormalized.Models[0].Capabilities.ProviderID != "stub" || parseNormalized.Models[0].Capabilities.ProviderLabel != "Stub" {
		parseT.Fatalf("normalizeCatalog() first model normalization = %+v", parseNormalized.Models[0])
	}
	if parseNormalized.Models[1].ProviderLabel != "Custom Stub" || parseNormalized.Models[1].Capabilities.ProviderID != "existing-provider" || parseNormalized.Models[1].Capabilities.ProviderLabel != "Custom Stub" {
		parseT.Fatalf("normalizeCatalog() second model normalization = %+v", parseNormalized.Models[1])
	}
	if parseNormalized.Options[0].ID != "model-a" || parseNormalized.Options[0].Label != "Model A" || parseNormalized.Options[0].Pricing.InputPerMillionUSD != 1.25 {
		parseT.Fatalf("normalizeCatalog() derived options = %+v", parseNormalized.Options)
	}

	parseWithExplicitOptions := parseNormalizeCatalog("stub", "Stub", Catalog{
		Models: []ModelMetadata{{ID: "model-a", DisplayName: "Model A"}},
		Options: []ModelOption{
			{ID: " model-x ", Label: "Model X"},
			{ID: " ", Label: "Skip"},
		},
		DefaultModel: " model-a ",
		TitleModel:   " model-x ",
	})
	if parseWithExplicitOptions.DefaultModel != "model-a" || parseWithExplicitOptions.TitleModel != "model-x" {
		parseT.Fatalf("normalizeCatalog(explicit defaults) = %+v", parseWithExplicitOptions)
	}
	if len(parseWithExplicitOptions.Options) != 1 || parseWithExplicitOptions.Options[0].ID != "model-x" || parseWithExplicitOptions.Options[0].Capabilities.ProviderID != "stub" || parseWithExplicitOptions.Options[0].Capabilities.ProviderLabel != "Stub" {
		parseT.Fatalf("normalizeCatalog(explicit options) = %+v", parseWithExplicitOptions.Options)
	}
}

func TestCatalogAndRegistryAdditionalErrorBranches(parseT *testing.T) {
	parseCatalog := Catalog{
		Models: []ModelMetadata{{ID: "model-a", DisplayName: "Model A"}},
		Options: []ModelOption{
			{ID: "model-a", Label: "Model A"},
		},
	}
	if !parseCatalog.ParseSupportsModel(" MODEL-A ") || parseCatalog.ParseSupportsModel("missing") {
		parseT.Fatalf("Catalog.SupportsModel() behavior mismatch")
	}
	parseOptions := parseCatalog.ParseModelOptions()
	parseOptions[0].Label = "mutated"
	if parseCatalog.Options[0].Label != "Model A" {
		parseT.Fatalf("Catalog.ModelOptions() did not clone options: %+v", parseCatalog.Options)
	}

	if parseGot := (*Registry)(nil).ParseDefaultModel(); parseGot != "" {
		parseT.Fatalf("(*Registry)(nil).DefaultModel() = %q, want empty", parseGot)
	}
	if parseGot2 := (*Registry)(nil).ParseModelOptions(); parseGot2 != nil {
		parseT.Fatalf("(*Registry)(nil).ModelOptions() = %#v, want nil", parseGot2)
	}
	if parseGot3 := (*Registry)(nil).ParseProviderInfos(); parseGot3 != nil {
		parseT.Fatalf("(*Registry)(nil).ProviderInfos() = %#v, want nil", parseGot3)
	}
	if parseGot4 := (*Registry)(nil).ParseHealthSnapshots(); parseGot4 != nil {
		parseT.Fatalf("(*Registry)(nil).HealthSnapshots() = %#v, want nil", parseGot4)
	}

	parseRegistry := ParseNewRegistry(&stubProvider{
		id:           "meta",
		available:    true,
		defaultModel: "model-a",
		models:       map[string]ModelCapabilities{"model-a": {ProviderID: "meta", ProviderLabel: "Meta"}},
		metadata:     map[string]ModelMetadata{},
	})
	if _, parseResolved, parseErr := parseRegistry.ParseModelMetadata("model-a"); parseErr == nil || parseResolved != "model-a" || !strings.Contains(parseErr.Error(), "metadata unavailable") {
		parseT.Fatalf("Registry.ModelMetadata() error = %v resolved=%q, want metadata unavailable", parseErr, parseResolved)
	}
	if _, _, parseErr2 := parseRegistry.ParsePricing("model-a"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "metadata unavailable") {
		parseT.Fatalf("Registry.Pricing() error = %v, want metadata unavailable", parseErr2)
	}

	var parseNilErr *NormalizedError
	if parseErr3 := parseNilErr.ParseUnwrap(); parseErr3 != nil {
		parseT.Fatalf("(*NormalizedError)(nil).Unwrap() = %v, want nil", parseErr3)
	}
	parseRootErr := errors.New("boom")
	parseWithProviderNoModel := (&NormalizedError{ProviderID: "openai", Message: "failed", Err: parseRootErr}).Error()
	if !strings.Contains(parseWithProviderNoModel, "openai provider error: failed") {
		parseT.Fatalf("NormalizedError without model = %q", parseWithProviderNoModel)
	}
}
