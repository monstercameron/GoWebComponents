package main

import (
	"context"
	"fmt"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
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

func (s *atlasServer) internalDashboardPageData(ctx context.Context) (dashboardPageData, error) {
	comments, err := s.store.Comments(ctx, "")
	if err != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard comments: %w", err)
	}
	transfers, err := s.store.Transfers(ctx)
	if err != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard transfers: %w", err)
	}
	receiving, err := s.store.ReceivingSessions(ctx)
	if err != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard receiving: %w", err)
	}
	orders, err := s.store.PurchaseOrders(ctx)
	if err != nil {
		return dashboardPageData{}, fmt.Errorf("load dashboard purchase orders: %w", err)
	}
	return dashboardPageData{
		Summary:   buildDashboardSummary(comments, transfers, receiving, orders),
		Alerts:    countCommentsByStatus(comments, "pending") + countCommentsByStatus(comments, "flagged") + countOpenReceivingSessions(receiving) + countPurchaseOrdersByStatus(orders, "submitted") + countPurchaseOrdersByStatus(orders, "on_hold"),
		Transfers: transfers,
		Receiving: receiving,
		Comments:  comments,
		Orders:    orders,
	}, nil
}

func (s *atlasServer) internalWarehouseOpsPageData(ctx context.Context) (warehouseOpsPageData, error) {
	items, err := s.store.WarehousePressure(ctx)
	if err != nil {
		return warehouseOpsPageData{}, fmt.Errorf("load warehouse ops: %w", err)
	}
	return warehouseOpsPageData{Summary: buildWarehouseOpsSummary(items), Items: items}, nil
}

func (s *atlasServer) internalPurchaseOrdersPageData(ctx context.Context) (purchaseOrdersPageData, error) {
	items, err := s.store.PurchaseOrders(ctx)
	if err != nil {
		return purchaseOrdersPageData{}, fmt.Errorf("load purchase orders: %w", err)
	}
	return purchaseOrdersPageData{Summary: buildPurchaseOrderSummary(items), Items: items}, nil
}

func (s *atlasServer) internalCommentsPageData(ctx context.Context, status string) (commentsPageData, error) {
	items, err := s.store.Comments(ctx, status)
	if err != nil {
		return commentsPageData{}, fmt.Errorf("load comments: %w", err)
	}
	return commentsPageData{Summary: buildCommentSummary(items), Items: items}, nil
}

func (s *atlasServer) internalSettingsPageData(ctx context.Context, ownerID string) (settingsPageData, error) {
	preferences, err := s.store.PreferencesByOwner(ctx, ownerID)
	if err != nil {
		return settingsPageData{}, fmt.Errorf("load preferences: %w", err)
	}
	savedViews, err := s.store.SavedViewsByOwner(ctx, ownerID)
	if err != nil {
		return settingsPageData{}, fmt.Errorf("load saved views: %w", err)
	}
	return settingsPageData{Summary: buildSettingsSummary(preferences, savedViews)}, nil
}

func buildDashboardSummary(comments []serverdb.CommentRecord, transfers []serverdb.TransferRecord, receiving []serverdb.ReceivingSessionRecord, orders []serverdb.PurchaseOrderRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Demand and operations overview",
		Items: []routeSummaryItemData{
			{Label: "Buyer inbox", Value: fmt.Sprintf("%d queued", len(comments)), Detail: fmt.Sprintf("%d pending and %d flagged items are shaping the next operator handoff.", countCommentsByStatus(comments, "pending"), countCommentsByStatus(comments, "flagged"))},
			{Label: "Transfers", Value: fmt.Sprintf("%d active", len(transfers)), Detail: fmt.Sprintf("%d submitted and %d approved moves are still influencing supply balance.", countTransfersByStatus(transfers, "submitted"), countTransfersByStatus(transfers, "approved"))},
			{Label: "Receiving", Value: fmt.Sprintf("%d sessions", len(receiving)), Detail: fmt.Sprintf("%d sessions still need closeout or discrepancy review.", countOpenReceivingSessions(receiving))},
			{Label: "Purchase orders", Value: fmt.Sprintf("%d open", len(orders)), Detail: fmt.Sprintf("%d submitted and %d approved orders are feeding inbound planning.", countPurchaseOrdersByStatus(orders, "submitted"), countPurchaseOrdersByStatus(orders, "approved"))},
		},
	}
}

