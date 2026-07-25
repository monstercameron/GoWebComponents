//go:build !js || !wasm

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

	"github.com/monstercameron/GoWebComponents/v5/diagnostics"
	"github.com/monstercameron/GoWebComponents/v5/head"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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

func newAppServer(parseRoot string) *appServer {
	return &appServer{
		wasmPath:     filepath.Join(parseRoot, "examples", "static", "bin", "ssr-server-routing.wasm"),
		cssPath:      filepath.Join(parseRoot, "examples", "static", "css", "tailwind.css"),
		loggerPath:   filepath.Join(parseRoot, "examples", "static", "script", "example-logger.js"),
		wasmExecPath: filepath.Join(parseRoot, "examples", "static", "script", "wasm_exec.js"),
	}
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

func (parseS *appServer) routes() http.Handler {
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/healthz", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "application/json")
		_, _ = parseW.Write([]byte(`{"ok":true}`))
	})
	parseMux.HandleFunc(bootstrapEndpointPath, parseS.handleBootstrap)
	parseMux.Handle("/assets/ssr-server-routing.wasm", fileHandler(parseS.wasmPath, "application/wasm"))
	parseMux.Handle("/assets/tailwind.css", fileHandler(parseS.cssPath, "text/css; charset=utf-8"))
	parseMux.Handle("/assets/example-logger.js", fileHandler(parseS.loggerPath, "text/javascript; charset=utf-8"))
	parseMux.Handle("/assets/wasm_exec.js", fileHandler(parseS.wasmExecPath, "text/javascript; charset=utf-8"))
	parseMux.HandleFunc("/", parseS.handlePage)
	return parseMux
}

func fileHandler(parsePath string, parseContentType string) http.Handler {
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseContentType != "" {
			parseW.Header().Set("Content-Type", parseContentType)
		}
		http.ServeFile(parseW, parseR, parsePath)
	})
}

func exampleServerRequestReport(parseSubject string, parsePath string, parseErr error, parseConsequence string, parseNext string) diagnostics.Report {
	return diagnostics.NewReport(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in " + strings.TrimSpace(parseSubject),
		Path:     strings.TrimSpace(parsePath),
		Runtime:  strings.TrimSpace(parseConsequence),
		Next:     strings.TrimSpace(parseNext),
		Docs:     exampleServerRequestDocs,
	})
}

func fatalExampleServerStartup(parseSubject string, parsePath string, parseErr error, parseNext string) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in " + strings.TrimSpace(parseSubject),
		Path:     strings.TrimSpace(parsePath),
		Runtime:  "the example server could not finish startup and no requests will be served.",
		Next:     strings.TrimSpace(parseNext),
		Docs:     exampleServerStartupDocs,
	}))
	os.Exit(1)
}

func (parseS *appServer) handleBootstrap(parseW http.ResponseWriter, parseR *http.Request) {
	parseQuery := cloneQueryWithoutPath(parseR.URL.Query())
	parseResolved := resolveRoute(parseR.URL.Query().Get("path"), parseQuery)
	if parseResolved.Redirect != "" {
		http.Redirect(parseW, parseR, parseResolved.Redirect, parseResolved.Status)
		return
	}
	parseData, parseErr := ui.MarshalSSRBootstrap(parseResolved.Bootstrap)
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handleBootstrap",
			parseR.URL.Path,
			parseErr,
			"the bootstrap payload was not serialized, so the client cannot hydrate this request.",
			"Inspect the resolved route bootstrap payload and serialization inputs for this request.",
		))
		return
	}
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = parseW.Write(parseData)
}

func cloneQueryWithoutPath(parseQuery url.Values) url.Values {
	parseClone := url.Values{}
	for parseKey, parseValues := range parseQuery {
		if parseKey == "path" {
			continue
		}
		parseClone[parseKey] = append([]string(nil), parseValues...)
	}
	return parseClone
}

