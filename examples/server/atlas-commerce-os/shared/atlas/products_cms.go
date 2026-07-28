package atlas

import (
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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
	return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
			productCMSSummaryBand(parsePage),
			catalogCMSFilterForm(parsePage.Filters),
			productCMSBulkActionRail(),
			productCMSTableShell(parsePage.Items),
		),
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
			html.Div(html.Props{Class: atlasRegionClass(), ID: "create-product"},
				html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Create product")),
				productCreateForm(parsePayload),
			),
			internalWorkflowSection("Product merchandising flows", "Use the product workspace for item creation, marketing copy, SEO, and catalog readiness, then branch into warehouse routes only when the work becomes lane-specific.",
				internalWorkflowCard("Flow 1", "Add new item", "Start a new Atlas SKU with baseline pricing, merchandising copy, and the first warehouse assignment already in the create form.", "/app/products#create-product"),
				internalWorkflowCard("Flow 2", "Update marketing copy", "Open the freshest product records first when merchandising language, summaries, or SEO need cleanup.", "/app/products?sort=updated"),
				internalWorkflowCard("Flow 3", "Review warehouse operations", "Jump into warehouse-native item management when a product change becomes a lane or replenishment task.", "/app/warehouses"),
				internalWorkflowCard("Flow 4", "Check buyer feedback", "Use real customer questions to decide whether copy, availability, or pricing context needs work.", "/app/comments"),
			),
			// The "Why this matters" card is gone. It described the layout the reader is
			// already looking at ("a reviewable catalog table in the main pane, a
			// creation rail on the side"), which is the purest form of copy that exists
			// only because there was a card to fill.
		),
	)
}

// productCMSSummaryBand is the catalog rollup.
//
// The eyebrow and paragraph above it were a design brief written into the product
// ("The products route should read like an operator table first…"). The table below is
// the argument; it does not need the caption.
func productCMSSummaryBand(parsePage productCMSPageData) ui.Node {
	return html.Div(html.Props{Class: atlasRegionClass()},
		atlasFactCluster(
			atlasFact("Products", fmt.Sprintf("%d", parsePage.Total)),
			atlasFact("Total volume", fmt.Sprintf("%d units", totalProductVolume(parsePage.Items))),
			atlasFact("Low stock", fmt.Sprintf("%d", countLowStockProducts(parsePage.Items))),
			atlasFact("Hubs", fmt.Sprintf("%d", productWarehouseCount(parsePage.Items))),
		),
	)
}

// productCMSBulkActionRail is the three hops out of the catalog.
func productCMSBulkActionRail() ui.Node {
	return html.Div(html.Props{Class: atlasRegionClass()},
		atlasSectionHead("Go to", ""),
		productBulkActionCard("Recently edited", "Products whose copy or SEO changed last.", "/app/products?sort=updated"),
		productBulkActionCard("Low stock", "Hand constrained products to inventory before a promise slips.", "/app/inventory?status=promise_risk"),
		productBulkActionCard("Warehouse ops", "Item management for a specific hub.", "/app/warehouses"),
	)
}

// productBulkActionCard is one hop: a hairline row, not an accent-bordered card, and
// without the "Open workflow" line that used to end every one of them identically.
func productBulkActionCard(parseTitle, parseCopy, parseHref string) ui.Node {
	return html.A(html.Props{Href: parseHref, Class: atlasRecordLinkRowClass()},
		html.Span(html.Props{Class: design.Class(design.Display(design.StepFine))}, html.Text(parseTitle)),
		html.Span(html.Props{Class: atlasProseClass()}, html.Text(parseCopy)),
	)
}

// productCMSTableShell is the catalog admin manifest.
//
// This is a queue, not a gallery, so it is design.Table: mono identity cells,
// right-aligned quantities and price, one hairline per row. The heading moved out of
// the card and the card became a single flush surface — two frames removed.
func productCMSTableShell(parseItems []productAdminCard) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, productCMSTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, atlasEmptyRow(8, "No products match these filters. Clear a filter, or create a SKU from the rail."))
	}
	return html.Div(html.Props{Class: atlasStackClass(design.Space3)},
		atlasSectionHead("Catalog", fmt.Sprintf("%d rows", len(parseItems))),
		atlasManifestTable("Product catalog", html.Tr(html.Props{},
			atlasHeaderCell("Product", false),
			atlasHeaderCell("Hub", false),
			atlasHeaderCell("Status", false),
			atlasHeaderCell("Available", true),
			atlasHeaderCell("Inbound", true),
			atlasHeaderCell("Volume", true),
			atlasHeaderCell("Price", true),
			atlasHeaderCell("Actions", false),
		), parseRows),
	)
}

