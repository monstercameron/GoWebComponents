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
func Compose(parseMetadata router.Metadata, parseExtras ...ui.Node) ui.Node {
	parseChildren := make([]ui.Node, 0, 1+len(parseExtras))
	parseChildren = append(parseChildren, router.BuildMetadataNode(parseMetadata))
	for _, parseExtra := range parseExtras {
		if parseExtra != nil {
			parseChildren = append(parseChildren, parseExtra)
		}
	}
	return ui.Fragment(parseChildren...)
}

// MetaName renders a meta tag with a name-based attribute.
func MetaName(parseName, parseContent string) ui.Node {
	parseName = strings.TrimSpace(parseName)
	parseContent = strings.TrimSpace(parseContent)
	if parseName == "" || parseContent == "" {
		return nil
	}
	return html.Meta(html.Props{Raw: map[string]interface{}{
		"name":    parseName,
		"content": parseContent,
	}})
}

// MetaProperty renders a meta tag with a property-based attribute.
func MetaProperty(parseProperty, parseContent string) ui.Node {
	parseProperty = strings.TrimSpace(parseProperty)
	parseContent = strings.TrimSpace(parseContent)
	if parseProperty == "" || parseContent == "" {
		return nil
	}
	return html.Meta(html.Props{Raw: map[string]interface{}{
		"property": parseProperty,
		"content":  parseContent,
	}})
}

// LinkRel renders a link tag for a relationship and target URL.
func LinkRel(parseRel, parseHref string) ui.Node {
	parseRel = strings.TrimSpace(parseRel)
	parseHref = strings.TrimSpace(parseHref)
	if parseRel == "" || parseHref == "" {
		return nil
	}
	return html.Link(html.Props{Raw: map[string]interface{}{
		"rel":  parseRel,
		"href": parseHref,
	}})
}

// Hreflang renders an alternate locale link tag.
func Hreflang(parseHrefLang, parseHref string) ui.Node {
	parseHrefLang = strings.TrimSpace(parseHrefLang)
	parseHref = strings.TrimSpace(parseHref)
	if parseHrefLang == "" || parseHref == "" {
		return nil
	}
	return html.Link(html.Props{Raw: map[string]interface{}{
		"rel":      "alternate",
		"hreflang": parseHrefLang,
		"href":     parseHref,
	}})
}

// Robots renders a robots meta tag.
func Robots(parseContent string) ui.Node {
	return MetaName("robots", parseContent)
}

// OpenGraph renders an Open Graph meta tag for the given field.
func OpenGraph(parseField, parseContent string) ui.Node {
	parseField = strings.TrimSpace(parseField)
	if parseField == "" {
		return nil
	}
	return MetaProperty("og:"+parseField, parseContent)
}

// Twitter renders a Twitter/X card meta tag for the given field.
func Twitter(parseField, parseContent string) ui.Node {
	parseField = strings.TrimSpace(parseField)
	if parseField == "" {
		return nil
	}
	return MetaName("twitter:"+parseField, parseContent)
}

// SocialTags renders a small common set of Open Graph and Twitter card tags.
func SocialTags(parseMetadata SocialMetadata) ui.Node {
	parseCard := strings.TrimSpace(parseMetadata.TwitterCard)
	if parseCard == "" && strings.TrimSpace(parseMetadata.ImageURL) != "" {
		parseCard = "summary_large_image"
	}
	return fragmentNonNil(
		OpenGraph("type", parseMetadata.Type),
		OpenGraph("title", parseMetadata.Title),
		OpenGraph("description", parseMetadata.Description),
		OpenGraph("image", parseMetadata.ImageURL),
		OpenGraph("url", parseMetadata.URL),
		Twitter("card", parseCard),
		Twitter("title", parseMetadata.Title),
		Twitter("description", parseMetadata.Description),
		Twitter("image", parseMetadata.ImageURL),
	)
}

// AlternateLinks renders alternate and hreflang link tags.
func AlternateLinks(parseLinks ...AlternateLink) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseLinks))
	for _, parseLink := range parseLinks {
		if parseNode := alternateLinkNode(parseLink); parseNode != nil {
			parseChildren = append(parseChildren, parseNode)
		}
	}
	return fragmentNonNil(parseChildren...)
}

// ResourceHints renders a bundle of explicit resource hint link tags.
func ResourceHints(parseHints ...ResourceHint) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseHints))
	for _, parseHint := range parseHints {
		if parseNode := resourceHintNode(parseHint); parseNode != nil {
			parseChildren = append(parseChildren, parseNode)
		}
	}
	return fragmentNonNil(parseChildren...)
}

