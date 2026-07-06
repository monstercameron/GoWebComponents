package agentbridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/events"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/state"
)

// RegisterWriteCommands installs the four input-level mutation commands on the
// agent bridge:
//
//   - bridge.set-atom  — set one atom by id from a JSON value
//   - bridge.emit      — invoke a fiber's event handler by node ref + event name
//   - bridge.publish   — publish to the in-app events bus
//   - bridge.navigate  — navigate the router to a new path
//
// Every handler first checks IsAgentModeActive() and refuses with
// ErrorCodeForbidden when the app has not been launched in agent mode.
//
// RegisterWriteCommands must be called once at bridge install time (typically
// inside EnableAgentBridge), before any commands are dispatched.
func RegisterWriteCommands() {
	RegisterAgentCommand("bridge.set-atom", writeHandleSetAtom)
	RegisterAgentCommand("bridge.set-state", writeHandleSetState)
	RegisterAgentCommand("bridge.emit", writeHandleEmit)
	RegisterAgentCommand("bridge.publish", writeHandlePublish)
	RegisterAgentCommand("bridge.navigate", writeHandleNavigate)
	RegisterAgentCommand("bridge.mount", writeHandleMount)
	RegisterAgentCommand("bridge.unmount", writeHandleUnmount)
	RegisterAgentCommand("bridge.delete-atom", writeHandleDeleteAtom)
	// Audit trail + undo for the mutations registered above.
	RegisterSafetyCommands()
}

// MountComponentFactory renders a bridge-mountable component from JSON-decoded
// props. Apps opt components into bridge.mount by registering a factory.
type MountComponentFactory func(map[string]any) *runtime.Element

type writeMountedRoot struct {
	Component string
	Selector  string
	// Props are the decoded mount props, retained so a bridge.unmount can be
	// reversed by remounting the same component with the same props.
	Props map[string]any
}

var (
	writeMountMu sync.Mutex
	// writeMaxMountedRoots bounds the live agent-mounted root registry.
	writeMaxMountedRoots = 256
	writeMountComponents = map[string]MountComponentFactory{}
	writeMountedRoots    = map[string]writeMountedRoot{}
)

// RegisterMountComponent registers a component factory for bridge.mount.
func RegisterMountComponent(parseName string, parseFactory MountComponentFactory) {
	parseTrimmed := strings.TrimSpace(parseName)
	if parseTrimmed == "" || parseFactory == nil {
		return
	}
	writeMountMu.Lock()
	defer writeMountMu.Unlock()
	writeMountComponents[parseTrimmed] = parseFactory
}

// ---------------------------------------------------------------------------
// bridge.set-atom
// ---------------------------------------------------------------------------

// writeSetAtomPayload is the decoded shape of a bridge.set-atom command. When
// DryRun is set the command validates and returns the would-be change without
// applying it, so an agent can preview the effect (and confirm the atom exists
// and the type matches) before committing.
type writeSetAtomPayload struct {
	ID     string          `json:"id"`
	Value  json.RawMessage `json:"value"`
	DryRun bool            `json:"dryRun,omitempty"`
}

// writeSetAtomDryRunResult previews a set-atom that was not applied.
type writeSetAtomDryRunResult struct {
	DryRun bool   `json:"dryRun"`
	ID     string `json:"id"`
	From   any    `json:"from"`
	To     any    `json:"to"`
}

