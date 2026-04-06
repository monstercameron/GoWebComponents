package fetch

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

const cachedResourceAtomPrefix = "__fetch_cached_resource:"

type CacheResumePolicy string

const (
	CacheResumeTrustOnce            CacheResumePolicy = "trust-once"
	CacheResumeStaleWhileRevalidate CacheResumePolicy = "stale-while-revalidate"
	CacheResumeAlwaysRefetch        CacheResumePolicy = "always-refetch"
)

type CacheOptions struct {
	StaleAfter   time.Duration
	MaxAge       time.Duration
	DisposeAfter time.Duration
	Persist      bool
}

// CachedResourceState describes the current state of a shared cached resource.
type CachedResourceState[T any] struct {
	Value     T
	Loading   bool
	Error     error
	Ready     bool
	Stale     bool
	UpdatedAt time.Time
}

// CachedResource exposes shared cached resource state and mutation helpers.
type CachedResource[T any] struct {
	get        func() CachedResourceState[T]
	reload     func()
	cancel     func()
	invalidate func()
	dispose    func()
	set        func(T)
	update     func(func(T) T)
}

type cachedResourceSnapshot struct {
	Value     interface{}
	Loading   bool
	Error     error
	Ready     bool
	Stale     bool
	UpdatedAt time.Time
}

type CachedResourceInspection struct {
	Key             string
	Loading         bool
	Ready           bool
	Stale           bool
	LastError       string
	UpdatedAt       time.Time
	LastLoaded      time.Time
	SubscriberCount int
	OwnerPaths      []string
	ResumePolicy    CacheResumePolicy
}

type cachedResourceEntry struct {
	mu           sync.Mutex
	valueType    reflect.Type
	staleAfter   time.Duration
	maxAge       time.Duration
	disposeAfter time.Duration
	lastLoaded   time.Time
	lastAccess   time.Time
	requestSeq   uint64
	pending      bool
	invalidated  bool
	cancel       context.CancelFunc
	done         *cachedResourceWaiters
	subscribers  int
	ownerPaths   map[string]int
	bootstrapped bool
	resumePolicy CacheResumePolicy
	persist      bool
	restored     bool
	restore      *cachedResourceWaiters
}

var cachedResourceRegistry sync.Map

type cachedResourceWaiters struct {
	ch   chan struct{}
	once sync.Once
}

// newCachedResourceWaiters is an internal cache helper.
func newCachedResourceWaiters() *cachedResourceWaiters {
	return &cachedResourceWaiters{ch: make(chan struct{})}
}

// Done is an internal cache helper.
func (parseW *cachedResourceWaiters) Done() <-chan struct{} {
	if parseW == nil {
		return nil
	}
	return parseW.ch
}

// Close is an internal cache helper.
func (parseW *cachedResourceWaiters) Close() {
	if parseW == nil {
		return
	}
	parseW.once.Do(func() {
		close(parseW.ch)
	})
}

