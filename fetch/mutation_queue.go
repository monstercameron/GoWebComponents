//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
)

const defaultMutationQueueStorageKey = "__gwc_mutation_queue__"

type MutationState string

const (
	MutationQueued   MutationState = "queued"
	MutationRetrying MutationState = "retrying"
	MutationDead     MutationState = "dead"
)

// MutationDraft describes one app-owned write to persist for later replay.
type MutationDraft struct {
	ID       string            `json:"id,omitempty"`
	Kind     string            `json:"kind,omitempty"`
	DedupKey string            `json:"dedupKey,omitempty"`
	Method   string            `json:"method,omitempty"`
	URL      string            `json:"url,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     interface{}       `json:"body,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// QueuedMutation is the persisted replay record stored in the mutation queue.
type QueuedMutation struct {
	ID            string            `json:"id,omitempty"`
	Kind          string            `json:"kind,omitempty"`
	DedupKey      string            `json:"dedupKey,omitempty"`
	Method        string            `json:"method,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          interface{}       `json:"body,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	State         MutationState     `json:"state,omitempty"`
	Attempts      int               `json:"attempts,omitempty"`
	MaxAttempts   int               `json:"maxAttempts,omitempty"`
	CreatedAt     time.Time         `json:"createdAt,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt,omitempty"`
	NextAttemptAt time.Time         `json:"nextAttemptAt,omitempty"`
	LastError     string            `json:"lastError,omitempty"`
}

// MutationQueueOptions configures persistent queue behavior.
type MutationQueueOptions struct {
	StorageKey      string
	MaxAttempts     int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	StorageResolver func() (interop.Storage, error)
	Now             func() time.Time
}

// MutationReplayReport summarizes one replay pass across queued entries.
type MutationReplayReport struct {
	Succeeded   int
	Deferred    int
	Retried     int
	DeadLetters int
	Remaining   int
}

// MutationExecutor performs the authoritative replay for one queued mutation.
type MutationExecutor func(context.Context, QueuedMutation) error

// MutationQueue persists writes locally and replays them later through an app-owned executor.
type MutationQueue struct {
	storageKey  string
	storage     interop.Storage
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	now         func() time.Time
}

// OpenMutationQueue opens the persistent mutation queue backed by browser storage.
func OpenMutationQueue(options ...MutationQueueOptions) (MutationQueue, error) {
	cfg := resolveMutationQueueOptions(options)

	resolver := cfg.StorageResolver
	if resolver == nil {
		resolver = interop.LocalStorage
	}
	storage, err := resolver()
	if err != nil {
		return MutationQueue{}, err
	}

	return MutationQueue{
		storageKey:  cfg.StorageKey,
		storage:     storage,
		maxAttempts: cfg.MaxAttempts,
		baseDelay:   cfg.BaseDelay,
		maxDelay:    cfg.MaxDelay,
		now:         cfg.Now,
	}, nil
}

// Enqueue stores a write for later replay, suppressing duplicates that share the same dedup key.
func (q MutationQueue) Enqueue(draft MutationDraft) (QueuedMutation, error) {
	if strings.TrimSpace(draft.URL) == "" {
		return QueuedMutation{}, fmt.Errorf("fetch mutation queue requires a non-empty URL")
	}
	method := strings.ToUpper(strings.TrimSpace(draft.Method))
	if method == "" {
		method = "POST"
	}

	entries, err := q.load()
	if err != nil {
		return QueuedMutation{}, err
	}
	for _, existing := range entries {
		if existing.State == MutationDead {
			continue
		}
		if draft.DedupKey != "" && existing.DedupKey == draft.DedupKey {
			return existing, nil
		}
	}

	now := q.currentTime()
	entry := QueuedMutation{
		ID:          normalizeMutationID(draft.ID, now, len(entries)+1),
		Kind:        strings.TrimSpace(draft.Kind),
		DedupKey:    strings.TrimSpace(draft.DedupKey),
		Method:      method,
		URL:         strings.TrimSpace(draft.URL),
		Headers:     cloneStringMap(draft.Headers),
		Body:        draft.Body,
		Metadata:    cloneStringMap(draft.Metadata),
		State:       MutationQueued,
		MaxAttempts: q.maxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	entries = append(entries, entry)
	if err := q.save(entries); err != nil {
		return QueuedMutation{}, err
	}
	runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticInformational, "mutation queued for replay", "", map[string]string{
		"id":   entry.ID,
		"kind": entry.Kind,
		"url":  entry.URL,
	})
	return entry, nil
}

// List returns all currently persisted queue entries in creation order.
func (q MutationQueue) List() ([]QueuedMutation, error) {
	entries, err := q.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

// Remove deletes one queue entry by id.
func (q MutationQueue) Remove(id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil
	}
	entries, err := q.load()
	if err != nil {
		return err
	}
	filtered := make([]QueuedMutation, 0, len(entries))
	for _, entry := range entries {
		if entry.ID != trimmed {
			filtered = append(filtered, entry)
		}
	}
	return q.save(filtered)
}

// Clear removes every persisted queue entry.
func (q MutationQueue) Clear() error {
	return q.storage.RemoveItem(q.storageKey)
}

// Replay replays due entries through the provided executor and persists the updated queue state.
func (q MutationQueue) Replay(ctx context.Context, executor MutationExecutor) (MutationReplayReport, error) {
	if executor == nil {
		return MutationReplayReport{}, fmt.Errorf("fetch mutation queue requires an executor")
	}

	entries, err := q.load()
	if err != nil {
		return MutationReplayReport{}, err
	}
	report := MutationReplayReport{}
	now := q.currentTime()
	remaining := make([]QueuedMutation, 0, len(entries))

	for index, entry := range entries {
		if ctx != nil {
			select {
			case <-ctx.Done():
				remaining = append(remaining, entries[index:]...)
				report.Remaining = len(remaining)
				if saveErr := q.save(remaining); saveErr != nil {
					return report, saveErr
				}
				return report, ctx.Err()
			default:
			}
		}

		if entry.State == MutationDead {
			report.DeadLetters++
			remaining = append(remaining, entry)
			continue
		}
		if !entry.NextAttemptAt.IsZero() && entry.NextAttemptAt.After(now) {
			report.Deferred++
			remaining = append(remaining, entry)
			continue
		}

		if err := executor(ctx, entry); err != nil {
			entry.Attempts++
			entry.UpdatedAt = now
			entry.LastError = err.Error()
			if entry.Attempts >= entry.MaxAttempts {
				entry.State = MutationDead
				entry.NextAttemptAt = time.Time{}
				report.DeadLetters++
				runtime.ReportLogWithFields("fetch", runtime.LogError, runtime.DiagnosticCorrectness, "mutation replay moved to dead-letter state", "", map[string]string{
					"id":      entry.ID,
					"kind":    entry.Kind,
					"url":     entry.URL,
					"attempt": fmt.Sprintf("%d", entry.Attempts),
					"error":   entry.LastError,
				})
			} else {
				entry.State = MutationRetrying
				entry.NextAttemptAt = now.Add(q.retryDelay(entry.Attempts))
				report.Retried++
				runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "mutation replay scheduled for retry", "", map[string]string{
					"id":      entry.ID,
					"kind":    entry.Kind,
					"url":     entry.URL,
					"attempt": fmt.Sprintf("%d", entry.Attempts),
					"error":   entry.LastError,
				})
			}
			remaining = append(remaining, entry)
			continue
		}

		report.Succeeded++
		runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticInformational, "mutation replay succeeded", "", map[string]string{
			"id":   entry.ID,
			"kind": entry.Kind,
			"url":  entry.URL,
		})
	}

	report.Remaining = len(remaining)
	if err := q.save(remaining); err != nil {
		return report, err
	}
	return report, nil
}

