//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
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
				html.Span(html.Props{Class: "mr-2"}, html.Text("ðŸ“§")),
				html.Text(user.Email),
			),
			html.P(
				html.Props{Class: "flex items-center"},
				html.Span(html.Props{Class: "mr-2"}, html.Text("ðŸŒ")),
				html.Text(user.Website),
			),
		),
	)
}

func App() ui.Node {
	users := ui.UseState([]User{})
	resource := fetch.UseFetch("https://jsonplaceholder.typicode.com/users")
	fetchState := resource.Get()

	ui.UseEffect(func() func() {
		if fetchState.Data != nil {
			dataStr := fetchState.Data.(string)
			if dataStr != "" {
				var fetchedUsers []User
				if err := json.Unmarshal([]byte(dataStr), &fetchedUsers); err == nil {
					go func() {
						time.Sleep(500 * time.Millisecond)
						users.Set(fetchedUsers)
					}()
				}
			}
		}
		return nil
	}, fetchState.Data)

	handleRefresh := ui.UseEvent(func() {
		users.Set([]User{})
		resource.Refetch()
	})

	var content ui.Node
	if fetchState.Error != "" {
		content = html.Div(
			html.Props{Class: "bg-red-500/10 border border-red-500/20 p-6 rounded-xl mb-8 mx-auto max-w-2xl text-center"},
			html.P(html.Props{Class: "text-red-400 font-medium"}, html.Text("Error: "+fetchState.Error)),
		)
	} else if fetchState.Loading || len(users.Get()) == 0 {
		content = html.Div(
			html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"},
			html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
			html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
			html.Div(html.Props{Class: "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
		)
	} else {
		userElements := make([]ui.Node, len(users.Get()))
		for i, user := range users.Get() {
			userElements[i] = UserCard(user)
		}
		content = html.Div(
			html.Props{Class: "grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"},
			userElements...,
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
					html.Text("Demonstrating async data fetching with GoWebComponents hooks."),
				),
				html.Button(
					html.Props{
						Class:   "mt-8 inline-flex items-center px-8 py-3 border border-transparent text-base font-medium rounded-lg text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 shadow-lg shadow-purple-500/20 transition-all duration-200",
						OnClick: handleRefresh,
					},
					html.Text(func() string {
						if fetchState.Loading {
							return "Refreshing..."
						}
						return "Refresh Data"
					}()),
				),
			),
			content,
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(App), "body")
}
