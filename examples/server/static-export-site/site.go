//go:build !js || !wasm

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/monstercameron/GoWebComponents/v5/head"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/prerender"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type exportedPage struct {
	Path        string
	Title       string
	Eyebrow     string
	Headline    string
	Description string
	Callout     string
	HeroAsset   string
	LazyAsset   string
}

var exportedPages = []exportedPage{
	{
		Path:        "/",
		Title:       "Static export home",
		Eyebrow:     "Static export",
		Headline:    "Prerender several routes once and host them anywhere.",
		Description: "This example writes a small multi-page marketing site to files with no request-time Go server in the final deployment shape.",
		Callout:     "Use this route as the landing page when the content can be produced entirely at build time.",
		HeroAsset:   "hero-home",
		LazyAsset:   "panel-home",
	},
	{
		Path:        "/pricing",
		Title:       "Static export pricing",
		Eyebrow:     "Pricing",
		Headline:    "The pricing page is just another prerendered route.",
		Description: "Each route becomes one HTML file under dist plus any optional bootstrap sidecars. There is no special server contract for the pricing page.",
		Callout:     "Good fit for public plans, product comparison pages, or release notes that update during CI rather than per request.",
		HeroAsset:   "hero-pricing",
		LazyAsset:   "panel-pricing",
	},
	{
		Path:        "/docs/getting-started",
		Title:       "Static export docs",
		Eyebrow:     "Docs",
		Headline:    "Nested paths export to nested folders.",
		Description: "The docs route demonstrates the same canonical path mapping used by the prerender helper: nested routes become nested folders ending in index.html.",
		Callout:     "That shape is friendly to commodity static hosts and keeps route URLs stable without a custom runtime.",
		HeroAsset:   "hero-docs",
		LazyAsset:   "panel-docs",
	},
}

var assetManifest = map[string]string{
	"site-css":      "/static/site-export.4f3a2b1c.css",
	"hero-home":     "/static/media/hero-home.2f71c1a0.svg",
	"hero-pricing":  "/static/media/hero-pricing.8db274c4.svg",
	"hero-docs":     "/static/media/hero-docs.9af4b205.svg",
	"panel-home":    "/static/media/panel-home.614ca941.svg",
	"panel-pricing": "/static/media/panel-pricing.6411ca74.svg",
	"panel-docs":    "/static/media/panel-docs.3e85d1bf.svg",
}

func exportRoutes() []prerender.Route {
	parseRoutes := make([]prerender.Route, 0, len(exportedPages))
	for _, parsePage := range exportedPages {
		parsePage2 := parsePage
		parseRoutes = append(parseRoutes, prerender.Route{
			Path: parsePage2.Path,
			Build: func(parseTarget prerender.Target) (prerender.RouteOutput, error) {
				parseMarkup, parseErr := renderStaticExportPage(parsePage2)
				if parseErr != nil {
					return prerender.RouteOutput{}, parseErr
				}
				return prerender.RouteOutput{HTML: parseMarkup}, nil
			},
		})
	}
	return parseRoutes
}

func exportExampleSite(parseOutDir string) (prerender.ExportSummary, error) {
	parseSummary, parseErr := prerender.Export(parseOutDir, exportRoutes())
	if parseErr != nil {
		return prerender.ExportSummary{}, parseErr
	}
	if parseErr2 := copyExampleAssets(parseOutDir); parseErr2 != nil {
		return prerender.ExportSummary{}, parseErr2
	}
	return parseSummary, nil
}

var _ = exportExampleSite

func renderStaticExportPage(parsePage exportedPage) (string, error) {
	parseHeadMarkup, parseErr := head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:        parsePage.Title,
			Description:  parsePage.Description,
			CanonicalURL: "https://example.invalid" + parsePage.Path,
		},
		ResourceHints: resourceHintsForPage(parsePage),
	})
	if parseErr != nil {
		return "", parseErr
	}

	parseBodyMarkup, parseErr := ui.RenderToString(renderStaticExportBody(parsePage))
	if parseErr != nil {
		return "", parseErr
	}

	return fmt.Sprintf("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">%s<link rel=\"stylesheet\" href=\"%s\"></head><body>%s</body></html>", parseHeadMarkup, assetURL("site-css"), parseBodyMarkup), nil
}

