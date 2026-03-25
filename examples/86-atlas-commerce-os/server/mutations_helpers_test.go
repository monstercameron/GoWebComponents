package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
)

func TestDecodeBodyOrForm_JSONFormAndErrors(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		type payload struct {
			Name string `json:"name"`
		}
		req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(`{"name":"atlas"}`))
		req.Header.Set("Content-Type", "application/json")
		var got payload
		if err := decodeBodyOrForm(req, &got, func(values url.Values) {}); err != nil {
			t.Fatalf("decodeBodyOrForm json: %v", err)
		}
		if got.Name != "atlas" {
			t.Fatalf("expected json decode to set name=atlas, got %q", got.Name)
		}
	})

	t.Run("form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader("name=atlas"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		gotName := ""
		if err := decodeBodyOrForm(req, &struct{}{}, func(values url.Values) {
			gotName = values.Get("name")
		}); err != nil {
			t.Fatalf("decodeBodyOrForm form: %v", err)
		}
		if gotName != "atlas" {
			t.Fatalf("expected form decode to set name=atlas, got %q", gotName)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		if err := decodeBodyOrForm(req, &struct{}{}, func(values url.Values) {}); err == nil {
			t.Fatal("expected json decode error")
		}
	})

	t.Run("invalid form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader("a=%zz"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err := decodeBodyOrForm(req, &struct{}{}, func(values url.Values) {}); err == nil {
			t.Fatal("expected form parse error")
		}
	})
}

