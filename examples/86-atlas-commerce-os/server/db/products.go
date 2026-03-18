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

func (s *Store) ProductAdminList(ctx context.Context, query ProductAdminQuery) ([]ProductAdminRecord, error) {
	where := []string{"1=1"}
	args := []any{}
	if trimmed := strings.TrimSpace(query.Search); trimmed != "" {
		where = append(where, `(sku like ? or slug like ? or title like ? or summary like ?)`)
		wildcard := "%" + trimmed + "%"
		args = append(args, wildcard, wildcard, wildcard, wildcard)
	}
	if trimmed := strings.TrimSpace(query.Category); trimmed != "" && trimmed != "all" {
		where = append(where, `category = ?`)
		args = append(args, trimmed)
	}
	if trimmed := strings.TrimSpace(query.Status); trimmed != "" && trimmed != "all" {
		where = append(where, `status = ?`)
		args = append(args, trimmed)
	}
	rows, err := s.db.QueryContext(ctx, `select sku from products where `+strings.Join(where, " and "), args...)
	if err != nil {
		return nil, fmt.Errorf("query product admin list: %w", err)
	}
	defer rows.Close()
	items := []ProductAdminRecord{}
	for rows.Next() {
		var sku string
		if err := rows.Scan(&sku); err != nil {
			return nil, fmt.Errorf("scan product admin sku: %w", err)
		}
		item, err := s.productAdminBySKU(ctx, sku)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	sortProductAdminItems(items, query.Sort)
	return items, nil
}

func (s *Store) ProductAdminBySlug(ctx context.Context, slug string) (ProductAdminRecord, error) {
	var sku string
	if err := s.db.QueryRowContext(ctx, `select sku from products where slug = ?`, strings.TrimSpace(slug)).Scan(&sku); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("query product admin by slug: %w", err)
	}
	return s.productAdminBySKU(ctx, sku)
}

func (s *Store) CreateProduct(ctx context.Context, input CreateProductInput) (ProductAdminRecord, error) {
	prepared := prepareCreateProductInput(input)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProductAdminRecord{}, fmt.Errorf("begin create product transaction: %w", err)
	}
	defer tx.Rollback()

	now := timestampNow()
	if _, err := tx.ExecContext(ctx, `insert into products(sku, slug, title, category, price_cents, status, finish, summary, details, seo_title, seo_description, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, prepared.SKU, prepared.Slug, prepared.Title, prepared.Category, prepared.PriceCents, prepared.Status, prepared.Finish, prepared.Summary, prepared.Details, prepared.SEOTitle, prepared.SEODescription, now, now); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("insert product: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `insert into inventory_levels(id, product_sku, warehouse_id, on_hand, reserved, available, inbound, damaged, reorder_point, safety_stock, status, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, makeID("inv"), prepared.SKU, prepared.WarehouseID, prepared.Available, 0, prepared.Available, prepared.Inbound, 0, 12, 6, inventoryStatusForProduct(prepared.Status, prepared.Available), now); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("insert product inventory: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("commit create product transaction: %w", err)
	}
	return s.ProductAdminBySlug(ctx, prepared.Slug)
}

