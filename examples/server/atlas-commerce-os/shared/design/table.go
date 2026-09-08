package design

import "github.com/monstercameron/GoWebComponents/v6/css"

// The data table — the primitive Atlas needs most, and the one the old design did
// not have.
//
// Almost every internal Atlas screen is a queue: open discrepancies, inbound
// receipts, transfers in flight, low-stock SKUs. The old design rendered those as
// lists of rounded cards, one card per row. A card list is the wrong shape for a
// queue for three concrete reasons:
//
//  1. Columns stop aligning. Twelve quantities down a card list cannot be compared,
//     which is the entire operation the operator came to perform.
//  2. Density collapses. A card row costs ~110px against a table row's ~34px, so a
//     40-item queue becomes four screens of scrolling.
//  3. Every row gets a border and a radius, so no row can be emphasized — the
//     exception you are hunting for looks exactly like the 39 rows that are fine.
//
// Queues become tables. Cards do not appear in this design system at all.
//
// # This bundle styles its own descendants
//
// Table is applied to the <table> element and styles thead/tbody/th/td through
// descendant selectors, so the converter writes plain semantic markup and gets the
// whole manifest look. That choice has a cascade consequence worth understanding,
// because it will bite anyone extending this file:
//
// A descendant rule from this bundle (".c-table tbody td", specificity 0-1-2)
// OUTRANKS a class applied directly to that cell (".c-numeric", 0-1-0). Left alone,
// NumericCell's text-align would silently lose on a <th> and win on a <td> — the
// worst kind of bug, because it works in the case you test first.
//
// The fix is cellOverride: every cell modifier is emitted through the "&&" variant,
// which doubles the generated class in the selector (".c-numeric.c-numeric", 0-2-0)
// and therefore outranks any descendant rule this bundle can produce. That is a
// deliberate, contained use of specificity doubling — it is the only place in the
// package that touches specificity on purpose, and TestCellModifiersOutrankTable
// pins it.

var tableBundle = clip(css.Rules(
	css.W(css.Full),
	css.Raw("border-collapse", "collapse"),
	css.Raw("table-layout", "auto"),
	css.TextColor(Ink()),

	// The table's default cell voice is Data — mono, tabular, dense. That is the
	// right default because a table in Atlas is a manifest: nearly every cell is a
	// SKU, a hub code, a quantity, a date or a status. Prose cells are the exception
	// and opt in via ProseCell, rather than every fact cell having to opt in to mono.
	// Defaults should cost nothing at the common call site.
	Data(StepFine),

	// --- header -----------------------------------------------------------------
	//
	// Header cells are Display micro: condensed uppercase graphite, closed by the
	// heavy 2px ink rule. That rule is the single mark that makes a <table> read as
	// printed paperwork rather than as a grid of divs, and it is why the header does
	// not need a filled background to separate itself.
	css.Descendant(css.El("thead th"),
		css.Rules(
			Display(StepMicro),
			css.TextColor(Graphite()),
			css.Raw("text-align", "left"),
			css.PaddingX(Space3),
			css.PaddingY(Space2),
			css.Raw("border-bottom", manifestRule()),
			css.Raw("white-space", "nowrap"),
			// Sticky header, so scrolling a 200-row queue does not lose the column
			// meanings. It needs an opaque background or the rows show through.
			css.Position.Sticky,
			css.Raw("top", "0"),
			css.Raw("z-index", "1"),
			css.Bg(Paper()),
		)...,
	),

	// --- body cells -------------------------------------------------------------
	css.Descendant(css.El("tbody td"),
		css.PaddingX(Space3),
		css.PaddingY(Space2),
		css.Raw("border-bottom", hairlineRule()),
		css.Raw("vertical-align", "top"),
	),
	css.Descendant(css.El("tbody th"),
		css.Rules(
			// A row header (a SKU in the first column) stays mono but gains weight,
			// so the row has an anchor to read from without a second color.
			css.FontWeight.Semibold,
			css.Raw("text-align", "left"),
			css.PaddingX(Space3),
			css.PaddingY(Space2),
			css.Raw("border-bottom", hairlineRule()),
			css.Raw("vertical-align", "top"),
			css.Raw("white-space", "nowrap"),
		)...,
	),

	// --- zebra ------------------------------------------------------------------
	//
	// Zebra via paper-sunk, i.e. one step of value on the same paper — not a tint, not
	// a translucent overlay. A translucent stripe changes the contrast of every
	// foreground color per row, which silently breaks the contrast of graphite meta
	// text on alternating rows.
	//
	// Zebra earns its place here specifically because these tables are wide: at 12+
	// columns, tracking one row across the screen is the failure mode, and stripes
	// fix it more cheaply than row hover alone.
	css.Descendant(css.El("tbody tr"),
		css.NthChild(css.Odd, css.Bg(PaperSunk()))...,
	),

	// --- row hover --------------------------------------------------------------
	//
	// TRAP: within one folded class, css/rule.go's canonicalize sorts emitted blocks
	// by (at-rule, selector) — NOT by source order. So writing the hover rule after
	// the zebra rule does nothing: ":hover" sorts before ":nth-child", and the zebra
	// would win the tie on order.
	//
	// The fix is to win on SPECIFICITY instead. Targeting the cells under a hovered
	// row gives "& tbody tr:hover td" (0-1-3) against the zebra's
	// "& tbody tr:nth-child(odd)" (0-1-2), so hover reliably beats the stripe on
	// every row. Never rely on source order inside a fold.
	css.Descendant(css.El("tbody tr"),
		css.Hover(
			css.Descendant(css.El("td"), css.Bg(Paper()))...,
		)...,
	),
	css.Descendant(css.El("tbody tr"),
		css.Hover(
			css.Descendant(css.El("th"), css.Bg(Paper()))...,
		)...,
	),

	// Last row keeps its rule: the table's bottom edge is the manifest's bottom edge,
	// and a queue that trails off without one looks truncated.
))

