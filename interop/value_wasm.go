//go:build js && wasm

package interop

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
)

var (
	callTrapInit sync.Once
	callTrapFn   js.Value
	invokeTrapFn js.Value
)

func recoverInteropException(parseOp, parseTarget string, parseErrp *error) {
	parseRecovered := recover()
	if parseRecovered == nil {
		return
	}
	*parseErrp = wrapError(parseOp, parseTarget, CodeRemote, jsExceptionError(parseRecovered))
}

func jsExceptionError(parseRecovered interface{}) error {
	switch parseTyped := parseRecovered.(type) {
	case nil:
		return errors.New("javascript exception")
	case error:
		return parseTyped
	case string:
		if parseTyped == "" {
			return errors.New("javascript exception")
		}
		return errors.New(parseTyped)
	default:
		return fmt.Errorf("javascript exception: %v", parseTyped)
	}
}

// GetGlobalThis returns the browser globalThis object wrapped in the generic interop value surface.
func GetGlobalThis() (Value, error) {
	parseGlobal := js.Global()
	if parseGlobal.IsUndefined() || parseGlobal.IsNull() {
		return Value{}, unavailable("GetGlobalThis", "")
	}
	return Value{raw: parseGlobal}, nil
}

func (parseV Value) rawValue() (js.Value, bool) {
	parseRaw, parseOk := parseV.raw.(js.Value)
	return parseRaw, parseOk
}

// Present reports whether the value is defined and non-null.
func (parseV Value) Present() bool {
	parseRaw, parseOk := parseV.rawValue()
	return parseOk && !parseRaw.IsUndefined() && !parseRaw.IsNull()
}

// Truthy reports whether the value is truthy in the JavaScript sense.
func (parseV Value) Truthy() bool {
	parseRaw, parseOk := parseV.rawValue()
	return parseOk && parseRaw.Truthy()
}

// IsUndefined reports whether the value is undefined.
func (parseV Value) IsUndefined() bool {
	parseRaw, parseOk := parseV.rawValue()
	return !parseOk || parseRaw.IsUndefined()
}

// IsNull reports whether the value is null.
func (parseV Value) IsNull() bool {
	parseRaw, parseOk := parseV.rawValue()
	return !parseOk || parseRaw.IsNull()
}

// String returns the value as a string when possible.
func (parseV Value) String() string {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return ""
	}
	return parseRaw.String()
}

// Bool returns the value as a boolean when possible.
func (parseV Value) Bool() bool {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return false
	}
	return parseRaw.Bool()
}

// Int returns the value as an integer when possible.
func (parseV Value) Int() int {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return 0
	}
	return parseRaw.Int()
}

// Float returns the value as a float when possible.
func (parseV Value) Float() float64 {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return 0
	}
	return parseRaw.Float()
}

// Get reads a property from the wrapped value.
func (parseV Value) Get(parseName string) Value {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return Value{}
	}
	return Value{raw: parseRaw.Get(parseName)}
}

// Set writes a property on the wrapped value.
func (parseV Value) Set(parseName string, parseValue any) error {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return unavailable("Value.Set", parseName)
	}
	parseJsValue, parseErr := goValueToJS("Value.Set", parseName, parseValue)
	if parseErr != nil {
		return parseErr
	}
	parseRaw.Set(parseName, parseJsValue)
	return nil
}

// Delete removes a property from the wrapped value.
func (parseV Value) Delete(parseName string) error {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return unavailable("Value.Delete", parseName)
	}
	parseRaw.Delete(parseName)
	return nil
}

// Call invokes a named method on the wrapped value.
func (parseV Value) Call(parseName string, parseArgs ...any) (Value, error) {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return Value{}, unavailable("Value.Call", parseName)
	}
	var parseCallErr error
	defer recoverInteropException("Value.Call", parseName, &parseCallErr)
	parseJsArgs, parseErr := goValuesToJS("Value.Call", parseName, parseArgs...)
	if parseErr != nil {
		return Value{}, parseErr
	}
	parseCallee := parseRaw.Get(parseName)
	if parseCallee.Type() != js.TypeFunction {
		return Value{}, wrapError("Value.Call", parseName, CodeNotFunction, errors.New("property is not callable"))
	}
	parseTrapped, parseErr := trapCall(parseRaw, parseName, parseJsArgs)
	if parseErr != nil {
		return Value{}, parseErr
	}
	parseResult := Value{raw: parseTrapped}
	if parseCallErr != nil {
		return Value{}, parseCallErr
	}
	return parseResult, nil
}

