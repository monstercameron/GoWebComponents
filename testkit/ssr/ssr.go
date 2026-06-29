package ssr

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
	xhtml "golang.org/x/net/html"
)

// Snapshot captures one server-rendered HTML result.
type Snapshot struct {
	HTML string
}

type MetaTag struct {
	Name       string
	Property   string
	Content    string
	Attributes map[string]string
}

type LinkTag struct {
	Rel        string
	Href       string
	HrefLang   string
	As         string
	Attributes map[string]string
}

type ScriptTag struct {
	ID         string
	Type       string
	Content    string
	Attributes map[string]string
}

type StructuredSnapshot struct {
	Title          string
	MetaByName     map[string][]MetaTag
	MetaByProperty map[string][]MetaTag
	LinksByRel     map[string][]LinkTag
	ScriptsByID    map[string]ScriptTag
	ScriptsByType  map[string][]ScriptTag
}

type StaticExport struct {
	Root      string
	HTMLFiles map[string]Snapshot
	Bootstrap map[string][]byte
}

type ExportedRoute struct {
	Path          string
	HTMLFile      string
	BootstrapFile string
	Snapshot      Snapshot
	Bootstrap     []byte
}

// Render snapshots one UI tree through the public SSR surface.
func Render(parseTb testing.TB, parseRoot ui.Node) Snapshot {
	parseTb.Helper()
	parseMarkup, parseErr := ui.RenderToString(parseRoot)
	if parseErr != nil {
		parseTb.Fatalf("ssr.Render failed: %v", parseErr)
	}
	return Snapshot{HTML: parseMarkup}
}

// Contains reports whether the rendered HTML contains the expected substring.
func (parseS Snapshot) Contains(parseSubstring string) bool {
	return strings.Contains(parseS.HTML, parseSubstring)
}

// Structured parses the snapshot into typed head-friendly structures.
func (parseS Snapshot) Structured(parseTb testing.TB) StructuredSnapshot {
	parseTb.Helper()
	parseRoot, parseErr := xhtml.Parse(strings.NewReader("<div>" + parseS.HTML + "</div>"))
	if parseErr != nil {
		parseTb.Fatalf("ssr.Snapshot.Structured failed to parse HTML: %v", parseErr)
	}
	parseResult := StructuredSnapshot{
		MetaByName:     map[string][]MetaTag{},
		MetaByProperty: map[string][]MetaTag{},
		LinksByRel:     map[string][]LinkTag{},
		ScriptsByID:    map[string]ScriptTag{},
		ScriptsByType:  map[string][]ScriptTag{},
	}
	collectStructuredSnapshot(&parseResult, parseRoot)
	return parseResult
}

func (parseS StructuredSnapshot) MetaName(parseName string) string {
	parseItems := parseS.MetaByName[strings.TrimSpace(parseName)]
	if len(parseItems) == 0 {
		return ""
	}
	return parseItems[0].Content
}

func (parseS StructuredSnapshot) MetaProperty(parseProperty string) string {
	parseItems := parseS.MetaByProperty[strings.TrimSpace(parseProperty)]
	if len(parseItems) == 0 {
		return ""
	}
	return parseItems[0].Content
}

func (parseS StructuredSnapshot) CanonicalURL() string {
	parseItems := parseS.LinksByRel["canonical"]
	if len(parseItems) == 0 {
		return ""
	}
	return parseItems[0].Href
}

func (parseS StructuredSnapshot) JSONLD(parseId string) string {
	if parseScript, parseOk := parseS.ScriptsByID[strings.TrimSpace(parseId)]; parseOk {
		return parseScript.Content
	}
	parseItems := parseS.ScriptsByType["application/ld+json"]
	if len(parseItems) == 0 {
		return ""
	}
	return parseItems[0].Content
}

// ApplyStructuredTitle asserts one parsed title value.
func (parseS StructuredSnapshot) ApplyStructuredTitle(parseTb testing.TB, parseExpected string) {
	parseTb.Helper()
	if parseS.Title != parseExpected {
		parseTb.Fatalf("ssr.StructuredSnapshot title mismatch: expected %q, got %q", parseExpected, parseS.Title)
	}
}

// ApplyStructuredMetaName asserts one parsed meta-name value.
func (parseS StructuredSnapshot) ApplyStructuredMetaName(parseTb testing.TB, parseName string, parseExpected string) {
	parseTb.Helper()
	parseGot := parseS.MetaName(parseName)
	if parseGot != parseExpected {
		parseTb.Fatalf("ssr.StructuredSnapshot meta[name=%q] mismatch: expected %q, got %q", parseName, parseExpected, parseGot)
	}
}

// ApplyStructuredMetaProperty asserts one parsed meta-property value.
func (parseS StructuredSnapshot) ApplyStructuredMetaProperty(parseTb testing.TB, parseProperty string, parseExpected string) {
	parseTb.Helper()
	parseGot := parseS.MetaProperty(parseProperty)
	if parseGot != parseExpected {
		parseTb.Fatalf("ssr.StructuredSnapshot meta[property=%q] mismatch: expected %q, got %q", parseProperty, parseExpected, parseGot)
	}
}

