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

	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/pwa"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
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

func normalizeCatalogMode(parseMode string) string {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "wasm":
		return "wasm"
	case "multi-client":
		return "multi-client"
	default:
		return "all"
	}
}

func normalizeCatalogFeatures(parseFeatures []string) []string {
	parseSeen := map[string]struct{}{}
	for _, parseFeature := range parseFeatures {
		parseFeature = strings.TrimSpace(strings.ToLower(parseFeature))
		if parseFeature == "" {
			continue
		}
		parseSeen[parseFeature] = struct{}{}
	}
	parseOrdered := make([]string, 0, len(parseSeen))
	for _, parseFeature2 := range catalogFeatureOrder {
		if _, parseOk := parseSeen[parseFeature2]; parseOk {
			parseOrdered = append(parseOrdered, parseFeature2)
			delete(parseSeen, parseFeature2)
		}
	}
	if len(parseSeen) > 0 {
		parseExtra := make([]string, 0, len(parseSeen))
		for parseFeature3 := range parseSeen {
			parseExtra = append(parseExtra, parseFeature3)
		}
		sort.Strings(parseExtra)
		parseOrdered = append(parseOrdered, parseExtra...)
	}
	return parseOrdered
}

func catalogFeatureLabel(parseFeature string) string {
	parseFeature = strings.TrimSpace(strings.ToLower(parseFeature))
	if parseLabel, parseOk := catalogFeatureLabels[parseFeature]; parseOk {
		return parseLabel
	}
	parseParts := strings.Fields(strings.ReplaceAll(parseFeature, "-", " "))
	for parseIndex, parsePart := range parseParts {
		if parsePart == "" {
			continue
		}
		parseParts[parseIndex] = strings.ToUpper(parsePart[:1]) + parsePart[1:]
	}
	return strings.Join(parseParts, " ")
}

func availableCatalogFeatures(parseEntries []catalogEntry) []string {
	parsePresent := map[string]struct{}{}
	for _, parseEntry := range parseEntries {
		for _, parseTag := range parseEntry.Tags {
			parseTag = strings.TrimSpace(strings.ToLower(parseTag))
			if _, parseOk := catalogFeatureLabels[parseTag]; parseOk {
				parsePresent[parseTag] = struct{}{}
			}
		}
	}
	parseAvailable := make([]string, 0, len(parsePresent))
	for _, parseFeature := range catalogFeatureOrder {
		if _, parseOk2 := parsePresent[parseFeature]; parseOk2 {
			parseAvailable = append(parseAvailable, parseFeature)
		}
	}
	return parseAvailable
}

func toggleCatalogFeature(parseFeatures []string, parseFeature string, isEnabled bool) []string {
	parseFeature = strings.TrimSpace(strings.ToLower(parseFeature))
	parseNext := make([]string, 0, len(parseFeatures)+1)
	for _, parseExisting := range normalizeCatalogFeatures(parseFeatures) {
		if parseExisting == parseFeature {
			continue
		}
		parseNext = append(parseNext, parseExisting)
	}
	if isEnabled && parseFeature != "" {
		parseNext = append(parseNext, parseFeature)
	}
	return normalizeCatalogFeatures(parseNext)
}

func featureSummaryLabel(parseFeatures []string) string {
	parseFeatures = normalizeCatalogFeatures(parseFeatures)
	if len(parseFeatures) == 0 {
		return "All framework features"
	}
	if len(parseFeatures) == 1 {
		return catalogFeatureLabel(parseFeatures[0])
	}
	if len(parseFeatures) == 2 {
		return catalogFeatureLabel(parseFeatures[0]) + " and " + catalogFeatureLabel(parseFeatures[1])
	}
	return fmt.Sprintf("%s, %s, and %d more", catalogFeatureLabel(parseFeatures[0]), catalogFeatureLabel(parseFeatures[1]), len(parseFeatures)-2)
}

func featureStorageValue(parseFeatures []string) string {
	return strings.Join(normalizeCatalogFeatures(parseFeatures), ",")
}

