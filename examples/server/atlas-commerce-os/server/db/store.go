package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/shared/repository"
)

type Store struct {
	db *sql.DB
}

type CatalogQuery struct {
	Search    string `json:"search,omitempty"`
	Category  string `json:"category,omitempty"`
	Warehouse string `json:"warehouse,omitempty"`
	Sort      string `json:"sort,omitempty"`
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
}

type CatalogPage struct {
	Items    []repository.Product `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
	Total    int                  `json:"total"`
	Query    CatalogQuery         `json:"query"`
}

type Warehouse struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
}

type WarehouseAvailability struct {
	Warehouse Warehouse          `json:"warehouse"`
	Product   repository.Product `json:"product"`
	Available int                `json:"available"`
	Inbound   int                `json:"inbound"`
	Status    string             `json:"status"`
}

type PreferencesRecord struct {
	OwnerID            string `json:"ownerId"`
	Theme              string `json:"theme"`
	Locale             string `json:"locale"`
	Density            string `json:"density"`
	DefaultWarehouseID string `json:"defaultWarehouseId"`
}

func NewStore(parseDatabase *sql.DB) *Store {
	return &Store{db: parseDatabase}
}

func (parseS *Store) Catalog(parseCtx context.Context, parseQuery CatalogQuery) (CatalogPage, error) {
	if parseQuery.Page < 1 {
		parseQuery.Page = 1
	}
	if parseQuery.PageSize < 1 {
		parseQuery.PageSize = 12
	}
	parseWhere := []string{"1=1"}
	parseArgs := []any{}
	if parseTrimmed := strings.TrimSpace(parseQuery.Search); parseTrimmed != "" {
		parseWhere = append(parseWhere, `(title like ? or summary like ? or slug like ?)`)
		parseWildcard := "%" + parseTrimmed + "%"
		parseArgs = append(parseArgs, parseWildcard, parseWildcard, parseWildcard)
	}
	if parseTrimmed2 := strings.TrimSpace(parseQuery.Category); parseTrimmed2 != "" && parseTrimmed2 != "all" {
		parseWhere = append(parseWhere, `category = ?`)
		parseArgs = append(parseArgs, parseTrimmed2)
	}
	if parseTrimmed3 := strings.TrimSpace(parseQuery.Warehouse); parseTrimmed3 != "" && parseTrimmed3 != "all" {
		parseWhere = append(parseWhere, `exists (select 1 from inventory_levels il where il.product_sku = products.sku and il.warehouse_id = ?)`)
		parseArgs = append(parseArgs, parseTrimmed3)
	}
	parseCountQuery := `select count(*) from products where ` + strings.Join(parseWhere, " and ")
	var parseTotal int
	if parseErr := parseS.db.QueryRowContext(parseCtx, parseCountQuery, parseArgs...).Scan(&parseTotal); parseErr != nil {
		return CatalogPage{}, fmt.Errorf("count catalog rows: %w", parseErr)
	}
	parseOrderBy := `title asc`
	if strings.EqualFold(strings.TrimSpace(parseQuery.Sort), "warehouse") {
		parseOrderBy = `slug asc`
	}
	parseOffset := (parseQuery.Page - 1) * parseQuery.PageSize
	parseRows, parseErr2 := parseS.db.QueryContext(parseCtx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where `+strings.Join(parseWhere, " and ")+` order by `+parseOrderBy+` limit ? offset ?`, append(parseArgs, parseQuery.PageSize, parseOffset)...)
	if parseErr2 != nil {
		return CatalogPage{}, fmt.Errorf("query catalog rows: %w", parseErr2)
	}
	defer parseRows.Close()
	parseItems := []repository.Product{}
	for parseRows.Next() {
		var parseItem repository.Product
		if parseErr3 := parseRows.Scan(&parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.Status, &parseItem.Summary, &parseItem.SEODescription); parseErr3 != nil {
			return CatalogPage{}, fmt.Errorf("scan catalog row: %w", parseErr3)
		}
		parseItems = append(parseItems, parseItem)
	}
	return CatalogPage{Items: parseItems, Page: parseQuery.Page, PageSize: parseQuery.PageSize, Total: parseTotal, Query: parseQuery}, nil
}

