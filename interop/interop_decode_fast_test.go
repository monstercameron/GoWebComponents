package interop

import (
	"encoding"
	"encoding/json"
	"reflect"
	"testing"
)

type decodeFastPayload struct {
	ID      string            `json:"id"`
	Enabled bool              `json:"enabled"`
	Meta    map[string]string `json:"meta"`
}

type decodeFastNumeric struct {
	Count int `json:"count"`
}

type decodeFastSlice struct {
	Items []string `json:"items"`
}

type decodeFastNested struct {
	Value struct {
		Name string `json:"name"`
	} `json:"value"`
}

type decodeFastEmbedded struct {
	decodeFastEmbeddedFields
}

type decodeFastEmbeddedFields struct {
	Name string `json:"name"`
}

type decodeFastCustom struct {
	Name string `json:"name"`
}

func (parsePayload *decodeFastCustom) UnmarshalJSON(parseData []byte) error {
	parsePayload.Name = "custom"
	return nil
}

type decodeFastPointer struct {
	Name *string `json:"name"`
}

type decodeFastStringOption struct {
	ID string `json:"id,string"`
}

type decodeFastMapAlias struct {
	Meta map[decodeFastTextKey]decodeFastJSONValue `json:"meta"`
}

type decodeFastTextKey string

func (parseValue *decodeFastTextKey) UnmarshalText([]byte) error {
	*parseValue = "text"
	return nil
}

type decodeFastJSONValue string

func (parseValue *decodeFastJSONValue) UnmarshalJSON([]byte) error {
	*parseValue = "json"
	return nil
}

var _ encoding.TextUnmarshaler = (*decodeFastTextKey)(nil)

type decodeFastCustomField struct {
	Name decodeFastCustomString `json:"name"`
}

type decodeFastTextField struct {
	Name decodeFastTextValue `json:"name"`
}

type decodeFastTextMapValue struct {
	Meta map[string]decodeFastTextValue `json:"meta"`
}

type decodeFastTextValue string

func (parseValue *decodeFastTextValue) UnmarshalText(parseText []byte) error {
	*parseValue = decodeFastTextValue("text:" + string(parseText))
	return nil
}

type decodeFastCustomString string

func (parseValue *decodeFastCustomString) UnmarshalJSON([]byte) error {
	*parseValue = "custom"
	return nil
}

func TestDecodeSimpleMapToStructCompatibility(t *testing.T) {
	parseCases := []struct {
		name       string
		value      map[string]any
		wantFast   bool
		wantResult decodeFastPayload
	}{
		{
			name:     "exact tags and nested string map",
			value:    map[string]any{"id": "bench", "enabled": true, "meta": map[string]any{"lane": "interop"}},
			wantFast: true,
			wantResult: decodeFastPayload{
				ID: "bench", Enabled: true, Meta: map[string]string{"lane": "interop"},
			},
		},
		{
			name:       "unknown field",
			value:      map[string]any{"id": "bench", "unknown": "ignored"},
			wantFast:   false,
			wantResult: decodeFastPayload{ID: "bench"},
		},
		{
			name:       "case insensitive field falls back",
			value:      map[string]any{"ID": "bench"},
			wantFast:   false,
			wantResult: decodeFastPayload{ID: "bench"},
		},
		{
			name:       "nil map falls back",
			value:      map[string]any{"meta": nil},
			wantFast:   false,
			wantResult: decodeFastPayload{Meta: nil},
		},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			var parseGot decodeFastPayload
			parseFast, parseErr := decodeSimpleMapToStruct(parseCase.value, &parseGot)
			if parseErr != nil || parseFast != parseCase.wantFast {
				t.Fatalf("fast path: handled=%v err=%v, want handled=%v", parseFast, parseErr, parseCase.wantFast)
			}
			if !parseCase.wantFast {
				if parseErr := Decode(parseCase.value, &parseGot); parseErr != nil {
					t.Fatalf("Decode fallback: %v", parseErr)
				}
			}
			if !reflect.DeepEqual(parseGot, parseCase.wantResult) {
				t.Fatalf("decoded result %#v, want %#v", parseGot, parseCase.wantResult)
			}
		})
	}
}

