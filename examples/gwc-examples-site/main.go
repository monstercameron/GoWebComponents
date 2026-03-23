//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/pwa"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	catalogStateKey    = "gwc:examples:catalog:filters"
	catalogCacheKey    = "gwc:examples:catalog:payload"
	catalogCacheName   = "gwc-examples-runtime-v1"
	catalogCachePrefix = "gwc-examples-runtime-"
)

var catalogPayloadMaxAge = 5 * time.Minute

var catalogFeatureOrder = []string{
	"ui",
	"html",
	"forms",
	"state",
	"fetch",
	"router",
	"interop",
	"devtools",
	"pwa",
	"i18n",
	"ssr",
	"hydration",
	"workers",
	"custom-elements",
	"overlays",
	"accessibility",
	"scheduling",
	"offline",
	"code-splitting",
}

var catalogFeatureLabels = map[string]string{
	"ui":              "UI",
	"html":            "HTML",
	"forms":           "Forms",
	"state":           "State",
	"fetch":           "Fetch",
	"router":          "Router",
	"interop":         "Interop",
	"devtools":        "Devtools",
	"pwa":             "PWA",
	"i18n":            "i18n",
	"ssr":             "SSR",
	"hydration":       "Hydration",
	"workers":         "Workers",
	"custom-elements": "Custom elements",
	"overlays":        "Overlays",
	"accessibility":   "Accessibility",
	"scheduling":      "Scheduling",
	"offline":         "Offline",
	"code-splitting":  "Code splitting",
}

type catalogPayload struct {
	GeneratedAt         string         `json:"generatedAt"`
	TotalExamples       int            `json:"totalExamples"`
	WasmExamples        int            `json:"wasmExamples"`
	MultiClientExamples int            `json:"multiClientExamples"`
	Examples            []catalogEntry `json:"examples"`
}

type catalogEntry struct {
	Name        string   `json:"name"`
	Href        string   `json:"href"`
	HTMLFile    string   `json:"htmlFile"`
	UsesWasm    bool     `json:"usesWasm"`
	WasmBinary  string   `json:"wasmBinary"`
	MultiClient bool     `json:"multiClient"`
	Tags        []string `json:"tags"`
}

type catalogFilterState struct {
	Query    string   `json:"query"`
	Mode     string   `json:"mode"`
	Features []string `json:"features,omitempty"`
}

type cachedCatalogPayload struct {
	SourceURL string         `json:"sourceURL"`
	CachedAt  string         `json:"cachedAt"`
	Payload   catalogPayload `json:"payload"`
}

type examplesRuntimeConfig struct {
	Mode        string
	RoutePath   string
	CatalogURL  string
	AssetBase   string
	WasmBase    string
	CatalogHref string
	ExampleSlug string
}

func normalizeCatalogMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "wasm":
		return "wasm"
	case "multi-client":
		return "multi-client"
	default:
		return "all"
	}
}

func normalizeCatalogFeatures(features []string) []string {
	seen := map[string]struct{}{}
	for _, feature := range features {
		feature = strings.TrimSpace(strings.ToLower(feature))
		if feature == "" {
			continue
		}
		seen[feature] = struct{}{}
	}
	ordered := make([]string, 0, len(seen))
	for _, feature := range catalogFeatureOrder {
		if _, ok := seen[feature]; ok {
			ordered = append(ordered, feature)
			delete(seen, feature)
		}
	}
	if len(seen) > 0 {
		extra := make([]string, 0, len(seen))
		for feature := range seen {
			extra = append(extra, feature)
		}
		sort.Strings(extra)
		ordered = append(ordered, extra...)
	}
	return ordered
}

