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

func (parseS *atlasServer) internalInventoryPageData(parseCtx context.Context, parseValues url.Values) (inventoryPageData, error) {
	// Keep raw query echoes for UI resume while separately mapping only supported sort keys into repository queries.
	parseFilters := map[string]string{
		"q":         strings.TrimSpace(parseValues.Get("q")),
		"warehouse": strings.TrimSpace(parseValues.Get("warehouse")),
		"status":    strings.TrimSpace(parseValues.Get("status")),
		"sort":      strings.TrimSpace(parseValues.Get("sort")),
	}
	parseQuery := repository.InventoryQuery{
		Warehouse:   parseFilters["warehouse"],
		StockHealth: parseFilters["status"],
		Search:      parseFilters["q"],
	}
	switch parseFilters["sort"] {
	case "inbound":
		parseQuery.SortKey = "inbound"
	case "updated":
		parseQuery.SortKey = "updated"
	}
	parseItems, parseErr := parseS.store.InventoryList(parseCtx, parseQuery)
	if parseErr != nil {
		return inventoryPageData{}, parseErr
	}
	return inventoryPageData{Summary: buildInventorySummary(parseItems), Items: parseItems, Filters: parseFilters}, nil
}

func (parseS *atlasServer) internalInventoryDetailPageData(parseCtx context.Context, parseSku string) (inventoryDetailPageData, error) {
	parseRows, parseErr := parseS.store.InventoryRowsBySKU(parseCtx, parseSku)
	if parseErr != nil {
		return inventoryDetailPageData{}, parseErr
	}
	return inventoryDetailPageData{SKU: parseRows[0].SKU, Title: parseRows[0].Title, Rows: parseRows}, nil
}

func (parseS *atlasServer) internalInventoryThresholdPanelPageData(parseCtx context.Context, parseSku string) (inventoryThresholdPanelPageData, error) {
	parseItems, parseErr := parseS.store.ThresholdHistory(parseCtx, parseSku)
	if parseErr != nil {
		return inventoryThresholdPanelPageData{}, fmt.Errorf("load threshold history: %w", parseErr)
	}
	parseRecommendations, parseErr := parseS.store.TransferRecommendations(parseCtx, parseSku)
	if parseErr != nil {
		return inventoryThresholdPanelPageData{}, fmt.Errorf("load transfer recommendations: %w", parseErr)
	}
	return inventoryThresholdPanelPageData{
		SKU:             parseSku,
		Items:           parseItems,
		Recommendations: parseRecommendations,
	}, nil
}

func (parseS *atlasServer) internalWarehouseDetailPageDataWithFilters(parseCtx context.Context, parseWarehouseID string, parseValues url.Values) (warehouseDetailPageData, error) {
	parseWarehouse, parseErr := parseS.store.WarehousePressureByID(parseCtx, parseWarehouseID)
	if parseErr != nil {
		return warehouseDetailPageData{}, parseErr
	}
	parseFilters := map[string]string{
		"q":      strings.TrimSpace(parseValues.Get("q")),
		"status": strings.TrimSpace(parseValues.Get("status")),
		"sort":   strings.TrimSpace(parseValues.Get("sort")),
	}
	parseInventory, parseErr := parseS.store.InventoryList(parseCtx, repository.InventoryQuery{Warehouse: parseWarehouseID})
	if parseErr != nil {
		return warehouseDetailPageData{}, fmt.Errorf("load warehouse inventory: %w", parseErr)
	}
	parseInventory = filterWarehouseInventoryRows(parseInventory, parseFilters)
	sortWarehouseInventoryRows(parseInventory, parseFilters["sort"])
	parseOrders, parseErr := parseS.store.PurchaseOrdersByWarehouse(parseCtx, parseWarehouseID)
	if parseErr != nil {
		return warehouseDetailPageData{}, fmt.Errorf("load warehouse purchase orders: %w", parseErr)
	}
	return warehouseDetailPageData{Summary: buildWarehouseDetailSummary(parseInventory, parseOrders, parseWarehouseID), Warehouse: parseWarehouse, Inventory: parseInventory, Orders: parseOrders, Filters: parseFilters}, nil
}

