package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// atlasWASMAssetPath is the StaticDir-relative (and therefore /assets-relative)
// location of the compiled client bundle. The document's hydration snippet
// fetches "/assets/" + atlasWASMAssetPath, and /assets/ is served from
// StaticDir, so deriving config.AtlasWASM from this same constant keeps the
// presence check and the served URL pointing at one file. Changing either one
// without the other is what previously let /healthz report wasmPresent:true
// while the browser fetch 404'd.
const atlasWASMAssetPath = "bin/atlas-commerce-os.wasm"

// atlasWASMBuildCommand is the command that produces atlasWASMAssetPath. It is
// surfaced in the startup warning so a missing bundle names its own fix.
const atlasWASMBuildCommand = "GOOS=js GOARCH=wasm go build -o examples/static/" + atlasWASMAssetPath + " ./examples/server/atlas-commerce-os/client"

// config is the set of resolved paths and flags the server needs.
//
// TWO FIELDS CAME OUT OF HERE, and the reason generalizes: a config field that is
// computed and asserted but never READ is worse than no field, because a test
// pinning its value looks like coverage of behaviour that does not exist.
//
//   - TailwindCSS pointed at examples/static/css/tailwind.css, which Atlas no longer
//     links (see atlasDesignStyleBlock in server.go). Nothing resolved it.
//   - ExampleLoggerJS pointed at examples/static/script/example-logger.js, which
//     Atlas stopped linking earlier. Nothing resolved it either.
//
// Neither was ever used to SERVE anything: /assets/ is one http.FileServer over
// StaticDir, so every file under examples/static is reachable without being named
// here. WASMExecJS is in the same shape and is kept on purpose — the document does
// link wasm_exec.js, so the field documents a real dependency and is the natural
// home for a presence check if one is ever added (AtlasWASM already has one).
type config struct {
	Addr              string
	LogsEnabled       bool
	RepoRoot          string
	ExampleRoot       string
	FallbackSchema    string
	MigrationsDir     string
	SQLitePath        string
	StaticDir         string
	WASMExecJS        string
	AtlasWASM         string
	DefaultPublicHost string
}

func loadConfig() (config, error) {
	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		return config{}, fmt.Errorf("get working directory: %w", parseErr)
	}
	parseRepoRoot, parseErr := findRepoRoot(parseWorkingDir)
	if parseErr != nil {
		return config{}, parseErr
	}
	parseExampleRoot := filepath.Join(parseRepoRoot, "examples", "server", "atlas-commerce-os")
	parseStaticDir := filepath.Join(parseRepoRoot, "examples", "static")
	parseAddr := strings.TrimSpace(os.Getenv("ATLAS_ADDR"))
	if parseAddr == "" {
		parseAddr = "127.0.0.1:8096"
	}
	parseLogsRaw := strings.TrimSpace(strings.ToLower(os.Getenv("ATLAS_DEBUG_LOGS")))
	parseLogsEnabled := parseLogsRaw == "1" || parseLogsRaw == "true" || parseLogsRaw == "yes" || parseLogsRaw == "on"
	parseDatabasePath := strings.TrimSpace(os.Getenv("ATLAS_DB_PATH"))
	if parseDatabasePath == "" {
		parseDatabasePath = filepath.Join(parseExampleRoot, "server", "data", "atlas-commerce-os.db")
	}
	parseDatabasePath, parseErr = filepath.Abs(parseDatabasePath)
	if parseErr != nil {
		return config{}, fmt.Errorf("resolve ATLAS_DB_PATH: %w", parseErr)
	}
	return config{
		Addr:              parseAddr,
		LogsEnabled:       parseLogsEnabled,
		RepoRoot:          parseRepoRoot,
		ExampleRoot:       parseExampleRoot,
		FallbackSchema:    filepath.Join(parseExampleRoot, "server", "data", "schema.sql"),
		MigrationsDir:     filepath.Join(parseExampleRoot, "server", "data", "migrations"),
		SQLitePath:        parseDatabasePath,
		StaticDir:         parseStaticDir,
		WASMExecJS:        filepath.Join(parseStaticDir, "script", "wasm_exec.js"),
		AtlasWASM:         filepath.Join(parseStaticDir, filepath.FromSlash(atlasWASMAssetPath)),
		DefaultPublicHost: "http://127.0.0.1:8096",
	}, nil
}

func findRepoRoot(parseStart string) (string, error) {
	parseCurrent := parseStart
	for {
		if _, parseErr := os.Stat(filepath.Join(parseCurrent, "go.mod")); parseErr == nil {
			return parseCurrent, nil
		}
		parseParent := filepath.Dir(parseCurrent)
		if parseParent == parseCurrent {
			return "", fmt.Errorf("could not locate repo root from %s", parseStart)
		}
		parseCurrent = parseParent
	}
}
