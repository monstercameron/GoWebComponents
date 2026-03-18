package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

type RelatedProductRecord struct {
	SKU           string `json:"sku"`
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	Summary       string `json:"summary"`
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	Reason        string `json:"reason"`
}

type WarehousePressureRecord struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
	Pressure      string `json:"pressure"`
	Staffing      string `json:"staffing"`
	Backlog       string `json:"backlog"`
	Focus         string `json:"focus"`
	Available     int    `json:"available"`
	Inbound       int    `json:"inbound"`
	RiskCount     int    `json:"riskCount"`
}

type ThresholdHistoryRecord struct {
	ID           string `json:"id"`
	ProductSKU   string `json:"productSku"`
	WarehouseID  string `json:"warehouseId"`
	ReorderPoint int    `json:"reorderPoint"`
	SafetyStock  int    `json:"safetyStock"`
	ActorName    string `json:"actorName"`
	Summary      string `json:"summary"`
	Detail       string `json:"detail"`
	CreatedAt    string `json:"createdAt"`
}

type TransferRecommendationRecord struct {
	ProductSKU               string `json:"productSku"`
	SourceWarehouseID        string `json:"sourceWarehouseId"`
	SourceWarehouseName      string `json:"sourceWarehouseName"`
	DestinationWarehouseID   string `json:"destinationWarehouseId"`
	DestinationWarehouseName string `json:"destinationWarehouseName"`
	Quantity                 int    `json:"quantity"`
	Priority                 string `json:"priority"`
	Reason                   string `json:"reason"`
}

type TransferLineRecord struct {
	ID         string `json:"id"`
	TransferID string `json:"transferId"`
	ProductSKU string `json:"productSku"`
	Quantity   int    `json:"quantity"`
}

type TransferDetailRecord struct {
	Transfer TransferRecord       `json:"transfer"`
	Lines    []TransferLineRecord `json:"lines"`
}

type ReceivingLineRecord struct {
	ID                 string `json:"id"`
	ReceivingSessionID string `json:"receivingSessionId"`
	ProductSKU         string `json:"productSku"`
	ExpectedQuantity   int    `json:"expectedQuantity"`
	ActualQuantity     int    `json:"actualQuantity"`
	DiscrepancyReason  string `json:"discrepancyReason"`
}

type ReceivingDetailRecord struct {
	Session ReceivingSessionRecord `json:"session"`
	Lines   []ReceivingLineRecord  `json:"lines"`
}

