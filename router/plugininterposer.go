//go:build js && wasm

package router

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
)

func init() {
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyRoute,
		Value: BuildRouteService(),
	})
}

type buildRouteService struct{}

// BuildRouteService returns one router-backed route service.
func BuildRouteService() pluginruntime.RouteService {
	return buildRouteService{}
}

// GetRouteSnapshot returns one normalized current-route snapshot.
func (buildRouteService) GetRouteSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.RouteSnapshot, error) {
	getInspection := InspectCurrentRoute()
	buildQuery := make(map[string][]string, len(getInspection.Query))
	for parseKey, parseValues := range getInspection.Query {
		buildQuery[parseKey] = append([]string(nil), parseValues...)
	}
	buildSnapshot := pluginruntime.RouteSnapshot{
		Meta:    pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
		Path:    getInspection.Path,
		Query:   buildQuery,
		Params:  copyParams(getInspection.Params),
		Loading: getInspection.Loading,
		LastRedirect: pluginruntime.RouteRedirect{
			Cause: getInspection.LastRedirect.Cause,
			From:  getInspection.LastRedirect.From,
			To:    getInspection.LastRedirect.To,
		},
		Metadata: pluginruntime.RouteMetadata{
			Title:        getInspection.Metadata.Title,
			Description:  getInspection.Metadata.Description,
			CanonicalURL: getInspection.Metadata.CanonicalURL,
		},
	}
	for _, getStack := range getInspection.Stack {
		buildSnapshot.Stack = append(buildSnapshot.Stack, pluginruntime.RouteStackEntry{
			ID:             getStack.ID,
			Path:           getStack.Path,
			Params:         copyParams(getStack.Params),
			HasLoader:      getStack.HasLoader,
			HasBeforeEnter: getStack.HasBeforeEnter,
			HasBeforeLeave: getStack.HasBeforeLeave,
			Metadata: pluginruntime.RouteMetadata{
				Title:        getStack.Metadata.Title,
				Description:  getStack.Metadata.Description,
				CanonicalURL: getStack.Metadata.CanonicalURL,
			},
		})
	}
	for _, getLoader := range getInspection.Loaders {
		buildSnapshot.Loaders = append(buildSnapshot.Loaders, pluginruntime.RouteLoaderEntry{
			Key:     getLoader.Key,
			Path:    getLoader.Path,
			Pending: getLoader.Pending,
			HasData: getLoader.HasData,
			Error:   getLoader.Error,
		})
	}
	if parseBudget.MaxItems > 0 && len(buildSnapshot.Loaders) > parseBudget.MaxItems {
		buildSnapshot.Loaders = append([]pluginruntime.RouteLoaderEntry(nil), buildSnapshot.Loaders[:parseBudget.MaxItems]...)
	}
	return buildSnapshot, nil
}

// NavigateRoute navigates to one path.
func (buildRouteService) NavigateRoute(parsePath string, parseOptions pluginruntime.CommandOptions) error {
	Navigate(parsePath)
	return nil
}

// RevalidateRoute forces the current route to revalidate.
func (buildRouteService) RevalidateRoute(parseOptions pluginruntime.CommandOptions) error {
	Revalidate()
	return nil
}

// RetryRouteLoader retries one active route loader by key.
func (buildRouteService) RetryRouteLoader(parseKey string, parseOptions pluginruntime.CommandOptions) error {
	if parseErr := RetryLoader(parseKey); parseErr != nil {
		return fmt.Errorf("router plugin interposer: %w", parseErr)
	}
	return nil
}