func productCMSTableRow(parseItem productAdminCard) ui.Node {
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: "/app/products/" + parseItem.Slug, Class: atlasCellLinkClass()}, html.Text(parseItem.Title)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.SKU)),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.Category)),
			),
		),
		html.Td(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.Span(html.Props{}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
				html.Span(html.Props{Class: atlasMetaClass()}, html.Text(fmt.Sprintf("%d hubs", maxProductHubCount(parseItem.HubCount)))),
			),
		),
		html.Td(html.Props{}, productCMSStatusBadge(parseItem.Status)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Available)),
		atlasNumericCell(fmt.Sprintf("%d", parseItem.Inbound)),
		atlasNumericCell(fmt.Sprintf("%d", productVolume(parseItem))),
		atlasNumericCell(formatPrice(parseItem.PriceCents)),
		html.Td(html.Props{},
			html.Div(html.Props{Class: atlasStackClass(design.Space1)},
				html.A(html.Props{Href: "/app/products/" + parseItem.Slug, Class: atlasCellLinkClass()}, html.Text("Edit product")),
				html.A(html.Props{Href: warehouseItemProfileHref(parseItem), Class: atlasCellLinkClass()}, html.Text("Open warehouse lane")),
			),
		),
	)
}

// The product summary sentence used to be a third line inside the identity cell of
// every catalog row. It is merchandising copy written for buyers, it is different for
// every product, and it is not what an operator is scanning a catalog table FOR — it
// made each row three lines tall and pushed the numeric columns apart. It lives on the
// product editor, where it is being edited, and on the storefront, where it is read.

// productCMSStatusBadge is the catalog status chip.
//
// The four hand-picked palettes it used to carry (emerald / amber / cyan / slate) are
// replaced by one tone lookup. Note in particular that "draft" was CYAN — an accent hue
// spent on the state that means "nobody has done anything to this yet" — and "low_stock"
// was amber. Both are ToneNeutral now: see inventoryStatusTone for the argument.
func productCMSStatusBadge(parseStatus string) ui.Node {
	return atlasStatusChip(parseStatus)
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
	return html.Section(html.Props{Class: atlasWorkspaceSplitClass()},
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
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
		html.Div(html.Props{Class: atlasStackClass(design.Space5)},
			productUpdateForm(parseItem, parsePayload),
			productDeleteForm(parseItem, parsePayload),
		),
	)
}

// productEditorPreviewHero is the product record's head: PageHead, the SKU as the
// eyebrow, the title, then the long detail copy and the three commercial figures.
func productEditorPreviewHero(parseItem productAdminCard) ui.Node {
	return html.Div(html.Props{Class: atlasStackClass(design.Space4)},
		html.Div(html.Props{Class: design.Class(design.PageHead())},
			html.Span(html.Props{Class: atlasMetaClass()}, html.Text(parseItem.SKU+" · "+parseItem.Category)),
			html.H2(html.Props{Class: design.Class(design.PageTitle())}, html.Text(parseItem.Title)),
		),
		html.Div(html.Props{Class: atlasClusterClass(design.Space3)},
			atlasStatusChip(parseItem.Status),
			html.Span(html.Props{Class: atlasMetaClass()}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
		),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())}, html.Text(parseItem.Details)),
		atlasFactCluster(
			atlasFact("Price", formatPrice(parseItem.PriceCents)),
			atlasFact("Available", fmt.Sprintf("%d units", parseItem.Available)),
			atlasFact("Inbound", fmt.Sprintf("%d units", parseItem.Inbound)),
		),
	)
}

