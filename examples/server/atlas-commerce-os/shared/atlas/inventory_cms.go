package atlas

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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

// savedViewButtonProps carries everything one saved-view button needs. Apply is a
// plain callback rather than the form/search/transition handles themselves, so the
// button knows how to ASK for a view change without knowing how one is performed.
type savedViewButtonProps struct {
	Saved     SavedViewPayload
	NextState atlasListFilterState
	Active    bool
	Apply     func(atlasListFilterState)
}

// savedViewButton is a component, not a helper, and that distinction is the whole
// point of this file's most subtle bug fix.
//
// The button's OnClick needs ui.UseEvent. This used to be inlined in a
// `for range parsePayload.SavedViews` loop inside inventoryCMSContent, which meant
// the PARENT fiber's hook count was len(SavedViews) — a number the user changes by
// saving or deleting a view. GWC hooks are matched to their state by call ORDER
// within a fiber, so adding one saved view shifted every hook slot after the loop
// by one. Nothing panics: internal/runtime records the hook signature but does not
// assert it, so the failure is silent state corruption, not a crash.
//
// Giving each button its own fiber makes the parent's hook count constant and each
// button's hook slot its own. This shape was NOT available earlier: wrapping a
// repeated leaf in ui.CreateElement used to collide, because the component-handle
// cache keyed on qualified function name and every closure from one literal shared
// a handle, so N siblings all rendered the last one's captures. That framework bug
// was fixed (per-closure physical identity), which is what makes this correct now.
//
// The rule worth carrying: if a thing renders once per row of user data AND calls a
// hook, it is a component. A helper that calls hooks silently makes its caller's
// hook count a function of the data.
func savedViewButton(parseProps savedViewButtonProps) ui.Node {
	parseApply := parseProps.Apply
	parseNextState := parseProps.NextState
	return html.Button(html.Props{
		Type:  "button",
		Class: atlasSavedViewButtonClass(parseProps.Active),
		OnClick: ui.UseEvent(func() {
			if parseApply == nil {
				return
			}
			parseApply(parseNextState)
		}),
	},
		html.Span(html.Props{Class: design.Class(design.Display(design.StepFine))}, html.Text(parseProps.Saved.Name)),
		// Scope, sort key and direction are all machine facts, so the whole line is
		// one mono meta string rather than a sentence about itself.
		html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseProps.Saved.Scope+" · "+parseProps.Saved.SortKey+" "+parseProps.Saved.SortDirection)),
	)
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
		// applySavedView is a plain closure, NOT a hook. It is created once per
		// render and handed to each button as a prop, so this component's hook
		// count does not depend on how many saved views exist.
		applySavedView := func(parseState atlasListFilterState) {
			parseTransition.Start(func() {
				parseForm.Set(parseState)
				parseSearch.ReplaceAll(atlasBuildListFilterQuery(parseSearch.Values(), parseState))
			})
		}
		parseViewNodes := make([]ui.Node, 0, len(parsePayload.SavedViews))
		for _, parseSaved := range parsePayload.SavedViews {
			parseNextState := atlasSavedViewFilterState(parseSaved)
			parseViewNodes = append(parseViewNodes, ui.CreateElement(savedViewButton, savedViewButtonProps{
				Saved:     parseSaved,
				NextState: parseNextState,
				Active:    atlasSameListFilterState(parseValue, parseNextState),
				Apply:     applySavedView,
			}))
		}
		if len(parseViewNodes) == 0 {
			parseViewNodes = append(parseViewNodes, html.P(html.Props{Class: atlasProseClass()}, html.Text("No saved views yet.")))
		} else if parseTransition.Pending() {
			parseViewNodes = append(parseViewNodes, html.P(html.Props{Class: atlasMetaClass()}, html.Text("Applying view…")))
		}
		return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
			html.Div(html.Props{Class: atlasStackClass(design.Space5)},
				html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Inventory triage shell")),
				inventoryCMSSummaryBand(parsePage, parseWorkspace),
				inventoryCMSFilterForm(parseForm, parseDebounced.Pending() || parseTransition.Pending(), parseSubmit, parseTransition),
				// The old "Route action cluster" region sat here. It is gone: all three
				// of its destinations (critical lanes, promise-risk lanes, receiving)
				// are in the triage band below, which shows the same links WITH their
				// counts. Two regions of identical weight offering the same three hops
				// is the "nothing is more important than anything else" failure in
				// miniature, and the one with numbers on it wins.
				inventoryTriageBand(parseWorkspace),
				inventoryQueueTable(parseWorkspace.Summaries),
			),
			html.Div(html.Props{Class: atlasStackClass(design.Space5)},
				inventoryOperationsRail(parseShellState),
				inventorySavedViewsRail(parseViewNodes),
				savedViewForm(parsePayload),
			),
		)
	})
}

// inventoryCMSSummaryBand is the route's rollup readout.
//
// The eyebrow and the paragraph that used to open it are gone. They restated the
// route title and the route description that the console shell already prints at the
// top of every internal page — the same words, a third time, in a third weight. The
// numbers are the only thing this region knows that the shell does not.
func inventoryCMSSummaryBand(parsePage inventoryCMSPage, parseWorkspace inventoryWorkspaceSnapshot) ui.Node {
	return html.Div(html.Props{Class: atlasStackClass(design.Space5)},
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: atlasRegionClass()},
			atlasFactCluster(
				atlasFact("Visible SKUs", fmt.Sprintf("%d", len(parseWorkspace.Summaries))),
				atlasFact("Risk lanes", fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)),
				atlasFact("Inbound units", fmt.Sprintf("%d", parseWorkspace.Rollup.InboundUnits)),
				atlasFact("Available units", fmt.Sprintf("%d", parseWorkspace.Rollup.TotalAvailable)),
			),
		),
	)
}

// inventoryActionCard is one hop out of the current route.
//
// It used to be an accent-bordered box — a card inside a card — and it ended with the
// words "Open workflow" on every single instance. Thirty-seven identical trailing
// labels teach a reader to stop reading the row: if a row has nothing distinguishing
// to say, it should say nothing. The link is the affordance.
//
// What replaces the box is a rule: each action is a hairline-topped row inside the
// region that already has a Surface, so a group of them reads as a list of hops rather
// than as four objects of equal weight to the queue beside them.
func inventoryActionCard(parseTitle, parseCopy, parseHref string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: atlasRecordLinkRowClass()},
		html.Span(html.Props{Class: design.Class(design.Display(design.StepFine))}, html.Text(parseTitle)),
		html.Span(html.Props{Class: atlasProseClass()}, html.Text(parseCopy)),
	)
}

// inventoryTriageBand is the four queue counts, each one a link into the queue it
// counts. This is the route's actual navigation and the only place these three
// destinations now appear.
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
			copy:  "Already failing. Fix these first.",
			href:  "/app/inventory?status=critical",
		},
		{
			label: "Promise risk",
			value: fmt.Sprintf("%d", parseRollup.PromiseRiskLanes),
			copy:  "Will slip without attention.",
			href:  "/app/inventory?status=promise_risk",
		},
		{
			label: "Inbound pending",
			value: fmt.Sprintf("%d", parseRollup.SKUsWithInbound),
			copy:  "Stock is already on the way.",
			href:  "/app/receiving",
		},
		{
			label: "Reorder now",
			value: fmt.Sprintf("%d", parseRollup.ReorderLanes),
			copy:  "Below reorder point. Needs a vendor order.",
			href:  "/app/purchase-orders",
		},
	}
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		// Label names the queue (Display), the count is a machine fact (Data at the
		// stat-figure step), and the sentence is Prose. Three roles, one row — which is
		// the whole reason the type system is the information architecture here.
		parseNodes = append(parseNodes, html.A(html.Props{Href: parseItem.href, Class: atlasRecordLinkRowClass()},
			html.Span(html.Props{Class: design.Class(design.FieldLabel())}, html.Text(parseItem.label)),
			html.Span(html.Props{Class: design.Class(design.Data(design.StepSubhead), css.Rules(css.FontWeight.Bold))}, html.Text(parseItem.value)),
			html.Span(html.Props{Class: atlasProseClass()}, html.Text(parseItem.copy)),
		))
	}
	parseChildren := []ui.Node{atlasSectionHead("Lane pressure", "")}
	parseChildren = append(parseChildren, parseNodes...)
	return html.Div(html.Props{Class: atlasRegionClass()}, parseChildren...)
}

// inventoryQueueRowCells is the queue row's cell run, shared with the row builder.
//
// Every cell here is a machine fact, which is why the table needs no per-cell family:
// design.Table's default cell voice is already mono and tabular. The quantity columns
// opt into NumericCell so their units digits line up, and only that.
func inventoryQueueRowCells(parseItem inventorySummaryCard) ui.Node {
	return ui.Fragment(
		html.Td(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: "/app/inventory/" + parseItem.SKU, Class: atlasCellLinkClass()}, html.Text(parseItem.Title)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.SKU)),
				atlasStatusChip(parseItem.Status),
			),
		),
		html.Td(html.Props{}, html.Text(parseItem.PrimaryLane)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Available)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Inbound)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.RiskLaneCount)),
		atlasMetaCell(parseItem.LastUpdated),
	)
}

// inventoryQueueTable is the inventory queue: the densest manifest in Atlas and the
// strongest case in the app for design.Table.
//
// Two nesting levels came off here. The old shape was a bordered gradient card
// containing a bordered gradient scroll container containing the table, with a heading
// block inside the outer card. Now the heading sits in the page's own Stack and the
// table gets ONE flush surface, so it bleeds to its own hairline instead of floating
// inside 20px of gutter it never needed.
func inventoryQueueTable(parseItems []inventorySummaryCard) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, inventoryQueueTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, atlasEmptyRow(8, "No SKUs match these filters. Clear a filter or pick a saved view."))
	}
	return html.Div(html.Props{Class: atlasStackClass(design.Space3)},
		atlasSectionHead("Inventory queue", fmt.Sprintf("%d rows", len(parseItems))),
		atlasManifestTable("Inventory queue", html.Tr(html.Props{},
			atlasHeaderCell("Item", false),
			atlasHeaderCell("Primary hub", false),
			atlasHeaderCell("Risk lanes", true),
			atlasHeaderCell("Status", false),
			atlasHeaderCell("Available", true),
			atlasHeaderCell("Inbound", true),
			atlasHeaderCell("Updated", false),
			atlasHeaderCell("Actions", false),
		), parseRows),
	)
}

func inventoryQueueTableRow(parseItem inventorySummaryCard) ui.Node {
	return html.Tr(html.Props{},
		// The SKU is the row's anchor, so the identity cell is a <th>: design.Table
		// gives a row header mono weight without spending a second color on it.
		html.Th(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: "/app/inventory/" + parseItem.SKU, Class: atlasCellLinkClass()}, html.Text(parseItem.Title)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.SKU)),
			),
		),
		html.Td(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.Span(html.Props{}, html.Text(parseItem.PrimaryLane)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(fmt.Sprintf("%d of %d lanes", maxInt(parseItem.LaneCount-parseItem.RiskLaneCount, 0), parseItem.LaneCount))),
			),
		),
		// Risk lanes: the number IS the status, so it is toned instead of chipped. Zero
		// risk lanes gets no tone at all — an operator scanning this column should see
		// oxide only where there is work.
		atlasNumericStatusCell(fmt.Sprintf("%d", parseItem.RiskLaneCount), atlasRiskCountTone(parseItem.RiskLaneCount)),
		html.Td(html.Props{}, atlasStatusChip(parseItem.Status)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Available)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Inbound)),
		atlasMetaCell(parseItem.LastUpdated),
		html.Td(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: "/app/inventory/" + parseItem.SKU, Class: atlasCellLinkClass()}, html.Text("Open SKU")),
				html.A(html.Props{Href: "/app/warehouses", Class: atlasCellLinkClass()}, html.Text("Open warehouse ops")),
			),
		),
	)
}

