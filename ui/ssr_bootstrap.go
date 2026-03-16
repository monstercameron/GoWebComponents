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

type SSRBootstrap struct {
	Route  SSRRouteBootstrap      `json:"route,omitempty"`
	Atoms  map[string]interface{} `json:"atoms,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
	IDSeed int                    `json:"idSeed,omitempty"`
}

type SSRRouteBootstrap struct {
	Path   string              `json:"path,omitempty"`
	Query  map[string][]string `json:"query,omitempty"`
	Params map[string]string   `json:"params,omitempty"`
}

type SSRBootstrapReference struct {
	URL    string `json:"url,omitempty"`
	Format string `json:"format,omitempty"`
}

func MarshalSSRBootstrap(payload SSRBootstrap) ([]byte, error) {
	data, err := json.Marshal(payload)
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
	return []byte(replacer.Replace(string(data))), nil
}

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

func MarshalSSRBootstrapBinary(payload SSRBootstrap) ([]byte, error) {
	return cbor.Marshal(payload)
}

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
	return payload
}

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

func RenderBootstrapReferenceScript(ref SSRBootstrapReference, scriptID string) (string, error) {
	encoded, err := json.Marshal(ref)
	if err != nil {
		return "", err
	}

	id := scriptID
	if id == "" {
		id = DefaultBootstrapReferenceScriptID
	}

	return `<script id="` + html.EscapeString(id) + `" type="application/json" data-gwc-bootstrap-ref="true">` + html.EscapeString(string(encoded)) + `</script>`, nil
}

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
