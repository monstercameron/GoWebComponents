package html

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

const reactiveTextGetterProp = "__gwc_reactive_text_getter"

// Text creates a text node from a string-like value.
func Text(parseContent any) ui.Node {
	switch parseValue := parseContent.(type) {
	case nil:
		return nil
	case ui.Node:
		return parseValue
	case string:
		return ui.Text(parseValue)
	case fmt.Stringer:
		return ui.Text(parseValue.String())
	case func() string:
		return runtime.CreateElement(runtime.ReactiveTextNodeType, map[string]any{
			reactiveTextGetterProp: parseValue,
		})
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
		return ui.Text(fmt.Sprint(parseValue))
	default:
		return ui.Text(fmt.Sprint(parseValue))
	}
}

// Textf formats a text node without requiring an explicit fmt.Sprintf call first.
func Textf(format string, parseArgs ...any) ui.Node {
	return Text(fmt.Sprintf(format, parseArgs...))
}

// TextIf emits a text node only when the condition is true.
func TextIf(isCondition bool, parseContent any) ui.Node {
	if !isCondition {
		return nil
	}
	return Text(parseContent)
}

// Children normalizes mixed shorthand child inputs into the existing []ui.Node builder contract.
func Children(parseValues ...any) []ui.Node {
	if len(parseValues) == 0 {
		return nil
	}

	parseNodes := make([]ui.Node, 0, len(parseValues))
	for _, parseValue := range parseValues {
		appendNormalizedChild(&parseNodes, parseValue)
	}
	if len(parseNodes) == 0 {
		return nil
	}
	return parseNodes
}

// When returns a class fragment only when the condition is true.
func When(isCondition bool, parseClassName string) string {
	if !isCondition {
		return ""
	}
	return strings.TrimSpace(parseClassName)
}

// ClassNames joins class fragments while dropping empty values and normalizing whitespace.
func ClassNames(parseParts ...any) string {
	if len(parseParts) == 0 {
		return ""
	}

	parseFragments := make([]string, 0, len(parseParts))
	appendClassFragments(&parseFragments, parseParts...)
	return strings.Join(parseFragments, " ")
}

// If returns the node only when the condition is true.
func If(isCondition bool, parseNode ui.Node) ui.Node {
	if isCondition {
		return parseNode
	}
	return nil
}

// IfElse selects between two node branches inline.
func IfElse(isCondition bool, parseWhenTrue ui.Node, parseWhenFalse ui.Node) ui.Node {
	if isCondition {
		return parseWhenTrue
	}
	return parseWhenFalse
}

// Unless returns the node only when the condition is false.
func Unless(isCondition bool, parseNode ui.Node) ui.Node {
	if isCondition {
		return nil
	}
	return parseNode
}

// Map renders a typed slice into ui.Node values while preserving order.
func Map[T any](parseItems []T, render func(T) ui.Node) []ui.Node {
	if len(parseItems) == 0 {
		return nil
	}

	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, render(parseItem))
	}
	return parseNodes
}

// WithKey applies a reconciliation key to an existing node explicitly.
func WithKey(parseNode ui.Node, parseKey any) ui.Node {
	if parseNode == nil {
		return nil
	}
	if parseNode.Props == nil {
		parseNode.Props = make(map[string]any, 1)
	}
	parseNode.Props["key"] = parseKey
	return parseNode
}

// MapKeyed renders a typed slice into keyed ui.Node values while preserving order.
func MapKeyed[T any](parseItems []T, parseKey func(T) any, render func(T) ui.Node) []ui.Node {
	if len(parseItems) == 0 {
		return nil
	}

	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, WithKey(render(parseItem), parseKey(parseItem)))
	}
	return parseNodes
}

// FlatMap renders each item into zero or more nodes and flattens the results.
func FlatMap[T any](parseItems []T, render func(T) []ui.Node) []ui.Node {
	if len(parseItems) == 0 {
		return nil
	}

	var parseNodes []ui.Node
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, render(parseItem)...)
	}
	if len(parseNodes) == 0 {
		return nil
	}
	return parseNodes
}

