package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
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

type warehouseDirectoryPage struct {
	Items []warehouseCard `json:"items"`
}

type availabilityPage struct {
	Warehouse warehouseCard `json:"warehouse"`
	Product   productCard   `json:"product"`
	Available int           `json:"available"`
	Inbound   int           `json:"inbound"`
	Status    string        `json:"status"`
}

type dashboardPage struct {
	Summary   pageSummary           `json:"summary"`
	Alerts    int                   `json:"alerts"`
	Transfers []transferRecord      `json:"transfers"`
	Receiving []receivingRecord     `json:"receiving"`
	Comments  []commentRecord       `json:"comments"`
	Orders    []purchaseOrderRecord `json:"orders"`
}

type pageSummary struct {
	Headline string            `json:"headline"`
	Items    []pageSummaryItem `json:"items"`
}

type pageSummaryItem struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
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
	Summary pageSummary          `json:"summary"`
	Items   []warehouseOpsRecord `json:"items"`
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
	Summary pageSummary           `json:"summary"`
	Items   []purchaseOrderRecord `json:"items"`
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
	Summary pageSummary     `json:"summary"`
	Items   []commentRecord `json:"items"`
}

type settingsPage struct {
	Summary pageSummary `json:"summary"`
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

type internalShellState struct {
	RouteKey        string
	SummaryLabel    string
	SummaryValue    string
	RouteBadges     map[string]string
	ActiveFilters   []string
	WorkspaceStats  []pageSummaryItem
	ActiveSavedView string
}

type shellPresentationState struct {
	RouteKey         string
	Theme            string
	Locale           string
	Density          string
	DefaultWarehouse string
	Diagnostics      bool
}

const (
	atlasShellPresentationAtomID   = "atlas-shell-presentation"
	atlasShellPresentationStoreKey = "atlas-shell-presentation"
	atlasRouteWorkspaceAtomID      = "atlas-route-workspace"
	atlasShellRootID               = "atlas-shell-root"
	atlasOverlayRootID             = "atlas-overlay-root"
)

func App(parsePayload Payload) ui.Node {
	parseSurface := strings.TrimSpace(parsePayload.Route.Surface)
	parseRootClass := "min-h-screen bg-[radial-gradient(circle_at_top,rgba(245,158,11,0.16),transparent_22%),linear-gradient(180deg,rgba(7,10,18,1),rgba(13,18,30,1)_42%,rgba(7,10,18,1))] text-stone-100"
	parseMainClass := "mx-auto grid w-full max-w-7xl gap-8 px-5 pb-12 pt-6 sm:px-6 lg:px-10"
	if parseSurface != "" && parseSurface != "public" {
		parseRootClass = "min-h-screen bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.12),transparent_28%),linear-gradient(180deg,rgba(2,6,23,1),rgba(15,23,42,1))] text-white"
		parseMainClass = "mx-auto grid w-full max-w-6xl gap-8 px-6 pb-12 pt-6 lg:px-10"
	}
	parseDesiredPresentation := shellPresentationStateFromPayload(parsePayload)
	parsePresentationAtom := useAtlasAtom(atlasShellPresentationAtomID, parseDesiredPresentation)
	parsePresentation := parsePresentationAtom.Get()
	if parsePresentation.RouteKey != parseDesiredPresentation.RouteKey {
		parsePresentation = parseDesiredPresentation
	}
	useAtlasEffect(func() func() {
		if parsePresentationAtom.Get() != parseDesiredPresentation {
			parsePresentationAtom.Set(parseDesiredPresentation)
		}
		return nil
	}, parseDesiredPresentation)
	useAtlasEffect(func() func() {
		_ = persistAtlasSnapshot(atlasShellPresentationStoreKey, atlasShellPresentationAtomID)
		return nil
	}, parsePresentation)
	parseDesiredWorkspace := useAtlasComputed(func() internalShellState {
		return internalShellStateForPayload(parsePayload)
	}, parsePayload.Route.Path, parsePayload.Data, parsePayload.SavedViews).Get()
	parseWorkspaceAtom := useAtlasAtom(atlasRouteWorkspaceAtomID, parseDesiredWorkspace)
	parseShellState := parseWorkspaceAtom.Get()
	if parseShellState.RouteKey != parseDesiredWorkspace.RouteKey {
		parseShellState = parseDesiredWorkspace
	}
	useAtlasEffect(func() func() {
		parseWorkspaceAtom.Set(parseDesiredWorkspace)
		return nil
	}, parseDesiredWorkspace)
	parseToastChannel := useAtlasChannel(atlasShellToastBus)
	parseToastState := useAtlasState(atlasShellToast{})
	parseAnnouncer := ui.UseAnnouncer()
	if parseToastChannel.Ok() {
		parseLatest := parseToastChannel.Get()
		if parseLatest != parseToastState.Get() {
			parseToastState.Set(parseLatest)
		}
	}
	parseRouteAnnouncement := strings.TrimSpace(fallback(parsePayload.Route.Title, parsePayload.Route.Screen))
	parsePreviousRouteAnnouncement := ui.UsePrevious(parseRouteAnnouncement)
	useAtlasEffect(func() func() {
		if !parsePreviousRouteAnnouncement.Ok() || strings.TrimSpace(parseRouteAnnouncement) == "" || parsePreviousRouteAnnouncement.Get() == parseRouteAnnouncement {
			return nil
		}
		parseAnnouncer.Polite("Route changed to " + parseRouteAnnouncement + ".")
		return nil
	}, parseRouteAnnouncement)
	parseNoticeMessage := atlasNoticeMessage(parsePayload)
	parsePreviousNotice := ui.UsePrevious(parseNoticeMessage)
	useAtlasEffect(func() func() {
		if strings.TrimSpace(parseNoticeMessage) == "" || (parsePreviousNotice.Ok() && parsePreviousNotice.Get() == parseNoticeMessage) {
			return nil
		}
		parseAnnouncer.Polite(parseNoticeMessage)
		return nil
	}, parseNoticeMessage)
	parsePreviousToast := ui.UsePrevious(parseToastState.Get())
	useAtlasEffect(func() func() {
		parseCurrent := parseToastState.Get()
		if strings.TrimSpace(parseCurrent.Title) == "" || (parsePreviousToast.Ok() && parsePreviousToast.Get() == parseCurrent) {
			return nil
		}
		parseAnnouncement := parseCurrent.Title
		if parseDetail := strings.TrimSpace(parseCurrent.Detail); parseDetail != "" {
			parseAnnouncement += ". " + parseDetail
		}
		parseAnnouncer.Polite(parseAnnouncement)
		return nil
	}, parseToastState.Get())
	useAtlasEffect(func() func() {
		parseCurrent2 := parseToastState.Get()
		if strings.TrimSpace(parseCurrent2.Title) == "" {
			return nil
		}
		parseStop := make(chan struct{})
		go func(parseExpected atlasShellToast) {
			select {
			case <-time.After(4 * time.Second):
				if parseToastState.Get() == parseExpected {
					parseToastState.Set(atlasShellToast{})
				}
			case <-parseStop:
			}
		}(parseCurrent2)
		return func() {
			close(parseStop)
		}
	}, parseToastState.Get())
	parseChildren := []ui.Node{header(parsePayload, parseShellState)}
	parseMainChildren := []ui.Node{}
	if parseToast := shellToastBanner(parseToastState.Get()); parseToast != nil {
		parseMainChildren = append(parseMainChildren, parseToast)
	}
	if parseBanner := noticeBanner(parsePayload); parseBanner != nil {
		parseMainChildren = append(parseMainChildren, parseBanner)
	}
	if parseDiagnostics := atlasDiagnosticsPanel(parsePayload, parseShellState, parsePresentation); parseDiagnostics != nil {
		parseMainChildren = append(parseMainChildren, parseDiagnostics)
	}
	parseMainChildren = append(parseMainChildren, hero(parsePayload, parseShellState, parsePresentation), pageContent(parsePayload))
	parseChildren = append(parseChildren, html.Main(html.Props{Class: parseMainClass}, parseMainChildren...))
	return html.Div(html.Props{},
		html.Div(html.Props{ID: atlasShellRootID, Class: parseRootClass}, parseChildren...),
		parseAnnouncer.Region(),
		html.Div(html.Props{ID: atlasOverlayRootID}),
	)
}

func header(parsePayload Payload, parseShellState internalShellState) ui.Node {
	parseLinks := []atlasPublicNavLink{
		{Label: publicNavStorefrontLabel, Href: RouteLanding},
		{Label: publicNavShopLabel, Href: RouteCatalog},
		{Label: publicNavWarehousesLabel, Href: RouteWarehouses},
	}
	if parsePayload.User != nil {
		parseLinks = append(parseLinks,
			atlasPublicNavLink{Label: "Dashboard", Href: RouteDashboard},
			atlasPublicNavLink{Label: "Inventory", Href: RouteInventory},
			atlasPublicNavLink{Label: "Products", Href: "/app/products"},
			atlasPublicNavLink{Label: "Warehouses", Href: RouteWarehouseOps},
			atlasPublicNavLink{Label: "Transfers", Href: RouteTransfers},
			atlasPublicNavLink{Label: "Purchase Orders", Href: RoutePurchaseOrders},
			atlasPublicNavLink{Label: "Receiving", Href: RouteReceiving},
			atlasPublicNavLink{Label: "Comments", Href: RouteComments},
			atlasPublicNavLink{Label: "Settings", Href: RouteSettings},
		)
	}
	if parsePayload.Route.Surface == "public" || parsePayload.Route.Surface == "" {
		return publicHeader(parsePayload, parseLinks)
	}
	return internalHeader(parsePayload, parseShellState)
}

type atlasPublicNavLink struct {
	Label string
	Href  string
}

func publicHeader(parsePayload Payload, parseLinks []atlasPublicNavLink) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-public-nav"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		parseIdentity := publicIdentityLabel
		if parsePayload.User != nil {
			parseIdentity = parsePayload.User.DisplayName + " | " + strings.ReplaceAll(parsePayload.User.Role, "_", " ")
		}
		parseDesktopNodes := make([]ui.Node, 0, len(parseLinks))
		parseMobileNodes := make([]ui.Node, 0, len(parseLinks))
		for _, parseLink := range parseLinks {
			parseClassName := "rounded-full border border-white/12 bg-white/6 px-4 py-2.5 text-sm font-medium text-stone-300 shadow-[0_10px_25px_rgba(0,0,0,0.16)] transition hover:border-amber-300/60 hover:bg-white/10 hover:text-white"
			parseMobileClassName := "flex items-center justify-between rounded-[1.35rem] border border-white/10 bg-white/5 px-4 py-4 text-left text-sm font-medium text-stone-200 transition hover:border-amber-300/45 hover:bg-white/10 hover:text-white"
			if activeNavLink(parsePayload.Route.Path, parseLink.Href) {
				parseClassName = "rounded-full border border-amber-300/70 bg-amber-300/12 px-4 py-2.5 text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
				parseMobileClassName = "flex items-center justify-between rounded-[1.35rem] border border-amber-300/60 bg-amber-300/12 px-4 py-4 text-left text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
			}
			parseDesktopNodes = append(parseDesktopNodes, html.A(html.Props{Href: parseLink.Href, Class: parseClassName}, html.Text(parseLink.Label)))
			parseMobileNodes = append(parseMobileNodes, html.A(html.Props{Href: parseLink.Href, Class: parseMobileClassName}, html.Text(parseLink.Label)))
		}
		return html.Header(html.Props{Class: "sticky top-0 z-20 border-b border-white/10 bg-[rgba(7,10,18,0.86)] backdrop-blur-xl"},
			html.Div(html.Props{Class: "mx-auto flex w-full max-w-7xl flex-col gap-4 px-5 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-10"},
				html.Div(html.Props{Class: "flex items-center justify-between gap-4"},
					html.Div(html.Props{Class: "flex flex-col"},
						html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.42em] text-amber-700"}, html.Text(publicBrandLabel)),
						html.P(html.Props{Class: "mt-2 text-sm text-stone-400"}, html.Text(parseIdentity)),
					),
					html.Div(html.Props{Class: "flex items-center gap-3 lg:hidden"},
						html.A(html.Props{Href: RouteCatalog, Class: "inline-flex rounded-full border border-amber-300/60 px-4 py-2 text-sm font-semibold text-amber-100 transition hover:bg-amber-300/12"}, html.Text(publicBrowseShopLabel)),
						html.Button(html.Props{Type: "button", Class: "inline-flex rounded-full border border-white/12 bg-white/8 px-4 py-2 text-sm font-semibold text-stone-100 transition hover:border-amber-300/45 hover:bg-white/12", OnClick: parseOpenDrawer}, html.Text("Menu")),
					),
				),
				html.Nav(html.Props{Class: "hidden flex-wrap items-center gap-2 lg:flex lg:justify-end"}, parseDesktopNodes...),
			),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-300"}, html.Text("Storefront navigation")),
						html.P(html.Props{ID: parseTitleID, Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("Browse Atlas")),
						html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-7 text-stone-300"}, html.Text("Open the public menu to jump between landing, shop, and warehouse discovery routes without losing the current storefront context.")),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-stone-200", OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "grid gap-3"}, parseMobileNodes...),
			)),
		)
	})
}

type atlasInternalNavLink struct {
	Label string
	Href  string
}

type atlasInternalNavGroup struct {
	Label string
	Links []atlasInternalNavLink
}

func internalHeader(parsePayload Payload, parseShellState internalShellState) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-internal-nav"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		parseIdentity := "Public browsing"
		if parsePayload.User != nil {
			parseIdentity = parsePayload.User.DisplayName + " | " + strings.ReplaceAll(parsePayload.User.Role, "_", " ")
		}
		parseContextPills := internalHeaderContextPills(parsePayload, parseShellState)
		parseGroups := internalNavGroups()
		return html.Header(html.Props{Class: "sticky top-0 z-20 border-b border-white/10 bg-slate-950/88 backdrop-blur-xl"},
			html.Div(html.Props{Class: "mx-auto grid w-full max-w-7xl gap-4 px-5 py-4 sm:px-6 lg:px-10"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.42em] text-cyan-300"}, html.Text(publicBrandLabel)),
						html.H1(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fallback(parsePayload.Route.Title, "Atlas workspace"))),
						html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text(routeSummary(parsePayload.Route.Path))),
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-slate-400"}, html.Text(parseIdentity)),
					),
					html.Div(html.Props{Class: "flex items-center gap-3"},
						html.A(html.Props{Href: RouteLanding, Class: "hidden rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200 transition hover:border-cyan-300/50 hover:text-white sm:inline-flex"}, html.Text("View storefront")),
						html.Button(html.Props{Type: "button", Class: "inline-flex items-center rounded-full border border-cyan-300/40 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-400/16 lg:hidden", OnClick: parseOpenDrawer}, html.Text("Workspace nav")),
					),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-2"}, parseContextPills...),
				html.Div(html.Props{Class: "hidden gap-3 lg:grid lg:grid-cols-4"},
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[0]),
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[1]),
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[2]),
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[3]),
				),
				html.Div(html.Props{Class: "flex gap-2 overflow-x-auto pb-1 lg:hidden"},
					internalMobileQuickLinks(parsePayload, parseShellState)...,
				),
			),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Workspace navigation")),
						html.P(html.Props{ID: parseTitleID, Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fallback(parsePayload.Route.Title, "Atlas workspace"))),
						html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-7 text-slate-300"}, html.Text("Use the grouped mobile drawer to jump across overview, catalog, operations, and support routes without losing the current shell context.")),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-2"}, parseContextPills...),
				html.Div(html.Props{Class: "grid gap-3"},
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[0]),
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[1]),
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[2]),
					internalNavGroupCard(parsePayload, parseShellState, parseGroups[3]),
				),
			)),
		)
	})
}

func internalNavGroups() []atlasInternalNavGroup {
	return []atlasInternalNavGroup{
		{
			Label: "Overview",
			Links: []atlasInternalNavLink{
				{Label: "Dashboard", Href: RouteDashboard},
				{Label: "Products", Href: "/app/products"},
			},
		},
		{
			Label: "Stock",
			Links: []atlasInternalNavLink{
				{Label: "Inventory", Href: RouteInventory},
				{Label: "Warehouses", Href: RouteWarehouseOps},
			},
		},
		{
			Label: "Logistics",
			Links: []atlasInternalNavLink{
				{Label: "Transfers", Href: RouteTransfers},
				{Label: "Purchase Orders", Href: RoutePurchaseOrders},
				{Label: "Receiving", Href: RouteReceiving},
			},
		},
		{
			Label: "Support",
			Links: []atlasInternalNavLink{
				{Label: "Comments", Href: RouteComments},
				{Label: "Settings", Href: RouteSettings},
			},
		},
	}
}

