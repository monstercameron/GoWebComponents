package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// a11yLinterName is the linter label for accessibility findings, so reports and editors group
// them and an a11y-only run is filterable.
const a11yLinterName = "gwc-a11y"

// collectLintA11yRuleIssues scans the project's HTML shells for static accessibility defects —
// images without alt text, interactive elements and form controls with no accessible name, a
// missing document language, and positive tabindex anti-patterns. This is the non-browser a11y
// gate the audit asked for (D4): it runs in `gwc lint` with no headless browser, complementing
// the browser-only axe checks. Findings carry the offending element as the Symbol so an editor
// can surface a named fix.
func collectLintA11yRuleIssues(parseRootPath string, parsePaths []string) ([]lintIssueRecord, error) {
	parseFiles, parseErr := collectLintHTMLFiles(parseRootPath)
	if parseErr != nil {
		return nil, parseErr
	}
	parseIssues := []lintIssueRecord{}
	for _, parsePath := range parseFiles {
		parseFileIssues, parseFileErr := collectA11yFileIssues(parseRootPath, parsePath)
		if parseFileErr != nil {
			continue // unparseable HTML is not an a11y finding; skip rather than fail the lint
		}
		parseIssues = append(parseIssues, parseFileIssues...)
	}
	return parseIssues, nil
}

// collectLintHTMLFiles returns every .html file under root, skipping vendored/build directories.
func collectLintHTMLFiles(parseRootPath string) ([]string, error) {
	parseFiles := []string{}
	parseErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseRootPath {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(parseEntry.Name()), ".html") {
			parseFiles = append(parseFiles, parsePath)
		}
		return nil
	})
	return parseFiles, parseErr
}

// collectA11yFileIssues parses one HTML file and applies the accessibility rules.
func collectA11yFileIssues(parseRootPath string, parsePath string) ([]lintIssueRecord, error) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, parseErr
	}
	parseContent := string(parseData)
	parseDoc, parseErr := html.Parse(strings.NewReader(parseContent))
	if parseErr != nil {
		return nil, parseErr
	}
	parseRel := relativeSlashPath(parseRootPath, parsePath)

	parseLabeledIDs, parseWrapped := collectA11yLabelTargets(parseDoc)
	parseTagSeen := map[string]int{}
	parseIssues := []lintIssueRecord{}

	var parseWalk func(parseNode *html.Node)
	parseWalk = func(parseNode *html.Node) {
		if parseNode.Type == html.ElementNode {
			parseTagSeen[parseNode.Data]++
			parseLine := locateNthTag(parseContent, parseNode.Data, parseTagSeen[parseNode.Data])
			parseIssues = append(parseIssues, evaluateA11yNode(parseNode, parseRel, parseLine, parseLabeledIDs, parseWrapped)...)
		}
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseWalk(parseChild)
		}
	}
	parseWalk(parseDoc)
	return parseIssues, nil
}

// evaluateA11yNode applies every rule to a single element node.
func evaluateA11yNode(parseNode *html.Node, parseRel string, parseLine int, parseLabeledIDs map[string]bool, parseWrapped map[*html.Node]bool) []lintIssueRecord {
	parseIssues := []lintIssueRecord{}
	parseMake := func(parseCode string, parseMessage string) lintIssueRecord {
		return lintIssueRecord{
			Linter:   a11yLinterName,
			Severity: "warning",
			Path:     parseRel,
			Line:     parseLine,
			Message:  parseMessage,
			Symbol:   parseCode,
		}
	}

	if parseTabindex := strings.TrimSpace(htmlAttr(parseNode, "tabindex")); parseTabindex != "" {
		if parseValue, parseConvErr := strconv.Atoi(parseTabindex); parseConvErr == nil && parseValue > 0 {
			parseIssues = append(parseIssues, parseMake("a11y/tabindex-positive",
				"positive tabindex ("+parseTabindex+") overrides natural focus order; use tabindex=\"0\" or restructure the DOM"))
		}
	}

	switch parseNode.DataAtom {
	case atom.Html:
		if strings.TrimSpace(htmlAttr(parseNode, "lang")) == "" {
			parseIssues = append(parseIssues, parseMake("a11y/html-lang",
				"<html> is missing a lang attribute; screen readers need it to pick a voice (e.g. lang=\"en\")"))
		}
	case atom.Img:
		if !htmlHasAttr(parseNode, "alt") {
			parseIssues = append(parseIssues, parseMake("a11y/img-alt",
				"<img> has no alt attribute; add alt text, or alt=\"\" if the image is decorative"))
		}
	case atom.Button:
		if !hasAccessibleName(parseNode) {
			parseIssues = append(parseIssues, parseMake("a11y/interactive-name",
				"<button> has no accessible name; add text content, aria-label, or aria-labelledby"))
		}
	case atom.A:
		if htmlHasAttr(parseNode, "href") && !hasAccessibleName(parseNode) {
			parseIssues = append(parseIssues, parseMake("a11y/interactive-name",
				"link has no accessible name; add link text, aria-label, or an img alt"))
		}
	case atom.Input, atom.Select, atom.Textarea:
		if a11yControlNeedsLabel(parseNode) && !a11yControlHasLabel(parseNode, parseLabeledIDs, parseWrapped) {
			parseIssues = append(parseIssues, parseMake("a11y/control-label",
				"form control has no associated label; add <label for>, aria-label, aria-labelledby, or wrap it in a <label>"))
		}
	}
	return parseIssues
}