// UseCachedResource is an internal cache helper.
func UseCachedResource[T any](parseKey string, parseLoader func(context.Context) (T, error), parseOptions ...CacheOptions) CachedResource[T] {
	parseResolved := resolveCacheOptions(parseOptions)
	parseEntry := getCachedResourceEntry(parseKey)
	configureCachedResourceEntry[T](parseKey, parseEntry, parseResolved)
	prepareCachedResourceEntry(parseKey, parseEntry)

	parseSnapshotAtom := state.UseAtom(cachedResourceAtomID(parseKey), cachedResourceSnapshot{})
	parseSnapshot := parseSnapshotAtom.Get()

	ui.UseEffect(func() func() {
		if parseKey == "" || parseLoader == nil {
			return nil
		}

		startCachedLoad(parseKey, parseEntry, func(parseCtx context.Context) (interface{}, error) {
			return parseLoader(parseCtx)
		}, false, nil)
		return nil
	}, parseKey, parseSnapshot.Ready, parseSnapshot.Stale, parseSnapshot.Loading, parseSnapshot.UpdatedAt, parseResolved.StaleAfter)

	ui.UseEffect(func() func() {
		if parseKey == "" {
			return nil
		}
		parseOwnerPath := runtime.CurrentFiberPath()
		retainCachedResource(parseKey, parseOwnerPath)
		return func() {
			releaseCachedResource(parseKey, parseOwnerPath)
		}
	}, parseKey)

	return CachedResource[T]{
		get: func() CachedResourceState[T] {
			if !shouldUseCachedHandle(parseKey, parseEntry) {
				var parseZero CachedResourceState[T]
				return parseZero
			}
			prepareCachedResourceEntry(parseKey, parseEntry)
			return toPublicCachedState[T](currentCachedSnapshot(parseKey))
		},
		reload: func() {
			if parseKey == "" || parseLoader == nil || !shouldUseCachedHandle(parseKey, parseEntry) {
				return
			}
			startCachedLoad(parseKey, parseEntry, func(parseCtx2 context.Context) (interface{}, error) {
				return parseLoader(parseCtx2)
			}, true, nil)
		},
		cancel: func() {
			if !shouldUseCachedHandle(parseKey, parseEntry) {
				return
			}
			cancelCachedLoad(parseKey)
		},
		invalidate: func() {
			if !shouldUseCachedHandle(parseKey, parseEntry) {
				return
			}
			InvalidateResource(parseKey)
		},
		dispose: func() {
			if !shouldUseCachedHandle(parseKey, parseEntry) {
				return
			}
			DisposeResource(parseKey)
		},
		set: func(parseValue T) {
			if !shouldUseCachedHandle(parseKey, parseEntry) {
				return
			}
			setCachedValue(parseKey, parseValue)
		},
		update: func(parseFn func(T) T) {
			if parseFn == nil || !shouldUseCachedHandle(parseKey, parseEntry) {
				return
			}
			cancelCachedLoad(parseKey)
			updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
				parseCurrent, _ := castCachedValue[T](parsePrev.Value)
				parsePrev.Value = parseFn(parseCurrent)
				parsePrev.Loading = false
				parsePrev.Error = nil
				parsePrev.Ready = true
				parsePrev.Stale = false
				parsePrev.UpdatedAt = time.Now()
				return parsePrev
			})
			markCachedEntryFresh(parseKey)
			persistCachedSnapshot(parseKey)
		},
	}
}

// Get returns the current cached resource state.
func (parseR CachedResource[T]) Get() CachedResourceState[T] {
	if parseR.get == nil {
		var parseZero CachedResourceState[T]
		return parseZero
	}

	return parseR.get()
}

// Reload starts a new cached resource load.
func (parseR CachedResource[T]) Reload() {
	if parseR.reload != nil {
		parseR.reload()
	}
}

// Cancel cancels the active cached resource load, if any.
func (parseR CachedResource[T]) Cancel() {
	if parseR.cancel != nil {
		parseR.cancel()
	}
}

// Invalidate marks the cached value stale and eligible for revalidation.
func (parseR CachedResource[T]) Invalidate() {
	if parseR.invalidate != nil {
		parseR.invalidate()
	}
}

// Dispose clears the cached value and removes the keyed entry from the shared registry.
func (parseR CachedResource[T]) Dispose() {
	if parseR.dispose != nil {
		parseR.dispose()
	}
}

// Set replaces the cached value optimistically.
func (parseR CachedResource[T]) Set(parseValue T) {
	if parseR.set != nil {
		parseR.set(parseValue)
	}
}

// Update replaces the cached value using the previous value.
func (parseR CachedResource[T]) Update(parseFn func(T) T) {
	if parseR.update != nil {
		parseR.update(parseFn)
	}
}