func internalHeaderContextPills(parsePayload Payload, parseShellState internalShellState) []ui.Node {
	parseItems := []ui.Node{
		internalHeaderContextPill("Surface", fallback(parsePayload.Route.Surface, "app")),
		internalHeaderContextPill("Warehouse", fallback(parsePayload.Preferences.DefaultWarehouse, "new-jersey-hub")),
	}
	if strings.TrimSpace(parseShellState.SummaryValue) != "" {
		parseItems = append(parseItems, internalHeaderContextPill(fallback(parseShellState.SummaryLabel, "Summary"), parseShellState.SummaryValue))
	}
	if strings.TrimSpace(parseShellState.ActiveSavedView) != "" {
		parseItems = append(parseItems, internalHeaderContextPill("Saved view", parseShellState.ActiveSavedView))
	}
	if len(parseShellState.ActiveFilters) > 0 {
		parseItems = append(parseItems, internalHeaderContextPill("Filters", fmt.Sprintf("%d active", len(parseShellState.ActiveFilters))))
	}
	return parseItems
}

func internalHeaderContextPill(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: internalSurfacePillClass()},
		html.Span(html.Props{Class: "text-slate-400"}, html.Text(parseLabel)),
		html.Span(html.Props{Class: "text-cyan-100"}, html.Text(parseValue)),
	)
}

func internalNavGroupCard(parsePayload Payload, parseShellState internalShellState, parseGroup atlasInternalNavGroup) ui.Node {
	parseLinks := make([]ui.Node, 0, len(parseGroup.Links))
	for _, parseLink := range parseGroup.Links {
		parseLinks = append(parseLinks, internalNavLink(parsePayload.Route.Path, parseShellState, parseLink))
	}
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-4"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text(parseGroup.Label)),
		html.Div(html.Props{Class: "grid gap-2"}, parseLinks...),
	)
}

func internalNavLink(parseCurrentPath string, parseShellState internalShellState, parseLink atlasInternalNavLink) ui.Node {
	parseClassName := "inline-flex items-center justify-between gap-3 " + internalInsetSurfaceClass() + " px-4 py-3 text-sm font-medium text-slate-200 transition hover:border-cyan-300/55 hover:text-white"
	if activeNavLink(parseCurrentPath, parseLink.Href) {
		parseClassName = "inline-flex items-center justify-between gap-3 rounded-[1rem] border border-cyan-300/45 bg-[linear-gradient(180deg,rgba(8,26,42,0.96),rgba(7,14,26,0.98))] px-4 py-3 text-sm font-semibold text-cyan-100 shadow-[0_14px_30px_rgba(34,211,238,0.12)]"
	}
	parseChildren := []ui.Node{
		html.Span(html.Props{}, html.Text(parseLink.Label)),
	}
	if parseBadge := strings.TrimSpace(parseShellState.RouteBadges[parseLink.Href]); parseBadge != "" {
		parseChildren = append(parseChildren, html.Span(html.Props{Class: "rounded-full border border-cyan-300/35 bg-cyan-400/10 px-2 py-1 text-[0.65rem] font-semibold uppercase tracking-[0.16em] text-cyan-100"}, html.Text(parseBadge)))
	}
	return html.A(html.Props{Href: parseLink.Href, Class: parseClassName}, parseChildren...)
}

func internalMobileQuickLinks(parsePayload Payload, parseShellState internalShellState) []ui.Node {
	parseLinks := []atlasInternalNavLink{
		{Label: "Dashboard", Href: RouteDashboard},
		{Label: "Inventory", Href: RouteInventory},
		{Label: "Products", Href: "/app/products"},
		{Label: "Warehouses", Href: RouteWarehouseOps},
	}
	parseNodes := make([]ui.Node, 0, len(parseLinks))
	for _, parseLink := range parseLinks {
		parseNodes = append(parseNodes, internalNavLink(parsePayload.Route.Path, parseShellState, parseLink))
	}
	return parseNodes
}

func internalHeroSurfaceClass() string {
	return "rounded-[2rem] border border-slate-800/95 bg-[linear-gradient(145deg,rgba(8,15,28,0.98),rgba(15,23,42,0.92)_58%,rgba(10,18,32,0.98))] p-8 shadow-[0_32px_90px_rgba(8,15,30,0.38)]"
}

func internalSurfaceCardClass() string {
	return "rounded-[1.55rem] border border-slate-800/95 bg-[linear-gradient(180deg,rgba(15,23,42,0.96),rgba(7,12,24,0.98))] shadow-[0_24px_60px_rgba(2,6,23,0.3)]"
}

func internalInsetSurfaceClass() string {
	return "rounded-[1.15rem] border border-slate-800/90 bg-[linear-gradient(180deg,rgba(10,17,30,0.92),rgba(6,10,20,0.96))] shadow-[inset_0_1px_0_rgba(148,163,184,0.05)]"
}

func internalAccentSurfaceClass() string {
	return "rounded-[1.2rem] border border-cyan-300/20 bg-[linear-gradient(180deg,rgba(8,20,36,0.95),rgba(6,12,24,0.98))] shadow-[0_18px_40px_rgba(2,6,23,0.24)]"
}

func internalSurfacePillClass() string {
	return "inline-flex items-center gap-2 rounded-full border border-slate-800/95 bg-[linear-gradient(180deg,rgba(15,23,42,0.92),rgba(7,12,24,0.96))] px-3 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-slate-200 shadow-[0_14px_28px_rgba(2,6,23,0.18)]"
}

func internalTableContainerClass() string {
	return "overflow-x-auto rounded-[1.4rem] border border-slate-800/95 bg-[linear-gradient(180deg,rgba(9,14,26,0.98),rgba(5,9,18,1))] shadow-[0_24px_60px_rgba(2,6,23,0.28)]"
}

func internalTableHeaderCellClass() string {
	return "px-4 py-3 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"
}

func internalTableRowClass() string {
	return "border-t border-slate-800/90 bg-transparent text-sm text-slate-200"
}

func noticeBanner(parsePayload Payload) ui.Node {
	parseNotice := atlasNoticeMessage(parsePayload)
	if parseNotice == "" {
		return nil
	}
	if parsePayload.Route.Surface == "public" || parsePayload.Route.Surface == "" {
		return html.Div(html.Props{Class: "rounded-[1.75rem] border border-emerald-400/30 bg-emerald-400/10 px-5 py-4 text-sm font-medium text-emerald-100 shadow-[0_18px_40px_rgba(16,185,129,0.12)]"}, html.Text(parseNotice))
	}
	return html.Div(html.Props{Class: "rounded-3xl border border-emerald-300/25 bg-emerald-400/10 px-5 py-4 text-sm text-emerald-100"}, html.Text(parseNotice))
}

func atlasNoticeMessage(parsePayload Payload) string {
	return strings.ReplaceAll(firstQueryValue(parsePayload.Route.Query, "atlas_notice"), "+", " ")
}

func shellToastBanner(parseToast atlasShellToast) ui.Node {
	if strings.TrimSpace(parseToast.Title) == "" {
		return nil
	}
	parseClassName := "grid gap-2 rounded-[1.6rem] border px-5 py-4 text-sm shadow-[0_18px_40px_rgba(2,6,23,0.18)]"
	switch strings.TrimSpace(strings.ToLower(parseToast.Tone)) {
	case "success":
		parseClassName += " border-emerald-400/30 bg-emerald-400/12 text-emerald-50"
	case "warn", "warning", "warm":
		parseClassName += " border-amber-400/30 bg-amber-400/12 text-amber-50"
	case "danger", "error":
		parseClassName += " border-rose-400/30 bg-rose-400/12 text-rose-50"
	default:
		parseClassName += " border-cyan-300/30 bg-cyan-300/12 text-cyan-50"
	}
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em]"}, html.Text(parseToast.Title)),
	}
	if strings.TrimSpace(parseToast.Detail) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "text-sm leading-6"}, html.Text(parseToast.Detail)))
	}
	return html.Div(html.Props{Class: parseClassName}, parseChildren...)
}

func atlasDiagnosticsPanel(parsePayload Payload, parseShellState internalShellState, parsePresentation shellPresentationState) ui.Node {
	if !parsePresentation.Diagnostics || parsePayload.User == nil {
		return nil
	}
	return ui.CreateElement(func() ui.Node {
		parseMetrics := useAtlasViewportMetrics()
		parseThrottled := useAtlasThrottled(parseMetrics, 180*time.Millisecond)
		parseCurrent := parseThrottled.Get()
		parseSnapshot := devtools.SnapshotNow()
		parseStickyState := "top of shell"
		if parseCurrent.ScrollY > 48 {
			parseStickyState = "sticky header engaged"
		}
		parseWorkspaceState := "general shell route"
		if strings.HasPrefix(parsePayload.Route.Path, RouteInventory) {
			parseWorkspaceState = "inventory workspace in view"
		} else if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouseOps) {
			parseWorkspaceState = "warehouse workspace in view"
		}
		parseCopy := "Atlas is sampling shell viewport and sticky-state diagnostics through a throttled stream so scroll and resize bursts do not redraw the panel on every browser event."
		if parseThrottled.Pending() {
			parseCopy = "Atlas is rate-limiting the latest viewport measurement while scroll or resize events are still arriving."
		}
		parseSnapshotRoute := fallback(parseSnapshot.Route.Path, parsePayload.Route.Path)
		parseSnapshotSummary := fmt.Sprintf(
			"SnapshotNow: route=%s | fibers=%d | diagnostics=%d | cache=%d | loaders=%d",
			parseSnapshotRoute,
			parseSnapshot.Stats.TotalFibers,
			len(parseSnapshot.Diagnostics),
			len(parseSnapshot.Cache),
			len(parseSnapshot.Route.Loaders),
		)
		return ui.Fragment(
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-cyan-300/20 bg-cyan-400/8 p-5"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-cyan-200"}, html.Text("Atlas diagnostics")),
					html.P(html.Props{Class: "text-sm leading-6 text-cyan-50"}, html.Text(parseCopy)),
				),
				html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
					statCard("Scroll Y", fmt.Sprintf("%dpx", parseCurrent.ScrollY)),
					statCard("Viewport", fmt.Sprintf("%dx%d", parseCurrent.Width, parseCurrent.Height)),
					statCard("Sticky shell", parseStickyState),
					statCard("Samples", fmt.Sprintf("%d", parseCurrent.SampleCount)),
				),
				html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
					statCard("Route", fallback(parsePayload.Route.Path, "/")),
					statCard("Workspace", parseWorkspaceState),
					statCard("Measured", fallback(parseCurrent.MeasuredAtUTC, "Awaiting browser metrics")),
				),
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-100/80"}, html.Text(parseSnapshotSummary)),
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-100/80"}, html.Text("Shell summary: "+fallback(parseShellState.SummaryLabel, "route")+" | "+fallback(parseShellState.SummaryValue, "n/a"))),
			),
			ui.CreateElement(devtools.Panel, devtools.PanelProps{
				Title:           "Atlas Devtools",
				InitiallyOpen:   false,
				RefreshInterval: 750 * time.Millisecond,
				MaxDepth:        5,
			}),
		)
	})
}

func hero(parsePayload Payload, parseShellState internalShellState, parsePresentation shellPresentationState) ui.Node {
	if parsePayload.Route.Surface == "public" || parsePayload.Route.Surface == "" {
		return renderPublicHero(parsePayload)
	}
	parseMeta := []ui.Node{
		statCard("Surface", fallback(parsePayload.Route.Surface, "public")),
		statCard("Theme", fallback(parsePresentation.Theme, "dark")),
		statCard("Locale", fallback(parsePresentation.Locale, "en")),
		statCard("Density", fallback(parsePresentation.Density, "compact")),
	}
	parseMeta = append(parseMeta, statCard("Warehouse", fallback(parsePresentation.DefaultWarehouse, "new-jersey-hub")))
	if parsePresentation.Diagnostics {
		parseMeta = append(parseMeta, statCard("Diagnostics", "on"))
	}
	if strings.TrimSpace(parseShellState.ActiveSavedView) != "" {
		parseMeta = append(parseMeta, statCard("Saved view", parseShellState.ActiveSavedView))
	}
	if len(parseShellState.ActiveFilters) > 0 {
		parseMeta = append(parseMeta, statCard("Filters", fmt.Sprintf("%d active", len(parseShellState.ActiveFilters))))
	}
	if strings.TrimSpace(parseShellState.SummaryValue) != "" {
		parseMeta = append([]ui.Node{statCard(fallback(parseShellState.SummaryLabel, "Shell summary"), parseShellState.SummaryValue)}, parseMeta...)
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.8fr)]"},
		html.Div(html.Props{Class: internalHeroSurfaceClass()},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.35em] text-cyan-300"}, html.Text(fallback(parsePayload.Route.Screen, "route"))),
			html.H1(html.Props{Class: "mt-4 text-4xl font-black tracking-tight text-white lg:text-5xl"}, html.Text(fallback(parsePayload.Route.Title, "Atlas"))),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-base leading-7 text-slate-300"}, html.Text(routeSummary(parsePayload.Route.Path))),
		),
		html.Div(html.Props{Class: "grid gap-4"}, parseMeta...),
	)
}

func shellPresentationStateFromPayload(parsePayload Payload) shellPresentationState {
	return shellPresentationState{
		RouteKey:         shellRouteKey(parsePayload.Route.Path, parsePayload.Route.Query),
		Theme:            parsePayload.Theme.Mode,
		Locale:           parsePayload.I18n.Locale,
		Density:          parsePayload.Preferences.Density,
		DefaultWarehouse: parsePayload.Preferences.DefaultWarehouse,
		Diagnostics:      strings.EqualFold(strings.TrimSpace(firstQueryValue(parsePayload.Route.Query, "diag")), "1"),
	}
}

func currentShellPresentationState(parsePayload Payload) shellPresentationState {
	return useAtlasAtom(atlasShellPresentationAtomID, shellPresentationStateFromPayload(parsePayload)).Get()
}

func currentRouteWorkspaceState(parsePayload Payload) internalShellState {
	return useAtlasAtom(atlasRouteWorkspaceAtomID, internalShellStateForPayload(parsePayload)).Get()
}

