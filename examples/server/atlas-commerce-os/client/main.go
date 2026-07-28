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

	"github.com/monstercameron/GoWebComponents/v5/css"
	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v5/fetch"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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

	// atlasLocaleQueryKey and atlasLocaleCookieName mirror the two inputs the server
	// resolves in resolveRequestLocale (server/server.go). They are spelled out here
	// rather than imported because shared/atlas holds the payload contract, not the
	// transport; the names have to match the server's by inspection, and the pair of
	// doc comments is the pin.
	atlasLocaleQueryKey     = "locale"
	atlasLocaleCookieName   = "atlas_locale"
	atlasLocaleCookieMaxAge = 365 * 24 * 60 * 60
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

// debugLog emits development diagnostics, and ONLY development diagnostics.
//
// It returns early unless data-atlas-debug-logs is set, which the server writes
// from cfg.LogsEnabled (formatAtlasDebugLogsFlag) and which is off by default. So
// everything logged through here is absent from a production page — fine for
// route.fetch.start, payload.build, list.query.state, and the overlay tracing this
// file is full of, all of which are noise once the feature works.
//
// It is NOT fine for failures, and this client used it for all of them. Use
// errorLog (bootsurface.go) for anything a support engineer would need after the
// fact; a diagnostic that is silenced in production is the same problem as no
// diagnostic, wearing a different costume.
//
// The always-on boot milestones live on the <html> data-atlas-boot-state attribute
// rather than here, so "did it get past hydration?" is answerable in every build
// without adding console noise to a healthy page.
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
		// errorLog, not debugLog: losing the bootstrap means the client falls back
		// to a zero payload and renders the default route with no user, no locale,
		// and no route data. It looks like a working page showing the wrong thing,
		// which is worse than a crash and used to be reported only when the
		// server had debug logs switched on.
		// Only a SERVER-rendered page is supposed to carry a bootstrap payload.
		// client/atlas-commerce-os.html is a static shell with no server behind it,
		// so on that page this is expected and stays a debug line; surfacing it
		// there put a failure banner over a demo that was working correctly.
		if !isServerRenderedDocument() {
			debugLog("bootstrap.absent.static_shell", map[string]any{
				"reason": "no data-atlas-surface attribute; treating this as a static shell with no SSR payload",
			})
			return ui.SSRBootstrap{}
		}
		errorLog("bootstrap.read.failed", parseErr.Error()+" / "+parseRefErr.Error(), map[string]any{
			"inlineScriptID":    atlasBootstrapScriptID,
			"referenceScriptID": atlasBootstrapReferenceScriptID,
		})
		surfaceAtlasFailure(atlasSurfaceModeNotice, "bootstrap-missing",
			"Atlas started without its server payload",
			"Neither the inline nor the external SSR bootstrap script could be read, so this page is rendering from empty state instead of the server's.",
			parseErr.Error())
		return ui.SSRBootstrap{}
	}
	parseBootstrap, parseRefErr = ui.ReadBootstrapReference(parseRef)
	if parseRefErr != nil {
		errorLog("bootstrap.reference.failed", parseRefErr.Error(), map[string]any{"url": parseRef.URL})
		surfaceAtlasFailure(atlasSurfaceModeNotice, "bootstrap-unreadable",
			"Atlas started without its server payload",
			"The external bootstrap reference was found but could not be fetched or decoded, so this page is rendering from empty state.",
			parseRef.URL+": "+parseRefErr.Error())
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
			errorLog("route.fetch.request.error", parseErr2.Error(), map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey})
			return nil, parseErr2
		}
		parseResponse, parseErr2 := http.DefaultClient.Do(parseRequest)
		if parseErr2 != nil {
			errorLog("route.fetch.network.error", parseErr2.Error(), map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey})
			return nil, parseErr2
		}
		if parseResponse.StatusCode < 200 || parseResponse.StatusCode >= 300 {
			// This branch returns (nil, nil): the status is DISCARDED and the
			// caller cannot distinguish "the server said 403" from "there was
			// nothing to load". Downstream that becomes a generic recovery page.
			// Fixing the signature is out of scope here, so at minimum the status
			// has to reach the console in every build — otherwise a 500 on a data
			// route is indistinguishable from an empty one.
			errorLog("route.fetch.non_ok", fmt.Sprintf("%s responded %d", parseRequestURL, parseResponse.StatusCode), map[string]any{
				"requestURL": parseRequestURL,
				"dataKey":    parseDataKey,
				"status":     parseResponse.StatusCode,
			})
			parseResponse.Body.Close()
			return nil, nil
		}
		var parseLoaded any
		if parseErr3 := decodeJSONBody(parseResponse, &parseLoaded); parseErr3 != nil {
			errorLog("route.fetch.decode.error", parseErr3.Error(), map[string]any{"requestURL": parseRequestURL, "dataKey": parseDataKey})
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
	applyDocumentLocale(parseDocument, parsePayload)
}

