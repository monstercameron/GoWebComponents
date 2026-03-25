package html

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

const reactiveTextGetterProp = "__gwc_reactive_text_getter"

// Text creates a text node from a string-like value.
func Text(content interface{}) ui.Node {
	switch value := content.(type) {
	case nil:
		return nil
	case ui.Node:
		return value
	case string:
		return ui.Text(value)
	case fmt.Stringer:
		return ui.Text(value.String())
	case func() string:
		return runtime.CreateElement(runtime.ReactiveTextNodeType, map[string]interface{}{
			reactiveTextGetterProp: value,
		})
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
		return ui.Text(fmt.Sprint(value))
	default:
		return ui.Text(fmt.Sprint(value))
	}
}

// Textf formats a text node without requiring an explicit fmt.Sprintf call first.
func Textf(format string, args ...interface{}) ui.Node {
	return Text(fmt.Sprintf(format, args...))
}

// TextIf emits a text node only when the condition is true.
func TextIf(condition bool, content interface{}) ui.Node {
	if !condition {
		return nil
	}
	return Text(content)
}

// Children normalizes mixed shorthand child inputs into the existing []ui.Node builder contract.
func Children(values ...interface{}) []ui.Node {
	if len(values) == 0 {
		return nil
	}

	nodes := make([]ui.Node, 0, len(values))
	for _, value := range values {
		appendNormalizedChild(&nodes, value)
	}
	if len(nodes) == 0 {
		return nil
	}
	return nodes
}

// When returns a class fragment only when the condition is true.
func When(condition bool, className string) string {
	if !condition {
		return ""
	}
	return strings.TrimSpace(className)
}

// ClassNames joins class fragments while dropping empty values and normalizing whitespace.
func ClassNames(parts ...interface{}) string {
	if len(parts) == 0 {
		return ""
	}

	fragments := make([]string, 0, len(parts))
	appendClassFragments(&fragments, parts...)
	return strings.Join(fragments, " ")
}

// If returns the node only when the condition is true.
func If(condition bool, node ui.Node) ui.Node {
	if condition {
		return node
	}
	return nil
}

// IfElse selects between two node branches inline.
func IfElse(condition bool, whenTrue ui.Node, whenFalse ui.Node) ui.Node {
	if condition {
		return whenTrue
	}
	return whenFalse
}

// Unless returns the node only when the condition is false.
func Unless(condition bool, node ui.Node) ui.Node {
	if condition {
		return nil
	}
	return node
}

// Map renders a typed slice into ui.Node values while preserving order.
func Map[T any](items []T, render func(T) ui.Node) []ui.Node {
	if len(items) == 0 {
		return nil
	}

	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, render(item))
	}
	return nodes
}

// WithKey applies a reconciliation key to an existing node explicitly.
func WithKey(node ui.Node, key interface{}) ui.Node {
	if node == nil {
		return nil
	}
	if node.Props == nil {
		node.Props = make(map[string]interface{}, 1)
	}
	node.Props["key"] = key
	return node
}

// MapKeyed renders a typed slice into keyed ui.Node values while preserving order.
func MapKeyed[T any](items []T, key func(T) interface{}, render func(T) ui.Node) []ui.Node {
	if len(items) == 0 {
		return nil
	}

	nodes := make([]ui.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, WithKey(render(item), key(item)))
	}
	return nodes
}

// FlatMap renders each item into zero or more nodes and flattens the results.
func FlatMap[T any](items []T, render func(T) []ui.Node) []ui.Node {
	if len(items) == 0 {
		return nil
	}

	var nodes []ui.Node
	for _, item := range items {
		nodes = append(nodes, render(item)...)
	}
	if len(nodes) == 0 {
		return nil
	}
	return nodes
}

// FilterMap renders each item conditionally and preserves order for realized nodes.
func FilterMap[T any](items []T, render func(T) (ui.Node, bool)) []ui.Node {
	if len(items) == 0 {
		return nil
	}

	var nodes []ui.Node
	for _, item := range items {
		node, ok := render(item)
		if ok && node != nil {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return nil
	}
	return nodes
}

// Join inserts a separator between realized nodes only.
func Join(separator ui.Node, nodes ...ui.Node) []ui.Node {
	if len(nodes) == 0 {
		return nil
	}

	joined := make([]ui.Node, 0, len(nodes)*2)
	realized := 0
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if realized > 0 && separator != nil {
			joined = append(joined, separator)
		}
		joined = append(joined, node)
		realized++
	}
	if len(joined) == 0 {
		return nil
	}
	return joined
}

