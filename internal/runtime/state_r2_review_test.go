package runtime

import "testing"

// TestRegisterDerivedAtomRejectedCycleKeepsExistingRegistration pins that a
// re-registration refused for a dependency cycle leaves the previous
// registration fully wired, including its dependents links.
func TestRegisterDerivedAtomRejectedCycleKeepsExistingRegistration(parseT *testing.T) {
	parseAr := NewAtomRegistry()
	parseAr.InitAtom("a", 1)
	parseRead := func(parseId string) int {
		parseValue, _ := parseAr.GetAtom(parseId)
		parseInt, _ := parseValue.(int)
		return parseInt
	}
	if parseErr := parseAr.RegisterDerivedAtom("d", []string{"a"}, func() any { return parseRead("a") * 10 }); parseErr != nil {
		parseT.Fatalf("register d: %v", parseErr)
	}
	if parseErr := parseAr.RegisterDerivedAtom("e", []string{"d"}, func() any { return parseRead("d") + 1 }); parseErr != nil {
		parseT.Fatalf("register e: %v", parseErr)
	}
	// d now depends on e, which depends on d: refused.
	if parseErr := parseAr.RegisterDerivedAtom("d", []string{"e"}, func() any { return parseRead("e") }); parseErr == nil {
		parseT.Fatal("expected a cycle error")
	}

	parseAr.setAtomAndNotify("a", 2, func(*Fiber) {})
	if parseGot := parseRead("d"); parseGot != 20 {
		parseT.Fatalf("refused re-registration unlinked the old derived atom: d=%d, want 20", parseGot)
	}
	if parseGot := parseRead("e"); parseGot != 21 {
		parseT.Fatalf("downstream derived atom went stale: e=%d, want 21", parseGot)
	}
}
