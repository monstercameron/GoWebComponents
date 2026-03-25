//go:build js && wasm
// +build js,wasm

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

func recoverInteropException(op, target string, errp *error) {
	recovered := recover()
	if recovered == nil {
		return
	}
	*errp = wrapError(op, target, CodeRemote, jsExceptionError(recovered))
}

func jsExceptionError(recovered interface{}) error {
	switch typed := recovered.(type) {
	case nil:
		return errors.New("javascript exception")
	case error:
		return typed
	case string:
		if typed == "" {
			return errors.New("javascript exception")
		}
		return errors.New(typed)
	default:
		return fmt.Errorf("javascript exception: %v", typed)
	}
}

// GetGlobalThis returns the browser globalThis object wrapped in the generic interop value surface.
func GetGlobalThis() (Value, error) {
	global := js.Global()
	if global.IsUndefined() || global.IsNull() {
		return Value{}, unavailable("GetGlobalThis", "")
	}
	return Value{raw: global}, nil
}

func (v Value) rawValue() (js.Value, bool) {
	raw, ok := v.raw.(js.Value)
	return raw, ok
}

// Present reports whether the value is defined and non-null.
func (v Value) Present() bool {
	raw, ok := v.rawValue()
	return ok && !raw.IsUndefined() && !raw.IsNull()
}

// Truthy reports whether the value is truthy in the JavaScript sense.
func (v Value) Truthy() bool {
	raw, ok := v.rawValue()
	return ok && raw.Truthy()
}

// IsUndefined reports whether the value is undefined.
func (v Value) IsUndefined() bool {
	raw, ok := v.rawValue()
	return !ok || raw.IsUndefined()
}

// IsNull reports whether the value is null.
func (v Value) IsNull() bool {
	raw, ok := v.rawValue()
	return !ok || raw.IsNull()
}

// String returns the value as a string when possible.
func (v Value) String() string {
	raw, ok := v.rawValue()
	if !ok {
		return ""
	}
	return raw.String()
}

// Bool returns the value as a boolean when possible.
func (v Value) Bool() bool {
	raw, ok := v.rawValue()
	if !ok {
		return false
	}
	return raw.Bool()
}

// Int returns the value as an integer when possible.
func (v Value) Int() int {
	raw, ok := v.rawValue()
	if !ok {
		return 0
	}
	return raw.Int()
}

// Float returns the value as a float when possible.
func (v Value) Float() float64 {
	raw, ok := v.rawValue()
	if !ok {
		return 0
	}
	return raw.Float()
}

// Get reads a property from the wrapped value.
func (v Value) Get(name string) Value {
	raw, ok := v.rawValue()
	if !ok {
		return Value{}
	}
	return Value{raw: raw.Get(name)}
}

// Set writes a property on the wrapped value.
func (v Value) Set(name string, value any) error {
	raw, ok := v.rawValue()
	if !ok {
		return unavailable("Value.Set", name)
	}
	jsValue, err := goValueToJS("Value.Set", name, value)
	if err != nil {
		return err
	}
	raw.Set(name, jsValue)
	return nil
}

// Delete removes a property from the wrapped value.
func (v Value) Delete(name string) error {
	raw, ok := v.rawValue()
	if !ok {
		return unavailable("Value.Delete", name)
	}
	raw.Delete(name)
	return nil
}

// Call invokes a named method on the wrapped value.
func (v Value) Call(name string, args ...any) (Value, error) {
	raw, ok := v.rawValue()
	if !ok {
		return Value{}, unavailable("Value.Call", name)
	}
	var callErr error
	defer recoverInteropException("Value.Call", name, &callErr)
	jsArgs, err := goValuesToJS("Value.Call", name, args...)
	if err != nil {
		return Value{}, err
	}
	callee := raw.Get(name)
	if callee.Type() != js.TypeFunction {
		return Value{}, wrapError("Value.Call", name, CodeNotFunction, errors.New("property is not callable"))
	}
	trapped, err := trapCall(raw, name, jsArgs)
	if err != nil {
		return Value{}, err
	}
	result := Value{raw: trapped}
	if callErr != nil {
		return Value{}, callErr
	}
	return result, nil
}

