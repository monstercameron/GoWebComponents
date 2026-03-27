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
	if parseSnapshot.RegionInstanceID == "" {
		return WorkerRenderInput{}, nil
	}
	if parseSnapshotErr := ValidateSnapshotEnvelope(parseSnapshot); parseSnapshotErr != nil {
		return WorkerRenderInput{}, parseSnapshotErr
	}
	parseSourceIDs := make([]string, 0, len(parseSnapshot.Sources))
	for parseSourceID := range parseSnapshot.Sources {
		parseSourceIDs = append(parseSourceIDs, parseSourceID)
	}
	parseSourceIDs, parseSourceIDsErr := NormalizeSourceIDs(parseSourceIDs)
	if parseSourceIDsErr != nil {
		return WorkerRenderInput{}, parseSourceIDsErr
	}
	parseSourceEntries := make([]WorkerRenderSourceEntry, 0, len(parseSourceIDs))
	for _, parseSourceID := range parseSourceIDs {
		parseSourceEntries = append(parseSourceEntries, WorkerRenderSourceEntry{
			GetSourceID:    parseSourceID,
			GetSourceValue: parseSnapshot.Sources[parseSourceID],
		})
	}
	return WorkerRenderInput{
		GetProps:         parseSnapshot.Props,
		GetSourceVersion: parseSnapshot.SourceVersion,
		GetSourceEntries: parseSourceEntries,
	}, nil
}
