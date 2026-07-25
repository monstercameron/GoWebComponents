// Package services is the payload-agnostic transport substrate for v5's
// off-thread work (plan item P3.3).
//
// runtime2 already solved transport selection and degradation properly: pick
// the best tier the browser supports, fall back cleanly when an encode fails,
// and never silently ship a worse payload than the caller thinks. That logic is
// correct and worth keeping — but in runtime2 it is welded to DOM-patch types
// (SnapshotEnvelope, PatchStreamRaw), so a domain service moving rows or
// projections cannot use any of it.
//
// This extracts the decision-making and leaves encoding to the payload, which
// is the only part that was ever DOM-specific. runtime2's own types satisfy the
// interface unchanged; see transport_runtime2_conformance_test.go.
//
// Deliberately NOT extracted here: the scheduler, recovery coordinator, and
// idempotency ledger. The plan's P3.3 contingency puts a decision point after
// the transports precisely so their extraction can be judged on measurements
// from this one rather than committed to up front.
package services

import (
	"errors"
	"fmt"
)

// Tier names one wire encoding, most preferred first.
type Tier string

const (
	// TierBinary is a compact byte encoding. Preferred: smallest payload and no
	// JSON parse on the receiving side.
	TierBinary Tier = "binary"
	// TierStructuredClone is a JSON encoding carried by postMessage's own
	// structured clone. Universally available and the fallback for everything.
	TierStructuredClone Tier = "structured-clone"
	// TierShared is a SharedArrayBuffer page. Fastest where cross-origin
	// isolation is available, which most deployments do not have — which is why
	// v5 made it opt-in and never required (decision D5).
	TierShared Tier = "shared"
)

// Capabilities reports what the current environment supports.
type Capabilities struct {
	HasBinary          bool
	HasStructuredClone bool
	HasShared          bool
}

// Codec encodes and decodes one payload type for each tier.
//
// Generic over the payload so the substrate never learns what it is carrying.
// A codec may report an error from any encode; that is the trigger for the
// documented downgrade rather than a fatal condition.
type Codec[T any] interface {
	EncodeBinary(parsePayload T) ([]byte, error)
	EncodeJSON(parsePayload T) ([]byte, error)
	DecodeBinary(parseBytes []byte) (T, error)
	DecodeJSON(parseBytes []byte) (T, error)
}

// ErrNoSupportedTier is returned when the environment supports no tier at all.
var ErrNoSupportedTier = errors.New("services: no supported transport tier is available")

// SelectTier picks the best available tier.
//
// Shared memory is NOT preferred automatically even when available. It requires
// cross-origin isolation, its ordering guarantees differ from message passing,
// and D5 made it opt-in — so a caller asks for it explicitly via
// SelectTierPreferring rather than silently receiving it.
func SelectTier(parseCapabilities Capabilities) (Tier, error) {
	if parseCapabilities.HasBinary {
		return TierBinary, nil
	}
	if parseCapabilities.HasStructuredClone {
		return TierStructuredClone, nil
	}
	return "", ErrNoSupportedTier
}

// SelectTierPreferring honors an explicit tier request, falling back through
// the normal order when it is unavailable.
func SelectTierPreferring(parseCapabilities Capabilities, parsePreferred Tier) (Tier, error) {
	switch parsePreferred {
	case TierShared:
		if parseCapabilities.HasShared {
			return TierShared, nil
		}
	case TierBinary:
		if parseCapabilities.HasBinary {
			return TierBinary, nil
		}
	case TierStructuredClone:
		if parseCapabilities.HasStructuredClone {
			return TierStructuredClone, nil
		}
	}
	return SelectTier(parseCapabilities)
}

// EncodeResult carries an encoded payload and how it was actually encoded.
//
// Tier is what the payload IS, not what was asked for. A caller that logs the
// requested tier rather than this one will misreport every downgrade.
type EncodeResult struct {
	Tier  Tier
	Bytes []byte
	// Downgraded records that the preferred tier failed to encode and a lower
	// one was used. R4: a degradation is reported, never silent.
	Downgraded bool
	// DowngradeReason explains why, for diagnostics.
	DowngradeReason string
}

