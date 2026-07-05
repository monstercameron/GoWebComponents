package runtime

import (
	"fmt"
	"html"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// isValidSSRAttrName reports whether a name conforms to the HTML attribute
// name production (XML/HTML-safe subset): ^[a-zA-Z_:][a-zA-Z0-9_.:-]*$.
// Hand-rolled instead of a regexp so wasm builds do not link the regexp
// package (~290 kB) for a single pattern.
func isValidSSRAttrName(parseName string) bool {
	if parseName == "" {
		return false
	}
	for parseIdx := 0; parseIdx < len(parseName); parseIdx++ {
		parseC := parseName[parseIdx]
		parseIsAlpha := (parseC >= 'a' && parseC <= 'z') || (parseC >= 'A' && parseC <= 'Z')
		if parseIdx == 0 {
			if !parseIsAlpha && parseC != '_' && parseC != ':' {
				return false
			}
			continue
		}
		if !parseIsAlpha && !(parseC >= '0' && parseC <= '9') && parseC != '_' && parseC != '.' && parseC != ':' && parseC != '-' {
			return false
		}
	}
	return true
}

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
	if parseErr2 := renderElementToString(&parseBuilder, parseElement, ssrHookOwnerContext(nil)); parseErr2 != nil {
		return "", parseErr2
	}
	return parseBuilder.String(), nil
}

// ssrHookOwnerContextKey is a reserved context-values key that carries the
// rendering goroutine's hook-owner id down the synchronous SSR walk, so each
// component resolution skips a full runtime.Stack traceback (the dominant SSR
// cost in development builds). It is negative and can never collide with
// context descriptor IDs, which come from a positive counter.
const ssrHookOwnerContextKey int64 = -1 << 62

// ssrHookOwnerContext derives a context-values map seeded with the calling
// goroutine's hook-owner id. Each goroutine that enters an SSR walk (string
// render, stream shell, stream boundary chunk) must seed its own map; the id
// is only valid on the call stack that computed it.
func ssrHookOwnerContext(parseParent map[int64]any) map[int64]any {
	if !hookThreadingGuardEnabled {
		return parseParent
	}
	parseOwner := computeHookGoroutineID()
	if parseOwner == 0 {
		return parseParent
	}
	parseCtx := make(map[int64]any, len(parseParent)+1)
	maps.Copy(parseCtx, parseParent)
	parseCtx[ssrHookOwnerContextKey] = parseOwner
	return parseCtx
}

// renderElementToString is a core package helper. parseCtx carries the inherited
// context values (descriptor ID -> value) down the tree so a server-rendered
// component's GoUseContextValue resolves to the nearest provider's value.
func renderElementToString(parseBuilder *strings.Builder, parseElement *Element, parseCtx map[int64]any) error {
	if parseElement == nil {
		return nil
	}

	if parseTyp, parseOk := parseElement.Type.(string); parseOk {
		switch parseTyp {
		case "TEXT_ELEMENT":
			parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
			return nil
		case "FRAGMENT":
			return renderChildrenToString(parseBuilder, parseElement.Children, parseCtx)
		default:
			return renderHostElementToString(parseBuilder, parseTyp, parseElement, parseCtx)
		}
	}

	if parseProvider, parseOk2 := parseElement.Type.(*ContextProviderType); parseOk2 {
		parseChildCtx := parseCtx
		if parseProvider.Descriptor != nil {
			parseChildCtx = deriveContextValues(parseCtx, parseProvider.Descriptor.ID, parseElement.Props["value"])
		}
		return renderChildrenToString(parseBuilder, parseElement.Children, parseChildCtx)
	}
	if _, parseOk3 := parseElement.Type.(*PortalElementType); parseOk3 {
		return renderChildrenToString(parseBuilder, parseElement.Children, parseCtx)
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
		return renderElementToString(parseBuilder, render(), parseCtx)
	}
	if _, parseOk6 := parseElement.Type.(*ErrorBoundaryType); parseOk6 {
		return renderErrorBoundaryToString(parseBuilder, parseElement, parseCtx)
	}
	if _, parseOk7 := parseElement.Type.(*AsyncBoundaryElementType); parseOk7 {
		return renderAsyncBoundaryToString(parseBuilder, parseElement, parseCtx)
	}

	parseResolved, parseErr := resolveComponentElement(parseElement, parseCtx)
	if parseErr != nil {
		return parseErr
	}
	if parseResolved == nil {
		return nil
	}
	return renderElementToString(parseBuilder, parseResolved, parseCtx)
}

