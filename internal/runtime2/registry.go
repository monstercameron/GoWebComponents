package runtime2

import (
	"fmt"
	"strings"
	"sync"
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
		if strings.TrimSpace(parseFeatureFlag) == "" {
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

// isRendererRefLikeFeatureFlag reports whether one renderer metadata feature flag is a disallowed ref marker.
func isRendererRefLikeFeatureFlag(parseFeatureFlag string) bool {
	parseNormalizedFeatureFlag := strings.ToLower(strings.TrimSpace(parseFeatureFlag))
	return parseNormalizedFeatureFlag == "ref" || parseNormalizedFeatureFlag == "refs"
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
