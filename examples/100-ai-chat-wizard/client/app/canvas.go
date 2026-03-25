//go:build js && wasm

package app

import (
	"fmt"
	"regexp"
	"strings"
)

type canvasArtifact struct {
	ID           string
	MessageIndex int
	BlockIndex   int
	Info         string
	Language     string
	Label        string
	Source       string
	Document     string
	Focus        canvasFocusRegion
	FocusOptions []canvasFocusRegion
}

type canvasFocusRegion struct {
	File      string
	Label     string
	Symbol    string
	Kind      string
	StartLine int
	EndLine   int
}

type canvasPatchRecord struct {
	ID           string
	Type         string
	TargetLabel  string
	Summary      string
	SourceBefore string
	SourceAfter  string
	Before       string
	After        string
	StartLine    int
	EndLine      int
}

type canvasConsoleEntry struct {
	Level         string
	Message       string
	Detail        string
	RenderVersion int
}

type canvasSessionState struct {
	Active                bool
	SessionID             string
	ArtifactID            string
	SourceMessageIndex    int
	CurrentFileID         string
	FocusedRegion         canvasFocusRegion
	FocusOptions          []canvasFocusRegion
	OriginalSource        string
	CurrentSource         string
	FocusDraft            string
	LatestRenderedVersion int
	LatestPatchVersion    int
	Dirty                 bool
	PreviewStatus         string
	RuntimeStatus         string
	ConsoleEntries        []canvasConsoleEntry
	PatchHistory          []canvasPatchRecord
	LayoutMode            string
	SplitRatio            float64
	ConsoleOpen           bool
}

const (
	canvasLayoutHidden  = "hidden"
	canvasLayoutSplit   = "split"
	canvasLayoutOverlay = "overlay"

	canvasPreviewNotRendered = "not-rendered"
	canvasPreviewRendering   = "rendering"
	canvasPreviewRendered    = "rendered"
	canvasPreviewRuntimeErr  = "runtime-error"
	canvasPreviewStale       = "stale"
	canvasPreviewPatched     = "patched"
)

