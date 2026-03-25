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

func installMockMutationQueueStorage(t *testing.T) map[string]string {
	t.Helper()

	global := js.Global()
	objectCtor := global.Get("Object")
	storage := objectCtor.New()
	data := map[string]string{}
	var keys []string

	reindex := func() {
		keys = keys[:0]
		for key := range data {
			keys = append(keys, key)
		}
		storage.Set("length", len(keys))
	}

	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value, ok := data[args[0].String()]
		if !ok {
			return js.Null()
		}
		return value
	})
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		data[args[0].String()] = args[1].String()
		reindex()
		return nil
	})
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		delete(data, args[0].String())
		reindex()
		return nil
	})
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for key := range data {
			delete(data, key)
		}
		reindex()
		return nil
	})
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		index := args[0].Int()
		if index < 0 || index >= len(keys) {
			return js.Null()
		}
		return keys[index]
	})

	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)

	restoreStorage := setGlobalJSValue("localStorage", storage)
	restoreIndexedDB := setGlobalJSValue("indexedDB", js.Undefined())
	t.Cleanup(func() {
		restoreIndexedDB()
		restoreStorage()
		getItemFn.Release()
		setItemFn.Release()
		removeItemFn.Release()
		clearFn.Release()
		keyFn.Release()
	})

	return data
}

func TestMutationQueueEnqueuePersistsAcrossReopen(t *testing.T) {
	storage := installMockMutationQueueStorage(t)
	now := time.Date(2026, time.March, 18, 9, 0, 0, 0, time.UTC)

	queue, err := OpenMutationQueue(MutationQueueOptions{
		StorageKey: "offline-persist",
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}

	entry, err := queue.Enqueue(MutationDraft{
		Kind:     "order.submit",
		Method:   "post",
		URL:      "/api/orders",
		DedupKey: "order:123",
		Headers:  map[string]string{"Content-Type": "application/json"},
		Body:     map[string]any{"id": 123},
		Metadata: map[string]string{"schema": "v1"},
	})
	if err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}
	if storage["offline-persist"] == "" {
		t.Fatal("expected queued mutation to persist into storage")
	}
	if entry.State != MutationQueued || entry.Method != "POST" {
		t.Fatalf("unexpected queued entry: %+v", entry)
	}

	reopened, err := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-persist"})
	if err != nil {
		t.Fatalf("expected reopened queue, got %v", err)
	}
	entries, err := reopened.List()
	if err != nil {
		t.Fatalf("expected persisted entries to load, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one persisted entry, got %d", len(entries))
	}
	if entries[0].DedupKey != "order:123" || entries[0].Kind != "order.submit" {
		t.Fatalf("unexpected persisted entry: %+v", entries[0])
	}
	if entries[0].Method != "POST" || entries[0].MaxAttempts == 0 {
		t.Fatalf("expected replay defaults to survive reopen, got %+v", entries[0])
	}
}

func TestMutationQueueReplayWritesFrameworkLogs(t *testing.T) {
	installMockMutationQueueStorage(t)
	runtime.ClearLogs()
	defer runtime.ClearLogs()

	queue, err := OpenMutationQueue(MutationQueueOptions{
		StorageKey: "offline-logs",
		Now:        func() time.Time { return time.Date(2026, time.March, 18, 9, 30, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}

	entry, err := queue.Enqueue(MutationDraft{
		Kind: "order.submit",
		URL:  "/api/orders",
	})
	if err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}

	report, err := queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		if mutation.ID != entry.ID {
			t.Fatalf("unexpected mutation replayed: %+v", mutation)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected replay to succeed, got %v", err)
	}
	if report.Succeeded != 1 {
		t.Fatalf("expected one successful replay, got %+v", report)
	}

	logs := runtime.GetLogs()
	foundQueued := false
	foundSucceeded := false
	for _, log := range logs {
		switch log.Message {
		case "mutation queued for replay":
			foundQueued = log.Fields["id"] == entry.ID
		case "mutation replay succeeded":
			foundSucceeded = log.Fields["id"] == entry.ID
		}
	}
	if !foundQueued || !foundSucceeded {
		t.Fatalf("expected queued and replay logs, got %+v", logs)
	}
}

func TestMutationQueueSuppressesDuplicatesByDedupKey(t *testing.T) {
	installMockMutationQueueStorage(t)
	queue, err := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-dedup"})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}

	first, err := queue.Enqueue(MutationDraft{
		URL:      "/api/orders",
		DedupKey: "draft:42",
	})
	if err != nil {
		t.Fatalf("expected first enqueue to succeed, got %v", err)
	}
	second, err := queue.Enqueue(MutationDraft{
		URL:      "/api/orders",
		DedupKey: "draft:42",
	})
	if err != nil {
		t.Fatalf("expected duplicate enqueue to short-circuit, got %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected duplicate enqueue to return existing entry, got %q and %q", first.ID, second.ID)
	}
	entries, err := queue.List()
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one queued mutation after dedup, got %d", len(entries))
	}
}

func TestMutationQueueReplaySuccessRemovesEntry(t *testing.T) {
	storage := installMockMutationQueueStorage(t)
	queue, err := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-success"})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/orders", Kind: "order.submit"}); err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}

	var replayed []string
	report, err := queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		replayed = append(replayed, mutation.Kind)
		return nil
	})
	if err != nil {
		t.Fatalf("expected replay to succeed, got %v", err)
	}
	if !reflect.DeepEqual(replayed, []string{"order.submit"}) {
		t.Fatalf("unexpected replay order: %#v", replayed)
	}
	if report.Succeeded != 1 || report.Remaining != 0 || report.Retried != 0 || report.DeadLetters != 0 {
		t.Fatalf("unexpected replay report: %+v", report)
	}
	if _, ok := storage["offline-success"]; ok {
		t.Fatalf("expected successful replay to clear persisted storage, got %#v", storage)
	}
}

