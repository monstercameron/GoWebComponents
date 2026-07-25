//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/router"
)

// DocsPage renders complete API documentation for the GoWebComponents framework.
// Features organized sections for types, hooks, HTML elements, events, and utilities
// with interactive examples and detailed explanations for each API.
func DocsPage(_ Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-[#0a0a0a] text-white"},

		// Documentation-specific navigation bar
		DocsNavBar(nil),

		// Documentation content
		Div(
			Attrs{"class": "pt-32 pb-20"},
			Div(
				Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},

				// Header
				Div(
					Attrs{"class": "text-center mb-16"},
					H1(
						Attrs{"class": "text-4xl font-bold text-white mb-4"},
						"📚 GoWebComponents API Documentation",
					),
					P(
						Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"},
						"Complete technical reference for the GoWebComponents public API. Build reactive web applications with Go's type safety, typed hooks, and WebAssembly delivery.",
					),
				),

				// Table of Contents
				TableOfContents(nil),

				// Core API Sections
				CoreTypesSection(nil),
				ComponentsHooksSection(nil),
				HTMLElementsSection(nil),
				EventHandlingSection(nil),
				StateManagementSection(nil),
				MemoryManagementSection(nil),
				UtilitiesSection(nil),
			),
		),

		// Footer
		FooterSection(nil),

		// Scroll to top button
		ScrollToTopButton(nil),
	)
}

// DocsNavBar provides navigation specific to the documentation page.
// Features back-to-app navigation, documentation title, and direct GitHub access
// with glassmorphism styling consistent with the main navbar.
func DocsNavBar(_ Attrs) *Element {
	return Nav(
		Attrs{
			"class": "fixed top-0 left-0 right-0 z-50 bg-[#0a0a0a]/95 backdrop-blur-xl border-b border-white/10 shadow-lg",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "flex items-center justify-between h-16 md:h-20"},

				// Back Home Link
				func() *Element {
					// Store UseEvent result in variable for proper event handling
					parseNavigateToHome := UseEvent(func(parseEvent MouseEvent) {
						router.Navigate("/")
					})

					return Button(
						Attrs{
							"class":   "group flex items-center space-x-3 px-4 py-2 text-gray-300 hover:text-white transition-all duration-300 font-medium rounded-xl hover:bg-white/10 cursor-pointer",
							"onclick": parseNavigateToHome,
						},
						Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:scale-110"}, "←"),
						Span(Attrs{"class": "font-semibold transition-transform duration-300 group-hover:translate-x-0.5"}, "Back to App"),
					)
				}(),

				// Title
				Div(
					Attrs{"class": "flex items-center"},
					H1(
						Attrs{"class": "text-xl md:text-2xl font-bold text-white"},
						"📚 Documentation",
					),
				),

				// GitHub button
				A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "group relative overflow-hidden px-5 py-2.5 bg-white/10 border border-white/10 text-white rounded-xl hover:bg-white/20 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 cursor-pointer",
					},
					Div(
						Attrs{"class": "absolute inset-0 bg-gradient-to-r from-gray-800 to-gray-900 opacity-0 group-hover:opacity-100 transition-opacity duration-300"},
					),
					Div(
						Attrs{"class": "relative flex items-center space-x-2"},
						Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:rotate-12"}, "🐙"),
						Span(Attrs{"class": "font-semibold text-sm"}, "GitHub"),
					),
				),
			),
		),
	)
}