func buildInventorySummary(rows []repository.InventoryRow) routeSummaryData {
	rollup := atlas.InventoryRollupFromRepositoryRows(rows)
	return routeSummaryData{
		Headline: "Inventory pressure baseline",
		Items: []routeSummaryItemData{
			{Label: "Visible lanes", Value: fmt.Sprintf("%d", rollup.VisibleLanes), Detail: "The current filter set drives both the queue and the operator action rail."},
			{Label: "Promise risk", Value: fmt.Sprintf("%d", rollup.PromiseRiskLanes), Detail: "These lanes are closest to slipping customer-facing availability promises."},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", rollup.InboundUnits), Detail: "Use inbound to separate lanes that need patience from lanes that need new action."},
			{Label: "Reorder units", Value: fmt.Sprintf("%d", rollup.ReorderUnits), Detail: "This total is already pointing at vendor replenishment instead of local balancing."},
		},
	}
}

func buildWarehouseOpsSummary(items []serverdb.WarehousePressureRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Warehouse operations baseline",
		Items: []routeSummaryItemData{
			{Label: "Facilities", Value: fmt.Sprintf("%d", len(items)), Detail: "Each warehouse keeps service posture, backlog, and risk lanes visible in one roster."},
			{Label: "Risk lanes", Value: fmt.Sprintf("%d", countWarehouseRiskLanes(items)), Detail: "Use the facility view first when the problem is local pressure rather than a cross-network SKU issue."},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", countWarehouseInbound(items)), Detail: "Inbound posture shows which facilities are already recovering without new vendor work."},
			{Label: "Priority lane", Value: fallbackWarehousePressure(items), Detail: "The roster keeps the hottest warehouse visible without drilling into every local item list."},
		},
	}
}

func buildWarehouseDetailSummary(items []repository.InventoryRow, orders []serverdb.PurchaseOrderRecord, warehouseID string) routeSummaryData {
	return routeSummaryData{
		Headline: "Warehouse route baseline",
		Items: []routeSummaryItemData{
			{Label: "Visible items", Value: fmt.Sprintf("%d", len(items)), Detail: "These item rows already reflect the warehouse-local filters applied to the route."},
			{Label: "Weekly demand", Value: fmt.Sprintf("%d units", countWarehouseDemand(items)), Detail: "Demand pressure is derived once on the server and reused by the route shell."},
			{Label: "Open POs", Value: fmt.Sprintf("%d", len(orders)), Detail: fmt.Sprintf("The purchase-order rail stays scoped to %s instead of recomputing a cross-network list.", warehouseID)},
			{Label: "Urgent items", Value: fmt.Sprintf("%d", countWarehouseUrgentItems(items)), Detail: "Urgency combines reorder pressure and hot-market demand cues."},
		},
	}
}

func buildPurchaseOrderSummary(items []serverdb.PurchaseOrderRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Vendor replenishment baseline",
		Items: []routeSummaryItemData{
			{Label: "Open orders", Value: fmt.Sprintf("%d", len(items)), Detail: "The list keeps vendor approvals and inbound coordination in one operator queue."},
			{Label: "Submitted", Value: fmt.Sprintf("%d", countPurchaseOrdersByStatus(items, "submitted")), Detail: "These orders still need approval or coordination before receiving can take over."},
			{Label: "Approved", Value: fmt.Sprintf("%d", countPurchaseOrdersByStatus(items, "approved")), Detail: "Approved orders should route directly into inbound monitoring and receiving readiness."},
			{Label: "Primary warehouse", Value: fallbackPurchaseOrderWarehouse(items), Detail: "This cue keeps the highest-volume inbound lane obvious at list level."},
		},
	}
}

