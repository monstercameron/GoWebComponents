package atlas

// =============================================================================
// THE PUBLIC STOREFRONT, rendered in the Go typed-CSS design system.
// =============================================================================
//
// Every class on this surface now comes from shared/design (read its doc.go
// first). There is no Tailwind string left in this file, and that is not a
// cosmetic migration — the old markup was the design system's own case study in
// what went wrong, so this file is where the fix has to be visible.
//
// WHAT THE OLD /shop LOOKED LIKE, AND WHY EACH PIECE IS GONE
//
//   - A product was FOUR nested rounded dark boxes: card -> "story" card ->
//     price rail -> two blurb pills, every one with the same border, radius and
//     shadow. Four identical frames means nothing on the card is more important
//     than anything else, so nothing reads as important at all. The design
//     system therefore ships no Card at all (surfaces.go) and the catalog is a
//     manifest (below).
//   - THREE competing heroes stacked before the first product: the route hero,
//     a "Catalog overview" band with its own stat cards, and a second stat row.
//     Now there is exactly one page head per route (renderPublicHero) and the
//     content starts immediately after it.
//   - The stat cards lied. "24 workspace products" was hardcoded above a
//     computed "4 products", and the warehouse directory printed "0 stocked
//     units" because /api/public/warehouses does not return stock at all. The
//     rule now: print a number only where the payload actually carries it, and
//     let the catalog's own result note be the count of record.
//   - Per-item filler repeated verbatim on every row ("Workspace upgrades with a
//     more considered systems view.", "Stock posture supports immediate quoting
//     and fulfillment follow-through."). Copy that is identical across items
//     carries zero information per item; it is noise with a word count. Deleted.
//     If a product has nothing distinguishing to say, this file says nothing.
//   - Redundant per-card chrome: the category twice (eyebrow + slug line), an
//     "ATLAS SYSTEM" badge on every card, a "READY FOR ACTIVE PROJECTS" badge on
//     every card, and the description twice (summary + SEO description). All
//     gone. One summary, one category, one status.
//
// WHY THE CATALOG IS A MANIFEST ROW LIST AND NOT A CARD GRID
//
// This is the decision a future contributor is most likely to reverse, so:
// design.Catalog renders <ul role="list"> of <li><a>, each row a CSS grid laid
// out on tracks shared with a labelled header strip. It is not a grid of cards
// and it is not a <table>. The full argument lives in design/catalog.go's header
// and should be read before changing the shape here, but the four reasons that
// bite hardest on THIS data set are:
//
//  1. Atlas has FOUR products. Four cards in a three-column grid is an orphan
//     row with two holes and no CSS fix, plus wildly unequal card heights. Rows
//     are full-width, so four items is four rows and five is five. The failure
//     mode is structurally impossible rather than merely tidied up.
//  2. Prices become a column. Comparing prices is most of what browsing a
//     catalog IS, and twenty-four prices in twenty-four boxes cannot be compared.
//  3. Availability gets its own labelled column instead of being the fourth line
//     of a paragraph. Warehouse-aware availability is Atlas's entire thesis; on
//     the old cards it was buried inside the filler. Promoting it is the point.
//  4. repository.Product has no image field, so the thumbnail slot falls back to
//     design.CatalogThumbPlate: the SKU tail set in mono on pressed paper, like a
//     bin label. That is legible and true, where an empty grey square is neither.
//
// If you are about to turn this back into cards, note that you would also be
// re-introducing the orphan row, the incomparable prices and the buried
// availability. The row list is the argument, not a styling preference.
//
// A NOTE ON THE PRODUCT TITLE'S TYPE ROLE
//
// design.CatalogTitle is Prose, not Display, and that looks like a violation of
// "every label is Display". It is not: Display names a REGION of the page, and
// twenty-four condensed uppercase product titles would be a wall of shouting
// that outranks the page title. catalog.go has the comment; do not "fix" it.
//
// WHAT DID NOT CHANGE
//
// No hook, handler, route, form action or form field name was touched. The
// public forms' `name` attributes are asserted by browser specs, so the field
// primitives keep their exact ids, names and aria wiring and only their classes
// changed. The lazy sections, error boundaries and cached resources keep their
// copy where a spec asserts it as evidence of the BEHAVIOUR (deferred mount,
// background refresh, cache reuse) rather than as decoration.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/api"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// =============================================================================
// LOCAL BUNDLES — the four shapes the design system deliberately does not ship
// =============================================================================
//
// Each of these is a []css.Rule built once at package init and folded by
// design.Class at the call site. That split matters: design.Class EMITS to the
// css sink, and a package-level fold would hand out class names whose CSS a
// css.Reset() in tests had already thrown away. Bundles are inert; folds are
// lazy. (See design/doc.go, "Why bundles are package-level vars".)
//
// All four spend design tokens only — no literal color, no off-scale spacing —
// so a theme change still reaches them. If a fifth surface needs one of these,
// promote it into shared/design instead of copying it.

// atlasBuyingPageBundle is the two-column page split: document on the left, a
// persistent buying rail on the right.
//
// design has exactly one grid (the catalog's shared track definition) and
// otherwise composes with Stack, Cluster and SplitRow — which is right for a
// document. The product and availability routes are a document PLUS a rail that
// must stay reachable while the buyer reads, and that is a page-level grid.
//
// grid-template-columns has no typed constructor (the same gap catalog.go calls
// out), so the tracks are css.Raw. The collapse point is 960px to match the
// breakpoint design uses for "the rail stops being a rail".
var atlasBuyingPageBundle = css.Rules(
	css.Raw("display", "grid"),
	css.Raw("grid-template-columns", "minmax(0,1.6fr) minmax(19rem,0.8fr)"),
	css.Raw("align-items", "start"),
	css.Gap(design.Space6),
	// min-width:0 on the container AND minmax(0,...) on the track: the pair that
	// makes a grid column actually shrinkable. Without it one long mono SKU sets
	// the column's min-content width and the page grows a horizontal scrollbar
	// that appears to come from nowhere.
	css.MinWidth(css.Zero),
	css.Media(css.MaxW(960), css.Raw("grid-template-columns", "minmax(0,1fr)")),
)

func atlasBuyingPageClass() string { return design.Class(atlasBuyingPageBundle) }

// atlasBuyingRailBundle is the sticky column the buying actions live in. It goes
// static below the collapse point, because a sticky element inside a single
// column steals viewport from the content it was meant to accompany.
var atlasBuyingRailBundle = css.Rules(
	design.Stack(design.Space5),
	css.Position.Sticky,
	css.Raw("top", string(design.Space5)),
	css.Media(css.MaxW(960), css.Position.Static),
)

func atlasBuyingRailClass() string { return design.Class(atlasBuyingRailBundle) }

// atlasRuledRowBundle is a full-width ruled list row that is also a link: hub
// rows, related-product rows, promise-lane rows.
//
// It is deliberately the same IDEA as design.CatalogRow — rows separated by
// hairlines rather than boxed — but CatalogRow is laid out on the catalog's
// product tracks (thumb / identity / availability / price / action), which is
// wrong for a two-field hub row. This is the generic case, and it is the one
// primitive this file would most like to see promoted into shared/design.
//
// Hairline, not border: a row is a division of one sheet, not a new object.
var atlasRuledRowBundle = css.Rules(
	design.Stack(design.Space1),
	css.PaddingY(design.Space3),
	css.PaddingX(design.Space3),
	css.BorderTop(design.HairlineWidth, design.Hairline()),
	css.TextColor(design.Ink()),
	css.Raw("text-decoration", "none"),
	css.Hover(css.Bg(design.PaperSunk())),
	css.FocusVisible(
		css.Outline(design.ManifestRuleWidth, design.Lane()),
		css.OutlineOffset(css.Px(-2)),
	),
)

func atlasRuledRowClass() string { return design.Class(atlasRuledRowBundle) }

// atlasRuledListBundle holds those rows. No gap: the rows carry their own
// hairlines, so the list is RULED rather than spaced — which is what lets values
// line up down the page.
var atlasRuledListBundle = css.Rules(
	css.Display.Flex,
	css.FlexDir.Col,
	css.Raw("list-style", "none"),
	css.Padding(css.Zero),
	css.Margin(css.Zero),
	css.MinWidth(css.Zero),
)

func atlasRuledListClass() string { return design.Class(atlasRuledListBundle) }

// atlasMeterTrackBundle / atlasMeterFillBundle are the buyer-sentiment bar.
//
// The fill is StatusVerified because the bar measures the share of approved
// feedback that is positive — a semantic tone, not a decorative green. Its width
// is the one inline style in this file: a percentage class would mint up to 101
// hashed classes for one bar.
var atlasMeterTrackBundle = css.Rules(
	css.Display.Block,
	css.W(css.Full),
	css.H(css.Px(6)),
	css.Bg(design.PaperSunk()),
	css.Border(design.HairlineWidth, design.Hairline()),
	css.Rounded(design.RadiusTag),
	css.Raw("overflow", "hidden"),
)

var atlasMeterFillBundle = css.Rules(
	css.Display.Block,
	css.H(css.Full),
	css.Bg(design.StatusVerified()),
)

// atlasSkeletonBundle is a loading placeholder: pressed paper at the height of
// the row it stands in for, so the deferred module does not shift the page when
// it resolves. No shimmer animation — a pulsing gradient is three of the things
// the design system removed.
var atlasSkeletonBundle = css.Rules(
	css.Display.Block,
	css.H(css.Px(60)),
	css.Bg(design.PaperSunk()),
	css.Rounded(design.RadiusTag),
)

// =============================================================================
// DOMAIN -> DESIGN MAPPINGS
// =============================================================================
//
// design.Availability and design.Posture are closed enums with no color escape
// hatch: a caller passes a domain state and the design system decides what it
// looks like. These four functions are the whole bridge, which is why they are
// small switches in one place rather than inline conditionals at call sites.

// atlasLineHub is the hub code for a manifest line, or "" when there is no hub.
//
// The guard is the point. page.go's atlasHubCode compresses a warehouse slug to
// initials and appends "-HUB", and for an EMPTY input it falls back to the bare
// string "HUB" — a sensible default when you know you are rendering a hub, and a
// live bug when you are rendering a product that has no hub: the catalog's
// availability line printed the label "HUB" with nothing after it on every row of
// /shop, which is worse than printing nothing, because it looks like a value that
// failed to load.
//
// So: no hub, no line. design.CatalogLine omits the promise line entirely when
// both Hub and Promise are empty, and the chip above it still carries the answer.
func atlasLineHub(parseWarehouseID string) string {
	if strings.TrimSpace(parseWarehouseID) == "" {
		return ""
	}
	return atlasHubCode(parseWarehouseID)
}

// atlasSKUCode renders a product SKU as the mono code the catalog's identity
// column and its thumbnail bin plate expect.
//
// Atlas's SKUs are slugs ("frame-desk"), and the console's atlasHubCode (page.go)
// is the wrong tool for them: it compresses a WAREHOUSE slug to initials plus
// "-HUB". A product code must stay whole — it is the string a buyer quotes back
// on the phone — so this only uppercases it. "FRAME-DESK" reads as a code where
// "frame-desk" reads as a URL fragment, and design.CatalogThumbPlate takes the
// tail after the last hyphen for the bin label.
func atlasSKUCode(parseValue string) string {
	return strings.ToUpper(strings.TrimSpace(parseValue))
}

// atlasStockAvailability maps real on-hand and inbound counts onto the buyer
// availability enum. Used wherever the payload actually carries counts: the
// availability route, and any product row whose inventory columns are populated.
func atlasStockAvailability(parseAvailable int, parseInbound int) design.Availability {
	switch {
	case parseAvailable > 0:
		return design.AvailStocked
	case parseInbound > 0:
		return design.AvailInbound
	default:
		return design.AvailNone
	}
}