// FilterMap renders each item conditionally and preserves order for realized nodes.
func FilterMap[T any](parseItems []T, render func(T) (ui.Node, bool)) []ui.Node {
	if len(parseItems) == 0 {
		return nil
	}

	var parseNodes []ui.Node
	for _, parseItem := range parseItems {
		parseNode, parseOk := render(parseItem)
		if parseOk && parseNode != nil {
			parseNodes = append(parseNodes, parseNode)
		}
	}
	if len(parseNodes) == 0 {
		return nil
	}
	return parseNodes
}

// Join inserts a separator between realized nodes only.
func Join(parseSeparator ui.Node, parseNodes ...ui.Node) []ui.Node {
	if len(parseNodes) == 0 {
		return nil
	}

	parseJoined := make([]ui.Node, 0, len(parseNodes)*2)
	parseRealized := 0
	for _, parseNode := range parseNodes {
		if parseNode == nil {
			continue
		}
		if parseRealized > 0 && parseSeparator != nil {
			parseJoined = append(parseJoined, parseSeparator)
		}
		parseJoined = append(parseJoined, parseNode)
		parseRealized++
	}
	if len(parseJoined) == 0 {
		return nil
	}
	return parseJoined
}

// Maybe renders a node only when a pointer value is present.
func Maybe[T any](parseValue *T, render func(T) ui.Node) ui.Node {
	if parseValue == nil {
		return nil
	}
	return render(*parseValue)
}

// OrElse returns the pointed value when present, otherwise the fallback.
func OrElse[T any](parseValue *T, parseFallback T) T {
	if parseValue == nil {
		return parseFallback
	}
	return *parseValue
}

// Coalesce returns the first non-nil pointer in order.
func Coalesce[T any](parseValues ...*T) *T {
	for _, parseValue := range parseValues {
		if parseValue != nil {
			return parseValue
		}
	}
	return nil
}

// SwitchBranch describes one branch in a Switch expression helper.
type SwitchBranch struct {
	value     any
	node      ui.Node
	isDefault bool
}

// Case creates a value-matching branch for Switch.
func Case(parseValue any, parseNode ui.Node) SwitchBranch {
	return SwitchBranch{value: parseValue, node: parseNode}
}

// Default creates the fallback branch for Switch.
func Default(parseNode ui.Node) SwitchBranch {
	return SwitchBranch{node: parseNode, isDefault: true}
}

// Switch selects the first matching branch and otherwise falls back to Default.
func Switch(parseValue any, parseBranches ...SwitchBranch) ui.Node {
	var parseFallback ui.Node
	for _, parseBranch := range parseBranches {
		if parseBranch.isDefault {
			if parseFallback == nil {
				parseFallback = parseBranch.node
			}
			continue
		}
		if reflect.DeepEqual(parseBranch.value, parseValue) {
			return parseBranch.node
		}
	}
	return parseFallback
}

// PropOption mutates Props through an explicit additive helper.
type PropOption interface {
	apply(*Props)
}

type optionFunc func(*Props)

// apply is a core package helper.
func (parseF optionFunc) apply(parseProps *Props) {
	parseF(parseProps)
}

// PropsOf builds Props from additive prop options using last-write-wins semantics.
func PropsOf(parseOptions ...PropOption) Props {
	var parseProps Props
	for _, parseOption := range parseOptions {
		if parseOption == nil {
			continue
		}
		parseOption.apply(&parseProps)
	}
	return parseProps
}

// WithProps applies additive prop options onto an existing Props value.
func WithProps(parseBase Props, parseOptions ...PropOption) Props {
	parseProps := cloneProps(parseBase)
	ApplyPropOptions(&parseProps, parseOptions...)
	return parseProps
}

// ApplyPropOptions applies options to parseProps in place, without the
// defensive clone WithProps performs.  Callers must own parseProps (and any
// maps it references): accumulation loops that build one fresh Props per
// element use this to avoid paying a full Props clone per option argument.
func ApplyPropOptions(parseProps *Props, parseOptions ...PropOption) {
	if parseProps == nil {
		return
	}
	for _, parseOption := range parseOptions {
		if parseOption == nil {
			continue
		}
		parseOption.apply(parseProps)
	}
}

// ID sets the id attribute on a Props.
func ID(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.ID = parseValue })
}

