//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	atlasNoticeQueryKey             = "atlas_notice"
	atlasBootstrapModeQueryKey      = "atlas_bootstrap"
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
var lastRenderedRouteKey string
var hasRenderedRoute bool
var routePayloadCache = map[string]atlas.Payload{}
var requestCache = map[string]atlas.Request{}

func debugLog(event string, details map[string]any) {
	console := js.Global().Get("console")
	if !console.Truthy() || !console.Get("log").Truthy() {
		return
	}
	if details == nil {
		details = map[string]any{}
	}
	details["event"] = event
	console.Call("log", "[atlas-wasm]", details)
	window := js.Global().Get("window")
	if window.Truthy() {
		window.Set("__atlasDebugLast", js.ValueOf(details))
	}
}

func loadBootstrap() ui.SSRBootstrap {
	bootstrap, err := ui.ReadBootstrapScript(atlasBootstrapScriptID)
	if err == nil {
		debugLog("bootstrap.read.ok", map[string]any{
			"path":      bootstrap.Route.Path,
			"queryKeys": len(bootstrap.Route.Query),
			"paramKeys": len(bootstrap.Route.Params),
			"mode":      "inline",
		})
		return bootstrap
	}
	ref, refErr := ui.ReadBootstrapReferenceScript(atlasBootstrapReferenceScriptID)
	if refErr != nil {
		debugLog("bootstrap.read.failed", map[string]any{"error": err.Error(), "referenceError": refErr.Error()})
		return ui.SSRBootstrap{}
	}
	bootstrap, refErr = ui.ReadBootstrapReference(ref)
	if refErr != nil {
		debugLog("bootstrap.reference.failed", map[string]any{"url": ref.URL, "error": refErr.Error()})
		return ui.SSRBootstrap{}
	}
	debugLog("bootstrap.read.ok", map[string]any{
		"path":      bootstrap.Route.Path,
		"queryKeys": len(bootstrap.Route.Query),
		"paramKeys": len(bootstrap.Route.Params),
		"mode":      "external",
		"url":       ref.URL,
	})
	return bootstrap
}

func cloneQueryValues(values url.Values) map[string][]string {
	cloned := make(map[string][]string, len(values))
	for key, item := range values {
		cloned[key] = append([]string(nil), item...)
	}
	return cloned
}

func cloneQueryMap(input map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(input))
	for key, item := range input {
		cloned[key] = append([]string(nil), item...)
	}
	return cloned
}

