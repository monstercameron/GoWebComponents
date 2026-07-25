package main

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	serverdb "github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/repository"
)

func (parseS *atlasServer) handlePublicCommentCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseProduct, parseErr := parseS.store.ProductBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "product_not_found", parseErr)
		return
	}
	parseInput, parseErr := decodeCommentRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_comment_request", parseErr)
		return
	}
	if parseFields := validateCommentRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_comment_request", "Fix the highlighted comment fields and try again.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.CreateComment(parseR.Context(), serverdb.CreateCommentInput{ProductSKU: parseProduct.SKU, AuthorName: parseInput.AuthorName, AuthorType: "public", Reaction: parseInput.Reaction, Subject: parseInput.Subject, Body: parseInput.Body})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "comment_create_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "comment-submitted")
}

func (parseS *atlasServer) handlePublicQuoteRequestCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseProduct, parseErr := parseS.store.ProductBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "product_not_found", parseErr)
		return
	}
	parseInput, parseErr := decodeQuoteRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_quote_request", parseErr)
		return
	}
	if parseFields := validateQuoteRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_quote_request", "Fix the highlighted quote request fields and try again.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.CreateQuoteRequest(parseR.Context(), serverdb.CreateQuoteRequestInput{ProductSKU: parseProduct.SKU, RequesterName: parseInput.RequesterName, CompanyName: parseInput.CompanyName, Email: parseInput.Email, Quantity: parseInput.Quantity, Note: parseInput.Note})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "quote_request_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "quote-request-submitted")
}

func (parseS *atlasServer) handlePublicRestockRequestCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseProduct, parseErr := parseS.store.ProductBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "product_not_found", parseErr)
		return
	}
	parseInput, parseErr := decodeRestockRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_restock_request", parseErr)
		return
	}
	if parseFields := validateRestockRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_restock_request", "Fix the highlighted restock request fields and try again.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.CreateRestockRequest(parseR.Context(), serverdb.CreateRestockRequestInput{ProductSKU: parseProduct.SKU, Email: parseInput.Email, PreferredWarehouseID: parseInput.PreferredWarehouseID})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "restock_request_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "restock-request-submitted")
}

func (parseS *atlasServer) handleInternalCommentModeration(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeModerationRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_moderation_request", parseErr)
		return
	}
	if parseFields := validateModerationRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_moderation_request", "Choose a valid moderation status before applying the review.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.ModerateComment(parseR.Context(), parseR.PathValue("id"), parseInput.Status, parseInput.Reason)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "moderation_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "comment-moderated")
}

func (parseS *atlasServer) handleInternalBulkCommentModeration(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeBulkModerationRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_bulk_moderation_request", parseErr)
		return
	}
	if parseFields := validateBulkModerationRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_bulk_moderation_request", "Choose at least one visible comment and a valid moderation status before applying the bulk review.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.ModerateComments(parseR.Context(), parseInput.IDs, parseInput.Status, parseInput.Reason)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "bulk_moderation_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, map[string]any{"items": parseUpdated}, "comments-bulk-moderated")
}

func (parseS *atlasServer) handleInternalThresholdUpdate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeThresholdRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_threshold_request", parseErr)
		return
	}
	if parseFields := validateThresholdRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_threshold_request", "Fix the threshold fields before saving the warehouse policy.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.UpdateThreshold(parseR.Context(), parseR.PathValue("sku"), parseInput.WarehouseID, parseInput.ReorderPoint, parseInput.SafetyStock)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "threshold_update_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "threshold-updated")
}