// writeHandleSetAtom handles the bridge.set-atom command. It decodes {"id":
// "…", "value": <raw json>} and applies the value to the named atom through
// the state.ApplySnapshot codec path (json.Unmarshal -> normalizeSnapshot),
// which is the same path the hotreload package uses. Unknown or unregistered
// atoms are rejected with ErrorCodeBadPayload and the atom value is not
// touched. Type-mismatched payloads that fail JSON unmarshalling are also
// rejected with ErrorCodeBadPayload.
func writeHandleSetAtom(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}

	parseDec, parseDecErr := writeDecodeSetAtom(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}

	// Decode the JSON value into a plain Go type via json.Unmarshal so that the
	// resulting value is the same generic map/slice/float64/string/bool/nil that
	// json.Unmarshal always produces. This matches what state.UnmarshalSnapshotJSON
	// does and is exactly the codec path hotreload/hotreload_wasm.go goes through
	// when it calls state.ApplySnapshot(parseSnapshot.State).
	var parseRawValue any
	if parseUnmarshalErr := json.Unmarshal(parseDec.Value, &parseRawValue); parseUnmarshalErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("set-atom: cannot decode value JSON for atom %q: %v", parseDec.ID, parseUnmarshalErr),
		}
	}

	// Verify the atom is registered before touching it. SnapshotAtoms reflects
	// all atoms that have ever been initialized in the current runtime.
	parseRt := runtime.GetGlobalRuntime()
	parseSnap := parseRt.SnapshotAtoms()
	parseCurrent, parseExists := parseSnap[parseDec.ID]
	if !parseExists {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("set-atom: atom %q is not registered in the current runtime", parseDec.ID),
		}
	}
	if parseSchemaErr := writeValidateRawValueAgainstSchema("set-atom", "value", parseDec.Value, controlJSONSchema(parseCurrent)); parseSchemaErr != nil {
		return nil, parseSchemaErr
	}

	// Fail closed on a type-family mismatch. The atom registry is type-erased
	// (stores `any`), so a wrong-typed write would be accepted here and only
	// blow up later at the typed UseAtom[T]().Get() site. Reject before
	// touching the atom so it keeps its prior value, the same guarantee the
	// todo promises. Number kinds are treated as one family because JSON
	// always decodes numbers to float64 regardless of the atom's Go type.
	if !writeAtomValueCompatible(parseCurrent, parseRawValue) {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("set-atom: value type %T is incompatible with atom %q's current type %T", parseRawValue, parseDec.ID, parseCurrent),
		}
	}

	// Dry run: validation passed, so report the would-be change without
	// applying it or recording a mutation.
	if parseDec.DryRun {
		parsePreview, parseMarshalErr := json.Marshal(writeSetAtomDryRunResult{
			DryRun: true, ID: parseDec.ID, From: parseCurrent, To: parseRawValue,
		})
		if parseMarshalErr != nil {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseMarshalErr.Error()}
		}
		return parsePreview, nil
	}

	// Apply as a single-entry snapshot through state.ApplySnapshot so that
	// subscribed fibers are scheduled for update exactly as they would be after
	// a hotreload restore.
	parseErr := state.ApplySnapshot(state.Snapshot{parseDec.ID: parseRawValue})
	if parseErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("set-atom: apply snapshot for atom %q failed: %v", parseDec.ID, parseErr),
		}
	}

	// Record the mutation with an undo that restores the captured prior value.
	parsePrior := parseCurrent
	parseAtomID := parseDec.ID
	RecordAgentMutation("bridge.set-atom", "set atom "+parseAtomID, func() *EnvelopeError {
		if parseRestoreErr := state.ApplySnapshot(state.Snapshot{parseAtomID: parsePrior}); parseRestoreErr != nil {
			return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "undo set-atom " + parseAtomID + ": " + parseRestoreErr.Error()}
		}
		runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
		return nil
	})

	parseRt.AdvanceAgentStateVersion()
	return writeEncodeOK()
}

// writeAtomValueCompatible reports whether a new JSON-decoded atom value is
// type-compatible with the atom's current value. It groups all numeric kinds
// into one family (JSON decodes every number to float64), and treats a nil
// current value as compatible with anything (no prior typed value to violate).
// Anything else must share the same reflect.Kind family.
func writeAtomValueCompatible(parseCurrent any, parseNew any) bool {
	if parseCurrent == nil || parseNew == nil {
		// A nil current value carries no type to violate; a nil new value
		// (JSON null) is a legal reset for any atom.
		return true
	}
	if writeValueFamily(parseCurrent) != writeValueFamily(parseNew) {
		return false
	}
	// The incoming value is always JSON-decoded, so a composite is a generic
	// []any / map[string]any. If the current atom holds a CONCRETE typed
	// composite (e.g. []models.Todo, map[string]int), that generic value would
	// fail the typed UseAtom[T]().Get() assertion and silently revert — so
	// reject it rather than report a false success.
	switch reflect.ValueOf(parseCurrent).Kind() {
	case reflect.Slice, reflect.Array:
		if _, parseOk := parseCurrent.([]any); !parseOk {
			return false
		}
	case reflect.Map:
		if _, parseOk := parseCurrent.(map[string]any); !parseOk {
			return false
		}
	}
	return true
}

