package devtools

import "sync"

var extensionSectionsState struct {
	mu       sync.RWMutex
	sections []ExtensionSection
}

// SetExtensionSections replaces the current app-owned or companion-owned devtools sections.
func SetExtensionSections(sections []ExtensionSection) {
	extensionSectionsState.mu.Lock()
	defer extensionSectionsState.mu.Unlock()
	extensionSectionsState.sections = cloneExtensionSections(sections)
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

func cloneExtensionSections(sections []ExtensionSection) []ExtensionSection {
	if len(sections) == 0 {
		return nil
	}
	cloned := make([]ExtensionSection, len(sections))
	for i, section := range sections {
		cloned[i] = ExtensionSection{
			Name:    section.Name,
			Summary: cloneMultiClientStringMap(section.Summary),
			Lines:   append([]string(nil), section.Lines...),
		}
	}
	return cloned
}