// Table is the dense manifest table. Apply it to the <table> element; thead, tbody,
// th and td are styled for you, so the markup stays plain and semantic.
//
//	html.Div(html.Props{Class: design.Class(design.SurfaceFlush())},
//	    html.Div(html.Props{Class: design.Class(design.TableScroll())},
//	        html.Table(html.Props{Class: design.Class(design.Table())}, …),
//	    ),
//	)
func Table() []css.Rule { return tableBundle }

var tableScrollBundle = clip(css.Rules(
	css.Raw("overflow-x", "auto"),
	css.MaxWidth(css.Full),
	css.MinWidth(css.Zero),
	// Momentum/inertial scrolling on touch, and contained overscroll so scrolling a wide
	// table sideways does not trigger the browser's back-navigation gesture.
	css.Raw("-webkit-overflow-scrolling", "touch"),
	css.Raw("overscroll-behavior-x", "contain"),

	// PRINT: overflow must go back to visible, and this is the print rule most likely
	// to be forgotten and most expensive to forget. An `overflow-x:auto` box does not
	// scroll on paper — it CLIPS. A 14-column manifest printed through this container
	// would silently lose every column past the page width, and the printout would look
	// complete: a correct-looking table with four columns missing off the right edge is
	// worse than a table that obviously did not fit.
	//
	// With overflow visible the fragmentation engine can at least shrink or overflow the
	// table honestly, and the print token block plus @page's 14mm margin give it room.
	css.Media(printQuery,
		css.Raw("overflow", "visible"),
		css.MaxWidth(css.RawLength("none")), // max-width takes `none`, not `auto`
	),
))

// TableScroll is the horizontal scroll container a wide table needs on a narrow
// viewport. This is how a 14-column manifest survives 380px: the table keeps its
// columns and the container scrolls, rather than the table reflowing into an
// unreadable stack of label/value pairs.
//
// An overflow container with no tab stop is a keyboard trap in reverse — the
// overflowing content is unreachable without a mouse — so give the element
// role="region", an aria-label, and tabindex="0":
//
//	html.Div(html.Props{
//	    Class: design.Class(design.TableScroll()),
//	    Role:  "region",
//	    Raw:   map[string]any{"tabIndex": 0},   // NOT Props.TabIndex — see below
//	    Aria:  map[string]string{"label": "Discrepancy lines"},
//	}, table)
//
// The tabindex has to go through Raw. html.Props silently omits a zero-valued TabIndex
// (html/html.go:59), and zero is precisely the value a scroll region needs, so
// Props{TabIndex: 0} renders no attribute at all and the region stays unreachable. This
// was found by asserting on rendered markup, not by reading the API. (Note the SSR
// serializer emits the Raw key verbatim, so the markup reads tabIndex="0"; HTML
// attribute names are case-insensitive, so that is correct, just unusual to see.)
func TableScroll() []css.Rule { return tableScrollBundle }

// --- cell modifiers -----------------------------------------------------------

// cellOverride is the specificity-doubling variant: it emits ".c-x.c-x" instead of
// ".c-x", so a modifier applied directly to a cell outranks the Table bundle's
// descendant rules for that cell (0-2-0 beats 0-1-2).
//
// css.DefineVariant takes any selector template containing "&", and "&&" is the
// standard doubling idiom. Reach for this ONLY where a bundle must beat another
// bundle's descendant rules; used casually it turns into a specificity arms race,
// which is what !important is downstream of.
var cellOverride = css.DefineVariant("&&")

var numericCellBundle = clip(cellOverride(
	css.Raw("text-align", "right"),
	css.FontVariantNumeric.TabularNums,
	css.Raw("white-space", "nowrap"),
	// Mono faces sometimes ligate "!=" or "->" inside an id; in a data cell a
	// ligature hides a character, so ligatures are off wherever numbers live.
	css.Raw("font-variant-ligatures", "none"),
))

// NumericCell right-aligns a quantity, price or count and pins it to tabular figures.
// Apply it to the <th> in the header AND every <td> in the column — a right-aligned
// column under a left-aligned header reads as broken.
//
// Right alignment is not a preference: it puts the units digit of every number in the
// same column, so magnitude is readable as a shape and 1,000 cannot be mistaken for
// 100 at a glance. Every numeric column in Atlas gets this.
func NumericCell() []css.Rule { return numericCellBundle }

var proseCellBundle = clip(cellOverride(
	fontFamily(TokenFontBody),
	css.FontWeight.Normal,
	css.Tracking(css.Zero),
	css.Raw("white-space", "normal"),
	css.MinWidth(css.Px(180)),
	css.MaxWidth(css.RawLength("42ch")),
))

// ProseCell is the ONE documented exception to "every table cell is mono": a cell
// holding a sentence — an operator note, a rejection reason, a product description.
//
// It exists because the alternative is worse. Without it, a 30-word note in mono
// either blows the column out or wraps into a wall of fixed-width text, and someone
// eventually "fixes" that by making the whole table proportional, which costs the
// column alignment the table was chosen for. One opt-out, named for what it is.
func ProseCell() []css.Rule { return proseCellBundle }

var cellMetaBundle = clip(cellOverride(
	css.TextColor(Graphite()),
))

// CellMeta de-emphasizes a cell that is context rather than content: a timestamp
// beside an event, a unit beside a quantity. Graphite is the only de-emphasis tool —
// there is no opacity scale, because translucent text over a zebra stripe has a
// different contrast on every other row.
func CellMeta() []css.Rule { return cellMetaBundle }