func cloneStringMap(input map[string]string) map[string]string {
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneDataMap(input map[string]any) map[string]any {
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneRequests(input map[string]atlas.Request) map[string]atlas.Request {
	cloned := make(map[string]atlas.Request, len(input))
	for key, request := range input {
		cloned[key] = atlas.Request{
			Method: request.Method,
			URL:    request.URL,
			Status: request.Status,
			Data:   cloneDataMap(request.Data),
		}
	}
	return cloned
}

func clonePayload(input atlas.Payload) atlas.Payload {
	payload := input
	payload.Route.Query = cloneQueryMap(input.Route.Query)
	payload.Route.Params = cloneStringMap(input.Route.Params)
	payload.Data = cloneDataMap(input.Data)
	payload.Requests = cloneRequests(input.Requests)
	payload.SavedViews = append([]atlas.SavedViewPayload(nil), input.SavedViews...)
	if input.User != nil {
		user := *input.User
		payload.User = &user
	}
	return payload
}

func queryValuesFromMap(input map[string][]string) url.Values {
	values := url.Values{}
	for key, items := range input {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return values
}

func decodeJSONBody(response *http.Response, target any) error {
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(target)
}

func routeDataQuery(query url.Values) url.Values {
	filtered := url.Values{}
	for key, items := range query {
		trimmedKey := strings.TrimSpace(key)
		if strings.EqualFold(trimmedKey, atlasNoticeQueryKey) || strings.EqualFold(trimmedKey, atlasBootstrapModeQueryKey) {
			continue
		}
		for _, item := range items {
			filtered.Add(key, item)
		}
	}
	return filtered
}

func routeKey(path string, query map[string][]string) string {
	values := url.Values{}
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	encoded := values.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func routeDataKey(path string, query url.Values) string {
	return routeKey(path, cloneQueryValues(routeDataQuery(query)))
}

func pathWithQuery(path string, query url.Values) string {
	encoded := query.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func startupRequestForPayload(payload atlas.Payload) (atlas.Request, bool) {
	request, ok := atlas.StartupRequest(payload, "page")
	if !ok {
		return atlas.Request{}, false
	}
	return request, true
}

func cachePayload(payload atlas.Payload) {
	dataKey := routeDataKey(payload.Route.Path, queryValuesFromMap(payload.Route.Query))
	routePayloadCache[dataKey] = clonePayload(payload)
	if request, ok := startupRequestForPayload(payload); ok && strings.TrimSpace(request.URL) != "" {
		requestCache[strings.TrimSpace(request.URL)] = atlas.Request{
			Method: request.Method,
			URL:    request.URL,
			Status: request.Status,
			Data:   cloneDataMap(request.Data),
		}
	}
}

func cachedPayloadForRoute(path string, query url.Values) (atlas.Payload, bool) {
	payload, ok := routePayloadCache[routeDataKey(path, query)]
	if !ok {
		return atlas.Payload{}, false
	}
	return clonePayload(payload), true
}

func fetchPageData(ctx context.Context, path string, query url.Values) (any, string, error) {
	requestURL := atlas.StartupRequestURL(path, routeDataQuery(query))
	debugLog("route.fetch.start", map[string]any{"path": path, "query": query.Encode(), "requestURL": requestURL})
	if strings.TrimSpace(requestURL) == "" {
		debugLog("route.fetch.skipped", map[string]any{"path": path, "reason": "empty request url"})
		return nil, "", nil
	}
	data, err := fetchRequestData(ctx, requestURL, "page")
	if err != nil {
		return nil, requestURL, err
	}
	return data, requestURL, nil
}

func fetchRequestData(ctx context.Context, requestURL string, dataKey string) (any, error) {
	if cached, ok := requestCache[strings.TrimSpace(requestURL)]; ok {
		debugLog("route.fetch.cache.hit", map[string]any{"requestURL": requestURL, "dataKey": dataKey})
		return cached.Data[dataKey], nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		debugLog("route.fetch.request.error", map[string]any{"requestURL": requestURL, "dataKey": dataKey, "error": err.Error()})
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		debugLog("route.fetch.network.error", map[string]any{"requestURL": requestURL, "dataKey": dataKey, "error": err.Error()})
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		debugLog("route.fetch.non_ok", map[string]any{"requestURL": requestURL, "dataKey": dataKey, "status": response.StatusCode})
		response.Body.Close()
		return nil, nil
	}
	var payloadData any
	if err := decodeJSONBody(response, &payloadData); err != nil {
		debugLog("route.fetch.decode.error", map[string]any{"requestURL": requestURL, "dataKey": dataKey, "error": err.Error()})
		return nil, err
	}
	requestCache[strings.TrimSpace(requestURL)] = atlas.Request{
		Method: http.MethodGet,
		URL:    requestURL,
		Status: http.StatusOK,
		Data:   map[string]any{dataKey: payloadData},
	}
	debugLog("route.fetch.ok", map[string]any{"requestURL": requestURL, "dataKey": dataKey})
	return payloadData, nil
}

func routeSurface(path string) string {
	if path == atlas.RouteAppRoot || strings.HasPrefix(path, "/app/") {
		return "internal"
	}
	return "public"
}

func routeScreen(path string) string {
	switch {
	case strings.HasPrefix(path, atlas.RouteInventory+"/") && strings.HasSuffix(path, "/threshold-history"):
		return "sku-threshold-history"
	case path == atlas.RouteLanding:
		return "landing"
	case path == atlas.RouteCatalog:
		return "catalog"
	case strings.HasPrefix(path, atlas.RouteCatalog+"/"):
		return "product"
	case path == atlas.RouteWarehouses:
		return "warehouses"
	case strings.Contains(path, "/availability/"):
		return "warehouse-availability"
	case strings.HasPrefix(path, atlas.RouteWarehouses+"/"):
		return "warehouse-detail"
	case path == atlas.RouteAppRoot || path == atlas.RouteDashboard:
		return "dashboard"
	case path == "/app/products":
		return "products"
	case strings.HasPrefix(path, "/app/products/"):
		return "product-editor"
	case path == atlas.RouteInventory:
		return "inventory"
	case strings.HasPrefix(path, atlas.RouteInventory+"/"):
		return "sku-detail"
	case path == atlas.RouteWarehouseOps:
		return "warehouse-ops"
	case strings.HasPrefix(path, atlas.RouteWarehouseOps+"/") && strings.Contains(path, "/items/"):
		return "warehouse-item-detail"
	case strings.HasPrefix(path, atlas.RouteWarehouseOps+"/"):
		return "warehouse-detail"
	case path == atlas.RouteTransfers:
		return "transfers"
	case strings.HasPrefix(path, atlas.RouteTransfers+"/"):
		return "transfer-detail"
	case path == atlas.RoutePurchaseOrders:
		return "purchase-orders"
	case strings.HasPrefix(path, atlas.RoutePurchaseOrders+"/"):
		return "purchase-order-detail"
	case path == atlas.RouteReceiving:
		return "receiving"
	case strings.HasPrefix(path, atlas.RouteReceiving+"/"):
		return "receiving-session-detail"
	case path == atlas.RouteComments:
		return "comments"
	case path == atlas.RouteSettings:
		return "settings"
	default:
		return "recovery"
	}
}

func decodeMapValue[T any](input any) (T, bool) {
	var target T
	encoded, err := json.Marshal(input)
	if err != nil {
		return target, false
	}
	if err := json.Unmarshal(encoded, &target); err != nil {
		return target, false
	}
	return target, true
}

func routeTitle(path string, pageData any) string {
	metadata := atlas.MetadataForPath(path)
	switch {
	case strings.HasPrefix(path, atlas.RouteCatalog+"/"):
		if product, ok := decodeMapValue[struct {
			Title   string `json:"title"`
			Product struct {
				Title string `json:"title"`
			} `json:"product"`
		}](pageData); ok && strings.TrimSpace(product.Title) != "" {
			return "Atlas " + product.Title
		} else if ok && strings.TrimSpace(product.Product.Title) != "" {
			return "Atlas " + product.Product.Title
		}
	case strings.HasPrefix(path, atlas.RouteWarehouses+"/") && !strings.Contains(path, "/availability/"):
		return "Atlas Warehouse Detail"
	case strings.HasPrefix(path, atlas.RouteWarehouseOps+"/") && strings.Contains(path, "/items/"):
		if itemPage, ok := decodeMapValue[struct {
			Item struct {
				Title string `json:"title"`
			} `json:"item"`
			Product struct {
				Title string `json:"title"`
			} `json:"product"`
		}](pageData); ok {
			if strings.TrimSpace(itemPage.Product.Title) != "" {
				return "Atlas " + itemPage.Product.Title
			}
			if strings.TrimSpace(itemPage.Item.Title) != "" {
				return "Atlas " + itemPage.Item.Title
			}
		}
	}
	return metadata.Title
}

func routeDescription(path string, pageData any) string {
	metadata := atlas.MetadataForPath(path)
	switch {
	case strings.HasPrefix(path, atlas.RouteCatalog+"/"):
		if product, ok := decodeMapValue[struct {
			SEODescription string `json:"seoDescription"`
			Summary        string `json:"summary"`
			Product        struct {
				SEODescription string `json:"seoDescription"`
				Summary        string `json:"summary"`
			} `json:"product"`
		}](pageData); ok {
			if strings.TrimSpace(product.SEODescription) != "" {
				return product.SEODescription
			}
			if strings.TrimSpace(product.Summary) != "" {
				return product.Summary
			}
			if strings.TrimSpace(product.Product.SEODescription) != "" {
				return product.Product.SEODescription
			}
			if strings.TrimSpace(product.Product.Summary) != "" {
				return product.Product.Summary
			}
		}
	case strings.HasPrefix(path, atlas.RouteWarehouses+"/") && !strings.Contains(path, "/availability/"):
		if warehousePage, ok := decodeMapValue[struct {
			Warehouse struct {
				PublicSummary string `json:"publicSummary"`
			} `json:"warehouse"`
		}](pageData); ok && strings.TrimSpace(warehousePage.Warehouse.PublicSummary) != "" {
			return warehousePage.Warehouse.PublicSummary
		}
	}
	return metadata.Description
}

func routeCanonical(path string) string {
	return path
}

func ensureManagedTitle(document js.Value) js.Value {
	title := document.Call("querySelector", `title[`+atlasManagedAttr+`="true"]`)
	if title.Truthy() {
		return title
	}
	title = document.Call("querySelector", "title")
	if title.Truthy() {
		title.Call("setAttribute", atlasManagedAttr, "true")
		return title
	}
	head := document.Get("head")
	if !head.Truthy() {
		return js.Undefined()
	}
	title = document.Call("createElement", "title")
	title.Call("setAttribute", atlasManagedAttr, "true")
	head.Call("appendChild", title)
	return title
}

func ensureManagedHeadNode(document js.Value, selector string, tag string, init func(js.Value)) js.Value {
	node := document.Call("querySelector", selector+`[`+atlasManagedAttr+`="true"]`)
	if node.Truthy() {
		return node
	}
	node = document.Call("querySelector", selector)
	if node.Truthy() {
		node.Call("setAttribute", atlasManagedAttr, "true")
		return node
	}
	head := document.Get("head")
	if !head.Truthy() {
		return js.Undefined()
	}
	node = document.Call("createElement", tag)
	if init != nil {
		init(node)
	}
	node.Call("setAttribute", atlasManagedAttr, "true")
	head.Call("appendChild", node)
	return node
}

func updateDocumentMetadata(payload atlas.Payload) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	debugLog("metadata.update", map[string]any{
		"path":        payload.Route.Path,
		"title":       payload.Route.Title,
		"description": payload.Route.Description,
		"canonical":   payload.Route.Canonical,
	})
	if strings.TrimSpace(payload.Route.Title) != "" {
		title := ensureManagedTitle(document)
		if title.Truthy() {
			title.Set("textContent", payload.Route.Title)
		}
		document.Set("title", payload.Route.Title)
	}
	if description := strings.TrimSpace(payload.Route.Description); description != "" {
		meta := ensureManagedHeadNode(document, `meta[name="description"]`, "meta", func(node js.Value) {
			node.Call("setAttribute", "name", "description")
		})
		if meta.Truthy() {
			meta.Call("setAttribute", "content", description)
		}
	}
	if canonicalValue := strings.TrimSpace(payload.Route.Canonical); canonicalValue != "" {
		canonical := ensureManagedHeadNode(document, `link[rel="canonical"]`, "link", func(node js.Value) {
			node.Call("setAttribute", "rel", "canonical")
		})
		if canonical.Truthy() {
			canonical.Call("setAttribute", "href", canonicalValue)
		}
	}
}

func syncNavigationPosition(payload atlas.Payload) {
	currentKey := routeKey(payload.Route.Path, payload.Route.Query)
	if !hasRenderedRoute {
		hasRenderedRoute = true
		lastRenderedRouteKey = currentKey
		return
	}
	if currentKey == lastRenderedRouteKey {
		return
	}
	lastRenderedRouteKey = currentKey
	window := js.Global().Get("window")
	if window.Truthy() && window.Get("scrollTo").Truthy() {
		window.Call("scrollTo", 0, 0)
	}
	debugLog("navigation.scroll.reset", map[string]any{"path": payload.Route.Path, "key": currentKey})
}

func buildPayload(path string, query url.Values, pageData any, requestURL string) atlas.Payload {
	data := map[string]any{"page": pageData}
	requests := map[string]atlas.Request{}
	if strings.TrimSpace(requestURL) != "" {
		requests["page"] = atlas.Request{
			Method: http.MethodGet,
			URL:    requestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"page": pageData},
		}
	}
	return buildPayloadWithData(path, query, data, requests)
}

func buildPayloadWithData(path string, query url.Values, data map[string]any, requests map[string]atlas.Request) atlas.Payload {
	pageData := data["page"]
	payload := initialPayload
	payload.Route = atlas.RouteBootstrap{
		Path:        path,
		Query:       cloneQueryValues(query),
		Params:      map[string]string{},
		Surface:     routeSurface(path),
		Screen:      routeScreen(path),
		Title:       routeTitle(path, pageData),
		Description: routeDescription(path, pageData),
		Canonical:   routeCanonical(path),
	}
	payload.Data = cloneDataMap(data)
	payload.Requests = cloneRequests(requests)
	debugLog("payload.build", map[string]any{
		"path":        path,
		"screen":      payload.Route.Screen,
		"surface":     payload.Route.Surface,
		"requestKeys": len(payload.Requests),
	})
	return payload
}

func routeRecoveryFallback(path string) (string, string) {
	if routeSurface(path) == "internal" {
		return atlas.RouteDashboard, "Back to dashboard"
	}
	return atlas.RouteLanding, "Back to Atlas"
}

func routeRecoveryCopy(path string) (string, string) {
	if routeSurface(path) == "internal" {
		return "Atlas could not load this admin screen right now.", "The page shell is still active, so you can recover without losing the rest of the session."
	}
	return "Atlas could not load this page right now.", "The page shell is still active, so you can recover without a full crash."
}

func buildRecoveryPayload(path string, query url.Values, requestURL string, err error) atlas.Payload {
	metadata := atlas.MetadataForPath(path)
	title := strings.TrimSpace(metadata.Title)
	if title == "" {
		title = "Atlas"
	}
	recoveryHref, recoveryLabel := routeRecoveryFallback(path)
	message, summary := routeRecoveryCopy(path)
	detailParts := []string{summary}
	if strings.TrimSpace(requestURL) != "" {
		detailParts = append(detailParts, fmt.Sprintf("Request: %s", requestURL))
	}
	if err != nil {
		detailParts = append(detailParts, "Technical detail: "+err.Error())
	}
	payload := initialPayload
	payload.Route = atlas.RouteBootstrap{
		Path:        path,
		Query:       cloneQueryValues(query),
		Params:      map[string]string{},
		Surface:     routeSurface(path),
		Screen:      "recovery",
		Title:       metadata.Title + " Unavailable",
		Description: metadata.Description,
		Canonical:   routeCanonical(path),
	}
	payload.Data = map[string]any{"page": map[string]any{
		"title":         title + " unavailable",
		"message":       message,
		"recoveryHref":  recoveryHref,
		"recoveryLabel": recoveryLabel,
		"detail":        strings.Join(detailParts, " "),
	}}
	payload.Requests = map[string]atlas.Request{}
	if strings.TrimSpace(requestURL) != "" {
		status := http.StatusServiceUnavailable
		if err == nil {
			status = http.StatusBadGateway
		}
		payload.Requests["page"] = atlas.Request{
			Method: http.MethodGet,
			URL:    requestURL,
			Status: status,
		}
	}
	debugLog("payload.recovery", map[string]any{
		"path":       path,
		"screen":     payload.Route.Screen,
		"requestURL": requestURL,
		"error": func() string {
			if err == nil {
				return ""
			}
			return err.Error()
		}(),
	})
	return payload
}

func loadRoutePayload(ctx context.Context, routeCtx router.RouteContext) (atlas.Payload, error) {
	debugLog("route.load.start", map[string]any{
		"path":  routeCtx.Path,
		"query": routeCtx.Query.Values().Encode(),
	})
	fullQuery := routeCtx.Query.Values()
	if cached, ok := cachedPayloadForRoute(routeCtx.Path, fullQuery); ok {
		cached.Route.Query = cloneQueryValues(fullQuery)
		debugLog("route.load.cache.hit", map[string]any{"path": routeCtx.Path})
		return cached, nil
	}
	pageData, requestURL, err := fetchPageData(ctx, routeCtx.Path, fullQuery)
	if err != nil {
		debugLog("route.load.error", map[string]any{"path": routeCtx.Path, "error": err.Error()})
		return buildRecoveryPayload(routeCtx.Path, fullQuery, requestURL, err), nil
	}
	if pageData == nil && strings.TrimSpace(requestURL) == "" && routeCtx.Path == atlas.RouteLanding {
		debugLog("route.load.static", map[string]any{"path": routeCtx.Path, "reason": "static landing route"})
		payload := buildPayload(routeCtx.Path, fullQuery, map[string]any{}, "")
		cachePayload(payload)
		return payload, nil
	}
	if pageData == nil {
		debugLog("route.load.missing", map[string]any{"path": routeCtx.Path, "requestURL": requestURL})
		payload := buildRecoveryPayload(routeCtx.Path, fullQuery, requestURL, nil)
		cachePayload(payload)
		return payload, nil
	}
	payload := buildPayload(routeCtx.Path, fullQuery, pageData, requestURL)
	cachePayload(payload)
	debugLog("route.load.ok", map[string]any{"path": routeCtx.Path, "title": payload.Route.Title})
	return payload, nil
}

func serializeGuardedForm(form js.Value) string {
	fields := form.Call("querySelectorAll", "input, textarea, select")
	parts := make([]string, 0, fields.Get("length").Int())
	for i := 0; i < fields.Get("length").Int(); i++ {
		field := fields.Index(i)
		if !field.Truthy() || field.Get("disabled").Bool() {
			continue
		}
		name := strings.TrimSpace(field.Get("name").String())
		if name == "" {
			continue
		}
		fieldType := strings.ToLower(strings.TrimSpace(field.Get("type").String()))
		switch fieldType {
		case "submit", "button", "reset", "file":
			continue
		}
		value := field.Get("value").String()
		if fieldType == "checkbox" || fieldType == "radio" {
			if field.Get("checked").Bool() {
				value = "1"
			} else {
				value = "0"
			}
		}
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, "\x1f")
}

func refreshDirtyGuardState(form js.Value) {
	initial := form.Get("__atlasDirtyGuardInitial").String()
	current := serializeGuardedForm(form)
	form.Set("__atlasDirtyGuardDirty", current != initial)
}

func bindDirtyGuardForms() {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	forms := document.Call("querySelectorAll", "form["+atlasDirtyGuardAttr+"]")
	for i := 0; i < forms.Get("length").Int(); i++ {
		form := forms.Index(i)
		if !form.Truthy() {
			continue
		}
		if form.Get("__atlasDirtyGuardBound").Truthy() && form.Get("__atlasDirtyGuardBound").Bool() {
			refreshDirtyGuardState(form)
			continue
		}
		form.Set("__atlasDirtyGuardInitial", serializeGuardedForm(form))
		form.Set("__atlasDirtyGuardDirty", false)
		form.Set("__atlasDirtyGuardSubmitting", false)

		updateHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			form.Set("__atlasDirtyGuardSubmitting", false)
			refreshDirtyGuardState(form)
			return nil
		})
		submitHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			form.Set("__atlasDirtyGuardSubmitting", true)
			form.Set("__atlasDirtyGuardDirty", false)
			return nil
		})
		form.Call("addEventListener", "input", updateHandler)
		form.Call("addEventListener", "change", updateHandler)
		form.Call("addEventListener", "submit", submitHandler)
		dirtyGuardHandlers = append(dirtyGuardHandlers, updateHandler, submitHandler)
		form.Set("__atlasDirtyGuardBound", true)
	}
}

