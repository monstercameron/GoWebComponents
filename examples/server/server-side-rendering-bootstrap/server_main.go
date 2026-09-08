//go:build !js || !wasm

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/diagnostics"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const (
	bootstrapRequestDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-request"
	bootstrapStartupDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-startup"
)

func bootstrapRequestReport(parsePath string, parseErr error, parseConsequence string, parseNext string) diagnostics.Report {
	return diagnostics.NewReport(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in ssr-bootstrap demo",
		Path:     strings.TrimSpace(parsePath),
		Runtime:  strings.TrimSpace(parseConsequence),
		Next:     strings.TrimSpace(parseNext),
		Docs:     bootstrapRequestDocs,
	})
}

func fatalBootstrapStartup(parseSubject string, parsePath string, parseErr error, parseNext string) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in " + strings.TrimSpace(parseSubject),
		Path:     strings.TrimSpace(parsePath),
		Runtime:  "the dedicated SSR bootstrap example could not start, so no requests will be served.",
		Next:     strings.TrimSpace(parseNext),
		Docs:     bootstrapStartupDocs,
	}))
	os.Exit(1)
}

func repoRoot(parseStart string) (string, error) {
	parseCurrent := parseStart
	for {
		if _, parseErr := os.Stat(filepath.Join(parseCurrent, "go.mod")); parseErr == nil {
			return parseCurrent, nil
		}
		parseParent := filepath.Dir(parseCurrent)
		if parseParent == parseCurrent {
			return "", fmt.Errorf("could not find repo root from %s", parseStart)
		}
		parseCurrent = parseParent
	}
}

func main() {
	parsePort := strings.TrimSpace(os.Getenv("PORT"))
	if parsePort == "" {
		parsePort = "8086"
	}
	parseWd, parseErr := os.Getwd()
	if parseErr != nil {
		fatalBootstrapStartup("main.os.Getwd", "cwd", parseErr, "Verify the example is being started from a readable working directory.")
	}
	parseRoot, parseErr := repoRoot(parseWd)
	if parseErr != nil {
		fatalBootstrapStartup("main.repoRoot", parseWd, parseErr, "Start the example inside the repo so the wasm binary and scripts can be discovered.")
	}
	parseLoggerScript := filepath.Join(parseRoot, "examples", "static", "script", "example-logger.js")
	parseWasmExec := filepath.Join(parseRoot, "examples", "static", "script", "wasm_exec.js")
	parseWasmBinary := filepath.Join(parseRoot, "examples", "static", "bin", "ssr-bootstrap.wasm")

	http.HandleFunc("/assets/example-logger.js", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		http.ServeFile(parseW, parseR, parseLoggerScript)
	})
	http.HandleFunc("/assets/wasm_exec.js", func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
		parseW2.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		http.ServeFile(parseW2, parseR2, parseWasmExec)
	})
	http.HandleFunc("/assets/ssr-bootstrap.wasm", func(parseW3 http.ResponseWriter, parseR3 *http.Request) {
		parseW3.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(parseW3, parseR3, parseWasmBinary)
	})
	http.HandleFunc("/", func(parseW4 http.ResponseWriter, parseR4 *http.Request) {
		parsePayload := ui.SSRBootstrap{
			Route: ui.SSRRouteBootstrap{Path: "/bootstrap"},
			Data:  map[string]any{"message": "inline bootstrap payload"},
		}
		parseBody, parseErr2 := ui.RenderToString(renderBootstrapView(bootstrapViewFromPayload(parsePayload)))
		if parseErr2 != nil {
			diagnostics.WriteHTTPError(parseW4, http.StatusInternalServerError, bootstrapRequestReport(
				parseR4.URL.Path,
				parseErr2,
				"the request failed before the bootstrap view HTML could be rendered.",
				"Inspect the bootstrap view render path and the payload being serialized into the document.",
			))
			return
		}
		parseScript, parseErr2 := ui.RenderBootstrapScript(parsePayload, "")
		if parseErr2 != nil {
			diagnostics.WriteHTTPError(parseW4, http.StatusInternalServerError, bootstrapRequestReport(
				parseR4.URL.Path,
				parseErr2,
				"the bootstrap payload script was not generated, so the wasm client cannot hydrate this document.",
				"Inspect the bootstrap payload contents and script generation path for this request.",
			))
			return
		}
		parseW4.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(parseW4, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>Dedicated SSR bootstrap</title>%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body style=\"margin:0;background:#08111d\"><p id=\"client-status\" style=\"margin:0;padding:16px 24px;color:#67e8f9;font:600 14px/1.4 ui-sans-serif,system-ui,sans-serif\">Waiting for wasm hydration...</p><div id=\"app\">%s</div><script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-bootstrap.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>", parseScript, parseBody)
	})
	fmt.Printf("Dedicated SSR bootstrap demo listening on http://127.0.0.1:%s\n", parsePort)
	if parseErr3 := http.ListenAndServe("127.0.0.1:"+parsePort, nil); parseErr3 != nil {
		fatalBootstrapStartup("main.http.ListenAndServe", "127.0.0.1:"+parsePort, parseErr3, "Free the port or update PORT before starting the example server again.")
	}
}