// atlasStatusAvailability maps a product STATUS onto the buyer availability enum,
// for the catalog rows, where the payload has no counts at all (see
// server/db/store.go Catalog: it selects sku, slug, title, category, price,
// status, summary, seo_description and nothing else).
//
// The interesting case is "low_stock" -> AvailInbound, and it is a compromise
// worth naming: design.Availability has no LIMITED state. AvailStocked would
// print "IN STOCK" in the verified tone over an item the operator has flagged as
// below threshold, which overstates it; AvailNearby would claim the units are in
// a different hub, which we do not know. AvailInbound prints "INBOUND" in the
// pending tone and changes nothing else, which is the honest de-emphasis. The
// real fix is a fifth enum case in design (reported as a gap).
func atlasStatusAvailability(parseStatus string) design.Availability {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return design.AvailStocked
	case "low_stock", "pending", "submitted", "in_review":
		return design.AvailInbound
	default:
		return design.AvailNone
	}
}

// atlasProductAvailability prefers real counts and falls back to status.
//
// Counts win because they are the specific claim and status is the summary of
// it. This is also what makes one catalog primitive serve both the storefront
// catalog (status only) and the warehouse-scoped product list (counts).
func atlasProductAvailability(parseItem productCard) design.Availability {
	if parseItem.Available > 0 || parseItem.Inbound > 0 {
		return atlasStockAvailability(parseItem.Available, parseItem.Inbound)
	}
	return atlasStatusAvailability(parseItem.Status)
}

// atlasStockPosture maps counts onto the lane placard's posture.
//
// PostureShort for "nothing on hand and nothing inbound" is the ONE place the
// storefront spends the exception hue, and it is earned: the promise cannot be
// met. Everything else is on-lane or held, so oxide stays rare enough to still
// mean "a human has to look at this".
func atlasStockPosture(parseAvailable int, parseInbound int) design.Posture {
	switch {
	case parseAvailable > 0:
		return design.PostureOnLane
	case parseInbound > 0:
		return design.PostureHeld
	default:
		return design.PostureShort
	}
}

// atlasPromiseWindow formats a warehouse service level ("2-4 days") as the mono
// promise value on a placard or an availability line.
//
// Atlas promises a WINDOW, not a date — there is no promise date in the public
// payload — so this prints the window rather than manufacturing a date. Mono and
// uppercase, because it is a machine fact and it has to align down a column.
func atlasPromiseWindow(parseServiceLevel string) string {
	return strings.ToUpper(strings.TrimSpace(parseServiceLevel))
}

// atlasBuyerDestination is the destination on a storefront lane placard.
//
// The placard is ORIGIN -> DEST and Atlas knows the origin hub exactly. It does
// not know the buyer's address, so the honest destination on a public page is
// the buyer themselves. Naming it in the buyer's own terms keeps the placard's
// accessible sentence true ("Lane NEW-JERSEY-HUB to YOUR SITE, promise 2-4
// DAYS, posture ON LANE") instead of inventing a hub code nobody can look up.
const atlasBuyerDestination = "YOUR SITE"

// =============================================================================
// LANDING (/)
// =============================================================================

func renderLandingContent() ui.Node {
	return html.Section(html.Props{Class: design.Class(design.Stack(design.Space5))},
		publicLandingIntroCard(),
	)
}

// publicLandingIntroCard is ONE surface where there used to be an intro card,
// two metric cards and three feature cards — eight boxes making the same claim
// eight times. The three points below are a ruled list inside the surface, not
// three more boxes: separation is a hairline, never another frame.
func publicLandingIntroCard() ui.Node {
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))},
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(publicWhyAtlasFeelsReady)),
		html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Every product shows the hub that ships it")),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepLede), design.Measure())},
			html.Text("Atlas prices workspace systems against real hub stock, so you can read a desk and its delivery window on the same line."),
		),
		html.Div(html.Props{Class: design.Class(design.Divider())}),
		html.Ul(html.Props{Class: atlasRuledListClass(), Role: "list"},
			publicLandingPoint("Price and stock together", "Each catalog line carries its price and whether a hub can fill it."),
			publicLandingPoint("One window per hub", "Open a hub to see what it holds for a product and how long it takes."),
			publicLandingPoint("Ask on the product", "Quotes, reservations and questions post from the product you are reading."),
		),
	)
}

func publicLandingPoint(parseTerm string, parseCopy string) ui.Node {
	return html.Li(html.Props{Class: design.Class(atlasRuledRowBundle)},
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseTerm)),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseCopy)),
	)
}

// =============================================================================
// CATALOG (/shop) — the manifest
// =============================================================================

// renderCatalogContent is the filter bar plus one manifest. Nothing else.
//
// Every hook, the deferred value, the debounce and the query-sync effect are
// unchanged; only the markup below them is. What came OUT is the entire second
// and third hero: publicCatalogOverview (an eyebrow, an H2, a paragraph and two
// stat cards, one of which recomputed a count the manifest already reports) and
// the four-deep product card.
func renderCatalogContent(parsePage catalogPage) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseSearch := useAtlasSearchParams()
		parseInitial := atlasCatalogFilterState(parsePage.Query)
		parseForm := ui.UseForm(parseInitial)
		useAtlasEffect(func() func() {
			if !atlasSameListFilterState(parseForm.Get(), parseInitial) {
				parseForm.Reset(parseInitial)
			}
			return nil
		}, parseInitial)
		parseValue := parseForm.Get()
		parseDeferred := ui.UseDeferredValue(parseValue)
		parseDebounced := ui.UseDebounced(parseValue, atlasFilterSyncDelay)
		useAtlasEffect(func() func() {
			parseNext := atlasBuildListFilterQuery(parseSearch.Values(), parseDebounced.Get())
			if parseNext.Encode() != parseSearch.Values().Encode() {
				parseSearch.ReplaceAll(parseNext)
			}
			return nil
		}, parseDebounced.Get())
		parseSubmit := ui.UseEvent(func(parseEvent ui.FormEvent) {
			parseEvent.PreventDefault()
			parseSearch.ReplaceAll(atlasBuildListFilterQuery(parseSearch.Values(), parseForm.Get()))
		})
		parseFilteredItems := filterCatalogItems(parsePage.Items, parseDeferred)
		return html.Section(html.Props{Class: design.Class(design.Stack(design.Space5))},
			html.P(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Catalog overview")),
			storeCatalogControls(parseForm, parseDebounced.Pending(), parseSubmit),
			// SurfaceFlush, not Surface: the manifest reaches its own hairline the
			// way a table does. A ruled list inside a 16px gutter throws away the
			// column alignment it was chosen for.
			html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
				design.Catalog(design.CatalogSpec{
					// The active hub filter is handed to the lines, not just to the
					// result note: when the buyer has narrowed to a hub, EVERY line in
					// the result is provably stocked there (store.Catalog filters on
					// `exists (select 1 from inventory_levels ...)`), so naming it in
					// the availability column is a fact the query already established.
					Items:      atlasCatalogLines(parseFilteredItems, parseDeferred.Warehouse),
					TotalCount: atlasCatalogTotal(parsePage),
					// The filter summary is empty unless something is actually
					// narrowed, so an unfiltered visit gets a bare count instead of
					// chrome the reader learns to ignore.
					FilterSummary:    atlasCatalogFilterSummary(parseDeferred),
					Label:            "Product catalog",
					EmptyTitle:       "No lines match this filter",
					EmptyBody:        "Nothing in the catalog matches what you asked for. Widen the category or clear the filter to see every line.",
					EmptyActionLabel: "Clear filters",
					EmptyActionHref:  RouteCatalog,
				}),
			),
		)
	})
}

// atlasCatalogTotal is how many lines exist before the client-side filter runs.
//
// design.Catalog prints "N OF M LINES" only when M is larger than what is shown,
// which is exactly the case this feeds: the server has already paginated, and the
// deferred filter narrows that page further. Passing the server's own total keeps
// a narrow filter from reading as an empty shop.
func atlasCatalogTotal(parsePage catalogPage) int {
	if parsePage.Total > len(parsePage.Items) {
		return parsePage.Total
	}
	return len(parsePage.Items)
}

// atlasCatalogLines turns product rows into manifest lines.
//
// # Where the availability column's hub line comes from, and where it does not
//
// This is the one place the storefront catalog can lie, so the sourcing is spelled
// out. The hub under the availability chip is taken, in order, from:
//
//  1. the ROW's own warehouse, when the payload carries one. It does on the
//     warehouse-scoped lists and in the test fixtures.
//  2. the ACTIVE HUB FILTER, when the buyer has narrowed to one. store.Catalog
//     implements that filter as `exists (select 1 from inventory_levels il where
//     il.product_sku = products.sku and il.warehouse_id = ?)`, so every returned
//     row is stocked at that hub by construction — printing it is reporting the
//     query, not guessing.
//  3. nothing. The line is omitted rather than rendered blank.
//
// Case 3 is the DEFAULT on an unfiltered /shop, and it is not an oversight: the
// public catalog payload is repository.Product, which is SKU, slug, title,
// category, price cents, status, summary and SEO description — no warehouse, no
// on-hand, no inbound, no promise date. server/db/store.go's Catalog query selects
// exactly those eight columns. There is no per-product hub or promise to render on
// an unfiltered catalog, and the honest response is silence.
//
// PROMISE is never set here for the same reason and one more: Atlas's public
// promise is a WINDOW published per hub ("2-4 days" on the warehouse record), not
// a per-product date, and the catalog payload carries neither. The warehouse-scoped
// manifest (atlasWarehouseCatalogLines) does have the hub record and does print it.
//
// If a future payload adds per-row availability, populate productCard.WarehouseID /
// Available / Inbound and every branch here starts working with no markup change —
// which is why case 1 is written first even though today it only fires off /shop.
//
// Case 2 is also currently unreachable on /shop for a reason that lives outside this
// file: filterCatalogItems (products_cms.go) re-applies the hub filter on the client
// against productCard.WarehouseID, which the public catalog payload never sets, so a
// hub-filtered /shop drops every row before it reaches here. That is a pre-existing
// behaviour bug in another surface, not something this markup pass introduced or is
// allowed to fix; the branch is written correctly so it starts working the day that
// filter does.
//
// # The thumbnail plate reads "DESK", not "40192"
//
// design.CatalogThumbPlate prints the SKU's tail after the last hyphen, on the
// assumption of "SKU-40192" — a code with a numeric distinguishing tail. Atlas's
// SKUs are noun-phrase slugs ("frame-desk", "cable-bridge"), so the tail is the
// CATEGORY noun and the head is the distinguishing word. The plates therefore read
// DESK / CONSOLE / BENCH / BRIDGE. That is legible, true, and unique across the
// current four products, so it is kept rather than worked around — but it is exactly
// backwards from the primitive's intent and it collides the moment a second desk
// ships. The fix belongs in design (a plate-code override on CatalogItem, or falling
// back to the leading segment when the tail is not numeric), not in a caller
// reshaping a SKU it also has to print correctly in the meta line.
//
// Note also what is NOT set: no ThumbSrc (repository.Product has no image, so the
// slot falls back to the SKU bin plate) and no per-item editorial copy. Summary is
// passed through verbatim and clamped by the primitive; the SEO description is not
// repeated under it.
//
// SKU is uppercased because Atlas's SKUs are slugs. The mono column and the bin
// plate both want a code, and "FRAME-DESK" reads as one where "frame-desk" reads
// as a URL fragment.
func atlasCatalogLines(parseItems []productCard, parseHubFilter string) []design.CatalogItem {
	parseFilterHub := ""
	if parseNormalized := atlasNormalizedFilterValue(parseHubFilter); parseNormalized != "" && parseNormalized != "all" {
		parseFilterHub = atlasLineHub(parseNormalized)
	}
	parseLines := make([]design.CatalogItem, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseHub := atlasLineHub(parseItem.WarehouseID)
		if parseHub == "" {
			parseHub = parseFilterHub
		}
		parseLines = append(parseLines, design.CatalogItem{
			Href:     RouteCatalog + "/" + parseItem.Slug,
			Title:    parseItem.Title,
			SKU:      atlasSKUCode(parseItem.SKU),
			Category: parseItem.Category,
			Price:    formatPrice(parseItem.PriceCents),
			Summary:  parseItem.Summary,
			Avail:    atlasProductAvailability(parseItem),
			Hub:      parseHub,
		})
	}
	return parseLines
}

