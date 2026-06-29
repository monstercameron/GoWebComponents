package localfirst

import "sort"

// Text is an op-based collaborative-text CRDT — a Replicated Growable Array (RGA). Where the
// last-write-wins Register keeps only one of two concurrent edits to a string field, Text keeps
// BOTH: every inserted character is an immutable element with a globally-unique id, positioned
// relative to the element it was typed after, and deletions are tombstones. All replicas
// converge to the same string regardless of the order edits and merges arrive, and no
// concurrent insertion is ever silently dropped. This is the answer to the audit's
// collaborative-text gap that the PN-[Counter] (integers only) could not cover.
//
// Text is value-semantic per replica; guard a Text shared across goroutines with a mutex. Each
// replica must construct its Text with a distinct, stable replica id (see [NewText]).
type Text struct {
	replica string
	seq     uint64
	elems   map[elemID]*textElem
}

// elemID is the globally-unique identity of one character element: the replica that created it
// and that replica's monotonically increasing sequence number. The zero value is the head
// sentinel that the first characters are inserted after.
type elemID struct {
	Replica string
	Seq     uint64
}

// textElem is one character in the RGA: its id, the id of the element it was inserted after, the
// character itself, and whether it has been tombstoned (deleted-wins on merge).
type textElem struct {
	id      elemID
	after   elemID
	char    string
	deleted bool
}

// NewText creates an empty collaborative text owned by the given replica id. The id must be
// unique per participant (a user/session/device id) and stable for that participant, because it
// breaks ties between concurrent insertions at the same position.
func NewText(parseReplica string) *Text {
	return &Text{replica: parseReplica, elems: map[elemID]*textElem{}}
}

// nextID mints the next id for a local operation, advancing this replica's sequence clock.
func (parseT *Text) nextID() elemID {
	parseT.seq++
	return elemID{Replica: parseT.replica, Seq: parseT.seq}
}

// Insert places s immediately before visible position index (0 = start, Len = append). Each
// character becomes a new element chained after the previous, so a multi-character insert stays
// contiguous and converges as a unit.
func (parseT *Text) Insert(parseIndex int, parseString string) {
	parseVisible := parseT.visible()
	parseAfter := elemID{} // head sentinel
	if parseIndex < 0 {
		parseIndex = 0
	}
	if parseIndex > len(parseVisible) {
		parseIndex = len(parseVisible)
	}
	if parseIndex > 0 {
		parseAfter = parseVisible[parseIndex-1].id
	}
	for _, parseRune := range parseString {
		parseID := parseT.nextID()
		parseT.elems[parseID] = &textElem{id: parseID, after: parseAfter, char: string(parseRune)}
		parseAfter = parseID
	}
}

// Delete tombstones count visible characters starting at visible position index. Out-of-range
// requests are clamped; deleting an already-deleted element is a no-op.
func (parseT *Text) Delete(parseIndex int, parseCount int) {
	parseVisible := parseT.visible()
	for parseI := range parseCount {
		parsePos := parseIndex + parseI
		if parsePos < 0 || parsePos >= len(parseVisible) {
			break
		}
		parseVisible[parsePos].deleted = true
	}
}

// Value returns the current text: every non-tombstoned element in convergent order.
func (parseT *Text) Value() string {
	parseBuilder := make([]byte, 0, len(parseT.elems))
	for _, parseElem := range parseT.ordered() {
		if !parseElem.deleted {
			parseBuilder = append(parseBuilder, parseElem.char...)
		}
	}
	return string(parseBuilder)
}

// Len returns the number of visible (non-deleted) characters.
func (parseT *Text) Len() int {
	return len(parseT.visible())
}

// visible returns the non-deleted elements in convergent order.
func (parseT *Text) visible() []*textElem {
	parseOrdered := parseT.ordered()
	parseOut := parseOrdered[:0]
	for _, parseElem := range parseOrdered {
		if !parseElem.deleted {
			parseOut = append(parseOut, parseElem)
		}
	}
	return parseOut
}

