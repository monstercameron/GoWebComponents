//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/diagnostics"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	bootstrapRequestDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-request"
	bootstrapStartupDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-startup"
)

func bootstrapRequestReport(path string, err error, consequence string, next string) diagnostics.Report {
	return diagnostics.Build(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in ssr-bootstrap demo",
		Path:     strings.TrimSpace(path),
		Runtime:  strings.TrimSpace(consequence),
		Next:     strings.TrimSpace(next),
		Docs:     bootstrapRequestDocs,
	})
}

func fatalBootstrapStartup(subject string, path string, err error, next string) {
	diagnostics.Emit(diagnostics.Build(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in " + strings.TrimSpace(subject),
		Path:     strings.TrimSpace(path),
		Runtime:  "the dedicated SSR bootstrap example could not start, so no requests will be served.",
		Next:     strings.TrimSpace(next),
		Docs:     bootstrapStartupDocs,
	}))
	os.Exit(1)
}

func repoRoot(start string) (string, error) {
	current := start
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find repo root from %s", start)
		}
		current = parent
	}
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8086"
	}
	wd, err := os.Getwd()
	if err != nil {
		fatalBootstrapStartup("main.os.Getwd", "cwd", err, "Verify the example is being started from a readable working directory.")
	}
	root, err := repoRoot(wd)
	if err != nil {
		fatalBootstrapStartup("main.repoRoot", wd, err, "Start the example inside the repo so the wasm binary and scripts can be discovered.")
	}
	loggerScript := filepath.Join(root, "examples", "static", "script", "example-logger.js")
	wasmExec := filepath.Join(root, "examples", "static", "script", "wasm_exec.js")
	wasmBinary := filepath.Join(root, "examples", "static", "bin", "ssr-bootstrap.wasm")

	http.HandleFunc("/assets/example-logger.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		http.ServeFile(w, r, loggerScript)
	})
	http.HandleFunc("/assets/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		http.ServeFile(w, r, wasmExec)
	})
	http.HandleFunc("/assets/ssr-bootstrap.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, wasmBinary)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		payload := ui.SSRBootstrap{
			Route: ui.SSRRouteBootstrap{Path: "/bootstrap"},
			Data:  map[string]interface{}{"message": "inline bootstrap payload"},
		}
		body, err := ui.RenderToString(renderBootstrapView(bootstrapViewFromPayload(payload)))
		if err != nil {
			diagnostics.WriteHTTPError(w, http.StatusInternalServerError, bootstrapRequestReport(
				r.URL.Path,
				err,
				"the request failed before the bootstrap view HTML could be rendered.",
				"Inspect the bootstrap view render path and the payload being serialized into the document.",
			))
			return
		}
		script, err := ui.RenderBootstrapScript(payload, "")
		if err != nil {
			diagnostics.WriteHTTPError(w, http.StatusInternalServerError, bootstrapRequestReport(
				r.URL.Path,
				err,
				"the bootstrap payload script was not generated, so the wasm client cannot hydrate this document.",
				"Inspect the bootstrap payload contents and script generation path for this request.",
			))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>Dedicated SSR bootstrap</title>%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body style=\"margin:0;background:#08111d\"><p id=\"client-status\" style=\"margin:0;padding:16px 24px;color:#67e8f9;font:600 14px/1.4 ui-sans-serif,system-ui,sans-serif\">Waiting for wasm hydration...</p><div id=\"app\">%s</div><script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-bootstrap.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>", script, body)
	})
	fmt.Printf("Dedicated SSR bootstrap demo listening on http://127.0.0.1:%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
		fatalBootstrapStartup("main.http.ListenAndServe", "127.0.0.1:"+port, err, "Free the port or update PORT before starting the example server again.")
	}
}
