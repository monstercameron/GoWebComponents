package runtime2

import (
	"errors"
	"strings"
	"testing"
)

type jsonHashTestHasher struct {
	storeBytes       []byte
	storeWriteErrors []error
	storeWriteCount  int
}

// buildJSONHashTestHasher constructs one test hasher that can capture bytes and inject write failures by write index.
func buildJSONHashTestHasher(parseWriteErrors ...error) *jsonHashTestHasher {
	return &jsonHashTestHasher{
		storeWriteErrors: parseWriteErrors,
	}
}

// Write captures bytes unless the current write index is configured to fail.
func (parseHasher *jsonHashTestHasher) Write(parseBytes []byte) (int, error) {
	if parseHasher.storeWriteCount < len(parseHasher.storeWriteErrors) {
		if parseWriteErr := parseHasher.storeWriteErrors[parseHasher.storeWriteCount]; parseWriteErr != nil {
			parseHasher.storeWriteCount++
			return 0, parseWriteErr
		}
	}
	parseHasher.storeWriteCount++
	parseHasher.storeBytes = append(parseHasher.storeBytes, parseBytes...)
	return len(parseBytes), nil
}

// Sum appends the captured bytes to the provided prefix.
func (parseHasher *jsonHashTestHasher) Sum(parsePrefix []byte) []byte {
	return append(parsePrefix, parseHasher.storeBytes...)
}

// Reset clears the captured bytes and write counters.
func (parseHasher *jsonHashTestHasher) Reset() {
	parseHasher.storeBytes = nil
	parseHasher.storeWriteCount = 0
}

// Size reports the current captured byte length for hash.Hash compatibility in tests.
func (parseHasher *jsonHashTestHasher) Size() int {
	return len(parseHasher.storeBytes)
}

// BlockSize reports one-byte blocks for hash.Hash compatibility in tests.
func (parseHasher *jsonHashTestHasher) BlockSize() int {
	return 1
}

// TestBuildJSONHashWriterBuffersPendingBytesAndFlushesTrailingContent verifies the streaming writer covers its fast-path and fallback buffering branches.
func TestBuildJSONHashWriterBuffersPendingBytesAndFlushesTrailingContent(parseT *testing.T) {
	var parseNilWriter *buildJSONHashWriter
	if _, parseErr := parseNilWriter.Write([]byte("x")); parseErr == nil {
		parseT.Fatal("expected nil json hash writer write to fail")
	}
	if parseErr := parseNilWriter.Flush(); parseErr != nil {
		parseT.Fatalf("expected nil json hash writer flush to be ignored, got %v", parseErr)
	}

	parseHasher := buildJSONHashTestHasher()
	parseWriter := &buildJSONHashWriter{getHasher: parseHasher}
	if parseWritten, parseErr := parseWriter.Write([]byte{}); parseErr != nil || parseWritten != 0 {
		parseT.Fatalf("expected empty write to be ignored, got written=%d err=%v", parseWritten, parseErr)
	}
	if parseWritten, parseErr := parseWriter.Write([]byte("a")); parseErr != nil || parseWritten != 1 {
		parseT.Fatalf("expected single-byte write to buffer trailing byte, got written=%d err=%v", parseWritten, parseErr)
	}
	if parseWritten, parseErr := parseWriter.Write([]byte("bc")); parseErr != nil || parseWritten != 2 {
		parseT.Fatalf("expected buffered fast-path write to succeed, got written=%d err=%v", parseWritten, parseErr)
	}
	parseLargePayload := []byte(strings.Repeat("d", 600))
	if parseWritten, parseErr := parseWriter.Write(parseLargePayload); parseErr != nil || parseWritten != len(parseLargePayload) {
		parseT.Fatalf("expected buffered fallback write to succeed, got written=%d err=%v", parseWritten, parseErr)
	}
	if parseErr := parseWriter.Flush(); parseErr != nil {
		parseT.Fatalf("expected flush to write pending trailing byte, got %v", parseErr)
	}
	parseExpected := "abc" + strings.Repeat("d", 600)
	if string(parseHasher.storeBytes) != parseExpected {
		parseT.Fatalf("captured json hash bytes = %q, want %q", string(parseHasher.storeBytes), parseExpected)
	}
}

