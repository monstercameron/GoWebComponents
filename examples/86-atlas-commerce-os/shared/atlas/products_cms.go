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

func productsCMSContent(payload Payload) ui.Node {
	page := decode[productCMSPageData](pageData(payload))
	if page.Total == 0 {
		page.Total = len(page.Items)
	}
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.22fr)_minmax(24rem,0.84fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			productCMSSummaryBand(page),
			catalogCMSFilterForm(page.Filters),
			productCMSBulkActionRail(),
			productCMSTableShell(page.Items),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5", ID: "create-product"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Create product")),
					html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Start new SKUs from the route rail while the main pane stays focused on dense catalog review and direct edit handoffs.")),
				),
				productCreateForm(payload),
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

func productCMSSummaryBand(page productCMSPageData) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Product table shell")),
			html.P(html.Props{Class: "max-w-3xl text-sm leading-6 text-slate-300"}, html.Text("The products route should read like an operator table first: visible catalog counts, explicit stock posture, and direct edit or warehouse handoffs instead of a merch-card gallery.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-4"},
			statCard("Products", fmt.Sprintf("%d live", page.Total)),
			statCard("Total volume", fmt.Sprintf("%d units", totalProductVolume(page.Items))),
			statCard("Low stock", fmt.Sprintf("%d flagged", countLowStockProducts(page.Items))),
			statCard("Warehouses", fmt.Sprintf("%d active", productWarehouseCount(page.Items))),
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

func productBulkActionCard(title, copy, href string) ui.Node {
	return html.A(html.Props{Href: href, Class: "grid gap-3 " + internalAccentSurfaceClass() + " p-4 transition hover:border-cyan-300/55 hover:bg-[linear-gradient(180deg,rgba(10,24,42,0.96),rgba(7,14,26,0.99))]"},
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Open workflow")),
	)
}

func productCMSTableShell(items []productAdminCard) ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, productCMSTableRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, html.Tr(html.Props{},
			html.Td(html.Props{Class: "px-4 py-6 text-sm text-slate-400", Raw: map[string]interface{}{"colSpan": 8}}, html.Text("No products match the current filter set. Clear filters or create a new Atlas SKU from the route rail.")),
		))
	}
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Catalog CRUD table")),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Review product posture, pricing, warehouse anchor, and edit handoffs in one dense table instead of scanning product cards.")),
			),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fmt.Sprintf("%d rows", len(items)))),
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
				html.Tbody(html.Props{}, rows...),
			),
		),
	)
}

func productCMSTableRow(item productAdminCard) ui.Node {
	return html.Tr(html.Props{Class: internalTableRowClass() + " align-top"},
		html.Td(html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/products/" + item.Slug, Class: "text-sm font-semibold text-white transition hover:text-cyan-200"}, html.Text(item.Title)),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.22em] text-slate-400"}, html.Text(item.SKU+" | "+item.Category)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Summary)),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "font-semibold text-white"}, html.Text(fallback(item.WarehouseName, item.WarehouseID))),
				html.P(html.Props{Class: "text-[0.68rem] uppercase tracking-[0.2em] text-slate-400"}, html.Text(fmt.Sprintf("%d hubs", maxProductHubCount(item.HubCount)))),
			),
		),
		html.Td(html.Props{Class: "px-4 py-4"}, productCMSStatusBadge(item.Status)),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Available))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", item.Inbound))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm text-slate-200"}, html.Text(fmt.Sprintf("%d", productVolume(item)))),
		html.Td(html.Props{Class: "px-4 py-4 text-sm font-semibold text-white"}, html.Text(formatPrice(item.PriceCents))),
		html.Td(html.Props{Class: "px-4 py-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.A(html.Props{Href: "/app/products/" + item.Slug, Class: "text-sm font-semibold text-cyan-200 transition hover:text-cyan-100"}, html.Text("Edit product")),
				html.A(html.Props{Href: warehouseItemProfileHref(item), Class: "text-sm text-slate-300 transition hover:text-white"}, html.Text("Open warehouse lane")),
			),
		),
	)
}

func productCMSStatusBadge(status string) ui.Node {
	tone := "border-white/12 bg-white/8 text-slate-200"
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "in_stock":
		tone = "border-emerald-400/25 bg-emerald-400/10 text-emerald-200"
	case "low_stock":
		tone = "border-amber-400/25 bg-amber-400/10 text-amber-200"
	case "draft":
		tone = "border-cyan-300/25 bg-cyan-400/10 text-cyan-200"
	case "archived":
		tone = "border-slate-600 bg-slate-800/70 text-slate-300"
	}
	return html.Span(html.Props{Class: "inline-flex rounded-full border px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] " + tone}, html.Text(strings.ReplaceAll(status, "_", " ")))
}

