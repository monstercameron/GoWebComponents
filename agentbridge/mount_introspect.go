package agentbridge

import (
	"sort"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// writeRemount re-mounts a previously-mounted component, reversing a
// bridge.unmount. It reserves the id, renders the component with the retained
// props, and rolls the reservation back on failure. Mirrors writeHandleMount's
// reservation discipline.
func writeRemount(parseID string, parseComponent string, parseSelector string, parseProps map[string]any) *EnvelopeError {
	writeMountMu.Lock()
	parseFactory := writeMountComponents[parseComponent]
	if parseFactory == nil {
		writeMountMu.Unlock()
		return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "remount: component " + parseComponent + " is not registered"}
	}
	if _, parseExists := writeMountedRoots[parseID]; parseExists {
		writeMountMu.Unlock()
		return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "remount: id " + parseID + " is already mounted"}
	}
	writeMountedRoots[parseID] = writeMountedRoot{Component: parseComponent, Selector: parseSelector, Props: parseProps}
	writeMountMu.Unlock()

	parseElement := parseFactory(parseProps)
	if parseElement == nil {
		writeMountReleaseReservation(parseID)
		return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "remount: component " + parseComponent + " rendered nil"}
	}
	if parseErr := writeRenderToSelector(parseSelector, parseElement); parseErr != nil {
		writeMountReleaseReservation(parseID)
		return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "remount: " + parseErr.Error()}
	}
	runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
	return nil
}

// writeMountReleaseReservation removes a mount-id reservation under the mount
// mutex. bridge.mount reserves an id before rendering so concurrent mounts of
// the same id cannot both run the factory; this rolls the reservation back when
// rendering fails.
func writeMountReleaseReservation(parseID string) {
	writeMountMu.Lock()
	defer writeMountMu.Unlock()
	delete(writeMountedRoots, parseID)
}

// ListMountComponents returns the sorted names of every component registered
// for bridge.mount via RegisterMountComponent. It is the discovery surface an
// agent uses to learn what bridge.mount accepts, instead of guessing component
// names.
func ListMountComponents() []string {
	writeMountMu.Lock()
	defer writeMountMu.Unlock()
	parseNames := make([]string, 0, len(writeMountComponents))
	for parseName := range writeMountComponents {
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)
	return parseNames
}
