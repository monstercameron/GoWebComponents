package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

func TestLoadMigrationsAndMigrateFallback(parseT *testing.T) {
	parseT.Parallel()

	parseTempDir := parseT.TempDir()
	parseMigrationsDir := filepath.Join(parseTempDir, "migrations")
	if parseErr := os.MkdirAll(parseMigrationsDir, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir migrations: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseMigrationsDir, "002_b.sql"), []byte("create table if not exists b(id text);"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write migration b: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseMigrationsDir, "001_a.sql"), []byte("create table if not exists a(id text);"), 0o644); parseErr3 != nil {
		parseT.Fatalf("write migration a: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseMigrationsDir, "README.txt"), []byte("ignored"), 0o644); parseErr4 != nil {
		parseT.Fatalf("write migration noise: %v", parseErr4)
	}

	parseFallback := filepath.Join(parseTempDir, "schema.sql")
	if parseErr5 := os.WriteFile(parseFallback, []byte("create table if not exists fallback_only(id text);"), 0o644); parseErr5 != nil {
		parseT.Fatalf("write fallback schema: %v", parseErr5)
	}

	parseLoaded, parseErr6 := loadMigrations(parseMigrationsDir, parseFallback)
	if parseErr6 != nil {
		parseT.Fatalf("load migrations: %v", parseErr6)
	}
	if len(parseLoaded) != 2 {
		parseT.Fatalf("expected 2 migrations, got %d", len(parseLoaded))
	}
	if parseLoaded[0].Version != "001_a.sql" || parseLoaded[1].Version != "002_b.sql" {
		parseT.Fatalf("migrations not sorted: %#v", parseLoaded)
	}

	parseFallbackOnlyDir := filepath.Join(parseTempDir, "missing-migrations")
	parseLoadedFallback, parseErr6 := loadMigrations(parseFallbackOnlyDir, parseFallback)
	if parseErr6 != nil {
		parseT.Fatalf("load fallback migrations: %v", parseErr6)
	}
	if len(parseLoadedFallback) != 1 || parseLoadedFallback[0].Version != "001_initial_schema.sql" {
		parseT.Fatalf("unexpected fallback migration payload: %#v", parseLoadedFallback)
	}

	_, parseErr6 = loadMigrations(parseFallbackOnlyDir, filepath.Join(parseTempDir, "missing-schema.sql"))
	if parseErr6 == nil {
		parseT.Fatalf("expected error when no migrations and no fallback schema")
	}

	parseCtx := context.Background()
	parseDatabase, parseErr6 := Open(parseCtx, filepath.Join(parseTempDir, "atlas.db"))
	if parseErr6 != nil {
		parseT.Fatalf("open sqlite: %v", parseErr6)
	}
	parseT.Cleanup(func() {
		_ = parseDatabase.Close()
	})
	if parseErr7 := Migrate(parseCtx, parseDatabase, parseFallbackOnlyDir, parseFallback); parseErr7 != nil {
		parseT.Fatalf("migrate with fallback: %v", parseErr7)
	}
	if parseErr8 := Migrate(parseCtx, parseDatabase, parseFallbackOnlyDir, parseFallback); parseErr8 != nil {
		parseT.Fatalf("migrate idempotency: %v", parseErr8)
	}

	var parseMigrationCount int
	if parseErr9 := parseDatabase.QueryRowContext(parseCtx, `select count(*) from schema_migrations`).Scan(&parseMigrationCount); parseErr9 != nil {
		parseT.Fatalf("count schema migrations: %v", parseErr9)
	}
	if parseMigrationCount != 1 {
		parseT.Fatalf("expected one applied migration, got %d", parseMigrationCount)
	}
}

