//go:build js && wasm
// +build js,wasm

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/interop"
)

func emitBrowserPanicReport(report PanicReport) bool {
	global, err := interop.GetGlobalThis()
	if err != nil {
		return false
	}
	console := global.Get("console")
	if !console.Present() {
		return false
	}

	header := fmt.Sprintf("[%s] %s panic in %s", report.Code, report.Phase, report.Subject)
	if summary := strings.TrimSpace(report.Summary); summary != "" {
		header += ": " + summary
	}
	payload := map[string]any{
		"source":          report.Source,
		"phase":           string(report.Phase),
		"subject":         report.Subject,
		"where":           report.Where,
		"path":            report.Path,
		"error":           report.Summary,
		"runtime":         report.Consequence,
		"next":            report.Remediation,
		"docs":            report.Docs,
		"code":            report.Code,
		"componentStack":  report.ComponentStack,
		"appFrames":       report.AppFrames,
		"frameworkFrames": report.FrameworkFrames,
		"platformFrames":  report.PlatformFrames,
	}
	payloadLine := "[GWC structured panic]"
	if encoded, err := json.Marshal(payload); err == nil {
		payloadLine += " " + string(encoded)
	}

	emitted := false
	grouped := false
	if _, err := console.Call("groupCollapsed", header); err == nil {
		emitted = true
		grouped = true
	}
	if _, err := console.Call("error", report.Formatted); err == nil {
		emitted = true
	}
	if _, err := console.Call("log", payloadLine); err == nil {
		emitted = true
	}
	if grouped {
		_, _ = console.Call("groupEnd")
	}
	return emitted
}
