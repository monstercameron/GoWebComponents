package atlas

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type PreferenceStore struct {
	Theme            string
	Locale           string
	Density          string
	DefaultWarehouse string
}

func DefaultPreferenceStore() PreferenceStore {
	return PreferenceStore{
		Theme:            "dark",
		Locale:           "en",
		Density:          "compact",
		DefaultWarehouse: "illinois-hub",
	}
}

const (
	RouteLanding                      = "/"
	RouteAppRoot                      = "/app"
	RouteCatalog                      = "/shop"
	RouteProduct                      = "/shop/frame-desk"
	RouteProductPattern               = "/shop/:productSlug"
	RouteWarehouses                   = "/warehouses"
	RouteWarehousePublicDetail        = "/warehouses/new-jersey-hub"
	RouteWarehousePublicDetailPattern = "/warehouses/:warehouseSlug"
	RouteWarehouseAvailability        = "/warehouses/new-jersey-hub/availability/frame-desk"
	RouteWarehouseAvailabilityPattern = "/warehouses/:warehouseSlug/availability/:productSlug"
	RouteDashboard                    = "/app/dashboard"
	RouteInventory                    = "/app/inventory"
	RouteInventoryShell               = "/app/inventory*"
	RouteSKUDetail                    = "/app/inventory/frame-desk"
	RouteSKUDetailRoute               = "/app/inventory/:sku"
	RouteWarehouseOps                 = "/app/warehouses"
	RouteWarehouseDetail              = "/app/warehouses/illinois-hub"
	RouteWarehouseDetailNevada        = "/app/warehouses/nevada-hub"
	RouteWarehouseDetailNewJersey     = "/app/warehouses/new-jersey-hub"
	RouteWarehouseDetailRoute         = "/app/warehouses/:warehouseId"
	RouteWarehouseItemDetail          = "/app/warehouses/illinois-hub/items/frame-desk"
	RouteWarehouseItemDetailRoute     = "/app/warehouses/:warehouseId/items/:sku"
	RouteTransfers                    = "/app/transfers"
	RouteTransfersShell               = "/app/transfers*"
	RouteTransferDetail               = "/app/transfers/tr-2048"
	RouteTransferDetailRoute          = "/app/transfers/:transferId"
	RoutePurchaseOrders               = "/app/purchase-orders"
	RoutePurchaseOrderDetail          = "/app/purchase-orders/po-1042"
	RoutePurchaseOrderDetailRoute     = "/app/purchase-orders/:purchaseOrderId"
	RouteReceiving                    = "/app/receiving"
	RouteReceivingShell               = "/app/receiving*"
	RouteReceivingSessionDetail       = "/app/receiving/illinois-accessories-042"
	RouteReceivingSessionDetailRoute  = "/app/receiving/:sessionId"
	RouteComments                     = "/app/comments"
	RouteSettings                     = "/app/settings"
	RouteCatchAll                     = "*"
)

type RouteRegistration struct {
	Path    string
	Surface string
	Screen  string
}

var RouteManifest = []RouteRegistration{
	{Path: RouteLanding, Surface: "public", Screen: "landing"},
	{Path: RouteCatalog, Surface: "public", Screen: "catalog"},
	{Path: RouteProduct, Surface: "public", Screen: "product"},
	{Path: RouteWarehouses, Surface: "public", Screen: "warehouses"},
	{Path: RouteWarehousePublicDetail, Surface: "public", Screen: "warehouse-detail"},
	{Path: RouteWarehouseAvailability, Surface: "public", Screen: "warehouse-availability"},
	{Path: RouteDashboard, Surface: "internal", Screen: "dashboard"},
	{Path: RouteInventory, Surface: "internal", Screen: "inventory"},
	{Path: RouteSKUDetail, Surface: "internal", Screen: "sku-detail"},
	{Path: RouteWarehouseOps, Surface: "internal", Screen: "warehouse-ops"},
	{Path: RouteWarehouseDetail, Surface: "internal", Screen: "warehouse-detail"},
	{Path: RouteWarehouseItemDetail, Surface: "internal", Screen: "warehouse-item-detail"},
	{Path: RouteTransfers, Surface: "internal", Screen: "transfers"},
	{Path: RouteTransferDetail, Surface: "internal", Screen: "transfer-detail"},
	{Path: RoutePurchaseOrders, Surface: "internal", Screen: "purchase-orders"},
	{Path: RoutePurchaseOrderDetail, Surface: "internal", Screen: "purchase-order-detail"},
	{Path: RouteReceiving, Surface: "internal", Screen: "receiving"},
	{Path: RouteReceivingSessionDetail, Surface: "internal", Screen: "receiving-session-detail"},
	{Path: RouteComments, Surface: "internal", Screen: "comments"},
	{Path: RouteSettings, Surface: "internal", Screen: "settings"},
}

