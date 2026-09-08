package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestAtlasPureHelpersCoverAvailabilityAndMutations verifies copy and routing helper branches.
func TestAtlasPureHelpersCoverAvailabilityAndMutations(parseT *testing.T) {
	parseAvailabilityCases := []struct {
		available int
		inbound   int
		want      string
	}{
		{available: 4, inbound: 2, want: "Current stock covers near-term demand"},
		{available: 4, inbound: 0, want: "immediate demand"},
		{available: 0, inbound: 3, want: "replenishment is already moving"},
		{available: 0, inbound: 0, want: "Inventory is currently constrained"},
	}
	for _, parseCase := range parseAvailabilityCases {
		if parseGot := availabilityStoryCopy(parseCase.available, parseCase.inbound); !strings.Contains(parseGot, parseCase.want) {
			parseT.Fatalf("availabilityStoryCopy(%d,%d) = %q, want substring %q", parseCase.available, parseCase.inbound, parseGot, parseCase.want)
		}
	}

	parsePlanHeadline, parsePlanCopy := availabilitySupportPlan(3, 1)
	if !strings.Contains(parsePlanHeadline, "support the project now") || !strings.Contains(parsePlanCopy, "quote capture") {
		parseT.Fatalf("unexpected immediate availability plan headline=%q copy=%q", parsePlanHeadline, parsePlanCopy)
	}
	parsePlanHeadline, parsePlanCopy = availabilitySupportPlan(0, 2)
	if !strings.Contains(parsePlanHeadline, "Reserve the next inbound wave") || !strings.Contains(parsePlanCopy, "Preserve buyer intent") {
		parseT.Fatalf("unexpected inbound availability plan headline=%q copy=%q", parsePlanHeadline, parsePlanCopy)
	}
	parsePlanHeadline, parsePlanCopy = availabilitySupportPlan(0, 0)
	if !strings.Contains(parsePlanHeadline, "Stay attached to this region") || !strings.Contains(parsePlanCopy, "notification path") {
		parseT.Fatalf("unexpected constrained availability plan headline=%q copy=%q", parsePlanHeadline, parsePlanCopy)
	}

	if parsePrefixes := mutationRequestPrefixesForRoutePrefix("/shop"); len(parsePrefixes) != 2 || parsePrefixes[0] != "/api/public/catalog" {
		parseT.Fatalf("unexpected shop mutation prefixes %#v", parsePrefixes)
	}
	if parsePrefixes := mutationRequestPrefixesForRoutePrefix("/app/settings"); len(parsePrefixes) != 2 || parsePrefixes[1] != "/api/app/saved-views" {
		parseT.Fatalf("unexpected settings mutation prefixes %#v", parsePrefixes)
	}
	if parsePrefixes := mutationRequestPrefixesForRoutePrefix("/unknown"); parsePrefixes != nil {
		parseT.Fatalf("expected nil mutation prefixes for unknown route, got %#v", parsePrefixes)
	}

	parseOriginalPayload := samplePayloadForRoute(RouteCatalog, map[string]any{"items": 1}, nil)
	parseOriginalPayload.Route.Query = map[string][]string{"q": {"desk"}}
	parseOriginalPayload.Route.Params = map[string]string{"warehouseId": "illinois-hub"}
	parseOriginalPayload.Data = map[string]any{"page": map[string]any{"total": 4}}
	parseOriginalPayload.Requests = map[string]Request{"page": {URL: "/api/public/catalog", Data: map[string]any{"page": "catalog"}}}
	parseOriginalPayload.SavedViews = []SavedViewPayload{{Name: "Low stock", Filters: map[string]string{"status": "critical"}}}
	parseOriginalPayload.User = &UserSession{ID: "user-1", DefaultWarehouse: "illinois-hub"}

	parseClonedPayload := clonePayloadForResourceCache(parseOriginalPayload)
	parseClonedPayload.Route.Query["q"][0] = "chair"
	parseClonedPayload.Route.Params["warehouseId"] = "nevada-hub"
	parseClonedPayload.Requests["page"] = Request{URL: "/api/public/catalog", Data: map[string]any{"page": "mutated"}}
	parseClonedPayload.SavedViews[0].Name = "Changed"
	parseClonedPayload.User.DefaultWarehouse = "new-jersey-hub"
	if parseOriginalPayload.Route.Query["q"][0] != "desk" || parseOriginalPayload.Route.Params["warehouseId"] != "illinois-hub" || parseOriginalPayload.SavedViews[0].Name != "Low stock" || parseOriginalPayload.User.DefaultWarehouse != "illinois-hub" {
		parseT.Fatalf("expected cloned payload mutation to leave original untouched, got original=%#v clone=%#v", parseOriginalPayload, parseClonedPayload)
	}
}