func catalogFeatureLabel(feature string) string {
	feature = strings.TrimSpace(strings.ToLower(feature))
	if label, ok := catalogFeatureLabels[feature]; ok {
		return label
	}
	parts := strings.Fields(strings.ReplaceAll(feature, "-", " "))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func availableCatalogFeatures(entries []catalogEntry) []string {
	present := map[string]struct{}{}
	for _, entry := range entries {
		for _, tag := range entry.Tags {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if _, ok := catalogFeatureLabels[tag]; ok {
				present[tag] = struct{}{}
			}
		}
	}
	available := make([]string, 0, len(present))
	for _, feature := range catalogFeatureOrder {
		if _, ok := present[feature]; ok {
			available = append(available, feature)
		}
	}
	return available
}

func toggleCatalogFeature(features []string, feature string, enabled bool) []string {
	feature = strings.TrimSpace(strings.ToLower(feature))
	next := make([]string, 0, len(features)+1)
	for _, existing := range normalizeCatalogFeatures(features) {
		if existing == feature {
			continue
		}
		next = append(next, existing)
	}
	if enabled && feature != "" {
		next = append(next, feature)
	}
	return normalizeCatalogFeatures(next)
}

func featureSummaryLabel(features []string) string {
	features = normalizeCatalogFeatures(features)
	if len(features) == 0 {
		return "All framework features"
	}
	if len(features) == 1 {
		return catalogFeatureLabel(features[0])
	}
	if len(features) == 2 {
		return catalogFeatureLabel(features[0]) + " and " + catalogFeatureLabel(features[1])
	}
	return fmt.Sprintf("%s, %s, and %d more", catalogFeatureLabel(features[0]), catalogFeatureLabel(features[1]), len(features)-2)
}

func featureStorageValue(features []string) string {
	return strings.Join(normalizeCatalogFeatures(features), ",")
}

func filterCatalogEntries(entries []catalogEntry, query string, mode string, selectedFeatures []string) []catalogEntry {
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	selectedFeatures = normalizeCatalogFeatures(selectedFeatures)
	filtered := make([]catalogEntry, 0, len(entries))
	for _, entry := range entries {
		if mode == "wasm" && !entry.UsesWasm {
			continue
		}
		if mode == "multi-client" && !entry.MultiClient {
			continue
		}
		if len(selectedFeatures) > 0 {
			matchesFeature := false
			for _, feature := range selectedFeatures {
				for _, tag := range entry.Tags {
					if strings.EqualFold(strings.TrimSpace(tag), feature) {
						matchesFeature = true
						break
					}
				}
				if matchesFeature {
					break
				}
			}
			if !matchesFeature {
				continue
			}
		}
		if trimmedQuery != "" {
			haystack := strings.ToLower(entry.Name + " " + entry.Href + " " + entry.WasmBinary + " " + strings.Join(entry.Tags, " "))
			matches := true
			for _, term := range strings.Fields(trimmedQuery) {
				if !strings.Contains(haystack, term) {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func catalogSummary(query string, mode string, selectedFeatures []string) string {
	parts := []string{"Browsing the wasm-first examples catalog"}
	if mode == "wasm" {
		parts = append(parts, "filtering to wasm-backed examples")
	}
	if mode == "multi-client" {
		parts = append(parts, "filtering to multi-client coordination demos")
	}
	selectedFeatures = normalizeCatalogFeatures(selectedFeatures)
	if len(selectedFeatures) > 0 {
		labels := make([]string, 0, len(selectedFeatures))
		for _, feature := range selectedFeatures {
			labels = append(labels, catalogFeatureLabel(feature))
		}
		parts = append(parts, fmt.Sprintf("matching framework features %s", strings.Join(labels, ", ")))
	}
	if strings.TrimSpace(query) != "" {
		parts = append(parts, fmt.Sprintf("searching for %q", strings.TrimSpace(query)))
	}
	return strings.Join(parts, " and ") + "."
}

func loadPersistedCatalogFilters() (catalogFilterState, bool) {
	storage, err := interop.LocalStorage()
	if err != nil {
		return catalogFilterState{}, false
	}
	value, ok, err := storage.GetItem(catalogStateKey)
	if err != nil || !ok || strings.TrimSpace(value) == "" {
		return catalogFilterState{}, false
	}
	var state catalogFilterState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		return catalogFilterState{}, false
	}
	state.Query = strings.TrimSpace(state.Query)
	state.Mode = normalizeCatalogMode(state.Mode)
	state.Features = normalizeCatalogFeatures(state.Features)
	return state, true
}

func savePersistedCatalogFilters(state catalogFilterState) {
	storage, err := interop.LocalStorage()
	if err != nil {
		return
	}
	state.Query = strings.TrimSpace(state.Query)
	state.Mode = normalizeCatalogMode(state.Mode)
	state.Features = normalizeCatalogFeatures(state.Features)
	encoded, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = storage.SetItem(catalogStateKey, string(encoded))
}

func openCatalogCacheStore(ctx context.Context) (interop.PersistentStore, error) {
	return interop.OpenPersistentStore(ctx, interop.PersistentStoreOptions{
		Name:         "examples-catalog-cache",
		DatabaseName: "gwc-examples-cache",
		Version:      1,
	})
}

func loadPersistedCatalogPayload(ctx context.Context) (cachedCatalogPayload, bool) {
	store, err := openCatalogCacheStore(ctx)
	if err != nil {
		return cachedCatalogPayload{}, false
	}
	defer store.Close()
	entry, ok, err := interop.LoadPersistentJSON[cachedCatalogPayload](ctx, store, catalogCacheKey)
	if err != nil || !ok || len(entry.Payload.Examples) == 0 {
		return cachedCatalogPayload{}, false
	}
	return entry, true
}

func savePersistedCatalogPayload(ctx context.Context, sourceURL string, payload catalogPayload) {
	store, err := openCatalogCacheStore(ctx)
	if err != nil {
		return
	}
	defer store.Close()
	_ = store.SetJSON(ctx, catalogCacheKey, cachedCatalogPayload{
		SourceURL: strings.TrimSpace(sourceURL),
		CachedAt:  time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	})
}

func loadExamplesRuntimeConfig() examplesRuntimeConfig {
	config := examplesRuntimeConfig{
		Mode:        "static",
		CatalogURL:  "catalog.json",
		AssetBase:   "",
		WasmBase:    "bin/",
		CatalogHref: "./index.html#/examples",
	}
	if bootstrap, err := ui.ReadBootstrapScript(""); err == nil {
		config.RoutePath = strings.TrimSpace(bootstrap.Route.Path)
		if raw, ok := bootstrap.Data["examples"]; ok {
			encoded, marshalErr := json.Marshal(raw)
			if marshalErr == nil {
				var decoded struct {
					Mode        string `json:"mode"`
					CatalogURL  string `json:"catalogURL"`
					AssetBase   string `json:"assetBase"`
					WasmBase    string `json:"wasmBase"`
					CatalogHref string `json:"catalogHref"`
					Slug        string `json:"slug"`
				}
				if json.Unmarshal(encoded, &decoded) == nil {
					if strings.TrimSpace(decoded.Mode) != "" {
						config.Mode = strings.TrimSpace(decoded.Mode)
					}
					if strings.TrimSpace(decoded.CatalogURL) != "" {
						config.CatalogURL = strings.TrimSpace(decoded.CatalogURL)
					}
					if strings.TrimSpace(decoded.AssetBase) != "" {
						config.AssetBase = strings.TrimSpace(decoded.AssetBase)
					}
					if strings.TrimSpace(decoded.WasmBase) != "" {
						config.WasmBase = strings.TrimSpace(decoded.WasmBase)
					}
					if strings.TrimSpace(decoded.CatalogHref) != "" {
						config.CatalogHref = strings.TrimSpace(decoded.CatalogHref)
					}
					config.ExampleSlug = strings.TrimSpace(decoded.Slug)
				}
			}
		}
	}
	if config.Mode == "static" {
		if location, err := interop.WindowLocation(); err == nil {
			if hashRoute := normalizeStaticExamplesRoute(location.Hash()); hashRoute != "" {
				config.RoutePath = hashRoute
			} else if strings.TrimSpace(config.RoutePath) == "" {
				config.RoutePath = strings.TrimSpace(location.Pathname())
			}
		}
		if strings.TrimSpace(config.RoutePath) == "" {
			config.RoutePath = "/examples/"
		}
		if strings.TrimSpace(config.ExampleSlug) == "" {
			config.ExampleSlug = staticExampleSlugFromRoute(config.RoutePath)
		}
	}
	return config
}

func normalizeStaticExamplesRoute(rawHash string) string {
	route := strings.TrimSpace(rawHash)
	route = strings.TrimPrefix(route, "#")
	route = strings.TrimSpace(route)
	if route == "" {
		return ""
	}
	if cutIndex := strings.Index(route, "?"); cutIndex >= 0 {
		route = route[:cutIndex]
	}
	route = strings.TrimSpace(route)
	if route == "" {
		return ""
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	route = "/" + strings.Trim(strings.TrimPrefix(route, "/"), "/")
	if route == "/" {
		return "/examples/"
	}
	if route == "/examples" {
		return "/examples/"
	}
	if !strings.HasPrefix(route, "/examples/") {
		return route
	}
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) < 2 {
		return "/examples/"
	}
	return "/examples/" + strings.TrimSpace(parts[1]) + "/"
}

func staticExampleSlugFromRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" || route == "/examples" || route == "/examples/" {
		return ""
	}
	if !strings.HasPrefix(route, "/examples/") {
		return ""
	}
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func catalogPayloadFresh(entry cachedCatalogPayload) bool {
	if catalogPayloadMaxAge <= 0 {
		return true
	}
	cachedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(entry.CachedAt))
	if err != nil {
		return false
	}
	return time.Since(cachedAt) <= catalogPayloadMaxAge
}

func prewarmCatalogAssets(config examplesRuntimeConfig, payload catalogPayload) {
	manager, err := pwa.OpenCacheStorageManager()
	if err != nil {
		return
	}
	assetBase := config.AssetBase
	immutableURLs := []string{
		assetBase + "script/wasm_exec.js",
		assetBase + "script/example-logger.js",
	}
	plan, err := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{
		CacheName:        catalogCacheName,
		ManifestRevision: strings.TrimSpace(payload.GeneratedAt),
		ImmutableURLs:    immutableURLs,
	}, pwa.CacheStoragePlanOptions{CachePrefix: catalogCachePrefix})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, _ = manager.Sync(ctx, plan)
}

func renderCatalogCards(entries []catalogEntry) []ui.Node {
	if len(entries) == 0 {
		return []ui.Node{
			html.Div(html.Props{Class: "rounded-[1.75rem] border border-dashed border-white/10 bg-slate-950/35 p-6 text-slate-400"},
				html.P(html.Props{Class: "text-lg font-semibold text-slate-100"}, html.Text("No examples matched this filter.")),
				html.P(html.Props{Class: "mt-2 text-sm leading-7"}, html.Text("Try clearing the search, changing browse mode, or removing framework feature filters.")),
			),
		}
	}
	nodes := make([]ui.Node, 0, len(entries))
	for _, entry := range entries {
		badges := []ui.Node{}
		if entry.UsesWasm {
			badges = append(badges, html.Span(html.Props{Class: "rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.25em] text-cyan-200"}, html.Text("WASM")))
		}
		if entry.MultiClient {
			badges = append(badges, html.Span(html.Props{Class: "rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.25em] text-emerald-200"}, html.Text("Multi-client")))
		}
		if entry.WasmBinary != "" {
			badges = append(badges, html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-300"}, html.Text(entry.WasmBinary)))
		}

		nodes = append(nodes,
			html.A(html.Props{Href: entry.Href, Class: "grid gap-4 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-6 text-left transition hover:-translate-y-1 hover:border-cyan-300/30 hover:bg-slate-950/60"},
				html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-3"},
					html.Div(html.Props{},
						html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text(entry.Name)),
						html.H2(html.Props{Class: "mt-3 text-2xl font-black tracking-tight text-white"}, html.Text(shortCatalogTitle(entry.Name))),
					),
					html.Div(html.Props{Class: "flex flex-wrap gap-2 justify-end"}, badges...),
				),
				html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("Entry point: %s", entry.HTMLFile))),
				html.P(html.Props{Class: "text-sm leading-7 text-slate-400"}, html.Text(fmt.Sprintf("Path: %s", entry.Href))),
			),
		)
	}
	return nodes
}

