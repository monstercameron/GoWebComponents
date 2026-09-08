package services

import (
	"context"
	"github.com/monstercameron/GoWebComponents/v5/desktop"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"testing"
)

// TestLegacyFileRouteDefaultsToDenied proves older bindings share the SDK host policy.
func TestLegacyFileRouteDefaultsToDenied(parseT *testing.T) {
	parseService := &CounterService{}
	if _, parseErr := parseService.OpenFile(context.Background()); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("legacy route: %v", parseErr)
	}
	for _, parseMethod := range parseService.GetCapabilities().Methods {
		if parseMethod == desktop.FileDialogMethod {
			parseT.Fatal("nil host advertised file support")
		}
	}
}
