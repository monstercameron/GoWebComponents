//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"errors"
	"reflect"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func installMockMutationQueueStorage(parseT *testing.T) map[string]string {
	parseT.Helper()

	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseStorage := parseObjectCtor.New()
	parseData := map[string]string{}
	var parseKeys []string

	parseReindex := func() {
		parseKeys = parseKeys[:0]
		for parseKey := range parseData {
			parseKeys = append(parseKeys, parseKey)
		}
		parseStorage.Set("length", len(parseKeys))
	}

	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseValue, parseOk := parseData[parseArgs[0].String()]
		if !parseOk {
			return js.Null()
		}
		return parseValue
	})
	setItemFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseData[parseArgs2[0].String()] = parseArgs2[1].String()
		parseReindex()
		return nil
	})
	parseRemoveItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		delete(parseData, parseArgs3[0].String())
		parseReindex()
		return nil
	})
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		for parseKey2 := range parseData {
			delete(parseData, parseKey2)
		}
		parseReindex()
		return nil
	})
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseIndex := parseArgs5[0].Int()
		if parseIndex < 0 || parseIndex >= len(parseKeys) {
			return js.Null()
		}
		return parseKeys[parseIndex]
	})

	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)

	parseRestoreStorage := setGlobalJSValue("localStorage", parseStorage)
	parseRestoreIndexedDB := setGlobalJSValue("indexedDB", js.Undefined())
	parseT.Cleanup(func() {
		parseRestoreIndexedDB()
		parseRestoreStorage()
		getItemFn.Release()
		setItemFn.Release()
		parseRemoveItemFn.Release()
		clearFn.Release()
		parseKeyFn.Release()
	})

	return parseData
}

func TestMutationQueueEnqueuePersistsAcrossReopen(parseT *testing.T) {
	parseStorage := installMockMutationQueueStorage(parseT)
	parseNow := time.Date(2026, time.March, 18, 9, 0, 0, 0, time.UTC)

	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey: "offline-persist",
		Now:        func() time.Time { return parseNow },
	})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}

	parseEntry, parseErr := parseQueue.Enqueue(MutationDraft{
		Kind:     "order.submit",
		Method:   "post",
		URL:      "/api/orders",
		DedupKey: "order:123",
		Headers:  map[string]string{"Content-Type": "application/json"},
		Body:     map[string]any{"id": 123},
		Metadata: map[string]string{"schema": "v1"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr)
	}
	if parseStorage["offline-persist"] == "" {
		parseT.Fatal("expected queued mutation to persist into storage")
	}
	if parseEntry.State != MutationQueued || parseEntry.Method != "POST" {
		parseT.Fatalf("unexpected queued entry: %+v", parseEntry)
	}

	parseReopened, parseErr := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-persist"})
	if parseErr != nil {
		parseT.Fatalf("expected reopened queue, got %v", parseErr)
	}
	parseEntries, parseErr := parseReopened.List()
	if parseErr != nil {
		parseT.Fatalf("expected persisted entries to load, got %v", parseErr)
	}
	if len(parseEntries) != 1 {
		parseT.Fatalf("expected one persisted entry, got %d", len(parseEntries))
	}
	if parseEntries[0].DedupKey != "order:123" || parseEntries[0].Kind != "order.submit" {
		parseT.Fatalf("unexpected persisted entry: %+v", parseEntries[0])
	}
	if parseEntries[0].Method != "POST" || parseEntries[0].MaxAttempts == 0 {
		parseT.Fatalf("expected replay defaults to survive reopen, got %+v", parseEntries[0])
	}
}

func TestMutationQueueReplayWritesFrameworkLogs(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	runtime.ClearLogs()
	defer runtime.ClearLogs()

	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey: "offline-logs",
		Now:        func() time.Time { return time.Date(2026, time.March, 18, 9, 30, 0, 0, time.UTC) },
	})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}

	parseEntry, parseErr := parseQueue.Enqueue(MutationDraft{
		Kind: "order.submit",
		URL:  "/api/orders",
	})
	if parseErr != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr)
	}

	parseReport, parseErr := parseQueue.Replay(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		if parseMutation.ID != parseEntry.ID {
			parseT.Fatalf("unexpected mutation replayed: %+v", parseMutation)
		}
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("expected replay to succeed, got %v", parseErr)
	}
	if parseReport.Succeeded != 1 {
		parseT.Fatalf("expected one successful replay, got %+v", parseReport)
	}

	parseLogs := runtime.GetLogs()
	isParseFoundQueued := false
	isParseFoundSucceeded := false
	for _, parseLog := range parseLogs {
		switch parseLog.Message {
		case "mutation queued for replay":
			isParseFoundQueued = parseLog.Fields["id"] == parseEntry.ID
		case "mutation replay succeeded":
			isParseFoundSucceeded = parseLog.Fields["id"] == parseEntry.ID
		}
	}
	if !isParseFoundQueued || !isParseFoundSucceeded {
		parseT.Fatalf("expected queued and replay logs, got %+v", parseLogs)
	}
}

