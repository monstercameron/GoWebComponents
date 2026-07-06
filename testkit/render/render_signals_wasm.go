//go:build js && wasm

package render

import (
	"sort"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// BuildOverlaySurfaces returns all rendered overlay surfaces sorted by depth.
func (parseF *Fixture) BuildOverlaySurfaces() []OverlaySurface {
	if parseF == nil || parseF.container == nil {
		return nil
	}
	parseMatches := collectNodes(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if len(parseMatches) == 0 {
		return nil
	}
	parseSurfaces := make([]OverlaySurface, 0, len(parseMatches))
	for _, parseNode := range parseMatches {
		parseSurfaces = append(parseSurfaces, OverlaySurface{
			SurfaceID:           strings.TrimSpace(parseNode.Attrs["id"]),
			Kind:                strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]),
			Depth:               buildOverlayInt(parseNode.Attrs["data-overlay-depth"]),
			HandlesEscape:       buildOverlayBool(parseNode.Attrs["data-overlay-handles-escape"]),
			HandlesOutsideClick: buildOverlayBool(parseNode.Attrs["data-overlay-handles-outside"]),
			TrapFocusOwner:      buildOverlayBool(parseNode.Attrs["data-overlay-trap-owner"]),
			IsModal:             strings.EqualFold(strings.TrimSpace(parseNode.Attrs["aria-modal"]), "true"),
			PortalTargetID:      buildOverlayPortalTargetID(parseNode),
		})
	}
	sort.SliceStable(parseSurfaces, func(parseLeft, parseRight int) bool {
		if parseSurfaces[parseLeft].Depth == parseSurfaces[parseRight].Depth {
			return parseSurfaces[parseLeft].SurfaceID < parseSurfaces[parseRight].SurfaceID
		}
		return parseSurfaces[parseLeft].Depth < parseSurfaces[parseRight].Depth
	})
	return parseSurfaces
}

// BuildOverlayEscapeSurfaceID returns the topmost escape-handling overlay surface id.
func (parseF *Fixture) BuildOverlayEscapeSurfaceID() string {
	return buildOverlayOwnerSurfaceID(parseF.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.HandlesEscape
	})
}

// BuildOverlayOutsideSurfaceID returns the topmost outside-click-handling overlay surface id.
func (parseF *Fixture) BuildOverlayOutsideSurfaceID() string {
	return buildOverlayOwnerSurfaceID(parseF.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.HandlesOutsideClick
	})
}

// BuildOverlayFocusSurfaceID returns the topmost trap-focus owner overlay surface id.
func (parseF *Fixture) BuildOverlayFocusSurfaceID() string {
	return buildOverlayOwnerSurfaceID(parseF.BuildOverlaySurfaces(), func(parseSurface OverlaySurface) bool {
		return parseSurface.TrapFocusOwner
	})
}

// BuildOverlayScrollLockActive reports whether overlay-driven scroll lock is active.
func (parseF *Fixture) BuildOverlayScrollLockActive() bool {
	parseOverflow := strings.TrimSpace(parseF.BuildOverlayBodyOverflow())
	if strings.EqualFold(parseOverflow, "hidden") {
		return true
	}
	for _, parseSurface := range parseF.BuildOverlaySurfaces() {
		if parseSurface.IsModal {
			return true
		}
	}
	return false
}

// BuildOverlayBodyOverflow reports the current browser document body overflow style.
func (parseF *Fixture) BuildOverlayBodyOverflow() string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseBody := parseDocument.Get("body")
	if !parseBody.Truthy() {
		return ""
	}
	parseStyle := parseBody.Get("style")
	if !parseStyle.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseStyle.Get("overflow").String())
}

// BuildOverlayPortalTargetID resolves one overlay surface to the nearest ancestor id.
func (parseF *Fixture) BuildOverlayPortalTargetID(parseSurfaceID string) string {
	if parseF == nil || parseF.container == nil {
		return ""
	}
	parseSurface := findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["id"]) == strings.TrimSpace(parseSurfaceID) && strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if parseSurface == nil {
		return ""
	}
	return buildOverlayPortalTargetID(parseSurface)
}

