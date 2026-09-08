package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStorageServiceSerializedLimitPreservesReadableStore checks the actual on-disk quota.
func TestStorageServiceSerializedLimitPreservesReadableStore(parseTest *testing.T) {
	parsePath := filepath.Join(parseTest.TempDir(), "state.json")
	parseService, parseErr := NewStorageService(parsePath, nil)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer func() {
		if parseCloseErr := parseService.closeStorage(); parseCloseErr != nil {
			parseTest.Error(parseCloseErr)
		}
	}()
	if parseErr = parseService.Save(context.Background(), storageWire("safe", "kept", 1)); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseBefore, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseRecords := map[string]storageEntry{}
	for parseIndex := 0; parseIndex < 13; parseIndex++ {
		parseKey := string(rune('a' + parseIndex))
		parseRecords[parseKey] = storageEntry{Key: parseKey, Value: make([]byte, storageMaxValueBytes), Version: 1}
	}
	if parseErr = validateStorageTotal(parseRecords); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr = parseService.writeFile(parseRecords); parseErr == nil {
		parseTest.Fatal("accepted encoded store too large to reopen")
	}
	parseAfter, parseErr := os.ReadFile(parsePath)
	if parseErr != nil || string(parseAfter) != string(parseBefore) {
		parseTest.Fatalf("rejected write changed durable file: %v", parseErr)
	}
}

// TestStorageCommitCancellationBeforeLockPreventsWrite checks the commit-side cancellation gate.
func TestStorageCommitCancellationBeforeLockPreventsWrite(parseTest *testing.T) {
	parseService, parseErr := NewStorageService(filepath.Join(parseTest.TempDir(), "state.json"), nil)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer func() {
		if parseCloseErr := parseService.closeStorage(); parseCloseErr != nil {
			parseTest.Error(parseCloseErr)
		}
	}()
	parseContext, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseService.parseMutex.Lock()
	parseFinished := make(chan error, 1)
	go func() { parseFinished <- parseService.commit(parseContext, storageEntry{Key: "a", Version: 1}) }()
	parseCancel()
	parseService.parseMutex.Unlock()
	select {
	case parseErr = <-parseFinished:
		if !errors.Is(parseErr, context.Canceled) {
			parseTest.Fatalf("commit after cancellation: %v", parseErr)
		}
	case <-time.After(time.Second):
		parseTest.Fatal("queued commit did not finish")
	}
	parseResult, parseErr := parseService.Load(context.Background(), "a")
	if parseErr != nil || parseResult.Found {
		parseTest.Fatalf("cancelled commit wrote data: %#v %v", parseResult, parseErr)
	}
}

// TestDecodeStorageEntryRejectsOversizedWire bounds allocation before base64 decoding.
func TestDecodeStorageEntryRejectsOversizedWire(parseTest *testing.T) {
	parseWire := storageWire("a", "", 1)
	parseWire.Value = strings.Repeat("A", ((storageMaxValueBytes+2)/3)*4+4)
	if _, parseErr := decodeStorageEntry(parseWire); parseErr == nil {
		parseTest.Fatal("accepted oversized encoded value")
	}
}
