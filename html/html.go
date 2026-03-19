package html

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Props contains the common HTML attributes and event handlers supported by the typed builders.
type Props struct {
	ID           string
	Class        string
	Key          string
	Slot         string
	Title        string
	Type         string
	Name         string
	Value        string
	Placeholder  string
	Accept       string
	Href         string
	Src          string
	Alt          string
	For          string
	Role         string
	Target       string
	Rel          string
	Action       string
	Method       string
	EncType      string
	AutoComplete string
	Min          string
	Max          string
	Step         string

	Rows int
	Cols int

	Checked   bool
	Disabled  bool
	Selected  bool
	Required  bool
	ReadOnly  bool
	Hidden    bool
	Multiple  bool
	AutoFocus bool

	Style map[string]string
	Data  map[string]string
	Aria  map[string]string
	Raw   map[string]interface{}

	OnClick   ui.Handler
	OnInput   ui.Handler
	OnChange  ui.Handler
	OnSubmit  ui.Handler
	OnKeyDown ui.Handler
	OnKeyUp   ui.Handler
	OnFocus   ui.Handler
	OnBlur    ui.Handler
}

// CustomElementProps makes attribute-versus-property intent explicit for
// browser-defined custom elements and web components.
type CustomElementProps struct {
	Props      Props
	Attributes map[string]string
	Presence   map[string]bool
	Properties map[string]interface{}
}

const customElementPropertyPrefix = "__gwc_prop__:"

// Text creates a text node.
func Text(content string) ui.Node {
	return ui.Text(content)
}

// Tag creates a node for an arbitrary HTML tag name.
func Tag(name string, props Props, children ...ui.Node) ui.Node {
	return runtime.CreateElement(name, toRuntimeProps(props), toInterfaces(children)...)
}

// CustomElement creates a browser-defined custom element with explicit
// attribute and property channels.
func CustomElement(name string, props CustomElementProps, children ...ui.Node) ui.Node {
	values := toRuntimeProps(props.Props)
	count := len(props.Attributes) + len(props.Presence) + len(props.Properties)
	if count == 0 {
		return runtime.CreateElement(name, values, toInterfaces(children)...)
	}
	if values == nil {
		values = make(map[string]interface{}, count)
	}
	for key, value := range props.Attributes {
		values[key] = value
	}
	for key, enabled := range props.Presence {
		if enabled {
			values[key] = ""
		}
	}
	for key, value := range props.Properties {
		values[customElementPropertyPrefix+key] = value
	}
	return runtime.CreateElement(name, values, toInterfaces(children)...)
}

// Fragment groups children without introducing an extra host element.
func Fragment(children ...ui.Node) ui.Node {
	return ui.Fragment(children...)
}

func A(props Props, children ...ui.Node) ui.Node {
	return Tag("a", props, children...)
}

func Article(props Props, children ...ui.Node) ui.Node {
	return Tag("article", props, children...)
}

func Aside(props Props, children ...ui.Node) ui.Node {
	return Tag("aside", props, children...)
}

func Blockquote(props Props, children ...ui.Node) ui.Node {
	return Tag("blockquote", props, children...)
}

func Br(props Props) ui.Node {
	return Tag("br", props)
}

func Button(props Props, children ...ui.Node) ui.Node {
	return Tag("button", props, children...)
}

func Code(props Props, children ...ui.Node) ui.Node {
	return Tag("code", props, children...)
}

func Dialog(props Props, children ...ui.Node) ui.Node {
	return Tag("dialog", props, children...)
}

func Div(props Props, children ...ui.Node) ui.Node {
	return Tag("div", props, children...)
}

func Em(props Props, children ...ui.Node) ui.Node {
	return Tag("em", props, children...)
}

func Fieldset(props Props, children ...ui.Node) ui.Node {
	return Tag("fieldset", props, children...)
}

func Footer(props Props, children ...ui.Node) ui.Node {
	return Tag("footer", props, children...)
}

func Form(props Props, children ...ui.Node) ui.Node {
	return Tag("form", props, children...)
}

func H1(props Props, children ...ui.Node) ui.Node {
	return Tag("h1", props, children...)
}

func H2(props Props, children ...ui.Node) ui.Node {
	return Tag("h2", props, children...)
}