func shellRouteKey(parsePath string, parseQuery map[string][]string) string {
	parseValues := url.Values{}
	for parseKey, parseItems := range parseQuery {
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	if parseEncoded := parseValues.Encode(); parseEncoded != "" {
		return parsePath + "?" + parseEncoded
	}
	return parsePath
}

func internalShellStateForPayload(parsePayload Payload) internalShellState {
	parseState := internalShellState{
		RouteKey:    shellRouteKey(parsePayload.Route.Path, parsePayload.Route.Query),
		RouteBadges: map[string]string{},
	}
	if parsePayload.User == nil {
		return parseState
	}
	parsePath := strings.TrimSpace(parsePayload.Route.Path)
	switch {
	case parsePath == RouteDashboard:
		parsePage := decode[dashboardPage](pageData(parsePayload))
		parseState.SummaryLabel = "Open alerts"
		parseState.SummaryValue = fmt.Sprintf("%d", parsePage.Alerts)
		parseState.RouteBadges[RouteDashboard] = fmt.Sprintf("%d", parsePage.Alerts)
		parseState.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(parsePage.Transfers))
		parseState.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(parsePage.Receiving))
		parseState.RouteBadges[RouteComments] = fmt.Sprintf("%d", len(parsePage.Comments))
		parseState.WorkspaceStats = []pageSummaryItem{
			{Label: "Alerts", Value: fmt.Sprintf("%d", parsePage.Alerts)},
			{Label: "Transfers", Value: fmt.Sprintf("%d", len(parsePage.Transfers))},
			{Label: "Receiving", Value: fmt.Sprintf("%d", len(parsePage.Receiving))},
			{Label: "Comments", Value: fmt.Sprintf("%d", len(parsePage.Comments))},
		}
	case parsePath == "/app/products":
		parsePage2 := decode[productCMSPageData](pageData(parsePayload))
		parseTotal := parsePage2.Total
		if parseTotal == 0 {
			parseTotal = len(parsePage2.Items)
		}
		parseState.SummaryLabel = "Catalog items"
		parseState.SummaryValue = fmt.Sprintf("%d", parseTotal)
		parseState.RouteBadges["/app/products"] = fmt.Sprintf("%d", parseTotal)
	case parsePath == RouteInventory:
		parsePage3 := decode[inventoryCMSPage](pageData(parsePayload))
		parseWorkspace := inventoryWorkspaceSnapshotFromRows(parsePage3.Items)
		parseState.SummaryLabel = "Risk lanes"
		parseState.SummaryValue = fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)
		parseState.RouteBadges[RouteInventory] = fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)
		parseState.ActiveFilters = inventoryFilterSummaryLabels(parsePage3.Filters)
		parseState.ActiveSavedView = matchingInventorySavedViewName(parsePayload, parsePage3.Filters)
		parseState.WorkspaceStats = []pageSummaryItem{
			{Label: "Visible SKUs", Value: fmt.Sprintf("%d", len(parseWorkspace.Summaries))},
			{Label: "Visible lanes", Value: fmt.Sprintf("%d", parseWorkspace.Rollup.VisibleLanes)},
			{Label: "Risk lanes", Value: fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", parseWorkspace.Rollup.InboundUnits)},
		}
	case strings.HasPrefix(parsePath, RouteInventory+"/"):
		parsePage4 := decode[inventoryDetailPage](pageData(parsePayload))
		parseRollup := inventoryRollupFromRows(parsePage4.Rows)
		parseState.SummaryLabel = "Tracked lanes"
		parseState.SummaryValue = fmt.Sprintf("%d", parseRollup.VisibleLanes)
		parseState.RouteBadges[RouteInventory] = fmt.Sprintf("%d", parseRollup.RiskLanes)
	case parsePath == RouteWarehouseOps:
		parsePage5 := decode[warehouseOpsList](pageData(parsePayload))
		parseRiskCount := countRiskWarehouses(parsePage5.Items)
		parseState.SummaryLabel = "Risk facilities"
		parseState.SummaryValue = fmt.Sprintf("%d", parseRiskCount)
		parseState.RouteBadges[RouteWarehouseOps] = fmt.Sprintf("%d", parseRiskCount)
	case strings.HasPrefix(parsePath, RouteWarehouseOps+"/"):
		parsePage6 := decode[warehouseInventoryDetailPage](pageData(parsePayload))
		if strings.TrimSpace(parsePage6.Warehouse.ID) != "" {
			parseWorkspace2 := warehouseDetailWorkspaceSnapshotFromPage(parsePage6)
			parseState.SummaryLabel = "Warehouse filters"
			parseState.SummaryValue = fmt.Sprintf("%d", len(warehouseFilterSummaryLabels(parsePage6.Filters)))
			parseState.RouteBadges[RouteWarehouseOps] = fmt.Sprintf("%d", parseWorkspace2.RiskLanes)
			parseState.ActiveFilters = warehouseFilterSummaryLabels(parsePage6.Filters)
			parseState.WorkspaceStats = []pageSummaryItem{
				{Label: "Visible items", Value: fmt.Sprintf("%d", len(parsePage6.Inventory))},
				{Label: "Risk lanes", Value: fmt.Sprintf("%d", parseWorkspace2.RiskLanes)},
				{Label: "Open orders", Value: fmt.Sprintf("%d", len(parsePage6.Orders))},
				{Label: "Weekly demand", Value: fmt.Sprintf("%d", parseWorkspace2.TotalDemand)},
			}
		}
	case parsePath == RouteTransfers:
		parsePage7 := decode[transferList](pageData(parsePayload))
		parseState.SummaryLabel = "Active transfers"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage7.Items))
		parseState.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(parsePage7.Items))
	case strings.HasPrefix(parsePath, RouteTransfers+"/"):
		parsePage8 := decode[transferDetailPage](pageData(parsePayload))
		parseState.SummaryLabel = "Transfer lines"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage8.Lines))
		parseState.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(parsePage8.Lines))
	case parsePath == RoutePurchaseOrders:
		parsePage9 := decode[purchaseOrderList](pageData(parsePayload))
		parseState.SummaryLabel = "Open purchase orders"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage9.Items))
		parseState.RouteBadges[RoutePurchaseOrders] = fmt.Sprintf("%d", len(parsePage9.Items))
	case strings.HasPrefix(parsePath, RoutePurchaseOrders+"/"):
		parsePage10 := decode[purchaseOrderDetailPage](pageData(parsePayload))
		parseState.SummaryLabel = "Inbound lines"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage10.Lines))
		parseState.RouteBadges[RoutePurchaseOrders] = fmt.Sprintf("%d", len(parsePage10.Lines))
	case parsePath == RouteReceiving:
		parsePage11 := decode[receivingList](pageData(parsePayload))
		parseState.SummaryLabel = "Receiving sessions"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage11.Items))
		parseState.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(parsePage11.Items))
	case strings.HasPrefix(parsePath, RouteReceiving+"/"):
		parsePage12 := decode[receivingDetailPage](pageData(parsePayload))
		parseState.SummaryLabel = "Receiving lines"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage12.Lines))
		parseState.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(parsePage12.Lines))
	case parsePath == RouteComments:
		parsePage13 := decode[commentList](pageData(parsePayload))
		parseState.SummaryLabel = "Visible comments"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage13.Items))
		parseState.RouteBadges[RouteComments] = fmt.Sprintf("%d", len(parsePage13.Items))
	case parsePath == RouteSettings:
		parseState.SummaryLabel = "Saved views"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePayload.SavedViews))
		parseState.RouteBadges[RouteSettings] = fmt.Sprintf("%d", len(parsePayload.SavedViews))
	}
	return parseState
}

func countRiskWarehouses(parseItems []warehouseOpsRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if parseItem.RiskCount > 0 {
			parseCount++
		}
	}
	return parseCount
}

func inventoryFilterSummaryLabels(parseFilters map[string]string) []string {
	parseLabels := []string{}
	if parseValue := strings.TrimSpace(parseFilters["q"]); parseValue != "" {
		parseLabels = append(parseLabels, "Search: "+parseValue)
	}
	if parseValue2 := strings.TrimSpace(parseFilters["warehouse"]); parseValue2 != "" && !strings.EqualFold(parseValue2, "all") {
		parseLabels = append(parseLabels, "Warehouse: "+parseValue2)
	}
	if parseValue3 := strings.TrimSpace(parseFilters["status"]); parseValue3 != "" && !strings.EqualFold(parseValue3, "all") {
		parseLabels = append(parseLabels, "Status: "+strings.ReplaceAll(parseValue3, "_", " "))
	}
	if parseValue4 := strings.TrimSpace(parseFilters["sort"]); parseValue4 != "" {
		parseLabels = append(parseLabels, "Sort: "+parseValue4)
	}
	return parseLabels
}

func warehouseFilterSummaryLabels(parseFilters map[string]string) []string {
	parseLabels := []string{}
	if parseValue := strings.TrimSpace(parseFilters["q"]); parseValue != "" {
		parseLabels = append(parseLabels, "Search: "+parseValue)
	}
	if parseValue2 := strings.TrimSpace(parseFilters["status"]); parseValue2 != "" && !strings.EqualFold(parseValue2, "all") {
		parseLabels = append(parseLabels, "Status: "+strings.ReplaceAll(parseValue2, "_", " "))
	}
	if parseValue3 := strings.TrimSpace(parseFilters["sort"]); parseValue3 != "" {
		parseLabels = append(parseLabels, "Sort: "+parseValue3)
	}
	return parseLabels
}

func matchingInventorySavedViewName(parsePayload Payload, parseFilters map[string]string) string {
	parseCurrentWarehouse := strings.TrimSpace(parseFilters["warehouse"])
	parseCurrentStatus := strings.TrimSpace(parseFilters["status"])
	parseCurrentSort := strings.TrimSpace(parseFilters["sort"])
	for _, parseSaved := range parsePayload.SavedViews {
		if !strings.EqualFold(strings.TrimSpace(parseSaved.Scope), "inventory") {
			continue
		}
		parseSavedWarehouse := strings.TrimSpace(parseSaved.Filters["warehouse"])
		parseSavedStatus := strings.TrimSpace(parseSaved.Filters["status"])
		if parseSavedStatus == "" {
			parseSavedStatus = strings.TrimSpace(parseSaved.Filters["stock-health"])
		}
		isParseMatchesWarehouse := parseSavedWarehouse == "" || strings.EqualFold(parseSavedWarehouse, parseCurrentWarehouse)
		isParseMatchesStatus := parseSavedStatus == "" || strings.EqualFold(parseSavedStatus, parseCurrentStatus)
		isParseMatchesSort := strings.TrimSpace(parseSaved.SortKey) == "" || strings.EqualFold(strings.TrimSpace(parseSaved.SortKey), parseCurrentSort)
		if isParseMatchesWarehouse && isParseMatchesStatus && isParseMatchesSort {
			return parseSaved.Name
		}
	}
	return ""
}

func pageContent(parsePayload Payload) ui.Node {
	if parsePayload.Route.Screen == "mock-sign-in" {
		return mockSignInContent(parsePayload)
	}
	if parsePayload.Route.Screen == "recovery" {
		return recoveryContent(parsePayload)
	}
	switch parsePayload.Route.Path {
	case RouteLanding:
		return renderLandingContent()
	case RouteCatalog:
		return renderCatalogContent(decode[catalogPage](pageData(parsePayload)))
	case RouteWarehouses:
		return renderWarehouseDirectoryContent(decode[warehouseDirectoryPage](pageData(parsePayload)))
	case RouteInventory:
		return inventoryContent(parsePayload)
	case "/app/products":
		return productsCMSContent(parsePayload)
	case RouteDashboard:
		return dashboardContent(parsePayload)
	case RouteWarehouseOps:
		return warehouseOpsContent(parsePayload)
	case RouteTransfers:
		return transfersContent(parsePayload)
	case RoutePurchaseOrders:
		return purchaseOrdersContent(parsePayload)
	case RouteReceiving:
		return receivingContent(parsePayload)
	case RouteComments:
		return commentsContent(parsePayload)
	case RouteSettings:
		return settingsContent(parsePayload)
	default:
		if strings.HasPrefix(parsePayload.Route.Path, RouteCatalog+"/") {
			return renderProductContent(decode[productDetailPage](pageData(parsePayload)), parsePayload)
		}
		if strings.Contains(parsePayload.Route.Path, "/availability/") {
			return renderAvailabilityContent(decode[availabilityPage](pageData(parsePayload)), parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouseOps+"/") && strings.Contains(parsePayload.Route.Path, "/items/") {
			return warehouseOpsDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouseOps+"/") {
			return warehouseOpsDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, "/app/products/") {
			return productEditorContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteInventory+"/") {
			return skuContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteTransfers+"/") {
			return transferDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RoutePurchaseOrders+"/") {
			return purchaseOrderDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteReceiving+"/") {
			return receivingDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouses+"/") {
			parsePage := decode[warehouseDetailPage](pageData(parsePayload))
			if strings.TrimSpace(parsePage.Warehouse.ID) == "" {
				parsePage.Warehouse = decode[warehouseCard](pageData(parsePayload))
			}
			return renderWarehouseDetailContent(parsePage, parsePayload)
		}
		return fallbackContent(parsePayload)
	}
}

func dashboardContent(parsePayload Payload) ui.Node {
	parsePage := decode[dashboardPage](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6"},
		dashboardSummaryBand(parsePage),
		html.Div(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.18fr)_minmax(22rem,0.82fr)] xl:items-start"},
			html.Div(html.Props{Class: "grid gap-6"},
				dashboardActionCluster(),
				dashboardActivityFeed(parsePage),
			),
			html.Div(html.Props{Class: "grid gap-6"},
				dashboardAttentionPanel(parsePage),
				dashboardPurchaseOrderSummary(parsePage.Orders),
			),
		),
	)
}

func dashboardSummaryBand(parsePage dashboardPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Dashboard summary band")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("The dashboard should read like a triage surface first: one summary band, one action cluster, one attention stack, one activity feed, and one purchase-order watch panel.")),
		),
		routeSummaryStrip(parsePage.Summary),
	)
}

func dashboardActionCluster() ui.Node {
	return internalWorkflowSection("Action cluster", "Start from the operator task that needs motion right now instead of treating the dashboard like a passive KPI wall.",
		internalWorkflowCard("Action 1", "Open low-stock inventory view", "Jump directly into promise-risk and low-stock lanes that need threshold, transfer, or replenishment decisions.", "/app/inventory?status=promise_risk"),
		internalWorkflowCard("Action 2", "Create transfer", "Move into balancing work when the issue is warehouse coverage, not vendor replenishment.", "/app/transfers"),
		internalWorkflowCard("Action 3", "Resume receiving session", "Close inbound discrepancies before they continue distorting availability posture.", "/app/receiving"),
		internalWorkflowCard("Action 4", "Review pending comments", "Route buyer questions and moderation backlog through the inbox without losing the dashboard handoff context.", "/app/comments?status=pending"),
	)
}

func dashboardAttentionPanel(parsePage dashboardPage) ui.Node {
	parsePendingComments := dashboardCommentStatusCount(parsePage.Comments, "pending")
	parseFlaggedComments := dashboardCommentStatusCount(parsePage.Comments, "flagged")
	parseOpenReceiving := dashboardOpenReceivingCount(parsePage.Receiving)
	parseSubmittedOrders := dashboardPurchaseOrderStatusCount(parsePage.Orders, "submitted")
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("High-attention panel")),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fmt.Sprintf("%d alerts need a route decision.", parsePage.Alerts))),
			html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text("These are the work queues most likely to change customer promise, inbound readiness, or moderation posture if an operator waits too long.")),
		),
		html.Div(html.Props{Class: "grid gap-3"},
			dashboardAttentionLink("Pending comments", fmt.Sprintf("%d waiting", parsePendingComments), "Review questions and moderation decisions that are shaping buyer follow-up right now.", "/app/comments?status=pending"),
			dashboardAttentionLink("Flagged comments", fmt.Sprintf("%d flagged", parseFlaggedComments), "Handle risky or unclear public notes before they create merch or support confusion.", "/app/comments?status=flagged"),
			dashboardAttentionLink("Receiving closeout", fmt.Sprintf("%d open sessions", parseOpenReceiving), "Resolve discrepancies and close receiving sessions so inbound stock can become trustworthy availability.", "/app/receiving"),
			dashboardAttentionLink("Submitted purchase orders", fmt.Sprintf("%d need review", parseSubmittedOrders), "Move draft or submitted vendor work forward before the replenishment lane stalls.", "/app/purchase-orders"),
		),
	)
}

func dashboardAttentionLink(parseTitle, parseValue, parseCopy, parseHref string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
		html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseTitle)),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(parseValue)),
		),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
	)
}

func dashboardActivityFeed(parsePage dashboardPage) ui.Node {
	parseItems := dashboardActivityItems(parsePage)
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.A(html.Props{Href: parseItem.Href, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(parseItem.Kicker)),
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.18em] text-slate-400"}, html.Text(parseItem.Meta)),
			),
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.Title)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Detail)),
		))
	}
	if len(parseNodes) == 0 {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text("No operator activity queued.")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Comments, transfers, receiving sessions, and purchase orders will populate this feed as soon as Atlas has active work.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Activity feed")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The dashboard feed should show the latest moderation, transfer, receiving, and purchase-order motion in one scan instead of making operators open four routes just to check momentum.")),
		),
		html.Div(html.Props{Class: "grid gap-3"}, parseNodes...),
	)
}

type dashboardActivityItem struct {
	Kicker string
	Title  string
	Detail string
	Meta   string
	Href   string
}

func dashboardActivityItems(parsePage dashboardPage) []dashboardActivityItem {
	parseItems := []dashboardActivityItem{}
	for parseIndex, parseItem := range parsePage.Comments {
		if parseIndex >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Moderation",
			Title:  fallback(parseItem.Subject, "Buyer question"),
			Detail: fallback(parseItem.Body, "Public feedback needs a routing decision."),
			Meta:   strings.ReplaceAll(fallback(parseItem.Status, "pending"), "_", " "),
			Href:   "/app/comments",
		})
	}
	for parseIndex2, parseItem2 := range parsePage.Transfers {
		if parseIndex2 >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Transfer",
			Title:  parseItem2.SourceWarehouseID + " to " + parseItem2.DestinationWarehouse,
			Detail: fallback(parseItem2.Reason, "Balancing action in flight."),
			Meta:   strings.ReplaceAll(fallback(parseItem2.Status, "submitted"), "_", " "),
			Href:   "/app/transfers/" + parseItem2.ID,
		})
	}
	for parseIndex3, parseItem3 := range parsePage.Receiving {
		if parseIndex3 >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Receiving",
			Title:  parseItem3.ID,
			Detail: fallback(parseItem3.DiscrepancySummary, "Inbound session still needs closeout."),
			Meta:   strings.ReplaceAll(fallback(parseItem3.Status, "open"), "_", " "),
			Href:   "/app/receiving/" + parseItem3.ID,
		})
	}
	for parseIndex4, parseItem4 := range parsePage.Orders {
		if parseIndex4 >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Purchase order",
			Title:  fallback(parseItem4.VendorName, parseItem4.ID),
			Detail: fallback(parseItem4.PriorityNote, "Vendor replenishment still in motion."),
			Meta:   strings.ReplaceAll(fallback(parseItem4.Status, "submitted"), "_", " "),
			Href:   "/app/purchase-orders/" + parseItem4.ID,
		})
	}
	return parseItems
}

