// Package typedroutesdemo shows `gwc routes gen` against a real package: declare route
// contracts as exported package vars, run `gwc routes gen`, and get typed Link* constructors
// in routes_gen.go that turn a typo'd path param into a compile error. `gwc routes check`
// (in CI) fails if routes_gen.go drifts from these contracts.
package typedroutesdemo

import "github.com/monstercameron/GoWebComponents/v4/router"

var (
	// HomeRoute is the marketing home page.
	HomeRoute = router.MustDefineRoute("/")
	// UserRoute is a user profile, parameterized by id.
	UserRoute = router.MustDefineRoute("/users/:id")
	// PostRoute is a post within a user, two params.
	PostRoute = router.MustDefineRoute("/users/:id/posts/:postID")
)