func filterCatalogEntries(parseEntries []catalogEntry, parseQuery string, parseMode string, parseSelectedFeatures []string) []catalogEntry {
	parseTrimmedQuery := strings.TrimSpace(strings.ToLower(parseQuery))
	parseSelectedFeatures = normalizeCatalogFeatures(parseSelectedFeatures)
	parseFiltered := make([]catalogEntry, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		if parseMode == "wasm" && !parseEntry.UsesWasm {
			continue
		}
		if parseMode == "multi-client" && !parseEntry.MultiClient {
			continue
		}
		if len(parseSelectedFeatures) > 0 {
			isParseMatchesFeature := false
			for _, parseFeature := range parseSelectedFeatures {
				for _, parseTag := range parseEntry.Tags {
					if strings.EqualFold(strings.TrimSpace(parseTag), parseFeature) {
						isParseMatchesFeature = true
						break
					}
				}
				if isParseMatchesFeature {
					break
				}
			}
			if !isParseMatchesFeature {
				continue
			}
		}
		if parseTrimmedQuery != "" {
			parseHaystack := strings.ToLower(parseEntry.Name + " " + parseEntry.Href + " " + parseEntry.WasmBinary + " " + strings.Join(parseEntry.Tags, " "))
			isParseMatches := true
			for _, parseTerm := range strings.Fields(parseTrimmedQuery) {
				if !strings.Contains(parseHaystack, parseTerm) {
					isParseMatches = false
					break
				}
			}
			if !isParseMatches {
				continue
			}
		}
		parseFiltered = append(parseFiltered, parseEntry)
	}
	return parseFiltered
}

func catalogSummary(parseQuery string, parseMode string, parseSelectedFeatures []string) string {
	parseParts := []string{"Browsing the wasm-first examples catalog"}
	if parseMode == "wasm" {
		parseParts = append(parseParts, "filtering to wasm-backed examples")
	}
	if parseMode == "multi-client" {
		parseParts = append(parseParts, "filtering to multi-client coordination demos")
	}
	parseSelectedFeatures = normalizeCatalogFeatures(parseSelectedFeatures)
	if len(parseSelectedFeatures) > 0 {
		parseLabels := make([]string, 0, len(parseSelectedFeatures))
		for _, parseFeature := range parseSelectedFeatures {
			parseLabels = append(parseLabels, catalogFeatureLabel(parseFeature))
		}
		parseParts = append(parseParts, fmt.Sprintf("matching framework features %s", strings.Join(parseLabels, ", ")))
	}
	if strings.TrimSpace(parseQuery) != "" {
		parseParts = append(parseParts, fmt.Sprintf("searching for %q", strings.TrimSpace(parseQuery)))
	}
	return strings.Join(parseParts, " and ") + "."
}

func loadPersistedCatalogFilters() (catalogFilterState, bool) {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return catalogFilterState{}, false
	}
	parseValue, parseOk, parseErr := parseStorage.GetItem(catalogStateKey)
	if parseErr != nil || !parseOk || strings.TrimSpace(parseValue) == "" {
		return catalogFilterState{}, false
	}
	var parseState catalogFilterState
	if parseErr2 := json.Unmarshal([]byte(parseValue), &parseState); parseErr2 != nil {
		return catalogFilterState{}, false
	}
	parseState.Query = strings.TrimSpace(parseState.Query)
	parseState.Mode = normalizeCatalogMode(parseState.Mode)
	parseState.Features = normalizeCatalogFeatures(parseState.Features)
	return parseState, true
}

func savePersistedCatalogFilters(parseState catalogFilterState) {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return
	}
	parseState.Query = strings.TrimSpace(parseState.Query)
	parseState.Mode = normalizeCatalogMode(parseState.Mode)
	parseState.Features = normalizeCatalogFeatures(parseState.Features)
	parseEncoded, parseErr := json.Marshal(parseState)
	if parseErr != nil {
		return
	}
	_ = parseStorage.SetItem(catalogStateKey, string(parseEncoded))
}

func openCatalogCacheStore(parseCtx context.Context) (interop.PersistentStore, error) {
	return interop.OpenPersistentStore(parseCtx, interop.PersistentStoreOptions{
		Name:         "examples-catalog-cache",
		DatabaseName: "gwc-examples-cache",
		Version:      1,
	})
}

func loadPersistedCatalogPayload(parseCtx context.Context) (cachedCatalogPayload, bool) {
	store, parseErr := openCatalogCacheStore(parseCtx)
	if parseErr != nil {
		return cachedCatalogPayload{}, false
	}
	defer store.Close()
	parseEntry, parseOk, parseErr := interop.LoadPersistentJSON[cachedCatalogPayload](parseCtx, store, catalogCacheKey)
	if parseErr != nil || !parseOk || len(parseEntry.Payload.Examples) == 0 {
		return cachedCatalogPayload{}, false
	}
	return parseEntry, true
}

