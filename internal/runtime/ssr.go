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
func RenderToString(parseElement *Element) (parseMarkup string, parseErr error) {
	if parseElement == nil {
		return "", nil
	}
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			if parseOriginal, parseSuppressed := finalizeUnhandledPanicContext("runtime", PanicPhaseSSR, "RenderToString", "", nil, parseRecovered); parseSuppressed {
				parseMarkup = ""
				parseErr = recoveredAsError(parseOriginal)
			}
		}
	}()

	var parseBuilder strings.Builder
	if parseErr2 := renderElementToString(&parseBuilder, parseElement); parseErr2 != nil {
		return "", parseErr2
	}
	return parseBuilder.String(), nil
}

// renderElementToString is a core package helper.
func renderElementToString(parseBuilder *strings.Builder, parseElement *Element) error {
	if parseElement == nil {
		return nil
	}

	if parseTyp, parseOk := parseElement.Type.(string); parseOk {
		switch parseTyp {
		case "TEXT_ELEMENT":
			parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
			return nil
		case "FRAGMENT":
			return renderChildrenToString(parseBuilder, parseElement.Children)
		default:
			return renderHostElementToString(parseBuilder, parseTyp, parseElement)
		}
	}

	if _, parseOk2 := parseElement.Type.(*ContextProviderType); parseOk2 {
		return renderChildrenToString(parseBuilder, parseElement.Children)
	}
	if _, parseOk3 := parseElement.Type.(*PortalElementType); parseOk3 {
		return renderChildrenToString(parseBuilder, parseElement.Children)
	}
	if _, parseOk4 := parseElement.Type.(*ReactiveTextElementType); parseOk4 {
		parseGetter, _ := parseElement.Props[reactiveTextGetterProp].(func() string)
		if parseGetter == nil {
			parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
			return nil
		}
		parseBuilder.WriteString(html.EscapeString(parseGetter()))
		return nil
	}
	if _, parseOk5 := parseElement.Type.(*ReactiveRegionElementType); parseOk5 {
		render, _ := parseElement.Props[reactiveRegionRenderProp].(func() *Element)
		if render == nil {
			return nil
		}
		return renderElementToString(parseBuilder, render())
	}
	if _, parseOk6 := parseElement.Type.(*ErrorBoundaryType); parseOk6 {
		return renderErrorBoundaryToString(parseBuilder, parseElement)
	}

	parseResolved, parseErr := resolveComponentElement(parseElement)
	if parseErr != nil {
		return parseErr
	}
	if parseResolved == nil {
		return nil
	}
	return renderElementToString(parseBuilder, parseResolved)
}

// renderErrorBoundaryToString is a core package helper.
func renderErrorBoundaryToString(parseBuilder *strings.Builder, parseElement *Element) (parseErr error) {
	if parseElement == nil {
		return nil
	}

	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			return
		}

		parseBoundaryErr := normalizeBoundaryError(parseRecovered)
		if parseOnError, _ := parseElement.Props["onError"].(func(error)); parseOnError != nil {
			func() {
				defer func() { _ = recover() }()
				parseOnError(parseBoundaryErr)
			}()
		}

		if parseFallbackFn, _ := parseElement.Props["errorFallback"].(func(error, func()) *Element); parseFallbackFn != nil {
			parseFallback := parseFallbackFn(parseBoundaryErr, func() {})
			parseErr = renderElementToString(parseBuilder, parseFallback)
			return
		}
		if parseFallback2, _ := parseElement.Props["fallback"].(*Element); parseFallback2 != nil {
			parseErr = renderElementToString(parseBuilder, parseFallback2)
			return
		}
		parseErr = nil
	}()

	return renderChildrenToString(parseBuilder, parseElement.Children)
}

// renderHostElementToString is a core package helper.
func renderHostElementToString(parseBuilder *strings.Builder, parseTag string, parseElement *Element) error {
	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseTag)

	for _, parseAttr := range serializeProps(parseElement.Props) {
		parseBuilder.WriteByte(' ')
		parseBuilder.WriteString(parseAttr)
	}
	parseBuilder.WriteByte('>')

	if isVoidElement(parseTag) {
		return nil
	}

	if parseElement.hasDirectText {
		parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
	} else if parseErr := renderChildrenToString(parseBuilder, getElementChildren(parseElement)); parseErr != nil {
		return parseErr
	}

	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseTag)
	parseBuilder.WriteByte('>')
	return nil
}

// renderChildrenToString is a core package helper.
func renderChildrenToString(parseBuilder *strings.Builder, parseChildren []interface{}) error {
	for _, parseChild := range parseChildren {
		switch parseValue := parseChild.(type) {
		case nil:
			continue
		case *Element:
			if parseErr := renderElementToString(parseBuilder, parseValue); parseErr != nil {
				return parseErr
			}
		case string:
			parseBuilder.WriteString(html.EscapeString(parseValue))
		default:
			parseBuilder.WriteString(html.EscapeString(fmt.Sprint(parseValue)))
		}
	}
	return nil
}

