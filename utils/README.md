# Utils Package

**Location:** `/utils`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/
├── render/
├── router/
├── fetch/
├── internal/
├── examples/
├── test/
├── tools/
└── utils/            ← YOU ARE HERE
    ├── utils.go
    └── utils_production.go
```

## Overview

The `utils` package provides utility functions and helpers used throughout GoWebComponents. These are general-purpose tools for common operations.

## Build Tags

The package uses Go build tags to provide different implementations for development and production:

- **`utils.go`** - Development utilities (debugging, logging)
- **`utils_production.go`** - Production-optimized versions (minimal overhead)

## Functions

### Development Utilities

These functions are available in development builds:

```go
// Debug logging
func DebugLog(message string)
func DebugLogf(format string, args ...interface{})

// Performance timing
func TimeStart(label string) func()
func TimeEnd(label string, start time.Time)

// Memory tracking
func LogMemoryUsage()
```

**Usage:**

```go
import "github.com/monstercameron/GoWebComponents/utils"

func MyComponent(props dom.Attrs) *dom.Element {
    defer utils.TimeStart("MyComponent render")()

    utils.DebugLog("Rendering MyComponent")

    // Component logic

    utils.LogMemoryUsage()

    return dom.Div(nil, dom.Text("My Component"))
}
```

**Output (Development):**

```
[DEBUG] Rendering MyComponent
[MEMORY] Heap: 2.5 MB, Stack: 64 KB
[TIME] MyComponent render: 12ms
```

### Production Build

In production builds (with `-tags production`), these functions become no-ops for zero overhead:

```bash
GOOS=js GOARCH=wasm go build -tags production -o main.wasm
```

### Type Helpers

```go
// Safe type assertions
func ToString(val interface{}) string
func ToInt(val interface{}) int
func ToBool(val interface{}) bool
func ToFloat(val interface{}) float64

// Map helpers
func GetMapValue(m map[string]interface{}, key string, defaultVal interface{}) interface{}
func SetMapValue(m map[string]interface{}, key string, val interface{})

// Slice helpers
func Contains(slice []interface{}, item interface{}) bool
func IndexOf(slice []interface{}, item interface{}) int
func Remove(slice []interface{}, index int) []interface{}
```

**Usage:**

```go
// Safe type conversion
props := dom.Attrs{"count": 42, "name": "test"}

count := utils.ToInt(props["count"])       // 42
name := utils.ToString(props["name"])      // "test"
enabled := utils.ToBool(props["enabled"])  // false (missing key)

// Map operations
value := utils.GetMapValue(props, "missing", "default") // "default"

// Slice operations
items := []interface{}{1, 2, 3, 4, 5}
if utils.Contains(items, 3) {
    fmt.Println("Found 3")
}

index := utils.IndexOf(items, 4)  // 3
newItems := utils.Remove(items, index)  // [1, 2, 3, 5]
```

### String Utilities

```go
// String manipulation
func Capitalize(s string) string
func CamelCase(s string) string
func SnakeCase(s string) string
func KebabCase(s string) string

// String checks
func IsEmpty(s string) bool
func IsBlank(s string) bool  // Empty or only whitespace
```

**Usage:**

```go
utils.Capitalize("hello")    // "Hello"
utils.CamelCase("hello-world")  // "helloWorld"
utils.SnakeCase("HelloWorld")   // "hello_world"
utils.KebabCase("HelloWorld")   // "hello-world"

utils.IsEmpty("")     // true
utils.IsBlank("  ")   // true
```

### Error Handling

```go
// Panic recovery
func Recover(fn func()) (err error)

// Error formatting
func FormatError(err error) string
func WrapError(err error, message string) error
```

**Usage:**

```go
err := utils.Recover(func() {
    // Code that might panic
    riskyOperation()
})

if err != nil {
    fmt.Println("Recovered from panic:", utils.FormatError(err))
}

// Wrap errors with context
if err != nil {
    return utils.WrapError(err, "failed to load user data")
}
```

### JavaScript Interop Helpers

```go
// Convert Go values to JavaScript
func ToJSValue(val interface{}) js.Value

// Convert JavaScript values to Go
func FromJSValue(val js.Value) interface{}

// Check JavaScript types
func IsJSUndefined(val js.Value) bool
func IsJSNull(val js.Value) bool
func IsJSFunction(val js.Value) bool
func IsJSObject(val js.Value) bool
```

**Usage:**

```go
// Go to JS
goMap := map[string]interface{}{"name": "John", "age": 30}
jsObj := utils.ToJSValue(goMap)
js.Global().Get("console").Call("log", jsObj)

// JS to Go
jsValue := js.Global().Get("someJSObject")
goValue := utils.FromJSValue(jsValue)

// Type checking
if utils.IsJSUndefined(jsValue) {
    fmt.Println("Value is undefined")
}
```

## Build Optimization

### Production Build

For production, use build tags to enable optimizations:

```bash
GOOS=js GOARCH=wasm go build \
  -tags production \
  -ldflags="-s -w" \
  -o main.wasm
```

**Flags explained:**

- `-tags production` - Use production utils (no debug logging)
- `-ldflags="-s -w"` - Strip debug info and symbol table
- Result: Smaller WASM binary, faster execution

### Development Build

For development, include debugging:

```bash
GOOS=js GOARCH=wasm go build -o main.wasm
```

## Common Patterns

### Conditional Debug Logging

```go
const DEBUG = true  // Set to false in production

func MyFunc() {
    if DEBUG {
        utils.DebugLog("Starting MyFunc")
    }

    // Function logic

    if DEBUG {
        utils.DebugLog("MyFunc completed")
    }
}
```

### Performance Profiling

```go
func ExpensiveOperation() {
    defer utils.TimeStart("ExpensiveOperation")()

    // Complex computation
    for i := 0; i < 1000000; i++ {
        // Work...
    }
}
```

### Safe Property Access

```go
func SafeComponent(props dom.Attrs) *dom.Element {
    // Safe with defaults
    title := utils.ToString(
        utils.GetMapValue(props, "title", "Default Title"))

    count := utils.ToInt(
        utils.GetMapValue(props, "count", 0))

    enabled := utils.ToBool(
        utils.GetMapValue(props, "enabled", true))

    return dom.Div(nil,
        dom.H1(nil, dom.Text(title)),
        dom.P(nil, dom.Text(fmt.Sprintf("Count: %d", count))),
    )
}
```

## Best Practices

### 1. Use Type Helpers for Props

```go
// ✅ Good - safe
count := utils.ToInt(props["count"])

// ❌ Bad - can panic
count := props["count"].(int)
```

### 2. Profile Performance-Critical Code

```go
// ✅ Good
defer utils.TimeStart("render")()

// ❌ Bad - no visibility into performance
```

### 3. Use Production Builds for Deployment

```bash
# ✅ Good
go build -tags production -ldflags="-s -w"

# ❌ Bad - debug overhead in production
go build
```

## Related Packages

- **[/dom](../dom/)** - Uses utils for type safety
- **[/hooks](../hooks/)** - Uses utils for debugging
- **[/internal/runtime](../internal/runtime/)** - Uses utils for performance tracking

## Contributing

When adding utilities:

1. Keep functions simple and focused
2. Provide both development and production versions
3. Include examples in comments
4. Add tests in `utils_test.go`
5. Document in this README

## Documentation

See [utils.go](./utils.go) for implementation details and inline documentation.
