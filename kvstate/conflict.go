package kvstate

// ConflictResolver decides the winner when a local record meets an incoming one
// (e.g. a cross-tab write). Implement it for custom merge semantics.
type ConflictResolver interface {
	Resolve(parseLocal, parseIncoming Record) Record
}

// LastWriteWins keeps the record with the newer UpdatedAt. It is the default.
type LastWriteWins struct{}

func (LastWriteWins) Resolve(parseLocal, parseIncoming Record) Record {
	if parseIncoming.UpdatedAt >= parseLocal.UpdatedAt {
		return parseIncoming
	}
	return parseLocal
}

// Versioned keeps the record with the higher Version, rejecting stale writes.
type Versioned struct{}

func (Versioned) Resolve(parseLocal, parseIncoming Record) Record {
	if parseIncoming.Version >= parseLocal.Version {
		return parseIncoming
	}
	return parseLocal
}