func activeDirtyGuardMessage() string {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return ""
	}
	form := document.Call("querySelector", "form["+atlasDirtyGuardAttr+"]")
	if !form.Truthy() || form.Get("__atlasDirtyGuardSubmitting").Bool() {
		return ""
	}
	refreshDirtyGuardState(form)
	if !form.Get("__atlasDirtyGuardDirty").Bool() {
		return ""
	}
	message := strings.TrimSpace(form.Call("getAttribute", atlasDirtyMessageAttr).String())
	if message == "" {
		message = "Leave this route and discard unsaved Atlas changes?"
	}
	return message
}

func registerBeforeUnloadGuard() {
	window := js.Global().Get("window")
	if !window.Truthy() || !window.Get("addEventListener").Truthy() {
		return
	}
	beforeUnloadHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		message := activeDirtyGuardMessage()
		if message == "" {
			return nil
		}
		event := args[0]
		if event.Get("preventDefault").Truthy() {
			event.Call("preventDefault")
		}
		event.Set("returnValue", message)
		return message
	})
	window.Call("addEventListener", "beforeunload", beforeUnloadHandler)
}

func shouldUseDocumentNavigation(path string) bool {
	return strings.HasPrefix(path, "/auth/")
}

func navigateDocument(target string) {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	location := window.Get("location")
	if location.Truthy() && location.Get("assign").Truthy() {
		location.Call("assign", target)
	}
}