// applyDocumentLocale writes <html lang> and <html dir> from the payload.
//
// The server emits both on a direct entry, so on first paint this is a no-op that
// agrees with the HTML. It matters on the OTHER path: the language switcher renders
// ordinary <a href="/shop?locale=fr"> elements, registerAnchorNavigation intercepts
// them, and the router serves the navigation from the client — the server is never
// asked, so nothing outside this function can update the two attributes that carry
// the document's language to a screen reader and its text direction to the layout
// engine. Before this existed, clicking FR changed location.search and left
// <html lang="en"> in place, which is the precise shape of "the switcher navigates
// and nothing happens".
//
// Both attributes are set unconditionally rather than only-when-different: reading
// the attribute to compare costs the same DOM round-trip as writing it, and a
// conditional write is one more state to be wrong about.
func applyDocumentLocale(parseDocument js.Value, parsePayload atlas.Payload) {
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return
	}
	parseLocale := strings.TrimSpace(parsePayload.I18n.Locale)
	if parseLocale == "" {
		parseLocale = "en"
	}
	parseDirection := strings.TrimSpace(parsePayload.I18n.Direction)
	if parseDirection == "" {
		// Derive rather than default to "ltr": a payload that carried a locale but no
		// direction would otherwise render Arabic left-to-right, which looks like a
		// CSS bug and is actually a missing field.
		parseDirection = atlas.LocaleDirection(parseLocale)
	}
	parseRoot.Call("setAttribute", "lang", parseLocale)
	parseRoot.Call("setAttribute", "dir", parseDirection)
}

// resolveClientLocale is the client half of the locale contract. It implements the
// same precedence as resolveRequestLocale in server/server.go — query parameter,
// then the atlas_locale cookie, then what the server last told us — and the two
// halves have to agree, because a visitor can arrive at any URL either way:
//
//	?locale=fr          direct entry  -> server resolves, emits lang/dir, sets cookie
//	?locale=fr          in-page click -> THIS resolves, writes lang/dir, sets cookie
//
// Level (3) is the SSR payload rather than a stored preference row because the
// client has no database: initialPayload.I18n.Locale is the server's answer for this
// visitor, which already folded in the cookie and the preference row.
//
// The cookie write is what makes the choice survive the next plain link. The nav
// links (/shop, /warehouses) carry no locale parameter — publicLocaleHref only
// decorates the CURRENT path — so without it, one click off the switcher and the
// storefront is English again.
func resolveClientLocale(parseQuery url.Values) string {
	if parseLocale, parseOK := supportedClientLocale(parseQuery.Get(atlasLocaleQueryKey)); parseOK {
		writeLocaleCookie(parseLocale)
		return parseLocale
	}
	if parseLocale, parseOK := supportedClientLocale(readLocaleCookie()); parseOK {
		return parseLocale
	}
	if parseLocale, parseOK := supportedClientLocale(initialPayload.I18n.Locale); parseOK {
		return parseLocale
	}
	return "en"
}

// supportedClientLocale validates against the same list the server does. Both sides
// read atlas.SupportedLocales, which is the point of that function living in
// shared/atlas: an unsupported tag has to be rejected identically in both lanes or a
// reload changes the language.
func supportedClientLocale(parseCandidate string) (string, bool) {
	parseCandidate = strings.ToLower(strings.TrimSpace(parseCandidate))
	if parseCandidate == "" {
		return "", false
	}
	for _, parseSupported := range atlas.SupportedLocales() {
		if parseCandidate == strings.ToLower(parseSupported) {
			return parseSupported, true
		}
	}
	return "", false
}

