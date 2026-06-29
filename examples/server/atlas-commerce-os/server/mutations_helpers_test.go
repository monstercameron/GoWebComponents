package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/atlas-commerce-os/shared/repository"
)

func TestDecodeBodyOrForm_JSONFormAndErrors(parseT *testing.T) {
	parseT.Run("json", func(parseT2 *testing.T) {
		type payload struct {
			Name string `json:"name"`
		}
		parseReq := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(`{"name":"atlas"}`))
		parseReq.Header.Set("Content-Type", "application/json")
		var parseGot payload
		if parseErr := decodeBodyOrForm(parseReq, &parseGot, func(parseValues url.Values) {}); parseErr != nil {
			parseT2.Fatalf("decodeBodyOrForm json: %v", parseErr)
		}
		if parseGot.Name != "atlas" {
			parseT2.Fatalf("expected json decode to set name=atlas, got %q", parseGot.Name)
		}
	})

	parseT.Run("form", func(parseT3 *testing.T) {
		parseReq2 := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader("name=atlas"))
		parseReq2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		parseGotName := ""
		if parseErr2 := decodeBodyOrForm(parseReq2, &struct{}{}, func(parseValues2 url.Values) {
			parseGotName = parseValues2.Get("name")
		}); parseErr2 != nil {
			parseT3.Fatalf("decodeBodyOrForm form: %v", parseErr2)
		}
		if parseGotName != "atlas" {
			parseT3.Fatalf("expected form decode to set name=atlas, got %q", parseGotName)
		}
	})

	parseT.Run("invalid json", func(parseT4 *testing.T) {
		parseReq3 := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader("{"))
		parseReq3.Header.Set("Content-Type", "application/json")
		if parseErr3 := decodeBodyOrForm(parseReq3, &struct{}{}, func(parseValues3 url.Values) {}); parseErr3 == nil {
			parseT4.Fatal("expected json decode error")
		}
	})

	parseT.Run("invalid form", func(parseT5 *testing.T) {
		parseReq4 := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader("a=%zz"))
		parseReq4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if parseErr4 := decodeBodyOrForm(parseReq4, &struct{}{}, func(parseValues4 url.Values) {}); parseErr4 == nil {
			parseT5.Fatal("expected form parse error")
		}
	})
}

