//go:build js && wasm
// +build js,wasm

package main

import (
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

const atlasOverlayRoot = "#atlas-overlay-root"
const atlasToastAtomID = "atlas-commerce-os.toast.atom"

type atlasToast struct {
	ID      string
	Title   string
	Message string
	Tone    string
}

type atlasToastCenter struct {
	Items state.Atom[[]atlasToast]
}

func useAtlasToastCenter() atlasToastCenter {
	return atlasToastCenter{Items: state.UseAtom(atlasToastAtomID, []atlasToast{})}
}

func (c atlasToastCenter) Push(title, message, tone string) {
	c.Items.Update(func(previous []atlasToast) []atlasToast {
		next := append([]atlasToast(nil), previous...)
		next = append(next, atlasToast{
			ID:      strings.ToLower(strings.ReplaceAll(title, " ", "-")) + "-toast",
			Title:   title,
			Message: message,
			Tone:    tone,
		})
		if len(next) > 3 {
			next = next[len(next)-3:]
		}
		return next
	})
}

func atlasToastViewport(center atlasToastCenter) ui.Node {
	items := center.Items.Get()
	if len(items) == 0 {
		return nil
	}
	items = append([]atlasToast(nil), items...)
	sort.SliceStable(items, func(left, right int) bool {
		return items[left].ID < items[right].ID
	})
	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes,
			html.Div(html.Props{ID: item.ID, Class: "rounded-[1.35rem] border border-white/10 bg-slate-950/94 p-4 text-slate-100 shadow-[0_18px_50px_rgba(15,23,42,0.35)]"},
				html.Div(html.Props{Class: "flex items-center justify-between gap-3"},
					html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(item.Title)),
					atlasBadge(strings.ToUpper(item.Tone), item.Tone),
				),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(item.Message)),
			),
		)
	}
	return ui.Portal(ui.PortalProps{
		Target: ui.PortalTarget{Selector: atlasOverlayRoot},
		Child:  html.Div(html.Props{ID: "atlas-toast-viewport", Class: "fixed bottom-6 right-6 z-[1200] grid w-full max-w-sm gap-3"}, nodes...),
	})
}

func warehouseLabel(value string) string {
	switch value {
	case "illinois-hub":
		return "Illinois Hub"
	case "nevada-hub":
		return "Nevada Hub"
	case "new-jersey-hub":
		return "New Jersey Hub"
	default:
		return strings.ReplaceAll(value, "-", " ")
	}
}

func atlasBadge(label, tone string) ui.Node {
	className := "atlas-badge"
	switch tone {
	case "warm":
		className += " atlas-badge-warm"
	case "success":
		className += " atlas-badge-success"
	case "danger":
		className += " atlas-badge-danger"
	default:
		className += " atlas-badge-neutral"
	}
	return html.Span(html.Props{Class: className}, html.Text(label))
}

func atlasMetricCard(label, value, note string, accent ui.Node) ui.Node {
	children := []ui.Node{
		html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
			accent,
		),
		html.P(html.Props{Class: "mt-4 text-3xl font-black text-white"}, html.Text(value)),
	}
	if strings.TrimSpace(note) != "" {
		children = append(children, html.P(html.Props{Class: "mt-3 atlas-panel-note"}, html.Text(note)))
	}
	return html.Div(html.Props{Class: "atlas-setting-card"}, children...)
}

func atlasEmptyStateCard(kicker, title, note string, accents ...ui.Node) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "atlas-kicker"}, html.Text(kicker)),
		html.P(html.Props{Class: "mt-3 text-lg font-semibold text-white"}, html.Text(title)),
		html.P(html.Props{Class: "mt-3 atlas-panel-note"}, html.Text(note)),
	}
	if len(accents) > 0 {
		children = append(children, html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-2"}, accents...))
	}
	return html.Div(html.Props{Class: "atlas-empty-state"}, children...)
}

func routeLabel(path string) string {
	if strings.HasPrefix(path, routeWarehouses+"/") && strings.Contains(path, "/availability/") {
		return "Warehouse Availability"
	}
	if strings.HasPrefix(path, routeWarehouses+"/") {
		return "Warehouse Detail"
	}
	if strings.HasPrefix(path, routeInventory+"/") {
		return "SKU Detail"
	}
	if strings.HasPrefix(path, routeWarehouseOps+"/") {
		return "Warehouse Detail"
	}
	if strings.HasPrefix(path, routeTransfers+"/") {
		return "Transfer Detail"
	}
	if strings.HasPrefix(path, routePurchaseOrders+"/") {
		return "Purchase Order Detail"
	}
	if strings.HasPrefix(path, routeReceiving+"/") {
		return "Receiving Session"
	}
	if strings.HasPrefix(path, routeCatalog+"/") {
		return "Product Detail"
	}
	switch path {
	case routeLanding:
		return "Storefront"
	case routeCatalog:
		return "Catalog"
	case routeProduct:
		return "Frame Desk"
	case routeWarehouses:
		return "Warehouses"
	case routeDashboard:
		return "Dashboard"
	case routeInventory:
		return "Inventory"
	case routeSKUDetail:
		return "SKU Detail"
	case routeWarehouseOps:
		return "Warehouses"
	case routeWarehouseDetail:
		return "Warehouse Detail"
	case routeTransfers:
		return "Transfers"
	case routeTransferDetail:
		return "Transfer Detail"
	case routePurchaseOrders:
		return "Purchase Orders"
	case routePurchaseOrderDetail:
		return "Purchase Order Detail"
	case routeReceiving:
		return "Receiving"
	case routeReceivingSessionDetail:
		return "Receiving Session"
	case routeComments:
		return "Comments"
	case routeSettings:
		return "Settings"
	default:
		return "Route"
	}
}