// inventoryOperationsRail is the route rail: what the queue is currently scoped to,
// and the hops out of it.
//
// Every card in here used to carry a paragraph explaining what the rail is for
// ("Keep the route rail focused on…", "The right rail mirrors the queue scope so…").
// A rail does not need to narrate itself; the eyebrow names it and the facts under it
// do the work. All four paragraphs are gone.
func inventoryOperationsRail(parseWorkspace internalShellState) ui.Node {
	filterNodes := workspaceFilterSummaryNodes(parseWorkspace.ActiveFilters, "No filters. Every row is visible.")
	parseStatNodes := []ui.Node{}
	for _, parseItem := range parseWorkspace.WorkspaceStats {
		parseStatNodes = append(parseStatNodes, warehouseInfoRow(parseItem.Label, parseItem.Value))
	}
	if len(parseStatNodes) == 0 {
		parseStatNodes = append(parseStatNodes, html.P(html.Props{Class: atlasProseClass()}, html.Text("No workspace summary for this route yet.")))
	}
	parseViewBody := []ui.Node{html.P(html.Props{Class: atlasProseClass()}, html.Text("No saved view matches the current filters."))}
	if strings.TrimSpace(parseWorkspace.ActiveSavedView) != "" {
		parseViewBody = []ui.Node{warehouseInfoRow("Active view", parseWorkspace.ActiveSavedView)}
	}
	// Four bordered boxes become four hairline-ruled rows in one region. The label
	// names the destination and nothing repeats an "open workflow" tail.
	parseActionNodes := []ui.Node{
		html.A(html.Props{Href: "/app/inventory?status=promise_risk", Class: atlasRecordLinkRowClass()}, html.Text("Promise-risk lanes")),
		html.A(html.Props{Href: "/app/warehouses", Class: atlasRecordLinkRowClass()}, html.Text("Warehouse ops")),
		html.A(html.Props{Href: "/app/purchase-orders", Class: atlasRecordLinkRowClass()}, html.Text("Purchase orders")),
		html.A(html.Props{Href: "/app/receiving", Class: atlasRecordLinkRowClass()}, html.Text("Receiving queue")),
	}
	return html.Div(html.Props{Class: atlasStackClass(design.Space5)},
		inventoryRailCard("Current view", "", atlasFactCluster(parseStatNodes...)),
		inventoryRailCard("Active filters", "", filterNodes...),
		inventoryRailCard("Saved-view match", "", parseViewBody...),
		inventoryRailCard("Go to", "", parseActionNodes...),
	)
}

func inventorySavedViewsRail(parseViewNodes []ui.Node) ui.Node {
	return inventoryRailCard("Saved views", "", parseViewNodes...)
}

// inventoryRailCard is one rail region: an Eyebrow that names it and a Stack of
// whatever it holds. One Surface, never two.
//
// parseCopy is optional now and omitted when empty. page.go still passes explanatory
// paragraphs at its own call sites; the surfaces in this file pass "" because the
// eyebrow already says what the region is.
func inventoryRailCard(parseTitle, parseCopy string, parseChildren ...ui.Node) ui.Node {
	parseContent := []ui.Node{
		html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseTitle)),
	}
	if strings.TrimSpace(parseCopy) != "" {
		parseContent = append(parseContent, html.P(html.Props{Class: atlasProseClass()}, html.Text(parseCopy)))
	}
	parseContent = append(parseContent, parseChildren...)
	return html.Div(html.Props{Class: atlasRegionClass()}, parseContent...)
}

// workspaceFilterSummaryNodes lists the filters currently narrowing a queue.
//
// A filter label is a machine fact ("Status: promise_risk"), and an active filter is
// exactly what ToneNeutral is for — "no claim, an informational tag". The neutral chip
// is an outline, so a row of five of them stays quiet, and the set is a wrapping
// Cluster instead of a column of bordered boxes.
func workspaceFilterSummaryNodes(parseLabels []string, parseEmptyCopy string) []ui.Node {
	if len(parseLabels) == 0 {
		return []ui.Node{html.P(html.Props{Class: atlasProseClass()}, html.Text(parseEmptyCopy))}
	}
	parseNodes := make([]ui.Node, 0, len(parseLabels))
	for _, parseLabel := range parseLabels {
		parseNodes = append(parseNodes, html.Span(
			html.Props{Class: design.Class(design.StatusChip(design.ToneNeutral))},
			html.Text(parseLabel),
		))
	}
	return []ui.Node{html.Div(html.Props{Class: atlasClusterClass(design.Space2)}, parseNodes...)}
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
	return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Inventory lane workspace")),
			inventoryDetailHero(parsePage, parseRollup),
			// The rollup appears exactly once now. It used to be printed three times on
			// this route: as four pills in the hero, as four stat cards here, and again
			// as six info rows in the rail's "SKU snapshot" card.
			html.Div(html.Props{Class: atlasRegionClass()},
				atlasFactCluster(
					atlasFact("Available", fmt.Sprintf("%d units", parseRollup.TotalAvailable)),
					atlasFact("Inbound", fmt.Sprintf("%d units", parseRollup.InboundUnits)),
					atlasFact("Flagged lanes", fmt.Sprintf("%d", parseRollup.RiskLanes)),
					atlasFact("Reorder lanes", fmt.Sprintf("%d", parseRollup.ReorderLanes)),
				),
			),
			skuLaneRosterTable(parsePage.Rows),
			inventoryLaneEditorsSection(parseLaneCards),
		),
		html.Div(html.Props{Class: atlasStackClass(design.Space5)}, parseAsideChildren...),
	)
}

