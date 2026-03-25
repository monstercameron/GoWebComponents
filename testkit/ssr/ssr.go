package ssr

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
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
func Render(tb testing.TB, root ui.Node) Snapshot {
	tb.Helper()
	markup, err := ui.RenderToString(root)
	if err != nil {
		tb.Fatalf("ssr.Render failed: %v", err)
	}
	return Snapshot{HTML: markup}
}

// Contains reports whether the rendered HTML contains the expected substring.
func (s Snapshot) Contains(substring string) bool {
	return strings.Contains(s.HTML, substring)
}

// Structured parses the snapshot into typed head-friendly structures.
func (s Snapshot) Structured(tb testing.TB) StructuredSnapshot {
	tb.Helper()
	root, err := xhtml.Parse(strings.NewReader("<div>" + s.HTML + "</div>"))
	if err != nil {
		tb.Fatalf("ssr.Snapshot.Structured failed to parse HTML: %v", err)
	}
	result := StructuredSnapshot{
		MetaByName:     map[string][]MetaTag{},
		MetaByProperty: map[string][]MetaTag{},
		LinksByRel:     map[string][]LinkTag{},
		ScriptsByID:    map[string]ScriptTag{},
		ScriptsByType:  map[string][]ScriptTag{},
	}
	collectStructuredSnapshot(&result, root)
	return result
}

func (s StructuredSnapshot) MetaName(name string) string {
	items := s.MetaByName[strings.TrimSpace(name)]
	if len(items) == 0 {
		return ""
	}
	return items[0].Content
}

func (s StructuredSnapshot) MetaProperty(property string) string {
	items := s.MetaByProperty[strings.TrimSpace(property)]
	if len(items) == 0 {
		return ""
	}
	return items[0].Content
}

func (s StructuredSnapshot) CanonicalURL() string {
	items := s.LinksByRel["canonical"]
	if len(items) == 0 {
		return ""
	}
	return items[0].Href
}

func (s StructuredSnapshot) JSONLD(id string) string {
	if script, ok := s.ScriptsByID[strings.TrimSpace(id)]; ok {
		return script.Content
	}
	items := s.ScriptsByType["application/ld+json"]
	if len(items) == 0 {
		return ""
	}
	return items[0].Content
}

// ApplyStructuredTitle asserts one parsed title value.
func (s StructuredSnapshot) ApplyStructuredTitle(tb testing.TB, expected string) {
	tb.Helper()
	if s.Title != expected {
		tb.Fatalf("ssr.StructuredSnapshot title mismatch: expected %q, got %q", expected, s.Title)
	}
}

// ApplyStructuredMetaName asserts one parsed meta-name value.
func (s StructuredSnapshot) ApplyStructuredMetaName(tb testing.TB, name string, expected string) {
	tb.Helper()
	got := s.MetaName(name)
	if got != expected {
		tb.Fatalf("ssr.StructuredSnapshot meta[name=%q] mismatch: expected %q, got %q", name, expected, got)
	}
}

// ApplyStructuredMetaProperty asserts one parsed meta-property value.
func (s StructuredSnapshot) ApplyStructuredMetaProperty(tb testing.TB, property string, expected string) {
	tb.Helper()
	got := s.MetaProperty(property)
	if got != expected {
		tb.Fatalf("ssr.StructuredSnapshot meta[property=%q] mismatch: expected %q, got %q", property, expected, got)
	}
}

// ApplyStructuredCanonicalURL asserts one parsed canonical URL value.
func (s StructuredSnapshot) ApplyStructuredCanonicalURL(tb testing.TB, expected string) {
	tb.Helper()
	got := s.CanonicalURL()
	if got != expected {
		tb.Fatalf("ssr.StructuredSnapshot canonical mismatch: expected %q, got %q", expected, got)
	}
}

// ApplyStructuredScriptID asserts one parsed script id is present and returns that script.
func (s StructuredSnapshot) ApplyStructuredScriptID(tb testing.TB, id string) ScriptTag {
	tb.Helper()
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		tb.Fatal("ssr.StructuredSnapshot.ApplyStructuredScriptID requires a script id")
	}
	script, ok := s.ScriptsByID[trimmed]
	if !ok {
		tb.Fatalf("ssr.StructuredSnapshot missing script id %q", trimmed)
	}
	return script
}

// ParseStructuredJSONLD decodes one JSON-LD script into a typed map.
func (s StructuredSnapshot) ParseStructuredJSONLD(tb testing.TB, id string) map[string]any {
	tb.Helper()
	payload := strings.TrimSpace(s.JSONLD(id))
	if payload == "" {
		tb.Fatalf("ssr.StructuredSnapshot missing JSON-LD payload for id %q", strings.TrimSpace(id))
	}
	return parseStructuredJSONObject(tb, payload, "jsonld:"+strings.TrimSpace(id))
}

// ApplyStructuredJSONLDType asserts one JSON-LD script has the expected `@type` value.
func (s StructuredSnapshot) ApplyStructuredJSONLDType(tb testing.TB, id string, expected string) {
	tb.Helper()
	doc := s.ParseStructuredJSONLD(tb, id)
	got, _ := doc["@type"].(string)
	if got != expected {
		tb.Fatalf("ssr.StructuredSnapshot JSON-LD @type mismatch for id %q: expected %q, got %q", strings.TrimSpace(id), expected, got)
	}
}

