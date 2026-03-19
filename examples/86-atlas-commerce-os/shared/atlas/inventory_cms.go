package atlas

import (
	"fmt"
	"net/url"
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

func inventoryCMSContent(payload Payload) ui.Node {
	page := decode[inventoryCMSPage](pageData(payload))
	summaries := inventorySummaryCards(page.Items)
	viewNodes := make([]ui.Node, 0, len(payload.SavedViews))
	for _, saved := range payload.SavedViews {
		viewNodes = append(viewNodes, html.Div(html.Props{Class: "border border-slate-700 bg-slate-950/80 px-3 py-3"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(saved.Name)),
			html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text(saved.Scope+" · "+saved.SortKey+"/"+saved.SortDirection)),
		))
	}
	if len(viewNodes) == 0 {
		viewNodes = append(viewNodes, html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("No saved inventory views yet.")))
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.32fr)_minmax(21rem,0.78fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseFeatureCard("Inventory control", "Triage stock pressure, inspect inbound exposure, and open the exact SKU editor without wading through route-explainer cards."),
			routeSummaryStrip(page.Summary),
			inventoryCMSFilterForm(page.Filters),
			inventoryTriageBand(page.Items, summaries),
			inventoryQueueTable(summaries),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			inventoryOperationsRail(page.Filters, summaries, page.Items),
			warehouseListCard("Saved views", viewNodes...),
			savedViewForm(payload),
		),
	)
}

func inventoryTriageBand(rows []inventoryRow, summaries []inventorySummaryCard) ui.Node {
	rollup := inventoryRollupFromRows(rows)
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
		nodes = append(nodes, html.A(html.Props{Href: item.href, Class: "grid gap-2 border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 transition hover:border-cyan-400/45 hover:bg-[linear-gradient(180deg,rgba(36,48,66,0.98),rgba(15,23,42,1))]"},
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text(item.label)),
			html.P(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(item.value)),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(item.copy)),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2 xl:grid-cols-4"}, nodes...)
}

func inventoryQueueTable(items []inventorySummaryCard) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, html.Tag("tr", html.Props{Class: "border-t border-slate-700/90 align-top"},
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
		))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-3 py-4 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 6}}, html.Text("No inventory rows match the current filter set.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("SKU queue")),
				html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("Open the exact inventory item that needs lane edits, replenishment, or threshold changes.")),
			),
			html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.24em] text-slate-500"}, html.Text(fmt.Sprintf("%d SKUs", len(items)))),
		),
		html.Div(html.Props{Class: "overflow-x-auto border border-slate-700 bg-slate-950/70"},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-900"},
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Item")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Primary lane")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Available")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Inbound")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Risk lanes")),
						html.Tag("th", html.Props{Class: "border-b px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Updated")),
					),
				),
				html.Tag("tbody", html.Props{}, rows...),
			),
		),
	)
}

