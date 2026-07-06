package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// mutationQueueLocks serializes each storage key's load-mutate-save sequence.
// MutationQueue is a value type shared by copy, so a mutex FIELD would give each
// copy its own lock and protect nothing; keying on the storage string makes the
// lock shared by every queue value (and every copy) that targets the same
// persisted list — without it two concurrent Enqueue/Remove/Replay calls could
// each load the same snapshot and the second save() would silently drop the
// first writer's durable mutation.
var mutationQueueLocks sync.Map // storageKey -> *sync.Mutex

func (parseQ MutationQueue) lock() *sync.Mutex {
	parseActual, _ := mutationQueueLocks.LoadOrStore(parseQ.storageKey, &sync.Mutex{})
	return parseActual.(*sync.Mutex)
}

const defaultMutationQueueStorageKey = "__gwc_mutation_queue__"
const defaultMutationQueueStoreName = "mutation-queue"

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
	Body     any               `json:"body,omitempty"`
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
	Body          any               `json:"body,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	State         MutationState     `json:"state,omitempty"`
	Attempts      int               `json:"attempts,omitempty"`
	MaxAttempts   int               `json:"maxAttempts,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	NextAttemptAt time.Time         `json:"nextAttemptAt"`
	LastError     string            `json:"lastError,omitempty"`
}

// MutationQueueOptions configures persistent queue behavior.
type MutationQueueOptions struct {
	StorageKey         string
	MaxAttempts        int
	BaseDelay          time.Duration
	MaxDelay           time.Duration
	DeleteOnCorruption bool
	StoreResolver      func(context.Context) (interop.PersistentStore, error)
	StorageResolver    func() (interop.Storage, error)
	Now                func() time.Time
}

type MutationConflict struct {
	Code          string            `json:"code,omitempty"`
	Message       string            `json:"message,omitempty"`
	LocalVersion  string            `json:"localVersion,omitempty"`
	RemoteVersion string            `json:"remoteVersion,omitempty"`
	Fields        map[string]string `json:"fields,omitempty"`
}

type MutationConflictError struct {
	Conflict MutationConflict
	Err      error
}

// Error is a core package helper.
func (parseE *MutationConflictError) Error() string {
	if parseE == nil {
		return "mutation conflict"
	}
	parseParts := []string{"mutation conflict"}
	if parseMessage := strings.TrimSpace(parseE.Conflict.Message); parseMessage != "" {
		parseParts = append(parseParts, parseMessage)
	}
	if parseE.Err != nil {
		parseParts = append(parseParts, parseE.Err.Error())
	}
	return strings.Join(parseParts, ": ")
}

// Unwrap is a core package helper.
func (parseE *MutationConflictError) Unwrap() error {
	if parseE == nil {
		return nil
	}
	return parseE.Err
}

// NewMutationConflict wraps err with structured conflict metadata, returning a *MutationConflictError.
func NewMutationConflict(parseErr error, parseConflict MutationConflict) error {
	parseConflict.Code = strings.TrimSpace(parseConflict.Code)
	parseConflict.Message = strings.TrimSpace(parseConflict.Message)
	parseConflict.LocalVersion = strings.TrimSpace(parseConflict.LocalVersion)
	parseConflict.RemoteVersion = strings.TrimSpace(parseConflict.RemoteVersion)
	parseConflict.Fields = cloneStringMap(parseConflict.Fields)
	if parseErr == nil {
		parseErr = errors.New("mutation conflict")
	}
	return &MutationConflictError{Conflict: parseConflict, Err: parseErr}
}

// IsMutationConflict reports whether err unwraps to a MutationConflictError.
func IsMutationConflict(parseErr error) bool {
	_, parseOk := AsMutationConflictError(parseErr)
	return parseOk
}

// AsMutationConflictError unwraps err into the structured MutationConflictError form.
func AsMutationConflictError(parseErr error) (*MutationConflictError, bool) {
	var parseConflictErr *MutationConflictError
	if !errors.As(parseErr, &parseConflictErr) {
		return nil, false
	}
	return parseConflictErr, true
}

