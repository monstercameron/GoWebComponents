//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"encoding/json"
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
const CacheBootstrapDataKey = "fetchCache"

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

type CacheBootstrapEntry struct {
	Key          string            `json:"key,omitempty"`
	Value        interface{}       `json:"value,omitempty"`
	UpdatedAt    time.Time         `json:"updatedAt,omitempty"`
	ResumePolicy CacheResumePolicy `json:"resumePolicy,omitempty"`
	StaleAfter   time.Duration     `json:"staleAfter,omitempty"`
}

type CacheBootstrap struct {
	Entries []CacheBootstrapEntry `json:"entries,omitempty"`
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
	bootstrapped bool
	resumePolicy CacheResumePolicy
}

var cachedResourceRegistry sync.Map

type cachedResourceWaiters struct {
	ch   chan struct{}
	once sync.Once
}

func newCachedResourceWaiters() *cachedResourceWaiters {
	return &cachedResourceWaiters{ch: make(chan struct{})}
}

func (w *cachedResourceWaiters) Done() <-chan struct{} {
	if w == nil {
		return nil
	}
	return w.ch
}

func (w *cachedResourceWaiters) Close() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		close(w.ch)
	})
}

func UseCachedResource[T any](key string, loader func(context.Context) (T, error), options ...CacheOptions) CachedResource[T] {
	resolved := resolveCacheOptions(options)
	entry := getCachedResourceEntry(key)
	configureCachedResourceEntry[T](key, entry, resolved)
	prepareCachedResourceEntry(key, entry)

	snapshotAtom := state.UseAtom(cachedResourceAtomID(key), cachedResourceSnapshot{})
	snapshot := snapshotAtom.Get()

	ui.UseEffect(func() func() {
		if key == "" || loader == nil {
			return nil
		}

		startCachedLoad(key, entry, func(ctx context.Context) (interface{}, error) {
			return loader(ctx)
		}, false, nil)
		return nil
	}, key, snapshot.Ready, snapshot.Stale, snapshot.Loading, snapshot.UpdatedAt, resolved.StaleAfter)

	ui.UseEffect(func() func() {
		if key == "" {
			return nil
		}
		retainCachedResource(key)
		return func() {
			releaseCachedResource(key)
		}
	}, key)

	return CachedResource[T]{
		get: func() CachedResourceState[T] {
			prepareCachedResourceEntry(key, entry)
			return toPublicCachedState[T](currentCachedSnapshot(key))
		},
		reload: func() {
			if key == "" || loader == nil {
				return
			}
			startCachedLoad(key, entry, func(ctx context.Context) (interface{}, error) {
				return loader(ctx)
			}, true, nil)
		},
		cancel: func() {
			cancelCachedLoad(key)
		},
		invalidate: func() {
			InvalidateResource(key)
		},
		dispose: func() {
			DisposeResource(key)
		},
		set: func(value T) {
			setCachedValue(key, value)
		},
		update: func(fn func(T) T) {
			if fn == nil {
				return
			}
			updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
				current, _ := castCachedValue[T](prev.Value)
				prev.Value = fn(current)
				prev.Loading = false
				prev.Error = nil
				prev.Ready = true
				prev.Stale = false
				prev.UpdatedAt = time.Now()
				return prev
			})
			markCachedEntryFresh(key)
		},
	}
}

// Get returns the current cached resource state.
func (r CachedResource[T]) Get() CachedResourceState[T] {
	if r.get == nil {
		var zero CachedResourceState[T]
		return zero
	}

	return r.get()
}

// Reload starts a new cached resource load.
func (r CachedResource[T]) Reload() {
	if r.reload != nil {
		r.reload()
	}
}

// Cancel cancels the active cached resource load, if any.
func (r CachedResource[T]) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
}

// Invalidate marks the cached value stale and eligible for revalidation.
func (r CachedResource[T]) Invalidate() {
	if r.invalidate != nil {
		r.invalidate()
	}
}

// Dispose clears the cached value and removes the keyed entry from the shared registry.
func (r CachedResource[T]) Dispose() {
	if r.dispose != nil {
		r.dispose()
	}
}

// Set replaces the cached value optimistically.
func (r CachedResource[T]) Set(value T) {
	if r.set != nil {
		r.set(value)
	}
}

// Update replaces the cached value using the previous value.
func (r CachedResource[T]) Update(fn func(T) T) {
	if r.update != nil {
		r.update(fn)
	}
}