// InvalidateResource marks the named cached resource stale.
func InvalidateResource(parseKey string) {
	if parseKey == "" {
		return
	}

	runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticInformational, "cached resource invalidated", "", map[string]string{
		"key": parseKey,
	})

	parseEntry := getCachedResourceEntry(parseKey)
	parseEntry.mu.Lock()
	parseEntry.invalidated = true
	parseEntry.mu.Unlock()

	updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		parsePrev.Error = nil
		parsePrev.Stale = parsePrev.Ready
		return parsePrev
	})
}

// DisposeResource clears the named cached resource and drops its registry entry.
func DisposeResource(parseKey string) {
	if parseKey == "" {
		return
	}

	parseRaw, parseOk := cachedResourceRegistry.LoadAndDelete(parseKey)
	if parseOk {
		parseEntry := parseRaw.(*cachedResourceEntry)
		parseEntry.mu.Lock()
		parseCancel := parseEntry.cancel
		parseDone := parseEntry.done
		parseRestore := parseEntry.restore
		parseEntry.cancel = nil
		parseEntry.done = nil
		parseEntry.restore = nil
		parseEntry.pending = false
		parseEntry.invalidated = false
		parseEntry.lastLoaded = time.Time{}
		parseEntry.lastAccess = time.Time{}
		parseEntry.subscribers = 0
		parseEntry.ownerPaths = nil
		parseEntry.bootstrapped = false
		parseEntry.resumePolicy = CacheResumeTrustOnce
		parseEntry.restored = false
		parseEntry.mu.Unlock()
		if parseCancel != nil {
			parseCancel()
		}
		parseDone.Close()
		parseRestore.Close()
	}

	clearCachedSnapshot(parseKey)
	deletePersistentCachedSnapshot(parseKey)
}

// InspectCachedResources returns a stable snapshot of shared cache state for diagnostics and devtools.
func InspectCachedResources() []CachedResourceInspection {
	parseInspections := make([]CachedResourceInspection, 0)
	cachedResourceRegistry.Range(func(parseKey, parseValue interface{}) bool {
		cacheKey, _ := parseKey.(string)
		parseEntry, _ := parseValue.(*cachedResourceEntry)
		if cacheKey == "" || parseEntry == nil {
			return true
		}

		parseEntry.mu.Lock()
		parseLastLoaded := parseEntry.lastLoaded
		parseSubscribers := parseEntry.subscribers
		parseOwnerPaths := cloneCachedOwnerPaths(parseEntry.ownerPaths)
		parseResumePolicy := parseEntry.resumePolicy
		parseEntry.mu.Unlock()

		parseSnapshot := currentCachedSnapshot(cacheKey)
		parseLastError := ""
		if parseSnapshot.Error != nil {
			parseLastError = parseSnapshot.Error.Error()
		}

		parseInspections = append(parseInspections, CachedResourceInspection{
			Key:             cacheKey,
			Loading:         parseSnapshot.Loading,
			Ready:           parseSnapshot.Ready,
			Stale:           parseSnapshot.Stale,
			LastError:       parseLastError,
			UpdatedAt:       parseSnapshot.UpdatedAt,
			LastLoaded:      parseLastLoaded,
			SubscriberCount: parseSubscribers,
			OwnerPaths:      parseOwnerPaths,
			ResumePolicy:    parseResumePolicy,
		})
		return true
	})

	sort.Slice(parseInspections, func(parseI, parseJ int) bool {
		return parseInspections[parseI].Key < parseInspections[parseJ].Key
	})
	return parseInspections
}

// SweepCachedResources clears expired or idle cache entries and returns the number removed.
func SweepCachedResources() int {
	parseNow := time.Now()
	var parseDisposed int
	cachedResourceRegistry.Range(func(parseKey, parseValue interface{}) bool {
		cacheKey, _ := parseKey.(string)
		parseEntry, _ := parseValue.(*cachedResourceEntry)
		if cacheKey == "" || parseEntry == nil {
			return true
		}
		if shouldDisposeCachedEntry(parseNow, parseEntry) {
			DisposeResource(cacheKey)
			parseDisposed++
		}
		return true
	})
	return parseDisposed
}

