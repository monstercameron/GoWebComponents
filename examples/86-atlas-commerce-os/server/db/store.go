package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
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

func NewStore(database *sql.DB) *Store {
	return &Store{db: database}
}

func (s *Store) Catalog(ctx context.Context, query CatalogQuery) (CatalogPage, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 12
	}
	where := []string{"1=1"}
	args := []any{}
	if trimmed := strings.TrimSpace(query.Search); trimmed != "" {
		where = append(where, `(title like ? or summary like ? or slug like ?)`)
		wildcard := "%" + trimmed + "%"
		args = append(args, wildcard, wildcard, wildcard)
	}
	if trimmed := strings.TrimSpace(query.Category); trimmed != "" && trimmed != "all" {
		where = append(where, `category = ?`)
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(query.Warehouse); trimmed != "" && trimmed != "all" {
		where = append(where, `exists (select 1 from inventory_levels il where il.product_sku = products.sku and il.warehouse_id = ?)`)
		args = append(args, trimmed)
	}
	countQuery := `select count(*) from products where ` + strings.Join(where, " and ")
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return CatalogPage{}, fmt.Errorf("count catalog rows: %w", err)
	}
	orderBy := `title asc`
	if strings.EqualFold(strings.TrimSpace(query.Sort), "warehouse") {
		orderBy = `slug asc`
	}
	offset := (query.Page - 1) * query.PageSize
	rows, err := s.db.QueryContext(ctx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where `+strings.Join(where, " and ")+` order by `+orderBy+` limit ? offset ?`, append(args, query.PageSize, offset)...)
	if err != nil {
		return CatalogPage{}, fmt.Errorf("query catalog rows: %w", err)
	}
	defer rows.Close()
	items := []repository.Product{}
	for rows.Next() {
		var item repository.Product
		if err := rows.Scan(&item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.Status, &item.Summary, &item.SEODescription); err != nil {
			return CatalogPage{}, fmt.Errorf("scan catalog row: %w", err)
		}
		items = append(items, item)
	}
	return CatalogPage{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total, Query: query}, nil
}

func (s *Store) ProductBySlug(ctx context.Context, slug string) (repository.Product, error) {
	var item repository.Product
	err := s.db.QueryRowContext(ctx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where slug = ?`, strings.TrimSpace(slug)).Scan(&item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.Status, &item.Summary, &item.SEODescription)
	if err != nil {
		return repository.Product{}, fmt.Errorf("query product by slug: %w", err)
	}
	return item, nil
}