func (parseS *atlasServer) handleInternalInventoryUpdate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeInventoryUpdateRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_inventory_update_request", parseErr)
		return
	}
	if parseFields := validateInventoryUpdateRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_inventory_update_request", "Fix the inventory fields before saving the warehouse lane.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.UpdateInventoryLevel(parseR.Context(), parseR.PathValue("sku"), serverdb.UpdateInventoryLevelInput{WarehouseID: parseInput.WarehouseID, OnHand: parseInput.OnHand, Reserved: parseInput.Reserved, Inbound: parseInput.Inbound, Damaged: parseInput.Damaged, ReorderPoint: parseInput.ReorderPoint, SafetyStock: parseInput.SafetyStock, Status: parseInput.Status})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "inventory_update_failed", parseErr)
		return
	}
	if wantsHTMLResponse(parseR) {
		if parseReturnPath := sanitizeNextPath(parseInput.ReturnPath); strings.TrimSpace(parseInput.ReturnPath) != "" {
			http.Redirect(parseW, parseR, withNotice(parseReturnPath, "inventory-updated"), http.StatusSeeOther)
			return
		}
		http.Redirect(parseW, parseR, withNotice("/app/inventory/"+parseUpdated.SKU, "inventory-updated"), http.StatusSeeOther)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "inventory-updated")
}

func (parseS *atlasServer) handleInternalPreferencesSave(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodePreferencesRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_preferences_request", parseErr)
		return
	}
	if parseFields := validatePreferencesRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_preferences_request", "Choose valid Atlas preference values before saving.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.SavePreferences(parseR.Context(), serverdb.PreferencesRecord{OwnerID: parseSession.UserID, Theme: parseInput.Theme, Locale: parseInput.Locale, Density: parseInput.Density, DefaultWarehouseID: parseInput.DefaultWarehouseID})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "preferences_save_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "preferences-saved")
}

func (parseS *atlasServer) handleInternalSavedViewCreate(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeSavedViewRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_saved_view_request", parseErr)
		return
	}
	if parseFields := validateSavedViewRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_saved_view_request", "Fix the saved-view fields before creating the view.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.SaveView(parseR.Context(), serverdb.SaveViewInput{OwnerID: parseSession.UserID, Name: parseInput.Name, Scope: parseInput.Scope, FiltersJSON: parseInput.FiltersJSON, SortKey: parseInput.SortKey, SortDirection: parseInput.SortDirection, Density: parseInput.Density, WarehouseID: parseInput.WarehouseID})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "saved_view_create_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "saved-view-created")
}

func (parseS *atlasServer) handleInternalSavedViewsExport(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItems, parseErr := parseS.store.SavedViewsByOwner(parseR.Context(), parseSession.UserID)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "saved_views_export_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, buildSavedViewTransferDocument(parseItems))
}

func (parseS *atlasServer) handleInternalSavedViewImport(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeSavedViewImportRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_saved_view_import_request", parseErr)
		return
	}
	if parseFields := validateSavedViewImportRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_saved_view_import_request", "Paste a valid saved-view export payload before importing Atlas presets.", parseFields)
		return
	}
	parseViews, parseErr := parseSavedViewTransferDocument(parseInput.ViewsJSON)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_saved_view_import_payload", parseErr)
		return
	}
	parseImported, parseErr := parseS.store.ImportSavedViews(parseR.Context(), parseSession.UserID, parseViews)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "saved_view_import_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, map[string]any{"items": parseImported}, "saved-views-imported")
}

func (parseS *atlasServer) handleInternalTransferCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeTransferRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_transfer_request", parseErr)
		return
	}
	if parseFields := validateTransferRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_transfer_request", "Fix the transfer fields before creating the workflow.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.CreateTransfer(parseR.Context(), serverdb.CreateTransferInput{SourceWarehouseID: parseInput.SourceWarehouseID, DestinationWarehouseID: parseInput.DestinationWarehouseID, Reason: parseInput.Reason, RecommendedBy: parseInput.RecommendedBy})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "transfer_create_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "transfer-created")
}

func (parseS *atlasServer) handleInternalReceivingReconcile(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodeReceivingRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_receiving_request", parseErr)
		return
	}
	if parseFields := validateReceivingRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_receiving_request", "Fix the receiving closeout fields before reconciling the session.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.ReconcileReceiving(parseR.Context(), parseR.PathValue("id"), serverdb.ReconcileReceivingInput{Status: parseInput.Status, DiscrepancySummary: parseInput.DiscrepancySummary})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "receiving_reconcile_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "receiving-reconciled")
}