// LoadCached reuses the shared cache from imperative code such as route loaders.
func LoadCached[T any](parseCtx context.Context, parseKey string, parseLoader func(context.Context) (T, error), parseOptions ...CacheOptions) (T, error) {
	var parseZero T
	if parseLoader == nil {
		return parseZero, fmt.Errorf("fetch: loader cannot be nil")
	}
	if parseKey == "" {
		return parseLoader(resolveCachedContext(parseCtx))
	}

	parseResolved := resolveCacheOptions(parseOptions)
	parseEntry := getCachedResourceEntry(parseKey)
	configureCachedResourceEntry[T](parseKey, parseEntry, parseResolved)

	parseAdapter := func(parseLoadCtx context.Context) (interface{}, error) {
		return parseLoader(parseLoadCtx)
	}

	for {
		prepareCachedResourceEntry(parseKey, parseEntry)
		parseSnapshot := currentCachedSnapshot(parseKey)
		if parseSnapshot.Ready && (parseSnapshot.Loading || parseSnapshot.Error != nil) {
			parseValue, _ := castCachedValue[T](parseSnapshot.Value)
			return parseValue, nil
		}
		if parseSnapshot.Error != nil && !parseSnapshot.Ready && !parseSnapshot.Loading {
			return parseZero, parseSnapshot.Error
		}

		parseEntry.mu.Lock()
		parseNeedsLoad := shouldLoadCachedEntry(parseSnapshot, parseEntry)
		if !parseNeedsLoad {
			parseWaiters := parseEntry.done
			if parseWaiters == nil {
				parseWaiters = parseEntry.restore
			}
			parseEntry.mu.Unlock()
			if parseSnapshot.Ready {
				parseValue2, _ := castCachedValue[T](parseSnapshot.Value)
				return parseValue2, nil
			}
			if parseSnapshot.Error != nil && parseWaiters == nil {
				return parseZero, parseSnapshot.Error
			}
			if parseWaiters == nil {
				return parseZero, nil
			}
			if parseErr := waitForCachedResource(parseCtx, parseWaiters); parseErr != nil {
				return parseZero, parseErr
			}
			continue
		}
		parseEntry.mu.Unlock()

		parseWaiters2, _ := startCachedLoad(parseKey, parseEntry, parseAdapter, false, parseCtx)
		if parseErr2 := waitForCachedResource(parseCtx, parseWaiters2); parseErr2 != nil {
			return parseZero, parseErr2
		}
	}
}

// resolveCacheOptions is an internal cache helper.
func resolveCacheOptions(parseOptions []CacheOptions) CacheOptions {
	if len(parseOptions) == 0 {
		return CacheOptions{}
	}
	return parseOptions[0]
}

// shouldUseCachedHandle reports whether a cached-resource handle still owns the live registry entry for its key.
func shouldUseCachedHandle(parseKey string, parseEntry *cachedResourceEntry) bool {
	if parseKey == "" || parseEntry == nil {
		return false
	}

	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return false
	}
	return parseRaw == parseEntry
}

// getCachedResourceEntry is an internal cache helper.
func getCachedResourceEntry(parseKey string) *cachedResourceEntry {
	if parseKey == "" {
		return &cachedResourceEntry{}
	}

	parseRaw, _ := cachedResourceRegistry.LoadOrStore(parseKey, &cachedResourceEntry{})
	return parseRaw.(*cachedResourceEntry)
}

// retainCachedResource is an internal cache helper.
func retainCachedResource(parseKey string, parseOwnerPath string) {
	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return
	}
	parseEntry := parseRaw.(*cachedResourceEntry)
	parseEntry.mu.Lock()
	parseEntry.subscribers++
	if parseTrimmedOwner := strings.TrimSpace(parseOwnerPath); parseTrimmedOwner != "" {
		if parseEntry.ownerPaths == nil {
			parseEntry.ownerPaths = map[string]int{}
		}
		parseEntry.ownerPaths[parseTrimmedOwner]++
	}
	parseEntry.lastAccess = time.Now()
	parseEntry.mu.Unlock()
}

