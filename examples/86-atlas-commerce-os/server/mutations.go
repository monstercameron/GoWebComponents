package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
)

func (s *atlasServer) handlePublicCommentCreate(w http.ResponseWriter, r *http.Request) {
	if !validateCSRFRequest(s, w, r) {
		return
	}
	product, err := s.store.ProductBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "product_not_found", err)
		return
	}
	input, err := decodeCommentRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_comment_request", err)
		return
	}
	if fields := validateCommentRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_comment_request", "Fix the highlighted comment fields and try again.", fields)
		return
	}
	created, err := s.store.CreateComment(r.Context(), serverdb.CreateCommentInput{ProductSKU: product.SKU, AuthorName: input.AuthorName, AuthorType: "public", Reaction: input.Reaction, Subject: input.Subject, Body: input.Body})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "comment_create_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "comment-submitted")
}

func (s *atlasServer) handlePublicQuoteRequestCreate(w http.ResponseWriter, r *http.Request) {
	if !validateCSRFRequest(s, w, r) {
		return
	}
	product, err := s.store.ProductBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "product_not_found", err)
		return
	}
	input, err := decodeQuoteRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_quote_request", err)
		return
	}
	if fields := validateQuoteRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_quote_request", "Fix the highlighted quote request fields and try again.", fields)
		return
	}
	created, err := s.store.CreateQuoteRequest(r.Context(), serverdb.CreateQuoteRequestInput{ProductSKU: product.SKU, RequesterName: input.RequesterName, CompanyName: input.CompanyName, Email: input.Email, Quantity: input.Quantity, Note: input.Note})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "quote_request_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "quote-request-submitted")
}

func (s *atlasServer) handlePublicRestockRequestCreate(w http.ResponseWriter, r *http.Request) {
	if !validateCSRFRequest(s, w, r) {
		return
	}
	product, err := s.store.ProductBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "product_not_found", err)
		return
	}
	input, err := decodeRestockRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_restock_request", err)
		return
	}
	if fields := validateRestockRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_restock_request", "Fix the highlighted restock request fields and try again.", fields)
		return
	}
	created, err := s.store.CreateRestockRequest(r.Context(), serverdb.CreateRestockRequestInput{ProductSKU: product.SKU, Email: input.Email, PreferredWarehouseID: input.PreferredWarehouseID})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "restock_request_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "restock-request-submitted")
}

func (s *atlasServer) handleInternalCommentModeration(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeModerationRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_moderation_request", err)
		return
	}
	if fields := validateModerationRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_moderation_request", "Choose a valid moderation status before applying the review.", fields)
		return
	}
	updated, err := s.store.ModerateComment(r.Context(), r.PathValue("id"), input.Status, input.Reason)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "moderation_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "comment-moderated")
}

func (s *atlasServer) handleInternalThresholdUpdate(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeThresholdRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_threshold_request", err)
		return
	}
	if fields := validateThresholdRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_threshold_request", "Fix the threshold fields before saving the warehouse policy.", fields)
		return
	}
	updated, err := s.store.UpdateThreshold(r.Context(), r.PathValue("sku"), input.WarehouseID, input.ReorderPoint, input.SafetyStock)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "threshold_update_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "threshold-updated")
}

