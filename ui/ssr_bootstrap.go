package ui

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

const DefaultBootstrapScriptID = "__GWC_BOOTSTRAP__"
const DefaultBootstrapReferenceScriptID = "__GWC_BOOTSTRAP_REF__"

const CurrentSSRBootstrapVersion = 1

var errUnsupportedSSRBootstrapVersion = fmt.Errorf("ui: unsupported SSR bootstrap version")

const (
	SSRBootstrapFormatJSON = "json"
	SSRBootstrapFormatCBOR = "cbor"
)

// SSRBootstrap captures the server-provided state needed to resume a route on the client.
type SSRBootstrap struct {
	Version       int               `json:"version,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty"`
	Route         SSRRouteBootstrap `json:"route"`
	Atoms         map[string]any    `json:"atoms,omitempty"`
	Data          map[string]any    `json:"data,omitempty"`
	I18n          SSRI18nBootstrap  `json:"i18n"`
	IDSeed        int               `json:"idSeed,omitempty"`
}

type SSRI18nBootstrap struct {
	Locale         string                               `json:"locale,omitempty"`
	FallbackLocale string                               `json:"fallbackLocale,omitempty"`
	Direction      string                               `json:"direction,omitempty"`
	Messages       map[string]map[string]SSRI18nMessage `json:"messages,omitempty"`
}

type SSRI18nMessage struct {
	Text      string            `json:"text,omitempty"`
	PluralArg string            `json:"pluralArg,omitempty"`
	Plural    map[string]string `json:"plural,omitempty"`
	SelectArg string            `json:"selectArg,omitempty"`
	Select    map[string]string `json:"select,omitempty"`
	Default   string            `json:"default,omitempty"`
}

// SSRRouteBootstrap captures the routed path, query, and params transferred from server to client.
type SSRRouteBootstrap struct {
	Path   string              `json:"path,omitempty"`
	Query  map[string][]string `json:"query,omitempty"`
	Params map[string]string   `json:"params,omitempty"`
}

