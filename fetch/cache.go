//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

const cachedResourceAtomPrefix = "__fetch_cached_resource:"

type CacheOptions struct {
	StaleAfter time.Duration
}

type CachedResourceState[T any] struct {
	Value     T
	Loading   bool
	Error     error
	Ready     bool
	Stale     bool
	UpdatedAt time.Time
}

type CachedResource[T any] struct {
	get        func() CachedResourceState[T]
	reload     func()
	cancel     func()
	invalidate func()
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

type cachedResourceEntry struct {
	mu          sync.Mutex
	valueType   reflect.Type
	staleAfter  time.Duration
	lastLoaded  time.Time
	requestSeq  uint64
	pending     bool
	invalidated bool
	cancel      context.CancelFunc
}

var cachedResourceRegistry sync.Map

func UseCachedResource[T any](key string, loader func(context.Context) (T, error), options ...CacheOptions) CachedResource[T] {
	resolved := resolveCacheOptions(options)
	entry := getCachedResourceEntry(key)
	configureCachedResourceEntry[T](key, entry, resolved)

	snapshotAtom := state.UseAtom(cachedResourceAtomID(key), cachedResourceSnapshot{})
	snapshot := snapshotAtom.Get()

	ui.UseEffect(func() func() {
		if key == "" || loader == nil {
			return nil
		}

		startCachedLoad(key, entry, func(ctx context.Context) (interface{}, error) {
			return loader(ctx)
		}, false)
		return nil
	}, key, snapshot.Ready, snapshot.Stale, snapshot.Loading, snapshot.UpdatedAt, resolved.StaleAfter)

	return CachedResource[T]{
		get: func() CachedResourceState[T] {
			return toPublicCachedState[T](currentCachedSnapshot(key))
		},
		reload: func() {
			if key == "" || loader == nil {
				return
			}
			startCachedLoad(key, entry, func(ctx context.Context) (interface{}, error) {
				return loader(ctx)
			}, true)
		},
		cancel: func() {
			cancelCachedLoad(key)
		},
		invalidate: func() {
			InvalidateResource(key)
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

func (r CachedResource[T]) Get() CachedResourceState[T] {
	if r.get == nil {
		var zero CachedResourceState[T]
		return zero
	}

	return r.get()
}

func (r CachedResource[T]) Reload() {
	if r.reload != nil {
		r.reload()
	}
}

func (r CachedResource[T]) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r CachedResource[T]) Invalidate() {
	if r.invalidate != nil {
		r.invalidate()
	}
}

func (r CachedResource[T]) Set(value T) {
	if r.set != nil {
		r.set(value)
	}
}

func (r CachedResource[T]) Update(fn func(T) T) {
	if r.update != nil {
		r.update(fn)
	}
}

func InvalidateResource(key string) {
	if key == "" {
		return
	}

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

func resolveCacheOptions(options []CacheOptions) CacheOptions {
	if len(options) == 0 {
		return CacheOptions{}
	}
	return options[0]
}

func getCachedResourceEntry(key string) *cachedResourceEntry {
	if key == "" {
		return &cachedResourceEntry{}
	}

	raw, _ := cachedResourceRegistry.LoadOrStore(key, &cachedResourceEntry{})
	return raw.(*cachedResourceEntry)
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
	entry.mu.Unlock()
}

func startCachedLoad(key string, entry *cachedResourceEntry, loader func(context.Context) (interface{}, error), force bool) {
	if key == "" || entry == nil || loader == nil {
		return
	}

	entry.mu.Lock()
	current := currentCachedSnapshot(key)
	if !force && !shouldLoadCachedEntry(current, entry) {
		entry.mu.Unlock()
		return
	}
	if entry.pending {
		entry.mu.Unlock()
		return
	}
	entry.requestSeq++
	seq := entry.requestSeq
	ctx, cancel := context.WithCancel(context.Background())
	entry.cancel = cancel
	entry.pending = true
	entry.invalidated = false
	entry.mu.Unlock()

	updateCachedSnapshot(key, func(prev cachedResourceSnapshot) cachedResourceSnapshot {
		prev.Loading = true
		prev.Error = nil
		if prev.Ready {
			prev.Stale = true
		}
		return prev
	})

	go func(requestSeq uint64, requestCtx context.Context) {
		value, err := loader(requestCtx)

		entry.mu.Lock()
		if requestSeq != entry.requestSeq {
			entry.mu.Unlock()
			return
		}
		entry.pending = false
		entry.cancel = nil
		if err == nil && requestCtx.Err() == nil {
			entry.lastLoaded = time.Now()
		}
		stillInvalidated := entry.invalidated
		entry.mu.Unlock()

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
	}(seq, ctx)
}

func shouldLoadCachedEntry(snapshot cachedResourceSnapshot, entry *cachedResourceEntry) bool {
	if entry.pending {
		return false
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
	entry.cancel = nil
	entry.pending = false
	stillInvalidated := entry.invalidated
	entry.mu.Unlock()

	if cancel != nil {
		cancel()
	}

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
