package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/css"
	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONSOLE STYLING — every class in the operator console comes from shared/design.
//
// The console used to carry hand-written Tailwind on every element, which is how
// it ended up with rounded bordered boxes nested four deep at identical visual
// weight. Read shared/design/doc.go before editing anything below; the three
// rules that the old markup broke, and that this file now holds, are:
//
//  1. ONE Surface per region. The design system ships no Card, Panel or
//     nested-surface primitive ON PURPOSE (shared/design/surfaces.go). Separation
//     INSIDE a region is a Divider (hairline) or a Stack/Cluster gap — never a
//     second bordered box. If you find yourself wanting a box inside a box, that
//     is the signal to use a rule or space instead, or to split the region into
//     two Surfaces at the same level.
//  2. QUEUES ARE TABLES. The activity feed, the attention stack and the vendor
//     order watch are design.Table rows, not card lists: a card list cannot align
//     a column of quantities, which is the whole operation the operator came to
//     perform (shared/design/table.go).
//  3. MACHINE FACTS ARE MONO. design.Data for a SKU, hub code, ETA, quantity,
//     price, status code or id — fixed advance width makes a transposed digit
//     visible and lines a column up. design.Prose for sentences. design.Display
//     names REGIONS (page and section titles), never content.
//
// The folded strings sit behind functions rather than package-level vars because
// design.Class emits lazily: an init-time fold would hand out class names whose
// CSS a later css.Reset() (which the design package's own tests call) has thrown
// away. css.New memoizes on the canonical rule-set, so a repeat fold is a map
// lookup — cheap enough for everything here. Hoist the STRING at the call site
// only for a genuinely hot loop, the way the table-cell helpers below are used.
// ─────────────────────────────────────────────────────────────────────────────

// consoleShellClass is the console frame: rail track plus content column.
func consoleShellClass() string { return design.Class(design.ConsoleShell()) }

// consoleRailClass is the persistent left rail. Below 960px design.ConsoleRail
// turns itself into a horizontally scrolling strip with the same markup, which is
// what replaced the old duplicate mobile nav drawer.
func consoleRailClass() string { return design.Class(design.ConsoleRail()) }

// consoleContentClass is the content column beside the rail.
func consoleContentClass() string { return design.Class(design.ContentColumn()) }

// consoleRegionStackClass separates page REGIONS. Space5 is the design system's
// between-sections step; the console never puts a border around a region just to
// group it.
func consoleRegionStackClass() string { return design.Class(design.Stack(design.Space5)) }

// consoleSurfaceClass is the one flat paper plane, with the default stack gap for
// its children. Use it ONCE per region — see rule 1 in the header comment.
func consoleSurfaceClass() string {
	return design.Class(design.Surface(), design.Stack(design.Space4))
}

// consoleFlushSurfaceClass is the same plane with no gutter, for a table that
// should bleed to its own hairline instead of floating inside 16px of padding.
func consoleFlushSurfaceClass() string { return design.Class(design.SurfaceFlush()) }

// consoleRecessClass is pressed paper: a payload, a preview, a raw snapshot — the
// INPUT to a page rather than its output. It has no border and no radius, which is
// what stops it being used as a second card.
func consoleRecessClass() string {
	return design.Class(design.Recess(), design.Stack(design.Space2))
}

func consoleStackTightClass() string { return design.Class(design.Stack(design.Space2)) }
func consoleStackClass() string      { return design.Class(design.Stack(design.Space4)) }
func consoleClusterClass() string    { return design.Class(design.Cluster(design.Space4)) }
func consoleChipRowClass() string    { return design.Class(design.Cluster(design.Space2)) }

// consoleFactRowClass spaces label/value pairs across a row. Space6 rather than
// Space4: a fact row is read as separate columns, and at Space4 two short facts
// read as one four-word phrase.
func consoleFactRowClass() string { return design.Class(design.Cluster(design.Space6)) }

// consoleSplitRowClass is "title on the left, actions on the right", collapsing to
// a wrapped stack instead of a horizontal scrollbar at 380px.
func consoleSplitRowClass() string { return design.Class(design.SplitRow(design.Space4)) }

func consoleDividerClass() string { return design.Class(design.Divider()) }

// --- type roles ---------------------------------------------------------------

func consolePageHeadClass() string     { return design.Class(design.PageHead()) }
func consolePageTitleClass() string    { return design.Class(design.PageTitle()) }
func consoleSectionTitleClass() string { return design.Class(design.SectionTitle()) }
func consoleEyebrowClass() string      { return design.Class(design.Eyebrow()) }

// consoleLedeClass is the one paragraph under a page title. Measure caps the line
// length; the content column deliberately does not (shared/design/shell.go).
func consoleLedeClass() string {
	return design.Class(design.Prose(design.StepLede), design.Measure())
}

func consoleProseClass() string {
	return design.Class(design.Prose(design.StepBase), design.Measure())
}

func consoleProseFineClass() string { return design.Class(design.Prose(design.StepFine)) }

// consoleMetaClass is de-emphasised prose: a timestamp beside an event, a hint.
//
// GAP: the design system exposes no "muted prose" role — the closest bundles are
// FieldHint (micro, form-scoped) and CellMeta (table-cell scoped, and emitted
// through the specificity-doubling variant so it is wrong outside a cell). One
// declaration on an exported token is the cheapest honest answer; graphite is the
// system's only de-emphasis tool, so this cannot drift into an opacity scale.
func consoleMetaClass() string {
	return design.Class(design.Prose(design.StepFine), []css.Rule{css.TextColor(design.Graphite())})
}

// consoleDataClass / consoleFigureClass are the mono voices: a fact inline, and a
// fact that IS the answer (a count, a total) at lede size.
func consoleDataClass() string   { return design.Class(design.Data(design.StepFine)) }
func consoleFigureClass() string { return design.Class(design.Data(design.StepLede)) }
func consoleCodeClass() string   { return design.Class(design.Data(design.StepMicro)) }

// --- table --------------------------------------------------------------------

func consoleTableClass() string       { return design.Class(design.Table()) }
func consoleTableScrollClass() string { return design.Class(design.TableScroll()) }

// consoleNumericCellClass right-aligns a quantity or count and pins tabular
// figures. Apply it to the <th> AND every <td> in the column: a right-aligned
// column under a left-aligned header reads as broken.
func consoleNumericCellClass() string { return design.Class(design.NumericCell()) }

// consoleProseCellClass is the ONE documented exception to "every cell is mono":
// a cell holding a sentence (an operator note, a rejection reason).
func consoleProseCellClass() string { return design.Class(design.ProseCell()) }

func consoleCellMetaClass() string { return design.Class(design.CellMeta()) }

// consoleCellLinkClass is the row anchor: a link inside an already-mono cell, so
// it inherits the cell's family and only takes the link's colour and underline.
func consoleCellLinkClass() string { return design.Class(design.Link()) }

// consoleQueueTable is the shape every Atlas queue takes: a FLUSH surface (a dense
// table inside 16px of padding wastes the density it was chosen for), a horizontal
// scroll region that a keyboard can actually reach, and the table itself.
//
// The scroll region needs tabindex="0" — without a tab stop, the columns that
// overflow at 380px are unreachable for anyone not using a mouse. html.TabIndexZero
// is the sentinel that makes a literal 0 expressible; note that shared/design's
// TableScroll doc still recommends Raw["tabIndex"], which predates the sentinel.
func consoleQueueTable(parseLabel string, parseHead ui.Node, parseRows []ui.Node) ui.Node {
	return html.Div(html.Props{Class: consoleFlushSurfaceClass()},
		html.Div(html.Props{
			Class:    consoleTableScrollClass(),
			Role:     "region",
			TabIndex: html.TabIndexZero,
			Aria:     map[string]string{"label": parseLabel},
		},
			html.Table(html.Props{Class: consoleTableClass()},
				html.Thead(html.Props{}, parseHead),
				html.Tbody(html.Props{}, parseRows...),
			),
		),
	)
}

// consoleColumn / consoleNumericColumn are header cells. A numeric column takes the
// modifier on the <th> as well as every <td>, because a right-aligned column under
// a left-aligned header reads as broken.
func consoleColumn(parseLabel string) ui.Node {
	return html.Th(html.Props{}, html.Text(parseLabel))
}

func consoleNumericColumn(parseLabel string) ui.Node {
	return html.Th(html.Props{Class: consoleNumericCellClass()}, html.Text(parseLabel))
}

// consoleEmptyRow is the empty state INSIDE the manifest, so the table keeps its
// header and its bottom rule instead of collapsing into a bare sentence.
//
// The copy is always "nothing is here yet, and here is what will fill it" — never
// an apology and never "Something went wrong", because an empty queue is a normal
// state, not a failure.
func consoleEmptyRow(parseColumns int, parseMessage string) ui.Node {
	return html.Tr(html.Props{},
		html.Td(html.Props{ColSpan: parseColumns, Class: consoleProseCellClass()}, html.Text(parseMessage)),
	)
}

// consoleSectionHead is the label for a region: an eyebrow, a section title, and an
// optional mono count pushed to the trailing edge.
//
// It is deliberately NOT a box. The old markup wrapped every one of these in a
// bordered panel and then put another bordered panel inside it for the content,
// which is the nesting this conversion exists to remove.
func consoleSectionHead(parseEyebrow string, parseTitle string, parseCount string) ui.Node {
	parseTitleBlock := []ui.Node{}
	if strings.TrimSpace(parseEyebrow) != "" {
		parseTitleBlock = append(parseTitleBlock, html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseEyebrow)))
	}
	if strings.TrimSpace(parseTitle) != "" {
		parseTitleBlock = append(parseTitleBlock, html.H2(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseTitle)))
	}
	parseChildren := []ui.Node{html.Div(html.Props{Class: consoleStackTightClass()}, parseTitleBlock...)}
	if strings.TrimSpace(parseCount) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleDataClass()}, html.Text(parseCount)))
	}
	return html.Div(html.Props{Class: consoleSplitRowClass()}, parseChildren...)
}

// --- controls -----------------------------------------------------------------

func consolePrimaryButtonClass() string   { return design.Class(design.ButtonPrimary()) }
func consoleSecondaryButtonClass() string { return design.Class(design.ButtonSecondary()) }
func consoleQuietButtonClass() string     { return design.Class(design.ButtonQuiet()) }
func consoleLinkClass() string            { return design.Class(design.Link()) }

func consoleFieldClass() string      { return design.Class(design.Field()) }
func consoleFieldLabelClass() string { return design.Class(design.FieldLabel()) }

// consoleInputClass is for WORDS a person types (a note, a reason);
// consoleInputDataClass is for MACHINE FACTS (a SKU, a hub id, a quantity, a
// date). The split is the Data/Prose information architecture pushed into the
// form layer, and it does real work: a mono SKU field makes a transposed
// character visible while the operator is still typing.
func consoleInputClass() string      { return design.Class(design.Input()) }
func consoleInputDataClass() string  { return design.Class(design.InputData()) }
func consoleFieldHintClass() string  { return design.Class(design.FieldHint()) }
func consoleFieldErrorClass() string { return design.Class(design.FieldError()) }

// consoleStatusChipClass is a machine-reported state, driven by a semantic tone
// rather than a colour. The LABEL is Atlas vocabulary and stays at the call site;
// colour is a second channel on top of the words, never the only one.
func consoleStatusChipClass(parseTone design.StatusTone) string {
	return design.Class(design.StatusChip(parseTone))
}

// consoleStatusValueClass tones a bare value when the NUMBER is the status — a
// short quantity, an overdue date. A chip beside a figure that already says "-4"
// is redundant, and a dense manifest cannot afford redundancy.
func consoleStatusValueClass(parseTone design.StatusTone) string {
	return design.Class(design.Data(design.StepFine), design.StatusValue(parseTone))
}

// atlasStatusTone maps Atlas's status vocabulary onto the design system's four
// semantic tones. It is the ONLY place in the console that turns a status string
// into a colour, which is what makes the mapping reviewable by reading one
// function — the old markup made that decision inline at ~30 call sites.
func atlasStatusTone(parseStatus string) design.StatusTone {
	switch strings.ReplaceAll(strings.TrimSpace(strings.ToLower(parseStatus)), "-", "_") {
	case "approved", "closed", "reconciled", "received", "resolved", "in_stock", "healthy", "available", "balanced", "published":
		return design.ToneVerified
	// Exception is the ONLY filled tone in the design system, which is what makes
	// it readable at a glance in a long queue: four filled rows among forty are
	// four blocks your eye lands on. That property is spent, not free — every
	// status added here dilutes it, and a status category that is routinely
	// non-empty turns the queue into a wall of oxide that operators learn to
	// ignore.
	//
	// The test for membership is BOTH: something has gone wrong, AND a human must
	// act. "flagged" and "overdue" pass it — someone marked this for moderation, a
	// date came and went. "short" and "failed" pass it obviously.
	case "flagged", "cancelled", "canceled", "rejected", "failed", "blocked", "short", "overdue", "critical":
		return design.ToneException
	// promise_risk fails the first half of that test: a commitment that MAY slip
	// has not slipped. It is exactly TonePending's meaning — not yet a failure, but
	// a human should know. Giving it the filled tone would put oxide on every lane
	// with any uncertainty, which is most of them.
	//
	// recovery and in_transit arrived here when inventory's separate switch was
	// merged in. Both are "work is under way and not finished", which is what
	// Pending means; leaving them out silently demoted them to Neutral, i.e. to
	// "no claim", which is a different and wrong statement about a lane someone is
	// actively rescuing.
	case "pending", "submitted", "in_review", "review", "open", "queued", "on_hold", "inbound", "watch", "promise_risk", "recovery", "in_transit":
		return design.TonePending
	// low_stock fails BOTH halves and is the most tempting mistake here. Being
	// under the reorder point is what reorder points are FOR — the system is
	// working as designed, nothing failed, and nobody is late. The row's own
	// quantity cell already says how low. Neutral.
	case "low_stock":
		return design.ToneNeutral
	default:
		return design.ToneNeutral
	}
}

// atlasLanePosture maps a status onto the lane placard's domain vocabulary. The
// two enums stay separate on purpose (shared/design/placard.go): Atlas can grow a
// posture without the design system growing a hue.
func atlasLanePosture(parseStatus string) design.Posture {
	switch strings.ReplaceAll(strings.TrimSpace(strings.ToLower(parseStatus)), "-", "_") {
	case "closed", "reconciled", "received", "approved":
		return design.PostureClosed
	case "on_hold", "review", "in_review", "blocked":
		return design.PostureHeld
	case "cancelled", "canceled", "short", "failed", "flagged", "promise_risk":
		return design.PostureShort
	default:
		return design.PostureOnLane
	}
}

// atlasHubCode compresses a warehouse slug into the short mono code operators
// actually say out loud: "new-jersey-hub" -> "NJ-HUB", "illinois-hub" -> "IL-HUB".
//
// It exists because the placard and the rail plate set hub codes in mono at the
// loudest step in the system, where a 14-character slug both overflows the 232px
// rail and stops reading as a code. Multi-word regions collapse to their
// initials; a single-word region keeps its first two letters, which is what makes
// "illinois" come out as IL rather than I.
func atlasHubCode(parseWarehouseID string) string {
	parseSegments := strings.Split(strings.TrimSpace(strings.ToLower(parseWarehouseID)), "-")
	parseWords := make([]string, 0, len(parseSegments))
	for _, parseSegment := range parseSegments {
		if parseSegment == "" || parseSegment == "hub" || parseSegment == "warehouse" {
			continue
		}
		parseWords = append(parseWords, parseSegment)
	}
	if len(parseWords) == 0 {
		return "HUB"
	}
	parseCode := ""
	if len(parseWords) == 1 {
		parseCode = parseWords[0]
		if len(parseCode) > 2 {
			parseCode = parseCode[:2]
		}
	} else {
		for _, parseWord := range parseWords {
			parseCode += parseWord[:1]
		}
	}
	return strings.ToUpper(parseCode) + "-HUB"
}

type catalogPage struct {
	Items    []productCard     `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Query    catalogQueryState `json:"query"`
	Focus    string            `json:"focus"`
}

type catalogQueryState struct {
	Search    string `json:"search"`
	Category  string `json:"category"`
	Status    string `json:"status"`
	Warehouse string `json:"warehouse"`
	Sort      string `json:"sort"`
}

type productCard struct {
	SKU            string `json:"sku"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	PriceCents     int    `json:"priceCents"`
	Status         string `json:"status"`
	Summary        string `json:"summary"`
	Details        string `json:"details"`
	Finish         string `json:"finish"`
	SEODescription string `json:"seoDescription"`
	WarehouseID    string `json:"warehouseId"`
	WarehouseName  string `json:"warehouseName"`
	Available      int    `json:"available"`
	Inbound        int    `json:"inbound"`
	Volume         int    `json:"volume"`
	UpdatedAt      string `json:"updatedAt"`
}

type productDetailPage struct {
	Product  productCard     `json:"product"`
	Comments []commentRecord `json:"comments"`
}

type warehouseCard struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
	Pressure      string `json:"pressure"`
	Staffing      string `json:"staffing"`
	Backlog       string `json:"backlog"`
	Focus         string `json:"focus"`
	Available     int    `json:"available"`
	Inbound       int    `json:"inbound"`
	RiskCount     int    `json:"riskCount"`
}

type warehouseDetailPage struct {
	Warehouse warehouseCard `json:"warehouse"`
	Products  []productCard `json:"products"`
}

type warehouseDirectoryPage struct {
	Items []warehouseCard `json:"items"`
}

type availabilityPage struct {
	Warehouse warehouseCard `json:"warehouse"`
	Product   productCard   `json:"product"`
	Available int           `json:"available"`
	Inbound   int           `json:"inbound"`
	Status    string        `json:"status"`
}

type dashboardPage struct {
	Summary   pageSummary           `json:"summary"`
	Alerts    int                   `json:"alerts"`
	Transfers []transferRecord      `json:"transfers"`
	Receiving []receivingRecord     `json:"receiving"`
	Comments  []commentRecord       `json:"comments"`
	Orders    []purchaseOrderRecord `json:"orders"`
}

type pageSummary struct {
	Headline string            `json:"headline"`
	Items    []pageSummaryItem `json:"items"`
}

type pageSummaryItem struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