func inventoryOperationsRail(filters map[string]string, summaries []inventorySummaryCard, rows []inventoryRow) ui.Node {
	if filters == nil {
		filters = map[string]string{}
	}
	rollup := inventoryRollupFromRows(rows)
	filterNodes := inventoryFilterSummaryNodes(filters)
	actionNodes := []ui.Node{
		html.A(html.Props{Href: "/app/inventory?status=promise_risk", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open promise-risk lanes")),
		html.A(html.Props{Href: "/app/warehouses", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Switch to warehouse ops")),
		html.A(html.Props{Href: "/app/purchase-orders", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open purchase orders")),
		html.A(html.Props{Href: "/app/receiving", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Check receiving queue")),
	}
	return html.Div(html.Props{Class: "grid gap-5"},
		warehouseListCard("Current view",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
				warehouseInfoRow("Visible SKUs", fmt.Sprintf("%d", len(summaries))),
				warehouseInfoRow("Visible lanes", fmt.Sprintf("%d", rollup.VisibleLanes)),
				warehouseInfoRow("Risk lanes", fmt.Sprintf("%d", rollup.RiskLanes)),
				warehouseInfoRow("Inbound units", fmt.Sprintf("%d", rollup.InboundUnits)),
			),
		),
		warehouseListCard("Active filters", filterNodes...),
		warehouseListCard("Immediate actions", actionNodes...),
	)
}

func inventoryFilterSummaryNodes(filters map[string]string) []ui.Node {
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
	if len(labels) == 0 {
		return []ui.Node{html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("All inventory rows are visible."))}
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
	rollup := inventoryRollupFromRows(page.Rows)
	laneCards := make([]ui.Node, 0, len(page.Rows))
	for _, row := range page.Rows {
		laneCards = append(laneCards, inventoryLaneEditorCard(row, payload))
	}
	asideChildren := []ui.Node{
		skuOperationsRail(page, rollup.TotalAvailable, rollup.InboundUnits, rollup.RiskLanes),
		html.Div(html.Props{ID: "sku-replenishment"}, orderInventoryModalCard(page.Rows, page.Rows[0].WarehouseID, page.Title, payload, true, "")),
	}
	if overlay := inventoryThresholdHistoryPanelNode(payload, page); overlay != nil {
		asideChildren = append(asideChildren, overlay)
	}
	asideChildren = append(asideChildren, thresholdForm(page.Rows[0], payload))
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.28fr)_minmax(22rem,0.82fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseFeatureCard(page.Title, fmt.Sprintf("%s is active across %d warehouse lanes. Use this page to inspect lane pressure first, then edit the specific lane that needs correction.", page.SKU, len(page.Rows))),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				warehouseStatCard("Available", fmt.Sprintf("%d units", rollup.TotalAvailable)),
				warehouseStatCard("Inbound", fmt.Sprintf("%d units", rollup.InboundUnits)),
				warehouseStatCard("Flagged lanes", fmt.Sprintf("%d", rollup.RiskLanes)),
				warehouseStatCard("Reorder lanes", fmt.Sprintf("%d", rollup.ReorderLanes)),
			),
			skuLaneRosterTable(page.Rows),
			warehouseListCard("Lane editors", laneCards...),
		),
		html.Div(html.Props{Class: "grid gap-5"}, asideChildren...),
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
	return html.Div(html.Props{Class: "grid gap-4 rounded-sm border border-cyan-400/35 bg-[linear-gradient(180deg,rgba(11,18,32,0.98),rgba(2,6,23,1))] p-4 shadow-[0_18px_48px_rgba(6,182,212,0.08)]", Raw: map[string]interface{}{"data-atlas-route-overlay": "threshold-history"}},
		html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Route overlay")),
				html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("Threshold history")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-400"}, html.Text("This side panel is deep-linkable, so operators can review threshold decisions without losing the SKU route context.")),
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
	)
}

func skuLaneRosterTable(rows []inventoryRow) ui.Node {
	items := make([]ui.Node, 0, len(rows))
	for _, row := range rows {
		items = append(items, html.Tag("tr", html.Props{Class: "border-t border-slate-700/90 align-top"},
			html.Tag("td", html.Props{Class: "px-3 py-3"},
				html.A(html.Props{Href: "/app/warehouses/" + row.WarehouseID + "/items/" + row.SKU, Class: "grid gap-2 transition hover:text-cyan-100"},
					html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
						html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(row.WarehouseName, row.WarehouseID))),
						html.Span(html.Props{Class: warehouseStatusClass(row.Status)}, html.Text(strings.ReplaceAll(row.Status, "_", " "))),
					),
					html.P(html.Props{Class: "text-[0.72rem] uppercase tracking-[0.22em] text-slate-500"}, html.Text(row.WarehouseID+" · "+row.MarketPressure)),
				),
			),
			html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", row.Available))),
			html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", row.Inbound))),
			html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d days", row.CoverDays))),
			html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", row.ReorderUnits))),
			html.Tag("td", html.Props{Class: "px-3 py-3 text-sm text-slate-400"}, html.Text(row.UpdatedAt)),
		))
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.Div(html.Props{Class: "grid gap-1"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Lane roster")),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("See every active warehouse lane for this SKU before touching quantities, thresholds, or replenishment.")),
		),
		html.Div(html.Props{Class: "overflow-x-auto border border-slate-700 bg-slate-950/70"},
			html.Tag("table", html.Props{Class: "min-w-full border-collapse text-left"},
				html.Tag("thead", html.Props{},
					html.Tag("tr", html.Props{Class: "bg-slate-900"},
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Warehouse lane")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Available")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Inbound")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Cover")),
						html.Tag("th", html.Props{Class: "border-b border-r border-slate-700 px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Reorder")),
						html.Tag("th", html.Props{Class: "border-b px-3 py-2 text-[0.7rem] font-semibold uppercase tracking-[0.24em] text-slate-500"}, html.Text("Updated")),
					),
				),
				html.Tag("tbody", html.Props{}, items...),
			),
		),
	)
}

