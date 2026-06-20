package kvstate

import (
	"encoding/json"

	cbor "github.com/fxamacker/cbor/v2"
)

// Codec encodes and decodes persisted values. Implement it to control the wire
// format (e.g. a compact or encrypted representation).
type Codec interface {
	Encode(parseValue any) ([]byte, error)
	Decode(parseData []byte, parseTarget any) error
}

// JSONCodec encodes values as JSON. It is the default.
type JSONCodec struct{}

func (JSONCodec) Encode(parseValue any) ([]byte, error) { return json.Marshal(parseValue) }

func (JSONCodec) Decode(parseData []byte, parseTarget any) error {
	return json.Unmarshal(parseData, parseTarget)
}

// CBORCodec encodes values as CBOR (compact binary). Uses the cbor dependency
// already vendored by the module.
type CBORCodec struct{}

func (CBORCodec) Encode(parseValue any) ([]byte, error) { return cbor.Marshal(parseValue) }

func (CBORCodec) Decode(parseData []byte, parseTarget any) error {
	return cbor.Unmarshal(parseData, parseTarget)
}
