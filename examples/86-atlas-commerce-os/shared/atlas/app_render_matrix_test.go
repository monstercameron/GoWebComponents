package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestAppRenderMatrixAcrossPublicAndInternalRoutes(t *testing.T) {
	baseInventoryRows := sampleInventoryRows()
	baseProducts := sampleProductCards()
	baseWarehouses := sampleWarehouseCards()
	baseComments := sampleCommentRecords()
	baseTransfers := sampleTransferRecords()
	baseOrders := samplePurchaseOrders()
	baseReceiving := sampleReceivingRecords()
	baseWarehouseOps := sampleWarehouseOpsRecords()
	baseProductAdmins := sampleProductAdminCards()

	cases := []struct {
		name    string
		path    string
		page    any
		overlay any
	}{
		{
			name: "landing",
			path: RouteLanding,
		},
		{
			name: "catalog",
			path: RouteCatalog,
			page: catalogPage{
				Items:    baseProducts,
				Total:    len(baseProducts),
				Page:     1,
				PageSize: 12,
				Query: catalogQueryState{
					Search:    "frame",
					Category:  "desks",
					Warehouse: "new-jersey-hub",
					Sort:      "featured",
				},
				Focus: "Featured Atlas systems",
			},
		},
		{
			name: "product-detail",
			path: RouteProduct,
			page: productDetailPage{
				Product:  baseProducts[0],
				Comments: baseComments,
			},
		},
		{
			name: "warehouses-directory",
			path: RouteWarehouses,
			page: warehouseDirectoryPage{
				Items: baseWarehouses,
			},
		},
		{
			name: "warehouse-public-detail",
			path: RouteWarehousePublicDetail,
			page: warehouseDetailPage{
				Warehouse: baseWarehouses[0],
				Products:  baseProducts,
			},
		},
		{
			name: "availability",
			path: RouteWarehouseAvailability,
			page: availabilityPage{
				Warehouse: baseWarehouses[0],
				Product:   baseProducts[0],
				Available: 2,
				Inbound:   4,
				Status:    "promise_risk",
			},
		},
		{
			name: "dashboard",
			path: RouteDashboard,
			page: dashboardPage{
				Summary:   sampleSummary("Dashboard pulse"),
				Alerts:    3,
				Transfers: baseTransfers,
				Receiving: baseReceiving,
				Comments:  baseComments,
				Orders:    baseOrders,
			},
		},
		{
			name: "inventory",
			path: RouteInventory,
			page: inventoryCMSPage{
				Summary: sampleSummary("Inventory shell"),
				Items:   baseInventoryRows,
				Filters: map[string]string{
					"warehouse": "new-jersey-hub",
					"status":    "promise_risk",
					"sort":      "available",
				},
			},
		},
		{
			name: "sku-detail",
			path: RouteSKUDetail,
			page: inventoryDetailPage{
				SKU:   "frame-desk",
				Title: "Frame Desk",
				Rows:  baseInventoryRows,
			},
		},
		{
			name: "sku-threshold-history",
			path: RouteSKUThresholdHistory,
			page: inventoryDetailPage{
				SKU:   "frame-desk",
				Title: "Frame Desk",
				Rows:  baseInventoryRows,
			},
			overlay: inventoryThresholdHistoryPanelPage{
				SKU: "frame-desk",
				Items: []inventoryThresholdHistoryItem{
					{
						ID:           "th-001",
						ProductSKU:   "frame-desk",
						WarehouseID:  "new-jersey-hub",
						ReorderPoint: 18,
						SafetyStock:  9,
						ActorName:    "Ops lead",
						Summary:      "Threshold tuned for launch demand",
						Detail:       "Raised reorder protection after east-coast surge.",
						CreatedAt:    "2026-03-25T00:00:00Z",
					},
				},
				Recommendations: []inventoryTransferRecommendationItem{
					{
						ProductSKU:               "frame-desk",
						SourceWarehouseID:        "nevada-hub",
						SourceWarehouseName:      "Nevada Hub",
						DestinationWarehouseID:   "new-jersey-hub",
						DestinationWarehouseName: "New Jersey Hub",
						Quantity:                 4,
						Priority:                 "Promise recovery",
						Reason:                   "Shift stock to protect east-coast commitments.",
					},
				},
			},
		},
		{
			name: "warehouse-ops",
			path: RouteWarehouseOps,
			page: warehouseOpsList{
				Summary: sampleSummary("Warehouse operations"),
				Items:   baseWarehouseOps,
			},
		},
		{
			name: "warehouse-detail-internal",
			path: RouteWarehouseDetail,
			page: warehouseInventoryDetailPage{
				Summary:   sampleSummary("Facility detail"),
				Warehouse: baseWarehouseOps[0],
				Inventory: baseInventoryRows,
				Orders:    baseOrders,
				Filters: map[string]string{
					"status": "promise_risk",
				},
			},
		},
		{
			name: "warehouse-item-detail",
			path: RouteWarehouseItemDetail,
			page: warehouseInventoryItemDetailPage{
				Warehouse: baseWarehouseOps[0],
				Item:      baseInventoryRows[0],
				Product:   baseProductAdmins[0],
				Network:   baseInventoryRows,
				Orders:    baseOrders,
				Filters: map[string]string{
					"status": "all",
					"sort":   "available",
				},
			},
		},
		{
			name: "transfers",
			path: RouteTransfers,
			page: transferList{
				Items: baseTransfers,
			},
		},
		{
			name: "transfer-detail",
			path: RouteTransferDetail,
			page: transferDetailPage{
				Transfer: baseTransfers[0],
				Lines: []transferLineRecord{
					{ID: "tr-line-1", TransferID: baseTransfers[0].ID, ProductSKU: "frame-desk", Quantity: 5},
				},
			},
		},
		{
			name: "purchase-orders",
			path: RoutePurchaseOrders,
			page: purchaseOrderList{
				Summary: sampleSummary("Purchase-order watch"),
				Items:   baseOrders,
			},
		},
		{
			name: "purchase-order-detail",
			path: RoutePurchaseOrderDetail,
			page: purchaseOrderDetailPage{
				Order: baseOrders[0],
				Lines: []purchaseOrderLineRecord{
					{ID: "po-line-1", PurchaseOrderID: baseOrders[0].ID, ProductSKU: "frame-desk", Quantity: 12, ETA: "Thu 09:30", Status: "submitted"},
				},
			},
		},
		{
			name: "receiving",
			path: RouteReceiving,
			page: receivingList{
				Items: baseReceiving,
			},
		},
		{
			name: "receiving-detail",
			path: RouteReceivingSessionDetail,
			page: receivingDetailPage{
				Session: baseReceiving[0],
				Lines: []receivingLineRecord{
					{ID: "rcv-line-1", ReceivingSessionID: baseReceiving[0].ID, ProductSKU: "frame-desk", ExpectedQuantity: 8, ActualQuantity: 7, DiscrepancyReason: "supplier short"},
				},
			},
		},
		{
			name: "comments",
			path: RouteComments,
			page: commentList{
				Summary: sampleSummary("Buyer inbox"),
				Items:   baseComments,
			},
		},
		{
			name: "settings",
			path: RouteSettings,
			page: settingsPage{
				Summary: sampleSummary("Shell settings"),
			},
		},
		{
			name: "products-cms",
			path: "/app/products",
			page: productCMSPageData{
				Items: baseProductAdmins,
				Filters: productCMSFilters{
					Search:   "frame",
					Category: "desks",
					Status:   "all",
					Sort:     "updated",
				},
				Total:    len(baseProductAdmins),
				Editable: true,
			},
		},
		{
			name: "product-editor",
			path: "/app/products/frame-desk",
			page: baseProductAdmins[0],
		},
		{
			name: "fallback-path",
			path: "/unknown-route",
			page: map[string]any{
				"headline": "fallback payload",
			},
		},
	}

	for _, tc := range cases {
		payload := samplePayloadForRoute(tc.path, tc.page, tc.overlay)
		markup, err := ui.RenderToString(App(payload))
		if err != nil {
			t.Fatalf("%s render failed: %v", tc.name, err)
		}
		if !strings.Contains(markup, "atlas-shell-root") {
			t.Fatalf("%s missing shell root markup", tc.name)
		}
	}
}

