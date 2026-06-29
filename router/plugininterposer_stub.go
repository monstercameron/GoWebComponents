//go:build !js || !wasm

package router

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
)

type buildRouteService struct{}

// BuildRouteService returns one native stub route service.
func BuildRouteService() pluginruntime.RouteService {
	return buildRouteService{}
}

// GetRouteSnapshot returns an empty native stub route snapshot.
func (buildRouteService) GetRouteSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.RouteSnapshot, error) {
	return pluginruntime.RouteSnapshot{
		Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDNative, false),
	}, nil
}

// NavigateRoute reports that route commands are unavailable on native builds.
func (buildRouteService) NavigateRoute(parsePath string, parseOptions pluginruntime.CommandOptions) error {
	return fmt.Errorf("router plugin interposer: navigate is unavailable on native builds")
}

// RevalidateRoute reports that revalidation is unavailable on native builds.
func (buildRouteService) RevalidateRoute(parseOptions pluginruntime.CommandOptions) error {
	return fmt.Errorf("router plugin interposer: revalidate is unavailable on native builds")
}

// RetryRouteLoader reports that loader retry is unavailable on native builds.
func (buildRouteService) RetryRouteLoader(parseKey string, parseOptions pluginruntime.CommandOptions) error {
	return fmt.Errorf("router plugin interposer: loader retry is unavailable on native builds")
}
