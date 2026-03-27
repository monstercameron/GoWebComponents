package runtime2

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
)

const binaryPatchTransportHeaderLength = 8

var binaryPatchTransportMagic = [4]byte{'P', 'T', 'C', 'H'}

const (
	parseBinaryPatchHeaderOnlyTokenStart         = `{"header":{"ProtocolVersion":`
	parseBinaryPatchHeaderOnlyTokenRegionID      = `,"RegionID":`
	parseBinaryPatchHeaderOnlyTokenEpoch         = `,"Epoch":`
	parseBinaryPatchHeaderOnlyTokenInputVersion  = `,"InputVersion":`
	parseBinaryPatchHeaderOnlyTokenPatchVersion  = `,"PatchVersion":`
	parseBinaryPatchHeaderOnlyTokenPatchIdentity = `},"patch_identity":`
	parseBinaryPatchHeaderOnlyTokenClose         = `}`
)

// BuildBinaryPatchPayload encodes one patch stream payload using runtime2 binary framing.
func BuildBinaryPatchPayload(parsePatchStream PatchStreamRaw) ([]byte, error) {
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return nil, parseHeaderErr
	}
	if len(parsePatchStream.GetStringTable) == 0 && len(parsePatchStream.GetOps) == 0 {
		buildPayload := make([]byte, binaryPatchTransportHeaderLength, binaryPatchTransportHeaderLength+getBinaryPatchHeaderOnlyJSONLength(parsePatchStream))
		buildPayload = appendBinaryPatchHeaderOnlyJSON(buildPayload, parsePatchStream)
		copy(buildPayload[0:4], binaryPatchTransportMagic[:])
		binary.LittleEndian.PutUint32(buildPayload[4:8], uint32(len(buildPayload)-binaryPatchTransportHeaderLength))
		return buildPayload, nil
	}
	parseBody, parseBodyErr := json.Marshal(parsePatchStream)
	if parseBodyErr != nil {
		return nil, fmt.Errorf("runtime2: encode binary patch payload body: %w", parseBodyErr)
	}
	buildPayload := make([]byte, binaryPatchTransportHeaderLength, binaryPatchTransportHeaderLength+len(parseBody))
	buildPayload = append(buildPayload, parseBody...)
	copy(buildPayload[0:4], binaryPatchTransportMagic[:])
	binary.LittleEndian.PutUint32(buildPayload[4:8], uint32(len(parseBody)))
	return buildPayload, nil
}

// ParseBinaryPatchPayload decodes one runtime2 binary patch payload frame into a validated patch stream.
func ParseBinaryPatchPayload(parsePayload []byte) (PatchStreamRaw, error) {
	if len(parsePayload) < binaryPatchTransportHeaderLength {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload is truncated")
	}
	if parsePayload[0] != binaryPatchTransportMagic[0] ||
		parsePayload[1] != binaryPatchTransportMagic[1] ||
		parsePayload[2] != binaryPatchTransportMagic[2] ||
		parsePayload[3] != binaryPatchTransportMagic[3] {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload magic is invalid")
	}
	parseBodyLength := binary.LittleEndian.Uint32(parsePayload[4:8])
	if parseBodyLength == 0 {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload body is required")
	}
	parseBody := parsePayload[binaryPatchTransportHeaderLength:]
	if int(parseBodyLength) != len(parseBody) {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch payload length mismatch")
	}
	if parsePatchStreamFast, hasParsePatchStreamFast := parseBinaryPatchHeaderOnlyJSON(parseBody); hasParsePatchStreamFast {
		if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStreamFast.GetHeader, parsePatchStreamFast.GetHeader.RegionID); parseHeaderErr != nil {
			return PatchStreamRaw{}, parseHeaderErr
		}
		return parsePatchStreamFast, nil
	}
	var parsePatchStream PatchStreamRaw
	if parseDecodeErr := json.Unmarshal(parseBody, &parsePatchStream); parseDecodeErr != nil {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: decode binary patch payload body: %w", parseDecodeErr)
	}
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return parsePatchStream, nil
}