func (q MutationQueue) load() ([]QueuedMutation, error) {
	value, ok, err := q.storage.GetItem(q.storageKey)
	if err != nil {
		return nil, err
	}
	if !ok || strings.TrimSpace(value) == "" {
		return []QueuedMutation{}, nil
	}
	var entries []QueuedMutation
	if err := json.Unmarshal([]byte(value), &entries); err != nil {
		return nil, err
	}
	for index := range entries {
		if entries[index].MaxAttempts <= 0 {
			entries[index].MaxAttempts = q.maxAttempts
		}
		if strings.TrimSpace(entries[index].Method) == "" {
			entries[index].Method = "POST"
		}
		if entries[index].State == "" {
			if entries[index].Attempts > 0 {
				entries[index].State = MutationRetrying
			} else {
				entries[index].State = MutationQueued
			}
		}
	}
	return entries, nil
}

func (q MutationQueue) save(entries []QueuedMutation) error {
	if len(entries) == 0 {
		return q.storage.RemoveItem(q.storageKey)
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return q.storage.SetItem(q.storageKey, string(data))
}

func (q MutationQueue) currentTime() time.Time {
	if q.now != nil {
		return q.now()
	}
	return time.Now()
}

func (q MutationQueue) retryDelay(attempt int) time.Duration {
	delay := q.baseDelay
	if delay <= 0 {
		delay = 2 * time.Second
	}
	maxDelay := q.maxDelay
	if maxDelay <= 0 {
		maxDelay = 2 * time.Minute
	}
	for i := 1; i < attempt; i++ {
		if delay >= maxDelay/2 {
			return maxDelay
		}
		delay *= 2
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func resolveMutationQueueOptions(options []MutationQueueOptions) MutationQueueOptions {
	cfg := MutationQueueOptions{
		StorageKey:  defaultMutationQueueStorageKey,
		MaxAttempts: 5,
		BaseDelay:   2 * time.Second,
		MaxDelay:    2 * time.Minute,
		Now:         time.Now,
	}
	if len(options) == 0 {
		return cfg
	}
	overrides := options[0]
	if strings.TrimSpace(overrides.StorageKey) != "" {
		cfg.StorageKey = strings.TrimSpace(overrides.StorageKey)
	}
	if overrides.MaxAttempts > 0 {
		cfg.MaxAttempts = overrides.MaxAttempts
	}
	if overrides.BaseDelay > 0 {
		cfg.BaseDelay = overrides.BaseDelay
	}
	if overrides.MaxDelay > 0 {
		cfg.MaxDelay = overrides.MaxDelay
	}
	if overrides.StorageResolver != nil {
		cfg.StorageResolver = overrides.StorageResolver
	}
	if overrides.Now != nil {
		cfg.Now = overrides.Now
	}
	return cfg
}

func normalizeMutationID(id string, now time.Time, ordinal int) string {
	trimmed := strings.TrimSpace(id)
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("mutation-%d-%d", now.UnixNano(), ordinal)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}