type PurchaseOrderRecord struct {
	ID            string `json:"id"`
	VendorName    string `json:"vendorName"`
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	Status        string `json:"status"`
	PriorityNote  string `json:"priorityNote"`
	ETA           string `json:"eta"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type PurchaseOrderLineRecord struct {
	ID              string `json:"id"`
	PurchaseOrderID string `json:"purchaseOrderId"`
	ProductSKU      string `json:"productSku"`
	Quantity        int    `json:"quantity"`
	ETA             string `json:"eta"`
	Status          string `json:"status"`
}

type PurchaseOrderDetailRecord struct {
	Order PurchaseOrderRecord       `json:"order"`
	Lines []PurchaseOrderLineRecord `json:"lines"`
}

func (s *Store) ProductComments(ctx context.Context, productSlug string, status string) ([]CommentRecord, error) {
	product, err := s.ProductBySlug(ctx, productSlug)
	if err != nil {
		return nil, err
	}
	query := `select id, product_sku, author_name, author_type, reaction, subject, body, status, coalesce(moderation_reason, ''), created_at, updated_at from comments where product_sku = ?`
	args := []any{product.SKU}
	trimmedStatus := strings.TrimSpace(status)
	if trimmedStatus != "" {
		query += ` and status = ?`
		args = append(args, trimmedStatus)
	}
	query += ` order by created_at desc`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query product comments: %w", err)
	}
	defer rows.Close()
	items := []CommentRecord{}
	for rows.Next() {
		var item CommentRecord
		if err := rows.Scan(&item.ID, &item.ProductSKU, &item.AuthorName, &item.AuthorType, &item.Reaction, &item.Subject, &item.Body, &item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product comment: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) RelatedProducts(ctx context.Context, productSlug string) ([]RelatedProductRecord, error) {
	product, err := s.ProductBySlug(ctx, productSlug)
	if err != nil {
		return nil, err
	}
	items := []repository.Product{}
	rows, err := s.db.QueryContext(ctx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where category = ? and sku <> ? order by title asc limit 3`, product.Category, product.SKU)
	if err != nil {
		return nil, fmt.Errorf("query related products: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item repository.Product
		if err := rows.Scan(&item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.Status, &item.Summary, &item.SEODescription); err != nil {
			return nil, fmt.Errorf("scan related product: %w", err)
		}
		items = append(items, item)
	}
	if len(items) < 3 {
		fallbackRows, err := s.db.QueryContext(ctx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where sku <> ? and category <> ? order by title asc limit ?`, product.SKU, product.Category, 3-len(items))
		if err != nil {
			return nil, fmt.Errorf("query fallback related products: %w", err)
		}
		defer fallbackRows.Close()
		for fallbackRows.Next() {
			var item repository.Product
			if err := fallbackRows.Scan(&item.SKU, &item.Slug, &item.Title, &item.Category, &item.PriceCents, &item.Status, &item.Summary, &item.SEODescription); err != nil {
				return nil, fmt.Errorf("scan fallback related product: %w", err)
			}
			items = append(items, item)
		}
	}
	result := make([]RelatedProductRecord, 0, len(items))
	for _, item := range items {
		warehouse, err := s.topWarehouseForProduct(ctx, item.SKU)
		if err != nil {
			return nil, err
		}
		result = append(result, RelatedProductRecord{
			SKU:           item.SKU,
			Slug:          item.Slug,
			Title:         item.Title,
			Category:      item.Category,
			Summary:       item.Summary,
			WarehouseID:   warehouse.ID,
			WarehouseName: warehouse.Name,
			Reason:        relatedProductReason(product, item),
		})
	}
	return result, nil
}

func (s *Store) WarehousePressure(ctx context.Context) ([]WarehousePressureRecord, error) {
	warehouses, err := s.Warehouses(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]WarehousePressureRecord, 0, len(warehouses))
	for _, warehouse := range warehouses {
		item, err := s.WarehousePressureByID(ctx, warehouse.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) WarehousePressureByID(ctx context.Context, warehouseID string) (WarehousePressureRecord, error) {
	warehouse, err := s.warehouseByID(ctx, warehouseID)
	if err != nil {
		return WarehousePressureRecord{}, err
	}
	item := WarehousePressureRecord{
		ID:            warehouse.ID,
		Slug:          warehouse.Slug,
		Name:          warehouse.Name,
		Region:        warehouse.Region,
		ServiceLevel:  warehouse.ServiceLevel,
		PublicSummary: warehouse.PublicSummary,
	}
	if err := s.db.QueryRowContext(ctx, `select coalesce(sum(available), 0), coalesce(sum(inbound), 0), coalesce(sum(case when status = 'promise_risk' then 1 else 0 end), 0) from inventory_levels where warehouse_id = ?`, warehouse.ID).Scan(&item.Available, &item.Inbound, &item.RiskCount); err != nil {
		return WarehousePressureRecord{}, fmt.Errorf("query warehouse pressure: %w", err)
	}
	item.Pressure, item.Staffing, item.Backlog, item.Focus = warehouseOperationalProfile(warehouse.ID)
	if item.RiskCount > 0 && item.Pressure == "Balancing lane" {
		item.Pressure = "Promise risk"
	}
	return item, nil
}

func (s *Store) ThresholdHistory(ctx context.Context, sku string) ([]ThresholdHistoryRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select id, product_sku, warehouse_id, reorder_point, safety_stock, actor_name, summary, detail, created_at from inventory_threshold_events where product_sku = ? order by created_at desc`, strings.TrimSpace(sku))
	if err != nil {
		return nil, fmt.Errorf("query threshold history: %w", err)
	}
	defer rows.Close()
	items := []ThresholdHistoryRecord{}
	for rows.Next() {
		var item ThresholdHistoryRecord
		if err := rows.Scan(&item.ID, &item.ProductSKU, &item.WarehouseID, &item.ReorderPoint, &item.SafetyStock, &item.ActorName, &item.Summary, &item.Detail, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan threshold history: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) TransferRecommendations(ctx context.Context, sku string) ([]TransferRecommendationRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select warehouse_id, available from inventory_levels where product_sku = ? order by available desc`, strings.TrimSpace(sku))
	if err != nil {
		return nil, fmt.Errorf("query transfer recommendations: %w", err)
	}
	defer rows.Close()
	type lane struct {
		warehouseID string
		available   int
	}
	lanes := []lane{}
	for rows.Next() {
		var item lane
		if err := rows.Scan(&item.warehouseID, &item.available); err != nil {
			return nil, fmt.Errorf("scan transfer recommendation lane: %w", err)
		}
		lanes = append(lanes, item)
	}
	if len(lanes) < 2 {
		return []TransferRecommendationRecord{}, nil
	}
	source := lanes[0]
	destination := lanes[len(lanes)-1]
	if source.available <= destination.available {
		return []TransferRecommendationRecord{}, nil
	}
	quantity := (source.available - destination.available) / 2
	if quantity < 1 {
		quantity = 1
	}
	sourceWarehouse, err := s.warehouseByID(ctx, source.warehouseID)
	if err != nil {
		return nil, err
	}
	destinationWarehouse, err := s.warehouseByID(ctx, destination.warehouseID)
	if err != nil {
		return nil, err
	}
	priority := "Balancing move"
	if destination.available <= 3 {
		priority = "Promise recovery"
	}
	return []TransferRecommendationRecord{{
		ProductSKU:               strings.TrimSpace(sku),
		SourceWarehouseID:        sourceWarehouse.ID,
		SourceWarehouseName:      sourceWarehouse.Name,
		DestinationWarehouseID:   destinationWarehouse.ID,
		DestinationWarehouseName: destinationWarehouse.Name,
		Quantity:                 quantity,
		Priority:                 priority,
		Reason:                   fmt.Sprintf("Shift %d units from %s to %s to stabilize the lowest availability lane.", quantity, sourceWarehouse.Name, destinationWarehouse.Name),
	}}, nil
}

func (s *Store) TransferByID(ctx context.Context, id string) (TransferRecord, error) {
	var item TransferRecord
	err := s.db.QueryRowContext(ctx, `select id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at from transfers where id = ?`, strings.TrimSpace(id)).Scan(&item.ID, &item.SourceWarehouseID, &item.DestinationWarehouse, &item.Status, &item.Reason, &item.RecommendedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return TransferRecord{}, fmt.Errorf("query transfer by id: %w", err)
	}
	return item, nil
}

func (s *Store) TransferLines(ctx context.Context, transferID string) ([]TransferLineRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select id, transfer_id, product_sku, quantity from transfer_lines where transfer_id = ? order by id asc`, strings.TrimSpace(transferID))
	if err != nil {
		return nil, fmt.Errorf("query transfer lines: %w", err)
	}
	defer rows.Close()
	items := []TransferLineRecord{}
	for rows.Next() {
		var item TransferLineRecord
		if err := rows.Scan(&item.ID, &item.TransferID, &item.ProductSKU, &item.Quantity); err != nil {
			return nil, fmt.Errorf("scan transfer line: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) TransferDetail(ctx context.Context, id string) (TransferDetailRecord, error) {
	transfer, err := s.TransferByID(ctx, id)
	if err != nil {
		return TransferDetailRecord{}, err
	}
	lines, err := s.TransferLines(ctx, id)
	if err != nil {
		return TransferDetailRecord{}, err
	}
	return TransferDetailRecord{Transfer: transfer, Lines: lines}, nil
}

func (s *Store) ReceivingLines(ctx context.Context, sessionID string) ([]ReceivingLineRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select id, receiving_session_id, product_sku, expected_quantity, actual_quantity, discrepancy_reason from receiving_lines where receiving_session_id = ? order by id asc`, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, fmt.Errorf("query receiving lines: %w", err)
	}
	defer rows.Close()
	items := []ReceivingLineRecord{}
	for rows.Next() {
		var item ReceivingLineRecord
		if err := rows.Scan(&item.ID, &item.ReceivingSessionID, &item.ProductSKU, &item.ExpectedQuantity, &item.ActualQuantity, &item.DiscrepancyReason); err != nil {
			return nil, fmt.Errorf("scan receiving line: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) ReceivingDetail(ctx context.Context, id string) (ReceivingDetailRecord, error) {
	session, err := s.receivingByID(ctx, id)
	if err != nil {
		return ReceivingDetailRecord{}, err
	}
	lines, err := s.ReceivingLines(ctx, id)
	if err != nil {
		return ReceivingDetailRecord{}, err
	}
	return ReceivingDetailRecord{Session: session, Lines: lines}, nil
}

func (s *Store) PurchaseOrders(ctx context.Context) ([]PurchaseOrderRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select po.id, po.vendor_name, po.warehouse_id, w.name, po.status, po.priority_note, po.eta, po.created_at, po.updated_at from purchase_orders po join warehouses w on w.id = po.warehouse_id order by po.created_at desc`)
	if err != nil {
		return nil, fmt.Errorf("query purchase orders: %w", err)
	}
	defer rows.Close()
	items := []PurchaseOrderRecord{}
	for rows.Next() {
		var item PurchaseOrderRecord
		if err := rows.Scan(&item.ID, &item.VendorName, &item.WarehouseID, &item.WarehouseName, &item.Status, &item.PriorityNote, &item.ETA, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan purchase order: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) PurchaseOrdersByWarehouse(ctx context.Context, warehouseID string) ([]PurchaseOrderRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select po.id, po.vendor_name, po.warehouse_id, w.name, po.status, po.priority_note, po.eta, po.created_at, po.updated_at from purchase_orders po join warehouses w on w.id = po.warehouse_id where po.warehouse_id = ? order by po.created_at desc`, strings.TrimSpace(warehouseID))
	if err != nil {
		return nil, fmt.Errorf("query purchase orders by warehouse: %w", err)
	}
	defer rows.Close()
	items := []PurchaseOrderRecord{}
	for rows.Next() {
		var item PurchaseOrderRecord
		if err := rows.Scan(&item.ID, &item.VendorName, &item.WarehouseID, &item.WarehouseName, &item.Status, &item.PriorityNote, &item.ETA, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan purchase order by warehouse: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) PurchaseOrderByID(ctx context.Context, id string) (PurchaseOrderRecord, error) {
	var item PurchaseOrderRecord
	err := s.db.QueryRowContext(ctx, `select po.id, po.vendor_name, po.warehouse_id, w.name, po.status, po.priority_note, po.eta, po.created_at, po.updated_at from purchase_orders po join warehouses w on w.id = po.warehouse_id where po.id = ?`, strings.TrimSpace(id)).Scan(&item.ID, &item.VendorName, &item.WarehouseID, &item.WarehouseName, &item.Status, &item.PriorityNote, &item.ETA, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return PurchaseOrderRecord{}, fmt.Errorf("query purchase order by id: %w", err)
	}
	return item, nil
}

func (s *Store) PurchaseOrderLines(ctx context.Context, purchaseOrderID string) ([]PurchaseOrderLineRecord, error) {
	rows, err := s.db.QueryContext(ctx, `select id, purchase_order_id, product_sku, quantity, eta, status from purchase_order_lines where purchase_order_id = ? order by id asc`, strings.TrimSpace(purchaseOrderID))
	if err != nil {
		return nil, fmt.Errorf("query purchase order lines: %w", err)
	}
	defer rows.Close()
	items := []PurchaseOrderLineRecord{}
	for rows.Next() {
		var item PurchaseOrderLineRecord
		if err := rows.Scan(&item.ID, &item.PurchaseOrderID, &item.ProductSKU, &item.Quantity, &item.ETA, &item.Status); err != nil {
			return nil, fmt.Errorf("scan purchase order line: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) PurchaseOrderDetail(ctx context.Context, id string) (PurchaseOrderDetailRecord, error) {
	order, err := s.PurchaseOrderByID(ctx, id)
	if err != nil {
		return PurchaseOrderDetailRecord{}, err
	}
	lines, err := s.PurchaseOrderLines(ctx, id)
	if err != nil {
		return PurchaseOrderDetailRecord{}, err
	}
	return PurchaseOrderDetailRecord{Order: order, Lines: lines}, nil
}

func (s *Store) topWarehouseForProduct(ctx context.Context, sku string) (Warehouse, error) {
	var warehouseID string
	err := s.db.QueryRowContext(ctx, `select warehouse_id from inventory_levels where product_sku = ? order by available desc, inbound desc limit 1`, sku).Scan(&warehouseID)
	if err != nil {
		if err == sql.ErrNoRows {
			return Warehouse{ID: "new-jersey-hub", Slug: "new-jersey-hub", Name: "New Jersey Hub"}, nil
		}
		return Warehouse{}, fmt.Errorf("query top warehouse for product: %w", err)
	}
	return s.warehouseByID(ctx, warehouseID)
}

func (s *Store) warehouseByID(ctx context.Context, id string) (Warehouse, error) {
	var item Warehouse
	err := s.db.QueryRowContext(ctx, `select id, slug, name, region, service_level, public_summary from warehouses where id = ?`, strings.TrimSpace(id)).Scan(&item.ID, &item.Slug, &item.Name, &item.Region, &item.ServiceLevel, &item.PublicSummary)
	if err != nil {
		return Warehouse{}, fmt.Errorf("query warehouse by id: %w", err)
	}
	return item, nil
}

func relatedProductReason(base repository.Product, candidate repository.Product) string {
	if candidate.Category == base.Category {
		return "Extends the same merchandising family for buyers comparing adjacent configurations."
	}
	return "Pairs naturally with the current product story when the buyer needs a broader system view."
}

func warehouseOperationalProfile(warehouseID string) (pressure string, staffing string, backlog string, focus string) {
	switch strings.TrimSpace(warehouseID) {
	case "new-jersey-hub":
		return "Promise risk", "88% staffed", "2 blocked receipts", "Protect east-coast promise windows and clear blocked receiving before launch traffic spikes."
	case "nevada-hub":
		return "Transfer source", "91% staffed", "1 receiving hold", "Use Nevada as the primary transfer source while inbound volume stays stable."
	default:
		return "Balancing lane", "84% staffed", "2 dock queues", "Balance inbound accessories against east-coast shortages."
	}
}
