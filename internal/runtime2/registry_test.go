package runtime2

import "testing"

// TestParseRendererIDRejectsEmptyOrWhitespace verifies renderer IDs require content.
func TestParseRendererIDRejectsEmptyOrWhitespace(parseT *testing.T) {
	parseCases := []string{"", "   "}
	for _, parseCase := range parseCases {
		if _, parseErr := ParseRendererID(parseCase); parseErr == nil {
			parseT.Fatalf("expected ParseRendererID(%q) to fail", parseCase)
		}
	}
}

// TestParseRendererIDRoundTripsValidValue verifies valid renderer IDs survive parse and encode.
func TestParseRendererIDRoundTripsValidValue(parseT *testing.T) {
	parseRendererID, parseErr := ParseRendererID("dashboard.hot-panel")
	if parseErr != nil {
		parseT.Fatalf("ParseRendererID returned error: %v", parseErr)
	}
	if string(parseRendererID) != "dashboard.hot-panel" {
		parseT.Fatalf("expected renderer ID round trip, got %q", parseRendererID)
	}
}

// TestParseRegionInstanceIDRejectsEmptyValue verifies region instance IDs require content.
func TestParseRegionInstanceIDRejectsEmptyValue(parseT *testing.T) {
	if _, parseErr := ParseRegionInstanceID(""); parseErr == nil {
		parseT.Fatal("expected empty region instance ID to fail")
	}
}

// TestParseRegionInstanceIDSeparatesDistinctValues verifies instance IDs do not collide accidentally.
func TestParseRegionInstanceIDSeparatesDistinctValues(parseT *testing.T) {
	parseRegionIDA, parseErr := ParseRegionInstanceID("region-a")
	if parseErr != nil {
		parseT.Fatalf("ParseRegionInstanceID(region-a) returned error: %v", parseErr)
	}
	parseRegionIDB, parseErr := ParseRegionInstanceID("region-b")
	if parseErr != nil {
		parseT.Fatalf("ParseRegionInstanceID(region-b) returned error: %v", parseErr)
	}
	if parseRegionIDA == parseRegionIDB {
		parseT.Fatalf("expected distinct region IDs, got %q", parseRegionIDA)
	}
}

// TestRegisterRendererResolvesRegisteredRenderer verifies registry lookup succeeds after registration.
func TestRegisterRendererResolvesRegisteredRenderer(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, parseErr := ParseRendererID("dashboard.hot-panel")
	if parseErr != nil {
		parseT.Fatalf("ParseRendererID returned error: %v", parseErr)
	}
	parseRenderer := func() {}
	if parseErr := RegisterRenderer(parseRendererID, parseRenderer, RendererMetadata{}); parseErr != nil {
		parseT.Fatalf("RegisterRenderer returned error: %v", parseErr)
	}
	parseResolvedRenderer, parseMetadata, parseErr := ResolveRenderer(parseRendererID)
	if parseErr != nil {
		parseT.Fatalf("ResolveRenderer returned error: %v", parseErr)
	}
	if parseResolvedRenderer == nil {
		parseT.Fatal("expected resolved renderer")
	}
	if parseMetadata.PropSchemaVersion != "" || len(parseMetadata.FeatureFlags) != 0 {
		parseT.Fatalf("expected default metadata, got %+v", parseMetadata)
	}
}

// TestRegisterRendererRejectsDuplicateRegistration verifies duplicate IDs fail clearly.
func TestRegisterRendererRejectsDuplicateRegistration(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	if parseErr := RegisterRenderer(parseRendererID, func() {}, RendererMetadata{}); parseErr != nil {
		parseT.Fatalf("first RegisterRenderer returned error: %v", parseErr)
	}
	if parseErr := RegisterRenderer(parseRendererID, func() {}, RendererMetadata{}); parseErr == nil {
		parseT.Fatal("expected duplicate RegisterRenderer to fail")
	}
}

// TestResolveRendererRejectsUnregisteredRenderer verifies missing IDs fail clearly.
func TestResolveRendererRejectsUnregisteredRenderer(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.missing")
	if _, _, parseErr := ResolveRenderer(parseRendererID); parseErr == nil {
		parseT.Fatal("expected ResolveRenderer to fail for an unregistered renderer")
	}
}

// TestResetRendererRegistryClearsEntries verifies test reset clears prior registrations.
func TestResetRendererRegistryClearsEntries(parseT *testing.T) {
	ResetRendererRegistry()
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	if parseErr := RegisterRenderer(parseRendererID, func() {}, RendererMetadata{}); parseErr != nil {
		parseT.Fatalf("RegisterRenderer returned error: %v", parseErr)
	}
	ResetRendererRegistry()
	if _, _, parseErr := ResolveRenderer(parseRendererID); parseErr == nil {
		parseT.Fatal("expected reset registry to clear prior registration")
	}
}

// TestRegisterRendererReturnsMetadata verifies valid metadata is returned from the registry.
func TestRegisterRendererReturnsMetadata(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	parseMetadata := RendererMetadata{
		PropSchemaVersion: "props.v1",
		FeatureFlags:      []string{"display-only", "derived-state"},
	}
	if parseErr := RegisterRenderer(parseRendererID, func() {}, parseMetadata); parseErr != nil {
		parseT.Fatalf("RegisterRenderer returned error: %v", parseErr)
	}
	_, parseResolvedMetadata, parseErr := ResolveRenderer(parseRendererID)
	if parseErr != nil {
		parseT.Fatalf("ResolveRenderer returned error: %v", parseErr)
	}
	if parseResolvedMetadata.PropSchemaVersion != "props.v1" {
		parseT.Fatalf("expected prop schema version to round trip, got %q", parseResolvedMetadata.PropSchemaVersion)
	}
	if len(parseResolvedMetadata.FeatureFlags) != 2 {
		parseT.Fatalf("expected feature flags to round trip, got %+v", parseResolvedMetadata.FeatureFlags)
	}
}

// TestRegisterRendererRejectsInvalidMetadata verifies invalid metadata is rejected during registration.
func TestRegisterRendererRejectsInvalidMetadata(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	parseMetadata := RendererMetadata{
		FeatureFlags: []string{"display-only", ""},
	}
	if parseErr := RegisterRenderer(parseRendererID, func() {}, parseMetadata); parseErr == nil {
		parseT.Fatal("expected invalid metadata to fail")
	}
}

// TestValidateRendererMetadataRejectsRefFeatureFlag verifies ref-like metadata feature flags are rejected.
func TestValidateRendererMetadataRejectsRefFeatureFlag(parseT *testing.T) {
	parseMetadata := RendererMetadata{
		FeatureFlags: []string{"display-only", "ref"},
	}
	if parseErr := ValidateRendererMetadata(parseMetadata); parseErr == nil {
		parseT.Fatal("expected ref-like metadata feature flag to fail")
	}
}
