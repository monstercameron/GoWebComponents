//go:build js && wasm
// +build js,wasm

package ui

func (m *overlayStackManager) subscribe(notify func()) func() {
	if notify == nil {
		return func() {}
	}
	m.mu.Lock()
	m.nextSubscriber++
	subscriberID := m.nextSubscriber
	m.subscribers = append(m.subscribers, overlaySubscriber{id: subscriberID, notify: notify})
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for index, subscriber := range m.subscribers {
			if subscriber.id == subscriberID {
				m.subscribers = append(m.subscribers[:index], m.subscribers[index+1:]...)
				return
			}
		}
	}
}

func (m *overlayStackManager) remove(id string) {
	if id == "" {
		return
	}
	m.mu.Lock()
	if _, exists := m.entries[id]; !exists {
		m.mu.Unlock()
		return
	}
	delete(m.entries, id)
	subscribers := append([]overlaySubscriber(nil), m.subscribers...)
	m.mu.Unlock()
	m.notify(subscribers)
}

func overlayRole(kind OverlayKind, explicit string) string {
	if explicit != "" {
		return explicit
	}
	switch kind {
	case OverlayKindTooltip:
		return "tooltip"
	case OverlayKindMenu:
		return "menu"
	default:
		return "dialog"
	}
}

func cloneOverlayStyle(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(source)+1)
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
