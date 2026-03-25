//go:build js && wasm
// +build js,wasm

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/interop"
)

// emitBrowserPanicReport is a core package helper.
func emitBrowserPanicReport(parseReport PanicReport) bool {
	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return false
	}
	parseConsole := parseGlobal.Get("console")
	if !parseConsole.Present() {
		return false
	}

	parseHeader := fmt.Sprintf("[%s] %s panic in %s", parseReport.Code, parseReport.Phase, parseReport.Subject)
	if parseSummary := strings.TrimSpace(parseReport.Summary); parseSummary != "" {
		parseHeader += ": " + parseSummary
	}
	parsePayload := map[string]any{
		"source":          parseReport.Source,
		"phase":           string(parseReport.Phase),
		"subject":         parseReport.Subject,
		"where":           parseReport.Where,
		"path":            parseReport.Path,
		"error":           parseReport.Summary,
		"runtime":         parseReport.Consequence,
		"next":            parseReport.Remediation,
		"docs":            parseReport.Docs,
		"code":            parseReport.Code,
		"componentStack":  parseReport.ComponentStack,
		"appFrames":       parseReport.AppFrames,
		"frameworkFrames": parseReport.FrameworkFrames,
		"platformFrames":  parseReport.PlatformFrames,
		"artifact":        parseReport.Artifact,
	}
	parsePayloadLine := "[GWC structured panic]"
	if parseEncoded, parseErr2 := json.Marshal(parsePayload); parseErr2 == nil {
		parsePayloadLine += " " + string(parseEncoded)
	}

	isParseEmitted := false
	isParseGrouped := false
	if _, parseErr3 := parseConsole.Call("groupCollapsed", parseHeader); parseErr3 == nil {
		isParseEmitted = true
		isParseGrouped = true
	}
	if _, parseErr4 := parseConsole.Call("error", parseReport.Formatted); parseErr4 == nil {
		isParseEmitted = true
	}
	if _, parseErr5 := parseConsole.Call("log", parsePayloadLine); parseErr5 == nil {
		isParseEmitted = true
	}
	if isParseGrouped {
		_, _ = parseConsole.Call("groupEnd")
	}
	return isParseEmitted
}
