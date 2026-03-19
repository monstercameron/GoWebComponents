package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
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

func (s *atlasServer) handleInternalProducts(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	items, err := s.store.ProductAdminList(r.Context(), serverdb.ProductAdminQuery{
		Search:   strings.TrimSpace(r.URL.Query().Get("q")),
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Sort:     strings.TrimSpace(r.URL.Query().Get("sort")),
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "product_admin_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"filters": map[string]string{
			"q":        strings.TrimSpace(r.URL.Query().Get("q")),
			"category": strings.TrimSpace(r.URL.Query().Get("category")),
			"status":   strings.TrimSpace(r.URL.Query().Get("status")),
			"sort":     strings.TrimSpace(r.URL.Query().Get("sort")),
		},
	})
}

func (s *atlasServer) handleInternalProductDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	item, err := s.store.ProductAdminBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "product_admin_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleInternalProductsPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	items, err := s.store.ProductAdminList(r.Context(), serverdb.ProductAdminQuery{
		Search:   strings.TrimSpace(r.URL.Query().Get("q")),
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Sort:     strings.TrimSpace(r.URL.Query().Get("sort")),
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "product_admin_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/products", Surface: "internal", Screen: "products", Title: "Atlas Product Merchandising", Description: "Manage product copy, pricing, launch posture, and merchandising details for Atlas workspace systems.", Canonical: "/app/products"}, map[string]any{
		"items": items,
		"filters": map[string]string{
			"q":        strings.TrimSpace(r.URL.Query().Get("q")),
			"category": strings.TrimSpace(r.URL.Query().Get("category")),
			"status":   strings.TrimSpace(r.URL.Query().Get("status")),
			"sort":     strings.TrimSpace(r.URL.Query().Get("sort")),
		},
	}, session)
}

func (s *atlasServer) handleInternalProductEditorPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	item, err := s.store.ProductAdminBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		path := "/app/products/" + r.PathValue("slug")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Product Editor Not Found", Description: "The requested product editor route could not be loaded.", Canonical: path}, session, "Product editor not found", "That product editor route is not available in the current Atlas catalog.", "/app/products", "Back to product merchandising", err)
		return
	}
	path := "/app/products/" + item.Slug
	s.renderPage(w, r, routeMeta{Path: path, Surface: "internal", Screen: "product-editor", Title: "Atlas Product Editor", Description: "Edit product copy, pricing, and volume for one Atlas storefront item.", Canonical: path}, item, session)
}

