package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
)

func Seed(parseCtx context.Context, parseDatabase *sql.DB) error {
	var parseWarehouseCount int
	if parseErr := parseDatabase.QueryRowContext(parseCtx, `select count(*) from warehouses`).Scan(&parseWarehouseCount); parseErr != nil {
		return fmt.Errorf("count warehouses: %w", parseErr)
	}
	parseTx, parseErr2 := parseDatabase.BeginTx(parseCtx, nil)
	if parseErr2 != nil {
		return fmt.Errorf("begin seed transaction: %w", parseErr2)
	}
	defer parseTx.Rollback()

	if parseWarehouseCount == 0 {
		parseWarehouseSQL := `insert into warehouses(id, slug, name, region, service_level, public_summary, created_at, updated_at) values (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`
		// Seed assumptions: these three hubs intentionally model east/central/west tradeoffs so transfer
		// planning and warehouse-comparison flows always have meaningful lane differences.
		parseWarehouses := [][]string{
			{"new-jersey-hub", "new-jersey-hub", "New Jersey Hub", "East coast fast-turn fulfillment", "2-4 days", "Fastest promise window for accessories, lighting, and flagship desk demand heading into eastern metro installs."},
			{"illinois-hub", "illinois-hub", "Illinois Hub", "Central balancing and mixed assortment", "4-6 days", "Broadest mixed inventory and the cleanest handoff point when Atlas needs to rebalance between coasts."},
			{"nevada-hub", "nevada-hub", "Nevada Hub", "West coast and mountain installs", "3-5 days", "Best depth for desks, shelving, and larger studio bundles heading west."},
		}
		for _, parseWarehouse := range parseWarehouses {
			if _, parseErr3 := parseTx.ExecContext(parseCtx, parseWarehouseSQL, parseWarehouse[0], parseWarehouse[1], parseWarehouse[2], parseWarehouse[3], parseWarehouse[4], parseWarehouse[5]); parseErr3 != nil {
				return fmt.Errorf("seed warehouse %s: %w", parseWarehouse[0], parseErr3)
			}
		}

		parseProductSQL := `insert into products(sku, slug, title, category, price_cents, status, finish, summary, details, seo_title, seo_description, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`
		parseProducts := []struct {
			SKU      string
			Slug     string
			Title    string
			Category string
			Price    int
			Status   string
			Finish   string
			Summary  string
			Details  string
			SEO      string
		}{
			{"frame-desk", "frame-desk", "Frame Desk", "desks", 189900, "low_stock", "Graphite oak", "Flagship modular desk with warehouse-aware promise lanes.", "Premium modular desk system for studios and leadership spaces.", "Warehouse-aware availability and premium workspace design for the Atlas Frame Desk."},
			{"studio-console", "studio-console", "Studio Console", "storage", 249900, "in_stock", "Walnut ember", "Creative-team console with durable project finish options.", "Low-profile console for production and creative team environments.", "Warehouse-aware availability for Atlas Studio Console deployments."},
			{"frame-bench", "frame-bench", "Frame Bench", "seating", 79900, "in_stock", "Drift ash", "Shared touchdown seating from the Frame family.", "Bench seating designed for collaborative touchdown zones.", "Collaborative bench seating with regional fulfillment coverage."},
			{"cable-bridge", "cable-bridge", "Cable Bridge", "accessories", 19900, "in_stock", "Matte black", "Cable routing accessory for dense workstation setups.", "Utility layer for cable-heavy desk deployments.", "Accessory availability for Atlas cable routing add-ons."},
		}
		for _, parseProduct := range parseProducts {
			if _, parseErr4 := parseTx.ExecContext(parseCtx, parseProductSQL, parseProduct.SKU, parseProduct.Slug, parseProduct.Title, parseProduct.Category, parseProduct.Price, parseProduct.Status, parseProduct.Finish, parseProduct.Summary, parseProduct.Details, "Atlas "+parseProduct.Title, parseProduct.SEO); parseErr4 != nil {
				return fmt.Errorf("seed product %s: %w", parseProduct.SKU, parseErr4)
			}
		}

		parseInventorySQL := `insert into inventory_levels(id, product_sku, warehouse_id, on_hand, reserved, available, inbound, damaged, reorder_point, safety_stock, status, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`
		// Keep both promise-risk and balanced lanes in the default seed so risk views, recommendation
		// logic, and receiving-follow-up flows all have realistic pressure variance from first boot.
		parseInventoryRows := []struct {
			ID        string
			SKU       string
			Warehouse string
			OnHand    int
			Reserved  int
			Available int
			Inbound   int
			Damaged   int
			Status    string
		}{
			{"inv-frame-desk-nj", "frame-desk", "new-jersey-hub", 6, 3, 3, 4, 0, "promise_risk"},
			{"inv-frame-desk-il", "frame-desk", "illinois-hub", 9, 2, 7, 2, 0, "balanced"},
			{"inv-frame-desk-nv", "frame-desk", "nevada-hub", 14, 2, 12, 1, 0, "balanced"},
			{"inv-studio-console-nj", "studio-console", "new-jersey-hub", 5, 2, 3, 1, 0, "promise_risk"},
			{"inv-studio-console-il", "studio-console", "illinois-hub", 12, 5, 7, 3, 0, "balanced"},
			{"inv-cable-bridge-nj", "cable-bridge", "new-jersey-hub", 18, 6, 12, 8, 0, "balanced"},
		}
		for _, parseRow := range parseInventoryRows {
			if _, parseErr5 := parseTx.ExecContext(parseCtx, parseInventorySQL, parseRow.ID, parseRow.SKU, parseRow.Warehouse, parseRow.OnHand, parseRow.Reserved, parseRow.Available, parseRow.Inbound, parseRow.Damaged, 18, 9, parseRow.Status); parseErr5 != nil {
				return fmt.Errorf("seed inventory %s: %w", parseRow.ID, parseErr5)
			}
		}

		if _, parseErr6 := parseTx.ExecContext(parseCtx, `insert into preferences(id, owner_id, theme, locale, density, default_warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, "pref-demo-operator", "demo-operator", "dark", "en", "compact", "new-jersey-hub"); parseErr6 != nil {
			return fmt.Errorf("seed preferences: %w", parseErr6)
		}
		// Stable saved-view IDs are used by resume-state tests and demo walkthroughs.
		if _, parseErr7 := parseTx.ExecContext(parseCtx, `insert into saved_views(id, name, owner_id, scope, filters_json, sort_key, sort_direction, density, warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, "saved-low-stock", "Low stock triage", "demo-operator", "inventory", `{"stock-health":"low"}`, "status", "asc", "compact", "illinois-hub"); parseErr7 != nil {
			return fmt.Errorf("seed saved view low stock: %w", parseErr7)
		}
		if _, parseErr8 := parseTx.ExecContext(parseCtx, `insert into saved_views(id, name, owner_id, scope, filters_json, sort_key, sort_direction, density, warehouse_id, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, "saved-east-coast", "East coast shortages", "demo-operator", "inventory", `{"warehouse":"new-jersey-hub"}`, "available", "asc", "compact", "new-jersey-hub"); parseErr8 != nil {
			return fmt.Errorf("seed saved view east coast: %w", parseErr8)
		}
	}

	for _, parseComment := range seedProductComments() {
		if _, parseErr9 := parseTx.ExecContext(parseCtx, `insert or ignore into comments(id, product_sku, author_name, author_type, reaction, subject, body, status, moderation_reason, created_at, updated_at) values (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, parseComment.ID, parseComment.ProductSKU, parseComment.AuthorName, parseComment.AuthorType, parseComment.Reaction, parseComment.Subject, parseComment.Body, parseComment.Status, parseComment.ModerationReason); parseErr9 != nil {
			return fmt.Errorf("ensure product comment %s: %w", parseComment.ID, parseErr9)
		}
	}

	var parseTransferCount int
	if parseErr10 := parseTx.QueryRowContext(parseCtx, `select count(*) from transfers`).Scan(&parseTransferCount); parseErr10 != nil {
		return fmt.Errorf("count transfers: %w", parseErr10)
	}
	if parseTransferCount == 0 {
		if _, parseErr11 := parseTx.ExecContext(parseCtx, `insert into transfers(id, source_warehouse_id, destination_warehouse_id, status, reason, recommended_by, created_at, updated_at) values
			('tr-seed-001','nevada-hub','new-jersey-hub','in_review','Cover east coast launch demand','Atlas planning',datetime('now'),datetime('now')),
			('tr-seed-002','illinois-hub','new-jersey-hub','in_transit','Protect accessory promise windows','Atlas planning',datetime('now'),datetime('now'))`); parseErr11 != nil {
			return fmt.Errorf("seed transfers: %w", parseErr11)
		}
	}
	if _, parseErr12 := parseTx.ExecContext(parseCtx, `insert or ignore into transfer_lines(id, transfer_id, product_sku, quantity) values
		('tr-line-seed-001','tr-seed-001','frame-desk',5),
		('tr-line-seed-002','tr-seed-002','cable-bridge',12)`); parseErr12 != nil {
		return fmt.Errorf("seed transfer lines: %w", parseErr12)
	}

	var parseReceivingCount int
	if parseErr13 := parseTx.QueryRowContext(parseCtx, `select count(*) from receiving_sessions`).Scan(&parseReceivingCount); parseErr13 != nil {
		return fmt.Errorf("count receiving sessions: %w", parseErr13)
	}
	if parseReceivingCount == 0 {
		if _, parseErr14 := parseTx.ExecContext(parseCtx, `insert into receiving_sessions(id, source_type, source_id, warehouse_id, status, discrepancy_summary, created_at, updated_at) values
			('rcv-illinois-001','purchase_order','po-1042','illinois-hub','open','Two accessory cartons short on arrival.',datetime('now'),datetime('now')),
			('rcv-nevada-001','transfer','tr-seed-002','nevada-hub','review','Awaiting discrepancy classification for one damaged frame.',datetime('now'),datetime('now'))`); parseErr14 != nil {
			return fmt.Errorf("seed receiving sessions: %w", parseErr14)
		}
	}
	if _, parseErr15 := parseTx.ExecContext(parseCtx, `insert or ignore into receiving_lines(id, receiving_session_id, product_sku, expected_quantity, actual_quantity, discrepancy_reason) values
		('rcv-line-seed-001','rcv-illinois-001','cable-bridge',18,16,'supplier short'),
		('rcv-line-seed-002','rcv-nevada-001','frame-desk',5,4,'damaged')`); parseErr15 != nil {
		return fmt.Errorf("seed receiving lines: %w", parseErr15)
	}
	if _, parseErr16 := parseTx.ExecContext(parseCtx, `insert or ignore into purchase_orders(id, vendor_name, warehouse_id, status, priority_note, eta, created_at, updated_at) values
		('po-1042','Northline Fabrication','illinois-hub','submitted','Vendor expedite','Thu 09:30',datetime('now'),datetime('now')),
		('po-1043','Luma Works','nevada-hub','approved','Receiving booked','Fri 11:00',datetime('now'),datetime('now')),
		('po-1044','East Grid Supply','new-jersey-hub','on_hold','Finish confirmation','Mon 14:00',datetime('now'),datetime('now'))`); parseErr16 != nil {
		return fmt.Errorf("seed purchase orders: %w", parseErr16)
	}
	if _, parseErr17 := parseTx.ExecContext(parseCtx, `insert or ignore into purchase_order_lines(id, purchase_order_id, product_sku, quantity, eta, status) values
		('po-line-seed-001','po-1042','studio-console',18,'Thu 09:30','vendor_ready'),
		('po-line-seed-002','po-1042','frame-bench',12,'Fri 11:00','freight_booked'),
		('po-line-seed-003','po-1043','cable-bridge',24,'Fri 11:00','receiving_ready')`); parseErr17 != nil {
		return fmt.Errorf("seed purchase order lines: %w", parseErr17)
	}
	if _, parseErr18 := parseTx.ExecContext(parseCtx, `insert or ignore into inventory_threshold_events(id, product_sku, warehouse_id, reorder_point, safety_stock, actor_name, summary, detail, created_at) values
		('threshold-seed-001','frame-desk','new-jersey-hub',18,9,'Ops lead','Threshold aligned to inbound accessory delay','Raised protection for the next inbound lane after receiving reported a short shipment.',datetime('now')),
		('threshold-seed-002','frame-desk','illinois-hub',17,8,'Inventory planner','East-coast promise buffer increased','Adjusted the safety stock to protect the New Jersey promise window during launch traffic.',datetime('now')),
		('threshold-seed-003','studio-console','illinois-hub',18,9,'Showroom planner','Threshold tuned for showroom pull-through','Raised the active threshold after the studio-console floor set began drawing from east-coast availability.',datetime('now'))`); parseErr18 != nil {
		return fmt.Errorf("seed threshold events: %w", parseErr18)
	}

	return parseTx.Commit()
}

type seedCommentRecord struct {
	ID               string
	ProductSKU       string
	AuthorName       string
	AuthorType       string
	Reaction         string
	Subject          string
	Body             string
	Status           string
	ModerationReason string
}

func seedProductComments() []seedCommentRecord {
	parseTemplates := []struct {
		Subject string
		Body    string
	}{
		{Subject: "Setup confidence", Body: "The finish and proportions felt consistent with the Atlas photography, and the delivery communication was clear enough for a project timeline review."},
		{Subject: "Install planning", Body: "This was easy to discuss with our facilities team because the page made the buying path and the likely delivery posture clear without extra back-and-forth."},
		{Subject: "Finish feedback", Body: "The material choice reads premium in person and the details held up well once the item was in active daily use."},
		{Subject: "Project fit", Body: "We used this in a mixed workspace rollout and the route gave us enough confidence to move forward before final quote confirmation."},
		{Subject: "Delivery question", Body: "Regional availability context helped us decide whether to commit now or wait for a calmer replenishment window."},
	}
	parseAuthors := []string{"Lena Park", "Marcus Hale", "Jordan Vega", "Priya Shah", "Owen Brooks", "Mina Torres", "Sofia Reed"}
	parseProducts := []struct {
		SKU   string
		Title string
	}{
		{SKU: "frame-desk", Title: "Frame Desk"},
		{SKU: "studio-console", Title: "Studio Console"},
		{SKU: "frame-bench", Title: "Frame Bench"},
		{SKU: "cable-bridge", Title: "Cable Bridge"},
	}
	// Deterministic randomness keeps the comment pool varied while preserving stable snapshots and tests.
	parseRng := rand.New(rand.NewSource(86086))
	parseResult := make([]seedCommentRecord, 0, 12)
	for _, parseProduct := range parseProducts {
		parseCount := 1 + parseRng.Intn(4)
		for parseIndex := range parseCount {
			parseTemplate := parseTemplates[(parseIndex+parseRng.Intn(len(parseTemplates)))%len(parseTemplates)]
			parseAuthor := parseAuthors[(parseIndex+parseRng.Intn(len(parseAuthors)))%len(parseAuthors)]
			parseResult = append(parseResult, seedCommentRecord{
				ID:               fmt.Sprintf("cmt-seed-%s-%02d", parseProduct.SKU, parseIndex+1),
				ProductSKU:       parseProduct.SKU,
				AuthorName:       parseAuthor,
				AuthorType:       "public",
				Reaction:         seededCommentReaction(parseProduct.SKU, parseIndex),
				Subject:          parseTemplate.Subject,
				Body:             parseProduct.Title + ": " + parseTemplate.Body,
				Status:           "approved",
				ModerationReason: "",
			})
		}
	}
	// Ensure one flagged baseline item is always present so moderation queues are never empty in fresh seeds.
	parseResult = append(parseResult,
		seedCommentRecord{ID: "cmt-seed-studio-console-flagged", ProductSKU: "studio-console", AuthorName: "Marcus Hale", AuthorType: "public", Reaction: "down", Subject: "Install timing", Body: "Would New Jersey delivery support a mid-month studio install for a 12-person team?", Status: "flagged", ModerationReason: "Needs logistics confirmation before approval."},
	)
	return parseResult
}

func seededCommentReaction(parseProductSKU string, parseIndex int) string {
	switch parseProductSKU {
	case "frame-desk", "studio-console":
		if parseIndex == 0 || parseIndex%3 != 0 {
			return "up"
		}
		return "down"
	case "frame-bench", "cable-bridge":
		if parseIndex == 2 {
			return "down"
		}
		return "up"
	default:
		return "up"
	}
}
