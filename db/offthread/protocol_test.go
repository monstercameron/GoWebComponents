package offthread

import (
	"bytes"
	"encoding/json"
	"testing"
)

// v5 P3.5 — the wire protocol.
//
// The property that matters is TOTALITY: every SQLite storage class has exactly
// one representation, and a value survives an encode/decode without the decoder
// inferring its type from what it happens to look like.

func TestValueRoundTripsEveryStorageClass(parseT *testing.T) {
	for _, parseCase := range []struct {
		label string
		input any
		kind  ValueKind
	}{
		{"null", nil, ValueNull},
		{"integer", int64(-9000), ValueInt},
		{"real", 3.5, ValueFloat},
		{"text", "hello", ValueText},
		{"blob", []byte{0, 1, 255}, ValueBlob},
	} {
		parseValue, parseErr := NewValue(parseCase.input)
		if parseErr != nil {
			parseT.Fatalf("%s: NewValue: %v", parseCase.label, parseErr)
		}
		if parseValue.Kind != parseCase.kind {
			parseT.Errorf("%s: kind = %s, want %s", parseCase.label, parseValue.Kind, parseCase.kind)
		}

		parseEncoded, parseEncodeErr := json.Marshal(parseValue)
		if parseEncodeErr != nil {
			parseT.Fatalf("%s: marshal: %v", parseCase.label, parseEncodeErr)
		}
		var parseDecoded Value
		if parseDecodeErr := json.Unmarshal(parseEncoded, &parseDecoded); parseDecodeErr != nil {
			parseT.Fatalf("%s: unmarshal: %v", parseCase.label, parseDecodeErr)
		}
		if parseDecoded.Kind != parseCase.kind {
			parseT.Errorf("%s: decoded kind = %s, want %s", parseCase.label, parseDecoded.Kind, parseCase.kind)
		}

		switch parseCase.kind {
		case ValueBlob:
			if !bytes.Equal(parseDecoded.Blob, parseCase.input.([]byte)) {
				parseT.Errorf("%s: blob = %v, want %v", parseCase.label, parseDecoded.Blob, parseCase.input)
			}
		case ValueNull:
			if parseDecoded.Go() != nil {
				parseT.Errorf("%s: Go() = %v, want nil", parseCase.label, parseDecoded.Go())
			}
		default:
			if parseDecoded.Go() != parseCase.input {
				parseT.Errorf("%s: Go() = %v, want %v", parseCase.label, parseDecoded.Go(), parseCase.input)
			}
		}
	}
}

// TestLargeIntegersSurviveEncoding is why the encoding is typed rather than
// `any`. An int64 row id above 2^53 sent as an untyped JSON number comes back as
// a float64 and silently changes value — the classic way a JS-adjacent boundary
// corrupts identifiers.
func TestLargeIntegersSurviveEncoding(parseT *testing.T) {
	const parseBeyondFloat64Precision = int64(9007199254740993) // 2^53 + 1

	parseValue, parseErr := NewValue(parseBeyondFloat64Precision)
	if parseErr != nil {
		parseT.Fatalf("NewValue: %v", parseErr)
	}
	parseEncoded, _ := json.Marshal(parseValue)

	var parseDecoded Value
	if parseDecodeErr := json.Unmarshal(parseEncoded, &parseDecoded); parseDecodeErr != nil {
		parseT.Fatalf("unmarshal: %v", parseDecodeErr)
	}
	if parseDecoded.Int != parseBeyondFloat64Precision {
		parseT.Errorf("decoded = %d, want %d — the integer lost precision crossing the boundary",
			parseDecoded.Int, parseBeyondFloat64Precision)
	}
}

func TestBooleansBecomeIntegers(parseT *testing.T) {
	parseTrue, _ := NewValue(true)
	parseFalse, _ := NewValue(false)
	if parseTrue.Kind != ValueInt || parseTrue.Int != 1 {
		parseT.Errorf("true = %+v, want int 1", parseTrue)
	}
	if parseFalse.Kind != ValueInt || parseFalse.Int != 0 {
		parseT.Errorf("false = %+v, want int 0", parseFalse)
	}
}