// Invoke calls the wrapped value as a function.
func (v Value) Invoke(args ...any) (Value, error) {
	raw, ok := v.rawValue()
	if !ok {
		return Value{}, unavailable("Value.Invoke", "")
	}
	var invokeErr error
	defer recoverInteropException("Value.Invoke", "", &invokeErr)
	jsArgs, err := goValuesToJS("Value.Invoke", "", args...)
	if err != nil {
		return Value{}, err
	}
	if raw.Type() != js.TypeFunction {
		return Value{}, wrapError("Value.Invoke", "", CodeNotFunction, errors.New("value is not callable"))
	}
	trapped, err := trapInvoke(raw, jsArgs)
	if err != nil {
		return Value{}, err
	}
	result := Value{raw: trapped}
	if invokeErr != nil {
		return Value{}, invokeErr
	}
	return result, nil
}

func trapCall(target js.Value, method string, args []interface{}) (js.Value, error) {
	ensureCallTraps()
	envelope := callTrapFn.Invoke(target, method, jsArgsToArray(args))
	if envelope.IsUndefined() || envelope.IsNull() {
		return js.Undefined(), wrapError("Value.Call", method, CodeRemote, errors.New("javascript call failed without details"))
	}
	if envelope.Get("ok").Bool() {
		return envelope.Get("value"), nil
	}
	return js.Undefined(), wrapError("Value.Call", method, CodeRemote, errors.New(jsErrorText(envelope.Get("error"))))
}

func trapInvoke(fn js.Value, args []interface{}) (js.Value, error) {
	ensureCallTraps()
	envelope := invokeTrapFn.Invoke(fn, jsArgsToArray(args))
	if envelope.IsUndefined() || envelope.IsNull() {
		return js.Undefined(), wrapError("Value.Invoke", "", CodeRemote, errors.New("javascript invoke failed without details"))
	}
	if envelope.Get("ok").Bool() {
		return envelope.Get("value"), nil
	}
	return js.Undefined(), wrapError("Value.Invoke", "", CodeRemote, errors.New(jsErrorText(envelope.Get("error"))))
}

func ensureCallTraps() {
	callTrapInit.Do(func() {
		functionCtor := js.Global().Get("Function")
		callTrapFn = functionCtor.New("target", "method", "args", "try { return { ok: true, value: target[method].apply(target, args) }; } catch (error) { return { ok: false, error: error }; }")
		invokeTrapFn = functionCtor.New("fn", "args", "try { return { ok: true, value: fn.apply(undefined, args) }; } catch (error) { return { ok: false, error: error }; }")
	})
}

func jsArgsToArray(values []interface{}) js.Value {
	array := js.Global().Get("Array").New(len(values))
	for index, value := range values {
		array.SetIndex(index, value)
	}
	return array
}

func jsErrorText(value js.Value) string {
	if value.IsUndefined() || value.IsNull() {
		return "javascript exception"
	}
	if value.Type() == js.TypeString {
		text := strings.TrimSpace(value.String())
		if text != "" {
			return text
		}
		return "javascript exception"
	}
	message := strings.TrimSpace(value.Get("message").String())
	if message != "" {
		return message
	}
	summary := strings.TrimSpace(jsValueSummary(value))
	if summary != "" {
		return summary
	}
	return "javascript exception"
}

// ToGo converts the wrapped value into a JSON-shaped Go representation.
func (v Value) ToGo() (any, error) {
	raw, ok := v.rawValue()
	if !ok {
		return nil, unavailable("Value.ToGo", "")
	}
	return jsValueToGo("Value.ToGo", "", raw)
}

// SetFunction binds a Go handler to a property on the wrapped value and restores
// the previous property value when the returned subscription is canceled.
func (v Value) SetFunction(name string, handler func(args ...Value) any) (Subscription, error) {
	if handler == nil {
		return Subscription{}, wrapError("Value.SetFunction", name, CodeInvalid, errors.New("handler is nil"))
	}
	raw, ok := v.rawValue()
	if !ok {
		return Subscription{}, unavailable("Value.SetFunction", name)
	}

	previous := raw.Get(name)
	var once sync.Once
	var callback js.Func
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wrapped := make([]Value, len(args))
		for index, arg := range args {
			wrapped[index] = Value{raw: arg}
		}
		result := handler(wrapped...)
		if result == nil {
			return nil
		}
		jsResult, err := goValueToJS("Value.SetFunction", name, result)
		if err != nil {
			return js.Undefined()
		}
		return jsResult
	})
	raw.Set(name, callback)

	return Subscription{cancel: func() {
		once.Do(func() {
			callback.Release()
			if previous.IsUndefined() || previous.IsNull() {
				raw.Delete(name)
				return
			}
			raw.Set(name, previous)
		})
	}}, nil
}