func (parseS *atlasServer) handleInternalReceivingAttachmentCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	if parseErr := parseR.ParseMultipartForm(12 << 20); parseErr != nil {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_receiving_attachment", "Choose at least one attachment before submitting receiving evidence.", map[string]string{"attachment": "Upload one or more files before submitting the evidence package."})
		return
	}
	parseFiles := []*multipart.FileHeader{}
	if parseR.MultipartForm != nil {
		parseFiles = parseR.MultipartForm.File["attachment"]
	}
	if len(parseFiles) == 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_receiving_attachment", "Choose at least one attachment before submitting receiving evidence.", map[string]string{"attachment": "Upload one or more files before submitting the evidence package."})
		return
	}
	parseSessionID := strings.TrimSpace(parseR.PathValue("id"))
	parseNote := strings.TrimSpace(parseR.FormValue("note"))
	parseItems := make([]map[string]any, 0, len(parseFiles))
	for _, parseFile := range parseFiles {
		parseName := strings.TrimSpace(parseFile.Filename)
		if parseName == "" {
			parseName = "unnamed-evidence"
		}
		parseItems = append(parseItems, map[string]any{
			"name": parseName,
			"size": parseFile.Size,
			"note": parseNote,
		})
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, map[string]any{
		"sessionId": parseSessionID,
		"items":     parseItems,
	}, "receiving-attachment-added")
}

func (parseS *atlasServer) handleInternalPurchaseOrderStatus(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodePurchaseOrderStatusRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_purchase_order_status_request", parseErr)
		return
	}
	if parseFields := validatePurchaseOrderStatusRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_purchase_order_status_request", "Choose a valid purchase-order status before updating the vendor workflow.", parseFields)
		return
	}
	parseUpdated, parseErr := parseS.store.UpdatePurchaseOrderStatus(parseR.Context(), parseR.PathValue("id"), serverdb.UpdatePurchaseOrderStatusInput{Status: parseInput.Status})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "purchase_order_status_failed", parseErr)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusOK, parseUpdated, "purchase-order-updated")
}

func (parseS *atlasServer) handleInternalPurchaseOrderCreate(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	if !validateCSRFRequest(parseS, parseW, parseR) {
		return
	}
	parseInput, parseErr := decodePurchaseOrderCreateRequest(parseR)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "invalid_purchase_order_request", parseErr)
		return
	}
	if parseFields := validatePurchaseOrderCreateRequest(parseInput); len(parseFields) > 0 {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_purchase_order_request", "Fix the order fields before creating the replenishment workflow.", parseFields)
		return
	}
	parseCreated, parseErr := parseS.store.CreatePurchaseOrder(parseR.Context(), serverdb.CreatePurchaseOrderInput{VendorName: parseInput.VendorName, WarehouseID: parseInput.WarehouseID, ProductSKU: parseInput.ProductSKU, Quantity: parseInput.Quantity, ETA: parseInput.ETA, PriorityNote: parseInput.PriorityNote, Status: parseInput.Status})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "purchase_order_create_failed", parseErr)
		return
	}
	if wantsHTMLResponse(parseR) {
		if parseReturnPath := sanitizeNextPath(parseInput.ReturnPath); strings.TrimSpace(parseInput.ReturnPath) != "" {
			http.Redirect(parseW, parseR, withNotice(parseReturnPath, "purchase-order-created"), http.StatusSeeOther)
			return
		}
		http.Redirect(parseW, parseR, withNotice("/app/purchase-orders/"+parseCreated.Order.ID, "purchase-order-created"), http.StatusSeeOther)
		return
	}
	parseS.respondMutation(parseW, parseR, http.StatusCreated, parseCreated, "purchase-order-created")
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

type bulkModerationRequest struct {
	IDs    []string `json:"ids"`
	Status string   `json:"status"`
	Reason string   `json:"reason"`
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

type savedViewImportRequest struct {
	ViewsJSON string `json:"views_json"`
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

func decodeCommentRequest(parseR *http.Request) (commentRequest, error) {
	var parsePayload commentRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.AuthorName = parseValues.Get("author_name")
		parsePayload.Reaction = parseValues.Get("reaction")
		parsePayload.Subject = parseValues.Get("subject")
		parsePayload.Body = parseValues.Get("body")
	})
}

