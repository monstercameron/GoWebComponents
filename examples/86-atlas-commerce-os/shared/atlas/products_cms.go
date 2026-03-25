package atlas

import (
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type productCMSFilters struct {
	Search   string `json:"q"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Sort     string `json:"sort"`
}

type productCMSPageData struct {
	Items    []productAdminCard `json:"items"`
	Filters  productCMSFilters  `json:"filters"`
	Total    int                `json:"total"`
	Editable bool               `json:"editable"`
}

type productAdminCard struct {
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

func productsCMSContent(parsePayload Payload) ui.Node {
	parsePage := decode[productCMSPageData](pageData(parsePayload))
	if parsePage.Total == 0 {
		parsePage.Total = len(parsePage.Items)
	}
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.22fr)_minmax(24rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			productCMSSummaryBand(parsePage),
			catalogCMSFilterForm(parsePage.Filters),
			productCMSBulkActionRail(),
			productCMSTableShell(parsePage.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5", ID: "create-product"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Create product")),
					html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Start new SKUs from the route rail while the main pane stays focused on dense catalog review and direct edit handoffs.")),
				),
				productCreateForm(parsePayload),
			),
			internalWorkflowSection("Product merchandising flows", "Use the product workspace for item creation, marketing copy, SEO, and catalog readiness, then branch into warehouse routes only when the work becomes lane-specific.",
				internalWorkflowCard("Flow 1", "Add new item", "Start a new Atlas SKU with baseline pricing, merchandising copy, and the first warehouse assignment already in the create form.", "/app/products#create-product"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the freshest product records first when merchandising language, summaries, or SEO need cleanup.", "/app/products?sort=updated"),
				internalWorkflowCard("Flow 3", "Review warehouse operations", "Jump into warehouse-native item management when a product change becomes a lane or replenishment task.", "/app/warehouses"),
				internalWorkflowCard("Flow 4", "Check buyer feedback", "Use real customer questions to decide whether copy, availability, or pricing context needs work.", "/app/comments"),
			),
			featureCard("Why this matters", "Atlas keeps the product workspace focused on structured CRUD: a reviewable catalog table in the main pane, a creation rail on the side, and direct handoffs into inventory or warehouse workflows when merchandising work becomes operational."),
		),
	)
}

func productCMSSummaryBand(parsePage productCMSPageData) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Product table shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("The products route should read like an operator table first: visible catalog counts, explicit stock posture, and direct edit or warehouse handoffs instead of a merch-card gallery.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Products", fmt.Sprintf("%d live", parsePage.Total)),
			statCard("Total volume", fmt.Sprintf("%d units", totalProductVolume(parsePage.Items))),
			statCard("Low stock", fmt.Sprintf("%d flagged", countLowStockProducts(parsePage.Items))),
			statCard("Warehouses", fmt.Sprintf("%d active", productWarehouseCount(parsePage.Items))),
		),
	)
}

func productCMSBulkActionRail() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("CRUD action rail")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Bulk motion stays non-destructive in the first milestone. These route actions are for triage, merchandising cleanup, and handoff into warehouse or inventory work.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
			productBulkActionCard("Freshest edits", "Review recently updated products first when copy or SEO needs cleanup.", "/app/products?sort=updated"),
			productBulkActionCard("Low-stock review", "Hand off constrained product posture into inventory follow-up before buyer promise slips.", "/app/inventory?status=promise_risk"),
			productBulkActionCard("Warehouse follow-up", "Move from product review into warehouse-native item management when the issue becomes lane-specific.", "/app/warehouses"),
		),
	)
}

func productBulkActionCard(parseTitle, parseCopy, parseHref string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: "grid gap-3 " + internalAccentSurfaceClass() + " p-4 transition hover:border-cyan-300/55 hover:bg-[linear-gradient(180deg,rgba(10,24,42,0.96),rgba(7,14,26,0.99))]"},
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Open workflow")),
	)
}

func productCMSTableShell(parseItems []productAdminCard) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, productCMSTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Td(html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 8}}, html.Text("No products match the current filter set. Clear filters or create a new Atlas SKU from the route rail.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Catalog CRUD table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review product posture, pricing, warehouse anchor, and edit handoffs in one dense table instead of scanning product cards.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d rows", len(parseItems)))),
		),
		html.Div(html.Props{Class: internalTableContainerClass()},
			html.Table(html.Props{Class: "min-w-full border-collapse text-left"},
				html.Thead(html.Props{},
					html.Tr(html.Props{Class: "bg-slate-950/80"},
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Product")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Warehouse")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Status")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Available")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Inbound")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Volume")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Price")),
						html.Th(html.Props{Class: internalTableHeaderCellClass()}, html.Text("Actions")),
					),
				),
				html.Tbody(html.Props{}, parseRows...),
			),
		),
	)
}

func productCMSTableRow(parseItem productAdminCard) ui.Node {
	return html.Tr(html.Props{Class: internalTableRowClass() + " align-top"},
		html.Td(html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/products/" + parseItem.Slug, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(parseItem.Title)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseItem.SKU+" | "+parseItem.Category)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Summary)),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.2em] text-slate-400"}, html.Text(fmt.Sprintf("%d hubs", maxProductHubCount(parseItem.HubCount)))),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4"}, productCMSStatusBadge(parseItem.Status)),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", parseItem.Inbound))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", productVolume(parseItem)))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(formatPrice(parseItem.PriceCents))),
		html.Td(html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/products/" + parseItem.Slug, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Edit product")),
				html.A(html.Props{Href: warehouseItemProfileHref(parseItem), Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open warehouse lane")),
			),
		),
	)
}

func productCMSStatusBadge(parseStatus string) ui.Node {
	parseTone := "border-white/12 bg-white/8 text-slate-200"
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "in_stock":
		parseTone = "border-emerald-400/25 bg-emerald-400/10 text-emerald-200"
	case "low_stock":
		parseTone = "border-amber-400/25 bg-amber-400/10 text-amber-200"
	case "draft":
		parseTone = "border-cyan-300/25 bg-cyan-400/10 text-cyan-200"
	case "archived":
		parseTone = "border-slate-600 bg-slate-800/70 text-slate-300"
	}
	return html.Span(html.Props{Class: "inline-flex rounded-full border px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] " + parseTone}, html.Text(strings.ReplaceAll(parseStatus, "_", " ")))
}

func productWarehouseCount(parseItems []productAdminCard) int {
	parseSeen := map[string]struct{}{}
	for _, parseItem := range parseItems {
		parseId := strings.TrimSpace(parseItem.WarehouseID)
		if parseId == "" {
			continue
		}
		parseSeen[parseId] = struct{}{}
	}
	return len(parseSeen)
}

func maxProductHubCount(parseCount int) int {
	if parseCount > 0 {
		return parseCount
	}
	return 1
}

func productEditorContent(parsePayload Payload) ui.Node {
	parseItem := decode[productAdminCard](pageData(parsePayload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.02fr)_minmax(24rem,0.92fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			productEditorPreviewHero(parseItem),
			productEditorPreviewSurface(parseItem),
			productEditorWarehouseContextCard(parseItem),
			productEditorQuickTasks(),
			internalWorkflowSection("Product editor flows", "Finish the product task here, then jump directly into the next operational route instead of backing out through unrelated screens.",
				internalWorkflowCard("Flow 1", "Back to product list", "Return to the merchandising list when you need to move from one product record to the next update.", "/app/products"),
				internalWorkflowCard("Flow 2", "Manage warehouse item", "Open the warehouse-native item profile when this product needs stock edits or replenishment controls.", warehouseItemProfileHref(parseItem)),
				internalWorkflowCard("Flow 3", "Review warehouse lanes", "Open the SKU lane view when the product change needs cross-warehouse inventory context.", "/app/inventory/"+parseItem.SKU),
				internalWorkflowCard("Flow 4", "Review buyer feedback", "Use the comment queue to validate whether the copy update resolved the original customer confusion.", "/app/comments"),
			),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			productUpdateForm(parseItem, parsePayload),
			productDeleteForm(parseItem, parsePayload),
		),
	)
}

func productEditorPreviewHero(parseItem productAdminCard) ui.Node {
	return html.Div(html.Props{Class: internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
			productCMSStatusBadge(parseItem.Status),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(parseItem.SKU+" | "+parseItem.Category)),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
		),
		html.H2(html.Props{Class: "mt-4 text-3xl font-black tracking-[-0.03em] text-white"}, html.Text(parseItem.Title)),
		html.P(html.Props{Class: "mt-4 max-w-3xl text-base leading-7 text-slate-300"}, html.Text(parseItem.Details)),
		html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-3"},
			statCard("Price", formatPrice(parseItem.PriceCents)),
			statCard("Available", fmt.Sprintf("%d units", parseItem.Available)),
			statCard("Inbound", fmt.Sprintf("%d units", parseItem.Inbound)),
		),
	)
}

func productEditorPreviewSurface(parseItem productAdminCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Preview surface")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep the buyer-facing product story visible while editing so merch changes are reviewed in the same pass as status, warehouse anchor, and SEO updates.")),
		),
		html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-5"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(parseItem.Title)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseItem.Summary)),
			),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"},
				infoRow("Finish", parseItem.Finish),
				infoRow("SEO title", parseItem.SEOTitle),
			),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.18em] text-slate-400"}, html.Text(fallback(parseItem.SEODescription, "SEO description not set"))),
		),
	)
}

func productEditorWarehouseContextCard(parseItem productAdminCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Warehouse context")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("This editor should keep warehouse impact visible so a copy or status change can pivot immediately into lane review when the issue becomes operational.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			statCard("Warehouse", fallback(parseItem.WarehouseName, parseItem.WarehouseID)),
			statCard("Volume", fmt.Sprintf("%d units", productVolume(parseItem))),
			statCard("Hubs", fmt.Sprintf("%d", maxProductHubCount(parseItem.HubCount))),
		),
		html.A(html.Props{Href: warehouseItemProfileHref(parseItem), Class: "inline-flex w-fit items-center rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-300/16"}, html.Text("Open warehouse item profile")),
	)
}

func productEditorQuickTasks() ui.Node {
	return internalWorkflowSection("Quick tasks", "Keep the common admin handoffs inside the product detail page so the next action is visible while you edit the product record.",
		internalWorkflowCard("Task 1", "Add item", "Create another catalog item without backing out of the current product workflow.", "/app/products#create-product"),
		internalWorkflowCard("Task 2", "Update copy", "Return to the freshest product records when merchandising language or SEO needs another pass.", "/app/products?sort=updated"),
		internalWorkflowCard("Task 3", "Fix stock risk", "Jump into the inventory pressure view when this product issue is really a stock problem.", "/app/inventory?status=promise_risk"),
		internalWorkflowCard("Task 4", "Order more units", "Open purchase-order planning when the product change needs vendor replenishment.", "/app/purchase-orders"),
		internalWorkflowCard("Task 5", "Receiving", "Check inbound receiving after replenishment starts moving toward the warehouse operations routes.", "/app/receiving"),
		internalWorkflowCard("Task 6", "Buyer inbox", "Review customer questions tied to this product before closing the edit pass.", "/app/comments"),
	)
}

func catalogCMSFilterForm(parseFilters productCMSFilters) ui.Node {
	return html.Form(html.Props{Action: "/app/products", Method: "get", Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5 lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.7fr))_auto] lg:items-end"},
		cmsTextInput("q", "Search products", parseFilters.Search),
		cmsSelectInput("category", "Category", parseFilters.Category, []optionItem{{"all", "All categories"}, {"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}),
		cmsSelectInput("status", "Status", parseFilters.Status, []optionItem{{"all", "All statuses"}, {"in_stock", "In stock"}, {"low_stock", "Low stock"}, {"draft", "Draft"}, {"archived", "Archived"}}),
		cmsSelectInput("sort", "Sort", parseFilters.Sort, []optionItem{{"updated", "Recently updated"}, {"volume", "Highest volume"}, {"price", "Highest price"}, {"status", "Status"}}),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text("Filter")),
	)
}

func productCreateForm(parsePayload Payload) ui.Node {
	return productCreateFormWithOptions(parsePayload, productFormOptions{LockedWarehouseID: fallback(parsePayload.Preferences.DefaultWarehouse, "new-jersey-hub")})
}

type productFormOptions struct {
	LockedWarehouseID string
	ReturnWarehouseID string
	IntroLabel        string
	SubmitLabel       string
	DeleteLabel       string
	DeleteCopy        string
}

type productEditorFormState struct {
	SKU             string
	Slug            string
	Title           string
	Category        string
	PriceCents      string
	Status          string
	Finish          string
	WarehouseID     string
	Available       string
	Inbound         string
	Summary         string
	Details         string
	SEOTitle        string
	SEODescription  string
	ReturnWarehouse string
}

func productCreateFormWithOptions(parsePayload Payload, parseOptions productFormOptions) ui.Node {
	parseWarehouseID := fallback(parseOptions.LockedWarehouseID, "new-jersey-hub")
	parseIntroLabel := fallback(parseOptions.IntroLabel, "Add catalog item")
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(productEditorFormState{
			Category:        "desks",
			PriceCents:      "189900",
			Status:          "in_stock",
			Finish:          "Graphite oak",
			WarehouseID:     parseWarehouseID,
			Available:       "8",
			Inbound:         "3",
			Summary:         "Short merchandising summary for the storefront card.",
			Details:         "Longer product detail copy for buyers and internal review.",
			SEOTitle:        "Atlas New Product",
			SEODescription:  "Search-friendly description for the Atlas storefront route.",
			ReturnWarehouse: parseOptions.ReturnWarehouseID,
		})
		parseValue := parseForm.Get()
		parseChildren := []ui.Node{
			cmsBoundTextInput("sku", "SKU", parseValue.SKU, "SKU", parseForm),
			cmsBoundTextInput("slug", "Slug", parseValue.Slug, "Slug", parseForm),
			cmsBoundTextInput("title", "Title", parseValue.Title, "Title", parseForm),
			cmsBoundSelectInput("category", "Category", parseValue.Category, "Category", productCategoryOptions(), parseForm),
			cmsBoundNumberInput("price_cents", "Price cents", parseValue.PriceCents, "PriceCents", parseForm),
			cmsBoundSelectInput("status", "Status", parseValue.Status, "Status", productStatusOptions(), parseForm),
			cmsBoundTextInput("finish", "Finish", parseValue.Finish, "Finish", parseForm),
			warehouseScopedBoundField("warehouse_id", parseValue.WarehouseID, parseOptions.LockedWarehouseID, "WarehouseID", warehouseOptions(), parseForm),
			cmsBoundNumberInput("available", "Available volume", parseValue.Available, "Available", parseForm),
			cmsBoundNumberInput("inbound", "Inbound volume", parseValue.Inbound, "Inbound", parseForm),
			cmsBoundTextarea("summary", "Summary", parseValue.Summary, "Summary", parseForm),
			cmsBoundTextarea("details", "Details", parseValue.Details, "Details", parseForm),
			cmsBoundTextInput("seo_title", "SEO title", parseValue.SEOTitle, "SEOTitle", parseForm),
			cmsBoundTextarea("seo_description", "SEO description", parseValue.SEODescription, "SEODescription", parseForm),
			html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(parseOptions.SubmitLabel, "Create product"))),
		}
		if strings.TrimSpace(parseValue.ReturnWarehouse) != "" {
			parseChildren = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: parseValue.ReturnWarehouse})}, parseChildren...)
		}
		return html.Form(html.Props{Action: "/api/app/products", Method: "post", Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
			append([]ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseIntroLabel))}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)...,
		)
	})
}

func productUpdateForm(parseItem productAdminCard, parsePayload Payload) ui.Node {
	return productUpdateFormWithOptions(parseItem, parsePayload, productFormOptions{LockedWarehouseID: parseItem.WarehouseID})
}

func productUpdateFormWithOptions(parseItem productAdminCard, parsePayload Payload, parseOptions productFormOptions) ui.Node {
	parseWarehouseID := fallback(parseOptions.LockedWarehouseID, parseItem.WarehouseID)
	parseIntroLabel := fallback(parseOptions.IntroLabel, "Edit catalog item")
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(productEditorFormState{
			SKU:             parseItem.SKU,
			Slug:            parseItem.Slug,
			Title:           parseItem.Title,
			Category:        parseItem.Category,
			PriceCents:      fmt.Sprintf("%d", parseItem.PriceCents),
			Status:          parseItem.Status,
			Finish:          parseItem.Finish,
			WarehouseID:     parseWarehouseID,
			Available:       fmt.Sprintf("%d", parseItem.Available),
			Inbound:         fmt.Sprintf("%d", parseItem.Inbound),
			Summary:         parseItem.Summary,
			Details:         parseItem.Details,
			SEOTitle:        parseItem.SEOTitle,
			SEODescription:  parseItem.SEODescription,
			ReturnWarehouse: parseOptions.ReturnWarehouseID,
		})
		parseValue := parseForm.Get()
		parsePrevious := ui.UsePrevious(parseValue)
		parseHiddenFields := []ui.Node{
			html.Input(html.Props{Type: "hidden", Name: "current_warehouse_id", Value: parseWarehouseID}),
			html.Input(html.Props{Type: "hidden", Name: "sku", Value: parseValue.SKU}),
		}
		parseMetadataSection := productEditorFormSection("Metadata form", "Keep product identity, category, commercial status, and finish settings grouped together.", []ui.Node{
			cmsReadOnlyField("SKU", parseValue.SKU),
			cmsBoundTextInput("slug", "Slug", parseValue.Slug, "Slug", parseForm),
			cmsBoundTextInput("title", "Title", parseValue.Title, "Title", parseForm),
			cmsBoundSelectInput("category", "Category", parseValue.Category, "Category", productCategoryOptions(), parseForm),
			cmsBoundNumberInput("price_cents", "Price cents", parseValue.PriceCents, "PriceCents", parseForm),
			cmsBoundSelectInput("status", "Status", parseValue.Status, "Status", productStatusOptions(), parseForm),
			cmsBoundTextInput("finish", "Finish", parseValue.Finish, "Finish", parseForm),
		}...)
		parseWarehouseSection := productEditorFormSection("Warehouse context", "Warehouse anchor, available units, and inbound posture stay together so merchandising edits can still respect the operating lane.", []ui.Node{
			warehouseScopedBoundField("warehouse_id", parseValue.WarehouseID, parseOptions.LockedWarehouseID, "WarehouseID", warehouseOptions(), parseForm),
			cmsBoundNumberInput("available", "Available volume", parseValue.Available, "Available", parseForm),
			cmsBoundNumberInput("inbound", "Inbound volume", parseValue.Inbound, "Inbound", parseForm),
		}...)
		parseMerchSection := productEditorFormSection("Merchandising copy", "Summary, long-form detail, and SEO fields stay in one content block so the public-facing product story is edited coherently.", []ui.Node{
			cmsBoundTextarea("summary", "Summary", parseValue.Summary, "Summary", parseForm),
			cmsBoundTextarea("details", "Details", parseValue.Details, "Details", parseForm),
			cmsBoundTextInput("seo_title", "SEO title", parseValue.SEOTitle, "SEOTitle", parseForm),
			cmsBoundTextarea("seo_description", "SEO description", parseValue.SEODescription, "SEODescription", parseForm),
		}...)
		parseChildren := []ui.Node{
			parseMetadataSection,
			parseWarehouseSection,
			parseMerchSection,
			html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(parseOptions.SubmitLabel, "Save product"))),
		}
		if parseChangeSummary := productDraftChangeSummary(parseValue, parsePrevious); parseChangeSummary != nil {
			parseChildren = append([]ui.Node{parseChangeSummary}, parseChildren...)
		}
		if strings.TrimSpace(parseValue.ReturnWarehouse) != "" {
			parseHiddenFields = append(parseHiddenFields, html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: parseValue.ReturnWarehouse}))
		}
		return html.Form(html.Props{
			Action: "/api/app/products/" + parseItem.Slug + "/update",
			Method: "post",
			Class:  "grid gap-4 " + internalSurfaceCardClass() + " p-5",
			Data: map[string]string{
				"atlas-dirty-guard":   "product-editor",
				"atlas-dirty-message": "Leave the product editor and discard unsaved Atlas product changes?",
			},
		},
			append([]ui.Node{
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseIntroLabel)),
			}, prependCSRFToken(parsePayload.CSRF, append(parseHiddenFields, parseChildren...)...)...)...,
		)
	})
}

func productEditorFormSection(parseTitle, parseCopy string, parseChildren ...ui.Node) ui.Node {
	parseSectionChildren := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseTitle)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseCopy)),
		),
	}
	parseSectionChildren = append(parseSectionChildren, parseChildren...)
	return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"}, parseSectionChildren...)
}

func productDraftChangeSummary(parseCurrent productEditorFormState, parsePrevious ui.Previous[productEditorFormState]) ui.Node {
	if !parsePrevious.Ok() {
		return nil
	}
	parsePrior := parsePrevious.Get()
	parseSummary := ""
	switch {
	case strings.TrimSpace(parsePrior.Title) != strings.TrimSpace(parseCurrent.Title):
		parseSummary = "Title changed from " + fallback(parsePrior.Title, "untitled item") + " to " + fallback(parseCurrent.Title, "untitled item")
	case strings.TrimSpace(parsePrior.Status) != strings.TrimSpace(parseCurrent.Status):
		parseSummary = "Status changed from " + fallback(parsePrior.Status, "unset") + " to " + fallback(parseCurrent.Status, "unset")
	case strings.TrimSpace(parsePrior.PriceCents) != strings.TrimSpace(parseCurrent.PriceCents):
		parseSummary = "Price cents changed from " + fallback(parsePrior.PriceCents, "0") + " to " + fallback(parseCurrent.PriceCents, "0")
	case strings.TrimSpace(parsePrior.Summary) != strings.TrimSpace(parseCurrent.Summary):
		parseSummary = "Summary copy was updated in the current draft."
	}
	if parseSummary == "" {
		return nil
	}
	return html.Div(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Recent draft change")),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(parseSummary)),
	)
}

func productDeleteForm(parseItem productAdminCard, parsePayload Payload) ui.Node {
	return productDeleteFormWithOptions(parseItem, parsePayload, productFormOptions{})
}

func productDeleteFormWithOptions(parseItem productAdminCard, parsePayload Payload, parseOptions productFormOptions) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(fallback(parseOptions.DeleteCopy, "Deleting a product removes its catalog row and cascades dependent records in the demo database. Use this only when you want the route gone."))),
		html.Button(html.Props{Type: "submit", Class: "rounded-full border border-rose-300/30 bg-rose-400/10 px-5 py-3 text-sm font-semibold text-rose-100"}, html.Text(fallback(parseOptions.DeleteLabel, "Delete product"))),
	}
	if strings.TrimSpace(parseOptions.ReturnWarehouseID) != "" {
		parseChildren = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: parseOptions.ReturnWarehouseID})}, parseChildren...)
	}
	return html.Form(html.Props{Action: "/api/app/products/" + parseItem.Slug + "/delete", Method: "post", Class: "grid gap-4 rounded-[1.5rem] border border-rose-300/20 bg-rose-400/5 p-5"},
		append([]ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-rose-200"}, html.Text("Retire catalog item"))}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)...,
	)
}

func warehouseScopedField(parseName string, parseValue string, parseLockedValue string, parseOptions []optionItem) ui.Node {
	if strings.TrimSpace(parseLockedValue) != "" {
		return html.Input(html.Props{Type: "hidden", Name: parseName, Value: parseValue})
	}
	return cmsSelectInput(parseName, "Warehouse", parseValue, parseOptions)
}

func warehouseScopedBoundField[T any](parseName string, parseValue string, parseLockedValue string, parseField string, parseOptions []optionItem, parseForm ui.Form[T]) ui.Node {
	if strings.TrimSpace(parseLockedValue) != "" {
		return html.Input(html.Props{Type: "hidden", Name: parseName, Value: parseValue})
	}
	return cmsBoundSelectInput(parseName, "Warehouse", parseValue, parseField, parseOptions, parseForm)
}

func productListMetric(parseLabel string, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-1 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm lg:bg-transparent lg:px-0 lg:py-0"},
		html.P(html.Props{Class: "text-[0.65rem] font-semibold uppercase tracking-[0.22em] text-slate-500"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseValue)),
	)
}

func productVolume(parseItem productAdminCard) int {
	if parseItem.Volume > 0 {
		return parseItem.Volume
	}
	return parseItem.Available + parseItem.Inbound
}

func warehouseItemProfileHref(parseItem productAdminCard) string {
	if strings.TrimSpace(parseItem.WarehouseID) != "" {
		return "/app/warehouses/" + parseItem.WarehouseID + "/items/" + parseItem.SKU
	}
	return "/app/inventory/" + parseItem.SKU
}

func totalProductVolume(parseItems []productAdminCard) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += productVolume(parseItem)
	}
	return parseTotal
}

type optionItem struct {
	Value string
	Label string
}

func cmsTextInput(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsBoundTextInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsTransitionBoundTextInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T], parseTransition atlasTransition) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{
			ID:    parseId,
			Name:  parseName,
			Value: parseValue,
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				atlasSetFormFieldInTransition(parseForm, parseField, parseEvent.GetValue())
			}),
			Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			Raw: map[string]interface{}{
				"aria-labelledby": parseId + "-label",
				"data-transition": parseTransition.Pending(),
			},
		}),
	)
}

func cmsReadOnlyField(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}, html.Text(parseValue)),
	)
}

func cmsNumberInput(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Type: "number", Name: parseName, Value: parseValue, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsBoundNumberInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Type: "number", Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsTextarea(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func cmsBoundTextarea[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func cmsSelectInput(parseName, parseLabel, parseValue string, parseOptions []optionItem) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "all")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Select(html.Props{ID: parseId, Name: parseName, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}, parseChildren...),
	)
}

func cmsBoundSelectInput[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "all")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Select(html.Props{ID: parseId, Name: parseName, OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": parseId + "-label"}}, parseChildren...),
	)
}

func cmsTransitionBoundSelectInput[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T], parseTransition atlasTransition) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "all")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Select(html.Props{
			ID:   parseId,
			Name: parseName,
			OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) {
				atlasSetFormFieldInTransition(parseForm, parseField, parseEvent.GetValue())
			}),
			Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			Raw: map[string]interface{}{
				"aria-labelledby": parseId + "-label",
				"data-transition": parseTransition.Pending(),
			},
		}, parseChildren...),
	)
}

func productCategoryOptions() []optionItem {
	return []optionItem{{"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}
}

func productStatusOptions() []optionItem {
	return []optionItem{{"in_stock", "In stock"}, {"low_stock", "Low stock"}, {"draft", "Draft"}, {"archived", "Archived"}}
}

func warehouseOptions() []optionItem {
	return []optionItem{{"new-jersey-hub", "New Jersey Hub"}, {"illinois-hub", "Illinois Hub"}, {"nevada-hub", "Nevada Hub"}}
}

func countLowStockProducts(parseItems []productAdminCard) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), "low_stock") || parseItem.Available <= 3 {
			parseCount++
		}
	}
	return parseCount
}

func storeCatalogControls(parseForm ui.Form[atlasListFilterState], isSyncing bool, parseSubmit ui.Handler) ui.Node {
	parseValue := parseForm.Get()
	return html.Form(html.Props{Action: "/shop", Method: "get", OnSubmit: parseSubmit, Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-5 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.75fr))_auto] lg:items-end"},
		publicCatalogBoundInput("q", "Search", parseValue.Query, "Query", parseForm),
		publicCatalogBoundSelect("category", "Category", parseValue.Category, "Category", []optionItem{{"all", "All categories"}, {"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}, parseForm),
		publicCatalogBoundSelect("warehouse", "Warehouse", parseValue.Warehouse, "Warehouse", []optionItem{{"all", "All warehouses"}, {"new-jersey-hub", "New Jersey Hub"}, {"illinois-hub", "Illinois Hub"}, {"nevada-hub", "Nevada Hub"}}, parseForm),
		publicCatalogBoundSelect("sort", "Sort", parseValue.Sort, "Sort", []optionItem{{"", "Featured"}, {"warehouse", "SKU / warehouse"}}, parseForm),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(atlasFilterSubmitLabel(isSyncing, "Apply"))),
	)
}

func publicCatalogInput(parseName, parseLabel, parseValue string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Input(html.Props{Name: parseName, Value: parseValue, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
	)
}

func publicCatalogBoundInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Input(html.Props{Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
	)
}

func publicCatalogSelect(parseName, parseLabel, parseValue string, parseOptions []optionItem) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Select(html.Props{Name: parseName, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}, parseChildren...),
	)
}

func publicCatalogBoundSelect[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T]) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Select(html.Props{Name: parseName, OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}, parseChildren...),
	)
}

func filterCatalogItems(parseItems []productCard, parseFilters atlasListFilterState) []productCard {
	parseNeedle := atlasNormalizedFilterValue(parseFilters.Query)
	parseCategory := atlasNormalizedFilterValue(parseFilters.Category)
	parseWarehouse := atlasNormalizedFilterValue(parseFilters.Warehouse)
	parseFiltered := make([]productCard, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if parseNeedle != "" {
			parseHaystack := atlasNormalizedFilterValue(parseItem.Title + " " + parseItem.SKU + " " + parseItem.Category + " " + parseItem.Summary + " " + parseItem.WarehouseName)
			if !strings.Contains(parseHaystack, parseNeedle) {
				continue
			}
		}
		if parseCategory != "" && parseCategory != "all" && !strings.EqualFold(strings.TrimSpace(parseItem.Category), parseCategory) {
			continue
		}
		if parseWarehouse != "" && parseWarehouse != "all" && !strings.EqualFold(strings.TrimSpace(parseItem.WarehouseID), parseWarehouse) {
			continue
		}
		parseFiltered = append(parseFiltered, parseItem)
	}
	if atlasNormalizedFilterValue(parseFilters.Sort) == "warehouse" {
		sort.SliceStable(parseFiltered, func(parseLeft, parseRight int) bool {
			if parseFiltered[parseLeft].WarehouseID == parseFiltered[parseRight].WarehouseID {
				if parseFiltered[parseLeft].SKU == parseFiltered[parseRight].SKU {
					return parseFiltered[parseLeft].Title < parseFiltered[parseRight].Title
				}
				return parseFiltered[parseLeft].SKU < parseFiltered[parseRight].SKU
			}
			return parseFiltered[parseLeft].WarehouseID < parseFiltered[parseRight].WarehouseID
		})
	}
	return parseFiltered
}
