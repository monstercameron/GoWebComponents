package ui

import (
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/query"
)

// UseQuery subscribes the calling component to a key in a query.Cache and returns the
// current result. It renders cached data immediately (a pure, no-fetch read) and, after
// commit, revalidates in the background via stale-while-revalidate — re-rendering the
// component when fresh data lands. Re-subscribes whenever key (or any extra dep) changes.
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
//
// When the fetcher closes over values that change independently of key (auth headers, a
// current user ID, filters), pass them as deps so the subscription re-runs with the fresh
// closure instead of silently using the stale one:
//
//	res := ui.UseQuery(appCache, "feed", func() (Feed, error) {
//	    return api.GetFeed(token) // token captured by the closure
//	}, token) // re-subscribe when token changes
func UseQuery[T any](parseCache *query.Cache, parseKey string, parseFetcher func() (T, error), parseDeps ...any) query.Result[T] {
	parseRerender := UseForceUpdate()
	parseEffectDeps := append([]any{parseKey}, parseDeps...)
	UseEffect(func() func() {
		query.SWR(parseCache, parseKey, parseFetcher, func(query.Result[T]) { parseRerender() })
		return nil
	}, parseEffectDeps...)
	return query.Snapshot[T](parseCache, parseKey)
}

// suspenseQueryBox holds one UseSuspenseQuery subscription's resume signal and the deps it was
// started for. It carries no result data — every value is read back from the mutex-guarded cache
// via query.Snapshot, so the loader goroutine and the render thread never race on a shared field.
type suspenseQueryBox struct {
	deps []any
	done chan struct{}
}

// UseSuspenseQuery is the suspending sibling of UseQuery: instead of returning a Result the caller
// must switch on, it returns the data directly, suspending the render until the data is ready and
// throwing the error to the nearest ErrorBoundary on failure. Wrap the subtree in an AsyncBoundary
// (for the loading fallback) and an ErrorBoundary (for failures); the component body then reads the
// value unconditionally — the React use(promise) / Solid createResource shape.
//
//	func UserCard(props UserProps) ui.Node {
//	    user := ui.UseSuspenseQuery(appCache, "user/"+props.ID, func() (User, error) {
//	        return api.GetUser(props.ID)
//	    })
//	    return Text(user.Name) // no Status switch — AsyncBoundary shows the fallback while loading
//	}
//
// Like UseQuery, pass deps when the fetcher closes over values that change independently of key;
// changing key or any dep starts a fresh load (and re-suspends until it resolves).
func UseSuspenseQuery[T any](parseCache *query.Cache, parseKey string, parseFetcher func() (T, error), parseDeps ...any) T {
	parseRerender := UseForceUpdate()
	parseBoxRef := UseRef((*suspenseQueryBox)(nil))

	parseWant := append([]any{parseKey}, parseDeps...)
	parseBox := parseBoxRef.Get()

	// Every value is read from the mutex-guarded cache, never from the loader goroutine's box —
	// query.Fetch's write and Snapshot's read are synchronized by the cache mutex, so there is no
	// shared mutable field between the goroutine and the render thread (the box's done channel is
	// only created here and closed by the goroutine, which is safe concurrently with a receive).
	parseSnap := query.Snapshot[T](parseCache, parseKey)

	if parseBox == nil || !suspenseDepsEqual(parseBox.deps, parseWant) {
		parseBox = &suspenseQueryBox{deps: parseWant, done: make(chan struct{})}
		parseBoxRef.Set(parseBox)

		// A settled snapshot — cached success (even if stale, to avoid a re-suspend flash) or a
		// recorded error — resolves the suspension immediately. Otherwise load in the background
		// (cache-aware + deduped) and resume when the fetch completes. close+rerender are DEFERRED
		// so a panicking fetcher (recovered just below) can't leave the component suspended forever.
		if parseSnap.Status == query.StatusSuccess || parseSnap.Status == query.StatusError {
			close(parseBox.done)
		} else {
			parseStarted := parseBox
			go func() {
				defer func() {
					close(parseStarted.done)
					parseRerender()
				}()
				defer runtime.RecoverContainedPanic("ui", "UseSuspenseQuery loader")
				query.Fetch(parseCache, parseKey, parseFetcher) // writes the cache under its own lock
			}()
		}
	}

	parseValue := SuspenseValue[T]{
		Value:  parseSnap.Data,
		Ready:  parseSnap.Status == query.StatusSuccess,
		Done:   parseBox.done,
		Reason: "query:" + parseKey,
	}
	if parseSnap.Status == query.StatusError {
		parseValue.Error = parseSnap.Err
	}
	return Await(parseValue)
}

// suspenseDepsEqual reports whether two dep lists are shallowly equal (Object.is-style, matching
// hook conventions). It is panic-safe: a non-comparable dep is treated as changed so the query
// re-loads conservatively rather than crashing the render.
func suspenseDepsEqual(parseA, parseB []any) (parseEqual bool) {
	if len(parseA) != len(parseB) {
		return false
	}
	defer func() {
		if recover() != nil {
			parseEqual = false
		}
	}()
	for parseI := range parseA {
		if parseA[parseI] != parseB[parseI] {
			return false
		}
	}
	return true
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
