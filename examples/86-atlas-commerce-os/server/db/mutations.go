package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

type CommentRecord struct {
	ID               string `json:"id"`
	ProductSKU       string `json:"productSku"`
	AuthorName       string `json:"authorName"`
	AuthorType       string `json:"authorType"`
	Reaction         string `json:"reaction"`
	Subject          string `json:"subject"`
	Body             string `json:"body"`
	Status           string `json:"status"`
	ModerationReason string `json:"moderationReason"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type QuoteRequestRecord struct {
	ID            string `json:"id"`
	ProductSKU    string `json:"productSku"`
	RequesterName string `json:"requesterName"`
	CompanyName   string `json:"companyName"`
	Email         string `json:"email"`
	Quantity      int    `json:"quantity"`
	Note          string `json:"note"`
	CreatedAt     string `json:"createdAt"`
}

type RestockRequestRecord struct {
	ID                   string `json:"id"`
	ProductSKU           string `json:"productSku"`
	Email                string `json:"email"`
	PreferredWarehouseID string `json:"preferredWarehouseId"`
	CreatedAt            string `json:"createdAt"`
}

type TransferRecord struct {
	ID                   string `json:"id"`
	SourceWarehouseID    string `json:"sourceWarehouseId"`
	DestinationWarehouse string `json:"destinationWarehouseId"`
	Status               string `json:"status"`
	Reason               string `json:"reason"`
	RecommendedBy        string `json:"recommendedBy"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type ReceivingSessionRecord struct {
	ID                 string `json:"id"`
	SourceType         string `json:"sourceType"`
	SourceID           string `json:"sourceId"`
	WarehouseID        string `json:"warehouseId"`
	Status             string `json:"status"`
	DiscrepancySummary string `json:"discrepancySummary"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type CreateCommentInput struct {
	ProductSKU string
	AuthorName string
	AuthorType string
	Reaction   string
	Subject    string
	Body       string
}

type CreateQuoteRequestInput struct {
	ProductSKU    string
	RequesterName string
	CompanyName   string
	Email         string
	Quantity      int
	Note          string
}

type CreateRestockRequestInput struct {
	ProductSKU           string
	Email                string
	PreferredWarehouseID string
}

type SaveViewInput struct {
	OwnerID       string
	Name          string
	Scope         string
	FiltersJSON   string
	SortKey       string
	SortDirection string
	Density       string
	WarehouseID   string
}

type CreateTransferInput struct {
	SourceWarehouseID      string
	DestinationWarehouseID string
	Reason                 string
	RecommendedBy          string
}

type ReconcileReceivingInput struct {
	Status             string
	DiscrepancySummary string
}

type UpdatePurchaseOrderStatusInput struct {
	Status string
}

type UpdateInventoryLevelInput struct {
	WarehouseID  string
	OnHand       int
	Reserved     int
	Inbound      int
	Damaged      int
	ReorderPoint int
	SafetyStock  int
	Status       string
}

type CreatePurchaseOrderInput struct {
	VendorName   string
	WarehouseID  string
	ProductSKU   string
	Quantity     int
	ETA          string
	PriorityNote string
	Status       string
}

func (s *Store) Comments(ctx context.Context, status string) ([]CommentRecord, error) {
	query := `select id, product_sku, author_name, author_type, reaction, subject, body, status, coalesce(moderation_reason, ''), created_at, updated_at from comments`
	args := []any{}
	if trimmed := strings.TrimSpace(status); trimmed != "" {
		query += ` where status = ?`
		args = append(args, trimmed)
	}
	query += ` order by created_at desc`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()
	items := []CommentRecord{}
	for rows.Next() {
		var item CommentRecord
		if err := rows.Scan(&item.ID, &item.ProductSKU, &item.AuthorName, &item.AuthorType, &item.Reaction, &item.Subject, &item.Body, &item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) CreateComment(ctx context.Context, input CreateCommentInput) (CommentRecord, error) {
	now := timestampNow()
	record := CommentRecord{
		ID:         makeID("cmt"),
		ProductSKU: strings.TrimSpace(input.ProductSKU),
		AuthorName: nonEmptyString(input.AuthorName, "Atlas visitor"),
		AuthorType: nonEmptyString(input.AuthorType, "public"),
		Reaction:   normalizeCommentReaction(input.Reaction),
		Subject:    nonEmptyString(input.Subject, "Product question"),
		Body:       strings.TrimSpace(input.Body),
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if record.Body == "" {
		return CommentRecord{}, fmt.Errorf("comment body is required")
	}
	if _, err := s.db.ExecContext(ctx, `insert into comments(id, product_sku, author_name, author_type, reaction, subject, body, status, moderation_reason, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?)`, record.ID, record.ProductSKU, record.AuthorName, record.AuthorType, record.Reaction, record.Subject, record.Body, record.Status, record.CreatedAt, record.UpdatedAt); err != nil {
		return CommentRecord{}, fmt.Errorf("insert comment: %w", err)
	}
	return record, nil
}

func (s *Store) ModerateComment(ctx context.Context, id string, status string, reason string) (CommentRecord, error) {
	trimmedStatus := strings.TrimSpace(status)
	if trimmedStatus == "" {
		trimmedStatus = "approved"
	}
	if _, err := s.db.ExecContext(ctx, `update comments set status = ?, moderation_reason = ?, updated_at = ? where id = ?`, trimmedStatus, strings.TrimSpace(reason), timestampNow(), strings.TrimSpace(id)); err != nil {
		return CommentRecord{}, fmt.Errorf("update comment moderation: %w", err)
	}
	return s.commentByID(ctx, id)
}

func (s *Store) CreateQuoteRequest(ctx context.Context, input CreateQuoteRequestInput) (QuoteRequestRecord, error) {
	now := timestampNow()
	record := QuoteRequestRecord{
		ID:            makeID("quote"),
		ProductSKU:    strings.TrimSpace(input.ProductSKU),
		RequesterName: nonEmptyString(input.RequesterName, "Atlas visitor"),
		CompanyName:   strings.TrimSpace(input.CompanyName),
		Email:         strings.TrimSpace(input.Email),
		Quantity:      input.Quantity,
		Note:          strings.TrimSpace(input.Note),
		CreatedAt:     now,
	}
	if record.Quantity < 1 {
		record.Quantity = 1
	}
	if record.Email == "" {
		return QuoteRequestRecord{}, fmt.Errorf("email is required")
	}
	if _, err := s.db.ExecContext(ctx, `insert into quote_requests(id, product_sku, requester_name, company_name, email, quantity, note, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.ProductSKU, record.RequesterName, record.CompanyName, record.Email, record.Quantity, record.Note, record.CreatedAt); err != nil {
		return QuoteRequestRecord{}, fmt.Errorf("insert quote request: %w", err)
	}
	return record, nil
}

func (s *Store) CreateRestockRequest(ctx context.Context, input CreateRestockRequestInput) (RestockRequestRecord, error) {
	record := RestockRequestRecord{
		ID:                   makeID("restock"),
		ProductSKU:           strings.TrimSpace(input.ProductSKU),
		Email:                strings.TrimSpace(input.Email),
		PreferredWarehouseID: strings.TrimSpace(input.PreferredWarehouseID),
		CreatedAt:            timestampNow(),
	}
	if record.Email == "" {
		return RestockRequestRecord{}, fmt.Errorf("email is required")
	}
	if _, err := s.db.ExecContext(ctx, `insert into restock_requests(id, product_sku, email, preferred_warehouse_id, created_at) values (?, ?, ?, ?, ?)`, record.ID, record.ProductSKU, record.Email, nullableString(record.PreferredWarehouseID), record.CreatedAt); err != nil {
		return RestockRequestRecord{}, fmt.Errorf("insert restock request: %w", err)
	}
	return record, nil
}

func (s *Store) SavePreferences(ctx context.Context, record PreferencesRecord) (PreferencesRecord, error) {
	record.OwnerID = nonEmptyString(record.OwnerID, "demo-operator")
	record.Theme = nonEmptyString(record.Theme, "dark")
	record.Locale = nonEmptyString(record.Locale, "en")
	record.Density = nonEmptyString(record.Density, "compact")
	record.DefaultWarehouseID = nonEmptyString(record.DefaultWarehouseID, "new-jersey-hub")
	if _, err := s.db.ExecContext(ctx, `insert into preferences(id, owner_id, theme, locale, density, default_warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?) on conflict(owner_id) do update set theme = excluded.theme, locale = excluded.locale, density = excluded.density, default_warehouse_id = excluded.default_warehouse_id, updated_at = excluded.updated_at`, makeID("pref"), record.OwnerID, record.Theme, record.Locale, record.Density, record.DefaultWarehouseID, timestampNow(), timestampNow()); err != nil {
		return PreferencesRecord{}, fmt.Errorf("upsert preferences: %w", err)
	}
	return s.PreferencesByOwner(ctx, record.OwnerID)
}

func (s *Store) SaveView(ctx context.Context, input SaveViewInput) (repository.SavedView, error) {
	record := repository.SavedView{
		ID:            makeID("view"),
		Name:          nonEmptyString(input.Name, "Untitled view"),
		Scope:         nonEmptyString(input.Scope, "inventory"),
		SortKey:       nonEmptyString(input.SortKey, "available"),
		SortDirection: nonEmptyString(input.SortDirection, "asc"),
		Density:       nonEmptyString(input.Density, "compact"),
		WarehouseID:   strings.TrimSpace(input.WarehouseID),
		FiltersJSON:   nonEmptyString(input.FiltersJSON, `{}`),
	}
	ownerID := nonEmptyString(input.OwnerID, "demo-operator")
	if _, err := s.db.ExecContext(ctx, `insert into saved_views(id, name, owner_id, scope, filters_json, sort_key, sort_direction, density, warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Name, ownerID, record.Scope, record.FiltersJSON, record.SortKey, record.SortDirection, record.Density, nullableString(record.WarehouseID), timestampNow(), timestampNow()); err != nil {
		return repository.SavedView{}, fmt.Errorf("insert saved view: %w", err)
	}
	return record, nil
}

func (s *Store) Transfers(ctx context.Context) ([]TransferRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at from transfers order by created_at desc`)
	if err != nil {
		return nil, fmt.Errorf("query transfers: %w", err)
	}
	defer rows.Close()
	items := []TransferRecord{}
	for rows.Next() {
		var item TransferRecord
		if err := rows.Scan(&item.ID, &item.SourceWarehouseID, &item.DestinationWarehouse, &item.Status, &item.Reason, &item.RecommendedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan transfer: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) CreateTransfer(ctx context.Context, input CreateTransferInput) (TransferRecord, error) {
	now := timestampNow()
	record := TransferRecord{
		ID:                   makeID("tr"),
		SourceWarehouseID:    nonEmptyString(input.SourceWarehouseID, "nevada-hub"),
		DestinationWarehouse: nonEmptyString(input.DestinationWarehouseID, "new-jersey-hub"),
		Status:               "draft",
		Reason:               nonEmptyString(input.Reason, "Balance demand between warehouses."),
		RecommendedBy:        nonEmptyString(input.RecommendedBy, "Atlas server"),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if _, err := s.db.ExecContext(ctx, `insert into transfers(id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.SourceWarehouseID, record.DestinationWarehouse, record.Status, record.Reason, record.RecommendedBy, record.CreatedAt, record.UpdatedAt); err != nil {
		return TransferRecord{}, fmt.Errorf("insert transfer: %w", err)
	}
	return record, nil
}

func (s *Store) ReceivingSessions(ctx context.Context) ([]ReceivingSessionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select id, source_type, source_id, warehouse_id, status, coalesce(discrepancy_summary, ''), created_at, updated_at from receiving_sessions order by created_at desc`)
	if err != nil {
		return nil, fmt.Errorf("query receiving sessions: %w", err)
	}
	defer rows.Close()
	items := []ReceivingSessionRecord{}
	for rows.Next() {
		var item ReceivingSessionRecord
		if err := rows.Scan(&item.ID, &item.SourceType, &item.SourceID, &item.WarehouseID, &item.Status, &item.DiscrepancySummary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan receiving session: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) ReconcileReceiving(ctx context.Context, id string, input ReconcileReceivingInput) (ReceivingSessionRecord, error) {
	status := nonEmptyString(input.Status, "closed")
	if _, err := s.db.ExecContext(ctx, `update receiving_sessions set status = ?, discrepancy_summary = ?, updated_at = ? where id = ?`, status, strings.TrimSpace(input.DiscrepancySummary), timestampNow(), strings.TrimSpace(id)); err != nil {
		return ReceivingSessionRecord{}, fmt.Errorf("update receiving session: %w", err)
	}
	return s.receivingByID(ctx, id)
}

func (s *Store) UpdatePurchaseOrderStatus(ctx context.Context, id string, input UpdatePurchaseOrderStatusInput) (PurchaseOrderRecord, error) {
	status := nonEmptyString(input.Status, "submitted")
	if _, err := s.db.ExecContext(ctx, `update purchase_orders set status = ?, updated_at = ? where id = ?`, status, timestampNow(), strings.TrimSpace(id)); err != nil {
		return PurchaseOrderRecord{}, fmt.Errorf("update purchase order status: %w", err)
	}
	return s.PurchaseOrderByID(ctx, id)
}

func (s *Store) UpdateInventoryLevel(ctx context.Context, sku string, input UpdateInventoryLevelInput) (repository.InventoryRow, error) {
	prepared := input
	prepared.WarehouseID = strings.TrimSpace(prepared.WarehouseID)
	prepared.Status = nonEmptyString(prepared.Status, "balanced")
	if prepared.WarehouseID == "" {
		return repository.InventoryRow{}, fmt.Errorf("warehouse id is required")
	}
	if prepared.OnHand < 0 {
		prepared.OnHand = 0
	}
	if prepared.Reserved < 0 {
		prepared.Reserved = 0
	}
	if prepared.Inbound < 0 {
		prepared.Inbound = 0
	}
	if prepared.Damaged < 0 {
		prepared.Damaged = 0
	}
	if prepared.ReorderPoint < 1 {
		prepared.ReorderPoint = 1
	}
	if prepared.SafetyStock < 0 {
		prepared.SafetyStock = 0
	}
	available := prepared.OnHand - prepared.Reserved - prepared.Damaged
	if available < 0 {
		available = 0
	}
	if _, err := s.db.ExecContext(ctx, `update inventory_levels set on_hand = ?, reserved = ?, available = ?, inbound = ?, damaged = ?, reorder_point = ?, safety_stock = ?, status = ?, updated_at = ? where product_sku = ? and warehouse_id = ?`, prepared.OnHand, prepared.Reserved, available, prepared.Inbound, prepared.Damaged, prepared.ReorderPoint, prepared.SafetyStock, prepared.Status, timestampNow(), strings.TrimSpace(sku), prepared.WarehouseID); err != nil {
		return repository.InventoryRow{}, fmt.Errorf("update inventory level: %w", err)
	}
	return s.inventoryLevelBySKUWarehouse(ctx, sku, prepared.WarehouseID)
}

func (s *Store) CreatePurchaseOrder(ctx context.Context, input CreatePurchaseOrderInput) (PurchaseOrderDetailRecord, error) {
	now := timestampNow()
	record := PurchaseOrderRecord{
		ID:           makeID("po"),
		VendorName:   nonEmptyString(input.VendorName, "Atlas Vendor Network"),
		WarehouseID:  nonEmptyString(input.WarehouseID, "new-jersey-hub"),
		Status:       nonEmptyString(input.Status, "submitted"),
		PriorityNote: nonEmptyString(input.PriorityNote, "Warehouse replenishment"),
		ETA:          nonEmptyString(input.ETA, "Next available lane"),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	line := PurchaseOrderLineRecord{
		ID:              makeID("po-line"),
		PurchaseOrderID: record.ID,
		ProductSKU:      strings.TrimSpace(input.ProductSKU),
		Quantity:        input.Quantity,
		ETA:             record.ETA,
		Status:          "submitted",
	}
	if line.ProductSKU == "" {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("product sku is required")
	}
	if line.Quantity < 1 {
		line.Quantity = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("begin purchase order transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `insert into purchase_orders(id, vendor_name, warehouse_id, status, priority_note, eta, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.VendorName, record.WarehouseID, record.Status, record.PriorityNote, record.ETA, record.CreatedAt, record.UpdatedAt); err != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("insert purchase order: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `insert into purchase_order_lines(id, purchase_order_id, product_sku, quantity, eta, status) values (?, ?, ?, ?, ?, ?)`, line.ID, line.PurchaseOrderID, line.ProductSKU, line.Quantity, line.ETA, line.Status); err != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("insert purchase order line: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `update inventory_levels set inbound = inbound + ?, updated_at = ? where product_sku = ? and warehouse_id = ?`, line.Quantity, now, line.ProductSKU, record.WarehouseID); err != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("increase inbound inventory: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("commit purchase order transaction: %w", err)
	}
	return s.PurchaseOrderDetail(ctx, record.ID)
}

func (s *Store) inventoryLevelBySKUWarehouse(ctx context.Context, sku string, warehouseID string) (repository.InventoryRow, error) {
	var item repository.InventoryRow
	err := s.db.QueryRowContext(ctx, `select il.id, p.sku, p.title, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where p.sku = ? and il.warehouse_id = ?`, strings.TrimSpace(sku), strings.TrimSpace(warehouseID)).Scan(&item.ID, &item.SKU, &item.Title, &item.WarehouseID, &item.WarehouseName, &item.OnHand, &item.Reserved, &item.Available, &item.CoverDays, &item.Inbound, &item.Damaged, &item.ReorderPoint, &item.SafetyStock, &item.Status, &item.UpdatedAt)
	if err != nil {
		return repository.InventoryRow{}, fmt.Errorf("query inventory level by sku and warehouse: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateThreshold(ctx context.Context, sku string, warehouseID string, reorderPoint int, safetyStock int) (repository.InventoryRow, error) {
	if reorderPoint < 1 {
		reorderPoint = 1
	}
	if safetyStock < 0 {
		safetyStock = 0
	}
	if _, err := s.db.ExecContext(ctx, `update inventory_levels set reorder_point = ?, safety_stock = ?, updated_at = ? where product_sku = ? and warehouse_id = ?`, reorderPoint, safetyStock, timestampNow(), strings.TrimSpace(sku), strings.TrimSpace(warehouseID)); err != nil {
		return repository.InventoryRow{}, fmt.Errorf("update threshold: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `insert into inventory_threshold_events(id, product_sku, warehouse_id, reorder_point, safety_stock, actor_name, summary, detail, created_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?)`, makeID("threshold"), strings.TrimSpace(sku), strings.TrimSpace(warehouseID), reorderPoint, safetyStock, "Atlas server", "Threshold update staged", fmt.Sprintf("Thresholds updated for %s in %s.", strings.TrimSpace(sku), strings.TrimSpace(warehouseID)), timestampNow()); err != nil {
		return repository.InventoryRow{}, fmt.Errorf("insert threshold history event: %w", err)
	}
	return s.InventoryBySKU(ctx, sku)
}

func (s *Store) commentByID(ctx context.Context, id string) (CommentRecord, error) {
	var item CommentRecord
	err := s.db.QueryRowContext(ctx, `select id, product_sku, author_name, author_type, reaction, subject, body, status, coalesce(moderation_reason, ''), created_at, updated_at from comments where id = ?`, strings.TrimSpace(id)).Scan(&item.ID, &item.ProductSKU, &item.AuthorName, &item.AuthorType, &item.Reaction, &item.Subject, &item.Body, &item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return CommentRecord{}, fmt.Errorf("query comment by id: %w", err)
	}
	return item, nil
}

func normalizeCommentReaction(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "down", "thumbs_down", "thumbs-down":
		return "down"
	default:
		return "up"
	}
}

func (s *Store) receivingByID(ctx context.Context, id string) (ReceivingSessionRecord, error) {
	var item ReceivingSessionRecord
	err := s.db.QueryRowContext(ctx, `select id, source_type, source_id, warehouse_id, status, coalesce(discrepancy_summary, ''), created_at, updated_at from receiving_sessions where id = ?`, strings.TrimSpace(id)).Scan(&item.ID, &item.SourceType, &item.SourceID, &item.WarehouseID, &item.Status, &item.DiscrepancySummary, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return ReceivingSessionRecord{}, fmt.Errorf("query receiving by id: %w", err)
	}
	return item, nil
}

func makeID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
}

func timestampNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return value
}

func nonEmptyString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
