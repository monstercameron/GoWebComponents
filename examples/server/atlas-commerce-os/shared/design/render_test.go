//go:build !(js && wasm)

package design_test

// An end-to-end SSR test: build one page out of the entire primitive set, render it
// with ui.RenderToString, and assert on the document that comes out.
//
// The per-primitive tests in design_test.go prove each bundle emits what it claims.
// This one proves the set COMPOSES — that a rail, a placard, a surface, a form and a
// table can coexist in one document without a primitive stomping another, and that the
// emitted stylesheet plus the emitted markup actually agree on class names.
//
// It also doubles as the eyeball harness. Set ATLAS_DESIGN_DUMP to a directory and it
// writes a standalone, self-contained HTML file there (SSR markup + the harvested
// <style> block, no external references) that you can open in a browser:
//
//	ATLAS_DESIGN_DUMP=/tmp go test ./examples/server/atlas-commerce-os/shared/design/ -run TestFullPageRenders
//
// Concurrency note: native SSR is single-flight per process —
// internal/runtime/reconciler.go keeps currentFiber in package globals, so two
// concurrent RenderToString calls trip GWC-RUNTIME-HOOK-THREADING. Renders in this
// package are serialized and nothing here calls t.Parallel().

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestFullPageRenders(t *testing.T) {
	reset(t)
	design.Install()

	// The probe renders BOTH shells in one document, because the design system has two
	// audiences and the review surface has to show both. The console half proves the
	// operator frame (rail, placard, queue table, form); the storefront half proves the
	// catalog manifest, which is the surface that had no answer at all.
	parseHTML := renderOnce(t, html.Fragment(consolePage(), storefrontPage()))
	parseCSS := css.Harvest()

	if parseDir := os.Getenv("ATLAS_DESIGN_DUMP"); parseDir != "" {
		parsePath := filepath.Join(parseDir, "atlas-design-probe.html")
		parseDoc := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
			`<meta name="viewport" content="width=device-width,initial-scale=1">` +
			`<title>Atlas Dock Manifest</title>` + css.StyleBlock() +
			`</head><body>` + parseHTML + `</body></html>`
		if parseErr := os.WriteFile(parsePath, []byte(parseDoc), 0o644); parseErr != nil {
			t.Fatalf("dump: %v", parseErr)
		}
		t.Logf("wrote %s (%d bytes html, %d bytes css)", parsePath, len(parseHTML), len(parseCSS))
	}

	// Semantic markup: the design system styles elements, it does not replace them.
	// A rail is an <aside>, a queue is a <table>, a title is an <h1>.
	for _, parseWant := range []string{
		"<aside class=", "<main class=", "<h1 class=", "<table class=",
		"<thead>", "<tbody>", "<th ", "<td ", "<label ", "<input ", "<button ",
	} {
		if !strings.Contains(parseHTML, parseWant) {
			t.Errorf("rendered page is missing %q — the primitives are not on semantic elements", parseWant)
		}
	}

	// Every class the markup references must exist in the emitted stylesheet. This is
	// the check that catches a primitive whose bundle folded to an empty class (an
	// empty rule-set folds to "", which would silently produce class="" in the markup).
	for _, parseClass := range extractClasses(parseHTML) {
		if parseClass == "" {
			t.Error("markup carries an empty class attribute: some bundle folded to nothing")
			continue
		}
		if !strings.Contains(parseCSS, "."+parseClass) {
			t.Errorf("markup references class %q with no matching rule in the stylesheet", parseClass)
		}
	}

	// And every emitted hashed class must be referenced by the markup — dead CSS in a
	// runtime-injected system means a bundle was folded and then not used, which in a
	// render loop is wasted injection on every client.
	for _, parseClass := range css.HarvestedClasses() {
		if !strings.HasPrefix(parseClass, "c-") {
			continue // global/:root/layer entries carry a different key prefix
		}
		if !strings.Contains(parseHTML, parseClass) {
			t.Errorf("emitted class %q is never used by the markup", parseClass)
		}
	}

	// The signature element survived composition inside a Surface-bearing page.
	if !strings.Contains(parseHTML, `role="group"`) || !strings.Contains(parseHTML, "aria-label=\"Lane NJ-HUB") {
		t.Error("the lane placard did not render inside the full page")
	}

	// The scroll container is reachable by keyboard. An overflow region with no tab stop
	// makes its overflowing content unreachable without a mouse.
	// Matched case-insensitively on purpose: the SSR serializer emits the framework's
	// camelCase Raw key verbatim as tabIndex="0". HTML attribute names are ASCII
	// case-insensitive so browsers read it correctly, but the assertion should not
	// depend on the casing.
	if !strings.Contains(strings.ToLower(parseHTML), `tabindex="0"`) || !strings.Contains(parseHTML, `role="region"`) {
		t.Error("the table scroll container is not keyboard reachable")
	}
}