func productWarehouseCount(items []productAdminCard) int {
	seen := map[string]struct{}{}
	for _, item := range items {
		id := strings.TrimSpace(item.WarehouseID)
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
	}
	return len(seen)
}

func maxProductHubCount(count int) int {
	if count > 0 {
		return count
	}
	return 1
}

func productEditorContent(payload Payload) ui.Node {
	item := decode[productAdminCard](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 xl:grid-cols-[minmax(0,1.02fr)_minmax(24rem,0.92fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-5"},
			productEditorPreviewHero(item),
			productEditorPreviewSurface(item),
			productEditorWarehouseContextCard(item),
			productEditorQuickTasks(),
			internalWorkflowSection("Product editor flows", "Finish the product task here, then jump directly into the next operational route instead of backing out through unrelated screens.",
				internalWorkflowCard("Flow 1", "Back to product list", "Return to the merchandising list when you need to move from one product record to the next update.", "/app/products"),
				internalWorkflowCard("Flow 2", "Manage warehouse item", "Open the warehouse-native item profile when this product needs stock edits or replenishment controls.", warehouseItemProfileHref(item)),
				internalWorkflowCard("Flow 3", "Review warehouse lanes", "Open the SKU lane view when the product change needs cross-warehouse inventory context.", "/app/inventory/"+item.SKU),
				internalWorkflowCard("Flow 4", "Review buyer feedback", "Use the comment queue to validate whether the copy update resolved the original customer confusion.", "/app/comments"),
			),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			productUpdateForm(item, payload),
			productDeleteForm(item, payload),
		),
	)
}

func productEditorPreviewHero(item productAdminCard) ui.Node {
	return html.Div(html.Props{Class: internalHeroSurfaceClass()},
		html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
			productCMSStatusBadge(item.Status),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(item.SKU+" | "+item.Category)),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-slate-400"}, html.Text(fallback(item.WarehouseName, item.WarehouseID))),
		),
		html.H2(html.Props{Class: "mt-4 text-3xl font-black tracking-[-0.03em] text-white"}, html.Text(item.Title)),
		html.P(html.Props{Class: "mt-4 max-w-3xl text-base leading-7 text-slate-300"}, html.Text(item.Details)),
		html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-3"},
			statCard("Price", formatPrice(item.PriceCents)),
			statCard("Available", fmt.Sprintf("%d units", item.Available)),
			statCard("Inbound", fmt.Sprintf("%d units", item.Inbound)),
		),
	)
}

func productEditorPreviewSurface(item productAdminCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Preview surface")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("Keep the buyer-facing product story visible while editing so merch changes are reviewed in the same pass as status, warehouse anchor, and SEO updates.")),
		),
		html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-5"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(item.Title)),
				html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Summary)),
			),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"},
				infoRow("Finish", item.Finish),
				infoRow("SEO title", item.SEOTitle),
			),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.18em] text-slate-400"}, html.Text(fallback(item.SEODescription, "SEO description not set"))),
		),
	)
}

func productEditorWarehouseContextCard(item productAdminCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Warehouse context")),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("This editor should keep warehouse impact visible so a copy or status change can pivot immediately into lane review when the issue becomes operational.")),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			statCard("Warehouse", fallback(item.WarehouseName, item.WarehouseID)),
			statCard("Volume", fmt.Sprintf("%d units", productVolume(item))),
			statCard("Hubs", fmt.Sprintf("%d", maxProductHubCount(item.HubCount))),
		),
		html.A(html.Props{Href: warehouseItemProfileHref(item), Class: "inline-flex w-fit items-center rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-300/16"}, html.Text("Open warehouse item profile")),
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

