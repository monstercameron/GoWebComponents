//go:build js && wasm
// +build js,wasm

package website
import (
	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/router"
)

// DocsPage renders complete API documentation for the GoWebComponents framework.
// Features organized sections for types, hooks, HTML elements, events, and utilities
// with interactive examples and detailed explanations for each API.
func DocsPage(_ Attrs) *Element {
	return dom.Div(
		Attrs{"class": "min-h-screen bg-gradient-to-br from-gray-50 to-blue-50"},

		// Documentation-specific navigation bar
		DocsNavBar(nil),

		// Documentation content
		dom.Div(
			Attrs{"class": "pt-32 pb-20"},
			dom.Div(
				Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},

				// Header
				dom.Div(
					Attrs{"class": "text-center mb-16"},
					dom.H1(
						Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
						"📚 GoWebComponents API Documentation",
					),
					dom.P(
						Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
						"Complete technical reference for the GoWebComponents fiber library. Build reactive web applications with Go's type safety and performance.",
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
	return dom.Nav(
		Attrs{
			"class": "fixed top-0 left-0 right-0 z-50 bg-white/95 backdrop-blur-xl border-b border-gray-200/50 shadow-lg",
		},
		dom.Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			dom.Div(
				Attrs{"class": "flex items-center justify-between h-16 md:h-20"},

				// Back Home Link
				func() *Element {
					// Store GoUseFunc result in variable for proper event handling
					navigateToHome := hooks.GoUseFunc(func(event dom.GoEvent) {
						router.Navigate("/")
					})

					return dom.Button(
						Attrs{
							"class":   "group flex items-center space-x-3 px-4 py-2 text-gray-700 hover:text-indigo-600 transition-all duration-300 font-medium rounded-xl hover:bg-gradient-to-r hover:from-indigo-50 hover:to-purple-50 cursor-pointer",
							"onclick": navigateToHome,
						},
						dom.Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:scale-110"}, "←"),
						dom.Span(Attrs{"class": "font-semibold transition-transform duration-300 group-hover:translate-x-0.5"}, "Back to App"),
					)
				}(),

				// Title
				dom.Div(
					Attrs{"class": "flex items-center"},
					dom.H1(
						Attrs{"class": "text-xl md:text-2xl font-bold text-gray-900"},
						"📚 Documentation",
					),
				),

				// GitHub button
				dom.A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "group relative overflow-hidden px-5 py-2.5 bg-gray-900 text-white rounded-xl hover:bg-gray-800 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 cursor-pointer",
					},
					dom.Div(
						Attrs{"class": "absolute inset-0 bg-gradient-to-r from-gray-800 to-gray-900 opacity-0 group-hover:opacity-100 transition-opacity duration-300"},
					),
					dom.Div(
						Attrs{"class": "relative flex items-center space-x-2"},
						dom.Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:rotate-12"}, "🐙"),
						dom.Span(Attrs{"class": "font-semibold text-sm"}, "GitHub"),
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
	return dom.Div(
		Attrs{"class": "bg-white rounded-xl shadow-lg border border-gray-200 p-6 mb-12"},
		dom.H2(
			Attrs{"class": "text-2xl font-bold text-gray-900 mb-6"},
			"📋 Table of Contents",
		),
		dom.Div(
			Attrs{"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"},
			TocLink("🏗️", "Core Types", "core-types", "Element, Fiber, Hooks, and fundamental data structures"),
			TocLink("🎣", "Components & Hooks", "hooks", "hooks.UseState, hooks.UseEffect, hooks.UseMemo, and component lifecycle"),
			TocLink("🌐", "HTML Elements", "html", "All HTML5 elements with props and children support"),
			TocLink("⚡", "Event Handling", "events", "GoEvent wrapper and event management"),
			TocLink("💾", "State Management", "state", "Global state, snapshots, and persistence"),
			TocLink("🧠", "Memory Management", "memory", "Pool optimization and cleanup utilities"),
			TocLink("🔧", "Utilities", "utils", "Debug, performance, and helper functions"),
		),
	)
}

// TocLink creates a table of contents link with scroll functionality
func TocLink(icon, title, sectionId, description string) *Element {
	handleClick := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		// Use the enhanced scroll function from navbar
		// ScrollToSectionSmoothEnhanced is defined in navbar.go in the same package
		ScrollToSectionSmoothEnhanced(sectionId)
	})

	return dom.Div(
		Attrs{
			"class":   "block p-4 rounded-lg border border-gray-200 hover:border-indigo-300 hover:bg-indigo-50 transition-all duration-200 group cursor-pointer",
			"onclick": handleClick,
		},
		dom.Div(
			Attrs{"class": "flex items-start space-x-3"},
			dom.Span(Attrs{"class": "text-2xl group-hover:scale-110 transition-transform duration-200"}, icon),
			dom.Div(nil,
				dom.H3(Attrs{"class": "font-semibold text-gray-900 group-hover:text-indigo-600"}, title),
				dom.P(Attrs{"class": "text-sm text-gray-600 mt-1"}, description),
			),
		),
	)
}

// CoreTypesSection documents the fundamental types and interfaces
func CoreTypesSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{"id": "core-types", "class": "mb-16"},
		SectionHeader("🏗️", "Core Types", "Fundamental data structures and interfaces"),

		dom.Div(
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

			// GoEvent type
			ApiCard("GoEvent", "struct", "Go wrapper for JavaScript events",
				`type GoEvent struct {
    jsEvent js.Value  // Underlying JavaScript event
}

// Methods:
func (e GoEvent) Target() js.Value
func (e GoEvent) PreventDefault()
func (e GoEvent) StopPropagation()`,
				"Provides a Go-friendly interface for handling JavaScript events with common methods and type safety."),

			// FetchState type
			ApiCard("FetchState", "struct", "State of HTTP fetch operations",
				`type FetchState struct {
    Data    interface{} // Response data
    Error   string      // Error message if any
    Loading bool        // Loading indicator
}`,
				"Represents the current state of an asynchronous fetch operation, commonly used with GoUseFetch hook."),

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
	return dom.Section(
		Attrs{"id": "hooks", "class": "mb-16"},
		SectionHeader("🎣", "Components & Hooks", "State management and component lifecycle"),

		dom.Div(
			Attrs{"class": "space-y-8"},

			// GoUseState
			ApiCard("GoUseState", "function", "Manages component state with type safety",
				`func GoUseState[T any](initialValue T) (func() T, func(T))

// Example usage:
count, setCount := GoUseState(0)
name, setName := GoUseState("John")
user, setUser := GoUseState(User{ID: 1, Name: "Alice"})

// In component:
if count() > 5 {
    setCount(0) // Reset counter
}`,
				"Generic state hook that provides type-safe state management. Returns a getter function and a setter function. Updates trigger component re-renders automatically. Supports any Go type including structs, slices, and primitives."),

			// GoUseEffect
			ApiCard("GoUseEffect", "function", "Handles side effects with dependency tracking",
				`func GoUseEffect(effect func(), deps ...interface{})

// Run once on mount:
GoUseEffect(func() {
    fmt.Println("Component mounted")
})

// Run when dependencies change:
GoUseEffect(func() {
    fmt.Printf("Count changed to: %d\n", count())
}, count())

// Cleanup pattern:
GoUseEffect(func() {
    timer := time.NewTicker(1 * time.Second)
    // Cleanup would be handled by framework
})`,
				"Executes side effects when component mounts or dependencies change. Dependencies are compared for equality to determine when to re-run. Useful for API calls, timers, subscriptions, and DOM manipulations."),

			// GoUseMemo
			ApiCard("GoUseMemo", "function", "Memoizes expensive computations",
				`func GoUseMemo(compute func() interface{}, deps ...interface{}) interface{}

// Example usage:
expensiveValue := GoUseMemo(func() interface{} {
    // Expensive computation
    result := performComplexCalculation(data())
    return result
}, data())

// Type assertion for specific types:
if computed, ok := expensiveValue.(ComputedType); ok {
    // Use typed result
}`,
				"Caches the result of expensive computations and only recalculates when dependencies change. Helps optimize performance by avoiding redundant calculations during re-renders."),

			// Component lifecycle
			ApiCard("CreateElement", "function", "Creates virtual DOM elements",
				`func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element

// Creating HTML elements:
CreateElement("div", Attrs{"class": "container"}, "Hello World")

// Creating component elements:
CreateElement(MyComponent, Attrs{"name": "value"}, child1, child2)`,
				"Low-level function for creating Element instances. Usually abstracted away by HTML helper functions like dom.Div(), but useful for dynamic element creation."),

			// Render function
			ApiCard("Render", "function", "Renders components to DOM",
				`func Render(element *Element, container js.Value)
func RenderTo(selector string, component interface{})

// Examples:
Render(App(nil), js.Global().Get("document").Call("getElementById", "root"))
RenderTo("#app", MyComponent)`,
				"Entry point for rendering GoWebComponents applications. Mounts the component tree to a DOM container and begins the reconciliation process."),
		),
	)
}

// HTMLElementsSection documents all HTML element functions
func HTMLElementsSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{"id": "html", "class": "mb-16"},
		SectionHeader("🌐", "HTML Elements", "Complete HTML5 element library"),

		dom.Div(
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
				{"Div", "Creates <div> element", `dom.Div(Attrs{"class": "container"}, children...)`},
				{"Span", "Creates <span> element", `dom.Span(Attrs{"class": "highlight"}, "text")`},
				{"Header", "Creates <header> element", `Header(nil, dom.Nav(nil, "Navigation"))`},
				{"Nav", "Creates <nav> element", `dom.Nav(Attrs{"class": "navbar"}, links...)`},
				{"Main", "Creates <main> element", `Main(nil, content...)`},
				{"Section", "Creates <section> element", `dom.Section(Attrs{"id": "about"}, content...)`},
				{"Article", "Creates <article> element", `dom.Article(nil, "Article content")`},
				{"Aside", "Creates <aside> element", `Aside(nil, "Sidebar content")`},
				{"Footer", "Creates <footer> element", `Footer(nil, "© 2024 My App")`},
			}),

			// Text elements
			HtmlElementGroup("Text & Typography", []HtmlElementDoc{
				{"P", "Creates <p> element", `dom.P(nil, "Paragraph text")`},
				{"H1", "Creates <h1> element", `dom.H1(Attrs{"class": "title"}, "Main Heading")`},
				{"H2", "Creates <h2> element", `dom.H2(nil, "Subheading")`},
				{"H3-H6", "Creates heading elements", `dom.H3(nil, "Section Title")`},
				{"Strong", "Creates <strong> element", `dom.Strong(nil, "Bold text")`},
				{"Em", "Creates <em> element", `Em(nil, "Emphasized text")`},
				{"Code", "Creates <code> element", `dom.Code(nil, "console.log('Hello')")`},
				{"Pre", "Creates <pre> element", `dom.Pre(nil, "Preformatted text")`},
			}),

			// Form elements
			HtmlElementGroup("Form Elements", []HtmlElementDoc{
				{"Form", "Creates <form> element", `Form(Attrs{"onsubmit": handleSubmit}, inputs...)`},
				{"Input", "Creates <input> element", `Input(Attrs{"type": "text", "placeholder": "Enter name"})`},
				{"Button", "Creates <button> element", `dom.Button(Attrs{"onclick": handleClick}, "Click Me")`},
				{"Textarea", "Creates <textarea> element", `Textarea(Attrs{"rows": "4", "oninput": handleInput})`},
				{"Select", "Creates <select> element", `Select(Attrs{"onchange": handleChange}, options...)`},
				{"Option", "Creates <option> element", `Option(Attrs{"value": "1"}, "Option 1")`},
				{"Label", "Creates <label> element", `Label(Attrs{"for": "email"}, "Email Address")`},
			}),

			// List elements
			HtmlElementGroup("Lists", []HtmlElementDoc{
				{"Ul", "Creates <ul> element", `dom.Ul(nil, dom.Li(nil, "Item 1"), dom.Li(nil, "Item 2"))`},
				{"Ol", "Creates <ol> element", `Ol(nil, dom.Li(nil, "First"), dom.Li(nil, "Second"))`},
				{"Li", "Creates <li> element", `dom.Li(Attrs{"class": "item"}, "List item")`},
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
	return dom.Section(
		Attrs{"id": "events", "class": "mb-16"},
		SectionHeader("⚡", "Event Handling", "JavaScript event integration and management"),

		dom.Div(
			Attrs{"class": "space-y-8"},

			// GoEvent
			ApiCard("GoEvent", "struct", "Go wrapper for JavaScript events",
				`type GoEvent struct {
    jsEvent js.Value
}

// Common methods:
func (e GoEvent) Target() js.Value          // Event target element
func (e GoEvent) PreventDefault()           // Prevent default behavior
func (e GoEvent) StopPropagation()          // Stop event bubbling
func (e GoEvent) CurrentTarget() js.Value   // Current event target
func (e GoEvent) Type() string              // Event type (click, change, etc.)`,
				"Provides a Go-friendly interface for JavaScript events with convenient methods for common operations."),

			// GoUseFunc
			ApiCard("GoUseFunc", "function", "Creates event handler functions",
				`func GoUseFunc(fn func(GoEvent)) interface{}

// Example usage:
handleClick := GoUseFunc(func(event GoEvent) {
    event.PreventDefault()
    fmt.Println("Button clicked!")
    // Update state, call APIs, etc.
})

dom.Button(Attrs{"onclick": handleClick}, "Click Me")`,
				"Wraps Go functions to be compatible with JavaScript event handlers. Automatically converts JavaScript events to GoEvent instances."),

			// Event examples
			ApiCard("Event Examples", "patterns", "Common event handling patterns",
				`// Form submission
handleSubmit := GoUseFunc(func(event GoEvent) {
    event.PreventDefault()
    formData := extractFormData(event.Target())
    submitForm(formData)
})

// Input changes
handleInput := GoUseFunc(func(event GoEvent) {
    value := event.Target().Get("value").String()
    setText(value)
})

// Key press
handleKeyPress := GoUseFunc(func(event GoEvent) {
    if event.jsEvent.Get("key").String() == "Enter" {
        handleSubmit(event)
    }
})

// Mouse events
handleMouseOver := GoUseFunc(func(event GoEvent) {
    setHovered(true)
})`,
				"Practical examples of handling different types of events in GoWebComponents applications."),
		),
	)
}

// StateManagementSection documents state management features
func StateManagementSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{"id": "state", "class": "mb-16"},
		SectionHeader("💾", "State Management", "Global state, persistence, and hot reload"),

		dom.Div(
			Attrs{"class": "space-y-8"},

			// Global state
			ApiCard("Global State", "functions", "Application-wide state management",
				`func SetGlobalState(key string, value interface{})
func GetGlobalState(key string) (interface{}, bool)
func ClearGlobalState()

// Example usage:
SetGlobalState("user", User{ID: 1, Name: "Alice"})
SetGlobalState("theme", "dark")

if user, exists := GetGlobalState("user"); exists {
    if u, ok := user.(User); ok {
        fmt.Printf("Current user: %s\n", u.Name)
    }
}`,
				"Manage state that needs to be shared across multiple components without prop drilling."),

			// State snapshots
			ApiCard("State Snapshots", "functions", "State persistence and hot reload",
				`func ExportAppState() js.Value
func ImportAppState(jsState js.Value)
func ExportStateSnapshot() ([]byte, error)
func ImportStateSnapshot(data []byte) error
func SaveStateToFile(filename string) error

// Example usage:
// Export current state
snapshot, err := ExportStateSnapshot()
if err == nil {
    // Save to storage or file
    SaveStateToFile("app-state.json")
}

// Restore state later
data := loadStateFromStorage()
ImportStateSnapshot(data)`,
				"Export and import application state for persistence, debugging, and hot reload during development."),

			// Hot reload
			ApiCard("Hot Reload", "functions", "Development-time state preservation",
				`func HotReloadWasm()
func EnableHotReload(enabled bool)
func IsHotReloadEnabled() bool
func RestoreStateFromStorage()

// Development workflow:
EnableHotReload(true)  // Enable hot reload
// State is automatically preserved during code changes
// Call HotReloadWasm() to reload with preserved state`,
				"Preserve component state during development for faster iteration cycles."),

			// State cleanup
			ApiCard("State Cleanup", "functions", "Memory management and cleanup",
				`func CleanupDOM()
func CleanupMemory()

// Use during navigation or component unmounting:
CleanupDOM()     // Clean up DOM references
CleanupMemory()  // Release memory pools`,
				"Clean up resources and memory when components are unmounted or during navigation."),
		),
	)
}

// MemoryManagementSection documents memory optimization features
func MemoryManagementSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{"id": "memory", "class": "mb-16"},
		SectionHeader("🧠", "Memory Management", "Pool optimization and performance tuning"),

		dom.Div(
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
func SetMemStatsSampleRate(rate int64)
func GetMemStatsSampleRate() int64

// Example monitoring:
stats := GetPoolUtilizationStats()
fmt.Printf("Pool hit rate: %.2f%%\n", stats["hitRate"])
fmt.Printf("Total allocations: %d\n", stats["totalAllocations"])`,
				"Monitor memory usage patterns and pool efficiency for performance optimization."),

			// Cleanup functions
			ApiCard("Cleanup Functions", "functions", "Memory and resource cleanup",
				`func CleanupMemory()
func CleanupDOM()
func forceGarbageCollection()  // Internal use

// Usage during cleanup:
CleanupMemory()  // Release all pools
CleanupDOM()     // Clean DOM references`,
				"Force cleanup of memory pools and DOM references when needed."),
		),
	)
}