// GetMutationConflict returns the structured conflict details carried by err.
func GetMutationConflict(parseErr error) (MutationConflict, bool) {
	parseConflictErr, parseOk := AsMutationConflictError(parseErr)
	if !parseOk || parseConflictErr == nil {
		return MutationConflict{}, false
	}
	return parseConflictErr.Conflict, true
}

type MutationResolutionAction string

const (
	MutationResolutionRetry   MutationResolutionAction = "retry"
	MutationResolutionDead    MutationResolutionAction = "dead"
	MutationResolutionRemove  MutationResolutionAction = "remove"
	MutationResolutionReplace MutationResolutionAction = "replace"
)

type MutationConflictResolution struct {
	Action  MutationResolutionAction
	Draft   MutationDraft
	Message string
}

type MutationConflictHandler func(context.Context, QueuedMutation, MutationConflict) (MutationConflictResolution, error)

type MutationReplayOptions struct {
	ConflictHandler MutationConflictHandler
}

// MutationReplayReport summarizes one replay pass across queued entries.
type MutationReplayReport struct {
	Succeeded   int
	Deferred    int
	Retried     int
	DeadLetters int
	Conflicts   int
	Resolved    int
	Remaining   int
}

// MutationExecutor performs the authoritative replay for one queued mutation.
type MutationExecutor func(context.Context, QueuedMutation) error

// MutationQueue persists writes locally and replays them later through an app-owned executor.
type MutationQueue struct {
	storageKey  string
	store       interop.PersistentStore
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	now         func() time.Time
}

// OpenMutationQueue opens the persistent mutation queue backed by browser storage.
func OpenMutationQueue(parseOptions ...MutationQueueOptions) (MutationQueue, error) {
	parseCfg := resolveMutationQueueOptions(parseOptions)

	parseResolver := parseCfg.StoreResolver
	if parseResolver == nil {
		parseFallbackResolver := parseCfg.StorageResolver
		if parseFallbackResolver == nil {
			parseFallbackResolver = interop.LocalStorage
		}
		parseResolver = func(parseCtx context.Context) (interop.PersistentStore, error) {
			return interop.OpenPersistentStore(parseCtx, interop.PersistentStoreOptions{
				Name:               defaultMutationQueueStoreName,
				DeleteOnCorruption: parseCfg.DeleteOnCorruption,
				FallbackResolver:   parseFallbackResolver,
				FallbackBackend:    "localStorage",
			})
		}
	}
	store, parseErr := parseResolver(context.Background())
	if parseErr != nil {
		return MutationQueue{}, parseErr
	}

	return MutationQueue{
		storageKey:  parseCfg.StorageKey,
		store:       store,
		maxAttempts: parseCfg.MaxAttempts,
		baseDelay:   parseCfg.BaseDelay,
		maxDelay:    parseCfg.MaxDelay,
		now:         parseCfg.Now,
	}, nil
}

// Enqueue stores a write for later replay, suppressing duplicates that share the same dedup key.
func (parseQ MutationQueue) Enqueue(parseDraft MutationDraft) (QueuedMutation, error) {
	if strings.TrimSpace(parseDraft.URL) == "" {
		return QueuedMutation{}, fmt.Errorf("fetch mutation queue requires a non-empty URL")
	}
	parseMethod := strings.ToUpper(strings.TrimSpace(parseDraft.Method))
	if parseMethod == "" {
		parseMethod = "POST"
	}

	parseMu := parseQ.lock()
	parseMu.Lock()
	defer parseMu.Unlock()

	parseEntries, parseErr := parseQ.load()
	if parseErr != nil {
		return QueuedMutation{}, parseErr
	}
	for _, parseExisting := range parseEntries {
		if parseExisting.State == MutationDead {
			continue
		}
		if parseDraft.DedupKey != "" && parseExisting.DedupKey == parseDraft.DedupKey {
			return parseExisting, nil
		}
	}

	parseNow := parseQ.currentTime()
	parseEntry := QueuedMutation{
		ID:          normalizeMutationID(parseDraft.ID, parseNow, len(parseEntries)+1),
		Kind:        strings.TrimSpace(parseDraft.Kind),
		DedupKey:    strings.TrimSpace(parseDraft.DedupKey),
		Method:      parseMethod,
		URL:         strings.TrimSpace(parseDraft.URL),
		Headers:     cloneStringMap(parseDraft.Headers),
		Body:        parseDraft.Body,
		Metadata:    cloneStringMap(parseDraft.Metadata),
		State:       MutationQueued,
		MaxAttempts: parseQ.maxAttempts,
		CreatedAt:   parseNow,
		UpdatedAt:   parseNow,
	}

	parseEntries = append(parseEntries, parseEntry)
	if parseErr2 := parseQ.save(parseEntries); parseErr2 != nil {
		return QueuedMutation{}, parseErr2
	}
	runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticInformational, "mutation queued for replay", "", map[string]string{
		"id":   parseEntry.ID,
		"kind": parseEntry.Kind,
		"url":  parseEntry.URL,
	})
	return parseEntry, nil
}