func TestDecodeRequestHelpers_FromForm(t *testing.T) {
	formRequest := func(body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	comment, err := decodeCommentRequest(formRequest("author_name=Cam&reaction=up&subject=Hi&body=Looks+great"))
	if err != nil {
		t.Fatalf("decodeCommentRequest: %v", err)
	}
	if comment.AuthorName != "Cam" || comment.Reaction != "up" {
		t.Fatalf("unexpected comment decode: %+v", comment)
	}

	quote, err := decodeQuoteRequest(formRequest("requester_name=Amy&company_name=Atlas&email=amy%40x.com&quantity=4&note=fast"))
	if err != nil {
		t.Fatalf("decodeQuoteRequest: %v", err)
	}
	if quote.Quantity != 4 || quote.Email != "amy@x.com" {
		t.Fatalf("unexpected quote decode: %+v", quote)
	}

	restock, err := decodeRestockRequest(formRequest("email=restock%40x.com&preferred_warehouse_id=illinois-hub"))
	if err != nil {
		t.Fatalf("decodeRestockRequest: %v", err)
	}
	if restock.PreferredWarehouseID != "illinois-hub" {
		t.Fatalf("unexpected restock decode: %+v", restock)
	}

	moderation, err := decodeModerationRequest(formRequest("status=approved&reason=clear"))
	if err != nil {
		t.Fatalf("decodeModerationRequest: %v", err)
	}
	if moderation.Status != "approved" {
		t.Fatalf("unexpected moderation decode: %+v", moderation)
	}

	bulk, err := decodeBulkModerationRequest(formRequest("ids=a%2Cb&ids=c&status=flagged&reason=spam"))
	if err != nil {
		t.Fatalf("decodeBulkModerationRequest: %v", err)
	}
	if !reflect.DeepEqual(bulk.IDs, []string{"a", "b", "c"}) {
		t.Fatalf("unexpected bulk ids: %#v", bulk.IDs)
	}

	threshold, err := decodeThresholdRequest(formRequest("warehouse_id=illinois-hub&reorder_point=5&safety_stock=2"))
	if err != nil {
		t.Fatalf("decodeThresholdRequest: %v", err)
	}
	if threshold.ReorderPoint != 5 || threshold.SafetyStock != 2 {
		t.Fatalf("unexpected threshold decode: %+v", threshold)
	}

	inv, err := decodeInventoryUpdateRequest(formRequest("warehouse_id=illinois-hub&on_hand=7&reserved=1&inbound=3&damaged=0&reorder_point=2&safety_stock=1&status=balanced&return_path=%2Fapp%2Finventory"))
	if err != nil {
		t.Fatalf("decodeInventoryUpdateRequest: %v", err)
	}
	if inv.OnHand != 7 || inv.Status != "balanced" || inv.ReturnPath != "/app/inventory" {
		t.Fatalf("unexpected inventory decode: %+v", inv)
	}

	prefs, err := decodePreferencesRequest(formRequest("theme=light&locale=en&density=compact&default_warehouse_id=illinois-hub"))
	if err != nil {
		t.Fatalf("decodePreferencesRequest: %v", err)
	}
	if prefs.Theme != "light" || prefs.Locale != "en" {
		t.Fatalf("unexpected preferences decode: %+v", prefs)
	}

	view, err := decodeSavedViewRequest(formRequest("name=Backlog&scope=inventory&filters_json=%7B%7D&sort_key=urgency&sort_direction=asc&density=compact&warehouse_id=illinois-hub"))
	if err != nil {
		t.Fatalf("decodeSavedViewRequest: %v", err)
	}
	if view.Name != "Backlog" || view.SortDirection != "asc" {
		t.Fatalf("unexpected saved view decode: %+v", view)
	}

	importReq, err := decodeSavedViewImportRequest(formRequest("views_json=%5B%5D"))
	if err != nil {
		t.Fatalf("decodeSavedViewImportRequest: %v", err)
	}
	if importReq.ViewsJSON != "[]" {
		t.Fatalf("unexpected saved view import decode: %+v", importReq)
	}

	transfer, err := decodeTransferRequest(formRequest("source_warehouse_id=illinois-hub&destination_warehouse_id=new-jersey-hub&reason=rebalance&recommended_by=ops"))
	if err != nil {
		t.Fatalf("decodeTransferRequest: %v", err)
	}
	if transfer.SourceWarehouseID != "illinois-hub" || transfer.DestinationWarehouseID != "new-jersey-hub" {
		t.Fatalf("unexpected transfer decode: %+v", transfer)
	}

	receiving, err := decodeReceivingRequest(formRequest("status=closed&discrepancy_summary=none"))
	if err != nil {
		t.Fatalf("decodeReceivingRequest: %v", err)
	}
	if receiving.Status != "closed" {
		t.Fatalf("unexpected receiving decode: %+v", receiving)
	}

	poStatus, err := decodePurchaseOrderStatusRequest(formRequest("status=approved"))
	if err != nil {
		t.Fatalf("decodePurchaseOrderStatusRequest: %v", err)
	}
	if poStatus.Status != "approved" {
		t.Fatalf("unexpected purchase order status decode: %+v", poStatus)
	}

	poCreate, err := decodePurchaseOrderCreateRequest(formRequest("vendor_name=Northwind&warehouse_id=illinois-hub&product_sku=frame-desk&quantity=12&eta=next+week&priority_note=critical&status=submitted&return_path=%2Fapp%2Fpurchase-orders"))
	if err != nil {
		t.Fatalf("decodePurchaseOrderCreateRequest: %v", err)
	}
	if poCreate.Quantity != 12 || poCreate.ReturnPath != "/app/purchase-orders" {
		t.Fatalf("unexpected purchase order create decode: %+v", poCreate)
	}
}

func TestDecodeRequestHelpers_FromJSON(t *testing.T) {
	jsonRequest := func(body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	comment, err := decodeCommentRequest(jsonRequest(`{"author_name":"Cam","reaction":"up","subject":"Hello","body":"Long enough body"}`))
	if err != nil {
		t.Fatalf("decodeCommentRequest json: %v", err)
	}
	if comment.AuthorName != "Cam" || comment.Body == "" {
		t.Fatalf("unexpected json comment decode: %+v", comment)
	}

	poCreate, err := decodePurchaseOrderCreateRequest(jsonRequest(`{"vendor_name":"Northwind","warehouse_id":"illinois-hub","product_sku":"frame-desk","quantity":9,"eta":"tomorrow","priority_note":"rush","status":"draft","return_path":"/app/purchase-orders"}`))
	if err != nil {
		t.Fatalf("decodePurchaseOrderCreateRequest json: %v", err)
	}
	if poCreate.Quantity != 9 || poCreate.Status != "draft" {
		t.Fatalf("unexpected json purchase order decode: %+v", poCreate)
	}
}

func TestMutationsHelpers_Basics(t *testing.T) {
	if got := mustAtoi(" 8 ", 1); got != 8 {
		t.Fatalf("mustAtoi expected 8, got %d", got)
	}
	if got := mustAtoi("bad", 3); got != 3 {
		t.Fatalf("mustAtoi fallback expected 3, got %d", got)
	}

	ids := collectListField(url.Values{"ids": {"a,b", " c ", "", "d"}}, "ids")
	if !reflect.DeepEqual(ids, []string{"a", "b", "c", "d"}) {
		t.Fatalf("collectListField mismatch: %#v", ids)
	}

	if !looksLikeEmail("user@example.com") || looksLikeEmail("bad") {
		t.Fatal("looksLikeEmail did not validate expected addresses")
	}
	if !isOneOf(" Approved ", "pending", "approved") {
		t.Fatal("isOneOf should trim and compare case-insensitively")
	}
	if isOneOf("unknown", "pending", "approved") {
		t.Fatal("isOneOf should reject unknown value")
	}
}

func TestWantsHTMLResponseAndNotice(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Accept", "text/html")
	if !wantsHTMLResponse(req) {
		t.Fatal("expected html response from accept header")
	}

	req = httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if !wantsHTMLResponse(req) {
		t.Fatal("expected html response from form content-type")
	}

	req = httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Content-Type", "application/json")
	if wantsHTMLResponse(req) {
		t.Fatal("expected json request to avoid html response mode")
	}

	with := withNotice("/app/dashboard?x=1", "saved")
	if !strings.Contains(with, "atlas_notice=saved") || !strings.Contains(with, "x=1") {
		t.Fatalf("withNotice should append notice query: %q", with)
	}
	raw := "http://%"
	if got := withNotice(raw, "saved"); got != raw {
		t.Fatalf("expected invalid URL fallback to return raw %q, got %q", raw, got)
	}
}

func TestValidateRequests_InvalidAndValid(t *testing.T) {
	assertHas := func(t *testing.T, fields map[string]string, key string) {
		t.Helper()
		if _, ok := fields[key]; !ok {
			t.Fatalf("expected validation field %q in %#v", key, fields)
		}
	}

	assertEmpty := func(t *testing.T, fields map[string]string) {
		t.Helper()
		if len(fields) != 0 {
			t.Fatalf("expected no validation fields, got %#v", fields)
		}
	}

	assertHas(t, validateCommentRequest(commentRequest{}), "author_name")
	assertHas(t, validateCommentRequest(commentRequest{}), "reaction")
	assertHas(t, validateCommentRequest(commentRequest{}), "subject")
	assertHas(t, validateCommentRequest(commentRequest{}), "body")
	assertEmpty(t, validateCommentRequest(commentRequest{AuthorName: "Cam", Reaction: "up", Subject: "Hi", Body: "Long enough body"}))

	assertHas(t, validateQuoteRequest(quoteRequest{}), "requester_name")
	assertHas(t, validateQuoteRequest(quoteRequest{}), "company_name")
	assertHas(t, validateQuoteRequest(quoteRequest{}), "email")
	assertHas(t, validateQuoteRequest(quoteRequest{}), "quantity")
	assertEmpty(t, validateQuoteRequest(quoteRequest{RequesterName: "A", CompanyName: "B", Email: "a@b.com", Quantity: 2}))

	assertHas(t, validateRestockRequest(restockRequest{}), "email")
	assertHas(t, validateRestockRequest(restockRequest{}), "preferred_warehouse_id")
	assertEmpty(t, validateRestockRequest(restockRequest{Email: "x@y.com", PreferredWarehouseID: "illinois-hub"}))

	assertHas(t, validateModerationRequest(moderationRequest{}), "status")
	assertEmpty(t, validateModerationRequest(moderationRequest{Status: "approved"}))

	assertHas(t, validateBulkModerationRequest(bulkModerationRequest{}), "ids")
	assertHas(t, validateBulkModerationRequest(bulkModerationRequest{}), "status")
	assertEmpty(t, validateBulkModerationRequest(bulkModerationRequest{IDs: []string{"a"}, Status: "flagged"}))

	assertHas(t, validateThresholdRequest(thresholdRequest{}), "warehouse_id")
	assertHas(t, validateThresholdRequest(thresholdRequest{WarehouseID: "illinois-hub", ReorderPoint: 0, SafetyStock: -1}), "reorder_point")
	assertHas(t, validateThresholdRequest(thresholdRequest{WarehouseID: "illinois-hub", ReorderPoint: 1, SafetyStock: -1}), "safety_stock")
	assertEmpty(t, validateThresholdRequest(thresholdRequest{WarehouseID: "illinois-hub", ReorderPoint: 1, SafetyStock: 0}))

	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{}), "warehouse_id")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "bad"}), "status")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", OnHand: -1}), "on_hand")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", Reserved: -1}), "reserved")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", Inbound: -1}), "inbound")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", Damaged: -1}), "damaged")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", ReorderPoint: 0}), "reorder_point")
	assertHas(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", ReorderPoint: 1, SafetyStock: -1}), "safety_stock")
	assertEmpty(t, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", ReorderPoint: 1}))

	assertHas(t, validatePreferencesRequest(preferencesRequest{}), "theme")
	assertHas(t, validatePreferencesRequest(preferencesRequest{}), "locale")
	assertHas(t, validatePreferencesRequest(preferencesRequest{}), "density")
	assertHas(t, validatePreferencesRequest(preferencesRequest{}), "default_warehouse_id")
	assertEmpty(t, validatePreferencesRequest(preferencesRequest{Theme: "dark", Locale: "en", Density: "compact", DefaultWarehouseID: "illinois-hub"}))

	assertHas(t, validateSavedViewRequest(savedViewRequest{}), "name")
	assertHas(t, validateSavedViewRequest(savedViewRequest{}), "scope")
	assertHas(t, validateSavedViewRequest(savedViewRequest{}), "sort_direction")
	assertHas(t, validateSavedViewRequest(savedViewRequest{Name: "x", Scope: "inventory", SortDirection: "asc", FiltersJSON: "{bad"}), "filters_json")
	assertEmpty(t, validateSavedViewRequest(savedViewRequest{Name: "x", Scope: "inventory", SortDirection: "desc", FiltersJSON: `{"k":"v"}`}))

	assertHas(t, validateSavedViewImportRequest(savedViewImportRequest{}), "views_json")
	assertHas(t, validateSavedViewImportRequest(savedViewImportRequest{ViewsJSON: "{"}), "views_json")
	assertEmpty(t, validateSavedViewImportRequest(savedViewImportRequest{ViewsJSON: `{"items":[{"name":"n","scope":"inventory"}]}`}))

	assertHas(t, validateTransferRequest(transferRequest{}), "source_warehouse_id")
	assertHas(t, validateTransferRequest(transferRequest{}), "destination_warehouse_id")
	assertHas(t, validateTransferRequest(transferRequest{}), "reason")
	assertHas(t, validateTransferRequest(transferRequest{SourceWarehouseID: "a", DestinationWarehouseID: "a", Reason: "x"}), "destination_warehouse_id")
	assertEmpty(t, validateTransferRequest(transferRequest{SourceWarehouseID: "a", DestinationWarehouseID: "b", Reason: "x"}))

	assertHas(t, validateReceivingRequest(receivingRequest{}), "status")
	assertHas(t, validateReceivingRequest(receivingRequest{}), "discrepancy_summary")
	assertEmpty(t, validateReceivingRequest(receivingRequest{Status: "closed", DiscrepancySummary: "none"}))

	assertHas(t, validatePurchaseOrderStatusRequest(purchaseOrderStatusRequest{}), "status")
	assertEmpty(t, validatePurchaseOrderStatusRequest(purchaseOrderStatusRequest{Status: "approved"}))

	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "vendor_name")
	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "warehouse_id")
	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "product_sku")
	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "quantity")
	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "eta")
	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "priority_note")
	assertHas(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "status")
	assertEmpty(t, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{
		VendorName:   "Northwind",
		WarehouseID:  "illinois-hub",
		ProductSKU:   "frame-desk",
		Quantity:     4,
		ETA:          "next week",
		PriorityNote: "rush lane",
		Status:       "submitted",
	}))
}

