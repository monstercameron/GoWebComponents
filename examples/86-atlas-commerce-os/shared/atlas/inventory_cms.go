package atlas

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type inventoryCMSPage struct {
	Summary pageSummary       `json:"summary"`
	Items   []inventoryRow    `json:"items"`
	Filters map[string]string `json:"filters"`
}

type inventoryDetailPage struct {
	SKU   string         `json:"sku"`
	Title string         `json:"title"`
	Rows  []inventoryRow `json:"rows"`
}

type inventoryThresholdHistoryPanelPage struct {
	SKU             string                                `json:"sku"`
	Items           []inventoryThresholdHistoryItem       `json:"items"`
	Recommendations []inventoryTransferRecommendationItem `json:"recommendations"`
}

type inventoryThresholdHistoryItem struct {
	ID           string `json:"id"`
	ProductSKU   string `json:"productSku"`
	WarehouseID  string `json:"warehouseId"`
	ReorderPoint int    `json:"reorderPoint"`
	SafetyStock  int    `json:"safetyStock"`
	ActorName    string `json:"actorName"`
	Summary      string `json:"summary"`
	Detail       string `json:"detail"`
	CreatedAt    string `json:"createdAt"`
}

type inventoryTransferRecommendationItem struct {
	ProductSKU               string `json:"productSku"`
	SourceWarehouseID        string `json:"sourceWarehouseId"`
	SourceWarehouseName      string `json:"sourceWarehouseName"`
	DestinationWarehouseID   string `json:"destinationWarehouseId"`
	DestinationWarehouseName string `json:"destinationWarehouseName"`
	Quantity                 int    `json:"quantity"`
	Priority                 string `json:"priority"`
	Reason                   string `json:"reason"`
}

type warehouseInventoryDetailPage struct {
	Summary   pageSummary           `json:"summary"`
	Warehouse warehouseOpsRecord    `json:"warehouse"`
	Inventory []inventoryRow        `json:"inventory"`
	Orders    []purchaseOrderRecord `json:"orders"`
	Filters   map[string]string     `json:"filters"`
}

type inventorySummaryCard struct {
	SKU           string
	Title         string
	Status        string
	PrimaryLane   string
	Available     int
	Inbound       int
	LaneCount     int
	RiskLaneCount int
	LastUpdated   string
}

type inventoryWorkspaceSnapshot struct {
	Summaries []inventorySummaryCard
	Rollup    InventoryRollup
}

type warehouseDetailWorkspaceSnapshot struct {
	TotalDemand  int
	TotalRevenue int
	UrgentCount  int
	RiskLanes    int
}

type inventoryLaneEditorFormState struct {
	WarehouseID  string
	OnHand       string
	Reserved     string
	Damaged      string
	Inbound      string
	ReorderPoint string
	SafetyStock  string
	Status       string
	ReturnPath   string
}

type purchaseOrderWorkflowState struct {
	ProductSKU      string
	WarehouseID     string
	VendorName      string
	Quantity        string
	ETA             string
	PriorityNote    string
	Status          string
	ReturnPath      string
	Stage           string
	LastEditedField string
}

type purchaseOrderWorkflowAction struct {
	Field string
	Value string
}

func defaultPurchaseOrderWorkflowState(defaultSKU, defaultWarehouse, returnPath string) purchaseOrderWorkflowState {
	return normalizePurchaseOrderWorkflowState(purchaseOrderWorkflowState{
		ProductSKU:   defaultSKU,
		WarehouseID:  defaultWarehouse,
		VendorName:   "Northline Fabrication",
		Quantity:     "12",
		ETA:          "Tue 10:00",
		PriorityNote: "Replenish active warehouse lane",
		Status:       "submitted",
		ReturnPath:   returnPath,
	})
}

func reducePurchaseOrderWorkflowState(state purchaseOrderWorkflowState, action purchaseOrderWorkflowAction) purchaseOrderWorkflowState {
	switch action.Field {
	case "product_sku":
		state.ProductSKU = action.Value
	case "warehouse_id":
		state.WarehouseID = action.Value
	case "vendor_name":
		state.VendorName = action.Value
	case "quantity":
		state.Quantity = action.Value
	case "eta":
		state.ETA = action.Value
	case "priority_note":
		state.PriorityNote = action.Value
	case "status":
		state.Status = action.Value
	case "return_path":
		state.ReturnPath = action.Value
	}
	state.LastEditedField = action.Field
	return normalizePurchaseOrderWorkflowState(state)
}

func normalizePurchaseOrderWorkflowState(state purchaseOrderWorkflowState) purchaseOrderWorkflowState {
	status := strings.TrimSpace(strings.ToLower(state.Status))
	quantity := strings.TrimSpace(state.Quantity)
	priority := strings.TrimSpace(state.PriorityNote)
	switch {
	case status == "approved":
		state.Stage = "inbound-confirmed"
	case status == "on_hold":
		state.Stage = "blocked"
	case status == "draft" && priority == "":
		state.Stage = "draft-needs-brief"
	case status == "draft":
		state.Stage = "draft-review"
	case quantity == "" || quantity == "0":
		state.Stage = "needs-quantity"
	default:
		state.Stage = "ready-to-submit"
	}
	return state
}

func inventoryWorkspaceSnapshotFromRows(rows []inventoryRow) inventoryWorkspaceSnapshot {
	return inventoryWorkspaceSnapshot{
		Summaries: inventorySummaryCards(rows),
		Rollup:    inventoryRollupFromRows(rows),
	}
}

func warehouseDetailWorkspaceSnapshotFromPage(page warehouseInventoryDetailPage) warehouseDetailWorkspaceSnapshot {
	snapshot := warehouseDetailWorkspaceSnapshot{}
	for _, item := range page.Inventory {
		snapshot.TotalDemand += item.WeeklyUnits
		snapshot.TotalRevenue += item.WeeklyRevenue
		if item.ReorderUnits > 0 || strings.EqualFold(item.MarketPressure, "hot market") {
			snapshot.UrgentCount++
		}
		if strings.EqualFold(strings.TrimSpace(item.Status), "critical") || strings.EqualFold(strings.TrimSpace(item.Status), "promise_risk") {
			snapshot.RiskLanes++
		}
	}
	return snapshot
}

func inventoryCMSContent(payload Payload) ui.Node {
	page := decode[inventoryCMSPage](pageData(payload))
	return ui.CreateElement(func() ui.Node {
		search := useAtlasSearchParams()
		shellState := currentRouteWorkspaceState(payload)
		initial := atlasMapFilterState(page.Filters)
		form := ui.UseForm(initial)
		transition := useAtlasTransition()
		useAtlasEffect(func() func() {
			if !atlasSameListFilterState(form.Get(), initial) {
				form.Reset(initial)
			}
			return nil
		}, initial)
		value := form.Get()
		deferred := ui.UseDeferredValue(value)
		debounced := ui.UseDebounced(value, atlasFilterSyncDelay)
		useAtlasEffect(func() func() {
			next := atlasBuildListFilterQuery(search.Values(), debounced.Get())
			if next.Encode() != search.Values().Encode() {
				search.ReplaceAll(next)
			}
			return nil
		}, debounced.Get())
		submit := ui.UseEvent(func(event ui.FormEvent) {
			event.PreventDefault()
			search.ReplaceAll(atlasBuildListFilterQuery(search.Values(), form.Get()))
		})
		workspace := inventoryWorkspaceSnapshotFromRows(filterInventoryRows(page.Items, deferred))
		viewNodes := make([]ui.Node, 0, len(payload.SavedViews))
		for _, saved := range payload.SavedViews {
			currentSaved := saved
			nextState := atlasSavedViewFilterState(currentSaved)
			className := "grid gap-2 border px-3 py-3 text-left transition"
			if atlasSameListFilterState(value, nextState) {
				className += " border-cyan-400/45 bg-cyan-500/10"
			} else {
				className += " border-slate-700 bg-slate-950/80 hover:border-cyan-400/45"
			}
			viewNodes = append(viewNodes, html.Button(html.Props{
				Type:  "button",
				Class: className,
				OnClick: ui.UseEvent(func() {
					transition.Start(func() {
						form.Set(nextState)
						search.ReplaceAll(atlasBuildListFilterQuery(search.Values(), nextState))
					})
				}),
			},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(saved.Name)),
				html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text(saved.Scope+" | "+saved.SortKey+"/"+saved.SortDirection)),
			))
		}
		if len(viewNodes) == 0 {
			viewNodes = append(viewNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No saved inventory views yet.")))
		} else if transition.Pending() {
			viewNodes = append(viewNodes, html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-200"}, html.Text("Applying the selected saved view without blocking the triage workspace.")))
		}
		return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.26fr)_minmax(22rem,0.74fr)] xl:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				inventoryCMSSummaryBand(page, workspace),
				inventoryCMSFilterForm(form, debounced.Pending() || transition.Pending(), submit, transition),
				inventoryCMSActionCluster(),
				inventoryTriageBand(workspace),
				inventoryQueueTable(workspace.Summaries),
			),
			html.Div(html.Props{Class: "grid gap-5"},
				inventoryOperationsRail(shellState),
				inventorySavedViewsRail(viewNodes),
				savedViewForm(payload),
			),
		)
	})
}

func inventoryCMSSummaryBand(page inventoryCMSPage, workspace inventoryWorkspaceSnapshot) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Inventory triage shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Keep the inventory route focused on operator triage: visible stock pressure, fast filter pivots, and direct SKU handoffs into lane-level correction without detouring through decorative route explainer cards.")),
		),
		routeSummaryStrip(page.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Visible SKUs", fmt.Sprintf("%d tracked", len(workspace.Summaries))),
			statCard("Risk lanes", fmt.Sprintf("%d flagged", workspace.Rollup.RiskLanes)),
			statCard("Inbound units", fmt.Sprintf("%d queued", workspace.Rollup.InboundUnits)),
			statCard("Available units", fmt.Sprintf("%d ready", workspace.Rollup.TotalAvailable)),
		),
	)
}

