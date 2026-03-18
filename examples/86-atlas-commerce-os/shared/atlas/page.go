package atlas

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type catalogPage struct {
	Items    []productCard     `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Query    catalogQueryState `json:"query"`
	Focus    string            `json:"focus"`
}

type catalogQueryState struct {
	Search    string `json:"search"`
	Category  string `json:"category"`
	Status    string `json:"status"`
	Warehouse string `json:"warehouse"`
	Sort      string `json:"sort"`
}

type productCard struct {
	SKU            string `json:"sku"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	PriceCents     int    `json:"priceCents"`
	Status         string `json:"status"`
	Summary        string `json:"summary"`
	Details        string `json:"details"`
	Finish         string `json:"finish"`
	SEODescription string `json:"seoDescription"`
	WarehouseID    string `json:"warehouseId"`
	WarehouseName  string `json:"warehouseName"`
	Available      int    `json:"available"`
	Inbound        int    `json:"inbound"`
	Volume         int    `json:"volume"`
	UpdatedAt      string `json:"updatedAt"`
}

type productDetailPage struct {
	Product  productCard     `json:"product"`
	Comments []commentRecord `json:"comments"`
}

type warehouseList struct {
	Items []warehouseCard `json:"items"`
}

type warehouseCard struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
	Pressure      string `json:"pressure"`
	Staffing      string `json:"staffing"`
	Backlog       string `json:"backlog"`
	Focus         string `json:"focus"`
	Available     int    `json:"available"`
	Inbound       int    `json:"inbound"`
	RiskCount     int    `json:"riskCount"`
}

type warehouseDetailPage struct {
	Warehouse warehouseCard `json:"warehouse"`
	Products  []productCard `json:"products"`
}

type availabilityPage struct {
	Warehouse warehouseCard `json:"warehouse"`
	Product   productCard   `json:"product"`
	Available int           `json:"available"`
	Inbound   int           `json:"inbound"`
	Status    string        `json:"status"`
}

type dashboardPage struct {
	Alerts    int               `json:"alerts"`
	Transfers []transferRecord  `json:"transfers"`
	Receiving []receivingRecord `json:"receiving"`
	Comments  []commentRecord   `json:"comments"`
}

type inventoryList struct {
	Items []inventoryRow `json:"items"`
}