// productEditorPreviewSurface shows the buyer-facing story while you edit it.
//
// It used to be a card inside a card: an outer surface with a heading, wrapping an
// inset surface with the actual preview, wrapping two bordered info rows. Now it is one
// Surface with a Divider between "what the buyer reads" and "the fields behind it" —
// which is the whole point of Divider existing.
func productEditorPreviewSurface(parseItem productAdminCard) ui.Node {
	return html.Div(html.Props{Class: atlasRegionClass()},
		html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Storefront preview")),
		html.H3(html.Props{Class: design.Class(design.Display(design.StepLede))}, html.Text(parseItem.Title)),
		html.P(html.Props{Class: atlasProseClass()}, html.Text(parseItem.Summary)),
		html.Div(html.Props{Class: design.Class(design.Divider())}),
		atlasFactCluster(
			atlasFact("Finish", parseItem.Finish),
			atlasFact("SEO title", parseItem.SEOTitle),
		),
		html.P(html.Props{Class: atlasMetaClass()}, html.Text(fallback(parseItem.SEODescription, "SEO description not set"))),
	)
}

// productEditorWarehouseContextCard keeps the operational consequence of a copy change
// on screen.
func productEditorWarehouseContextCard(parseItem productAdminCard) ui.Node {
	return html.Div(html.Props{Class: atlasRegionClass()},
		atlasSectionHead("Warehouse context", ""),
		atlasFactCluster(
			atlasFact("Hub", fallback(parseItem.WarehouseName, parseItem.WarehouseID)),
			atlasFact("Volume", fmt.Sprintf("%d units", productVolume(parseItem))),
			atlasFact("Hubs", fmt.Sprintf("%d", maxProductHubCount(parseItem.HubCount))),
		),
		html.A(html.Props{Href: warehouseItemProfileHref(parseItem), Class: warehouseSecondaryButtonClass()}, html.Text("Open warehouse item")),
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

// catalogCMSFilterForm narrows the catalog. Recess, like every other filter bar in the
// console: input to the page, marked by a texture change rather than by another box.
func catalogCMSFilterForm(parseFilters productCMSFilters) ui.Node {
	return html.Form(html.Props{Action: "/app/products", Method: "get", Class: design.Class(design.Recess()), Aria: map[string]string{"label": "Filter products"}, Role: "search"},
		html.Div(html.Props{Class: atlasFilterRowClass()},
			cmsTextInput("q", "Search products", parseFilters.Search),
			cmsSelectInput("category", "Category", parseFilters.Category, []optionItem{{"all", "All categories"}, {"desks", "Desks"}, {"storage", "Storage"}, {"seating", "Seating"}, {"accessories", "Accessories"}, {"lighting", "Lighting"}, {"bundles", "Bundles"}}),
			cmsSelectInput("status", "Status", parseFilters.Status, []optionItem{{"all", "All statuses"}, {"in_stock", "In stock"}, {"low_stock", "Low stock"}, {"draft", "Draft"}, {"archived", "Archived"}}),
			cmsSelectInput("sort", "Sort", parseFilters.Sort, []optionItem{{"updated", "Recently updated"}, {"volume", "Highest volume"}, {"price", "Highest price"}, {"status", "Status"}}),
			html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text("Filter")),
		),
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
			html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text(fallback(parseOptions.SubmitLabel, "Create product"))),
		}
		if strings.TrimSpace(parseValue.ReturnWarehouse) != "" {
			parseChildren = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: parseValue.ReturnWarehouse})}, parseChildren...)
		}
		// The create form renders inside a rail region that already has a Surface, so it
		// is a Recess — the same "this is input" texture the filter bars use — instead of
		// a second bordered card inside the first.
		return html.Form(html.Props{Action: "/api/app/products", Method: "post", Class: design.Class(design.Recess(), design.Stack(design.Space4))},
			append([]ui.Node{html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseIntroLabel))}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)...,
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
		parseMetadataSection := productEditorFormSection("Identity", "", []ui.Node{
			cmsReadOnlyField("SKU", parseValue.SKU),
			cmsBoundTextInput("slug", "Slug", parseValue.Slug, "Slug", parseForm),
			cmsBoundTextInput("title", "Title", parseValue.Title, "Title", parseForm),
			cmsBoundSelectInput("category", "Category", parseValue.Category, "Category", productCategoryOptions(), parseForm),
			cmsBoundNumberInput("price_cents", "Price cents", parseValue.PriceCents, "PriceCents", parseForm),
			cmsBoundSelectInput("status", "Status", parseValue.Status, "Status", productStatusOptions(), parseForm),
			cmsBoundTextInput("finish", "Finish", parseValue.Finish, "Finish", parseForm),
		}...)
		parseWarehouseSection := productEditorFormSection("Stock", "", []ui.Node{
			warehouseScopedBoundField("warehouse_id", parseValue.WarehouseID, parseOptions.LockedWarehouseID, "WarehouseID", warehouseOptions(), parseForm),
			cmsBoundNumberInput("available", "Available volume", parseValue.Available, "Available", parseForm),
			cmsBoundNumberInput("inbound", "Inbound volume", parseValue.Inbound, "Inbound", parseForm),
		}...)
		parseMerchSection := productEditorFormSection("Copy and SEO", "", []ui.Node{
			cmsBoundTextarea("summary", "Summary", parseValue.Summary, "Summary", parseForm),
			cmsBoundTextarea("details", "Details", parseValue.Details, "Details", parseForm),
			cmsBoundTextInput("seo_title", "SEO title", parseValue.SEOTitle, "SEOTitle", parseForm),
			cmsBoundTextarea("seo_description", "SEO description", parseValue.SEODescription, "SEODescription", parseForm),
		}...)
		parseChildren := []ui.Node{
			parseMetadataSection,
			parseWarehouseSection,
			parseMerchSection,
			html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text(fallback(parseOptions.SubmitLabel, "Save product"))),
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
			Class:  design.Class(design.Surface(), design.Stack(design.Space5)),
			Data: map[string]string{
				"atlas-dirty-guard":   "product-editor",
				"atlas-dirty-message": "Leave the product editor and discard unsaved Atlas product changes?",
			},
		},
			append([]ui.Node{
				html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseIntroLabel)),
			}, prependCSRFToken(parsePayload.CSRF, append(parseHiddenFields, parseChildren...)...)...)...,
		)
	})
}

