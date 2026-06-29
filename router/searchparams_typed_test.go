package router

import (
	"net/url"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/validate"
)

type listQuery struct {
	Page int      `query:"page" json:"page" validate:"gte=1"`
	Sort string   `query:"sort" json:"sort" validate:"omitempty,oneof=asc desc"`
	Live bool     `query:"live" json:"live"`
	Tags []string `query:"tag" json:"tag"`
}

// TestDecodeQueryTypedAndValidated proves url.Values decode into the typed struct with
// conversion (int, bool, []string) and pass validation for good input.
func TestDecodeQueryTypedAndValidated(parseT *testing.T) {
	parseValues := url.Values{
		"page": {"2"},
		"sort": {"desc"},
		"live": {"true"},
		"tag":  {"go", "wasm"},
	}
	parseQuery, parseErr := DecodeQuery[listQuery](parseValues)
	if parseErr != nil {
		parseT.Fatalf("expected valid decode, got %v", parseErr)
	}
	if parseQuery.Page != 2 || parseQuery.Sort != "desc" || !parseQuery.Live {
		parseT.Fatalf("unexpected decode: %+v", parseQuery)
	}
	if strings.Join(parseQuery.Tags, ",") != "go,wasm" {
		parseT.Fatalf("expected repeated tag params, got %v", parseQuery.Tags)
	}
}

// TestDecodeQueryMissingOptionalIsZero proves absent params leave zero values (and an
// optional field with omitempty still validates).
func TestDecodeQueryMissingOptionalIsZero(parseT *testing.T) {
	parseQuery, parseErr := DecodeQuery[listQuery](url.Values{"page": {"1"}})
	if parseErr != nil {
		parseT.Fatalf("expected valid decode, got %v", parseErr)
	}
	if parseQuery.Sort != "" || parseQuery.Live || len(parseQuery.Tags) != 0 {
		parseT.Fatalf("expected zero values for missing params, got %+v", parseQuery)
	}
}

// TestDecodeQueryValidationFailureReturnsResult proves a value that breaks a validate rule
// returns a validate.Result carrying the field error.
func TestDecodeQueryValidationFailureReturnsResult(parseT *testing.T) {
	_, parseErr := DecodeQuery[listQuery](url.Values{"page": {"0"}}) // gte=1
	if parseErr == nil {
		parseT.Fatal("expected a validation error for page=0")
	}
	parseResult, parseOk := parseErr.(validate.Result)
	if !parseOk {
		parseT.Fatalf("expected a validate.Result, got %T", parseErr)
	}
	if parseResult.Valid() {
		parseT.Fatal("result should be invalid")
	}
	if parseResult.Fields()["page"] == "" {
		parseT.Fatalf("expected a page field error, got %#v", parseResult.Fields())
	}
}

// TestDecodeQueryMalformedValueErrors proves a non-numeric int param is a conversion error
// naming the offending param.
func TestDecodeQueryMalformedValueErrors(parseT *testing.T) {
	_, parseErr := DecodeQuery[listQuery](url.Values{"page": {"abc"}})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "page") {
		parseT.Fatalf("expected a conversion error mentioning page, got %v", parseErr)
	}
}

// TestEncodeQueryRoundTrips proves EncodeQuery is the inverse of DecodeQuery, skipping
// zero-valued fields for clean URLs.
func TestEncodeQueryRoundTrips(parseT *testing.T) {
	parseOriginal := listQuery{Page: 3, Sort: "asc", Live: true, Tags: []string{"a", "b"}}
	parseValues := EncodeQuery(parseOriginal)

	if parseValues.Get("page") != "3" || parseValues.Get("sort") != "asc" || parseValues.Get("live") != "true" {
		parseT.Fatalf("unexpected encode: %v", parseValues)
	}
	if strings.Join(parseValues["tag"], ",") != "a,b" {
		parseT.Fatalf("expected repeated tag values, got %v", parseValues["tag"])
	}

	parseRoundTrip, parseErr := DecodeQuery[listQuery](parseValues)
	if parseErr != nil {
		parseT.Fatalf("round-trip decode failed: %v", parseErr)
	}
	if parseRoundTrip.Page != 3 || parseRoundTrip.Sort != "asc" || !parseRoundTrip.Live || strings.Join(parseRoundTrip.Tags, ",") != "a,b" {
		parseT.Fatalf("round-trip mismatch: %+v", parseRoundTrip)
	}

	// A zero-valued field is omitted entirely.
	parseSparse := EncodeQuery(listQuery{Page: 5})
	if parseSparse.Has("sort") || parseSparse.Has("live") {
		parseT.Fatalf("expected zero fields to be omitted, got %v", parseSparse)
	}
}
