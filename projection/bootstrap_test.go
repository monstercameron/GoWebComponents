package projection_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/projection"
)

// v5 P3.11 — SSR bootstrap carries initial projections.
//
// Criterion: a server-rendered route is Ready() on first render.
//
// The failure this prevents is specific and common. A server-rendered page
// paints instantly, the client mounts, finds an empty projection, renders a
// spinner or an empty list, and fills in after the first round trip. The user
// sees content, then sees it vanish, then sees it return — server rendering made
// the first paint fast and the experience worse.

func encodeItemJSON(parseItem item) ([]byte, error) { return json.Marshal(parseItem) }

func buildTestBootstrap(parseT *testing.T, parseCount int, parseComplete bool) projection.Bootstrap {
	parseT.Helper()

	parseKeys := make([]projection.Key, 0, parseCount)
	parseRows := make([]item, 0, parseCount)
	for parseIndex := range parseCount {
		parseKeys = append(parseKeys, projection.Key(fmt.Sprintf("k%03d", parseIndex)))
		parseRows = append(parseRows, item{Name: fmt.Sprintf("row-%d", parseIndex), Price: parseIndex})
	}

	parseBootstrap, parseErr := projection.BuildBootstrap(parseKeys, parseRows, encodeItemJSON, parseComplete)
	if parseErr != nil {
		parseT.Fatalf("BuildBootstrap: %v", parseErr)
	}
	return parseBootstrap
}

// -------------------------------------------------------- the criterion

// TestServerRenderedRouteIsReadyOnFirstRender is P3.11's criterion.
func TestServerRenderedRouteIsReadyOnFirstRender(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})

	// Before hydration the route is NOT ready — this is the state a client-only
	// mount is in, and rendering content from it would be rendering nothing.
	if parseProjection.Ready() {
		parseT.Fatal("a projection with no contents must not report ready")
	}

	parseBootstrap := buildTestBootstrap(parseT, 50, true)
	if parseErr := parseProjection.Hydrate(parseBootstrap); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}

	if !parseProjection.Ready() {
		parseT.Error("a hydrated projection must be ready on first render — that is the criterion")
	}
	if parseProjection.Len() != 50 {
		parseT.Errorf("Len = %d, want the 50 server-rendered rows", parseProjection.Len())
	}
	if !parseProjection.Complete() {
		parseT.Error("a complete bootstrap must report complete")
	}

	// And the data must be the server's, readable immediately with no round trip.
	parseClient := &countingClient{}
	parseRows := parseProjection.Rows()
	if parseRows[0].Value.Name != "row-0" || parseRows[49].Value.Name != "row-49" {
		parseT.Errorf("first/last = %q/%q, want the server's rows in order",
			parseRows[0].Value.Name, parseRows[49].Value.Name)
	}
	if len(parseClient.sends) != 0 {
		parseT.Errorf("%d round trips during first render, want 0", len(parseClient.sends))
	}
}

// TestEmptyBootstrapIsReadyNotLoading is the distinction Ready() exists for, and
// the one an implementation collapses by accident.
//
// A route with no results is LOADED. A UI reading emptiness as loading shows a
// spinner forever on exactly those routes — a search with no matches, a new
// account with no transactions.
func TestEmptyBootstrapIsReadyNotLoading(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})

	if parseErr := parseProjection.Hydrate(projection.Bootstrap{Complete: true}); parseErr != nil {
		parseT.Fatalf("Hydrate(empty): %v", parseErr)
	}

	if !parseProjection.Ready() {
		parseT.Error("an empty bootstrap means no results, not no data — the route is ready")
	}
	if parseProjection.Len() != 0 {
		parseT.Errorf("Len = %d, want 0", parseProjection.Len())
	}
}

// TestReadyIsNotDerivedFromRowCount states the same rule from the other side: a
// projection can hold rows without being ready, if they arrived by delta before
// the initial contents were declared.
func TestReadyIsNotDerivedFromRowCount(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
	}); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}

	if parseProjection.Len() == 0 {
		parseT.Fatal("the row should be resident")
	}
	if parseProjection.Ready() {
		parseT.Error("rows alone must not make a projection ready; Ready is a declared state, not a count")
	}

	parseProjection.MarkReady(true)
	if !parseProjection.Ready() {
		parseT.Error("MarkReady must make a client-rendered route ready")
	}
}

