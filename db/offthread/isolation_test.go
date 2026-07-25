package offthread_test

import (
	"os/exec"
	"strings"
	"testing"
)

// v5 P3.5 criterion (b) — an app using the off-thread model has no wazero symbol
// in app.wasm.
//
// This is the criterion the whole package split exists to satisfy, so it is
// checked mechanically rather than by inspection. The dependency GRAPH is the
// right thing to assert: a symbol grep on a built binary depends on how the
// linker felt about dead code that day, while an import edge is exact and fails
// the moment someone adds a convenience import.
//
// The check runs for js/wasm specifically. On native, db/sqlite uses
// modernc.org/sqlite instead of ncruces+wazero, so a native-only check would
// pass while the browser binary carried the engine — the exact false negative
// worth avoiding.

// enginePackages are the import paths that mean "the SQLite engine is linked".
var enginePackages = []string{
	"github.com/tetratelabs/wazero",
	"github.com/ncruces/go-sqlite3",
	"modernc.org/sqlite",
	"github.com/monstercameron/GoWebComponents/v5/db/sqlite",
}

// listDeps returns the transitive dependency list of a package for one platform.
func listDeps(parseT *testing.T, parseGOOS string, parseGOARCH string, parsePackage string) []string {
	parseT.Helper()

	parseCommand := exec.Command("go", "list", "-deps", parsePackage)
	parseCommand.Env = append(parseCommand.Environ(), "GOOS="+parseGOOS, "GOARCH="+parseGOARCH)
	parseOutput, parseErr := parseCommand.Output()
	if parseErr != nil {
		parseT.Skipf("go list unavailable for %s/%s: %v", parseGOOS, parseGOARCH, parseErr)
	}
	return strings.Split(strings.TrimSpace(string(parseOutput)), "\n")
}

// TestOffThreadClientDoesNotLinkTheEngine is P3.5 criterion (b).
func TestOffThreadClientDoesNotLinkTheEngine(parseT *testing.T) {
	parseDeps := listDeps(parseT, "js", "wasm", "github.com/monstercameron/GoWebComponents/v5/db/offthread")

	for _, parseDep := range parseDeps {
		for _, parseEnginePackage := range enginePackages {
			if parseDep == parseEnginePackage || strings.HasPrefix(parseDep, parseEnginePackage+"/") {
				parseT.Errorf("db/offthread depends on %q — an app using only the off-thread model would link the SQLite engine into app.wasm", parseDep)
			}
		}
	}
}

// TestOffThreadServerDoesLinkTheEngine is the other half of the same claim.
//
// Without it, criterion (b) could be satisfied by a client that simply does not
// work: this asserts the engine lives SOMEWHERE, and that the somewhere is the
// package meant for domain.wasm.
func TestOffThreadServerDoesLinkTheEngine(parseT *testing.T) {
	parseDeps := listDeps(parseT, "js", "wasm", "github.com/monstercameron/GoWebComponents/v5/db/offthread/server")

	hasEngine := false
	for _, parseDep := range parseDeps {
		if strings.HasPrefix(parseDep, "github.com/ncruces/go-sqlite3") || strings.HasPrefix(parseDep, "github.com/tetratelabs/wazero") {
			hasEngine = true
			break
		}
	}
	if !hasEngine {
		parseT.Error("db/offthread/server does not link a SQLite engine — the split has separated the client from nothing")
	}
}

// TestConventionalSQLiteDoesNotPullTheOffThreadClient is P3.5 criterion (a) —
// an app not using the off-thread model runs byte-identically to v4.
//
// The criterion cannot be proven by comparing binaries in a unit test, but the
// only way P3.5 could break it is by making db/sqlite depend on the new code, so
// that is what this asserts. The dependency direction must stay one-way:
// server → sqlite, never sqlite → offthread.
func TestConventionalSQLiteDoesNotPullTheOffThreadClient(parseT *testing.T) {
	for _, parseDep := range listDeps(parseT, "js", "wasm", "github.com/monstercameron/GoWebComponents/v5/db/sqlite") {
		if strings.HasPrefix(parseDep, "github.com/monstercameron/GoWebComponents/v5/db/offthread") {
			parseT.Errorf("db/sqlite depends on %q — every existing app's binary would change", parseDep)
		}
	}
}

// TestOffThreadClientHasNoDatabaseSQLDependency guards the design decision that
// the client returns materialized rows rather than *sql.Rows.
//
// database/sql is not the engine and would not fail criterion (b) on size, but
// depending on it would mean the client exposes a cursor type whose contract it
// cannot honor across a thread boundary — one round trip per row. Keeping the
// dependency out keeps that mistake unavailable.
func TestOffThreadClientHasNoDatabaseSQLDependency(parseT *testing.T) {
	for _, parseDep := range listDeps(parseT, "js", "wasm", "github.com/monstercameron/GoWebComponents/v5/db/offthread") {
		if parseDep == "database/sql" {
			parseT.Error("db/offthread depends on database/sql — results must be materialized, not cursor-shaped")
		}
	}
}
