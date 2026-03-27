package runtime2

// WorkerRenderSourceEntry stores one deterministic declared-source entry for worker renderer input.
type WorkerRenderSourceEntry struct {
	GetSourceID    string
	GetSourceValue any
}

// WorkerRenderInput stores deterministic worker renderer input derived from one snapshot envelope.
type WorkerRenderInput struct {
	GetProps         any
	GetSourceVersion uint64
	GetSourceEntries []WorkerRenderSourceEntry
}

// BuildWorkerRenderInput builds deterministic worker renderer input from one validated snapshot envelope.
func BuildWorkerRenderInput(parseSnapshot SnapshotEnvelope) (WorkerRenderInput, error) {
	getRenderInput, _, parseErr := buildWorkerRenderInputWithSourceOrder(parseSnapshot, nil)
	if parseErr != nil {
		return WorkerRenderInput{}, parseErr
	}
	return getRenderInput, nil
}

// buildWorkerRenderInputWithSourceOrder builds deterministic worker renderer input and reuses one cached normalized source-ID order when the source keyset is unchanged.
func buildWorkerRenderInputWithSourceOrder(parseSnapshot SnapshotEnvelope, parseCachedSourceIDs []string) (WorkerRenderInput, []string, error) {
	if parseSnapshot.RegionInstanceID == "" {
		return WorkerRenderInput{}, nil, nil
	}
	if parseSnapshotErr := ValidateSnapshotEnvelope(parseSnapshot); parseSnapshotErr != nil {
		return WorkerRenderInput{}, nil, parseSnapshotErr
	}
	parseSourceIDs := parseCachedSourceIDs
	if !hasWorkerRenderSnapshotSourceOrderMatch(parseSnapshot.Sources, parseCachedSourceIDs) {
		parseSourceIDs = make([]string, 0, len(parseSnapshot.Sources))
		for parseSourceID := range parseSnapshot.Sources {
			parseSourceIDs = append(parseSourceIDs, parseSourceID)
		}
		getNormalizedSourceIDs, parseSourceIDsErr := NormalizeSourceIDs(parseSourceIDs)
		if parseSourceIDsErr != nil {
			return WorkerRenderInput{}, nil, parseSourceIDsErr
		}
		parseSourceIDs = getNormalizedSourceIDs
	}
	buildSourceEntries := make([]WorkerRenderSourceEntry, len(parseSourceIDs))
	for parseSourceIndex, parseSourceID := range parseSourceIDs {
		buildSourceEntries[parseSourceIndex] = WorkerRenderSourceEntry{
			GetSourceID:    parseSourceID,
			GetSourceValue: parseSnapshot.Sources[parseSourceID],
		}
	}
	return WorkerRenderInput{
		GetProps:         parseSnapshot.Props,
		GetSourceVersion: parseSnapshot.SourceVersion,
		GetSourceEntries: buildSourceEntries,
	}, append([]string(nil), parseSourceIDs...), nil
}

// hasWorkerRenderSnapshotSourceOrderMatch reports whether one cached source-ID order exactly matches the current snapshot source keyset.
func hasWorkerRenderSnapshotSourceOrderMatch(parseSnapshotSources map[string]any, parseCachedSourceIDs []string) bool {
	if len(parseSnapshotSources) != len(parseCachedSourceIDs) {
		return false
	}
	for _, parseSourceID := range parseCachedSourceIDs {
		if _, hasSourceID := parseSnapshotSources[parseSourceID]; !hasSourceID {
			return false
		}
	}
	return true
}