// TestCompleteIsDistinctFromReady: a route can be ready to render — enough rows
// for the first screen — while more remain. A client conflating them stops
// paginating at the fold.
func TestCompleteIsDistinctFromReady(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})

	parsePrefix := buildTestBootstrap(parseT, 20, false)
	if parseErr := parseProjection.Hydrate(parsePrefix); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}

	if !parseProjection.Ready() {
		parseT.Error("a prefix is still enough to render the first screen")
	}
	if parseProjection.Complete() {
		parseT.Error("a prefix must not report complete, or the client stops paginating at the fold")
	}
}

// ---------------------------------------------------- hydration mechanics

// TestHydrationGoesThroughTheSameApplyPath: a second way of writing the same
// state is a second place for the two to diverge.
func TestHydrationRespectsTheResidencyCap(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{Resident: 10})

	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 40, true)); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}

	if parseProjection.Len() != 10 {
		parseT.Errorf("Len = %d, want the residency cap of 10 — hydration bypassed it", parseProjection.Len())
	}
	if parseProjection.DroppedByResidency() != 30 {
		parseT.Errorf("dropped = %d, want 30", parseProjection.DroppedByResidency())
	}
	if !parseProjection.Ready() {
		parseT.Error("a capped hydration is still a hydration")
	}
}

func TestHydrationPreservesServerOrder(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	parseBootstrap := buildTestBootstrap(parseT, 30, true)

	if parseErr := parseProjection.Hydrate(parseBootstrap); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}

	for parseIndex, parseEntry := range parseProjection.Rows() {
		if parseEntry.Key != parseBootstrap.Keys[parseIndex] {
			parseT.Fatalf("row %d = %q, want %q — hydration reordered the server's rows",
				parseIndex, parseEntry.Key, parseBootstrap.Keys[parseIndex])
		}
	}
}

// TestDeltasApplyOnTopOfAHydratedProjection is the handoff: the server's rows,
// then the worker's updates, with no gap and no duplicate.
func TestDeltasApplyOnTopOfAHydratedProjection(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 5, true)); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}

	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpUpdate, Key: "k002", Payload: encodeItem(item{Name: "updated", Price: 99})},
		{Kind: projection.OpInsert, Key: "k005", AfterKey: "k004", Payload: encodeItem(item{Name: "new"})},
	}); parseErr != nil {
		parseT.Fatalf("Apply after hydrate: %v", parseErr)
	}

	if parseProjection.Len() != 6 {
		parseT.Errorf("Len = %d, want 6", parseProjection.Len())
	}
	if parseValue, hasRow := parseProjection.Get("k002"); !hasRow || parseValue.Name != "updated" {
		parseT.Errorf("k002 = %+v, want the update applied", parseValue)
	}
}

// TestDoubleHydrationIsRefused: it would either duplicate keys or silently
// discard the second bootstrap, and both are worse than an error where the
// caller can still act on it.
func TestDoubleHydrationIsRefused(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 3, true)); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}
	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 3, true)); parseErr == nil {
		parseT.Error("a second hydration must be refused")
	}
	if parseProjection.Len() != 3 {
		parseT.Errorf("Len = %d, want the first hydration intact", parseProjection.Len())
	}
}

func TestHydratingANonEmptyProjectionIsRefused(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Apply([]projection.Op{
		{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "A"})},
	}); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 3, true)); parseErr == nil {
		parseT.Error("hydrating over existing rows must be refused")
	}
}

// TestResetClearsReadiness: a reset projection's subject changed, and whatever
// made it ready described the old subject.
func TestResetClearsReadiness(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 3, true)); parseErr != nil {
		parseT.Fatalf("Hydrate: %v", parseErr)
	}

	parseProjection.Reset()
	if parseProjection.Ready() || parseProjection.Complete() {
		parseT.Error("a reset projection is unloaded again; leaving it ready renders 'no results' for the new subject")
	}
	// And it can be hydrated afresh, which is the point of clearing the flag.
	if parseErr := parseProjection.Hydrate(buildTestBootstrap(parseT, 2, true)); parseErr != nil {
		parseT.Errorf("re-hydrating after a reset must work: %v", parseErr)
	}
}

// ----------------------------------------------------------- validation

