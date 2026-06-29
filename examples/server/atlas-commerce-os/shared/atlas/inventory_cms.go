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

func defaultPurchaseOrderWorkflowState(parseDefaultSKU, parseDefaultWarehouse, parseReturnPath string) purchaseOrderWorkflowState {
	return normalizePurchaseOrderWorkflowState(purchaseOrderWorkflowState{
		ProductSKU:   parseDefaultSKU,
		WarehouseID:  parseDefaultWarehouse,
		VendorName:   "Northline Fabrication",
		Quantity:     "12",
		ETA:          "Tue 10:00",
		PriorityNote: "Replenish active warehouse lane",
		Status:       "submitted",
		ReturnPath:   parseReturnPath,
	})
}

func reducePurchaseOrderWorkflowState(parseState purchaseOrderWorkflowState, parseAction purchaseOrderWorkflowAction) purchaseOrderWorkflowState {
	switch parseAction.Field {
	case "product_sku":
		parseState.ProductSKU = parseAction.Value
	case "warehouse_id":
		parseState.WarehouseID = parseAction.Value
	case "vendor_name":
		parseState.VendorName = parseAction.Value
	case "quantity":
		parseState.Quantity = parseAction.Value
	case "eta":
		parseState.ETA = parseAction.Value
	case "priority_note":
		parseState.PriorityNote = parseAction.Value
	case "status":
		parseState.Status = parseAction.Value
	case "return_path":
		parseState.ReturnPath = parseAction.Value
	}
	parseState.LastEditedField = parseAction.Field
	return normalizePurchaseOrderWorkflowState(parseState)
}

func normalizePurchaseOrderWorkflowState(parseState purchaseOrderWorkflowState) purchaseOrderWorkflowState {
	parseStatus := strings.TrimSpace(strings.ToLower(parseState.Status))
	parseQuantity := strings.TrimSpace(parseState.Quantity)
	parsePriority := strings.TrimSpace(parseState.PriorityNote)
	switch {
	case parseStatus == "approved":
		parseState.Stage = "inbound-confirmed"
	case parseStatus == "on_hold":
		parseState.Stage = "blocked"
	case parseStatus == "draft" && parsePriority == "":
		parseState.Stage = "draft-needs-brief"
	case parseStatus == "draft":
		parseState.Stage = "draft-review"
	case parseQuantity == "" || parseQuantity == "0":
		parseState.Stage = "needs-quantity"
	default:
		parseState.Stage = "ready-to-submit"
	}
	return parseState
}

func inventoryWorkspaceSnapshotFromRows(parseRows []inventoryRow) inventoryWorkspaceSnapshot {
	return inventoryWorkspaceSnapshot{
		Summaries: inventorySummaryCards(parseRows),
		Rollup:    inventoryRollupFromRows(parseRows),
	}
}

func warehouseDetailWorkspaceSnapshotFromPage(parsePage warehouseInventoryDetailPage) warehouseDetailWorkspaceSnapshot {
	parseSnapshot := warehouseDetailWorkspaceSnapshot{}
	for _, parseItem := range parsePage.Inventory {
		parseSnapshot.TotalDemand += parseItem.WeeklyUnits
		parseSnapshot.TotalRevenue += parseItem.WeeklyRevenue
		if parseItem.ReorderUnits > 0 || strings.EqualFold(parseItem.MarketPressure, "hot market") {
			parseSnapshot.UrgentCount++
		}
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), "critical") || strings.EqualFold(strings.TrimSpace(parseItem.Status), "promise_risk") {
			parseSnapshot.RiskLanes++
		}
	}
	return parseSnapshot
}

func inventoryCMSContent(parsePayload Payload) ui.Node {
	parsePage := decode[inventoryCMSPage](pageData(parsePayload))
	return ui.CreateElement(func() ui.Node {
		parseSearch := useAtlasSearchParams()
		parseShellState := currentRouteWorkspaceState(parsePayload)
		parseInitial := atlasMapFilterState(parsePage.Filters)
		parseForm := ui.UseForm(parseInitial)
		parseTransition := useAtlasTransition()
		useAtlasEffect(func() func() {
			if !atlasSameListFilterState(parseForm.Get(), parseInitial) {
				parseForm.Reset(parseInitial)
			}
			return nil
		}, parseInitial)
		parseValue := parseForm.Get()
		parseDeferred := ui.UseDeferredValue(parseValue)
		parseDebounced := ui.UseDebounced(parseValue, atlasFilterSyncDelay)
		useAtlasEffect(func() func() {
			parseNext := atlasBuildListFilterQuery(parseSearch.Values(), parseDebounced.Get())
			if parseNext.Encode() != parseSearch.Values().Encode() {
				parseSearch.ReplaceAll(parseNext)
			}
			return nil
		}, parseDebounced.Get())
		parseSubmit := ui.UseEvent(func(parseEvent ui.FormEvent) {
			parseEvent.PreventDefault()
			parseSearch.ReplaceAll(atlasBuildListFilterQuery(parseSearch.Values(), parseForm.Get()))
		})
		parseWorkspace := inventoryWorkspaceSnapshotFromRows(filterInventoryRows(parsePage.Items, parseDeferred))
		parseViewNodes := make([]ui.Node, 0, len(parsePayload.SavedViews))
		for _, parseSaved := range parsePayload.SavedViews {
			parseCurrentSaved := parseSaved
			parseNextState := atlasSavedViewFilterState(parseCurrentSaved)
			parseClassName := "grid gap-2 border px-3 py-3 text-left transition"
			if atlasSameListFilterState(parseValue, parseNextState) {
				parseClassName += " border-cyan-400/45 bg-cyan-500/10"
			} else {
				parseClassName += " border-slate-700 bg-slate-950/80 hover:border-cyan-400/45"
			}
			parseViewNodes = append(parseViewNodes, html.Button(html.Props{
				Type:  "button",
				Class: parseClassName,
				OnClick: ui.UseEvent(func() {
					parseTransition.Start(func() {
						parseForm.Set(parseNextState)
						parseSearch.ReplaceAll(atlasBuildListFilterQuery(parseSearch.Values(), parseNextState))
					})
				}),
			},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseSaved.Name)),
				html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseSaved.Scope+" | "+parseSaved.SortKey+"/"+parseSaved.SortDirection)),
			))
		}
		if len(parseViewNodes) == 0 {
			parseViewNodes = append(parseViewNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No saved inventory views yet.")))
		} else if parseTransition.Pending() {
			parseViewNodes = append(parseViewNodes, html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-cyan-200"}, html.Text("Applying the selected saved view without blocking the triage workspace.")))
		}
		return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.26fr)_minmax(22rem,0.74fr)] xl:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				inventoryCMSSummaryBand(parsePage, parseWorkspace),
				inventoryCMSFilterForm(parseForm, parseDebounced.Pending() || parseTransition.Pending(), parseSubmit, parseTransition),
				inventoryCMSActionCluster(),
				inventoryTriageBand(parseWorkspace),
				inventoryQueueTable(parseWorkspace.Summaries),
			),
			html.Div(html.Props{Class: "grid gap-5"},
				inventoryOperationsRail(parseShellState),
				inventorySavedViewsRail(parseViewNodes),
				savedViewForm(parsePayload),
			),
		)
	})
}

func inventoryCMSSummaryBand(parsePage inventoryCMSPage, parseWorkspace inventoryWorkspaceSnapshot) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Inventory triage shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("Keep the inventory route focused on operator triage: visible stock pressure, fast filter pivots, and direct SKU handoffs into lane-level correction without detouring through decorative route explainer cards.")),
		),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Visible SKUs", fmt.Sprintf("%d tracked", len(parseWorkspace.Summaries))),
			statCard("Risk lanes", fmt.Sprintf("%d flagged", parseWorkspace.Rollup.RiskLanes)),
			statCard("Inbound units", fmt.Sprintf("%d queued", parseWorkspace.Rollup.InboundUnits)),
			statCard("Available units", fmt.Sprintf("%d ready", parseWorkspace.Rollup.TotalAvailable)),
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

func inventoryActionCard(parseTitle, parseCopy, parseHref string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: "grid gap-3 " + internalAccentSurfaceClass() + " p-4 transition hover:border-cyan-300/55 hover:bg-[linear-gradient(180deg,rgba(10,24,42,0.96),rgba(7,14,26,0.99))]"},
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Open workflow")),
	)
}

func inventoryTriageBand(parseWorkspace inventoryWorkspaceSnapshot) ui.Node {
	parseRollup := parseWorkspace.Rollup
	parseItems := []struct {
		label string
		value string
		copy  string
		href  string
	}{
		{
			label: "Critical lanes",
			value: fmt.Sprintf("%d", parseRollup.CriticalLanes),
			copy:  "Immediate lane corrections before promise failure.",
			href:  "/app/inventory?status=critical",
		},
		{
			label: "Promise risk",
			value: fmt.Sprintf("%d", parseRollup.PromiseRiskLanes),
			copy:  "SKUs likely to slip without attention.",
			href:  "/app/inventory?status=promise_risk",
		},
		{
			label: "Inbound pending",
			value: fmt.Sprintf("%d", parseRollup.SKUsWithInbound),
			copy:  "Items with open inbound already on the way.",
			href:  "/app/receiving",
		},
		{
			label: "Reorder now",
			value: fmt.Sprintf("%d", parseRollup.ReorderLanes),
			copy:  "Lanes already signaling vendor replenishment.",
			href:  "/app/purchase-orders",
		},
	}
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.A(html.Props{Href: parseItem.href, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text(parseItem.label)),
			html.P(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(parseItem.value)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.copy)),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Triage summary band")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The lead inventory counts stay clickable so operators can pivot from a high-level pressure readout into the exact queue that needs intervention.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2 xl:grid-cols-4"}, parseNodes...),
	)
}

func atlasSectionMeta(parseEyebrow, parseCopy string) ui.Node {
	return ui.Fragment(
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text(parseEyebrow)),
		html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(parseCopy)),
	)
}