func (s *atlasServer) handleInternalInventoryUpdate(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeInventoryUpdateRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_inventory_update_request", err)
		return
	}
	if fields := validateInventoryUpdateRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_inventory_update_request", "Fix the inventory fields before saving the warehouse lane.", fields)
		return
	}
	updated, err := s.store.UpdateInventoryLevel(r.Context(), r.PathValue("sku"), serverdb.UpdateInventoryLevelInput{WarehouseID: input.WarehouseID, OnHand: input.OnHand, Reserved: input.Reserved, Inbound: input.Inbound, Damaged: input.Damaged, ReorderPoint: input.ReorderPoint, SafetyStock: input.SafetyStock, Status: input.Status})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "inventory_update_failed", err)
		return
	}
	if wantsHTMLResponse(r) {
		if returnPath := sanitizeNextPath(input.ReturnPath); strings.TrimSpace(input.ReturnPath) != "" {
			http.Redirect(w, r, withNotice(returnPath, "inventory-updated"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, withNotice("/app/inventory/"+updated.SKU, "inventory-updated"), http.StatusSeeOther)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "inventory-updated")
}

func (s *atlasServer) handleInternalPreferencesSave(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodePreferencesRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_preferences_request", err)
		return
	}
	if fields := validatePreferencesRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_preferences_request", "Choose valid Atlas preference values before saving.", fields)
		return
	}
	updated, err := s.store.SavePreferences(r.Context(), serverdb.PreferencesRecord{OwnerID: session.UserID, Theme: input.Theme, Locale: input.Locale, Density: input.Density, DefaultWarehouseID: input.DefaultWarehouseID})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "preferences_save_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "preferences-saved")
}

func (s *atlasServer) handleInternalSavedViewCreate(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeSavedViewRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_saved_view_request", err)
		return
	}
	if fields := validateSavedViewRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_saved_view_request", "Fix the saved-view fields before creating the view.", fields)
		return
	}
	created, err := s.store.SaveView(r.Context(), serverdb.SaveViewInput{OwnerID: session.UserID, Name: input.Name, Scope: input.Scope, FiltersJSON: input.FiltersJSON, SortKey: input.SortKey, SortDirection: input.SortDirection, Density: input.Density, WarehouseID: input.WarehouseID})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "saved_view_create_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "saved-view-created")
}

func (s *atlasServer) handleInternalTransferCreate(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeTransferRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_transfer_request", err)
		return
	}
	if fields := validateTransferRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_transfer_request", "Fix the transfer fields before creating the workflow.", fields)
		return
	}
	created, err := s.store.CreateTransfer(r.Context(), serverdb.CreateTransferInput{SourceWarehouseID: input.SourceWarehouseID, DestinationWarehouseID: input.DestinationWarehouseID, Reason: input.Reason, RecommendedBy: input.RecommendedBy})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "transfer_create_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "transfer-created")
}

func (s *atlasServer) handleInternalReceivingReconcile(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodeReceivingRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_receiving_request", err)
		return
	}
	if fields := validateReceivingRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_receiving_request", "Fix the receiving closeout fields before reconciling the session.", fields)
		return
	}
	updated, err := s.store.ReconcileReceiving(r.Context(), r.PathValue("id"), serverdb.ReconcileReceivingInput{Status: input.Status, DiscrepancySummary: input.DiscrepancySummary})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "receiving_reconcile_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "receiving-reconciled")
}

func (s *atlasServer) handleInternalPurchaseOrderStatus(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodePurchaseOrderStatusRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_purchase_order_status_request", err)
		return
	}
	if fields := validatePurchaseOrderStatusRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_purchase_order_status_request", "Choose a valid purchase-order status before updating the vendor workflow.", fields)
		return
	}
	updated, err := s.store.UpdatePurchaseOrderStatus(r.Context(), r.PathValue("id"), serverdb.UpdatePurchaseOrderStatusInput{Status: input.Status})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "purchase_order_status_failed", err)
		return
	}
	s.respondMutation(w, r, http.StatusOK, updated, "purchase-order-updated")
}