func shellSubtitle(path string) string {
	if strings.HasPrefix(path, routeWarehouses+"/") && strings.Contains(path, "/availability/") {
		return "Make one product promise feel specific to one warehouse and region."
	}
	if strings.HasPrefix(path, routeWarehouses+"/") {
		return "Keep regional promise speed, stocked highlights, and availability drill-ins together."
	}
	if strings.HasPrefix(path, routeInventory+"/") {
		return "Keep warehouse detail, thresholds, and activity in one working view."
	}
	if strings.HasPrefix(path, routeWarehouseOps+"/") {
		return "Keep staffing, backlog, and next action visible for one warehouse."
	}
	if strings.HasPrefix(path, routeTransfers+"/") {
		return "Keep approval state, lane context, and audit history in one route."
	}
	if strings.HasPrefix(path, routePurchaseOrders+"/") {
		return "Keep vendor context, approval state, and inbound rows in one route."
	}
	if strings.HasPrefix(path, routeReceiving+"/") {
		return "Keep classification, closeout readiness, and session context in one route."
	}
	if strings.HasPrefix(path, routeCatalog+"/") {
		return "Balance conversion copy, credible questions, and warehouse-backed availability."
	}
	switch path {
	case routeLanding:
		return "Lead with premium product discovery and clear regional trust signals."
	case routeCatalog:
		return "Keep browsing controls strong without crowding the merchandising story."
	case routeProduct:
		return "Balance conversion copy, credible questions, and warehouse-backed availability."
	case routeWarehouses:
		return "Make locality and fulfillment confidence feel specific instead of generic."
	case routeDashboard:
		return "Prioritize the next action, not decorative analytics."
	case routeInventory:
		return "Preserve resume context and fast operator scanning."
	case routeSKUDetail:
		return "Keep warehouse detail, thresholds, and activity in one working view."
	case routeWarehouseOps:
		return "Compare staffing pressure and backlog across hubs before drilling deeper."
	case routeWarehouseDetail:
		return "Keep staffing, backlog, and next action visible for one warehouse."
	case routeTransfers:
		return "Show where stock should move and why before approval."
	case routeTransferDetail:
		return "Keep approval state, lane context, and audit history in one route."
	case routePurchaseOrders:
		return "Keep vendor approvals and inbound rows visible before the receiving handoff."
	case routePurchaseOrderDetail:
		return "Keep vendor context, approval state, and inbound rows in one route."
	case routeReceiving:
		return "Resume inbound work cleanly and make discrepancies explicit."
	case routeReceivingSessionDetail:
		return "Keep classification, closeout readiness, and session context in one route."
	case routeComments:
		return "Map public trust back to moderation and response quality."
	case routeSettings:
		return "Keep preferences calm, legible, and persistent across route entry."
	default:
		return "Keep the route readable and useful while the example expands."
	}
}

func routeAlertSummary(path string) string {
	if strings.HasPrefix(path, routeWarehouses+"/") && strings.Contains(path, "/availability/") {
		return "availability detail ready"
	}
	if strings.HasPrefix(path, routeWarehouses+"/") {
		return "regional promise ready"
	}
	if strings.HasPrefix(path, routeInventory+"/") {
		return "threshold review ready"
	}
	if strings.HasPrefix(path, routeWarehouseOps+"/") {
		return "warehouse handoff ready"
	}
	if strings.HasPrefix(path, routeTransfers+"/") {
		return "approval ready"
	}
	if strings.HasPrefix(path, routePurchaseOrders+"/") {
		return "vendor review ready"
	}
	if strings.HasPrefix(path, routeReceiving+"/") {
		return "session review ready"
	}
	switch path {
	case routeDashboard:
		return "3 active queues"
	case routeInventory:
		return "2 low-stock clusters"
	case routeWarehouseOps:
		return "3 warehouse nodes"
	case routeTransfers:
		return "4 transfer suggestions"
	case routeTransferDetail:
		return "approval ready"
	case routePurchaseOrders:
		return "3 vendor reviews"
	case routePurchaseOrderDetail:
		return "vendor review ready"
	case routeReceiving:
		return "2 blocked receipts"
	case routeReceivingSessionDetail:
		return "session review ready"
	case routeComments:
		return "9 open reviews"
	case routeSettings:
		return "preferences synced"
	default:
		return "stable route"
	}
}