func inventoryQueueRowCells(parseItem inventorySummaryCard) ui.Node {
	return ui.Fragment(
		html.Tag("td", html.Props{Class: "px-3 py-3"},
			html.A(html.Props{Href: "/app/inventory/" + parseItem.SKU, Class: "grid gap-2 transition hover:text-cyan-100"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseItem.Title)),
					html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " "))),
				),
				html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-500"}, html.Text(parseItem.SKU)),
			),
		),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-300"}, html.Text(parseItem.PrimaryLane)),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Inbound))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.RiskLaneCount))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-400"}, html.Text(parseItem.LastUpdated)),
	)
}

func inventoryQueueTable(parseItems []inventorySummaryCard) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, inventoryQueueTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Td(html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]any{"colSpan": 8}}, html.Text("No inventory rows match the current filter set. Clear filters or pivot into a saved view with live lane pressure.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Dense queue table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Scan SKU posture, lane spread, and inbound exposure in one table, then branch directly into the SKU route or warehouse lane workspace that needs edits.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d rows", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Table(html.Props{Class: "min-w-full border-collapse text-left"},
				html.Thead(html.Props{},
					html.Tr(html.Props{Class: "bg-slate-950/80"},
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Item")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Primary lane")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Lane spread")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Inbound")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tbody(html.Props{}, parseRows...),
			),
		),
	)
}

func inventoryQueueTableRow(parseItem inventorySummaryCard) ui.Node {
	return html.Tr(html.Props{Class: internalTableRowClass() + " align-top"},
		html.Td(html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/inventory/" + parseItem.SKU, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.Title)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseItem.SKU)),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseItem.PrimaryLane)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d total lanes", parseItem.LaneCount))),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-300"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fmt.Sprintf("%d risk lanes", parseItem.RiskLaneCount))),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d healthy lanes", maxInt(parseItem.LaneCount-parseItem.RiskLaneCount, 0)))),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " ")))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Inbound))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(parseItem.LastUpdated)),
		html.Td(html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/inventory/" + parseItem.SKU, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open SKU")),
				html.A(html.Props{Href: "/app/warehouses", Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open warehouse ops")),
			),
		),
	)
}

func inventoryOperationsRail(parseWorkspace internalShellState) ui.Node {
	filterNodes := workspaceFilterSummaryNodes(parseWorkspace.ActiveFilters, "All inventory rows are visible.")
	parseStatNodes := []ui.Node{}
	for _, parseItem := range parseWorkspace.WorkspaceStats {
		parseStatNodes = append(parseStatNodes, warehouseInfoRow(parseItem.Label, parseItem.Value))
	}
	if len(parseStatNodes) == 0 {
		parseStatNodes = append(parseStatNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No shared workspace summary is available for this route yet.")))
	}
	parseViewTitle := "Saved-view selection"
	parseViewBody := []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No saved inventory view currently matches the active workspace filters."))}
	if strings.TrimSpace(parseWorkspace.ActiveSavedView) != "" {
		parseViewBody = []ui.Node{warehouseInfoRow("Active view", parseWorkspace.ActiveSavedView)}
	}
	parseActionNodes := []ui.Node{
		html.A(html.Props{Href: "/app/inventory?status=promise_risk", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open promise-risk lanes")),
		html.A(html.Props{Href: "/app/warehouses", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Switch to warehouse ops")),
		html.A(html.Props{Href: "/app/purchase-orders", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open purchase orders")),
		html.A(html.Props{Href: "/app/receiving", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Check receiving queue")),
	}
	return html.Div(html.Props{Class: "grid gap-5"},
		inventoryRailCard("Current view", "Keep the route rail focused on the exact workspace scope currently driving the queue table.",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"}, parseStatNodes...),
		),
		inventoryRailCard("Active filters", "The right rail mirrors the queue scope so saved views and side workflows stay interpretable while filters change.", filterNodes...),
		inventoryRailCard(parseViewTitle, "Atlas keeps the active saved-view selection visible in the route rail so operators can confirm why the current queue looks the way it does.", parseViewBody...),
		inventoryRailCard("Immediate actions", "Use the rail for adjacent operational hops after the main queue identifies the problem surface.", parseActionNodes...),
	)
}

func inventorySavedViewsRail(parseViewNodes []ui.Node) ui.Node {
	return inventoryRailCard("Saved views", "Apply inventory presets without losing the triage shell. Saved views stay in the side rail so the queue remains the route's main visual priority.", parseViewNodes...)
}

func inventoryRailCard(parseTitle, parseCopy string, parseChildren ...ui.Node) ui.Node {
	parseContent := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text(parseTitle)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
		),
	}
	parseContent = append(parseContent, parseChildren...)
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"}, parseContent...)
}

func workspaceFilterSummaryNodes(parseLabels []string, parseEmptyCopy string) []ui.Node {
	if len(parseLabels) == 0 {
		return []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(parseEmptyCopy))}
	}
	parseNodes := make([]ui.Node, 0, len(parseLabels))
	for _, parseLabel := range parseLabels {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "border border-slate-700 bg-slate-950/75 px-3 py-3 text-sm text-slate-200"}, html.Text(parseLabel)))
	}
	return parseNodes
}

func inventoryDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[inventoryDetailPage](pageData(parsePayload))
	if len(parsePage.Rows) == 0 {
		return featureCard("Inventory item unavailable", "This SKU does not currently have any active inventory lanes.")
	}
	parseCurrentPath := "/app/inventory/" + parsePage.SKU
	parseRollup := inventoryRollupFromRows(parsePage.Rows)
	parseLaneCards := make([]ui.Node, 0, len(parsePage.Rows))
	for _, parseRow := range parsePage.Rows {
		parseLaneCards = append(parseLaneCards, inventoryLaneEditorCardWithOptions(parseRow, parsePayload, parseCurrentPath))
	}
	parseAsideChildren := []ui.Node{
		skuOperationsRail(parsePage, parseRollup.TotalAvailable, parseRollup.InboundUnits, parseRollup.RiskLanes),
		html.Div(html.Props{ID: "sku-replenishment"}, orderInventoryModalCard(parsePage.Rows, parsePage.Rows[0].WarehouseID, parsePage.Title, parsePayload, true, parseCurrentPath)),
	}
	if parseOverlay := inventoryThresholdHistoryPanelNode(parsePayload, parsePage); parseOverlay != nil {
		parseAsideChildren = append(parseAsideChildren, parseOverlay)
	}
	parseAsideChildren = append(parseAsideChildren, thresholdForm(parsePage.Rows[0], parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.18fr)_minmax(23rem,0.82fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryDetailHero(parsePage, parseRollup),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				statCard("Available", fmt.Sprintf("%d units", parseRollup.TotalAvailable)),
				statCard("Inbound", fmt.Sprintf("%d units", parseRollup.InboundUnits)),
				statCard("Flagged lanes", fmt.Sprintf("%d lanes", parseRollup.RiskLanes)),
				statCard("Reorder lanes", fmt.Sprintf("%d lanes", parseRollup.ReorderLanes)),
			),
			skuLaneRosterTable(parsePage.Rows),
			inventoryLaneEditorsSection(parseLaneCards),
		),
		html.Div(html.Props{Class: "grid gap-5"}, parseAsideChildren...),
	)
}

func inventoryDetailHero(parsePage inventoryDetailPage, parseRollup InventoryRollup) ui.Node {
	parsePrimaryWarehouse := fallback(parsePage.Rows[0].WarehouseName, parsePage.Rows[0].WarehouseID)
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Inventory lane workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(parsePage.Title)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("%s is active across %d warehouse lanes. Use this route to compare pressure, edit the exact lane that drifted, and branch into replenishment only when balancing is no longer enough.", parsePage.SKU, len(parsePage.Rows)))),
				),
			),
			html.A(html.Props{Href: "/app/inventory", Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to inventory queue")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text("SKU "+parsePage.SKU)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fmt.Sprintf("%d lanes live", len(parsePage.Rows)))),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePrimaryWarehouse+" primary")),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fmt.Sprintf("%d risk lanes", parseRollup.RiskLanes))),
		),
	)
}

func inventoryThresholdHistoryPanelNode(parsePayload Payload, parsePage inventoryDetailPage) ui.Node {
	if parseOutlet := routeOutletNode(); parseOutlet != nil {
		return parseOutlet
	}
	parsePanel := decode[inventoryThresholdHistoryPanelPage](payloadDataValue(parsePayload, "overlay"))
	if parsePanel.SKU == "" || !strings.EqualFold(strings.TrimSpace(parsePanel.SKU), strings.TrimSpace(parsePage.SKU)) {
		return nil
	}
	return inventoryThresholdHistoryPanel(parsePanel, parsePage)
}

func InventoryThresholdHistoryOverlay(parsePayload Payload) ui.Node {
	parsePanel := decode[inventoryThresholdHistoryPanelPage](payloadDataValue(parsePayload, "overlay"))
	parsePage := decode[inventoryDetailPage](payloadDataValue(parsePayload, "page"))
	if parsePanel.SKU == "" || parsePage.SKU == "" {
		return nil
	}
	return inventoryThresholdHistoryPanel(parsePanel, parsePage)
}