// readLocaleCookie pulls atlas_locale out of document.cookie.
//
// document.cookie is a single "a=1; b=2" string with no accessor, so this is a hand
// split. It reads only the one name it knows and validates the value against the
// supported list at the call site, so a malformed or hostile cookie jar cannot put
// anything into <html lang>.
func readLocaleCookie() string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseJar := parseDocument.Get("cookie")
	if !parseJar.Truthy() {
		return ""
	}
	for _, parseEntry := range strings.Split(parseJar.String(), ";") {
		parseName, parseValue, parseFound := strings.Cut(strings.TrimSpace(parseEntry), "=")
		if !parseFound {
			continue
		}
		if strings.TrimSpace(parseName) == atlasLocaleCookieName {
			return strings.TrimSpace(parseValue)
		}
	}
	return ""
}

// writeLocaleCookie records an explicit in-page locale choice where the next full
// page load will find it.
//
// Same attributes the server writes (path=/, one year, SameSite=Lax) so a
// client-set and a server-set cookie are indistinguishable — otherwise the two
// lanes would produce two cookies with the same name and different scopes, and which
// one wins would depend on the browser's ordering rules.
func writeLocaleCookie(parseLocale string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseDocument.Set("cookie", atlasLocaleCookieName+"="+parseLocale+
		"; path=/; max-age="+fmt.Sprintf("%d", atlasLocaleCookieMaxAge)+"; samesite=lax")
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
	// Re-resolve the locale for EVERY client-built payload.
	//
	// parsePayload starts as a copy of initialPayload, so without this line I18n is
	// frozen at whatever the server rendered on first entry and the language switcher
	// is inert for the whole session. atlas.DefaultI18n derives Direction from the
	// locale, so `dir` cannot drift out of step with `lang`.
	parsePayload.I18n = atlas.DefaultI18n(resolveClientLocale(parseQuery))
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
	// A recovery screen is still a page in a language; see buildPayloadWithData.
	parsePayload.I18n = atlas.DefaultI18n(resolveClientLocale(parseQuery))
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
	// Always-on: every "missing"/"error" branch upstream funnels through here, so
	// this single line is the durable record that a user was shown a recovery
	// screen instead of the page they asked for. As a debugLog it printed nothing
	// in production, which meant the app could serve recovery pages all day and
	// leave no trace anywhere the operator could reach.
	parseReason := "no data returned"
	if parseErr != nil {
		parseReason = parseErr.Error()
	}
	errorLog("payload.recovery", parsePath+": "+parseReason, map[string]any{
		"path":       parsePath,
		"screen":     parsePayload.Route.Screen,
		"requestURL": parseRequestURL,
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
			errorLog("route.load.error", parseErr.Error(), map[string]any{"path": parseRouteCtx.Path})
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
		errorLog("router.loader.error", parseErr.Error(), map[string]any{"path": parseRouteCtx.Path})
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
			errorLog("route.overlay.page.error", parseErr.Error(), map[string]any{"path": parseRouteCtx.Path})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, parseErr), nil
		}
		if parsePageData == nil {
			debugLog("route.overlay.page.missing", map[string]any{"path": parseRouteCtx.Path, "requestURL": parsePageRequestURL})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, nil), nil
		}
		parseOverlayRequestURL := atlas.StartupRequestURL(parseRouteCtx.Path, routeDataQuery(parseFullQuery))
		parseOverlayData, parseErr := fetchRequestData(parseLoadCtx, parseOverlayRequestURL, "overlay")
		if parseErr != nil {
			errorLog("route.overlay.data.error", parseErr.Error(), map[string]any{"path": parseRouteCtx.Path, "requestURL": parseOverlayRequestURL})
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
			errorLog("route.nested.page.error", parseErr.Error(), map[string]any{"path": parseRouteCtx.Path})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, parseErr), nil
		}
		if parsePageData == nil {
			debugLog("route.nested.page.missing", map[string]any{"path": parseRouteCtx.Path, "requestURL": parsePageRequestURL})
			return buildRecoveryPayload(parseRouteCtx.Path, parseFullQuery, parsePageRequestURL, nil), nil
		}
		parseItemRequestURL := atlas.StartupRequestURL(parseRouteCtx.Path, routeDataQuery(parseFullQuery))
		parseItemData, parseErr := fetchRequestData(parseLoadCtx, parseItemRequestURL, "item")
		if parseErr != nil {
			errorLog("route.nested.item.error", parseErr.Error(), map[string]any{"path": parseRouteCtx.Path, "requestURL": parseItemRequestURL})
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

// atlasSkeletonBar is one placeholder block for the route loading state.
//
// Styled inline rather than with .atlas-* classes on purpose: the Atlas design
// tokens live in an inline <style> in client/atlas-commerce-os.html and are absent
// from the two stylesheets the server links, so a class-styled skeleton renders as
// nothing on the served page. A loading state you cannot see is the same defect as
// no loading state.
func atlasSkeletonBar(parseWidth string, parseHeight string) ui.Node {
	return html.Div(html.Props{
		Class: "atlas-boot-bar",
		Style: map[string]string{
			"width":         parseWidth,
			"height":        parseHeight,
			"margin-bottom": "0.75rem",
			"border-radius": "0.5rem",
			"background":    "currentColor",
			"opacity":       "0.5",
		},
	})
}

// atlasLoadingElement is the element the router shows while a route loader is in
// flight. It used to be html.Div(Class: "min-h-screen") — a full-height EMPTY div,
// which is a blank page with extra steps.
//
// WHY THIS IS THE RIGHT PLACE FOR A LOADING STATE IN GWC:
//
// router.Options.Loading is rendered by the router itself while
// ensureLoaderResult reports pending, and swapped for the route component the
// moment the loader resolves (router/router_render.go). That means no manual
// show/hide, no timer to cancel, and no window where both are mounted — the three
// ways a hand-rolled spinner leaks. Anything imperative here (a setTimeout that
// pokes innerHTML, a global "isLoading" flag) would have to be torn down by the
// same code that just failed, which is how loading states get stuck on screen.
//
// It does NOT cover the pre-hydration window. On the server-rendered shell,
// nothing at all is inside #app until the 19 MB wasm downloads and instantiates
// (measured: first content at ~1.3 s), and no client-side code can paint before
// its own binary exists. That placeholder has to ship in the server's HTML; see
// the comment in main() and the pattern in client/atlas-commerce-os.html.
func atlasLoadingElement() *router.Element {
	return ui.CreateElement(func() ui.Node {
		return html.Div(html.Props{
			Class: "min-h-screen",
			Role:  "status",
			// aria-busy tells assistive tech the region is mid-update, and
			// aria-live announces the label once. Without them a screen reader
			// hears silence for the whole load, which is this ticket's theme
			// again in a different medium.
			Aria:  map[string]string{"busy": "true", "live": "polite"},
			Data:  map[string]string{"atlas-loading": "route"},
			Style: map[string]string{"padding": "2.5rem 1.25rem", "max-width": "52rem", "margin": "0 auto"},
		},
			html.P(html.Props{Style: map[string]string{
				"margin":         "0 0 1.25rem 0",
				"font-size":      "0.78rem",
				"font-weight":    "700",
				"letter-spacing": "0.22em",
				"text-transform": "uppercase",
				"opacity":        "0.7",
			}}, ui.Text("Loading Atlas…")),
			atlasSkeletonBar("38%", "1.75rem"),
			atlasSkeletonBar("68%", "0.9rem"),
			atlasSkeletonBar("54%", "0.9rem"),
			atlasSkeletonBar("100%", "9rem"),
		)
	})
}

// atlasRouteErrorElement renders a route loader failure.
//
// Without router.Options.Error the router falls back to
// runtime.Div(nil, runtime.Text(err.Error())) — an unstyled bare div with a raw Go
// error string in it (router/router_state.go:renderRouteError). On the served
// shell that is a line of unformatted text on an unstyled page, which reads as a
// broken render rather than as a reported failure.
//
// Most Atlas loaders never reach here because loadRoutePayload converts fetch
// failures into a "recovery" payload and returns a nil error, but the overlay and
// nested loaders (atlasThresholdHistoryOverlayLoader, atlasWarehouseItemNestedLoader)
// do propagate errors, and fetch.LoadCached can fail on its own.
func atlasRouteErrorElement(parseAttrs router.Attrs) *router.Element {
	parseMessage, _ := parseAttrs["error"].(string)
	parsePath, _ := parseAttrs["path"].(string)
	if strings.TrimSpace(parseMessage) == "" {
		parseMessage = "The route loader failed without reporting a reason."
	}
	errorLog("route.error.surface", parseMessage, map[string]any{"path": parsePath})
	return ui.CreateElement(func() ui.Node {
		return html.Div(html.Props{
			Role:  "alert",
			Data:  map[string]string{"atlas-route-error": "true"},
			Style: map[string]string{"padding": "2.5rem 1.25rem", "max-width": "52rem", "margin": "0 auto"},
		},
			html.P(html.Props{Style: map[string]string{"margin": "0 0 0.5rem 0", "font-size": "1.15rem", "font-weight": "700"}},
				ui.Text("Atlas could not load this route")),
			html.P(html.Props{Style: map[string]string{"margin": "0 0 0.75rem 0", "opacity": "0.85"}},
				ui.Text("The page shell is still running, so navigating elsewhere should work.")),
			html.Pre(html.Props{Style: map[string]string{
				"margin":        "0",
				"padding":       "0.7rem 0.85rem",
				"border-radius": "0.5rem",
				"background":    "rgba(127,29,29,0.18)",
				"font-size":     "0.82rem",
				"white-space":   "pre-wrap",
				"word-break":    "break-word",
			}}, ui.Text(strings.TrimSpace(parsePath+"\n"+parseMessage))),
		)
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
		// Every route that can load can fail to load, so the error state is
		// registered from the same branch as the loader. Registering them apart is
		// how a route ends up with a loader and no error surface.
		parseOptions.Error = atlasRouteErrorElement
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

// atlasMountSelector is the one place the mount point is named. It is a CSS
// SELECTOR, not an id: ui.Render(..., "root") in a sibling example passed a bare
// id, matched nothing, and hung — defect 1 in the M10 write-up.
const atlasMountSelector = "#app"

// atlasHydrationAuditDelayMS is how long the client waits before declaring an
// empty mount point a failure. Measured baseline on localhost: first content at
// ~1.3 s from navigation (19 MB wasm plus the first route fetch). 3 s leaves
// headroom for a slower machine without leaving a blank page unexplained.
const atlasHydrationAuditDelayMS = 3000

func main() {
	// Order matters: the failure surface goes in before anything that can fail.
	// Installing it after the risky work means the one boot that needed it is the
	// one boot that did not have it.
	installAtlasBootSurface()
	installRuntimePanicCapture()
	installGoLivenessProbe()
	markAtlasBootState(atlasBootStateBooting)

	bootAtlasClient()

	// Park the runtime whatever happened during boot.
	//
	// Returning from main() exits the Go runtime with code 0, and after that every
	// registered callback throws "Go program has already exited". A contained
	// panic leaves a LIVE runtime with a partly rendered page, which is worth
	// keeping: navigation, anchors, and the rest of the shell still work. Exiting
	// here would convert a degraded page into a dead one on purpose.
	select {}
}

// bootAtlasClient runs the boot sequence behind a recover().
//
// WHY recover() AT THE BOOT BOUNDARY RATHER THAN TRUSTING THE RUNTIME:
//
// A hook-rule violation panics by design — the runtime would rather stop than keep
// mutating state it cannot reason about. That is correct, but an unrecovered panic
// in a wasm main() exits the Go runtime with code 2, and the DOM keeps whatever it
// had: an empty #app. The user sees a blank page and the diagnosis sits in a
// console nobody opened. Atlas shipped in exactly that state for four months.
//
// So: catch it here, and put the failure where a human will actually look.
//
// AND WHY recover() IS NOT ENOUGH ON ITS OWN:
//
// Panics raised INSIDE the GWC runtime never reach this recover(). ui's wasm
// initialisation sets HideRawPanicOutput: true, so internal/runtime formats the
// report, emits it, and returns normally instead of re-panicking. ui.Hydrate then
// returns a nil error and the DOM stays empty. That is why boot has three
// independent guards and not one:
//
//   - this recover(), for panics in Atlas's own boot code (bootstrap decode, a
//     js.Value type assertion, a nil map);
//   - installRuntimePanicCapture, for contained runtime panics, which arrive as a
//     gwc:runtime-panic CustomEvent and nowhere else in-page;
//   - scheduleHydrationOutputAudit, for the residual case where nothing panicked
//     anywhere and the mount point is simply still empty.
func bootAtlasClient() {
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			return
		}
		surfaceAtlasFailure(atlasSurfaceModeFatal, "boot-panic",
			"Atlas failed to start",
			"The client panicked while booting, so no part of the app mounted. This is a defect in the client, not a network or data problem.",
			fmt.Sprintf("%v", parseRecovered))
	}()

	// Styling boundary, and it comes before hydration for two reasons.
	//
	// css.SeedFromDocument() FIRST: it reads the class list out of the server's
	// <style data-gwc-css="…"> block and marks every one of those rules as already
	// present. Without it, design.Install() below would re-emit the whole global
	// layer into a second <style> element — a duplicate reset, a duplicate :root token
	// block, and a duplicate print layer, all winning ties over the server's copy by
	// emission order. Nothing would look broken, which is what makes it worth a
	// comment; it would just quietly double the CSS on every page.
	//
	// design.Install() SECOND, and unconditionally: on a served page it is a no-op
	// after seeding, and on client/atlas-commerce-os.html — a static shell with no
	// server behind it and therefore no <style> block to seed from — it is the only
	// thing that emits the base layer at all. One call covers both.
	//
	// Both run before ui.Hydrate, because primitives folded during the first render
	// must land AFTER the reset and the token block, and the css package emits in
	// call order.
	css.SeedFromDocument()
	design.Install()

	// Hydration boundary: resume from the exact server-emitted bootstrap payload so first paint metadata,
	// document language, and route data do not drift between SSR and client ownership.
	initialBootstrap = loadBootstrap()
	initialPayload = atlas.PayloadFromSSRBootstrap(initialBootstrap)
	logHydrationFallbackConditions(initialPayload)
	logHydrationDocumentMismatch(initialPayload)
	if parseErr := restoreAtlasFetchCacheBootstrap(initialPayload); parseErr != nil {
		errorLog("bootstrap.cache.restore.failed", parseErr.Error(), nil)
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

	debugLog("hydrate.start", map[string]any{"selector": atlasMountSelector, "path": initialPayload.Route.Path})

	// Check the mount point BEFORE handing it to the runtime.
	//
	// runtime.HydrateTo does check, and its report is excellent — it names the
	// selector and the phase. But under crash containment it does not stop: the
	// contained panic returns, Hydrate proceeds with a null container, and the
	// caller gets a nil error. So the only way for the client to react to a
	// missing mount point is to have looked itself.
	if !atlasMountPointExists(atlasMountSelector) {
		surfaceAtlasFailure(atlasSurfaceModeFatal, "mount-missing",
			"Atlas has nowhere to render",
			"The page did not contain an element matching "+atlasMountSelector+", so hydration had no target. Check that the server shell still emits it.",
			"document.querySelector(\""+atlasMountSelector+"\") returned null")
		return
	}

	// Hand atlas.App to the runtime as a component plus props instead of invoking it here: its body runs
	// hooks (useAtlasAtom/state.UseAtom), which are only legal inside a render pass owned by a fiber.
	_, parseHydrateErr := ui.Hydrate(ui.CreateElement(atlas.App, initialPayload), atlasMountSelector, ui.HydrationOptions{Bootstrap: initialBootstrap})
	if parseHydrateErr != nil {
		// This return value used to be discarded with `_, _ =`. It reports bootstrap
		// decode failures and atom-snapshot restore failures — i.e. exactly the
		// cases where the app would mount against state that is not the state the
		// server rendered.
		surfaceAtlasFailure(atlasSurfaceModeFatal, "hydrate-error",
			"Atlas could not resume the server's render",
			"Hydration returned an error, so client state does not match the HTML the server sent. Continuing would render a page built from half-restored state.",
			parseHydrateErr.Error())
		return
	}
	debugLog("hydrate.done", map[string]any{"selector": atlasMountSelector})
	bindDirtyGuardForms()
	registerBeforeUnloadGuard()
	parseR.HydrateMount(atlasMountSelector)
	debugLog("router.mount", map[string]any{"selector": atlasMountSelector})
	registerAnchorNavigation(parseR)
	registerOverlayDebugTracking()

	// Hydration returned clean. That is not the same as "rendered something", so
	// the state recorded here is a milestone and not a verdict; the audit below is
	// the verdict.
	markAtlasBootState(atlasBootStateHydrated)

	// Tear down the pre-hydration placeholder now that the client owns the screen.
	//
	// NOTE — a gap this client cannot close from here: the server shell
	// (server/server.go renderPageStatusWithPayload) emits a bare
	// `<div id="app"></div>` with no placeholder to tear down, so on served pages
	// this is a no-op and the page really is blank for the whole ~1.3 s boot. The
	// pattern that fixes it is in client/atlas-commerce-os.html: ship the
	// placeholder as a SIBLING of the mount point (inside it, hydration would
	// diff against it and report a mismatch) and let the client remove it here.
	clearAtlasBootPlaceholder()

	// Last guard, and the one that would have caught the four-month blank page:
	// nothing above can tell the difference between "hydrated" and "hydrated and
	// visible", so ask the DOM once the first commit has had time to land.
	scheduleHydrationOutputAudit(atlasMountSelector, atlasHydrationAuditDelayMS)
}