// atlasCatalogFilterSummary formats the active filter for the manifest's result
// note, in Data voice because a filter expression is a machine fact.
//
// It returns "" when nothing is narrowed. That is the whole trick: a note that
// says "everything" on every visit is invisible by the time it says "twelve".
func atlasCatalogFilterSummary(parseFilters atlasListFilterState) string {
	parseParts := make([]string, 0, 3)
	if parseQuery := strings.TrimSpace(parseFilters.Query); parseQuery != "" {
		parseParts = append(parseParts, "MATCHING "+strings.ToUpper(parseQuery))
	}
	if parseCategory := atlasNormalizedFilterValue(parseFilters.Category); parseCategory != "" && parseCategory != "all" {
		parseParts = append(parseParts, "CATEGORY "+strings.ToUpper(parseCategory))
	}
	if parseWarehouse := atlasNormalizedFilterValue(parseFilters.Warehouse); parseWarehouse != "" && parseWarehouse != "all" {
		parseParts = append(parseParts, "HUB "+atlasLineHub(parseWarehouse))
	}
	return strings.Join(parseParts, " · ")
}

// publicLazySectionFallback is the placeholder a deferred section shows before it
// mounts. One surface, one eyebrow, one title, one sentence.
func publicLazySectionFallback(parseEyebrow, parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))},
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseEyebrow)),
		html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseTitle)),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())}, html.Text(parseCopy)),
	)
}

// =============================================================================
// PRODUCT DETAIL (/shop/{slug})
// =============================================================================

func renderProductContent(parsePage productDetailPage, parsePayload Payload) ui.Node {
	parseProduct := parsePage.Product
	return html.Section(html.Props{Class: atlasBuyingPageClass()},
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space5))},
			publicProductHeroCard(parseProduct),
			atlasLazySection(func() ui.Node {
				return publicProductFeedbackSection(parseProduct, parsePage.Comments, parsePayload)
			}, publicLazySectionFallback("Customer reviews and questions", "Loading buyer feedback", "Atlas waits until the primary product story is stable before mounting the heavier review and question workflow."), parseProduct.Slug, commentRecordsSignature(parsePage.Comments)),
			publicProductPromiseLanesIsland(parseProduct),
		),
		publicProductActionRail(parseProduct, parsePayload),
	)
}

type relatedProductRecord struct {
	SKU           string `json:"sku"`
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	Summary       string `json:"summary"`
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	Reason        string `json:"reason"`
}

// publicProductHeroCard is the product's one surface: what it is, what it costs,
// and whether it can ship.
//
// It does NOT repeat the product name. The route's <h1> is already "Atlas Frame
// Desk" (server.go builds it from the product title), and the old page printed
// that name again as an <h2> immediately underneath — one of the three heroes.
// One page, one title.
//
// What else came off: three "signal pills" of status-derived filler, a nested
// price card, and the SEO description printed under the summary that already
// said the same thing.
func publicProductHeroCard(parseProduct productCard) ui.Node {
	parseAvail := atlasProductAvailability(parseProduct)
	parseChildren := []ui.Node{
		publicProductIdentity(parseProduct),
	}
	if strings.TrimSpace(parseProduct.Summary) != "" {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Prose(design.StepLede), design.Measure())}, html.Text(parseProduct.Summary)),
		)
	}
	parseChildren = append(parseChildren,
		html.Div(html.Props{Class: design.Class(design.ManifestRule())}),
		// The price is the row a buyer came for, so it gets the heavy rule above it
		// and the only StepLede mono figure on the page. Data, not Prose: a price is
		// a machine fact, and tabular figures make two prices comparable.
		html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space4))},
			html.Div(html.Props{Class: design.Class(design.Stack(design.Space1))},
				html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(publicStartingAtLabel)),
				html.Span(html.Props{Class: design.Class(design.Data(design.StepHead))}, html.Text(formatPrice(parseProduct.PriceCents))),
			),
			html.Span(html.Props{Class: design.Class(design.StatusChip(parseAvail.Tone()))}, html.Text(parseAvail.Label())),
		),
	)
	if parseFinish := strings.TrimSpace(parseProduct.Finish); parseFinish != "" {
		parseChildren = append(parseChildren, publicProductContextColumn("Finish", parseFinish))
	}
	if parseDetails := strings.TrimSpace(parseProduct.Details); parseDetails != "" {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())}, html.Text(parseDetails)),
		)
	}
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))}, parseChildren...)
}

// publicProductIdentity is the mono identity line: SKU, then category. Both are
// machine facts, so both are Data, and being mono they align with the same pair
// on every catalog row the buyer just came from.
//
// The category used to appear twice here (an amber eyebrow AND a slug line) plus
// an "Atlas system" badge on every product. Once.
func publicProductIdentity(parseProduct productCard) ui.Node {
	parseParts := make([]ui.Node, 0, 3)
	if parseSKU := atlasSKUCode(parseProduct.SKU); parseSKU != "" {
		parseParts = append(parseParts, html.Span(html.Props{}, html.Text(parseSKU)))
	}
	if parseCategory := strings.TrimSpace(parseProduct.Category); parseCategory != "" {
		if len(parseParts) > 0 {
			parseParts = append(parseParts,
				html.Span(html.Props{Aria: map[string]string{"hidden": "true"}}, html.Text("·")))
		}
		parseParts = append(parseParts, html.Span(html.Props{}, html.Text(strings.ToUpper(parseCategory))))
	}
	return html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2), design.Data(design.StepFine))}, parseParts...)
}

// publicProductContextColumn is a label/value pair, not a card.
//
// It used to be a bordered, shadowed, radiused box, which is how a page ends up
// with four nested frames. A label in Eyebrow voice above a value needs no frame
// to read as a pair.
func publicProductContextColumn(parseLabel string, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.Stack(design.Space1))},
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseLabel)),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseCopy)),
	)
}

// publicProductPromiseLanesIsland is unchanged behaviour: an error boundary
// around a resource-backed async boundary, keyed on the product.
func publicProductPromiseLanesIsland(parseProduct productCard) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ResetKeys: []any{parseProduct.Slug, parseProduct.Status},
		ErrorFallback: func(parseErr error, reset func()) ui.Node {
			return publicProductPromiseLanesError(parseProduct, parseErr, reset)
		},
		Child: ui.CreateElement(func() ui.Node {
			// A TYPED SERVER FUNCTION, not a URL and a decode.
			//
			// This used to read:
			//
			//	fetchAtlasJSON[warehouseDirectoryPage](parseCtx, "/api/public/warehouses")
			//
			// Three things were wrong with that and none of them were visible here.
			// The path was a string, so a route rename broke it at runtime. The
			// response type was asserted by the caller rather than agreed with the
			// server, so a renamed JSON field decoded to a zero value and rendered as
			// an empty region. And warehouseDirectoryPage carried operator-only fields
			// (Pressure, Staffing, Backlog, Focus) that the public endpoint never
			// populates — which is exactly how this page ended up printing a hardcoded
			// "Core team assigned" to buyers for every hub.
			//
			// api.ListWarehouses is one Go signature that `gwc server gen` turns into
			// both the server registration and this call. The compiler checks it, and
			// api.Warehouse is the PUBLIC shape, so the operator fields cannot be
			// reached from here even by accident.
			parseResource := useAtlasResource(func(parseCtx context.Context) (api.ListWarehousesResponse, error) {
				return api.ListWarehouses(parseCtx, api.ListWarehousesRequest{})
			}, parseProduct.Slug)
			parseState := parseResource.Get()
			parseContent := publicProductPromiseLanesCard(parseProduct, parseState.Value.Warehouses, parseState.Loading && parseState.Ready)
			return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
				Pending:  !parseState.Ready && parseState.Error == nil,
				Error:    parseState.Error,
				Fallback: publicProductPromiseLanesFallback(parseProduct),
				ErrorFallback: func(parseErr2 error) ui.Node {
					return publicProductPromiseLanesError(parseProduct, parseErr2, parseResource.Reload)
				},
				Content: parseContent,
			})
		}),
	})
}

// publicProductPromiseLanesCard lists the hubs that can serve this product.
//
// NO lane placard here, deliberately, and it is worth saying why given that a
// placard is the signature device and this section is about lanes: the placard
// prints a POSTURE, and /api/public/warehouses returns id, slug, name, region,
// service_level and public_summary — no stock, no inbound, no promise. A posture
// derived from nothing is a claim the payload cannot back, so this stays a ruled
// list of hub rows and the placard appears one route deeper, on the availability
// page, where the counts are real.
// The parameter is []api.Warehouse — the public wire type — rather than the
// internal warehouseCard. This card reads exactly three fields (Slug, Name,
// ServiceLevel) and api.Warehouse carries exactly what a buyer may see, so taking
// the narrower type makes it impossible for this surface to grow a dependency on
// an operator-only field the public payload never fills.
func publicProductPromiseLanesCard(parseProduct productCard, parseWarehouses []api.Warehouse, isRefreshing bool) ui.Node {
	parseRows := make([]ui.Node, 0, 3)
	for _, parseItem := range parseWarehouses {
		parseRows = append(parseRows, html.Li(html.Props{},
			html.A(html.Props{
				Href:  RouteWarehouses + "/" + parseItem.Slug + "/availability/" + parseProduct.Slug,
				Class: atlasRuledRowClass(),
			},
				html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3))},
					html.Span(html.Props{Class: design.Class(design.Prose(design.StepBase)), Style: map[string]string{"font-weight": "600"}}, html.Text(parseItem.Name)),
					html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(atlasLineHub(parseItem.Slug))),
				),
				html.Span(html.Props{Class: design.Class(design.Prose(design.StepFine))}, html.Text(fallback(parseItem.ServiceLevel, "Regional service posture"))),
				html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Open "+parseItem.Name+" availability")),
			),
		))
		if len(parseRows) == 3 {
			break
		}
	}
	parseChildren := []ui.Node{
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Regional promise lanes")),
		html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Pick the hub that serves you")),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())},
			html.Text("This below-the-fold module loads after the main product story, so route-critical content stays stable while the regional availability lanes resolve independently."),
		),
	}
	if isRefreshing {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Recess(), design.Prose(design.StepFine))},
				html.Text("Refreshing the regional lane list in the background while the current panel stays visible."),
			),
		)
	}
	if len(parseRows) == 0 {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())},
				html.Text("Atlas is still resolving warehouse lanes for this product."),
			),
		)
	} else {
		parseChildren = append(parseChildren,
			html.Ul(html.Props{Class: atlasRuledListClass(), Role: "list"}, parseRows...),
		)
	}
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))}, parseChildren...)
}