func inventoryThresholdHistoryPanel(parsePanel inventoryThresholdHistoryPanelPage, parsePage inventoryDetailPage) ui.Node {
	parseCloseHref := "/app/inventory/" + parsePage.SKU
	parsePanelID := ui.UseId()
	parseTitleID := parsePanelID + "-title"
	parseDescriptionID := parsePanelID + "-description"
	parseCloseID := parsePanelID + "-close"
	parseLatestSignature := ""
	if len(parsePanel.Items) > 0 {
		parseLatest := parsePanel.Items[0]
		parseLatestSignature = parseLatest.ID + ":" + fmt.Sprintf("%d", parseLatest.ReorderPoint) + ":" + fmt.Sprintf("%d", parseLatest.SafetyStock) + ":" + parseLatest.CreatedAt
	}
	parsePreviousLatest := ui.UsePrevious(parseLatestSignature)
	parseHistoryNodes := make([]ui.Node, 0, len(parsePanel.Items))
	for _, parseItem := range parsePanel.Items {
		parseHistoryNodes = append(parseHistoryNodes, html.Div(html.Props{Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/75 px-4 py-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.Summary)),
				html.Span(html.Props{Class: warehouseStatusClass("pending")}, html.Text(parseItem.WarehouseID)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Detail)),
			html.Div(html.Props{Class: "grid gap-2 text-xs uppercase tracking-[0.2em] text-slate-500 md:grid-cols-3"},
				html.Span(html.Props{}, html.Text("Reorder "+fmt.Sprintf("%d", parseItem.ReorderPoint))),
				html.Span(html.Props{}, html.Text("Safety "+fmt.Sprintf("%d", parseItem.SafetyStock))),
				html.Span(html.Props{}, html.Text(parseItem.CreatedAt)),
			),
			html.P(html.Props{Class: "text-xs text-slate-500"}, html.Text("Updated by "+fallback(parseItem.ActorName, "Atlas operations"))),
		))
	}
	if len(parseHistoryNodes) == 0 {
		parseHistoryNodes = append(parseHistoryNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No threshold history events are recorded for this SKU yet.")))
	}
	parseRecommendationNodes := make([]ui.Node, 0, len(parsePanel.Recommendations))
	for _, parseItem2 := range parsePanel.Recommendations {
		parseRecommendationNodes = append(parseRecommendationNodes, html.Div(html.Props{Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/75 px-4 py-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(fallback(parseItem2.SourceWarehouseName, parseItem2.SourceWarehouseID)+" -> "+fallback(parseItem2.DestinationWarehouseName, parseItem2.DestinationWarehouseID))),
				html.Span(html.Props{Class: warehouseStatusClass(parseItem2.Priority)}, html.Text(fallback(parseItem2.Priority, "review"))),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem2.Reason)),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(fmt.Sprintf("%d units of %s", parseItem2.Quantity, parsePage.SKU))),
		))
	}
	if len(parseRecommendationNodes) == 0 {
		parseRecommendationNodes = append(parseRecommendationNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No transfer recommendation is stronger than local threshold tuning right now.")))
	}
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Route overlay")),
				html.P(html.Props{ID: parseTitleID, Class: "text-lg font-semibold text-white"}, html.Text("Threshold history")),
				html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text("This side panel is deep-linkable, so operators can review threshold decisions without losing the SKU route context.")),
			),
			html.A(html.Props{Href: parseCloseHref, Class: warehouseSecondaryButtonClass()}, html.Text("Close")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
			warehouseInfoRow("SKU", parsePage.SKU),
			warehouseInfoRow("History events", fmt.Sprintf("%d", len(parsePanel.Items))),
			warehouseInfoRow("Transfer cues", fmt.Sprintf("%d", len(parsePanel.Recommendations))),
			warehouseInfoRow("Return path", parseCloseHref),
		),
		warehouseListCard("Threshold changes", parseHistoryNodes...),
		warehouseListCard("Transfer recommendations", parseRecommendationNodes...),
	}
	if parseChangeSummary := thresholdHistoryChangeSummary(parsePanel.Items, parsePreviousLatest, parseLatestSignature); parseChangeSummary != nil {
		parseChildren = append([]ui.Node{parseChangeSummary}, parseChildren...)
	}
	parseChildren[0] = html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Route overlay")),
			html.P(html.Props{ID: parseTitleID, Class: "text-lg font-semibold text-white"}, html.Text("Threshold history")),
			html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-6 text-slate-400"}, html.Text("This side panel is deep-linkable, so operators can review threshold decisions without losing the SKU route context.")),
		),
		html.A(html.Props{ID: parseCloseID, Href: parseCloseHref, Class: warehouseSecondaryButtonClass()}, html.Text("Close")),
	)
	return atlasRouteSheetOverlay(parsePanelID, parseTitleID, parseDescriptionID, "#"+parseCloseID, html.Div(html.Props{
		Class: "grid gap-4",
		Raw: map[string]any{
			"data-atlas-route-overlay": "threshold-history",
		},
	}, parseChildren...))
}

func thresholdHistoryChangeSummary(parseItems []inventoryThresholdHistoryItem, parsePrevious ui.Previous[string], parseLatestSignature string) ui.Node {
	if !parsePrevious.Ok() || parseLatestSignature == "" || parsePrevious.Get() == "" || parsePrevious.Get() == parseLatestSignature || len(parseItems) == 0 {
		return nil
	}
	parseLatest := parseItems[0]
	return html.Div(html.Props{Class: "rounded-sm border border-cyan-300/25 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Latest threshold change")),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(fmt.Sprintf("Timeline refreshed with reorder %d and safety %d for %s.", parseLatest.ReorderPoint, parseLatest.SafetyStock, parseLatest.WarehouseID))),
	)
}

func skuLaneRosterTable(parseRows []inventoryRow) ui.Node {
	parseItems := make([]ui.Node, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseItems = append(parseItems, html.Tr(html.Props{Class: internalTableRowClass() + " align-top"},
			html.Td(html.Props{Class: "px-4 py-4"},
				html.A(html.Props{Href: "/app/warehouses/" + parseRow.WarehouseID + "/items/" + parseRow.SKU, Class: "grid gap-2 transition hover:text-cyan-100"},
					html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
						html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(parseRow.WarehouseName, parseRow.WarehouseID))),
						html.Span(html.Props{Class: warehouseStatusClass(parseRow.Status)}, html.Text(strings.ReplaceAll(parseRow.Status, "_", " "))),
					),
					html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-500"}, html.Text(parseRow.WarehouseID+" | "+parseRow.MarketPressure)),
				),
			),
			html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseRow.Available))),
			html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseRow.Inbound))),
			html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d days", parseRow.CoverDays))),
			html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseRow.ReorderUnits))),
			html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-400"}, html.Text(parseRow.UpdatedAt)),
			html.Td(html.Props{Class: "px-4 py-4"},
				html.A(html.Props{Href: "/app/warehouses/" + parseRow.WarehouseID + "/items/" + parseRow.SKU, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open lane")),
			),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Lane roster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Compare every active warehouse lane before touching quantities, thresholds, or replenishment so the operator can tell whether the issue is local or network-wide.")),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Table(html.Props{Class: "min-w-full border-collapse text-left"},
				html.Thead(html.Props{},
					html.Tr(html.Props{Class: "bg-slate-950/80"},
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Warehouse lane")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Inbound")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Cover")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Reorder")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Updated")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tbody(html.Props{}, parseItems...),
			),
		),
	)
}

func inventoryLaneEditorsSection(parseLaneCards []ui.Node) ui.Node {
	parseContent := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Lane editors")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Edit each lane in place after the roster confirms where the pressure really lives. Quantity, threshold, and status changes stay grouped under one inventory-specific section instead of scattered across utility cards.")),
		),
	}
	parseContent = append(parseContent, parseLaneCards...)
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"}, parseContent...)
}

func skuOperationsRail(parsePage inventoryDetailPage, parseTotalAvailable int, parseTotalInbound int, parseRiskLanes int) ui.Node {
	parsePrimaryHref := inventoryPrimaryWarehouseHref(parsePage.Rows)
	return html.Div(html.Props{Class: "grid gap-5"},
		inventoryRailCard("SKU actions", "Keep the right rail focused on the immediate handoffs around this SKU: merchandising, warehouse drill-in, replenishment, and receiving follow-up.",
			html.Div(html.Props{Class: "grid gap-3"},
				inventoryActionCard("Open product record", "Jump into the matching product editor when the lane issue surfaces a catalog or copy follow-up.", "/app/products?q="+url.QueryEscape(parsePage.SKU)),
				inventoryActionCard("Open primary warehouse item", "Drill into the warehouse-native item workspace when this issue is facility-specific.", parsePrimaryHref),
				inventoryActionCard("Open purchase orders", "Move into replenishment review if this SKU already needs vendor-side recovery.", "/app/purchase-orders"),
				inventoryActionCard("Open receiving", "Hand off to receiving once inbound work is actually moving and needs confirmation.", "/app/receiving"),
			),
		),
		inventoryRailCard("SKU snapshot", "Use the route rail to confirm lane count, stock posture, and the current anchor warehouse before changing the lane editors below.",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
				warehouseInfoRow("SKU", parsePage.SKU),
				warehouseInfoRow("Active lanes", fmt.Sprintf("%d", len(parsePage.Rows))),
				warehouseInfoRow("Available units", fmt.Sprintf("%d", parseTotalAvailable)),
				warehouseInfoRow("Inbound units", fmt.Sprintf("%d", parseTotalInbound)),
				warehouseInfoRow("Flagged lanes", fmt.Sprintf("%d", parseRiskLanes)),
				warehouseInfoRow("Primary warehouse", fallback(parsePage.Rows[0].WarehouseName, parsePage.Rows[0].WarehouseID)),
			),
		),
		inventoryRailCard("What this page controls", "This route is for lane comparison first, then lane correction, with replenishment and threshold work treated as explicit secondary workflows.",
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text("Use the lane roster to compare warehouses, edit the specific lane below, then create replenishment only when the numbers show that internal balancing is not enough.")),
		),
	)
}

func warehouseInventoryDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[warehouseInventoryDetailPage](pageData(parsePayload))
	return ui.CreateElement(func() ui.Node {
		parseSearch := useAtlasSearchParams()
		parseInitial := atlasMapFilterState(parsePage.Filters)
		parseForm := ui.UseForm(parseInitial)
		parseTransition := useAtlasTransition()
		useAtlasEffect(func() func() {
			if !atlasSameListFilterState(parseForm.Get(), parseInitial) {
				parseForm.Reset(parseInitial)
			}
			return nil
		}, parseInitial)
		parseValue := parseForm.Get()
		parseDeferred := ui.UseDeferredValue(parseValue)
		parseDebounced := ui.UseDebounced(parseValue, atlasFilterSyncDelay)
		useAtlasEffect(func() func() {
			parseNext := atlasBuildListFilterQuery(parseSearch.Values(), parseDebounced.Get())
			if parseNext.Encode() != parseSearch.Values().Encode() {
				parseSearch.ReplaceAll(parseNext)
			}
			return nil
		}, parseDebounced.Get())
		parseSubmit := ui.UseEvent(func(parseEvent ui.FormEvent) {
			parseEvent.PreventDefault()
			parseSearch.ReplaceAll(atlasBuildListFilterQuery(parseSearch.Values(), parseForm.Get()))
		})
		parseVisibleInventory := filterWarehouseInventoryRows(parsePage.Inventory, parseDeferred)
		parseWorkspace := warehouseDetailWorkspaceSnapshot{}
		for _, parseItem := range parseVisibleInventory {
			parseWorkspace.TotalDemand += parseItem.WeeklyUnits
			parseWorkspace.TotalRevenue += parseItem.WeeklyRevenue
			if parseItem.ReorderUnits > 0 || strings.EqualFold(parseItem.MarketPressure, "hot market") {
				parseWorkspace.UrgentCount++
			}
			if strings.EqualFold(strings.TrimSpace(parseItem.Status), "critical") || strings.EqualFold(strings.TrimSpace(parseItem.Status), "promise_risk") {
				parseWorkspace.RiskLanes++
			}
		}
		parseOrderNodes := make([]ui.Node, 0, minInt(len(parsePage.Orders), 4))
		for parseIndex, parseOrder := range parsePage.Orders {
			if parseIndex >= 4 {
				break
			}
			parseOrderNodes = append(parseOrderNodes,
				html.A(html.Props{Href: "/app/purchase-orders/" + parseOrder.ID, Class: "grid gap-2 " + internalInsetSurfaceClass() + " px-4 py-4 transition hover:border-cyan-300/45 hover:text-white"},
					html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseOrder.ID+" | "+parseOrder.VendorName)),
					html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text(parseOrder.Status+" | ETA "+parseOrder.ETA)),
				),
			)
		}
		if len(parseOrderNodes) == 0 {
			parseOrderNodes = append(parseOrderNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No recent replenishment orders are tied to this warehouse yet.")))
		}
		return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.14fr)_minmax(24rem,0.86fr)] xl:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				warehouseBreadcrumbBar(
					warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
					warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
					warehouseBreadcrumbLink{Label: fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID), Href: "/app/warehouses/" + parsePage.Warehouse.ID, Current: true},
				),
				warehouseDetailHero(parsePage, parseWorkspace),
				routeSummaryStrip(parsePage.Summary),
				warehouseInventoryFilterForm(parsePage.Warehouse.ID, parseForm, parseDebounced.Pending() || parseTransition.Pending(), parseSubmit, parseTransition),
				html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
					statCard("Warehouse items", fmt.Sprintf("%d active", len(parseVisibleInventory))),
					statCard("Weekly demand", fmt.Sprintf("%d units", parseWorkspace.TotalDemand)),
					statCard("Weekly revenue", formatPrice(parseWorkspace.TotalRevenue)),
					statCard("Urgent actions", fmt.Sprintf("%d flagged", parseWorkspace.UrgentCount)),
				),
				warehouseDetailActionCluster(parsePage.Warehouse.ID),
				warehouseOpsItemPanelNode(parsePayload),
				warehouseDetailInventoryTable(parsePage.Warehouse.ID, parseVisibleInventory),
			),
			html.Div(html.Props{Class: "grid gap-5"},
				inventoryRailCard("Facility snapshot", "Use the right rail to keep the facility context visible while the main pane stays focused on the dense item roster.",
					html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
						warehouseInfoRow("Region", parsePage.Warehouse.Region),
						warehouseInfoRow("Service level", parsePage.Warehouse.ServiceLevel),
						warehouseInfoRow("Pressure", parsePage.Warehouse.Pressure),
						warehouseInfoRow("Backlog", parsePage.Warehouse.Backlog),
						warehouseInfoRow("Staffing", parsePage.Warehouse.Staffing),
						warehouseInfoRow("Risk lanes", fmt.Sprintf("%d", parsePage.Warehouse.RiskCount)),
					),
				),
				inventoryRailCard("Active workspace filters", "Warehouse filters stay mirrored in the rail so nested item work and replenishment actions remain interpretable.", workspaceFilterSummaryNodes(currentRouteWorkspaceState(parsePayload).ActiveFilters, "All warehouse items are visible.")...),
				html.Div(html.Props{ID: "warehouse-create-item"}, productCreateFormWithOptions(parsePayload, productFormOptions{LockedWarehouseID: parsePage.Warehouse.ID, ReturnWarehouseID: parsePage.Warehouse.ID, IntroLabel: "Add warehouse item", SubmitLabel: "Create warehouse item"})),
				html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard(parsePage.Inventory, parsePage.Warehouse.ID, parsePage.Warehouse.Name, parsePayload, false, "/app/warehouses/"+parsePage.Warehouse.ID)),
				inventoryRailCard("Recent purchase orders", "Keep recent replenishment work visible beside the warehouse roster so operators can decide whether to edit locally or follow existing inbound recovery.", parseOrderNodes...),
			),
		)
	})
}

func warehouseDetailHero(parsePage warehouseInventoryDetailPage, parseWorkspace warehouseDetailWorkspaceSnapshot) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Warehouse facility workspace")),
				html.Div(html.Props{Class: "grid gap-2"},
					html.H2(html.Props{Class: "text-3xl font-semibold text-white"}, html.Text(parsePage.Warehouse.Name)),
					html.P(html.Props{Class: "text-sm leading-7 text-slate-300"}, html.Text(parsePage.Warehouse.Focus)),
				),
			),
			html.A(html.Props{Href: RouteWarehouseOps, Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-4 py-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-300/45 hover:text-white"}, html.Text("Back to facility table")),
		),
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Warehouse.Region)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Warehouse.ServiceLevel)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(parsePage.Warehouse.Pressure)),
			html.Span(html.Props{Class: internalSurfacePillClass()}, html.Text(fmt.Sprintf("%d risk lanes", parseWorkspace.RiskLanes))),
		),
	)
}

func warehouseDetailActionCluster(parseWarehouseID string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Facility action cluster")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Use the warehouse route for item creation, flagged-lane review, replenishment staging, and receiving follow-up without losing the facility context.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-4"},
			inventoryActionCard("Add warehouse item", "Create a new managed item directly inside this warehouse workspace.", "/app/warehouses/"+parseWarehouseID+"#warehouse-create-item"),
			inventoryActionCard("Review flagged lanes", "Tighten the route to the lanes that need action before editing quantities or thresholds.", "/app/warehouses/"+parseWarehouseID+"?status=promise_risk"),
			inventoryActionCard("Open inventory risk", "Compare this facility against the wider SKU pressure board before committing local changes.", "/app/inventory?status=promise_risk"),
			inventoryActionCard("Check receiving", "Follow inbound recovery through receiving once replenishment has actually moved.", "/app/receiving"),
		),
	)
}

func warehouseDetailInventoryTable(parseWarehouseID string, parseItems []inventoryRow) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, warehouseDetailInventoryTableRow(parseWarehouseID, parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]any{"colSpan": 7}}, html.Text("No warehouse items match the current view. Adjust filters or create a new managed item from the route rail.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Warehouse item table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review item posture, demand, revenue, and reorder pressure in one dense facility table before drilling into the nested item workspace.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d items", len(parseItems)))),
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
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

func warehouseDetailInventoryTableRow(parseWarehouseID string, parseItem inventoryRow) ui.Node {
	return html.Tag("tr", html.Props{Class: internalTableRowClass() + " align-top"},
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + parseWarehouseID + "/items/" + parseItem.SKU, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.Title)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseItem.SKU+" | "+strings.Title(parseItem.Category))),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.MarketSignal+" | "+parseItem.MarketPressure)),
			),
		),
		html.Tag("td", html.Props{Class: "px-4 py-4"}, html.Span(html.Props{Class: warehouseStatusClass(parseItem.Status)}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " ")))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d / week", parseItem.WeeklyUnits))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(formatPrice(parseItem.WeeklyRevenue))),
		html.Tag("td", html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.ReorderUnits))),
		html.Tag("td", html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/warehouses/" + parseWarehouseID + "/items/" + parseItem.SKU, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Open item")),
				html.A(html.Props{Href: "/app/products/" + parseItem.Slug, Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open product")),
			),
		),
	)
}

// WarehouseOpsDetailPanel renders the nested warehouse-detail workspace panel for child warehouse routes.
func WarehouseOpsDetailPanel(parsePayload Payload) ui.Node {
	parsePage := decode[warehouseInventoryDetailPage](payloadDataValue(parsePayload, "detail"))
	if strings.TrimSpace(parsePage.Warehouse.ID) == "" {
		parsePage = decode[warehouseInventoryDetailPage](pageData(parsePayload))
	}
	if strings.TrimSpace(parsePage.Warehouse.ID) == "" {
		return nil
	}
	return warehouseOpsDetailNestedContent(parsePayload, parsePage)
}

func warehouseOpsDetailNestedContent(parsePayload Payload, parsePage warehouseInventoryDetailPage) ui.Node {
	parseWorkspace := warehouseDetailWorkspaceSnapshotFromPage(parsePage)
	return html.Section(html.Props{Class: "grid gap-5 rounded-[1.6rem] border border-cyan-300/25 bg-[linear-gradient(180deg,rgba(8,16,30,0.95),rgba(6,12,24,0.98))] p-5 shadow-[0_24px_60px_rgba(3,10,24,0.36)]"},
		warehouseBreadcrumbBar(
			warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
			warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
			warehouseBreadcrumbLink{Label: fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID), Href: "/app/warehouses/" + parsePage.Warehouse.ID, Current: true},
		),
		warehouseDetailHero(parsePage, parseWorkspace),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Warehouse items", fmt.Sprintf("%d active", len(parsePage.Inventory))),
			statCard("Weekly demand", fmt.Sprintf("%d units", parseWorkspace.TotalDemand)),
			statCard("Weekly revenue", formatPrice(parseWorkspace.TotalRevenue)),
			statCard("Urgent actions", fmt.Sprintf("%d flagged", parseWorkspace.UrgentCount)),
		),
		warehouseDetailActionCluster(parsePage.Warehouse.ID),
		warehouseOpsItemPanelNode(parsePayload),
		warehouseDetailInventoryTable(parsePage.Warehouse.ID, parsePage.Inventory),
	)
}

