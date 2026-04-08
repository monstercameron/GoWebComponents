//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Website  string `json:"website"`
}

func loadJSON[T any](parseCtx context.Context, parseUrl string) (T, error) {
	var parseZero T
	parseResultCh := fetch.Fetch(parseUrl, fetch.Options{})

	select {
	case <-parseCtx.Done():
		return parseZero, parseCtx.Err()
	case parseResult := <-parseResultCh:
		if parseResult.Err != nil {
			return parseZero, parseResult.Err
		}

		parsePayload, parseOk := parseResult.Data.(string)
		if !parseOk || parsePayload == "" {
			return parseZero, fmt.Errorf("empty fetch payload")
		}

		if parseErr := json.Unmarshal([]byte(parsePayload), &parseZero); parseErr != nil {
			return parseZero, parseErr
		}

		return parseZero, nil
	}
}

func UserCard(parseUser User) ui.Node {
	return html.Div(
		html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm hover:bg-white/10 transition-all duration-300"},
		html.Div(
			html.Props{Class: "flex items-center space-x-4 mb-4"},
			html.Div(
				html.Props{Class: "w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-bold text-xl shadow-lg"},
				html.Text(string(parseUser.Name[0])),
			),
			html.Div(
				html.Props{},
				html.H3(html.Props{Class: "text-lg font-bold text-white"}, html.Text(parseUser.Name)),
				html.P(html.Props{Class: "text-sm text-blue-400"}, html.Text("@"+parseUser.Username)),
			),
		),
		html.Div(
			html.Props{Class: "space-y-2 text-sm text-gray-400"},
			html.P(
				html.Props{Class: "flex items-center"},
				html.Span(html.Props{Class: "mr-2"}, html.Text("Email")),
				html.Text(parseUser.Email),
			),
			html.P(
				html.Props{Class: "flex items-center"},
				html.Span(html.Props{Class: "mr-2"}, html.Text("Web")),
				html.Text(parseUser.Website),
			),
		),
	)
}

