package atlas

import (
	"context"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// These small custom hooks keep hook calls visible as named render-time
// helpers. Atlas has a number of reusable element builders whose component
// body is supplied to ui.CreateElement; routing the primitive calls through
// these helpers preserves the same render-time ownership while keeping the
// builders' event closures free of primitive hook calls.
func useAtlasForm[T any](parseInitial T) ui.Form[T] { return ui.UseForm(parseInitial) }

func useAtlasDeferredValue[T any](parseValue T) T { return ui.UseDeferredValue(parseValue) }

func useAtlasDebounced[T any](parseValue T, parseDelay time.Duration) ui.Debounced[T] {
	return ui.UseDebounced(parseValue, parseDelay)
}

func useAtlasEvent(parseFn any) ui.Handler { return ui.UseEvent(parseFn) }

func useAtlasID() string { return ui.UseId() }

func useAtlasPrevious[T any](parseValue T) ui.Previous[T] { return ui.UsePrevious(parseValue) }

func useAtlasReducer[S any, A any](parseReducer func(S, A) S, parseInitial S) ui.Reducer[S, A] {
	return ui.UseReducer(parseReducer, parseInitial)
}

func useAtlasUIState[T any](parseInitial T) ui.State[T] { return ui.UseState(parseInitial) }

func useAtlasLazyNode(parseLoader func(context.Context) (ui.Node, error), parseDeps ...any) ui.LazyNode {
	return ui.UseLazyNode(parseLoader, parseDeps...)
}

func useAtlasCompositeNavigation(parseItems []ui.CompositeItem, parseOptions ...ui.CompositeNavigationOptions) ui.CompositeNavigation {
	return ui.UseCompositeNavigation(parseItems, parseOptions...)
}