func H3(props Props, children ...ui.Node) ui.Node {
	return Tag("h3", props, children...)
}

func H4(props Props, children ...ui.Node) ui.Node {
	return Tag("h4", props, children...)
}

func H5(props Props, children ...ui.Node) ui.Node {
	return Tag("h5", props, children...)
}

func H6(props Props, children ...ui.Node) ui.Node {
	return Tag("h6", props, children...)
}

func Header(props Props, children ...ui.Node) ui.Node {
	return Tag("header", props, children...)
}

func Hr(props Props) ui.Node {
	return Tag("hr", props)
}

func Img(props Props) ui.Node {
	return Tag("img", props)
}

func Input(props Props) ui.Node {
	return Tag("input", props)
}

func HiddenInput(name string, value string) ui.Node {
	return Input(Props{Type: "hidden", Name: name, Value: value})
}

func Label(props Props, children ...ui.Node) ui.Node {
	return Tag("label", props, children...)
}

func Legend(props Props, children ...ui.Node) ui.Node {
	return Tag("legend", props, children...)
}

func Li(props Props, children ...ui.Node) ui.Node {
	return Tag("li", props, children...)
}

func Main(props Props, children ...ui.Node) ui.Node {
	return Tag("main", props, children...)
}

func Nav(props Props, children ...ui.Node) ui.Node {
	return Tag("nav", props, children...)
}

func Option(props Props, children ...ui.Node) ui.Node {
	return Tag("option", props, children...)
}

func P(props Props, children ...ui.Node) ui.Node {
	return Tag("p", props, children...)
}

func Pre(props Props, children ...ui.Node) ui.Node {
	return Tag("pre", props, children...)
}

func Section(props Props, children ...ui.Node) ui.Node {
	return Tag("section", props, children...)
}

func Select(props Props, children ...ui.Node) ui.Node {
	return Tag("select", props, children...)
}

func Small(props Props, children ...ui.Node) ui.Node {
	return Tag("small", props, children...)
}

func Span(props Props, children ...ui.Node) ui.Node {
	return Tag("span", props, children...)
}

func Strong(props Props, children ...ui.Node) ui.Node {
	return Tag("strong", props, children...)
}

func Textarea(props Props, children ...ui.Node) ui.Node {
	return Tag("textarea", props, children...)
}

func Time(props Props, children ...ui.Node) ui.Node {
	return Tag("time", props, children...)
}

func Ul(props Props, children ...ui.Node) ui.Node {
	return Tag("ul", props, children...)
}