func skuOperationsRail(page inventoryDetailPage, totalAvailable int, totalInbound int, riskLanes int) ui.Node {
	primaryHref := inventoryPrimaryWarehouseHref(page.Rows)
	return html.Div(html.Props{Class: "grid gap-5"},
		warehouseListCard("SKU actions",
			html.A(html.Props{Href: "/app/products?q=" + url.QueryEscape(page.SKU), Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open product record")),
			html.A(html.Props{Href: primaryHref, Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open primary warehouse item")),
			html.A(html.Props{Href: "/app/purchase-orders", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open purchase orders")),
			html.A(html.Props{Href: "/app/receiving", Class: "border border-slate-700 bg-slate-950/80 px-3 py-3 text-sm text-slate-200 transition hover:border-cyan-400/45 hover:text-white"}, html.Text("Open receiving")),
		),
		warehouseListCard("SKU snapshot",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
				warehouseInfoRow("SKU", page.SKU),
				warehouseInfoRow("Active lanes", fmt.Sprintf("%d", len(page.Rows))),
				warehouseInfoRow("Available units", fmt.Sprintf("%d", totalAvailable)),
				warehouseInfoRow("Inbound units", fmt.Sprintf("%d", totalInbound)),
				warehouseInfoRow("Flagged lanes", fmt.Sprintf("%d", riskLanes)),
				warehouseInfoRow("Primary warehouse", fallback(page.Rows[0].WarehouseName, page.Rows[0].WarehouseID)),
			),
		),
		warehouseListCard("What this page controls",
			html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text("Use the lane roster to compare warehouses, edit the specific lane below, then create replenishment only when the numbers show that internal balancing is not enough.")),
		),
	)
}

func warehouseInventoryDetailContent(payload Payload) ui.Node {
	page := decode[warehouseInventoryDetailPage](pageData(payload))
	filters := page.Filters
	if filters == nil {
		filters = map[string]string{}
	}
	totalDemand := 0
	totalRevenue := 0
	urgentCount := 0
	inventoryNodes := make([]ui.Node, 0, len(page.Inventory))
	for _, item := range page.Inventory {
		totalDemand += item.WeeklyUnits
		totalRevenue += item.WeeklyRevenue
		if item.ReorderUnits > 0 || strings.EqualFold(item.MarketPressure, "hot market") {
			urgentCount++
		}
		inventoryNodes = append(inventoryNodes,
			html.A(html.Props{Href: "/app/warehouses/" + page.Warehouse.ID + "/items/" + item.SKU, Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(30,41,59,0.94),rgba(15,23,42,0.98))] p-3 transition hover:border-cyan-400/45 hover:bg-[linear-gradient(180deg,rgba(36,48,66,0.98),rgba(15,23,42,1))]"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
					html.Div(html.Props{Class: "grid gap-1"},
						html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.Title+" · "+item.SKU)),
						html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(strings.Title(item.Category)+" · "+item.MarketSignal)),
					),
					html.Div(html.Props{Class: "grid justify-items-end gap-2"},
						html.Span(html.Props{Class: warehouseStatusClass(item.Status)}, html.Text(strings.ReplaceAll(item.Status, "_", " "))),
						html.Span(html.Props{Class: marketPressureClass(item.MarketPressure)}, html.Text(item.MarketPressure)),
					),
				),
				html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-300 md:grid-cols-4"},
					warehouseInfoRow("Available", fmt.Sprintf("%d units", item.Available)),
					warehouseInfoRow("Demand", fmt.Sprintf("%d units / week", item.WeeklyUnits)),
					warehouseInfoRow("Revenue", formatPrice(item.WeeklyRevenue)),
					warehouseInfoRow("Reorder", fmt.Sprintf("%d units", item.ReorderUnits)),
				),
			),
		)
	}
	if len(inventoryNodes) == 0 {
		inventoryNodes = append(inventoryNodes, featureCard("No warehouse items match the current view", "Adjust the warehouse filters or create a new managed item directly in this warehouse workspace."))
	}
	orderNodes := make([]ui.Node, 0, minInt(len(page.Orders), 4))
	for index, order := range page.Orders {
		if index >= 4 {
			break
		}
		orderNodes = append(orderNodes,
			html.A(html.Props{Href: "/app/purchase-orders/" + order.ID, Class: "grid gap-2 rounded-sm border border-slate-700 bg-slate-950/80 px-3 py-3 transition hover:border-cyan-400/45 hover:bg-slate-950"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(order.ID+" · "+order.VendorName)),
				html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text(order.Status+" · ETA "+order.ETA)),
			),
		)
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.08fr)_minmax(24rem,0.88fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
				warehouseBreadcrumbLink{Label: fallback(page.Warehouse.Name, page.Warehouse.ID), Href: "/app/warehouses/" + page.Warehouse.ID, Current: true},
			),
			warehouseFeatureCard(page.Warehouse.Name, page.Warehouse.Focus),
			routeSummaryStrip(page.Summary),
			warehouseInventoryFilterForm(page.Warehouse.ID, filters),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
				warehouseStatCard("Warehouse items", fmt.Sprintf("%d active", len(page.Inventory))),
				warehouseStatCard("Weekly demand", fmt.Sprintf("%d units", totalDemand)),
				warehouseStatCard("Weekly revenue", formatPrice(totalRevenue)),
				warehouseStatCard("Urgent actions", fmt.Sprintf("%d flagged", urgentCount)),
			),
			warehouseListCard("Warehouse items",
				warehouseInfoRow(page.Warehouse.Region, page.Warehouse.ServiceLevel),
				warehouseInfoRow(page.Warehouse.Pressure, page.Warehouse.Backlog),
				warehouseInfoRow(page.Warehouse.Staffing, fmt.Sprintf("%d risk lanes", page.Warehouse.RiskCount)),
			),
			internalWorkflowSection("Warehouse admin flows", "Keep warehouse work local: add the item, fix the lane, create replenishment, then hand off to receiving only when inbound is actually moving.",
				internalWorkflowCard("Flow 1", "Add new warehouse item", "Jump straight to the create form in this warehouse workspace and seed the new item into the current facility.", "/app/warehouses/"+page.Warehouse.ID+"#warehouse-create-item"),
				internalWorkflowCard("Flow 2", "Review flagged lanes", "Filter this warehouse down to low-confidence lanes before you start changing quantities or thresholds.", "/app/warehouses/"+page.Warehouse.ID+"?status=promise_risk"),
				internalWorkflowCard("Flow 3", "Order more units", "Open the replenishment workspace for this warehouse when internal edits are not enough to recover supply.", "/app/warehouses/"+page.Warehouse.ID+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Check receiving", "Use receiving as the final step after warehouse replenishment actually turns into inbound movement.", "/app/receiving"),
			),
			html.Div(html.Props{Class: "grid gap-4"}, inventoryNodes...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-1"},
				warehouseStatCard("Available", fmt.Sprintf("%d units", page.Warehouse.Available)),
				warehouseStatCard("Inbound", fmt.Sprintf("%d units", page.Warehouse.Inbound)),
				warehouseStatCard("Backlog", page.Warehouse.Backlog),
			),
			html.Div(html.Props{ID: "warehouse-create-item"}, productCreateFormWithOptions(payload, productFormOptions{LockedWarehouseID: page.Warehouse.ID, ReturnWarehouseID: page.Warehouse.ID, IntroLabel: "Add warehouse item", SubmitLabel: "Create warehouse item"})),
			html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard(page.Inventory, page.Warehouse.ID, page.Warehouse.Name, payload, false, "/app/warehouses/"+page.Warehouse.ID)),
			warehouseListCard("Recent purchase orders", orderNodes...),
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