// InvalidateResource marks the named cached resource stale.
func InvalidateResource(key string) {
	if key == "" {
		return
	}

	runtime.ReportLogWithFields("fetch", runtime.LogInfo, runtime.DiagnosticInformational, "cached resource invalidated", "", map[string]string{
		"key": key,
	})

	entry := getCachedResourceEntry(key)
	entry.mu.Lock()
	entry.invalidated = true
	entry.mu.Unlock()

	updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
		prev.Error = nil
		prev.Stale = prev.Ready
		return prev
	})
}

// DisposeResource clears the named cached resource and drops its registry entry.
func DisposeResource(key string) {
	if key == "" {
		return
	}

	raw, ok := cachedResourceRegistry.LoadAndDelete(key)
	if ok {
		entry := raw.(*cachedResourceEntry)
		entry.mu.Lock()
		cancel := entry.cancel
		done := entry.done
		entry.cancel = nil
		entry.done = nil
		entry.pending = false
		entry.invalidated = false
		entry.lastLoaded = time.Time{}
		entry.lastAccess = time.Time{}
		entry.subscribers = 0
		entry.bootstrapped = false
		entry.resumePolicy = CacheResumeTrustOnce
		entry.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		done.Close()
	}

	clearCachedSnapshot(key)
}

// InspectCachedResources returns a stable snapshot of shared cache state for diagnostics and devtools.
func InspectCachedResources() []CachedResourceInspection {
	inspections := make([]CachedResourceInspection, 0)
	cachedResourceRegistry.Range(func(key, value interface{}) bool {
		cacheKey, _ := key.(string)
		entry, _ := value.(*cachedResourceEntry)
		if cacheKey == "" || entry == nil {
			return true
		}

		entry.mu.Lock()
		lastLoaded := entry.lastLoaded
		subscribers := entry.subscribers
		resumePolicy := entry.resumePolicy
		entry.mu.Unlock()

		snapshot := currentCachedSnapshot(cacheKey)
		lastError := ""
		if snapshot.Error != nil {
			lastError = snapshot.Error.Error()
		}

		inspections = append(inspections, CachedResourceInspection{
			Key:             cacheKey,
			Loading:         snapshot.Loading,
			Ready:           snapshot.Ready,
			Stale:           snapshot.Stale,
			LastError:       lastError,
			UpdatedAt:       snapshot.UpdatedAt,
			LastLoaded:      lastLoaded,
			SubscriberCount: subscribers,
			ResumePolicy:    resumePolicy,
		})
		return true
	})

	sort.Slice(inspections, func(i, j int) bool {
		return inspections[i].Key < inspections[j].Key
	})
	return inspections
}

// RestoreCacheBootstrap seeds shared cached resources from a UI bootstrap payload.
func RestoreCacheBootstrap(payload ui.SSRBootstrap) error {
	bootstrap, err := readCacheBootstrap(payload.Data)
	if err != nil {
		return err
	}
	restoreCacheBootstrapEntries(bootstrap.Entries)
	return nil
}

// SweepCachedResources clears expired or idle cache entries and returns the number removed.
func SweepCachedResources() int {
	now := time.Now()
	var disposed int
	cachedResourceRegistry.Range(func(key, value interface{}) bool {
		cacheKey, _ := key.(string)
		entry, _ := value.(*cachedResourceEntry)
		if cacheKey == "" || entry == nil {
			return true
		}
		if shouldDisposeCachedEntry(now, entry) {
			DisposeResource(cacheKey)
			disposed++
		}
		return true
	})
	return disposed
}

// LoadCached reuses the shared cache from imperative code such as route loaders.
func LoadCached[T any](ctx context.Context, key string, loader func(context.Context) (T, error), options ...CacheOptions) (T, error) {
	var zero T
	if loader == nil {
		return zero, fmt.Errorf("fetch: loader cannot be nil")
	}
	if key == "" {
		return loader(resolveCachedContext(ctx))
	}

	resolved := resolveCacheOptions(options)
	entry := getCachedResourceEntry(key)
	configureCachedResourceEntry[T](key, entry, resolved)

	adapter := func(loadCtx context.Context) (interface{}, error) {
		return loader(loadCtx)
	}

	for {
		prepareCachedResourceEntry(key, entry)
		snapshot := currentCachedSnapshot(key)
		if snapshot.Ready && (snapshot.Loading || snapshot.Error != nil) {
			value, _ := castCachedValue[T](snapshot.Value)
			return value, nil
		}

		entry.mu.Lock()
		needsLoad := shouldLoadCachedEntry(snapshot, entry)
		if !needsLoad {
			waiters := entry.done
			entry.mu.Unlock()
			if snapshot.Ready {
				value, _ := castCachedValue[T](snapshot.Value)
				return value, nil
			}
			if snapshot.Error != nil && waiters == nil {
				return zero, snapshot.Error
			}
			if waiters == nil {
				return zero, nil
			}
			if err := waitForCachedResource(ctx, waiters); err != nil {
				return zero, err
			}
			continue
		}
		entry.mu.Unlock()

		waiters, _ := startCachedLoad(key, entry, adapter, false, ctx)
		if err := waitForCachedResource(ctx, waiters); err != nil {
			return zero, err
		}
	}
}