func dashboardPurchaseOrderSummary(parseOrders []purchaseOrderRecord) ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseOrders))
	for parseIndex, parseItem := range parseOrders {
		if parseIndex >= 4 {
			break
		}
		parseNodes = append(parseNodes, html.A(html.Props{Href: "/app/purchase-orders/" + parseItem.ID, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(fallback(parseItem.VendorName, parseItem.ID))),
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(strings.ReplaceAll(fallback(parseItem.Status, "submitted"), "_", " "))),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(fallback(parseItem.PriorityNote, "Vendor replenishment still needs operator follow-through."))),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.18em] text-slate-400"}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID)+" | ETA "+fallback(parseItem.ETA, "pending"))),
		))
	}
	if len(parseNodes) == 0 {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text("No purchase orders are open.")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Atlas will list the highest-priority vendor work here once replenishment needs a formal PO lane.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Purchase-order summary")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep vendor work visible from the dashboard so replenishment planning does not disappear behind the logistics route boundary.")),
		),
		html.Div(html.Props{Class: "grid gap-3"}, parseNodes...),
	)
}

func dashboardCommentStatusCount(parseItems []commentRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func dashboardOpenReceivingCount(parseItems []receivingRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		parseStatus := strings.TrimSpace(strings.ToLower(parseItem.Status))
		if parseStatus != "reconciled" && parseStatus != "closed" {
			parseCount++
		}
	}
	return parseCount
}

func dashboardPurchaseOrderStatusCount(parseItems []purchaseOrderRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func inventoryContent(parsePayload Payload) ui.Node {
	return inventoryCMSContent(parsePayload)
}

func skuContent(parsePayload Payload) ui.Node {
	return inventoryDetailContent(parsePayload)
}

func warehouseOpsContent(parsePayload Payload) ui.Node {
	parsePage := decode[warehouseOpsList](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.16fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps, Current: true},
			),
			warehouseOpsSummaryBand(parsePage),
			warehouseOpsActionCluster(),
			warehouseOpsTable(parsePage.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryRailCard("Cross-route handoffs", "Warehouse work should hand off cleanly into inventory, purchasing, and receiving instead of trapping the operator in one facility view.",
				html.Div(html.Props{Class: "grid gap-3"},
					inventoryActionCard("Inventory risk view", "Compare cross-warehouse SKU pressure before changing a local lane.", "/app/inventory?status=promise_risk"),
					inventoryActionCard("Create replenishment", "Escalate from warehouse pressure into vendor-side inbound planning when balancing is no longer enough.", "/app/purchase-orders"),
					inventoryActionCard("Close inbound work", "Return to receiving after warehouse recovery starts moving through inbound confirmation.", "/app/receiving"),
				),
			),
			inventoryRailCard("What this route controls", "Use the warehouse workspace to choose the right facility, confirm backlog and staffing posture, then branch into nested item work only after the facility context is clear.",
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The warehouse list is now table-first on purpose: it should answer which facility is under pressure before the operator opens a SKU, purchase order, or receiving session.")),
			),
		),
	)
}

func warehouseOpsSummaryBand(parsePage warehouseOpsList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Warehouse operations shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat the warehouse route as a facility triage board first: visible backlog pressure, clear service posture, and direct drill-ins into the facility that actually needs action.")),
		),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Facilities", fmt.Sprintf("%d active", len(parsePage.Items))),
			statCard("Available", fmt.Sprintf("%d units", totalWarehouseAvailable(parsePage.Items))),
			statCard("Inbound", fmt.Sprintf("%d units", totalWarehouseInbound(parsePage.Items))),
			statCard("Risk facilities", fmt.Sprintf("%d flagged", countRiskWarehouses(parsePage.Items))),
		),
	)
}

func warehouseOpsActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Lead with the facility action that matters now: open a roster, add an item, review flagged lanes, or escalate into replenishment planning.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
			inventoryActionCard("Open a roster", "Choose the facility first when the problem is local backlog, staffing, or regional supply pressure.", "/app/warehouses"),
			inventoryActionCard("Add warehouse item", "Seed a new managed item directly inside the facility workspace rather than bouncing through catalog-first flows.", "/app/warehouses/new-jersey-hub#warehouse-create-item"),
			inventoryActionCard("Review flagged lanes", "Move into a facility detail page already filtered to the lanes that need action.", "/app/warehouses/new-jersey-hub?status=promise_risk"),
			inventoryActionCard("Order more units", "Open replenishment only after the warehouse context proves inbound recovery is the right move.", "/app/purchase-orders"),
		),
	)
}

func warehouseOpsTable(parseItems []warehouseOpsRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, warehouseOpsTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No warehouses are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Facility table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scan service posture, backlog, inbound exposure, and risk count in one dense table before drilling into a specific facility workspace.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d facilities", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Warehouse")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Pressure")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Inbound")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Risks")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func warehouseOpsTableRow(parseItem warehouseOpsRecord) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + parseItem.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.Name)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseItem.Region+" | "+parseItem.ServiceLevel)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Focus)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseItem.Pressure+" | "+parseItem.Backlog)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Inbound))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d active", parseItem.RiskCount))),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + parseItem.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open workspace")),
				html.A(html.Props{Href: "/app/warehouses/" + parseItem.ID + "#warehouse-replenishment", Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open replenishment")),
			),
		),
	)
}

func totalWarehouseAvailable(parseItems []warehouseOpsRecord) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Available
	}
	return parseTotal
}

func totalWarehouseInbound(parseItems []warehouseOpsRecord) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Inbound
	}
	return parseTotal
}

func warehouseOpsDetailContent(parsePayload Payload) ui.Node {
	return warehouseInventoryDetailContent(parsePayload)
}

func transfersContent(parsePayload Payload) ui.Node {
	parsePage := decode[transferList](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			transfersSummaryBand(parsePage),
			transfersActionCluster(),
			transfersTable(parsePage.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			transferForm(parsePayload),
			inventoryRailCard("Transfer handoffs", "Transfers should stay tied to the larger warehouse recovery path rather than becoming an isolated page-level queue.",
				html.Div(html.Props{Class: "grid gap-3"},
					inventoryActionCard("Review warehouse pressure", "Check which warehouse is starving before you commit a transfer lane.", "/app/warehouses"),
					inventoryActionCard("Open inventory pressure", "Validate that the SKU really needs a rebalance instead of a replenishment order.", "/app/inventory?status=promise_risk"),
					inventoryActionCard("Confirm receiving follow-through", "Keep the receiving route in view for downstream reconciliation once the transfer lands.", "/app/receiving"),
				),
			),
		),
	)
}

func transfersSummaryBand(parsePage transferList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Transfer shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat transfers as a balancing board: source and destination lanes, recommendation context, and direct handoff into receiving once the movement is committed.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Transfers", fmt.Sprintf("%d active", len(parsePage.Items))),
			statCard("Pending", fmt.Sprintf("%d queued", countTransferStatus(parsePage.Items, "pending"))),
			statCard("Approved", fmt.Sprintf("%d moving", countTransferStatus(parsePage.Items, "approved"))),
			statCard("Cancelled", fmt.Sprintf("%d dropped", countTransferStatus(parsePage.Items, "cancelled"))),
		),
	)
}

func transfersActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Lead transfer work from the warehouse and inventory pressure that justified internal balancing in the first place.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
			inventoryActionCard("Review warehouses", "Check which facility is starving before you commit a balancing lane.", "/app/warehouses"),
			inventoryActionCard("Open inventory pressure", "Validate that the SKU needs a rebalance instead of vendor replenishment.", "/app/inventory?status=promise_risk"),
			inventoryActionCard("Transfer queue", "Review the current movement backlog and recommendation notes.", "/app/transfers"),
			inventoryActionCard("Check receiving", "Close the physical movement once the transfer actually lands.", "/app/receiving"),
		),
	)
}

func transfersTable(parseItems []transferRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, transferTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No transfer recommendations are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Transfer table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review source, destination, status, and recommendation context in one dense table before opening a transfer detail route.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d transfers", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Transfer")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Reason")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Recommended by")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func transferTableRow(parseItem transferRecord) ui.Node {
	parseLane := parseItem.SourceWarehouseID + " -> " + parseItem.DestinationWarehouse
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/transfers/" + parseItem.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.ID)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseLane)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(parseItem.Reason)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fallback(parseItem.RecommendedBy, "Atlas planning"))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(parseItem.UpdatedAt)),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.A(html.Props{Href: "/app/transfers/" + parseItem.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open transfer"))),
	)
}

func countTransferStatus(parseItems []transferRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func transferDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[transferDetailPage](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			transferDetailHero(parsePage),
			transferLineTable(parsePage.Lines),
		),
		html.Div(html.Props{Class: "grid gap-4"},
			routeRevalidationCard("Transfer route refresh", "Re-run the transfer loader after approval or cancellation work if you want to confirm the latest lane state without leaving the detail route."),
			statCard("Lane", parsePage.Transfer.SourceWarehouseID+" -> "+parsePage.Transfer.DestinationWarehouse),
			statCard("Status", parsePage.Transfer.Status),
			statCard("Recommended by", fallback(parsePage.Transfer.RecommendedBy, "Atlas planning")),
		),
	)
}

func transferDetailHero(parsePage transferDetailPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Transfer workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(parsePage.Transfer.ID)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(parsePage.Transfer.Reason)),
				),
			),
			html.A(html.Props{Href: "/app/transfers", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to transfers")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Transfer.SourceWarehouseID+" -> "+parsePage.Transfer.DestinationWarehouse)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(strings.ReplaceAll(parsePage.Transfer.Status, "_", " "))),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fallback(parsePage.Transfer.RecommendedBy, "Atlas planning"))),
		),
	)
}

func transferLineTable(parseLines []transferLineRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseRows = append(parseRows, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(parseLine.ProductSKU)),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d units", parseLine.Quantity))),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 2}}, html.Text("No transfer lines are recorded for this movement yet.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Transfer lines")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep transfer quantities in one dense table so the balancing move stays reviewable beside its route-level status and refresh controls.")),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{}, html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
					html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Product")),
					html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Quantity")),
				)),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func routeRevalidationCard(parseTitle string, parseDetail string) ui.Node {
	parseRevalidator := useAtlasRevalidator()
	parseLabel := "Refresh route data"
	if parseRevalidator.Loading() {
		parseLabel = "Refreshing route..."
	}
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-1"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(parseTitle)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseDetail)),
		),
		html.Button(html.Props{
			Type:     "button",
			Class:    "inline-flex items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-300/15 hover:text-white disabled:cursor-not-allowed disabled:border-slate-700 disabled:bg-slate-900 disabled:text-slate-500",
			Disabled: parseRevalidator.Loading(),
			OnClick:  ui.UseEvent(func() { parseRevalidator.Revalidate() }),
		}, html.Text(parseLabel)),
	)
}

func purchaseOrdersContent(parsePayload Payload) ui.Node {
	parsePage := decode[purchaseOrderList](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.14fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			purchaseOrdersSummaryBand(parsePage),
			purchaseOrdersActionCluster(),
			purchaseOrdersTable(parsePage.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryRailCard("Purchase-order handoffs", "The purchase-order workspace should hand off forward into receiving and backward into warehouse or inventory context without losing the vendor decision thread.",
				html.Div(html.Props{Class: "grid gap-3"},
					inventoryActionCard("Review receiving", "Check whether an inbound session already exists for the warehouse and vendor plan you are reviewing.", "/app/receiving"),
					inventoryActionCard("Review transfers", "If vendor replenishment is too slow, compare whether an internal balancing move is the better short-term action.", "/app/transfers"),
					inventoryActionCard("Back to warehouses", "Re-open the owning facility view when inbound ownership or backlog posture needs another look.", "/app/warehouses"),
				),
			),
		),
	)
}

func purchaseOrdersSummaryBand(parsePage purchaseOrderList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Purchase-order shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat purchase orders as a vendor-side recovery board: visible status posture, ETA clarity, and direct movement into approval, hold, and receiving follow-through.")),
		),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Orders", fmt.Sprintf("%d active", len(parsePage.Items))),
			statCard("Submitted", fmt.Sprintf("%d queued", countPurchaseOrdersByStatus(parsePage.Items, "submitted"))),
			statCard("Approved", fmt.Sprintf("%d inbound", countPurchaseOrdersByStatus(parsePage.Items, "approved"))),
			statCard("On hold", fmt.Sprintf("%d blocked", countPurchaseOrdersByStatus(parsePage.Items, "on_hold"))),
		),
	)
}

func purchaseOrdersActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Start PO work from the warehouse or inventory pressure that justified the vendor order, then follow it through approval and receiving.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
			inventoryActionCard("Start from stock risk", "Open the inventory pressure view before creating or reviewing a replenishment plan.", "/app/inventory?status=promise_risk"),
			inventoryActionCard("Check warehouse context", "Use warehouse operations to confirm which facility should own the inbound units.", "/app/warehouses"),
			inventoryActionCard("Submitted queue", "Review vendor orders waiting on approval or hold decisions.", "/app/purchase-orders"),
			inventoryActionCard("Close in receiving", "Treat receiving as the final step once the purchase order turns into a real inbound session.", "/app/receiving"),
		),
	)
}

func purchaseOrdersTable(parseItems []purchaseOrderRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, purchaseOrdersTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 7}}, html.Text("No purchase orders are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Vendor order table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scan vendor, owning warehouse, ETA posture, and status decisions in one dense table before opening the order detail route.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d orders", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Order")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Warehouse")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("ETA")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Priority")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func purchaseOrdersTableRow(parseItem purchaseOrderRecord) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/purchase-orders/" + parseItem.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.ID)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.VendorName)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseItem.WarehouseName)),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseItem.ETA)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(fallback(parseItem.PriorityNote, "No priority note"))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(parseItem.UpdatedAt)),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.A(html.Props{Href: "/app/purchase-orders/" + parseItem.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open order")),
		),
	)
}

func countPurchaseOrdersByStatus(parseItems []purchaseOrderRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func purchaseOrderDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[purchaseOrderDetailPage](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			purchaseOrderDetailHero(parsePage),
			purchaseOrderLineTable(parsePage.Lines),
		),
		purchaseOrderDetailRail(parsePayload, parsePage),
	)
}

func purchaseOrderDetailHero(parsePage purchaseOrderDetailPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Purchase-order workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(parsePage.Order.VendorName)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fallback(parsePage.Order.PriorityNote, "Vendor replenishment lane"))),
				),
			),
			html.A(html.Props{Href: "/app/purchase-orders", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to PO table")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Order.ID)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Order.WarehouseName)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Order.ETA)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(strings.ReplaceAll(parsePage.Order.Status, "_", " "))),
		),
	)
}

func purchaseOrderLineTable(parseLines []purchaseOrderLineRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseRows = append(parseRows, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(parseLine.ProductSKU)),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d units", parseLine.Quantity))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseLine.ETA)),
			html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseLine.Status)}, html.Text(strings.ReplaceAll(parseLine.Status, "_", " ")))),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 4}}, html.Text("No inbound lines are recorded for this purchase order yet.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Line-item context")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep inbound lines in one dense table so quantity, ETA, and status are reviewable before any approval or hold action on the vendor order.")),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Product")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Quantity")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("ETA")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func receivingContent(parsePayload Payload) ui.Node {
	parsePage := decode[receivingList](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			receivingSummaryBand(parsePage),
			receivingActionCluster(),
			receivingTable(parsePage.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			receivingForm(parsePayload),
			inventoryRailCard("Receiving handoffs", "Receiving closes the loop on purchase orders, transfer lanes, and warehouse recovery work.",
				html.Div(html.Props{Class: "grid gap-3"},
					inventoryActionCard("Review purchase orders", "Confirm the inbound plan before you reconcile a warehouse session.", "/app/purchase-orders"),
					inventoryActionCard("Open warehouse pressure", "Check whether the inbound changes a specific warehouse recovery path.", "/app/warehouses"),
					inventoryActionCard("Return to inventory", "Verify the quantity changes landed where the operators expect them.", "/app/inventory"),
				),
			),
		),
	)
}

func receivingSummaryBand(parsePage receivingList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Receiving shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat receiving as the closeout board for inbound work: visible discrepancy posture, owning source, and direct movement into reconciliation and inventory verification.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Sessions", fmt.Sprintf("%d active", len(parsePage.Items))),
			statCard("Open", fmt.Sprintf("%d live", countReceivingStatus(parsePage.Items, "open"))),
			statCard("Closed", fmt.Sprintf("%d closed", countReceivingStatus(parsePage.Items, "closed"))),
			statCard("Discrepancies", fmt.Sprintf("%d flagged", countReceivingWithDiscrepancy(parsePage.Items))),
		),
	)
}

func receivingActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Lead receiving work from the upstream inbound plan, then close the session through reconciliation and inventory verification.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
			inventoryActionCard("Review purchase orders", "Confirm the inbound plan before reconciling a warehouse session.", "/app/purchase-orders"),
			inventoryActionCard("Transfer follow-through", "Cross-check transfer-backed receiving work when the session closes an internal balancing move.", "/app/transfers"),
			inventoryActionCard("Receiving queue", "Review the active discrepancy and closeout backlog.", "/app/receiving"),
			inventoryActionCard("Verify inventory", "Return to inventory after closeout to confirm quantity changes landed in the expected lanes.", "/app/inventory"),
		),
	)
}

func receivingTable(parseItems []receivingRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, receivingTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No receiving sessions are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Receiving table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review source, warehouse, discrepancy posture, and closeout status in one dense table before opening the receiving detail route.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d sessions", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Session")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Warehouse")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Source")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Discrepancy")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func receivingTableRow(parseItem receivingRecord) ui.Node {
	parseSource := parseItem.SourceType + " " + parseItem.SourceID
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/receiving/" + parseItem.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.ID)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.CreatedAt)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseItem.WarehouseID)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseSource)),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(fallback(parseItem.DiscrepancySummary, "No discrepancy recorded"))),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.A(html.Props{Href: "/app/receiving/" + parseItem.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open session"))),
	)
}

func countReceivingStatus(parseItems []receivingRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func countReceivingWithDiscrepancy(parseItems []receivingRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.TrimSpace(parseItem.DiscrepancySummary) != "" {
			parseCount++
		}
	}
	return parseCount
}

func receivingDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[receivingDetailPage](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			receivingDetailHero(parsePage),
			receivingLineTable(parsePage.Lines),
		),
		receivingDetailRail(parsePayload, parsePage),
	)
}

func receivingDetailHero(parsePage receivingDetailPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Receiving workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(parsePage.Session.ID)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fallback(parsePage.Session.DiscrepancySummary, "Receiving session is ready for closeout."))),
				),
			),
			html.A(html.Props{Href: "/app/receiving", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to receiving")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Session.WarehouseID)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Session.SourceType+" "+parsePage.Session.SourceID)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(strings.ReplaceAll(parsePage.Session.Status, "_", " "))),
		),
	)
}

func receivingLineTable(parseLines []receivingLineRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseReason := fallback(parseLine.DiscrepancyReason, "matched")
		parseRows = append(parseRows, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(parseLine.ProductSKU)),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseLine.ExpectedQuantity))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseLine.ActualQuantity))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(parseReason)),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 4}}, html.Text("No receiving lines are recorded for this session yet.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Receiving lines")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep expected and actual quantities in one dense table so discrepancy review stays local to the receiving workspace.")),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{}, html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
					html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Product")),
					html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Expected")),
					html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actual")),
					html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Discrepancy")),
				)),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func atlasLazySection(parseLoader func() ui.Node, parseFallback ui.Node, parseDeps ...interface{}) ui.Node {
	return ui.CreateElement(func() ui.Node {
		handle := ui.UseLazyNode(func(context.Context) (ui.Node, error) {
			return parseLoader(), nil
		}, parseDeps...)
		parseState := handle.Get()
		return ui.AsyncBoundary(ui.AsyncBoundaryProps{
			Pending:  parseState.Loading,
			Error:    parseState.Error,
			Fallback: parseFallback,
			ErrorFallback: func(parseErr error) ui.Node {
				return parseFallback
			},
			Content: parseState.Node,
			Delay:   120 * time.Millisecond,
		})
	})
}

func internalLazyRailFallback(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"},
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
	)
}

func useAtlasFocusContainment(isActive bool, parseContainerSelector, parseInitialFocusSelector string) {
	parseManager := ui.UseFocusManager()
	parseArmed := ui.UseRef(false)
	ui.UseEffect(func() func() {
		if !isActive {
			parseArmed.Set(false)
			return nil
		}
		parseManager.RememberActive()
		parseArmed.Set(true)
		return func() {
			if parseArmed.Get() {
				parseManager.Restore()
				parseArmed.Set(false)
			}
		}
	}, isActive, parseContainerSelector, parseInitialFocusSelector)
	ui.UseFocusTrap(ui.FocusTrapOptions{
		Active:                isActive,
		ContainerSelector:     parseContainerSelector,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: parseContainerSelector,
	})
}

func atlasOverlayTarget() ui.PortalTarget {
	return ui.PortalTarget{Selector: "#" + atlasOverlayRootID}
}

func atlasDialogOverlay(isOpen bool, parseModalID, parseLabelledBy, parseDescribedBy, parseInitialFocusSelector string, parseOnDismiss func(), parseChild ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  isOpen,
		SurfaceID:             parseModalID,
		Kind:                  ui.OverlayKindDialog,
		LabelledBy:            parseLabelledBy,
		DescribedBy:           parseDescribedBy,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: "#" + parseModalID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         parseOnDismiss != nil,
		CloseOnOutsideClick:   parseOnDismiss != nil,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         "fixed inset-0 flex items-center justify-center bg-slate-950/75 p-4",
		SurfaceClass:          "w-[min(92vw,34rem)] grid gap-4 border border-slate-700 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]",
		OnDismiss:             parseOnDismiss,
		Child:                 parseChild,
	})
}

func atlasRouteSheetOverlay(parseSurfaceID, parseLabelledBy, parseDescribedBy, parseInitialFocusSelector string, parseChild ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  true,
		Target:                atlasOverlayTarget(),
		AppRootSelector:       "#" + atlasShellRootID,
		SurfaceID:             parseSurfaceID,
		Kind:                  ui.OverlayKindSheet,
		LabelledBy:            parseLabelledBy,
		DescribedBy:           parseDescribedBy,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: "#" + parseSurfaceID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         false,
		CloseOnOutsideClick:   false,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         "fixed inset-0 flex items-stretch justify-end bg-slate-950/72 p-4",
		SurfaceClass:          "grid h-full w-[min(92vw,36rem)] gap-4 overflow-y-auto rounded-sm border border-cyan-400/35 bg-[linear-gradient(180deg,rgba(11,18,32,0.98),rgba(2,6,23,1))] p-4 shadow-[0_18px_48px_rgba(6,182,212,0.08)]",
		Child:                 parseChild,
	})
}

func atlasDismissibleSheet(isOpen bool, parseSurfaceID, parseLabelledBy, parseDescribedBy, parseInitialFocusSelector string, parseOnDismiss func(), parseChild ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  isOpen,
		Target:                atlasOverlayTarget(),
		AppRootSelector:       "#" + atlasShellRootID,
		SurfaceID:             parseSurfaceID,
		Kind:                  ui.OverlayKindSheet,
		LabelledBy:            parseLabelledBy,
		DescribedBy:           parseDescribedBy,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: "#" + parseSurfaceID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         parseOnDismiss != nil,
		CloseOnOutsideClick:   parseOnDismiss != nil,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         "fixed inset-0 flex items-stretch justify-end bg-slate-950/72 p-4",
		SurfaceClass:          "grid h-full w-[min(92vw,34rem)] gap-4 overflow-y-auto rounded-[1.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]",
		OnDismiss:             parseOnDismiss,
		Child:                 parseChild,
	})
}

func atlasConfirmationDialog(isOpen bool, parseModalID, parseTitle, parseCopy, parseConfirmLabel string, parseOnDismiss func(), parseBody ...ui.Node) ui.Node {
	if !isOpen {
		return nil
	}
	parseTitleID := parseModalID + "-title"
	parseDescriptionID := parseModalID + "-description"
	parseDismissHandler := ui.UseEvent(func() {
		if parseOnDismiss != nil {
			parseOnDismiss()
		}
	})
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Confirmation")),
			html.P(html.Props{ID: parseTitleID, Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
			html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text(parseCopy)),
		),
	}
	parseChildren = append(parseChildren, parseBody...)
	parseChildren = append(parseChildren,
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-end gap-3"},
			html.Button(html.Props{Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: parseDismissHandler}, html.Text("Cancel")),
			html.Button(html.Props{ID: parseModalID + "-confirm", Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text(parseConfirmLabel)),
		),
	)
	return atlasDialogOverlay(isOpen, parseModalID, parseTitleID, parseDescriptionID, "#"+parseModalID+"-confirm", parseOnDismiss, html.Div(html.Props{Class: "grid gap-4"}, parseChildren...))
}

func atlasSidePanelErrorBoundary(parseChild ui.Node, parseTitle string, parseFallback ui.Node, resetKeys ...interface{}) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		Child:     parseChild,
		ResetKeys: resetKeys,
		ErrorFallback: func(parseErr error, reset func()) ui.Node {
			return detailRailErrorIsland(parseTitle, parseErr, reset, parseFallback)
		},
	})
}

func purchaseOrderDetailRail(parsePayload Payload, parsePage purchaseOrderDetailPage) ui.Node {
	parseFallbackNode := internalLazyRailFallback("Loading purchase-order rail", "Atlas is preparing the secondary vendor panel after the primary route body stabilizes.")
	return atlasSidePanelErrorBoundary(atlasLazySection(func() ui.Node {
		return html.Div(html.Props{Class: "grid gap-5"},
			routeRevalidationCard("Purchase-order route refresh", "Re-run the current purchase-order loader after an approval or hold action so the latest vendor posture and inbound timing stay authoritative on this screen."),
			purchaseOrderDetailStatsIsland(parsePayload, parsePage),
			purchaseOrderStatusForm(parsePage.Order.ID, parsePage.Order.Status, parsePayload),
		)
	}, parseFallbackNode, parsePage.Order.ID, parsePage.Order.Status, parsePage.Order.ETA), "Purchase-order side panel", parseFallbackNode, parsePage.Order.ID, parsePage.Order.Status, parsePage.Order.ETA)
}

func receivingDetailRail(parsePayload Payload, parsePage receivingDetailPage) ui.Node {
	parseFallbackNode := internalLazyRailFallback("Loading receiving rail", "Atlas is preparing the secondary receiving panel after the primary route body stabilizes.")
	return atlasSidePanelErrorBoundary(atlasLazySection(func() ui.Node {
		return html.Div(html.Props{Class: "grid gap-5"},
			routeRevalidationCard("Receiving route refresh", "Re-run the receiving loader after reconcile or classification work so discrepancy status and closeout readiness refresh in place."),
			receivingDetailStatsIsland(parsePayload, parsePage),
			receivingFormForID(parsePage.Session.ID, parsePayload),
		)
	}, parseFallbackNode, parsePage.Session.ID, parsePage.Session.Status, parsePage.Session.DiscrepancySummary), "Receiving side panel", parseFallbackNode, parsePage.Session.ID, parsePage.Session.Status, parsePage.Session.DiscrepancySummary)
}

func purchaseOrderDetailStatsIsland(parsePayload Payload, parsePage purchaseOrderDetailPage) ui.Node {
	parseRequest, parseOk := StartupRequest(parsePayload, "page")
	parseRequestURL := strings.TrimSpace(parseRequest.URL)
	parseFallbackNode := purchaseOrderDetailStatsContent(parsePage, false, false, "", ui.Handler{})
	if !parseOk || parseRequestURL == "" {
		return parseFallbackNode
	}
	return ui.CreateElement(func() ui.Node {
		parsePanelState := useAtlasState(parsePage)
		parseRefreshingState := useAtlasState(false)
		parseRefreshErrorState := useAtlasState("")
		parseResource := useAtlasStartupPageResource(parsePayload)
		parseResourceState := parseResource.Get()
		parseCachedPage := purchaseOrderDetailPage{}
		if parseResourceState.Ready {
			parseCachedPage = decode[purchaseOrderDetailPage](parseResourceState.Value)
		}
		useAtlasEffect(func() func() {
			if parseResourceState.Ready && strings.TrimSpace(parseCachedPage.Order.ID) != "" {
				parsePanelState.Set(parseCachedPage)
			}
			return nil
		}, parseResourceState.Ready, parseCachedPage.Order.ID, parseCachedPage.Order.Status, parseCachedPage.Order.ETA, parseCachedPage.Order.WarehouseName)
		parseRefreshPanel := ui.UseEvent(func() {
			if parseRefreshingState.Get() {
				return
			}
			parseRefreshingState.Set(true)
			parseRefreshErrorState.Set("")
			go func() {
				parseResult := <-atlasFetch(parseRequestURL, atlasFetchOptions{
					Method: "GET",
					Headers: map[string]interface{}{
						"Accept": "application/json",
					},
				})
				if strings.TrimSpace(parseResult.Error) != "" {
					parseRefreshErrorState.Set(parseResult.Error)
					parseRefreshingState.Set(false)
					return
				}
				parseDecoded := decodeJSONText[purchaseOrderDetailPage](parseResult.Data)
				if strings.TrimSpace(parseDecoded.Order.ID) == "" {
					parseRefreshErrorState.Set("Atlas returned an empty purchase-order panel payload.")
					parseRefreshingState.Set(false)
					return
				}
				parsePanelState.Set(parseDecoded)
				parseResource.Set(parseDecoded)
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Purchase-order panel refreshed",
					Detail: "The vendor-side status panel picked up the latest order snapshot without rerendering the whole route.",
					Tone:   "success",
				})
				parseRefreshingState.Set(false)
			}()
		})
		parseCurrent := parsePanelState.Get()
		return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
			Pending:  !parseResourceState.Ready && parseResourceState.Error == nil,
			Error:    parseResourceState.Error,
			Fallback: parseFallbackNode,
			ErrorFallback: func(parseErr error) ui.Node {
				return detailRailErrorIsland("Purchase-order side panel", parseErr, parseResource.Reload, parseFallbackNode)
			},
			Content: purchaseOrderDetailStatsContent(parseCurrent, parseResourceState.Loading && parseResourceState.Ready, parseRefreshingState.Get(), parseRefreshErrorState.Get(), parseRefreshPanel),
		})
	})
}

func purchaseOrderDetailStatsContent(parsePage purchaseOrderDetailPage, isRefreshing bool, isFetchRefreshing bool, parseRefreshError string, parseRefetch ui.Handler) ui.Node {
	parseChildren := []ui.Node{}
	parseLabel := "Refresh panel snapshot"
	if isFetchRefreshing {
		parseLabel = "Refreshing panel..."
	}
	parseChildren = append(parseChildren, html.Button(html.Props{
		Type:     "button",
		Class:    "inline-flex items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-300/15 hover:text-white disabled:cursor-not-allowed disabled:border-slate-700 disabled:bg-slate-900 disabled:text-slate-500",
		Disabled: isFetchRefreshing,
		OnClick:  parseRefetch,
	}, html.Text(parseLabel)))
	if isRefreshing || isFetchRefreshing {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.35rem] border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm leading-6 text-cyan-100"}, html.Text("Refreshing the purchase-order side panel while the current snapshot stays visible.")))
	}
	if strings.TrimSpace(parseRefreshError) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 text-sm leading-6 text-rose-100"}, html.Text(parseRefreshError)))
	}
	parseChildren = append(parseChildren,
		statCard("Warehouse", parsePage.Order.WarehouseName),
		statCard("ETA", parsePage.Order.ETA),
	)
	return html.Div(html.Props{Class: "grid gap-5"}, parseChildren...)
}

func receivingDetailStatsIsland(parsePayload Payload, parsePage receivingDetailPage) ui.Node {
	parseRequest, parseOk := StartupRequest(parsePayload, "page")
	parseRequestURL := strings.TrimSpace(parseRequest.URL)
	parseFallbackNode := receivingDetailStatsContent(parsePage, false, false, "", ui.Handler{})
	if !parseOk || parseRequestURL == "" {
		return parseFallbackNode
	}
	return ui.CreateElement(func() ui.Node {
		parsePanelState := useAtlasState(parsePage)
		parseRefreshingState := useAtlasState(false)
		parseRefreshErrorState := useAtlasState("")
		parseResource := useAtlasStartupPageResource(parsePayload)
		parseResourceState := parseResource.Get()
		parseCachedPage := receivingDetailPage{}
		if parseResourceState.Ready {
			parseCachedPage = decode[receivingDetailPage](parseResourceState.Value)
		}
		useAtlasEffect(func() func() {
			if parseResourceState.Ready && strings.TrimSpace(parseCachedPage.Session.ID) != "" {
				parsePanelState.Set(parseCachedPage)
			}
			return nil
		}, parseResourceState.Ready, parseCachedPage.Session.ID, parseCachedPage.Session.Status, parseCachedPage.Session.WarehouseID)
		parseRefreshPanel := ui.UseEvent(func() {
			if parseRefreshingState.Get() {
				return
			}
			parseRefreshingState.Set(true)
			parseRefreshErrorState.Set("")
			go func() {
				parseResult := <-atlasFetch(parseRequestURL, atlasFetchOptions{
					Method: "GET",
					Headers: map[string]interface{}{
						"Accept": "application/json",
					},
				})
				if strings.TrimSpace(parseResult.Error) != "" {
					parseRefreshErrorState.Set(parseResult.Error)
					parseRefreshingState.Set(false)
					return
				}
				parseDecoded := decodeJSONText[receivingDetailPage](parseResult.Data)
				if strings.TrimSpace(parseDecoded.Session.ID) == "" {
					parseRefreshErrorState.Set("Atlas returned an empty receiving panel payload.")
					parseRefreshingState.Set(false)
					return
				}
				parsePanelState.Set(parseDecoded)
				parseResource.Set(parseDecoded)
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Receiving panel refreshed",
					Detail: "The receiving side panel picked up the latest session snapshot without rerendering the whole route.",
					Tone:   "success",
				})
				parseRefreshingState.Set(false)
			}()
		})
		parseCurrent := parsePanelState.Get()
		return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
			Pending:  !parseResourceState.Ready && parseResourceState.Error == nil,
			Error:    parseResourceState.Error,
			Fallback: parseFallbackNode,
			ErrorFallback: func(parseErr error) ui.Node {
				return detailRailErrorIsland("Receiving side panel", parseErr, parseResource.Reload, parseFallbackNode)
			},
			Content: receivingDetailStatsContent(parseCurrent, parseResourceState.Loading && parseResourceState.Ready, parseRefreshingState.Get(), parseRefreshErrorState.Get(), parseRefreshPanel),
		})
	})
}