func savePersistedCatalogPayload(parseCtx context.Context, parseSourceURL string, parsePayload catalogPayload) {
	store, parseErr := openCatalogCacheStore(parseCtx)
	if parseErr != nil {
		return
	}
	defer store.Close()
	_ = store.SetJSON(parseCtx, catalogCacheKey, cachedCatalogPayload{
		SourceURL: strings.TrimSpace(parseSourceURL),
		CachedAt:  time.Now().UTC().Format(time.RFC3339),
		Payload:   parsePayload,
	})
}

func loadExamplesRuntimeConfig() examplesRuntimeConfig {
	parseConfig := examplesRuntimeConfig{
		Mode:        "static",
		CatalogURL:  "catalog.json",
		AssetBase:   "",
		WasmBase:    "bin/",
		CatalogHref: "./index.html#/examples",
	}
	if parseBootstrap, parseErr := ui.ReadBootstrapScript(""); parseErr == nil {
		parseConfig.RoutePath = strings.TrimSpace(parseBootstrap.Route.Path)
		if parseRaw, parseOk := parseBootstrap.Data["examples"]; parseOk {
			parseEncoded, parseMarshalErr := json.Marshal(parseRaw)
			if parseMarshalErr == nil {
				var parseDecoded struct {
					Mode        string `json:"mode"`
					CatalogURL  string `json:"catalogURL"`
					AssetBase   string `json:"assetBase"`
					WasmBase    string `json:"wasmBase"`
					CatalogHref string `json:"catalogHref"`
					Slug        string `json:"slug"`
				}
				if json.Unmarshal(parseEncoded, &parseDecoded) == nil {
					if strings.TrimSpace(parseDecoded.Mode) != "" {
						parseConfig.Mode = strings.TrimSpace(parseDecoded.Mode)
					}
					if strings.TrimSpace(parseDecoded.CatalogURL) != "" {
						parseConfig.CatalogURL = strings.TrimSpace(parseDecoded.CatalogURL)
					}
					if strings.TrimSpace(parseDecoded.AssetBase) != "" {
						parseConfig.AssetBase = strings.TrimSpace(parseDecoded.AssetBase)
					}
					if strings.TrimSpace(parseDecoded.WasmBase) != "" {
						parseConfig.WasmBase = strings.TrimSpace(parseDecoded.WasmBase)
					}
					if strings.TrimSpace(parseDecoded.CatalogHref) != "" {
						parseConfig.CatalogHref = strings.TrimSpace(parseDecoded.CatalogHref)
					}
					parseConfig.ExampleSlug = strings.TrimSpace(parseDecoded.Slug)
				}
			}
		}
	}
	if parseConfig.Mode == "static" {
		if parseLocation, parseErr2 := interop.GetWindowLocation(); parseErr2 == nil {
			if parseHashRoute := normalizeStaticExamplesRoute(parseLocation.Hash()); parseHashRoute != "" {
				parseConfig.RoutePath = parseHashRoute
			} else if strings.TrimSpace(parseConfig.RoutePath) == "" {
				parseConfig.RoutePath = strings.TrimSpace(parseLocation.Pathname())
			}
		}
		if strings.TrimSpace(parseConfig.RoutePath) == "" {
			parseConfig.RoutePath = "/examples/"
		}
		if strings.TrimSpace(parseConfig.ExampleSlug) == "" {
			parseConfig.ExampleSlug = staticExampleSlugFromRoute(parseConfig.RoutePath)
		}
	}
	return parseConfig
}

func normalizeStaticExamplesRoute(parseRawHash string) string {
	parseRoute := strings.TrimSpace(parseRawHash)
	parseRoute = strings.TrimPrefix(parseRoute, "#")
	parseRoute = strings.TrimSpace(parseRoute)
	if parseRoute == "" {
		return ""
	}
	if parseCutIndex := strings.Index(parseRoute, "?"); parseCutIndex >= 0 {
		parseRoute = parseRoute[:parseCutIndex]
	}
	parseRoute = strings.TrimSpace(parseRoute)
	if parseRoute == "" {
		return ""
	}
	if !strings.HasPrefix(parseRoute, "/") {
		parseRoute = "/" + parseRoute
	}
	parseRoute = "/" + strings.Trim(strings.TrimPrefix(parseRoute, "/"), "/")
	if parseRoute == "/" {
		return "/examples/"
	}
	if parseRoute == "/examples" {
		return "/examples/"
	}
	if !strings.HasPrefix(parseRoute, "/examples/") {
		return parseRoute
	}
	parseParts := strings.Split(strings.Trim(parseRoute, "/"), "/")
	if len(parseParts) < 2 {
		return "/examples/"
	}
	return "/examples/" + strings.TrimSpace(parseParts[1]) + "/"
}

