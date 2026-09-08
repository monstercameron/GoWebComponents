package servercomponents

import "github.com/monstercameron/GoWebComponents/v6/ui"

// Descriptor is the serializable server-component model shared with build tools.
type Descriptor struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Props       map[string]string `json:"props,omitempty"`
	ClientSlots []ClientReference `json:"clientSlots,omitempty"`
}

// ClientReference identifies a client-side island passed through a server-only tree.
type ClientReference struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Props  map[string]string `json:"props,omitempty"`
	Export string            `json:"export,omitempty"`
}

// Props configures one server-only component boundary.
type Props struct {
	ID          string
	Name        string
	Props       map[string]string
	ClientSlots []ClientReference
	Render      func() ui.Node
	Placeholder ui.Node
}

// Manifest is emitted by server rendering so bundlers can omit server-only code from the wasm entry.
type Manifest struct {
	Components []Descriptor `json:"components"`
}

// Collect appends a server-only descriptor to a manifest.
func Collect(parseManifest *Manifest, parseProps Props) {
	if parseManifest == nil {
		return
	}
	parseManifest.Components = append(parseManifest.Components, Descriptor{
		ID:          parseProps.ID,
		Name:        parseProps.Name,
		Props:       cloneStringMap(parseProps.Props),
		ClientSlots: cloneClientReferences(parseProps.ClientSlots),
	})
}

func cloneStringMap(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return nil
	}
	parseCloned := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

func cloneClientReferences(parseValues []ClientReference) []ClientReference {
	if len(parseValues) == 0 {
		return nil
	}
	parseCloned := make([]ClientReference, len(parseValues))
	for parseIndex, parseValue := range parseValues {
		parseCloned[parseIndex] = parseValue
		parseCloned[parseIndex].Props = cloneStringMap(parseValue.Props)
	}
	return parseCloned
}
