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

// escapeJSONForInlineScript is a core package helper.
func escapeJSONForInlineScript(parseText string) string {
	parseReplacer := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
		"\u2028", `\u2028`,
		"\u2029", `\u2029`,
	)
	return parseReplacer.Replace(parseText)
}

// normalizeSSRPayloadOptions is a core package helper.
func normalizeSSRPayloadOptions(parseOptions []SSRPayloadOptions) SSRPayloadOptions {
	if len(parseOptions) == 0 {
		return SSRPayloadOptions{}
	}
	return parseOptions[0]
}

// normalizeSSRPayloadEnvelope is a core package helper.
func normalizeSSRPayloadEnvelope(parseEnvelope SSRPayloadEnvelope) (SSRPayloadEnvelope, error) {
	parseVersion, parseErr := normalizeSSRBootstrapVersion(parseEnvelope.Version)
	if parseErr != nil {
		return SSRPayloadEnvelope{}, parseErr
	}
	parseEnvelope.Version = parseVersion
	if parseEnvelope.Kind == "" {
		parseEnvelope.Kind = SSRPayloadKindData
	}
	if parseEnvelope.Scope == "" {
		parseEnvelope.Scope = SSRPayloadScopeApp
	}
	if parseEnvelope.ReusePolicy == "" {
		parseEnvelope.ReusePolicy = SSRPayloadReuseTrustOnFirstResume
	}
	if parseEnvelope.Encoding == "" {
		switch {
		case len(parseEnvelope.JSON) > 0:
			parseEnvelope.Encoding = SSRPayloadEncodingJSON
		case parseEnvelope.Text != "":
			parseEnvelope.Encoding = SSRPayloadEncodingText
		case len(parseEnvelope.Binary) > 0:
			parseEnvelope.Encoding = SSRPayloadEncodingBinary
		default:
			parseEnvelope.Encoding = SSRPayloadEncodingJSON
		}
	}
	if parseErr2 := validateSSRPayloadEnum(parseEnvelope.Kind, parseEnvelope.Scope, parseEnvelope.ReusePolicy, parseEnvelope.Encoding); parseErr2 != nil {
		return SSRPayloadEnvelope{}, parseErr2
	}
	return parseEnvelope, nil
}

// validateSSRPayloadEnum is a core package helper.
func validateSSRPayloadEnum(parseKind SSRPayloadKind, parseScope SSRPayloadScope, parseReuse SSRPayloadReusePolicy, parseEncoding SSRPayloadEncoding) error {
	switch parseKind {
	case SSRPayloadKindData, SSRPayloadKindRouteData, SSRPayloadKindFormDefaults, SSRPayloadKindCacheSeed, SSRPayloadKindSessionHint:
	default:
		return fmt.Errorf("ui: unsupported payload kind %q", parseKind)
	}
	switch parseScope {
	case SSRPayloadScopeApp, SSRPayloadScopeRoute, SSRPayloadScopeSubtree:
	default:
		return fmt.Errorf("ui: unsupported payload scope %q", parseScope)
	}
	switch parseReuse {
	case SSRPayloadReuseTrustOnFirstResume, SSRPayloadReuseRevalidateAfterResume, SSRPayloadReuseClientOwned:
	default:
		return fmt.Errorf("ui: unsupported payload reuse policy %q", parseReuse)
	}
	switch parseEncoding {
	case SSRPayloadEncodingJSON, SSRPayloadEncodingText, SSRPayloadEncodingBinary, SSRPayloadEncodingTimeRFC3339, SSRPayloadEncodingTimeUnixNano, SSRPayloadEncodingCBOR:
	default:
		return fmt.Errorf("ui: unsupported payload encoding %q", parseEncoding)
	}
	return nil
}