func catalogCMSFilterForm(filters productCMSFilters) ui.Node {
	return html.Form(html.Props{Action: "/app/products", Method: "get", Class: "grid gap-4 " + internalSurfaceCardClass() + " p-5 lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.7fr))_auto] lg:items-end"},
		cmsTextInput("q", "Search products", filters.Search),
		cmsSelectInput("category", "Category", filters.Category, []optionItem{{"all", "All categories"}, {"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}),
		cmsSelectInput("status", "Status", filters.Status, []optionItem{{"all", "All statuses"}, {"in_stock", "In stock"}, {"low_stock", "Low stock"}, {"draft", "Draft"}, {"archived", "Archived"}}),
		cmsSelectInput("sort", "Sort", filters.Sort, []optionItem{{"updated", "Recently updated"}, {"volume", "Highest volume"}, {"price", "Highest price"}, {"status", "Status"}}),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text("Filter")),
	)
}

func productCreateForm(payload Payload) ui.Node {
	return productCreateFormWithOptions(payload, productFormOptions{LockedWarehouseID: fallback(payload.Preferences.DefaultWarehouse, "new-jersey-hub")})
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

func productCreateFormWithOptions(payload Payload, options productFormOptions) ui.Node {
	warehouseID := fallback(options.LockedWarehouseID, "new-jersey-hub")
	introLabel := fallback(options.IntroLabel, "Add catalog item")
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(productEditorFormState{
			Category:        "desks",
			PriceCents:      "189900",
			Status:          "in_stock",
			Finish:          "Graphite oak",
			WarehouseID:     warehouseID,
			Available:       "8",
			Inbound:         "3",
			Summary:         "Short merchandising summary for the storefront card.",
			Details:         "Longer product detail copy for buyers and internal review.",
			SEOTitle:        "Atlas New Product",
			SEODescription:  "Search-friendly description for the Atlas storefront route.",
			ReturnWarehouse: options.ReturnWarehouseID,
		})
		value := form.Get()
		children := []ui.Node{
			cmsBoundTextInput("sku", "SKU", value.SKU, "SKU", form),
			cmsBoundTextInput("slug", "Slug", value.Slug, "Slug", form),
			cmsBoundTextInput("title", "Title", value.Title, "Title", form),
			cmsBoundSelectInput("category", "Category", value.Category, "Category", productCategoryOptions(), form),
			cmsBoundNumberInput("price_cents", "Price cents", value.PriceCents, "PriceCents", form),
			cmsBoundSelectInput("status", "Status", value.Status, "Status", productStatusOptions(), form),
			cmsBoundTextInput("finish", "Finish", value.Finish, "Finish", form),
			warehouseScopedBoundField("warehouse_id", value.WarehouseID, options.LockedWarehouseID, "WarehouseID", warehouseOptions(), form),
			cmsBoundNumberInput("available", "Available volume", value.Available, "Available", form),
			cmsBoundNumberInput("inbound", "Inbound volume", value.Inbound, "Inbound", form),
			cmsBoundTextarea("summary", "Summary", value.Summary, "Summary", form),
			cmsBoundTextarea("details", "Details", value.Details, "Details", form),
			cmsBoundTextInput("seo_title", "SEO title", value.SEOTitle, "SEOTitle", form),
			cmsBoundTextarea("seo_description", "SEO description", value.SEODescription, "SEODescription", form),
			html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(options.SubmitLabel, "Create product"))),
		}
		if strings.TrimSpace(value.ReturnWarehouse) != "" {
			children = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: value.ReturnWarehouse})}, children...)
		}
		return html.Form(html.Props{Action: "/api/app/products", Method: "post", Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
			append([]ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(introLabel))}, prependCSRFToken(payload.CSRF, children...)...)...,
		)
	})
}

func productUpdateForm(item productAdminCard, payload Payload) ui.Node {
	return productUpdateFormWithOptions(item, payload, productFormOptions{LockedWarehouseID: item.WarehouseID})
}

