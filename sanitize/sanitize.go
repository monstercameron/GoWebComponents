// Package sanitize provides HTML sanitization that strips XSS vectors while
// preserving safe formatting and structure markup.
package sanitize

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Policy defines the allowlist for HTML sanitization. Tags, attributes, and
// URL schemes outside these sets are stripped or unwrapped.
//
// A Policy's maps are only ever READ during Sanitize, so one Policy may be
// shared across concurrent Sanitize calls — but callers must treat a shared
// Policy as immutable: mutating its maps (e.g. AllowedURLSchemes["data"]=true)
// while another goroutine sanitizes is a data race. Build the Policy fully
// before sharing it.
type Policy struct {
	// AllowedTags is the set of element names that may appear in output.
	AllowedTags map[string]bool
	// AllowedAttributes is the set of attribute names that may appear in output.
	AllowedAttributes map[string]bool
	// AllowedURLSchemes is the set of scheme names (e.g. "https") allowed in
	// URL-bearing attributes. Relative and fragment URLs are always allowed.
	AllowedURLSchemes map[string]bool
}

// DefaultPolicy returns a safe allowlist covering common formatting and
// document-structure elements without permitting script execution vectors.
func DefaultPolicy() Policy {
	return Policy{
		AllowedTags: map[string]bool{
			"a": true, "p": true, "div": true, "span": true,
			"b": true, "strong": true, "i": true, "em": true,
			"u": true, "code": true, "pre": true, "blockquote": true,
			"ul": true, "ol": true, "li": true,
			"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
			"br": true, "hr": true,
			"table": true, "thead": true, "tbody": true, "tr": true, "td": true, "th": true,
			"img": true, "figure": true, "figcaption": true,
		},
		AllowedAttributes: map[string]bool{
			"href": true, "src": true, "alt": true, "title": true,
			"class": true, "id": true, "colspan": true, "rowspan": true,
		},
		AllowedURLSchemes: map[string]bool{
			"http": true, "https": true, "mailto": true,
		},
	}
}

// dropWithContents lists tags whose entire subtree — element and all
// descendants — must be removed. Unwrapping these would leak their text.
var dropWithContents = map[string]bool{
	"script": true, "style": true, "iframe": true, "object": true,
	"embed": true, "form": true, "svg": true, "math": true,
	"link": true, "meta": true, "base": true,
	// golang.org/x/net/html does NOT model <template>'s inert .content
	// fragment (a documented divergence from the HTML5 spec): its children are
	// ordinary tree children. Unwrapping them would promote inert template
	// markup into live, out-of-context output (invalid fragments — a mutation-
	// XSS precondition), so drop the whole subtree like svg/math/form.
	"template": true,
}

// voidElements are elements that have no children and need no closing tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// urlAttributes are attributes whose values must pass scheme validation.
var urlAttributes = map[string]bool{
	"href": true, "src": true, "action": true, "formaction": true, "xlink:href": true,
}

// Sanitize parses parseHTML and returns an XSS-safe HTML string. The first
// element of parseOptions, if provided, is used as the sanitization policy;
// otherwise DefaultPolicy is applied.
func Sanitize(parseHTML string, parseOptions ...Policy) string {
	parsePolicy := DefaultPolicy()
	if len(parseOptions) > 0 {
		parsePolicy = parseOptions[0]
	}
	return parsePolicy.Sanitize(parseHTML)
}

// Sanitize parses parseHTML and returns an XSS-safe HTML string using the
// receiver policy as the allowlist.
//
// Sanitize fails closed: any parse error (including input nested deeper than
// x/net/html's 512-node open-element-stack cap) yields "" rather than leaking
// raw input. It imposes no input-size ceiling of its own, so callers exposing
// it to untrusted request bodies should cap the body size upstream.
func (parsePolicy Policy) Sanitize(parseHTML string) string {
	if strings.TrimSpace(parseHTML) == "" {
		return ""
	}

	parseContextNode := &html.Node{
		Type:     html.ElementNode,
		Data:     "div",
		DataAtom: atom.Div,
	}
	parseNodes, parseErr := html.ParseFragment(strings.NewReader(parseHTML), parseContextNode)
	if parseErr != nil {
		// On parse failure return empty string rather than leaking raw input.
		return ""
	}

	var parseBuf strings.Builder
	for _, parseNode := range parseNodes {
		parsePolicy.walk(parseNode, &parseBuf)
	}
	return parseBuf.String()
}

