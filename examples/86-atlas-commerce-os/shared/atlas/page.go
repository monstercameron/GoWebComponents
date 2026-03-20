package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

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

func App(payload Payload) ui.Node {
	surface := strings.TrimSpace(payload.Route.Surface)
	rootClass := "min-h-screen bg-[radial-gradient(circle_at_top,rgba(245,158,11,0.16),transparent_22%),linear-gradient(180deg,rgba(7,10,18,1),rgba(13,18,30,1)_42%,rgba(7,10,18,1))] text-stone-100"
	mainClass := "mx-auto grid w-full max-w-7xl gap-8 px-5 pb-12 pt-6 sm:px-6 lg:px-10"
	if surface != "" && surface != "public" {
		rootClass = "min-h-screen bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.12),transparent_28%),linear-gradient(180deg,rgba(2,6,23,1),rgba(15,23,42,1))] text-white"
		mainClass = "mx-auto grid w-full max-w-6xl gap-8 px-6 pb-12 pt-6 lg:px-10"
	}
	desiredPresentation := shellPresentationStateFromPayload(payload)
	presentationAtom := useAtlasAtom(atlasShellPresentationAtomID, desiredPresentation)
	presentation := presentationAtom.Get()
	if presentation.RouteKey != desiredPresentation.RouteKey {
		presentation = desiredPresentation
	}
	useAtlasEffect(func() func() {
		if presentationAtom.Get() != desiredPresentation {
			presentationAtom.Set(desiredPresentation)
		}
		return nil
	}, desiredPresentation)
	useAtlasEffect(func() func() {
		_ = persistAtlasSnapshot(atlasShellPresentationStoreKey, atlasShellPresentationAtomID)
		return nil
	}, presentation)
	desiredWorkspace := useAtlasComputed(func() internalShellState {
		return internalShellStateForPayload(payload)
	}, payload.Route.Path, payload.Data, payload.SavedViews).Get()
	workspaceAtom := useAtlasAtom(atlasRouteWorkspaceAtomID, desiredWorkspace)
	shellState := workspaceAtom.Get()
	if shellState.RouteKey != desiredWorkspace.RouteKey {
		shellState = desiredWorkspace
	}
	useAtlasEffect(func() func() {
		workspaceAtom.Set(desiredWorkspace)
		return nil
	}, desiredWorkspace)
	toastChannel := useAtlasChannel(atlasShellToastBus)
	toastState := useAtlasState(atlasShellToast{})
	announcer := ui.UseAnnouncer()
	if toastChannel.Ok() {
		latest := toastChannel.Get()
		if latest != toastState.Get() {
			toastState.Set(latest)
		}
	}
	routeAnnouncement := strings.TrimSpace(fallback(payload.Route.Title, payload.Route.Screen))
	previousRouteAnnouncement := ui.UsePrevious(routeAnnouncement)
	useAtlasEffect(func() func() {
		if !previousRouteAnnouncement.Ok() || strings.TrimSpace(routeAnnouncement) == "" || previousRouteAnnouncement.Get() == routeAnnouncement {
			return nil
		}
		announcer.Polite("Route changed to " + routeAnnouncement + ".")
		return nil
	}, routeAnnouncement)
	noticeMessage := atlasNoticeMessage(payload)
	previousNotice := ui.UsePrevious(noticeMessage)
	useAtlasEffect(func() func() {
		if strings.TrimSpace(noticeMessage) == "" || (previousNotice.Ok() && previousNotice.Get() == noticeMessage) {
			return nil
		}
		announcer.Polite(noticeMessage)
		return nil
	}, noticeMessage)
	previousToast := ui.UsePrevious(toastState.Get())
	useAtlasEffect(func() func() {
		current := toastState.Get()
		if strings.TrimSpace(current.Title) == "" || (previousToast.Ok() && previousToast.Get() == current) {
			return nil
		}
		announcement := current.Title
		if detail := strings.TrimSpace(current.Detail); detail != "" {
			announcement += ". " + detail
		}
		announcer.Polite(announcement)
		return nil
	}, toastState.Get())
	useAtlasEffect(func() func() {
		current := toastState.Get()
		if strings.TrimSpace(current.Title) == "" {
			return nil
		}
		stop := make(chan struct{})
		go func(expected atlasShellToast) {
			select {
			case <-time.After(4 * time.Second):
				if toastState.Get() == expected {
					toastState.Set(atlasShellToast{})
				}
			case <-stop:
			}
		}(current)
		return func() {
			close(stop)
		}
	}, toastState.Get())
	children := []ui.Node{header(payload, shellState)}
	mainChildren := []ui.Node{}
	if toast := shellToastBanner(toastState.Get()); toast != nil {
		mainChildren = append(mainChildren, toast)
	}
	if banner := noticeBanner(payload); banner != nil {
		mainChildren = append(mainChildren, banner)
	}
	if diagnostics := atlasDiagnosticsPanel(payload, shellState, presentation); diagnostics != nil {
		mainChildren = append(mainChildren, diagnostics)
	}
	mainChildren = append(mainChildren, hero(payload, shellState, presentation), pageContent(payload))
	children = append(children, html.Main(html.Props{Class: mainClass}, mainChildren...))
	return html.Div(html.Props{},
		html.Div(html.Props{ID: atlasShellRootID, Class: rootClass}, children...),
		announcer.Region(),
		html.Div(html.Props{ID: atlasOverlayRootID}),
	)
}

