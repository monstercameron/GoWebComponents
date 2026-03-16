//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/router"
)

func main() {
	fmt.Println("OMI example starting...")

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", OverviewPage)
	r.Register("/data", DataPage)
	r.Register("/data/:id", DataDetailPage, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			records, err := loadHookRecords(ctx)
			if err != nil {
				return nil, err
			}

			slug := routeCtx.Params.Get("id")
			for _, record := range records {
				if recordSlug(record.Name) == slug {
					return router.Attrs{
						"slug":     slug,
						"name":     record.Name,
						"category": record.Category,
						"summary":  record.Summary,
						"hooks":    record.Hooks,
					}, nil
				}
			}

			return nil, fmt.Errorf("no record matched route slug %q", slug)
		},
		Loading: DataDetailRouteLoading,
		Error:   DataDetailRouteError,
	})
	r.Register("/search", SearchPage, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			records, err := loadHookRecords(ctx)
			if err != nil {
				return nil, err
			}

			queryTerm := routeCtx.Query.Get("q")
			return router.Attrs{
				"query":   queryTerm,
				"results": filterHookRecords(records, queryTerm),
			}, nil
		},
		Loading: SearchRouteLoading,
		Error:   SearchRouteError,
	})
	r.Register("/secure", SecurePage, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(120 * time.Millisecond):
			}

			if routeCtx.Query.Get("auth") != "true" {
				return nil, fmt.Errorf("access denied: open /secure?auth=true to simulate an authenticated session")
			}

			role := routeCtx.Query.Get("role")
			if role == "" {
				role = "admin"
			}

			return router.Attrs{
				"userName":  "Morgan Reconciler",
				"role":      role,
				"grantedBy": "route loader",
			}, nil
		},
		Loading: SecureRouteLoading,
		Error:   SecureRouteError,
	})
	r.Register("/playground", PlaygroundPage)
	r.Register("/playground/:mode", PlaygroundPage)
	r.Register("*", NotFoundPage)
	r.Mount("#app")

	fmt.Println("OMI example mounted")
	select {}
}