func toRuntimeProps(props Props) map[string]interface{} {
	onClick := props.OnClick.Value()
	onInput := props.OnInput.Value()
	onChange := props.OnChange.Value()
	onSubmit := props.OnSubmit.Value()
	onKeyDown := props.OnKeyDown.Value()
	onKeyUp := props.OnKeyUp.Value()
	onFocus := props.OnFocus.Value()
	onBlur := props.OnBlur.Value()

	count := len(props.Data) + len(props.Aria) + len(props.Raw)
	if props.ID != "" {
		count++
	}
	if props.Class != "" {
		count++
	}
	if props.Key != "" {
		count++
	}
	if props.Slot != "" {
		count++
	}
	if props.Title != "" {
		count++
	}
	if props.Type != "" {
		count++
	}
	if props.Name != "" {
		count++
	}
	if props.Value != "" {
		count++
	}
	if props.Placeholder != "" {
		count++
	}
	if props.Accept != "" {
		count++
	}
	if props.Href != "" {
		count++
	}
	if props.Src != "" {
		count++
	}
	if props.Alt != "" {
		count++
	}
	if props.For != "" {
		count++
	}
	if props.Role != "" {
		count++
	}
	if props.Target != "" {
		count++
	}
	if props.Rel != "" {
		count++
	}
	if props.Action != "" {
		count++
	}
	if props.Method != "" {
		count++
	}
	if props.EncType != "" {
		count++
	}
	if props.AutoComplete != "" {
		count++
	}
	if props.Min != "" {
		count++
	}
	if props.Max != "" {
		count++
	}
	if props.Step != "" {
		count++
	}
	if props.Rows != 0 {
		count++
	}
	if props.Cols != 0 {
		count++
	}
	if props.Checked {
		count++
	}
	if props.Disabled {
		count++
	}
	if props.Selected {
		count++
	}
	if props.Required {
		count++
	}
	if props.ReadOnly {
		count++
	}
	if props.Hidden {
		count++
	}
	if props.Multiple {
		count++
	}
	if props.AutoFocus {
		count++
	}
	if props.Style != nil {
		count++
	}
	if onClick != nil {
		count++
	}
	if onInput != nil {
		count++
	}
	if onChange != nil {
		count++
	}
	if onSubmit != nil {
		count++
	}
	if onKeyDown != nil {
		count++
	}
	if onKeyUp != nil {
		count++
	}
	if onFocus != nil {
		count++
	}
	if onBlur != nil {
		count++
	}

	if count == 0 {
		return nil
	}

	values := make(map[string]interface{}, count)
	if props.ID != "" {
		values["id"] = props.ID
	}
	if props.Class != "" {
		values["class"] = props.Class
	}
	if props.Key != "" {
		values["key"] = props.Key
	}
	if props.Slot != "" {
		values["slot"] = props.Slot
	}
	if props.Title != "" {
		values["title"] = props.Title
	}
	if props.Type != "" {
		values["type"] = props.Type
	}
	if props.Name != "" {
		values["name"] = props.Name
	}
	if props.Value != "" {
		values["value"] = props.Value
	}
	if props.Placeholder != "" {
		values["placeholder"] = props.Placeholder
	}
	if props.Accept != "" {
		values["accept"] = props.Accept
	}
	if props.Href != "" {
		values["href"] = props.Href
	}
	if props.Src != "" {
		values["src"] = props.Src
	}
	if props.Alt != "" {
		values["alt"] = props.Alt
	}
	if props.For != "" {
		values["htmlFor"] = props.For
	}
	if props.Role != "" {
		values["role"] = props.Role
	}
	if props.Target != "" {
		values["target"] = props.Target
	}
	if props.Rel != "" {
		values["rel"] = props.Rel
	}
	if props.Action != "" {
		values["action"] = props.Action
	}
	if props.Method != "" {
		values["method"] = props.Method
	}
	if props.EncType != "" {
		values["enctype"] = props.EncType
	}
	if props.AutoComplete != "" {
		values["autocomplete"] = props.AutoComplete
	}
	if props.Min != "" {
		values["min"] = props.Min
	}
	if props.Max != "" {
		values["max"] = props.Max
	}
	if props.Step != "" {
		values["step"] = props.Step
	}
	if props.Rows != 0 {
		values["rows"] = props.Rows
	}
	if props.Cols != 0 {
		values["cols"] = props.Cols
	}
	if props.Checked {
		values["checked"] = true
	}
	if props.Disabled {
		values["disabled"] = true
	}
	if props.Selected {
		values["selected"] = true
	}
	if props.Required {
		values["required"] = true
	}
	if props.ReadOnly {
		values["readOnly"] = true
	}
	if props.Hidden {
		values["hidden"] = true
	}
	if props.Multiple {
		values["multiple"] = true
	}
	if props.AutoFocus {
		values["autofocus"] = true
	}
	if props.Style != nil {
		values["style"] = props.Style
	}

	if len(props.Data) != 0 {
		for key, value := range props.Data {
			values["data-"+key] = value
		}
	}
	if len(props.Aria) != 0 {
		for key, value := range props.Aria {
			values["aria-"+key] = value
		}
	}
	if onClick != nil {
		values["onclick"] = onClick
	}
	if onInput != nil {
		values["oninput"] = onInput
	}
	if onChange != nil {
		values["onchange"] = onChange
	}
	if onSubmit != nil {
		values["onsubmit"] = onSubmit
	}
	if onKeyDown != nil {
		values["onkeydown"] = onKeyDown
	}
	if onKeyUp != nil {
		values["onkeyup"] = onKeyUp
	}
	if onFocus != nil {
		values["onfocus"] = onFocus
	}
	if onBlur != nil {
		values["onblur"] = onBlur
	}
	if len(props.Raw) != 0 {
		for key, value := range props.Raw {
			values[key] = value
		}
	}

	return values
}

func toInterfaces(children []ui.Node) []interface{} {
	if len(children) == 0 {
		return nil
	}

	values := make([]interface{}, 0, len(children))
	for _, child := range children {
		values = append(values, child)
	}

	return values
}
