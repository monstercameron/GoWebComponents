package runtime

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// fuzzRowModel is the independent source of truth the DOM is verified against.
type fuzzRowModel struct {
	Key   int
	Text  string
	Class string
}

// buildFuzzListElement renders the model through the public element API.
func buildFuzzListElement(parseRows []fuzzRowModel, isKeyed bool) *Element {
	parseChildren := make([]interface{}, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseProps := map[string]interface{}{"class": parseRow.Class, "data-k": fmt.Sprintf("%d", parseRow.Key)}
		if isKeyed {
			parseProps["key"] = parseRow.Key
		}
		parseChildren = append(parseChildren, CreateElement("li", parseProps, parseRow.Text))
	}
	return CreateElement("ul", map[string]interface{}{"id": "fuzz-list"}, parseChildren...)
}

// describeFuzzDOM walks the committed test DOM and flattens it for comparison.
func describeFuzzDOM(parseT *testing.T, parseContainer DOMNode) []string {
	parseT.Helper()
	parseRoot, parseOk := parseContainer.(*testDOMNode)
	if !parseOk {
		parseT.Fatalf("container is %T, not *testDOMNode", parseContainer)
	}
	if len(parseRoot.children) != 1 {
		parseT.Fatalf("expected exactly one <ul> under container, got %d children", len(parseRoot.children))
	}
	parseList, parseOk := parseRoot.children[0].(*testDOMNode)
	if !parseOk || parseList.tag != "ul" {
		parseT.Fatalf("expected <ul> root, got %+v", parseRoot.children[0])
	}
	parseRows := make([]string, 0, len(parseList.children))
	for _, parseChild := range parseList.children {
		parseRow, parseOk := parseChild.(*testDOMNode)
		if !parseOk || parseRow.tag != "li" {
			parseT.Fatalf("expected <li> child, got %+v", parseChild)
		}
		parseText := parseRow.text
		if parseText == "" {
			// Text may live in a child text node rather than direct text.
			parseParts := make([]string, 0, len(parseRow.children))
			for _, parseTextChild := range parseRow.children {
				if parseTextNode, parseOk := parseTextChild.(*testDOMNode); parseOk {
					parseParts = append(parseParts, parseTextNode.text)
				}
			}
			parseText = strings.Join(parseParts, "")
		}
		parseRows = append(parseRows, fmt.Sprintf("k=%s class=%s text=%s", parseRow.attributes["data-k"], parseRow.attributes["class"], parseText))
	}
	return parseRows
}

// describeFuzzModel flattens the model the same way for comparison.
func describeFuzzModel(parseRows []fuzzRowModel) []string {
	parseOut := make([]string, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseOut = append(parseOut, fmt.Sprintf("k=%d class=%s text=%s", parseRow.Key, parseRow.Class, parseRow.Text))
	}
	return parseOut
}

// fuzzMutate applies one random structural or content mutation to the model.
func fuzzMutate(parseRng *rand.Rand, parseRows []fuzzRowModel, parseNextKey *int) []fuzzRowModel {
	parseOp := parseRng.Intn(6)
	switch parseOp {
	case 0: // insert at random position
		parseIdx := parseRng.Intn(len(parseRows) + 1)
		parseRow := fuzzRowModel{Key: *parseNextKey, Text: fmt.Sprintf("row-%d", *parseNextKey), Class: "fresh"}
		*parseNextKey++
		parseRows = append(parseRows, fuzzRowModel{})
		copy(parseRows[parseIdx+1:], parseRows[parseIdx:])
		parseRows[parseIdx] = parseRow
	case 1: // remove at random position
		if len(parseRows) > 0 {
			parseIdx := parseRng.Intn(len(parseRows))
			parseRows = append(parseRows[:parseIdx], parseRows[parseIdx+1:]...)
		}
	case 2: // swap two rows
		if len(parseRows) >= 2 {
			parseA, parseB := parseRng.Intn(len(parseRows)), parseRng.Intn(len(parseRows))
			parseRows[parseA], parseRows[parseB] = parseRows[parseB], parseRows[parseA]
		}
	case 3: // update text
		if len(parseRows) > 0 {
			parseIdx := parseRng.Intn(len(parseRows))
			parseRows[parseIdx].Text = fmt.Sprintf("upd-%d-%d", parseRows[parseIdx].Key, parseRng.Intn(1000))
		}
	case 4: // update class
		if len(parseRows) > 0 {
			parseIdx := parseRng.Intn(len(parseRows))
			parseRows[parseIdx].Class = fmt.Sprintf("c%d", parseRng.Intn(5))
		}
	case 5: // reverse the whole list (worst-case reorder)
		for parseI, parseJ := 0, len(parseRows)-1; parseI < parseJ; parseI, parseJ = parseI+1, parseJ-1 {
			parseRows[parseI], parseRows[parseJ] = parseRows[parseJ], parseRows[parseI]
		}
	}
	return parseRows
}

// runReconcilerFuzz drives random render sequences and verifies the committed
// DOM against the independently-maintained model after every single commit.
func runReconcilerFuzz(parseT *testing.T, parseSeed int64, parseSteps int, isKeyed bool) {
	parseT.Helper()
	parseRng := rand.New(rand.NewSource(parseSeed))
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseRows := []fuzzRowModel{}
	parseNextKey := 0
	for parseIdx := 0; parseIdx < 4; parseIdx++ {
		parseRows = fuzzMutate(parseRng, parseRows, &parseNextKey)
	}

	for parseStep := 0; parseStep < parseSteps; parseStep++ {
		parseRows = fuzzMutate(parseRng, parseRows, &parseNextKey)
		parseRt.Render(buildFuzzListElement(parseRows, isKeyed), parseContainer)

		parseGot := describeFuzzDOM(parseT, parseContainer)
		parseWant := describeFuzzModel(parseRows)
		if len(parseGot) != len(parseWant) {
			parseT.Fatalf("seed=%d keyed=%v step=%d: DOM has %d rows, model has %d\n got: %v\nwant: %v",
				parseSeed, isKeyed, parseStep, len(parseGot), len(parseWant), parseGot, parseWant)
		}
		for parseIdx := range parseWant {
			if parseGot[parseIdx] != parseWant[parseIdx] {
				parseT.Fatalf("seed=%d keyed=%v step=%d row=%d:\n got: %q\nwant: %q\nfull got: %v\nfull want: %v",
					parseSeed, isKeyed, parseStep, parseIdx, parseGot[parseIdx], parseWant[parseIdx], parseGot, parseWant)
			}
		}
	}
}

// TestReconcilerFuzzKeyedListCorrectness adversarially verifies keyed
// reconciliation: every commit's DOM must exactly match the model.
func TestReconcilerFuzzKeyedListCorrectness(parseT *testing.T) {
	for parseSeed := int64(1); parseSeed <= 8; parseSeed++ {
		parseT.Run(fmt.Sprintf("seed-%d", parseSeed), func(parseT2 *testing.T) {
			runReconcilerFuzz(parseT2, parseSeed, 120, true)
		})
	}
}

// TestReconcilerFuzzUnkeyedListCorrectness does the same for the positional
// (unkeyed) reconciliation path.
func TestReconcilerFuzzUnkeyedListCorrectness(parseT *testing.T) {
	for parseSeed := int64(101); parseSeed <= 108; parseSeed++ {
		parseT.Run(fmt.Sprintf("seed-%d", parseSeed), func(parseT2 *testing.T) {
			runReconcilerFuzz(parseT2, parseSeed, 120, false)
		})
	}
}