func TestNewValuesReportsTheOffendingArgument(parseT *testing.T) {
	_, parseErr := NewValues([]any{1, "ok", struct{}{}})
	if parseErr == nil {
		parseT.Fatal("an unsupported argument must be rejected")
	}
	// The index matters: with five bind parameters, "one of these is wrong" is
	// not a usable diagnostic.
	if !bytes.Contains([]byte(parseErr.Error()), []byte("argument 2")) {
		parseT.Errorf("err = %v, want it to name argument 2", parseErr)
	}
}

func TestNewValuesOnAnEmptyListAllocatesNothing(parseT *testing.T) {
	parseValues, parseErr := NewValues(nil)
	if parseErr != nil {
		parseT.Fatalf("NewValues: %v", parseErr)
	}
	if parseValues != nil {
		parseT.Errorf("values = %v, want nil for an empty argument list", parseValues)
	}
}

func TestRequestAndResponseRoundTrip(parseT *testing.T) {
	parseRequest := Request{
		Op:      OpQuery,
		SQL:     "SELECT a, b FROM t WHERE a > ?",
		Args:    []Value{{Kind: ValueInt, Int: 5}},
		TxID:    "tx-3",
		MaxRows: 100,
	}
	parseEncoded, parseErr := json.Marshal(parseRequest)
	if parseErr != nil {
		parseT.Fatalf("marshal request: %v", parseErr)
	}
	var parseDecodedRequest Request
	if parseErr := json.Unmarshal(parseEncoded, &parseDecodedRequest); parseErr != nil {
		parseT.Fatalf("unmarshal request: %v", parseErr)
	}
	if parseDecodedRequest.SQL != parseRequest.SQL || parseDecodedRequest.TxID != parseRequest.TxID ||
		parseDecodedRequest.MaxRows != parseRequest.MaxRows || len(parseDecodedRequest.Args) != 1 {
		parseT.Errorf("decoded request = %+v, want %+v", parseDecodedRequest, parseRequest)
	}

	parseResponse := Response{
		Columns:   []string{"a", "b"},
		Rows:      [][]Value{{{Kind: ValueInt, Int: 6}, {Kind: ValueText, Text: "x"}}},
		Truncated: true,
	}
	parseEncodedResponse, _ := json.Marshal(parseResponse)
	var parseDecodedResponse Response
	if parseErr := json.Unmarshal(parseEncodedResponse, &parseDecodedResponse); parseErr != nil {
		parseT.Fatalf("unmarshal response: %v", parseErr)
	}
	if len(parseDecodedResponse.Rows) != 1 || parseDecodedResponse.Rows[0][1].Text != "x" || !parseDecodedResponse.Truncated {
		parseT.Errorf("decoded response = %+v, want the rows and truncation preserved", parseDecodedResponse)
	}
}

func TestUnknownOpsAreRefused(parseT *testing.T) {
	if IsKnownOp(Op("drop-everything")) {
		parseT.Error("an unknown op must be refused, not silently ignored")
	}
	if IsKnownOp(Op("")) {
		parseT.Error("an empty op must be refused")
	}
	for _, parseOp := range []Op{OpExec, OpQuery, OpBegin, OpCommit, OpRollback, OpFlush, OpClose} {
		if !IsKnownOp(parseOp) {
			parseT.Errorf("op %q must be known", parseOp)
		}
	}
}

func TestValueKindLabelsAreDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseKind := range []ValueKind{ValueNull, ValueInt, ValueFloat, ValueText, ValueBlob} {
		if parseSeen[parseKind.String()] {
			parseT.Errorf("kind label %q is not distinct", parseKind)
		}
		parseSeen[parseKind.String()] = true
	}
	if ValueKind(99).String() == "" {
		parseT.Error("an unknown kind must still render for diagnostics")
	}
}

func TestRemoteErrorNamesTheOperation(parseT *testing.T) {
	parseErr := &RemoteError{Op: OpQuery, Message: "no such column: z"}
	parseText := parseErr.Error()
	if !bytes.Contains([]byte(parseText), []byte("query")) || !bytes.Contains([]byte(parseText), []byte("no such column: z")) {
		parseT.Errorf("error text = %q, want it to name both the op and the cause", parseText)
	}
}