func requireInternalSessionGuard(ctx router.RouteContext) router.GuardResult {
	if initialPayload.User != nil {
		return router.AllowNavigation()
	}
	next := pathWithQuery(ctx.Path, ctx.Query.Values())
	values := url.Values{}
	values.Set("next", next)
	target := atlasMockSignInPath + "?" + values.Encode()
	debugLog("route.guard.auth.redirect", map[string]any{"path": ctx.Path, "target": target})
	navigateDocument(target)
	return router.BlockNavigation("Redirecting to mock sign in.")
}

func productEditorBeforeLeaveGuard(current router.RouteContext, next router.RouteContext) router.GuardResult {
	message := activeDirtyGuardMessage()
	if message == "" {
		return router.AllowNavigation()
	}
	window := js.Global().Get("window")
	if window.Truthy() && window.Get("confirm").Truthy() && window.Call("confirm", message).Bool() {
		return router.AllowNavigation()
	}
	return router.BlockNavigation(message)
}

func requestCachePrefixesForRoutePrefix(prefix string) []string {
	switch prefix {
	case "/shop":
		return []string{"/api/public/catalog", "/api/public/products"}
	case "/warehouses":
		return []string{"/api/public/warehouses"}
	case "/app/dashboard":
		return []string{"/api/app/dashboard"}
	case "/app/products":
		return []string{"/api/app/products"}
	case "/app/inventory":
		return []string{"/api/app/inventory"}
	case "/app/warehouses":
		return []string{"/api/app/warehouses"}
	case "/app/transfers":
		return []string{"/api/app/transfers"}
	case "/app/purchase-orders":
		return []string{"/api/app/purchase-orders"}
	case "/app/receiving":
		return []string{"/api/app/receiving"}
	case "/app/comments":
		return []string{"/api/app/comments"}
	case "/app/settings":
		return []string{"/api/app/settings", "/api/app/saved-views"}
	default:
		return nil
	}
}