// writeValueFamily maps a value to a coarse type family for compatibility
// checks: "number" for every numeric kind, otherwise the reflect.Kind string.
func writeValueFamily(parseValue any) string {
	parseKind := reflect.ValueOf(parseValue).Kind()
	switch parseKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	default:
		return parseKind.String()
	}
}

// writeDecodeSetAtom decodes and validates the set-atom payload.
func writeDecodeSetAtom(parseRaw json.RawMessage) (*writeSetAtomPayload, *EnvelopeError) {
	var parseDec writeSetAtomPayload
	if parseDecErr := json.Unmarshal(parseRaw, &parseDec); parseDecErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("set-atom: malformed payload: %v", parseDecErr),
		}
	}
	if parseDec.ID == "" {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "set-atom: missing required field \"id\"",
		}
	}
	if len(parseDec.Value) == 0 {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "set-atom: missing required field \"value\"",
		}
	}
	return &parseDec, nil
}

// ---------------------------------------------------------------------------
// bridge.set-state
// ---------------------------------------------------------------------------

type writeSetStatePayload struct {
	Ref   string          `json:"ref"`
	Slot  int             `json:"slot"`
	Value json.RawMessage `json:"value"`
}

func writeHandleSetState(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}
	parseDec, parseDecErr := writeDecodeSetState(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}
	var parseValue any
	if parseUnmarshalErr := json.Unmarshal(parseDec.Value, &parseValue); parseUnmarshalErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("set-state: cannot decode value JSON for ref %q slot %d: %v", parseDec.Ref, parseDec.Slot, parseUnmarshalErr),
		}
	}
	parseRt := runtime.GetGlobalRuntime()
	// Capture the prior slot value before writing so the set-state can be
	// recorded as a reversible mutation.
	parsePrior, parsePriorErr := runtime.GetAgentState(parseRt, parseDec.Ref, parseDec.Slot)
	parseErr := runtime.SetAgentState(parseRt, parseDec.Ref, parseDec.Slot, parseValue)
	if parseErr == nil {
		parseRt.AdvanceAgentStateVersion()
		parseLabel := fmt.Sprintf("set-state %s[%d]", parseDec.Ref, parseDec.Slot)
		if parsePriorErr == nil {
			parseRef := parseDec.Ref
			parseSlot := parseDec.Slot
			parsePriorVal := parsePrior
			RecordAgentMutation("bridge.set-state", parseLabel, func() *EnvelopeError {
				if parseUndoErr := runtime.SetAgentState(runtime.GetGlobalRuntime(), parseRef, parseSlot, parsePriorVal); parseUndoErr != nil {
					return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "undo set-state: " + parseUndoErr.Error()}
				}
				runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
				return nil
			})
		} else {
			// Could not read the prior value; record without an undo.
			RecordAgentMutation("bridge.set-state", parseLabel+" (prior value unavailable; not reversible)", nil)
		}
		return writeEncodeOK()
	}
	if errors.Is(parseErr, runtime.ErrAgentRefStale) {
		return nil, &EnvelopeError{Code: ErrorCodeStaleRef, Message: parseErr.Error()}
	}
	return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseErr.Error()}
}

func writeDecodeSetState(parseRaw json.RawMessage) (*writeSetStatePayload, *EnvelopeError) {
	var parseDec writeSetStatePayload
	if parseDecErr := json.Unmarshal(parseRaw, &parseDec); parseDecErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("set-state: malformed payload: %v", parseDecErr)}
	}
	if parseDec.Ref == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "set-state: missing required field \"ref\""}
	}
	if parseDec.Slot < 0 {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "set-state: field \"slot\" must be >= 0"}
	}
	if len(parseDec.Value) == 0 {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "set-state: missing required field \"value\""}
	}
	return &parseDec, nil
}