// TestRepeatedFoldsEmitOnce proves the dedup that makes this approach viable at all:
// rendering three table rows that each fold the same cell bundles must not grow the
// stylesheet. If it did, a 200-row queue would inject 200 copies of every cell rule.
func TestRepeatedFoldsEmitOnce(t *testing.T) {
	reset(t)
	design.Install()
	renderOnce(t, html.Fragment(consolePage(), storefrontPage()))
	parseOne := len(css.HarvestedClasses())

	// Render the same page again into the same registry.
	renderOnce(t, html.Fragment(consolePage(), storefrontPage()))
	if parseTwo := len(css.HarvestedClasses()); parseTwo != parseOne {
		t.Errorf("re-rendering grew the registry from %d to %d entries; dedup is broken", parseOne, parseTwo)
	}
}

// --- the page -----------------------------------------------------------------

func consolePage() ui.Node {
	return html.Div(html.Props{Class: design.Class(design.ConsoleShell())},
		// The rail is composed as a placard column: a plate at the head, then groups of
		// items each carrying a mono route code at the trailing edge. The codes are the
		// mark that makes the frame Atlas's rather than every app's — see shell.go.
		html.Aside(html.Props{Class: design.Class(design.ConsoleRail())},
			design.RailPlateBlock("Operations console", "IL-HUB"),
			html.Div(html.Props{Class: design.Class(design.RailGroupLabel())}, html.Text("Inbound")),
			railItem("/internal/receiving", "Receiving", "RCV", true),
			railItem("/internal/purchase-orders", "Purchase orders", "PO", false),
			html.Div(html.Props{Class: design.Class(design.RailGroupLabel())}, html.Text("Inventory")),
			railItem("/internal/inventory", "On hand", "INV", false),
			railItem("/internal/transfers", "Transfers", "XFER", false),
		),
		html.Main(html.Props{Class: design.Class(design.ContentColumn())},
			html.Div(html.Props{Class: design.Class(design.PageHead())},
				html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Receiving / RCV-ILLINOIS-001")),
				html.H1(html.Props{Class: design.Class(design.PageTitle())}, html.Text("Inbound discrepancies")),
			),

			design.LanePlacard(design.PlacardSpec{
				OriginHub: "NJ-HUB",
				DestHub:   "IL-HUB",
				Promise:   "2026-08-04",
				LaneID:    "LN-4471",
				Posture:   design.PostureShort,
			}),

			html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))},
				html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Summary")),
				html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())},
					html.Text("Four of the eleven lines on this receipt came up short. Reconcile each line against the carrier manifest before closing the receipt."),
				),
				html.Div(html.Props{Class: design.Class(design.Divider())}),
				html.Div(html.Props{Class: design.Class(design.Cluster(design.Space3))},
					html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneException))}, html.Text("SHORT 4")),
					html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneVerified))}, html.Text("CLOSED CLEAN 6")),
					html.Span(html.Props{Class: design.Class(design.StatusChip(design.TonePending))}, html.Text("IN TRANSIT 1")),
					html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneNeutral))}, html.Text("DRAFT")),
				),
				html.Div(html.Props{Class: design.Class(design.ManifestRule())}),
				html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
					html.Button(html.Props{Type: "button", Class: design.Class(design.ButtonPrimary())}, html.Text("Close receipt")),
					html.Button(html.Props{Type: "button", Class: design.Class(design.ButtonSecondary())}, html.Text("Rebook lane")),
					html.Button(html.Props{Type: "button", Class: design.Class(design.ButtonQuiet())}, html.Text("Cancel")),
				),
			),

			// ScreenOnly: a live filter bar is chrome, and chrome does not print. Note
			// this is a caller decision, not a design-system heuristic — the same Recess
			// holding a filled-in receiving form must print. See print.go.
			html.Div(html.Props{Class: design.Class(design.Recess(), design.Cluster(design.Space3), design.ScreenOnly())},
				html.Div(html.Props{Class: design.Class(design.Field())},
					html.Label(html.Props{For: "q", Class: design.Class(design.FieldLabel())}, html.Text("Search notes")),
					html.Input(html.Props{ID: "q", Type: "search", Class: design.Class(design.Input()),
						Aria: map[string]string{"describedby": "q-hint"}}),
					html.Span(html.Props{ID: "q-hint", Class: design.Class(design.FieldHint())}, html.Text("Matches operator notes only.")),
				),
				html.Div(html.Props{Class: design.Class(design.Field())},
					html.Label(html.Props{For: "sku", Class: design.Class(design.FieldLabel())}, html.Text("SKU")),
					html.Input(html.Props{ID: "sku", Class: design.Class(design.InputData()),
						Aria: map[string]string{"invalid": "true", "describedby": "sku-err"}}),
					html.Span(html.Props{ID: "sku-err", Class: design.Class(design.FieldError())}, html.Text("No such SKU in this hub.")),
				),
			),

			html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
				// tabindex="0" must go through Raw, not Props.TabIndex: html.Props
				// silently omits a zero-valued TabIndex (html/html.go:59), and zero is
				// exactly the value a scroll region needs. See TableScroll's doc.
				html.Div(html.Props{Class: design.Class(design.TableScroll()), Role: "region",
					Raw:  map[string]any{"tabIndex": 0},
					Aria: map[string]string{"label": "Discrepancy lines"}},
					html.Table(html.Props{Class: design.Class(design.Table())},
						html.Thead(html.Props{},
							html.Tr(html.Props{},
								html.Th(html.Props{}, html.Text("SKU")),
								html.Th(html.Props{}, html.Text("Origin")),
								html.Th(html.Props{}, html.Text("Dest")),
								html.Th(html.Props{Class: design.Class(design.NumericCell())}, html.Text("Short")),
								html.Th(html.Props{Class: design.Class(design.NumericCell())}, html.Text("Promise")),
								html.Th(html.Props{}, html.Text("Posture")),
								html.Th(html.Props{}, html.Text("Note")),
							),
						),
						html.Tbody(html.Props{},
							queueRow("SKU-40192", "NJ-HUB", "IL-HUB", "-4", "2026-08-04", design.ToneException, "SHORT 4"),
							queueRow("SKU-40193", "NJ-HUB", "IL-HUB", "0", "2026-08-04", design.ToneVerified, "CLEAN"),
							queueRow("SKU-41007", "TX-HUB", "IL-HUB", "0", "2026-08-11", design.TonePending, "IN TRANSIT"),
						),
					),
				),
			),

			html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space4))},
				html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text("RCV-ILLINOIS-001")),
				html.A(html.Props{Href: "/internal/inventory", Class: design.Class(design.Link())}, html.Text("Open on-hand for this hub")),
			),

			// PrintOnly: the provenance footer. A printed receiving document that cannot
			// be traced back to its hub, its id and its print time is not paperwork.
			html.Div(html.Props{Class: design.Class(design.PrintOnly(), design.Data(design.StepMicro))},
				html.Text("RCV-ILLINOIS-001 · IL-HUB · PRINTED 2026-07-26"),
			),
		),
	)
}