func TestMutationQueueSuppressesDuplicatesByDedupKey(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-dedup"})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}

	parseFirst, parseErr := parseQueue.Enqueue(MutationDraft{
		URL:      "/api/orders",
		DedupKey: "draft:42",
	})
	if parseErr != nil {
		parseT.Fatalf("expected first enqueue to succeed, got %v", parseErr)
	}
	parseSecond, parseErr := parseQueue.Enqueue(MutationDraft{
		URL:      "/api/orders",
		DedupKey: "draft:42",
	})
	if parseErr != nil {
		parseT.Fatalf("expected duplicate enqueue to short-circuit, got %v", parseErr)
	}

	if parseFirst.ID != parseSecond.ID {
		parseT.Fatalf("expected duplicate enqueue to return existing entry, got %q and %q", parseFirst.ID, parseSecond.ID)
	}
	parseEntries, parseErr := parseQueue.List()
	if parseErr != nil {
		parseT.Fatalf("expected list to succeed, got %v", parseErr)
	}
	if len(parseEntries) != 1 {
		parseT.Fatalf("expected one queued mutation after dedup, got %d", len(parseEntries))
	}
}

func TestMutationQueueReplaySuccessRemovesEntry(parseT *testing.T) {
	parseStorage := installMockMutationQueueStorage(parseT)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-success"})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}
	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/orders", Kind: "order.submit"}); parseErr2 != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr2)
	}

	var parseReplayed []string
	parseReport, parseErr := parseQueue.Replay(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		parseReplayed = append(parseReplayed, parseMutation.Kind)
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("expected replay to succeed, got %v", parseErr)
	}
	if !reflect.DeepEqual(parseReplayed, []string{"order.submit"}) {
		parseT.Fatalf("unexpected replay order: %#v", parseReplayed)
	}
	if parseReport.Succeeded != 1 || parseReport.Remaining != 0 || parseReport.Retried != 0 || parseReport.DeadLetters != 0 {
		parseT.Fatalf("unexpected replay report: %+v", parseReport)
	}
	if _, parseOk := parseStorage["offline-success"]; parseOk {
		parseT.Fatalf("expected successful replay to clear persisted storage, got %#v", parseStorage)
	}
}

func TestMutationQueueReplayFailureSchedulesRetryAndDefersUntilDue(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	parseCurrent := time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey:  "offline-retry",
		BaseDelay:   time.Second,
		MaxDelay:    10 * time.Second,
		MaxAttempts: 3,
		Now:         func() time.Time { return parseCurrent },
	})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}
	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/orders"}); parseErr2 != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr2)
	}

	parseAttempts := 0
	parseReport, parseErr := parseQueue.Replay(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		parseAttempts++
		return errors.New("temporary outage")
	})
	if parseErr != nil {
		parseT.Fatalf("expected replay to persist retry state, got %v", parseErr)
	}
	if parseAttempts != 1 || parseReport.Retried != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("unexpected retry report: attempts=%d report=%+v", parseAttempts, parseReport)
	}

	parseEntries, parseErr := parseQueue.List()
	if parseErr != nil {
		parseT.Fatalf("expected list to succeed, got %v", parseErr)
	}
	if len(parseEntries) != 1 {
		parseT.Fatalf("expected one retained queued mutation, got %d", len(parseEntries))
	}
	parseEntry := parseEntries[0]
	if parseEntry.State != MutationRetrying || parseEntry.Attempts != 1 || parseEntry.LastError != "temporary outage" {
		parseT.Fatalf("unexpected retry entry: %+v", parseEntry)
	}
	if !parseEntry.NextAttemptAt.Equal(parseCurrent.Add(time.Second)) {
		parseT.Fatalf("expected next attempt after backoff, got %s", parseEntry.NextAttemptAt)
	}

	parseCurrent = parseCurrent.Add(500 * time.Millisecond)
	parseAttempts = 0
	parseReport, parseErr = parseQueue.Replay(context.Background(), func(parseCtx2 context.Context, parseMutation2 QueuedMutation) error {
		parseAttempts++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("expected deferred replay to succeed, got %v", parseErr)
	}
	if parseAttempts != 0 || parseReport.Deferred != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("expected replay to defer until due, attempts=%d report=%+v", parseAttempts, parseReport)
	}
}