// renderErrorBoundaryToString is a core package helper.
func renderErrorBoundaryToString(parseBuilder *strings.Builder, parseElement *Element, parseCtx map[int64]any) (parseErr error) {
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
			parseErr = renderElementToString(parseBuilder, parseFallback, parseCtx)
			return
		}
		if parseFallback2, _ := parseElement.Props["fallback"].(*Element); parseFallback2 != nil {
			parseErr = renderElementToString(parseBuilder, parseFallback2, parseCtx)
			return
		}
		parseErr = nil
	}()

	return renderChildrenToString(parseBuilder, parseElement.Children, parseCtx)
}

// renderAsyncBoundaryToString renders async fallback content when a child suspends.
func renderAsyncBoundaryToString(parseBuilder *strings.Builder, parseElement *Element, parseCtx map[int64]any) (parseErr error) {
	if parseElement == nil {
		return nil
	}
	if parseErr2, _ := parseElement.Props["error"].(error); parseErr2 != nil {
		return renderAsyncBoundaryFallbackToString(parseBuilder, parseElement, parseErr2, parseCtx)
	}
	if parsePending, _ := parseElement.Props["pending"].(bool); parsePending {
		return renderAsyncBoundaryFallbackToString(parseBuilder, parseElement, nil, parseCtx)
	}

	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			return
		}
		if _, parseOk := AsSuspension(parseRecovered); parseOk {
			parseErr = renderAsyncBoundaryFallbackToString(parseBuilder, parseElement, nil, parseCtx)
			return
		}
		panic(parseRecovered)
	}()

	var parseContentBuilder strings.Builder
	if parseContent, _ := parseElement.Props["content"].(*Element); parseContent != nil {
		if parseErr2 := renderElementToString(&parseContentBuilder, parseContent, parseCtx); parseErr2 != nil {
			return parseErr2
		}
		parseBuilder.WriteString(parseContentBuilder.String())
		return nil
	}
	if parseErr2 := renderChildrenToString(&parseContentBuilder, parseElement.Children, parseCtx); parseErr2 != nil {
		return parseErr2
	}
	parseBuilder.WriteString(parseContentBuilder.String())
	return nil
}

// renderAsyncBoundaryFallbackToString renders the best available async fallback.
func renderAsyncBoundaryFallbackToString(parseBuilder *strings.Builder, parseElement *Element, parseErr error, parseCtx map[int64]any) error {
	if parseElement == nil || parseElement.Props == nil {
		return nil
	}
	if parseErr != nil {
		if parseFallbackFn, _ := parseElement.Props["errorFallback"].(func(error) *Element); parseFallbackFn != nil {
			return renderElementToString(parseBuilder, parseFallbackFn(parseErr), parseCtx)
		}
	}
	if parseFallback, _ := parseElement.Props["fallback"].(*Element); parseFallback != nil {
		return renderElementToString(parseBuilder, parseFallback, parseCtx)
	}
	return nil
}

// renderHostElementToString is a core package helper.
// textareaControlledValue reports the string a <textarea> must render as its text
// content. HTML ignores a `value` attribute on <textarea> — the displayed value
// is its child text — so a controlled textarea's value prop has to be serialized
// as content or it renders empty server-side (and mismatches on hydration). React
// renders a textarea's value/defaultValue the same way.
func textareaControlledValue(parseTag string, parseProps map[string]any) (string, bool) {
	if !strings.EqualFold(parseTag, "textarea") || parseProps == nil {
		return "", false
	}
	if parseVal, parseOk := parseProps["value"].(string); parseOk {
		return parseVal, true
	}
	return "", false
}

