package ui

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
)

const CurrentSSRStateUpdateVersion = 1

const (
	DefaultRouteBootstrapPayloadKey = "route:data"
	formBootstrapPayloadPrefix      = "form:"
	cacheBootstrapPayloadPrefix     = "cache:"
	sessionBootstrapPayloadPrefix   = "session:"
	defaultInlineWarnBytes          = 16 * 1024
	defaultInlineErrorBytes         = 32 * 1024
	defaultSidecarWarnBytes         = 32 * 1024
	defaultSidecarErrorBytes        = 64 * 1024
	defaultBinaryWarnBytes          = 24 * 1024
	defaultBinaryErrorBytes         = 48 * 1024
)

type SSRPayloadKind string

const (
	SSRPayloadKindData         SSRPayloadKind = "data"
	SSRPayloadKindRouteData    SSRPayloadKind = "route-data"
	SSRPayloadKindFormDefaults SSRPayloadKind = "form-defaults"
	SSRPayloadKindCacheSeed    SSRPayloadKind = "cache-seed"
	SSRPayloadKindSessionHint  SSRPayloadKind = "session-hint"
)

type SSRPayloadScope string

const (
	SSRPayloadScopeApp     SSRPayloadScope = "app"
	SSRPayloadScopeRoute   SSRPayloadScope = "route"
	SSRPayloadScopeSubtree SSRPayloadScope = "subtree"
)

type SSRPayloadReusePolicy string

const (
	SSRPayloadReuseTrustOnFirstResume    SSRPayloadReusePolicy = "trust-once"
	SSRPayloadReuseRevalidateAfterResume SSRPayloadReusePolicy = "revalidate-after-resume"
	SSRPayloadReuseClientOwned           SSRPayloadReusePolicy = "client-owned"
)

type SSRPayloadEncoding string

const (
	SSRPayloadEncodingJSON         SSRPayloadEncoding = "json"
	SSRPayloadEncodingText         SSRPayloadEncoding = "text"
	SSRPayloadEncodingBinary       SSRPayloadEncoding = "binary"
	SSRPayloadEncodingTimeRFC3339  SSRPayloadEncoding = "time-rfc3339"
	SSRPayloadEncodingTimeUnixNano SSRPayloadEncoding = "time-unix-nano"
	SSRPayloadEncodingCBOR         SSRPayloadEncoding = "cbor"
)

// SSRPayloadEnvelope stores one typed bootstrap payload entry inside SSRBootstrap.Data.
type SSRPayloadEnvelope struct {
	Version     int                   `json:"version,omitempty"`
	Kind        SSRPayloadKind        `json:"kind,omitempty"`
	Scope       SSRPayloadScope       `json:"scope,omitempty"`
	Target      string                `json:"target,omitempty"`
	ReusePolicy SSRPayloadReusePolicy `json:"reusePolicy,omitempty"`
	Revision    string                `json:"revision,omitempty"`
	Encoding    SSRPayloadEncoding    `json:"encoding,omitempty"`
	JSON        json.RawMessage       `json:"json,omitempty"`
	Text        string                `json:"text,omitempty"`
	Binary      []byte                `json:"binary,omitempty"`
}

// SSRPayloadOptions configures one typed bootstrap payload registration.
type SSRPayloadOptions struct {
	Kind        SSRPayloadKind
	Scope       SSRPayloadScope
	Target      string
	ReusePolicy SSRPayloadReusePolicy
	Revision    string
	Encoding    SSRPayloadEncoding
}

// SSRPayloadValue is the typed result of reading a bootstrap payload entry.
type SSRPayloadValue[T any] struct {
	Key         string
	Kind        SSRPayloadKind
	Scope       SSRPayloadScope
	Target      string
	ReusePolicy SSRPayloadReusePolicy
	Revision    string
	Encoding    SSRPayloadEncoding
	Value       T
}

// SSRPayloadFilter filters payload inspection output by kind, scope, or target.
type SSRPayloadFilter struct {
	Kind   SSRPayloadKind
	Scope  SSRPayloadScope
	Target string
}

