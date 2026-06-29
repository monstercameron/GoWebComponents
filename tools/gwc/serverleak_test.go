package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServerLeakFlagsClientOnlyImports proves the analyzer flags a server-only import in a
// browser-only (js && wasm) file, but NOT in a server-tagged file or a shared no-constraint
// file (no false positives).
func TestServerLeakFlagsClientOnlyImports(parseT *testing.T) {
	parseDir := parseT.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(parseDir, name), []byte(content), 0644); err != nil {
			parseT.Fatalf("write %s: %v", name, err)
		}
	}
	// Browser-only file importing os/exec -> LEAK.
	write("client.go", "//go:build js && wasm\n\npackage x\n\nimport _ \"os/exec\"\n")
	// Server-tagged file importing os/exec -> OK.
	write("server.go", "//go:build !js || !wasm\n\npackage x\n\nimport _ \"os/exec\"\n")
	// Shared (no constraint) file importing os/exec -> OK (not flagged; could be server).
	write("shared.go", "package x\n\nimport _ \"os/exec\"\n")
	// Browser-only file importing a fine package -> OK.
	write("client_ok.go", "//go:build js && wasm\n\npackage x\n\nimport _ \"strings\"\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	if len(parseDiags) != 1 {
		parseT.Fatalf("expected exactly one leak (client.go), got %d: %#v", len(parseDiags), parseDiags)
	}
	if !strings.Contains(parseDiags[0].File, "client.go") || !strings.Contains(parseDiags[0].Message, "os/exec") {
		parseT.Fatalf("leak should name client.go + os/exec, got %#v", parseDiags[0])
	}
}

// TestServerLeakFlagsThirdPartyServerSDKs proves the analyzer no longer relies on a tiny stdlib
// deny-list: a browser file importing gorm/grpc/an AWS SDK is caught by the server-package
// prefix classifier (the exact gap the audit named).
func TestServerLeakFlagsThirdPartyServerSDKs(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "client.go",
		"//go:build js && wasm\n\npackage x\n\nimport (\n\t_ \"gorm.io/gorm\"\n\t_ \"google.golang.org/grpc\"\n\t_ \"github.com/aws/aws-sdk-go/aws\"\n)\n")
	// A path that merely shares a prefix segment must NOT be flagged.
	mustWrite(parseT, parseDir, "client_ok.go",
		"//go:build js && wasm\n\npackage x\n\nimport _ \"github.com/awesome/widget\"\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	parseLeaked := map[string]bool{}
	for _, parseDiag := range parseDiags {
		parseLeaked[parseDiag.Attributes["leaked"]] = true
	}
	for _, parseWant := range []string{"gorm.io/gorm", "google.golang.org/grpc", "github.com/aws/aws-sdk-go/aws"} {
		if !parseLeaked[parseWant] {
			parseT.Fatalf("expected %s flagged, got leaks %v", parseWant, parseLeaked)
		}
	}
	if parseLeaked["github.com/awesome/widget"] {
		parseT.Fatal("github.com/awesome/widget shares no real prefix and must not be flagged")
	}
}

// TestServerLeakConfigExtendsPrefixes proves the curated third-party list is not a ceiling: a
// project-supplied gwc-serverleak.json prefix flags an otherwise-unknown server library imported
// into the browser build (closing the FB1 "fragile fixed list" gap).
func TestServerLeakConfigExtendsPrefixes(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	mustWrite(parseT, parseDir, "gwc-serverleak.json",
		`{"serverOnlyPrefixes":[{"prefix":"github.com/acme/internal-db","reason":"internal DB client"}]}`)
	mustWrite(parseT, parseDir, "client.go",
		"//go:build js && wasm\n\npackage x\n\nimport _ \"github.com/acme/internal-db/pg\"\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	parseFound := false
	for _, parseDiag := range parseDiags {
		if parseDiag.Attributes["leaked"] == "github.com/acme/internal-db/pg" {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatalf("expected the configured prefix to flag github.com/acme/internal-db/pg, got %d diags", len(parseDiags))
	}
}

// TestServerLeakFollowsTransitiveLocalImport proves the analyzer walks the import graph: a
// browser file that imports a LOCAL helper package which itself imports a database driver is
// flagged, even though the client file's own imports look clean. This is the core upgrade over
// the old direct-imports-only scan.
func TestServerLeakFollowsTransitiveLocalImport(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	// Client entry imports a local helper — its own imports are clean.
	mustWrite(parseT, parseDir, "ui/client.go",
		"//go:build js && wasm\n\npackage ui\n\nimport _ \"example.com/app/store\"\n")
	// The helper (shared, builds under wasm) drags in a Postgres driver.
	mustWrite(parseT, parseDir, "store/store.go",
		"package store\n\nimport _ \"github.com/jackc/pgx/v5\"\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	parseFound := false
	for _, parseDiag := range parseDiags {
		if parseDiag.Attributes["leaked"] == "github.com/jackc/pgx/v5" &&
			strings.Contains(parseDiag.File, "client.go") &&
			strings.Contains(parseDiag.Message, "via") {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatalf("expected a transitive pgx leak through example.com/app/store, got %#v", parseDiags)
	}
}

// TestServerLeakFlagsLocalPackageConstrainedOffWasm proves importing a local package that has no
// wasm-buildable files (entirely server-constrained) is itself a leak — the client build cannot
// compile it.
func TestServerLeakFlagsLocalPackageConstrainedOffWasm(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	mustWrite(parseT, parseDir, "ui/client.go",
		"//go:build js && wasm\n\npackage ui\n\nimport _ \"example.com/app/serverdb\"\n")
	mustWrite(parseT, parseDir, "serverdb/db.go",
		"//go:build !js || !wasm\n\npackage serverdb\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	parseFound := false
	for _, parseDiag := range parseDiags {
		if parseDiag.Attributes["leaked"] == "example.com/app/serverdb" && strings.Contains(parseDiag.Message, "constrained off js/wasm") {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatalf("expected serverdb (constrained off js/wasm) flagged, got %#v", parseDiags)
	}
}

func mustWrite(parseT *testing.T, parseDir string, parseName string, parseContent string) {
	parseT.Helper()
	parseFull := filepath.Join(parseDir, parseName)
	if parseErr := os.MkdirAll(filepath.Dir(parseFull), 0755); parseErr != nil {
		parseT.Fatalf("mkdir for %s: %v", parseName, parseErr)
	}
	if parseErr := os.WriteFile(parseFull, []byte(parseContent), 0644); parseErr != nil {
		parseT.Fatalf("write %s: %v", parseName, parseErr)
	}
}