// RenderJSONLD renders one JSON-LD script block for direct insertion into SSR head markup.
func RenderJSONLD(parseValue interface{}, parseScriptID string) (string, error) {
	if parseValue == nil {
		return "", nil
	}
	parseEncoded, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return "", parseErr
	}
	parseScriptID = strings.TrimSpace(parseScriptID)
	if parseScriptID == "" {
		return `<script type="application/ld+json">` + escapeJSONForInlineScript(string(parseEncoded)) + `</script>`, nil
	}
	return `<script id="` + escapeHTMLAttribute(parseScriptID) + `" type="application/ld+json">` + escapeJSONForInlineScript(string(parseEncoded)) + `</script>`, nil
}

// RenderToString emits a full SSR head fragment using router-managed metadata plus companion-owned helpers.
func RenderToString(parseDocument Document) (string, error) {
	parseExtras := make([]ui.Node, 0, len(parseDocument.Extras)+3)
	parseExtras = append(parseExtras,
		Robots(parseDocument.Robots),
		SocialTags(parseDocument.Social),
		AlternateLinks(parseDocument.Alternates...),
		ResourceHints(parseDocument.ResourceHints...),
	)
	parseExtras = append(parseExtras, parseDocument.Extras...)

	parseMarkup, parseErr := ui.RenderToString(Compose(parseDocument.Metadata, parseExtras...))
	if parseErr != nil {
		return "", parseErr
	}

	if len(parseDocument.JSONLD) == 0 {
		return parseMarkup, nil
	}

	var parseBuilder strings.Builder
	parseBuilder.WriteString(parseMarkup)
	for _, parseBlock := range parseDocument.JSONLD {
		parseScript, renderErr := RenderJSONLD(parseBlock.Value, parseBlock.ID)
		if renderErr != nil {
			return "", renderErr
		}
		parseBuilder.WriteString(parseScript)
	}
	return parseBuilder.String(), nil
}

// Merge combines a base head document with an override document.
func Merge(parseBase, parseOverride Document, parseOptions ...MergeOptions) Document {
	parseOpts := MergeOptions{}
	if len(parseOptions) > 0 {
		parseOpts = parseOptions[0]
	}

	parseResolved := cloneDocument(parseBase)
	parseResolved.Metadata = mergeMetadata(parseBase.Metadata, parseOverride.Metadata)
	parseResolved.Robots = mergeRobots(parseBase.Robots, parseOverride.Robots, parseOpts)
	parseResolved.Social = mergeSocial(parseBase.Social, parseOverride.Social, parseOpts)
	parseResolved.Alternates = mergeAlternates(parseBase.Alternates, parseOverride.Alternates, parseOpts)
	parseResolved.ResourceHints = mergeResourceHints(parseBase.ResourceHints, parseOverride.ResourceHints, parseOpts)
	parseResolved.JSONLD = mergeJSONLD(parseBase.JSONLD, parseOverride.JSONLD, parseOpts)
	parseResolved.Extras = mergeExtras(parseBase.Extras, parseOverride.Extras, parseOpts)
	return parseResolved
}

// Resolve folds route layers from parent defaults to leaf overrides.
func Resolve(parseLayers ...RouteLayer) Document {
	if len(parseLayers) == 0 {
		return Document{}
	}

	parseResolved := cloneDocument(parseLayers[0].Document)
	for parseI := 1; parseI < len(parseLayers); parseI++ {
		parseResolved = Merge(parseResolved, parseLayers[parseI].Document, parseLayers[parseI].Merge)
	}
	return parseResolved
}

func alternateLinkNode(parseLink AlternateLink) ui.Node {
	parseHref := strings.TrimSpace(parseLink.Href)
	if parseHref == "" {
		return nil
	}

	parseRaw := map[string]interface{}{
		"rel":  "alternate",
		"href": parseHref,
	}
	if parseHrefLang := strings.TrimSpace(parseLink.HrefLang); parseHrefLang != "" {
		parseRaw["hreflang"] = parseHrefLang
	}
	if parseMedia := strings.TrimSpace(parseLink.Media); parseMedia != "" {
		parseRaw["media"] = parseMedia
	}
	if parseTyp := strings.TrimSpace(parseLink.Type); parseTyp != "" {
		parseRaw["type"] = parseTyp
	}
	if parseTitle := strings.TrimSpace(parseLink.Title); parseTitle != "" {
		parseRaw["title"] = parseTitle
	}
	return html.Link(html.Props{Raw: parseRaw})
}