func inventoryCMSActionCluster() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Route action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Move straight into the pressure queue that matters now: critical lanes, promise risk, or the downstream receiving and warehouse workflows that resolve those problems.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
			inventoryActionCard("Critical lanes", "Review the lanes already in failure posture before promise slip becomes customer-visible.", "/app/inventory?status=critical"),
			inventoryActionCard("Promise-risk queue", "Open constrained SKUs first when inbound or threshold changes are needed to protect active promises.", "/app/inventory?status=promise_risk"),
			inventoryActionCard("Receiving follow-up", "Jump into receiving when inbound work is already staged and the issue is confirmation rather than replenishment planning.", "/app/receiving"),
		),
	)
}

func inventoryActionCard(title, copy, href string) ui.Node {
	return html.A(html.Props{Href: href, Class: "grid gap-3 " + internalAccentSurfaceClass() + " p-4 transition hover:border-cyan-300/55 hover:bg-[linear-gradient(180deg,rgba(10,24,42,0.96),rgba(7,14,26,0.99))]"},
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Open workflow")),
	)
}

func inventoryTriageBand(workspace inventoryWorkspaceSnapshot) ui.Node {
	rollup := workspace.Rollup
	items := []struct {
		label string
		value string
		copy  string
		href  string
	}{
		{
			label: "Critical lanes",
			value: fmt.Sprintf("%d", rollup.CriticalLanes),
			copy:  "Immediate lane corrections before promise failure.",
			href:  "/app/inventory?status=critical",
		},
		{
			label: "Promise risk",
			value: fmt.Sprintf("%d", rollup.PromiseRiskLanes),
			copy:  "SKUs likely to slip without attention.",
			href:  "/app/inventory?status=promise_risk",
		},
		{
			label: "Inbound pending",
			value: fmt.Sprintf("%d", rollup.SKUsWithInbound),
			copy:  "Items with open inbound already on the way.",
			href:  "/app/receiving",
		},
		{
			label: "Reorder now",
			value: fmt.Sprintf("%d", rollup.ReorderLanes),
			copy:  "Lanes already signaling vendor replenishment.",
			href:  "/app/purchase-orders",
		},
	}
	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, html.A(html.Props{Href: item.href, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text(item.label)),
			html.P(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(item.value)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.copy)),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Triage summary band")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The lead inventory counts stay clickable so operators can pivot from a high-level pressure readout into the exact queue that needs intervention.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2 xl:grid-cols-4"}, nodes...),
	)
}

func atlasSectionMeta(eyebrow, copy string) ui.Node {
	return ui.Fragment(
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text(eyebrow)),
		html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(copy)),
	)
}

func inventoryQueueRowCells(item inventorySummaryCard) ui.Node {
	return ui.Fragment(
		html.Tag("td", html.Props{Class: "px-3 py-3"},
			html.A(html.Props{Href: "/app/inventory/" + item.SKU, Class: "grid gap-2 transition hover:text-cyan-100"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.Title)),
					html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " "))),
				),
				html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-500"}, html.Text(item.SKU)),
			),
		),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-300"}, html.Text(item.PrimaryLane)),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Available))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Inbound))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.RiskLaneCount))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-400"}, html.Text(item.LastUpdated)),
	)
}

func inventoryQueueTable(items []inventorySummaryCard) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, inventoryQueueTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 8}}, html.Text("No inventory rows match the current filter set. Clear filters or pivot into a saved view with live lane pressure.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Dense queue table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scan SKU posture, lane spread, and inbound exposure in one table, then branch directly into the SKU route or warehouse lane workspace that needs edits.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d rows", len(items)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Item")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Primary lane")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Lane spread")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Inbound")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func inventoryQueueTableRow(item inventorySummaryCard) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/inventory/" + item.SKU, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.Title)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(item.SKU)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.PrimaryLane)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d total lanes", item.LaneCount))),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-300"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fmt.Sprintf("%d risk lanes", item.RiskLaneCount))),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d healthy lanes", maxInt(item.LaneCount-item.RiskLaneCount, 0)))),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Available))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Inbound))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(item.LastUpdated)),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/inventory/" + item.SKU, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open SKU")),
				html.A(html.Props{Href: "/app/warehouses", Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open warehouse ops")),
			),
		),
	)
}

func inventoryOperationsRail(workspace internalShellState) ui.Node {
	filterNodes := workspaceFilterSummaryNodes(workspace.ActiveFilters, "All inventory rows are visible.")
	statNodes := []ui.Node{}
	for _, item := range workspace.WorkspaceStats {
		statNodes = append(statNodes, warehouseInfoRow(item.Label, item.Value))
	}
	if len(statNodes) == 0 {
		statNodes = append(statNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No shared workspace summary is available for this route yet.")))
	}
	viewTitle := "Saved-view selection"
	viewBody := []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No saved inventory view currently matches the active workspace filters."))}
	if strings.TrimSpace(workspace.ActiveSavedView) != "" {
		viewBody = []ui.Node{warehouseInfoRow("Active view", workspace.ActiveSavedView)}
	}
	actionNodes := []ui.Node{
		html.A(html.Props{Href: "/app/inventory?status=promise_risk", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open promise-risk lanes")),
		html.A(html.Props{Href: "/app/warehouses", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Switch to warehouse ops")),
		html.A(html.Props{Href: "/app/purchase-orders", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open purchase orders")),
		html.A(html.Props{Href: "/app/receiving", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Check receiving queue")),
	}
	return html.Div(html.Props{Class: "grid gap-5"},
		inventoryRailCard("Current view", "Keep the route rail focused on the exact workspace scope currently driving the queue table.",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"}, statNodes...),
		),
		inventoryRailCard("Active filters", "The right rail mirrors the queue scope so saved views and side workflows stay interpretable while filters change.", filterNodes...),
		inventoryRailCard(viewTitle, "Atlas keeps the active saved-view selection visible in the route rail so operators can confirm why the current queue looks the way it does.", viewBody...),
		inventoryRailCard("Immediate actions", "Use the rail for adjacent operational hops after the main queue identifies the problem surface.", actionNodes...),
	)
}

func inventorySavedViewsRail(viewNodes []ui.Node) ui.Node {
	return inventoryRailCard("Saved views", "Apply inventory presets without losing the triage shell. Saved views stay in the side rail so the queue remains the route's main visual priority.", viewNodes...)
}

func inventoryRailCard(title, copy string, children ...ui.Node) ui.Node {
	content := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text(title)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		),
	}
	content = append(content, children...)
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"}, content...)
}

func workspaceFilterSummaryNodes(labels []string, emptyCopy string) []ui.Node {
	if len(labels) == 0 {
		return []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(emptyCopy))}
	}
	nodes := make([]ui.Node, 0, len(labels))
	for _, label := range labels {
		nodes = append(nodes, html.Div(html.Props{Class: "border border-slate-700 bg-slate-950/75 px-3 py-3 text-sm text-slate-200"}, html.Text(label)))
	}
	return nodes
}

func inventoryDetailContent(payload Payload) ui.Node {
	page := decode[inventoryDetailPage](pageData(payload))
	if len(page.Rows) == 0 {
		return featureCard("Inventory item unavailable", "This SKU does not currently have any active inventory lanes.")
	}
	currentPath := "/app/inventory/" + page.SKU
	rollup := inventoryRollupFromRows(page.Rows)
	laneCards := make([]ui.Node, 0, len(page.Rows))
	for _, row := range page.Rows {
		laneCards = append(laneCards, inventoryLaneEditorCardWithOptions(row, payload, currentPath))
	}
	asideChildren := []ui.Node{
		skuOperationsRail(page, rollup.TotalAvailable, rollup.InboundUnits, rollup.RiskLanes),
		html.Div(html.Props{ID: "sku-replenishment"}, orderInventoryModalCard(page.Rows, page.Rows[0].WarehouseID, page.Title, payload, true, currentPath)),
	}
	if overlay := inventoryThresholdHistoryPanelNode(payload, page); overlay != nil {
		asideChildren = append(asideChildren, overlay)
	}
	asideChildren = append(asideChildren, thresholdForm(page.Rows[0], payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.18fr)_minmax(23rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryDetailHero(page, rollup),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				statCard("Available", fmt.Sprintf("%d units", rollup.TotalAvailable)),
				statCard("Inbound", fmt.Sprintf("%d units", rollup.InboundUnits)),
				statCard("Flagged lanes", fmt.Sprintf("%d lanes", rollup.RiskLanes)),
				statCard("Reorder lanes", fmt.Sprintf("%d lanes", rollup.ReorderLanes)),
			),
			skuLaneRosterTable(page.Rows),
			inventoryLaneEditorsSection(laneCards),
		),
		html.Div(html.Props{Class: "grid gap-5"}, asideChildren...),
	)
}

func inventoryDetailHero(page inventoryDetailPage, rollup InventoryRollup) ui.Node {
	primaryWarehouse := fallback(page.Rows[0].WarehouseName, page.Rows[0].WarehouseID)
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Inventory lane workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(page.Title)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("%s is active across %d warehouse lanes. Use this route to compare pressure, edit the exact lane that drifted, and branch into replenishment only when balancing is no longer enough.", page.SKU, len(page.Rows)))),
				),
			),
			html.A(html.Props{Href: "/app/inventory", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to inventory queue")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text("SKU "+page.SKU)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fmt.Sprintf("%d lanes live", len(page.Rows)))),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(primaryWarehouse+" primary")),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fmt.Sprintf("%d risk lanes", rollup.RiskLanes))),
		),
	)
}

