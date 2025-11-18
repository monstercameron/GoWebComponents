# Examples

**Location:** `/examples`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/
├── render/
├── router/
├── fetch/
├── internal/
├── examples/         ← YOU ARE HERE
│   ├── 01-counter/
│   ├── 02-text-input/
│   ├── 03-toggle/
│   ├── 04-form/
│   ├── 05-todo-basic/
│   ├── 06-todo-advanced/
│   ├── 07-goroutines/
│   ├── 08-fetch/
│   ├── 09-atoms/
│   ├── 10-advanced-form/
│   ├── 11-blog/
│   ├── 12-portfolio-site/
│   ├── build.ps1
│   ├── shared/
│   └── static/
└── ...
```

## Overview

This directory contains 12 progressive examples demonstrating GoWebComponents features, from basic state management to complex applications with routing and data fetching.

## Quick Start

### View All Examples

Open `examples/static/index.html` in your browser to see the examples directory with links to all demos.

### Build All Examples

**Windows (PowerShell):**

```powershell
cd examples
.\build.ps1
```

**Linux/macOS:**

```bash
cd examples
chmod +x build.ps1
./build.ps1
```

The build script automatically discovers all `##-*` directories and compiles them to WebAssembly.

### Run Individual Example

```bash
cd examples/01-counter
GOOS=js GOARCH=wasm go build -o ../static/bin/counter.wasm main.go
cd ../static
python -m http.server 8080
# Open http://localhost:8080/01-counter/counter.html
```

## Examples Index

### 01 - Counter

**File:** `/examples/01-counter/main.go`

**Demonstrates:**

- `hooks.UseState` for number state
- Event handlers with `js.FuncOf`
- Basic component structure

**Concepts:**

```go
count, setCount := hooks.UseState(0)
setCount(count() + 1)  // Update state, triggers re-render
```

**Learn:** State management basics, event handling

---

### 02 - Text Input

**File:** `/examples/02-text-input/main.go`

**Demonstrates:**

- `hooks.UseState` for string state
- Input event handling
- Controlled components

**Concepts:**

```go
text, setText := hooks.UseState("")
handleInput := func(this js.Value, args []js.Value) interface{} {
    newText := args[0].Get("target").Get("value").String()
    setText(newText)
    return nil
}
```

**Learn:** Form inputs, controlled components

---

### 03 - Toggle

**File:** `/examples/03-toggle/main.go`

**Demonstrates:**

- `hooks.UseState` for boolean state
- Conditional rendering
- Dynamic attributes

**Concepts:**

```go
isOn, setIsOn := hooks.UseState(false)
className := func() string {
    if isOn() { return "on" }
    return "off"
}()
```

**Learn:** Boolean state, conditional UI

---

### 04 - Form

**File:** `/examples/04-form/main.go`

**Demonstrates:**

- Struct-based state management
- Multiple related inputs
- Form validation patterns

**Concepts:**

```go
type Person struct {
    Name string
    Age  int
}
person, setPerson := hooks.UseState(Person{})
```

**Learn:** Complex state types, forms

---

### 05 - Todo Basic

**File:** `/examples/05-todo-basic/main.go`

**Demonstrates:**

- Array/slice state management
- Dynamic list rendering
- Add/remove operations

**Concepts:**

```go
todos, setTodos := hooks.UseState([]string{})
newTodos := append(todos(), newTodo)
setTodos(newTodos)
```

**Learn:** Lists, dynamic UI, array operations

---

### 06 - Todo Advanced

**File:** `/examples/06-todo-advanced/main.go` (400+ lines)

**Demonstrates:**

- Complex state structures
- Filtering and searching
- Component composition
- Multiple sub-components

**Features:**

- Todo priorities (low, medium, high)
- Categories
- Due dates
- Filters (all, active, completed)
- Search functionality

**Learn:** Complex apps, component architecture

---

### 07 - Goroutines

**File:** `/examples/07-goroutines/main.go`

**Demonstrates:**

- Background tasks with goroutines
- State updates from goroutines
- Channel-based cancellation
- Concurrent timers

**Concepts:**

```go
go func() {
    for i := 0; i <= 100; i += 10 {
        select {
        case <-cancelChan:
            return
        case <-time.After(500 * time.Millisecond):
            setProgress(i)
        }
    }
}()
```

**Learn:** Concurrency, async state updates, cancellation

---

### 08 - Fetch

**File:** `/examples/08-fetch/main.go`

**Demonstrates:**

- `hooks.UseFetch` for data fetching
- Loading, error, and data states
- Manual refetch
- API integration

**Concepts:**

```go
getFetchState, refetch := hooks.UseFetch(url)
state := getFetchState()

if state.Loading { /* show spinner */ }
if state.Error != "" { /* show error */ }
// Use state.Data
```

**Learn:** HTTP requests, async data, loading states

---

### 09 - Atoms

**File:** `/examples/09-atoms/main.go` (220+ lines)

**Demonstrates:**

- `state.UseAtom` for global state
- State sharing between components
- Multiple independent atoms

**Components:**

- CounterController - Updates shared counter
- CounterDisplay - Displays shared counter
- TextInputComponent - Updates shared text
- TextDisplayComponent - Displays shared text

