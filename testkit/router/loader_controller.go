//go:build js && wasm

package routertest

import (
	"context"
	"net/url"
	"sync"

	appRouter "github.com/monstercameron/GoWebComponents/v4/router"
	baseRender "github.com/monstercameron/GoWebComponents/v4/testkit/render"
)

type LoaderAttempt struct {
	Index     int
	Path      string
	Query     url.Values
	Params    map[string]string
	Cancelled bool
}

// LoaderController provides deterministic control over route-loader attempts in tests.
type LoaderController struct {
	resource *baseRender.ResourceController[appRouter.Attrs]
	mu       sync.Mutex
	attempts []LoaderAttempt
}

// NewLoaderController creates one route-loader controller.
func NewLoaderController() *LoaderController {
	return &LoaderController{
		resource: baseRender.NewResourceController[appRouter.Attrs](),
	}
}

// Loader returns a router.LoaderFunc that blocks until the controller resolves it.
func (parseC *LoaderController) Loader() appRouter.LoaderFunc {
	return func(parseCtx context.Context, parseRouteCtx appRouter.RouteContext) (appRouter.Attrs, error) {
		if parseC != nil && parseC.resource != nil {
			parseC.mu.Lock()
			parseC.attempts = append(parseC.attempts, LoaderAttempt{
				Index:  parseC.resource.AttemptCount() + 1,
				Path:   parseRouteCtx.Path,
				Query:  cloneURLValues(parseRouteCtx.Query.Values()),
				Params: cloneStringMap(parseRouteCtx.Params.Values()),
			})
			parseC.mu.Unlock()
		}
		return parseC.resource.Await(parseCtx)
	}
}

// Resolve completes the current loader attempt with a successful value.
func (parseC *LoaderController) Resolve(parseValue appRouter.Attrs) {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.Resolve(parseValue)
}

// Reject completes the current loader attempt with an error.
func (parseC *LoaderController) Reject(parseErr error) {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.Reject(parseErr)
}

// RejectLoaderFailure completes the current loader attempt with a typed loader failure.
func (parseC *LoaderController) RejectLoaderFailure(parsePath string, parseReason string) {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.Reject(baseRender.BuildLoaderFailureError(parsePath, parseReason))
}

// RejectRouteGuardFailure completes the current loader attempt with a typed guard failure.
func (parseC *LoaderController) RejectRouteGuardFailure(parsePath string, parseReason string) {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.Reject(baseRender.BuildRouteGuardFailureError(parsePath, parseReason))
}

// RejectCacheConflict completes the current loader attempt with a typed cache conflict failure.
func (parseC *LoaderController) RejectCacheConflict(parseEntity string) {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.RejectCacheConflict(parseEntity)
}

// RejectOfflineReplay completes the current loader attempt with a typed offline replay failure.
func (parseC *LoaderController) RejectOfflineReplay(parseEntity string, parseReason string) {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.RejectOfflineReplay(parseEntity, parseReason)
}

// Cancel completes the current loader attempt with context cancellation.
func (parseC *LoaderController) Cancel() {
	if parseC == nil || parseC.resource == nil {
		return
	}
	parseC.resource.Cancel()
}

// Pending reports whether one loader attempt is currently stalled.
func (parseC *LoaderController) Pending() bool {
	if parseC == nil || parseC.resource == nil {
		return false
	}
	return parseC.resource.Pending()
}

// AttemptCount reports how many loader attempts have started.
func (parseC *LoaderController) AttemptCount() int {
	if parseC == nil || parseC.resource == nil {
		return 0
	}
	return parseC.resource.AttemptCount()
}

// Attempts returns a snapshot of all started loader attempts.
func (parseC *LoaderController) Attempts() []LoaderAttempt {
	if parseC == nil {
		return nil
	}
	parseC.mu.Lock()
	parseSnapshot := append([]LoaderAttempt(nil), parseC.attempts...)
	parseC.mu.Unlock()
	parseResourceAttempts := parseC.resource.Attempts()
	for parseI := range parseSnapshot {
		if parseI < len(parseResourceAttempts) {
			parseSnapshot[parseI].Cancelled = parseResourceAttempts[parseI].Cancelled
		}
	}
	return parseSnapshot
}

// Started returns a channel that receives each started loader attempt index.
func (parseC *LoaderController) Started() <-chan int {
	if parseC == nil || parseC.resource == nil {
		return nil
	}
	return parseC.resource.Started()
}

func cloneURLValues(parseValues url.Values) url.Values {
	if len(parseValues) == 0 {
		return url.Values{}
	}
	parseCloned := make(url.Values, len(parseValues))
	for parseKey, parseItems := range parseValues {
		parseCopied := make([]string, len(parseItems))
		copy(parseCopied, parseItems)
		parseCloned[parseKey] = parseCopied
	}
	return parseCloned
}

func cloneStringMap(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return map[string]string{}
	}
	parseCloned := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}
