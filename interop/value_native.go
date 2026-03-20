//go:build !js || !wasm
// +build !js !wasm

package interop

// GlobalThis returns an unavailable stub on non-browser builds.
func GlobalThis() (Value, error) {
	return Value{}, unavailable("GlobalThis", "")
}

// Present reports whether the value is defined and non-null.
func (v Value) Present() bool { return false }

// Truthy reports whether the value is truthy in the JavaScript sense.
func (v Value) Truthy() bool { return false }

// IsUndefined reports whether the value is undefined.
func (v Value) IsUndefined() bool { return true }

// IsNull reports whether the value is null.
func (v Value) IsNull() bool { return true }

// String returns the value as a string when possible.
func (v Value) String() string { return "" }

// Bool returns the value as a boolean when possible.
func (v Value) Bool() bool { return false }

// Int returns the value as an integer when possible.
func (v Value) Int() int { return 0 }

// Float returns the value as a float when possible.
func (v Value) Float() float64 { return 0 }

// Get reads a property from the wrapped value.
func (v Value) Get(name string) Value {
	_ = name
	return Value{}
}

// Set writes a property on the wrapped value.
func (v Value) Set(name string, value any) error {
	_ = name
	_ = value
	return unavailable("Value.Set", "")
}

// Delete removes a property from the wrapped value.
func (v Value) Delete(name string) error {
	_ = name
	return unavailable("Value.Delete", "")
}

// Call invokes a named method on the wrapped value.
func (v Value) Call(name string, args ...any) (Value, error) {
	_ = name
	_ = args
	return Value{}, unavailable("Value.Call", "")
}

// Invoke calls the wrapped value as a function.
func (v Value) Invoke(args ...any) (Value, error) {
	_ = args
	return Value{}, unavailable("Value.Invoke", "")
}

// ToGo converts the wrapped value into a JSON-shaped Go representation.
func (v Value) ToGo() (any, error) {
	return nil, unavailable("Value.ToGo", "")
}

// SetFunction binds a Go handler to a property on the wrapped value.
func (v Value) SetFunction(name string, handler func(args ...Value) any) (Subscription, error) {
	_ = name
	_ = handler
	return Subscription{}, unavailable("Value.SetFunction", "")
}
