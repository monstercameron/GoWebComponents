package runtime2

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const binaryPatchTransportHeaderLength = 8

var binaryPatchTransportMagic = [4]byte{'P', 'T', 'C', 'H'}

// BuildBinaryPatchPayload encodes one patch stream payload using runtime2 binary framing.
func BuildBinaryPatchPayload(parsePatchStream PatchStreamRaw) ([]byte, error) {
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return nil, parseHeaderErr
	}
	parseBody, parseBodyErr := json.Marshal(parsePatchStream)
	if parseBodyErr != nil {
		return nil, fmt.Errorf("runtime2: encode binary patch payload body: %w", parseBodyErr)
	}
	buildPayload := make([]byte, binaryPatchTransportHeaderLength+len(parseBody))
	copy(buildPayload[0:4], binaryPatchTransportMagic[:])
	binary.LittleEndian.PutUint32(buildPayload[4:8], uint32(len(parseBody)))
	copy(buildPayload[binaryPatchTransportHeaderLength:], parseBody)
	return buildPayload, nil
}

// ParseBinaryPatchPayload decodes one runtime2 binary patch payload frame into a validated patch stream.
func ParseBinaryPatchPayload(parsePayload []byte) (PatchStreamRaw, error) {
	if len(parsePayload) < binaryPatchTransportHeaderLength {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload is truncated")
	}
	if string(parsePayload[0:4]) != string(binaryPatchTransportMagic[:]) {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload magic is invalid")
	}
	parseBodyLength := binary.LittleEndian.Uint32(parsePayload[4:8])
	if parseBodyLength == 0 {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload body is required")
	}
	if int(parseBodyLength) != len(parsePayload)-binaryPatchTransportHeaderLength {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload length mismatch")
	}
	var parsePatchStream PatchStreamRaw
	if parseDecodeErr := json.Unmarshal(parsePayload[binaryPatchTransportHeaderLength:], &parsePatchStream); parseDecodeErr != nil {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: decode binary patch payload body: %w", parseDecodeErr)
	}
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return parsePatchStream, nil
}
