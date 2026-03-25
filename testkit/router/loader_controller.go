//go:build js && wasm
// +build js,wasm

package routertest

import (
	"context"
	"net/url"

	appRouter "github.com/monstercameron/GoWebComponents/router"
	baseRender "github.com/monstercameron/GoWebComponents/testkit/render"
)

type LoaderAttempt struct {
	Index  int
	Path   string
	Query  url.Values
	Params map[string]string
}

// LoaderController provides deterministic control over route-loader attempts in tests.
type LoaderController struct {
	resource *baseRender.ResourceController[appRouter.Attrs]
	attempts []LoaderAttempt
}

// NewLoaderController creates one route-loader controller.
func NewLoaderController() *LoaderController {
	return &LoaderController{
		resource: baseRender.NewResourceController[appRouter.Attrs](),
	}
}

// Loader returns a router.LoaderFunc that blocks until the controller resolves it.
func (c *LoaderController) Loader() appRouter.LoaderFunc {
	return func(ctx context.Context, routeCtx appRouter.RouteContext) (appRouter.Attrs, error) {
		if c != nil && c.resource != nil {
			c.attempts = append(c.attempts, LoaderAttempt{
				Index:  c.resource.AttemptCount() + 1,
				Path:   routeCtx.Path,
				Query:  cloneURLValues(routeCtx.Query.Values()),
				Params: cloneStringMap(routeCtx.Params.Values()),
			})
		}
		return c.resource.Await(ctx)
	}
}

// Resolve completes the current loader attempt with a successful value.
func (c *LoaderController) Resolve(value appRouter.Attrs) {
	if c == nil || c.resource == nil {
		return
	}
	c.resource.Resolve(value)
}

// Reject completes the current loader attempt with an error.
func (c *LoaderController) Reject(err error) {
	if c == nil || c.resource == nil {
		return
	}
	c.resource.Reject(err)
}

// Cancel completes the current loader attempt with context cancellation.
func (c *LoaderController) Cancel() {
	if c == nil || c.resource == nil {
		return
	}
	c.resource.Cancel()
}

// Pending reports whether one loader attempt is currently stalled.
func (c *LoaderController) Pending() bool {
	if c == nil || c.resource == nil {
		return false
	}
	return c.resource.Pending()
}

// AttemptCount reports how many loader attempts have started.
func (c *LoaderController) AttemptCount() int {
	if c == nil || c.resource == nil {
		return 0
	}
	return c.resource.AttemptCount()
}

// Attempts returns a snapshot of all started loader attempts.
func (c *LoaderController) Attempts() []LoaderAttempt {
	if c == nil {
		return nil
	}
	return append([]LoaderAttempt(nil), c.attempts...)
}

// Started returns a channel that receives each started loader attempt index.
func (c *LoaderController) Started() <-chan int {
	if c == nil || c.resource == nil {
		return nil
	}
	return c.resource.Started()
}

func cloneURLValues(values url.Values) url.Values {
	if len(values) == 0 {
		return url.Values{}
	}
	cloned := make(url.Values, len(values))
	for key, items := range values {
		copied := make([]string, len(items))
		copy(copied, items)
		cloned[key] = copied
	}
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
