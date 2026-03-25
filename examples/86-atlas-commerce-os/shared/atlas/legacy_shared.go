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
	RouteSKUThresholdHistory          = "/app/inventory/frame-desk/threshold-history"
	RouteSKUThresholdHistoryRoute     = "/app/inventory/:sku/threshold-history"
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
	{Path: RouteSKUThresholdHistory, Surface: "internal", Screen: "sku-threshold-history"},
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

func MetadataForPath(parsePath string) PageMetadata {
	if strings.HasPrefix(parsePath, RouteWarehouses+"/") && strings.Contains(parsePath, "/availability/") {
		return PageMetadata{Title: "Atlas Warehouse Availability", Description: "Inspect a warehouse-specific product promise for one Atlas item and one regional fulfillment hub.", Canonical: RouteWarehouseAvailability, OGImage: "/examples/static/img/atlas-warehouse-availability-og.png"}
	}
	if strings.HasPrefix(parsePath, RouteWarehouseOps+"/") && strings.Contains(parsePath, "/items/") {
		return PageMetadata{Title: "Atlas Warehouse Item", Description: "Manage one warehouse item with inventory edits, replenishment, and local demand context.", Canonical: RouteWarehouseItemDetail, OGImage: "/examples/static/img/atlas-warehouse-detail-og.png"}
	}
	if strings.HasPrefix(parsePath, RouteInventory+"/") && strings.HasSuffix(parsePath, "/threshold-history") {
		return PageMetadata{Title: "Atlas Threshold History", Description: "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context.", Canonical: RouteSKUThresholdHistory, OGImage: "/examples/static/img/atlas-sku-og.png"}
	}
	if strings.HasPrefix(parsePath, RouteWarehouses+"/") {
		return PageMetadata{Title: "Atlas Warehouse Region", Description: "Inspect one Atlas delivery region, including service posture, stocked highlights, and product-specific availability links.", Canonical: RouteWarehousePublicDetail, OGImage: "/examples/static/img/atlas-warehouse-public-detail-og.png"}
	}
	switch parsePath {
	case RouteLanding:
		return PageMetadata{Title: "Atlas Commerce OS", Description: "Premium modular workspace systems with warehouse-aware availability.", Canonical: "/", OGImage: "/examples/static/img/atlas-landing-og.png"}
	case RouteCatalog:
		return PageMetadata{Title: "Atlas Shop", Description: "Browse Atlas modular workspace systems with filterable discovery and regional fulfillment context.", Canonical: "/shop", OGImage: "/examples/static/img/atlas-catalog-og.png"}
	case RouteProduct:
		return PageMetadata{Title: "Atlas Frame Desk", Description: "Warehouse-aware availability and premium workspace design for the Atlas Frame Desk.", Canonical: "/shop/frame-desk", OGImage: "/examples/static/img/atlas-frame-desk-og.png"}
	case RouteWarehouses:
		return PageMetadata{Title: "Atlas Delivery Regions", Description: "Compare Atlas delivery regions, service levels, and stocked highlights before opening a warehouse route.", Canonical: "/warehouses", OGImage: "/examples/static/img/atlas-warehouses-og.png"}
	case RouteDashboard:
		return PageMetadata{Title: "Atlas Ops Dashboard", Description: "Operational overview of buyer questions, stock pressure, receiving exceptions, and warehouse health.", Canonical: "/app/dashboard", OGImage: "/examples/static/img/atlas-dashboard-og.png"}
	case RouteInventory:
		return PageMetadata{Title: "Atlas Inventory", Description: "Review inventory health, saved views, and warehouse-aware stock pressure.", Canonical: "/app/inventory", OGImage: "/examples/static/img/atlas-inventory-og.png"}
	case RouteWarehouseOps:
		return PageMetadata{Title: "Atlas Warehouse Operations", Description: "Compare staffing, backlog, service posture, and warehouse pressure across the Atlas network.", Canonical: "/app/warehouses", OGImage: "/examples/static/img/atlas-warehouses-internal-og.png"}
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
		return PageMetadata{Title: "Atlas Buyer Inbox", Description: "Review buyer questions, moderation decisions, and follow-up paths into product, inventory, or warehouse work.", Canonical: "/app/comments", OGImage: "/examples/static/img/atlas-comments-og.png"}
	case RouteSettings:
		return PageMetadata{Title: "Atlas Settings", Description: "Manage theme, locale, density, default warehouse, and saved-view preferences.", Canonical: "/app/settings", OGImage: "/examples/static/img/atlas-settings-og.png"}
	default:
		return PageMetadata{Title: "Atlas Commerce OS", Description: "Flagship example for GoWebComponents.", Canonical: parsePath, OGImage: "/examples/static/img/atlas-default-og.png"}
	}
}