func header(payload Payload, shellState internalShellState) ui.Node {
	links := []atlasPublicNavLink{
		{Label: publicNavStorefrontLabel, Href: RouteLanding},
		{Label: publicNavShopLabel, Href: RouteCatalog},
		{Label: publicNavWarehousesLabel, Href: RouteWarehouses},
	}
	if payload.User != nil {
		links = append(links,
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
	if payload.Route.Surface == "public" || payload.Route.Surface == "" {
		return publicHeader(payload, links)
	}
	return internalHeader(payload, shellState)
}

type atlasPublicNavLink struct {
	Label string
	Href  string
}

func publicHeader(payload Payload, links []atlasPublicNavLink) ui.Node {
	return ui.CreateElement(func() ui.Node {
		open := ui.UseState(false)
		sheetID := ui.UseId() + "-public-nav"
		titleID := sheetID + "-title"
		descriptionID := sheetID + "-description"
		closeID := sheetID + "-close"
		openDrawer := ui.UseEvent(func() { open.Set(true) })
		closeDrawer := func() { open.Set(false) }
		identity := publicIdentityLabel
		if payload.User != nil {
			identity = payload.User.DisplayName + " | " + strings.ReplaceAll(payload.User.Role, "_", " ")
		}
		desktopNodes := make([]ui.Node, 0, len(links))
		mobileNodes := make([]ui.Node, 0, len(links))
		for _, link := range links {
			className := "rounded-full border border-white/12 bg-white/6 px-4 py-2.5 text-sm font-medium text-stone-300 shadow-[0_10px_25px_rgba(0,0,0,0.16)] transition hover:border-amber-300/60 hover:bg-white/10 hover:text-white"
			mobileClassName := "flex items-center justify-between rounded-[1.35rem] border border-white/10 bg-white/5 px-4 py-4 text-left text-sm font-medium text-stone-200 transition hover:border-amber-300/45 hover:bg-white/10 hover:text-white"
			if activeNavLink(payload.Route.Path, link.Href) {
				className = "rounded-full border border-amber-300/70 bg-amber-300/12 px-4 py-2.5 text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
				mobileClassName = "flex items-center justify-between rounded-[1.35rem] border border-amber-300/60 bg-amber-300/12 px-4 py-4 text-left text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
			}
			desktopNodes = append(desktopNodes, html.A(html.Props{Href: link.Href, Class: className}, html.Text(link.Label)))
			mobileNodes = append(mobileNodes, html.A(html.Props{Href: link.Href, Class: mobileClassName}, html.Text(link.Label)))
		}
		return html.Header(html.Props{Class: "sticky top-0 z-20 border-b border-white/10 bg-[rgba(7,10,18,0.86)] backdrop-blur-xl"},
			html.Div(html.Props{Class: "mx-auto flex w-full max-w-7xl flex-col gap-4 px-5 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-10"},
				html.Div(html.Props{Class: "flex items-center justify-between gap-4"},
					html.Div(html.Props{Class: "flex flex-col"},
						html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.42em] text-amber-700"}, html.Text(publicBrandLabel)),
						html.P(html.Props{Class: "mt-2 text-sm text-stone-400"}, html.Text(identity)),
					),
					html.Div(html.Props{Class: "flex items-center gap-3 lg:hidden"},
						html.A(html.Props{Href: RouteCatalog, Class: "inline-flex rounded-full border border-amber-300/60 px-4 py-2 text-sm font-semibold text-amber-100 transition hover:bg-amber-300/12"}, html.Text(publicBrowseShopLabel)),
						html.Button(html.Props{Type: "button", Class: "inline-flex rounded-full border border-white/12 bg-white/8 px-4 py-2 text-sm font-semibold text-stone-100 transition hover:border-amber-300/45 hover:bg-white/12", OnClick: openDrawer}, html.Text("Menu")),
					),
				),
				html.Nav(html.Props{Class: "hidden flex-wrap items-center gap-2 lg:flex lg:justify-end"}, desktopNodes...),
			),
			atlasDismissibleSheet(open.Get(), sheetID, titleID, descriptionID, "#"+closeID, closeDrawer, html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-300"}, html.Text("Storefront navigation")),
						html.P(html.Props{ID: titleID, Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("Browse Atlas")),
						html.P(html.Props{ID: descriptionID, Class: "text-sm leading-7 text-stone-300"}, html.Text("Open the public menu to jump between landing, shop, and warehouse discovery routes without losing the current storefront context.")),
					),
					html.Button(html.Props{ID: closeID, Type: "button", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-stone-200", OnClick: ui.UseEvent(func() { closeDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "grid gap-3"}, mobileNodes...),
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

func internalHeader(payload Payload, shellState internalShellState) ui.Node {
	return ui.CreateElement(func() ui.Node {
		open := ui.UseState(false)
		sheetID := ui.UseId() + "-internal-nav"
		titleID := sheetID + "-title"
		descriptionID := sheetID + "-description"
		closeID := sheetID + "-close"
		openDrawer := ui.UseEvent(func() { open.Set(true) })
		closeDrawer := func() { open.Set(false) }
		identity := "Public browsing"
		if payload.User != nil {
			identity = payload.User.DisplayName + " | " + strings.ReplaceAll(payload.User.Role, "_", " ")
		}
		contextPills := internalHeaderContextPills(payload, shellState)
		groups := internalNavGroups()
		return html.Header(html.Props{Class: "sticky top-0 z-20 border-b border-white/10 bg-slate-950/88 backdrop-blur-xl"},
			html.Div(html.Props{Class: "mx-auto grid w-full max-w-7xl gap-4 px-5 py-4 sm:px-6 lg:px-10"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.42em] text-cyan-300"}, html.Text(publicBrandLabel)),
						html.H1(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fallback(payload.Route.Title, "Atlas workspace"))),
						html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text(routeSummary(payload.Route.Path))),
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-slate-400"}, html.Text(identity)),
					),
					html.Div(html.Props{Class: "flex items-center gap-3"},
						html.A(html.Props{Href: RouteLanding, Class: "hidden rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200 transition hover:border-cyan-300/50 hover:text-white sm:inline-flex"}, html.Text("View storefront")),
						html.Button(html.Props{Type: "button", Class: "inline-flex items-center rounded-full border border-cyan-300/40 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-400/16 lg:hidden", OnClick: openDrawer}, html.Text("Workspace nav")),
					),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-2"}, contextPills...),
				html.Div(html.Props{Class: "hidden gap-3 lg:grid lg:grid-cols-4"},
					internalNavGroupCard(payload, shellState, groups[0]),
					internalNavGroupCard(payload, shellState, groups[1]),
					internalNavGroupCard(payload, shellState, groups[2]),
					internalNavGroupCard(payload, shellState, groups[3]),
				),
				html.Div(html.Props{Class: "flex gap-2 overflow-x-auto pb-1 lg:hidden"},
					internalMobileQuickLinks(payload, shellState)...,
				),
			),
			atlasDismissibleSheet(open.Get(), sheetID, titleID, descriptionID, "#"+closeID, closeDrawer, html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Workspace navigation")),
						html.P(html.Props{ID: titleID, Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fallback(payload.Route.Title, "Atlas workspace"))),
						html.P(html.Props{ID: descriptionID, Class: "text-sm leading-7 text-slate-300"}, html.Text("Use the grouped mobile drawer to jump across overview, catalog, operations, and support routes without losing the current shell context.")),
					),
					html.Button(html.Props{ID: closeID, Type: "button", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { closeDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-2"}, contextPills...),
				html.Div(html.Props{Class: "grid gap-3"},
					internalNavGroupCard(payload, shellState, groups[0]),
					internalNavGroupCard(payload, shellState, groups[1]),
					internalNavGroupCard(payload, shellState, groups[2]),
					internalNavGroupCard(payload, shellState, groups[3]),
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

func internalHeaderContextPills(payload Payload, shellState internalShellState) []ui.Node {
	items := []ui.Node{
		internalHeaderContextPill("Surface", fallback(payload.Route.Surface, "app")),
		internalHeaderContextPill("Warehouse", fallback(payload.Preferences.DefaultWarehouse, "new-jersey-hub")),
	}
	if strings.TrimSpace(shellState.SummaryValue) != "" {
		items = append(items, internalHeaderContextPill(fallback(shellState.SummaryLabel, "Summary"), shellState.SummaryValue))
	}
	if strings.TrimSpace(shellState.ActiveSavedView) != "" {
		items = append(items, internalHeaderContextPill("Saved view", shellState.ActiveSavedView))
	}
	if len(shellState.ActiveFilters) > 0 {
		items = append(items, internalHeaderContextPill("Filters", fmt.Sprintf("%d active", len(shellState.ActiveFilters))))
	}
	return items
}

func internalHeaderContextPill(label, value string) ui.Node {
	return html.Div(html.Props{Class: internalSurfacePillClass()},
		html.Span(html.Props{Class: "text-slate-400"}, html.Text(label)),
		html.Span(html.Props{Class: "text-cyan-100"}, html.Text(value)),
	)
}

func internalNavGroupCard(payload Payload, shellState internalShellState, group atlasInternalNavGroup) ui.Node {
	links := make([]ui.Node, 0, len(group.Links))
	for _, link := range group.Links {
		links = append(links, internalNavLink(payload.Route.Path, shellState, link))
	}
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-4"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text(group.Label)),
		html.Div(html.Props{Class: "grid gap-2"}, links...),
	)
}

func internalNavLink(currentPath string, shellState internalShellState, link atlasInternalNavLink) ui.Node {
	className := "inline-flex items-center justify-between gap-3 " + internalInsetSurfaceClass() + " px-4 py-3 text-sm font-medium text-slate-200 transition hover:border-cyan-300/55 hover:text-white"
	if activeNavLink(currentPath, link.Href) {
		className = "inline-flex items-center justify-between gap-3 rounded-[1rem] border border-cyan-300/45 bg-[linear-gradient(180deg,rgba(8,26,42,0.96),rgba(7,14,26,0.98))] px-4 py-3 text-sm font-semibold text-cyan-100 shadow-[0_14px_30px_rgba(34,211,238,0.12)]"
	}
	children := []ui.Node{
		html.Span(html.Props{}, html.Text(link.Label)),
	}
	if badge := strings.TrimSpace(shellState.RouteBadges[link.Href]); badge != "" {
		children = append(children, html.Span(html.Props{Class: "rounded-full border border-cyan-300/35 bg-cyan-400/10 px-2 py-1 text-[0.65rem] font-semibold uppercase tracking-[0.16em] text-cyan-100"}, html.Text(badge)))
	}
	return html.A(html.Props{Href: link.Href, Class: className}, children...)
}

func internalMobileQuickLinks(payload Payload, shellState internalShellState) []ui.Node {
	links := []atlasInternalNavLink{
		{Label: "Dashboard", Href: RouteDashboard},
		{Label: "Inventory", Href: RouteInventory},
		{Label: "Products", Href: "/app/products"},
		{Label: "Warehouses", Href: RouteWarehouseOps},
	}
	nodes := make([]ui.Node, 0, len(links))
	for _, link := range links {
		nodes = append(nodes, internalNavLink(payload.Route.Path, shellState, link))
	}
	return nodes
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

func noticeBanner(payload Payload) ui.Node {
	notice := atlasNoticeMessage(payload)
	if notice == "" {
		return nil
	}
	if payload.Route.Surface == "public" || payload.Route.Surface == "" {
		return html.Div(html.Props{Class: "rounded-[1.75rem] border border-emerald-400/30 bg-emerald-400/10 px-5 py-4 text-sm font-medium text-emerald-100 shadow-[0_18px_40px_rgba(16,185,129,0.12)]"}, html.Text(notice))
	}
	return html.Div(html.Props{Class: "rounded-3xl border border-emerald-300/25 bg-emerald-400/10 px-5 py-4 text-sm text-emerald-100"}, html.Text(notice))
}

func atlasNoticeMessage(payload Payload) string {
	return strings.ReplaceAll(firstQueryValue(payload.Route.Query, "atlas_notice"), "+", " ")
}

func shellToastBanner(toast atlasShellToast) ui.Node {
	if strings.TrimSpace(toast.Title) == "" {
		return nil
	}
	className := "grid gap-2 rounded-[1.6rem] border px-5 py-4 text-sm shadow-[0_18px_40px_rgba(2,6,23,0.18)]"
	switch strings.TrimSpace(strings.ToLower(toast.Tone)) {
	case "success":
		className += " border-emerald-400/30 bg-emerald-400/12 text-emerald-50"
	case "warn", "warning", "warm":
		className += " border-amber-400/30 bg-amber-400/12 text-amber-50"
	case "danger", "error":
		className += " border-rose-400/30 bg-rose-400/12 text-rose-50"
	default:
		className += " border-cyan-300/30 bg-cyan-300/12 text-cyan-50"
	}
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em]"}, html.Text(toast.Title)),
	}
	if strings.TrimSpace(toast.Detail) != "" {
		children = append(children, html.P(html.Props{Class: "text-sm leading-6"}, html.Text(toast.Detail)))
	}
	return html.Div(html.Props{Class: className}, children...)
}

func atlasDiagnosticsPanel(payload Payload, shellState internalShellState, presentation shellPresentationState) ui.Node {
	if !presentation.Diagnostics || payload.User == nil {
		return nil
	}
	return ui.CreateElement(func() ui.Node {
		metrics := useAtlasViewportMetrics()
		throttled := useAtlasThrottled(metrics, 180*time.Millisecond)
		current := throttled.Get()
		stickyState := "top of shell"
		if current.ScrollY > 48 {
			stickyState = "sticky header engaged"
		}
		workspaceState := "general shell route"
		if strings.HasPrefix(payload.Route.Path, RouteInventory) {
			workspaceState = "inventory workspace in view"
		} else if strings.HasPrefix(payload.Route.Path, RouteWarehouseOps) {
			workspaceState = "warehouse workspace in view"
		}
		copy := "Atlas is sampling shell viewport and sticky-state diagnostics through a throttled stream so scroll and resize bursts do not redraw the panel on every browser event."
		if throttled.Pending() {
			copy = "Atlas is rate-limiting the latest viewport measurement while scroll or resize events are still arriving."
		}
		return html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-cyan-300/20 bg-cyan-400/8 p-5"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-cyan-200"}, html.Text("Atlas diagnostics")),
				html.P(html.Props{Class: "text-sm leading-6 text-cyan-50"}, html.Text(copy)),
			),
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
				statCard("Scroll Y", fmt.Sprintf("%dpx", current.ScrollY)),
				statCard("Viewport", fmt.Sprintf("%dx%d", current.Width, current.Height)),
				statCard("Sticky shell", stickyState),
				statCard("Samples", fmt.Sprintf("%d", current.SampleCount)),
			),
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
				statCard("Route", fallback(payload.Route.Path, "/")),
				statCard("Workspace", workspaceState),
				statCard("Measured", fallback(current.MeasuredAtUTC, "Awaiting browser metrics")),
			),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-100/80"}, html.Text("Shell summary: "+fallback(shellState.SummaryLabel, "route")+" | "+fallback(shellState.SummaryValue, "n/a"))),
		)
	})
}

func hero(payload Payload, shellState internalShellState, presentation shellPresentationState) ui.Node {
	if payload.Route.Surface == "public" || payload.Route.Surface == "" {
		return renderPublicHero(payload)
	}
	meta := []ui.Node{
		statCard("Surface", fallback(payload.Route.Surface, "public")),
		statCard("Theme", fallback(presentation.Theme, "dark")),
		statCard("Locale", fallback(presentation.Locale, "en")),
		statCard("Density", fallback(presentation.Density, "compact")),
	}
	meta = append(meta, statCard("Warehouse", fallback(presentation.DefaultWarehouse, "new-jersey-hub")))
	if presentation.Diagnostics {
		meta = append(meta, statCard("Diagnostics", "on"))
	}
	if strings.TrimSpace(shellState.ActiveSavedView) != "" {
		meta = append(meta, statCard("Saved view", shellState.ActiveSavedView))
	}
	if len(shellState.ActiveFilters) > 0 {
		meta = append(meta, statCard("Filters", fmt.Sprintf("%d active", len(shellState.ActiveFilters))))
	}
	if strings.TrimSpace(shellState.SummaryValue) != "" {
		meta = append([]ui.Node{statCard(fallback(shellState.SummaryLabel, "Shell summary"), shellState.SummaryValue)}, meta...)
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.8fr)]"},
		html.Div(html.Props{Class: internalHeroSurfaceClass()},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.35em] text-cyan-300"}, html.Text(fallback(payload.Route.Screen, "route"))),
			html.H1(html.Props{Class: "mt-4 text-4xl font-black tracking-tight text-white lg:text-5xl"}, html.Text(fallback(payload.Route.Title, "Atlas"))),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-base leading-7 text-slate-300"}, html.Text(routeSummary(payload.Route.Path))),
		),
		html.Div(html.Props{Class: "grid gap-4"}, meta...),
	)
}

func shellPresentationStateFromPayload(payload Payload) shellPresentationState {
	return shellPresentationState{
		RouteKey:         shellRouteKey(payload.Route.Path, payload.Route.Query),
		Theme:            payload.Theme.Mode,
		Locale:           payload.I18n.Locale,
		Density:          payload.Preferences.Density,
		DefaultWarehouse: payload.Preferences.DefaultWarehouse,
		Diagnostics:      strings.EqualFold(strings.TrimSpace(firstQueryValue(payload.Route.Query, "diag")), "1"),
	}
}

func currentShellPresentationState(payload Payload) shellPresentationState {
	return useAtlasAtom(atlasShellPresentationAtomID, shellPresentationStateFromPayload(payload)).Get()
}

func currentRouteWorkspaceState(payload Payload) internalShellState {
	return useAtlasAtom(atlasRouteWorkspaceAtomID, internalShellStateForPayload(payload)).Get()
}

func shellRouteKey(path string, query map[string][]string) string {
	values := url.Values{}
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func internalShellStateForPayload(payload Payload) internalShellState {
	state := internalShellState{
		RouteKey:    shellRouteKey(payload.Route.Path, payload.Route.Query),
		RouteBadges: map[string]string{},
	}
	if payload.User == nil {
		return state
	}
	path := strings.TrimSpace(payload.Route.Path)
	switch {
	case path == RouteDashboard:
		page := decode[dashboardPage](pageData(payload))
		state.SummaryLabel = "Open alerts"
		state.SummaryValue = fmt.Sprintf("%d", page.Alerts)
		state.RouteBadges[RouteDashboard] = fmt.Sprintf("%d", page.Alerts)
		state.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(page.Transfers))
		state.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(page.Receiving))
		state.RouteBadges[RouteComments] = fmt.Sprintf("%d", len(page.Comments))
		state.WorkspaceStats = []pageSummaryItem{
			{Label: "Alerts", Value: fmt.Sprintf("%d", page.Alerts)},
			{Label: "Transfers", Value: fmt.Sprintf("%d", len(page.Transfers))},
			{Label: "Receiving", Value: fmt.Sprintf("%d", len(page.Receiving))},
			{Label: "Comments", Value: fmt.Sprintf("%d", len(page.Comments))},
		}
	case path == "/app/products":
		page := decode[productCMSPageData](pageData(payload))
		total := page.Total
		if total == 0 {
			total = len(page.Items)
		}
		state.SummaryLabel = "Catalog items"
		state.SummaryValue = fmt.Sprintf("%d", total)
		state.RouteBadges["/app/products"] = fmt.Sprintf("%d", total)
	case path == RouteInventory:
		page := decode[inventoryCMSPage](pageData(payload))
		workspace := inventoryWorkspaceSnapshotFromRows(page.Items)
		state.SummaryLabel = "Risk lanes"
		state.SummaryValue = fmt.Sprintf("%d", workspace.Rollup.RiskLanes)
		state.RouteBadges[RouteInventory] = fmt.Sprintf("%d", workspace.Rollup.RiskLanes)
		state.ActiveFilters = inventoryFilterSummaryLabels(page.Filters)
		state.ActiveSavedView = matchingInventorySavedViewName(payload, page.Filters)
		state.WorkspaceStats = []pageSummaryItem{
			{Label: "Visible SKUs", Value: fmt.Sprintf("%d", len(workspace.Summaries))},
			{Label: "Visible lanes", Value: fmt.Sprintf("%d", workspace.Rollup.VisibleLanes)},
			{Label: "Risk lanes", Value: fmt.Sprintf("%d", workspace.Rollup.RiskLanes)},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", workspace.Rollup.InboundUnits)},
		}
	case strings.HasPrefix(path, RouteInventory+"/"):
		page := decode[inventoryDetailPage](pageData(payload))
		rollup := inventoryRollupFromRows(page.Rows)
		state.SummaryLabel = "Tracked lanes"
		state.SummaryValue = fmt.Sprintf("%d", rollup.VisibleLanes)
		state.RouteBadges[RouteInventory] = fmt.Sprintf("%d", rollup.RiskLanes)
	case path == RouteWarehouseOps:
		page := decode[warehouseOpsList](pageData(payload))
		riskCount := countRiskWarehouses(page.Items)
		state.SummaryLabel = "Risk facilities"
		state.SummaryValue = fmt.Sprintf("%d", riskCount)
		state.RouteBadges[RouteWarehouseOps] = fmt.Sprintf("%d", riskCount)
	case strings.HasPrefix(path, RouteWarehouseOps+"/"):
		page := decode[warehouseInventoryDetailPage](pageData(payload))
		if strings.TrimSpace(page.Warehouse.ID) != "" {
			workspace := warehouseDetailWorkspaceSnapshotFromPage(page)
			state.SummaryLabel = "Warehouse filters"
			state.SummaryValue = fmt.Sprintf("%d", len(warehouseFilterSummaryLabels(page.Filters)))
			state.RouteBadges[RouteWarehouseOps] = fmt.Sprintf("%d", workspace.RiskLanes)
			state.ActiveFilters = warehouseFilterSummaryLabels(page.Filters)
			state.WorkspaceStats = []pageSummaryItem{
				{Label: "Visible items", Value: fmt.Sprintf("%d", len(page.Inventory))},
				{Label: "Risk lanes", Value: fmt.Sprintf("%d", workspace.RiskLanes)},
				{Label: "Open orders", Value: fmt.Sprintf("%d", len(page.Orders))},
				{Label: "Weekly demand", Value: fmt.Sprintf("%d", workspace.TotalDemand)},
			}
		}
	case path == RouteTransfers:
		page := decode[transferList](pageData(payload))
		state.SummaryLabel = "Active transfers"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Items))
		state.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(page.Items))
	case strings.HasPrefix(path, RouteTransfers+"/"):
		page := decode[transferDetailPage](pageData(payload))
		state.SummaryLabel = "Transfer lines"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Lines))
		state.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(page.Lines))
	case path == RoutePurchaseOrders:
		page := decode[purchaseOrderList](pageData(payload))
		state.SummaryLabel = "Open purchase orders"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Items))
		state.RouteBadges[RoutePurchaseOrders] = fmt.Sprintf("%d", len(page.Items))
	case strings.HasPrefix(path, RoutePurchaseOrders+"/"):
		page := decode[purchaseOrderDetailPage](pageData(payload))
		state.SummaryLabel = "Inbound lines"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Lines))
		state.RouteBadges[RoutePurchaseOrders] = fmt.Sprintf("%d", len(page.Lines))
	case path == RouteReceiving:
		page := decode[receivingList](pageData(payload))
		state.SummaryLabel = "Receiving sessions"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Items))
		state.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(page.Items))
	case strings.HasPrefix(path, RouteReceiving+"/"):
		page := decode[receivingDetailPage](pageData(payload))
		state.SummaryLabel = "Receiving lines"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Lines))
		state.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(page.Lines))
	case path == RouteComments:
		page := decode[commentList](pageData(payload))
		state.SummaryLabel = "Visible comments"
		state.SummaryValue = fmt.Sprintf("%d", len(page.Items))
		state.RouteBadges[RouteComments] = fmt.Sprintf("%d", len(page.Items))
	case path == RouteSettings:
		state.SummaryLabel = "Saved views"
		state.SummaryValue = fmt.Sprintf("%d", len(payload.SavedViews))
		state.RouteBadges[RouteSettings] = fmt.Sprintf("%d", len(payload.SavedViews))
	}
	return state
}