func inventoryThresholdHistoryPanelNode(payload Payload, page inventoryDetailPage) ui.Node {
	if outlet := routeOutletNode(); outlet != nil {
		return outlet
	}
	panel := decode[inventoryThresholdHistoryPanelPage](payloadDataValue(payload, "overlay"))
	if panel.SKU == "" || !strings.EqualFold(strings.TrimSpace(panel.SKU), strings.TrimSpace(page.SKU)) {
		return nil
	}
	return inventoryThresholdHistoryPanel(panel, page)
}

func InventoryThresholdHistoryOverlay(payload Payload) ui.Node {
	panel := decode[inventoryThresholdHistoryPanelPage](payloadDataValue(payload, "overlay"))
	page := decode[inventoryDetailPage](payloadDataValue(payload, "page"))
	if panel.SKU == "" || page.SKU == "" {
		return nil
	}
	return inventoryThresholdHistoryPanel(panel, page)
}

func inventoryThresholdHistoryPanel(panel inventoryThresholdHistoryPanelPage, page inventoryDetailPage) ui.Node {
	closeHref := "/app/inventory/" + page.SKU
	panelID := ui.UseId()
	titleID := panelID + "-title"
	descriptionID := panelID + "-description"
	closeID := panelID + "-close"
	latestSignature := ""
	if len(panel.Items) > 0 {
		latest := panel.Items[0]
		latestSignature = latest.ID + ":" + fmt.Sprintf("%d", latest.ReorderPoint) + ":" + fmt.Sprintf("%d", latest.SafetyStock) + ":" + latest.CreatedAt
	}
	previousLatest := ui.UsePrevious(latestSignature)
	historyNodes := make([]ui.Node, 0, len(panel.Items))
	for _, item := range panel.Items {
		historyNodes = append(historyNodes, html.Div(html.Props{Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/75 px-4 py-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.Summary)),
				html.Span(html.Props{Class: warehouseStatusClass("pending")}, html.Text(item.WarehouseID)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Detail)),
			html.Div(html.Props{Class: "grid gap-2 text-xs uppercase tracking-[0.2em] text-slate-500 md:grid-cols-3"},
				html.Span(html.Props{}, html.Text("Reorder "+fmt.Sprintf("%d", item.ReorderPoint))),
				html.Span(html.Props{}, html.Text("Safety "+fmt.Sprintf("%d", item.SafetyStock))),
				html.Span(html.Props{}, html.Text(item.CreatedAt)),
			),
			html.P(html.Props{Class: "text-xs text-slate-500"}, html.Text("Updated by "+fallback(item.ActorName, "Atlas operations"))),
		))
	}
	if len(historyNodes) == 0 {
		historyNodes = append(historyNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No threshold history events are recorded for this SKU yet.")))
	}
	recommendationNodes := make([]ui.Node, 0, len(panel.Recommendations))
	for _, item := range panel.Recommendations {
		recommendationNodes = append(recommendationNodes, html.Div(html.Props{Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/75 px-4 py-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(fallback(item.SourceWarehouseName, item.SourceWarehouseID)+" -> "+fallback(item.DestinationWarehouseName, item.DestinationWarehouseID))),
				html.Span(html.Props{Class: warehouseStatusClass(item.Priority)}, html.Text(fallback(item.Priority, "review"))),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Reason)),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(fmt.Sprintf("%d units of %s", item.Quantity, page.SKU))),
		))
	}
	if len(recommendationNodes) == 0 {
		recommendationNodes = append(recommendationNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No transfer recommendation is stronger than local threshold tuning right now.")))
	}
	children := []ui.Node{
		html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Route overlay")),
				html.P(html.Props{ID: titleID, Class: "text-lg font-semibold text-white"}, html.Text("Threshold history")),
				html.P(html.Props{ID: descriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text("This side panel is deep-linkable, so operators can review threshold decisions without losing the SKU route context.")),
			),
			html.A(html.Props{Href: closeHref, Class: warehouseSecondaryButtonClass()}, html.Text("Close")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
			warehouseInfoRow("SKU", page.SKU),
			warehouseInfoRow("History events", fmt.Sprintf("%d", len(panel.Items))),
			warehouseInfoRow("Transfer cues", fmt.Sprintf("%d", len(panel.Recommendations))),
			warehouseInfoRow("Return path", closeHref),
		),
		warehouseListCard("Threshold changes", historyNodes...),
		warehouseListCard("Transfer recommendations", recommendationNodes...),
	}
	if changeSummary := thresholdHistoryChangeSummary(panel.Items, previousLatest, latestSignature); changeSummary != nil {
		children = append([]ui.Node{changeSummary}, children...)
	}
	children[0] = html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Route overlay")),
			html.P(html.Props{ID: titleID, Class: "text-lg font-semibold text-white"}, html.Text("Threshold history")),
			html.P(html.Props{ID: descriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text("This side panel is deep-linkable, so operators can review threshold decisions without losing the SKU route context.")),
		),
		html.A(html.Props{ID: closeID, Href: closeHref, Class: warehouseSecondaryButtonClass()}, html.Text("Close")),
	)
	return atlasRouteSheetOverlay(panelID, titleID, descriptionID, "#"+closeID, html.Div(html.Props{
		Class: "grid gap-4",
		Raw: map[string]interface{}{
			"data-atlas-route-overlay": "threshold-history",
		},
	}, children...))
}

func thresholdHistoryChangeSummary(items []inventoryThresholdHistoryItem, previous ui.Previous[string], latestSignature string) ui.Node {
	if !previous.Ok() || latestSignature == "" || previous.Get() == "" || previous.Get() == latestSignature || len(items) == 0 {
		return nil
	}
	latest := items[0]
	return html.Div(html.Props{Class: "rounded-sm border border-cyan-300/25 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Latest threshold change")),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(fmt.Sprintf("Timeline refreshed with reorder %d and safety %d for %s.", latest.ReorderPoint, latest.SafetyStock, latest.WarehouseID))),
	)
}

func skuLaneRosterTable(rows []inventoryRow) ui.Node {
	items := make([]ui.Node, 0, len(rows))
	for _, row := range rows {
		items = append(items, html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
			html.Tag("td", html.Props{Class: "px-4 py-4"},
				html.A(html.Props{Href: "/app/warehouses/" + row.WarehouseID + "/items/" + row.SKU, Class: "grid gap-2 transition hover:text-cyan-100"},
					html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
						html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(row.WarehouseName, row.WarehouseID))),
						html.Span(html.Props{Class: warehouseStatusClass(row.Status)}, html.Text(strings.ReplaceAll(row.Status, "_", " "))),
					),
					html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-500"}, html.Text(row.WarehouseID+" | "+row.MarketPressure)),
				),
			),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", row.Available))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", row.Inbound))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d days", row.CoverDays))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", row.ReorderUnits))),
			html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(row.UpdatedAt)),
			html.Tag("td", html.Props{Class: "px-4 py-4"},
				html.A(html.Props{Href: "/app/warehouses/" + row.WarehouseID + "/items/" + row.SKU, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open lane")),
			),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Lane roster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Compare every active warehouse lane before touching quantities, thresholds, or replenishment so the operator can tell whether the issue is local or network-wide.")),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Warehouse lane")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Inbound")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Cover")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Reorder")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, items...),
			),
		),
	)
}

func inventoryLaneEditorsSection(laneCards []ui.Node) ui.Node {
	content := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Lane editors")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Edit each lane in place after the roster confirms where the pressure really lives. Quantity, threshold, and status changes stay grouped under one inventory-specific section instead of scattered across utility cards.")),
		),
	}
	content = append(content, laneCards...)
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"}, content...)
}

func skuOperationsRail(page inventoryDetailPage, totalAvailable int, totalInbound int, riskLanes int) ui.Node {
	primaryHref := inventoryPrimaryWarehouseHref(page.Rows)
	return html.Div(html.Props{Class: "grid gap-5"},
		inventoryRailCard("SKU actions", "Keep the right rail focused on the immediate handoffs around this SKU: merchandising, warehouse drill-in, replenishment, and receiving follow-up.",
			html.Div(html.Props{Class: "grid gap-3"},
				inventoryActionCard("Open product record", "Jump into the matching product editor when the lane issue surfaces a catalog or copy follow-up.", "/app/products?q="+url.QueryEscape(page.SKU)),
				inventoryActionCard("Open primary warehouse item", "Drill into the warehouse-native item workspace when this issue is facility-specific.", primaryHref),
				inventoryActionCard("Open purchase orders", "Move into replenishment review if this SKU already needs vendor-side recovery.", "/app/purchase-orders"),
				inventoryActionCard("Open receiving", "Hand off to receiving once inbound work is actually moving and needs confirmation.", "/app/receiving"),
			),
		),
		inventoryRailCard("SKU snapshot", "Use the route rail to confirm lane count, stock posture, and the current anchor warehouse before changing the lane editors below.",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
				warehouseInfoRow("SKU", page.SKU),
				warehouseInfoRow("Active lanes", fmt.Sprintf("%d", len(page.Rows))),
				warehouseInfoRow("Available units", fmt.Sprintf("%d", totalAvailable)),
				warehouseInfoRow("Inbound units", fmt.Sprintf("%d", totalInbound)),
				warehouseInfoRow("Flagged lanes", fmt.Sprintf("%d", riskLanes)),
				warehouseInfoRow("Primary warehouse", fallback(page.Rows[0].WarehouseName, page.Rows[0].WarehouseID)),
			),
		),
		inventoryRailCard("What this page controls", "This route is for lane comparison first, then lane correction, with replenishment and threshold work treated as explicit secondary workflows.",
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text("Use the lane roster to compare warehouses, edit the specific lane below, then create replenishment only when the numbers show that internal balancing is not enough.")),
		),
	)
}