func WarehouseOpsItemPanel(parsePayload Payload) ui.Node {
	parsePage := decode[warehouseInventoryItemDetailPage](payloadDataValue(parsePayload, "item"))
	if strings.TrimSpace(parsePage.Item.SKU) == "" {
		parsePage = decode[warehouseInventoryItemDetailPage](pageData(parsePayload))
	}
	if strings.TrimSpace(parsePage.Item.SKU) == "" {
		return nil
	}
	return warehouseOpsItemNestedContent(parsePayload, parsePage)
}

func warehouseOpsItemPanelNode(parsePayload Payload) ui.Node {
	if parseOutlet := routeOutletNode(); parseOutlet != nil {
		return parseOutlet
	}
	return WarehouseOpsItemPanel(parsePayload)
}

func warehouseOpsItemNestedContent(parsePayload Payload, parsePage warehouseInventoryItemDetailPage) ui.Node {
	parseCurrentPath := "/app/warehouses/" + parsePage.Warehouse.ID + "/items/" + parsePage.Item.SKU
	parseParentPath := warehouseParentWorkspaceHref(parsePayload, parsePage.Warehouse.ID)
	parseFilters := parsePage.Filters
	if parseFilters == nil {
		parseFilters = map[string]string{"sort": "updated", "dir": "desc"}
	}
	if strings.TrimSpace(parseFilters["sort"]) == "" {
		parseFilters["sort"] = "updated"
	}
	if strings.TrimSpace(parseFilters["dir"]) == "" {
		parseFilters["dir"] = warehouseItemDefaultDirection(parseFilters["sort"])
	}
	parseOrderNodes := make([]ui.Node, 0, len(parsePage.Orders))
	for _, parseOrder := range parsePage.Orders {
		parseOrderNodes = append(parseOrderNodes, html.A(html.Props{Href: "/app/purchase-orders/" + parseOrder.ID, Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 transition hover:border-cyan-400/45 hover:bg-slate-950"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseOrder.ID+" · "+parseOrder.VendorName)),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(parseOrder.Status+" · ETA "+parseOrder.ETA)),
		))
	}
	if len(parseOrderNodes) == 0 {
		parseOrderNodes = append(parseOrderNodes, html.Div(html.Props{Class: "rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-400"}, html.Text("No open replenishment orders are tied to this item in the current warehouse.")))
	}
	return html.Section(html.Props{Class: "grid gap-5 rounded-[1.4rem] border border-cyan-400/25 bg-[linear-gradient(180deg,rgba(9,14,29,0.98),rgba(15,23,42,0.98))] p-5 shadow-[0_28px_70px_rgba(8,15,28,0.42)] lg:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.88fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-slate-950/70 p-4"},
				html.Div(html.Props{Class: "flex flex-wrap items-start justify-between gap-3"},
					html.Div(html.Props{Class: "grid gap-1"},
						html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Nested warehouse item workspace")),
						html.H2(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(parsePage.Product.Title)),
						html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parsePage.Product.Summary)),
					),
					html.A(html.Props{Href: parseParentPath, Class: "inline-flex items-center justify-center rounded-full border border-slate-700 px-3 py-2 text-xs font-semibold uppercase tracking-[0.24em] text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Back to warehouse roster")),
				),
				html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-300 md:grid-cols-4"},
					warehouseInfoRow("SKU", parsePage.Item.SKU),
					warehouseInfoRow("Warehouse", fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID)),
					warehouseInfoRow("Status", strings.ReplaceAll(parsePage.Item.Status, "_", " ")),
					warehouseInfoRow("Regional pressure", parsePage.Item.MarketPressure),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
				warehouseStatCard("Available", fmt.Sprintf("%d units", parsePage.Item.Available)),
				warehouseStatCard("Weekly demand", fmt.Sprintf("%d units", parsePage.Item.WeeklyUnits)),
				warehouseStatCard("Weekly revenue", formatPrice(parsePage.Item.WeeklyRevenue)),
				warehouseStatCard("Order more", fmt.Sprintf("%d units", parsePage.Item.ReorderUnits)),
			),
			internalWorkflowSection("Warehouse item admin flows", "Stay inside the warehouse workspace while you fix this lane, then jump out only when the issue truly becomes merchandising, replenishment, or receiving work.",
				internalWorkflowCard("Flow 1", "Edit stock lane", "Jump straight to the lane editor on this nested panel when the task is quantity, threshold, or status correction.", parseCurrentPath+"#warehouse-lane-editor"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product editor when the warehouse issue surfaces a summary, details, or SEO problem.", "/app/products/"+parsePage.Product.Slug),
				internalWorkflowCard("Flow 3", "Order more units", "Use the replenishment panel on this route when the fix requires vendor-side inbound, not only local edits.", parseCurrentPath+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Return to warehouse filters", "Drop back to the parent warehouse roster with the same route-owned filter context still visible.", parseParentPath),
			),
			warehouseItemNetworkTable(parseCurrentPath, parseFilters, parsePage.Warehouse.ID, parsePage.Network),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{ID: "warehouse-lane-editor"}, inventoryLaneEditorCardWithOptions(parsePage.Item, parsePayload, parseCurrentPath)),
			productUpdateFormWithOptions(parsePage.Product, parsePayload, productFormOptions{LockedWarehouseID: parsePage.Warehouse.ID, ReturnWarehouseID: parsePage.Warehouse.ID, IntroLabel: "Edit warehouse item", SubmitLabel: "Save warehouse item"}),
			html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard([]inventoryRow{parsePage.Item}, parsePage.Warehouse.ID, parsePage.Product.Title, parsePayload, true, parseCurrentPath)),
			warehouseListCard("Related replenishment orders", parseOrderNodes...),
			productDeleteFormWithOptions(parsePage.Product, parsePayload, productFormOptions{ReturnWarehouseID: parsePage.Warehouse.ID, DeleteLabel: "Delete warehouse item", DeleteCopy: "Deleting this item removes the product and its warehouse inventory from the Atlas demo data. Use this only when the warehouse should stop managing it entirely."}),
		),
	)
}

func warehouseOpsItemContent(parsePayload Payload) ui.Node {
	parsePage := decode[warehouseInventoryItemDetailPage](pageData(parsePayload))
	parseCurrentPath := "/app/warehouses/" + parsePage.Warehouse.ID + "/items/" + parsePage.Item.SKU
	parseFilters := parsePage.Filters
	if parseFilters == nil {
		parseFilters = map[string]string{"sort": "updated", "dir": "desc"}
	}
	if strings.TrimSpace(parseFilters["sort"]) == "" {
		parseFilters["sort"] = "updated"
	}
	if strings.TrimSpace(parseFilters["dir"]) == "" {
		parseFilters["dir"] = warehouseItemDefaultDirection(parseFilters["sort"])
	}
	parseOrderNodes := make([]ui.Node, 0, len(parsePage.Orders))
	for _, parseOrder := range parsePage.Orders {
		parseOrderNodes = append(parseOrderNodes, html.A(html.Props{Href: "/app/purchase-orders/" + parseOrder.ID, Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 transition hover:border-cyan-400/45 hover:bg-slate-950"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseOrder.ID+" · "+parseOrder.VendorName)),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(parseOrder.Status+" · ETA "+parseOrder.ETA)),
		))
	}
	if len(parseOrderNodes) == 0 {
		parseOrderNodes = append(parseOrderNodes, html.Div(html.Props{Class: "rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-400"}, html.Text("No open replenishment orders are tied to this item in the current warehouse.")))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.08fr)_minmax(24rem,0.88fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
				warehouseBreadcrumbLink{Label: fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID), Href: "/app/warehouses/" + parsePage.Warehouse.ID},
				warehouseBreadcrumbLink{Label: fallback(parsePage.Product.Title, parsePage.Item.SKU), Href: parseCurrentPath, Current: true},
			),
			warehouseFeatureCard(parsePage.Product.Title, parsePage.Product.Summary),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
				warehouseStatCard("Available", fmt.Sprintf("%d units", parsePage.Item.Available)),
				warehouseStatCard("Weekly demand", fmt.Sprintf("%d units", parsePage.Item.WeeklyUnits)),
				warehouseStatCard("Weekly revenue", formatPrice(parsePage.Item.WeeklyRevenue)),
				warehouseStatCard("Order more", fmt.Sprintf("%d units", parsePage.Item.ReorderUnits)),
			),
			warehouseListCard("Market and sales readout",
				warehouseInfoRow("Category", strings.Title(parsePage.Item.Category)),
				warehouseInfoRow("Sell-through", fmt.Sprintf("%d%%", parsePage.Item.SellThrough)),
				warehouseInfoRow("Demand score", fmt.Sprintf("%d / 100", parsePage.Item.DemandScore)),
				warehouseInfoRow("Regional share", fmt.Sprintf("%d%%", parsePage.Item.RegionalShare)),
			),
			internalWorkflowSection("Warehouse item admin flows", "Complete the warehouse item job here, then jump directly into copy, replenishment, or receiving routes without retracing your steps.",
				internalWorkflowCard("Flow 1", "Edit stock lane", "Jump straight to the lane editor on this page when the task is quantity, threshold, or status correction.", parseCurrentPath+"#warehouse-lane-editor"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product editor when the warehouse issue surfaces a summary, details, or SEO problem.", "/app/products/"+parsePage.Product.Slug),
				internalWorkflowCard("Flow 3", "Order more units", "Use the replenishment panel on this route when the fix requires vendor-side inbound, not only local edits.", parseCurrentPath+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Back to warehouse roster", "Return to the parent warehouse list when you need to move from this item into the next local task.", "/app/warehouses/"+parsePage.Warehouse.ID),
			),
			warehouseItemNetworkTable(parseCurrentPath, parseFilters, parsePage.Warehouse.ID, parsePage.Network),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{ID: "warehouse-lane-editor"}, inventoryLaneEditorCardWithOptions(parsePage.Item, parsePayload, parseCurrentPath)),
			productUpdateFormWithOptions(parsePage.Product, parsePayload, productFormOptions{LockedWarehouseID: parsePage.Warehouse.ID, ReturnWarehouseID: parsePage.Warehouse.ID, IntroLabel: "Edit warehouse item", SubmitLabel: "Save warehouse item"}),
			html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard([]inventoryRow{parsePage.Item}, parsePage.Warehouse.ID, parsePage.Product.Title, parsePayload, true, parseCurrentPath)),
			warehouseListCard("Related replenishment orders", parseOrderNodes...),
			productDeleteFormWithOptions(parsePage.Product, parsePayload, productFormOptions{ReturnWarehouseID: parsePage.Warehouse.ID, DeleteLabel: "Delete warehouse item", DeleteCopy: "Deleting this item removes the product and its warehouse inventory from the Atlas demo data. Use this only when the warehouse should stop managing it entirely."}),
		),
	)
}