// releaseCachedResource is an internal cache helper.
func releaseCachedResource(parseKey string, parseOwnerPath string) {
	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return
	}
	parseEntry := parseRaw.(*cachedResourceEntry)
	parseEntry.mu.Lock()
	if parseEntry.subscribers > 0 {
		parseEntry.subscribers--
	}
	if parseTrimmedOwner := strings.TrimSpace(parseOwnerPath); parseTrimmedOwner != "" && len(parseEntry.ownerPaths) > 0 {
		if parseEntry.ownerPaths[parseTrimmedOwner] <= 1 {
			delete(parseEntry.ownerPaths, parseTrimmedOwner)
		} else {
			parseEntry.ownerPaths[parseTrimmedOwner]--
		}
	}
	parseEntry.lastAccess = time.Now()
	parseEntry.mu.Unlock()
}

// cloneCachedOwnerPaths is an internal cache helper.
func cloneCachedOwnerPaths(parseInput map[string]int) []string {
	if len(parseInput) == 0 {
		return nil
	}
	parseOwners := make([]string, 0, len(parseInput))
	for parsePath := range parseInput {
		parseOwners = append(parseOwners, parsePath)
	}
	sort.Strings(parseOwners)
	return parseOwners
}

// configureCachedResourceEntry is an internal cache helper.
func configureCachedResourceEntry[T any](parseKey string, parseEntry *cachedResourceEntry, parseOptions CacheOptions) {
	if parseKey == "" || parseEntry == nil {
		return
	}

	parseDesiredType := reflect.TypeOf((*T)(nil)).Elem()
	parseEntry.mu.Lock()
	defer parseEntry.mu.Unlock()

	if parseEntry.valueType == nil {
		parseEntry.valueType = parseDesiredType
	} else if parseEntry.valueType != parseDesiredType {
		runtime.ReportDiagnostic("fetch", runtime.DiagnosticWarning, fmt.Sprintf("UseCachedResource key %q requested with conflicting value types %s and %s", parseKey, parseEntry.valueType, parseDesiredType))
	}

	if parseOptions.StaleAfter > 0 {
		parseEntry.staleAfter = parseOptions.StaleAfter
	}
	if parseOptions.MaxAge > 0 {
		parseEntry.maxAge = parseOptions.MaxAge
	}
	if parseOptions.DisposeAfter > 0 {
		parseEntry.disposeAfter = parseOptions.DisposeAfter
	}
	if parseOptions.Persist {
		parseEntry.persist = true
	}
}

// cachedResourceAtomID is an internal cache helper.
func cachedResourceAtomID(parseKey string) string {
	return cachedResourceAtomPrefix + parseKey
}

// currentCachedSnapshot is an internal cache helper.
func currentCachedSnapshot(parseKey string) cachedResourceSnapshot {
	if parseKey == "" {
		return cachedResourceSnapshot{}
	}

	parseRt := runtime.GetGlobalRuntime()
	if parseRt == nil {
		return cachedResourceSnapshot{}
	}

	parseValue, parseOk := parseRt.GetAtomValue(cachedResourceAtomID(parseKey))
	if !parseOk {
		return cachedResourceSnapshot{}
	}

	parseSnapshot, parseOk := parseValue.(cachedResourceSnapshot)
	if !parseOk {
		return cachedResourceSnapshot{}
	}

	return parseSnapshot
}