func RouteBootstrapForPath(parsePath string) (RouteBootstrap, bool) {
	for _, parseRoute := range RouteManifest {
		if parseRoute.Path == parsePath {
			parseMeta := MetadataForPath(parsePath)
			return RouteBootstrap{
				Path:        parseRoute.Path,
				Surface:     parseRoute.Surface,
				Screen:      parseRoute.Screen,
				Title:       parseMeta.Title,
				Description: parseMeta.Description,
				Canonical:   parseMeta.Canonical,
			}, true
		}
	}
	return RouteBootstrap{}, false
}

func CatalogPageValue(parseValue string) int {
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
	if parseErr != nil || parseParsed < 1 {
		return 1
	}
	return parseParsed
}

func BuildCatalogQueryValues(parseSearchQuery, parseCategory, parseWarehouse, parseSortKey string, parsePage int) url.Values {
	parseValues := url.Values{}
	if strings.TrimSpace(parseSearchQuery) != "" {
		parseValues.Set("q", strings.TrimSpace(parseSearchQuery))
	}
	if parseCategory != "" && parseCategory != "all" {
		parseValues.Set("category", parseCategory)
	}
	if parseWarehouse != "" && parseWarehouse != "all" {
		parseValues.Set("warehouse", parseWarehouse)
	}
	if parseSortKey != "" && parseSortKey != "featured" {
		parseValues.Set("sort", parseSortKey)
	}
	if parsePage > 1 {
		parseValues.Set("page", fmt.Sprintf("%d", parsePage))
	}
	return parseValues
}

