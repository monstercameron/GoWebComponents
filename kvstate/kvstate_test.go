package kvstate

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCodecRoundTrips(t *testing.T) {
	type payload struct {
		Name  string
		Count int
		Tags  []string
	}
	parseValue := payload{Name: "hi", Count: 7, Tags: []string{"a", "b"}}
	for parseName, parseCodec := range map[string]Codec{"json": JSONCodec{}, "cbor": CBORCodec{}} {
		parseData, parseErr := parseCodec.Encode(parseValue)
		if parseErr != nil {
			t.Fatalf("%s encode: %v", parseName, parseErr)
		}
		var parseBack payload
		if parseErr := parseCodec.Decode(parseData, &parseBack); parseErr != nil {
			t.Fatalf("%s decode: %v", parseName, parseErr)
		}
		if parseBack.Name != parseValue.Name || parseBack.Count != parseValue.Count || len(parseBack.Tags) != 2 {
			t.Fatalf("%s round-trip mismatch: %+v", parseName, parseBack)
		}
	}
}

func TestLastWriteWinsResolver(t *testing.T) {
	parseLocal := Record{Key: "k", Value: []byte("old"), UpdatedAt: 100}
	parseIncoming := Record{Key: "k", Value: []byte("new"), UpdatedAt: 200}
	if parseGot := (LastWriteWins{}).Resolve(parseLocal, parseIncoming); string(parseGot.Value) != "new" {
		t.Fatalf("newer write should win, got %q", parseGot.Value)
	}
	// Older incoming loses.
	parseStale := Record{Key: "k", Value: []byte("stale"), UpdatedAt: 50}
	if parseGot := (LastWriteWins{}).Resolve(parseLocal, parseStale); string(parseGot.Value) != "old" {
		t.Fatalf("stale write should lose, got %q", parseGot.Value)
	}
}

func TestVersionedResolver(t *testing.T) {
	parseLocal := Record{Key: "k", Value: []byte("v2"), Version: 2}
	if parseGot := (Versioned{}).Resolve(parseLocal, Record{Value: []byte("v1"), Version: 1}); string(parseGot.Value) != "v2" {
		t.Fatalf("lower version must be rejected, got %q", parseGot.Value)
	}
	if parseGot := (Versioned{}).Resolve(parseLocal, Record{Value: []byte("v3"), Version: 3}); string(parseGot.Value) != "v3" {
		t.Fatalf("higher version must win, got %q", parseGot.Value)
	}
}

func TestImmediateStrategyFlushesSynchronously(t *testing.T) {
	parseFlushed := 0
	parseStrategy := Immediate{}
	parseStrategy.OnWrite(context.Background(), "k", func(context.Context) error { parseFlushed++; return nil })
	if parseFlushed != 1 {
		t.Fatalf("Immediate should flush on write, got %d", parseFlushed)
	}
	if parseErr := parseStrategy.Close(context.Background()); parseErr != nil {
		t.Fatalf("Close: %v", parseErr)
	}
}

func TestDebouncedStrategyCoalescesAndFlushes(t *testing.T) {
	var parseMu sync.Mutex
	parseFlushed := 0
	parseFlush := func(context.Context) error { parseMu.Lock(); parseFlushed++; parseMu.Unlock(); return nil }
	parseStrategy := Debounced(40 * time.Millisecond)

	// Burst of writes coalesces into one flush.
	for parseI := 0; parseI < 5; parseI++ {
		parseStrategy.OnWrite(context.Background(), "k", parseFlush)
	}
	time.Sleep(100 * time.Millisecond)
	parseMu.Lock()
	parseAfterBurst := parseFlushed
	parseMu.Unlock()
	if parseAfterBurst != 1 {
		t.Fatalf("debounced burst should flush once, got %d", parseAfterBurst)
	}

	// Close flushes any pending write immediately.
	parseStrategy.OnWrite(context.Background(), "k", parseFlush)
	if parseErr := parseStrategy.Close(context.Background()); parseErr != nil {
		t.Fatalf("Close: %v", parseErr)
	}
	parseMu.Lock()
	parseFinal := parseFlushed
	parseMu.Unlock()
	if parseFinal != 2 {
		t.Fatalf("Close should flush the pending write, got %d", parseFinal)
	}
}

