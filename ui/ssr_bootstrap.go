package ui

import (
	"encoding/json"
	"html"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

const DefaultBootstrapScriptID = "__GWC_BOOTSTRAP__"
const DefaultBootstrapReferenceScriptID = "__GWC_BOOTSTRAP_REF__"

const (
	SSRBootstrapFormatJSON = "json"
	SSRBootstrapFormatCBOR = "cbor"
)

// SSRBootstrap captures the server-provided state needed to resume a route on the client.
type SSRBootstrap struct {
	Route  SSRRouteBootstrap      `json:"route,omitempty"`
	Atoms  map[string]interface{} `json:"atoms,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
	I18n   SSRI18nBootstrap       `json:"i18n,omitempty"`
	IDSeed int                    `json:"idSeed,omitempty"`
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
	URL    string `json:"url,omitempty"`
	Format string `json:"format,omitempty"`
}

// MarshalSSRBootstrap serializes a bootstrap payload to safe inline JSON.
func MarshalSSRBootstrap(payload SSRBootstrap) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	replacer := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
		"\u2028", `\u2028`,
		"\u2029", `\u2029`,
	)
	return []byte(replacer.Replace(string(jsonData))), nil
}

// UnmarshalSSRBootstrap deserializes a JSON bootstrap payload.
func UnmarshalSSRBootstrap(data []byte) (SSRBootstrap, error) {
	if len(data) == 0 {
		return SSRBootstrap{}, nil
	}

	var payload SSRBootstrap
	if err := json.Unmarshal(data, &payload); err != nil {
		return SSRBootstrap{}, err
	}
	return normalizeSSRBootstrap(payload), nil
}

// MarshalSSRBootstrapBinary serializes a bootstrap payload to CBOR.
func MarshalSSRBootstrapBinary(payload SSRBootstrap) ([]byte, error) {
	return cbor.Marshal(payload)
}

// UnmarshalSSRBootstrapBinary deserializes a CBOR bootstrap payload.
func UnmarshalSSRBootstrapBinary(data []byte) (SSRBootstrap, error) {
	if len(data) == 0 {
		return SSRBootstrap{}, nil
	}

	var payload SSRBootstrap
	if err := cbor.Unmarshal(data, &payload); err != nil {
		return SSRBootstrap{}, err
	}
	return normalizeSSRBootstrap(payload), nil
}

func normalizeSSRBootstrap(payload SSRBootstrap) SSRBootstrap {
	if payload.Route.Query == nil {
		payload.Route.Query = map[string][]string{}
	}
	if payload.Route.Params == nil {
		payload.Route.Params = map[string]string{}
	}
	if payload.Atoms == nil {
		payload.Atoms = map[string]interface{}{}
	}
	if payload.Data == nil {
		payload.Data = map[string]interface{}{}
	}
	if payload.I18n.Messages == nil {
		payload.I18n.Messages = map[string]map[string]SSRI18nMessage{}
	}
	return payload
}

// RenderBootstrapScript renders an inline bootstrap script tag.
func RenderBootstrapScript(payload SSRBootstrap, scriptID string) (string, error) {
	encoded, err := MarshalSSRBootstrap(payload)
	if err != nil {
		return "", err
	}

	id := scriptID
	if id == "" {
		id = DefaultBootstrapScriptID
	}

	return `<script id="` + html.EscapeString(id) + `" type="application/json">` + string(encoded) + `</script>`, nil
}

// RenderBootstrapReferenceScript renders an inline script tag that points at an external bootstrap payload.
func RenderBootstrapReferenceScript(ref SSRBootstrapReference, scriptID string) (string, error) {
	encoded, err := json.Marshal(ref)
	if err != nil {
		return "", err
	}

	replacer := strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
		"\u2028", `\u2028`,
		"\u2029", `\u2029`,
	)
	safeJSON := replacer.Replace(string(encoded))

	id := scriptID
	if id == "" {
		id = DefaultBootstrapReferenceScriptID
	}

	return `<script id="` + html.EscapeString(id) + `" type="application/json" data-gwc-bootstrap-ref="true">` + safeJSON + `</script>`, nil
}

// UnmarshalSSRBootstrapReference deserializes a bootstrap reference payload.
func UnmarshalSSRBootstrapReference(data []byte) (SSRBootstrapReference, error) {
	if len(data) == 0 {
		return SSRBootstrapReference{}, nil
	}

	var ref SSRBootstrapReference
	if err := json.Unmarshal(data, &ref); err != nil {
		return SSRBootstrapReference{}, err
	}
	if ref.Format == "" {
		ref.Format = SSRBootstrapFormatJSON
	}
	return ref, nil
}