// TableOfContents creates an interactive navigation grid for documentation sections.
// Features smooth scrolling to sections and visual hierarchy with icons and descriptions
// for easy API discovery and navigation.
func TableOfContents(_ Attrs) *Element {
	return Div(
		Attrs{"class": "bg-white/5 rounded-xl shadow-lg border border-white/10 p-6 mb-12 backdrop-blur-sm"},
		H2(
			Attrs{"class": "text-2xl font-bold text-white mb-6"},
			"📋 Table of Contents",
		),
		Div(
			Attrs{"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"},
			TocLink("🏗️", "Core Types", "core-types", "Element, Fiber, Hooks, and fundamental data structures"),
			TocLink("🎣", "Components & Hooks", "hooks", "UseState, UseEffect, UseMemo, and component lifecycle"),
			TocLink("🌐", "HTML Elements", "html", "All HTML5 elements with props and children support"),
			TocLink("⚡", "Event Handling", "events", "ui.UseEvent and typed browser event helpers"),
			TocLink("💾", "State Management", "state", "Global state, snapshots, and persistence"),
			TocLink("🧠", "Memory Management", "memory", "Pool optimization and cleanup utilities"),
			TocLink("🔧", "Utilities", "utils", "Debug, performance, and helper functions"),
		),
	)
}

// TocLink creates a table of contents link with scroll functionality
func TocLink(parseIcon, parseTitle, parseSectionId, parseDescription string) *Element {
	handleClick := UseEvent(func(parseEvent MouseEvent) {
		parseEvent.PreventDefault()
		// Use the enhanced scroll function from navbar
		// ScrollToSectionSmoothEnhanced is defined in navbar.go in the same package
		ScrollToSectionSmoothEnhanced(parseSectionId)
	})

	return Div(
		Attrs{
			"class":   "block p-4 rounded-lg border border-white/10 hover:border-blue-500/50 hover:bg-white/5 transition-all duration-200 group cursor-pointer",
			"onclick": handleClick,
		},
		Div(
			Attrs{"class": "flex items-start space-x-3"},
			Span(Attrs{"class": "text-2xl group-hover:scale-110 transition-transform duration-200"}, parseIcon),
			Div(nil,
				H3(Attrs{"class": "font-semibold text-white group-hover:text-blue-400"}, parseTitle),
				P(Attrs{"class": "text-sm text-gray-400 mt-1"}, parseDescription),
			),
		),
	)
}

// CoreTypesSection documents the fundamental types and interfaces
func CoreTypesSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "core-types", "class": "mb-16"},
		SectionHeader("🏗️", "Core Types", "Fundamental data structures and interfaces"),

		Div(
			Attrs{"class": "space-y-8"},

			// Element type
			ApiCard("Element", "struct", "Represents a virtual DOM node",
				`type Element struct {
    Type     interface{}            // Component type or HTML tag
    Props    map[string]interface{} // Element properties/attributes
    Children []interface{}          // Child elements or text nodes
}`,
				"The foundation of GoWebComponents' virtual DOM. Every UI element is represented as an Element with a type (either a string for HTML tags or a function for components), props for configuration, and children for nested content."),

			// Fiber type
			ApiCard("Fiber", "struct", "Unit of work in the virtual DOM tree",
				`type Fiber struct {
    parent    *Fiber     // Parent fiber
    alternate *Fiber     // Previous version for diffing
    child     *Fiber     // First child fiber
    sibling   *Fiber     // Next sibling fiber
    hooks     *Hooks     // Component hooks container
    
    typeOf     interface{}            // Component type
    props      map[string]interface{} // Props
    dom        js.Value              // DOM reference
    effectTag  string                // Update type marker
    effects    []func()              // Side effects to run
}`,
				"Internal work unit representing a single component in the reconciliation process. Fibers form a tree structure that enables efficient updates and rendering."),

			// Hooks type
			ApiCard("Hooks", "struct", "Container for component state and effects",
				`type Hooks struct {
    index     int                    // Current hook position
    state     []interface{}         // State values
    deps      [][]interface{}       // Effect dependencies
    memos     []memoizedValue       // Memoized computations
    callOrder []HookCall           // Hook call sequence
}`,
				"Manages all hooks for a component, ensuring consistent ordering and efficient state management across re-renders."),

			// Typed public event aliases
			ApiCard("Typed Events", "aliases", "Public event types used with ui.UseEvent",
				`handleClick := ui.UseEvent(func(event ui.MouseEvent) {
    event.PreventDefault()
})

handleInput := ui.UseEvent(func(event ui.InputEvent) {
    setText(event.GetValue())
})

handleToggle := ui.UseEvent(func(event ui.ChangeEvent) {
    setEnabled(event.IsChecked())
})

handleSubmit := ui.UseEvent(func(event ui.FormEvent) {
    event.PreventDefault()
})`,
				"The public ui package exposes typed event aliases so handlers can declare the event shape they expect instead of working through a single generic wrapper."),

			// FetchState type
			ApiCard("FetchState", "struct", "State of HTTP fetch operations",
				`type FetchState struct {
    Data    interface{} // Response data
    Error   string      // Error message if any
    Loading bool        // Loading indicator
}`,
				"Represents the current state of an asynchronous fetch operation, commonly used with UseFetch when working with raw fetch state."),

			// Attrs type
			ApiCard("Attrs", "type alias", "Element attributes and properties",
				`type Attrs map[string]interface{}
type Attributes map[string]interface{} // Alternative name`,
				"Convenient type alias for passing properties to elements. Supports any HTML attribute, CSS classes, event handlers, and custom properties."),
		),
	)
}

