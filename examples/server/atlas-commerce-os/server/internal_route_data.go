package main

import (
	"context"
	"fmt"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/shared/repository"
)

type routeSummaryData struct {
	Headline string                 `json:"headline"`
	Items    []routeSummaryItemData `json:"items"`
}

type routeSummaryItemData struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

type dashboardPageData struct {
	Summary   routeSummaryData                  `json:"summary"`
	Alerts    int                               `json:"alerts"`
	Transfers []serverdb.TransferRecord         `json:"transfers"`
	Receiving []serverdb.ReceivingSessionRecord `json:"receiving"`
	Comments  []serverdb.CommentRecord          `json:"comments"`
	Orders    []serverdb.PurchaseOrderRecord    `json:"orders"`
}

type warehouseOpsPageData struct {
	Summary routeSummaryData                   `json:"summary"`
	Items   []serverdb.WarehousePressureRecord `json:"items"`
}

type purchaseOrdersPageData struct {
	Summary routeSummaryData               `json:"summary"`
	Items   []serverdb.PurchaseOrderRecord `json:"items"`
}

type commentsPageData struct {
	Summary routeSummaryData         `json:"summary"`
	Items   []serverdb.CommentRecord `json:"items"`
}

type settingsPageData struct {
	Summary routeSummaryData `json:"summary"`
}

func (parseS *atlasServer) internalDashboardPageData(parseCtx context.Context) (dashboardPageData, error) {
	parseComments, parseErr := parseS.store.Comments(parseCtx, "")
	if parseErr != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard comments: %w", parseErr)
	}
	parseTransfers, parseErr := parseS.store.Transfers(parseCtx)
	if parseErr != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard transfers: %w", parseErr)
	}
	parseReceiving, parseErr := parseS.store.ReceivingSessions(parseCtx)
	if parseErr != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard receiving: %w", parseErr)
	}
	parseOrders, parseErr := parseS.store.PurchaseOrders(parseCtx)
	if parseErr != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard purchase orders: %w", parseErr)
	}
	return dashboardPageData{
		Summary:   buildDashboardSummary(parseComments, parseTransfers, parseReceiving, parseOrders),
		Alerts:    countCommentsByStatus(parseComments, "pending") + countCommentsByStatus(parseComments, "flagged") + countOpenReceivingSessions(parseReceiving) + countPurchaseOrdersByStatus(parseOrders, "submitted") + countPurchaseOrdersByStatus(parseOrders, "on_hold"),
		Transfers: parseTransfers,
		Receiving: parseReceiving,
		Comments:  parseComments,
		Orders:    parseOrders,
	}, nil
}

func (parseS *atlasServer) internalWarehouseOpsPageData(parseCtx context.Context) (warehouseOpsPageData, error) {
	parseItems, parseErr := parseS.store.WarehousePressure(parseCtx)
	if parseErr != nil {
		return warehouseOpsPageData{}, fmt.Errorf("load warehouse ops: %w", parseErr)
	}
	return warehouseOpsPageData{Summary: buildWarehouseOpsSummary(parseItems), Items: parseItems}, nil
}

func (parseS *atlasServer) internalPurchaseOrdersPageData(parseCtx context.Context) (purchaseOrdersPageData, error) {
	parseItems, parseErr := parseS.store.PurchaseOrders(parseCtx)
	if parseErr != nil {
		return purchaseOrdersPageData{}, fmt.Errorf("load purchase orders: %w", parseErr)
	}
	return purchaseOrdersPageData{Summary: buildPurchaseOrderSummary(parseItems), Items: parseItems}, nil
}

func (parseS *atlasServer) internalCommentsPageData(parseCtx context.Context, parseStatus string) (commentsPageData, error) {
	parseItems, parseErr := parseS.store.Comments(parseCtx, parseStatus)
	if parseErr != nil {
		return commentsPageData{}, fmt.Errorf("load comments: %w", parseErr)
	}
	return commentsPageData{Summary: buildCommentSummary(parseItems), Items: parseItems}, nil
}

func (parseS *atlasServer) internalSettingsPageData(parseCtx context.Context, parseOwnerID string) (settingsPageData, error) {
	parsePreferences, parseErr := parseS.store.PreferencesByOwner(parseCtx, parseOwnerID)
	if parseErr != nil {
		return settingsPageData{}, fmt.Errorf("load preferences: %w", parseErr)
	}
	parseSavedViews, parseErr := parseS.store.SavedViewsByOwner(parseCtx, parseOwnerID)
	if parseErr != nil {
		return settingsPageData{}, fmt.Errorf("load saved views: %w", parseErr)
	}
	return settingsPageData{Summary: buildSettingsSummary(parsePreferences, parseSavedViews)}, nil
}