func TestDecodeSimpleMapToStructFallsBackForUnsupportedShapes(t *testing.T) {
	parseCases := []struct {
		name   string
		value  map[string]any
		target any
	}{
		{name: "numeric", value: map[string]any{"count": float64(2)}, target: &decodeFastNumeric{}},
		{name: "slice", value: map[string]any{"items": []any{"a"}}, target: &decodeFastSlice{}},
		{name: "nested struct", value: map[string]any{"value": map[string]any{"name": "nested"}}, target: &decodeFastNested{}},
		{name: "embedded field", value: map[string]any{"name": "embedded"}, target: &decodeFastEmbedded{}},
		{name: "custom unmarshaler", value: map[string]any{"name": "input"}, target: &decodeFastCustom{}},
		{name: "custom field unmarshaler", value: map[string]any{"name": "input"}, target: &decodeFastCustomField{}},
		{name: "text field unmarshaler", value: map[string]any{"name": "input"}, target: &decodeFastTextField{}},
		{name: "text map value unmarshaler", value: map[string]any{"meta": map[string]any{"key": "input"}}, target: &decodeFastTextMapValue{}},
		{name: "pointer field", value: map[string]any{"name": "pointer"}, target: &decodeFastPointer{}},
		{name: "string tag option", value: map[string]any{"id": `"7"`}, target: &decodeFastStringOption{}},
		{name: "map alias hooks", value: map[string]any{"meta": map[string]any{"key": "value"}}, target: &decodeFastMapAlias{}},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			if parseFast, parseErr := decodeSimpleMapToStruct(parseCase.value, parseCase.target); parseFast || parseErr != nil {
				t.Fatalf("fast path handled unsupported shape: handled=%v err=%v", parseFast, parseErr)
			}
			parseWantData, parseErr := json.Marshal(parseCase.value)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			parseWant := reflect.New(reflect.ValueOf(parseCase.target).Elem().Type()).Interface()
			if parseErr := json.Unmarshal(parseWantData, parseWant); parseErr != nil {
				t.Fatal(parseErr)
			}
			if parseErr := Decode(parseCase.value, parseCase.target); parseErr != nil {
				t.Fatalf("Decode fallback: %v", parseErr)
			}
			if !reflect.DeepEqual(parseCase.target, parseWant) {
				t.Fatalf("decoded result %#v, want %#v", parseCase.target, parseWant)
			}
		})
	}
}

func TestDecodeSimpleMapToStructPreservesExistingMapEntries(t *testing.T) {
	parseTarget := &decodeFastPayload{Meta: map[string]string{"old": "value"}}
	parseValue := map[string]any{"meta": map[string]any{"new": "value"}}
	if parseFast, parseErr := decodeSimpleMapToStruct(parseValue, parseTarget); !parseFast || parseErr != nil {
		t.Fatalf("fast path: handled=%v err=%v", parseFast, parseErr)
	}
	parseWant := map[string]string{"old": "value", "new": "value"}
	if !reflect.DeepEqual(parseTarget.Meta, parseWant) {
		t.Fatalf("decoded map %#v, want %#v", parseTarget.Meta, parseWant)
	}
}

func TestDecodeSimpleMapToStructTypedNilMapMatchesJSON(t *testing.T) {
	parseValue := map[string]any{"meta": map[string]any(nil)}
	parseCases := []struct {
		name   string
		target *decodeFastPayload
	}{
		{name: "nil target", target: &decodeFastPayload{}},
		{name: "preexisting target", target: &decodeFastPayload{Meta: map[string]string{"old": "value"}}},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			parseWant := &decodeFastPayload{}
			parseWantData, parseErr := json.Marshal(parseValue)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if parseErr := json.Unmarshal(parseWantData, parseWant); parseErr != nil {
				t.Fatal(parseErr)
			}
			if parseFast, parseErr := decodeSimpleMapToStruct(parseValue, parseCase.target); parseFast || parseErr != nil {
				t.Fatalf("fast path: handled=%v err=%v, want fallback", parseFast, parseErr)
			}
			if parseErr := Decode(parseValue, parseCase.target); parseErr != nil {
				t.Fatalf("Decode fallback: %v", parseErr)
			}
			if !reflect.DeepEqual(parseCase.target, parseWant) {
				t.Fatalf("decoded result %#v, want %#v", parseCase.target, parseWant)
			}
		})
	}
}

func TestDecodeSimpleMapToStructUnknownUnsupportedValueFallsBack(t *testing.T) {
	parseTarget := &decodeFastPayload{}
	parseValue := map[string]any{"id": "bench", "unknown": make(chan int)}
	if parseFast, parseErr := decodeSimpleMapToStruct(parseValue, parseTarget); parseFast || parseErr != nil {
		t.Fatalf("fast path: handled=%v err=%v", parseFast, parseErr)
	}
	if parseErr := Decode(parseValue, parseTarget); !IsCode(parseErr, CodeEncode) {
		t.Fatalf("Decode error = %v, want CodeEncode", parseErr)
	}
}