// ComponentsHooksSection documents the hooks API
func ComponentsHooksSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "hooks", "class": "mb-16"},
		SectionHeader("🎣", "Components & Hooks", "State management and component lifecycle"),

		Div(
			Attrs{"class": "space-y-8"},

			// UseState
			ApiCard("UseState", "function", "Manages component state with type safety",
				`func UseState[T any](initialValue T) ui.State[T]

// Example usage:
count := UseState(0)
name := UseState("John")
user := UseState(User{ID: 1, Name: "Alice"})

// Read current state:
current := count.Get()

// Write state:
count.Set(0)

// Functional update:
count.Update(func(prev int) int {
    return prev + 1
})`,
				"Generic state hook that returns a typed state handle with Get, Set, and Update helpers. Updates trigger component re-renders automatically and support any Go type including structs, slices, and primitives."),

			// UseEffect
			ApiCard("UseEffect", "function", "Handles side effects with dependency tracking",
				`func UseEffect(effect func() func(), deps ...interface{})

// Run once on mount:
UseEffect(func() func() {
    fmt.Println("Component mounted")
    return nil
})

// Run when dependencies change:
UseEffect(func() func() {
    fmt.Printf("Count changed to: %d\n", count.Get())
    return nil
}, count.Get())

// Cleanup pattern:
UseEffect(func() func() {
    timer := time.NewTicker(1 * time.Second)
    return func() {
        timer.Stop()
    }
}, true)`,
				"Executes side effects when a component mounts or dependencies change. Effects may return a cleanup function, making the hook suitable for timers, subscriptions, async orchestration, and DOM interactions."),

			// UseMemo
			ApiCard("UseMemo", "function", "Memoizes expensive computations",
				`func UseMemo[T any](compute func() T, deps ...interface{}) T

// Example usage:
expensiveValue := UseMemo(func() ComputedType {
    return performComplexCalculation(data.Get())
}, data.Get())

// Typed result with no assertion required:
fmt.Println(expensiveValue)`,
				"Caches the result of expensive computations and only recalculates when dependencies change. The typed API avoids manual type assertions in callers."),

			// Component lifecycle
			ApiCard("CreateElement", "function", "Creates virtual DOM elements",
				`func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element

// Creating HTML elements:
CreateElement("div", Attrs{"class": "container"}, "Hello World")

// Creating component elements:
CreateElement(MyComponent, Attrs{"name": "value"}, child1, child2)`,
				"Low-level function for creating Element instances. Usually abstracted away by HTML helper functions like Div(), but useful for dynamic element creation."),

			// Render function
			ApiCard("Render", "function", "Renders components to DOM",
				`func Render(root Node, selector string)
func Hydrate(root Node, selector string, options ...HydrationOptions) (SSRBootstrap, error)

// Examples:
selector := exampleboot.GetExampleMountSelector()
ui.Render(ui.CreateElement(App), selector)
_, _ = ui.Hydrate(ui.CreateElement(App), selector)`,
				"Render mounts a client-only tree, while Hydrate resumes markup that was already rendered on the server."),
		),
	)
}

