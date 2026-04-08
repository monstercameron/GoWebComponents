package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/server/db"
)

type productRequest struct {
	SKU               string `json:"sku"`
	Slug              string `json:"slug"`
	Title             string `json:"title"`
	Category          string `json:"category"`
	PriceCents        int    `json:"price_cents"`
	Status            string `json:"status"`
	Finish            string `json:"finish"`
	Summary           string `json:"summary"`
	Details           string `json:"details"`
	SEOTitle          string `json:"seo_title"`
	SEODescription    string `json:"seo_description"`
	WarehouseID       string `json:"warehouse_id"`
	CurrentWarehouse  string `json:"current_warehouse_id"`
	ReturnWarehouseID string `json:"return_warehouse_id"`
	Available         int    `json:"available"`
	Inbound           int    `json:"inbound"`
}

func (parseS *atlasServer) handleInternalProducts(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItems, parseErr := parseS.store.ProductAdminList(parseR.Context(), serverdb.ProductAdminQuery{
		Search:   strings.TrimSpace(parseR.URL.Query().Get("q")),
		Category: strings.TrimSpace(parseR.URL.Query().Get("category")),
		Status:   strings.TrimSpace(parseR.URL.Query().Get("status")),
		Sort:     strings.TrimSpace(parseR.URL.Query().Get("sort")),
	})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "product_admin_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{
		"items": parseItems,
		"filters": map[string]string{
			"q":        strings.TrimSpace(parseR.URL.Query().Get("q")),
			"category": strings.TrimSpace(parseR.URL.Query().Get("category")),
			"status":   strings.TrimSpace(parseR.URL.Query().Get("status")),
			"sort":     strings.TrimSpace(parseR.URL.Query().Get("sort")),
		},
	})
}

func (parseS *atlasServer) handleInternalProductDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItem, parseErr := parseS.store.ProductAdminBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "product_admin_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleInternalProductsPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItems, parseErr := parseS.store.ProductAdminList(parseR.Context(), serverdb.ProductAdminQuery{
		Search:   strings.TrimSpace(parseR.URL.Query().Get("q")),
		Category: strings.TrimSpace(parseR.URL.Query().Get("category")),
		Status:   strings.TrimSpace(parseR.URL.Query().Get("status")),
		Sort:     strings.TrimSpace(parseR.URL.Query().Get("sort")),
	})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "product_admin_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/products", Surface: "internal", Screen: "products", Title: "Atlas Product Merchandising", Description: "Manage product copy, pricing, launch posture, and merchandising details for Atlas workspace systems.", Canonical: "/app/products"}, map[string]any{
		"items": parseItems,
		"filters": map[string]string{
			"q":        strings.TrimSpace(parseR.URL.Query().Get("q")),
			"category": strings.TrimSpace(parseR.URL.Query().Get("category")),
			"status":   strings.TrimSpace(parseR.URL.Query().Get("status")),
			"sort":     strings.TrimSpace(parseR.URL.Query().Get("sort")),
		},
	}, parseSession)
}

func (parseS *atlasServer) handleInternalProductEditorPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItem, parseErr := parseS.store.ProductAdminBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parsePath := "/app/products/" + parseR.PathValue("slug")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Product Editor Not Found", Description: "The requested product editor route could not be loaded.", Canonical: parsePath}, parseSession, "Product editor not found", "That product editor route is not available in the current Atlas catalog.", "/app/products", "Back to product merchandising", parseErr)
		return
	}
	parsePath2 := "/app/products/" + parseItem.Slug
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath2, Surface: "internal", Screen: "product-editor", Title: "Atlas Product Editor", Description: "Edit product copy, pricing, and volume for one Atlas storefront item.", Canonical: parsePath2}, parseItem, parseSession)
}

func (parseS *atlasServer) handleInternalProductCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeProductRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_product_request", parseErr)
		return
	}
	if parseFields := validateProductRequest(parseInput, true); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_product_request", "Fix the product fields before publishing the item into the Atlas catalog.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.CreateProduct(parseR.Context(), serverdb.CreateProductInput{
		SKU:            parseInput.SKU,
		Slug:           parseInput.Slug,
		Title:          parseInput.Title,
		Category:       parseInput.Category,
		PriceCents:     parseInput.PriceCents,
		Status:         parseInput.Status,
		Finish:         parseInput.Finish,
		Summary:        parseInput.Summary,
		Details:        parseInput.Details,
		SEOTitle:       parseInput.SEOTitle,
		SEODescription: parseInput.SEODescription,
		WarehouseID:    parseInput.WarehouseID,
		Available:      parseInput.Available,
		Inbound:        parseInput.Inbound,
	})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "product_create_failed", parseErr)
		return
	}
	if wantsHTMLResponse(parseR) {
		if strings.TrimSpace(parseInput.ReturnWarehouseID) != "" {
			http.Redirect(parseW, parseR, withNotice("/app/warehouses/"+parseInput.ReturnWarehouseID+"/items/"+parseCreated.SKU, "product-created"), http.StatusSeeOther)
			return
		}
		http.Redirect(parseW, parseR, withNotice("/app/products/"+parseCreated.Slug, "product-created"), http.StatusSeeOther)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "product-created")
}