func productUpdateFormWithOptions(item productAdminCard, payload Payload, options productFormOptions) ui.Node {
	warehouseID := fallback(options.LockedWarehouseID, item.WarehouseID)
	introLabel := fallback(options.IntroLabel, "Edit catalog item")
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(productEditorFormState{
			SKU:             item.SKU,
			Slug:            item.Slug,
			Title:           item.Title,
			Category:        item.Category,
			PriceCents:      fmt.Sprintf("%d", item.PriceCents),
			Status:          item.Status,
			Finish:          item.Finish,
			WarehouseID:     warehouseID,
			Available:       fmt.Sprintf("%d", item.Available),
			Inbound:         fmt.Sprintf("%d", item.Inbound),
			Summary:         item.Summary,
			Details:         item.Details,
			SEOTitle:        item.SEOTitle,
			SEODescription:  item.SEODescription,
			ReturnWarehouse: options.ReturnWarehouseID,
		})
		value := form.Get()
		previous := ui.UsePrevious(value)
		hiddenFields := []ui.Node{
			html.Input(html.Props{Type: "hidden", Name: "current_warehouse_id", Value: warehouseID}),
			html.Input(html.Props{Type: "hidden", Name: "sku", Value: value.SKU}),
		}
		metadataSection := productEditorFormSection("Metadata form", "Keep product identity, category, commercial status, and finish settings grouped together.", []ui.Node{
			cmsReadOnlyField("SKU", value.SKU),
			cmsBoundTextInput("slug", "Slug", value.Slug, "Slug", form),
			cmsBoundTextInput("title", "Title", value.Title, "Title", form),
			cmsBoundSelectInput("category", "Category", value.Category, "Category", productCategoryOptions(), form),
			cmsBoundNumberInput("price_cents", "Price cents", value.PriceCents, "PriceCents", form),
			cmsBoundSelectInput("status", "Status", value.Status, "Status", productStatusOptions(), form),
			cmsBoundTextInput("finish", "Finish", value.Finish, "Finish", form),
		}...)
		warehouseSection := productEditorFormSection("Warehouse context", "Warehouse anchor, available units, and inbound posture stay together so merchandising edits can still respect the operating lane.", []ui.Node{
			warehouseScopedBoundField("warehouse_id", value.WarehouseID, options.LockedWarehouseID, "WarehouseID", warehouseOptions(), form),
			cmsBoundNumberInput("available", "Available volume", value.Available, "Available", form),
			cmsBoundNumberInput("inbound", "Inbound volume", value.Inbound, "Inbound", form),
		}...)
		merchSection := productEditorFormSection("Merchandising copy", "Summary, long-form detail, and SEO fields stay in one content block so the public-facing product story is edited coherently.", []ui.Node{
			cmsBoundTextarea("summary", "Summary", value.Summary, "Summary", form),
			cmsBoundTextarea("details", "Details", value.Details, "Details", form),
			cmsBoundTextInput("seo_title", "SEO title", value.SEOTitle, "SEOTitle", form),
			cmsBoundTextarea("seo_description", "SEO description", value.SEODescription, "SEODescription", form),
		}...)
		children := []ui.Node{
			metadataSection,
			warehouseSection,
			merchSection,
			html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(options.SubmitLabel, "Save product"))),
		}
		if changeSummary := productDraftChangeSummary(value, previous); changeSummary != nil {
			children = append([]ui.Node{changeSummary}, children...)
		}
		if strings.TrimSpace(value.ReturnWarehouse) != "" {
			hiddenFields = append(hiddenFields, html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: value.ReturnWarehouse}))
		}
		return html.Form(html.Props{
			Action: "/api/app/products/" + item.Slug + "/update",
			Method: "post",
			Class:  "grid gap-4 " + internalSurfaceCardClass() + " p-5",
			Data: map[string]string{
				"atlas-dirty-guard":   "product-editor",
				"atlas-dirty-message": "Leave the product editor and discard unsaved Atlas product changes?",
			},
		},
			append([]ui.Node{
				html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(introLabel)),
			}, prependCSRFToken(payload.CSRF, append(hiddenFields, children...)...)...)...,
		)
	})
}

func productEditorFormSection(title, copy string, children ...ui.Node) ui.Node {
	sectionChildren := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(title)),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(copy)),
		),
	}
	sectionChildren = append(sectionChildren, children...)
	return html.Div(html.Props{Class: "grid gap-4 " + internalInsetSurfaceClass() + " p-4"}, sectionChildren...)
}

func productDraftChangeSummary(current productEditorFormState, previous ui.Previous[productEditorFormState]) ui.Node {
	if !previous.Ok() {
		return nil
	}
	prior := previous.Get()
	summary := ""
	switch {
	case strings.TrimSpace(prior.Title) != strings.TrimSpace(current.Title):
		summary = "Title changed from " + fallback(prior.Title, "untitled item") + " to " + fallback(current.Title, "untitled item")
	case strings.TrimSpace(prior.Status) != strings.TrimSpace(current.Status):
		summary = "Status changed from " + fallback(prior.Status, "unset") + " to " + fallback(current.Status, "unset")
	case strings.TrimSpace(prior.PriceCents) != strings.TrimSpace(current.PriceCents):
		summary = "Price cents changed from " + fallback(prior.PriceCents, "0") + " to " + fallback(current.PriceCents, "0")
	case strings.TrimSpace(prior.Summary) != strings.TrimSpace(current.Summary):
		summary = "Summary copy was updated in the current draft."
	}
	if summary == "" {
		return nil
	}
	return html.Div(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/20 bg-cyan-400/8 px-4 py-3 text-sm text-cyan-100"},
		html.P(html.Props{Class: "font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Recent draft change")),
		html.P(html.Props{Class: "mt-2 leading-6"}, html.Text(summary)),
	)
}

