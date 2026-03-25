package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestDecodeProductRequestFromFormAndJSON(t *testing.T) {
	t.Run("form", func(t *testing.T) {
		form := url.Values{
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
		req := formRequest(form.Encode())

		got, err := decodeProductRequest(req)
		if err != nil {
			t.Fatalf("decodeProductRequest(form): %v", err)
		}
		if got.SKU != " task-lamp " || got.Slug != " task-lamp " || got.PriceCents != 12999 || got.Available != 8 || got.Inbound != 3 {
			t.Fatalf("unexpected form payload: %+v", got)
		}
		if got.CurrentWarehouse != " illinois-hub " || got.ReturnWarehouseID != " new-jersey-hub " {
			t.Fatalf("unexpected warehouse payload: %+v", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		req := jsonRequest(`{
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

		got, err := decodeProductRequest(req)
		if err != nil {
			t.Fatalf("decodeProductRequest(json): %v", err)
		}
		if got.SKU != "frame-desk" || got.Title != "Frame Desk" || got.PriceCents != 249900 {
			t.Fatalf("unexpected json payload: %+v", got)
		}
		if got.WarehouseID != "new-jersey-hub" || got.CurrentWarehouse != "illinois-hub" || got.ReturnWarehouseID != "nevada-hub" {
			t.Fatalf("unexpected warehouse payload: %+v", got)
		}
	})
}

func TestProductHelpersValidationAndSanitization(t *testing.T) {
	fields := validateProductRequest(productRequest{}, true)
	for _, key := range []string{"sku", "slug", "title", "category", "status", "summary", "warehouse_id"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("expected validation error for %q, got %+v", key, fields)
		}
	}

	updateFields := validateProductRequest(productRequest{
		Title:       "Task Lamp",
		Category:    "lighting",
		PriceCents:  -1,
		Status:      "available",
		Summary:     "summary",
		WarehouseID: "warehouse-a",
		Available:   -2,
		Inbound:     -3,
	}, false)
	if _, ok := updateFields["sku"]; ok {
		t.Fatalf("did not expect sku to be required for update: %+v", updateFields)
	}
	for _, key := range []string{"price_cents", "available", "inbound"} {
		if _, ok := updateFields[key]; !ok {
			t.Fatalf("expected numeric validation error for %q, got %+v", key, updateFields)
		}
	}

	if got := sanitizeWarehouseReturnID(" new-jersey-hub "); got != "new-jersey-hub" {
		t.Fatalf("sanitizeWarehouseReturnID(valid) = %q", got)
	}
	for _, raw := range []string{"", "warehouse/a", `warehouse\\a`} {
		if got := sanitizeWarehouseReturnID(raw); got != "" {
			t.Fatalf("sanitizeWarehouseReturnID(%q) = %q, want empty", raw, got)
		}
	}

	if got := mustProductInt(" 42 ", 7); got != 42 {
		t.Fatalf("mustProductInt(valid) = %d, want 42", got)
	}
	if got := mustProductInt("nope", 7); got != 7 {
		t.Fatalf("mustProductInt(invalid) = %d, want 7", got)
	}
}

func TestDecodeProductRequestInvalidBody(t *testing.T) {
	req := jsonRequest(`{"sku":`)
	_, err := decodeProductRequest(req)
	if err == nil {
		t.Fatal("expected decodeProductRequest to fail on invalid JSON")
	}
}

func formRequest(body string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func jsonRequest(body string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
