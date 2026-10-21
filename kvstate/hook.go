package kvstate

import (
	"context"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// PersistedState is a state handle whose value is durably persisted via the
// configured backend. On native/SSR builds it degrades to plain in-memory state
// (the underlying ui.UseEffect is a no-op, so no engine is opened).
type PersistedState[T any] struct {
	get     func() T
	set     func(T)
	loading func() bool
	err     func() error
}

// Get returns the current value.
func (parsePs PersistedState[T]) Get() T { return parsePs.get() }

// Set updates the value in memory and persists it per the WriteStrategy.
func (parsePs PersistedState[T]) Set(parseValue T) { parsePs.set(parseValue) }

// Loading reports whether the initial hydrate from the backend is still pending.
func (parsePs PersistedState[T]) Loading() bool { return parsePs.loading() }

// Err returns the last persistence error, or nil.
func (parsePs PersistedState[T]) Err() error { return parsePs.err() }

// persistedSinks are the state setters a persisted-state hydration reports
// into. The hook wires them to ui.UseState handles; tests wire them to fakes so
// the hydration/Set ordering can be exercised without a renderer.
type persistedSinks[T any] struct {
	setValue   func(T)
	setLoading func(bool)
	setErr     func(error)
}

// reset returns the shared state to its pre-hydration shape for a new binding
// lifetime (first mount, or a key change): no engine, nothing parked, version 0.
func (parseS *boundAtomState) reset() {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	parseS.engine = nil
	parseS.pending, parseS.hasPending = nil, false
	parseS.version = 0
}

// persistPersistedValue writes parseValue durably through parseEngine. The
// version is claimed synchronously so concurrent Set calls get distinct,
// call-ordered versions; the save itself runs on its own goroutine.
func persistPersistedValue[T any](parseCtx context.Context, parseShared *boundAtomState, parseOpts Options, parseKey string, parseEngine *engine, parseValue T, parseSinks persistedSinks[T]) {
	parseNextVersion := parseShared.nextVersion()
	go func() {
		parseData, parseEncErr := parseOpts.Codec.Encode(parseValue)
		if parseCtx.Err() != nil {
			return
		}
		if parseEncErr != nil {
			parseSinks.setErr(parseEncErr)
			return
		}
		parseRec := Record{
			Key:       parseKey,
			Value:     parseData,
			Version:   parseNextVersion,
			UpdatedAt: time.Now().UnixMilli(),
		}
		if parseSaveErr := parseEngine.backend.Save(parseCtx, parseRec); parseSaveErr != nil {
			if parseCtx.Err() != nil {
				return
			}
			parseSinks.setErr(parseSaveErr)
			return
		}
		parseOpts.Strategy.OnWrite(parseCtx, parseKey, parseEngine.flush)
		if !parseOpts.ExternalInvalidation {
			broadcastCrossTab(parseOpts.Name, parseKey, parseNextVersion)
		}
	}()
}

// setPersistedValue applies a local Set: the in-memory value updates at once,
// then the write is persisted, or parked when hydration has not published the
// engine yet (so it is versioned after the stored row instead of against 0).
func setPersistedValue[T any](parseCtx context.Context, parseShared *boundAtomState, parseOpts Options, parseKey string, parseValue T, parseSinks persistedSinks[T]) {
	parseSinks.setValue(parseValue) // optimistic; UI updates immediately
	if parseCtx == nil || parseCtx.Err() != nil {
		return
	}
	parseEngine := parseShared.getEngineOrPark(parseValue)
	if parseEngine == nil {
		return // parked until hydration publishes the engine
	}
	persistPersistedValue(parseCtx, parseShared, parseOpts, parseKey, parseEngine, parseValue, parseSinks)
}

// runPersistedHydration opens the engine, loads the stored row, publishes the
// engine and installs the cross-tab watcher. A Set that landed before the engine
// was published is newer than the stored row: it is kept (not overwritten by the
// load) and persisted after the stored version is known.
func runPersistedHydration[T any](parseLifetime *bindingLifetime, parseShared *boundAtomState, parseOpts Options, parseKey string, parseInitial T, parseSinks persistedSinks[T]) {
	parseCtx := parseLifetime.parseContext
	parseEngine, parseErr := acquireEngine(parseCtx, parseOpts)
	if parseCtx.Err() != nil {
		return
	}
	if parseErr != nil {
		parseSinks.setErr(parseErr)
		parseSinks.setLoading(false)
		return
	}

	var parseLoaded T
	parseHasLoaded := false
	parseStoredVersion := int64(0)
	if parseOpts.Hydrate == HydrateEager {
		if parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseCtx, parseKey); parseLoadErr != nil {
			if parseCtx.Err() != nil {
				return
			}
			parseSinks.setErr(parseLoadErr)
		} else if parseFound {
			if parseCtx.Err() != nil {
				return
			}
			if parseDecodeErr := parseOpts.Codec.Decode(parseRec.Value, &parseLoaded); parseDecodeErr == nil {
				parseStoredVersion = parseRec.Version
				parseHasLoaded = true
			} else {
				// Surface schema drift / corrupted rows instead of
				// silently keeping the initial value with Err()==nil.
				parseSinks.setErr(parseDecodeErr)
			}
		} else {
			// A deleted desktop key still carries a version for safe recreation.
			if parseCtx.Err() != nil {
				return
			}
			parseStoredVersion = parseRec.Version
		}
	}
	if parseCtx.Err() != nil {
		return
	}
	parsePending, parseHasPending := parseShared.publishEngine(parseEngine, parseStoredVersion)
	if parseHasPending {
		if parsePendingValue, parseIsT := parsePending.(T); parseIsT {
			persistPersistedValue(parseCtx, parseShared, parseOpts, parseKey, parseEngine, parsePendingValue, parseSinks)
		}
	} else if parseHasLoaded {
		parseSinks.setValue(parseLoaded)
	}
	parseSinks.setLoading(false)

	parseLifetime.setStop(subscribeBinding(parseOpts, parseKey, func() {
		parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseCtx, parseKey)
		if parseCtx.Err() != nil {
			return
		}
		if parseLoadErr != nil {
			parseSinks.setErr(parseLoadErr)
			return
		}
		if !shouldApplyBindingRecord(parseShared.getVersion(), parseRec, parseOpts.Conflict) {
			return
		}
		if !parseFound {
			parseShared.setVersion(parseRec.Version)
			parseSinks.setValue(parseInitial)
			return
		}
		var parseValue T
		if parseDecodeErr := parseOpts.Codec.Decode(parseRec.Value, &parseValue); parseDecodeErr == nil {
			parseShared.setVersion(parseRec.Version)
			parseSinks.setValue(parseValue)
		} else {
			parseSinks.setErr(parseDecodeErr)
		}
	}))
}

