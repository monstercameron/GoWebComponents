package devtools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/ui"
)

var serializationBoundaryInspection struct {
	mu    sync.RWMutex
	state BoundaryInspection
}

// SetSerializationBoundaryInspection stores app-owned boundary inspection state.
func SetSerializationBoundaryInspection(state BoundaryInspection) {
	serializationBoundaryInspection.mu.Lock()
	defer serializationBoundaryInspection.mu.Unlock()
	serializationBoundaryInspection.state = cloneBoundaryInspection(state)
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
func InspectBootstrapBoundaries(bootstrap ui.SSRBootstrap) (BoundaryInspection, error) {
	report, err := ui.AnalyzeSSRBootstrapSize(bootstrap, ui.SSRBootstrapBudget{})
	if err != nil {
		return BoundaryInspection{}, err
	}

	inspection := BoundaryInspection{
		Entries: []Boundary{{
			Name:        "ssr.bootstrap",
			Kind:        "ssr-bootstrap",
			Direction:   "server-to-client",
			Transport:   report.Recommendation,
			Encoding:    "json",
			Scope:       "app",
			Status:      boundaryStatus(report),
			SizeBytes:   report.JSONPayloadBytes,
			InlineBytes: report.InlineScriptBytes,
			BinaryBytes: report.BinaryPayloadBytes,
			Notes:       appendBoundaryNotes(report.Warnings, report.Errors),
			Rejected:    append([]string(nil), report.Errors...),
		}},
	}

	for _, payload := range report.Payloads {
		raw := bootstrap.Data[payload.Key]
		entry := Boundary{
			Name:      payload.Key,
			Kind:      "ssr-payload",
			Direction: "server-to-client",
			Transport: report.Recommendation,
			Encoding:  string(payload.Encoding),
			Scope:     string(payload.Scope),
			Target:    payload.Target,
			Status:    "observed",
			SizeBytes: approximateBoundarySize(raw),
			Notes: []string{
				fmt.Sprintf("kind=%s", payload.Kind),
				fmt.Sprintf("reuse=%s", payload.ReusePolicy),
			},
		}
		if payload.Version > 0 {
			entry.Notes = append(entry.Notes, fmt.Sprintf("version=%d", payload.Version))
		}
		if strings.TrimSpace(payload.Revision) != "" {
			entry.Notes = append(entry.Notes, "revision="+payload.Revision)
		}
		if payload.Legacy {
			entry.Status = "downgraded"
			entry.Downgraded = []string{"legacy bootstrap payload envelope"}
		}
		inspection.Entries = append(inspection.Entries, entry)
	}

	return inspection, nil
}

func appendBoundaryNotes(warnings []string, errors []string) []string {
	notes := make([]string, 0, len(warnings)+len(errors))
	for _, warning := range warnings {
		notes = append(notes, "warning: "+warning)
	}
	for _, failure := range errors {
		notes = append(notes, "error: "+failure)
	}
	return notes
}

func boundaryStatus(report ui.SSRBootstrapSizeReport) string {
	if len(report.Errors) > 0 {
		return "rejected"
	}
	if len(report.Warnings) > 0 {
		return "warning"
	}
	return "observed"
}

func approximateBoundarySize(raw interface{}) int {
	if raw == nil {
		return 0
	}
	switch typed := raw.(type) {
	case []byte:
		return len(typed)
	case string:
		return len(typed)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return 0
	}
	return len(encoded)
}

func cloneBoundaryInspection(state BoundaryInspection) BoundaryInspection {
	if len(state.Entries) == 0 {
		return BoundaryInspection{}
	}
	cloned := BoundaryInspection{Entries: make([]Boundary, len(state.Entries))}
	for i, entry := range state.Entries {
		cloned.Entries[i] = Boundary{
			Name:          entry.Name,
			Kind:          entry.Kind,
			Direction:     entry.Direction,
			Transport:     entry.Transport,
			Encoding:      entry.Encoding,
			Scope:         entry.Scope,
			Target:        entry.Target,
			Status:        entry.Status,
			CorrelationID: entry.CorrelationID,
			SizeBytes:     entry.SizeBytes,
			InlineBytes:   entry.InlineBytes,
			BinaryBytes:   entry.BinaryBytes,
			Notes:         append([]string(nil), entry.Notes...),
			Redacted:      append([]string(nil), entry.Redacted...),
			Downgraded:    append([]string(nil), entry.Downgraded...),
			Rejected:      append([]string(nil), entry.Rejected...),
		}
	}
	return cloned
}
