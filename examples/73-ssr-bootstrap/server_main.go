//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

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
		panic(err)
	}
	root, err := repoRoot(wd)
	if err != nil {
		panic(err)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		script, err := ui.RenderBootstrapScript(payload, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>Dedicated SSR bootstrap</title>%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body style=\"margin:0;background:#08111d\"><p id=\"client-status\" style=\"margin:0;padding:16px 24px;color:#67e8f9;font:600 14px/1.4 ui-sans-serif,system-ui,sans-serif\">Waiting for wasm hydration...</p><div id=\"app\">%s</div><script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-bootstrap.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>", script, body)
	})
	fmt.Printf("Dedicated SSR bootstrap demo listening on http://127.0.0.1:%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
		panic(err)
	}
}