// detectSSRPayloadEncoding is a core package helper.
func detectSSRPayloadEncoding(parseValue any, parseOptions SSRPayloadOptions) SSRPayloadEncoding {
	if parseOptions.Encoding != "" {
		return parseOptions.Encoding
	}
	switch parseTyped := parseValue.(type) {
	case []byte:
		_ = parseTyped
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

// encodeSSRPayloadEnvelope is a core package helper.
func encodeSSRPayloadEnvelope(parseValue any, parseOptions SSRPayloadOptions) (SSRPayloadEnvelope, error) {
	parseEnvelope := SSRPayloadEnvelope{
		Version:     CurrentSSRBootstrapVersion,
		Kind:        parseOptions.Kind,
		Scope:       parseOptions.Scope,
		Target:      strings.TrimSpace(parseOptions.Target),
		ReusePolicy: parseOptions.ReusePolicy,
		Revision:    strings.TrimSpace(parseOptions.Revision),
	}
	if parseEnvelope.Kind == "" {
		parseEnvelope.Kind = SSRPayloadKindData
	}
	if parseEnvelope.Scope == "" {
		parseEnvelope.Scope = SSRPayloadScopeApp
	}
	if parseEnvelope.ReusePolicy == "" {
		parseEnvelope.ReusePolicy = SSRPayloadReuseTrustOnFirstResume
	}
	parseEnvelope.Encoding = detectSSRPayloadEncoding(parseValue, parseOptions)

	switch parseEnvelope.Encoding {
	case SSRPayloadEncodingJSON:
		parseEncoded, parseErr := json.Marshal(parseValue)
		if parseErr != nil {
			return SSRPayloadEnvelope{}, parseErr
		}
		parseEnvelope.JSON = parseEncoded
	case SSRPayloadEncodingText:
		parseText, parseErr2 := encodeSSRTextPayload(parseValue)
		if parseErr2 != nil {
			return SSRPayloadEnvelope{}, parseErr2
		}
		parseEnvelope.Text = parseText
	case SSRPayloadEncodingBinary:
		parseBytes, parseErr3 := encodeSSRBinaryPayload(parseValue)
		if parseErr3 != nil {
			return SSRPayloadEnvelope{}, parseErr3
		}
		parseEnvelope.Binary = parseBytes
	case SSRPayloadEncodingTimeRFC3339:
		parseStamp, parseErr4 := encodeSSRTimePayload(parseValue)
		if parseErr4 != nil {
			return SSRPayloadEnvelope{}, parseErr4
		}
		parseEnvelope.Text = parseStamp.UTC().Format(time.RFC3339Nano)
	case SSRPayloadEncodingTimeUnixNano:
		parseStamp2, parseErr5 := encodeSSRTimePayload(parseValue)
		if parseErr5 != nil {
			return SSRPayloadEnvelope{}, parseErr5
		}
		parseEnvelope.Text = strconv.FormatInt(parseStamp2.UTC().UnixNano(), 10)
	case SSRPayloadEncodingCBOR:
		parseEncoded2, parseErr6 := cbor.Marshal(parseValue)
		if parseErr6 != nil {
			return SSRPayloadEnvelope{}, parseErr6
		}
		parseEnvelope.Binary = parseEncoded2
	}
	return normalizeSSRPayloadEnvelope(parseEnvelope)
}

// encodeSSRTextPayload is a core package helper.
func encodeSSRTextPayload(parseValue any) (string, error) {
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped, nil
	case encoding.TextMarshaler:
		parseText, parseErr := parseTyped.MarshalText()
		if parseErr != nil {
			return "", parseErr
		}
		return string(parseText), nil
	default:
		return "", fmt.Errorf("ui: payload value of type %T does not support text encoding", parseValue)
	}
}

// encodeSSRBinaryPayload is a core package helper.
func encodeSSRBinaryPayload(parseValue any) ([]byte, error) {
	switch parseTyped := parseValue.(type) {
	case []byte:
		return append([]byte(nil), parseTyped...), nil
	default:
		return nil, fmt.Errorf("ui: payload value of type %T does not support binary encoding", parseValue)
	}
}

// encodeSSRTimePayload is a core package helper.
func encodeSSRTimePayload(parseValue any) (time.Time, error) {
	switch parseTyped := parseValue.(type) {
	case time.Time:
		return parseTyped, nil
	default:
		return time.Time{}, fmt.Errorf("ui: payload value of type %T does not support time encoding", parseValue)
	}
}

// legacySSRPayloadEnvelope is a core package helper.
func legacySSRPayloadEnvelope(parseRaw any) (SSRPayloadEnvelope, error) {
	parseEncoded, parseErr := json.Marshal(parseRaw)
	if parseErr != nil {
		return SSRPayloadEnvelope{}, parseErr
	}
	return normalizeSSRPayloadEnvelope(SSRPayloadEnvelope{
		Version:     CurrentSSRBootstrapVersion,
		Kind:        SSRPayloadKindData,
		Scope:       SSRPayloadScopeApp,
		ReusePolicy: SSRPayloadReuseTrustOnFirstResume,
		Encoding:    SSRPayloadEncodingJSON,
		JSON:        parseEncoded,
	})
}

// envelopeFromBootstrapData is a core package helper.
func envelopeFromBootstrapData(parseRaw any) (SSRPayloadEnvelope, bool, error) {
	parseEncoded, parseErr := json.Marshal(parseRaw)
	if parseErr != nil {
		return SSRPayloadEnvelope{}, false, parseErr
	}
	var parseEnvelope SSRPayloadEnvelope
	if parseErr2 := json.Unmarshal(parseEncoded, &parseEnvelope); parseErr2 == nil {
		if parseEnvelope.Version != 0 || parseEnvelope.Kind != "" || parseEnvelope.Scope != "" || parseEnvelope.Target != "" || parseEnvelope.ReusePolicy != "" || parseEnvelope.Revision != "" || parseEnvelope.Encoding != "" || len(parseEnvelope.JSON) > 0 || parseEnvelope.Text != "" || len(parseEnvelope.Binary) > 0 {
			parseNormalized, parseErr3 := normalizeSSRPayloadEnvelope(parseEnvelope)
			return parseNormalized, false, parseErr3
		}
	}
	parseLegacy, parseErr := legacySSRPayloadEnvelope(parseRaw)
	return parseLegacy, true, parseErr
}

// decodeSSRPayloadEnvelope is a core package helper.
func decodeSSRPayloadEnvelope[T any](parseEnvelope SSRPayloadEnvelope) (T, error) {
	var parseValue T
	parseEnvelope, parseErr := normalizeSSRPayloadEnvelope(parseEnvelope)
	if parseErr != nil {
		return parseValue, parseErr
	}

	switch parseEnvelope.Encoding {
	case SSRPayloadEncodingJSON:
		if len(parseEnvelope.JSON) == 0 {
			return parseValue, nil
		}
		if parseErr2 := json.Unmarshal(parseEnvelope.JSON, &parseValue); parseErr2 != nil {
			return parseValue, parseErr2
		}
		return parseValue, nil
	case SSRPayloadEncodingText:
		return assignDecodedSSRValue[T](parseEnvelope.Text)
	case SSRPayloadEncodingBinary:
		return assignDecodedSSRValue[T](append([]byte(nil), parseEnvelope.Binary...))
	case SSRPayloadEncodingTimeRFC3339:
		parseStamp, parseErr3 := time.Parse(time.RFC3339Nano, parseEnvelope.Text)
		if parseErr3 != nil {
			return parseValue, parseErr3
		}
		return assignDecodedSSRValue[T](parseStamp)
	case SSRPayloadEncodingTimeUnixNano:
		parseNs, parseErr4 := strconv.ParseInt(strings.TrimSpace(parseEnvelope.Text), 10, 64)
		if parseErr4 != nil {
			return parseValue, parseErr4
		}
		return assignDecodedSSRValue[T](time.Unix(0, parseNs).UTC())
	case SSRPayloadEncodingCBOR:
		if parseErr5 := cbor.Unmarshal(parseEnvelope.Binary, &parseValue); parseErr5 != nil {
			return parseValue, parseErr5
		}
		return parseValue, nil
	default:
		return parseValue, fmt.Errorf("ui: unsupported payload encoding %q", parseEnvelope.Encoding)
	}
}

// assignDecodedSSRValue is a core package helper.
func assignDecodedSSRValue[T any](parseDecoded any) (T, error) {
	var parseValue T
	if parseUnmarshaler, parseOk := any(&parseValue).(encoding.TextUnmarshaler); parseOk {
		if parseText, parseOk2 := parseDecoded.(string); parseOk2 {
			if parseErr := parseUnmarshaler.UnmarshalText([]byte(parseText)); parseErr != nil {
				return parseValue, parseErr
			}
			return parseValue, nil
		}
	}

	parseTarget := reflect.ValueOf(&parseValue).Elem()
	if !parseTarget.CanSet() {
		return parseValue, fmt.Errorf("ui: could not set decoded payload value")
	}
	parseSource := reflect.ValueOf(parseDecoded)
	if !parseSource.IsValid() {
		return parseValue, nil
	}
	if parseSource.Type().AssignableTo(parseTarget.Type()) {
		parseTarget.Set(parseSource)
		return parseValue, nil
	}
	if parseSource.Type().ConvertibleTo(parseTarget.Type()) {
		parseTarget.Set(parseSource.Convert(parseTarget.Type()))
		return parseValue, nil
	}
	return parseValue, fmt.Errorf("ui: decoded value of type %s cannot populate %s", parseSource.Type(), parseTarget.Type())
}

// RegisterBootstrapPayload stores one typed payload entry under SSRBootstrap.Data.
func RegisterBootstrapPayload[T any](parseBootstrap *SSRBootstrap, parseKey string, parseValue T, parseOptions ...SSRPayloadOptions) error {
	if parseBootstrap == nil {
		return fmt.Errorf("ui: bootstrap cannot be nil")
	}
	parseTrimmedKey := strings.TrimSpace(parseKey)
	if parseTrimmedKey == "" {
		return fmt.Errorf("ui: payload key cannot be empty")
	}
	parseEnvelope, parseErr := encodeSSRPayloadEnvelope(parseValue, normalizeSSRPayloadOptions(parseOptions))
	if parseErr != nil {
		return parseErr
	}
	if parseBootstrap.Data == nil {
		parseBootstrap.Data = map[string]any{}
	}
	parseBootstrap.Data[parseTrimmedKey] = parseEnvelope
	return nil
}

// ReadBootstrapPayload reads one typed payload entry from SSRBootstrap.Data.
func ReadBootstrapPayload[T any](parseBootstrap SSRBootstrap, parseKey string) (SSRPayloadValue[T], bool, error) {
	parseTrimmedKey := strings.TrimSpace(parseKey)
	if parseTrimmedKey == "" {
		return SSRPayloadValue[T]{}, false, fmt.Errorf("ui: payload key cannot be empty")
	}
	parseRaw, parseOk := parseBootstrap.Data[parseTrimmedKey]
	if !parseOk {
		return SSRPayloadValue[T]{}, false, nil
	}
	parseEnvelope, _, parseErr := envelopeFromBootstrapData(parseRaw)
	if parseErr != nil {
		return SSRPayloadValue[T]{}, false, parseErr
	}
	parseValue, parseErr := decodeSSRPayloadEnvelope[T](parseEnvelope)
	if parseErr != nil {
		return SSRPayloadValue[T]{}, false, parseErr
	}
	return SSRPayloadValue[T]{
		Key:         parseTrimmedKey,
		Kind:        parseEnvelope.Kind,
		Scope:       parseEnvelope.Scope,
		Target:      parseEnvelope.Target,
		ReusePolicy: parseEnvelope.ReusePolicy,
		Revision:    parseEnvelope.Revision,
		Encoding:    parseEnvelope.Encoding,
		Value:       parseValue,
	}, true, nil
}

// RegisterRouteBootstrapData stores typed route data under a route-scoped payload key.
func RegisterRouteBootstrapData[T any](parseBootstrap *SSRBootstrap, parseKey string, parseRoutePath string, parseValue T, parseOptions ...SSRPayloadOptions) error {
	parseResolved := normalizeSSRPayloadOptions(parseOptions)
	parseResolved.Kind = SSRPayloadKindRouteData
	parseResolved.Scope = SSRPayloadScopeRoute
	parseResolved.Target = strings.TrimSpace(parseRoutePath)
	return RegisterBootstrapPayload(parseBootstrap, routeBootstrapPayloadKey(parseKey, parseRoutePath), parseValue, parseResolved)
}

// ReadRouteBootstrapData reads typed route data from a route-scoped payload key.
func ReadRouteBootstrapData[T any](parseBootstrap SSRBootstrap, parseKey string, parseRoutePath string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](parseBootstrap, routeBootstrapPayloadKey(parseKey, parseRoutePath))
}

