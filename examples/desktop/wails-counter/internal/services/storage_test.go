package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/desktop"
)

func storageWire(parseKey, parseValue string, parseVersion int64) desktop.StorageWireRecord {
	return desktop.StorageWireRecord{Key: parseKey, Value: base64.StdEncoding.EncodeToString([]byte(parseValue)), Version: fmt.Sprint(parseVersion), UpdatedAt: fmt.Sprint(parseVersion)}
}

func TestStorageServiceRestartAndTombstone(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "state.json")
	parseService, parseErr := NewStorageService(parsePath, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "one", 1)); parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr = parseService.closeStorage(); parseErr != nil {
		t.Fatal(parseErr)
	}
	parseService, parseErr = NewStorageService(parsePath, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	parseLoaded, parseErr := parseService.Load(context.Background(), "a")
	if parseErr != nil || !parseLoaded.Found || parseLoaded.Record == nil {
		t.Fatalf("load after restart: %#v %v", parseLoaded, parseErr)
	}
	if parseErr = parseService.Delete(context.Background(), "a"); parseErr != nil {
		t.Fatal(parseErr)
	}
	parseLoaded, parseErr = parseService.Load(context.Background(), "a")
	if parseErr != nil || parseLoaded.Found {
		t.Fatalf("tombstone should hide key: %#v %v", parseLoaded, parseErr)
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "stale", 1)); parseErr == nil {
		t.Fatal("expected stale resurrection rejection")
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "new", 3)); parseErr != nil {
		t.Fatalf("newer recreation: %v", parseErr)
	}
	_ = parseService.closeStorage()
}

func TestStorageServiceRejectsCorruptOversizedAndClosedReads(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "state.json")
	if parseErr := os.WriteFile(parsePath, []byte(`{"records":{"bad":{"key":"bad","value":null,"version":"1 garbage","updatedAt":1}}}`), 0o600); parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, parseErr := NewStorageService(parsePath, nil); parseErr == nil {
		t.Fatal("expected corrupt version rejection")
	}
	if parseErr := os.WriteFile(parsePath, []byte(strings.Repeat("x", storageMaxTotalBytes+1)), 0o600); parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, parseErr := NewStorageService(parsePath, nil); parseErr == nil {
		t.Fatal("expected oversized file rejection")
	}
	parseService, parseErr := NewStorageService(filepath.Join(t.TempDir(), "closed.json"), nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr = parseService.closeStorage(); parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, parseErr = parseService.Load(context.Background(), "x"); parseErr == nil {
		t.Fatal("expected closed load rejection")
	}
	if _, parseErr = parseService.Keys(context.Background()); parseErr == nil {
		t.Fatal("expected closed keys rejection")
	}
}

func TestStorageSubprocessHelper(t *testing.T) {
	if os.Getenv("GWC_STORAGE_HELPER") != "write" {
		return
	}
	parsePath := os.Getenv("GWC_STORAGE_PATH")
	parseService, parseErr := NewStorageService(parsePath, nil)
	if parseErr != nil {
		os.Exit(2)
	}
	if parseErr = parseService.Save(context.Background(), storageWire("crash", "persisted", 1)); parseErr != nil {
		os.Exit(3)
	}
	// Deliberately skip ServiceShutdown: process termination must release the OS lock.
	os.Exit(0)
}

func TestStorageServiceSubprocessPersistenceAndCrashLockRelease(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "state.json")
	parseCommand := exec.Command(os.Args[0], "-test.run", "^TestStorageSubprocessHelper$")
	parseCommand.Env = append(os.Environ(), "GWC_STORAGE_HELPER=write", "GWC_STORAGE_PATH="+parsePath)
	if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
		t.Fatalf("writer process failed: %v (%s)", parseErr, parseOutput)
	}
	parseService, parseErr := NewStorageService(parsePath, nil)
	if parseErr != nil {
		t.Fatalf("reader could not acquire released lock: %v", parseErr)
	}
	parseLoaded, parseErr := parseService.Load(context.Background(), "crash")
	if parseErr != nil || !parseLoaded.Found || parseLoaded.Record == nil {
		t.Fatalf("reader could not load writer data: %#v %v", parseLoaded, parseErr)
	}
	_ = parseService.closeStorage()
}

func TestStorageServiceLockAndStaleWrites(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "state.json")
	parseService, parseErr := NewStorageService(parsePath, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, parseErr = NewStorageService(parsePath, nil); parseErr == nil {
		t.Fatal("expected ownership lock")
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "one", 2)); parseErr != nil {
		t.Fatal(parseErr)
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "equal", 2)); parseErr == nil {
		t.Fatal("expected equal version rejection")
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "stale", 1)); parseErr == nil {
		t.Fatal("expected stale version rejection")
	}
	_ = parseService.closeStorage()
}

func TestStorageServiceConcurrentStaleWritesAndCallback(t *testing.T) {
	parsePath := filepath.Join(t.TempDir(), "state.json")
	var parseCallbacks []desktop.StorageCommit
	var parseCallbackMutex sync.Mutex
	parseService, parseErr := NewStorageService(parsePath, func(parseCommit desktop.StorageCommit) {
		parseCallbackMutex.Lock()
		parseCallbacks = append(parseCallbacks, parseCommit)
		parseCallbackMutex.Unlock()
	})
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	var parseWait sync.WaitGroup
	for parseVersion := int64(1); parseVersion <= 8; parseVersion++ {
		parseWait.Add(1)
		go func(parseVersion int64) {
			defer parseWait.Done()
			_ = parseService.Save(context.Background(), storageWire("a", "value", parseVersion))
		}(parseVersion)
	}
	parseWait.Wait()
	parseLoaded, parseErr := parseService.Load(context.Background(), "a")
	if parseErr != nil || !parseLoaded.Found || parseLoaded.Record == nil || parseLoaded.Record.Version != "8" {
		t.Fatalf("expected newest commit: %#v %v", parseLoaded, parseErr)
	}
	parseCallbackMutex.Lock()
	defer parseCallbackMutex.Unlock()
	if len(parseCallbacks) == 0 {
		t.Fatal("expected commit callback")
	}
	_ = parseService.closeStorage()
}

func TestStorageServiceValidationAndContext(t *testing.T) {
	parseService, parseErr := NewStorageService(filepath.Join(t.TempDir(), "state.json"), nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	parseCancelled, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if parseErr = parseService.Save(parseCancelled, storageWire("a", "x", 1)); parseErr == nil {
		t.Fatal("expected cancelled context")
	}
	if parseErr = parseService.Save(context.Background(), storageWire("", "x", 1)); parseErr == nil {
		t.Fatal("expected invalid key")
	}
	if parseErr = parseService.Save(context.Background(), storageWire("a", "x", 0)); parseErr == nil {
		t.Fatal("expected invalid version")
	}
	_ = parseService.closeStorage()
}
