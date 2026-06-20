//go:build !(js && wasm)

package ui

import "testing"

func TestNativeFileHelpersAreSafeNoOps(t *testing.T) {
	defer func() {
		if parseR := recover(); parseR != nil {
			t.Fatalf("native file helpers panicked: %v", parseR)
		}
	}()
	Download([]byte("data"), "f.txt", "text/plain")
	PickFile("image/*", func(PickedFile) { t.Fatal("native PickFile should not call onPick") })
	PickFile(".csv", nil) // nil onPick guarded
}

func TestPickFileSeamSwappable(t *testing.T) {
	parsePrev := pickFileFn
	defer func() { pickFileFn = parsePrev }()
	var parseGotAccept string
	pickFileFn = func(parseAccept string, parseOnPick func(PickedFile)) {
		parseGotAccept = parseAccept
		parseOnPick(PickedFile{Name: "x.csv", Data: []byte("a,b")})
	}
	var parseGot PickedFile
	PickFile(".csv", func(parseFile PickedFile) { parseGot = parseFile })
	if parseGotAccept != ".csv" || parseGot.Name != "x.csv" || string(parseGot.Data) != "a,b" {
		t.Fatalf("seam not wired: accept=%q got=%+v", parseGotAccept, parseGot)
	}
}