// SSRBootstrapReference points the client at an external bootstrap payload.
type SSRBootstrapReference struct {
	Version int    `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
	Format  string `json:"format,omitempty"`
}

type SSRScriptOptions struct {
	Nonce string
}

// marshalSSRBootstrapJSON is a core package helper.
func marshalSSRBootstrapJSON(parsePayload SSRBootstrap) ([]byte, error) {
	parsePayload, parseErr := normalizeSSRBootstrap(parsePayload)
	if parseErr != nil {
		return nil, parseErr
	}
	parseJsonData, parseErr := json.Marshal(parsePayload)
	if parseErr != nil {
		return nil, parseErr
	}
	return []byte(escapeJSONForInlineScript(string(parseJsonData))), nil
}

// MarshalSSRBootstrap serializes a bootstrap payload to safe inline JSON.
func MarshalSSRBootstrap(parsePayload SSRBootstrap) ([]byte, error) {
	return MarshalSSRBootstrapObserved(parsePayload, SSRObservabilityOptions{})
}

// MarshalSSRBootstrapObserved serializes a bootstrap payload and emits size metrics.
func MarshalSSRBootstrapObserved(parsePayload SSRBootstrap, parseOptions SSRObservabilityOptions) ([]byte, error) {
	parseEncoded, parseErr := marshalSSRBootstrapJSON(parsePayload)
	dispatchSSRObservation(parseOptions, newSSRBootstrapObservation(parseOptions, SSRBootstrapFormatJSON, len(parseEncoded), 0, parseErr))
	return parseEncoded, parseErr
}

// UnmarshalSSRBootstrap deserializes a JSON bootstrap payload.
func UnmarshalSSRBootstrap(parseData []byte) (SSRBootstrap, error) {
	if len(parseData) == 0 {
		return normalizeSSRBootstrap(SSRBootstrap{})
	}

	var parsePayload SSRBootstrap
	if parseErr := json.Unmarshal(parseData, &parsePayload); parseErr != nil {
		return SSRBootstrap{}, parseErr
	}
	return normalizeSSRBootstrap(parsePayload)
}

// marshalSSRBootstrapBinary is a core package helper.
func marshalSSRBootstrapBinary(parsePayload SSRBootstrap) ([]byte, error) {
	parsePayload, parseErr := normalizeSSRBootstrap(parsePayload)
	if parseErr != nil {
		return nil, parseErr
	}
	return cbor.Marshal(parsePayload)
}

// MarshalSSRBootstrapBinary serializes a bootstrap payload to CBOR.
func MarshalSSRBootstrapBinary(parsePayload SSRBootstrap) ([]byte, error) {
	return MarshalSSRBootstrapBinaryObserved(parsePayload, SSRObservabilityOptions{})
}

// MarshalSSRBootstrapBinaryObserved serializes a bootstrap payload to CBOR and emits size metrics.
func MarshalSSRBootstrapBinaryObserved(parsePayload SSRBootstrap, parseOptions SSRObservabilityOptions) ([]byte, error) {
	parseEncoded, parseErr := marshalSSRBootstrapBinary(parsePayload)
	dispatchSSRObservation(parseOptions, newSSRBootstrapObservation(parseOptions, SSRBootstrapFormatCBOR, len(parseEncoded), 0, parseErr))
	return parseEncoded, parseErr
}

// UnmarshalSSRBootstrapBinary deserializes a CBOR bootstrap payload.
func UnmarshalSSRBootstrapBinary(parseData []byte) (SSRBootstrap, error) {
	if len(parseData) == 0 {
		return normalizeSSRBootstrap(SSRBootstrap{})
	}

	var parsePayload SSRBootstrap
	if parseErr := cbor.Unmarshal(parseData, &parsePayload); parseErr != nil {
		return SSRBootstrap{}, parseErr
	}
	return normalizeSSRBootstrap(parsePayload)
}

// normalizeSSRBootstrap is a core package helper.
func normalizeSSRBootstrap(parsePayload SSRBootstrap) (SSRBootstrap, error) {
	parseVersion, parseErr := normalizeSSRBootstrapVersion(parsePayload.Version)
	if parseErr != nil {
		return SSRBootstrap{}, parseErr
	}
	parsePayload.Version = parseVersion
	if parsePayload.Route.Query == nil {
		parsePayload.Route.Query = map[string][]string{}
	}
	if parsePayload.Route.Params == nil {
		parsePayload.Route.Params = map[string]string{}
	}
	if parsePayload.Atoms == nil {
		parsePayload.Atoms = map[string]any{}
	}
	if parsePayload.Data == nil {
		parsePayload.Data = map[string]any{}
	}
	if parsePayload.I18n.Messages == nil {
		parsePayload.I18n.Messages = map[string]map[string]SSRI18nMessage{}
	}
	return parsePayload, nil
}

// RenderBootstrapScript renders an inline bootstrap script tag.
func RenderBootstrapScript(parsePayload SSRBootstrap, parseScriptID string) (string, error) {
	return RenderBootstrapScriptWithOptions(parsePayload, parseScriptID, SSRScriptOptions{})
}

// RenderBootstrapScriptObserved renders an inline bootstrap script tag and emits size metrics.
func RenderBootstrapScriptObserved(parsePayload SSRBootstrap, parseScriptID string, parseOptions SSRObservabilityOptions) (string, error) {
	return renderBootstrapScript(parsePayload, parseScriptID, SSRScriptOptions{}, parseOptions)
}

func RenderBootstrapScriptWithOptions(parsePayload SSRBootstrap, parseScriptID string, parseScriptOptions SSRScriptOptions) (string, error) {
	return renderBootstrapScript(parsePayload, parseScriptID, parseScriptOptions, SSRObservabilityOptions{})
}

func renderBootstrapScript(parsePayload SSRBootstrap, parseScriptID string, parseScriptOptions SSRScriptOptions, parseOptions SSRObservabilityOptions) (string, error) {
	parseEncoded, parseErr := marshalSSRBootstrapJSON(parsePayload)
	if parseErr != nil {
		dispatchSSRObservation(parseOptions, newSSRBootstrapObservation(parseOptions, SSRBootstrapFormatJSON, 0, 0, parseErr))
		return "", parseErr
	}

	parseId := parseScriptID
	if parseId == "" {
		parseId = DefaultBootstrapScriptID
	}

	parseScript := `<script id="` + html.EscapeString(parseId) + `" type="application/json"` + renderSSRScriptNonceAttr(parseScriptOptions.Nonce) + `>` + string(parseEncoded) + `</script>`
	dispatchSSRObservation(parseOptions, newSSRBootstrapObservation(parseOptions, SSRBootstrapFormatJSON, len(parseEncoded), len(parseScript), nil))
	return parseScript, nil
}

// RenderBootstrapReferenceScript renders an inline script tag that points at an external bootstrap payload.
func RenderBootstrapReferenceScript(parseRef SSRBootstrapReference, parseScriptID string) (string, error) {
	return RenderBootstrapReferenceScriptWithOptions(parseRef, parseScriptID, SSRScriptOptions{})
}

func RenderBootstrapReferenceScriptWithOptions(parseRef SSRBootstrapReference, parseScriptID string, parseScriptOptions SSRScriptOptions) (string, error) {
	parseRef, parseErr := normalizeSSRBootstrapReference(parseRef)
	if parseErr != nil {
		return "", parseErr
	}
	parseEncoded, parseErr := json.Marshal(parseRef)
	if parseErr != nil {
		return "", parseErr
	}
	parseSafeJSON := escapeJSONForInlineScript(string(parseEncoded))

	parseId := parseScriptID
	if parseId == "" {
		parseId = DefaultBootstrapReferenceScriptID
	}

	return `<script id="` + html.EscapeString(parseId) + `" type="application/json" data-gwc-bootstrap-ref="true"` + renderSSRScriptNonceAttr(parseScriptOptions.Nonce) + `>` + parseSafeJSON + `</script>`, nil
}

// UnmarshalSSRBootstrapReference deserializes a bootstrap reference payload.
func UnmarshalSSRBootstrapReference(parseData []byte) (SSRBootstrapReference, error) {
	if len(parseData) == 0 {
		return normalizeSSRBootstrapReference(SSRBootstrapReference{})
	}

	var parseRef SSRBootstrapReference
	if parseErr := json.Unmarshal(parseData, &parseRef); parseErr != nil {
		return SSRBootstrapReference{}, parseErr
	}
	return normalizeSSRBootstrapReference(parseRef)
}

// normalizeSSRBootstrapVersion is a core package helper.
func normalizeSSRBootstrapVersion(parseVersion int) (int, error) {
	if parseVersion < 0 {
		return 0, fmt.Errorf("%w %d", errUnsupportedSSRBootstrapVersion, parseVersion)
	}
	if parseVersion == 0 {
		return CurrentSSRBootstrapVersion, nil
	}
	if parseVersion > CurrentSSRBootstrapVersion {
		return 0, fmt.Errorf("%w %d", errUnsupportedSSRBootstrapVersion, parseVersion)
	}
	return parseVersion, nil
}

// normalizeSSRBootstrapReference is a core package helper.
func normalizeSSRBootstrapReference(parseRef SSRBootstrapReference) (SSRBootstrapReference, error) {
	parseVersion, parseErr := normalizeSSRBootstrapVersion(parseRef.Version)
	if parseErr != nil {
		return SSRBootstrapReference{}, parseErr
	}
	parseRef.Version = parseVersion
	if parseRef.Format == "" {
		parseRef.Format = SSRBootstrapFormatJSON
	}
	return parseRef, nil
}

func renderSSRScriptNonceAttr(parseNonce string) string {
	parseNonce = strings.TrimSpace(parseNonce)
	if parseNonce == "" {
		return ""
	}
	return ` nonce="` + html.EscapeString(parseNonce) + `"`
}
