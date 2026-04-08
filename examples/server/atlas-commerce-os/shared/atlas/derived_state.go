package atlas

import (
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/server/atlas-commerce-os/shared/repository"
)

type InventoryRollup struct {
	VisibleLanes     int
	TotalAvailable   int
	CriticalLanes    int
	PromiseRiskLanes int
	RiskLanes        int
	InboundUnits     int
	ReorderUnits     int
	ReorderLanes     int
	SKUCount         int
	SKUsWithInbound  int
}

func InventoryRollupFromRepositoryRows(parseRows []repository.InventoryRow) InventoryRollup {
	parseRollup := InventoryRollup{VisibleLanes: len(parseRows)}
	parseSeenSKUs := map[string]bool{}
	parseSkusWithInbound := map[string]bool{}
	for _, parseRow := range parseRows {
		parseRollup.TotalAvailable += parseRow.Available
		parseRollup.InboundUnits += parseRow.Inbound
		parseRollup.ReorderUnits += parseRow.ReorderUnits
		switch normalizedAtlasStatus(parseRow.Status) {
		case "critical":
			parseRollup.CriticalLanes++
			parseRollup.RiskLanes++
		case "promise_risk", "recovery":
			parseRollup.PromiseRiskLanes++
			parseRollup.RiskLanes++
		}
		if parseRow.ReorderUnits > 0 {
			parseRollup.ReorderLanes++
		}
		if !parseSeenSKUs[parseRow.SKU] {
			parseSeenSKUs[parseRow.SKU] = true
			parseRollup.SKUCount++
		}
		if parseRow.Inbound > 0 && !parseSkusWithInbound[parseRow.SKU] {
			parseSkusWithInbound[parseRow.SKU] = true
			parseRollup.SKUsWithInbound++
		}
	}
	return parseRollup
}

func inventoryRollupFromRows(parseRows []inventoryRow) InventoryRollup {
	parseRollup := InventoryRollup{VisibleLanes: len(parseRows)}
	parseSeenSKUs := map[string]bool{}
	parseSkusWithInbound := map[string]bool{}
	for _, parseRow := range parseRows {
		parseRollup.TotalAvailable += parseRow.Available
		parseRollup.InboundUnits += parseRow.Inbound
		parseRollup.ReorderUnits += parseRow.ReorderUnits
		switch normalizedAtlasStatus(parseRow.Status) {
		case "critical":
			parseRollup.CriticalLanes++
			parseRollup.RiskLanes++
		case "promise_risk", "recovery":
			parseRollup.PromiseRiskLanes++
			parseRollup.RiskLanes++
		}
		if parseRow.ReorderUnits > 0 {
			parseRollup.ReorderLanes++
		}
		if !parseSeenSKUs[parseRow.SKU] {
			parseSeenSKUs[parseRow.SKU] = true
			parseRollup.SKUCount++
		}
		if parseRow.Inbound > 0 && !parseSkusWithInbound[parseRow.SKU] {
			parseSkusWithInbound[parseRow.SKU] = true
			parseRollup.SKUsWithInbound++
		}
	}
	return parseRollup
}

func inventorySummaryCards(parseRows []inventoryRow) []inventorySummaryCard {
	parseGrouped := map[string]*inventorySummaryCard{}
	for _, parseRow := range parseRows {
		parseEntry, parseOk := parseGrouped[parseRow.SKU]
		if !parseOk {
			parseEntry = &inventorySummaryCard{
				SKU:         parseRow.SKU,
				Title:       parseRow.Title,
				Status:      parseRow.Status,
				PrimaryLane: fallback(parseRow.WarehouseName, parseRow.WarehouseID),
				LastUpdated: parseRow.UpdatedAt,
			}
			parseGrouped[parseRow.SKU] = parseEntry
		}
		parseEntry.Available += parseRow.Available
		parseEntry.Inbound += parseRow.Inbound
		parseEntry.LaneCount++
		if normalizedAtlasStatus(parseRow.Status) != "balanced" {
			parseEntry.RiskLaneCount++
			parseEntry.Status = parseRow.Status
		}
		if parseRow.UpdatedAt > parseEntry.LastUpdated {
			parseEntry.LastUpdated = parseRow.UpdatedAt
		}
	}
	parseItems := make([]inventorySummaryCard, 0, len(parseGrouped))
	for _, parseItem := range parseGrouped {
		parseItems = append(parseItems, *parseItem)
	}
	sort.Slice(parseItems, func(parseLeft, parseRight int) bool {
		if parseItems[parseLeft].Title == parseItems[parseRight].Title {
			return parseItems[parseLeft].SKU < parseItems[parseRight].SKU
		}
		return parseItems[parseLeft].Title < parseItems[parseRight].Title
	})
	return parseItems
}

func normalizedAtlasStatus(parseValue string) string {
	return strings.TrimSpace(strings.ToLower(parseValue))
}

func catalogPromiseCopy(parseStatus string) string {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return "Ready for active projects"
	case "low_stock", "pending", "submitted", "in_review":
		return "Best planned with support"
	default:
		return "Atlas-verified catalog entry"
	}
}

