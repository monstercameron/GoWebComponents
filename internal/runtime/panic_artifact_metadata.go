package runtime

import "sync"

// WASMArtifactMetadata identifies the emitted browser artifact and related build records.
type WASMArtifactMetadata struct {
	BuildID      string
	ArtifactPath string
	SHA256       string
	ManifestPath string
	SymbolSet    string
	Version      string
}

var wasmArtifactMetadata struct {
	mu   sync.RWMutex
	meta WASMArtifactMetadata
}

// SetWASMArtifactMetadata stores the current emitted artifact metadata for panic correlation.
func SetWASMArtifactMetadata(parseArtifactMeta WASMArtifactMetadata) {
	wasmArtifactMetadata.mu.Lock()
	defer wasmArtifactMetadata.mu.Unlock()
	wasmArtifactMetadata.meta = parseArtifactMeta
}

// ResetWASMArtifactMetadata clears the current emitted artifact metadata.
func ResetWASMArtifactMetadata() {
	SetWASMArtifactMetadata(WASMArtifactMetadata{})
}

// currentWASMArtifactMetadata is a core package helper.
func currentWASMArtifactMetadata() WASMArtifactMetadata {
	wasmArtifactMetadata.mu.RLock()
	defer wasmArtifactMetadata.mu.RUnlock()
	return wasmArtifactMetadata.meta
}

// panicArtifactFields is a core package helper.
func panicArtifactFields(parseArtifactMeta WASMArtifactMetadata) map[string]string {
	parseArtifactFields := map[string]string{}
	if parseArtifactMeta.BuildID != "" {
		parseArtifactFields["artifact_build_id"] = parseArtifactMeta.BuildID
	}
	if parseArtifactMeta.ArtifactPath != "" {
		parseArtifactFields["artifact_path"] = parseArtifactMeta.ArtifactPath
	}
	if parseArtifactMeta.SHA256 != "" {
		parseArtifactFields["artifact_sha256"] = parseArtifactMeta.SHA256
	}
	if parseArtifactMeta.ManifestPath != "" {
		parseArtifactFields["artifact_manifest"] = parseArtifactMeta.ManifestPath
	}
	if parseArtifactMeta.SymbolSet != "" {
		parseArtifactFields["artifact_symbols"] = parseArtifactMeta.SymbolSet
	}
	if parseArtifactMeta.Version != "" {
		parseArtifactFields["artifact_version"] = parseArtifactMeta.Version
	}
	if len(parseArtifactFields) == 0 {
		return nil
	}
	return parseArtifactFields
}
