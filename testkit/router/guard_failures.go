//go:build js && wasm
// +build js,wasm

package routertest

import (
	"context"
	"strings"

	appRouter "github.com/monstercameron/GoWebComponents/router"
)

// BuildGuardBlocked returns a guard that always blocks navigation with reason.
func BuildGuardBlocked(parseReason string) appRouter.GuardFunc {
	parseReason := strings.TrimSpace(parseReason)
	return func(parseCtx appRouter.RouteContext) appRouter.GuardResult {
		if parseReason == "" {
			return appRouter.BlockNavigation("blocked by test helper")
		}
		return appRouter.BlockNavigation(parseReason)
	}
}

// BuildGuardRedirect returns a guard that always redirects navigation to path.
func BuildGuardRedirect(parsePath string) appRouter.GuardFunc {
	parsePath := strings.TrimSpace(parsePath)
	return func(parseCtx appRouter.RouteContext) appRouter.GuardResult {
		if parsePath == "" {
			return appRouter.RedirectNavigation("/")
		}
		return appRouter.RedirectNavigation(parsePath)
	}
}

// BuildAsyncGuardDenied returns an async guard that blocks with denied state.
func BuildAsyncGuardDenied(parseReason string) appRouter.AsyncGuardFunc {
	parseReason := strings.TrimSpace(parseReason)
	if parseReason == "" {
		parseReason = "denied by test helper"
	}
	return func(parseCtx context.Context, parseRouteCtx appRouter.RouteContext) appRouter.GuardDecision {
		return appRouter.GuardDecision{
			Blocked: true,
			Denied:  true,
			Reason:  parseReason,
		}
	}
}