// updateCachedSnapshot is an internal cache helper.
func updateCachedSnapshot(parseKey string, parseUpdate func(cachedResourceSnapshot) cachedResourceSnapshot) {
	if parseKey == "" || parseUpdate == nil {
		return
	}

	parseSnapshot := parseUpdate(currentCachedSnapshot(parseKey))
	parseRt := runtime.GetGlobalRuntime()
	if parseRt == nil {
		return
	}
	_ = parseRt.RestoreAtomSnapshot(map[string]interface{}{cachedResourceAtomID(parseKey): parseSnapshot})
}

// setCachedValue is an internal cache helper.
func setCachedValue[T any](parseKey string, parseValue T) {
	if parseKey == "" {
		return
	}

	cancelCachedLoad(parseKey)
	updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		parsePrev.Value = parseValue
		parsePrev.Loading = false
		parsePrev.Error = nil
		parsePrev.Ready = true
		parsePrev.Stale = false
		parsePrev.UpdatedAt = time.Now()
		return parsePrev
	})
	markCachedEntryFresh(parseKey)
	persistCachedSnapshot(parseKey)
}

// markCachedEntryFresh is an internal cache helper.
func markCachedEntryFresh(parseKey string) {
	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return
	}

	parseEntry := parseRaw.(*cachedResourceEntry)
	parseEntry.mu.Lock()
	parseEntry.invalidated = false
	parseEntry.lastLoaded = time.Now()
	parseEntry.lastAccess = parseEntry.lastLoaded
	parseEntry.mu.Unlock()
}

// startCachedLoad is an internal cache helper.
func startCachedLoad(parseKey string, parseEntry *cachedResourceEntry, parseLoader func(context.Context) (interface{}, error), isForce bool, parseParent context.Context) (*cachedResourceWaiters, bool) {
	if parseKey == "" || parseEntry == nil || parseLoader == nil {
		return nil, false
	}

	parseEntry.mu.Lock()
	parseCurrent := currentCachedSnapshot(parseKey)
	if !isForce && !shouldLoadCachedEntry(parseCurrent, parseEntry) {
		parseWaiters := parseEntry.done
		parseEntry.mu.Unlock()
		return parseWaiters, false
	}
	if parseEntry.pending {
		parseWaiters2 := parseEntry.done
		parseEntry.mu.Unlock()
		return parseWaiters2, false
	}
	parseEntry.requestSeq++
	parseSeq := parseEntry.requestSeq
	parseCtx, parseCancel := context.WithCancel(resolveCachedContext(parseParent))
	parseWaiters3 := newCachedResourceWaiters()
	parseEntry.cancel = parseCancel
	parseEntry.pending = true
	parseEntry.invalidated = false
	parseEntry.done = parseWaiters3
	parseEntry.lastAccess = time.Now()
	parseEntry.bootstrapped = false
	parseEntry.mu.Unlock()

	updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		parsePrev.Loading = true
		parsePrev.Error = nil
		if parsePrev.Ready {
			parsePrev.Stale = true
		}
		return parsePrev
	})

	go func(parseRequestSeq uint64, parseRequestCtx context.Context, parseDone *cachedResourceWaiters) {
		parseValue, parseErr := parseLoader(parseRequestCtx)

		parseEntry.mu.Lock()
		if parseRequestSeq != parseEntry.requestSeq {
			if parseEntry.done == parseDone {
				parseEntry.done = nil
			}
			parseEntry.mu.Unlock()
			parseDone.Close()
			return
		}
		parseEntry.pending = false
		parseEntry.cancel = nil
		parseEntry.done = nil
		if parseErr == nil && parseRequestCtx.Err() == nil {
			parseEntry.lastLoaded = time.Now()
		}
		parseStillInvalidated := parseEntry.invalidated
		parseEntry.mu.Unlock()
		parseDone.Close()

		if parseRequestCtx.Err() != nil {
			updateCachedSnapshot(parseKey, func(parsePrev2 cachedResourceSnapshot) cachedResourceSnapshot {
				parsePrev2.Loading = false
				if parsePrev2.Ready {
					parsePrev2.Stale = parsePrev2.Stale || parseStillInvalidated
				} else {
					parsePrev2.Stale = false
				}
				return parsePrev2
			})
			return
		}

		updateCachedSnapshot(parseKey, func(parsePrev3 cachedResourceSnapshot) cachedResourceSnapshot {
			parsePrev3.Loading = false
			parsePrev3.UpdatedAt = time.Now()
			if parseErr != nil {
				parsePrev3.Error = parseErr
				if parsePrev3.Ready {
					parsePrev3.Stale = true
				} else {
					parsePrev3.Stale = false
				}
				return parsePrev3
			}

			parsePrev3.Value = parseValue
			parsePrev3.Error = nil
			parsePrev3.Ready = true
			parsePrev3.Stale = false
			return parsePrev3
		})
		persistCachedSnapshot(parseKey)
	}(parseSeq, parseCtx, parseWaiters3)
	return parseWaiters3, true
}

