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

func latestCanvasPreview(messages []message) (canvasArtifact, bool) {
	artifacts := allCanvasArtifacts(messages)
	if len(artifacts) == 0 {
		return canvasArtifact{}, false
	}
	return artifacts[len(artifacts)-1], true
}

func allCanvasArtifacts(messages []message) []canvasArtifact {
	artifacts := make([]canvasArtifact, 0, 4)
	for idx, messageItem := range messages {
		if messageItem.Role != roleAssistant || messageItem.Pending {
			continue
		}
		artifacts = append(artifacts, canvasArtifactsFromMarkdown(idx, messageItem.Content)...)
	}
	return artifacts
}

func canvasArtifactsForMessage(messages []message, messageIndex int) []canvasArtifact {
	if messageIndex < 0 || messageIndex >= len(messages) {
		return nil
	}
	messageItem := messages[messageIndex]
	if messageItem.Role != roleAssistant || messageItem.Pending {
		return nil
	}
	return canvasArtifactsFromMarkdown(messageIndex, messageItem.Content)
}

func findCanvasArtifact(messages []message, artifactID string) (canvasArtifact, bool) {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return canvasArtifact{}, false
	}
	for _, artifact := range allCanvasArtifacts(messages) {
		if artifact.ID == artifactID {
			return artifact, true
		}
	}
	return canvasArtifact{}, false
}

func canvasPreviewFromMarkdown(markdown string) (canvasArtifact, bool) {
	artifacts := canvasArtifactsFromMarkdown(0, markdown)
	if len(artifacts) == 0 {
		return canvasArtifact{}, false
	}
	return artifacts[len(artifacts)-1], true
}

func canvasArtifactsFromMarkdown(messageIndex int, markdown string) []canvasArtifact {
	matches := fencedBlockPattern.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}
	artifacts := make([]canvasArtifact, 0, len(matches))
	for blockIndex, match := range matches {
		info := strings.TrimSpace(match[1])
		source := strings.TrimSpace(match[2])
		if source == "" {
			continue
		}
		language, ok := canvasFenceLanguage(info)
		if !ok {
			continue
		}
		focusOptions := deriveCanvasFocusRegions(source)
		focus := canvasDefaultFocusRegion(focusOptions, source)
		label := canvasArtifactLabel(language, source, blockIndex)
		artifacts = append(artifacts, canvasArtifact{
			ID:           fmt.Sprintf("m%d-b%d", messageIndex, blockIndex),
			MessageIndex: messageIndex,
			BlockIndex:   blockIndex,
			Info:         info,
			Language:     language,
			Label:        label,
			Source:       source,
			Document:     buildCanvasDocument(source),
			Focus:        focus,
			FocusOptions: focusOptions,
		})
	}
	return artifacts
}

func canvasFenceLanguage(info string) (string, bool) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(info)))
	if len(fields) == 0 {
		return "", false
	}
	for _, field := range fields {
		switch field {
		case "canvas":
			return "canvas", true
		}
	}
	switch fields[0] {
	case "html", "htm":
		return "html", true
	case "javascript", "js":
		return "javascript", true
	}
	return "", false
}

func canvasArtifactLabel(language, source string, blockIndex int) string {
	switch language {
	case "html":
		return "HTML demo"
	case "javascript":
		if strings.Contains(source, "function App") || strings.Contains(source, "const App") {
			return "App component"
		}
		return "JavaScript demo"
	case "canvas":
		if strings.Contains(strings.ToLower(source), "<html") {
			return "Canvas page"
		}
		if strings.Contains(source, "function App") || strings.Contains(source, "const App") {
			return "App component"
		}
		return fmt.Sprintf("Canvas block %d", blockIndex+1)
	default:
		return fmt.Sprintf("Canvas block %d", blockIndex+1)
	}
}

func canvasDefaultFocusRegion(regions []canvasFocusRegion, source string) canvasFocusRegion {
	for _, region := range regions {
		if region.Kind != "file" {
			return region
		}
	}
	lineCount := 1 + strings.Count(source, "\n")
	return canvasFocusRegion{
		File:      "artifact",
		Label:     "Full file",
		Kind:      "file",
		StartLine: 1,
		EndLine:   lineCount,
	}
}

