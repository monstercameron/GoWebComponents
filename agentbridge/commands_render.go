package agentbridge

import (
	"encoding/json"
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// RegisterRenderCommands installs bridge.render-tree, which interprets a JSON
// element tree into a live DOM subtree using the runtime element builders and
// mounts it into a selector. This is structure-only: it injects arbitrary
// markup (tags, attributes, text, nesting) at runtime without compiling any Go
// code. It carries no behavior — event handlers and hooks are Go closures that
// cannot be shipped over the wire, so on* attributes are dropped.
func RegisterRenderCommands() {
	RegisterAgentCommand("bridge.render-tree", renderHandleTree)
}

// renderTreeNode is one node of an agent-supplied element tree.
type renderTreeNode struct {
	Tag      string            `json:"tag"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Text     string            `json:"text,omitempty"`
	Children []renderTreeNode  `json:"children,omitempty"`
}

// renderTreePayload is the decoded bridge.render-tree command.
type renderTreePayload struct {
	Selector string         `json:"selector"`
	Tree     renderTreeNode `json:"tree"`
}

// renderAllowedTags is the allowlist of element tags an agent may inject.
// Behavior-bearing or dangerous tags (script, style, iframe, object, ...) are
// excluded so a structured injection cannot become a code-execution vector.
var renderAllowedTags = map[string]bool{
	"div": true, "span": true, "p": true, "section": true, "header": true,
	"footer": true, "main": true, "article": true, "nav": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true, "a": true, "button": true,
	"strong": true, "em": true, "b": true, "i": true, "small": true,
	"code": true, "pre": true, "br": true, "hr": true, "img": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "td": true, "th": true,
	"label": true, "blockquote": true,
}

// renderHandleTree builds and mounts an agent-supplied element tree.
func renderHandleTree(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}
	var parseDec renderTreePayload
	if parseErr := json.Unmarshal(parsePayload, &parseDec); parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "render-tree: malformed payload: " + parseErr.Error()}
	}
	parseSelector := strings.TrimSpace(parseDec.Selector)
	if parseSelector == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "render-tree: missing required field \"selector\""}
	}
	if !runtime.GetGlobalRuntime().SelectorResolves(parseSelector) {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "render-tree: selector " + parseSelector + " did not resolve to a container"}
	}

	parseElement, parseBuildErr := renderBuildElement(parseDec.Tree)
	if parseBuildErr != nil {
		return nil, parseBuildErr
	}
	// Render through an ISOLATED runtime (RenderDetached) so the injected tree
	// gets its own root and cannot tear down the app's single currentRoot the
	// way a plain RenderTo into a second container would.
	if parseErr := runtime.RenderDetached(parseSelector, parseElement); parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "render-tree: " + parseErr.Error()}
	}

	runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
	// Reversible: undo clears the target container (same isolated runtime).
	parseUndoSelector := parseSelector
	RecordAgentMutation("bridge.render-tree", "render tree into "+parseSelector, func() *EnvelopeError {
		_ = runtime.RenderDetached(parseUndoSelector, nil)
		runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
		return nil
	})
	return writeEncodeOK()
}

// renderBuildElement recursively interprets a renderTreeNode into a runtime
// Element using the runtime builders. It allowlists tags, strips on* event
// attributes, and blocks unsafe href/src schemes.
func renderBuildElement(parseNode renderTreeNode) (*runtime.Element, *EnvelopeError) {
	parseTag := strings.ToLower(strings.TrimSpace(parseNode.Tag))
	if !renderAllowedTags[parseTag] {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "render-tree: tag " + parseNode.Tag + " is not allowed"}
	}

	parseProps := map[string]any{}
	for parseKey, parseValue := range parseNode.Attrs {
		parseLowerKey := strings.ToLower(strings.TrimSpace(parseKey))
		if strings.HasPrefix(parseLowerKey, "on") {
			continue // behavior cannot be injected
		}
		if (parseLowerKey == "href" || parseLowerKey == "src") && renderUnsafeURL(parseValue) {
			continue
		}
		parseProps[parseKey] = parseValue
	}

	var parseChildren []any
	if parseNode.Text != "" {
		parseChildren = append(parseChildren, runtime.CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": parseNode.Text}))
	}
	for _, parseChild := range parseNode.Children {
		parseBuilt, parseErr := renderBuildElement(parseChild)
		if parseErr != nil {
			return nil, parseErr
		}
		parseChildren = append(parseChildren, parseBuilt)
	}
	return runtime.CreateElement(parseTag, parseProps, parseChildren...), nil
}

// renderUnsafeURL reports whether a URL uses a script-bearing scheme.
func renderUnsafeURL(parseValue string) bool {
	parseTrimmed := strings.ToLower(strings.TrimSpace(parseValue))
	return strings.HasPrefix(parseTrimmed, "javascript:") ||
		strings.HasPrefix(parseTrimmed, "data:") ||
		strings.HasPrefix(parseTrimmed, "vbscript:")
}