func TestMutationQueueMovesToDeadLetterAfterMaxAttempts(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	parseCurrent := time.Date(2026, time.March, 18, 11, 0, 0, 0, time.UTC)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey:  "offline-dead",
		BaseDelay:   time.Second,
		MaxAttempts: 2,
		Now:         func() time.Time { return parseCurrent },
	})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}
	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/orders"}); parseErr2 != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr2)
	}

	_, parseErr = parseQueue.Replay(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		return errors.New("still offline")
	})
	if parseErr != nil {
		parseT.Fatalf("expected first replay to persist retry state, got %v", parseErr)
	}
	parseCurrent = parseCurrent.Add(2 * time.Second)

	parseReport, parseErr := parseQueue.Replay(context.Background(), func(parseCtx2 context.Context, parseMutation2 QueuedMutation) error {
		return errors.New("still offline")
	})
	if parseErr != nil {
		parseT.Fatalf("expected second replay to persist dead-letter state, got %v", parseErr)
	}
	if parseReport.DeadLetters != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("unexpected dead-letter report: %+v", parseReport)
	}

	parseEntries, parseErr := parseQueue.List()
	if parseErr != nil {
		parseT.Fatalf("expected list to succeed, got %v", parseErr)
	}
	if len(parseEntries) != 1 {
		parseT.Fatalf("expected one dead-letter entry, got %d", len(parseEntries))
	}
	parseEntry := parseEntries[0]
	if parseEntry.State != MutationDead || parseEntry.Attempts != 2 || !parseEntry.NextAttemptAt.IsZero() {
		parseT.Fatalf("unexpected dead-letter entry: %+v", parseEntry)
	}
}

func TestMutationQueueReplayCancellationKeepsRemainingEntries(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-cancel"})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}
	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/first", Kind: "first"}); parseErr2 != nil {
		parseT.Fatalf("expected first enqueue to succeed, got %v", parseErr2)
	}
	if _, parseErr3 := parseQueue.Enqueue(MutationDraft{URL: "/api/second", Kind: "second"}); parseErr3 != nil {
		parseT.Fatalf("expected second enqueue to succeed, got %v", parseErr3)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseReport, parseErr := parseQueue.Replay(parseCtx, func(parseCtx2 context.Context, parseMutation QueuedMutation) error {
		parseCancel()
		return nil
	})
	if !errors.Is(parseErr, context.Canceled) {
		parseT.Fatalf("expected replay cancellation, got %v", parseErr)
	}
	if parseReport.Succeeded != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("unexpected cancellation report: %+v", parseReport)
	}

	parseEntries, parseErr := parseQueue.List()
	if parseErr != nil {
		parseT.Fatalf("expected list to succeed, got %v", parseErr)
	}
	if len(parseEntries) != 1 || parseEntries[0].Kind != "second" {
		parseT.Fatalf("expected second entry to remain queued, got %+v", parseEntries)
	}
}

func TestMutationQueueConflictWithoutHandlerMovesToDeadLetter(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-conflict-dead"})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}
	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/orders", Kind: "order.submit"}); parseErr2 != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr2)
	}

	parseReport, parseErr := parseQueue.Replay(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		return NewMutationConflict(errors.New("etag mismatch"), MutationConflict{Code: "etag_mismatch", LocalVersion: "1", RemoteVersion: "2", Message: "server revision is newer"})
	})
	if parseErr != nil {
		parseT.Fatalf("expected replay to persist conflict dead-letter state, got %v", parseErr)
	}
	if parseReport.Conflicts != 1 || parseReport.DeadLetters != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("unexpected conflict report: %+v", parseReport)
	}

	parseEntries, parseErr := parseQueue.List()
	if parseErr != nil {
		parseT.Fatalf("expected list to succeed, got %v", parseErr)
	}
	if len(parseEntries) != 1 || parseEntries[0].State != MutationDead {
		parseT.Fatalf("expected one dead-letter conflict entry, got %+v", parseEntries)
	}
	if parseEntries[0].LastError == "" || parseEntries[0].Attempts != 1 {
		parseT.Fatalf("expected conflict details to be retained, got %+v", parseEntries[0])
	}
}

