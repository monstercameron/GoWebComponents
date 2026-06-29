//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

const authAtomID = "catalog-router-guards-auth"
const dirtyAtomID = "catalog-router-guards-dirty"

var guardAuth bool
var guardDirty bool

func loginPageView() ui.Node {
	parseNav := router.UseNavigate()
	parseAuthed := state.UseAtom(authAtomID, false)
	return shared.ExamplePage(
		"router guards",
		"Block or redirect navigation through BeforeEnter and BeforeLeave",
		"Guards let routes stop navigation synchronously or redirect it before the router commits the transition.",
		shared.ExamplePanel("BeforeEnter redirect",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The protected route redirects here when auth is false.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Sign in and open editor", ui.UseEvent(func() {
					parseAuthed.Set(true)
					guardAuth = true
					parseNav.Navigate("/editor")
				})),
			),
		),
	)
}

func loginPage(router.Attrs) *router.Element {
	return ui.CreateElement(loginPageView)
}

func editorPageView() ui.Node {
	parseNav := router.UseNavigate()
	parseAuthed := state.UseAtom(authAtomID, false)
	parseDirty := state.UseAtom(dirtyAtomID, false)
	return shared.ExamplePage(
		"Protected editor",
		"Guarded route target",
		"Leaving while dirty is blocked. Clearing the dirty flag allows the route change to proceed.",
		shared.ExamplePanel("Guard actions",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Mark unsaved changes", ui.UseEvent(func() {
					parseDirty.Set(true)
					guardDirty = true
				})),
				shared.ExampleButton("Save changes", ui.UseEvent(func() {
					parseDirty.Set(false)
					guardDirty = false
				})),
				shared.ExampleButton("Try leaving to home", ui.UseEvent(func() { parseNav.Navigate("/") })),
				shared.ExampleButton("Sign out", ui.UseEvent(func() {
					parseAuthed.Set(false)
					parseDirty.Set(false)
					guardAuth = false
					guardDirty = false
					parseNav.Navigate("/")
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Signed in", map[bool]string{true: "Yes", false: "No"}[parseAuthed.Get()]),
				shared.ExampleStat("Unsaved changes", map[bool]string{true: "Yes", false: "No"}[parseDirty.Get()]),
			),
		),
	)
}

func editorPage(router.Attrs) *router.Element {
	return ui.CreateElement(editorPageView)
}

func homePageView() ui.Node {
	parseNav := router.UseNavigate()
	parseAuthed := state.UseAtom(authAtomID, false)
	return shared.ExamplePage(
		"Guard home",
		"Guard entry point",
		"Use the button below to try entering the protected editor route. The route guard will either redirect to login or allow navigation based on auth state.",
		shared.ExamplePanel("Entry flow",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Open protected editor", ui.UseEvent(func() { parseNav.Navigate("/editor") })),
				shared.ExampleButton("Reset auth", ui.UseEvent(func() {
					parseAuthed.Set(false)
					guardAuth = false
					guardDirty = false
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Signed in", map[bool]string{true: "Yes", false: "No"}[parseAuthed.Get()]),
				shared.ExampleStat("Protected path", "/editor"),
			),
		),
	)
}

func homePage(router.Attrs) *router.Element {
	return ui.CreateElement(homePageView)
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", homePage)
	parseR.Register("/login", loginPage)
	parseR.Register("/editor", editorPage, router.Options{
		BeforeEnter: func(parseCtx router.RouteContext) router.GuardResult {
			if !guardAuth {
				return router.RedirectNavigation("/login")
			}
			return router.AllowNavigation()
		},
		BeforeLeave: func(parseCurrent router.RouteContext, parseNext router.RouteContext) router.GuardResult {
			if guardDirty {
				return router.BlockNavigation("Unsaved changes are blocking navigation")
			}
			return router.AllowNavigation()
		},
	})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