func headDocumentForRoute(parseResolved resolvedRoute) head.Document {
	parseDocument := head.Document{
		Metadata: router.Metadata{
			Title:        parseResolved.Title,
			Description:  parseResolved.Description,
			CanonicalURL: parseResolved.CanonicalURL,
		},
		Social: head.SocialMetadata{
			Type:        "website",
			Title:       parseResolved.Title,
			Description: parseResolved.Description,
			URL:         parseResolved.CanonicalURL,
		},
		ResourceHints: []head.ResourceHint{
			{Rel: "preload", Href: "/assets/tailwind.css", As: "style"},
			{Rel: "preload", Href: "/assets/wasm_exec.js", As: "script"},
			{Rel: "preload", Href: "/assets/ssr-server-routing.wasm", As: "fetch", Type: "application/wasm"},
		},
	}

	switch parseResolved.View.Page {
	case serverPageDocs:
		parseDocument.Robots = "index,follow"
		parseDocument.JSONLD = []head.JSONLDBlock{
			{ID: "route-jsonld", Value: map[string]any{
				"@context":         "https://schema.org",
				"@type":            "TechArticle",
				"headline":         parseResolved.Title,
				"description":      parseResolved.Description,
				"mainEntityOfPage": parseResolved.CanonicalURL,
			}},
		}
		parseDocument.Alternates = []head.AlternateLink{
			{HrefLang: "en", Href: parseResolved.CanonicalURL},
			{HrefLang: "x-default", Href: parseResolved.CanonicalURL},
		}
	case serverPageSearch:
		parseDocument.Robots = "noindex,follow"
		parseDocument.JSONLD = []head.JSONLDBlock{
			{ID: "route-jsonld", Value: map[string]any{
				"@context": "https://schema.org",
				"@type":    "SearchResultsPage",
				"name":     parseResolved.Title,
				"url":      parseResolved.CanonicalURL,
			}},
		}
	case serverPageSecure, serverPageSignIn, serverPageNotFound:
		parseDocument.Robots = "noindex,nofollow"
	default:
		parseDocument.Robots = "index,follow"
		parseDocument.JSONLD = []head.JSONLDBlock{
			{ID: "route-jsonld", Value: map[string]any{
				"@context": "https://schema.org",
				"@type":    "WebPage",
				"name":     parseResolved.Title,
				"url":      parseResolved.CanonicalURL,
			}},
		}
	}

	return parseDocument
}

func (parseS *appServer) handlePage(parseW http.ResponseWriter, parseR *http.Request) {
	parseResolved := resolveRoute(parseR.URL.Path, parseR.URL.Query())
	if parseResolved.Redirect != "" {
		http.Redirect(parseW, parseR, parseResolved.Redirect, parseResolved.Status)
		return
	}
	parseStreamDeferredPanel := shouldStreamDeferredDocsPanel(parseResolved)
	parseBodyNode := renderDemoShell(parseResolved.View)
	if parseStreamDeferredPanel {
		parseBodyNode = renderDemoShellStreamShell(parseResolved.View)
	}
	parseBody, parseErr := ui.RenderToString(parseBodyNode)
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handlePage.renderDemoShell",
			parseR.URL.Path,
			parseErr,
			"the server could not render the demo shell, so this request returned HTTP 500 without HTML.",
			"Inspect the resolved route view and server render path for this request.",
		))
		return
	}
	parseHeadMetadata, parseErr := head.RenderToString(headDocumentForRoute(parseResolved))
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handlePage.metadata",
			parseR.URL.Path,
			parseErr,
			"the server could not render route metadata, so the HTML head for this request is incomplete.",
			"Inspect the metadata generation path and route metadata inputs for this request.",
		))
		return
	}
	parseRefScript, parseErr := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
		URL:    bootstrapReferenceURL(parseResolved.Path, parseR.URL.Query()),
		Format: ui.SSRBootstrapFormatJSON,
	}, "")
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, exampleServerRequestReport(
			"appServer.handlePage.bootstrapReference",
			parseR.URL.Path,
			parseErr,
			"the server could not generate the bootstrap reference script, so hydration cannot locate its payload.",
			"Inspect the bootstrap reference URL and script generation inputs for this request.",
		))
		return
	}
	parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	parseW.WriteHeader(parseResolved.Status)
	if parseStreamDeferredPanel {
		parseS.writeStreamedDocument(parseW, parseBody, parseResolved.View, parseHeadMetadata, parseRefScript)
		return
	}
	_, _ = fmt.Fprintf(parseW, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">%s<link rel=\"stylesheet\" href=\"/assets/tailwind.css\">%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body class=\"bg-[#07131d] text-slate-100 min-h-screen\"><div id=\"app\">%s</div><script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-server-routing.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>", parseHeadMetadata, parseRefScript, parseBody)
}