var fencedBlockPattern = regexp.MustCompile("(?s)```([^\\n`]*)\\n(.*?)\\n```")
var canvasFunctionPattern = regexp.MustCompile(`^\s*(?:export\s+default\s+|export\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)`)
var canvasClassPattern = regexp.MustCompile(`^\s*(?:export\s+default\s+|export\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
var canvasConstPattern = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)

func parseLatestCanvasPreview(parseMessages []message) (canvasArtifact, bool) {
	parseArtifacts := parseAllCanvasArtifacts(parseMessages)
	if len(parseArtifacts) == 0 {
		return canvasArtifact{}, false
	}
	return parseArtifacts[len(parseArtifacts)-1], true
}

func parseAllCanvasArtifacts(parseMessages []message) []canvasArtifact {
	parseArtifacts := make([]canvasArtifact, 0, 4)
	for parseIdx, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		parseArtifacts = append(parseArtifacts, canvasArtifactsFromMarkdown(parseIdx, parseMessageItem.Content)...)
	}
	return parseArtifacts
}

func canvasArtifactsForMessage(parseMessages []message, parseMessageIndex int) []canvasArtifact {
	if parseMessageIndex < 0 || parseMessageIndex >= len(parseMessages) {
		return nil
	}
	parseMessageItem := parseMessages[parseMessageIndex]
	if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
		return nil
	}
	return canvasArtifactsFromMarkdown(parseMessageIndex, parseMessageItem.Content)
}

func parseFindCanvasArtifact(parseMessages []message, parseArtifactID string) (canvasArtifact, bool) {
	parseArtifactID = strings.TrimSpace(parseArtifactID)
	if parseArtifactID == "" {
		return canvasArtifact{}, false
	}
	for _, parseArtifact := range parseAllCanvasArtifacts(parseMessages) {
		if parseArtifact.ParseID == parseArtifactID {
			return parseArtifact, true
		}
	}
	return canvasArtifact{}, false
}

func canvasPreviewFromMarkdown(parseMarkdown string) (canvasArtifact, bool) {
	parseArtifacts := canvasArtifactsFromMarkdown(0, parseMarkdown)
	if len(parseArtifacts) == 0 {
		return canvasArtifact{}, false
	}
	return parseArtifacts[len(parseArtifacts)-1], true
}

func canvasArtifactsFromMarkdown(parseMessageIndex int, parseMarkdown string) []canvasArtifact {
	parseMatches := fencedBlockPattern.FindAllStringSubmatch(parseMarkdown, -1)
	if len(parseMatches) == 0 {
		return nil
	}
	parseArtifacts := make([]canvasArtifact, 0, len(parseMatches))
	for parseBlockIndex, parseMatch := range parseMatches {
		parseInfo := strings.TrimSpace(parseMatch[1])
		parseSource := strings.TrimSpace(parseMatch[2])
		if parseSource == "" {
			continue
		}
		parseLanguage, parseOk := canvasFenceLanguage(parseInfo)
		if !parseOk {
			continue
		}
		parseFocusOptions := parseDeriveCanvasFocusRegions(parseSource)
		parseFocus := canvasDefaultFocusRegion(parseFocusOptions, parseSource)
		parseLabel := canvasArtifactLabel(parseLanguage, parseSource, parseBlockIndex)
		parseArtifacts = append(parseArtifacts, canvasArtifact{
			ID:           fmt.Sprintf("m%d-b%d", parseMessageIndex, parseBlockIndex),
			MessageIndex: parseMessageIndex,
			BlockIndex:   parseBlockIndex,
			Info:         parseInfo,
			Language:     parseLanguage,
			Label:        parseLabel,
			Source:       parseSource,
			Document:     buildCanvasDocument(parseSource),
			Focus:        parseFocus,
			FocusOptions: parseFocusOptions,
		})
	}
	return parseArtifacts
}

func canvasFenceLanguage(parseInfo string) (string, bool) {
	parseFields := strings.Fields(strings.ToLower(strings.TrimSpace(parseInfo)))
	if len(parseFields) == 0 {
		return "", false
	}
	for _, parseField := range parseFields {
		switch parseField {
		case "canvas":
			return "canvas", true
		}
	}
	switch parseFields[0] {
	case "html", "htm":
		return "html", true
	case "javascript", "js":
		return "javascript", true
	}
	return "", false
}

func canvasArtifactLabel(parseLanguage, parseSource string, parseBlockIndex int) string {
	switch parseLanguage {
	case "html":
		return "HTML demo"
	case "javascript":
		if strings.Contains(parseSource, "function App") || strings.Contains(parseSource, "const App") {
			return "App component"
		}
		return "JavaScript demo"
	case "canvas":
		if strings.Contains(strings.ToLower(parseSource), "<html") {
			return "Canvas page"
		}
		if strings.Contains(parseSource, "function App") || strings.Contains(parseSource, "const App") {
			return "App component"
		}
		return fmt.Sprintf("Canvas block %d", parseBlockIndex+1)
	default:
		return fmt.Sprintf("Canvas block %d", parseBlockIndex+1)
	}
}

func canvasDefaultFocusRegion(parseRegions []canvasFocusRegion, parseSource string) canvasFocusRegion {
	for _, parseRegion := range parseRegions {
		if parseRegion.Kind != "file" {
			return parseRegion
		}
	}
	parseLineCount := 1 + strings.Count(parseSource, "\n")
	return canvasFocusRegion{
		File:      "artifact",
		Label:     "Full file",
		Kind:      "file",
		StartLine: 1,
		EndLine:   parseLineCount,
	}
}

func parseDeriveCanvasFocusRegions(parseSource string) []canvasFocusRegion {
	parseLines := strings.Split(strings.ReplaceAll(parseSource, "\r\n", "\n"), "\n")
	parseRegions := make([]canvasFocusRegion, 0, 8)
	parseRegions = append(parseRegions, canvasFocusRegion{
		File:      "artifact",
		Label:     "Full file",
		Kind:      "file",
		StartLine: 1,
		EndLine:   len(parseLines),
	})
	for parseIdx, parseLine := range parseLines {
		parseLineNumber := parseIdx + 1
		if parseMatch := canvasFunctionPattern.FindStringSubmatch(parseLine); len(parseMatch) == 2 {
			parseRegions = append(parseRegions, canvasFocusRegion{
				File:      "artifact",
				Label:     parseMatch[1],
				Symbol:    parseMatch[1],
				Kind:      "function",
				StartLine: parseLineNumber,
				EndLine:   parseEstimateCanvasRegionEnd(parseLines, parseIdx),
			})
			continue
		}
		if parseMatch2 := canvasClassPattern.FindStringSubmatch(parseLine); len(parseMatch2) == 2 {
			parseRegions = append(parseRegions, canvasFocusRegion{
				File:      "artifact",
				Label:     parseMatch2[1],
				Symbol:    parseMatch2[1],
				Kind:      "class",
				StartLine: parseLineNumber,
				EndLine:   parseEstimateCanvasRegionEnd(parseLines, parseIdx),
			})
			continue
		}
		if parseMatch3 := canvasConstPattern.FindStringSubmatch(parseLine); len(parseMatch3) == 2 {
			parseRegions = append(parseRegions, canvasFocusRegion{
				File:      "artifact",
				Label:     parseMatch3[1],
				Symbol:    parseMatch3[1],
				Kind:      "block",
				StartLine: parseLineNumber,
				EndLine:   parseEstimateCanvasRegionEnd(parseLines, parseIdx),
			})
		}
	}
	return parseDedupeCanvasFocusRegions(parseRegions)
}

func parseDedupeCanvasFocusRegions(parseRegions []canvasFocusRegion) []canvasFocusRegion {
	if len(parseRegions) == 0 {
		return nil
	}
	parseSeen := make(map[string]bool, len(parseRegions))
	parseOut := make([]canvasFocusRegion, 0, len(parseRegions))
	for _, parseRegion := range parseRegions {
		parseKey := fmt.Sprintf("%s:%d:%d:%s", parseRegion.Kind, parseRegion.StartLine, parseRegion.EndLine, parseRegion.Label)
		if parseSeen[parseKey] {
			continue
		}
		parseSeen[parseKey] = true
		parseOut = append(parseOut, parseRegion)
	}
	return parseOut
}

func parseEstimateCanvasRegionEnd(parseLines []string, parseStart int) int {
	if parseStart < 0 || parseStart >= len(parseLines) {
		return len(parseLines)
	}
	parseDepth := 0
	isParseSeenBrace := false
	for parseIdx := parseStart; parseIdx < len(parseLines); parseIdx++ {
		parseLine := parseLines[parseIdx]
		parseDepth += strings.Count(parseLine, "{")
		if strings.Contains(parseLine, "{") {
			isParseSeenBrace = true
		}
		parseDepth -= strings.Count(parseLine, "}")
		if isParseSeenBrace && parseIdx > parseStart && parseDepth <= 0 {
			return parseIdx + 1
		}
		if !isParseSeenBrace && parseIdx > parseStart && strings.TrimSpace(parseLine) == "" {
			return parseIdx
		}
	}
	return len(parseLines)
}

func buildCanvasDocument(parseSource string) string {
	return buildCanvasRuntimeDocument(parseSource, "", "", 0)
}

func buildCanvasRuntimeDocument(parseSource, parseSessionID, parseArtifactID string, renderVersion int) string {
	parseTrimmed := strings.TrimSpace(parseSource)
	if parseTrimmed == "" {
		return ""
	}
	parseLower := strings.ToLower(parseTrimmed)
	parseDocument := parseTrimmed
	if !strings.Contains(parseLower, "<html") && !strings.Contains(parseLower, "<!doctype") {
		if !strings.Contains(parseTrimmed, "<") {
			parseTrimmed = fmt.Sprintf("<script>%s</script>", strings.ReplaceAll(parseTrimmed, "</script", "<\\/script"))
		}
		parseDocument = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <style>
    :root { color-scheme: light; }
    html, body {
      margin: 0;
      min-height: 100%%;
      background: #f6f8f7;
      font-family: Georgia, "Iowan Old Style", "Palatino Linotype", serif;
    }
    body {
      padding: 0;
    }
    #canvas {
      min-height: 100vh;
      box-sizing: border-box;
      padding: 1rem;
    }
  </style>
</head>
<body>
  <div id="canvas"></div>
  %s
</body>
</html>`, parseTrimmed)
	}
	return parseInjectCanvasRuntimeBridge(parseDocument, parseSessionID, parseArtifactID, renderVersion)
}

