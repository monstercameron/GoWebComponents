package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/monstercameron/GoWebComponents/head"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/prerender"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
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
	routes := make([]prerender.Route, 0, len(exportedPages))
	for _, page := range exportedPages {
		page := page
		routes = append(routes, prerender.Route{
			Path: page.Path,
			Build: func(target prerender.Target) (prerender.RouteOutput, error) {
				markup, err := renderStaticExportPage(page)
				if err != nil {
					return prerender.RouteOutput{}, err
				}
				return prerender.RouteOutput{HTML: markup}, nil
			},
		})
	}
	return routes
}

func exportExampleSite(outDir string) (prerender.ExportSummary, error) {
	summary, err := prerender.Export(outDir, exportRoutes())
	if err != nil {
		return prerender.ExportSummary{}, err
	}
	if err := copyExampleAssets(outDir); err != nil {
		return prerender.ExportSummary{}, err
	}
	return summary, nil
}

func renderStaticExportPage(page exportedPage) (string, error) {
	headMarkup, err := head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:        page.Title,
			Description:  page.Description,
			CanonicalURL: "https://example.invalid" + page.Path,
		},
		ResourceHints: resourceHintsForPage(page),
	})
	if err != nil {
		return "", err
	}

	bodyMarkup, err := ui.RenderToString(renderStaticExportBody(page))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">%s<link rel=\"stylesheet\" href=\"%s\"></head><body>%s</body></html>", headMarkup, assetURL("site-css"), bodyMarkup), nil
}

func renderStaticExportBody(page exportedPage) ui.Node {
	return html.Main(html.Props{},
		html.Div(html.Props{Class: "shell"},
			html.Section(html.Props{Class: "card"},
				html.P(html.Props{Class: "eyebrow"}, html.Text(page.Eyebrow)),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(page.Headline)),
				html.P(html.Props{Class: "lead"}, html.Text(page.Description)),
				html.Div(html.Props{Class: "link-row"},
					staticLink("/", "Home"),
					staticLink("/pricing/", "Pricing"),
					staticLink("/docs/getting-started/", "Docs"),
				),
				html.Div(html.Props{Class: "media-frame"},
					html.Img(html.Props{
						Src: assetURL(page.HeroAsset),
						Alt: page.Headline,
						Raw: map[string]interface{}{
							"srcset":   responsiveSrcSet(page.HeroAsset),
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
					html.P(html.Props{Class: "lead"}, html.Text(page.Callout)),
					html.P(html.Props{Class: "lead"}, html.Text("Run the export command once, point any static host at the generated dist directory, and the site loads without a custom Go request handler.")),
					html.P(html.Props{Class: "lead"}, html.Text("This is the current asset-delivery reference because it proves manifest-backed hashed asset URLs, route-scoped preload hints, responsive media markup, and lazy media loading in exported HTML.")),
					html.Div(html.Props{Class: "media-frame media-frame-secondary"},
						html.Img(html.Props{
							Src: assetURL(page.LazyAsset),
							Alt: page.Eyebrow + " secondary panel",
							Raw: map[string]interface{}{
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

func staticLink(href string, label string) ui.Node {
	return html.A(html.Props{Href: href, Class: "link-chip"}, html.Text(label))
}

func resourceHintsForPage(page exportedPage) []head.ResourceHint {
	hints := []head.ResourceHint{
		{Rel: "preload", Href: assetURL("site-css"), As: "style"},
		{Rel: "preload", Href: assetURL(page.HeroAsset), As: "image"},
	}
	if page.Path == "/pricing" {
		hints = append(hints, head.ResourceHint{Rel: "prefetch", Href: assetURL("panel-docs"), As: "image"})
	}
	return hints
}

func assetURL(key string) string {
	if value, ok := assetManifest[key]; ok {
		return value
	}
	return ""
}

func responsiveSrcSet(key string) string {
	primary := assetURL(key)
	if primary == "" {
		return ""
	}
	return primary + " 1280w, " + primary + " 640w"
}

func copyExampleAssets(outDir string) error {
	sourceRoot, err := resolveExampleAssetRoot()
	if err != nil {
		return err
	}
	return filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(outDir, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func resolveExampleAssetRoot() (string, error) {
	candidates := []string{
		"assets",
		filepath.Join("examples", "102-static-export-site", "assets"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("static export assets directory not found")
}
