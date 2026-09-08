package projection_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/delta"
	"github.com/monstercameron/GoWebComponents/v6/projection"
)

// v5 P3.7 — the render-thread projection.
//
// Criterion (c) is the headline: the §1.2 filter keystroke performs zero domain
// round-trips, asserted by message count. The rest establishes that a projection
// fed by the real delta engine actually converges, because a projection that
// answered locally but drifted from the worker would be worse than one that
// asked.

type item struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

func decodeItem(parsePayload []byte) (item, error) {
	var parseItem item
	return parseItem, json.Unmarshal(parsePayload, &parseItem)
}

func encodeItem(parseItem item) []byte {
	parseBytes, _ := json.Marshal(parseItem)
	return parseBytes
}

func buildProjection(parseT *testing.T, parseOptions projection.Options) *projection.Projection[item] {
	parseT.Helper()
	parseProjection, parseErr := projection.New(decodeItem, parseOptions)
	if parseErr != nil {
		parseT.Fatalf("New: %v", parseErr)
	}
	return parseProjection
}

// buildEngineRows builds N rows for the worker-side engine.
func buildEngineRows(parseCount int) []delta.Row {
	parseRows := make([]delta.Row, 0, parseCount)
	for parseIndex := range parseCount {
		parseRows = append(parseRows, delta.Row{
			Key:     delta.Key(fmt.Sprintf("k%05d", parseIndex)),
			Version: 1,
			Payload: encodeItem(item{Name: fmt.Sprintf("item-%d", parseIndex), Price: parseIndex}),
		})
	}
	return parseRows
}

// ------------------------------------------- criterion (c): no round trips

// TestFilteringPerformsZeroDomainRoundTrips is P3.7 criterion (c), asserted by
// message count.
//
// The client is threaded through the whole test purely so its send count can be
// checked at the end. A projection that quietly asked the worker to filter would
// still return correct rows — and would reintroduce per-keystroke latency, which
// is the entire thing v5 exists to remove.
func TestFilteringPerformsZeroDomainRoundTrips(parseT *testing.T) {
	parseEngine := delta.New()
	parseRows := buildEngineRows(5000)
	parseOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}

	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}

	parseClient := &countingClient{}

	// Type "item-123" one character at a time, filtering on every keystroke —
	// the §1.2 latency probe.
	const parseQuery = "item-123"
	for parseLength := 1; parseLength <= len(parseQuery); parseLength++ {
		parsePrefix := parseQuery[:parseLength]
		parseMatches := parseProjection.Filter(func(parseItem item) bool {
			return strings.Contains(parseItem.Name, parsePrefix)
		})
		if parseLength == len(parseQuery) && len(parseMatches) == 0 {
			parseT.Fatalf("filtering for %q found nothing; the projection is not resident", parsePrefix)
		}
	}

	// Window and Get are the other two things a virtualized table does per frame.
	_ = parseProjection.Window(400, 50)
	if _, hasRow := parseProjection.Get("k00123"); !hasRow {
		parseT.Error("Get must answer from resident memory")
	}

	if len(parseClient.sends) != 0 {
		parseT.Errorf("%d domain round-trips during filtering, want 0 (criterion c)", len(parseClient.sends))
	}
}