// inventoryDetailHero is the SKU record's head.
//
// It is design.PageHead: an eyebrow, the record name, and the 2px ink rule that closes
// it. That rule is this route's one heavy structural mark, which is what earns the
// hero its weight without a gradient, a 2rem radius or a 90px shadow.
//
// Deleted here: the sentence "…Use this route to compare pressure, edit the exact lane
// that drifted, and branch into replenishment only when balancing is no longer enough",
// which described the route rather than the SKU and said the same thing as the rail's
// "What this page controls" card and the shell header's route description. The two
// facts inside it that were real — lane count and primary hub — are now facts.
func inventoryDetailHero(parsePage inventoryDetailPage, parseRollup InventoryRollup) ui.Node {
	parsePrimaryWarehouse := fallback(parsePage.Rows[0].WarehouseName, parsePage.Rows[0].WarehouseID)
	return html.Div(html.Props{Class: atlasStackClass(design.Space4)},
		html.Div(html.Props{Class: design.Class(design.PageHead())},
			// The SKU is a machine fact, so the eyebrow slot gets Data rather than the
			// Display-role Eyebrow: an eyebrow that is an identifier should read as
			// something the system printed.
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text("SKU "+parsePage.SKU)),
			html.H2(html.Props{Class: design.Class(design.PageTitle())}, html.Text(parsePage.Title)),
		),
		html.Div(html.Props{Class: atlasSplitRowClass()},
			atlasFactCluster(
				atlasFact("Lanes", fmt.Sprintf("%d", len(parsePage.Rows))),
				atlasFact("Primary hub", parsePrimaryWarehouse),
			),
			html.A(html.Props{Href: "/app/inventory", Class: warehouseQuietButtonClass()}, html.Text("Back to inventory queue")),
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

// inventoryThresholdHistoryPanel is a COMPONENT: it calls ui.UseId and
// ui.UsePrevious.
//
// It is rendered conditionally — inventoryDetailContent only appends it when the
// threshold-history overlay route is active — so as a plain helper its two hook
// slots appeared and disappeared inside the CALLER's fiber. Opening or closing the
// overlay shifted every hook declared after it. Owning a fiber makes the panel's
// presence invisible to its parent's hook sequence, which is the whole point of the
// rule: conditional UI is fine, conditional HOOKS are not.
func inventoryThresholdHistoryPanel(parsePanel inventoryThresholdHistoryPanelPage, parsePage inventoryDetailPage) ui.Node {
	return ui.CreateElement(func() ui.Node {
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
		// Threshold history is a log: one row per change, same columns every time. As a
		// list of bordered boxes the reorder and safety numbers could not be compared
		// down the column, which is the only reason an operator opens a history at all.
		parseHistoryRows := make([]ui.Node, 0, len(parsePanel.Items))
		for _, parseItem := range parsePanel.Items {
			parseHistoryRows = append(parseHistoryRows, html.Tr(html.Props{},
				html.Th(html.Props{}, html.Text(parseItem.WarehouseID)),
				atlasNumericCell(fmt.Sprintf("%d", parseItem.ReorderPoint)),
				atlasNumericCell(fmt.Sprintf("%d", parseItem.SafetyStock)),
				atlasProseCell(
					html.Div(html.Props{Class: atlasStackClass(design.Space1)},
						html.Span(html.Props{}, html.Text(parseItem.Summary)),
						html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.Detail)),
					),
				),
				// "Updated by X" was its own line on every card; as a column the
				// preposition is the header and the cell is just the name.
				html.Td(html.Props{}, html.Text(fallback(parseItem.ActorName, "Atlas operations"))),
				atlasMetaCell(parseItem.CreatedAt),
			))
		}
		if len(parseHistoryRows) == 0 {
			parseHistoryRows = append(parseHistoryRows, atlasEmptyRow(6, "No threshold changes recorded for this SKU."))
		}
		parseHistoryTable := atlasManifestTable("Threshold changes", html.Tr(html.Props{},
			atlasHeaderCell("Hub", false),
			atlasHeaderCell("Reorder", true),
			atlasHeaderCell("Safety", true),
			atlasHeaderCell("Change", false),
			atlasHeaderCell("By", false),
			atlasHeaderCell("When", false),
		), parseHistoryRows)

		// Transfer recommendations are the lane placard's home surface: origin hub,
		// destination hub, and a posture. There are at most a handful of them, one per
		// proposed route, so they stay a stack of placards rather than a table — a table
		// of three rows would throw away the one device in this design system that an
		// operator learns to read at a glance.
		parseRecommendationNodes := make([]ui.Node, 0, len(parsePanel.Recommendations))
		for _, parseItem2 := range parsePanel.Recommendations {
			parseRecommendationNodes = append(parseRecommendationNodes, inventoryTransferRecommendation(parseItem2, parsePage.SKU))
		}
		if len(parseRecommendationNodes) == 0 {
			parseRecommendationNodes = append(parseRecommendationNodes, html.P(html.Props{Class: atlasProseClass()}, html.Text("No transfer beats tuning the threshold here.")))
		}
		// The panel head is built once. The old code built it, then rebuilt the same
		// four nodes verbatim into parseChildren[0] purely to add an id to the close
		// link, so the title and the description existed twice in the source.
		parsePanelHead := func() ui.Node {
			return html.Div(html.Props{Class: atlasSplitRowClass()},
				html.Div(html.Props{Class: atlasStackClass(design.Space1)},
					html.H2(html.Props{ID: parseTitleID, Class: design.Class(design.SectionTitle())}, html.Text("Threshold history")),
					// The overlay is deep-linkable, which is a fact about the URL, not
					// something to tell the operator in a sentence. What the panel is
					// FOR is the SKU it is about.
					html.P(html.Props{ID: parseDescriptionID, Class: atlasMetaClass()}, html.Text(parsePage.SKU)),
				),
				html.A(html.Props{ID: parseCloseID, Href: parseCloseHref, Class: warehouseQuietButtonClass()}, html.Text("Close")),
			)
		}
		parseChildren := []ui.Node{
			parsePanelHead(),
			atlasFactCluster(
				atlasFact("Changes", fmt.Sprintf("%d", len(parsePanel.Items))),
				atlasFact("Transfers suggested", fmt.Sprintf("%d", len(parsePanel.Recommendations))),
			),
			html.Div(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Threshold changes")),
			parseHistoryTable,
			html.Div(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Transfer recommendations")),
		}
		parseChildren = append(parseChildren, parseRecommendationNodes...)
		if parseChangeSummary := thresholdHistoryChangeSummary(parsePanel.Items, parsePreviousLatest, parseLatestSignature); parseChangeSummary != nil {
			parseChildren = append([]ui.Node{parseChangeSummary}, parseChildren...)
		}
		return atlasRouteSheetOverlay(parsePanelID, parseTitleID, parseDescriptionID, "#"+parseCloseID, html.Div(html.Props{
			Class: atlasStackClass(design.Space4),
			Raw: map[string]any{
				"data-atlas-route-overlay": "threshold-history",
			},
		}, parseChildren...))
	})
}

// inventoryTransferRecommendation renders one proposed stock move as a lane placard.
//
// TWO GAPS IN THE DESIGN SYSTEM show up here, and both are worth reporting rather than
// working around silently:
//
//  1. PlacardSpec has no quantity field, and placardField is unexported, so the number
//     of units — the single most important fact about a transfer recommendation —
//     cannot go on the tag. It sits underneath as a Data fact instead.
//  2. Posture has no member for a lane that has been PROPOSED but not created. The
//     four postures are ON LANE / HELD / SHORT / CLOSED, all of which describe freight
//     that already exists. PostureHeld is the least untrue of the four: the units are
//     currently sitting at the source hub and a human should know about it, which is
//     exactly what HELD means. Inventing a fifth posture belongs in the design system,
//     not in a call site.
//
// The priority string ("Promise recovery") is Atlas's own word for why this move
// matters, so it stays a chip — but on the paper BELOW the bar, not inside it. The
// placard's own posture chips are filled and tuned for the inked bar; a StatusChip
// dropped into that bar would be a low-contrast outline, which placard.go warns about
// explicitly.
func inventoryTransferRecommendation(parseItem inventoryTransferRecommendationItem, parseSKU string) ui.Node {
	return html.Div(html.Props{Class: atlasStackClass(design.Space2)},
		design.LanePlacard(design.PlacardSpec{
			// Hub codes, not slugs: "nevada-hub" set in mono at the placard's loudest
			// step reads as a URL fragment, "NV-HUB" reads as a dock tag.
			OriginHub: atlasHubCode(fallback(parseItem.SourceWarehouseID, parseItem.SourceWarehouseName)),
			DestHub:   atlasHubCode(fallback(parseItem.DestinationWarehouseID, parseItem.DestinationWarehouseName)),
			Posture:   design.PostureHeld,
			Label: fmt.Sprintf("Suggested transfer: %d units of %s from %s to %s",
				parseItem.Quantity,
				parseSKU,
				fallback(parseItem.SourceWarehouseName, parseItem.SourceWarehouseID),
				fallback(parseItem.DestinationWarehouseName, parseItem.DestinationWarehouseID),
			),
		}),
		html.Div(html.Props{Class: atlasClusterClass(design.Space3)},
			atlasFact("Move", fmt.Sprintf("%d units", parseItem.Quantity)),
			atlasFact("SKU", parseSKU),
			atlasStatusChip(fallback(parseItem.Priority, "review")),
		),
		html.P(html.Props{Class: atlasProseClass()}, html.Text(parseItem.Reason)),
	)
}

// thresholdHistoryChangeSummary announces that the timeline just gained an entry.
//
// It used to be a cyan-tinted rounded panel. Nothing has gone wrong when a log
// refreshes, so nothing here should be colored: it is now the 2px ink rule with a
// mono label and one sentence under it, which is how the rest of the app says "new
// information, here".
func thresholdHistoryChangeSummary(parseItems []inventoryThresholdHistoryItem, parsePrevious ui.Previous[string], parseLatestSignature string) ui.Node {
	if !parsePrevious.Ok() || parseLatestSignature == "" || parsePrevious.Get() == "" || parsePrevious.Get() == parseLatestSignature || len(parseItems) == 0 {
		return nil
	}
	parseLatest := parseItems[0]
	return html.Div(html.Props{Class: atlasNoticeClass()},
		html.P(html.Props{Class: design.Class(design.FieldLabel())}, html.Text("Latest threshold change")),
		html.P(html.Props{Class: atlasProseClass()}, html.Text(fmt.Sprintf("Reorder %d, safety %d, at %s.", parseLatest.ReorderPoint, parseLatest.SafetyStock, parseLatest.WarehouseID))),
	)
}

// skuLaneRosterTable is on-hand by hub: the comparison an operator opens this route to
// make. Four numeric columns down a hairline-ruled table, so a hub that is short is
// visible as a shape rather than as a number you have to hunt for.
//
// Market pressure moved out of the identity cell. It is a demand reading, not a lane
// status, and gluing it to the hub code with a pipe made a machine fact and a market
// opinion look like one string.
func skuLaneRosterTable(parseRows []inventoryRow) ui.Node {
	parseItems := make([]ui.Node, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseLaneHref := "/app/warehouses/" + parseRow.WarehouseID + "/items/" + parseRow.SKU
		parseItems = append(parseItems, html.Tr(html.Props{},
			html.Th(html.Props{},
				html.Div(html.Props{Class: atlasStackClass(design.Space1)},
					html.A(html.Props{Href: parseLaneHref, Class: atlasCellLinkClass()}, html.Text(fallback(parseRow.WarehouseName, parseRow.WarehouseID))),
					html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseRow.WarehouseID)),
				),
			),
			html.Td(html.Props{}, atlasStatusChip(parseRow.Status)),
			atlasNumericCell(fmt.Sprintf("%d", parseRow.Available)),
			atlasNumericCell(fmt.Sprintf("%d", parseRow.Inbound)),
			// Cover days: the number IS the posture. Under a week of cover is the lane
			// that will break a promise, so it is toned rather than chipped, and the
			// unit lives in the column header instead of being repeated in every cell.
			atlasNumericStatusCell(fmt.Sprintf("%d", parseRow.CoverDays), atlasCoverTone(parseRow.CoverDays)),
			atlasNumericCell(fmt.Sprintf("%d", parseRow.ReorderUnits)),
			atlasMetaCell(parseRow.MarketPressure),
			atlasMetaCell(parseRow.UpdatedAt),
			html.Td(html.Props{},
				html.A(html.Props{Href: parseLaneHref, Class: atlasCellLinkClass()}, html.Text("Open lane")),
			),
		))
	}
	if len(parseItems) == 0 {
		parseItems = append(parseItems, atlasEmptyRow(9, "No hubs stock this SKU."))
	}
	return html.Div(html.Props{Class: atlasStackClass(design.Space3)},
		atlasSectionHead("On hand by hub", fmt.Sprintf("%d lanes", len(parseRows))),
		atlasManifestTable("On hand by hub", html.Tr(html.Props{},
			atlasHeaderCell("Hub", false),
			atlasHeaderCell("Status", false),
			atlasHeaderCell("Available", true),
			atlasHeaderCell("Inbound", true),
			atlasHeaderCell("Cover (days)", true),
			atlasHeaderCell("Reorder", true),
			atlasHeaderCell("Demand", false),
			atlasHeaderCell("Updated", false),
			atlasHeaderCell("Actions", false),
		), parseItems),
	)
}

// inventoryLaneEditorsSection holds one editor per hub.
//
// The paragraph that used to introduce it explained where the section sits in the page
// ("…stay grouped under one inventory-specific section instead of scattered across
// utility cards"), which is a note about a past refactor rather than anything an
// operator needs. The editors are their own regions, so this is just the heading now —
// no outer Surface wrapping N inner surfaces.
func inventoryLaneEditorsSection(parseLaneCards []ui.Node) ui.Node {
	parseContent := []ui.Node{atlasSectionHead("Edit a lane", "")}
	parseContent = append(parseContent, parseLaneCards...)
	return html.Div(html.Props{Class: atlasStackClass(design.Space4)}, parseContent...)
}

// skuOperationsRail is the SKU route's rail.
//
// Two of its three cards are gone:
//
//   - "SKU snapshot" printed available units, inbound units, flagged lanes, lane count
//     and primary hub — all five already on screen in the hero and the rollup band.
//     Its parameters are kept in the signature because the caller's shape is not this
//     task's business, and Go is happy to ignore them.
//   - "What this page controls" was two sentences of route narration, said once in the
//     eyebrow copy and again in the paragraph under it.
func skuOperationsRail(parsePage inventoryDetailPage, parseTotalAvailable int, parseTotalInbound int, parseRiskLanes int) ui.Node {
	parsePrimaryHref := inventoryPrimaryWarehouseHref(parsePage.Rows)
	return html.Div(html.Props{Class: atlasStackClass(design.Space5)},
		inventoryRailCard("Go to", "",
			inventoryActionCard("Product record", "Catalog copy, pricing and SEO for this SKU.", "/app/products?q="+url.QueryEscape(parsePage.SKU)),
			inventoryActionCard("Primary hub item", "The warehouse-native workspace for this SKU.", parsePrimaryHref),
			inventoryActionCard("Purchase orders", "Vendor replenishment already in flight.", "/app/purchase-orders"),
			inventoryActionCard("Receiving", "Confirm inbound that has arrived.", "/app/receiving"),
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
			parseOrderNodes = append(parseOrderNodes, atlasPurchaseOrderRow(parseOrder))
		}
		if len(parseOrderNodes) == 0 {
			parseOrderNodes = append(parseOrderNodes, html.P(html.Props{Class: atlasProseClass()}, html.Text("No replenishment orders for this hub yet.")))
		}
		return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
			html.Div(html.Props{Class: atlasStackClass(design.Space5)},
				warehouseBreadcrumbBar(
					warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
					warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
					warehouseBreadcrumbLink{Label: fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID), Href: "/app/warehouses/" + parsePage.Warehouse.ID, Current: true},
				),
				warehouseDetailHero(parsePage, parseWorkspace),
				routeSummaryStrip(parsePage.Summary),
				warehouseInventoryFilterForm(parsePage.Warehouse.ID, parseForm, parseDebounced.Pending() || parseTransition.Pending(), parseSubmit, parseTransition),
				html.Div(html.Props{Class: atlasRegionClass()},
					atlasFactCluster(
						atlasFact("Items", fmt.Sprintf("%d", len(parseVisibleInventory))),
						atlasFact("Weekly demand", fmt.Sprintf("%d units", parseWorkspace.TotalDemand)),
						atlasFact("Weekly revenue", formatPrice(parseWorkspace.TotalRevenue)),
						atlasFact("Needs action", fmt.Sprintf("%d", parseWorkspace.UrgentCount)),
					),
				),
				warehouseDetailActionCluster(parsePage.Warehouse.ID),
				warehouseOpsItemPanelNode(parsePayload),
				warehouseDetailInventoryTable(parsePage.Warehouse.ID, parseVisibleInventory),
			),
			html.Div(html.Props{Class: atlasStackClass(design.Space5)},
				inventoryRailCard("Facility", "",
					atlasFactCluster(
						warehouseInfoRow("Region", parsePage.Warehouse.Region),
						warehouseInfoRow("Service level", parsePage.Warehouse.ServiceLevel),
						warehouseInfoRow("Pressure", parsePage.Warehouse.Pressure),
						warehouseInfoRow("Backlog", parsePage.Warehouse.Backlog),
						warehouseInfoRow("Staffing", parsePage.Warehouse.Staffing),
						warehouseInfoRow("Risk lanes", fmt.Sprintf("%d", parsePage.Warehouse.RiskCount)),
					),
				),
				inventoryRailCard("Active filters", "", workspaceFilterSummaryNodes(currentRouteWorkspaceState(parsePayload).ActiveFilters, "No filters. Every item is visible.")...),
				html.Div(html.Props{ID: "warehouse-create-item"}, productCreateFormWithOptions(parsePayload, productFormOptions{LockedWarehouseID: parsePage.Warehouse.ID, ReturnWarehouseID: parsePage.Warehouse.ID, IntroLabel: "Add warehouse item", SubmitLabel: "Create warehouse item"})),
				html.Div(html.Props{ID: "warehouse-replenishment"}, orderInventoryModalCard(parsePage.Inventory, parsePage.Warehouse.ID, parsePage.Warehouse.Name, parsePayload, false, "/app/warehouses/"+parsePage.Warehouse.ID)),
				inventoryRailCard("Recent purchase orders", "", parseOrderNodes...),
			),
		)
	})
}