func TestDecodeRequestHelpers_FromForm(parseT *testing.T) {
	parseFormRequest := func(parseBody string) *http.Request {
		parseReq := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(parseBody))
		parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return parseReq
	}

	parseComment, parseErr := decodeCommentRequest(parseFormRequest("author_name=Cam&reaction=up&subject=Hi&body=Looks+great"))
	if parseErr != nil {
		parseT.Fatalf("decodeCommentRequest: %v", parseErr)
	}
	if parseComment.AuthorName != "Cam" || parseComment.Reaction != "up" {
		parseT.Fatalf("unexpected comment decode: %+v", parseComment)
	}

	parseQuote, parseErr := decodeQuoteRequest(parseFormRequest("requester_name=Amy&company_name=Atlas&email=amy%40x.com&quantity=4&note=fast"))
	if parseErr != nil {
		parseT.Fatalf("decodeQuoteRequest: %v", parseErr)
	}
	if parseQuote.Quantity != 4 || parseQuote.Email != "amy@x.com" {
		parseT.Fatalf("unexpected quote decode: %+v", parseQuote)
	}

	parseRestock, parseErr := decodeRestockRequest(parseFormRequest("email=restock%40x.com&preferred_warehouse_id=illinois-hub"))
	if parseErr != nil {
		parseT.Fatalf("decodeRestockRequest: %v", parseErr)
	}
	if parseRestock.PreferredWarehouseID != "illinois-hub" {
		parseT.Fatalf("unexpected restock decode: %+v", parseRestock)
	}

	parseModeration, parseErr := decodeModerationRequest(parseFormRequest("status=approved&reason=clear"))
	if parseErr != nil {
		parseT.Fatalf("decodeModerationRequest: %v", parseErr)
	}
	if parseModeration.Status != "approved" {
		parseT.Fatalf("unexpected moderation decode: %+v", parseModeration)
	}

	parseBulk, parseErr := decodeBulkModerationRequest(parseFormRequest("ids=a%2Cb&ids=c&status=flagged&reason=spam"))
	if parseErr != nil {
		parseT.Fatalf("decodeBulkModerationRequest: %v", parseErr)
	}
	if !reflect.DeepEqual(parseBulk.IDs, []string{"a", "b", "c"}) {
		parseT.Fatalf("unexpected bulk ids: %#v", parseBulk.IDs)
	}

	parseThreshold, parseErr := decodeThresholdRequest(parseFormRequest("warehouse_id=illinois-hub&reorder_point=5&safety_stock=2"))
	if parseErr != nil {
		parseT.Fatalf("decodeThresholdRequest: %v", parseErr)
	}
	if parseThreshold.ReorderPoint != 5 || parseThreshold.SafetyStock != 2 {
		parseT.Fatalf("unexpected threshold decode: %+v", parseThreshold)
	}

	parseInv, parseErr := decodeInventoryUpdateRequest(parseFormRequest("warehouse_id=illinois-hub&on_hand=7&reserved=1&inbound=3&damaged=0&reorder_point=2&safety_stock=1&status=balanced&return_path=%2Fapp%2Finventory"))
	if parseErr != nil {
		parseT.Fatalf("decodeInventoryUpdateRequest: %v", parseErr)
	}
	if parseInv.OnHand != 7 || parseInv.Status != "balanced" || parseInv.ReturnPath != "/app/inventory" {
		parseT.Fatalf("unexpected inventory decode: %+v", parseInv)
	}

	parsePrefs, parseErr := decodePreferencesRequest(parseFormRequest("theme=light&locale=en&density=compact&default_warehouse_id=illinois-hub"))
	if parseErr != nil {
		parseT.Fatalf("decodePreferencesRequest: %v", parseErr)
	}
	if parsePrefs.Theme != "light" || parsePrefs.Locale != "en" {
		parseT.Fatalf("unexpected preferences decode: %+v", parsePrefs)
	}

	parseView, parseErr := decodeSavedViewRequest(parseFormRequest("name=Backlog&scope=inventory&filters_json=%7B%7D&sort_key=urgency&sort_direction=asc&density=compact&warehouse_id=illinois-hub"))
	if parseErr != nil {
		parseT.Fatalf("decodeSavedViewRequest: %v", parseErr)
	}
	if parseView.Name != "Backlog" || parseView.SortDirection != "asc" {
		parseT.Fatalf("unexpected saved view decode: %+v", parseView)
	}

	parseImportReq, parseErr := decodeSavedViewImportRequest(parseFormRequest("views_json=%5B%5D"))
	if parseErr != nil {
		parseT.Fatalf("decodeSavedViewImportRequest: %v", parseErr)
	}
	if parseImportReq.ViewsJSON != "[]" {
		parseT.Fatalf("unexpected saved view import decode: %+v", parseImportReq)
	}

	parseTransfer, parseErr := decodeTransferRequest(parseFormRequest("source_warehouse_id=illinois-hub&destination_warehouse_id=new-jersey-hub&reason=rebalance&recommended_by=ops"))
	if parseErr != nil {
		parseT.Fatalf("decodeTransferRequest: %v", parseErr)
	}
	if parseTransfer.SourceWarehouseID != "illinois-hub" || parseTransfer.DestinationWarehouseID != "new-jersey-hub" {
		parseT.Fatalf("unexpected transfer decode: %+v", parseTransfer)
	}

	parseReceiving, parseErr := decodeReceivingRequest(parseFormRequest("status=closed&discrepancy_summary=none"))
	if parseErr != nil {
		parseT.Fatalf("decodeReceivingRequest: %v", parseErr)
	}
	if parseReceiving.Status != "closed" {
		parseT.Fatalf("unexpected receiving decode: %+v", parseReceiving)
	}

	parsePoStatus, parseErr := decodePurchaseOrderStatusRequest(parseFormRequest("status=approved"))
	if parseErr != nil {
		parseT.Fatalf("decodePurchaseOrderStatusRequest: %v", parseErr)
	}
	if parsePoStatus.Status != "approved" {
		parseT.Fatalf("unexpected purchase order status decode: %+v", parsePoStatus)
	}

	parsePoCreate, parseErr := decodePurchaseOrderCreateRequest(parseFormRequest("vendor_name=Northwind&warehouse_id=illinois-hub&product_sku=frame-desk&quantity=12&eta=next+week&priority_note=critical&status=submitted&return_path=%2Fapp%2Fpurchase-orders"))
	if parseErr != nil {
		parseT.Fatalf("decodePurchaseOrderCreateRequest: %v", parseErr)
	}
	if parsePoCreate.Quantity != 12 || parsePoCreate.ReturnPath != "/app/purchase-orders" {
		parseT.Fatalf("unexpected purchase order create decode: %+v", parsePoCreate)
	}
}

