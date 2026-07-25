//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/query"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// userCache is one shared query cache (a package var). Keyed reads de-duplicate and stay warm until
// they go stale, so switching back to an already-fetched key serves the cached value without
// re-running the fetcher.
var userCache = query.New(query.WithStaleTime(30 * time.Second))

type user struct {
	Name  string
	Email string
}

// useQueryExample demonstrates ui.UseQuery: a keyed cached read whose fetcher only runs on a cache
// miss. The "fetcher runs" counter proves that re-selecting a cached user does not refetch.
func useQueryExample() ui.Node {
	parseID := ui.UseState("1")
	parseFetcherRuns := ui.UseRef(0)

	parseResult := ui.UseQuery(userCache, "user/"+parseID.Get(), func() (user, error) {
		parseFetcherRuns.Set(parseFetcherRuns.Get() + 1)
		return user{Name: "User " + parseID.Get(), Email: "user" + parseID.Get() + "@example.com"}, nil
	}, parseID.Get())

	parsePick1 := ui.UseEvent(func() { parseID.Set("1") })
	parsePick2 := ui.UseEvent(func() { parseID.Set("2") })
	parsePick3 := ui.UseEvent(func() { parseID.Set("3") })
	parseInvalidate := ui.UseEvent(func() { userCache.Invalidate("user/" + parseID.Get()) })

	return shared.ExamplePage(
		"ui.UseQuery",
		"Keyed cache with de-duplication and invalidation",
		"Switch between users: the first time a user is selected the fetcher runs once; selecting an already-cached user serves it without refetching. Invalidate marks the current key stale so the next read refetches. ui.UseMutation adds optimistic writes with rollback.",
		shared.ExamplePanel("Cached user read",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("User 1", parsePick1),
				shared.ExampleButton("User 2", parsePick2),
				shared.ExampleButton("User 3", parsePick3),
				shared.ExampleButton("Invalidate current", parseInvalidate),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-4"},
				shared.ExampleStat("Name", parseResult.Data.Name),
				shared.ExampleStat("Email", parseResult.Data.Email),
				shared.ExampleStat("Status", parseResult.Status.String()),
				shared.ExampleStat("Fetcher runs", fmt.Sprintf("%d", parseFetcherRuns.Get())),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(useQueryExample))
	exampleboot.WaitExampleRuntime()
}