func deriveCanvasFocusRegions(source string) []canvasFocusRegion {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	regions := make([]canvasFocusRegion, 0, 8)
	regions = append(regions, canvasFocusRegion{
		File:      "artifact",
		Label:     "Full file",
		Kind:      "file",
		StartLine: 1,
		EndLine:   len(lines),
	})
	for idx, line := range lines {
		lineNumber := idx + 1
		if match := canvasFunctionPattern.FindStringSubmatch(line); len(match) == 2 {
			regions = append(regions, canvasFocusRegion{
				File:      "artifact",
				Label:     match[1],
				Symbol:    match[1],
				Kind:      "function",
				StartLine: lineNumber,
				EndLine:   estimateCanvasRegionEnd(lines, idx),
			})
			continue
		}
		if match := canvasClassPattern.FindStringSubmatch(line); len(match) == 2 {
			regions = append(regions, canvasFocusRegion{
				File:      "artifact",
				Label:     match[1],
				Symbol:    match[1],
				Kind:      "class",
				StartLine: lineNumber,
				EndLine:   estimateCanvasRegionEnd(lines, idx),
			})
			continue
		}
		if match := canvasConstPattern.FindStringSubmatch(line); len(match) == 2 {
			regions = append(regions, canvasFocusRegion{
				File:      "artifact",
				Label:     match[1],
				Symbol:    match[1],
				Kind:      "block",
				StartLine: lineNumber,
				EndLine:   estimateCanvasRegionEnd(lines, idx),
			})
		}
	}
	return dedupeCanvasFocusRegions(regions)
}

func dedupeCanvasFocusRegions(regions []canvasFocusRegion) []canvasFocusRegion {
	if len(regions) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(regions))
	out := make([]canvasFocusRegion, 0, len(regions))
	for _, region := range regions {
		key := fmt.Sprintf("%s:%d:%d:%s", region.Kind, region.StartLine, region.EndLine, region.Label)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, region)
	}
	return out
}

func estimateCanvasRegionEnd(lines []string, start int) int {
	if start < 0 || start >= len(lines) {
		return len(lines)
	}
	depth := 0
	seenBrace := false
	for idx := start; idx < len(lines); idx++ {
		line := lines[idx]
		depth += strings.Count(line, "{")
		if strings.Contains(line, "{") {
			seenBrace = true
		}
		depth -= strings.Count(line, "}")
		if seenBrace && idx > start && depth <= 0 {
			return idx + 1
		}
		if !seenBrace && idx > start && strings.TrimSpace(line) == "" {
			return idx
		}
	}
	return len(lines)
}

func buildCanvasDocument(source string) string {
	return buildCanvasRuntimeDocument(source, "", "", 0)
}

func buildCanvasRuntimeDocument(source, sessionID, artifactID string, renderVersion int) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	document := trimmed
	if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype") {
		if !strings.Contains(trimmed, "<") {
			trimmed = fmt.Sprintf("<script>%s</script>", strings.ReplaceAll(trimmed, "</script", "<\\/script"))
		}
		document = fmt.Sprintf(`<!DOCTYPE html>
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
</html>`, trimmed)
	}
	return injectCanvasRuntimeBridge(document, sessionID, artifactID, renderVersion)
}

func injectCanvasRuntimeBridge(document, sessionID, artifactID string, renderVersion int) string {
	bridge := fmt.Sprintf(`<script>(function(){var sessionID=%q;var artifactID=%q;var renderVersion=%d;
function send(kind,payload){try{if(window.parent&&window.parent!==window){window.parent.postMessage({__gwcCanvas:true,sessionID:sessionID,artifactID:artifactID,renderVersion:renderVersion,kind:kind,payload:payload||{}}, "*");}}catch(_){}} 
function stringifyArgs(args){var out=[];for(var i=0;i<args.length;i++){var value=args[i];if(typeof value==="string"){out.push(value);continue;}try{out.push(JSON.stringify(value));}catch(_){out.push(String(value));}}return out.join(" ");} 
["log","info","warn","error"].forEach(function(level){if(typeof console[level]!=="function"){return;}var original=console[level].bind(console);console[level]=function(){send("console",{level:level,message:stringifyArgs(arguments)});return original.apply(console, arguments);};});
window.addEventListener("error", function(event){send("runtime_error",{message:String(event&&event.message||"Runtime error"),detail:event&&event.error&&event.error.stack?String(event.error.stack):""});});
window.addEventListener("unhandledrejection", function(event){var reason=event&&event.reason;send("runtime_error",{message:"Unhandled promise rejection",detail:reason&&reason.stack?String(reason.stack):String(reason)});});
document.addEventListener("DOMContentLoaded", function(){send("status",{previewStatus:"rendered",runtimeStatus:"ready"});});
send("status",{previewStatus:"rendering",runtimeStatus:"booting"});
})();</script>`, sessionID, artifactID, renderVersion)
	lower := strings.ToLower(document)
	if idx := strings.LastIndex(lower, "</body>"); idx >= 0 {
		return document[:idx] + bridge + document[idx:]
	}
	if idx := strings.LastIndex(lower, "</html>"); idx >= 0 {
		return document[:idx] + bridge + document[idx:]
	}
	return document + bridge
}