func mutationRoutePrefixes(path string, notice string) []string {
	trimmed := strings.TrimSpace(strings.ToLower(notice))
	switch {
	case strings.HasPrefix(trimmed, "product-"):
		return []string{"/app/products", "/app/inventory", "/app/warehouses", "/shop"}
	case trimmed == "inventory-updated" || trimmed == "threshold-updated":
		return []string{"/app/inventory", "/app/warehouses", "/app/dashboard"}
	case strings.HasPrefix(trimmed, "purchase-order-"):
		return []string{"/app/purchase-orders", "/app/receiving", "/app/dashboard"}
	case trimmed == "receiving-reconciled":
		return []string{"/app/receiving", "/app/purchase-orders", "/app/dashboard"}
	case strings.Contains(trimmed, "comment"):
		return []string{"/app/comments", "/app/dashboard", "/shop"}
	case strings.HasPrefix(trimmed, "saved-view"):
		return []string{"/app/settings", "/app/inventory", "/app/dashboard"}
	case trimmed == "preferences-saved":
		return []string{"/app/settings", "/app/dashboard", "/app/inventory", "/app/warehouses"}
	case strings.HasPrefix(trimmed, "transfer-"):
		return []string{"/app/transfers", "/app/inventory", "/app/dashboard"}
	case strings.HasPrefix(trimmed, "quote-request") || strings.HasPrefix(trimmed, "restock-request"):
		return []string{"/shop", "/warehouses"}
	default:
		return []string{path}
	}
}