// productEditorFormSection groups a handful of fields under one label.
//
// This is where the nesting was worst: an editor Surface, holding three inset Surfaces,
// each holding a heading block and a grid of fields, each field a bordered control —
// four frames deep, uniform weight, before you reach anything typeable. A field group
// is not an object; it is a run of fields with a name. So the name is a FieldLabel-voiced
// line, the separation is a hairline above it, and the box is gone.
//
// parseCopy is kept in the signature and rendered only when it is non-empty. The three
// call sites in this file now pass "": their sentences all explained that related fields
// were grouped together, which is what a heading above a group already says.
func productEditorFormSection(parseTitle, parseCopy string, parseChildren ...ui.Node) ui.Node {
	parseSectionChildren := []ui.Node{
		html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseTitle)),
	}
	if strings.TrimSpace(parseCopy) != "" {
		parseSectionChildren = append(parseSectionChildren, html.P(html.Props{Class: atlasProseClass()}, html.Text(parseCopy)))
	}
	parseSectionChildren = append(parseSectionChildren, html.Div(html.Props{Class: atlasFormGridClass()}, parseChildren...))
	return html.Div(html.Props{Class: atlasFormSectionClass()}, parseSectionChildren...)
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
	// Same treatment as the threshold-history notice: an ink rule and a label, not a
	// tinted panel. An unsaved edit is information, not an alarm.
	return html.Div(html.Props{Class: atlasNoticeClass()},
		html.P(html.Props{Class: design.Class(design.FieldLabel())}, html.Text("Recent draft change")),
		html.P(html.Props{Class: atlasProseClass()}, html.Text(parseSummary)),
	)
}

func productDeleteForm(parseItem productAdminCard, parsePayload Payload) ui.Node {
	return productDeleteFormWithOptions(parseItem, parsePayload, productFormOptions{})
}

