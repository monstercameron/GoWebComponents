package runtime

import "sync"

// ComponentType provides a stable runtime-recognized component handle that can
// carry logical identity separately from the current callable implementation.
type ComponentType struct {
	ID            string
	Name          string
	QualifiedName string

	mu             sync.RWMutex
	implementation interface{}
	render         func(interface{}, map[string]interface{}) *Element
}

// NewComponentType constructs a component handle recognized by the runtime.
func NewComponentType(parseId string, parseName string, parseQualifiedName string, parseImplementation interface{}, render func(interface{}, map[string]interface{}) *Element) *ComponentType {
	return &ComponentType{
		ID:             parseId,
		Name:           parseName,
		QualifiedName:  parseQualifiedName,
		implementation: parseImplementation,
		render:         render,
	}
}

// Render invokes the current implementation attached to the component handle.
func (parseComponent *ComponentType) Render(parseProps map[string]interface{}) *Element {
	if parseComponent == nil {
		return nil
	}

	parseComponent.mu.RLock()
	parseImplementation := parseComponent.implementation
	render := parseComponent.render
	parseComponent.mu.RUnlock()

	if parseImplementation == nil || render == nil {
		return nil
	}
	return render(parseImplementation, parseProps)
}

// SetImplementation updates the current implementation for a stable component handle.
func (parseComponent *ComponentType) SetImplementation(parseImplementation interface{}) {
	if parseComponent == nil {
		return
	}
	parseComponent.mu.Lock()
	parseComponent.implementation = parseImplementation
	parseComponent.mu.Unlock()
}

// IdentityKey returns the logical identity used to compare component handles.
func (parseComponent *ComponentType) IdentityKey() string {
	if parseComponent == nil {
		return ""
	}
	if parseComponent.ID != "" {
		return parseComponent.ID
	}
	if parseComponent.QualifiedName != "" {
		return parseComponent.QualifiedName
	}
	return parseComponent.Name
}