func countRiskWarehouses(items []warehouseOpsRecord) int {
	count := 0
	for _, item := range items {
		if item.RiskCount > 0 {
			count++
		}
	}
	return count
}

func inventoryFilterSummaryLabels(filters map[string]string) []string {
	labels := []string{}
	if value := strings.TrimSpace(filters["q"]); value != "" {
		labels = append(labels, "Search: "+value)
	}
	if value := strings.TrimSpace(filters["warehouse"]); value != "" && !strings.EqualFold(value, "all") {
		labels = append(labels, "Warehouse: "+value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" && !strings.EqualFold(value, "all") {
		labels = append(labels, "Status: "+strings.ReplaceAll(value, "_", " "))
	}
	if value := strings.TrimSpace(filters["sort"]); value != "" {
		labels = append(labels, "Sort: "+value)
	}
	return labels
}

func warehouseFilterSummaryLabels(filters map[string]string) []string {
	labels := []string{}
	if value := strings.TrimSpace(filters["q"]); value != "" {
		labels = append(labels, "Search: "+value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" && !strings.EqualFold(value, "all") {
		labels = append(labels, "Status: "+strings.ReplaceAll(value, "_", " "))
	}
	if value := strings.TrimSpace(filters["sort"]); value != "" {
		labels = append(labels, "Sort: "+value)
	}
	return labels
}

func matchingInventorySavedViewName(payload Payload, filters map[string]string) string {
	currentWarehouse := strings.TrimSpace(filters["warehouse"])
	currentStatus := strings.TrimSpace(filters["status"])
	currentSort := strings.TrimSpace(filters["sort"])
	for _, saved := range payload.SavedViews {
		if !strings.EqualFold(strings.TrimSpace(saved.Scope), "inventory") {
			continue
		}
		savedWarehouse := strings.TrimSpace(saved.Filters["warehouse"])
		savedStatus := strings.TrimSpace(saved.Filters["status"])
		if savedStatus == "" {
			savedStatus = strings.TrimSpace(saved.Filters["stock-health"])
		}
		matchesWarehouse := savedWarehouse == "" || strings.EqualFold(savedWarehouse, currentWarehouse)
		matchesStatus := savedStatus == "" || strings.EqualFold(savedStatus, currentStatus)
		matchesSort := strings.TrimSpace(saved.SortKey) == "" || strings.EqualFold(strings.TrimSpace(saved.SortKey), currentSort)
		if matchesWarehouse && matchesStatus && matchesSort {
			return saved.Name
		}
	}
	return ""
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
		return renderWarehouseDirectoryContent(decode[warehouseDirectoryPage](pageData(payload)))
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
			return warehouseOpsDetailContent(payload)
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
			return renderWarehouseDetailContent(page, payload)
		}
		return fallbackContent(payload)
	}
}

func dashboardContent(payload Payload) ui.Node {
	page := decode[dashboardPage](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6"},
		dashboardSummaryBand(page),
		html.Div(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.18fr)_minmax(22rem,0.82fr)] xl:items-start"},
			html.Div(html.Props{Class: "grid gap-6"},
				dashboardActionCluster(),
				dashboardActivityFeed(page),
			),
			html.Div(html.Props{Class: "grid gap-6"},
				dashboardAttentionPanel(page),
				dashboardPurchaseOrderSummary(page.Orders),
			),
		),
	)
}

func dashboardSummaryBand(page dashboardPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Dashboard summary band")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("The dashboard should read like a triage surface first: one summary band, one action cluster, one attention stack, one activity feed, and one purchase-order watch panel.")),
		),
		routeSummaryStrip(page.Summary),
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

func dashboardAttentionPanel(page dashboardPage) ui.Node {
	pendingComments := dashboardCommentStatusCount(page.Comments, "pending")
	flaggedComments := dashboardCommentStatusCount(page.Comments, "flagged")
	openReceiving := dashboardOpenReceivingCount(page.Receiving)
	submittedOrders := dashboardPurchaseOrderStatusCount(page.Orders, "submitted")
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("High-attention panel")),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fmt.Sprintf("%d alerts need a route decision.", page.Alerts))),
			html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text("These are the work queues most likely to change customer promise, inbound readiness, or moderation posture if an operator waits too long.")),
		),
		html.Div(html.Props{Class: "grid gap-3"},
			dashboardAttentionLink("Pending comments", fmt.Sprintf("%d waiting", pendingComments), "Review questions and moderation decisions that are shaping buyer follow-up right now.", "/app/comments?status=pending"),
			dashboardAttentionLink("Flagged comments", fmt.Sprintf("%d flagged", flaggedComments), "Handle risky or unclear public notes before they create merch or support confusion.", "/app/comments?status=flagged"),
			dashboardAttentionLink("Receiving closeout", fmt.Sprintf("%d open sessions", openReceiving), "Resolve discrepancies and close receiving sessions so inbound stock can become trustworthy availability.", "/app/receiving"),
			dashboardAttentionLink("Submitted purchase orders", fmt.Sprintf("%d need review", submittedOrders), "Move draft or submitted vendor work forward before the replenishment lane stalls.", "/app/purchase-orders"),
		),
	)
}

func dashboardAttentionLink(title, value, copy, href string) ui.Node {
	return html.A(html.Props{Href: href, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
		html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(title)),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(value)),
		),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
	)
}

func dashboardActivityFeed(page dashboardPage) ui.Node {
	items := dashboardActivityItems(page)
	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, html.A(html.Props{Href: item.Href, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(item.Kicker)),
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.18em] text-slate-400"}, html.Text(item.Meta)),
			),
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.Title)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Detail)),
		))
	}
	if len(nodes) == 0 {
		nodes = append(nodes, html.Div(html.Props{Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text("No operator activity queued.")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Comments, transfers, receiving sessions, and purchase orders will populate this feed as soon as Atlas has active work.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Activity feed")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The dashboard feed should show the latest moderation, transfer, receiving, and purchase-order motion in one scan instead of making operators open four routes just to check momentum.")),
		),
		html.Div(html.Props{Class: "grid gap-3"}, nodes...),
	)
}

type dashboardActivityItem struct {
	Kicker string
	Title  string
	Detail string
	Meta   string
	Href   string
}

func dashboardActivityItems(page dashboardPage) []dashboardActivityItem {
	items := []dashboardActivityItem{}
	for index, item := range page.Comments {
		if index >= 2 {
			break
		}
		items = append(items, dashboardActivityItem{
			Kicker: "Moderation",
			Title:  fallback(item.Subject, "Buyer question"),
			Detail: fallback(item.Body, "Public feedback needs a routing decision."),
			Meta:   strings.ReplaceAll(fallback(item.Status, "pending"), "_", " "),
			Href:   "/app/comments",
		})
	}
	for index, item := range page.Transfers {
		if index >= 2 {
			break
		}
		items = append(items, dashboardActivityItem{
			Kicker: "Transfer",
			Title:  item.SourceWarehouseID + " to " + item.DestinationWarehouse,
			Detail: fallback(item.Reason, "Balancing action in flight."),
			Meta:   strings.ReplaceAll(fallback(item.Status, "submitted"), "_", " "),
			Href:   "/app/transfers/" + item.ID,
		})
	}
	for index, item := range page.Receiving {
		if index >= 2 {
			break
		}
		items = append(items, dashboardActivityItem{
			Kicker: "Receiving",
			Title:  item.ID,
			Detail: fallback(item.DiscrepancySummary, "Inbound session still needs closeout."),
			Meta:   strings.ReplaceAll(fallback(item.Status, "open"), "_", " "),
			Href:   "/app/receiving/" + item.ID,
		})
	}
	for index, item := range page.Orders {
		if index >= 2 {
			break
		}
		items = append(items, dashboardActivityItem{
			Kicker: "Purchase order",
			Title:  fallback(item.VendorName, item.ID),
			Detail: fallback(item.PriorityNote, "Vendor replenishment still in motion."),
			Meta:   strings.ReplaceAll(fallback(item.Status, "submitted"), "_", " "),
			Href:   "/app/purchase-orders/" + item.ID,
		})
	}
	return items
}

