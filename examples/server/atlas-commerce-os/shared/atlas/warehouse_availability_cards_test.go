package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestPublicProductPromiseLanesCardRendersWarehouseCards verifies warehouse-specific promise cards and service-level fallback copy.
func TestPublicProductPromiseLanesCardRendersWarehouseCards(parseT *testing.T) {
	parseT.Parallel()

	parseProduct := sampleProductCards()[0]
	parseWarehouses := []warehouseCard{
		{Name: "New Jersey Hub", Slug: "new-jersey-hub", Region: "East", ServiceLevel: "next-day"},
		{Name: "Illinois Hub", Slug: "illinois-hub", Region: "Midwest", ServiceLevel: ""},
	}
	parseMarkup, parseErr := ui.RenderToString(publicProductPromiseLanesCard(parseProduct, parseWarehouses, false))
	if parseErr != nil {
		parseT.Fatalf("publicProductPromiseLanesCard render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Regional promise lanes") {
		parseT.Fatalf("expected promise-lane heading, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Open New Jersey Hub availability") || !strings.Contains(parseMarkup, "Open Illinois Hub availability") {
		parseT.Fatalf("expected warehouse-specific availability links, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Regional service posture") {
		parseT.Fatalf("expected service-level fallback copy, got %q", parseMarkup)
	}
}

// TestPublicProductPromiseLanesCardRendersRefreshAndEmptyState verifies lane refresh messaging and empty fallback rendering.
func TestPublicProductPromiseLanesCardRendersRefreshAndEmptyState(parseT *testing.T) {
	parseT.Parallel()

	parseMarkup, parseErr := ui.RenderToString(publicProductPromiseLanesCard(sampleProductCards()[0], nil, true))
	if parseErr != nil {
		parseT.Fatalf("publicProductPromiseLanesCard empty render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Refreshing the regional lane list in the background") {
		parseT.Fatalf("expected refresh copy, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Atlas is still resolving warehouse lanes for this product.") {
		parseT.Fatalf("expected empty fallback copy, got %q", parseMarkup)
	}
}

// TestPublicAvailabilityPromiseBandRendersWarehouseSpecificCopy verifies warehouse-specific promise card content and availability metric copy.
func TestPublicAvailabilityPromiseBandRendersWarehouseSpecificCopy(parseT *testing.T) {
	parseT.Parallel()

	parseAvailability := availabilityPage{
		Warehouse: warehouseCard{
			Name:          "Nevada Hub",
			Region:        "West",
			ServiceLevel:  "priority",
			Focus:         "West-coast launch support",
			PublicSummary: "This region can absorb launch-week demand.",
		},
		Product:   sampleProductCards()[0],
		Available: 3,
		Inbound:   5,
		Status:    "promise_risk",
	}
	parseMarkup, parseErr := ui.RenderToString(publicAvailabilityPromiseBand(parseAvailability))
	if parseErr != nil {
		parseT.Fatalf("publicAvailabilityPromiseBand render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "Warehouse-specific promise") {
		parseT.Fatalf("expected promise band heading, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "West-coast launch support") || !strings.Contains(parseMarkup, "This region can absorb launch-week demand.") {
		parseT.Fatalf("expected warehouse-specific focus and summary copy, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, availabilityStoryCopy(3, 5)) {
		parseT.Fatalf("expected availability story metric copy, got %q", parseMarkup)
	}
}

// TestPublicAvailabilitySupportPointsCoverAvailabilityStates verifies support-point copy across in-stock, inbound-only, and constrained states.
func TestPublicAvailabilitySupportPointsCoverAvailabilityStates(parseT *testing.T) {
	parseT.Parallel()

	parseInStock := publicAvailabilitySupportPoints(availabilityPage{
		Warehouse: warehouseCard{Region: "East", ServiceLevel: "next-day", Backlog: "Dock queue controlled"},
		Available: 4,
		Inbound:   0,
	})
	if !strings.Contains(strings.Join(parseInStock, " "), "Quote now if the project can move on this region's current stock posture.") {
		parseT.Fatalf("expected in-stock support point, got %#v", parseInStock)
	}

	parseInboundOnly := publicAvailabilitySupportPoints(availabilityPage{
		Warehouse: warehouseCard{Region: "Midwest", ServiceLevel: "standard"},
		Available: 0,
		Inbound:   6,
	})
	if !strings.Contains(strings.Join(parseInboundOnly, " "), "Use reserve capture to hold buyer intent against the inbound recovery window for this specific hub.") {
		parseT.Fatalf("expected inbound support point, got %#v", parseInboundOnly)
	}

	parseConstrained := publicAvailabilitySupportPoints(availabilityPage{
		Warehouse: warehouseCard{Region: "West", ServiceLevel: "priority"},
		Available: 0,
		Inbound:   0,
	})
	if !strings.Contains(strings.Join(parseConstrained, " "), "Use the support path to discuss substitutions or a different warehouse before promising timing.") {
		parseT.Fatalf("expected constrained support point, got %#v", parseConstrained)
	}
}