func TestAppRenderSpecialScreens(t *testing.T) {
	mockPayload := samplePayloadForRoute("/auth/mock-sign-in", mockSignInPage{
		Roles: []mockSignInRole{
			{Value: "inventory_manager", Label: "Inventory manager", Description: "Manages inventory workflows."},
			{Value: "buyer_support", Label: "Buyer support", Description: "Handles buyer inbox and moderation."},
		},
		Next:    RouteDashboard,
		Message: "Choose an Atlas role to continue.",
	}, nil)
	mockPayload.Route.Screen = "mock-sign-in"
	mockPayload.Route.Surface = "internal"
	mockPayload.User = nil

	mockMarkup, err := ui.RenderToString(App(mockPayload))
	if err != nil {
		t.Fatalf("mock-sign-in render failed: %v", err)
	}
	if !strings.Contains(mockMarkup, "Start session") {
		t.Fatalf("mock-sign-in markup missing expected action")
	}

	recoveryPayload := samplePayloadForRoute("/app/recovery", recoveryPage{
		Title:         "Atlas recovery",
		Message:       "Route payload could not be resolved.",
		RecoveryHref:  RouteDashboard,
		RecoveryLabel: "Back to dashboard",
		Detail:        "Try reopening the route after data reload.",
	}, nil)
	recoveryPayload.Route.Screen = "recovery"
	recoveryPayload.Route.Surface = "internal"
	recoveryPayload.User = &UserSession{
		ID:               "ops-1",
		DisplayName:      "Atlas Operator",
		Role:             "inventory_manager",
		DefaultWarehouse: "new-jersey-hub",
	}

	recoveryMarkup, err := ui.RenderToString(App(recoveryPayload))
	if err != nil {
		t.Fatalf("recovery render failed: %v", err)
	}
	if !strings.Contains(recoveryMarkup, "Back to dashboard") {
		t.Fatalf("recovery markup missing expected recovery link")
	}
}

