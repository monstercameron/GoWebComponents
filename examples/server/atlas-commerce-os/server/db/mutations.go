package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/repository"
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

func (parseS *Store) Comments(parseCtx context.Context, parseStatus string) ([]CommentRecord, error) {
	parseQuery := `select id, product_sku, author_name, author_type, reaction, subject, body, status, coalesce(moderation_reason, ''), created_at, updated_at from comments`
	parseArgs := []any{}
	if parseTrimmed := strings.TrimSpace(parseStatus); parseTrimmed != "" {
		parseQuery += ` where status = ?`
		parseArgs = append(parseArgs, parseTrimmed)
	}
	parseQuery += ` order by created_at desc`
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, parseQuery, parseArgs...)
	if parseErr != nil {
		return nil, fmt.Errorf("query comments: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []CommentRecord{}
	for parseRows.Next() {
		var parseItem CommentRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.ProductSKU, &parseItem.AuthorName, &parseItem.AuthorType, &parseItem.Reaction, &parseItem.Subject, &parseItem.Body, &parseItem.Status, &parseItem.ModerationReason, &parseItem.CreatedAt, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan comment: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) CreateComment(parseCtx context.Context, parseInput CreateCommentInput) (CommentRecord, error) {
	parseNow := timestampNow()
	parseRecord := CommentRecord{
		ID:         makeID("cmt"),
		ProductSKU: strings.TrimSpace(parseInput.ProductSKU),
		AuthorName: nonEmptyString(parseInput.AuthorName, "Atlas visitor"),
		AuthorType: nonEmptyString(parseInput.AuthorType, "public"),
		Reaction:   normalizeCommentReaction(parseInput.Reaction),
		Subject:    nonEmptyString(parseInput.Subject, "Product question"),
		Body:       strings.TrimSpace(parseInput.Body),
		Status:     "pending",
		CreatedAt:  parseNow,
		UpdatedAt:  parseNow,
	}
	if parseRecord.Body == "" {
		return CommentRecord{}, fmt.Errorf("comment body is required")
	}
	if _, parseErr := parseS.db.ExecContext(parseCtx, `insert into comments(id, product_sku, author_name, author_type, reaction, subject, body, status, moderation_reason, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?)`, parseRecord.ID, parseRecord.ProductSKU, parseRecord.AuthorName, parseRecord.AuthorType, parseRecord.Reaction, parseRecord.Subject, parseRecord.Body, parseRecord.Status, parseRecord.CreatedAt, parseRecord.UpdatedAt); parseErr != nil {
		return CommentRecord{}, fmt.Errorf("insert comment: %w", parseErr)
	}
	return parseRecord, nil
}

func (parseS *Store) ModerateComment(parseCtx context.Context, parseId string, parseStatus string, parseReason string) (CommentRecord, error) {
	parseTrimmedStatus := strings.TrimSpace(parseStatus)
	if parseTrimmedStatus == "" {
		parseTrimmedStatus = "approved"
	}
	if _, parseErr := parseS.db.ExecContext(parseCtx, `update comments set status = ?, moderation_reason = ?, updated_at = ? where id = ?`, parseTrimmedStatus, strings.TrimSpace(parseReason), timestampNow(), strings.TrimSpace(parseId)); parseErr != nil {
		return CommentRecord{}, fmt.Errorf("update comment moderation: %w", parseErr)
	}
	return parseS.commentByID(parseCtx, parseId)
}

func (parseS *Store) ModerateComments(parseCtx context.Context, parseIds []string, parseStatus string, parseReason string) ([]CommentRecord, error) {
	parseTrimmedStatus := strings.TrimSpace(parseStatus)
	if parseTrimmedStatus == "" {
		parseTrimmedStatus = "approved"
	}
	parseTx, parseErr := parseS.db.BeginTx(parseCtx, nil)
	if parseErr != nil {
		return nil, fmt.Errorf("begin bulk moderation transaction: %w", parseErr)
	}
	parseNormalized := make([]string, 0, len(parseIds))
	for _, parseId := range parseIds {
		parseTrimmedID := strings.TrimSpace(parseId)
		if parseTrimmedID == "" {
			continue
		}
		parseNormalized = append(parseNormalized, parseTrimmedID)
		if _, parseErr2 := parseTx.ExecContext(parseCtx, `update comments set status = ?, moderation_reason = ?, updated_at = ? where id = ?`, parseTrimmedStatus, strings.TrimSpace(parseReason), timestampNow(), parseTrimmedID); parseErr2 != nil {
			_ = parseTx.Rollback()
			return nil, fmt.Errorf("update bulk comment moderation: %w", parseErr2)
		}
	}
	if parseErr3 := parseTx.Commit(); parseErr3 != nil {
		return nil, fmt.Errorf("commit bulk moderation transaction: %w", parseErr3)
	}
	parseItems := make([]CommentRecord, 0, len(parseNormalized))
	for _, parseId2 := range parseNormalized {
		parseItem, parseErr4 := parseS.commentByID(parseCtx, parseId2)
		if parseErr4 != nil {
			return nil, parseErr4
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) CreateQuoteRequest(parseCtx context.Context, parseInput CreateQuoteRequestInput) (QuoteRequestRecord, error) {
	parseNow := timestampNow()
	parseRecord := QuoteRequestRecord{
		ID:            makeID("quote"),
		ProductSKU:    strings.TrimSpace(parseInput.ProductSKU),
		RequesterName: nonEmptyString(parseInput.RequesterName, "Atlas visitor"),
		CompanyName:   strings.TrimSpace(parseInput.CompanyName),
		Email:         strings.TrimSpace(parseInput.Email),
		Quantity:      parseInput.Quantity,
		Note:          strings.TrimSpace(parseInput.Note),
		CreatedAt:     parseNow,
	}
	if parseRecord.Quantity < 1 {
		parseRecord.Quantity = 1
	}
	if parseRecord.Email == "" {
		return QuoteRequestRecord{}, fmt.Errorf("email is required")
	}
	if _, parseErr := parseS.db.ExecContext(parseCtx, `insert into quote_requests(id, product_sku, requester_name, company_name, email, quantity, note, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)`, parseRecord.ID, parseRecord.ProductSKU, parseRecord.RequesterName, parseRecord.CompanyName, parseRecord.Email, parseRecord.Quantity, parseRecord.Note, parseRecord.CreatedAt); parseErr != nil {
		return QuoteRequestRecord{}, fmt.Errorf("insert quote request: %w", parseErr)
	}
	return parseRecord, nil
}

func (parseS *Store) CreateRestockRequest(parseCtx context.Context, parseInput CreateRestockRequestInput) (RestockRequestRecord, error) {
	parseRecord := RestockRequestRecord{
		ID:                   makeID("restock"),
		ProductSKU:           strings.TrimSpace(parseInput.ProductSKU),
		Email:                strings.TrimSpace(parseInput.Email),
		PreferredWarehouseID: strings.TrimSpace(parseInput.PreferredWarehouseID),
		CreatedAt:            timestampNow(),
	}
	if parseRecord.Email == "" {
		return RestockRequestRecord{}, fmt.Errorf("email is required")
	}
	if _, parseErr := parseS.db.ExecContext(parseCtx, `insert into restock_requests(id, product_sku, email, preferred_warehouse_id, created_at) values (?, ?, ?, ?, ?)`, parseRecord.ID, parseRecord.ProductSKU, parseRecord.Email, nullableString(parseRecord.PreferredWarehouseID), parseRecord.CreatedAt); parseErr != nil {
		return RestockRequestRecord{}, fmt.Errorf("insert restock request: %w", parseErr)
	}
	return parseRecord, nil
}

func (parseS *Store) SavePreferences(parseCtx context.Context, parseRecord PreferencesRecord) (PreferencesRecord, error) {
	parseRecord.OwnerID = nonEmptyString(parseRecord.OwnerID, "demo-operator")
	parseRecord.Theme = nonEmptyString(parseRecord.Theme, "dark")
	parseRecord.Locale = nonEmptyString(parseRecord.Locale, "en")
	parseRecord.Density = nonEmptyString(parseRecord.Density, "compact")
	parseRecord.DefaultWarehouseID = nonEmptyString(parseRecord.DefaultWarehouseID, "new-jersey-hub")
	if _, parseErr := parseS.db.ExecContext(parseCtx, `insert into preferences(id, owner_id, theme, locale, density, default_warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?) on conflict(owner_id) do update set theme = excluded.theme, locale = excluded.locale, density = excluded.density, default_warehouse_id = excluded.default_warehouse_id, updated_at = excluded.updated_at`, makeID("pref"), parseRecord.OwnerID, parseRecord.Theme, parseRecord.Locale, parseRecord.Density, parseRecord.DefaultWarehouseID, timestampNow(), timestampNow()); parseErr != nil {
		return PreferencesRecord{}, fmt.Errorf("upsert preferences: %w", parseErr)
	}
	return parseS.PreferencesByOwner(parseCtx, parseRecord.OwnerID)
}

func (parseS *Store) SaveView(parseCtx context.Context, parseInput SaveViewInput) (repository.SavedView, error) {
	parseRecord := repository.SavedView{
		ID:            makeID("view"),
		Name:          nonEmptyString(parseInput.Name, "Untitled view"),
		Scope:         nonEmptyString(parseInput.Scope, "inventory"),
		SortKey:       nonEmptyString(parseInput.SortKey, "available"),
		SortDirection: nonEmptyString(parseInput.SortDirection, "asc"),
		Density:       nonEmptyString(parseInput.Density, "compact"),
		WarehouseID:   strings.TrimSpace(parseInput.WarehouseID),
		FiltersJSON:   nonEmptyString(parseInput.FiltersJSON, `{}`),
	}
	parseOwnerID := nonEmptyString(parseInput.OwnerID, "demo-operator")
	if _, parseErr := parseS.db.ExecContext(parseCtx, `insert into saved_views(id, name, owner_id, scope, filters_json, sort_key, sort_direction, density, warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, parseRecord.ID, parseRecord.Name, parseOwnerID, parseRecord.Scope, parseRecord.FiltersJSON, parseRecord.SortKey, parseRecord.SortDirection, parseRecord.Density, nullableString(parseRecord.WarehouseID), timestampNow(), timestampNow()); parseErr != nil {
		return repository.SavedView{}, fmt.Errorf("insert saved view: %w", parseErr)
	}
	return parseRecord, nil
}

func (parseS *Store) ImportSavedViews(parseCtx context.Context, parseOwnerID string, parseViews []repository.SavedView) ([]repository.SavedView, error) {
	parseImported := make([]repository.SavedView, 0, len(parseViews))
	parseNormalizedOwnerID := nonEmptyString(parseOwnerID, "demo-operator")
	for _, parseView := range parseViews {
		parseCreated, parseErr := parseS.SaveView(parseCtx, SaveViewInput{
			OwnerID:       parseNormalizedOwnerID,
			Name:          parseView.Name,
			Scope:         parseView.Scope,
			FiltersJSON:   parseView.FiltersJSON,
			SortKey:       parseView.SortKey,
			SortDirection: parseView.SortDirection,
			Density:       parseView.Density,
			WarehouseID:   parseView.WarehouseID,
		})
		if parseErr != nil {
			return nil, fmt.Errorf("import saved view %q: %w", parseView.Name, parseErr)
		}
		parseImported = append(parseImported, parseCreated)
	}
	return parseImported, nil
}

func (parseS *Store) Transfers(parseCtx context.Context) ([]TransferRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at from transfers order by created_at desc`)
	if parseErr != nil {
		return nil, fmt.Errorf("query transfers: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []TransferRecord{}
	for parseRows.Next() {
		var parseItem TransferRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.SourceWarehouseID, &parseItem.DestinationWarehouse, &parseItem.Status, &parseItem.Reason, &parseItem.RecommendedBy, &parseItem.CreatedAt, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan transfer: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) CreateTransfer(parseCtx context.Context, parseInput CreateTransferInput) (TransferRecord, error) {
	parseNow := timestampNow()
	parseRecord := TransferRecord{
		ID:                   makeID("tr"),
		SourceWarehouseID:    nonEmptyString(parseInput.SourceWarehouseID, "nevada-hub"),
		DestinationWarehouse: nonEmptyString(parseInput.DestinationWarehouseID, "new-jersey-hub"),
		Status:               "draft",
		Reason:               nonEmptyString(parseInput.Reason, "Balance demand between warehouses."),
		RecommendedBy:        nonEmptyString(parseInput.RecommendedBy, "Atlas server"),
		CreatedAt:            parseNow,
		UpdatedAt:            parseNow,
	}
	if _, parseErr := parseS.db.ExecContext(parseCtx, `insert into transfers(id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?)`, parseRecord.ID, parseRecord.SourceWarehouseID, parseRecord.DestinationWarehouse, parseRecord.Status, parseRecord.Reason, parseRecord.RecommendedBy, parseRecord.CreatedAt, parseRecord.UpdatedAt); parseErr != nil {
		return TransferRecord{}, fmt.Errorf("insert transfer: %w", parseErr)
	}
	return parseRecord, nil
}

func (parseS *Store) ReceivingSessions(parseCtx context.Context) ([]ReceivingSessionRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, source_type, source_id, warehouse_id, status, coalesce(discrepancy_summary, ''), created_at, updated_at from receiving_sessions order by created_at desc`)
	if parseErr != nil {
		return nil, fmt.Errorf("query receiving sessions: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []ReceivingSessionRecord{}
	for parseRows.Next() {
		var parseItem ReceivingSessionRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.SourceType, &parseItem.SourceID, &parseItem.WarehouseID, &parseItem.Status, &parseItem.DiscrepancySummary, &parseItem.CreatedAt, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan receiving session: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) ReconcileReceiving(parseCtx context.Context, parseId string, parseInput ReconcileReceivingInput) (ReceivingSessionRecord, error) {
	parseStatus := nonEmptyString(parseInput.Status, "closed")
	if _, parseErr := parseS.db.ExecContext(parseCtx, `update receiving_sessions set status = ?, discrepancy_summary = ?, updated_at = ? where id = ?`, parseStatus, strings.TrimSpace(parseInput.DiscrepancySummary), timestampNow(), strings.TrimSpace(parseId)); parseErr != nil {
		return ReceivingSessionRecord{}, fmt.Errorf("update receiving session: %w", parseErr)
	}
	return parseS.receivingByID(parseCtx, parseId)
}

func (parseS *Store) UpdatePurchaseOrderStatus(parseCtx context.Context, parseId string, parseInput UpdatePurchaseOrderStatusInput) (PurchaseOrderRecord, error) {
	parseStatus := nonEmptyString(parseInput.Status, "submitted")
	if _, parseErr := parseS.db.ExecContext(parseCtx, `update purchase_orders set status = ?, updated_at = ? where id = ?`, parseStatus, timestampNow(), strings.TrimSpace(parseId)); parseErr != nil {
		return PurchaseOrderRecord{}, fmt.Errorf("update purchase order status: %w", parseErr)
	}
	return parseS.PurchaseOrderByID(parseCtx, parseId)
}

func (parseS *Store) UpdateInventoryLevel(parseCtx context.Context, parseSku string, parseInput UpdateInventoryLevelInput) (repository.InventoryRow, error) {
	parsePrepared := parseInput
	parsePrepared.WarehouseID = strings.TrimSpace(parsePrepared.WarehouseID)
	parsePrepared.Status = nonEmptyString(parsePrepared.Status, "balanced")
	if parsePrepared.WarehouseID == "" {
		return repository.InventoryRow{}, fmt.Errorf("warehouse id is required")
	}
	if parsePrepared.OnHand < 0 {
		parsePrepared.OnHand = 0
	}
	if parsePrepared.Reserved < 0 {
		parsePrepared.Reserved = 0
	}
	if parsePrepared.Inbound < 0 {
		parsePrepared.Inbound = 0
	}
	if parsePrepared.Damaged < 0 {
		parsePrepared.Damaged = 0
	}
	if parsePrepared.ReorderPoint < 1 {
		parsePrepared.ReorderPoint = 1
	}
	if parsePrepared.SafetyStock < 0 {
		parsePrepared.SafetyStock = 0
	}
	parseAvailable := max(parsePrepared.OnHand-parsePrepared.Reserved-parsePrepared.Damaged, 0)
	if _, parseErr := parseS.db.ExecContext(parseCtx, `update inventory_levels set on_hand = ?, reserved = ?, available = ?, inbound = ?, damaged = ?, reorder_point = ?, safety_stock = ?, status = ?, updated_at = ? where product_sku = ? and warehouse_id = ?`, parsePrepared.OnHand, parsePrepared.Reserved, parseAvailable, parsePrepared.Inbound, parsePrepared.Damaged, parsePrepared.ReorderPoint, parsePrepared.SafetyStock, parsePrepared.Status, timestampNow(), strings.TrimSpace(parseSku), parsePrepared.WarehouseID); parseErr != nil {
		return repository.InventoryRow{}, fmt.Errorf("update inventory level: %w", parseErr)
	}
	return parseS.inventoryLevelBySKUWarehouse(parseCtx, parseSku, parsePrepared.WarehouseID)
}

func (parseS *Store) CreatePurchaseOrder(parseCtx context.Context, parseInput CreatePurchaseOrderInput) (PurchaseOrderDetailRecord, error) {
	parseNow := timestampNow()
	parseRecord := PurchaseOrderRecord{
		ID:           makeID("po"),
		VendorName:   nonEmptyString(parseInput.VendorName, "Atlas Vendor Network"),
		WarehouseID:  nonEmptyString(parseInput.WarehouseID, "new-jersey-hub"),
		Status:       nonEmptyString(parseInput.Status, "submitted"),
		PriorityNote: nonEmptyString(parseInput.PriorityNote, "Warehouse replenishment"),
		ETA:          nonEmptyString(parseInput.ETA, "Next available lane"),
		CreatedAt:    parseNow,
		UpdatedAt:    parseNow,
	}
	parseLine := PurchaseOrderLineRecord{
		ID:              makeID("po-line"),
		PurchaseOrderID: parseRecord.ID,
		ProductSKU:      strings.TrimSpace(parseInput.ProductSKU),
		Quantity:        parseInput.Quantity,
		ETA:             parseRecord.ETA,
		Status:          "submitted",
	}
	if parseLine.ProductSKU == "" {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("product sku is required")
	}
	if parseLine.Quantity < 1 {
		parseLine.Quantity = 1
	}
	parseTx, parseErr := parseS.db.BeginTx(parseCtx, nil)
	if parseErr != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("begin purchase order transaction: %w", parseErr)
	}
	defer parseTx.Rollback()
	if _, parseErr2 := parseTx.ExecContext(parseCtx, `insert into purchase_orders(id, vendor_name, warehouse_id, status, priority_note, eta, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?)`, parseRecord.ID, parseRecord.VendorName, parseRecord.WarehouseID, parseRecord.Status, parseRecord.PriorityNote, parseRecord.ETA, parseRecord.CreatedAt, parseRecord.UpdatedAt); parseErr2 != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("insert purchase order: %w", parseErr2)
	}
	if _, parseErr3 := parseTx.ExecContext(parseCtx, `insert into purchase_order_lines(id, purchase_order_id, product_sku, quantity, eta, status) values (?, ?, ?, ?, ?, ?)`, parseLine.ID, parseLine.PurchaseOrderID, parseLine.ProductSKU, parseLine.Quantity, parseLine.ETA, parseLine.Status); parseErr3 != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("insert purchase order line: %w", parseErr3)
	}
	if _, parseErr4 := parseTx.ExecContext(parseCtx, `update inventory_levels set inbound = inbound + ?, updated_at = ? where product_sku = ? and warehouse_id = ?`, parseLine.Quantity, parseNow, parseLine.ProductSKU, parseRecord.WarehouseID); parseErr4 != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("increase inbound inventory: %w", parseErr4)
	}
	if parseErr5 := parseTx.Commit(); parseErr5 != nil {
		return PurchaseOrderDetailRecord{}, fmt.Errorf("commit purchase order transaction: %w", parseErr5)
	}
	return parseS.PurchaseOrderDetail(parseCtx, parseRecord.ID)
}

func (parseS *Store) inventoryLevelBySKUWarehouse(parseCtx context.Context, parseSku string, parseWarehouseID string) (repository.InventoryRow, error) {
	var parseItem repository.InventoryRow
	parseErr := parseS.db.QueryRowContext(parseCtx, `select il.id, p.sku, p.title, il.warehouse_id, w.name, il.on_hand, il.reserved, il.available, 14 as cover_days, il.inbound, il.damaged, il.reorder_point, il.safety_stock, il.status, il.updated_at from inventory_levels il join products p on p.sku = il.product_sku join warehouses w on w.id = il.warehouse_id where p.sku = ? and il.warehouse_id = ?`, strings.TrimSpace(parseSku), strings.TrimSpace(parseWarehouseID)).Scan(&parseItem.ID, &parseItem.SKU, &parseItem.Title, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.OnHand, &parseItem.Reserved, &parseItem.Available, &parseItem.CoverDays, &parseItem.Inbound, &parseItem.Damaged, &parseItem.ReorderPoint, &parseItem.SafetyStock, &parseItem.Status, &parseItem.UpdatedAt)
	if parseErr != nil {
		return repository.InventoryRow{}, fmt.Errorf("query inventory level by sku and warehouse: %w", parseErr)
	}
	return parseItem, nil
}

func (parseS *Store) UpdateThreshold(parseCtx context.Context, parseSku string, parseWarehouseID string, parseReorderPoint int, parseSafetyStock int) (repository.InventoryRow, error) {
	if parseReorderPoint < 1 {
		parseReorderPoint = 1
	}
	if parseSafetyStock < 0 {
		parseSafetyStock = 0
	}
	if _, parseErr := parseS.db.ExecContext(parseCtx, `update inventory_levels set reorder_point = ?, safety_stock = ?, updated_at = ? where product_sku = ? and warehouse_id = ?`, parseReorderPoint, parseSafetyStock, timestampNow(), strings.TrimSpace(parseSku), strings.TrimSpace(parseWarehouseID)); parseErr != nil {
		return repository.InventoryRow{}, fmt.Errorf("update threshold: %w", parseErr)
	}
	if _, parseErr2 := parseS.db.ExecContext(parseCtx, `insert into inventory_threshold_events(id, product_sku, warehouse_id, reorder_point, safety_stock, actor_name, summary, detail, created_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?)`, makeID("threshold"), strings.TrimSpace(parseSku), strings.TrimSpace(parseWarehouseID), parseReorderPoint, parseSafetyStock, "Atlas server", "Threshold update staged", fmt.Sprintf("Thresholds updated for %s in %s.", strings.TrimSpace(parseSku), strings.TrimSpace(parseWarehouseID)), timestampNow()); parseErr2 != nil {
		return repository.InventoryRow{}, fmt.Errorf("insert threshold history event: %w", parseErr2)
	}
	return parseS.InventoryBySKU(parseCtx, parseSku)
}

func (parseS *Store) commentByID(parseCtx context.Context, parseId string) (CommentRecord, error) {
	var parseItem CommentRecord
	parseErr := parseS.db.QueryRowContext(parseCtx, `select id, product_sku, author_name, author_type, reaction, subject, body, status, coalesce(moderation_reason, ''), created_at, updated_at from comments where id = ?`, strings.TrimSpace(parseId)).Scan(&parseItem.ID, &parseItem.ProductSKU, &parseItem.AuthorName, &parseItem.AuthorType, &parseItem.Reaction, &parseItem.Subject, &parseItem.Body, &parseItem.Status, &parseItem.ModerationReason, &parseItem.CreatedAt, &parseItem.UpdatedAt)
	if parseErr != nil {
		return CommentRecord{}, fmt.Errorf("query comment by id: %w", parseErr)
	}
	return parseItem, nil
}

func normalizeCommentReaction(parseValue string) string {
	switch strings.TrimSpace(strings.ToLower(parseValue)) {
	case "down", "thumbs_down", "thumbs-down":
		return "down"
	default:
		return "up"
	}
}

func (parseS *Store) receivingByID(parseCtx context.Context, parseId string) (ReceivingSessionRecord, error) {
	var parseItem ReceivingSessionRecord
	parseErr := parseS.db.QueryRowContext(parseCtx, `select id, source_type, source_id, warehouse_id, status, coalesce(discrepancy_summary, ''), created_at, updated_at from receiving_sessions where id = ?`, strings.TrimSpace(parseId)).Scan(&parseItem.ID, &parseItem.SourceType, &parseItem.SourceID, &parseItem.WarehouseID, &parseItem.Status, &parseItem.DiscrepancySummary, &parseItem.CreatedAt, &parseItem.UpdatedAt)
	if parseErr != nil {
		return ReceivingSessionRecord{}, fmt.Errorf("query receiving by id: %w", parseErr)
	}
	return parseItem, nil
}

func makeID(parsePrefix string) string {
	return fmt.Sprintf("%s-%d", parsePrefix, time.Now().UTC().UnixNano())
}

func timestampNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func nullableString(parseValue string) any {
	if strings.TrimSpace(parseValue) == "" {
		return sql.NullString{}
	}
	return parseValue
}

func nonEmptyString(parseValue string, parseFallback string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return parseFallback
	}
	return parseTrimmed
}
