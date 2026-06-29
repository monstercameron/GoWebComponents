package kvstate

import (
	"context"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/ui"
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

// UsePersistedState returns state durably persisted under parseKey. It starts at
// parseInitial and asynchronously hydrates from the backend (the value is not
// available synchronously because the database opens asynchronously). Use
// Loading() to render a pending state if needed.
func UsePersistedState[T any](parseKey string, parseInitial T, parseOptions ...Options) PersistedState[T] {
	parseOpts := firstOptions(parseOptions).withDefaults()

	parseValState := ui.UseState(parseInitial)
	parseLoadingState := ui.UseState(true)
	parseErrState := ui.UseState[error](nil)
	parseEngineRef := ui.UseRef[*engine](nil)
	parseVersionRef := ui.UseRef[int64](0)

	ui.UseEffect(func() func() {
		var parseCancelWatch func()
		go func() {
			parseCtx := context.Background()
			parseEngine, parseErr := acquireEngine(parseCtx, parseOpts)
			if parseErr != nil {
				parseErrState.Set(parseErr)
				parseLoadingState.Set(false)
				return
			}
			parseEngineRef.Set(parseEngine)

			if parseOpts.Hydrate == HydrateEager {
				if parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseCtx, parseKey); parseLoadErr != nil {
					parseErrState.Set(parseLoadErr)
				} else if parseFound {
					var parseValue T
					if parseEngine != nil && parseOpts.Codec.Decode(parseRec.Value, &parseValue) == nil {
						parseVersionRef.Set(parseRec.Version)
						parseValState.Set(parseValue)
					}
				}
			}
			parseLoadingState.Set(false)

			parseCancelWatch = subscribeCrossTab(parseOpts.Name, parseKey, func() {
				parseWatchCtx := context.Background()
				parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseWatchCtx, parseKey)
				if parseLoadErr != nil || !parseFound {
					return
				}
				parseLocal := Record{Key: parseKey, Version: parseVersionRef.Get()}
				parseWinner := parseOpts.Conflict.Resolve(parseLocal, parseRec)
				if parseWinner.Version < parseVersionRef.Get() {
					return // local wins; ignore the incoming write
				}
				var parseValue T
				if parseOpts.Codec.Decode(parseRec.Value, &parseValue) == nil {
					parseVersionRef.Set(parseRec.Version)
					parseValState.Set(parseValue)
				}
			})
		}()
		return func() {
			if parseCancelWatch != nil {
				parseCancelWatch()
			}
			_ = parseOpts.Strategy.Close(context.Background())
		}
	}, parseKey)

	parseSet := func(parseValue T) {
		parseValState.Set(parseValue) // optimistic; UI updates immediately
		parseEngine := parseEngineRef.Get()
		if parseEngine == nil {
			return // engine not ready yet; value lives in memory until it is
		}
		go func() {
			parseCtx := context.Background()
			parseData, parseEncErr := parseOpts.Codec.Encode(parseValue)
			if parseEncErr != nil {
				parseErrState.Set(parseEncErr)
				return
			}
			parseNextVersion := parseVersionRef.Get() + 1
			parseVersionRef.Set(parseNextVersion)
			parseRec := Record{
				Key:       parseKey,
				Value:     parseData,
				Version:   parseNextVersion,
				UpdatedAt: time.Now().UnixMilli(),
			}
			if parseSaveErr := parseEngine.backend.Save(parseCtx, parseRec); parseSaveErr != nil {
				parseErrState.Set(parseSaveErr)
				return
			}
			parseOpts.Strategy.OnWrite(parseCtx, parseKey, parseEngine.flush)
			broadcastCrossTab(parseOpts.Name, parseKey, parseNextVersion)
		}()
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