func publicProductPromiseLanesFallback(parseProduct productCard) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))},
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Regional promise lanes")),
		html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Pick the hub that serves you")),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())},
			html.Text("Atlas defers this secondary lane module until after hydration so loading stays local to the panel instead of blocking the product route."),
		),
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space2))},
			html.Div(html.Props{Class: design.Class(atlasSkeletonBundle)}),
			html.Div(html.Props{Class: design.Class(atlasSkeletonBundle)}),
			html.Div(html.Props{Class: design.Class(atlasSkeletonBundle)}),
		),
		html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(atlasSKUCode(parseProduct.SKU))),
	)
}

// publicProductPromiseLanesError is a COMPONENT because it is an ERROR FALLBACK.
//
// publicProductPromiseLanesIsland hands it to both an ui.ErrorBoundary
// ErrorFallback and an ui.AsyncBoundary ErrorFallback. The runtime calls those
// from recovery bookkeeping (internal/runtime/error_boundary.go and
// internal/runtime/async_boundary.go), not from a component render, so no fiber is
// current and a hook in the callback body panics.
//
// Observable failure: the storefront product page looks fine until
// /api/public/warehouses fails, and then the code meant to REPORT that failure
// panics instead — replacing a recoverable "could not load lanes" panel with a
// re-thrown "async boundary fallback panic" at the next boundary out.
//
// The retry button does not need a stable hook slot for correctness, only to avoid
// re-wrapping a js.Func on every commit; ui.CreateElement keeps ui.UseEvent (and
// therefore that caching) while giving the hook a fiber to live in. Do NOT reach
// for ui.WrapHandler with a raw closure here — that leaks one js.Func per call.
//
// Styling note: the error surface is a plain Surface with an exception CHIP, not
// an oxide-tinted panel. A whole panel painted in the exception hue spends the
// design system's scarcest resource on a recoverable network hiccup; the chip
// says the same thing and leaves oxide meaning something.
func publicProductPromiseLanesError(parseProduct productCard, parseErr error, parseRetry func()) ui.Node {
	return ui.CreateElement(func() ui.Node {
		return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))},
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
				html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Regional promise lanes")),
				html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneException))}, html.Text("LANES UNAVAILABLE")),
			),
			html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Atlas could not load the lane panel.")),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseErr.Error())),
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
				html.Button(html.Props{
					Type:    "button",
					Class:   design.Class(design.ButtonSecondary()),
					OnClick: ui.UseEvent(func() { parseRetry() }),
				}, html.Text("Retry lane panel")),
				html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(parseProduct.Title)),
			),
		)
	})
}

// publicProductActionRail is the buying column. Structure and hooks unchanged —
// including the reason buildRailChildren's children must each own a fiber.
func publicProductActionRail(parseProduct productCard, parsePayload Payload) ui.Node {
	parseProductSupportTitle, parseProductSupportCopy, parseProductSupportPoints := publicBuyerNextStep(parseProduct.Status)
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-public-action-rail"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		// buildRailChildren is invoked TWICE per render — once for the desktop column
		// and once for the mobile drawer — so everything it returns must be a COMPONENT.
		//
		// This is the failure that made the rule concrete. productPrimaryActionForm's
		// fields used to be plain helpers calling ui.UseId, so both invocations drew
		// from THIS fiber's single id sequence: the desktop email input got gwc-7-3 and
		// the drawer's got a different id, while the count itself varied because
		// productPrimaryActionForm switches on product status and emits two fields for
		// constrained stock and five for available stock. Result: aria-labelledby in the
		// drawer pointed at the desktop copy's label, and the parent's hook count moved
		// with product status.
		//
		// Now every child owns a fiber (productPrimaryActionForm's fields, publicRelatedProductsCard,
		// productSecondaryActionCard), so the two copies are independent and internally
		// consistent, and this component's own hook count is fixed at three.
		buildRailChildren := func() []ui.Node {
			return []ui.Node{
				html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))},
					html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(publicBuyerNextStepLabel)),
					html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseProductSupportTitle)),
					html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())}, html.Text(parseProductSupportCopy)),
					html.Ul(html.Props{Class: design.Class(design.Stack(design.Space2)), Role: "list"}, publicSupportPoints(parseProductSupportPoints)...),
				),
				productPrimaryActionForm(parseProduct, parsePayload),
				publicRelatedProductsCard(parseProduct),
				productSecondaryActionCard(parseProduct),
			}
		}
		return html.Div(html.Props{Class: atlasBuyingRailClass()},
			html.Button(html.Props{Type: "button", Class: design.Class(design.ButtonSecondary()), OnClick: parseOpenDrawer}, html.Text("Open buying drawer")),
			html.Div(html.Props{Class: design.Class(design.Stack(design.Space5))}, buildRailChildren()...),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: design.Class(design.Stack(design.Space5))},
				html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3))},
					html.Div(html.Props{Class: design.Class(design.Stack(design.Space1))},
						html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(publicBuyerNextStepLabel)),
						html.H3(html.Props{ID: parseTitleID, Class: design.Class(design.SectionTitle())}, html.Text(parseProductSupportTitle)),
						html.P(html.Props{ID: parseDescriptionID, Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseProductSupportCopy)),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: design.Class(design.ButtonQuiet()), OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: design.Class(design.Stack(design.Space5))}, buildRailChildren()...),
			)),
		)
	})
}

// publicBuyerNextStep is the buying guidance for a product status.
//
// It replaces derived_state.go's productSupportPlan at this call site, and the
// reason is the copy, not the shape: that function hands the BUYER sentences
// written for the team building the page ("Use the quote form as the primary
// action for real project intent.", "Explain that inventory is constrained in
// plain buyer language."). Those are design notes. A buyer needs plain verbs and
// what happens next.
func publicBuyerNextStep(parseStatus string) (string, string, []string) {
	switch normalizedAtlasStatus(parseStatus) {
	case "in_stock", "healthy", "approved", "available":
		return "Ask for a price", "This system is on hand, so a quote can go out today.", []string{
			"Send quantity and install dates and you get a price back.",
			"Pick a delivery region first if the date matters more than the price.",
		}
	case "low_stock", "pending", "submitted", "in_review":
		return "Reserve the next units", "Stock is thin. Reserving holds your place in the next batch.", []string{
			"Reserving costs nothing and does not commit you to buy.",
			"Name a hub and you get the window that hub can hit.",
		}
	default:
		return "Get told when it lands", "Nothing is on hand and no batch is booked yet.", []string{
			"Leave an email and you hear the day stock arrives.",
			"Browse the same category for something you can buy now.",
		}
	}
}

// =============================================================================
// PRODUCT FEEDBACK — reviews and questions
// =============================================================================

func publicProductFeedbackSection(parseProduct productCard, parseComments []commentRecord, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(publicCommentFormState{Reaction: "up"})
		parsePendingCommentsState := useAtlasState([]commentRecord{})
		parseSubmittingState := useAtlasState(false)
		parseSubmissionMessageState := useAtlasState("")
		parseCommentsResource := useAtlasCachedResource(CachedRequestResourceKey("/api/public/products/"+parseProduct.Slug+"/comments", "items"), func(parseCtx context.Context) ([]commentRecord, error) {
			return fetchPublicProductComments(parseCtx, parseProduct.Slug)
		})
		parseResourceState := parseCommentsResource.Get()
		useAtlasEffect(func() func() {
			parseCommentsResource.Set(cloneCommentRecords(parseComments))
			parsePendingCommentsState.Set([]commentRecord{})
			return nil
		}, parseProduct.Slug, commentRecordsSignature(parseComments))
		parseValue := parseForm.Get()
		parseVisibleComments := cloneCommentRecords(parseComments)
		if parseResourceState.Ready {
			parseVisibleComments = cloneCommentRecords(parseResourceState.Value)
		}
		for _, parsePending := range parsePendingCommentsState.Get() {
			parseVisibleComments = mergePublicCommentList(parseVisibleComments, parsePending)
		}
		parseSubmitting := parseSubmittingState.Get()
		parseSubmissionMessage := parseSubmissionMessageState.Get()
		isParseRefreshing := parseResourceState.Loading && parseResourceState.Ready
		parseCountLabel := fmt.Sprintf("%d buyer notes", len(parseVisibleComments))
		if len(parseVisibleComments) == 1 {
			parseCountLabel = "1 buyer note"
		}
		setAuthorName := ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField("AuthorName", parseEvent.GetValue()) })
		setSubject := ui.UseEvent(func(parseEvent2 ui.InputEvent) { parseForm.SetField("Subject", parseEvent2.GetValue()) })
		setBody := ui.UseEvent(func(parseEvent3 ui.InputEvent) { parseForm.SetField("Body", parseEvent3.GetValue()) })
		setReaction := ui.UseEvent(func(parseEvent4 ui.ChangeEvent) { parseForm.SetField("Reaction", parseEvent4.GetValue()) })
		parseSubmit := ui.UseEvent(func(parseEvent5 ui.FormEvent) {
			parseEvent5.PreventDefault()
			if !parseForm.Validate(validatePublicCommentForm) {
				return
			}
			if parseSubmittingState.Get() {
				return
			}
			parseSubmittingState.Set(true)
			parseSubmissionMessageState.Set("")
			parseForm.SetFormError("")
			parseSnapshot := parseForm.Get()
			go func() {
				parseCreated, parseServerErrors, parseErr := submitPublicComment(parseProduct.Slug, parseSnapshot, parsePayload.CSRF)
				if parseErr != nil {
					parseForm.SetFormError("Comment submit failed. Retry in a moment.")
					parseSubmittingState.Set(false)
					return
				}
				if !parseForm.ApplyServerErrors(parseServerErrors) {
					parseSubmittingState.Set(false)
					return
				}
				parseForm.Reset(publicCommentFormState{Reaction: "up"})
				parseForm.SetErrors(nil)
				parseForm.SetFormError("")
				parseSubmissionMessageState.Set("Your comment was submitted for review.")
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Comment queued",
					Detail: "Atlas submitted the buyer comment and refreshed the thread without leaving the product route.",
					Tone:   "success",
				})
				parsePendingCommentsState.Set(mergePublicCommentList(parsePendingCommentsState.Get(), parseCreated))
				parseCommentsResource.Update(func(parseItems []commentRecord) []commentRecord {
					return mergePublicCommentList(parseItems, parseCreated)
				})
				parseCommentsResource.Reload()
				parseSubmittingState.Set(false)
			}()
		})
		return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))},
			html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3))},
				html.Div(html.Props{Class: design.Class(design.Stack(design.Space1))},
					html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Customer reviews and questions")),
					html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("What buyers are asking before they commit.")),
				),
				// The count is Data: it is a computed fact, and in mono it stays
				// readable as a number instead of reading as a sentence fragment.
				html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(parseCountLabel)),
			),
			html.Div(html.Props{Class: design.Class(design.Divider())}),
			publicProductFeedbackList(parseVisibleComments, isParseRefreshing),
			publicProductFeedbackForm(parseProduct, parsePayload, parseForm, parseValue, parseSubmitting, parseSubmissionMessage, setAuthorName, setSubject, setBody, setReaction, parseSubmit),
		)
	})
}

