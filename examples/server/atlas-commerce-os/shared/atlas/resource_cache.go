package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	atlasNoticeQueryResourceKey        = "atlas_notice"
	atlasBootstrapModeQueryResourceKey = "atlas_bootstrap"
	atlasRouteResourcePrefix           = "atlas:route:"
	atlasRequestResourcePrefix         = "atlas:request:"
)

func RoutePayloadResourceKey(parsePath string, parseQuery url.Values) string {
	return atlasRouteResourcePrefix + routeDataResourceKey(parsePath, parseQuery)
}

func CachedRequestResourceKey(parseRequestURL string, parseDataKey string) string {
	return atlasRequestResourcePrefix + strings.TrimSpace(parseRequestURL) + "::" + strings.TrimSpace(parseDataKey)
}

func PayloadResourceKeys(parsePayload Payload) map[string]struct{} {
	parseKeys := map[string]struct{}{
		RoutePayloadResourceKey(parsePayload.Route.Path, resourceQueryValues(parsePayload.Route.Query)): {},
	}
	visitPayloadRequestData(parsePayload, func(parseRequestURL string, parseDataKey string, _ any) {
		parseKeys[CachedRequestResourceKey(parseRequestURL, parseDataKey)] = struct{}{}
	})
	return parseKeys
}

func fetchAtlasJSON[T any](parseCtx context.Context, parseRequestURL string) (T, error) {
	var parseZero T
	parseRequest, parseErr := http.NewRequestWithContext(parseCtx, http.MethodGet, parseRequestURL, nil)
	if parseErr != nil {
		return parseZero, parseErr
	}
	parseRequest.Header.Set("Accept", "application/json")

	parseResponse, parseErr := http.DefaultClient.Do(parseRequest)
	if parseErr != nil {
		return parseZero, parseErr
	}
	defer parseResponse.Body.Close()

	if parseResponse.StatusCode < http.StatusOK || parseResponse.StatusCode >= http.StatusMultipleChoices {
		return parseZero, fmt.Errorf("request %s returned status %d", parseRequestURL, parseResponse.StatusCode)
	}

	var parsePayload T
	if parseErr2 := json.NewDecoder(parseResponse.Body).Decode(&parsePayload); parseErr2 != nil {
		return parseZero, parseErr2
	}
	return parsePayload, nil
}

func useAtlasStartupPageResource(parsePayload Payload) atlasCachedResource[any] {
	parseRequest, parseOk := StartupRequest(parsePayload, "page")
	if !parseOk {
		return atlasCachedResource[any]{}
	}
	parseRequestURL := strings.TrimSpace(parseRequest.URL)
	if parseRequestURL == "" {
		return atlasCachedResource[any]{}
	}
	return useAtlasCachedResource(CachedRequestResourceKey(parseRequestURL, "page"), func(parseCtx context.Context) (any, error) {
		return fetchAtlasJSON[any](parseCtx, parseRequestURL)
	})
}

func resourceQueryValues(parseInput map[string][]string) url.Values {
	parseValues := url.Values{}
	for parseKey, parseItems := range parseInput {
		for _, parseItem := range parseItems {
			parseValues.Add(parseKey, parseItem)
		}
	}
	return parseValues
}

func routeDataResourceKey(parsePath string, parseQuery url.Values) string {
	parseFiltered := url.Values{}
	for parseKey, parseItems := range parseQuery {
		parseTrimmedKey := strings.TrimSpace(parseKey)
		if strings.EqualFold(parseTrimmedKey, atlasNoticeQueryResourceKey) || strings.EqualFold(parseTrimmedKey, atlasBootstrapModeQueryResourceKey) {
			continue
		}
		for _, parseItem := range parseItems {
			parseFiltered.Add(parseKey, parseItem)
		}
	}
	parseEncoded := parseFiltered.Encode()
	if parseEncoded == "" {
		return parsePath
	}
	return parsePath + "?" + parseEncoded
}

func clonePayloadForResourceCache(parseInput Payload) Payload {
	parsePayload := parseInput
	parsePayload.Route.Query = cloneQuery(parseInput.Route.Query)
	parsePayload.Route.Params = cloneParams(parseInput.Route.Params)
	parsePayload.Data = cloneData(parseInput.Data)
	parsePayload.Requests = cloneRequests(parseInput.Requests)
	parsePayload.SavedViews = append([]SavedViewPayload(nil), parseInput.SavedViews...)
	if parseInput.User != nil {
		parseUser := *parseInput.User
		parsePayload.User = &parseUser
	}
	return parsePayload
}

func visitPayloadRequestData(parsePayload Payload, parseVisit func(requestURL string, dataKey string, value any)) {
	for parseKey, parseRequest := range parsePayload.Requests {
		parseRequestURL := strings.TrimSpace(parseRequest.URL)
		if parseRequestURL == "" {
			continue
		}
		parseRequestData := cloneData(parseRequest.Data)
		if len(parseRequestData) == 0 {
			if parseValue, parseOk := parsePayload.Data[parseKey]; parseOk {
				parseRequestData[parseKey] = parseValue
			} else if parseKey == "page" {
				if parseValue2, parseOk2 := parsePayload.Data["page"]; parseOk2 {
					parseRequestData["page"] = parseValue2
				}
			}
		}
		for parseDataKey, parseValue3 := range parseRequestData {
			parseVisit(parseRequestURL, parseDataKey, parseValue3)
		}
	}
}