func productDeleteForm(item productAdminCard, payload Payload) ui.Node {
	return productDeleteFormWithOptions(item, payload, productFormOptions{})
}

func productDeleteFormWithOptions(item productAdminCard, payload Payload, options productFormOptions) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(fallback(options.DeleteCopy, "Deleting a product removes its catalog row and cascades dependent records in the demo database. Use this only when you want the route gone."))),
		html.Button(html.Props{Type: "submit", Class: "rounded-full border border-rose-300/30 bg-rose-400/10 px-5 py-3 text-sm font-semibold text-rose-100"}, html.Text(fallback(options.DeleteLabel, "Delete product"))),
	}
	if strings.TrimSpace(options.ReturnWarehouseID) != "" {
		children = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: options.ReturnWarehouseID})}, children...)
	}
	return html.Form(html.Props{Action: "/api/app/products/" + item.Slug + "/delete", Method: "post", Class: "grid gap-4 rounded-[1.5rem] border border-rose-300/20 bg-rose-400/5 p-5"},
		append([]ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-rose-200"}, html.Text("Retire catalog item"))}, prependCSRFToken(payload.CSRF, children...)...)...,
	)
}

func warehouseScopedField(name string, value string, lockedValue string, options []optionItem) ui.Node {
	if strings.TrimSpace(lockedValue) != "" {
		return html.Input(html.Props{Type: "hidden", Name: name, Value: value})
	}
	return cmsSelectInput(name, "Warehouse", value, options)
}

func warehouseScopedBoundField[T any](name string, value string, lockedValue string, field string, options []optionItem, form ui.Form[T]) ui.Node {
	if strings.TrimSpace(lockedValue) != "" {
		return html.Input(html.Props{Type: "hidden", Name: name, Value: value})
	}
	return cmsBoundSelectInput(name, "Warehouse", value, field, options, form)
}

func productListMetric(label string, value string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-1 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm lg:bg-transparent lg:px-0 lg:py-0"},
		html.P(html.Props{Class: "text-[0.65rem] font-semibold uppercase tracking-[0.22em] text-slate-500"}, html.Text(label)),
		html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(value)),
	)
}

func productVolume(item productAdminCard) int {
	if item.Volume > 0 {
		return item.Volume
	}
	return item.Available + item.Inbound
}

func warehouseItemProfileHref(item productAdminCard) string {
	if strings.TrimSpace(item.WarehouseID) != "" {
		return "/app/warehouses/" + item.WarehouseID + "/items/" + item.SKU
	}
	return "/app/inventory/" + item.SKU
}

func totalProductVolume(items []productAdminCard) int {
	total := 0
	for _, item := range items {
		total += productVolume(item)
	}
	return total
}

type optionItem struct {
	Value string
	Label string
}

func cmsTextInput(name, label, value string) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func cmsBoundTextInput[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func cmsTransitionBoundTextInput[T any](name, label, value, field string, form ui.Form[T], transition atlasTransition) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{
			ID:    id,
			Name:  name,
			Value: value,
			OnInput: ui.UseEvent(func(event ui.InputEvent) {
				atlasSetFormFieldInTransition(form, field, event.GetValue())
			}),
			Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			Raw: map[string]interface{}{
				"aria-labelledby": id + "-label",
				"data-transition": transition.Pending(),
			},
		}),
	)
}

func cmsReadOnlyField(label, value string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}, html.Text(value)),
	)
}

func cmsNumberInput(name, label, value string) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Type: "number", Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func cmsBoundNumberInput[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Input(html.Props{ID: id, Type: "number", Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}),
	)
}

func cmsTextarea(name, label, value string) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Textarea(html.Props{ID: id, Name: name, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, html.Text(value)),
	)
}

func cmsBoundTextarea[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Textarea(html.Props{ID: id, Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, html.Text(value)),
	)
}

func cmsSelectInput(name, label, value string, options []optionItem) ui.Node {
	id := ui.UseId()
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "all")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Select(html.Props{ID: id, Name: name, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, children...),
	)
}