func warehouseItemNetworkTable(currentPath string, filters map[string]string, currentWarehouseID string, items []inventoryRow) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		badgeClass := warehouseStatusClass(item.Status)
		warehouseLabelClass := "text-xs uppercase tracking-[0.22em] text-slate-400"
		if strings.EqualFold(item.WarehouseID, currentWarehouseID) {
			badgeClass = "inline-flex items-center border border-cyan-400/45 bg-cyan-500/10 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-200"
			warehouseLabelClass = "text-xs uppercase tracking-[0.22em] text-cyan-200"
		}
		rows = append(rows, html.Tag("tr", html.Props{Class: "border-t border-slate-700/90 align-top"},
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
		))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tag("tr", html.Props{},
			html.Tag("td", html.Props{Class: "px-3 py-4 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 7}}, html.Text("No warehouse lanes are available for this item.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-3 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.Div(html.Props{Class: "grid gap-1"},
			html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.26em] text-cyan-300"}, html.Text("Warehouse item table")),
			html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("Track the current warehouse lane beside the other hubs with the standard Atlas inventory columns plus demand context. Click any column header to reorder the table.")),
		),
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

func inventoryCMSFilterForm(filters map[string]string) ui.Node {
	if filters == nil {
		filters = map[string]string{}
	}
	return html.Form(html.Props{Action: "/app/inventory", Method: "get", Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5 lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.7fr))_auto] lg:items-end"},
		cmsTextInput("q", "Search inventory", filters["q"]),
		cmsSelectInput("warehouse", "Warehouse", filters["warehouse"], append([]optionItem{{"all", "All warehouses"}}, warehouseOptions()...)),
		cmsSelectInput("status", "Lane status", filters["status"], append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...)),
		cmsSelectInput("sort", "Sort", filters["sort"], []optionItem{{"updated", "Updated recently"}, {"inbound", "Highest inbound"}}),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text("Filter")),
	)
}