// SSRPayloadMetadata summarizes one payload registration for diagnostics and scoping.
type SSRPayloadMetadata struct {
	Key         string
	Version     int
	Kind        SSRPayloadKind
	Scope       SSRPayloadScope
	Target      string
	ReusePolicy SSRPayloadReusePolicy
	Revision    string
	Encoding    SSRPayloadEncoding
	Legacy      bool
}

// SSRStateUpdate is a versioned text or binary update envelope for post-hydration payload refreshes.
type SSRStateUpdate struct {
	Version       int                           `json:"version,omitempty"`
	CorrelationID string                        `json:"correlationId,omitempty"`
	Scope         SSRPayloadScope               `json:"scope,omitempty"`
	Target        string                        `json:"target,omitempty"`
	Upserts       map[string]SSRPayloadEnvelope `json:"upserts,omitempty"`
	Deletes       []string                      `json:"deletes,omitempty"`
}

// SSRBootstrapBudget defines threshold bands for inline, sidecar, and binary payloads.
type SSRBootstrapBudget struct {
	InlineWarnBytes   int
	InlineErrorBytes  int
	SidecarWarnBytes  int
	SidecarErrorBytes int
	BinaryWarnBytes   int
	BinaryErrorBytes  int
}

// SSRBootstrapSizeReport summarizes payload sizes, warnings, and the recommended transport mode.
type SSRBootstrapSizeReport struct {
	Version            int
	JSONPayloadBytes   int
	InlineScriptBytes  int
	BinaryPayloadBytes int
	Recommendation     string
	Warnings           []string
	Errors             []string
	Payloads           []SSRPayloadMetadata
}

func escapeJSONForInlineScript(text string) string {
	replacer := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
		"\u2028", `\u2028`,
		"\u2029", `\u2029`,
	)
	return replacer.Replace(text)
}

func normalizeSSRPayloadOptions(options []SSRPayloadOptions) SSRPayloadOptions {
	if len(options) == 0 {
		return SSRPayloadOptions{}
	}
	return options[0]
}

func normalizeSSRPayloadEnvelope(envelope SSRPayloadEnvelope) (SSRPayloadEnvelope, error) {
	version, err := normalizeSSRBootstrapVersion(envelope.Version)
	if err != nil {
		return SSRPayloadEnvelope{}, err
	}
	envelope.Version = version
	if envelope.Kind == "" {
		envelope.Kind = SSRPayloadKindData
	}
	if envelope.Scope == "" {
		envelope.Scope = SSRPayloadScopeApp
	}
	if envelope.ReusePolicy == "" {
		envelope.ReusePolicy = SSRPayloadReuseTrustOnFirstResume
	}
	if envelope.Encoding == "" {
		switch {
		case len(envelope.JSON) > 0:
			envelope.Encoding = SSRPayloadEncodingJSON
		case envelope.Text != "":
			envelope.Encoding = SSRPayloadEncodingText
		case len(envelope.Binary) > 0:
			envelope.Encoding = SSRPayloadEncodingBinary
		default:
			envelope.Encoding = SSRPayloadEncodingJSON
		}
	}
	if err := validateSSRPayloadEnum(envelope.Kind, envelope.Scope, envelope.ReusePolicy, envelope.Encoding); err != nil {
		return SSRPayloadEnvelope{}, err
	}
	return envelope, nil
}

func validateSSRPayloadEnum(kind SSRPayloadKind, scope SSRPayloadScope, reuse SSRPayloadReusePolicy, encoding SSRPayloadEncoding) error {
	switch kind {
	case SSRPayloadKindData, SSRPayloadKindRouteData, SSRPayloadKindFormDefaults, SSRPayloadKindCacheSeed, SSRPayloadKindSessionHint:
	default:
		return fmt.Errorf("ui: unsupported payload kind %q", kind)
	}
	switch scope {
	case SSRPayloadScopeApp, SSRPayloadScopeRoute, SSRPayloadScopeSubtree:
	default:
		return fmt.Errorf("ui: unsupported payload scope %q", scope)
	}
	switch reuse {
	case SSRPayloadReuseTrustOnFirstResume, SSRPayloadReuseRevalidateAfterResume, SSRPayloadReuseClientOwned:
	default:
		return fmt.Errorf("ui: unsupported payload reuse policy %q", reuse)
	}
	switch encoding {
	case SSRPayloadEncodingJSON, SSRPayloadEncodingText, SSRPayloadEncodingBinary, SSRPayloadEncodingTimeRFC3339, SSRPayloadEncodingTimeUnixNano, SSRPayloadEncodingCBOR:
	default:
		return fmt.Errorf("ui: unsupported payload encoding %q", encoding)
	}
	return nil
}