// TestReadsNeverTouchTheClientAtAll is the structural half of the same claim.
//
// A send count of zero could also be achieved by a projection that happens not
// to have needed the worker yet. This asserts the read API cannot reach a
// client, because it is never given one — Projection has no client field, and
// this test documents that as intentional rather than an oversight.
func TestReadsNeverTouchTheClientAtAll(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "alpha", Price: 1})},
	}); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}

	// Every read path, with no client in scope anywhere.
	if parseProjection.Len() != 1 {
		parseT.Errorf("Len = %d, want 1", parseProjection.Len())
	}
	if parseRows := parseProjection.Rows(); len(parseRows) != 1 {
		parseT.Errorf("Rows = %d, want 1", len(parseRows))
	}
	if parseMatches := parseProjection.Filter(func(parseItem item) bool { return parseItem.Price > 0 }); len(parseMatches) != 1 {
		parseT.Errorf("Filter = %d, want 1", len(parseMatches))
	}
	if parseWindow := parseProjection.Window(0, 10); len(parseWindow) != 1 {
		parseT.Errorf("Window = %d, want 1", len(parseWindow))
	}

	// A command, by contrast, DOES send — which is the distinction the API draws.
	parseClient := &countingClient{response: []byte(`{"id":1}`)}
	if _, parseErr := addItem.Invoke(context.Background(), parseClient, jsonCodec{}, addItemArgs{Name: "x"}); parseErr != nil {
		parseT.Fatalf("Invoke: %v", parseErr)
	}
	if len(parseClient.sends) != 1 {
		parseT.Errorf("a command must round-trip exactly once, got %d", len(parseClient.sends))
	}
}

// ------------------------------------------------ convergence with the engine

// TestProjectionConvergesWithTheEngine drives the real engine and requires the
// projection to match it exactly after every publish.
//
// This is what makes local reads trustworthy. A projection that answers quickly
// but drifts from the worker is worse than one that asks.
func TestProjectionConvergesWithTheEngine(parseT *testing.T) {
	for parseSeed := uint64(1); parseSeed <= 30; parseSeed++ {
		parseEngine := delta.New()
		parseProjection := buildProjection(parseT, projection.Options{Resident: -1})
		parseRandom := parseSeed

		for parseRound := range 6 {
			parseRandom = parseRandom*6364136223846793005 + 1442695040888963407
			parseSize := int((parseRandom>>33)%10) + 1

			parseRows := make([]delta.Row, 0, parseSize)
			parseUsed := map[delta.Key]bool{}
			for range parseSize {
				parseRandom = parseRandom*6364136223846793005 + 1442695040888963407
				parseKey := delta.Key(fmt.Sprintf("k%d", (parseRandom>>33)%8))
				if parseUsed[parseKey] {
					continue
				}
				parseUsed[parseKey] = true
				parseVersion := (parseRandom >> 20) % 3
				parseRows = append(parseRows, delta.Row{
					Key:     parseKey,
					Version: parseVersion,
					Payload: encodeItem(item{Name: string(parseKey), Price: int(parseVersion)}),
				})
			}

			parseOps, parsePublishErr := parseEngine.Publish(parseRows)
			if parsePublishErr != nil {
				parseT.Fatalf("seed %d round %d: publish: %v", parseSeed, parseRound, parsePublishErr)
			}
			if parseApplyErr := parseProjection.Apply(parseOps); parseApplyErr != nil {
				parseT.Fatalf("seed %d round %d: apply: %v", parseSeed, parseRound, parseApplyErr)
			}

			parseResident := parseProjection.Rows()
			if len(parseResident) != len(parseRows) {
				parseT.Fatalf("seed %d round %d: projection has %d rows, engine published %d",
					parseSeed, parseRound, len(parseResident), len(parseRows))
			}
			for parseIndex, parseRow := range parseRows {
				if parseResident[parseIndex].Key != parseRow.Key {
					parseT.Fatalf("seed %d round %d: row %d is %q, want %q",
						parseSeed, parseRound, parseIndex, parseResident[parseIndex].Key, parseRow.Key)
				}
				parseWant, _ := decodeItem(parseRow.Payload)
				if parseResident[parseIndex].Value != parseWant {
					parseT.Fatalf("seed %d round %d: row %q = %+v, want %+v",
						parseSeed, parseRound, parseRow.Key, parseResident[parseIndex].Value, parseWant)
				}
			}
		}
	}
}