type PageMetadata struct {
	Title       string
	Description string
	Canonical   string
	OGImage     string
}

func MetadataForPath(path string) PageMetadata {
	if strings.HasPrefix(path, RouteWarehouses+"/") && strings.Contains(path, "/availability/") {
		return PageMetadata{Title: "Atlas Warehouse Availability", Description: "Inspect a warehouse-specific product promise for one Atlas item and one regional fulfillment hub.", Canonical: RouteWarehouseAvailability, OGImage: "/examples/static/img/atlas-warehouse-availability-og.png"}
	}
	if strings.HasPrefix(path, RouteWarehouseOps+"/") && strings.Contains(path, "/items/") {
		return PageMetadata{Title: "Atlas Warehouse Item", Description: "Manage one warehouse item with inventory edits, replenishment, and local demand context.", Canonical: RouteWarehouseItemDetail, OGImage: "/examples/static/img/atlas-warehouse-detail-og.png"}
	}
	if strings.HasPrefix(path, RouteWarehouses+"/") {
		return PageMetadata{Title: "Atlas Warehouse Detail Public", Description: "Inspect one public Atlas warehouse, including regional promise speed and product-specific availability links.", Canonical: RouteWarehousePublicDetail, OGImage: "/examples/static/img/atlas-warehouse-public-detail-og.png"}
	}
	switch path {
	case RouteLanding:
		return PageMetadata{Title: "Atlas Commerce OS", Description: "Premium modular workspace systems with warehouse-aware availability.", Canonical: "/", OGImage: "/examples/static/img/atlas-landing-og.png"}
	case RouteCatalog:
		return PageMetadata{Title: "Atlas Shop", Description: "Browse Atlas modular workspace systems with filterable discovery and regional fulfillment context.", Canonical: "/shop", OGImage: "/examples/static/img/atlas-catalog-og.png"}
	case RouteProduct:
		return PageMetadata{Title: "Atlas Frame Desk", Description: "Warehouse-aware availability and premium workspace design for the Atlas Frame Desk.", Canonical: "/shop/frame-desk", OGImage: "/examples/static/img/atlas-frame-desk-og.png"}
	case RouteWarehouses:
		return PageMetadata{Title: "Atlas Product Volume", Description: "Browse Atlas products by live volume with search, filters, and sort controls instead of a warehouse selector.", Canonical: "/warehouses", OGImage: "/examples/static/img/atlas-warehouses-og.png"}
	case RouteDashboard:
		return PageMetadata{Title: "Atlas Ops Dashboard", Description: "Operational overview of buyer follow-up, stock pressure, receiving exceptions, and warehouse health.", Canonical: "/app/dashboard", OGImage: "/examples/static/img/atlas-dashboard-og.png"}
	case RouteInventory:
		return PageMetadata{Title: "Atlas Inventory", Description: "Review inventory health, saved views, and warehouse-aware stock pressure.", Canonical: "/app/inventory", OGImage: "/examples/static/img/atlas-inventory-og.png"}
	case RouteWarehouseOps:
		return PageMetadata{Title: "Atlas Warehouses Internal", Description: "Compare staffing, backlog, and warehouse pressure across the Atlas internal network.", Canonical: "/app/warehouses", OGImage: "/examples/static/img/atlas-warehouses-internal-og.png"}
	case RouteWarehouseDetail:
		return PageMetadata{Title: "Atlas Warehouse Detail", Description: "Inspect staffing, backlog, and next action for a single Atlas warehouse.", Canonical: "/app/warehouses/illinois-hub", OGImage: "/examples/static/img/atlas-warehouse-detail-og.png"}
	case RouteSKUDetail:
		return PageMetadata{Title: "Atlas SKU Detail", Description: "Inspect warehouse breakdown, thresholds, and activity for a single Atlas SKU.", Canonical: "/app/inventory/frame-desk", OGImage: "/examples/static/img/atlas-sku-og.png"}
	case RouteTransfers:
		return PageMetadata{Title: "Atlas Transfers", Description: "Plan, review, and approve cross-warehouse transfer recommendations.", Canonical: "/app/transfers", OGImage: "/examples/static/img/atlas-transfers-og.png"}
	case RouteTransferDetail:
		return PageMetadata{Title: "Atlas Transfer Detail", Description: "Inspect one Atlas transfer, including approval state, lane context, and audit activity.", Canonical: "/app/transfers/tr-2048", OGImage: "/examples/static/img/atlas-transfer-detail-og.png"}
	case RoutePurchaseOrders:
		return PageMetadata{Title: "Atlas Purchase Orders", Description: "Review vendor approvals, inbound shipment rows, and purchase-order planning context.", Canonical: "/app/purchase-orders", OGImage: "/examples/static/img/atlas-purchase-orders-og.png"}
	case RoutePurchaseOrderDetail:
		return PageMetadata{Title: "Atlas Purchase Order Detail", Description: "Inspect one Atlas purchase order, including vendor state, ETA, and inbound shipment rows.", Canonical: "/app/purchase-orders/po-1042", OGImage: "/examples/static/img/atlas-purchase-order-detail-og.png"}
	case RouteReceiving:
		return PageMetadata{Title: "Atlas Receiving", Description: "Track inbound sessions, discrepancies, and receiving closeout state.", Canonical: "/app/receiving", OGImage: "/examples/static/img/atlas-receiving-og.png"}
	case RouteReceivingSessionDetail:
		return PageMetadata{Title: "Atlas Receiving Session", Description: "Inspect one receiving session, including discrepancy classification and closeout readiness.", Canonical: "/app/receiving/illinois-accessories-042", OGImage: "/examples/static/img/atlas-receiving-session-og.png"}
	case RouteComments:
		return PageMetadata{Title: "Atlas Buyer Follow-Up", Description: "Work through buyer questions and route product or inventory follow-up from one manager lane.", Canonical: "/app/comments", OGImage: "/examples/static/img/atlas-comments-og.png"}
	case RouteSettings:
		return PageMetadata{Title: "Atlas Settings", Description: "Manage theme, locale, density, default warehouse, and saved-view preferences.", Canonical: "/app/settings", OGImage: "/examples/static/img/atlas-settings-og.png"}
	default:
		return PageMetadata{Title: "Atlas Commerce OS", Description: "Flagship example for GoWebComponents.", Canonical: path, OGImage: "/examples/static/img/atlas-default-og.png"}
	}
}