func samplePayloadForRoute(path string, page any, overlay any) Payload {
	route, ok := RouteBootstrapForPath(path)
	if !ok {
		meta := MetadataForPath(path)
		surface := "public"
		if strings.HasPrefix(path, "/app") {
			surface = "internal"
		}
		route = RouteBootstrap{
			Path:        path,
			Surface:     surface,
			Screen:      "custom",
			Title:       meta.Title,
			Description: meta.Description,
			Canonical:   meta.Canonical,
		}
	}
	if route.Path == "" {
		route.Path = path
	}
	if route.Surface == "" {
		route.Surface = "public"
		if strings.HasPrefix(path, "/app") {
			route.Surface = "internal"
		}
	}
	route.Query = map[string][]string{
		"status": {"promise_risk"},
		"sort":   {"updated"},
	}
	route.Params = map[string]string{
		"sku":         "frame-desk",
		"warehouseId": "illinois-hub",
		"productSlug": "frame-desk",
	}

	payload := Payload{
		Route:       route,
		Preferences: DefaultPreferences(),
		I18n:        DefaultI18n("en"),
		Theme:       DefaultTheme(),
		Data:        map[string]any{},
		Requests:    map[string]Request{},
		SavedViews: []SavedViewPayload{
			{
				Name:          "Low stock triage",
				Scope:         "inventory",
				SortKey:       "available",
				SortDirection: "asc",
				Filters: map[string]string{
					"warehouse": "new-jersey-hub",
					"status":    "promise_risk",
				},
			},
		},
		CSRF: "test-csrf-token",
	}
	if strings.HasPrefix(path, "/app") {
		payload.User = &UserSession{
			ID:               "ops-1",
			DisplayName:      "Atlas Operator",
			Role:             "inventory_manager",
			DefaultWarehouse: "new-jersey-hub",
		}
		payload.Route.Surface = "internal"
	}
	if page != nil {
		payload.Data["page"] = page
		payload.Requests["page"] = Request{
			Method: "GET",
			URL:    StartupRequestURL(path, nil),
			Status: 200,
			Data: map[string]any{
				"page": page,
			},
		}
	}
	if overlay != nil {
		payload.Data["overlay"] = overlay
		payload.Requests["overlay"] = Request{
			Method: "GET",
			URL:    StartupRequestURL(path, nil) + "#overlay",
			Status: 200,
			Data: map[string]any{
				"overlay": overlay,
			},
		}
	}
	return payload
}

