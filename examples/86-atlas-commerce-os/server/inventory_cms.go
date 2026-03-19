package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

type inventoryPageData struct {
	Summary routeSummaryData          `json:"summary"`
	Items   []repository.InventoryRow `json:"items"`
	Filters map[string]string         `json:"filters"`
}

type inventoryDetailPageData struct {
	SKU   string                    `json:"sku"`
	Title string                    `json:"title"`
	Rows  []repository.InventoryRow `json:"rows"`
}

type inventoryThresholdPanelPageData struct {
	SKU             string                                  `json:"sku"`
	Items           []serverdb.ThresholdHistoryRecord       `json:"items"`
	Recommendations []serverdb.TransferRecommendationRecord `json:"recommendations"`
}

type warehouseDetailPageData struct {
	Summary   routeSummaryData                 `json:"summary"`
	Warehouse serverdb.WarehousePressureRecord `json:"warehouse"`
	Inventory []repository.InventoryRow        `json:"inventory"`
	Orders    []serverdb.PurchaseOrderRecord   `json:"orders"`
	Filters   map[string]string                `json:"filters"`
}

type warehouseItemDetailPageData struct {
	Warehouse serverdb.WarehousePressureRecord `json:"warehouse"`
	Item      repository.InventoryRow          `json:"item"`
	Product   serverdb.ProductAdminRecord      `json:"product"`
	Network   []repository.InventoryRow        `json:"network"`
	Orders    []serverdb.PurchaseOrderRecord   `json:"orders"`
	Filters   map[string]string                `json:"filters"`
}

func (s *atlasServer) internalInventoryPageData(ctx context.Context, values url.Values) (inventoryPageData, error) {
	filters := map[string]string{
		"q":         strings.TrimSpace(values.Get("q")),
		"warehouse": strings.TrimSpace(values.Get("warehouse")),
		"status":    strings.TrimSpace(values.Get("status")),
		"sort":      strings.TrimSpace(values.Get("sort")),
	}
	query := repository.InventoryQuery{
		Warehouse:   filters["warehouse"],
		StockHealth: filters["status"],
		Search:      filters["q"],
	}
	switch filters["sort"] {
	case "inbound":
		query.SortKey = "inbound"
	case "updated":
		query.SortKey = "updated"
	}
	items, err := s.store.InventoryList(ctx, query)
	if err != nil {
		return inventoryPageData{}, err
	}
	return inventoryPageData{Summary: buildInventorySummary(items), Items: items, Filters: filters}, nil
}

func (s *atlasServer) internalInventoryDetailPageData(ctx context.Context, sku string) (inventoryDetailPageData, error) {
	rows, err := s.store.InventoryRowsBySKU(ctx, sku)
	if err != nil {
		return inventoryDetailPageData{}, err
	}
	return inventoryDetailPageData{SKU: rows[0].SKU, Title: rows[0].Title, Rows: rows}, nil
}

func (s *atlasServer) internalInventoryThresholdPanelPageData(ctx context.Context, sku string) (inventoryThresholdPanelPageData, error) {
	items, err := s.store.ThresholdHistory(ctx, sku)
	if err != nil {
		return inventoryThresholdPanelPageData{}, fmt.Errorf("load threshold history: %w", err)
	}
	recommendations, err := s.store.TransferRecommendations(ctx, sku)
	if err != nil {
		return inventoryThresholdPanelPageData{}, fmt.Errorf("load transfer recommendations: %w", err)
	}
	return inventoryThresholdPanelPageData{
		SKU:             sku,
		Items:           items,
		Recommendations: recommendations,
	}, nil
}