func canvasFocusSnippet(source string, focus canvasFocusRegion) string {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	start, end := clampCanvasFocusRange(focus, len(lines))
	if start <= 0 || end <= 0 || start > end {
		return source
	}
	return strings.Join(lines[start-1:end], "\n")
}

func clampCanvasFocusRange(focus canvasFocusRegion, lineCount int) (int, int) {
	if lineCount <= 0 {
		return 0, 0
	}
	start := focus.StartLine
	end := focus.EndLine
	if start <= 0 {
		start = 1
	}
	if end <= 0 || end > lineCount {
		end = lineCount
	}
	if start > end {
		start = 1
		end = lineCount
	}
	return start, end
}

func replaceCanvasFocusRegion(source string, focus canvasFocusRegion, replacement string) (string, canvasPatchRecord, bool) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return source, canvasPatchRecord{}, false
	}
	start, end := clampCanvasFocusRange(focus, len(lines))
	if start <= 0 || end <= 0 || start > end {
		return source, canvasPatchRecord{}, false
	}
	before := strings.Join(lines[start-1:end], "\n")
	replacementLines := strings.Split(strings.ReplaceAll(replacement, "\r\n", "\n"), "\n")
	nextLines := append([]string{}, lines[:start-1]...)
	nextLines = append(nextLines, replacementLines...)
	nextLines = append(nextLines, lines[end:]...)
	nextSource := strings.Join(nextLines, "\n")
	record := canvasPatchRecord{
		Type:         "patch_region",
		TargetLabel:  canvasFocusDisplayLabel(focus),
		Summary:      fmt.Sprintf("Patched %s", canvasFocusDisplayLabel(focus)),
		SourceBefore: source,
		SourceAfter:  nextSource,
		Before:       before,
		After:        replacement,
		StartLine:    start,
		EndLine:      end,
	}
	return nextSource, record, true
}

func deriveCanvasPatch(previous, next string, focus canvasFocusRegion) (canvasPatchRecord, bool) {
	if previous == next {
		return canvasPatchRecord{}, false
	}
	prevLines := strings.Split(strings.ReplaceAll(previous, "\r\n", "\n"), "\n")
	nextLines := strings.Split(strings.ReplaceAll(next, "\r\n", "\n"), "\n")
	prefix := 0
	for prefix < len(prevLines) && prefix < len(nextLines) && prevLines[prefix] == nextLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(prevLines)-prefix && suffix < len(nextLines)-prefix &&
		prevLines[len(prevLines)-1-suffix] == nextLines[len(nextLines)-1-suffix] {
		suffix++
	}
	prevEnd := len(prevLines) - suffix
	nextEnd := len(nextLines) - suffix
	if prevEnd < prefix {
		prevEnd = prefix
	}
	if nextEnd < prefix {
		nextEnd = prefix
	}
	startLine := prefix + 1
	endLine := prevEnd
	targetLabel := "file"
	patchType := "patch_file"
	if canvasRangesOverlap(startLine, maxInt(startLine, endLine), focus.StartLine, focus.EndLine) {
		targetLabel = canvasFocusDisplayLabel(focus)
		patchType = "patch_region"
	}
	if prefix == 0 && suffix == 0 {
		patchType = "replace_file"
		targetLabel = "file"
	}
	return canvasPatchRecord{
		Type:         patchType,
		TargetLabel:  targetLabel,
		Summary:      fmt.Sprintf("Updated %s", targetLabel),
		SourceBefore: previous,
		SourceAfter:  next,
		Before:       strings.Join(prevLines[prefix:prevEnd], "\n"),
		After:        strings.Join(nextLines[prefix:nextEnd], "\n"),
		StartLine:    startLine,
		EndLine:      endLine,
	}, true
}

func canvasRangesOverlap(aStart, aEnd, bStart, bEnd int) bool {
	if aStart <= 0 || aEnd <= 0 || bStart <= 0 || bEnd <= 0 {
		return false
	}
	return aStart <= bEnd && bStart <= aEnd
}

func canvasFocusDisplayLabel(focus canvasFocusRegion) string {
	if strings.TrimSpace(focus.Label) != "" {
		return focus.Label
	}
	if strings.TrimSpace(focus.Symbol) != "" {
		return focus.Symbol
	}
	if focus.Kind == "file" {
		return "file"
	}
	return "focused region"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
