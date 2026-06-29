package main

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type serverStatusCaptureWriter struct {
	http.ResponseWriter
	status     int
	hasWritten bool
}

// newServerStatusCaptureWriter initializes a response writer wrapper that tracks final status code.
func newServerStatusCaptureWriter(parseResponse http.ResponseWriter) *serverStatusCaptureWriter {
	return &serverStatusCaptureWriter{ResponseWriter: parseResponse, status: http.StatusOK}
}

// WriteHeader captures status codes before forwarding to the wrapped response writer.
func (parseW *serverStatusCaptureWriter) WriteHeader(parseStatus int) {
	parseW.status = parseStatus
	parseW.hasWritten = true
	parseW.ResponseWriter.WriteHeader(parseStatus)
}

// Write ensures status tracking defaults to 200 when handlers only write bodies.
func (parseW *serverStatusCaptureWriter) Write(parseData []byte) (int, error) {
	if !parseW.hasWritten {
		parseW.WriteHeader(http.StatusOK)
	}
	return parseW.ResponseWriter.Write(parseData)
}

// logServerRequestEvent emits structured page-entry and mutation-result logs with guardrails.
func (parseS *atlasServer) logServerRequestEvent(parseR *http.Request, parseStatus int, parseDuration time.Duration, parseHeaders http.Header) {
	if !parseS.cfg.LogsEnabled || parseR == nil {
		return
	}
	parsePath := strings.TrimSpace(parseR.URL.Path)
	parseMethod := strings.ToUpper(strings.TrimSpace(parseR.Method))
	parseDurationMS := max(parseDuration.Milliseconds(), 0)
	if parseMethod == http.MethodGet && !strings.HasPrefix(parsePath, "/api/") && !strings.HasPrefix(parsePath, "/assets/") {
		parseSurface := "public"
		if strings.HasPrefix(parsePath, "/app/") {
			parseSurface = "internal"
		}
		parseLoggedQuery := url.Values{}
		parseAllowedQueryKeys := map[string]bool{
			"q":                 true,
			"status":            true,
			"sort":              true,
			"warehouse":         true,
			"page":              true,
			atlasNoticeQueryKey: true,
		}
		for parseKey, parseValues := range parseR.URL.Query() {
			parseNormalizedKey := strings.TrimSpace(strings.ToLower(parseKey))
			if !parseAllowedQueryKeys[parseNormalizedKey] {
				continue
			}
			for _, parseValue := range parseValues {
				parseTrimmedValue := strings.TrimSpace(parseValue)
				if len(parseTrimmedValue) > 96 {
					parseTrimmedValue = parseTrimmedValue[:96]
				}
				parseLoggedQuery.Add(parseNormalizedKey, parseTrimmedValue)
			}
		}
		parseFields := map[string]any{
			"method":      parseMethod,
			"path":        parsePath,
			"surface":     parseSurface,
			"status":      parseStatus,
			"duration_ms": parseDurationMS,
		}
		if parseQuery := parseLoggedQuery.Encode(); parseQuery != "" {
			parseFields["query"] = parseQuery
		}
		if parseBootstrapBytes := strings.TrimSpace(parseHeaders.Get("X-Atlas-Bootstrap-Bytes")); parseBootstrapBytes != "" {
			parseFields["bootstrap_bytes"] = parseBootstrapBytes
		}
		if parseBootstrapMode := strings.TrimSpace(parseHeaders.Get("X-Atlas-Bootstrap-Mode")); parseBootstrapMode != "" {
			parseFields["bootstrap_mode"] = parseBootstrapMode
		}
		parseS.writeServerLogEvent("page.entry", parseFields)
		return
	}
	if parseMethod != http.MethodPost && parseMethod != http.MethodPut && parseMethod != http.MethodPatch && parseMethod != http.MethodDelete {
		return
	}
	if !strings.HasPrefix(parsePath, "/api/") {
		return
	}
	parseSegments := strings.Split(strings.Trim(parsePath, "/"), "/")
	parseMutationType := "mutation_unknown"
	parseMutationID := ""
	if len(parseSegments) >= 5 && parseSegments[0] == "api" && parseSegments[1] == "public" && parseSegments[2] == "products" {
		parseMutationID = parseSegments[3]
		switch parseSegments[4] {
		case "comments":
			parseMutationType = "public_comment_create"
		case "quote-requests":
			parseMutationType = "public_quote_request_create"
		case "restock-requests":
			parseMutationType = "public_restock_request_create"
		}
	}
	if len(parseSegments) >= 3 && parseSegments[0] == "api" && parseSegments[1] == "app" {
		switch parseSegments[2] {
		case "comments":
			if len(parseSegments) >= 5 && parseSegments[4] == "moderate" {
				parseMutationType = "internal_comment_moderate"
				parseMutationID = parseSegments[3]
			}
			if len(parseSegments) >= 4 && parseSegments[3] == "bulk-moderate" {
				parseMutationType = "internal_comment_bulk_moderate"
			}
		case "inventory":
			if len(parseSegments) >= 5 && parseSegments[4] == "update" {
				parseMutationType = "internal_inventory_update"
				parseMutationID = parseSegments[3]
			}
			if len(parseSegments) >= 5 && parseSegments[4] == "threshold" {
				parseMutationType = "internal_inventory_threshold_update"
				parseMutationID = parseSegments[3]
			}
		case "transfers":
			if len(parseSegments) == 3 {
				parseMutationType = "internal_transfer_create"
				parseMutationID = extractServerResourceID(parseHeaders, "/app/transfers/")
			}
		case "receiving":
			if len(parseSegments) >= 5 && parseSegments[4] == "reconcile" {
				parseMutationType = "internal_receiving_reconcile"
				parseMutationID = parseSegments[3]
			}
			if len(parseSegments) >= 5 && parseSegments[4] == "attachments" {
				parseMutationType = "internal_receiving_attachment_create"
				parseMutationID = parseSegments[3]
			}
		case "purchase-orders":
			if len(parseSegments) == 3 {
				parseMutationType = "internal_purchase_order_create"
				parseMutationID = extractServerResourceID(parseHeaders, "/app/purchase-orders/")
			}
			if len(parseSegments) >= 5 && parseSegments[4] == "status" {
				parseMutationType = "internal_purchase_order_status_update"
				parseMutationID = parseSegments[3]
			}
		case "products":
			if len(parseSegments) == 3 {
				parseMutationType = "internal_product_create"
				parseMutationID = extractServerResourceID(parseHeaders, "/app/products/")
			}
			if len(parseSegments) >= 5 && parseSegments[4] == "update" {
				parseMutationType = "internal_product_update"
				parseMutationID = parseSegments[3]
			}
			if len(parseSegments) >= 5 && parseSegments[4] == "delete" {
				parseMutationType = "internal_product_delete"
				parseMutationID = parseSegments[3]
			}
		case "saved-views":
			if len(parseSegments) == 3 {
				parseMutationType = "internal_saved_view_create"
			}
			if len(parseSegments) >= 4 && parseSegments[3] == "import" {
				parseMutationType = "internal_saved_view_import"
			}
		case "preferences":
			parseMutationType = "internal_preferences_save"
		}
	}
	if parseMutationType == "mutation_unknown" && parseStatus < http.StatusBadRequest {
		return
	}
	parseFields := map[string]any{
		"method":        parseMethod,
		"path":          parsePath,
		"status":        parseStatus,
		"duration_ms":   parseDurationMS,
		"mutation_type": parseMutationType,
	}
	if strings.TrimSpace(parseMutationID) != "" {
		parseFields["mutation_id"] = parseMutationID
	}
	if parseResultState := extractServerResultState(parseHeaders); parseResultState != "" {
		parseFields["result_state"] = parseResultState
	}
	parseS.writeServerLogEvent("mutation.result", parseFields)
}