func (parseS *Store) ProductBySlug(parseCtx context.Context, parseSlug string) (repository.Product, error) {
	var parseItem repository.Product
	parseErr := parseS.db.QueryRowContext(parseCtx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where slug = ?`, strings.TrimSpace(parseSlug)).Scan(&parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.Status, &parseItem.Summary, &parseItem.SEODescription)
	if parseErr != nil {
		return repository.Product{}, fmt.Errorf("query product by slug: %w", parseErr)
	}
	return parseItem, nil
}

func (parseS *Store) Warehouses(parseCtx context.Context) ([]Warehouse, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, slug, name, region, service_level, public_summary from warehouses order by name asc`)
	if parseErr != nil {
		return nil, fmt.Errorf("query warehouses: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []Warehouse{}
	for parseRows.Next() {
		var parseItem Warehouse
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.Slug, &parseItem.Name, &parseItem.Region, &parseItem.ServiceLevel, &parseItem.PublicSummary); parseErr2 != nil {
			return nil, fmt.Errorf("scan warehouse: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) WarehouseBySlug(parseCtx context.Context, parseSlug string) (Warehouse, error) {
	var parseItem Warehouse
	parseErr := parseS.db.QueryRowContext(parseCtx, `select id, slug, name, region, service_level, public_summary from warehouses where slug = ?`, strings.TrimSpace(parseSlug)).Scan(&parseItem.ID, &parseItem.Slug, &parseItem.Name, &parseItem.Region, &parseItem.ServiceLevel, &parseItem.PublicSummary)
	if parseErr != nil {
		return Warehouse{}, fmt.Errorf("query warehouse by slug: %w", parseErr)
	}
	return parseItem, nil
}

func (parseS *Store) Availability(parseCtx context.Context, parseWarehouseSlug string, parseProductSlug string) (WarehouseAvailability, error) {
	parseWarehouse, parseErr := parseS.WarehouseBySlug(parseCtx, parseWarehouseSlug)
	if parseErr != nil {
		return WarehouseAvailability{}, parseErr
	}
	parseProduct, parseErr := parseS.ProductBySlug(parseCtx, parseProductSlug)
	if parseErr != nil {
		return WarehouseAvailability{}, parseErr
	}
	parseResult := WarehouseAvailability{Warehouse: parseWarehouse, Product: parseProduct}
	parseErr = parseS.db.QueryRowContext(parseCtx, `select available, inbound, status from inventory_levels where warehouse_id = ? and product_sku = ?`, parseWarehouse.ID, parseProduct.SKU).Scan(&parseResult.Available, &parseResult.Inbound, &parseResult.Status)
	if parseErr != nil {
		return WarehouseAvailability{}, fmt.Errorf("query availability: %w", parseErr)
	}
	return parseResult, nil
}

func (parseS *Store) InventoryList(parseCtx context.Context, parseQuery repository.InventoryQuery) ([]repository.InventoryRow, error) {
	parseWhere := []string{"1=1"}
	parseArgs := []any{}
	if parseTrimmed := strings.TrimSpace(parseQuery.Warehouse); parseTrimmed != "" {
		parseWhere = append(parseWhere, `il.warehouse_id = ?`)
		parseArgs = append(parseArgs, parseTrimmed)
	}
	if parseTrimmed2 := strings.TrimSpace(parseQuery.StockHealth); parseTrimmed2 != "" && parseTrimmed2 != "all" {
		parseWhere = append(parseWhere, `il.status = ?`)
		parseArgs = append(parseArgs, parseTrimmed2)
	}
	if parseTrimmed3 := strings.TrimSpace(parseQuery.Search); parseTrimmed3 != "" {
		parseWhere = append(parseWhere, `(p.sku like ? or p.title like ?)`)
		parseWildcard := "%" + parseTrimmed3 + "%"
		parseArgs = append(parseArgs, parseWildcard, parseWildcard)
	}
	parseOrderBy := `il.available asc, p.sku asc`
	if strings.EqualFold(parseQuery.SortKey, "updated") {
		parseOrderBy = `il.updated_at desc, p.sku asc`
	} else if strings.EqualFold(parseQuery.SortKey, "inbound") {
		parseOrderBy = `il.inbound desc, p.sku asc`
	}
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select il.id, p.sku, p.slug, p.title, p.category, p.price_cents, p.status, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where `+strings.Join(parseWhere, " and ")+` order by `+parseOrderBy, parseArgs...)
	if parseErr != nil {
		return nil, fmt.Errorf("query inventory list: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []repository.InventoryRow{}
	for parseRows.Next() {
		var parseItem repository.InventoryRow
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.ProductStatus, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.OnHand, &parseItem.Reserved, &parseItem.Available, &parseItem.CoverDays, &parseItem.Inbound, &parseItem.Damaged, &parseItem.ReorderPoint, &parseItem.SafetyStock, &parseItem.Status, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan inventory row: %w", parseErr2)
		}
		enrichInventoryRow(&parseItem)
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) InventoryBySKU(parseCtx context.Context, parseSku string) (repository.InventoryRow, error) {
	var parseItem repository.InventoryRow
	parseErr := parseS.db.QueryRowContext(parseCtx, `select il.id, p.sku, p.slug, p.title, p.category, p.price_cents, p.status, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where p.sku = ? order by il.available desc, il.inbound desc limit 1`, strings.TrimSpace(parseSku)).Scan(&parseItem.ID, &parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.ProductStatus, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.OnHand, &parseItem.Reserved, &parseItem.Available, &parseItem.CoverDays, &parseItem.Inbound, &parseItem.Damaged, &parseItem.ReorderPoint, &parseItem.SafetyStock, &parseItem.Status, &parseItem.UpdatedAt)
	if parseErr != nil {
		return repository.InventoryRow{}, fmt.Errorf("query inventory by sku: %w", parseErr)
	}
	enrichInventoryRow(&parseItem)
	return parseItem, nil
}

func (parseS *Store) InventoryRowsBySKU(parseCtx context.Context, parseSku string) ([]repository.InventoryRow, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select il.id, p.sku, p.slug, p.title, p.category, p.price_cents, p.status, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where p.sku = ? order by w.name asc`, strings.TrimSpace(parseSku))
	if parseErr != nil {
		return nil, fmt.Errorf("query inventory rows by sku: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []repository.InventoryRow{}
	for parseRows.Next() {
		var parseItem repository.InventoryRow
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.ProductStatus, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.OnHand, &parseItem.Reserved, &parseItem.Available, &parseItem.CoverDays, &parseItem.Inbound, &parseItem.Damaged, &parseItem.ReorderPoint, &parseItem.SafetyStock, &parseItem.Status, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan inventory row by sku: %w", parseErr2)
		}
		enrichInventoryRow(&parseItem)
		parseItems = append(parseItems, parseItem)
	}
	if len(parseItems) == 0 {
		return nil, sql.ErrNoRows
	}
	sort.Slice(parseItems, func(parseLeft, parseRight int) bool {
		if parseItems[parseLeft].WeeklyRevenue == parseItems[parseRight].WeeklyRevenue {
			return parseItems[parseLeft].WarehouseName < parseItems[parseRight].WarehouseName
		}
		return parseItems[parseLeft].WeeklyRevenue > parseItems[parseRight].WeeklyRevenue
	})
	return parseItems, nil
}

func enrichInventoryRow(parseItem *repository.InventoryRow) {
	if parseItem == nil {
		return
	}
	parseVelocityBase := inventoryWeeklyUnits(parseItem.SKU, parseItem.WarehouseID)
	parsePriceCents := max(parseItem.PriceCents, 0)
	parseItem.WeeklyUnits = parseVelocityBase
	parseItem.WeeklyRevenue = parseVelocityBase * parsePriceCents
	parseItem.SellThrough = clampInt(28+parseVelocityBase*4+parseItem.Reserved*2-parseItem.Available, 12, 96)
	parseItem.DemandScore = clampInt(35+parseVelocityBase*6+parseItem.Reserved*3+parseItem.Inbound-parseItem.Damaged*4, 18, 99)
	parseItem.RegionalShare = inventoryRegionalShare(parseItem.WarehouseID)
	parseItem.ReorderUnits = maxInt(0, parseVelocityBase*2+parseItem.SafetyStock-parseItem.Available-parseItem.Inbound)
	parseItem.MarketPressure = inventoryMarketPressure(parseItem.DemandScore, parseItem.Available, parseItem.Inbound)
	parseItem.MarketSignal = inventoryMarketSignal(parseItem.Category, parseItem.WarehouseID, parseItem.WeeklyUnits)
}

func inventoryWeeklyUnits(parseSku string, parseWarehouseID string) int {
	parseBase := map[string]int{
		"frame-desk":     8,
		"studio-console": 6,
		"frame-bench":    5,
		"cable-bridge":   14,
	}[strings.TrimSpace(strings.ToLower(parseSku))]
	if parseBase == 0 {
		parseBase = 4
	}
	switch strings.TrimSpace(strings.ToLower(parseWarehouseID)) {
	case "new-jersey-hub":
		parseBase += 2
	case "illinois-hub":
		parseBase += 1
	case "nevada-hub":
		if strings.Contains(strings.ToLower(parseSku), "desk") {
			parseBase += 3
		}
	}
	return parseBase
}

func inventoryRegionalShare(parseWarehouseID string) int {
	switch strings.TrimSpace(strings.ToLower(parseWarehouseID)) {
	case "new-jersey-hub":
		return 42
	case "illinois-hub":
		return 33
	case "nevada-hub":
		return 25
	default:
		return 20
	}
}

func inventoryMarketPressure(parseDemandScore int, parseAvailable int, parseInbound int) string {
	if parseDemandScore >= 80 && parseAvailable <= 6 {
		return "Hot market"
	}
	if parseDemandScore >= 68 || parseInbound > parseAvailable {
		return "Growing demand"
	}
	if parseAvailable > 14 && parseInbound == 0 {
		return "Softening"
	}
	return "Stable"
}

func inventoryMarketSignal(parseCategory string, parseWarehouseID string, parseWeeklyUnits int) string {
	parseCategoryKey := strings.TrimSpace(strings.ToLower(parseCategory))
	parseWarehouseKey := strings.TrimSpace(strings.ToLower(parseWarehouseID))
	switch {
	case parseCategoryKey == "accessories" && parseWarehouseKey == "new-jersey-hub":
		return "Eastern accessory bundles are converting fastest this week."
	case parseCategoryKey == "desks" && parseWarehouseKey == "nevada-hub":
		return "West-coast studio projects are pulling larger workstation orders forward."
	case parseWeeklyUnits >= 10:
		return "Commercial demand is accelerating ahead of the next replenishment cycle."
	default:
		return "Demand is steady enough to balance service without emergency transfers."
	}
}

func clampInt(parseValue int, parseMinValue int, parseMaxValue int) int {
	if parseValue < parseMinValue {
		return parseMinValue
	}
	if parseValue > parseMaxValue {
		return parseMaxValue
	}
	return parseValue
}

func maxInt(parseLeft int, parseRight int) int {
	if parseLeft > parseRight {
		return parseLeft
	}
	return parseRight
}

func (parseS *Store) PreferencesByOwner(parseCtx context.Context, parseOwnerID string) (PreferencesRecord, error) {
	var parseRecord PreferencesRecord
	parseErr := parseS.db.QueryRowContext(parseCtx, `select owner_id, theme, locale, density, default_warehouse_id from preferences where owner_id = ?`, parseOwnerID).Scan(&parseRecord.OwnerID, &parseRecord.Theme, &parseRecord.Locale, &parseRecord.Density, &parseRecord.DefaultWarehouseID)
	if parseErr != nil {
		if parseErr == sql.ErrNoRows {
			return PreferencesRecord{OwnerID: parseOwnerID, Theme: "dark", Locale: "en", Density: "compact", DefaultWarehouseID: "new-jersey-hub"}, nil
		}
		return PreferencesRecord{}, fmt.Errorf("query preferences: %w", parseErr)
	}
	return parseRecord, nil
}

func (parseS *Store) SavedViewsByOwner(parseCtx context.Context, parseOwnerID string) ([]repository.SavedView, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, name, scope, sort_key, sort_direction, density, warehouse_id, filters_json from saved_views where owner_id = ? order by name asc`, parseOwnerID)
	if parseErr != nil {
		return nil, fmt.Errorf("query saved views: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []repository.SavedView{}
	for parseRows.Next() {
		var parseItem repository.SavedView
		var parseWarehouseID sql.NullString
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.Name, &parseItem.Scope, &parseItem.SortKey, &parseItem.SortDirection, &parseItem.Density, &parseWarehouseID, &parseItem.FiltersJSON); parseErr2 != nil {
			return nil, fmt.Errorf("scan saved view: %w", parseErr2)
		}
		if parseWarehouseID.Valid {
			parseItem.WarehouseID = parseWarehouseID.String
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}
