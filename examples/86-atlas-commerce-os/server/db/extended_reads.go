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

func (parseS *Store) ProductComments(parseCtx context.Context, parseProductSlug string, parseStatus string) ([]CommentRecord, error) {
	parseProduct, parseErr := parseS.ProductBySlug(parseCtx, parseProductSlug)
	if parseErr != nil {
		return nil, parseErr
	}
	parseQuery := `select id, product_sku, author_name, author_type, reaction, subject, body, status, coalesce(moderation_reason, ''), created_at, updated_at from comments where product_sku = ?`
	parseArgs := []any{parseProduct.SKU}
	parseTrimmedStatus := strings.TrimSpace(parseStatus)
	if parseTrimmedStatus != "" {
		parseQuery += ` and status = ?`
		parseArgs = append(parseArgs, parseTrimmedStatus)
	}
	parseQuery += ` order by created_at desc`
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, parseQuery, parseArgs...)
	if parseErr != nil {
		return nil, fmt.Errorf("query product comments: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []CommentRecord{}
	for parseRows.Next() {
		var parseItem CommentRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.ProductSKU, &parseItem.AuthorName, &parseItem.AuthorType, &parseItem.Reaction, &parseItem.Subject, &parseItem.Body, &parseItem.Status, &parseItem.ModerationReason, &parseItem.CreatedAt, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan product comment: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) RelatedProducts(parseCtx context.Context, parseProductSlug string) ([]RelatedProductRecord, error) {
	parseProduct, parseErr := parseS.ProductBySlug(parseCtx, parseProductSlug)
	if parseErr != nil {
		return nil, parseErr
	}
	parseItems := []repository.Product{}
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where category = ? and sku <> ? order by title asc limit 3`, parseProduct.Category, parseProduct.SKU)
	if parseErr != nil {
		return nil, fmt.Errorf("query related products: %w", parseErr)
	}
	defer parseRows.Close()
	for parseRows.Next() {
		var parseItem repository.Product
		if parseErr2 := parseRows.Scan(&parseItem.SKU, &parseItem.Slug, &parseItem.Title, &parseItem.Category, &parseItem.PriceCents, &parseItem.Status, &parseItem.Summary, &parseItem.SEODescription); parseErr2 != nil {
			return nil, fmt.Errorf("scan related product: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	if len(parseItems) < 3 {
		parseFallbackRows, parseErr3 := parseS.db.QueryContext(parseCtx, `select sku, slug, title, category, price_cents, status, summary, seo_description from products where sku <> ? and category <> ? order by title asc limit ?`, parseProduct.SKU, parseProduct.Category, 3-len(parseItems))
		if parseErr3 != nil {
			return nil, fmt.Errorf("query fallback related products: %w", parseErr3)
		}
		defer parseFallbackRows.Close()
		for parseFallbackRows.Next() {
			var parseItem2 repository.Product
			if parseErr4 := parseFallbackRows.Scan(&parseItem2.SKU, &parseItem2.Slug, &parseItem2.Title, &parseItem2.Category, &parseItem2.PriceCents, &parseItem2.Status, &parseItem2.Summary, &parseItem2.SEODescription); parseErr4 != nil {
				return nil, fmt.Errorf("scan fallback related product: %w", parseErr4)
			}
			parseItems = append(parseItems, parseItem2)
		}
	}
	parseResult := make([]RelatedProductRecord, 0, len(parseItems))
	for _, parseItem3 := range parseItems {
		parseWarehouse, parseErr5 := parseS.topWarehouseForProduct(parseCtx, parseItem3.SKU)
		if parseErr5 != nil {
			return nil, parseErr5
		}
		parseResult = append(parseResult, RelatedProductRecord{
			SKU:           parseItem3.SKU,
			Slug:          parseItem3.Slug,
			Title:         parseItem3.Title,
			Category:      parseItem3.Category,
			Summary:       parseItem3.Summary,
			WarehouseID:   parseWarehouse.ID,
			WarehouseName: parseWarehouse.Name,
			Reason:        relatedProductReason(parseProduct, parseItem3),
		})
	}
	return parseResult, nil
}

func (parseS *Store) WarehousePressure(parseCtx context.Context) ([]WarehousePressureRecord, error) {
	parseWarehouses, parseErr := parseS.Warehouses(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseItems := make([]WarehousePressureRecord, 0, len(parseWarehouses))
	for _, parseWarehouse := range parseWarehouses {
		parseItem, parseErr2 := parseS.WarehousePressureByID(parseCtx, parseWarehouse.ID)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) WarehousePressureByID(parseCtx context.Context, parseWarehouseID string) (WarehousePressureRecord, error) {
	parseWarehouse, parseErr := parseS.warehouseByID(parseCtx, parseWarehouseID)
	if parseErr != nil {
		return WarehousePressureRecord{}, parseErr
	}
	parseItem := WarehousePressureRecord{
		ID:            parseWarehouse.ID,
		Slug:          parseWarehouse.Slug,
		Name:          parseWarehouse.Name,
		Region:        parseWarehouse.Region,
		ServiceLevel:  parseWarehouse.ServiceLevel,
		PublicSummary: parseWarehouse.PublicSummary,
	}
	if parseErr2 := parseS.db.QueryRowContext(parseCtx, `select coalesce(sum(available), 0), coalesce(sum(inbound), 0), coalesce(sum(case when status = 'promise_risk' then 1 else 0 end), 0) from inventory_levels where warehouse_id = ?`, parseWarehouse.ID).Scan(&parseItem.Available, &parseItem.Inbound, &parseItem.RiskCount); parseErr2 != nil {
		return WarehousePressureRecord{}, fmt.Errorf("query warehouse pressure: %w", parseErr2)
	}
	parseItem.Pressure, parseItem.Staffing, parseItem.Backlog, parseItem.Focus = warehouseOperationalProfile(parseWarehouse.ID)
	if parseItem.RiskCount > 0 && parseItem.Pressure == "Balancing lane" {
		parseItem.Pressure = "Promise risk"
	}
	return parseItem, nil
}

func (parseS *Store) ThresholdHistory(parseCtx context.Context, parseSku string) ([]ThresholdHistoryRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, product_sku, warehouse_id, reorder_point, safety_stock, actor_name, summary, detail, created_at from inventory_threshold_events where product_sku = ? order by created_at desc`, strings.TrimSpace(parseSku))
	if parseErr != nil {
		return nil, fmt.Errorf("query threshold history: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []ThresholdHistoryRecord{}
	for parseRows.Next() {
		var parseItem ThresholdHistoryRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.ProductSKU, &parseItem.WarehouseID, &parseItem.ReorderPoint, &parseItem.SafetyStock, &parseItem.ActorName, &parseItem.Summary, &parseItem.Detail, &parseItem.CreatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan threshold history: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) TransferRecommendations(parseCtx context.Context, parseSku string) ([]TransferRecommendationRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select warehouse_id, available from inventory_levels where product_sku = ? order by available desc`, strings.TrimSpace(parseSku))
	if parseErr != nil {
		return nil, fmt.Errorf("query transfer recommendations: %w", parseErr)
	}
	defer parseRows.Close()
	type lane struct {
		warehouseID string
		available   int
	}
	parseLanes := []lane{}
	for parseRows.Next() {
		var parseItem lane
		if parseErr2 := parseRows.Scan(&parseItem.warehouseID, &parseItem.available); parseErr2 != nil {
			return nil, fmt.Errorf("scan transfer recommendation lane: %w", parseErr2)
		}
		parseLanes = append(parseLanes, parseItem)
	}
	if len(parseLanes) < 2 {
		return []TransferRecommendationRecord{}, nil
	}
	parseSource := parseLanes[0]
	parseDestination := parseLanes[len(parseLanes)-1]
	if parseSource.available <= parseDestination.available {
		return []TransferRecommendationRecord{}, nil
	}
	parseQuantity := (parseSource.available - parseDestination.available) / 2
	if parseQuantity < 1 {
		parseQuantity = 1
	}
	parseSourceWarehouse, parseErr := parseS.warehouseByID(parseCtx, parseSource.warehouseID)
	if parseErr != nil {
		return nil, parseErr
	}
	parseDestinationWarehouse, parseErr := parseS.warehouseByID(parseCtx, parseDestination.warehouseID)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePriority := "Balancing move"
	if parseDestination.available <= 3 {
		parsePriority = "Promise recovery"
	}
	return []TransferRecommendationRecord{{
		ProductSKU:               strings.TrimSpace(parseSku),
		SourceWarehouseID:        parseSourceWarehouse.ID,
		SourceWarehouseName:      parseSourceWarehouse.Name,
		DestinationWarehouseID:   parseDestinationWarehouse.ID,
		DestinationWarehouseName: parseDestinationWarehouse.Name,
		Quantity:                 parseQuantity,
		Priority:                 parsePriority,
		Reason:                   fmt.Sprintf("Shift %d units from %s to %s to stabilize the lowest availability lane.", parseQuantity, parseSourceWarehouse.Name, parseDestinationWarehouse.Name),
	}}, nil
}

func (parseS *Store) TransferByID(parseCtx context.Context, parseId string) (TransferRecord, error) {
	var parseItem TransferRecord
	parseErr := parseS.db.QueryRowContext(parseCtx, `select id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at from transfers where id = ?`, strings.TrimSpace(parseId)).Scan(&parseItem.ID, &parseItem.SourceWarehouseID, &parseItem.DestinationWarehouse, &parseItem.Status, &parseItem.Reason, &parseItem.RecommendedBy, &parseItem.CreatedAt, &parseItem.UpdatedAt)
	if parseErr != nil {
		return TransferRecord{}, fmt.Errorf("query transfer by id: %w", parseErr)
	}
	return parseItem, nil
}

func (parseS *Store) TransferLines(parseCtx context.Context, parseTransferID string) ([]TransferLineRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, transfer_id, product_sku, quantity from transfer_lines where transfer_id = ? order by id asc`, strings.TrimSpace(parseTransferID))
	if parseErr != nil {
		return nil, fmt.Errorf("query transfer lines: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []TransferLineRecord{}
	for parseRows.Next() {
		var parseItem TransferLineRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.TransferID, &parseItem.ProductSKU, &parseItem.Quantity); parseErr2 != nil {
			return nil, fmt.Errorf("scan transfer line: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) TransferDetail(parseCtx context.Context, parseId string) (TransferDetailRecord, error) {
	parseTransfer, parseErr := parseS.TransferByID(parseCtx, parseId)
	if parseErr != nil {
		return TransferDetailRecord{}, parseErr
	}
	parseLines, parseErr := parseS.TransferLines(parseCtx, parseId)
	if parseErr != nil {
		return TransferDetailRecord{}, parseErr
	}
	return TransferDetailRecord{Transfer: parseTransfer, Lines: parseLines}, nil
}

func (parseS *Store) ReceivingLines(parseCtx context.Context, parseSessionID string) ([]ReceivingLineRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, receiving_session_id, product_sku, expected_quantity, actual_quantity, discrepancy_reason from receiving_lines where receiving_session_id = ? order by id asc`, strings.TrimSpace(parseSessionID))
	if parseErr != nil {
		return nil, fmt.Errorf("query receiving lines: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []ReceivingLineRecord{}
	for parseRows.Next() {
		var parseItem ReceivingLineRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.ReceivingSessionID, &parseItem.ProductSKU, &parseItem.ExpectedQuantity, &parseItem.ActualQuantity, &parseItem.DiscrepancyReason); parseErr2 != nil {
			return nil, fmt.Errorf("scan receiving line: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) ReceivingDetail(parseCtx context.Context, parseId string) (ReceivingDetailRecord, error) {
	parseSession, parseErr := parseS.receivingByID(parseCtx, parseId)
	if parseErr != nil {
		return ReceivingDetailRecord{}, parseErr
	}
	parseLines, parseErr := parseS.ReceivingLines(parseCtx, parseId)
	if parseErr != nil {
		return ReceivingDetailRecord{}, parseErr
	}
	return ReceivingDetailRecord{Session: parseSession, Lines: parseLines}, nil
}

func (parseS *Store) PurchaseOrders(parseCtx context.Context) ([]PurchaseOrderRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select po.id, po.vendor_name, po.warehouse_id, w.name, po.status, po.priority_note, po.eta, po.created_at, po.updated_at from purchase_orders po join warehouses w on w.id = po.warehouse_id order by po.created_at desc`)
	if parseErr != nil {
		return nil, fmt.Errorf("query purchase orders: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []PurchaseOrderRecord{}
	for parseRows.Next() {
		var parseItem PurchaseOrderRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.VendorName, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.Status, &parseItem.PriorityNote, &parseItem.ETA, &parseItem.CreatedAt, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan purchase order: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) PurchaseOrdersByWarehouse(parseCtx context.Context, parseWarehouseID string) ([]PurchaseOrderRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select po.id, po.vendor_name, po.warehouse_id, w.name, po.status, po.priority_note, po.eta, po.created_at, po.updated_at from purchase_orders po join warehouses w on w.id = po.warehouse_id where po.warehouse_id = ? order by po.created_at desc`, strings.TrimSpace(parseWarehouseID))
	if parseErr != nil {
		return nil, fmt.Errorf("query purchase orders by warehouse: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []PurchaseOrderRecord{}
	for parseRows.Next() {
		var parseItem PurchaseOrderRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.VendorName, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.Status, &parseItem.PriorityNote, &parseItem.ETA, &parseItem.CreatedAt, &parseItem.UpdatedAt); parseErr2 != nil {
			return nil, fmt.Errorf("scan purchase order by warehouse: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) PurchaseOrderByID(parseCtx context.Context, parseId string) (PurchaseOrderRecord, error) {
	var parseItem PurchaseOrderRecord
	parseErr := parseS.db.QueryRowContext(parseCtx, `select po.id, po.vendor_name, po.warehouse_id, w.name, po.status, po.priority_note, po.eta, po.created_at, po.updated_at from purchase_orders po join warehouses w on w.id = po.warehouse_id where po.id = ?`, strings.TrimSpace(parseId)).Scan(&parseItem.ID, &parseItem.VendorName, &parseItem.WarehouseID, &parseItem.WarehouseName, &parseItem.Status, &parseItem.PriorityNote, &parseItem.ETA, &parseItem.CreatedAt, &parseItem.UpdatedAt)
	if parseErr != nil {
		return PurchaseOrderRecord{}, fmt.Errorf("query purchase order by id: %w", parseErr)
	}
	return parseItem, nil
}

func (parseS *Store) PurchaseOrderLines(parseCtx context.Context, parsePurchaseOrderID string) ([]PurchaseOrderLineRecord, error) {
	parseRows, parseErr := parseS.db.QueryContext(parseCtx, `select id, purchase_order_id, product_sku, quantity, eta, status from purchase_order_lines where purchase_order_id = ? order by id asc`, strings.TrimSpace(parsePurchaseOrderID))
	if parseErr != nil {
		return nil, fmt.Errorf("query purchase order lines: %w", parseErr)
	}
	defer parseRows.Close()
	parseItems := []PurchaseOrderLineRecord{}
	for parseRows.Next() {
		var parseItem PurchaseOrderLineRecord
		if parseErr2 := parseRows.Scan(&parseItem.ID, &parseItem.PurchaseOrderID, &parseItem.ProductSKU, &parseItem.Quantity, &parseItem.ETA, &parseItem.Status); parseErr2 != nil {
			return nil, fmt.Errorf("scan purchase order line: %w", parseErr2)
		}
		parseItems = append(parseItems, parseItem)
	}
	return parseItems, nil
}

func (parseS *Store) PurchaseOrderDetail(parseCtx context.Context, parseId string) (PurchaseOrderDetailRecord, error) {
	parseOrder, parseErr := parseS.PurchaseOrderByID(parseCtx, parseId)
	if parseErr != nil {
		return PurchaseOrderDetailRecord{}, parseErr
	}
	parseLines, parseErr := parseS.PurchaseOrderLines(parseCtx, parseId)
	if parseErr != nil {
		return PurchaseOrderDetailRecord{}, parseErr
	}
	return PurchaseOrderDetailRecord{Order: parseOrder, Lines: parseLines}, nil
}

func (parseS *Store) topWarehouseForProduct(parseCtx context.Context, parseSku string) (Warehouse, error) {
	var parseWarehouseID string
	parseErr := parseS.db.QueryRowContext(parseCtx, `select warehouse_id from inventory_levels where product_sku = ? order by available desc, inbound desc limit 1`, parseSku).Scan(&parseWarehouseID)
	if parseErr != nil {
		if parseErr == sql.ErrNoRows {
			return Warehouse{ID: "new-jersey-hub", Slug: "new-jersey-hub", Name: "New Jersey Hub"}, nil
		}
		return Warehouse{}, fmt.Errorf("query top warehouse for product: %w", parseErr)
	}
	return parseS.warehouseByID(parseCtx, parseWarehouseID)
}

func (parseS *Store) warehouseByID(parseCtx context.Context, parseId string) (Warehouse, error) {
	var parseItem Warehouse
	parseErr := parseS.db.QueryRowContext(parseCtx, `select id, slug, name, region, service_level, public_summary from warehouses where id = ?`, strings.TrimSpace(parseId)).Scan(&parseItem.ID, &parseItem.Slug, &parseItem.Name, &parseItem.Region, &parseItem.ServiceLevel, &parseItem.PublicSummary)
	if parseErr != nil {
		return Warehouse{}, fmt.Errorf("query warehouse by id: %w", parseErr)
	}
	return parseItem, nil
}

func relatedProductReason(parseBase repository.Product, parseCandidate repository.Product) string {
	if parseCandidate.Category == parseBase.Category {
		return "Extends the same merchandising family for buyers comparing adjacent configurations."
	}
	return "Pairs naturally with the current product story when the buyer needs a broader system view."
}

func warehouseOperationalProfile(parseWarehouseID string) (parsePressure string, parseStaffing string, parseBacklog string, parseFocus string) {
	switch strings.TrimSpace(parseWarehouseID) {
	case "new-jersey-hub":
		return "Promise risk", "88% staffed", "2 blocked receipts", "Protect east-coast promise windows and clear blocked receiving before launch traffic spikes."
	case "nevada-hub":
		return "Transfer source", "91% staffed", "1 receiving hold", "Use Nevada as the primary transfer source while inbound volume stays stable."
	default:
		return "Balancing lane", "84% staffed", "2 dock queues", "Balance inbound accessories against east-coast shortages."
	}
}
