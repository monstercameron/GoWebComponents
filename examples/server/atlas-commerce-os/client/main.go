//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/v4/fetch"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

const (
	atlasNoticeQueryKey             = "atlas_notice"
	atlasBootstrapModeQueryKey      = "atlas_bootstrap"
	atlasDebugLogsAttr              = "data-atlas-debug-logs"
	atlasManagedAttr                = "data-gwc-router-managed"
	atlasDirtyGuardAttr             = "data-atlas-dirty-guard"
	atlasDirtyMessageAttr           = "data-atlas-dirty-message"
	atlasMockSignInPath             = "/auth/mock-sign-in"
	atlasBootstrapScriptID          = "__ATLAS_BOOTSTRAP__"
	atlasBootstrapReferenceScriptID = "__ATLAS_BOOTSTRAP_REF__"
)

var initialBootstrap ui.SSRBootstrap
var initialPayload atlas.Payload
var anchorNavigationHandler js.Func
var beforeUnloadHandler js.Func
var dirtyGuardHandlers []js.Func
var trackOverlayDebugState map[string]bool
var watchOverlayDebugObserver js.Value
var handleOverlayDebugMutation js.Func
var handleOverlayDebugFocus js.Func
var cacheOverlayFocusMismatch string
var cacheRevalidationNotice string
var cacheDebugEventAt = map[string]time.Time{}
var lastRenderedRouteKey string
var hasRenderedRoute bool

var atlasCacheOptions = fetch.CacheOptions{
	StaleAfter:   5 * time.Minute,
	MaxAge:       30 * time.Minute,
	DisposeAfter: 90 * time.Minute,
}

// isAtlasDebugLoggingEnabled checks runtime debug toggles before emitting browser diagnostics.
func isAtlasDebugLoggingEnabled() bool {
	parseDocument := js.Global().Get("document")
	if parseDocument.Truthy() {
		parseRoot := parseDocument.Get("documentElement")
		if parseRoot.Truthy() {
			parseAttr := strings.TrimSpace(parseRoot.Call("getAttribute", atlasDebugLogsAttr).String())
			if parseAttr == "1" || strings.EqualFold(parseAttr, "true") || strings.EqualFold(parseAttr, "yes") || strings.EqualFold(parseAttr, "on") {
				return true
			}
		}
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() {
		parseFlag := parseWindow.Get("__atlasDebugLogs")
		if parseFlag.Truthy() {
			switch parseFlag.Type() {
			case js.TypeBoolean:
				return parseFlag.Bool()
			case js.TypeString:
				parseRaw := strings.TrimSpace(parseFlag.String())
				return parseRaw == "1" || strings.EqualFold(parseRaw, "true") || strings.EqualFold(parseRaw, "yes") || strings.EqualFold(parseRaw, "on")
			}
		}
	}
	return false
}

// sanitizeDebugDetailValue redacts sensitive fragments and truncates deep structures for safer debug output.
func sanitizeDebugDetailValue(parseValue any, parseDepth int) any {
	if parseDepth > 4 {
		return "[truncated]"
	}
	switch parseTyped := parseValue.(type) {
	case string:
		parseTrimmed := strings.TrimSpace(parseTyped)
		if len(parseTrimmed) > 160 {
			return parseTrimmed[:160]
		}
		return parseTrimmed
	case map[string]any:
		parseSanitized := map[string]any{}
		for parseKey, parseItem := range parseTyped {
			parseLowerKey := strings.ToLower(strings.TrimSpace(parseKey))
			if strings.Contains(parseLowerKey, "token") || strings.Contains(parseLowerKey, "csrf") || strings.Contains(parseLowerKey, "cookie") || strings.Contains(parseLowerKey, "password") || strings.Contains(parseLowerKey, "secret") || strings.Contains(parseLowerKey, "email") {
				parseSanitized[parseKey] = "[redacted]"
				continue
			}
			parseSanitized[parseKey] = sanitizeDebugDetailValue(parseItem, parseDepth+1)
		}
		return parseSanitized
	case map[string]string:
		parseSanitized := map[string]any{}
		for parseKey, parseItem := range parseTyped {
			parseLowerKey := strings.ToLower(strings.TrimSpace(parseKey))
			if strings.Contains(parseLowerKey, "token") || strings.Contains(parseLowerKey, "csrf") || strings.Contains(parseLowerKey, "cookie") || strings.Contains(parseLowerKey, "password") || strings.Contains(parseLowerKey, "secret") || strings.Contains(parseLowerKey, "email") {
				parseSanitized[parseKey] = "[redacted]"
				continue
			}
			parseSanitized[parseKey] = sanitizeDebugDetailValue(parseItem, parseDepth+1)
		}
		return parseSanitized
	case []any:
		parseLimit := len(parseTyped)
		if parseLimit > 12 {
			parseLimit = 12
		}
		parseItems := make([]any, 0, parseLimit)
		for parseIndex := 0; parseIndex < parseLimit; parseIndex++ {
			parseItems = append(parseItems, sanitizeDebugDetailValue(parseTyped[parseIndex], parseDepth+1))
		}
		return parseItems
	case []string:
		parseLimit := len(parseTyped)
		if parseLimit > 12 {
			parseLimit = 12
		}
		parseItems := make([]any, 0, parseLimit)
		for parseIndex := 0; parseIndex < parseLimit; parseIndex++ {
			parseItems = append(parseItems, sanitizeDebugDetailValue(parseTyped[parseIndex], parseDepth+1))
		}
		return parseItems
	default:
		return parseValue
	}
}

// sanitizeDebugDetails applies key-based redaction and shape trimming before debug logs reach the console.
func sanitizeDebugDetails(parseDetails map[string]any) map[string]any {
	parseSanitized := map[string]any{}
	for parseKey, parseValue := range parseDetails {
		parseLowerKey := strings.ToLower(strings.TrimSpace(parseKey))
		if strings.Contains(parseLowerKey, "token") || strings.Contains(parseLowerKey, "csrf") || strings.Contains(parseLowerKey, "cookie") || strings.Contains(parseLowerKey, "password") || strings.Contains(parseLowerKey, "secret") || strings.Contains(parseLowerKey, "email") {
			parseSanitized[parseKey] = "[redacted]"
			continue
		}
		parseSanitized[parseKey] = sanitizeDebugDetailValue(parseValue, 0)
	}
	return parseSanitized
}

// shouldSkipDebugEvent suppresses repeated identical debug events within a short window to reduce noise.
func shouldSkipDebugEvent(parseEvent string, parseDetails map[string]any) bool {
	parseEncoded, parseErr := json.Marshal(parseDetails)
	if parseErr != nil {
		return false
	}
	parseSignature := parseEvent + "|" + string(parseEncoded)
	parseNow := time.Now()
	if parsePrevious, parseOK := cacheDebugEventAt[parseSignature]; parseOK && parseNow.Sub(parsePrevious) < (1200*time.Millisecond) {
		return true
	}
	cacheDebugEventAt[parseSignature] = parseNow
	if len(cacheDebugEventAt) > 256 {
		for parseKey, parseAt := range cacheDebugEventAt {
			if parseNow.Sub(parseAt) > 45*time.Second {
				delete(cacheDebugEventAt, parseKey)
			}
		}
	}
	return false
}

func debugLog(parseEvent string, parseDetails map[string]any) {
	if !isAtlasDebugLoggingEnabled() {
		return
	}
	parseConsole := js.Global().Get("console")
	if !parseConsole.Truthy() || !parseConsole.Get("log").Truthy() {
		return
	}
	if parseDetails == nil {
		parseDetails = map[string]any{}
	}
	parseDetails["event"] = parseEvent
	parseDetails = sanitizeDebugDetails(parseDetails)
	if shouldSkipDebugEvent(parseEvent, parseDetails) {
		return
	}
	parseConsole.Call("log", "[atlas-wasm]", parseDetails)
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() {
		parseWindow.Set("__atlasDebugLast", js.ValueOf(parseDetails))
	}
}

// hasAtlasClass checks whether the document class list contains one expected token.
func hasAtlasClass(parseClassList string, parseExpected string) bool {
	for _, parseClassName := range strings.Fields(strings.TrimSpace(parseClassList)) {
		if parseClassName == parseExpected {
			return true
		}
	}
	return false
}

// logHydrationDocumentMismatch emits one debug-only event when SSR document attributes drift from bootstrap state.
func logHydrationDocumentMismatch(parsePayload atlas.Payload) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return
	}
	parseExpectedLocale := strings.TrimSpace(parsePayload.I18n.Locale)
	if parseExpectedLocale == "" {
		parseExpectedLocale = "en"
	}
	parseExpectedDirection := strings.TrimSpace(parsePayload.I18n.Direction)
	if parseExpectedDirection == "" {
		parseExpectedDirection = atlas.LocaleDirection(parseExpectedLocale)
	}
	parseExpectedThemeClass := "atlas-theme-dark"
	if strings.EqualFold(strings.TrimSpace(parsePayload.Theme.Mode), "light") {
		parseExpectedThemeClass = "atlas-theme-light"
	}
	parseExpectedDensityClass := "atlas-density-compact"
	if strings.EqualFold(strings.TrimSpace(parsePayload.Preferences.Density), "comfortable") {
		parseExpectedDensityClass = "atlas-density-comfortable"
	}
	parseActualClasses := strings.TrimSpace(parseRoot.Get("className").String())
	parseMismatches := map[string]any{}
	if parseActualLocale := strings.TrimSpace(parseRoot.Get("lang").String()); !strings.EqualFold(parseActualLocale, parseExpectedLocale) {
		parseMismatches["lang"] = map[string]string{"expected": parseExpectedLocale, "actual": parseActualLocale}
	}
	if parseActualDirection := strings.TrimSpace(parseRoot.Get("dir").String()); !strings.EqualFold(parseActualDirection, parseExpectedDirection) {
		parseMismatches["dir"] = map[string]string{"expected": parseExpectedDirection, "actual": parseActualDirection}
	}
	if !hasAtlasClass(parseActualClasses, parseExpectedThemeClass) {
		parseMismatches["theme_class"] = map[string]string{"expected": parseExpectedThemeClass, "actual": parseActualClasses}
	}
	if !hasAtlasClass(parseActualClasses, parseExpectedDensityClass) {
		parseMismatches["density_class"] = map[string]string{"expected": parseExpectedDensityClass, "actual": parseActualClasses}
	}
	if len(parseMismatches) == 0 {
		return
	}
	debugLog("hydrate.document.mismatch", map[string]any{
		"path":       parsePayload.Route.Path,
		"mismatches": parseMismatches,
	})
}