func buildDashboardSummary(parseComments []serverdb.CommentRecord, parseTransfers []serverdb.TransferRecord, parseReceiving []serverdb.ReceivingSessionRecord, parseOrders []serverdb.PurchaseOrderRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Demand and operations overview",
		Items: []routeSummaryItemData{
			{Label: "Buyer inbox", Value: fmt.Sprintf("%d queued", len(parseComments)), Detail: fmt.Sprintf("%d pending and %d flagged items are shaping the next operator handoff.", countCommentsByStatus(parseComments, "pending"), countCommentsByStatus(parseComments, "flagged"))},
			{Label: "Transfers", Value: fmt.Sprintf("%d active", len(parseTransfers)), Detail: fmt.Sprintf("%d submitted and %d approved moves are still influencing supply balance.", countTransfersByStatus(parseTransfers, "submitted"), countTransfersByStatus(parseTransfers, "approved"))},
			{Label: "Receiving", Value: fmt.Sprintf("%d sessions", len(parseReceiving)), Detail: fmt.Sprintf("%d sessions still need closeout or discrepancy review.", countOpenReceivingSessions(parseReceiving))},
			{Label: "Purchase orders", Value: fmt.Sprintf("%d open", len(parseOrders)), Detail: fmt.Sprintf("%d submitted and %d approved orders are feeding inbound planning.", countPurchaseOrdersByStatus(parseOrders, "submitted"), countPurchaseOrdersByStatus(parseOrders, "approved"))},
		},
	}
}

func buildInventorySummary(parseRows []repository.InventoryRow) routeSummaryData {
	parseRollup := atlas.InventoryRollupFromRepositoryRows(parseRows)
	return routeSummaryData{
		Headline: "Inventory pressure baseline",
		Items: []routeSummaryItemData{
			{Label: "Visible lanes", Value: fmt.Sprintf("%d", parseRollup.VisibleLanes), Detail: "The current filter set drives both the queue and the operator action rail."},
			{Label: "Promise risk", Value: fmt.Sprintf("%d", parseRollup.PromiseRiskLanes), Detail: "These lanes are closest to slipping customer-facing availability promises."},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", parseRollup.InboundUnits), Detail: "Use inbound to separate lanes that need patience from lanes that need new action."},
			{Label: "Reorder units", Value: fmt.Sprintf("%d", parseRollup.ReorderUnits), Detail: "This total is already pointing at vendor replenishment instead of local balancing."},
		},
	}
}

func buildWarehouseOpsSummary(parseItems []serverdb.WarehousePressureRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Warehouse operations baseline",
		Items: []routeSummaryItemData{
			{Label: "Facilities", Value: fmt.Sprintf("%d", len(parseItems)), Detail: "Each warehouse keeps service posture, backlog, and risk lanes visible in one roster."},
			{Label: "Risk lanes", Value: fmt.Sprintf("%d", countWarehouseRiskLanes(parseItems)), Detail: "Use the facility view first when the problem is local pressure rather than a cross-network SKU issue."},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", countWarehouseInbound(parseItems)), Detail: "Inbound posture shows which facilities are already recovering without new vendor work."},
			{Label: "Priority lane", Value: fallbackWarehousePressure(parseItems), Detail: "The roster keeps the hottest warehouse visible without drilling into every local item list."},
		},
	}
}

func buildWarehouseDetailSummary(parseItems []repository.InventoryRow, parseOrders []serverdb.PurchaseOrderRecord, parseWarehouseID string) routeSummaryData {
	return routeSummaryData{
		Headline: "Warehouse route baseline",
		Items: []routeSummaryItemData{
			{Label: "Visible items", Value: fmt.Sprintf("%d", len(parseItems)), Detail: "These item rows already reflect the warehouse-local filters applied to the route."},
			{Label: "Weekly demand", Value: fmt.Sprintf("%d units", countWarehouseDemand(parseItems)), Detail: "Demand pressure is derived once on the server and reused by the route shell."},
			{Label: "Open POs", Value: fmt.Sprintf("%d", len(parseOrders)), Detail: fmt.Sprintf("The purchase-order rail stays scoped to %s instead of recomputing a cross-network list.", parseWarehouseID)},
			{Label: "Urgent items", Value: fmt.Sprintf("%d", countWarehouseUrgentItems(parseItems)), Detail: "Urgency combines reorder pressure and hot-market demand cues."},
		},
	}
}

func buildPurchaseOrderSummary(parseItems []serverdb.PurchaseOrderRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Vendor replenishment baseline",
		Items: []routeSummaryItemData{
			{Label: "Open orders", Value: fmt.Sprintf("%d", len(parseItems)), Detail: "The list keeps vendor approvals and inbound coordination in one operator queue."},
			{Label: "Submitted", Value: fmt.Sprintf("%d", countPurchaseOrdersByStatus(parseItems, "submitted")), Detail: "These orders still need approval or coordination before receiving can take over."},
			{Label: "Approved", Value: fmt.Sprintf("%d", countPurchaseOrdersByStatus(parseItems, "approved")), Detail: "Approved orders should route directly into inbound monitoring and receiving readiness."},
			{Label: "Primary warehouse", Value: fallbackPurchaseOrderWarehouse(parseItems), Detail: "This cue keeps the highest-volume inbound lane obvious at list level."},
		},
	}
}

