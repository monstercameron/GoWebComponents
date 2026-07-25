//go:build !js || !wasm

package router

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/pluginruntime"
)

func TestNativeRegisteredRoutesReturnsNonNilEmptySlice(parseT *testing.T) {
	parseRoutes := RegisteredRoutes()
	if parseRoutes == nil {
		parseT.Fatal("RegisteredRoutes returned nil")
	}
	if len(parseRoutes) != 0 {
		parseT.Fatalf("RegisteredRoutes = %v, want empty", parseRoutes)
	}
}

func TestNativeRouteServiceSnapshotAndCommandErrors(parseT *testing.T) {
	parseService := BuildRouteService()
	parseSnapshot, parseErr := parseService.GetRouteSnapshot(pluginruntime.QueryBudget{MaxItems: 1})
	if parseErr != nil {
		parseT.Fatalf("GetRouteSnapshot returned error: %v", parseErr)
	}
	if parseSnapshot.Meta != pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDNative, false) {
		parseT.Fatalf("unexpected route snapshot meta: %#v", parseSnapshot.Meta)
	}
	parseChecks := []struct {
		name string
		err  error
		want string
	}{
		{name: "navigate", err: parseService.NavigateRoute("/next", pluginruntime.CommandOptions{}), want: "navigate is unavailable"},
		{name: "revalidate", err: parseService.RevalidateRoute(pluginruntime.CommandOptions{}), want: "revalidate is unavailable"},
		{name: "retry", err: parseService.RetryRouteLoader("loader", pluginruntime.CommandOptions{}), want: "loader retry is unavailable"},
	}
	for _, parseCheck := range parseChecks {
		if parseCheck.err == nil || !strings.Contains(parseCheck.err.Error(), parseCheck.want) {
			parseT.Fatalf("%s error = %v, want %q", parseCheck.name, parseCheck.err, parseCheck.want)
		}
	}
}
