package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestAppRenderMatrixAcrossPublicAndInternalRoutes(parseT *testing.T) {
	parseBaseInventoryRows := sampleInventoryRows()
	parseBaseProducts := sampleProductCards()
	parseBaseWarehouses := sampleWarehouseCards()
	parseBaseComments := sampleCommentRecords()
	parseBaseTransfers := sampleTransferRecords()
	parseBaseOrders := samplePurchaseOrders()
	parseBaseReceiving := sampleReceivingRecords()
	parseBaseWarehouseOps := sampleWarehouseOpsRecords()
	parseBaseProductAdmins := sampleProductAdminCards()

	parseCases := []struct {
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
				Items:    parseBaseProducts,
				Total:    len(parseBaseProducts),
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
				Product:  parseBaseProducts[0],
				Comments: parseBaseComments,
			},
		},
		{
			name: "warehouses-directory",
			path: RouteWarehouses,
			page: warehouseDirectoryPage{
				Items: parseBaseWarehouses,
			},
		},
		{
			name: "warehouse-public-detail",
			path: RouteWarehousePublicDetail,
			page: warehouseDetailPage{
				Warehouse: parseBaseWarehouses[0],
				Products:  parseBaseProducts,
			},
		},
		{
			name: "availability",
			path: RouteWarehouseAvailability,
			page: availabilityPage{
				Warehouse: parseBaseWarehouses[0],
				Product:   parseBaseProducts[0],
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
				Transfers: parseBaseTransfers,
				Receiving: parseBaseReceiving,
				Comments:  parseBaseComments,
				Orders:    parseBaseOrders,
			},
		},
		{
			name: "inventory",
			path: RouteInventory,
			page: inventoryCMSPage{
				Summary: sampleSummary("Inventory shell"),
				Items:   parseBaseInventoryRows,
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
				Rows:  parseBaseInventoryRows,
			},
		},
		{
			name: "sku-threshold-history",
			path: RouteSKUThresholdHistory,
			page: inventoryDetailPage{
				SKU:   "frame-desk",
				Title: "Frame Desk",
				Rows:  parseBaseInventoryRows,
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
				Items:   parseBaseWarehouseOps,
			},
		},
		{
			name: "warehouse-detail-internal",
			path: RouteWarehouseDetail,
			page: warehouseInventoryDetailPage{
				Summary:   sampleSummary("Facility detail"),
				Warehouse: parseBaseWarehouseOps[0],
				Inventory: parseBaseInventoryRows,
				Orders:    parseBaseOrders,
				Filters: map[string]string{
					"status": "promise_risk",
				},
			},
		},
		{
			name: "warehouse-item-detail",
			path: RouteWarehouseItemDetail,
			page: warehouseInventoryItemDetailPage{
				Warehouse: parseBaseWarehouseOps[0],
				Item:      parseBaseInventoryRows[0],
				Product:   parseBaseProductAdmins[0],
				Network:   parseBaseInventoryRows,
				Orders:    parseBaseOrders,
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
				Items: parseBaseTransfers,
			},
		},
		{
			name: "transfer-detail",
			path: RouteTransferDetail,
			page: transferDetailPage{
				Transfer: parseBaseTransfers[0],
				Lines: []transferLineRecord{
					{ID: "tr-line-1", TransferID: parseBaseTransfers[0].ID, ProductSKU: "frame-desk", Quantity: 5},
				},
			},
		},
		{
			name: "purchase-orders",
			path: RoutePurchaseOrders,
			page: purchaseOrderList{
				Summary: sampleSummary("Purchase-order watch"),
				Items:   parseBaseOrders,
			},
		},
		{
			name: "purchase-order-detail",
			path: RoutePurchaseOrderDetail,
			page: purchaseOrderDetailPage{
				Order: parseBaseOrders[0],
				Lines: []purchaseOrderLineRecord{
					{ID: "po-line-1", PurchaseOrderID: parseBaseOrders[0].ID, ProductSKU: "frame-desk", Quantity: 12, ETA: "Thu 09:30", Status: "submitted"},
				},
			},
		},
		{
			name: "receiving",
			path: RouteReceiving,
			page: receivingList{
				Items: parseBaseReceiving,
			},
		},
		{
			name: "receiving-detail",
			path: RouteReceivingSessionDetail,
			page: receivingDetailPage{
				Session: parseBaseReceiving[0],
				Lines: []receivingLineRecord{
					{ID: "rcv-line-1", ReceivingSessionID: parseBaseReceiving[0].ID, ProductSKU: "frame-desk", ExpectedQuantity: 8, ActualQuantity: 7, DiscrepancyReason: "supplier short"},
				},
			},
		},
		{
			name: "comments",
			path: RouteComments,
			page: commentList{
				Summary: sampleSummary("Buyer inbox"),
				Items:   parseBaseComments,
			},
		},
		{
			name: "comments-moderation",
			path: RouteCommentsModeration,
			page: commentList{
				Summary: sampleSummary("Buyer inbox moderation"),
				Items:   parseBaseComments,
			},
		},
		{
			name: "comment-detail",
			path: RouteCommentDetail,
			page: commentList{
				Summary: sampleSummary("Buyer inbox detail"),
				Items:   parseBaseComments,
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
			name: "settings-appearance",
			path: RouteSettingsAppearance,
			page: settingsPage{
				Summary: sampleSummary("Appearance settings"),
			},
		},
		{
			name: "settings-locale",
			path: RouteSettingsLocale,
			page: settingsPage{
				Summary: sampleSummary("Locale settings"),
			},
		},
		{
			name: "settings-workspace-defaults",
			path: RouteSettingsWorkspaceDefaults,
			page: settingsPage{
				Summary: sampleSummary("Workspace defaults"),
			},
		},
		{
			name: "products-cms",
			path: "/app/products",
			page: productCMSPageData{
				Items: parseBaseProductAdmins,
				Filters: productCMSFilters{
					Search:   "frame",
					Category: "desks",
					Status:   "all",
					Sort:     "updated",
				},
				Total:    len(parseBaseProductAdmins),
				Editable: true,
			},
		},
		{
			name: "product-editor",
			path: "/app/products/frame-desk",
			page: parseBaseProductAdmins[0],
		},
		{
			name: "fallback-path",
			path: "/unknown-route",
			page: map[string]any{
				"headline": "fallback payload",
			},
		},
	}

	for _, parseTc := range parseCases {
		parsePayload := samplePayloadForRoute(parseTc.path, parseTc.page, parseTc.overlay)
		// App must be handed to the runtime as component + props, never invoked as
		// App(payload). Its body runs hooks (useAtlasAtom -> state.UseAtom), which
		// require the fiber the runtime is currently rendering; calling it directly
		// panics with GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT. This test used to do
		// exactly that and still passed, because the native hooks were stubs -
		// which is how client/main.go shipped the same mistake in a form that
		// white-screened the browser. Keeping the supported form here is what makes
		// this matrix real evidence that the app can boot.
		parseMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parsePayload))
		if parseErr != nil {
			parseT.Fatalf("%s render failed: %v", parseTc.name, parseErr)
		}
		if !strings.Contains(parseMarkup, "atlas-shell-root") {
			parseT.Fatalf("%s missing shell root markup", parseTc.name)
		}
	}
}