func detectSSRPayloadEncoding(value interface{}, options SSRPayloadOptions) SSRPayloadEncoding {
	if options.Encoding != "" {
		return options.Encoding
	}
	switch typed := value.(type) {
	case []byte:
		_ = typed
		return SSRPayloadEncodingBinary
	case time.Time:
		return SSRPayloadEncodingTimeRFC3339
	case string:
		return SSRPayloadEncodingText
	case encoding.TextMarshaler:
		return SSRPayloadEncodingText
	default:
		return SSRPayloadEncodingJSON
	}
}

func encodeSSRPayloadEnvelope(value interface{}, options SSRPayloadOptions) (SSRPayloadEnvelope, error) {
	envelope := SSRPayloadEnvelope{
		Version:     CurrentSSRBootstrapVersion,
		Kind:        options.Kind,
		Scope:       options.Scope,
		Target:      strings.TrimSpace(options.Target),
		ReusePolicy: options.ReusePolicy,
		Revision:    strings.TrimSpace(options.Revision),
	}
	if envelope.Kind == "" {
		envelope.Kind = SSRPayloadKindData
	}
	if envelope.Scope == "" {
		envelope.Scope = SSRPayloadScopeApp
	}
	if envelope.ReusePolicy == "" {
		envelope.ReusePolicy = SSRPayloadReuseTrustOnFirstResume
	}
	envelope.Encoding = detectSSRPayloadEncoding(value, options)

	switch envelope.Encoding {
	case SSRPayloadEncodingJSON:
		encoded, err := json.Marshal(value)
		if err != nil {
			return SSRPayloadEnvelope{}, err
		}
		envelope.JSON = encoded
	case SSRPayloadEncodingText:
		text, err := encodeSSRTextPayload(value)
		if err != nil {
			return SSRPayloadEnvelope{}, err
		}
		envelope.Text = text
	case SSRPayloadEncodingBinary:
		bytes, err := encodeSSRBinaryPayload(value)
		if err != nil {
			return SSRPayloadEnvelope{}, err
		}
		envelope.Binary = bytes
	case SSRPayloadEncodingTimeRFC3339:
		stamp, err := encodeSSRTimePayload(value)
		if err != nil {
			return SSRPayloadEnvelope{}, err
		}
		envelope.Text = stamp.UTC().Format(time.RFC3339Nano)
	case SSRPayloadEncodingTimeUnixNano:
		stamp, err := encodeSSRTimePayload(value)
		if err != nil {
			return SSRPayloadEnvelope{}, err
		}
		envelope.Text = strconv.FormatInt(stamp.UTC().UnixNano(), 10)
	case SSRPayloadEncodingCBOR:
		encoded, err := cbor.Marshal(value)
		if err != nil {
			return SSRPayloadEnvelope{}, err
		}
		envelope.Binary = encoded
	}
	return normalizeSSRPayloadEnvelope(envelope)
}

func encodeSSRTextPayload(value interface{}) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case encoding.TextMarshaler:
		text, err := typed.MarshalText()
		if err != nil {
			return "", err
		}
		return string(text), nil
	default:
		return "", fmt.Errorf("ui: payload value of type %T does not support text encoding", value)
	}
}

func encodeSSRBinaryPayload(value interface{}) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...), nil
	default:
		return nil, fmt.Errorf("ui: payload value of type %T does not support binary encoding", value)
	}
}

func encodeSSRTimePayload(value interface{}) (time.Time, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	default:
		return time.Time{}, fmt.Errorf("ui: payload value of type %T does not support time encoding", value)
	}
}

func legacySSRPayloadEnvelope(raw interface{}) (SSRPayloadEnvelope, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return SSRPayloadEnvelope{}, err
	}
	return normalizeSSRPayloadEnvelope(SSRPayloadEnvelope{
		Version:     CurrentSSRBootstrapVersion,
		Kind:        SSRPayloadKindData,
		Scope:       SSRPayloadScopeApp,
		ReusePolicy: SSRPayloadReuseTrustOnFirstResume,
		Encoding:    SSRPayloadEncodingJSON,
		JSON:        encoded,
	})
}