// UsePersistedState returns state durably persisted under parseKey. It starts at
// parseInitial and asynchronously hydrates from the backend (the value is not
// available synchronously because the database opens asynchronously). Use
// Loading() to render a pending state if needed.
//
// A Set made before the initial hydrate finishes is newer than the stored row:
// it is kept and persisted once the stored version is known, rather than being
// overwritten by the load (the same rule BindAtom follows).
func UsePersistedState[T any](parseKey string, parseInitial T, parseOptions ...Options) PersistedState[T] {
	parseOpts := firstOptions(parseOptions).withDefaults()

	parseValState := ui.UseState(parseInitial)
	parseLoadingState := ui.UseState(true)
	parseErrState := ui.UseState[error](nil)
	parseSharedRef := ui.UseRef[*boundAtomState](nil)
	parseContextRef := ui.UseRef[context.Context](nil)

	if parseSharedRef.Get() == nil {
		parseSharedRef.Set(&boundAtomState{})
	}
	parseShared := parseSharedRef.Get()
	parseSinks := persistedSinks[T]{
		setValue:   parseValState.Set,
		setLoading: parseLoadingState.Set,
		setErr:     parseErrState.Set,
	}

	ui.UseEffect(func() func() {
		parseLifetime := newBindingLifetime(context.Background())
		parseContextRef.Set(parseLifetime.parseContext)
		parseShared.reset()
		parseLoadingState.Set(true)
		go runPersistedHydration(parseLifetime, parseShared, parseOpts, parseKey, parseInitial, parseSinks)
		return func() {
			parseLifetime.close()
			parseShared.reset()
			_ = parseOpts.Strategy.Close(context.Background())
		}
	}, parseKey)

	parseSet := func(parseValue T) {
		setPersistedValue(parseContextRef.Get(), parseShared, parseOpts, parseKey, parseValue, parseSinks)
	}

	return PersistedState[T]{
		get:     parseValState.Get,
		set:     parseSet,
		loading: parseLoadingState.Get,
		err:     parseErrState.Get,
	}
}

func firstOptions(parseOptions []Options) Options {
	if len(parseOptions) > 0 {
		return parseOptions[0]
	}
	return Options{}
}
