package ui

import (
	"github.com/monstercameron/GoWebComponents/v6/internal/errcode"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/query"
)

// UseAsyncMutation is the async sibling of UseMutation: the returned function applies the
// optimistic value immediately (re-rendering now) and reconciles in the background —
// committing the server result or rolling back on error and re-rendering again when it
// settles. This is the fire-and-forget optimistic action; UseMutation wraps the *blocking*
// query.Mutate, this one wraps query.MutateAsync.
func UseAsyncMutation[T any](parseCache *query.Cache, parseKey string) func(parseOptimistic T, parseFn func() (T, error)) {
	parseRerender := UseForceUpdate()
	return func(parseOptimistic T, parseFn func() (T, error)) {
		query.MutateAsync(parseCache, parseKey, parseOptimistic, containAsyncFn("UseAsyncMutation action", parseFn), func(query.Result[T]) { parseRerender() })
		parseRerender()
	}
}

// UseOptimistic returns the current value for key plus a setter that applies an optimistic
// value immediately and re-renders — GoWebComponents' answer to React's useOptimistic,
// backed by the query cache. The optimistic value is superseded by the next authoritative
// write (a mutation result or a query refresh) for the same key.
//
//	value, setOptimistic := ui.UseOptimistic[int](appCache, "likes")
//	onClick := func() {
//	    setOptimistic(value.Data + 1)                       // UI updates now
//	    go ui.UseAsyncMutation[int](appCache, "likes")(...) // server reconciles
//	}
func UseOptimistic[T any](parseCache *query.Cache, parseKey string) (query.Result[T], func(T)) {
	parseRerender := UseForceUpdate()
	parseSet := func(parseOptimistic T) {
		parseCache.Set(parseKey, parseOptimistic)
		parseRerender()
	}
	return query.Snapshot[T](parseCache, parseKey), parseSet
}

// UseAction binds a server action to key in the cache: it returns the current result and a
// run function that fires the action with an optimistic value and reconciles in the
// background (commit or rollback), re-rendering on settle. Pair run's fn with serverfn.Call
// to make a one-call optimistic server action.
//
//	res, runLike := ui.UseAction[int](appCache, "post/"+id+"/likes")
//	onClick := func() { runLike(res.Data+1, func() (int, error) { return serverfn.Call(...) }) }
func UseAction[T any](parseCache *query.Cache, parseKey string) (query.Result[T], func(parseOptimistic T, parseFn func() (T, error))) {
	parseRun := UseAsyncMutation[T](parseCache, parseKey)
	return query.Snapshot[T](parseCache, parseKey), parseRun
}

// containAsyncFn wraps a background callback so a panic is contained and reported as an ordinary error.
// The callbacks run on their own goroutines, where an escaping panic would end the whole process (in the
// browser the whole wasm program), and where merely recovering would leave the owning hook's state stuck
// in its running/loading phase because the settle step after the callback never runs.
func containAsyncFn[T any](parseSubject string, parseFn func() (T, error)) func() (T, error) {
	if parseFn == nil {
		return parseFn
	}
	return func() (parseValue T, parseErr error) {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				runtime.ContainPanic("ui", runtime.PanicPhaseAsync, parseSubject, parseRecovered)
				var parseZero T
				parseValue = parseZero
				parseErr = errcode.New(errcode.RuntimePanicAsync, "ui: "+parseSubject+" panicked", parseRecovered)
			}
		}()
		return parseFn()
	}
}
