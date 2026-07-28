//go:build !js || !wasm

// Server-only half of Atlas's typed API boundary.
//
// The build constraint is load-bearing and `gwc server gen` refuses to run
// without it: this file's functions are the REAL implementations, and the
// generated client stub declares functions with the same names. If both entered
// the wasm build they would collide, so the constraint keeps the implementations
// out of the browser binary and the stub takes their place there.
//
// It also means the database driver, the query layer and everything they pull in
// stay out of app.wasm — which is not a detail. The v5 plan's M5 size budget is
// the reason the two-artifact split exists at all; a server function that
// accidentally reached the client would drag its whole dependency graph with it.
package api

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// WarehouseSource is the shape Atlas's storage layer must satisfy to serve
// ListWarehouses.
//
// A server function is a plain top-level func — the codegen requires exactly
// (context.Context, Request) (Response, error) — so it cannot take a *db.Store
// parameter. Dependencies arrive through this seam instead, registered once at
// startup by the server's main.
//
// Declaring the dependency as a function type rather than importing server/db
// keeps the direction of the dependency right: api defines what it needs, and the
// server supplies it. api never imports db, so db can never import api back into
// a cycle, and a test can register a fake in one line.
type WarehouseSource func(parseCtx context.Context) ([]Warehouse, error)

var (
	warehouseSourceMu sync.RWMutex
	warehouseSource   WarehouseSource
)

// SetWarehouseSource installs the implementation behind ListWarehouses. The
// server calls this during startup, before it begins serving.
//
// Guarded by a mutex rather than assigned directly because tests replace it while
// other tests may be reading it, and a data race here is the kind that passes for
// months and then fails once in CI.
func SetWarehouseSource(parseSource WarehouseSource) {
	warehouseSourceMu.Lock()
	defer warehouseSourceMu.Unlock()
	warehouseSource = parseSource
}

// ErrWarehouseSourceUnset is returned when ListWarehouses is called before the
// server wired its dependency.
//
// A named error rather than a generic one because this failure has exactly one
// cause — a startup ordering mistake — and the message should say so instead of
// surfacing in the browser as an empty list, which is what a nil-check-and-return
// would have produced.
var ErrWarehouseSourceUnset = errors.New("api: warehouse source is not registered; call api.SetWarehouseSource during server startup")

// ListWarehouses returns the fulfilment hubs a buyer is allowed to see.
//
//gwc:server
func ListWarehouses(parseCtx context.Context, parseRequest ListWarehousesRequest) (ListWarehousesResponse, error) {
	warehouseSourceMu.RLock()
	parseSource := warehouseSource
	warehouseSourceMu.RUnlock()

	if parseSource == nil {
		return ListWarehousesResponse{}, ErrWarehouseSourceUnset
	}

	parseAll, parseErr := parseSource(parseCtx)
	if parseErr != nil {
		return ListWarehousesResponse{}, parseErr
	}

	parseRegion := strings.TrimSpace(parseRequest.Region)
	if parseRegion == "" {
		// Always a non-nil slice: encoding/json renders a nil slice as null and an
		// empty one as [], and a client that ranges over the decoded value should
		// not have to care which. "No hubs" is an empty list, not an absent field.
		if parseAll == nil {
			return ListWarehousesResponse{Warehouses: []Warehouse{}}, nil
		}
		return ListWarehousesResponse{Warehouses: parseAll}, nil
	}

	parseFiltered := make([]Warehouse, 0, len(parseAll))
	for _, parseWarehouse := range parseAll {
		if strings.EqualFold(strings.TrimSpace(parseWarehouse.Region), parseRegion) {
			parseFiltered = append(parseFiltered, parseWarehouse)
		}
	}
	return ListWarehousesResponse{Warehouses: parseFiltered}, nil
}
