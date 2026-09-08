//go:build js && wasm

package atlas

import (
	"time"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func BootstrapCacheEntries(parsePayload Payload, parseOptions fetch.CacheOptions, parseUpdatedAt time.Time) []fetch.CacheBootstrapEntry {
	parseEntries := []fetch.CacheBootstrapEntry{{
		Key:          RoutePayloadResourceKey(parsePayload.Route.Path, resourceQueryValues(parsePayload.Route.Query)),
		Value:        clonePayloadForResourceCache(parsePayload),
		UpdatedAt:    parseUpdatedAt,
		ResumePolicy: fetch.CacheResumeTrustOnce,
		StaleAfter:   parseOptions.StaleAfter,
	}}
	visitPayloadRequestData(parsePayload, func(parseRequestURL string, parseDataKey string, parseValue any) {
		parseEntries = append(parseEntries, fetch.CacheBootstrapEntry{
			Key:          CachedRequestResourceKey(parseRequestURL, parseDataKey),
			Value:        parseValue,
			UpdatedAt:    parseUpdatedAt,
			ResumePolicy: fetch.CacheResumeTrustOnce,
			StaleAfter:   parseOptions.StaleAfter,
		})
	})
	return parseEntries
}

func SeedFetchCacheBootstrap(parseBootstrap *ui.SSRBootstrap, parsePayload Payload, parseOptions fetch.CacheOptions, parseUpdatedAt time.Time) {
	if parseBootstrap == nil {
		return
	}
	if parseBootstrap.Data == nil {
		parseBootstrap.Data = map[string]any{}
	}
	parseBootstrap.Data[fetch.CacheBootstrapDataKey] = fetch.CacheBootstrap{
		Entries: BootstrapCacheEntries(parsePayload, parseOptions, parseUpdatedAt),
	}
}
