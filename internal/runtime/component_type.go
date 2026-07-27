package runtime

import (
	"sync"
	"sync/atomic"
)

type componentRenderState struct {
	getImplementation any
	getRender         func(any, map[string]any) *Element
}

// ComponentType provides a stable runtime-recognized component handle that can
// carry logical identity separately from the current callable implementation.
type ComponentType struct {
	ID            string
	Name          string
	QualifiedName string

	getState atomic.Value
	// setMu serializes the read-modify-write in SetImplementationRenderer;
	// atomic.Value alone would let two concurrent swaps lose one update.
	// Render stays lock-free (Load only).
	setMu sync.Mutex
}

// NewComponentType constructs a component handle recognized by the runtime.
func NewComponentType(parseComponentID string, parseComponentName string, parseComponentQualifiedName string, parseComponentImplementation any, render func(any, map[string]any) *Element) *ComponentType {
	getComponentType := &ComponentType{
		ID:            parseComponentID,
		Name:          parseComponentName,
		QualifiedName: parseComponentQualifiedName,
	}
	getComponentType.getState.Store(componentRenderState{
		getImplementation: parseComponentImplementation,
		getRender:         render,
	})
	return getComponentType
}

// Render invokes the current implementation attached to the component handle.
func (parseComponentType *ComponentType) Render(parseComponentProps map[string]any) *Element {
	if parseComponentType == nil {
		return nil
	}

	parseStateValue := parseComponentType.getState.Load()
	if parseStateValue == nil {
		return nil
	}
	parseState := parseStateValue.(componentRenderState)

	if parseState.getImplementation == nil || parseState.getRender == nil {
		return nil
	}
	return parseState.getRender(parseState.getImplementation, parseComponentProps)
}

// SetImplementation updates the current implementation for a stable component handle.
func (parseComponentType *ComponentType) SetImplementation(parseComponentImplementation any) {
	parseComponentType.SetImplementationRenderer(parseComponentImplementation, nil)
}

// SetImplementationRenderer updates the current implementation and, when
// provided, swaps in one matching renderer.
//
// ONLY THE OWNER OF A HANDLE MAY CALL THIS. A handle is shared by every element
// built from it, so a swap here retroactively changes what those elements
// render — that is the point for hot reload and for ui.Typed installing its
// static renderer, and it is a data-corruption bug for anything else.
//
// Concretely: ui.getComponentHandle used to call this whenever a lookup arrived
// with a function value different from the cached one. Since every closure
// created from a single `func` literal reports the same qualified name, N
// sibling components built from one literal resolved to one handle and each
// registration overwrote the last — so all N rendered the final closure's
// captured props (observed: four form inputs all named "subject"). Distinct
// closures now get distinct handles; see the identity discussion in
// ui/component_handle_shared.go before reintroducing a name-keyed swap.
func (parseComponentType *ComponentType) SetImplementationRenderer(parseComponentImplementation any, parseRender func(any, map[string]any) *Element) {
	if parseComponentType == nil {
		return
	}
	parseComponentType.setMu.Lock()
	defer parseComponentType.setMu.Unlock()
	parseCurrentStateValue := parseComponentType.getState.Load()
	parseCurrentState := componentRenderState{}
	if parseCurrentStateValue != nil {
		parseCurrentState = parseCurrentStateValue.(componentRenderState)
	}
	parseCurrentState.getImplementation = parseComponentImplementation
	if parseRender != nil {
		parseCurrentState.getRender = parseRender
	}
	parseComponentType.getState.Store(parseCurrentState)
}

// ImplementationMatches reports whether the handle's current implementation is
// the very same function value — code pointer AND closure data, so two closures
// compiled from one `func` literal with different captures do NOT match.
//
// That distinction is the whole point: a name (or a bare code pointer) cannot
// tell those two closures apart, and treating them as one component is what let
// sibling components overwrite each other's implementations. Callers use this
// to answer "is this handle already carrying exactly this code?" — yes means
// reuse it as-is (the steady-state fast path for top-level components, which
// present one stable function value forever), no means this is a different
// program and needs a handle of its own.
func (parseComponentType *ComponentType) ImplementationMatches(parseComponentImplementation any) bool {
	if parseComponentType == nil {
		return false
	}
	parseStateValue := parseComponentType.getState.Load()
	if parseStateValue == nil {
		return false
	}
	parseState := parseStateValue.(componentRenderState)
	return parseState.getRender != nil && sameFunctionIdentity(parseState.getImplementation, parseComponentImplementation)
}

// IdentityKey returns the logical identity used to compare component handles.
func (parseComponentType *ComponentType) IdentityKey() string {
	if parseComponentType == nil {
		return ""
	}
	if parseComponentType.ID != "" {
		return parseComponentType.ID
	}
	if parseComponentType.QualifiedName != "" {
		return parseComponentType.QualifiedName
	}
	return parseComponentType.Name
}