func TestDecodeRequestHelpers_FromJSON(parseT *testing.T) {
	parseJsonRequest := func(parseBody string) *http.Request {
		parseReq := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(parseBody))
		parseReq.Header.Set("Content-Type", "application/json")
		return parseReq
	}

	parseComment, parseErr := decodeCommentRequest(parseJsonRequest(`{"author_name":"Cam","reaction":"up","subject":"Hello","body":"Long enough body"}`))
	if parseErr != nil {
		parseT.Fatalf("decodeCommentRequest json: %v", parseErr)
	}
	if parseComment.AuthorName != "Cam" || parseComment.Body == "" {
		parseT.Fatalf("unexpected json comment decode: %+v", parseComment)
	}

	parsePoCreate, parseErr := decodePurchaseOrderCreateRequest(parseJsonRequest(`{"vendor_name":"Northwind","warehouse_id":"illinois-hub","product_sku":"frame-desk","quantity":9,"eta":"tomorrow","priority_note":"rush","status":"draft","return_path":"/app/purchase-orders"}`))
	if parseErr != nil {
		parseT.Fatalf("decodePurchaseOrderCreateRequest json: %v", parseErr)
	}
	if parsePoCreate.Quantity != 9 || parsePoCreate.Status != "draft" {
		parseT.Fatalf("unexpected json purchase order decode: %+v", parsePoCreate)
	}
}

func TestMutationsHelpers_Basics(parseT *testing.T) {
	if parseGot := mustAtoi(" 8 ", 1); parseGot != 8 {
		parseT.Fatalf("mustAtoi expected 8, got %d", parseGot)
	}
	if parseGot2 := mustAtoi("bad", 3); parseGot2 != 3 {
		parseT.Fatalf("mustAtoi fallback expected 3, got %d", parseGot2)
	}

	parseIds := collectListField(url.Values{"ids": {"a,b", " c ", "", "d"}}, "ids")
	if !reflect.DeepEqual(parseIds, []string{"a", "b", "c", "d"}) {
		parseT.Fatalf("collectListField mismatch: %#v", parseIds)
	}

	if !looksLikeEmail("user@example.com") || looksLikeEmail("bad") {
		parseT.Fatal("looksLikeEmail did not validate expected addresses")
	}
	if !isOneOf(" Approved ", "pending", "approved") {
		parseT.Fatal("isOneOf should trim and compare case-insensitively")
	}
	if isOneOf("unknown", "pending", "approved") {
		parseT.Fatal("isOneOf should reject unknown value")
	}
}

func TestWantsHTMLResponseAndNotice(parseT *testing.T) {
	parseReq := httptest.NewRequest(http.MethodPost, "/x", nil)
	parseReq.Header.Set("Accept", "text/html")
	if !wantsHTMLResponse(parseReq) {
		parseT.Fatal("expected html response from accept header")
	}

	parseReq = httptest.NewRequest(http.MethodPost, "/x", nil)
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if !wantsHTMLResponse(parseReq) {
		parseT.Fatal("expected html response from form content-type")
	}

	parseReq = httptest.NewRequest(http.MethodPost, "/x", nil)
	parseReq.Header.Set("Content-Type", "application/json")
	if wantsHTMLResponse(parseReq) {
		parseT.Fatal("expected json request to avoid html response mode")
	}

	parseWith := withNotice("/app/dashboard?x=1", "saved")
	if !strings.Contains(parseWith, "atlas_notice=saved") || !strings.Contains(parseWith, "x=1") {
		parseT.Fatalf("withNotice should append notice query: %q", parseWith)
	}
	parseRaw := "http://%"
	if parseGot := withNotice(parseRaw, "saved"); parseGot != parseRaw {
		parseT.Fatalf("expected invalid URL fallback to return raw %q, got %q", parseRaw, parseGot)
	}
}