func envelopeFromBootstrapData(raw interface{}) (SSRPayloadEnvelope, bool, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return SSRPayloadEnvelope{}, false, err
	}
	var envelope SSRPayloadEnvelope
	if err := json.Unmarshal(encoded, &envelope); err == nil {
		if envelope.Version != 0 || envelope.Kind != "" || envelope.Scope != "" || envelope.Target != "" || envelope.ReusePolicy != "" || envelope.Revision != "" || envelope.Encoding != "" || len(envelope.JSON) > 0 || envelope.Text != "" || len(envelope.Binary) > 0 {
			normalized, err := normalizeSSRPayloadEnvelope(envelope)
			return normalized, false, err
		}
	}
	legacy, err := legacySSRPayloadEnvelope(raw)
	return legacy, true, err
}

func decodeSSRPayloadEnvelope[T any](envelope SSRPayloadEnvelope) (T, error) {
	var value T
	envelope, err := normalizeSSRPayloadEnvelope(envelope)
	if err != nil {
		return value, err
	}

	switch envelope.Encoding {
	case SSRPayloadEncodingJSON:
		if len(envelope.JSON) == 0 {
			return value, nil
		}
		if err := json.Unmarshal(envelope.JSON, &value); err != nil {
			return value, err
		}
		return value, nil
	case SSRPayloadEncodingText:
		return assignDecodedSSRValue[T](envelope.Text)
	case SSRPayloadEncodingBinary:
		return assignDecodedSSRValue[T](append([]byte(nil), envelope.Binary...))
	case SSRPayloadEncodingTimeRFC3339:
		stamp, err := time.Parse(time.RFC3339Nano, envelope.Text)
		if err != nil {
			return value, err
		}
		return assignDecodedSSRValue[T](stamp)
	case SSRPayloadEncodingTimeUnixNano:
		ns, err := strconv.ParseInt(strings.TrimSpace(envelope.Text), 10, 64)
		if err != nil {
			return value, err
		}
		return assignDecodedSSRValue[T](time.Unix(0, ns).UTC())
	case SSRPayloadEncodingCBOR:
		if err := cbor.Unmarshal(envelope.Binary, &value); err != nil {
			return value, err
		}
		return value, nil
	default:
		return value, fmt.Errorf("ui: unsupported payload encoding %q", envelope.Encoding)
	}
}

func assignDecodedSSRValue[T any](decoded interface{}) (T, error) {
	var value T
	if unmarshaler, ok := any(&value).(encoding.TextUnmarshaler); ok {
		if text, ok := decoded.(string); ok {
			if err := unmarshaler.UnmarshalText([]byte(text)); err != nil {
				return value, err
			}
			return value, nil
		}
	}

	target := reflect.ValueOf(&value).Elem()
	if !target.CanSet() {
		return value, fmt.Errorf("ui: could not set decoded payload value")
	}
	source := reflect.ValueOf(decoded)
	if !source.IsValid() {
		return value, nil
	}
	if source.Type().AssignableTo(target.Type()) {
		target.Set(source)
		return value, nil
	}
	if source.Type().ConvertibleTo(target.Type()) {
		target.Set(source.Convert(target.Type()))
		return value, nil
	}
	return value, fmt.Errorf("ui: decoded value of type %s cannot populate %s", source.Type(), target.Type())
}

// RegisterBootstrapPayload stores one typed payload entry under SSRBootstrap.Data.
func RegisterBootstrapPayload[T any](bootstrap *SSRBootstrap, key string, value T, options ...SSRPayloadOptions) error {
	if bootstrap == nil {
		return fmt.Errorf("ui: bootstrap cannot be nil")
	}
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return fmt.Errorf("ui: payload key cannot be empty")
	}
	envelope, err := encodeSSRPayloadEnvelope(value, normalizeSSRPayloadOptions(options))
	if err != nil {
		return err
	}
	if bootstrap.Data == nil {
		bootstrap.Data = map[string]interface{}{}
	}
	bootstrap.Data[trimmedKey] = envelope
	return nil
}