func RouteBootstrapForPath(path string) (RouteBootstrap, bool) {
	for _, route := range RouteManifest {
		if route.Path == path {
			return RouteBootstrap{Path: route.Path, Surface: route.Surface, Screen: route.Screen, Title: MetadataForPath(path).Title}, true
		}
	}
	return RouteBootstrap{}, false
}

func CatalogPageValue(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}

func BuildCatalogQueryValues(searchQuery, category, warehouse, sortKey string, page int) url.Values {
	values := url.Values{}
	if strings.TrimSpace(searchQuery) != "" {
		values.Set("q", strings.TrimSpace(searchQuery))
	}
	if category != "" && category != "all" {
		values.Set("category", category)
	}
	if warehouse != "" && warehouse != "all" {
		values.Set("warehouse", warehouse)
	}
	if sortKey != "" && sortKey != "featured" {
		values.Set("sort", sortKey)
	}
	if page > 1 {
		values.Set("page", fmt.Sprintf("%d", page))
	}
	return values
}

func StartupRequestURL(path string, query url.Values) string {
	switch {
	case path == RouteCatalog:
		if encoded := query.Encode(); encoded != "" {
			return "/api/public/catalog?" + encoded
		}
		return "/api/public/catalog"
	case strings.HasPrefix(path, RouteCatalog+"/"):
		return "/api/public/products/" + strings.TrimPrefix(path, RouteCatalog+"/")
	case path == RouteWarehouses:
		if encoded := query.Encode(); encoded != "" {
			return "/api/public/warehouses?" + encoded
		}
		return "/api/public/warehouses"
	case strings.Contains(path, "/availability/"):
		parts := strings.Split(strings.TrimPrefix(path, RouteWarehouses+"/"), "/availability/")
		if len(parts) != 2 {
			return ""
		}
		return "/api/public/warehouses/" + parts[0] + "/availability/" + parts[1]
	case strings.HasPrefix(path, RouteWarehouses+"/"):
		return "/api/public/warehouses/" + strings.TrimPrefix(path, RouteWarehouses+"/")
	case path == RouteDashboard:
		return "/api/app/bootstrap?path=" + url.QueryEscape(RouteDashboard)
	case path == "/app/products":
		if encoded := query.Encode(); encoded != "" {
			return "/api/app/products?" + encoded
		}
		return "/api/app/products"
	case strings.HasPrefix(path, "/app/products/"):
		return "/api/app/products/" + strings.TrimPrefix(path, "/app/products/")
	case path == RouteInventory:
		if encoded := query.Encode(); encoded != "" {
			return "/api/app/inventory?" + encoded
		}
		return "/api/app/inventory"
	case strings.HasPrefix(path, RouteInventory+"/"):
		return "/api/app/inventory/" + strings.TrimPrefix(path, RouteInventory+"/")
	case path == RouteWarehouseOps:
		return "/api/app/warehouses"
	case strings.HasPrefix(path, RouteWarehouseOps+"/") && strings.Contains(strings.TrimPrefix(path, RouteWarehouseOps+"/"), "/items/"):
		if encoded := query.Encode(); encoded != "" {
			return "/api/app/warehouses/" + strings.TrimPrefix(path, RouteWarehouseOps+"/") + "?" + encoded
		}
		return "/api/app/warehouses/" + strings.TrimPrefix(path, RouteWarehouseOps+"/")
	case strings.HasPrefix(path, RouteWarehouseOps+"/"):
		if encoded := query.Encode(); encoded != "" {
			return "/api/app/warehouses/" + strings.TrimPrefix(path, RouteWarehouseOps+"/") + "?" + encoded
		}
		return "/api/app/warehouses/" + strings.TrimPrefix(path, RouteWarehouseOps+"/")
	case path == RouteTransfers:
		return "/api/app/transfers"
	case strings.HasPrefix(path, RouteTransfers+"/"):
		return "/api/app/transfers/" + strings.TrimPrefix(path, RouteTransfers+"/")
	case path == RoutePurchaseOrders:
		return "/api/app/purchase-orders"
	case strings.HasPrefix(path, RoutePurchaseOrders+"/"):
		return "/api/app/purchase-orders/" + strings.TrimPrefix(path, RoutePurchaseOrders+"/")
	case path == RouteReceiving:
		return "/api/app/receiving"
	case strings.HasPrefix(path, RouteReceiving+"/"):
		return "/api/app/receiving/" + strings.TrimPrefix(path, RouteReceiving+"/")
	case path == RouteComments:
		return "/api/app/comments"
	case path == RouteSettings:
		return "/api/app/preferences"
	default:
		return ""
	}
}