func parseInjectCanvasRuntimeBridge(parseDocument, parseSessionID, parseArtifactID string, renderVersion int) string {
	parseBridge := fmt.Sprintf(`<script>(function(){var sessionID=%q;var artifactID=%q;var renderVersion=%d;
function send(kind,payload){try{if(window.parent&&window.parent!==window){window.parent.postMessage({__gwcCanvas:true,sessionID:sessionID,artifactID:artifactID,renderVersion:renderVersion,kind:kind,payload:payload||{}}, "*");}}catch(_){}} 
function stringifyArgs(args){var out=[];for(var i=0;i<args.length;i++){var value=args[i];if(typeof value==="string"){out.push(value);continue;}try{out.push(JSON.stringify(value));}catch(_){out.push(String(value));}}return out.join(" ");} 
["log","info","warn","error"].forEach(function(level){if(typeof console[level]!=="function"){return;}var original=console[level].bind(console);console[level]=function(){send("console",{level:level,message:stringifyArgs(arguments)});return original.apply(console, arguments);};});
window.addEventListener("error", function(event){send("runtime_error",{message:String(event&&event.message||"Runtime error"),detail:event&&event.error&&event.error.stack?String(event.error.stack):""});});
window.addEventListener("unhandledrejection", function(event){var reason=event&&event.reason;send("runtime_error",{message:"Unhandled promise rejection",detail:reason&&reason.stack?String(reason.stack):String(reason)});});
document.addEventListener("DOMContentLoaded", function(){send("status",{previewStatus:"rendered",runtimeStatus:"ready"});});
send("status",{previewStatus:"rendering",runtimeStatus:"booting"});
})();</script>`, parseSessionID, parseArtifactID, renderVersion)
	parseLower := strings.ToLower(parseDocument)
	if parseIdx := strings.LastIndex(parseLower, "</body>"); parseIdx >= 0 {
		return parseDocument[:parseIdx] + parseBridge + parseDocument[parseIdx:]
	}
	if parseIdx2 := strings.LastIndex(parseLower, "</html>"); parseIdx2 >= 0 {
		return parseDocument[:parseIdx2] + parseBridge + parseDocument[parseIdx2:]
	}
	return parseDocument + parseBridge
}

