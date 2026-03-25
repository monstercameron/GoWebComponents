package devtools

import "sync"

var extensionSectionsState struct {
	mu       sync.RWMutex
	sections []ExtensionSection
}

// SetExtensionSections replaces the current app-owned or companion-owned devtools sections.
func SetExtensionSections(parseSections []ExtensionSection) {
	extensionSectionsState.mu.Lock()
	defer extensionSectionsState.mu.Unlock()
	extensionSectionsState.sections = cloneExtensionSections(parseSections)
}

// ResetExtensionSections clears the current app-owned or companion-owned devtools sections.
func ResetExtensionSections() {
	SetExtensionSections(nil)
}

// InspectExtensionSections returns the current app-owned or companion-owned devtools sections.
func InspectExtensionSections() []ExtensionSection {
	extensionSectionsState.mu.RLock()
	defer extensionSectionsState.mu.RUnlock()
	return cloneExtensionSections(extensionSectionsState.sections)
}

func cloneExtensionSections(parseSections []ExtensionSection) []ExtensionSection {
	if len(parseSections) == 0 {
		return nil
	}
	parseCloned := make([]ExtensionSection, len(parseSections))
	for parseI, parseSection := range parseSections {
		parseCloned[parseI] = ExtensionSection{
			Name:    parseSection.Name,
			Summary: cloneMultiClientStringMap(parseSection.Summary),
			Lines:   append([]string(nil), parseSection.Lines...),
		}
	}
	return parseCloned
}