// ---------------------------------------------------------------------------
// bridge.emit
// ---------------------------------------------------------------------------

// writeEmitPayload is the decoded shape of a bridge.emit command.
type writeEmitPayload struct {
	Ref     string         `json:"ref"`
	Event   string         `json:"event"`
	Payload map[string]any `json:"payload"`
}

// writeHandleEmit handles the bridge.emit command. It resolves the node ref and
// invokes the named event handler via runtime.EmitAgentEvent. A stale ref maps
// to ErrorCodeStaleRef; an invalid ref or missing handler maps to
// ErrorCodeBadPayload.
func writeHandleEmit(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}

	parseDec, parseDecErr := writeDecodeEmit(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}

	parseRt := runtime.GetGlobalRuntime()
	parseEmitErr := runtime.EmitAgentEvent(parseRt, parseDec.Ref, parseDec.Event, parseDec.Payload)
	if parseEmitErr == nil {
		parseRt.AdvanceAgentStateVersion()
		// Emitting an event fires arbitrary handler side effects; it cannot be
		// reversed, but it must still appear in the audit trail.
		RecordAgentMutation("bridge.emit", fmt.Sprintf("emit %s on %s", parseDec.Event, parseDec.Ref), nil)
		return writeEncodeOK()
	}

	// Map runtime errors to wire error codes.
	if errors.Is(parseEmitErr, runtime.ErrAgentRefStale) {
		return nil, &EnvelopeError{Code: ErrorCodeStaleRef, Message: parseEmitErr.Error()}
	}
	if errors.Is(parseEmitErr, runtime.ErrAgentRefInvalid) {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseEmitErr.Error()}
	}
	if errors.Is(parseEmitErr, runtime.ErrAgentNoHandler) {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseEmitErr.Error()}
	}
	return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseEmitErr.Error()}
}

// writeDecodeEmit decodes and validates the emit payload.
func writeDecodeEmit(parseRaw json.RawMessage) (*writeEmitPayload, *EnvelopeError) {
	var parseDec writeEmitPayload
	if parseDecErr := json.Unmarshal(parseRaw, &parseDec); parseDecErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("emit: malformed payload: %v", parseDecErr),
		}
	}
	if parseDec.Ref == "" {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "emit: missing required field \"ref\"",
		}
	}
	if parseDec.Event == "" {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "emit: missing required field \"event\"",
		}
	}
	return &parseDec, nil
}

// ---------------------------------------------------------------------------
// bridge.publish
// ---------------------------------------------------------------------------

// writePublishPayload is the decoded shape of a bridge.publish command.
type writePublishPayload struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

// writeHandlePublish handles the bridge.publish command. It routes the raw JSON
// payload through events.PublishJSON, which delivers to the app's TYPED
// subscribers when the topic has a registered type codec (events.RegisterTopic
// [T]) and otherwise falls back to an any-typed publish. describe's
// publishableTopics lists which topics have a codec, so an agent knows whether
// a publish will reach typed subscribers before sending it.
func writeHandlePublish(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}

	parseDec, parseDecErr := writeDecodePublish(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}

	if parsePublishErr := events.PublishJSON(parseDec.Topic, parseDec.Payload); parsePublishErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("publish: topic %q: %v", parseDec.Topic, parsePublishErr),
		}
	}
	runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
	RecordAgentMutation("bridge.publish", "publish to "+parseDec.Topic, nil)
	return writeEncodeOK()
}

// writeDecodePublish decodes and validates the publish payload.
func writeDecodePublish(parseRaw json.RawMessage) (*writePublishPayload, *EnvelopeError) {
	var parseDec writePublishPayload
	if parseDecErr := json.Unmarshal(parseRaw, &parseDec); parseDecErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("publish: malformed payload: %v", parseDecErr),
		}
	}
	if parseDec.Topic == "" {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "publish: missing required field \"topic\"",
		}
	}
	if len(parseDec.Payload) == 0 {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "publish: missing required field \"payload\"",
		}
	}
	return &parseDec, nil
}