func TestMutationConflictHelpersExposeStructuredConflict(parseT *testing.T) {
	parseErr := NewMutationConflict(errors.New("etag mismatch"), MutationConflict{
		Code:          "etag_mismatch",
		LocalVersion:  "1",
		RemoteVersion: "2",
		Message:       "server revision is newer",
	})

	if !IsMutationConflict(parseErr) {
		parseT.Fatal("expected IsMutationConflict to recognize wrapped conflict errors")
	}
	if IsMutationConflict(errors.New("plain error")) {
		parseT.Fatal("expected IsMutationConflict to reject non-conflict errors")
	}

	parseConflictErr, parseOk := AsMutationConflictError(parseErr)
	if !parseOk || parseConflictErr == nil {
		parseT.Fatal("expected AsMutationConflictError to unwrap the structured conflict")
	}
	if parseConflictErr.Conflict.Code != "etag_mismatch" {
		parseT.Fatalf("expected conflict code to round-trip, got %+v", parseConflictErr.Conflict)
	}

	parseConflict, parseOk := GetMutationConflict(parseErr)
	if !parseOk {
		parseT.Fatal("expected GetMutationConflict to expose conflict details")
	}
	if parseConflict.LocalVersion != "1" || parseConflict.RemoteVersion != "2" || parseConflict.Message != "server revision is newer" {
		parseT.Fatalf("unexpected conflict details: %+v", parseConflict)
	}

	if _, parseOk2 := GetMutationConflict(errors.New("plain error")); parseOk2 {
		parseT.Fatal("expected GetMutationConflict to reject non-conflict errors")
	}
}

func TestMutationQueueConflictHandlerCanRequeueResolvedMutation(parseT *testing.T) {
	installMockMutationQueueStorage(parseT)
	parseCurrent := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey: "offline-conflict-resolve",
		Now:        func() time.Time { return parseCurrent },
	})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue to open, got %v", parseErr)
	}
	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/orders", Kind: "order.submit", Metadata: map[string]string{"revision": "1"}}); parseErr2 != nil {
		parseT.Fatalf("expected enqueue to succeed, got %v", parseErr2)
	}

	parseReport, parseErr := parseQueue.ReplayWithOptions(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		if parseMutation.Metadata["resolved"] == "server-2" {
			return nil
		}
		return NewMutationConflict(errors.New("etag mismatch"), MutationConflict{Code: "etag_mismatch", LocalVersion: parseMutation.Metadata["revision"], RemoteVersion: "2", Message: "server revision is newer"})
	}, MutationReplayOptions{ConflictHandler: func(parseCtx2 context.Context, parseMutation2 QueuedMutation, parseConflict MutationConflict) (MutationConflictResolution, error) {
		return MutationConflictResolution{
			Action:  MutationResolutionReplace,
			Message: "rebased onto server revision 2",
			Draft: MutationDraft{
				Metadata: map[string]string{"revision": parseConflict.RemoteVersion, "resolved": "server-2"},
				Body:     map[string]any{"merge": "accepted"},
			},
		}, nil
	}})
	if parseErr != nil {
		parseT.Fatalf("expected conflict resolution replay to succeed, got %v", parseErr)
	}
	if parseReport.Conflicts != 1 || parseReport.Resolved != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("unexpected conflict resolution report: %+v", parseReport)
	}

	parseEntries, parseErr := parseQueue.List()
	if parseErr != nil {
		parseT.Fatalf("expected list to succeed, got %v", parseErr)
	}
	if len(parseEntries) != 1 {
		parseT.Fatalf("expected one requeued resolved entry, got %d", len(parseEntries))
	}
	parseEntry := parseEntries[0]
	if parseEntry.State != MutationQueued || parseEntry.Attempts != 0 || parseEntry.Metadata["resolved"] != "server-2" || parseEntry.Metadata["revision"] != "2" {
		parseT.Fatalf("unexpected resolved entry: %+v", parseEntry)
	}
	if parseEntry.LastError != "rebased onto server revision 2" {
		parseT.Fatalf("expected resolution note to persist, got %+v", parseEntry)
	}

	parseFinalReport, parseErr := parseQueue.Replay(context.Background(), func(parseCtx3 context.Context, parseMutation3 QueuedMutation) error {
		if parseMutation3.Metadata["resolved"] != "server-2" {
			parseT.Fatalf("expected rebased mutation to be replayed, got %+v", parseMutation3)
		}
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("expected resolved replay to succeed, got %v", parseErr)
	}
	if parseFinalReport.Succeeded != 1 || parseFinalReport.Remaining != 0 {
		parseT.Fatalf("unexpected final replay report: %+v", parseFinalReport)
	}
}