// warehouseDetailHero is the facility record's head: PageHead, the hub's name, and the
// hub code as its eyebrow.
//
// The four pills below the title are gone. Region, service level and pressure were all
// three repeated verbatim in the rail's facility card two hundred pixels to the right,
// and the risk-lane count was repeated in the rollup band directly underneath. Focus
// stays — it is a sentence from the data, not narration about the page.
func warehouseDetailHero(parsePage warehouseInventoryDetailPage, parseWorkspace warehouseDetailWorkspaceSnapshot) ui.Node {
	return html.Div(html.Props{Class: atlasStackClass(design.Space4)},
		html.Div(html.Props{Class: design.Class(design.PageHead())},
			html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parsePage.Warehouse.ID)),
			html.H2(html.Props{Class: design.Class(design.PageTitle())}, html.Text(parsePage.Warehouse.Name)),
		),
		html.Div(html.Props{Class: atlasSplitRowClass()},
			html.P(html.Props{Class: atlasProseClass()}, html.Text(parsePage.Warehouse.Focus)),
			html.A(html.Props{Href: RouteWarehouseOps, Class: warehouseQuietButtonClass()}, html.Text("Back to hubs")),
		),
	)
}

// warehouseDetailActionCluster is the facility route's four hops out.
func warehouseDetailActionCluster(parseWarehouseID string) ui.Node {
	return html.Div(html.Props{Class: atlasRegionClass()},
		atlasSectionHead("Facility actions", ""),
		inventoryActionCard("Add an item", "Put a new managed item in this hub.", "/app/warehouses/"+parseWarehouseID+"#warehouse-create-item"),
		inventoryActionCard("Flagged lanes", "Narrow this hub to the lanes that need action.", "/app/warehouses/"+parseWarehouseID+"?status=promise_risk"),
		inventoryActionCard("Network risk", "Compare this hub against every other one.", "/app/inventory?status=promise_risk"),
		inventoryActionCard("Receiving", "Confirm inbound that has arrived.", "/app/receiving"),
	)
}

// warehouseDetailInventoryTable is the facility's item roster.
//
// Market signal and market pressure used to be glued together with a pipe inside the
// identity cell, as a third line under the title — a sentence and a category label
// sharing one string. The signal is a sentence, so it gets the one ProseCell in the
// table; the pressure is a short reading, so it is a quiet meta column.
func warehouseDetailInventoryTable(parseWarehouseID string, parseItems []inventoryRow) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, warehouseDetailInventoryTableRow(parseWarehouseID, parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, atlasEmptyRow(8, "No items match these filters. Clear a filter, or add an item from the rail."))
	}
	return html.Div(html.Props{Class: atlasStackClass(design.Space3)},
		atlasSectionHead("Items in this hub", fmt.Sprintf("%d items", len(parseItems))),
		atlasManifestTable("Items in this hub", html.Tr(html.Props{},
			atlasHeaderCell("Item", false),
			atlasHeaderCell("Status", false),
			atlasHeaderCell("Available", true),
			atlasHeaderCell("Demand (per week)", true),
			atlasHeaderCell("Revenue (per week)", true),
			atlasHeaderCell("Reorder", true),
			atlasHeaderCell("Signal", false),
			atlasHeaderCell("Actions", false),
		), parseRows),
	)
}

func warehouseDetailInventoryTableRow(parseWarehouseID string, parseItem inventoryRow) ui.Node {
	parseItemHref := "/app/warehouses/" + parseWarehouseID + "/items/" + parseItem.SKU
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: parseItemHref, Class: atlasCellLinkClass()}, html.Text(parseItem.Title)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.SKU)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(strings.Title(parseItem.Category))),
			),
		),
		html.Td(html.Props{}, atlasStatusChip(parseItem.Status)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Available)),
		// The unit moved into the header, so the cell holds only the number and the
		// column stays comparable. "%d / week" in every cell repeated the header 40
		// times and pushed the digits out of alignment.
		atlasNumericCell(fmt.Sprintf("%d", parseItem.WeeklyUnits)),
		atlasNumericCell(formatPrice(parseItem.WeeklyRevenue)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.ReorderUnits)),
		atlasProseCell(
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.Span(html.Props{}, html.Text(parseItem.MarketSignal)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.MarketPressure)),
			),
		),
		html.Td(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: parseItemHref, Class: atlasCellLinkClass()}, html.Text("Open item")),
				html.A(html.Props{Href: "/app/products/" + parseItem.Slug, Class: atlasCellLinkClass()}, html.Text("Open product")),
			),
		),
	)
}

// atlasPurchaseOrderRow is one replenishment order in a rail list: the order id and
// vendor on top, then status and ETA as machine facts. Same hairline row as every other
// record list in these surfaces.
func atlasPurchaseOrderRow(parseOrder purchaseOrderRecord) ui.Node {
	return html.A(html.Props{Href: "/app/purchase-orders/" + parseOrder.ID, Class: atlasRecordLinkRowClass()},
		html.Span(html.Props{Class: design.Class(design.Data(design.StepFine), css.Rules(css.FontWeight.Semibold))}, html.Text(parseOrder.ID)),
		html.Span(html.Props{Class: atlasProseClass()}, html.Text(parseOrder.VendorName)),
		html.Span(html.Props{Class: atlasMetaClass()}, html.Text(strings.ReplaceAll(parseOrder.Status, "_", " ")+" · ETA "+parseOrder.ETA)),
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
	// A nested route panel is still one region, so it is one Surface. The old version
	// was an accent-bordered gradient box with a 60px shadow wrapping the same content
	// the parent route already wraps — a fourth frame that said "this is different" when
	// what it actually is, is the same workspace one level down. The breadcrumb says
	// that in words.
	return html.Section(html.Props{Class: atlasStackClass(design.Space5)},
		warehouseBreadcrumbBar(
			warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
			warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
			warehouseBreadcrumbLink{Label: fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID), Href: "/app/warehouses/" + parsePage.Warehouse.ID, Current: true},
		),
		warehouseDetailHero(parsePage, parseWorkspace),
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: atlasRegionClass()},
			atlasFactCluster(
				atlasFact("Items", fmt.Sprintf("%d", len(parsePage.Inventory))),
				atlasFact("Weekly demand", fmt.Sprintf("%d units", parseWorkspace.TotalDemand)),
				atlasFact("Weekly revenue", formatPrice(parseWorkspace.TotalRevenue)),
				atlasFact("Needs action", fmt.Sprintf("%d", parseWorkspace.UrgentCount)),
			),
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
		parseOrderNodes = append(parseOrderNodes, atlasPurchaseOrderRow(parseOrder))
	}
	if len(parseOrderNodes) == 0 {
		parseOrderNodes = append(parseOrderNodes, html.P(html.Props{Class: atlasProseClass()}, html.Text("No open replenishment orders are tied to this item in the current warehouse.")))
	}
	return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
			// One region for the item's identity: eyebrow, name, summary sentence, and
			// the four facts. It used to be a bordered box inside an accent-bordered
			// gradient section, with each of the four facts in a bordered box of its own
			// — three levels of frame for one record header.
			html.Div(html.Props{Class: atlasRegionClass()},
				html.Div(html.Props{Class: atlasSplitRowClass()},
					html.Div(html.Props{Class: atlasStackClass(design.Space1)},
						html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Nested warehouse item workspace")),
						html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parsePage.Product.Title)),
					),
					html.A(html.Props{Href: parseParentPath, Class: warehouseQuietButtonClass()}, html.Text("Back to hub roster")),
				),
				html.P(html.Props{Class: atlasProseClass()}, html.Text(parsePage.Product.Summary)),
				atlasFactCluster(
					warehouseInfoRow("SKU", parsePage.Item.SKU),
					warehouseInfoRow("Hub", fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID)),
					warehouseInfoRow("Available", fmt.Sprintf("%d units", parsePage.Item.Available)),
					warehouseInfoRow("Weekly demand", fmt.Sprintf("%d units", parsePage.Item.WeeklyUnits)),
					warehouseInfoRow("Weekly revenue", formatPrice(parsePage.Item.WeeklyRevenue)),
					warehouseInfoRow("Order more", fmt.Sprintf("%d units", parsePage.Item.ReorderUnits)),
				),
				// Status is a chip, so it is not one of the facts above: a chip in a
				// label/value pair would be a status pretending to be a measurement.
				html.Div(html.Props{Class: atlasClusterClass(design.Space2)},
					atlasStatusChip(parsePage.Item.Status),
					html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parsePage.Item.MarketPressure)),
				),
			),
			internalWorkflowSection("Warehouse item admin flows", "Stay inside the warehouse workspace while you fix this lane, then jump out only when the issue truly becomes merchandising, replenishment, or receiving work.",
				internalWorkflowCard("Flow 1", "Edit stock lane", "Jump straight to the lane editor on this nested panel when the task is quantity, threshold, or status correction.", parseCurrentPath+"#warehouse-lane-editor"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product editor when the warehouse issue surfaces a summary, details, or SEO problem.", "/app/products/"+parsePage.Product.Slug),
				internalWorkflowCard("Flow 3", "Order more units", "Use the replenishment panel on this route when the fix requires vendor-side inbound, not only local edits.", parseCurrentPath+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Return to warehouse filters", "Drop back to the parent warehouse roster with the same route-owned filter context still visible.", parseParentPath),
			),
			warehouseItemNetworkTable(parseCurrentPath, parseFilters, parsePage.Warehouse.ID, parsePage.Network),
		),
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
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
		parseOrderNodes = append(parseOrderNodes, atlasPurchaseOrderRow(parseOrder))
	}
	if len(parseOrderNodes) == 0 {
		parseOrderNodes = append(parseOrderNodes, html.P(html.Props{Class: atlasProseClass()}, html.Text("No open replenishment orders are tied to this item in the current warehouse.")))
	}
	return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
			warehouseBreadcrumbBar(
				warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
				warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps},
				warehouseBreadcrumbLink{Label: fallback(parsePage.Warehouse.Name, parsePage.Warehouse.ID), Href: "/app/warehouses/" + parsePage.Warehouse.ID},
				warehouseBreadcrumbLink{Label: fallback(parsePage.Product.Title, parsePage.Item.SKU), Href: parseCurrentPath, Current: true},
			),
			warehouseFeatureCard(parsePage.Product.Title, parsePage.Product.Summary),
			// Stock figures and market figures are one region now, not two: four boxes
			// in a grid above a titled card holding four more boxes was eight frames for
			// eight numbers that an operator reads as one block.
			html.Div(html.Props{Class: atlasRegionClass()},
				atlasFactCluster(
					warehouseStatCard("Available", fmt.Sprintf("%d units", parsePage.Item.Available)),
					warehouseStatCard("Weekly demand", fmt.Sprintf("%d units", parsePage.Item.WeeklyUnits)),
					warehouseStatCard("Weekly revenue", formatPrice(parsePage.Item.WeeklyRevenue)),
					warehouseStatCard("Order more", fmt.Sprintf("%d units", parsePage.Item.ReorderUnits)),
				),
				html.Div(html.Props{Class: design.Class(design.Divider())}),
				html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Market and sales readout")),
				atlasFactCluster(
					warehouseInfoRow("Category", strings.Title(parsePage.Item.Category)),
					warehouseInfoRow("Sell-through", fmt.Sprintf("%d%%", parsePage.Item.SellThrough)),
					warehouseInfoRow("Demand score", fmt.Sprintf("%d / 100", parsePage.Item.DemandScore)),
					warehouseInfoRow("Regional share", fmt.Sprintf("%d%%", parsePage.Item.RegionalShare)),
				),
			),
			internalWorkflowSection("Warehouse item admin flows", "Complete the warehouse item job here, then jump directly into copy, replenishment, or receiving routes without retracing your steps.",
				internalWorkflowCard("Flow 1", "Edit stock lane", "Jump straight to the lane editor on this page when the task is quantity, threshold, or status correction.", parseCurrentPath+"#warehouse-lane-editor"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the product editor when the warehouse issue surfaces a summary, details, or SEO problem.", "/app/products/"+parsePage.Product.Slug),
				internalWorkflowCard("Flow 3", "Order more units", "Use the replenishment panel on this route when the fix requires vendor-side inbound, not only local edits.", parseCurrentPath+"#warehouse-replenishment"),
				internalWorkflowCard("Flow 4", "Back to warehouse roster", "Return to the parent warehouse list when you need to move from this item into the next local task.", "/app/warehouses/"+parsePage.Warehouse.ID),
			),
			warehouseItemNetworkTable(parseCurrentPath, parseFilters, parsePage.Warehouse.ID, parsePage.Network),
		),
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
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