func dashboardPurchaseOrderSummary(orders []purchaseOrderRecord) ui.Node {
	nodes := make([]ui.Node, 0, len(orders))
	for index, item := range orders {
		if index >= 4 {
			break
		}
		nodes = append(nodes, html.A(html.Props{Href: "/app/purchase-orders/" + item.ID, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(fallback(item.VendorName, item.ID))),
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(strings.ReplaceAll(fallback(item.Status, "submitted"), "_", " "))),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(fallback(item.PriorityNote, "Vendor replenishment still needs operator follow-through."))),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.18em] text-slate-400"}, html.Text(fallback(item.WarehouseName, item.WarehouseID)+" | ETA "+fallback(item.ETA, "pending"))),
		))
	}
	if len(nodes) == 0 {
		nodes = append(nodes, html.Div(html.Props{Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text("No purchase orders are open.")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Atlas will list the highest-priority vendor work here once replenishment needs a formal PO lane.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Purchase-order summary")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep vendor work visible from the dashboard so replenishment planning does not disappear behind the logistics route boundary.")),
		),
		html.Div(html.Props{Class: "grid gap-3"}, nodes...),
	)
}

func dashboardCommentStatusCount(items []commentRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
			count++
		}
	}
	return count
}

func dashboardOpenReceivingCount(items []receivingRecord) int {
	count := 0
	for _, item := range items {
		status := strings.TrimSpace(strings.ToLower(item.Status))
		if status != "reconciled" && status != "closed" {
			count++
		}
	}
	return count
}

func dashboardPurchaseOrderStatusCount(items []purchaseOrderRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
			count++
		}
	}
	return count
}

func inventoryContent(payload Payload) ui.Node {
	return inventoryCMSContent(payload)
}

func skuContent(payload Payload) ui.Node {
	return inventoryDetailContent(payload)
}

func warehouseOpsContent(payload Payload) ui.Node {
	page := decode[warehouseOpsList](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.16fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps, Current: true},
			),
			warehouseOpsSummaryBand(page),
			warehouseOpsActionCluster(),
			warehouseOpsTable(page.Items),
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

func warehouseOpsSummaryBand(page warehouseOpsList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Warehouse operations shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat the warehouse route as a facility triage board first: visible backlog pressure, clear service posture, and direct drill-ins into the facility that actually needs action.")),
		),
		routeSummaryStrip(page.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Facilities", fmt.Sprintf("%d active", len(page.Items))),
			statCard("Available", fmt.Sprintf("%d units", totalWarehouseAvailable(page.Items))),
			statCard("Inbound", fmt.Sprintf("%d units", totalWarehouseInbound(page.Items))),
			statCard("Risk facilities", fmt.Sprintf("%d flagged", countRiskWarehouses(page.Items))),
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

func warehouseOpsTable(items []warehouseOpsRecord) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, warehouseOpsTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No warehouses are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Facility table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scan service posture, backlog, inbound exposure, and risk count in one dense table before drilling into a specific facility workspace.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d facilities", len(items)))),
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func warehouseOpsTableRow(item warehouseOpsRecord) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + item.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.Name)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(item.Region+" | "+item.ServiceLevel)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Focus)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(item.Pressure+" | "+item.Backlog)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Available))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Inbound))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d active", item.RiskCount))),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + item.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open workspace")),
				html.A(html.Props{Href: "/app/warehouses/" + item.ID + "#warehouse-replenishment", Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open replenishment")),
			),
		),
	)
}

func totalWarehouseAvailable(items []warehouseOpsRecord) int {
	total := 0
	for _, item := range items {
		total += item.Available
	}
	return total
}

func totalWarehouseInbound(items []warehouseOpsRecord) int {
	total := 0
	for _, item := range items {
		total += item.Inbound
	}
	return total
}

func warehouseOpsDetailContent(payload Payload) ui.Node {
	return warehouseInventoryDetailContent(payload)
}

func transfersContent(payload Payload) ui.Node {
	page := decode[transferList](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			transfersSummaryBand(page),
			transfersActionCluster(),
			transfersTable(page.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			transferForm(payload),
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

func transfersSummaryBand(page transferList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Transfer shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat transfers as a balancing board: source and destination lanes, recommendation context, and direct handoff into receiving once the movement is committed.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Transfers", fmt.Sprintf("%d active", len(page.Items))),
			statCard("Pending", fmt.Sprintf("%d queued", countTransferStatus(page.Items, "pending"))),
			statCard("Approved", fmt.Sprintf("%d moving", countTransferStatus(page.Items, "approved"))),
			statCard("Cancelled", fmt.Sprintf("%d dropped", countTransferStatus(page.Items, "cancelled"))),
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

func transfersTable(items []transferRecord) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, transferTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No transfer recommendations are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Transfer table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review source, destination, status, and recommendation context in one dense table before opening a transfer detail route.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d transfers", len(items)))),
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func transferTableRow(item transferRecord) ui.Node {
	lane := item.SourceWarehouseID + " -> " + item.DestinationWarehouse
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/transfers/" + item.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.ID)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(lane)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(item.Reason)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fallback(item.RecommendedBy, "Atlas planning"))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(item.UpdatedAt)),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.A(html.Props{Href: "/app/transfers/" + item.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open transfer"))),
	)
}

func countTransferStatus(items []transferRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
			count++
		}
	}
	return count
}

func transferDetailContent(payload Payload) ui.Node {
	page := decode[transferDetailPage](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			transferDetailHero(page),
			transferLineTable(page.Lines),
		),
		html.Div(html.Props{Class: "grid gap-4"},
			routeRevalidationCard("Transfer route refresh", "Re-run the transfer loader after approval or cancellation work if you want to confirm the latest lane state without leaving the detail route."),
			statCard("Lane", page.Transfer.SourceWarehouseID+" -> "+page.Transfer.DestinationWarehouse),
			statCard("Status", page.Transfer.Status),
			statCard("Recommended by", fallback(page.Transfer.RecommendedBy, "Atlas planning")),
		),
	)
}

func transferDetailHero(page transferDetailPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Transfer workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(page.Transfer.ID)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(page.Transfer.Reason)),
				),
			),
			html.A(html.Props{Href: "/app/transfers", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to transfers")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Transfer.SourceWarehouseID+" -> "+page.Transfer.DestinationWarehouse)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(strings.ReplaceAll(page.Transfer.Status, "_", " "))),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fallback(page.Transfer.RecommendedBy, "Atlas planning"))),
		),
	)
}

func transferLineTable(lines []transferLineRecord) ui.Node {
	rows := make([]ui.Node, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(line.ProductSKU)),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d units", line.Quantity))),
		))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func routeRevalidationCard(title string, detail string) ui.Node {
	revalidator := useAtlasRevalidator()
	label := "Refresh route data"
	if revalidator.Loading() {
		label = "Refreshing route..."
	}
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-1"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(title)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(detail)),
		),
		html.Button(html.Props{
			Type:     "button",
			Class:    "inline-flex items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-300/15 hover:text-white disabled:cursor-not-allowed disabled:border-slate-700 disabled:bg-slate-900 disabled:text-slate-500",
			Disabled: revalidator.Loading(),
			OnClick:  ui.UseEvent(func() { revalidator.Revalidate() }),
		}, html.Text(label)),
	)
}

func purchaseOrdersContent(payload Payload) ui.Node {
	page := decode[purchaseOrderList](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.14fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			purchaseOrdersSummaryBand(page),
			purchaseOrdersActionCluster(),
			purchaseOrdersTable(page.Items),
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

func purchaseOrdersSummaryBand(page purchaseOrderList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Purchase-order shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat purchase orders as a vendor-side recovery board: visible status posture, ETA clarity, and direct movement into approval, hold, and receiving follow-through.")),
		),
		routeSummaryStrip(page.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Orders", fmt.Sprintf("%d active", len(page.Items))),
			statCard("Submitted", fmt.Sprintf("%d queued", countPurchaseOrdersByStatus(page.Items, "submitted"))),
			statCard("Approved", fmt.Sprintf("%d inbound", countPurchaseOrdersByStatus(page.Items, "approved"))),
			statCard("On hold", fmt.Sprintf("%d blocked", countPurchaseOrdersByStatus(page.Items, "on_hold"))),
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

func purchaseOrdersTable(items []purchaseOrderRecord) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, purchaseOrdersTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 7}}, html.Text("No purchase orders are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Vendor order table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scan vendor, owning warehouse, ETA posture, and status decisions in one dense table before opening the order detail route.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d orders", len(items)))),
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func purchaseOrdersTableRow(item purchaseOrderRecord) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/purchase-orders/" + item.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.ID)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.VendorName)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(item.WarehouseName)),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(item.ETA)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(fallback(item.PriorityNote, "No priority note"))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(item.UpdatedAt)),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.A(html.Props{Href: "/app/purchase-orders/" + item.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open order")),
		),
	)
}

func countPurchaseOrdersByStatus(items []purchaseOrderRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
			count++
		}
	}
	return count
}

func purchaseOrderDetailContent(payload Payload) ui.Node {
	page := decode[purchaseOrderDetailPage](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			purchaseOrderDetailHero(page),
			purchaseOrderLineTable(page.Lines),
		),
		purchaseOrderDetailRail(payload, page),
	)
}

func purchaseOrderDetailHero(page purchaseOrderDetailPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Purchase-order workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(page.Order.VendorName)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fallback(page.Order.PriorityNote, "Vendor replenishment lane"))),
				),
			),
			html.A(html.Props{Href: "/app/purchase-orders", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to PO table")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Order.ID)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Order.WarehouseName)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Order.ETA)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(strings.ReplaceAll(page.Order.Status, "_", " "))),
		),
	)
}

func purchaseOrderLineTable(lines []purchaseOrderLineRecord) ui.Node {
	rows := make([]ui.Node, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(line.ProductSKU)),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d units", line.Quantity))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(line.ETA)),
			html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(line.Status)}, html.Text(strings.ReplaceAll(line.Status, "_", " ")))),
		))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func receivingContent(payload Payload) ui.Node {
	page := decode[receivingList](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			receivingSummaryBand(page),
			receivingActionCluster(),
			receivingTable(page.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			receivingForm(payload),
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

func receivingSummaryBand(page receivingList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Receiving shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat receiving as the closeout board for inbound work: visible discrepancy posture, owning source, and direct movement into reconciliation and inventory verification.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Sessions", fmt.Sprintf("%d active", len(page.Items))),
			statCard("Open", fmt.Sprintf("%d live", countReceivingStatus(page.Items, "open"))),
			statCard("Closed", fmt.Sprintf("%d closed", countReceivingStatus(page.Items, "closed"))),
			statCard("Discrepancies", fmt.Sprintf("%d flagged", countReceivingWithDiscrepancy(page.Items))),
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

func receivingTable(items []receivingRecord) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, receivingTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No receiving sessions are available in the current Atlas workspace.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Receiving table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review source, warehouse, discrepancy posture, and closeout status in one dense table before opening the receiving detail route.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d sessions", len(items)))),
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func receivingTableRow(item receivingRecord) ui.Node {
	source := item.SourceType + " " + item.SourceID
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/receiving/" + item.ID, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.ID)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.CreatedAt)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(item.WarehouseID)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(source)),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(fallback(item.DiscrepancySummary, "No discrepancy recorded"))),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.A(html.Props{Href: "/app/receiving/" + item.ID, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open session"))),
	)
}