// HandleOverlayOutsideClick dispatches one outside-click dismissal through the overlay backdrop.
func (parseF *Fixture) HandleOverlayOutsideClick(parseSurfaceID string) bool {
	parseF.tb.Helper()
	parseF.requireActive()
	parseSurface := findNode(parseF.container, func(parseNode *mockdom.MockDOMNode) bool {
		return strings.TrimSpace(parseNode.Attrs["id"]) == strings.TrimSpace(parseSurfaceID) && strings.TrimSpace(parseNode.Attrs["data-overlay-kind"]) != ""
	})
	if parseSurface == nil || parseSurface.Parent == nil {
		return false
	}
	if parseSurface.Parent.Props["onclick"] == nil {
		return false
	}
	parseF.dispatch(parseSurface.Parent, "onclick", Event{})
	return true
}

// BuildDiagnostics returns structured runtime diagnostics captured for this fixture run.
func (parseF *Fixture) BuildDiagnostics() []DiagnosticSignal {
	parseDiagnostics := runtime.GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		return nil
	}
	parseSignals := make([]DiagnosticSignal, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		parseSignals = append(parseSignals, DiagnosticSignal{
			Source:         parseDiagnostic.Source,
			Severity:       string(parseDiagnostic.Severity),
			Classification: string(parseDiagnostic.Classification),
			Code:           parseDiagnostic.Code,
			Message:        parseDiagnostic.Message,
			Count:          parseDiagnostic.Count,
			Path:           parseDiagnostic.Path,
			Recoverable:    parseDiagnostic.Recoverable,
			TopFrame:       parseDiagnostic.TopFrame,
			Consequence:    parseDiagnostic.Consequence,
			ComponentStack: append([]string(nil), parseDiagnostic.ComponentStack...),
			Fields:         cloneSignalFields(parseDiagnostic.Fields),
		})
	}
	return parseSignals
}

// BuildLogs returns buffered runtime logs captured for this fixture run.
func (parseF *Fixture) BuildLogs() []LogSignal {
	parseLogs := runtime.GetLogs()
	if len(parseLogs) == 0 {
		return nil
	}
	parseSignals := make([]LogSignal, 0, len(parseLogs))
	for _, parseLog := range parseLogs {
		parseSignals = append(parseSignals, LogSignal{
			Domain:         parseLog.Domain,
			Level:          string(parseLog.Level),
			Classification: string(parseLog.Classification),
			Code:           parseLog.Code,
			Message:        parseLog.Message,
			Timestamp:      parseLog.Timestamp,
			CorrelationID:  parseLog.CorrelationID,
			Recoverable:    parseLog.Recoverable,
			TopFrame:       parseLog.TopFrame,
			Consequence:    parseLog.Consequence,
			Fields:         cloneSignalFields(parseLog.Fields),
		})
	}
	return parseSignals
}

// BuildRenderCounts returns structured component render-count signals from profiling snapshots.
func (parseF *Fixture) BuildRenderCounts() []RenderCountSignal {
	parseSnapshot := runtime.GetGlobalRuntime().Inspect()
	parseTraces := parseSnapshot.Profiling.ComponentRenders
	if len(parseTraces) == 0 {
		return nil
	}
	parseSignals := make([]RenderCountSignal, 0, len(parseTraces))
	for _, parseTrace := range parseTraces {
		parseSignals = append(parseSignals, RenderCountSignal{
			Name:                    parseTrace.Name,
			Path:                    parseTrace.Path,
			RenderCount:             parseTrace.RenderCount,
			RerenderCount:           parseTrace.RerenderCount,
			LastTrigger:             parseTrace.LastTrigger,
			TotalRenderDurationNs:   parseTrace.TotalRenderDurationNs,
			AverageRenderDurationNs: parseTrace.AverageRenderDurationNs,
		})
	}
	sort.SliceStable(parseSignals, func(parseLeft, parseRight int) bool {
		if parseSignals[parseLeft].RenderCount == parseSignals[parseRight].RenderCount {
			return parseSignals[parseLeft].Path < parseSignals[parseRight].Path
		}
		return parseSignals[parseLeft].RenderCount > parseSignals[parseRight].RenderCount
	})
	return parseSignals
}

// BuildWarningDiagnostics returns diagnostics whose severity is warning.
func (parseF *Fixture) BuildWarningDiagnostics() []DiagnosticSignal {
	parseDiagnostics := parseF.BuildDiagnostics()
	if len(parseDiagnostics) == 0 {
		return nil
	}
	parseWarnings := make([]DiagnosticSignal, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.EqualFold(strings.TrimSpace(parseDiagnostic.Severity), "warning") {
			parseWarnings = append(parseWarnings, parseDiagnostic)
		}
	}
	return parseWarnings
}

