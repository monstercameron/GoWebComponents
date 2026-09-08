//go:build js && wasm

package runtime

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
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

	parsePanicMessage := buildPanicConsoleMessage(parsePanicReport)
	parsePanicRecord := buildPanicConsoleRecord(parsePanicReport, parsePanicMessage)
	emitBrowserPanicEvent(parsePanicReport, parsePanicMessage)

	isPanicEmitted := false
	isPanicGrouped := false
	isPanicPayloadEmitted := false
	if _, parseGroupErr := parseBrowserConsole.Call("groupCollapsed", parsePanicMessage); parseGroupErr == nil {
		isPanicEmitted = true
		isPanicGrouped = true
	}
	if _, parseErrorErr := parseBrowserConsole.Call("error", parsePanicRecord); parseErrorErr == nil {
		isPanicEmitted = true
		isPanicPayloadEmitted = true
	}
	if !isPanicPayloadEmitted {
		if _, parseFallbackErr := parseBrowserConsole.Call("log", parsePanicRecord); parseFallbackErr == nil {
			isPanicEmitted = true
			isPanicPayloadEmitted = true
		}
	}
	if !isPanicPayloadEmitted {
		if _, parseTextErr := parseBrowserConsole.Call("error", strings.TrimSpace(parsePanicReport.Formatted)); parseTextErr == nil {
			isPanicEmitted = true
			isPanicPayloadEmitted = true
		}
	}
	if !isPanicPayloadEmitted {
		if _, parseTextFallbackErr := parseBrowserConsole.Call("log", strings.TrimSpace(parsePanicReport.Formatted)); parseTextFallbackErr == nil {
			isPanicEmitted = true
			isPanicPayloadEmitted = true
		}
	}
	if isPanicGrouped {
		if _, parseDetailErr := parseBrowserConsole.Call("log", strings.TrimSpace(parsePanicReport.Formatted)); parseDetailErr == nil {
			isPanicEmitted = true
		}
	}
	if isPanicGrouped {
		_, _ = parseBrowserConsole.Call("groupEnd")
	}
	return isPanicEmitted
}

// emitBrowserPanicEvent publishes the same structured panic record for dev-only
// tooling that wants to avoid scraping console output.
func emitBrowserPanicEvent(parsePanicReport PanicReport, parsePanicMessage string) {
	defer func() { _ = recover() }()
	parseCustomEvent := js.Global().Get("CustomEvent")
	if parseCustomEvent.IsUndefined() || parseCustomEvent.IsNull() {
		return
	}
	parsePanicAttributes := js.Global().Get("Object").New()
	parsePanicAttributes.Set("code", parsePanicReport.Code)
	parsePanicAttributes.Set("docs", parsePanicReport.Docs)
	parsePanicAttributes.Set("error", parsePanicReport.Summary)
	parsePanicAttributes.Set("next", parsePanicReport.Remediation)
	parsePanicAttributes.Set("path", parsePanicReport.Path)
	parsePanicAttributes.Set("phase", string(parsePanicReport.Phase))
	parsePanicAttributes.Set("runtime", parsePanicReport.Consequence)
	parsePanicAttributes.Set("source", parsePanicReport.Source)
	parsePanicAttributes.Set("subject", parsePanicReport.Subject)
	parsePanicAttributes.Set("where", parsePanicReport.Where)
	parsePanicAttributes.Set("appFrames", panicStringArrayJSValue(parsePanicReport.AppFrames))
	parsePanicAttributes.Set("componentStack", panicStringArrayJSValue(parsePanicReport.ComponentStack))
	parsePanicAttributes.Set("frameworkFrames", panicStringArrayJSValue(parsePanicReport.FrameworkFrames))
	parsePanicAttributes.Set("platformFrames", panicStringArrayJSValue(parsePanicReport.PlatformFrames))

	parsePanicRecord := js.Global().Get("Object").New()
	parsePanicRecord.Set("attributes", parsePanicAttributes)
	parsePanicRecord.Set("code", parsePanicReport.Code)
	parsePanicRecord.Set("docs", parsePanicReport.Docs)
	parsePanicRecord.Set("error", parsePanicReport.Summary)
	parsePanicRecord.Set("formatted", strings.TrimSpace(parsePanicReport.Formatted))
	parsePanicRecord.Set("level", "error")
	parsePanicRecord.Set("message", strings.TrimSpace(parsePanicMessage))
	parsePanicRecord.Set("next", parsePanicReport.Remediation)
	parsePanicRecord.Set("path", parsePanicReport.Path)
	parsePanicRecord.Set("phase", string(parsePanicReport.Phase))
	parsePanicRecord.Set("runtime", parsePanicReport.Consequence)
	parsePanicRecord.Set("scope", "runtime.panic")
	parsePanicRecord.Set("severity_number", 17)
	parsePanicRecord.Set("severity_text", "ERROR")
	parsePanicRecord.Set("source", parsePanicReport.Source)
	parsePanicRecord.Set("subject", parsePanicReport.Subject)
	parsePanicRecord.Set("where", parsePanicReport.Where)

	parseInit := js.Global().Get("Object").New()
	parseInit.Set("detail", parsePanicRecord)
	js.Global().Call("dispatchEvent", parseCustomEvent.New("gwc:runtime-panic", parseInit))
}