func decodeQuoteRequest(parseR *http.Request) (quoteRequest, error) {
	var parsePayload quoteRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.RequesterName = parseValues.Get("requester_name")
		parsePayload.CompanyName = parseValues.Get("company_name")
		parsePayload.Email = parseValues.Get("email")
		parsePayload.Quantity = mustAtoi(parseValues.Get("quantity"), 1)
		parsePayload.Note = parseValues.Get("note")
	})
}

func decodeRestockRequest(parseR *http.Request) (restockRequest, error) {
	var parsePayload restockRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.Email = parseValues.Get("email")
		parsePayload.PreferredWarehouseID = parseValues.Get("preferred_warehouse_id")
	})
}

func decodeModerationRequest(parseR *http.Request) (moderationRequest, error) {
	var parsePayload moderationRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.Status = parseValues.Get("status")
		parsePayload.Reason = parseValues.Get("reason")
	})
}

func decodeBulkModerationRequest(parseR *http.Request) (bulkModerationRequest, error) {
	var parsePayload bulkModerationRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.IDs = collectListField(parseValues, "ids")
		parsePayload.Status = parseValues.Get("status")
		parsePayload.Reason = parseValues.Get("reason")
	})
}

func decodeThresholdRequest(parseR *http.Request) (thresholdRequest, error) {
	var parsePayload thresholdRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.WarehouseID = parseValues.Get("warehouse_id")
		parsePayload.ReorderPoint = mustAtoi(parseValues.Get("reorder_point"), 1)
		parsePayload.SafetyStock = mustAtoi(parseValues.Get("safety_stock"), 0)
	})
}

func decodeInventoryUpdateRequest(parseR *http.Request) (inventoryUpdateRequest, error) {
	var parsePayload inventoryUpdateRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.WarehouseID = parseValues.Get("warehouse_id")
		parsePayload.OnHand = mustAtoi(parseValues.Get("on_hand"), 0)
		parsePayload.Reserved = mustAtoi(parseValues.Get("reserved"), 0)
		parsePayload.Inbound = mustAtoi(parseValues.Get("inbound"), 0)
		parsePayload.Damaged = mustAtoi(parseValues.Get("damaged"), 0)
		parsePayload.ReorderPoint = mustAtoi(parseValues.Get("reorder_point"), 1)
		parsePayload.SafetyStock = mustAtoi(parseValues.Get("safety_stock"), 0)
		parsePayload.Status = parseValues.Get("status")
		parsePayload.ReturnPath = parseValues.Get("return_path")
	})
}

func decodePreferencesRequest(parseR *http.Request) (preferencesRequest, error) {
	var parsePayload preferencesRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.Theme = parseValues.Get("theme")
		parsePayload.Locale = parseValues.Get("locale")
		parsePayload.Density = parseValues.Get("density")
		parsePayload.DefaultWarehouseID = parseValues.Get("default_warehouse_id")
	})
}

func decodeSavedViewRequest(parseR *http.Request) (savedViewRequest, error) {
	var parsePayload savedViewRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.Name = parseValues.Get("name")
		parsePayload.Scope = parseValues.Get("scope")
		parsePayload.FiltersJSON = parseValues.Get("filters_json")
		parsePayload.SortKey = parseValues.Get("sort_key")
		parsePayload.SortDirection = parseValues.Get("sort_direction")
		parsePayload.Density = parseValues.Get("density")
		parsePayload.WarehouseID = parseValues.Get("warehouse_id")
	})
}

func decodeSavedViewImportRequest(parseR *http.Request) (savedViewImportRequest, error) {
	var parsePayload savedViewImportRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.ViewsJSON = parseValues.Get("views_json")
	})
}

