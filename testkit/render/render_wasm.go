//go:build js && wasm
// +build js,wasm

package render

import (
	"strings"
	"sync"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type config struct {
	synchronous bool
}

// Option configures the render fixture.
type Option func(*config)

// Event describes one synthetic event payload for fixture dispatch helpers.
type Event struct {
	Value   string
	Checked bool
	Key     string
	KeyCode int
}

var fixtureMu sync.Mutex

// WithQueuedScheduler configures the harness to queue work until Flush is called.
func WithQueuedScheduler() Option {
	return func(cfg *config) {
		cfg.synchronous = false
	}
}

// Fixture owns one mock DOM container, scheduler, and configured global runtime.
//
// The current harness serializes fixture ownership because the runtime hook
// surface is global on js/wasm builds.
type Fixture struct {
	tb        testing.TB
	adapter   *mockdom.MockDOMAdapter
	scheduler *mockdom.MockScheduler
	container *mockdom.MockDOMNode
	cleaned   bool
	unlock    sync.Once
}

// QueryNode wraps one matched rendered node.
type QueryNode struct {
	fixture *Fixture
	node    *mockdom.MockDOMNode
}

// New creates a controlled render fixture for js/wasm tests.
func New(tb testing.TB, options ...Option) *Fixture {
	tb.Helper()
	cfg := config{synchronous: true}
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}

	fixtureMu.Lock()
	adapter := mockdom.NewMockDOMAdapter()
	scheduler := mockdom.NewMockScheduler(cfg.synchronous)
	container, _ := adapter.CreateElement("div").(*mockdom.MockDOMNode)
	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})
	runtime.ClearDiagnostics()
	runtime.ClearLogs()

	fixture := &Fixture{
		tb:        tb,
		adapter:   adapter,
		scheduler: scheduler,
		container: container,
	}
	tb.Cleanup(func() {
		fixture.Cleanup()
	})
	return fixture
}

// Render mounts a UI tree into the fixture container.
func (f *Fixture) Render(root ui.Node) {
	f.tb.Helper()
	f.requireActive()
	if err := runtime.GetGlobalRuntime().RenderInto(f.container, root); err != nil {
		f.tb.Fatalf("render fixture failed to mount root: %v", err)
	}
	f.Flush()
}

// Rerender replaces the current tree with a new root.
func (f *Fixture) Rerender(root ui.Node) {
	f.Render(root)
}

// Flush drains queued scheduler work until the fixture settles.
func (f *Fixture) Flush() {
	f.tb.Helper()
	f.requireActive()
	f.scheduler.FlushAll()
}

// FlushTimers drains queued timeout work and any follow-up render work.
func (f *Fixture) FlushTimers() {
	f.tb.Helper()
	f.requireActive()
	f.scheduler.FlushTimeouts()
	f.scheduler.FlushAll()
}

// Stabilize drains pending scheduled work until the fixture settles.
func (f *Fixture) Stabilize() {
	f.Flush()
}

// Cleanup releases fixture ownership and clears buffered diagnostics.
func (f *Fixture) Cleanup() {
	if f == nil || f.cleaned {
		return
	}
	f.cleaned = true
	runtime.ClearDiagnostics()
	runtime.ClearLogs()
	f.container = nil
	f.adapter = nil
	f.scheduler = nil
	f.unlock.Do(func() {
		fixtureMu.Unlock()
	})
}

// Container returns the fixture root container.
func (f *Fixture) Container() *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	return &QueryNode{fixture: f, node: f.container}
}

// Target returns the explicit DOM target owned by the fixture.
func (f *Fixture) Target() any {
	if f == nil || f.container == nil {
		return nil
	}
	return f.container
}

// ByRole returns the first node whose computed role matches, optionally filtered by accessible name.
func (f *Fixture) ByRole(role string, name string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	wantRole := normalizeText(role)
	wantName := normalizeText(name)
	return f.wrap(findNode(f.container, func(node *mockdom.MockDOMNode) bool {
		if node == nil || normalizeText(nodeRole(node)) != wantRole {
			return false
		}
		if wantName == "" {
			return true
		}
		return normalizeText(accessibleName(f.container, node)) == wantName
	}))
}

// AllByRole returns all nodes whose computed role matches.
func (f *Fixture) AllByRole(role string) []*QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	wantRole := normalizeText(role)
	matches := collectNodes(f.container, func(node *mockdom.MockDOMNode) bool {
		return normalizeText(nodeRole(node)) == wantRole
	})
	result := make([]*QueryNode, 0, len(matches))
	for _, match := range matches {
		result = append(result, f.wrap(match))
	}
	return result
}

// ByID returns the first node with the requested id attribute.
func (f *Fixture) ByID(id string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	return f.wrap(findNode(f.container, func(node *mockdom.MockDOMNode) bool {
		return node.Attrs["id"] == id
	}))
}

