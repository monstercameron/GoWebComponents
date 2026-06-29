package runtime2

import "testing"

// buildPatchTransportFuzzSeedPatchStream builds one minimal valid patch stream used for fuzz seeding.
func buildPatchTransportFuzzSeedPatchStream() PatchStreamRaw {
	return PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-1",
			Epoch:           1,
			InputVersion:    1,
			PatchVersion:    1,
		},
	}
}

// buildPatchTransportFuzzSeedStructuredPayload builds one valid structured-clone patch envelope payload used for fuzz seeding.
func buildPatchTransportFuzzSeedStructuredPayload() []byte {
	parsePatchStream := buildPatchTransportFuzzSeedPatchStream()
	parsePatchPayload, parsePatchPayloadErr := BuildStructuredClonePatchPayloadJSON(parsePatchStream)
	if parsePatchPayloadErr != nil {
		return []byte(`{"region_instance_id":"region-1","epoch":1,"patch_version":1,"patch_payload":"e30="}`)
	}
	parseEnvelopePayload, parseEnvelopePayloadErr := BuildStructuredClonePatchEnvelopeJSON(StructuredClonePatchEnvelope{
		RegionInstanceID: "region-1",
		Epoch:            1,
		PatchVersion:     1,
		PatchPayload:     parsePatchPayload,
	})
	if parseEnvelopePayloadErr != nil {
		return []byte(`{"region_instance_id":"region-1","epoch":1,"patch_version":1,"patch_payload":"e30="}`)
	}
	return parseEnvelopePayload
}

// buildPatchTransportFuzzSeedBinaryPayload builds one valid binary patch payload used for fuzz seeding.
func buildPatchTransportFuzzSeedBinaryPayload() []byte {
	parsePayload, parsePayloadErr := BuildBinaryPatchPayload(buildPatchTransportFuzzSeedPatchStream())
	if parsePayloadErr != nil {
		return []byte{'P', 'T', 'C', 'H', 2, 0, 0, 0, '{', '}'}
	}
	return parsePayload
}

// FuzzParseStructuredClonePatchEnvelopeJSON fuzzes structured-clone patch-envelope decode and validation stability.
func FuzzParseStructuredClonePatchEnvelopeJSON(parseF *testing.F) {
	parseF.Add([]byte(`{"region_instance_id":"region-1","epoch":1,"patch_version":1,"patch_payload":"e30="}`))
	parseF.Add([]byte(`{"region_instance_id":"region-1","epoch":0,"patch_version":1,"patch_payload":"e30="}`))
	parseF.Add([]byte(`{"region_instance_id":"region-1","epoch":1,"patch_version":1}`))
	parseF.Add([]byte(`{`))
	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		parseEnvelope, parseEnvelopeErr := ParseStructuredClonePatchEnvelopeJSON(parsePayload)
		if parseEnvelopeErr != nil {
			return
		}
		if parseValidateErr := ValidateStructuredClonePatchEnvelope(parseEnvelope); parseValidateErr != nil {
			parseT.Fatalf("ValidateStructuredClonePatchEnvelope(fuzz decoded envelope) returned error: %v", parseValidateErr)
		}
		parseRoundTripPayload, parseRoundTripPayloadErr := BuildStructuredClonePatchEnvelopeJSON(parseEnvelope)
		if parseRoundTripPayloadErr != nil {
			parseT.Fatalf("BuildStructuredClonePatchEnvelopeJSON(fuzz decoded envelope) returned error: %v", parseRoundTripPayloadErr)
		}
		if _, parseRoundTripErr := ParseStructuredClonePatchEnvelopeJSON(parseRoundTripPayload); parseRoundTripErr != nil {
			parseT.Fatalf("ParseStructuredClonePatchEnvelopeJSON(round-trip payload) returned error: %v", parseRoundTripErr)
		}
	})
}

// FuzzParseBinaryPatchPayload fuzzes binary patch payload decode and header-parse stability.
func FuzzParseBinaryPatchPayload(parseF *testing.F) {
	parseF.Add(buildPatchTransportFuzzSeedBinaryPayload())
	parseF.Add([]byte{'P', 'T', 'C', 'H', 1, 0, 0, 0, '{'})
	parseF.Add([]byte{'P', 'T', 'X', 'H'})
	parseF.Add([]byte{})
	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		parsePatchStream, parsePatchStreamErr := ParseBinaryPatchPayload(parsePayload)
		if parsePatchStreamErr != nil {
			return
		}
		if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
			parseT.Fatalf("ParsePatchStreamHeader(fuzz decoded binary payload) returned error: %v", parseHeaderErr)
		}
		parseRoundTripPayload, parseRoundTripPayloadErr := BuildBinaryPatchPayload(parsePatchStream)
		if parseRoundTripPayloadErr != nil {
			parseT.Fatalf("BuildBinaryPatchPayload(fuzz decoded binary payload) returned error: %v", parseRoundTripPayloadErr)
		}
		if _, parseRoundTripErr := ParseBinaryPatchPayload(parseRoundTripPayload); parseRoundTripErr != nil {
			parseT.Fatalf("ParseBinaryPatchPayload(round-trip payload) returned error: %v", parseRoundTripErr)
		}
	})
}

// FuzzParseSharedPatchPayloadFromPage fuzzes shared-buffer patch-page publish and parse paths for panic safety and stable validation.
func FuzzParseSharedPatchPayloadFromPage(parseF *testing.F) {
	parseF.Add(buildPatchTransportFuzzSeedStructuredPayload())
	parseF.Add([]byte(`{"invalid"`))
	parseF.Add([]byte{})
	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		if len(parsePayload) > 4096 {
			return
		}
		parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(sharedSnapshotPageHeaderSize + 4096)
		if parseSharedPatchPageErr != nil {
			parseT.Fatalf("BuildSharedPatchPage(fuzz) returned error: %v", parseSharedPatchPageErr)
		}
		if _, parsePublishErr := parseSharedPatchPage.HandleSharedPatchPublishPayload(parsePayload); parsePublishErr != nil {
			return
		}
		parseEnvelope, parseEnvelopeErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage)
		if parseEnvelopeErr != nil {
			return
		}
		if parseValidateErr := ValidateStructuredClonePatchEnvelope(parseEnvelope); parseValidateErr != nil {
			parseT.Fatalf("ValidateStructuredClonePatchEnvelope(fuzz shared decode) returned error: %v", parseValidateErr)
		}
		parseRoundTripPayload, parseRoundTripPayloadErr := BuildStructuredClonePatchEnvelopeJSON(parseEnvelope)
		if parseRoundTripPayloadErr != nil {
			parseT.Fatalf("BuildStructuredClonePatchEnvelopeJSON(fuzz shared decode) returned error: %v", parseRoundTripPayloadErr)
		}
		if _, parsePublishRoundTripErr := parseSharedPatchPage.HandleSharedPatchPublishPayload(parseRoundTripPayload); parsePublishRoundTripErr != nil {
			parseT.Fatalf("HandleSharedPatchPublishPayload(round-trip payload) returned error: %v", parsePublishRoundTripErr)
		}
		if _, parseRoundTripErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage); parseRoundTripErr != nil {
			parseT.Fatalf("ParseSharedPatchPayloadFromPage(round-trip payload) returned error: %v", parseRoundTripErr)
		}
	})
}
