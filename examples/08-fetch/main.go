//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func loadJSON[T any](ctx context.Context, url string) (T, error) {
	var zero T
	resultCh := fetch.Fetch(url, fetch.Options{})

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return zero, result.Err
		}

		payload, ok := result.Data.(string)
		if !ok || payload == "" {
			return zero, fmt.Errorf("empty fetch payload")
		}

		if err := json.Unmarshal([]byte(payload), &zero); err != nil {
			return zero, err
		}

		return zero, nil
	}
}

func UserCard(user User) ui.Node {
	return html.Div(
		html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm hover:bg-white/10 transition-all duration-300"},
		html.Div(
			html.Props{Class: "flex items-center space-x-4 mb-4"},
			html.Div(
				html.Props{Class: "w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-bold text-xl shadow-lg"},
				html.Text(string(user.Name[0])),
			),
			html.Div(
				html.Props{},
				html.H3(html.Props{Class: "text-lg font-bold text-white"}, html.Text(user.Name)),
				html.P(html.Props{Class: "text-sm text-blue-400"}, html.Text("@"+user.Username)),
			),
		),
		html.Div(
			html.Props{Class: "space-y-2 text-sm text-gray-400"},
			html.P(
				html.Props{Class: "flex items-center"},
				html.Span(html.Props{Class: "mr-2"}, html.Text("Email")),
				html.Text(user.Email),
			),
			html.P(
				html.Props{Class: "flex items-center"},
				html.Span(html.Props{Class: "mr-2"}, html.Text("Web")),
				html.Text(user.Website),
			),
		),
	)
}