// publicProductFeedbackList is the sentiment read plus the notes themselves.
//
// The notes are ruled rows, not eight more bordered cards. The sentiment summary
// keeps one meter and one figure where it used to have a 3xl percentage, two
// tinted count cards and a paragraph explaining what a percentage is.
func publicProductFeedbackList(parseComments []commentRecord, isRefreshing bool) ui.Node {
	parseThumbsUp, parseThumbsDown := publicCommentReactionCounts(parseComments)
	parseRatioLabel, parseRatioValue := publicCommentRatioSummary(parseThumbsUp, parseThumbsDown)
	parseNodes := make([]ui.Node, 0, len(parseComments))
	for _, parseItem := range parseComments {
		parseNodes = append(parseNodes, html.Li(html.Props{Class: design.Class(atlasRuledRowBundle)},
			html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space2))},
				html.Span(html.Props{Class: design.Class(design.Prose(design.StepBase)), Style: map[string]string{"font-weight": "600"}}, html.Text(parseItem.Subject)),
				html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
					publicCommentReactionBadge(parseItem.Reaction),
					publicCommentStatusBadge(parseItem.Status),
				),
			),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseItem.Body)),
			html.Span(html.Props{Class: design.Class(design.Data(design.StepMicro))},
				html.Text(parseItem.AuthorName+" · "+strings.ToUpper(strings.ReplaceAll(parseItem.AuthorType, "_", " "))+" · "+formatPublicCommentDate(parseItem.CreatedAt)),
			),
		))
	}
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space2))},
			html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3))},
				html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Buyer sentiment")),
				html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(parseRatioLabel)),
			),
			html.Div(html.Props{Class: design.Class(atlasMeterTrackBundle)},
				html.Div(html.Props{
					Class: design.Class(atlasMeterFillBundle),
					Style: map[string]string{"width": fmt.Sprintf("%d%%", parseRatioValue)},
				}),
			),
			html.Span(html.Props{Class: design.Class(design.Data(design.StepMicro))},
				html.Text(fmt.Sprintf("%d UP · %d DOWN", parseThumbsUp, parseThumbsDown)),
			),
		),
	}
	if isRefreshing {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Recess(), design.Prose(design.StepFine))}, html.Text("Refreshing buyer notes...")),
		)
	}
	if len(parseNodes) == 0 {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())},
				html.Text("No buyer notes have been shared yet. The first approved question or review will appear here once Atlas has it."),
			),
		)
	} else {
		parseChildren = append(parseChildren,
			html.Ul(html.Props{Class: atlasRuledListClass(), Role: "list"}, parseNodes...),
		)
	}
	return html.Div(html.Props{Class: design.Class(design.Stack(design.Space3))}, parseChildren...)
}

func publicCommentReactionCounts(parseComments []commentRecord) (int, int) {
	parseThumbsUp := 0
	parseThumbsDown := 0
	for _, parseItem := range parseComments {
		if strings.TrimSpace(strings.ToLower(parseItem.Reaction)) == "down" {
			parseThumbsDown++
			continue
		}
		parseThumbsUp++
	}
	return parseThumbsUp, parseThumbsDown
}

func publicCommentRatioSummary(parseThumbsUp int, parseThumbsDown int) (string, int) {
	parseTotal := parseThumbsUp + parseThumbsDown
	if parseTotal == 0 {
		return "No ratings yet", 0
	}
	parsePercentage := int(float64(parseThumbsUp)*100/float64(parseTotal) + 0.5)
	return fmt.Sprintf("%d%% thumbs up", parsePercentage), parsePercentage
}

// publicCommentReactionBadge is a StatusChip driven by a tone, not a hue.
//
// Thumbs up is ToneVerified and thumbs down is ToneNeutral rather than
// ToneException, on the same argument design.Availability makes about
// out-of-stock: nothing has gone wrong because a buyer disliked a finish, and
// oxide has to stay rare to stay loud. The WORDS carry the difference.
func publicCommentReactionBadge(parseReaction string) ui.Node {
	if strings.TrimSpace(strings.ToLower(parseReaction)) == "down" {
		return html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneNeutral))}, html.Text("Thumbs down"))
	}
	return html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneVerified))}, html.Text("Thumbs up"))
}

type publicCommentFormState struct {
	AuthorName string
	Reaction   string
	Subject    string
	Body       string
}

func validatePublicCommentForm(parseValue publicCommentFormState) ui.FieldErrors {
	parseErrors := ui.FieldErrors{}
	if strings.TrimSpace(parseValue.AuthorName) == "" {
		parseErrors["AuthorName"] = "Enter your name before sharing feedback."
	}
	parseReaction := strings.TrimSpace(strings.ToLower(parseValue.Reaction))
	if parseReaction != "up" && parseReaction != "down" {
		parseErrors["Reaction"] = "Choose thumbs up or thumbs down."
	}
	if strings.TrimSpace(parseValue.Subject) == "" {
		parseErrors["Subject"] = "Add a short headline for your review or question."
	}
	if len(strings.TrimSpace(parseValue.Body)) < 8 {
		parseErrors["Body"] = "Share a more specific note so other buyers get useful context."
	}
	return parseErrors
}

// publicProductFeedbackForm is a COMPONENT: it calls ui.UseId four times.
//
// As a plain helper those four id slots were drawn from publicProductFeedbackSection's
// fiber, interleaved with that component's UseForm/UseState/UseEvent slots — and it is
// itself built inside an atlasLazySection loader, so on the browser path there was no
// fiber to draw them from at all. Owning a fiber makes the four ids stable and local.
//
// The field ids, name attributes and aria wiring below are byte-identical to what
// shipped: browser specs assert the `name` values, and one of them broke earlier
// today. Only the classes changed.
func publicProductFeedbackForm(parseProduct productCard, parsePayload Payload, parseForm ui.Form[publicCommentFormState], parseValue publicCommentFormState, isSubmitting bool, parseSubmissionMessage string, setAuthorName ui.Handler, setSubject ui.Handler, setBody ui.Handler, setReaction ui.Handler, parseSubmit ui.Handler) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseAuthorInputID := ui.UseId()
		parseAuthorErrorID := parseAuthorInputID + "-error"
		parseReactionFieldID := ui.UseId()
		parseReactionErrorID := parseReactionFieldID + "-error"
		parseSubjectInputID := ui.UseId()
		parseSubjectErrorID := parseSubjectInputID + "-error"
		parseBodyInputID := ui.UseId()
		parseBodyErrorID := parseBodyInputID + "-error"
		parseChildren := []ui.Node{
			html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Share your review or question")),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text("Leave a review or ask a buying question here and it stays attached to this product.")),
		}
		if strings.TrimSpace(parseSubmissionMessage) != "" {
			parseChildren = append(parseChildren, html.P(html.Props{Class: design.Class(design.Recess(), design.Prose(design.StepFine))}, html.Text(parseSubmissionMessage)))
		}
		if strings.TrimSpace(parseForm.FormError()) != "" {
			parseChildren = append(parseChildren, html.P(html.Props{Class: design.Class(design.FieldError())}, html.Text(parseForm.FormError())))
		}
		parseChildren = append(parseChildren, prependCSRFToken(parsePayload.CSRF)...)
		parseChildren = append(parseChildren,
			html.Label(html.Props{Class: design.Class(design.Field())},
				html.Span(html.Props{ID: parseAuthorInputID + "-label", Class: design.Class(design.FieldLabel())}, html.Text("Name")),
				html.Input(html.Props{ID: parseAuthorInputID, Name: "author_name", Value: parseValue.AuthorName, AutoComplete: "name", OnInput: setAuthorName, Class: publicCommentFieldClass(parseForm.Error("AuthorName") != ""), Raw: map[string]any{"aria-labelledby": parseAuthorInputID + "-label", "aria-describedby": parseAuthorErrorID, "aria-invalid": parseForm.Error("AuthorName") != ""}}),
				publicCommentFieldError(parseAuthorErrorID, parseForm.Error("AuthorName")),
			),
			publicCommentReactionInput(parseReactionFieldID, parseReactionErrorID, parseValue.Reaction, setReaction, parseForm.Error("Reaction")),
			html.Label(html.Props{Class: design.Class(design.Field())},
				html.Span(html.Props{ID: parseSubjectInputID + "-label", Class: design.Class(design.FieldLabel())}, html.Text("Headline")),
				html.Input(html.Props{ID: parseSubjectInputID, Name: "subject", Value: parseValue.Subject, OnInput: setSubject, Class: publicCommentFieldClass(parseForm.Error("Subject") != ""), Raw: map[string]any{"aria-labelledby": parseSubjectInputID + "-label", "aria-describedby": parseSubjectErrorID, "aria-invalid": parseForm.Error("Subject") != ""}}),
				publicCommentFieldError(parseSubjectErrorID, parseForm.Error("Subject")),
			),
			html.Label(html.Props{Class: design.Class(design.Field())},
				html.Span(html.Props{ID: parseBodyInputID + "-label", Class: design.Class(design.FieldLabel())}, html.Text("Comment")),
				html.Textarea(html.Props{ID: parseBodyInputID, Name: "body", OnInput: setBody, Class: publicCommentTextareaClass(parseForm.Error("Body") != ""), Raw: map[string]any{"aria-labelledby": parseBodyInputID + "-label", "aria-describedby": parseBodyErrorID, "aria-invalid": parseForm.Error("Body") != ""}}, html.Text(parseValue.Body)),
				publicCommentFieldError(parseBodyErrorID, parseForm.Error("Body")),
			),
			html.Button(html.Props{Type: "submit", Disabled: isSubmitting, Class: publicCommentSubmitClass(isSubmitting)}, html.Text(publicCommentSubmitLabel(isSubmitting))),
		)
		return html.Form(html.Props{Action: "/api/public/products/" + parseProduct.Slug + "/comments", Method: "post", OnSubmit: parseSubmit, Class: design.Class(design.Recess(), design.Stack(design.Space3))}, parseChildren...)
	})
}

func cloneCommentRecords(parseItems []commentRecord) []commentRecord {
	if len(parseItems) == 0 {
		return []commentRecord{}
	}
	parseCloned := make([]commentRecord, len(parseItems))
	copy(parseCloned, parseItems)
	return parseCloned
}

func commentRecordsSignature(parseItems []commentRecord) string {
	if len(parseItems) == 0 {
		return ""
	}
	parseParts := make([]string, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseParts = append(parseParts, parseItem.ID+":"+parseItem.Status+":"+parseItem.UpdatedAt)
	}
	return strings.Join(parseParts, "|")
}

func submitPublicComment(parseSlug string, parseInput publicCommentFormState, parseCsrfToken string) (commentRecord, ui.ServerFormErrors, error) {
	parseBody := map[string]string{
		"author_name": parseInput.AuthorName,
		"reaction":    parseInput.Reaction,
		"subject":     parseInput.Subject,
		"body":        parseInput.Body,
	}
	parseHeaderName, parseHeaderValue := ui.NewCSRFToken(parseCsrfToken).Header()
	parseResult := <-atlasFetch("/api/public/products/"+parseSlug+"/comments", atlasFetchOptions{
		Method: http.MethodPost,
		Headers: map[string]any{
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			parseHeaderName: parseHeaderValue,
		},
		Body: parseBody,
	})
	if strings.TrimSpace(parseResult.Error) != "" {
		return commentRecord{}, ui.ServerFormErrors{}, fmt.Errorf("%s", parseResult.Error)
	}
	if parseResult.Status >= 200 && parseResult.Status < 300 {
		var parseCreated commentRecord
		if parseErr := json.Unmarshal([]byte(parseResult.Data), &parseCreated); parseErr != nil {
			return commentRecord{}, ui.ServerFormErrors{}, parseErr
		}
		return parseCreated, ui.ServerFormErrors{}, nil
	}
	var parseFailure ui.ServerFormErrors
	if parseErr2 := json.Unmarshal([]byte(parseResult.Data), &parseFailure); parseErr2 != nil {
		return commentRecord{}, ui.ServerFormErrors{}, parseErr2
	}
	parseFailure.Fields = normalizePublicCommentFieldErrors(parseFailure.Fields)
	if parseFailure.FormMessage() == "" {
		parseFailure.Message = "Fix the highlighted fields and try again."
	}
	return commentRecord{}, parseFailure, nil
}