// RegisterFormBootstrapDefaults stores typed form defaults under a form-scoped payload key.
func RegisterFormBootstrapDefaults[T any](parseBootstrap *SSRBootstrap, parseFormID string, parseValue T, parseOptions ...SSRPayloadOptions) error {
	parseTrimmedFormID := strings.TrimSpace(parseFormID)
	parseResolved := normalizeSSRPayloadOptions(parseOptions)
	parseResolved.Kind = SSRPayloadKindFormDefaults
	parseResolved.Scope = SSRPayloadScopeSubtree
	parseResolved.Target = parseTrimmedFormID
	return RegisterBootstrapPayload(parseBootstrap, formBootstrapPayloadPrefix+parseTrimmedFormID, parseValue, parseResolved)
}

// ReadFormBootstrapDefaults reads typed form defaults from a form-scoped payload key.
func ReadFormBootstrapDefaults[T any](parseBootstrap SSRBootstrap, parseFormID string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](parseBootstrap, formBootstrapPayloadPrefix+strings.TrimSpace(parseFormID))
}

// RegisterCacheBootstrapSeed stores a typed cache seed under a cache-scoped payload key.
func RegisterCacheBootstrapSeed[T any](parseBootstrap *SSRBootstrap, cacheKey string, parseValue T, parseOptions ...SSRPayloadOptions) error {
	parseTrimmedCacheKey := strings.TrimSpace(cacheKey)
	parseResolved := normalizeSSRPayloadOptions(parseOptions)
	parseResolved.Kind = SSRPayloadKindCacheSeed
	parseResolved.Scope = SSRPayloadScopeRoute
	parseResolved.Target = parseTrimmedCacheKey
	if parseResolved.ReusePolicy == "" {
		parseResolved.ReusePolicy = SSRPayloadReuseRevalidateAfterResume
	}
	return RegisterBootstrapPayload(parseBootstrap, cacheBootstrapPayloadPrefix+parseTrimmedCacheKey, parseValue, parseResolved)
}