// logHydrationFallbackConditions emits debug-only logs when bootstrap-derived hydration fields are missing.
func logHydrationFallbackConditions(parsePayload atlas.Payload) {
	if strings.TrimSpace(parsePayload.Route.Path) == "" {
		debugLog("hydrate.fallback.missing_route_path", map[string]any{"reason": "bootstrap_missing_route_path"})
	}
	if strings.TrimSpace(parsePayload.I18n.Locale) == "" {
		debugLog("hydrate.fallback.missing_locale", map[string]any{"reason": "bootstrap_missing_locale"})
	}
	if strings.TrimSpace(parsePayload.Theme.Mode) == "" {
		debugLog("hydrate.fallback.missing_theme", map[string]any{"reason": "bootstrap_missing_theme"})
	}
}

func loadBootstrap() ui.SSRBootstrap {
	parseBootstrap, parseErr := ui.ReadBootstrapScript(atlasBootstrapScriptID)
	if parseErr == nil {
		debugLog("bootstrap.read.ok", map[string]any{
			"path":      parseBootstrap.Route.Path,
			"queryKeys": len(parseBootstrap.Route.Query),
			"paramKeys": len(parseBootstrap.Route.Params),
			"mode":      "inline",
		})
		return parseBootstrap
	}
	parseRef, parseRefErr := ui.ReadBootstrapReferenceScript(atlasBootstrapReferenceScriptID)
	if parseRefErr != nil {
		debugLog("bootstrap.read.failed", map[string]any{"error": parseErr.Error(), "referenceError": parseRefErr.Error()})
		return ui.SSRBootstrap{}
	}
	parseBootstrap, parseRefErr = ui.ReadBootstrapReference(parseRef)
	if parseRefErr != nil {
		debugLog("bootstrap.reference.failed", map[string]any{"url": parseRef.URL, "error": parseRefErr.Error()})
		return ui.SSRBootstrap{}
	}
	debugLog("bootstrap.read.ok", map[string]any{
		"path":      parseBootstrap.Route.Path,
		"queryKeys": len(parseBootstrap.Route.Query),
		"paramKeys": len(parseBootstrap.Route.Params),
		"mode":      "external",
		"url":       parseRef.URL,
	})
	return parseBootstrap
}

func cloneQueryValues(parseValues url.Values) map[string][]string {
	parseCloned := make(map[string][]string, len(parseValues))
	for parseKey, parseItem := range parseValues {
		parseCloned[parseKey] = append([]string(nil), parseItem...)
	}
	return parseCloned
}

func cloneDataMap(parseInput map[string]any) map[string]any {
	parseCloned := make(map[string]any, len(parseInput))
	for parseKey, parseValue := range parseInput {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

func cloneRequests(parseInput map[string]atlas.Request) map[string]atlas.Request {
	parseCloned := make(map[string]atlas.Request, len(parseInput))
	for parseKey, parseRequest := range parseInput {
		parseCloned[parseKey] = atlas.Request{
			Method: parseRequest.Method,
			URL:    parseRequest.URL,
			Status: parseRequest.Status,
			Data:   cloneDataMap(parseRequest.Data),
		}
	}
	return parseCloned
}

func queryValuesFromMap(parseInput map[string][]string) url.Values {
	parseValues := url.Values{}
	for parseKey, parseItems := range parseInput {
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	return parseValues
}

func decodeJSONBody(parseResponse *http.Response, parseTarget any) error {
	defer parseResponse.Body.Close()
	return json.NewDecoder(parseResponse.Body).Decode(parseTarget)
}

func routeDataQuery(parseQuery url.Values) url.Values {
	parseFiltered := url.Values{}
	for parseKey, parseItems := range parseQuery {
		parseTrimmedKey := strings.TrimSpace(parseKey)
		if strings.EqualFold(parseTrimmedKey, atlasNoticeQueryKey) || strings.EqualFold(parseTrimmedKey, atlasBootstrapModeQueryKey) {
			continue
		}
		for _, parseItem := range parseItems {
			parseFiltered.Add(parseKey, parseItem)
		}
	}
	return parseFiltered
}

func routeKey(parsePath string, parseQuery map[string][]string) string {
	parseValues := url.Values{}
	for parseKey, parseItems := range parseQuery {
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	parseEncoded := parseValues.Encode()
	if parseEncoded == "" {
		return parsePath
	}
	return parsePath + "?" + parseEncoded
}

func pathWithQuery(parsePath string, parseQuery url.Values) string {
	parseEncoded := parseQuery.Encode()
	if parseEncoded == "" {
		return parsePath
	}
	return parsePath + "?" + parseEncoded
}

func restoreAtlasFetchCacheBootstrap(parsePayload atlas.Payload) error {
	atlas.SeedFetchCacheBootstrap(&initialBootstrap, parsePayload, atlasCacheOptions, time.Now())
	return fetch.RestoreCacheBootstrap(initialBootstrap)
}

func fetchPageData(parseCtx context.Context, parsePath string, parseQuery url.Values) (any, string, error) {
	parseRequestURL := atlas.StartupRequestURL(parsePath, routeDataQuery(parseQuery))
	debugLog("route.fetch.start", map[string]any{"path": parsePath, "query": parseQuery.Encode(), "requestURL": parseRequestURL})
	if strings.TrimSpace(parseRequestURL) == "" {
		debugLog("route.fetch.skipped", map[string]any{"path": parsePath, "reason": "empty request url"})
		return nil, "", nil
	}
	parseData, parseErr := fetchRequestData(parseCtx, parseRequestURL, "page")
	if parseErr != nil {
		return nil, parseRequestURL, parseErr
	}
	return parseData, parseRequestURL, nil
}

func fetchRequestData(parseCtx context.Context, parseRequestURL string, parseDataKey string) (any, error) {
	parsePayloadData, parseErr := fetch.LoadCached(parseCtx, atlas.CachedRequestResourceKey(parseRequestURL, parseDataKey), func(parseLoadCtx context.Context) (any, error) {
		parseRequest, parseErr2 := http.NewRequestWithContext(parseLoadCtx, http.MethodGet, parseRequestURL, nil)
		if parseErr2 != nil {
			debugLog("route.fetch.request.error", map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey, "error": parseErr2.Error()})
			return nil, parseErr2
		}
		parseResponse, parseErr2 := http.DefaultClient.Do(parseRequest)
		if parseErr2 != nil {
			debugLog("route.fetch.network.error", map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey, "error": parseErr2.Error()})
			return nil, parseErr2
		}
		if parseResponse.StatusCode < 200 || parseResponse.StatusCode >= 300 {
			debugLog("route.fetch.non_ok", map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey, "status": parseResponse.StatusCode})
			parseResponse.Body.Close()
			return nil, nil
		}
		var parseLoaded any
		if parseErr3 := decodeJSONBody(parseResponse, &parseLoaded); parseErr3 != nil {
			debugLog("route.fetch.decode.error", map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey, "error": parseErr3.Error()})
			return nil, parseErr3
		}
		debugLog("route.fetch.ok", map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey})
		return parseLoaded, nil
	}, atlasCacheOptions)
	if parseErr == nil {
		debugLog("route.fetch.cache.hit", map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey})
		if parseNotice := strings.TrimSpace(cacheRevalidationNotice); parseNotice != "" {
			debugLog("revalidation.stale.refresh.cache", map[string]any{
				"notice":     parseNotice,
				"requestURL": parseRequestURL,
				"dataKey":    parseDataKey,
			})
		}
	}
	return parsePayloadData, parseErr
}