func StartupRequestURL(parsePath string, parseQuery url.Values) string {
	switch {
	case parsePath == RouteCatalog:
		if parseEncoded := parseQuery.Encode(); parseEncoded != "" {
			return "/api/public/catalog?" + parseEncoded
		}
		return "/api/public/catalog"
	case strings.HasPrefix(parsePath, RouteCatalog+"/"):
		return "/api/public/products/" + strings.TrimPrefix(parsePath, RouteCatalog+"/")
	case parsePath == RouteWarehouses:
		if parseEncoded2 := parseQuery.Encode(); parseEncoded2 != "" {
			return "/api/public/warehouses?" + parseEncoded2
		}
		return "/api/public/warehouses"
	case strings.Contains(parsePath, "/availability/"):
		parseParts := strings.Split(strings.TrimPrefix(parsePath, RouteWarehouses+"/"), "/availability/")
		if len(parseParts) != 2 {
			return ""
		}
		return "/api/public/warehouses/" + parseParts[0] + "/availability/" + parseParts[1]
	case strings.HasPrefix(parsePath, RouteWarehouses+"/"):
		return "/api/public/warehouses/" + strings.TrimPrefix(parsePath, RouteWarehouses+"/")
	case parsePath == RouteDashboard:
		return "/api/app/dashboard"
	case parsePath == "/app/products":
		if parseEncoded3 := parseQuery.Encode(); parseEncoded3 != "" {
			return "/api/app/products?" + parseEncoded3
		}
		return "/api/app/products"
	case strings.HasPrefix(parsePath, "/app/products/"):
		return "/api/app/products/" + strings.TrimPrefix(parsePath, "/app/products/")
	case parsePath == RouteInventory:
		if parseEncoded4 := parseQuery.Encode(); parseEncoded4 != "" {
			return "/api/app/inventory?" + parseEncoded4
		}
		return "/api/app/inventory"
	case strings.HasPrefix(parsePath, RouteInventory+"/") && strings.HasSuffix(parsePath, "/threshold-history"):
		return "/api/app/inventory/" + strings.TrimSuffix(strings.TrimPrefix(parsePath, RouteInventory+"/"), "/threshold-history") + "/threshold-panel"
	case strings.HasPrefix(parsePath, RouteInventory+"/"):
		return "/api/app/inventory/" + strings.TrimPrefix(parsePath, RouteInventory+"/")
	case parsePath == RouteWarehouseOps:
		return "/api/app/warehouses"
	case strings.HasPrefix(parsePath, RouteWarehouseOps+"/") && strings.Contains(strings.TrimPrefix(parsePath, RouteWarehouseOps+"/"), "/items/"):
		if parseEncoded5 := parseQuery.Encode(); parseEncoded5 != "" {
			return "/api/app/warehouses/" + strings.TrimPrefix(parsePath, RouteWarehouseOps+"/") + "?" + parseEncoded5
		}
		return "/api/app/warehouses/" + strings.TrimPrefix(parsePath, RouteWarehouseOps+"/")
	case strings.HasPrefix(parsePath, RouteWarehouseOps+"/"):
		if parseEncoded6 := parseQuery.Encode(); parseEncoded6 != "" {
			return "/api/app/warehouses/" + strings.TrimPrefix(parsePath, RouteWarehouseOps+"/") + "?" + parseEncoded6
		}
		return "/api/app/warehouses/" + strings.TrimPrefix(parsePath, RouteWarehouseOps+"/")
	case parsePath == RouteTransfers:
		return "/api/app/transfers"
	case strings.HasPrefix(parsePath, RouteTransfers+"/"):
		return "/api/app/transfers/" + strings.TrimPrefix(parsePath, RouteTransfers+"/")
	case parsePath == RoutePurchaseOrders:
		return "/api/app/purchase-orders"
	case strings.HasPrefix(parsePath, RoutePurchaseOrders+"/"):
		return "/api/app/purchase-orders/" + strings.TrimPrefix(parsePath, RoutePurchaseOrders+"/")
	case parsePath == RouteReceiving:
		return "/api/app/receiving"
	case strings.HasPrefix(parsePath, RouteReceiving+"/"):
		return "/api/app/receiving/" + strings.TrimPrefix(parsePath, RouteReceiving+"/")
	case parsePath == RouteComments:
		return "/api/app/comments"
	case parsePath == RouteSettings:
		return "/api/app/settings"
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

func ResolveAtlasProductHeroState(parseProductSlug, parseFinish string) ProductHeroState {
	parseState := ProductHeroState{Slug: parseProductSlug, Label: strings.Title(strings.ReplaceAll(parseProductSlug, "-", " ")), Finish: parseFinish, LeadTime: "2-4 days", Availability: "Low stock", AvailabilityTone: "warm", WarehousePromise: "Rebalanced from Illinois and New Jersey for east-coast installs."}
	switch parseFinish {
	case "Drift ash":
		parseState.LeadTime = "5-7 days"
		parseState.Availability = "Made to order"
		parseState.AvailabilityTone = "neutral"
		parseState.WarehousePromise = "Built in the next upholstery batch with receiving priority held in Nevada."
	case "Walnut ember":
		parseState.LeadTime = "Ready this week"
		parseState.Availability = "In stock"
		parseState.AvailabilityTone = "success"
		parseState.WarehousePromise = "Reserved on-hand inventory is available for premium studio and boardroom projects."
	}
	return parseState
}

func ResolveAtlasPromiseLanes(parseProductSlug string) []WarehousePromise {
	return []WarehousePromise{
		{Warehouse: "New Jersey Hub", Promise: "2-4 days", Note: "Fastest east-coast install window for the flagship route, but inventory should stay explicit because launch traffic drains this lane first.", Tone: "warm", Href: "#/warehouses/new-jersey-hub/availability/" + parseProductSlug},
		{Warehouse: "Illinois Hub", Promise: "4-6 days", Note: "Most balanced mixed-region lane when the product needs a calmer promise and Atlas wants to preserve east-coast depth.", Tone: "neutral", Href: "#/warehouses/illinois-hub/availability/" + parseProductSlug},
		{Warehouse: "Nevada Hub", Promise: "Ready this week", Note: "Strongest western stock story for larger project orders or premium finishes that need deeper inventory coverage.", Tone: "success", Href: "#/warehouses/nevada-hub/availability/" + parseProductSlug},
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

func InventorySummaryFor(parseSavedView, parseWarehouse string) InventorySummary {
	parseAvailableByWarehouse := map[string]int{"illinois-hub": 7, "nevada-hub": 12, "new-jersey-hub": 3}
	parseTotalAvailable := 0
	parsePressureWarehouses := 0
	for parseWarehouseID, parseAvailable := range parseAvailableByWarehouse {
		parseTotalAvailable += parseAvailable
		if parseWarehouseID == "new-jersey-hub" || parseAvailable <= 4 {
			parsePressureWarehouses++
		}
	}
	parseStatusLabel := "Balanced"
	parseSuggestedAction := "Keep monitoring inbound receiving and moderate threshold edits through the SKU route."
	if parseSavedView == "East coast shortages" || parseWarehouse == "new-jersey-hub" {
		parseStatusLabel = "Promise risk"
		parseSuggestedAction = "Prioritize east-coast replenishment and hand the operator into transfer planning before SLA copy slips."
	} else if parseSavedView == "Low stock triage" {
		parseStatusLabel = "Low stock watch"
		parseSuggestedAction = "Review low-cover SKUs, confirm thresholds, and clear receiving blockers before broader catalog demand spikes."
	}
	return InventorySummary{TotalAvailable: parseTotalAvailable, PressureWarehouses: parsePressureWarehouses, ActiveWarehouseLabel: WarehouseLabel(parseWarehouse), ActiveSavedView: parseSavedView, SuggestedAction: parseSuggestedAction, StatusLabel: parseStatusLabel}
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

func ModerationScenarioForStatus(parseStatus string) ModerationScenario {
	parseScenario := ModerationScenario{Status: parseStatus, Record: "Frame Desk cable tray question", Decision: "Awaiting moderator decision", Detail: "Buyer asks about cable-tray clearance on the flagship desk."}
	switch parseStatus {
	case "approved":
		parseScenario.Record = "Shelf finish delivery praise"
		parseScenario.Decision = "Approved and returned to the public thread"
		parseScenario.Detail = "Short positive delivery feedback that reinforces the launch experience."
	case "flagged":
		parseScenario.Record = "Aggressive stock complaint"
		parseScenario.Decision = "Flagged for escalation and staff follow-up"
		parseScenario.Detail = "Escalated complaint references stock accuracy and needs a careful response path."
	case "rejected":
		parseScenario.Record = "Promotional spam post"
		parseScenario.Decision = "Rejected and withheld from the public thread"
		parseScenario.Detail = "Promotional spam content is removed to keep the product thread credible."
	}
	return parseScenario
}

func ModerationDecisionForAction(parseAction string) ModerationDecisionOutcome {
	switch parseAction {
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

func WarehouseLabel(parseValue string) string {
	switch parseValue {
	case "illinois-hub":
		return "Illinois Hub"
	case "nevada-hub":
		return "Nevada Hub"
	case "new-jersey-hub":
		return "New Jersey Hub"
	default:
		return strings.ReplaceAll(parseValue, "-", " ")
	}
}