func countReceivingStatus(items []receivingRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
			count++
		}
	}
	return count
}

func countReceivingWithDiscrepancy(items []receivingRecord) int {
	count := 0
	for _, item := range items {
		if strings.TrimSpace(item.DiscrepancySummary) != "" {
			count++
		}
	}
	return count
}

func receivingDetailContent(payload Payload) ui.Node {
	page := decode[receivingDetailPage](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			receivingDetailHero(page),
			receivingLineTable(page.Lines),
		),
		receivingDetailRail(payload, page),
	)
}

func receivingDetailHero(page receivingDetailPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Receiving workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(page.Session.ID)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fallback(page.Session.DiscrepancySummary, "Receiving session is ready for closeout."))),
				),
			),
			html.A(html.Props{Href: "/app/receiving", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to receiving")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Session.WarehouseID)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Session.SourceType+" "+page.Session.SourceID)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(strings.ReplaceAll(page.Session.Status, "_", " "))),
		),
	)
}

func receivingLineTable(lines []receivingLineRecord) ui.Node {
	rows := make([]ui.Node, 0, len(lines))
	for _, line := range lines {
		reason := fallback(line.DiscrepancyReason, "matched")
		rows = append(rows, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(line.ProductSKU)),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", line.ExpectedQuantity))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", line.ActualQuantity))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"}, html.Text(reason)),
		))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func atlasLazySection(loader func() ui.Node, fallback ui.Node, deps ...interface{}) ui.Node {
	return ui.CreateElement(func() ui.Node {
		handle := ui.UseLazyNode(func(context.Context) (ui.Node, error) {
			return loader(), nil
		}, deps...)
		state := handle.Get()
		return ui.AsyncBoundary(ui.AsyncBoundaryProps{
			Pending:  state.Loading,
			Error:    state.Error,
			Fallback: fallback,
			ErrorFallback: func(err error) ui.Node {
				return fallback
			},
			Content: state.Node,
			Delay:   120 * time.Millisecond,
		})
	})
}

func internalLazyRailFallback(title, copy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"},
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(title)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
	)
}

func useAtlasFocusContainment(active bool, containerSelector, initialFocusSelector string) {
	manager := ui.UseFocusManager()
	armed := ui.UseRef(false)
	ui.UseEffect(func() func() {
		if !active {
			armed.Set(false)
			return nil
		}
		manager.RememberActive()
		armed.Set(true)
		return func() {
			if armed.Get() {
				manager.Restore()
				armed.Set(false)
			}
		}
	}, active, containerSelector, initialFocusSelector)
	ui.UseFocusTrap(ui.FocusTrapOptions{
		Active:                active,
		ContainerSelector:     containerSelector,
		InitialFocusSelector:  initialFocusSelector,
		FallbackFocusSelector: containerSelector,
	})
}

func atlasOverlayTarget() ui.PortalTarget {
	return ui.PortalTarget{Selector: "#" + atlasOverlayRootID}
}

func atlasDialogOverlay(open bool, modalID, labelledBy, describedBy, initialFocusSelector string, onDismiss func(), child ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  open,
		SurfaceID:             modalID,
		Kind:                  ui.OverlayKindDialog,
		LabelledBy:            labelledBy,
		DescribedBy:           describedBy,
		InitialFocusSelector:  initialFocusSelector,
		FallbackFocusSelector: "#" + modalID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         onDismiss != nil,
		CloseOnOutsideClick:   onDismiss != nil,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         "fixed inset-0 flex items-center justify-center bg-slate-950/75 p-4",
		SurfaceClass:          "w-[min(92vw,34rem)] grid gap-4 border border-slate-700 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]",
		OnDismiss:             onDismiss,
		Child:                 child,
	})
}

func atlasRouteSheetOverlay(surfaceID, labelledBy, describedBy, initialFocusSelector string, child ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  true,
		Target:                atlasOverlayTarget(),
		AppRootSelector:       "#" + atlasShellRootID,
		SurfaceID:             surfaceID,
		Kind:                  ui.OverlayKindSheet,
		LabelledBy:            labelledBy,
		DescribedBy:           describedBy,
		InitialFocusSelector:  initialFocusSelector,
		FallbackFocusSelector: "#" + surfaceID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         false,
		CloseOnOutsideClick:   false,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         "fixed inset-0 flex items-stretch justify-end bg-slate-950/72 p-4",
		SurfaceClass:          "grid h-full w-[min(92vw,36rem)] gap-4 overflow-y-auto rounded-sm border border-cyan-400/35 bg-[linear-gradient(180deg,rgba(11,18,32,0.98),rgba(2,6,23,1))] p-4 shadow-[0_18px_48px_rgba(6,182,212,0.08)]",
		Child:                 child,
	})
}

func atlasDismissibleSheet(open bool, surfaceID, labelledBy, describedBy, initialFocusSelector string, onDismiss func(), child ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  open,
		Target:                atlasOverlayTarget(),
		AppRootSelector:       "#" + atlasShellRootID,
		SurfaceID:             surfaceID,
		Kind:                  ui.OverlayKindSheet,
		LabelledBy:            labelledBy,
		DescribedBy:           describedBy,
		InitialFocusSelector:  initialFocusSelector,
		FallbackFocusSelector: "#" + surfaceID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         onDismiss != nil,
		CloseOnOutsideClick:   onDismiss != nil,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         "fixed inset-0 flex items-stretch justify-end bg-slate-950/72 p-4",
		SurfaceClass:          "grid h-full w-[min(92vw,34rem)] gap-4 overflow-y-auto rounded-[1.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]",
		OnDismiss:             onDismiss,
		Child:                 child,
	})
}

func atlasConfirmationDialog(open bool, modalID, title, copy, confirmLabel string, onDismiss func(), body ...ui.Node) ui.Node {
	if !open {
		return nil
	}
	titleID := modalID + "-title"
	descriptionID := modalID + "-description"
	dismissHandler := ui.UseEvent(func() {
		if onDismiss != nil {
			onDismiss()
		}
	})
	children := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Confirmation")),
			html.P(html.Props{ID: titleID, Class: "text-lg font-semibold text-white"}, html.Text(title)),
			html.P(html.Props{ID: descriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text(copy)),
		),
	}
	children = append(children, body...)
	children = append(children,
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-end gap-3"},
			html.Button(html.Props{Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: dismissHandler}, html.Text("Cancel")),
			html.Button(html.Props{ID: modalID + "-confirm", Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text(confirmLabel)),
		),
	)
	return atlasDialogOverlay(open, modalID, titleID, descriptionID, "#"+modalID+"-confirm", onDismiss, html.Div(html.Props{Class: "grid gap-4"}, children...))
}

func atlasSidePanelErrorBoundary(child ui.Node, title string, fallback ui.Node, resetKeys ...interface{}) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		Child:     child,
		ResetKeys: resetKeys,
		ErrorFallback: func(err error, reset func()) ui.Node {
			return detailRailErrorIsland(title, err, reset, fallback)
		},
	})
}

func purchaseOrderDetailRail(payload Payload, page purchaseOrderDetailPage) ui.Node {
	fallbackNode := internalLazyRailFallback("Loading purchase-order rail", "Atlas is preparing the secondary vendor panel after the primary route body stabilizes.")
	return atlasSidePanelErrorBoundary(atlasLazySection(func() ui.Node {
		return html.Div(html.Props{Class: "grid gap-5"},
			routeRevalidationCard("Purchase-order route refresh", "Re-run the current purchase-order loader after an approval or hold action so the latest vendor posture and inbound timing stay authoritative on this screen."),
			purchaseOrderDetailStatsIsland(payload, page),
			purchaseOrderStatusForm(page.Order.ID, page.Order.Status, payload),
		)
	}, fallbackNode, page.Order.ID, page.Order.Status, page.Order.ETA), "Purchase-order side panel", fallbackNode, page.Order.ID, page.Order.Status, page.Order.ETA)
}

func receivingDetailRail(payload Payload, page receivingDetailPage) ui.Node {
	fallbackNode := internalLazyRailFallback("Loading receiving rail", "Atlas is preparing the secondary receiving panel after the primary route body stabilizes.")
	return atlasSidePanelErrorBoundary(atlasLazySection(func() ui.Node {
		return html.Div(html.Props{Class: "grid gap-5"},
			routeRevalidationCard("Receiving route refresh", "Re-run the receiving loader after reconcile or classification work so discrepancy status and closeout readiness refresh in place."),
			receivingDetailStatsIsland(payload, page),
			receivingFormForID(page.Session.ID, payload),
		)
	}, fallbackNode, page.Session.ID, page.Session.Status, page.Session.DiscrepancySummary), "Receiving side panel", fallbackNode, page.Session.ID, page.Session.Status, page.Session.DiscrepancySummary)
}

func purchaseOrderDetailStatsIsland(payload Payload, page purchaseOrderDetailPage) ui.Node {
	request, ok := StartupRequest(payload, "page")
	requestURL := strings.TrimSpace(request.URL)
	fallbackNode := purchaseOrderDetailStatsContent(page, false, false, "", ui.Handler{})
	if !ok || requestURL == "" {
		return fallbackNode
	}
	return ui.CreateElement(func() ui.Node {
		panelState := useAtlasState(page)
		refreshingState := useAtlasState(false)
		refreshErrorState := useAtlasState("")
		resource := useAtlasStartupPageResource(payload)
		resourceState := resource.Get()
		cachedPage := purchaseOrderDetailPage{}
		if resourceState.Ready {
			cachedPage = decode[purchaseOrderDetailPage](resourceState.Value)
		}
		useAtlasEffect(func() func() {
			if resourceState.Ready && strings.TrimSpace(cachedPage.Order.ID) != "" {
				panelState.Set(cachedPage)
			}
			return nil
		}, resourceState.Ready, cachedPage.Order.ID, cachedPage.Order.Status, cachedPage.Order.ETA, cachedPage.Order.WarehouseName)
		refreshPanel := ui.UseEvent(func() {
			if refreshingState.Get() {
				return
			}
			refreshingState.Set(true)
			refreshErrorState.Set("")
			go func() {
				result := <-atlasFetch(requestURL, atlasFetchOptions{
					Method: "GET",
					Headers: map[string]interface{}{
						"Accept": "application/json",
					},
				})
				if strings.TrimSpace(result.Error) != "" {
					refreshErrorState.Set(result.Error)
					refreshingState.Set(false)
					return
				}
				decoded := decodeJSONText[purchaseOrderDetailPage](result.Data)
				if strings.TrimSpace(decoded.Order.ID) == "" {
					refreshErrorState.Set("Atlas returned an empty purchase-order panel payload.")
					refreshingState.Set(false)
					return
				}
				panelState.Set(decoded)
				resource.Set(decoded)
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Purchase-order panel refreshed",
					Detail: "The vendor-side status panel picked up the latest order snapshot without rerendering the whole route.",
					Tone:   "success",
				})
				refreshingState.Set(false)
			}()
		})
		current := panelState.Get()
		return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
			Pending:  !resourceState.Ready && resourceState.Error == nil,
			Error:    resourceState.Error,
			Fallback: fallbackNode,
			ErrorFallback: func(err error) ui.Node {
				return detailRailErrorIsland("Purchase-order side panel", err, resource.Reload, fallbackNode)
			},
			Content: purchaseOrderDetailStatsContent(current, resourceState.Loading && resourceState.Ready, refreshingState.Get(), refreshErrorState.Get(), refreshPanel),
		})
	})
}

func purchaseOrderDetailStatsContent(page purchaseOrderDetailPage, refreshing bool, fetchRefreshing bool, refreshError string, refetch ui.Handler) ui.Node {
	children := []ui.Node{}
	label := "Refresh panel snapshot"
	if fetchRefreshing {
		label = "Refreshing panel..."
	}
	children = append(children, html.Button(html.Props{
		Type:     "button",
		Class:    "inline-flex items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-300/15 hover:text-white disabled:cursor-not-allowed disabled:border-slate-700 disabled:bg-slate-900 disabled:text-slate-500",
		Disabled: fetchRefreshing,
		OnClick:  refetch,
	}, html.Text(label)))
	if refreshing || fetchRefreshing {
		children = append(children, html.P(html.Props{Class: "rounded-[1.35rem] border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm leading-6 text-cyan-100"}, html.Text("Refreshing the purchase-order side panel while the current snapshot stays visible.")))
	}
	if strings.TrimSpace(refreshError) != "" {
		children = append(children, html.P(html.Props{Class: "rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 text-sm leading-6 text-rose-100"}, html.Text(refreshError)))
	}
	children = append(children,
		statCard("Warehouse", page.Order.WarehouseName),
		statCard("ETA", page.Order.ETA),
	)
	return html.Div(html.Props{Class: "grid gap-5"}, children...)
}

