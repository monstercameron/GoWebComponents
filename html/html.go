//go:build js && wasm
// +build js,wasm

package html

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Props struct {
	ID           string
	Class        string
	Key          string
	Title        string
	Type         string
	Name         string
	Value        string
	Placeholder  string
	Href         string
	Src          string
	Alt          string
	For          string
	Role         string
	Target       string
	Rel          string
	Action       string
	Method       string
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

func Text(content string) ui.Node {
	return ui.Text(content)
}

func Tag(name string, props Props, children ...ui.Node) ui.Node {
	return runtime.CreateElement(name, toRuntimeProps(props), toInterfaces(children)...)
}

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
	values := map[string]interface{}{
		"id":           props.ID,
		"class":        props.Class,
		"key":          props.Key,
		"title":        props.Title,
		"type":         props.Type,
		"name":         props.Name,
		"value":        props.Value,
		"placeholder":  props.Placeholder,
		"href":         props.Href,
		"src":          props.Src,
		"alt":          props.Alt,
		"htmlFor":      props.For,
		"role":         props.Role,
		"target":       props.Target,
		"rel":          props.Rel,
		"action":       props.Action,
		"method":       props.Method,
		"autocomplete": props.AutoComplete,
		"min":          props.Min,
		"max":          props.Max,
		"step":         props.Step,
		"rows":         props.Rows,
		"cols":         props.Cols,
		"checked":      props.Checked,
		"disabled":     props.Disabled,
		"selected":     props.Selected,
		"required":     props.Required,
		"readOnly":     props.ReadOnly,
		"hidden":       props.Hidden,
		"multiple":     props.Multiple,
		"autofocus":    props.AutoFocus,
	}

	if props.Style != nil {
		values["style"] = props.Style
	}

	for key, value := range props.Data {
		values["data-"+key] = value
	}

	for key, value := range props.Aria {
		values["aria-"+key] = value
	}

	if props.OnClick.Value() != nil {
		values["onclick"] = props.OnClick.Value()
	}
	if props.OnInput.Value() != nil {
		values["oninput"] = props.OnInput.Value()
	}
	if props.OnChange.Value() != nil {
		values["onchange"] = props.OnChange.Value()
	}
	if props.OnSubmit.Value() != nil {
		values["onsubmit"] = props.OnSubmit.Value()
	}
	if props.OnKeyDown.Value() != nil {
		values["onkeydown"] = props.OnKeyDown.Value()
	}
	if props.OnKeyUp.Value() != nil {
		values["onkeyup"] = props.OnKeyUp.Value()
	}
	if props.OnFocus.Value() != nil {
		values["onfocus"] = props.OnFocus.Value()
	}
	if props.OnBlur.Value() != nil {
		values["onblur"] = props.OnBlur.Value()
	}

	for key, value := range props.Raw {
		values[key] = value
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
