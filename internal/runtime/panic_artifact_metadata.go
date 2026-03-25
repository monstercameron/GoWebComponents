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
func SetWASMArtifactMetadata(parseMeta WASMArtifactMetadata) {
	wasmArtifactMetadata.mu.Lock()
	defer wasmArtifactMetadata.mu.Unlock()
	wasmArtifactMetadata.meta = parseMeta
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
func panicArtifactFields(parseMeta WASMArtifactMetadata) map[string]string {
	parseFields := map[string]string{}
	if parseMeta.BuildID != "" {
		parseFields["artifact_build_id"] = parseMeta.BuildID
	}
	if parseMeta.ArtifactPath != "" {
		parseFields["artifact_path"] = parseMeta.ArtifactPath
	}
	if parseMeta.SHA256 != "" {
		parseFields["artifact_sha256"] = parseMeta.SHA256
	}
	if parseMeta.ManifestPath != "" {
		parseFields["artifact_manifest"] = parseMeta.ManifestPath
	}
	if parseMeta.SymbolSet != "" {
		parseFields["artifact_symbols"] = parseMeta.SymbolSet
	}
	if parseMeta.Version != "" {
		parseFields["artifact_version"] = parseMeta.Version
	}
	if len(parseFields) == 0 {
		return nil
	}
	return parseFields
}