func TestStoreReadFlows(parseT *testing.T) {
	parseT.Parallel()

	parseCtx, store, parseDatabase := newSeededStore(parseT)
	if parseErr := Seed(parseCtx, parseDatabase); parseErr != nil {
		parseT.Fatalf("seed second pass: %v", parseErr)
	}

	parseCatalog, parseErr2 := store.Catalog(parseCtx, CatalogQuery{
		Search:    "frame",
		Category:  "desks",
		Warehouse: "new-jersey-hub",
		Sort:      "warehouse",
	})
	if parseErr2 != nil {
		parseT.Fatalf("catalog: %v", parseErr2)
	}
	if parseCatalog.Page != 1 || parseCatalog.PageSize != 12 {
		parseT.Fatalf("catalog paging defaults not applied: %#v", parseCatalog.Query)
	}
	if parseCatalog.Total < 1 || len(parseCatalog.Items) < 1 {
		parseT.Fatalf("expected catalog rows, got total=%d items=%d", parseCatalog.Total, len(parseCatalog.Items))
	}

	parseProduct, parseErr2 := store.ProductBySlug(parseCtx, "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("product by slug: %v", parseErr2)
	}
	if parseProduct.SKU != "frame-desk" {
		parseT.Fatalf("unexpected product sku: %s", parseProduct.SKU)
	}

	parseWarehouses, parseErr2 := store.Warehouses(parseCtx)
	if parseErr2 != nil {
		parseT.Fatalf("warehouses: %v", parseErr2)
	}
	if len(parseWarehouses) < 3 {
		parseT.Fatalf("expected seeded warehouses, got %d", len(parseWarehouses))
	}
	parseWarehouse, parseErr2 := store.WarehouseBySlug(parseCtx, "new-jersey-hub")
	if parseErr2 != nil {
		parseT.Fatalf("warehouse by slug: %v", parseErr2)
	}
	if parseWarehouse.ID != "new-jersey-hub" {
		parseT.Fatalf("unexpected warehouse id: %s", parseWarehouse.ID)
	}

	parseAvailability, parseErr2 := store.Availability(parseCtx, "new-jersey-hub", "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("availability: %v", parseErr2)
	}
	if parseAvailability.Product.SKU != "frame-desk" || parseAvailability.Warehouse.ID != "new-jersey-hub" {
		parseT.Fatalf("unexpected availability payload: %#v", parseAvailability)
	}

	parseInventoryRows, parseErr2 := store.InventoryList(parseCtx, repository.InventoryQuery{
		Warehouse:   "new-jersey-hub",
		StockHealth: "promise_risk",
		Search:      "frame",
		SortKey:     "inbound",
	})
	if parseErr2 != nil {
		parseT.Fatalf("inventory list: %v", parseErr2)
	}
	if len(parseInventoryRows) == 0 {
		parseT.Fatalf("expected inventory rows")
	}
	for _, parseRow := range parseInventoryRows {
		if parseRow.MarketPressure == "" || parseRow.MarketSignal == "" {
			parseT.Fatalf("inventory enrichment missing: %#v", parseRow)
		}
	}

	parseBestInventory, parseErr2 := store.InventoryBySKU(parseCtx, "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("inventory by sku: %v", parseErr2)
	}
	if parseBestInventory.SKU != "frame-desk" || parseBestInventory.WarehouseID == "" {
		parseT.Fatalf("unexpected best inventory row: %#v", parseBestInventory)
	}

	parseAllRows, parseErr2 := store.InventoryRowsBySKU(parseCtx, "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("inventory rows by sku: %v", parseErr2)
	}
	if len(parseAllRows) < 2 {
		parseT.Fatalf("expected multiple warehouse rows, got %d", len(parseAllRows))
	}
	_, parseErr2 = store.InventoryRowsBySKU(parseCtx, "missing-sku")
	if !errors.Is(parseErr2, sql.ErrNoRows) {
		parseT.Fatalf("expected sql.ErrNoRows for missing sku, got %v", parseErr2)
	}

	parsePref, parseErr2 := store.PreferencesByOwner(parseCtx, "demo-operator")
	if parseErr2 != nil {
		parseT.Fatalf("preferences existing owner: %v", parseErr2)
	}
	if parsePref.OwnerID != "demo-operator" {
		parseT.Fatalf("unexpected preferences owner: %s", parsePref.OwnerID)
	}
	parseDefaultPref, parseErr2 := store.PreferencesByOwner(parseCtx, "missing-owner")
	if parseErr2 != nil {
		parseT.Fatalf("preferences default owner: %v", parseErr2)
	}
	if parseDefaultPref.Theme == "" || parseDefaultPref.DefaultWarehouseID == "" {
		parseT.Fatalf("expected default preference fallback values: %#v", parseDefaultPref)
	}

	parseViews, parseErr2 := store.SavedViewsByOwner(parseCtx, "demo-operator")
	if parseErr2 != nil {
		parseT.Fatalf("saved views: %v", parseErr2)
	}
	if len(parseViews) == 0 {
		parseT.Fatalf("expected seeded saved views")
	}

	parseProductComments, parseErr2 := store.ProductComments(parseCtx, "studio-console", "approved")
	if parseErr2 != nil {
		parseT.Fatalf("product comments: %v", parseErr2)
	}
	if len(parseProductComments) == 0 {
		parseT.Fatalf("expected approved comments for studio-console")
	}

	parseRelated, parseErr2 := store.RelatedProducts(parseCtx, "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("related products: %v", parseErr2)
	}
	if len(parseRelated) == 0 {
		parseT.Fatalf("expected related products")
	}
	if parseRelated[0].Reason == "" || parseRelated[0].WarehouseID == "" {
		parseT.Fatalf("related product missing derived fields: %#v", parseRelated[0])
	}

	parsePressureList, parseErr2 := store.WarehousePressure(parseCtx)
	if parseErr2 != nil {
		parseT.Fatalf("warehouse pressure: %v", parseErr2)
	}
	if len(parsePressureList) != len(parseWarehouses) {
		parseT.Fatalf("expected pressure for each warehouse, got %d for %d warehouses", len(parsePressureList), len(parseWarehouses))
	}
	parsePressureByID, parseErr2 := store.WarehousePressureByID(parseCtx, "illinois-hub")
	if parseErr2 != nil {
		parseT.Fatalf("warehouse pressure by id: %v", parseErr2)
	}
	if parsePressureByID.ID != "illinois-hub" {
		parseT.Fatalf("unexpected warehouse pressure id: %s", parsePressureByID.ID)
	}

	parseHistory, parseErr2 := store.ThresholdHistory(parseCtx, "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("threshold history: %v", parseErr2)
	}
	if len(parseHistory) == 0 {
		parseT.Fatalf("expected threshold history rows")
	}

	parseRecommendations, parseErr2 := store.TransferRecommendations(parseCtx, "frame-desk")
	if parseErr2 != nil {
		parseT.Fatalf("transfer recommendations: %v", parseErr2)
	}
	if len(parseRecommendations) > 0 {
		if parseRecommendations[0].SourceWarehouseID == "" || parseRecommendations[0].DestinationWarehouseID == "" {
			parseT.Fatalf("invalid transfer recommendation: %#v", parseRecommendations[0])
		}
	}

	parseTransfer, parseErr2 := store.TransferByID(parseCtx, "tr-seed-001")
	if parseErr2 != nil {
		parseT.Fatalf("transfer by id: %v", parseErr2)
	}
	parseLines, parseErr2 := store.TransferLines(parseCtx, parseTransfer.ID)
	if parseErr2 != nil {
		parseT.Fatalf("transfer lines: %v", parseErr2)
	}
	if len(parseLines) == 0 {
		parseT.Fatalf("expected transfer lines")
	}
	parseTransferDetail, parseErr2 := store.TransferDetail(parseCtx, parseTransfer.ID)
	if parseErr2 != nil {
		parseT.Fatalf("transfer detail: %v", parseErr2)
	}
	if len(parseTransferDetail.Lines) == 0 {
		parseT.Fatalf("expected transfer detail lines")
	}

	parseReceivingLines, parseErr2 := store.ReceivingLines(parseCtx, "rcv-illinois-001")
	if parseErr2 != nil {
		parseT.Fatalf("receiving lines: %v", parseErr2)
	}
	if len(parseReceivingLines) == 0 {
		parseT.Fatalf("expected receiving lines")
	}
	parseReceivingDetail, parseErr2 := store.ReceivingDetail(parseCtx, "rcv-illinois-001")
	if parseErr2 != nil {
		parseT.Fatalf("receiving detail: %v", parseErr2)
	}
	if parseReceivingDetail.Session.ID != "rcv-illinois-001" {
		parseT.Fatalf("unexpected receiving detail session id: %s", parseReceivingDetail.Session.ID)
	}

	parsePurchaseOrders, parseErr2 := store.PurchaseOrders(parseCtx)
	if parseErr2 != nil {
		parseT.Fatalf("purchase orders: %v", parseErr2)
	}
	if len(parsePurchaseOrders) == 0 {
		parseT.Fatalf("expected purchase orders")
	}
	parseByWarehouse, parseErr2 := store.PurchaseOrdersByWarehouse(parseCtx, "illinois-hub")
	if parseErr2 != nil {
		parseT.Fatalf("purchase orders by warehouse: %v", parseErr2)
	}
	if len(parseByWarehouse) == 0 {
		parseT.Fatalf("expected purchase orders in illinois-hub")
	}
	parseOrder, parseErr2 := store.PurchaseOrderByID(parseCtx, "po-1042")
	if parseErr2 != nil {
		parseT.Fatalf("purchase order by id: %v", parseErr2)
	}
	parseOrderLines, parseErr2 := store.PurchaseOrderLines(parseCtx, parseOrder.ID)
	if parseErr2 != nil {
		parseT.Fatalf("purchase order lines: %v", parseErr2)
	}
	if len(parseOrderLines) == 0 {
		parseT.Fatalf("expected purchase order lines")
	}
	parseOrderDetail, parseErr2 := store.PurchaseOrderDetail(parseCtx, parseOrder.ID)
	if parseErr2 != nil {
		parseT.Fatalf("purchase order detail: %v", parseErr2)
	}
	if len(parseOrderDetail.Lines) == 0 {
		parseT.Fatalf("expected purchase order detail lines")
	}
}

