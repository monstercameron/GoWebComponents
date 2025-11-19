//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Website  string `json:"website"`
}

func UserCard(user User) *dom.Element {
	return dom.Div(
		dom.Attrs{"class": "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm hover:bg-white/10 transition-all duration-300"},
		dom.Div(
			dom.Attrs{"class": "flex items-center space-x-4 mb-4"},
			dom.Div(
				dom.Attrs{"class": "w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-bold text-xl shadow-lg"},
				string(user.Name[0]),
			),
			dom.Div(
				nil,
				dom.H3(dom.Attrs{"class": "text-lg font-bold text-white"}, user.Name),
				dom.P(dom.Attrs{"class": "text-sm text-blue-400"}, "@"+user.Username),
			),
		),
		dom.Div(
			dom.Attrs{"class": "space-y-2 text-sm text-gray-400"},
			dom.P(
				dom.Attrs{"class": "flex items-center"},
				dom.Span(dom.Attrs{"class": "mr-2"}, "📧"),
				user.Email,
			),
			dom.P(
				dom.Attrs{"class": "flex items-center"},
				dom.Span(dom.Attrs{"class": "mr-2"}, "🌐"),
				user.Website,
			),
		),
	)
}

func App(_ dom.Attrs) *dom.Element {
	// State for users list
	users, setUsers := hooks.UseState([]User{})

	// Use the custom fetch hook
	getFetchState, refetch := hooks.UseFetch("https://jsonplaceholder.typicode.com/users")
	fetchState := getFetchState()

	// Update users when data changes
	hooks.UseEffect(func() func() {
		if fetchState.Data != nil {
			dataStr := fetchState.Data.(string)
			if dataStr != "" {
				var fetchedUsers []User
				if err := json.Unmarshal([]byte(dataStr), &fetchedUsers); err == nil {
					// Simulate network delay for better UX demonstration
					go func() {
						time.Sleep(500 * time.Millisecond)
						setUsers(fetchedUsers)
					}()
				}
			}
		}
		return nil
	}, fetchState.Data)

	handleRefresh := hooks.GoUseFunc(func(e dom.GoEvent) {
		setUsers([]User{}) // Clear current users to show loading state
		refetch()
	})

	return dom.Div(
		dom.Attrs{"class": "min-h-screen bg-[#0a0a0a] text-white py-12 px-4 sm:px-6 lg:px-8"},
		dom.Div(
			dom.Attrs{"class": "max-w-7xl mx-auto"},
			dom.Div(
				dom.Attrs{"class": "text-center mb-12"},
				dom.H1(
					dom.Attrs{"class": "text-4xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500 sm:text-5xl sm:tracking-tight lg:text-6xl"},
					"User Directory",
				),
				dom.P(
					dom.Attrs{"class": "mt-5 max-w-xl mx-auto text-xl text-gray-400"},
					"Demonstrating async data fetching with GoWebComponents hooks.",
				),
				dom.Button(
					dom.Attrs{
						"class":   "mt-8 inline-flex items-center px-8 py-3 border border-transparent text-base font-medium rounded-lg text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:opacity-90 shadow-lg shadow-purple-500/20 transition-all duration-200",
						"onclick": handleRefresh,
					},
					func() string {
						if fetchState.Loading {
							return "Refreshing..."
						}
						return "Refresh Data"
					}(),
				),
			),

			func() *dom.Element {
				if fetchState.Error != "" {
					return dom.Div(
						dom.Attrs{"class": "bg-red-500/10 border border-red-500/20 p-6 rounded-xl mb-8 mx-auto max-w-2xl text-center"},
						dom.P(dom.Attrs{"class": "text-red-400 font-medium"}, "Error: "+fetchState.Error),
					)
				}

				if fetchState.Loading || len(users()) == 0 {
					return dom.Div(
						dom.Attrs{"class": "grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"},
						// Skeleton loaders
						dom.Div(dom.Attrs{"class": "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
						dom.Div(dom.Attrs{"class": "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
						dom.Div(dom.Attrs{"class": "bg-white/5 border border-white/5 p-6 rounded-xl animate-pulse h-48"}),
					)
				}

				// Grid of user cards
				userElements := make([]interface{}, len(users()))
				for i, user := range users() {
					userElements[i] = UserCard(user)
				}

				return dom.Div(
					dom.Attrs{"class": "grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"},
					userElements...,
				)
			}(),
		),
	)
}

func main() {
	render.To(dom.CreateElement(App, nil), "body")
}