func warehouseInventoryDetailContent(payload Payload) ui.Node {
	page := decode[warehouseInventoryDetailPage](pageData(payload))
	return ui.CreateElement(func() ui.Node {
		search := useAtlasSearchParams()
		initial := atlasMapFilterState(page.Filters)
		form := ui.UseForm(initial)
		transition := useAtlasTransition()
		useAtlasEffect(func() func() {
			if !atlasSameListFilterState(form.Get(), initial) {
				form.Reset(initial)
			}
			return nil
		}, initial)
		value := form.Get()
		deferred := ui.UseDeferredValue(value)
		debounced := ui.UseDebounced(value, atlasFilterSyncDelay)
		useAtlasEffect(func() func() {
			next := atlasBuildListFilterQuery(search.Values(), debounced.Get())
			if next.Encode() != search.Values().Encode() {
				search.ReplaceAll(next)
			}
			return nil
		}, debounced.Get())
		submit := ui.UseEvent(func(event ui.FormEvent) {
			event.PreventDefault()
			search.ReplaceAll(atlasBuildListFilterQuery(search.Values(), form.Get()))
		})
		visibleInventory := filterWarehouseInventoryRows(page.Inventory, deferred)
		workspace := warehouseDetailWorkspaceSnapshot{}
		for _, item := range visibleInventory {
			workspace.TotalDemand += item.WeeklyUnits
			workspace.TotalRevenue += item.WeeklyRevenue
			if item.ReorderUnits > 0 || strings.EqualFold(item.MarketPressure, "hot market") {
				workspace.UrgentCount++
			}
			if strings.EqualFold(strings.TrimSpace(item.Status), "critical") || strings.EqualFold(strings.TrimSpace(item.Status), "promise_risk") {
				workspace.RiskLanes++
			}
		}
		orderNodes := make([]ui.Node, 0, minInt(len(page.Orders), 4))
		for index, order := range page.Orders {
			if index >= 4 {
				break
			}
			orderNodes = append(orderNodes,
				html.A(html.Props{Href: "/app/purchase-orders/" + order.ID, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
					html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(order.ID+" | "+order.VendorName)),
					html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(order.Status+" | ETA "+order.ETA)),
				),
			)
		}
		if len(orderNodes) == 0 {
			orderNodes = append(orderNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No recent replenishment orders are tied to this warehouse yet.")))
		}
		return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.14fr)_minmax(24rem,0.86fr)] xl:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				warehouseBreadcrumbBar(
					warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
					warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
					warehouseBreadcrumbLink{Label: fallback(page.Warehouse.Name, page.Warehouse.ID), Href: "/app/warehouses/" + page.Warehouse.ID, Current: true},
				),
				warehouseDetailHero(page, workspace),
				routeSummaryStrip(page.Summary),
				warehouseInventoryFilterForm(page.Warehouse.ID, form, debounced.Pending() || transition.Pending(), submit, transition),
				html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
					statCard("Warehouse items", fmt.Sprintf("%d active", len(visibleInventory))),
					statCard("Weekly demand", fmt.Sprintf("%d units", workspace.TotalDemand)),
					statCard("Weekly revenue", formatPrice(workspace.TotalRevenue)),
					statCard("Urgent actions", fmt.Sprintf("%d flagged", workspace.UrgentCount)),
				),
				warehouseDetailActionCluster(page.Warehouse.ID),
				warehouseOpsItemPanelNode(payload),
				warehouseDetailInventoryTable(page.Warehouse.ID, visibleInventory),
			),
			html.Div(html.Props{Class: "grid gap-5"},
				inventoryRailCard("Facility snapshot", "Use the right rail to keep the facility context visible while the main pane stays focused on the dense item roster.",
					html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
						warehouseInfoRow("Region", page.Warehouse.Region),
						warehouseInfoRow("Service level", page.Warehouse.ServiceLevel),
						warehouseInfoRow("Pressure", page.Warehouse.Pressure),
						warehouseInfoRow("Backlog", page.Warehouse.Backlog),
						warehouseInfoRow("Staffing", page.Warehouse.Staffing),
						warehouseInfoRow("Risk lanes", fmt.Sprintf("%d", page.Warehouse.RiskCount)),
					),
				),
				inventoryRailCard("Active workspace filters", "Warehouse filters stay mirrored in the rail so nested item work and replenishment actions remain interpretable.", workspaceFilterSummaryNodes(currentRouteWorkspaceState(payload).ActiveFilters, "All warehouse items are visible.")...),
				html.Div(html.Props{ID: "warehouse-create-item"}, productCreateFormWithOptions(payload, productFormOptions{LockedWarehouseID: page.Warehouse.ID, ReturnWarehouseID: page.Warehouse.ID, IntroLabel: "Add warehouse item", SubmitLabel: "Create warehouse item"})),
				html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard(page.Inventory, page.Warehouse.ID, page.Warehouse.Name, payload, false, "/app/warehouses/"+page.Warehouse.ID)),
				inventoryRailCard("Recent purchase orders", "Keep recent replenishment work visible beside the warehouse roster so operators can decide whether to edit locally or follow existing inbound recovery.", orderNodes...),
			),
		)
	})
}

func warehouseDetailHero(page warehouseInventoryDetailPage, workspace warehouseDetailWorkspaceSnapshot) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Warehouse facility workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(page.Warehouse.Name)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(page.Warehouse.Focus)),
				),
			),
			html.A(html.Props{Href: RouteWarehouseOps, Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to facility table")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Warehouse.Region)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Warehouse.ServiceLevel)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(page.Warehouse.Pressure)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fmt.Sprintf("%d risk lanes", workspace.RiskLanes))),
		),
	)
}

func warehouseDetailActionCluster(warehouseID string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Facility action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Use the warehouse route for item creation, flagged-lane review, replenishment staging, and receiving follow-up without losing the facility context.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
			inventoryActionCard("Add warehouse item", "Create a new managed item directly inside this warehouse workspace.", "/app/warehouses/"+warehouseID+"#warehouse-create-item"),
			inventoryActionCard("Review flagged lanes", "Tighten the route to the lanes that need action before editing quantities or thresholds.", "/app/warehouses/"+warehouseID+"?status=promise_risk"),
			inventoryActionCard("Open inventory risk", "Compare this facility against the wider SKU pressure board before committing local changes.", "/app/inventory?status=promise_risk"),
			inventoryActionCard("Check receiving", "Follow inbound recovery through receiving once replenishment has actually moved.", "/app/receiving"),
		),
	)
}

func warehouseDetailInventoryTable(warehouseID string, items []inventoryRow) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, warehouseDetailInventoryTableRow(warehouseID, item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 7}}, html.Text("No warehouse items match the current view. Adjust filters or create a new managed item from the route rail.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Warehouse item table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review item posture, demand, revenue, and reorder pressure in one dense facility table before drilling into the nested item workspace.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d items", len(items)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-950/80"},
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Item")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Demand")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Revenue")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Reorder")),
						html.Tag("th", html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func warehouseDetailInventoryTableRow(warehouseID string, item inventoryRow) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + warehouseID + "/items/" + item.SKU, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.Title)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(item.SKU+" | "+strings.Title(item.Category))),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.MarketSignal+" | "+item.MarketPressure)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Available))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d / week", item.WeeklyUnits))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(formatPrice(item.WeeklyRevenue))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.ReorderUnits))),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + warehouseID + "/items/" + item.SKU, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open item")),
				html.A(html.Props{Href: "/app/products/" + item.Slug, Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open product")),
			),
		),
	)
}

func WarehouseOpsItemPanel(payload Payload) ui.Node {
	page := decode[warehouseInventoryItemDetailPage](payloadDataValue(payload, "item"))
	if strings.TrimSpace(page.Item.SKU) == "" {
		page = decode[warehouseInventoryItemDetailPage](pageData(payload))
	}
	if strings.TrimSpace(page.Item.SKU) == "" {
		return nil
	}
	return warehouseOpsItemNestedContent(payload, page)
}

func warehouseOpsItemPanelNode(payload Payload) ui.Node {
	if outlet := routeOutletNode(); outlet != nil {
		return outlet
	}
	return WarehouseOpsItemPanel(payload)
}

