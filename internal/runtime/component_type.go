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
func NewComponentType(id string, name string, qualifiedName string, implementation interface{}, render func(interface{}, map[string]interface{}) *Element) *ComponentType {
	return &ComponentType{
		ID:             id,
		Name:           name,
		QualifiedName:  qualifiedName,
		implementation: implementation,
		render:         render,
	}
}

// Render invokes the current implementation attached to the component handle.
func (component *ComponentType) Render(props map[string]interface{}) *Element {
	if component == nil {
		return nil
	}

	component.mu.RLock()
	implementation := component.implementation
	render := component.render
	component.mu.RUnlock()

	if implementation == nil || render == nil {
		return nil
	}
	return render(implementation, props)
}

// SetImplementation updates the current implementation for a stable component handle.
func (component *ComponentType) SetImplementation(implementation interface{}) {
	if component == nil {
		return
	}
	component.mu.Lock()
	component.implementation = implementation
	component.mu.Unlock()
}

// IdentityKey returns the logical identity used to compare component handles.
func (component *ComponentType) IdentityKey() string {
	if component == nil {
		return ""
	}
	if component.ID != "" {
		return component.ID
	}
	if component.QualifiedName != "" {
		return component.QualifiedName
	}
	return component.Name
}