// ReadBootstrapPayload reads one typed payload entry from SSRBootstrap.Data.
func ReadBootstrapPayload[T any](bootstrap SSRBootstrap, key string) (SSRPayloadValue[T], bool, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return SSRPayloadValue[T]{}, false, fmt.Errorf("ui: payload key cannot be empty")
	}
	raw, ok := bootstrap.Data[trimmedKey]
	if !ok {
		return SSRPayloadValue[T]{}, false, nil
	}
	envelope, _, err := envelopeFromBootstrapData(raw)
	if err != nil {
		return SSRPayloadValue[T]{}, false, err
	}
	value, err := decodeSSRPayloadEnvelope[T](envelope)
	if err != nil {
		return SSRPayloadValue[T]{}, false, err
	}
	return SSRPayloadValue[T]{
		Key:         trimmedKey,
		Kind:        envelope.Kind,
		Scope:       envelope.Scope,
		Target:      envelope.Target,
		ReusePolicy: envelope.ReusePolicy,
		Revision:    envelope.Revision,
		Encoding:    envelope.Encoding,
		Value:       value,
	}, true, nil
}

// RegisterRouteBootstrapData stores typed route data under a route-scoped payload key.
func RegisterRouteBootstrapData[T any](bootstrap *SSRBootstrap, key string, routePath string, value T, options ...SSRPayloadOptions) error {
	resolved := normalizeSSRPayloadOptions(options)
	resolved.Kind = SSRPayloadKindRouteData
	resolved.Scope = SSRPayloadScopeRoute
	resolved.Target = strings.TrimSpace(routePath)
	return RegisterBootstrapPayload(bootstrap, routeBootstrapPayloadKey(key, routePath), value, resolved)
}

// ReadRouteBootstrapData reads typed route data from a route-scoped payload key.
func ReadRouteBootstrapData[T any](bootstrap SSRBootstrap, key string, routePath string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](bootstrap, routeBootstrapPayloadKey(key, routePath))
}

// RegisterFormBootstrapDefaults stores typed form defaults under a form-scoped payload key.
func RegisterFormBootstrapDefaults[T any](bootstrap *SSRBootstrap, formID string, value T, options ...SSRPayloadOptions) error {
	trimmedFormID := strings.TrimSpace(formID)
	resolved := normalizeSSRPayloadOptions(options)
	resolved.Kind = SSRPayloadKindFormDefaults
	resolved.Scope = SSRPayloadScopeSubtree
	resolved.Target = trimmedFormID
	return RegisterBootstrapPayload(bootstrap, formBootstrapPayloadPrefix+trimmedFormID, value, resolved)
}

// ReadFormBootstrapDefaults reads typed form defaults from a form-scoped payload key.
func ReadFormBootstrapDefaults[T any](bootstrap SSRBootstrap, formID string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](bootstrap, formBootstrapPayloadPrefix+strings.TrimSpace(formID))
}

// RegisterCacheBootstrapSeed stores a typed cache seed under a cache-scoped payload key.
func RegisterCacheBootstrapSeed[T any](bootstrap *SSRBootstrap, cacheKey string, value T, options ...SSRPayloadOptions) error {
	trimmedCacheKey := strings.TrimSpace(cacheKey)
	resolved := normalizeSSRPayloadOptions(options)
	resolved.Kind = SSRPayloadKindCacheSeed
	resolved.Scope = SSRPayloadScopeRoute
	resolved.Target = trimmedCacheKey
	if resolved.ReusePolicy == "" {
		resolved.ReusePolicy = SSRPayloadReuseRevalidateAfterResume
	}
	return RegisterBootstrapPayload(bootstrap, cacheBootstrapPayloadPrefix+trimmedCacheKey, value, resolved)
}

// ReadCacheBootstrapSeed reads a typed cache seed from a cache-scoped payload key.
func ReadCacheBootstrapSeed[T any](bootstrap SSRBootstrap, cacheKey string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](bootstrap, cacheBootstrapPayloadPrefix+strings.TrimSpace(cacheKey))
}

