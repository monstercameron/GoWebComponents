//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"
	"unicode"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// getExampleCatalogItems keeps the public site focused on runnable learning examples.
func getExampleCatalogItems(parseItems []docsItem) []docsItem {
	parseExamples := make([]docsItem, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if parseItem.Type == kindExample || parseItem.Content.Kind == contentKindExample || parseItem.Content.Kind == contentKindCounter {
			parseExamples = append(parseExamples, parseItem)
		}
	}
	return parseExamples
}

// getStatusItemCount counts items that match one stability label.
func getStatusItemCount(parseItems []docsItem, parseStatus string) int {
	parseCount := 0
	for _, parseItem := range parseItems {
		if parseItem.Status == parseStatus {
			parseCount++
		}
	}
	return parseCount
}

// getDistinctModuleCount counts the unique module labels represented in the current gallery set.
func getDistinctModuleCount(parseItems []docsItem) int {
	parseModules := make(map[string]struct{}, len(parseItems))
	for _, parseItem := range parseItems {
		parseModule := strings.TrimSpace(parseItem.Module)
		if parseModule == "" {
			continue
		}
		parseModules[parseModule] = struct{}{}
	}
	return len(parseModules)
}

// getVisibleCatalogValues keeps one ordered filter list aligned with the currently visible example set.
func getVisibleCatalogValues(parseCatalogValues []string, parseItems []docsItem, parseGetValue func(docsItem) string) []string {
	parsePresent := make(map[string]struct{}, len(parseItems))
	for _, parseItem := range parseItems {
		parseValue := strings.TrimSpace(parseGetValue(parseItem))
		if parseValue == "" {
			continue
		}
		parsePresent[parseValue] = struct{}{}
	}

	parseValues := make([]string, 0, len(parseCatalogValues))
	for _, parseValue := range parseCatalogValues {
		if _, parseFound := parsePresent[parseValue]; parseFound {
			parseValues = append(parseValues, parseValue)
		}
	}
	return parseValues
}

// getExampleFocusSummary returns one concise, reader-facing summary for the selected example.
func getExampleFocusSummary(parseItem docsItem) string {
	parseSummary := strings.TrimSpace(parseItem.Content.Description)
	if parseSummary == "" {
		parseSummary = strings.TrimSpace(parseItem.Blurb)
	}
	switch {
	case strings.HasPrefix(parseSummary, "Run the live ") && strings.Contains(parseSummary, "inspect the mirrored Go source side by side."):
		return "Start with the live surface, then map the rendered elements back to the Go source beside it."
	case strings.HasPrefix(parseSummary, "Interactive example for ") && strings.Contains(parseSummary, "with mirrored Go source and a generated wasm build."):
		return "A runnable Go + WASM example focused on " + parseItem.Title + "."
	default:
		return parseSummary
	}
}

// formatExampleConceptLabel converts slug-like catalog values into short reader-facing labels.
func formatExampleConceptLabel(parseRaw string) string {
	parseKnownLabels := map[string]string{
		"api":  "API",
		"apis": "APIs",
		"css":  "CSS",
		"dom":  "DOM",
		"go":   "Go",
		"html": "HTML",
		"id":   "ID",
		"ids":  "IDs",
		"pwa":  "PWA",
		"ssr":  "SSR",
		"ui":   "UI",
		"url":  "URL",
		"urls": "URLs",
		"wasm": "WASM",
		"ws":   "WebSocket",
	}

	parseWords := strings.FieldsFunc(strings.TrimSpace(parseRaw), func(parseRune rune) bool {
		return parseRune == '-' || parseRune == '_' || parseRune == '/' || unicode.IsSpace(parseRune)
	})
	if len(parseWords) == 0 {
		return ""
	}

	for parseIndex, parseWord := range parseWords {
		parseLower := strings.ToLower(parseWord)
		if parseKnown, parseFound := parseKnownLabels[parseLower]; parseFound {
			parseWords[parseIndex] = parseKnown
			continue
		}
		parseRunes := []rune(strings.ToLower(parseWord))
		if len(parseRunes) == 0 {
			continue
		}
		parseRunes[0] = unicode.ToUpper(parseRunes[0])
		parseWords[parseIndex] = string(parseRunes)
	}

	return strings.Join(parseWords, " ")
}

// buildExampleConceptLabels converts tags and module metadata into concise concept chips.
func buildExampleConceptLabels(parseItem docsItem) []string {
	parseLabels := make([]string, 0, len(parseItem.Tags)+1)
	parseSeen := make(map[string]struct{}, len(parseItem.Tags)+1)
	parseAddLabel := func(parseRaw string) {
		parseLabel := strings.TrimSpace(formatExampleConceptLabel(parseRaw))
		if parseLabel == "" {
			return
		}
		parseKey := strings.ToLower(parseLabel)
		if _, parseExists := parseSeen[parseKey]; parseExists {
			return
		}
		parseSeen[parseKey] = struct{}{}
		parseLabels = append(parseLabels, parseLabel)
	}

	for _, parseTag := range parseItem.Tags {
		parseAddLabel(parseTag)
		if len(parseLabels) >= 5 {
			return parseLabels
		}
	}
	parseAddLabel(parseItem.Module)
	if len(parseLabels) == 0 {
		parseAddLabel(parseItem.Title)
	}
	return parseLabels
}

// buildExampleLearningPoints returns short study prompts that keep the detail view centered on concepts.
func buildExampleLearningPoints(parseItem docsItem) []string {
	parsePoints := make([]string, 0, 4)
	parseSeen := make(map[string]struct{}, 4)
	parseAddPoint := func(parsePoint string) {
		parsePoint = strings.TrimSpace(parsePoint)
		if parsePoint == "" {
			return
		}
		parseKey := strings.ToLower(parsePoint)
		if _, parseExists := parseSeen[parseKey]; parseExists {
			return
		}
		parseSeen[parseKey] = struct{}{}
		parsePoints = append(parsePoints, parsePoint)
	}

	switch strings.TrimSpace(parseItem.Module) {
	case "state":
		parseAddPoint("Track which rendered elements react to state changes and which parts of the surface stay static.")
	case "router":
		parseAddPoint("Watch how navigation, params, loaders, or layout boundaries reshape what the page renders.")
	case "rendering":
		parseAddPoint("Compare the first visible markup with the client behavior that takes over after the runtime boots.")
	case "interop":
		parseAddPoint("Notice where the example leans on browser APIs, multiple windows, tabs, or external DOM state.")
	case "data":
		parseAddPoint("Follow how the page expresses loading, success, empty, and failure states.")
	default:
		parseAddPoint("Identify the smallest interactive surface this example is designed to teach.")
	}

	parseAddPoint("Map the visible elements back to the handlers, hooks, and typed HTML builders in main.go.")
	if parseItem.Content.PreviewPath != "" {
		parseAddPoint("Use the standalone preview when the concept depends on its own document, route, popup, or browser-global behavior.")
	} else {
		parseAddPoint("Stay source-first here and trace the concept through the mirrored Go source.")
	}

	return parsePoints
}

// renderExampleConceptChipNodes renders one shared chip style for concept labels across cards and detail panels.
func renderExampleConceptChipNodes(parseLabels []string) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseLabels))
	for _, parseLabel := range parseLabels {
		parseNodes = append(parseNodes,
			Span(ClassStr("rounded-full border border-cyan-300/20 bg-cyan-400/10 px-3 py-1 text-[11px] uppercase tracking-[0.16em] text-cyan-100"), Text(parseLabel)),
		)
	}
	return parseNodes
}