// warehouseItemNetworkRowCells is one hub's row in the cross-network comparison.
//
// The hub the operator is currently standing in used to be marked with a cyan badge and
// cyan label text — a fourth accent hue, used to mean "you are here", competing with
// the three status colors in the same row. "You are here" is not a status, so it is now
// carried by weight and a mono "this hub" marker instead of by color, which also means
// it survives a printed manifest and a colorblind reader.
func warehouseItemNetworkRowCells(parseCurrentWarehouseID string, parseItem inventoryRow) ui.Node {
	isCurrentHub := strings.EqualFold(parseItem.WarehouseID, parseCurrentWarehouseID)
	parseIdentity := []ui.Node{
		html.Span(html.Props{Class: design.Class(design.Data(design.StepFine), css.Rules(css.FontWeight.Semibold))}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
		html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.WarehouseID)),
	}
	if isCurrentHub {
		parseIdentity = append(parseIdentity, html.Span(html.Props{Class: atlasMetaClass()}, html.Text("this hub")))
	}
	return ui.Fragment(
		html.Th(html.Props{}, html.Div(html.Props{Class: atlasStackClass(design.Space1)}, parseIdentity...)),
		html.Td(html.Props{}, atlasStatusChip(parseItem.Status)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Available)),
		atlasNumericStatusCell(fmt.Sprintf("%d", parseItem.CoverDays), atlasCoverTone(parseItem.CoverDays)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Inbound)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.WeeklyUnits)),
		atlasNumericCell(fmt.Sprintf("%d%%", parseItem.RegionalShare)),
		atlasProseCell(html.Text(parseItem.MarketSignal)),
		atlasMetaCell(parseItem.UpdatedAt),
	)
}

// warehouseItemNetworkTable compares this SKU across every hub that holds it.
//
// "Click any column header to reorder the table" is gone: a sortable header is
// discoverable by being a link, and a sentence teaching an operator to click a heading
// is the kind of copy that survives forever because nobody reads it.
func warehouseItemNetworkTable(parseCurrentPath string, parseFilters map[string]string, parseCurrentWarehouseID string, parseItems []inventoryRow) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Tr(html.Props{}, warehouseItemNetworkRowCells(parseCurrentWarehouseID, parseItem)))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, atlasEmptyRow(9, "No hubs hold this item."))
	}
	return html.Div(html.Props{Class: atlasStackClass(design.Space3)},
		atlasSectionHead("Warehouse item table", fmt.Sprintf("%d hubs", len(parseItems))),
		atlasManifestTable("This item across every hub", html.Tr(html.Props{},
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "warehouse", "Hub", false),
			atlasHeaderCell("Status", false),
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "available", "Available", true),
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "cover", "Cover (days)", true),
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "inbound", "Inbound", true),
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "demand", "Demand (per week)", true),
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "share", "Regional share", true),
			atlasHeaderCell("Signal", false),
			warehouseItemTableHeader(parseCurrentPath, parseFilters, "updated", "Updated", false),
		), parseRows),
	)
}

type warehouseBreadcrumbLink struct {
	Label   string
	Href    string
	Current bool
}

// warehouseBreadcrumbBar is the route trail.
//
// It is no longer a bordered bar. A breadcrumb is a line of text about where you are,
// not an object on the page, and boxing it made it compete with the page head two
// pixels below it. The separator is a graphite slash and is aria-hidden — read aloud,
// "Dashboard slash Warehouses slash Illinois Hub" is worse than the labels alone.
func warehouseBreadcrumbBar(parseLinks ...warehouseBreadcrumbLink) ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseLinks)*2)
	for parseIndex, parseLink := range parseLinks {
		parseLabel := fallback(strings.TrimSpace(parseLink.Label), "Route")
		if parseIndex > 0 {
			parseNodes = append(parseNodes, html.Span(
				html.Props{Class: atlasMetaClass(), Aria: map[string]string{"hidden": "true"}},
				html.Text("/"),
			))
		}
		if parseLink.Current || strings.TrimSpace(parseLink.Href) == "" {
			parseNodes = append(parseNodes, html.Span(
				html.Props{Class: design.Class(design.Display(design.StepMicro)), Aria: map[string]string{"current": "page"}},
				html.Text(parseLabel),
			))
			continue
		}
		parseNodes = append(parseNodes, html.A(
			html.Props{Href: parseLink.Href, Class: design.Class(design.Display(design.StepMicro), design.Link())},
			html.Text(parseLabel),
		))
	}
	return html.Div(html.Props{Class: atlasClusterClass(design.Space2), Aria: map[string]string{"label": "Breadcrumb"}, Role: "navigation"}, parseNodes...)
}

// warehouseItemTableHeader is a sortable column header.
//
// design.Table already sets the header's voice, so this only adds the link, the
// direction indicator and — for a numeric column — NumericCell, so the header aligns
// with the digits under it. The arrow is aria-hidden because it is a picture of the
// sort direction, and the direction is already in the link's own URL.
func warehouseItemTableHeader(parseCurrentPath string, parseFilters map[string]string, parseSortKey string, parseLabel string, isNumeric bool) ui.Node {
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
	if parseActive {
		if strings.EqualFold(strings.TrimSpace(parseFilters["dir"]), "asc") {
			parseIndicator = "↑"
		} else {
			parseIndicator = "↓"
		}
	}
	parseChildren := []ui.Node{html.Text(parseLabel)}
	if parseIndicator != "" {
		parseChildren = append(parseChildren, html.Span(
			html.Props{Aria: map[string]string{"hidden": "true"}},
			html.Text(parseIndicator),
		))
	}
	parseHeaderProps := html.Props{}
	if isNumeric {
		parseHeaderProps.Class = design.Class(design.NumericCell())
	}
	if parseActive {
		parseHeaderProps.Aria = map[string]string{"sort": atlasAriaSortValue(parseFilters["dir"])}
	}
	return html.Th(parseHeaderProps, html.A(html.Props{
		Href:   parseCurrentPath + "?" + parseValues.Encode(),
		Target: "_self",
		Class:  atlasColumnSortLinkClass(parseActive),
	}, parseChildren...))
}