func cmsBoundSelectInput[T any](name, label, value, field string, options []optionItem, form ui.Form[T]) ui.Node {
	id := ui.UseId()
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "all")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Select(html.Props{ID: id, Name: name, OnChange: ui.UseEvent(func(event ui.ChangeEvent) { form.SetField(field, event.GetValue()) }), Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100", Raw: map[string]interface{}{"aria-labelledby": id + "-label"}}, children...),
	)
}

func cmsTransitionBoundSelectInput[T any](name, label, value, field string, options []optionItem, form ui.Form[T], transition atlasTransition) ui.Node {
	id := ui.UseId()
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "all")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{ID: id + "-label"}, html.Text(label)),
		html.Select(html.Props{
			ID:   id,
			Name: name,
			OnChange: ui.UseEvent(func(event ui.ChangeEvent) {
				atlasSetFormFieldInTransition(form, field, event.GetValue())
			}),
			Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100",
			Raw: map[string]interface{}{
				"aria-labelledby": id + "-label",
				"data-transition": transition.Pending(),
			},
		}, children...),
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

func countLowStockProducts(items []productAdminCard) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "low_stock") || item.Available <= 3 {
			count++
		}
	}
	return count
}

func storeCatalogControls(form ui.Form[atlasListFilterState], syncing bool, submit ui.Handler) ui.Node {
	value := form.Get()
	return html.Form(html.Props{Action: "/shop", Method: "get", OnSubmit: submit, Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-5 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.75fr))_auto] lg:items-end"},
		publicCatalogBoundInput("q", "Search", value.Query, "Query", form),
		publicCatalogBoundSelect("category", "Category", value.Category, "Category", []optionItem{{"all", "All categories"}, {"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}, form),
		publicCatalogBoundSelect("warehouse", "Warehouse", value.Warehouse, "Warehouse", []optionItem{{"all", "All warehouses"}, {"new-jersey-hub", "New Jersey Hub"}, {"illinois-hub", "Illinois Hub"}, {"nevada-hub", "Nevada Hub"}}, form),
		publicCatalogBoundSelect("sort", "Sort", value.Sort, "Sort", []optionItem{{"", "Featured"}, {"warehouse", "SKU / warehouse"}}, form),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(atlasFilterSubmitLabel(syncing, "Apply"))),
	)
}

func publicCatalogInput(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
	)
}

func publicCatalogBoundInput[T any](name, label, value, field string, form ui.Form[T]) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, OnInput: ui.UseEvent(func(event ui.InputEvent) { form.SetField(field, event.GetValue()) }), Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
	)
}

func publicCatalogSelect(name, label, value string, options []optionItem) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Select(html.Props{Name: name, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}, children...),
	)
}

func publicCatalogBoundSelect[T any](name, label, value, field string, options []optionItem, form ui.Form[T]) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Select(html.Props{Name: name, OnChange: ui.UseEvent(func(event ui.ChangeEvent) { form.SetField(field, event.GetValue()) }), Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}, children...),
	)
}

func filterCatalogItems(items []productCard, filters atlasListFilterState) []productCard {
	needle := atlasNormalizedFilterValue(filters.Query)
	category := atlasNormalizedFilterValue(filters.Category)
	warehouse := atlasNormalizedFilterValue(filters.Warehouse)
	filtered := make([]productCard, 0, len(items))
	for _, item := range items {
		if needle != "" {
			haystack := atlasNormalizedFilterValue(item.Title + " " + item.SKU + " " + item.Category + " " + item.Summary + " " + item.WarehouseName)
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		if category != "" && category != "all" && !strings.EqualFold(strings.TrimSpace(item.Category), category) {
			continue
		}
		if warehouse != "" && warehouse != "all" && !strings.EqualFold(strings.TrimSpace(item.WarehouseID), warehouse) {
			continue
		}
		filtered = append(filtered, item)
	}
	if atlasNormalizedFilterValue(filters.Sort) == "warehouse" {
		sort.SliceStable(filtered, func(left, right int) bool {
			if filtered[left].WarehouseID == filtered[right].WarehouseID {
				if filtered[left].SKU == filtered[right].SKU {
					return filtered[left].Title < filtered[right].Title
				}
				return filtered[left].SKU < filtered[right].SKU
			}
			return filtered[left].WarehouseID < filtered[right].WarehouseID
		})
	}
	return filtered
}