func routeSurface(parsePath string) string {
	if parsePath == atlas.RouteAppRoot || strings.HasPrefix(parsePath, "/app/") {
		return "internal"
	}
	return "public"
}

func routeScreen(parsePath string) string {
	switch {
	case strings.HasPrefix(parsePath, atlas.RouteInventory+"/") && strings.HasSuffix(parsePath, "/threshold-history"):
		return "sku-threshold-history"
	case parsePath == atlas.RouteLanding:
		return "landing"
	case parsePath == atlas.RouteCatalog:
		return "catalog"
	case strings.HasPrefix(parsePath, atlas.RouteCatalog+"/"):
		return "product"
	case parsePath == atlas.RouteWarehouses:
		return "warehouses"
	case strings.Contains(parsePath, "/availability/"):
		return "warehouse-availability"
	case strings.HasPrefix(parsePath, atlas.RouteWarehouses+"/"):
		return "warehouse-detail"
	case parsePath == atlas.RouteAppRoot || parsePath == atlas.RouteDashboard:
		return "dashboard"
	case parsePath == "/app/products":
		return "products"
	case strings.HasPrefix(parsePath, "/app/products/"):
		return "product-editor"
	case parsePath == atlas.RouteInventory:
		return "inventory"
	case strings.HasPrefix(parsePath, atlas.RouteInventory+"/"):
		return "sku-detail"
	case parsePath == atlas.RouteWarehouseOps:
		return "warehouse-ops"
	case strings.HasPrefix(parsePath, atlas.RouteWarehouseOps+"/") && strings.Contains(parsePath, "/items/"):
		return "warehouse-item-detail"
	case strings.HasPrefix(parsePath, atlas.RouteWarehouseOps+"/"):
		return "warehouse-detail"
	case parsePath == atlas.RouteTransfers:
		return "transfers"
	case strings.HasPrefix(parsePath, atlas.RouteTransfers+"/"):
		return "transfer-detail"
	case parsePath == atlas.RoutePurchaseOrders:
		return "purchase-orders"
	case strings.HasPrefix(parsePath, atlas.RoutePurchaseOrders+"/"):
		return "purchase-order-detail"
	case parsePath == atlas.RouteReceiving:
		return "receiving"
	case strings.HasPrefix(parsePath, atlas.RouteReceiving+"/"):
		return "receiving-session-detail"
	case strings.HasPrefix(parsePath, atlas.RouteComments+"/moderation/"):
		return "comments-moderation"
	case strings.HasPrefix(parsePath, atlas.RouteComments+"/"):
		return "comment-detail"
	case parsePath == atlas.RouteComments:
		return "comments"
	case parsePath == atlas.RouteSettingsAppearance:
		return "settings-appearance"
	case parsePath == atlas.RouteSettingsLocale:
		return "settings-locale"
	case parsePath == atlas.RouteSettingsWorkspaceDefaults:
		return "settings-workspace-defaults"
	case parsePath == atlas.RouteSettings:
		return "settings"
	default:
		return "recovery"
	}
}

func decodeMapValue[T any](parseInput any) (T, bool) {
	var parseTarget T
	parseEncoded, parseErr := json.Marshal(parseInput)
	if parseErr != nil {
		return parseTarget, false
	}
	if parseErr2 := json.Unmarshal(parseEncoded, &parseTarget); parseErr2 != nil {
		return parseTarget, false
	}
	return parseTarget, true
}

func routeTitle(parsePath string, parsePageData any) string {
	parseMetadata := atlas.MetadataForPath(parsePath)
	switch {
	case strings.HasPrefix(parsePath, atlas.RouteCatalog+"/"):
		if parseProduct, parseOk := decodeMapValue[struct {
			Title   string `json:"title"`
			Product struct {
				Title string `json:"title"`
			} `json:"product"`
		}](parsePageData); parseOk && strings.TrimSpace(parseProduct.Title) != "" {
			return "Atlas " + parseProduct.Title
		} else if parseOk && strings.TrimSpace(parseProduct.Product.Title) != "" {
			return "Atlas " + parseProduct.Product.Title
		}
	case strings.HasPrefix(parsePath, atlas.RouteWarehouses+"/") && !strings.Contains(parsePath, "/availability/"):
		return "Atlas Warehouse Detail"
	case strings.HasPrefix(parsePath, atlas.RouteWarehouseOps+"/") && strings.Contains(parsePath, "/items/"):
		if parseItemPage, parseOk2 := decodeMapValue[struct {
			Item struct {
				Title string `json:"title"`
			} `json:"item"`
			Product struct {
				Title string `json:"title"`
			} `json:"product"`
		}](parsePageData); parseOk2 {
			if strings.TrimSpace(parseItemPage.Product.Title) != "" {
				return "Atlas " + parseItemPage.Product.Title
			}
			if strings.TrimSpace(parseItemPage.Item.Title) != "" {
				return "Atlas " + parseItemPage.Item.Title
			}
		}
	}
	return parseMetadata.Title
}

func routeDescription(parsePath string, parsePageData any) string {
	parseMetadata := atlas.MetadataForPath(parsePath)
	switch {
	case strings.HasPrefix(parsePath, atlas.RouteCatalog+"/"):
		if parseProduct, parseOk := decodeMapValue[struct {
			SEODescription string `json:"seoDescription"`
			Summary        string `json:"summary"`
			Product        struct {
				SEODescription string `json:"seoDescription"`
				Summary        string `json:"summary"`
			} `json:"product"`
		}](parsePageData); parseOk {
			if strings.TrimSpace(parseProduct.SEODescription) != "" {
				return parseProduct.SEODescription
			}
			if strings.TrimSpace(parseProduct.Summary) != "" {
				return parseProduct.Summary
			}
			if strings.TrimSpace(parseProduct.Product.SEODescription) != "" {
				return parseProduct.Product.SEODescription
			}
			if strings.TrimSpace(parseProduct.Product.Summary) != "" {
				return parseProduct.Product.Summary
			}
		}
	case strings.HasPrefix(parsePath, atlas.RouteWarehouses+"/") && !strings.Contains(parsePath, "/availability/"):
		if parseWarehousePage, parseOk2 := decodeMapValue[struct {
			Warehouse struct {
				PublicSummary string `json:"publicSummary"`
			} `json:"warehouse"`
		}](parsePageData); parseOk2 && strings.TrimSpace(parseWarehousePage.Warehouse.PublicSummary) != "" {
			return parseWarehousePage.Warehouse.PublicSummary
		}
	}
	return parseMetadata.Description
}

func routeCanonical(parsePath string) string {
	return parsePath
}

func ensureManagedTitle(parseDocument js.Value) js.Value {
	parseTitle := parseDocument.Call("querySelector", `title[`+atlasManagedAttr+`="true"]`)
	if parseTitle.Truthy() {
		return parseTitle
	}
	parseTitle = parseDocument.Call("querySelector", "title")
	if parseTitle.Truthy() {
		parseTitle.Call("setAttribute", atlasManagedAttr, "true")
		return parseTitle
	}
	parseHead := parseDocument.Get("head")
	if !parseHead.Truthy() {
		return js.Undefined()
	}
	parseTitle = parseDocument.Call("createElement", "title")
	parseTitle.Call("setAttribute", atlasManagedAttr, "true")
	parseHead.Call("appendChild", parseTitle)
	return parseTitle
}

func ensureManagedHeadNode(parseDocument js.Value, parseSelector string, parseTag string, parseInit func(js.Value)) js.Value {
	parseNode := parseDocument.Call("querySelector", parseSelector+`[`+atlasManagedAttr+`="true"]`)
	if parseNode.Truthy() {
		return parseNode
	}
	parseNode = parseDocument.Call("querySelector", parseSelector)
	if parseNode.Truthy() {
		parseNode.Call("setAttribute", atlasManagedAttr, "true")
		return parseNode
	}
	parseHead := parseDocument.Get("head")
	if !parseHead.Truthy() {
		return js.Undefined()
	}
	parseNode = parseDocument.Call("createElement", parseTag)
	if parseInit != nil {
		parseInit(parseNode)
	}
	parseNode.Call("setAttribute", atlasManagedAttr, "true")
	parseHead.Call("appendChild", parseNode)
	return parseNode
}

func updateDocumentMetadata(parsePayload atlas.Payload) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	debugLog("metadata.update", map[string]any{
		"path":        parsePayload.Route.Path,
		"title":       parsePayload.Route.Title,
		"description": parsePayload.Route.Description,
		"canonical":   parsePayload.Route.Canonical,
	})
	if strings.TrimSpace(parsePayload.Route.Title) != "" {
		parseTitle := ensureManagedTitle(parseDocument)
		if parseTitle.Truthy() {
			parseTitle.Set("textContent", parsePayload.Route.Title)
		}
		parseDocument.Set("title", parsePayload.Route.Title)
	}
	if parseDescription := strings.TrimSpace(parsePayload.Route.Description); parseDescription != "" {
		parseMeta := ensureManagedHeadNode(parseDocument, `meta[name="description"]`, "meta", func(parseNode js.Value) {
			parseNode.Call("setAttribute", "name", "description")
		})
		if parseMeta.Truthy() {
			parseMeta.Call("setAttribute", "content", parseDescription)
		}
	}
	if parseCanonicalValue := strings.TrimSpace(parsePayload.Route.Canonical); parseCanonicalValue != "" {
		parseCanonical := ensureManagedHeadNode(parseDocument, `link[rel="canonical"]`, "link", func(parseNode2 js.Value) {
			parseNode2.Call("setAttribute", "rel", "canonical")
		})
		if parseCanonical.Truthy() {
			parseCanonical.Call("setAttribute", "href", parseCanonicalValue)
		}
	}
}