// atlasAriaSortValue maps Atlas's dir query value onto aria-sort's vocabulary, so the
// sorted column is announced rather than only drawn.
func atlasAriaSortValue(parseDirection string) string {
	if strings.EqualFold(strings.TrimSpace(parseDirection), "asc") {
		return "ascending"
	}
	return "descending"
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

// inventoryCMSFilterForm narrows the queue.
//
// A filter bar is INPUT to the page rather than output of it, which is exactly what
// design.Recess is for: pressed paper, no border, no radius. That is the whole reason
// it does not need to be a card — and the paragraph explaining that the filters are
// "deep-linkable state" is gone, because the URL demonstrates that by itself.
func inventoryCMSFilterForm(parseForm ui.Form[atlasListFilterState], isSyncing bool, parseSubmit ui.Handler, parseTransition atlasTransition) ui.Node {
	parseValue := parseForm.Get()
	return html.Form(html.Props{Action: "/app/inventory", Method: "get", OnSubmit: parseSubmit, Class: design.Class(design.Recess()), Aria: map[string]string{"label": "Filter inventory"}, Role: "search"},
		html.Div(html.Props{Class: atlasFilterRowClass()},
			cmsTransitionBoundTextInput("q", "Search inventory", parseValue.Query, "Query", parseForm, parseTransition),
			cmsTransitionBoundSelectInput("warehouse", "Warehouse", parseValue.Warehouse, "Warehouse", append([]optionItem{{"all", "All warehouses"}}, warehouseOptions()...), parseForm, parseTransition),
			cmsTransitionBoundSelectInput("status", "Lane status", parseValue.Status, "Status", append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...), parseForm, parseTransition),
			cmsTransitionBoundSelectInput("sort", "Sort", parseValue.Sort, "Sort", []optionItem{{"updated", "Updated recently"}, {"inbound", "Highest inbound"}}, parseForm, parseTransition),
			html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text(atlasFilterSubmitLabel(isSyncing, "Filter"))),
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
			// ButtonSecondary, not primary: there is one of these editors per hub, and
			// the design system's one-primary-per-view rule is not a style preference —
			// four yellow buttons on one route means the route has no primary action.
			html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text("Save lane")),
		}
		if strings.TrimSpace(parseValue.ReturnPath) != "" {
			parseChildren = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_path", Value: parseValue.ReturnPath})}, parseChildren...)
		}
		// One Surface for the lane editor, and a Divider between "what this lane is" and
		// "what you can change about it". That rule replaces what used to be an inset
		// bordered card holding three more bordered info boxes plus the form.
		return html.Div(html.Props{Class: atlasRegionClass()},
			html.Div(html.Props{Class: atlasSplitRowClass()},
				html.Div(html.Props{Class: atlasStackClass(design.Space1)},
					html.P(html.Props{Class: design.Class(design.Display(design.StepFine))}, html.Text(parseWarehouseLabel)),
					html.P(html.Props{Class: atlasMetaClass()}, html.Text(parseRow.WarehouseID+" · updated "+parseRow.UpdatedAt)),
				),
				atlasStatusChip(parseValue.Status),
			),
			atlasFactCluster(
				warehouseInfoRow("Available", fmt.Sprintf("%d units", parseRow.Available)),
				warehouseInfoRow("Inbound", fmt.Sprintf("%d units", parseRow.Inbound)),
				warehouseInfoRow("Cover", fmt.Sprintf("%d days", parseRow.CoverDays)),
			),
			html.Div(html.Props{Class: design.Class(design.Divider())}),
			html.Form(html.Props{Action: "/api/app/inventory/" + parseRow.SKU + "/update", Method: "post", Class: atlasFormGridClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...),
		)
	})
}

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
	parseDescription := "Creates a purchase order and puts inbound units on the hub lane you pick."
	if isSkuScoped {
		parseButtonLabel = "Order replenishment"
		parseDescription = "Creates a purchase order for this SKU without leaving the editor."
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
				// The locked item is a read-only fact, so it is a label/value pair like
				// any other fact — not a bordered pseudo-field pretending to be an input
				// the operator can edit.
				atlasFact("Item", parseTitle+" · "+parseValue.ProductSKU),
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
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space3), css.Rules(css.Justify.End))},
				html.Button(html.Props{Type: "button", Class: warehouseQuietButtonClass(), OnClick: parseCloseModal}, html.Text("Cancel")),
				// The one honest ButtonPrimary in these surfaces. A dialog IS a view, and
				// this is the only thing to do in it.
				html.Button(html.Props{Type: "submit", Class: warehousePrimaryButtonClass()}, html.Text("Create order")),
			),
		)
		parseModalSurfaceID := parseModalID + "-surface"
		parseModalCloseID := parseModalID + "-close"
		useAtlasFocusContainment(parseOpen.Get(), "#"+parseModalSurfaceID, "#"+parseModalCloseID)
		parseChildren := []ui.Node{
			html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Replenishment")),
			html.P(html.Props{Class: atlasProseClass()}, html.Text(parseDescription)),
			html.Button(html.Props{Type: "button", Class: warehouseSecondaryButtonClass(), OnClick: parseOpenModal}, html.Text(parseButtonLabel)),
		}
		if parseOpen.Get() {
			parseChildren = append(parseChildren, html.Div(html.Props{Class: atlasModalScrimClass()},
				html.Button(html.Props{Type: "button", Class: atlasModalScrimButtonClass(), OnClick: parseCloseModal, Aria: map[string]string{"label": "Close replenishment dialog"}}),
				html.Div(html.Props{
					ID:    parseModalSurfaceID,
					Class: atlasModalSurfaceClass(),
					Role:  "dialog",
					// tabindex="-1" so the dialog can take focus by script without
					// entering the tab order. -1 is expressible in the typed field; only
					// a literal zero needs the sentinel.
					TabIndex: -1,
					Aria:     map[string]string{"modal": "true"},
				},
					html.Div(html.Props{Class: atlasSplitRowClass()},
						html.Div(html.Props{Class: atlasStackClass(design.Space1)},
							// The dialog's own head says the item and what will happen.
							// It used to repeat the eyebrow, the title AND the same
							// description already printed on the launcher behind it.
							html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseTitle)),
							html.P(html.Props{Class: atlasProseClass()}, html.Text(parseDescription)),
						),
						html.Button(html.Props{ID: parseModalCloseID, Type: "button", Class: warehouseQuietButtonClass(), OnClick: parseCloseModal}, html.Text("Close")),
					),
					html.Form(html.Props{Action: "/api/app/purchase-orders", Method: "post", Class: atlasFormGridClass()}, parseFormChildren...),
				),
			))
		}
		return html.Div(html.Props{Class: atlasRegionClass()}, parseChildren...)
	})
}

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
	parseEdited := "Staged from the current defaults."
	if strings.TrimSpace(parseState.LastEditedField) != "" {
		parseEdited = "Last edited: " + strings.ReplaceAll(parseState.LastEditedField, "_", " ")
	}
	// The draft's stage IS a status, so it is a chip with a tone rather than a tinted
	// panel: "blocked" needs a human (exception), everything else is either awaited or
	// simply not a claim yet. inventoryStatusTone owns that decision.
	return html.Div(html.Props{Class: atlasWorkflowNoticeClass()},
		html.Div(html.Props{Class: atlasClusterClass(design.Space2)},
			atlasStatusChip(atlasWorkflowStageStatus(parseState.Stage)),
			html.Span(html.Props{Class: design.Class(design.Display(design.StepFine))}, html.Text(parseStageLabel)),
		),
		html.P(html.Props{Class: atlasProseClass()}, html.Text(parseStageCopy)),
		html.P(html.Props{Class: atlasMetaClass()}, html.Text(parseEdited)),
	)
}

// atlasWorkflowStageStatus maps a purchase-order draft stage onto a status word that
// inventoryStatusTone already knows how to tone. "blocked" is the only stage that needs a
// human, so it is the only one that reaches oxide; a draft that is merely missing a
// field is neutral, because nothing has gone wrong yet.
func atlasWorkflowStageStatus(parseStage string) string {
	switch parseStage {
	case "blocked":
		return "critical"
	case "inbound-confirmed":
		return "approved"
	case "ready-to-submit":
		return "submitted"
	default:
		return "draft"
	}
}

// atlasWorkflowNoticeClass is the draft-state strip inside the replenishment form: a
// recessed band that spans both form columns, because it describes the whole form
// rather than any one field.
func atlasWorkflowNoticeClass() string {
	return design.Class(
		design.Recess(),
		design.Stack(design.Space2),
		css.Rules(css.Media(css.MinW(768), css.Raw("grid-column", "1 / -1"))),
	)
}

func inventoryReducerTextField(parseName, parseLabel, parseValue string, parseWorkflow ui.Reducer[purchaseOrderWorkflowState, purchaseOrderWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
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
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{
			// A quantity is a machine fact, so the control is the mono/tabular variant.
			ID: parseId, Type: "number", Name: parseName, Value: parseValue, Class: warehouseDataInputClass(),
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
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Select(html.Props{
			ID: parseId, Name: parseName, Class: warehouseInputClass(),
			OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) {
				parseWorkflow.Dispatch(purchaseOrderWorkflowAction{Field: parseName, Value: parseEvent.GetValue()})
			}),
			Raw: map[string]any{"aria-labelledby": parseId + "-label"},
		}, parseChildren...),
	)
}

// warehouseInventoryFilterForm narrows one hub's roster. Same shape as the inventory
// filter bar: a Recess, because it is input to the page.
func warehouseInventoryFilterForm(parseWarehouseID string, parseForm ui.Form[atlasListFilterState], isSyncing bool, parseSubmit ui.Handler, parseTransition atlasTransition) ui.Node {
	parseValue := parseForm.Get()
	return html.Form(html.Props{Action: "/app/warehouses/" + parseWarehouseID, Method: "get", OnSubmit: parseSubmit, Class: design.Class(design.Recess()), Aria: map[string]string{"label": "Filter items in this hub"}, Role: "search"},
		html.Div(html.Props{Class: atlasFilterRowClass()},
			cmsTransitionBoundTextInput("q", "Search items", parseValue.Query, "Query", parseForm, parseTransition),
			cmsTransitionBoundSelectInput("status", "Lane status", parseValue.Status, "Status", append([]optionItem{{"all", "All statuses"}}, inventoryStatusOptions()...), parseForm, parseTransition),
			cmsTransitionBoundSelectInput("sort", "Sort", parseValue.Sort, "Sort", []optionItem{{"updated", "Updated"}, {"available", "Lowest available"}, {"demand", "Highest demand"}, {"revenue", "Highest revenue"}}, parseForm, parseTransition),
			html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text(atlasFilterSubmitLabel(isSyncing, "Apply"))),
		),
	)
}

// marketPressureClass styles a market-pressure reading, and it now returns the same
// quiet meta class for every value.
//
// It used to return four different saturated palettes — rose for "hot market", amber
// for "growing demand", slate for "softening", emerald for everything else. That is the
// free-form color escape hatch the design system deliberately does not have, and it was
// spending three hues on a reading that is not a status: nothing has gone wrong when a
// market is hot, and nothing has been checked and passed when it is steady. Per
// design/status.go, a state that does not map to one of the four tones is not a status
// and should be graphite text — so that is what every label gets.
//
// The parameter stays so existing callers keep compiling; it is intentionally unused.
func marketPressureClass(parseLabel string) string {
	return atlasMetaClass()
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

// The inventory field helpers all use design.Field (label above control) and pick
// between design.Input and design.InputData by what the field HOLDS: words go in the
// proportional control, machine facts in the mono tabular one. That is the same
// prose/data distinction the type roles encode, pushed into the form layer.

func inventoryTextField(parseName, parseLabel, parseValue string) ui.Node {
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{Name: parseName, Value: parseValue, Class: warehouseInputClass()}),
	)
}

func inventoryNumberField(parseName, parseLabel, parseValue string) ui.Node {
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{Type: "number", Name: parseName, Value: parseValue, Class: warehouseDataInputClass()}),
	)
}

func inventoryBoundNumberField[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Type: "number", Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseDataInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func inventorySelectField(parseName, parseLabel, parseValue string, parseOptions []optionItem) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value))}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Select(html.Props{Name: parseName, Class: warehouseInputClass()}, parseChildren...),
	)
}

func inventoryBoundSelectField[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value))}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Select(html.Props{ID: parseId, Name: parseName, OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, parseChildren...),
	)
}

// --- design system bridge -----------------------------------------------------
//
// Everything below folds shared/design bundles into class strings for the inventory,
// warehouse and catalog surfaces. Four rules are worth stating once, because the
// markup these helpers replaced broke all four:
//
//  1. ONE Surface per region. The old markup nested rounded bordered boxes up to four
//     deep — surface card > inset surface > info row > badge — every level with the
//     same border, radius and shadow, so nothing read as more important than anything
//     else. When something inside a region needs separating it gets a hairline
//     (design.Divider, or a border-top on a repeating row) or a Stack gap. The design
//     system ships no nested-card primitive on purpose; see design/surfaces.go.
//  2. Machine facts are design.Data — mono, tabular: SKU, hub code, quantity,
//     threshold, cover days, ETA, price, ids. Sentences are design.Prose.
//     design.Display only ever NAMES a region: page and section titles, eyebrows,
//     column headers, button labels.
//  3. Queues are tables, not card lists. Inventory is the densest case in the app, so
//     every roster in these files goes through design.Table: hairline rows, mono
//     cells, right-aligned numerics, zebra from the recessed paper token.
//  4. Status is a tone, never a color (inventoryStatusTone). There is no color argument
//     anywhere in the design system, which is what stops the three unrelated accent
//     hues from growing back.
//
// These are functions rather than package-level vars deliberately. design.Class folds
// lazily and css.New memoizes on the canonical rule-set, so a repeat call is a map
// lookup — while an init-time fold would hand out class names whose CSS a css.Reset()
// in a test had already thrown away.
//
// KNOWN DUPLICATION, to be collapsed once the whole console conversion has landed:
// page.go grew a parallel console*Class() bridge over the same design bundles during
// the same pass. It costs nothing at runtime — two helpers that fold identical rule
// sets get the SAME hashed class out of css.New, so there is no duplicated CSS — but
// the package should end up with one bridge, not two. The console* set is the more
// complete one; the primitives worth keeping from this side are the ones the design
// system does not cover at all: atlasWorkspaceSplitClass, atlasFormGridClass,
// atlasFilterRowClass, atlasRecordRowClass and the modal shell.

