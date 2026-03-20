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

func RoutePayloadResourceKey(path string, query url.Values) string {
	return atlasRouteResourcePrefix + routeDataResourceKey(path, query)
}

func CachedRequestResourceKey(requestURL string, dataKey string) string {
	return atlasRequestResourcePrefix + strings.TrimSpace(requestURL) + "::" + strings.TrimSpace(dataKey)
}

func PayloadResourceKeys(payload Payload) map[string]struct{} {
	keys := map[string]struct{}{
		RoutePayloadResourceKey(payload.Route.Path, resourceQueryValues(payload.Route.Query)): {},
	}
	visitPayloadRequestData(payload, func(requestURL string, dataKey string, _ any) {
		keys[CachedRequestResourceKey(requestURL, dataKey)] = struct{}{}
	})
	return keys
}

func fetchAtlasJSON[T any](ctx context.Context, requestURL string) (T, error) {
	var zero T
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return zero, err
	}
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return zero, fmt.Errorf("request %s returned status %d", requestURL, response.StatusCode)
	}

	var payload T
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return zero, err
	}
	return payload, nil
}

func useAtlasStartupPageResource(payload Payload) atlasCachedResource[any] {
	request, ok := StartupRequest(payload, "page")
	if !ok {
		return atlasCachedResource[any]{}
	}
	requestURL := strings.TrimSpace(request.URL)
	if requestURL == "" {
		return atlasCachedResource[any]{}
	}
	return useAtlasCachedResource(CachedRequestResourceKey(requestURL, "page"), func(ctx context.Context) (any, error) {
		return fetchAtlasJSON[any](ctx, requestURL)
	})
}

func resourceQueryValues(input map[string][]string) url.Values {
	values := url.Values{}
	for key, items := range input {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return values
}

func routeDataResourceKey(path string, query url.Values) string {
	filtered := url.Values{}
	for key, items := range query {
		trimmedKey := strings.TrimSpace(key)
		if strings.EqualFold(trimmedKey, atlasNoticeQueryResourceKey) || strings.EqualFold(trimmedKey, atlasBootstrapModeQueryResourceKey) {
			continue
		}
		for _, item := range items {
			filtered.Add(key, item)
		}
	}
	encoded := filtered.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func clonePayloadForResourceCache(input Payload) Payload {
	payload := input
	payload.Route.Query = cloneQuery(input.Route.Query)
	payload.Route.Params = cloneParams(input.Route.Params)
	payload.Data = cloneData(input.Data)
	payload.Requests = cloneRequests(input.Requests)
	payload.SavedViews = append([]SavedViewPayload(nil), input.SavedViews...)
	if input.User != nil {
		user := *input.User
		payload.User = &user
	}
	return payload
}

func visitPayloadRequestData(payload Payload, visit func(requestURL string, dataKey string, value any)) {
	for key, request := range payload.Requests {
		requestURL := strings.TrimSpace(request.URL)
		if requestURL == "" {
			continue
		}
		requestData := cloneData(request.Data)
		if len(requestData) == 0 {
			if value, ok := payload.Data[key]; ok {
				requestData[key] = value
			} else if key == "page" {
				if value, ok := payload.Data["page"]; ok {
					requestData["page"] = value
				}
			}
		}
		for dataKey, value := range requestData {
			visit(requestURL, dataKey, value)
		}
	}
}