func warehouseParentWorkspaceHref(parsePayload Payload, parseWarehouseID string) string {
	parseValues := url.Values{}
	for parseKey, parseItems := range parsePayload.Route.Query {
		if strings.EqualFold(strings.TrimSpace(parseKey), "sort") || strings.EqualFold(strings.TrimSpace(parseKey), "dir") {
			continue
		}
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	parseHref := "/app/warehouses/" + parseWarehouseID
	if parseEncoded := parseValues.Encode(); parseEncoded != "" {
		return parseHref + "?" + parseEncoded
	}
	return parseHref
}

func warehouseItemNetworkRowCells(parseCurrentWarehouseID string, parseItem inventoryRow) ui.Node {
	parseBadgeClass := warehouseStatusClass(parseItem.Status)
	parseWarehouseLabelClass := "text-xs uppercase tracking-[0.22em] text-slate-400"
	if strings.EqualFold(parseItem.WarehouseID, parseCurrentWarehouseID) {
		parseBadgeClass = "inline-flex items-center border border-cyan-400/45 bg-cyan-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-200"
		parseWarehouseLabelClass = "text-xs uppercase tracking-[0.22em] text-cyan-200"
	}
	return ui.Fragment(
		html.Tag("td", html.Props{Class: "px-3 py-3"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
					html.Span(html.Props{Class: parseBadgeClass}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " "))),
				),
				html.P(html.Props{Class: parseWarehouseLabelClass}, html.Text(parseItem.WarehouseID+" · "+parseItem.MarketPressure)),
				html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(parseItem.MarketSignal)),
			),
		),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d days", parseItem.CoverDays))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Inbound))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d / week", parseItem.WeeklyUnits))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d%%", parseItem.RegionalShare))),
		html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-300"}, html.Text(parseItem.UpdatedAt)),
	)
}

func warehouseItemNetworkTable(parseCurrentPath string, parseFilters map[string]string, parseCurrentWarehouseID string, parseItems []inventoryRow) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Tag("tr", html.Props{Class: "border-t border-slate-700/90 align-top"}, warehouseItemNetworkRowCells(parseCurrentWarehouseID, parseItem)))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-3 py-4 text-sm text-slate-400", Raw: map[string]any{"colSpan": 7}}, html.Text("No warehouse lanes are available for this item.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.Div(html.Props{Class: "grid gap-1"}, atlasSectionMeta("Warehouse item table", "Track the current warehouse lane beside the other hubs with the standard Atlas inventory columns plus demand context. Click any column header to reorder the table.")),
		html.Div(html.Props{Class: "overflow-x-auto border border-slate-700 bg-slate-950/70"},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-900"},
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "warehouse", "Warehouse"),
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "available", "Available"),
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "cover", "Cover days"),
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "inbound", "Inbound"),
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "demand", "Demand"),
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "share", "Regional share"),
						warehouseItemTableHeader(parseCurrentPath, parseFilters, "updated", "Updated"),
					),
				),
				html.Tag("tbody", html.Props{}, parseRows...),
			),
		),
	)
}

type warehouseBreadcrumbLink struct {
	Label   string
	Href    string
	Current bool
}

func warehouseBreadcrumbBar(parseLinks ...warehouseBreadcrumbLink) ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseLinks)*2)
	for parseIndex, parseLink := range parseLinks {
		parseLabel := fallback(strings.TrimSpace(parseLink.Label), "Route")
		if parseIndex > 0 {
			parseNodes = append(parseNodes, html.Span(html.Props{Class: "text-slate-600"}, html.Text("/")))
		}
		if parseLink.Current || strings.TrimSpace(parseLink.Href) == "" {
			parseNodes = append(parseNodes, html.Span(html.Props{Class: "text-slate-100"}, html.Text(parseLabel)))
			continue
		}
		parseNodes = append(parseNodes, html.A(html.Props{Href: parseLink.Href, Class: "transition hover:text-cyan-100"}, html.Text(parseLabel)))
	}
	return html.Div(html.Props{Class: "flex flex-wrap items-center gap-2 border border-slate-700 bg-slate-950/80 px-3 py-2 text-[0.72rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, parseNodes...)
}

func warehouseItemTableHeader(parseCurrentPath string, parseFilters map[string]string, parseSortKey string, parseLabel string) ui.Node {
	parseActive := strings.EqualFold(strings.TrimSpace(parseFilters["sort"]), parseSortKey)
	parseDirection := warehouseItemDefaultDirection(parseSortKey)
	if parseActive {
		if strings.EqualFold(strings.TrimSpace(parseFilters["dir"]), "asc") {
			parseDirection = "desc"
		} else {
			parseDirection = "asc"
		}
	}
	parseValues := url.Values{}
	parseValues.Set("sort", parseSortKey)
	parseValues.Set("dir", parseDirection)
	parseIndicator := ""
	parseLinkClass := "inline-flex items-center gap-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500 transition hover:text-cyan-200"
	if parseActive {
		parseLinkClass = "inline-flex items-center gap-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-cyan-200 transition hover:text-cyan-100"
		if strings.EqualFold(strings.TrimSpace(parseFilters["dir"]), "asc") {
			parseIndicator = "↑"
		} else {
			parseIndicator = "↓"
		}
	}
	parseChildren := []ui.Node{html.Text(parseLabel)}
	if parseIndicator != "" {
		parseChildren = append(parseChildren, html.Span(html.Props{Class: "text-cyan-200"}, html.Text(parseIndicator)))
	}
	return html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 last:border-r-0"}, html.A(html.Props{Href: parseCurrentPath + "?" + parseValues.Encode(), Target: "_self", Class: parseLinkClass}, parseChildren...))
}

func warehouseItemDefaultDirection(parseSortKey string) string {
	switch strings.TrimSpace(strings.ToLower(parseSortKey)) {
	case "warehouse", "cover":
		return "asc"
	default:
		return "desc"
	}
}

func filterInventoryRows(parseItems []inventoryRow, parseFilters atlasListFilterState) []inventoryRow {
	parseNeedle := atlasNormalizedFilterValue(parseFilters.Query)
	parseWarehouse := atlasNormalizedFilterValue(parseFilters.Warehouse)
	parseStatus := atlasNormalizedFilterValue(parseFilters.Status)
	parseFiltered := make([]inventoryRow, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if parseNeedle != "" {
			parseHaystack := atlasNormalizedFilterValue(parseItem.Title + " " + parseItem.SKU + " " + parseItem.Category + " " + parseItem.WarehouseName + " " + parseItem.WarehouseID + " " + parseItem.MarketSignal + " " + parseItem.MarketPressure)
			if !strings.Contains(parseHaystack, parseNeedle) {
				continue
			}
		}
		if parseWarehouse != "" && parseWarehouse != "all" && !strings.EqualFold(strings.TrimSpace(parseItem.WarehouseID), parseWarehouse) {
			continue
		}
		if parseStatus != "" && parseStatus != "all" && !strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			continue
		}
		parseFiltered = append(parseFiltered, parseItem)
	}
	switch atlasNormalizedFilterValue(parseFilters.Sort) {
	case "inbound":
		sort.SliceStable(parseFiltered, func(parseLeft, parseRight int) bool {
			if parseFiltered[parseLeft].Inbound == parseFiltered[parseRight].Inbound {
				return parseFiltered[parseLeft].UpdatedAt > parseFiltered[parseRight].UpdatedAt
			}
			return parseFiltered[parseLeft].Inbound > parseFiltered[parseRight].Inbound
		})
	default:
		sort.SliceStable(parseFiltered, func(parseLeft2, parseRight2 int) bool {
			return parseFiltered[parseLeft2].UpdatedAt > parseFiltered[parseRight2].UpdatedAt
		})
	}
	return parseFiltered
}

func filterWarehouseInventoryRows(parseItems []inventoryRow, parseFilters atlasListFilterState) []inventoryRow {
	parseNeedle := atlasNormalizedFilterValue(parseFilters.Query)
	parseStatus := atlasNormalizedFilterValue(parseFilters.Status)
	parseFiltered := make([]inventoryRow, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if parseNeedle != "" {
			parseHaystack := atlasNormalizedFilterValue(parseItem.Title + " " + parseItem.SKU + " " + parseItem.Category + " " + parseItem.MarketSignal + " " + parseItem.MarketPressure)
			if !strings.Contains(parseHaystack, parseNeedle) {
				continue
			}
		}
		if parseStatus != "" && parseStatus != "all" && !strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			continue
		}
		parseFiltered = append(parseFiltered, parseItem)
	}
	switch atlasNormalizedFilterValue(parseFilters.Sort) {
	case "available":
		sort.SliceStable(parseFiltered, func(parseLeft, parseRight int) bool {
			if parseFiltered[parseLeft].Available == parseFiltered[parseRight].Available {
				return parseFiltered[parseLeft].UpdatedAt > parseFiltered[parseRight].UpdatedAt
			}
			return parseFiltered[parseLeft].Available < parseFiltered[parseRight].Available
		})
	case "demand":
		sort.SliceStable(parseFiltered, func(parseLeft2, parseRight2 int) bool {
			if parseFiltered[parseLeft2].WeeklyUnits == parseFiltered[parseRight2].WeeklyUnits {
				return parseFiltered[parseLeft2].UpdatedAt > parseFiltered[parseRight2].UpdatedAt
			}
			return parseFiltered[parseLeft2].WeeklyUnits > parseFiltered[parseRight2].WeeklyUnits
		})
	case "revenue":
		sort.SliceStable(parseFiltered, func(parseLeft3, parseRight3 int) bool {
			if parseFiltered[parseLeft3].WeeklyRevenue == parseFiltered[parseRight3].WeeklyRevenue {
				return parseFiltered[parseLeft3].UpdatedAt > parseFiltered[parseRight3].UpdatedAt
			}
			return parseFiltered[parseLeft3].WeeklyRevenue > parseFiltered[parseRight3].WeeklyRevenue
		})
	default:
		sort.SliceStable(parseFiltered, func(parseLeft4, parseRight4 int) bool {
			return parseFiltered[parseLeft4].UpdatedAt > parseFiltered[parseRight4].UpdatedAt
		})
	}
	return parseFiltered
}