// List returns all currently persisted queue entries in creation order.
func (parseQ MutationQueue) List() ([]QueuedMutation, error) {
	parseEntries, parseErr := parseQ.load()
	if parseErr != nil {
		return nil, parseErr
	}
	sort.SliceStable(parseEntries, func(parseI, parseJ int) bool {
		return parseEntries[parseI].CreatedAt.Before(parseEntries[parseJ].CreatedAt)
	})
	return parseEntries, nil
}

// Remove deletes one queue entry by id.
func (parseQ MutationQueue) Remove(parseId string) error {
	parseTrimmed := strings.TrimSpace(parseId)
	if parseTrimmed == "" {
		return nil
	}
	parseMu := parseQ.lock()
	parseMu.Lock()
	defer parseMu.Unlock()
	parseEntries, parseErr := parseQ.load()
	if parseErr != nil {
		return parseErr
	}
	parseFiltered := make([]QueuedMutation, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		if parseEntry.ID != parseTrimmed {
			parseFiltered = append(parseFiltered, parseEntry)
		}
	}
	return parseQ.save(parseFiltered)
}

// Clear removes every persisted queue entry.
func (parseQ MutationQueue) Clear() error {
	return parseQ.store.RemoveItem(context.Background(), parseQ.storageKey)
}

// Replay replays due entries through the provided executor and persists the updated queue state.
func (parseQ MutationQueue) Replay(parseCtx context.Context, parseExecutor MutationExecutor) (MutationReplayReport, error) {
	return parseQ.ReplayWithOptions(parseCtx, parseExecutor)
}