// RegisterSessionBootstrapHint stores a typed session hint under an app-scoped payload key.
func RegisterSessionBootstrapHint[T any](bootstrap *SSRBootstrap, hintKey string, value T, options ...SSRPayloadOptions) error {
	trimmedHintKey := strings.TrimSpace(hintKey)
	resolved := normalizeSSRPayloadOptions(options)
	resolved.Kind = SSRPayloadKindSessionHint
	resolved.Scope = SSRPayloadScopeApp
	resolved.Target = trimmedHintKey
	if resolved.ReusePolicy == "" {
		resolved.ReusePolicy = SSRPayloadReuseClientOwned
	}
	return RegisterBootstrapPayload(bootstrap, sessionBootstrapPayloadPrefix+trimmedHintKey, value, resolved)
}

// ReadSessionBootstrapHint reads a typed session hint from an app-scoped payload key.
func ReadSessionBootstrapHint[T any](bootstrap SSRBootstrap, hintKey string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](bootstrap, sessionBootstrapPayloadPrefix+strings.TrimSpace(hintKey))
}

func routeBootstrapPayloadKey(key string, routePath string) string {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		trimmedKey = DefaultRouteBootstrapPayloadKey
	}
	return strings.TrimSpace(routePath) + "::" + trimmedKey
}

// InspectBootstrapPayloads lists typed payload registrations and legacy payloads, optionally filtered by kind, scope, or target.
func InspectBootstrapPayloads(bootstrap SSRBootstrap, filter SSRPayloadFilter) ([]SSRPayloadMetadata, error) {
	if len(bootstrap.Data) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(bootstrap.Data))
	for key := range bootstrap.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]SSRPayloadMetadata, 0, len(keys))
	for _, key := range keys {
		envelope, legacy, err := envelopeFromBootstrapData(bootstrap.Data[key])
		if err != nil {
			return nil, err
		}
		if filter.Kind != "" && envelope.Kind != filter.Kind {
			continue
		}
		if filter.Scope != "" && envelope.Scope != filter.Scope {
			continue
		}
		if strings.TrimSpace(filter.Target) != "" && strings.TrimSpace(envelope.Target) != strings.TrimSpace(filter.Target) {
			continue
		}
		items = append(items, SSRPayloadMetadata{
			Key:         key,
			Version:     envelope.Version,
			Kind:        envelope.Kind,
			Scope:       envelope.Scope,
			Target:      envelope.Target,
			ReusePolicy: envelope.ReusePolicy,
			Revision:    envelope.Revision,
			Encoding:    envelope.Encoding,
			Legacy:      legacy,
		})
	}
	return items, nil
}

func normalizeSSRStateUpdate(update SSRStateUpdate) (SSRStateUpdate, error) {
	version, err := normalizeSSRStateUpdateVersion(update.Version)
	if err != nil {
		return SSRStateUpdate{}, err
	}
	update.Version = version
	if update.Scope == "" {
		update.Scope = SSRPayloadScopeApp
	}
	if update.Upserts == nil {
		update.Upserts = map[string]SSRPayloadEnvelope{}
	}
	for key, envelope := range update.Upserts {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return SSRStateUpdate{}, fmt.Errorf("ui: state update upsert key cannot be empty")
		}
		if envelope.Scope == "" {
			envelope.Scope = update.Scope
		}
		if envelope.Target == "" && update.Target != "" {
			envelope.Target = update.Target
		}
		if envelope.Version == 0 {
			envelope.Version = CurrentSSRBootstrapVersion
		}
		normalized, err := normalizeSSRPayloadEnvelope(envelope)
		if err != nil {
			return SSRStateUpdate{}, err
		}
		if trimmedKey != key {
			delete(update.Upserts, key)
		}
		update.Upserts[trimmedKey] = normalized
	}
	if len(update.Deletes) > 0 {
		trimmedDeletes := make([]string, 0, len(update.Deletes))
		seen := map[string]bool{}
		for _, key := range update.Deletes {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" || seen[trimmedKey] {
				continue
			}
			trimmedDeletes = append(trimmedDeletes, trimmedKey)
			seen[trimmedKey] = true
		}
		update.Deletes = trimmedDeletes
	}
	return update, nil
}

func normalizeSSRStateUpdateVersion(version int) (int, error) {
	if version < 0 {
		return 0, fmt.Errorf("ui: unsupported SSR state update version %d", version)
	}
	if version == 0 {
		return CurrentSSRStateUpdateVersion, nil
	}
	if version > CurrentSSRStateUpdateVersion {
		return 0, fmt.Errorf("ui: unsupported SSR state update version %d", version)
	}
	return version, nil
}