func TestMutationQueueReplayFailureSchedulesRetryAndDefersUntilDue(t *testing.T) {
	installMockMutationQueueStorage(t)
	current := time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC)
	queue, err := OpenMutationQueue(MutationQueueOptions{
		StorageKey:  "offline-retry",
		BaseDelay:   time.Second,
		MaxDelay:    10 * time.Second,
		MaxAttempts: 3,
		Now:         func() time.Time { return current },
	})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/orders"}); err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}

	attempts := 0
	report, err := queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		attempts++
		return errors.New("temporary outage")
	})
	if err != nil {
		t.Fatalf("expected replay to persist retry state, got %v", err)
	}
	if attempts != 1 || report.Retried != 1 || report.Remaining != 1 {
		t.Fatalf("unexpected retry report: attempts=%d report=%+v", attempts, report)
	}

	entries, err := queue.List()
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one retained queued mutation, got %d", len(entries))
	}
	entry := entries[0]
	if entry.State != MutationRetrying || entry.Attempts != 1 || entry.LastError != "temporary outage" {
		t.Fatalf("unexpected retry entry: %+v", entry)
	}
	if !entry.NextAttemptAt.Equal(current.Add(time.Second)) {
		t.Fatalf("expected next attempt after backoff, got %s", entry.NextAttemptAt)
	}

	current = current.Add(500 * time.Millisecond)
	attempts = 0
	report, err = queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		attempts++
		return nil
	})
	if err != nil {
		t.Fatalf("expected deferred replay to succeed, got %v", err)
	}
	if attempts != 0 || report.Deferred != 1 || report.Remaining != 1 {
		t.Fatalf("expected replay to defer until due, attempts=%d report=%+v", attempts, report)
	}
}

func TestMutationQueueMovesToDeadLetterAfterMaxAttempts(t *testing.T) {
	installMockMutationQueueStorage(t)
	current := time.Date(2026, time.March, 18, 11, 0, 0, 0, time.UTC)
	queue, err := OpenMutationQueue(MutationQueueOptions{
		StorageKey:  "offline-dead",
		BaseDelay:   time.Second,
		MaxAttempts: 2,
		Now:         func() time.Time { return current },
	})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/orders"}); err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}

	_, err = queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		return errors.New("still offline")
	})
	if err != nil {
		t.Fatalf("expected first replay to persist retry state, got %v", err)
	}
	current = current.Add(2 * time.Second)

	report, err := queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		return errors.New("still offline")
	})
	if err != nil {
		t.Fatalf("expected second replay to persist dead-letter state, got %v", err)
	}
	if report.DeadLetters != 1 || report.Remaining != 1 {
		t.Fatalf("unexpected dead-letter report: %+v", report)
	}

	entries, err := queue.List()
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one dead-letter entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.State != MutationDead || entry.Attempts != 2 || !entry.NextAttemptAt.IsZero() {
		t.Fatalf("unexpected dead-letter entry: %+v", entry)
	}
}

func TestMutationQueueReplayCancellationKeepsRemainingEntries(t *testing.T) {
	installMockMutationQueueStorage(t)
	queue, err := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-cancel"})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/first", Kind: "first"}); err != nil {
		t.Fatalf("expected first enqueue to succeed, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/second", Kind: "second"}); err != nil {
		t.Fatalf("expected second enqueue to succeed, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	report, err := queue.Replay(ctx, func(ctx context.Context, mutation QueuedMutation) error {
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected replay cancellation, got %v", err)
	}
	if report.Succeeded != 1 || report.Remaining != 1 {
		t.Fatalf("unexpected cancellation report: %+v", report)
	}

	entries, err := queue.List()
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(entries) != 1 || entries[0].Kind != "second" {
		t.Fatalf("expected second entry to remain queued, got %+v", entries)
	}
}