func shortCatalogTitle(name string) string {
	toWords := func(value string) string {
		parts := strings.Fields(strings.ReplaceAll(value, "-", " "))
		for index, part := range parts {
			if part == "" {
				continue
			}
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
		return strings.Join(parts, " ")
	}
	if parts := strings.SplitN(name, "-", 2); len(parts) == 2 {
		return toWords(parts[1])
	}
	return toWords(name)
}

func findCatalogEntry(entries []catalogEntry, slug string) (catalogEntry, bool) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return catalogEntry{}, false
	}
	for _, entry := range entries {
		if strings.EqualFold(strings.TrimSpace(entry.Name), slug) {
			return entry, true
		}
	}
	return catalogEntry{}, false
}

func loadCatalogFromNetwork(config examplesRuntimeConfig) (catalogPayload, string, error) {
	var lastErr error
	candidates := []string{config.CatalogURL}
	for _, fallback := range []string{"/examples/catalog.json", "catalog.json"} {
		if strings.TrimSpace(fallback) == "" || strings.EqualFold(strings.TrimSpace(config.CatalogURL), fallback) {
			continue
		}
		candidates = append(candidates, fallback)
	}
	for _, catalogURL := range candidates {
		response, err := http.Get(catalogURL)
		if err != nil {
			lastErr = err
			continue
		}
		if response.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("catalog request failed for %s with status %d", catalogURL, response.StatusCode)
			response.Body.Close()
			continue
		}
		var payload catalogPayload
		decodeErr := json.NewDecoder(response.Body).Decode(&payload)
		response.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}
		return payload, catalogURL, nil
	}
	if lastErr != nil {
		return catalogPayload{}, "", lastErr
	}
	return catalogPayload{}, "", fmt.Errorf("catalog request failed")
}

