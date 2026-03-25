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
func SetWASMArtifactMetadata(meta WASMArtifactMetadata) {
	wasmArtifactMetadata.mu.Lock()
	defer wasmArtifactMetadata.mu.Unlock()
	wasmArtifactMetadata.meta = meta
}

// ResetWASMArtifactMetadata clears the current emitted artifact metadata.
func ResetWASMArtifactMetadata() {
	SetWASMArtifactMetadata(WASMArtifactMetadata{})
}

func currentWASMArtifactMetadata() WASMArtifactMetadata {
	wasmArtifactMetadata.mu.RLock()
	defer wasmArtifactMetadata.mu.RUnlock()
	return wasmArtifactMetadata.meta
}

func panicArtifactFields(meta WASMArtifactMetadata) map[string]string {
	fields := map[string]string{}
	if meta.BuildID != "" {
		fields["artifact_build_id"] = meta.BuildID
	}
	if meta.ArtifactPath != "" {
		fields["artifact_path"] = meta.ArtifactPath
	}
	if meta.SHA256 != "" {
		fields["artifact_sha256"] = meta.SHA256
	}
	if meta.ManifestPath != "" {
		fields["artifact_manifest"] = meta.ManifestPath
	}
	if meta.SymbolSet != "" {
		fields["artifact_symbols"] = meta.SymbolSet
	}
	if meta.Version != "" {
		fields["artifact_version"] = meta.Version
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}