// ---------------------------------------------------------------------------
// bridge.navigate (platform-split — see commands_write_wasm.go / commands_write_native.go)
// ---------------------------------------------------------------------------

// writeNavigatePayload is the decoded shape of a bridge.navigate command.
type writeNavigatePayload struct {
	Path string `json:"path"`
}

// writeHandleNavigate handles the bridge.navigate command. The platform-
// specific implementation (wasm: calls router.Navigate; native: returns
// ErrorCodeBadPayload) lives in commands_write_wasm.go /
// commands_write_native.go respectively. The payload decoding and agent-mode
// gate are handled here so both compilation paths test the same logic.
func writeHandleNavigate(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}

	parseDec, parseDecErr := writeDecodeNavigate(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}

	parseResult, parseNavErr := writeNavigatePlatform(parseDec.Path)
	if parseNavErr == nil {
		runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
		// Navigation changes the active route; record it so the audit trail is
		// complete. It is not auto-reversible (the prior route is not captured
		// across the platform boundary), so an agent navigates back explicitly.
		RecordAgentMutation("bridge.navigate", "navigate to "+parseDec.Path, nil)
	}
	return parseResult, parseNavErr
}

// writeDecodeNavigate decodes and validates the navigate payload.
func writeDecodeNavigate(parseRaw json.RawMessage) (*writeNavigatePayload, *EnvelopeError) {
	var parseDec writeNavigatePayload
	if parseDecErr := json.Unmarshal(parseRaw, &parseDec); parseDecErr != nil {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: fmt.Sprintf("navigate: malformed payload: %v", parseDecErr),
		}
	}
	if parseDec.Path == "" {
		return nil, &EnvelopeError{
			Code:    ErrorCodeBadPayload,
			Message: "navigate: missing required field \"path\"",
		}
	}
	return &parseDec, nil
}

// ---------------------------------------------------------------------------
// bridge.mount / bridge.unmount
// ---------------------------------------------------------------------------

type writeMountPayload struct {
	ID        string          `json:"id"`
	Component string          `json:"component"`
	Selector  string          `json:"selector"`
	Props     json.RawMessage `json:"props,omitempty"`
}

type writeUnmountPayload struct {
	ID string `json:"id"`
}

func writeHandleMount(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}
	parseDec, parseDecErr := writeDecodeMount(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}
	parseProps := map[string]any{}
	if len(parseDec.Props) > 0 {
		if parseErr := json.Unmarshal(parseDec.Props, &parseProps); parseErr != nil {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: props must be a JSON object: %v", parseErr)}
		}
	}

	// Reject a non-existent selector up front. RenderTo's not-found signal is a
	// panic that the production policy suppresses, so without this a mount onto
	// a bad selector would silently "succeed" and render nowhere.
	if !runtime.GetGlobalRuntime().SelectorResolves(parseDec.Selector) {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: selector %q did not resolve to a container", parseDec.Selector)}
	}

	// Reserve the id atomically with the existence check so two concurrent
	// mounts of the same id cannot both pass the check and both call the
	// factory (double-render, orphaned first mount). The reservation is rolled
	// back below if rendering fails.
	writeMountMu.Lock()
	parseFactory := writeMountComponents[parseDec.Component]
	if parseFactory == nil {
		writeMountMu.Unlock()
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: component %q is not registered", parseDec.Component)}
	}
	if _, parseExists := writeMountedRoots[parseDec.ID]; parseExists {
		writeMountMu.Unlock()
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: id %q is already mounted", parseDec.ID)}
	}
	// Bound concurrent mounts: an agent calling bridge.mount with unique ids
	// and never unmounting would otherwise grow this map (and the live DOM
	// subtrees it tracks) without limit. Mirrors the auditCap ring elsewhere.
	if len(writeMountedRoots) >= writeMaxMountedRoots {
		writeMountMu.Unlock()
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: mounted-root limit (%d) reached; unmount before mounting more", writeMaxMountedRoots)}
	}
	writeMountedRoots[parseDec.ID] = writeMountedRoot{Component: parseDec.Component, Selector: parseDec.Selector, Props: parseProps}
	writeMountMu.Unlock()

	parseElement := parseFactory(parseProps)
	if parseElement == nil {
		writeMountReleaseReservation(parseDec.ID)
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: component %q rendered nil", parseDec.Component)}
	}
	if parseErr := writeRenderToSelector(parseDec.Selector, parseElement); parseErr != nil {
		writeMountReleaseReservation(parseDec.ID)
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "mount: " + parseErr.Error()}
	}

	// Record a reversible mutation whose undo unmounts the component.
	parseMountID := parseDec.ID
	parseMountSelector := parseDec.Selector
	RecordAgentMutation("bridge.mount", "mount "+parseDec.Component+" as "+parseMountID, func() *EnvelopeError {
		_ = writeRenderToSelector(parseMountSelector, nil)
		writeMountReleaseReservation(parseMountID)
		runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
		return nil
	})
	runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
	return writeEncodeOK()
}

