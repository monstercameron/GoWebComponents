//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/diagnostics"
	"github.com/monstercameron/GoWebComponents/head"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	exampleServerRequestDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-request"
	exampleServerStartupDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-startup"
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

func exampleServerRequestReport(subject string, path string, err error, consequence string, next string) diagnostics.Report {
	return diagnostics.NewReport(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in " + strings.TrimSpace(subject),
		Path:     strings.TrimSpace(path),
		Runtime:  strings.TrimSpace(consequence),
		Next:     strings.TrimSpace(next),
		Docs:     exampleServerRequestDocs,
	})
}

func fatalExampleServerStartup(subject string, path string, err error, next string) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in " + strings.TrimSpace(subject),
		Path:     strings.TrimSpace(path),
		Runtime:  "the example server could not finish startup and no requests will be served.",
		Next:     strings.TrimSpace(next),
		Docs:     exampleServerStartupDocs,
	}))
	os.Exit(1)
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
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handleBootstrap",
			r.URL.Path,
			err,
			"the bootstrap payload was not serialized, so the client cannot hydrate this request.",
			"Inspect the resolved route bootstrap payload and serialization inputs for this request.",
		))
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

func headDocumentForRoute(resolved resolvedRoute) head.Document {
	document := head.Document{
		Metadata: router.Metadata{
			Title:        resolved.Title,
			Description:  resolved.Description,
			CanonicalURL: resolved.CanonicalURL,
		},
		Social: head.SocialMetadata{
			Type:        "website",
			Title:       resolved.Title,
			Description: resolved.Description,
			URL:         resolved.CanonicalURL,
		},
		ResourceHints: []head.ResourceHint{
			{Rel: "preload", Href: "/assets/tailwind.css", As: "style"},
			{Rel: "preload", Href: "/assets/wasm_exec.js", As: "script"},
			{Rel: "preload", Href: "/assets/ssr-server-routing.wasm", As: "fetch", Type: "application/wasm"},
		},
	}

	switch resolved.View.Page {
	case serverPageDocs:
		document.Robots = "index,follow"
		document.JSONLD = []head.JSONLDBlock{
			{ID: "route-jsonld", Value: map[string]interface{}{
				"@context":         "https://schema.org",
				"@type":            "TechArticle",
				"headline":         resolved.Title,
				"description":      resolved.Description,
				"mainEntityOfPage": resolved.CanonicalURL,
			}},
		}
		document.Alternates = []head.AlternateLink{
			{HrefLang: "en", Href: resolved.CanonicalURL},
			{HrefLang: "x-default", Href: resolved.CanonicalURL},
		}
	case serverPageSearch:
		document.Robots = "noindex,follow"
		document.JSONLD = []head.JSONLDBlock{
			{ID: "route-jsonld", Value: map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "SearchResultsPage",
				"name":     resolved.Title,
				"url":      resolved.CanonicalURL,
			}},
		}
	case serverPageSecure, serverPageSignIn, serverPageNotFound:
		document.Robots = "noindex,nofollow"
	default:
		document.Robots = "index,follow"
		document.JSONLD = []head.JSONLDBlock{
			{ID: "route-jsonld", Value: map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "WebPage",
				"name":     resolved.Title,
				"url":      resolved.CanonicalURL,
			}},
		}
	}

	return document
}

func (s *appServer) handlePage(w http.ResponseWriter, r *http.Request) {
	resolved := resolveRoute(r.URL.Path, r.URL.Query())
	if resolved.Redirect != "" {
		http.Redirect(w, r, resolved.Redirect, resolved.Status)
		return
	}
	streamDeferredPanel := shouldStreamDeferredDocsPanel(resolved)
	bodyNode := renderDemoShell(resolved.View)
	if streamDeferredPanel {
		bodyNode = renderDemoShellStreamShell(resolved.View)
	}
	body, err := ui.RenderToString(bodyNode)
	if err != nil {
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handlePage.renderDemoShell",
			r.URL.Path,
			err,
			"the server could not render the demo shell, so this request returned HTTP 500 without HTML.",
			"Inspect the resolved route view and server render path for this request.",
		))
		return
	}
	headMetadata, err := head.RenderToString(headDocumentForRoute(resolved))
	if err != nil {
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handlePage.metadata",
			r.URL.Path,
			err,
			"the server could not render route metadata, so the HTML head for this request is incomplete.",
			"Inspect the metadata generation path and route metadata inputs for this request.",
		))
		return
	}
	refScript, err := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
		URL:    bootstrapReferenceURL(resolved.Path, r.URL.Query()),
		Format: ui.SSRBootstrapFormatJSON,
	}, "")
	if err != nil {
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handlePage.bootstrapReference",
			r.URL.Path,
			err,
			"the server could not generate the bootstrap reference script, so hydration cannot locate its payload.",
			"Inspect the bootstrap reference URL and script generation inputs for this request.",
		))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(resolved.Status)
	if streamDeferredPanel {
		s.writeStreamedDocument(w, body, resolved.View, headMetadata, refScript)
		return
	}
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">%s<link rel=\"stylesheet\" href=\"/assets/tailwind.css\">%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body class=\"bg-[#07131d] text-slate-100 min-h-screen\"><div id=\"app\">%s</div><script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-server-routing.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>", headMetadata, refScript, body)
}

func shouldStreamDeferredDocsPanel(resolved resolvedRoute) bool {
	return resolved.View.Page == serverPageDocs && resolved.View.CurrentTab == serverTabLoader
}

func (s *appServer) writeStreamedDocument(w http.ResponseWriter, shellBody string, view demoShellView, headMetadata string, refScript string) {
	flusher, _ := w.(http.Flusher)
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">%s<link rel=\"stylesheet\" href=\"/assets/tailwind.css\">%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body class=\"bg-[#07131d] text-slate-100 min-h-screen\"><div id=\"app\">%s</div>", headMetadata, refScript, shellBody)
	if flusher != nil {
		flusher.Flush()
	}

	time.Sleep(120 * time.Millisecond)

	deferredPanelMarkup, err := ui.RenderToString(renderDeferredDocsPanel(view))
	if err == nil {
		replacementScript := streamedPanelReplacementScript(serverDeferredPanelID, deferredPanelMarkup)
		if replacementScript != "" {
			_, _ = w.Write([]byte(replacementScript))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}

	_, _ = w.Write([]byte("<script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-server-routing.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>"))
}

func streamedPanelReplacementScript(targetID string, markup string) string {
	encodedTargetID, err := json.Marshal(targetID)
	if err != nil {
		return ""
	}
	encodedMarkup, err := json.Marshal(markup)
	if err != nil {
		return ""
	}
	return "<script>(function(){const target=document.getElementById(" + string(encodedTargetID) + ");if(!target){return;}target.outerHTML=" + string(encodedMarkup) + ";})();</script>"
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8079"
	}
	workingDir, err := os.Getwd()
	if err != nil {
		fatalExampleServerStartup("main.os.Getwd", "cwd", err, "Verify the example is being started from a readable working directory.")
	}
	root, err := findRepoRoot(workingDir)
	if err != nil {
		fatalExampleServerStartup("main.findRepoRoot", workingDir, err, "Start the example inside the repo so the static assets and go.mod can be discovered.")
	}
	server := newAppServer(root)
	fmt.Printf("Server SSR demo listening on http://127.0.0.1:%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, server.routes()); err != nil {
		fatalExampleServerStartup("main.http.ListenAndServe", "127.0.0.1:"+port, err, "Free the port or update PORT before starting the example server again.")
	}
}