func TestStoreAdminAndMutationFlows(parseT *testing.T) {
	parseT.Parallel()

	parseCtx, store, _ := newSeededStore(parseT)

	parseCreatedProduct, parseErr := store.CreateProduct(parseCtx, CreateProductInput{
		SKU:        "  integration-widget  ",
		Slug:       " integration-widget ",
		Status:     "low_stock",
		PriceCents: -100,
		Available:  -5,
		Inbound:    -2,
	})
	if parseErr != nil {
		parseT.Fatalf("create product: %v", parseErr)
	}
	if parseCreatedProduct.SKU != "integration-widget" || parseCreatedProduct.PriceCents != 0 {
		parseT.Fatalf("unexpected created product normalization: %#v", parseCreatedProduct)
	}

	parseAdminList, parseErr := store.ProductAdminList(parseCtx, ProductAdminQuery{
		Search: "integration-widget",
		Status: "all",
		Sort:   "price",
	})
	if parseErr != nil {
		parseT.Fatalf("product admin list: %v", parseErr)
	}
	if len(parseAdminList) == 0 {
		parseT.Fatalf("expected product in admin list")
	}
	parseAdminBySlug, parseErr := store.ProductAdminBySlug(parseCtx, parseCreatedProduct.Slug)
	if parseErr != nil {
		parseT.Fatalf("product admin by slug: %v", parseErr)
	}
	if parseAdminBySlug.SKU != parseCreatedProduct.SKU {
		parseT.Fatalf("unexpected admin by slug product: %#v", parseAdminBySlug)
	}

	parseUpdatedProduct, parseErr := store.UpdateProduct(parseCtx, parseCreatedProduct.Slug, UpdateProductInput{
		Slug:             "integration-widget-v2",
		Title:            "Integration Widget v2",
		Category:         "accessories",
		CurrentWarehouse: "new-jersey-hub",
		WarehouseID:      "illinois-hub",
		PriceCents:       -1,
		Available:        -1,
		Inbound:          -1,
	})
	if parseErr != nil {
		parseT.Fatalf("update product: %v", parseErr)
	}
	if parseUpdatedProduct.Slug != "integration-widget-v2" || parseUpdatedProduct.WarehouseID != "illinois-hub" {
		parseT.Fatalf("unexpected updated product payload: %#v", parseUpdatedProduct)
	}

	if parseErr2 := store.DeleteProduct(parseCtx, parseUpdatedProduct.Slug); parseErr2 != nil {
		parseT.Fatalf("delete product: %v", parseErr2)
	}
	if parseErr3 := store.DeleteProduct(parseCtx, "missing-product"); !errors.Is(parseErr3, sql.ErrNoRows) {
		parseT.Fatalf("expected sql.ErrNoRows for missing product, got %v", parseErr3)
	}

	_, parseErr = store.CreateComment(parseCtx, CreateCommentInput{
		ProductSKU: "frame-desk",
		Body:       "   ",
	})
	if parseErr == nil {
		parseT.Fatalf("expected create comment validation error")
	}
	parseCreatedComment, parseErr := store.CreateComment(parseCtx, CreateCommentInput{
		ProductSKU: "frame-desk",
		AuthorName: "Integration Tester",
		Reaction:   "thumbs_down",
		Subject:    "Coverage",
		Body:       "Integration moderation flow check.",
	})
	if parseErr != nil {
		parseT.Fatalf("create comment success: %v", parseErr)
	}
	if parseCreatedComment.Reaction != "down" || parseCreatedComment.Status != "pending" {
		parseT.Fatalf("unexpected created comment fields: %#v", parseCreatedComment)
	}
	parseModeratedComment, parseErr := store.ModerateComment(parseCtx, parseCreatedComment.ID, "", "")
	if parseErr != nil {
		parseT.Fatalf("moderate comment: %v", parseErr)
	}
	if parseModeratedComment.Status != "approved" {
		parseT.Fatalf("expected default approved moderation status, got %s", parseModeratedComment.Status)
	}

	parseModeratedBatch, parseErr := store.ModerateComments(parseCtx, []string{parseCreatedComment.ID, "cmt-seed-studio-console-flagged", " "}, "rejected", "triage")
	if parseErr != nil {
		parseT.Fatalf("moderate comments: %v", parseErr)
	}
	if len(parseModeratedBatch) != 2 {
		parseT.Fatalf("expected 2 moderated comments, got %d", len(parseModeratedBatch))
	}
	parseRejectedComments, parseErr := store.Comments(parseCtx, "rejected")
	if parseErr != nil {
		parseT.Fatalf("list rejected comments: %v", parseErr)
	}
	if len(parseRejectedComments) == 0 {
		parseT.Fatalf("expected rejected comments after moderation")
	}

	_, parseErr = store.CreateQuoteRequest(parseCtx, CreateQuoteRequestInput{
		ProductSKU: "frame-desk",
		Email:      "",
	})
	if parseErr == nil {
		parseT.Fatalf("expected quote request validation error")
	}
	parseQuote, parseErr := store.CreateQuoteRequest(parseCtx, CreateQuoteRequestInput{
		ProductSKU:    "frame-desk",
		RequesterName: "Ops Lead",
		CompanyName:   "Atlas QA",
		Email:         "qa@example.com",
		Quantity:      0,
		Note:          "Need rapid quote.",
	})
	if parseErr != nil {
		parseT.Fatalf("create quote request: %v", parseErr)
	}
	if parseQuote.Quantity != 1 {
		parseT.Fatalf("expected quote quantity normalization to 1, got %d", parseQuote.Quantity)
	}

	_, parseErr = store.CreateRestockRequest(parseCtx, CreateRestockRequestInput{
		ProductSKU: "frame-desk",
		Email:      "",
	})
	if parseErr == nil {
		parseT.Fatalf("expected restock request validation error")
	}
	parseRestock, parseErr := store.CreateRestockRequest(parseCtx, CreateRestockRequestInput{
		ProductSKU:           "frame-desk",
		Email:                "notify@example.com",
		PreferredWarehouseID: "new-jersey-hub",
	})
	if parseErr != nil {
		parseT.Fatalf("create restock request: %v", parseErr)
	}
	if parseRestock.Email == "" || parseRestock.ID == "" {
		parseT.Fatalf("unexpected restock request payload: %#v", parseRestock)
	}

	parseUpdatedPrefs, parseErr := store.SavePreferences(parseCtx, PreferencesRecord{
		OwnerID: "ops-owner",
		Theme:   "",
	})
	if parseErr != nil {
		parseT.Fatalf("save preferences: %v", parseErr)
	}
	if parseUpdatedPrefs.Theme == "" || parseUpdatedPrefs.Locale == "" {
		parseT.Fatalf("expected normalized preferences: %#v", parseUpdatedPrefs)
	}
	parseSavedView, parseErr := store.SaveView(parseCtx, SaveViewInput{
		OwnerID: "ops-owner",
		Name:    "Ops critical queue",
		Scope:   "",
	})
	if parseErr != nil {
		parseT.Fatalf("save view: %v", parseErr)
	}
	if parseSavedView.ID == "" || parseSavedView.Scope == "" {
		parseT.Fatalf("unexpected saved view payload: %#v", parseSavedView)
	}
	parseImportedViews, parseErr := store.ImportSavedViews(parseCtx, "ops-owner", []repository.SavedView{
		{Name: "Imported A", Scope: "inventory", FiltersJSON: `{"status":"risk"}`},
	})
	if parseErr != nil {
		parseT.Fatalf("import saved views: %v", parseErr)
	}
	if len(parseImportedViews) != 1 {
		parseT.Fatalf("expected one imported view, got %d", len(parseImportedViews))
	}
	parseOwnerViews, parseErr := store.SavedViewsByOwner(parseCtx, "ops-owner")
	if parseErr != nil {
		parseT.Fatalf("saved views by owner after import: %v", parseErr)
	}
	if len(parseOwnerViews) < 2 {
		parseT.Fatalf("expected multiple owner views after save/import, got %d", len(parseOwnerViews))
	}

	parseExistingTransfers, parseErr := store.Transfers(parseCtx)
	if parseErr != nil {
		parseT.Fatalf("list transfers: %v", parseErr)
	}
	parseCreatedTransfer, parseErr := store.CreateTransfer(parseCtx, CreateTransferInput{})
	if parseErr != nil {
		parseT.Fatalf("create transfer: %v", parseErr)
	}
	if parseCreatedTransfer.Status != "draft" {
		parseT.Fatalf("unexpected created transfer status: %s", parseCreatedTransfer.Status)
	}
	parseUpdatedTransfers, parseErr := store.Transfers(parseCtx)
	if parseErr != nil {
		parseT.Fatalf("list transfers after create: %v", parseErr)
	}
	if len(parseUpdatedTransfers) < len(parseExistingTransfers)+1 {
		parseT.Fatalf("expected transfer count increase, before=%d after=%d", len(parseExistingTransfers), len(parseUpdatedTransfers))
	}

	parseReceivingSessions, parseErr := store.ReceivingSessions(parseCtx)
	if parseErr != nil {
		parseT.Fatalf("receiving sessions: %v", parseErr)
	}
	if len(parseReceivingSessions) == 0 {
		parseT.Fatalf("expected receiving sessions")
	}
	parseReconciled, parseErr := store.ReconcileReceiving(parseCtx, "rcv-illinois-001", ReconcileReceivingInput{
		Status:             "",
		DiscrepancySummary: "Resolved in integration test",
	})
	if parseErr != nil {
		parseT.Fatalf("reconcile receiving: %v", parseErr)
	}
	if parseReconciled.Status != "closed" {
		parseT.Fatalf("expected reconcile default closed status, got %s", parseReconciled.Status)
	}

	parseOrderStatusUpdated, parseErr := store.UpdatePurchaseOrderStatus(parseCtx, "po-1042", UpdatePurchaseOrderStatusInput{
		Status: "approved",
	})
	if parseErr != nil {
		parseT.Fatalf("update purchase order status: %v", parseErr)
	}
	if parseOrderStatusUpdated.Status != "approved" {
		parseT.Fatalf("unexpected purchase order status: %s", parseOrderStatusUpdated.Status)
	}

	_, parseErr = store.UpdateInventoryLevel(parseCtx, "frame-desk", UpdateInventoryLevelInput{})
	if parseErr == nil {
		parseT.Fatalf("expected update inventory level warehouse validation error")
	}
	parseUpdatedInventory, parseErr := store.UpdateInventoryLevel(parseCtx, "frame-desk", UpdateInventoryLevelInput{
		WarehouseID:  "new-jersey-hub",
		OnHand:       -2,
		Reserved:     -1,
		Inbound:      -3,
		Damaged:      -4,
		ReorderPoint: 0,
		SafetyStock:  -2,
		Status:       "",
	})
	if parseErr != nil {
		parseT.Fatalf("update inventory level: %v", parseErr)
	}
	if parseUpdatedInventory.Available != 0 || parseUpdatedInventory.ReorderPoint != 1 || parseUpdatedInventory.Status != "balanced" {
		parseT.Fatalf("unexpected normalized inventory row: %#v", parseUpdatedInventory)
	}

	parseBeforeHistory, parseErr := store.ThresholdHistory(parseCtx, "frame-desk")
	if parseErr != nil {
		parseT.Fatalf("threshold history before update: %v", parseErr)
	}
	_, parseErr = store.UpdateThreshold(parseCtx, "frame-desk", "new-jersey-hub", 0, -1)
	if parseErr != nil {
		parseT.Fatalf("update threshold: %v", parseErr)
	}
	parseAfterHistory, parseErr := store.ThresholdHistory(parseCtx, "frame-desk")
	if parseErr != nil {
		parseT.Fatalf("threshold history after update: %v", parseErr)
	}
	if len(parseAfterHistory) < len(parseBeforeHistory)+1 {
		parseT.Fatalf("expected threshold history growth, before=%d after=%d", len(parseBeforeHistory), len(parseAfterHistory))
	}

	_, parseErr = store.CreatePurchaseOrder(parseCtx, CreatePurchaseOrderInput{
		WarehouseID: "new-jersey-hub",
		ProductSKU:  "",
	})
	if parseErr == nil {
		parseT.Fatalf("expected create purchase order validation error")
	}
	parseBeforeInbound, parseErr := store.inventoryLevelBySKUWarehouse(parseCtx, "frame-desk", "new-jersey-hub")
	if parseErr != nil {
		parseT.Fatalf("inventory level before purchase order: %v", parseErr)
	}
	parseCreatedPO, parseErr := store.CreatePurchaseOrder(parseCtx, CreatePurchaseOrderInput{
		WarehouseID: "new-jersey-hub",
		ProductSKU:  "frame-desk",
		Quantity:    0,
		Status:      "",
	})
	if parseErr != nil {
		parseT.Fatalf("create purchase order: %v", parseErr)
	}
	if parseCreatedPO.Order.ID == "" || len(parseCreatedPO.Lines) != 1 {
		parseT.Fatalf("unexpected purchase order payload: %#v", parseCreatedPO)
	}
	parseAfterInbound, parseErr := store.inventoryLevelBySKUWarehouse(parseCtx, "frame-desk", "new-jersey-hub")
	if parseErr != nil {
		parseT.Fatalf("inventory level after purchase order: %v", parseErr)
	}
	if parseAfterInbound.Inbound <= parseBeforeInbound.Inbound {
		parseT.Fatalf("expected inbound increase after purchase order, before=%d after=%d", parseBeforeInbound.Inbound, parseAfterInbound.Inbound)
	}
}