func decodeTransferRequest(parseR *http.Request) (transferRequest, error) {
	var parsePayload transferRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.SourceWarehouseID = parseValues.Get("source_warehouse_id")
		parsePayload.DestinationWarehouseID = parseValues.Get("destination_warehouse_id")
		parsePayload.Reason = parseValues.Get("reason")
		parsePayload.RecommendedBy = parseValues.Get("recommended_by")
	})
}

func decodeReceivingRequest(parseR *http.Request) (receivingRequest, error) {
	var parsePayload receivingRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.Status = parseValues.Get("status")
		parsePayload.DiscrepancySummary = parseValues.Get("discrepancy_summary")
	})
}

func decodePurchaseOrderStatusRequest(parseR *http.Request) (purchaseOrderStatusRequest, error) {
	var parsePayload purchaseOrderStatusRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.Status = parseValues.Get("status")
	})
}

func decodePurchaseOrderCreateRequest(parseR *http.Request) (purchaseOrderCreateRequest, error) {
	var parsePayload purchaseOrderCreateRequest
	return parsePayload, decodeBodyOrForm(parseR, &parsePayload, func(parseValues url.Values) {
		parsePayload.VendorName = parseValues.Get("vendor_name")
		parsePayload.WarehouseID = parseValues.Get("warehouse_id")
		parsePayload.ProductSKU = parseValues.Get("product_sku")
		parsePayload.Quantity = mustAtoi(parseValues.Get("quantity"), 1)
		parsePayload.ETA = parseValues.Get("eta")
		parsePayload.PriorityNote = parseValues.Get("priority_note")
		parsePayload.Status = parseValues.Get("status")
		parsePayload.ReturnPath = parseValues.Get("return_path")
	})
}

func decodeBodyOrForm(parseR *http.Request, parseTarget any, parseAssignForm func(url.Values)) error {
	parseContentType := strings.ToLower(strings.TrimSpace(parseR.Header.Get("Content-Type")))
	if strings.Contains(parseContentType, "application/json") {
		if parseErr := json.NewDecoder(parseR.Body).Decode(parseTarget); parseErr != nil {
			return fmt.Errorf("decode json body: %w", parseErr)
		}
		return nil
	}
	if parseErr2 := parseR.ParseForm(); parseErr2 != nil {
		return fmt.Errorf("parse form body: %w", parseErr2)
	}
	parseAssignForm(parseR.Form)
	return nil
}

func (parseS *atlasServer) respondMutation(parseW http.ResponseWriter, parseR *http.Request, parseStatus int, parsePayload any, parseNotice string) {
	if wantsHTMLResponse(parseR) {
		parseReferer := strings.TrimSpace(parseR.Header.Get("Referer"))
		if parseReferer == "" {
			parseReferer = "/"
		}
		http.Redirect(parseW, parseR, withNotice(parseReferer, parseNotice), http.StatusSeeOther)
		return
	}
	parseS.writeJSON(parseW, parseStatus, parsePayload)
}

func wantsHTMLResponse(parseR *http.Request) bool {
	parseAccept := strings.ToLower(parseR.Header.Get("Accept"))
	parseContentType := strings.ToLower(parseR.Header.Get("Content-Type"))
	return strings.Contains(parseAccept, "text/html") || strings.Contains(parseContentType, "application/x-www-form-urlencoded") || strings.Contains(parseContentType, "multipart/form-data")
}

func withNotice(parseRaw string, parseNotice string) string {
	parseParsed, parseErr := url.Parse(parseRaw)
	if parseErr != nil {
		return parseRaw
	}
	parseQuery := parseParsed.Query()
	parseQuery.Set("atlas_notice", parseNotice)
	parseParsed.RawQuery = parseQuery.Encode()
	return parseParsed.String()
}

func mustAtoi(parseValue string, parseFallback int) int {
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
	if parseErr != nil {
		return parseFallback
	}
	return parseParsed
}

