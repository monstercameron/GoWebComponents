package devtools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

var serializationBoundaryInspection struct {
	mu    sync.RWMutex
	state BoundaryInspection
}

// SetSerializationBoundaryInspection stores app-owned boundary inspection state.
func SetSerializationBoundaryInspection(parseState BoundaryInspection) {
	serializationBoundaryInspection.mu.Lock()
	defer serializationBoundaryInspection.mu.Unlock()
	serializationBoundaryInspection.state = cloneBoundaryInspection(parseState)
}

// ResetSerializationBoundaryInspection clears the app-owned boundary inspection state.
func ResetSerializationBoundaryInspection() {
	SetSerializationBoundaryInspection(BoundaryInspection{})
}

// InspectSerializationBoundaries returns the current app-owned boundary inspection state.
func InspectSerializationBoundaries() BoundaryInspection {
	serializationBoundaryInspection.mu.RLock()
	defer serializationBoundaryInspection.mu.RUnlock()
	return cloneBoundaryInspection(serializationBoundaryInspection.state)
}

// InspectBootstrapBoundaries converts an SSR bootstrap payload into a boundary inspection snapshot.
func InspectBootstrapBoundaries(parseBootstrap ui.SSRBootstrap) (BoundaryInspection, error) {
	parseReport, parseErr := ui.InspectSSRBootstrapSize(parseBootstrap, ui.SSRBootstrapBudget{})
	if parseErr != nil {
		return BoundaryInspection{}, parseErr
	}

	parseInspection := BoundaryInspection{
		Entries: []Boundary{{
			Name:        "ssr.bootstrap",
			Kind:        "ssr-bootstrap",
			Direction:   "server-to-client",
			Transport:   parseReport.Recommendation,
			Encoding:    "json",
			Scope:       "app",
			Status:      boundaryStatus(parseReport),
			SizeBytes:   parseReport.JSONPayloadBytes,
			InlineBytes: parseReport.InlineScriptBytes,
			BinaryBytes: parseReport.BinaryPayloadBytes,
			Notes:       appendBoundaryNotes(parseReport.Warnings, parseReport.Errors),
			Rejected:    append([]string(nil), parseReport.Errors...),
		}},
	}

	for _, parsePayload := range parseReport.Payloads {
		parseRaw := parseBootstrap.Data[parsePayload.Key]
		parseEntry := Boundary{
			Name:      parsePayload.Key,
			Kind:      "ssr-payload",
			Direction: "server-to-client",
			Transport: parseReport.Recommendation,
			Encoding:  string(parsePayload.Encoding),
			Scope:     string(parsePayload.Scope),
			Target:    parsePayload.Target,
			Status:    "observed",
			SizeBytes: approximateBoundarySize(parseRaw),
			Notes: []string{
				fmt.Sprintf("kind=%s", parsePayload.Kind),
				fmt.Sprintf("reuse=%s", parsePayload.ReusePolicy),
			},
		}
		if parsePayload.Version > 0 {
			parseEntry.Notes = append(parseEntry.Notes, fmt.Sprintf("version=%d", parsePayload.Version))
		}
		if strings.TrimSpace(parsePayload.Revision) != "" {
			parseEntry.Notes = append(parseEntry.Notes, "revision="+parsePayload.Revision)
		}
		if parsePayload.Legacy {
			parseEntry.Status = "downgraded"
			parseEntry.Downgraded = []string{"legacy bootstrap payload envelope"}
		}
		parseInspection.Entries = append(parseInspection.Entries, parseEntry)
	}

	return parseInspection, nil
}

func appendBoundaryNotes(parseWarnings []string, parseErrors []string) []string {
	parseNotes := make([]string, 0, len(parseWarnings)+len(parseErrors))
	for _, parseWarning := range parseWarnings {
		parseNotes = append(parseNotes, "warning: "+parseWarning)
	}
	for _, parseFailure := range parseErrors {
		parseNotes = append(parseNotes, "error: "+parseFailure)
	}
	return parseNotes
}

func boundaryStatus(parseReport ui.SSRBootstrapSizeReport) string {
	if len(parseReport.Errors) > 0 {
		return "rejected"
	}
	if len(parseReport.Warnings) > 0 {
		return "warning"
	}
	return "observed"
}

func approximateBoundarySize(parseRaw any) int {
	if parseRaw == nil {
		return 0
	}
	switch parseTyped := parseRaw.(type) {
	case []byte:
		return len(parseTyped)
	case string:
		return len(parseTyped)
	}
	parseEncoded, parseErr := json.Marshal(parseRaw)
	if parseErr != nil {
		return 0
	}
	return len(parseEncoded)
}

func cloneBoundaryInspection(parseState BoundaryInspection) BoundaryInspection {
	if len(parseState.Entries) == 0 {
		return BoundaryInspection{}
	}
	parseCloned := BoundaryInspection{Entries: make([]Boundary, len(parseState.Entries))}
	for parseI, parseEntry := range parseState.Entries {
		parseCloned.Entries[parseI] = Boundary{
			Name:          parseEntry.Name,
			Kind:          parseEntry.Kind,
			Direction:     parseEntry.Direction,
			Transport:     parseEntry.Transport,
			Encoding:      parseEntry.Encoding,
			Scope:         parseEntry.Scope,
			Target:        parseEntry.Target,
			Status:        parseEntry.Status,
			CorrelationID: parseEntry.CorrelationID,
			SizeBytes:     parseEntry.SizeBytes,
			InlineBytes:   parseEntry.InlineBytes,
			BinaryBytes:   parseEntry.BinaryBytes,
			Notes:         append([]string(nil), parseEntry.Notes...),
			Redacted:      append([]string(nil), parseEntry.Redacted...),
			Downgraded:    append([]string(nil), parseEntry.Downgraded...),
			Rejected:      append([]string(nil), parseEntry.Rejected...),
		}
	}
	return parseCloned
}
