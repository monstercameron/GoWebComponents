//go:build js && wasm
// +build js,wasm

package atlas

import (
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/ui"
)

func BootstrapCacheEntries(payload Payload, options fetch.CacheOptions, updatedAt time.Time) []fetch.CacheBootstrapEntry {
	entries := []fetch.CacheBootstrapEntry{{
		Key:          RoutePayloadResourceKey(payload.Route.Path, resourceQueryValues(payload.Route.Query)),
		Value:        clonePayloadForResourceCache(payload),
		UpdatedAt:    updatedAt,
		ResumePolicy: fetch.CacheResumeTrustOnce,
		StaleAfter:   options.StaleAfter,
	}}
	visitPayloadRequestData(payload, func(requestURL string, dataKey string, value any) {
		entries = append(entries, fetch.CacheBootstrapEntry{
			Key:          CachedRequestResourceKey(requestURL, dataKey),
			Value:        value,
			UpdatedAt:    updatedAt,
			ResumePolicy: fetch.CacheResumeTrustOnce,
			StaleAfter:   options.StaleAfter,
		})
	})
	return entries
}

func SeedFetchCacheBootstrap(bootstrap *ui.SSRBootstrap, payload Payload, options fetch.CacheOptions, updatedAt time.Time) {
	if bootstrap == nil {
		return
	}
	if bootstrap.Data == nil {
		bootstrap.Data = map[string]any{}
	}
	bootstrap.Data[fetch.CacheBootstrapDataKey] = fetch.CacheBootstrap{
		Entries: BootstrapCacheEntries(payload, options, updatedAt),
	}
}