// ReplayWithOptions replays due entries and applies optional conflict-resolution policy.
func (parseQ MutationQueue) ReplayWithOptions(parseCtx context.Context, parseExecutor MutationExecutor, parseOptions ...MutationReplayOptions) (MutationReplayReport, error) {
	if parseExecutor == nil {
		return MutationReplayReport{}, fmt.Errorf("fetch mutation queue requires an executor")
	}
	parseReplayOptions := MutationReplayOptions{}
	if len(parseOptions) > 0 {
		parseReplayOptions = parseOptions[0]
	}

	// Replay is exclusive for its storage key: it holds the lock across the
	// whole load-execute-save so a concurrent Enqueue/Remove can't interleave a
	// stale snapshot save. Executors do network I/O, so this serializes replays
	// (intended for a durable queue) rather than optimizing throughput.
	parseMu := parseQ.lock()
	parseMu.Lock()
	defer parseMu.Unlock()

	parseEntries, parseErr := parseQ.load()
	if parseErr != nil {
		return MutationReplayReport{}, parseErr
	}
	parseReport := MutationReplayReport{}
	parseNow := parseQ.currentTime()
	parseRemaining := make([]QueuedMutation, 0, len(parseEntries))

	for parseIndex, parseEntry := range parseEntries {
		if parseCtx != nil {
			select {
			case <-parseCtx.Done():
				parseRemaining = append(parseRemaining, parseEntries[parseIndex:]...)
				parseReport.Remaining = len(parseRemaining)
				if parseSaveErr := parseQ.save(parseRemaining); parseSaveErr != nil {
					return parseReport, parseSaveErr
				}
				return parseReport, parseCtx.Err()
			default:
			}
		}

		if parseEntry.State == MutationDead {
			parseReport.DeadLetters++
			parseRemaining = append(parseRemaining, parseEntry)
			continue
		}
		if !parseEntry.NextAttemptAt.IsZero() && parseEntry.NextAttemptAt.After(parseNow) {
			parseReport.Deferred++
			parseRemaining = append(parseRemaining, parseEntry)
			continue
		}

		if parseErr2 := parseExecutor(parseCtx, parseEntry); parseErr2 != nil {
			if parseConflictErr, parseOk := AsMutationConflictError(parseErr2); parseOk {
				parseReport.Conflicts++
				parseResolved, parseHandled, parseResolveErr := parseQ.handleConflict(parseCtx, parseEntry, parseConflictErr, parseReplayOptions.ConflictHandler, parseNow)
				if parseResolveErr != nil {
					return parseReport, parseResolveErr
				}
				if parseHandled {
					switch parseResolved.State {
					case MutationDead:
						parseReport.DeadLetters++
						parseRemaining = append(parseRemaining, parseResolved)
					case MutationQueued, MutationRetrying:
						parseReport.Resolved++
						parseRemaining = append(parseRemaining, parseResolved)
					default:
						parseReport.Resolved++
					}
					continue
				}
			}
			parseEntry.Attempts++
			parseEntry.UpdatedAt = parseNow
			parseEntry.LastError = parseErr2.Error()
			if parseEntry.Attempts >= parseEntry.MaxAttempts {
				parseEntry.State = MutationDead
				parseEntry.NextAttemptAt = time.Time{}
				parseReport.DeadLetters++
				runtime.ReportLogWithFields("fetch", runtime.LogError, runtime.DiagnosticCorrectness, "mutation replay moved to dead-letter state", "", map[string]string{
					"id":      parseEntry.ID,
					"kind":    parseEntry.Kind,
					"url":     parseEntry.URL,
					"attempt": fmt.Sprintf("%d", parseEntry.Attempts),
					"error":   parseEntry.LastError,
				})
			} else {
				parseEntry.State = MutationRetrying
				parseEntry.NextAttemptAt = parseNow.Add(parseQ.retryDelay(parseEntry.Attempts))
				parseReport.Retried++
				runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "mutation replay scheduled for retry", "", map[string]string{
					"id":      parseEntry.ID,
					"kind":    parseEntry.Kind,
					"url":     parseEntry.URL,
					"attempt": fmt.Sprintf("%d", parseEntry.Attempts),
					"error":   parseEntry.LastError,
				})
			}
			parseRemaining = append(parseRemaining, parseEntry)
			continue
		}

		parseReport.Succeeded++
		runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticInformational, "mutation replay succeeded", "", map[string]string{
			"id":   parseEntry.ID,
			"kind": parseEntry.Kind,
			"url":  parseEntry.URL,
		})
	}

	parseReport.Remaining = len(parseRemaining)
	if parseErr3 := parseQ.save(parseRemaining); parseErr3 != nil {
		return parseReport, parseErr3
	}
	return parseReport, nil
}