// atlasRegionClass is one page region: a single flat Surface with vertical rhythm
// inside it. If a second one feels necessary inside the first, the region is doing two
// jobs and wants to be two siblings instead.
func atlasRegionClass() string {
	return design.Class(design.Surface(), design.Stack(design.Space4))
}

// atlasStackClass and atlasClusterClass are the two spacing answers: down the page and
// across it. Cluster wraps, which is most of why these surfaces survive a narrow
// viewport without per-component media queries.
func atlasStackClass(parseGap css.Length) string {
	return design.Class(design.Stack(parseGap))
}

func atlasClusterClass(parseGap css.Length) string {
	return design.Class(design.Cluster(parseGap))
}

// atlasSplitRowClass is a title on the left and its actions or count on the right,
// wrapping instead of overflowing.
func atlasSplitRowClass() string {
	return design.Class(design.SplitRow(design.Space3))
}

// atlasProseClass is a sentence, capped at a readable measure. The console's content
// column is uncapped on purpose (tables want the width), so the cap belongs on the
// prose — see design.Measure.
func atlasProseClass() string {
	return design.Class(design.Prose(design.StepFine), design.Measure())
}

// atlasMetaClass is a quiet machine fact beside something else: a row count, an
// updated-at stamp, a hub code under a name. Graphite is the only de-emphasis tool in
// the design system; there is no opacity scale, because translucent text over a zebra
// stripe has a different contrast on every other row.
func atlasMetaClass() string {
	return design.Class(design.Data(design.StepMicro), css.Rules(css.TextColor(design.Graphite())))
}

// atlasFact is a label above a machine fact: Display micro for the label (the same
// voice as a column header, because they do the same job) and mono for the value.
//
// This is what four "stat cards" collapse into. The label/value pair carries the same
// information as the box did, and a Cluster of six of them reads as a manifest block
// rather than as six competing objects.
func atlasFact(parseLabel string, parseValue string) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Span(html.Props{Class: design.Class(design.Data(design.StepLede))}, html.Text(parseValue)),
	)
}

// atlasFactCluster lays a row of facts out with section-level spacing.
func atlasFactCluster(parseFacts ...ui.Node) ui.Node {
	return html.Div(html.Props{Class: atlasClusterClass(design.Space5)}, parseFacts...)
}

// atlasWorkspaceSplitClass is the console's "main pane plus route rail" layout.
//
// GAP IN THE DESIGN SYSTEM: design ships ConsoleShell (rail + content) and
// ContentColumn, but nothing for the two-column split that every internal workspace
// route in Atlas uses inside the content column. It is authored here, from design
// tokens only, rather than smuggled in as Tailwind: one column until there is real
// room, then content and rail, top-aligned. minmax(0,…) on both tracks is load-bearing
// — a bare 1fr has an auto minimum, so one long mono SKU sets the column's min-content
// width and the whole page grows a horizontal scrollbar.
func atlasWorkspaceSplitClass() string {
	return design.Class(css.Rules(
		css.Display.Grid,
		css.Gap(design.Space5),
		css.MinWidth(css.Zero),
		css.Media(css.MinW(1180),
			css.GridCols(
				css.MinMax(css.TrackLen(css.Zero), css.Fr(1.26)),
				css.MinMax(css.TrackLen(css.Rem(22)), css.Fr(0.74)),
			),
			css.Items.Start,
		),
	))
}

// atlasFormGridClass is a form's field grid: one column narrow, two when there is
// room. Fields themselves are design.Field (label above control) — this only decides
// how many fit side by side.
func atlasFormGridClass() string {
	return design.Class(css.Rules(
		css.Display.Grid,
		css.Gap(design.Space4),
		css.MinWidth(css.Zero),
		css.Media(css.MinW(768), css.GridCols(css.Repeat(2, css.MinMax(css.TrackLen(css.Zero), css.Fr(1))))),
	))
}

// atlasTextareaClass is design.Input with a usable minimum height. A textarea holds
// sentences, so it is the proportional control, never the mono one.
func atlasTextareaClass() string {
	return design.Class(design.Input(), css.Rules(css.MinHeight(css.Rem(7))))
}

// atlasFormSectionClass separates one group of fields from the next with a hairline and
// space — no border, no radius, no background. A field group is a run of fields with a
// name, not an object on the page.
func atlasFormSectionClass() string {
	return design.Class(css.Rules(
		design.Stack(design.Space3),
		css.BorderTop(design.HairlineWidth, design.Hairline()),
		css.Raw("padding-top", string(design.Space4)),
	))
}

// atlasFilterRowClass lays a filter bar out as controls that wrap, with the submit
// button on the same baseline. A filter bar is INPUT to the page rather than output of
// it, so its callers put it in a Recess — pressed paper, no border, no radius — which
// is the design system's way of marking a region without adding another box.
func atlasFilterRowClass() string {
	return design.Class(css.Rules(
		design.Cluster(design.Space4),
		css.Items.End,
		css.Descendant(css.El("label"), css.Raw("flex", "1 1 12rem")),
	))
}

// atlasRecordRowClass is one entry in a list of records: a hairline above it, a Stack
// inside it, and no border on the other three sides.
//
// This is the direct replacement for "every row is a bordered box". A rule costs one
// pixel and says "these are entries in the same list"; a box costs a border, a radius
// and two gutters and says "these are separate objects". Inside a list the first
// statement is the true one.
func atlasRecordRowClass() string {
	return design.Class(css.Rules(
		design.Stack(design.Space1),
		css.BorderTop(design.HairlineWidth, design.Hairline()),
		css.PaddingY(design.Space3),
	))
}

// atlasRecordLinkRowClass is atlasRecordRowClass for a row that is itself a link: the
// whole row is the target, the text keeps the surrounding ink color instead of turning
// into a wall of blue, and hover presses the paper rather than adding a frame.
func atlasRecordLinkRowClass() string {
	return design.Class(css.Rules(
		design.Stack(design.Space1),
		css.BorderTop(design.HairlineWidth, design.Hairline()),
		css.PaddingY(design.Space3),
		css.Raw("text-decoration", "none"),
		css.TextColor(design.Ink()),
		css.Hover(css.Bg(design.PaperSunk()), css.TextColor(design.Lane())),
	))
}

// atlasSectionHead names a region and optionally states how many rows are in it.
//
// Section titles are the design system's second and last heading level. There is no
// third: a page that needs one needs two regions instead.
func atlasSectionHead(parseTitle string, parseCount string) ui.Node {
	parseChildren := []ui.Node{
		html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseTitle)),
	}
	if strings.TrimSpace(parseCount) != "" {
		parseChildren = append(parseChildren, html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseCount)))
	}
	return html.Div(html.Props{Class: atlasSplitRowClass()}, parseChildren...)
}

// atlasManifestTable wraps a table in the two things a dense table needs and almost
// never gets: a surface that lets it bleed to its own hairline instead of floating
// inside a gutter, and a keyboard-reachable horizontal scroll region.
//
// The tabindex goes through the html.TabIndexZero sentinel rather than TabIndex: 0,
// because a literal zero is Go's "field not set" and emits no attribute at all — which
// is exactly the value a scroll region needs, and exactly the bug that makes an
// overflowing manifest unreachable without a mouse.
func atlasManifestTable(parseLabel string, parseHead ui.Node, parseRows []ui.Node) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
		html.Div(html.Props{
			Class:    design.Class(design.TableScroll()),
			Role:     "region",
			TabIndex: html.TabIndexZero,
			Aria:     map[string]string{"label": parseLabel},
		},
			html.Table(html.Props{Class: design.Class(design.Table())},
				html.Thead(html.Props{}, parseHead),
				html.Tbody(html.Props{}, parseRows...),
			),
		),
	)
}

// atlasHeaderCell is a column header. design.Table already styles thead th as Display
// micro graphite closed by the heavy ink rule, so a header needs no class of its own —
// unless it heads a numeric column, in which case it MUST carry NumericCell too: a
// right-aligned column under a left-aligned header reads as broken.
func atlasHeaderCell(parseLabel string, isNumeric bool) ui.Node {
	if isNumeric {
		return html.Th(html.Props{Class: design.Class(design.NumericCell())}, html.Text(parseLabel))
	}
	return html.Th(html.Props{}, html.Text(parseLabel))
}

// atlasNumericCell is a quantity, count or price: right-aligned tabular mono, so the
// units digit of every number in the column lines up and magnitude reads as a shape.
func atlasNumericCell(parseValue string) ui.Node {
	return html.Td(html.Props{Class: design.Class(design.NumericCell())}, html.Text(parseValue))
}

// atlasNumericStatusCell is a numeric cell whose NUMBER is the status — a negative
// on-hand, a zero cover, a variance. StatusValue tones the figure itself instead of
// parking a chip next to it, because a chip beside a number that already says "-4" is
// redundant and a dense manifest cannot afford redundancy.
func atlasNumericStatusCell(parseValue string, parseTone design.StatusTone) ui.Node {
	return html.Td(
		html.Props{Class: design.Class(design.NumericCell(), design.StatusValue(parseTone))},
		html.Text(parseValue),
	)
}

// atlasRiskCountTone tones a count of flagged lanes.
//
// Zero is NEUTRAL, not verified: a zero in a risk column is the absence of a finding,
// not a finding that passed, and toning it green would put color on 90% of the rows in
// a healthy queue. Any positive count is an EXCEPTION, because a flagged lane is by
// definition a lane a human already needs to look at — this is the one column in the
// queue where the filled tone is earned.
func atlasRiskCountTone(parseCount int) design.StatusTone {
	if parseCount > 0 {
		return design.ToneException
	}
	return design.ToneNeutral
}

// atlasModalScrimClass, atlasModalScrimButtonClass and atlasModalSurfaceClass are the
// replenishment dialog's shell.
//
// GAP IN THE DESIGN SYSTEM: there is no modal or dialog primitive, so the overlay is
// authored here — from design tokens, and with the design system's own constraints kept
// (2px radius, hairline border, no shadow scale, no gradient). The scrim is the only
// translucency in these surfaces and it is unavoidable: a dialog has to dim what it
// covers. If a Dialog primitive ever lands, this is what it should replace.
func atlasModalScrimClass() string {
	return design.Class(css.Rules(
		css.Position.Fixed,
		css.Raw("inset", "0"),
		css.Raw("z-index", "50"),
		css.Display.Flex,
		css.Items.Center,
		css.Justify.Center,
		css.Padding(design.Space4),
		css.Raw("background-color", "color-mix(in srgb, var("+design.TokenInk+") 55%, transparent)"),
	))
}

func atlasModalScrimButtonClass() string {
	return design.Class(css.Rules(
		css.Position.Absolute,
		css.Raw("inset", "0"),
		css.Bg(css.Transparent),
		css.Raw("border", "none"),
		css.Cursor.Pointer,
	))
}

func atlasModalSurfaceClass() string {
	return design.Class(
		design.Surface(),
		design.Stack(design.Space4),
		css.Rules(
			css.Position.Relative,
			css.Raw("z-index", "10"),
			css.W(css.RawLength("min(92vw, 40rem)")),
			css.Raw("max-height", "86vh"),
			css.Raw("overflow-y", "auto"),
		),
	)
}