// BuildWarningLogs returns logs whose level is warn.
func (parseF *Fixture) BuildWarningLogs() []LogSignal {
	parseLogs := parseF.BuildLogs()
	if len(parseLogs) == 0 {
		return nil
	}
	parseDiagnostics := parseF.BuildWarningDiagnostics()
	parseWarnings := make([]LogSignal, 0, len(parseLogs))
	for _, parseLog := range parseLogs {
		if strings.EqualFold(strings.TrimSpace(parseLog.Level), "warn") && !isDiagnosticMirroredLog(parseDiagnostics, parseLog) {
			parseWarnings = append(parseWarnings, parseLog)
		}
	}
	return parseWarnings
}

// ApplyDiagnosticCode asserts that one diagnostic with the requested code exists.
func (parseF *Fixture) ApplyDiagnosticCode(parseCode string) DiagnosticSignal {
	parseF.tb.Helper()
	parseCode = strings.TrimSpace(parseCode)
	for _, parseDiagnostic := range parseF.BuildDiagnostics() {
		if strings.TrimSpace(parseDiagnostic.Code) == parseCode {
			return parseDiagnostic
		}
	}
	parseF.tb.Fatalf("render fixture could not find diagnostic code %q", parseCode)
	return DiagnosticSignal{}
}

// ApplyDiagnosticMessage asserts that one diagnostic message contains the provided fragment.
func (parseF *Fixture) ApplyDiagnosticMessage(parseFragment string) DiagnosticSignal {
	parseF.tb.Helper()
	parseFragment = strings.TrimSpace(parseFragment)
	for _, parseDiagnostic := range parseF.BuildDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, parseFragment) {
			return parseDiagnostic
		}
	}
	parseF.tb.Fatalf("render fixture could not find diagnostic message fragment %q", parseFragment)
	return DiagnosticSignal{}
}

// ApplyLogCode asserts that one buffered log with the requested code exists.
func (parseF *Fixture) ApplyLogCode(parseCode string) LogSignal {
	parseF.tb.Helper()
	parseCode = strings.TrimSpace(parseCode)
	for _, parseLog := range parseF.BuildLogs() {
		if strings.TrimSpace(parseLog.Code) == parseCode {
			return parseLog
		}
	}
	parseF.tb.Fatalf("render fixture could not find log code %q", parseCode)
	return LogSignal{}
}

// ApplyLogMessage asserts that one buffered log message contains the provided fragment.
func (parseF *Fixture) ApplyLogMessage(parseFragment string) LogSignal {
	parseF.tb.Helper()
	parseFragment = strings.TrimSpace(parseFragment)
	for _, parseLog := range parseF.BuildLogs() {
		if strings.Contains(parseLog.Message, parseFragment) {
			return parseLog
		}
	}
	parseF.tb.Fatalf("render fixture could not find log message fragment %q", parseFragment)
	return LogSignal{}
}

// ApplyRenderCountMax asserts one component's render count does not exceed max.
func (parseF *Fixture) ApplyRenderCountMax(parseComponent string, parseMax int) RenderCountSignal {
	parseF.tb.Helper()
	parseSignal := applyRenderCountSignal(parseF.BuildRenderCounts(), parseComponent)
	if strings.TrimSpace(parseSignal.Name) == "" && strings.TrimSpace(parseSignal.Path) == "" {
		parseF.tb.Fatalf("render fixture could not resolve render-count signal for component %q", strings.TrimSpace(parseComponent))
	}
	if parseSignal.RenderCount > parseMax {
		parseF.tb.Fatalf("render fixture component %q exceeded max render count %d with %d renders (path=%q)", strings.TrimSpace(parseComponent), parseMax, parseSignal.RenderCount, parseSignal.Path)
	}
	return parseSignal
}

// ApplyRenderRerenderMax asserts one component's rerender count does not exceed max.
func (parseF *Fixture) ApplyRenderRerenderMax(parseComponent string, parseMax int) RenderCountSignal {
	parseF.tb.Helper()
	parseSignal := applyRenderCountSignal(parseF.BuildRenderCounts(), parseComponent)
	if strings.TrimSpace(parseSignal.Name) == "" && strings.TrimSpace(parseSignal.Path) == "" {
		parseF.tb.Fatalf("render fixture could not resolve rerender signal for component %q", strings.TrimSpace(parseComponent))
	}
	if parseSignal.RerenderCount > parseMax {
		parseF.tb.Fatalf("render fixture component %q exceeded max rerender count %d with %d rerenders (path=%q)", strings.TrimSpace(parseComponent), parseMax, parseSignal.RerenderCount, parseSignal.Path)
	}
	return parseSignal
}