// Invoke calls the wrapped value as a function.
func (parseV Value) Invoke(parseArgs ...any) (Value, error) {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return Value{}, unavailable("Value.Invoke", "")
	}
	var parseInvokeErr error
	defer recoverInteropException("Value.Invoke", "", &parseInvokeErr)
	parseJsArgs, parseErr := goValuesToJS("Value.Invoke", "", parseArgs...)
	if parseErr != nil {
		return Value{}, parseErr
	}
	if parseRaw.Type() != js.TypeFunction {
		return Value{}, wrapError("Value.Invoke", "", CodeNotFunction, errors.New("value is not callable"))
	}
	parseTrapped, parseErr := trapInvoke(parseRaw, parseJsArgs)
	if parseErr != nil {
		return Value{}, parseErr
	}
	parseResult := Value{raw: parseTrapped}
	if parseInvokeErr != nil {
		return Value{}, parseInvokeErr
	}
	return parseResult, nil
}

func trapCall(parseTarget js.Value, parseMethod string, parseArgs []interface{}) (js.Value, error) {
	ensureCallTraps()
	parseEnvelope := callTrapFn.Invoke(parseTarget, parseMethod, jsArgsToArray(parseArgs))
	if parseEnvelope.IsUndefined() || parseEnvelope.IsNull() {
		return js.Undefined(), wrapError("Value.Call", parseMethod, CodeRemote, errors.New("javascript call failed without details"))
	}
	if parseEnvelope.Get("ok").Bool() {
		return parseEnvelope.Get("value"), nil
	}
	return js.Undefined(), wrapError("Value.Call", parseMethod, CodeRemote, errors.New(jsErrorText(parseEnvelope.Get("error"))))
}

func trapInvoke(parseFn js.Value, parseArgs []interface{}) (js.Value, error) {
	ensureCallTraps()
	parseEnvelope := invokeTrapFn.Invoke(parseFn, jsArgsToArray(parseArgs))
	if parseEnvelope.IsUndefined() || parseEnvelope.IsNull() {
		return js.Undefined(), wrapError("Value.Invoke", "", CodeRemote, errors.New("javascript invoke failed without details"))
	}
	if parseEnvelope.Get("ok").Bool() {
		return parseEnvelope.Get("value"), nil
	}
	return js.Undefined(), wrapError("Value.Invoke", "", CodeRemote, errors.New(jsErrorText(parseEnvelope.Get("error"))))
}

func ensureCallTraps() {
	callTrapInit.Do(func() {
		parseFunctionCtor := js.Global().Get("Function")
		callTrapFn = parseFunctionCtor.New("target", "method", "args", "try { return { ok: true, value: target[method].apply(target, args) }; } catch (error) { return { ok: false, error: error }; }")
		invokeTrapFn = parseFunctionCtor.New("fn", "args", "try { return { ok: true, value: fn.apply(undefined, args) }; } catch (error) { return { ok: false, error: error }; }")
	})
}

func jsArgsToArray(parseValues []interface{}) js.Value {
	parseArray := js.Global().Get("Array").New(len(parseValues))
	for parseIndex, parseValue := range parseValues {
		parseArray.SetIndex(parseIndex, parseValue)
	}
	return parseArray
}

func jsErrorText(parseValue js.Value) string {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return "javascript exception"
	}
	if parseValue.Type() == js.TypeString {
		parseText := strings.TrimSpace(parseValue.String())
		if parseText != "" {
			return parseText
		}
		return "javascript exception"
	}
	parseMessage := strings.TrimSpace(parseValue.Get("message").String())
	if parseMessage != "" {
		return parseMessage
	}
	parseSummary := strings.TrimSpace(jsValueSummary(parseValue))
	if parseSummary != "" {
		return parseSummary
	}
	return "javascript exception"
}

// ToGo converts the wrapped value into a JSON-shaped Go representation.
func (parseV Value) ToGo() (any, error) {
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return nil, unavailable("Value.ToGo", "")
	}
	return jsValueToGo("Value.ToGo", "", parseRaw)
}

// SetFunction binds a Go handler to a property on the wrapped value and restores
// the previous property value when the returned subscription is canceled.
func (parseV Value) SetFunction(parseName string, parseHandler func(args ...Value) any) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Value.SetFunction", parseName, CodeInvalid, errors.New("handler is nil"))
	}
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return Subscription{}, unavailable("Value.SetFunction", parseName)
	}

	parsePrevious := parseRaw.Get(parseName)
	var parseOnce sync.Once
	var parseCallback js.Func
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("SetFunction callback")
		parseWrapped := make([]Value, len(parseArgs))
		for parseIndex, parseArg := range parseArgs {
			parseWrapped[parseIndex] = Value{raw: parseArg}
		}
		parseResult := parseHandler(parseWrapped...)
		if parseResult == nil {
			return nil
		}
		parseJsResult, parseErr := goValueToJS("Value.SetFunction", parseName, parseResult)
		if parseErr != nil {
			return js.Undefined()
		}
		return parseJsResult
	})
	parseRaw.Set(parseName, parseCallback)

	return Subscription{cancel: func() {
		parseOnce.Do(func() {
			parseCallback.Release()
			if parsePrevious.IsUndefined() || parsePrevious.IsNull() {
				parseRaw.Delete(parseName)
				return
			}
			parseRaw.Set(parseName, parsePrevious)
		})
	}}, nil
}