func warehouseOpsItemNestedContent(payload Payload, page warehouseInventoryItemDetailPage) ui.Node {
	currentPath := "/app/warehouses/" + page.Warehouse.ID + "/items/" + page.Item.SKU
	parentPath := warehouseParentWorkspaceHref(payload, page.Warehouse.ID)
	filters := page.Filters
	if filters == nil {
		filters = map[string]string{"sort": "updated", "dir": "desc"}
	}
	if strings.TrimSpace(filters["sort"]) == "" {
		filters["sort"] = "updated"
	}
	if strings.TrimSpace(filters["dir"]) == "" {
		filters["dir"] = warehouseItemDefaultDirection(filters["sort"])
	}
	orderNodes := make([]ui.Node, 0, len(page.Orders))
	for _, order := range page.Orders {
		orderNodes = append(orderNodes, html.A(html.Props{Href: "/app/purchase-orders/" + order.ID, Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 transition hover:border-cyan-400/45 hover:bg-slate-950"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(order.ID+" · "+order.VendorName)),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(order.Status+" · ETA "+order.ETA)),
		))
	}
	if len(orderNodes) == 0 {
		orderNodes = append(orderNodes, html.Div(html.Props{Class: "rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-400"}, html.Text("No open replenishment orders are tied to this item in the current warehouse.")))
	}
	return html.Section(html.Props{Class: "grid gap-5 rounded-[1.4rem] border border-cyan-400/25 bg-[linear-gradient(180deg,rgba(9,14,29,0.98),rgba(15,23,42,0.98))] p-5 shadow-[0_28px_70px_rgba(8,15,28,0.42)] lg:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.88fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-slate-950/70 p-4"},
				html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-3"},
					html.Div(html.Props{Class: "grid gap-1"},
						html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Nested warehouse item workspace")),
						html.H2(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(page.Product.Title)),
						html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(page.Product.Summary)),
					),
					html.A(html.Props{Href: parentPath, Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-3 py-2 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Back to warehouse roster")),
				),
				html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-300 md:grid-cols-4"},
					warehouseInfoRow("SKU", page.Item.SKU),
					warehouseInfoRow("Warehouse", fallback(page.Warehouse.Name, page.Warehouse.ID)),
					warehouseInfoRow("Status", strings.ReplaceAll(page.Item.Status, "_", " ")),
					warehouseInfoRow("Regional pressure", page.Item.MarketPressure),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
				warehouseStatCard("Available", fmt.Sprintf("%d units", page.Item.Available)),
				warehouseStatCard("Weekly demand", fmt.Sprintf("%d units", page.Item.WeeklyUnits)),
				warehouseStatCard("Weekly revenue", formatPrice(page.Item.WeeklyRevenue)),
				warehouseStatCard("Order more", fmt.Sprintf("%d units", page.Item.ReorderUnits)),
			),
			internalWorkflowSection("Warehouse item admin flows", "Stay inside the warehouse workspace while you fix this lane, then jump out only when the issue truly becomes merchandising, replenishment, or receiving work.",
				internalWorkflowCard("Flow 1", "Edit stock lane", "Jump straight to the lane editor on this nested panel when the task is quantity, threshold, or status correction.", currentPath+"#warehouse-lane-editor"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product editor when the warehouse issue surfaces a summary, details, or SEO problem.", "/app/products/"+page.Product.Slug),
				internalWorkflowCard("Flow 3", "Order more units", "Use the replenishment panel on this route when the fix requires vendor-side inbound, not only local edits.", currentPath+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Return to warehouse filters", "Drop back to the parent warehouse roster with the same route-owned filter context still visible.", parentPath),
			),
			warehouseItemNetworkTable(currentPath, filters, page.Warehouse.ID, page.Network),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{ID: "warehouse-lane-editor"}, inventoryLaneEditorCardWithOptions(page.Item, payload, currentPath)),
			productUpdateFormWithOptions(page.Product, payload, productFormOptions{LockedWarehouseID: page.Warehouse.ID, ReturnWarehouseID: page.Warehouse.ID, IntroLabel: "Edit warehouse item", SubmitLabel: "Save warehouse item"}),
			html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard([]inventoryRow{page.Item}, page.Warehouse.ID, page.Product.Title, payload, true, currentPath)),
			warehouseListCard("Related replenishment orders", orderNodes...),
			productDeleteFormWithOptions(page.Product, payload, productFormOptions{ReturnWarehouseID: page.Warehouse.ID, DeleteLabel: "Delete warehouse item", DeleteCopy: "Deleting this item removes the product and its warehouse inventory from the Atlas demo data. Use this only when the warehouse should stop managing it entirely."}),
		),
	)
}

func warehouseOpsItemContent(payload Payload) ui.Node {
	page := decode[warehouseInventoryItemDetailPage](pageData(payload))
	currentPath := "/app/warehouses/" + page.Warehouse.ID + "/items/" + page.Item.SKU
	filters := page.Filters
	if filters == nil {
		filters = map[string]string{"sort": "updated", "dir": "desc"}
	}
	if strings.TrimSpace(filters["sort"]) == "" {
		filters["sort"] = "updated"
	}
	if strings.TrimSpace(filters["dir"]) == "" {
		filters["dir"] = warehouseItemDefaultDirection(filters["sort"])
	}
	orderNodes := make([]ui.Node, 0, len(page.Orders))
	for _, order := range page.Orders {
		orderNodes = append(orderNodes, html.A(html.Props{Href: "/app/purchase-orders/" + order.ID, Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 transition hover:border-cyan-400/45 hover:bg-slate-950"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(order.ID+" · "+order.VendorName)),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(order.Status+" · ETA "+order.ETA)),
		))
	}
	if len(orderNodes) == 0 {
		orderNodes = append(orderNodes, html.Div(html.Props{Class: "rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-400"}, html.Text("No open replenishment orders are tied to this item in the current warehouse.")))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.08fr)_minmax(24rem,0.88fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
				warehouseBreadcrumbLink{Label: fallback(page.Warehouse.Name, page.Warehouse.ID), Href: "/app/warehouses/" + page.Warehouse.ID},
				warehouseBreadcrumbLink{Label: fallback(page.Product.Title, page.Item.SKU), Href: currentPath, Current: true},
			),
			warehouseFeatureCard(page.Product.Title, page.Product.Summary),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
				warehouseStatCard("Available", fmt.Sprintf("%d units", page.Item.Available)),
				warehouseStatCard("Weekly demand", fmt.Sprintf("%d units", page.Item.WeeklyUnits)),
				warehouseStatCard("Weekly revenue", formatPrice(page.Item.WeeklyRevenue)),
				warehouseStatCard("Order more", fmt.Sprintf("%d units", page.Item.ReorderUnits)),
			),
			warehouseListCard("Market and sales readout",
				warehouseInfoRow("Category", strings.Title(page.Item.Category)),
				warehouseInfoRow("Sell-through", fmt.Sprintf("%d%%", page.Item.SellThrough)),
				warehouseInfoRow("Demand score", fmt.Sprintf("%d / 100", page.Item.DemandScore)),
				warehouseInfoRow("Regional share", fmt.Sprintf("%d%%", page.Item.RegionalShare)),
			),
			internalWorkflowSection("Warehouse item admin flows", "Complete the warehouse item job here, then jump directly into copy, replenishment, or receiving routes without retracing your steps.",
				internalWorkflowCard("Flow 1", "Edit stock lane", "Jump straight to the lane editor on this page when the task is quantity, threshold, or status correction.", currentPath+"#warehouse-lane-editor"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product editor when the warehouse issue surfaces a summary, details, or SEO problem.", "/app/products/"+page.Product.Slug),
				internalWorkflowCard("Flow 3", "Order more units", "Use the replenishment panel on this route when the fix requires vendor-side inbound, not only local edits.", currentPath+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Back to warehouse roster", "Return to the parent warehouse list when you need to move from this item into the next local task.", "/app/warehouses/"+page.Warehouse.ID),
			),
			warehouseItemNetworkTable(currentPath, filters, page.Warehouse.ID, page.Network),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{ID: "warehouse-lane-editor"}, inventoryLaneEditorCardWithOptions(page.Item, payload, currentPath)),
			productUpdateFormWithOptions(page.Product, payload, productFormOptions{LockedWarehouseID: page.Warehouse.ID, ReturnWarehouseID: page.Warehouse.ID, IntroLabel: "Edit warehouse item", SubmitLabel: "Save warehouse item"}),
			html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard([]inventoryRow{page.Item}, page.Warehouse.ID, page.Product.Title, payload, true, currentPath)),
			warehouseListCard("Related replenishment orders", orderNodes...),
			productDeleteFormWithOptions(page.Product, payload, productFormOptions{ReturnWarehouseID: page.Warehouse.ID, DeleteLabel: "Delete warehouse item", DeleteCopy: "Deleting this item removes the product and its warehouse inventory from the Atlas demo data. Use this only when the warehouse should stop managing it entirely."}),
		),
	)
}

func warehouseParentWorkspaceHref(payload Payload, warehouseID string) string {
	values := url.Values{}
	for key, items := range payload.Route.Query {
		if strings.EqualFold(strings.TrimSpace(key), "sort") || strings.EqualFold(strings.TrimSpace(key), "dir") {
			continue
		}
		for _, item := range items {
			values.Add(key, item)
		}
	}
	href := "/app/warehouses/" + warehouseID
	if encoded := values.Encode(); encoded != "" {
		return href + "?" + encoded
	}
	return href
}

func warehouseItemNetworkRowCells(currentWarehouseID string, item inventoryRow) ui.Node {
	badgeClass := warehouseStatusClass(item.Status)
	warehouseLabelClass := "text-xs uppercase tracking-[0.22em] text-slate-400"
	if strings.EqualFold(item.WarehouseID, currentWarehouseID) {
		badgeClass = "inline-flex items-center border border-cyan-400/45 bg-cyan-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-200"
		warehouseLabelClass = "text-xs uppercase tracking-[0.22em] text-cyan-200"
	}
	return ui.Fragment(
		html.Tag("td", html.Props{Class: "px-3 py-3"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(item.WarehouseName, item.WarehouseID))),
					html.Span(html.Props{Class: badgeClass}, html.Text(strings.ReplaceAll(item.Status, "_", " "))),
				),
				html.P(html.Props{Class: warehouseLabelClass}, html.Text(item.WarehouseID+" · "+item.MarketPressure)),
				html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(item.MarketSignal)),
			),
		),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Available))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d days", item.CoverDays))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Inbound))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d / week", item.WeeklyUnits))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d%%", item.RegionalShare))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-300"}, html.Text(item.UpdatedAt)),
	)
}

func warehouseItemNetworkTable(currentPath string, filters map[string]string, currentWarehouseID string, items []inventoryRow) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, html.Tag("tr", html.Props{Class: "border-t border-slate-700/90 align-top"}, warehouseItemNetworkRowCells(currentWarehouseID, item)))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-3 py-4 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 7}}, html.Text("No warehouse lanes are available for this item.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.Div(html.Props{Class: "grid gap-1"}, atlasSectionMeta("Warehouse item table", "Track the current warehouse lane beside the other hubs with the standard Atlas inventory columns plus demand context. Click any column header to reorder the table.")),
		html.Div(html.Props{Class: "overflow-x-auto border border-slate-700 bg-slate-950/70"},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-900"},
						warehouseItemTableHeader(currentPath, filters, "warehouse", "Warehouse"),
						warehouseItemTableHeader(currentPath, filters, "available", "Available"),
						warehouseItemTableHeader(currentPath, filters, "cover", "Cover days"),
						warehouseItemTableHeader(currentPath, filters, "inbound", "Inbound"),
						warehouseItemTableHeader(currentPath, filters, "demand", "Demand"),
						warehouseItemTableHeader(currentPath, filters, "share", "Regional share"),
						warehouseItemTableHeader(currentPath, filters, "updated", "Updated"),
					),
				),
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