func TestSavedViewTransferDocumentHelpers(t *testing.T) {
	input := []repository.SavedView{{
		Name:          "Backlog",
		Scope:         "inventory",
		SortKey:       "urgency",
		SortDirection: "asc",
		Density:       "compact",
		WarehouseID:   "illinois-hub",
		FiltersJSON:   `{"status":"critical"}`,
	}}
	doc := buildSavedViewTransferDocument(input)
	if len(doc.Items) != 1 || doc.Items[0].Name != "Backlog" {
		t.Fatalf("unexpected transfer document: %#v", doc)
	}

	parsed, err := parseSavedViewTransferDocument(`{"items":[{"name":"Backlog","scope":"inventory","sortKey":"urgency","sortDirection":"asc","density":"compact","warehouseId":"illinois-hub","filtersJSON":"{}"}]}`)
	if err != nil {
		t.Fatalf("parseSavedViewTransferDocument object: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "Backlog" {
		t.Fatalf("unexpected parsed object payload: %#v", parsed)
	}

	parsed, err = parseSavedViewTransferDocument(`[{"name":"Fallback","scope":"inventory","sortKey":"urgency","sortDirection":"desc","density":"comfortable","warehouseId":"new-jersey-hub","filtersJSON":"{}"}]`)
	if err != nil {
		t.Fatalf("parseSavedViewTransferDocument array: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "Fallback" {
		t.Fatalf("unexpected parsed array payload: %#v", parsed)
	}

	if _, err := parseSavedViewTransferDocument(" "); err == nil {
		t.Fatal("expected error for empty payload")
	}
	if _, err := parseSavedViewTransferDocument("{"); err == nil {
		t.Fatal("expected error for invalid payload")
	}
}