// HTMLElementsSection documents all HTML element functions
func HTMLElementsSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "html", "class": "mb-16"},
		SectionHeader("🌐", "HTML Elements", "Complete HTML5 element library"),

		Div(
			Attrs{"class": "space-y-8"},

			// Document structure
			HtmlElementGroup("Document Structure", []HtmlElementDoc{
				{"Html", "Creates <html> element", `Html(Attrs{"lang": "en"}, Head(nil), Body(nil))`},
				{"Head", "Creates <head> element", `Head(nil, Title(nil, "Page Title"), Meta(Attrs{"charset": "utf-8"}))`},
				{"Body", "Creates <body> element", `Body(Attrs{"class": "app"}, content...)`},
				{"Title", "Creates <title> element", `Title(nil, "My App")`},
				{"Meta", "Creates <meta> element", `Meta(Attrs{"name": "viewport", "content": "width=device-width"})`},
				{"Link", "Creates <link> element", `Link(Attrs{"rel": "stylesheet", "href": "style.css"})`},
				{"Script", "Creates <script> element", `Script(Attrs{"src": "app.js"})`},
			}),

			// Layout elements
			HtmlElementGroup("Layout & Structure", []HtmlElementDoc{
				{"Div", "Creates <div> element", `Div(Attrs{"class": "container"}, children...)`},
				{"Span", "Creates <span> element", `Span(Attrs{"class": "highlight"}, "text")`},
				{"Header", "Creates <header> element", `Header(nil, Nav(nil, "Navigation"))`},
				{"Nav", "Creates <nav> element", `Nav(Attrs{"class": "navbar"}, links...)`},
				{"Main", "Creates <main> element", `Main(nil, content...)`},
				{"Section", "Creates <section> element", `Section(Attrs{"id": "about"}, content...)`},
				{"Article", "Creates <article> element", `Article(nil, "Article content")`},
				{"Aside", "Creates <aside> element", `Aside(nil, "Sidebar content")`},
				{"Footer", "Creates <footer> element", `Footer(nil, "© 2024 My App")`},
			}),

			// Text elements
			HtmlElementGroup("Text & Typography", []HtmlElementDoc{
				{"P", "Creates <p> element", `P(nil, "Paragraph text")`},
				{"H1", "Creates <h1> element", `H1(Attrs{"class": "title"}, "Main Heading")`},
				{"H2", "Creates <h2> element", `H2(nil, "Subheading")`},
				{"H3-H6", "Creates heading elements", `H3(nil, "Section Title")`},
				{"Strong", "Creates <strong> element", `Strong(nil, "Bold text")`},
				{"Em", "Creates <em> element", `Em(nil, "Emphasized text")`},
				{"Code", "Creates <code> element", `Code(nil, "console.log('Hello')")`},
				{"Pre", "Creates <pre> element", `Pre(nil, "Preformatted text")`},
			}),

			// Form elements
			HtmlElementGroup("Form Elements", []HtmlElementDoc{
				{"Form", "Creates <form> element", `Form(Attrs{"onsubmit": handleSubmit}, inputs...)`},
				{"Input", "Creates <input> element", `Input(Attrs{"type": "text", "placeholder": "Enter name"})`},
				{"Button", "Creates <button> element", `Button(Attrs{"onclick": handleClick}, "Click Me")`},
				{"Textarea", "Creates <textarea> element", `Textarea(Attrs{"rows": "4", "oninput": handleInput})`},
				{"Select", "Creates <select> element", `Select(Attrs{"onchange": handleChange}, options...)`},
				{"Option", "Creates <option> element", `Option(Attrs{"value": "1"}, "Option 1")`},
				{"Label", "Creates <label> element", `Label(Attrs{"for": "email"}, "Email Address")`},
			}),

			// List elements
			HtmlElementGroup("Lists", []HtmlElementDoc{
				{"Ul", "Creates <ul> element", `Ul(nil, Li(nil, "Item 1"), Li(nil, "Item 2"))`},
				{"Ol", "Creates <ol> element", `Ol(nil, Li(nil, "First"), Li(nil, "Second"))`},
				{"Li", "Creates <li> element", `Li(Attrs{"class": "item"}, "List item")`},
			}),

			// Table elements
			HtmlElementGroup("Tables", []HtmlElementDoc{
				{"Table", "Creates <table> element", `Table(nil, Thead(nil), Tbody(nil))`},
				{"Thead", "Creates <thead> element", `Thead(nil, Tr(nil, Th(nil, "Header")))`},
				{"Tbody", "Creates <tbody> element", `Tbody(nil, rows...)`},
				{"Tr", "Creates <tr> element", `Tr(nil, Td(nil, "Cell 1"), Td(nil, "Cell 2"))`},
				{"Th", "Creates <th> element", `Th(nil, "Header Cell")`},
				{"Td", "Creates <td> element", `Td(nil, "Data Cell")`},
			}),

			// Media elements
			HtmlElementGroup("Media", []HtmlElementDoc{
				{"Img", "Creates <img> element", `Img(Attrs{"src": "image.jpg", "alt": "Description"})`},
				{"Video", "Creates <video> element", `Video(Attrs{"controls": true, "src": "video.mp4"})`},
				{"Audio", "Creates <audio> element", `Audio(Attrs{"controls": true, "src": "audio.mp3"})`},
			}),
		),
	)
}