// ApplyStructuredCanonicalURL asserts one parsed canonical URL value.
func (parseS StructuredSnapshot) ApplyStructuredCanonicalURL(parseTb testing.TB, parseExpected string) {
	parseTb.Helper()
	parseGot := parseS.CanonicalURL()
	if parseGot != parseExpected {
		parseTb.Fatalf("ssr.StructuredSnapshot canonical mismatch: expected %q, got %q", parseExpected, parseGot)
	}
}

// ApplyStructuredScriptID asserts one parsed script id is present and returns that script.
func (parseS StructuredSnapshot) ApplyStructuredScriptID(parseTb testing.TB, parseId string) ScriptTag {
	parseTb.Helper()
	parseTrimmed := strings.TrimSpace(parseId)
	if parseTrimmed == "" {
		parseTb.Fatal("ssr.StructuredSnapshot.ApplyStructuredScriptID requires a script id")
	}
	parseScript, parseOk := parseS.ScriptsByID[parseTrimmed]
	if !parseOk {
		parseTb.Fatalf("ssr.StructuredSnapshot missing script id %q", parseTrimmed)
	}
	return parseScript
}

// ParseStructuredJSONLD decodes one JSON-LD script into a typed map.
func (parseS StructuredSnapshot) ParseStructuredJSONLD(parseTb testing.TB, parseId string) map[string]any {
	parseTb.Helper()
	parsePayload := strings.TrimSpace(parseS.JSONLD(parseId))
	if parsePayload == "" {
		parseTb.Fatalf("ssr.StructuredSnapshot missing JSON-LD payload for id %q", strings.TrimSpace(parseId))
	}
	return parseStructuredJSONObject(parseTb, parsePayload, "jsonld:"+strings.TrimSpace(parseId))
}

// ApplyStructuredJSONLDType asserts one JSON-LD script has the expected `@type` value.
func (parseS StructuredSnapshot) ApplyStructuredJSONLDType(parseTb testing.TB, parseId string, parseExpected string) {
	parseTb.Helper()
	parseDoc := parseS.ParseStructuredJSONLD(parseTb, parseId)
	parseGot, _ := parseDoc["@type"].(string)
	if parseGot != parseExpected {
		parseTb.Fatalf("ssr.StructuredSnapshot JSON-LD @type mismatch for id %q: expected %q, got %q", strings.TrimSpace(parseId), parseExpected, parseGot)
	}
}

// ParseStructuredBootstrapScript decodes one inline bootstrap script into a typed map.
func (parseS StructuredSnapshot) ParseStructuredBootstrapScript(parseTb testing.TB, parseId string) map[string]any {
	parseTb.Helper()
	parseScript := parseS.ApplyStructuredScriptID(parseTb, parseId)
	if strings.TrimSpace(parseScript.Type) != "application/json" {
		parseTb.Fatalf("ssr.StructuredSnapshot bootstrap script %q expected type application/json, got %q", strings.TrimSpace(parseId), parseScript.Type)
	}
	parsePayload := strings.TrimSpace(parseScript.Content)
	if parsePayload == "" {
		parseTb.Fatalf("ssr.StructuredSnapshot bootstrap script %q is empty", strings.TrimSpace(parseId))
	}
	return parseStructuredJSONObject(parseTb, parsePayload, "bootstrap:"+strings.TrimSpace(parseId))
}

// parseStructuredJSONObject decodes one JSON object string for structured assertions.
func parseStructuredJSONObject(parseTb testing.TB, parsePayload string, parseLabel string) map[string]any {
	parseTb.Helper()
	parseDecoded := map[string]any{}
	if parseErr := json.Unmarshal([]byte(parsePayload), &parseDecoded); parseErr != nil {
		parseTb.Fatalf("ssr.StructuredSnapshot failed to decode %s JSON object: %v (payload=%q)", strings.TrimSpace(parseLabel), parseErr, parsePayload)
	}
	return parseDecoded
}

// RequirePayload reads one typed bootstrap payload entry and fails the test if it is missing.
func RequirePayload[T any](parseTb testing.TB, parseBootstrap ui.SSRBootstrap, parseKey string) ui.SSRPayloadValue[T] {
	parseTb.Helper()
	parseValue, parseOk, parseErr := ui.ReadBootstrapPayload[T](parseBootstrap, parseKey)
	if parseErr != nil {
		parseTb.Fatalf("ssr.RequirePayload failed for key %q: %v", parseKey, parseErr)
	}
	if !parseOk {
		parseTb.Fatalf("ssr.RequirePayload could not find key %q", parseKey)
	}
	return parseValue
}

// LoadStaticExport reads one prerendered output directory into structured HTML and bootstrap maps.
func LoadStaticExport(parseTb testing.TB, parseOutputDir string) StaticExport {
	parseTb.Helper()
	parseRoot := strings.TrimSpace(parseOutputDir)
	if parseRoot == "" {
		parseTb.Fatal("ssr.LoadStaticExport requires an output directory")
	}
	parseExport := StaticExport{
		Root:      parseRoot,
		HTMLFiles: map[string]Snapshot{},
		Bootstrap: map[string][]byte{},
	}
	if parseErr := collectStaticExportFiles(parseRoot, "", &parseExport); parseErr != nil {
		parseTb.Fatalf("ssr.LoadStaticExport failed: %v", parseErr)
	}
	return parseExport
}

