package services

import (
	"encoding/json"
	"errors"
	"testing"
)

// v5 P3.3 — the extracted transport substrate.
//
// The behavior being preserved from runtime2 is the DEGRADATION policy, not the
// encoding: prefer binary, fall back to structured clone when binary fails,
// fail rather than pretend when no fallback exists, and always report which
// tier the bytes actually are.

// testPayload is a trivial payload; the substrate never inspects it.
type testPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// testCodec encodes JSON both ways, with switches to simulate encode failures.
type testCodec struct {
	failBinary bool
	failJSON   bool
	binaryHits int
	jsonHits   int
}

func (parseC *testCodec) EncodeBinary(parsePayload testPayload) ([]byte, error) {
	parseC.binaryHits++
	if parseC.failBinary {
		return nil, errors.New("binary encode unavailable for this payload")
	}
	// A stand-in binary form; the substrate only cares that it round-trips.
	return append([]byte("BIN:"), mustJSON(parsePayload)...), nil
}

func (parseC *testCodec) EncodeJSON(parsePayload testPayload) ([]byte, error) {
	parseC.jsonHits++
	if parseC.failJSON {
		return nil, errors.New("json encode failed")
	}
	return mustJSON(parsePayload), nil
}

func (parseC *testCodec) DecodeBinary(parseBytes []byte) (testPayload, error) {
	if len(parseBytes) < 4 || string(parseBytes[:4]) != "BIN:" {
		return testPayload{}, errors.New("not a binary payload")
	}
	var parseOut testPayload
	return parseOut, json.Unmarshal(parseBytes[4:], &parseOut)
}

func (parseC *testCodec) DecodeJSON(parseBytes []byte) (testPayload, error) {
	var parseOut testPayload
	return parseOut, json.Unmarshal(parseBytes, &parseOut)
}

func mustJSON(parseValue testPayload) []byte {
	parseBytes, _ := json.Marshal(parseValue)
	return parseBytes
}

// ---------------------------------------------------------------- selection

func TestSelectTier_PrefersBinary(parseT *testing.T) {
	parseTier, parseErr := SelectTier(Capabilities{HasBinary: true, HasStructuredClone: true})
	if parseErr != nil {
		parseT.Fatalf("SelectTier: %v", parseErr)
	}
	if parseTier != TierBinary {
		parseT.Errorf("tier = %q, want %q", parseTier, TierBinary)
	}
}

func TestSelectTier_FallsBackToStructuredClone(parseT *testing.T) {
	parseTier, parseErr := SelectTier(Capabilities{HasStructuredClone: true})
	if parseErr != nil {
		parseT.Fatalf("SelectTier: %v", parseErr)
	}
	if parseTier != TierStructuredClone {
		parseT.Errorf("tier = %q, want %q", parseTier, TierStructuredClone)
	}
}

func TestSelectTier_ErrorsWhenNothingSupported(parseT *testing.T) {
	if _, parseErr := SelectTier(Capabilities{}); !errors.Is(parseErr, ErrNoSupportedTier) {
		parseT.Errorf("err = %v, want ErrNoSupportedTier", parseErr)
	}
}

// TestSelectTier_SharedIsNeverImplicit pins decision D5. Shared memory needs
// cross-origin isolation, which most deployments do not have, so it must be
// asked for rather than silently selected.
func TestSelectTier_SharedIsNeverImplicit(parseT *testing.T) {
	parseTier, parseErr := SelectTier(Capabilities{HasShared: true, HasStructuredClone: true})
	if parseErr != nil {
		parseT.Fatalf("SelectTier: %v", parseErr)
	}
	if parseTier == TierShared {
		parseT.Error("shared memory must be opt-in, never selected implicitly (D5)")
	}
}

func TestSelectTierPreferring_HonorsExplicitRequest(parseT *testing.T) {
	parseTier, parseErr := SelectTierPreferring(
		Capabilities{HasShared: true, HasBinary: true, HasStructuredClone: true}, TierShared)
	if parseErr != nil {
		parseT.Fatalf("SelectTierPreferring: %v", parseErr)
	}
	if parseTier != TierShared {
		parseT.Errorf("tier = %q, want %q when explicitly requested", parseTier, TierShared)
	}
}