// EventHandlingSection documents event handling
func EventHandlingSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "events", "class": "mb-16"},
		SectionHeader("⚡", "Event Handling", "ui.UseEvent and typed browser event helpers"),

		Div(
			Attrs{"class": "space-y-8"},

			// UseEvent
			ApiCard("ui.UseEvent", "function", "Wrap typed event handlers for stable DOM integration",
				`func UseEvent(fn interface{}) ui.Handler

// Example usage:
handleClick := ui.UseEvent(func(event ui.MouseEvent) {
	event.PreventDefault()
	fmt.Println("Button clicked")
})

handleInput := ui.UseEvent(func(event ui.InputEvent) {
	setText(event.GetValue())
})`,
				"UseEvent is the primary public event API. Handlers stay typed at the call site, while the framework handles the DOM wiring and event conversion."),

			// Event examples
			ApiCard("Typed Event Patterns", "patterns", "Common public event-handling shapes",
				`// Form submission
handleSubmit := ui.UseEvent(func(event ui.FormEvent) {
	event.PreventDefault()
	submitForm()
})

// Input changes
handleInput := ui.UseEvent(func(event ui.InputEvent) {
	setText(event.GetValue())
})

// Key press
handleKeyPress := ui.UseEvent(func(event ui.KeyboardEvent) {
	if event.GetKey() == "Enter" {
		submitForm()
	}
})

// Checkbox changes
handleToggle := ui.UseEvent(func(event ui.ChangeEvent) {
	setEnabled(event.IsChecked())
})`,
				"Prefer the typed ui event aliases over direct js.Value access for common interaction code. They keep handlers shorter and easier to audit."),
		),
	)
}