func (s *atlasServer) handleInternalPurchaseOrderCreate(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	if !validateCSRFRequest(s, w, r) {
		return
	}
	input, err := decodePurchaseOrderCreateRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_purchase_order_request", err)
		return
	}
	if fields := validatePurchaseOrderCreateRequest(input); len(fields) > 0 {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_purchase_order_request", "Fix the order fields before creating the replenishment workflow.", fields)
		return
	}
	created, err := s.store.CreatePurchaseOrder(r.Context(), serverdb.CreatePurchaseOrderInput{VendorName: input.VendorName, WarehouseID: input.WarehouseID, ProductSKU: input.ProductSKU, Quantity: input.Quantity, ETA: input.ETA, PriorityNote: input.PriorityNote, Status: input.Status})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "purchase_order_create_failed", err)
		return
	}
	if wantsHTMLResponse(r) {
		if returnPath := sanitizeNextPath(input.ReturnPath); strings.TrimSpace(input.ReturnPath) != "" {
			http.Redirect(w, r, withNotice(returnPath, "purchase-order-created"), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, withNotice("/app/purchase-orders/"+created.Order.ID, "purchase-order-created"), http.StatusSeeOther)
		return
	}
	s.respondMutation(w, r, http.StatusCreated, created, "purchase-order-created")
}