// handleConflict is a core package helper.
func (parseQ MutationQueue) handleConflict(parseCtx context.Context, parseEntry QueuedMutation, parseConflictErr *MutationConflictError, parseHandler MutationConflictHandler, parseNow time.Time) (QueuedMutation, bool, error) {
	if parseHandler == nil {
		parseEntry.Attempts++
		parseEntry.State = MutationDead
		parseEntry.UpdatedAt = parseNow
		parseEntry.NextAttemptAt = time.Time{}
		parseEntry.LastError = parseConflictErr.Error()
		runtime.ReportLogWithFields("fetch", runtime.LogError, runtime.DiagnosticCorrectness, "mutation replay requires conflict resolution", "", map[string]string{
			"id":             parseEntry.ID,
			"kind":           parseEntry.Kind,
			"url":            parseEntry.URL,
			"conflict_code":  parseConflictErr.Conflict.Code,
			"local_version":  parseConflictErr.Conflict.LocalVersion,
			"remote_version": parseConflictErr.Conflict.RemoteVersion,
		})
		return parseEntry, true, nil
	}
	parseResolution, parseErr := parseHandler(parseCtx, parseEntry, parseConflictErr.Conflict)
	if parseErr != nil {
		return QueuedMutation{}, false, parseErr
	}
	parseMessage := strings.TrimSpace(parseResolution.Message)
	if parseMessage == "" {
		parseMessage = parseConflictErr.Error()
	}
	switch parseResolution.Action {
	case MutationResolutionRemove:
		runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticRecovered, "mutation conflict resolved by removal", "", map[string]string{
			"id":   parseEntry.ID,
			"kind": parseEntry.Kind,
			"url":  parseEntry.URL,
		})
		return QueuedMutation{}, true, nil
	case MutationResolutionReplace:
		parseResolved := mergeResolvedMutation(parseEntry, parseResolution.Draft, parseNow, parseMessage)
		runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticRecovered, "mutation conflict resolved by requeue", "", map[string]string{
			"id":             parseResolved.ID,
			"kind":           parseResolved.Kind,
			"url":            parseResolved.URL,
			"conflict_code":  parseConflictErr.Conflict.Code,
			"local_version":  parseConflictErr.Conflict.LocalVersion,
			"remote_version": parseConflictErr.Conflict.RemoteVersion,
		})
		return parseResolved, true, nil
	case MutationResolutionDead:
		parseEntry.Attempts++
		parseEntry.State = MutationDead
		parseEntry.UpdatedAt = parseNow
		parseEntry.NextAttemptAt = time.Time{}
		parseEntry.LastError = parseMessage
		runtime.ReportLogWithFields("fetch", runtime.LogError, runtime.DiagnosticCorrectness, "mutation conflict moved to dead-letter state", "", map[string]string{
			"id":             parseEntry.ID,
			"kind":           parseEntry.Kind,
			"url":            parseEntry.URL,
			"conflict_code":  parseConflictErr.Conflict.Code,
			"local_version":  parseConflictErr.Conflict.LocalVersion,
			"remote_version": parseConflictErr.Conflict.RemoteVersion,
		})
		return parseEntry, true, nil
	case MutationResolutionRetry, "":
		return QueuedMutation{}, false, nil
	default:
		return QueuedMutation{}, false, fmt.Errorf("unsupported mutation conflict resolution action %q", parseResolution.Action)
	}
}

// load is a core package helper.
func (parseQ MutationQueue) load() ([]QueuedMutation, error) {
	parseValue, parseOk, parseErr := parseQ.store.GetItem(context.Background(), parseQ.storageKey)
	if parseErr != nil {
		return nil, parseErr
	}
	if !parseOk || strings.TrimSpace(parseValue) == "" {
		return []QueuedMutation{}, nil
	}
	var parseEntries []QueuedMutation
	if parseErr2 := json.Unmarshal([]byte(parseValue), &parseEntries); parseErr2 != nil {
		return nil, parseErr2
	}
	for parseIndex := range parseEntries {
		if parseEntries[parseIndex].MaxAttempts <= 0 {
			parseEntries[parseIndex].MaxAttempts = parseQ.maxAttempts
		}
		if strings.TrimSpace(parseEntries[parseIndex].Method) == "" {
			parseEntries[parseIndex].Method = "POST"
		}
		if parseEntries[parseIndex].State == "" {
			if parseEntries[parseIndex].Attempts > 0 {
				parseEntries[parseIndex].State = MutationRetrying
			} else {
				parseEntries[parseIndex].State = MutationQueued
			}
		}
	}
	return parseEntries, nil
}

// save is a core package helper.
func (parseQ MutationQueue) save(parseEntries []QueuedMutation) error {
	if len(parseEntries) == 0 {
		return parseQ.store.RemoveItem(context.Background(), parseQ.storageKey)
	}
	parseData, parseErr := json.Marshal(parseEntries)
	if parseErr != nil {
		return parseErr
	}
	return parseQ.store.SetItem(context.Background(), parseQ.storageKey, string(parseData))
}

// currentTime is a core package helper.
func (parseQ MutationQueue) currentTime() time.Time {
	if parseQ.now != nil {
		return parseQ.now()
	}
	return time.Now()
}