func TestPureHelperFunctions(parseT *testing.T) {
	parseT.Parallel()

	if parseGot := normalizeCommentReaction("thumbs-down"); parseGot != "down" {
		parseT.Fatalf("normalizeCommentReaction thumbs-down=%s", parseGot)
	}
	if parseGot2 := normalizeCommentReaction("something-else"); parseGot2 != "up" {
		parseT.Fatalf("normalizeCommentReaction default=%s", parseGot2)
	}

	if parseGot3 := nonEmptyString("  ", "fallback"); parseGot3 != "fallback" {
		parseT.Fatalf("nonEmptyString fallback=%s", parseGot3)
	}
	if parseGot4 := nonEmptyString(" value ", "fallback"); parseGot4 != "value" {
		parseT.Fatalf("nonEmptyString trim=%s", parseGot4)
	}

	if parseGot5 := nullableString("  "); parseGot5.(sql.NullString).Valid {
		parseT.Fatalf("nullableString empty should be invalid null string")
	}
	if parseGot6 := nullableString("x"); parseGot6.(string) != "x" {
		parseT.Fatalf("nullableString value=%v", parseGot6)
	}

	if parseGot7 := clampInt(2, 3, 5); parseGot7 != 3 {
		parseT.Fatalf("clampInt low=%d", parseGot7)
	}
	if parseGot8 := clampInt(6, 3, 5); parseGot8 != 5 {
		parseT.Fatalf("clampInt high=%d", parseGot8)
	}
	if parseGot9 := clampInt(4, 3, 5); parseGot9 != 4 {
		parseT.Fatalf("clampInt middle=%d", parseGot9)
	}
	if parseGot10 := maxInt(8, 3); parseGot10 != 8 {
		parseT.Fatalf("maxInt expected 8 got %d", parseGot10)
	}

	if parseGot11 := inventoryWeeklyUnits("frame-desk", "new-jersey-hub"); parseGot11 <= 0 {
		parseT.Fatalf("inventoryWeeklyUnits unexpected %d", parseGot11)
	}
	if parseGot12 := inventoryWeeklyUnits("unknown", "unknown"); parseGot12 != 4 {
		parseT.Fatalf("inventoryWeeklyUnits default expected 4 got %d", parseGot12)
	}
	if parseGot13 := inventoryRegionalShare("new-jersey-hub"); parseGot13 != 42 {
		parseT.Fatalf("inventoryRegionalShare new jersey %d", parseGot13)
	}
	if parseGot14 := inventoryRegionalShare("unknown"); parseGot14 != 20 {
		parseT.Fatalf("inventoryRegionalShare default %d", parseGot14)
	}

	if parseGot15 := inventoryMarketPressure(82, 4, 0); parseGot15 != "Hot market" {
		parseT.Fatalf("inventoryMarketPressure hot=%s", parseGot15)
	}
	if parseGot16 := inventoryMarketPressure(70, 9, 12); parseGot16 != "Growing demand" {
		parseT.Fatalf("inventoryMarketPressure growth=%s", parseGot16)
	}
	if parseGot17 := inventoryMarketPressure(50, 20, 0); parseGot17 != "Softening" {
		parseT.Fatalf("inventoryMarketPressure softening=%s", parseGot17)
	}
	if parseGot18 := inventoryMarketPressure(50, 10, 2); parseGot18 != "Stable" {
		parseT.Fatalf("inventoryMarketPressure stable=%s", parseGot18)
	}

	if parseGot19 := inventoryMarketSignal("accessories", "new-jersey-hub", 5); parseGot19 == "" {
		parseT.Fatalf("inventoryMarketSignal accessories empty")
	}
	if parseGot20 := inventoryMarketSignal("desks", "nevada-hub", 5); parseGot20 == "" {
		parseT.Fatalf("inventoryMarketSignal desks empty")
	}
	if parseGot21 := inventoryMarketSignal("other", "other", 12); parseGot21 == "" {
		parseT.Fatalf("inventoryMarketSignal high velocity empty")
	}
	if parseGot22 := inventoryMarketSignal("other", "other", 2); parseGot22 == "" {
		parseT.Fatalf("inventoryMarketSignal default empty")
	}

	if parseGot23 := inventoryStatusForProduct("low_stock", 20); parseGot23 != "promise_risk" {
		parseT.Fatalf("inventoryStatusForProduct low stock=%s", parseGot23)
	}
	if parseGot24 := inventoryStatusForProduct("in_stock", 0); parseGot24 != "promise_risk" {
		parseT.Fatalf("inventoryStatusForProduct zero=%s", parseGot24)
	}
	if parseGot25 := inventoryStatusForProduct("in_stock", 10); parseGot25 != "balanced" {
		parseT.Fatalf("inventoryStatusForProduct balanced=%s", parseGot25)
	}

	parseCreatedInput := prepareCreateProductInput(CreateProductInput{
		SKU:        "  sku-1  ",
		Slug:       " slug-1 ",
		PriceCents: -1,
		Available:  -2,
		Inbound:    -3,
	})
	if parseCreatedInput.SKU != "sku-1" || parseCreatedInput.Slug != "slug-1" {
		parseT.Fatalf("prepareCreateProductInput trim failed: %#v", parseCreatedInput)
	}
	if parseCreatedInput.PriceCents != 0 || parseCreatedInput.Available != 0 || parseCreatedInput.Inbound != 0 {
		parseT.Fatalf("prepareCreateProductInput numeric normalization failed: %#v", parseCreatedInput)
	}
	if parseCreatedInput.Title == "" || parseCreatedInput.WarehouseID == "" {
		parseT.Fatalf("prepareCreateProductInput defaults missing: %#v", parseCreatedInput)
	}

	parseCurrent := ProductAdminRecord{
		SKU:            "sku",
		Slug:           "slug",
		Title:          "title",
		Category:       "category",
		PriceCents:     900,
		Status:         "in_stock",
		Finish:         "finish",
		Summary:        "summary",
		Details:        "details",
		SEOTitle:       "seo title",
		SEODescription: "seo desc",
		WarehouseID:    "new-jersey-hub",
		Available:      4,
		Inbound:        2,
	}
	parseUpdatedInput := prepareUpdateProductInput(parseCurrent, UpdateProductInput{
		PriceCents: -1,
		Available:  -1,
		Inbound:    -1,
	})
	if parseUpdatedInput.SKU != "sku" || parseUpdatedInput.PriceCents != 900 {
		parseT.Fatalf("prepareUpdateProductInput normalization failed: %#v", parseUpdatedInput)
	}
	if parseUpdatedInput.Available != 4 || parseUpdatedInput.Inbound != 2 {
		parseT.Fatalf("prepareUpdateProductInput quantity fallback failed: %#v", parseUpdatedInput)
	}

	parseItems := []ProductAdminRecord{
		{Title: "B", PriceCents: 100, Volume: 1, Status: "in_stock", UpdatedAt: "1"},
		{Title: "A", PriceCents: 1000, Volume: 10, Status: "flagged", UpdatedAt: "2"},
	}
	sortProductAdminItems(parseItems, "price")
	if parseItems[0].Title != "A" {
		parseT.Fatalf("sort by price failed: %#v", parseItems)
	}
	sortProductAdminItems(parseItems, "volume")
	if parseItems[0].Title != "A" {
		parseT.Fatalf("sort by volume failed: %#v", parseItems)
	}
	sortProductAdminItems(parseItems, "status")
	if !sort.SliceIsSorted(parseItems, func(parseI, parseJ int) bool { return parseItems[parseI].Status <= parseItems[parseJ].Status }) {
		parseT.Fatalf("sort by status failed: %#v", parseItems)
	}
	sortProductAdminItems(parseItems, "updated")
	if parseItems[0].UpdatedAt != "2" {
		parseT.Fatalf("sort by default updated failed: %#v", parseItems)
	}

	parseBase := repository.Product{Category: "desks"}
	if parseReason := relatedProductReason(parseBase, repository.Product{Category: "desks"}); parseReason == "" {
		parseT.Fatalf("relatedProductReason same-category empty")
	}
	if parseReason2 := relatedProductReason(parseBase, repository.Product{Category: "storage"}); parseReason2 == "" {
		parseT.Fatalf("relatedProductReason cross-category empty")
	}

	if parsePressure, _, _, _ := warehouseOperationalProfile("new-jersey-hub"); parsePressure == "" {
		parseT.Fatalf("warehouseOperationalProfile new jersey empty")
	}
	if parsePressure2, _, _, _ := warehouseOperationalProfile("nevada-hub"); parsePressure2 == "" {
		parseT.Fatalf("warehouseOperationalProfile nevada empty")
	}
	if parsePressure3, _, _, _ := warehouseOperationalProfile("unknown"); parsePressure3 == "" {
		parseT.Fatalf("warehouseOperationalProfile default empty")
	}
}