func App() ui.Node {
	selectedUserID := ui.UseState(1)
	usersResource := fetch.UseCachedResource("users", func(ctx context.Context) ([]User, error) {
		return loadJSON[[]User](ctx, "https://jsonplaceholder.typicode.com/users")
	}, fetch.CacheOptions{StaleAfter: 20 * time.Second})
	summaryResource := fetch.UseCachedResource("users", func(ctx context.Context) ([]User, error) {
		return loadJSON[[]User](ctx, "https://jsonplaceholder.typicode.com/users")
	}, fetch.CacheOptions{StaleAfter: 20 * time.Second})
	usersState := usersResource.Get()
	summaryState := summaryResource.Get()

	detailCacheKey := fmt.Sprintf("user:%d", selectedUserID.Get())
	detailResource := fetch.UseCachedResource(detailCacheKey, func(ctx context.Context) (User, error) {
		select {
		case <-ctx.Done():
			return User{}, ctx.Err()
		case <-time.After(350 * time.Millisecond):
		}

		return loadJSON[User](ctx, fmt.Sprintf("https://jsonplaceholder.typicode.com/users/%d", selectedUserID.Get()))
	}, fetch.CacheOptions{StaleAfter: 15 * time.Second})
	detailState := detailResource.Get()
	deferredInsights := ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(ctx context.Context) (ui.Node, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(250 * time.Millisecond):
			}

			return html.Div(
				html.Props{Class: "mt-6 rounded-xl border border-cyan-500/20 bg-cyan-500/5 p-4 text-sm text-cyan-100"},
				html.P(html.Props{Class: "font-semibold"}, html.Text("Async UI primitive demo")),
				html.P(html.Props{Class: "mt-2 text-cyan-50/80"}, html.Text("This note is resolved through ui.Lazy, while the surrounding panels use ui.AsyncBoundary instead of open-coded loading branches.")),
			), nil
		},
		Dependencies: []interface{}{selectedUserID.Get()},
		Delay:        100 * time.Millisecond,
		Fallback: html.Div(
			html.Props{Class: "mt-6 rounded-xl border border-white/10 bg-white/5 p-4 animate-pulse"},
			html.Div(html.Props{Class: "h-4 w-40 rounded bg-white/10"}),
			html.Div(html.Props{Class: "mt-3 h-4 w-full rounded bg-white/10"}),
		),
		ErrorFallback: func(err error) ui.Node {
			return html.Div(
				html.Props{Class: "mt-6 rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-300"},
				html.Text("Deferred insights failed: "+err.Error()),
			)
		},
	})

	handleRefresh := ui.UseEvent(func() {
		usersResource.Reload()
		detailResource.Reload()
	})

	handleInvalidateShared := ui.UseEvent(func() {
		fetch.InvalidateResource("users")
	})

	handleCancel := ui.UseEvent(func() {
		usersResource.Cancel()
		detailResource.Cancel()
	})

	handleOptimisticRename := ui.UseEvent(func() {
		if !detailState.Ready {
			return
		}

		updatedName := detailState.Value.Name + " (local)"
		detailResource.Update(func(prev User) User {
			prev.Name = updatedName
			return prev
		})
		usersResource.Update(func(prev []User) []User {
			next := make([]User, len(prev))
			copy(next, prev)
			for i := range next {
				if next[i].ID == detailState.Value.ID {
					next[i].Name = updatedName
					break
				}
			}
			return next
		})
	})

	var content ui.Node
	{
		userElements := make([]ui.Node, len(usersState.Value))
		for i, user := range usersState.Value {
			selected := user.ID == selectedUserID.Get()
			userElements[i] = html.Div(
				html.Props{},
				html.Button(
					html.Props{
						OnClick: ui.UseEvent(func() {
							selectedUserID.Set(user.ID)
						}),
						Class: func() string {
							if selected {
								return "block w-full text-left ring-2 ring-cyan-400 rounded-xl"
							}
							return "block w-full text-left rounded-xl"
						}(),
					},
					UserCard(user),
				),
			)
		}
		content = html.Div(
			html.Props{Class: "grid grid-cols-1 gap-8 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]"},
			ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
				Pending: (usersState.Loading && !usersState.Ready) || (!usersState.Ready && len(usersState.Value) == 0),
				Error:   usersState.Error,
				Fallback: html.Div(
					html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2"},
					html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
					html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
					html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
				),
				ErrorFallback: func(err error) ui.Node {
					return html.Div(
						html.Props{Class: "bg-red-500/10 border border-red-500/20 p-6 rounded-xl mb-8 mx-auto max-w-2xl text-center"},
						html.P(html.Props{Class: "text-red-400 font-medium"}, html.Text("Error: "+err.Error())),
					)
				},
				Content: html.Div(
					html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2"},
					append([]ui.Node{func() ui.Node {
						if usersState.Loading && usersState.Ready {
							return html.Div(
								html.Props{Class: "sm:col-span-2 rounded-xl border border-cyan-500/20 bg-cyan-500/10 px-4 py-3 text-sm text-cyan-100"},
								html.Text("Revalidating the shared users query in the background while cached results stay on screen."),
							)
						}
						return html.Fragment()
					}()}, userElements...)...,
				),
			}),
			html.Div(
				html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm h-fit sticky top-6"},
				html.H2(html.Props{Class: "text-xl font-bold text-white mb-2"}, html.Text("Selected User")),
				html.P(html.Props{Class: "text-sm text-gray-400 mb-6"}, html.Text("This panel uses ui.AsyncBoundary around fetch.UseCachedResource state, and the note below is deferred through ui.Lazy.")),
				html.Div(
					html.Props{Class: "mb-6 rounded-xl border border-white/10 bg-black/20 p-4 text-sm text-gray-300"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text("Shared cache status")),
					html.P(html.Props{Class: "mt-2 text-gray-400"}, html.Text(fmt.Sprintf("%d users cached; background reloads are deduplicated across panels.", len(summaryState.Value)))),
					func() ui.Node {
						if summaryState.UpdatedAt.IsZero() {
							return html.P(html.Props{Class: "mt-2 text-gray-500"}, html.Text("No successful shared query yet."))
						}
						return html.P(html.Props{Class: "mt-2 text-gray-500"}, html.Text("Last shared update: "+summaryState.UpdatedAt.Format(time.Kitchen)))
					}(),
				),
				ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
					Pending: detailState.Loading && !detailState.Ready,
					Error:   detailState.Error,
					Fallback: html.Div(
						html.Props{},
						html.Div(html.Props{Class: "bg-white/5 border border-white/5 rounded-lg animate-pulse h-8 mb-4"}),
						html.Div(html.Props{Class: "bg-white/5 border border-white/5 rounded-lg animate-pulse h-24"}),
					),
					ErrorFallback: func(err error) ui.Node {
						return html.Div(
							html.Props{Class: "text-red-400 space-y-3"},
							html.P(html.Props{}, html.Text("Detail error: "+err.Error())),
							html.Button(html.Props{OnClick: ui.UseEvent(func() { detailResource.Reload() }), Class: "px-4 py-2 bg-red-500/20 border border-red-500/30 rounded-lg hover:bg-red-500/30 transition-colors"}, html.Text("Retry Detail")),
						)
					},
					Content: func() ui.Node {
						if !detailState.Ready {
							return html.P(html.Props{Class: "text-gray-500"}, html.Text("Select a user to inspect details."))
						}

						user := detailState.Value
						return html.Div(
							html.Props{Class: "space-y-3 text-sm text-gray-300"},
							func() ui.Node {
								if detailState.Loading && detailState.Ready {
									return html.Div(
										html.Props{Class: "rounded-lg border border-cyan-500/20 bg-cyan-500/10 px-3 py-2 text-xs text-cyan-100"},
										html.Text("Showing cached detail while the selected user refreshes in the background."),
									)
								}
								return html.Fragment()
							}(),
							html.H3(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(user.Name)),
							html.P(html.Props{}, html.Text("Username: @"+user.Username)),
							html.P(html.Props{}, html.Text("Email: "+user.Email)),
							html.P(html.Props{}, html.Text("Website: "+user.Website)),
							html.Div(
								html.Props{Class: "flex flex-wrap gap-3 pt-4"},
								html.Button(html.Props{OnClick: ui.UseEvent(func() { detailResource.Reload() }), Class: "px-4 py-2 bg-cyan-500 text-black rounded-lg hover:bg-cyan-400 transition-colors font-semibold"}, html.Text("Reload Detail")),
								html.Button(html.Props{OnClick: handleOptimisticRename, Class: "px-4 py-2 bg-violet-500 text-white rounded-lg hover:bg-violet-400 transition-colors font-semibold"}, html.Text("Optimistic Rename")),
								html.Button(html.Props{OnClick: ui.UseEvent(func() { detailResource.Cancel() }), Class: "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Cancel Detail")),
							),
						)
					}(),
				}),
				deferredInsights,
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
							if usersState.Loading || detailState.Loading {
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
			content,
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
