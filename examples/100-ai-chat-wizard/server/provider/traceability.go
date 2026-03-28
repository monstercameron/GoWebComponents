package provider

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// parseBuildTraceabilityMiddleware builds one request middleware that forwards traceability headers to provider APIs.
func parseBuildTraceabilityMiddleware() func(*http.Request, func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	return func(parseReq *http.Request, parseNext func(*http.Request) (*http.Response, error)) (*http.Response, error) {
		if parseReq != nil {
			parseApplyTraceabilityHeaders(parseReq.Context(), parseReq.Header)
		}
		return parseNext(parseReq)
	}
}

// parseApplyTraceabilityHeaders copies request and correlation metadata into one outbound HTTP header set.
func parseApplyTraceabilityHeaders(parseCtx context.Context, parseHeaders http.Header) {
	if parseHeaders == nil {
		return
	}
	parseTraceabilityHeaders := parseResolveTraceabilityHeaders(parseCtx)
	for parseHeaderKey, parseHeaderValue := range parseTraceabilityHeaders {
		if strings.TrimSpace(parseHeaderValue) == "" {
			continue
		}
		parseHeaders.Set(parseHeaderKey, strings.TrimSpace(parseHeaderValue))
	}
}

// parseResolveTraceabilityHeaders resolves one set of traceability headers from incoming or outgoing metadata.
func parseResolveTraceabilityHeaders(parseCtx context.Context) map[string]string {
	parseTraceabilityHeaders := map[string]string{}
	parseTraceabilityHeaderKeys := []string{
		"x-request-id",
		"x-correlation-id",
		"traceparent",
		"tracestate",
	}
	parseApplyMetadata := func(parseMD metadata.MD) {
		for _, parseHeaderKey := range parseTraceabilityHeaderKeys {
			if parseValues := parseMD.Get(parseHeaderKey); len(parseValues) > 0 {
				parseTraceabilityHeaders[parseHeaderKey] = strings.TrimSpace(parseValues[0])
			}
		}
	}
	if parseIncomingMD, parseOk := metadata.FromIncomingContext(parseCtx); parseOk {
		parseApplyMetadata(parseIncomingMD)
	}
	if parseOutgoingMD, parseOk := metadata.FromOutgoingContext(parseCtx); parseOk {
		parseApplyMetadata(parseOutgoingMD)
	}
	return parseTraceabilityHeaders
}