// Class sets the class attribute on a Props.
func Class(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Class = parseValue })
}

// For sets the for attribute on a Props.
func For(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.For = parseValue })
}

// Name sets the name attribute on a Props.
func Name(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Name = parseValue })
}

// Title sets the title attribute on a Props.
func Title(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Title = parseValue })
}

// Value sets the value attribute on a Props.
//
// When parseValue is the empty string the key is written into Raw["value"] so
// that toRuntimeProps emits it even for an empty controlled input.  Using
// Props.Value="" directly is silently dropped by toRuntimeProps; prefer
// html.Value("") or html.Attr("value","") when you need to clear a controlled
// input via the typed-builder path.
func Value(parseValue string) PropOption {
	if parseValue == "" {
		return optionFunc(func(parseProps *Props) {
			parseProps.Raw = mergeAnyMap(parseProps.Raw, map[string]any{"value": ""})
		})
	}
	return optionFunc(func(parseProps *Props) { parseProps.Value = parseValue })
}

// Placeholder sets the placeholder attribute on a Props.
func Placeholder(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Placeholder = parseValue })
}

// Type sets the type attribute on a Props.
func Type(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Type = parseValue })
}

// Href sets the href attribute on a Props.
func Href(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Href = parseValue })
}

// Src sets the src attribute on a Props.
func Src(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Src = parseValue })
}

// Role sets the role attribute on a Props.
func Role(parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Role = parseValue })
}

// Rows sets the rows attribute on a Props.
func Rows(parseValue int) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.Rows = parseValue })
}

// TabIndex sets the tabindex attribute on a Props.
//
// When parseValue is 0 the key is written into Raw["tabIndex"] so that
// toRuntimeProps emits it explicitly.  Props.TabIndex==0 is silently dropped
// by toRuntimeProps; prefer html.TabIndex(0) when tabindex=0 is intentional.
func TabIndex(parseValue int) PropOption {
	if parseValue == 0 {
		return optionFunc(func(parseProps *Props) {
			parseProps.Raw = mergeAnyMap(parseProps.Raw, map[string]any{"tabIndex": 0})
		})
	}
	return optionFunc(func(parseProps *Props) { parseProps.TabIndex = parseValue })
}

// Disabled sets the disabled boolean on a Props; passes true when no argument is given.
func Disabled(parseValues ...bool) PropOption {
	isParseEnabled := true
	if len(parseValues) > 0 {
		isParseEnabled = parseValues[0]
	}
	return optionFunc(func(parseProps *Props) { parseProps.Disabled = isParseEnabled })
}

// Checked sets the checked boolean on a Props; passes true when no argument is given.
func Checked(parseValues ...bool) PropOption {
	isParseEnabled := true
	if len(parseValues) > 0 {
		isParseEnabled = parseValues[0]
	}
	return optionFunc(func(parseProps *Props) { parseProps.Checked = isParseEnabled })
}

// Selected sets the selected boolean on a Props; passes true when no argument is given.
func Selected(parseValues ...bool) PropOption {
	isParseEnabled := true
	if len(parseValues) > 0 {
		isParseEnabled = parseValues[0]
	}
	return optionFunc(func(parseProps *Props) { parseProps.Selected = isParseEnabled })
}

// Required sets the required boolean on a Props; passes true when no argument is given.
func Required(parseValues ...bool) PropOption {
	isParseEnabled := true
	if len(parseValues) > 0 {
		isParseEnabled = parseValues[0]
	}
	return optionFunc(func(parseProps *Props) { parseProps.Required = isParseEnabled })
}

// ReadOnly sets the readonly boolean on a Props; passes true when no argument is given.
func ReadOnly(parseValues ...bool) PropOption {
	isParseEnabled := true
	if len(parseValues) > 0 {
		isParseEnabled = parseValues[0]
	}
	return optionFunc(func(parseProps *Props) { parseProps.ReadOnly = isParseEnabled })
}

// AutoFocus sets the autofocus boolean on a Props; passes true when no argument is given.
func AutoFocus(parseValues ...bool) PropOption {
	isParseEnabled := true
	if len(parseValues) > 0 {
		isParseEnabled = parseValues[0]
	}
	return optionFunc(func(parseProps *Props) { parseProps.AutoFocus = isParseEnabled })
}