func mergePublicCommentList(parseItems []commentRecord, parsePendingComment commentRecord) []commentRecord {
	parseMerged := cloneCommentRecords(parseItems)
	if strings.TrimSpace(parsePendingComment.ID) == "" {
		return parseMerged
	}
	for _, parseItem := range parseMerged {
		if parseItem.ID == parsePendingComment.ID {
			return parseMerged
		}
	}
	return append([]commentRecord{parsePendingComment}, parseMerged...)
}

// publicCommentStatusBadge shows a moderation state, and only when there is one:
// an approved note needs no badge, because "approved" is the default a reader
// already assumes. Pending is TonePending; flagged and rejected are the one
// moderation case that genuinely needs a human, so they get ToneException.
func publicCommentStatusBadge(parseStatus string) ui.Node {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseStatus))
	if parseTrimmed == "" || parseTrimmed == "approved" {
		return nil
	}
	parseTone := design.ToneNeutral
	switch parseTrimmed {
	case "pending":
		parseTone = design.TonePending
	case "flagged", "rejected":
		parseTone = design.ToneException
	}
	return html.Span(html.Props{Class: design.Class(design.StatusChip(parseTone))}, html.Text(strings.ReplaceAll(parseTrimmed, "_", " ")))
}

func fetchPublicProductComments(parseCtx context.Context, parseSlug string) ([]commentRecord, error) {
	parsePayload, parseErr := fetchAtlasJSON[struct {
		Items []commentRecord `json:"items"`
	}](parseCtx, "/api/public/products/"+parseSlug+"/comments")
	if parseErr != nil {
		return nil, parseErr
	}
	return cloneCommentRecords(parsePayload.Items), nil
}

func fetchPublicRelatedProducts(parseCtx context.Context, parseSlug string) ([]relatedProductRecord, error) {
	parsePayload, parseErr := fetchAtlasJSON[struct {
		Items []relatedProductRecord `json:"items"`
	}](parseCtx, "/api/public/products/"+parseSlug+"/related-products")
	if parseErr != nil {
		return nil, parseErr
	}
	parseItems := make([]relatedProductRecord, len(parsePayload.Items))
	copy(parseItems, parsePayload.Items)
	return parseItems, nil
}

func normalizePublicCommentFieldErrors(parseFields ui.FieldErrors) ui.FieldErrors {
	if len(parseFields) == 0 {
		return nil
	}
	parseNormalized := ui.FieldErrors{}
	for parseKey, parseValue := range parseFields {
		switch strings.TrimSpace(strings.ToLower(parseKey)) {
		case "author_name":
			parseNormalized["AuthorName"] = parseValue
		case "reaction":
			parseNormalized["Reaction"] = parseValue
		case "subject":
			parseNormalized["Subject"] = parseValue
		case "body":
			parseNormalized["Body"] = parseValue
		default:
			parseNormalized[parseKey] = parseValue
		}
	}
	return parseNormalized
}

// publicCommentSubmitBusyBundle is the in-flight look for the submit button:
// wait cursor and reduced opacity over ButtonPrimary. No second button style —
// the design system has exactly three buttons and a fourth "sending" variant
// would be a fourth.
var publicCommentSubmitBusyBundle = css.Rules(
	css.Cursor.Wait,
	css.OpacityNum(css.Num(0.7)),
)

func publicCommentSubmitClass(isSubmitting bool) string {
	if isSubmitting {
		return design.Class(design.ButtonPrimary(), publicCommentSubmitBusyBundle)
	}
	return design.Class(design.ButtonPrimary())
}

func publicCommentSubmitLabel(isSubmitting bool) string {
	if isSubmitting {
		return "Sending..."
	}
	return "Share feedback"
}

// publicCommentReactionInput keeps its fieldset/legend/radio structure, its ids
// and its `name="reaction"` values exactly as they were — a browser spec drives
// these controls by name — and drops the two tinted, bordered, radiused label
// boxes. A radio with a label beside it does not need a card around it.
func publicCommentReactionInput(parseFieldID string, parseErrorID string, parseSelected string, parseHandler ui.Handler, parseErrorText string) ui.Node {
	if strings.TrimSpace(parseSelected) == "" {
		parseSelected = "up"
	}
	return html.Fieldset(html.Props{Class: design.Class(design.Field()), Raw: map[string]any{"aria-describedby": parseErrorID, "aria-invalid": strings.TrimSpace(parseErrorText) != ""}},
		html.Legend(html.Props{ID: parseFieldID + "-legend", Class: design.Class(design.FieldLabel())}, html.Text("Your reaction")),
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space2))},
			publicCommentReactionChoice(parseFieldID, parseErrorID, "up", "Thumbs up", "Recommend it or confirm the setup met expectations.", parseSelected, parseHandler),
			publicCommentReactionChoice(parseFieldID, parseErrorID, "down", "Thumbs down", "Call out delivery friction, finish issues, or fit concerns buyers should know.", parseSelected, parseHandler),
		),
		publicCommentFieldError(parseErrorID, parseErrorText),
	)
}

func publicCommentReactionChoice(parseFieldID string, parseErrorID string, parseValue string, parseLabel string, parseHint string, parseSelected string, parseHandler ui.Handler) ui.Node {
	return html.Label(html.Props{Class: design.Class(design.Cluster(design.Space2)), Style: map[string]string{"align-items": "start", "cursor": "pointer"}},
		html.Input(html.Props{ID: parseFieldID + "-" + parseValue, Type: "radio", Name: "reaction", Value: parseValue, Checked: strings.TrimSpace(parseSelected) == parseValue, OnChange: parseHandler, Raw: map[string]any{"aria-labelledby": parseFieldID + "-legend", "aria-describedby": parseErrorID}}),
		html.Span(html.Props{Class: design.Class(design.Stack(design.Space1))},
			html.Span(html.Props{Class: design.Class(design.Prose(design.StepFine)), Style: map[string]string{"font-weight": "600"}}, html.Text(parseLabel)),
			html.Span(html.Props{Class: design.Class(design.FieldHint())}, html.Text(parseHint)),
		),
	)
}

// publicCommentFieldErrorBundle marks an invalid control by re-drawing its border
// in the exception tone. Border, not background: an oxide-filled input is
// unreadable in dark mode and it is the same "paint the whole thing" move the
// design system removed everywhere else.
var publicCommentFieldErrorBundle = css.Rules(
	css.Border(design.HairlineWidth, design.StatusException()),
)

// publicCommentTextareaBundle is the one thing a textarea needs that an input
// does not: room for more than one line. Same primitive otherwise, so the two
// controls cannot drift apart.
var publicCommentTextareaBundle = css.Rules(
	css.MinHeight(css.RawLength("7rem")),
)

func publicCommentFieldClass(hasError bool) string {
	if hasError {
		return design.Class(design.Input(), publicCommentFieldErrorBundle)
	}
	return design.Class(design.Input())
}

func publicCommentTextareaClass(hasError bool) string {
	if hasError {
		return design.Class(design.Input(), publicCommentTextareaBundle, publicCommentFieldErrorBundle)
	}
	return design.Class(design.Input(), publicCommentTextareaBundle)
}

func publicCommentFieldError(parseId string, parseMessage string) ui.Node {
	if strings.TrimSpace(parseMessage) == "" {
		return nil
	}
	return html.P(html.Props{ID: parseId, Class: design.Class(design.FieldError())}, html.Text(parseMessage))
}

func formatPublicCommentDate(parseValue string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return "recently"
	}
	if len(parseTrimmed) >= 10 {
		return parseTrimmed[:10]
	}
	return parseTrimmed
}

// publicRelatedProductsCard keeps its cached-resource behaviour and its copy —
// two browser-facing tests read the cache-reuse sentence as evidence the second
// open did not refetch — and becomes one surface with ruled rows.
func publicRelatedProductsCard(parseProduct productCard) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseResource := useAtlasCachedResource(CachedRequestResourceKey("/api/public/products/"+parseProduct.Slug+"/related-products", "items"), func(parseCtx context.Context) ([]relatedProductRecord, error) {
			return fetchPublicRelatedProducts(parseCtx, parseProduct.Slug)
		})
		parseState := parseResource.Get()
		parseItems := []relatedProductRecord{}
		if parseState.Ready {
			parseItems = parseState.Value
		}
		parseDescription := "Open adjacent Atlas systems without leaving the same buying context."
		if parseState.Loading && !parseState.Ready {
			parseDescription = "Loading adjacent systems for this product..."
		} else if parseState.Error != nil && !parseState.Ready {
			parseDescription = "Atlas could not load related systems right now. Reopen the route to retry."
		}
		parseChildren := []ui.Node{
			html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Related systems")),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseDescription)),
		}
		parseChildren = append(parseChildren, publicRelatedProductNodes(parseItems)...)
		return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))}, parseChildren...)
	})
}

func publicRelatedProductNodes(parseItems []relatedProductRecord) []ui.Node {
	if len(parseItems) == 0 {
		return []ui.Node{
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())},
				html.Text("Repeat-open visits can reuse the cached related-product list after the first lookup, so Atlas does not need to rebuild this secondary panel every time."),
			),
		}
	}
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Li(html.Props{},
			html.A(html.Props{Href: RouteCatalog + "/" + parseItem.Slug, Class: atlasRuledRowClass()},
				html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space2))},
					html.Span(html.Props{Class: design.Class(design.Prose(design.StepFine)), Style: map[string]string{"font-weight": "600"}}, html.Text(parseItem.Title)),
					html.Span(html.Props{Class: design.Class(design.Data(design.StepMicro))}, html.Text(atlasSKUCode(parseItem.SKU))),
				),
				html.Span(html.Props{Class: design.Class(design.Prose(design.StepFine))}, html.Text(fallback(parseItem.Reason, parseItem.Summary))),
				html.Span(html.Props{Class: design.Class(design.Data(design.StepMicro))}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
			),
		))
	}
	return []ui.Node{html.Ul(html.Props{Class: atlasRuledListClass(), Role: "list"}, parseRows...)}
}

// publicSupportPoints returns <li> items now, so its caller can put them in a
// real list. Each point used to be a bordered pill; a list of short sentences is
// a list.
func publicSupportPoints(parsePoints []string) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parsePoints))
	for _, parsePoint := range parsePoints {
		parseNodes = append(parseNodes, html.Li(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parsePoint)))
	}
	return parseNodes
}

// =============================================================================
// WAREHOUSE DIRECTORY (/warehouses)
// =============================================================================

// renderWarehouseDirectoryContent is one surface holding a ruled list of hubs.
//
// What came off, and why: the overview band carried FOUR stat cards, two of
// which ("%d stocked units", "%d inbound units") summed fields that
// /api/public/warehouses does not return — so in production they printed "0
// stocked units" under a heading claiming a merchandised network. A number that
// is always zero is worse than no number, because a reader believes it. The two
// duplicate call-to-action buttons also came off: the route hero already offers
// the catalog, and every row already links to its hub.
func renderWarehouseDirectoryContent(parsePage warehouseDirectoryPage) ui.Node {
	return html.Section(html.Props{Class: design.Class(design.Stack(design.Space5))},
		publicWarehouseDirectoryOverview(parsePage),
	)
}

func publicWarehouseDirectoryOverview(parsePage warehouseDirectoryPage) ui.Node {
	parseChildren := []ui.Node{
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Atlas delivery regions")),
		html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Pick the hub that serves your site")),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())},
			html.Text("Each hub publishes its own delivery window. Open one to see what it holds for a product."),
		),
	}
	if len(parsePage.Items) == 0 {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())},
				html.Text("No delivery regions are available right now. Retry the page or return to the storefront."),
			),
		)
		return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))}, parseChildren...)
	}
	parseRows := make([]ui.Node, 0, len(parsePage.Items))
	for _, parseItem := range parsePage.Items {
		parseRows = append(parseRows, publicWarehouseDirectoryCard(parseItem))
	}
	parseChildren = append(parseChildren,
		html.Ul(html.Props{Class: atlasRuledListClass(), Role: "list"}, parseRows...),
	)
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))}, parseChildren...)
}

