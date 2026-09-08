package shorthand

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/sanitize"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// RawHTML parses sanitized markup into nodes (safe for untrusted input).
// Delegates to [html.RawHTML].
func RawHTML(parseMarkup string) []ui.Node { return html.RawHTML(parseMarkup) }

// RawHTMLWith parses markup under a caller-provided sanitize policy.
func RawHTMLWith(parseMarkup string, parsePolicy sanitize.Policy) []ui.Node {
	return html.RawHTMLWith(parseMarkup, parsePolicy)
}

// RawHTMLUnsafe parses trusted markup without sanitization (still a real node
// tree, never innerHTML). Delegates to [html.RawHTMLUnsafe].
func RawHTMLUnsafe(parseMarkup string) []ui.Node { return html.RawHTMLUnsafe(parseMarkup) }
