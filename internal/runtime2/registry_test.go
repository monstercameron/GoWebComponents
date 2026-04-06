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

// TestResolveRendererMetadataRejectsUnregisteredRenderer verifies metadata lookup fails clearly for missing IDs.
func TestResolveRendererMetadataRejectsUnregisteredRenderer(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.missing")
	if _, parseErr := ResolveRendererMetadata(parseRendererID); parseErr == nil {
		parseT.Fatal("expected ResolveRendererMetadata to fail for an unregistered renderer")
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
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{
					SlotID:    "slot-1",
					EventType: "click",
				},
			},
		},
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
	if parseResolvedMetadata.EventSlotMetadata.Version != EventSlotMetadataVersionV1 {
		parseT.Fatalf("expected event-slot metadata version to round trip, got %+v", parseResolvedMetadata.EventSlotMetadata)
	}
	if len(parseResolvedMetadata.EventSlotMetadata.Slots) != 1 || parseResolvedMetadata.EventSlotMetadata.Slots[0].EventType != "click" {
		parseT.Fatalf("expected event-slot metadata slots to round trip, got %+v", parseResolvedMetadata.EventSlotMetadata.Slots)
	}
}

// TestResolveRendererMetadataReturnsCopy verifies metadata-only lookup returns the registered metadata by value.
func TestResolveRendererMetadataReturnsCopy(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	parseMetadata := RendererMetadata{
		PropSchemaVersion: "props.v1",
		FeatureFlags:      []string{"display-only"},
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{
					SlotID:    "slot-1",
					EventType: "click",
				},
			},
		},
	}
	if parseErr := RegisterRenderer(parseRendererID, func() {}, parseMetadata); parseErr != nil {
		parseT.Fatalf("RegisterRenderer returned error: %v", parseErr)
	}
	getResolvedMetadata, parseErr := ResolveRendererMetadata(parseRendererID)
	if parseErr != nil {
		parseT.Fatalf("ResolveRendererMetadata returned error: %v", parseErr)
	}
	getResolvedMetadata.FeatureFlags[0] = "mutated"
	getResolvedMetadata.EventSlotMetadata.Slots[0].EventType = "submit"
	getResolvedMetadataAgain, parseResolveErr := ResolveRendererMetadata(parseRendererID)
	if parseResolveErr != nil {
		parseT.Fatalf("ResolveRendererMetadata(second) returned error: %v", parseResolveErr)
	}
	if getResolvedMetadataAgain.FeatureFlags[0] != "display-only" {
		parseT.Fatalf("expected feature flags to be copied, got %+v", getResolvedMetadataAgain.FeatureFlags)
	}
	if getResolvedMetadataAgain.EventSlotMetadata.Slots[0].EventType != "click" {
		parseT.Fatalf("expected event-slot metadata to be copied, got %+v", getResolvedMetadataAgain.EventSlotMetadata)
	}
}

// TestSetRendererMetadataUpdatesRegisteredMetadata verifies metadata updates replace the stored registry metadata by value.
func TestSetRendererMetadataUpdatesRegisteredMetadata(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	if parseErr := RegisterRenderer(parseRendererID, func() {}, RendererMetadata{
		FeatureFlags: []string{"display-only"},
	}); parseErr != nil {
		parseT.Fatalf("RegisterRenderer returned error: %v", parseErr)
	}
	parseUpdatedMetadata := RendererMetadata{
		FeatureFlags: []string{"display-only"},
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{
					SlotID:    "primary.action",
					EventType: "click",
				},
			},
		},
	}
	if parseErr := SetRendererMetadata(parseRendererID, parseUpdatedMetadata); parseErr != nil {
		parseT.Fatalf("SetRendererMetadata returned error: %v", parseErr)
	}
	getResolvedMetadata, parseResolveErr := ResolveRendererMetadata(parseRendererID)
	if parseResolveErr != nil {
		parseT.Fatalf("ResolveRendererMetadata returned error: %v", parseResolveErr)
	}
	if len(getResolvedMetadata.EventSlotMetadata.Slots) != 1 || getResolvedMetadata.EventSlotMetadata.Slots[0].SlotID != "primary.action" {
		parseT.Fatalf("expected updated event-slot metadata, got %+v", getResolvedMetadata.EventSlotMetadata)
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

// TestHasRendererFeatureFlagMatchesNormalizedValue verifies metadata feature-flag lookup is case-insensitive and trim-safe.
func TestHasRendererFeatureFlagMatchesNormalizedValue(parseT *testing.T) {
	parseMetadata := RendererMetadata{
		FeatureFlags: []string{"display-only", "Derived-State"},
	}
	if !HasRendererFeatureFlag(parseMetadata, " display-only ") {
		parseT.Fatal("expected display-only feature flag lookup to succeed")
	}
	if !HasRendererFeatureFlag(parseMetadata, "derived-state") {
		parseT.Fatal("expected derived-state feature flag lookup to succeed")
	}
	if HasRendererFeatureFlag(parseMetadata, "interactive") {
		parseT.Fatal("expected unknown feature flag lookup to fail")
	}
}

// TestRegisterRendererRejectsInvalidEventSlotMetadata verifies invalid event-slot metadata is rejected during registration.
func TestRegisterRendererRejectsInvalidEventSlotMetadata(parseT *testing.T) {
	ResetRendererRegistry()
	parseT.Cleanup(ResetRendererRegistry)
	parseRendererID, _ := ParseRendererID("dashboard.hot-panel")
	parseMetadata := RendererMetadata{
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{
					SlotID:    "",
					EventType: "click",
				},
			},
		},
	}
	if parseErr := RegisterRenderer(parseRendererID, func() {}, parseMetadata); parseErr == nil {
		parseT.Fatal("expected invalid event-slot metadata to fail")
	}
}
