package runtime2

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

// RegionRenderer identifies a worker-renderable region implementation.
type RegionRenderer func()

// RendererID identifies one registered renderer implementation.
type RendererID string

// RegionInstanceID identifies one mounted region instance.
type RegionInstanceID string

// RendererMetadata stores optional renderer capability metadata.
type RendererMetadata struct {
	PropSchemaVersion string
	FeatureFlags      []string
	EventSlotMetadata EventSlotMetadata
}

type rendererRegistryEntry struct {
	renderer RegionRenderer
	metadata RendererMetadata
}

var (
	storeRendererRegistryMu      sync.RWMutex
	cacheRendererRegistryEntries = map[RendererID]rendererRegistryEntry{}
)

// ParseRendererID validates and normalizes a renderer identifier.
func ParseRendererID(parseRaw string) (RendererID, error) {
	parseTrimmed := strings.TrimSpace(parseRaw)
	if parseTrimmed == "" {
		return "", fmt.Errorf("runtime2: renderer ID is required")
	}
	return RendererID(parseTrimmed), nil
}

// ParseRegionInstanceID validates and normalizes a region-instance identifier.
func ParseRegionInstanceID(parseRaw string) (RegionInstanceID, error) {
	parseTrimmed := strings.TrimSpace(parseRaw)
	if parseTrimmed == "" {
		return "", fmt.Errorf("runtime2: region instance ID is required")
	}
	return RegionInstanceID(parseTrimmed), nil
}

// ValidateRendererMetadata verifies optional renderer metadata is internally consistent.
func ValidateRendererMetadata(parseMetadata RendererMetadata) error {
	if strings.TrimSpace(parseMetadata.PropSchemaVersion) != parseMetadata.PropSchemaVersion {
		return fmt.Errorf("runtime2: prop schema version must not contain surrounding whitespace")
	}
	parseSeenFeatureFlags := make(map[string]bool, len(parseMetadata.FeatureFlags))
	for _, parseFeatureFlag := range parseMetadata.FeatureFlags {
		if !parseRuntimeHasTrimmedNonWhitespaceText(parseFeatureFlag) {
			return fmt.Errorf("runtime2: feature flag is required")
		}
		if isRendererRefLikeFeatureFlag(parseFeatureFlag) {
			return fmt.Errorf("runtime2: feature flag %q uses unsupported ref marker", parseFeatureFlag)
		}
		if parseSeenFeatureFlags[parseFeatureFlag] {
			return fmt.Errorf("runtime2: duplicate feature flag %q", parseFeatureFlag)
		}
		parseSeenFeatureFlags[parseFeatureFlag] = true
	}
	if parseErr := ValidateEventSlotMetadata(parseMetadata.EventSlotMetadata); parseErr != nil {
		return parseErr
	}
	return nil
}

// HasRendererFeatureFlag reports whether renderer metadata contains one normalized feature flag.
func HasRendererFeatureFlag(parseMetadata RendererMetadata, parseFeatureFlag string) bool {
	if !parseRuntimeHasTrimmedNonWhitespaceText(parseFeatureFlag) {
		return false
	}
	for _, getFeatureFlag := range parseMetadata.FeatureFlags {
		if parseMatchRendererFeatureFlagFold(getFeatureFlag, parseFeatureFlag) {
			return true
		}
	}
	return false
}

// isRendererRefLikeFeatureFlag reports whether one renderer metadata feature flag is a disallowed ref marker.
func isRendererRefLikeFeatureFlag(parseFeatureFlag string) bool {
	return parseMatchRendererFeatureFlagFold(parseFeatureFlag, "ref") || parseMatchRendererFeatureFlagFold(parseFeatureFlag, "refs")
}

// parseMatchRendererFeatureFlagFold reports whether two feature flags match ignoring surrounding whitespace and ASCII case, with Unicode-safe fallback semantics.
func parseMatchRendererFeatureFlagFold(parseCurrent string, parseExpected string) bool {
	parseCurrentStart, parseCurrentEnd, hasParseCurrentASCII := parseBuildRendererFeatureFlagASCIIBounds(parseCurrent)
	parseExpectedStart, parseExpectedEnd, hasParseExpectedASCII := parseBuildRendererFeatureFlagASCIIBounds(parseExpected)
	if !hasParseCurrentASCII || !hasParseExpectedASCII {
		parseTrimmedCurrent := strings.TrimSpace(parseCurrent)
		parseTrimmedExpected := strings.TrimSpace(parseExpected)
		if parseTrimmedCurrent == "" || parseTrimmedExpected == "" {
			return false
		}
		return strings.EqualFold(parseTrimmedCurrent, parseTrimmedExpected)
	}
	if parseCurrentStart == parseCurrentEnd || parseExpectedStart == parseExpectedEnd {
		return false
	}
	if parseCurrentEnd-parseCurrentStart != parseExpectedEnd-parseExpectedStart {
		return false
	}
	for parseIndex := 0; parseIndex < parseCurrentEnd-parseCurrentStart; parseIndex++ {
		parseCurrentByte := parseCurrent[parseCurrentStart+parseIndex]
		parseExpectedByte := parseExpected[parseExpectedStart+parseIndex]
		if parseCurrentByte >= 'A' && parseCurrentByte <= 'Z' {
			parseCurrentByte += 'a' - 'A'
		}
		if parseExpectedByte >= 'A' && parseExpectedByte <= 'Z' {
			parseExpectedByte += 'a' - 'A'
		}
		if parseCurrentByte != parseExpectedByte {
			return false
		}
	}
	return true
}