func catalogApp() ui.Node {
	runtimeConfig := loadExamplesRuntimeConfig()
	catalog := ui.UseState(catalogPayload{})
	loading := ui.UseState(true)
	loadStatus := ui.UseState("Loading catalog metadata from the examples server...")
	search := ui.UseState("")
	mode := ui.UseState("all")
	selectedFeatures := ui.UseState([]string{})
	filtersReady := ui.UseState(false)
	activity := ui.UseState(catalogSummary("", "all", nil))

	applyFilterState := func(nextQuery string, nextMode string, nextFeatures []string) {
		nextQuery = strings.TrimSpace(nextQuery)
		nextMode = normalizeCatalogMode(nextMode)
		nextFeatures = normalizeCatalogFeatures(nextFeatures)
		nextSummary := catalogSummary(nextQuery, nextMode, nextFeatures)
		search.Set(nextQuery)
		mode.Set(nextMode)
		selectedFeatures.Set(nextFeatures)
		activity.Set(nextSummary)
	}

	updateSearch := ui.UseEvent(func(event ui.InputEvent) {
		applyFilterState(event.GetValue(), mode.Get(), selectedFeatures.Get())
	})

	showAll := ui.UseEvent(func() {
		applyFilterState(search.Get(), "all", selectedFeatures.Get())
	})

	showWasm := ui.UseEvent(func() {
		applyFilterState(search.Get(), "wasm", selectedFeatures.Get())
	})

	showMultiClient := ui.UseEvent(func() {
		applyFilterState(search.Get(), "multi-client", selectedFeatures.Get())
	})

	toggleFeature := ui.UseEvent(func(event ui.ChangeEvent) {
		nextFeatures := toggleCatalogFeature(selectedFeatures.Get(), event.GetValue(), event.IsChecked())
		applyFilterState(search.Get(), mode.Get(), nextFeatures)
	})

	clearFeatures := ui.UseEvent(func() {
		applyFilterState(search.Get(), mode.Get(), nil)
	})

	ui.UseEffect(func() func() {
		if state, ok := loadPersistedCatalogFilters(); ok {
			applyFilterState(state.Query, state.Mode, state.Features)
		}
		filtersReady.Set(true)
		return nil
	}, "catalog-filters-load")

	ui.UseEffect(func() func() {
		if !filtersReady.Get() {
			return nil
		}
		savePersistedCatalogFilters(catalogFilterState{Query: search.Get(), Mode: mode.Get(), Features: selectedFeatures.Get()})
		return nil
	}, filtersReady.Get(), search.Get(), mode.Get(), featureStorageValue(selectedFeatures.Get()))

	ui.UseEffect(func() func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cached, hasCached := loadPersistedCatalogPayload(ctx)
			if hasCached {
				catalog.Set(cached.Payload)
				loading.Set(false)
				if catalogPayloadFresh(cached) {
					loadStatus.Set(fmt.Sprintf("Loaded %d examples from the durable catalog cache.", cached.Payload.TotalExamples))
					go prewarmCatalogAssets(runtimeConfig, cached.Payload)
					return
				}
				loadStatus.Set(fmt.Sprintf("Loaded %d cached examples and refreshing catalog metadata in the background.", cached.Payload.TotalExamples))
				go prewarmCatalogAssets(runtimeConfig, cached.Payload)
			}

			payload, sourceURL, err := loadCatalogFromNetwork(runtimeConfig)
			if err != nil {
				if hasCached {
					loadStatus.Set(fmt.Sprintf("Using cached catalog metadata because refresh failed: %v", err))
					return
				}
				loading.Set(false)
				loadStatus.Set(fmt.Sprintf("Catalog unavailable: %v", err))
				return
			}
			catalog.Set(payload)
			loading.Set(false)
			loadStatus.Set(fmt.Sprintf("Loaded %d examples from the examples catalog.", payload.TotalExamples))
			savePersistedCatalogPayload(context.Background(), sourceURL, payload)
			go prewarmCatalogAssets(runtimeConfig, payload)
		}()
		return nil
	}, "catalog-load")

	availableFeatures := availableCatalogFeatures(catalog.Get().Examples)
	filtered := filterCatalogEntries(catalog.Get().Examples, search.Get(), mode.Get(), selectedFeatures.Get())
	selectedEntry, hasSelectedEntry := findCatalogEntry(catalog.Get().Examples, runtimeConfig.ExampleSlug)
	displayedEntries := filtered
	if strings.TrimSpace(runtimeConfig.ExampleSlug) != "" {
		if hasSelectedEntry {
			displayedEntries = []catalogEntry{selectedEntry}
		} else {
			displayedEntries = nil
		}
	}
	cards := renderCatalogCards(displayedEntries)
	featureControls := make([]ui.Node, 0, len(availableFeatures))
	for _, feature := range availableFeatures {
		checked := false
		for _, selected := range selectedFeatures.Get() {
			if selected == feature {
				checked = true
				break
			}
		}
		featureControls = append(featureControls,
			html.Label(html.Props{Class: "flex items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm text-slate-200 hover:border-cyan-300/30"},
				html.Input(html.Props{Type: "checkbox", Value: feature, Checked: checked, OnChange: toggleFeature, Class: "h-4 w-4 rounded border-white/20 bg-slate-950/80 text-cyan-300"}),
				html.Span(html.Props{}, html.Text(catalogFeatureLabel(feature))),
			),
		)
	}
	activeModeClass := func(expected string) string {
		if mode.Get() == expected {
			return "rounded-full border border-cyan-400/30 bg-cyan-400/15 px-4 py-2 text-sm font-semibold text-cyan-100"
		}
		return "rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200"
	}

	return html.Main(html.Props{Class: "mx-auto max-w-7xl px-6 py-10 text-slate-100"},
		html.Section(html.Props{Class: "rounded-[2.25rem] border border-cyan-300/15 bg-[linear-gradient(135deg,rgba(15,23,42,0.96),rgba(15,23,42,0.72))] p-8 shadow-[0_24px_60px_rgba(2,6,23,0.45)] backdrop-blur-sm"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("GoWebComponents Examples")),
			html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white md:text-7xl"}, html.Text("Wasm-first catalog, served by Go.")),
			html.P(html.Props{Class: "mt-5 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("The primary examples surface boots as a Go/wasm client while the Go examples server still owns discovery, catalog metadata, and routing.")),
			html.Div(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"},
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Examples")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", catalog.Get().TotalExamples))),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Wasm bins")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", catalog.Get().WasmExamples))),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Multi-client demos")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", catalog.Get().MultiClientExamples))),
				),
			),
		),
		html.Section(html.Props{Class: "mt-8"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-white/5 p-6"},
				func() ui.Node {
					if strings.TrimSpace(runtimeConfig.ExampleSlug) == "" {
						return nil
					}
					if hasSelectedEntry {
						return html.Div(html.Props{Class: "mb-5 rounded-[1.5rem] border border-cyan-300/20 bg-cyan-400/10 p-5"},
							html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-cyan-200"}, html.Text("Selected example route")),
							html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(shortCatalogTitle(selectedEntry.Name))),
							html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("This static shell resolved a hash-route target and is keeping the catalog focused on that example while preserving a direct link to the original example entrypoint.")),
							html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
								html.A(html.Props{Href: selectedEntry.Href, Class: "rounded-full border border-cyan-300/30 bg-cyan-300/15 px-4 py-2 text-sm font-semibold text-cyan-100"}, html.Text("Open example")),
								html.A(html.Props{Href: runtimeConfig.CatalogHref, Class: "rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Back to catalog")),
							),
						)
					}
					return html.Div(html.Props{Class: "mb-5 rounded-[1.5rem] border border-amber-300/20 bg-amber-400/10 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-amber-200"}, html.Text("Selected example route")),
						html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Example not found in this catalog")),
						html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("The current static route points at an example slug that is not present in the loaded catalog manifest.")),
						html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
							html.A(html.Props{Href: runtimeConfig.CatalogHref, Class: "rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Back to catalog")),
						),
					)
				}(),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.Input(html.Props{Value: search.Get(), OnInput: updateSearch, Placeholder: "Search examples by slug, path, wasm bin, or tag", Class: "min-w-[18rem] flex-1 rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"}),
					html.Button(html.Props{Type: "button", OnClick: showAll, Class: activeModeClass("all")}, html.Text("All")),
					html.Button(html.Props{Type: "button", OnClick: showWasm, Class: activeModeClass("wasm")}, html.Text("Wasm-backed")),
					html.Button(html.Props{Type: "button", OnClick: showMultiClient, Class: activeModeClass("multi-client")}, html.Text("Multi-client")),
					html.Details(html.Props{Class: "min-w-[18rem] rounded-2xl border border-white/10 bg-slate-950/55"},
						html.Summary(html.Props{Class: "cursor-pointer list-none rounded-2xl px-4 py-3 text-sm font-semibold text-slate-100"}, html.Text("Framework features: "+featureSummaryLabel(selectedFeatures.Get()))),
						html.Div(html.Props{Class: "grid gap-3 border-t border-white/10 p-4"},
							html.Div(html.Props{Class: "flex items-center justify-between gap-3 text-xs uppercase tracking-[0.2em] text-slate-400"},
								html.Span(html.Props{}, html.Text("Match any selected feature")),
								html.Button(html.Props{Type: "button", OnClick: clearFeatures, Class: "rounded-full border border-white/10 px-3 py-1 text-[11px] font-semibold text-slate-300 hover:border-cyan-300/30 hover:text-cyan-100"}, html.Text("Clear")),
							),
							html.Div(html.Props{Class: "grid gap-2 sm:grid-cols-2"}, featureControls...),
						),
					),
				),
				html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-400"}, html.Text(activity.Get())),
				func() ui.Node {
					if loading.Get() {
						return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(loadStatus.Get()))
					}
					if strings.TrimSpace(runtimeConfig.ExampleSlug) != "" {
						if hasSelectedEntry {
							return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Resolved the current route to a single example target inside the static host shell."))
						}
						return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The current static route did not match any example slug in the loaded catalog."))
					}
					return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("Showing %d of %d examples.", len(filtered), catalog.Get().TotalExamples)))
				}(),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"}, cards...),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(catalogApp), "#app")
	select {}
}