// Maybe renders a node only when a pointer value is present.
func Maybe[T any](value *T, render func(T) ui.Node) ui.Node {
	if value == nil {
		return nil
	}
	return render(*value)
}

// OrElse returns the pointed value when present, otherwise the fallback.
func OrElse[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}

// Coalesce returns the first non-nil pointer in order.
func Coalesce[T any](values ...*T) *T {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

// SwitchBranch describes one branch in a Switch expression helper.
type SwitchBranch struct {
	value     interface{}
	node      ui.Node
	isDefault bool
}

// Case creates a value-matching branch for Switch.
func Case(value interface{}, node ui.Node) SwitchBranch {
	return SwitchBranch{value: value, node: node}
}

// Default creates the fallback branch for Switch.
func Default(node ui.Node) SwitchBranch {
	return SwitchBranch{node: node, isDefault: true}
}

// Switch selects the first matching branch and otherwise falls back to Default.
func Switch(value interface{}, branches ...SwitchBranch) ui.Node {
	var fallback ui.Node
	for _, branch := range branches {
		if branch.isDefault {
			if fallback == nil {
				fallback = branch.node
			}
			continue
		}
		if reflect.DeepEqual(branch.value, value) {
			return branch.node
		}
	}
	return fallback
}

// PropOption mutates Props through an explicit additive helper.
type PropOption interface {
	apply(*Props)
}

type optionFunc func(*Props)

func (f optionFunc) apply(props *Props) {
	f(props)
}

// PropsOf builds Props from additive prop options using last-write-wins semantics.
func PropsOf(options ...PropOption) Props {
	var props Props
	for _, option := range options {
		if option == nil {
			continue
		}
		option.apply(&props)
	}
	return props
}

// WithProps applies additive prop options onto an existing Props value.
func WithProps(base Props, options ...PropOption) Props {
	props := cloneProps(base)
	for _, option := range options {
		if option == nil {
			continue
		}
		option.apply(&props)
	}
	return props
}

func ID(value string) PropOption    { return optionFunc(func(props *Props) { props.ID = value }) }
func Class(value string) PropOption { return optionFunc(func(props *Props) { props.Class = value }) }
func For(value string) PropOption   { return optionFunc(func(props *Props) { props.For = value }) }
func Name(value string) PropOption  { return optionFunc(func(props *Props) { props.Name = value }) }
func Title(value string) PropOption { return optionFunc(func(props *Props) { props.Title = value }) }
func Value(value string) PropOption { return optionFunc(func(props *Props) { props.Value = value }) }
func Placeholder(value string) PropOption {
	return optionFunc(func(props *Props) { props.Placeholder = value })
}
func Type(value string) PropOption  { return optionFunc(func(props *Props) { props.Type = value }) }
func Href(value string) PropOption  { return optionFunc(func(props *Props) { props.Href = value }) }
func Src(value string) PropOption   { return optionFunc(func(props *Props) { props.Src = value }) }
func Role(value string) PropOption  { return optionFunc(func(props *Props) { props.Role = value }) }
func Rows(value int) PropOption     { return optionFunc(func(props *Props) { props.Rows = value }) }
func TabIndex(value int) PropOption { return optionFunc(func(props *Props) { props.TabIndex = value }) }

func Disabled(values ...bool) PropOption {
	enabled := true
	if len(values) > 0 {
		enabled = values[0]
	}
	return optionFunc(func(props *Props) { props.Disabled = enabled })
}

func Checked(values ...bool) PropOption {
	enabled := true
	if len(values) > 0 {
		enabled = values[0]
	}
	return optionFunc(func(props *Props) { props.Checked = enabled })
}

func Selected(values ...bool) PropOption {
	enabled := true
	if len(values) > 0 {
		enabled = values[0]
	}
	return optionFunc(func(props *Props) { props.Selected = enabled })
}

func Required(values ...bool) PropOption {
	enabled := true
	if len(values) > 0 {
		enabled = values[0]
	}
	return optionFunc(func(props *Props) { props.Required = enabled })
}

func ReadOnly(values ...bool) PropOption {
	enabled := true
	if len(values) > 0 {
		enabled = values[0]
	}
	return optionFunc(func(props *Props) { props.ReadOnly = enabled })
}

func AutoFocus(values ...bool) PropOption {
	enabled := true
	if len(values) > 0 {
		enabled = values[0]
	}
	return optionFunc(func(props *Props) { props.AutoFocus = enabled })
}

func DisabledIf(condition bool) PropOption { return Disabled(condition) }
func ReadOnlyIf(condition bool) PropOption { return ReadOnly(condition) }
func SelectedIf(condition bool) PropOption { return Selected(condition) }

func Style(values map[string]string) PropOption {
	clone := cloneStringMap(values)
	return optionFunc(func(props *Props) {
		props.Style = mergeStringMap(props.Style, clone)
	})
}

func Data(name string, value string) PropOption {
	return optionFunc(func(props *Props) {
		props.Data = mergeStringMap(props.Data, map[string]string{name: value})
	})
}

func Dataset(values map[string]string) PropOption {
	clone := cloneStringMap(values)
	return optionFunc(func(props *Props) {
		props.Data = mergeStringMap(props.Data, clone)
	})
}

func Aria(name string, value string) PropOption {
	return optionFunc(func(props *Props) {
		props.Aria = mergeStringMap(props.Aria, map[string]string{name: value})
	})
}

func AriaSet(values map[string]string) PropOption {
	clone := cloneStringMap(values)
	return optionFunc(func(props *Props) {
		props.Aria = mergeStringMap(props.Aria, clone)
	})
}

func Attr(key string, value interface{}) PropOption {
	return optionFunc(func(props *Props) {
		props.Raw = mergeAnyMap(props.Raw, map[string]interface{}{key: value})
	})
}

func Attrs(values map[string]interface{}) PropOption {
	clone := cloneAnyMap(values)
	return optionFunc(func(props *Props) {
		props.Raw = mergeAnyMap(props.Raw, clone)
	})
}

func OnClick(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnClick = toHandler(callback) })
}
func OnInput(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnInput = toHandler(callback) })
}
func OnChange(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnChange = toHandler(callback) })
}
func OnSubmit(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnSubmit = toHandler(callback) })
}
func OnKeyDown(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnKeyDown = toHandler(callback) })
}
func OnKeyUp(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnKeyUp = toHandler(callback) })
}
func OnMouseUp(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnMouseUp = toHandler(callback) })
}
func OnFocus(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnFocus = toHandler(callback) })
}
func OnBlur(callback interface{}) PropOption {
	return optionFunc(func(props *Props) { props.OnBlur = toHandler(callback) })
}