// Encode encodes a payload at the best available tier, downgrading on failure.
//
// Mirrors runtime2's BuildSnapshotTransportPayloadWithFallback: prefer binary,
// and if binary encoding fails, fall back to structured clone WHEN AVAILABLE.
// If it is not available the binary error is returned unchanged, because
// reporting a downgrade that did not happen would be worse than failing.
func Encode[T any](parsePayload T, parseCodec Codec[T], parseCapabilities Capabilities) (EncodeResult, error) {
	if parseCodec == nil {
		return EncodeResult{}, errors.New("services: codec is nil")
	}

	if parseCapabilities.HasBinary {
		parseBytes, parseErr := parseCodec.EncodeBinary(parsePayload)
		if parseErr == nil {
			return EncodeResult{Tier: TierBinary, Bytes: parseBytes}, nil
		}
		if !parseCapabilities.HasStructuredClone {
			return EncodeResult{}, fmt.Errorf("services: binary encode failed with no fallback available: %w", parseErr)
		}
		parseJSONBytes, parseJSONErr := parseCodec.EncodeJSON(parsePayload)
		if parseJSONErr != nil {
			return EncodeResult{}, fmt.Errorf("services: binary encode failed (%v) and structured-clone fallback also failed: %w", parseErr, parseJSONErr)
		}
		return EncodeResult{
			Tier:            TierStructuredClone,
			Bytes:           parseJSONBytes,
			Downgraded:      true,
			DowngradeReason: parseErr.Error(),
		}, nil
	}

	if parseCapabilities.HasStructuredClone {
		parseJSONBytes, parseJSONErr := parseCodec.EncodeJSON(parsePayload)
		if parseJSONErr != nil {
			return EncodeResult{}, fmt.Errorf("services: structured-clone encode failed: %w", parseJSONErr)
		}
		return EncodeResult{Tier: TierStructuredClone, Bytes: parseJSONBytes}, nil
	}

	return EncodeResult{}, ErrNoSupportedTier
}

// Decode decodes a payload that was encoded at a known tier.
//
// The tier is carried explicitly rather than sniffed from the bytes. Guessing
// would make a downgraded payload indistinguishable from a corrupt one, and the
// receiver would decode the wrong way with no error.
func Decode[T any](parseBytes []byte, parseTier Tier, parseCodec Codec[T]) (T, error) {
	var parseZero T
	if parseCodec == nil {
		return parseZero, errors.New("services: codec is nil")
	}
	switch parseTier {
	case TierBinary:
		return parseCodec.DecodeBinary(parseBytes)
	case TierStructuredClone, TierShared:
		// Shared pages carry the structured-clone encoding; the tier describes
		// how the bytes travelled, not how they were serialized.
		return parseCodec.DecodeJSON(parseBytes)
	default:
		return parseZero, fmt.Errorf("services: unknown transport tier %q", parseTier)
	}
}

// RoundTrip encodes then decodes, returning the tier actually used.
//
// Exists for conformance testing: a codec whose decode cannot read its own
// encode is broken in a way neither operation reveals alone.
func RoundTrip[T any](parsePayload T, parseCodec Codec[T], parseCapabilities Capabilities) (T, Tier, error) {
	var parseZero T
	parseResult, parseErr := Encode(parsePayload, parseCodec, parseCapabilities)
	if parseErr != nil {
		return parseZero, "", parseErr
	}
	parseDecoded, parseDecodeErr := Decode(parseResult.Bytes, parseResult.Tier, parseCodec)
	if parseDecodeErr != nil {
		return parseZero, parseResult.Tier, parseDecodeErr
	}
	return parseDecoded, parseResult.Tier, nil
}