**Concepts:**

```go
count, setCount := state.UseAtom("shared-counter", 0)
// Any component can access "shared-counter"
```

**Learn:** Global state, component communication

---

### 10 - Advanced Form

**File:** `/examples/10-advanced-form/main.go`

**Demonstrates:**

- Multi-step forms
- Form validation
- Complex form state
- Conditional rendering

**Learn:** Form workflows, validation patterns

---

### 11 - Blog

**File:** `/examples/11-blog/main.go`

**Demonstrates:**

- Blog landing page layout
- Content-heavy components
- Semantic HTML structure

**Learn:** Content layouts, semantic markup

---

### 12 - Portfolio Site

**Files:** `/examples/12-portfolio-site/*.go` (20+ files)

**Demonstrates:**

- Complete SPA application
- Client-side routing
- Multiple pages/routes
- Dark mode toggle
- Contact forms
- Navigation
- Responsive design

**Components:**

- App.go - Main application
- Router.go - Route configuration
- Navbar.go - Navigation bar
- Hero.go - Landing page hero
- Features.go - Feature showcase
- Portfolio.go - Project gallery
- Contact.go - Contact form
- And more...

**Learn:** Full applications, routing, layouts

---

## Build System

### Auto-Discovery Build Script

The `build.ps1` script automatically:

1. Scans for directories matching `##-*` pattern
2. Compiles `main.go` in each directory
3. Outputs WASM files to `static/bin/`
4. Shows build time and file size
5. Reports success/failure

**Output Example:**

```
Building 01-counter...
✅ 01-counter built successfully (2.8 MB) in 1.2s

Building 02-text-input...
✅ 02-text-input built successfully (2.8 MB) in 1.1s

...

Total: 10/12 examples built successfully
```

### Shared Types

`/examples/shared/types.go` provides common type aliases:

```go
package shared

type Attrs = dom.Attrs
type Element = render.Element
```

Import in examples for cleaner code:

```go
import "github.com/monstercameron/GoWebComponents/examples/shared"

func MyComponent(props shared.Attrs) *shared.Element {
    // ...
}
```

## Static Assets

### `/examples/static/`

**Structure:**

```
static/
├── index.html           # Examples directory page
├── css/
│   └── tailwind.css     # Tailwind CSS styles
├── script/
│   └── wasm_exec.js     # Go WASM runtime
└── bin/
    ├── counter.wasm
    ├── text-input.wasm
    ├── toggle.wasm
    └── ...              # All compiled WASM files
```

**HTML Template:**

Each example has an HTML file in its directory:

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Example Name</title>
    <link rel="stylesheet" href="../static/css/tailwind.css" />
    <script src="../static/script/wasm_exec.js"></script>
  </head>
  <body>
    <div id="app"></div>
    <script>
      const go = new Go();
      WebAssembly.instantiateStreaming(
        fetch("../static/bin/example.wasm"),
        go.importObject
      ).then((result) => go.run(result.instance));
    </script>
  </body>
</html>
```

## Learning Path

**Beginner:**

1. 01-counter (State basics)
2. 02-text-input (Form inputs)
3. 03-toggle (Conditionals)
4. 05-todo-basic (Lists)

**Intermediate:** 5. 04-form (Complex state) 6. 07-goroutines (Concurrency) 7. 08-fetch (HTTP) 8. 09-atoms (Global state)

**Advanced:** 9. 06-todo-advanced (Full CRUD app) 10. 10-advanced-form (Multi-step forms) 11. 12-portfolio-site (Complete SPA)

## Running Examples Locally

### Option 1: Python HTTP Server

```bash
cd examples/static
python -m http.server 8080
# Open http://localhost:8080
```

### Option 2: Go HTTP Server

```bash
cd examples/static
go run ../../tools/serve.go
```

### Option 3: Live Reload

```bash
cd tools
./livereload.sh  # or livereload.ps1 on Windows
```

## Related Packages

All examples use:

- **[/dom](../dom/)** - Element creation
- **[/hooks](../hooks/)** - State and effects
- **[/render](../render/)** - Rendering engine

Some examples use:

- **[/state](../state/)** - Global state (09-atoms)
- **[/fetch](../fetch/)** - HTTP requests (08-fetch)
- **[/router](../router/)** - Routing (12-portfolio-site)

## Contributing Examples

To add a new example:

1. Create directory: `examples/##-example-name/`
2. Add `main.go` with your example code
3. Create `example-name.html` loader file
4. Run `build.ps1` to compile
5. Test at `http://localhost:8080/##-example-name/example-name.html`
6. Add entry to `static/index.html`

## Troubleshooting

**WASM file not found:**

- Ensure you ran `build.ps1`
- Check that WASM file exists in `static/bin/`

**Example not rendering:**

- Check browser console for errors
- Verify `wasm_exec.js` is loaded
- Ensure `#app` div exists in HTML

**Build fails:**

- Check Go version (1.22.0+)
- Ensure all imports are correct
- Run `go mod tidy`

## Documentation

Each example's code includes inline comments explaining the concepts. For API documentation, see the main package READMEs.