// UtilitiesSection documents utility functions
func UtilitiesSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{"id": "utils", "class": "mb-16"},
		SectionHeader("🔧", "Utilities", "Debug tools, performance helpers, and configuration"),

		dom.Div(
			Attrs{"class": "space-y-8"},

			// Debug functions
			ApiCard("Debug Functions", "functions", "Development and debugging tools",
				`func SetDebug(enabled bool)
func SetDebugNamespace(namespace string, enabled bool)
func EnableAllDebug()
func DisableAllDebug()
func GetDebugStatus() map[string]bool

// Example usage:
SetDebug(true)                          // Enable all debug output
SetDebugNamespace("HOOKS", true)        // Enable only hooks debugging
SetDebugNamespace("FIBER", false)       // Disable fiber debugging

// Check current status
status := GetDebugStatus()
fmt.Printf("Debug enabled: %v\n", status)`,
				"Control debug output and logging for different parts of the framework during development."),

			// Performance monitoring
			ApiCard("Performance Monitoring", "functions", "Performance tracking and optimization",
				`func EnableGoroutineMonitoring()
func DisableGoroutineMonitoring()
func SetGoroutineThreshold(threshold int)
func GetGoroutineStats() map[string]int64
func ResetGoroutineBaseline()

// Example usage:
EnableGoroutineMonitoring()
SetGoroutineThreshold(100)  // Alert if >100 goroutines

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
				`func GoUseFetch(url string, options ...FetchOptions) (func() FetchState, func())

// Example usage:
getFetchState, refetch := GoUseFetch("https://api.example.com/data")
state := getFetchState()

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
refetch()`,
				"Built-in hook for making HTTP requests with automatic loading states and error handling."),
		),
	)
}

// Helper components for documentation structure

// SectionHeader creates a consistent section header
func SectionHeader(icon, title, description string) *Element {
	return dom.Div(
		Attrs{"class": "mb-8"},
		dom.H2(
			Attrs{"class": "text-3xl font-bold text-gray-900 mb-2 flex items-center"},
			dom.Span(Attrs{"class": "mr-3"}, icon),
			title,
		),
		dom.P(Attrs{"class": "text-lg text-gray-600"}, description),
	)
}

// ApiCard creates a documentation card for API items
func ApiCard(name, apiType, description, code, details string) *Element {
	return dom.Div(
		Attrs{"class": "bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden"},

		// Header
		dom.Div(
			Attrs{"class": "bg-gradient-to-r from-indigo-50 to-purple-50 px-6 py-4 border-b border-gray-200"},
			dom.Div(
				Attrs{"class": "flex items-center justify-between"},
				dom.Div(nil,
					dom.H3(Attrs{"class": "text-xl font-bold text-gray-900"}, name),
					dom.Span(Attrs{"class": "inline-block px-2 py-1 bg-indigo-100 text-indigo-700 text-xs font-medium rounded-full mt-1"}, apiType),
				),
			),
			dom.P(Attrs{"class": "text-gray-600 mt-2"}, description),
		),

		// Code example
		dom.Div(
			Attrs{"class": "bg-gray-900 text-gray-100 p-6"},
			dom.Pre(
				Attrs{"class": "text-sm overflow-x-auto"},
				dom.Code(nil, code),
			),
		),

		// Details
		dom.Div(
			Attrs{"class": "px-6 py-4"},
			dom.P(Attrs{"class": "text-gray-700 leading-relaxed"}, details),
		),
	)
}

// HtmlElementGroup creates a group of HTML element documentation
func HtmlElementGroup(title string, elements []HtmlElementDoc) *Element {
	elementCards := make([]interface{}, len(elements))
	for i, elem := range elements {
		elementCards[i] = HtmlElementCard(elem)
	}

	return dom.Div(
		Attrs{"class": "mb-8"},
		dom.H3(Attrs{"class": "text-xl font-semibold text-gray-900 mb-4"}, title),
		dom.Div(
			Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-4"},
			elementCards...,
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
func HtmlElementCard(elem HtmlElementDoc) *Element {
	return dom.Div(
		Attrs{"class": "bg-white border border-gray-200 rounded-lg p-4"},
		dom.H4(Attrs{"class": "font-semibold text-gray-900 mb-2"}, elem.Name),
		dom.P(Attrs{"class": "text-sm text-gray-600 mb-3"}, elem.Description),
		dom.Div(
			Attrs{"class": "bg-gray-900 text-gray-100 p-3 rounded text-xs overflow-x-auto"},
			dom.Code(nil, elem.Example),
		),
	)
}