func syncNavigationPosition(parsePayload atlas.Payload) {
	parseCurrentKey := routeKey(parsePayload.Route.Path, parsePayload.Route.Query)
	if !hasRenderedRoute {
		hasRenderedRoute = true
		lastRenderedRouteKey = parseCurrentKey
		return
	}
	if parseCurrentKey == lastRenderedRouteKey {
		return
	}
	lastRenderedRouteKey = parseCurrentKey
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() && parseWindow.Get("scrollTo").Truthy() {
		parseWindow.Call("scrollTo", 0, 0)
	}
	debugLog("navigation.scroll.reset", map[string]any{"path": parsePayload.Route.Path, "key": parseCurrentKey})
}

func buildPayload(parsePath string, parseQuery url.Values, parsePageData any, parseRequestURL string) atlas.Payload {
	parseData := map[string]any{"page": parsePageData}
	parseRequests := map[string]atlas.Request{}
	if strings.TrimSpace(parseRequestURL) != "" {
		parseRequests["page"] = atlas.Request{
			Method: http.MethodGet,
			URL:    parseRequestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"page": parsePageData},
		}
	}
	return buildPayloadWithData(parsePath, parseQuery, parseData, parseRequests)
}

func buildPayloadWithData(parsePath string, parseQuery url.Values, parseData map[string]any, parseRequests map[string]atlas.Request) atlas.Payload {
	parsePageData := parseData["page"]
	parsePayload := initialPayload
	parsePayload.Route = atlas.RouteBootstrap{
		Path:        parsePath,
		Query:       cloneQueryValues(parseQuery),
		Params:      map[string]string{},
		Surface:     routeSurface(parsePath),
		Screen:      routeScreen(parsePath),
		Title:       routeTitle(parsePath, parsePageData),
		Description: routeDescription(parsePath, parsePageData),
		Canonical:   routeCanonical(parsePath),
	}
	parsePayload.Data = cloneDataMap(parseData)
	parsePayload.Requests = cloneRequests(parseRequests)
	debugLog("payload.build", map[string]any{
		"path":        parsePath,
		"screen":      parsePayload.Route.Screen,
		"surface":     parsePayload.Route.Surface,
		"requestKeys": len(parsePayload.Requests),
	})
	return parsePayload
}

func routeRecoveryFallback(parsePath string) (string, string) {
	if routeSurface(parsePath) == "internal" {
		return atlas.RouteDashboard, "Back to dashboard"
	}
	return atlas.RouteLanding, "Back to Atlas"
}

func routeRecoveryCopy(parsePath string) (string, string) {
	if routeSurface(parsePath) == "internal" {
		return "Atlas could not load this admin screen right now.", "The page shell is still active, so you can recover without losing the rest of the session."
	}
	return "Atlas could not load this page right now.", "The page shell is still active, so you can recover without a full crash."
}

func buildRecoveryPayload(parsePath string, parseQuery url.Values, parseRequestURL string, parseErr error) atlas.Payload {
	parseMetadata := atlas.MetadataForPath(parsePath)
	parseTitle := strings.TrimSpace(parseMetadata.Title)
	if parseTitle == "" {
		parseTitle = "Atlas"
	}
	parseRecoveryHref, parseRecoveryLabel := routeRecoveryFallback(parsePath)
	parseMessage, parseSummary := routeRecoveryCopy(parsePath)
	parseDetailParts := []string{parseSummary}
	if strings.TrimSpace(parseRequestURL) != "" {
		parseDetailParts = append(parseDetailParts, fmt.Sprintf("Request: %s", parseRequestURL))
	}
	if parseErr != nil {
		parseDetailParts = append(parseDetailParts, "Technical detail: "+parseErr.Error())
	}
	parsePayload := initialPayload
	parsePayload.Route = atlas.RouteBootstrap{
		Path:        parsePath,
		Query:       cloneQueryValues(parseQuery),
		Params:      map[string]string{},
		Surface:     routeSurface(parsePath),
		Screen:      "recovery",
		Title:       parseMetadata.Title + " Unavailable",
		Description: parseMetadata.Description,
		Canonical:   routeCanonical(parsePath),
	}
	parsePayload.Data = map[string]any{"page": map[string]any{
		"title":         parseTitle + " unavailable",
		"message":       parseMessage,
		"recoveryHref":  parseRecoveryHref,
		"recoveryLabel": parseRecoveryLabel,
		"detail":        strings.Join(parseDetailParts, " "),
	}}
	parsePayload.Requests = map[string]atlas.Request{}
	if strings.TrimSpace(parseRequestURL) != "" {
		parseStatus := http.StatusServiceUnavailable
		if parseErr == nil {
			parseStatus = http.StatusBadGateway
		}
		parsePayload.Requests["page"] = atlas.Request{
			Method: http.MethodGet,
			URL:    parseRequestURL,
			Status: parseStatus,
		}
	}
	debugLog("payload.recovery", map[string]any{
		"path":       parsePath,
		"screen":     parsePayload.Route.Screen,
		"requestURL": parseRequestURL,
		"error": func() string {
			if parseErr == nil {
				return ""
			}
			return parseErr.Error()
		}(),
	})
	return parsePayload
}

func loadRoutePayload(parseCtx context.Context, parseRouteCtx router.RouteContext) (atlas.Payload, error) {
	debugLog("route.load.start", map[string]any{
		"path":  parseRouteCtx.Path,
		"query": parseRouteCtx.Query.Values().Encode(),
	})
	parseRevalidationNotice := strings.TrimSpace(cacheRevalidationNotice)
	parseLoadStarted := time.Now()
	if parseRevalidationNotice != "" {
		debugLog("revalidation.loader.start", map[string]any{
			"path":   parseRouteCtx.Path,
			"notice": parseRevalidationNotice,
		})
	}
	parseFullQuery := parseRouteCtx.Query.Values()
	parsePayload, parseLoadErr := fetch.LoadCached(parseCtx, atlas.RoutePayloadResourceKey(parseRouteCtx.Path, parseFullQuery), func(parseLoadCtx context.Context) (atlas.Payload, error) {
		parsePageData, parseRequestURL, parseErr := fetchPageData(parseLoadCtx, parseRouteCtx.Path, parseFullQuery)
		if parseErr != nil {
			debugLog("route.load.error", map[string]any{"path": parseRouteCtx.Path, "error": parseErr.Error()})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parseRequestURL, parseErr), nil
		}
		if parsePageData == nil && strings.TrimSpace(parseRequestURL) == "" && parseRouteCtx.Path == atlas.RouteLanding {
			debugLog("route.load.static", map[string]any{"path": parseRouteCtx.Path, "reason": "static landing route"})
			return buildPayload(parseRouteCtx.Path, parseFullQuery, map[string]any{}, ""), nil
		}
		if parsePageData == nil {
			debugLog("route.load.missing", map[string]any{"path": parseRouteCtx.Path, "requestURL": parseRequestURL})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parseRequestURL, nil), nil
		}
		parsePayload := buildPayload(parseRouteCtx.Path, parseFullQuery, parsePageData, parseRequestURL)
		debugLog("route.load.ok", map[string]any{"path": parseRouteCtx.Path, "title": parsePayload.Route.Title})
		return parsePayload, nil
	}, atlasCacheOptions)
	if parseRevalidationNotice != "" {
		parseState := "ok"
		if parseLoadErr != nil {
			parseState = "error"
		}
		parseDurationMS := time.Since(parseLoadStarted).Milliseconds()
		if parseDurationMS < 0 {
			parseDurationMS = 0
		}
		debugLog("revalidation.loader.complete", map[string]any{
			"path":        parseRouteCtx.Path,
			"notice":      parseRevalidationNotice,
			"duration_ms": parseDurationMS,
			"state":       parseState,
		})
		cacheRevalidationNotice = ""
	}
	return parsePayload, parseLoadErr
}

