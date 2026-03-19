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

var initialBootstrap ui.SSRBootstrap
var initialPayload atlas.Payload
var anchorNavigationHandler js.Func
var lastRenderedRouteKey string
var hasRenderedRoute bool

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
	bootstrap, err := ui.ReadBootstrapScript("__ATLAS_BOOTSTRAP__")
	if err != nil {
		debugLog("bootstrap.read.failed", map[string]any{"error": err.Error()})
		return ui.SSRBootstrap{}
	}
	debugLog("bootstrap.read.ok", map[string]any{
		"path":      bootstrap.Route.Path,
		"queryKeys": len(bootstrap.Route.Query),
		"paramKeys": len(bootstrap.Route.Params),
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

func decodeJSONBody(response *http.Response, target any) error {
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(target)
}

func fetchPageData(ctx context.Context, path string, query url.Values) (any, string, error) {
	requestURL := atlas.StartupRequestURL(path, query)
	debugLog("route.fetch.start", map[string]any{"path": path, "query": query.Encode(), "requestURL": requestURL})
	if strings.TrimSpace(requestURL) == "" {
		debugLog("route.fetch.skipped", map[string]any{"path": path, "reason": "empty request url"})
		return nil, "", nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		debugLog("route.fetch.request.error", map[string]any{"path": path, "requestURL": requestURL, "error": err.Error()})
		return nil, requestURL, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		debugLog("route.fetch.network.error", map[string]any{"path": path, "requestURL": requestURL, "error": err.Error()})
		return nil, requestURL, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		debugLog("route.fetch.non_ok", map[string]any{"path": path, "requestURL": requestURL, "status": response.StatusCode})
		response.Body.Close()
		return nil, requestURL, nil
	}
	var pageData any
	if err := decodeJSONBody(response, &pageData); err != nil {
		debugLog("route.fetch.decode.error", map[string]any{"path": path, "requestURL": requestURL, "error": err.Error()})
		return nil, requestURL, err
	}
	debugLog("route.fetch.ok", map[string]any{"path": path, "requestURL": requestURL})
	return pageData, requestURL, nil
}

func routeSurface(path string) string {
	if strings.HasPrefix(path, "/app/") {
		return "internal"
	}
	return "public"
}

func routeScreen(path string) string {
	switch {
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
	case path == atlas.RouteDashboard:
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

func updateDocumentMetadata(payload atlas.Payload) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	debugLog("metadata.update", map[string]any{
		"path":  payload.Route.Path,
		"title": payload.Route.Title,
	})
	if strings.TrimSpace(payload.Route.Title) != "" {
		document.Set("title", payload.Route.Title)
	}
	if head := document.Get("head"); head.Truthy() {
		description := routeDescription(payload.Route.Path, payload.Data["page"])
		if strings.TrimSpace(description) != "" {
			meta := document.Call("querySelector", `meta[name="description"]`)
			if meta.Truthy() {
				meta.Call("setAttribute", "content", description)
			}
		}
		canonical := document.Call("querySelector", `link[rel="canonical"]`)
		if canonical.Truthy() {
			canonical.Call("setAttribute", "href", payload.Route.Path)
		}
	}
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
	payload := initialPayload
	payload.Route = atlas.RouteBootstrap{
		Path:    path,
		Query:   cloneQueryValues(query),
		Params:  map[string]string{},
		Surface: routeSurface(path),
		Screen:  routeScreen(path),
		Title:   routeTitle(path, pageData),
	}
	payload.Data = map[string]any{"page": pageData}
	payload.Requests = map[string]atlas.Request{}
	if strings.TrimSpace(requestURL) != "" {
		payload.Requests["page"] = atlas.Request{
			Method: http.MethodGet,
			URL:    requestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"page": pageData},
		}
	}
	debugLog("payload.build", map[string]any{
		"path":       path,
		"screen":     payload.Route.Screen,
		"surface":    payload.Route.Surface,
		"requestURL": requestURL,
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
		Path:    path,
		Query:   cloneQueryValues(query),
		Params:  map[string]string{},
		Surface: routeSurface(path),
		Screen:  "recovery",
		Title:   metadata.Title + " Unavailable",
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

func loadRoutePayload(ctx context.Context, routeCtx router.RouteContext) (atlas.Payload, error) {
	debugLog("route.load.start", map[string]any{
		"path":  routeCtx.Path,
		"query": routeCtx.Query.Values().Encode(),
	})
	query := routeCtx.Query.Values()
	pageData, requestURL, err := fetchPageData(ctx, routeCtx.Path, query)
	if err != nil {
		debugLog("route.load.error", map[string]any{"path": routeCtx.Path, "error": err.Error()})
		return buildRecoveryPayload(routeCtx.Path, query, requestURL, err), nil
	}
	if pageData == nil && strings.TrimSpace(requestURL) == "" && routeCtx.Path == atlas.RouteLanding {
		debugLog("route.load.static", map[string]any{"path": routeCtx.Path, "reason": "static landing route"})
		return buildPayload(routeCtx.Path, query, map[string]any{}, ""), nil
	}
	if pageData == nil {
		debugLog("route.load.missing", map[string]any{"path": routeCtx.Path, "requestURL": requestURL})
		return buildRecoveryPayload(routeCtx.Path, query, requestURL, nil), nil
	}
	payload := buildPayload(routeCtx.Path, query, pageData, requestURL)
	debugLog("route.load.ok", map[string]any{"path": routeCtx.Path, "title": payload.Route.Title})
	return payload, nil
}

func atlasRouteComponent(attrs router.Attrs) *router.Element {
	payload, _ := attrs["payload"].(atlas.Payload)
	debugLog("route.render", map[string]any{"path": payload.Route.Path, "screen": payload.Route.Screen})
	updateDocumentMetadata(payload)
	syncNavigationPosition(payload)
	return ui.CreateElement(func() ui.Node {
		return atlas.App(payload)
	})
}

func main() {
	initialBootstrap = loadBootstrap()
	initialPayload = atlas.PayloadFromSSRBootstrap(initialBootstrap)
	debugLog("app.init", map[string]any{
		"path":    initialPayload.Route.Path,
		"screen":  initialPayload.Route.Screen,
		"surface": initialPayload.Route.Surface,
	})
	lastRenderedRouteKey = routeKey(initialPayload.Route.Path, initialPayload.Route.Query)
	hasRenderedRoute = true
	updateDocumentMetadata(initialPayload)

	r := router.NewRouter(router.RouterOptions{DefaultRoute: atlas.RouteLanding})
	r.Register("*", atlasRouteComponent, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			debugLog("router.loader", map[string]any{"path": routeCtx.Path})
			payload, err := loadRoutePayload(ctx, routeCtx)
			if err != nil {
				debugLog("router.loader.error", map[string]any{"path": routeCtx.Path, "error": err.Error()})
				return nil, err
			}
			return router.Attrs{"payload": payload}, nil
		},
		Loading: ui.CreateElement(func() ui.Node {
			return html.Div(html.Props{Class: "min-h-screen"})
		}),
	})

	debugLog("hydrate.start", map[string]any{"selector": "#app", "path": initialPayload.Route.Path})
	_, _ = ui.Hydrate(atlas.App(initialPayload), "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	debugLog("hydrate.done", map[string]any{"selector": "#app"})
	r.HydrateMount("#app")
	debugLog("router.mount", map[string]any{"selector": "#app"})
	registerAnchorNavigation(r)
	select {}
}