func (s *atlasServer) internalWarehouseDetailPageDataWithFilters(ctx context.Context, warehouseID string, values url.Values) (warehouseDetailPageData, error) {
	warehouse, err := s.store.WarehousePressureByID(ctx, warehouseID)
	if err != nil {
		return warehouseDetailPageData{}, err
	}
	filters := map[string]string{
		"q":      strings.TrimSpace(values.Get("q")),
		"status": strings.TrimSpace(values.Get("status")),
		"sort":   strings.TrimSpace(values.Get("sort")),
	}
	inventory, err := s.store.InventoryList(ctx, repository.InventoryQuery{Warehouse: warehouseID})
	if err != nil {
		return warehouseDetailPageData{}, fmt.Errorf("load warehouse inventory: %w", err)
	}
	inventory = filterWarehouseInventoryRows(inventory, filters)
	sortWarehouseInventoryRows(inventory, filters["sort"])
	orders, err := s.store.PurchaseOrdersByWarehouse(ctx, warehouseID)
	if err != nil {
		return warehouseDetailPageData{}, fmt.Errorf("load warehouse purchase orders: %w", err)
	}
	return warehouseDetailPageData{Summary: buildWarehouseDetailSummary(inventory, orders, warehouseID), Warehouse: warehouse, Inventory: inventory, Orders: orders, Filters: filters}, nil
}

func (s *atlasServer) internalWarehouseItemPageData(ctx context.Context, warehouseID string, sku string, values url.Values) (warehouseItemDetailPageData, error) {
	warehouse, err := s.store.WarehousePressureByID(ctx, warehouseID)
	if err != nil {
		return warehouseItemDetailPageData{}, err
	}
	filters := map[string]string{
		"sort": normalizeWarehouseItemSort(strings.TrimSpace(values.Get("sort"))),
		"dir":  normalizeWarehouseItemDirection(strings.TrimSpace(values.Get("sort")), strings.TrimSpace(values.Get("dir"))),
	}
	network, err := s.store.InventoryRowsBySKU(ctx, sku)
	if err != nil {
		return warehouseItemDetailPageData{}, err
	}
	item, ok := findWarehouseInventoryRow(network, warehouseID)
	if !ok {
		return warehouseItemDetailPageData{}, sql.ErrNoRows
	}
	sortWarehouseItemRows(network, filters["sort"], filters["dir"])
	product, err := s.store.ProductAdminBySlug(ctx, item.Slug)
	if err != nil {
		return warehouseItemDetailPageData{}, fmt.Errorf("load warehouse item product: %w", err)
	}
	orders, err := s.warehouseOrdersForSKU(ctx, warehouseID, sku)
	if err != nil {
		return warehouseItemDetailPageData{}, err
	}
	return warehouseItemDetailPageData{Warehouse: warehouse, Item: item, Product: product, Network: network, Orders: orders, Filters: filters}, nil
}

func normalizeWarehouseItemSort(sortKey string) string {
	switch strings.TrimSpace(strings.ToLower(sortKey)) {
	case "warehouse", "available", "cover", "inbound", "demand", "share", "updated":
		return strings.TrimSpace(strings.ToLower(sortKey))
	default:
		return "updated"
	}
}

func normalizeWarehouseItemDirection(sortKey string, direction string) string {
	switch strings.TrimSpace(strings.ToLower(direction)) {
	case "asc", "desc":
		return strings.TrimSpace(strings.ToLower(direction))
	}
	switch normalizeWarehouseItemSort(sortKey) {
	case "warehouse":
		return "asc"
	case "cover":
		return "asc"
	default:
		return "desc"
	}
}