// RegisterStateUpdatePayload stores one typed payload upsert in a state-update envelope.
func RegisterStateUpdatePayload[T any](update *SSRStateUpdate, key string, value T, options ...SSRPayloadOptions) error {
	if update == nil {
		return fmt.Errorf("ui: state update cannot be nil")
	}
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return fmt.Errorf("ui: state update payload key cannot be empty")
	}
	resolved := normalizeSSRPayloadOptions(options)
	if resolved.Scope == "" && update.Scope != "" {
		resolved.Scope = update.Scope
	}
	if resolved.Target == "" && update.Target != "" {
		resolved.Target = update.Target
	}
	envelope, err := encodeSSRPayloadEnvelope(value, resolved)
	if err != nil {
		return err
	}
	if update.Upserts == nil {
		update.Upserts = map[string]SSRPayloadEnvelope{}
	}
	update.Upserts[trimmedKey] = envelope
	return nil
}

// MarshalSSRStateUpdateText encodes a state-update envelope as JSON text.
func MarshalSSRStateUpdateText(update SSRStateUpdate) ([]byte, error) {
	update, err := normalizeSSRStateUpdate(update)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(update)
	if err != nil {
		return nil, err
	}
	return []byte(escapeJSONForInlineScript(string(encoded))), nil
}

// UnmarshalSSRStateUpdateText decodes a JSON text state-update envelope.
func UnmarshalSSRStateUpdateText(data []byte) (SSRStateUpdate, error) {
	if len(data) == 0 {
		return normalizeSSRStateUpdate(SSRStateUpdate{})
	}
	var update SSRStateUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return SSRStateUpdate{}, err
	}
	return normalizeSSRStateUpdate(update)
}

// MarshalSSRStateUpdateBinary encodes a state-update envelope as CBOR.
func MarshalSSRStateUpdateBinary(update SSRStateUpdate) ([]byte, error) {
	update, err := normalizeSSRStateUpdate(update)
	if err != nil {
		return nil, err
	}
	return cbor.Marshal(update)
}

// UnmarshalSSRStateUpdateBinary decodes a CBOR state-update envelope.
func UnmarshalSSRStateUpdateBinary(data []byte) (SSRStateUpdate, error) {
	if len(data) == 0 {
		return normalizeSSRStateUpdate(SSRStateUpdate{})
	}
	var update SSRStateUpdate
	if err := cbor.Unmarshal(data, &update); err != nil {
		return SSRStateUpdate{}, err
	}
	return normalizeSSRStateUpdate(update)
}

// ApplySSRStateUpdate merges a text or binary update envelope into a bootstrap payload snapshot.
func ApplySSRStateUpdate(bootstrap *SSRBootstrap, update SSRStateUpdate) error {
	if bootstrap == nil {
		return fmt.Errorf("ui: bootstrap cannot be nil")
	}
	update, err := normalizeSSRStateUpdate(update)
	if err != nil {
		return err
	}
	if bootstrap.Data == nil {
		bootstrap.Data = map[string]interface{}{}
	}
	for _, key := range update.Deletes {
		delete(bootstrap.Data, key)
	}
	for key, envelope := range update.Upserts {
		bootstrap.Data[key] = envelope
	}
	return nil
}

// NewSSRBootstrapBudget returns the default bootstrap size thresholds.
func NewSSRBootstrapBudget() SSRBootstrapBudget {
	return SSRBootstrapBudget{
		InlineWarnBytes:   defaultInlineWarnBytes,
		InlineErrorBytes:  defaultInlineErrorBytes,
		SidecarWarnBytes:  defaultSidecarWarnBytes,
		SidecarErrorBytes: defaultSidecarErrorBytes,
		BinaryWarnBytes:   defaultBinaryWarnBytes,
		BinaryErrorBytes:  defaultBinaryErrorBytes,
	}
}