type commentRequest struct {
	AuthorName string `json:"author_name"`
	Reaction   string `json:"reaction"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
}

type quoteRequest struct {
	RequesterName string `json:"requester_name"`
	CompanyName   string `json:"company_name"`
	Email         string `json:"email"`
	Quantity      int    `json:"quantity"`
	Note          string `json:"note"`
}

type restockRequest struct {
	Email                string `json:"email"`
	PreferredWarehouseID string `json:"preferred_warehouse_id"`
}

type moderationRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type thresholdRequest struct {
	WarehouseID  string `json:"warehouse_id"`
	ReorderPoint int    `json:"reorder_point"`
	SafetyStock  int    `json:"safety_stock"`
}

type inventoryUpdateRequest struct {
	WarehouseID  string `json:"warehouse_id"`
	OnHand       int    `json:"on_hand"`
	Reserved     int    `json:"reserved"`
	Inbound      int    `json:"inbound"`
	Damaged      int    `json:"damaged"`
	ReorderPoint int    `json:"reorder_point"`
	SafetyStock  int    `json:"safety_stock"`
	Status       string `json:"status"`
	ReturnPath   string `json:"return_path"`
}

type preferencesRequest struct {
	Theme              string `json:"theme"`
	Locale             string `json:"locale"`
	Density            string `json:"density"`
	DefaultWarehouseID string `json:"default_warehouse_id"`
}

type savedViewRequest struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	FiltersJSON   string `json:"filters_json"`
	SortKey       string `json:"sort_key"`
	SortDirection string `json:"sort_direction"`
	Density       string `json:"density"`
	WarehouseID   string `json:"warehouse_id"`
}

type transferRequest struct {
	SourceWarehouseID      string `json:"source_warehouse_id"`
	DestinationWarehouseID string `json:"destination_warehouse_id"`
	Reason                 string `json:"reason"`
	RecommendedBy          string `json:"recommended_by"`
}

type receivingRequest struct {
	Status             string `json:"status"`
	DiscrepancySummary string `json:"discrepancy_summary"`
}

type purchaseOrderStatusRequest struct {
	Status string `json:"status"`
}

type purchaseOrderCreateRequest struct {
	VendorName   string `json:"vendor_name"`
	WarehouseID  string `json:"warehouse_id"`
	ProductSKU   string `json:"product_sku"`
	Quantity     int    `json:"quantity"`
	ETA          string `json:"eta"`
	PriorityNote string `json:"priority_note"`
	Status       string `json:"status"`
	ReturnPath   string `json:"return_path"`
}

func decodeCommentRequest(r *http.Request) (commentRequest, error) {
	var payload commentRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.AuthorName = values.Get("author_name")
		payload.Reaction = values.Get("reaction")
		payload.Subject = values.Get("subject")
		payload.Body = values.Get("body")
	})
}

func decodeQuoteRequest(r *http.Request) (quoteRequest, error) {
	var payload quoteRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.RequesterName = values.Get("requester_name")
		payload.CompanyName = values.Get("company_name")
		payload.Email = values.Get("email")
		payload.Quantity = mustAtoi(values.Get("quantity"), 1)
		payload.Note = values.Get("note")
	})
}

func decodeRestockRequest(r *http.Request) (restockRequest, error) {
	var payload restockRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.Email = values.Get("email")
		payload.PreferredWarehouseID = values.Get("preferred_warehouse_id")
	})
}

func decodeModerationRequest(r *http.Request) (moderationRequest, error) {
	var payload moderationRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.Status = values.Get("status")
		payload.Reason = values.Get("reason")
	})
}

func decodeThresholdRequest(r *http.Request) (thresholdRequest, error) {
	var payload thresholdRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.WarehouseID = values.Get("warehouse_id")
		payload.ReorderPoint = mustAtoi(values.Get("reorder_point"), 1)
		payload.SafetyStock = mustAtoi(values.Get("safety_stock"), 0)
	})
}

func decodeInventoryUpdateRequest(r *http.Request) (inventoryUpdateRequest, error) {
	var payload inventoryUpdateRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.WarehouseID = values.Get("warehouse_id")
		payload.OnHand = mustAtoi(values.Get("on_hand"), 0)
		payload.Reserved = mustAtoi(values.Get("reserved"), 0)
		payload.Inbound = mustAtoi(values.Get("inbound"), 0)
		payload.Damaged = mustAtoi(values.Get("damaged"), 0)
		payload.ReorderPoint = mustAtoi(values.Get("reorder_point"), 1)
		payload.SafetyStock = mustAtoi(values.Get("safety_stock"), 0)
		payload.Status = values.Get("status")
		payload.ReturnPath = values.Get("return_path")
	})
}

func decodePreferencesRequest(r *http.Request) (preferencesRequest, error) {
	var payload preferencesRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.Theme = values.Get("theme")
		payload.Locale = values.Get("locale")
		payload.Density = values.Get("density")
		payload.DefaultWarehouseID = values.Get("default_warehouse_id")
	})
}

func decodeSavedViewRequest(r *http.Request) (savedViewRequest, error) {
	var payload savedViewRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.Name = values.Get("name")
		payload.Scope = values.Get("scope")
		payload.FiltersJSON = values.Get("filters_json")
		payload.SortKey = values.Get("sort_key")
		payload.SortDirection = values.Get("sort_direction")
		payload.Density = values.Get("density")
		payload.WarehouseID = values.Get("warehouse_id")
	})
}

func decodeTransferRequest(r *http.Request) (transferRequest, error) {
	var payload transferRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.SourceWarehouseID = values.Get("source_warehouse_id")
		payload.DestinationWarehouseID = values.Get("destination_warehouse_id")
		payload.Reason = values.Get("reason")
		payload.RecommendedBy = values.Get("recommended_by")
	})
}

func decodeReceivingRequest(r *http.Request) (receivingRequest, error) {
	var payload receivingRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.Status = values.Get("status")
		payload.DiscrepancySummary = values.Get("discrepancy_summary")
	})
}

func decodePurchaseOrderStatusRequest(r *http.Request) (purchaseOrderStatusRequest, error) {
	var payload purchaseOrderStatusRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.Status = values.Get("status")
	})
}

func decodePurchaseOrderCreateRequest(r *http.Request) (purchaseOrderCreateRequest, error) {
	var payload purchaseOrderCreateRequest
	return payload, decodeBodyOrForm(r, &payload, func(values url.Values) {
		payload.VendorName = values.Get("vendor_name")
		payload.WarehouseID = values.Get("warehouse_id")
		payload.ProductSKU = values.Get("product_sku")
		payload.Quantity = mustAtoi(values.Get("quantity"), 1)
		payload.ETA = values.Get("eta")
		payload.PriorityNote = values.Get("priority_note")
		payload.Status = values.Get("status")
		payload.ReturnPath = values.Get("return_path")
	})
}

func decodeBodyOrForm(r *http.Request, target any, assignForm func(url.Values)) error {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(target); err != nil {
			return fmt.Errorf("decode json body: %w", err)
		}
		return nil
	}
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse form body: %w", err)
	}
	assignForm(r.Form)
	return nil
}

func (s *atlasServer) respondMutation(w http.ResponseWriter, r *http.Request, status int, payload any, notice string) {
	if wantsHTMLResponse(r) {
		referer := strings.TrimSpace(r.Header.Get("Referer"))
		if referer == "" {
			referer = "/"
		}
		http.Redirect(w, r, withNotice(referer, notice), http.StatusSeeOther)
		return
	}
	s.writeJSON(w, status, payload)
}

func wantsHTMLResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	return strings.Contains(accept, "text/html") || strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data")
}

func withNotice(raw string, notice string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	query.Set("atlas_notice", notice)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func mustAtoi(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func validateCommentRequest(input commentRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.AuthorName) == "" {
		fields["author_name"] = "Enter the author name."
	}
	if !isOneOf(input.Reaction, "up", "down") {
		fields["reaction"] = "Choose thumbs up or thumbs down."
	}
	if strings.TrimSpace(input.Subject) == "" {
		fields["subject"] = "Enter a short subject for the comment."
	}
	if len(strings.TrimSpace(input.Body)) < 8 {
		fields["body"] = "Enter a more specific comment or question."
	}
	return fields
}

func validateQuoteRequest(input quoteRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.RequesterName) == "" {
		fields["requester_name"] = "Enter the requester name."
	}
	if strings.TrimSpace(input.CompanyName) == "" {
		fields["company_name"] = "Enter the company name."
	}
	if !looksLikeEmail(input.Email) {
		fields["email"] = "Enter a valid contact email address."
	}
	if input.Quantity < 1 {
		fields["quantity"] = "Enter a quantity of at least 1."
	}
	return fields
}

func validateRestockRequest(input restockRequest) map[string]string {
	fields := map[string]string{}
	if !looksLikeEmail(input.Email) {
		fields["email"] = "Enter a valid email address for restock updates."
	}
	if strings.TrimSpace(input.PreferredWarehouseID) == "" {
		fields["preferred_warehouse_id"] = "Choose a preferred warehouse."
	}
	return fields
}

func validateModerationRequest(input moderationRequest) map[string]string {
	fields := map[string]string{}
	if !isOneOf(input.Status, "pending", "approved", "rejected", "flagged") {
		fields["status"] = "Choose pending, approved, rejected, or flagged."
	}
	return fields
}

func validateThresholdRequest(input thresholdRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.WarehouseID) == "" {
		fields["warehouse_id"] = "Choose a warehouse for the threshold policy."
	}
	if input.ReorderPoint < 1 {
		fields["reorder_point"] = "Enter a reorder point greater than zero."
	}
	if input.SafetyStock < 0 {
		fields["safety_stock"] = "Safety stock cannot be negative."
	}
	return fields
}

func validateInventoryUpdateRequest(input inventoryUpdateRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.WarehouseID) == "" {
		fields["warehouse_id"] = "Choose the warehouse lane you are editing."
	}
	if input.OnHand < 0 {
		fields["on_hand"] = "On-hand units cannot be negative."
	}
	if input.Reserved < 0 {
		fields["reserved"] = "Reserved units cannot be negative."
	}
	if input.Inbound < 0 {
		fields["inbound"] = "Inbound units cannot be negative."
	}
	if input.Damaged < 0 {
		fields["damaged"] = "Damaged units cannot be negative."
	}
	if input.ReorderPoint < 1 {
		fields["reorder_point"] = "Reorder point must be at least 1."
	}
	if input.SafetyStock < 0 {
		fields["safety_stock"] = "Safety stock cannot be negative."
	}
	if !isOneOf(input.Status, "balanced", "promise_risk", "critical", "recovery") {
		fields["status"] = "Choose a supported warehouse status."
	}
	return fields
}

func validatePreferencesRequest(input preferencesRequest) map[string]string {
	fields := map[string]string{}
	if !isOneOf(input.Theme, "dark", "light") {
		fields["theme"] = "Choose the dark or light Atlas theme."
	}
	if !isOneOf(input.Locale, "en", "fr", "ar") {
		fields["locale"] = "Choose one of the supported Atlas locales."
	}
	if !isOneOf(input.Density, "compact", "comfortable") {
		fields["density"] = "Choose compact or comfortable density."
	}
	if strings.TrimSpace(input.DefaultWarehouseID) == "" {
		fields["default_warehouse_id"] = "Choose a default warehouse."
	}
	return fields
}

func validateSavedViewRequest(input savedViewRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.Name) == "" {
		fields["name"] = "Enter a name for the saved view."
	}
	if strings.TrimSpace(input.Scope) == "" {
		fields["scope"] = "Choose a scope for the saved view."
	}
	if !isOneOf(input.SortDirection, "asc", "desc") {
		fields["sort_direction"] = "Choose asc or desc sort direction."
	}
	if strings.TrimSpace(input.FiltersJSON) != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(input.FiltersJSON), &parsed); err != nil {
			fields["filters_json"] = "Enter valid JSON for the saved-view filters."
		}
	}
	return fields
}

func validateTransferRequest(input transferRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.SourceWarehouseID) == "" {
		fields["source_warehouse_id"] = "Choose a source warehouse."
	}
	if strings.TrimSpace(input.DestinationWarehouseID) == "" {
		fields["destination_warehouse_id"] = "Choose a destination warehouse."
	}
	if strings.EqualFold(strings.TrimSpace(input.SourceWarehouseID), strings.TrimSpace(input.DestinationWarehouseID)) && strings.TrimSpace(input.SourceWarehouseID) != "" {
		fields["destination_warehouse_id"] = "Choose a destination warehouse different from the source."
	}
	if strings.TrimSpace(input.Reason) == "" {
		fields["reason"] = "Enter the operational reason for this transfer."
	}
	return fields
}

func validateReceivingRequest(input receivingRequest) map[string]string {
	fields := map[string]string{}
	if !isOneOf(input.Status, "open", "in_review", "closed") {
		fields["status"] = "Choose open, in_review, or closed."
	}
	if strings.TrimSpace(input.DiscrepancySummary) == "" {
		fields["discrepancy_summary"] = "Summarize the receiving discrepancy or closeout note."
	}
	return fields
}

func validatePurchaseOrderStatusRequest(input purchaseOrderStatusRequest) map[string]string {
	fields := map[string]string{}
	if !isOneOf(input.Status, "submitted", "approved", "on_hold", "received") {
		fields["status"] = "Choose submitted, approved, on_hold, or received."
	}
	return fields
}

func validatePurchaseOrderCreateRequest(input purchaseOrderCreateRequest) map[string]string {
	fields := map[string]string{}
	if strings.TrimSpace(input.VendorName) == "" {
		fields["vendor_name"] = "Enter the vendor for this replenishment order."
	}
	if strings.TrimSpace(input.WarehouseID) == "" {
		fields["warehouse_id"] = "Choose the receiving warehouse."
	}
	if strings.TrimSpace(input.ProductSKU) == "" {
		fields["product_sku"] = "Choose the inventory item to order."
	}
	if input.Quantity < 1 {
		fields["quantity"] = "Order quantity must be at least 1."
	}
	if strings.TrimSpace(input.ETA) == "" {
		fields["eta"] = "Enter the expected arrival window."
	}
	if strings.TrimSpace(input.PriorityNote) == "" {
		fields["priority_note"] = "Add a short purchasing note for the lane."
	}
	if !isOneOf(input.Status, "draft", "submitted", "approved", "on_hold") {
		fields["status"] = "Choose a valid purchase-order status."
	}
	return fields
}

func looksLikeEmail(value string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(value))
	return err == nil
}

func isOneOf(value string, allowed ...string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	for _, item := range allowed {
		if trimmed == strings.ToLower(item) {
			return true
		}
	}
	return false
}