func invalidateCachesForPayload(payload atlas.Payload) {
	notice := strings.TrimSpace(queryValuesFromMap(payload.Route.Query).Get(atlasNoticeQueryKey))
	if notice == "" {
		return
	}
	currentDataKey := routeDataKey(payload.Route.Path, queryValuesFromMap(payload.Route.Query))
	currentRequestURL := ""
	if request, ok := startupRequestForPayload(payload); ok {
		currentRequestURL = strings.TrimSpace(request.URL)
	}
	routePrefixes := mutationRoutePrefixes(payload.Route.Path, notice)
	for key := range routePayloadCache {
		if key == currentDataKey {
			continue
		}
		for _, prefix := range routePrefixes {
			if strings.HasPrefix(key, prefix) {
				delete(routePayloadCache, key)
				break
			}
		}
	}
	requestPrefixes := []string{}
	for _, prefix := range routePrefixes {
		requestPrefixes = append(requestPrefixes, requestCachePrefixesForRoutePrefix(prefix)...)
	}
	for key := range requestCache {
		if key == currentRequestURL {
			continue
		}
		for _, prefix := range requestPrefixes {
			if strings.HasPrefix(key, prefix) {
				delete(requestCache, key)
				break
			}
		}
	}
	debugLog("route.cache.invalidate", map[string]any{
		"path":         payload.Route.Path,
		"notice":       notice,
		"routeTargets": routePrefixes,
	})
}