// resolveComponentElement is a core package helper.
func resolveComponentElement(parseElement *Element) (*Element, error) {
	if parseComponent, parseOk := parseElement.Type.(*ComponentType); parseOk {
		return parseComponent.Render(parseElement.Props), nil
	}

	parseValue := reflect.ValueOf(parseElement.Type)
	if !parseValue.IsValid() || parseValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("ssr: unsupported element type %T", parseElement.Type)
	}

	parseTyp := parseValue.Type()
	if parseTyp.NumOut() != 1 {
		return nil, fmt.Errorf("ssr: component %T must return exactly one value", parseElement.Type)
	}
	if parseTyp.Out(0) != reflect.TypeOf((*Element)(nil)) {
		return nil, fmt.Errorf("ssr: component %T must return *runtime.Element", parseElement.Type)
	}

	var parseArgs []reflect.Value
	switch parseTyp.NumIn() {
	case 0:
		parseArgs = nil
	case 1:
		parseArg, parseErr := buildComponentArg(parseTyp.In(0), parseElement.Props)
		if parseErr != nil {
			return nil, parseErr
		}
		parseArgs = []reflect.Value{parseArg}
	default:
		return nil, fmt.Errorf("ssr: component %T has unsupported arity %d", parseElement.Type, parseTyp.NumIn())
	}

	parseResult := parseValue.Call(parseArgs)
	if len(parseResult) != 1 || parseResult[0].IsNil() {
		return nil, nil
	}
	parseResolved, _ := parseResult[0].Interface().(*Element)
	return parseResolved, nil
}

// buildComponentArg is a core package helper.
func buildComponentArg(parseTarget reflect.Type, parseProps map[string]interface{}) (reflect.Value, error) {
	if parseProps == nil {
		return reflect.Zero(parseTarget), nil
	}

	parseProvided := reflect.ValueOf(Attrs(parseProps))
	if parseProvided.Type() == parseTarget {
		return parseProvided, nil
	}
	if parseProvided.Type().AssignableTo(parseTarget) {
		return parseProvided, nil
	}
	if parseProvided.Type().ConvertibleTo(parseTarget) {
		return parseProvided.Convert(parseTarget), nil
	}
	return reflect.Zero(parseTarget), fmt.Errorf("ssr: unsupported component prop type %s", parseTarget)
}

// serializeProps is a core package helper.
func serializeProps(parseProps map[string]interface{}) []string {
	if len(parseProps) == 0 {
		return nil
	}

	parseKeys := make([]string, 0, len(parseProps))
	for parseKey, parseValue := range parseProps {
		if shouldSkipSSRProp(parseKey, parseValue) {
			continue
		}
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)

	parseAttrs := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseName := normalizeSSRAttrName(parseKey2)
		parseValue2 := parseProps[parseKey2]
		parseSerialized, parseOk := serializeSSRAttr(parseName, parseValue2)
		if parseOk {
			parseAttrs = append(parseAttrs, parseSerialized)
		}
	}
	return parseAttrs
}

// shouldSkipSSRProp is a core package helper.
func shouldSkipSSRProp(parseKey string, parseValue interface{}) bool {
	if parseKey == "children" || parseKey == "key" || parseValue == nil {
		return true
	}
	if strings.HasPrefix(parseKey, "__gwc_prop__:") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(parseKey), "on") {
		return true
	}
	return false
}

// normalizeSSRAttrName is a core package helper.
func normalizeSSRAttrName(parseKey string) string {
	switch parseKey {
	case "className":
		return "class"
	case "htmlFor":
		return "for"
	default:
		return parseKey
	}
}

// serializeSSRAttr is a core package helper.
func serializeSSRAttr(parseName string, parseValue interface{}) (string, bool) {
	switch parseTyped := parseValue.(type) {
	case bool:
		if !parseTyped {
			return "", false
		}
		return parseName, true
	case string:
		return parseName + `="` + html.EscapeString(parseTyped) + `"`, true
	case map[string]string:
		if parseName != "style" {
			return parseName + `="` + html.EscapeString(fmt.Sprint(parseTyped)) + `"`, true
		}
		return parseName + `="` + html.EscapeString(serializeStyleMap(parseTyped)) + `"`, true
	default:
		return parseName + `="` + html.EscapeString(fmt.Sprint(parseValue)) + `"`, true
	}
}

// serializeStyleMap is a core package helper.
func serializeStyleMap(parseStyles map[string]string) string {
	if len(parseStyles) == 0 {
		return ""
	}
	parseKeys := make([]string, 0, len(parseStyles))
	for parseKey := range parseStyles {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)

	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseParts = append(parseParts, parseKey2+":"+parseStyles[parseKey2])
	}
	return strings.Join(parseParts, ";")
}

// isVoidElement is a core package helper.
func isVoidElement(parseTag string) bool {
	switch strings.ToLower(parseTag) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