// ReadCacheBootstrapSeed reads a typed cache seed from a cache-scoped payload key.
func ReadCacheBootstrapSeed[T any](parseBootstrap SSRBootstrap, cacheKey string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](parseBootstrap, cacheBootstrapPayloadPrefix+strings.TrimSpace(cacheKey))
}

// RegisterSessionBootstrapHint stores a typed session hint under an app-scoped payload key.
func RegisterSessionBootstrapHint[T any](parseBootstrap *SSRBootstrap, parseHintKey string, parseValue T, parseOptions ...SSRPayloadOptions) error {
	parseTrimmedHintKey := strings.TrimSpace(parseHintKey)
	parseResolved := normalizeSSRPayloadOptions(parseOptions)
	parseResolved.Kind = SSRPayloadKindSessionHint
	parseResolved.Scope = SSRPayloadScopeApp
	parseResolved.Target = parseTrimmedHintKey
	if parseResolved.ReusePolicy == "" {
		parseResolved.ReusePolicy = SSRPayloadReuseClientOwned
	}
	return RegisterBootstrapPayload(parseBootstrap, sessionBootstrapPayloadPrefix+parseTrimmedHintKey, parseValue, parseResolved)
}

// ReadSessionBootstrapHint reads a typed session hint from an app-scoped payload key.
func ReadSessionBootstrapHint[T any](parseBootstrap SSRBootstrap, parseHintKey string) (SSRPayloadValue[T], bool, error) {
	return ReadBootstrapPayload[T](parseBootstrap, sessionBootstrapPayloadPrefix+strings.TrimSpace(parseHintKey))
}

