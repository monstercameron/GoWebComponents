package atlas

import (
	"fmt"
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
	productCards := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		productCards = append(productCards,
			html.A(html.Props{Href: "/app/products/" + item.Slug, Class: "grid gap-4 rounded-[1.45rem] border border-white/10 bg-white/5 p-5 transition hover:border-cyan-300/60 hover:bg-white/10"},
				html.Div(html.Props{Class: "grid gap-4 lg:grid-cols-[minmax(0,1.35fr)_repeat(4,minmax(0,0.6fr))] lg:items-center"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
							html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(item.Title)),
							html.Span(html.Props{Class: "rounded-full border border-cyan-300/25 bg-cyan-400/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-cyan-200"}, html.Text(strings.ReplaceAll(item.Status, "_", " "))),
						),
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-slate-400"}, html.Text(item.SKU+" · "+item.Category)),
						html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(item.Summary)),
					),
					productListMetric("Available", fmt.Sprintf("%d", item.Available)),
					productListMetric("Inbound", fmt.Sprintf("%d", item.Inbound)),
					productListMetric("Volume", fmt.Sprintf("%d", productVolume(item))),
					productListMetric("Price", formatPrice(item.PriceCents)),
				),
			),
		)
	}
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(24rem,0.9fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			catalogCMSFilterForm(page.Filters),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				statCard("Products", fmt.Sprintf("%d live", page.Total)),
				statCard("Total volume", fmt.Sprintf("%d units", totalProductVolume(page.Items))),
				statCard("Low stock", fmt.Sprintf("%d flagged", countLowStockProducts(page.Items))),
			),
			internalWorkflowSection("Product merchandising flows", "Use the product workspace for item creation, marketing copy, SEO, and catalog readiness, then branch into warehouse routes only when the work becomes lane-specific.",
				internalWorkflowCard("Flow 1", "Add new item", "Start a new Atlas SKU with baseline pricing, merchandising copy, and the first warehouse assignment already in the create form.", "/app/products#create-product"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the freshest product records first when merchandising language, summaries, or SEO need cleanup.", "/app/products?sort=updated"),
				internalWorkflowCard("Flow 3", "Review warehouse operations", "Jump into warehouse-native item management when a product change becomes a lane or replenishment task.", "/app/warehouses"),
				internalWorkflowCard("Flow 4", "Check buyer feedback", "Use real customer questions to decide whether copy, availability, or pricing context needs work.", "/app/comments"),
			),
			listCard("Product merchandising list", productCards...),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{ID: "create-product"}, productCreateForm(payload)),
			featureCard("Why this matters", "Atlas keeps the product workspace focused on merchandising, pricing, and catalog clarity, while warehouse-specific adjustments stay inside warehouse workflows."),
		),
	)
}

func productEditorContent(payload Payload) ui.Node {
	item := decode[productAdminCard](pageData(payload))
	return html.Section(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(24rem,0.88fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			featureCard(item.Title, item.Details),
			productEditorQuickTasks(),
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"},
				statCard("SKU", item.SKU),
				statCard("Price", formatPrice(item.PriceCents)),
				statCard("Status", strings.ReplaceAll(item.Status, "_", " ")),
				statCard("Volume", fmt.Sprintf("%d units", productVolume(item))),
				statCard("Available", fmt.Sprintf("%d units", item.Available)),
				statCard("Inbound", fmt.Sprintf("%d units", item.Inbound)),
			),
			listCard("Catalog copy",
				infoRow("Finish", item.Finish),
				infoRow("SEO title", item.SEOTitle),
				infoRow("SEO description", item.SEODescription),
			),
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
	return html.Form(html.Props{Action: "/app/products", Method: "get", Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5 lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.7fr))_auto] lg:items-end"},
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

func productCreateFormWithOptions(payload Payload, options productFormOptions) ui.Node {
	warehouseID := fallback(options.LockedWarehouseID, "new-jersey-hub")
	children := []ui.Node{
		cmsTextInput("sku", "SKU", ""),
		cmsTextInput("slug", "Slug", ""),
		cmsTextInput("title", "Title", ""),
		cmsSelectInput("category", "Category", "desks", productCategoryOptions()),
		cmsNumberInput("price_cents", "Price cents", "189900"),
		cmsSelectInput("status", "Status", "in_stock", productStatusOptions()),
		cmsTextInput("finish", "Finish", "Graphite oak"),
		warehouseScopedField("warehouse_id", warehouseID, options.LockedWarehouseID, warehouseOptions()),
		cmsNumberInput("available", "Available volume", "8"),
		cmsNumberInput("inbound", "Inbound volume", "3"),
		cmsTextarea("summary", "Summary", "Short merchandising summary for the storefront card."),
		cmsTextarea("details", "Details", "Longer product detail copy for buyers and internal review."),
		cmsTextInput("seo_title", "SEO title", "Atlas New Product"),
		cmsTextarea("seo_description", "SEO description", "Search-friendly description for the Atlas storefront route."),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(options.SubmitLabel, "Create product"))),
	}
	if strings.TrimSpace(options.ReturnWarehouseID) != "" {
		children = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: options.ReturnWarehouseID})}, children...)
	}
	introLabel := fallback(options.IntroLabel, "Add catalog item")
	return html.Form(html.Props{Action: "/api/app/products", Method: "post", Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
		append([]ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(introLabel))}, prependCSRFToken(payload.CSRF, children...)...)...,
	)
}