func (parseS *atlasServer) internalWarehouseItemPageData(parseCtx context.Context, parseWarehouseID string, parseSku string, parseValues url.Values) (warehouseItemDetailPageData, error) {
	parseWarehouse, parseErr := parseS.store.WarehousePressureByID(parseCtx, parseWarehouseID)
	if parseErr != nil {
		return warehouseItemDetailPageData{}, parseErr
	}
	parseFilters := map[string]string{
		"sort": normalizeWarehouseItemSort(strings.TrimSpace(parseValues.Get("sort"))),
		"dir":  normalizeWarehouseItemDirection(strings.TrimSpace(parseValues.Get("sort")), strings.TrimSpace(parseValues.Get("dir"))),
	}
	parseNetwork, parseErr := parseS.store.InventoryRowsBySKU(parseCtx, parseSku)
	if parseErr != nil {
		return warehouseItemDetailPageData{}, parseErr
	}
	parseItem, parseOk := findWarehouseInventoryRow(parseNetwork, parseWarehouseID)
	if !parseOk {
		return warehouseItemDetailPageData{}, sql.ErrNoRows
	}
	sortWarehouseItemRows(parseNetwork, parseFilters["sort"], parseFilters["dir"])
	parseProduct, parseErr := parseS.store.ProductAdminBySlug(parseCtx, parseItem.Slug)
	if parseErr != nil {
		return warehouseItemDetailPageData{}, fmt.Errorf("load warehouse item product: %w", parseErr)
	}
	parseOrders, parseErr := parseS.warehouseOrdersForSKU(parseCtx, parseWarehouseID, parseSku)
	if parseErr != nil {
		return warehouseItemDetailPageData{}, parseErr
	}
	return warehouseItemDetailPageData{Warehouse: parseWarehouse, Item: parseItem, Product: parseProduct, Network: parseNetwork, Orders: parseOrders, Filters: parseFilters}, nil
}

func normalizeWarehouseItemSort(parseSortKey string) string {
	// Canonicalize item-detail sort keys to the small supported set so deep links stay deterministic.
	switch strings.TrimSpace(strings.ToLower(parseSortKey)) {
	case "warehouse", "available", "cover", "inbound", "demand", "share", "updated":
		return strings.TrimSpace(strings.ToLower(parseSortKey))
	default:
		return "updated"
	}
}

func normalizeWarehouseItemDirection(parseSortKey string, parseDirection string) string {
	// Direction defaults depend on metric semantics: text and "cover" views read naturally ascending, while
	// time/volume-centric fields default to descending so highest-pressure lanes surface first.
	switch strings.TrimSpace(strings.ToLower(parseDirection)) {
	case "asc", "desc":
		return strings.TrimSpace(strings.ToLower(parseDirection))
	}
	switch normalizeWarehouseItemSort(parseSortKey) {
	case "warehouse":
		return "asc"
	case "cover":
		return "asc"
	default:
		return "desc"
	}
}

func sortWarehouseItemRows(parseItems []repository.InventoryRow, parseSortKey string, parseDirection string) {
	parseKey := normalizeWarehouseItemSort(parseSortKey)
	parseDir := normalizeWarehouseItemDirection(parseKey, parseDirection)
	parseCompareText := func(parseLeft string, parseRight string) int {
		parseLeftValue := strings.ToLower(strings.TrimSpace(parseLeft))
		parseRightValue := strings.ToLower(strings.TrimSpace(parseRight))
		switch {
		case parseLeftValue < parseRightValue:
			return -1
		case parseLeftValue > parseRightValue:
			return 1
		default:
			return 0
		}
	}
	parseCompareInt := func(parseLeft2 int, parseRight2 int) int {
		switch {
		case parseLeft2 < parseRight2:
			return -1
		case parseLeft2 > parseRight2:
			return 1
		default:
			return 0
		}
	}
	parseCompareUpdated := func(parseLeft3 string, parseRight3 string) int {
		return parseCompareText(parseLeft3, parseRight3)
	}
	sort.Slice(parseItems, func(parseLeft4, parseRight4 int) bool {
		parseLeftItem := parseItems[parseLeft4]
		parseRightItem := parseItems[parseRight4]
		parseResult := 0
		switch parseKey {
		case "warehouse":
			parseResult = parseCompareText(fallbackWarehouseLabel(parseLeftItem), fallbackWarehouseLabel(parseRightItem))
		case "available":
			parseResult = parseCompareInt(parseLeftItem.Available, parseRightItem.Available)
		case "cover":
			parseResult = parseCompareInt(parseLeftItem.CoverDays, parseRightItem.CoverDays)
		case "inbound":
			parseResult = parseCompareInt(parseLeftItem.Inbound, parseRightItem.Inbound)
		case "demand":
			parseResult = parseCompareInt(parseLeftItem.WeeklyUnits, parseRightItem.WeeklyUnits)
		case "share":
			parseResult = parseCompareInt(parseLeftItem.RegionalShare, parseRightItem.RegionalShare)
		default:
			parseResult = parseCompareUpdated(parseLeftItem.UpdatedAt, parseRightItem.UpdatedAt)
		}
		if parseResult == 0 {
			parseResult = parseCompareText(parseLeftItem.Title, parseRightItem.Title)
		}
		if parseResult == 0 {
			parseResult = parseCompareText(parseLeftItem.WarehouseID, parseRightItem.WarehouseID)
		}
		if parseDir == "asc" {
			return parseResult < 0
		}
		return parseResult > 0
	})
}