// retryDelay is a core package helper.
func (parseQ MutationQueue) retryDelay(parseAttempt int) time.Duration {
	parseDelay := parseQ.baseDelay
	if parseDelay <= 0 {
		parseDelay = 2 * time.Second
	}
	parseMaxDelay := parseQ.maxDelay
	if parseMaxDelay <= 0 {
		parseMaxDelay = 2 * time.Minute
	}
	for parseI := 1; parseI < parseAttempt; parseI++ {
		if parseDelay >= parseMaxDelay/2 {
			return parseMaxDelay
		}
		parseDelay *= 2
	}
	if parseDelay > parseMaxDelay {
		return parseMaxDelay
	}
	return parseDelay
}

// resolveMutationQueueOptions is a core package helper.
func resolveMutationQueueOptions(parseOptions []MutationQueueOptions) MutationQueueOptions {
	parseCfg := MutationQueueOptions{
		StorageKey:  defaultMutationQueueStorageKey,
		MaxAttempts: 5,
		BaseDelay:   2 * time.Second,
		MaxDelay:    2 * time.Minute,
		Now:         time.Now,
	}
	if len(parseOptions) == 0 {
		return parseCfg
	}
	parseOverrides := parseOptions[0]
	if strings.TrimSpace(parseOverrides.StorageKey) != "" {
		parseCfg.StorageKey = strings.TrimSpace(parseOverrides.StorageKey)
	}
	if parseOverrides.MaxAttempts > 0 {
		parseCfg.MaxAttempts = parseOverrides.MaxAttempts
	}
	if parseOverrides.BaseDelay > 0 {
		parseCfg.BaseDelay = parseOverrides.BaseDelay
	}
	if parseOverrides.MaxDelay > 0 {
		parseCfg.MaxDelay = parseOverrides.MaxDelay
	}
	parseCfg.DeleteOnCorruption = parseOverrides.DeleteOnCorruption
	if parseOverrides.StorageResolver != nil {
		parseCfg.StorageResolver = parseOverrides.StorageResolver
	}
	if parseOverrides.StoreResolver != nil {
		parseCfg.StoreResolver = parseOverrides.StoreResolver
	}
	if parseOverrides.Now != nil {
		parseCfg.Now = parseOverrides.Now
	}
	return parseCfg
}

// normalizeMutationID is a core package helper.
func normalizeMutationID(parseId string, parseNow time.Time, parseOrdinal int) string {
	parseTrimmed := strings.TrimSpace(parseId)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	return fmt.Sprintf("mutation-%d-%d", parseNow.UnixNano(), parseOrdinal)
}

// cloneStringMap is a core package helper.
func cloneStringMap(parseInput map[string]string) map[string]string {
	if len(parseInput) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseInput))
	maps.Copy(parseClone, parseInput)
	return parseClone
}

// mergeResolvedMutation is a core package helper.
func mergeResolvedMutation(parseExisting QueuedMutation, parseDraft MutationDraft, parseNow time.Time, parseMessage string) QueuedMutation {
	parseResolved := parseExisting
	if parseId := strings.TrimSpace(parseDraft.ID); parseId != "" {
		parseResolved.ID = parseId
	}
	if parseKind := strings.TrimSpace(parseDraft.Kind); parseKind != "" {
		parseResolved.Kind = parseKind
	}
	if parseDedupKey := strings.TrimSpace(parseDraft.DedupKey); parseDedupKey != "" {
		parseResolved.DedupKey = parseDedupKey
	}
	if parseMethod := strings.ToUpper(strings.TrimSpace(parseDraft.Method)); parseMethod != "" {
		parseResolved.Method = parseMethod
	}
	if parseUrl := strings.TrimSpace(parseDraft.URL); parseUrl != "" {
		parseResolved.URL = parseUrl
	}
	if parseDraft.Headers != nil {
		parseResolved.Headers = cloneStringMap(parseDraft.Headers)
	}
	if parseDraft.Body != nil {
		parseResolved.Body = parseDraft.Body
	}
	if parseDraft.Metadata != nil {
		parseResolved.Metadata = cloneStringMap(parseDraft.Metadata)
	}
	parseResolved.State = MutationQueued
	parseResolved.Attempts = 0
	parseResolved.NextAttemptAt = time.Time{}
	parseResolved.LastError = parseMessage
	parseResolved.UpdatedAt = parseNow
	return parseResolved
}