func writeHandleUnmount(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}
	parseDec, parseDecErr := writeDecodeUnmount(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}

	// Claim the id by deleting it under the lock BEFORE rendering. This makes
	// check-and-claim atomic (a concurrent unmount of the same id sees it gone
	// and returns "not mounted" — no double nil-render) and guarantees a render
	// error cannot leave the id stuck as "mounted" forever.
	writeMountMu.Lock()
	parseMounted, parseExists := writeMountedRoots[parseDec.ID]
	if !parseExists {
		writeMountMu.Unlock()
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("unmount: id %q is not mounted", parseDec.ID)}
	}
	delete(writeMountedRoots, parseDec.ID)
	writeMountMu.Unlock()

	if parseErr := writeRenderToSelector(parseMounted.Selector, nil); parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "unmount: " + parseErr.Error()}
	}

	// Record the unmount with an undo that remounts the same component+props.
	parseUnmountID := parseDec.ID
	parseRemount := parseMounted
	RecordAgentMutation("bridge.unmount", "unmount "+parseUnmountID, func() *EnvelopeError {
		return writeRemount(parseUnmountID, parseRemount.Component, parseRemount.Selector, parseRemount.Props)
	})
	runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
	return writeEncodeOK()
}

func writeDecodeMount(parseRaw json.RawMessage) (*writeMountPayload, *EnvelopeError) {
	var parseDec writeMountPayload
	if parseErr := json.Unmarshal(parseRaw, &parseDec); parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("mount: malformed payload: %v", parseErr)}
	}
	parseDec.ID = strings.TrimSpace(parseDec.ID)
	parseDec.Component = strings.TrimSpace(parseDec.Component)
	parseDec.Selector = strings.TrimSpace(parseDec.Selector)
	if parseDec.ID == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "mount: missing required field \"id\""}
	}
	if parseDec.Component == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "mount: missing required field \"component\""}
	}
	if parseDec.Selector == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "mount: missing required field \"selector\""}
	}
	return &parseDec, nil
}

func writeDecodeUnmount(parseRaw json.RawMessage) (*writeUnmountPayload, *EnvelopeError) {
	var parseDec writeUnmountPayload
	if parseErr := json.Unmarshal(parseRaw, &parseDec); parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("unmount: malformed payload: %v", parseErr)}
	}
	parseDec.ID = strings.TrimSpace(parseDec.ID)
	if parseDec.ID == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "unmount: missing required field \"id\""}
	}
	return &parseDec, nil
}

func writeRenderToSelector(parseSelector string, parseElement *runtime.Element) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("%v", parseRecovered)
		}
	}()
	runtime.GetGlobalRuntime().RenderTo(parseSelector, parseElement)
	return nil
}

// ---------------------------------------------------------------------------
// bridge.delete-atom
// ---------------------------------------------------------------------------

type writeDeleteAtomPayload struct {
	ID    string `json:"id"`
	Force bool   `json:"force,omitempty"`
}

type writeDeleteAtomResult struct {
	OK          bool     `json:"ok"`
	Deleted     bool     `json:"deleted"`
	Subscribers []string `json:"subscribers,omitempty"`
}