// shouldLoadCachedEntry is an internal cache helper.
func shouldLoadCachedEntry(parseSnapshot cachedResourceSnapshot, parseEntry *cachedResourceEntry) bool {
	if parseEntry.pending {
		return false
	}
	if parseEntry.restore != nil {
		return false
	}
	if parseEntry.bootstrapped && parseEntry.resumePolicy == CacheResumeAlwaysRefetch {
		return true
	}
	if !parseSnapshot.Ready {
		return !parseSnapshot.Loading
	}
	if parseEntry.invalidated || parseSnapshot.Stale {
		return true
	}
	if parseEntry.staleAfter > 0 && !parseEntry.lastLoaded.IsZero() && time.Since(parseEntry.lastLoaded) >= parseEntry.staleAfter {
		return true
	}
	return false
}

// cancelCachedLoad is an internal cache helper.
func cancelCachedLoad(parseKey string) {
	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return
	}

	parseEntry := parseRaw.(*cachedResourceEntry)
	parseEntry.mu.Lock()
	parseCancel := parseEntry.cancel
	parseDone := parseEntry.done
	parseEntry.cancel = nil
	parseEntry.done = nil
	parseEntry.pending = false
	parseStillInvalidated := parseEntry.invalidated
	parseEntry.mu.Unlock()

	if parseCancel != nil {
		parseCancel()
	}
	parseDone.Close()

	updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		parsePrev.Loading = false
		if parsePrev.Ready {
			parsePrev.Stale = parsePrev.Stale || parseStillInvalidated
		} else {
			parsePrev.Stale = false
		}
		return parsePrev
	})
}

// resolveCachedContext is an internal cache helper.
func resolveCachedContext(parseCtx context.Context) context.Context {
	if parseCtx != nil {
		return parseCtx
	}
	return context.Background()
}

// waitForCachedResource is an internal cache helper.
func waitForCachedResource(parseCtx context.Context, parseWaiters *cachedResourceWaiters) error {
	if parseWaiters == nil {
		return nil
	}
	parseWaitCh := parseWaiters.Done()
	if parseWaitCh == nil {
		return nil
	}
	if parseCtx == nil {
		<-parseWaitCh
		return nil
	}
	select {
	case <-parseCtx.Done():
		return parseCtx.Err()
	case <-parseWaitCh:
		return nil
	}
}