func inventoryCMSFilterForm(parseForm ui.Form[atlasListFilterState], isSyncing bool, parseSubmit ui.Handler, parseTransition atlasTransition) ui.Node {
	parseValue := parseForm.Get()
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Filter model")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep query, warehouse, and lane-status pivots visible in one row so operators can tighten the queue without losing deep-linkable state.")),
		),
		html.Form(html.Props{Action: "/app/inventory", Method: "get", OnSubmit: parseSubmit, Class: "grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.7fr))_auto] lg:items-end"},
			cmsTransitionBoundTextInput("q", "Search inventory", parseValue.Query, "Query", parseForm, parseTransition),
			cmsTransitionBoundSelectInput("warehouse", "Warehouse", parseValue.Warehouse, "Warehouse", append([]optionItem{{"all", "All warehouses"}}, warehouseOptions()...), parseForm, parseTransition),
			cmsTransitionBoundSelectInput("status", "Lane status", parseValue.Status, "Status", append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...), parseForm, parseTransition),
			cmsTransitionBoundSelectInput("sort", "Sort", parseValue.Sort, "Sort", []optionItem{{"updated", "Updated recently"}, {"inbound", "Highest inbound"}}, parseForm, parseTransition),
			html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(atlasFilterSubmitLabel(isSyncing, "Filter"))),
		),
	)
}

func inventoryLaneEditorCardWithOptions(parseRow inventoryRow, parsePayload Payload, parseReturnPath string) ui.Node {
	parseWarehouseLabel := fallback(parseRow.WarehouseName, parseRow.WarehouseID)
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(inventoryLaneEditorFormState{
			WarehouseID:  parseRow.WarehouseID,
			OnHand:       fmt.Sprintf("%d", parseRow.OnHand),
			Reserved:     fmt.Sprintf("%d", parseRow.Reserved),
			Damaged:      fmt.Sprintf("%d", parseRow.Damaged),
			Inbound:      fmt.Sprintf("%d", parseRow.Inbound),
			ReorderPoint: fmt.Sprintf("%d", parseRow.ReorderPoint),
			SafetyStock:  fmt.Sprintf("%d", parseRow.SafetyStock),
			Status:       parseRow.Status,
			ReturnPath:   parseReturnPath,
		})
		parseValue := parseForm.Get()
		parseChildren := []ui.Node{
			html.Input(html.Props{Type: "hidden", Name: "warehouse_id", Value: parseValue.WarehouseID}),
			inventoryBoundNumberField("on_hand", "On hand", parseValue.OnHand, "OnHand", parseForm),
			inventoryBoundNumberField("reserved", "Reserved", parseValue.Reserved, "Reserved", parseForm),
			inventoryBoundNumberField("damaged", "Damaged", parseValue.Damaged, "Damaged", parseForm),
			inventoryBoundNumberField("inbound", "Inbound", parseValue.Inbound, "Inbound", parseForm),
			inventoryBoundNumberField("reorder_point", "Reorder point", parseValue.ReorderPoint, "ReorderPoint", parseForm),
			inventoryBoundNumberField("safety_stock", "Safety stock", parseValue.SafetyStock, "SafetyStock", parseForm),
			inventoryBoundSelectField("status", "Lane status", parseValue.Status, "Status", inventoryStatusOptions(), parseForm),
			html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Save lane")),
		}
		if strings.TrimSpace(parseValue.ReturnPath) != "" {
			parseChildren = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: parseValue.ReturnPath})}, parseChildren...)
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-1"},
					html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.2em] text-white"}, html.Text(parseWarehouseLabel)),
					html.P(html.Props{Class: "text-[0.7rem] uppercase tracking-[0.24em] text-slate-500"}, html.Text(parseRow.WarehouseID+" | updated "+parseRow.UpdatedAt)),
				),
				html.Span(html.Props{Class: warehouseStatusClass(parseValue.Status)}, html.Text(strings.ReplaceAll(parseValue.Status, "_", " "))),
			),
			html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-300 md:grid-cols-3"},
				warehouseInfoRow("Available", fmt.Sprintf("%d units", parseRow.Available)),
				warehouseInfoRow("Inbound", fmt.Sprintf("%d units", parseRow.Inbound)),
				warehouseInfoRow("Cover", fmt.Sprintf("%d days", parseRow.CoverDays)),
			),
			html.Form(html.Props{Action: "/api/app/inventory/" + parseRow.SKU + "/update", Method: "post", Class: "grid gap-4 md:grid-cols-2"}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...),
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
func orderInventoryModalCard(parseRows []inventoryRow, parseDefaultWarehouse string, parseTitle string, parsePayload Payload, isSkuScoped bool, parseReturnPath string) ui.Node {
	if len(parseRows) == 0 {
		return nil
	}
	parseProductOptions := inventoryProductOptions(parseRows)
	parseWarehouseOptionsList := inventoryWarehouseLaneOptions(parseRows)
	parseDefaultSKU := parseRows[0].SKU
	if parseDefaultWarehouse == "" {
		parseDefaultWarehouse = parseRows[0].WarehouseID
	}
	parseModalID := inventoryModalID(parseDefaultSKU, parseDefaultWarehouse)
	parseButtonLabel := "Order new items"
	parseDescription := "Create a purchase order from the active management surface and push inbound units onto the selected warehouse lane immediately."
	if isSkuScoped {
		parseButtonLabel = "Order replenishment"
		parseDescription = "Launch a replenishment order for this SKU without leaving the inventory editor."
	}
	return ui.CreateElement(func() ui.Node {
		parseWorkflow := ui.UseReducer(reducePurchaseOrderWorkflowState, defaultPurchaseOrderWorkflowState(parseDefaultSKU, parseDefaultWarehouse, parseReturnPath))
		parseOpen := ui.UseState(false)
		parseOpenModal := ui.UseEvent(func(ui.Event) {
			parseOpen.Set(true)
		})
		parseCloseModal := ui.UseEvent(func(ui.Event) {
			parseOpen.Set(false)
		})
		parseValue := parseWorkflow.Get()
		parseFields := []ui.Node{
			purchaseOrderWorkflowSummaryCard(parseValue),
			inventoryReducerTextField("vendor_name", "Vendor", parseValue.VendorName, parseWorkflow),
			inventoryReducerNumberField("quantity", "Quantity", parseValue.Quantity, parseWorkflow),
			inventoryReducerTextField("eta", "ETA", parseValue.ETA, parseWorkflow),
			inventoryReducerTextField("priority_note", "Priority note", parseValue.PriorityNote, parseWorkflow),
			inventoryReducerSelectField("status", "Order status", parseValue.Status, []optionItem{{"draft", "Draft"}, {"submitted", "Submitted"}, {"approved", "Approved"}, {"on_hold", "On hold"}}, parseWorkflow),
		}
		if isSkuScoped || len(parseProductOptions) == 1 {
			parseFields = append([]ui.Node{
				html.Input(html.Props{Type: "hidden", Name: "product_sku", Value: parseValue.ProductSKU}),
				html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm text-slate-300"}, html.Text("Item: "+parseTitle+" · "+parseValue.ProductSKU)),
			}, parseFields...)
		} else {
			parseFields = append([]ui.Node{inventoryReducerSelectField("product_sku", "Inventory item", parseValue.ProductSKU, parseProductOptions, parseWorkflow)}, parseFields...)
		}
		if len(parseWarehouseOptionsList) == 1 {
			parseFields = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "warehouse_id", Value: parseValue.WarehouseID})}, parseFields...)
		} else {
			parseFields = append([]ui.Node{inventoryReducerSelectField("warehouse_id", "Warehouse", parseValue.WarehouseID, parseWarehouseOptionsList, parseWorkflow)}, parseFields...)
		}
		if strings.TrimSpace(parseValue.ReturnPath) != "" {
			parseFields = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: parseValue.ReturnPath})}, parseFields...)
		}
		parseFormChildren := prependCSRFToken(parsePayload.CSRF, parseFields...)
		parseFormChildren = append(parseFormChildren,
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-end gap-3 md:col-span-2"},
				html.Button(html.Props{Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: parseCloseModal}, html.Text("Cancel")),
				html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Create order")),
			),
		)
		parseModalSurfaceID := parseModalID + "-surface"
		parseModalCloseID := parseModalID + "-close"
		useAtlasFocusContainment(parseOpen.Get(), "#"+parseModalSurfaceID, "#"+parseModalCloseID)
		parseChildren := []ui.Node{
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Replenishment")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(parseDescription)),
			html.Button(html.Props{Type: "button", Class: "w-fit border border-cyan-400/45 bg-cyan-500/10 px-4 py-2 text-sm font-semibold uppercase tracking-[0.18em] text-cyan-200 transition hover:border-cyan-300 hover:bg-cyan-500/16 hover:text-cyan-100", OnClick: parseOpenModal}, html.Text(parseButtonLabel)),
		}
		if parseOpen.Get() {
			parseChildren = append(parseChildren, html.Div(html.Props{Class: "fixed inset-0 z-50 flex items-center justify-center bg-slate-950/75 p-4"},
				html.Button(html.Props{Type: "button", Class: "absolute inset-0", OnClick: parseCloseModal, Raw: map[string]any{"aria-label": "Close replenishment dialog"}}),
				html.Div(html.Props{
					ID:    parseModalSurfaceID,
					Class: "relative z-10 grid w-[min(92vw,40rem)] gap-4 border border-slate-700 bg-[linear-gradient(180deg,rgba(17,24,39,0.99),rgba(2,6,23,1))] p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)]",
					Raw: map[string]any{
						"role":       "dialog",
						"aria-modal": "true",
						"tabindex":   "-1",
					},
				},
					html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
						html.Div(html.Props{Class: "grid gap-2"},
							html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Order new items")),
							html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
							html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(parseDescription)),
						),
						html.Button(html.Props{ID: parseModalCloseID, Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: parseCloseModal}, html.Text("Close")),
					),
					html.Form(html.Props{Action: "/api/app/purchase-orders", Method: "post", Class: "grid gap-4 md:grid-cols-2"}, parseFormChildren...),
				),
			))
		}
		return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"}, parseChildren...)
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