type warehouseBreadcrumbLink struct {
	Label   string
	Href    string
	Current bool
}

func warehouseBreadcrumbBar(links ...warehouseBreadcrumbLink) ui.Node {
	nodes := make([]ui.Node, 0, len(links)*2)
	for index, link := range links {
		label := fallback(strings.TrimSpace(link.Label), "Route")
		if index > 0 {
			nodes = append(nodes, html.Span(html.Props{Class: "text-slate-600"}, html.Text("/")))
		}
		if link.Current || strings.TrimSpace(link.Href) == "" {
			nodes = append(nodes, html.Span(html.Props{Class: "text-slate-100"}, html.Text(label)))
			continue
		}
		nodes = append(nodes, html.A(html.Props{Href: link.Href, Class: "transition hover:text-cyan-100"}, html.Text(label)))
	}
	return html.Div(html.Props{Class: "flex flex-wrap items-center gap-2 border border-slate-700 bg-slate-950/80 px-3 py-2 text-[0.72rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, nodes...)
}

func warehouseItemTableHeader(currentPath string, filters map[string]string, sortKey string, label string) ui.Node {
	active := strings.EqualFold(strings.TrimSpace(filters["sort"]), sortKey)
	direction := warehouseItemDefaultDirection(sortKey)
	if active {
		if strings.EqualFold(strings.TrimSpace(filters["dir"]), "asc") {
			direction = "desc"
		} else {
			direction = "asc"
		}
	}
	values := url.Values{}
	values.Set("sort", sortKey)
	values.Set("dir", direction)
	indicator := ""
	linkClass := "inline-flex items-center gap-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500 transition hover:text-cyan-200"
	if active {
		linkClass = "inline-flex items-center gap-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-cyan-200 transition hover:text-cyan-100"
		if strings.EqualFold(strings.TrimSpace(filters["dir"]), "asc") {
			indicator = "↑"
		} else {
			indicator = "↓"
		}
	}
	children := []ui.Node{html.Text(label)}
	if indicator != "" {
		children = append(children, html.Span(html.Props{Class: "text-cyan-200"}, html.Text(indicator)))
	}
	return html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 last:border-r-0"}, html.A(html.Props{Href: currentPath + "?" + values.Encode(), Target: "_self", Class: linkClass}, children...))
}

func warehouseItemDefaultDirection(sortKey string) string {
	switch strings.TrimSpace(strings.ToLower(sortKey)) {
	case "warehouse", "cover":
		return "asc"
	default:
		return "desc"
	}
}

func filterInventoryRows(items []inventoryRow, filters atlasListFilterState) []inventoryRow {
	needle := atlasNormalizedFilterValue(filters.Query)
	warehouse := atlasNormalizedFilterValue(filters.Warehouse)
	status := atlasNormalizedFilterValue(filters.Status)
	filtered := make([]inventoryRow, 0, len(items))
	for _, item := range items {
		if needle != "" {
			haystack := atlasNormalizedFilterValue(item.Title + " " + item.SKU + " " + item.Category + " " + item.WarehouseName + " " + item.WarehouseID + " " + item.MarketSignal + " " + item.MarketPressure)
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		if warehouse != "" && warehouse != "all" && !strings.EqualFold(strings.TrimSpace(item.WarehouseID), warehouse) {
			continue
		}
		if status != "" && status != "all" && !strings.EqualFold(strings.TrimSpace(item.Status), status) {
			continue
		}
		filtered = append(filtered, item)
	}
	switch atlasNormalizedFilterValue(filters.Sort) {
	case "inbound":
		sort.SliceStable(filtered, func(left, right int) bool {
			if filtered[left].Inbound == filtered[right].Inbound {
				return filtered[left].UpdatedAt > filtered[right].UpdatedAt
			}
			return filtered[left].Inbound > filtered[right].Inbound
		})
	default:
		sort.SliceStable(filtered, func(left, right int) bool {
			return filtered[left].UpdatedAt > filtered[right].UpdatedAt
		})
	}
	return filtered
}

func filterWarehouseInventoryRows(items []inventoryRow, filters atlasListFilterState) []inventoryRow {
	needle := atlasNormalizedFilterValue(filters.Query)
	status := atlasNormalizedFilterValue(filters.Status)
	filtered := make([]inventoryRow, 0, len(items))
	for _, item := range items {
		if needle != "" {
			haystack := atlasNormalizedFilterValue(item.Title + " " + item.SKU + " " + item.Category + " " + item.MarketSignal + " " + item.MarketPressure)
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		if status != "" && status != "all" && !strings.EqualFold(strings.TrimSpace(item.Status), status) {
			continue
		}
		filtered = append(filtered, item)
	}
	switch atlasNormalizedFilterValue(filters.Sort) {
	case "available":
		sort.SliceStable(filtered, func(left, right int) bool {
			if filtered[left].Available == filtered[right].Available {
				return filtered[left].UpdatedAt > filtered[right].UpdatedAt
			}
			return filtered[left].Available < filtered[right].Available
		})
	case "demand":
		sort.SliceStable(filtered, func(left, right int) bool {
			if filtered[left].WeeklyUnits == filtered[right].WeeklyUnits {
				return filtered[left].UpdatedAt > filtered[right].UpdatedAt
			}
			return filtered[left].WeeklyUnits > filtered[right].WeeklyUnits
		})
	case "revenue":
		sort.SliceStable(filtered, func(left, right int) bool {
			if filtered[left].WeeklyRevenue == filtered[right].WeeklyRevenue {
				return filtered[left].UpdatedAt > filtered[right].UpdatedAt
			}
			return filtered[left].WeeklyRevenue > filtered[right].WeeklyRevenue
		})
	default:
		sort.SliceStable(filtered, func(left, right int) bool {
			return filtered[left].UpdatedAt > filtered[right].UpdatedAt
		})
	}
	return filtered
}

func inventoryCMSFilterForm(form ui.Form[atlasListFilterState], syncing bool, submit ui.Handler, transition atlasTransition) ui.Node {
	value := form.Get()
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Filter model")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep query, warehouse, and lane-status pivots visible in one row so operators can tighten the queue without losing deep-linkable state.")),
		),
		html.Form(html.Props{Action: "/app/inventory", Method: "get", OnSubmit: submit, Class: "grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.7fr))_auto] lg:items-end"},
			cmsTransitionBoundTextInput("q", "Search inventory", value.Query, "Query", form, transition),
			cmsTransitionBoundSelectInput("warehouse", "Warehouse", value.Warehouse, "Warehouse", append([]optionItem{{"all", "All warehouses"}}, warehouseOptions()...), form, transition),
			cmsTransitionBoundSelectInput("status", "Lane status", value.Status, "Status", append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...), form, transition),
			cmsTransitionBoundSelectInput("sort", "Sort", value.Sort, "Sort", []optionItem{{"updated", "Updated recently"}, {"inbound", "Highest inbound"}}, form, transition),
			html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(atlasFilterSubmitLabel(syncing, "Filter"))),
		),
	)
}

func inventoryLaneEditorCard(row inventoryRow, payload Payload) ui.Node {
	return inventoryLaneEditorCardWithOptions(row, payload, "")
}

func inventoryLaneEditorCardWithOptions(row inventoryRow, payload Payload, returnPath string) ui.Node {
	warehouseLabel := fallback(row.WarehouseName, row.WarehouseID)
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(inventoryLaneEditorFormState{
			WarehouseID:  row.WarehouseID,
			OnHand:       fmt.Sprintf("%d", row.OnHand),
			Reserved:     fmt.Sprintf("%d", row.Reserved),
			Damaged:      fmt.Sprintf("%d", row.Damaged),
			Inbound:      fmt.Sprintf("%d", row.Inbound),
			ReorderPoint: fmt.Sprintf("%d", row.ReorderPoint),
			SafetyStock:  fmt.Sprintf("%d", row.SafetyStock),
			Status:       row.Status,
			ReturnPath:   returnPath,
		})
		value := form.Get()
		children := []ui.Node{
			html.Input(html.Props{Type: "hidden", Name: "warehouse_id", Value: value.WarehouseID}),
			inventoryBoundNumberField("on_hand", "On hand", value.OnHand, "OnHand", form),
			inventoryBoundNumberField("reserved", "Reserved", value.Reserved, "Reserved", form),
			inventoryBoundNumberField("damaged", "Damaged", value.Damaged, "Damaged", form),
			inventoryBoundNumberField("inbound", "Inbound", value.Inbound, "Inbound", form),
			inventoryBoundNumberField("reorder_point", "Reorder point", value.ReorderPoint, "ReorderPoint", form),
			inventoryBoundNumberField("safety_stock", "Safety stock", value.SafetyStock, "SafetyStock", form),
			inventoryBoundSelectField("status", "Lane status", value.Status, "Status", inventoryStatusOptions(), form),
			html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Save lane")),
		}
		if strings.TrimSpace(value.ReturnPath) != "" {
			children = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: value.ReturnPath})}, children...)
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-1"},
					html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.2em] text-white"}, html.Text(warehouseLabel)),
					html.P(html.Props{Class: "text-[0.7rem] uppercase tracking-[0.24em] text-slate-500"}, html.Text(row.WarehouseID+" | updated "+row.UpdatedAt)),
				),
				html.Span(html.Props{Class: warehouseStatusClass(value.Status)}, html.Text(strings.ReplaceAll(value.Status, "_", " "))),
			),
			html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-300 md:grid-cols-3"},
				warehouseInfoRow("Available", fmt.Sprintf("%d units", row.Available)),
				warehouseInfoRow("Inbound", fmt.Sprintf("%d units", row.Inbound)),
				warehouseInfoRow("Cover", fmt.Sprintf("%d days", row.CoverDays)),
			),
			html.Form(html.Props{Action: "/api/app/inventory/" + row.SKU + "/update", Method: "post", Class: "grid gap-4 md:grid-cols-2"}, prependCSRFToken(payload.CSRF, children...)...),
		)
	})
}

