//go:build !(js && wasm)

package css

import (
	"strings"
	"testing"
)

// TestInjectSameIDSemantics pins idempotent same-body injects and first-wins for
// a different body under one id without a false class-collision diagnostic.
func TestInjectSameIDSemantics(parseT *testing.T) {
	Reset()
	Inject("slot", `.a{color:red}`)
	Inject("slot", `.a{color:red}`)
	if parseGot := strings.Count(Harvest(), ".a{color:red}"); parseGot != 1 {
		parseT.Fatalf("same body must be idempotent, got %d copies", parseGot)
	}
	parseReports := 0
	parsePrev := reportClassHashCollision
	defer func() { reportClassHashCollision = parsePrev }()
	reportClassHashCollision = func(string) { parseReports++ }
	Inject("slot", `.a{color:blue}`)
	parseOut := Harvest()
	if !strings.Contains(parseOut, ".a{color:red}") || strings.Contains(parseOut, "blue") {
		parseT.Fatalf("different body must be ignored without replacing, got %q", parseOut)
	}
	if parseReports != 0 {
		parseT.Fatalf("replacement must not emit diagnostics, got %d", parseReports)
	}
}

// TestGridLineAtZeroClamped pins that the nonexistent grid line 0 is not emitted.
func TestGridLineAtZeroClamped(parseT *testing.T) {
	if parseGot := GridLineAt(0).String(); parseGot != "1" {
		parseT.Fatalf("GridLineAt(0) = %q, want 1", parseGot)
	}
	if parseGot := GridLineAt(-2).String(); parseGot != "-2" {
		parseT.Fatalf("GridLineAt(-2) = %q", parseGot)
	}
}