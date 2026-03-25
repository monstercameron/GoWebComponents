//go:build !js || !wasm
// +build !js !wasm

package interop

// GetGlobalThis returns an unavailable stub on non-browser builds.
func GetGlobalThis() (Value, error) {
	return Value{}, unavailable("GlobalThis", "")
}

// Present reports whether the value is defined and non-null.
func (parseV Value) Present() bool { return false }

// Truthy reports whether the value is truthy in the JavaScript sense.
func (parseV Value) Truthy() bool { return false }

// IsUndefined reports whether the value is undefined.
func (parseV Value) IsUndefined() bool { return true }

// IsNull reports whether the value is null.
func (parseV Value) IsNull() bool { return true }

// String returns the value as a string when possible.
func (parseV Value) String() string { return "" }

// Bool returns the value as a boolean when possible.
func (parseV Value) Bool() bool { return false }

// Int returns the value as an integer when possible.
func (parseV Value) Int() int { return 0 }

// Float returns the value as a float when possible.
func (parseV Value) Float() float64 { return 0 }

// Get reads a property from the wrapped value.
func (parseV Value) Get(parseName string) Value {
	_ = parseName
	return Value{}
}

// Set writes a property on the wrapped value.
func (parseV Value) Set(parseName string, parseValue any) error {
	_ = parseName
	_ = parseValue
	return unavailable("Value.Set", "")
}

// Delete removes a property from the wrapped value.
func (parseV Value) Delete(parseName string) error {
	_ = parseName
	return unavailable("Value.Delete", "")
}

// Call invokes a named method on the wrapped value.
func (parseV Value) Call(parseName string, parseArgs ...any) (Value, error) {
	_ = parseName
	_ = parseArgs
	return Value{}, unavailable("Value.Call", "")
}

// Invoke calls the wrapped value as a function.
func (parseV Value) Invoke(parseArgs ...any) (Value, error) {
	_ = parseArgs
	return Value{}, unavailable("Value.Invoke", "")
}

// ToGo converts the wrapped value into a JSON-shaped Go representation.
func (parseV Value) ToGo() (any, error) {
	return nil, unavailable("Value.ToGo", "")
}

// SetFunction binds a Go handler to a property on the wrapped value.
func (parseV Value) SetFunction(parseName string, parseHandler func(args ...Value) any) (Subscription, error) {
	_ = parseName
	_ = parseHandler
	return Subscription{}, unavailable("Value.SetFunction", "")
}