// ByText returns the first node whose full text content matches the provided text.
func (f *Fixture) ByText(text string) *QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	want := normalizeText(text)
	return f.wrap(findNode(f.container, func(node *mockdom.MockDOMNode) bool {
		return normalizeText(nodeText(node)) == want
	}))
}

// AllByTag returns all nodes matching the requested tag name.
func (f *Fixture) AllByTag(tag string) []*QueryNode {
	if f == nil || f.container == nil {
		return nil
	}
	matches := collectNodes(f.container, func(node *mockdom.MockDOMNode) bool {
		return strings.EqualFold(node.Tag, tag)
	})
	result := make([]*QueryNode, 0, len(matches))
	for _, match := range matches {
		result = append(result, f.wrap(match))
	}
	return result
}

// Text returns the full fixture container text.
func (f *Fixture) Text() string {
	if f == nil || f.container == nil {
		return ""
	}
	return nodeText(f.container)
}

// DispatchByID invokes one handler property on the matched node and settles the fixture.
func (f *Fixture) DispatchByID(id string, property string, event Event) {
	f.tb.Helper()
	node := f.ByID(id)
	if node == nil {
		f.tb.Fatalf("render fixture could not find node with id %q", id)
	}
	node.Dispatch(property, event)
}

// ClickByID invokes the matched node's `onclick` handler and settles the fixture.
func (f *Fixture) ClickByID(id string) {
	f.DispatchByID(id, "onclick", Event{})
}

// InputByID updates the matched node value, invokes `oninput`, and settles the fixture.
func (f *Fixture) InputByID(id string, value string) {
	f.DispatchByID(id, "oninput", Event{Value: value})
}

// ChangeByID updates the matched node value, invokes `onchange`, and settles the fixture.
func (f *Fixture) ChangeByID(id string, value string) {
	f.DispatchByID(id, "onchange", Event{Value: value})
}

// SubmitByID invokes the matched node's `onsubmit` handler and settles the fixture.
func (f *Fixture) SubmitByID(id string) {
	f.DispatchByID(id, "onsubmit", Event{})
}

// Exists reports whether the node wrapper points at a real node.
func (n *QueryNode) Exists() bool {
	return n != nil && n.node != nil
}

// Tag returns the node tag name.
func (n *QueryNode) Tag() string {
	if n == nil || n.node == nil {
		return ""
	}
	return n.node.Tag
}

// Name returns the node accessible name used by role-based queries.
func (n *QueryNode) Name() string {
	if n == nil || n.node == nil || n.fixture == nil || n.fixture.container == nil {
		return ""
	}
	return accessibleName(n.fixture.container, n.node)
}

// Text returns the full text content beneath the node.
func (n *QueryNode) Text() string {
	if n == nil || n.node == nil {
		return ""
	}
	return nodeText(n.node)
}

// Attr returns one attribute value.
func (n *QueryNode) Attr(name string) string {
	if n == nil || n.node == nil {
		return ""
	}
	return n.node.Attrs[name]
}

// NodeID returns the stable mock-DOM node id for identity-sensitive assertions.
func (n *QueryNode) NodeID() int {
	if n == nil || n.node == nil {
		return 0
	}
	return n.node.ID
}

// Property returns one raw property value from the rendered node.
func (n *QueryNode) Property(name string) any {
	if n == nil || n.node == nil {
		return nil
	}
	return n.node.Props[name]
}

// Children returns wrapped child nodes.
func (n *QueryNode) Children() []*QueryNode {
	if n == nil || n.node == nil || len(n.node.Children) == 0 {
		return nil
	}
	children := make([]*QueryNode, 0, len(n.node.Children))
	for _, child := range n.node.Children {
		children = append(children, n.fixture.wrap(child))
	}
	return children
}

// Dispatch invokes one handler property on the current node and settles the fixture.
func (n *QueryNode) Dispatch(property string, event Event) {
	if n == nil || n.node == nil || n.fixture == nil {
		return
	}
	n.fixture.dispatch(n.node, property, event)
}

// Click invokes the current node's `onclick` handler and settles the fixture.
func (n *QueryNode) Click() {
	n.Dispatch("onclick", Event{})
}

// Input updates the current node value, invokes `oninput`, and settles the fixture.
func (n *QueryNode) Input(value string) {
	n.Dispatch("oninput", Event{Value: value})
}

// Change updates the current node value, invokes `onchange`, and settles the fixture.
func (n *QueryNode) Change(value string) {
	n.Dispatch("onchange", Event{Value: value})
}

// Submit invokes the current node's `onsubmit` handler and settles the fixture.
func (n *QueryNode) Submit() {
	n.Dispatch("onsubmit", Event{})
}

func (f *Fixture) requireActive() {
	if f == nil || f.cleaned || f.container == nil || f.scheduler == nil {
		f.tb.Fatal("render fixture is no longer active")
	}
}

func (f *Fixture) wrap(node *mockdom.MockDOMNode) *QueryNode {
	if node == nil {
		return nil
	}
	return &QueryNode{fixture: f, node: node}
}