func resolveCacheOptions(options []CacheOptions) CacheOptions {
	if len(options) == 0 {
		return CacheOptions{}
	}
	return options[0]
}

func readCacheBootstrap(data map[string]interface{}) (CacheBootstrap, error) {
	if len(data) == 0 {
		return CacheBootstrap{}, nil
	}
	raw, ok := data[CacheBootstrapDataKey]
	if !ok || raw == nil {
		return CacheBootstrap{}, nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return CacheBootstrap{}, err
	}
	var bootstrap CacheBootstrap
	if err := json.Unmarshal(encoded, &bootstrap); err != nil {
		return CacheBootstrap{}, err
	}
	return bootstrap, nil
}

func restoreCacheBootstrapEntries(entries []CacheBootstrapEntry) {
	now := time.Now()
	for _, item := range entries {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		entry := getCachedResourceEntry(key)
		entry.mu.Lock()
		entry.lastLoaded = item.UpdatedAt
		entry.lastAccess = now
		entry.resumePolicy = normalizeResumePolicy(item.ResumePolicy)
		entry.bootstrapped = true
		entry.pending = false
		entry.cancel = nil
		entry.done = nil
		entry.mu.Unlock()

		updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
			return cachedResourceSnapshot{
				Value:     item.Value,
				Loading:   false,
				Error:     nil,
				Ready:     true,
				Stale:     bootstrapEntryShouldStartStale(now, item),
				UpdatedAt: item.UpdatedAt,
			}
		})
	}
}

func getCachedResourceEntry(key string) *cachedResourceEntry {
	if key == "" {
		return &cachedResourceEntry{}
	}

	raw, _ := cachedResourceRegistry.LoadOrStore(key, &cachedResourceEntry{})
	return raw.(*cachedResourceEntry)
}

func retainCachedResource(key string) {
	raw, ok := cachedResourceRegistry.Load(key)
	if !ok {
		return
	}
	entry := raw.(*cachedResourceEntry)
	entry.mu.Lock()
	entry.subscribers++
	entry.lastAccess = time.Now()
	entry.mu.Unlock()
}

func releaseCachedResource(key string) {
	raw, ok := cachedResourceRegistry.Load(key)
	if !ok {
		return
	}
	entry := raw.(*cachedResourceEntry)
	entry.mu.Lock()
	if entry.subscribers > 0 {
		entry.subscribers--
	}
	entry.lastAccess = time.Now()
	entry.mu.Unlock()
}

func configureCachedResourceEntry[T any](key string, entry *cachedResourceEntry, options CacheOptions) {
	if key == "" || entry == nil {
		return
	}

	desiredType := reflect.TypeOf((*T)(nil)).Elem()
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.valueType == nil {
		entry.valueType = desiredType
	} else if entry.valueType != desiredType {
		runtime.ReportDiagnostic("fetch", runtime.DiagnosticWarning, fmt.Sprintf("UseCachedResource key %q requested with conflicting value types %s and %s", key, entry.valueType, desiredType))
	}

	if options.StaleAfter > 0 {
		entry.staleAfter = options.StaleAfter
	}
	if options.MaxAge > 0 {
		entry.maxAge = options.MaxAge
	}
	if options.DisposeAfter > 0 {
		entry.disposeAfter = options.DisposeAfter
	}
}

func cachedResourceAtomID(key string) string {
	return cachedResourceAtomPrefix + key
}