func newSeededStore(parseT *testing.T) (context.Context, *Store, *sql.DB) {
	parseT.Helper()

	parseCtx := context.Background()
	parseDatabasePath := filepath.Join(parseT.TempDir(), "atlas.db")
	parseDatabase, parseErr := Open(parseCtx, parseDatabasePath)
	if parseErr != nil {
		parseT.Fatalf("open sqlite: %v", parseErr)
	}
	parseT.Cleanup(func() {
		_ = parseDatabase.Close()
	})

	parseMigrationsPath := resolveDataPath(parseT, "migrations")
	parseFallbackSchemaPath := resolveDataPath(parseT, "schema.sql")
	if parseErr2 := Migrate(parseCtx, parseDatabase, parseMigrationsPath, parseFallbackSchemaPath); parseErr2 != nil {
		parseT.Fatalf("migrate database: %v", parseErr2)
	}
	if parseErr3 := Seed(parseCtx, parseDatabase); parseErr3 != nil {
		parseT.Fatalf("seed database: %v", parseErr3)
	}
	return parseCtx, NewStore(parseDatabase), parseDatabase
}

func resolveDataPath(parseT *testing.T, parseParts ...string) string {
	parseT.Helper()

	parseCandidates := []string{
		filepath.Join(append([]string{"..", "data"}, parseParts...)...),
		filepath.Join(append([]string{"examples", "86-atlas-commerce-os", "server", "data"}, parseParts...)...),
	}
	for _, parseCandidate := range parseCandidates {
		if _, parseErr := os.Stat(parseCandidate); parseErr == nil {
			return parseCandidate
		}
	}
	parseT.Fatalf("unable to resolve data path for %v", parseParts)
	return ""
}