func staticExampleSlugFromRoute(parseRoute string) string {
	parseRoute = strings.TrimSpace(parseRoute)
	if parseRoute == "" || parseRoute == "/examples" || parseRoute == "/examples/" {
		return ""
	}
	if !strings.HasPrefix(parseRoute, "/examples/") {
		return ""
	}
	parseParts := strings.Split(strings.Trim(parseRoute, "/"), "/")
	if len(parseParts) < 2 {
		return ""
	}
	return strings.TrimSpace(parseParts[1])
}

func catalogPayloadFresh(parseEntry cachedCatalogPayload) bool {
	if catalogPayloadMaxAge <= 0 {
		return true
	}
	parseCachedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseEntry.CachedAt))
	if parseErr != nil {
		return false
	}
	return time.Since(parseCachedAt) <= catalogPayloadMaxAge
}

func prewarmCatalogAssets(parseConfig examplesRuntimeConfig, parsePayload catalogPayload) {
	parseManager, parseErr := pwa.OpenCacheStorageManager()
	if parseErr != nil {
		return
	}
	parseAssetBase := parseConfig.AssetBase
	parseImmutableURLs := []string{
		parseAssetBase + "script/wasm_exec.js",
		parseAssetBase + "script/example-logger.js",
	}
	parsePlan, parseErr := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{
		CacheName:        catalogCacheName,
		ManifestRevision: strings.TrimSpace(parsePayload.GeneratedAt),
		ImmutableURLs:    parseImmutableURLs,
	}, pwa.CacheStoragePlanOptions{CachePrefix: catalogCachePrefix})
	if parseErr != nil {
		return
	}
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer parseCancel()
	_, _ = parseManager.Sync(parseCtx, parsePlan)
}

func renderCatalogCards(parseEntries []catalogEntry) []ui.Node {
	if len(parseEntries) == 0 {
		return []ui.Node{
			html.Div(html.Props{Class: "rounded-[1.75rem] border border-dashed border-white/10 bg-slate-950/35 p-6 text-slate-400"},
				html.P(html.Props{Class: "text-lg font-semibold text-slate-100"}, html.Text("No examples matched this filter.")),
				html.P(html.Props{Class: "mt-2 text-sm leading-7"}, html.Text("Try clearing the search, changing browse mode, or removing framework feature filters.")),
			),
		}
	}
	parseNodes := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseBadges := []ui.Node{}
		if parseEntry.UsesWasm {
			parseBadges = append(parseBadges, html.Span(html.Props{Class: "rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.25em] text-cyan-200"}, html.Text("WASM")))
		}
		if parseEntry.MultiClient {
			parseBadges = append(parseBadges, html.Span(html.Props{Class: "rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.25em] text-emerald-200"}, html.Text("Multi-client")))
		}
		if parseEntry.WasmBinary != "" {
			parseBadges = append(parseBadges, html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-300"}, html.Text(parseEntry.WasmBinary)))
		}

		parseNodes = append(parseNodes,
			html.A(html.Props{Href: parseEntry.Href, Class: "grid gap-4 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-6 text-left transition hover:-translate-y-1 hover:border-cyan-300/30 hover:bg-slate-950/60"},
				html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-3"},
					html.Div(html.Props{},
						html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text(parseEntry.Name)),
						html.H2(html.Props{Class: "mt-3 text-2xl font-black tracking-tight text-white"}, html.Text(shortCatalogTitle(parseEntry.Name))),
					),
					html.Div(html.Props{Class: "flex flex-wrap gap-2 justify-end"}, parseBadges...),
				),
				html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("Entry point: %s", parseEntry.HTMLFile))),
				html.P(html.Props{Class: "text-sm leading-7 text-slate-400"}, html.Text(fmt.Sprintf("Path: %s", parseEntry.Href))),
			),
		)
	}
	return parseNodes
}

