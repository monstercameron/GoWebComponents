//go:build !(js && wasm)

package css

import "testing"

// TestRegisterAndEmitDetectsHashCollision pins the #84 collision verification:
// two distinct rule-sets that hash to the same class name (different CSS) are
// reported instead of silently applying the first's styles to the second's
// element. An identical re-emit and a Seeded class must NOT false-report.
func TestRegisterAndEmitDetectsHashCollision(parseT *testing.T) {
	parsePrev := reportClassHashCollision
	defer func() { reportClassHashCollision = parsePrev }()

	var parseCollided []string
	reportClassHashCollision = func(parseClass string) { parseCollided = append(parseCollided, parseClass) }

	// Same class, DIFFERENT css → collision reported.
	Reset()
	parseCollided = nil
	registerAndEmit("c-forced", ".c-forced{color:red}")
	registerAndEmit("c-forced", ".c-forced{color:blue}")
	if len(parseCollided) != 1 || parseCollided[0] != "c-forced" {
		parseT.Fatalf("expected one collision report for c-forced, got %v", parseCollided)
	}

	// Same class, IDENTICAL css → benign fold, no collision.
	Reset()
	parseCollided = nil
	registerAndEmit("c-dup", ".c-dup{color:red}")
	registerAndEmit("c-dup", ".c-dup{color:red}")
	if len(parseCollided) != 0 {
		parseT.Fatalf("identical CSS re-emit must not report a collision, got %v", parseCollided)
	}

	// A Seeded class (registered without CSS) then emitted must NOT false-report.
	Reset()
	parseCollided = nil
	Seed("c-seed")
	registerAndEmit("c-seed", ".c-seed{color:green}")
	if len(parseCollided) != 0 {
		parseT.Fatalf("seeded class must not report a collision, got %v", parseCollided)
	}
}