func sortWarehouseItemRows(items []repository.InventoryRow, sortKey string, direction string) {
	key := normalizeWarehouseItemSort(sortKey)
	dir := normalizeWarehouseItemDirection(key, direction)
	compareText := func(left string, right string) int {
		leftValue := strings.ToLower(strings.TrimSpace(left))
		rightValue := strings.ToLower(strings.TrimSpace(right))
		switch {
		case leftValue < rightValue:
			return -1
		case leftValue > rightValue:
			return 1
		default:
			return 0
		}
	}
	compareInt := func(left int, right int) int {
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	}
	compareUpdated := func(left string, right string) int {
		return compareText(left, right)
	}
	sort.Slice(items, func(left, right int) bool {
		leftItem := items[left]
		rightItem := items[right]
		result := 0
		switch key {
		case "warehouse":
			result = compareText(fallbackWarehouseLabel(leftItem), fallbackWarehouseLabel(rightItem))
		case "available":
			result = compareInt(leftItem.Available, rightItem.Available)
		case "cover":
			result = compareInt(leftItem.CoverDays, rightItem.CoverDays)
		case "inbound":
			result = compareInt(leftItem.Inbound, rightItem.Inbound)
		case "demand":
			result = compareInt(leftItem.WeeklyUnits, rightItem.WeeklyUnits)
		case "share":
			result = compareInt(leftItem.RegionalShare, rightItem.RegionalShare)
		default:
			result = compareUpdated(leftItem.UpdatedAt, rightItem.UpdatedAt)
		}
		if result == 0 {
			result = compareText(leftItem.Title, rightItem.Title)
		}
		if result == 0 {
			result = compareText(leftItem.WarehouseID, rightItem.WarehouseID)
		}
		if dir == "asc" {
			return result < 0
		}
		return result > 0
	})
}

func fallbackWarehouseLabel(item repository.InventoryRow) string {
	if strings.TrimSpace(item.WarehouseName) != "" {
		return item.WarehouseName
	}
	return item.WarehouseID
}

func filterWarehouseInventoryRows(items []repository.InventoryRow, filters map[string]string) []repository.InventoryRow {
	if len(items) == 0 {
		return items
	}
	query := strings.TrimSpace(strings.ToLower(filters["q"]))
	status := strings.TrimSpace(strings.ToLower(filters["status"]))
	filtered := make([]repository.InventoryRow, 0, len(items))
	for _, item := range items {
		if status != "" && status != "all" && strings.TrimSpace(strings.ToLower(item.Status)) != status {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(strings.Join([]string{item.SKU, item.Title, item.Category, item.MarketPressure}, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func sortWarehouseInventoryRows(items []repository.InventoryRow, sortKey string) {
	switch strings.TrimSpace(strings.ToLower(sortKey)) {
	case "available":
		sort.Slice(items, func(left, right int) bool {
			if items[left].Available == items[right].Available {
				return items[left].Title < items[right].Title
			}
			return items[left].Available < items[right].Available
		})
	case "demand":
		sort.Slice(items, func(left, right int) bool {
			if items[left].DemandScore == items[right].DemandScore {
				return items[left].Title < items[right].Title
			}
			return items[left].DemandScore > items[right].DemandScore
		})
	case "revenue":
		sort.Slice(items, func(left, right int) bool {
			if items[left].WeeklyRevenue == items[right].WeeklyRevenue {
				return items[left].Title < items[right].Title
			}
			return items[left].WeeklyRevenue > items[right].WeeklyRevenue
		})
	default:
		sort.Slice(items, func(left, right int) bool {
			if items[left].UpdatedAt == items[right].UpdatedAt {
				return items[left].Title < items[right].Title
			}
			return items[left].UpdatedAt > items[right].UpdatedAt
		})
	}
}

func findWarehouseInventoryRow(items []repository.InventoryRow, warehouseID string) (repository.InventoryRow, bool) {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.WarehouseID), strings.TrimSpace(warehouseID)) {
			return item, true
		}
	}
	return repository.InventoryRow{}, false
}

func (s *atlasServer) warehouseOrdersForSKU(ctx context.Context, warehouseID string, sku string) ([]serverdb.PurchaseOrderRecord, error) {
	orders, err := s.store.PurchaseOrdersByWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("load warehouse sku orders: %w", err)
	}
	filtered := make([]serverdb.PurchaseOrderRecord, 0, len(orders))
	for _, order := range orders {
		detail, err := s.store.PurchaseOrderDetail(ctx, order.ID)
		if err != nil {
			return nil, fmt.Errorf("load purchase order detail: %w", err)
		}
		for _, line := range detail.Lines {
			if strings.EqualFold(strings.TrimSpace(line.ProductSKU), strings.TrimSpace(sku)) {
				filtered = append(filtered, order)
				break
			}
		}
	}
	return filtered, nil
}