func serializeGuardedForm(parseForm js.Value) string {
	parseFields := parseForm.Call("querySelectorAll", "input, textarea, select")
	parseParts := make([]string, 0, parseFields.Get("length").Int())
	for parseI := 0; parseI < parseFields.Get("length").Int(); parseI++ {
		parseField := parseFields.Index(parseI)
		if !parseField.Truthy() || parseField.Get("disabled").Bool() {
			continue
		}
		parseName := strings.TrimSpace(parseField.Get("name").String())
		if parseName == "" {
			continue
		}
		parseFieldType := strings.ToLower(strings.TrimSpace(parseField.Get("type").String()))
		switch parseFieldType {
		case "submit", "button", "reset", "file":
			continue
		}
		parseValue := parseField.Get("value").String()
		if parseFieldType == "checkbox" || parseFieldType == "radio" {
			if parseField.Get("checked").Bool() {
				parseValue = "1"
			} else {
				parseValue = "0"
			}
		}
		parseParts = append(parseParts, parseName+"="+parseValue)
	}
	return strings.Join(parseParts, "\x1f")
}

func refreshDirtyGuardState(parseForm js.Value) {
	parseInitial := parseForm.Get("__atlasDirtyGuardInitial").String()
	parseCurrent := serializeGuardedForm(parseForm)
	parseForm.Set("__atlasDirtyGuardDirty", parseCurrent != parseInitial)
}

func bindDirtyGuardForms() {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseForms := parseDocument.Call("querySelectorAll", "form["+atlasDirtyGuardAttr+"]")
	for parseI := 0; parseI < parseForms.Get("length").Int(); parseI++ {
		parseForm := parseForms.Index(parseI)
		if !parseForm.Truthy() {
			continue
		}
		if parseForm.Get("__atlasDirtyGuardBound").Truthy() && parseForm.Get("__atlasDirtyGuardBound").Bool() {
			refreshDirtyGuardState(parseForm)
			continue
		}
		parseForm.Set("__atlasDirtyGuardInitial", serializeGuardedForm(parseForm))
		parseForm.Set("__atlasDirtyGuardDirty", false)
		parseForm.Set("__atlasDirtyGuardSubmitting", false)

		parseUpdateHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseForm.Set("__atlasDirtyGuardSubmitting", false)
			refreshDirtyGuardState(parseForm)
			return nil
		})
		parseSubmitHandler := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseForm.Set("__atlasDirtyGuardSubmitting", true)
			parseForm.Set("__atlasDirtyGuardDirty", false)
			return nil
		})
		parseForm.Call("addEventListener", "input", parseUpdateHandler)
		parseForm.Call("addEventListener", "change", parseUpdateHandler)
		parseForm.Call("addEventListener", "submit", parseSubmitHandler)
		dirtyGuardHandlers = append(dirtyGuardHandlers, parseUpdateHandler, parseSubmitHandler)
		parseForm.Set("__atlasDirtyGuardBound", true)
	}
}

func activeDirtyGuardMessage() string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseForm := parseDocument.Call("querySelector", "form["+atlasDirtyGuardAttr+"]")
	if !parseForm.Truthy() || parseForm.Get("__atlasDirtyGuardSubmitting").Bool() {
		return ""
	}
	refreshDirtyGuardState(parseForm)
	if !parseForm.Get("__atlasDirtyGuardDirty").Bool() {
		return ""
	}
	parseMessage := strings.TrimSpace(parseForm.Call("getAttribute", atlasDirtyMessageAttr).String())
	if parseMessage == "" {
		parseMessage = "Leave this route and discard unsaved Atlas changes?"
	}
	return parseMessage
}

func registerBeforeUnloadGuard() {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
		return
	}
	beforeUnloadHandler = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseMessage := activeDirtyGuardMessage()
		if parseMessage == "" {
			return nil
		}
		parseEvent := parseArgs[0]
		if parseEvent.Get("preventDefault").Truthy() {
			parseEvent.Call("preventDefault")
		}
		parseEvent.Set("returnValue", parseMessage)
		return parseMessage
	})
	parseWindow.Call("addEventListener", "beforeunload", beforeUnloadHandler)
}

func shouldUseDocumentNavigation(parsePath string) bool {
	return strings.HasPrefix(parsePath, "/auth/")
}

func navigateDocument(parseTarget string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.Truthy() && parseLocation.Get("assign").Truthy() {
		parseLocation.Call("assign", parseTarget)
	}
}

func requireInternalSessionGuard(parseCtx router.RouteContext) router.GuardResult {
	if initialPayload.User != nil {
		return router.AllowNavigation()
	}
	parseNext := pathWithQuery(parseCtx.Path, parseCtx.Query.Values())
	parseValues := url.Values{}
	parseValues.Set("next", parseNext)
	parseTarget := atlasMockSignInPath + "?" + parseValues.Encode()
	debugLog("route.guard.auth.redirect", map[string]any{"path": parseCtx.Path, "target": parseTarget})
	navigateDocument(parseTarget)
	return router.BlockNavigation("Redirecting to mock sign in.")
}

