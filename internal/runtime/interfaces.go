package runtime

// DOMNode is an opaque reference to a platform-specific DOM node.
//
// Contract:
// - implementations must treat a nil receiver as null and return true from IsNull
// - nil interface values and explicit null-node wrappers are equivalent absence states
// - callers should prefer IsDOMNodeNull and IsSameDOMNode instead of open-coding mixed nil and IsNull checks
type DOMNode interface {
	IsNull() bool
	Equals(other DOMNode) bool
}

// IsDOMNodeNull reports whether a DOMNode is absent, whether that absence is represented as a nil interface or one explicit null-node wrapper.
func IsDOMNodeNull(parseNode DOMNode) bool {
	if parseNode == nil {
		return true
	}
	return parseNode.IsNull()
}

// IsSameDOMNode reports whether two DOMNode values reference the same platform node, treating all null-node representations as equivalent.
func IsSameDOMNode(parseLeft DOMNode, parseRight DOMNode) bool {
	if IsDOMNodeNull(parseLeft) || IsDOMNodeNull(parseRight) {
		return IsDOMNodeNull(parseLeft) == IsDOMNodeNull(parseRight)
	}
	return parseLeft.Equals(parseRight)
}

// DOMAdapter abstracts all DOM operations used by the runtime.
type DOMAdapter interface {
	// Element creation and manipulation
	CreateElement(tag string) DOMNode
	CreateTextNode(text string) DOMNode
	SetAttribute(node DOMNode, name, value string)
	RemoveAttribute(node DOMNode, name string)
	SetProperty(node DOMNode, name string, value any)
	GetProperty(node DOMNode, name string) any

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

	// Function wrapping
	WrapFunction(fn any) any

	// NOTE: the DOM adapter intentionally exposes no raw-HTML sink
	// (SetInnerHTML/innerHTML). The reconciler builds the tree exclusively
	// through CreateElement/CreateTextNode/SetTextContent/SetAttribute, so
	// untrusted content is always inserted as text or attributes, never parsed
	// as markup. TestDOMAdapterHasNoRawHTMLSink enforces that this stays true.
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
	PushState(state any, title, url string)
	ReplaceState(state any, title, url string)
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
