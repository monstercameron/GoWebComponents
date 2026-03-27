package runtime2

import (
	"encoding/json"
	"fmt"
	"hash"
)

type buildJSONHashWriter struct {
	getHasher        hash.Hash
	getPendingByte   byte
	hasPendingByte   bool
	getPendingBuffer [1]byte
	getWriteBuffer   [512]byte
}

// Write buffers one trailing byte so JSON encoder newlines can be conditionally suppressed during Flush.
func (parseWriter *buildJSONHashWriter) Write(parseBytes []byte) (int, error) {
	if parseWriter == nil {
		return 0, fmt.Errorf("runtime2: json hash writer is nil")
	}
	if len(parseBytes) == 0 {
		return 0, nil
	}
	if parseWriter.hasPendingByte {
		if len(parseBytes) > 1 && len(parseBytes) <= len(parseWriter.getWriteBuffer) {
			parseWriter.getWriteBuffer[0] = parseWriter.getPendingByte
			copy(parseWriter.getWriteBuffer[1:], parseBytes[:len(parseBytes)-1])
			if _, parseErr := parseWriter.getHasher.Write(parseWriter.getWriteBuffer[:len(parseBytes)]); parseErr != nil {
				return 0, parseErr
			}
			parseWriter.hasPendingByte = false
			parseWriter.getPendingByte = parseBytes[len(parseBytes)-1]
			parseWriter.hasPendingByte = true
			return len(parseBytes), nil
		}
		parseWriter.getPendingBuffer[0] = parseWriter.getPendingByte
		if _, parseErr := parseWriter.getHasher.Write(parseWriter.getPendingBuffer[:]); parseErr != nil {
			return 0, parseErr
		}
		parseWriter.hasPendingByte = false
	}
	if len(parseBytes) > 1 {
		if _, parseErr := parseWriter.getHasher.Write(parseBytes[:len(parseBytes)-1]); parseErr != nil {
			return 0, parseErr
		}
	}
	parseWriter.getPendingByte = parseBytes[len(parseBytes)-1]
	parseWriter.hasPendingByte = true
	return len(parseBytes), nil
}

// Flush commits any pending trailing byte except a final JSON encoder newline.
func (parseWriter *buildJSONHashWriter) Flush() error {
	if parseWriter == nil || !parseWriter.hasPendingByte {
		return nil
	}
	parsePendingByte := parseWriter.getPendingByte
	parseWriter.hasPendingByte = false
	if parsePendingByte == '\n' {
		return nil
	}
	parseWriter.getPendingBuffer[0] = parsePendingByte
	_, parseErr := parseWriter.getHasher.Write(parseWriter.getPendingBuffer[:])
	return parseErr
}

// buildJSONHashDigest encodes one payload into the hasher without allocating a full JSON byte slice.
func buildJSONHashDigest(parseHasher hash.Hash, parseValue any) error {
	if parseHasher == nil {
		return fmt.Errorf("runtime2: json hash digest hasher is nil")
	}
	parseWriter := buildJSONHashWriter{
		getHasher: parseHasher,
	}
	parseEncoder := json.NewEncoder(&parseWriter)
	if parseErr := parseEncoder.Encode(parseValue); parseErr != nil {
		return fmt.Errorf("runtime2: encode json hash payload: %w", parseErr)
	}
	if parseErr := parseWriter.Flush(); parseErr != nil {
		return fmt.Errorf("runtime2: flush json hash payload: %w", parseErr)
	}
	return nil
}

// writeJSONHashLiteral appends one fixed JSON token into the destination hasher.
func writeJSONHashLiteral(parseHasher hash.Hash, parseLiteral []byte) error {
	if parseHasher == nil {
		return fmt.Errorf("runtime2: json hash literal hasher is nil")
	}
	if len(parseLiteral) == 0 {
		return nil
	}
	_, parseErr := parseHasher.Write(parseLiteral)
	return parseErr
}
