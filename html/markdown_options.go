package html

import "github.com/monstercameron/GoWebComponents/v6/ui"

// denyAllMarkdownSchemes is an allowlist no real URL scheme can match; it makes an
// explicit empty WithURLSchemes list reject every schemed destination.
var denyAllMarkdownSchemes = []string{"\x00"}

// MarkdownOption configures one aspect of [Markdown]. Build options with
// [WithURLSchemes], or pass a [MarkdownRenderOptions] value, which applies its
// whole configuration at once.
type MarkdownOption interface {
	applyMarkdown(*MarkdownRenderOptions)
}

// applyMarkdown replaces the whole configuration with the receiver, so a
// MarkdownRenderOptions value passed to [Markdown] behaves as it always did.
// Pass it before any other option; later options refine it.
func (parseOptions MarkdownRenderOptions) applyMarkdown(parseTarget *MarkdownRenderOptions) {
	*parseTarget = parseOptions
}

// markdownOptionFunc adapts a function to [MarkdownOption].
type markdownOptionFunc func(*MarkdownRenderOptions)

// applyMarkdown calls the wrapped function.
func (parseFunc markdownOptionFunc) applyMarkdown(parseTarget *MarkdownRenderOptions) {
	parseFunc(parseTarget)
}

// WithURLSchemes sets the URL scheme allowlist for links, images and autolinks.
// Schemes are written without the trailing colon. Calling it with no schemes
// allows none, so only relative paths, absolute paths and fragments survive.
// Omitting WithURLSchemes keeps the default allowlist (http, https, mailto).
func WithURLSchemes(parseSchemes ...string) MarkdownOption {
	parseCopy := append([]string{}, parseSchemes...)
	return markdownOptionFunc(func(parseTarget *MarkdownRenderOptions) {
		parseTarget.AllowedURLSchemes = parseCopy
		parseTarget.isSchemeOverride = true
	})
}

// GetAllowedSchemes returns the allowlist handed to SanitizeMarkdownHref: the
// configured list, or a deny-all list when WithURLSchemes was given no schemes.
func (parseOptions MarkdownRenderOptions) GetAllowedSchemes() []string {
	if parseOptions.isSchemeOverride && len(parseOptions.AllowedURLSchemes) == 0 {
		return denyAllMarkdownSchemes
	}
	return parseOptions.AllowedURLSchemes
}

// Markdown renders Markdown source into nodes, applying every option in order.
// Untrusted source is still scheme-sanitized; raw HTML is never emitted (route
// untrusted HTML through the sanitize package instead). Unlike the earlier
// variadic-struct form, no option is silently dropped.
func Markdown(parseSource string, parseOptions ...MarkdownOption) []ui.Node {
	var parseConfig MarkdownRenderOptions
	for _, parseOption := range parseOptions {
		if parseOption != nil {
			parseOption.applyMarkdown(&parseConfig)
		}
	}
	return renderMarkdownConfig(parseSource, parseConfig)
}