func TestAppRenderSpecialScreens(parseT *testing.T) {
	parseMockPayload := samplePayloadForRoute("/auth/mock-sign-in", mockSignInPage{
		Roles: []mockSignInRole{
			{Value: "inventory_manager", Label: "Inventory manager", Description: "Manages inventory workflows."},
			{Value: "buyer_support", Label: "Buyer support", Description: "Handles buyer inbox and moderation."},
		},
		Next:    RouteDashboard,
		Message: "Choose an Atlas role to continue.",
	}, nil)
	parseMockPayload.Route.Screen = "mock-sign-in"
	parseMockPayload.Route.Surface = "internal"
	parseMockPayload.User = nil

	parseMockMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parseMockPayload))
	if parseErr != nil {
		parseT.Fatalf("mock-sign-in render failed: %v", parseErr)
	}
	if !strings.Contains(parseMockMarkup, "Start session") {
		parseT.Fatalf("mock-sign-in markup missing expected action")
	}

	parseRecoveryPayload := samplePayloadForRoute("/app/recovery", recoveryPage{
		Title:         "Atlas recovery",
		Message:       "Route payload could not be resolved.",
		RecoveryHref:  RouteDashboard,
		RecoveryLabel: "Back to dashboard",
		Detail:        "Try reopening the route after data reload.",
	}, nil)
	parseRecoveryPayload.Route.Screen = "recovery"
	parseRecoveryPayload.Route.Surface = "internal"
	parseRecoveryPayload.User = &UserSession{
		ID:               "ops-1",
		DisplayName:      "Atlas Operator",
		Role:             "inventory_manager",
		DefaultWarehouse: "new-jersey-hub",
	}

	parseRecoveryMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parseRecoveryPayload))
	if parseErr != nil {
		parseT.Fatalf("recovery render failed: %v", parseErr)
	}
	if !strings.Contains(parseRecoveryMarkup, "Back to dashboard") {
		parseT.Fatalf("recovery markup missing expected recovery link")
	}
}