// selectControlledValue reports the controlled value of a <select> and whether
// the element is a select carrying a string value.
func selectControlledValue(parseTag string, parseProps map[string]any) (string, bool) {
	if !strings.EqualFold(parseTag, "select") || parseProps == nil {
		return "", false
	}
	if parseVal, parseOk := parseProps["value"].(string); parseOk {
		return parseVal, true
	}
	return "", false
}

// optionMatchValue returns the value an <option> is matched against by a
// controlled <select>: its value prop if present, else its text content (React's
// fallback when an option has no value attribute).
func optionMatchValue(parseOption *Element) string {
	if parseOption.Props != nil {
		if parseVal, parseOk := parseOption.Props["value"].(string); parseOk {
			return parseVal
		}
	}
	if parseOption.hasDirectText {
		return parseOption.TextContent
	}
	var parseText strings.Builder
	for _, parseChild := range getElementChildren(parseOption) {
		switch parseTyped := parseChild.(type) {
		case string:
			parseText.WriteString(parseTyped)
		case *Element:
			// A ui.Text(...) child is a TEXT_ELEMENT carrying its content.
			if parseTag, parseOk := parseTyped.Type.(string); parseOk && parseTag == "TEXT_ELEMENT" {
				parseText.WriteString(parseTyped.TextContent)
			}
		}
	}
	return parseText.String()
}

// elementWithSelected returns a shallow copy of an <option> element with
// selected=true added to its props, without mutating the source element.
// Typed fast-lane options materialize their attribute view first so the copy
// keeps every compact attribute in the serialized output.
func elementWithSelected(parseOption *Element) *Element {
	var parseProps map[string]any
	if parseOption.Props == nil && parseOption.isCompactHostProps {
		// The materialized view is exclusively owned by this call, so the
		// selected flag can land in it directly without a defensive copy.
		parseProps = fastLanePropsView(parseOption.getHostAttrs, parseOption.Key, nil, false)
	} else {
		parseProps = make(map[string]any, len(parseOption.Props)+1)
		maps.Copy(parseProps, parseOption.Props)
	}
	parseProps["selected"] = true
	return &Element{
		Type:          parseOption.Type,
		Props:         parseProps,
		Children:      parseOption.Children,
		TextContent:   parseOption.TextContent,
		Key:           parseOption.Key,
		hasDirectText: parseOption.hasDirectText,
	}
}

// renderSelectChildrenToString renders a <select>'s children, marking each
// <option> whose value matches the select's controlled value as selected (unless
// it already declares selected). It recurses into <optgroup> so nested options
// are matched too.
func renderSelectChildrenToString(parseBuilder *strings.Builder, parseChildren []any, parseSelectValue string, parseCtx map[int64]any) error {
	for _, parseChild := range parseChildren {
		parseEl, parseOk := parseChild.(*Element)
		if !parseOk || parseEl == nil {
			if parseErr := renderChildrenToString(parseBuilder, []any{parseChild}, parseCtx); parseErr != nil {
				return parseErr
			}
			continue
		}
		parseType, _ := parseEl.Type.(string)
		switch {
		case strings.EqualFold(parseType, "option"):
			parseRender := parseEl
			if _, parseAlready := parseEl.Props["selected"]; !parseAlready && optionMatchValue(parseEl) == parseSelectValue {
				parseRender = elementWithSelected(parseEl)
			}
			if parseErr := renderElementToString(parseBuilder, parseRender, parseCtx); parseErr != nil {
				return parseErr
			}
		case strings.EqualFold(parseType, "optgroup"):
			parseBuilder.WriteByte('<')
			parseBuilder.WriteString(parseType)
			if parseEl.isCompactHostProps && parseEl.Props == nil {
				writeSSRCompactAttrs(parseBuilder, parseEl.getHostAttrs)
			} else {
				writeSSRProps(parseBuilder, parseEl.Props)
			}
			parseBuilder.WriteByte('>')
			if parseErr := renderSelectChildrenToString(parseBuilder, getElementChildren(parseEl), parseSelectValue, parseCtx); parseErr != nil {
				return parseErr
			}
			parseBuilder.WriteString("</")
			parseBuilder.WriteString(parseType)
			parseBuilder.WriteByte('>')
		default:
			if parseErr := renderElementToString(parseBuilder, parseEl, parseCtx); parseErr != nil {
				return parseErr
			}
		}
	}
	return nil
}