func sampleSummary(headline string) pageSummary {
	return pageSummary{
		Headline: headline,
		Items: []pageSummaryItem{
			{Label: "Metric one", Value: "12", Detail: "Primary route signal"},
			{Label: "Metric two", Value: "7", Detail: "Secondary route signal"},
		},
	}
}

func sampleProductCards() []productCard {
	return []productCard{
		{
			SKU:            "frame-desk",
			Slug:           "frame-desk",
			Title:          "Frame Desk",
			Category:       "desks",
			PriceCents:     189900,
			Status:         "low_stock",
			Summary:        "Flagship modular desk with warehouse-aware promise lanes.",
			Details:        "Premium modular desk system for studio and leadership spaces.",
			Finish:         "Graphite oak",
			SEODescription: "Warehouse-aware availability for Atlas Frame Desk.",
			WarehouseID:    "new-jersey-hub",
			WarehouseName:  "New Jersey Hub",
			Available:      3,
			Inbound:        4,
			Volume:         7,
			UpdatedAt:      "2026-03-25T00:00:00Z",
		},
		{
			SKU:            "studio-console",
			Slug:           "studio-console",
			Title:          "Studio Console",
			Category:       "storage",
			PriceCents:     249900,
			Status:         "in_stock",
			Summary:        "Creative-team console with durable project finish options.",
			Details:        "Low-profile console for production environments.",
			Finish:         "Walnut ember",
			SEODescription: "Availability for Atlas Studio Console.",
			WarehouseID:    "illinois-hub",
			WarehouseName:  "Illinois Hub",
			Available:      7,
			Inbound:        3,
			Volume:         10,
			UpdatedAt:      "2026-03-25T00:00:00Z",
		},
	}
}

func sampleWarehouseCards() []warehouseCard {
	return []warehouseCard{
		{
			ID:            "new-jersey-hub",
			Slug:          "new-jersey-hub",
			Name:          "New Jersey Hub",
			Region:        "East coast",
			ServiceLevel:  "2-4 days",
			PublicSummary: "Fastest promise lane for eastern installs.",
			Pressure:      "Promise risk",
			Staffing:      "88% staffed",
			Backlog:       "2 blocked receipts",
			Focus:         "Protect east-coast promise windows.",
			Available:     18,
			Inbound:       6,
			RiskCount:     2,
		},
		{
			ID:            "illinois-hub",
			Slug:          "illinois-hub",
			Name:          "Illinois Hub",
			Region:        "Central",
			ServiceLevel:  "4-6 days",
			PublicSummary: "Balancing lane for mixed assortment.",
			Pressure:      "Balancing lane",
			Staffing:      "84% staffed",
			Backlog:       "2 dock queues",
			Focus:         "Balance inbound accessories.",
			Available:     24,
			Inbound:       5,
			RiskCount:     1,
		},
	}
}