type ProductHeroState struct {
	Slug             string
	Label            string
	Finish           string
	LeadTime         string
	Availability     string
	AvailabilityTone string
	WarehousePromise string
}

type WarehousePromise struct {
	Warehouse string
	Promise   string
	Note      string
	Tone      string
	Href      string
}

func ResolveAtlasProductHeroState(productSlug, finish string) ProductHeroState {
	state := ProductHeroState{Slug: productSlug, Label: strings.Title(strings.ReplaceAll(productSlug, "-", " ")), Finish: finish, LeadTime: "2-4 days", Availability: "Low stock", AvailabilityTone: "warm", WarehousePromise: "Rebalanced from Illinois and New Jersey for east-coast installs."}
	switch finish {
	case "Drift ash":
		state.LeadTime = "5-7 days"
		state.Availability = "Made to order"
		state.AvailabilityTone = "neutral"
		state.WarehousePromise = "Built in the next upholstery batch with receiving priority held in Nevada."
	case "Walnut ember":
		state.LeadTime = "Ready this week"
		state.Availability = "In stock"
		state.AvailabilityTone = "success"
		state.WarehousePromise = "Reserved on-hand inventory is available for premium studio and boardroom projects."
	}
	return state
}

func ResolveAtlasPromiseLanes(productSlug string) []WarehousePromise {
	return []WarehousePromise{
		{Warehouse: "New Jersey Hub", Promise: "2-4 days", Note: "Fastest east-coast install window for the flagship route, but inventory should stay explicit because launch traffic drains this lane first.", Tone: "warm", Href: "#/warehouses/new-jersey-hub/availability/" + productSlug},
		{Warehouse: "Illinois Hub", Promise: "4-6 days", Note: "Most balanced mixed-region lane when the product needs a calmer promise and Atlas wants to preserve east-coast depth.", Tone: "neutral", Href: "#/warehouses/illinois-hub/availability/" + productSlug},
		{Warehouse: "Nevada Hub", Promise: "Ready this week", Note: "Strongest western stock story for larger project orders or premium finishes that need deeper inventory coverage.", Tone: "success", Href: "#/warehouses/nevada-hub/availability/" + productSlug},
	}
}