func (s *Store) UpdateProduct(ctx context.Context, currentSlug string, input UpdateProductInput) (ProductAdminRecord, error) {
	current, err := s.ProductAdminBySlug(ctx, currentSlug)
	if err != nil {
		return ProductAdminRecord{}, err
	}
	prepared := prepareUpdateProductInput(current, input)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProductAdminRecord{}, fmt.Errorf("begin update product transaction: %w", err)
	}
	defer tx.Rollback()

	now := timestampNow()
	if _, err := tx.ExecContext(ctx, `update products set sku = ?, slug = ?, title = ?, category = ?, price_cents = ?, status = ?, finish = ?, summary = ?, details = ?, seo_title = ?, seo_description = ?, updated_at = ? where slug = ?`, prepared.SKU, prepared.Slug, prepared.Title, prepared.Category, prepared.PriceCents, prepared.Status, prepared.Finish, prepared.Summary, prepared.Details, prepared.SEOTitle, prepared.SEODescription, now, strings.TrimSpace(currentSlug)); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("update product: %w", err)
	}
	if prepared.CurrentWarehouse != "" && prepared.CurrentWarehouse != prepared.WarehouseID {
		if _, err := tx.ExecContext(ctx, `delete from inventory_levels where product_sku = ? and warehouse_id = ?`, current.SKU, prepared.CurrentWarehouse); err != nil {
			return ProductAdminRecord{}, fmt.Errorf("move managed inventory row: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `insert into inventory_levels(id, product_sku, warehouse_id, on_hand, reserved, available, inbound, damaged, reorder_point, safety_stock, status, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) on conflict(product_sku, warehouse_id) do update set on_hand = excluded.on_hand, reserved = excluded.reserved, available = excluded.available, inbound = excluded.inbound, damaged = excluded.damaged, reorder_point = excluded.reorder_point, safety_stock = excluded.safety_stock, status = excluded.status, updated_at = excluded.updated_at`, makeID("inv"), prepared.SKU, prepared.WarehouseID, prepared.Available, 0, prepared.Available, prepared.Inbound, 0, 12, 6, inventoryStatusForProduct(prepared.Status, prepared.Available), now); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("upsert product inventory: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ProductAdminRecord{}, fmt.Errorf("commit update product transaction: %w", err)
	}
	return s.ProductAdminBySlug(ctx, prepared.Slug)
}

func (s *Store) DeleteProduct(ctx context.Context, slug string) error {
	result, err := s.db.ExecContext(ctx, `delete from products where slug = ?`, strings.TrimSpace(slug))
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted products: %w", err)
	}
	if deleted == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) productAdminBySKU(ctx context.Context, sku string) (ProductAdminRecord, error) {
	var item ProductAdminRecord
	err := s.db.QueryRowContext(ctx, `select sku, slug, title, category, price_cents, status, finish, summary, details, seo_title, seo_description, updated_at from products where sku = ?`, strings.TrimSpace(sku)).Scan(&item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.Status, &item.Finish, &item.Summary, &item.Details, &item.SEOTitle, &item.SEODescription, &item.UpdatedAt)
	if err != nil {
		return ProductAdminRecord{}, fmt.Errorf("query product admin by sku: %w", err)
	}
	warehouseID, warehouseName, available, inbound, hubCount, err := s.productInventorySummary(ctx, item.SKU)
	if err != nil {
		return ProductAdminRecord{}, err
	}
	item.WarehouseID = warehouseID
	item.WarehouseName = warehouseName
	item.Available = available
	item.Inbound = inbound
	item.Volume = available + inbound
	item.HubCount = hubCount
	return item, nil
}

func (s *Store) productInventorySummary(ctx context.Context, sku string) (string, string, int, int, int, error) {
	warehouseID := ""
	available := 0
	inbound := 0
	if err := s.db.QueryRowContext(ctx, `select warehouse_id, available, inbound from inventory_levels where product_sku = ? order by available desc, inbound desc, updated_at desc limit 1`, strings.TrimSpace(sku)).Scan(&warehouseID, &available, &inbound); err != nil && err != sql.ErrNoRows {
		return "", "", 0, 0, 0, fmt.Errorf("query product inventory summary: %w", err)
	}
	warehouseName := ""
	if warehouseID != "" {
		warehouse, err := s.warehouseByID(ctx, warehouseID)
		if err != nil {
			return "", "", 0, 0, 0, err
		}
		warehouseName = warehouse.Name
	}
	hubCount := 0
	if err := s.db.QueryRowContext(ctx, `select count(*) from inventory_levels where product_sku = ?`, strings.TrimSpace(sku)).Scan(&hubCount); err != nil {
		return "", "", 0, 0, 0, fmt.Errorf("count product inventory hubs: %w", err)
	}
	return warehouseID, warehouseName, available, inbound, hubCount, nil
}