func renderStaticExportBody(parsePage exportedPage) ui.Node {
	return html.Main(html.Props{},
		html.Div(html.Props{Class: "shell"},
			html.Section(html.Props{Class: "card"},
				html.P(html.Props{Class: "eyebrow"}, html.Text(parsePage.Eyebrow)),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(parsePage.Headline)),
				html.P(html.Props{Class: "lead"}, html.Text(parsePage.Description)),
				html.Div(html.Props{Class: "link-row"},
					staticLink("/", "Home"),
					staticLink("/pricing/", "Pricing"),
					staticLink("/docs/getting-started/", "Docs"),
				),
				html.Div(html.Props{Class: "media-frame"},
					html.Img(html.Props{
						Src: assetURL(parsePage.HeroAsset),
						Alt: parsePage.Headline,
						Raw: map[string]any{
							"srcset":   responsiveSrcSet(parsePage.HeroAsset),
							"sizes":    "(min-width: 1100px) 960px, 100vw",
							"decoding": "async",
							"width":    "1280",
							"height":   "720",
						},
					}),
				),
			),
			html.Section(html.Props{Class: "card"},
				html.H2(html.Props{Class: "text-2xl font-black text-white"}, html.Text("Why this example exists")),
				html.Div(html.Props{Class: "grid mt-4"},
					html.P(html.Props{Class: "lead"}, html.Text(parsePage.Callout)),
					html.P(html.Props{Class: "lead"}, html.Text("Run the export command once, point any static host at the generated dist directory, and the site loads without a custom Go request handler.")),
					html.P(html.Props{Class: "lead"}, html.Text("This is the current asset-delivery reference because it proves manifest-backed hashed asset URLs, route-scoped preload hints, responsive media markup, and lazy media loading in exported HTML.")),
					html.Div(html.Props{Class: "media-frame media-frame-secondary"},
						html.Img(html.Props{
							Src: assetURL(parsePage.LazyAsset),
							Alt: parsePage.Eyebrow + " secondary panel",
							Raw: map[string]any{
								"loading":  "lazy",
								"decoding": "async",
								"width":    "1280",
								"height":   "720",
							},
						}),
					),
				),
			),
		),
	)
}

func staticLink(parseHref string, parseLabel string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: "link-chip"}, html.Text(parseLabel))
}

func resourceHintsForPage(parsePage exportedPage) []head.ResourceHint {
	parseHints := []head.ResourceHint{
		{Rel: "preload", Href: assetURL("site-css"), As: "style"},
		{Rel: "preload", Href: assetURL(parsePage.HeroAsset), As: "image"},
	}
	if parsePage.Path == "/pricing" {
		parseHints = append(parseHints, head.ResourceHint{Rel: "prefetch", Href: assetURL("panel-docs"), As: "image"})
	}
	return parseHints
}

func assetURL(parseKey string) string {
	if parseValue, parseOk := assetManifest[parseKey]; parseOk {
		return parseValue
	}
	return ""
}

func responsiveSrcSet(parseKey string) string {
	parsePrimary := assetURL(parseKey)
	if parsePrimary == "" {
		return ""
	}
	return parsePrimary + " 1280w, " + parsePrimary + " 640w"
}

func copyExampleAssets(parseOutDir string) error {
	parseSourceRoot, parseErr := resolveExampleAssetRoot()
	if parseErr != nil {
		return parseErr
	}
	return filepath.Walk(parseSourceRoot, func(parsePath string, parseInfo os.FileInfo, parseErr2 error) error {
		if parseErr2 != nil {
			return parseErr2
		}
		if parseInfo.IsDir() {
			return nil
		}
		parseRelative, parseErr2 := filepath.Rel(parseSourceRoot, parsePath)
		if parseErr2 != nil {
			return parseErr2
		}
		parseTarget := filepath.Join(parseOutDir, parseRelative)
		if parseErr3 := os.MkdirAll(filepath.Dir(parseTarget), 0o755); parseErr3 != nil {
			return parseErr3
		}
		parseData, parseErr2 := os.ReadFile(parsePath)
		if parseErr2 != nil {
			return parseErr2
		}
		return os.WriteFile(parseTarget, parseData, 0o644)
	})
}

func resolveExampleAssetRoot() (string, error) {
	parseCandidates := []string{
		"assets",
		filepath.Join("examples", "102-static-export-site", "assets"),
	}
	for _, parseCandidate := range parseCandidates {
		if parseInfo, parseErr := os.Stat(parseCandidate); parseErr == nil && parseInfo.IsDir() {
			return parseCandidate, nil
		}
	}
	return "", fmt.Errorf("static export assets directory not found")
}