// StateManagementSection documents state management features
func StateManagementSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "state", "class": "mb-16"},
		SectionHeader("💾", "State Management", "Atoms, snapshots, and browser storage"),

		Div(
			Attrs{"class": "space-y-8"},

			// Atoms
			ApiCard("Atoms", "functions", "Application-wide reactive state",
				`func UseAtom[T any](id string, initialValue T) Atom[T]
func (a Atom[T]) Get() T
func (a Atom[T]) Set(value T)
func (a Atom[T]) Update(fn func(T) T)

// Example usage:
theme := state.UseAtom("theme", "dark")
theme.Set("light")

if theme.Get() == "light" {
    fmt.Println("Using the light theme")
}`,
				"Atoms provide shared reactive state keyed by ID, making cross-component coordination explicit and type-safe."),

			// State snapshots
			ApiCard("State Snapshots", "functions", "Capture and restore registered atoms",
				`func GetSnapshot() (Snapshot, error)
func ApplySnapshot(snapshot Snapshot) error
func MarshalSnapshotJSON(snapshot Snapshot) ([]byte, error)
func UnmarshalSnapshotJSON(data []byte) (Snapshot, error)

// Example usage:
snap, _ := state.GetSnapshot()
snapshot := snap.Select("theme", "user")
encoded, err := state.MarshalSnapshotJSON(snapshot)
if err == nil {
    restored, _ := state.UnmarshalSnapshotJSON(encoded)
    _ = state.ApplySnapshot(restored)
}`,
				"Snapshots are the durable boundary for state transfer, debugging, and persistence. Select only the atoms you need before exporting."),

			// Snapshot storage
			ApiCard("Snapshot Storage", "functions", "Persist snapshots in browser storage",
				`type StorageArea string

const (
    LocalStorage StorageArea = "localStorage"
    SessionStorage StorageArea = "sessionStorage"
)

func SaveSnapshot(key string, snapshot Snapshot, area StorageArea) error
func LoadSnapshot(key string, area StorageArea) (Snapshot, bool, error)
func RestoreSnapshot(key string, area StorageArea) (bool, error)

// Example usage:
snap, _ := state.GetSnapshot()
snapshot := snap.Select("theme")
_ = state.SaveSnapshot("app-state", snapshot, state.LocalStorage)
_, _ = state.RestoreSnapshot("app-state", state.LocalStorage)`,
				"Storage helpers serialize snapshots as JSON so a small set of shared atoms can survive refreshes or be restored on demand."),

			// Derived state
			ApiCard("Derived State", "functions", "Compute read-only values from other state",
				`func UseMemo[T any](compute func() T, deps ...any) T   // preferred (ui package)
func UseDerived[T any](id string, compute func() T, deps ...string) Derived[T]
func (d Derived[T]) Get() T
// state.UseComputed is the deprecated predecessor of ui.UseMemo.

// Example usage:
total := state.UseAtom("total", 24)
taxed := ui.UseMemo(func() string {
    return fmt.Sprintf("$%0.2f", float64(total.Get())*1.2)
}, total.Get())

status := state.UseDerived("cart-status", func() string {
    if total.Get() == 0 {
        return "Empty"
    }
    return "Ready"
}, "total")`,
				"ui.UseMemo computes a render-local derived value (preferred; state.UseComputed is its deprecated predecessor), while UseDerived registers a shared read-only atom keyed by ID."),
		),
	)
}