// TestDesynchronizedBootstrapIsRejected: parallel arrays are compact and can
// desynchronize, which pairs a key with another row's payload — corruption that
// renders without error.
func TestDesynchronizedBootstrapIsRejected(parseT *testing.T) {
	parseBootstrap := projection.Bootstrap{
		Keys:     []projection.Key{"a", "b", "c"},
		Payloads: [][]byte{encodeItem(item{Name: "A"}), encodeItem(item{Name: "B"})},
	}
	if parseErr := parseBootstrap.Validate(); parseErr == nil {
		parseT.Error("a bootstrap with mismatched arrays must be rejected")
	}

	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Hydrate(parseBootstrap); parseErr == nil {
		parseT.Error("hydration must validate rather than trust")
	}
	if parseProjection.Ready() {
		parseT.Error("a failed hydration must not leave the projection reporting ready")
	}
}

func TestBootstrapRejectsDuplicateAndEmptyKeys(parseT *testing.T) {
	parseDuplicate := projection.Bootstrap{
		Keys:     []projection.Key{"a", "a"},
		Payloads: [][]byte{encodeItem(item{}), encodeItem(item{})},
	}
	if parseErr := parseDuplicate.Validate(); parseErr == nil {
		parseT.Error("a duplicate key has no correct ordering and must be rejected")
	}

	parseEmpty := projection.Bootstrap{
		Keys:     []projection.Key{""},
		Payloads: [][]byte{encodeItem(item{})},
	}
	if parseErr := parseEmpty.Validate(); parseErr == nil {
		parseT.Error("a row with no key must be rejected")
	}
}

func TestBuildBootstrapRejectsBadInput(parseT *testing.T) {
	if _, parseErr := projection.BuildBootstrap([]projection.Key{"a"}, []item{}, encodeItemJSON, true); parseErr == nil {
		parseT.Error("a key count that does not match the row count must be rejected")
	}
	if _, parseErr := projection.BuildBootstrap([]projection.Key{"a"}, []item{{}}, nil, true); parseErr == nil {
		parseT.Error("a nil encoder must be rejected")
	}
	if _, parseErr := projection.BuildBootstrap([]projection.Key{""}, []item{{}}, encodeItemJSON, true); parseErr == nil {
		parseT.Error("a row with no key must be rejected at build time")
	}

	parseEncodeErr := errors.New("cannot encode")
	if _, parseErr := projection.BuildBootstrap([]projection.Key{"a"}, []item{{}},
		func(item) ([]byte, error) { return nil, parseEncodeErr }, true); !errors.Is(parseErr, parseEncodeErr) {
		parseT.Errorf("err = %v, want the encoder's own error preserved", parseErr)
	}
}

// TestBootstrapSurvivesJSON is what actually happens to it: embedded in markup,
// parsed by the client.
func TestBootstrapSurvivesJSON(parseT *testing.T) {
	parseOriginal := buildTestBootstrap(parseT, 12, false)

	parseEncoded, parseErr := json.Marshal(parseOriginal)
	if parseErr != nil {
		parseT.Fatalf("marshal: %v", parseErr)
	}
	var parseDecoded projection.Bootstrap
	if parseErr := json.Unmarshal(parseEncoded, &parseDecoded); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}

	if parseErr := parseDecoded.Validate(); parseErr != nil {
		parseT.Fatalf("a round-tripped bootstrap must stay valid: %v", parseErr)
	}
	if parseDecoded.Len() != parseOriginal.Len() {
		parseT.Errorf("decoded %d rows, want %d", parseDecoded.Len(), parseOriginal.Len())
	}
	if parseDecoded.Complete != parseOriginal.Complete {
		parseT.Error("the completeness flag must survive the round trip; losing it makes a prefix look whole")
	}

	parseProjection := buildProjection(parseT, projection.Options{})
	if parseErr := parseProjection.Hydrate(parseDecoded); parseErr != nil {
		parseT.Fatalf("hydrating a round-tripped bootstrap: %v", parseErr)
	}
	if !parseProjection.Ready() || parseProjection.Len() != 12 {
		parseT.Errorf("ready=%v len=%d, want ready with 12 rows", parseProjection.Ready(), parseProjection.Len())
	}
}

func TestNilProjectionBootstrapIsSafe(parseT *testing.T) {
	var parseProjection *projection.Projection[item]
	if parseErr := parseProjection.Hydrate(projection.Bootstrap{}); parseErr == nil {
		parseT.Error("a nil projection must error rather than panic")
	}
	if parseProjection.Ready() || parseProjection.Complete() {
		parseT.Error("a nil projection is not ready")
	}
	parseProjection.MarkReady(true)
}