func inventoryLaneEditorCard(row inventoryRow, payload Payload) ui.Node {
	return inventoryLaneEditorCardWithOptions(row, payload, "")
}

func inventoryLaneEditorCardWithOptions(row inventoryRow, payload Payload, returnPath string) ui.Node {
	warehouseLabel := fallback(row.WarehouseName, row.WarehouseID)
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
	return html.Div(html.Props{Class: "grid gap-4 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
		html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.2em] text-white"}, html.Text(warehouseLabel)),
				html.P(html.Props{Class: "text-[0.7rem] uppercase tracking-[0.24em] text-slate-500"}, html.Text(row.WarehouseID+" · updated "+row.UpdatedAt)),
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
	fields := []ui.Node{
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
	return html.Div(html.Props{Class: "grid gap-4 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 shadow-[inset_0_1px_0_rgba(148,163,184,0.08)]"},
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
	)
}

func warehouseInventoryFilterForm(warehouseID string, filters map[string]string) ui.Node {
	if filters == nil {
		filters = map[string]string{}
	}
	return html.Form(html.Props{Action: "/app/warehouses/" + warehouseID, Method: "get", Class: "grid gap-4 rounded-sm border border-slate-700 bg-[linear-gradient(180deg,rgba(31,41,55,0.92),rgba(15,23,42,0.98))] p-4 lg:grid-cols-[minmax(0,1.2fr)_repeat(2,minmax(0,0.75fr))_auto] lg:items-end"},
		cmsTextInput("q", "Search items", filters["q"]),
		cmsSelectInput("status", "Lane status", filters["status"], append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...)),
		cmsSelectInput("sort", "Sort", filters["sort"], []optionItem{{"updated", "Updated"}, {"available", "Lowest available"}, {"demand", "Highest demand"}, {"revenue", "Highest revenue"}}),
		html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Apply")),
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