func canvasFocusSnippet(parseSource string, parseFocus canvasFocusRegion) string {
	parseLines := strings.Split(strings.ReplaceAll(parseSource, "\r\n", "\n"), "\n")
	if len(parseLines) == 0 {
		return ""
	}
	parseStart, parseEnd := parseClampCanvasFocusRange(parseFocus, len(parseLines))
	if parseStart <= 0 || parseEnd <= 0 || parseStart > parseEnd {
		return parseSource
	}
	return strings.Join(parseLines[parseStart-1:parseEnd], "\n")
}

func parseClampCanvasFocusRange(parseFocus canvasFocusRegion, parseLineCount int) (int, int) {
	if parseLineCount <= 0 {
		return 0, 0
	}
	parseStart := parseFocus.StartLine
	parseEnd := parseFocus.EndLine
	if parseStart <= 0 {
		parseStart = 1
	}
	if parseEnd <= 0 || parseEnd > parseLineCount {
		parseEnd = parseLineCount
	}
	if parseStart > parseEnd {
		parseStart = 1
		parseEnd = parseLineCount
	}
	return parseStart, parseEnd
}

func parseReplaceCanvasFocusRegion(parseSource string, parseFocus canvasFocusRegion, parseReplacement string) (string, canvasPatchRecord, bool) {
	parseLines := strings.Split(strings.ReplaceAll(parseSource, "\r\n", "\n"), "\n")
	if len(parseLines) == 0 {
		return parseSource, canvasPatchRecord{}, false
	}
	parseStart, parseEnd := parseClampCanvasFocusRange(parseFocus, len(parseLines))
	if parseStart <= 0 || parseEnd <= 0 || parseStart > parseEnd {
		return parseSource, canvasPatchRecord{}, false
	}
	parseBefore := strings.Join(parseLines[parseStart-1:parseEnd], "\n")
	parseReplacementLines := strings.Split(strings.ReplaceAll(parseReplacement, "\r\n", "\n"), "\n")
	parseNextLines := append([]string{}, parseLines[:parseStart-1]...)
	parseNextLines = append(parseNextLines, parseReplacementLines...)
	parseNextLines = append(parseNextLines, parseLines[parseEnd:]...)
	parseNextSource := strings.Join(parseNextLines, "\n")
	parseRecord := canvasPatchRecord{
		Type:         "patch_region",
		TargetLabel:  canvasFocusDisplayLabel(parseFocus),
		Summary:      fmt.Sprintf("Patched %s", canvasFocusDisplayLabel(parseFocus)),
		SourceBefore: parseSource,
		SourceAfter:  parseNextSource,
		Before:       parseBefore,
		After:        parseReplacement,
		StartLine:    parseStart,
		EndLine:      parseEnd,
	}
	return parseNextSource, parseRecord, true
}