// getBinaryPatchHeaderOnlyJSONLength returns an upper-bound capacity for one header-only patch JSON payload.
func getBinaryPatchHeaderOnlyJSONLength(parsePatchStream PatchStreamRaw) int {
	return len(`{"header":{"ProtocolVersion":`) +
		2 + len(parsePatchStream.GetHeader.ProtocolVersion) +
		len(`,"RegionID":`) +
		2 + len(parsePatchStream.GetHeader.RegionID) +
		len(`,"Epoch":`) + 20 +
		len(`,"InputVersion":`) + 20 +
		len(`,"PatchVersion":`) + 20 +
		len(`},"patch_identity":`) +
		2 + len(parsePatchStream.GetPatchIdentity) +
		1
}

// appendBinaryPatchHeaderOnlyJSON appends one header-only patch payload JSON body.
func appendBinaryPatchHeaderOnlyJSON(parseDst []byte, parsePatchStream PatchStreamRaw) []byte {
	parseHeader := parsePatchStream.GetHeader
	parseDst = append(parseDst, `{"header":{"ProtocolVersion":`...)
	parseDst = strconv.AppendQuote(parseDst, parseHeader.ProtocolVersion)
	parseDst = append(parseDst, `,"RegionID":`...)
	parseDst = strconv.AppendQuote(parseDst, parseHeader.RegionID)
	parseDst = append(parseDst, `,"Epoch":`...)
	parseDst = strconv.AppendUint(parseDst, parseHeader.Epoch, 10)
	parseDst = append(parseDst, `,"InputVersion":`...)
	parseDst = strconv.AppendUint(parseDst, parseHeader.InputVersion, 10)
	parseDst = append(parseDst, `,"PatchVersion":`...)
	parseDst = strconv.AppendUint(parseDst, parseHeader.PatchVersion, 10)
	parseDst = append(parseDst, `},"patch_identity":`...)
	parseDst = strconv.AppendQuote(parseDst, parsePatchStream.GetPatchIdentity)
	parseDst = append(parseDst, '}')
	return parseDst
}

// parseBinaryPatchHeaderOnlyJSON decodes one header-only patch JSON body produced by appendBinaryPatchHeaderOnlyJSON.
func parseBinaryPatchHeaderOnlyJSON(parseBody []byte) (PatchStreamRaw, bool) {
	parseOffset, hasParseToken := hasBinaryPatchLiteralAt(parseBody, 0, parseBinaryPatchHeaderOnlyTokenStart)
	if !hasParseToken {
		return PatchStreamRaw{}, false
	}
	parseProtocolVersion, parseOffset, hasParseQuotedText := parseBinaryPatchQuotedText(parseBody, parseOffset)
	if !hasParseQuotedText {
		return PatchStreamRaw{}, false
	}
	parseOffset, hasParseToken = hasBinaryPatchLiteralAt(parseBody, parseOffset, parseBinaryPatchHeaderOnlyTokenRegionID)
	if !hasParseToken {
		return PatchStreamRaw{}, false
	}
	parseRegionID, parseOffset, hasParseQuotedText := parseBinaryPatchQuotedText(parseBody, parseOffset)
	if !hasParseQuotedText {
		return PatchStreamRaw{}, false
	}
	parseOffset, hasParseToken = hasBinaryPatchLiteralAt(parseBody, parseOffset, parseBinaryPatchHeaderOnlyTokenEpoch)
	if !hasParseToken {
		return PatchStreamRaw{}, false
	}
	parseEpoch, parseOffset, hasParseUint := parseBinaryPatchUint(parseBody, parseOffset)
	if !hasParseUint {
		return PatchStreamRaw{}, false
	}
	parseOffset, hasParseToken = hasBinaryPatchLiteralAt(parseBody, parseOffset, parseBinaryPatchHeaderOnlyTokenInputVersion)
	if !hasParseToken {
		return PatchStreamRaw{}, false
	}
	parseInputVersion, parseOffset, hasParseUint := parseBinaryPatchUint(parseBody, parseOffset)
	if !hasParseUint {
		return PatchStreamRaw{}, false
	}
	parseOffset, hasParseToken = hasBinaryPatchLiteralAt(parseBody, parseOffset, parseBinaryPatchHeaderOnlyTokenPatchVersion)
	if !hasParseToken {
		return PatchStreamRaw{}, false
	}
	parsePatchVersion, parseOffset, hasParseUint := parseBinaryPatchUint(parseBody, parseOffset)
	if !hasParseUint {
		return PatchStreamRaw{}, false
	}
	parseOffset, hasParseToken = hasBinaryPatchLiteralAt(parseBody, parseOffset, parseBinaryPatchHeaderOnlyTokenPatchIdentity)
	if !hasParseToken {
		return PatchStreamRaw{}, false
	}
	parsePatchIdentity, parseOffset, hasParseQuotedText := parseBinaryPatchQuotedText(parseBody, parseOffset)
	if !hasParseQuotedText {
		return PatchStreamRaw{}, false
	}
	parseOffset, hasParseToken = hasBinaryPatchLiteralAt(parseBody, parseOffset, parseBinaryPatchHeaderOnlyTokenClose)
	if !hasParseToken || parseOffset != len(parseBody) {
		return PatchStreamRaw{}, false
	}
	return PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: parseProtocolVersion,
			RegionID:        parseRegionID,
			Epoch:           parseEpoch,
			InputVersion:    parseInputVersion,
			PatchVersion:    parsePatchVersion,
		},
		GetPatchIdentity: parsePatchIdentity,
	}, true
}

