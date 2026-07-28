//go:build !js || !wasm

package api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// restoreWarehouseSource swaps the package-level dependency for the duration of a
// test and puts it back afterwards. The seam is process-global by design (a server
// wires it once at startup), so a test that forgets to restore it silently changes
// the behaviour of every test that runs after it.
func restoreWarehouseSource(parseT *testing.T, parseSource WarehouseSource) {
	parseT.Helper()
	warehouseSourceMu.RLock()
	parsePrevious := warehouseSource
	warehouseSourceMu.RUnlock()
	SetWarehouseSource(parseSource)
	parseT.Cleanup(func() { SetWarehouseSource(parsePrevious) })
}

func TestListWarehousesFailsLoudlyWhenTheSourceIsUnset(parseT *testing.T) {
	restoreWarehouseSource(parseT, nil)

	_, parseErr := ListWarehouses(context.Background(), ListWarehousesRequest{})
	if !errors.Is(parseErr, ErrWarehouseSourceUnset) {
		parseT.Fatalf("want ErrWarehouseSourceUnset, got %v", parseErr)
	}
	// The point of the named error: a startup-ordering mistake must not present as
	// a successful call returning nothing, which is what a nil check that returned
	// an empty response would have produced. An operator seeing an empty hub list
	// cannot tell "no hubs" from "the server was never wired".
}

func TestListWarehousesReturnsEveryHubWhenNoRegionIsGiven(parseT *testing.T) {
	restoreWarehouseSource(parseT, func(context.Context) ([]Warehouse, error) {
		return []Warehouse{
			{ID: "nj", Slug: "new-jersey-hub", Name: "New Jersey Hub", Region: "East"},
			{ID: "nv", Slug: "nevada-hub", Name: "Nevada Hub", Region: "West"},
		}, nil
	})

	parseResponse, parseErr := ListWarehouses(context.Background(), ListWarehousesRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListWarehouses: %v", parseErr)
	}
	if len(parseResponse.Warehouses) != 2 {
		parseT.Fatalf("want 2 hubs, got %d", len(parseResponse.Warehouses))
	}
}

func TestListWarehousesFiltersByRegionCaseInsensitively(parseT *testing.T) {
	restoreWarehouseSource(parseT, func(context.Context) ([]Warehouse, error) {
		return []Warehouse{
			{ID: "nj", Region: "East"},
			{ID: "nv", Region: "West"},
			{ID: "il", Region: "Central"},
		}, nil
	})

	// Mixed case and surrounding space on purpose: a region reaches this from a
	// query string a human typed or a link someone hand-edited.
	parseResponse, parseErr := ListWarehouses(context.Background(), ListWarehousesRequest{Region: "  wEsT "})
	if parseErr != nil {
		parseT.Fatalf("ListWarehouses: %v", parseErr)
	}
	if len(parseResponse.Warehouses) != 1 || parseResponse.Warehouses[0].ID != "nv" {
		parseT.Fatalf("want only the West hub, got %+v", parseResponse.Warehouses)
	}
}

// A no-match filter and an empty database must both encode as [], never null.
//
// This asserts the JSON rather than the Go value because the difference only
// exists on the wire: a nil slice and an empty slice are both len 0 in Go, and the
// client that has to cope with `null` is on the other side of encoding/json.
func TestListWarehousesEncodesEmptyResultsAsAnArray(parseT *testing.T) {
	parseCases := map[string]WarehouseSource{
		"no rows at all": func(context.Context) ([]Warehouse, error) { return nil, nil },
		"filter matches none": func(context.Context) ([]Warehouse, error) {
			return []Warehouse{{ID: "nj", Region: "East"}}, nil
		},
	}
	parseRequests := map[string]ListWarehousesRequest{
		"no rows at all":      {},
		"filter matches none": {Region: "West"},
	}

	for parseName, parseSource := range parseCases {
		parseT.Run(parseName, func(parseSubT *testing.T) {
			restoreWarehouseSource(parseSubT, parseSource)
			parseResponse, parseErr := ListWarehouses(context.Background(), parseRequests[parseName])
			if parseErr != nil {
				parseSubT.Fatalf("ListWarehouses: %v", parseErr)
			}
			parseEncoded, parseEncodeErr := json.Marshal(parseResponse)
			if parseEncodeErr != nil {
				parseSubT.Fatalf("marshal: %v", parseEncodeErr)
			}
			if string(parseEncoded) != `{"warehouses":[]}` {
				parseSubT.Fatalf("empty results must encode as an array, got %s", parseEncoded)
			}
		})
	}
}

func TestListWarehousesPropagatesSourceErrors(parseT *testing.T) {
	parseBoom := errors.New("database unavailable")
	restoreWarehouseSource(parseT, func(context.Context) ([]Warehouse, error) {
		return nil, parseBoom
	})

	_, parseErr := ListWarehouses(context.Background(), ListWarehousesRequest{})
	if !errors.Is(parseErr, parseBoom) {
		parseT.Fatalf("want the source error to reach the caller, got %v", parseErr)
	}
	// serverfn maps this to a 5xx and shows the browser a generic message; the real
	// error goes to the server-side logger (serverfn.SetErrorLogger). What matters
	// here is that the function does not swallow it into an empty success.
}