// productDeleteFormWithOptions retires a catalog item.
//
// The design system has no ButtonDanger and no red panel, and that is deliberate: a
// permanently red control teaches operators to ignore red, which is expensive in a
// system whose job is flagging real discrepancies. So the trigger is a ButtonSecondary
// and the oxide appears once, on the CONSEQUENCE — the sentence saying what deletion
// destroys — via design.FieldError, which is the design system's "this is the warning
// text" voice.
func productDeleteFormWithOptions(parseItem productAdminCard, parsePayload Payload, parseOptions productFormOptions) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: design.Class(design.FieldError(), design.Measure())}, html.Text(fallback(parseOptions.DeleteCopy, "Deleting a product removes its catalog row and every record that depends on it. There is no undo."))),
		html.Button(html.Props{Type: "submit", Class: warehouseSecondaryButtonClass()}, html.Text(fallback(parseOptions.DeleteLabel, "Delete product"))),
	}
	if strings.TrimSpace(parseOptions.ReturnWarehouseID) != "" {
		parseChildren = append([]ui.Node{html.Input(html.Props{Type: "hidden", Name: "return_warehouse_id", Value: parseOptions.ReturnWarehouseID})}, parseChildren...)
	}
	return html.Form(html.Props{Action: "/api/app/products/" + parseItem.Slug + "/delete", Method: "post", Class: atlasRegionClass()},
		append([]ui.Node{html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Retire catalog item"))}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)...,
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

// productListMetric is a label above a machine fact, same as every other fact in these
// surfaces. The responsive "box on mobile, bare on desktop" trick it used to do is gone
// with the box.
func productListMetric(parseLabel string, parseValue string) ui.Node {
	return atlasFact(parseLabel, parseValue)
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
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsBoundTextInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsTransitionBoundTextInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T], parseTransition atlasTransition) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{
			ID:    parseId,
			Name:  parseName,
			Value: parseValue,
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				atlasSetFormFieldInTransition(parseForm, parseField, parseEvent.GetValue())
			}),
			Class: warehouseInputClass(),
			Raw: map[string]any{
				"aria-labelledby": parseId + "-label",
				"data-transition": parseTransition.Pending(),
			},
		}),
	)
}

// cmsReadOnlyField is a value the operator cannot change — the SKU on an existing
// product. It is a fact, not a disabled input: a bordered box that looks like a field
// and refuses every keystroke is a worse answer than not drawing a field at all.
func cmsReadOnlyField(parseLabel, parseValue string) ui.Node {
	return atlasFact(parseLabel, parseValue)
}

func cmsNumberInput(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Type: "number", Name: parseName, Value: parseValue, Class: warehouseDataInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsBoundNumberInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Type: "number", Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseDataInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func cmsTextarea(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Class: atlasTextareaClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func cmsBoundTextarea[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: atlasTextareaClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func cmsSelectInput(parseName, parseLabel, parseValue string, parseOptions []optionItem) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "all")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Select(html.Props{ID: parseId, Name: parseName, Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, parseChildren...),
	)
}

func cmsBoundSelectInput[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "all")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Select(html.Props{ID: parseId, Name: parseName, OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: warehouseInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, parseChildren...),
	)
}

func cmsTransitionBoundSelectInput[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T], parseTransition atlasTransition) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "all")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: design.Class(design.Field())},
		html.Span(html.Props{ID: parseId + "-label", Class: design.Class(design.FieldLabel())}, html.Text(parseLabel)),
		html.Select(html.Props{
			ID:   parseId,
			Name: parseName,
			OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) {
				atlasSetFormFieldInTransition(parseForm, parseField, parseEvent.GetValue())
			}),
			Class: warehouseInputClass(),
			Raw: map[string]any{
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

// --- storefront controls: NOT converted, deliberately ---------------------------
//
// storeCatalogControls and the four publicCatalog* helpers below render the /shop
// filter bar, which is a PUBLIC surface. They are left on the shared public helpers
// (publicCatalogControlShellClass, publicFormControlClass in visual_primitives.go) on
// purpose: those helpers are the storefront's own vocabulary and are being converted
// with the rest of the public surfaces, and a half-converted filter bar — design-system
// labels around storefront-styled controls — would look like a bug rather than a
// migration. When visual_primitives.go moves, these follow it for free, because the
// control classes come from there rather than from here.
//
// The one exception is the submit button's inline class on the next function, which is
// the last hand-written colour in this file and belongs to the same handoff.
func storeCatalogControls(parseForm ui.Form[atlasListFilterState], isSyncing bool, parseSubmit ui.Handler) ui.Node {
	parseValue := parseForm.Get()
	return html.Form(html.Props{Action: "/shop", Method: "get", OnSubmit: parseSubmit, Class: publicCatalogControlShellClass()},
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
		html.Input(html.Props{Name: parseName, Value: parseValue, Class: publicFormControlClass()}),
	)
}

func publicCatalogBoundInput[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Input(html.Props{Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: publicFormControlClass()}),
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
		html.Select(html.Props{Name: parseName, Class: publicFormControlClass()}, parseChildren...),
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
		html.Select(html.Props{Name: parseName, OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: publicFormControlClass()}, parseChildren...),
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
