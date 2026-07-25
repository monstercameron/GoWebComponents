package ui

import "github.com/monstercameron/GoWebComponents/v5/query"

// UseAsyncMutation is the async sibling of UseMutation: the returned function applies the
// optimistic value immediately (re-rendering now) and reconciles in the background —
// committing the server result or rolling back on error and re-rendering again when it
// settles. This is the fire-and-forget optimistic action; UseMutation wraps the *blocking*
// query.Mutate, this one wraps query.MutateAsync.
func UseAsyncMutation[T any](parseCache *query.Cache, parseKey string) func(parseOptimistic T, parseFn func() (T, error)) {
	parseRerender := UseForceUpdate()
	return func(parseOptimistic T, parseFn func() (T, error)) {
		query.MutateAsync(parseCache, parseKey, parseOptimistic, parseFn, func(query.Result[T]) { parseRerender() })
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