func normalizeSSRBootstrapBudget(budget SSRBootstrapBudget) SSRBootstrapBudget {
	defaults := NewSSRBootstrapBudget()
	if budget.InlineWarnBytes <= 0 {
		budget.InlineWarnBytes = defaults.InlineWarnBytes
	}
	if budget.InlineErrorBytes <= 0 || budget.InlineErrorBytes < budget.InlineWarnBytes {
		budget.InlineErrorBytes = defaults.InlineErrorBytes
	}
	if budget.SidecarWarnBytes <= 0 {
		budget.SidecarWarnBytes = defaults.SidecarWarnBytes
	}
	if budget.SidecarErrorBytes <= 0 || budget.SidecarErrorBytes < budget.SidecarWarnBytes {
		budget.SidecarErrorBytes = defaults.SidecarErrorBytes
	}
	if budget.BinaryWarnBytes <= 0 {
		budget.BinaryWarnBytes = defaults.BinaryWarnBytes
	}
	if budget.BinaryErrorBytes <= 0 || budget.BinaryErrorBytes < budget.BinaryWarnBytes {
		budget.BinaryErrorBytes = defaults.BinaryErrorBytes
	}
	return budget
}

// InspectSSRBootstrapSize measures payload sizes, budget bands, and the recommended transport mode.
func InspectSSRBootstrapSize(payload SSRBootstrap, budget SSRBootstrapBudget) (SSRBootstrapSizeReport, error) {
	budget = normalizeSSRBootstrapBudget(budget)
	normalized, err := normalizeSSRBootstrap(payload)
	if err != nil {
		return SSRBootstrapSizeReport{}, err
	}
	jsonPayload, err := marshalSSRBootstrapJSON(normalized)
	if err != nil {
		return SSRBootstrapSizeReport{}, err
	}
	binaryPayload, err := marshalSSRBootstrapBinary(normalized)
	if err != nil {
		return SSRBootstrapSizeReport{}, err
	}
	inlineScript := `<script id="` + DefaultBootstrapScriptID + `" type="application/json">` + string(jsonPayload) + `</script>`
	payloads, err := InspectBootstrapPayloads(normalized, SSRPayloadFilter{})
	if err != nil {
		return SSRBootstrapSizeReport{}, err
	}
	report := SSRBootstrapSizeReport{
		Version:            normalized.Version,
		JSONPayloadBytes:   len(jsonPayload),
		InlineScriptBytes:  len(inlineScript),
		BinaryPayloadBytes: len(binaryPayload),
		Recommendation:     "inline-json",
		Payloads:           payloads,
	}
	if report.InlineScriptBytes >= budget.InlineWarnBytes {
		report.Warnings = append(report.Warnings, fmt.Sprintf("inline bootstrap script is %d bytes", report.InlineScriptBytes))
	}
	if report.InlineScriptBytes >= budget.InlineErrorBytes {
		report.Errors = append(report.Errors, fmt.Sprintf("inline bootstrap script exceeds the %d byte inline budget", budget.InlineErrorBytes))
	}
	if report.JSONPayloadBytes >= budget.SidecarWarnBytes {
		report.Warnings = append(report.Warnings, fmt.Sprintf("JSON bootstrap payload is %d bytes", report.JSONPayloadBytes))
	}
	if report.JSONPayloadBytes >= budget.SidecarErrorBytes {
		report.Errors = append(report.Errors, fmt.Sprintf("JSON bootstrap payload exceeds the %d byte sidecar budget", budget.SidecarErrorBytes))
	}
	if report.BinaryPayloadBytes >= budget.BinaryWarnBytes {
		report.Warnings = append(report.Warnings, fmt.Sprintf("binary bootstrap payload is %d bytes", report.BinaryPayloadBytes))
	}
	if report.BinaryPayloadBytes >= budget.BinaryErrorBytes {
		report.Errors = append(report.Errors, fmt.Sprintf("binary bootstrap payload exceeds the %d byte binary budget", budget.BinaryErrorBytes))
	}
	if report.InlineScriptBytes >= budget.InlineWarnBytes {
		report.Recommendation = "sidecar-json"
	}
	if report.BinaryPayloadBytes < report.JSONPayloadBytes && (report.JSONPayloadBytes >= budget.SidecarWarnBytes || report.InlineScriptBytes >= budget.InlineWarnBytes) {
		report.Recommendation = "sidecar-cbor"
	}
	return report, nil
}