func TestOnUnloadStrategyDefersToClose(t *testing.T) {
	parseFlushed := 0
	parseStrategy := OnUnload()
	parseStrategy.OnWrite(context.Background(), "k", func(context.Context) error { parseFlushed++; return nil })
	if parseFlushed != 0 {
		t.Fatalf("OnUnload should not flush on write, got %d", parseFlushed)
	}
	if parseErr := parseStrategy.Close(context.Background()); parseErr != nil {
		t.Fatalf("Close: %v", parseErr)
	}
	if parseFlushed != 1 {
		t.Fatalf("OnUnload should flush on Close, got %d", parseFlushed)
	}
}

func TestRegistryRoundTrips(t *testing.T) {
	RegisterStrategy("imm", Immediate{})
	if parseGot, parseOk := StrategyByName("imm"); !parseOk || parseGot == nil {
		t.Fatal("strategy not retrievable by name")
	}
	if _, parseOk := StrategyByName("nope"); parseOk {
		t.Fatal("unknown strategy should not resolve")
	}
	RegisterCodec("j", JSONCodec{})
	if _, parseOk := CodecByName("j"); !parseOk {
		t.Fatal("codec not retrievable")
	}
	RegisterResolver("lww", LastWriteWins{})
	if _, parseOk := ResolverByName("lww"); !parseOk {
		t.Fatal("resolver not retrievable")
	}
	parseBackend := newMemBackend()
	RegisterBackend("mem", parseBackend)
	if _, parseOk := BackendByName("mem"); !parseOk {
		t.Fatal("backend not retrievable")
	}
}

// memBackend is a custom in-memory PersistenceBackend, proving the extension
// interface (a user can persist anywhere with the same hook/atom API).
type memBackend struct {
	mu      sync.Mutex
	records map[string]Record
}

func newMemBackend() *memBackend { return &memBackend{records: map[string]Record{}} }

func (parseB *memBackend) Load(parseCtx context.Context, parseKey string) (Record, bool, error) {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	parseRec, parseOk := parseB.records[parseKey]
	return parseRec, parseOk, nil
}
func (parseB *memBackend) Save(parseCtx context.Context, parseRecord Record) error {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	parseB.records[parseRecord.Key] = parseRecord
	return nil
}
func (parseB *memBackend) Delete(parseCtx context.Context, parseKey string) error {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	delete(parseB.records, parseKey)
	return nil
}
func (parseB *memBackend) Keys(parseCtx context.Context) ([]string, error) {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	parseKeys := make([]string, 0, len(parseB.records))
	for parseK := range parseB.records {
		parseKeys = append(parseKeys, parseK)
	}
	return parseKeys, nil
}

func TestCustomBackendCRUD(t *testing.T) {
	parseCtx := context.Background()
	parseBackend := newMemBackend()
	if parseErr := parseBackend.Save(parseCtx, Record{Key: "a", Value: []byte("1"), Version: 1, UpdatedAt: 10}); parseErr != nil {
		t.Fatal(parseErr)
	}
	parseRec, parseFound, parseErr := parseBackend.Load(parseCtx, "a")
	if parseErr != nil || !parseFound || string(parseRec.Value) != "1" {
		t.Fatalf("load round-trip failed: rec=%+v found=%v err=%v", parseRec, parseFound, parseErr)
	}
	parseKeys, _ := parseBackend.Keys(parseCtx)
	if len(parseKeys) != 1 || parseKeys[0] != "a" {
		t.Fatalf("keys: %v", parseKeys)
	}
	if parseErr := parseBackend.Delete(parseCtx, "a"); parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, parseFound, _ := parseBackend.Load(parseCtx, "a"); parseFound {
		t.Fatal("record should be gone after delete")
	}
}
