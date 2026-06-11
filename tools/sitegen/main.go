// Command sitegen statically generates the GoWebComponents docs site.
//
// Every page is prerendered through the framework's own ui.RenderToString —
// the docs site is itself a demonstration of the SSR pipeline. Output is
// plain HTML + one stylesheet + one script; wasm loads only on example pages.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/prerender"
	"github.com/monstercameron/GoWebComponents/ui"
)

//go:embed assets/site.css
var siteCSS string

//go:embed assets/site.js
var siteJS string

func main() {
	parseRepoRoot := flag.String("root", ".", "repository root")
	parseOutDir := flag.String("out", "examples/site-dist", "output directory for the generated site")
	parseServeAddr := flag.String("serve", "", "after generating, serve the site locally on this address (e.g. 127.0.0.1:8090)")
	flag.Parse()

	if parseErr := generateSite(*parseRepoRoot, *parseOutDir); parseErr != nil {
		fmt.Fprintln(os.Stderr, "sitegen:", parseErr)
		os.Exit(1)
	}
	if *parseServeAddr != "" {
		if parseErr := servePreview(*parseRepoRoot, *parseOutDir, *parseServeAddr); parseErr != nil {
			fmt.Fprintln(os.Stderr, "sitegen:", parseErr)
			os.Exit(1)
		}
	}
}

// servePreview hosts the generated site with the same path layout as the
// GitHub Pages deployment: the catalog app and shared static assets are
// mounted beside the prerendered pages.
func servePreview(parseRepoRoot string, parseOutDir string, parseAddr string) error {
	parseMux := http.NewServeMux()
	parseMux.Handle("/public-examples-site/", http.StripPrefix("/public-examples-site/",
		http.FileServer(http.Dir(filepath.Join(parseRepoRoot, "examples", "public-examples-site")))))
	parseMux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(filepath.Join(parseRepoRoot, "examples", "static")))))
	parseMux.Handle("/", http.FileServer(http.Dir(parseOutDir)))
	fmt.Printf("sitegen: serving site at http://%s/\n", parseAddr)
	return http.ListenAndServe(parseAddr, parseMux)
}

// generateSite renders every static page and writes site assets.
func generateSite(parseRepoRoot string, parseOutDir string) error {
	parseManifest, parseErr := loadCatalog(parseRepoRoot)
	if parseErr != nil {
		return parseErr
	}
	parseChapters, parseErr2 := loadChapters(parseRepoRoot)
	if parseErr2 != nil {
		return parseErr2
	}
	parseExamples := itemsOfType(parseManifest, "Example")
	parseConcepts := itemsOfType(parseManifest, "Concept")
	parseAPIItems := itemsOfType(parseManifest, "API")
	parseHeroSource := loadHeroSource(parseRepoRoot)

	parseRoutes := []prerender.Route{
		buildRoute("/", renderLandingPage(parseHeroSource, len(parseExamples), len(parseChapters))),
		buildRoute("/learn/", renderLearnIndex(parseChapters, parseConcepts)),
		buildRoute("/examples/", renderExamplesGallery(parseExamples)),
		buildRoute("/api/", renderAPIIndex(parseAPIItems)),
	}
	for parseIndex := range parseChapters {
		parseRoutes = append(parseRoutes,
			buildRoute("/learn/"+parseChapters[parseIndex].Slug+"/", renderChapterPage(parseChapters, parseIndex)))
	}
	parseConceptSlugsByDoc := map[string]string{}
	for _, parseConcept := range parseConcepts {
		if strings.HasSuffix(parseConcept.Content.SourcePath, ".md") {
			parseConceptSlugsByDoc[filepath.Base(parseConcept.Content.SourcePath)] = slugify(parseConcept.Title)
		}
	}
	for _, parseConcept := range parseConcepts {
		parseMarkdown := loadConceptMarkdown(parseRepoRoot, parseConcept)
		parseRoutes = append(parseRoutes,
			buildRoute("/learn/concepts/"+slugify(parseConcept.Title)+"/", renderConceptPage(parseConcept, parseMarkdown, parseConceptSlugsByDoc)))
	}

	parseSummary, parseExportErr := prerender.Export(parseOutDir, parseRoutes)
	if parseExportErr != nil {
		return parseExportErr
	}

	if parseAssetErr := writeSiteAssets(parseOutDir, parseManifest, parseChapters); parseAssetErr != nil {
		return parseAssetErr
	}

	fmt.Printf("sitegen: wrote %d pages to %s\n", len(parseSummary.HTMLFiles), parseOutDir)
	return nil
}

