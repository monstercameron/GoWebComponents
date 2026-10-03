// Package attrtable is the single source of truth for the generated attribute
// builders shared by package html and package html/shorthand. The generator in
// html/internal/attrgen turns Entries into attrs_gen.go in both packages, and
// both packages' tests walk Entries so the two surfaces cannot drift.
package attrtable

// Kind selects how a generated builder stores its attribute on Props.
type Kind int

const (
	// KindString stores a string value in Props.Raw under Attr.
	KindString Kind = iota
	// KindEnumBool stores the strings "true" or "false" in Props.Raw under Attr
	// for a bool argument (enumerated attributes such as draggable).
	KindEnumBool
	// KindFlag stores a boolean attribute in Props.Raw under Attr: present when
	// true, absent when false.
	KindFlag
	// KindAriaString stores a string in Props.Aria under the name after "aria-".
	KindAriaString
	// KindAriaBool stores "true" or "false" in Props.Aria for a bool argument.
	KindAriaBool
	// KindField stores a string in the typed Props field named by Field.
	KindField
)

// Entry describes one generated attribute builder.
type Entry struct {
	// Name is the exported builder name, identical in html and html/shorthand.
	Name string
	// Attr is the rendered attribute name (for aria kinds, including "aria-").
	Attr string
	// Kind selects the storage strategy.
	Kind Kind
	// Field is the Props field name for KindField.
	Field string
	// Doc is the sentence that follows the builder name in its GoDoc comment.
	Doc string
}

