//go:build js && wasm

package router

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestIntrospectRegisteredRoutesMethod verifies the Router method returns
// routes in sorted order and handles edge cases without panicking.
func TestIntrospectRegisteredRoutesMethod(parseT *testing.T) {
	parseT.Run("returns sorted routes for populated router", func(parseT2 *testing.T) {
		parseR := NewHashRouter()
		parseComp := func(parseProps Attrs) *Element {
			return runtime.Div(nil, runtime.Text("x"))
		}
		parseR.Register("/zebra", parseComp)
		parseR.Register("/alpha", parseComp)
		parseR.Register("/mango", parseComp)

		parseGot := parseR.RegisteredRoutes()
		parseWant := []string{"/alpha", "/mango", "/zebra"}
		if !reflect.DeepEqual(parseGot, parseWant) {
			parseT2.Fatalf("RegisteredRoutes() = %v, want %v", parseGot, parseWant)
		}
	})

	parseT.Run("returns non-nil empty slice for empty router", func(parseT2 *testing.T) {
		parseR := NewHashRouter()
		parseGot := parseR.RegisteredRoutes()
		if parseGot == nil {
			parseT2.Fatal("RegisteredRoutes() returned nil for empty router, want non-nil empty slice")
		}
		if len(parseGot) != 0 {
			parseT2.Fatalf("RegisteredRoutes() = %v, want empty slice", parseGot)
		}
	})

	parseT.Run("returns non-nil empty slice for nil receiver", func(parseT2 *testing.T) {
		var parseR *Router
		parseGot := parseR.RegisteredRoutes()
		if parseGot == nil {
			parseT2.Fatal("RegisteredRoutes() on nil *Router returned nil, want non-nil empty slice")
		}
		if len(parseGot) != 0 {
			parseT2.Fatalf("RegisteredRoutes() on nil *Router = %v, want empty slice", parseGot)
		}
	})
}

// TestIntrospectRegisteredRoutesPackageFunc verifies the package-level
// RegisteredRoutes delegates to the global router correctly.
func TestIntrospectRegisteredRoutesPackageFunc(parseT *testing.T) {
	// Save and restore global router so we don't pollute other tests.
	parsePrev := globalRouter
	defer func() { globalRouter = parsePrev }()

	parseComp := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("y"))
	}

	parseRouter := NewHashRouter()
	parseRouter.Register("/beta", parseComp)
	parseRouter.Register("/alpha", parseComp)
	globalRouter = parseRouter

	parseGot := RegisteredRoutes()
	parseWant := []string{"/alpha", "/beta"}
	if !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("RegisteredRoutes() = %v, want %v", parseGot, parseWant)
	}
}

// TestIntrospectRegisteredRoutesNilGlobal verifies the package-level function
// handles a nil global router without panicking.
func TestIntrospectRegisteredRoutesNilGlobal(parseT *testing.T) {
	parsePrev := globalRouter
	defer func() { globalRouter = parsePrev }()

	globalRouter = nil
	parseGot := RegisteredRoutes()
	if parseGot == nil {
		parseT.Fatal("RegisteredRoutes() with nil global router returned nil, want non-nil empty slice")
	}
	if len(parseGot) != 0 {
		parseT.Fatalf("RegisteredRoutes() with nil global router = %v, want empty slice", parseGot)
	}
}