func TestSelectTierPreferring_FallsBackWhenUnavailable(parseT *testing.T) {
	parseTier, parseErr := SelectTierPreferring(
		Capabilities{HasBinary: true, HasStructuredClone: true}, TierShared)
	if parseErr != nil {
		parseT.Fatalf("SelectTierPreferring: %v", parseErr)
	}
	if parseTier != TierBinary {
		parseT.Errorf("tier = %q, want binary when shared is unavailable", parseTier)
	}
}

// ----------------------------------------------------------------- encoding

func TestEncode_UsesBinaryWhenAvailable(parseT *testing.T) {
	parseCodec := &testCodec{}
	parseResult, parseErr := Encode(testPayload{Name: "a", Count: 1}, parseCodec,
		Capabilities{HasBinary: true, HasStructuredClone: true})
	if parseErr != nil {
		parseT.Fatalf("Encode: %v", parseErr)
	}
	if parseResult.Tier != TierBinary {
		parseT.Errorf("tier = %q, want binary", parseResult.Tier)
	}
	if parseResult.Downgraded {
		parseT.Error("a successful binary encode must not report a downgrade")
	}
	if parseCodec.jsonHits != 0 {
		parseT.Error("the JSON encoder ran despite binary succeeding")
	}
}

// TestEncode_DowngradesAndSaysSo is the behavior worth preserving from
// runtime2: a failed binary encode falls back rather than failing, and the
// result records that it happened (R4 — degradation is never silent).
func TestEncode_DowngradesAndSaysSo(parseT *testing.T) {
	parseCodec := &testCodec{failBinary: true}
	parseResult, parseErr := Encode(testPayload{Name: "b", Count: 2}, parseCodec,
		Capabilities{HasBinary: true, HasStructuredClone: true})
	if parseErr != nil {
		parseT.Fatalf("Encode should have downgraded, got error: %v", parseErr)
	}
	if parseResult.Tier != TierStructuredClone {
		parseT.Errorf("tier = %q, want structured-clone after downgrade", parseResult.Tier)
	}
	if !parseResult.Downgraded {
		parseT.Error("a downgrade must be reported on the result")
	}
	if parseResult.DowngradeReason == "" {
		parseT.Error("a downgrade must carry its reason")
	}
}

// TestEncode_FailsRatherThanPretendWithNoFallback: if binary fails and no
// fallback exists, returning an error is correct. Reporting a downgrade that
// did not happen would be worse than failing.
func TestEncode_FailsRatherThanPretendWithNoFallback(parseT *testing.T) {
	parseCodec := &testCodec{failBinary: true}
	if _, parseErr := Encode(testPayload{}, parseCodec, Capabilities{HasBinary: true}); parseErr == nil {
		parseT.Error("expected an error when binary fails with no fallback available")
	}
}

func TestEncode_BothEncodersFailing(parseT *testing.T) {
	parseCodec := &testCodec{failBinary: true, failJSON: true}
	if _, parseErr := Encode(testPayload{}, parseCodec,
		Capabilities{HasBinary: true, HasStructuredClone: true}); parseErr == nil {
		parseT.Error("expected an error when both encoders fail")
	}
}

func TestEncode_NoSupportedTier(parseT *testing.T) {
	if _, parseErr := Encode(testPayload{}, &testCodec{}, Capabilities{}); !errors.Is(parseErr, ErrNoSupportedTier) {
		parseT.Errorf("err = %v, want ErrNoSupportedTier", parseErr)
	}
}

func TestEncode_NilCodec(parseT *testing.T) {
	if _, parseErr := Encode(testPayload{}, nil, Capabilities{HasBinary: true}); parseErr == nil {
		parseT.Error("a nil codec must be rejected rather than panicking")
	}
}