// DisabledIf sets the disabled boolean conditionally on a Props.
func DisabledIf(isCondition bool) PropOption { return Disabled(isCondition) }

// ReadOnlyIf sets the readonly boolean conditionally on a Props.
func ReadOnlyIf(isCondition bool) PropOption { return ReadOnly(isCondition) }

// SelectedIf sets the selected boolean conditionally on a Props.
func SelectedIf(isCondition bool) PropOption { return Selected(isCondition) }

// Style merges the given CSS style map into the Props Style field.
func Style(parseValues map[string]string) PropOption {
	parseClone := cloneStringMap(parseValues)
	return optionFunc(func(parseProps *Props) {
		parseProps.Style = mergeStringMap(parseProps.Style, parseClone)
	})
}

// Data sets a single data-* attribute on the Props.
func Data(parseName string, parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) {
		parseProps.Data = mergeStringMap(parseProps.Data, map[string]string{parseName: parseValue})
	})
}

// Dataset merges multiple data-* attributes into the Props.
func Dataset(parseValues map[string]string) PropOption {
	parseClone := cloneStringMap(parseValues)
	return optionFunc(func(parseProps *Props) {
		parseProps.Data = mergeStringMap(parseProps.Data, parseClone)
	})
}

// Aria sets a single aria-* attribute on the Props.
func Aria(parseName string, parseValue string) PropOption {
	return optionFunc(func(parseProps *Props) {
		parseProps.Aria = mergeStringMap(parseProps.Aria, map[string]string{parseName: parseValue})
	})
}

// AriaSet merges multiple aria-* attributes into the Props.
func AriaSet(parseValues map[string]string) PropOption {
	parseClone := cloneStringMap(parseValues)
	return optionFunc(func(parseProps *Props) {
		parseProps.Aria = mergeStringMap(parseProps.Aria, parseClone)
	})
}

// Attr sets a single raw HTML attribute on the Props.
func Attr(parseKey string, parseValue any) PropOption {
	return optionFunc(func(parseProps *Props) {
		parseProps.Raw = mergeAnyMap(parseProps.Raw, map[string]any{parseKey: parseValue})
	})
}

// Attrs merges multiple raw HTML attributes into the Props.
func Attrs(parseValues map[string]any) PropOption {
	parseClone := cloneAnyMap(parseValues)
	return optionFunc(func(parseProps *Props) {
		parseProps.Raw = mergeAnyMap(parseProps.Raw, parseClone)
	})
}

// OnClick registers an onclick event handler on the Props.
func OnClick(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnClick = toHandler(parseCallback) })
}

// OnClickParallel registers one local click handler and marks the node as a public parallel-region click slot.
func OnClickParallel(parseSlotID string, parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) {
		parseProps.OnClick = toHandler(parseCallback)
		parseProps.Data = mergeStringMap(parseProps.Data, map[string]string{
			parallelRegionClickSlotDataKey: parseSlotID,
		})
	})
}

// OnInput registers an oninput event handler on the Props.
func OnInput(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnInput = toHandler(parseCallback) })
}

// OnChange registers an onchange event handler on the Props.
func OnChange(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnChange = toHandler(parseCallback) })
}

// OnSubmit registers an onsubmit event handler on the Props.
func OnSubmit(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnSubmit = toHandler(parseCallback) })
}

// OnKeyDown registers an onkeydown event handler on the Props.
func OnKeyDown(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnKeyDown = toHandler(parseCallback) })
}

// OnKeyUp registers an onkeyup event handler on the Props.
func OnKeyUp(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnKeyUp = toHandler(parseCallback) })
}

// OnMouseUp registers an onmouseup event handler on the Props.
func OnMouseUp(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnMouseUp = toHandler(parseCallback) })
}

// OnMouseDown registers an onmousedown event handler on the Props.
func OnMouseDown(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnMouseDown = toHandler(parseCallback) })
}

// OnFocus registers an onfocus event handler on the Props.
func OnFocus(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnFocus = toHandler(parseCallback) })
}

