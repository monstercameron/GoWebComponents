package seed

type ScreenData struct {
	Title   string
	Feature string
	Summary string
	Bullets []string
	Stats   [][2]string
}

var Landing = ScreenData{
	Title:   "Atlas storefront landing",
	Feature: "Public shell kickoff",
	Summary: "The landing route establishes the editorial storefront surface with warehouse-backed reliability, category-led discovery, and a premium product story.",
	Bullets: []string{
		"Lead with product-system confidence rather than generic ecommerce chrome.",
		"Use warehouse-backed proof early so regional reliability feels core to the brand.",
		"Keep quote capture present but secondary to product discovery.",
	},
	Stats: [][2]string{{"Featured products", "24"}, {"Warehouse regions", "3"}, {"Primary calls to action", "3"}},
}

var Catalog = ScreenData{
	Title:   "Atlas catalog",
	Feature: "Filterable public product browsing",
	Summary: "The catalog route should own search, sorting, filter chips, and deep-linkable browsing without losing the premium merchandising posture.",
	Bullets: []string{
		"Results header, sort, and filters should feel first-class and obvious.",
		"Constrained-stock states should push users toward quote and restock paths calmly.",
		"The layout should support quick scanability on large screens and clean collapse on smaller ones.",
	},
	Stats: [][2]string{{"Core categories", "6"}, {"Default sort", "Featured"}, {"Filter model", "URL-backed"}},
}

var Product = ScreenData{
	Title:   "Atlas Frame Desk",
	Feature: "Warehouse-aware product detail",
	Summary: "The product route should combine premium editorial layout, exact specifications, and a warehouse-aware availability story in the first screenful.",
	Bullets: []string{
		"Show availability by region before the user has to hunt through the page.",
		"Low-stock states should pivot into alternatives, quote capture, or restock support.",
		"Comments and product questions should feel integrated rather than bolted on.",
	},
	Stats: [][2]string{{"Availability status", "Low stock"}, {"Nearest warehouse", "Illinois"}, {"Next best action", "Request quote"}},
}

var Warehouses = ScreenData{
	Title:   "Warehouse network",
	Feature: "Public regional fulfillment overview",
	Summary: "Warehouse routes should make locality feel operationally real without turning into logistics dashboards.",
	Bullets: []string{
		"Lead with service region and promise speed.",
		"Use stocked highlights to connect place to product confidence.",
		"Keep operational notices only when they help customer decisions.",
	},
	Stats: [][2]string{{"Public warehouses", "3"}, {"Fast-turn region", "New Jersey"}, {"Balancing hub", "Illinois"}},
}

var Dashboard = ScreenData{
	Title:   "Operations dashboard",
	Feature: "Internal triage shell",
	Summary: "The dashboard is a triage surface, not a decorative KPI wall. It should route users quickly into low-stock, receiving, transfer, and moderation work.",
	Bullets: []string{
		"Urgent alerts should outrank vanity summaries.",
		"Quick actions should push users into inventory, transfers, and receiving.",
		"Moderation should remain visible because the public surface depends on it.",
	},
	Stats: [][2]string{{"Urgent alerts", "7"}, {"Inbound sessions", "3"}, {"Pending reviews", "5"}},
}

var Inventory = ScreenData{
	Title:   "Inventory workspace",
	Feature: "Dense internal list management",
	Summary: "Inventory should be the strongest operational surface in the example: sortable, filterable, saved, and fast enough to feel trustworthy.",
	Bullets: []string{
		"Status, cover days, and availability should scan cleanly row by row.",
		"Saved views should make repeated operational contexts easy to restore.",
		"Threshold edits and transfer actions should preserve list context through overlays or side sheets.",
	},
	Stats: [][2]string{{"Visible columns", "8"}, {"Saved views", "4"}, {"Low-stock SKUs", "13"}},
}

var SKUDetail = ScreenData{
	Title:   "SKU detail - Atlas Frame Desk",
	Feature: "Inventory detail route",
	Summary: "The SKU detail route should consolidate warehouse breakdown, threshold context, activity, and operational notes without losing the surrounding inventory workflow.",
	Bullets: []string{
		"Keep the warehouse breakdown near the top of the page.",
		"Put threshold and transfer actions in the first action cluster.",
		"Treat notes and activity as supporting context, not the primary visual weight.",
	},
	Stats: [][2]string{{"Warehouses", "3"}, {"Reorder point", "18"}, {"Open notes", "2"}},
}

var Transfers = ScreenData{
	Title:   "Transfer planning",
	Feature: "Cross-warehouse rebalance workflow",
	Summary: "Transfers should explain why a move is recommended, what inventory is affected, and how the operator can approve with confidence.",
	Bullets: []string{
		"Show source and destination warehouses clearly.",
		"Explain the imbalance that triggered the recommendation.",
		"Keep draft, submitted, and approved states visually distinct.",
	},
	Stats: [][2]string{{"Recommendations", "4"}, {"In transit", "1"}, {"Cancelled", "1"}},
}

var Receiving = ScreenData{
	Title:   "Receiving queue",
	Feature: "Inbound reconciliation workflow",
	Summary: "Receiving should let operators resume sessions, inspect mismatches, and close inbound work with precise discrepancy handling.",
	Bullets: []string{
		"Separate clean receipts from discrepancy-heavy sessions.",
		"Make short shipment, overage, and damage paths explicit.",
		"Keep final closeout summary concise and auditable.",
	},
	Stats: [][2]string{{"Open sessions", "3"}, {"With discrepancies", "2"}, {"Ready to close", "1"}},
}

var Comments = ScreenData{
	Title:   "Moderation queue",
	Feature: "Public comment review flow",
	Summary: "The moderation screen should balance queue efficiency with enough detail to make approve, reject, and flag actions feel defensible.",
	Bullets: []string{
		"Use a split queue-and-detail layout by default.",
		"Keep status filters simple and immediately useful.",
		"Make the public visibility consequence obvious before the reviewer acts.",
	},
	Stats: [][2]string{{"Pending items", "12"}, {"Flagged items", "3"}, {"Primary actions", "3"}},
}

var Settings = ScreenData{
	Title:   "Settings and preferences",
	Feature: "Theme, locale, density, and workspace defaults",
	Summary: "Settings should group preferences by user intent so appearance, language, workspace defaults, and diagnostics stay legible and demo-friendly.",
	Bullets: []string{
		"Theme and density belong together under appearance.",
		"Locale and route strategy belong together under language.",
		"Developer-facing toggles should stay contained in diagnostics rather than leaking into the main UX.",
	},
	Stats: [][2]string{{"Theme modes", "2"}, {"Supported locales", "3"}, {"Preference groups", "4"}},
}

var InventoryColumns = []string{"SKU", "Title", "Warehouse", "Available", "Cover days", "Inbound", "Status", "Updated"}

var InventoryFilters = []string{"warehouse", "category", "supplier", "stock-health", "availability", "search"}

var SavedViews = []string{"Low stock triage", "East coast shortages", "Desk replenishment", "Accessory watchlist"}