func buildCommentSummary(items []serverdb.CommentRecord) routeSummaryData {
	return routeSummaryData{
		Headline: "Buyer inbox baseline",
		Items: []routeSummaryItemData{
			{Label: "Visible items", Value: fmt.Sprintf("%d", len(items)), Detail: "Use the visible queue for both one-off review and bulk moderation passes."},
			{Label: "Pending", Value: fmt.Sprintf("%d", countCommentsByStatus(items, "pending")), Detail: "Pending buyer questions are the best candidates for a first-pass review sweep."},
			{Label: "Flagged", Value: fmt.Sprintf("%d", countCommentsByStatus(items, "flagged")), Detail: "Flagged records need extra context before they can safely return to the public thread."},
			{Label: "Approved", Value: fmt.Sprintf("%d", countCommentsByStatus(items, "approved")), Detail: "Approved items confirm that queue work is actually clearing back into the storefront."},
		},
	}
}

func buildSettingsSummary(preferences serverdb.PreferencesRecord, views []repository.SavedView) routeSummaryData {
	return routeSummaryData{
		Headline: "Workspace preferences and transfer tools",
		Items: []routeSummaryItemData{
			{Label: "Theme", Value: nonEmpty(preferences.Theme, "dark"), Detail: "Theme stays in bootstrap so SSR and hydration read the same shell styling."},
			{Label: "Locale", Value: nonEmpty(preferences.Locale, "en"), Detail: "Locale drives document direction and route copy from the first server render."},
			{Label: "Saved views", Value: fmt.Sprintf("%d", len(views)), Detail: "Export and import let Atlas carry operator view presets between runs without manual recreation."},
			{Label: "Default warehouse", Value: nonEmpty(preferences.DefaultWarehouseID, "new-jersey-hub"), Detail: "The shell keeps one facility ready as the fallback for product and inventory workflows."},
		},
	}
}

func countCommentsByStatus(items []serverdb.CommentRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), strings.TrimSpace(status)) {
			count++
		}
	}
	return count
}

func countTransfersByStatus(items []serverdb.TransferRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), strings.TrimSpace(status)) {
			count++
		}
	}
	return count
}

func countOpenReceivingSessions(items []serverdb.ReceivingSessionRecord) int {
	count := 0
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.Status), "closed") {
			count++
		}
	}
	return count
}

func countWarehouseRiskLanes(items []serverdb.WarehousePressureRecord) int {
	total := 0
	for _, item := range items {
		total += item.RiskCount
	}
	return total
}

func countWarehouseInbound(items []serverdb.WarehousePressureRecord) int {
	total := 0
	for _, item := range items {
		total += item.Inbound
	}
	return total
}

func fallbackWarehousePressure(items []serverdb.WarehousePressureRecord) string {
	if len(items) == 0 {
		return "No active lane"
	}
	primary := items[0]
	for _, item := range items[1:] {
		if item.RiskCount > primary.RiskCount {
			primary = item
		}
	}
	if strings.TrimSpace(primary.Name) == "" {
		return primary.ID
	}
	return primary.Name
}

func countWarehouseDemand(items []repository.InventoryRow) int {
	total := 0
	for _, item := range items {
		total += item.WeeklyUnits
	}
	return total
}

func countWarehouseUrgentItems(items []repository.InventoryRow) int {
	count := 0
	for _, item := range items {
		if item.ReorderUnits > 0 || strings.EqualFold(strings.TrimSpace(item.MarketPressure), "hot market") {
			count++
		}
	}
	return count
}

func countPurchaseOrdersByStatus(items []serverdb.PurchaseOrderRecord, status string) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), strings.TrimSpace(status)) {
			count++
		}
	}
	return count
}

func fallbackPurchaseOrderWarehouse(items []serverdb.PurchaseOrderRecord) string {
	if len(items) == 0 {
		return "No active inbound lane"
	}
	counts := map[string]int{}
	labels := map[string]string{}
	for _, item := range items {
		key := strings.TrimSpace(item.WarehouseID)
		counts[key]++
		labels[key] = nonEmpty(item.WarehouseName, item.WarehouseID)
	}
	bestKey := ""
	bestCount := -1
	for key, count := range counts {
		if count > bestCount {
			bestKey = key
			bestCount = count
		}
	}
	return nonEmpty(labels[bestKey], bestKey)
}