// prepareCachedResourceEntry is an internal cache helper.
func prepareCachedResourceEntry(parseKey string, parseEntry *cachedResourceEntry) {
	if parseKey == "" || parseEntry == nil {
		return
	}

	parseNow := time.Now()
	isParseExpireSnapshot := false
	isParseDisposeEntry := false
	isParsePersistSnapshot := false
	isParseStartRestore := false
	parseSnapshot := currentCachedSnapshot(parseKey)

	parseEntry.mu.Lock()
	if !parseEntry.pending && parseEntry.maxAge > 0 && !parseEntry.lastLoaded.IsZero() && parseNow.Sub(parseEntry.lastLoaded) >= parseEntry.maxAge {
		parseEntry.lastLoaded = time.Time{}
		parseEntry.invalidated = false
		isParseExpireSnapshot = true
	}
	if !parseEntry.pending && parseEntry.disposeAfter > 0 && !parseEntry.lastAccess.IsZero() && parseNow.Sub(parseEntry.lastAccess) >= parseEntry.disposeAfter {
		isParseDisposeEntry = true
	}
	if parseEntry.persist && !parseEntry.restored && parseEntry.restore == nil {
		if parseSnapshot.Ready {
			parseEntry.restored = true
			isParsePersistSnapshot = true
		} else {
			parseEntry.restore = newCachedResourceWaiters()
			isParseStartRestore = true
		}
	}
	parseEntry.lastAccess = parseNow
	parseEntry.mu.Unlock()

	if isParseDisposeEntry {
		resetCachedResourceEntry(parseKey, parseEntry)
		return
	}
	if isParseExpireSnapshot {
		clearCachedSnapshot(parseKey)
		deletePersistentCachedSnapshot(parseKey)
	}
	if isParseStartRestore {
		startPersistentCachedRestore(parseKey, parseEntry)
	}
	if isParsePersistSnapshot {
		persistCachedSnapshot(parseKey)
	}
}

// clearCachedSnapshot is an internal cache helper.
func clearCachedSnapshot(parseKey string) {
	updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		return cachedResourceSnapshot{}
	})
}

// resetCachedResourceEntry is an internal cache helper.
func resetCachedResourceEntry(parseKey string, parseEntry *cachedResourceEntry) {
	if parseKey == "" || parseEntry == nil {
		return
	}

	parseEntry.mu.Lock()
	parseCancel := parseEntry.cancel
	parseDone := parseEntry.done
	parseRestore := parseEntry.restore
	parseEntry.cancel = nil
	parseEntry.done = nil
	parseEntry.restore = nil
	parseEntry.pending = false
	parseEntry.invalidated = false
	parseEntry.lastLoaded = time.Time{}
	parseEntry.lastAccess = time.Now()
	parseEntry.bootstrapped = false
	parseEntry.restored = false
	parseEntry.mu.Unlock()

	if parseCancel != nil {
		parseCancel()
	}
	parseDone.Close()
	parseRestore.Close()
	clearCachedSnapshot(parseKey)
}

// shouldDisposeCachedEntry is an internal cache helper.
func shouldDisposeCachedEntry(parseNow time.Time, parseEntry *cachedResourceEntry) bool {
	parseEntry.mu.Lock()
	defer parseEntry.mu.Unlock()
	if parseEntry.pending || parseEntry.restore != nil {
		return false
	}
	if parseEntry.disposeAfter > 0 && !parseEntry.lastAccess.IsZero() && parseNow.Sub(parseEntry.lastAccess) >= parseEntry.disposeAfter {
		return true
	}
	if parseEntry.maxAge > 0 && !parseEntry.lastLoaded.IsZero() && parseNow.Sub(parseEntry.lastLoaded) >= parseEntry.maxAge {
		return true
	}
	return false
}

// toPublicCachedState is an internal cache helper.
func toPublicCachedState[T any](parseSnapshot cachedResourceSnapshot) CachedResourceState[T] {
	parseValue, _ := castCachedValue[T](parseSnapshot.Value)
	return CachedResourceState[T]{
		Value:     parseValue,
		Loading:   parseSnapshot.Loading,
		Error:     parseSnapshot.Error,
		Ready:     parseSnapshot.Ready,
		Stale:     parseSnapshot.Stale,
		UpdatedAt: parseSnapshot.UpdatedAt,
	}
}

// castCachedValue is an internal cache helper.
func castCachedValue[T any](parseValue interface{}) (T, bool) {
	parseCast, parseOk := parseValue.(T)
	if parseOk {
		return parseCast, true
	}

	var parseZero T
	return parseZero, false
}