type inventoryRow struct {
	ID             string `json:"id"`
	SKU            string `json:"sku"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	PriceCents     int    `json:"priceCents"`
	ProductStatus  string `json:"productStatus"`
	WarehouseID    string `json:"warehouseId"`
	WarehouseName  string `json:"warehouseName"`
	OnHand         int    `json:"onHand"`
	Reserved       int    `json:"reserved"`
	Available      int    `json:"available"`
	CoverDays      int    `json:"coverDays"`
	Inbound        int    `json:"inbound"`
	Damaged        int    `json:"damaged"`
	ReorderPoint   int    `json:"reorderPoint"`
	SafetyStock    int    `json:"safetyStock"`
	Status         string `json:"status"`
	WeeklyUnits    int    `json:"weeklyUnits"`
	WeeklyRevenue  int    `json:"weeklyRevenue"`
	SellThrough    int    `json:"sellThrough"`
	DemandScore    int    `json:"demandScore"`
	RegionalShare  int    `json:"regionalShare"`
	ReorderUnits   int    `json:"reorderUnits"`
	MarketPressure string `json:"marketPressure"`
	MarketSignal   string `json:"marketSignal"`
	UpdatedAt      string `json:"updatedAt"`
}

type warehouseOpsList struct {
	Summary pageSummary          `json:"summary"`
	Items   []warehouseOpsRecord `json:"items"`
}

type warehouseOpsRecord struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	ServiceLevel  string `json:"serviceLevel"`
	PublicSummary string `json:"publicSummary"`
	Pressure      string `json:"pressure"`
	Staffing      string `json:"staffing"`
	Backlog       string `json:"backlog"`
	Focus         string `json:"focus"`
	Available     int    `json:"available"`
	Inbound       int    `json:"inbound"`
	RiskCount     int    `json:"riskCount"`
}

type warehouseInventoryItemDetailPage struct {
	Warehouse warehouseOpsRecord    `json:"warehouse"`
	Item      inventoryRow          `json:"item"`
	Product   productAdminCard      `json:"product"`
	Network   []inventoryRow        `json:"network"`
	Orders    []purchaseOrderRecord `json:"orders"`
	Filters   map[string]string     `json:"filters"`
}

type transferList struct {
	Items []transferRecord `json:"items"`
}

type transferRecord struct {
	ID                   string `json:"id"`
	SourceWarehouseID    string `json:"sourceWarehouseId"`
	DestinationWarehouse string `json:"destinationWarehouseId"`
	Status               string `json:"status"`
	Reason               string `json:"reason"`
	RecommendedBy        string `json:"recommendedBy"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type transferLineRecord struct {
	ID         string `json:"id"`
	TransferID string `json:"transferId"`
	ProductSKU string `json:"productSku"`
	Quantity   int    `json:"quantity"`
}

type transferDetailPage struct {
	Transfer transferRecord       `json:"transfer"`
	Lines    []transferLineRecord `json:"lines"`
}

type purchaseOrderList struct {
	Summary pageSummary           `json:"summary"`
	Items   []purchaseOrderRecord `json:"items"`
}

type purchaseOrderRecord struct {
	ID            string `json:"id"`
	VendorName    string `json:"vendorName"`
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	Status        string `json:"status"`
	PriorityNote  string `json:"priorityNote"`
	ETA           string `json:"eta"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type purchaseOrderLineRecord struct {
	ID              string `json:"id"`
	PurchaseOrderID string `json:"purchaseOrderId"`
	ProductSKU      string `json:"productSku"`
	Quantity        int    `json:"quantity"`
	ETA             string `json:"eta"`
	Status          string `json:"status"`
}

type purchaseOrderDetailPage struct {
	Order purchaseOrderRecord       `json:"order"`
	Lines []purchaseOrderLineRecord `json:"lines"`
}

type receivingList struct {
	Items []receivingRecord `json:"items"`
}

type receivingRecord struct {
	ID                 string `json:"id"`
	SourceType         string `json:"sourceType"`
	SourceID           string `json:"sourceId"`
	WarehouseID        string `json:"warehouseId"`
	Status             string `json:"status"`
	DiscrepancySummary string `json:"discrepancySummary"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type receivingLineRecord struct {
	ID                 string `json:"id"`
	ReceivingSessionID string `json:"receivingSessionId"`
	ProductSKU         string `json:"productSku"`
	ExpectedQuantity   int    `json:"expectedQuantity"`
	ActualQuantity     int    `json:"actualQuantity"`
	DiscrepancyReason  string `json:"discrepancyReason"`
}

type receivingDetailPage struct {
	Session receivingRecord       `json:"session"`
	Lines   []receivingLineRecord `json:"lines"`
}

type commentList struct {
	Summary pageSummary     `json:"summary"`
	Items   []commentRecord `json:"items"`
}

type settingsPage struct {
	Summary pageSummary `json:"summary"`
}

type commentRecord struct {
	ID               string `json:"id"`
	ProductSKU       string `json:"productSku"`
	AuthorName       string `json:"authorName"`
	AuthorType       string `json:"authorType"`
	Reaction         string `json:"reaction"`
	Subject          string `json:"subject"`
	Body             string `json:"body"`
	Status           string `json:"status"`
	ModerationReason string `json:"moderationReason"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type mockSignInRole struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type mockSignInPage struct {
	Roles   []mockSignInRole `json:"roles"`
	Next    string           `json:"next"`
	Message string           `json:"message"`
}

type recoveryPage struct {
	Title         string `json:"title"`
	Message       string `json:"message"`
	RecoveryHref  string `json:"recoveryHref"`
	RecoveryLabel string `json:"recoveryLabel"`
	Detail        string `json:"detail"`
}

type internalShellState struct {
	RouteKey        string
	SummaryLabel    string
	SummaryValue    string
	RouteBadges     map[string]string
	ActiveFilters   []string
	WorkspaceStats  []pageSummaryItem
	ActiveSavedView string
}

type shellPresentationState struct {
	RouteKey         string
	Theme            string
	Locale           string
	Density          string
	DefaultWarehouse string
}

const (
	atlasShellPresentationAtomID   = "atlas-shell-presentation"
	atlasShellPresentationStoreKey = "atlas-shell-presentation"
	atlasRouteWorkspaceAtomID      = "atlas-route-workspace"
	atlasShellRootID               = "atlas-shell-root"
	atlasOverlayRootID             = "atlas-overlay-root"
)

func App(parsePayload Payload) ui.Node {
	parseSurface := strings.TrimSpace(parsePayload.Route.Surface)
	parseRootClass := atlasRootSurfaceClass(atlasVisualSurfacePublic)
	parseMainClass := atlasMainShellClass(atlasVisualSurfacePublic)
	if parseSurface != "" && parseSurface != "public" {
		// The console frame is design.ConsoleShell: a rail track plus a content
		// column, sized by ONE grid so the two can never drift apart. The old
		// internal shell was a centered max-w-6xl strip under a full-width header,
		// which is why navigation had to eat the top of every screen.
		parseRootClass = consoleShellClass()
		parseMainClass = consoleContentClass()
	}
	parseDesiredPresentation := shellPresentationStateFromPayload(parsePayload)
	parsePresentationAtom := useAtlasAtom(atlasShellPresentationAtomID, parseDesiredPresentation)
	parsePresentation := parsePresentationAtom.Get()
	if parsePresentation.RouteKey != parseDesiredPresentation.RouteKey {
		parsePresentation = parseDesiredPresentation
	}
	useAtlasEffect(func() func() {
		if parsePresentationAtom.Get() != parseDesiredPresentation {
			parsePresentationAtom.Set(parseDesiredPresentation)
		}
		return nil
	}, parseDesiredPresentation)
	useAtlasEffect(func() func() {
		_ = persistAtlasSnapshot(atlasShellPresentationStoreKey, atlasShellPresentationAtomID)
		return nil
	}, parsePresentation)
	parseDesiredWorkspace := useAtlasComputed(func() internalShellState {
		return internalShellStateForPayload(parsePayload)
	}, parsePayload.Route.Path, parsePayload.Data, parsePayload.SavedViews).Get()
	parseWorkspaceAtom := useAtlasAtom(atlasRouteWorkspaceAtomID, parseDesiredWorkspace)
	parseShellState := parseWorkspaceAtom.Get()
	if parseShellState.RouteKey != parseDesiredWorkspace.RouteKey {
		parseShellState = parseDesiredWorkspace
	}
	useAtlasEffect(func() func() {
		parseWorkspaceAtom.Set(parseDesiredWorkspace)
		return nil
	}, parseDesiredWorkspace)
	parseToastChannel := useAtlasChannel(atlasShellToastBus)
	parseToastState := useAtlasState(atlasShellToast{})
	parseAnnouncer := ui.UseAnnouncer()
	if parseToastChannel.Ok() {
		parseLatest := parseToastChannel.Get()
		if parseLatest != parseToastState.Get() {
			parseToastState.Set(parseLatest)
		}
	}
	parseRouteAnnouncement := strings.TrimSpace(fallback(parsePayload.Route.Title, parsePayload.Route.Screen))
	parsePreviousRouteAnnouncement := ui.UsePrevious(parseRouteAnnouncement)
	useAtlasEffect(func() func() {
		if !parsePreviousRouteAnnouncement.Ok() || strings.TrimSpace(parseRouteAnnouncement) == "" || parsePreviousRouteAnnouncement.Get() == parseRouteAnnouncement {
			return nil
		}
		parseAnnouncer.Polite("Route changed to " + parseRouteAnnouncement + ".")
		return nil
	}, parseRouteAnnouncement)
	parseNoticeMessage := atlasNoticeMessage(parsePayload)
	parsePreviousNotice := ui.UsePrevious(parseNoticeMessage)
	useAtlasEffect(func() func() {
		if strings.TrimSpace(parseNoticeMessage) == "" || (parsePreviousNotice.Ok() && parsePreviousNotice.Get() == parseNoticeMessage) {
			return nil
		}
		parseAnnouncer.Polite(parseNoticeMessage)
		return nil
	}, parseNoticeMessage)
	parsePreviousToast := ui.UsePrevious(parseToastState.Get())
	useAtlasEffect(func() func() {
		parseCurrent := parseToastState.Get()
		if strings.TrimSpace(parseCurrent.Title) == "" || (parsePreviousToast.Ok() && parsePreviousToast.Get() == parseCurrent) {
			return nil
		}
		parseAnnouncement := parseCurrent.Title
		if parseDetail := strings.TrimSpace(parseCurrent.Detail); parseDetail != "" {
			parseAnnouncement += ". " + parseDetail
		}
		parseAnnouncer.Polite(parseAnnouncement)
		return nil
	}, parseToastState.Get())
	useAtlasEffect(func() func() {
		parseCurrent2 := parseToastState.Get()
		if strings.TrimSpace(parseCurrent2.Title) == "" {
			return nil
		}
		parseStop := make(chan struct{})
		go func(parseExpected atlasShellToast) {
			select {
			case <-time.After(4 * time.Second):
				if parseToastState.Get() == parseExpected {
					parseToastState.Set(atlasShellToast{})
				}
			case <-parseStop:
			}
		}(parseCurrent2)
		return func() {
			close(parseStop)
		}
	}, parseToastState.Get())
	parseChildren := []ui.Node{header(parsePayload, parseShellState)}
	parseMainChildren := []ui.Node{}
	if parseToast := shellToastBanner(parseToastState.Get()); parseToast != nil {
		parseMainChildren = append(parseMainChildren, parseToast)
	}
	if parseBanner := noticeBanner(parsePayload); parseBanner != nil {
		parseMainChildren = append(parseMainChildren, parseBanner)
	}
	if parseDemoPanel := renderGuidedDemoPanel(parsePayload); parseDemoPanel != nil {
		parseMainChildren = append(parseMainChildren, parseDemoPanel)
	}
	// WHY pageContent goes through ui.CreateElement instead of being called here.
	//
	// pageContent is a route SWITCH: it dispatches on parsePayload.Route.Path into
	// ~25 different content functions, and many of those call hooks (directly or
	// through helpers). Calling it inline made those hooks land in App's OWN fiber,
	// which means App's hook sequence depended on the current route: /app/inventory
	// contributed a ui.UseForm plus a search-params atom, /app/dashboard contributed
	// nothing, /shop/:slug contributed something else again.
	//
	// Hook slots are positional. A fiber whose hook count changes reads every later
	// slot as the wrong hook. That was masked only because client/main.go builds a
	// fresh closure per route render, forcing a full remount; the moment App
	// re-rendered IN PLACE — any atom write, any revalidation, any toast — it would
	// have read shifted slots. This gives the route body its own fiber, so App's
	// hook sequence is now route-independent by construction.
	parseMainChildren = append(parseMainChildren, hero(parsePayload, parseShellState, parsePresentation), ui.CreateElement(func() ui.Node {
		return pageContent(parsePayload)
	}))
	parseChildren = append(parseChildren, html.Main(html.Props{Class: parseMainClass}, parseMainChildren...))
	return html.Div(html.Props{},
		html.Div(html.Props{ID: atlasShellRootID, Class: parseRootClass}, parseChildren...),
		parseAnnouncer.Region(),
		html.Div(html.Props{ID: atlasOverlayRootID}),
	)
}

func header(parsePayload Payload, parseShellState internalShellState) ui.Node {
	parseLinks := []atlasPublicNavLink{
		{Label: publicLocalizedCopy(parsePayload.I18n.Locale, "nav.storefront", publicNavStorefrontLabel), Href: RouteLanding},
		{Label: publicLocalizedCopy(parsePayload.I18n.Locale, "nav.shop", publicNavShopLabel), Href: RouteCatalog},
		{Label: publicLocalizedCopy(parsePayload.I18n.Locale, "nav.warehouses", publicNavWarehousesLabel), Href: RouteWarehouses},
	}
	if parsePayload.User != nil {
		parseLinks = append(parseLinks,
			atlasPublicNavLink{Label: "Dashboard", Href: RouteDashboard},
			atlasPublicNavLink{Label: "Inventory", Href: RouteInventory},
			atlasPublicNavLink{Label: "Products", Href: "/app/products"},
			atlasPublicNavLink{Label: "Warehouses", Href: RouteWarehouseOps},
			atlasPublicNavLink{Label: "Transfers", Href: RouteTransfers},
			atlasPublicNavLink{Label: "Purchase Orders", Href: RoutePurchaseOrders},
			atlasPublicNavLink{Label: "Receiving", Href: RouteReceiving},
			atlasPublicNavLink{Label: "Comments", Href: RouteComments},
			atlasPublicNavLink{Label: "Settings", Href: RouteSettings},
		)
	}
	if parsePayload.Route.Surface == "public" || parsePayload.Route.Surface == "" {
		return publicHeader(parsePayload, parseLinks)
	}
	return internalHeader(parsePayload, parseShellState)
}

type atlasPublicNavLink struct {
	Label string
	Href  string
}

func publicHeader(parsePayload Payload, parseLinks []atlasPublicNavLink) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-public-nav"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		parseIdentity := publicIdentityLabel
		if parsePayload.User != nil {
			parseIdentity = parsePayload.User.DisplayName + " | " + strings.ReplaceAll(parsePayload.User.Role, "_", " ")
		}
		parseDesktopNodes := make([]ui.Node, 0, len(parseLinks))
		parseMobileNodes := make([]ui.Node, 0, len(parseLinks))
		for _, parseLink := range parseLinks {
			isParseActive := activeNavLink(parsePayload.Route.Path, parseLink.Href)
			parseClassName := publicNavLinkClass(isParseActive)
			parseMobileClassName := publicMobileNavLinkClass(isParseActive)
			parseDesktopNodes = append(parseDesktopNodes, html.A(html.Props{Href: parseLink.Href, Class: parseClassName}, html.Text(parseLink.Label)))
			parseMobileNodes = append(parseMobileNodes, html.A(html.Props{Href: parseLink.Href, Class: parseMobileClassName}, html.Text(parseLink.Label)))
		}
		return html.Header(html.Props{Class: publicHeaderShellClass()},
			html.Div(html.Props{Class: publicHeaderInnerClass()},
				html.Div(html.Props{Class: "flex items-center justify-between gap-4"},
					html.Div(html.Props{Class: "flex flex-col"},
						html.P(html.Props{Class: "text-[0.7rem] font-semibold uppercase tracking-[0.42em] text-amber-700"}, html.Text(publicBrandLabel)),
						html.P(html.Props{Class: "mt-2 text-sm text-stone-400"}, html.Text(parseIdentity)),
					),
					html.Div(html.Props{Class: "flex items-center gap-3 lg:hidden"},
						html.A(html.Props{Href: RouteCatalog, Class: "inline-flex rounded-full border border-amber-300/60 px-4 py-2 text-sm font-semibold text-amber-100 transition hover:bg-amber-300/12"}, html.Text(publicLocalizedCopy(parsePayload.I18n.Locale, "browse.shop", publicBrowseShopLabel))),
						html.Button(html.Props{Type: "button", Class: "inline-flex rounded-full border border-white/12 bg-white/8 px-4 py-2 text-sm font-semibold text-stone-100 transition hover:border-amber-300/45 hover:bg-white/12", OnClick: parseOpenDrawer}, html.Text(publicLocalizedCopy(parsePayload.I18n.Locale, "menu", "Menu"))),
					),
				),
				html.Div(html.Props{Class: "hidden flex-wrap items-center gap-3 lg:flex lg:justify-end"},
					html.Nav(html.Props{Class: "flex flex-wrap items-center gap-2"}, parseDesktopNodes...),
					publicLanguageControl(parsePayload),
				),
			),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-300"}, html.Text("Storefront navigation")),
						html.P(html.Props{ID: parseTitleID, Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("Browse Atlas")),
						html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-7 text-stone-300"}, html.Text("Open the public menu to jump between landing, shop, and warehouse discovery routes without losing the current storefront context.")),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-stone-200", OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "grid gap-3"}, parseMobileNodes...),
				html.Div(html.Props{Class: "lg:hidden"}, publicLanguageControl(parsePayload)),
			)),
		)
	})
}

func publicLanguageControl(parsePayload Payload) ui.Node {
	parseCurrentLocale := fallback(parsePayload.I18n.Locale, "en")
	parseLocales := parsePayload.I18n.SupportedLocales
	if len(parseLocales) == 0 {
		parseLocales = SupportedLocales()
	}
	parseNodes := make([]ui.Node, 0, len(parseLocales))
	for _, parseLocale := range parseLocales {
		parseClassName := "rounded-full border border-white/10 bg-white/6 px-3 py-2 text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-stone-300 transition hover:border-amber-300/45 hover:bg-white/10 hover:text-white"
		if strings.EqualFold(parseCurrentLocale, parseLocale) {
			parseClassName = "rounded-full border border-amber-300/70 bg-amber-300/12 px-3 py-2 text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-amber-100"
		}
		parseNodes = append(parseNodes, html.A(html.Props{
			Href:  publicLocaleHref(parsePayload.Route.Path, parsePayload.Route.Query, parseLocale),
			Class: parseClassName,
			Raw: map[string]any{
				"aria-current": boolToAriaCurrent(strings.EqualFold(parseCurrentLocale, parseLocale)),
				"hreflang":     parseLocale,
			},
		}, html.Text(publicLocaleShortLabel(parseLocale))))
	}
	return html.Div(html.Props{Class: "grid gap-2 rounded-[1.2rem] border border-white/10 bg-white/6 p-2", Raw: map[string]any{"aria-label": publicLocalizedCopy(parseCurrentLocale, "language", "Language")}},
		html.P(html.Props{Class: "px-2 text-[0.62rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(publicLocalizedCopy(parseCurrentLocale, "language", "Language"))),
		html.Div(html.Props{Class: "flex flex-wrap gap-2"}, parseNodes...),
	)
}

func publicLocaleHref(parsePath string, parseQuery map[string][]string, parseLocale string) string {
	parseValues := url.Values{}
	for parseKey, parseItems := range parseQuery {
		if strings.EqualFold(strings.TrimSpace(parseKey), "locale") {
			continue
		}
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	parseValues.Set("locale", strings.TrimSpace(parseLocale))
	parseEncoded := parseValues.Encode()
	if parseEncoded == "" {
		return fallback(parsePath, RouteLanding)
	}
	return fallback(parsePath, RouteLanding) + "?" + parseEncoded
}

func publicLocaleShortLabel(parseLocale string) string {
	switch strings.ToLower(strings.TrimSpace(parseLocale)) {
	case "fr":
		return "FR"
	case "ar":
		return "AR"
	default:
		return "EN"
	}
}

func boolToAriaCurrent(parseActive bool) string {
	if parseActive {
		return "true"
	}
	return "false"
}

func publicLocalizedCopy(parseLocale string, parseKey string, parseFallback string) string {
	parseLocale = strings.ToLower(strings.TrimSpace(parseLocale))
	parseTable := map[string]map[string]string{
		"fr": {
			"browse.shop":    "Explorer la boutique",
			"language":       "Langue",
			"menu":           "Menu",
			"nav.shop":       "Boutique",
			"nav.storefront": "Accueil",
			"nav.warehouses": "Entrepots",
		},
		"ar": {
			"browse.shop":    "تصفح المتجر",
			"language":       "اللغة",
			"menu":           "القائمة",
			"nav.shop":       "المتجر",
			"nav.storefront": "الواجهة",
			"nav.warehouses": "المستودعات",
		},
	}
	if parseLocaleMap, parseOK := parseTable[parseLocale]; parseOK {
		if parseValue := strings.TrimSpace(parseLocaleMap[parseKey]); parseValue != "" {
			return parseValue
		}
	}
	return parseFallback
}

type atlasInternalNavLink struct {
	Label string
	Href  string
	// Code is the route's mono rail code ("RCV", "PO", "XFER"). It is real Atlas
	// information, not decoration: an operator says "RCV", not "the receiving
	// screen", and a column of fixed-advance-width codes down the rail's trailing
	// edge is what makes the frame read as a column of manifest lines rather than
	// as a generic left nav.
	Code string
}

type atlasInternalNavGroup struct {
	Label string
	Links []atlasInternalNavLink
}

// internalHeader is the console's persistent LEFT RAIL, not a header.
//
// What it replaced, and why: navigation used to render as four bordered boxes
// spanning the full width — OVERVIEW / STOCK / LOGISTICS / SUPPORT — under a
// sticky bar that also carried the brand, the page title, the route summary, the
// operator identity and five context pills. That layout spent the most valuable
// space on the page (the top of every screen) on nine links, gave no sense of
// place, and duplicated the page title and description that the hero renders
// anyway. It also shipped a second, separate mobile drawer holding the same nav,
// which is a copy of the map that has to be kept in sync by hand.
//
// design.ConsoleRail is the answer to all of that: one element, always visible,
// that becomes a horizontally scrolling strip below 960px with THE SAME children
// (shared/design/shell.go). The rail plate at its head answers "which console and
// which hub am I in", each item carries its mono route code at the trailing edge,
// and the current route is marked by an inset lane bar plus aria-current.
func internalHeader(parsePayload Payload, parseShellState internalShellState) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseIdentity := "Public browsing"
		if parsePayload.User != nil {
			parseIdentity = parsePayload.User.DisplayName + " · " + strings.ReplaceAll(parsePayload.User.Role, "_", " ")
		}
		parseChildren := []ui.Node{
			design.RailPlateBlock(publicBrandLabel+" console", atlasHubCode(fallback(parsePayload.Preferences.DefaultWarehouse, "new-jersey-hub"))),
		}
		for _, parseGroup := range internalNavGroups() {
			parseChildren = append(parseChildren,
				html.Div(html.Props{Class: design.Class(design.RailGroupLabel())}, html.Text(parseGroup.Label)))
			for _, parseLink := range parseGroup.Links {
				parseChildren = append(parseChildren, internalNavLink(parsePayload.Route.Path, parseShellState, parseLink))
			}
		}
		parseChildren = append(parseChildren,
			html.Div(html.Props{Class: design.Class(design.RailGroupLabel())}, html.Text("Storefront")),
			internalNavLink(parsePayload.Route.Path, parseShellState, atlasInternalNavLink{Label: "View storefront", Href: RouteLanding, Code: "SHOP"}),
			// The operator's own name and role close the rail, in graphite meta
			// rather than as a sixth pill. It answers "who am I signed in as", which
			// is a footnote, not a headline.
			html.P(html.Props{Class: design.Class(design.FieldHint(), []css.Rule{css.PaddingX(design.Space2), css.Raw("padding-top", string(design.Space3))})}, html.Text(parseIdentity)),
		)
		return html.Aside(html.Props{
			Class: consoleRailClass(),
			Role:  "navigation",
			Aria:  map[string]string{"label": "Operator console"},
		}, parseChildren...)
	})
}

func internalNavGroups() []atlasInternalNavGroup {
	return []atlasInternalNavGroup{
		{
			Label: "Overview",
			Links: []atlasInternalNavLink{
				{Label: "Dashboard", Href: RouteDashboard, Code: "DASH"},
				{Label: "Products", Href: "/app/products", Code: "PROD"},
			},
		},
		{
			Label: "Stock",
			Links: []atlasInternalNavLink{
				{Label: "Inventory", Href: RouteInventory, Code: "INV"},
				{Label: "Warehouses", Href: RouteWarehouseOps, Code: "HUB"},
			},
		},
		{
			Label: "Logistics",
			Links: []atlasInternalNavLink{
				{Label: "Transfers", Href: RouteTransfers, Code: "XFER"},
				{Label: "Purchase Orders", Href: RoutePurchaseOrders, Code: "PO"},
				{Label: "Receiving", Href: RouteReceiving, Code: "RCV"},
			},
		},
		{
			Label: "Support",
			Links: []atlasInternalNavLink{
				{Label: "Comments", Href: RouteComments, Code: "MSG"},
				{Label: "Settings", Href: RouteSettings, Code: "CFG"},
			},
		},
	}
}

// internalNavLink is one rail item: a prose label, the route's open-work count
// when there is any, and the mono route code.
//
// The label is Prose even though it sits beside plenty of mono (design.RailLink
// sets that): a monospace navigation reads as a terminal, not as a place. The
// count is a status chip rather than a bare number because "3" alone does not say
// whether three is normal or a problem.
func internalNavLink(parseCurrentPath string, parseShellState internalShellState, parseLink atlasInternalNavLink) ui.Node {
	isParseCurrent := activeNavLink(parseCurrentPath, parseLink.Href)
	parseClassName := design.Class(design.RailLink())
	parseProps := html.Props{Href: parseLink.Href, Class: parseClassName}
	if isParseCurrent {
		// The inset lane bar is a VISUAL marker and carries no semantics, so
		// aria-current has to be set explicitly beside it.
		parseProps.Class = design.Class(design.RailLinkCurrent())
		parseProps.Aria = map[string]string{"current": "page"}
	}
	parseTrailing := []ui.Node{}
	if parseBadge := strings.TrimSpace(parseShellState.RouteBadges[parseLink.Href]); parseBadge != "" {
		parseTrailing = append(parseTrailing,
			html.Span(html.Props{Class: consoleStatusChipClass(design.TonePending)}, html.Text(parseBadge)))
	}
	if parseCode := strings.TrimSpace(parseLink.Code); parseCode != "" {
		parseTrailing = append(parseTrailing,
			html.Span(html.Props{Class: design.Class(design.RailCode())}, html.Text(parseCode)))
	}
	return html.A(parseProps,
		html.Span(html.Props{}, html.Text(parseLink.Label)),
		// RailLink is justify-between, so wrapping the count and the code in one
		// trailing group is what keeps every code in the same column down the rail.
		html.Span(html.Props{Class: consoleChipRowClass()}, parseTrailing...),
	)
}

// --- legacy class helpers ------------------------------------------------------
//
// DEPRECATED. Nothing in this file calls these any more — the console is styled
// entirely from shared/design (see the header comment). They stay because
// inventory_cms.go and products_cms.go still call them and are converted by a
// separate pass, and because visual_primitives_test.go pins their shape.
//
// Do not reach for them in new markup. Every one of them is a rounded, bordered,
// shadowed box, which is the exact vocabulary the design system refuses: use
// design.Surface once per region and separate with design.Divider or a Stack gap.

func internalHeroSurfaceClass() string {
	return "rounded-[2rem] border border-slate-800/95 bg-[linear-gradient(145deg,rgba(8,15,28,0.98),rgba(15,23,42,0.92)_58%,rgba(10,18,32,0.98))] p-8 shadow-[0_32px_90px_rgba(8,15,30,0.38)]"
}

func internalSurfaceCardClass() string {
	return "rounded-[1.55rem] border border-slate-800/95 bg-[linear-gradient(180deg,rgba(15,23,42,0.96),rgba(7,12,24,0.98))] shadow-[0_24px_60px_rgba(2,6,23,0.3)]"
}

func internalInsetSurfaceClass() string {
	return "rounded-[1.15rem] border border-slate-800/90 bg-[linear-gradient(180deg,rgba(10,17,30,0.92),rgba(6,10,20,0.96))] shadow-[inset_0_1px_0_rgba(148,163,184,0.05)]"
}

func internalAccentSurfaceClass() string {
	return "rounded-[1.2rem] border border-cyan-300/20 bg-[linear-gradient(180deg,rgba(8,20,36,0.95),rgba(6,12,24,0.98))] shadow-[0_18px_40px_rgba(2,6,23,0.24)]"
}

func internalSurfacePillClass() string {
	return "inline-flex items-center gap-2 rounded-full border border-slate-800/95 bg-[linear-gradient(180deg,rgba(15,23,42,0.92),rgba(7,12,24,0.96))] px-3 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-slate-200 shadow-[0_14px_28px_rgba(2,6,23,0.18)]"
}

func internalTableContainerClass() string {
	return "overflow-x-auto rounded-[1.4rem] border border-slate-800/95 bg-[linear-gradient(180deg,rgba(9,14,26,0.98),rgba(5,9,18,1))] shadow-[0_24px_60px_rgba(2,6,23,0.28)]"
}

func internalTableHeaderCellClass() string {
	return "px-4 py-3 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"
}

func internalTableRowClass() string {
	return "border-t border-slate-800/90 bg-transparent text-sm text-slate-200"
}

func noticeBanner(parsePayload Payload) ui.Node {
	parseNotice := atlasNoticeMessage(parsePayload)
	if parseNotice == "" {
		return nil
	}
	if parsePayload.Route.Surface == "public" || parsePayload.Route.Surface == "" {
		return html.Div(html.Props{Class: "rounded-[1.75rem] border border-emerald-400/30 bg-emerald-400/10 px-5 py-4 text-sm font-medium text-emerald-100 shadow-[0_18px_40px_rgba(16,185,129,0.12)]"}, html.Text(parseNotice))
	}
	// A completed mutation is a VERIFIED state, so it is a status chip beside the
	// sentence rather than a green-tinted box: the tone is a second channel on the
	// words, and one flat Surface keeps the banner at the same weight as the region
	// it sits above.
	return html.Div(html.Props{Class: design.Class(design.Surface(), design.Cluster(design.Space3)), Role: "status"},
		html.Span(html.Props{Class: consoleStatusChipClass(design.ToneVerified)}, html.Text("DONE")),
		html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseNotice)),
	)
}

func atlasNoticeMessage(parsePayload Payload) string {
	return strings.ReplaceAll(firstQueryValue(parsePayload.Route.Query, "atlas_notice"), "+", " ")
}

func shellToastBanner(parseToast atlasShellToast) ui.Node {
	if strings.TrimSpace(parseToast.Title) == "" {
		return nil
	}
	// The toast used to encode its tone as a tinted, tinted-border box per tone —
	// four bespoke colour trios. Now the tone drives ONE status chip and the banner
	// itself stays a plain Surface, so a warning and a success differ in the one
	// place a reader looks for state instead of in the whole panel's colour.
	parseTone := design.TonePending
	parseLabel := "NOTICE"
	switch strings.TrimSpace(strings.ToLower(parseToast.Tone)) {
	case "success":
		parseTone = design.ToneVerified
		parseLabel = "DONE"
	case "warn", "warning", "warm":
		parseTone = design.TonePending
		parseLabel = "CHECK"
	case "danger", "error":
		parseTone = design.ToneException
		parseLabel = "FAILED"
	}
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: consoleChipRowClass()},
			html.Span(html.Props{Class: consoleStatusChipClass(parseTone)}, html.Text(parseLabel)),
			html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseToast.Title)),
		),
	}
	if strings.TrimSpace(parseToast.Detail) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseToast.Detail)))
	}
	return html.Div(html.Props{Class: consoleSurfaceClass(), Role: "status"}, parseChildren...)
}

// renderGuidedDemoPanel renders optional reviewer walkthrough prompts for Atlas signature flows.
func renderGuidedDemoPanel(parsePayload Payload) ui.Node {
	if !strings.EqualFold(strings.TrimSpace(firstQueryValue(parsePayload.Route.Query, "demo")), "1") {
		return nil
	}
	parseSteps := []struct {
		Step  string
		Title string
		Copy  string
		Href  string
	}{
		{Step: "01", Title: "Start in storefront", Copy: "Open the public catalog and inspect warehouse-aware messaging before switching to internal operations.", Href: "/shop"},
		{Step: "02", Title: "Triage inventory", Copy: "Review promise-risk lanes and move directly into SKU-level correction workflows.", Href: "/app/inventory?status=promise_risk"},
		{Step: "03", Title: "Rebalance transfers", Copy: "Create a transfer lane when cross-warehouse balancing beats vendor-side replenishment.", Href: "/app/transfers"},
		{Step: "04", Title: "Close receiving", Copy: "Resolve discrepancy notes and close receiving sessions so availability can normalize.", Href: "/app/receiving"},
		{Step: "05", Title: "Verify settings resume", Copy: "Switch locale/theme/density and confirm route entry keeps operator defaults intact.", Href: "/app/settings"},
	}
	// A numbered walkthrough IS a sequence, so this is the one place in the console
	// where step numbers are correct — unlike the dashboard shortcuts, where they
	// implied an order that does not exist. It renders as a table because the steps
	// are a queue of five identical rows, and the step number is mono so the column
	// aligns.
	parseRows := make([]ui.Node, 0, len(parseSteps))
	for _, parseStep := range parseSteps {
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Th(html.Props{}, html.Text(parseStep.Step)),
			html.Td(html.Props{},
				html.A(html.Props{Href: parseStep.Href, Class: consoleCellLinkClass()}, html.Text(parseStep.Title)),
			),
			html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseStep.Copy)),
		))
	}
	return html.Div(html.Props{Class: consoleSurfaceClass()},
		html.Div(html.Props{Class: consoleStackTightClass()},
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Guided demo mode")),
			html.H2(html.Props{Class: consoleSectionTitleClass()}, html.Text("Walk the five Atlas flows in order")),
		),
		html.Div(html.Props{Class: consoleTableScrollClass(), Role: "region", TabIndex: html.TabIndexZero, Aria: map[string]string{"label": "Guided demo steps"}},
			html.Table(html.Props{Class: consoleTableClass()},
				html.Thead(html.Props{}, html.Tr(html.Props{},
					html.Th(html.Props{}, html.Text("Step")),
					html.Th(html.Props{}, html.Text("Flow")),
					html.Th(html.Props{}, html.Text("What to look at")),
				)),
				html.Tbody(html.Props{}, parseRows...),
			),
		),
	)
}

func hero(parsePayload Payload, parseShellState internalShellState, parsePresentation shellPresentationState) ui.Node {
	if parsePayload.Route.Surface == "public" || parsePayload.Route.Surface == "" {
		return renderPublicHero(parsePayload)
	}
	// The page head, and the ONLY place the route title and its description render.
	// They used to appear two or three times per screen — the sticky header, this
	// hero, and a breadcrumb — with identical text, which trains the reader to skip
	// all of them. design.PageHead closes the title with the heavy ink rule that
	// makes a page start.
	//
	// The hero also used to be a two-column grid whose left cell was a 2rem-radius
	// gradient panel with ~250px of dead space under three lines of text. There is
	// no filler now: title, one sentence, then the facts.
	parseFacts := []ui.Node{}
	if strings.TrimSpace(parseShellState.SummaryValue) != "" {
		parseFacts = append(parseFacts, statCard(fallback(parseShellState.SummaryLabel, "Open work"), parseShellState.SummaryValue))
	}
	// Warehouse is a working CONTEXT — which hub this operator is standing in — so
	// it stays. THEME, LOCALE and DENSITY used to sit here beside OPEN ALERTS as if
	// they were measurements: they are settings, they live on /app/settings, and the
	// values printed here were stale anyway because they came from a different
	// snapshot than the form that edits them.
	parseFacts = append(parseFacts, statCard("Warehouse", atlasHubCode(fallback(parsePresentation.DefaultWarehouse, "new-jersey-hub"))))
	if strings.TrimSpace(parseShellState.ActiveSavedView) != "" {
		parseFacts = append(parseFacts, statCard("Saved view", parseShellState.ActiveSavedView))
	}
	if len(parseShellState.ActiveFilters) > 0 {
		parseFacts = append(parseFacts, statCard("Filters", fmt.Sprintf("%d active", len(parseShellState.ActiveFilters))))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		html.Div(html.Props{Class: consolePageHeadClass()},
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(fallback(parsePayload.Route.Screen, "route"))),
			html.H1(html.Props{Class: consolePageTitleClass()}, html.Text(fallback(parsePayload.Route.Title, "Atlas"))),
		),
		html.P(html.Props{Class: consoleLedeClass()}, html.Text(routeSummary(parsePayload.Route.Path))),
		html.Div(html.Props{Class: consoleFactRowClass()}, parseFacts...),
	)
}

func shellPresentationStateFromPayload(parsePayload Payload) shellPresentationState {
	return shellPresentationState{
		RouteKey:         shellRouteKey(parsePayload.Route.Path, parsePayload.Route.Query),
		Theme:            parsePayload.Theme.Mode,
		Locale:           parsePayload.I18n.Locale,
		Density:          parsePayload.Preferences.Density,
		DefaultWarehouse: parsePayload.Preferences.DefaultWarehouse,
	}
}

func currentShellPresentationState(parsePayload Payload) shellPresentationState {
	return useAtlasAtom(atlasShellPresentationAtomID, shellPresentationStateFromPayload(parsePayload)).Get()
}

func currentRouteWorkspaceState(parsePayload Payload) internalShellState {
	return useAtlasAtom(atlasRouteWorkspaceAtomID, internalShellStateForPayload(parsePayload)).Get()
}

func shellRouteKey(parsePath string, parseQuery map[string][]string) string {
	parseValues := url.Values{}
	for parseKey, parseItems := range parseQuery {
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	if parseEncoded := parseValues.Encode(); parseEncoded != "" {
		return parsePath + "?" + parseEncoded
	}
	return parsePath
}

func internalShellStateForPayload(parsePayload Payload) internalShellState {
	parseState := internalShellState{
		RouteKey:    shellRouteKey(parsePayload.Route.Path, parsePayload.Route.Query),
		RouteBadges: map[string]string{},
	}
	if parsePayload.User == nil {
		return parseState
	}
	parsePath := strings.TrimSpace(parsePayload.Route.Path)
	switch {
	case parsePath == RouteDashboard:
		parsePage := decode[dashboardPage](pageData(parsePayload))
		parseState.SummaryLabel = "Open alerts"
		parseState.SummaryValue = fmt.Sprintf("%d", parsePage.Alerts)
		parseState.RouteBadges[RouteDashboard] = fmt.Sprintf("%d", parsePage.Alerts)
		parseState.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(parsePage.Transfers))
		parseState.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(parsePage.Receiving))
		parseState.RouteBadges[RouteComments] = fmt.Sprintf("%d", len(parsePage.Comments))
		parseState.WorkspaceStats = []pageSummaryItem{
			{Label: "Alerts", Value: fmt.Sprintf("%d", parsePage.Alerts)},
			{Label: "Transfers", Value: fmt.Sprintf("%d", len(parsePage.Transfers))},
			{Label: "Receiving", Value: fmt.Sprintf("%d", len(parsePage.Receiving))},
			{Label: "Comments", Value: fmt.Sprintf("%d", len(parsePage.Comments))},
		}
	case parsePath == "/app/products":
		parsePage2 := decode[productCMSPageData](pageData(parsePayload))
		parseTotal := parsePage2.Total
		if parseTotal == 0 {
			parseTotal = len(parsePage2.Items)
		}
		parseState.SummaryLabel = "Catalog items"
		parseState.SummaryValue = fmt.Sprintf("%d", parseTotal)
		parseState.RouteBadges["/app/products"] = fmt.Sprintf("%d", parseTotal)
	case parsePath == RouteInventory:
		parsePage3 := decode[inventoryCMSPage](pageData(parsePayload))
		parseWorkspace := inventoryWorkspaceSnapshotFromRows(parsePage3.Items)
		parseState.SummaryLabel = "Risk lanes"
		parseState.SummaryValue = fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)
		parseState.RouteBadges[RouteInventory] = fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)
		parseState.ActiveFilters = inventoryFilterSummaryLabels(parsePage3.Filters)
		parseState.ActiveSavedView = matchingInventorySavedViewName(parsePayload, parsePage3.Filters)
		parseState.WorkspaceStats = []pageSummaryItem{
			{Label: "Visible SKUs", Value: fmt.Sprintf("%d", len(parseWorkspace.Summaries))},
			{Label: "Visible lanes", Value: fmt.Sprintf("%d", parseWorkspace.Rollup.VisibleLanes)},
			{Label: "Risk lanes", Value: fmt.Sprintf("%d", parseWorkspace.Rollup.RiskLanes)},
			{Label: "Inbound units", Value: fmt.Sprintf("%d", parseWorkspace.Rollup.InboundUnits)},
		}
	case strings.HasPrefix(parsePath, RouteInventory+"/"):
		parsePage4 := decode[inventoryDetailPage](pageData(parsePayload))
		parseRollup := inventoryRollupFromRows(parsePage4.Rows)
		parseState.SummaryLabel = "Tracked lanes"
		parseState.SummaryValue = fmt.Sprintf("%d", parseRollup.VisibleLanes)
		parseState.RouteBadges[RouteInventory] = fmt.Sprintf("%d", parseRollup.RiskLanes)
	case parsePath == RouteWarehouseOps:
		parsePage5 := decode[warehouseOpsList](pageData(parsePayload))
		parseRiskCount := countRiskWarehouses(parsePage5.Items)
		parseState.SummaryLabel = "Risk facilities"
		parseState.SummaryValue = fmt.Sprintf("%d", parseRiskCount)
		parseState.RouteBadges[RouteWarehouseOps] = fmt.Sprintf("%d", parseRiskCount)
	case strings.HasPrefix(parsePath, RouteWarehouseOps+"/"):
		parsePage6 := decode[warehouseInventoryDetailPage](payloadDataValue(parsePayload, "detail"))
		if strings.TrimSpace(parsePage6.Warehouse.ID) == "" {
			parsePage6 = decode[warehouseInventoryDetailPage](pageData(parsePayload))
		}
		if strings.TrimSpace(parsePage6.Warehouse.ID) != "" {
			parseWorkspace2 := warehouseDetailWorkspaceSnapshotFromPage(parsePage6)
			parseState.SummaryLabel = "Warehouse filters"
			parseState.SummaryValue = fmt.Sprintf("%d", len(warehouseFilterSummaryLabels(parsePage6.Filters)))
			parseState.RouteBadges[RouteWarehouseOps] = fmt.Sprintf("%d", parseWorkspace2.RiskLanes)
			parseState.ActiveFilters = warehouseFilterSummaryLabels(parsePage6.Filters)
			parseState.WorkspaceStats = []pageSummaryItem{
				{Label: "Visible items", Value: fmt.Sprintf("%d", len(parsePage6.Inventory))},
				{Label: "Risk lanes", Value: fmt.Sprintf("%d", parseWorkspace2.RiskLanes)},
				{Label: "Open orders", Value: fmt.Sprintf("%d", len(parsePage6.Orders))},
				{Label: "Weekly demand", Value: fmt.Sprintf("%d", parseWorkspace2.TotalDemand)},
			}
		}
	case parsePath == RouteTransfers:
		parsePage7 := decode[transferList](pageData(parsePayload))
		parseState.SummaryLabel = "Active transfers"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage7.Items))
		parseState.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(parsePage7.Items))
	case strings.HasPrefix(parsePath, RouteTransfers+"/"):
		parsePage8 := decode[transferDetailPage](pageData(parsePayload))
		parseState.SummaryLabel = "Transfer lines"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage8.Lines))
		parseState.RouteBadges[RouteTransfers] = fmt.Sprintf("%d", len(parsePage8.Lines))
	case parsePath == RoutePurchaseOrders:
		parsePage9 := decode[purchaseOrderList](pageData(parsePayload))
		parseState.SummaryLabel = "Open purchase orders"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage9.Items))
		parseState.RouteBadges[RoutePurchaseOrders] = fmt.Sprintf("%d", len(parsePage9.Items))
	case strings.HasPrefix(parsePath, RoutePurchaseOrders+"/"):
		parsePage10 := decode[purchaseOrderDetailPage](payloadDataValue(parsePayload, "detail"))
		if strings.TrimSpace(parsePage10.Order.ID) == "" {
			parsePage10 = decode[purchaseOrderDetailPage](pageData(parsePayload))
		}
		parseState.SummaryLabel = "Inbound lines"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage10.Lines))
		parseState.RouteBadges[RoutePurchaseOrders] = fmt.Sprintf("%d", len(parsePage10.Lines))
	case parsePath == RouteReceiving:
		parsePage11 := decode[receivingList](pageData(parsePayload))
		parseState.SummaryLabel = "Receiving sessions"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage11.Items))
		parseState.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(parsePage11.Items))
	case strings.HasPrefix(parsePath, RouteReceiving+"/"):
		parsePage12 := decode[receivingDetailPage](pageData(parsePayload))
		parseState.SummaryLabel = "Receiving lines"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage12.Lines))
		parseState.RouteBadges[RouteReceiving] = fmt.Sprintf("%d", len(parsePage12.Lines))
	case parsePath == RouteComments || strings.HasPrefix(parsePath, RouteComments+"/"):
		parsePage13 := decode[commentList](pageData(parsePayload))
		parseState.SummaryLabel = "Visible comments"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePage13.Items))
		parseState.RouteBadges[RouteComments] = fmt.Sprintf("%d", len(parsePage13.Items))
	case parsePath == RouteSettings || strings.HasPrefix(parsePath, RouteSettings+"/"):
		parseState.SummaryLabel = "Saved views"
		parseState.SummaryValue = fmt.Sprintf("%d", len(parsePayload.SavedViews))
		parseState.RouteBadges[RouteSettings] = fmt.Sprintf("%d", len(parsePayload.SavedViews))
	}
	return parseState
}

func countRiskWarehouses(parseItems []warehouseOpsRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if parseItem.RiskCount > 0 {
			parseCount++
		}
	}
	return parseCount
}

func inventoryFilterSummaryLabels(parseFilters map[string]string) []string {
	parseLabels := []string{}
	if parseValue := strings.TrimSpace(parseFilters["q"]); parseValue != "" {
		parseLabels = append(parseLabels, "Search: "+parseValue)
	}
	if parseValue2 := strings.TrimSpace(parseFilters["warehouse"]); parseValue2 != "" && !strings.EqualFold(parseValue2, "all") {
		parseLabels = append(parseLabels, "Warehouse: "+parseValue2)
	}
	if parseValue3 := strings.TrimSpace(parseFilters["status"]); parseValue3 != "" && !strings.EqualFold(parseValue3, "all") {
		parseLabels = append(parseLabels, "Status: "+strings.ReplaceAll(parseValue3, "_", " "))
	}
	if parseValue4 := strings.TrimSpace(parseFilters["sort"]); parseValue4 != "" {
		parseLabels = append(parseLabels, "Sort: "+parseValue4)
	}
	return parseLabels
}

func warehouseFilterSummaryLabels(parseFilters map[string]string) []string {
	parseLabels := []string{}
	if parseValue := strings.TrimSpace(parseFilters["q"]); parseValue != "" {
		parseLabels = append(parseLabels, "Search: "+parseValue)
	}
	if parseValue2 := strings.TrimSpace(parseFilters["status"]); parseValue2 != "" && !strings.EqualFold(parseValue2, "all") {
		parseLabels = append(parseLabels, "Status: "+strings.ReplaceAll(parseValue2, "_", " "))
	}
	if parseValue3 := strings.TrimSpace(parseFilters["sort"]); parseValue3 != "" {
		parseLabels = append(parseLabels, "Sort: "+parseValue3)
	}
	return parseLabels
}

func matchingInventorySavedViewName(parsePayload Payload, parseFilters map[string]string) string {
	parseCurrentWarehouse := strings.TrimSpace(parseFilters["warehouse"])
	parseCurrentStatus := strings.TrimSpace(parseFilters["status"])
	parseCurrentSort := strings.TrimSpace(parseFilters["sort"])
	for _, parseSaved := range parsePayload.SavedViews {
		if !strings.EqualFold(strings.TrimSpace(parseSaved.Scope), "inventory") {
			continue
		}
		parseSavedWarehouse := strings.TrimSpace(parseSaved.Filters["warehouse"])
		parseSavedStatus := strings.TrimSpace(parseSaved.Filters["status"])
		if parseSavedStatus == "" {
			parseSavedStatus = strings.TrimSpace(parseSaved.Filters["stock-health"])
		}
		isParseMatchesWarehouse := parseSavedWarehouse == "" || strings.EqualFold(parseSavedWarehouse, parseCurrentWarehouse)
		isParseMatchesStatus := parseSavedStatus == "" || strings.EqualFold(parseSavedStatus, parseCurrentStatus)
		isParseMatchesSort := strings.TrimSpace(parseSaved.SortKey) == "" || strings.EqualFold(strings.TrimSpace(parseSaved.SortKey), parseCurrentSort)
		if isParseMatchesWarehouse && isParseMatchesStatus && isParseMatchesSort {
			return parseSaved.Name
		}
	}
	return ""
}

func pageContent(parsePayload Payload) ui.Node {
	if parsePayload.Route.Screen == "mock-sign-in" {
		return mockSignInContent(parsePayload)
	}
	if parsePayload.Route.Screen == "recovery" {
		return recoveryContent(parsePayload)
	}
	switch parsePayload.Route.Path {
	case RouteLanding:
		return renderLandingContent()
	case RouteCatalog:
		return renderCatalogContent(decode[catalogPage](pageData(parsePayload)))
	case RouteWarehouses:
		return renderWarehouseDirectoryContent(decode[warehouseDirectoryPage](pageData(parsePayload)))
	case RouteInventory:
		return inventoryContent(parsePayload)
	case "/app/products":
		return productsCMSContent(parsePayload)
	case RouteDashboard:
		return dashboardContent(parsePayload)
	case RouteWarehouseOps:
		return warehouseOpsContent(parsePayload)
	case RouteTransfers:
		return transfersContent(parsePayload)
	case RoutePurchaseOrders:
		return purchaseOrdersContent(parsePayload)
	case RouteReceiving:
		return receivingContent(parsePayload)
	case RouteComments:
		return commentsContent(parsePayload)
	case RouteSettings:
		return settingsContent(parsePayload)
	default:
		if strings.HasPrefix(parsePayload.Route.Path, RouteCatalog+"/") {
			return renderProductContent(decode[productDetailPage](pageData(parsePayload)), parsePayload)
		}
		if strings.Contains(parsePayload.Route.Path, "/availability/") {
			return renderAvailabilityContent(decode[availabilityPage](pageData(parsePayload)), parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouseOps+"/") && strings.Contains(parsePayload.Route.Path, "/items/") {
			return warehouseOpsContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouseOps+"/") {
			return warehouseOpsContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, "/app/products/") {
			return productEditorContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteInventory+"/") {
			return skuContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteTransfers+"/") {
			return transferDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RoutePurchaseOrders+"/") {
			return purchaseOrdersContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteReceiving+"/") {
			return receivingDetailContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteComments+"/") {
			return commentsContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteSettings+"/") {
			return settingsContent(parsePayload)
		}
		if strings.HasPrefix(parsePayload.Route.Path, RouteWarehouses+"/") {
			parsePage := decode[warehouseDetailPage](pageData(parsePayload))
			if strings.TrimSpace(parsePage.Warehouse.ID) == "" {
				parsePage.Warehouse = decode[warehouseCard](pageData(parsePayload))
			}
			return renderWarehouseDetailContent(parsePage, parsePayload)
		}
		return fallbackContent(parsePayload)
	}
}

// dashboardContent is a triage surface, read top to bottom: what is open, what
// needs a decision, where to start, what just moved, what vendors owe us, and only
// then the trend.
//
// It is ONE column now. The old layout was a two-column grid of grids, and the
// design system provides no multi-column region primitive on purpose — see the
// note in the report; Stack is the answer it does provide, and a console read in
// one vertical scan does not need the reader to choose a column first.
func dashboardContent(parsePayload Payload) ui.Node {
	parsePage := decode[dashboardPage](pageData(parsePayload))
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		dashboardSummaryBand(parsePage),
		dashboardAttentionPanel(parsePage),
		dashboardActionCluster(),
		dashboardActivityFeed(parsePage),
		dashboardPurchaseOrderSummary(parsePage.Orders),
	)
}

// REMOVED: dashboardAnalyticsPanels — a "Trend / Seven-day movement" panel whose
// every layer was fabricated.
//
// It is recorded here rather than deleted silently because the panel LOOKED like
// the most professional thing on the dashboard, and that is exactly what made it
// dangerous in an example other people copy.
//
// What it claimed, and what was actually behind it:
//
//   - "Sell-through 78%" — computed as
//     clamp(62 + orders*3 + transfers*2 - openReceiving*2, 35, 96).
//     That is not a sell-through rate. It is a magic constant plus unrelated row
//     counts, clamped so it always lands in a plausible-looking band. The same
//     shape produced "Stockout exposure" and "Fulfilment speed".
//   - "7d +3.4%" — a hardcoded string. There is no previous value to compare to.
//   - The sparkline — five hardcoded points with the fake figure appended, or a
//     series synthesised FROM the current value (+8, +4, +2, -1, …) so the bars
//     would slope the right way for the label next to them.
//   - "Rising while replenishment and transfers stay aligned." — narration of a
//     trend that was never measured.
//
// Atlas has no time-series store. There is no history table, nothing records a
// daily snapshot, and no request can answer "what was this yesterday". A trend
// panel is therefore not implementable here, and the honest move is to not have
// one rather than to render a convincing placeholder.
//
// If real trends are wanted later, they need a stored series first — a table of
// dated snapshots, written on a schedule — and THEN a chart. Do not reintroduce
// this by deriving history from the current value; that is the bug, not a
// shortcut to fixing it.
//
// Note the dashboard did not lose any real information: open receiving, flagged
// comments, and submitted/approved order counts all still appear in the summary
// band and the attention stack, where they are counted rather than modelled.

// dashboardSparkRowClass / dashboardSparkBarClass are the sparkline.
//
// GAP: the design system has no chart, sparkline or bar primitive — its subject is
// paperwork, and a chart is the one thing paperwork does not contain. These are
// therefore one-off typed rules rather than bundles, and they are held to the same
// contract as everything else: lane blue from the token (so they invert with the
// theme), square-cut (RadiusNone), no gradient. The old bars were rounded-full
// cyan-300/35, i.e. a colour the palette does not contain.
func dashboardSparkRowClass() string {
	return design.Class(design.Cluster(design.Space1), []css.Rule{
		css.Raw("flex-wrap", "nowrap"),
		css.Items.End,
		css.H(css.Rem(3)),
		css.W(css.Full),
	})
}

func dashboardSparkBarClass() string {
	return design.Class([]css.Rule{
		css.Bg(design.Lane()),
		css.Rounded(design.RadiusNone),
		css.Raw("flex", "1 1 0"),
		css.MinWidth(css.Px(2)),
	})
}

// dashboardSummaryBand is the route's own numbers and nothing else.
//
// The design note that used to render here as body copy — "the dashboard should
// read like a triage surface first: one summary band, one action cluster, one
// attention stack…" — described the LAYOUT to the operator. It is gone; the layout
// now says that by being that.
func dashboardSummaryBand(parsePage dashboardPage) ui.Node {
	return routeSummaryStrip(parsePage.Summary)
}

// dashboardActionCluster is four independent shortcuts, deliberately UNNUMBERED.
//
// They used to be labelled ACTION 1 through ACTION 4, which implies a sequence:
// an operator reading "Action 3" looks for what Action 2 was and whether skipping
// it matters. Nothing here is ordered — you take whichever one matches the problem
// in front of you.
func dashboardActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		internalWorkflowCard("", "Open low-stock lanes", "Promise-risk and low-stock lanes that need a threshold, transfer or replenishment decision.", "/app/inventory?status=promise_risk"),
		internalWorkflowCard("", "Create a transfer", "When the problem is warehouse coverage, not vendor supply.", "/app/transfers"),
		internalWorkflowCard("", "Resume receiving", "Close inbound discrepancies before they distort availability.", "/app/receiving"),
		internalWorkflowCard("", "Review comments", "Work the buyer question and moderation backlog.", RouteCommentsModeration),
	)
}

// dashboardAttentionPanel is the attention stack, and it is a TABLE.
//
// It was four bordered link cards, one per queue, which meant the four counts —
// the only numbers on the panel and the entire reason to look at it — sat at four
// different x positions and could not be compared. As mono right-aligned cells in
// one column they compare at a glance, and the largest number is the one to open.
func dashboardAttentionPanel(parsePage dashboardPage) ui.Node {
	parsePendingComments := dashboardCommentStatusCount(parsePage.Comments, "pending")
	parseFlaggedComments := dashboardCommentStatusCount(parsePage.Comments, "flagged")
	parseOpenReceiving := dashboardOpenReceivingCount(parsePage.Receiving)
	parseSubmittedOrders := dashboardPurchaseOrderStatusCount(parsePage.Orders, "submitted")
	parseRows := []ui.Node{
		dashboardAttentionRow("Pending comments", parsePendingComments, "Buyer questions waiting on a moderation decision.", RouteCommentsModeration),
		dashboardAttentionRow("Flagged comments", parseFlaggedComments, "Public notes that need a human before they mislead a buyer.", RouteComments+"/moderation/flagged"),
		dashboardAttentionRow("Open receiving sessions", parseOpenReceiving, "Discrepancies to resolve before inbound stock counts as available.", "/app/receiving"),
		dashboardAttentionRow("Submitted purchase orders", parseSubmittedOrders, "Vendor orders waiting on approval or a hold.", "/app/purchase-orders"),
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Needs attention", fmt.Sprintf("%d alerts need a decision", parsePage.Alerts), ""),
		consoleQueueTable("Queues needing attention",
			html.Tr(html.Props{},
				consoleColumn("Queue"),
				consoleNumericColumn("Waiting"),
				consoleColumn("Why it matters"),
			), parseRows),
	)
}

// dashboardAttentionRow tones the count rather than adding a chip beside it: when
// the number IS the status, a chip that repeats it is redundancy a dense manifest
// cannot afford. Zero is neutral — an empty queue is not an exception.
func dashboardAttentionRow(parseQueue string, parseCount int, parseWhy string, parseHref string) ui.Node {
	parseTone := design.ToneNeutral
	if parseCount > 0 {
		parseTone = design.TonePending
	}
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.A(html.Props{Href: parseHref, Class: consoleCellLinkClass()}, html.Text(parseQueue)),
		),
		html.Td(html.Props{Class: consoleNumericCellClass()},
			html.Span(html.Props{Class: consoleStatusValueClass(parseTone)}, html.Text(fmt.Sprintf("%d", parseCount))),
		),
		html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseWhy)),
	)
}

// dashboardActivityFeed is the other queue that used to be a card list. Same
// argument as the attention stack: four sources × two entries is eight rows, and a
// card each cost ~110px against a table row's ~34px, so the feed did not fit on a
// screen.
//
// The design note that shipped as its description — "the dashboard feed should show
// the latest moderation, transfer, receiving, and purchase-order motion in one scan
// instead of making operators open four routes…" — is deleted. SOURCE is a column
// now, which is the same claim made structurally.
func dashboardActivityFeed(parsePage dashboardPage) ui.Node {
	parseItems := dashboardActivityItems(parsePage)
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(parseItem.Kicker)),
			html.Th(html.Props{},
				html.A(html.Props{Href: parseItem.Href, Class: consoleCellLinkClass()}, html.Text(parseItem.Title)),
			),
			html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseItem.Detail)),
			html.Td(html.Props{},
				html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Meta))}, html.Text(parseItem.Meta)),
			),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(4, "Nothing has moved yet. Comments, transfers, receiving sessions and purchase orders show up here as they change."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Activity feed", "Latest motion", fmt.Sprintf("%d entries", len(parseItems))),
		consoleQueueTable("Recent operator activity",
			html.Tr(html.Props{},
				consoleColumn("Source"),
				consoleColumn("Item"),
				consoleColumn("Detail"),
				consoleColumn("Status"),
			), parseRows),
	)
}

type dashboardActivityItem struct {
	Kicker string
	Title  string
	Detail string
	Meta   string
	Href   string
}

func dashboardActivityItems(parsePage dashboardPage) []dashboardActivityItem {
	parseItems := []dashboardActivityItem{}
	for parseIndex, parseItem := range parsePage.Comments {
		if parseIndex >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Moderation",
			Title:  fallback(parseItem.Subject, "Buyer question"),
			Detail: fallback(parseItem.Body, "Public feedback needs a routing decision."),
			Meta:   strings.ReplaceAll(fallback(parseItem.Status, "pending"), "_", " "),
			Href:   "/app/comments",
		})
	}
	for parseIndex2, parseItem2 := range parsePage.Transfers {
		if parseIndex2 >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Transfer",
			Title:  parseItem2.SourceWarehouseID + " to " + parseItem2.DestinationWarehouse,
			Detail: fallback(parseItem2.Reason, "Balancing action in flight."),
			Meta:   strings.ReplaceAll(fallback(parseItem2.Status, "submitted"), "_", " "),
			Href:   "/app/transfers/" + parseItem2.ID,
		})
	}
	for parseIndex3, parseItem3 := range parsePage.Receiving {
		if parseIndex3 >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Receiving",
			Title:  parseItem3.ID,
			Detail: fallback(parseItem3.DiscrepancySummary, "Inbound session still needs closeout."),
			Meta:   strings.ReplaceAll(fallback(parseItem3.Status, "open"), "_", " "),
			Href:   "/app/receiving/" + parseItem3.ID,
		})
	}
	for parseIndex4, parseItem4 := range parsePage.Orders {
		if parseIndex4 >= 2 {
			break
		}
		parseItems = append(parseItems, dashboardActivityItem{
			Kicker: "Purchase order",
			Title:  fallback(parseItem4.VendorName, parseItem4.ID),
			Detail: fallback(parseItem4.PriorityNote, "Vendor replenishment still in motion."),
			Meta:   strings.ReplaceAll(fallback(parseItem4.Status, "submitted"), "_", " "),
			Href:   "/app/purchase-orders/" + parseItem4.ID,
		})
	}
	return parseItems
}

// dashboardPurchaseOrderSummary is the vendor watch panel: the third card list
// that is now a table. ETA is mono and gets its own column because "will it land
// before the promise" is the only question this panel answers, and a date buried in
// a meta line at the bottom of a card cannot be compared across four vendors.
func dashboardPurchaseOrderSummary(parseOrders []purchaseOrderRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseOrders))
	for parseIndex, parseItem := range parseOrders {
		if parseIndex >= 4 {
			break
		}
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Th(html.Props{},
				html.A(html.Props{Href: "/app/purchase-orders/" + parseItem.ID, Class: consoleCellLinkClass()}, html.Text(fallback(parseItem.VendorName, parseItem.ID))),
			),
			html.Td(html.Props{}, html.Text(atlasHubCode(fallback(parseItem.WarehouseID, parseItem.WarehouseName)))),
			html.Td(html.Props{}, html.Text(fallback(parseItem.ETA, "unscheduled"))),
			html.Td(html.Props{},
				html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(strings.ReplaceAll(fallback(parseItem.Status, "submitted"), "_", " "))),
			),
			html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(fallback(parseItem.PriorityNote, "No note"))),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(5, "No purchase orders are open. Vendor work appears here once replenishment needs a formal order."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Purchase orders", "Vendor watch", fmt.Sprintf("%d open", len(parseOrders))),
		consoleQueueTable("Open purchase orders",
			html.Tr(html.Props{},
				consoleColumn("Vendor"),
				consoleColumn("Hub"),
				consoleColumn("ETA"),
				consoleColumn("Status"),
				consoleColumn("Note"),
			), parseRows),
	)
}

func dashboardCommentStatusCount(parseItems []commentRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func dashboardOpenReceivingCount(parseItems []receivingRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		parseStatus := strings.TrimSpace(strings.ToLower(parseItem.Status))
		if parseStatus != "reconciled" && parseStatus != "closed" {
			parseCount++
		}
	}
	return parseCount
}

func dashboardPurchaseOrderStatusCount(parseItems []purchaseOrderRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func inventoryContent(parsePayload Payload) ui.Node {
	return inventoryCMSContent(parsePayload)
}

func skuContent(parsePayload Payload) ui.Node {
	return inventoryDetailContent(parsePayload)
}

func warehouseOpsContent(parsePayload Payload) ui.Node {
	parsePage := decode[warehouseOpsList](pageData(parsePayload))
	parsePrimaryNodes := []ui.Node{
		warehouseBreadcrumbBar(
			warehouseBreadcrumbLink{Label: "Dashboard", Href: RouteDashboard},
			warehouseBreadcrumbLink{Label: "Warehouses", Href: RouteWarehouseOps, Current: true},
		),
		warehouseOpsSummaryBand(parsePage),
		warehouseOpsActionCluster(),
	}
	if parseNestedPanel := warehouseOpsNestedPanelNode(parsePayload); parseNestedPanel != nil {
		parsePrimaryNodes = append(parsePrimaryNodes, parseNestedPanel)
	}
	parsePrimaryNodes = append(parsePrimaryNodes,
		warehouseOpsTable(parsePage.Items),
		renderWarehouseMapCard(parsePage.Items),
		inventoryRailCard("Next steps", "Where warehouse work usually goes next.",
			html.Div(html.Props{Class: consoleStackTightClass()},
				inventoryActionCard("Compare inventory pressure", "Check cross-warehouse SKU pressure before changing a local lane.", "/app/inventory?status=promise_risk"),
				inventoryActionCard("Order replenishment", "Escalate to vendor-side inbound when balancing is no longer enough.", "/app/purchase-orders"),
				inventoryActionCard("Close inbound work", "Return to receiving once recovery stock starts arriving.", "/app/receiving"),
			),
		),
	)
	// One column. The right-hand rail used to be a second grid track holding three
	// more bordered panels; flattening it means the facility table is the widest
	// thing on the page, which is what a 14-column manifest needs.
	return html.Section(html.Props{Class: consoleRegionStackClass()}, parsePrimaryNodes...)
}

func warehouseOpsNestedPanelNode(parsePayload Payload) ui.Node {
	if parseOutlet := routeOutletNode(); parseOutlet != nil {
		return parseOutlet
	}
	return WarehouseOpsDetailPanel(parsePayload)
}

// renderWarehouseMapCard renders a lightweight transfer-oriented warehouse map for visual lane scanning.
func renderWarehouseMapCard(parseItems []warehouseOpsRecord) ui.Node {
	if len(parseItems) == 0 {
		return inventoryRailCard("Warehouse transfer map", "The lane map needs at least one warehouse record.",
			html.P(html.Props{Class: consoleMetaClass()}, html.Text("No warehouse geometry to render.")),
		)
	}
	parseLimit := min(len(parseItems), 5)
	parseVisible := parseItems[:parseLimit]
	parseX := []int{96, 256, 416, 576, 336}
	parseY := []int{74, 188, 74, 188, 250}
	parseLineNodes := make([]ui.Node, 0, parseLimit)
	parsePointNodes := make([]ui.Node, 0, parseLimit*4)
	parseLegendNodes := make([]ui.Node, 0, parseLimit)
	parseHubLabel := fallback(parseVisible[0].Name, parseVisible[0].ID)
	for parseIndex, parseItem := range parseVisible {
		parseNodeX := parseX[parseIndex]
		parseNodeY := parseY[parseIndex]
		parseFill, parseStroke := formatWarehouseMapNodeTone(parseItem)
		if parseIndex > 0 {
			parseLineNodes = append(parseLineNodes, html.Tag("line", html.Props{Raw: map[string]any{
				"x1": parseX[0],
				"y1": parseY[0],
				"x2": parseNodeX,
				"y2": parseNodeY,
			}, Class: "stroke-cyan-200/45 stroke-[2]"}))
		}
		parsePointNodes = append(parsePointNodes,
			html.Tag("circle", html.Props{Raw: map[string]any{
				"cx": parseNodeX,
				"cy": parseNodeY,
				"r":  24,
			}, Class: parseFill + " " + parseStroke + " stroke-[2]"}),
			html.Tag("text", html.Props{Raw: map[string]any{
				"x":              parseNodeX,
				"y":              parseNodeY - 32,
				"text-anchor":    "middle",
				"font-size":      "11",
				"letter-spacing": "0.04em",
			}, Class: "fill-slate-200"}, html.Text(strings.ToUpper(fallback(parseItem.Region, parseItem.ID)))),
			html.Tag("text", html.Props{Raw: map[string]any{
				"x":           parseNodeX,
				"y":           parseNodeY + 5,
				"text-anchor": "middle",
				"font-size":   "11",
			}, Class: "fill-white font-semibold"}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
			html.Tag("text", html.Props{Raw: map[string]any{
				"x":           parseNodeX,
				"y":           parseNodeY + 21,
				"text-anchor": "middle",
				"font-size":   "9",
			}, Class: "fill-slate-300"}, html.Text("inbound "+fmt.Sprintf("%d", parseItem.Inbound))),
		)
		// The legend entry is a status chip, so a facility carrying risk reads the same
		// way here as it does in the facility table two regions up.
		parseLegendTone := design.ToneNeutral
		if parseItem.RiskCount > 0 {
			parseLegendTone = design.ToneException
		}
		parseLegendNodes = append(parseLegendNodes, html.Span(html.Props{Class: consoleStatusChipClass(parseLegendTone)},
			html.Text(atlasHubCode(parseItem.ID)+" "+fmt.Sprintf("%d risk", parseItem.RiskCount)),
		))
	}
	return inventoryRailCard("Warehouse transfer map", "Which facilities are carrying risk, and where a balancing lane would start.",
		// GAP: the SVG's own fills and strokes stay on utility classes. The design
		// system has no data-visualisation vocabulary (formatWarehouseMapNodeTone is
		// pinned by tests to those exact strings), and inventing chart tones here would
		// be adding a palette outside the one place that owns tone -> hue.
		html.Div(html.Props{Class: consoleRecessClass()},
			html.Tag("svg", html.Props{Raw: map[string]any{
				"viewBox":    "0 0 672 296",
				"role":       "img",
				"aria-label": fmt.Sprintf("Atlas warehouse transfer map centered on %s", parseHubLabel),
			}, Class: "h-auto w-full"}, append(parseLineNodes, parsePointNodes...)...),
		),
		html.Div(html.Props{Class: consoleChipRowClass()}, parseLegendNodes...),
	)
}

// formatWarehouseMapNodeTone maps warehouse risk posture to visual tone classes inside the SVG map.
func formatWarehouseMapNodeTone(parseItem warehouseOpsRecord) (string, string) {
	if parseItem.RiskCount > 3 || strings.EqualFold(strings.TrimSpace(parseItem.Pressure), "critical") {
		return "fill-rose-400/25", "stroke-rose-300"
	}
	if parseItem.RiskCount > 0 || strings.EqualFold(strings.TrimSpace(parseItem.Pressure), "watch") {
		return "fill-amber-300/25", "stroke-amber-200"
	}
	return "fill-cyan-300/20", "stroke-cyan-200"
}

func warehouseOpsSummaryBand(parsePage warehouseOpsList) ui.Node {
	// The route's own numbers, as a fact row rather than four bordered stat boxes.
	// The paragraph that used to sit above them ("Treat the warehouse route as a
	// facility triage board first…") described the layout to the operator; the layout
	// says it by being a table-first triage board.
	return html.Div(html.Props{Class: consoleStackClass()},
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: consoleFactRowClass()},
			statCard("Facilities", fmt.Sprintf("%d", len(parsePage.Items))),
			statCard("Available units", fmt.Sprintf("%d", totalWarehouseAvailable(parsePage.Items))),
			statCard("Inbound units", fmt.Sprintf("%d", totalWarehouseInbound(parsePage.Items))),
			statCard("Facilities at risk", fmt.Sprintf("%d", countRiskWarehouses(parsePage.Items))),
		),
	)
}

func warehouseOpsActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		inventoryActionCard("Open a roster", "Start with the facility when the problem is local backlog, staffing or regional supply.", "/app/warehouses"),
		inventoryActionCard("Add warehouse item", "Seed a managed item inside the facility instead of going through the catalogue.", "/app/warehouses/new-jersey-hub#warehouse-create-item"),
		inventoryActionCard("Review flagged lanes", "Open a facility already filtered to the lanes that need action.", "/app/warehouses/new-jersey-hub?status=promise_risk"),
		inventoryActionCard("Order more units", "Open replenishment once the facility view proves inbound is the right move.", "/app/purchase-orders"),
	)
}

// warehouseOpsTable is the facility queue.
//
// Available, Inbound and Risks are NumericCell columns — right-aligned tabular
// figures — so magnitude is readable as a shape down the column and 1,000 cannot be
// mistaken for 100. The service level and the facility focus sentence move into
// their own cells instead of being stacked inside the name cell: three lines of
// mixed voice in one <td> is a card with extra steps.
func warehouseOpsTable(parseItems []warehouseOpsRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, warehouseOpsTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(8, "No warehouses in this workspace yet."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Facility table", "Facilities under pressure", fmt.Sprintf("%d facilities", len(parseItems))),
		consoleQueueTable("Warehouse facilities",
			html.Tr(html.Props{},
				consoleColumn("Hub"),
				consoleColumn("Facility"),
				consoleColumn("Service"),
				consoleColumn("Pressure"),
				consoleNumericColumn("Available"),
				consoleNumericColumn("Inbound"),
				consoleNumericColumn("Risks"),
				consoleColumn("Open"),
			), parseRows),
	)
}

func warehouseOpsTableRow(parseItem warehouseOpsRecord) ui.Node {
	// Risk count is toned, not chipped: the number IS the status here, and zero risk
	// is a verified state rather than a missing one.
	parseRiskTone := design.ToneVerified
	if parseItem.RiskCount > 0 {
		parseRiskTone = design.ToneException
	}
	return html.Tr(html.Props{},
		// The row anchor is the hub CODE, in mono, because that is what aligns down a
		// column and what an operator says out loud. The human-readable name lives in
		// the prose cell beside it.
		html.Th(html.Props{},
			html.A(html.Props{Href: "/app/warehouses/" + parseItem.ID, Class: consoleCellLinkClass()}, html.Text(atlasHubCode(parseItem.ID))),
		),
		html.Td(html.Props{Class: consoleProseCellClass()},
			html.Span(html.Props{}, html.Text(parseItem.Name)),
			html.Span(html.Props{Class: consoleMetaClass()}, html.Text(" "+parseItem.Region+" · "+parseItem.Focus)),
		),
		html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(parseItem.ServiceLevel)),
		html.Td(html.Props{Class: consoleProseCellClass()},
			html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Pressure))}, html.Text(parseItem.Pressure)),
			html.Span(html.Props{Class: consoleMetaClass()}, html.Text(" "+parseItem.Backlog)),
		),
		html.Td(html.Props{Class: consoleNumericCellClass()}, html.Text(fmt.Sprintf("%d", parseItem.Available))),
		html.Td(html.Props{Class: consoleNumericCellClass()}, html.Text(fmt.Sprintf("%d", parseItem.Inbound))),
		html.Td(html.Props{Class: consoleNumericCellClass()},
			html.Span(html.Props{Class: consoleStatusValueClass(parseRiskTone)}, html.Text(fmt.Sprintf("%d", parseItem.RiskCount))),
		),
		html.Td(html.Props{},
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.A(html.Props{Href: "/app/warehouses/" + parseItem.ID, Class: consoleCellLinkClass()}, html.Text("Workspace")),
				html.A(html.Props{Href: "/app/warehouses/" + parseItem.ID + "#warehouse-replenishment", Class: consoleCellLinkClass()}, html.Text("Replenishment")),
			),
		),
	)
}

func totalWarehouseAvailable(parseItems []warehouseOpsRecord) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Available
	}
	return parseTotal
}

func totalWarehouseInbound(parseItems []warehouseOpsRecord) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Inbound
	}
	return parseTotal
}

func warehouseOpsDetailContent(parsePayload Payload) ui.Node {
	return WarehouseOpsDetailPanel(parsePayload)
}

func transfersContent(parsePayload Payload) ui.Node {
	parsePage := decode[transferList](pageData(parsePayload))
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		transfersSummaryBand(parsePage),
		transfersActionCluster(),
		transfersTable(parsePage.Items),
		transferForm(parsePayload),
		inventoryRailCard("Next steps", "Where a balancing decision usually comes from, and where it lands.",
			html.Div(html.Props{Class: consoleStackTightClass()},
				inventoryActionCard("Review warehouse pressure", "Check which facility is starving before committing a lane.", "/app/warehouses"),
				inventoryActionCard("Open inventory pressure", "Confirm the SKU needs a rebalance rather than a vendor order.", "/app/inventory?status=promise_risk"),
				inventoryActionCard("Confirm receiving", "Reconcile once the transfer lands.", "/app/receiving"),
			),
		),
	)
}

func transfersSummaryBand(parsePage transferList) ui.Node {
	// "Treat transfers as a balancing board: source and destination lanes,
	// recommendation context, and direct handoff into receiving…" was a note to
	// whoever built the page, printed at the operator. The counts are the content.
	return html.Div(html.Props{Class: consoleFactRowClass()},
		statCard("Transfers", fmt.Sprintf("%d", len(parsePage.Items))),
		statCard("Pending", fmt.Sprintf("%d", countTransferStatus(parsePage.Items, "pending"))),
		statCard("Approved", fmt.Sprintf("%d", countTransferStatus(parsePage.Items, "approved"))),
		statCard("Cancelled", fmt.Sprintf("%d", countTransferStatus(parsePage.Items, "cancelled"))),
	)
}

func transfersActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		inventoryActionCard("Review warehouses", "Check which facility is starving before committing a balancing lane.", "/app/warehouses"),
		inventoryActionCard("Open inventory pressure", "Confirm the SKU needs a rebalance instead of vendor replenishment.", "/app/inventory?status=promise_risk"),
		inventoryActionCard("Transfer queue", "The current movement backlog and its recommendation notes.", "/app/transfers"),
		inventoryActionCard("Check receiving", "Close the movement once the transfer actually lands.", "/app/receiving"),
	)
}

// transfersTable gives the lane its own two columns — FROM and TO, both mono hub
// codes — instead of stacking "nevada-hub -> new-jersey-hub" under the id as a
// sentence. Two aligned code columns let an operator see at a glance that four
// transfers all drain the same source hub, which is the pattern that matters.
func transfersTable(parseItems []transferRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, transferTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(7, "No transfers are recommended right now."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Transfer table", "Lanes in flight", fmt.Sprintf("%d transfers", len(parseItems))),
		consoleQueueTable("Transfer lanes",
			html.Tr(html.Props{},
				consoleColumn("Transfer"),
				consoleColumn("From"),
				consoleColumn("To"),
				consoleColumn("Status"),
				consoleColumn("Reason"),
				consoleColumn("Recommended by"),
				consoleColumn("Updated"),
			), parseRows),
	)
}

func transferTableRow(parseItem transferRecord) ui.Node {
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.A(html.Props{Href: "/app/transfers/" + parseItem.ID, Class: consoleCellLinkClass()}, html.Text(parseItem.ID)),
		),
		html.Td(html.Props{}, html.Text(atlasHubCode(parseItem.SourceWarehouseID))),
		html.Td(html.Props{}, html.Text(atlasHubCode(parseItem.DestinationWarehouse))),
		html.Td(html.Props{},
			html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " "))),
		),
		html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseItem.Reason)),
		html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(fallback(parseItem.RecommendedBy, "Atlas planning"))),
		html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(parseItem.UpdatedAt)),
	)
}

func countTransferStatus(parseItems []transferRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func transferDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[transferDetailPage](pageData(parsePayload))
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		transferDetailHero(parsePage),
		transferLineTable(parsePage.Lines),
		routeRevalidationCard("Refresh this transfer", "Re-read the lane after an approval or cancellation."),
	)
}

// transferDetailHero leads with the LANE PLACARD, the design system's signature
// device: origin hub, destination hub and posture on an inked perforated dock tag.
//
// This is exactly the object the placard exists for — a transfer IS a route plus a
// state — and it replaces three identical grey pills that spelled the same lane out
// as "nevada-hub -> new-jersey-hub" in body text. The reason sentence stays prose
// under it; a placard carries codes, never sentences.
func transferDetailHero(parsePage transferDetailPage) ui.Node {
	return html.Div(html.Props{Class: consoleStackClass()},
		design.LanePlacard(design.PlacardSpec{
			OriginHub: atlasHubCode(parsePage.Transfer.SourceWarehouseID),
			DestHub:   atlasHubCode(parsePage.Transfer.DestinationWarehouse),
			LaneID:    parsePage.Transfer.ID,
			Posture:   atlasLanePosture(parsePage.Transfer.Status),
		}),
		html.Div(html.Props{Class: consoleSplitRowClass()},
			html.Div(html.Props{Class: consoleStackTightClass()},
				html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Transfer workspace")),
				html.P(html.Props{Class: consoleProseClass()}, html.Text(parsePage.Transfer.Reason)),
				html.P(html.Props{Class: consoleMetaClass()}, html.Text("Recommended by "+fallback(parsePage.Transfer.RecommendedBy, "Atlas planning"))),
			),
			html.A(html.Props{Href: "/app/transfers", Class: consoleSecondaryButtonClass()}, html.Text("Back to transfers")),
		),
	)
}

func transferLineTable(parseLines []transferLineRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Th(html.Props{}, html.Text(parseLine.ProductSKU)),
			// The unit is a column header, not a suffix on every value: "12 units"
			// repeated down a column costs six characters per row and breaks the tabular
			// alignment that made the column readable.
			html.Td(html.Props{Class: consoleNumericCellClass()}, html.Text(fmt.Sprintf("%d", parseLine.Quantity))),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(2, "No lines on this transfer yet."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Transfer lines", "What moves", fmt.Sprintf("%d lines", len(parseLines))),
		consoleQueueTable("Transfer lines",
			html.Tr(html.Props{},
				consoleColumn("SKU"),
				consoleNumericColumn("Units"),
			), parseRows),
	)
}

// routeRevalidationCard is a COMPONENT, not a helper, because it calls hooks
// (useAtlasRevalidator and ui.UseEvent).
//
// It is rendered from nine different places, and two of them are the loaders of
// purchaseOrderDetailRail and receivingDetailRail — code that ui.UseLazyNode runs
// on a goroutine after the runtime has cleared the current fiber. Called there as
// a plain helper, its first hook hit a nil fiber and panicked; the panic was
// contained inside the loader goroutine, so the lazy state stayed {Loading:true}
// and the entire side rail simply never appeared on /app/purchase-orders/:id and
// /app/receiving/:id. No error boundary could see it: a goroutine panic has no
// path to one.
//
// Returning ui.CreateElement moves the hooks into a fiber the runtime owns. As a
// bonus it also takes them OUT of whatever route-content fiber used to host them,
// so a route that conditionally renders this card no longer changes its own hook
// count between renders.
func routeRevalidationCard(parseTitle string, parseDetail string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseRevalidator := useAtlasRevalidator()
		parseLabel := "Refresh route data"
		if parseRevalidator.Loading() {
			parseLabel = "Refreshing route..."
		}
		return html.Div(html.Props{Class: consoleSurfaceClass()},
			html.Div(html.Props{Class: consoleStackTightClass()},
				html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseTitle)),
				html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseDetail)),
			),
			// Reloading data is not THE action on any of these routes, so it is a
			// secondary control. There is exactly one primary per view and it belongs to
			// the form that changes something.
			html.Button(html.Props{
				Type:     "button",
				Class:    consoleSecondaryButtonClass(),
				Disabled: parseRevalidator.Loading(),
				OnClick:  ui.UseEvent(func() { parseRevalidator.Revalidate() }),
			}, html.Text(parseLabel)),
		)
	})
}

func purchaseOrdersContent(parsePayload Payload) ui.Node {
	parsePage := decode[purchaseOrderList](pageData(parsePayload))
	parsePrimaryNodes := []ui.Node{
		purchaseOrdersSummaryBand(parsePage),
		purchaseOrdersActionCluster(),
	}
	if parseNestedPanel := purchaseOrdersNestedPanelNode(parsePayload); parseNestedPanel != nil {
		parsePrimaryNodes = append(parsePrimaryNodes, parseNestedPanel)
	}
	parsePrimaryNodes = append(parsePrimaryNodes,
		purchaseOrdersTable(parsePage.Items),
		inventoryRailCard("Next steps", "Where vendor work goes forward and back.",
			html.Div(html.Props{Class: consoleStackTightClass()},
				inventoryActionCard("Review receiving", "Check whether an inbound session already exists for this vendor and hub.", "/app/receiving"),
				inventoryActionCard("Review transfers", "If the vendor is too slow, compare an internal balancing move.", "/app/transfers"),
				inventoryActionCard("Back to warehouses", "Re-open the owning facility when inbound ownership needs another look.", "/app/warehouses"),
			),
		),
	)
	return html.Section(html.Props{Class: consoleRegionStackClass()}, parsePrimaryNodes...)
}

func purchaseOrdersNestedPanelNode(parsePayload Payload) ui.Node {
	if parseOutlet := routeOutletNode(); parseOutlet != nil {
		return parseOutlet
	}
	return PurchaseOrderDetailPanel(parsePayload)
}

func purchaseOrdersSummaryBand(parsePage purchaseOrderList) ui.Node {
	return html.Div(html.Props{Class: consoleStackClass()},
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: consoleFactRowClass()},
			statCard("Orders", fmt.Sprintf("%d", len(parsePage.Items))),
			statCard("Submitted", fmt.Sprintf("%d", countPurchaseOrdersByStatus(parsePage.Items, "submitted"))),
			statCard("Approved", fmt.Sprintf("%d", countPurchaseOrdersByStatus(parsePage.Items, "approved"))),
			statCard("On hold", fmt.Sprintf("%d", countPurchaseOrdersByStatus(parsePage.Items, "on_hold"))),
		),
	)
}

func purchaseOrdersActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		inventoryActionCard("Start from stock risk", "Open the pressure view before creating or reviewing a replenishment plan.", "/app/inventory?status=promise_risk"),
		inventoryActionCard("Check warehouse context", "Confirm which facility should own the inbound units.", "/app/warehouses"),
		inventoryActionCard("Submitted queue", "Vendor orders waiting on an approval or hold decision.", "/app/purchase-orders"),
		inventoryActionCard("Close in receiving", "The last step, once the order becomes a real inbound session.", "/app/receiving"),
	)
}

// purchaseOrdersTable keeps the order id as the mono row anchor and gives ETA its
// own column: "will it land before the promise" is the question this queue answers,
// and an ISO-ish date in a tabular mono column sorts by eye.
func purchaseOrdersTable(parseItems []purchaseOrderRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, purchaseOrdersTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(7, "No purchase orders in this workspace yet."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Vendor order table", "Orders on the way", fmt.Sprintf("%d orders", len(parseItems))),
		consoleQueueTable("Purchase orders",
			html.Tr(html.Props{},
				consoleColumn("Order"),
				consoleColumn("Vendor"),
				consoleColumn("Hub"),
				consoleColumn("ETA"),
				consoleColumn("Status"),
				consoleColumn("Note"),
				consoleColumn("Updated"),
			), parseRows),
	)
}

func purchaseOrdersTableRow(parseItem purchaseOrderRecord) ui.Node {
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.A(html.Props{Href: "/app/purchase-orders/" + parseItem.ID, Class: consoleCellLinkClass()}, html.Text(parseItem.ID)),
		),
		html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseItem.VendorName)),
		html.Td(html.Props{}, html.Text(atlasHubCode(fallback(parseItem.WarehouseID, parseItem.WarehouseName)))),
		html.Td(html.Props{}, html.Text(fallback(parseItem.ETA, "unscheduled"))),
		html.Td(html.Props{},
			html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " "))),
		),
		html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(fallback(parseItem.PriorityNote, "No note"))),
		html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(parseItem.UpdatedAt)),
	)
}

func countPurchaseOrdersByStatus(parseItems []purchaseOrderRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

// PurchaseOrderDetailPanel renders a nested purchase-order detail panel when a child PO route is active.
func PurchaseOrderDetailPanel(parsePayload Payload) ui.Node {
	parsePage := decode[purchaseOrderDetailPage](payloadDataValue(parsePayload, "detail"))
	if strings.TrimSpace(parsePage.Order.ID) == "" {
		parsePage = decode[purchaseOrderDetailPage](pageData(parsePayload))
	}
	if strings.TrimSpace(parsePage.Order.ID) == "" {
		return nil
	}
	return purchaseOrderDetailSection(parsePayload, parsePage)
}

func purchaseOrderDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[purchaseOrderDetailPage](pageData(parsePayload))
	if strings.TrimSpace(parsePage.Order.ID) == "" {
		parsePage = decode[purchaseOrderDetailPage](payloadDataValue(parsePayload, "detail"))
	}
	return purchaseOrderDetailSection(parsePayload, parsePage)
}

func purchaseOrderDetailSection(parsePayload Payload, parsePage purchaseOrderDetailPage) ui.Node {
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		purchaseOrderDetailHero(parsePage),
		purchaseOrderLineTable(parsePage.Lines),
		purchaseOrderDetailRail(parsePayload, parsePage),
	)
}

// purchaseOrderDetailHero also leads with the lane placard: a purchase order is a
// lane from a vendor to a hub with a promise date, which is the placard's four facts
// exactly. ETA becomes PROMISE, the order id becomes the lane id, and the four grey
// pills that used to spell all of this out in body text are gone.
//
// The accessible label is written explicitly because the vendor name is not a hub
// code, so the generated sentence ("Lane NORTHLINE FABRICATION to IL-HUB") would
// read oddly; that is exactly what PlacardSpec.Label is for.
func purchaseOrderDetailHero(parsePage purchaseOrderDetailPage) ui.Node {
	parseVendor := strings.ToUpper(fallback(parsePage.Order.VendorName, "VENDOR"))
	parseHub := atlasHubCode(fallback(parsePage.Order.WarehouseID, parsePage.Order.WarehouseName))
	return html.Div(html.Props{Class: consoleStackClass()},
		design.LanePlacard(design.PlacardSpec{
			OriginHub: parseVendor,
			DestHub:   parseHub,
			Promise:   parsePage.Order.ETA,
			LaneID:    parsePage.Order.ID,
			Posture:   atlasLanePosture(parsePage.Order.Status),
			Label:     "Purchase order " + parsePage.Order.ID + " from " + parsePage.Order.VendorName + " into " + parseHub + ", promised " + fallback(parsePage.Order.ETA, "unscheduled"),
		}),
		html.Div(html.Props{Class: consoleSplitRowClass()},
			html.Div(html.Props{Class: consoleStackTightClass()},
				html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Purchase-order workspace")),
				html.H2(html.Props{Class: consoleSectionTitleClass()}, html.Text(parsePage.Order.VendorName)),
				html.P(html.Props{Class: consoleProseClass()}, html.Text(fallback(parsePage.Order.PriorityNote, "No vendor note on this order."))),
			),
			html.A(html.Props{Href: "/app/purchase-orders", Class: consoleSecondaryButtonClass()}, html.Text("Back to PO table")),
		),
	)
}

func purchaseOrderLineTable(parseLines []purchaseOrderLineRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Th(html.Props{}, html.Text(parseLine.ProductSKU)),
			html.Td(html.Props{Class: consoleNumericCellClass()}, html.Text(fmt.Sprintf("%d", parseLine.Quantity))),
			html.Td(html.Props{}, html.Text(fallback(parseLine.ETA, "unscheduled"))),
			html.Td(html.Props{},
				html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseLine.Status))}, html.Text(strings.ReplaceAll(parseLine.Status, "_", " "))),
			),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(4, "No inbound lines on this order yet."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Line-item context", "What is on the order", fmt.Sprintf("%d lines", len(parseLines))),
		consoleQueueTable("Purchase-order lines",
			html.Tr(html.Props{},
				consoleColumn("SKU"),
				consoleNumericColumn("Units"),
				consoleColumn("ETA"),
				consoleColumn("Status"),
			), parseRows),
	)
}

func receivingContent(parsePayload Payload) ui.Node {
	parsePage := decode[receivingList](pageData(parsePayload))
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		receivingSummaryBand(parsePage),
		receivingActionCluster(),
		receivingTable(parsePage.Items),
		receivingForm(parsePayload),
		inventoryRailCard("Next steps", "Receiving closes the loop on purchase orders, transfers and warehouse recovery.",
			html.Div(html.Props{Class: consoleStackTightClass()},
				inventoryActionCard("Review purchase orders", "Confirm the inbound plan before reconciling a session.", "/app/purchase-orders"),
				inventoryActionCard("Open warehouse pressure", "Check whether this inbound changes a facility's recovery path.", "/app/warehouses"),
				inventoryActionCard("Return to inventory", "Verify the quantities landed where operators expect them.", "/app/inventory"),
			),
		),
	)
}

func receivingSummaryBand(parsePage receivingList) ui.Node {
	// Discrepancies is toned in the fact row by being the count operators hunt for;
	// the "treat receiving as the closeout board for inbound work…" paragraph that
	// used to sit above it told the reader what the page is instead of showing them.
	return html.Div(html.Props{Class: consoleFactRowClass()},
		statCard("Sessions", fmt.Sprintf("%d", len(parsePage.Items))),
		statCard("Open", fmt.Sprintf("%d", countReceivingStatus(parsePage.Items, "open"))),
		statCard("Closed", fmt.Sprintf("%d", countReceivingStatus(parsePage.Items, "closed"))),
		statCard("Discrepancies", fmt.Sprintf("%d", countReceivingWithDiscrepancy(parsePage.Items))),
	)
}

func receivingActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		inventoryActionCard("Review purchase orders", "Confirm the inbound plan before reconciling a session.", "/app/purchase-orders"),
		inventoryActionCard("Transfer follow-through", "Cross-check a session that closes an internal balancing move.", "/app/transfers"),
		inventoryActionCard("Receiving queue", "The active discrepancy and closeout backlog.", "/app/receiving"),
		inventoryActionCard("Verify inventory", "Confirm quantity changes landed in the expected lanes.", "/app/inventory"),
	)
}

// receivingTable splits the session's source into TYPE and ID columns, both mono:
// "purchase_order po-1042" as one string cannot be scanned for "which of these came
// off a transfer", which is the grouping question on this queue.
func receivingTable(parseItems []receivingRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, receivingTableRow(parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(6, "No receiving sessions in this workspace yet."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Receiving table", "Sessions to close", fmt.Sprintf("%d sessions", len(parseItems))),
		consoleQueueTable("Receiving sessions",
			html.Tr(html.Props{},
				consoleColumn("Session"),
				consoleColumn("Hub"),
				consoleColumn("Source"),
				consoleColumn("Status"),
				consoleColumn("Discrepancy"),
				consoleColumn("Opened"),
			), parseRows),
	)
}

func receivingTableRow(parseItem receivingRecord) ui.Node {
	parseDiscrepancy := strings.TrimSpace(parseItem.DiscrepancySummary)
	parseDiscrepancyNode := html.Text("None")
	if parseDiscrepancy != "" {
		// A recorded discrepancy is the one thing on this row that needs a human, so it
		// is the one thing that gets the exception tone.
		parseDiscrepancyNode = html.Span(html.Props{Class: consoleStatusValueClass(design.ToneException)}, html.Text(parseDiscrepancy))
	}
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.A(html.Props{Href: "/app/receiving/" + parseItem.ID, Class: consoleCellLinkClass()}, html.Text(parseItem.ID)),
		),
		html.Td(html.Props{}, html.Text(atlasHubCode(parseItem.WarehouseID))),
		html.Td(html.Props{}, html.Text(strings.ReplaceAll(parseItem.SourceType, "_", " ")+" "+parseItem.SourceID)),
		html.Td(html.Props{},
			html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " "))),
		),
		html.Td(html.Props{Class: consoleProseCellClass()}, parseDiscrepancyNode),
		html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(parseItem.CreatedAt)),
	)
}

func countReceivingStatus(parseItems []receivingRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

func countReceivingWithDiscrepancy(parseItems []receivingRecord) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.TrimSpace(parseItem.DiscrepancySummary) != "" {
			parseCount++
		}
	}
	return parseCount
}

func receivingDetailContent(parsePayload Payload) ui.Node {
	parsePage := decode[receivingDetailPage](pageData(parsePayload))
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		receivingDetailHero(parsePage),
		receivingLineTable(parsePage.Lines),
		receivingDetailRail(parsePayload, parsePage),
	)
}

// receivingDetailHero is the third placard site: a receiving session is a lane that
// has arrived — from a purchase order or a transfer, into a hub — and its posture is
// whether closeout is clean.
func receivingDetailHero(parsePage receivingDetailPage) ui.Node {
	parseSource := strings.ToUpper(fallback(parsePage.Session.SourceID, strings.ReplaceAll(parsePage.Session.SourceType, "_", "-")))
	parseHub := atlasHubCode(parsePage.Session.WarehouseID)
	return html.Div(html.Props{Class: consoleStackClass()},
		design.LanePlacard(design.PlacardSpec{
			OriginHub: parseSource,
			DestHub:   parseHub,
			LaneID:    parsePage.Session.ID,
			Posture:   atlasLanePosture(parsePage.Session.Status),
			Label:     "Receiving session " + parsePage.Session.ID + " from " + parseSource + " into " + parseHub,
		}),
		html.Div(html.Props{Class: consoleSplitRowClass()},
			html.Div(html.Props{Class: consoleStackTightClass()},
				html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Receiving workspace")),
				html.P(html.Props{Class: consoleProseClass()}, html.Text(fallback(parsePage.Session.DiscrepancySummary, "No discrepancy recorded. This session is ready to close."))),
			),
			html.A(html.Props{Href: "/app/receiving", Class: consoleSecondaryButtonClass()}, html.Text("Back to receiving")),
		),
	)
}

// receivingLineTable is the discrepancy queue, and the one table where the numbers
// carry the whole story: expected against actual, in two right-aligned tabular
// columns so the gap is visible as a shape. A short line tones its actual quantity
// as an exception — the number IS the status, so it does not also need a chip.
func receivingLineTable(parseLines []receivingLineRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseActual := html.Text(fmt.Sprintf("%d", parseLine.ActualQuantity))
		if parseLine.ActualQuantity != parseLine.ExpectedQuantity {
			parseActual = html.Span(
				html.Props{Class: consoleStatusValueClass(design.ToneException)},
				html.Text(fmt.Sprintf("%d", parseLine.ActualQuantity)),
			)
		}
		parseRows = append(parseRows, html.Tr(html.Props{},
			html.Th(html.Props{}, html.Text(parseLine.ProductSKU)),
			html.Td(html.Props{Class: consoleNumericCellClass()}, html.Text(fmt.Sprintf("%d", parseLine.ExpectedQuantity))),
			html.Td(html.Props{Class: consoleNumericCellClass()}, parseActual),
			html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(fallback(parseLine.DiscrepancyReason, "matched"))),
		))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(4, "No lines counted on this session yet."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Receiving lines", "Counted against expected", fmt.Sprintf("%d lines", len(parseLines))),
		consoleQueueTable("Receiving lines",
			html.Tr(html.Props{},
				consoleColumn("SKU"),
				consoleNumericColumn("Expected"),
				consoleNumericColumn("Counted"),
				consoleColumn("Discrepancy"),
			), parseRows),
	)
}

// atlasLazySection defers a subtree until after the primary route body is stable.
//
// # THE LOADER INVARIANT — read this before you pass anything to this helper
//
// A loader handed to atlasLazySection may only BUILD elements. It must NOT call
// hooks, and it must NOT call helpers that call hooks. If the subtree it wants
// needs hooks — and in Atlas almost every rail does, for UseEvent handlers,
// UseId label wiring, resources, or local state — the loader must hand the
// runtime a COMPONENT:
//
//	atlasLazySection(func() ui.Node {
//	    return ui.CreateElement(myRailComponent)   // correct: runtime owns the fiber
//	}, fallback, deps...)
//
//	atlasLazySection(func() ui.Node {
//	    return html.Div(html.Props{}, someHelperThatCallsUseEvent())  // WRONG
//	}, fallback, deps...)
//
// # Why the rule exists (the mechanism, not just the style)
//
// ui.UseLazyNode runs this loader on a goroutine spawned from an effect
// (ui/ui_async.go). By the time that goroutine runs, the render pass that armed
// the fiber has finished and the runtime has already executed SetCurrentFiber(nil),
// so runtime.GetCurrentFiber() is nil and the goroutine is not the render-owner
// goroutine either. Every GWC hook resolves its per-component slots through
// requireCurrentHookFiber (internal/runtime/reconciler_elements.go), which panics
// with GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT in that situation.
//
// The panic is then CONTAINED: UseLazyNode's goroutine carries
// `defer runtime.RecoverContainedPanic("ui", "UseLazyNode loader")`. Containment
// keeps the page alive, but it also means the lazy state never leaves
// {Loading: true} and no ui.ErrorBoundary can ever see the failure — a goroutine
// panic cannot propagate to a boundary. The subtree just never appears.
//
// ui.CreateElement (ui/ui.go) is the escape hatch precisely because it is LAZY:
// it records the component function in an element and returns, and the runtime
// invokes that function later inside a fiber it owns. Building elements is safe
// on any goroutine; running hooks is not.
func atlasLazySection(parseLoader func() ui.Node, parseFallback ui.Node, parseDeps ...any) ui.Node {
	return ui.CreateElement(func() ui.Node {
		handle := ui.UseLazyNode(func(context.Context) (ui.Node, error) {
			return parseLoader(), nil
		}, parseDeps...)
		parseState := handle.Get()
		return ui.AsyncBoundary(ui.AsyncBoundaryProps{
			Pending:  parseState.Loading,
			Error:    parseState.Error,
			Fallback: parseFallback,
			ErrorFallback: func(parseErr error) ui.Node {
				return parseFallback
			},
			Content: parseState.Node,
			Delay:   120 * time.Millisecond,
		})
	})
}

func internalLazyRailFallback(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: consoleSurfaceClass()},
		html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseTitle)),
		html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseCopy)),
	)
}

func useAtlasFocusContainment(isActive bool, parseContainerSelector, parseInitialFocusSelector string) {
	parseManager := ui.UseFocusManager()
	parseArmed := ui.UseRef(false)
	ui.UseEffect(func() func() {
		if !isActive {
			parseArmed.Set(false)
			return nil
		}
		parseManager.RememberActive()
		parseArmed.Set(true)
		return func() {
			if parseArmed.Get() {
				parseManager.Restore()
				parseArmed.Set(false)
			}
		}
	}, isActive, parseContainerSelector, parseInitialFocusSelector)
	ui.UseFocusTrap(ui.FocusTrapOptions{
		Active:                isActive,
		ContainerSelector:     parseContainerSelector,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: parseContainerSelector,
	})
}

// --- overlay chrome -----------------------------------------------------------
//
// GAP: the design system has no Overlay, Scrim or Sheet primitive, so these four
// class strings are the only place the console authors layout rules of its own.
// They are held to the same contract as a bundle would be: the scrim is the INK
// token mixed toward transparent (so it inverts with the theme instead of being a
// hard-coded slate), the panel is a real design.Surface, and nothing here invents a
// radius, a shadow or a colour. A design.Overlay(kind) taking the backdrop + panel
// pair would delete all four.

// atlasScrimClass is the dimmed layer behind a modal. parseAlignEnd anchors the
// panel to the trailing edge (a sheet) instead of the centre (a dialog).
func atlasScrimClass(isAlignEnd bool) string {
	parseRules := []css.Rule{
		css.Position.Fixed,
		css.Inset(css.Zero),
		css.ZIndex(40),
		css.Display.Flex,
		css.Padding(design.Space4),
		// color-mix against the ink token rather than an rgba literal: the scrim has to
		// darken the page in the light theme and still read as a scrim in the dark one,
		// and only a token can do both.
		css.Bg(css.ColorMix(design.Ink(), css.Transparent, 72)),
	}
	if isAlignEnd {
		parseRules = append(parseRules, css.Items.Stretch, css.Justify.End)
	} else {
		parseRules = append(parseRules, css.Items.Center, css.Justify.Center)
	}
	return design.Class(parseRules)
}

// atlasDialogPanelClass is the centred confirmation panel: one Surface, capped so it
// cannot exceed the viewport, with its own scroll if the body is long.
func atlasDialogPanelClass() string {
	return design.Class(design.Surface(), design.Stack(design.Space4), []css.Rule{
		css.W(css.MinLen(css.Vw(92), css.Rem(34))),
		css.MaxHeight(css.Vh(92)),
		css.Raw("overflow-y", "auto"),
	})
}

// atlasSheetPanelClass is the full-height trailing sheet used for route-owned
// workflows.
func atlasSheetPanelClass() string {
	return design.Class(design.Surface(), design.Stack(design.Space4), []css.Rule{
		css.W(css.MinLen(css.Vw(92), css.Rem(36))),
		css.H(css.Full),
		css.Raw("overflow-y", "auto"),
	})
}

func atlasOverlayTarget() ui.PortalTarget {
	// All secondary Atlas workflows share one portal root so stacked dialogs and sheets coordinate z-order
	// and focus management instead of competing with route-owned DOM order.
	return ui.PortalTarget{Selector: "#" + atlasOverlayRootID}
}

func atlasDialogOverlay(isOpen bool, parseModalID, parseLabelledBy, parseDescribedBy, parseInitialFocusSelector string, parseOnDismiss func(), parseChild ui.Node) ui.Node {
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  isOpen,
		SurfaceID:             parseModalID,
		Kind:                  ui.OverlayKindDialog,
		LabelledBy:            parseLabelledBy,
		DescribedBy:           parseDescribedBy,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: "#" + parseModalID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         parseOnDismiss != nil,
		CloseOnOutsideClick:   parseOnDismiss != nil,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         atlasScrimClass(false),
		SurfaceClass:          atlasDialogPanelClass(),
		OnDismiss:             parseOnDismiss,
		Child:                 parseChild,
	})
}

func atlasRouteSheetOverlay(parseSurfaceID, parseLabelledBy, parseDescribedBy, parseInitialFocusSelector string, parseChild ui.Node) ui.Node {
	// Route-owned sheets intentionally disable escape/outside dismissal because these workflows represent
	// route context, not disposable popovers, and should close through explicit in-route actions.
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  true,
		Target:                atlasOverlayTarget(),
		AppRootSelector:       "#" + atlasShellRootID,
		SurfaceID:             parseSurfaceID,
		Kind:                  ui.OverlayKindSheet,
		LabelledBy:            parseLabelledBy,
		DescribedBy:           parseDescribedBy,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: "#" + parseSurfaceID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         false,
		CloseOnOutsideClick:   false,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         atlasScrimClass(true),
		SurfaceClass:          atlasSheetPanelClass(),
		Child:                 parseChild,
	})
}

func atlasDismissibleSheet(isOpen bool, parseSurfaceID, parseLabelledBy, parseDescribedBy, parseInitialFocusSelector string, parseOnDismiss func(), parseChild ui.Node) ui.Node {
	// Dismissible sheets reuse the same overlay host so focus return and backdrop behavior stay consistent
	// with modal dialogs while still allowing route-local quick-action drawers.
	return ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                  isOpen,
		Target:                atlasOverlayTarget(),
		AppRootSelector:       "#" + atlasShellRootID,
		SurfaceID:             parseSurfaceID,
		Kind:                  ui.OverlayKindSheet,
		LabelledBy:            parseLabelledBy,
		DescribedBy:           parseDescribedBy,
		InitialFocusSelector:  parseInitialFocusSelector,
		FallbackFocusSelector: "#" + parseSurfaceID,
		Modal:                 true,
		TrapFocus:             true,
		RestoreFocus:          true,
		CloseOnEscape:         parseOnDismiss != nil,
		CloseOnOutsideClick:   parseOnDismiss != nil,
		LockScroll:            true,
		Backdrop:              true,
		BackdropClass:         atlasScrimClass(true),
		SurfaceClass:          atlasSheetPanelClass(),
		OnDismiss:             parseOnDismiss,
		Child:                 parseChild,
	})
}

// atlasConfirmationDialog declares its ui.UseEvent BEFORE the `!isOpen` early
// return. That ordering is the fix; do not "tidy" the hook back down next to the
// button it feeds.
//
// # What was broken
//
// The hook used to sit AFTER `if !isOpen { return nil }`, and this function is
// inlined into four component bodies (moderationForm, bulkModerationForm,
// transferForm, receivingFormForID). Hook slots are positional and are matched by
// call ORDER within a fiber, so opening the dialog ADDED a slot to the host
// component's sequence and closing it REMOVED one. Every hook declared after the
// dialog in that host then shifted by one: on the render where the user clicks
// "Review closeout", a ui.UseState reads the previous slot's value and a
// ui.UseForm reads a handler ref. That is silent state corruption in the exact
// component whose job is to confirm a destructive action.
//
// Declaring the hook unconditionally makes the host's hook count independent of
// isOpen, which is what the positional model requires. The dialog still renders
// nothing when closed — returning nil is fine, SKIPPING A HOOK is not.
//
// # Why this is not wrapped in ui.CreateElement like the other fixes
//
// It would be the cleaner shape, but /app/comments renders moderationForm and
// bulkModerationForm at the same time, so two instances of this dialog are live
// together. ui.CreateElement keys anonymous closures by source location (see the
// FIELD PRIMITIVES note above), so both instances would collapse onto one handle
// and render the same dialog body. Hoisting the hook fixes the actual defect
// without that hazard.
//
// # The general rule
//
// Hooks are never conditional. No hook after an early return, inside an if, or in
// a loop whose length varies.
func atlasConfirmationDialog(isOpen bool, parseModalID, parseTitle, parseCopy, parseConfirmLabel string, parseOnDismiss func(), parseBody ...ui.Node) ui.Node {
	parseDismissHandler := ui.UseEvent(func() {
		if parseOnDismiss != nil {
			parseOnDismiss()
		}
	})
	if !isOpen {
		return nil
	}
	parseTitleID := parseModalID + "-title"
	parseDescriptionID := parseModalID + "-description"
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: consoleStackTightClass()},
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Confirmation")),
			html.P(html.Props{ID: parseTitleID, Class: consoleSectionTitleClass()}, html.Text(parseTitle)),
			html.P(html.Props{ID: parseDescriptionID, Class: consoleProseFineClass()}, html.Text(parseCopy)),
		),
		// A hairline between the question and what is being confirmed. This is where
		// the old markup put a second bordered box inside the dialog panel.
		html.Hr(html.Props{Class: consoleDividerClass()}),
	}
	parseChildren = append(parseChildren, parseBody...)
	parseChildren = append(parseChildren,
		// Confirming IS the action here, so this is the one primary button in the view,
		// and Cancel is quiet rather than a second outlined control competing with it.
		html.Div(html.Props{Class: design.Class(design.Cluster(design.Space3), []css.Rule{css.Justify.End})},
			html.Button(html.Props{Type: "button", Class: consoleQuietButtonClass(), OnClick: parseDismissHandler}, html.Text("Cancel")),
			html.Button(html.Props{ID: parseModalID + "-confirm", Type: "submit", Class: consolePrimaryButtonClass()}, html.Text(parseConfirmLabel)),
		),
	)
	return atlasDialogOverlay(isOpen, parseModalID, parseTitleID, parseDescriptionID, "#"+parseModalID+"-confirm", parseOnDismiss, html.Div(html.Props{Class: consoleStackClass()}, parseChildren...))
}

func atlasSidePanelErrorBoundary(parseChild ui.Node, parseTitle string, parseFallback ui.Node, resetKeys ...any) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		Child:     parseChild,
		ResetKeys: resetKeys,
		ErrorFallback: func(parseErr error, reset func()) ui.Node {
			return detailRailErrorIsland(parseTitle, parseErr, reset, parseFallback)
		},
	})
}

// purchaseOrderDetailRail is the rail that used to be missing entirely on
// /app/purchase-orders/:id — no panel, no loading skeleton, no console output.
//
// WHY every child of this loader is a component, not inline nodes:
//
// atlasLazySection hands this closure to ui.UseLazyNode, which runs it on a
// goroutine spawned from an effect — after the runtime has already cleared the
// current fiber. Any hook called here (useAtlasRevalidator, ui.UseEvent, ui.UseId)
// hits a nil fiber and panics, and RecoverContainedPanic swallows it, so the lazy
// state never leaves {Loading:true} and the subtree simply never appears. No error
// boundary can help: a goroutine panic has no path to one.
//
// So the loader body below is allowed to do exactly one thing — BUILD elements.
// Each of the three children owns its own fiber:
//   - routeRevalidationCard      → returns ui.CreateElement
//   - purchaseOrderDetailStatsIsland → returns ui.CreateElement
//   - purchaseOrderStatusForm    → returns ui.CreateElement
//
// If you add a child here, check that it is a component. The test for "is this
// safe?" is not "does it look like markup" — it is "does anything it reaches call
// a hook before ui.CreateElement gets involved".
func purchaseOrderDetailRail(parsePayload Payload, parsePage purchaseOrderDetailPage) ui.Node {
	parseFallbackNode := internalLazyRailFallback("Loading purchase-order rail", "Atlas is preparing the secondary vendor panel after the primary route body stabilizes.")
	return atlasSidePanelErrorBoundary(atlasLazySection(func() ui.Node {
		return html.Div(html.Props{Class: consoleRegionStackClass()},
			routeRevalidationCard("Purchase-order route refresh", "Re-read the order after an approval or hold so vendor posture and inbound timing stay current."),
			purchaseOrderDetailStatsIsland(parsePayload, parsePage),
			purchaseOrderStatusForm(parsePage.Order.ID, parsePage.Order.Status, parsePayload),
		)
	}, parseFallbackNode, parsePage.Order.ID, parsePage.Order.Status, parsePage.Order.ETA), "Purchase-order side panel", parseFallbackNode, parsePage.Order.ID, parsePage.Order.Status, parsePage.Order.ETA)
}

// receivingDetailRail is the /app/receiving/:id twin of purchaseOrderDetailRail and
// had the identical defect: the whole rail was absent, silently.
//
// Same invariant applies — this loader may only BUILD. Its four children are all
// components: routeRevalidationCard, receivingDetailStatsIsland, receivingFormForID,
// and renderReceivingAttachmentForm.
func receivingDetailRail(parsePayload Payload, parsePage receivingDetailPage) ui.Node {
	parseFallbackNode := internalLazyRailFallback("Loading receiving rail", "Atlas is preparing the secondary receiving panel after the primary route body stabilizes.")
	return atlasSidePanelErrorBoundary(atlasLazySection(func() ui.Node {
		return html.Div(html.Props{Class: consoleRegionStackClass()},
			routeRevalidationCard("Receiving route refresh", "Re-read the session after reconcile or classification work."),
			receivingDetailStatsIsland(parsePayload, parsePage),
			receivingFormForID(parsePage.Session.ID, parsePayload),
			renderReceivingAttachmentForm(parsePage.Session.ID, parsePayload),
		)
	}, parseFallbackNode, parsePage.Session.ID, parsePage.Session.Status, parsePage.Session.DiscrepancySummary), "Receiving side panel", parseFallbackNode, parsePage.Session.ID, parsePage.Session.Status, parsePage.Session.DiscrepancySummary)
}

func purchaseOrderDetailStatsIsland(parsePayload Payload, parsePage purchaseOrderDetailPage) ui.Node {
	parseRequest, parseOk := StartupRequest(parsePayload, "page")
	parseRequestURL := strings.TrimSpace(parseRequest.URL)
	parseFallbackNode := purchaseOrderDetailStatsContent(parsePage, false, false, "", ui.Handler{})
	if !parseOk || parseRequestURL == "" {
		return parseFallbackNode
	}
	return ui.CreateElement(func() ui.Node {
		parsePanelState := useAtlasState(parsePage)
		parseRefreshingState := useAtlasState(false)
		parseRefreshErrorState := useAtlasState("")
		parseResource := useAtlasStartupPageResource(parsePayload)
		parseResourceState := parseResource.Get()
		parseCachedPage := purchaseOrderDetailPage{}
		if parseResourceState.Ready {
			parseCachedPage = decode[purchaseOrderDetailPage](parseResourceState.Value)
		}
		useAtlasEffect(func() func() {
			if parseResourceState.Ready && strings.TrimSpace(parseCachedPage.Order.ID) != "" {
				parsePanelState.Set(parseCachedPage)
			}
			return nil
		}, parseResourceState.Ready, parseCachedPage.Order.ID, parseCachedPage.Order.Status, parseCachedPage.Order.ETA, parseCachedPage.Order.WarehouseName)
		parseRefreshPanel := ui.UseEvent(func() {
			if parseRefreshingState.Get() {
				return
			}
			parseRefreshingState.Set(true)
			parseRefreshErrorState.Set("")
			go func() {
				parseResult := <-atlasFetch(parseRequestURL, atlasFetchOptions{
					Method: "GET",
					Headers: map[string]any{
						"Accept": "application/json",
					},
				})
				if strings.TrimSpace(parseResult.Error) != "" {
					parseRefreshErrorState.Set(parseResult.Error)
					parseRefreshingState.Set(false)
					return
				}
				parseDecoded := decodeJSONText[purchaseOrderDetailPage](parseResult.Data)
				if strings.TrimSpace(parseDecoded.Order.ID) == "" {
					parseRefreshErrorState.Set("Atlas returned an empty purchase-order panel payload.")
					parseRefreshingState.Set(false)
					return
				}
				parsePanelState.Set(parseDecoded)
				parseResource.Set(parseDecoded)
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Purchase-order panel refreshed",
					Detail: "The vendor-side status panel picked up the latest order snapshot without rerendering the whole route.",
					Tone:   "success",
				})
				parseRefreshingState.Set(false)
			}()
		})
		parseCurrent := parsePanelState.Get()
		return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
			Pending:  !parseResourceState.Ready && parseResourceState.Error == nil,
			Error:    parseResourceState.Error,
			Fallback: parseFallbackNode,
			ErrorFallback: func(parseErr error) ui.Node {
				return detailRailErrorIsland("Purchase-order side panel", parseErr, parseResource.Reload, parseFallbackNode)
			},
			Content: purchaseOrderDetailStatsContent(parseCurrent, parseResourceState.Loading && parseResourceState.Ready, parseRefreshingState.Get(), parseRefreshErrorState.Get(), parseRefreshPanel),
		})
	})
}

func purchaseOrderDetailStatsContent(parsePage purchaseOrderDetailPage, isRefreshing bool, isFetchRefreshing bool, parseRefreshError string, parseRefetch ui.Handler) ui.Node {
	parseChildren := []ui.Node{}
	parseLabel := "Refresh panel snapshot"
	if isFetchRefreshing {
		parseLabel = "Refreshing panel..."
	}
	parseChildren = append(parseChildren, html.Button(html.Props{
		Type:     "button",
		Class:    consoleSecondaryButtonClass(),
		Disabled: isFetchRefreshing,
		OnClick:  parseRefetch,
	}, html.Text(parseLabel)))
	if isRefreshing || isFetchRefreshing {
		// In-flight status is meta prose, not a tinted panel: the snapshot below stays
		// visible and authoritative, so this line must not outweigh it.
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleMetaClass()}, html.Text("Refreshing the purchase-order side panel while the current snapshot stays visible.")))
	}
	if strings.TrimSpace(parseRefreshError) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleFieldErrorClass()}, html.Text(parseRefreshError)))
	}
	parseChildren = append(parseChildren,
		html.Div(html.Props{Class: consoleFactRowClass()},
			statCard("Hub", parsePage.Order.WarehouseName),
			statCard("ETA", parsePage.Order.ETA),
		),
	)
	return html.Div(html.Props{Class: consoleSurfaceClass()}, parseChildren...)
}

func receivingDetailStatsIsland(parsePayload Payload, parsePage receivingDetailPage) ui.Node {
	parseRequest, parseOk := StartupRequest(parsePayload, "page")
	parseRequestURL := strings.TrimSpace(parseRequest.URL)
	parseFallbackNode := receivingDetailStatsContent(parsePage, false, false, "", ui.Handler{})
	if !parseOk || parseRequestURL == "" {
		return parseFallbackNode
	}
	return ui.CreateElement(func() ui.Node {
		parsePanelState := useAtlasState(parsePage)
		parseRefreshingState := useAtlasState(false)
		parseRefreshErrorState := useAtlasState("")
		parseResource := useAtlasStartupPageResource(parsePayload)
		parseResourceState := parseResource.Get()
		parseCachedPage := receivingDetailPage{}
		if parseResourceState.Ready {
			parseCachedPage = decode[receivingDetailPage](parseResourceState.Value)
		}
		useAtlasEffect(func() func() {
			if parseResourceState.Ready && strings.TrimSpace(parseCachedPage.Session.ID) != "" {
				parsePanelState.Set(parseCachedPage)
			}
			return nil
		}, parseResourceState.Ready, parseCachedPage.Session.ID, parseCachedPage.Session.Status, parseCachedPage.Session.WarehouseID)
		parseRefreshPanel := ui.UseEvent(func() {
			if parseRefreshingState.Get() {
				return
			}
			parseRefreshingState.Set(true)
			parseRefreshErrorState.Set("")
			go func() {
				parseResult := <-atlasFetch(parseRequestURL, atlasFetchOptions{
					Method: "GET",
					Headers: map[string]any{
						"Accept": "application/json",
					},
				})
				if strings.TrimSpace(parseResult.Error) != "" {
					parseRefreshErrorState.Set(parseResult.Error)
					parseRefreshingState.Set(false)
					return
				}
				parseDecoded := decodeJSONText[receivingDetailPage](parseResult.Data)
				if strings.TrimSpace(parseDecoded.Session.ID) == "" {
					parseRefreshErrorState.Set("Atlas returned an empty receiving panel payload.")
					parseRefreshingState.Set(false)
					return
				}
				parsePanelState.Set(parseDecoded)
				parseResource.Set(parseDecoded)
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Receiving panel refreshed",
					Detail: "The receiving side panel picked up the latest session snapshot without rerendering the whole route.",
					Tone:   "success",
				})
				parseRefreshingState.Set(false)
			}()
		})
		parseCurrent := parsePanelState.Get()
		return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
			Pending:  !parseResourceState.Ready && parseResourceState.Error == nil,
			Error:    parseResourceState.Error,
			Fallback: parseFallbackNode,
			ErrorFallback: func(parseErr error) ui.Node {
				return detailRailErrorIsland("Receiving side panel", parseErr, parseResource.Reload, parseFallbackNode)
			},
			Content: receivingDetailStatsContent(parseCurrent, parseResourceState.Loading && parseResourceState.Ready, parseRefreshingState.Get(), parseRefreshErrorState.Get(), parseRefreshPanel),
		})
	})
}

func receivingDetailStatsContent(parsePage receivingDetailPage, isRefreshing bool, isFetchRefreshing bool, parseRefreshError string, parseRefetch ui.Handler) ui.Node {
	parseChildren := []ui.Node{}
	parseLabel := "Refresh panel snapshot"
	if isFetchRefreshing {
		parseLabel = "Refreshing panel..."
	}
	parseChildren = append(parseChildren, html.Button(html.Props{
		Type:     "button",
		Class:    consoleSecondaryButtonClass(),
		Disabled: isFetchRefreshing,
		OnClick:  parseRefetch,
	}, html.Text(parseLabel)))
	if isRefreshing || isFetchRefreshing {
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleMetaClass()}, html.Text("Refreshing the receiving side panel while the current snapshot stays visible.")))
	}
	if strings.TrimSpace(parseRefreshError) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleFieldErrorClass()}, html.Text(parseRefreshError)))
	}
	parseChildren = append(parseChildren,
		html.Div(html.Props{Class: consoleFactRowClass()},
			statCard("Hub", parsePage.Session.WarehouseID),
			statCard("Status", parsePage.Session.Status),
		),
	)
	return html.Div(html.Props{Class: consoleSurfaceClass()}, parseChildren...)
}

// detailRailErrorIsland is a COMPONENT because an ERROR FALLBACK is not rendered
// inside the fiber that declared it.
//
// This node is returned from three fallback callbacks:
//   - atlasSidePanelErrorBoundary's ui.ErrorBoundaryProps.ErrorFallback
//   - purchaseOrderDetailStatsIsland's ui.AsyncBoundaryProps.ErrorFallback
//   - receivingDetailStatsIsland's ui.AsyncBoundaryProps.ErrorFallback
//
// The runtime invokes those callbacks from renderBoundaryFallback
// (internal/runtime/error_boundary.go) and safeAsyncBoundaryFallback
// (internal/runtime/async_boundary.go). Neither path arms a current fiber — it is
// recovery bookkeeping, not a component render — so a hook called directly in the
// callback body finds GetCurrentFiber() == nil and panics.
//
// The failure mode is the nastiest kind: the error UI detonates EXACTLY when an
// error occurs. safeAsyncBoundaryFallback re-panics as "async boundary fallback
// panic: …", which propagates to the next boundary out or takes down the page,
// so the original error is replaced by a second, unrelated one.
//
// Returning ui.CreateElement means the callback only BUILDS an element; the
// runtime renders it later inside a fiber, and ui.UseEvent gets its slot.
func detailRailErrorIsland(parseTitle string, parseErr error, parseRetry func(), parseFallbackNode ui.Node) ui.Node {
	return ui.CreateElement(func() ui.Node {
		// A failed panel is an EXCEPTION, and oxide is reserved for exactly that — so
		// the chip and the message carry the tone and the panel itself stays paper. The
		// old version tinted the whole box rose, which made a recoverable panel error
		// look like a page-level failure.
		return html.Div(html.Props{Class: consoleRegionStackClass()},
			html.Div(html.Props{Class: consoleSurfaceClass(), Role: "alert"},
				html.Div(html.Props{Class: consoleChipRowClass()},
					html.Span(html.Props{Class: consoleStatusChipClass(design.ToneException)}, html.Text("FAILED")),
					html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseTitle)),
				),
				html.P(html.Props{Class: consoleFieldErrorClass()}, html.Text(parseErr.Error())),
				html.Button(html.Props{
					Type:    "button",
					Class:   consoleSecondaryButtonClass(),
					OnClick: ui.UseEvent(func() { parseRetry() }),
				}, html.Text("Retry panel")),
			),
			parseFallbackNode,
		)
	})
}

func commentsContent(parsePayload Payload) ui.Node {
	parsePage := decode[commentList](pageData(parsePayload))
	parseNestedNode := commentsNestedPanelForPayload(parsePayload, parsePage)
	if parseOutlet := routeOutletNode(); parseOutlet != nil {
		parseNestedNode = parseOutlet
	}
	parseNodes := []ui.Node{
		commentsSummaryBand(parsePage),
		commentsActionCluster(),
		commentsTable(parsePayload, parsePage.Items),
		commentsModerationFiltersCard(parsePayload, parsePage.Items),
	}
	if parseNestedNode != nil {
		parseNodes = append(parseNodes, parseNestedNode)
	} else {
		parseNodes = append(parseNodes, moderationForm(parsePage.Items, parsePayload), bulkModerationForm(parsePage.Items, parsePayload))
	}
	parseNodes = append(parseNodes,
		inventoryRailCard("Next steps", "Where a buyer question goes when it is really about something else.",
			html.Div(html.Props{Class: consoleStackTightClass()},
				inventoryActionCard("Update marketing copy", "When the issue is messaging, not stock.", "/app/products"),
				inventoryActionCard("Check inventory promise", "When the buyer is really asking about supply or timing.", "/app/inventory"),
				inventoryActionCard("Open warehouse ops", "When the answer depends on a specific hub or lane.", "/app/warehouses"),
			),
		),
	)
	return html.Section(html.Props{Class: consoleRegionStackClass()}, parseNodes...)
}

// settingsContent is a COMPONENT because it calls currentShellPresentationState
// (which calls useAtlasAtom) and because its rail contents are chosen by an
// if/else. As a plain helper both facts leaked upward: the atom subscription
// landed in the route-body fiber, and whether /settings or /settings/appearance
// was active changed how many hooks that fiber consumed.
func settingsContent(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parsePage := decode[settingsPage](pageData(parsePayload))
		parsePresentation := currentShellPresentationState(parsePayload)
		parseRailNodes := []ui.Node{
			settingsSubrouteNavCard(parsePayload),
		}
		if parseNestedPanel := settingsPanelNode(parsePayload); parseNestedPanel != nil {
			parseRailNodes = append(parseRailNodes, parseNestedPanel)
		} else {
			parseRailNodes = append(parseRailNodes,
				renderRoleSwitcherDemoCard(parsePayload),
				preferenceForm(parsePayload),
				savedViewBrowserCard(parsePayload),
				savedViewTransferCard(parsePayload),
				operatorWorkspaceSnapshotCard(parsePayload),
			)
		}
		return html.Section(html.Props{Class: consoleRegionStackClass()},
			append([]ui.Node{
				settingsSummaryBand(parsePage, parsePresentation),
				settingsActionCluster(),
			}, parseRailNodes...)...,
		)
	})
}

func settingsPanelNode(parsePayload Payload) ui.Node {
	if parseOutlet := routeOutletNode(); parseOutlet != nil {
		return parseOutlet
	}
	return SettingsNestedPanel(parsePayload)
}

// SettingsNestedPanel renders route-specific settings content for appearance, locale, and workspace-default sub-routes.
func SettingsNestedPanel(parsePayload Payload) ui.Node {
	parsePath := strings.TrimSpace(parsePayload.Route.Path)
	switch parsePath {
	case RouteSettingsAppearance:
		return settingsAppearancePanel(parsePayload)
	case RouteSettingsLocale:
		return settingsLocalePanel(parsePayload)
	case RouteSettingsWorkspaceDefaults:
		return settingsWorkspaceDefaultsPanel(parsePayload)
	default:
		return nil
	}
}

func settingsSubrouteNavCard(parsePayload Payload) ui.Node {
	parsePath := strings.TrimSpace(parsePayload.Route.Path)
	parseLinks := []struct {
		Label string
		Href  string
	}{
		{Label: "Appearance", Href: RouteSettingsAppearance},
		{Label: "Locale", Href: RouteSettingsLocale},
		{Label: "Workspace defaults", Href: RouteSettingsWorkspaceDefaults},
	}
	// Sub-route navigation reuses the RAIL item vocabulary rather than inventing a
	// third link style: same two states (here / not here), same aria-current, and no
	// bordered pill per link. Inside a Surface these read as ruled lines in a column.
	parseNodes := make([]ui.Node, 0, len(parseLinks)+1)
	parseNodes = append(parseNodes, settingsSubrouteLink("All settings", RouteSettings, parsePath == RouteSettings))
	for _, parseLink := range parseLinks {
		parseNodes = append(parseNodes, settingsSubrouteLink(parseLink.Label, parseLink.Href, parsePath == parseLink.Href))
	}
	return html.Div(html.Props{Class: consoleSurfaceClass()},
		consoleSectionHead("Settings sub-routes", "", ""),
		html.Div(html.Props{Class: consoleStackTightClass()}, parseNodes...),
	)
}

func settingsSubrouteLink(parseLabel string, parseHref string, isCurrent bool) ui.Node {
	parseProps := html.Props{Href: parseHref, Class: design.Class(design.RailLink())}
	if isCurrent {
		parseProps.Class = design.Class(design.RailLinkCurrent())
		parseProps.Aria = map[string]string{"current": "page"}
	}
	return html.A(parseProps, html.Span(html.Props{}, html.Text(parseLabel)))
}

func settingsAppearancePanel(parsePayload Payload) ui.Node {
	return html.Div(html.Props{Class: consoleRegionStackClass()},
		inventoryRailCard("Appearance defaults", "Theme and density for this operator.",
			infoRow("Route", RouteSettingsAppearance),
			infoRow("Controls", "Theme and density"),
			html.A(html.Props{Href: RouteSettings, Class: consoleSecondaryButtonClass()}, html.Text("Back to full settings")),
		),
		preferenceForm(parsePayload),
	)
}

func settingsLocalePanel(parsePayload Payload) ui.Node {
	parseSupportedLocales := strings.Join(parsePayload.I18n.SupportedLocales, ", ")
	if strings.TrimSpace(parseSupportedLocales) == "" {
		parseSupportedLocales = "en"
	}
	return html.Div(html.Props{Class: consoleRegionStackClass()},
		inventoryRailCard("Locale defaults", "Language and reading direction for this operator.",
			infoRow("Route", RouteSettingsLocale),
			infoRow("Current locale", fallback(parsePayload.I18n.Locale, "en")),
			infoRow("Direction", fallback(parsePayload.I18n.Direction, LocaleDirection(parsePayload.I18n.Locale))),
			infoRow("Supported locales", parseSupportedLocales),
			html.A(html.Props{Href: RouteSettings, Class: consoleSecondaryButtonClass()}, html.Text("Back to full settings")),
		),
		preferenceForm(parsePayload),
	)
}

func settingsWorkspaceDefaultsPanel(parsePayload Payload) ui.Node {
	return html.Div(html.Props{Class: consoleRegionStackClass()},
		inventoryRailCard("Workspace defaults", "Default hub, saved-view exchange and workspace snapshot.",
			infoRow("Route", RouteSettingsWorkspaceDefaults),
			infoRow("Default warehouse", fallback(parsePayload.Preferences.DefaultWarehouse, "new-jersey-hub")),
			infoRow("Saved views", fmt.Sprintf("%d", len(parsePayload.SavedViews))),
			html.A(html.Props{Href: RouteSettings, Class: consoleSecondaryButtonClass()}, html.Text("Back to full settings")),
		),
		preferenceForm(parsePayload),
		savedViewBrowserCard(parsePayload),
		savedViewTransferCard(parsePayload),
		operatorWorkspaceSnapshotCard(parsePayload),
	)
}

func commentsSummaryBand(parsePage commentList) ui.Node {
	// Bare counts. "12 open" / "3 queued" / "2 live" put a different unit noun on
	// every figure, which stops the four numbers being comparable — the label already
	// says what they count.
	return html.Div(html.Props{Class: consoleStackClass()},
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: consoleFactRowClass()},
			statCard("Comments", fmt.Sprintf("%d", len(parsePage.Items))),
			statCard("Pending", fmt.Sprintf("%d", countCommentStatus(parsePage.Items, "pending"))),
			statCard("Approved", fmt.Sprintf("%d", countCommentStatus(parsePage.Items, "approved"))),
			statCard("Flagged", fmt.Sprintf("%d", countCommentStatus(parsePage.Items, "flagged"))),
		),
	)
}

func commentsActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		inventoryActionCard("Update marketing copy", "When the issue is messaging, not stock.", "/app/products"),
		inventoryActionCard("Check inventory promise", "When the buyer is really asking about supply or timing.", "/app/inventory"),
		inventoryActionCard("Open warehouse ops", "When the answer depends on a specific hub or lane.", "/app/warehouses"),
	)
}

// CommentsNestedPanel renders nested comments route content for moderation-filter and record-detail routes.
func CommentsNestedPanel(parsePayload Payload) ui.Node {
	parsePage := decode[commentList](pageData(parsePayload))
	return commentsNestedPanelForPayload(parsePayload, parsePage)
}

func commentsNestedPanelForPayload(parsePayload Payload, parsePage commentList) ui.Node {
	parsePath := strings.TrimSpace(parsePayload.Route.Path)
	switch {
	case strings.HasPrefix(parsePath, RouteComments+"/moderation/"):
		parseStatus := strings.TrimSpace(strings.TrimPrefix(parsePath, RouteComments+"/moderation/"))
		if parseStatus == "" {
			return nil
		}
		return commentsModerationNestedPanel(parseStatus, parsePage.Items, parsePayload)
	case strings.HasPrefix(parsePath, RouteComments+"/"):
		parseCommentID := strings.TrimSpace(strings.TrimPrefix(parsePath, RouteComments+"/"))
		if parseCommentID == "" || strings.EqualFold(parseCommentID, "moderation") {
			return nil
		}
		return commentsRecordNestedPanel(parseCommentID, parsePage.Items, parsePayload)
	default:
		return nil
	}
}

func commentsModerationFiltersCard(parsePayload Payload, parseItems []commentRecord) ui.Node {
	parsePath := strings.TrimSpace(parsePayload.Route.Path)
	parseCurrentStatus := commentsModerationStatusFromPath(parsePath)
	type commentsModerationLink struct {
		Label   string
		Href    string
		Count   int
		Current bool
	}
	parseLinks := []commentsModerationLink{
		{Label: "All", Href: RouteComments, Count: len(parseItems), Current: parsePath == RouteComments},
		{Label: "Pending", Href: commentsModerationHref("pending"), Count: countCommentStatus(parseItems, "pending"), Current: strings.EqualFold(parseCurrentStatus, "pending")},
		{Label: "Flagged", Href: commentsModerationHref("flagged"), Count: countCommentStatus(parseItems, "flagged"), Current: strings.EqualFold(parseCurrentStatus, "flagged")},
		{Label: "Approved", Href: commentsModerationHref("approved"), Count: countCommentStatus(parseItems, "approved"), Current: strings.EqualFold(parseCurrentStatus, "approved")},
	}
	parseNodes := make([]ui.Node, 0, len(parseLinks))
	for _, parseLink := range parseLinks {
		parseProps := html.Props{Href: parseLink.Href, Class: design.Class(design.RailLink())}
		if parseLink.Current {
			parseProps.Class = design.Class(design.RailLinkCurrent())
			parseProps.Aria = map[string]string{"current": "page"}
		}
		parseNodes = append(parseNodes, html.A(parseProps,
			html.Span(html.Props{}, html.Text(parseLink.Label)),
			// The count is mono so the four filter counts line up in one column, which
			// is the whole reason to show them next to each other. NOT design.RailCode:
			// that bundle hides itself below the rail breakpoint, which is correct in the
			// rail (the label already carries the meaning) and wrong here, where the
			// number IS the information.
			html.Span(html.Props{Class: consoleCodeClass()}, html.Text(fmt.Sprintf("%d", parseLink.Count))),
		))
	}
	return html.Div(html.Props{Class: consoleSurfaceClass()},
		consoleSectionHead("Moderation filters", "Filter the inbox by moderation state", ""),
		html.Div(html.Props{Class: consoleStackTightClass()}, parseNodes...),
	)
}

func commentsModerationNestedPanel(parseStatus string, parseItems []commentRecord, parsePayload Payload) ui.Node {
	parseVisible := filterCommentsByStatus(parseItems, parseStatus)
	parseStatusLabel := formatCommentStatusLabel(parseStatus)
	parseChildren := []ui.Node{
		inventoryRailCard("Moderation queue", "Scoped to one moderation state.",
			infoRow("Filter", parseStatusLabel),
			infoRow("Visible records", fmt.Sprintf("%d", len(parseVisible))),
			html.A(html.Props{Href: RouteComments, Class: consoleSecondaryButtonClass()}, html.Text("Back to full inbox")),
		),
	}
	if len(parseVisible) > 0 {
		parseChildren = append(parseChildren, moderationForm(parseVisible, parsePayload), bulkModerationForm(parseVisible, parsePayload))
	} else {
		parseChildren = append(parseChildren, inventoryRailCard("No records in filter", "Nothing matches this moderation state right now.",
			html.A(html.Props{Href: RouteComments, Class: consoleSecondaryButtonClass()}, html.Text("Open all comments")),
		))
	}
	return html.Div(html.Props{Class: consoleRegionStackClass()}, parseChildren...)
}

func commentsRecordNestedPanel(parseCommentID string, parseItems []commentRecord, parsePayload Payload) ui.Node {
	parseItem, parseFound := findCommentByID(parseItems, parseCommentID)
	if !parseFound {
		return inventoryRailCard("Comment record unavailable", "This comment is not in the current route payload.",
			html.A(html.Props{Href: RouteComments, Class: consoleSecondaryButtonClass()}, html.Text("Back to comments")),
		)
	}
	return html.Div(html.Props{Class: consoleRegionStackClass()},
		inventoryRailCard("Selected buyer record", "",
			infoRow("Comment ID", parseItem.ID),
			infoRow("Status", formatCommentStatusLabel(parseItem.Status)),
			infoRow("Product SKU", parseItem.ProductSKU),
			infoRow("Author", parseItem.AuthorName+" · "+strings.ReplaceAll(parseItem.AuthorType, "_", " ")),
			html.Hr(html.Props{Class: consoleDividerClass()}),
			// The subject is a heading for this record, the body is what the buyer wrote:
			// Display names the region, Prose carries the sentence.
			html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseItem.Subject)),
			html.P(html.Props{Class: consoleProseClass()}, html.Text(parseItem.Body)),
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.A(html.Props{Href: commentsModerationHref(parseItem.Status), Class: consoleLinkClass()}, html.Text("Open status filter")),
				html.A(html.Props{Href: "/app/products?q=" + url.QueryEscape(parseItem.ProductSKU), Class: consoleLinkClass()}, html.Text("Open product context")),
			),
		),
		moderationForm([]commentRecord{parseItem}, parsePayload),
		bulkModerationForm([]commentRecord{parseItem}, parsePayload),
	)
}

func commentsModerationStatusFromPath(parsePath string) string {
	if !strings.HasPrefix(parsePath, RouteComments+"/moderation/") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(parsePath, RouteComments+"/moderation/"))
}

func commentsModerationHref(parseStatus string) string {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseStatus))
	if parseTrimmed == "" || parseTrimmed == "all" {
		return RouteComments
	}
	return RouteComments + "/moderation/" + parseTrimmed
}

func commentsRecordHref(parsePayload Payload, parseCommentID string) string {
	parseTrimmedID := strings.TrimSpace(parseCommentID)
	if parseTrimmedID == "" {
		return RouteComments
	}
	parseHref := RouteComments + "/" + parseTrimmedID
	parseStatus := commentsModerationStatusFromPath(strings.TrimSpace(parsePayload.Route.Path))
	if parseStatus == "" {
		parseStatus = strings.TrimSpace(firstQueryValue(parsePayload.Route.Query, "status"))
	}
	if parseStatus == "" || strings.EqualFold(parseStatus, "all") {
		return parseHref
	}
	parseValues := url.Values{}
	parseValues.Set("status", parseStatus)
	return parseHref + "?" + parseValues.Encode()
}

func filterCommentsByStatus(parseItems []commentRecord, parseStatus string) []commentRecord {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseStatus))
	if parseTrimmed == "" || parseTrimmed == "all" {
		return append([]commentRecord(nil), parseItems...)
	}
	parseFiltered := make([]commentRecord, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseTrimmed) {
			parseFiltered = append(parseFiltered, parseItem)
		}
	}
	return parseFiltered
}

func findCommentByID(parseItems []commentRecord, parseCommentID string) (commentRecord, bool) {
	parseTrimmedID := strings.TrimSpace(parseCommentID)
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.ID), parseTrimmedID) {
			return parseItem, true
		}
	}
	return commentRecord{}, false
}

func formatCommentStatusLabel(parseStatus string) string {
	parseTrimmed := strings.TrimSpace(parseStatus)
	if parseTrimmed == "" {
		return "Unknown"
	}
	return strings.ReplaceAll(parseTrimmed, "_", " ")
}

func commentsTable(parsePayload Payload, parseItems []commentRecord) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, commentTableRow(parsePayload, parseItem))
	}
	if len(parseRows) == 0 {
		parseRows = append(parseRows, consoleEmptyRow(6, "No buyer questions are waiting."))
	}
	return html.Section(html.Props{Class: consoleStackClass()},
		consoleSectionHead("Buyer question table", "Questions to answer", fmt.Sprintf("%d comments", len(parseItems))),
		consoleQueueTable("Buyer questions",
			html.Tr(html.Props{},
				consoleColumn("Subject"),
				consoleColumn("Question"),
				consoleColumn("SKU"),
				consoleColumn("Author"),
				consoleColumn("Status"),
				consoleColumn("Updated"),
			), parseRows),
	)
}

// commentTableRow is the one queue where a whole SENTENCE has to live in a cell, so
// the question body is a ProseCell — the design system's single documented exception
// to "every table cell is mono". Without it a 30-word question either blows the
// column out or wraps into a wall of fixed-width text, and the fix someone reaches
// for next is making the whole table proportional, which costs the alignment the
// table was chosen for.
func commentTableRow(parsePayload Payload, parseItem commentRecord) ui.Node {
	parseRecordHref := commentsRecordHref(parsePayload, parseItem.ID)
	return html.Tr(html.Props{},
		html.Th(html.Props{},
			html.A(html.Props{Href: parseRecordHref, Class: consoleCellLinkClass()}, html.Text(parseItem.Subject)),
		),
		html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseItem.Body)),
		html.Td(html.Props{}, html.Text(parseItem.ProductSKU)),
		html.Td(html.Props{Class: consoleProseCellClass()}, html.Text(parseItem.AuthorName+" · "+strings.ReplaceAll(parseItem.AuthorType, "_", " "))),
		html.Td(html.Props{},
			html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(strings.ReplaceAll(parseItem.Status, "_", " "))),
		),
		html.Td(html.Props{Class: consoleCellMetaClass()}, html.Text(parseItem.UpdatedAt)),
	)
}

func countCommentStatus(parseItems []commentRecord, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if strings.EqualFold(strings.TrimSpace(parseItem.Status), parseStatus) {
			parseCount++
		}
	}
	return parseCount
}

// settingsSummaryBand shows the CURRENT values of the settings this route edits,
// which is the one page where theme, locale and density belong: here they are the
// state of the thing the form below changes, not dashboard metrics.
func settingsSummaryBand(parsePage settingsPage, parsePresentation shellPresentationState) ui.Node {
	return html.Div(html.Props{Class: consoleStackClass()},
		routeSummaryStrip(parsePage.Summary),
		html.Div(html.Props{Class: consoleFactRowClass()},
			statCard("Theme", fallback(parsePresentation.Theme, "system")),
			statCard("Locale", fallback(parsePresentation.Locale, "en")),
			statCard("Density", fallback(parsePresentation.Density, "compact")),
			statCard("Default hub", fallback(parsePresentation.DefaultWarehouse, "new-jersey-hub")),
		),
	)
}

func settingsActionCluster() ui.Node {
	return internalWorkflowSection("Quick actions", "",
		inventoryActionCard("Open dashboard", "Back to the overview after changing defaults.", "/app/dashboard"),
		inventoryActionCard("Inspect inventory views", "Check saved-view presets against the live workspace.", "/app/inventory"),
		inventoryActionCard("Review comments", "If the next task is moderation rather than configuration.", "/app/comments"),
	)
}

// renderRoleSwitcherDemoCard renders quick role-switch controls so reviewers can inspect role-specific shell behavior.
func renderRoleSwitcherDemoCard(parsePayload Payload) ui.Node {
	parseNext := fallback(parsePayload.Route.Path, "/app/dashboard")
	parseCurrentRole := ""
	if parsePayload.User != nil {
		parseCurrentRole = strings.TrimSpace(parsePayload.User.Role)
	}
	parseRoleCards := []ui.Node{}
	parseRoles := []struct {
		Value string
		Label string
		Copy  string
	}{
		{Value: "inventory_manager", Label: "Inventory manager", Copy: "Prioritizes SKU pressure, saved views, and replenishment sequencing."},
		{Value: "warehouse_supervisor", Label: "Warehouse supervisor", Copy: "Keeps facility backlog, receiving exceptions, and lane fixes front-and-center."},
		{Value: "ops_lead", Label: "Operations lead", Copy: "Balances moderation, transfer planning, and cross-route handoffs."},
	}
	for parseIndex, parseRole := range parseRoles {
		isParseCurrent := strings.EqualFold(parseCurrentRole, parseRole.Value)
		parseRoleChildren := []ui.Node{
			html.HiddenInput("role", parseRole.Value),
			html.HiddenInput("next", parseNext),
		}
		// A hairline between roles rather than a box around each: three bordered boxes
		// inside this Surface is the nested-card move, and the boxes were carrying no
		// information that the current-role chip does not.
		if parseIndex > 0 {
			parseRoleChildren = append(parseRoleChildren, html.Hr(html.Props{Class: consoleDividerClass()}))
		}
		parseRoleHead := []ui.Node{html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseRole.Label))}
		if isParseCurrent {
			parseRoleHead = append(parseRoleHead, html.Span(html.Props{Class: consoleStatusChipClass(design.ToneVerified)}, html.Text("SIGNED IN")))
		}
		parseRoleChildren = append(parseRoleChildren,
			html.Div(html.Props{Class: consoleChipRowClass()}, parseRoleHead...),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseRole.Copy)),
			html.Button(html.Props{Type: "submit", Class: consoleSecondaryButtonClass()}, html.Text("Switch role")),
		)
		parseRoleCards = append(parseRoleCards, html.Form(html.Props{
			Action: "/auth/mock-sign-in",
			Method: "post",
			Class:  consoleStackTightClass(),
		}, parseRoleChildren...))
	}
	return html.Div(html.Props{Class: consoleSurfaceClass()},
		consoleSectionHead("Role switcher", "Sign in as another operator role", ""),
		html.Div(html.Props{Class: consoleStackClass()}, parseRoleCards...),
	)
}

func savedViewBrowserCard(parsePayload Payload) ui.Node {
	if len(parsePayload.SavedViews) == 0 {
		return listCard("Saved views", html.P(html.Props{Class: consoleMetaClass()}, html.Text("No saved views yet.")))
	}
	parseItems := make([]ui.CompositeItem, 0, len(parsePayload.SavedViews))
	for parseIndex, parseSaved := range parsePayload.SavedViews {
		parseItems = append(parseItems, ui.CompositeItem{
			ID:   fmt.Sprintf("atlas-saved-view-%d", parseIndex),
			Text: strings.TrimSpace(parseSaved.Name + " " + parseSaved.Scope + " " + parseSaved.SortKey + " " + parseSaved.SortDirection),
		})
	}
	return ui.CreateElement(func() ui.Node {
		parseNav := ui.UseCompositeNavigation(parseItems, ui.CompositeNavigationOptions{Orientation: "vertical", Loop: true})
		parseListboxKeyDown := ui.UseEvent(func(parseEvent ui.KeyboardEvent) {
			parseNav.OnKeyDown(parseEvent)
		})
		parseActiveIndex := parseNav.ActiveIndex()
		if parseActiveIndex < 0 || parseActiveIndex >= len(parsePayload.SavedViews) {
			parseActiveIndex = 0
		}
		parseActive := parsePayload.SavedViews[parseActiveIndex]
		parseOptions := make([]ui.Node, 0, len(parsePayload.SavedViews))
		for parseIndex2, parseSaved2 := range parsePayload.SavedViews {
			parseCurrentIndex := parseIndex2
			parseCurrentItem := parseItems[parseIndex2]
			// Listbox options reuse the rail's two-state item vocabulary: selected or
			// not, marked by the inset lane bar, with no third "hover-ish" style and no
			// border per option.
			parseClassName := design.Class(design.RailLink())
			if parseNav.IsActive(parseIndex2) {
				parseClassName = design.Class(design.RailLinkCurrent())
			}
			parseOptions = append(parseOptions, html.Div(html.Props{
				ID:    parseCurrentItem.ID,
				Role:  "option",
				Class: parseClassName,
				OnClick: ui.UseEvent(func() {
					parseNav.SetActive(parseCurrentIndex)
				}),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[parseNav.IsActive(parseIndex2)],
				},
			},
				html.Span(html.Props{}, html.Text(parseSaved2.Name)),
				html.Span(html.Props{Class: consoleCodeClass()}, html.Text(parseSaved2.Scope+" · "+parseSaved2.SortKey+"/"+parseSaved2.SortDirection)),
			))
		}
		filterEntries := make([]string, 0, len(parseActive.Filters))
		for parseKey, parseValue := range parseActive.Filters {
			filterEntries = append(filterEntries, parseKey+": "+parseValue)
		}
		sort.Strings(filterEntries)
		filterNodes := make([]ui.Node, 0, len(filterEntries))
		for _, parseEntry := range filterEntries {
			// A saved filter is a machine fact ("warehouse: new-jersey-hub"), so it is a
			// neutral status chip rather than a bespoke pill.
			filterNodes = append(filterNodes, html.Span(html.Props{Class: consoleStatusChipClass(design.ToneNeutral)}, html.Text(parseEntry)))
		}
		parseChildren := []ui.Node{
			consoleSectionHead("Saved views", "", ""),
			html.P(html.Props{Class: consoleFieldHintClass()}, html.Text("Focus the list and use ArrowUp, ArrowDown, Home, End, or typeahead to inspect Atlas workspace presets without leaving the settings route.")),
			// Recess, not another bordered box: the listbox is INPUT to this panel, and
			// pressed paper says "region" where a border would say "second object".
			html.Div(html.Props{
				Role:      "listbox",
				Class:     consoleRecessClass(),
				OnKeyDown: parseListboxKeyDown,
				Aria: map[string]string{
					"activedescendant": parseNav.ActiveDescendant(),
				},
				TabIndex: html.TabIndexZero,
			}, parseOptions...),
			html.Div(html.Props{Class: consoleFactRowClass()},
				statCard("Scope", parseActive.Scope),
				statCard("Sort key", parseActive.SortKey),
				statCard("Direction", parseActive.SortDirection),
			),
		}
		if len(filterNodes) > 0 {
			parseChildren = append(parseChildren, html.Div(html.Props{Class: consoleChipRowClass()}, filterNodes...))
		}
		return html.Div(html.Props{Class: consoleSurfaceClass()}, parseChildren...)
	})
}

func mockSignInContent(parsePayload Payload) ui.Node {
	parsePage := decode[mockSignInPage](pageData(parsePayload))
	if len(parsePage.Roles) == 0 {
		parsePage.Roles = []mockSignInRole{{Value: "inventory_manager", Label: "Inventory Manager", Description: "Default Atlas internal operator role."}}
	}
	parseRoleForms := make([]ui.Node, 0, len(parsePage.Roles))
	for parseIndex, parseRole := range parsePage.Roles {
		parseRoleChildren := []ui.Node{
			html.Input(html.Props{Type: "hidden", Name: "role", Value: parseRole.Value}),
			html.Input(html.Props{Type: "hidden", Name: "next", Value: fallback(parsePage.Next, "/app/dashboard")}),
		}
		if parseIndex > 0 {
			parseRoleChildren = append(parseRoleChildren, html.Hr(html.Props{Class: consoleDividerClass()}))
		}
		parseRoleChildren = append(parseRoleChildren,
			html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseRole.Label)),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseRole.Description)),
			// Starting the session IS the action of this screen, so each role's submit is
			// the primary control on its own form.
			html.Button(html.Props{Type: "submit", Class: consolePrimaryButtonClass()}, html.Text("Start session")),
		)
		parseRoleForms = append(parseRoleForms, html.Form(html.Props{
			Action: "/auth/mock-sign-in",
			Method: "post",
			Class:  consoleStackTightClass(),
		}, parseRoleChildren...))
	}
	return html.Section(html.Props{Class: consoleRegionStackClass()},
		featureCard("Mock internal access", fallback(parsePage.Message, "Start a mock Atlas session to open the operator console.")),
		html.Div(html.Props{Class: consoleSurfaceClass()},
			consoleSectionHead("Choose a role", "", ""),
			html.Div(html.Props{Class: consoleStackClass()}, parseRoleForms...),
		),
		listCard("Recovery path",
			infoRow("Next route", fallback(parsePage.Next, "/app/dashboard")),
			infoRow("Session model", "Cookie-backed mock operator role"),
		),
	)
}

func recoveryContent(parsePayload Payload) ui.Node {
	parsePage := decode[recoveryPage](pageData(parsePayload))
	parseChildren := []ui.Node{
		featureCard(fallback(parsePage.Title, "Route recovery"), fallback(parsePage.Message, "This route is not part of the current Atlas demo route set.")),
		// Getting back to a working route is THE action on a recovery screen.
		html.A(html.Props{Href: fallback(parsePage.RecoveryHref, "/"), Class: consolePrimaryButtonClass()}, html.Text(fallback(parsePage.RecoveryLabel, "Back to Atlas"))),
	}
	if strings.TrimSpace(parsePage.Detail) != "" {
		parseChildren = append(parseChildren, listCard("Recovery detail", html.P(html.Props{Class: consoleProseClass()}, html.Text(parsePage.Detail))))
	}
	return html.Section(html.Props{Class: consoleRegionStackClass()}, parseChildren...)
}

// fallbackContent dumps whatever payload a route did carry. It is a debug surface,
// so it is Recess (pressed paper, the "raw input" texture) with mono lines — not a
// list of bordered boxes pretending each key is a card.
func fallbackContent(parsePayload Payload) ui.Node {
	parseEntries := sortedMapStrings(parsePayload.Data)
	parseNodes := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseNodes = append(parseNodes, html.P(html.Props{Class: consoleDataClass()}, html.Text(parseEntry)))
	}
	if len(parseNodes) == 0 {
		return listCard("Route data", html.P(html.Props{Class: consoleMetaClass()}, html.Text("This route carried no payload.")))
	}
	return listCard("Route data", html.Div(html.Props{Class: consoleRecessClass()}, parseNodes...))
}

// routeSummaryStrip is the route's own metric band: label, figure, one line of
// context. Each metric is a STACK, not a bordered box — four boxes in a row is the
// KPI-wall look this conversion removed, and the figure reads as the important thing
// because it is mono at lede size, not because it has a frame.
func routeSummaryStrip(parseSummary pageSummary) ui.Node {
	if len(parseSummary.Items) == 0 {
		return html.Div(html.Props{})
	}
	parseNodes := make([]ui.Node, 0, len(parseSummary.Items))
	for _, parseItem := range parseSummary.Items {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: consoleStackTightClass()},
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseItem.Label)),
			html.P(html.Props{Class: consoleFigureClass()}, html.Text(parseItem.Value)),
			html.P(html.Props{Class: consoleMetaClass()}, html.Text(parseItem.Detail)),
		))
	}
	return html.Div(html.Props{Class: consoleStackClass()},
		html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(fallback(parseSummary.Headline, "Route summary"))),
		html.Div(html.Props{Class: consoleFactRowClass()}, parseNodes...),
	)
}

func featureCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: consoleSurfaceClass()},
		html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseTitle)),
		html.P(html.Props{Class: consoleProseClass()}, html.Text(parseCopy)),
	)
}

func publicFeatureCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: publicGlassCardClass()},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-stone-300"}, html.Text(parseCopy)),
	)
}

func publicMetricCard(parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: publicMetricSurfaceClass()},
		html.P(html.Props{Class: "text-xl font-bold tracking-[-0.03em] text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "mt-2 text-sm leading-6 text-stone-300"}, html.Text(parseCopy)),
	)
}

func publicSignalPill(parseLabel string) ui.Node {
	return html.Div(html.Props{Class: publicSignalPillClass()}, html.Text(parseLabel))
}

func publicStatusClass(parseStatus string) string {
	parseBase := "rounded-full border px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.22em]"
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "in_stock", "healthy", "approved", "available":
		return parseBase + " border-emerald-400/30 bg-emerald-400/12 text-emerald-200"
	case "low_stock", "pending", "submitted", "in_review":
		return parseBase + " border-amber-400/30 bg-amber-400/12 text-amber-200"
	default:
		return parseBase + " border-white/12 bg-white/8 text-stone-300"
	}
}

func publicStatusLabel(parseStatus string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(parseStatus)), "_", " ")
}

// productPrimaryActionForm branches on product status and emits FIVE fields in one
// arm and TWO in the others. That is only safe because publicInput and
// publicTextarea are components: the field hooks live in their own fibers, so the
// number of hooks this function contributes to its caller is zero regardless of
// which arm runs. If you ever add a direct hook call to this body, it must move
// inside a ui.CreateElement first — a data-dependent hook count is a corrupted
// fiber.
func productPrimaryActionForm(parseProduct productCard, parsePayload Payload) ui.Node {
	switch strings.TrimSpace(strings.ToLower(parseProduct.Status)) {
	case "in_stock", "healthy", "approved", "available":
		return publicFormCard("Start a project quote", "Capture pricing, volume, and install timing while this item is ready to support an active project.", "/api/public/products/"+parseProduct.Slug+"/quote-requests", "Request pricing", parsePayload.CSRF, []ui.Node{
			publicInput("requester_name", "Name"),
			publicInput("company_name", "Company"),
			publicInput("email", "Email"),
			publicInput("quantity", "Quantity"),
			publicTextarea("note", "Project note"),
		})
	case "low_stock", "pending", "submitted", "in_review":
		return publicFormCard("Reserve upcoming availability", "Hold intent against the next recovery window so the team can plan around constrained stock.", "/api/public/products/"+parseProduct.Slug+"/restock-requests", "Reserve availability", parsePayload.CSRF, []ui.Node{
			publicInput("email", "Email"),
			publicInput("preferred_warehouse_id", "Preferred delivery region"),
		})
	default:
		return publicFormCard("Notify me when available", "Stay attached to this SKU without forcing a buyer to think in replenishment or internal operations terms.", "/api/public/products/"+parseProduct.Slug+"/restock-requests", "Notify me", parsePayload.CSRF, []ui.Node{
			publicInput("email", "Email"),
			publicInput("preferred_warehouse_id", "Preferred delivery region"),
		})
	}
}

func productSecondaryActionCard(parseProduct productCard) ui.Node {
	parseStatus := strings.TrimSpace(strings.ToLower(parseProduct.Status))
	if parseStatus == "in_stock" || parseStatus == "healthy" || parseStatus == "approved" || parseStatus == "available" {
		return html.A(html.Props{Href: "/warehouses", Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] transition hover:border-amber-300/35 hover:bg-white/10"},
			html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("Check delivery for your region")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Compare regional delivery options before you lock in your project timeline and installation plan.")),
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-amber-700"}, html.Text("See regional delivery options")),
		)
	}
	return html.A(html.Props{Href: "/shop?category=" + url.QueryEscape(parseProduct.Category), Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] transition hover:border-amber-300/35 hover:bg-white/10"},
		html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("See similar options")),
		html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("If this SKU cannot support the buyer timeline, keep momentum by browsing adjacent in-category systems.")),
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.25em] text-amber-700"}, html.Text("Browse alternatives")),
	)
}

// availabilityPrimaryActionForm has the same shape as productPrimaryActionForm: the
// available-stock branch renders five fields, the other two render two. Safe for the
// same reason — the field primitives are components, so this function contributes no
// hooks of its own to whichever fiber renders it.
func availabilityPrimaryActionForm(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	if parseAvailability.Available > 0 {
		return publicFormCard("Request pricing for this region", "Capture pricing and project timing while this warehouse can still support the current demand window.", "/api/public/products/"+parseAvailability.Product.Slug+"/quote-requests", "Start quote", parsePayload.CSRF, []ui.Node{
			publicInput("requester_name", "Name"),
			publicInput("company_name", "Company"),
			publicInput("email", "Email"),
			publicInput("quantity", "Quantity"),
			publicTextarea("note", "Project note"),
		})
	}
	parseTitle := "Notify me when available"
	parseCopy := "Stay tied to this regional lane without making the buyer restate the product and warehouse context later."
	parseButton := "Notify me"
	if parseAvailability.Inbound > 0 {
		parseTitle = "Reserve upcoming availability"
		parseCopy = "Hold this buyer against the next inbound recovery window for the current warehouse lane."
		parseButton = "Reserve availability"
	}
	return publicFormCard(parseTitle, parseCopy, "/api/public/products/"+parseAvailability.Product.Slug+"/restock-requests", parseButton, parsePayload.CSRF, []ui.Node{
		publicInput("email", "Email"),
		publicInputWithValue("preferred_warehouse_id", "Preferred delivery region", parseAvailability.Warehouse.ID),
	})
}

func availabilityQuestionActionForm(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	return publicFormCard("Ask about delivery or fit", "Keep regional delivery, installation, and substitution questions attached to the actual availability context.", "/api/public/products/"+parseAvailability.Product.Slug+"/comments", "Send question", parsePayload.CSRF, []ui.Node{
		publicInput("author_name", "Name"),
		publicInput("subject", "Subject"),
		publicTextarea("body", "Question"),
	})
}

// statCard is misnamed and is deliberately NOT a card any more: it is a label above
// a figure, sized to sit in a fact row.
//
// The name is kept because inventory_cms.go and products_cms.go call it; renaming it
// is a separate, mechanical change in files this pass does not own. The VALUE is
// Data at lede step — a count, a code or a date is a machine fact, and mono is what
// lets a row of them be compared.
func statCard(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: consoleStackTightClass()},
		html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseLabel)),
		html.P(html.Props{Class: consoleFigureClass()}, html.Text(parseValue)),
	)
}

func listCard(parseTitle string, parseChildren ...ui.Node) ui.Node {
	if len(parseChildren) == 0 {
		parseChildren = []ui.Node{html.P(html.Props{Class: consoleMetaClass()}, html.Text("Nothing here yet."))}
	}
	parseContent := []ui.Node{html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseTitle))}
	parseContent = append(parseContent, parseChildren...)
	return html.Div(html.Props{Class: consoleSurfaceClass()}, parseContent...)
}

func publicFormCard(parseTitle, parseCopy, parseAction, parseSubmitLabel string, parseCsrfToken string, parseFields []ui.Node) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseTitle)),
	}
	if strings.TrimSpace(parseCopy) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseCopy)))
	}
	parseChildren = append(parseChildren, prependCSRFToken(parseCsrfToken)...)
	parseChildren = append(parseChildren, parseFields...)
	parseChildren = append(parseChildren, html.Button(html.Props{Type: "submit", Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(fallback(strings.TrimSpace(parseSubmitLabel), "Submit"))))
	return html.Form(html.Props{Action: parseAction, Method: "post", Class: "grid gap-4 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, parseChildren...)
}

type preferenceFormState struct {
	Theme              string
	Locale             string
	Density            string
	DefaultWarehouseID string
}

type moderationActionFormState struct {
	Status string
	Reason string
}

type bulkModerationActionFormState struct {
	IDs    string
	Status string
	Reason string
}

type savedViewImportFormState struct {
	ViewsJSON string
}

type workspaceSnapshotFormState struct {
	WorkspaceSnapshotJSON string
}

type transferFormState struct {
	SourceWarehouseID      string
	DestinationWarehouseID string
	Reason                 string
	RecommendedBy          string
}

type receivingFormState struct {
	Status             string
	DiscrepancySummary string
}

type receivingResolutionWorkflowState struct {
	Status             string
	DiscrepancySummary string
	Stage              string
	LastEditedField    string
}

type receivingResolutionWorkflowAction struct {
	Field string
	Value string
}

func normalizeReceivingResolutionWorkflowState(parseState receivingResolutionWorkflowState) receivingResolutionWorkflowState {
	parseStatus := strings.TrimSpace(strings.ToLower(parseState.Status))
	parseSummary := strings.TrimSpace(parseState.DiscrepancySummary)
	// Classification rule keeps closeout strict: "closed" requires a discrepancy or closeout note,
	// while non-closed statuses remain in active review stages.
	switch {
	case parseStatus == "closed" && parseSummary == "":
		parseState.Stage = "closeout-needs-note"
	case parseStatus == "closed":
		parseState.Stage = "closeout-ready"
	case parseStatus == "review":
		parseState.Stage = "discrepancy-review"
	default:
		parseState.Stage = "classification-open"
	}
	return parseState
}

func reduceReceivingResolutionWorkflowState(parseState receivingResolutionWorkflowState, parseAction receivingResolutionWorkflowAction) receivingResolutionWorkflowState {
	switch parseAction.Field {
	case "status":
		parseState.Status = parseAction.Value
	case "discrepancy_summary":
		parseState.DiscrepancySummary = parseAction.Value
	}
	parseState.LastEditedField = parseAction.Field
	return normalizeReceivingResolutionWorkflowState(parseState)
}

func preferenceForm(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		// Inside the component, not above it: currentShellPresentationState calls
		// useAtlasAtom, so calling it before ui.CreateElement put this card's atom
		// subscription in the caller's fiber instead of its own.
		parsePresentation := currentShellPresentationState(parsePayload)
		parseForm := ui.UseForm(preferenceFormState{
			Theme:              fallback(parsePresentation.Theme, "dark"),
			Locale:             fallback(parsePresentation.Locale, "en"),
			Density:            fallback(parsePresentation.Density, "compact"),
			DefaultWarehouseID: fallback(parsePresentation.DefaultWarehouse, "new-jersey-hub"),
		})
		parseTransition := useAtlasTransition()
		parseValue := parseForm.Get()
		parseChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Operator preferences")),
			boundInputWithValue("theme", "Theme", parseValue.Theme, "Theme", parseForm),
			boundInputWithValue("locale", "Locale", parseValue.Locale, "Locale", parseForm),
			boundTransitionSelectWithValue("density", "Density", parseValue.Density, "Density", []optionItem{{"compact", "Compact"}, {"comfortable", "Comfortable"}}, parseForm, parseTransition),
			preferenceDensityPreviewCard(parseValue.Density, parseTransition.Pending()),
			boundInputWithValue("default_warehouse_id", "Default warehouse", parseValue.DefaultWarehouseID, "DefaultWarehouseID", parseForm),
			submitButton("Save preferences"),
		}
		return html.Form(html.Props{Action: "/api/app/preferences", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func moderationForm(parseItems []commentRecord, parsePayload Payload) ui.Node {
	parseId := ""
	if len(parseItems) > 0 {
		parseId = parseItems[0].ID
	}
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(moderationActionFormState{Status: "approved", Reason: "Reviewed by Atlas operations."})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Review first queued buyer question")),
			boundInputWithValue("status", "Status", parseValue.Status, "Status", parseForm),
			boundTextareaWithValue("reason", "Reason", parseValue.Reason, "Reason", parseForm),
			html.Button(html.Props{Type: "button", Class: consolePrimaryButtonClass(), OnClick: parseOpenConfirm}, html.Text("Review decision")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-comment-review-confirm", "Confirm buyer review", "Atlas keeps keyboard focus inside the moderation confirmation step until you either cancel or apply the review.", "Apply review", parseCloseConfirm,
				statCard("Status", parseValue.Status),
				html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseValue.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/comments/" + parseId + "/moderate", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func bulkModerationForm(parseItems []commentRecord, parsePayload Payload) ui.Node {
	parseIds := make([]string, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseIds = append(parseIds, parseItem.ID)
	}
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(bulkModerationActionFormState{
			IDs:    strings.Join(parseIds, ","),
			Status: "approved",
			Reason: "Bulk-reviewed from the visible Atlas buyer inbox queue.",
		})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Bulk review visible queue")),
			html.Input(html.Props{Type: "hidden", Name: "ids", Value: parseValue.IDs}),
			boundInputWithValue("status", "Status", parseValue.Status, "Status", parseForm),
			boundTextareaWithValue("reason", "Reason", parseValue.Reason, "Reason", parseForm),
			html.Button(html.Props{Type: "button", Class: consolePrimaryButtonClass(), OnClick: parseOpenConfirm}, html.Text("Review bulk action")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-bulk-review-confirm", "Confirm bulk moderation", "The buyer-inbox bulk action now traps focus inside its confirmation step instead of leaving focus scattered behind the modal.", "Apply bulk review", parseCloseConfirm,
				statCard("Items", fmt.Sprintf("%d", len(parseItems))),
				statCard("Status", parseValue.Status),
				html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseValue.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/comments/bulk-moderate", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func savedViewForm(parsePayload Payload) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Create saved view")),
		input("name", "Name"),
		inputWithValue("scope", "Scope", "inventory"),
		inputWithValue("sort_key", "Sort key", "available"),
		inputWithValue("sort_direction", "Sort direction", "asc"),
		inputWithValue("density", "Density", "compact"),
		inputWithValue("warehouse_id", "Warehouse", "new-jersey-hub"),
		textareaWithValue("filters_json", "Filters JSON", `{"warehouse":"new-jersey-hub"}`),
		submitButton("Save view"),
	}
	return html.Form(html.Props{Action: "/api/app/saved-views", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
}

// savedViewTransferDefaultPayload is a PURE builder: no hooks, so it is safe to
// call from anywhere. The presentation state it needs is passed in rather than read
// with a hook, which is what lets the hook stay inside the component below.
func savedViewTransferDefaultPayload(parsePayload Payload, parsePresentation shellPresentationState) string {
	type savedViewTransferItem struct {
		Name          string `json:"name"`
		Scope         string `json:"scope"`
		SortKey       string `json:"sortKey"`
		SortDirection string `json:"sortDirection"`
		Density       string `json:"density"`
		WarehouseID   string `json:"warehouseId"`
		FiltersJSON   string `json:"filtersJSON"`
	}
	parseItems := make([]savedViewTransferItem, 0, len(parsePayload.SavedViews))
	for _, parseSaved := range parsePayload.SavedViews {
		parseFiltersJSON, parseErr := json.Marshal(parseSaved.Filters)
		if parseErr != nil {
			parseFiltersJSON = []byte(`{}`)
		}
		parseItems = append(parseItems, savedViewTransferItem{
			Name:          parseSaved.Name,
			Scope:         parseSaved.Scope,
			SortKey:       parseSaved.SortKey,
			SortDirection: parseSaved.SortDirection,
			Density:       fallback(parsePresentation.Density, "compact"),
			WarehouseID:   fallback(parseSaved.Filters["warehouse"], parsePresentation.DefaultWarehouse),
			FiltersJSON:   string(parseFiltersJSON),
		})
	}
	parseExportPayload := map[string]any{"items": parseItems}
	parseEncoded, parseErr2 := json.MarshalIndent(parseExportPayload, "", "  ")
	if parseErr2 != nil {
		parseEncoded = []byte(`{"items":[]}`)
	}
	return string(parseEncoded)
}

func savedViewTransferCard(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		// currentShellPresentationState calls useAtlasAtom. It used to be called on
		// the line ABOVE this ui.CreateElement, which meant the atom subscription was
		// taken out of the CALLER's fiber (settingsContent's, and therefore App's)
		// while every other hook in this card lived here. That is the subtle version
		// of the hooks-outside-render bug: the hook does run inside *a* fiber, just
		// not this component's, so the card silently borrowed a slot from a fiber
		// whose hook count then depended on which settings rail was rendered.
		parsePresentation := currentShellPresentationState(parsePayload)
		parseDefaultImportPayload := savedViewTransferDefaultPayload(parsePayload, parsePresentation)
		parseForm := ui.UseForm(savedViewImportFormState{ViewsJSON: parseDefaultImportPayload})
		parseValidator := useAtlasWorkerTask[savedViewImportValidationRequest, savedViewImportValidationProgress, savedViewImportValidationResult](interop.WorkerOptions{
			URL:   "/assets/script/atlas-saved-view-import-worker.js",
			Ready: true,
			Name:  "atlas-saved-view-import",
		}, "validate-saved-view-import")
		useAtlasEffect(func() func() {
			parseTrimmed := strings.TrimSpace(parseForm.Get().ViewsJSON)
			if parseTrimmed == "" {
				parseValidator.Cancel()
				return nil
			}
			parseValidator.Start(savedViewImportValidationRequest{Text: parseForm.Get().ViewsJSON})
			return nil
		}, parseForm.Get().ViewsJSON)
		parseValidationState := parseValidator.Get()
		parseValue := parseForm.Get()
		parseChildren := []ui.Node{
			html.Div(html.Props{Class: consoleSurfaceClass()},
				html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Export saved views")),
				html.P(html.Props{Class: consoleProseFineClass()}, html.Text("Download or copy the current saved-view payload so Atlas presets can move between runs without rebuilding them by hand.")),
				html.A(html.Props{Href: "/api/app/saved-views/export", Class: consoleSecondaryButtonClass()}, html.Text("Open export payload")),
				// The payload is INPUT to this panel, so it is Recess (pressed paper) plus
				// mono, and it scrolls sideways rather than wrapping JSON.
				html.Pre(html.Props{Class: design.Class(design.Recess(), design.Data(design.StepFine), []css.Rule{css.Raw("overflow-x", "auto")})}, html.Text(parseDefaultImportPayload)),
			),
		}
		parseImportChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Import saved views")),
			boundTextareaWithValue("views_json", "Saved-view payload", parseValue.ViewsJSON, "ViewsJSON", parseForm),
			savedViewImportValidationCard(parseValidationState),
			submitButton("Import saved views"),
		}
		parseChildren = append(parseChildren, html.Form(html.Props{Action: "/api/app/saved-views/import", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseImportChildren...)...))
		return html.Div(html.Props{Class: consoleRegionStackClass()}, parseChildren...)
	})
}

type savedViewImportValidationRequest struct {
	Text string `json:"text"`
}

type savedViewImportValidationProgress struct {
	Percent int    `json:"percent"`
	Stage   string `json:"stage"`
}

type savedViewImportValidationResult struct {
	Valid        bool     `json:"valid"`
	ItemCount    int      `json:"itemCount"`
	ScopeCount   int      `json:"scopeCount"`
	Scopes       []string `json:"scopes"`
	Preview      []string `json:"preview"`
	InvalidCount int      `json:"invalidCount"`
	Warnings     []string `json:"warnings"`
}

// savedViewImportValidationCard is a validation result, so it uses the FORM error
// and hint voices rather than four differently tinted boxes. It sits inside the
// import form's Surface, which is exactly the place a second bordered box would be
// wrong: a Recess (pressed paper, no border, no radius) says "region" instead.
func savedViewImportValidationCard(parseState atlasWorkerTaskState[savedViewImportValidationProgress, savedViewImportValidationResult]) ui.Node {
	if parseState.Error != nil {
		return html.Div(html.Props{Class: consoleRecessClass(), Role: "alert"},
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.Span(html.Props{Class: consoleStatusChipClass(design.ToneException)}, html.Text("FAILED")),
				html.P(html.Props{Class: consoleFieldLabelClass()}, html.Text("Worker validation failed")),
			),
			html.P(html.Props{Class: consoleFieldErrorClass()}, html.Text(parseState.Error.Error())),
		)
	}
	if parseState.Running {
		parseStage := "validating saved-view payload"
		parseProgress := "Worker running"
		if parseState.ProgressReady {
			parseStage = fallback(parseState.Progress.Stage, parseStage)
			parseProgress = fmt.Sprintf("%d%% complete", parseState.Progress.Percent)
		}
		return html.Div(html.Props{Class: consoleRecessClass()},
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.Span(html.Props{Class: consoleStatusChipClass(design.TonePending)}, html.Text("RUNNING")),
				html.P(html.Props{Class: consoleFieldLabelClass()}, html.Text("Worker validation in progress")),
			),
			html.P(html.Props{Class: consoleFieldHintClass()}, html.Text(parseProgress+" · "+parseStage)),
		)
	}
	if parseState.Ready {
		parseSummary := "Payload is ready to import."
		if !parseState.Value.Valid {
			parseSummary = "Payload needs fixes before import."
		}
		parsePreview := "No saved-view names detected yet."
		if len(parseState.Value.Preview) > 0 {
			parsePreview = strings.Join(parseState.Value.Preview, " · ")
		}
		parseScopeSummary := "No scopes detected."
		if len(parseState.Value.Scopes) > 0 {
			parseScopeSummary = strings.Join(parseState.Value.Scopes, ", ")
		}
		parseTone := design.ToneVerified
		if !parseState.Value.Valid {
			parseTone = design.ToneException
		}
		parseChildren := []ui.Node{
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.Span(html.Props{Class: consoleStatusChipClass(parseTone)}, html.Text(map[bool]string{true: "READY", false: "NEEDS FIXES"}[parseState.Value.Valid])),
				html.P(html.Props{Class: consoleFieldLabelClass()}, html.Text("Worker import preview")),
			),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseSummary)),
			html.Div(html.Props{Class: consoleFactRowClass()},
				statCard("Items", fmt.Sprintf("%d", parseState.Value.ItemCount)),
				statCard("Scopes", fmt.Sprintf("%d", parseState.Value.ScopeCount)),
				statCard("Invalid", fmt.Sprintf("%d", parseState.Value.InvalidCount)),
			),
			html.P(html.Props{Class: consoleFieldHintClass()}, html.Text("Preview: "+parsePreview)),
			html.P(html.Props{Class: consoleFieldHintClass()}, html.Text("Scopes: "+parseScopeSummary)),
		}
		if len(parseState.Value.Warnings) > 0 {
			parseWarnings := make([]ui.Node, 0, len(parseState.Value.Warnings))
			for _, parseWarning := range parseState.Value.Warnings {
				parseWarnings = append(parseWarnings, html.P(html.Props{Class: consoleFieldErrorClass()}, html.Text(parseWarning)))
			}
			parseChildren = append(parseChildren, html.Div(html.Props{Class: consoleStackTightClass()}, parseWarnings...))
		}
		return html.Div(html.Props{Class: consoleRecessClass()}, parseChildren...)
	}
	return html.Div(html.Props{Class: consoleRecessClass()},
		html.P(html.Props{Class: consoleFieldLabelClass()}, html.Text("Worker import preview")),
		html.P(html.Props{Class: consoleFieldHintClass()}, html.Text("Atlas will validate the saved-view import payload in a dedicated worker as you edit it.")),
	)
}

func operatorWorkspaceSnapshotCard(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(workspaceSnapshotFormState{WorkspaceSnapshotJSON: operatorWorkspaceSnapshotJSON(parsePayload)})
		parseValue := parseForm.Get()
		return html.Div(html.Props{Class: consoleSurfaceClass()},
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Export workspace snapshot")),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text("Copy the current operator shell, route workspace, and saved-view snapshot when a reviewer or another operator needs the same Atlas context without opening extra tooling surfaces.")),
			boundTextareaWithValue("workspace_snapshot_json", "Workspace snapshot", parseValue.WorkspaceSnapshotJSON, "WorkspaceSnapshotJSON", parseForm),
		)
	})
}

func operatorWorkspaceSnapshotJSON(parsePayload Payload) string {
	type savedViewSnapshotItem struct {
		Name          string            `json:"name"`
		Scope         string            `json:"scope"`
		SortKey       string            `json:"sortKey"`
		SortDirection string            `json:"sortDirection"`
		Density       string            `json:"density"`
		WarehouseID   string            `json:"warehouseId"`
		Filters       map[string]string `json:"filters"`
	}
	parsePresentation := currentShellPresentationState(parsePayload)
	parseWorkspace := currentRouteWorkspaceState(parsePayload)
	parseSavedViews := make([]savedViewSnapshotItem, 0, len(parsePayload.SavedViews))
	for _, parseSaved := range parsePayload.SavedViews {
		parseSavedViews = append(parseSavedViews, savedViewSnapshotItem{
			Name:          parseSaved.Name,
			Scope:         parseSaved.Scope,
			SortKey:       parseSaved.SortKey,
			SortDirection: parseSaved.SortDirection,
			Density:       parsePresentation.Density,
			WarehouseID:   fallback(parseSaved.Filters["warehouse"], parsePresentation.DefaultWarehouse),
			Filters:       parseSaved.Filters,
		})
	}
	parseExportPayload := map[string]any{
		"route": map[string]any{
			"path":   parsePayload.Route.Path,
			"screen": parsePayload.Route.Screen,
			"query":  parsePayload.Route.Query,
		},
		"shellPresentation": parsePresentation,
		"routeWorkspace":    parseWorkspace,
		"savedViews":        parseSavedViews,
	}
	parseEncoded, parseErr := json.MarshalIndent(parseExportPayload, "", "  ")
	if parseErr != nil {
		return `{"error":"workspace snapshot encode failed"}`
	}
	return string(parseEncoded)
}

func transferForm(parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(transferFormState{
			SourceWarehouseID:      "nevada-hub",
			DestinationWarehouseID: "new-jersey-hub",
			Reason:                 "Support east-coast promise windows",
			RecommendedBy:          "Atlas Hydration Demo",
		})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Create transfer")),
			boundInputWithValue("source_warehouse_id", "Source warehouse", parseValue.SourceWarehouseID, "SourceWarehouseID", parseForm),
			boundInputWithValue("destination_warehouse_id", "Destination warehouse", parseValue.DestinationWarehouseID, "DestinationWarehouseID", parseForm),
			boundInputWithValue("reason", "Reason", parseValue.Reason, "Reason", parseForm),
			boundInputWithValue("recommended_by", "Recommended by", parseValue.RecommendedBy, "RecommendedBy", parseForm),
			html.Button(html.Props{Type: "button", Class: consolePrimaryButtonClass(), OnClick: parseOpenConfirm}, html.Text("Review transfer")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-transfer-confirm", "Confirm transfer plan", "Review the transfer before Atlas posts it so focus stays inside the confirmation step until you dismiss or submit.", "Create transfer", parseCloseConfirm,
				statCard("From", parseValue.SourceWarehouseID),
				statCard("To", parseValue.DestinationWarehouseID),
				html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseValue.Reason)),
			),
		}
		return html.Form(html.Props{Action: "/api/app/transfers", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func thresholdForm(parseItem inventoryRow, parsePayload Payload) ui.Node {
	parseChildren := []ui.Node{
		html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Adjust thresholds")),
		inputWithValue("warehouse_id", "Warehouse", parseItem.WarehouseID),
		inputWithValue("reorder_point", "Reorder point", "18"),
		inputWithValue("safety_stock", "Safety stock", "9"),
		submitButton("Save threshold"),
	}
	return html.Form(html.Props{Action: "/api/app/inventory/" + parseItem.SKU + "/threshold", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
}

func receivingForm(parsePayload Payload) ui.Node {
	return receivingFormForID("rcv-illinois-001", parsePayload)
}

func receivingFormForID(parseSessionID string, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(receivingFormState{
			Status:             "closed",
			DiscrepancySummary: "Short shipment recorded and available stock released.",
		})
		parseConfirmOpen := ui.UseState(false)
		parseValue := parseForm.Get()
		parsePrevious := ui.UsePrevious(parseValue)
		parseWorkflow := ui.UseReducer(reduceReceivingResolutionWorkflowState, normalizeReceivingResolutionWorkflowState(receivingResolutionWorkflowState{
			Status:             parseValue.Status,
			DiscrepancySummary: parseValue.DiscrepancySummary,
		}))
		parseWorkflowState := parseWorkflow.Get()
		parseOpenConfirm := ui.UseEvent(func() { parseConfirmOpen.Set(true) })
		parseCloseConfirm := func() { parseConfirmOpen.Set(false) }
		parseChildren := []ui.Node{
			receivingResolutionSummaryCard(parseWorkflowState),
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Reconcile receiving")),
			receivingWorkflowInputWithValue("status", "Status", parseValue.Status, "Status", parseForm, parseWorkflow),
			receivingWorkflowTextareaWithValue("discrepancy_summary", "Discrepancy summary", parseValue.DiscrepancySummary, "DiscrepancySummary", parseForm, parseWorkflow),
			html.Button(html.Props{Type: "button", Class: consolePrimaryButtonClass(), OnClick: parseOpenConfirm}, html.Text("Review closeout")),
			atlasConfirmationDialog(parseConfirmOpen.Get(), "atlas-receiving-confirm", "Confirm receiving closeout", "Atlas keeps focus inside the discrepancy confirmation step until you cancel or submit the reconciliation.", "Close session", parseCloseConfirm,
				statCard("Status", parseValue.Status),
				html.P(html.Props{Class: consoleProseFineClass()}, html.Text(fallback(parseValue.DiscrepancySummary, "No discrepancy note entered."))),
			),
		}
		if parseChangeSummary := receivingDraftChangeSummary(parseValue, parsePrevious); parseChangeSummary != nil {
			parseChildren = append([]ui.Node{parseChangeSummary}, parseChildren...)
		}
		return html.Form(html.Props{Action: "/api/app/receiving/" + parseSessionID + "/reconcile", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

// renderReceivingAttachmentForm renders a receiving evidence upload form for photos and supporting docs.
//
// It is a COMPONENT for the same reason as purchaseOrderStatusForm: it is built
// inside receivingDetailRail's atlasLazySection loader, which runs on a goroutine
// with no current fiber, and its textareaWithValue field calls ui.UseId.
func renderReceivingAttachmentForm(parseSessionID string, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Attach receiving evidence")),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text("Upload dock photos, carrier notes, or signed paperwork so discrepancy closeout has supporting context.")),
			html.Label(html.Props{Class: consoleFieldClass()},
				html.Span(html.Props{Class: consoleFieldLabelClass()}, html.Text("Evidence files")),
				html.Input(html.Props{Name: "attachment", Type: "file", Class: consoleInputClass(), Raw: map[string]any{"multiple": true, "accept": ".png,.jpg,.jpeg,.webp,.pdf,.txt,.csv"}}),
			),
			textareaWithValue("note", "Attachment note", "Shipment seal mismatch documented at dock door B."),
			submitButton("Upload evidence"),
		}
		return html.Form(html.Props{Action: "/api/app/receiving/" + parseSessionID + "/attachments", Method: "post", Class: consoleSurfaceClass(), Raw: map[string]any{"encType": "multipart/form-data"}}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func receivingDraftChangeSummary(parseCurrent receivingFormState, parsePrevious ui.Previous[receivingFormState]) ui.Node {
	if !parsePrevious.Ok() {
		return nil
	}
	parsePrior := parsePrevious.Get()
	parseSummary := ""
	switch {
	case strings.TrimSpace(parsePrior.Status) != strings.TrimSpace(parseCurrent.Status):
		parseSummary = "Status changed from " + fallback(parsePrior.Status, "unset") + " to " + fallback(parseCurrent.Status, "unset")
	case strings.TrimSpace(parsePrior.DiscrepancySummary) != strings.TrimSpace(parseCurrent.DiscrepancySummary):
		parseSummary = "Discrepancy notes were updated for the current reconciliation draft."
	}
	if parseSummary == "" {
		return nil
	}
	// A draft-change note is a hint about the form it sits in, not a banner: field
	// hint voice, no box, so it cannot outweigh the fields it describes.
	return html.Div(html.Props{Class: consoleStackTightClass()},
		html.P(html.Props{Class: consoleFieldLabelClass()}, html.Text("Recent reconcile edit")),
		html.P(html.Props{Class: consoleFieldHintClass()}, html.Text(parseSummary)),
	)
}

// receivingResolutionSummaryCard reports where the closeout draft currently stands.
//
// The stage now drives a status TONE as well as a label, which is the point of a
// semantic palette: "closeout needs note" is a thing a human must fix, so it is the
// exception tone, and "closeout ready" is verified. The old version painted all four
// stages the same cyan, so the one stage that blocks submit looked like the three
// that do not.
func receivingResolutionSummaryCard(parseState receivingResolutionWorkflowState) ui.Node {
	parseStageLabel := "Classification open"
	parseStageCopy := "Waiting on the reconcile outcome and the discrepancy note."
	parseTone := design.ToneNeutral
	switch parseState.Stage {
	case "closeout-ready":
		parseStageLabel = "Closeout ready"
		parseStageCopy = "Status and discrepancy notes line up. This session can close cleanly."
		parseTone = design.ToneVerified
	case "closeout-needs-note":
		parseStageLabel = "Closeout needs note"
		parseStageCopy = "A closed session still has to say what happened. Add a discrepancy or closeout note before submitting."
		parseTone = design.ToneException
	case "discrepancy-review":
		parseStageLabel = "Discrepancy review"
		parseStageCopy = "Still under review, so this counts as an open exception."
		parseTone = design.TonePending
	}
	parseEdited := "Staged from the current receiving defaults."
	if strings.TrimSpace(parseState.LastEditedField) != "" {
		parseEdited = "Last updated field: " + strings.ReplaceAll(parseState.LastEditedField, "_", " ") + "."
	}
	return html.Div(html.Props{Class: consoleStackTightClass()},
		html.Div(html.Props{Class: consoleChipRowClass()},
			html.Span(html.Props{Class: consoleStatusChipClass(parseTone)}, html.Text(parseStageLabel)),
		),
		html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseStageCopy)),
		html.P(html.Props{Class: consoleFieldHintClass()}, html.Text(parseEdited)),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// FIELD PRIMITIVES — these call ui.UseId/ui.UseEvent and stay HELPERS on purpose.
// Read this before "fixing" them, because the obvious fix does not work.
//
// # The general rule, and the exception these are
//
// The rule everywhere else in this file is: a function that calls a hook must BE
// a component, i.e. return ui.CreateElement(...) so the runtime owns its fiber.
// The field builders below are the documented exception, and the reason is a
// property of ui.CreateElement's component-identity model.
//
// # Why ui.CreateElement cannot wrap a repeated leaf helper
//
// ui.CreateElement resolves a component to a cached *runtime.ComponentType handle
// keyed by the function's identity, and that identity comes from
// runtime.FuncForPC(codePointer).Name() — see ui/component_handle_shared.go,
// getComponentHandle and describeComponentIdentity. Every closure created at the
// SAME source line therefore maps to the SAME handle, and each new closure
// overwrites that handle's implementation renderer.
//
// Consequence: a helper that returns ui.CreateElement(func() ui.Node { … }) works
// only while ONE instance is live. Render three of them as siblings and all three
// resolve to one handle whose implementation is the LAST closure created, so all
// three render the last call's captured arguments.
//
// That is not hypothetical. Wrapping boundInputWithValue this way made the
// operator preferences form render "Default warehouse" three times — the theme and
// locale inputs both took the third call's captures — and dropped name="locale"
// from the submitted form entirely. phase_d_internal_regression_test.go catches it.
//
// # So what is still wrong, and what protects it
//
// Because these stay helpers, their hook slots are consumed from the CALLER's
// fiber. Two consequences remain open (see the report accompanying this pass):
//
//   - A caller that emits a different number of fields per branch changes its own
//     hook count between renders (productPrimaryActionForm and
//     availabilityPrimaryActionForm switch on status and emit 2 or 5 fields).
//   - publicProductActionRail invokes one closure twice (desktop column + mobile
//     drawer), so the same logical field draws two different ids and the drawer's
//     aria-labelledby points at the desktop label.
//
// What keeps these from being fatal today: every FORM that contains them is a
// component (purchaseOrderStatusForm, renderReceivingAttachmentForm,
// receivingFormForID, preferenceForm, moderationForm, …), so the varying hook
// count is contained inside a fiber that owns nothing else, and no hook is
// declared after the fields in those bodies.
//
// The real fix needs per-instance component identity — a named component taking
// props, or an identity key — not an anonymous closure. Do not reintroduce the
// closure wrapper.
// ─────────────────────────────────────────────────────────────────────────────

func receivingWorkflowInputWithValue(parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[receivingFormState], parseWorkflow ui.Reducer[receivingResolutionWorkflowState, receivingResolutionWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Input(html.Props{
			ID: parseId, Name: parseName, Value: parseValue, Class: consoleInputDataClass(),
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				parseNext := parseEvent.GetValue()
				parseForm.SetField(parseField, parseNext)
				parseWorkflow.Dispatch(receivingResolutionWorkflowAction{Field: parseName, Value: parseNext})
			}),
			Raw: map[string]any{"aria-labelledby": parseId + "-label"},
		}),
	)
}

func receivingWorkflowTextareaWithValue(parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[receivingFormState], parseWorkflow ui.Reducer[receivingResolutionWorkflowState, receivingResolutionWorkflowAction]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Textarea(html.Props{
			ID: parseId, Name: parseName, Value: parseValue, Class: consoleInputClass(),
			OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) {
				parseNext := parseEvent.GetValue()
				parseForm.SetField(parseField, parseNext)
				parseWorkflow.Dispatch(receivingResolutionWorkflowAction{Field: parseName, Value: parseNext})
			}),
			Raw: map[string]any{"aria-labelledby": parseId + "-label"},
		}, html.Text(parseValue)),
	)
}

// purchaseOrderStatusForm is a COMPONENT because the fields it renders own hooks.
//
// It is reached from purchaseOrderDetailRail's atlasLazySection loader — a
// goroutine with no current fiber — so building its subtree eagerly there used to
// execute inputWithValue's and textareaWithValue's ui.UseId immediately and panic.
// Both field primitives are components now, so the ui.CreateElement here is
// belt-and-braces: it guarantees this form owns a fiber even if a future edit adds
// a direct hook call to the body.
func purchaseOrderStatusForm(parseId string, parseStatus string, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseChildren := []ui.Node{
			html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Update order status")),
			inputWithValue("status", "Status", fallback(parseStatus, "submitted")),
			textareaWithValue("note", "Note", "Validated by Atlas operations."),
			submitButton("Update order"),
		}
		return html.Form(html.Props{Action: "/api/app/purchase-orders/" + parseId + "/status", Method: "post", Class: consoleSurfaceClass()}, prependCSRFToken(parsePayload.CSRF, parseChildren...)...)
	})
}

func prependCSRFToken(parseToken string, parseChildren ...ui.Node) []ui.Node {
	if strings.TrimSpace(parseToken) == "" {
		return parseChildren
	}
	parseFieldName, parseFieldValue := ui.NewCSRFToken(parseToken).FormField()
	parseResult := make([]ui.Node, 0, len(parseChildren)+1)
	parseResult = append(parseResult, html.HiddenInput(parseFieldName, parseFieldValue))
	parseResult = append(parseResult, parseChildren...)
	return parseResult
}

func input(parseName, parseLabel string) ui.Node {
	return inputWithValue(parseName, parseLabel, "")
}

func publicInput(parseName, parseLabel string) ui.Node {
	return publicInputWithValue(parseName, parseLabel, "")
}

func publicInputWithValue(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, Class: "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]", Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func inputWithValue(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, Class: consoleInputDataClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func boundInputWithValue[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: consoleInputDataClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func boundTransitionSelectWithValue[T any](parseName, parseLabel, parseValue, parseField string, parseOptions []optionItem, parseForm ui.Form[T], parseTransition atlasTransition) ui.Node {
	parseId := ui.UseId()
	parseChildren := make([]ui.Node, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		isParseSelected := strings.EqualFold(strings.TrimSpace(parseValue), strings.TrimSpace(parseOption.Value)) || (strings.TrimSpace(parseValue) == "" && parseOption.Value == "")
		parseChildren = append(parseChildren, html.Option(html.Props{Value: parseOption.Value, Selected: isParseSelected}, html.Text(parseOption.Label)))
	}
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Select(html.Props{
			ID:   parseId,
			Name: parseName,
			OnChange: ui.UseEvent(func(parseEvent ui.ChangeEvent) {
				atlasSetFormFieldInTransition(parseForm, parseField, parseEvent.GetValue())
			}),
			Class: consoleInputDataClass(),
			Raw:   map[string]any{"aria-labelledby": parseId + "-label", "data-transition": parseTransition.Pending()},
		}, parseChildren...),
	)
}

func publicTextarea(parseName, parseLabel string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
		html.Span(html.Props{ID: parseId + "-label"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Class: "min-h-28 rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]", Raw: map[string]any{"aria-labelledby": parseId + "-label"}}),
	)
}

func textareaWithValue(parseName, parseLabel, parseValue string) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Class: consoleInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func boundTextareaWithValue[T any](parseName, parseLabel, parseValue, parseField string, parseForm ui.Form[T]) ui.Node {
	parseId := ui.UseId()
	return html.Label(html.Props{Class: consoleFieldClass()},
		html.Span(html.Props{ID: parseId + "-label", Class: consoleFieldLabelClass()}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseId, Name: parseName, Value: parseValue, OnInput: ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField(parseField, parseEvent.GetValue()) }), Class: consoleInputClass(), Raw: map[string]any{"aria-labelledby": parseId + "-label"}}, html.Text(parseValue)),
	)
}

func preferenceDensityPreviewCard(parseValue string, isPending bool) ui.Node {
	parseDensity := atlasNormalizedFilterValue(parseValue)
	if parseDensity == "" {
		parseDensity = "compact"
	}
	parseClassName := "atlas-density-preview atlas-density-preview-compact"
	parseCopy := "Compact density keeps Atlas information-dense for operators working wide tables and queue summaries."
	if parseDensity == "comfortable" {
		parseClassName = "atlas-density-preview atlas-density-preview-comfortable"
		parseCopy = "Comfortable density increases whitespace so route summaries and action rails breathe more during review."
	}
	if isPending {
		parseCopy = "Applying the next density preview in a transition so the settings form stays responsive."
	}
	// The atlas-density-preview classes are NOT design-system classes and are not
	// converted: they are defined in client/atlas-commerce-os.html and are what the
	// preview is demonstrating. Everything around them is.
	return html.Div(html.Props{Class: consoleRecessClass()},
		html.P(html.Props{Class: consoleFieldLabelClass()}, html.Text("Density preview")),
		html.Div(html.Props{Class: parseClassName},
			// The chip uppercases in CSS, so the TEXT stays human-cased — the DOM keeps a
			// readable string for tests and assistive tech either way.
			html.Span(html.Props{Class: consoleStatusChipClass(design.TonePending)}, html.Text(strings.Title(parseDensity))),
			html.Span(html.Props{Class: consoleStatusChipClass(design.ToneNeutral)}, html.Text("Warehouse shell")),
			html.Span(html.Props{Class: consoleStatusChipClass(design.ToneNeutral)}, html.Text("Inventory rail")),
		),
		html.P(html.Props{Class: consoleFieldHintClass()}, html.Text(parseCopy)),
	)
}

func submitButton(parseLabel string) ui.Node {
	return html.Button(html.Props{Type: "submit", Class: consolePrimaryButtonClass()}, html.Text(parseLabel))
}

// commentNodes, transferNodes and receivingNodes are compact list renderers. Each
// entry is a Stack separated from the next by space, not a bordered row: a list of
// four bordered boxes inside a panel is the nested-card shape, and none of the three
// carries enough per-item structure to need a frame.
func commentNodes(parseItems []commentRecord) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: consoleStackTightClass()},
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.P(html.Props{Class: consoleSectionTitleClass()}, html.Text(parseItem.Subject)),
				html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(parseItem.Status)),
			),
			html.P(html.Props{Class: consoleCodeClass()}, html.Text(parseItem.ProductSKU+" · "+parseItem.AuthorName+" · "+strings.ReplaceAll(parseItem.AuthorType, "_", " "))),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseItem.Body)),
		))
	}
	return parseNodes
}

func internalWorkflowSection(parseTitle, parseCopy string, parseCards ...ui.Node) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-workflow-sheet"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		parseHead := []ui.Node{html.P(html.Props{Class: consoleEyebrowClass()}, html.Text(parseTitle))}
		if strings.TrimSpace(parseCopy) != "" {
			parseHead = append(parseHead, html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseCopy)))
		}
		parseChildren := []ui.Node{
			html.Div(html.Props{Class: consoleStackTightClass()}, parseHead...),
			// The shortcuts wrap in a Cluster instead of switching between a hidden grid
			// and a drawer at 1024px. The drawer stays because it is behaviour this pass
			// does not change, but the shortcuts are readable at 380px without it.
			html.Div(html.Props{Class: consoleClusterClass()}, parseCards...),
			html.Button(html.Props{Type: "button", Class: consoleQuietButtonClass(), OnClick: parseOpenDrawer}, html.Text("Open quick actions")),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: consoleStackClass()},
				html.Div(html.Props{Class: consoleSplitRowClass()},
					html.Div(html.Props{Class: consoleStackTightClass()},
						html.P(html.Props{Class: consoleEyebrowClass()}, html.Text("Quick actions")),
						html.P(html.Props{ID: parseTitleID, Class: consoleSectionTitleClass()}, html.Text(parseTitle)),
						html.P(html.Props{ID: parseDescriptionID, Class: consoleProseFineClass()}, html.Text(parseCopy)),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: consoleQuietButtonClass(), OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: consoleStackClass()}, parseCards...),
			)),
		}
		return html.Div(html.Props{Class: consoleSurfaceClass()}, parseChildren...)
	})
}

// internalWorkflowCard is a shortcut, and parseStep is now OPTIONAL.
//
// The dashboard used to pass "Action 1" … "Action 4" for four independent
// shortcuts, which implies an order that does not exist — an operator reading
// "Action 3" looks for what Action 2 was. Callers that have a real sequence (the
// guided demo) still pass a step; everyone else passes "" and the eyebrow is
// omitted rather than rendered blank.
func internalWorkflowCard(parseStep, parseTitle, parseCopy, parseHref string) ui.Node {
	parseChildren := []ui.Node{}
	if strings.TrimSpace(parseStep) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: consoleCodeClass()}, html.Text(parseStep)))
	}
	parseChildren = append(parseChildren,
		// The title carries the link's colour and underline: in a dense console, colour
		// alone does not separate a link from a status value, and dropping the underline
		// is the most common way a screen becomes unusable for a colourblind operator.
		html.P(html.Props{Class: design.Class(design.SectionTitle(), design.Link())}, html.Text(parseTitle)),
		html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseCopy)),
	)
	// A shortcut is a LINK, not a card. The trailing "Open workflow" line is gone,
	// because a link that also says "open" is telling the reader what a link is.
	return html.A(html.Props{
		Href:  parseHref,
		Class: design.Class(design.Stack(design.Space1), []css.Rule{css.Raw("text-decoration", "none"), css.MaxWidth(css.Rem(22))}),
	}, parseChildren...)
}

func transferNodes(parseItems []transferRecord) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: consoleStackTightClass()},
			html.Div(html.Props{Class: consoleChipRowClass()},
				// The lane is two hub codes and an arrow, in mono, so a stack of lanes
				// aligns on the arrow.
				html.P(html.Props{Class: consoleDataClass()}, html.Text(parseItem.SourceWarehouseID+" → "+parseItem.DestinationWarehouse)),
				html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(parseItem.Status)),
			),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text(parseItem.Reason)),
		))
	}
	return parseNodes
}

func receivingNodes(parseItems []receivingRecord) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: consoleStackTightClass()},
			html.Div(html.Props{Class: consoleChipRowClass()},
				html.P(html.Props{Class: consoleDataClass()}, html.Text(parseItem.ID)),
				html.Span(html.Props{Class: consoleStatusChipClass(atlasStatusTone(parseItem.Status))}, html.Text(parseItem.Status)),
			),
			html.P(html.Props{Class: consoleProseFineClass()}, html.Text(fallback(parseItem.DiscrepancySummary, "No discrepancies recorded."))),
		))
	}
	return parseNodes
}

// infoRow is a label/value line, not a box: the label sits left in the field-label
// voice and the value right in mono, so a stack of them reads as a spec sheet with
// its values in one column.
func infoRow(parsePrimary string, parseSecondary string) ui.Node {
	return html.Div(html.Props{Class: design.Class(design.SplitRow(design.Space3), []css.Rule{css.Raw("align-items", "baseline")})},
		html.Span(html.Props{Class: consoleFieldLabelClass()}, html.Text(parsePrimary)),
		html.Span(html.Props{Class: consoleDataClass()}, html.Text(parseSecondary)),
	)
}

func activeNavLink(parseCurrentPath string, parseHref string) bool {
	parseCurrent := strings.TrimSpace(parseCurrentPath)
	parseTarget := strings.TrimSpace(parseHref)
	if parseCurrent == parseTarget {
		return true
	}
	if parseTarget == "/" {
		return false
	}
	return strings.HasPrefix(parseCurrent, parseTarget+"/")
}

func routeSummary(parsePath string) string {
	switch {
	case parsePath == "/":
		return "Atlas helps teams discover workspace products, compare regional delivery options, and request pricing without losing context."
	case strings.HasPrefix(parsePath, "/shop/"):
		return "This product page keeps pricing, delivery questions, and next steps together so buyers can decide with confidence."
	case parsePath == "/shop":
		return "Browse workspace systems with clear pricing cues, delivery context, and straightforward next steps."
	case strings.HasPrefix(parsePath, "/warehouses"):
		return "Explore Atlas delivery regions, compare service levels, and open the warehouse view that best fits your project timing."
	case strings.HasPrefix(parsePath, "/app"):
		return "Internal routes keep buyer inbox, merchandising, inventory, transfers, receiving, and preferences inside one server-backed Atlas console."
	default:
		return "Atlas route payload hydrated successfully from the server bootstrap."
	}
}

func payloadDataValue(parsePayload Payload, parseKey string) any {
	if parseRequest, parseOk := parsePayload.Requests[strings.TrimSpace(parseKey)]; parseOk && len(parseRequest.Data) > 0 {
		if parseValue, parseExists := parseRequest.Data[strings.TrimSpace(parseKey)]; parseExists {
			return parseValue
		}
	}
	if parsePayload.Data == nil {
		return nil
	}
	return parsePayload.Data[strings.TrimSpace(parseKey)]
}

func pageData(parsePayload Payload) any {
	return payloadDataValue(parsePayload, "page")
}

func decode[T any](parseValue any) T {
	var parseResult T
	if parseValue == nil {
		return parseResult
	}
	parseEncoded, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return parseResult
	}
	_ = json.Unmarshal(parseEncoded, &parseResult)
	return parseResult
}

func decodeJSONText[T any](parseValue string) T {
	var parseResult T
	if strings.TrimSpace(parseValue) == "" {
		return parseResult
	}
	_ = json.Unmarshal([]byte(parseValue), &parseResult)
	return parseResult
}

func fallback(parseValue string, parseDefaultValue string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return parseDefaultValue
	}
	return parseTrimmed
}

func firstQueryValue(parseValues map[string][]string, parseKey string) string {
	if len(parseValues[parseKey]) == 0 {
		return ""
	}
	return parseValues[parseKey][0]
}

func formatPrice(parseCents int) string {
	return fmt.Sprintf("$%0.2f", float64(parseCents)/100)
}

func sortedMapStrings(parseValues map[string]any) []string {
	if len(parseValues) == 0 {
		return nil
	}
	parseKeys := make([]string, 0, len(parseValues))
	for parseKey := range parseValues {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseLines := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseLines = append(parseLines, fmt.Sprintf("%s: %v", parseKey2, parseValues[parseKey2]))
	}
	return parseLines
}