// TestAtlasInventoryAndProductHelpersRenderMarkup verifies small inventory and product helper outputs.
func TestAtlasInventoryAndProductHelpersRenderMarkup(parseT *testing.T) {
	parseRows := sampleInventoryRows()
	parseProducts := sampleProductAdminCards()

	// Market pressure is a demand reading, not a status: nothing has gone wrong when a
	// market is hot and nothing has passed a check when it is steady. It therefore gets
	// the same quiet meta class whatever it says, and this test pins that rather than
	// the four saturated palettes it used to assert (rose / amber / slate / emerald).
	// Those assertions could only pass while a call site was choosing a colour by hand,
	// which is exactly what the design system has no API for.
	parseQuietClass := marketPressureClass("steady")
	if parseQuietClass == "" {
		parseT.Fatal("expected market pressure to fold to a class")
	}
	for _, parseLabel := range []string{"hot market", "growing demand", "softening", "steady"} {
		if parseClass := marketPressureClass(parseLabel); parseClass != parseQuietClass {
			parseT.Fatalf("market pressure %q returned %q, want the same neutral meta class %q", parseLabel, parseClass, parseQuietClass)
		}
	}
	if parseMin := minInt(3, 7); parseMin != 3 {
		parseT.Fatalf("minInt(3,7) = %d, want 3", parseMin)
	}

	parseLaneOptions := inventoryWarehouseLaneOptions(parseRows)
	if len(parseLaneOptions) == 0 || parseLaneOptions[0].Label == "" {
		parseT.Fatalf("expected warehouse lane options, got %#v", parseLaneOptions)
	}
	if parseLabel := inventoryWarehouseLabel(parseRows[0].WarehouseID, parseRows); parseLabel != parseRows[0].WarehouseName {
		parseT.Fatalf("expected warehouse label to use warehouse name, got %q", parseLabel)
	}
	if parseLabel := inventoryWarehouseLabel("unknown", parseRows); parseLabel != "unknown" {
		parseT.Fatalf("expected unknown warehouse label to fall back to id, got %q", parseLabel)
	}

	parseMarkup := renderAtlasMarkupForTest(parseT, inventoryTextField("query", "Search", "desk"))
	if !strings.Contains(parseMarkup, "Search") || !strings.Contains(parseMarkup, "value=\"desk\"") || !strings.Contains(parseMarkup, "name=\"query\"") {
		parseT.Fatalf("unexpected inventory text field markup %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, inventoryNumberField("quantity", "Quantity", "12"))
	if !strings.Contains(parseMarkup, "type=\"number\"") || !strings.Contains(parseMarkup, "Quantity") {
		parseT.Fatalf("unexpected inventory number field markup %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, inventorySelectField("status", "Status", "critical", inventoryStatusOptions()))
	if !strings.Contains(parseMarkup, "<select") || !strings.Contains(parseMarkup, "Critical") || !strings.Contains(parseMarkup, "selected") {
		parseT.Fatalf("unexpected inventory select markup %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, warehouseFeatureCard("Urgent replenishment", "Route warehouse operators into the right lane."))
	if !strings.Contains(parseMarkup, "Warehouse operations") || !strings.Contains(parseMarkup, "Urgent replenishment") {
		parseT.Fatalf("unexpected warehouse feature card markup %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, warehouseListCard("Open issues"))
	if !strings.Contains(parseMarkup, "Open issues") || !strings.Contains(parseMarkup, "No records yet.") {
		parseT.Fatalf("unexpected warehouse list card markup %q", parseMarkup)
	}

	parseMarkup = renderAtlasMarkupForTest(parseT, warehouseScopedField("warehouse_id", "illinois-hub", "illinois-hub", nil))
	if !strings.Contains(parseMarkup, "type=\"hidden\"") {
		parseT.Fatalf("expected locked warehouse field to render hidden input, got %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, warehouseScopedField("warehouse_id", "illinois-hub", "", inventoryWarehouseLaneOptions(parseRows)))
	if !strings.Contains(parseMarkup, "<select") || !strings.Contains(parseMarkup, "Warehouse") {
		parseT.Fatalf("expected unlocked warehouse field to render select input, got %q", parseMarkup)
	}

	parseMarkup = renderAtlasMarkupForTest(parseT, productListMetric("Units", "144"))
	if !strings.Contains(parseMarkup, "Units") || !strings.Contains(parseMarkup, "144") {
		parseT.Fatalf("unexpected product metric markup %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, publicCommentReactionBadge("down"))
	if !strings.Contains(parseMarkup, "Thumbs down") {
		parseT.Fatalf("unexpected reaction badge markup %q", parseMarkup)
	}
	parseMarkup = renderAtlasMarkupForTest(parseT, publicCommentReactionInput("reaction", "reaction-error", "", ui.Handler{}, "Choose a reaction"))
	if !strings.Contains(parseMarkup, "Thumbs up") || !strings.Contains(parseMarkup, "Choose a reaction") {
		parseT.Fatalf("unexpected reaction input markup %q", parseMarkup)
	}

	if parseVolume := productVolume(parseProducts[0]); parseVolume != parseProducts[0].Volume {
		parseT.Fatalf("expected explicit volume to win, got %d", parseVolume)
	}
	parseFallbackProduct := parseProducts[0]
	parseFallbackProduct.Volume = 0
	parseFallbackProduct.Available = 3
	parseFallbackProduct.Inbound = 2
	if parseVolume := productVolume(parseFallbackProduct); parseVolume != 5 {
		parseT.Fatalf("expected fallback product volume from available+inbound, got %d", parseVolume)
	}
	if parseHref := warehouseItemProfileHref(parseProducts[0]); !strings.Contains(parseHref, parseProducts[0].WarehouseID) {
		parseT.Fatalf("expected warehouse item href to include warehouse id, got %q", parseHref)
	}
	parseFallbackProduct.WarehouseID = ""
	if parseHref := warehouseItemProfileHref(parseFallbackProduct); parseHref != "/app/inventory/"+parseFallbackProduct.SKU {
		parseT.Fatalf("expected fallback inventory href, got %q", parseHref)
	}

	if parseNode := productDraftChangeSummary(productEditorFormState{Title: parseProducts[0].Title}, ui.Previous[productEditorFormState]{}); parseNode != nil {
		parseT.Fatalf("expected zero previous state to suppress draft summary, got %#v", parseNode)
	}
	if parseNode := thresholdHistoryChangeSummary([]inventoryThresholdHistoryItem{{WarehouseID: "illinois-hub", ReorderPoint: 8, SafetyStock: 3}}, ui.Previous[string]{}, "latest"); parseNode != nil {
		parseT.Fatalf("expected zero previous signature to suppress threshold history summary, got %#v", parseNode)
	}
}

// TestAtlasPublicCommentHelpersNormalizeAndSummarize verifies public comment helper branches.
func TestAtlasPublicCommentHelpersNormalizeAndSummarize(parseT *testing.T) {
	parseThumbsUp, parseThumbsDown := publicCommentReactionCounts(sampleCommentRecords())
	if parseThumbsUp == 0 || parseThumbsDown == 0 {
		parseT.Fatalf("expected mixed reaction counts, got up=%d down=%d", parseThumbsUp, parseThumbsDown)
	}
	if parseSummary, parsePct := publicCommentRatioSummary(0, 0); parseSummary != "No ratings yet" || parsePct != 0 {
		parseT.Fatalf("unexpected zero ratio summary summary=%q pct=%d", parseSummary, parsePct)
	}
	if parseSummary, parsePct := publicCommentRatioSummary(3, 1); parseSummary != "75% thumbs up" || parsePct != 75 {
		parseT.Fatalf("unexpected ratio summary summary=%q pct=%d", parseSummary, parsePct)
	}

	parseErrors := validatePublicCommentForm(publicCommentFormState{})
	if len(parseErrors) != 4 {
		parseT.Fatalf("expected all validation errors for blank comment form, got %#v", parseErrors)
	}
	parseNormalized := normalizePublicCommentFieldErrors(ui.FieldErrors{
		"author_name": "Name required",
		"reaction":    "Choose one",
		"subject":     "Headline required",
		"body":        "Body required",
		"other":       "Other field",
	})
	if parseNormalized["AuthorName"] != "Name required" || parseNormalized["Reaction"] != "Choose one" || parseNormalized["Subject"] != "Headline required" || parseNormalized["Body"] != "Body required" || parseNormalized["other"] != "Other field" {
		parseT.Fatalf("unexpected normalized public comment field errors %#v", parseNormalized)
	}
	if normalizePublicCommentFieldErrors(nil) != nil {
		parseT.Fatal("expected nil field errors to stay nil")
	}

	// These three helpers now fold shared/design bundles into ONE hashed class
	// (design.Class), so there is no literal "cursor-wait" or "border-rose-300"
	// substring left to match — the utility strings they used to concatenate are
	// exactly what the design-system conversion removed. What the assertions
	// protected was that each helper's two states are DISTINGUISHABLE and neither
	// is empty, so that is what they assert now, on the folded class name.
	if publicCommentSubmitClass(true) == publicCommentSubmitClass(false) {
		parseT.Fatalf("expected the submitting submit class to differ from the idle one, got %q", publicCommentSubmitClass(true))
	}
	if strings.TrimSpace(publicCommentSubmitClass(false)) == "" {
		parseT.Fatal("expected the idle submit class to fold to a class name")
	}
	if publicCommentSubmitLabel(true) != "Sending..." || publicCommentSubmitLabel(false) != "Share feedback" {
		parseT.Fatal("unexpected public comment submit labels")
	}
	if publicCommentFieldClass(true) == publicCommentFieldClass(false) {
		parseT.Fatalf("expected the invalid field class to differ from the valid one, got %q", publicCommentFieldClass(true))
	}
	if publicCommentTextareaClass(true) == publicCommentTextareaClass(false) {
		parseT.Fatalf("expected the invalid textarea class to differ from the valid one, got %q", publicCommentTextareaClass(true))
	}
	if publicCommentTextareaClass(false) == publicCommentFieldClass(false) {
		parseT.Fatal("expected the textarea class to add its own min-height on top of the shared input primitive")
	}
	if formatPublicCommentDate("") != "recently" || formatPublicCommentDate("2026-03-26T12:34:56Z") != "2026-03-26" || formatPublicCommentDate("short") != "short" {
		parseT.Fatal("unexpected public comment date formatting")
	}

	parseComments := sampleCommentRecords()
	parseClonedComments := cloneCommentRecords(parseComments)
	parseClonedComments[0].Subject = "Changed"
	if parseComments[0].Subject == "Changed" {
		parseT.Fatalf("expected cloneCommentRecords to copy slice values, got %#v", parseComments)
	}
	if parseSignature := commentRecordsSignature(parseComments); !strings.Contains(parseSignature, parseComments[0].ID) || !strings.Contains(parseSignature, parseComments[0].Status) {
		parseT.Fatalf("unexpected comment records signature %q", parseSignature)
	}
	if commentRecordsSignature(nil) != "" {
		parseT.Fatal("expected empty signature for nil comment list")
	}
}