// TestBuildJSONHashWriterFlushSuppressesTrailingEncoderNewline verifies the writer drops the JSON encoder newline during flush.
func TestBuildJSONHashWriterFlushSuppressesTrailingEncoderNewline(parseT *testing.T) {
	parseHasher := buildJSONHashTestHasher()
	parseWriter := &buildJSONHashWriter{getHasher: parseHasher}
	if _, parseErr := parseWriter.Write([]byte("{\"name\":\"demo\"}\n")); parseErr != nil {
		parseT.Fatalf("expected newline-terminated json write to succeed, got %v", parseErr)
	}
	if parseErr := parseWriter.Flush(); parseErr != nil {
		parseT.Fatalf("expected newline suppression flush to succeed, got %v", parseErr)
	}
	if string(parseHasher.storeBytes) != "{\"name\":\"demo\"}" {
		parseT.Fatalf("captured json hash bytes = %q, want newline-free payload", string(parseHasher.storeBytes))
	}
}

// TestBuildJSONHashDigestAndLiteralPropagateErrors verifies digest and literal helpers surface hasher and encoder failures while preserving successful output semantics.
func TestBuildJSONHashDigestAndLiteralPropagateErrors(parseT *testing.T) {
	parseWriteErr := errors.New("write boom")
	parseWriter := &buildJSONHashWriter{getHasher: buildJSONHashTestHasher(parseWriteErr)}
	if _, parseErr := parseWriter.Write([]byte("ab")); !errors.Is(parseErr, parseWriteErr) {
		parseT.Fatalf("expected direct writer error propagation, got %v", parseErr)
	}

	parseFlushErr := errors.New("flush boom")
	parseFlushWriter := &buildJSONHashWriter{
		getHasher:      buildJSONHashTestHasher(parseFlushErr),
		getPendingByte: 'x',
		hasPendingByte: true,
	}
	if parseErr := parseFlushWriter.Flush(); !errors.Is(parseErr, parseFlushErr) {
		parseT.Fatalf("expected flush writer error propagation, got %v", parseErr)
	}

	if parseErr := buildJSONHashDigest(nil, map[string]any{"name": "demo"}); parseErr == nil {
		parseT.Fatal("expected nil hasher digest to fail")
	}

	parseDigestHasher := buildJSONHashTestHasher()
	if parseErr := buildJSONHashDigest(parseDigestHasher, struct {
		Name string `json:"name"`
	}{Name: "demo"}); parseErr != nil {
		parseT.Fatalf("expected successful digest build, got %v", parseErr)
	}
	if string(parseDigestHasher.storeBytes) != "{\"name\":\"demo\"}" {
		parseT.Fatalf("captured digest bytes = %q, want %q", string(parseDigestHasher.storeBytes), "{\"name\":\"demo\"}")
	}

	if parseErr := buildJSONHashDigest(buildJSONHashTestHasher(), make(chan int)); parseErr == nil {
		parseT.Fatal("expected unsupported payload digest to fail encoding")
	}

	if parseErr := writeJSONHashLiteral(nil, []byte("token")); parseErr == nil {
		parseT.Fatal("expected nil hasher literal write to fail")
	}
	if parseErr := writeJSONHashLiteral(buildJSONHashTestHasher(), nil); parseErr != nil {
		parseT.Fatalf("expected empty literal write to be ignored, got %v", parseErr)
	}
	parseLiteralErr := errors.New("literal boom")
	if parseErr := writeJSONHashLiteral(buildJSONHashTestHasher(parseLiteralErr), []byte("token")); !errors.Is(parseErr, parseLiteralErr) {
		parseT.Fatalf("expected literal writer error propagation, got %v", parseErr)
	}
}