func purchaseOrderWorkflowSummaryCard(parseState purchaseOrderWorkflowState) ui.Node {
	parseStageLabel := "Ready to submit"
	parseStageCopy := "Vendor, quantity, and warehouse context are aligned for a replenishment order."
	switch parseState.Stage {
	case "inbound-confirmed":
		parseStageLabel = "Inbound confirmed"
		parseStageCopy = "This draft is already approved, so Atlas treats it as committed inbound inventory."
	case "blocked":
		parseStageLabel = "Blocked"
		parseStageCopy = "The order is on hold and should not move until the note explains the blocker."
	case "draft-needs-brief":
		parseStageLabel = "Needs brief"
		parseStageCopy = "Add a priority note before this stays in draft so the next operator has context."
	case "draft-review":
		parseStageLabel = "Draft review"
		parseStageCopy = "The order is still a draft, but it already has enough context for review."
	case "needs-quantity":
		parseStageLabel = "Needs quantity"
		parseStageCopy = "Atlas will not treat this replenishment as actionable until quantity is filled in."
	}
	parseEdited := "Workflow staged from the current replenishment defaults."
	if strings.TrimSpace(parseState.LastEditedField) != "" {
		parseEdited = "Last updated field: " + strings.ReplaceAll(parseState.LastEditedField, "_", " ") + "."
	}
	return html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100 md:col-span-2"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text(parseStageLabel)),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(parseStageCopy)),
		html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.2em] text-cyan-200/75"}, html.Text(parseEdited)),
	)
}

func inventoryReducerTextField(parseName, parseLabel, parseValue string, parseWorkflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{
			ID: parseId, Name: parseName, Value: parseValue, Class: warehouseInputClass(),
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				parseWorkflow.Dispatch(purchaseOrderWorkflowAction{Field: parseName, Value: parseEvent.GetValue()})
			}),
			Raw: map[string]any{"aria-labelledby": parseId + "-label"},
		}),
	)
}

func inventoryReducerNumberField(parseName, parseLabel, parseValue string, parseWorkflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{
			ID: parseId, Type: "number", Name: parseName, Value: parseValue, Class: warehouseInputClass(),
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				parseWorkflow.Dispatch(purchaseOrderWorkflowAction{Field: parseName, Value: parseEvent.GetValue()})
			}),
			Raw: map[string]any{"aria-labelledby": parseId + "-label"},
		}),
	)
}

func inventoryReducerSelectField(parseName, parseLabel, parseValue string, parseOptions []optionItem, parseWorkflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value))}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Select(html.Props{
			ID: parseId, Name: parseName, Class: warehouseInputClass(),
			OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) {
				parseWorkflow.Dispatch(purchaseOrderWorkflowAction{Field: parseName, Value: parseEvent.GetValue()})
			}),
			Raw: map[string]any{"aria-labelledby": parseId + "-label"},
		}, parseChildren...),
	)
}

func warehouseInventoryFilterForm(parseWarehouseID string, parseForm ui.Form[atlasListFilterState], isSyncing bool, parseSubmit ui.Handler, parseTransition atlasTransition) ui.Node {
	parseValue := parseForm.Get()
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Facility filter model")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep warehouse item search, lane status, and sort order visible in one card so the facility roster stays URL-backed and easy to narrow under hydration.")),
		),
		html.Form(html.Props{Action: "/app/warehouses/" + parseWarehouseID, Method: "get", OnSubmit: parseSubmit, Class: "grid gap-4 lg:grid-cols-[minmax(0,1.2fr)_repeat(2,minmax(0,0.75fr))_auto] lg:items-end"},
			cmsTransitionBoundTextInput("q", "Search items", parseValue.Query, "Query", parseForm, parseTransition),
			cmsTransitionBoundSelectInput("status", "Lane status", parseValue.Status, "Status", append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...), parseForm, parseTransition),
			cmsTransitionBoundSelectInput("sort", "Sort", parseValue.Sort, "Sort", []optionItem{{"updated", "Updated"}, {"available", "Lowest available"}, {"demand", "Highest demand"}, {"revenue", "Highest revenue"}}, parseForm, parseTransition),
			html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text(atlasFilterSubmitLabel(isSyncing, "Apply"))),
		),
	)
}

func marketPressureClass(parseLabel string) string {
	switch strings.TrimSpace(strings.ToLower(parseLabel)) {
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

func inventoryProductOptions(parseRows []inventoryRow) []optionItem {
	parseSeen := map[string]bool{}
	parseOptions := make([]optionItem, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseSeen[parseRow.SKU] {
			continue
		}
		parseSeen[parseRow.SKU] = true
		parseOptions = append(parseOptions, optionItem{Value: parseRow.SKU, Label: parseRow.Title + " · " + parseRow.SKU})
	}
	return parseOptions
}

func inventoryWarehouseLaneOptions(parseRows []inventoryRow) []optionItem {
	parseSeen := map[string]bool{}
	parseOptions := make([]optionItem, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseSeen[parseRow.WarehouseID] {
			continue
		}
		parseSeen[parseRow.WarehouseID] = true
		parseOptions = append(parseOptions, optionItem{Value: parseRow.WarehouseID, Label: inventoryWarehouseLabel(parseRow.WarehouseID, parseRows)})
	}
	return parseOptions
}

func inventoryWarehouseLabel(parseWarehouseID string, parseRows []inventoryRow) string {
	for _, parseRow := range parseRows {
		if parseRow.WarehouseID == parseWarehouseID {
			return fallback(parseRow.WarehouseName, parseRow.WarehouseID)
		}
	}
	return parseWarehouseID
}

func inventoryTextField(parseName, parseLabel, parseValue string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Input(html.Props{Name: parseName, Value: parseValue, Class: warehouseInputClass()}),
	)
}

func inventoryNumberField(parseName, parseLabel, parseValue string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Input(html.Props{Type: "number", Name: parseName, Value: parseValue, Class: warehouseInputClass()}),
	)
}

func inventoryBoundNumberField[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Type: "number", Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func inventorySelectField(parseName, parseLabel, parseValue string, parseOptions []optionItem) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value))}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Select(html.Props{Name: parseName, Class: warehouseInputClass()}, parseChildren...),
	)
}

func inventoryBoundSelectField[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value))}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Select(html.Props{ID: parseId, Name: parseName, OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, parseChildren...),
	)
}

func warehouseFeatureCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-2 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Warehouse operations")),
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text(parseCopy)),
	)
}

func warehouseStatCard(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-sm border border-slate-700 bg-slate-950/80 p-4"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.26em] text-slate-500"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-2 text-lg font-semibold text-white"}, html.Text(parseValue)),
	)
}

func warehouseListCard(parseTitle string, parseChildren ...ui.Node) ui.Node {
	if len(parseChildren) == 0 {
		parseChildren = []ui.Node{html.P(html.Props{Class: "text-sm text-slate-500"}, html.Text("No records yet."))}
	}
	parseContent := []ui.Node{html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text(parseTitle))}
	parseContent = append(parseContent, parseChildren...)
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"}, parseContent...)
}

func warehouseInfoRow(parsePrimary string, parseSecondary string) ui.Node {
	return html.Div(html.Props{Class: "border border-slate-700 bg-slate-950/75 px-3 py-3"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.22em] text-slate-500"}, html.Text(parsePrimary)),
		html.P(html.Props{Class: "mt-1 text-sm text-slate-200"}, html.Text(parseSecondary)),
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

func warehouseStatusClass(parseStatus string) string {
	parseBase := "inline-flex items-center border px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em]"
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "balanced", "in_stock", "healthy", "approved", "available":
		return parseBase + " border-emerald-400/45 bg-emerald-500/10 text-emerald-100"
	case "promise_risk", "low_stock", "pending", "submitted", "in_review":
		return parseBase + " border-amber-400/45 bg-amber-500/10 text-amber-100"
	case "critical":
		return parseBase + " border-rose-400/45 bg-rose-500/10 text-rose-100"
	default:
		return parseBase + " border-slate-500/45 bg-slate-500/10 text-slate-200"
	}
}

func minInt(parseLeft, parseRight int) int {
	if parseLeft < parseRight {
		return parseLeft
	}
	return parseRight
}

func maxInt(parseLeft, parseRight int) int {
	if parseLeft > parseRight {
		return parseLeft
	}
	return parseRight
}

func inventoryPrimaryWarehouseHref(parseRows []inventoryRow) string {
	if len(parseRows) == 0 {
		return "/app/inventory"
	}
	parsePrimary := parseRows[0]
	return "/app/warehouses/" + parsePrimary.WarehouseID + "/items/" + parsePrimary.SKU
}

func inventoryModalID(parseSku string, parseWarehouseID string) string {
	parseReplacer := strings.NewReplacer("/", "-", " ", "-", ".", "-", ":", "-", "_", "-")
	return "inventory-order-" + parseReplacer.Replace(strings.ToLower(strings.TrimSpace(parseSku))) + "-" + parseReplacer.Replace(strings.ToLower(strings.TrimSpace(parseWarehouseID)))
}
