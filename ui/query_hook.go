package ui

import "github.com/monstercameron/GoWebComponents/query"

// UseQuery subscribes the calling component to a key in a query.Cache and returns the
// current result. It renders cached data immediately (a pure, no-fetch read) and, after
// commit, revalidates in the background via stale-while-revalidate — re-rendering the
// component when fresh data lands. Re-subscribes whenever key changes.
//
// The fetcher is an ordinary func() (T, error): no special async plumbing, and the same
// call works against a server-seeded cache during SSR.
//
//	func UserCard(props UserProps) ui.Node {
//	    res := ui.UseQuery(appCache, "user/"+props.ID, func() (User, error) {
//	        return api.GetUser(props.ID)
//	    })
//	    if res.Status == query.StatusLoading {
//	        return Spinner()
//	    }
//	    return Text(res.Data.Name) // updates in place when the background refresh resolves
//	}
func UseQuery[T any](parseCache *query.Cache, parseKey string, parseFetcher func() (T, error)) query.Result[T] {
	parseRerender := UseForceUpdate()
	UseEffect(func() func() {
		query.SWR(parseCache, parseKey, parseFetcher, func(query.Result[T]) { parseRerender() })
		return nil
	}, parseKey)
	return query.Snapshot[T](parseCache, parseKey)
}

// UseMutation returns a typed mutate function bound to key in a query.Cache. Calling it
// applies the optimistic value to the cache immediately, runs fn, commits fn's result on
// success or rolls back to the exact prior state on error, then re-renders the component
// with the outcome. This is the one-call optimistic-update path.
//
//	like := ui.UseMutation[int](appCache, "post/"+id+"/likes")
//	onClick := func() {
//	    like(current+1, func() (int, error) { return api.Like(id) }) // optimistic +1, auto-rollback
//	}
func UseMutation[T any](parseCache *query.Cache, parseKey string) func(parseOptimistic T, parseFn func() (T, error)) query.Result[T] {
	parseRerender := UseForceUpdate()
	return func(parseOptimistic T, parseFn func() (T, error)) query.Result[T] {
		parseRes := query.Mutate(parseCache, parseKey, parseOptimistic, parseFn)
		parseRerender()
		return parseRes
	}
}
