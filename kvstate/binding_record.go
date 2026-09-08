package kvstate

// shouldApplyBindingRecord prevents delayed loads and tombstones rolling versions back.
// Conflict resolution must choose the incoming version, not merely return a local
// version equal to the current one (which previously admitted every stale load).
func shouldApplyBindingRecord(parseLocalVersion int64, parseIncoming Record, parseResolver ConflictResolver) bool {
	if parseIncoming.Version < parseLocalVersion {
		return false
	}
	parseLocal := Record{Key: parseIncoming.Key, Version: parseLocalVersion}
	return parseResolver.Resolve(parseLocal, parseIncoming).Version == parseIncoming.Version
}