func validateCommentRequest(parseInput commentRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.AuthorName) == "" {
		parseFields["author_name"] = "Enter the author name."
	}
	if !isOneOf(parseInput.Reaction, "up", "down") {
		parseFields["reaction"] = "Choose thumbs up or thumbs down."
	}
	if strings.TrimSpace(parseInput.Subject) == "" {
		parseFields["subject"] = "Enter a short subject for the comment."
	}
	if len(strings.TrimSpace(parseInput.Body)) < 8 {
		parseFields["body"] = "Enter a more specific comment or question."
	}
	return parseFields
}

func validateQuoteRequest(parseInput quoteRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.RequesterName) == "" {
		parseFields["requester_name"] = "Enter the requester name."
	}
	if strings.TrimSpace(parseInput.CompanyName) == "" {
		parseFields["company_name"] = "Enter the company name."
	}
	if !looksLikeEmail(parseInput.Email) {
		parseFields["email"] = "Enter a valid contact email address."
	}
	if parseInput.Quantity < 1 {
		parseFields["quantity"] = "Enter a quantity of at least 1."
	}
	return parseFields
}

func validateRestockRequest(parseInput restockRequest) map[string]string {
	parseFields := map[string]string{}
	if !looksLikeEmail(parseInput.Email) {
		parseFields["email"] = "Enter a valid email address for restock updates."
	}
	if strings.TrimSpace(parseInput.PreferredWarehouseID) == "" {
		parseFields["preferred_warehouse_id"] = "Choose a preferred warehouse."
	}
	return parseFields
}

func validateModerationRequest(parseInput moderationRequest) map[string]string {
	parseFields := map[string]string{}
	if !isOneOf(parseInput.Status, "pending", "approved", "rejected", "flagged") {
		parseFields["status"] = "Choose pending, approved, rejected, or flagged."
	}
	return parseFields
}

func validateBulkModerationRequest(parseInput bulkModerationRequest) map[string]string {
	parseFields := map[string]string{}
	if len(parseInput.IDs) == 0 {
		parseFields["ids"] = "Choose at least one visible comment for bulk review."
	}
	if !isOneOf(parseInput.Status, "pending", "approved", "rejected", "flagged") {
		parseFields["status"] = "Choose pending, approved, rejected, or flagged."
	}
	return parseFields
}

func validateThresholdRequest(parseInput thresholdRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.WarehouseID) == "" {
		parseFields["warehouse_id"] = "Choose a warehouse for the threshold policy."
	}
	if parseInput.ReorderPoint < 1 {
		parseFields["reorder_point"] = "Enter a reorder point greater than zero."
	}
	if parseInput.SafetyStock < 0 {
		parseFields["safety_stock"] = "Safety stock cannot be negative."
	}
	return parseFields
}

func validateInventoryUpdateRequest(parseInput inventoryUpdateRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.WarehouseID) == "" {
		parseFields["warehouse_id"] = "Choose the warehouse lane you are editing."
	}
	if parseInput.OnHand < 0 {
		parseFields["on_hand"] = "On-hand units cannot be negative."
	}
	if parseInput.Reserved < 0 {
		parseFields["reserved"] = "Reserved units cannot be negative."
	}
	if parseInput.Inbound < 0 {
		parseFields["inbound"] = "Inbound units cannot be negative."
	}
	if parseInput.Damaged < 0 {
		parseFields["damaged"] = "Damaged units cannot be negative."
	}
	if parseInput.ReorderPoint < 1 {
		parseFields["reorder_point"] = "Reorder point must be at least 1."
	}
	if parseInput.SafetyStock < 0 {
		parseFields["safety_stock"] = "Safety stock cannot be negative."
	}
	if !isOneOf(parseInput.Status, "balanced", "promise_risk", "critical", "recovery") {
		parseFields["status"] = "Choose a supported warehouse status."
	}
	return parseFields
}

func validatePreferencesRequest(parseInput preferencesRequest) map[string]string {
	parseFields := map[string]string{}
	if !isOneOf(parseInput.Theme, "dark", "light") {
		parseFields["theme"] = "Choose the dark or light Atlas theme."
	}
	if !isOneOf(parseInput.Locale, "en", "fr", "ar") {
		parseFields["locale"] = "Choose one of the supported Atlas locales."
	}
	if !isOneOf(parseInput.Density, "compact", "comfortable") {
		parseFields["density"] = "Choose compact or comfortable density."
	}
	if strings.TrimSpace(parseInput.DefaultWarehouseID) == "" {
		parseFields["default_warehouse_id"] = "Choose a default warehouse."
	}
	return parseFields
}

