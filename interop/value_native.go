//go:build !js || !wasm
// +build !js !wasm

package interop

// GetGlobalThis returns an unavailable stub on non-browser builds.
func GetGlobalThis() (Value, error) {
	return Value{}, unavailable("GlobalThis", "")
}

// Present reports whether the value is defined and non-null.
func (parseValue Value) Present() bool {
	_ = parseValue
	return false
}

// Truthy reports whether the value is truthy in the JavaScript sense.
func (parseValue Value) Truthy() bool {
	_ = parseValue
	return false
}

// IsUndefined reports whether the value is undefined.
func (parseValue Value) IsUndefined() bool {
	_ = parseValue
	return true
}

// IsNull reports whether the value is null.
func (parseValue Value) IsNull() bool {
	_ = parseValue
	return true
}

// String returns the value as a string when possible.
func (parseValue Value) String() string {
	_ = parseValue
	return ""
}

// Bool returns the value as a boolean when possible.
func (parseValue Value) Bool() bool {
	_ = parseValue
	return false
}

// Int returns the value as an integer when possible.
func (parseValue Value) Int() int {
	_ = parseValue
	return 0
}

// Float returns the value as a float when possible.
func (parseValue Value) Float() float64 {
	_ = parseValue
	return 0
}

// Get reads a property from the wrapped value.
func (parseValue Value) Get(parseValueName string) Value {
	_ = parseValue
	_ = parseValueName
	return Value{}
}

// Set writes a property on the wrapped value.
func (parseValue Value) Set(parseValueName string, parseValueData any) error {
	_ = parseValue
	_ = parseValueName
	_ = parseValueData
	return unavailable("Value.Set", "")
}

// Delete removes a property from the wrapped value.
func (parseValue Value) Delete(parseValueName string) error {
	_ = parseValue
	_ = parseValueName
	return unavailable("Value.Delete", "")
}

// Call invokes a named method on the wrapped value.
func (parseValue Value) Call(parseMethodName string, parseMethodArgs ...any) (Value, error) {
	_ = parseValue
	_ = parseMethodName
	_ = parseMethodArgs
	return Value{}, unavailable("Value.Call", "")
}

// Invoke calls the wrapped value as a function.
func (parseValue Value) Invoke(parseCallArgs ...any) (Value, error) {
	_ = parseValue
	_ = parseCallArgs
	return Value{}, unavailable("Value.Invoke", "")
}

// ToGo converts the wrapped value into a JSON-shaped Go representation.
func (parseValue Value) ToGo() (any, error) {
	_ = parseValue
	return nil, unavailable("Value.ToGo", "")
}

// SetFunction binds a Go handler to a property on the wrapped value.
func (parseValue Value) SetFunction(parseFunctionName string, parseFunctionHandler func(args ...Value) any) (Subscription, error) {
	_ = parseValue
	_ = parseFunctionName
	_ = parseFunctionHandler
	return Subscription{}, unavailable("Value.SetFunction", "")
}