func shouldStreamDeferredDocsPanel(parseResolved resolvedRoute) bool {
	return parseResolved.View.Page == serverPageDocs && parseResolved.View.CurrentTab == serverTabLoader
}

func (parseS *appServer) writeStreamedDocument(parseW http.ResponseWriter, parseShellBody string, parseView demoShellView, parseHeadMetadata string, parseRefScript string) {
	parseFlusher, _ := parseW.(http.Flusher)
	_, _ = fmt.Fprintf(parseW, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">%s<link rel=\"stylesheet\" href=\"/assets/tailwind.css\">%s<script src=\"/assets/wasm_exec.js\"></script><script src=\"/assets/example-logger.js\"></script></head><body class=\"bg-[#07131d] text-slate-100 min-h-screen\"><div id=\"app\">%s</div>", parseHeadMetadata, parseRefScript, parseShellBody)
	if parseFlusher != nil {
		parseFlusher.Flush()
	}

	time.Sleep(120 * time.Millisecond)

	parseDeferredPanelMarkup, parseErr := ui.RenderToString(renderDeferredDocsPanel(parseView))
	if parseErr == nil {
		parseReplacementScript := streamedPanelReplacementScript(serverDeferredPanelID, parseDeferredPanelMarkup)
		if parseReplacementScript != "" {
			_, _ = parseW.Write([]byte(parseReplacementScript))
			if parseFlusher != nil {
				parseFlusher.Flush()
			}
		}
	}

	_, _ = parseW.Write([]byte("<script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/ssr-server-routing.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to load WASM:',err));</script></body></html>"))
}

func streamedPanelReplacementScript(parseTargetID string, parseMarkup string) string {
	parseEncodedTargetID, parseErr := json.Marshal(parseTargetID)
	if parseErr != nil {
		return ""
	}
	parseEncodedMarkup, parseErr := json.Marshal(parseMarkup)
	if parseErr != nil {
		return ""
	}
	return "<script>(function(){const target=document.getElementById(" + string(parseEncodedTargetID) + ");if(!target){return;}target.outerHTML=" + string(parseEncodedMarkup) + ";})();</script>"
}

func main() {
	parsePort := strings.TrimSpace(os.Getenv("PORT"))
	if parsePort == "" {
		parsePort = "8079"
	}
	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		fatalExampleServerStartup("main.os.Getwd", "cwd", parseErr, "Verify the example is being started from a readable working directory.")
	}
	parseRoot, parseErr := findRepoRoot(parseWorkingDir)
	if parseErr != nil {
		fatalExampleServerStartup("main.findRepoRoot", parseWorkingDir, parseErr, "Start the example inside the repo so the static assets and go.mod can be discovered.")
	}
	parseServer := newAppServer(parseRoot)
	fmt.Printf("Server SSR demo listening on http://127.0.0.1:%s\n", parsePort)
	if parseErr2 := http.ListenAndServe("127.0.0.1:"+parsePort, parseServer.routes()); parseErr2 != nil {
		fatalExampleServerStartup("main.http.ListenAndServe", "127.0.0.1:"+parsePort, parseErr2, "Free the port or update PORT before starting the example server again.")
	}
}
