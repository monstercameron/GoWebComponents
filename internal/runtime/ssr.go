package runtime

import (
	"fmt"
	"html"
	"reflect"
	"sort"
	"strings"
)

// RenderToString renders a virtual element tree to HTML.
//
// This is the first internal SSR slice: it supports host elements, text nodes,
// fragments, and simple function components that return *Element. Hydration and
// browser bootstrap are intentionally out of scope here.
func RenderToString(element *Element) (markup string, err error) {
	if element == nil {
		return "", nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			if original, suppressed := finalizeUnhandledPanicContext("runtime", PanicPhaseSSR, "RenderToString", "", nil, recovered); suppressed {
				markup = ""
				err = recoveredAsError(original)
			}
		}
	}()

	var builder strings.Builder
	if err := renderElementToString(&builder, element); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func renderElementToString(builder *strings.Builder, element *Element) error {
	if element == nil {
		return nil
	}

	if typ, ok := element.Type.(string); ok {
		switch typ {
		case "TEXT_ELEMENT":
			builder.WriteString(html.EscapeString(element.TextContent))
			return nil
		case "FRAGMENT":
			return renderChildrenToString(builder, element.Children)
		default:
			return renderHostElementToString(builder, typ, element)
		}
	}

	if _, ok := element.Type.(*ContextProviderType); ok {
		return renderChildrenToString(builder, element.Children)
	}
	if _, ok := element.Type.(*PortalElementType); ok {
		return renderChildrenToString(builder, element.Children)
	}
	if _, ok := element.Type.(*ReactiveRegionElementType); ok {
		render, _ := element.Props[reactiveRegionRenderProp].(func() *Element)
		if render == nil {
			return nil
		}
		return renderElementToString(builder, render())
	}
	if _, ok := element.Type.(*ErrorBoundaryType); ok {
		return renderErrorBoundaryToString(builder, element)
	}

	resolved, err := resolveComponentElement(element)
	if err != nil {
		return err
	}
	if resolved == nil {
		return nil
	}
	return renderElementToString(builder, resolved)
}

func renderErrorBoundaryToString(builder *strings.Builder, element *Element) (err error) {
	if element == nil {
		return nil
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		boundaryErr := normalizeBoundaryError(recovered)
		if onError, _ := element.Props["onError"].(func(error)); onError != nil {
			func() {
				defer func() { _ = recover() }()
				onError(boundaryErr)
			}()
		}

		if fallbackFn, _ := element.Props["errorFallback"].(func(error, func()) *Element); fallbackFn != nil {
			var fallback *Element
			fallback = fallbackFn(boundaryErr, func() {})
			err = renderElementToString(builder, fallback)
			return
		}
		if fallback, _ := element.Props["fallback"].(*Element); fallback != nil {
			err = renderElementToString(builder, fallback)
			return
		}
		err = nil
	}()

	return renderChildrenToString(builder, element.Children)
}

func renderHostElementToString(builder *strings.Builder, tag string, element *Element) error {
	builder.WriteByte('<')
	builder.WriteString(tag)

	for _, attr := range serializeProps(element.Props) {
		builder.WriteByte(' ')
		builder.WriteString(attr)
	}
	builder.WriteByte('>')

	if isVoidElement(tag) {
		return nil
	}

	if err := renderChildrenToString(builder, element.Children); err != nil {
		return err
	}

	builder.WriteString("</")
	builder.WriteString(tag)
	builder.WriteByte('>')
	return nil
}

func renderChildrenToString(builder *strings.Builder, children []interface{}) error {
	for _, child := range children {
		switch value := child.(type) {
		case nil:
			continue
		case *Element:
			if err := renderElementToString(builder, value); err != nil {
				return err
			}
		case string:
			builder.WriteString(html.EscapeString(value))
		default:
			builder.WriteString(html.EscapeString(fmt.Sprint(value)))
		}
	}
	return nil
}

func resolveComponentElement(element *Element) (*Element, error) {
	if component, ok := element.Type.(*ComponentType); ok {
		return component.Render(element.Props), nil
	}

	value := reflect.ValueOf(element.Type)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return nil, fmt.Errorf("ssr: unsupported element type %T", element.Type)
	}

	typ := value.Type()
	if typ.NumOut() != 1 {
		return nil, fmt.Errorf("ssr: component %T must return exactly one value", element.Type)
	}
	if typ.Out(0) != reflect.TypeOf((*Element)(nil)) {
		return nil, fmt.Errorf("ssr: component %T must return *runtime.Element", element.Type)
	}

	var args []reflect.Value
	switch typ.NumIn() {
	case 0:
		args = nil
	case 1:
		arg, err := buildComponentArg(typ.In(0), element.Props)
		if err != nil {
			return nil, err
		}
		args = []reflect.Value{arg}
	default:
		return nil, fmt.Errorf("ssr: component %T has unsupported arity %d", element.Type, typ.NumIn())
	}

	result := value.Call(args)
	if len(result) != 1 || result[0].IsNil() {
		return nil, nil
	}
	resolved, _ := result[0].Interface().(*Element)
	return resolved, nil
}

func buildComponentArg(target reflect.Type, props map[string]interface{}) (reflect.Value, error) {
	if props == nil {
		return reflect.Zero(target), nil
	}

	provided := reflect.ValueOf(Attrs(props))
	if provided.Type() == target {
		return provided, nil
	}
	if provided.Type().AssignableTo(target) {
		return provided, nil
	}
	if provided.Type().ConvertibleTo(target) {
		return provided.Convert(target), nil
	}
	return reflect.Zero(target), fmt.Errorf("ssr: unsupported component prop type %s", target)
}

func serializeProps(props map[string]interface{}) []string {
	if len(props) == 0 {
		return nil
	}

	keys := make([]string, 0, len(props))
	for key, value := range props {
		if shouldSkipSSRProp(key, value) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	attrs := make([]string, 0, len(keys))
	for _, key := range keys {
		name := normalizeSSRAttrName(key)
		value := props[key]
		serialized, ok := serializeSSRAttr(name, value)
		if ok {
			attrs = append(attrs, serialized)
		}
	}
	return attrs
}

func shouldSkipSSRProp(key string, value interface{}) bool {
	if key == "children" || key == "key" || value == nil {
		return true
	}
	if strings.HasPrefix(key, "__gwc_prop__:") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(key), "on") {
		return true
	}
	return false
}

func normalizeSSRAttrName(key string) string {
	switch key {
	case "className":
		return "class"
	case "htmlFor":
		return "for"
	default:
		return key
	}
}

func serializeSSRAttr(name string, value interface{}) (string, bool) {
	switch typed := value.(type) {
	case bool:
		if !typed {
			return "", false
		}
		return name, true
	case string:
		return name + `="` + html.EscapeString(typed) + `"`, true
	case map[string]string:
		if name != "style" {
			return name + `="` + html.EscapeString(fmt.Sprint(typed)) + `"`, true
		}
		return name + `="` + html.EscapeString(serializeStyleMap(typed)) + `"`, true
	default:
		return name + `="` + html.EscapeString(fmt.Sprint(value)) + `"`, true
	}
}

func serializeStyleMap(styles map[string]string) string {
	if len(styles) == 0 {
		return ""
	}
	keys := make([]string, 0, len(styles))
	for key := range styles {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+styles[key])
	}
	return strings.Join(parts, ";")
}

func isVoidElement(tag string) bool {
	switch strings.ToLower(tag) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