func parseDeriveCanvasPatch(parsePrevious, parseNext string, parseFocus canvasFocusRegion) (canvasPatchRecord, bool) {
	if parsePrevious == parseNext {
		return canvasPatchRecord{}, false
	}
	parsePrevLines := strings.Split(strings.ReplaceAll(parsePrevious, "\r\n", "\n"), "\n")
	parseNextLines := strings.Split(strings.ReplaceAll(parseNext, "\r\n", "\n"), "\n")
	parsePrefix := 0
	for parsePrefix < len(parsePrevLines) && parsePrefix < len(parseNextLines) && parsePrevLines[parsePrefix] == parseNextLines[parsePrefix] {
		parsePrefix++
	}
	parseSuffix := 0
	for parseSuffix < len(parsePrevLines)-parsePrefix && parseSuffix < len(parseNextLines)-parsePrefix &&
		parsePrevLines[len(parsePrevLines)-1-parseSuffix] == parseNextLines[len(parseNextLines)-1-parseSuffix] {
		parseSuffix++
	}
	parsePrevEnd := len(parsePrevLines) - parseSuffix
	parseNextEnd := len(parseNextLines) - parseSuffix
	if parsePrevEnd < parsePrefix {
		parsePrevEnd = parsePrefix
	}
	if parseNextEnd < parsePrefix {
		parseNextEnd = parsePrefix
	}
	parseStartLine := parsePrefix + 1
	parseEndLine := parsePrevEnd
	parseTargetLabel := "file"
	parsePatchType := "patch_file"
	if canvasRangesOverlap(parseStartLine, parseMaxInt(parseStartLine, parseEndLine), parseFocus.StartLine, parseFocus.EndLine) {
		parseTargetLabel = canvasFocusDisplayLabel(parseFocus)
		parsePatchType = "patch_region"
	}
	if parsePrefix == 0 && parseSuffix == 0 {
		parsePatchType = "replace_file"
		parseTargetLabel = "file"
	}
	return canvasPatchRecord{
		Type:         parsePatchType,
		TargetLabel:  parseTargetLabel,
		Summary:      fmt.Sprintf("Updated %s", parseTargetLabel),
		SourceBefore: parsePrevious,
		SourceAfter:  parseNext,
		Before:       strings.Join(parsePrevLines[parsePrefix:parsePrevEnd], "\n"),
		After:        strings.Join(parseNextLines[parsePrefix:parseNextEnd], "\n"),
		StartLine:    parseStartLine,
		EndLine:      parseEndLine,
	}, true
}

func canvasRangesOverlap(parseAStart, parseAEnd, parseBStart, parseBEnd int) bool {
	if parseAStart <= 0 || parseAEnd <= 0 || parseBStart <= 0 || parseBEnd <= 0 {
		return false
	}
	return parseAStart <= parseBEnd && parseBStart <= parseAEnd
}

func canvasFocusDisplayLabel(parseFocus canvasFocusRegion) string {
	if strings.TrimSpace(parseFocus.Label) != "" {
		return parseFocus.Label
	}
	if strings.TrimSpace(parseFocus.Symbol) != "" {
		return parseFocus.Symbol
	}
	if parseFocus.Kind == "file" {
		return "file"
	}
	return "focused region"
}

func parseMaxInt(parseA, parseB int) int {
	if parseA > parseB {
		return parseA
	}
	return parseB
}
