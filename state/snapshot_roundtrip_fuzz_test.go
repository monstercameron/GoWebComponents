//go:build !js || !wasm

package state

import (
	"bytes"
	"testing"
)

// FuzzSnapshotJSONRoundTrip fuzzes the snapshot wire codec for two invariants:
//
//  1. Robustness: UnmarshalSnapshotJSON must never panic on arbitrary bytes
//     (it may legitimately return an error).
//  2. Fixpoint stability: once a value has survived one Marshal∘Unmarshal cycle,
//     every subsequent cycle must produce byte-identical output. This guards the
//     normalizeSnapshot float64->int folding (and key ordering) from drifting a
//     persisted snapshot on each save/load, which would defeat dedup and make
//     stored state churn.
func FuzzSnapshotJSONRoundTrip(parseF *testing.F) {
	for _, parseSeed := range []string{
		``,
		`{}`,
		`{"protocol":"gwc.state.snapshot","version":1,"state":{"a":1}}`,
		`{"protocol":"gwc.state.snapshot","version":1,"state":{"n":3.5,"b":true,"s":"x","arr":[1,2,3],"obj":{"k":9007199254740993}}}`,
		`{"a":1,"b":"two","c":[true,null,1e20]}`,
		`{"big":12345678901234567890,"whole":5.0,"neg":-9007199254740992}`,
		`not json`,
		`{"state":null}`,
	} {
		parseF.Add([]byte(parseSeed))
	}

	parseF.Fuzz(func(parseT *testing.T, parseData []byte) {
		parseSnap, parseErr := UnmarshalSnapshotJSON(parseData)
		if parseErr != nil {
			return // arbitrary bytes may not decode — that's fine, just no panic
		}

		// First normalization cycle: marshal the decoded snapshot.
		parseB1, parseErr := MarshalSnapshotJSON(parseSnap)
		if parseErr != nil {
			parseT.Fatalf("re-marshal of decoded snapshot failed: %v (input %q)", parseErr, parseData)
		}
		parseSnap2, parseErr := UnmarshalSnapshotJSON(parseB1)
		if parseErr != nil {
			parseT.Fatalf("decode of our own output failed: %v (b1 %q)", parseErr, parseB1)
		}
		parseB2, parseErr := MarshalSnapshotJSON(parseSnap2)
		if parseErr != nil {
			parseT.Fatalf("second re-marshal failed: %v", parseErr)
		}

		// b1 and b2 are both produced from already-normalized snapshots, so they
		// must be byte-identical — the codec has reached a fixpoint.
		if !bytes.Equal(parseB1, parseB2) {
			parseT.Fatalf("round-trip not a fixpoint:\n b1=%q\n b2=%q\n input=%q", parseB1, parseB2, parseData)
		}
	})
}