func currentCachedSnapshot(key string) cachedResourceSnapshot {
	if key == "" {
		return cachedResourceSnapshot{}
	}

	rt := runtime.GetGlobalRuntime()
	if rt == nil {
		return cachedResourceSnapshot{}
	}

	value, ok := rt.GetAtomValue(cachedResourceAtomID(key))
	if !ok {
		return cachedResourceSnapshot{}
	}

	snapshot, ok := value.(cachedResourceSnapshot)
	if !ok {
		return cachedResourceSnapshot{}
	}

	return snapshot
}

func updateCachedSnapshot(key string, update func(cachedResourceSnapshot) cachedResourceSnapshot) {
	if key == "" || update == nil {
		return
	}

	snapshot := update(currentCachedSnapshot(key))
	rt := runtime.GetGlobalRuntime()
	if rt == nil {
		return
	}
	_ = rt.RestoreAtomSnapshot(map[string]interface{}{cachedResourceAtomID(key): snapshot})
}

func setCachedValue[T any](key string, value T) {
	if key == "" {
		return
	}

	updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
		prev.Value = value
		prev.Loading = false
		prev.Error = nil
		prev.Ready = true
		prev.Stale = false
		prev.UpdatedAt = time.Now()
		return prev
	})
	markCachedEntryFresh(key)
}

func markCachedEntryFresh(key string) {
	raw, ok := cachedResourceRegistry.Load(key)
	if !ok {
		return
	}

	entry := raw.(*cachedResourceEntry)
	entry.mu.Lock()
	entry.invalidated = false
	entry.lastLoaded = time.Now()
	entry.lastAccess = entry.lastLoaded
	entry.mu.Unlock()
}

func startCachedLoad(key string, entry *cachedResourceEntry, loader func(context.Context) (interface{}, error), force bool, parent context.Context) (*cachedResourceWaiters, bool) {
	if key == "" || entry == nil || loader == nil {
		return nil, false
	}

	entry.mu.Lock()
	current := currentCachedSnapshot(key)
	if !force && !shouldLoadCachedEntry(current, entry) {
		waiters := entry.done
		entry.mu.Unlock()
		return waiters, false
	}
	if entry.pending {
		waiters := entry.done
		entry.mu.Unlock()
		return waiters, false
	}
	entry.requestSeq++
	seq := entry.requestSeq
	ctx, cancel := context.WithCancel(resolveCachedContext(parent))
	waiters := newCachedResourceWaiters()
	entry.cancel = cancel
	entry.pending = true
	entry.invalidated = false
	entry.done = waiters
	entry.lastAccess = time.Now()
	entry.bootstrapped = false
	entry.mu.Unlock()

	updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
		prev.Loading = true
		prev.Error = nil
		if prev.Ready {
			prev.Stale = true
		}
		return prev
	})

	go func(requestSeq uint64, requestCtx context.Context, done *cachedResourceWaiters) {
		value, err := loader(requestCtx)

		entry.mu.Lock()
		if requestSeq != entry.requestSeq {
			if entry.done == done {
				entry.done = nil
			}
			entry.mu.Unlock()
			done.Close()
			return
		}
		entry.pending = false
		entry.cancel = nil
		entry.done = nil
		if err == nil && requestCtx.Err() == nil {
			entry.lastLoaded = time.Now()
		}
		stillInvalidated := entry.invalidated
		entry.mu.Unlock()
		done.Close()

		if requestCtx.Err() != nil {
			updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
				prev.Loading = false
				if prev.Ready {
					prev.Stale = prev.Stale || stillInvalidated
				} else {
					prev.Stale = false
				}
				return prev
			})
			return
		}

		updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
			prev.Loading = false
			prev.UpdatedAt = time.Now()
			if err != nil {
				prev.Error = err
				if prev.Ready {
					prev.Stale = true
				} else {
					prev.Stale = false
				}
				return prev
			}

			prev.Value = value
			prev.Error = nil
			prev.Ready = true
			prev.Stale = false
			return prev
		})
	}(seq, ctx, waiters)
	return waiters, true
}

func shouldLoadCachedEntry(snapshot cachedResourceSnapshot, entry *cachedResourceEntry) bool {
	if entry.pending {
		return false
	}
	if entry.bootstrapped && entry.resumePolicy == CacheResumeAlwaysRefetch {
		return true
	}
	if !snapshot.Ready {
		return !snapshot.Loading
	}
	if entry.invalidated || snapshot.Stale {
		return true
	}
	if entry.staleAfter > 0 && !entry.lastLoaded.IsZero() && time.Since(entry.lastLoaded) >= entry.staleAfter {
		return true
	}
	return false
}