// collectA11yLabelTargets gathers the ids referenced by <label for> and the set of controls
// wrapped directly inside a <label>, so control-label evaluation can resolve associations.
func collectA11yLabelTargets(parseRoot *html.Node) (map[string]bool, map[*html.Node]bool) {
	parseLabeledIDs := map[string]bool{}
	parseWrapped := map[*html.Node]bool{}
	var parseWalk func(parseNode *html.Node)
	parseWalk = func(parseNode *html.Node) {
		if parseNode.Type == html.ElementNode && parseNode.DataAtom == atom.Label {
			if parseFor := strings.TrimSpace(htmlAttr(parseNode, "for")); parseFor != "" {
				parseLabeledIDs[parseFor] = true
			}
			markWrappedControls(parseNode, parseWrapped)
		}
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseWalk(parseChild)
		}
	}
	parseWalk(parseRoot)
	return parseLabeledIDs, parseWrapped
}

// markWrappedControls records every form control nested under a label element.
func markWrappedControls(parseNode *html.Node, parseWrapped map[*html.Node]bool) {
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		if parseChild.Type == html.ElementNode {
			switch parseChild.DataAtom {
			case atom.Input, atom.Select, atom.Textarea:
				parseWrapped[parseChild] = true
			}
		}
		markWrappedControls(parseChild, parseWrapped)
	}
}

// a11yControlNeedsLabel reports whether a form control requires a label. Hidden inputs and
// button-like inputs (submit/button/reset/image) carry their own name and are exempt.
func a11yControlNeedsLabel(parseNode *html.Node) bool {
	if parseNode.DataAtom != atom.Input {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(htmlAttr(parseNode, "type"))) {
	case "hidden", "submit", "button", "reset", "image":
		return false
	default:
		return true
	}
}

// a11yControlHasLabel reports whether a control has any accessible-name association.
func a11yControlHasLabel(parseNode *html.Node, parseLabeledIDs map[string]bool, parseWrapped map[*html.Node]bool) bool {
	if strings.TrimSpace(htmlAttr(parseNode, "aria-label")) != "" ||
		strings.TrimSpace(htmlAttr(parseNode, "aria-labelledby")) != "" ||
		strings.TrimSpace(htmlAttr(parseNode, "title")) != "" {
		return true
	}
	if parseID := strings.TrimSpace(htmlAttr(parseNode, "id")); parseID != "" && parseLabeledIDs[parseID] {
		return true
	}
	return parseWrapped[parseNode]
}

// hasAccessibleName reports whether an element exposes a non-empty accessible name through text
// content, aria-label/labelledby, title, or (for links) a labeled image child.
func hasAccessibleName(parseNode *html.Node) bool {
	if strings.TrimSpace(htmlAttr(parseNode, "aria-label")) != "" ||
		strings.TrimSpace(htmlAttr(parseNode, "aria-labelledby")) != "" ||
		strings.TrimSpace(htmlAttr(parseNode, "title")) != "" {
		return true
	}
	if strings.TrimSpace(nodeText(parseNode)) != "" {
		return true
	}
	return hasLabeledImageDescendant(parseNode)
}

// hasLabeledImageDescendant reports whether the node contains an <img> with alt text or an
// element carrying aria-label, which would give an otherwise text-less control a name.
func hasLabeledImageDescendant(parseNode *html.Node) bool {
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		if parseChild.Type == html.ElementNode {
			if parseChild.DataAtom == atom.Img && strings.TrimSpace(htmlAttr(parseChild, "alt")) != "" {
				return true
			}
			if strings.TrimSpace(htmlAttr(parseChild, "aria-label")) != "" {
				return true
			}
		}
		if hasLabeledImageDescendant(parseChild) {
			return true
		}
	}
	return false
}

// nodeText returns the concatenated text-node content under a node.
func nodeText(parseNode *html.Node) string {
	var parseBuilder strings.Builder
	var parseWalk func(parseNode *html.Node)
	parseWalk = func(parseNode *html.Node) {
		if parseNode.Type == html.TextNode {
			parseBuilder.WriteString(parseNode.Data)
		}
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseWalk(parseChild)
		}
	}
	parseWalk(parseNode)
	return parseBuilder.String()
}

// htmlAttr returns an element's attribute value (empty when absent).
func htmlAttr(parseNode *html.Node, parseKey string) string {
	for _, parseAttr := range parseNode.Attr {
		if strings.EqualFold(parseAttr.Key, parseKey) {
			return parseAttr.Val
		}
	}
	return ""
}

// htmlHasAttr reports whether an element carries an attribute, regardless of value (so alt=""
// counts as present — an intentional decorative image).
func htmlHasAttr(parseNode *html.Node, parseKey string) bool {
	for _, parseAttr := range parseNode.Attr {
		if strings.EqualFold(parseAttr.Key, parseKey) {
			return true
		}
	}
	return false
}

// locateNthTag returns the 1-based line of the nth opening occurrence of a tag in the raw HTML,
// for a best-effort source location on a finding (0 when not found). It matches source order
// because the walk visits same-tag elements in document order.
func locateNthTag(parseContent string, parseTag string, parseN int) int {
	parseNeedle := "<" + parseTag
	parseOffset := 0
	parseFound := 0
	for {
		parseIndex := indexFoldFrom(parseContent, parseNeedle, parseOffset)
		if parseIndex < 0 {
			return 0
		}
		parseFound++
		if parseFound == parseN {
			return strings.Count(parseContent[:parseIndex], "\n") + 1
		}
		parseOffset = parseIndex + len(parseNeedle)
	}
}

// indexFoldFrom finds needle in s starting at from, case-insensitively, returning -1 if absent.
func indexFoldFrom(parseS string, parseNeedle string, parseFrom int) int {
	if parseFrom < 0 || parseFrom > len(parseS) {
		return -1
	}
	parseIndex := strings.Index(strings.ToLower(parseS[parseFrom:]), strings.ToLower(parseNeedle))
	if parseIndex < 0 {
		return -1
	}
	return parseFrom + parseIndex
}