// ----------------------------------------------------------------- decoding

// TestDecode_TierIsCarriedNotSniffed pins why the tier travels with the bytes.
// Sniffing would make a downgraded payload indistinguishable from a corrupt
// one, and the receiver would decode the wrong way with no error at all.
func TestDecode_TierIsCarriedNotSniffed(parseT *testing.T) {
	parseCodec := &testCodec{}
	parsePayload := testPayload{Name: "c", Count: 3}

	parseBinary, _ := parseCodec.EncodeBinary(parsePayload)
	if _, parseErr := Decode(parseBinary, TierStructuredClone, parseCodec); parseErr == nil {
		parseT.Error("decoding binary bytes as structured-clone must fail loudly")
	}
	parseDecoded, parseErr := Decode(parseBinary, TierBinary, parseCodec)
	if parseErr != nil {
		parseT.Fatalf("Decode(binary): %v", parseErr)
	}
	if parseDecoded != parsePayload {
		parseT.Errorf("decoded = %+v, want %+v", parseDecoded, parsePayload)
	}
}

func TestDecode_UnknownTier(parseT *testing.T) {
	if _, parseErr := Decode([]byte("{}"), Tier("made-up"), &testCodec{}); parseErr == nil {
		parseT.Error("an unknown tier must be rejected")
	}
}

// TestDecode_SharedUsesJSONEncoding: the shared tier describes how bytes
// TRAVELLED, not how they were serialized.
func TestDecode_SharedUsesJSONEncoding(parseT *testing.T) {
	parseCodec := &testCodec{}
	parsePayload := testPayload{Name: "d", Count: 4}
	parseJSON, _ := parseCodec.EncodeJSON(parsePayload)

	parseDecoded, parseErr := Decode(parseJSON, TierShared, parseCodec)
	if parseErr != nil {
		parseT.Fatalf("Decode(shared): %v", parseErr)
	}
	if parseDecoded != parsePayload {
		parseT.Errorf("decoded = %+v, want %+v", parseDecoded, parsePayload)
	}
}

// --------------------------------------------------------------- round trip

func TestRoundTrip_AcrossEveryReachableTier(parseT *testing.T) {
	parsePayload := testPayload{Name: "round", Count: 42}

	for _, parseCase := range []struct {
		label        string
		capabilities Capabilities
		wantTier     Tier
	}{
		{"binary", Capabilities{HasBinary: true, HasStructuredClone: true}, TierBinary},
		{"structured-clone only", Capabilities{HasStructuredClone: true}, TierStructuredClone},
	} {
		parseDecoded, parseTier, parseErr := RoundTrip(parsePayload, &testCodec{}, parseCase.capabilities)
		if parseErr != nil {
			parseT.Errorf("%s: RoundTrip: %v", parseCase.label, parseErr)
			continue
		}
		if parseTier != parseCase.wantTier {
			parseT.Errorf("%s: tier = %q, want %q", parseCase.label, parseTier, parseCase.wantTier)
		}
		if parseDecoded != parsePayload {
			parseT.Errorf("%s: decoded = %+v, want %+v", parseCase.label, parseDecoded, parsePayload)
		}
	}
}

// TestRoundTrip_SurvivesADowngrade: the decode has to follow the tier that was
// actually used, not the one that was preferred.
func TestRoundTrip_SurvivesADowngrade(parseT *testing.T) {
	parsePayload := testPayload{Name: "downgraded", Count: 7}
	parseDecoded, parseTier, parseErr := RoundTrip(parsePayload, &testCodec{failBinary: true},
		Capabilities{HasBinary: true, HasStructuredClone: true})
	if parseErr != nil {
		parseT.Fatalf("RoundTrip: %v", parseErr)
	}
	if parseTier != TierStructuredClone {
		parseT.Errorf("tier = %q, want structured-clone", parseTier)
	}
	if parseDecoded != parsePayload {
		parseT.Errorf("decoded = %+v, want %+v", parseDecoded, parsePayload)
	}
}
