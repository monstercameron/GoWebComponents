// Package wholestack composes a GoWebComponents app into a single http.Handler — the
// one-binary, whole-stack, edge-portable deployment (FC5). One `go build` produces an
// executable that serves the embedded wasm bundle (index.html, the .wasm, wasm_exec.js) AND
// the app's //gwc:server functions, with SPA fallback so client-routed paths resolve to the
// shell. No Node, no separate static host, no reverse proxy — the binary is the app, and it
// runs anywhere Go's net/http runs, including edge runtimes.
package wholestack

import (
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/monstercameron/GoWebComponents/serverfn"
)

// ServerFnPrefix re-exports the path server functions are mounted under.
const ServerFnPrefix = serverfn.RoutePrefix

// Options configures the whole-stack handler.
type Options struct {
	// Assets is the filesystem of built client assets (commonly an embed.FS sub-tree). It
	// must contain IndexFile at its root.
	Assets fs.FS
	// RegisterServerFns wires the app's //gwc:server functions onto the mux (typically the
	// generated RegisterServerFunctions). May be nil for a static-only app.
	RegisterServerFns func(*http.ServeMux)
	// IndexFile is the SPA shell served at "/" and as the fallback for unknown paths.
	// Defaults to "index.html".
	IndexFile string
	// DisableSPAFallback turns off serving the index shell for unknown, non-asset GETs.
	// The default (false) keeps SPA fallback ON so client-side routes deep-link correctly.
	DisableSPAFallback bool
}

// Handler builds the single http.Handler that is the whole app: server functions under
// serverfn.RoutePrefix, static assets everywhere else, and (unless disabled) an SPA
// fallback to the index shell for client-routed paths.
func Handler(parseOptions Options) http.Handler {
	parseIndex := parseOptions.IndexFile
	if parseIndex == "" {
		parseIndex = "index.html"
	}
	parseSPA := !parseOptions.DisableSPAFallback

	parseMux := http.NewServeMux()
	if parseOptions.RegisterServerFns != nil {
		parseOptions.RegisterServerFns(parseMux)
	}
	parseFileServer := http.FileServer(http.FS(parseOptions.Assets))
	parseMux.Handle("/", staticOrIndex(parseOptions.Assets, parseFileServer, parseIndex, parseSPA))
	return parseMux
}

// ListenAndServe builds the whole-stack handler and serves it on addr — the one-line
// production entry point for a single-binary app.
func ListenAndServe(parseAddr string, parseOptions Options) error {
	return http.ListenAndServe(parseAddr, Handler(parseOptions))
}

// staticOrIndex serves a static asset when it exists, otherwise (for a GET under SPA mode)
// the index shell, otherwise 404.
func staticOrIndex(parseAssets fs.FS, parseFileServer http.Handler, parseIndex string, parseSPA bool) http.HandlerFunc {
	return func(parseW http.ResponseWriter, parseR *http.Request) {
		parsePath := strings.TrimPrefix(parseR.URL.Path, "/")
		if parsePath == "" {
			serveFile(parseW, parseR, parseAssets, parseIndex)
			return
		}
		if assetExists(parseAssets, parsePath) {
			parseFileServer.ServeHTTP(parseW, parseR)
			return
		}
		if parseSPA && parseR.Method == http.MethodGet {
			serveFile(parseW, parseR, parseAssets, parseIndex)
			return
		}
		http.NotFound(parseW, parseR)
	}
}

// assetExists reports whether name resolves to a regular file in the asset FS.
func assetExists(parseAssets fs.FS, parseName string) bool {
	if parseAssets == nil {
		return false
	}
	parseFile, parseErr := parseAssets.Open(parseName)
	if parseErr != nil {
		return false
	}
	defer parseFile.Close()
	parseInfo, parseErr := parseFile.Stat()
	if parseErr != nil {
		return false
	}
	return !parseInfo.IsDir()
}

// serveFile writes one asset (the SPA shell) with a sensible content type.
func serveFile(parseW http.ResponseWriter, parseR *http.Request, parseAssets fs.FS, parseName string) {
	parseFile, parseErr := parseAssets.Open(parseName)
	if parseErr != nil {
		http.NotFound(parseW, parseR)
		return
	}
	defer parseFile.Close()
	if strings.HasSuffix(parseName, ".html") {
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	_, _ = io.Copy(parseW, parseFile)
}
