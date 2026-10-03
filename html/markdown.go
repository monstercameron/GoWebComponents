package html

import (
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/internal/errcode"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// MarkdownClasses configures optional class names applied to rendered markdown nodes.
type MarkdownClasses struct {
	Heading1           string
	Heading2           string
	Heading3           string
	Heading4           string
	Heading5           string
	Heading6           string
	Paragraph          string
	Blockquote         string
	List               string
	OrderedList        string
	UnorderedList      string
	ListItem           string
	CodeBlockContainer string
	CodeBlockLabel     string
	CodeBlockPre       string
	InlineCode         string
	Strong             string
	Emphasis           string
	Link               string
	HorizontalRule     string
}

// MarkdownRenderOptions configures markdown rendering into ui.Node values.
type MarkdownRenderOptions struct {
	SourcePath     string
	ResolveHref    func(sourcePath, destination string) string
	Classes        MarkdownClasses
	CodeBlockLabel string
	LinkTarget     string
	LinkRel        string
	// AllowedURLSchemes overrides the scheme allowlist applied to link, image,
	// and autolink destinations. When nil the default allowlist is used
	// (http, https, mailto); relative paths, absolute paths, and fragments
	// carry no scheme and are always allowed. Any other scheme - notably
	// javascript:, data:, and vbscript: - is dropped so user-supplied markdown
	// cannot smuggle an executable URL into an href or src sink. Supply schemes
	// without the trailing colon (for example []string{"http", "https",
	// "mailto", "data"} to additionally permit inline data: images).
	AllowedURLSchemes []string

	// isSchemeOverride is set by WithURLSchemes so an explicit empty list means
	// "allow no schemes" instead of falling back to the default allowlist.
	isSchemeOverride bool
}

// defaultMarkdownSchemes is the scheme allowlist used when a caller does not
// override it. It deliberately excludes javascript:, data:, vbscript:, and
// every other active scheme.
var defaultMarkdownSchemes = map[string]bool{"http": true, "https": true, "mailto": true}

// ResolveSafeHref resolves a markdown destination against SourcePath (through
// ResolveHref when set) and then drops it when its URL scheme is not on the
// configured allowlist, so dangerous schemes never reach an href or src attribute.
// Markdown engines call it for every link and image destination.
func (parseOptions MarkdownRenderOptions) ResolveSafeHref(parseDestination string) string {
	return SanitizeMarkdownHref(resolveMarkdownHref(parseOptions, parseDestination), parseOptions.GetAllowedSchemes())
}

// ResolveMarkdownHref resolves a markdown destination against the source document path.
func ResolveMarkdownHref(parseSourcePath, parseDestination string) string {
	parseDestination = strings.TrimSpace(parseDestination)
	if parseDestination == "" {
		return ""
	}
	parseParsedDestination, parseErr := url.Parse(parseDestination)
	if parseErr != nil {
		return parseDestination
	}
	if parseParsedDestination.IsAbs() || strings.HasPrefix(parseDestination, "#") || strings.HasPrefix(parseDestination, "/") {
		return parseDestination
	}
	parseBase := strings.TrimSpace(parseSourcePath)
	if parseBase == "" {
		return parseDestination
	}
	parseParsedBase, parseErr := url.Parse(parseBase)
	if parseErr == nil && (parseParsedBase.IsAbs() || strings.HasPrefix(parseBase, "/")) {
		return parseParsedBase.ResolveReference(parseParsedDestination).String()
	}
	parseBasePath := strings.ReplaceAll(parseBase, "\\", "/")
	parseResolved := *parseParsedDestination
	parseResolved.Path = path.Clean(path.Join(path.Dir(parseBasePath), parseParsedDestination.Path))
	if strings.HasPrefix(parseBasePath, "/") && !strings.HasPrefix(parseResolved.Path, "/") {
		parseResolved.Path = "/" + parseResolved.Path
	}
	return parseResolved.String()
}

// resolveMarkdownHref is a core package helper.
func resolveMarkdownHref(parseConfig MarkdownRenderOptions, parseDestination string) string {
	if parseConfig.ResolveHref != nil {
		return parseConfig.ResolveHref(parseConfig.SourcePath, parseDestination)
	}
	return ResolveMarkdownHref(parseConfig.SourcePath, parseDestination)
}

// SanitizeMarkdownHref returns destination unchanged when it carries no scheme
// (relative, absolute-path, or fragment) or an allowed scheme, and returns ""
// otherwise. The allowed argument lists schemes without the trailing colon; a
// nil or empty list uses the default allowlist (http, https, mailto). Scheme
// detection tolerates the usual obfuscations - leading/embedded whitespace and
// control characters (java\tscript:), mixed case (JavaScript:), and a leading
// space ( javascript:) - by normalizing the scheme token before the check.
func SanitizeMarkdownHref(parseDestination string, parseAllowed []string) string {
	parseTrimmed := strings.TrimSpace(parseDestination)
	if parseTrimmed == "" {
		return ""
	}
	parseScheme, parseHasScheme := markdownURLScheme(parseTrimmed)
	if !parseHasScheme {
		return parseDestination
	}
	if markdownAllowedSchemes(parseAllowed)[normalizeMarkdownScheme(parseScheme)] {
		return parseDestination
	}
	return ""
}

// markdownURLScheme returns the scheme token of a destination (the text before
// the first ':' that precedes any '/', '?', '#', or '\\'). It reports
// hasScheme=false for relative paths, absolute paths, and fragments, which have
// no scheme separator.
func markdownURLScheme(parseDestination string) (parseScheme string, parseHasScheme bool) {
	for parseIndex := 0; parseIndex < len(parseDestination); parseIndex++ {
		switch parseDestination[parseIndex] {
		case ':':
			return parseDestination[:parseIndex], true
		case '/', '?', '#', '\\':
			return "", false
		}
	}
	return "", false
}

// normalizeMarkdownScheme lowercases a scheme token and strips whitespace and
// control characters so obfuscated schemes collapse onto their real form.
func normalizeMarkdownScheme(parseScheme string) string {
	var parseBuilder strings.Builder
	for parseIndex := 0; parseIndex < len(parseScheme); parseIndex++ {
		parseByte := parseScheme[parseIndex]
		if parseByte <= ' ' || parseByte == 0x7f {
			continue
		}
		parseBuilder.WriteByte(parseByte)
	}
	return strings.ToLower(parseBuilder.String())
}

// markdownAllowedSchemes returns the effective scheme allowlist, defaulting to
// http/https/mailto when no override is supplied.
func markdownAllowedSchemes(parseAllowed []string) map[string]bool {
	if len(parseAllowed) == 0 {
		return defaultMarkdownSchemes
	}
	parseSet := make(map[string]bool, len(parseAllowed))
	for _, parseScheme := range parseAllowed {
		parseSet[normalizeMarkdownScheme(parseScheme)] = true
	}
	return parseSet
}

// MarkdownEngine converts markdown source into nodes using a resolved
// configuration. Engines must honor MarkdownRenderOptions, route every link and
// image destination through MarkdownRenderOptions.ResolveSafeHref (autolinks
// through SanitizeMarkdownHref with GetAllowedSchemes), and never emit raw HTML.
type MarkdownEngine func(parseMarkdown string, parseConfig MarkdownRenderOptions) []ui.Node

var (
	markdownEngineMu    sync.RWMutex
	markdownEngine      MarkdownEngine
	markdownMissingOnce sync.Once
)

// RegisterMarkdownEngine installs the engine used by Markdown and RenderMarkdown.
// It is safe for concurrent use and last registration wins; passing nil removes
// the engine so rendering falls back to escaped plain text. Import
// html/markdown to register the bundled goldmark engine.
func RegisterMarkdownEngine(parseEngine MarkdownEngine) {
	markdownEngineMu.Lock()
	markdownEngine = parseEngine
	markdownEngineMu.Unlock()
}

// getMarkdownEngine returns the registered engine, or nil when none is installed.
func getMarkdownEngine() MarkdownEngine {
	markdownEngineMu.RLock()
	defer markdownEngineMu.RUnlock()
	return markdownEngine
}

// RenderMarkdown parses markdown text and returns semantic ui.Node values. It
// needs a registered engine (import html/markdown); without one the source is
// rendered as escaped plain text inside a div marked data-gwc-markdown-unavailable
// and one GWC-UI-MARKDOWN-ENGINE-MISSING warning is reported.
func RenderMarkdown(parseMarkdown string, parseOptions ...MarkdownRenderOptions) []ui.Node {
	parseConfig := MarkdownRenderOptions{}
	if len(parseOptions) > 0 {
		parseConfig = parseOptions[0]
	}
	return renderMarkdownConfig(parseMarkdown, parseConfig)
}

// renderMarkdownConfig dispatches to the registered engine or the plain-text fallback.
func renderMarkdownConfig(parseMarkdown string, parseConfig MarkdownRenderOptions) []ui.Node {
	if parseEngine := getMarkdownEngine(); parseEngine != nil {
		return parseEngine(parseMarkdown, parseConfig)
	}
	markdownMissingOnce.Do(func() {
		runtime.ReportDiagnostic("html", runtime.DiagnosticWarning, errcode.New(errcode.UIMarkdownEngineMissing,
			"html.Markdown has no engine; import _ \"github.com/monstercameron/GoWebComponents/v6/html/markdown\"").Error())
	})
	// Escaped text node only: the unregistered path must never interpret markup.
	return []ui.Node{Div(Props{Data: map[string]string{"gwc-markdown-unavailable": ""}}, Text(parseMarkdown))}
}
