//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/router"
)

func main() {
	fmt.Println("OMI example starting...")

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", OverviewPage)
	parseR.Register("/data", DataPage)
	parseR.Register("/data/:id", DataDetailPage, router.Options{
		Loader: func(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
			parseRecords, parseErr := loadHookRecords(parseCtx)
			if parseErr != nil {
				return nil, parseErr
			}

			parseSlug := parseRouteCtx.Params.Get("id")
			for _, parseRecord := range parseRecords {
				if recordSlug(parseRecord.Name) == parseSlug {
					return router.Attrs{
						"slug":     parseSlug,
						"name":     parseRecord.Name,
						"category": parseRecord.Category,
						"summary":  parseRecord.Summary,
						"hooks":    parseRecord.Hooks,
					}, nil
				}
			}

			return nil, fmt.Errorf("no record matched route slug %q", parseSlug)
		},
		Loading: DataDetailRouteLoading,
		Error:   DataDetailRouteError,
	})
	parseR.Register("/search", SearchPage, router.Options{
		Loader: func(parseCtx2 context.Context, parseRouteCtx2 router.RouteContext) (router.Attrs, error) {
			parseRecords2, parseErr2 := loadHookRecords(parseCtx2)
			if parseErr2 != nil {
				return nil, parseErr2
			}

			parseQueryTerm := parseRouteCtx2.Query.Get("q")
			return router.Attrs{
				"query":   parseQueryTerm,
				"results": filterHookRecords(parseRecords2, parseQueryTerm),
			}, nil
		},
		Loading: SearchRouteLoading,
		Error:   SearchRouteError,
	})
	parseR.Register("/secure", SecurePage, router.Options{
		Loader: func(parseCtx3 context.Context, parseRouteCtx3 router.RouteContext) (router.Attrs, error) {
			select {
			case <-parseCtx3.Done():
				return nil, parseCtx3.Err()
			case <-time.After(120 * time.Millisecond):
			}

			if parseRouteCtx3.Query.Get("auth") != "true" {
				return nil, fmt.Errorf("access denied: open /secure?auth=true to simulate an authenticated session")
			}

			parseRole := parseRouteCtx3.Query.Get("role")
			if parseRole == "" {
				parseRole = "admin"
			}

			return router.Attrs{
				"userName":  "Morgan Reconciler",
				"role":      parseRole,
				"grantedBy": "route loader",
			}, nil
		},
		Loading: SecureRouteLoading,
		Error:   SecureRouteError,
	})
	parseR.Register("/playground", PlaygroundPage)
	parseR.Register("/playground/:mode", PlaygroundPage)
	parseR.Register("*", NotFoundPage)
	parseR.Mount("#app")

	fmt.Println("OMI example mounted")
	select {}
}