func resourceHintNode(parseHint ResourceHint) ui.Node {
	parseRel := strings.TrimSpace(parseHint.Rel)
	parseHref := strings.TrimSpace(parseHint.Href)
	if parseRel == "" || parseHref == "" {
		return nil
	}

	parseRaw := map[string]interface{}{
		"rel":  parseRel,
		"href": parseHref,
	}
	if parseAs := strings.TrimSpace(parseHint.As); parseAs != "" {
		parseRaw["as"] = parseAs
	}
	if parseCrossOrigin := strings.TrimSpace(parseHint.CrossOrigin); parseCrossOrigin != "" {
		parseRaw["crossorigin"] = parseCrossOrigin
	}
	if parseTyp := strings.TrimSpace(parseHint.Type); parseTyp != "" {
		parseRaw["type"] = parseTyp
	}
	if parseMedia := strings.TrimSpace(parseHint.Media); parseMedia != "" {
		parseRaw["media"] = parseMedia
	}
	return html.Link(html.Props{Raw: parseRaw})
}

func escapeJSONForInlineScript(parseText string) string {
	parseReplacer := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
		"\u2028", `\u2028`,
		"\u2029", `\u2029`,
	)
	return parseReplacer.Replace(parseText)
}

func escapeHTMLAttribute(parseText string) string {
	parseReplacer := strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return parseReplacer.Replace(parseText)
}

func cloneDocument(parseDocument Document) Document {
	parseClone := parseDocument
	parseClone.Alternates = append([]AlternateLink(nil), parseDocument.Alternates...)
	parseClone.ResourceHints = append([]ResourceHint(nil), parseDocument.ResourceHints...)
	parseClone.JSONLD = append([]JSONLDBlock(nil), parseDocument.JSONLD...)
	parseClone.Extras = append([]ui.Node(nil), parseDocument.Extras...)
	return parseClone
}

func mergeMetadata(parseBase, parseOverride router.Metadata) router.Metadata {
	return router.Metadata{
		Title:        mergeNonEmpty(parseBase.Title, parseOverride.Title),
		Description:  mergeNonEmpty(parseBase.Description, parseOverride.Description),
		CanonicalURL: mergeNonEmpty(parseBase.CanonicalURL, parseOverride.CanonicalURL),
	}
}

func mergeRobots(parseBase, parseOverride string, parseOptions MergeOptions) string {
	if parseOptions.ClearRobots {
		parseBase = ""
	}
	return mergeNonEmpty(parseBase, parseOverride)
}

func mergeSocial(parseBase, parseOverride SocialMetadata, parseOptions MergeOptions) SocialMetadata {
	if parseOptions.ClearSocial {
		parseBase = SocialMetadata{}
	}
	return SocialMetadata{
		Type:        mergeNonEmpty(parseBase.Type, parseOverride.Type),
		Title:       mergeNonEmpty(parseBase.Title, parseOverride.Title),
		Description: mergeNonEmpty(parseBase.Description, parseOverride.Description),
		ImageURL:    mergeNonEmpty(parseBase.ImageURL, parseOverride.ImageURL),
		URL:         mergeNonEmpty(parseBase.URL, parseOverride.URL),
		TwitterCard: mergeNonEmpty(parseBase.TwitterCard, parseOverride.TwitterCard),
	}
}

func mergeAlternates(parseBase, parseOverride []AlternateLink, parseOptions MergeOptions) []AlternateLink {
	if parseOptions.ReplaceAlternates {
		return append([]AlternateLink(nil), parseOverride...)
	}
	parseMerged := append([]AlternateLink(nil), parseBase...)
	return append(parseMerged, parseOverride...)
}

func mergeResourceHints(parseBase, parseOverride []ResourceHint, parseOptions MergeOptions) []ResourceHint {
	if parseOptions.ReplaceResourceHints {
		return append([]ResourceHint(nil), parseOverride...)
	}
	parseMerged := append([]ResourceHint(nil), parseBase...)
	return append(parseMerged, parseOverride...)
}

func mergeJSONLD(parseBase, parseOverride []JSONLDBlock, parseOptions MergeOptions) []JSONLDBlock {
	if parseOptions.ReplaceJSONLD {
		return append([]JSONLDBlock(nil), parseOverride...)
	}
	parseMerged := append([]JSONLDBlock(nil), parseBase...)
	return append(parseMerged, parseOverride...)
}

func mergeExtras(parseBase, parseOverride []ui.Node, parseOptions MergeOptions) []ui.Node {
	if parseOptions.ReplaceExtras {
		return append([]ui.Node(nil), parseOverride...)
	}
	parseMerged := append([]ui.Node(nil), parseBase...)
	return append(parseMerged, parseOverride...)
}

func mergeNonEmpty(parseBase, parseOverride string) string {
	if strings.TrimSpace(parseOverride) != "" {
		return parseOverride
	}
	return parseBase
}

func fragmentNonNil(parseChildren ...ui.Node) ui.Node {
	parseFiltered := make([]ui.Node, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		if parseChild != nil {
			parseFiltered = append(parseFiltered, parseChild)
		}
	}
	if len(parseFiltered) == 0 {
		return nil
	}
	if len(parseFiltered) == 1 {
		return parseFiltered[0]
	}
	return ui.Fragment(parseFiltered...)
}