func receivingDetailStatsContent(parsePage receivingDetailPage, isRefreshing bool, isFetchRefreshing bool, parseRefreshError string, parseRefetch ui.Handler) ui.Node {
	parseChildren := []ui.Node{}
	parseLabel := "Refresh panel snapshot"
	if isFetchRefreshing {
		parseLabel = "Refreshing panel..."
	}
	parseChildren = append(parseChildren, html.Button(html.Props{
		Type:     "button",
		Class:    "inline-flex items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-300/15 hover:text-white disabled:cursor-not-allowed disabled:border-slate-700 disabled:bg-slate-900 disabled:text-slate-500",
		Disabled: isFetchRefreshing,
		OnClick:  parseRefetch,
	}, html.Text(parseLabel)))
	if isRefreshing || isFetchRefreshing {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.35rem] border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm leading-6 text-cyan-100"}, html.Text("Refreshing the receiving side panel while the current snapshot stays visible.")))
	}
	if strings.TrimSpace(parseRefreshError) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 text-sm leading-6 text-rose-100"}, html.Text(parseRefreshError)))
	}
	parseChildren = append(parseChildren,
		statCard("Warehouse", parsePage.Session.WarehouseID),
		statCard("Status", parsePage.Session.Status),
	)
	return html.Div(html.Props{Class: "grid gap-5"}, parseChildren...)
}

func detailRailErrorIsland(parseTitle string, parseErr error, parseRetry func(), parseFallbackNode ui.Node) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-rose-200"}, html.Text(parseTitle)),
			html.P(html.Props{Class: "text-sm leading-6 text-rose-100"}, html.Text(parseErr.Error())),
			html.Button(html.Props{
				Type:    "button",
				Class:   "inline-flex items-center justify-center rounded-full border border-rose-200/40 bg-rose-200/10 px-4 py-3 text-sm font-semibold text-white transition hover:bg-rose-200/20",
				OnClick: ui.UseEvent(func() { parseRetry() }),
			}, html.Text("Retry panel")),
		),
		parseFallbackNode,
	)
}

func commentsContent(parsePayload Payload) ui.Node {
	parsePage := decode[commentList](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			commentsSummaryBand(parsePage),
			commentsActionCluster(),
			commentsTable(parsePage.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryRailCard("Buyer inbox handoffs", "Move from public feedback into the right internal route without losing the original question context.",
				html.Div(html.Props{Class: "grid gap-3"},
					inventoryActionCard("Update marketing copy", "Route unclear product questions into the merch workspace when the issue is messaging, not stock.", "/app/products"),
					inventoryActionCard("Check inventory promise", "Open the inventory workspace when the customer is really asking about supply or timing.", "/app/inventory"),
					inventoryActionCard("Open warehouse ops", "Use warehouse-native routes when the answer depends on a specific hub or recovery lane.", "/app/warehouses"),
				),
			),
			moderationForm(parsePage.Items, parsePayload),
			bulkModerationForm(parsePage.Items, parsePayload),
		),
	)
}

func settingsContent(parsePayload Payload) ui.Node {
	parsePage := decode[settingsPage](pageData(parsePayload))
	parsePresentation := currentShellPresentationState(parsePayload)
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			settingsSummaryBand(parsePage, parsePresentation),
			settingsActionCluster(),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			preferenceForm(parsePayload),
			savedViewBrowserCard(parsePayload),
			savedViewTransferCard(parsePayload),
			operatorWorkspaceSnapshotCard(parsePayload),
		),
	)
}

func commentsSummaryBand(parsePage commentList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Buyer inbox shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat buyer comments as an operator inbox: visible moderation posture, product context, and direct handoff into merchandising or supply routes when the question reveals a broader issue.")),
		),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Comments", fmt.Sprintf("%d open", len(parsePage.Items))),
			statCard("Pending", fmt.Sprintf("%d queued", countCommentStatus(parsePage.Items, "pending"))),
			statCard("Approved", fmt.Sprintf("%d live", countCommentStatus(parsePage.Items, "approved"))),
			statCard("Flagged", fmt.Sprintf("%d escalated", countCommentStatus(parsePage.Items, "flagged"))),
		),
	)
}

func commentsActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Use buyer questions to route the operator into the exact internal follow-up: merchandising, inventory promise, or warehouse operations.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
			inventoryActionCard("Update marketing copy", "Route unclear product questions into the merch workspace when the issue is messaging, not stock.", "/app/products"),
			inventoryActionCard("Check inventory promise", "Open the inventory workspace when the customer is really asking about supply or timing.", "/app/inventory"),
			inventoryActionCard("Open warehouse ops", "Use warehouse-native routes when the answer depends on a specific hub or recovery lane.", "/app/warehouses"),
		),
	)
}

func commentsTable(parseItems []commentRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, commentTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 5}}, html.Text("No buyer questions are waiting in the Atlas inbox.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Buyer question table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review subject, product context, author, and moderation posture in one dense table before opening a workflow on the right rail.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d comments", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Comment")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Product")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Author")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func commentTableRow(parseItem commentRecord) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.Subject)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Body)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseItem.ProductSKU)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(parseItem.AuthorName+" | "+strings.ReplaceAll(parseItem.AuthorType, "_", " "))),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(parseItem.UpdatedAt)),
	)
}

func countCommentStatus(parseItems []commentRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func settingsSummaryBand(parsePage settingsPage, parsePresentation shellPresentationState) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Settings shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat settings as the operator control room for shell presentation, saved-view exchange, and workspace snapshot handoff rather than a narrow preference form.")),
		),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-5"},
			statCard("Theme", fallback(parsePresentation.Theme, "dark")),
			statCard("Locale", fallback(parsePresentation.Locale, "en")),
			statCard("Density", fallback(parsePresentation.Density, "compact")),
			statCard("Warehouse", fallback(parsePresentation.DefaultWarehouse, "new-jersey-hub")),
			statCard("Diagnostics", map[bool]string{true: "on", false: "off"}[parsePresentation.Diagnostics]),
		),
	)
}

func settingsActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Use settings to control shell presentation, move saved views between sessions, and export a workspace snapshot for reviewer or operator handoff.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
			inventoryActionCard("Open dashboard", "Return to the internal overview after changing presentation or workspace defaults.", "/app/dashboard"),
			inventoryActionCard("Inspect inventory views", "Cross-check saved-view presets against the live inventory workspace.", "/app/inventory"),
			inventoryActionCard("Review comments", "Jump into the buyer inbox if the next task is moderation rather than shell configuration.", "/app/comments"),
		),
	)
}

func savedViewBrowserCard(parsePayload Payload) ui.Node {
	if len(parsePayload.SavedViews) == 0 {
		return listCard("Saved views", html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No records yet.")))
	}
	parseItems := make([]ui.CompositeItem, 0, len(parsePayload.SavedViews))
	for parseIndex, parseSaved := range parsePayload.SavedViews {
		parseItems = append(parseItems, ui.CompositeItem{
			ID:   fmt.Sprintf("atlas-saved-view-%d", parseIndex),
			Text: strings.TrimSpace(parseSaved.Name + " " + parseSaved.Scope + " " + parseSaved.SortKey + " " + parseSaved.SortDirection),
		})
	}
	return ui.CreateElement(func() ui.Node {
		parseNav := ui.UseCompositeNavigation(parseItems, ui.CompositeNavigationOptions{Orientation: "vertical", Loop: true})
		parseListboxKeyDown := ui.UseEvent(func(parseEvent ui.KeyboardEvent) {
			parseNav.OnKeyDown(parseEvent)
		})
		parseActiveIndex := parseNav.ActiveIndex()
		if parseActiveIndex < 0 || parseActiveIndex >= len(parsePayload.SavedViews) {
			parseActiveIndex = 0
		}
		parseActive := parsePayload.SavedViews[parseActiveIndex]
		parseOptions := make([]ui.Node, 0, len(parsePayload.SavedViews))
		for parseIndex2, parseSaved2 := range parsePayload.SavedViews {
			parseCurrentIndex := parseIndex2
			parseCurrentItem := parseItems[parseIndex2]
			parseClassName := "grid gap-2 rounded-[1.2rem] border px-4 py-3 text-left text-sm transition"
			if parseNav.IsActive(parseIndex2) {
				parseClassName += " border-cyan-300/35 bg-cyan-300/10 text-cyan-50"
			} else {
				parseClassName += " border-white/10 bg-slate-950/40 text-slate-200"
			}
			parseOptions = append(parseOptions, html.Div(html.Props{
				ID:    parseCurrentItem.ID,
				Role:  "option",
				Class: parseClassName,
				OnClick: ui.UseEvent(func() {
					parseNav.SetActive(parseCurrentIndex)
				}),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[parseNav.IsActive(parseIndex2)],
				},
			},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseSaved2.Name)),
				html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseSaved2.Scope+" | "+parseSaved2.SortKey+"/"+parseSaved2.SortDirection)),
			))
		}
		filterEntries := make([]string, 0, len(parseActive.Filters))
		for parseKey, parseValue := range parseActive.Filters {
			filterEntries = append(filterEntries, parseKey+": "+parseValue)
		}
		sort.Strings(filterEntries)
		filterNodes := make([]ui.Node, 0, len(filterEntries))
		for _, parseEntry := range filterEntries {
			filterNodes = append(filterNodes, html.P(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs uppercase tracking-[0.18em] text-slate-300"}, html.Text(parseEntry)))
		}
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Saved views")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Focus the list and use ArrowUp, ArrowDown, Home, End, or typeahead to inspect Atlas workspace presets without leaving the settings route.")),
			html.Div(html.Props{
				Role:      "listbox",
				Class:     "grid gap-3 rounded-[1.35rem] border border-white/10 bg-slate-950/55 p-4",
				OnKeyDown: parseListboxKeyDown,
				Aria: map[string]string{
					"activedescendant": parseNav.ActiveDescendant(),
				},
				Raw: map[string]interface{}{
					"tabIndex": 0,
				},
			}, parseOptions...),
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-white/5 p-4"},
				html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
					statCard("Scope", parseActive.Scope),
					statCard("Sort key", parseActive.SortKey),
					statCard("Direction", parseActive.SortDirection),
				),
			),
		}
		if len(filterNodes) > 0 {
			parseChildren = append(parseChildren, html.Div(html.Props{Class: "flex flex-wrap gap-2"}, filterNodes...))
		}
		return html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, parseChildren...)
	})
}

func mockSignInContent(parsePayload Payload) ui.Node {
	parsePage := decode[mockSignInPage](pageData(parsePayload))
	if len(parsePage.Roles) == 0 {
		parsePage.Roles = []mockSignInRole{{Value: "inventory_manager", Label: "Inventory Manager", Description: "Default Atlas internal operator role."}}
	}
	parseRoleCards := make([]ui.Node, 0, len(parsePage.Roles))
	for _, parseRole := range parsePage.Roles {
		parseRoleCards = append(parseRoleCards,
			html.Form(html.Props{Action: "/auth/mock-sign-in", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
				html.Input(html.Props{Type: "hidden", Name: "role", Value: parseRole.Value}),
				html.Input(html.Props{Type: "hidden", Name: "next", Value: fallback(parsePage.Next, "/app/dashboard")}),
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseRole.Label)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseRole.Description)),
				html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text("Start session")),
			),
		)
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.85fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard("Mock internal access", fallback(parsePage.Message, "Start a mock Atlas internal session to access the operator console.")),
			listCard("Recovery path",
				infoRow("Next route", fallback(parsePage.Next, "/app/dashboard")),
				infoRow("Session model", "Cookie-backed mock operator role"),
			),
		),
		html.Div(html.Props{Class: "grid gap-4"}, parseRoleCards...),
	)
}

func recoveryContent(parsePayload Payload) ui.Node {
	parsePage := decode[recoveryPage](pageData(parsePayload))
	parseChildren := []ui.Node{
		featureCard(fallback(parsePage.Title, "Route recovery"), fallback(parsePage.Message, "Atlas could not resolve the requested route.")),
		html.A(html.Props{Href: fallback(parsePage.RecoveryHref, "/"), Class: "inline-flex w-fit rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(parsePage.RecoveryLabel, "Back to Atlas"))),
	}
	if strings.TrimSpace(parsePage.Detail) != "" {
		parseChildren = append(parseChildren, listCard("Recovery detail", html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(parsePage.Detail))))
	}
	return html.Section(html.Props{Class: "grid gap-6"}, parseChildren...)
}

func fallbackContent(parsePayload Payload) ui.Node {
	parseEntries := sortedMapStrings(parsePayload.Data)
	parseNodes := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-slate-300"}, html.Text(parseEntry)))
	}
	return listCard("Route data", parseNodes...)
}

func routeSummaryStrip(parseSummary pageSummary) ui.Node {
	if len(parseSummary.Items) == 0 {
		return html.Div(html.Props{})
	}
	parseNodes := make([]ui.Node, 0, len(parseSummary.Items))
	for _, parseItem := range parseSummary.Items {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "grid gap-2 " + internalInsetSurfaceClass() + " p-4"},
			html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-cyan-300"}, html.Text(parseItem.Label)),
			html.P(html.Props{Class: "text-xl font-semibold text-white"}, html.Text(parseItem.Value)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Detail)),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4"},
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text(fallback(parseSummary.Headline, "Route summary"))),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2 xl:grid-cols-4"}, parseNodes...),
	)
}

func featureCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: internalSurfaceCardClass() + " p-5"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
	)
}

func publicFeatureCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-stone-300"}, html.Text(parseCopy)),
	)
}

func publicMetricCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.06),rgba(255,255,255,0.03))] p-5 shadow-[0_14px_32px_rgba(0,0,0,0.18)]"},
		html.P(html.Props{Class: "text-xl font-bold tracking-[-0.03em] text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "mt-2 text-sm leading-6 text-stone-300"}, html.Text(parseCopy)),
	)
}

func publicSignalPill(parseLabel string) ui.Node {
	return html.Div(html.Props{Class: "rounded-full border border-white/12 bg-white/8 px-4 py-2 text-sm font-semibold text-stone-200"}, html.Text(parseLabel))
}

func publicStatusClass(parseStatus string) string {
	parseBase := "rounded-full border px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.22em]"
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "in_stock", "healthy", "approved", "available":
		return parseBase + " border-emerald-400/30 bg-emerald-400/12 text-emerald-200"
	case "low_stock", "pending", "submitted", "in_review":
		return parseBase + " border-amber-400/30 bg-amber-400/12 text-amber-200"
	default:
		return parseBase + " border-white/12 bg-white/8 text-stone-300"
	}
}

func publicStatusLabel(parseStatus string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(parseStatus)), "_", " ")
}

func productPrimaryActionForm(parseProduct productCard, parsePayload Payload) ui.Node {
	switch strings.TrimSpace(strings.ToLower(parseProduct.Status)) {
	case "in_stock", "healthy", "approved", "available":
		return publicFormCard("Start a project quote", "Capture pricing, volume, and install timing while this item is ready to support an active project.", "/api/public/products/"+parseProduct.Slug+"/quote-requests", "Request pricing", parsePayload.CSRF, []ui.Node{
			publicInput("requester_name", "Name"),
			publicInput("company_name", "Company"),
			publicInput("email", "Email"),
			publicInput("quantity", "Quantity"),
			publicTextarea("note", "Project note"),
		})
	case "low_stock", "pending", "submitted", "in_review":
		return publicFormCard("Reserve upcoming availability", "Hold intent against the next recovery window so the team can plan around constrained stock.", "/api/public/products/"+parseProduct.Slug+"/restock-requests", "Reserve availability", parsePayload.CSRF, []ui.Node{
			publicInput("email", "Email"),
			publicInput("preferred_warehouse_id", "Preferred delivery region"),
		})
	default:
		return publicFormCard("Notify me when available", "Stay attached to this SKU without forcing a buyer to think in replenishment or internal operations terms.", "/api/public/products/"+parseProduct.Slug+"/restock-requests", "Notify me", parsePayload.CSRF, []ui.Node{
			publicInput("email", "Email"),
			publicInput("preferred_warehouse_id", "Preferred delivery region"),
		})
	}
}

