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

func TestLoadMigrationsAndMigrateFallback(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "002_b.sql"), []byte("create table if not exists b(id text);"), 0o644); err != nil {
		t.Fatalf("write migration b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_a.sql"), []byte("create table if not exists a(id text);"), 0o644); err != nil {
		t.Fatalf("write migration a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "README.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write migration noise: %v", err)
	}

	fallback := filepath.Join(tempDir, "schema.sql")
	if err := os.WriteFile(fallback, []byte("create table if not exists fallback_only(id text);"), 0o644); err != nil {
		t.Fatalf("write fallback schema: %v", err)
	}

	loaded, err := loadMigrations(migrationsDir, fallback)
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(loaded))
	}
	if loaded[0].Version != "001_a.sql" || loaded[1].Version != "002_b.sql" {
		t.Fatalf("migrations not sorted: %#v", loaded)
	}

	fallbackOnlyDir := filepath.Join(tempDir, "missing-migrations")
	loadedFallback, err := loadMigrations(fallbackOnlyDir, fallback)
	if err != nil {
		t.Fatalf("load fallback migrations: %v", err)
	}
	if len(loadedFallback) != 1 || loadedFallback[0].Version != "001_initial_schema.sql" {
		t.Fatalf("unexpected fallback migration payload: %#v", loadedFallback)
	}

	_, err = loadMigrations(fallbackOnlyDir, filepath.Join(tempDir, "missing-schema.sql"))
	if err == nil {
		t.Fatalf("expected error when no migrations and no fallback schema")
	}

	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(tempDir, "atlas.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	if err := Migrate(ctx, database, fallbackOnlyDir, fallback); err != nil {
		t.Fatalf("migrate with fallback: %v", err)
	}
	if err := Migrate(ctx, database, fallbackOnlyDir, fallback); err != nil {
		t.Fatalf("migrate idempotency: %v", err)
	}

	var migrationCount int
	if err := database.QueryRowContext(ctx, `select count(*) from schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count schema migrations: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("expected one applied migration, got %d", migrationCount)
	}
}

func TestStoreReadFlows(t *testing.T) {
	t.Parallel()

	ctx, store, database := newSeededStore(t)
	if err := Seed(ctx, database); err != nil {
		t.Fatalf("seed second pass: %v", err)
	}

	catalog, err := store.Catalog(ctx, CatalogQuery{
		Search:    "frame",
		Category:  "desks",
		Warehouse: "new-jersey-hub",
		Sort:      "warehouse",
	})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if catalog.Page != 1 || catalog.PageSize != 12 {
		t.Fatalf("catalog paging defaults not applied: %#v", catalog.Query)
	}
	if catalog.Total < 1 || len(catalog.Items) < 1 {
		t.Fatalf("expected catalog rows, got total=%d items=%d", catalog.Total, len(catalog.Items))
	}

	product, err := store.ProductBySlug(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("product by slug: %v", err)
	}
	if product.SKU != "frame-desk" {
		t.Fatalf("unexpected product sku: %s", product.SKU)
	}

	warehouses, err := store.Warehouses(ctx)
	if err != nil {
		t.Fatalf("warehouses: %v", err)
	}
	if len(warehouses) < 3 {
		t.Fatalf("expected seeded warehouses, got %d", len(warehouses))
	}
	warehouse, err := store.WarehouseBySlug(ctx, "new-jersey-hub")
	if err != nil {
		t.Fatalf("warehouse by slug: %v", err)
	}
	if warehouse.ID != "new-jersey-hub" {
		t.Fatalf("unexpected warehouse id: %s", warehouse.ID)
	}

	availability, err := store.Availability(ctx, "new-jersey-hub", "frame-desk")
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if availability.Product.SKU != "frame-desk" || availability.Warehouse.ID != "new-jersey-hub" {
		t.Fatalf("unexpected availability payload: %#v", availability)
	}

	inventoryRows, err := store.InventoryList(ctx, repository.InventoryQuery{
		Warehouse:   "new-jersey-hub",
		StockHealth: "promise_risk",
		Search:      "frame",
		SortKey:     "inbound",
	})
	if err != nil {
		t.Fatalf("inventory list: %v", err)
	}
	if len(inventoryRows) == 0 {
		t.Fatalf("expected inventory rows")
	}
	for _, row := range inventoryRows {
		if row.MarketPressure == "" || row.MarketSignal == "" {
			t.Fatalf("inventory enrichment missing: %#v", row)
		}
	}

	bestInventory, err := store.InventoryBySKU(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("inventory by sku: %v", err)
	}
	if bestInventory.SKU != "frame-desk" || bestInventory.WarehouseID == "" {
		t.Fatalf("unexpected best inventory row: %#v", bestInventory)
	}

	allRows, err := store.InventoryRowsBySKU(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("inventory rows by sku: %v", err)
	}
	if len(allRows) < 2 {
		t.Fatalf("expected multiple warehouse rows, got %d", len(allRows))
	}
	_, err = store.InventoryRowsBySKU(ctx, "missing-sku")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for missing sku, got %v", err)
	}

	pref, err := store.PreferencesByOwner(ctx, "demo-operator")
	if err != nil {
		t.Fatalf("preferences existing owner: %v", err)
	}
	if pref.OwnerID != "demo-operator" {
		t.Fatalf("unexpected preferences owner: %s", pref.OwnerID)
	}
	defaultPref, err := store.PreferencesByOwner(ctx, "missing-owner")
	if err != nil {
		t.Fatalf("preferences default owner: %v", err)
	}
	if defaultPref.Theme == "" || defaultPref.DefaultWarehouseID == "" {
		t.Fatalf("expected default preference fallback values: %#v", defaultPref)
	}

	views, err := store.SavedViewsByOwner(ctx, "demo-operator")
	if err != nil {
		t.Fatalf("saved views: %v", err)
	}
	if len(views) == 0 {
		t.Fatalf("expected seeded saved views")
	}

	productComments, err := store.ProductComments(ctx, "studio-console", "approved")
	if err != nil {
		t.Fatalf("product comments: %v", err)
	}
	if len(productComments) == 0 {
		t.Fatalf("expected approved comments for studio-console")
	}

	related, err := store.RelatedProducts(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("related products: %v", err)
	}
	if len(related) == 0 {
		t.Fatalf("expected related products")
	}
	if related[0].Reason == "" || related[0].WarehouseID == "" {
		t.Fatalf("related product missing derived fields: %#v", related[0])
	}

	pressureList, err := store.WarehousePressure(ctx)
	if err != nil {
		t.Fatalf("warehouse pressure: %v", err)
	}
	if len(pressureList) != len(warehouses) {
		t.Fatalf("expected pressure for each warehouse, got %d for %d warehouses", len(pressureList), len(warehouses))
	}
	pressureByID, err := store.WarehousePressureByID(ctx, "illinois-hub")
	if err != nil {
		t.Fatalf("warehouse pressure by id: %v", err)
	}
	if pressureByID.ID != "illinois-hub" {
		t.Fatalf("unexpected warehouse pressure id: %s", pressureByID.ID)
	}

	history, err := store.ThresholdHistory(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("threshold history: %v", err)
	}
	if len(history) == 0 {
		t.Fatalf("expected threshold history rows")
	}

	recommendations, err := store.TransferRecommendations(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("transfer recommendations: %v", err)
	}
	if len(recommendations) > 0 {
		if recommendations[0].SourceWarehouseID == "" || recommendations[0].DestinationWarehouseID == "" {
			t.Fatalf("invalid transfer recommendation: %#v", recommendations[0])
		}
	}

	transfer, err := store.TransferByID(ctx, "tr-seed-001")
	if err != nil {
		t.Fatalf("transfer by id: %v", err)
	}
	lines, err := store.TransferLines(ctx, transfer.ID)
	if err != nil {
		t.Fatalf("transfer lines: %v", err)
	}
	if len(lines) == 0 {
		t.Fatalf("expected transfer lines")
	}
	transferDetail, err := store.TransferDetail(ctx, transfer.ID)
	if err != nil {
		t.Fatalf("transfer detail: %v", err)
	}
	if len(transferDetail.Lines) == 0 {
		t.Fatalf("expected transfer detail lines")
	}

	receivingLines, err := store.ReceivingLines(ctx, "rcv-illinois-001")
	if err != nil {
		t.Fatalf("receiving lines: %v", err)
	}
	if len(receivingLines) == 0 {
		t.Fatalf("expected receiving lines")
	}
	receivingDetail, err := store.ReceivingDetail(ctx, "rcv-illinois-001")
	if err != nil {
		t.Fatalf("receiving detail: %v", err)
	}
	if receivingDetail.Session.ID != "rcv-illinois-001" {
		t.Fatalf("unexpected receiving detail session id: %s", receivingDetail.Session.ID)
	}

	purchaseOrders, err := store.PurchaseOrders(ctx)
	if err != nil {
		t.Fatalf("purchase orders: %v", err)
	}
	if len(purchaseOrders) == 0 {
		t.Fatalf("expected purchase orders")
	}
	byWarehouse, err := store.PurchaseOrdersByWarehouse(ctx, "illinois-hub")
	if err != nil {
		t.Fatalf("purchase orders by warehouse: %v", err)
	}
	if len(byWarehouse) == 0 {
		t.Fatalf("expected purchase orders in illinois-hub")
	}
	order, err := store.PurchaseOrderByID(ctx, "po-1042")
	if err != nil {
		t.Fatalf("purchase order by id: %v", err)
	}
	orderLines, err := store.PurchaseOrderLines(ctx, order.ID)
	if err != nil {
		t.Fatalf("purchase order lines: %v", err)
	}
	if len(orderLines) == 0 {
		t.Fatalf("expected purchase order lines")
	}
	orderDetail, err := store.PurchaseOrderDetail(ctx, order.ID)
	if err != nil {
		t.Fatalf("purchase order detail: %v", err)
	}
	if len(orderDetail.Lines) == 0 {
		t.Fatalf("expected purchase order detail lines")
	}
}

func TestStoreAdminAndMutationFlows(t *testing.T) {
	t.Parallel()

	ctx, store, _ := newSeededStore(t)

	createdProduct, err := store.CreateProduct(ctx, CreateProductInput{
		SKU:        "  integration-widget  ",
		Slug:       " integration-widget ",
		Status:     "low_stock",
		PriceCents: -100,
		Available:  -5,
		Inbound:    -2,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if createdProduct.SKU != "integration-widget" || createdProduct.PriceCents != 0 {
		t.Fatalf("unexpected created product normalization: %#v", createdProduct)
	}

	adminList, err := store.ProductAdminList(ctx, ProductAdminQuery{
		Search: "integration-widget",
		Status: "all",
		Sort:   "price",
	})
	if err != nil {
		t.Fatalf("product admin list: %v", err)
	}
	if len(adminList) == 0 {
		t.Fatalf("expected product in admin list")
	}
	adminBySlug, err := store.ProductAdminBySlug(ctx, createdProduct.Slug)
	if err != nil {
		t.Fatalf("product admin by slug: %v", err)
	}
	if adminBySlug.SKU != createdProduct.SKU {
		t.Fatalf("unexpected admin by slug product: %#v", adminBySlug)
	}

	updatedProduct, err := store.UpdateProduct(ctx, createdProduct.Slug, UpdateProductInput{
		Slug:             "integration-widget-v2",
		Title:            "Integration Widget v2",
		Category:         "accessories",
		CurrentWarehouse: "new-jersey-hub",
		WarehouseID:      "illinois-hub",
		PriceCents:       -1,
		Available:        -1,
		Inbound:          -1,
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updatedProduct.Slug != "integration-widget-v2" || updatedProduct.WarehouseID != "illinois-hub" {
		t.Fatalf("unexpected updated product payload: %#v", updatedProduct)
	}

	if err := store.DeleteProduct(ctx, updatedProduct.Slug); err != nil {
		t.Fatalf("delete product: %v", err)
	}
	if err := store.DeleteProduct(ctx, "missing-product"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for missing product, got %v", err)
	}

	_, err = store.CreateComment(ctx, CreateCommentInput{
		ProductSKU: "frame-desk",
		Body:       "   ",
	})
	if err == nil {
		t.Fatalf("expected create comment validation error")
	}
	createdComment, err := store.CreateComment(ctx, CreateCommentInput{
		ProductSKU: "frame-desk",
		AuthorName: "Integration Tester",
		Reaction:   "thumbs_down",
		Subject:    "Coverage",
		Body:       "Integration moderation flow check.",
	})
	if err != nil {
		t.Fatalf("create comment success: %v", err)
	}
	if createdComment.Reaction != "down" || createdComment.Status != "pending" {
		t.Fatalf("unexpected created comment fields: %#v", createdComment)
	}
	moderatedComment, err := store.ModerateComment(ctx, createdComment.ID, "", "")
	if err != nil {
		t.Fatalf("moderate comment: %v", err)
	}
	if moderatedComment.Status != "approved" {
		t.Fatalf("expected default approved moderation status, got %s", moderatedComment.Status)
	}

	moderatedBatch, err := store.ModerateComments(ctx, []string{createdComment.ID, "cmt-seed-studio-console-flagged", " "}, "rejected", "triage")
	if err != nil {
		t.Fatalf("moderate comments: %v", err)
	}
	if len(moderatedBatch) != 2 {
		t.Fatalf("expected 2 moderated comments, got %d", len(moderatedBatch))
	}
	rejectedComments, err := store.Comments(ctx, "rejected")
	if err != nil {
		t.Fatalf("list rejected comments: %v", err)
	}
	if len(rejectedComments) == 0 {
		t.Fatalf("expected rejected comments after moderation")
	}

	_, err = store.CreateQuoteRequest(ctx, CreateQuoteRequestInput{
		ProductSKU: "frame-desk",
		Email:      "",
	})
	if err == nil {
		t.Fatalf("expected quote request validation error")
	}
	quote, err := store.CreateQuoteRequest(ctx, CreateQuoteRequestInput{
		ProductSKU:    "frame-desk",
		RequesterName: "Ops Lead",
		CompanyName:   "Atlas QA",
		Email:         "qa@example.com",
		Quantity:      0,
		Note:          "Need rapid quote.",
	})
	if err != nil {
		t.Fatalf("create quote request: %v", err)
	}
	if quote.Quantity != 1 {
		t.Fatalf("expected quote quantity normalization to 1, got %d", quote.Quantity)
	}

	_, err = store.CreateRestockRequest(ctx, CreateRestockRequestInput{
		ProductSKU: "frame-desk",
		Email:      "",
	})
	if err == nil {
		t.Fatalf("expected restock request validation error")
	}
	restock, err := store.CreateRestockRequest(ctx, CreateRestockRequestInput{
		ProductSKU:           "frame-desk",
		Email:                "notify@example.com",
		PreferredWarehouseID: "new-jersey-hub",
	})
	if err != nil {
		t.Fatalf("create restock request: %v", err)
	}
	if restock.Email == "" || restock.ID == "" {
		t.Fatalf("unexpected restock request payload: %#v", restock)
	}

	updatedPrefs, err := store.SavePreferences(ctx, PreferencesRecord{
		OwnerID: "ops-owner",
		Theme:   "",
	})
	if err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	if updatedPrefs.Theme == "" || updatedPrefs.Locale == "" {
		t.Fatalf("expected normalized preferences: %#v", updatedPrefs)
	}
	savedView, err := store.SaveView(ctx, SaveViewInput{
		OwnerID: "ops-owner",
		Name:    "Ops critical queue",
		Scope:   "",
	})
	if err != nil {
		t.Fatalf("save view: %v", err)
	}
	if savedView.ID == "" || savedView.Scope == "" {
		t.Fatalf("unexpected saved view payload: %#v", savedView)
	}
	importedViews, err := store.ImportSavedViews(ctx, "ops-owner", []repository.SavedView{
		{Name: "Imported A", Scope: "inventory", FiltersJSON: `{"status":"risk"}`},
	})
	if err != nil {
		t.Fatalf("import saved views: %v", err)
	}
	if len(importedViews) != 1 {
		t.Fatalf("expected one imported view, got %d", len(importedViews))
	}
	ownerViews, err := store.SavedViewsByOwner(ctx, "ops-owner")
	if err != nil {
		t.Fatalf("saved views by owner after import: %v", err)
	}
	if len(ownerViews) < 2 {
		t.Fatalf("expected multiple owner views after save/import, got %d", len(ownerViews))
	}

	existingTransfers, err := store.Transfers(ctx)
	if err != nil {
		t.Fatalf("list transfers: %v", err)
	}
	createdTransfer, err := store.CreateTransfer(ctx, CreateTransferInput{})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if createdTransfer.Status != "draft" {
		t.Fatalf("unexpected created transfer status: %s", createdTransfer.Status)
	}
	updatedTransfers, err := store.Transfers(ctx)
	if err != nil {
		t.Fatalf("list transfers after create: %v", err)
	}
	if len(updatedTransfers) < len(existingTransfers)+1 {
		t.Fatalf("expected transfer count increase, before=%d after=%d", len(existingTransfers), len(updatedTransfers))
	}

	receivingSessions, err := store.ReceivingSessions(ctx)
	if err != nil {
		t.Fatalf("receiving sessions: %v", err)
	}
	if len(receivingSessions) == 0 {
		t.Fatalf("expected receiving sessions")
	}
	reconciled, err := store.ReconcileReceiving(ctx, "rcv-illinois-001", ReconcileReceivingInput{
		Status:             "",
		DiscrepancySummary: "Resolved in integration test",
	})
	if err != nil {
		t.Fatalf("reconcile receiving: %v", err)
	}
	if reconciled.Status != "closed" {
		t.Fatalf("expected reconcile default closed status, got %s", reconciled.Status)
	}

	orderStatusUpdated, err := store.UpdatePurchaseOrderStatus(ctx, "po-1042", UpdatePurchaseOrderStatusInput{
		Status: "approved",
	})
	if err != nil {
		t.Fatalf("update purchase order status: %v", err)
	}
	if orderStatusUpdated.Status != "approved" {
		t.Fatalf("unexpected purchase order status: %s", orderStatusUpdated.Status)
	}

	_, err = store.UpdateInventoryLevel(ctx, "frame-desk", UpdateInventoryLevelInput{})
	if err == nil {
		t.Fatalf("expected update inventory level warehouse validation error")
	}
	updatedInventory, err := store.UpdateInventoryLevel(ctx, "frame-desk", UpdateInventoryLevelInput{
		WarehouseID:  "new-jersey-hub",
		OnHand:       -2,
		Reserved:     -1,
		Inbound:      -3,
		Damaged:      -4,
		ReorderPoint: 0,
		SafetyStock:  -2,
		Status:       "",
	})
	if err != nil {
		t.Fatalf("update inventory level: %v", err)
	}
	if updatedInventory.Available != 0 || updatedInventory.ReorderPoint != 1 || updatedInventory.Status != "balanced" {
		t.Fatalf("unexpected normalized inventory row: %#v", updatedInventory)
	}

	beforeHistory, err := store.ThresholdHistory(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("threshold history before update: %v", err)
	}
	_, err = store.UpdateThreshold(ctx, "frame-desk", "new-jersey-hub", 0, -1)
	if err != nil {
		t.Fatalf("update threshold: %v", err)
	}
	afterHistory, err := store.ThresholdHistory(ctx, "frame-desk")
	if err != nil {
		t.Fatalf("threshold history after update: %v", err)
	}
	if len(afterHistory) < len(beforeHistory)+1 {
		t.Fatalf("expected threshold history growth, before=%d after=%d", len(beforeHistory), len(afterHistory))
	}

	_, err = store.CreatePurchaseOrder(ctx, CreatePurchaseOrderInput{
		WarehouseID: "new-jersey-hub",
		ProductSKU:  "",
	})
	if err == nil {
		t.Fatalf("expected create purchase order validation error")
	}
	beforeInbound, err := store.inventoryLevelBySKUWarehouse(ctx, "frame-desk", "new-jersey-hub")
	if err != nil {
		t.Fatalf("inventory level before purchase order: %v", err)
	}
	createdPO, err := store.CreatePurchaseOrder(ctx, CreatePurchaseOrderInput{
		WarehouseID: "new-jersey-hub",
		ProductSKU:  "frame-desk",
		Quantity:    0,
		Status:      "",
	})
	if err != nil {
		t.Fatalf("create purchase order: %v", err)
	}
	if createdPO.Order.ID == "" || len(createdPO.Lines) != 1 {
		t.Fatalf("unexpected purchase order payload: %#v", createdPO)
	}
	afterInbound, err := store.inventoryLevelBySKUWarehouse(ctx, "frame-desk", "new-jersey-hub")
	if err != nil {
		t.Fatalf("inventory level after purchase order: %v", err)
	}
	if afterInbound.Inbound <= beforeInbound.Inbound {
		t.Fatalf("expected inbound increase after purchase order, before=%d after=%d", beforeInbound.Inbound, afterInbound.Inbound)
	}
}

func TestPureHelperFunctions(t *testing.T) {
	t.Parallel()

	if got := normalizeCommentReaction("thumbs-down"); got != "down" {
		t.Fatalf("normalizeCommentReaction thumbs-down=%s", got)
	}
	if got := normalizeCommentReaction("something-else"); got != "up" {
		t.Fatalf("normalizeCommentReaction default=%s", got)
	}

	if got := nonEmptyString("  ", "fallback"); got != "fallback" {
		t.Fatalf("nonEmptyString fallback=%s", got)
	}
	if got := nonEmptyString(" value ", "fallback"); got != "value" {
		t.Fatalf("nonEmptyString trim=%s", got)
	}

	if got := nullableString("  "); got.(sql.NullString).Valid {
		t.Fatalf("nullableString empty should be invalid null string")
	}
	if got := nullableString("x"); got.(string) != "x" {
		t.Fatalf("nullableString value=%v", got)
	}

	if got := clampInt(2, 3, 5); got != 3 {
		t.Fatalf("clampInt low=%d", got)
	}
	if got := clampInt(6, 3, 5); got != 5 {
		t.Fatalf("clampInt high=%d", got)
	}
	if got := clampInt(4, 3, 5); got != 4 {
		t.Fatalf("clampInt middle=%d", got)
	}
	if got := maxInt(8, 3); got != 8 {
		t.Fatalf("maxInt expected 8 got %d", got)
	}

	if got := inventoryWeeklyUnits("frame-desk", "new-jersey-hub"); got <= 0 {
		t.Fatalf("inventoryWeeklyUnits unexpected %d", got)
	}
	if got := inventoryWeeklyUnits("unknown", "unknown"); got != 4 {
		t.Fatalf("inventoryWeeklyUnits default expected 4 got %d", got)
	}
	if got := inventoryRegionalShare("new-jersey-hub"); got != 42 {
		t.Fatalf("inventoryRegionalShare new jersey %d", got)
	}
	if got := inventoryRegionalShare("unknown"); got != 20 {
		t.Fatalf("inventoryRegionalShare default %d", got)
	}

	if got := inventoryMarketPressure(82, 4, 0); got != "Hot market" {
		t.Fatalf("inventoryMarketPressure hot=%s", got)
	}
	if got := inventoryMarketPressure(70, 9, 12); got != "Growing demand" {
		t.Fatalf("inventoryMarketPressure growth=%s", got)
	}
	if got := inventoryMarketPressure(50, 20, 0); got != "Softening" {
		t.Fatalf("inventoryMarketPressure softening=%s", got)
	}
	if got := inventoryMarketPressure(50, 10, 2); got != "Stable" {
		t.Fatalf("inventoryMarketPressure stable=%s", got)
	}

	if got := inventoryMarketSignal("accessories", "new-jersey-hub", 5); got == "" {
		t.Fatalf("inventoryMarketSignal accessories empty")
	}
	if got := inventoryMarketSignal("desks", "nevada-hub", 5); got == "" {
		t.Fatalf("inventoryMarketSignal desks empty")
	}
	if got := inventoryMarketSignal("other", "other", 12); got == "" {
		t.Fatalf("inventoryMarketSignal high velocity empty")
	}
	if got := inventoryMarketSignal("other", "other", 2); got == "" {
		t.Fatalf("inventoryMarketSignal default empty")
	}

	if got := inventoryStatusForProduct("low_stock", 20); got != "promise_risk" {
		t.Fatalf("inventoryStatusForProduct low stock=%s", got)
	}
	if got := inventoryStatusForProduct("in_stock", 0); got != "promise_risk" {
		t.Fatalf("inventoryStatusForProduct zero=%s", got)
	}
	if got := inventoryStatusForProduct("in_stock", 10); got != "balanced" {
		t.Fatalf("inventoryStatusForProduct balanced=%s", got)
	}

	createdInput := prepareCreateProductInput(CreateProductInput{
		SKU:        "  sku-1  ",
		Slug:       " slug-1 ",
		PriceCents: -1,
		Available:  -2,
		Inbound:    -3,
	})
	if createdInput.SKU != "sku-1" || createdInput.Slug != "slug-1" {
		t.Fatalf("prepareCreateProductInput trim failed: %#v", createdInput)
	}
	if createdInput.PriceCents != 0 || createdInput.Available != 0 || createdInput.Inbound != 0 {
		t.Fatalf("prepareCreateProductInput numeric normalization failed: %#v", createdInput)
	}
	if createdInput.Title == "" || createdInput.WarehouseID == "" {
		t.Fatalf("prepareCreateProductInput defaults missing: %#v", createdInput)
	}

	current := ProductAdminRecord{
		SKU:          "sku",
		Slug:         "slug",
		Title:        "title",
		Category:     "category",
		PriceCents:   900,
		Status:       "in_stock",
		Finish:       "finish",
		Summary:      "summary",
		Details:      "details",
		SEOTitle:     "seo title",
		SEODescription: "seo desc",
		WarehouseID:  "new-jersey-hub",
		Available:    4,
		Inbound:      2,
	}
	updatedInput := prepareUpdateProductInput(current, UpdateProductInput{
		PriceCents: -1,
		Available:  -1,
		Inbound:    -1,
	})
	if updatedInput.SKU != "sku" || updatedInput.PriceCents != 900 {
		t.Fatalf("prepareUpdateProductInput normalization failed: %#v", updatedInput)
	}
	if updatedInput.Available != 4 || updatedInput.Inbound != 2 {
		t.Fatalf("prepareUpdateProductInput quantity fallback failed: %#v", updatedInput)
	}

	items := []ProductAdminRecord{
		{Title: "B", PriceCents: 100, Volume: 1, Status: "in_stock", UpdatedAt: "1"},
		{Title: "A", PriceCents: 1000, Volume: 10, Status: "flagged", UpdatedAt: "2"},
	}
	sortProductAdminItems(items, "price")
	if items[0].Title != "A" {
		t.Fatalf("sort by price failed: %#v", items)
	}
	sortProductAdminItems(items, "volume")
	if items[0].Title != "A" {
		t.Fatalf("sort by volume failed: %#v", items)
	}
	sortProductAdminItems(items, "status")
	if !sort.SliceIsSorted(items, func(i, j int) bool { return items[i].Status <= items[j].Status }) {
		t.Fatalf("sort by status failed: %#v", items)
	}
	sortProductAdminItems(items, "updated")
	if items[0].UpdatedAt != "2" {
		t.Fatalf("sort by default updated failed: %#v", items)
	}

	base := repository.Product{Category: "desks"}
	if reason := relatedProductReason(base, repository.Product{Category: "desks"}); reason == "" {
		t.Fatalf("relatedProductReason same-category empty")
	}
	if reason := relatedProductReason(base, repository.Product{Category: "storage"}); reason == "" {
		t.Fatalf("relatedProductReason cross-category empty")
	}

	if pressure, _, _, _ := warehouseOperationalProfile("new-jersey-hub"); pressure == "" {
		t.Fatalf("warehouseOperationalProfile new jersey empty")
	}
	if pressure, _, _, _ := warehouseOperationalProfile("nevada-hub"); pressure == "" {
		t.Fatalf("warehouseOperationalProfile nevada empty")
	}
	if pressure, _, _, _ := warehouseOperationalProfile("unknown"); pressure == "" {
		t.Fatalf("warehouseOperationalProfile default empty")
	}
}

func newSeededStore(t *testing.T) (context.Context, *Store, *sql.DB) {
	t.Helper()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "atlas.db")
	database, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	migrationsPath := resolveDataPath(t, "migrations")
	fallbackSchemaPath := resolveDataPath(t, "schema.sql")
	if err := Migrate(ctx, database, migrationsPath, fallbackSchemaPath); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := Seed(ctx, database); err != nil {
		t.Fatalf("seed database: %v", err)
	}
	return ctx, NewStore(database), database
}

func resolveDataPath(t *testing.T, parts ...string) string {
	t.Helper()

	candidates := []string{
		filepath.Join(append([]string{"..", "data"}, parts...)...),
		filepath.Join(append([]string{"examples", "86-atlas-commerce-os", "server", "data"}, parts...)...),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatalf("unable to resolve data path for %v", parts)
	return ""
}