/*
	 children := []ui.Node{
			html.Input(html.Props{Type: "hidden", Name: "warehouse_id", Value: row.WarehouseID}),
			inventoryNumberField("on_hand", "On hand", fmt.Sprintf("%d", row.OnHand)),
			inventoryNumberField("reserved", "Reserved", fmt.Sprintf("%d", row.Reserved)),
			inventoryNumberField("damaged", "Damaged", fmt.Sprintf("%d", row.Damaged)),
			inventoryNumberField("inbound", "Inbound", fmt.Sprintf("%d", row.Inbound)),
			inventoryNumberField("reorder_point", "Reorder point", fmt.Sprintf("%d", row.ReorderPoint)),
			inventoryNumberField("safety_stock", "Safety stock", fmt.Sprintf("%d", row.SafetyStock)),
			inventorySelectField("status", "Lane status", row.Status, inventoryStatusOptions()),
			html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Save lane")),
		}
		if strings.TrimSpace(returnPath) != "" {
			children = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: returnPath})}, children...)
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-1"},
					html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.2em] text-white"}, html.Text(warehouseLabel)),
					html.P(html.Props{Class: "text-[0.7rem] uppercase tracking-[0.24em] text-slate-500"}, html.Text(row.WarehouseID+" | updated "+row.UpdatedAt)),
				),
				html.Span(html.Props{Class: warehouseStatusClass(row.Status)}, html.Text(strings.ReplaceAll(row.Status, "_", " "))),
			),
			html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-300 md:grid-cols-3"},
				warehouseInfoRow("Available", fmt.Sprintf("%d units", row.Available)),
				warehouseInfoRow("Inbound", fmt.Sprintf("%d units", row.Inbound)),
				warehouseInfoRow("Cover", fmt.Sprintf("%d days", row.CoverDays)),
			),
			html.Form(html.Props{Action: "/api/app/inventory/" + row.SKU + "/update", Method: "post", Class: "grid gap-4 md:grid-cols-2"}, prependCSRFToken(payload.CSRF, children...)...),
		)
	}
*/
func orderInventoryModalCard(rows []inventoryRow, defaultWarehouse string, title string, payload Payload, skuScoped bool, returnPath string) ui.Node {
	if len(rows) == 0 {
		return nil
	}
	productOptions := inventoryProductOptions(rows)
	warehouseOptionsList := inventoryWarehouseLaneOptions(rows)
	defaultSKU := rows[0].SKU
	if defaultWarehouse == "" {
		defaultWarehouse = rows[0].WarehouseID
	}
	modalID := inventoryModalID(defaultSKU, defaultWarehouse)
	buttonLabel := "Order new items"
	description := "Create a purchase order from the active management surface and push inbound units onto the selected warehouse lane immediately."
	if skuScoped {
		buttonLabel = "Order replenishment"
		description = "Launch a replenishment order for this SKU without leaving the inventory editor."
	}
	return ui.CreateElement(func() ui.Node {
		workflow := ui.UseReducer(reducePurchaseOrderWorkflowState, defaultPurchaseOrderWorkflowState(defaultSKU, defaultWarehouse, returnPath))
		open := ui.UseState(false)
		openModal := ui.UseEvent(func(ui.Event) {
			open.Set(true)
		})
		closeModal := ui.UseEvent(func(ui.Event) {
			open.Set(false)
		})
		value := workflow.Get()
		fields := []ui.Node{
			purchaseOrderWorkflowSummaryCard(value),
			inventoryReducerTextField("vendor_name", "Vendor", value.VendorName, workflow),
			inventoryReducerNumberField("quantity", "Quantity", value.Quantity, workflow),
			inventoryReducerTextField("eta", "ETA", value.ETA, workflow),
			inventoryReducerTextField("priority_note", "Priority note", value.PriorityNote, workflow),
			inventoryReducerSelectField("status", "Order status", value.Status, []optionItem{{"draft", "Draft"}, {"submitted", "Submitted"}, {"approved", "Approved"}, {"on_hold", "On hold"}}, workflow),
		}
		if skuScoped || len(productOptions) == 1 {
			fields = append([]ui.Node{
				html.Input(html.Props{Type: "hidden", Name: "product_sku", Value: value.ProductSKU}),
				html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm text-slate-300"}, html.Text("Item: "+title+" · "+value.ProductSKU)),
			}, fields...)
		} else {
			fields = append([]ui.Node{inventoryReducerSelectField("product_sku", "Inventory item", value.ProductSKU, productOptions, workflow)}, fields...)
		}
		if len(warehouseOptionsList) == 1 {
			fields = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "warehouse_id", Value: value.WarehouseID})}, fields...)
		} else {
			fields = append([]ui.Node{inventoryReducerSelectField("warehouse_id", "Warehouse", value.WarehouseID, warehouseOptionsList, workflow)}, fields...)
		}
		if strings.TrimSpace(value.ReturnPath) != "" {
			fields = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: value.ReturnPath})}, fields...)
		}
		formChildren := prependCSRFToken(payload.CSRF, fields...)
		formChildren = append(formChildren,
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-end gap-3 md:col-span-2"},
				html.Button(html.Props{Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: closeModal}, html.Text("Cancel")),
				html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Create order")),
			),
		)
		modalSurfaceID := modalID + "-surface"
		modalCloseID := modalID + "-close"
		useAtlasFocusContainment(open.Get(), "#"+modalSurfaceID, "#"+modalCloseID)
		children := []ui.Node{
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Replenishment")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(description)),
			html.Button(html.Props{Type: "button", Class: "w-fit border border-cyan-400/45 bg-cyan-500/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.18em] text-cyan-200 transition hover:border-cyan-300 hover:bg-cyan-500/16 hover:text-cyan-100", OnClick: openModal}, html.Text(buttonLabel)),
		}
		if open.Get() {
			children = append(children, html.Div(html.Props{Class: "fixed inset-0 z-50 flex items-center justify-center bg-slate-950/75 p-4"},
				html.Button(html.Props{Type: "button", Class: "absolute inset-0", OnClick: closeModal, Raw: map[string]interface{}{"aria-label": "Close replenishment dialog"}}),
				html.Div(html.Props{
					ID:    modalSurfaceID,
					Class: "relative z-10 grid w-[min(92vw,40rem)] gap-4 border border-slate-700 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]",
					Raw: map[string]interface{}{
						"role":       "dialog",
						"aria-modal": "true",
						"tabindex":   "-1",
					},
				},
					html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
						html.Div(html.Props{Class: "grid gap-2"},
							html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Order new items")),
							html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(title)),
							html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(description)),
						),
						html.Button(html.Props{ID: modalCloseID, Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: closeModal}, html.Text("Close")),
					),
					html.Form(html.Props{Action: "/api/app/purchase-orders", Method: "post", Class: "grid gap-4 md:grid-cols-2"}, formChildren...),
				),
			))
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"}, children...)
	})
}

/* fields := []ui.Node{
	inventoryTextField("vendor_name", "Vendor", "Northline Fabrication"),
	inventoryNumberField("quantity", "Quantity", "12"),
	inventoryTextField("eta", "ETA", "Tue 10:00"),
	inventoryTextField("priority_note", "Priority note", "Replenish active warehouse lane"),
	inventorySelectField("status", "Order status", "submitted", []optionItem{{"draft", "Draft"}, {"submitted", "Submitted"}, {"approved", "Approved"}, {"on_hold", "On hold"}}),
}
if skuScoped || len(productOptions) == 1 {
	fields = append([]ui.Node{
		html.Input(html.Props{Type: "hidden", Name: "product_sku", Value: defaultSKU}),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm text-slate-300"}, html.Text("Item: "+title+" · "+defaultSKU)),
	}, fields...)
} else {
	fields = append([]ui.Node{inventorySelectField("product_sku", "Inventory item", defaultSKU, productOptions)}, fields...)
}
if len(warehouseOptionsList) == 1 {
	fields = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "warehouse_id", Value: defaultWarehouse})}, fields...)
} else {
	fields = append([]ui.Node{inventorySelectField("warehouse_id", "Warehouse", defaultWarehouse, warehouseOptionsList)}, fields...)
}
if strings.TrimSpace(returnPath) != "" {
	fields = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: returnPath})}, fields...)
}
formChildren := prependCSRFToken(payload.CSRF, fields...)
formChildren = append(formChildren,
	html.Div(html.Props{Class: "flex flex-wrap items-center justify-end gap-3 md:col-span-2"},
		html.Label(html.Props{For: modalID, Class: warehouseSecondaryButtonClass()}, html.Text("Cancel")),
		html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Create order")),
	),
)
return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"},
	html.Input(html.Props{Type: "checkbox", ID: modalID, Class: "peer hidden"}),
	html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Replenishment")),
	html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(description)),
	html.Label(html.Props{For: modalID, Class: "w-fit cursor-pointer border border-cyan-400/45 bg-cyan-500/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.18em] text-cyan-200 transition hover:border-cyan-300 hover:bg-cyan-500/16 hover:text-cyan-100"}, html.Text(buttonLabel)),
	html.Div(html.Props{Class: "pointer-events-none invisible fixed inset-0 z-50 flex items-center justify-center bg-slate-950/75 p-4 opacity-0 transition peer-checked:pointer-events-auto peer-checked:visible peer-checked:opacity-100"},
		html.Label(html.Props{For: modalID, Class: "absolute inset-0 cursor-pointer"}),
		html.Div(html.Props{Class: "relative z-10 grid w-[min(92vw,40rem)] gap-4 border border-slate-700 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Order new items")),
					html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(title)),
					html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(description)),
				),
				html.Label(html.Props{For: modalID, Class: warehouseSecondaryButtonClass()}, html.Text("Close")),
			),
			html.Form(html.Props{Action: "/api/app/purchase-orders", Method: "post", Class: "grid gap-4 md:grid-cols-2"}, formChildren...),
		),
	),
) */