func validateSavedViewRequest(parseInput savedViewRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.Name) == "" {
		parseFields["name"] = "Enter a name for the saved view."
	}
	if strings.TrimSpace(parseInput.Scope) == "" {
		parseFields["scope"] = "Choose a scope for the saved view."
	}
	if !isOneOf(parseInput.SortDirection, "asc", "desc") {
		parseFields["sort_direction"] = "Choose asc or desc sort direction."
	}
	if strings.TrimSpace(parseInput.FiltersJSON) != "" {
		var parseParsed map[string]any
		if parseErr := json.Unmarshal([]byte(parseInput.FiltersJSON), &parseParsed); parseErr != nil {
			parseFields["filters_json"] = "Enter valid JSON for the saved-view filters."
		}
	}
	return parseFields
}

func validateSavedViewImportRequest(parseInput savedViewImportRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.ViewsJSON) == "" {
		parseFields["views_json"] = "Paste the saved-view export payload before importing."
		return parseFields
	}
	if _, parseErr := parseSavedViewTransferDocument(parseInput.ViewsJSON); parseErr != nil {
		parseFields["views_json"] = "Paste a valid saved-view export payload."
	}
	return parseFields
}

func collectListField(parseValues url.Values, parseKey string) []string {
	parseResult := []string{}
	for _, parseRaw := range parseValues[parseKey] {
		for parsePart := range strings.SplitSeq(parseRaw, ",") {
			parseTrimmed := strings.TrimSpace(parsePart)
			if parseTrimmed != "" {
				parseResult = append(parseResult, parseTrimmed)
			}
		}
	}
	return parseResult
}

type savedViewTransferDocument struct {
	Items []savedViewTransferItem `json:"items"`
}

type savedViewTransferItem struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	SortKey       string `json:"sortKey"`
	SortDirection string `json:"sortDirection"`
	Density       string `json:"density"`
	WarehouseID   string `json:"warehouseId"`
	FiltersJSON   string `json:"filtersJSON"`
}

func buildSavedViewTransferDocument(parseItems []repository.SavedView) savedViewTransferDocument {
	parseResult := savedViewTransferDocument{Items: make([]savedViewTransferItem, 0, len(parseItems))}
	for _, parseItem := range parseItems {
		parseResult.Items = append(parseResult.Items, savedViewTransferItem{
			Name:          parseItem.Name,
			Scope:         parseItem.Scope,
			SortKey:       parseItem.SortKey,
			SortDirection: parseItem.SortDirection,
			Density:       parseItem.Density,
			WarehouseID:   parseItem.WarehouseID,
			FiltersJSON:   parseItem.FiltersJSON,
		})
	}
	return parseResult
}

func parseSavedViewTransferDocument(parseRaw string) ([]repository.SavedView, error) {
	parseTrimmed := strings.TrimSpace(parseRaw)
	if parseTrimmed == "" {
		return nil, fmt.Errorf("saved-view transfer payload is required")
	}
	parseDocument := savedViewTransferDocument{}
	if parseErr := json.Unmarshal([]byte(parseTrimmed), &parseDocument); parseErr == nil && len(parseDocument.Items) > 0 {
		parseItems := make([]repository.SavedView, 0, len(parseDocument.Items))
		for _, parseItem := range parseDocument.Items {
			parseItems = append(parseItems, repository.SavedView{
				Name:          parseItem.Name,
				Scope:         parseItem.Scope,
				SortKey:       parseItem.SortKey,
				SortDirection: parseItem.SortDirection,
				Density:       parseItem.Density,
				WarehouseID:   parseItem.WarehouseID,
				FiltersJSON:   parseItem.FiltersJSON,
			})
		}
		return parseItems, nil
	}
	parseItems2 := []savedViewTransferItem{}
	if parseErr2 := json.Unmarshal([]byte(parseTrimmed), &parseItems2); parseErr2 != nil {
		return nil, fmt.Errorf("decode saved-view transfer payload: %w", parseErr2)
	}
	parseResult := make([]repository.SavedView, 0, len(parseItems2))
	for _, parseItem2 := range parseItems2 {
		parseResult = append(parseResult, repository.SavedView{
			Name:          parseItem2.Name,
			Scope:         parseItem2.Scope,
			SortKey:       parseItem2.SortKey,
			SortDirection: parseItem2.SortDirection,
			Density:       parseItem2.Density,
			WarehouseID:   parseItem2.WarehouseID,
			FiltersJSON:   parseItem2.FiltersJSON,
		})
	}
	return parseResult, nil
}

