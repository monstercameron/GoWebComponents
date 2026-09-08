package design

import "github.com/monstercameron/GoWebComponents/v6/css"

// Buttons, links and form fields.
//
// # Three buttons. Not four.
//
// ButtonPrimary, ButtonSecondary, ButtonQuiet — that is the whole set, and the count
// is the design decision:
//
//   - Primary is safety yellow and there is ONE per view. Not one per Surface, not
//     one per row: one per view. If a screen has two primary buttons it has no
//     primary action, which is exactly the state the old design was in with a yellow
//     button, a teal badge and an orange eyebrow all competing.
//   - Secondary is a lane-blue outline: every other real action.
//   - Quiet is text only: cancel, dismiss, "show all", anything whose presence should
//     not cost visual weight.
//
// There is deliberately NO ButtonDanger. A destructive action in Atlas is a
// ButtonSecondary that opens a confirmation, and the confirmation is where oxide
// appears — on the consequence, not on the trigger. A permanently red button teaches
// operators to ignore red, which is expensive in a system whose whole job is flagging
// discrepancies.
//
// There is also no size scale. One height (34px), because a design system with three
// button sizes has three opinions about density and every screen picks a different
// one.

var buttonBase = css.Rules(
	css.Display.InlineFlex,
	css.Items.Center,
	css.Justify.Center,
	css.Gap(Space2),
	css.MinHeight(css.Px(34)),
	css.PaddingX(Space3),
	css.PaddingY(Space2),
	css.Rounded(RadiusTag),
	// A button label names an action, so it is Display: condensed, uppercase, tracked.
	// It is the same voice as a table header and a page title, which is what makes the
	// interface read as one document.
	Display(StepMicro),
	css.Cursor.Pointer,
	css.UserSelect.None,
	css.Raw("text-decoration", "none"),
	css.Raw("white-space", "nowrap"),
	// A transparent border on every variant, so the three styles have identical box
	// metrics. Without it, adding a border on hover shifts the label by 1px and the
	// button appears to twitch — and a quiet button next to an outlined one sits 2px
	// wider.
	css.Border(HairlineWidth, css.Transparent),
	withMotion(css.PropColors, css.Ms(120)),
	// The a11y floor is already global (see installAccessibilityFloor); this restates
	// it locally with an explicit offset because a button's ring needs to clear its
	// own border rather than sit on it.
	css.FocusVisible(
		css.Outline(ManifestRuleWidth, Lane()),
		css.OutlineOffset(css.Px(2)),
	),
	// PRINT: buttons leave the paper. A control that cannot be actuated is not
	// information, and an action row printed on a receipt is three words of noise at the
	// exact place a reader is looking for the total.
	//
	// This has to live HERE rather than in print.go's global layer, and the reason is a
	// cascade fact worth remembering: `@media print { button { display: none } }` is
	// specificity 0-0-1 and loses to this bundle's own `.c-x { display: inline-flex }` at
	// 0-1-0. An at-rule does not raise specificity. The global rule in print.go is still
	// worth having as a floor for a <button> that never got a design class, but a styled
	// button can only be hidden by the bundle that styled it. The first pass shipped only
	// the global rule and every button printed anyway.
	css.Media(printQuery, css.Display.None),
	css.Disabled(
		css.Cursor.NotAllowed,
		// Disabled uses graphite-on-sunk-paper rather than opacity. Opacity on a
		// button also fades its focus ring and its border, so a disabled button can
		// become invisible on one surface and fine on another.
		//
		// The border stays on Hairline rather than Edge on purpose. WCAG 1.4.11
		// excludes inactive components from the 3:1 boundary requirement, and a
		// disabled control SHOULD read as fainter than a live one — the label still
		// clears 4.5:1 (graphite on paper-sunk), so the state is legible without the
		// border having to shout it.
		css.Bg(PaperSunk()),
		css.TextColor(Graphite()),
		css.Border(HairlineWidth, Hairline()),
	),
)