func productUpdateForm(item productAdminCard, payload Payload) ui.Node {
	return productUpdateFormWithOptions(item, payload, productFormOptions{LockedWarehouseID: item.WarehouseID})
}

func productUpdateFormWithOptions(item productAdminCard, payload Payload, options productFormOptions) ui.Node {
	warehouseID := fallback(options.LockedWarehouseID, item.WarehouseID)
	children := []ui.Node{
		html.Input(html.Props{Type: "hidden", Name: "current_warehouse_id", Value: warehouseID}),
		html.Input(html.Props{Type: "hidden", Name: "sku", Value: item.SKU}),
		cmsReadOnlyField("SKU", item.SKU),
		cmsTextInput("slug", "Slug", item.Slug),
		cmsTextInput("title", "Title", item.Title),
		cmsSelectInput("category", "Category", item.Category, productCategoryOptions()),
		cmsNumberInput("price_cents", "Price cents", fmt.Sprintf("%d", item.PriceCents)),
		cmsSelectInput("status", "Status", item.Status, productStatusOptions()),
		cmsTextInput("finish", "Finish", item.Finish),
		warehouseScopedField("warehouse_id", warehouseID, options.LockedWarehouseID, warehouseOptions()),
		cmsNumberInput("available", "Available volume", fmt.Sprintf("%d", item.Available)),
		cmsNumberInput("inbound", "Inbound volume", fmt.Sprintf("%d", item.Inbound)),
		cmsTextarea("summary", "Summary", item.Summary),
		cmsTextarea("details", "Details", item.Details),
		cmsTextInput("seo_title", "SEO title", item.SEOTitle),
		cmsTextarea("seo_description", "SEO description", item.SEODescription),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950"}, html.Text(fallback(options.SubmitLabel, "Save product"))),
	}
	if strings.TrimSpace(options.ReturnWarehouseID) != "" {
		children = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: options.ReturnWarehouseID})}, children...)
	}
	introLabel := fallback(options.IntroLabel, "Edit catalog item")
	return html.Form(html.Props{
		Action: "/api/app/products/" + item.Slug + "/update",
		Method: "post",
		Class:  "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5",
		Data: map[string]string{
			"atlas-dirty-guard":   "product-editor",
			"atlas-dirty-message": "Leave the product editor and discard unsaved Atlas product changes?",
		},
	},
		append([]ui.Node{html.P(html.Props{Class: "text-sm font-semibold uppercase tracking-[0.25em] text-cyan-300"}, html.Text(introLabel))}, prependCSRFToken(payload.CSRF, children...)...)...,
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
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}),
	)
}

func cmsReadOnlyField(label, value string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}, html.Text(value)),
	)
}

func cmsNumberInput(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Type: "number", Name: name, Value: value, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}),
	)
}

func cmsTextarea(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Textarea(html.Props{Name: name, Class: "min-h-28 rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}, html.Text(value)),
	)
}

func cmsSelectInput(name, label, value string, options []optionItem) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "all")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
		html.Span(html.Props{}, html.Text(label)),
		html.Select(html.Props{Name: name, Class: "rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-slate-100"}, children...),
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

func storeCatalogControls(page catalogPage) ui.Node {
	return html.Form(html.Props{Action: "/shop", Method: "get", Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-5 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.75fr))_auto] lg:items-end"},
		publicCatalogInput("q", "Search", page.Query.Search),
		publicCatalogSelect("category", "Category", page.Query.Category, []optionItem{{"all", "All categories"}, {"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}),
		publicCatalogSelect("warehouse", "Warehouse", page.Query.Warehouse, []optionItem{{"all", "All warehouses"}, {"new-jersey-hub", "New Jersey Hub"}, {"illinois-hub", "Illinois Hub"}, {"nevada-hub", "Nevada Hub"}}),
		publicCatalogSelect("sort", "Sort", page.Query.Sort, []optionItem{{"", "Featured"}, {"warehouse", "SKU / warehouse"}}),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text("Apply")),
	)
}

func publicCatalogInput(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"}),
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