// Prevent wraps a callback so the event default is prevented before callback execution.
func Prevent(callback interface{}) interface{} {
	return func(event ui.Event) {
		event.PreventDefault()
		invokeEventCallback(callback, event)
	}
}

// Stop wraps a callback so propagation is stopped before callback execution.
func Stop(callback interface{}) interface{} {
	return func(event ui.Event) {
		event.StopPropagation()
		invokeEventCallback(callback, event)
	}
}

// Debounce delays callback execution until no newer event has arrived for delay.
func Debounce(delay time.Duration, callback interface{}) interface{} {
	if delay <= 0 {
		return callback
	}

	var mu sync.Mutex
	var timer *time.Timer
	var latest ui.Event

	return func(event ui.Event) {
		mu.Lock()
		latest = event
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(delay, func() {
			mu.Lock()
			eventCopy := latest
			timer = nil
			mu.Unlock()
			invokeEventCallback(callback, eventCopy)
		})
		mu.Unlock()
	}
}

// Throttle invokes immediately and then at most once per interval with the latest pending event.
func Throttle(interval time.Duration, callback interface{}) interface{} {
	if interval <= 0 {
		return callback
	}

	var mu sync.Mutex
	var timer *time.Timer
	var latest ui.Event
	var pending bool
	var lastInvoke time.Time

	return func(event ui.Event) {
		mu.Lock()
		now := time.Now()
		latest = event
		if lastInvoke.IsZero() || now.Sub(lastInvoke) >= interval {
			lastInvoke = now
			pending = false
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			mu.Unlock()
			invokeEventCallback(callback, event)
			return
		}

		pending = true
		wait := interval - now.Sub(lastInvoke)
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(wait, func() {
			mu.Lock()
			if !pending {
				timer = nil
				mu.Unlock()
				return
			}
			eventCopy := latest
			pending = false
			lastInvoke = time.Now()
			timer = nil
			mu.Unlock()
			invokeEventCallback(callback, eventCopy)
		})
		mu.Unlock()
	}
}

