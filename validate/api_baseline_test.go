package validate_test

import (
	"embed"
	"os"
	"runtime"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/apidump"
)

//go:embed *.go
var parseAPISources embed.FS

//go:embed api_baseline.txt
var parseAPIGolden []byte

// TestPublicAPIBaseline pins this package's exported surface. A change to any exported
// type, function, method, const, or var fails this test; regenerate the golden with
// UPDATE_API_BASELINE=1 after an intentional, reviewed change.
func TestPublicAPIBaseline(t *testing.T) {
	if os.Getenv("UPDATE_API_BASELINE") == "1" {
		if runtime.GOOS == "js" {
			t.Fatal("regenerate API baselines with the native Go test target, then rebuild the Wasm test")
		}
		apidump.Check(t, ".", "api_baseline.txt")
		return
	}
	parseEntries, parseErr := parseAPISources.ReadDir(".")
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	parseSources := map[string][]byte{}
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() {
			continue
		}
		parseData, parseReadErr := parseAPISources.ReadFile(parseEntry.Name())
		if parseReadErr != nil {
			t.Fatal(parseReadErr)
		}
		parseSources[parseEntry.Name()] = parseData
	}
	apidump.CheckSources(t, parseSources, parseAPIGolden)
}