// extractServerResourceID extracts one trailing identifier from a redirect Location path prefix.
func extractServerResourceID(parseHeaders http.Header, parsePathPrefix string) string {
	parseLocation := strings.TrimSpace(parseHeaders.Get("Location"))
	if parseLocation == "" {
		return ""
	}
	parseParsed, parseErr := url.Parse(parseLocation)
	if parseErr != nil {
		return ""
	}
	parsePath := strings.TrimSpace(parseParsed.Path)
	if !strings.HasPrefix(parsePath, parsePathPrefix) {
		return ""
	}
	parseResourceID := strings.Trim(strings.TrimPrefix(parsePath, parsePathPrefix), "/")
	if parseResourceID == "" || strings.Contains(parseResourceID, "/") {
		return ""
	}
	return parseResourceID
}

// extractServerResultState pulls stable result-state hints from redirect notices when available.
func extractServerResultState(parseHeaders http.Header) string {
	parseLocation := strings.TrimSpace(parseHeaders.Get("Location"))
	if parseLocation == "" {
		return ""
	}
	parseParsed, parseErr := url.Parse(parseLocation)
	if parseErr != nil {
		return ""
	}
	return strings.TrimSpace(parseParsed.Query().Get(atlasNoticeQueryKey))
}

// writeServerLogEvent writes one structured JSON log line when logging is enabled.
func (parseS *atlasServer) writeServerLogEvent(parseEvent string, parseFields map[string]any) {
	if !parseS.cfg.LogsEnabled {
		return
	}
	parsePayload := map[string]any{
		"event":   strings.TrimSpace(parseEvent),
		"service": "atlas-commerce-os",
		"time":    time.Now().UTC().Format(time.RFC3339Nano),
	}
	maps.Copy(parsePayload, parseFields)
	parseEncoded, parseErr := json.Marshal(parsePayload)
	if parseErr != nil {
		log.Printf(`{"event":"server_log_encode_failed","service":"atlas-commerce-os","error":%q}`, parseErr.Error())
		return
	}
	log.Print(string(parseEncoded))
}
