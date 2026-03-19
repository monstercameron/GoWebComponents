package atlas

import (
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
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

func InventoryRollupFromRepositoryRows(rows []repository.InventoryRow) InventoryRollup {
	rollup := InventoryRollup{VisibleLanes: len(rows)}
	seenSKUs := map[string]bool{}
	skusWithInbound := map[string]bool{}
	for _, row := range rows {
		rollup.TotalAvailable += row.Available
		rollup.InboundUnits += row.Inbound
		rollup.ReorderUnits += row.ReorderUnits
		switch normalizedAtlasStatus(row.Status) {
		case "critical":
			rollup.CriticalLanes++
			rollup.RiskLanes++
		case "promise_risk", "recovery":
			rollup.PromiseRiskLanes++
			rollup.RiskLanes++
		}
		if row.ReorderUnits > 0 {
			rollup.ReorderLanes++
		}
		if !seenSKUs[row.SKU] {
			seenSKUs[row.SKU] = true
			rollup.SKUCount++
		}
		if row.Inbound > 0 && !skusWithInbound[row.SKU] {
			skusWithInbound[row.SKU] = true
			rollup.SKUsWithInbound++
		}
	}
	return rollup
}

func inventoryRollupFromRows(rows []inventoryRow) InventoryRollup {
	rollup := InventoryRollup{VisibleLanes: len(rows)}
	seenSKUs := map[string]bool{}
	skusWithInbound := map[string]bool{}
	for _, row := range rows {
		rollup.TotalAvailable += row.Available
		rollup.InboundUnits += row.Inbound
		rollup.ReorderUnits += row.ReorderUnits
		switch normalizedAtlasStatus(row.Status) {
		case "critical":
			rollup.CriticalLanes++
			rollup.RiskLanes++
		case "promise_risk", "recovery":
			rollup.PromiseRiskLanes++
			rollup.RiskLanes++
		}
		if row.ReorderUnits > 0 {
			rollup.ReorderLanes++
		}
		if !seenSKUs[row.SKU] {
			seenSKUs[row.SKU] = true
			rollup.SKUCount++
		}
		if row.Inbound > 0 && !skusWithInbound[row.SKU] {
			skusWithInbound[row.SKU] = true
			rollup.SKUsWithInbound++
		}
	}
	return rollup
}

func inventorySummaryCards(rows []inventoryRow) []inventorySummaryCard {
	grouped := map[string]*inventorySummaryCard{}
	for _, row := range rows {
		entry, ok := grouped[row.SKU]
		if !ok {
			entry = &inventorySummaryCard{
				SKU:         row.SKU,
				Title:       row.Title,
				Status:      row.Status,
				PrimaryLane: fallback(row.WarehouseName, row.WarehouseID),
				LastUpdated: row.UpdatedAt,
			}
			grouped[row.SKU] = entry
		}
		entry.Available += row.Available
		entry.Inbound += row.Inbound
		entry.LaneCount++
		if normalizedAtlasStatus(row.Status) != "balanced" {
			entry.RiskLaneCount++
			entry.Status = row.Status
		}
		if row.UpdatedAt > entry.LastUpdated {
			entry.LastUpdated = row.UpdatedAt
		}
	}
	items := make([]inventorySummaryCard, 0, len(grouped))
	for _, item := range grouped {
		items = append(items, *item)
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].Title == items[right].Title {
			return items[left].SKU < items[right].SKU
		}
		return items[left].Title < items[right].Title
	})
	return items
}

func normalizedAtlasStatus(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func catalogPromiseCopy(status string) string {
	switch normalizedAtlasStatus(status) {
	case "in_stock", "healthy", "approved", "available":
		return "Ready for active projects"
	case "low_stock", "pending", "submitted", "in_review":
		return "Best planned with support"
	default:
		return "Atlas-verified catalog entry"
	}
}

func catalogActionPlan(status string) (string, string) {
	switch normalizedAtlasStatus(status) {
	case "in_stock", "healthy", "approved", "available":
		return "View quote-ready product", "Ready for pricing and project review"
	case "low_stock", "pending", "submitted", "in_review":
		return "View availability options", "Constrained stock, reserve the next realistic window"
	default:
		return "View alternatives", "Best used for notification or substitution planning"
	}
}

func catalogEditorialCopy(product productCard) string {
	if strings.TrimSpace(product.SEODescription) != "" {
		return product.SEODescription
	}
	return "Built to keep material quality, fulfillment posture, and commercial next steps readable in one route."
}

func productCategoryCue(category string) string {
	switch strings.TrimSpace(strings.ToLower(category)) {
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

func productSupportCue(status string) string {
	switch normalizedAtlasStatus(status) {
	case "in_stock", "healthy", "approved", "available":
		return "Stock posture supports immediate quoting and fulfillment follow-through."
	case "low_stock", "pending", "submitted", "in_review":
		return "Inventory looks constrained, so recovery and quote paths matter more."
	default:
		return "Atlas keeps recovery paths visible when availability needs more coordination."
	}
}

func productBuyingMotion(status string) string {
	switch normalizedAtlasStatus(status) {
	case "in_stock", "healthy", "approved", "available":
		return "Lead with a project quote, then use warehouse context to confirm delivery confidence."
	case "low_stock", "pending", "submitted", "in_review":
		return "Guide the buyer toward reserving the next available units instead of exposing raw replenishment language."
	default:
		return "Preserve buyer intent with a notification path and a clear alternative route when immediate supply is not realistic."
	}
}

func productSupportPlan(status string) (string, string, []string) {
	switch normalizedAtlasStatus(status) {
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

func warehouseServiceTone(serviceLevel string) string {
	switch strings.TrimSpace(strings.ToLower(serviceLevel)) {
	case "next-day", "priority", "priority coverage":
		return "Fastest fit for tighter delivery windows and higher-priority installs."
	case "two-day", "standard-plus":
		return "Balanced timing for routine commercial installs and steady planning."
	default:
		return "Steady coverage for standard project scheduling and mixed-cart orders."
	}
}

func warehouseRegionCue(region string) string {
	switch strings.TrimSpace(strings.ToLower(region)) {
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

func availabilityStoryCopy(available int, inbound int) string {
	if available > 0 && inbound > 0 {
		return "Current stock covers near-term demand while inbound units support the next replenishment wave."
	}
	if available > 0 {
		return "This hub can support immediate demand from on-hand inventory without relying on inbound receipts."
	}
	if inbound > 0 {
		return "Stock is constrained now, but replenishment is already moving into the lane for recovery planning."
	}
	return "Inventory is currently constrained, so demand recovery and warehouse-specific follow-up are the right next steps."
}

func availabilitySupportPlan(available int, inbound int) (string, string) {
	if available > 0 {
		return "This region can support the project now.", "Use this route to confirm regional promise, then move directly into quote capture while the delivery context is still fresh."
	}
	if inbound > 0 {
		return "Reserve the next inbound wave.", "This region is constrained today, but inbound units are already moving. Preserve buyer intent against this specific hub instead of sending them back to a generic form."
	}
	return "Stay attached to this region.", "Immediate fulfillment is not realistic here, so the right UX is a notification path plus a support channel for alternative planning."
}
