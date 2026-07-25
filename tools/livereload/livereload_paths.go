package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/tools/runnerconfig"
)

func resolveBuildDir(parseEntryPath string) (string, error) {
	parseAbsPath, parseErr := filepath.Abs(strings.TrimSpace(parseEntryPath))
	if parseErr != nil {
		return "", fmt.Errorf("failed to resolve entry path: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsPath)
	if parseErr != nil {
		return "", fmt.Errorf("failed to inspect entry path %s: %w", parseAbsPath, parseErr)
	}
	if parseInfo.IsDir() {
		return parseAbsPath, nil
	}
	return filepath.Dir(parseAbsPath), nil
}

func resolveModuleRoot(parseStartPath string) string {
	parseCurrent := strings.TrimSpace(parseStartPath)
	if parseCurrent == "" {
		return ""
	}

	parseAbsPath, parseErr := filepath.Abs(parseCurrent)
	if parseErr != nil {
		return ""
	}

	parseInfo, parseErr := os.Stat(parseAbsPath)
	if parseErr != nil {
		return ""
	}
	if !parseInfo.IsDir() {
		parseAbsPath = filepath.Dir(parseAbsPath)
	}

	for {
		if _, parseErr2 := os.Stat(filepath.Join(parseAbsPath, "go.mod")); parseErr2 == nil {
			return parseAbsPath
		}
		parseParent := filepath.Dir(parseAbsPath)
		if parseParent == parseAbsPath {
			return ""
		}
		parseAbsPath = parseParent
	}
}

func resolveStaticDir(parseProjectRoot string) string {
	parseCandidates := []string{
		filepath.Join(parseProjectRoot, "static"),
		filepath.Join(filepath.Dir(parseProjectRoot), "static"),
	}

	for _, parseCandidate := range parseCandidates {
		if parseInfo, parseErr := os.Stat(parseCandidate); parseErr == nil && parseInfo.IsDir() {
			return parseCandidate
		}
	}

	return ""
}

func resolveClientScriptPath() string {
	return resolveConfiguredClientScriptPath()
}

func firstExistingPath(parseCandidates ...string) string {
	for _, parseCandidate := range parseCandidates {
		if strings.TrimSpace(parseCandidate) == "" {
			continue
		}
		parseCleanCandidate := filepath.Clean(parseCandidate)
		if _, parseErr := os.Stat(parseCleanCandidate); parseErr == nil {
			return parseCleanCandidate
		}
	}
	return ""
}

func resolveConfiguredClientScriptPath() string {
	parseCwd, parseErr := livereloadConfigGetwd()
	if parseErr != nil {
		parseCwd = ""
	}
	parseResolved, parseOk, parseErr := runnerconfig.ResolveConfiguredPath(parseCwd, func(parsePaths runnerconfig.Paths) string {
		return parsePaths.LivereloadClientScript
	}, "livereloadClientScript", livereloadRunnerConfigFS())
	if parseErr != nil || !parseOk {
		return ""
	}
	if _, parseErr2 := os.Stat(parseResolved); parseErr2 == nil {
		return parseResolved
	}
	return ""
}

func resolveLivereloadRunnerConfigPath() string {
	parseCwd, parseErr := livereloadConfigGetwd()
	if parseErr != nil {
		parseCwd = ""
	}
	parseConfigPath, parseErr := runnerconfig.ResolveConfigPath(parseCwd, livereloadRunnerConfigFS())
	if parseErr != nil {
		return ""
	}
	return parseConfigPath
}

func netAddr(parseHost, parsePort string) string {
	if strings.TrimSpace(parseHost) == "" {
		parseHost = defaultHost
	}
	if strings.TrimSpace(parsePort) == "" {
		parsePort = defaultPort
	}
	return net.JoinHostPort(parseHost, parsePort)
}

func main() {
	parseAppPath := flag.String("app", "", "Path to the app main.go file or the app directory")
	parseMainPath := flag.String("main", "", "Legacy alias for -app")
	parseRootPath := flag.String("root", "", "Project root to watch and serve")
	parseHtmlPath := flag.String("html", "", "HTML file to serve, relative to the project root")
	parseIndexPath := flag.String("index", "", "Legacy alias for -html")
	parseWasmPath := flag.String("wasm", "", "WASM output path, relative to the build directory")
	parseOutputPath := flag.String("output", "", "Legacy alias for -wasm")
	parseHost := flag.String("host", defaultHost, "Host to bind")
	parsePort := flag.String("port", defaultPort, "Port to bind")
	parseHot := flag.Bool("hot", true, "Always use hot reload on successful rebuilds")
	parseClientScriptPath := flag.String("client-script", "", "Optional override path to a custom livereload client script")
	parseAllowAnyOrigin := flag.Bool("allow-any-origin", false, "Disable WebSocket Origin validation (tunnel/LAN dev only; logs a warning)")
	flag.Parse()

	parseSelectedAppPath := strings.TrimSpace(*parseAppPath)
	if parseSelectedAppPath == "" {
		parseSelectedAppPath = strings.TrimSpace(*parseMainPath)
	}
	parseSelectedHTMLPath := strings.TrimSpace(*parseHtmlPath)
	if parseSelectedHTMLPath == "" {
		parseSelectedHTMLPath = strings.TrimSpace(*parseIndexPath)
	}
	parseSelectedWASMPath := strings.TrimSpace(*parseWasmPath)
	if parseSelectedWASMPath == "" {
		parseSelectedWASMPath = strings.TrimSpace(*parseOutputPath)
	}

	parseServer, parseErr := NewLiveReloadServerWithOptions(LiveReloadOptions{
		MainPath:         parseSelectedAppPath,
		ProjectRoot:      *parseRootPath,
		IndexPath:        parseSelectedHTMLPath,
		OutputPath:       parseSelectedWASMPath,
		Host:             *parseHost,
		Port:             *parsePort,
		AlwaysHotReload:  *parseHot,
		ClientScriptPath: *parseClientScriptPath,
		AllowAnyOrigin:   *parseAllowAnyOrigin,
	})
	if parseErr != nil {
		fatalLivereloadStartup("main.NewLiveReloadServerWithOptions", strings.TrimSpace(*parseRootPath), parseErr, "Inspect the selected app, project root, HTML path, and client-script arguments before starting livereload again.")
	}

	defer parseServer.cleanup()

	parseErr = parseServer.Start()
	if parseErr != nil {
		fatalLivereloadStartup("main.Start", netAddr(*parseHost, *parsePort), parseErr, "Inspect watcher initialization and HTTP startup errors before restarting the livereload tool.")
	}
}