func prepareCreateProductInput(input CreateProductInput) CreateProductInput {
	prepared := input
	prepared.SKU = strings.TrimSpace(prepared.SKU)
	prepared.Slug = strings.TrimSpace(prepared.Slug)
	prepared.Title = nonEmptyString(prepared.Title, prepared.SKU)
	prepared.Category = nonEmptyString(prepared.Category, "desks")
	prepared.Status = nonEmptyString(prepared.Status, "in_stock")
	prepared.Finish = nonEmptyString(prepared.Finish, "Graphite oak")
	prepared.Summary = nonEmptyString(prepared.Summary, "Atlas catalog item")
	prepared.Details = nonEmptyString(prepared.Details, prepared.Summary)
	prepared.SEOTitle = nonEmptyString(prepared.SEOTitle, "Atlas "+prepared.Title)
	prepared.SEODescription = nonEmptyString(prepared.SEODescription, prepared.Summary)
	prepared.WarehouseID = nonEmptyString(prepared.WarehouseID, "new-jersey-hub")
	if prepared.PriceCents < 0 {
		prepared.PriceCents = 0
	}
	if prepared.Available < 0 {
		prepared.Available = 0
	}
	if prepared.Inbound < 0 {
		prepared.Inbound = 0
	}
	return prepared
}

func prepareUpdateProductInput(current ProductAdminRecord, input UpdateProductInput) UpdateProductInput {
	prepared := input
	prepared.SKU = current.SKU
	prepared.Slug = nonEmptyString(prepared.Slug, current.Slug)
	prepared.Title = nonEmptyString(prepared.Title, current.Title)
	prepared.Category = nonEmptyString(prepared.Category, current.Category)
	prepared.Status = nonEmptyString(prepared.Status, current.Status)
	prepared.Finish = nonEmptyString(prepared.Finish, current.Finish)
	prepared.Summary = nonEmptyString(prepared.Summary, current.Summary)
	prepared.Details = nonEmptyString(prepared.Details, current.Details)
	prepared.SEOTitle = nonEmptyString(prepared.SEOTitle, current.SEOTitle)
	prepared.SEODescription = nonEmptyString(prepared.SEODescription, current.SEODescription)
	prepared.CurrentWarehouse = nonEmptyString(prepared.CurrentWarehouse, current.WarehouseID)
	prepared.WarehouseID = nonEmptyString(prepared.WarehouseID, current.WarehouseID)
	if prepared.PriceCents < 0 {
		prepared.PriceCents = current.PriceCents
	}
	if prepared.Available < 0 {
		prepared.Available = current.Available
	}
	if prepared.Inbound < 0 {
		prepared.Inbound = current.Inbound
	}
	return prepared
}

func sortProductAdminItems(items []ProductAdminRecord, sortKey string) {
	switch strings.TrimSpace(strings.ToLower(sortKey)) {
	case "price":
		sort.SliceStable(items, func(left, right int) bool {
			if items[left].PriceCents == items[right].PriceCents {
				return strings.ToLower(items[left].Title) < strings.ToLower(items[right].Title)
			}
			return items[left].PriceCents > items[right].PriceCents
		})
	case "volume":
		sort.SliceStable(items, func(left, right int) bool {
			if items[left].Volume == items[right].Volume {
				return strings.ToLower(items[left].Title) < strings.ToLower(items[right].Title)
			}
			return items[left].Volume > items[right].Volume
		})
	case "status":
		sort.SliceStable(items, func(left, right int) bool {
			leftStatus := strings.ToLower(items[left].Status)
			rightStatus := strings.ToLower(items[right].Status)
			if leftStatus == rightStatus {
				return strings.ToLower(items[left].Title) < strings.ToLower(items[right].Title)
			}
			return leftStatus < rightStatus
		})
	default:
		sort.SliceStable(items, func(left, right int) bool {
			if items[left].UpdatedAt == items[right].UpdatedAt {
				return strings.ToLower(items[left].Title) < strings.ToLower(items[right].Title)
			}
			return items[left].UpdatedAt > items[right].UpdatedAt
		})
	}
}

func inventoryStatusForProduct(productStatus string, available int) string {
	if available <= 0 {
		return "promise_risk"
	}
	if strings.EqualFold(strings.TrimSpace(productStatus), "low_stock") || available <= 3 {
		return "promise_risk"
	}
	return "balanced"
}