// walk recursively renders parseNode into parseBuf, applying the policy rules
// at every level of the tree.
func (parsePolicy Policy) walk(parseNode *html.Node, parseBuf *strings.Builder) {
	switch parseNode.Type {
	case html.TextNode:
		// Escape text so embedded HTML characters can't re-open a tag context.
		parseBuf.WriteString(html.EscapeString(parseNode.Data))

	case html.CommentNode:
		// Comments are an XSS vector (IE conditional comments, etc.); drop them.

	case html.ElementNode:
		parseTag := strings.ToLower(parseNode.Data)

		if dropWithContents[parseTag] {
			// Discard the element and every descendant — no unwrapping.
			return
		}

		if parsePolicy.AllowedTags[parseTag] {
			// Emit the opening tag with filtered attributes.
			parseBuf.WriteByte('<')
			parseBuf.WriteString(parseTag)
			for _, parseAttr := range parseNode.Attr {
				parsePolicy.writeAttr(parseBuf, parseAttr)
			}
			parseBuf.WriteByte('>')

			if !voidElements[parseTag] {
				// Recurse into children, then close.
				for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
					parsePolicy.walk(parseChild, parseBuf)
				}
				parseBuf.WriteString("</")
				parseBuf.WriteString(parseTag)
				parseBuf.WriteByte('>')
			}
		} else {
			// Disallowed non-drop tag: unwrap — keep sanitized children, lose the tag.
			for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
				parsePolicy.walk(parseChild, parseBuf)
			}
		}

	default:
		// Document, DocumentFragment, or other container: recurse into children.
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parsePolicy.walk(parseChild, parseBuf)
		}
	}
}

// writeAttr emits one attribute into parseBuf after applying security rules.
// Event handlers, the style attribute, and URL attributes with unsafe schemes
// are silently dropped.
func (parsePolicy Policy) writeAttr(parseBuf *strings.Builder, parseAttr html.Attribute) {
	parseName := strings.ToLower(parseAttr.Key)

	// Strip event handlers — anything starting with "on".
	if strings.HasPrefix(parseName, "on") {
		return
	}

	// Strip the style attribute regardless of policy — it's a script vector.
	if parseName == "style" {
		return
	}

	// Enforce the attribute allowlist.
	if !parsePolicy.AllowedAttributes[parseName] {
		return
	}

	parseValue := parseAttr.Val

	// Validate URL schemes for URL-bearing attributes. Emit the SAME cleaned
	// string that was validated (not the raw value): validating a control-char-
	// stripped copy while emitting the original is the shape of bug that becomes
	// a bypass the moment the stripped set and the browser's scheme recovery
	// diverge.
	if urlAttributes[parseName] {
		parseValue = stripControlChars(parseAttr.Val)
		if !parsePolicy.isSafeURL(parseValue) {
			return
		}
	}

	// colspan/rowspan are allow-listed by name only; an oversized or negative
	// value is spec-invalid and a known client-side rendering-hang vector. Drop
	// the attribute unless it is a small positive integer within the HTML caps.
	if parseName == "colspan" && !isValidSpanValue(parseValue, 1000) {
		return
	}
	if parseName == "rowspan" && !isValidSpanValue(parseValue, 65534) {
		return
	}

	parseBuf.WriteByte(' ')
	parseBuf.WriteString(parseName)
	parseBuf.WriteString(`="`)
	parseBuf.WriteString(html.EscapeString(parseValue))
	parseBuf.WriteByte('"')
}

// isValidSpanValue reports whether a colspan/rowspan value is a positive integer
// no greater than parseMax.
func isValidSpanValue(parseValue string, parseMax int) bool {
	parseN, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
	if parseErr != nil {
		return false
	}
	return parseN >= 1 && parseN <= parseMax
}

// isSafeURL returns true when parseURL is safe to emit in an href/src context.
// Relative URLs (no scheme before the first path/query/fragment character) are
// always allowed. Absolute URLs must use a scheme listed in AllowedURLSchemes.
func (parsePolicy Policy) isSafeURL(parseURL string) bool {
	// Strip leading whitespace and control characters before inspecting.
	parseClean := stripControlChars(parseURL)

	// Find the first colon that precedes any '/', '?', or '#'.
	parseColonIdx := -1
	for parseI, parseCh := range parseClean {
		if parseCh == '/' || parseCh == '?' || parseCh == '#' {
			// Reached a path/query/fragment delimiter before any colon.
			// This is a relative URL — safe.
			return true
		}
		if parseCh == ':' {
			parseColonIdx = parseI
			break
		}
	}

	if parseColonIdx < 0 {
		// No colon found — relative URL or bare fragment. Safe.
		return true
	}

	parseScheme := strings.ToLower(parseClean[:parseColonIdx])
	return parsePolicy.AllowedURLSchemes[parseScheme]
}

// stripControlChars removes ASCII control characters (Unicode code points
// 0x00–0x20) that can be embedded in URLs to disguise scheme names.
func stripControlChars(parseInput string) string {
	var parseBuf strings.Builder
	parseBuf.Grow(len(parseInput))
	for _, parseRune := range parseInput {
		if unicode.IsSpace(parseRune) || unicode.IsControl(parseRune) || unicode.Is(unicode.Cf, parseRune) {
			continue
		}
		parseBuf.WriteRune(parseRune)
	}
	return parseBuf.String()
}