func shortCatalogTitle(parseName string) string {
	parseToWords := func(parseValue string) string {
		parseParts := strings.Fields(strings.ReplaceAll(parseValue, "-", " "))
		for parseIndex, parsePart := range parseParts {
			if parsePart == "" {
				continue
			}
			parseParts[parseIndex] = strings.ToUpper(parsePart[:1]) + parsePart[1:]
		}
		return strings.Join(parseParts, " ")
	}
	if parseParts2 := strings.SplitN(parseName, "-", 2); len(parseParts2) == 2 {
		return parseToWords(parseParts2[1])
	}
	return parseToWords(parseName)
}

func findCatalogEntry(parseEntries []catalogEntry, parseSlug string) (catalogEntry, bool) {
	parseSlug = strings.TrimSpace(strings.ToLower(parseSlug))
	if parseSlug == "" {
		return catalogEntry{}, false
	}
	for _, parseEntry := range parseEntries {
		if strings.EqualFold(strings.TrimSpace(parseEntry.Name), parseSlug) {
			return parseEntry, true
		}
	}
	return catalogEntry{}, false
}

func loadCatalogFromNetwork(parseConfig examplesRuntimeConfig) (catalogPayload, string, error) {
	var parseLastErr error
	parseCandidates := []string{parseConfig.CatalogURL}
	for _, parseFallback := range []string{"/examples/catalog.json", "catalog.json"} {
		if strings.TrimSpace(parseFallback) == "" || strings.EqualFold(strings.TrimSpace(parseConfig.CatalogURL), parseFallback) {
			continue
		}
		parseCandidates = append(parseCandidates, parseFallback)
	}
	for _, parseCatalogURL := range parseCandidates {
		parseResponse, parseErr := http.Get(parseCatalogURL)
		if parseErr != nil {
			parseLastErr = parseErr
			continue
		}
		if parseResponse.StatusCode != http.StatusOK {
			parseLastErr = fmt.Errorf("catalog request failed for %s with status %d", parseCatalogURL, parseResponse.StatusCode)
			parseResponse.Body.Close()
			continue
		}
		var parsePayload catalogPayload
		parseDecodeErr := json.NewDecoder(parseResponse.Body).Decode(&parsePayload)
		parseResponse.Body.Close()
		if parseDecodeErr != nil {
			parseLastErr = parseDecodeErr
			continue
		}
		return parsePayload, parseCatalogURL, nil
	}
	if parseLastErr != nil {
		return catalogPayload{}, "", parseLastErr
	}
	return catalogPayload{}, "", fmt.Errorf("catalog request failed")
}

