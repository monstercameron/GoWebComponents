//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

type appServer struct {
	wasmPath     string
	cssPath      string
	loggerPath   string
	wasmExecPath string
}

func newAppServer(root string) *appServer {
	return &appServer{
		wasmPath:     filepath.Join(root, "examples", "static", "bin", "ssr-server-routing.wasm"),
		cssPath:      filepath.Join(root, "examples", "static", "css", "tailwind.css"),
		loggerPath:   filepath.Join(root, "examples", "static", "script", "example-logger.js"),
		wasmExecPath: filepath.Join(root, "examples", "static", "script", "wasm_exec.js"),
	}
}

func findRepoRoot(start string) (string, error) {
	current := start
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not locate repo root from %s", start)
		}
		current = parent
	}
}

func (s *appServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc(bootstrapEndpointPath, s.handleBootstrap)
	mux.Handle("/assets/ssr-server-routing.wasm", fileHandler(s.wasmPath, "application/wasm"))
	mux.Handle("/assets/tailwind.css", fileHandler(s.cssPath, "text/css; charset=utf-8"))
	mux.Handle("/assets/example-logger.js", fileHandler(s.loggerPath, "text/javascript; charset=utf-8"))
	mux.Handle("/assets/wasm_exec.js", fileHandler(s.wasmExecPath, "text/javascript; charset=utf-8"))
	mux.HandleFunc("/", s.handlePage)
	return mux
}

func fileHandler(path string, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		http.ServeFile(w, r, path)
	})
}

func (s *appServer) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	query := cloneQueryWithoutPath(r.URL.Query())
	resolved := resolveRoute(r.URL.Query().Get("path"), query)
	if resolved.Redirect != "" {
		http.Redirect(w, r, resolved.Redirect, resolved.Status)
		return
	}
	data, err := ui.MarshalSSRBootstrap(resolved.Bootstrap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

func cloneQueryWithoutPath(query url.Values) url.Values {
	clone := url.Values{}
	for key, values := range query {
		if key == "path" {
			continue
		}
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func (s *appServer) handlePage(w http.ResponseWriter, r *http.Request) {
	resolved := resolveRoute(r.URL.Path, r.URL.Query())
	if resolved.Redirect != "" {
		http.Redirect(w, r, resolved.Redirect, resolved.Status)
		return
	}
	body, err := ui.RenderToString(renderDemoShell(resolved.View))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	refScript, err := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
		URL:    bootstrapReferenceURL(resolved.Path, r.URL.Query()),
		Format: ui.SSRBootstrapFormatJSON,
	}, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(resolved.Status)
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>%s</title><meta name=\"description\" content=\"%s\"><link rel=\"canonical\" href=\"%s\"><link rel=\"stylesheet\" href=\"/assets/tailwind.css\">%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body class=\"bg-[#07131d] text-slate-100 min-h-screen\"><div id=\"app\">%s</div><script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-server-routing.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>", template.HTMLEscapeString(resolved.Title), template.HTMLEscapeString(resolved.Description), template.HTMLEscapeString(resolved.CanonicalURL), refScript, body)
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8079"
	}
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root, err := findRepoRoot(workingDir)
	if err != nil {
		panic(err)
	}
	server := newAppServer(root)
	fmt.Printf("Server SSR demo listening on http://127.0.0.1:%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, server.routes()); err != nil {
		panic(err)
	}
}
