package html

//go:generate go run ./internal/attrgen ../..

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// formatBoolAttr renders a bool as the "true" or "false" string used by
// enumerated HTML and ARIA attributes.
func formatBoolAttr(isParseEnabled bool) string {
	if isParseEnabled {
		return "true"
	}
	return "false"
}

// CheckedIf sets the checked boolean conditionally on a Props.
func CheckedIf(isCondition bool) PropOption { return Checked(isCondition) }

// RequiredIf sets the required boolean conditionally on a Props.
func RequiredIf(isCondition bool) PropOption { return Required(isCondition) }

// IfProp applies parseOptions only when the condition is true; otherwise it is a
// no-op option (nil, which PropsOf and WithProps skip).
func IfProp(isCondition bool, parseOptions ...PropOption) PropOption {
	if !isCondition || len(parseOptions) == 0 {
		return nil
	}
	return optionFunc(func(parseProps *Props) {
		ApplyPropOptions(parseProps, parseOptions...)
	})
}

// UnlessProp applies parseOptions only when the condition is false; it is the
// negation of [IfProp].
func UnlessProp(isCondition bool, parseOptions ...PropOption) PropOption {
	return IfProp(!isCondition, parseOptions...)
}

// InputFn is the set of callback shapes OnInput and OnChange accept: no
// arguments, the element's current string value, the full event, or an existing
// ui.Handler (for example from ui.UseEvent). A callback
// of any other shape is a compile error instead of a silently ignored handler.
type InputFn interface {
	~func() | ~func(string) | ~func(ui.Event) | ui.Handler
}

// ChangeFn is the callback constraint for [OnChange]; it is identical to [InputFn].
type ChangeFn = InputFn

// OnInput registers an oninput event handler on the Props. The callback may take
// no arguments, the input's string value, or the event. To pass a handler that
// is already a ui.Handler (for example from ui.UseEvent) use [OnInputHandler].
func OnInput[F InputFn](parseCallback F) PropOption {
	parseNormalized := normalizeInputFn(parseCallback)
	return optionFunc(func(parseProps *Props) { parseProps.OnInput = toHandler(parseNormalized) })
}

// OnInputHandler registers an oninput event handler that is already a ui.Handler.
func OnInputHandler(parseHandler ui.Handler) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnInput = toHandler(parseHandler) })
}

// OnChange registers an onchange event handler on the Props. The callback may
// take no arguments, the control's string value, or the event. To pass a handler
// that is already a ui.Handler use [OnChangeHandler].
func OnChange[F ChangeFn](parseCallback F) PropOption {
	parseNormalized := normalizeInputFn(parseCallback)
	return optionFunc(func(parseProps *Props) { parseProps.OnChange = toHandler(parseNormalized) })
}

// OnChangeHandler registers an onchange event handler that is already a ui.Handler.
func OnChangeHandler(parseHandler ui.Handler) PropOption {
	return optionFunc(func(parseProps *Props) { parseProps.OnChange = toHandler(parseHandler) })
}

// normalizeInputFn converts a callback of a named func type to its plain func
// type so downstream type switches match it. A nil func yields a nil interface,
// which toHandler turns into an empty handler.
func normalizeInputFn[F InputFn](parseCallback F) any {
	switch parseTyped := any(parseCallback).(type) {
	case ui.Handler:
		return parseTyped
	case func():
		if parseTyped == nil {
			return nil
		}
		return parseTyped
	case func(string):
		if parseTyped == nil {
			return nil
		}
		return parseTyped
	case func(ui.Event):
		if parseTyped == nil {
			return nil
		}
		return parseTyped
	}
	parseValue := reflect.ValueOf(parseCallback)
	if parseValue.IsNil() {
		return nil
	}
	switch parseValue.Type().NumIn() {
	case 0:
		return parseValue.Convert(reflect.TypeOf(func() {})).Interface()
	default:
		if parseValue.Type().In(0).Kind() == reflect.String {
			return parseValue.Convert(reflect.TypeOf(func(string) {})).Interface()
		}
		return parseValue.Convert(reflect.TypeOf(func(ui.Event) {})).Interface()
	}
}