func productSecondaryActionCard(parseProduct productCard) ui.Node {
	parseStatus := strings.TrimSpace(strings.ToLower(parseProduct.Status))
	if parseStatus == "in_stock" || parseStatus == "healthy" || parseStatus == "approved" || parseStatus == "available" {
		return html.A(html.Props{Href: "/warehouses", Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] transition hover:border-amber-300/35 hover:bg-white/10"},
			html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("Check delivery for your region")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Compare regional delivery options before you lock in your project timeline and installation plan.")),
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-amber-700"}, html.Text("See regional delivery options")),
		)
	}
	return html.A(html.Props{Href: "/shop?category=" + url.QueryEscape(parseProduct.Category), Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] transition hover:border-amber-300/35 hover:bg-white/10"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("See similar options")),
		html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("If this SKU cannot support the buyer timeline, keep momentum by browsing adjacent in-category systems.")),
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-amber-700"}, html.Text("Browse alternatives")),
	)
}

func availabilityPrimaryActionForm(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	if parseAvailability.Available > 0 {
		return publicFormCard("Request pricing for this region", "Capture pricing and project timing while this warehouse can still support the current demand window.", "/api/public/products/"+parseAvailability.Product.Slug+"/quote-requests", "Start quote", parsePayload.CSRF, []ui.Node{
			publicInput("requester_name", "Name"),
			publicInput("company_name", "Company"),
			publicInput("email", "Email"),
			publicInput("quantity", "Quantity"),
			publicTextarea("note", "Project note"),
		})
	}
	parseTitle := "Notify me when available"
	parseCopy := "Stay tied to this regional lane without making the buyer restate the product and warehouse context later."
	parseButton := "Notify me"
	if parseAvailability.Inbound > 0 {
		parseTitle = "Reserve upcoming availability"
		parseCopy = "Hold this buyer against the next inbound recovery window for the current warehouse lane."
		parseButton = "Reserve availability"
	}
	return publicFormCard(parseTitle, parseCopy, "/api/public/products/"+parseAvailability.Product.Slug+"/restock-requests", parseButton, parsePayload.CSRF, []ui.Node{
		publicInput("email", "Email"),
		publicInputWithValue("preferred_warehouse_id", "Preferred delivery region", parseAvailability.Warehouse.ID),
	})
}

func availabilityQuestionActionForm(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	return publicFormCard("Ask about delivery or fit", "Keep regional delivery, installation, and substitution questions attached to the actual availability context.", "/api/public/products/"+parseAvailability.Product.Slug+"/comments", "Send question", parsePayload.CSRF, []ui.Node{
		publicInput("author_name", "Name"),
		publicInput("subject", "Subject"),
		publicTextarea("body", "Question"),
	})
}

func statCard(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: internalSurfaceCardClass() + " p-5"},
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text(parseValue)),
	)
}

func listCard(parseTitle string, parseChildren ...ui.Node) ui.Node {
	if len(parseChildren) == 0 {
		parseChildren = []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No records yet."))}
	}
	parseContent := []ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseTitle))}
	parseContent = append(parseContent, parseChildren...)
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"}, parseContent...)
}

func publicFormCard(parseTitle, parseCopy, parseAction, parseSubmitLabel string, parseCsrfToken string, parseFields []ui.Node) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseTitle)),
	}
	if strings.TrimSpace(parseCopy) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseCopy)))
	}
	parseChildren = append(parseChildren, prependCSRFToken(parseCsrfToken)...)
	parseChildren = append(parseChildren, parseFields...)
	parseChildren = append(parseChildren, html.Button(html.Props{Type: "submit", Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(fallback(strings.TrimSpace(parseSubmitLabel), "Submit"))))
	return html.Form(html.Props{Action: parseAction, Method: "post", Class: "grid gap-4 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, parseChildren...)
}

type preferenceFormState struct {
	Theme              string
	Locale             string
	Density            string
	DefaultWarehouseID string
}

type moderationActionFormState struct {
	Status string
	Reason string
}

type bulkModerationActionFormState struct {
	IDs    string
	Status string
	Reason string
}

type savedViewImportFormState struct {
	ViewsJSON string
}

type workspaceSnapshotFormState struct {
	WorkspaceSnapshotJSON string
}

type transferFormState struct {
	SourceWarehouseID      string
	DestinationWarehouseID string
	Reason                 string
	RecommendedBy          string
}

type receivingFormState struct {
	Status             string
	DiscrepancySummary string
}

type receivingResolutionWorkflowState struct {
	Status             string
	DiscrepancySummary string
	Stage              string
	LastEditedField    string
}

type receivingResolutionWorkflowAction struct {
	Field string
	Value string
}

func normalizeReceivingResolutionWorkflowState(parseState receivingResolutionWorkflowState) receivingResolutionWorkflowState {
	parseStatus := strings.TrimSpace(strings.ToLower(parseState.Status))
	parseSummary := strings.TrimSpace(parseState.DiscrepancySummary)
	switch {
	case parseStatus == "closed" && parseSummary == "":
		parseState.Stage = "closeout-needs-note"
	case parseStatus == "closed":
		parseState.Stage = "closeout-ready"
	case parseStatus == "review":
		parseState.Stage = "discrepancy-review"
	default:
		parseState.Stage = "classification-open"
	}
	return parseState
}

func reduceReceivingResolutionWorkflowState(parseState receivingResolutionWorkflowState, parseAction receivingResolutionWorkflowAction) receivingResolutionWorkflowState {
	switch parseAction.Field {
	case "status":
		parseState.Status = parseAction.Value
	case "discrepancy_summary":
		parseState.DiscrepancySummary = parseAction.Value
	}
	parseState.LastEditedField = parseAction.Field
	return normalizeReceivingResolutionWorkflowState(parseState)
}

func preferenceForm(parsePayload Payload) ui.Node {
	parsePresentation := currentShellPresentationState(parsePayload)
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(preferenceFormState{
			Theme:              fallback(parsePresentation.Theme, "dark"),
			Locale:             fallback(parsePresentation.Locale, "en"),
			Density:            fallback(parsePresentation.Density, "compact"),
			DefaultWarehouseID: fallback(parsePresentation.DefaultWarehouse, "new-jersey-hub"),
		})
		parseTransition := useAtlasTransition()
		parseValue := parseForm.Get()
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Operator preferences")),
			boundInputWithValue("theme", "Theme", parseValue.Theme, "Theme", parseForm),
			boundInputWithValue("locale", "Locale", parseValue.Locale, "Locale", parseForm),
			boundTransitionSelectWithValue("density", "Density", parseValue.Density, "Density", []optionItem{{"compact", "Compact"}, {"comfortable", "Comfortable"}}, parseForm, parseTransition),
			preferenceDensityPreviewCard(parseValue.Density, parseTransition.Pending()),
			boundInputWithValue("default_warehouse_id", "Default warehouse", parseValue.DefaultWarehouseID, "DefaultWarehouseID", parseForm),
			submitButton("Save preferences"),
		}
		return html.Form(html.Props{Action: "/api/app/preferences", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func moderationForm(parseItems []commentRecord, parsePayload Payload) ui.Node {
	parseId := ""
	if len(parseItems) > 0 {
		parseId = parseItems[0].ID
	}
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(moderationActionFormState{Status: "approved", Reason: "Reviewed by Atlas operations."})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Review first queued buyer question")),
			boundInputWithValue("status", "Status", parseValue.Status, "Status", parseForm),
			boundTextareaWithValue("reason", "Reason", parseValue.Reason, "Reason", parseForm),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: parseOpenConfirm}, html.Text("Review decision")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-comment-review-confirm", "Confirm buyer review", "Atlas keeps keyboard focus inside the moderation confirmation step until you either cancel or apply the review.", "Apply review", parseCloseConfirm,
				statCard("Status", parseValue.Status),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseValue.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/comments/" + parseId + "/moderate", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func bulkModerationForm(parseItems []commentRecord, parsePayload Payload) ui.Node {
	parseIds := make([]string, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseIds = append(parseIds, parseItem.ID)
	}
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(bulkModerationActionFormState{
			IDs:    strings.Join(parseIds, ","),
			Status: "approved",
			Reason: "Bulk-reviewed from the visible Atlas buyer inbox queue.",
		})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Bulk review visible queue")),
			html.Input(html.Props{Type: "hidden", Name: "ids", Value: parseValue.IDs}),
			boundInputWithValue("status", "Status", parseValue.Status, "Status", parseForm),
			boundTextareaWithValue("reason", "Reason", parseValue.Reason, "Reason", parseForm),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: parseOpenConfirm}, html.Text("Review bulk action")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-bulk-review-confirm", "Confirm bulk moderation", "The buyer-inbox bulk action now traps focus inside its confirmation step instead of leaving focus scattered behind the modal.", "Apply bulk review", parseCloseConfirm,
				statCard("Items", fmt.Sprintf("%d", len(parseItems))),
				statCard("Status", parseValue.Status),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseValue.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/comments/bulk-moderate", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func savedViewForm(parsePayload Payload) ui.Node {
	parseChildren := []ui.Node{
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
	return html.Form(html.Props{Action: "/api/app/saved-views", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
}

func savedViewTransferCard(parsePayload Payload) ui.Node {
	parsePresentation := currentShellPresentationState(parsePayload)
	type savedViewTransferItem struct {
		Name          string `json:"name"`
		Scope         string `json:"scope"`
		SortKey       string `json:"sortKey"`
		SortDirection string `json:"sortDirection"`
		Density       string `json:"density"`
		WarehouseID   string `json:"warehouseId"`
		FiltersJSON   string `json:"filtersJSON"`
	}
	parseItems := make([]savedViewTransferItem, 0, len(parsePayload.SavedViews))
	for _, parseSaved := range parsePayload.SavedViews {
		parseFiltersJSON, parseErr := json.Marshal(parseSaved.Filters)
		if parseErr != nil {
			parseFiltersJSON = []byte(`{}`)
		}
		parseItems = append(parseItems, savedViewTransferItem{
			Name:          parseSaved.Name,
			Scope:         parseSaved.Scope,
			SortKey:       parseSaved.SortKey,
			SortDirection: parseSaved.SortDirection,
			Density:       fallback(parsePresentation.Density, "compact"),
			WarehouseID:   fallback(parseSaved.Filters["warehouse"], parsePresentation.DefaultWarehouse),
			FiltersJSON:   string(parseFiltersJSON),
		})
	}
	parseExportPayload := map[string]any{"items": parseItems}
	parseEncoded, parseErr2 := json.MarshalIndent(parseExportPayload, "", "  ")
	if parseErr2 != nil {
		parseEncoded = []byte(`{"items":[]}`)
	}
	parseDefaultImportPayload := string(parseEncoded)
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(savedViewImportFormState{ViewsJSON: parseDefaultImportPayload})
		parseValidator := useAtlasWorkerTask[savedViewImportValidationRequest, savedViewImportValidationProgress, savedViewImportValidationResult](interop.WorkerOptions{
			URL:   "/assets/script/atlas-saved-view-import-worker.js",
			Ready: true,
			Name:  "atlas-saved-view-import",
		}, "validate-saved-view-import")
		useAtlasEffect(func() func() {
			parseTrimmed := strings.TrimSpace(parseForm.Get().ViewsJSON)
			if parseTrimmed == "" {
				parseValidator.Cancel()
				return nil
			}
			parseValidator.Start(savedViewImportValidationRequest{Text: parseForm.Get().ViewsJSON})
			return nil
		}, parseForm.Get().ViewsJSON)
		parseValidationState := parseValidator.Get()
		parseValue := parseForm.Get()
		parseChildren := []ui.Node{
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Export saved views")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Download or copy the current saved-view payload so Atlas presets can move between runs without rebuilding them by hand.")),
				html.A(html.Props{Href: "/api/app/saved-views/export", Class: "inline-flex w-fit rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text("Open export payload")),
				html.Pre(html.Props{Class: "overflow-x-auto rounded-[1.2rem] border border-white/10 bg-slate-950/70 p-4 text-xs leading-6 text-slate-300"}, html.Text(parseDefaultImportPayload)),
			),
		}
		parseImportChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Import saved views")),
			boundTextareaWithValue("views_json", "Saved-view payload", parseValue.ViewsJSON, "ViewsJSON", parseForm),
			savedViewImportValidationCard(parseValidationState),
			submitButton("Import saved views"),
		}
		parseChildren = append(parseChildren, html.Form(html.Props{Action: "/api/app/saved-views/import", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseImportChildren...)...))
		return html.Div(html.Props{Class: "grid gap-5"}, parseChildren...)
	})
}

type savedViewImportValidationRequest struct {
	Text string `json:"text"`
}

type savedViewImportValidationProgress struct {
	Percent int    `json:"percent"`
	Stage   string `json:"stage"`
}

type savedViewImportValidationResult struct {
	Valid        bool     `json:"valid"`
	ItemCount    int      `json:"itemCount"`
	ScopeCount   int      `json:"scopeCount"`
	Scopes       []string `json:"scopes"`
	Preview      []string `json:"preview"`
	InvalidCount int      `json:"invalidCount"`
	Warnings     []string `json:"warnings"`
}

func savedViewImportValidationCard(parseState atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]) ui.Node {
	if parseState.Error != nil {
		return html.Div(html.Props{Class: "grid gap-2 rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-rose-200"}, html.Text("Worker validation failed")),
			html.P(html.Props{Class: "text-sm leading-6 text-rose-100"}, html.Text(parseState.Error.Error())),
		)
	}
	if parseState.Running {
		parseStage := "validating saved-view payload"
		parseProgress := "Worker running"
		if parseState.ProgressReady {
			parseStage = fallback(parseState.Progress.Stage, parseStage)
			parseProgress = fmt.Sprintf("%d%% complete", parseState.Progress.Percent)
		}
		return html.Div(html.Props{Class: "grid gap-2 rounded-[1.35rem] border border-cyan-300/30 bg-cyan-300/10 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-cyan-100"}, html.Text("Worker validation in progress")),
			html.P(html.Props{Class: "text-sm leading-6 text-cyan-50"}, html.Text(parseProgress+" · "+parseStage)),
		)
	}
	if parseState.Ready {
		parseSummary := "Payload is ready to import."
		if !parseState.Value.Valid {
			parseSummary = "Payload needs fixes before import."
		}
		parsePreview := "No saved-view names detected yet."
		if len(parseState.Value.Preview) > 0 {
			parsePreview = strings.Join(parseState.Value.Preview, " · ")
		}
		parseScopeSummary := "No scopes detected."
		if len(parseState.Value.Scopes) > 0 {
			parseScopeSummary = strings.Join(parseState.Value.Scopes, ", ")
		}
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Worker import preview")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseSummary)),
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
				statCard("Items", fmt.Sprintf("%d", parseState.Value.ItemCount)),
				statCard("Scopes", fmt.Sprintf("%d", parseState.Value.ScopeCount)),
				statCard("Invalid", fmt.Sprintf("%d", parseState.Value.InvalidCount)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Preview: "+parsePreview)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scopes: "+parseScopeSummary)),
		}
		if len(parseState.Value.Warnings) > 0 {
			parseWarnings := make([]ui.Node, 0, len(parseState.Value.Warnings))
			for _, parseWarning := range parseState.Value.Warnings {
				parseWarnings = append(parseWarnings, html.P(html.Props{Class: "rounded-[1.1rem] border border-amber-400/25 bg-amber-400/10 px-3 py-2 text-sm text-amber-100"}, html.Text(parseWarning)))
			}
			parseChildren = append(parseChildren, html.Div(html.Props{Class: "grid gap-2"}, parseWarnings...))
		}
		return html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-slate-950/40 p-4"}, parseChildren...)
	}
	return html.Div(html.Props{Class: "grid gap-2 rounded-[1.35rem] border border-white/10 bg-slate-950/40 p-4"},
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-slate-300"}, html.Text("Worker import preview")),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text("Atlas will validate the saved-view import payload in a dedicated worker as you edit it.")),
	)
}

