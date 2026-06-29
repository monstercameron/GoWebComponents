//go:build js && wasm

package ui

// subscribe is a core package helper.
func (parseM *overlayStackManager) subscribe(parseNotify func()) func() {
	if parseNotify == nil {
		return func() {}
	}
	parseM.mu.Lock()
	parseM.nextSubscriber++
	parseSubscriberID := parseM.nextSubscriber
	parseM.subscribers = append(parseM.subscribers, overlaySubscriber{id: parseSubscriberID, notify: parseNotify})
	parseM.mu.Unlock()
	return func() {
		parseM.mu.Lock()
		defer parseM.mu.Unlock()
		for parseIndex, parseSubscriber := range parseM.subscribers {
			if parseSubscriber.id == parseSubscriberID {
				parseM.subscribers = append(parseM.subscribers[:parseIndex], parseM.subscribers[parseIndex+1:]...)
				return
			}
		}
	}
}

// remove is a core package helper.
func (parseM *overlayStackManager) remove(parseId string) {
	if parseId == "" {
		return
	}
	parseM.mu.Lock()
	if _, parseExists := parseM.entries[parseId]; !parseExists {
		parseM.mu.Unlock()
		return
	}
	delete(parseM.entries, parseId)
	parseSubscribers := append([]overlaySubscriber(nil), parseM.subscribers...)
	parseM.mu.Unlock()
	parseM.notify(parseSubscribers)
}

// overlayRole is a core package helper.
func overlayRole(parseKind OverlayKind, parseExplicit string) string {
	if parseExplicit != "" {
		return parseExplicit
	}
	switch parseKind {
	case OverlayKindTooltip:
		return "tooltip"
	case OverlayKindMenu:
		return "menu"
	default:
		return "dialog"
	}
}

// cloneOverlayStyle is a core package helper.
func cloneOverlayStyle(parseSource map[string]string) map[string]string {
	if len(parseSource) == 0 {
		return map[string]string{}
	}
	parseClone := make(map[string]string, len(parseSource)+1)
	for parseKey, parseValue := range parseSource {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}
