// Package agentui is GoWebComponents' agent-native generative-UI runtime (FC3). An agent
// (typically running server-side behind a //gwc:server function) emits a typed, JSON
// renderable schema; that schema is validated against a component allow-list and only then
// rendered to real UI.
//
// The safety property is structural, not heuristic: a Node carries no code, no event
// handlers, and no raw HTML — only a component Type that must appear in the allow-list,
// string Props the component explicitly permits, plain Text, and Children. An agent can
// therefore compose allow-listed components but can never inject behavior or markup. This
// is the "allow-listed components, not raw code" guarantee the A2UI / MCP-UI space
// converged on, and GoWebComponents' typed component model expresses it natively.
package agentui

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// Node is the typed, JSON-serializable renderable schema an agent emits.
type Node struct {
	// Type is the component name; it must be registered in the allow-list.
	Type string `json:"type"`
	// Props are string-only attributes; each key must be permitted by the component.
	Props map[string]string `json:"props,omitempty"`
	// Text is plain text content (escaped on render — never interpreted as markup).
	Text string `json:"text,omitempty"`
	// Children are nested nodes, each validated against the allow-list in turn.
	Children []Node `json:"children,omitempty"`
}

// ComponentSpec is one allow-list entry: a component name, the prop keys it permits, and
// how it renders to a safe ui.Node.
type ComponentSpec struct {
	Name         string
	AllowedProps []string
	// Render builds the safe ui.Node for this component. parseChildren are the ALREADY
	// validated and rendered child ui.Nodes (not raw agent output), so do not re-validate
	// them — just place them. parseProps holds the allow-listed prop values.
	Render func(parseProps map[string]string, parseChildren []ui.Node) ui.Node
}

func (parseS ComponentSpec) allowsProp(parseKey string) bool {
	return slices.Contains(parseS.AllowedProps, parseKey)
}

// Registry is the component allow-list a generative-UI tree is validated and rendered
// against. An app registers exactly the components it is willing to let an agent compose.
type Registry struct {
	specs map[string]ComponentSpec
}

// NewRegistry creates an empty allow-list.
func NewRegistry() *Registry {
	return &Registry{specs: map[string]ComponentSpec{}}
}

// Register adds (or replaces) a component in the allow-list.
func (parseR *Registry) Register(parseSpec ComponentSpec) {
	parseR.specs[parseSpec.Name] = parseSpec
}

// Allowed reports whether a component type is in the allow-list.
func (parseR *Registry) Allowed(parseType string) bool {
	_, parseOk := parseR.specs[parseType]
	return parseOk
}

// ComponentInfo describes one allow-listed component for introspection: its name and the
// prop keys it permits.
type ComponentInfo struct {
	Name         string   `json:"name"`
	AllowedProps []string `json:"allowedProps"`
}

// Catalog returns the allow-list as sorted, JSON-serializable descriptors — the data an
// agent (or an MCP tool exposing this registry) reads to learn exactly which components and
// props it may emit before generating a tree, turning the allow-list from an after-the-fact
// rejection into up-front guidance.
func (parseR *Registry) Catalog() []ComponentInfo {
	parseCatalog := make([]ComponentInfo, 0, len(parseR.specs))
	for _, parseSpec := range parseR.specs {
		parseProps := append([]string(nil), parseSpec.AllowedProps...)
		sort.Strings(parseProps)
		parseCatalog = append(parseCatalog, ComponentInfo{Name: parseSpec.Name, AllowedProps: parseProps})
	}
	sort.Slice(parseCatalog, func(parseA, parseB int) bool { return parseCatalog[parseA].Name < parseCatalog[parseB].Name })
	return parseCatalog
}

// Limits bound the size of an agent-emitted tree so untrusted output cannot exhaust memory
// or stack: MaxDepth caps nesting, MaxNodes caps the total node count. A zero field means
// "unbounded" for that dimension.
type Limits struct {
	MaxDepth int
	MaxNodes int
}

// DefaultLimits are the safe-by-default bounds applied by Validate — generous enough for
// any real UI, tight enough to reject a hostile or runaway tree.
//
// This is a package-level var read by Validate, so reassigning it changes the bounds
// process-wide. Treat it as read-only: to use different bounds for one surface, pass
// explicit Limits to ValidateWithLimits instead of mutating this value.
var DefaultLimits = Limits{MaxDepth: 32, MaxNodes: 10000}

// Validate checks a tree against the allow-list under DefaultLimits: every Type must be
// registered and every prop key permitted by its component, recursively, and the tree must
// stay within the default depth/size bounds. The returned error names the first violation
// and its path, so an agent's output is rejected with a precise reason.
func (parseR *Registry) Validate(parseNode Node) error {
	return parseR.ValidateWithLimits(parseNode, DefaultLimits)
}

