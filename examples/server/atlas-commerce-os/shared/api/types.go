// Package api is Atlas's typed server-function boundary.
//
// # What this package is for
//
// Everything Atlas's browser half needs from its server half crosses here, as an
// ordinary Go function call. `gwc server gen` reads the //gwc:server directives in
// server_functions.go and writes two files from ONE signature:
//
//	serverfn_gen_server.go  registers the real function on the HTTP mux
//	serverfn_gen_client.go  a stub with the same signature that calls it
//
// So a component calls api.ListWarehouses(ctx, req) and gets a typed response,
// with no URL string, no fetch, no json.Unmarshal, and no place for the two sides
// to disagree. If the response type gains a field, both halves gain it together;
// if a caller passes the wrong type, it does not compile.
//
// That is worth stating against what it replaces. Atlas used to build request URLs
// by string concatenation in legacy_shared.go — "/api/public/warehouses?" +
// encoded, "/api/app/products?" + encoded — decode the reply into an any-shaped
// payload, and hope the server still returned what the caller expected. Nothing
// checked the two ends against each other; a renamed JSON field failed at runtime,
// in the browser, as an empty region of the page.
//
// # Why the types live in this file and not next to the functions
//
// server_functions.go carries `//go:build !js || !wasm` because it imports the
// database and must never enter the wasm binary. The generated client stub takes
// its place there. But BOTH halves need the request and response types, so the
// types sit here, unconstrained, and are compiled into both binaries.
//
// The codegen enforces a related rule: a server function's request and response
// types must be declared in THIS package. That is why these are Atlas-owned
// structs rather than aliases of server/db types — the wire contract is a thing
// Atlas owns and versions deliberately, not an accident of whatever shape the
// storage layer happens to have today. A column rename in db should be a mapping
// change here, not a silent change to what the browser receives.
package api

// Warehouse is the public projection of a fulfilment hub.
//
// Deliberately NOT server/db.Warehouse: this is what the browser is allowed to
// know. Operator-only fields (staffing, backlog, pressure) exist on the storage
// type and must not arrive here just because someone added them upstream.
type Warehouse struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
}

// ListWarehousesRequest has no fields yet.
//
// It exists anyway, rather than the function taking a bare struct{}, because a
// request type is where the first filter will go — and adding a field to an
// existing named type is a source-compatible change on both sides, whereas
// changing the parameter type is not.
type ListWarehousesRequest struct {
	// Region optionally narrows the result. Empty means every hub.
	Region string `json:"region,omitempty"`
}

// ListWarehousesResponse wraps the slice instead of returning []Warehouse.
//
// A bare slice is a dead end: the day this needs a total, a cursor, or a
// "generated at" stamp, every caller has to change. A struct absorbs that.
type ListWarehousesResponse struct {
	Warehouses []Warehouse `json:"warehouses"`
}
