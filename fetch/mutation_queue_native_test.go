//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestFetchNativeMutationQueueHelpers(parseT *testing.T) {
	parseConflictErr := NewMutationConflict(nil, MutationConflict{
		Code:          " conflict ",
		Message:       " version mismatch ",
		LocalVersion:  " local ",
		RemoteVersion: " remote ",
		Fields:        map[string]string{"id": "42"},
	})
	if !IsMutationConflict(parseConflictErr) {
		parseT.Fatal("expected NewMutationConflict to produce a structured conflict error")
	}
	parseWrapped, parseOk := AsMutationConflictError(parseConflictErr)
	if !parseOk || parseWrapped.Conflict.Code != "conflict" || parseWrapped.Conflict.Message != "version mismatch" {
		parseT.Fatalf("unexpected structured conflict error: %+v ok=%t", parseWrapped, parseOk)
	}
	if parseWrapped.Unwrap() == nil {
		parseT.Fatal("expected structured conflict error to unwrap the inner error")
	}
	if parseConflict, parseOk2 := GetMutationConflict(parseConflictErr); !parseOk2 || parseConflict.LocalVersion != "local" || parseConflict.RemoteVersion != "remote" {
		parseT.Fatalf("unexpected structured conflict payload: %+v ok=%t", parseConflict, parseOk2)
	}

	parseNow := time.Date(2026, time.April, 6, 12, 30, 0, 0, time.UTC)
	parseResolved := resolveMutationQueueOptions([]MutationQueueOptions{
		{
			StorageKey:         " offline ",
			MaxAttempts:        7,
			BaseDelay:          time.Second,
			MaxDelay:           8 * time.Second,
			DeleteOnCorruption: true,
			Now:                func() time.Time { return parseNow },
		},
	})
	if parseResolved.StorageKey != "offline" || parseResolved.MaxAttempts != 7 || !parseResolved.DeleteOnCorruption {
		parseT.Fatalf("unexpected resolved mutation queue options: %+v", parseResolved)
	}

	parseQueue := MutationQueue{baseDelay: time.Second, maxDelay: 4 * time.Second}
	if parseDelay := parseQueue.retryDelay(1); parseDelay != time.Second {
		parseT.Fatalf("expected first retry delay to use base delay, got %s", parseDelay)
	}
	if parseDelay := parseQueue.retryDelay(4); parseDelay != 4*time.Second {
		parseT.Fatalf("expected retry delay to clamp to max delay, got %s", parseDelay)
	}

	if parseID := normalizeMutationID("", parseNow, 3); !strings.HasPrefix(parseID, "mutation-") {
		parseT.Fatalf("expected generated mutation id, got %q", parseID)
	}
	parseSource := map[string]string{"theme": "dark"}
	parseClone := cloneStringMap(parseSource)
	parseClone["theme"] = "light"
	if parseSource["theme"] != "dark" {
		parseT.Fatalf("expected cloneStringMap to copy values, got %#v", parseSource)
	}

	parseMerged := mergeResolvedMutation(
		QueuedMutation{ID: "m1", Kind: "draft", Method: "POST", URL: "/api/drafts", Attempts: 2},
		MutationDraft{Kind: "publish", Method: "patch", URL: "/api/posts/1", Metadata: map[string]string{"schema": "v2"}},
		parseNow,
		"resolved",
	)
	if parseMerged.Kind != "publish" || parseMerged.Method != "PATCH" || parseMerged.Attempts != 0 || parseMerged.LastError != "resolved" {
		parseT.Fatalf("unexpected merged mutation: %+v", parseMerged)
	}

	if _, parseErr := OpenMutationQueue(MutationQueueOptions{
		StoreResolver: func(parseCtx context.Context) (interop.PersistentStore, error) {
			_ = parseCtx
			return interop.PersistentStore{}, errors.New("open failed")
		},
	}); parseErr == nil {
		parseT.Fatal("expected OpenMutationQueue to surface store-open failures")
	}

	parseNowQueue := MutationQueue{}
	if parseNowQueue.currentTime().IsZero() {
		parseT.Fatal("expected currentTime to fall back to time.Now")
	}
	if normalizeMutationID(" explicit ", parseNow, 1) != "explicit" {
		parseT.Fatal("expected normalizeMutationID to preserve trimmed explicit ids")
	}
}