func catalogApp() ui.Node {
	parseRuntimeConfig := loadExamplesRuntimeConfig()
	parseCatalog := ui.UseState(catalogPayload{})
	parseLoading := ui.UseState(true)
	parseLoadStatus := ui.UseState("Loading catalog metadata from the examples server...")
	parseSearch := ui.UseState("")
	parseMode := ui.UseState("all")
	parseSelectedFeatures := ui.UseState([]string{})
	parseFiltersReady := ui.UseState(false)
	parseActivity := ui.UseState(catalogSummary("", "all", nil))

	applyFilterState := func(parseNextQuery string, parseNextMode string, parseNextFeatures2 []string) {
		parseNextQuery = strings.TrimSpace(parseNextQuery)
		parseNextMode = normalizeCatalogMode(parseNextMode)
		parseNextFeatures2 = normalizeCatalogFeatures(parseNextFeatures2)
		parseNextSummary := catalogSummary(parseNextQuery, parseNextMode, parseNextFeatures2)
		parseSearch.Set(parseNextQuery)
		parseMode.Set(parseNextMode)
		parseSelectedFeatures.Set(parseNextFeatures2)
		parseActivity.Set(parseNextSummary)
	}

	parseUpdateSearch := ui.UseEvent(func(parseEvent ui.InputEvent) {
		applyFilterState(parseEvent.GetValue(), parseMode.Get(), parseSelectedFeatures.Get())
	})

	parseShowAll := ui.UseEvent(func() {
		applyFilterState(parseSearch.Get(), "all", parseSelectedFeatures.Get())
	})

	parseShowWasm := ui.UseEvent(func() {
		applyFilterState(parseSearch.Get(), "wasm", parseSelectedFeatures.Get())
	})

	parseShowMultiClient := ui.UseEvent(func() {
		applyFilterState(parseSearch.Get(), "multi-client", parseSelectedFeatures.Get())
	})

	parseToggleFeature := ui.UseEvent(func(parseEvent2 ui.ChangeEvent) {
		parseNextFeatures := toggleCatalogFeature(parseSelectedFeatures.Get(), parseEvent2.GetValue(), parseEvent2.IsChecked())
		applyFilterState(parseSearch.Get(), parseMode.Get(), parseNextFeatures)
	})

	clearFeatures := ui.UseEvent(func() {
		applyFilterState(parseSearch.Get(), parseMode.Get(), nil)
	})

	ui.UseEffect(func() func() {
		if parseState, parseOk := loadPersistedCatalogFilters(); parseOk {
			applyFilterState(parseState.Query, parseState.Mode, parseState.Features)
		}
		parseFiltersReady.Set(true)
		return nil
	}, "catalog-filters-load")

	ui.UseEffect(func() func() {
		if !parseFiltersReady.Get() {
			return nil
		}
		savePersistedCatalogFilters(catalogFilterState{Query: parseSearch.Get(), Mode: parseMode.Get(), Features: parseSelectedFeatures.Get()})
		return nil
	}, parseFiltersReady.Get(), parseSearch.Get(), parseMode.Get(), featureStorageValue(parseSelectedFeatures.Get()))

	ui.UseEffect(func() func() {
		go func() {
			parseCtx, parseCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer parseCancel()

			parseCached, hasCached := loadPersistedCatalogPayload(parseCtx)
			if hasCached {
				parseCatalog.Set(parseCached.Payload)
				parseLoading.Set(false)
				if catalogPayloadFresh(parseCached) {
					parseLoadStatus.Set(fmt.Sprintf("Loaded %d examples from the durable catalog cache.", parseCached.Payload.TotalExamples))
					go prewarmCatalogAssets(parseRuntimeConfig, parseCached.Payload)
					return
				}
				parseLoadStatus.Set(fmt.Sprintf("Loaded %d cached examples and refreshing catalog metadata in the background.", parseCached.Payload.TotalExamples))
				go prewarmCatalogAssets(parseRuntimeConfig, parseCached.Payload)
			}

			parsePayload, parseSourceURL, parseErr := loadCatalogFromNetwork(parseRuntimeConfig)
			if parseErr != nil {
				if hasCached {
					parseLoadStatus.Set(fmt.Sprintf("Using cached catalog metadata because refresh failed: %v", parseErr))
					return
				}
				parseLoading.Set(false)
				parseLoadStatus.Set(fmt.Sprintf("Catalog unavailable: %v", parseErr))
				return
			}
			parseCatalog.Set(parsePayload)
			parseLoading.Set(false)
			parseLoadStatus.Set(fmt.Sprintf("Loaded %d examples from the examples catalog.", parsePayload.TotalExamples))
			savePersistedCatalogPayload(context.Background(), parseSourceURL, parsePayload)
			go prewarmCatalogAssets(parseRuntimeConfig, parsePayload)
		}()
		return nil
	}, "catalog-load")

	parseAvailableFeatures := availableCatalogFeatures(parseCatalog.Get().Examples)
	parseFiltered := filterCatalogEntries(parseCatalog.Get().Examples, parseSearch.Get(), parseMode.Get(), parseSelectedFeatures.Get())
	parseSelectedEntry, hasSelectedEntry := findCatalogEntry(parseCatalog.Get().Examples, parseRuntimeConfig.ExampleSlug)
	parseDisplayedEntries := parseFiltered
	if strings.TrimSpace(parseRuntimeConfig.ExampleSlug) != "" {
		if hasSelectedEntry {
			parseDisplayedEntries = []catalogEntry{parseSelectedEntry}
		} else {
			parseDisplayedEntries = nil
		}
	}
	parseCards := renderCatalogCards(parseDisplayedEntries)
	parseFeatureControls := make([]ui.Node, 0, len(parseAvailableFeatures))
	for _, parseFeature := range parseAvailableFeatures {
		isParseChecked := false
		for _, parseSelected := range parseSelectedFeatures.Get() {
			if parseSelected == parseFeature {
				isParseChecked = true
				break
			}
		}
		parseFeatureControls = append(parseFeatureControls,
			html.Label(html.Props{Class: "flex items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3 text-sm text-slate-200 hover:border-cyan-300/30"},
				html.Input(html.Props{Type: "checkbox", Value: parseFeature, Checked: isParseChecked, OnChange: parseToggleFeature, Class: "h-4 w-4 rounded border-white/20 bg-slate-950/80 text-cyan-300"}),
				html.Span(html.Props{}, html.Text(catalogFeatureLabel(parseFeature))),
			),
		)
	}
	parseActiveModeClass := func(parseExpected string) string {
		if parseMode.Get() == parseExpected {
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
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", parseCatalog.Get().TotalExamples))),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Wasm bins")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", parseCatalog.Get().WasmExamples))),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Multi-client demos")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", parseCatalog.Get().MultiClientExamples))),
				),
			),
		),
		html.Section(html.Props{Class: "mt-8"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-white/5 p-6"},
				func() ui.Node {
					if strings.TrimSpace(parseRuntimeConfig.ExampleSlug) == "" {
						return nil
					}
					if hasSelectedEntry {
						return html.Div(html.Props{Class: "mb-5 rounded-[1.5rem] border border-cyan-300/20 bg-cyan-400/10 p-5"},
							html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-cyan-200"}, html.Text("Selected example route")),
							html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(shortCatalogTitle(parseSelectedEntry.Name))),
							html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("This static shell resolved a hash-route target and is keeping the catalog focused on that example while preserving a direct link to the original example entrypoint.")),
							html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
								html.A(html.Props{Href: parseSelectedEntry.Href, Class: "rounded-full border border-cyan-300/30 bg-cyan-300/15 px-4 py-2 text-sm font-semibold text-cyan-100"}, html.Text("Open example")),
								html.A(html.Props{Href: parseRuntimeConfig.CatalogHref, Class: "rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Back to catalog")),
							),
						)
					}
					return html.Div(html.Props{Class: "mb-5 rounded-[1.5rem] border border-amber-300/20 bg-amber-400/10 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-amber-200"}, html.Text("Selected example route")),
						html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Example not found in this catalog")),
						html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("The current static route points at an example slug that is not present in the loaded catalog manifest.")),
						html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
							html.A(html.Props{Href: parseRuntimeConfig.CatalogHref, Class: "rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Back to catalog")),
						),
					)
				}(),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.Input(html.Props{Value: parseSearch.Get(), OnInput: parseUpdateSearch, Placeholder: "Search examples by slug, path, wasm bin, or tag", Class: "min-w-[18rem] flex-1 rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"}),
					html.Button(html.Props{Type: "button", OnClick: parseShowAll, Class: parseActiveModeClass("all")}, html.Text("All")),
					html.Button(html.Props{Type: "button", OnClick: parseShowWasm, Class: parseActiveModeClass("wasm")}, html.Text("Wasm-backed")),
					html.Button(html.Props{Type: "button", OnClick: parseShowMultiClient, Class: parseActiveModeClass("multi-client")}, html.Text("Multi-client")),
					html.Details(html.Props{Class: "min-w-[18rem] rounded-2xl border border-white/10 bg-slate-950/55"},
						html.Summary(html.Props{Class: "cursor-pointer list-none rounded-2xl px-4 py-3 text-sm font-semibold text-slate-100"}, html.Text("Framework features: "+featureSummaryLabel(parseSelectedFeatures.Get()))),
						html.Div(html.Props{Class: "grid gap-3 border-t border-white/10 p-4"},
							html.Div(html.Props{Class: "flex items-center justify-between gap-3 text-xs uppercase tracking-[0.2em] text-slate-400"},
								html.Span(html.Props{}, html.Text("Match any selected feature")),
								html.Button(html.Props{Type: "button", OnClick: clearFeatures, Class: "rounded-full border border-white/10 px-3 py-1 text-[11px] font-semibold text-slate-300 hover:border-cyan-300/30 hover:text-cyan-100"}, html.Text("Clear")),
							),
							html.Div(html.Props{Class: "grid gap-2 sm:grid-cols-2"}, parseFeatureControls...),
						),
					),
				),
				html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-400"}, html.Text(parseActivity.Get())),
				func() ui.Node {
					if parseLoading.Get() {
						return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseLoadStatus.Get()))
					}
					if strings.TrimSpace(parseRuntimeConfig.ExampleSlug) != "" {
						if hasSelectedEntry {
							return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Resolved the current route to a single example target inside the static host shell."))
						}
						return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The current static route did not match any example slug in the loaded catalog."))
					}
					return html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("Showing %d of %d examples.", len(parseFiltered), parseCatalog.Get().TotalExamples)))
				}(),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"}, parseCards...),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(catalogApp), "#app")
	select {}
}
