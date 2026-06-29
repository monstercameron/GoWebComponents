package kvstate

import (
	"context"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/state"
)

// BoundAtom is a global state.Atom whose value is durably persisted. Its
// Set/Update write through to the backend; reads come from the atom.
//
// Atoms expose no global out-of-component subscription, so — like
// ui.UsePersistedState — persistence is captured through this returned handle.
type BoundAtom[T any] struct {
	atom    state.Atom[T]
	set     func(T)
	loading func() bool
	err     func() error
}

// Get returns the current atom value.
func (parseB BoundAtom[T]) Get() T { return parseB.atom.Get() }

// Set updates the atom and persists the value.
func (parseB BoundAtom[T]) Set(parseValue T) { parseB.set(parseValue) }

// Update updates the atom from its previous value and persists the result.
func (parseB BoundAtom[T]) Update(parseFn func(T) T) { parseB.set(parseFn(parseB.atom.Get())) }

// Loading reports whether the initial hydrate is still pending.
func (parseB BoundAtom[T]) Loading() bool { return parseB.loading() }

// Err returns the last persistence error, or nil.
func (parseB BoundAtom[T]) Err() error { return parseB.err() }

// BindAtom binds a global atom to durable storage under parseKey. It loads the
// stored value asynchronously and applies it to the atom, then returns a
// write-through handle. Cross-tab writes are reflected back into the atom.
func BindAtom[T any](parseCtx context.Context, parseAtom state.Atom[T], parseKey string, parseOptions ...Options) BoundAtom[T] {
	parseOpts := firstOptions(parseOptions).withDefaults()
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	parseShared := &boundAtomState{loading: true}

	go func() {
		parseEngine, parseErr := acquireEngine(parseCtx, parseOpts)
		if parseErr != nil {
			parseShared.setError(parseErr)
			parseShared.setLoading(false)
			return
		}
		parseShared.setEngine(parseEngine)

		if parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseCtx, parseKey); parseLoadErr != nil {
			parseShared.setError(parseLoadErr)
		} else if parseFound {
			var parseValue T
			if parseOpts.Codec.Decode(parseRec.Value, &parseValue) == nil {
				parseShared.setVersion(parseRec.Version)
				parseAtom.Set(parseValue)
			}
		}
		parseShared.setLoading(false)

		subscribeCrossTab(parseOpts.Name, parseKey, func() {
			parseRec, parseFound, parseLoadErr := parseEngine.backend.Load(parseCtx, parseKey)
			if parseLoadErr != nil || !parseFound {
				return
			}
			parseLocal := Record{Key: parseKey, Version: parseShared.getVersion()}
			if parseOpts.Conflict.Resolve(parseLocal, parseRec).Version < parseShared.getVersion() {
				return
			}
			var parseValue T
			if parseOpts.Codec.Decode(parseRec.Value, &parseValue) == nil {
				parseShared.setVersion(parseRec.Version)
				parseAtom.Set(parseValue)
			}
		})
	}()

	parseSet := func(parseValue T) {
		parseAtom.Set(parseValue)
		parseEngine := parseShared.getEngine()
		if parseEngine == nil {
			return
		}
		go func() {
			parseData, parseEncErr := parseOpts.Codec.Encode(parseValue)
			if parseEncErr != nil {
				parseShared.setError(parseEncErr)
				return
			}
			parseNextVersion := parseShared.getVersion() + 1
			parseShared.setVersion(parseNextVersion)
			parseRec := Record{
				Key:       parseKey,
				Value:     parseData,
				Version:   parseNextVersion,
				UpdatedAt: time.Now().UnixMilli(),
			}
			if parseSaveErr := parseEngine.backend.Save(parseCtx, parseRec); parseSaveErr != nil {
				parseShared.setError(parseSaveErr)
				return
			}
			parseOpts.Strategy.OnWrite(parseCtx, parseKey, parseEngine.flush)
			broadcastCrossTab(parseOpts.Name, parseKey, parseNextVersion)
		}()
	}

	return BoundAtom[T]{
		atom:    parseAtom,
		set:     parseSet,
		loading: parseShared.getLoading,
		err:     parseShared.getError,
	}
}

// boundAtomState guards the mutable fields shared between the bind goroutine and
// the write-through closure.
type boundAtomState struct {
	mu      sync.Mutex
	engine  *engine
	version int64
	loading bool
	lastErr error
}

func (parseS *boundAtomState) setEngine(parseE *engine) {
	parseS.mu.Lock()
	parseS.engine = parseE
	parseS.mu.Unlock()
}

func (parseS *boundAtomState) getEngine() *engine {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return parseS.engine
}

func (parseS *boundAtomState) setVersion(parseV int64) {
	parseS.mu.Lock()
	parseS.version = parseV
	parseS.mu.Unlock()
}

func (parseS *boundAtomState) getVersion() int64 {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return parseS.version
}

func (parseS *boundAtomState) setLoading(parseV bool) {
	parseS.mu.Lock()
	parseS.loading = parseV
	parseS.mu.Unlock()
}

func (parseS *boundAtomState) getLoading() bool {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return parseS.loading
}

func (parseS *boundAtomState) setError(parseErr error) {
	parseS.mu.Lock()
	parseS.lastErr = parseErr
	parseS.mu.Unlock()
}

func (parseS *boundAtomState) getError() error {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return parseS.lastErr
}