// ApplyWarningCountMax asserts warnings across diagnostics and logs do not exceed max.
func (parseF *Fixture) ApplyWarningCountMax(parseMax int) {
	parseF.tb.Helper()
	parseWarningDiagnostics := parseF.BuildWarningDiagnostics()
	parseWarningLogs := parseF.BuildWarningLogs()
	parseWarningCount := len(parseWarningDiagnostics) + len(parseWarningLogs)
	if parseWarningCount > parseMax {
		parseF.tb.Fatalf("render fixture warning count %d exceeds max %d (diagnostics=%d logs=%d)", parseWarningCount, parseMax, len(parseWarningDiagnostics), len(parseWarningLogs))
	}
}

// ApplyWarningNone asserts no warning diagnostics or warn-level logs were emitted.
func (parseF *Fixture) ApplyWarningNone() {
	parseF.tb.Helper()
	parseF.ApplyWarningCountMax(0)
}

// buildOverlayBool parses one overlay boolean attribute value.
func buildOverlayBool(parseValue string) bool {
	return strings.EqualFold(strings.TrimSpace(parseValue), "true")
}

// buildOverlayInt parses one overlay integer attribute value.
func buildOverlayInt(parseValue string) int {
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
	if parseErr != nil {
		return 0
	}
	return parseParsed
}

// buildOverlayOwnerSurfaceID resolves the topmost overlay id for one ownership selector.
func buildOverlayOwnerSurfaceID(parseSurfaces []OverlaySurface, parseMatch func(OverlaySurface) bool) string {
	for parseIndex := len(parseSurfaces) - 1; parseIndex >= 0; parseIndex-- {
		if parseMatch(parseSurfaces[parseIndex]) {
			return parseSurfaces[parseIndex].SurfaceID
		}
	}
	return ""
}

// buildOverlayPortalTargetID resolves the nearest ancestor id for one overlay node.
func buildOverlayPortalTargetID(parseSurface *mockdom.MockDOMNode) string {
	if parseSurface == nil {
		return ""
	}
	for parseParent := parseSurface.Parent; parseParent != nil; parseParent = parseParent.Parent {
		parseID := strings.TrimSpace(parseParent.Attrs["id"])
		if parseID != "" {
			return parseID
		}
	}
	return ""
}

// cloneSignalFields clones one log or diagnostic field map.
func cloneSignalFields(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return map[string]string{}
	}
	parseCloned := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

// applyRenderCountSignal resolves one render-count signal by component name or path.
func applyRenderCountSignal(parseSignals []RenderCountSignal, parseComponent string) RenderCountSignal {
	parseComponent = strings.TrimSpace(parseComponent)
	if len(parseSignals) == 0 {
		return RenderCountSignal{}
	}
	if parseComponent == "" {
		return parseSignals[0]
	}
	// Two-pass resolution: prefer an EXACT Name/Path match before falling back to
	// a substring match. parseSignals is sorted by render count descending, so a
	// single-pass substring match could resolve "Button" to an unrelated
	// "IconButton" that happens to have more renders — silently checking the wrong
	// component's budget (a false pass/fail).
	for _, parseSignal := range parseSignals {
		if parseSignal.Name == parseComponent || parseSignal.Path == parseComponent {
			return parseSignal
		}
	}
	for _, parseSignal := range parseSignals {
		if strings.Contains(parseSignal.Path, parseComponent) {
			return parseSignal
		}
	}
	return RenderCountSignal{}
}

// isDiagnosticMirroredLog reports whether one log entry is the structured-log mirror of one warning diagnostic.
func isDiagnosticMirroredLog(parseDiagnostics []DiagnosticSignal, parseLog LogSignal) bool {
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.TrimSpace(parseDiagnostic.Source) != strings.TrimSpace(parseLog.Domain) {
			continue
		}
		if strings.TrimSpace(parseDiagnostic.Code) != strings.TrimSpace(parseLog.Code) {
			continue
		}
		if strings.TrimSpace(parseDiagnostic.Message) != strings.TrimSpace(parseLog.Message) {
			continue
		}
		return true
	}
	return false
}