// normalizeFormValueProps maps React's uncontrolled form props to their
// controlled HTML equivalents for SSR: defaultValue -> value and
// defaultChecked -> checked (only when the controlled prop is not already set).
// Without this, defaultValue/defaultChecked serialize as browser-ignored
// attributes and the field renders empty/unchecked server-side. Returns the
// original map untouched when neither default* prop is present (the common case).
func normalizeFormValueProps(parseProps map[string]any) map[string]any {
	if parseProps == nil {
		return parseProps
	}
	_, parseHasDefaultValue := parseProps["defaultValue"]
	_, parseHasDefaultChecked := parseProps["defaultChecked"]
	if !parseHasDefaultValue && !parseHasDefaultChecked {
		return parseProps
	}
	parseOut := make(map[string]any, len(parseProps))
	maps.Copy(parseOut, parseProps)
	if parseDV, parseOk := parseOut["defaultValue"]; parseOk {
		if _, parseHasValue := parseOut["value"]; !parseHasValue {
			parseOut["value"] = parseDV
		}
		delete(parseOut, "defaultValue")
	}
	if parseDC, parseOk := parseOut["defaultChecked"]; parseOk {
		if _, parseHasChecked := parseOut["checked"]; !parseHasChecked {
			parseOut["checked"] = parseDC
		}
		delete(parseOut, "defaultChecked")
	}
	return parseOut
}

func renderHostElementToString(parseBuilder *strings.Builder, parseTag string, parseElement *Element, parseCtx map[int64]any) error {
	parseProps := normalizeFormValueProps(parseElement.Props)

	// A controlled <textarea value="x"> renders its value as text content, not as
	// a (browser-ignored) value attribute.
	if parseTextareaValue, parseIsTextarea := textareaControlledValue(parseTag, parseProps); parseIsTextarea {
		parseBuilder.WriteByte('<')
		parseBuilder.WriteString(parseTag)
		writeSSRProps(parseBuilder, parseProps, "value")
		parseBuilder.WriteByte('>')
		parseBuilder.WriteString(html.EscapeString(parseTextareaValue))
		parseBuilder.WriteString("</")
		parseBuilder.WriteString(parseTag)
		parseBuilder.WriteByte('>')
		return nil
	}

	// A controlled <select value="x"> marks the matching <option> as selected and
	// drops the (browser-ignored) value attribute from the <select> itself.
	if parseSelectValue, parseIsSelect := selectControlledValue(parseTag, parseProps); parseIsSelect {
		parseBuilder.WriteByte('<')
		parseBuilder.WriteString(parseTag)
		writeSSRProps(parseBuilder, parseProps, "value")
		parseBuilder.WriteByte('>')
		if parseErr := renderSelectChildrenToString(parseBuilder, getElementChildren(parseElement), parseSelectValue, parseCtx); parseErr != nil {
			return parseErr
		}
		parseBuilder.WriteString("</")
		parseBuilder.WriteString(parseTag)
		parseBuilder.WriteByte('>')
		return nil
	}

	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseTag)
	if parseElement.isCompactHostProps && parseProps == nil {
		writeSSRCompactAttrs(parseBuilder, parseElement.getHostAttrs)
	} else {
		writeSSRProps(parseBuilder, parseProps)
	}
	parseBuilder.WriteByte('>')

	if isVoidElement(parseTag) {
		return nil
	}

	if parseElement.hasDirectText {
		parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
	} else if parseErr := renderChildrenToString(parseBuilder, getElementChildren(parseElement), parseCtx); parseErr != nil {
		return parseErr
	}

	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseTag)
	parseBuilder.WriteByte('>')
	return nil
}