func App() ui.Node {
	parseSelectedUserID := ui.UseState(1)
	parseUsersResource := fetch.UseCachedResource("users", func(parseCtx context.Context) ([]User, error) {
		return loadJSON[[]User](parseCtx, "https://jsonplaceholder.typicode.com/users")
	}, fetch.CacheOptions{StaleAfter: 20 * time.Second})
	parseSummaryResource := fetch.UseCachedResource("users", func(parseCtx2 context.Context) ([]User, error) {
		return loadJSON[[]User](parseCtx2, "https://jsonplaceholder.typicode.com/users")
	}, fetch.CacheOptions{StaleAfter: 20 * time.Second})
	parseUsersState := parseUsersResource.Get()
	parseSummaryState := parseSummaryResource.Get()

	parseDetailCacheKey := fmt.Sprintf("user:%d", parseSelectedUserID.Get())
	parseDetailResource := fetch.UseCachedResource(parseDetailCacheKey, func(parseCtx3 context.Context) (User, error) {
		select {
		case <-parseCtx3.Done():
			return User{}, parseCtx3.Err()
		case <-time.After(350 * time.Millisecond):
		}

		return loadJSON[User](parseCtx3, fmt.Sprintf("https://jsonplaceholder.typicode.com/users/%d", parseSelectedUserID.Get()))
	}, fetch.CacheOptions{StaleAfter: 15 * time.Second})
	parseDetailState := parseDetailResource.Get()
	parseDeferredInsights := ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(parseCtx4 context.Context) (ui.Node, error) {
			select {
			case <-parseCtx4.Done():
				return nil, parseCtx4.Err()
			case <-time.After(250 * time.Millisecond):
			}

			return html.Div(
				html.Props{Class: "mt-6 rounded-xl border border-cyan-500/20 bg-cyan-500/5 p-4 text-sm text-cyan-100"},
				html.P(html.Props{Class: "font-semibold"}, html.Text("Async UI primitive demo")),
				html.P(html.Props{Class: "mt-2 text-cyan-50/80"}, html.Text("This note is resolved through ui.Lazy, while the surrounding panels use ui.AsyncBoundary instead of open-coded loading branches.")),
			), nil
		},
		Dependencies: []interface{}{parseSelectedUserID.Get()},
		Delay:        100 * time.Millisecond,
		Fallback: html.Div(
			html.Props{Class: "mt-6 rounded-xl border border-white/10 bg-white/5 p-4 animate-pulse"},
			html.Div(html.Props{Class: "h-4 w-40 rounded bg-white/10"}),
			html.Div(html.Props{Class: "mt-3 h-4 w-full rounded bg-white/10"}),
		),
		ErrorFallback: func(parseErr error) ui.Node {
			return html.Div(
				html.Props{Class: "mt-6 rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-300"},
				html.Text("Deferred insights failed: "+parseErr.Error()),
			)
		},
	})

	handleRefresh := ui.UseEvent(func() {
		parseUsersResource.Reload()
		parseDetailResource.Reload()
	})

	handleInvalidateShared := ui.UseEvent(func() {
		fetch.InvalidateResource("users")
	})

	handleCancel := ui.UseEvent(func() {
		parseUsersResource.Cancel()
		parseDetailResource.Cancel()
	})

	handleOptimisticRename := ui.UseEvent(func() {
		if !parseDetailState.Ready {
			return
		}

		parseUpdatedName := parseDetailState.Value.Name + " (local)"
		parseDetailResource.Update(func(parsePrev User) User {
			parsePrev.Name = parseUpdatedName
			return parsePrev
		})
		parseUsersResource.Update(func(parsePrev2 []User) []User {
			parseNext := make([]User, len(parsePrev2))
			copy(parseNext, parsePrev2)
			for parseI := range parseNext {
				if parseNext[parseI].ID == parseDetailState.Value.ID {
					parseNext[parseI].Name = parseUpdatedName
					break
				}
			}
			return parseNext
		})
	})

	var parseContent ui.Node
	{
		parseUserElements := make([]ui.Node, len(parseUsersState.Value))
		for parseI2, parseUser := range parseUsersState.Value {
			isParseSelected := parseUser.ID == parseSelectedUserID.Get()
			parseUserElements[parseI2] = html.Div(
				html.Props{},
				html.Button(
					html.Props{
						OnClick: ui.UseEvent(func() {
							parseSelectedUserID.Set(parseUser.ID)
						}),
						Class: func() string {
							if isParseSelected {
								return "block w-full text-left ring-2 ring-cyan-400 rounded-xl"
							}
							return "block w-full text-left rounded-xl"
						}(),
					},
					UserCard(parseUser),
				),
			)
		}
		parseContent = html.Div(
			html.Props{Class: "grid grid-cols-1 gap-8 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]"},
			ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
				Pending: (parseUsersState.Loading && !parseUsersState.Ready) || (!parseUsersState.Ready && len(parseUsersState.Value) == 0),
				Error:   parseUsersState.Error,
				Fallback: html.Div(
					html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2"},
					html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
					html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
					html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
				),
				ErrorFallback: func(parseErr2 error) ui.Node {
					return html.Div(
						html.Props{Class: "bg-red-500/10 border border-red-500/20 p-6 rounded-xl mb-8 mx-auto max-w-2xl text-center"},
						html.P(html.Props{Class: "text-red-400 font-medium"}, html.Text("Error: "+parseErr2.Error())),
					)
				},
				Content: html.Div(
					html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2"},
					append([]ui.Node{func() ui.Node {
						if parseUsersState.Loading && parseUsersState.Ready {
							return html.Div(
								html.Props{Class: "sm:col-span-2 rounded-xl border border-cyan-500/20 bg-cyan-500/10 px-4 py-3 text-sm text-cyan-100"},
								html.Text("Revalidating the shared users query in the background while cached results stay on screen."),
							)
						}
						return html.Fragment()
					}()}, parseUserElements...)...,
				),
			}),
			html.Div(
				html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm h-fit sticky top-6"},
				html.H2(html.Props{Class: "text-xl font-bold text-white mb-2"}, html.Text("Selected User")),
				html.P(html.Props{Class: "text-sm text-gray-400 mb-6"}, html.Text("This panel uses ui.AsyncBoundary around fetch.UseCachedResource state, and the note below is deferred through ui.Lazy.")),
				html.Div(
					html.Props{Class: "mb-6 rounded-xl border border-white/10 bg-black/20 p-4 text-sm text-gray-300"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text("Shared cache status")),
					html.P(html.Props{Class: "mt-2 text-gray-400"}, html.Text(fmt.Sprintf("%d users cached; background reloads are deduplicated across panels.", len(parseSummaryState.Value)))),
					func() ui.Node {
						if parseSummaryState.UpdatedAt.IsZero() {
							return html.P(html.Props{Class: "mt-2 text-gray-500"}, html.Text("No successful shared query yet."))
						}
						return html.P(html.Props{Class: "mt-2 text-gray-500"}, html.Text("Last shared update: "+parseSummaryState.UpdatedAt.Format(time.Kitchen)))
					}(),
				),
				ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
					Pending: parseDetailState.Loading && !parseDetailState.Ready,
					Error:   parseDetailState.Error,
					Fallback: html.Div(
						html.Props{},
						html.Div(html.Props{Class: "bg-white/5 border border-white/5 rounded-lg animate-pulse h-8 mb-4"}),
						html.Div(html.Props{Class: "bg-white/5 border border-white/5 rounded-lg animate-pulse h-24"}),
					),
					ErrorFallback: func(parseErr3 error) ui.Node {
						return html.Div(
							html.Props{Class: "text-red-400 space-y-3"},
							html.P(html.Props{}, html.Text("Detail error: "+parseErr3.Error())),
							html.Button(html.Props{OnClick: ui.UseEvent(func() { parseDetailResource.Reload() }), Class: "px-4 py-2 bg-red-500/20 border border-red-500/30 rounded-lg hover:bg-red-500/30 transition-colors"}, html.Text("Retry Detail")),
						)
					},
					Content: func() ui.Node {
						if !parseDetailState.Ready {
							return html.P(html.Props{Class: "text-gray-500"}, html.Text("Select a user to inspect details."))
						}

						parseUser2 := parseDetailState.Value
						return html.Div(
							html.Props{Class: "space-y-3 text-sm text-gray-300"},
							func() ui.Node {
								if parseDetailState.Loading && parseDetailState.Ready {
									return html.Div(
										html.Props{Class: "rounded-lg border border-cyan-500/20 bg-cyan-500/10 px-3 py-2 text-xs text-cyan-100"},
										html.Text("Showing cached detail while the selected user refreshes in the background."),
									)
								}
								return html.Fragment()
							}(),
							html.H3(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(parseUser2.Name)),
							html.P(html.Props{}, html.Text("Username: @"+parseUser2.Username)),
							html.P(html.Props{}, html.Text("Email: "+parseUser2.Email)),
							html.P(html.Props{}, html.Text("Website: "+parseUser2.Website)),
							html.Div(
								html.Props{Class: "flex flex-wrap gap-3 pt-4"},
								html.Button(html.Props{OnClick: ui.UseEvent(func() { parseDetailResource.Reload() }), Class: "px-4 py-2 bg-cyan-500 text-black rounded-lg hover:bg-cyan-400 transition-colors font-semibold"}, html.Text("Reload Detail")),
								html.Button(html.Props{OnClick: handleOptimisticRename, Class: "px-4 py-2 bg-violet-500 text-white rounded-lg hover:bg-violet-400 transition-colors font-semibold"}, html.Text("Optimistic Rename")),
								html.Button(html.Props{OnClick: ui.UseEvent(func() { parseDetailResource.Cancel() }), Class: "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Cancel Detail")),
							),
						)
					}(),
				}),
				parseDeferredInsights,
			),
		)
	}

	return html.Div(
		html.Props{Class: "min-h-screen bg-[#0a0a0a] text-white py-12 px-4 sm:px-6 lg:px-8"},
		html.Div(
			html.Props{Class: "max-w-7xl mx-auto"},
			html.Div(
				html.Props{Class: "text-center mb-12"},
				html.H1(
					html.Props{Class: "text-4xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500 sm:text-5xl sm:tracking-tight lg:text-6xl"},
					html.Text("User Directory"),
				),
				html.P(
					html.Props{Class: "mt-5 max-w-xl mx-auto text-xl text-gray-400"},
					html.Text("Demonstrating shared cached queries, stale-while-revalidate refreshes, optimistic mutation, and explicit async boundaries."),
				),
				html.Div(
					html.Props{Class: "mt-8 flex flex-wrap items-center justify-center gap-3"},
					html.Button(
						html.Props{
							Class:   "inline-flex items-center px-8 py-3 border border-transparent text-base font-medium rounded-lg text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 shadow-lg shadow-purple-500/20 transition-all duration-200",
							OnClick: handleRefresh,
						},
						html.Text(func() string {
							if parseUsersState.Loading || parseDetailState.Loading {
								return "Refreshing..."
							}
							return "Reload Resources"
						}()),
					),
					html.Button(
						html.Props{
							Class:   "inline-flex items-center px-6 py-3 text-base font-medium rounded-lg text-cyan-100 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/20 transition-all duration-200",
							OnClick: handleInvalidateShared,
						},
						html.Text("Invalidate Shared Query"),
					),
					html.Button(
						html.Props{
							Class:   "inline-flex items-center px-6 py-3 text-base font-medium rounded-lg text-white bg-white/10 hover:bg-white/20 border border-white/10 transition-all duration-200",
							OnClick: handleCancel,
						},
						html.Text("Cancel In-Flight Work"),
					),
				),
			),
			parseContent,
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	exampleboot.WaitExampleRuntime()
}