// ValidateWithLimits is Validate with explicit size bounds — use it to tighten or relax the
// depth/node caps for a particular surface.
func (parseR *Registry) ValidateWithLimits(parseNode Node, parseLimits Limits) error {
	parseCount := 0
	return parseR.validateAt(parseNode, "root", 1, parseLimits, &parseCount)
}

func (parseR *Registry) validateAt(parseNode Node, parsePath string, parseDepth int, parseLimits Limits, parseCount *int) error {
	if parseLimits.MaxDepth > 0 && parseDepth > parseLimits.MaxDepth {
		return fmt.Errorf("%s: tree exceeds max depth %d", parsePath, parseLimits.MaxDepth)
	}
	*parseCount++
	if parseLimits.MaxNodes > 0 && *parseCount > parseLimits.MaxNodes {
		return fmt.Errorf("%s: tree exceeds max node count %d", parsePath, parseLimits.MaxNodes)
	}

	parseSpec, parseOk := parseR.specs[parseNode.Type]
	if !parseOk {
		return fmt.Errorf("%s: component type %q is not in the allow-list", parsePath, parseNode.Type)
	}
	for parseKey := range parseNode.Props {
		if !parseSpec.allowsProp(parseKey) {
			return fmt.Errorf("%s: component %q does not allow prop %q", parsePath, parseNode.Type, parseKey)
		}
	}
	for parseIndex, parseChild := range parseNode.Children {
		if parseErr := parseR.validateAt(parseChild, fmt.Sprintf("%s > %s[%d]", parsePath, parseNode.Type, parseIndex), parseDepth+1, parseLimits, parseCount); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// Render validates the tree and, only if it fully passes, builds it into a ui.Node — so
// untrusted agent output can never render unchecked.
func (parseR *Registry) Render(parseNode Node) (ui.Node, error) {
	if parseErr := parseR.Validate(parseNode); parseErr != nil {
		return nil, parseErr
	}
	return parseR.render(parseNode), nil
}

func (parseR *Registry) render(parseNode Node) ui.Node {
	parseSpec := parseR.specs[parseNode.Type]
	parseChildren := make([]ui.Node, 0, len(parseNode.Children)+1)
	if parseNode.Text != "" {
		parseChildren = append(parseChildren, html.Text(parseNode.Text))
	}
	for _, parseChild := range parseNode.Children {
		parseChildren = append(parseChildren, parseR.render(parseChild))
	}
	return parseSpec.Render(parseNode.Props, parseChildren)
}

// Parse decodes an agent's JSON schema into a Node.
func Parse(parseData []byte) (Node, error) {
	var parseNode Node
	if parseErr := json.Unmarshal(parseData, &parseNode); parseErr != nil {
		return Node{}, fmt.Errorf("parse agent UI schema: %w", parseErr)
	}
	return parseNode, nil
}

// RenderJSON parses, validates, and renders an agent's JSON schema in one step.
func (parseR *Registry) RenderJSON(parseData []byte) (ui.Node, error) {
	parseNode, parseErr := Parse(parseData)
	if parseErr != nil {
		return nil, parseErr
	}
	return parseR.Render(parseNode)
}

// DefaultRegistry is a ready-made allow-list of safe, presentational components — layout
// and content only, no interactivity — suitable for rendering agent-authored views out of
// the box. Apps can start here and Register more.
func DefaultRegistry() *Registry {
	parseRegistry := NewRegistry()

	parseTag := func(parseName, parseTagName, parseBaseClass string) {
		parseRegistry.Register(ComponentSpec{
			Name:         parseName,
			AllowedProps: []string{"class"},
			Render: func(parseProps map[string]string, parseChildren []ui.Node) ui.Node {
				return html.Tag(parseTagName, html.Props{Class: mergeClass(parseBaseClass, parseProps["class"])}, parseChildren...)
			},
		})
	}
	parseTag("stack", "div", "agentui-stack")
	parseTag("row", "div", "agentui-row")
	parseTag("card", "section", "agentui-card")
	parseTag("text", "p", "agentui-text")
	parseTag("badge", "span", "agentui-badge")
	parseTag("list", "ul", "agentui-list")
	parseTag("item", "li", "agentui-item")

	parseRegistry.Register(ComponentSpec{
		Name:         "heading",
		AllowedProps: []string{"class", "level"},
		Render: func(parseProps map[string]string, parseChildren []ui.Node) ui.Node {
			parseTagName := "h2"
			switch parseProps["level"] {
			case "1":
				parseTagName = "h1"
			case "3":
				parseTagName = "h3"
			case "4":
				parseTagName = "h4"
			}
			return html.Tag(parseTagName, html.Props{Class: mergeClass("agentui-heading", parseProps["class"])}, parseChildren...)
		},
	})

	return parseRegistry
}

// mergeClass appends an optional agent-supplied class to the component's base class. A CSS
// class string carries no behavior, so this stays within the safety boundary.
func mergeClass(parseBase, parseExtra string) string {
	if parseExtra == "" {
		return parseBase
	}
	return parseBase + " " + parseExtra
}
