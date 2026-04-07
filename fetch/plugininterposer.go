package fetch

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
)

func init() {
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyFetch,
		Value: BuildFetchService(),
	})
}

type buildFetchService struct{}

// BuildFetchService returns one fetch-cache interposer service.
func BuildFetchService() pluginruntime.FetchService {
	return buildFetchService{}
}

// GetFetchSnapshot returns one normalized fetch-cache snapshot.
func (buildFetchService) GetFetchSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.FetchSnapshot, error) {
	getEntries := InspectCachedResources()
	buildSnapshot := pluginruntime.FetchSnapshot{
		Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
	}
	for _, getEntry := range getEntries {
		buildSnapshot.Entries = append(buildSnapshot.Entries, pluginruntime.FetchCacheEntry{
			Key:             getEntry.Key,
			Loading:         getEntry.Loading,
			Ready:           getEntry.Ready,
			Stale:           getEntry.Stale,
			LastError:       getEntry.LastError,
			UpdatedAt:       getEntry.UpdatedAt,
			LastLoaded:      getEntry.LastLoaded,
			SubscriberCount: getEntry.SubscriberCount,
			OwnerPaths:      append([]string(nil), getEntry.OwnerPaths...),
			ResumePolicy:    string(getEntry.ResumePolicy),
		})
	}
	if parseBudget.MaxItems > 0 && len(buildSnapshot.Entries) > parseBudget.MaxItems {
		buildSnapshot.Entries = append([]pluginruntime.FetchCacheEntry(nil), buildSnapshot.Entries[:parseBudget.MaxItems]...)
	}
	return buildSnapshot, nil
}

// ClearFetchEntry clears one fetch cache entry by key.
func (buildFetchService) ClearFetchEntry(parseKey string, parseOptions pluginruntime.CommandOptions) error {
	if parseKey == "" {
		return fmt.Errorf("fetch plugin interposer: cache key is required")
	}
	DisposeResource(parseKey)
	return nil
}

// RevalidateFetchEntry marks one fetch cache entry stale by key.
func (buildFetchService) RevalidateFetchEntry(parseKey string, parseOptions pluginruntime.CommandOptions) error {
	if parseKey == "" {
		return fmt.Errorf("fetch plugin interposer: cache key is required")
	}
	InvalidateResource(parseKey)
	return nil
}