var buttonPrimaryBundle = clip(css.Rules(
	buttonBase,
	// Near-black on signal yellow. Yellow is a light color: paper-on-yellow would be
	// roughly 1.6:1 and unreadable, so the primary button is dark-on-bright, which
	// also makes it the highest-contrast element on the page — appropriate for the one
	// thing you most want clicked.
	//
	// The label is SignalFgOnly, not Ink, and that distinction was a measured bug
	// rather than a preference. Ink() looks right in review and is right in light mode
	// (9.18:1) but ink INVERTS in dark mode while signal does not, so the dark primary
	// button was #E8E7E1 on #F0C244 — 1.36:1, i.e. an unreadable button on the one
	// control the design system spends its loudest color on. SignalFgOnly is pinned
	// near-black in both themes: 9.18:1 light, 10.79:1 dark.
	css.Bg(PrimaryActionOnly()),
	css.TextColor(SignalFgOnly()),
	css.Border(HairlineWidth, PrimaryActionOnly()),
	// Hover stamps a keyline inside the button instead of shifting its color. There is
	// no darker-yellow token, and inventing one per interaction state is how a
	// nine-color palette becomes a twenty-seven-color palette.
	//
	// The keyline is drawn in SignalFgOnly for the same reason the label is: it lives
	// ON the yellow, so it must not invert with the theme. An ink keyline would go
	// near-white on yellow in dark mode and the hover state would vanish.
	css.Hover(css.Shadow(css.ShadowToken("inset 0 0 0 2px var("+TokenSignalFg+")"))),
	css.Active(css.Shadow(css.ShadowToken("inset 0 0 0 3px var("+TokenSignalFg+")"))),
))

// ButtonPrimary is the single primary action for a view: signal yellow, ink label.
//
// One per view. If you are writing the second one, one of them is a ButtonSecondary.
func ButtonPrimary() []css.Rule { return buttonPrimaryBundle }

var buttonSecondaryBundle = clip(css.Rules(
	buttonBase,
	css.Bg(Paper()),
	css.TextColor(Lane()),
	css.Border(HairlineWidth, Lane()),
	// Full inversion on hover. An outlined control has room to invert without
	// inventing a token, and the swap reads as a stamp landing — which is the right
	// feel for warehouse paperwork.
	css.Hover(css.Bg(Lane()), css.TextColor(Paper())),
))

// ButtonSecondary is every action that is not THE action: lane-blue outline that
// inverts on hover.
func ButtonSecondary() []css.Rule { return buttonSecondaryBundle }

var buttonQuietBundle = clip(css.Rules(
	buttonBase,
	css.Bg(css.Transparent),
	css.TextColor(Graphite()),
	css.Hover(css.Bg(PaperSunk()), css.TextColor(Ink())),
))

// ButtonQuiet is the tertiary action: cancel, dismiss, "show all", a row-level
// action. No border, no fill until hover.
//
// Its job is to be reachable without being visible, so it costs a queue of 40 rows
// nothing to have a per-row action.
func ButtonQuiet() []css.Rule { return buttonQuietBundle }

// --- links --------------------------------------------------------------------

var linkBundle = clip(css.Rules(
	css.TextColor(Lane()),
	css.Raw("text-decoration", "underline"),
	// Underline offset and thinner strokes so the underline does not collide with
	// descenders — an underline through a "g" makes the letter ambiguous, which in a
	// linked SKU or hub code is a real read error.
	css.Raw("text-underline-offset", "0.18em"),
	css.Raw("text-decoration-thickness", "1px"),
	withMotion(css.PropColors, css.Ms(120)),
	css.Hover(css.TextColor(Ink()), css.Raw("text-decoration-thickness", "2px")),
	css.FocusVisible(css.Outline(ManifestRuleWidth, Lane()), css.OutlineOffset(css.Px(2))),
))

// Link is an inline link: lane blue, underlined.
//
// It stays underlined at rest. In a dense manifest full of mono identifiers, color
// alone does not distinguish a link from a status value, and removing the underline is
// the most common way an app becomes unusable for a colorblind operator.
func Link() []css.Rule { return linkBundle }

// --- fields -------------------------------------------------------------------

var fieldBundle = clip(css.Rules(
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(Space1),
	css.MinWidth(css.Zero),
))

// Field is the label-above-control wrapper.
//
// Labels go above, never beside and never floating inside. Beside costs horizontal
// room the console does not have at 380px; floating-inside labels disappear the moment
// the field has a value, which is exactly when an operator re-reads the form to check
// what they typed.
func Field() []css.Rule { return fieldBundle }