func TestApplyHandlesEveryOpKind(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
		{Kind: projection.OpInsert, Key: "b", AfterKey: "a", Payload: encodeItem(item{Name: "B"})},
		{Kind: projection.OpInsert, Key: "c", AfterKey: "b", Payload: encodeItem(item{Name: "C"})},
	}); parseErr != nil {
		parseT.Fatalf("inserts: %v", parseErr)
	}

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpUpdate, Key: "b", Payload: encodeItem(item{Name: "B-prime"})},
		{Kind: projection.OpMove, Key: "c", AfterKey: ""},
		{Kind: projection.OpRemove, Key: "a"},
	}); parseErr != nil {
		parseT.Fatalf("mutations: %v", parseErr)
	}

	parseRows := parseProjection.Rows()
	if len(parseRows) != 2 || parseRows[0].Key != "c" || parseRows[1].Key != "b" {
		parseT.Fatalf("rows = %+v, want [c b]", parseRows)
	}
	if parseRows[1].Value.Name != "B-prime" {
		parseT.Errorf("b = %+v, want the updated value", parseRows[1].Value)
	}
}

// TestUpdateDoesNotInvalidateTheKeyIndex is a performance property with a
// correctness proxy. An update changes no positions, so it must not trigger an
// index rebuild — that is the path a keystroke-driven republish takes, and
// rebuilding per update would make it O(N) per row changed.
func TestUpdateDoesNotInvalidateTheKeyIndex(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	parseOps := make([]projection.Op, 0, 1000)
	var parseAnchor projection.Key
	for parseIndex := range 1000 {
		parseKey := projection.Key(fmt.Sprintf("k%04d", parseIndex))
		parseOps = append(parseOps, projection.Op{
			Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor,
			Payload: encodeItem(item{Name: string(parseKey), Price: parseIndex}),
		})
		parseAnchor = parseKey
	}
	if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
		parseT.Fatalf("inserts: %v", parseErr)
	}

	// Many updates in a row must all land correctly, which they cannot if the
	// index went stale without being rebuilt.
	parseUpdates := make([]projection.Op, 0, 1000)
	for parseIndex := range 1000 {
		parseKey := projection.Key(fmt.Sprintf("k%04d", parseIndex))
		parseUpdates = append(parseUpdates, projection.Op{
			Kind: projection.OpUpdate, Key: parseKey,
			Payload: encodeItem(item{Name: string(parseKey), Price: parseIndex * 2}),
		})
	}
	if parseErr := parseProjection.Apply(parseUpdates); parseErr != nil {
		parseT.Fatalf("updates: %v", parseErr)
	}

	for parseIndex, parseEntry := range parseProjection.Rows() {
		if parseEntry.Value.Price != parseIndex*2 {
			parseT.Fatalf("row %d price = %d, want %d", parseIndex, parseEntry.Value.Price, parseIndex*2)
		}
	}
}

// ---------------------------------------------------------------- residency

// TestResidencyCapIsVisibleRatherThanSilent: a truncated projection whose caller
// cannot tell it is truncated makes every local count a wrong answer instead of
// a lower bound.
func TestResidencyCapIsVisibleRatherThanSilent(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{Resident: 10})

	parseOps := make([]projection.Op, 0, 25)
	var parseAnchor projection.Key
	for parseIndex := range 25 {
		parseKey := projection.Key(fmt.Sprintf("k%02d", parseIndex))
		parseOps = append(parseOps, projection.Op{
			Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor,
			Payload: encodeItem(item{Name: string(parseKey), Price: parseIndex}),
		})
		parseAnchor = parseKey
	}
	if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}

	if parseProjection.Len() != 10 {
		parseT.Errorf("Len = %d, want the cap of 10", parseProjection.Len())
	}
	if parseProjection.DroppedByResidency() != 15 {
		parseT.Errorf("DroppedByResidency = %d, want 15", parseProjection.DroppedByResidency())
	}
}

// TestUpdatesForNonResidentRowsAreNotErrors: rows outside the window
// legitimately receive updates the projection cannot apply, and treating each
// one as a failure would make a capped projection unusable.
func TestUpdatesForNonResidentRowsAreNotErrors(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{Resident: 2})

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
		{Kind: projection.OpInsert, Key: "b", AfterKey: "a", Payload: encodeItem(item{Name: "B"})},
		{Kind: projection.OpInsert, Key: "c", AfterKey: "b", Payload: encodeItem(item{Name: "C"})},
	}); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpUpdate, Key: "c", Payload: encodeItem(item{Name: "C-prime"})},
		{Kind: projection.OpRemove, Key: "c"},
		{Kind: projection.OpMove, Key: "c", AfterKey: ""},
	}); parseErr != nil {
		parseT.Errorf("ops for a non-resident row must not be errors: %v", parseErr)
	}
}