func TestMutationQueueConflictWithoutHandlerMovesToDeadLetter(t *testing.T) {
	installMockMutationQueueStorage(t)
	queue, err := OpenMutationQueue(MutationQueueOptions{StorageKey: "offline-conflict-dead"})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/orders", Kind: "order.submit"}); err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}

	report, err := queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		return NewMutationConflict(errors.New("etag mismatch"), MutationConflict{Code: "etag_mismatch", LocalVersion: "1", RemoteVersion: "2", Message: "server revision is newer"})
	})
	if err != nil {
		t.Fatalf("expected replay to persist conflict dead-letter state, got %v", err)
	}
	if report.Conflicts != 1 || report.DeadLetters != 1 || report.Remaining != 1 {
		t.Fatalf("unexpected conflict report: %+v", report)
	}

	entries, err := queue.List()
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(entries) != 1 || entries[0].State != MutationDead {
		t.Fatalf("expected one dead-letter conflict entry, got %+v", entries)
	}
	if entries[0].LastError == "" || entries[0].Attempts != 1 {
		t.Fatalf("expected conflict details to be retained, got %+v", entries[0])
	}
}

func TestMutationConflictHelpersExposeStructuredConflict(t *testing.T) {
	err := NewMutationConflict(errors.New("etag mismatch"), MutationConflict{
		Code:          "etag_mismatch",
		LocalVersion:  "1",
		RemoteVersion: "2",
		Message:       "server revision is newer",
	})

	if !IsMutationConflict(err) {
		t.Fatal("expected IsMutationConflict to recognize wrapped conflict errors")
	}
	if IsMutationConflict(errors.New("plain error")) {
		t.Fatal("expected IsMutationConflict to reject non-conflict errors")
	}

	conflictErr, ok := AsMutationConflictError(err)
	if !ok || conflictErr == nil {
		t.Fatal("expected AsMutationConflictError to unwrap the structured conflict")
	}
	if conflictErr.Conflict.Code != "etag_mismatch" {
		t.Fatalf("expected conflict code to round-trip, got %+v", conflictErr.Conflict)
	}

	conflict, ok := GetMutationConflict(err)
	if !ok {
		t.Fatal("expected GetMutationConflict to expose conflict details")
	}
	if conflict.LocalVersion != "1" || conflict.RemoteVersion != "2" || conflict.Message != "server revision is newer" {
		t.Fatalf("unexpected conflict details: %+v", conflict)
	}

	if _, ok := GetMutationConflict(errors.New("plain error")); ok {
		t.Fatal("expected GetMutationConflict to reject non-conflict errors")
	}
}

func TestMutationQueueConflictHandlerCanRequeueResolvedMutation(t *testing.T) {
	installMockMutationQueueStorage(t)
	current := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	queue, err := OpenMutationQueue(MutationQueueOptions{
		StorageKey: "offline-conflict-resolve",
		Now:        func() time.Time { return current },
	})
	if err != nil {
		t.Fatalf("expected mutation queue to open, got %v", err)
	}
	if _, err := queue.Enqueue(MutationDraft{URL: "/api/orders", Kind: "order.submit", Metadata: map[string]string{"revision": "1"}}); err != nil {
		t.Fatalf("expected enqueue to succeed, got %v", err)
	}

	report, err := queue.ReplayWithOptions(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		if mutation.Metadata["resolved"] == "server-2" {
			return nil
		}
		return NewMutationConflict(errors.New("etag mismatch"), MutationConflict{Code: "etag_mismatch", LocalVersion: mutation.Metadata["revision"], RemoteVersion: "2", Message: "server revision is newer"})
	}, MutationReplayOptions{ConflictHandler: func(ctx context.Context, mutation QueuedMutation, conflict MutationConflict) (MutationConflictResolution, error) {
		return MutationConflictResolution{
			Action:  MutationResolutionReplace,
			Message: "rebased onto server revision 2",
			Draft: MutationDraft{
				Metadata: map[string]string{"revision": conflict.RemoteVersion, "resolved": "server-2"},
				Body:     map[string]any{"merge": "accepted"},
			},
		}, nil
	}})
	if err != nil {
		t.Fatalf("expected conflict resolution replay to succeed, got %v", err)
	}
	if report.Conflicts != 1 || report.Resolved != 1 || report.Remaining != 1 {
		t.Fatalf("unexpected conflict resolution report: %+v", report)
	}

	entries, err := queue.List()
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one requeued resolved entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.State != MutationQueued || entry.Attempts != 0 || entry.Metadata["resolved"] != "server-2" || entry.Metadata["revision"] != "2" {
		t.Fatalf("unexpected resolved entry: %+v", entry)
	}
	if entry.LastError != "rebased onto server revision 2" {
		t.Fatalf("expected resolution note to persist, got %+v", entry)
	}

	finalReport, err := queue.Replay(context.Background(), func(ctx context.Context, mutation QueuedMutation) error {
		if mutation.Metadata["resolved"] != "server-2" {
			t.Fatalf("expected rebased mutation to be replayed, got %+v", mutation)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected resolved replay to succeed, got %v", err)
	}
	if finalReport.Succeeded != 1 || finalReport.Remaining != 0 {
		t.Fatalf("unexpected final replay report: %+v", finalReport)
	}
}