func TestFetchNativeMutationQueueStorageAndReplay(parseT *testing.T) {
	parseStore, parseData := buildFetchTestPersistentStore(parseT)
	parseNow := time.Date(2026, time.April, 6, 13, 0, 0, 0, time.UTC)

	var parseResolverCalls int
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey:  "offline",
		MaxAttempts: 2,
		BaseDelay:   time.Second,
		MaxDelay:    2 * time.Second,
		StoreResolver: func(parseCtx context.Context) (interop.PersistentStore, error) {
			_ = parseCtx
			parseResolverCalls++
			return parseStore, nil
		},
		Now: func() time.Time { return parseNow },
	})
	if parseErr != nil || parseResolverCalls != 1 {
		parseT.Fatalf("expected mutation queue to open through the injected store resolver, calls=%d err=%v", parseResolverCalls, parseErr)
	}

	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{}); parseErr2 == nil {
		parseT.Fatal("expected enqueue to reject empty mutation URLs")
	}

	parseEntry, parseErr := parseQueue.Enqueue(MutationDraft{
		Kind:     "post.publish",
		Method:   "post",
		URL:      "/api/posts",
		DedupKey: "post:1",
		Metadata: map[string]string{"schema": "v1"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected enqueue success, got %v", parseErr)
	}
	parseDuplicate, parseErr := parseQueue.Enqueue(MutationDraft{
		URL:      "/api/posts",
		DedupKey: "post:1",
	})
	if parseErr != nil || parseDuplicate.ID != parseEntry.ID {
		parseT.Fatalf("expected enqueue deduplication, duplicate=%+v err=%v", parseDuplicate, parseErr)
	}

	if _, parseErr2 := parseQueue.ReplayWithOptions(context.Background(), nil); parseErr2 == nil {
		parseT.Fatal("expected ReplayWithOptions to reject a nil executor")
	}

	parseReport, parseErr := parseQueue.ReplayWithOptions(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		_ = parseCtx
		_ = parseMutation
		return NewMutationConflict(errors.New("conflict"), MutationConflict{Code: "409", Message: "version mismatch"})
	}, MutationReplayOptions{
		ConflictHandler: func(parseCtx context.Context, parseMutation QueuedMutation, parseConflict MutationConflict) (MutationConflictResolution, error) {
			_ = parseCtx
			_ = parseMutation
			_ = parseConflict
			return MutationConflictResolution{Action: MutationResolutionRemove}, nil
		},
	})
	if parseErr != nil || parseReport.Conflicts != 1 || parseReport.Resolved != 1 || parseReport.Remaining != 0 {
		parseT.Fatalf("expected conflict removal replay success, report=%+v err=%v", parseReport, parseErr)
	}

	parseEntry, parseErr = parseQueue.Enqueue(MutationDraft{URL: "/api/posts", Kind: "post.publish"})
	if parseErr != nil {
		parseT.Fatalf("expected second enqueue success, got %v", parseErr)
	}
	parseReport, parseErr = parseQueue.ReplayWithOptions(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		_ = parseCtx
		_ = parseMutation
		return NewMutationConflict(errors.New("conflict"), MutationConflict{Code: "409"})
	}, MutationReplayOptions{
		ConflictHandler: func(parseCtx context.Context, parseMutation QueuedMutation, parseConflict MutationConflict) (MutationConflictResolution, error) {
			_ = parseCtx
			_ = parseMutation
			_ = parseConflict
			return MutationConflictResolution{Action: MutationResolutionDead, Message: "manual review"}, nil
		},
	})
	if parseErr != nil || parseReport.DeadLetters != 1 || parseReport.Remaining != 1 {
		parseT.Fatalf("expected dead-letter replay result, report=%+v err=%v", parseReport, parseErr)
	}
	if parseEntries, parseErr2 := parseQueue.List(); parseErr2 != nil || len(parseEntries) != 1 || parseEntries[0].ID != parseEntry.ID || parseEntries[0].State != MutationDead {
		parseT.Fatalf("expected dead-letter mutation to remain stored, entries=%+v err=%v", parseEntries, parseErr2)
	}
	if parseErr2 := parseQueue.Remove(" "); parseErr2 != nil {
		parseT.Fatalf("expected blank Remove call to no-op, got %v", parseErr2)
	}
	if parseErr2 := parseQueue.Remove(parseEntry.ID); parseErr2 != nil {
		parseT.Fatalf("expected Remove success, got %v", parseErr2)
	}
	if parseEntries, parseErr2 := parseQueue.List(); parseErr2 != nil || len(parseEntries) != 0 {
		parseT.Fatalf("expected Remove to empty the queue, entries=%+v err=%v", parseEntries, parseErr2)
	}

	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/posts"}); parseErr2 != nil {
		parseT.Fatalf("expected third enqueue success, got %v", parseErr2)
	}
	if _, parseErr2 := parseQueue.ReplayWithOptions(context.Background(), func(parseCtx context.Context, parseMutation QueuedMutation) error {
		_ = parseCtx
		_ = parseMutation
		return NewMutationConflict(errors.New("conflict"), MutationConflict{Code: "409"})
	}, MutationReplayOptions{
		ConflictHandler: func(parseCtx context.Context, parseMutation QueuedMutation, parseConflict MutationConflict) (MutationConflictResolution, error) {
			_ = parseCtx
			_ = parseMutation
			_ = parseConflict
			return MutationConflictResolution{Action: MutationResolutionAction("unsupported")}, nil
		},
	}); parseErr2 == nil {
		parseT.Fatal("expected unsupported conflict action to fail replay")
	}

	if parseErr2 := parseQueue.Clear(); parseErr2 != nil {
		parseT.Fatalf("expected Clear success, got %v", parseErr2)
	}
	if _, parseOk := parseData["offline"]; parseOk {
		parseT.Fatalf("expected Clear to remove stored queue data, data=%#v", parseData)
	}

	if _, parseErr2 := parseQueue.Enqueue(MutationDraft{URL: "/api/posts", Kind: "post.publish"}); parseErr2 != nil {
		parseT.Fatalf("expected fourth enqueue success, got %v", parseErr2)
	}
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseReport, parseErr = parseQueue.Replay(parseCtx, func(parseReplayCtx context.Context, parseMutation QueuedMutation) error {
		_ = parseReplayCtx
		_ = parseMutation
		return nil
	})
	if !errors.Is(parseErr, context.Canceled) || parseReport.Remaining != 1 {
		parseT.Fatalf("expected canceled replay to preserve the queue, report=%+v err=%v", parseReport, parseErr)
	}

	parseData["offline"] = "{"
	if _, parseErr2 := parseQueue.List(); parseErr2 == nil {
		parseT.Fatal("expected corrupted stored queue JSON to fail List")
	}
}
