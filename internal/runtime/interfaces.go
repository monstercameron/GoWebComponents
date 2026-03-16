package runtime

// DOMNode is an opaque reference to a platform-specific DOM node.
type DOMNode interface {
	IsNull() bool
	// TODO: clarify how nil vs IsNull interact for adapters to avoid mixed nil/IsNull checks
	Equals(other DOMNode) bool
}

// DOMAdapter abstracts all DOM operations used by the runtime.
type DOMAdapter interface {
	// Element creation and manipulation
	CreateElement(tag string) DOMNode
	CreateTextNode(text string) DOMNode
	SetAttribute(node DOMNode, name, value string)
	RemoveAttribute(node DOMNode, name string)
	SetProperty(node DOMNode, name string, value interface{})
	GetProperty(node DOMNode, name string) interface{}

	// Tree manipulation
	AppendChild(parent, child DOMNode)
	RemoveChild(parent, child DOMNode)
	InsertBefore(parent, newNode, referenceNode DOMNode)
	ReplaceChild(parent, newNode, oldNode DOMNode)

	// Queries
	GetParent(node DOMNode) DOMNode
	GetChildren(node DOMNode) []DOMNode
	GetFirstChild(node DOMNode) DOMNode
	GetNextSibling(node DOMNode) DOMNode

	// Text nodes
	SetTextContent(node DOMNode, text string)

	// Styling
	SetStyle(node DOMNode, property, value string)
	SetStyles(node DOMNode, styles map[string]string)
	SetInnerHTML(node DOMNode, html string)

	// Function wrapping
	WrapFunction(fn interface{}) interface{}
}

// EventHandler is an opaque reference to a platform-specific event handler.
type EventHandler interface {
	Release()
}

// Event abstracts platform-specific events.
type Event interface {
	PreventDefault()
	StopPropagation()
	GetValue() string
	GetTarget() DOMNode
	GetKeyCode() int
	GetKey() string
	IsChecked() bool
}

// EventAdapter abstracts event handling.
type EventAdapter interface {
	AddEventListener(node DOMNode, eventType string, handler EventHandler)
	RemoveEventListener(node DOMNode, eventType string, handler EventHandler)
	CreateEventHandler(fn func(Event)) EventHandler
	ReleaseEventHandler(handler EventHandler)
}

// Deadline provides timing information for idle callbacks.
type Deadline interface {
	TimeRemaining() float64
	DidTimeout() bool
}

// Scheduler abstracts work scheduling.
type Scheduler interface {
	RequestIdleCallback(callback func(deadline Deadline))
	SetTimeout(callback func(), delay int)
}

// BrowserState abstracts browser state APIs.
type BrowserState interface {
	// History
	PushState(state interface{}, title, url string)
	ReplaceState(state interface{}, title, url string)
	GetCurrentPath() string
	OnPopState(callback func(path string))

	// Storage
	SetItem(key, value string) error
	GetItem(key string) (string, bool)
	RemoveItem(key string)

	// Location
	GetHash() string
	SetHash(hash string)
	Reload()
}
