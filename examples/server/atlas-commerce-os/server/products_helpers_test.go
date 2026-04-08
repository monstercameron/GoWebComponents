package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestDecodeProductRequestFromFormAndJSON(parseT *testing.T) {
	parseT.Run("form", func(parseT2 *testing.T) {
		parseForm := url.Values{
			"sku":                  {" task-lamp "},
			"slug":                 {" task-lamp "},
			"title":                {" Task Lamp "},
			"category":             {" lighting "},
			"price_cents":          {" 12999 "},
			"status":               {" available "},
			"finish":               {" walnut "},
			"summary":              {" Compact task light "},
			"details":              {" Adjustable head "},
			"seo_title":            {" Task Lamp SEO "},
			"seo_description":      {" Detailed SEO description "},
			"warehouse_id":         {" nevada-hub "},
			"current_warehouse_id": {" illinois-hub "},
			"return_warehouse_id":  {" new-jersey-hub "},
			"available":            {" 8 "},
			"inbound":              {" 3 "},
		}
		parseReq := formRequest(parseForm.Encode())

		parseGot, parseErr := decodeProductRequest(parseReq)
		if parseErr != nil {
			parseT2.Fatalf("decodeProductRequest(form): %v", parseErr)
		}
		if parseGot.SKU != " task-lamp " || parseGot.Slug != " task-lamp " || parseGot.PriceCents != 12999 || parseGot.Available != 8 || parseGot.Inbound != 3 {
			parseT2.Fatalf("unexpected form payload: %+v", parseGot)
		}
		if parseGot.CurrentWarehouse != " illinois-hub " || parseGot.ReturnWarehouseID != " new-jersey-hub " {
			parseT2.Fatalf("unexpected warehouse payload: %+v", parseGot)
		}
	})

	parseT.Run("json", func(parseT3 *testing.T) {
		parseReq2 := jsonRequest(`{
			"sku":"frame-desk",
			"slug":"frame-desk",
			"title":"Frame Desk",
			"category":"desks",
			"price_cents":249900,
			"status":"available",
			"finish":"oak",
			"summary":"Editorial summary",
			"details":"Longer details",
			"seo_title":"SEO title",
			"seo_description":"SEO description",
			"warehouse_id":"new-jersey-hub",
			"current_warehouse_id":"illinois-hub",
			"return_warehouse_id":"nevada-hub",
			"available":5,
			"inbound":2
		}`)

		parseGot2, parseErr2 := decodeProductRequest(parseReq2)
		if parseErr2 != nil {
			parseT3.Fatalf("decodeProductRequest(json): %v", parseErr2)
		}
		if parseGot2.SKU != "frame-desk" || parseGot2.Title != "Frame Desk" || parseGot2.PriceCents != 249900 {
			parseT3.Fatalf("unexpected json payload: %+v", parseGot2)
		}
		if parseGot2.WarehouseID != "new-jersey-hub" || parseGot2.CurrentWarehouse != "illinois-hub" || parseGot2.ReturnWarehouseID != "nevada-hub" {
			parseT3.Fatalf("unexpected warehouse payload: %+v", parseGot2)
		}
	})
}

func TestProductHelpersValidationAndSanitization(parseT *testing.T) {
	parseFields := validateProductRequest(productRequest{}, true)
	for _, parseKey := range []string{"sku", "slug", "title", "category", "status", "summary", "warehouse_id"} {
		if _, parseOk := parseFields[parseKey]; !parseOk {
			parseT.Fatalf("expected validation error for %q, got %+v", parseKey, parseFields)
		}
	}

	parseUpdateFields := validateProductRequest(productRequest{
		Title:       "Task Lamp",
		Category:    "lighting",
		PriceCents:  -1,
		Status:      "available",
		Summary:     "summary",
		WarehouseID: "warehouse-a",
		Available:   -2,
		Inbound:     -3,
	}, false)
	if _, parseOk2 := parseUpdateFields["sku"]; parseOk2 {
		parseT.Fatalf("did not expect sku to be required for update: %+v", parseUpdateFields)
	}
	for _, parseKey2 := range []string{"price_cents", "available", "inbound"} {
		if _, parseOk3 := parseUpdateFields[parseKey2]; !parseOk3 {
			parseT.Fatalf("expected numeric validation error for %q, got %+v", parseKey2, parseUpdateFields)
		}
	}

	if parseGot := sanitizeWarehouseReturnID(" new-jersey-hub "); parseGot != "new-jersey-hub" {
		parseT.Fatalf("sanitizeWarehouseReturnID(valid) = %q", parseGot)
	}
	for _, parseRaw := range []string{"", "warehouse/a", `warehouse\\a`} {
		if parseGot2 := sanitizeWarehouseReturnID(parseRaw); parseGot2 != "" {
			parseT.Fatalf("sanitizeWarehouseReturnID(%q) = %q, want empty", parseRaw, parseGot2)
		}
	}

	if parseGot3 := mustProductInt(" 42 ", 7); parseGot3 != 42 {
		parseT.Fatalf("mustProductInt(valid) = %d, want 42", parseGot3)
	}
	if parseGot4 := mustProductInt("nope", 7); parseGot4 != 7 {
		parseT.Fatalf("mustProductInt(invalid) = %d, want 7", parseGot4)
	}
}

func TestDecodeProductRequestInvalidBody(parseT *testing.T) {
	parseReq := jsonRequest(`{"sku":`)
	_, parseErr := decodeProductRequest(parseReq)
	if parseErr == nil {
		parseT.Fatal("expected decodeProductRequest to fail on invalid JSON")
	}
}

func formRequest(parseBody string) *http.Request {
	parseReq, _ := http.NewRequest(http.MethodPost, "/products", strings.NewReader(parseBody))
	parseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return parseReq
}

func jsonRequest(parseBody string) *http.Request {
	parseReq, _ := http.NewRequest(http.MethodPost, "/products", strings.NewReader(parseBody))
	parseReq.Header.Set("Content-Type", "application/json")
	return parseReq
}