// publicWarehouseDirectoryCard is one hub row.
//
// It used to print the hub name THREE times (a badge, a heading, and a metric
// labelled "Region" whose value was the region) plus six metric boxes, three of
// which were zero-valued or fallback strings. Now: name, hub code, the real
// delivery window, the hub's own summary, and one action.
func publicWarehouseDirectoryCard(parseItem warehouseCard) ui.Node {
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3))},
			html.Span(html.Props{Class: design.Class(design.Prose(design.StepBase)), Style: map[string]string{"font-weight": "600"}}, html.Text(parseItem.Name)),
			html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(atlasLineHub(parseItem.ID))),
		),
	}
	if parseSummary := strings.TrimSpace(parseItem.PublicSummary); parseSummary != "" {
		parseChildren = append(parseChildren,
			html.Span(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseSummary)),
		)
	}
	parseMeta := make([]ui.Node, 0, 2)
	if parseWindow := atlasPromiseWindow(parseItem.ServiceLevel); parseWindow != "" {
		parseMeta = append(parseMeta, html.Span(html.Props{}, html.Text(parseWindow)))
	}
	if parseRegion := strings.TrimSpace(parseItem.Region); parseRegion != "" {
		if len(parseMeta) > 0 {
			parseMeta = append(parseMeta,
				html.Span(html.Props{Aria: map[string]string{"hidden": "true"}}, html.Text("·")))
		}
		parseMeta = append(parseMeta, html.Span(html.Props{}, html.Text(strings.ToUpper(parseRegion))))
	}
	if len(parseMeta) > 0 {
		parseChildren = append(parseChildren,
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2), design.Data(design.StepMicro))}, parseMeta...),
		)
	}
	parseChildren = append(parseChildren,
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Open warehouse route")),
	)
	return html.Li(html.Props{},
		html.A(html.Props{Href: RouteWarehouses + "/" + parseItem.Slug, Class: atlasRuledRowClass()}, parseChildren...),
	)
}

// publicWarehouseDirectoryMetric is a label/value line, used where a hub really
// does publish a value. Not a box: a SplitRow with a Data value on the right
// lines up down the page for free.
func publicWarehouseDirectoryMetric(parseLabel string, parseValue string) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3))},
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseLabel)),
		html.Span(html.Props{Class: design.Class(design.Data(design.StepFine))}, html.Text(parseValue)),
	)
}

// primaryWarehouseServiceLevel, publicWarehouseUnitsTotal and
// publicWarehouseInboundTotal were deleted with the directory's stat block.
//
// The two totals summed warehouseCard.Available and .Inbound across the hubs, and
// store.Warehouses selects id, slug, name, region, service_level and public_summary
// — so on the live storefront they always summed zero and the directory advertised
// "0 stocked units" and "0 inbound units". The third picked the FIRST hub's service
// level and printed it as if it described the network. All three are gone rather
// than fixed here, because the fix is a payload that carries stock, not a caller
// that averages what it does not have.

// =============================================================================
// WAREHOUSE DETAIL (/warehouses/{slug})
// =============================================================================

func renderWarehouseDetailContent(parsePage warehouseDetailPage, parsePayload Payload) ui.Node {
	parseWarehouse := parsePage.Warehouse
	return html.Section(html.Props{Class: atlasBuyingPageClass()},
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space5))},
			publicWarehouseDetailHero(parseWarehouse),
			publicWarehouseRegionalProductShowcase(parseWarehouse, parsePage.Products),
		),
		publicWarehouseSideDataCard(parsePage, parsePayload),
	)
}

// publicWarehouseDetailHero is the hub's one surface.
//
// The old version was a gradient panel with two blurred glow divs, three chips,
// two "context columns" whose copy came from a switch that never matched the real
// region strings (so it always printed the same fallback paragraph), three metric
// cards repeating the region and service level a third time, and two buttons. It
// is now the hub's name, its window, its own summary, and one action — plus a
// three-card feature strip that has been deleted outright, because "Regional
// pages stay calm and readable instead of collapsing into shipping jargon" is the
// page grading itself.
func publicWarehouseDetailHero(parseWarehouse warehouseCard) ui.Node {
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2), design.Data(design.StepFine))},
			html.Span(html.Props{}, html.Text(atlasLineHub(parseWarehouse.ID))),
			html.Span(html.Props{Aria: map[string]string{"hidden": "true"}}, html.Text("·")),
			html.Span(html.Props{}, html.Text(publicRegionalHubLabel)),
		),
		html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseWarehouse.Name)),
	}
	if parseSummary := strings.TrimSpace(parseWarehouse.PublicSummary); parseSummary != "" {
		parseChildren = append(parseChildren,
			html.P(html.Props{Class: design.Class(design.Prose(design.StepLede), design.Measure())}, html.Text(parseSummary)),
		)
	}
	parseChildren = append(parseChildren, html.Div(html.Props{Class: design.Class(design.ManifestRule())}))
	if parseWindow := atlasPromiseWindow(parseWarehouse.ServiceLevel); parseWindow != "" {
		parseChildren = append(parseChildren, publicWarehouseDirectoryMetric(publicServiceLevelLabel, parseWindow))
	}
	if parseRegion := strings.TrimSpace(parseWarehouse.Region); parseRegion != "" {
		parseChildren = append(parseChildren, publicWarehouseDirectoryMetric(publicRegionalFocusLabel, parseRegion))
	}
	if parseFocus := strings.TrimSpace(parseWarehouse.Focus); parseFocus != "" {
		parseChildren = append(parseChildren, publicProductContextColumn(publicPromiseLensLabel, parseFocus))
	}
	parseChildren = append(parseChildren,
		html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
			html.A(html.Props{
				Href:  RouteCatalog + "?warehouse=" + url.QueryEscape(parseWarehouse.ID),
				Class: design.Class(design.ButtonPrimary()),
			}, html.Text(publicBrowseRegionalProductsLabel)),
		),
	)
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))}, parseChildren...)
}

// publicWarehouseSideDataCard keeps its startup-resource behaviour and the
// cache-reuse sentence a contract test reads as evidence of it.
//
// The five metric cards under it are gone. Pressure, staffing, backlog and
// regional focus are OPERATOR fields, and store.WarehouseBySlug does not select
// them for the public route — so every one of them rendered its "Core team
// assigned" style fallback, on a buyer-facing page, forever.
func publicWarehouseSideDataCard(parsePage warehouseDetailPage, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseResource := useAtlasStartupPageResource(parsePayload)
		parseState := parseResource.Get()
		parseCurrent := parsePage
		if parseState.Ready {
			parseDecoded := decode[warehouseDetailPage](parseState.Value)
			if strings.TrimSpace(parseDecoded.Warehouse.ID) != "" {
				parseCurrent = parseDecoded
			}
		}
		parseWarehouse := parseCurrent.Warehouse
		parseStatusCopy := "Repeat-open visits can reuse this side snapshot without waiting for the whole warehouse route to rebuild."
		if parseState.Loading && parseState.Ready {
			parseStatusCopy = "Refreshing the latest warehouse posture..."
		} else if parseState.Error != nil && !parseState.Ready {
			parseStatusCopy = "Showing the SSR warehouse snapshot until Atlas can reload the side data."
		}
		parseChildren := []ui.Node{
			html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Warehouse side data")),
			html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(fallback(parseWarehouse.Name, "Regional hub posture"))),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepFine), design.Measure())}, html.Text(parseStatusCopy)),
		}
		if parseWindow := atlasPromiseWindow(parseWarehouse.ServiceLevel); parseWindow != "" {
			parseChildren = append(parseChildren, html.Div(html.Props{Class: design.Class(design.Divider())}))
			parseChildren = append(parseChildren, publicWarehouseDirectoryMetric(publicServiceLevelLabel, parseWindow))
		}
		if parseCount := len(parseCurrent.Products); parseCount > 0 {
			parseChildren = append(parseChildren, publicWarehouseDirectoryMetric("Stocked lines", fmt.Sprintf("%d", parseCount)))
		}
		return html.Div(html.Props{Class: atlasBuyingRailClass()},
			html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))}, parseChildren...),
		)
	})
}

// publicWarehouseRegionalProductShowcase is the hub's product list — and it is
// the same manifest primitive /shop uses, on purpose.
//
// It used to be a fourth card style: bordered product cards inside a bordered
// showcase panel inside the page. Reusing design.Catalog means the buyer reads
// the same row shape, the same column order and the same availability column
// they just read on /shop, and the only difference is where the row goes: each
// href is this hub's availability route for that product.
func publicWarehouseRegionalProductShowcase(parseWarehouse warehouseCard, parseProducts []productCard) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.Stack(design.Space3))},
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space1))},
			html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(publicRegionalAvailabilityPicksLabel)),
			html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text("Open stocked systems that fit this regional route.")),
		),
		html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
			design.Catalog(design.CatalogSpec{
				Items:      atlasWarehouseCatalogLines(parseWarehouse, parseProducts),
				Label:      "Products stocked at " + parseWarehouse.Name,
				EmptyTitle: "No stocked lines at this hub",
				EmptyBody:  "This hub has nothing on the manifest right now. Compare another region or browse the full catalog.",
				// The recovery action is the catalog, because the buyer's problem
				// here is "this hub cannot help me", not "my filter is too narrow".
				EmptyActionLabel: "Browse the catalog",
				EmptyActionHref:  RouteCatalog,
			}),
		),
	)
}

// atlasWarehouseCatalogLines builds manifest lines that point at this hub's
// availability route rather than at the product page, and label the action
// accordingly. The hub column is this hub, which is the one fact this list adds
// over the storefront catalog.
func atlasWarehouseCatalogLines(parseWarehouse warehouseCard, parseProducts []productCard) []design.CatalogItem {
	parseLines := make([]design.CatalogItem, 0, len(parseProducts))
	for _, parseItem := range parseProducts {
		parseLines = append(parseLines, design.CatalogItem{
			Href:        RouteWarehouses + "/" + parseWarehouse.Slug + "/availability/" + parseItem.Slug,
			Title:       parseItem.Title,
			SKU:         atlasSKUCode(parseItem.SKU),
			Category:    parseItem.Category,
			Price:       formatPrice(parseItem.PriceCents),
			Summary:     parseItem.Summary,
			Avail:       atlasProductAvailability(parseItem),
			Hub:         atlasLineHub(parseWarehouse.ID),
			Promise:     atlasPromiseWindow(parseWarehouse.ServiceLevel),
			ActionLabel: "AVAILABILITY",
			Label:       parseItem.Title + ", " + formatPrice(parseItem.PriceCents) + ", availability at " + parseWarehouse.Name,
		})
	}
	return parseLines
}

// =============================================================================
// PRODUCT AVAILABILITY (/warehouses/{slug}/availability/{productSlug})
// =============================================================================
//
// This is the lane placard's home surface. It is the one public route where the
// payload carries real counts (store.Availability returns available, inbound and
// a status), so it is the one public route that can honestly print a posture.

func renderAvailabilityContent(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	return html.Section(html.Props{Class: atlasBuyingPageClass()},
		html.Div(html.Props{Class: design.Class(design.Stack(design.Space5))},
			publicAvailabilityHero(parseAvailability),
			publicAvailabilityPromiseBand(parseAvailability),
		),
		publicAvailabilityActionRail(parseAvailability, parsePayload),
	)
}