func atlasRouteLoader(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
	debugLog("router.loader", map[string]any{"path": routeCtx.Path})
	payload, err := loadRoutePayload(ctx, routeCtx)
	if err != nil {
		debugLog("router.loader.error", map[string]any{"path": routeCtx.Path, "error": err.Error()})
		return nil, err
	}
	return router.Attrs{"payload": payload}, nil
}

func loadInventoryThresholdHistoryPayload(ctx context.Context, routeCtx router.RouteContext) (atlas.Payload, error) {
	fullQuery := routeCtx.Query.Values()
	if cached, ok := cachedPayloadForRoute(routeCtx.Path, fullQuery); ok {
		cached.Route.Query = cloneQueryValues(fullQuery)
		debugLog("route.overlay.cache.hit", map[string]any{"path": routeCtx.Path})
		return cached, nil
	}
	sku := strings.TrimSpace(routeCtx.Params.Get("sku"))
	parentPath := atlas.RouteInventory + "/" + sku
	pageData, pageRequestURL, err := fetchPageData(ctx, parentPath, fullQuery)
	if err != nil {
		debugLog("route.overlay.page.error", map[string]any{"path": routeCtx.Path, "error": err.Error()})
		return buildRecoveryPayload(routeCtx.Path, fullQuery, pageRequestURL, err), nil
	}
	if pageData == nil {
		debugLog("route.overlay.page.missing", map[string]any{"path": routeCtx.Path, "requestURL": pageRequestURL})
		return buildRecoveryPayload(routeCtx.Path, fullQuery, pageRequestURL, nil), nil
	}
	overlayRequestURL := atlas.StartupRequestURL(routeCtx.Path, routeDataQuery(fullQuery))
	overlayData, err := fetchRequestData(ctx, overlayRequestURL, "overlay")
	if err != nil {
		debugLog("route.overlay.data.error", map[string]any{"path": routeCtx.Path, "requestURL": overlayRequestURL, "error": err.Error()})
		return buildRecoveryPayload(routeCtx.Path, fullQuery, overlayRequestURL, err), nil
	}
	requests := map[string]atlas.Request{}
	if strings.TrimSpace(pageRequestURL) != "" {
		requests["page"] = atlas.Request{
			Method: http.MethodGet,
			URL:    pageRequestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"page": pageData},
		}
	}
	if strings.TrimSpace(overlayRequestURL) != "" && overlayData != nil {
		requests["overlay"] = atlas.Request{
			Method: http.MethodGet,
			URL:    overlayRequestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"overlay": overlayData},
		}
	}
	payload := buildPayloadWithData(routeCtx.Path, fullQuery, map[string]any{
		"page":    pageData,
		"overlay": overlayData,
	}, requests)
	cachePayload(payload)
	return payload, nil
}

func atlasThresholdHistoryOverlayLoader(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
	debugLog("router.loader.overlay", map[string]any{"path": routeCtx.Path})
	payload, err := loadInventoryThresholdHistoryPayload(ctx, routeCtx)
	if err != nil {
		return nil, err
	}
	return router.Attrs{"payload": payload}, nil
}

func applyPayloadSideEffects(payload atlas.Payload, resetScroll bool) {
	invalidateCachesForPayload(payload)
	cachePayload(payload)
	updateDocumentMetadata(payload)
	if resetScroll {
		syncNavigationPosition(payload)
	}
	bindDirtyGuardForms()
}

func atlasRouteComponent(attrs router.Attrs) *router.Element {
	payload, _ := attrs["payload"].(atlas.Payload)
	debugLog("route.render", map[string]any{"path": payload.Route.Path, "screen": payload.Route.Screen})
	applyPayloadSideEffects(payload, true)
	return ui.CreateElement(func() ui.Node {
		return atlas.App(payload)
	})
}

