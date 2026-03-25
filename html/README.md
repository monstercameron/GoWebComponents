# GWC | Html Library

```text
  ____ ____      __
 / ___|\\ \\ \\    / /
| |  _ \\ \\ \\\\ /\\ / /
| |_| | \\ V  V /
 \\____|  \\_/\\_/
GoWebComponents (GWC)
```

## High-Level Overview

The `html` library exposes ergonomic HTML element constructors and helpers for building GWC component trees.

## Public APIs

### `github.com/monstercameron/GoWebComponents/html` (`package html`)
- Functions: `A`, `Aria`, `AriaSet`, `Article`, `Aside`, `Attr`, `Attrs`, `AutoFocus`, `Blockquote`, `Body`, `Br`, `Button`, `Case`, `Checked`, `Children`, `Class`, `ClassNames`, `Coalesce`, `Code`, `CustomElement`, `DNSPrefetch`, `Data`, `Dataset`, `Debounce`, `Default`, `Details`, `Dialog`, `Disabled`, `DisabledIf`, `Div`, `Em`, `Fieldset`, `FilterMap`, `FlatMap`, `Footer`, `For`, `Form`, `Fragment`, `H1`, `H2`, `H3`, `H4`, `H5`, `H6`, `Head`, `Header`, `HiddenInput`, `Hr`, `Href`, `Html`, `ID`, `If`, `IfElse`, `Img`, `Input`, `Join`, `Label`, `Legend`, `Li`, `Link`, `Main`, `Map`, `MapKeyed`, `Mark`, `Maybe`, `Meta`, `ModulePreload`, `Name`, `Nav`, `NoScript`, `OnBlur`, `OnChange`, `OnClick`, `OnFocus`, `OnInput`, `OnKeyDown`, `OnKeyUp`, `OnMouseUp`, `OnSubmit`, `Option`, `OrElse`, `P`, `Placeholder`, `Pre`, `Preconnect`, `Prefetch`, `Preload`, `Prevent`, `PropsOf`, `ReadOnly`, `ReadOnlyIf`, `RenderMarkdown`, `Required`, `ResolveMarkdownHref`, `Role`, `Rows`, `Script`, `Section`, `Select`, `Selected`, `SelectedIf`, `Small`, `Span`, `Src`, `Stop`, `Strong`, `Style`, `Summary`, `Switch`, `TabIndex`, `Table`, `Tag`, `Tbody`, `Td`, `Text`, `TextIf`, `Textarea`, `Textf`, `Th`, `Thead`, `Throttle`, `Time`, `Title`, `Tr`, `Type`, `Ul`, `Unless`, `Value`, `When`, `WithKey`, `WithProps`
- Types: `CustomElementProps`, `MarkdownClasses`, `MarkdownRenderOptions`, `PropOption`, `Props`, `SwitchBranch`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/html/shorthand` (`package shorthand`)
- Functions: `A`, `Aria`, `AriaSet`, `Article`, `Attr`, `Attrs`, `AutoFocus`, `Body`, `Br`, `Button`, `Case`, `Checked`, `Children`, `Class`, `ClassNames`, `Coalesce`, `Code`, `Data`, `Dataset`, `Debounce`, `Default`, `Details`, `Disabled`, `DisabledIf`, `Div`, `FilterMap`, `FlatMap`, `For`, `Form`, `Fragment`, `FromProps`, `H1`, `H2`, `H3`, `Head`, `Header`, `Hr`, `Href`, `Html`, `ID`, `If`, `IfElse`, `Img`, `Input`, `Join`, `Label`, `Li`, `Main`, `Map`, `MapKeyed`, `Mark`, `Maybe`, `Meta`, `Name`, `NoScript`, `OnBlur`, `OnChange`, `OnClick`, `OnFocus`, `OnInput`, `OnKeyDown`, `OnKeyUp`, `OnMouseUp`, `OnSubmit`, `Option`, `OrElse`, `P`, `Placeholder`, `Pre`, `Prevent`, `PropsOf`, `ReadOnly`, `ReadOnlyIf`, `Required`, `Role`, `Rows`, `Script`, `Section`, `Select`, `Selected`, `SelectedIf`, `Span`, `Src`, `Stop`, `Style`, `Summary`, `Switch`, `TabIndex`, `Table`, `Tag`, `Tbody`, `Td`, `Text`, `TextIf`, `Textf`, `Th`, `Thead`, `Throttle`, `Title`, `Tr`, `Type`, `Ul`, `Unless`, `Value`, `When`, `WithKey`, `WithProps`
- Types: `PropOption`, `Props`, `SwitchBranch`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `shorthand/` - Shorthand element helpers (4 files).
- `custom_elements_test.go` - Tests for custom_elements behavior
- `doc.go` - Package-level Go documentation
- `html.go` - Core implementation for html
- `html_native_test.go` - Tests for html_native behavior
- `html_wasm_test.go` - Tests for html_wasm behavior
- `markdown.go` - Core implementation for markdown
- `markdown_test.go` - Tests for markdown behavior
- `sugar.go` - Core implementation for sugar
- `sugar_test.go` - Tests for sugar behavior

## ASCII File List

```text
html/
|-- shorthand/
|   |-- doc.go
|   |-- shorthand.go
|   |-- shorthand_more_test.go
|   \-- shorthand_test.go
|-- custom_elements_test.go
|-- doc.go
|-- html.go
|-- html_native_test.go
|-- html_wasm_test.go
|-- markdown.go
|-- markdown_test.go
|-- sugar.go
\-- sugar_test.go
```
