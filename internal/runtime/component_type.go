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

// SetImplementationRenderer updates the current implementation and, when provided, swaps in one matching renderer.
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
// the very same function value (code pointer and closure data). Hot-swap
// callers use it to skip rebuilding an identical renderer on every element
// creation; a recreated closure (fresh captures, same code) does not match, so
// hot-reload and inline-component swaps still take the update path.
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