func (f *Fixture) dispatch(node *mockdom.MockDOMNode, property string, event Event) {
	f.tb.Helper()
	f.requireActive()
	if node == nil {
		f.tb.Fatalf("render fixture cannot dispatch %q on a nil node", property)
	}
	handler := node.Props[property]
	if handler == nil {
		f.tb.Fatalf("render fixture node %q does not expose handler property %q", node.Attrs["id"], property)
	}
	event.apply(node)
	synthetic := event.syntheticValue()
	syntheticGoEvent := runtime.NewGoEvent(synthetic)

	switch typed := handler.(type) {
	case func():
		typed()
	case func(string):
		typed(event.Value)
	case func(js.Value):
		typed(synthetic)
	case func(js.Value) error:
		if err := typed(synthetic); err != nil {
			f.tb.Fatalf("render fixture handler %q returned error: %v", property, err)
		}
	case func(runtime.GoEvent):
		typed(syntheticGoEvent)
	case func(runtime.GoEvent) error:
		if err := typed(syntheticGoEvent); err != nil {
			f.tb.Fatalf("render fixture handler %q returned error: %v", property, err)
		}
	case func() error:
		if err := typed(); err != nil {
			f.tb.Fatalf("render fixture handler %q returned error: %v", property, err)
		}
	default:
		f.tb.Fatalf("render fixture does not know how to dispatch handler property %q with type %T", property, handler)
	}
	if property == "onchange" || property == "oninput" || property == "onsubmit" || property == "onclick" {
		f.Stabilize()
	}
}

func findNode(node *mockdom.MockDOMNode, match func(*mockdom.MockDOMNode) bool) *mockdom.MockDOMNode {
	if node == nil {
		return nil
	}
	if match(node) {
		return node
	}
	for _, child := range node.Children {
		if found := findNode(child, match); found != nil {
			return found
		}
	}
	return nil
}

func collectNodes(node *mockdom.MockDOMNode, match func(*mockdom.MockDOMNode) bool) []*mockdom.MockDOMNode {
	if node == nil {
		return nil
	}
	result := make([]*mockdom.MockDOMNode, 0)
	if match(node) {
		result = append(result, node)
	}
	for _, child := range node.Children {
		result = append(result, collectNodes(child, match)...)
	}
	return result
}

func nodeText(node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if node.Tag == "#text" {
		return node.TextContent
	}
	parts := make([]string, 0, len(node.Children)+1)
	if strings.TrimSpace(node.TextContent) != "" {
		parts = append(parts, strings.TrimSpace(node.TextContent))
	}
	for _, child := range node.Children {
		text := strings.TrimSpace(nodeText(child))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func nodeRole(node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if role := strings.TrimSpace(node.Attrs["role"]); role != "" {
		return role
	}
	switch strings.ToLower(strings.TrimSpace(node.Tag)) {
	case "button":
		return "button"
	case "a":
		if strings.TrimSpace(node.Attrs["href"]) != "" {
			return "link"
		}
	case "textarea":
		return "textbox"
	case "select":
		return "combobox"
	case "img":
		return "img"
	case "form":
		return "form"
	case "input":
		switch strings.ToLower(strings.TrimSpace(node.Attrs["type"])) {
		case "button", "submit", "reset":
			return "button"
		case "checkbox":
			return "checkbox"
		case "radio":
			return "radio"
		case "range":
			return "slider"
		case "email", "password", "search", "tel", "text", "url", "":
			return "textbox"
		}
	}
	return ""
}

func accessibleName(root *mockdom.MockDOMNode, node *mockdom.MockDOMNode) string {
	if node == nil {
		return ""
	}
	if label := normalizeText(node.Attrs["aria-label"]); label != "" {
		return label
	}
	if refs := normalizeText(node.Attrs["aria-labelledby"]); refs != "" {
		parts := make([]string, 0)
		for _, ref := range strings.Fields(refs) {
			if target := findNode(root, func(candidate *mockdom.MockDOMNode) bool {
				return candidate != nil && candidate.Attrs["id"] == ref
			}); target != nil {
				text := normalizeText(nodeText(target))
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return normalizeText(nodeText(node))
}

func (e Event) apply(node *mockdom.MockDOMNode) {
	if node == nil {
		return
	}
	if e.Value != "" || node.Tag == "input" || node.Tag == "textarea" || node.Tag == "select" {
		node.Attrs["value"] = e.Value
		node.Props["value"] = e.Value
	}
	node.Props["checked"] = e.Checked
	if e.Checked {
		node.Attrs["checked"] = ""
	} else {
		delete(node.Attrs, "checked")
	}
}

func (e Event) syntheticValue() js.Value {
	object := js.Global().Get("Object")
	target := object.New()
	target.Set("value", e.Value)
	target.Set("checked", e.Checked)
	event := object.New()
	event.Set("target", target)
	event.Set("key", e.Key)
	event.Set("keyCode", e.KeyCode)
	return event
}
