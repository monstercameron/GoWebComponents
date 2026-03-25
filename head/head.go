package head

import (
	"encoding/json"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// SocialMetadata describes common Open Graph and Twitter card metadata.
type SocialMetadata struct {
	Type        string
	Title       string
	Description string
	ImageURL    string
	URL         string
	TwitterCard string
}

// AlternateLink describes one alternate route or locale URL.
type AlternateLink struct {
	Href     string
	HrefLang string
	Media    string
	Type     string
	Title    string
}

// ResourceHint describes one explicit resource hint link tag.
type ResourceHint struct {
	Rel         string
	Href        string
	As          string
	CrossOrigin string
	Type        string
	Media       string
}

// JSONLDBlock describes one JSON-LD script block.
type JSONLDBlock struct {
	ID    string
	Value interface{}
}

// Document describes a higher-level SSR head bundle layered on router metadata.
type Document struct {
	Metadata      router.Metadata
	Robots        string
	Social        SocialMetadata
	Alternates    []AlternateLink
	ResourceHints []ResourceHint
	JSONLD        []JSONLDBlock
	Extras        []ui.Node
}

// MergeOptions controls how non-router metadata is composed across route layers.
type MergeOptions struct {
	ClearRobots          bool
	ClearSocial          bool
	ReplaceAlternates    bool
	ReplaceResourceHints bool
	ReplaceJSONLD        bool
	ReplaceExtras        bool
}

// RouteLayer describes one route-owned head contribution plus its merge behavior.
type RouteLayer struct {
	Document Document
	Merge    MergeOptions
}

// Compose combines router-managed metadata with optional explicit head tags.
func Compose(metadata router.Metadata, extras ...ui.Node) ui.Node {
	children := make([]ui.Node, 0, 1+len(extras))
	children = append(children, router.MetadataNode(metadata))
	for _, extra := range extras {
		if extra != nil {
			children = append(children, extra)
		}
	}
	return ui.Fragment(children...)
}

// MetaName renders a meta tag with a name-based attribute.
func MetaName(name, content string) ui.Node {
	name = strings.TrimSpace(name)
	content = strings.TrimSpace(content)
	if name == "" || content == "" {
		return nil
	}
	return html.Meta(html.Props{Raw: map[string]interface{}{
		"name":    name,
		"content": content,
	}})
}

// MetaProperty renders a meta tag with a property-based attribute.
func MetaProperty(property, content string) ui.Node {
	property = strings.TrimSpace(property)
	content = strings.TrimSpace(content)
	if property == "" || content == "" {
		return nil
	}
	return html.Meta(html.Props{Raw: map[string]interface{}{
		"property": property,
		"content":  content,
	}})
}

// LinkRel renders a link tag for a relationship and target URL.
func LinkRel(rel, href string) ui.Node {
	rel = strings.TrimSpace(rel)
	href = strings.TrimSpace(href)
	if rel == "" || href == "" {
		return nil
	}
	return html.Link(html.Props{Raw: map[string]interface{}{
		"rel":  rel,
		"href": href,
	}})
}

// Hreflang renders an alternate locale link tag.
func Hreflang(hrefLang, href string) ui.Node {
	hrefLang = strings.TrimSpace(hrefLang)
	href = strings.TrimSpace(href)
	if hrefLang == "" || href == "" {
		return nil
	}
	return html.Link(html.Props{Raw: map[string]interface{}{
		"rel":      "alternate",
		"hreflang": hrefLang,
		"href":     href,
	}})
}

// Robots renders a robots meta tag.
func Robots(content string) ui.Node {
	return MetaName("robots", content)
}

// OpenGraph renders an Open Graph meta tag for the given field.
func OpenGraph(field, content string) ui.Node {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil
	}
	return MetaProperty("og:"+field, content)
}

// Twitter renders a Twitter/X card meta tag for the given field.
func Twitter(field, content string) ui.Node {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil
	}
	return MetaName("twitter:"+field, content)
}