type InventorySummary struct {
	TotalAvailable       int
	PressureWarehouses   int
	ActiveWarehouseLabel string
	ActiveSavedView      string
	SuggestedAction      string
	StatusLabel          string
}

func InventorySummaryFor(savedView, warehouse string) InventorySummary {
	availableByWarehouse := map[string]int{"illinois-hub": 7, "nevada-hub": 12, "new-jersey-hub": 3}
	totalAvailable := 0
	pressureWarehouses := 0
	for warehouseID, available := range availableByWarehouse {
		totalAvailable += available
		if warehouseID == "new-jersey-hub" || available <= 4 {
			pressureWarehouses++
		}
	}
	statusLabel := "Balanced"
	suggestedAction := "Keep monitoring inbound receiving and moderate threshold edits through the SKU route."
	if savedView == "East coast shortages" || warehouse == "new-jersey-hub" {
		statusLabel = "Promise risk"
		suggestedAction = "Prioritize east-coast replenishment and hand the operator into transfer planning before SLA copy slips."
	} else if savedView == "Low stock triage" {
		statusLabel = "Low stock watch"
		suggestedAction = "Review low-cover SKUs, confirm thresholds, and clear receiving blockers before broader catalog demand spikes."
	}
	return InventorySummary{TotalAvailable: totalAvailable, PressureWarehouses: pressureWarehouses, ActiveWarehouseLabel: WarehouseLabel(warehouse), ActiveSavedView: savedView, SuggestedAction: suggestedAction, StatusLabel: statusLabel}
}

type ModerationScenario struct {
	Status   string
	Record   string
	Decision string
	Detail   string
}

type ModerationDecisionOutcome struct {
	Status         string
	Confirmation   string
	ToastTitle     string
	ToastDetail    string
	ToastTone      string
	ScenarioStatus string
}

func ModerationScenarioForStatus(status string) ModerationScenario {
	scenario := ModerationScenario{Status: status, Record: "Frame Desk cable tray question", Decision: "Awaiting moderator decision", Detail: "Buyer asks about cable-tray clearance on the flagship desk."}
	switch status {
	case "approved":
		scenario.Record = "Shelf finish delivery praise"
		scenario.Decision = "Approved and returned to the public thread"
		scenario.Detail = "Short positive delivery feedback that reinforces the launch experience."
	case "flagged":
		scenario.Record = "Aggressive stock complaint"
		scenario.Decision = "Flagged for escalation and staff follow-up"
		scenario.Detail = "Escalated complaint references stock accuracy and needs a careful response path."
	case "rejected":
		scenario.Record = "Promotional spam post"
		scenario.Decision = "Rejected and withheld from the public thread"
		scenario.Detail = "Promotional spam content is removed to keep the product thread credible."
	}
	return scenario
}

func ModerationDecisionForAction(action string) ModerationDecisionOutcome {
	switch action {
	case "approve":
		return ModerationDecisionOutcome{Status: "approved", Confirmation: "Moderation decision confirmed through the Atlas overlay.", ToastTitle: "Comment approved", ToastDetail: "Moderation approval was confirmed through the Atlas overlay.", ToastTone: "success", ScenarioStatus: "approved"}
	case "reject":
		return ModerationDecisionOutcome{Status: "rejected", Confirmation: "Moderation decision confirmed through the Atlas overlay.", ToastTitle: "Comment rejected", ToastDetail: "Moderation rejection was confirmed through the Atlas overlay.", ToastTone: "warm", ScenarioStatus: "rejected"}
	case "flag":
		return ModerationDecisionOutcome{Status: "flagged", Confirmation: "Moderation decision confirmed through the Atlas overlay.", ToastTitle: "Comment flagged", ToastDetail: "Moderation escalation was confirmed through the Atlas overlay.", ToastTone: "danger", ScenarioStatus: "flagged"}
	default:
		return ModerationDecisionOutcome{Status: "pending", ScenarioStatus: "pending"}
	}
}

func WarehouseLabel(value string) string {
	switch value {
	case "illinois-hub":
		return "Illinois Hub"
	case "nevada-hub":
		return "Nevada Hub"
	case "new-jersey-hub":
		return "New Jersey Hub"
	default:
		return strings.ReplaceAll(value, "-", " ")
	}
}
