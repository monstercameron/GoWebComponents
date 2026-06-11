package runtime

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// TestHydrationRoundTripFuzz adversarially verifies the full SSR contract:
// a server-rendered DOM hydrated by a fresh runtime with the same element
// model must (1) reuse every existing node (identity, not just content),
// (2) produce zero hydration mismatch diagnostics, and (3) leave the
// hydrated fibers fully updatable — a post-hydration mutation must
// reconcile to exactly the new model.
func TestHydrationRoundTripFuzz(parseT *testing.T) {
	for parseSeed := int64(500); parseSeed < 516; parseSeed++ {
		parseSeed2 := parseSeed
		parseT.Run(fmt.Sprintf("seed-%d", parseSeed2), func(parseT2 *testing.T) {
			runHydrationRoundTrip(parseT2, parseSeed2)
		})
	}
}

func runHydrationRoundTrip(parseT *testing.T, parseSeed int64) {
	parseT.Helper()
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRng := rand.New(rand.NewSource(parseSeed))
	parseAdapter := newTestDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")

	// Random model, 1-8 rows.
	parseRows := []fuzzRowModel{}
	parseNextKey := 0
	for parseIdx := 0; parseIdx < 1+parseRng.Intn(8); parseIdx++ {
		parseRows = fuzzMutate(parseRng, parseRows, &parseNextKey)
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, fuzzRowModel{Key: parseNextKey, Text: "only", Class: "fresh"})
		parseNextKey++
	}

	// "Server" pass: a first runtime renders the model into the container,
	// standing in for SSR-produced markup with identical structure.
	parseServerRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseServerRt.Render(buildFuzzListElement(parseRows, true), parseContainer)

	// Capture server node identities.
	parseServerList := parseAdapter.GetChildren(parseContainer)[0]
	parseServerRowNodes := parseAdapter.GetChildren(parseServerList)

	// Client pass: a FRESH runtime hydrates the same model onto that DOM.
	parseScheduler := newTestScheduler()
	parseClientRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseClientRt.Hydrate(buildFuzzListElement(parseRows, true), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	// (1) Content must match the model exactly.
	parseGot := describeFuzzDOM(parseT, parseContainer)
	parseWant := describeFuzzModel(parseRows)
	for parseIdx := range parseWant {
		if parseIdx >= len(parseGot) || parseGot[parseIdx] != parseWant[parseIdx] {
			parseT.Fatalf("seed=%d: post-hydration DOM diverged at row %d:\n got: %v\nwant: %v", parseSeed, parseIdx, parseGot, parseWant)
		}
	}

	// (2) Node identity: hydration must have REUSED the server nodes.
	parseHydratedList := parseAdapter.GetChildren(parseContainer)[0]
	if !parseHydratedList.Equals(parseServerList) {
		parseT.Fatalf("seed=%d: hydration replaced the list node instead of reusing it", parseSeed)
	}
	parseHydratedRows := parseAdapter.GetChildren(parseHydratedList)
	if len(parseHydratedRows) != len(parseServerRowNodes) {
		parseT.Fatalf("seed=%d: hydration changed row count %d -> %d", parseSeed, len(parseServerRowNodes), len(parseHydratedRows))
	}
	for parseIdx := range parseServerRowNodes {
		if !parseHydratedRows[parseIdx].Equals(parseServerRowNodes[parseIdx]) {
			parseT.Fatalf("seed=%d: row %d node was replaced, not reused", parseSeed, parseIdx)
		}
	}

	// (3) Zero hydration mismatch diagnostics for a matching tree.
	for _, parseDiag := range GetDiagnostics() {
		if strings.Contains(parseDiag.Message, "hydration") && strings.Contains(parseDiag.Message, "mismatch") {
			parseT.Fatalf("seed=%d: unexpected hydration mismatch on identical tree: %s", parseSeed, parseDiag.Message)
		}
	}

	// (4) Post-hydration updatability: mutate and re-render on the client
	// runtime; the DOM must converge to the new model.
	for parseStep := 0; parseStep < 10; parseStep++ {
		parseRows = fuzzMutate(parseRng, parseRows, &parseNextKey)
		parseClientRt.Render(buildFuzzListElement(parseRows, true), parseContainer)
		for len(parseScheduler.timeouts) > 0 {
			parsePump := parseScheduler.timeouts[0]
			parseScheduler.timeouts = parseScheduler.timeouts[1:]
			parsePump()
		}
		parseGot := describeFuzzDOM(parseT, parseContainer)
		parseWant := describeFuzzModel(parseRows)
		if len(parseGot) != len(parseWant) {
			parseT.Fatalf("seed=%d step=%d: post-hydration update count %d != %d\n got: %v\nwant: %v",
				parseSeed, parseStep, len(parseGot), len(parseWant), parseGot, parseWant)
		}
		for parseIdx := range parseWant {
			if parseGot[parseIdx] != parseWant[parseIdx] {
				parseT.Fatalf("seed=%d step=%d row=%d: %q != %q", parseSeed, parseStep, parseIdx, parseGot[parseIdx], parseWant[parseIdx])
			}
		}
	}
}