func TestValidateRequests_InvalidAndValid(parseT *testing.T) {
	parseAssertHas := func(parseT2 *testing.T, parseFields map[string]string, parseKey string) {
		parseT2.Helper()
		if _, parseOk := parseFields[parseKey]; !parseOk {
			parseT2.Fatalf("expected validation field %q in %#v", parseKey, parseFields)
		}
	}

	parseAssertEmpty := func(parseT3 *testing.T, parseFields2 map[string]string) {
		parseT3.Helper()
		if len(parseFields2) != 0 {
			parseT3.Fatalf("expected no validation fields, got %#v", parseFields2)
		}
	}

	parseAssertHas(parseT, validateCommentRequest(commentRequest{}), "author_name")
	parseAssertHas(parseT, validateCommentRequest(commentRequest{}), "reaction")
	parseAssertHas(parseT, validateCommentRequest(commentRequest{}), "subject")
	parseAssertHas(parseT, validateCommentRequest(commentRequest{}), "body")
	parseAssertEmpty(parseT, validateCommentRequest(commentRequest{AuthorName: "Cam", Reaction: "up", Subject: "Hi", Body: "Long enough body"}))

	parseAssertHas(parseT, validateQuoteRequest(quoteRequest{}), "requester_name")
	parseAssertHas(parseT, validateQuoteRequest(quoteRequest{}), "company_name")
	parseAssertHas(parseT, validateQuoteRequest(quoteRequest{}), "email")
	parseAssertHas(parseT, validateQuoteRequest(quoteRequest{}), "quantity")
	parseAssertEmpty(parseT, validateQuoteRequest(quoteRequest{RequesterName: "A", CompanyName: "B", Email: "a@b.com", Quantity: 2}))

	parseAssertHas(parseT, validateRestockRequest(restockRequest{}), "email")
	parseAssertHas(parseT, validateRestockRequest(restockRequest{}), "preferred_warehouse_id")
	parseAssertEmpty(parseT, validateRestockRequest(restockRequest{Email: "x@y.com", PreferredWarehouseID: "illinois-hub"}))

	parseAssertHas(parseT, validateModerationRequest(moderationRequest{}), "status")
	parseAssertEmpty(parseT, validateModerationRequest(moderationRequest{Status: "approved"}))

	parseAssertHas(parseT, validateBulkModerationRequest(bulkModerationRequest{}), "ids")
	parseAssertHas(parseT, validateBulkModerationRequest(bulkModerationRequest{}), "status")
	parseAssertEmpty(parseT, validateBulkModerationRequest(bulkModerationRequest{IDs: []string{"a"}, Status: "flagged"}))

	parseAssertHas(parseT, validateThresholdRequest(thresholdRequest{}), "warehouse_id")
	parseAssertHas(parseT, validateThresholdRequest(thresholdRequest{WarehouseID: "illinois-hub", ReorderPoint: 0, SafetyStock: -1}), "reorder_point")
	parseAssertHas(parseT, validateThresholdRequest(thresholdRequest{WarehouseID: "illinois-hub", ReorderPoint: 1, SafetyStock: -1}), "safety_stock")
	parseAssertEmpty(parseT, validateThresholdRequest(thresholdRequest{WarehouseID: "illinois-hub", ReorderPoint: 1, SafetyStock: 0}))

	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{}), "warehouse_id")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "bad"}), "status")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", OnHand: -1}), "on_hand")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", Reserved: -1}), "reserved")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", Inbound: -1}), "inbound")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", Damaged: -1}), "damaged")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", ReorderPoint: 0}), "reorder_point")
	parseAssertHas(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", ReorderPoint: 1, SafetyStock: -1}), "safety_stock")
	parseAssertEmpty(parseT, validateInventoryUpdateRequest(inventoryUpdateRequest{WarehouseID: "w", Status: "balanced", ReorderPoint: 1}))

	parseAssertHas(parseT, validatePreferencesRequest(preferencesRequest{}), "theme")
	parseAssertHas(parseT, validatePreferencesRequest(preferencesRequest{}), "locale")
	parseAssertHas(parseT, validatePreferencesRequest(preferencesRequest{}), "density")
	parseAssertHas(parseT, validatePreferencesRequest(preferencesRequest{}), "default_warehouse_id")
	parseAssertEmpty(parseT, validatePreferencesRequest(preferencesRequest{Theme: "dark", Locale: "en", Density: "compact", DefaultWarehouseID: "illinois-hub"}))

	parseAssertHas(parseT, validateSavedViewRequest(savedViewRequest{}), "name")
	parseAssertHas(parseT, validateSavedViewRequest(savedViewRequest{}), "scope")
	parseAssertHas(parseT, validateSavedViewRequest(savedViewRequest{}), "sort_direction")
	parseAssertHas(parseT, validateSavedViewRequest(savedViewRequest{Name: "x", Scope: "inventory", SortDirection: "asc", FiltersJSON: "{bad"}), "filters_json")
	parseAssertEmpty(parseT, validateSavedViewRequest(savedViewRequest{Name: "x", Scope: "inventory", SortDirection: "desc", FiltersJSON: `{"k":"v"}`}))

	parseAssertHas(parseT, validateSavedViewImportRequest(savedViewImportRequest{}), "views_json")
	parseAssertHas(parseT, validateSavedViewImportRequest(savedViewImportRequest{ViewsJSON: "{"}), "views_json")
	parseAssertEmpty(parseT, validateSavedViewImportRequest(savedViewImportRequest{ViewsJSON: `{"items":[{"name":"n","scope":"inventory"}]}`}))

	parseAssertHas(parseT, validateTransferRequest(transferRequest{}), "source_warehouse_id")
	parseAssertHas(parseT, validateTransferRequest(transferRequest{}), "destination_warehouse_id")
	parseAssertHas(parseT, validateTransferRequest(transferRequest{}), "reason")
	parseAssertHas(parseT, validateTransferRequest(transferRequest{SourceWarehouseID: "a", DestinationWarehouseID: "a", Reason: "x"}), "destination_warehouse_id")
	parseAssertEmpty(parseT, validateTransferRequest(transferRequest{SourceWarehouseID: "a", DestinationWarehouseID: "b", Reason: "x"}))

	parseAssertHas(parseT, validateReceivingRequest(receivingRequest{}), "status")
	parseAssertHas(parseT, validateReceivingRequest(receivingRequest{}), "discrepancy_summary")
	parseAssertHas(parseT, validateReceivingRequest(receivingRequest{Status: "review", DiscrepancySummary: "legacy value"}), "status")
	parseAssertEmpty(parseT, validateReceivingRequest(receivingRequest{Status: "open", DiscrepancySummary: "waiting for count"}))
	parseAssertEmpty(parseT, validateReceivingRequest(receivingRequest{Status: "in_review", DiscrepancySummary: "classification in progress"}))
	parseAssertEmpty(parseT, validateReceivingRequest(receivingRequest{Status: "closed", DiscrepancySummary: "none"}))

	parseAssertHas(parseT, validatePurchaseOrderStatusRequest(purchaseOrderStatusRequest{}), "status")
	parseAssertEmpty(parseT, validatePurchaseOrderStatusRequest(purchaseOrderStatusRequest{Status: "approved"}))

	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "vendor_name")
	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "warehouse_id")
	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "product_sku")
	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "quantity")
	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "eta")
	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "priority_note")
	parseAssertHas(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{}), "status")
	parseAssertEmpty(parseT, validatePurchaseOrderCreateRequest(purchaseOrderCreateRequest{
		VendorName:   "Northwind",
		WarehouseID:  "illinois-hub",
		ProductSKU:   "frame-desk",
		Quantity:     4,
		ETA:          "next week",
		PriorityNote: "rush lane",
		Status:       "submitted",
	}))
}