// buildRoute wraps a rendered page node as a prerender route.
func buildRoute(parsePath string, parsePage ui.Node) prerender.Route {
	return prerender.Route{
		Path: parsePath,
		Build: func(parseTarget prerender.Target) (prerender.RouteOutput, error) {
			parseMarkup, parseErr := ui.RenderToString(parsePage)
			if parseErr != nil {
				return prerender.RouteOutput{}, fmt.Errorf("render %s: %w", parsePath, parseErr)
			}
			return prerender.RouteOutput{HTML: "<!DOCTYPE html>\n" + parseMarkup}, nil
		},
	}
}

// loadConceptMarkdown returns the concept's full markdown source when one exists.
func loadConceptMarkdown(parseRepoRoot string, parseConcept catalogItem) string {
	parseSourcePath := strings.TrimSpace(parseConcept.Content.SourcePath)
	if parseSourcePath == "" || !strings.HasSuffix(parseSourcePath, ".md") {
		return ""
	}
	parseRaw, parseErr := os.ReadFile(filepath.Join(parseRepoRoot, "examples", "public-examples-site", filepath.FromSlash(parseSourcePath)))
	if parseErr != nil {
		return ""
	}
	return string(parseRaw)
}

// writeSiteAssets emits the stylesheet, script, and search index.
func writeSiteAssets(parseOutDir string, parseManifest catalogManifest, parseChapters []chapter) error {
	parseSiteDir := filepath.Join(parseOutDir, "site")
	if parseErr := os.MkdirAll(parseSiteDir, 0o755); parseErr != nil {
		return fmt.Errorf("sitegen: create site asset dir: %w", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseSiteDir, "site.css"), []byte(siteCSS), 0o644); parseErr != nil {
		return fmt.Errorf("sitegen: write site.css: %w", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseSiteDir, "site.js"), []byte(siteJS), 0o644); parseErr != nil {
		return fmt.Errorf("sitegen: write site.js: %w", parseErr)
	}

	parseIndex := buildSearchIndex(parseManifest, parseChapters)
	parseEncoded, parseErr := json.Marshal(parseIndex)
	if parseErr != nil {
		return fmt.Errorf("sitegen: encode search index: %w", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseSiteDir, "search-index.json"), parseEncoded, 0o644); parseErr2 != nil {
		return fmt.Errorf("sitegen: write search index: %w", parseErr2)
	}
	return nil
}

// buildSearchIndex flattens chapters, concepts, examples, and API groups into
// one client-searchable record list.
func buildSearchIndex(parseManifest catalogManifest, parseChapters []chapter) []searchEntry {
	var parseEntries []searchEntry
	for _, parseChapter := range parseChapters {
		parseEntries = append(parseEntries, searchEntry{
			Title:   parseChapter.Title,
			Href:    "learn/" + parseChapter.Slug + "/index.html",
			Kind:    "Manual",
			Snippet: parseChapter.Summary,
		})
	}
	for _, parseItem := range parseManifest.Items {
		parseEntry := searchEntry{
			Title:   parseItem.Title,
			Kind:    parseItem.Type,
			Tags:    strings.Join(append(append([]string{}, parseItem.Tags...), parseItem.SearchTags...), " "),
			Snippet: parseItem.Blurb,
		}
		switch parseItem.Type {
		case "Concept":
			parseEntry.Href = "learn/concepts/" + slugify(parseItem.Title) + "/index.html"
		case "Example":
			parseEntry.Href = "public-examples-site/" + parseItem.Content.PreviewPath
		case "API":
			parseEntry.Href = "public-examples-site/" + parseItem.Content.SourcePath
			if parseItem.Content.AnchorID != "" {
				parseEntry.Href += "#" + parseItem.Content.AnchorID
			}
		default:
			continue
		}
		parseEntries = append(parseEntries, parseEntry)
	}
	return parseEntries
}