// publicAvailabilityHero leads with the lane placard.
//
// The placard is the design system's one heavy signature device and this is what
// it is for: ORIGIN -> DEST, the promise, the posture, on an inked perforated
// dock tag. The origin is the hub; the destination is the buyer (see
// atlasBuyerDestination for why that is the honest label when Atlas does not know
// the buyer's address); the promise is the hub's published window; and the
// posture comes from the real counts, so PostureShort — the only oxide on the
// storefront — appears exactly when the promise cannot be met.
//
// Everything under it is the two numbers that back the placard, in Data so they
// align, and one sentence. What came off: two "context columns" of generated
// filler, three metric cards restating the same two numbers, and the
// "Why this availability page is easier to use" strip.
func publicAvailabilityHero(parseAvailability availabilityPage) ui.Node {
	parseAvail := atlasStockAvailability(parseAvailability.Available, parseAvailability.Inbound)
	return html.Div(html.Props{Class: design.Class(design.Stack(design.Space4))},
		// The placard sits directly on Paper, not inside the Surface below it: its
		// punched hole and torn edge are painted in the paper token, so on any other
		// background they read a shade off.
		design.LanePlacard(design.PlacardSpec{
			OriginHub: atlasLineHub(parseAvailability.Warehouse.ID),
			DestHub:   atlasBuyerDestination,
			Promise:   atlasPromiseWindow(parseAvailability.Warehouse.ServiceLevel),
			Posture:   atlasStockPosture(parseAvailability.Available, parseAvailability.Inbound),
		}),
		html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))},
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2), design.Data(design.StepFine))},
				html.Span(html.Props{}, html.Text(atlasSKUCode(parseAvailability.Product.SKU))),
				html.Span(html.Props{Aria: map[string]string{"hidden": "true"}}, html.Text("·")),
				html.Span(html.Props{}, html.Text(atlasLineHub(parseAvailability.Warehouse.ID))),
			),
			html.H2(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseAvailability.Product.Title)),
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
				html.Span(html.Props{Class: design.Class(design.StatusChip(parseAvail.Tone()))}, html.Text(parseAvail.Label())),
				html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneNeutral))}, html.Text(publicStatusLabel(parseAvailability.Status))),
			),
			html.Div(html.Props{Class: design.Class(design.ManifestRule())}),
			publicWarehouseDirectoryMetric("On hand", fmt.Sprintf("%d", parseAvailability.Available)),
			publicWarehouseDirectoryMetric("Inbound", fmt.Sprintf("%d", parseAvailability.Inbound)),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())},
				html.Text(fmt.Sprintf("%d on hand and %d inbound at %s.", parseAvailability.Available, parseAvailability.Inbound, parseAvailability.Warehouse.Name)),
			),
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))},
				html.A(html.Props{Href: RouteCatalog + "/" + parseAvailability.Product.Slug, Class: design.Class(design.ButtonSecondary())}, html.Text("Back to product detail")),
				html.A(html.Props{Href: RouteWarehouses + "/" + parseAvailability.Warehouse.Slug, Class: design.Class(design.ButtonQuiet())}, html.Text("See warehouse route")),
			),
		),
	)
}

// publicAvailabilityActionRail is the capture column. The two forms come from
// page.go untouched — their field names are asserted by a browser spec — and the
// duplicate "keep warehouse context visible" card came off, because the hero
// already links both routes it offered.
func publicAvailabilityActionRail(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	parseAvailabilityTitle, parseAvailabilityCopy := availabilitySupportPlan(parseAvailability.Available, parseAvailability.Inbound)
	return html.Div(html.Props{Class: atlasBuyingRailClass()},
		html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))},
			html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(publicRegionalNextStepLabel)),
			html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(parseAvailabilityTitle)),
			html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())}, html.Text(parseAvailabilityCopy)),
			html.Ul(html.Props{Class: design.Class(design.Stack(design.Space2)), Role: "list"}, publicSupportPoints(publicAvailabilitySupportPoints(parseAvailability))...),
		),
		availabilityPrimaryActionForm(parseAvailability, parsePayload),
		availabilityQuestionActionForm(parseAvailability, parsePayload),
	)
}

// publicAvailabilityPromiseBand is what this hub can say about this product.
//
// It keeps the hub's own words (Focus and PublicSummary are real, editor-written
// fields) and replaces three metric cards of generated cue copy with the two
// facts a buyer can act on: the delivery window and the region.
func publicAvailabilityPromiseBand(parseAvailability availabilityPage) ui.Node {
	parseChildren := []ui.Node{
		html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text("Warehouse-specific promise")),
		html.H3(html.Props{Class: design.Class(design.SectionTitle())}, html.Text(fallback(parseAvailability.Warehouse.Focus, "Regional delivery posture"))),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepBase), design.Measure())},
			html.Text(fallback(parseAvailability.Warehouse.PublicSummary, "This route should explain what this warehouse can realistically support for this product before the buyer submits follow-up.")),
		),
		html.Div(html.Props{Class: design.Class(design.Divider())}),
	}
	if parseWindow := atlasPromiseWindow(parseAvailability.Warehouse.ServiceLevel); parseWindow != "" {
		parseChildren = append(parseChildren, publicWarehouseDirectoryMetric(publicServicePostureLabel, parseWindow))
	}
	if parseRegion := strings.TrimSpace(parseAvailability.Warehouse.Region); parseRegion != "" {
		parseChildren = append(parseChildren, publicWarehouseDirectoryMetric(publicRegionalReadLabel, parseRegion))
	}
	parseChildren = append(parseChildren,
		publicProductContextColumn(publicPromiseLensLabel, availabilityStoryCopy(parseAvailability.Available, parseAvailability.Inbound)),
	)
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space3))}, parseChildren...)
}

// publicAvailabilitySupportPoints is the capture column's bullet list.
//
// It used to route two of its three points through derived_state.go's
// warehouseRegionCue and warehouseServiceTone, which switch on short region
// names ("west", "midwest", "east") that the real payload never contains — the
// seed's regions are sentences like "East coast fast-turn fulfillment". So in
// production both points printed the same default paragraph on every hub. These
// points now print the hub's actual name, window and region, and one plain
// sentence about what the buyer can do next.
func publicAvailabilitySupportPoints(parseAvailability availabilityPage) []string {
	parsePoints := make([]string, 0, 3)
	parseName := strings.TrimSpace(parseAvailability.Warehouse.Name)
	parseWindow := strings.TrimSpace(parseAvailability.Warehouse.ServiceLevel)
	switch {
	case parseName != "" && parseWindow != "":
		parsePoints = append(parsePoints, "Ships from "+parseName+" in "+parseWindow+".")
	case parseName != "":
		parsePoints = append(parsePoints, "Ships from "+parseName+".")
	case parseWindow != "":
		parsePoints = append(parsePoints, "Delivery window is "+parseWindow+".")
	}
	if parseRegion := strings.TrimSpace(parseAvailability.Warehouse.Region); parseRegion != "" {
		parsePoints = append(parsePoints, "Serves "+parseRegion+".")
	}
	switch {
	case parseAvailability.Available > 0:
		parsePoints = append(parsePoints, "Ask for a price now: this hub can fill the order from stock.")
	case parseAvailability.Inbound > 0:
		parsePoints = append(parsePoints, "Reserve now and you hold your place in the next inbound batch for this hub.")
	default:
		parsePoints = append(parsePoints, "Nothing is on hand here. Ask about another hub or a substitute.")
	}
	return parsePoints
}

// =============================================================================
// THE ONE PAGE HEAD
// =============================================================================

// renderPublicHero is the storefront's single page head: eyebrow, title, one
// sentence, and at most two actions.
//
// This replaces two hero variants (a large one for list routes and a compact one
// for detail routes) that between them shipped four gradients, three blurred glow
// divs, a second eyebrow chip, three metric cards and three "signal pills". Two
// of those pills and one of those cards printed "24 workspace products" — a
// hardcoded number, on a storefront that has four, directly above a computed
// count that said 4. Both are gone: the catalog's own result note is the count of
// record, because it is derived from the payload and cannot drift.
//
// design.PageHead supplies the heavy ink rule under the title. That rule is the
// mark that makes the page START, and it is why the route content below needs no
// second banner to separate itself.
func renderPublicHero(parsePayload Payload) ui.Node {
	parseConfig := publicHeroConfig(parsePayload.Route.Path)
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: design.Class(design.PageHead())},
			html.Span(html.Props{Class: design.Class(design.Eyebrow())}, html.Text(parseConfig.kicker)),
			html.H1(html.Props{Class: design.Class(design.PageTitle())}, html.Text(fallback(parsePayload.Route.Title, "Atlas"))),
		),
		html.P(html.Props{Class: design.Class(design.Prose(design.StepLede), design.Measure())}, html.Text(parseConfig.summary)),
	}
	parseActions := make([]ui.Node, 0, 2)
	if parseConfig.primaryLabel != "" {
		parseActions = append(parseActions,
			html.A(html.Props{Href: parseConfig.primaryHref, Class: design.Class(design.ButtonPrimary())}, html.Text(parseConfig.primaryLabel)),
		)
	}
	if parseConfig.secondaryLabel != "" {
		parseActions = append(parseActions,
			html.A(html.Props{Href: parseConfig.secondaryHref, Class: design.Class(design.ButtonSecondary())}, html.Text(parseConfig.secondaryLabel)),
		)
	}
	if len(parseActions) > 0 {
		parseChildren = append(parseChildren,
			html.Div(html.Props{Class: design.Class(design.Cluster(design.Space2))}, parseActions...),
		)
	}
	return html.Section(html.Props{Class: design.Class(design.Stack(design.Space3))}, parseChildren...)
}

type publicHeroState struct {
	kicker         string
	summary        string
	primaryLabel   string
	primaryHref    string
	secondaryLabel string
	secondaryHref  string
}

// publicHeroConfig is the per-route head copy.
//
// The sentence is written here rather than taken from page.go's routeSummary
// because that function's storefront copy is corporate mush ("Browse workspace
// systems with clear pricing cues, delivery context, and straightforward next
// steps") — three abstract nouns and no verb the reader can act on. Each sentence
// below names something the buyer controls and says what they will see.
//
// A route omits an action rather than offering one that points at itself: on
// /shop the primary action is not "browse the catalog", because the catalog is
// already on screen.
func publicHeroConfig(parsePath string) publicHeroState {
	parseState := publicHeroState{
		kicker:         "Warehouse-backed workspace systems",
		summary:        "Compare workspace systems with the delivery window attached to every line.",
		primaryLabel:   "Browse the catalog",
		primaryHref:    RouteCatalog,
		secondaryLabel: "Compare delivery regions",
		secondaryHref:  RouteWarehouses,
	}
	switch {
	case parsePath == RouteCatalog:
		parseState.kicker = "Catalog"
		parseState.summary = "Filter the manifest, then compare price and availability line by line."
		parseState.primaryLabel = ""
		parseState.secondaryLabel = "Compare delivery regions"
		parseState.secondaryHref = RouteWarehouses
	case strings.HasPrefix(parsePath, RouteCatalog+"/"):
		parseState.kicker = "Product"
		parseState.summary = "Check the price, then check which hub can ship it to you."
		parseState.primaryLabel = ""
		parseState.secondaryLabel = "Back to the catalog"
		parseState.secondaryHref = RouteCatalog
	case parsePath == RouteWarehouses:
		parseState.kicker = "Delivery regions"
		parseState.summary = "Pick the hub that serves your site, then see what it holds."
		parseState.primaryLabel = "Browse the catalog"
		parseState.secondaryLabel = ""
	case strings.HasPrefix(parsePath, RouteWarehouses+"/"):
		parseState.kicker = "Delivery region"
		parseState.summary = "What this hub holds, and how long it takes to reach you."
		parseState.primaryLabel = "Browse the catalog"
		parseState.secondaryLabel = ""
	}
	return parseState
}