func operatorWorkspaceSnapshotCard(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(workspaceSnapshotFormState{WorkspaceSnapshotJSON: operatorWorkspaceSnapshotJSON(parsePayload)})
		parseValue := parseForm.Get()
		return html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Export workspace snapshot")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Copy the current operator shell, route workspace, and saved-view snapshot when a reviewer or another operator needs the same Atlas context without opening diagnostics mode.")),
			boundTextareaWithValue("workspace_snapshot_json", "Workspace snapshot", parseValue.WorkspaceSnapshotJSON, "WorkspaceSnapshotJSON", parseForm),
		)
	})
}

func operatorWorkspaceSnapshotJSON(parsePayload Payload) string {
	type savedViewSnapshotItem struct {
		Name          string            `json:"name"`
		Scope         string            `json:"scope"`
		SortKey       string            `json:"sortKey"`
		SortDirection string            `json:"sortDirection"`
		Density       string            `json:"density"`
		WarehouseID   string            `json:"warehouseId"`
		Filters       map[string]string `json:"filters"`
	}
	parsePresentation := currentShellPresentationState(parsePayload)
	parseWorkspace := currentRouteWorkspaceState(parsePayload)
	parseSavedViews := make([]savedViewSnapshotItem, 0, len(parsePayload.SavedViews))
	for _, parseSaved := range parsePayload.SavedViews {
		parseSavedViews = append(parseSavedViews, savedViewSnapshotItem{
			Name:          parseSaved.Name,
			Scope:         parseSaved.Scope,
			SortKey:       parseSaved.SortKey,
			SortDirection: parseSaved.SortDirection,
			Density:       parsePresentation.Density,
			WarehouseID:   fallback(parseSaved.Filters["warehouse"], parsePresentation.DefaultWarehouse),
			Filters:       parseSaved.Filters,
		})
	}
	parseExportPayload := map[string]any{
		"route": map[string]any{
			"path":   parsePayload.Route.Path,
			"screen": parsePayload.Route.Screen,
			"query":  parsePayload.Route.Query,
		},
		"shellPresentation": parsePresentation,
		"routeWorkspace":    parseWorkspace,
		"savedViews":        parseSavedViews,
	}
	parseEncoded, parseErr := json.MarshalIndent(parseExportPayload, "", "  ")
	if parseErr != nil {
		return `{"error":"workspace snapshot encode failed"}`
	}
	return string(parseEncoded)
}

func transferForm(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(transferFormState{
			SourceWarehouseID:      "nevada-hub",
			DestinationWarehouseID: "new-jersey-hub",
			Reason:                 "Support east-coast promise windows",
			RecommendedBy:          "Atlas Hydration Demo",
		})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Create transfer")),
			boundInputWithValue("source_warehouse_id", "Source warehouse", parseValue.SourceWarehouseID, "SourceWarehouseID", parseForm),
			boundInputWithValue("destination_warehouse_id", "Destination warehouse", parseValue.DestinationWarehouseID, "DestinationWarehouseID", parseForm),
			boundInputWithValue("reason", "Reason", parseValue.Reason, "Reason", parseForm),
			boundInputWithValue("recommended_by", "Recommended by", parseValue.RecommendedBy, "RecommendedBy", parseForm),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: parseOpenConfirm}, html.Text("Review transfer")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-transfer-confirm", "Confirm transfer plan", "Review the transfer before Atlas posts it so focus stays inside the confirmation step until you dismiss or submit.", "Create transfer", parseCloseConfirm,
				statCard("From", parseValue.SourceWarehouseID),
				statCard("To", parseValue.DestinationWarehouseID),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseValue.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/transfers", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func thresholdForm(parseItem inventoryRow, parsePayload Payload) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Adjust thresholds")),
		inputWithValue("warehouse_id", "Warehouse", parseItem.WarehouseID),
		inputWithValue("reorder_point", "Reorder point", "18"),
		inputWithValue("safety_stock", "Safety stock", "9"),
		submitButton("Save threshold"),
	}
	return html.Form(html.Props{Action: "/api/app/inventory/" + parseItem.SKU + "/threshold", Method: "post", Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
}

func receivingForm(parsePayload Payload) ui.Node {
	return receivingFormForID("rcv-illinois-001", parsePayload)
}

func receivingFormForID(parseSessionID string, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(receivingFormState{
			Status:             "closed",
			DiscrepancySummary: "Short shipment recorded and available stock released.",
		})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parsePrevious := ui.UsePrevious(parseValue)
		parseWorkflow := ui.UseReducer(reduceReceivingResolutionWorkflowState, normalizeReceivingResolutionWorkflowState(receivingResolutionWorkflowState{
			Status:             parseValue.Status,
			DiscrepancySummary: parseValue.DiscrepancySummary,
		}))
		parseWorkflowState := parseWorkflow.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			receivingResolutionSummaryCard(parseWorkflowState),
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Reconcile receiving")),
			receivingWorkflowInputWithValue("status", "Status", parseValue.Status, "Status", parseForm, parseWorkflow),
			receivingWorkflowTextareaWithValue("discrepancy_summary", "Discrepancy summary", parseValue.DiscrepancySummary, "DiscrepancySummary", parseForm, parseWorkflow),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: parseOpenConfirm}, html.Text("Review closeout")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-receiving-confirm", "Confirm receiving closeout", "Atlas keeps focus inside the discrepancy confirmation step until you cancel or submit the reconciliation.", "Close session", parseCloseConfirm,
				statCard("Status", parseValue.Status),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(fallback(parseValue.DiscrepancySummary, "No discrepancy note entered."))),
			),
		}
		if parseChangeSummary := receivingDraftChangeSummary(parseValue, parsePrevious); parseChangeSummary != nil {
			parseChildren = append([]ui.Node{parseChangeSummary}, parseChildren...)
		}
		return html.Form(html.Props{Action: "/api/app/receiving/" + parseSessionID + "/reconcile", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func receivingDraftChangeSummary(parseCurrent receivingFormState, parsePrevious ui.Previous[receivingFormState]) ui.Node {
	if !parsePrevious.Ok() {
		return nil
	}
	parsePrior := parsePrevious.Get()
	parseSummary := ""
	switch {
	case strings.TrimSpace(parsePrior.Status) != strings.TrimSpace(parseCurrent.Status):
		parseSummary = "Status changed from " + fallback(parsePrior.Status, "unset") + " to " + fallback(parseCurrent.Status, "unset")
	case strings.TrimSpace(parsePrior.DiscrepancySummary) != strings.TrimSpace(parseCurrent.DiscrepancySummary):
		parseSummary = "Discrepancy notes were updated for the current reconciliation draft."
	}
	if parseSummary == "" {
		return nil
	}
	return html.Div(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Recent reconcile edit")),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(parseSummary)),
	)
}

func receivingResolutionSummaryCard(parseState receivingResolutionWorkflowState) ui.Node {
	parseStageLabel := "Classification open"
	parseStageCopy := "Atlas is still waiting for the reconcile outcome and discrepancy note to settle."
	switch parseState.Stage {
	case "closeout-ready":
		parseStageLabel = "Closeout ready"
		parseStageCopy = "Status and discrepancy notes line up, so this session can close cleanly."
	case "closeout-needs-note":
		parseStageLabel = "Closeout needs note"
		parseStageCopy = "Closed sessions should still explain the discrepancy or closeout outcome before submit."
	case "discrepancy-review":
		parseStageLabel = "Discrepancy review"
		parseStageCopy = "The session is still under review, so Atlas treats this as an active exception workflow."
	}
	parseEdited := "Workflow staged from the current receiving defaults."
	if strings.TrimSpace(parseState.LastEditedField) != "" {
		parseEdited = "Last updated field: " + strings.ReplaceAll(parseState.LastEditedField, "_", " ") + "."
	}
	return html.Div(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(parseStageLabel)),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(parseStageCopy)),
		html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.2em] text-cyan-200/75"}, html.Text(parseEdited)),
	)
}

func receivingWorkflowInputWithValue(parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[receivingFormState], parseWorkflow ui.Reducer[receivingResolutionWorkflowState, receivingResolutionWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{
			ID: parseId, Name: parseName, Value: parseValue, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				parseNext := parseEvent.GetValue()
				parseForm.SetField(parseField, parseNext)
				parseWorkflow.Dispatch(receivingResolutionWorkflowAction{Field: parseName, Value: parseNext})
			}),
			Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"},
		}),
	)
}

func receivingWorkflowTextareaWithValue(parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[receivingFormState], parseWorkflow ui.Reducer[receivingResolutionWorkflowState, receivingResolutionWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{
			ID: parseId, Name: parseName, Value: parseValue, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				parseNext := parseEvent.GetValue()
				parseForm.SetField(parseField, parseNext)
				parseWorkflow.Dispatch(receivingResolutionWorkflowAction{Field: parseName, Value: parseNext})
			}),
			Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"},
		}, html.Text(parseValue)),
	)
}

func purchaseOrderStatusForm(parseId string, parseStatus string, parsePayload Payload) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Update order status")),
		inputWithValue("status", "Status", fallback(parseStatus, "submitted")),
		textareaWithValue("note", "Note", "Validated by Atlas operations."),
		submitButton("Update order"),
	}
	return html.Form(html.Props{Action: "/api/app/purchase-orders/" + parseId + "/status", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
}

func prependCSRFToken(parseToken string, parseChildren ...ui.Node) []ui.Node {
	if strings.TrimSpace(parseToken) == "" {
		return parseChildren
	}
	parseFieldName, parseFieldValue := ui.NewCSRFToken(parseToken).FormField()
	parseResult := make([]ui.Node, 0, len(parseChildren)+1)
	parseResult = append(parseResult, html.HiddenInput(parseFieldName, parseFieldValue))
	parseResult = append(parseResult, parseChildren...)
	return parseResult
}

func input(parseName, parseLabel string) ui.Node {
	return inputWithValue(parseName, parseLabel, "")
}

func publicInput(parseName, parseLabel string) ui.Node {
	return publicInputWithValue(parseName, parseLabel, "")
}

func publicInputWithValue(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func inputWithValue(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func boundInputWithValue[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func boundTransitionSelectWithValue[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T], parseTransition atlasTransition) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Select(html.Props{
			ID:   parseId,
			Name: parseName,
			OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) {
				atlasSetFormFieldInTransition(parseForm, parseField, parseEvent.GetValue())
			}),
			Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			Raw:   map[string]interface{}{"aria-labelledby": parseId + "-label", "data-transition": parseTransition.Pending()},
		}, parseChildren...),
	)
}

func publicTextarea(parseName, parseLabel string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Class: "min-h-28 rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func textareaWithValue(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func boundTextareaWithValue[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func preferenceDensityPreviewCard(parseValue string, isPending bool) ui.Node {
	parseDensity := atlasNormalizedFilterValue(parseValue)
	if parseDensity == "" {
		parseDensity = "compact"
	}
	parseClassName := "atlas-density-preview atlas-density-preview-compact"
	parseCopy := "Compact density keeps Atlas information-dense for operators working wide tables and queue summaries."
	if parseDensity == "comfortable" {
		parseClassName = "atlas-density-preview atlas-density-preview-comfortable"
		parseCopy = "Comfortable density increases whitespace so route summaries and action rails breathe more during review."
	}
	if isPending {
		parseCopy = "Applying the next density preview in a transition so the settings form stays responsive."
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-slate-950/45 p-4"},
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text("Density preview")),
		html.Div(html.Props{Class: parseClassName},
			html.Div(html.Props{Class: "rounded-full border border-cyan-300/35 bg-cyan-300/12 px-3 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-cyan-100"}, html.Text(strings.Title(parseDensity))),
			html.Div(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs uppercase tracking-[0.2em] text-slate-300"}, html.Text("Warehouse shell")),
			html.Div(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs uppercase tracking-[0.2em] text-slate-300"}, html.Text("Inventory rail")),
		),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
	)
}

func submitButton(parseLabel string) ui.Node {
	return html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text(parseLabel))
}

func commentNodes(parseItems []commentRecord) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.Subject+" · "+parseItem.Status)),
			html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseItem.ProductSKU+" · "+parseItem.AuthorName+" · "+strings.ReplaceAll(parseItem.AuthorType, "_", " "))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(parseItem.Body)),
		))
	}
	return parseNodes
}

func internalWorkflowSection(parseTitle, parseCopy string, parseCards ...ui.Node) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-workflow-sheet"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		parseChildren := []ui.Node{
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseTitle)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
			),
			html.Button(html.Props{Type: "button", Class: "inline-flex w-fit items-center rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 lg:hidden", OnClick: parseOpenDrawer}, html.Text("Open quick actions")),
			html.Div(html.Props{Class: "hidden gap-4 md:grid-cols-2 xl:grid-cols-3 lg:grid"}, parseCards...),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: "grid gap-4"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Quick actions")),
						html.P(html.Props{ID: parseTitleID, Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
						html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text(parseCopy)),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "grid gap-4"}, parseCards...),
			)),
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"}, parseChildren...)
	})
}

func internalWorkflowCard(parseStep, parseTitle, parseCopy, parseHref string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: "grid gap-3 " + internalAccentSurfaceClass() + " p-4 transition hover:border-cyan-300/55 hover:bg-[linear-gradient(180deg,rgba(10,24,42,0.96),rgba(7,14,26,0.99))]"},
		html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text(parseStep)),
		html.P(html.Props{Class: "text-base font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text("Open workflow")),
	)
}

func transferNodes(parseItems []transferRecord) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.SourceWarehouseID+" → "+parseItem.DestinationWarehouse)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(parseItem.Reason+" · "+parseItem.Status)),
		))
	}
	return parseNodes
}

func receivingNodes(parseItems []receivingRecord) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.ID+" · "+parseItem.Status)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(fallback(parseItem.DiscrepancySummary, "No discrepancies recorded."))),
		))
	}
	return parseNodes
}

func infoRow(parsePrimary string, parseSecondary string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parsePrimary)),
		html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(parseSecondary)),
	)
}

func activeNavLink(parseCurrentPath string, parseHref string) bool {
	parseCurrent := strings.TrimSpace(parseCurrentPath)
	parseTarget := strings.TrimSpace(parseHref)
	if parseCurrent == parseTarget {
		return true
	}
	if parseTarget == "/" {
		return false
	}
	return strings.HasPrefix(parseCurrent, parseTarget+"/")
}

func routeSummary(parsePath string) string {
	switch {
	case parsePath == "/":
		return "Atlas helps teams discover workspace products, compare regional delivery options, and request pricing without losing context."
	case strings.HasPrefix(parsePath, "/shop/"):
		return "This product page keeps pricing, delivery questions, and next steps together so buyers can decide with confidence."
	case parsePath == "/shop":
		return "Browse workspace systems with clear pricing cues, delivery context, and straightforward next steps."
	case strings.HasPrefix(parsePath, "/warehouses"):
		return "Explore Atlas delivery regions, compare service levels, and open the warehouse view that best fits your project timing."
	case strings.HasPrefix(parsePath, "/app"):
		return "Internal routes keep buyer inbox, merchandising, inventory, transfers, receiving, and preferences inside one server-backed Atlas console."
	default:
		return "Atlas route payload hydrated successfully from the server bootstrap."
	}
}

func payloadDataValue(parsePayload Payload, parseKey string) any {
	if parseRequest, parseOk := parsePayload.Requests[strings.TrimSpace(parseKey)]; parseOk && len(parseRequest.Data) > 0 {
		if parseValue, parseExists := parseRequest.Data[strings.TrimSpace(parseKey)]; parseExists {
			return parseValue
		}
	}
	if parsePayload.Data == nil {
		return nil
	}
	return parsePayload.Data[strings.TrimSpace(parseKey)]
}

func pageData(parsePayload Payload) any {
	return payloadDataValue(parsePayload, "page")
}

func decode[T any](parseValue any) T {
	var parseResult T
	if parseValue == nil {
		return parseResult
	}
	parseEncoded, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return parseResult
	}
	_ = json.Unmarshal(parseEncoded, &parseResult)
	return parseResult
}

func decodeJSONText[T any](parseValue string) T {
	var parseResult T
	if strings.TrimSpace(parseValue) == "" {
		return parseResult
	}
	_ = json.Unmarshal([]byte(parseValue), &parseResult)
	return parseResult
}

func fallback(parseValue string, parseDefaultValue string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return parseDefaultValue
	}
	return parseTrimmed
}

func firstQueryValue(parseValues map[string][]string, parseKey string) string {
	if len(parseValues[parseKey]) == 0 {
		return ""
	}
	return parseValues[parseKey][0]
}

func formatPrice(parseCents int) string {
	return fmt.Sprintf("$%0.2f", float64(parseCents)/100)
}

func sortedMapStrings(parseValues map[string]any) []string {
	if len(parseValues) == 0 {
		return nil
	}
	parseKeys := make([]string, 0, len(parseValues))
	for parseKey := range parseValues {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseLines := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseLines = append(parseLines, fmt.Sprintf("%s: %v", parseKey2, parseValues[parseKey2]))
	}
	return parseLines
}