// railItem is the rail's label + mono route code pair. RailLink is already
// justify-between, so the code lands in a trailing column that aligns down the rail —
// which is the whole point of setting it in Data.
func railItem(parseHref, parseLabel, parseCode string, parseCurrent bool) ui.Node {
	parseClass := design.Class(design.RailLink())
	parseProps := html.Props{Href: parseHref, Class: parseClass}
	if parseCurrent {
		parseProps.Class = design.Class(design.RailLinkCurrent())
		parseProps.Aria = map[string]string{"current": "page"}
	}
	return html.A(parseProps,
		html.Span(html.Props{}, html.Text(parseLabel)),
		html.Span(html.Props{Class: design.Class(design.RailCode())}, html.Text(parseCode)),
	)
}

// --- the storefront half of the probe -----------------------------------------

// storefrontPage exercises the catalog manifest, which is the primitive this probe exists
// to review. It renders BOTH populated and empty states in one document, because the
// empty state is the one a reviewer never sees and therefore the one that ships broken.
//
// The four products are deliberately four — the count that leaves an orphan row with two
// holes in a 3-column card grid, which is one of the concrete failures the manifest form
// makes structurally impossible.
func storefrontPage() ui.Node {
	return html.Div(html.Props{},
		html.Div(html.Props{Class: design.Class(design.StorefrontHeader())},
			html.Div(html.Props{Class: design.Class(design.StorefrontHeaderInner())},
				html.Span(html.Props{Class: design.Class(design.RailPlateCode())}, html.Text("ATLAS")),
				html.A(html.Props{Href: "/shop", Class: design.Class(design.Link())}, html.Text("Catalog")),
			),
		),
		html.Div(html.Props{Class: design.Class(design.StorefrontMain())},
			html.Div(html.Props{Class: design.Class(design.PageHead())},
				html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Catalog / Workspace systems")),
				html.H2(html.Props{Class: design.Class(design.PageTitle())}, html.Text("Modular workspace catalog")),
			),

			// The populated, FILTERED manifest: four of sixty-two lines, and the result
			// note says so — the difference between "narrow filter" and "empty shop".
			html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
				design.Catalog(design.CatalogSpec{
					Label:         "Product catalog",
					TotalCount:    62,
					FilterSummary: "CATEGORY DESKS",
					Items: []design.CatalogItem{
						{
							Href: "/shop/meridian-standing-desk", Title: "Meridian height-adjustable desk",
							SKU: "SKU-40192", Category: "Desks", Price: "$1,240.00",
							Summary: "Dual-motor column with a 24mm compressed-bamboo top and a cable spine that runs the full width of the frame.",
							Avail:   design.AvailStocked, Hub: "IL-HUB", Promise: "2026-08-04",
						},
						{
							Href: "/shop/harbor-bench-run", Title: "Harbor six-seat bench run",
							SKU: "SKU-40193", Category: "Desks", Price: "$4,980.00",
							Summary: "Shared spine bench for six, shipped as three two-seat modules so a freight lane can carry a partial order.",
							Avail:   design.AvailNearby, Hub: "NJ-HUB", Promise: "2026-08-19",
						},
						{
							Href: "/shop/kestrel-sit-stand", Title: "Kestrel compact sit-stand",
							SKU: "SKU-41007", Category: "Desks", Price: "$860.00",
							Summary: "Single-column desk for 1200mm bays, with the same controller as the Meridian so a floor can mix both.",
							Avail:   design.AvailInbound, Hub: "TX-HUB", Promise: "2026-09-02",
						},
						{
							Href: "/shop/foundry-drafting-table", Title: "Foundry drafting table",
							SKU: "SKU-41184", Category: "Desks", Price: "$2,150.00",
							Summary: "Tilting steel-frame drafting surface. Discontinued by the supplier; no replenishment lane is open.",
							Avail:   design.AvailNone,
						},
					},
				}),
			),

			// The EMPTY manifest, over-filtered, with one recovery action.
			html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
				design.Catalog(design.CatalogSpec{
					Label:            "Product catalog, no results",
					TotalCount:       62,
					FilterSummary:    "CATEGORY SEATING · IN STOCK · HUB TX-HUB",
					EmptyTitle:       "No lines match this filter",
					EmptyBody:        "Nothing in Seating is on hand at TX-HUB right now. Widen the hub or drop the stock filter to see inbound lines.",
					EmptyActionLabel: "Clear filters",
					EmptyActionHref:  "/shop",
				}),
			),
		),
	)
}