func panicStringArrayJSValue(parseValues []string) js.Value {
	parseArray := js.Global().Get("Array").New()
	for _, parseValue := range parseValues {
		parseArray.Call("push", parseValue)
	}
	return parseArray
}

// buildPanicConsoleMessage returns the browser-console headline for one wrapped panic report.
func buildPanicConsoleMessage(parsePanicReport PanicReport) string {
	parsePanicMessage := fmt.Sprintf("[%s] %s panic in %s", parsePanicReport.Code, parsePanicReport.Phase, parsePanicReport.Subject)
	if parsePanicSummary := strings.TrimSpace(parsePanicReport.Summary); parsePanicSummary != "" {
		parsePanicMessage += ": " + parsePanicSummary
	}
	return parsePanicMessage
}

// buildPanicConsoleRecord returns one structured error record for browser-console panic capture.
func buildPanicConsoleRecord(parsePanicReport PanicReport, parsePanicMessage string) map[string]any {
	parsePanicAttributes := buildPanicConsoleAttributes(parsePanicReport)
	parsePanicRecord := map[string]any{
		"attributes":      parsePanicAttributes,
		"code":            parsePanicReport.Code,
		"docs":            parsePanicReport.Docs,
		"error":           parsePanicReport.Summary,
		"formatted":       strings.TrimSpace(parsePanicReport.Formatted),
		"level":           "error",
		"message":         strings.TrimSpace(parsePanicMessage),
		"next":            parsePanicReport.Remediation,
		"path":            parsePanicReport.Path,
		"phase":           string(parsePanicReport.Phase),
		"runtime":         parsePanicReport.Consequence,
		"scope":           "runtime.panic",
		"severity_number": 17,
		"severity_text":   "ERROR",
		"source":          parsePanicReport.Source,
		"subject":         parsePanicReport.Subject,
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
		"where":           parsePanicReport.Where,
	}
	for parseFieldKey, parseFieldValue := range parsePanicAttributes {
		parsePanicRecord[parseFieldKey] = parseFieldValue
	}
	return parsePanicRecord
}

// buildPanicConsoleAttributes returns the panic-specific payload fields attached to one console record.
func buildPanicConsoleAttributes(parsePanicReport PanicReport) map[string]any {
	return map[string]any{
		"code":            parsePanicReport.Code,
		"docs":            parsePanicReport.Docs,
		"error":           parsePanicReport.Summary,
		"next":            parsePanicReport.Remediation,
		"path":            parsePanicReport.Path,
		"phase":           string(parsePanicReport.Phase),
		"runtime":         parsePanicReport.Consequence,
		"source":          parsePanicReport.Source,
		"subject":         parsePanicReport.Subject,
		"where":           parsePanicReport.Where,
		"appFrames":       append([]string(nil), parsePanicReport.AppFrames...),
		"artifact":        parsePanicReport.Artifact,
		"componentStack":  append([]string(nil), parsePanicReport.ComponentStack...),
		"frameworkFrames": append([]string(nil), parsePanicReport.FrameworkFrames...),
		"platformFrames":  append([]string(nil), parsePanicReport.PlatformFrames...),
	}
}