func buildCommentSummary(parseItems []serverdb.CommentRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Buyer inbox baseline",
		Items: []routeSummaryItemData{
			{Label: "Visible items", Value: fmt.Sprintf("%d", len(parseItems)), Detail: "Use the visible queue for both one-off review and bulk moderation passes."},
			{Label: "Pending", Value: fmt.Sprintf("%d", countCommentsByStatus(parseItems, "pending")), Detail: "Pending buyer questions are the best candidates for a first-pass review sweep."},
			{Label: "Flagged", Value: fmt.Sprintf("%d", countCommentsByStatus(parseItems, "flagged")), Detail: "Flagged records need extra context before they can safely return to the public thread."},
			{Label: "Approved", Value: fmt.Sprintf("%d", countCommentsByStatus(parseItems, "approved")), Detail: "Approved items confirm that queue work is actually clearing back into the storefront."},
		},
	}
}

func buildSettingsSummary(parsePreferences serverdb.PreferencesRecord, parseViews []repository.SavedView) routeSummaryData {
	return routeSummaryData{
		Headline: "Workspace preferences and transfer tools",
		Items: []routeSummaryItemData{
			{Label: "Theme", Value: nonEmpty(parsePreferences.Theme, "dark"), Detail: "Theme stays in bootstrap so SSR and hydration read the same shell styling."},
			{Label: "Locale", Value: nonEmpty(parsePreferences.Locale, "en"), Detail: "Locale drives document direction and route copy from the first server render."},
			{Label: "Saved views", Value: fmt.Sprintf("%d", len(parseViews)), Detail: "Export and import let Atlas carry operator view presets between runs without manual recreation."},
			{Label: "Default warehouse", Value: nonEmpty(parsePreferences.DefaultWarehouseID, "new-jersey-hub"), Detail: "The shell keeps one facility ready as the fallback for product and inventory workflows."},
		},
	}
}

func countCommentsByStatus(parseItems []serverdb.CommentRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), strings.TrimSpace(parseStatus)) {
			parseCount++
		}
	}
	return parseCount
}

func countTransfersByStatus(parseItems []serverdb.TransferRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), strings.TrimSpace(parseStatus)) {
			parseCount++
		}
	}
	return parseCount
}

func countOpenReceivingSessions(parseItems []serverdb.ReceivingSessionRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if !strings.EqualFold(strings.TrimSpace(parseItem.Status), "closed") {
			parseCount++
		}
	}
	return parseCount
}

func countWarehouseRiskLanes(parseItems []serverdb.WarehousePressureRecord) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.RiskCount
	}
	return parseTotal
}

func countWarehouseInbound(parseItems []serverdb.WarehousePressureRecord) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Inbound
	}
	return parseTotal
}

func fallbackWarehousePressure(parseItems []serverdb.WarehousePressureRecord) string {
	if len(parseItems) == 0 {
		return "No active lane"
	}
	parsePrimary := parseItems[0]
	for _, parseItem := range parseItems[1:] {
		if parseItem.RiskCount > parsePrimary.RiskCount {
			parsePrimary = parseItem
		}
	}
	if strings.TrimSpace(parsePrimary.Name) == "" {
		return parsePrimary.ID
	}
	return parsePrimary.Name
}

func countWarehouseDemand(parseItems []repository.InventoryRow) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.WeeklyUnits
	}
	return parseTotal
}

func countWarehouseUrgentItems(parseItems []repository.InventoryRow) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if parseItem.ReorderUnits > 0 || strings.EqualFold(strings.TrimSpace(parseItem.MarketPressure), "hot market") {
			parseCount++
		}
	}
	return parseCount
}

func countPurchaseOrdersByStatus(parseItems []serverdb.PurchaseOrderRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), strings.TrimSpace(parseStatus)) {
			parseCount++
		}
	}
	return parseCount
}

func fallbackPurchaseOrderWarehouse(parseItems []serverdb.PurchaseOrderRecord) string {
	if len(parseItems) == 0 {
		return "No active inbound lane"
	}
	parseCounts := map[string]int{}
	parseLabels := map[string]string{}
	for _, parseItem := range parseItems {
		parseKey := strings.TrimSpace(parseItem.WarehouseID)
		parseCounts[parseKey]++
		parseLabels[parseKey] = nonEmpty(parseItem.WarehouseName, parseItem.WarehouseID)
	}
	parseBestKey := ""
	parseBestCount := -1
	for parseKey2, parseCount := range parseCounts {
		if parseCount > parseBestCount {
			parseBestKey = parseKey2
			parseBestCount = parseCount
		}
	}
	return nonEmpty(parseLabels[parseBestKey], parseBestKey)
}