// OnBlur registers an onblur event handler on the Props.
func OnBlur(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnBlur = toHandler(parseCallback) })
}

// OnScroll registers an onscroll event handler on the Props.
func OnScroll(parseCallback any) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnScroll = toHandler(parseCallback) })
}

// Prevent wraps a callback so the event default is prevented before callback execution.
func Prevent(parseCallback any) any {
	return func(parseEvent ui.Event) {
		parseEvent.PreventDefault()
		invokeEventCallback(parseCallback, parseEvent)
	}
}

// Stop wraps a callback so propagation is stopped before callback execution.
func Stop(parseCallback any) any {
	return func(parseEvent ui.Event) {
		parseEvent.StopPropagation()
		invokeEventCallback(parseCallback, parseEvent)
	}
}

// Debounce delays callback execution until no newer event has arrived for delay.
func Debounce(parseDelay time.Duration, parseCallback any) any {
	if parseDelay <= 0 {
		return parseCallback
	}

	var parseMu sync.Mutex
	var parseTimer *time.Timer
	var parseLatest ui.Event

	return func(parseEvent ui.Event) {
		parseMu.Lock()
		parseLatest = parseEvent
		if parseTimer != nil {
			parseTimer.Stop()
		}
		parseTimer = time.AfterFunc(parseDelay, func() {
			parseMu.Lock()
			parseEventCopy := parseLatest
			parseTimer = nil
			parseMu.Unlock()
			invokeEventCallback(parseCallback, parseEventCopy)
		})
		parseMu.Unlock()
	}
}

// Throttle invokes immediately and then at most once per interval with the latest pending event.
func Throttle(parseInterval time.Duration, parseCallback any) any {
	if parseInterval <= 0 {
		return parseCallback
	}

	var parseMu sync.Mutex
	var parseTimer *time.Timer
	var parseLatest ui.Event
	var isPending bool
	var parseLastInvoke time.Time

	return func(parseEvent ui.Event) {
		parseMu.Lock()
		parseNow := time.Now()
		parseLatest = parseEvent
		if parseLastInvoke.IsZero() || parseNow.Sub(parseLastInvoke) >= parseInterval {
			parseLastInvoke = parseNow
			isPending = false
			if parseTimer != nil {
				parseTimer.Stop()
				parseTimer = nil
			}
			parseMu.Unlock()
			invokeEventCallback(parseCallback, parseEvent)
			return
		}

		isPending = true
		parseWait := parseInterval - parseNow.Sub(parseLastInvoke)
		if parseTimer != nil {
			parseTimer.Stop()
		}
		parseTimer = time.AfterFunc(parseWait, func() {
			parseMu.Lock()
			if !isPending {
				parseTimer = nil
				parseMu.Unlock()
				return
			}
			parseEventCopy := parseLatest
			isPending = false
			parseLastInvoke = time.Now()
			parseTimer = nil
			parseMu.Unlock()
			invokeEventCallback(parseCallback, parseEventCopy)
		})
		parseMu.Unlock()
	}
}

// appendNormalizedChild is a core package helper.
func appendNormalizedChild(parseDst *[]ui.Node, parseValue any) {
	switch parseTyped := parseValue.(type) {
	case nil:
		return
	case ui.Node:
		*parseDst = append(*parseDst, parseTyped)
	case string:
		*parseDst = append(*parseDst, Text(parseTyped))
	case fmt.Stringer:
		*parseDst = append(*parseDst, Text(parseTyped.String()))
	case func() string:
		*parseDst = append(*parseDst, Text(parseTyped))
	case []ui.Node:
		for _, parseChild := range parseTyped {
			appendNormalizedChild(parseDst, parseChild)
		}
	case []string:
		for _, parseChild2 := range parseTyped {
			appendNormalizedChild(parseDst, parseChild2)
		}
	case []any:
		for _, parseChild3 := range parseTyped {
			appendNormalizedChild(parseDst, parseChild3)
		}
	default:
		parseValueOf := reflect.ValueOf(parseValue)
		if parseValueOf.IsValid() {
			switch parseValueOf.Kind() {
			case reflect.Slice, reflect.Array:
				for parseIndex := 0; parseIndex < parseValueOf.Len(); parseIndex++ {
					appendNormalizedChild(parseDst, parseValueOf.Index(parseIndex).Interface())
				}
				return
			}
		}
		*parseDst = append(*parseDst, Text(parseTyped))
	}
}