// ParseStructuredBootstrapScript decodes one inline bootstrap script into a typed map.
func (s StructuredSnapshot) ParseStructuredBootstrapScript(tb testing.TB, id string) map[string]any {
	tb.Helper()
	script := s.ApplyStructuredScriptID(tb, id)
	if strings.TrimSpace(script.Type) != "application/json" {
		tb.Fatalf("ssr.StructuredSnapshot bootstrap script %q expected type application/json, got %q", strings.TrimSpace(id), script.Type)
	}
	payload := strings.TrimSpace(script.Content)
	if payload == "" {
		tb.Fatalf("ssr.StructuredSnapshot bootstrap script %q is empty", strings.TrimSpace(id))
	}
	return parseStructuredJSONObject(tb, payload, "bootstrap:"+strings.TrimSpace(id))
}

// parseStructuredJSONObject decodes one JSON object string for structured assertions.
func parseStructuredJSONObject(tb testing.TB, payload string, label string) map[string]any {
	tb.Helper()
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		tb.Fatalf("ssr.StructuredSnapshot failed to decode %s JSON object: %v (payload=%q)", strings.TrimSpace(label), err, payload)
	}
	return decoded
}

// RequirePayload reads one typed bootstrap payload entry and fails the test if it is missing.
func RequirePayload[T any](tb testing.TB, bootstrap ui.SSRBootstrap, key string) ui.SSRPayloadValue[T] {
	tb.Helper()
	value, ok, err := ui.ReadBootstrapPayload[T](bootstrap, key)
	if err != nil {
		tb.Fatalf("ssr.RequirePayload failed for key %q: %v", key, err)
	}
	if !ok {
		tb.Fatalf("ssr.RequirePayload could not find key %q", key)
	}
	return value
}

// LoadStaticExport reads one prerendered output directory into structured HTML and bootstrap maps.
func LoadStaticExport(tb testing.TB, outputDir string) StaticExport {
	tb.Helper()
	root := strings.TrimSpace(outputDir)
	if root == "" {
		tb.Fatal("ssr.LoadStaticExport requires an output directory")
	}
	export := StaticExport{
		Root:      root,
		HTMLFiles: map[string]Snapshot{},
		Bootstrap: map[string][]byte{},
	}
	if err := collectStaticExportFiles(root, "", &export); err != nil {
		tb.Fatalf("ssr.LoadStaticExport failed: %v", err)
	}
	return export
}

// Route resolves one route path into its emitted HTML file and optional bootstrap sidecar.
func (e StaticExport) Route(routePath string) (ExportedRoute, error) {
	normalized, err := normalizeStaticRoutePath(routePath)
	if err != nil {
		return ExportedRoute{}, err
	}
	htmlFile := staticHTMLFile(normalized)
	snapshot, ok := e.HTMLFiles[htmlFile]
	if !ok {
		return ExportedRoute{}, fmt.Errorf("ssr.StaticExport route %q missing html file %q", normalized, htmlFile)
	}
	result := ExportedRoute{
		Path:     normalized,
		HTMLFile: htmlFile,
		Snapshot: snapshot,
	}
	for file, data := range e.Bootstrap {
		if staticBootstrapMatches(normalized, file) {
			result.BootstrapFile = file
			result.Bootstrap = append([]byte(nil), data...)
			break
		}
	}
	return result, nil
}

func collectStructuredSnapshot(result *StructuredSnapshot, node *xhtml.Node) {
	if node == nil {
		return
	}
	if node.Type == xhtml.ElementNode {
		attrs := htmlAttributes(node)
		switch node.Data {
		case "title":
			result.Title = strings.TrimSpace(nodeText(node))
		case "meta":
			tag := MetaTag{
				Name:       attrs["name"],
				Property:   attrs["property"],
				Content:    attrs["content"],
				Attributes: attrs,
			}
			if tag.Name != "" {
				result.MetaByName[tag.Name] = append(result.MetaByName[tag.Name], tag)
			}
			if tag.Property != "" {
				result.MetaByProperty[tag.Property] = append(result.MetaByProperty[tag.Property], tag)
			}
		case "link":
			tag := LinkTag{
				Rel:        attrs["rel"],
				Href:       attrs["href"],
				HrefLang:   attrs["hreflang"],
				As:         attrs["as"],
				Attributes: attrs,
			}
			if tag.Rel != "" {
				result.LinksByRel[tag.Rel] = append(result.LinksByRel[tag.Rel], tag)
			}
		case "script":
			tag := ScriptTag{
				ID:         attrs["id"],
				Type:       attrs["type"],
				Content:    nodeText(node),
				Attributes: attrs,
			}
			if tag.ID != "" {
				result.ScriptsByID[tag.ID] = tag
			}
			if tag.Type != "" {
				result.ScriptsByType[tag.Type] = append(result.ScriptsByType[tag.Type], tag)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectStructuredSnapshot(result, child)
	}
}

func htmlAttributes(node *xhtml.Node) map[string]string {
	attrs := make(map[string]string, len(node.Attr))
	for _, attr := range node.Attr {
		attrs[attr.Key] = attr.Val
	}
	return attrs
}

func nodeText(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == xhtml.TextNode {
		return node.Data
	}
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(nodeText(child))
	}
	return builder.String()
}

func normalizeStaticRoutePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("ssr.StaticExport route %q must start with '/'", path)
	}
	if trimmed != "/" {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed, nil
}

func staticHTMLFile(routePath string) string {
	trimmed := strings.Trim(routePath, "/")
	if trimmed == "" {
		return "index.html"
	}
	return filepath.ToSlash(filepath.Join(trimmed, "index.html"))
}

func staticBootstrapMatches(routePath, file string) bool {
	trimmed := strings.Trim(routePath, "/")
	if trimmed == "" {
		trimmed = "index"
	}
	expectedPrefix := filepath.ToSlash(filepath.Join("bootstrap", trimmed))
	return strings.HasPrefix(file, expectedPrefix+".")
}