func cancelCachedLoad(key string) {
	raw, ok := cachedResourceRegistry.Load(key)
	if !ok {
		return
	}

	entry := raw.(*cachedResourceEntry)
	entry.mu.Lock()
	cancel := entry.cancel
	done := entry.done
	entry.cancel = nil
	entry.done = nil
	entry.pending = false
	stillInvalidated := entry.invalidated
	entry.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	done.Close()

	updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
		prev.Loading = false
		if prev.Ready {
			prev.Stale = prev.Stale || stillInvalidated
		} else {
			prev.Stale = false
		}
		return prev
	})
}

func resolveCachedContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func waitForCachedResource(ctx context.Context, waiters *cachedResourceWaiters) error {
	if waiters == nil {
		return nil
	}
	waitCh := waiters.Done()
	if waitCh == nil {
		return nil
	}
	if ctx == nil {
		<-waitCh
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCh:
		return nil
	}
}

func normalizeResumePolicy(policy CacheResumePolicy) CacheResumePolicy {
	switch policy {
	case CacheResumeStaleWhileRevalidate, CacheResumeAlwaysRefetch:
		return policy
	default:
		return CacheResumeTrustOnce
	}
}

func bootstrapEntryShouldStartStale(now time.Time, entry CacheBootstrapEntry) bool {
	switch normalizeResumePolicy(entry.ResumePolicy) {
	case CacheResumeAlwaysRefetch:
		return true
	case CacheResumeStaleWhileRevalidate:
		if entry.UpdatedAt.IsZero() || entry.StaleAfter <= 0 {
			return true
		}
		return now.Sub(entry.UpdatedAt) >= entry.StaleAfter
	default:
		return false
	}
}

func prepareCachedResourceEntry(key string, entry *cachedResourceEntry) {
	if key == "" || entry == nil {
		return
	}

	now := time.Now()
	expireSnapshot := false
	disposeEntry := false

	entry.mu.Lock()
	if !entry.pending && entry.maxAge > 0 && !entry.lastLoaded.IsZero() && now.Sub(entry.lastLoaded) >= entry.maxAge {
		entry.lastLoaded = time.Time{}
		entry.invalidated = false
		expireSnapshot = true
	}
	if !entry.pending && entry.disposeAfter > 0 && !entry.lastAccess.IsZero() && now.Sub(entry.lastAccess) >= entry.disposeAfter {
		disposeEntry = true
	}
	entry.lastAccess = now
	entry.mu.Unlock()

	if disposeEntry {
		resetCachedResourceEntry(key, entry)
		return
	}
	if expireSnapshot {
		clearCachedSnapshot(key)
	}
}

func clearCachedSnapshot(key string) {
	updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
		return cachedResourceSnapshot{}
	})
}

func resetCachedResourceEntry(key string, entry *cachedResourceEntry) {
	if key == "" || entry == nil {
		return
	}

	entry.mu.Lock()
	cancel := entry.cancel
	done := entry.done
	entry.cancel = nil
	entry.done = nil
	entry.pending = false
	entry.invalidated = false
	entry.lastLoaded = time.Time{}
	entry.lastAccess = time.Now()
	entry.bootstrapped = false
	entry.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	done.Close()
	clearCachedSnapshot(key)
}

func shouldDisposeCachedEntry(now time.Time, entry *cachedResourceEntry) bool {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.pending {
		return false
	}
	if entry.disposeAfter > 0 && !entry.lastAccess.IsZero() && now.Sub(entry.lastAccess) >= entry.disposeAfter {
		return true
	}
	if entry.maxAge > 0 && !entry.lastLoaded.IsZero() && now.Sub(entry.lastLoaded) >= entry.maxAge {
		return true
	}
	return false
}

func toPublicCachedState[T any](snapshot cachedResourceSnapshot) CachedResourceState[T] {
	value, _ := castCachedValue[T](snapshot.Value)
	return CachedResourceState[T]{
		Value:     value,
		Loading:   snapshot.Loading,
		Error:     snapshot.Error,
		Ready:     snapshot.Ready,
		Stale:     snapshot.Stale,
		UpdatedAt: snapshot.UpdatedAt,
	}
}

func castCachedValue[T any](value interface{}) (T, bool) {
	cast, ok := value.(T)
	if ok {
		return cast, true
	}

	var zero T
	return zero, false
}