func receivingDetailStatsIsland(payload Payload, page receivingDetailPage) ui.Node {
	request, ok := StartupRequest(payload, "page")
	requestURL := strings.TrimSpace(request.URL)
	fallbackNode := receivingDetailStatsContent(page, false, false, "", ui.Handler{})
	if !ok || requestURL == "" {
		return fallbackNode
	}
	return ui.CreateElement(func() ui.Node {
		panelState := useAtlasState(page)
		refreshingState := useAtlasState(false)
		refreshErrorState := useAtlasState("")
		resource := useAtlasStartupPageResource(payload)
		resourceState := resource.Get()
		cachedPage := receivingDetailPage{}
		if resourceState.Ready {
			cachedPage = decode[receivingDetailPage](resourceState.Value)
		}
		useAtlasEffect(func() func() {
			if resourceState.Ready && strings.TrimSpace(cachedPage.Session.ID) != "" {
				panelState.Set(cachedPage)
			}
			return nil
		}, resourceState.Ready, cachedPage.Session.ID, cachedPage.Session.Status, cachedPage.Session.WarehouseID)
		refreshPanel := ui.UseEvent(func() {
			if refreshingState.Get() {
				return
			}
			refreshingState.Set(true)
			refreshErrorState.Set("")
			go func() {
				result := <-atlasFetch(requestURL, atlasFetchOptions{
					Method: "GET",
					Headers: map[string]interface{}{
						"Accept": "application/json",
					},
				})
				if strings.TrimSpace(result.Error) != "" {
					refreshErrorState.Set(result.Error)
					refreshingState.Set(false)
					return
				}
				decoded := decodeJSONText[receivingDetailPage](result.Data)
				if strings.TrimSpace(decoded.Session.ID) == "" {
					refreshErrorState.Set("Atlas returned an empty receiving panel payload.")
					refreshingState.Set(false)
					return
				}
				panelState.Set(decoded)
				resource.Set(decoded)
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Receiving panel refreshed",
					Detail: "The receiving side panel picked up the latest session snapshot without rerendering the whole route.",
					Tone:   "success",
				})
				refreshingState.Set(false)
			}()
		})
		current := panelState.Get()
		return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
			Pending:  !resourceState.Ready && resourceState.Error == nil,
			Error:    resourceState.Error,
			Fallback: fallbackNode,
			ErrorFallback: func(err error) ui.Node {
				return detailRailErrorIsland("Receiving side panel", err, resource.Reload, fallbackNode)
			},
			Content: receivingDetailStatsContent(current, resourceState.Loading && resourceState.Ready, refreshingState.Get(), refreshErrorState.Get(), refreshPanel),
		})
	})
}

func receivingDetailStatsContent(page receivingDetailPage, refreshing bool, fetchRefreshing bool, refreshError string, refetch ui.Handler) ui.Node {
	children := []ui.Node{}
	label := "Refresh panel snapshot"
	if fetchRefreshing {
		label = "Refreshing panel..."
	}
	children = append(children, html.Button(html.Props{
		Type:     "button",
		Class:    "inline-flex items-center justify-center rounded-full border border-cyan-300/40 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:border-cyan-200 hover:bg-cyan-300/15 hover:text-white disabled:cursor-not-allowed disabled:border-slate-700 disabled:bg-slate-900 disabled:text-slate-500",
		Disabled: fetchRefreshing,
		OnClick:  refetch,
	}, html.Text(label)))
	if refreshing || fetchRefreshing {
		children = append(children, html.P(html.Props{Class: "rounded-[1.35rem] border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm leading-6 text-cyan-100"}, html.Text("Refreshing the receiving side panel while the current snapshot stays visible.")))
	}
	if strings.TrimSpace(refreshError) != "" {
		children = append(children, html.P(html.Props{Class: "rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 text-sm leading-6 text-rose-100"}, html.Text(refreshError)))
	}
	children = append(children,
		statCard("Warehouse", page.Session.WarehouseID),
		statCard("Status", page.Session.Status),
	)
	return html.Div(html.Props{Class: "grid gap-5"}, children...)
}

func detailRailErrorIsland(title string, err error, retry func(), fallbackNode ui.Node) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-rose-200"}, html.Text(title)),
			html.P(html.Props{Class: "text-sm leading-6 text-rose-100"}, html.Text(err.Error())),
			html.Button(html.Props{
				Type:    "button",
				Class:   "inline-flex items-center justify-center rounded-full border border-rose-200/40 bg-rose-200/10 px-4 py-3 text-sm font-semibold text-white transition hover:bg-rose-200/20",
				OnClick: ui.UseEvent(func() { retry() }),
			}, html.Text("Retry panel")),
		),
		fallbackNode,
	)
}

func commentsContent(payload Payload) ui.Node {
	page := decode[commentList](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			commentsSummaryBand(page),
			commentsActionCluster(),
			commentsTable(page.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryRailCard("Buyer inbox handoffs", "Move from public feedback into the right internal route without losing the original question context.",
				html.Div(html.Props{Class: "grid gap-3"},
					inventoryActionCard("Update marketing copy", "Route unclear product questions into the merch workspace when the issue is messaging, not stock.", "/app/products"),
					inventoryActionCard("Check inventory promise", "Open the inventory workspace when the customer is really asking about supply or timing.", "/app/inventory"),
					inventoryActionCard("Open warehouse ops", "Use warehouse-native routes when the answer depends on a specific hub or recovery lane.", "/app/warehouses"),
				),
			),
			moderationForm(page.Items, payload),
			bulkModerationForm(page.Items, payload),
		),
	)
}

func settingsContent(payload Payload) ui.Node {
	page := decode[settingsPage](pageData(payload))
	presentation := currentShellPresentationState(payload)
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(22rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			settingsSummaryBand(page, presentation),
			settingsActionCluster(),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			preferenceForm(payload),
			savedViewBrowserCard(payload),
			savedViewTransferCard(payload),
			operatorWorkspaceSnapshotCard(payload),
		),
	)
}

func commentsSummaryBand(page commentList) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Buyer inbox shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat buyer comments as an operator inbox: visible moderation posture, product context, and direct handoff into merchandising or supply routes when the question reveals a broader issue.")),
		),
		routeSummaryStrip(page.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Comments", fmt.Sprintf("%d open", len(page.Items))),
			statCard("Pending", fmt.Sprintf("%d queued", countCommentStatus(page.Items, "pending"))),
			statCard("Approved", fmt.Sprintf("%d live", countCommentStatus(page.Items, "approved"))),
			statCard("Flagged", fmt.Sprintf("%d escalated", countCommentStatus(page.Items, "flagged"))),
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

func commentsTable(items []commentRecord) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, commentTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 5}}, html.Text("No buyer questions are waiting in the Atlas inbox.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Buyer question table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review subject, product context, author, and moderation posture in one dense table before opening a workflow on the right rail.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d comments", len(items)))),
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
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func commentTableRow(item commentRecord) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.Subject)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Body)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(item.ProductSKU)),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(item.AuthorName+" | "+strings.ReplaceAll(item.AuthorType, "_", " "))),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(item.UpdatedAt)),
	)
}

func countCommentStatus(items []commentRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
			count++
		}
	}
	return count
}

func settingsSummaryBand(page settingsPage, presentation shellPresentationState) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Settings shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Treat settings as the operator control room for shell presentation, saved-view exchange, and workspace snapshot handoff rather than a narrow preference form.")),
		),
		routeSummaryStrip(page.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-5"},
			statCard("Theme", fallback(presentation.Theme, "dark")),
			statCard("Locale", fallback(presentation.Locale, "en")),
			statCard("Density", fallback(presentation.Density, "compact")),
			statCard("Warehouse", fallback(presentation.DefaultWarehouse, "new-jersey-hub")),
			statCard("Diagnostics", map[bool]string{true: "on", false: "off"}[presentation.Diagnostics]),
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

func savedViewBrowserCard(payload Payload) ui.Node {
	if len(payload.SavedViews) == 0 {
		return listCard("Saved views", html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No records yet.")))
	}
	items := make([]ui.CompositeItem, 0, len(payload.SavedViews))
	for index, saved := range payload.SavedViews {
		items = append(items, ui.CompositeItem{
			ID:   fmt.Sprintf("atlas-saved-view-%d", index),
			Text: strings.TrimSpace(saved.Name + " " + saved.Scope + " " + saved.SortKey + " " + saved.SortDirection),
		})
	}
	return ui.CreateElement(func() ui.Node {
		nav := ui.UseCompositeNavigation(items, ui.CompositeNavigationOptions{Orientation: "vertical", Loop: true})
		listboxKeyDown := ui.UseEvent(func(event ui.KeyboardEvent) {
			nav.OnKeyDown(event)
		})
		activeIndex := nav.ActiveIndex()
		if activeIndex < 0 || activeIndex >= len(payload.SavedViews) {
			activeIndex = 0
		}
		active := payload.SavedViews[activeIndex]
		options := make([]ui.Node, 0, len(payload.SavedViews))
		for index, saved := range payload.SavedViews {
			currentIndex := index
			currentItem := items[index]
			className := "grid gap-2 rounded-[1.2rem] border px-4 py-3 text-left text-sm transition"
			if nav.IsActive(index) {
				className += " border-cyan-300/35 bg-cyan-300/10 text-cyan-50"
			} else {
				className += " border-white/10 bg-slate-950/40 text-slate-200"
			}
			options = append(options, html.Div(html.Props{
				ID:    currentItem.ID,
				Role:  "option",
				Class: className,
				OnClick: ui.UseEvent(func() {
					nav.SetActive(currentIndex)
				}),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[nav.IsActive(index)],
				},
			},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(saved.Name)),
				html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(saved.Scope+" | "+saved.SortKey+"/"+saved.SortDirection)),
			))
		}
		filterEntries := make([]string, 0, len(active.Filters))
		for key, value := range active.Filters {
			filterEntries = append(filterEntries, key+": "+value)
		}
		sort.Strings(filterEntries)
		filterNodes := make([]ui.Node, 0, len(filterEntries))
		for _, entry := range filterEntries {
			filterNodes = append(filterNodes, html.P(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs uppercase tracking-[0.18em] text-slate-300"}, html.Text(entry)))
		}
		children := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Saved views")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Focus the list and use ArrowUp, ArrowDown, Home, End, or typeahead to inspect Atlas workspace presets without leaving the settings route.")),
			html.Div(html.Props{
				Role:      "listbox",
				Class:     "grid gap-3 rounded-[1.35rem] border border-white/10 bg-slate-950/55 p-4",
				OnKeyDown: listboxKeyDown,
				Aria: map[string]string{
					"activedescendant": nav.ActiveDescendant(),
				},
				Raw: map[string]interface{}{
					"tabIndex": 0,
				},
			}, options...),
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-white/5 p-4"},
				html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
					statCard("Scope", active.Scope),
					statCard("Sort key", active.SortKey),
					statCard("Direction", active.SortDirection),
				),
			),
		}
		if len(filterNodes) > 0 {
			children = append(children, html.Div(html.Props{Class: "flex flex-wrap gap-2"}, filterNodes...))
		}
		return html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, children...)
	})
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

func routeSummaryStrip(summary pageSummary) ui.Node {
	if len(summary.Items) == 0 {
		return html.Div(html.Props{})
	}
	nodes := make([]ui.Node, 0, len(summary.Items))
	for _, item := range summary.Items {
		nodes = append(nodes, html.Div(html.Props{Class: "grid gap-2 " + internalInsetSurfaceClass() + " p-4"},
			html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-cyan-300"}, html.Text(item.Label)),
			html.P(html.Props{Class: "text-xl font-semibold text-white"}, html.Text(item.Value)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Detail)),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4"},
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text(fallback(summary.Headline, "Route summary"))),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2 xl:grid-cols-4"}, nodes...),
	)
}