// Entries lists every generated builder in declaration order.
var Entries = []Entry{
	{Name: "Draggable", Attr: "draggable", Kind: KindEnumBool, Doc: "sets the draggable attribute to \"true\" or \"false\"."},
	{Name: "Scope", Attr: "scope", Kind: KindString, Doc: "sets the scope attribute on a table header cell."},
	{Name: "Download", Attr: "download", Kind: KindString, Doc: "sets the download attribute; an empty value keeps the linked file name."},
	{Name: "ContentEditable", Attr: "contenteditable", Kind: KindString, Doc: "sets the contenteditable attribute (\"true\", \"false\" or \"plaintext-only\")."},
	{Name: "InputMode", Attr: "inputmode", Kind: KindString, Doc: "sets the inputmode attribute that hints which virtual keyboard to show."},
	{Name: "SlotAttr", Attr: "slot", Kind: KindField, Field: "Slot", Doc: "sets the slot attribute that assigns the element to a named shadow DOM slot."},
	{Name: "Part", Attr: "part", Kind: KindString, Doc: "sets the part attribute that exposes the element to ::part() selectors."},
	{Name: "Popover", Attr: "popover", Kind: KindString, Doc: "sets the popover attribute (\"auto\", \"manual\" or \"hint\")."},
	{Name: "PopoverTarget", Attr: "popovertarget", Kind: KindString, Doc: "sets the popovertarget attribute to the id of the popover the control toggles."},
	{Name: "FormAction", Attr: "formaction", Kind: KindString, Doc: "sets the formaction attribute that overrides the form's action for one submit control."},
	{Name: "FormMethod", Attr: "formmethod", Kind: KindString, Doc: "sets the formmethod attribute that overrides the form's method for one submit control."},
	{Name: "FormNoValidate", Attr: "formnovalidate", Kind: KindFlag, Doc: "sets the formnovalidate boolean attribute; absent when the argument is false."},
	{Name: "FormTarget", Attr: "formtarget", Kind: KindString, Doc: "sets the formtarget attribute that overrides the form's target for one submit control."},
	{Name: "Enctype", Attr: "enctype", Kind: KindField, Field: "EncType", Doc: "sets the enctype attribute on a form."},

	{Name: "AriaLabel", Attr: "aria-label", Kind: KindAriaString, Doc: "sets the aria-label attribute."},
	{Name: "AriaLabelledBy", Attr: "aria-labelledby", Kind: KindAriaString, Doc: "sets the aria-labelledby attribute to one or more element ids."},
	{Name: "AriaDescribedBy", Attr: "aria-describedby", Kind: KindAriaString, Doc: "sets the aria-describedby attribute to one or more element ids."},
	{Name: "AriaControls", Attr: "aria-controls", Kind: KindAriaString, Doc: "sets the aria-controls attribute to one or more element ids."},
	{Name: "AriaLive", Attr: "aria-live", Kind: KindAriaString, Doc: "sets the aria-live attribute (\"off\", \"polite\" or \"assertive\")."},
	{Name: "AriaCurrent", Attr: "aria-current", Kind: KindAriaString, Doc: "sets the aria-current attribute (\"page\", \"step\", \"true\" and so on)."},
	{Name: "AriaHasPopup", Attr: "aria-haspopup", Kind: KindAriaString, Doc: "sets the aria-haspopup attribute (\"menu\", \"dialog\", \"true\" and so on)."},
	{Name: "AriaHidden", Attr: "aria-hidden", Kind: KindAriaBool, Doc: "sets the aria-hidden attribute to \"true\" or \"false\"."},
	{Name: "AriaExpanded", Attr: "aria-expanded", Kind: KindAriaBool, Doc: "sets the aria-expanded attribute to \"true\" or \"false\"."},
	{Name: "AriaPressed", Attr: "aria-pressed", Kind: KindAriaBool, Doc: "sets the aria-pressed attribute to \"true\" or \"false\"."},
	{Name: "AriaSelected", Attr: "aria-selected", Kind: KindAriaBool, Doc: "sets the aria-selected attribute to \"true\" or \"false\"."},
	{Name: "AriaChecked", Attr: "aria-checked", Kind: KindAriaBool, Doc: "sets the aria-checked attribute to \"true\" or \"false\"; use Aria(\"checked\", \"mixed\") for the tri-state."},
	{Name: "AriaDisabled", Attr: "aria-disabled", Kind: KindAriaBool, Doc: "sets the aria-disabled attribute to \"true\" or \"false\"."},
	{Name: "AriaModal", Attr: "aria-modal", Kind: KindAriaBool, Doc: "sets the aria-modal attribute to \"true\" or \"false\"."},
	{Name: "AriaBusy", Attr: "aria-busy", Kind: KindAriaBool, Doc: "sets the aria-busy attribute to \"true\" or \"false\"."},

	{Name: "ViewBox", Attr: "viewBox", Kind: KindString, Doc: "sets the SVG viewBox attribute."},
	{Name: "Stroke", Attr: "stroke", Kind: KindString, Doc: "sets the SVG stroke paint attribute."},
	{Name: "StrokeWidth", Attr: "stroke-width", Kind: KindString, Doc: "sets the SVG stroke-width attribute."},
	{Name: "StrokeLinecap", Attr: "stroke-linecap", Kind: KindString, Doc: "sets the SVG stroke-linecap attribute."},
	{Name: "StrokeLinejoin", Attr: "stroke-linejoin", Kind: KindString, Doc: "sets the SVG stroke-linejoin attribute."},
	{Name: "Fill", Attr: "fill", Kind: KindString, Doc: "sets the SVG fill paint attribute."},
	{Name: "FillRule", Attr: "fill-rule", Kind: KindString, Doc: "sets the SVG fill-rule attribute."},
	{Name: "D", Attr: "d", Kind: KindString, Doc: "sets the SVG path data attribute d."},
	{Name: "Cx", Attr: "cx", Kind: KindString, Doc: "sets the SVG cx attribute."},
	{Name: "Cy", Attr: "cy", Kind: KindString, Doc: "sets the SVG cy attribute."},
	{Name: "R", Attr: "r", Kind: KindString, Doc: "sets the SVG circle radius attribute r."},
	{Name: "Rx", Attr: "rx", Kind: KindString, Doc: "sets the SVG rx attribute."},
	{Name: "Ry", Attr: "ry", Kind: KindString, Doc: "sets the SVG ry attribute."},
	{Name: "X1", Attr: "x1", Kind: KindString, Doc: "sets the SVG x1 attribute."},
	{Name: "Y1", Attr: "y1", Kind: KindString, Doc: "sets the SVG y1 attribute."},
	{Name: "X2", Attr: "x2", Kind: KindString, Doc: "sets the SVG x2 attribute."},
	{Name: "Y2", Attr: "y2", Kind: KindString, Doc: "sets the SVG y2 attribute."},
	{Name: "Points", Attr: "points", Kind: KindString, Doc: "sets the SVG points attribute of a polyline or polygon."},
	{Name: "Xmlns", Attr: "xmlns", Kind: KindString, Doc: "sets the xmlns namespace attribute."},
	{Name: "PreserveAspectRatio", Attr: "preserveAspectRatio", Kind: KindString, Doc: "sets the SVG preserveAspectRatio attribute."},
}