func sampleInventoryRows() []inventoryRow {
	return []inventoryRow{
		{
			ID:             "inv-frame-nj",
			SKU:            "frame-desk",
			Slug:           "frame-desk",
			Title:          "Frame Desk",
			Category:       "desks",
			PriceCents:     189900,
			ProductStatus:  "low_stock",
			WarehouseID:    "new-jersey-hub",
			WarehouseName:  "New Jersey Hub",
			OnHand:         6,
			Reserved:       3,
			Available:      3,
			CoverDays:      8,
			Inbound:        4,
			Damaged:        0,
			ReorderPoint:   18,
			SafetyStock:    9,
			Status:         "promise_risk",
			WeeklyUnits:    10,
			WeeklyRevenue:  1899000,
			SellThrough:    74,
			DemandScore:    82,
			RegionalShare:  42,
			ReorderUnits:   8,
			MarketPressure: "Hot market",
			MarketSignal:   "Commercial demand accelerating.",
			UpdatedAt:      "2026-03-25T00:00:00Z",
		},
		{
			ID:             "inv-frame-il",
			SKU:            "frame-desk",
			Slug:           "frame-desk",
			Title:          "Frame Desk",
			Category:       "desks",
			PriceCents:     189900,
			ProductStatus:  "in_stock",
			WarehouseID:    "illinois-hub",
			WarehouseName:  "Illinois Hub",
			OnHand:         10,
			Reserved:       2,
			Available:      8,
			CoverDays:      14,
			Inbound:        2,
			Damaged:        0,
			ReorderPoint:   14,
			SafetyStock:    7,
			Status:         "balanced",
			WeeklyUnits:    7,
			WeeklyRevenue:  1329300,
			SellThrough:    58,
			DemandScore:    66,
			RegionalShare:  33,
			ReorderUnits:   1,
			MarketPressure: "Growing demand",
			MarketSignal:   "Regional demand steady.",
			UpdatedAt:      "2026-03-25T00:00:00Z",
		},
	}
}

func sampleTransferRecords() []transferRecord {
	return []transferRecord{
		{
			ID:                   "tr-seed-001",
			SourceWarehouseID:    "nevada-hub",
			DestinationWarehouse: "new-jersey-hub",
			Status:               "in_review",
			Reason:               "Protect east-coast launch demand.",
			RecommendedBy:        "Atlas planning",
			CreatedAt:            "2026-03-25T00:00:00Z",
			UpdatedAt:            "2026-03-25T00:00:00Z",
		},
		{
			ID:                   "tr-seed-002",
			SourceWarehouseID:    "illinois-hub",
			DestinationWarehouse: "new-jersey-hub",
			Status:               "approved",
			Reason:               "Accessory promise stabilization.",
			RecommendedBy:        "Atlas planning",
			CreatedAt:            "2026-03-25T00:00:00Z",
			UpdatedAt:            "2026-03-25T00:00:00Z",
		},
	}
}

func samplePurchaseOrders() []purchaseOrderRecord {
	return []purchaseOrderRecord{
		{
			ID:            "po-1042",
			VendorName:    "Northline Fabrication",
			WarehouseID:   "illinois-hub",
			WarehouseName: "Illinois Hub",
			Status:        "submitted",
			PriorityNote:  "Vendor expedite",
			ETA:           "Thu 09:30",
			CreatedAt:     "2026-03-25T00:00:00Z",
			UpdatedAt:     "2026-03-25T00:00:00Z",
		},
		{
			ID:            "po-1043",
			VendorName:    "Luma Works",
			WarehouseID:   "nevada-hub",
			WarehouseName: "Nevada Hub",
			Status:        "approved",
			PriorityNote:  "Receiving booked",
			ETA:           "Fri 11:00",
			CreatedAt:     "2026-03-25T00:00:00Z",
			UpdatedAt:     "2026-03-25T00:00:00Z",
		},
	}
}

func sampleReceivingRecords() []receivingRecord {
	return []receivingRecord{
		{
			ID:                 "rcv-illinois-001",
			SourceType:         "purchase_order",
			SourceID:           "po-1042",
			WarehouseID:        "illinois-hub",
			Status:             "open",
			DiscrepancySummary: "Two accessory cartons short.",
			CreatedAt:          "2026-03-25T00:00:00Z",
			UpdatedAt:          "2026-03-25T00:00:00Z",
		},
		{
			ID:                 "rcv-nevada-001",
			SourceType:         "transfer",
			SourceID:           "tr-seed-002",
			WarehouseID:        "nevada-hub",
			Status:             "review",
			DiscrepancySummary: "Awaiting discrepancy classification.",
			CreatedAt:          "2026-03-25T00:00:00Z",
			UpdatedAt:          "2026-03-25T00:00:00Z",
		},
	}
}