func featureCard(title, copy string) ui.Node {
	return html.Div(html.Props{Class: internalSurfaceCardClass() + " p-5"},
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
	return html.Div(html.Props{Class: internalSurfaceCardClass() + " p-5"},
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
	return html.Div(html.Props{Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"}, content...)
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

func normalizeReceivingResolutionWorkflowState(state receivingResolutionWorkflowState) receivingResolutionWorkflowState {
	status := strings.TrimSpace(strings.ToLower(state.Status))
	summary := strings.TrimSpace(state.DiscrepancySummary)
	switch {
	case status == "closed" && summary == "":
		state.Stage = "closeout-needs-note"
	case status == "closed":
		state.Stage = "closeout-ready"
	case status == "review":
		state.Stage = "discrepancy-review"
	default:
		state.Stage = "classification-open"
	}
	return state
}

func reduceReceivingResolutionWorkflowState(state receivingResolutionWorkflowState, action receivingResolutionWorkflowAction) receivingResolutionWorkflowState {
	switch action.Field {
	case "status":
		state.Status = action.Value
	case "discrepancy_summary":
		state.DiscrepancySummary = action.Value
	}
	state.LastEditedField = action.Field
	return normalizeReceivingResolutionWorkflowState(state)
}

func preferenceForm(payload Payload) ui.Node {
	presentation := currentShellPresentationState(payload)
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(preferenceFormState{
			Theme:              fallback(presentation.Theme, "dark"),
			Locale:             fallback(presentation.Locale, "en"),
			Density:            fallback(presentation.Density, "compact"),
			DefaultWarehouseID: fallback(presentation.DefaultWarehouse, "new-jersey-hub"),
		})
		transition := useAtlasTransition()
		value := form.Get()
		children := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Operator preferences")),
			boundInputWithValue("theme", "Theme", value.Theme, "Theme", form),
			boundInputWithValue("locale", "Locale", value.Locale, "Locale", form),
			boundTransitionSelectWithValue("density", "Density", value.Density, "Density", []optionItem{{"compact", "Compact"}, {"comfortable", "Comfortable"}}, form, transition),
			preferenceDensityPreviewCard(value.Density, transition.Pending()),
			boundInputWithValue("default_warehouse_id", "Default warehouse", value.DefaultWarehouseID, "DefaultWarehouseID", form),
			submitButton("Save preferences"),
		}
		return html.Form(html.Props{Action: "/api/app/preferences", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
	})
}

func moderationForm(items []commentRecord, payload Payload) ui.Node {
	id := ""
	if len(items) > 0 {
		id = items[0].ID
	}
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(moderationActionFormState{Status: "approved", Reason: "Reviewed by Atlas operations."})
		confirmOpen := ui.UseState(false)
		value := form.Get()
		openConfirm := ui.UseEvent(func() { confirmOpen.Set(true) })
		closeConfirm := func() { confirmOpen.Set(false) }
		children := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Review first queued buyer question")),
			boundInputWithValue("status", "Status", value.Status, "Status", form),
			boundTextareaWithValue("reason", "Reason", value.Reason, "Reason", form),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: openConfirm}, html.Text("Review decision")),
			atlasConfirmationDialog(confirmOpen.Get(), "atlas-comment-review-confirm", "Confirm buyer review", "Atlas keeps keyboard focus inside the moderation confirmation step until you either cancel or apply the review.", "Apply review", closeConfirm,
				statCard("Status", value.Status),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(value.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/comments/" + id + "/moderate", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
	})
}

func bulkModerationForm(items []commentRecord, payload Payload) ui.Node {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(bulkModerationActionFormState{
			IDs:    strings.Join(ids, ","),
			Status: "approved",
			Reason: "Bulk-reviewed from the visible Atlas buyer inbox queue.",
		})
		confirmOpen := ui.UseState(false)
		value := form.Get()
		openConfirm := ui.UseEvent(func() { confirmOpen.Set(true) })
		closeConfirm := func() { confirmOpen.Set(false) }
		children := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Bulk review visible queue")),
			html.Input(html.Props{Type: "hidden", Name: "ids", Value: value.IDs}),
			boundInputWithValue("status", "Status", value.Status, "Status", form),
			boundTextareaWithValue("reason", "Reason", value.Reason, "Reason", form),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: openConfirm}, html.Text("Review bulk action")),
			atlasConfirmationDialog(confirmOpen.Get(), "atlas-bulk-review-confirm", "Confirm bulk moderation", "The buyer-inbox bulk action now traps focus inside its confirmation step instead of leaving focus scattered behind the modal.", "Apply bulk review", closeConfirm,
				statCard("Items", fmt.Sprintf("%d", len(items))),
				statCard("Status", value.Status),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(value.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/comments/bulk-moderate", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
	})
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

func savedViewTransferCard(payload Payload) ui.Node {
	presentation := currentShellPresentationState(payload)
	type savedViewTransferItem struct {
		Name          string `json:"name"`
		Scope         string `json:"scope"`
		SortKey       string `json:"sortKey"`
		SortDirection string `json:"sortDirection"`
		Density       string `json:"density"`
		WarehouseID   string `json:"warehouseId"`
		FiltersJSON   string `json:"filtersJSON"`
	}
	items := make([]savedViewTransferItem, 0, len(payload.SavedViews))
	for _, saved := range payload.SavedViews {
		filtersJSON, err := json.Marshal(saved.Filters)
		if err != nil {
			filtersJSON = []byte(`{}`)
		}
		items = append(items, savedViewTransferItem{
			Name:          saved.Name,
			Scope:         saved.Scope,
			SortKey:       saved.SortKey,
			SortDirection: saved.SortDirection,
			Density:       fallback(presentation.Density, "compact"),
			WarehouseID:   fallback(saved.Filters["warehouse"], presentation.DefaultWarehouse),
			FiltersJSON:   string(filtersJSON),
		})
	}
	exportPayload := map[string]any{"items": items}
	encoded, err := json.MarshalIndent(exportPayload, "", "  ")
	if err != nil {
		encoded = []byte(`{"items":[]}`)
	}
	defaultImportPayload := string(encoded)
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(savedViewImportFormState{ViewsJSON: defaultImportPayload})
		validator := useAtlasWorkerTask[savedViewImportValidationRequest, savedViewImportValidationProgress, savedViewImportValidationResult](interop.WorkerOptions{
			URL:   "/assets/script/atlas-saved-view-import-worker.js",
			Ready: true,
			Name:  "atlas-saved-view-import",
		}, "validate-saved-view-import")
		useAtlasEffect(func() func() {
			trimmed := strings.TrimSpace(form.Get().ViewsJSON)
			if trimmed == "" {
				validator.Cancel()
				return nil
			}
			validator.Start(savedViewImportValidationRequest{Text: form.Get().ViewsJSON})
			return nil
		}, form.Get().ViewsJSON)
		validationState := validator.Get()
		value := form.Get()
		children := []ui.Node{
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Export saved views")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Download or copy the current saved-view payload so Atlas presets can move between runs without rebuilding them by hand.")),
				html.A(html.Props{Href: "/api/app/saved-views/export", Class: "inline-flex w-fit rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950"}, html.Text("Open export payload")),
				html.Pre(html.Props{Class: "overflow-x-auto rounded-[1.2rem] border border-white/10 bg-slate-950/70 p-4 text-xs leading-6 text-slate-300"}, html.Text(defaultImportPayload)),
			),
		}
		importChildren := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Import saved views")),
			boundTextareaWithValue("views_json", "Saved-view payload", value.ViewsJSON, "ViewsJSON", form),
			savedViewImportValidationCard(validationState),
			submitButton("Import saved views"),
		}
		children = append(children, html.Form(html.Props{Action: "/api/app/saved-views/import", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, importChildren...)...))
		return html.Div(html.Props{Class: "grid gap-5"}, children...)
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

func savedViewImportValidationCard(state atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]) ui.Node {
	if state.Error != nil {
		return html.Div(html.Props{Class: "grid gap-2 rounded-[1.35rem] border border-rose-400/25 bg-rose-400/10 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-rose-200"}, html.Text("Worker validation failed")),
			html.P(html.Props{Class: "text-sm leading-6 text-rose-100"}, html.Text(state.Error.Error())),
		)
	}
	if state.Running {
		stage := "validating saved-view payload"
		progress := "Worker running"
		if state.ProgressReady {
			stage = fallback(state.Progress.Stage, stage)
			progress = fmt.Sprintf("%d%% complete", state.Progress.Percent)
		}
		return html.Div(html.Props{Class: "grid gap-2 rounded-[1.35rem] border border-cyan-300/30 bg-cyan-300/10 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-cyan-100"}, html.Text("Worker validation in progress")),
			html.P(html.Props{Class: "text-sm leading-6 text-cyan-50"}, html.Text(progress+" · "+stage)),
		)
	}
	if state.Ready {
		summary := "Payload is ready to import."
		if !state.Value.Valid {
			summary = "Payload needs fixes before import."
		}
		preview := "No saved-view names detected yet."
		if len(state.Value.Preview) > 0 {
			preview = strings.Join(state.Value.Preview, " · ")
		}
		scopeSummary := "No scopes detected."
		if len(state.Value.Scopes) > 0 {
			scopeSummary = strings.Join(state.Value.Scopes, ", ")
		}
		children := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Worker import preview")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(summary)),
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
				statCard("Items", fmt.Sprintf("%d", state.Value.ItemCount)),
				statCard("Scopes", fmt.Sprintf("%d", state.Value.ScopeCount)),
				statCard("Invalid", fmt.Sprintf("%d", state.Value.InvalidCount)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Preview: "+preview)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scopes: "+scopeSummary)),
		}
		if len(state.Value.Warnings) > 0 {
			warnings := make([]ui.Node, 0, len(state.Value.Warnings))
			for _, warning := range state.Value.Warnings {
				warnings = append(warnings, html.P(html.Props{Class: "rounded-[1.1rem] border border-amber-400/25 bg-amber-400/10 px-3 py-2 text-sm text-amber-100"}, html.Text(warning)))
			}
			children = append(children, html.Div(html.Props{Class: "grid gap-2"}, warnings...))
		}
		return html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-slate-950/40 p-4"}, children...)
	}
	return html.Div(html.Props{Class: "grid gap-2 rounded-[1.35rem] border border-white/10 bg-slate-950/40 p-4"},
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.22em] text-slate-300"}, html.Text("Worker import preview")),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text("Atlas will validate the saved-view import payload in a dedicated worker as you edit it.")),
	)
}

func operatorWorkspaceSnapshotCard(payload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(workspaceSnapshotFormState{WorkspaceSnapshotJSON: operatorWorkspaceSnapshotJSON(payload)})
		value := form.Get()
		return html.Div(html.Props{Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Export workspace snapshot")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Copy the current operator shell, route workspace, and saved-view snapshot when a reviewer or another operator needs the same Atlas context without opening diagnostics mode.")),
			boundTextareaWithValue("workspace_snapshot_json", "Workspace snapshot", value.WorkspaceSnapshotJSON, "WorkspaceSnapshotJSON", form),
		)
	})
}

func operatorWorkspaceSnapshotJSON(payload Payload) string {
	type savedViewSnapshotItem struct {
		Name          string            `json:"name"`
		Scope         string            `json:"scope"`
		SortKey       string            `json:"sortKey"`
		SortDirection string            `json:"sortDirection"`
		Density       string            `json:"density"`
		WarehouseID   string            `json:"warehouseId"`
		Filters       map[string]string `json:"filters"`
	}
	presentation := currentShellPresentationState(payload)
	workspace := currentRouteWorkspaceState(payload)
	savedViews := make([]savedViewSnapshotItem, 0, len(payload.SavedViews))
	for _, saved := range payload.SavedViews {
		savedViews = append(savedViews, savedViewSnapshotItem{
			Name:          saved.Name,
			Scope:         saved.Scope,
			SortKey:       saved.SortKey,
			SortDirection: saved.SortDirection,
			Density:       presentation.Density,
			WarehouseID:   fallback(saved.Filters["warehouse"], presentation.DefaultWarehouse),
			Filters:       saved.Filters,
		})
	}
	exportPayload := map[string]any{
		"route": map[string]any{
			"path":   payload.Route.Path,
			"screen": payload.Route.Screen,
			"query":  payload.Route.Query,
		},
		"shellPresentation": presentation,
		"routeWorkspace":    workspace,
		"savedViews":        savedViews,
	}
	encoded, err := json.MarshalIndent(exportPayload, "", "  ")
	if err != nil {
		return `{"error":"workspace snapshot encode failed"}`
	}
	return string(encoded)
}