type inventoryRow struct {
	ID             string `json:"id"`
	SKU            string `json:"sku"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	PriceCents     int    `json:"priceCents"`
	ProductStatus  string `json:"productStatus"`
	WarehouseID    string `json:"warehouseId"`
	WarehouseName  string `json:"warehouseName"`
	OnHand         int    `json:"onHand"`
	Reserved       int    `json:"reserved"`
	Available      int    `json:"available"`
	CoverDays      int    `json:"coverDays"`
	Inbound        int    `json:"inbound"`
	Damaged        int    `json:"damaged"`
	ReorderPoint   int    `json:"reorderPoint"`
	SafetyStock    int    `json:"safetyStock"`
	Status         string `json:"status"`
	WeeklyUnits    int    `json:"weeklyUnits"`
	WeeklyRevenue  int    `json:"weeklyRevenue"`
	SellThrough    int    `json:"sellThrough"`
	DemandScore    int    `json:"demandScore"`
	RegionalShare  int    `json:"regionalShare"`
	ReorderUnits   int    `json:"reorderUnits"`
	MarketPressure string `json:"marketPressure"`
	MarketSignal   string `json:"marketSignal"`
	UpdatedAt      string `json:"updatedAt"`
}

type warehouseOpsList struct {
	Items []warehouseOpsRecord `json:"items"`
}

type warehouseOpsRecord struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
	Pressure      string `json:"pressure"`
	Staffing      string `json:"staffing"`
	Backlog       string `json:"backlog"`
	Focus         string `json:"focus"`
	Available     int    `json:"available"`
	Inbound       int    `json:"inbound"`
	RiskCount     int    `json:"riskCount"`
}

type warehouseInventoryItemDetailPage struct {
	Warehouse warehouseOpsRecord    `json:"warehouse"`
	Item      inventoryRow          `json:"item"`
	Product   productAdminCard      `json:"product"`
	Network   []inventoryRow        `json:"network"`
	Orders    []purchaseOrderRecord `json:"orders"`
	Filters   map[string]string     `json:"filters"`
}

type transferList struct {
	Items []transferRecord `json:"items"`
}

type transferRecord struct {
	ID                   string `json:"id"`
	SourceWarehouseID    string `json:"sourceWarehouseId"`
	DestinationWarehouse string `json:"destinationWarehouseId"`
	Status               string `json:"status"`
	Reason               string `json:"reason"`
	RecommendedBy        string `json:"recommendedBy"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type transferLineRecord struct {
	ID         string `json:"id"`
	TransferID string `json:"transferId"`
	ProductSKU string `json:"productSku"`
	Quantity   int    `json:"quantity"`
}

type transferDetailPage struct {
	Transfer transferRecord       `json:"transfer"`
	Lines    []transferLineRecord `json:"lines"`
}

type purchaseOrderList struct {
	Items []purchaseOrderRecord `json:"items"`
}

type purchaseOrderRecord struct {
	ID            string `json:"id"`
	VendorName    string `json:"vendorName"`
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	Status        string `json:"status"`
	PriorityNote  string `json:"priorityNote"`
	ETA           string `json:"eta"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type purchaseOrderLineRecord struct {
	ID              string `json:"id"`
	PurchaseOrderID string `json:"purchaseOrderId"`
	ProductSKU      string `json:"productSku"`
	Quantity        int    `json:"quantity"`
	ETA             string `json:"eta"`
	Status          string `json:"status"`
}

type purchaseOrderDetailPage struct {
	Order purchaseOrderRecord       `json:"order"`
	Lines []purchaseOrderLineRecord `json:"lines"`
}

type receivingList struct {
	Items []receivingRecord `json:"items"`
}

type receivingRecord struct {
	ID                 string `json:"id"`
	SourceType         string `json:"sourceType"`
	SourceID           string `json:"sourceId"`
	WarehouseID        string `json:"warehouseId"`
	Status             string `json:"status"`
	DiscrepancySummary string `json:"discrepancySummary"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type receivingLineRecord struct {
	ID                 string `json:"id"`
	ReceivingSessionID string `json:"receivingSessionId"`
	ProductSKU         string `json:"productSku"`
	ExpectedQuantity   int    `json:"expectedQuantity"`
	ActualQuantity     int    `json:"actualQuantity"`
	DiscrepancyReason  string `json:"discrepancyReason"`
}

type receivingDetailPage struct {
	Session receivingRecord       `json:"session"`
	Lines   []receivingLineRecord `json:"lines"`
}

type commentList struct {
	Items []commentRecord `json:"items"`
}

type commentRecord struct {
	ID               string `json:"id"`
	ProductSKU       string `json:"productSku"`
	AuthorName       string `json:"authorName"`
	AuthorType       string `json:"authorType"`
	Reaction         string `json:"reaction"`
	Subject          string `json:"subject"`
	Body             string `json:"body"`
	Status           string `json:"status"`
	ModerationReason string `json:"moderationReason"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type mockSignInRole struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type mockSignInPage struct {
	Roles   []mockSignInRole `json:"roles"`
	Next    string           `json:"next"`
	Message string           `json:"message"`
}

type recoveryPage struct {
	Title         string `json:"title"`
	Message       string `json:"message"`
	RecoveryHref  string `json:"recoveryHref"`
	RecoveryLabel string `json:"recoveryLabel"`
	Detail        string `json:"detail"`
}

func App(payload Payload) ui.Node {
	surface := strings.TrimSpace(payload.Route.Surface)
	rootClass := "min-h-screen bg-[radial-gradient(circle_at_top,rgba(245,158,11,0.16),transparent_22%),linear-gradient(180deg,rgba(7,10,18,1),rgba(13,18,30,1)_42%,rgba(7,10,18,1))] text-stone-100"
	mainClass := "mx-auto grid w-full max-w-7xl gap-8 px-5 pb-12 pt-6 sm:px-6 lg:px-10"
	if surface != "" && surface != "public" {
		rootClass = "min-h-screen bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.12),transparent_28%),linear-gradient(180deg,rgba(2,6,23,1),rgba(15,23,42,1))] text-white"
		mainClass = "mx-auto grid w-full max-w-6xl gap-8 px-6 pb-12 pt-6 lg:px-10"
	}
	children := []ui.Node{header(payload)}
	mainChildren := []ui.Node{}
	if banner := noticeBanner(payload); banner != nil {
		mainChildren = append(mainChildren, banner)
	}
	mainChildren = append(mainChildren, hero(payload), pageContent(payload))
	children = append(children, html.Main(html.Props{Class: mainClass}, mainChildren...))
	return html.Div(html.Props{Class: rootClass}, children...)
}

func header(payload Payload) ui.Node {
	links := []struct{ label, href string }{
		{publicNavStorefrontLabel, RouteLanding},
		{publicNavShopLabel, RouteCatalog},
		{publicNavWarehousesLabel, RouteWarehouses},
	}
	if payload.User != nil {
		links = append(links,
			struct{ label, href string }{"Dashboard", RouteDashboard},
			struct{ label, href string }{"Inventory", RouteInventory},
			struct{ label, href string }{"Products", "/app/products"},
			struct{ label, href string }{"Warehouses", RouteWarehouseOps},
			struct{ label, href string }{"Transfers", RouteTransfers},
			struct{ label, href string }{"Purchase Orders", RoutePurchaseOrders},
			struct{ label, href string }{"Receiving", RouteReceiving},
			struct{ label, href string }{"Comments", RouteComments},
			struct{ label, href string }{"Settings", RouteSettings},
		)
	}
	nodes := make([]ui.Node, 0, len(links))
	if payload.Route.Surface == "public" || payload.Route.Surface == "" {
		for _, link := range links {
			className := "rounded-full border border-white/12 bg-white/6 px-4 py-2.5 text-sm font-medium text-stone-300 shadow-[0_10px_25px_rgba(0,0,0,0.16)] transition hover:border-amber-300/60 hover:bg-white/10 hover:text-white"
			if activeNavLink(payload.Route.Path, link.href) {
				className = "rounded-full border border-amber-300/70 bg-amber-300/12 px-4 py-2.5 text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
			}
			nodes = append(nodes, html.A(html.Props{Href: link.href, Class: className}, html.Text(link.label)))
		}
		identity := publicIdentityLabel
		if payload.User != nil {
			identity = payload.User.DisplayName + " · " + strings.ReplaceAll(payload.User.Role, "_", " ")
		}
		return html.Header(html.Props{Class: "sticky top-0 z-20 border-b border-white/10 bg-[rgba(7,10,18,0.86)] backdrop-blur-xl"},
			html.Div(html.Props{Class: "mx-auto flex w-full max-w-7xl flex-col gap-4 px-5 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-10"},
				html.Div(html.Props{Class: "flex items-center justify-between gap-4"},
					html.Div(html.Props{Class: "flex flex-col"},
						html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.42em] text-amber-700"}, html.Text(publicBrandLabel)),
						html.P(html.Props{Class: "mt-2 text-sm text-stone-400"}, html.Text(identity)),
					),
					html.A(html.Props{Href: RouteCatalog, Class: "inline-flex rounded-full border border-amber-300/60 px-4 py-2 text-sm font-semibold text-amber-100 transition hover:bg-amber-300/12 lg:hidden"}, html.Text(publicBrowseShopLabel)),
				),
				html.Nav(html.Props{Class: "flex flex-wrap items-center gap-2 lg:justify-end"}, nodes...),
			),
		)
	}
	for _, link := range links {
		className := "rounded-full border border-white/10 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-cyan-300 hover:text-white"
		if activeNavLink(payload.Route.Path, link.href) {
			className = "rounded-full border border-cyan-300 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100"
		}
		nodes = append(nodes, html.A(html.Props{Href: link.href, Class: className}, html.Text(link.label)))
	}
	identity := "Public browsing"
	if payload.User != nil {
		identity = payload.User.DisplayName + " · " + strings.ReplaceAll(payload.User.Role, "_", " ")
	}
	return html.Header(html.Props{Class: "border-b border-white/10 bg-slate-950/80 backdrop-blur"},
		html.Div(html.Props{Class: "mx-auto grid w-full max-w-6xl gap-4 px-6 py-5 lg:px-10"},
			html.Div(html.Props{Class: "flex items-center justify-between gap-4"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.35em] text-cyan-300"}, html.Text(publicBrandLabel)),
					html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(identity)),
				),
				html.Nav(html.Props{Class: "flex flex-wrap justify-end gap-2"}, nodes...),
			),
		),
	)
}

func noticeBanner(payload Payload) ui.Node {
	notice := firstQueryValue(payload.Route.Query, "atlas_notice")
	if notice == "" {
		return nil
	}
	if payload.Route.Surface == "public" || payload.Route.Surface == "" {
		return html.Div(html.Props{Class: "rounded-[1.75rem] border border-emerald-400/30 bg-emerald-400/10 px-5 py-4 text-sm font-medium text-emerald-100 shadow-[0_18px_40px_rgba(16,185,129,0.12)]"}, html.Text(strings.ReplaceAll(notice, "+", " ")))
	}
	return html.Div(html.Props{Class: "rounded-3xl border border-emerald-300/25 bg-emerald-400/10 px-5 py-4 text-sm text-emerald-100"}, html.Text(strings.ReplaceAll(notice, "+", " ")))
}

func hero(payload Payload) ui.Node {
	if payload.Route.Surface == "public" || payload.Route.Surface == "" {
		return renderPublicHero(payload)
	}
	meta := []ui.Node{
		statCard("Surface", fallback(payload.Route.Surface, "public")),
		statCard("Locale", fallback(payload.I18n.Locale, "en")),
		statCard("Density", fallback(payload.Preferences.Density, "compact")),
	}
	if payload.User != nil {
		meta = append(meta, statCard("Warehouse", fallback(payload.User.DefaultWarehouse, payload.Preferences.DefaultWarehouse)))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.8fr)]"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[linear-gradient(145deg,rgba(10,18,32,0.96),rgba(15,23,42,0.88))] p-8 shadow-[0_32px_90px_rgba(8,15,30,0.35)]"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.35em] text-cyan-300"}, html.Text(fallback(payload.Route.Screen, "route"))),
			html.H1(html.Props{Class: "mt-4 text-4xl font-black tracking-tight text-white lg:text-5xl"}, html.Text(fallback(payload.Route.Title, "Atlas"))),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-base leading-7 text-slate-300"}, html.Text(routeSummary(payload.Route.Path))),
		),
		html.Div(html.Props{Class: "grid gap-4"}, meta...),
	)
}

func pageContent(payload Payload) ui.Node {
	if payload.Route.Screen == "mock-sign-in" {
		return mockSignInContent(payload)
	}
	if payload.Route.Screen == "recovery" {
		return recoveryContent(payload)
	}
	switch payload.Route.Path {
	case RouteLanding:
		return renderLandingContent()
	case RouteCatalog:
		return renderCatalogContent(decode[catalogPage](pageData(payload)))
	case RouteWarehouses:
		return renderWarehouseListContent(decode[catalogPage](pageData(payload)))
	case RouteInventory:
		return inventoryContent(payload)
	case "/app/products":
		return productsCMSContent(payload)
	case RouteDashboard:
		return dashboardContent(payload)
	case RouteWarehouseOps:
		return warehouseOpsContent(payload)
	case RouteTransfers:
		return transfersContent(payload)
	case RoutePurchaseOrders:
		return purchaseOrdersContent(payload)
	case RouteReceiving:
		return receivingContent(payload)
	case RouteComments:
		return commentsContent(payload)
	case RouteSettings:
		return settingsContent(payload)
	default:
		if strings.HasPrefix(payload.Route.Path, RouteCatalog+"/") {
			return renderProductContent(decode[productDetailPage](pageData(payload)), payload)
		}
		if strings.Contains(payload.Route.Path, "/availability/") {
			return renderAvailabilityContent(decode[availabilityPage](pageData(payload)), payload)
		}
		if strings.HasPrefix(payload.Route.Path, RouteWarehouseOps+"/") && strings.Contains(payload.Route.Path, "/items/") {
			return warehouseOpsItemContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, RouteWarehouseOps+"/") {
			return warehouseOpsDetailContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, "/app/products/") {
			return productEditorContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, RouteInventory+"/") {
			return skuContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, RouteTransfers+"/") {
			return transferDetailContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, RoutePurchaseOrders+"/") {
			return purchaseOrderDetailContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, RouteReceiving+"/") {
			return receivingDetailContent(payload)
		}
		if strings.HasPrefix(payload.Route.Path, RouteWarehouses+"/") {
			page := decode[warehouseDetailPage](pageData(payload))
			if strings.TrimSpace(page.Warehouse.ID) == "" {
				page.Warehouse = decode[warehouseCard](pageData(payload))
			}
			return renderWarehouseDetailContent(page)
		}
		return fallbackContent(payload)
	}
}

func dashboardContent(payload Payload) ui.Node {
	page := decode[dashboardPage](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-6"},
			featureCard("Demand and operations", "Use this workspace to route buyer follow-up, catalog accuracy work, and stock-pressure decisions instead of bouncing between unrelated pages."),
			internalWorkflowSection("Admin flows", "Start from a real operator job, then move through the connected routes without guessing where the next step lives.",
				internalWorkflowCard("Flow 1", "Add new item", "Create a product record, assign the first warehouse lane, and keep the new SKU inside Atlas management routes.", "/app/products#create-product"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product CMS and work through summary, details, finish, and SEO fields from one merch route.", "/app/products?sort=updated"),
				internalWorkflowCard("Flow 3", "Fix stock risk", "Open the inventory workspace already narrowed to pressure lanes that need threshold or quantity changes.", "/app/inventory?status=promise_risk"),
				internalWorkflowCard("Flow 4", "Order more units", "Jump into purchase-order planning when warehouse demand needs an actual replenishment action.", "/app/purchase-orders"),
				internalWorkflowCard("Flow 5", "Reconcile receiving", "Close inbound sessions, capture discrepancies, and return inventory to available units cleanly.", "/app/receiving"),
				internalWorkflowCard("Flow 6", "Review buyer questions", "Start in the buyer inbox, then branch into product or inventory work only when the question proves it is needed.", "/app/comments"),
			),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				statCard("Alerts", fmt.Sprintf("%d open", page.Alerts)),
				statCard("Transfers", fmt.Sprintf("%d active", len(page.Transfers))),
				statCard("Receiving", fmt.Sprintf("%d sessions", len(page.Receiving))),
			),
			listCard("Buyer follow-up queue", commentNodes(page.Comments)...),
			listCard("Transfer watch", transferNodes(page.Transfers)...),
			listCard("Receiving exceptions", receivingNodes(page.Receiving)...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard("Manager routing", "Customer-facing actions should resolve into clear lanes: buyer follow-up for questions, product CMS for merchandising, and inventory for supply decisions."),
			preferenceForm(payload),
			moderationForm(page.Comments, payload),
		),
	)
}

func inventoryContent(payload Payload) ui.Node {
	return inventoryCMSContent(payload)
}

func skuContent(payload Payload) ui.Node {
	return inventoryDetailContent(payload)
}

func warehouseOpsContent(payload Payload) ui.Node {
	page := decode[warehouseOpsList](pageData(payload))
	nodes := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		nodes = append(nodes, html.A(html.Props{Href: "/app/warehouses/" + item.ID, Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-white/5 p-5 transition hover:border-cyan-300/60 hover:bg-white/10"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(item.Name)),
					html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(item.Region+" · "+item.ServiceLevel)),
				),
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(item.Pressure)),
			),
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(item.Focus)),
			html.Div(html.Props{Class: "grid gap-3 text-sm text-slate-200 md:grid-cols-3"},
				statCard("Available", fmt.Sprintf("%d units", item.Available)),
				statCard("Inbound", fmt.Sprintf("%d inbound", item.Inbound)),
				statCard("Risks", fmt.Sprintf("%d active", item.RiskCount)),
			),
		))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps, Current: true},
			),
			internalWorkflowSection("Warehouse network flows", "Start with the warehouse map when the operator knows the facility problem but not yet the exact SKU or replenishment action.",
				internalWorkflowCard("Flow 1", "Open a warehouse roster", "Choose the facility first when the work is local backlog, staffing, or regional supply pressure.", "/app/warehouses"),
				internalWorkflowCard("Flow 2", "Add warehouse item", "Create a new managed item directly inside the warehouse route instead of bouncing through generic catalog pages.", "/app/warehouses/new-jersey-hub#warehouse-create-item"),
				internalWorkflowCard("Flow 3", "Review flagged lanes", "Move into a facility detail page already filtered to the lanes that need action.", "/app/warehouses/new-jersey-hub?status=promise_risk"),
				internalWorkflowCard("Flow 4", "Order more units", "Open replenishment only after the warehouse context proves that inbound recovery is the right move.", "/app/purchase-orders"),
			),
			listCard("Warehouse network", nodes...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			internalWorkflowSection("Cross-route handoffs", "Warehouse work should hand off cleanly into inventory, purchasing, and receiving.",
				internalWorkflowCard("Inventory", "Open risk view", "Compare cross-warehouse SKU pressure before changing a local lane.", "/app/inventory?status=promise_risk"),
				internalWorkflowCard("Purchase orders", "Create replenishment", "Escalate from local warehouse pressure into vendor-side inbound planning.", "/app/purchase-orders"),
				internalWorkflowCard("Receiving", "Close inbound work", "Return here after receiving confirms the recovery landed in the warehouse.", "/app/receiving"),
			),
		),
	)
}

func warehouseOpsDetailContent(payload Payload) ui.Node {
	return warehouseInventoryDetailContent(payload)
}

func transfersContent(payload Payload) ui.Node {
	page := decode[transferList](pageData(payload))
	rows := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		rows = append(rows, html.A(html.Props{Href: "/app/transfers/" + item.ID, Class: "grid gap-2 rounded-[1.35rem] border border-white/10 bg-white/5 p-4 transition hover:border-cyan-300/60 hover:bg-white/10"},
			html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.SourceWarehouseID+" → "+item.DestinationWarehouse)),
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(item.Reason)),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-300"}, html.Text(item.Status+" · "+item.RecommendedBy)),
		))
	}
	leftChildren := []ui.Node{
		internalWorkflowSection("Transfer flows", "Use transfers as one step in a larger warehouse recovery path, not an isolated page.",
			internalWorkflowCard("Start", "Review warehouse pressure", "Check which warehouse is starving before you commit a transfer lane.", "/app/warehouses"),
			internalWorkflowCard("Next", "Open inventory pressure", "Validate that the SKU really needs a rebalance instead of a replenishment order.", "/app/inventory?status=promise_risk"),
			internalWorkflowCard("Finish", "Confirm receiving follow-through", "After transfer approval, keep the receiving route in view for downstream reconciliation work.", "/app/receiving"),
		),
	}
	leftChildren = append(leftChildren, rows...)
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-4"}, leftChildren...),
		html.Div(html.Props{Class: "grid gap-5"}, transferForm(payload)),
	)
}

func transferDetailContent(payload Payload) ui.Node {
	page := decode[transferDetailPage](pageData(payload))
	lineNodes := make([]ui.Node, 0, len(page.Lines))
	for _, line := range page.Lines {
		lineNodes = append(lineNodes, infoRow(line.ProductSKU, fmt.Sprintf("%d units", line.Quantity)))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard(page.Transfer.ID, page.Transfer.Reason),
			internalWorkflowSection("Transfer detail flow", "A transfer is only useful if it stays tied to the warehouse and receiving steps around it.",
				internalWorkflowCard("Back", "Warehouse network", "Re-check the broader warehouse posture if this transfer no longer looks like the right balancing move.", "/app/warehouses"),
				internalWorkflowCard("Next", "Inventory pressure", "Validate that the SKU still needs movement rather than fresh vendor replenishment.", "/app/inventory?status=promise_risk"),
				internalWorkflowCard("Finish", "Receiving follow-through", "Use receiving to close the physical movement once the transfer actually lands.", "/app/receiving"),
			),
			listCard("Transfer lines", lineNodes...),
		),
		html.Div(html.Props{Class: "grid gap-4"},
			statCard("Lane", page.Transfer.SourceWarehouseID+" → "+page.Transfer.DestinationWarehouse),
			statCard("Status", page.Transfer.Status),
			statCard("Recommended by", fallback(page.Transfer.RecommendedBy, "Atlas planning")),
		),
	)
}

func purchaseOrdersContent(payload Payload) ui.Node {
	page := decode[purchaseOrderList](pageData(payload))
	rows := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		rows = append(rows, html.A(html.Props{Href: "/app/purchase-orders/" + item.ID, Class: "grid gap-2 rounded-[1.35rem] border border-white/10 bg-white/5 p-4 transition hover:border-cyan-300/60 hover:bg-white/10"},
			html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.ID+" · "+item.VendorName)),
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(item.WarehouseName+" · ETA "+item.ETA)),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-300"}, html.Text(item.Status+" · "+item.PriorityNote)),
		))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			internalWorkflowSection("Purchase-order flows", "Use purchase orders when stock recovery crosses the line from internal balancing into vendor replenishment.",
				internalWorkflowCard("Flow 1", "Start from stock risk", "Open the inventory pressure view before creating a replenishment plan.", "/app/inventory?status=promise_risk"),
				internalWorkflowCard("Flow 2", "Check warehouse context", "Use warehouse operations to confirm which facility should own the inbound units.", "/app/warehouses"),
				internalWorkflowCard("Flow 3", "Close in receiving", "Treat receiving as the final step once the purchase order turns into a real inbound session.", "/app/receiving"),
			),
			listCard("Purchase orders", rows...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			internalWorkflowSection("Common next steps", "The purchase-order route should hand off forward into receiving and backward into warehouse context.",
				internalWorkflowCard("Next", "Review receiving", "Check whether an inbound session already exists for the warehouse and vendor plan you are reviewing.", "/app/receiving"),
				internalWorkflowCard("Alternate", "Review transfers", "If vendor replenishment is too slow, compare whether an internal balancing move is the better short-term action.", "/app/transfers"),
				internalWorkflowCard("Context", "Back to products", "Return to product copy only when demand signals show the issue is storefront clarity, not supply.", "/app/products"),
			),
		),
	)
}

func purchaseOrderDetailContent(payload Payload) ui.Node {
	page := decode[purchaseOrderDetailPage](pageData(payload))
	lineNodes := make([]ui.Node, 0, len(page.Lines))
	for _, line := range page.Lines {
		lineNodes = append(lineNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(line.ProductSKU+" · "+line.Status)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(fmt.Sprintf("%d units · ETA %s", line.Quantity, line.ETA))),
		))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard(page.Order.VendorName, fallback(page.Order.PriorityNote, "Vendor planning lane")),
			internalWorkflowSection("Purchase-order detail flow", "Resolve the order here, then move directly into the route that consumes the decision.",
				internalWorkflowCard("Back", "All purchase orders", "Return to the order list when you need to compare other vendor lanes or ETA posture.", "/app/purchase-orders"),
				internalWorkflowCard("Next", "Receiving session", "Move into receiving once this order is ready to be reconciled physically at the warehouse.", "/app/receiving"),
				internalWorkflowCard("Cross-check", "Warehouse detail", "Re-open the owning warehouse route if the inbound allocation needs another look.", "/app/warehouses"),
			),
			listCard("Inbound lines", lineNodes...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			purchaseOrderStatusForm(page.Order.ID, page.Order.Status, payload),
			statCard("Warehouse", page.Order.WarehouseName),
			statCard("ETA", page.Order.ETA),
		),
	)
}

func receivingContent(payload Payload) ui.Node {
	page := decode[receivingList](pageData(payload))
	rows := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		rows = append(rows, html.A(html.Props{Href: "/app/receiving/" + item.ID, Class: "grid gap-2 rounded-[1.35rem] border border-white/10 bg-white/5 p-4 transition hover:border-cyan-300/60 hover:bg-white/10"},
			html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.ID+" · "+item.Status)),
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(item.WarehouseID+" · "+item.SourceType+" "+item.SourceID)),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(fallback(item.DiscrepancySummary, "No discrepancies recorded."))),
		))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-4"}, rows...),
		html.Div(html.Props{Class: "grid gap-5"},
			internalWorkflowSection("Receiving flows", "Receiving closes the loop on purchase orders, transfer lanes, and warehouse recovery work.",
				internalWorkflowCard("Start", "Review purchase orders", "Confirm the inbound plan before you reconcile a warehouse session.", "/app/purchase-orders"),
				internalWorkflowCard("Next", "Open warehouse pressure", "Check whether the inbound changes a specific warehouse recovery path.", "/app/warehouses"),
				internalWorkflowCard("Finish", "Return to inventory", "Verify the quantity changes landed where the operators expect them.", "/app/inventory"),
			),
			receivingForm(payload),
		),
	)
}

func receivingDetailContent(payload Payload) ui.Node {
	page := decode[receivingDetailPage](pageData(payload))
	lineNodes := make([]ui.Node, 0, len(page.Lines))
	for _, line := range page.Lines {
		lineNodes = append(lineNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(line.ProductSKU+" · "+fallback(line.DiscrepancyReason, "matched"))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(fmt.Sprintf("Expected %d · Actual %d", line.ExpectedQuantity, line.ActualQuantity))),
		))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard(page.Session.ID, fallback(page.Session.DiscrepancySummary, "Receiving session is ready for closeout.")),
			internalWorkflowSection("Receiving detail flow", "Close the session here, then route immediately into the place where the result matters.",
				internalWorkflowCard("Back", "Receiving queue", "Return to the receiving list when you need to move to the next inbound session.", "/app/receiving"),
				internalWorkflowCard("Next", "Inventory verification", "Open inventory after closeout to confirm the adjusted units landed in the expected SKU lanes.", "/app/inventory"),
				internalWorkflowCard("Cross-check", "Warehouse roster", "Re-open the warehouse route when the receiving outcome changes the local pressure plan.", "/app/warehouses"),
			),
			listCard("Receiving lines", lineNodes...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			receivingFormForID(page.Session.ID, payload),
			statCard("Warehouse", page.Session.WarehouseID),
			statCard("Status", page.Session.Status),
		),
	)
}

func commentsContent(payload Payload) ui.Node {
	page := decode[commentList](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard("Buyer follow-up queue", "Treat this route as the manager inbox for customer questions and product-level follow-up, then route deeper catalog or inventory work from here."),
			listCard("Open buyer questions", commentNodes(page.Items)...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			internalWorkflowSection("Buyer follow-up flows", "Move from public feedback into the right internal route without losing the original question context.",
				internalWorkflowCard("Flow 1", "Update marketing copy", "Route unclear product questions into the merch workspace when the issue is messaging, not stock.", "/app/products"),
				internalWorkflowCard("Flow 2", "Check inventory promise", "Open the inventory workspace when the customer is really asking about supply or timing.", "/app/inventory"),
				internalWorkflowCard("Flow 3", "Open warehouse ops", "Use warehouse-native routes when the answer depends on a specific hub or recovery lane.", "/app/warehouses"),
			),
			moderationForm(page.Items, payload),
		),
	)
}

func settingsContent(payload Payload) ui.Node {
	viewNodes := make([]ui.Node, 0, len(payload.SavedViews))
	for _, saved := range payload.SavedViews {
		viewNodes = append(viewNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(saved.Name)),
			html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text(saved.Scope+" · "+saved.SortKey+"/"+saved.SortDirection)),
		))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"},
			statCard("Theme", payload.Preferences.Theme),
			statCard("Locale", payload.Preferences.Locale),
			statCard("Density", payload.Preferences.Density),
			statCard("Warehouse", payload.Preferences.DefaultWarehouse),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			preferenceForm(payload),
			listCard("Saved views", viewNodes...),
		),
	)
}

func mockSignInContent(payload Payload) ui.Node {
	page := decode[mockSignInPage](pageData(payload))
	if len(page.Roles) == 0 {
		page.Roles = []mockSignInRole{{Value: "inventory_manager", Label: "Inventory Manager", Description: "Default Atlas internal operator role."}}
	}
	roleCards := make([]ui.Node, 0, len(page.Roles))
	for _, role := range page.Roles {
		roleCards = append(roleCards,
			html.Form(html.Props{Action: "/auth/mock-sign-in", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
				html.Input(html.Props{Type: "hidden", Name: "role", Value: role.Value}),
				html.Input(html.Props{Type: "hidden", Name: "next", Value: fallback(page.Next, "/app/dashboard")}),
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(role.Label)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(role.Description)),
				html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text("Start session")),
			),
		)
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.85fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard("Mock internal access", fallback(page.Message, "Start a mock Atlas internal session to access the operator console.")),
			listCard("Recovery path",
				infoRow("Next route", fallback(page.Next, "/app/dashboard")),
				infoRow("Session model", "Cookie-backed mock operator role"),
			),
		),
		html.Div(html.Props{Class: "grid gap-4"}, roleCards...),
	)
}

func recoveryContent(payload Payload) ui.Node {
	page := decode[recoveryPage](pageData(payload))
	children := []ui.Node{
		featureCard(fallback(page.Title, "Route recovery"), fallback(page.Message, "Atlas could not resolve the requested route.")),
		html.A(html.Props{Href: fallback(page.RecoveryHref, "/"), Class: "inline-flex w-fit rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(page.RecoveryLabel, "Back to Atlas"))),
	}
	if strings.TrimSpace(page.Detail) != "" {
		children = append(children, listCard("Recovery detail", html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(page.Detail))))
	}
	return html.Section(html.Props{Class: "grid gap-6"}, children...)
}

func fallbackContent(payload Payload) ui.Node {
	entries := sortedMapStrings(payload.Data)
	nodes := make([]ui.Node, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-slate-300"}, html.Text(entry)))
	}
	return listCard("Route data", nodes...)
}

func featureCard(title, copy string) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(copy)),
	)
}

func publicFeatureCard(title, copy string) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-stone-300"}, html.Text(copy)),
	)
}

func publicMetricCard(title, copy string) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.06),rgba(255,255,255,0.03))] p-5 shadow-[0_14px_32px_rgba(0,0,0,0.18)]"},
		html.P(html.Props{Class: "text-xl font-bold tracking-[-0.03em] text-white"}, html.Text(title)),
		html.P(html.Props{Class: "mt-2 text-sm leading-6 text-stone-300"}, html.Text(copy)),
	)
}

func publicSignalPill(label string) ui.Node {
	return html.Div(html.Props{Class: "rounded-full border border-white/12 bg-white/8 px-4 py-2 text-sm font-semibold text-stone-200"}, html.Text(label))
}

func publicStatusClass(status string) string {
	base := "rounded-full border px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.22em]"
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock", "healthy", "approved", "available":
		return base + " border-emerald-400/30 bg-emerald-400/12 text-emerald-200"
	case "low_stock", "pending", "submitted", "in_review":
		return base + " border-amber-400/30 bg-amber-400/12 text-amber-200"
	default:
		return base + " border-white/12 bg-white/8 text-stone-300"
	}
}

func publicStatusLabel(status string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(status)), "_", " ")
}

func catalogPromiseCopy(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock", "healthy", "approved", "available":
		return "Ready for active projects"
	case "low_stock", "pending", "submitted", "in_review":
		return "Best planned with support"
	default:
		return "Atlas-verified catalog entry"
	}
}

func catalogActionPlan(status string) (string, string) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock", "healthy", "approved", "available":
		return "View quote-ready product", "Ready for pricing and project review"
	case "low_stock", "pending", "submitted", "in_review":
		return "View availability options", "Constrained stock, reserve the next realistic window"
	default:
		return "View alternatives", "Best used for notification or substitution planning"
	}
}

func catalogEditorialCopy(product productCard) string {
	if strings.TrimSpace(product.SEODescription) != "" {
		return product.SEODescription
	}
	return "Built to keep material quality, fulfillment posture, and commercial next steps readable in one route."
}

func productCategoryCue(category string) string {
	switch strings.TrimSpace(strings.ToLower(category)) {
	case "desks":
		return "Focused workstation layouts and planning conversations."
	case "storage":
		return "Organization layers that support daily workspace rhythm."
	case "lighting":
		return "Task-ready illumination that finishes a calmer setup."
	case "bundles":
		return "Coordinated system buying for faster project decisions."
	default:
		return "Workspace upgrades with a more considered systems view."
	}
}

func productSupportCue(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock", "healthy", "approved", "available":
		return "Stock posture supports immediate quoting and fulfillment follow-through."
	case "low_stock", "pending", "submitted", "in_review":
		return "Inventory looks constrained, so recovery and quote paths matter more."
	default:
		return "Atlas keeps recovery paths visible when availability needs more coordination."
	}
}

func productBuyingMotion(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock", "healthy", "approved", "available":
		return "Lead with a project quote, then use warehouse context to confirm delivery confidence."
	case "low_stock", "pending", "submitted", "in_review":
		return "Guide the buyer toward reserving the next available units instead of exposing raw replenishment language."
	default:
		return "Preserve buyer intent with a notification path and a clear alternative route when immediate supply is not realistic."
	}
}

func productSupportPlan(status string) (string, string, []string) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock", "healthy", "approved", "available":
		return "Move from shortlist to quote.", "This SKU can support an active buying conversation now. Lead with pricing, timing, and regional delivery confidence.", []string{
			"Use the quote form as the primary action for real project intent.",
			"Keep delivery and fit questions available, but secondary.",
			"Use regional delivery details to confirm timing, not to learn Atlas internals.",
		}
	case "low_stock", "pending", "submitted", "in_review":
		return "Keep the project moving while supply is tight.", "This SKU still has demand value, but the UX should set realistic expectations and preserve buyer intent for the next available units.", []string{
			"Offer a reservation-style action instead of a raw restock request.",
			"Explain that inventory is constrained in plain buyer language.",
			"Keep a human support path nearby for timing or substitution questions.",
		}
	default:
		return "Stay in the loop without losing the product context.", "When immediate fulfillment is not realistic, the product page should shift from conversion to intent capture and alternative discovery.", []string{
			"Use a notification flow instead of implying immediate purchase readiness.",
			"Offer similar options so the buyer is not trapped at a dead end.",
			"Keep specialist support available for spec, finish, and delivery questions.",
		}
	}
}

func productPrimaryActionForm(product productCard, payload Payload) ui.Node {
	switch strings.TrimSpace(strings.ToLower(product.Status)) {
	case "in_stock", "healthy", "approved", "available":
		return publicFormCard("Start a project quote", "Capture pricing, volume, and install timing while this item is ready to support an active project.", "/api/public/products/"+product.Slug+"/quote-requests", "Request pricing", payload.CSRF, []ui.Node{
			publicInput("requester_name", "Name"),
			publicInput("company_name", "Company"),
			publicInput("email", "Email"),
			publicInput("quantity", "Quantity"),
			publicTextarea("note", "Project note"),
		})
	case "low_stock", "pending", "submitted", "in_review":
		return publicFormCard("Reserve upcoming availability", "Hold intent against the next recovery window so the team can plan around constrained stock.", "/api/public/products/"+product.Slug+"/restock-requests", "Reserve availability", payload.CSRF, []ui.Node{
			publicInput("email", "Email"),
			publicInput("preferred_warehouse_id", "Preferred delivery region"),
		})
	default:
		return publicFormCard("Notify me when available", "Stay attached to this SKU without forcing a buyer to think in replenishment or internal operations terms.", "/api/public/products/"+product.Slug+"/restock-requests", "Notify me", payload.CSRF, []ui.Node{
			publicInput("email", "Email"),
			publicInput("preferred_warehouse_id", "Preferred delivery region"),
		})
	}
}

func productQuestionActionForm(product productCard, payload Payload) ui.Node {
	title := "Ask about delivery or fit"
	copy := "Keep practical finish, delivery, and installation questions close to the product decision surface."
	if normalized := strings.TrimSpace(strings.ToLower(product.Status)); normalized != "in_stock" && normalized != "healthy" && normalized != "approved" && normalized != "available" {
		title = "Talk to a specialist"
		copy = "Use a lower-friction support path when the buyer needs help evaluating timing, substitutions, or delivery tradeoffs."
	}
	return publicFormCard(title, copy, "/api/public/products/"+product.Slug+"/comments", "Send question", payload.CSRF, []ui.Node{
		publicInput("author_name", "Name"),
		publicInput("subject", "Subject"),
		publicTextarea("body", "Question"),
	})
}

func productSecondaryActionCard(product productCard) ui.Node {
	status := strings.TrimSpace(strings.ToLower(product.Status))
	if status == "in_stock" || status == "healthy" || status == "approved" || status == "available" {
		return html.A(html.Props{Href: "/warehouses", Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] transition hover:border-amber-300/35 hover:bg-white/10"},
			html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("Check delivery for your region")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Compare regional delivery options before you lock in your project timeline and installation plan.")),
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-amber-700"}, html.Text("See regional delivery options")),
		)
	}
	return html.A(html.Props{Href: "/shop?category=" + url.QueryEscape(product.Category), Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] transition hover:border-amber-300/35 hover:bg-white/10"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("See similar options")),
		html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("If this SKU cannot support the buyer timeline, keep momentum by browsing adjacent in-category systems.")),
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-amber-700"}, html.Text("Browse alternatives")),
	)
}

func warehouseServiceTone(serviceLevel string) string {
	switch strings.TrimSpace(strings.ToLower(serviceLevel)) {
	case "next-day", "priority", "priority coverage":
		return "Fastest fit for tighter delivery windows and higher-priority installs."
	case "two-day", "standard-plus":
		return "Balanced timing for routine commercial installs and steady planning."
	default:
		return "Steady coverage for standard project scheduling and mixed-cart orders."
	}
}

func warehouseRegionCue(region string) string {
	switch strings.TrimSpace(strings.ToLower(region)) {
	case "west", "west coast", "western":
		return "Supports west-coast schedules and shorter transit expectations for nearby teams."
	case "midwest", "central":
		return "Acts as a stabilizing central lane for broader multi-region coverage."
	case "east", "east coast", "eastern":
		return "Helps protect east-coast promise windows and denser delivery expectations."
	default:
		return "Service territory and stock depth stay readable here before buyers open a product-specific availability view."
	}
}

func availabilityStoryCopy(available int, inbound int) string {
	if available > 0 && inbound > 0 {
		return "Current stock covers near-term demand while inbound units support the next replenishment wave."
	}
	if available > 0 {
		return "This hub can support immediate demand from on-hand inventory without relying on inbound receipts."
	}
	if inbound > 0 {
		return "Stock is constrained now, but replenishment is already moving into the lane for recovery planning."
	}
	return "Inventory is currently constrained, so demand recovery and warehouse-specific follow-up are the right next steps."
}

func availabilitySupportPlan(available int, inbound int) (string, string) {
	if available > 0 {
		return "This region can support the project now.", "Use this route to confirm regional promise, then move directly into quote capture while the delivery context is still fresh."
	}
	if inbound > 0 {
		return "Reserve the next inbound wave.", "This region is constrained today, but inbound units are already moving. Preserve buyer intent against this specific hub instead of sending them back to a generic form."
	}
	return "Stay attached to this region.", "Immediate fulfillment is not realistic here, so the right UX is a notification path plus a support channel for alternative planning."
}

func availabilityPrimaryActionForm(availability availabilityPage, payload Payload) ui.Node {
	if availability.Available > 0 {
		return publicFormCard("Request pricing for this region", "Capture pricing and project timing while this warehouse can still support the current demand window.", "/api/public/products/"+availability.Product.Slug+"/quote-requests", "Start quote", payload.CSRF, []ui.Node{
			publicInput("requester_name", "Name"),
			publicInput("company_name", "Company"),
			publicInput("email", "Email"),
			publicInput("quantity", "Quantity"),
			publicTextarea("note", "Project note"),
		})
	}
	title := "Notify me when available"
	copy := "Stay tied to this regional lane without making the buyer restate the product and warehouse context later."
	button := "Notify me"
	if availability.Inbound > 0 {
		title = "Reserve upcoming availability"
		copy = "Hold this buyer against the next inbound recovery window for the current warehouse lane."
		button = "Reserve availability"
	}
	return publicFormCard(title, copy, "/api/public/products/"+availability.Product.Slug+"/restock-requests", button, payload.CSRF, []ui.Node{
		publicInput("email", "Email"),
		publicInputWithValue("preferred_warehouse_id", "Preferred delivery region", availability.Warehouse.ID),
	})
}

func availabilityQuestionActionForm(availability availabilityPage, payload Payload) ui.Node {
	return publicFormCard("Ask about delivery or fit", "Keep regional delivery, installation, and substitution questions attached to the actual availability context.", "/api/public/products/"+availability.Product.Slug+"/comments", "Send question", payload.CSRF, []ui.Node{
		publicInput("author_name", "Name"),
		publicInput("subject", "Subject"),
		publicTextarea("body", "Question"),
	})
}

func statCard(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text(value)),
	)
}

func listCard(title string, children ...ui.Node) ui.Node {
	if len(children) == 0 {
		children = []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No records yet."))}
	}
	content := []ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(title))}
	content = append(content, children...)
	return html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, content...)
}

func formCard(title, action string, csrfToken string, fields []ui.Node) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(title)),
	}
	children = append(children, prependCSRFToken(csrfToken)...)
	children = append(children, fields...)
	children = append(children, submitButton("Submit"))
	return html.Form(html.Props{Action: action, Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, children...)
}

func publicFormCard(title, copy, action, submitLabel string, csrfToken string, fields []ui.Node) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(title)),
	}
	if strings.TrimSpace(copy) != "" {
		children = append(children, html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(copy)))
	}
	children = append(children, prependCSRFToken(csrfToken)...)
	children = append(children, fields...)
	children = append(children, html.Button(html.Props{Type: "submit", Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(fallback(strings.TrimSpace(submitLabel), "Submit"))))
	return html.Form(html.Props{Action: action, Method: "post", Class: "grid gap-4 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, children...)
}

func preferenceForm(payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Operator preferences")),
		inputWithValue("theme", "Theme", payload.Preferences.Theme),
		inputWithValue("locale", "Locale", payload.Preferences.Locale),
		inputWithValue("density", "Density", payload.Preferences.Density),
		inputWithValue("default_warehouse_id", "Default warehouse", payload.Preferences.DefaultWarehouse),
		submitButton("Save preferences"),
	}
	return html.Form(html.Props{Action: "/api/app/preferences", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func moderationForm(items []commentRecord, payload Payload) ui.Node {
	id := ""
	if len(items) > 0 {
		id = items[0].ID
	}
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Review first queued buyer question")),
		inputWithValue("status", "Status", "approved"),
		textareaWithValue("reason", "Reason", "Reviewed by Atlas operations."),
		submitButton("Apply review"),
	}
	return html.Form(html.Props{Action: "/api/app/comments/" + id + "/moderate", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func savedViewForm(payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Create saved view")),
		input("name", "Name"),
		inputWithValue("scope", "Scope", "inventory"),
		inputWithValue("sort_key", "Sort key", "available"),
		inputWithValue("sort_direction", "Sort direction", "asc"),
		inputWithValue("density", "Density", "compact"),
		inputWithValue("warehouse_id", "Warehouse", "new-jersey-hub"),
		textareaWithValue("filters_json", "Filters JSON", `{"warehouse":"new-jersey-hub"}`),
		submitButton("Save view"),
	}
	return html.Form(html.Props{Action: "/api/app/saved-views", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func transferForm(payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Create transfer")),
		inputWithValue("source_warehouse_id", "Source warehouse", "nevada-hub"),
		inputWithValue("destination_warehouse_id", "Destination warehouse", "new-jersey-hub"),
		inputWithValue("reason", "Reason", "Support east-coast promise windows"),
		inputWithValue("recommended_by", "Recommended by", "Atlas Hydration Demo"),
		submitButton("Create transfer"),
	}
	return html.Form(html.Props{Action: "/api/app/transfers", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func thresholdForm(item inventoryRow, payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Adjust thresholds")),
		inputWithValue("warehouse_id", "Warehouse", item.WarehouseID),
		inputWithValue("reorder_point", "Reorder point", "18"),
		inputWithValue("safety_stock", "Safety stock", "9"),
		submitButton("Save threshold"),
	}
	return html.Form(html.Props{Action: "/api/app/inventory/" + item.SKU + "/threshold", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func receivingForm(payload Payload) ui.Node {
	return receivingFormForID("rcv-illinois-001", payload)
}

func receivingFormForID(sessionID string, payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Reconcile receiving")),
		inputWithValue("status", "Status", "closed"),
		textareaWithValue("discrepancy_summary", "Discrepancy summary", "Short shipment recorded and available stock released."),
		submitButton("Close session"),
	}
	return html.Form(html.Props{Action: "/api/app/receiving/" + sessionID + "/reconcile", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func purchaseOrderStatusForm(id string, status string, payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Update order status")),
		inputWithValue("status", "Status", fallback(status, "submitted")),
		textareaWithValue("note", "Note", "Validated by Atlas operations."),
		submitButton("Update order"),
	}
	return html.Form(html.Props{Action: "/api/app/purchase-orders/" + id + "/status", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func prependCSRFToken(token string, children ...ui.Node) []ui.Node {
	if strings.TrimSpace(token) == "" {
		return children
	}
	result := make([]ui.Node, 0, len(children)+1)
	result = append(result, html.Input(html.Props{Type: "hidden", Name: "csrf_token", Value: token}))
	result = append(result, children...)
	return result
}

func input(name, label string) ui.Node {
	return inputWithValue(name, label, "")
}

func publicInput(name, label string) ui.Node {
	return publicInputWithValue(name, label, "")
}

func publicInputWithValue(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
	)
}

func inputWithValue(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}),
	)
}

func textarea(name, label string) ui.Node {
	return textareaWithValue(name, label, "")
}

func publicTextarea(name, label string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Textarea(html.Props{Name: name, Class: "min-h-28 rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
	)
}

func textareaWithValue(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Textarea(html.Props{Name: name, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}, html.Text(value)),
	)
}

func submitButton(label string) ui.Node {
	return html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text(label))
}

func commentNodes(items []commentRecord) []ui.Node {
	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.Subject+" · "+item.Status)),
			html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text(item.ProductSKU+" · "+item.AuthorName+" · "+strings.ReplaceAll(item.AuthorType, "_", " "))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(item.Body)),
		))
	}
	return nodes
}

func internalActionCard(title, copy, href string) ui.Node {
	return html.A(html.Props{Href: href, Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5 transition hover:border-cyan-300/60 hover:bg-white/10"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Open lane")),
	)
}

func internalWorkflowSection(title, copy string, cards ...ui.Node) ui.Node {
	children := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(title)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		),
	}
	children = append(children, html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2 xl:grid-cols-3"}, cards...))
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, children...)
}

func internalWorkflowCard(step, title, copy, href string) ui.Node {
	return html.A(html.Props{Href: href, Class: "grid gap-3 rounded-[1.2rem] border border-cyan-300/15 bg-slate-950/45 p-4 transition hover:border-cyan-300/55 hover:bg-slate-950/65"},
		html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text(step)),
		html.P(html.Props{Class: "text-base font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text("Open workflow")),
	)
}

func transferNodes(items []transferRecord) []ui.Node {
	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.SourceWarehouseID+" → "+item.DestinationWarehouse)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(item.Reason+" · "+item.Status)),
		))
	}
	return nodes
}

func receivingNodes(items []receivingRecord) []ui.Node {
	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.ID+" · "+item.Status)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(fallback(item.DiscrepancySummary, "No discrepancies recorded."))),
		))
	}
	return nodes
}

func infoRow(primary string, secondary string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(primary)),
		html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(secondary)),
	)
}

func activeNavLink(currentPath string, href string) bool {
	current := strings.TrimSpace(currentPath)
	target := strings.TrimSpace(href)
	if current == target {
		return true
	}
	if target == "/" {
		return false
	}
	return strings.HasPrefix(current, target+"/")
}

func routeSummary(path string) string {
	switch {
	case path == "/":
		return "Atlas helps teams discover workspace products, compare regional delivery options, and request pricing without losing context."
	case strings.HasPrefix(path, "/shop/"):
		return "This product page keeps pricing, delivery questions, and next steps together so buyers can decide with confidence."
	case path == "/shop":
		return "Browse workspace systems with clear pricing cues, delivery context, and straightforward next steps."
	case strings.HasPrefix(path, "/warehouses"):
		return "Compare regional delivery options and choose the best fit for your project timing."
	case strings.HasPrefix(path, "/app"):
		return "Internal routes keep saved views, moderation, thresholds, transfers, receiving, and preferences inside one server-backed Atlas console."
	default:
		return "Atlas route payload hydrated successfully from the server bootstrap."
	}
}

func pageData(payload Payload) any {
	if request, ok := StartupRequest(payload, "page"); ok && len(request.Data) > 0 {
		return request.Data["page"]
	}
	if payload.Data == nil {
		return nil
	}
	return payload.Data["page"]
}

func decode[T any](value any) T {
	var result T
	if value == nil {
		return result
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return result
	}
	_ = json.Unmarshal(encoded, &result)
	return result
}

func fallback(value string, defaultValue string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}
	return trimmed
}

func firstQueryValue(values map[string][]string, key string) string {
	if len(values[key]) == 0 {
		return ""
	}
	return values[key][0]
}

func formatPrice(cents int) string {
	return fmt.Sprintf("$%0.2f", float64(cents)/100)
}

func sortedMapStrings(values map[string]any) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %v", key, values[key]))
	}
	return lines
}