func fallbackWarehouseLabel(parseItem repository.InventoryRow) string {
	if strings.TrimSpace(parseItem.WarehouseName) != "" {
		return parseItem.WarehouseName
	}
	return parseItem.WarehouseID
}

func filterWarehouseInventoryRows(parseItems []repository.InventoryRow, parseFilters map[string]string) []repository.InventoryRow {
	if len(parseItems) == 0 {
		return parseItems
	}
	parseQuery := strings.TrimSpace(strings.ToLower(parseFilters["q"]))
	parseStatus := strings.TrimSpace(strings.ToLower(parseFilters["status"]))
	parseFiltered := make([]repository.InventoryRow, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if parseStatus != "" && parseStatus != "all" && strings.TrimSpace(strings.ToLower(parseItem.Status)) != parseStatus {
			continue
		}
		if parseQuery != "" {
			parseHaystack := strings.ToLower(strings.Join([]string{parseItem.SKU, parseItem.Title, parseItem.Category, parseItem.MarketPressure}, " "))
			if !strings.Contains(parseHaystack, parseQuery) {
				continue
			}
		}
		parseFiltered = append(parseFiltered, parseItem)
	}
	return parseFiltered
}

func sortWarehouseInventoryRows(parseItems []repository.InventoryRow, parseSortKey string) {
	switch strings.TrimSpace(strings.ToLower(parseSortKey)) {
	case "available":
		sort.Slice(parseItems, func(parseLeft, parseRight int) bool {
			if parseItems[parseLeft].Available == parseItems[parseRight].Available {
				return parseItems[parseLeft].Title < parseItems[parseRight].Title
			}
			return parseItems[parseLeft].Available < parseItems[parseRight].Available
		})
	case "demand":
		sort.Slice(parseItems, func(parseLeft2, parseRight2 int) bool {
			if parseItems[parseLeft2].DemandScore == parseItems[parseRight2].DemandScore {
				return parseItems[parseLeft2].Title < parseItems[parseRight2].Title
			}
			return parseItems[parseLeft2].DemandScore > parseItems[parseRight2].DemandScore
		})
	case "revenue":
		sort.Slice(parseItems, func(parseLeft3, parseRight3 int) bool {
			if parseItems[parseLeft3].WeeklyRevenue == parseItems[parseRight3].WeeklyRevenue {
				return parseItems[parseLeft3].Title < parseItems[parseRight3].Title
			}
			return parseItems[parseLeft3].WeeklyRevenue > parseItems[parseRight3].WeeklyRevenue
		})
	default:
		sort.Slice(parseItems, func(parseLeft4, parseRight4 int) bool {
			if parseItems[parseLeft4].UpdatedAt == parseItems[parseRight4].UpdatedAt {
				return parseItems[parseLeft4].Title < parseItems[parseRight4].Title
			}
			return parseItems[parseLeft4].UpdatedAt > parseItems[parseRight4].UpdatedAt
		})
	}
}

func findWarehouseInventoryRow(parseItems []repository.InventoryRow, parseWarehouseID string) (repository.InventoryRow, bool) {
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.WarehouseID), strings.TrimSpace(parseWarehouseID)) {
			return parseItem, true
		}
	}
	return repository.InventoryRow{}, false
}

func (parseS *atlasServer) warehouseOrdersForSKU(parseCtx context.Context, parseWarehouseID string, parseSku string) ([]serverdb.PurchaseOrderRecord, error) {
	parseOrders, parseErr := parseS.store.PurchaseOrdersByWarehouse(parseCtx, parseWarehouseID)
	if parseErr != nil {
		return nil, fmt.Errorf("load warehouse sku orders: %w", parseErr)
	}
	parseFiltered := make([]serverdb.PurchaseOrderRecord, 0, len(parseOrders))
	for _, parseOrder := range parseOrders {
		parseDetail, parseErr2 := parseS.store.PurchaseOrderDetail(parseCtx, parseOrder.ID)
		if parseErr2 != nil {
			return nil, fmt.Errorf("load purchase order detail: %w", parseErr2)
		}
		for _, parseLine := range parseDetail.Lines {
			if strings.EqualFold(strings.TrimSpace(parseLine.ProductSKU), strings.TrimSpace(parseSku)) {
				parseFiltered = append(parseFiltered, parseOrder)
				break
			}
		}
	}
	return parseFiltered, nil
}
