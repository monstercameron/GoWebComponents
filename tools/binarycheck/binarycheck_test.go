package binarycheck

import (
	"reflect"
	"testing"
)

// TestFindTracked verifies binary detection, case folding, and the allowlist.
func TestFindTracked(t *testing.T) {
	allowedBinaries["vendor/ok.wasm"] = "test fixture"
	defer delete(allowedBinaries, "vendor/ok.wasm")

	parseGot := FindTracked([]string{
		"main.go", "gwc.exe", "pkg/lib.SO", "a/b.test", "a/b.test.log",
		`web\app.wasm`, "vendor/ok.wasm", "x/y.dylib", "z.dll", "README",
	})
	parseWant := []string{"gwc.exe", "pkg/lib.SO", "a/b.test", "web/app.wasm", "x/y.dylib", "z.dll"}
	if !reflect.DeepEqual(parseGot, parseWant) {
		t.Fatalf("got %v want %v", parseGot, parseWant)
	}
	if len(FindTracked(nil)) != 0 {
		t.Fatal("expected no findings for empty input")
	}
}