// hasBinaryPatchLiteralAt reports whether one literal token appears at the given byte offset.
func hasBinaryPatchLiteralAt(parseBody []byte, parseOffset int, parseLiteral string) (int, bool) {
	if parseOffset < 0 || len(parseBody)-parseOffset < len(parseLiteral) {
		return 0, false
	}
	for parseIndex := 0; parseIndex < len(parseLiteral); parseIndex++ {
		if parseBody[parseOffset+parseIndex] != parseLiteral[parseIndex] {
			return 0, false
		}
	}
	return parseOffset + len(parseLiteral), true
}

// parseBinaryPatchQuotedText parses one JSON quoted string value at the given byte offset.
func parseBinaryPatchQuotedText(parseBody []byte, parseOffset int) (string, int, bool) {
	if parseOffset < 0 || parseOffset >= len(parseBody) || parseBody[parseOffset] != '"' {
		return "", 0, false
	}
	parseStart := parseOffset + 1
	hasParseEscape := false
	for parseCursor := parseStart; parseCursor < len(parseBody); parseCursor++ {
		parseByte := parseBody[parseCursor]
		if parseByte < 0x20 {
			return "", 0, false
		}
		if parseByte == '\\' {
			hasParseEscape = true
			continue
		}
		if parseByte != '"' {
			continue
		}
		parseBackslashCount := 0
		for parseBack := parseCursor - 1; parseBack >= parseStart && parseBody[parseBack] == '\\'; parseBack-- {
			parseBackslashCount++
		}
		if parseBackslashCount%2 == 1 {
			continue
		}
		if !hasParseEscape {
			return string(parseBody[parseStart:parseCursor]), parseCursor + 1, true
		}
		parseDecoded, parseDecodeErr := strconv.Unquote(string(parseBody[parseOffset : parseCursor+1]))
		if parseDecodeErr != nil {
			return "", 0, false
		}
		return parseDecoded, parseCursor + 1, true
	}
	return "", 0, false
}

// parseBinaryPatchUint parses one JSON uint64 value at the given byte offset.
func parseBinaryPatchUint(parseBody []byte, parseOffset int) (uint64, int, bool) {
	if parseOffset < 0 || parseOffset >= len(parseBody) || parseBody[parseOffset] < '0' || parseBody[parseOffset] > '9' {
		return 0, 0, false
	}
	var parseValue uint64
	for parseOffset < len(parseBody) {
		parseByte := parseBody[parseOffset]
		if parseByte < '0' || parseByte > '9' {
			break
		}
		parseDigit := uint64(parseByte - '0')
		if parseValue > (^uint64(0)-parseDigit)/10 {
			return 0, 0, false
		}
		parseValue = parseValue*10 + parseDigit
		parseOffset++
	}
	return parseValue, parseOffset, true
}
