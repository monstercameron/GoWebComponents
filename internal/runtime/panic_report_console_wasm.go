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
func emitBrowserPanicReport(parsePanicReport PanicReport) bool {
	parseBrowserGlobal, parseGlobalErr := interop.GetGlobalThis()
	if parseGlobalErr != nil {
		return false
	}
	parseBrowserConsole := parseBrowserGlobal.Get("console")
	if !parseBrowserConsole.Present() {
		return false
	}

	parsePanicHeader := fmt.Sprintf("[%s] %s panic in %s", parsePanicReport.Code, parsePanicReport.Phase, parsePanicReport.Subject)
	if parsePanicSummary := strings.TrimSpace(parsePanicReport.Summary); parsePanicSummary != "" {
		parsePanicHeader += ": " + parsePanicSummary
	}
	parsePanicPayload := map[string]any{
		"source":          parsePanicReport.Source,
		"phase":           string(parsePanicReport.Phase),
		"subject":         parsePanicReport.Subject,
		"where":           parsePanicReport.Where,
		"path":            parsePanicReport.Path,
		"error":           parsePanicReport.Summary,
		"runtime":         parsePanicReport.Consequence,
		"next":            parsePanicReport.Remediation,
		"docs":            parsePanicReport.Docs,
		"code":            parsePanicReport.Code,
		"componentStack":  parsePanicReport.ComponentStack,
		"appFrames":       parsePanicReport.AppFrames,
		"frameworkFrames": parsePanicReport.FrameworkFrames,
		"platformFrames":  parsePanicReport.PlatformFrames,
		"artifact":        parsePanicReport.Artifact,
	}
	parsePanicPayloadLine := "[GWC structured panic]"
	if parsePanicPayloadEncoded, parsePanicPayloadErr := json.Marshal(parsePanicPayload); parsePanicPayloadErr == nil {
		parsePanicPayloadLine += " " + string(parsePanicPayloadEncoded)
	}

	isPanicEmitted := false
	isPanicGrouped := false
	if _, parseGroupErr := parseBrowserConsole.Call("groupCollapsed", parsePanicHeader); parseGroupErr == nil {
		isPanicEmitted = true
		isPanicGrouped = true
	}
	if _, parseErrorErr := parseBrowserConsole.Call("error", parsePanicReport.Formatted); parseErrorErr == nil {
		isPanicEmitted = true
	}
	if _, parseLogErr := parseBrowserConsole.Call("log", parsePanicPayloadLine); parseLogErr == nil {
		isPanicEmitted = true
	}
	if isPanicGrouped {
		_, _ = parseBrowserConsole.Call("groupEnd")
	}
	return isPanicEmitted
}