// Route resolves one route path into its emitted HTML file and optional bootstrap sidecar.
func (parseE StaticExport) Route(parseRoutePath string) (ExportedRoute, error) {
	parseNormalized, parseErr := normalizeStaticRoutePath(parseRoutePath)
	if parseErr != nil {
		return ExportedRoute{}, parseErr
	}
	parseHtmlFile := staticHTMLFile(parseNormalized)
	parseSnapshot, parseOk := parseE.HTMLFiles[parseHtmlFile]
	if !parseOk {
		return ExportedRoute{}, fmt.Errorf("ssr.StaticExport route %q missing html file %q", parseNormalized, parseHtmlFile)
	}
	parseResult := ExportedRoute{
		Path:     parseNormalized,
		HTMLFile: parseHtmlFile,
		Snapshot: parseSnapshot,
	}
	for parseFile, parseData := range parseE.Bootstrap {
		if staticBootstrapMatches(parseNormalized, parseFile) {
			parseResult.BootstrapFile = parseFile
			parseResult.Bootstrap = append([]byte(nil), parseData...)
			break
		}
	}
	return parseResult, nil
}

func collectStructuredSnapshot(parseResult *StructuredSnapshot, parseNode *xhtml.Node) {
	if parseNode == nil {
		return
	}
	if parseNode.Type == xhtml.ElementNode {
		parseAttrs := htmlAttributes(parseNode)
		switch parseNode.Data {
		case "title":
			parseResult.Title = strings.TrimSpace(nodeText(parseNode))
		case "meta":
			parseTag := MetaTag{
				Name:       parseAttrs["name"],
				Property:   parseAttrs["property"],
				Content:    parseAttrs["content"],
				Attributes: parseAttrs,
			}
			if parseTag.Name != "" {
				parseResult.MetaByName[parseTag.Name] = append(parseResult.MetaByName[parseTag.Name], parseTag)
			}
			if parseTag.Property != "" {
				parseResult.MetaByProperty[parseTag.Property] = append(parseResult.MetaByProperty[parseTag.Property], parseTag)
			}
		case "link":
			parseTag2 := LinkTag{
				Rel:        parseAttrs["rel"],
				Href:       parseAttrs["href"],
				HrefLang:   parseAttrs["hreflang"],
				As:         parseAttrs["as"],
				Attributes: parseAttrs,
			}
			if parseTag2.Rel != "" {
				parseResult.LinksByRel[parseTag2.Rel] = append(parseResult.LinksByRel[parseTag2.Rel], parseTag2)
			}
		case "script":
			parseTag3 := ScriptTag{
				ID:         parseAttrs["id"],
				Type:       parseAttrs["type"],
				Content:    nodeText(parseNode),
				Attributes: parseAttrs,
			}
			if parseTag3.ID != "" {
				parseResult.ScriptsByID[parseTag3.ID] = parseTag3
			}
			if parseTag3.Type != "" {
				parseResult.ScriptsByType[parseTag3.Type] = append(parseResult.ScriptsByType[parseTag3.Type], parseTag3)
			}
		}
	}
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		collectStructuredSnapshot(parseResult, parseChild)
	}
}

func htmlAttributes(parseNode *xhtml.Node) map[string]string {
	parseAttrs := make(map[string]string, len(parseNode.Attr))
	for _, parseAttr := range parseNode.Attr {
		parseAttrs[parseAttr.Key] = parseAttr.Val
	}
	return parseAttrs
}

func nodeText(parseNode *xhtml.Node) string {
	if parseNode == nil {
		return ""
	}
	if parseNode.Type == xhtml.TextNode {
		return parseNode.Data
	}
	var parseBuilder strings.Builder
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		parseBuilder.WriteString(nodeText(parseChild))
	}
	return parseBuilder.String()
}

func normalizeStaticRoutePath(parsePath string) (string, error) {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		parseTrimmed = "/"
	}
	if !strings.HasPrefix(parseTrimmed, "/") {
		return "", fmt.Errorf("ssr.StaticExport route %q must start with '/'", parsePath)
	}
	if parseTrimmed != "/" {
		parseTrimmed = strings.TrimRight(parseTrimmed, "/")
	}
	return parseTrimmed, nil
}

func staticHTMLFile(parseRoutePath string) string {
	parseTrimmed := strings.Trim(parseRoutePath, "/")
	if parseTrimmed == "" {
		return "index.html"
	}
	return filepath.ToSlash(filepath.Join(parseTrimmed, "index.html"))
}

func staticBootstrapMatches(parseRoutePath, parseFile string) bool {
	parseTrimmed := strings.Trim(parseRoutePath, "/")
	if parseTrimmed == "" {
		parseTrimmed = "index"
	}
	parseExpectedPrefix := filepath.ToSlash(filepath.Join("bootstrap", parseTrimmed))
	return strings.HasPrefix(parseFile, parseExpectedPrefix+".")
}