func sampleCommentRecords() []commentRecord {
	return []commentRecord{
		{
			ID:         "cmt-001",
			ProductSKU: "frame-desk",
			AuthorName: "Lena Park",
			AuthorType: "public",
			Reaction:   "up",
			Subject:    "Setup confidence",
			Body:       "Delivery and fit context were clear for planning.",
			Status:     "pending",
			CreatedAt:  "2026-03-25T00:00:00Z",
			UpdatedAt:  "2026-03-25T00:00:00Z",
		},
		{
			ID:               "cmt-002",
			ProductSKU:       "studio-console",
			AuthorName:       "Marcus Hale",
			AuthorType:       "public",
			Reaction:         "down",
			Subject:          "Install timing",
			Body:             "Need confirmation on timing for regional delivery.",
			Status:           "flagged",
			ModerationReason: "Needs logistics verification.",
			CreatedAt:        "2026-03-25T00:00:00Z",
			UpdatedAt:        "2026-03-25T00:00:00Z",
		},
	}
}

func sampleWarehouseOpsRecords() []warehouseOpsRecord {
	return []warehouseOpsRecord{
		{
			ID:            "illinois-hub",
			Slug:          "illinois-hub",
			Name:          "Illinois Hub",
			Region:        "Central",
			ServiceLevel:  "4-6 days",
			PublicSummary: "Balancing lane.",
			Pressure:      "Balancing lane",
			Staffing:      "84% staffed",
			Backlog:       "2 dock queues",
			Focus:         "Balance inbound accessories.",
			Available:     24,
			Inbound:       5,
			RiskCount:     1,
		},
		{
			ID:            "new-jersey-hub",
			Slug:          "new-jersey-hub",
			Name:          "New Jersey Hub",
			Region:        "East coast",
			ServiceLevel:  "2-4 days",
			PublicSummary: "Promise lane.",
			Pressure:      "Promise risk",
			Staffing:      "88% staffed",
			Backlog:       "2 blocked receipts",
			Focus:         "Protect east-coast windows.",
			Available:     18,
			Inbound:       6,
			RiskCount:     2,
		},
	}
}

func sampleProductAdminCards() []productAdminCard {
	return []productAdminCard{
		{
			SKU:            "frame-desk",
			Slug:           "frame-desk",
			Title:          "Frame Desk",
			Category:       "desks",
			PriceCents:     189900,
			Status:         "low_stock",
			Finish:         "Graphite oak",
			Summary:        "Flagship modular desk.",
			Details:        "Detailed merchandising copy.",
			SEOTitle:       "Atlas Frame Desk",
			SEODescription: "Availability-aware workspace desk.",
			WarehouseID:    "new-jersey-hub",
			WarehouseName:  "New Jersey Hub",
			Available:      3,
			Inbound:        4,
			Volume:         7,
			HubCount:       2,
			UpdatedAt:      "2026-03-25T00:00:00Z",
		},
		{
			SKU:            "studio-console",
			Slug:           "studio-console",
			Title:          "Studio Console",
			Category:       "storage",
			PriceCents:     249900,
			Status:         "in_stock",
			Finish:         "Walnut ember",
			Summary:        "Creative-team console.",
			Details:        "Low-profile production console.",
			SEOTitle:       "Atlas Studio Console",
			SEODescription: "Creative-team storage system.",
			WarehouseID:    "illinois-hub",
			WarehouseName:  "Illinois Hub",
			Available:      7,
			Inbound:        3,
			Volume:         10,
			HubCount:       2,
			UpdatedAt:      "2026-03-25T00:00:00Z",
		},
	}
}
