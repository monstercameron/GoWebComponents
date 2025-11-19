# DOM Package

**Location:** `/dom`

```
GoWebComponents/
├── dom/              ← YOU ARE HERE
│   ├── doc.go
│   ├── dom.go
│   ├── event.go
│   └── html.go
├── hooks/
├── state/
├── render/
├── router/
├── fetch/
├── internal/
├── examples/
└── ...
```

## Overview

The `dom` package provides a comprehensive API for creating and manipulating HTML elements in a type-safe, declarative way. It's the foundation for building user interfaces with GoWebComponents.

## Key Components

### Element Creation (`dom.go`)

- `CreateElement()` - Factory function for creating virtual DOM elements
- Element type definitions and attribute handling
- Virtual DOM node structure

### HTML Element Constructors (`html.go`)

Over 80 HTML element functions including:

- **Layout**: `Div`, `Span`, `Section`, `Article`, `Header`, `Footer`, `Nav`, `Main`, `Aside`
- **Text**: `H1`-`H6`, `P`, `Blockquote`, `Pre`, `Code`, `Em`, `Strong`, `Small`
- **Forms**: `Form`, `Input`, `Textarea`, `Select`, `Option`, `Button`, `Label`, `Fieldset`
- **Interactive**: `A`, `Button`, `Details`, `Summary`, `Dialog`
- **Media**: `Img`, `Video`, `Audio`, `Canvas`, `Svg`, `Picture`, `Source`
- **Tables**: `Table`, `Thead`, `Tbody`, `Tfoot`, `Tr`, `Th`, `Td`, `Caption`
- **Lists**: `Ul`, `Ol`, `Li`, `Dl`, `Dt`, `Dd`
- **Semantic**: `Time`, `Mark`, `Progress`, `Meter`, `Data`, `Output`

### Event Handling (`event.go`)

- `GoEvent` struct - Type-safe event wrapper
- Event helper methods:
  - `GetValue()` - Extract input values
  - `IsChecked()` - Get checkbox state
  - `GetKey()` - Keyboard event handling
  - `PreventDefault()` - Prevent default behavior
  - `StopPropagation()` - Stop event bubbling

### Type Definitions

```go
type Attrs = map[string]interface{}  // Element attributes
type Element struct {                // Virtual DOM element
    Type     string
    Props    Attrs
    Children []interface{}
}
```

## Usage Example

```go
import (
    "github.com/monstercameron/GoWebComponents/dom"
    "syscall/js"
)

func MyComponent(props dom.Attrs) *dom.Element {
    handleClick := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        println("Button clicked!")
        return nil
    })

    return dom.Div(dom.Attrs{
        "class": "container",
    },
        dom.H1(nil, dom.Text("Welcome")),
        dom.P(dom.Attrs{
            "class": "description",
        }, dom.Text("This is a demo component")),
        dom.Button(dom.Attrs{
            "onclick": handleClick,
            "class": "btn btn-primary",
        }, dom.Text("Click Me")),
    )
}
```

## Related Packages

- **[/render](../render/)** - Uses this package to render elements to actual DOM
- **[/hooks](../hooks/)** - Provides hooks that work with DOM elements
- **[/examples](../examples/)** - See practical usage in example applications

## Documentation

See [doc.go](./doc.go) for the official Go package documentation.
