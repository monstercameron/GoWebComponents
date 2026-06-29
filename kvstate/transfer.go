package kvstate

import "context"

// Export/Import are the ingress/egress surface for moving persisted state across
// apps, domains, or devices: Export reads every record out of a backend; Import
// writes records into a backend with conflict resolution. Wire either side to an
// HTTP endpoint (a custom PersistenceBackend, or a plain handler over these) and
// state syncs across origins — the browser BroadcastChannel only reaches the
// same origin, but bytes over an API have no such limit.

// Export reads every record from a backend (egress). The result is a portable
// snapshot you can JSON/CBOR-encode and POST elsewhere.
func Export(parseCtx context.Context, parseBackend PersistenceBackend) ([]Record, error) {
	parseKeys, parseErr := parseBackend.Keys(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseOut := make([]Record, 0, len(parseKeys))
	for _, parseKey := range parseKeys {
		parseRecord, parseFound, parseLoadErr := parseBackend.Load(parseCtx, parseKey)
		if parseLoadErr != nil {
			return nil, parseLoadErr
		}
		if parseFound {
			parseOut = append(parseOut, parseRecord)
		}
	}
	return parseOut, nil
}

// Import writes records into a backend (ingress), resolving conflicts against any
// existing record with parseResolver. A nil resolver defaults to LastWriteWins.
// Records the resolver rejects (e.g. stale Versioned writes) are skipped, so an
// import never clobbers newer local state.
func Import(parseCtx context.Context, parseBackend PersistenceBackend, parseRecords []Record, parseResolver ConflictResolver) error {
	if parseResolver == nil {
		parseResolver = LastWriteWins{}
	}
	for _, parseIncoming := range parseRecords {
		parseLocal, parseFound, parseErr := parseBackend.Load(parseCtx, parseIncoming.Key)
		if parseErr != nil {
			return parseErr
		}
		parseWinner := parseIncoming
		if parseFound {
			parseWinner = parseResolver.Resolve(parseLocal, parseIncoming)
		}
		if parseErr := parseBackend.Save(parseCtx, parseWinner); parseErr != nil {
			return parseErr
		}
	}
	return nil
}