func TestSavedViewTransferDocumentHelpers(parseT *testing.T) {
	parseInput := []repository.SavedView{{
		Name:          "Backlog",
		Scope:         "inventory",
		SortKey:       "urgency",
		SortDirection: "asc",
		Density:       "compact",
		WarehouseID:   "illinois-hub",
		FiltersJSON:   `{"status":"critical"}`,
	}}
	parseDoc := buildSavedViewTransferDocument(parseInput)
	if len(parseDoc.Items) != 1 || parseDoc.Items[0].Name != "Backlog" {
		parseT.Fatalf("unexpected transfer document: %#v", parseDoc)
	}

	parseParsed, parseErr := parseSavedViewTransferDocument(`{"items":[{"name":"Backlog","scope":"inventory","sortKey":"urgency","sortDirection":"asc","density":"compact","warehouseId":"illinois-hub","filtersJSON":"{}"}]}`)
	if parseErr != nil {
		parseT.Fatalf("parseSavedViewTransferDocument object: %v", parseErr)
	}
	if len(parseParsed) != 1 || parseParsed[0].Name != "Backlog" {
		parseT.Fatalf("unexpected parsed object payload: %#v", parseParsed)
	}

	parseParsed, parseErr = parseSavedViewTransferDocument(`[{"name":"Fallback","scope":"inventory","sortKey":"urgency","sortDirection":"desc","density":"comfortable","warehouseId":"new-jersey-hub","filtersJSON":"{}"}]`)
	if parseErr != nil {
		parseT.Fatalf("parseSavedViewTransferDocument array: %v", parseErr)
	}
	if len(parseParsed) != 1 || parseParsed[0].Name != "Fallback" {
		parseT.Fatalf("unexpected parsed array payload: %#v", parseParsed)
	}

	if _, parseErr2 := parseSavedViewTransferDocument(" "); parseErr2 == nil {
		parseT.Fatal("expected error for empty payload")
	}
	if _, parseErr3 := parseSavedViewTransferDocument("{"); parseErr3 == nil {
		parseT.Fatal("expected error for invalid payload")
	}
}