// SocialTags renders a small common set of Open Graph and Twitter card tags.
func SocialTags(metadata SocialMetadata) ui.Node {
	card := strings.TrimSpace(metadata.TwitterCard)
	if card == "" && strings.TrimSpace(metadata.ImageURL) != "" {
		card = "summary_large_image"
	}
	return fragmentNonNil(
		OpenGraph("type", metadata.Type),
		OpenGraph("title", metadata.Title),
		OpenGraph("description", metadata.Description),
		OpenGraph("image", metadata.ImageURL),
		OpenGraph("url", metadata.URL),
		Twitter("card", card),
		Twitter("title", metadata.Title),
		Twitter("description", metadata.Description),
		Twitter("image", metadata.ImageURL),
	)
}

// AlternateLinks renders alternate and hreflang link tags.
func AlternateLinks(links ...AlternateLink) ui.Node {
	children := make([]ui.Node, 0, len(links))
	for _, link := range links {
		if node := alternateLinkNode(link); node != nil {
			children = append(children, node)
		}
	}
	return fragmentNonNil(children...)
}

// ResourceHints renders a bundle of explicit resource hint link tags.
func ResourceHints(hints ...ResourceHint) ui.Node {
	children := make([]ui.Node, 0, len(hints))
	for _, hint := range hints {
		if node := resourceHintNode(hint); node != nil {
			children = append(children, node)
		}
	}
	return fragmentNonNil(children...)
}

// RenderJSONLD renders one JSON-LD script block for direct insertion into SSR head markup.
func RenderJSONLD(value interface{}, scriptID string) (string, error) {
	if value == nil {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	scriptID = strings.TrimSpace(scriptID)
	if scriptID == "" {
		return `<script type="application/ld+json">` + escapeJSONForInlineScript(string(encoded)) + `</script>`, nil
	}
	return `<script id="` + escapeHTMLAttribute(scriptID) + `" type="application/ld+json">` + escapeJSONForInlineScript(string(encoded)) + `</script>`, nil
}

// RenderToString emits a full SSR head fragment using router-managed metadata plus companion-owned helpers.
func RenderToString(document Document) (string, error) {
	extras := make([]ui.Node, 0, len(document.Extras)+3)
	extras = append(extras,
		Robots(document.Robots),
		SocialTags(document.Social),
		AlternateLinks(document.Alternates...),
		ResourceHints(document.ResourceHints...),
	)
	extras = append(extras, document.Extras...)

	markup, err := ui.RenderToString(Compose(document.Metadata, extras...))
	if err != nil {
		return "", err
	}

	if len(document.JSONLD) == 0 {
		return markup, nil
	}

	var builder strings.Builder
	builder.WriteString(markup)
	for _, block := range document.JSONLD {
		script, renderErr := RenderJSONLD(block.Value, block.ID)
		if renderErr != nil {
			return "", renderErr
		}
		builder.WriteString(script)
	}
	return builder.String(), nil
}

// Merge combines a base head document with an override document.
func Merge(base, override Document, options ...MergeOptions) Document {
	opts := MergeOptions{}
	if len(options) > 0 {
		opts = options[0]
	}

	resolved := cloneDocument(base)
	resolved.Metadata = mergeMetadata(base.Metadata, override.Metadata)
	resolved.Robots = mergeRobots(base.Robots, override.Robots, opts)
	resolved.Social = mergeSocial(base.Social, override.Social, opts)
	resolved.Alternates = mergeAlternates(base.Alternates, override.Alternates, opts)
	resolved.ResourceHints = mergeResourceHints(base.ResourceHints, override.ResourceHints, opts)
	resolved.JSONLD = mergeJSONLD(base.JSONLD, override.JSONLD, opts)
	resolved.Extras = mergeExtras(base.Extras, override.Extras, opts)
	return resolved
}

// Resolve folds route layers from parent defaults to leaf overrides.
func Resolve(layers ...RouteLayer) Document {
	if len(layers) == 0 {
		return Document{}
	}

	resolved := cloneDocument(layers[0].Document)
	for i := 1; i < len(layers); i++ {
		resolved = Merge(resolved, layers[i].Document, layers[i].Merge)
	}
	return resolved
}

func alternateLinkNode(link AlternateLink) ui.Node {
	href := strings.TrimSpace(link.Href)
	if href == "" {
		return nil
	}

	raw := map[string]interface{}{
		"rel":  "alternate",
		"href": href,
	}
	if hrefLang := strings.TrimSpace(link.HrefLang); hrefLang != "" {
		raw["hreflang"] = hrefLang
	}
	if media := strings.TrimSpace(link.Media); media != "" {
		raw["media"] = media
	}
	if typ := strings.TrimSpace(link.Type); typ != "" {
		raw["type"] = typ
	}
	if title := strings.TrimSpace(link.Title); title != "" {
		raw["title"] = title
	}
	return html.Link(html.Props{Raw: raw})
}

func resourceHintNode(hint ResourceHint) ui.Node {
	rel := strings.TrimSpace(hint.Rel)
	href := strings.TrimSpace(hint.Href)
	if rel == "" || href == "" {
		return nil
	}

	raw := map[string]interface{}{
		"rel":  rel,
		"href": href,
	}
	if as := strings.TrimSpace(hint.As); as != "" {
		raw["as"] = as
	}
	if crossOrigin := strings.TrimSpace(hint.CrossOrigin); crossOrigin != "" {
		raw["crossorigin"] = crossOrigin
	}
	if typ := strings.TrimSpace(hint.Type); typ != "" {
		raw["type"] = typ
	}
	if media := strings.TrimSpace(hint.Media); media != "" {
		raw["media"] = media
	}
	return html.Link(html.Props{Raw: raw})
}

func escapeJSONForInlineScript(text string) string {
	replacer := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
		"\u2028", `\u2028`,
		"\u2029", `\u2029`,
	)
	return replacer.Replace(text)
}