// atlasCoverTone tones a cover-days figure, which is the one number on a lane that
// says whether a promise can be kept.
//
// Zero or negative cover is an EXCEPTION: there is nothing on the shelf, so the next
// order fails. One to six days is PENDING — the system is still covering, but a human
// should know. A week or more is NEUTRAL rather than verified, deliberately: healthy
// cover is the normal case, and a column of green numbers would be color spent on the
// rows nobody needs to look at.
func atlasCoverTone(parseCoverDays int) design.StatusTone {
	switch {
	case parseCoverDays <= 0:
		return design.ToneException
	case parseCoverDays < 7:
		return design.TonePending
	default:
		return design.ToneNeutral
	}
}

// atlasSavedViewButtonClass is a saved-view row: a hairline-topped list entry that
// happens to be a <button>, not a bordered pill.
//
// The selected view is marked with the 2px ink-weight rule on its leading edge in lane
// blue — structural, "the system is showing you this one" — and the base keeps a
// transparent leading border so selecting a view does not shift the text by 2px.
func atlasSavedViewButtonClass(isActive bool) string {
	parseBundle := css.Rules(
		design.Stack(design.Space1),
		css.W(css.Full),
		css.Bg(css.Transparent),
		css.BorderTop(design.HairlineWidth, design.Hairline()),
		css.BorderLeft(design.ManifestRuleWidth, css.Transparent),
		css.PaddingY(design.Space3),
		css.PaddingX(design.Space2),
		css.Raw("text-align", "left"),
		css.Cursor.Pointer,
		css.TextColor(design.Ink()),
		css.Hover(css.Bg(design.PaperSunk())),
		css.FocusVisible(css.Outline(design.ManifestRuleWidth, design.Lane()), css.OutlineOffset(css.Px(2))),
	)
	if isActive {
		return design.Class(parseBundle, css.Rules(
			css.Bg(design.PaperSunk()),
			css.BorderLeft(design.ManifestRuleWidth, design.Lane()),
		))
	}
	return design.Class(parseBundle)
}

// atlasMetaCell is context rather than content: an updated-at stamp, a unit.
func atlasMetaCell(parseValue string) ui.Node {
	return html.Td(html.Props{Class: design.Class(design.CellMeta())}, html.Text(parseValue))
}

// atlasProseCell is the one documented exception to "every table cell is mono": a cell
// holding a sentence, such as an operator note or a rejection reason.
func atlasProseCell(parseChildren ...ui.Node) ui.Node {
	return html.Td(html.Props{Class: design.Class(design.ProseCell())}, parseChildren...)
}

// atlasEmptyRow is a table's empty state. It spans the whole manifest and says what to
// do next in one sentence.
func atlasEmptyRow(parseColumns int, parseCopy string) ui.Node {
	return html.Tr(html.Props{},
		html.Td(html.Props{Class: design.Class(design.ProseCell()), ColSpan: parseColumns}, html.Text(parseCopy)),
	)
}

// atlasCellLinkClass is a link inside a table cell. It inherits the cell's family and
// size (a link is not a different KIND of string) and stays underlined, because in a
// manifest full of mono identifiers color alone does not distinguish a link from a
// status value.
func atlasCellLinkClass() string {
	return design.Class(css.Rules(
		css.TextColor(design.Lane()),
		css.Raw("text-decoration", "underline"),
		css.Raw("text-underline-offset", "0.18em"),
		css.Raw("text-decoration-thickness", "1px"),
		css.Hover(css.TextColor(design.Ink()), css.Raw("text-decoration-thickness", "2px")),
	))
}

// atlasColumnSortLinkClass is the sortable column header link. It keeps the header's
// own voice (inherit, not lane blue) so seven sortable headers do not read as seven
// links, and it earns its underline only on hover and focus.
func atlasColumnSortLinkClass(isActive bool) string {
	parseBundle := css.Rules(
		css.Display.InlineFlex,
		css.Items.Center,
		css.Gap(design.Space1),
		css.Raw("color", "inherit"),
		css.Raw("text-decoration", "none"),
		css.Hover(css.TextColor(design.Lane()), css.Raw("text-decoration", "underline")),
	)
	if isActive {
		return design.Class(parseBundle, css.Rules(css.TextColor(design.Ink()), css.FontWeight.Bold))
	}
	return design.Class(parseBundle)
}

// atlasNoticeClass is the one-line "something just changed" strip: a hairline rule
// above the sentence rather than a tinted box.
//
// The old version was a cyan-tinted rounded panel, which is the ornamental use of a
// saturated hue the design system exists to prevent — nothing has gone wrong when a
// timeline refreshes, so nothing should be colored.
func atlasNoticeClass() string {
	return design.Class(css.Rules(
		design.Stack(design.Space1),
		css.BorderTop(design.ManifestRuleWidth, design.Ink()),
		css.PaddingY(design.Space2),
	))
}

// warehouseFeatureCard names a record and says one sentence about it.
//
// It used to be a gradient-filled bordered box carrying a hard-coded "Warehouse
// operations" eyebrow above the title. The eyebrow stays because it names the region
// (and a test pins it), but the box is now one flat Surface: title in the Display
// role, sentence in Prose, capped at a readable measure.
func warehouseFeatureCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: atlasRegionClass()},
		html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Warehouse operations")),
		html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseTitle)),
		html.P(html.Props{Class: atlasProseClass()}, html.Text(parseCopy)),
	)
}

// warehouseStatCard is a single figure with a label, and it is deliberately no longer
// a card: the label is a Display micro line and the value is a mono Data figure, with
// nothing but space around them.
//
// Four of these in a row used to be four bordered boxes inside a bordered box inside a
// bordered section — three nesting levels to show four numbers. Now the caller puts
// them in one Cluster inside the region that already has a Surface.
func warehouseStatCard(parseLabel, parseValue string) ui.Node {
	return atlasFact(parseLabel, parseValue)
}

// warehouseListCard is a titled region holding a short list.
//
// One Surface, an Eyebrow that names it, and a Stack for the rows. The rows separate
// themselves with a hairline (see atlasRecordRowClass) rather than each becoming its
// own bordered box — a rule says "same thing, next entry", a box says "different
// thing", and inside one list the first statement is the true one.
func warehouseListCard(parseTitle string, parseChildren ...ui.Node) ui.Node {
	if len(parseChildren) == 0 {
		parseChildren = []ui.Node{html.P(html.Props{Class: atlasProseClass()}, html.Text("No records yet."))}
	}
	parseContent := []ui.Node{html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseTitle))}
	parseContent = append(parseContent, parseChildren...)
	return html.Div(html.Props{Class: atlasRegionClass()}, parseContent...)
}

// warehouseInfoRow is a label plus a machine fact — a hub code, a count, a service
// level. It is the most-used helper in this file (30-odd call sites), which is why it
// mattered most that it stopped being a box.
func warehouseInfoRow(parsePrimary string, parseSecondary string) ui.Node {
	return atlasFact(parsePrimary, parseSecondary)
}

// warehousePrimaryButtonClass is THE action for a view, and the design system means
// "one per view" literally (see design/controls.go).
//
// That is why almost nothing in this file uses it: a SKU route has one lane editor per
// warehouse lane, so "Save lane" is a repeated action and gets ButtonSecondary. The
// one honest primary in these surfaces is "Create order" inside the replenishment
// dialog — a dialog is its own view, with exactly one thing to do in it.
func warehousePrimaryButtonClass() string {
	return design.Class(design.ButtonPrimary())
}

// warehouseSecondaryButtonClass is every real action that is not THE action: save,
// apply, open, delete. There is no ButtonDanger in the design system, so a delete
// submit lands here too and the oxide appears on the consequence, not the trigger.
func warehouseSecondaryButtonClass() string {
	return design.Class(design.ButtonSecondary())
}

// warehouseQuietButtonClass is for dismissals — cancel, close, back. Text only until
// hover, so a per-row or per-dialog escape hatch costs no visual weight.
func warehouseQuietButtonClass() string {
	return design.Class(design.ButtonQuiet())
}

// warehouseInputClass is the control for words a person types: a search string, a
// vendor name, a note. Proportional.
func warehouseInputClass() string {
	return design.Class(design.Input())
}

// warehouseDataInputClass is the control for machine facts a person types: a quantity,
// a threshold, a SKU, a hub code, an ETA. Monospace and tabular, so a transposed digit
// is visible while it is still being typed and a column of quantities lines up.
func warehouseDataInputClass() string {
	return design.Class(design.InputData())
}

// inventoryStatusTone maps Atlas's lane and record vocabulary onto the design system's
// four status tones. It is the ONLY place in these surfaces that decides how loud a
// state is, which is the point: the design system takes a tone, never a color, so
// there is nowhere else to smuggle a hue in.
//
// The mapping is not a rename of the old palette, and two rows changed on purpose:
//
//   - low_stock is NEUTRAL, not amber. Being under the reorder point is what reorder
//     points are FOR: nothing failed, nobody is late, and the row's own quantity cell
//     already says how low it is. This follows the design system's own precedent that
//     "not stocked" is neutral (design/catalog.go) — a wall of warning color across a
//     40-row queue teaches operators to ignore warning color, and then the four rows
//     that really are broken read exactly like the thirty-six that are fine.
//   - promise_risk is PENDING, not the same tone as critical. Promise risk means a
//     commitment may slip and a human should know — that is precisely PostureHeld's
//     "not yet a failure". critical is the lane that has already failed, and it is the
//     only inventory state that earns the one FILLED tone.
//
// balanced/in_stock/healthy/available are VERIFIED because they are the outcome of a
// check that passed (on-hand against threshold). Verified is an outline, not a fill,
// so a queue of healthy rows stays quiet.
// inventoryStatusTone delegates to atlasStatusTone (page.go), which is the single
// status→tone mapping for the whole example.
//
// It used to be a second, independent switch, and for a while the package tuned
// the same status two different ways depending on which file happened to render
// it: page.go called low_stock an Exception, this file called it Neutral. Both
// were defensible in isolation and the pair was indefensible — an operator
// scanning two screens would see the same SKU flagged on one and quiet on the
// other, and would rightly stop trusting the colour.
//
// The merged mapping kept THIS file's judgement, because it is the one that
// preserves what the filled tone is for: low_stock → Neutral (a reorder point
// doing its job is not a failure) and promise_risk → Pending (a commitment that
// may slip has not slipped). See the reasoning at atlasStatusTone.
//
// This wrapper stays rather than being inlined at ~40 call sites: the name says
// "inventory's answer to this question", and if inventory ever genuinely needs to
// diverge, this is the one place to do it — deliberately, with a comment, instead
// of by a second switch drifting into existence again.
func inventoryStatusTone(parseStatus string) design.StatusTone {
	return atlasStatusTone(parseStatus)
}

// warehouseStatusClass folds the chip bundle for a status string. Kept as a class
// helper because page.go still calls it; new code in these surfaces should reach for
// atlasStatusChip, which also formats the label.
func warehouseStatusClass(parseStatus string) string {
	return design.Class(design.StatusChip(inventoryStatusTone(parseStatus)))
}

// atlasStatusChip renders the whole chip: tone from the status, label from Atlas's own
// vocabulary with the underscore removed. The chip uppercases in CSS, so the label is
// passed through as data rather than pre-shouted.
func atlasStatusChip(parseStatus string) ui.Node {
	return html.Span(
		html.Props{Class: warehouseStatusClass(parseStatus)},
		html.Text(strings.ReplaceAll(strings.TrimSpace(parseStatus), "_", " ")),
	)
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