func appendNormalizedChild(dst *[]ui.Node, value interface{}) {
	switch typed := value.(type) {
	case nil:
		return
	case ui.Node:
		*dst = append(*dst, typed)
	case string:
		*dst = append(*dst, Text(typed))
	case fmt.Stringer:
		*dst = append(*dst, Text(typed.String()))
	case func() string:
		*dst = append(*dst, Text(typed))
	case []ui.Node:
		for _, child := range typed {
			appendNormalizedChild(dst, child)
		}
	case []string:
		for _, child := range typed {
			appendNormalizedChild(dst, child)
		}
	case []interface{}:
		for _, child := range typed {
			appendNormalizedChild(dst, child)
		}
	default:
		valueOf := reflect.ValueOf(value)
		if valueOf.IsValid() {
			switch valueOf.Kind() {
			case reflect.Slice, reflect.Array:
				for index := 0; index < valueOf.Len(); index++ {
					appendNormalizedChild(dst, valueOf.Index(index).Interface())
				}
				return
			}
		}
		*dst = append(*dst, Text(typed))
	}
}

func appendClassFragments(dst *[]string, values ...interface{}) {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			continue
		case string:
			for _, fragment := range strings.Fields(typed) {
				if fragment != "" {
					*dst = append(*dst, fragment)
				}
			}
		case fmt.Stringer:
			appendClassFragments(dst, typed.String())
		case []string:
			for _, entry := range typed {
				appendClassFragments(dst, entry)
			}
		case []interface{}:
			appendClassFragments(dst, typed...)
		default:
			valueOf := reflect.ValueOf(value)
			if valueOf.IsValid() {
				switch valueOf.Kind() {
				case reflect.Slice, reflect.Array:
					for index := 0; index < valueOf.Len(); index++ {
						appendClassFragments(dst, valueOf.Index(index).Interface())
					}
					continue
				}
			}
			appendClassFragments(dst, fmt.Sprint(typed))
		}
	}
}

func toHandler(callback interface{}) ui.Handler {
	switch typed := callback.(type) {
	case nil:
		return ui.Handler{}
	case ui.Handler:
		return typed
	default:
		return ui.UseEvent(typed)
	}
}

func invokeEventCallback(callback interface{}, event ui.Event) {
	if callback == nil {
		return
	}
	value := reflect.ValueOf(callback)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return
	}
	callbackType := value.Type()
	switch callbackType.NumIn() {
	case 0:
		value.Call(nil)
	case 1:
		eventValue := reflect.ValueOf(event)
		argType := callbackType.In(0)
		if eventValue.Type().AssignableTo(argType) {
			value.Call([]reflect.Value{eventValue})
			return
		}
		if eventValue.Type().ConvertibleTo(argType) {
			value.Call([]reflect.Value{eventValue.Convert(argType)})
		}
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func cloneProps(input Props) Props {
	clone := input
	clone.Style = cloneStringMap(input.Style)
	clone.Data = cloneStringMap(input.Data)
	clone.Aria = cloneStringMap(input.Aria)
	clone.Raw = cloneAnyMap(input.Raw)
	return clone
}

func mergeStringMap(dst map[string]string, values map[string]string) map[string]string {
	if len(values) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]string, len(values))
	}
	for key, value := range values {
		dst[key] = value
	}
	return dst
}

func cloneAnyMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	clone := make(map[string]interface{}, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func mergeAnyMap(dst map[string]interface{}, values map[string]interface{}) map[string]interface{} {
	if len(values) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]interface{}, len(values))
	}
	for key, value := range values {
		dst[key] = value
	}
	return dst
}
