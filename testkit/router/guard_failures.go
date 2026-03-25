//go:build js && wasm
// +build js,wasm

package routertest

import (
	"context"
	"strings"

	appRouter "github.com/monstercameron/GoWebComponents/router"
)

// BuildGuardBlocked returns a guard that always blocks navigation with reason.
func BuildGuardBlocked(reason string) appRouter.GuardFunc {
	parseReason := strings.TrimSpace(reason)
	return func(ctx appRouter.RouteContext) appRouter.GuardResult {
		if parseReason == "" {
			return appRouter.BlockNavigation("blocked by test helper")
		}
		return appRouter.BlockNavigation(parseReason)
	}
}

// BuildGuardRedirect returns a guard that always redirects navigation to path.
func BuildGuardRedirect(path string) appRouter.GuardFunc {
	parsePath := strings.TrimSpace(path)
	return func(ctx appRouter.RouteContext) appRouter.GuardResult {
		if parsePath == "" {
			return appRouter.RedirectNavigation("/")
		}
		return appRouter.RedirectNavigation(parsePath)
	}
}

// BuildAsyncGuardDenied returns an async guard that blocks with denied state.
func BuildAsyncGuardDenied(reason string) appRouter.AsyncGuardFunc {
	parseReason := strings.TrimSpace(reason)
	if parseReason == "" {
		parseReason = "denied by test helper"
	}
	return func(ctx context.Context, routeCtx appRouter.RouteContext) appRouter.GuardDecision {
		return appRouter.GuardDecision{
			Blocked: true,
			Denied:  true,
			Reason:  parseReason,
		}
	}
}