// MemoryManagementSection documents memory optimization features
func MemoryManagementSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "memory", "class": "mb-16"},
		SectionHeader("🧠", "Memory Management", "Pool optimization and performance tuning"),

		Div(
			Attrs{"class": "space-y-8"},

			// Pool management
			ApiCard("Pool Management", "functions", "Object pool optimization",
				`func GetPoolSizes() map[string]int32
func SetMemoryLimits(maxCalls, maxPool int)
func ConfigureMemoryLimits(maxCalls, maxPool int)
func EnableAdaptivePoolSizing(enabled bool)

// Example usage:
// Configure memory limits
ConfigureMemoryLimits(10000, 1000)
EnableAdaptivePoolSizing(true)

// Monitor pool sizes
sizes := GetPoolSizes()
fmt.Printf("Fiber pool: %d\n", sizes["fiber"])`,
				"Configure and monitor object pools for optimal memory usage and performance."),

			// Memory monitoring
			ApiCard("Memory Monitoring", "functions", "Performance tracking and diagnostics",
				`func GetPoolUtilizationStats() map[string]interface{}
func ResetPoolUtilizationStats()
func ConfigureMemStatsSampleRate(rate int64)
func GetMemStatsSampleRate() int64

// Example monitoring:
stats := GetPoolUtilizationStats()
fmt.Printf("Pool hit rate: %.2f%%\n", stats["hitRate"])
fmt.Printf("Total allocations: %d\n", stats["totalAllocations"])`,
				"Monitor memory usage patterns and pool efficiency for performance optimization."),

			// Snapshot boundaries
			ApiCard("Snapshot Boundaries", "functions", "Choose what state survives beyond the current render",
				`func (s Snapshot) Select(keys ...string) Snapshot
func MarshalSnapshotJSON(snapshot Snapshot) ([]byte, error)
func UnmarshalSnapshotJSON(data []byte) (Snapshot, error)

// Example usage:
snap, _ := state.GetSnapshot()
selected := snap.Select("theme", "locale")
payload, _ := state.MarshalSnapshotJSON(selected)
fmt.Println(string(payload))`,
				"Use selective snapshots when you need persistence or diagnostics without serializing every atom in the runtime."),
		),
	)
}

// UtilitiesSection documents utility functions
func UtilitiesSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "utils", "class": "mb-16"},
		SectionHeader("🔧", "Utilities", "Debug tools, performance helpers, and configuration"),

		Div(
			Attrs{"class": "space-y-8"},

			// Debug functions
			ApiCard("Debug Functions", "functions", "Development and debugging tools",
				`func EnableDebug()\nfunc DisableDebug()
func ConfigureDebugNamespace(namespace string, enabled bool)
func EnableAllDebug()
func DisableAllDebug()
func GetDebugStatus() map[string]bool

// Example usage:
EnableDebug()                          // Enable all debug output
ConfigureDebugNamespace("HOOKS", true)        // Enable only hooks debugging
ConfigureDebugNamespace("FIBER", false)       // Disable fiber debugging

// Check current status
status := GetDebugStatus()
fmt.Printf("Debug enabled: %v\n", status)`,
				"Control debug output and logging for different parts of the framework during development."),

			// Performance monitoring
			ApiCard("Performance Monitoring", "functions", "Performance tracking and optimization",
				`func EnableGoroutineMonitoring()
func DisableGoroutineMonitoring()
func ConfigureGoroutineThreshold(threshold int)
func GetGoroutineStats() map[string]int64
func ResetGoroutineBaseline()

// Example usage:
EnableGoroutineMonitoring()
ConfigureGoroutineThreshold(100)  // Alert if >100 goroutines

stats := GetGoroutineStats()
fmt.Printf("Active goroutines: %d\n", stats["current"])`,
				"Monitor goroutine usage and detect potential memory leaks in long-running applications."),

			// UI queue management
			ApiCard("UI Queue Management", "functions", "Asynchronous update management",
				`func SetUIQueueBufferSize(size int)
func SetUIQueueLimits(minSize, maxSize int)
func GetUIQueueStats() map[string]int64
func ResetUIQueueStats()
func SetUIQueueAutoOptimization(enabled bool)

// Example usage:
SetUIQueueBufferSize(2048)      // Set buffer size
SetUIQueueLimits(256, 4096)     // Set min/max limits
SetUIQueueAutoOptimization(true) // Enable auto-sizing

// Monitor queue performance
stats := GetUIQueueStats()
fmt.Printf("Queue overflows: %d\n", stats["overflows"])`,
				"Configure and monitor the UI update queue for optimal performance in high-throughput applications."),

			// Fetch utilities
			ApiCard("Fetch Utilities", "hook", "HTTP request management",
				`func UseFetch(url string, options ...fetch.Options) fetch.Resource
func UseResource[T any](loader func(context.Context) (T, error), deps ...interface{}) fetch.AsyncResource[T]
func Fetch(url string, options fetch.Options) <-chan fetch.Result

// Example usage:
resource := UseFetch("https://api.example.com/data")
state := resource.Get()

if state.Loading {
    // Show loading indicator
} else if state.Error != "" {
    // Show error message
} else if state.Data != nil {
    // Use the data
    if data, ok := state.Data.(map[string]interface{}); ok {
        // Process data
    }
}

// Refetch data
resource.Refetch()`,
				"UseFetch exposes raw fetch state, while UseResource adds typed values, cancellation, and dependency-driven reloads for non-trivial async loading."),
		),
	)
}

