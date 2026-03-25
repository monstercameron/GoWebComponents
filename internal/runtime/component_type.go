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
func NewComponentType(parseComponentID string, parseComponentName string, parseComponentQualifiedName string, parseComponentImplementation interface{}, render func(interface{}, map[string]interface{}) *Element) *ComponentType {
	return &ComponentType{
		ID:             parseComponentID,
		Name:           parseComponentName,
		QualifiedName:  parseComponentQualifiedName,
		implementation: parseComponentImplementation,
		render:         render,
	}
}

// Render invokes the current implementation attached to the component handle.
func (parseComponentType *ComponentType) Render(parseComponentProps map[string]interface{}) *Element {
	if parseComponentType == nil {
		return nil
	}

	parseComponentType.mu.RLock()
	parseCurrentImplementation := parseComponentType.implementation
	render := parseComponentType.render
	parseComponentType.mu.RUnlock()

	if parseCurrentImplementation == nil || render == nil {
		return nil
	}
	return render(parseCurrentImplementation, parseComponentProps)
}

// SetImplementation updates the current implementation for a stable component handle.
func (parseComponentType *ComponentType) SetImplementation(parseComponentImplementation interface{}) {
	if parseComponentType == nil {
		return
	}
	parseComponentType.mu.Lock()
	parseComponentType.implementation = parseComponentImplementation
	parseComponentType.mu.Unlock()
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