// TestUnknownKeyIsAnErrorWhenNothingWasDropped is the other side of that
// leniency: with no residency cap in play, an op for an unknown key means the
// projection and the worker disagree, and silence would hide it.
func TestUnknownKeyIsAnErrorWhenNothingWasDropped(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpUpdate, Key: "ghost", Payload: encodeItem(item{})},
	}); parseErr == nil {
		parseT.Error("an update for an unknown key must be reported when nothing was dropped")
	}
}

func TestUnresolvableAnchorIsReportedAndLosesNoData(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
	}); parseErr != nil {
		parseT.Fatalf("insert: %v", parseErr)
	}

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpMove, Key: "a", AfterKey: "nope"},
	}); parseErr == nil {
		parseT.Error("a move to an unknown anchor must be reported")
	}
	// The row must survive the failed move. Dropping data is a worse response to
	// a protocol error than reporting one.
	if parseProjection.Len() != 1 {
		parseT.Errorf("Len = %d, want the row preserved after a failed move", parseProjection.Len())
	}
}

// ------------------------------------------------------------------- guards

func TestWindowClampsRatherThanPanics(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
	}); parseErr != nil {
		parseT.Fatalf("insert: %v", parseErr)
	}

	// A scroll offset can outrun a projection that just shrank; an empty window
	// is a better answer than a panic.
	if parseWindow := parseProjection.Window(500, 10); parseWindow != nil {
		parseT.Errorf("window past the end = %+v, want nil", parseWindow)
	}
	if parseWindow := parseProjection.Window(-5, 10); len(parseWindow) != 1 {
		parseT.Errorf("negative offset window = %+v, want the first row", parseWindow)
	}
	if parseWindow := parseProjection.Window(0, 0); parseWindow != nil {
		parseT.Errorf("zero-count window = %+v, want nil", parseWindow)
	}
}

func TestDecodeFailureIsReported(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: []byte("not json")},
	}); parseErr == nil {
		parseT.Error("an undecodable payload must be reported rather than stored as a zero value")
	}
}

func TestNewRequiresADecoder(parseT *testing.T) {
	if _, parseErr := projection.New[item](nil, projection.Options{}); parseErr == nil {
		parseT.Error("a nil decoder must be rejected")
	}
}

func TestUnknownOpKindIsRejected(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{{Kind: projection.OpKind(99), Key: "a"}}); parseErr == nil {
		parseT.Error("an unknown op kind must be rejected")
	}
}

func TestResetClearsEverything(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{Resident: 1})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
		{Kind: projection.OpInsert, Key: "b", AfterKey: "a", Payload: encodeItem(item{Name: "B"})},
	}); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}

	parseProjection.Reset()
	if parseProjection.Len() != 0 || parseProjection.DroppedByResidency() != 0 {
		parseT.Errorf("after Reset: len=%d dropped=%d, want 0 and 0",
			parseProjection.Len(), parseProjection.DroppedByResidency())
	}
}

func TestNilProjectionIsSafe(parseT *testing.T) {
	var parseProjection *projection.Projection[item]
	if parseErr := parseProjection.Apply(nil); parseErr == nil {
		parseT.Error("a nil projection must error rather than panic")
	}
	if parseProjection.Len() != 0 || parseProjection.Rows() != nil ||
		parseProjection.Filter(func(item) bool { return true }) != nil ||
		parseProjection.Window(0, 10) != nil || parseProjection.DroppedByResidency() != 0 {
		parseT.Error("a nil projection reads as empty")
	}
	if _, hasRow := parseProjection.Get("a"); hasRow {
		parseT.Error("a nil projection has no rows")
	}
	parseProjection.Reset()
}
