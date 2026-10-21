package fixture

import (
	"github.com/monstercameron/GoWebComponents/v7/html"
	. "github.com/monstercameron/GoWebComponents/v7/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// FlaggedRange builds a handler per item in a range body.
func FlaggedRange(parseItems []string) ui.Node {
	parseRows := []ui.Node{}
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Button(html.OnClick(func() { _ = parseItem })))
	}
	return nil
}

// FlaggedMapKeyed builds a handler inside a MapKeyed callback.
func FlaggedMapKeyed(parseItems []string) ui.Node {
	return Div(html.MapKeyed(parseItems, func(parseItem string) any { return parseItem }, func(parseItem string) ui.Node {
		return Button(OnClick(func() { _ = parseItem }))
	})...)
}

// FlaggedMap builds a handler inside a Map callback.
func FlaggedMap(parseItems []string) ui.Node {
	return Div(html.Map(parseItems, func(parseItem string) ui.Node {
		return Button(OnInput(func() { _ = parseItem }))
	})...)
}

// CleanTopLevel allocates its handler once, outside any list.
func CleanTopLevel() ui.Node {
	return Button(OnClick(func() {}))
}

// CleanVariableHandler passes a prebuilt handler into the loop.
func CleanVariableHandler(parseItems []string) ui.Node {
	parseHandler := ui.UseEvent(func() {})
	for range parseItems {
		_ = Button(OnClick(parseHandler))
	}
	return nil
}

// CleanNestedClosure builds the handler inside a closure the loop only defines.
func CleanNestedClosure(parseItems []string) ui.Node {
	for range parseItems {
		_ = func() ui.Node { return Button(OnClick(func() {})) }
	}
	return nil
}

// CleanMapKeyedComponent lets each row own its handlers.
func CleanMapKeyedComponent(parseItems []string) ui.Node {
	return Div(MapKeyedComponent(parseItems, func(parseItem string) any { return parseItem }, func(parseItem string) ui.Node {
		return Button(OnClick(func() { _ = parseItem }))
	})...)
}

// CleanUnrelatedCallback is a Map-named func from another package.
func CleanUnrelatedCallback(parseItems []string) {
	for range parseItems {
		other.OnClick(func() {})
	}
}