var fieldLabelBundle = clip(css.Rules(
	Display(StepMicro),
	css.TextColor(Graphite()),
))

// FieldLabel is the label above a control: condensed uppercase graphite, the same
// voice as a table header, because a label and a column header do the same job.
//
// Use a real <label for="..."> element. The style carries no association; the
// attribute does.
func FieldLabel() []css.Rule { return fieldLabelBundle }

var inputBase = css.Rules(
	css.W(css.Full),
	css.MinWidth(css.Zero),
	css.MinHeight(css.Px(34)),
	css.PaddingX(Space3),
	css.PaddingY(Space2),
	css.Bg(Paper()),
	css.TextColor(Ink()),
	// Edge, not Hairline. An unfilled text field is identified by its border and by
	// nothing else, so this border is "visual information required to identify a user
	// interface component" (WCAG 1.4.11) and owes 3:1. On the hairline it measured
	// 1.5:1 — the fields were, literally, invisible boxes that reviewers found because
	// they already knew where they were. See Edge in tokens.go for the full argument.
	css.Border(HairlineWidth, Edge()),
	css.Rounded(RadiusTag),
	// appearance:none so a <select> and an <input> can share one primitive and one box
	// metric on every platform. It also removes the UA's dropdown arrow on a <select>,
	// so a select needs its own indicator — draw it on the wrapper, not here, because a
	// background-image arrow on the control itself would collide with the focus border.
	css.Raw("appearance", "none"),
	withMotion(css.PropColors, css.Ms(120)),
	// Two focus treatments on purpose. The border-color change (:focus) marks the
	// field for a mouse user; the ring (:focus-visible) is the keyboard affordance.
	// Text inputs match :focus-visible on click too, per spec, so both fire together
	// there — which is fine, they reinforce.
	css.Focus(css.Border(HairlineWidth, Lane())),
	css.FocusVisible(
		css.Outline(ManifestRuleWidth, Lane()),
		css.OutlineOffset(css.Px(1)),
	),
	css.Disabled(
		css.Bg(PaperSunk()),
		css.TextColor(Graphite()),
		css.Cursor.NotAllowed,
	),
)

var inputBundle = clip(css.Rules(
	inputBase,
	// Prose, because this is the variant for words a person types: a search string, a
	// product title, a note.
	Prose(StepFine),
))

var inputDataBundle = clip(css.Rules(
	inputBase,
	// Data, because this is the variant for machine facts a person types: a SKU, a
	// hub code, a quantity, a date, a lane id.
	Data(StepFine),
))

// Input is the text control for WORDS: search, names, titles, notes. Proportional.
//
// InputData is the text control for MACHINE FACTS: SKU, hub code, quantity, promise
// date, lane id. Monospace and tabular.
//
// The split is not cosmetic. It is the same information architecture the display /
// prose / data roles encode (see the package doc), pushed into the form layer, where
// it does real work: a mono SKU field makes a transposed character visible while the
// operator is still typing, and a tabular quantity field lines up down a column of
// repeated rows in a receiving form.
//
// Both apply to <input>, <select> and <textarea>.
func Input() []css.Rule     { return inputBundle }
func InputData() []css.Rule { return inputDataBundle }

var fieldHintBundle = clip(css.Rules(
	Prose(StepMicro),
	css.TextColor(Graphite()),
))

// FieldHint is the helper line under a control. Wire it up with
// aria-describedby — a visually adjacent hint is not an announced hint.
func FieldHint() []css.Rule { return fieldHintBundle }

var fieldErrorBundle = clip(css.Rules(
	Prose(StepMicro),
	css.TextColor(StatusException()),
	css.FontWeight.Semibold,
))

// FieldError is the validation message under a control.
//
// It is one of only three places oxide appears (here, StatusChip/StatusValue with
// ToneException, and a destructive confirmation), which is what keeps oxide meaning
// "a human needs to look at this". Prose rather than Data, because a validation
// message is a sentence someone wrote. Pair it with aria-invalid and
// aria-describedby; color is never the only signal.
func FieldError() []css.Rule { return fieldErrorBundle }