// renderChildrenToString is a core package helper.
func renderChildrenToString(parseBuilder *strings.Builder, parseChildren []any, parseCtx map[int64]any) error {
	for _, parseChild := range parseChildren {
		switch parseValue := parseChild.(type) {
		case nil:
			continue
		case *Element:
			if parseErr := renderElementToString(parseBuilder, parseValue, parseCtx); parseErr != nil {
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
// withSSRHookFiber runs render with a transient fiber installed as the current
// hook fiber, so a component may call hooks during server rendering. The fiber
// carries the inherited context values (for GoUseContextValue) and a fresh hooks
// store (GoUseState returns its initial value; GoUseRef/GoUseMemo compute;
// GoUseEffect queues an effect that is never committed, so it never runs on the
// server). The previous current fiber is restored afterwards.
func withSSRHookFiber(parseType any, parseProps map[string]any, parseCtx map[int64]any, parseRender func() *Element) *Element {
	parseFiber := &Fiber{typeOf: parseType, props: parseProps, contextValues: parseCtx}
	parsePrev := GetCurrentFiber()
	parsePrevOwner := currentFiberOwnerGoroutineID
	parseOwner, _ := parseCtx[ssrHookOwnerContextKey].(uint64)
	setCurrentFiberOwned(parseFiber, parseOwner)
	defer setCurrentFiberOwned(parsePrev, parsePrevOwner)
	return parseRender()
}

func resolveComponentElement(parseElement *Element, parseCtx map[int64]any) (*Element, error) {
	if parseComponent, parseOk := parseElement.Type.(*ComponentType); parseOk {
		return withSSRHookFiber(parseElement.Type, parseElement.Props, parseCtx, func() *Element {
			return parseComponent.Render(parseElement.Props)
		}), nil
	}

	parseValue := reflect.ValueOf(parseElement.Type)
	if !parseValue.IsValid() || parseValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("ssr: unsupported element type %T", parseElement.Type)
	}

	parseTyp := parseValue.Type()
	if parseTyp.NumOut() != 1 {
		return nil, fmt.Errorf("ssr: component %T must return exactly one value", parseElement.Type)
	}
	if parseTyp.Out(0) != reflect.TypeFor[*Element]() {
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

	parseResolved := withSSRHookFiber(parseElement.Type, parseElement.Props, parseCtx, func() *Element {
		parseResult := parseValue.Call(parseArgs)
		if len(parseResult) != 1 || parseResult[0].IsNil() {
			return nil
		}
		parseOut, _ := parseResult[0].Interface().(*Element)
		return parseOut
	})
	return parseResolved, nil
}

// buildComponentArg is a core package helper.
func buildComponentArg(parseTarget reflect.Type, parseProps map[string]any) (reflect.Value, error) {
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
func serializeProps(parseProps map[string]any) []string {
	if len(parseProps) == 0 {
		return nil
	}

	// Stack-allocated scratch for the common case (≤16 attributes): avoids a
	// per-element heap slice when collecting and sorting attribute keys.
	var parseKeyStorage [16]string
	parseKeys := parseKeyStorage[:0]
	for parseKey, parseValue := range parseProps {
		if shouldSkipSSRProp(parseKey, parseValue) {
			continue
		}
		parseKeys = append(parseKeys, parseKey)
	}
	slices.Sort(parseKeys)

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

// writeSSRProps writes sorted, validated attributes directly into the builder.
// It is the streaming twin of serializeProps: per-attribute it avoids the
// intermediate `name="value"` string (and the slice holding them) that the
// builder would immediately copy — the serializer's largest allocation source.
// writeSSRCompactAttrs serializes one typed fast-lane attribute slice with the
// same ordering, name normalization, sanitization, and escaping as
// writeSSRProps, so fast-lane and map-built elements produce identical markup.
func writeSSRCompactAttrs(parseBuilder *strings.Builder, parseAttrs []HostAttr) {
	if len(parseAttrs) == 0 {
		return
	}
	// writeSSRProps sorts by legacy prop-map key; mirror that so output stays
	// byte-identical. The slice is tiny, so an insertion sort on a stack copy
	// avoids heap traffic.
	var parseStorage [16]HostAttr
	parsePairs := parseStorage[:0]
	for _, parseAttr := range parseAttrs {
		parsePairs = append(parsePairs, HostAttr{Name: compactAttrPropName(parseAttr.Name), Value: parseAttr.Value})
	}
	for parseIndex := 1; parseIndex < len(parsePairs); parseIndex++ {
		parsePair := parsePairs[parseIndex]
		parseSlot := parseIndex
		for parseSlot > 0 && parsePairs[parseSlot-1].Name > parsePair.Name {
			parsePairs[parseSlot] = parsePairs[parseSlot-1]
			parseSlot--
		}
		parsePairs[parseSlot] = parsePair
	}
	for _, parsePair := range parsePairs {
		parseName := normalizeSSRAttrName(parsePair.Name)
		if !isValidSSRAttrName(parseName) {
			continue
		}
		parseValue := parsePair.Value
		if urlBearingSSRAttr(parseName) {
			parseValue = sanitizeSSRURLValue(parseValue)
		}
		parseBuilder.WriteByte(' ')
		parseBuilder.WriteString(parseName)
		parseBuilder.WriteString(`="`)
		parseBuilder.WriteString(html.EscapeString(parseValue))
		parseBuilder.WriteByte('"')
	}
}

func writeSSRProps(parseBuilder *strings.Builder, parseProps map[string]any, parseSkip ...string) {
	if len(parseProps) == 0 {
		return
	}
	// Stack-allocated scratch for the common case (≤16 attributes): avoids a
	// per-element heap slice when collecting and sorting attribute keys.
	var parseKeyStorage [16]string
	parseKeys := parseKeyStorage[:0]
	for parseKey, parseValue := range parseProps {
		if shouldSkipSSRProp(parseKey, parseValue) {
			continue
		}
		if len(parseSkip) > 0 && slices.Contains(parseSkip, parseKey) {
			continue
		}
		parseKeys = append(parseKeys, parseKey)
	}
	slices.Sort(parseKeys)
	for _, parseKey := range parseKeys {
		parseName := normalizeSSRAttrName(parseKey)
		if !isValidSSRAttrName(parseName) {
			continue
		}
		switch parseTyped := parseProps[parseKey].(type) {
		case bool:
			if parseTyped {
				parseBuilder.WriteByte(' ')
				parseBuilder.WriteString(parseName)
			}
		case string:
			if urlBearingSSRAttr(parseName) {
				parseTyped = sanitizeSSRURLValue(parseTyped)
			}
			parseBuilder.WriteByte(' ')
			parseBuilder.WriteString(parseName)
			parseBuilder.WriteString(`="`)
			parseBuilder.WriteString(html.EscapeString(parseTyped))
			parseBuilder.WriteByte('"')
		case map[string]string:
			parseBuilder.WriteByte(' ')
			parseBuilder.WriteString(parseName)
			parseBuilder.WriteString(`="`)
			parseBuilder.WriteString(html.EscapeString(serializeStyleMap(parseTyped)))
			parseBuilder.WriteByte('"')
		default:
			parseBuilder.WriteByte(' ')
			parseBuilder.WriteString(parseName)
			parseBuilder.WriteString(`="`)
			parseBuilder.WriteString(html.EscapeString(fmt.Sprint(parseTyped)))
			parseBuilder.WriteByte('"')
		}
	}
}

// shouldSkipSSRProp is a core package helper.
func shouldSkipSSRProp(parseKey string, parseValue any) bool {
	if parseKey == "children" || parseKey == "key" || parseValue == nil {
		return true
	}
	if parseKey == DOMRefKey {
		// A DOM ref sink is runtime plumbing, never an attribute — and on the SSR
		// path it never resolves to a node anyway.
		return true
	}
	if strings.HasPrefix(parseKey, "__gwc_prop__:") {
		return true
	}
	// Case-insensitive "on" prefix without strings.ToLower: the lowered copy
	// allocated one string per event-handler prop per element per render.
	if len(parseKey) >= 2 && (parseKey[0] == 'o' || parseKey[0] == 'O') && (parseKey[1] == 'n' || parseKey[1] == 'N') {
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

// urlBearingSSRAttr reports whether the browser interprets an attribute's value
// as a URL, so a javascript:/vbscript: scheme in it would execute on navigation.
func urlBearingSSRAttr(parseName string) bool {
	switch strings.ToLower(parseName) {
	case "href", "src", "action", "formaction", "poster", "data",
		"xlink:href", "ping", "background", "cite", "longdesc":
		return true
	}
	return false
}

// ssrURLScheme returns the scheme token of a URL value (the text before the first
// ':' that precedes any '/', '?', '#', or '\\'). Relative paths, absolute paths,
// and fragments have no scheme separator and report hasScheme=false.
func ssrURLScheme(parseValue string) (parseScheme string, parseHasScheme bool) {
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		switch parseValue[parseIndex] {
		case ':':
			return parseValue[:parseIndex], true
		case '/', '?', '#', '\\':
			return "", false
		}
	}
	return "", false
}

// normalizeSSRScheme lowercases a scheme token and strips whitespace and control
// characters so obfuscated schemes (java\tscript:, " javascript:", JavaScript:)
// collapse onto their real form before the denylist check.
func normalizeSSRScheme(parseScheme string) string {
	var parseBuilder strings.Builder
	for parseIndex := 0; parseIndex < len(parseScheme); parseIndex++ {
		parseByte := parseScheme[parseIndex]
		if parseByte <= ' ' || parseByte == 0x7f {
			continue
		}
		parseBuilder.WriteByte(parseByte)
	}
	return strings.ToLower(parseBuilder.String())
}

// sanitizeSSRURLValue neutralizes a script-executing scheme (javascript: or
// vbscript:) in a URL-bearing attribute value, tolerating the usual obfuscations.
// data:, http(s):, mailto:, relative, and fragment values pass through unchanged
// (data:image/... on <img src> is legitimate, so the denylist is narrow). A
// blocked value is replaced with the inert about:blank sentinel.
func sanitizeSSRURLValue(parseValue string) string {
	parseScheme, parseHasScheme := ssrURLScheme(strings.TrimSpace(parseValue))
	if !parseHasScheme {
		return parseValue
	}
	switch normalizeSSRScheme(parseScheme) {
	case "javascript", "vbscript":
		return "about:blank"
	}
	return parseValue
}

// SanitizeURLAttributeValue neutralizes a script-executing scheme
// (javascript: or vbscript:) in the value of a URL-bearing attribute such as
// href/src/action. Non-URL attributes and safe schemes (http(s), data, mailto,
// relative, fragment) are returned unchanged. It is shared by the SSR serializer
// and the browser DOM adapter so both render paths block the identical XSS
// vector at the point a value reaches the document.
func SanitizeURLAttributeValue(parseName, parseValue string) string {
	if !urlBearingSSRAttr(parseName) {
		return parseValue
	}
	return sanitizeSSRURLValue(parseValue)
}

// serializeSSRAttr is a core package helper.
func serializeSSRAttr(parseName string, parseValue any) (string, bool) {
	// Finding #57: reject attribute names that do not conform to the HTML/XML
	// attribute name production to prevent injection via a crafted name.
	if !isValidSSRAttrName(parseName) {
		return "", false
	}
	switch parseTyped := parseValue.(type) {
	case bool:
		if !parseTyped {
			return "", false
		}
		return parseName, true
	case string:
		if urlBearingSSRAttr(parseName) {
			parseTyped = sanitizeSSRURLValue(parseTyped)
		}
		return parseName + `="` + html.EscapeString(parseTyped) + `"`, true
	case map[string]string:
		// Serialized with sorted keys so SSR output is deterministic (#58).
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
	var parseKeyStorage [16]string
	parseKeys := parseKeyStorage[:0]
	for parseKey := range parseStyles {
		parseKeys = append(parseKeys, parseKey)
	}
	slices.Sort(parseKeys)

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