// Helper components for documentation structure

// SectionHeader creates a consistent section header
func SectionHeader(parseIcon, parseTitle, parseDescription string) *Element {
	return Div(
		Attrs{"class": "mb-8"},
		H2(
			Attrs{"class": "text-3xl font-bold text-white mb-2 flex items-center"},
			Span(Attrs{"class": "mr-3"}, parseIcon),
			parseTitle,
		),
		P(Attrs{"class": "text-lg text-gray-400"}, parseDescription),
	)
}

// ApiCard creates a documentation card for API items
func ApiCard(parseName, parseApiType, parseDescription, parseCode, parseDetails string) *Element {
	return Div(
		Attrs{"class": "bg-white/5 rounded-xl shadow-lg border border-white/10 overflow-hidden backdrop-blur-sm"},

		// Header
		Div(
			Attrs{"class": "bg-white/5 px-6 py-4 border-b border-white/10"},
			Div(
				Attrs{"class": "flex items-center justify-between"},
				Div(nil,
					H3(Attrs{"class": "text-xl font-bold text-white"}, parseName),
					Span(Attrs{"class": "inline-block px-2 py-1 bg-blue-900/30 text-blue-300 border border-blue-500/30 text-xs font-medium rounded-full mt-1"}, parseApiType),
				),
			),
			P(Attrs{"class": "text-gray-400 mt-2"}, parseDescription),
		),

		// Code example
		Div(
			Attrs{"class": "bg-[#0a0a0a] text-gray-300 p-6 border-y border-white/10"},
			Pre(
				Attrs{"class": "text-sm overflow-x-auto"},
				Code(nil, parseCode),
			),
		),

		// Details
		Div(
			Attrs{"class": "px-6 py-4"},
			P(Attrs{"class": "text-gray-300 leading-relaxed"}, parseDetails),
		),
	)
}

// HtmlElementGroup creates a group of HTML element documentation
func HtmlElementGroup(parseTitle string, parseElements []HtmlElementDoc) *Element {
	parseElementCards := make([]interface{}, len(parseElements))
	for parseI, parseElem := range parseElements {
		parseElementCards[parseI] = HtmlElementCard(parseElem)
	}

	return Div(
		Attrs{"class": "mb-8"},
		H3(Attrs{"class": "text-xl font-semibold text-white mb-4"}, parseTitle),
		Div(
			Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-4"},
			parseElementCards...,
		),
	)
}

// HtmlElementDoc represents documentation for an HTML element
type HtmlElementDoc struct {
	Name        string
	Description string
	Example     string
}

// HtmlElementCard creates a card for an HTML element
func HtmlElementCard(parseElem HtmlElementDoc) *Element {
	return Div(
		Attrs{"class": "bg-white/5 border border-white/10 rounded-lg p-4 hover:bg-white/10 transition-colors"},
		H4(Attrs{"class": "font-semibold text-white mb-2"}, parseElem.Name),
		P(Attrs{"class": "text-sm text-gray-400 mb-3"}, parseElem.Description),
		Div(
			Attrs{"class": "bg-[#0a0a0a] text-gray-300 p-3 rounded text-xs overflow-x-auto border border-white/5"},
			Code(nil, parseElem.Example),
		),
	)
}