func (s *atlasServer) handleInternalProductCreate(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeProductRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_product_request", err)
		return
	}
	if fields := validateProductRequest(input, true); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_product_request", "Fix the product fields before publishing the item into the Atlas catalog.", fields)
		return
	}
	created, err := s.store.CreateProduct(r.Context(), serverdb.CreateProductInput{
		SKU:            input.SKU,
		Slug:           input.Slug,
		Title:          input.Title,
		Category:       input.Category,
		PriceCents:     input.PriceCents,
		Status:         input.Status,
		Finish:         input.Finish,
		Summary:        input.Summary,
		Details:        input.Details,
		SEOTitle:       input.SEOTitle,
		SEODescription: input.SEODescription,
		WarehouseID:    input.WarehouseID,
		Available:      input.Available,
		Inbound:        input.Inbound,
	})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "product_create_failed", err)
		return
	}
	if wantsHTMLResponse(r) {
		if strings.TrimSpace(input.ReturnWarehouseID) != "" {
			http.Redirect(w, r, withNotice("/app/warehouses/"+input.ReturnWarehouseID+"/items/"+created.SKU, "product-created"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, withNotice("/app/products/"+created.Slug, "product-created"), http.StatusSeeOther)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "product-created")
}

func (s *atlasServer) handleInternalProductUpdate(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeProductRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_product_request", err)
		return
	}
	if fields := validateProductRequest(input, false); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_product_request", "Fix the product fields before saving the Atlas catalog item.", fields)
		return
	}
	updated, err := s.store.UpdateProduct(r.Context(), r.PathValue("slug"), serverdb.UpdateProductInput{
		SKU:              input.SKU,
		Slug:             input.Slug,
		Title:            input.Title,
		Category:         input.Category,
		PriceCents:       input.PriceCents,
		Status:           input.Status,
		Finish:           input.Finish,
		Summary:          input.Summary,
		Details:          input.Details,
		SEOTitle:         input.SEOTitle,
		SEODescription:   input.SEODescription,
		CurrentWarehouse: input.CurrentWarehouse,
		WarehouseID:      input.WarehouseID,
		Available:        input.Available,
		Inbound:          input.Inbound,
	})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "product_update_failed", err)
		return
	}
	if wantsHTMLResponse(r) {
		if strings.TrimSpace(input.ReturnWarehouseID) != "" {
			http.Redirect(w, r, withNotice("/app/warehouses/"+input.ReturnWarehouseID+"/items/"+updated.SKU, "product-updated"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, withNotice("/app/products/"+updated.Slug, "product-updated"), http.StatusSeeOther)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "product-updated")
}

func (s *atlasServer) handleInternalProductDelete(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	returnWarehouseID := sanitizeWarehouseReturnID(r.FormValue("return_warehouse_id"))
	if err := s.store.DeleteProduct(r.Context(), r.PathValue("slug")); err != nil {
		s.writeError(w, http.StatusBadRequest, "product_delete_failed", err)
		return
	}
	if wantsHTMLResponse(r) {
		if strings.TrimSpace(returnWarehouseID) != "" {
			http.Redirect(w, r, withNotice("/app/warehouses/"+returnWarehouseID, "product-deleted"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, withNotice("/app/products", "product-deleted"), http.StatusSeeOther)
		return
	}
	s.respondMutation(w, r, http.StatusOK, map[string]any{"ok": true}, "product-deleted")
}

func decodeProductRequest(r *http.Request) (productRequest, error) {
	var payload productRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.SKU = values.Get("sku")
		payload.Slug = values.Get("slug")
		payload.Title = values.Get("title")
		payload.Category = values.Get("category")
		payload.PriceCents = mustProductInt(values.Get("price_cents"), 0)
		payload.Status = values.Get("status")
		payload.Finish = values.Get("finish")
		payload.Summary = values.Get("summary")
		payload.Details = values.Get("details")
		payload.SEOTitle = values.Get("seo_title")
		payload.SEODescription = values.Get("seo_description")
		payload.WarehouseID = values.Get("warehouse_id")
		payload.CurrentWarehouse = values.Get("current_warehouse_id")
		payload.ReturnWarehouseID = values.Get("return_warehouse_id")
		payload.Available = mustProductInt(values.Get("available"), 0)
		payload.Inbound = mustProductInt(values.Get("inbound"), 0)
	})
}

func sanitizeWarehouseReturnID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return ""
	}
	return trimmed
}

func validateProductRequest(input productRequest, requireIdentity bool) map[string]string {
	fields := map[string]string{}
	if requireIdentity && strings.TrimSpace(input.SKU) == "" {
		fields["sku"] = "Enter a stable SKU for the catalog item."
	}
	if requireIdentity && strings.TrimSpace(input.Slug) == "" {
		fields["slug"] = "Enter a storefront slug for the product route."
	}
	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "Enter the product name shown in the storefront."
	}
	if strings.TrimSpace(input.Category) == "" {
		fields["category"] = "Choose a product category."
	}
	if input.PriceCents < 0 {
		fields["price_cents"] = "Price cannot be negative."
	}
	if strings.TrimSpace(input.Status) == "" {
		fields["status"] = "Choose an inventory-facing product status."
	}
	if strings.TrimSpace(input.Summary) == "" {
		fields["summary"] = "Enter a short merchandised summary."
	}
	if strings.TrimSpace(input.WarehouseID) == "" {
		fields["warehouse_id"] = "Choose the managed warehouse lane for this item."
	}
	if input.Available < 0 {
		fields["available"] = "Available units cannot be negative."
	}
	if input.Inbound < 0 {
		fields["inbound"] = "Inbound units cannot be negative."
	}
	return fields
}

func mustProductInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
