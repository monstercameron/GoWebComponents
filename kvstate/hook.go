package kvstate

import (
	"context"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/ui"
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
	parseContextRef := ui.UseRef[context.Context](nil)

	ui.UseEffect(func() func() {
		parseLifetime := newBindingLifetime(context.Background())
		parseContextRef.Set(parseLifetime.parseContext)
		parseVersionRef.Set(0)
		parseLoadingState.Set(true)
		go func() {
			parseCtx := parseLifetime.parseContext
			parseEngine, parseErr := acquireEngine(parseCtx, parseOpts)
			if parseCtx.Err() != nil {
				return
			}
			if parseErr != nil {
				parseErrState.Set(parseErr)
				parseLoadingState.Set(false)
				return
			}
			parseEngineRef.Set(parseEngine)

			if parseOpts.Hydrate == HydrateEager {
				if parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseCtx, parseKey); parseLoadErr != nil {
					if parseCtx.Err() != nil {
						return
					}
					parseErrState.Set(parseLoadErr)
				} else if parseFound {
					if parseCtx.Err() != nil {
						return
					}
					var parseValue T
					if parseDecodeErr := parseOpts.Codec.Decode(parseRec.Value, &parseValue); parseDecodeErr == nil {
						parseVersionRef.Set(parseRec.Version)
						parseValState.Set(parseValue)
					} else {
						// Surface schema drift / corrupted rows instead of
						// silently keeping the initial value with Err()==nil.
						parseErrState.Set(parseDecodeErr)
					}
				} else {
					// A deleted desktop key still carries a version for safe recreation.
					if parseCtx.Err() != nil {
						return
					}
					parseVersionRef.Set(parseRec.Version)
				}
			}
			if parseCtx.Err() != nil {
				return
			}
			parseLoadingState.Set(false)

			parseLifetime.setStop(subscribeBinding(parseOpts, parseKey, func() {
				parseWatchCtx := parseCtx
				parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseWatchCtx, parseKey)
				if parseWatchCtx.Err() != nil {
					return
				}
				if parseLoadErr != nil {
					parseErrState.Set(parseLoadErr)
					return
				}
				if !shouldApplyBindingRecord(parseVersionRef.Get(), parseRec, parseOpts.Conflict) {
					return
				}
				if !parseFound {
					parseVersionRef.Set(parseRec.Version)
					parseValState.Set(parseInitial)
					return
				}
				var parseValue T
				if parseDecodeErr := parseOpts.Codec.Decode(parseRec.Value, &parseValue); parseDecodeErr == nil {
					parseVersionRef.Set(parseRec.Version)
					parseValState.Set(parseValue)
				} else {
					parseErrState.Set(parseDecodeErr)
				}
			}))
		}()
		return func() {
			parseLifetime.close()
			parseEngineRef.Set(nil)
			_ = parseOpts.Strategy.Close(context.Background())
		}
	}, parseKey)

	parseSet := func(parseValue T) {
		parseValState.Set(parseValue) // optimistic; UI updates immediately
		parseEngine := parseEngineRef.Get()
		if parseEngine == nil {
			return // engine not ready yet; value lives in memory until it is
		}
		parseCtx := parseContextRef.Get()
		if parseCtx == nil || parseCtx.Err() != nil {
			return
		}
		// Claim versions in input order rather than racing async save goroutines.
		parseNextVersion := parseVersionRef.Get() + 1
		parseVersionRef.Set(parseNextVersion)
		go func() {
			parseData, parseEncErr := parseOpts.Codec.Encode(parseValue)
			if parseCtx.Err() != nil {
				return
			}
			if parseEncErr != nil {
				parseErrState.Set(parseEncErr)
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
				parseErrState.Set(parseSaveErr)
				return
			}
			parseOpts.Strategy.OnWrite(parseCtx, parseKey, parseEngine.flush)
			if !parseOpts.ExternalInvalidation {
				broadcastCrossTab(parseOpts.Name, parseKey, parseNextVersion)
			}
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