func escapeHTMLAttribute(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(text)
}

func cloneDocument(document Document) Document {
	clone := document
	clone.Alternates = append([]AlternateLink(nil), document.Alternates...)
	clone.ResourceHints = append([]ResourceHint(nil), document.ResourceHints...)
	clone.JSONLD = append([]JSONLDBlock(nil), document.JSONLD...)
	clone.Extras = append([]ui.Node(nil), document.Extras...)
	return clone
}

func mergeMetadata(base, override router.Metadata) router.Metadata {
	return router.Metadata{
		Title:        mergeNonEmpty(base.Title, override.Title),
		Description:  mergeNonEmpty(base.Description, override.Description),
		CanonicalURL: mergeNonEmpty(base.CanonicalURL, override.CanonicalURL),
	}
}

func mergeRobots(base, override string, options MergeOptions) string {
	if options.ClearRobots {
		base = ""
	}
	return mergeNonEmpty(base, override)
}

func mergeSocial(base, override SocialMetadata, options MergeOptions) SocialMetadata {
	if options.ClearSocial {
		base = SocialMetadata{}
	}
	return SocialMetadata{
		Type:        mergeNonEmpty(base.Type, override.Type),
		Title:       mergeNonEmpty(base.Title, override.Title),
		Description: mergeNonEmpty(base.Description, override.Description),
		ImageURL:    mergeNonEmpty(base.ImageURL, override.ImageURL),
		URL:         mergeNonEmpty(base.URL, override.URL),
		TwitterCard: mergeNonEmpty(base.TwitterCard, override.TwitterCard),
	}
}

func mergeAlternates(base, override []AlternateLink, options MergeOptions) []AlternateLink {
	if options.ReplaceAlternates {
		return append([]AlternateLink(nil), override...)
	}
	merged := append([]AlternateLink(nil), base...)
	return append(merged, override...)
}

func mergeResourceHints(base, override []ResourceHint, options MergeOptions) []ResourceHint {
	if options.ReplaceResourceHints {
		return append([]ResourceHint(nil), override...)
	}
	merged := append([]ResourceHint(nil), base...)
	return append(merged, override...)
}

func mergeJSONLD(base, override []JSONLDBlock, options MergeOptions) []JSONLDBlock {
	if options.ReplaceJSONLD {
		return append([]JSONLDBlock(nil), override...)
	}
	merged := append([]JSONLDBlock(nil), base...)
	return append(merged, override...)
}

func mergeExtras(base, override []ui.Node, options MergeOptions) []ui.Node {
	if options.ReplaceExtras {
		return append([]ui.Node(nil), override...)
	}
	merged := append([]ui.Node(nil), base...)
	return append(merged, override...)
}

func mergeNonEmpty(base, override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	return base
}

func fragmentNonNil(children ...ui.Node) ui.Node {
	filtered := make([]ui.Node, 0, len(children))
	for _, child := range children {
		if child != nil {
			filtered = append(filtered, child)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return ui.Fragment(filtered...)
}