func transferForm(payload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(transferFormState{
			SourceWarehouseID:      "nevada-hub",
			DestinationWarehouseID: "new-jersey-hub",
			Reason:                 "Support east-coast promise windows",
			RecommendedBy:          "Atlas Hydration Demo",
		})
		confirmOpen := ui.UseState(false)
		value := form.Get()
		openConfirm := ui.UseEvent(func() { confirmOpen.Set(true) })
		closeConfirm := func() { confirmOpen.Set(false) }
		children := []ui.Node{
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Create transfer")),
			boundInputWithValue("source_warehouse_id", "Source warehouse", value.SourceWarehouseID, "SourceWarehouseID", form),
			boundInputWithValue("destination_warehouse_id", "Destination warehouse", value.DestinationWarehouseID, "DestinationWarehouseID", form),
			boundInputWithValue("reason", "Reason", value.Reason, "Reason", form),
			boundInputWithValue("recommended_by", "Recommended by", value.RecommendedBy, "RecommendedBy", form),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: openConfirm}, html.Text("Review transfer")),
			atlasConfirmationDialog(confirmOpen.Get(), "atlas-transfer-confirm", "Confirm transfer plan", "Review the transfer before Atlas posts it so focus stays inside the confirmation step until you dismiss or submit.", "Create transfer", closeConfirm,
				statCard("From", value.SourceWarehouseID),
				statCard("To", value.DestinationWarehouseID),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(value.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/transfers", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
	})
}

func thresholdForm(item inventoryRow, payload Payload) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Adjust thresholds")),
		inputWithValue("warehouse_id", "Warehouse", item.WarehouseID),
		inputWithValue("reorder_point", "Reorder point", "18"),
		inputWithValue("safety_stock", "Safety stock", "9"),
		submitButton("Save threshold"),
	}
	return html.Form(html.Props{Action: "/api/app/inventory/" + item.SKU + "/threshold", Method: "post", Class: "grid gap-3 " + internalSurfaceCardClass() + " p-5"}, prependCSRFToken(payload.CSRF, children...)...)
}

func receivingForm(payload Payload) ui.Node {
	return receivingFormForID("rcv-illinois-001", payload)
}

func receivingFormForID(sessionID string, payload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(receivingFormState{
			Status:             "closed",
			DiscrepancySummary: "Short shipment recorded and available stock released.",
		})
		confirmOpen := ui.UseState(false)
		value := form.Get()
		previous := ui.UsePrevious(value)
		workflow := ui.UseReducer(reduceReceivingResolutionWorkflowState, normalizeReceivingResolutionWorkflowState(receivingResolutionWorkflowState{
			Status:             value.Status,
			DiscrepancySummary: value.DiscrepancySummary,
		}))
		workflowState := workflow.Get()
		openConfirm := ui.UseEvent(func() { confirmOpen.Set(true) })
		closeConfirm := func() { confirmOpen.Set(false) }
		children := []ui.Node{
			receivingResolutionSummaryCard(workflowState),
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Reconcile receiving")),
			receivingWorkflowInputWithValue("status", "Status", value.Status, "Status", form, workflow),
			receivingWorkflowTextareaWithValue("discrepancy_summary", "Discrepancy summary", value.DiscrepancySummary, "DiscrepancySummary", form, workflow),
			html.Button(html.Props{Type: "button", Class: "rounded-full bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950", OnClick: openConfirm}, html.Text("Review closeout")),
			atlasConfirmationDialog(confirmOpen.Get(), "atlas-receiving-confirm", "Confirm receiving closeout", "Atlas keeps focus inside the discrepancy confirmation step until you cancel or submit the reconciliation.", "Close session", closeConfirm,
				statCard("Status", value.Status),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(fallback(value.DiscrepancySummary, "No discrepancy note entered."))),
			),
		}
		if changeSummary := receivingDraftChangeSummary(value, previous); changeSummary != nil {
			children = append([]ui.Node{changeSummary}, children...)
		}
		return html.Form(html.Props{Action: "/api/app/receiving/" + sessionID + "/reconcile", Method: "post", Class: "grid gap-3 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"}, prependCSRFToken(payload.CSRF, children...)...)
	})
}

func receivingDraftChangeSummary(current receivingFormState, previous ui.Previous[receivingFormState]) ui.Node {
	if !previous.Ok() {
		return nil
	}
	prior := previous.Get()
	summary := ""
	switch {
	case strings.TrimSpace(prior.Status) != strings.TrimSpace(current.Status):
		summary = "Status changed from " + fallback(prior.Status, "unset") + " to " + fallback(current.Status, "unset")
	case strings.TrimSpace(prior.DiscrepancySummary) != strings.TrimSpace(current.DiscrepancySummary):
		summary = "Discrepancy notes were updated for the current reconciliation draft."
	}
	if summary == "" {
		return nil
	}
	return html.Div(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Recent reconcile edit")),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(summary)),
	)
}

func receivingResolutionSummaryCard(state receivingResolutionWorkflowState) ui.Node {
	stageLabel := "Classification open"
	stageCopy := "Atlas is still waiting for the reconcile outcome and discrepancy note to settle."
	switch state.Stage {
	case "closeout-ready":
		stageLabel = "Closeout ready"
		stageCopy = "Status and discrepancy notes line up, so this session can close cleanly."
	case "closeout-needs-note":
		stageLabel = "Closeout needs note"
		stageCopy = "Closed sessions should still explain the discrepancy or closeout outcome before submit."
	case "discrepancy-review":
		stageLabel = "Discrepancy review"
		stageCopy = "The session is still under review, so Atlas treats this as an active exception workflow."
	}
	edited := "Workflow staged from the current receiving defaults."
	if strings.TrimSpace(state.LastEditedField) != "" {
		edited = "Last updated field: " + strings.ReplaceAll(state.LastEditedField, "_", " ") + "."
	}
	return html.Div(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(stageLabel)),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(stageCopy)),
		html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.2em] text-cyan-200/75"}, html.Text(edited)),
	)
}

func receivingWorkflowInputWithValue(name, label, value, field string, form ui.Form[receivingFormState], workflow ui.Reducer[receivingResolutionWorkflowState, receivingResolutionWorkflowAction]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{
			ID: id, Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			OnInput: ui.UseEvent(func(event ui.InputEvent) {
				next := event.GetValue()
				form.SetField(field, next)
				workflow.Dispatch(receivingResolutionWorkflowAction{Field: name, Value: next})
			}),
			Raw: map[string]interface{}{"aria-labelledby": id + "-label"},
		}),
	)
}

func receivingWorkflowTextareaWithValue(name, label, value, field string, form ui.Form[receivingFormState], workflow ui.Reducer[receivingResolutionWorkflowState, receivingResolutionWorkflowAction]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Textarea(html.Props{
			ID: id, Name: name, Value: value, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			OnInput: ui.UseEvent(func(event ui.InputEvent) {
				next := event.GetValue()
				form.SetField(field, next)
				workflow.Dispatch(receivingResolutionWorkflowAction{Field: name, Value: next})
			}),
			Raw: map[string]interface{}{"aria-labelledby": id + "-label"},
		}, html.Text(value)),
	)
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
	fieldName, fieldValue := ui.NewCSRFToken(token).FormField()
	result := make([]ui.Node, 0, len(children)+1)
	result = append(result, html.HiddenInput(fieldName, fieldValue))
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
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Name: name, Value: value, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func inputWithValue(name, label, value string) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func boundInputWithValue[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func boundTransitionSelectWithValue[T any](name, label, value, field string, options []optionItem, form ui.Form[T], transition atlasTransition) ui.Node {
	id := ui.UseId()
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Select(html.Props{
			ID:   id,
			Name: name,
			OnChange: ui.UseEvent(func(event ui.ChangeEvent) {
				atlasSetFormFieldInTransition(form, field, event.GetValue())
			}),
			Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			Raw:   map[string]interface{}{"aria-labelledby": id + "-label", "data-transition": transition.Pending()},
		}, children...),
	)
}

func publicTextarea(name, label string) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Textarea(html.Props{ID: id, Name: name, Class: "min-h-28 rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func textareaWithValue(name, label, value string) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Textarea(html.Props{ID: id, Name: name, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, html.Text(value)),
	)
}

func boundTextareaWithValue[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Textarea(html.Props{ID: id, Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, html.Text(value)),
	)
}

func preferenceDensityPreviewCard(value string, pending bool) ui.Node {
	density := atlasNormalizedFilterValue(value)
	if density == "" {
		density = "compact"
	}
	className := "atlas-density-preview atlas-density-preview-compact"
	copy := "Compact density keeps Atlas information-dense for operators working wide tables and queue summaries."
	if density == "comfortable" {
		className = "atlas-density-preview atlas-density-preview-comfortable"
		copy = "Comfortable density increases whitespace so route summaries and action rails breathe more during review."
	}
	if pending {
		copy = "Applying the next density preview in a transition so the settings form stays responsive."
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-slate-950/45 p-4"},
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text("Density preview")),
		html.Div(html.Props{Class: className},
			html.Div(html.Props{Class: "rounded-full border border-cyan-300/35 bg-cyan-300/12 px-3 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-cyan-100"}, html.Text(strings.Title(density))),
			html.Div(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs uppercase tracking-[0.2em] text-slate-300"}, html.Text("Warehouse shell")),
			html.Div(html.Props{Class: "rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs uppercase tracking-[0.2em] text-slate-300"}, html.Text("Inventory rail")),
		),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
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

func internalWorkflowSection(title, copy string, cards ...ui.Node) ui.Node {
	return ui.CreateElement(func() ui.Node {
		open := ui.UseState(false)
		sheetID := ui.UseId() + "-workflow-sheet"
		titleID := sheetID + "-title"
		descriptionID := sheetID + "-description"
		closeID := sheetID + "-close"
		openDrawer := ui.UseEvent(func() { open.Set(true) })
		closeDrawer := func() { open.Set(false) }
		children := []ui.Node{
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(title)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
			),
			html.Button(html.Props{Type: "button", Class: "inline-flex w-fit items-center rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 lg:hidden", OnClick: openDrawer}, html.Text("Open quick actions")),
			html.Div(html.Props{Class: "hidden gap-4 md:grid-cols-2 xl:grid-cols-3 lg:grid"}, cards...),
			atlasDismissibleSheet(open.Get(), sheetID, titleID, descriptionID, "#"+closeID, closeDrawer, html.Div(html.Props{Class: "grid gap-4"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Quick actions")),
						html.P(html.Props{ID: titleID, Class: "text-lg font-semibold text-white"}, html.Text(title)),
						html.P(html.Props{ID: descriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text(copy)),
					),
					html.Button(html.Props{ID: closeID, Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: ui.UseEvent(func() { closeDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "grid gap-4"}, cards...),
			)),
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"}, children...)
	})
}

func internalWorkflowCard(step, title, copy, href string) ui.Node {
	return html.A(html.Props{Href: href, Class: "grid gap-3 " + internalAccentSurfaceClass() + " p-4 transition hover:border-cyan-300/55 hover:bg-[linear-gradient(180deg,rgba(10,24,42,0.96),rgba(7,14,26,0.99))]"},
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
		return "Explore Atlas delivery regions, compare service levels, and open the warehouse view that best fits your project timing."
	case strings.HasPrefix(path, "/app"):
		return "Internal routes keep buyer inbox, merchandising, inventory, transfers, receiving, and preferences inside one server-backed Atlas console."
	default:
		return "Atlas route payload hydrated successfully from the server bootstrap."
	}
}

func payloadDataValue(payload Payload, key string) any {
	if request, ok := payload.Requests[strings.TrimSpace(key)]; ok && len(request.Data) > 0 {
		if value, exists := request.Data[strings.TrimSpace(key)]; exists {
			return value
		}
	}
	if payload.Data == nil {
		return nil
	}
	return payload.Data[strings.TrimSpace(key)]
}

func pageData(payload Payload) any {
	return payloadDataValue(payload, "page")
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

func decodeJSONText[T any](value string) T {
	var result T
	if strings.TrimSpace(value) == "" {
		return result
	}
	_ = json.Unmarshal([]byte(value), &result)
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