func productEditorBeforeLeaveGuard(parseCurrent router.RouteContext, parseNext router.RouteContext) router.GuardResult {
	parseMessage := activeDirtyGuardMessage()
	if parseMessage == "" {
		return router.AllowNavigation()
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.Truthy() && parseWindow.Get("confirm").Truthy() && parseWindow.Call("confirm", parseMessage).Bool() {
		return router.AllowNavigation()
	}
	return router.BlockNavigation(parseMessage)
}

func invalidateCachesForPayload(parsePayload atlas.Payload) {
	parseNotice := strings.TrimSpace(queryValuesFromMap(parsePayload.Route.Query).Get(atlasNoticeQueryKey))
	if parseNotice == "" {
		return
	}
	cacheRevalidationNotice = parseNotice
	parseRoutePrefixes := atlas.MutationRoutePrefixes(parsePayload.Route.Path, parseNotice)
	parseActiveKeys := atlas.PayloadResourceKeys(parsePayload)
	parseRouteResourcePrefixes := make([]string, 0, len(parseRoutePrefixes))
	for _, parsePrefix := range parseRoutePrefixes {
		parseRouteResourcePrefixes = append(parseRouteResourcePrefixes, "atlas:route:"+parsePrefix)
	}
	parseRequestPrefixes := atlas.MutationRequestPrefixes(parseRoutePrefixes)
	parseRequestResourcePrefixes := make([]string, 0, len(parseRequestPrefixes))
	for _, parsePrefix2 := range parseRequestPrefixes {
		parseRequestResourcePrefixes = append(parseRequestResourcePrefixes, "atlas:request:"+parsePrefix2)
	}
	for _, parseInspection := range fetch.InspectCachedResources() {
		parseKey := parseInspection.Key
		if parseKey == "" {
			continue
		}
		if _, parseOk := parseActiveKeys[parseKey]; parseOk {
			continue
		}
		isParseMatched := false
		for _, parsePrefix3 := range parseRouteResourcePrefixes {
			if strings.HasPrefix(parseKey, parsePrefix3) {
				fetch.InvalidateResource(parseKey)
				isParseMatched = true
				break
			}
		}
		if isParseMatched {
			continue
		}
		for _, parsePrefix4 := range parseRequestResourcePrefixes {
			if strings.HasPrefix(parseKey, parsePrefix4) {
				fetch.InvalidateResource(parseKey)
				break
			}
		}
	}
	debugLog("route.cache.invalidate", map[string]any{
		"path":         parsePayload.Route.Path,
		"notice":       parseNotice,
		"routeTargets": parseRoutePrefixes,
	})
	debugLog("revalidation.invalidate", map[string]any{
		"path":         parsePayload.Route.Path,
		"notice":       parseNotice,
		"routeTargets": parseRoutePrefixes,
	})
}

// isAtlasListRoutePath reports whether parsePath is one of the list-heavy routes worth query-state logging.
func isAtlasListRoutePath(parsePath string) bool {
	parseTrimmedPath := strings.TrimSpace(parsePath)
	switch parseTrimmedPath {
	case atlas.RouteCatalog, atlas.RouteInventory, atlas.RouteWarehouseOps, atlas.RouteTransfers, atlas.RoutePurchaseOrders, atlas.RouteReceiving, atlas.RouteComments, "/app/products":
		return true
	default:
		return false
	}
}

// extractRouteQueryState captures the first value for each route query key to keep list-query logs concise.
func extractRouteQueryState(parsePayload atlas.Payload) map[string]string {
	parseState := map[string]string{}
	for parseKey, parseValues := range parsePayload.Route.Query {
		parseTrimmedKey := strings.TrimSpace(parseKey)
		if parseTrimmedKey == "" || len(parseValues) == 0 {
			continue
		}
		parseState[parseTrimmedKey] = strings.TrimSpace(parseValues[0])
	}
	return parseState
}

// extractPageFilterState reads server-provided list filter state from payload data when available.
func extractPageFilterState(parsePayload atlas.Payload) map[string]string {
	parseState := map[string]string{}
	parsePageData, parseOK := parsePayload.Data["page"]
	if !parseOK || parsePageData == nil {
		return parseState
	}
	parsePageMap, parseOK := parsePageData.(map[string]any)
	if !parseOK {
		return parseState
	}
	parseFiltersRaw, parseOK := parsePageMap["filters"]
	if !parseOK || parseFiltersRaw == nil {
		return parseState
	}
	if parseFilters, parseTyped := parseFiltersRaw.(map[string]any); parseTyped {
		for parseKey, parseValue := range parseFilters {
			parseTrimmedKey := strings.TrimSpace(parseKey)
			if parseTrimmedKey == "" {
				continue
			}
			switch parseTypedValue := parseValue.(type) {
			case string:
				parseState[parseTrimmedKey] = strings.TrimSpace(parseTypedValue)
			case float64:
				parseState[parseTrimmedKey] = fmt.Sprintf("%.0f", parseTypedValue)
			}
		}
		return parseState
	}
	if parseFilters, parseTyped := parseFiltersRaw.(map[string]string); parseTyped {
		for parseKey, parseValue := range parseFilters {
			parseTrimmedKey := strings.TrimSpace(parseKey)
			if parseTrimmedKey != "" {
				parseState[parseTrimmedKey] = strings.TrimSpace(parseValue)
			}
		}
	}
	return parseState
}

// findMatchingSavedViewName returns the first inventory saved view matching current sort and warehouse filters.
func findMatchingSavedViewName(parsePayload atlas.Payload, parseFilters map[string]string) string {
	parseCurrentSort := strings.TrimSpace(parseFilters["sort"])
	parseCurrentWarehouse := strings.TrimSpace(parseFilters["warehouse"])
	for _, parseSaved := range parsePayload.SavedViews {
		if !strings.EqualFold(strings.TrimSpace(parseSaved.Scope), "inventory") {
			continue
		}
		parseSavedSort := strings.TrimSpace(parseSaved.SortKey)
		parseSavedWarehouse := strings.TrimSpace(parseSaved.Filters["warehouse"])
		if parseSavedSort != "" && parseCurrentSort != "" && !strings.EqualFold(parseSavedSort, parseCurrentSort) {
			continue
		}
		if parseSavedWarehouse != "" && parseCurrentWarehouse != "" && !strings.EqualFold(parseSavedWarehouse, parseCurrentWarehouse) {
			continue
		}
		return strings.TrimSpace(parseSaved.Name)
	}
	return ""
}

// logListQueryState emits parsed list query and filter state for list-heavy route debugging.
func logListQueryState(parsePayload atlas.Payload) {
	if !isAtlasListRoutePath(parsePayload.Route.Path) {
		return
	}
	parseQueryState := extractRouteQueryState(parsePayload)
	parseFilterState := extractPageFilterState(parsePayload)
	parseSavedView := findMatchingSavedViewName(parsePayload, parseFilterState)
	debugLog("list.query.state", map[string]any{
		"path":             parsePayload.Route.Path,
		"query":            parseQueryState,
		"filters":          parseFilterState,
		"saved_view":       parseSavedView,
		"saved_view_count": len(parsePayload.SavedViews),
	})
}

func atlasRouteLoader(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
	debugLog("router.loader", map[string]any{"path": parseRouteCtx.Path})
	parsePayload, parseErr := loadRoutePayload(parseCtx, parseRouteCtx)
	if parseErr != nil {
		debugLog("router.loader.error", map[string]any{"path": parseRouteCtx.Path, "error": parseErr.Error()})
		return nil, parseErr
	}
	return router.Attrs{"payload": parsePayload}, nil
}

func loadInventoryThresholdHistoryPayload(parseCtx context.Context, parseRouteCtx router.RouteContext) (atlas.Payload, error) {
	parseFullQuery := parseRouteCtx.Query.Values()
	return fetch.LoadCached(parseCtx, atlas.RoutePayloadResourceKey(parseRouteCtx.Path, parseFullQuery), func(parseLoadCtx context.Context) (atlas.Payload, error) {
		parseSku := strings.TrimSpace(parseRouteCtx.Params.Get("sku"))
		parseParentPath := atlas.RouteInventory + "/" + parseSku
		parsePageData, parsePageRequestURL, parseErr := fetchPageData(parseLoadCtx, parseParentPath, parseFullQuery)
		if parseErr != nil {
			debugLog("route.overlay.page.error", map[string]any{"path": parseRouteCtx.Path, "error": parseErr.Error()})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, parseErr), nil
		}
		if parsePageData == nil {
			debugLog("route.overlay.page.missing", map[string]any{"path": parseRouteCtx.Path, "requestURL": parsePageRequestURL})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, nil), nil
		}
		parseOverlayRequestURL := atlas.StartupRequestURL(parseRouteCtx.Path, routeDataQuery(parseFullQuery))
		parseOverlayData, parseErr := fetchRequestData(parseLoadCtx, parseOverlayRequestURL, "overlay")
		if parseErr != nil {
			debugLog("route.overlay.data.error", map[string]any{"path": parseRouteCtx.Path, "requestURL": parseOverlayRequestURL, "error": parseErr.Error()})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parseOverlayRequestURL, parseErr), nil
		}
		parseRequests := map[string]atlas.Request{}
		if strings.TrimSpace(parsePageRequestURL) != "" {
			parseRequests["page"] = atlas.Request{
				Method: http.MethodGet,
				URL:    parsePageRequestURL,
				Status: http.StatusOK,
				Data:   map[string]any{"page": parsePageData},
			}
		}
		if strings.TrimSpace(parseOverlayRequestURL) != "" && parseOverlayData != nil {
			parseRequests["overlay"] = atlas.Request{
				Method: http.MethodGet,
				URL:    parseOverlayRequestURL,
				Status: http.StatusOK,
				Data:   map[string]any{"overlay": parseOverlayData},
			}
		}
		return buildPayloadWithData(parseRouteCtx.Path, parseFullQuery, map[string]any{
			"page":    parsePageData,
			"overlay": parseOverlayData,
		}, parseRequests), nil
	}, atlasCacheOptions)
}

func loadWarehouseItemNestedPayload(parseCtx context.Context, parseRouteCtx router.RouteContext) (atlas.Payload, error) {
	parseFullQuery := parseRouteCtx.Query.Values()
	return fetch.LoadCached(parseCtx, atlas.RoutePayloadResourceKey(parseRouteCtx.Path, parseFullQuery), func(parseLoadCtx context.Context) (atlas.Payload, error) {
		parseWarehouseID := strings.TrimSpace(parseRouteCtx.Params.Get("warehouseId"))
		parseParentPath := atlas.RouteWarehouseOps + "/" + parseWarehouseID
		parsePageData, parsePageRequestURL, parseErr := fetchPageData(parseLoadCtx, parseParentPath, parseFullQuery)
		if parseErr != nil {
			debugLog("route.nested.page.error", map[string]any{"path": parseRouteCtx.Path, "error": parseErr.Error()})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, parseErr), nil
		}
		if parsePageData == nil {
			debugLog("route.nested.page.missing", map[string]any{"path": parseRouteCtx.Path, "requestURL": parsePageRequestURL})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, nil), nil
		}
		parseItemRequestURL := atlas.StartupRequestURL(parseRouteCtx.Path, routeDataQuery(parseFullQuery))
		parseItemData, parseErr := fetchRequestData(parseLoadCtx, parseItemRequestURL, "item")
		if parseErr != nil {
			debugLog("route.nested.item.error", map[string]any{"path": parseRouteCtx.Path, "requestURL": parseItemRequestURL, "error": parseErr.Error()})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parseItemRequestURL, parseErr), nil
		}
		if parseItemData == nil {
			debugLog("route.nested.item.missing", map[string]any{"path": parseRouteCtx.Path, "requestURL": parseItemRequestURL})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parseItemRequestURL, nil), nil
		}
		parseRequests := map[string]atlas.Request{}
		if strings.TrimSpace(parsePageRequestURL) != "" {
			parseRequests["page"] = atlas.Request{
				Method: http.MethodGet,
				URL:    parsePageRequestURL,
				Status: http.StatusOK,
				Data:   map[string]any{"page": parsePageData},
			}
		}
		if strings.TrimSpace(parseItemRequestURL) != "" {
			parseRequests["item"] = atlas.Request{
				Method: http.MethodGet,
				URL:    parseItemRequestURL,
				Status: http.StatusOK,
				Data:   map[string]any{"item": parseItemData},
			}
		}
		return buildPayloadWithData(parseRouteCtx.Path, parseFullQuery, map[string]any{
			"page": parsePageData,
			"item": parseItemData,
		}, parseRequests), nil
	}, atlasCacheOptions)
}

func atlasThresholdHistoryOverlayLoader(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
	debugLog("router.loader.overlay", map[string]any{"path": parseRouteCtx.Path})
	parsePayload, parseErr := loadInventoryThresholdHistoryPayload(parseCtx, parseRouteCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	return router.Attrs{"payload": parsePayload}, nil
}

func atlasWarehouseItemNestedLoader(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
	debugLog("router.loader.nested", map[string]any{"path": parseRouteCtx.Path})
	parsePayload, parseErr := loadWarehouseItemNestedPayload(parseCtx, parseRouteCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	return router.Attrs{"payload": parsePayload}, nil
}

func applyPayloadSideEffects(parsePayload atlas.Payload, isResetScroll bool) {
	invalidateCachesForPayload(parsePayload)
	logListQueryState(parsePayload)
	updateDocumentMetadata(parsePayload)
	if isResetScroll {
		syncNavigationPosition(parsePayload)
	}
	bindDirtyGuardForms()
}

func atlasRouteComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, true)
	return ui.CreateElement(func() ui.Node {
		return atlas.App(parsePayload)
	})
}

func atlasThresholdHistoryOverlayComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render.overlay", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.InventoryThresholdHistoryOverlay(parsePayload)
	})
}

func atlasWarehouseItemNestedComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render.nested", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.WarehouseOpsItemPanel(parsePayload)
	})
}

func atlasWarehouseDetailNestedComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render.warehouse.nested", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.WarehouseOpsDetailPanel(parsePayload)
	})
}

func atlasCommentsNestedComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render.comments.nested", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.CommentsNestedPanel(parsePayload)
	})
}

func atlasPurchaseOrderNestedComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render.purchase_orders.nested", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.PurchaseOrderDetailPanel(parsePayload)
	})
}

func atlasSettingsNestedComponent(parseAttrs router.Attrs) *router.Element {
	parsePayload, _ := parseAttrs["payload"].(atlas.Payload)
	debugLog("route.render.settings.nested", map[string]any{"path": parsePayload.Route.Path, "screen": parsePayload.Route.Screen})
	applyPayloadSideEffects(parsePayload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.SettingsNestedPanel(parsePayload)
	})
}

type atlasRouteDefinition struct {
	Path        string
	MetadataKey string
	Layout      bool
	UseLoader   bool
	Loader      router.LoaderFunc
	Internal    bool
	BeforeLeave router.LeaveGuardFunc
	Redirect    string
	Component   func(router.Attrs) *router.Element
}

func atlasLoadingElement() *router.Element {
	return ui.CreateElement(func() ui.Node {
		return html.Div(html.Props{Class: "min-h-screen"})
	})
}

func routeOptionsForDefinition(parseDef atlasRouteDefinition) router.Options {
	parseOptions := router.Options{
		Redirect:    parseDef.Redirect,
		BeforeLeave: parseDef.BeforeLeave,
		Layout:      parseDef.Layout,
	}
	if strings.TrimSpace(parseDef.MetadataKey) != "" {
		parseMetadata := atlas.MetadataForPath(parseDef.MetadataKey)
		parseOptions.Title = parseMetadata.Title
		parseOptions.Description = parseMetadata.Description
		parseOptions.CanonicalURL = parseMetadata.Canonical
	}
	if parseDef.Internal {
		parseOptions.BeforeEnter = requireInternalSessionGuard
	}
	if parseDef.UseLoader {
		if parseDef.Loader != nil {
			parseOptions.Loader = parseDef.Loader
		} else {
			parseOptions.Loader = atlasRouteLoader
		}
		parseOptions.Loading = atlasLoadingElement()
	}
	return parseOptions
}

func registerAtlasRoutes(parseR *router.Router) {
	parseDefinitions := []atlasRouteDefinition{
		{Path: atlas.RouteLanding, MetadataKey: atlas.RouteLanding, UseLoader: true},
		{Path: atlas.RouteCatalog, MetadataKey: atlas.RouteCatalog, UseLoader: true},
		{Path: atlas.RouteProductPattern, MetadataKey: atlas.RouteProduct, UseLoader: true},
		{Path: atlas.RouteWarehouses, MetadataKey: atlas.RouteWarehouses, UseLoader: true},
		{Path: atlas.RouteWarehouseAvailabilityPattern, MetadataKey: atlas.RouteWarehouseAvailability, UseLoader: true},
		{Path: atlas.RouteWarehousePublicDetailPattern, MetadataKey: atlas.RouteWarehousePublicDetail, UseLoader: true},
		{Path: atlas.RouteAppRoot, MetadataKey: atlas.RouteDashboard, Internal: true, Redirect: atlas.RouteDashboard},
		{Path: atlas.RouteDashboard, MetadataKey: atlas.RouteDashboard, UseLoader: true, Internal: true},
		{Path: "/app/products", MetadataKey: "/app/products", UseLoader: true, Internal: true},
		{Path: "/app/products/:slug", MetadataKey: atlas.RouteProduct, UseLoader: true, Internal: true, BeforeLeave: productEditorBeforeLeaveGuard},
		{Path: atlas.RouteInventory, MetadataKey: atlas.RouteInventory, UseLoader: true, Internal: true},
		{Path: atlas.RouteSKUDetailRoute, MetadataKey: atlas.RouteSKUDetail, Layout: true, UseLoader: true, Internal: true},
		{Path: atlas.RouteSKUThresholdHistoryRoute, MetadataKey: atlas.RouteSKUThresholdHistory, UseLoader: true, Loader: atlasThresholdHistoryOverlayLoader, Internal: true, Component: atlasThresholdHistoryOverlayComponent},
		{Path: atlas.RouteWarehouseOps, MetadataKey: atlas.RouteWarehouseOps, Layout: true, UseLoader: true, Internal: true},
		{Path: atlas.RouteWarehouseDetailRoute, MetadataKey: atlas.RouteWarehouseDetail, Layout: true, UseLoader: true, Internal: true, Component: atlasWarehouseDetailNestedComponent},
		{Path: atlas.RouteWarehouseItemDetailRoute, MetadataKey: atlas.RouteWarehouseItemDetail, UseLoader: true, Loader: atlasWarehouseItemNestedLoader, Internal: true, Component: atlasWarehouseItemNestedComponent},
		{Path: atlas.RouteTransfers, MetadataKey: atlas.RouteTransfers, UseLoader: true, Internal: true},
		{Path: atlas.RouteTransferDetailRoute, MetadataKey: atlas.RouteTransferDetail, UseLoader: true, Internal: true},
		{Path: atlas.RoutePurchaseOrders, MetadataKey: atlas.RoutePurchaseOrders, Layout: true, UseLoader: true, Internal: true},
		{Path: atlas.RoutePurchaseOrderDetailRoute, MetadataKey: atlas.RoutePurchaseOrderDetail, UseLoader: true, Internal: true, Component: atlasPurchaseOrderNestedComponent},
		{Path: atlas.RouteReceiving, MetadataKey: atlas.RouteReceiving, UseLoader: true, Internal: true},
		{Path: atlas.RouteReceivingSessionDetailRoute, MetadataKey: atlas.RouteReceivingSessionDetail, UseLoader: true, Internal: true},
		{Path: atlas.RouteComments, MetadataKey: atlas.RouteComments, Layout: true, UseLoader: true, Internal: true},
		{Path: atlas.RouteCommentsModerationRoute, MetadataKey: atlas.RouteCommentsModeration, UseLoader: true, Internal: true, Component: atlasCommentsNestedComponent},
		{Path: atlas.RouteCommentDetailRoute, MetadataKey: atlas.RouteCommentDetail, UseLoader: true, Internal: true, Component: atlasCommentsNestedComponent},
		{Path: atlas.RouteSettings, MetadataKey: atlas.RouteSettings, Layout: true, UseLoader: true, Internal: true},
		{Path: atlas.RouteSettingsAppearance, MetadataKey: atlas.RouteSettingsAppearance, UseLoader: true, Internal: true, Component: atlasSettingsNestedComponent},
		{Path: atlas.RouteSettingsLocale, MetadataKey: atlas.RouteSettingsLocale, UseLoader: true, Internal: true, Component: atlasSettingsNestedComponent},
		{Path: atlas.RouteSettingsWorkspaceDefaults, MetadataKey: atlas.RouteSettingsWorkspaceDefaults, UseLoader: true, Internal: true, Component: atlasSettingsNestedComponent},
		{Path: atlas.RouteCatchAll, UseLoader: true},
	}
	for _, parseDef := range parseDefinitions {
		parseComponent := atlasRouteComponent
		if parseDef.Component != nil {
			parseComponent = parseDef.Component
		}
		parseR.Register(parseDef.Path, parseComponent, routeOptionsForDefinition(parseDef))
	}
}

