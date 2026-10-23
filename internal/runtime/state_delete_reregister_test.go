package runtime

import (
	"sync/atomic"
	"testing"
)

func TestDeleteAndReregisterDerivedKeepsDownstreamChain(parseT *testing.T) {
	parseReg := NewAtomRegistry()
	parseReg.InitAtom("src", 1)
	parseReadInt := func(parseID string) int {
		parseValue, parseOK := parseReg.GetAtom(parseID)
		if !parseOK {
			parseT.Fatalf("expected atom %q to exist", parseID)
		}
		return parseValue.(int)
	}
	if parseErr := parseReg.RegisterDerivedAtom("d", []string{"src"}, func() any {
		return parseReadInt("src") * 2
	}); parseErr != nil {
		parseT.Fatalf("register d: %v", parseErr)
	}
	if parseErr := parseReg.RegisterDerivedAtom("e", []string{"d"}, func() any {
		return parseReadInt("d") + 100
	}); parseErr != nil {
		parseT.Fatalf("register e: %v", parseErr)
	}
	if parseGot := parseReadInt("e"); parseGot != 102 {
		parseT.Fatalf("initial e = %d, want 102", parseGot)
	}

	parseReg.deleteAtom("d")
	if parseDependents := parseReg.listDependents("d"); len(parseDependents) != 1 || parseDependents[0] != "e" {
		parseT.Fatalf("deleting d lost its active downstream edge: dependents[d] = %v, want [e]", parseDependents)
	}
	if _, parseTracked := parseReg.atomVersions["d"]; !parseTracked {
		parseT.Fatal("deleting d dropped the revision still needed by its active dependent e")
	}

	// While d is absent, changing src has no effect on the deleted registration.
	parseReg.SetAtom("src", 3)
	if parseErr := parseReg.RegisterDerivedAtom("d", []string{"src"}, func() any {
		return parseReadInt("src") * 2
	}); parseErr != nil {
		parseT.Fatalf("re-register d: %v", parseErr)
	}
	if parseGot := parseReadInt("e"); parseGot != 106 {
		parseT.Fatalf("e after d re-registration = %d, want 106", parseGot)
	}

	parseReg.setAtomAndNotify("src", 4, func(*Fiber) {})
	if parseGot := parseReadInt("e"); parseGot != 108 {
		parseT.Fatalf("e after subsequent source update = %d, want 108", parseGot)
	}

	// A downstream edge left alive across deletion must still participate in
	// cycle detection when d is registered again.
	parseReg.deleteAtom("d")
	if parseErr := parseReg.RegisterDerivedAtom("d", []string{"e"}, func() any {
		return parseReadInt("e")
	}); parseErr == nil {
		parseT.Fatal("expected re-registering d through e to be rejected as a cycle")
	}
	if parseErr := parseReg.RegisterDerivedAtom("d", []string{"src"}, func() any {
		return parseReadInt("src") * 2
	}); parseErr != nil {
		parseT.Fatalf("register d after rejected cycle: %v", parseErr)
	}
	if parseGot := parseReadInt("e"); parseGot != 108 {
		parseT.Fatalf("e after cycle rejection and valid re-registration = %d, want 108", parseGot)
	}

	// Removing the last downstream registration releases the retained edge and
	// all revision entries; repeated lifecycles must not accumulate bookkeeping.
	parseReg.deleteAtom("e")
	parseReg.deleteAtom("d")
	if len(parseReg.dependents) != 0 {
		parseT.Fatalf("delete lifecycle left dependent edges: %v", parseReg.dependents)
	}
	if len(parseReg.atomVersions) != 0 {
		parseT.Fatalf("delete lifecycle left revision entries: %v", parseReg.atomVersions)
	}
	for parseIndex := range 64 {
		parseID := "dynamic-derived"
		if parseErr := parseReg.RegisterDerivedAtom(parseID, []string{"src"}, func() any {
			return parseReadInt("src")
		}); parseErr != nil {
			parseT.Fatalf("lifecycle %d register: %v", parseIndex, parseErr)
		}
		parseReg.deleteAtom(parseID)
		if len(parseReg.dependents) != 0 || len(parseReg.atomVersions) != 0 {
			parseT.Fatalf("lifecycle %d retained bookkeeping: dependents=%v revisions=%v", parseIndex, parseReg.dependents, parseReg.atomVersions)
		}
	}
}