// ordered returns every element (including tombstones) in the RGA's convergent total order:
// a depth-first walk from the head, where the children inserted after the same element are
// visited highest-id-first. Because the order depends only on element ids — never on insertion
// or merge order — every replica that holds the same element set produces the same string.
func (parseT *Text) ordered() []*textElem {
	parseChildren := map[elemID][]*textElem{}
	for _, parseElem := range parseT.elems {
		parseChildren[parseElem.after] = append(parseChildren[parseElem.after], parseElem)
	}
	for _, parseList := range parseChildren {
		sort.Slice(parseList, func(parseI int, parseJ int) bool {
			return idGreater(parseList[parseI].id, parseList[parseJ].id)
		})
	}
	parseResult := make([]*textElem, 0, len(parseT.elems))
	var parseWalk func(parseParent elemID)
	parseWalk = func(parseParent elemID) {
		for _, parseChild := range parseChildren[parseParent] {
			parseResult = append(parseResult, parseChild)
			parseWalk(parseChild.id)
		}
	}
	parseWalk(elemID{})
	return parseResult
}

// idGreater is the total order on element ids used to break ties between concurrent insertions
// at the same position: higher sequence first, then higher replica id. A consistent total order
// is all convergence requires; this one keeps the most recent concurrent insert nearest its
// anchor.
func idGreater(parseA elemID, parseB elemID) bool {
	if parseA.Seq != parseB.Seq {
		return parseA.Seq > parseB.Seq
	}
	return parseA.Replica > parseB.Replica
}

// Merge converges this text with another by unioning their element sets: an element present in
// either side is kept, and a tombstone on either side wins (deleted is monotonic). The local
// sequence clock is advanced past every merged id so future local inserts mint fresh, ordered
// ids. Merge is commutative, associative, and idempotent.
func (parseT *Text) Merge(parseOther *Text) {
	if parseOther == nil {
		return
	}
	for parseID, parseElem := range parseOther.elems {
		if parseExisting, parseFound := parseT.elems[parseID]; parseFound {
			if parseElem.deleted {
				parseExisting.deleted = true
			}
			continue
		}
		parseCopy := *parseElem
		parseT.elems[parseID] = &parseCopy
		if parseID.Seq > parseT.seq {
			parseT.seq = parseID.Seq
		}
	}
}

// TextOp is the JSON-serializable form of one character element, for persistence/transport.
type TextOp struct {
	Replica      string `json:"r"`
	Seq          uint64 `json:"s"`
	AfterReplica string `json:"ar,omitempty"`
	AfterSeq     uint64 `json:"as,omitempty"`
	Char         string `json:"c,omitempty"`
	Deleted      bool   `json:"d,omitempty"`
}

// TextState is the full op log of a Text — the set of character elements — for persistence and
// transport. Being op-based, the state IS the operation set.
type TextState struct {
	Replica string   `json:"replica"`
	Ops     []TextOp `json:"ops"`
}

// Export captures the text's state in a deterministic (convergent) op order, so two replicas
// with the same elements serialize identically.
func (parseT *Text) Export() TextState {
	parseOrdered := parseT.ordered()
	parseOps := make([]TextOp, 0, len(parseOrdered))
	for _, parseElem := range parseOrdered {
		parseOps = append(parseOps, TextOp{
			Replica:      parseElem.id.Replica,
			Seq:          parseElem.id.Seq,
			AfterReplica: parseElem.after.Replica,
			AfterSeq:     parseElem.after.Seq,
			Char:         parseElem.char,
			Deleted:      parseElem.deleted,
		})
	}
	return TextState{Replica: parseT.replica, Ops: parseOps}
}

// RestoreText rebuilds a Text from persisted state under the given replica id. The replica id is
// supplied by the caller (not read from the state) so a participant can adopt a shared document
// under its own identity; the local sequence clock is advanced past every restored id.
func RestoreText(parseState TextState, parseReplica string) *Text {
	parseText := NewText(parseReplica)
	for _, parseOp := range parseState.Ops {
		parseID := elemID{Replica: parseOp.Replica, Seq: parseOp.Seq}
		parseText.elems[parseID] = &textElem{
			id:      parseID,
			after:   elemID{Replica: parseOp.AfterReplica, Seq: parseOp.AfterSeq},
			char:    parseOp.Char,
			deleted: parseOp.Deleted,
		}
		if parseID.Seq > parseText.seq {
			parseText.seq = parseID.Seq
		}
	}
	return parseText
}