func samplePayloadForRoute(parsePath string, parsePage any, parseOverlay any) Payload {
	parseRoute, parseOk := RouteBootstrapForPath(parsePath)
	if !parseOk {
		parseMeta := MetadataForPath(parsePath)
		parseSurface := "public"
		if strings.HasPrefix(parsePath, "/app") {
			parseSurface = "internal"
		}
		parseRoute = RouteBootstrap{
			Path:        parsePath,
			Surface:     parseSurface,
			Screen:      "custom",
			Title:       parseMeta.Title,
			Description: parseMeta.Description,
			Canonical:   parseMeta.Canonical,
		}
	}
	if parseRoute.Path == "" {
		parseRoute.Path = parsePath
	}
	if parseRoute.Surface == "" {
		parseRoute.Surface = "public"
		if strings.HasPrefix(parsePath, "/app") {
			parseRoute.Surface = "internal"
		}
	}
	parseRoute.Query = map[string][]string{
		"status": {"promise_risk"},
		"sort":   {"updated"},
	}
	parseRoute.Params = map[string]string{
		"sku":         "frame-desk",
		"warehouseId": "illinois-hub",
		"productSlug": "frame-desk",
	}

	parsePayload := Payload{
		Route:       parseRoute,
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
	if strings.HasPrefix(parsePath, "/app") {
		parsePayload.User = &UserSession{
			ID:               "ops-1",
			DisplayName:      "Atlas Operator",
			Role:             "inventory_manager",
			DefaultWarehouse: "new-jersey-hub",
		}
		parsePayload.Route.Surface = "internal"
	}
	if parsePage != nil {
		parsePayload.Data["page"] = parsePage
		parsePayload.Requests["page"] = Request{
			Method: "GET",
			URL:    StartupRequestURL(parsePath, nil),
			Status: 200,
			Data: map[string]any{
				"page": parsePage,
			},
		}
	}
	if parseOverlay != nil {
		parsePayload.Data["overlay"] = parseOverlay
		parsePayload.Requests["overlay"] = Request{
			Method: "GET",
			URL:    StartupRequestURL(parsePath, nil) + "#overlay",
			Status: 200,
			Data: map[string]any{
				"overlay": parseOverlay,
			},
		}
	}
	return parsePayload
}

func sampleSummary(parseHeadline string) pageSummary {
	return pageSummary{
		Headline: parseHeadline,
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