// routeBootstrapPayloadKey is a core package helper.
func routeBootstrapPayloadKey(parseKey string, parseRoutePath string) string {
	parseTrimmedKey := strings.TrimSpace(parseKey)
	if parseTrimmedKey == "" {
		parseTrimmedKey = DefaultRouteBootstrapPayloadKey
	}
	return strings.TrimSpace(parseRoutePath) + "::" + parseTrimmedKey
}

// InspectBootstrapPayloads lists typed payload registrations and legacy payloads, optionally filtered by kind, scope, or target.
func InspectBootstrapPayloads(parseBootstrap SSRBootstrap, filter SSRPayloadFilter) ([]SSRPayloadMetadata, error) {
	if len(parseBootstrap.Data) == 0 {
		return nil, nil
	}
	parseKeys := make([]string, 0, len(parseBootstrap.Data))
	for parseKey := range parseBootstrap.Data {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseItems := make([]SSRPayloadMetadata, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseEnvelope, parseLegacy, parseErr := envelopeFromBootstrapData(parseBootstrap.Data[parseKey2])
		if parseErr != nil {
			return nil, parseErr
		}
		if filter.Kind != "" && parseEnvelope.Kind != filter.Kind {
			continue
		}
		if filter.Scope != "" && parseEnvelope.Scope != filter.Scope {
			continue
		}
		if strings.TrimSpace(filter.Target) != "" && strings.TrimSpace(parseEnvelope.Target) != strings.TrimSpace(filter.Target) {
			continue
		}
		parseItems = append(parseItems, SSRPayloadMetadata{
			Key:         parseKey2,
			Version:     parseEnvelope.Version,
			Kind:        parseEnvelope.Kind,
			Scope:       parseEnvelope.Scope,
			Target:      parseEnvelope.Target,
			ReusePolicy: parseEnvelope.ReusePolicy,
			Revision:    parseEnvelope.Revision,
			Encoding:    parseEnvelope.Encoding,
			Legacy:      parseLegacy,
		})
	}
	return parseItems, nil
}

// normalizeSSRStateUpdate is a core package helper.
func normalizeSSRStateUpdate(parseUpdate SSRStateUpdate) (SSRStateUpdate, error) {
	parseVersion, parseErr := normalizeSSRStateUpdateVersion(parseUpdate.Version)
	if parseErr != nil {
		return SSRStateUpdate{}, parseErr
	}
	parseUpdate.Version = parseVersion
	if parseUpdate.Scope == "" {
		parseUpdate.Scope = SSRPayloadScopeApp
	}
	if parseUpdate.Upserts == nil {
		parseUpdate.Upserts = map[string]SSRPayloadEnvelope{}
	}
	// Finding #59: collect all key/envelope changes first, then apply them
	// after the loop ends to avoid inserting new keys during range iteration
	// (unspecified visitation behavior per the Go specification).
	type parseUpsertChange struct {
		oldKey string
		newKey string
		env    SSRPayloadEnvelope
	}
	parseChanges := make([]parseUpsertChange, 0, len(parseUpdate.Upserts))
	for parseKey, parseEnvelope := range parseUpdate.Upserts {
		parseTrimmedKey := strings.TrimSpace(parseKey)
		if parseTrimmedKey == "" {
			return SSRStateUpdate{}, fmt.Errorf("ui: state update upsert key cannot be empty")
		}
		if parseEnvelope.Scope == "" {
			parseEnvelope.Scope = parseUpdate.Scope
		}
		if parseEnvelope.Target == "" && parseUpdate.Target != "" {
			parseEnvelope.Target = parseUpdate.Target
		}
		if parseEnvelope.Version == 0 {
			parseEnvelope.Version = CurrentSSRBootstrapVersion
		}
		parseNormalized, parseErr2 := normalizeSSRPayloadEnvelope(parseEnvelope)
		if parseErr2 != nil {
			return SSRStateUpdate{}, parseErr2
		}
		parseChanges = append(parseChanges, parseUpsertChange{oldKey: parseKey, newKey: parseTrimmedKey, env: parseNormalized})
	}
	for _, parseChange := range parseChanges {
		if parseChange.newKey != parseChange.oldKey {
			delete(parseUpdate.Upserts, parseChange.oldKey)
		}
		parseUpdate.Upserts[parseChange.newKey] = parseChange.env
	}
	if len(parseUpdate.Deletes) > 0 {
		parseTrimmedDeletes := make([]string, 0, len(parseUpdate.Deletes))
		parseSeen := map[string]bool{}
		for _, parseKey2 := range parseUpdate.Deletes {
			parseTrimmedKey2 := strings.TrimSpace(parseKey2)
			if parseTrimmedKey2 == "" || parseSeen[parseTrimmedKey2] {
				continue
			}
			parseTrimmedDeletes = append(parseTrimmedDeletes, parseTrimmedKey2)
			parseSeen[parseTrimmedKey2] = true
		}
		parseUpdate.Deletes = parseTrimmedDeletes
	}
	return parseUpdate, nil
}

// normalizeSSRStateUpdateVersion is a core package helper.
func normalizeSSRStateUpdateVersion(parseVersion int) (int, error) {
	if parseVersion < 0 {
		return 0, fmt.Errorf("ui: unsupported SSR state update version %d", parseVersion)
	}
	if parseVersion == 0 {
		return CurrentSSRStateUpdateVersion, nil
	}
	if parseVersion > CurrentSSRStateUpdateVersion {
		return 0, fmt.Errorf("ui: unsupported SSR state update version %d", parseVersion)
	}
	return parseVersion, nil
}

// RegisterStateUpdatePayload stores one typed payload upsert in a state-update envelope.
func RegisterStateUpdatePayload[T any](parseUpdate *SSRStateUpdate, parseKey string, parseValue T, parseOptions ...SSRPayloadOptions) error {
	if parseUpdate == nil {
		return fmt.Errorf("ui: state update cannot be nil")
	}
	parseTrimmedKey := strings.TrimSpace(parseKey)
	if parseTrimmedKey == "" {
		return fmt.Errorf("ui: state update payload key cannot be empty")
	}
	parseResolved := normalizeSSRPayloadOptions(parseOptions)
	if parseResolved.Scope == "" && parseUpdate.Scope != "" {
		parseResolved.Scope = parseUpdate.Scope
	}
	if parseResolved.Target == "" && parseUpdate.Target != "" {
		parseResolved.Target = parseUpdate.Target
	}
	parseEnvelope, parseErr := encodeSSRPayloadEnvelope(parseValue, parseResolved)
	if parseErr != nil {
		return parseErr
	}
	if parseUpdate.Upserts == nil {
		parseUpdate.Upserts = map[string]SSRPayloadEnvelope{}
	}
	parseUpdate.Upserts[parseTrimmedKey] = parseEnvelope
	return nil
}

// MarshalSSRStateUpdateText encodes a state-update envelope as JSON text.
func MarshalSSRStateUpdateText(parseUpdate SSRStateUpdate) ([]byte, error) {
	parseUpdate, parseErr := normalizeSSRStateUpdate(parseUpdate)
	if parseErr != nil {
		return nil, parseErr
	}
	parseEncoded, parseErr := json.Marshal(parseUpdate)
	if parseErr != nil {
		return nil, parseErr
	}
	return []byte(escapeJSONForInlineScript(string(parseEncoded))), nil
}

// UnmarshalSSRStateUpdateText decodes a JSON text state-update envelope.
func UnmarshalSSRStateUpdateText(parseData []byte) (SSRStateUpdate, error) {
	if len(parseData) == 0 {
		return normalizeSSRStateUpdate(SSRStateUpdate{})
	}
	var parseUpdate SSRStateUpdate
	if parseErr := json.Unmarshal(parseData, &parseUpdate); parseErr != nil {
		return SSRStateUpdate{}, parseErr
	}
	return normalizeSSRStateUpdate(parseUpdate)
}

// MarshalSSRStateUpdateBinary encodes a state-update envelope as CBOR.
func MarshalSSRStateUpdateBinary(parseUpdate SSRStateUpdate) ([]byte, error) {
	parseUpdate, parseErr := normalizeSSRStateUpdate(parseUpdate)
	if parseErr != nil {
		return nil, parseErr
	}
	return cbor.Marshal(parseUpdate)
}

// UnmarshalSSRStateUpdateBinary decodes a CBOR state-update envelope.
func UnmarshalSSRStateUpdateBinary(parseData []byte) (SSRStateUpdate, error) {
	if len(parseData) == 0 {
		return normalizeSSRStateUpdate(SSRStateUpdate{})
	}
	var parseUpdate SSRStateUpdate
	if parseErr := cbor.Unmarshal(parseData, &parseUpdate); parseErr != nil {
		return SSRStateUpdate{}, parseErr
	}
	return normalizeSSRStateUpdate(parseUpdate)
}

// ApplySSRStateUpdate merges a text or binary update envelope into a bootstrap payload snapshot.
func ApplySSRStateUpdate(parseBootstrap *SSRBootstrap, parseUpdate SSRStateUpdate) error {
	if parseBootstrap == nil {
		return fmt.Errorf("ui: bootstrap cannot be nil")
	}
	parseUpdate, parseErr := normalizeSSRStateUpdate(parseUpdate)
	if parseErr != nil {
		return parseErr
	}
	if parseBootstrap.Data == nil {
		parseBootstrap.Data = map[string]any{}
	}
	for _, parseKey := range parseUpdate.Deletes {
		delete(parseBootstrap.Data, parseKey)
	}
	for parseKey2, parseEnvelope := range parseUpdate.Upserts {
		parseBootstrap.Data[parseKey2] = parseEnvelope
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

// normalizeSSRBootstrapBudget is a core package helper.
func normalizeSSRBootstrapBudget(parseBudget SSRBootstrapBudget) SSRBootstrapBudget {
	parseDefaults := NewSSRBootstrapBudget()
	if parseBudget.InlineWarnBytes <= 0 {
		parseBudget.InlineWarnBytes = parseDefaults.InlineWarnBytes
	}
	if parseBudget.InlineErrorBytes <= 0 || parseBudget.InlineErrorBytes < parseBudget.InlineWarnBytes {
		parseBudget.InlineErrorBytes = parseDefaults.InlineErrorBytes
	}
	if parseBudget.SidecarWarnBytes <= 0 {
		parseBudget.SidecarWarnBytes = parseDefaults.SidecarWarnBytes
	}
	if parseBudget.SidecarErrorBytes <= 0 || parseBudget.SidecarErrorBytes < parseBudget.SidecarWarnBytes {
		parseBudget.SidecarErrorBytes = parseDefaults.SidecarErrorBytes
	}
	if parseBudget.BinaryWarnBytes <= 0 {
		parseBudget.BinaryWarnBytes = parseDefaults.BinaryWarnBytes
	}
	if parseBudget.BinaryErrorBytes <= 0 || parseBudget.BinaryErrorBytes < parseBudget.BinaryWarnBytes {
		parseBudget.BinaryErrorBytes = parseDefaults.BinaryErrorBytes
	}
	return parseBudget
}

// InspectSSRBootstrapSize measures payload sizes, budget bands, and the recommended transport mode.
func InspectSSRBootstrapSize(parsePayload SSRBootstrap, parseBudget SSRBootstrapBudget) (SSRBootstrapSizeReport, error) {
	parseBudget = normalizeSSRBootstrapBudget(parseBudget)
	parseNormalized, parseErr := normalizeSSRBootstrap(parsePayload)
	if parseErr != nil {
		return SSRBootstrapSizeReport{}, parseErr
	}
	parseJsonPayload, parseErr := marshalSSRBootstrapJSON(parseNormalized)
	if parseErr != nil {
		return SSRBootstrapSizeReport{}, parseErr
	}
	parseBinaryPayload, parseErr := marshalSSRBootstrapBinary(parseNormalized)
	if parseErr != nil {
		return SSRBootstrapSizeReport{}, parseErr
	}
	parseInlineScript := `<script id="` + DefaultBootstrapScriptID + `" type="application/json">` + string(parseJsonPayload) + `</script>`
	parsePayloads, parseErr := InspectBootstrapPayloads(parseNormalized, SSRPayloadFilter{})
	if parseErr != nil {
		return SSRBootstrapSizeReport{}, parseErr
	}
	parseReport := SSRBootstrapSizeReport{
		Version:            parseNormalized.Version,
		JSONPayloadBytes:   len(parseJsonPayload),
		InlineScriptBytes:  len(parseInlineScript),
		BinaryPayloadBytes: len(parseBinaryPayload),
		Recommendation:     "inline-json",
		Payloads:           parsePayloads,
	}
	if parseReport.InlineScriptBytes >= parseBudget.InlineWarnBytes {
		parseReport.Warnings = append(parseReport.Warnings, fmt.Sprintf("inline bootstrap script is %d bytes", parseReport.InlineScriptBytes))
	}
	if parseReport.InlineScriptBytes >= parseBudget.InlineErrorBytes {
		parseReport.Errors = append(parseReport.Errors, fmt.Sprintf("inline bootstrap script exceeds the %d byte inline budget", parseBudget.InlineErrorBytes))
	}
	if parseReport.JSONPayloadBytes >= parseBudget.SidecarWarnBytes {
		parseReport.Warnings = append(parseReport.Warnings, fmt.Sprintf("JSON bootstrap payload is %d bytes", parseReport.JSONPayloadBytes))
	}
	if parseReport.JSONPayloadBytes >= parseBudget.SidecarErrorBytes {
		parseReport.Errors = append(parseReport.Errors, fmt.Sprintf("JSON bootstrap payload exceeds the %d byte sidecar budget", parseBudget.SidecarErrorBytes))
	}
	if parseReport.BinaryPayloadBytes >= parseBudget.BinaryWarnBytes {
		parseReport.Warnings = append(parseReport.Warnings, fmt.Sprintf("binary bootstrap payload is %d bytes", parseReport.BinaryPayloadBytes))
	}
	if parseReport.BinaryPayloadBytes >= parseBudget.BinaryErrorBytes {
		parseReport.Errors = append(parseReport.Errors, fmt.Sprintf("binary bootstrap payload exceeds the %d byte binary budget", parseBudget.BinaryErrorBytes))
	}
	if parseReport.InlineScriptBytes >= parseBudget.InlineWarnBytes {
		parseReport.Recommendation = "sidecar-json"
	}
	if parseReport.BinaryPayloadBytes < parseReport.JSONPayloadBytes && (parseReport.JSONPayloadBytes >= parseBudget.SidecarWarnBytes || parseReport.InlineScriptBytes >= parseBudget.InlineWarnBytes) {
		parseReport.Recommendation = "sidecar-cbor"
	}
	return parseReport, nil
}