func purchaseOrderWorkflowSummaryCard(state purchaseOrderWorkflowState) ui.Node {
	stageLabel := "Ready to submit"
	stageCopy := "Vendor, quantity, and warehouse context are aligned for a replenishment order."
	switch state.Stage {
	case "inbound-confirmed":
		stageLabel = "Inbound confirmed"
		stageCopy = "This draft is already approved, so Atlas treats it as committed inbound inventory."
	case "blocked":
		stageLabel = "Blocked"
		stageCopy = "The order is on hold and should not move until the note explains the blocker."
	case "draft-needs-brief":
		stageLabel = "Needs brief"
		stageCopy = "Add a priority note before this stays in draft so the next operator has context."
	case "draft-review":
		stageLabel = "Draft review"
		stageCopy = "The order is still a draft, but it already has enough context for review."
	case "needs-quantity":
		stageLabel = "Needs quantity"
		stageCopy = "Atlas will not treat this replenishment as actionable until quantity is filled in."
	}
	edited := "Workflow staged from the current replenishment defaults."
	if strings.TrimSpace(state.LastEditedField) != "" {
		edited = "Last updated field: " + strings.ReplaceAll(state.LastEditedField, "_", " ") + "."
	}
	return html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100 md:col-span-2"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(stageLabel)),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(stageCopy)),
		html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.2em] text-cyan-200/75"}, html.Text(edited)),
	)
}

func inventoryReducerTextField(name, label, value string, workflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{
			ID: id, Name: name, Value: value, Class: warehouseInputClass(),
			OnInput: ui.UseEvent(func(event ui.InputEvent) {
				workflow.Dispatch(purchaseOrderWorkflowAction{Field: name, Value: event.GetValue()})
			}),
			Raw: map[string]interface{}{"aria-labelledby": id + "-label"},
		}),
	)
}

func inventoryReducerNumberField(name, label, value string, workflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{
			ID: id, Type: "number", Name: name, Value: value, Class: warehouseInputClass(),
			OnInput: ui.UseEvent(func(event ui.InputEvent) {
				workflow.Dispatch(purchaseOrderWorkflowAction{Field: name, Value: event.GetValue()})
			}),
			Raw: map[string]interface{}{"aria-labelledby": id + "-label"},
		}),
	)
}

func inventoryReducerSelectField(name, label, value string, options []optionItem, workflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	id := ui.UseId()
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value))}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Select(html.Props{
			ID: id, Name: name, Class: warehouseInputClass(),
			OnChange: ui.UseEvent(func(event ui.ChangeEvent) {
				workflow.Dispatch(purchaseOrderWorkflowAction{Field: name, Value: event.GetValue()})
			}),
			Raw: map[string]interface{}{"aria-labelledby": id + "-label"},
		}, children...),
	)
}

func warehouseInventoryFilterForm(warehouseID string, form ui.Form[atlasListFilterState], syncing bool, submit ui.Handler, transition atlasTransition) ui.Node {
	value := form.Get()
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Facility filter model")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep warehouse item search, lane status, and sort order visible in one card so the facility roster stays URL-backed and easy to narrow under hydration.")),
		),
		html.Form(html.Props{Action: "/app/warehouses/" + warehouseID, Method: "get", OnSubmit: submit, Class: "grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_repeat(2,minmax(0,0.75fr))_auto] lg:items-end"},
			cmsTransitionBoundTextInput("q", "Search items", value.Query, "Query", form, transition),
			cmsTransitionBoundSelectInput("status", "Lane status", value.Status, "Status", append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...), form, transition),
			cmsTransitionBoundSelectInput("sort", "Sort", value.Sort, "Sort", []optionItem{{"updated", "Updated"}, {"available", "Lowest available"}, {"demand", "Highest demand"}, {"revenue", "Highest revenue"}}, form, transition),
			html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text(atlasFilterSubmitLabel(syncing, "Apply"))),
		),
	)
}

func marketPressureClass(label string) string {
	switch strings.TrimSpace(strings.ToLower(label)) {
	case "hot market":
		return "inline-flex items-center border border-rose-400/45 bg-rose-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-rose-100"
	case "growing demand":
		return "inline-flex items-center border border-amber-400/45 bg-amber-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-amber-100"
	case "softening":
		return "inline-flex items-center border border-slate-500/45 bg-slate-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-200"
	default:
		return "inline-flex items-center border border-emerald-400/45 bg-emerald-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-emerald-100"
	}
}

func inventoryStatusOptions() []optionItem {
	return []optionItem{{"balanced", "Balanced"}, {"promise_risk", "Promise risk"}, {"critical", "Critical"}, {"recovery", "Recovery"}}
}

func inventoryProductOptions(rows []inventoryRow) []optionItem {
	seen := map[string]bool{}
	options := make([]optionItem, 0, len(rows))
	for _, row := range rows {
		if seen[row.SKU] {
			continue
		}
		seen[row.SKU] = true
		options = append(options, optionItem{Value: row.SKU, Label: row.Title + " · " + row.SKU})
	}
	return options
}

func inventoryWarehouseLaneOptions(rows []inventoryRow) []optionItem {
	seen := map[string]bool{}
	options := make([]optionItem, 0, len(rows))
	for _, row := range rows {
		if seen[row.WarehouseID] {
			continue
		}
		seen[row.WarehouseID] = true
		options = append(options, optionItem{Value: row.WarehouseID, Label: inventoryWarehouseLabel(row.WarehouseID, rows)})
	}
	return options
}

func inventoryWarehouseLabel(warehouseID string, rows []inventoryRow) string {
	for _, row := range rows {
		if row.WarehouseID == warehouseID {
			return fallback(row.WarehouseName, row.WarehouseID)
		}
	}
	return warehouseID
}

func inventoryTextField(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: warehouseInputClass()}),
	)
}

func inventoryNumberField(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Type: "number", Name: name, Value: value, Class: warehouseInputClass()}),
	)
}

func inventoryBoundNumberField[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Type: "number", Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func inventorySelectField(name, label, value string, options []optionItem) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value))}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Select(html.Props{Name: name, Class: warehouseInputClass()}, children...),
	)
}

func inventoryBoundSelectField[T any](name, label, value, field string, options []optionItem, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value))}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Select(html.Props{ID: id, Name: name, OnChange: ui.UseEvent(func(event ui.ChangeEvent) { form.SetField(field, event.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, children...),
	)
}

func warehouseFeatureCard(title, copy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-2 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Warehouse operations")),
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(copy)),
	)
}

func warehouseStatCard(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-sm border border-slate-700 bg-slate-950/80 p-4"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.26em] text-slate-500"}, html.Text(label)),
		html.P(html.Props{Class: "mt-2 text-lg font-semibold text-white"}, html.Text(value)),
	)
}

func warehouseListCard(title string, children ...ui.Node) ui.Node {
	if len(children) == 0 {
		children = []ui.Node{html.P(html.Props{Class: "text-sm text-slate-500"}, html.Text("No records yet."))}
	}
	content := []ui.Node{html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text(title))}
	content = append(content, children...)
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"}, content...)
}

func warehouseInfoRow(primary string, secondary string) ui.Node {
	return html.Div(html.Props{Class: "border border-slate-700 bg-slate-950/75 px-3 py-3"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.22em] text-slate-500"}, html.Text(primary)),
		html.P(html.Props{Class: "mt-1 text-sm text-slate-200"}, html.Text(secondary)),
	)
}

func warehousePrimaryButtonClass() string {
	return "border border-cyan-400/55 bg-cyan-500/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.18em] text-cyan-200 transition hover:border-cyan-300 hover:bg-cyan-500/16 hover:text-cyan-100"
}

func warehouseSecondaryButtonClass() string {
	return "cursor-pointer border border-slate-600 px-4 py-2 text-sm font-semibold uppercase tracking-[0.18em] text-slate-200 transition hover:border-slate-400 hover:text-white"
}

func warehouseInputClass() string {
	return "border border-slate-700 bg-slate-950/85 px-3 py-2 text-slate-100"
}

func warehouseStatusClass(status string) string {
	base := "inline-flex items-center border px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em]"
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "balanced", "in_stock", "healthy", "approved", "available":
		return base + " border-emerald-400/45 bg-emerald-500/10 text-emerald-100"
	case "promise_risk", "low_stock", "pending", "submitted", "in_review":
		return base + " border-amber-400/45 bg-amber-500/10 text-amber-100"
	case "critical":
		return base + " border-rose-400/45 bg-rose-500/10 text-rose-100"
	default:
		return base + " border-slate-500/45 bg-slate-500/10 text-slate-200"
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func inventoryPrimaryWarehouseHref(rows []inventoryRow) string {
	if len(rows) == 0 {
		return "/app/inventory"
	}
	primary := rows[0]
	return "/app/warehouses/" + primary.WarehouseID + "/items/" + primary.SKU
}

func inventoryModalID(sku string, warehouseID string) string {
	replacer := strings.NewReplacer("/", "-", " ", "-", ".", "-", ":", "-", "_", "-")
	return "inventory-order-" + replacer.Replace(strings.ToLower(strings.TrimSpace(sku))) + "-" + replacer.Replace(strings.ToLower(strings.TrimSpace(warehouseID)))
}