func catalogActionPlan(parseStatus string) (string, string) {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return "View quote-ready product", "Ready for pricing and project review"
	case "low_stock", "pending", "submitted", "in_review":
		return "View availability options", "Constrained stock, reserve the next realistic window"
	default:
		return "View alternatives", "Best used for notification or substitution planning"
	}
}

func catalogEditorialCopy(parseProduct productCard) string {
	if strings.TrimSpace(parseProduct.SEODescription) != "" {
		return parseProduct.SEODescription
	}
	return "Built to keep material quality, fulfillment posture, and commercial next steps readable in one route."
}

func productCategoryCue(parseCategory string) string {
	switch strings.TrimSpace(strings.ToLower(parseCategory)) {
	case "desks":
		return "Focused workstation layouts and planning conversations."
	case "storage":
		return "Organization layers that support daily workspace rhythm."
	case "lighting":
		return "Task-ready illumination that finishes a calmer setup."
	case "bundles":
		return "Coordinated system buying for faster project decisions."
	default:
		return "Workspace upgrades with a more considered systems view."
	}
}

func productSupportCue(parseStatus string) string {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return "Stock posture supports immediate quoting and fulfillment follow-through."
	case "low_stock", "pending", "submitted", "in_review":
		return "Inventory looks constrained, so recovery and quote paths matter more."
	default:
		return "Atlas keeps recovery paths visible when availability needs more coordination."
	}
}

func productBuyingMotion(parseStatus string) string {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return "Lead with a project quote, then use warehouse context to confirm delivery confidence."
	case "low_stock", "pending", "submitted", "in_review":
		return "Guide the buyer toward reserving the next available units instead of exposing raw replenishment language."
	default:
		return "Preserve buyer intent with a notification path and a clear alternative route when immediate supply is not realistic."
	}
}

func productSupportPlan(parseStatus string) (string, string, []string) {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return "Move from shortlist to quote.", "This SKU can support an active buying conversation now. Lead with pricing, timing, and regional delivery confidence.", []string{
			"Use the quote form as the primary action for real project intent.",
			"Keep delivery and fit questions available, but secondary.",
			"Use regional delivery details to confirm timing, not to learn Atlas internals.",
		}
	case "low_stock", "pending", "submitted", "in_review":
		return "Keep the project moving while supply is tight.", "This SKU still has demand value, but the UX should set realistic expectations and preserve buyer intent for the next available units.", []string{
			"Offer a reservation-style action instead of a raw restock request.",
			"Explain that inventory is constrained in plain buyer language.",
			"Keep a human support path nearby for timing or substitution questions.",
		}
	default:
		return "Stay in the loop without losing the product context.", "When immediate fulfillment is not realistic, the product page should shift from conversion to intent capture and alternative discovery.", []string{
			"Use a notification flow instead of implying immediate purchase readiness.",
			"Offer similar options so the buyer is not trapped at a dead end.",
			"Keep specialist support available for spec, finish, and delivery questions.",
		}
	}
}

func warehouseServiceTone(parseServiceLevel string) string {
	switch strings.TrimSpace(strings.ToLower(parseServiceLevel)) {
	case "next-day", "priority", "priority coverage":
		return "Fastest fit for tighter delivery windows and higher-priority installs."
	case "two-day", "standard-plus":
		return "Balanced timing for routine commercial installs and steady planning."
	default:
		return "Steady coverage for standard project scheduling and mixed-cart orders."
	}
}

func warehouseRegionCue(parseRegion string) string {
	switch strings.TrimSpace(strings.ToLower(parseRegion)) {
	case "west", "west coast", "western":
		return "Supports west-coast schedules and shorter transit expectations for nearby teams."
	case "midwest", "central":
		return "Acts as a stabilizing central lane for broader multi-region coverage."
	case "east", "east coast", "eastern":
		return "Helps protect east-coast promise windows and denser delivery expectations."
	default:
		return "Service territory and stock depth stay readable here before buyers open a product-specific availability view."
	}
}

func availabilityStoryCopy(parseAvailable int, parseInbound int) string {
	// Availability messaging intentionally prioritizes current shippable units over inbound promises so
	// public copy stays realistic about what buyers can act on immediately.
	if parseAvailable > 0 && parseInbound > 0 {
		return "Current stock covers near-term demand while inbound units support the next replenishment wave."
	}
	if parseAvailable > 0 {
		return "This hub can support immediate demand from on-hand inventory without relying on inbound receipts."
	}
	if parseInbound > 0 {
		return "Stock is constrained now, but replenishment is already moving into the lane for recovery planning."
	}
	return "Inventory is currently constrained, so demand recovery and warehouse-specific follow-up are the right next steps."
}

func availabilitySupportPlan(parseAvailable int, parseInbound int) (string, string) {
	// Decision rule: immediate stock wins, inbound becomes reserve guidance, and zero-stock falls back to
	// intent capture plus alternatives instead of implying a hard purchase path.
	if parseAvailable > 0 {
		return "This region can support the project now.", "Use this route to confirm regional promise, then move directly into quote capture while the delivery context is still fresh."
	}
	if parseInbound > 0 {
		return "Reserve the next inbound wave.", "This region is constrained today, but inbound units are already moving. Preserve buyer intent against this specific hub instead of sending them back to a generic form."
	}
	return "Stay attached to this region.", "Immediate fulfillment is not realistic here, so the right UX is a notification path plus a support channel for alternative planning."
}
