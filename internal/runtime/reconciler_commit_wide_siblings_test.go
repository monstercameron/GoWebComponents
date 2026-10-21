package runtime

import (
	"fmt"
	"runtime/debug"
	"testing"
)

// buildWideSiblingHost builds one host element with parseCount direct children.
// Every fourth child is a function component with an effect so the effect walk
// is exercised as well as the DOM commit walk.
func buildWideSiblingHost(parseCount int, parseLabel string) *Element {
	parseEffectChild := func(parseProps map[string]any) *Element {
		GoUseEffect(func() func() { return func() {} })
		return CreateElement("li", nil, parseLabel)
	}
	parseChildren := make([]any, 0, parseCount)
	for parseIndex := range parseCount {
		if parseIndex%4 == 0 {
			parseChildren = append(parseChildren, CreateElement(parseEffectChild, map[string]any{"key": fmt.Sprintf("k%d", parseIndex)}))
			continue
		}
		parseChildren = append(parseChildren, CreateElement("li", map[string]any{"key": fmt.Sprintf("k%d", parseIndex)}, parseLabel))
	}
	return CreateElement("ul", nil, parseChildren...)
}

// TestCommitWideSiblingListsDoNotRecursePerSibling pins that commit, effect, and
// deletion walks are iterative across siblings. Under js/wasm the JS stack
// overflowed near 5,000 siblings; here a small goroutine stack limit makes a
// per-sibling recursion fatal long before 10,000 siblings.
func TestCommitWideSiblingListsDoNotRecursePerSibling(parseT *testing.T) {
	parsePrevMax := debug.SetMaxStack(1 << 20)
	defer debug.SetMaxStack(parsePrevMax)

	for _, parseCount := range []int{10000, 20000} {
		parseT.Run(fmt.Sprint(parseCount), func(parseT *testing.T) {
			parseAdapter := newTestDOMAdapter()
			parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
			parseContainer := parseAdapter.CreateElement("section")

			parseRt.Render(buildWideSiblingHost(parseCount, "a"), parseContainer)
			parseRt.Render(buildWideSiblingHost(parseCount, "b"), parseContainer)
			parseRt.Render(CreateElement("ul", nil), parseContainer)
		})
	}
}

// TestCommitHundredSiblingAllocations records the allocation cost of mounting
// 100 siblings so the iterative walk can be proven allocation-neutral.
func TestCommitHundredSiblingAllocations(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("section")
	parseElement := buildWideSiblingHost(100, "a")
	parseRt.Render(parseElement, parseContainer)
	parseAllocs := testing.AllocsPerRun(20, func() {
		parseRt.Render(parseElement, parseContainer)
	})
	parseT.Logf("100-sibling re-render allocs/op = %.0f", parseAllocs)
}

// TestCreateSkipsEmptyClassAndStyleAttributes pins that a freshly created
// element does not receive class="" or style="" while other empty-valued
// attributes keep their meaning, and that a later non-empty class still lands.
func TestCreateSkipsEmptyClassAndStyleAttributes(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("section")

	parseBuild := func(parseClass string) *Element {
		return CreateElement("div", nil,
			CreateElement("p", map[string]any{"class": parseClass, "style": "", "id": "a", "title": "", "aria-label": ""}, "x"),
			CreateElement("p", map[string]any{"className": parseClass, "data-empty": ""}, CreateElement("b", nil, "y")),
		)
	}
	parseRt.Render(parseBuild(""), parseContainer)

	parseRoot := parseContainer.(*testDOMNode).children[0].(*testDOMNode)
	for parseIndex, parseChild := range parseRoot.children {
		parseNode := parseChild.(*testDOMNode)
		for _, parseName := range []string{"class", "style"} {
			if parseValue, parseHas := parseNode.attributes[parseName]; parseHas {
				parseT.Errorf("child %d wrote %s=%q on create", parseIndex, parseName, parseValue)
			}
		}
	}
	parseFirst := parseRoot.children[0].(*testDOMNode)
	for _, parseName := range []string{"title", "aria-label"} {
		if parseValue, parseHas := parseFirst.attributes[parseName]; !parseHas || parseValue != "" {
			parseT.Errorf("empty %s must still be written, got %q present=%v", parseName, parseValue, parseHas)
		}
	}
	if _, parseHas := parseRoot.children[1].(*testDOMNode).attributes["data-empty"]; !parseHas {
		parseT.Error("empty data-empty attribute must still be written")
	}

	parseRt.Render(parseBuild("on"), parseContainer)
	if parseValue := parseFirst.attributes["class"]; parseValue != "on" {
		parseT.Errorf("class after update = %q, want on", parseValue)
	}
}
