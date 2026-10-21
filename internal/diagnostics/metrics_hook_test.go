package diagnostics

import "testing"

// TestNotifyPanicCallsObserver proves the observer receives the code and that
// removing it silences further notifications.
func TestNotifyPanicCallsObserver(parseT *testing.T) {
	var parseGot []string
	SetPanicObserver(func(parseCode string) { parseGot = append(parseGot, parseCode) })
	defer SetPanicObserver(nil)
	NotifyPanic("GWC-1")
	NotifyPanic("GWC-2")
	if len(parseGot) != 2 || parseGot[0] != "GWC-1" || parseGot[1] != "GWC-2" {
		parseT.Fatalf("observer calls = %v", parseGot)
	}
	SetPanicObserver(nil)
	NotifyPanic("GWC-3")
	if len(parseGot) != 2 {
		parseT.Fatalf("observer called after removal: %v", parseGot)
	}
}

// TestNotifyPanicSurvivesObserverPanic proves a failing observer cannot escape.
func TestNotifyPanicSurvivesObserverPanic(parseT *testing.T) {
	SetPanicObserver(func(string) { panic("boom") })
	defer SetPanicObserver(nil)
	NotifyPanic("GWC-1")
}
