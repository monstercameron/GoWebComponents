//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"fmt"
	templatehtml "html/template"
	"net/http"
	"os"
	"strings"

	"github.com/monstercameron/GoWebComponents/diagnostics"
	"github.com/monstercameron/GoWebComponents/head"
	gwchtml "github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	renderToStringRequestDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-request"
	renderToStringStartupDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-startup"
)

func renderToStringRequestReport(path string, err error) diagnostics.Report {
	return diagnostics.NewReport(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in handleRenderToString",
		Path:     strings.TrimSpace(path),
		Runtime:  "the request failed before ui.RenderToString could return HTML.",
		Next:     "Inspect the server render path and the component tree being serialized for this request.",
		Docs:     renderToStringRequestDocs,
	})
}

func fatalRenderToStringStartup(path string, err error) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in render-to-string demo",
		Path:     strings.TrimSpace(path),
		Runtime:  "the ui.RenderToString example server did not start, so no requests can be served.",
		Next:     "Verify the port is free and the example is being started with a valid server environment.",
		Docs:     renderToStringStartupDocs,
	}))
	os.Exit(1)
}

func renderCard() ui.Node {
	return gwchtml.Div(gwchtml.Props{Class: "rounded-[1.75rem] border border-slate-800 bg-slate-950 p-8 text-slate-100"},
		gwchtml.P(gwchtml.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, ui.Text("Server render")),
		gwchtml.H1(gwchtml.Props{Class: "mt-4 text-4xl font-black text-white"}, ui.Text("ui.RenderToString generated this card")),
		gwchtml.P(gwchtml.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, ui.Text("The server never mounted a browser runtime. It rendered this tree directly into an HTML string.")),
	)
}

func renderHeadMarkup() (string, error) {
	return head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:        "ui.RenderToString",
			Description:  "Minimal prerender-style HTML generation with the GoWebComponents head companion package.",
			CanonicalURL: "http://127.0.0.1:8084/",
		},
		Robots: "index,follow",
		Social: head.SocialMetadata{
			Type:        "article",
			Title:       "ui.RenderToString",
			Description: "Minimal prerender-style HTML generation with the GoWebComponents head companion package.",
			URL:         "http://127.0.0.1:8084/",
		},
		JSONLD: []head.JSONLDBlock{
			{ID: "render-to-string-jsonld", Value: map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "WebPage",
				"name":     "ui.RenderToString",
				"url":      "http://127.0.0.1:8084/",
			}},
		},
	})
}

func handleRenderToString(w http.ResponseWriter, r *http.Request) {
	markup, err := ui.RenderToString(renderCard())
	if err != nil {
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, renderToStringRequestReport(r.URL.Path, err))
		return
	}
	headMarkup, err := renderHeadMarkup()
	if err != nil {
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, renderToStringRequestReport(r.URL.Path, err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">%s<style>body{margin:0;background:#08111d;color:#e2e8f0;font-family:ui-sans-serif,system-ui,sans-serif}main{max-width:1100px;margin:0 auto;padding:48px 24px}section{display:grid;gap:24px}article{border:1px solid rgba(255,255,255,.08);background:rgba(15,23,42,.82);border-radius:28px;padding:28px}pre{overflow:auto;white-space:pre-wrap;word-break:break-word;background:#020617;border:1px solid rgba(255,255,255,.08);padding:20px;border-radius:20px;color:#cbd5e1}</style></head><body><main><section><article><p style=\"margin:0;font-size:12px;letter-spacing:.35em;text-transform:uppercase;color:#67e8f9\">Minimal SSR</p><h1 style=\"margin:16px 0 0;font-size:52px;line-height:1;font-weight:900\">ui.RenderToString</h1><p style=\"margin:16px 0 0;max-width:760px;font-size:18px;line-height:1.8;color:#cbd5e1\">This page shows the exact HTML string returned by the server render next to the rendered preview, including the same companion-owned head metadata you would emit in a prerender or static export flow.</p></article><article><h2 style=\"margin:0 0 16px;font-size:24px\">Returned HTML string</h2><pre>%s</pre></article><article><h2 style=\"margin:0 0 16px;font-size:24px\">Rendered preview</h2>%s</article></section></main></body></html>", headMarkup, templatehtml.HTMLEscapeString(markup), markup)
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8084"
	}
	http.HandleFunc("/", handleRenderToString)
	fmt.Printf("ui.RenderToString demo listening on http://127.0.0.1:%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
		fatalRenderToStringStartup("127.0.0.1:"+port, err)
	}
}