func (parseS *atlasServer) handleInternalProductUpdate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeProductRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_product_request", parseErr)
		return
	}
	if parseFields := validateProductRequest(parseInput, false); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_product_request", "Fix the product fields before saving the Atlas catalog item.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.UpdateProduct(parseR.Context(), parseR.PathValue("slug"), serverdb.UpdateProductInput{
		SKU:              parseInput.SKU,
		Slug:             parseInput.Slug,
		Title:            parseInput.Title,
		Category:         parseInput.Category,
		PriceCents:       parseInput.PriceCents,
		Status:           parseInput.Status,
		Finish:           parseInput.Finish,
		Summary:          parseInput.Summary,
		Details:          parseInput.Details,
		SEOTitle:         parseInput.SEOTitle,
		SEODescription:   parseInput.SEODescription,
		CurrentWarehouse: parseInput.CurrentWarehouse,
		WarehouseID:      parseInput.WarehouseID,
		Available:        parseInput.Available,
		Inbound:          parseInput.Inbound,
	})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "product_update_failed", parseErr)
		return
	}
	if wantsHTMLResponse(parseR) {
		if strings.TrimSpace(parseInput.ReturnWarehouseID) != "" {
			http.Redirect(parseW, parseR, withNotice("/app/warehouses/"+parseInput.ReturnWarehouseID+"/items/"+parseUpdated.SKU, "product-updated"), http.StatusSeeOther)
			return
		}
		http.Redirect(parseW, parseR, withNotice("/app/products/"+parseUpdated.Slug, "product-updated"), http.StatusSeeOther)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "product-updated")
}

func (parseS *atlasServer) handleInternalProductDelete(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseReturnWarehouseID := sanitizeWarehouseReturnID(parseR.FormValue("return_warehouse_id"))
	if parseErr := parseS.store.DeleteProduct(parseR.Context(), parseR.PathValue("slug")); parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "product_delete_failed", parseErr)
		return
	}
	if wantsHTMLResponse(parseR) {
		if strings.TrimSpace(parseReturnWarehouseID) != "" {
			http.Redirect(parseW, parseR, withNotice("/app/warehouses/"+parseReturnWarehouseID, "product-deleted"), http.StatusSeeOther)
			return
		}
		http.Redirect(parseW, parseR, withNotice("/app/products", "product-deleted"), http.StatusSeeOther)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, map[string]any{"ok": true}, "product-deleted")
}

func decodeProductRequest(parseR *http.Request) (productRequest, error) {
	var parsePayload productRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.SKU = parseValues.Get("sku")
		parsePayload.Slug = parseValues.Get("slug")
		parsePayload.Title = parseValues.Get("title")
		parsePayload.Category = parseValues.Get("category")
		parsePayload.PriceCents = mustProductInt(parseValues.Get("price_cents"), 0)
		parsePayload.Status = parseValues.Get("status")
		parsePayload.Finish = parseValues.Get("finish")
		parsePayload.Summary = parseValues.Get("summary")
		parsePayload.Details = parseValues.Get("details")
		parsePayload.SEOTitle = parseValues.Get("seo_title")
		parsePayload.SEODescription = parseValues.Get("seo_description")
		parsePayload.WarehouseID = parseValues.Get("warehouse_id")
		parsePayload.CurrentWarehouse = parseValues.Get("current_warehouse_id")
		parsePayload.ReturnWarehouseID = parseValues.Get("return_warehouse_id")
		parsePayload.Available = mustProductInt(parseValues.Get("available"), 0)
		parsePayload.Inbound = mustProductInt(parseValues.Get("inbound"), 0)
	})
}

func sanitizeWarehouseReturnID(parseValue string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" || strings.Contains(parseTrimmed, "/") || strings.Contains(parseTrimmed, "\\") {
		return ""
	}
	return parseTrimmed
}

func validateProductRequest(parseInput productRequest, isRequireIdentity bool) map[string]string {
	parseFields := map[string]string{}
	if isRequireIdentity && strings.TrimSpace(parseInput.SKU) == "" {
		parseFields["sku"] = "Enter a stable SKU for the catalog item."
	}
	if isRequireIdentity && strings.TrimSpace(parseInput.Slug) == "" {
		parseFields["slug"] = "Enter a storefront slug for the product route."
	}
	if strings.TrimSpace(parseInput.Title) == "" {
		parseFields["title"] = "Enter the product name shown in the storefront."
	}
	if strings.TrimSpace(parseInput.Category) == "" {
		parseFields["category"] = "Choose a product category."
	}
	if parseInput.PriceCents < 0 {
		parseFields["price_cents"] = "Price cannot be negative."
	}
	if strings.TrimSpace(parseInput.Status) == "" {
		parseFields["status"] = "Choose an inventory-facing product status."
	}
	if strings.TrimSpace(parseInput.Summary) == "" {
		parseFields["summary"] = "Enter a short merchandised summary."
	}
	if strings.TrimSpace(parseInput.WarehouseID) == "" {
		parseFields["warehouse_id"] = "Choose the managed warehouse lane for this item."
	}
	if parseInput.Available < 0 {
		parseFields["available"] = "Available units cannot be negative."
	}
	if parseInput.Inbound < 0 {
		parseFields["inbound"] = "Inbound units cannot be negative."
	}
	return parseFields
}

func mustProductInt(parseValue string, parseFallback int) int {
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
	if parseErr != nil {
		return parseFallback
	}
	return parseParsed
}