func (s *Store) Warehouses(ctx context.Context) ([]Warehouse, error) {
	rows, err := s.db.QueryContext(ctx, `select id, slug, name, region, service_level, public_summary from warehouses order by name asc`)
	if err != nil {
		return nil, fmt.Errorf("query warehouses: %w", err)
	}
	defer rows.Close()
	items := []Warehouse{}
	for rows.Next() {
		var item Warehouse
		if err := rows.Scan(&item.ID, &item.Slug, &item.Name, &item.Region, &item.ServiceLevel, &item.PublicSummary); err != nil {
			return nil, fmt.Errorf("scan warehouse: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) WarehouseBySlug(ctx context.Context, slug string) (Warehouse, error) {
	var item Warehouse
	err := s.db.QueryRowContext(ctx, `select id, slug, name, region, service_level, public_summary from warehouses where slug = ?`, strings.TrimSpace(slug)).Scan(&item.ID, &item.Slug, &item.Name, &item.Region, &item.ServiceLevel, &item.PublicSummary)
	if err != nil {
		return Warehouse{}, fmt.Errorf("query warehouse by slug: %w", err)
	}
	return item, nil
}

func (s *Store) Availability(ctx context.Context, warehouseSlug string, productSlug string) (WarehouseAvailability, error) {
	warehouse, err := s.WarehouseBySlug(ctx, warehouseSlug)
	if err != nil {
		return WarehouseAvailability{}, err
	}
	product, err := s.ProductBySlug(ctx, productSlug)
	if err != nil {
		return WarehouseAvailability{}, err
	}
	result := WarehouseAvailability{Warehouse: warehouse, Product: product}
	err = s.db.QueryRowContext(ctx, `select available, inbound, status from inventory_levels where warehouse_id = ? and product_sku = ?`, warehouse.ID, product.SKU).Scan(&result.Available, &result.Inbound, &result.Status)
	if err != nil {
		return WarehouseAvailability{}, fmt.Errorf("query availability: %w", err)
	}
	return result, nil
}

func (s *Store) InventoryList(ctx context.Context, query repository.InventoryQuery) ([]repository.InventoryRow, error) {
	where := []string{"1=1"}
	args := []any{}
	if trimmed := strings.TrimSpace(query.Warehouse); trimmed != "" {
		where = append(where, `il.warehouse_id = ?`)
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(query.StockHealth); trimmed != "" && trimmed != "all" {
		where = append(where, `il.status = ?`)
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(query.Search); trimmed != "" {
		where = append(where, `(p.sku like ? or p.title like ?)`)
		wildcard := "%" + trimmed + "%"
		args = append(args, wildcard, wildcard)
	}
	orderBy := `il.available asc, p.sku asc`
	if strings.EqualFold(query.SortKey, "updated") {
		orderBy = `il.updated_at desc, p.sku asc`
	} else if strings.EqualFold(query.SortKey, "inbound") {
		orderBy = `il.inbound desc, p.sku asc`
	}
	rows, err := s.db.QueryContext(ctx, `select il.id, p.sku, p.slug, p.title, p.category, p.price_cents, p.status, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where `+strings.Join(where, " and ")+` order by `+orderBy, args...)
	if err != nil {
		return nil, fmt.Errorf("query inventory list: %w", err)
	}
	defer rows.Close()
	items := []repository.InventoryRow{}
	for rows.Next() {
		var item repository.InventoryRow
		if err := rows.Scan(&item.ID, &item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.ProductStatus, &item.WarehouseID, &item.WarehouseName, &item.OnHand, &item.Reserved, &item.Available, &item.CoverDays, &item.Inbound, &item.Damaged, &item.ReorderPoint, &item.SafetyStock, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan inventory row: %w", err)
		}
		enrichInventoryRow(&item)
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) InventoryBySKU(ctx context.Context, sku string) (repository.InventoryRow, error) {
	var item repository.InventoryRow
	err := s.db.QueryRowContext(ctx, `select il.id, p.sku, p.slug, p.title, p.category, p.price_cents, p.status, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where p.sku = ? order by il.available desc, il.inbound desc limit 1`, strings.TrimSpace(sku)).Scan(&item.ID, &item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.ProductStatus, &item.WarehouseID, &item.WarehouseName, &item.OnHand, &item.Reserved, &item.Available, &item.CoverDays, &item.Inbound, &item.Damaged, &item.ReorderPoint, &item.SafetyStock, &item.Status, &item.UpdatedAt)
	if err != nil {
		return repository.InventoryRow{}, fmt.Errorf("query inventory by sku: %w", err)
	}
	enrichInventoryRow(&item)
	return item, nil
}

func (s *Store) InventoryRowsBySKU(ctx context.Context, sku string) ([]repository.InventoryRow, error) {
	rows, err := s.db.QueryContext(ctx, `select il.id, p.sku, p.slug, p.title, p.category, p.price_cents, p.status, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where p.sku = ? order by w.name asc`, strings.TrimSpace(sku))
	if err != nil {
		return nil, fmt.Errorf("query inventory rows by sku: %w", err)
	}
	defer rows.Close()
	items := []repository.InventoryRow{}
	for rows.Next() {
		var item repository.InventoryRow
		if err := rows.Scan(&item.ID, &item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.ProductStatus, &item.WarehouseID, &item.WarehouseName, &item.OnHand, &item.Reserved, &item.Available, &item.CoverDays, &item.Inbound, &item.Damaged, &item.ReorderPoint, &item.SafetyStock, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan inventory row by sku: %w", err)
		}
		enrichInventoryRow(&item)
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].WeeklyRevenue == items[right].WeeklyRevenue {
			return items[left].WarehouseName < items[right].WarehouseName
		}
		return items[left].WeeklyRevenue > items[right].WeeklyRevenue
	})
	return items, nil
}

func enrichInventoryRow(item *repository.InventoryRow) {
	if item == nil {
		return
	}
	velocityBase := inventoryWeeklyUnits(item.SKU, item.WarehouseID)
	priceCents := item.PriceCents
	if priceCents < 0 {
		priceCents = 0
	}
	item.WeeklyUnits = velocityBase
	item.WeeklyRevenue = velocityBase * priceCents
	item.SellThrough = clampInt(28+velocityBase*4+item.Reserved*2-item.Available, 12, 96)
	item.DemandScore = clampInt(35+velocityBase*6+item.Reserved*3+item.Inbound-item.Damaged*4, 18, 99)
	item.RegionalShare = inventoryRegionalShare(item.WarehouseID)
	item.ReorderUnits = maxInt(0, velocityBase*2+item.SafetyStock-item.Available-item.Inbound)
	item.MarketPressure = inventoryMarketPressure(item.DemandScore, item.Available, item.Inbound)
	item.MarketSignal = inventoryMarketSignal(item.Category, item.WarehouseID, item.WeeklyUnits)
}

func inventoryWeeklyUnits(sku string, warehouseID string) int {
	base := map[string]int{
		"frame-desk":     8,
		"studio-console": 6,
		"frame-bench":    5,
		"cable-bridge":   14,
	}[strings.TrimSpace(strings.ToLower(sku))]
	if base == 0 {
		base = 4
	}
	switch strings.TrimSpace(strings.ToLower(warehouseID)) {
	case "new-jersey-hub":
		base += 2
	case "illinois-hub":
		base += 1
	case "nevada-hub":
		if strings.Contains(strings.ToLower(sku), "desk") {
			base += 3
		}
	}
	return base
}

func inventoryRegionalShare(warehouseID string) int {
	switch strings.TrimSpace(strings.ToLower(warehouseID)) {
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

func inventoryMarketPressure(demandScore int, available int, inbound int) string {
	if demandScore >= 80 && available <= 6 {
		return "Hot market"
	}
	if demandScore >= 68 || inbound > available {
		return "Growing demand"
	}
	if available > 14 && inbound == 0 {
		return "Softening"
	}
	return "Stable"
}

func inventoryMarketSignal(category string, warehouseID string, weeklyUnits int) string {
	categoryKey := strings.TrimSpace(strings.ToLower(category))
	warehouseKey := strings.TrimSpace(strings.ToLower(warehouseID))
	switch {
	case categoryKey == "accessories" && warehouseKey == "new-jersey-hub":
		return "Eastern accessory bundles are converting fastest this week."
	case categoryKey == "desks" && warehouseKey == "nevada-hub":
		return "West-coast studio projects are pulling larger workstation orders forward."
	case weeklyUnits >= 10:
		return "Commercial demand is accelerating ahead of the next replenishment cycle."
	default:
		return "Demand is steady enough to balance service without emergency transfers."
	}
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func (s *Store) PreferencesByOwner(ctx context.Context, ownerID string) (PreferencesRecord, error) {
	var record PreferencesRecord
	err := s.db.QueryRowContext(ctx, `select owner_id, theme, locale, density, default_warehouse_id from preferences where owner_id = ?`, ownerID).Scan(&record.OwnerID, &record.Theme, &record.Locale, &record.Density, &record.DefaultWarehouseID)
	if err != nil {
		if err == sql.ErrNoRows {
			return PreferencesRecord{OwnerID: ownerID, Theme: "dark", Locale: "en", Density: "compact", DefaultWarehouseID: "new-jersey-hub"}, nil
		}
		return PreferencesRecord{}, fmt.Errorf("query preferences: %w", err)
	}
	return record, nil
}

func (s *Store) SavedViewsByOwner(ctx context.Context, ownerID string) ([]repository.SavedView, error) {
	rows, err := s.db.QueryContext(ctx, `select id, name, scope, sort_key, sort_direction, density, warehouse_id, filters_json from saved_views where owner_id = ? order by name asc`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query saved views: %w", err)
	}
	defer rows.Close()
	items := []repository.SavedView{}
	for rows.Next() {
		var item repository.SavedView
		if err := rows.Scan(&item.ID, &item.Name, &item.Scope, &item.SortKey, &item.SortDirection, &item.Density, &item.WarehouseID, &item.FiltersJSON); err != nil {
			return nil, fmt.Errorf("scan saved view: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}
