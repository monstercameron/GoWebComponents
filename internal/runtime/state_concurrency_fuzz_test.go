package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestAtomRegistryTracksOnlyActiveDerivedDependencies(parseT *testing.T) {
	parseReg := NewAtomRegistry()
	parseReg.InitAtom("src", 0)
	if len(parseReg.atomVersions) != 0 {
		parseT.Fatalf("untracked source unexpectedly has a revision: %v", parseReg.atomVersions)
	}
	if parseErr := parseReg.RegisterDerivedAtom("doubled", []string{"src"}, func() any {
		parseValue, _ := parseReg.GetAtom("src")
		parseInt, _ := parseValue.(int)
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("register derived: %v", parseErr)
	}
	parseInitialRevision := *parseReg.atomVersions["src"]
	if parseInitialRevision == 0 || len(parseReg.atomVersions) != 1 {
		parseT.Fatalf("expected only src to be tracked, got %v", parseReg.atomVersions)
	}

	// Dynamic cached-resource atoms should not leave revision tombstones behind.
	for parseIndex := range 1000 {
		parseReg.SetAtom(fmt.Sprintf("cached-resource-%d", parseIndex), parseIndex)
	}
	if len(parseReg.atomVersions) != 1 {
		parseT.Fatalf("untracked dynamic writes grew revision map: %d entries", len(parseReg.atomVersions))
	}
	parseWideDeps := []string{"wide-0", "wide-1", "wide-2", "wide-3", "wide-4"}
	for parseIndex, parseDep := range parseWideDeps {
		parseReg.InitAtom(parseDep, parseIndex)
	}
	if parseErr := parseReg.RegisterDerivedAtom("wide-sum", parseWideDeps, func() any {
		parseSum := 0
		for _, parseDep := range parseWideDeps {
			parseValue, _ := parseReg.GetAtom(parseDep)
			parseInt, _ := parseValue.(int)
			parseSum += parseInt
		}
		return parseSum
	}); parseErr != nil {
		parseT.Fatalf("register wide derived: %v", parseErr)
	}
	parseReg.setAtomAndNotify("wide-4", 10, func(*Fiber) {})
	if parseSum, _ := parseReg.GetAtom("wide-sum"); parseSum != 16 {
		parseT.Fatalf("wide dependency recompute got %v, want 16", parseSum)
	}
	parseReg.deleteAtom("wide-sum")
	if len(parseReg.atomVersions) != 1 {
		parseT.Fatalf("removing wide derivation retained revisions: %v", parseReg.atomVersions)
	}

	if parseErr := parseReg.RegisterDerivedAtom("quadrupled", []string{"doubled"}, func() any {
		parseValue, _ := parseReg.GetAtom("doubled")
		parseInt, _ := parseValue.(int)
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("register chained derived: %v", parseErr)
	}
	if len(parseReg.atomVersions) != 2 {
		parseT.Fatalf("expected src and doubled to be tracked, got %v", parseReg.atomVersions)
	}
	parseReg.deleteAtom("quadrupled")
	if _, parseTracked := parseReg.atomVersions["doubled"]; parseTracked || len(parseReg.atomVersions) != 1 {
		parseT.Fatalf("removed last dependent left a tracked derived key: %v", parseReg.atomVersions)
	}
	parseReg.deleteAtom("doubled")
	if len(parseReg.atomVersions) != 0 {
		parseT.Fatalf("removing final dependent left revision tombstones: %v", parseReg.atomVersions)
	}
	if parseErr := parseReg.RegisterDerivedAtom("doubled", []string{"src"}, func() any {
		parseValue, _ := parseReg.GetAtom("src")
		parseInt, _ := parseValue.(int)
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("re-register derived: %v", parseErr)
	}
	if *parseReg.atomVersions["src"] == parseInitialRevision {
		parseT.Fatalf("dependency revision token was reused after untracking: %d", parseInitialRevision)
	}
}

func TestAtomRegistryStaleDerivedRecomputeCannotOverwriteNewerSource(parseT *testing.T) {
	parseReg := NewAtomRegistry()
	parseReg.InitAtom("src", 0)
	parseBlockNext := atomic.Bool{}
	parseReadOld := make(chan struct{})
	parseReleaseOld := make(chan struct{})
	if parseErr := parseReg.RegisterDerivedAtom("doubled", []string{"src"}, func() any {
		parseValue, _ := parseReg.GetAtom("src")
		parseInt, _ := parseValue.(int)
		if parseBlockNext.Swap(false) {
			close(parseReadOld)
			<-parseReleaseOld
		}
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("register derived: %v", parseErr)
	}
	if parseErr := parseReg.RegisterDerivedAtom("quadrupled", []string{"doubled"}, func() any {
		parseValue, _ := parseReg.GetAtom("doubled")
		parseInt, _ := parseValue.(int)
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("register chained derived: %v", parseErr)
	}

	parseBlockNext.Store(true)
	parseOldWriteDone := make(chan struct{})
	go func() {
		parseReg.setAtomAndNotify("src", 1, func(*Fiber) {})
		close(parseOldWriteDone)
	}()
	<-parseReadOld // the old recomputation has captured src=1

	// This newer write recomputes and commits doubled=4 while the old compute is
	// paused. Releasing the old compute must not allow doubled=2 to overwrite it.
	parseReg.setAtomAndNotify("src", 2, func(*Fiber) {})
	close(parseReleaseOld)
	<-parseOldWriteDone

	parseFinal, _ := parseReg.GetAtom("src")
	parseDerived, _ := parseReg.GetAtom("doubled")
	parseChainedDerived, _ := parseReg.GetAtom("quadrupled")
	if parseFinal != 2 || parseDerived != 4 || parseChainedDerived != 8 {
		parseT.Fatalf("stale recompute overwrote newer result: src=%v doubled=%v quadrupled=%v, want 2, 4, and 8", parseFinal, parseDerived, parseChainedDerived)
	}
}

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
	if parseErr := parseReg.RegisterDerivedAtom("doubled", []string{"src"}, func() any {
		parseVal, _ := parseReg.GetAtom("src")
		parseInt, _ := parseVal.(int)
		return parseInt * 2
	}); parseErr != nil {
		parseT.Fatalf("register derived: %v", parseErr)
	}

	var parseWg sync.WaitGroup
	parseSeen := make([]map[int]bool, parseWriters)
	for parseW := range parseWriters {
		parseW2 := parseW
		parseSeen[parseW2] = map[int]bool{}
		parseWg.Go(func() {
			for parseI := range parseWritesPerWriter {
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
		})
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
}
