package runtime

import (
	"fmt"
	"sync"
	"testing"
)

// TestAtomRegistryConcurrentInvariants hammers the atom registry from many
// goroutines (the real concurrency surface: loaders and workers write atoms
// while the render goroutine reads) and verifies the invariants that the
// derived-generation fix and snapshot serialization are supposed to provide:
// every read observes some written value (no torn or zero reads), and at
// quiescence each derived atom equals its function of the final source value.
func TestAtomRegistryConcurrentInvariants(parseT *testing.T) {
	parseReg := NewAtomRegistry()
	const parseWriters = 8
	const parseWritesPerWriter = 500

	parseReg.InitAtom("src", 0)
	if parseErr := parseReg.RegisterDerivedAtom("doubled", []string{"src"}, func() interface{} {
		parseVal, _ := parseReg.GetAtom("src")
		parseInt, _ := parseVal.(int)
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("register derived: %v", parseErr)
	}

	var parseWg sync.WaitGroup
	parseSeen := make([]map[int]bool, parseWriters)
	for parseW := 0; parseW < parseWriters; parseW++ {
		parseW2 := parseW
		parseSeen[parseW2] = map[int]bool{}
		parseWg.Add(1)
		go func() {
			defer parseWg.Done()
			for parseI := 0; parseI < parseWritesPerWriter; parseI++ {
				parseVal := parseW2*parseWritesPerWriter + parseI + 1
				parseReg.setAtomAndNotify("src", parseVal, func(*Fiber) {})
				parseGot, parseOk := parseReg.GetAtom("src")
				if !parseOk {
					parseT.Errorf("writer %d: atom vanished mid-run", parseW2)
					return
				}
				parseInt, parseOk := parseGot.(int)
				if !parseOk {
					parseT.Errorf("writer %d: torn/typeless read %T", parseW2, parseGot)
					return
				}
				parseSeen[parseW2][parseInt] = true
			}
		}()
	}
	parseWg.Wait()

	// Quiescent invariant: derived equals exactly 2x the final source value.
	parseFinal, _ := parseReg.GetAtom("src")
	parseFinalInt, parseOk := parseFinal.(int)
	if !parseOk || parseFinalInt < 1 || parseFinalInt > parseWriters*parseWritesPerWriter {
		parseT.Fatalf("final source value out of range: %v", parseFinal)
	}
	parseDerived, _ := parseReg.GetAtom("doubled")
	parseDerivedInt, parseOk := parseDerived.(int)
	if !parseOk {
		parseT.Fatalf("derived atom torn: %T %v", parseDerived, parseDerived)
	}
	if parseDerivedInt != parseFinalInt*2 {
		parseT.Fatalf("derived invariant violated at quiescence: src=%d doubled=%d (stale recompute overwrote fresher value?)",
			parseFinalInt, parseDerivedInt)
	}

	// Every value any goroutine read back must be a value some goroutine wrote.
	for parseW, parseValues := range parseSeen {
		for parseVal := range parseValues {
			if parseVal < 1 || parseVal > parseWriters*parseWritesPerWriter {
				parseT.Fatalf("writer %d observed impossible value %d", parseW, parseVal)
			}
		}
	}
	_ = fmt.Sprintf
}