func registerAnchorNavigation(parseRouterInstance *router.Router) {
	parseDocument := js.Global().Get("document")
	parseWindow := js.Global().Get("window")
	if !parseDocument.Truthy() || !parseWindow.Truthy() || !parseDocument.Get("addEventListener").Truthy() {
		return
	}
	anchorNavigationHandler = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEvent := parseArgs[0]
		if parseEvent.Get("defaultPrevented").Bool() {
			return nil
		}
		if parseEvent.Get("button").Truthy() && parseEvent.Get("button").Int() != 0 {
			return nil
		}
		if parseEvent.Get("metaKey").Bool() || parseEvent.Get("ctrlKey").Bool() || parseEvent.Get("shiftKey").Bool() || parseEvent.Get("altKey").Bool() {
			return nil
		}
		parseTarget := parseEvent.Get("target")
		if !parseTarget.Truthy() {
			return nil
		}
		parseAnchor := parseTarget.Call("closest", "a[href]")
		if !parseAnchor.Truthy() {
			return nil
		}
		if strings.TrimSpace(parseAnchor.Get("target").String()) != "" || parseAnchor.Get("download").Truthy() {
			return nil
		}
		parseHref := strings.TrimSpace(parseAnchor.Get("href").String())
		if parseHref == "" || strings.HasPrefix(parseHref, "mailto:") || strings.HasPrefix(parseHref, "tel:") || strings.HasPrefix(parseHref, "javascript:") {
			return nil
		}
		parseLocation := parseWindow.Get("location")
		parseAnchorURL := js.Global().Get("URL").New(parseHref, parseLocation.Get("href"))
		if parseAnchorURL.Get("origin").String() != parseLocation.Get("origin").String() {
			return nil
		}
		parsePathname := parseAnchorURL.Get("pathname").String()
		parseSearch := parseAnchorURL.Get("search").String()
		parseHash := parseAnchorURL.Get("hash").String()
		if shouldUseDocumentNavigation(parsePathname) {
			return nil
		}
		if parseHash != "" && parsePathname == parseLocation.Get("pathname").String() && parseSearch == parseLocation.Get("search").String() {
			return nil
		}
		debugLog("navigation.anchor.intercept", map[string]any{
			"href":     parseHref,
			"pathname": parsePathname,
			"search":   parseSearch,
			"hash":     parseHash,
		})
		parseEvent.Call("preventDefault")
		parseRouterInstance.Navigate(parsePathname + parseSearch)
		return nil
	})
	parseDocument.Call("addEventListener", "click", anchorNavigationHandler)
	debugLog("navigation.anchor.registered", nil)
}

// getOverlayDebugSelectors returns stable overlay selectors that are relevant to Atlas workflow debugging.
func getOverlayDebugSelectors() map[string]string {
	return map[string]string{
		"threshold-history-route-overlay": `[data-atlas-route-overlay="threshold-history"]`,
		"transfer-confirm-dialog":         `#atlas-transfer-confirm`,
		"receiving-confirm-dialog":        `#atlas-receiving-confirm`,
		"comment-confirm-dialog":          `#atlas-comment-review-confirm`,
		"bulk-moderation-confirm-dialog":  `#atlas-bulk-review-confirm`,
	}
}

// collectOverlayDebugState reports whether each tracked overlay selector is currently mounted.
func collectOverlayDebugState(parseDocument js.Value) map[string]bool {
	parseState := map[string]bool{}
	for parseName, parseSelector := range getOverlayDebugSelectors() {
		parseNode := parseDocument.Call("querySelector", parseSelector)
		parseState[parseName] = parseNode.Truthy()
	}
	return parseState
}

// getOverlayLocationPath returns the active browser pathname for overlay debug events.
func getOverlayLocationPath() string {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return ""
	}
	parseLocation := parseWindow.Get("location")
	if !parseLocation.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseLocation.Get("pathname").String())
}

// formatOverlayElementSummary builds a concise identifier for focused-element routing diagnostics.
func formatOverlayElementSummary(parseElement js.Value) string {
	if !parseElement.Truthy() {
		return "unknown"
	}
	parseTag := strings.ToLower(strings.TrimSpace(parseElement.Get("tagName").String()))
	parseID := strings.TrimSpace(parseElement.Get("id").String())
	parseClassName := strings.TrimSpace(parseElement.Get("className").String())
	parseSummary := parseTag
	if parseSummary == "" {
		parseSummary = "element"
	}
	if parseID != "" {
		parseSummary += "#" + parseID
	}
	if parseClassName != "" {
		parseSummary += "." + strings.ReplaceAll(strings.Join(strings.Fields(parseClassName), "."), "..", ".")
	}
	return parseSummary
}

// logOverlayDebugTransitions logs overlay mount and dismiss transitions in debug mode.
func logOverlayDebugTransitions(parsePrevious map[string]bool, parseCurrent map[string]bool) {
	parsePath := getOverlayLocationPath()
	for parseName, isCurrentOpen := range parseCurrent {
		isPreviousOpen := parsePrevious[parseName]
		if isCurrentOpen && !isPreviousOpen {
			debugLog("overlay.state.open", map[string]any{"overlay": parseName, "path": parsePath})
			if parseName == "receiving-confirm-dialog" {
				debugLog("overlay.discrepancy.confirm.open", map[string]any{"overlay": parseName, "path": parsePath})
			}
			continue
		}
		if !isCurrentOpen && isPreviousOpen {
			debugLog("overlay.state.dismiss", map[string]any{"overlay": parseName, "path": parsePath})
		}
	}
}

// checkOverlayFocusRouting logs when focus moves outside mounted overlays while modal flows remain open.
func checkOverlayFocusRouting(parseDocument js.Value, parseOpenState map[string]bool) {
	parseOpenNames := make([]string, 0, len(parseOpenState))
	parseOpenElements := make([]js.Value, 0, len(parseOpenState))
	for parseName, isOpen := range parseOpenState {
		if !isOpen {
			continue
		}
		parseSelector := getOverlayDebugSelectors()[parseName]
		parseElement := parseDocument.Call("querySelector", parseSelector)
		if !parseElement.Truthy() {
			continue
		}
		parseOpenNames = append(parseOpenNames, parseName)
		parseOpenElements = append(parseOpenElements, parseElement)
	}
	if len(parseOpenNames) == 0 {
		cacheOverlayFocusMismatch = ""
		return
	}
	parseActive := parseDocument.Get("activeElement")
	if !parseActive.Truthy() {
		return
	}
	for _, parseOverlayElement := range parseOpenElements {
		if parseOverlayElement.Call("contains", parseActive).Bool() {
			cacheOverlayFocusMismatch = ""
			return
		}
	}
	sort.Strings(parseOpenNames)
	parseActiveSummary := formatOverlayElementSummary(parseActive)
	parseMismatchKey := strings.Join(parseOpenNames, "|") + ">" + parseActiveSummary
	if parseMismatchKey == cacheOverlayFocusMismatch {
		return
	}
	cacheOverlayFocusMismatch = parseMismatchKey
	debugLog("overlay.focus.routing.issue", map[string]any{
		"path":          getOverlayLocationPath(),
		"open_overlays": parseOpenNames,
		"active":        parseActiveSummary,
	})
}

// registerOverlayDebugTracking mounts debug-only overlay observers for modal open/dismiss and focus routing.
func registerOverlayDebugTracking() {
	if !isAtlasDebugLoggingEnabled() {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() || !parseDocument.Get("body").Truthy() {
		return
	}
	trackOverlayDebugState = collectOverlayDebugState(parseDocument)
	handleOverlayDebugMutation = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseCurrentState := collectOverlayDebugState(parseDocument)
		logOverlayDebugTransitions(trackOverlayDebugState, parseCurrentState)
		checkOverlayFocusRouting(parseDocument, parseCurrentState)
		trackOverlayDebugState = parseCurrentState
		return nil
	})
	parseObserverCtor := js.Global().Get("MutationObserver")
	if parseObserverCtor.Truthy() {
		watchOverlayDebugObserver = parseObserverCtor.New(handleOverlayDebugMutation)
		watchOverlayDebugObserver.Call("observe", parseDocument.Get("body"), js.ValueOf(map[string]any{
			"childList":     true,
			"subtree":       true,
			"attributes":    true,
			"characterData": false,
		}))
	}
	handleOverlayDebugFocus = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		checkOverlayFocusRouting(parseDocument, trackOverlayDebugState)
		return nil
	})
	parseDocument.Call("addEventListener", "focusin", handleOverlayDebugFocus)
}

func main() {
	// Hydration boundary: resume from the exact server-emitted bootstrap payload so first paint metadata,
	// document language, and route data do not drift between SSR and client ownership.
	initialBootstrap = loadBootstrap()
	initialPayload = atlas.PayloadFromSSRBootstrap(initialBootstrap)
	logHydrationFallbackConditions(initialPayload)
	logHydrationDocumentMismatch(initialPayload)
	if parseErr := restoreAtlasFetchCacheBootstrap(initialPayload); parseErr != nil {
		debugLog("bootstrap.cache.restore.failed", map[string]any{"error": parseErr.Error()})
	}
	debugLog("app.init", map[string]any{
		"path":    initialPayload.Route.Path,
		"screen":  initialPayload.Route.Screen,
		"surface": initialPayload.Route.Surface,
	})
	lastRenderedRouteKey = routeKey(initialPayload.Route.Path, initialPayload.Route.Query)
	hasRenderedRoute = true
	updateDocumentMetadata(initialPayload)

	parseR := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: atlas.RouteLanding})
	registerAtlasRoutes(parseR)

	debugLog("hydrate.start", map[string]any{"selector": "#app", "path": initialPayload.Route.Path})
	_, _ = ui.Hydrate(atlas.App(initialPayload), "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	debugLog("hydrate.done", map[string]any{"selector": "#app"})
	bindDirtyGuardForms()
	registerBeforeUnloadGuard()
	parseR.HydrateMount("#app")
	debugLog("router.mount", map[string]any{"selector": "#app"})
	registerAnchorNavigation(parseR)
	registerOverlayDebugTracking()
	select {}
}