func atlasThresholdHistoryOverlayComponent(attrs router.Attrs) *router.Element {
	payload, _ := attrs["payload"].(atlas.Payload)
	debugLog("route.render.overlay", map[string]any{"path": payload.Route.Path, "screen": payload.Route.Screen})
	applyPayloadSideEffects(payload, false)
	return ui.CreateElement(func() ui.Node {
		return atlas.InventoryThresholdHistoryOverlay(payload)
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

func routeOptionsForDefinition(def atlasRouteDefinition) router.Options {
	options := router.Options{
		Redirect:    def.Redirect,
		BeforeLeave: def.BeforeLeave,
		Layout:      def.Layout,
	}
	if strings.TrimSpace(def.MetadataKey) != "" {
		metadata := atlas.MetadataForPath(def.MetadataKey)
		options.Title = metadata.Title
		options.Description = metadata.Description
		options.CanonicalURL = metadata.Canonical
	}
	if def.Internal {
		options.BeforeEnter = requireInternalSessionGuard
	}
	if def.UseLoader {
		if def.Loader != nil {
			options.Loader = def.Loader
		} else {
			options.Loader = atlasRouteLoader
		}
		options.Loading = atlasLoadingElement()
	}
	return options
}

func registerAtlasRoutes(r *router.Router) {
	definitions := []atlasRouteDefinition{
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
		{Path: atlas.RouteWarehouseOps, MetadataKey: atlas.RouteWarehouseOps, UseLoader: true, Internal: true},
		{Path: atlas.RouteWarehouseDetailRoute, MetadataKey: atlas.RouteWarehouseDetail, UseLoader: true, Internal: true},
		{Path: atlas.RouteWarehouseItemDetailRoute, MetadataKey: atlas.RouteWarehouseItemDetail, UseLoader: true, Internal: true},
		{Path: atlas.RouteTransfers, MetadataKey: atlas.RouteTransfers, UseLoader: true, Internal: true},
		{Path: atlas.RouteTransferDetailRoute, MetadataKey: atlas.RouteTransferDetail, UseLoader: true, Internal: true},
		{Path: atlas.RoutePurchaseOrders, MetadataKey: atlas.RoutePurchaseOrders, UseLoader: true, Internal: true},
		{Path: atlas.RoutePurchaseOrderDetailRoute, MetadataKey: atlas.RoutePurchaseOrderDetail, UseLoader: true, Internal: true},
		{Path: atlas.RouteReceiving, MetadataKey: atlas.RouteReceiving, UseLoader: true, Internal: true},
		{Path: atlas.RouteReceivingSessionDetailRoute, MetadataKey: atlas.RouteReceivingSessionDetail, UseLoader: true, Internal: true},
		{Path: atlas.RouteComments, MetadataKey: atlas.RouteComments, UseLoader: true, Internal: true},
		{Path: atlas.RouteSettings, MetadataKey: atlas.RouteSettings, UseLoader: true, Internal: true},
		{Path: atlas.RouteCatchAll, UseLoader: true},
	}
	for _, def := range definitions {
		component := atlasRouteComponent
		if def.Component != nil {
			component = def.Component
		}
		r.Register(def.Path, component, routeOptionsForDefinition(def))
	}
}

func registerAnchorNavigation(routerInstance *router.Router) {
	document := js.Global().Get("document")
	window := js.Global().Get("window")
	if !document.Truthy() || !window.Truthy() || !document.Get("addEventListener").Truthy() {
		return
	}
	anchorNavigationHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		if event.Get("defaultPrevented").Bool() {
			return nil
		}
		if event.Get("button").Truthy() && event.Get("button").Int() != 0 {
			return nil
		}
		if event.Get("metaKey").Bool() || event.Get("ctrlKey").Bool() || event.Get("shiftKey").Bool() || event.Get("altKey").Bool() {
			return nil
		}
		target := event.Get("target")
		if !target.Truthy() {
			return nil
		}
		anchor := target.Call("closest", "a[href]")
		if !anchor.Truthy() {
			return nil
		}
		if strings.TrimSpace(anchor.Get("target").String()) != "" || anchor.Get("download").Truthy() {
			return nil
		}
		href := strings.TrimSpace(anchor.Get("href").String())
		if href == "" || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "javascript:") {
			return nil
		}
		location := window.Get("location")
		anchorURL := js.Global().Get("URL").New(href, location.Get("href"))
		if anchorURL.Get("origin").String() != location.Get("origin").String() {
			return nil
		}
		pathname := anchorURL.Get("pathname").String()
		search := anchorURL.Get("search").String()
		hash := anchorURL.Get("hash").String()
		if shouldUseDocumentNavigation(pathname) {
			return nil
		}
		if hash != "" && pathname == location.Get("pathname").String() && search == location.Get("search").String() {
			return nil
		}
		debugLog("navigation.anchor.intercept", map[string]any{
			"href":     href,
			"pathname": pathname,
			"search":   search,
			"hash":     hash,
		})
		event.Call("preventDefault")
		routerInstance.Navigate(pathname + search)
		return nil
	})
	document.Call("addEventListener", "click", anchorNavigationHandler)
	debugLog("navigation.anchor.registered", nil)
}

func main() {
	initialBootstrap = loadBootstrap()
	initialPayload = atlas.PayloadFromSSRBootstrap(initialBootstrap)
	cachePayload(initialPayload)
	debugLog("app.init", map[string]any{
		"path":    initialPayload.Route.Path,
		"screen":  initialPayload.Route.Screen,
		"surface": initialPayload.Route.Surface,
	})
	lastRenderedRouteKey = routeKey(initialPayload.Route.Path, initialPayload.Route.Query)
	hasRenderedRoute = true
	updateDocumentMetadata(initialPayload)

	r := router.NewRouter(router.RouterOptions{DefaultRoute: atlas.RouteLanding})
	registerAtlasRoutes(r)

	debugLog("hydrate.start", map[string]any{"selector": "#app", "path": initialPayload.Route.Path})
	_, _ = ui.Hydrate(atlas.App(initialPayload), "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	debugLog("hydrate.done", map[string]any{"selector": "#app"})
	bindDirtyGuardForms()
	registerBeforeUnloadGuard()
	r.HydrateMount("#app")
	debugLog("router.mount", map[string]any{"selector": "#app"})
	registerAnchorNavigation(r)
	select {}
}
