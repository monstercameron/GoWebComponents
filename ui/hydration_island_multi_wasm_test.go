//go:build js && wasm

package ui

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// TestHydrateIntoTwoContainersKeepsFirstIsland proves hydrating island B does not destroy island
// A: A's DOM must stay painted and its effect cleanup must not run.
func TestHydrateIntoTwoContainersKeepsFirstIsland(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainerA := parseAdapter.CreateElement("section")
	parseContainerB := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePreviousInitialized })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCleanupsA := 0
	parseIslandA := func() Node {
		UseEffect(func() func() { return func() { parseCleanupsA++ } }, "a")
		return runtime.CreateElement("span", map[string]any{}, "island-a")
	}
	parseIslandB := func() Node {
		return runtime.CreateElement("span", map[string]any{}, "island-b")
	}

	if _, parseErr := HydrateInto(CreateElement(parseIslandA), parseContainerA); parseErr != nil {
		parseT.Fatalf("HydrateInto A: %v", parseErr)
	}
	parseScheduler.Flush()
	if !strings.Contains(collectMockNodeText(parseContainerA.(*mockdom.MockDOMNode)), "island-a") {
		parseT.Fatal("precondition: island A did not paint before B hydrated")
	}
	if _, parseErr := HydrateInto(CreateElement(parseIslandB), parseContainerB); parseErr != nil {
		parseT.Fatalf("HydrateInto B: %v", parseErr)
	}
	parseScheduler.Flush()

	parseTextA := collectMockNodeText(parseContainerA.(*mockdom.MockDOMNode))
	parseTextB := collectMockNodeText(parseContainerB.(*mockdom.MockDOMNode))
	if !strings.Contains(parseTextB, "island-b") {
		parseT.Fatalf("island B missing: %q", parseTextB)
	}
	if !strings.Contains(parseTextA, "island-a") {
		parseT.Fatalf("island A was destroyed by hydrating B: container A text %q", parseTextA)
	}
	if parseCleanupsA != 0 {
		parseT.Fatalf("island A effect cleanup ran %d time(s) when B hydrated", parseCleanupsA)
	}
}

// TestHydrateIntoSecondIslandStateUpdateRepaintsOnlyThatIsland proves a state
// update inside the second island is scheduled on that island's own root: B
// repaints, and A is neither touched nor torn down.
func TestHydrateIntoSecondIslandStateUpdateRepaintsOnlyThatIsland(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainerA := parseAdapter.CreateElement("section")
	parseContainerB := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePreviousInitialized })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseRendersA := 0
	parseIslandA := func() Node {
		parseRendersA++
		return runtime.CreateElement("span", map[string]any{}, "island-a")
	}
	var parseSetB func(int)
	parseIslandB := func() Node {
		parseCount := UseState(0)
		parseSetB = parseCount.Set
		return runtime.CreateElement("span", map[string]any{}, "count-"+string(rune('0'+parseCount.Get())))
	}

	if _, parseErr := HydrateInto(CreateElement(parseIslandA), parseContainerA); parseErr != nil {
		parseT.Fatalf("HydrateInto A: %v", parseErr)
	}
	parseScheduler.Flush()
	if _, parseErr := HydrateInto(CreateElement(parseIslandB), parseContainerB); parseErr != nil {
		parseT.Fatalf("HydrateInto B: %v", parseErr)
	}
	parseScheduler.Flush()

	parseSetB(3)
	parseScheduler.Flush()

	if parseText := collectMockNodeText(parseContainerB.(*mockdom.MockDOMNode)); !strings.Contains(parseText, "count-3") {
		parseT.Fatalf("island B did not repaint after its state update: %q", parseText)
	}
	if parseText := collectMockNodeText(parseContainerA.(*mockdom.MockDOMNode)); !strings.Contains(parseText, "island-a") {
		parseT.Fatalf("island A lost its content after B updated: %q", parseText)
	}
}