func TestDeleteAtomInvalidatesInflightDownstreamRecompute(parseT *testing.T) {
	parseReg := NewAtomRegistry()
	parseReg.InitAtom("src", 1)
	parseReadInt := func(parseID string) int {
		parseValue, parseOK := parseReg.GetAtom(parseID)
		if !parseOK {
			parseT.Fatalf("expected atom %q to exist", parseID)
		}
		return parseValue.(int)
	}
	if parseErr := parseReg.RegisterDerivedAtom("d", []string{"src"}, func() any {
		return parseReadInt("src") * 2
	}); parseErr != nil {
		parseT.Fatalf("register d: %v", parseErr)
	}
	var parseBlockNext atomic.Bool
	parseReadOld := make(chan struct{})
	parseReleaseOld := make(chan struct{})
	parseLookupFailure := make(chan string, 1)
	parseBlockIfRequested := func() {
		if parseBlockNext.Swap(false) {
			close(parseReadOld)
			<-parseReleaseOld
		}
	}
	parseReportLookupFailure := func(parseMessage string) {
		select {
		case parseLookupFailure <- parseMessage:
		default:
		}
	}
	if parseErr := parseReg.RegisterDerivedAtom("e", []string{"d"}, func() any {
		parseValue, parseOK := parseReg.GetAtom("d")
		if !parseOK {
			parseReportLookupFailure(`atom "d" is missing`)
			parseBlockIfRequested()
			return -1
		}
		parseD, parseIsInt := parseValue.(int)
		if !parseIsInt {
			parseReportLookupFailure(`atom "d" is not an int`)
			parseBlockIfRequested()
			return -1
		}
		parseBlockIfRequested()
		return parseD + 100
	}); parseErr != nil {
		parseT.Fatalf("register e: %v", parseErr)
	}

	// Give the stale result a distinguishable value to overwrite if deletion
	// fails to invalidate the in-flight computation.
	parseReg.SetAtom("e", 150)
	parseBlockNext.Store(true)
	parseRecomputeErr := make(chan error, 1)
	go func() {
		_, parseErr := parseReg.recomputeDerived("e", nil)
		parseRecomputeErr <- parseErr
	}()
	<-parseReadOld // e captured d=2 and is paused before attempting its commit
	parseReg.deleteAtom("d")
	close(parseReleaseOld)
	if parseErr := <-parseRecomputeErr; parseErr != nil {
		parseT.Fatalf("in-flight recompute returned error: %v", parseErr)
	}
	select {
	case parseFailure := <-parseLookupFailure:
		parseT.Fatalf("in-flight recompute lookup failed: %s", parseFailure)
	default:
	}
	if parseGot, _ := parseReg.GetAtom("e"); parseGot != 150 {
		parseT.Fatalf("stale in-flight recompute overwrote e after d deletion: got %#v, want 150", parseGot)
	}

	// Re-register d from a newer source value. The preserved edge must cause e
	// to recompute again after the stale in-flight result has been discarded.
	parseReg.SetAtom("src", 5)
	if parseErr := parseReg.RegisterDerivedAtom("d", []string{"src"}, func() any {
		return parseReadInt("src") * 2
	}); parseErr != nil {
		parseT.Fatalf("re-register d: %v", parseErr)
	}
	if parseGot := parseReadInt("e"); parseGot != 110 {
		parseT.Fatalf("e after d re-registration = %d, want 110", parseGot)
	}
}
