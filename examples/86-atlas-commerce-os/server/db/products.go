package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type ProductAdminQuery struct {
	Search   string
	Category string
	Status   string
	Sort     string
}

type ProductAdminRecord struct {
	SKU            string `json:"sku"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	PriceCents     int    `json:"priceCents"`
	Status         string `json:"status"`
	Finish         string `json:"finish"`
	Summary        string `json:"summary"`
	Details        string `json:"details"`
	SEOTitle       string `json:"seoTitle"`
	SEODescription string `json:"seoDescription"`
	WarehouseID    string `json:"warehouseId"`
	WarehouseName  string `json:"warehouseName"`
	Available      int    `json:"available"`
	Inbound        int    `json:"inbound"`
	Volume         int    `json:"volume"`
	HubCount       int    `json:"hubCount"`
	UpdatedAt      string `json:"updatedAt"`
}

type CreateProductInput struct {
	SKU            string
	Slug           string
	Title          string
	Category       string
	PriceCents     int
	Status         string
	Finish         string
	Summary        string
	Details        string
	SEOTitle       string
	SEODescription string
	WarehouseID    string
	Available      int
	Inbound        int
}

type UpdateProductInput struct {
	SKU              string
	Slug             string
	Title            string
	Category         string
	PriceCents       int
	Status           string
	Finish           string
	Summary          string
	Details          string
	SEOTitle         string
	SEODescription   string
	CurrentWarehouse string
	WarehouseID      string
	Available        int
	Inbound          int
}

func (parseS *Store) ProductAdminList(parseCtx context.Context, parseQuery ProductAdminQuery) ([]ProductAdminRecord, error) {
	parseWhere := []string{"1=1"}
	parseArgs := []any{}
	if parseTrimmed := strings.TrimSpace(parseQuery.Search); parseTrimmed != "" {
		parseWhere = append(parseWhere, `(sku like ? or slug like ? or title like ? or summary like ?)`)
		parseWildcard := "%" + parseTrimmed + "%"
		parseArgs = append(parseArgs, parseWildcard, parseWildcard, parseWildcard, parseWildcard)
	}
	if parseTrimmed2 := strings.TrimSpace(parseQuery.Category); parseTrimmed2 != "" && parseTrimmed2 != "all" {
		parseWhere = append(parseWhere, `category = ?`)
		parseArgs = append(parseArgs, parseTrimmed2)
	}
	if parseTrimmed3 := strings.TrimSpace(parseQuery.Status); parseTrimmed3 != "" && parseTrimmed3 != "all" {
		parseWhere = append(parseWhere, `status = ?`)
		parseArgs = append(parseArgs, parseTrimmed3)
	}
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select sku from products where `+strings.Join(parseWhere, " and "), parseArgs...)
	if parseErr != nil {
		return nil, fmt.Errorf("query product admin list: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []ProductAdminRecord{}
	for parseRows.Next() {
		var parseSku string
		if parseErr2 := parseRows.Scan(&parseSku); parseErr2 != nil {
			return nil, fmt.Errorf("scan product admin sku: %w", parseErr2)
		}
		parseItem, parseErr3 := parseS.productAdminBySKU(parseCtx, parseSku)
		if parseErr3 != nil {
			return nil, parseErr3
		}
		parseItems = append(parseItems, parseItem)
	}
	sortProductAdminItems(parseItems, parseQuery.Sort)
	return parseItems, nil
}

func (parseS *Store) ProductAdminBySlug(parseCtx context.Context, parseSlug string) (ProductAdminRecord, error) {
	var parseSku string
	if parseErr := parseS.db.QueryRowContext(parseCtx, `select sku from products where slug = ?`, strings.TrimSpace(parseSlug)).Scan(&parseSku); parseErr != nil {
		return ProductAdminRecord{}, fmt.Errorf("query product admin by slug: %w", parseErr)
	}
	return parseS.productAdminBySKU(parseCtx, parseSku)
}

func (parseS *Store) CreateProduct(parseCtx context.Context, parseInput CreateProductInput) (ProductAdminRecord, error) {
	parsePrepared := prepareCreateProductInput(parseInput)
	parseTx, parseErr := parseS.db.BeginTx(parseCtx, nil)
	if parseErr != nil {
		return ProductAdminRecord{}, fmt.Errorf("begin create product transaction: %w", parseErr)
	}
	defer parseTx.Rollback()

	parseNow := timestampNow()
	if _, parseErr2 := parseTx.ExecContext(parseCtx, `insert into products(sku, slug, title, category, price_cents, status, finish, summary, details, seo_title, seo_description, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, parsePrepared.SKU, parsePrepared.Slug, parsePrepared.Title, parsePrepared.Category, parsePrepared.PriceCents, parsePrepared.Status, parsePrepared.Finish, parsePrepared.Summary, parsePrepared.Details, parsePrepared.SEOTitle, parsePrepared.SEODescription, parseNow, parseNow); parseErr2 != nil {
		return ProductAdminRecord{}, fmt.Errorf("insert product: %w", parseErr2)
	}
	if _, parseErr3 := parseTx.ExecContext(parseCtx, `insert into inventory_levels(id, product_sku, warehouse_id, on_hand, reserved, available, inbound, damaged, reorder_point, safety_stock, status, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, makeID("inv"), parsePrepared.SKU, parsePrepared.WarehouseID, parsePrepared.Available, 0, parsePrepared.Available, parsePrepared.Inbound, 0, 12, 6, inventoryStatusForProduct(parsePrepared.Status, parsePrepared.Available), parseNow); parseErr3 != nil {
		return ProductAdminRecord{}, fmt.Errorf("insert product inventory: %w", parseErr3)
	}
	if parseErr4 := parseTx.Commit(); parseErr4 != nil {
		return ProductAdminRecord{}, fmt.Errorf("commit create product transaction: %w", parseErr4)
	}
	return parseS.ProductAdminBySlug(parseCtx, parsePrepared.Slug)
}

func (parseS *Store) UpdateProduct(parseCtx context.Context, parseCurrentSlug string, parseInput UpdateProductInput) (ProductAdminRecord, error) {
	parseCurrent, parseErr := parseS.ProductAdminBySlug(parseCtx, parseCurrentSlug)
	if parseErr != nil {
		return ProductAdminRecord{}, parseErr
	}
	parsePrepared := prepareUpdateProductInput(parseCurrent, parseInput)
	parseTx, parseErr := parseS.db.BeginTx(parseCtx, nil)
	if parseErr != nil {
		return ProductAdminRecord{}, fmt.Errorf("begin update product transaction: %w", parseErr)
	}
	defer parseTx.Rollback()

	parseNow := timestampNow()
	if _, parseErr2 := parseTx.ExecContext(parseCtx, `update products set sku = ?, slug = ?, title = ?, category = ?, price_cents = ?, status = ?, finish = ?, summary = ?, details = ?, seo_title = ?, seo_description = ?, updated_at = ? where slug = ?`, parsePrepared.SKU, parsePrepared.Slug, parsePrepared.Title, parsePrepared.Category, parsePrepared.PriceCents, parsePrepared.Status, parsePrepared.Finish, parsePrepared.Summary, parsePrepared.Details, parsePrepared.SEOTitle, parsePrepared.SEODescription, parseNow, strings.TrimSpace(parseCurrentSlug)); parseErr2 != nil {
		return ProductAdminRecord{}, fmt.Errorf("update product: %w", parseErr2)
	}
	if parsePrepared.CurrentWarehouse != "" && parsePrepared.CurrentWarehouse != parsePrepared.WarehouseID {
		if _, parseErr3 := parseTx.ExecContext(parseCtx, `delete from inventory_levels where product_sku = ? and warehouse_id = ?`, parseCurrent.SKU, parsePrepared.CurrentWarehouse); parseErr3 != nil {
			return ProductAdminRecord{}, fmt.Errorf("move managed inventory row: %w", parseErr3)
		}
	}
	if _, parseErr4 := parseTx.ExecContext(parseCtx, `insert into inventory_levels(id, product_sku, warehouse_id, on_hand, reserved, available, inbound, damaged, reorder_point, safety_stock, status, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) on conflict(product_sku, warehouse_id) do update set on_hand = excluded.on_hand, reserved = excluded.reserved, available = excluded.available, inbound = excluded.inbound, damaged = excluded.damaged, reorder_point = excluded.reorder_point, safety_stock = excluded.safety_stock, status = excluded.status, updated_at = excluded.updated_at`, makeID("inv"), parsePrepared.SKU, parsePrepared.WarehouseID, parsePrepared.Available, 0, parsePrepared.Available, parsePrepared.Inbound, 0, 12, 6, inventoryStatusForProduct(parsePrepared.Status, parsePrepared.Available), parseNow); parseErr4 != nil {
		return ProductAdminRecord{}, fmt.Errorf("upsert product inventory: %w", parseErr4)
	}
	if parseErr5 := parseTx.Commit(); parseErr5 != nil {
		return ProductAdminRecord{}, fmt.Errorf("commit update product transaction: %w", parseErr5)
	}
	return parseS.ProductAdminBySlug(parseCtx, parsePrepared.Slug)
}

func (parseS *Store) DeleteProduct(parseCtx context.Context, parseSlug string) error {
	parseResult, parseErr := parseS.db.ExecContext(parseCtx, `delete from products where slug = ?`, strings.TrimSpace(parseSlug))
	if parseErr != nil {
		return fmt.Errorf("delete product: %w", parseErr)
	}
	parseDeleted, parseErr := parseResult.RowsAffected()
	if parseErr != nil {
		return fmt.Errorf("count deleted products: %w", parseErr)
	}
	if parseDeleted == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (parseS *Store) productAdminBySKU(parseCtx context.Context, parseSku string) (ProductAdminRecord, error) {
	var parseItem ProductAdminRecord
	parseErr := parseS.db.QueryRowContext(parseCtx, `select sku, slug, title, category, price_cents, status, finish, summary, details, seo_title, seo_description, updated_at from products where sku = ?`, strings.TrimSpace(parseSku)).Scan(&parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.Status, &parseItem.Finish, &parseItem.Summary, &parseItem.Details, &parseItem.SEOTitle, &parseItem.SEODescription, &parseItem.UpdatedAt)
	if parseErr != nil {
		return ProductAdminRecord{}, fmt.Errorf("query product admin by sku: %w", parseErr)
	}
	parseWarehouseID, parseWarehouseName, parseAvailable, parseInbound, parseHubCount, parseErr := parseS.productInventorySummary(parseCtx, parseItem.SKU)
	if parseErr != nil {
		return ProductAdminRecord{}, parseErr
	}
	parseItem.WarehouseID = parseWarehouseID
	parseItem.WarehouseName = parseWarehouseName
	parseItem.Available = parseAvailable
	parseItem.Inbound = parseInbound
	parseItem.Volume = parseAvailable + parseInbound
	parseItem.HubCount = parseHubCount
	return parseItem, nil
}

func (parseS *Store) productInventorySummary(parseCtx context.Context, parseSku string) (string, string, int, int, int, error) {
	parseWarehouseID := ""
	parseAvailable := 0
	parseInbound := 0
	if parseErr := parseS.db.QueryRowContext(parseCtx, `select warehouse_id, available, inbound from inventory_levels where product_sku = ? order by available desc, inbound desc, updated_at desc limit 1`, strings.TrimSpace(parseSku)).Scan(&parseWarehouseID, &parseAvailable, &parseInbound); parseErr != nil && parseErr != sql.ErrNoRows {
		return "", "", 0, 0, 0, fmt.Errorf("query product inventory summary: %w", parseErr)
	}
	parseWarehouseName := ""
	if parseWarehouseID != "" {
		parseWarehouse, parseErr2 := parseS.warehouseByID(parseCtx, parseWarehouseID)
		if parseErr2 != nil {
			return "", "", 0, 0, 0, parseErr2
		}
		parseWarehouseName = parseWarehouse.Name
	}
	parseHubCount := 0
	if parseErr3 := parseS.db.QueryRowContext(parseCtx, `select count(*) from inventory_levels where product_sku = ?`, strings.TrimSpace(parseSku)).Scan(&parseHubCount); parseErr3 != nil {
		return "", "", 0, 0, 0, fmt.Errorf("count product inventory hubs: %w", parseErr3)
	}
	return parseWarehouseID, parseWarehouseName, parseAvailable, parseInbound, parseHubCount, nil
}

func prepareCreateProductInput(parseInput CreateProductInput) CreateProductInput {
	parsePrepared := parseInput
	parsePrepared.SKU = strings.TrimSpace(parsePrepared.SKU)
	parsePrepared.Slug = strings.TrimSpace(parsePrepared.Slug)
	parsePrepared.Title = nonEmptyString(parsePrepared.Title, parsePrepared.SKU)
	parsePrepared.Category = nonEmptyString(parsePrepared.Category, "desks")
	parsePrepared.Status = nonEmptyString(parsePrepared.Status, "in_stock")
	parsePrepared.Finish = nonEmptyString(parsePrepared.Finish, "Graphite oak")
	parsePrepared.Summary = nonEmptyString(parsePrepared.Summary, "Atlas catalog item")
	parsePrepared.Details = nonEmptyString(parsePrepared.Details, parsePrepared.Summary)
	parsePrepared.SEOTitle = nonEmptyString(parsePrepared.SEOTitle, "Atlas "+parsePrepared.Title)
	parsePrepared.SEODescription = nonEmptyString(parsePrepared.SEODescription, parsePrepared.Summary)
	parsePrepared.WarehouseID = nonEmptyString(parsePrepared.WarehouseID, "new-jersey-hub")
	if parsePrepared.PriceCents < 0 {
		parsePrepared.PriceCents = 0
	}
	if parsePrepared.Available < 0 {
		parsePrepared.Available = 0
	}
	if parsePrepared.Inbound < 0 {
		parsePrepared.Inbound = 0
	}
	return parsePrepared
}

func prepareUpdateProductInput(parseCurrent ProductAdminRecord, parseInput UpdateProductInput) UpdateProductInput {
	parsePrepared := parseInput
	parsePrepared.SKU = parseCurrent.SKU
	parsePrepared.Slug = nonEmptyString(parsePrepared.Slug, parseCurrent.Slug)
	parsePrepared.Title = nonEmptyString(parsePrepared.Title, parseCurrent.Title)
	parsePrepared.Category = nonEmptyString(parsePrepared.Category, parseCurrent.Category)
	parsePrepared.Status = nonEmptyString(parsePrepared.Status, parseCurrent.Status)
	parsePrepared.Finish = nonEmptyString(parsePrepared.Finish, parseCurrent.Finish)
	parsePrepared.Summary = nonEmptyString(parsePrepared.Summary, parseCurrent.Summary)
	parsePrepared.Details = nonEmptyString(parsePrepared.Details, parseCurrent.Details)
	parsePrepared.SEOTitle = nonEmptyString(parsePrepared.SEOTitle, parseCurrent.SEOTitle)
	parsePrepared.SEODescription = nonEmptyString(parsePrepared.SEODescription, parseCurrent.SEODescription)
	parsePrepared.CurrentWarehouse = nonEmptyString(parsePrepared.CurrentWarehouse, parseCurrent.WarehouseID)
	parsePrepared.WarehouseID = nonEmptyString(parsePrepared.WarehouseID, parseCurrent.WarehouseID)
	if parsePrepared.PriceCents < 0 {
		parsePrepared.PriceCents = parseCurrent.PriceCents
	}
	if parsePrepared.Available < 0 {
		parsePrepared.Available = parseCurrent.Available
	}
	if parsePrepared.Inbound < 0 {
		parsePrepared.Inbound = parseCurrent.Inbound
	}
	return parsePrepared
}

func sortProductAdminItems(parseItems []ProductAdminRecord, parseSortKey string) {
	switch strings.TrimSpace(strings.ToLower(parseSortKey)) {
	case "price":
		sort.SliceStable(parseItems, func(parseLeft, parseRight int) bool {
			if parseItems[parseLeft].PriceCents == parseItems[parseRight].PriceCents {
				return strings.ToLower(parseItems[parseLeft].Title) < strings.ToLower(parseItems[parseRight].Title)
			}
			return parseItems[parseLeft].PriceCents > parseItems[parseRight].PriceCents
		})
	case "volume":
		sort.SliceStable(parseItems, func(parseLeft2, parseRight2 int) bool {
			if parseItems[parseLeft2].Volume == parseItems[parseRight2].Volume {
				return strings.ToLower(parseItems[parseLeft2].Title) < strings.ToLower(parseItems[parseRight2].Title)
			}
			return parseItems[parseLeft2].Volume > parseItems[parseRight2].Volume
		})
	case "status":
		sort.SliceStable(parseItems, func(parseLeft3, parseRight3 int) bool {
			parseLeftStatus := strings.ToLower(parseItems[parseLeft3].Status)
			parseRightStatus := strings.ToLower(parseItems[parseRight3].Status)
			if parseLeftStatus == parseRightStatus {
				return strings.ToLower(parseItems[parseLeft3].Title) < strings.ToLower(parseItems[parseRight3].Title)
			}
			return parseLeftStatus < parseRightStatus
		})
	default:
		sort.SliceStable(parseItems, func(parseLeft4, parseRight4 int) bool {
			if parseItems[parseLeft4].UpdatedAt == parseItems[parseRight4].UpdatedAt {
				return strings.ToLower(parseItems[parseLeft4].Title) < strings.ToLower(parseItems[parseRight4].Title)
			}
			return parseItems[parseLeft4].UpdatedAt > parseItems[parseRight4].UpdatedAt
		})
	}
}

func inventoryStatusForProduct(parseProductStatus string, parseAvailable int) string {
	if parseAvailable <= 0 {
		return "promise_risk"
	}
	if strings.EqualFold(strings.TrimSpace(parseProductStatus), "low_stock") || parseAvailable <= 3 {
		return "promise_risk"
	}
	return "balanced"
}