// parseBuildRendererFeatureFlagASCIIBounds returns the trimmed ASCII slice bounds for one feature flag and reports whether ASCII-fast matching is safe.
func parseBuildRendererFeatureFlagASCIIBounds(parseRaw string) (int, int, bool) {
	for parseIndex := 0; parseIndex < len(parseRaw); parseIndex++ {
		if parseRaw[parseIndex] >= utf8.RuneSelf {
			return 0, 0, false
		}
	}
	parseStart := 0
	parseEnd := len(parseRaw)
	for parseStart < parseEnd && parseIsRendererFeatureFlagWhitespace(parseRaw[parseStart]) {
		parseStart++
	}
	for parseEnd > parseStart && parseIsRendererFeatureFlagWhitespace(parseRaw[parseEnd-1]) {
		parseEnd--
	}
	return parseStart, parseEnd, true
}

// parseIsRendererFeatureFlagWhitespace reports whether one byte is treated as feature-flag surrounding whitespace.
func parseIsRendererFeatureFlagWhitespace(parseByte byte) bool {
	switch parseByte {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

// RegisterRenderer registers a region renderer and its metadata by stable ID.
func RegisterRenderer(parseRendererID RendererID, parseRenderer RegionRenderer, parseMetadata RendererMetadata) error {
	if _, parseErr := ParseRendererID(string(parseRendererID)); parseErr != nil {
		return parseErr
	}
	if parseRenderer == nil {
		return fmt.Errorf("runtime2: renderer is required")
	}
	if parseErr := ValidateRendererMetadata(parseMetadata); parseErr != nil {
		return parseErr
	}
	storeRendererRegistryMu.Lock()
	defer storeRendererRegistryMu.Unlock()
	if _, parseExists := cacheRendererRegistryEntries[parseRendererID]; parseExists {
		return fmt.Errorf("runtime2: renderer %q is already registered", parseRendererID)
	}
	cacheRendererRegistryEntries[parseRendererID] = rendererRegistryEntry{
		renderer: parseRenderer,
		metadata: buildRendererMetadataCopy(parseMetadata),
	}
	return nil
}

// ResolveRenderer resolves a registered region renderer by stable ID.
func ResolveRenderer(parseRendererID RendererID) (RegionRenderer, RendererMetadata, error) {
	if _, parseErr := ParseRendererID(string(parseRendererID)); parseErr != nil {
		return nil, RendererMetadata{}, parseErr
	}
	storeRendererRegistryMu.RLock()
	defer storeRendererRegistryMu.RUnlock()
	parseEntry, parseExists := cacheRendererRegistryEntries[parseRendererID]
	if !parseExists {
		return nil, RendererMetadata{}, fmt.Errorf("runtime2: renderer %q is not registered", parseRendererID)
	}
	return parseEntry.renderer, buildRendererMetadataCopy(parseEntry.metadata), nil
}

// ResolveRendererMetadata resolves registered renderer metadata by stable ID.
func ResolveRendererMetadata(parseRendererID RendererID) (RendererMetadata, error) {
	if _, parseErr := ParseRendererID(string(parseRendererID)); parseErr != nil {
		return RendererMetadata{}, parseErr
	}
	storeRendererRegistryMu.RLock()
	defer storeRendererRegistryMu.RUnlock()
	parseEntry, parseExists := cacheRendererRegistryEntries[parseRendererID]
	if !parseExists {
		return RendererMetadata{}, fmt.Errorf("runtime2: renderer %q is not registered", parseRendererID)
	}
	return buildRendererMetadataCopy(parseEntry.metadata), nil
}

// SetRendererMetadata updates registered renderer metadata by stable ID.
func SetRendererMetadata(parseRendererID RendererID, parseMetadata RendererMetadata) error {
	if _, parseErr := ParseRendererID(string(parseRendererID)); parseErr != nil {
		return parseErr
	}
	if parseErr := ValidateRendererMetadata(parseMetadata); parseErr != nil {
		return parseErr
	}
	storeRendererRegistryMu.Lock()
	defer storeRendererRegistryMu.Unlock()
	parseEntry, parseExists := cacheRendererRegistryEntries[parseRendererID]
	if !parseExists {
		return fmt.Errorf("runtime2: renderer %q is not registered", parseRendererID)
	}
	parseEntry.metadata = buildRendererMetadataCopy(parseMetadata)
	cacheRendererRegistryEntries[parseRendererID] = parseEntry
	return nil
}

// ResetRendererRegistry clears the renderer registry for deterministic tests.
func ResetRendererRegistry() {
	storeRendererRegistryMu.Lock()
	defer storeRendererRegistryMu.Unlock()
	cacheRendererRegistryEntries = map[RendererID]rendererRegistryEntry{}
}

// buildRendererMetadataCopy clones one renderer metadata payload for registry storage and lookup.
func buildRendererMetadataCopy(parseMetadata RendererMetadata) RendererMetadata {
	return RendererMetadata{
		PropSchemaVersion: parseMetadata.PropSchemaVersion,
		FeatureFlags:      append([]string(nil), parseMetadata.FeatureFlags...),
		EventSlotMetadata: buildEventSlotMetadataCopy(parseMetadata.EventSlotMetadata),
	}
}

// buildEventSlotMetadataCopy clones one event-slot metadata payload.
func buildEventSlotMetadataCopy(parseMetadata EventSlotMetadata) EventSlotMetadata {
	return EventSlotMetadata{
		Version: parseMetadata.Version,
		Slots:   append([]EventSlotRecord(nil), parseMetadata.Slots...),
	}
}