func validateTransferRequest(parseInput transferRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.SourceWarehouseID) == "" {
		parseFields["source_warehouse_id"] = "Choose a source warehouse."
	}
	if strings.TrimSpace(parseInput.DestinationWarehouseID) == "" {
		parseFields["destination_warehouse_id"] = "Choose a destination warehouse."
	}
	if strings.EqualFold(strings.TrimSpace(parseInput.SourceWarehouseID), strings.TrimSpace(parseInput.DestinationWarehouseID)) && strings.TrimSpace(parseInput.SourceWarehouseID) != "" {
		parseFields["destination_warehouse_id"] = "Choose a destination warehouse different from the source."
	}
	if strings.TrimSpace(parseInput.Reason) == "" {
		parseFields["reason"] = "Enter the operational reason for this transfer."
	}
	return parseFields
}

func validateReceivingRequest(parseInput receivingRequest) map[string]string {
	parseFields := map[string]string{}
	if !isOneOf(parseInput.Status, "open", "in_review", "closed") {
		parseFields["status"] = "Choose open, in_review, or closed."
	}
	if strings.TrimSpace(parseInput.DiscrepancySummary) == "" {
		parseFields["discrepancy_summary"] = "Summarize the receiving discrepancy or closeout note."
	}
	return parseFields
}

func validatePurchaseOrderStatusRequest(parseInput purchaseOrderStatusRequest) map[string]string {
	parseFields := map[string]string{}
	if !isOneOf(parseInput.Status, "submitted", "approved", "on_hold", "received") {
		parseFields["status"] = "Choose submitted, approved, on_hold, or received."
	}
	return parseFields
}

func validatePurchaseOrderCreateRequest(parseInput purchaseOrderCreateRequest) map[string]string {
	parseFields := map[string]string{}
	if strings.TrimSpace(parseInput.VendorName) == "" {
		parseFields["vendor_name"] = "Enter the vendor for this replenishment order."
	}
	if strings.TrimSpace(parseInput.WarehouseID) == "" {
		parseFields["warehouse_id"] = "Choose the receiving warehouse."
	}
	if strings.TrimSpace(parseInput.ProductSKU) == "" {
		parseFields["product_sku"] = "Choose the inventory item to order."
	}
	if parseInput.Quantity < 1 {
		parseFields["quantity"] = "Order quantity must be at least 1."
	}
	if strings.TrimSpace(parseInput.ETA) == "" {
		parseFields["eta"] = "Enter the expected arrival window."
	}
	if strings.TrimSpace(parseInput.PriorityNote) == "" {
		parseFields["priority_note"] = "Add a short purchasing note for the lane."
	}
	if !isOneOf(parseInput.Status, "draft", "submitted", "approved", "on_hold") {
		parseFields["status"] = "Choose a valid purchase-order status."
	}
	return parseFields
}

func looksLikeEmail(parseValue string) bool {
	_, parseErr := mail.ParseAddress(strings.TrimSpace(parseValue))
	return parseErr == nil
}

func isOneOf(parseValue string, parseAllowed ...string) bool {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseValue))
	for _, parseItem := range parseAllowed {
		if parseTrimmed == strings.ToLower(parseItem) {
			return true
		}
	}
	return false
}