// appendClassFragments is a core package helper.
func appendClassFragments(parseDst *[]string, parseValues ...any) {
	for _, parseValue := range parseValues {
		switch parseTyped := parseValue.(type) {
		case nil:
			continue
		case string:
			for parseFragment := range strings.FieldsSeq(parseTyped) {
				if parseFragment != "" {
					*parseDst = append(*parseDst, parseFragment)
				}
			}
		case fmt.Stringer:
			appendClassFragments(parseDst, parseTyped.String())
		case []string:
			for _, parseEntry := range parseTyped {
				appendClassFragments(parseDst, parseEntry)
			}
		case []any:
			appendClassFragments(parseDst, parseTyped...)
		default:
			parseValueOf := reflect.ValueOf(parseValue)
			if parseValueOf.IsValid() {
				switch parseValueOf.Kind() {
				case reflect.Slice, reflect.Array:
					for parseIndex := 0; parseIndex < parseValueOf.Len(); parseIndex++ {
						appendClassFragments(parseDst, parseValueOf.Index(parseIndex).Interface())
					}
					continue
				}
			}
			appendClassFragments(parseDst, fmt.Sprint(parseTyped))
		}
	}
}

// toHandler is a core package helper.
func toHandler(parseCallback any) ui.Handler {
	switch parseTyped := parseCallback.(type) {
	case nil:
		return ui.Handler{}
	case ui.Handler:
		return parseTyped
	}
	return ui.UseEvent(parseCallback)
}

// invokeEventCallback is a core package helper.
func invokeEventCallback(parseCallback any, parseEvent ui.Event) {
	if parseCallback == nil {
		return
	}
	parseValue := reflect.ValueOf(parseCallback)
	if !parseValue.IsValid() || parseValue.Kind() != reflect.Func {
		return
	}
	parseCallbackType := parseValue.Type()
	switch parseCallbackType.NumIn() {
	case 0:
		parseValue.Call(nil)
	case 1:
		parseEventValue := reflect.ValueOf(parseEvent)
		parseArgType := parseCallbackType.In(0)
		if parseEventValue.Type().AssignableTo(parseArgType) {
			parseValue.Call([]reflect.Value{parseEventValue})
			return
		}
		if parseEventValue.Type().ConvertibleTo(parseArgType) {
			parseValue.Call([]reflect.Value{parseEventValue.Convert(parseArgType)})
		}
	}
}

// cloneStringMap is a core package helper.
func cloneStringMap(parseInput map[string]string) map[string]string {
	if len(parseInput) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseInput))
	maps.Copy(parseClone, parseInput)
	return parseClone
}

// cloneProps is a core package helper.
func cloneProps(parseInput Props) Props {
	parseClone := parseInput
	parseClone.Style = cloneStringMap(parseInput.Style)
	parseClone.Data = cloneStringMap(parseInput.Data)
	parseClone.Aria = cloneStringMap(parseInput.Aria)
	parseClone.Raw = cloneAnyMap(parseInput.Raw)
	return parseClone
}

// mergeStringMap is a core package helper.
func mergeStringMap(parseDst map[string]string, parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return parseDst
	}
	if parseDst == nil {
		parseDst = make(map[string]string, len(parseValues))
	}
	maps.Copy(parseDst, parseValues)
	return parseDst
}

// cloneAnyMap is a core package helper.
func cloneAnyMap(parseInput map[string]any) map[string]any {
	if len(parseInput) == 0 {
		return nil
	}
	parseClone := make(map[string]any, len(parseInput))
	maps.Copy(parseClone, parseInput)
	return parseClone
}

// mergeAnyMap is a core package helper.
func mergeAnyMap(parseDst map[string]any, parseValues map[string]any) map[string]any {
	if len(parseValues) == 0 {
		return parseDst
	}
	if parseDst == nil {
		parseDst = make(map[string]any, len(parseValues))
	}
	maps.Copy(parseDst, parseValues)
	return parseDst
}