func writeHandleDeleteAtom(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}
	parseDec, parseDecErr := writeDecodeDeleteAtom(parsePayload)
	if parseDecErr != nil {
		return nil, parseDecErr
	}
	// Capture the prior value before deleting so the delete is reversible.
	parsePriorVal, parseHadPrior := runtime.GetGlobalRuntime().SnapshotAtoms()[parseDec.ID]
	parseResult, parseErr := runtime.DeleteAgentAtom(runtime.GetGlobalRuntime(), parseDec.ID, parseDec.Force)
	if parseErr != nil {
		parseMessage := parseErr.Error()
		if len(parseResult.Subscribers) > 0 {
			parseMessage = fmt.Sprintf("%s; subscribers=%s", parseMessage, strings.Join(parseResult.Subscribers, ","))
		}
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseMessage}
	}
	if parseResult.Deleted && parseHadPrior {
		parseDeletedID := parseDec.ID
		parseRestoreVal := parsePriorVal
		RecordAgentMutation("bridge.delete-atom", "delete atom "+parseDeletedID, func() *EnvelopeError {
			if parseRestoreErr := state.ApplySnapshot(state.Snapshot{parseDeletedID: parseRestoreVal}); parseRestoreErr != nil {
				return &EnvelopeError{Code: ErrorCodeBadPayload, Message: "undo delete-atom " + parseDeletedID + ": " + parseRestoreErr.Error()}
			}
			runtime.GetGlobalRuntime().AdvanceAgentStateVersion()
			return nil
		})
	}
	parseBytes, parseMarshalErr := json.Marshal(writeDeleteAtomResult{OK: true, Deleted: parseResult.Deleted, Subscribers: parseResult.Subscribers})
	if parseMarshalErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseMarshalErr.Error()}
	}
	return parseBytes, nil
}

func writeDecodeDeleteAtom(parseRaw json.RawMessage) (*writeDeleteAtomPayload, *EnvelopeError) {
	var parseDec writeDeleteAtomPayload
	if parseErr := json.Unmarshal(parseRaw, &parseDec); parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("delete-atom: malformed payload: %v", parseErr)}
	}
	parseDec.ID = strings.TrimSpace(parseDec.ID)
	if parseDec.ID == "" {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "delete-atom: missing required field \"id\""}
	}
	return &parseDec, nil
}

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

// writeEncodeOK returns a minimal success payload JSON object.
func writeEncodeOK() (json.RawMessage, *EnvelopeError) {
	return json.RawMessage(`{"ok":true}`), nil
}

func writeValidateRawValueAgainstSchema(parseCommand string, parsePath string, parseRaw json.RawMessage, parseSchema map[string]any) *EnvelopeError {
	parseType, _ := parseSchema["type"].(string)
	if parseType == "" {
		return nil
	}
	var parseValue any
	if parseErr := json.Unmarshal(parseRaw, &parseValue); parseErr != nil {
		return &EnvelopeError{Code: ErrorCodeBadPayload, Message: fmt.Sprintf("%s: schema path %s: malformed JSON: %v", parseCommand, parsePath, parseErr)}
	}
	if writeJSONValueMatchesSchemaType(parseValue, parseType) {
		return nil
	}
	return &EnvelopeError{
		Code:    ErrorCodeBadPayload,
		Message: fmt.Sprintf("%s: schema path %s expected %s, got %T", parseCommand, parsePath, parseType, parseValue),
	}
}

func writeJSONValueMatchesSchemaType(parseValue any, parseType string) bool {
	switch parseType {
	case "null":
		return parseValue == nil
	case "boolean":
		_, parseOK := parseValue.(bool)
		return parseOK
	case "string":
		_, parseOK := parseValue.(string)
		return parseOK
	case "number":
		_, parseOK := parseValue.(float64)
		return parseOK
	case "array":
		_, parseOK := parseValue.([]any)
		return parseOK
	case "object":
		_, parseOK := parseValue.(map[string]any)
		return parseOK
	default:
		return true
	}
}
