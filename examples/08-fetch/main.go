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
	usersResource := fetch.UseResource(func(ctx context.Context) ([]User, error) {
		return loadJSON[[]User](ctx, "https://jsonplaceholder.typicode.com/users")
	})
	usersState := usersResource.Get()

	detailResource := fetch.UseResource(func(ctx context.Context) (User, error) {
		select {
		case <-ctx.Done():
			return User{}, ctx.Err()
		case <-time.After(350 * time.Millisecond):
		}

		return loadJSON[User](ctx, fmt.Sprintf("https://jsonplaceholder.typicode.com/users/%d", selectedUserID.Get()))
	}, selectedUserID.Get())
	detailState := detailResource.Get()

	handleRefresh := ui.UseEvent(func() {
		usersResource.Reload()
		detailResource.Reload()
	})

	handleCancel := ui.UseEvent(func() {
		usersResource.Cancel()
		detailResource.Cancel()
	})

	var content ui.Node
	if usersState.Error != nil {
		content = html.Div(
			html.Props{Class: "bg-red-500/10 border border-red-500/20 p-6 rounded-xl mb-8 mx-auto max-w-2xl text-center"},
			html.P(html.Props{Class: "text-red-400 font-medium"}, html.Text("Error: "+usersState.Error.Error())),
		)
	} else if usersState.Loading || !usersState.Ready || len(usersState.Value) == 0 {
		content = html.Div(
			html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"},
			html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
			html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
			html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
		)
	} else {
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
			html.Div(
				html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2"},
				userElements...,
			),
			html.Div(
				html.Props{Class: "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm h-fit sticky top-6"},
				html.H2(html.Props{Class: "text-xl font-bold text-white mb-2"}, html.Text("Selected User")),
				html.P(html.Props{Class: "text-sm text-gray-400 mb-6"}, html.Text("This panel uses fetch.UseResource with dependency-based reloads, explicit retry, and cancellation.")),
				func() ui.Node {
					if detailState.Loading {
						return html.Div(
							html.Props{},
							html.Div(html.Props{Class: "bg-white/5 border border-white/5 rounded-lg animate-pulse h-8 mb-4"}),
							html.Div(html.Props{Class: "bg-white/5 border border-white/5 rounded-lg animate-pulse h-24"}),
						)
					}
					if detailState.Error != nil {
						return html.Div(
							html.Props{Class: "text-red-400 space-y-3"},
							html.P(html.Props{}, html.Text("Detail error: "+detailState.Error.Error())),
							html.Button(html.Props{OnClick: ui.UseEvent(func() { detailResource.Reload() }), Class: "px-4 py-2 bg-red-500/20 border border-red-500/30 rounded-lg hover:bg-red-500/30 transition-colors"}, html.Text("Retry Detail")),
						)
					}
					if !detailState.Ready {
						return html.P(html.Props{Class: "text-gray-500"}, html.Text("Select a user to inspect details."))
					}

					user := detailState.Value
					return html.Div(
						html.Props{Class: "space-y-3 text-sm text-gray-300"},
						html.H3(html.Props{Class: "text-2xl font-semibold text-white"}, html.Text(user.Name)),
						html.P(html.Props{}, html.Text("Username: @"+user.Username)),
						html.P(html.Props{}, html.Text("Email: "+user.Email)),
						html.P(html.Props{}, html.Text("Website: "+user.Website)),
						html.Div(
							html.Props{Class: "flex flex-wrap gap-3 pt-4"},
							html.Button(html.Props{OnClick: ui.UseEvent(func() { detailResource.Reload() }), Class: "px-4 py-2 bg-cyan-500 text-black rounded-lg hover:bg-cyan-400 transition-colors font-semibold"}, html.Text("Reload Detail")),
							html.Button(html.Props{OnClick: ui.UseEvent(func() { detailResource.Cancel() }), Class: "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors border border-white/10"}, html.Text("Cancel Detail")),
						),
					)
				}(),
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
					html.Text("Demonstrating typed async resources with list loading, detail loading, retries, and cancellation."),
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
}