func queueRow(parseSKU, parseFrom, parseTo, parseShort, parsePromise string, parseTone design.StatusTone, parseLabel string) ui.Node {
	// Note the roles at work: the SKU, hub codes, quantity and date are machine facts
	// and inherit the table's Data voice; the short count additionally gets StatusValue
	// so the number itself carries the exception; the note is the one ProseCell.
	parseShortClass := design.Class(design.NumericCell())
	if parseTone == design.ToneException {
		parseShortClass = design.Class(design.NumericCell(), design.StatusValue(design.ToneException))
	}
	return html.Tr(html.Props{},
		html.Th(html.Props{}, html.Text(parseSKU)),
		html.Td(html.Props{}, html.Text(parseFrom)),
		html.Td(html.Props{}, html.Text(parseTo)),
		html.Td(html.Props{Class: parseShortClass}, html.Text(parseShort)),
		html.Td(html.Props{Class: design.Class(design.NumericCell(), design.CellMeta())}, html.Text(parsePromise)),
		html.Td(html.Props{},
			html.Span(html.Props{Class: design.Class(design.StatusChip(parseTone))}, html.Text(parseLabel)),
		),
		html.Td(html.Props{Class: design.Class(design.ProseCell())},
			html.Text("Carrier reported a partial load at the dock; remainder rebooked on the next lane."),
		),
	)
}

// extractClasses pulls every class name out of rendered markup.
func extractClasses(parseHTML string) []string {
	var parseOut []string
	parseRest := parseHTML
	for {
		parseIndex := strings.Index(parseRest, `class="`)
		if parseIndex < 0 {
			return parseOut
		}
		parseRest = parseRest[parseIndex+len(`class="`):]
		parseEnd := strings.Index(parseRest, `"`)
		if parseEnd < 0 {
			return parseOut
		}
		parseOut = append(parseOut, strings.Fields(parseRest[:parseEnd])...)
		parseRest = parseRest[parseEnd:]
	}
}
