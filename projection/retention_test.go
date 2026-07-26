package projection

import "testing"

// Internal test: asserting that removal RELEASES memory means reading the
// backing array past len, which the external projection_test package cannot do.

// TestRemovedRowsAreReleased pins the release of removed rows.
//
// applyRemove used to compact with append(entries[:i], entries[i+1:]...), which
// shifts left but leaves the old last element in the slot past the new length.
// Every removal left its own residue, so a projection that churned its window
// kept a decoded row at every position behind the shortened slice — this exact
// case (insert 8, remove 8) reported zero rows while all eight payloads stayed
// reachable for the life of the projection.
//
// That is the memory M12 budgets, so it is the residency cap's own guarantee
// that was being violated. See memory_test.go for the budget itself.
func TestRemovedRowsAreReleased(parseT *testing.T) {
	parseDecode := func(parsePayload []byte) (string, error) { return string(parsePayload), nil }
	parseProjection, parseErr := New(parseDecode, Options{Resident: 100})
	if parseErr != nil {
		parseT.Fatalf("New: %v", parseErr)
	}

	const parseRowCount = 8
	parseInserts := make([]Op, 0, parseRowCount)
	for parseIndex := range parseRowCount {
		parseInserts = append(parseInserts, Op{
			Kind:    OpInsert,
			Key:     Key(string(rune('a' + parseIndex))),
			Payload: []byte("payload-" + string(rune('a'+parseIndex))),
		})
	}
	if parseApplyErr := parseProjection.Apply(parseInserts); parseApplyErr != nil {
		parseT.Fatalf("Apply(inserts): %v", parseApplyErr)
	}

	parseRemovals := make([]Op, 0, parseRowCount)
	for parseIndex := range parseRowCount {
		parseRemovals = append(parseRemovals, Op{Kind: OpRemove, Key: Key(string(rune('a' + parseIndex)))})
	}
	if parseApplyErr := parseProjection.Apply(parseRemovals); parseApplyErr != nil {
		parseT.Fatalf("Apply(removals): %v", parseApplyErr)
	}

	if len(parseProjection.entries) != 0 {
		parseT.Fatalf("projection reports %d rows after removing all of them", len(parseProjection.entries))
	}

	parseRetained := 0
	for _, parseEntry := range parseProjection.entries[:cap(parseProjection.entries)] {
		if parseEntry.Value != "" || parseEntry.Key != "" {
			parseRetained++
		}
	}
	if parseRetained != 0 {
		parseT.Errorf("an empty projection still holds %d row(s) reachable in its backing array; removal must release the slot it vacates", parseRetained)
	}
}

// TestMoveReleasesTheSlotItVacates: a move is a remove followed by a re-insert,
// and the remove half compacts the same way. The row itself is re-inserted, so
// what must not survive is a SECOND copy of it left behind at the tail.
func TestMoveReleasesTheSlotItVacates(parseT *testing.T) {
	parseDecode := func(parsePayload []byte) (string, error) { return string(parsePayload), nil }
	parseProjection, parseErr := New(parseDecode, Options{Resident: 100})
	if parseErr != nil {
		parseT.Fatalf("New: %v", parseErr)
	}

	if parseApplyErr := parseProjection.Apply([]Op{
		{Kind: OpInsert, Key: "a", Payload: []byte("row-a")},
		{Kind: OpInsert, Key: "b", AfterKey: "a", Payload: []byte("row-b")},
		{Kind: OpInsert, Key: "c", AfterKey: "b", Payload: []byte("row-c")},
	}); parseApplyErr != nil {
		parseT.Fatalf("Apply(inserts): %v", parseApplyErr)
	}

	// Move the head to the tail.
	if parseApplyErr := parseProjection.Apply([]Op{{Kind: OpMove, Key: "a", AfterKey: "c"}}); parseApplyErr != nil {
		parseT.Fatalf("Apply(move): %v", parseApplyErr)
	}

	if len(parseProjection.entries) != 3 {
		parseT.Fatalf("expected 3 rows after a move, got %d", len(parseProjection.entries))
	}
	parseSeen := map[Key]int{}
	for _, parseEntry := range parseProjection.entries[:cap(parseProjection.entries)] {
		if parseEntry.Key != "" {
			parseSeen[parseEntry.Key]++
		}
	}
	for parseKey, parseCount := range parseSeen {
		if parseCount > 1 {
			parseT.Errorf("row %q appears %d times in the backing array; the move left a copy behind", parseKey, parseCount)
		}
	}
}
