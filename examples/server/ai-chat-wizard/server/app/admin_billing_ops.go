package app

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseRequireAdminBillingUserScope enforces admin-role scope for one billing-target user id.
func (parseS *chatServer) parseRequireAdminBillingUserScope(parseCtx context.Context, parseUserID int64, parseSliceKey string) (parseAdminAccessScope, error) {
	if parseUserID <= 0 {
		return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "user id is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, parseSliceKey)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	if _, hasParseUser := parseScope.userIDs[parseUserID]; !hasParseUser {
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "target user outside workspace-admin scope")
	}
	return parseScope, nil
}

// parseBuildAdminBillingAccessOverrideEntry maps one billing override row into protobuf form.
func parseBuildAdminBillingAccessOverrideEntry(parseRow parseBillingAccessOverrideRow) *chatpb.AdminBillingAccessOverrideEntry {
	return &chatpb.AdminBillingAccessOverrideEntry{
		Id:            parseRow.ID,
		CustomerId:    parseRow.CustomerID,
		OverrideKey:   parseRow.OverrideKey,
		OverrideValue: parseRow.OverrideValue,
		Reason:        parseRow.Reason,
		IsEnabled:     parseRow.IsEnabled,
		StartsAt:      parseRow.StartsAt,
		EndsAt:        parseRow.EndsAt,
		ActorUserId:   parseRow.ActorUserID,
		CreatedAt:     parseRow.CreatedAt,
		UpdatedAt:     parseRow.UpdatedAt,
	}
}

// parseBuildAdminBillingEventEntry maps one billing event row into protobuf form.
func parseBuildAdminBillingEventEntry(parseRow parseBillingEventRow) *chatpb.AdminBillingEventEntry {
	return &chatpb.AdminBillingEventEntry{
		Id:               parseRow.ID,
		CustomerId:       parseRow.CustomerID,
		SubscriptionId:   parseRow.SubscriptionID,
		InvoiceId:        parseRow.InvoiceID,
		EventType:        parseRow.EventType,
		EventSource:      parseRow.EventSource,
		EventSummary:     parseRow.EventSummary,
		EventPayloadJson: parseRow.EventPayloadJSON,
		ActorUserId:      parseRow.ActorUserID,
		CreatedAt:        parseRow.CreatedAt,
	}
}

// parseBuildAdminBillingDunningEventEntry maps one dunning-event row into protobuf form.
func parseBuildAdminBillingDunningEventEntry(parseRow parseBillingDunningEventRow) *chatpb.AdminBillingDunningEventEntry {
	return &chatpb.AdminBillingDunningEventEntry{
		Id:             parseRow.ID,
		CustomerId:     parseRow.CustomerID,
		SubscriptionId: parseRow.SubscriptionID,
		InvoiceId:      parseRow.InvoiceID,
		Status:         parseRow.Status,
		AttemptCount:   parseRow.AttemptCount,
		FailureReason:  parseRow.FailureReason,
		NextAttemptAt:  parseRow.NextAttemptAt,
		ResolvedAt:     parseRow.ResolvedAt,
		CreatedAt:      parseRow.CreatedAt,
		UpdatedAt:      parseRow.UpdatedAt,
	}
}

// parseBuildAdminBillingDunningEventEntryFromBillingEvent maps one billing-event row into a fallback dunning-event protobuf entry.
func parseBuildAdminBillingDunningEventEntryFromBillingEvent(parseRow parseBillingEventRow) *chatpb.AdminBillingDunningEventEntry {
	parseStatus := "open"
	parseResolvedAt := ""
	parseEventType := strings.ToLower(strings.TrimSpace(parseRow.EventType))
	if strings.Contains(parseEventType, "resolved") {
		parseStatus = "resolved"
		parseResolvedAt = parseRow.CreatedAt
	}
	return &chatpb.AdminBillingDunningEventEntry{
		Id:             parseRow.ID,
		CustomerId:     parseRow.CustomerID,
		SubscriptionId: parseRow.SubscriptionID,
		InvoiceId:      parseRow.InvoiceID,
		Status:         parseStatus,
		AttemptCount:   0,
		FailureReason:  parseRow.EventSummary,
		NextAttemptAt:  "",
		ResolvedAt:     parseResolvedAt,
		CreatedAt:      parseRow.CreatedAt,
		UpdatedAt:      parseRow.CreatedAt,
	}
}

// ListAdminBillingAccessOverrides returns typed billing access overrides for one admin-target user.
func (parseS *chatServer) ListAdminBillingAccessOverrides(parseCtx context.Context, parseReq *chatpb.ListAdminBillingAccessOverridesRequest) (*chatpb.ListAdminBillingAccessOverridesResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminBillingAccessOverrides"))
	var parseUserID int64
	var parseOverrideKey string
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseOverrideKey = strings.TrimSpace(parseReq.GetOverrideKey())
		parseListQuery = parseReq.GetListQuery()
	}
	parseListQueryShape := parseBuildAdminListQueryShape(0, parseListQuery)
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.billing.access_overrides")
	if parseErr != nil {
		return nil, parseErr
	}
	parseLogger.Info("rpc.ListAdminBillingAccessOverrides: slice fetch", slog.Int64("admin_user_id", parseScope.adminUserID), slog.Int64("target_user_id", parseUserID))
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"billing-access-overrides",
		"Admin billing access overrides viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListAdminBillingAccessOverrides: store unavailable")
		return &chatpb.ListAdminBillingAccessOverridesResponse{}, nil
	}
	parseOverrideRows, parseErr := parseS.store.parseListAdminBillingAccessOverridesByUser(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminBillingAccessOverrides: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin billing access overrides: %v", parseErr)
	}
	parseOverrideRows = parseFilterAdminBillingAccessOverrideRows(parseOverrideRows, parseOverrideKey, parseListQueryShape.parseSearch)
	parseSortAdminBillingAccessOverrideRows(parseOverrideRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
	parseOverrideRows = parseApplyAdminSliceWindow(parseOverrideRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	parseResponse := &chatpb.ListAdminBillingAccessOverridesResponse{
		Overrides: make([]*chatpb.AdminBillingAccessOverrideEntry, 0, len(parseOverrideRows)),
	}
	for _, parseOverrideRow := range parseOverrideRows {
		parseResponse.Overrides = append(parseResponse.Overrides, parseBuildAdminBillingAccessOverrideEntry(parseOverrideRow))
	}
	parseLogger.Info("rpc.ListAdminBillingAccessOverrides: complete", slog.Int("count", len(parseResponse.Overrides)))
	return parseResponse, nil
}

// ListAdminBillingEvents returns typed billing event-inspection rows for one admin-target user.
func (parseS *chatServer) ListAdminBillingEvents(parseCtx context.Context, parseReq *chatpb.ListAdminBillingEventsRequest) (*chatpb.ListAdminBillingEventsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminBillingEvents"))
	var parseUserID int64
	var parseLimit int32
	var parseEventTypeFilter string
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseLimit = parseReq.GetLimit()
		parseEventTypeFilter = strings.TrimSpace(parseReq.GetEventType())
		parseListQuery = parseReq.GetListQuery()
	}
	parseListQueryShape := parseBuildAdminListQueryShape(parseLimit, parseListQuery)
	parseLimit = parseListQueryShape.parseLimit
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.billing.events")
	if parseErr != nil {
		return nil, parseErr
	}
	parseLogger.Info("rpc.ListAdminBillingEvents: slice fetch", slog.Int64("admin_user_id", parseScope.adminUserID), slog.Int64("target_user_id", parseUserID), slog.Int("limit", int(parseLimit)))
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"billing-events",
		"Admin billing events viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListAdminBillingEvents: store unavailable")
		return &chatpb.ListAdminBillingEventsResponse{}, nil
	}
	parseEventRows, parseErr := parseS.store.parseListAdminBillingEventsByUser(parseUserID, parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminBillingEvents: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin billing events: %v", parseErr)
	}
	parseEventRows = parseFilterAdminBillingEventRows(parseEventRows, parseEventTypeFilter, parseListQueryShape.parseSearch)
	parseSortAdminBillingEventRows(parseEventRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
	parseEventRows = parseApplyAdminSliceWindow(parseEventRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	parseResponse := &chatpb.ListAdminBillingEventsResponse{
		Events: make([]*chatpb.AdminBillingEventEntry, 0, len(parseEventRows)),
	}
	for _, parseEventRow := range parseEventRows {
		parseResponse.Events = append(parseResponse.Events, parseRedactAdminBillingEventEntryByScope(parseScope, parseBuildAdminBillingEventEntry(parseEventRow)))
	}
	parseLogger.Info("rpc.ListAdminBillingEvents: complete", slog.Int("count", len(parseResponse.Events)))
	return parseResponse, nil
}

// ListAdminBillingDunningEvents returns typed dunning-review rows for one admin-target user.
func (parseS *chatServer) ListAdminBillingDunningEvents(parseCtx context.Context, parseReq *chatpb.ListAdminBillingDunningEventsRequest) (*chatpb.ListAdminBillingDunningEventsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminBillingDunningEvents"))
	var parseUserID int64
	var parseLimit int32
	var parseStatusFilter string
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseLimit = parseReq.GetLimit()
		parseStatusFilter = strings.TrimSpace(parseReq.GetStatus())
		parseListQuery = parseReq.GetListQuery()
	}
	parseListQueryShape := parseBuildAdminListQueryShape(parseLimit, parseListQuery)
	parseLimit = parseListQueryShape.parseLimit
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.billing.dunning")
	if parseErr != nil {
		return nil, parseErr
	}
	parseLogger.Info("rpc.ListAdminBillingDunningEvents: slice fetch", slog.Int64("admin_user_id", parseScope.adminUserID), slog.Int64("target_user_id", parseUserID), slog.Int("limit", int(parseLimit)))
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"billing-dunning",
		"Admin billing dunning rows viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListAdminBillingDunningEvents: store unavailable")
		return &chatpb.ListAdminBillingDunningEventsResponse{}, nil
	}
	parseDunningRows, parseErr := parseS.store.parseListAdminBillingDunningEventsByUser(parseUserID, parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminBillingDunningEvents: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin billing dunning events: %v", parseErr)
	}
	parseResponse := &chatpb.ListAdminBillingDunningEventsResponse{
		Events: make([]*chatpb.AdminBillingDunningEventEntry, 0, len(parseDunningRows)),
	}
	if len(parseDunningRows) == 0 {
		parseEventRows, parseEventErr := parseS.store.parseListAdminBillingEventsByUser(parseUserID, parseAdminScopedScanLimit)
		if parseEventErr != nil {
			parseLogger.Error("rpc.ListAdminBillingDunningEvents: billing event fallback query failed", slog.String("error", parseEventErr.Error()))
			return nil, status.Errorf(codes.Internal, "list admin billing dunning fallback events: %v", parseEventErr)
		}
		for _, parseEventRow := range parseEventRows {
			if !parseHasBillingDunningEvent(parseEventRow.EventType) {
				continue
			}
			parseResponse.Events = append(parseResponse.Events, parseBuildAdminBillingDunningEventEntryFromBillingEvent(parseEventRow))
		}
	} else {
		parseDunningRows = parseFilterAdminBillingDunningRows(parseDunningRows, parseStatusFilter, parseListQueryShape.parseSearch)
		parseSortAdminBillingDunningRows(parseDunningRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
		parseDunningRows = parseApplyAdminSliceWindow(parseDunningRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
		for _, parseDunningRow := range parseDunningRows {
			parseResponse.Events = append(parseResponse.Events, parseBuildAdminBillingDunningEventEntry(parseDunningRow))
		}
	}
	if len(parseDunningRows) == 0 && len(parseResponse.Events) > 0 {
		parseFilteredFallbackRows := make([]*chatpb.AdminBillingDunningEventEntry, 0, len(parseResponse.Events))
		parseStatusFilter = strings.TrimSpace(strings.ToLower(parseStatusFilter))
		parseSearchValue := strings.TrimSpace(strings.ToLower(parseListQueryShape.parseSearch))
		for _, parseFallbackRow := range parseResponse.Events {
			if parseFallbackRow == nil {
				continue
			}
			if parseStatusFilter != "" && strings.TrimSpace(strings.ToLower(parseFallbackRow.GetStatus())) != parseStatusFilter {
				continue
			}
			if parseSearchValue != "" {
				if !strings.Contains(strconv.FormatInt(parseFallbackRow.GetInvoiceId(), 10), parseSearchValue) &&
					!strings.Contains(strings.ToLower(strings.TrimSpace(parseFallbackRow.GetFailureReason())), parseSearchValue) &&
					!strings.Contains(strings.ToLower(strings.TrimSpace(parseFallbackRow.GetStatus())), parseSearchValue) {
					continue
				}
			}
			parseFilteredFallbackRows = append(parseFilteredFallbackRows, parseFallbackRow)
		}
		parseSortAdminBillingDunningEntryRows(parseFilteredFallbackRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
		parseResponse.Events = parseApplyAdminSliceWindow(parseFilteredFallbackRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	}
	parseLogger.Info("rpc.ListAdminBillingDunningEvents: complete", slog.Int("count", len(parseResponse.Events)))
	return parseResponse, nil
}

// parseFilterAdminBillingAccessOverrideRows filters billing-access overrides by optional override key and search term.
func parseFilterAdminBillingAccessOverrideRows(parseRows []parseBillingAccessOverrideRow, parseOverrideKey string, parseSearch string) []parseBillingAccessOverrideRow {
	parseOverrideKey = strings.TrimSpace(strings.ToLower(parseOverrideKey))
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	parseFilteredRows := make([]parseBillingAccessOverrideRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseOverrideKey != "" && strings.TrimSpace(strings.ToLower(parseRow.OverrideKey)) != parseOverrideKey {
			continue
		}
		if parseSearch != "" {
			if !strings.Contains(strconv.FormatInt(parseRow.ID, 10), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.OverrideKey)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.OverrideValue)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Reason)), parseSearch) {
				continue
			}
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseSortAdminBillingAccessOverrideRows sorts billing-access override rows by one typed sort key and direction.
func parseSortAdminBillingAccessOverrideRows(parseRows []parseBillingAccessOverrideRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "id":
			return parseCompareAdminInt64(parseLeftRow.ID, parseRightRow.ID, isParseSortAscending)
		case "override_key":
			return parseCompareAdminString(parseLeftRow.OverrideKey, parseRightRow.OverrideKey, isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		case "updated_at":
			return parseCompareAdminString(parseLeftRow.UpdatedAt, parseRightRow.UpdatedAt, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseFilterAdminBillingEventRows filters billing-event rows by optional type and search term.
func parseFilterAdminBillingEventRows(parseRows []parseBillingEventRow, parseEventType string, parseSearch string) []parseBillingEventRow {
	parseEventType = strings.TrimSpace(strings.ToLower(parseEventType))
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	parseFilteredRows := make([]parseBillingEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseEventType != "" && strings.TrimSpace(strings.ToLower(parseRow.EventType)) != parseEventType {
			continue
		}
		if parseSearch != "" {
			if !strings.Contains(strconv.FormatInt(parseRow.ID, 10), parseSearch) &&
				!strings.Contains(strings.TrimSpace(strings.ToLower(parseRow.EventType)), parseSearch) &&
				!strings.Contains(strings.TrimSpace(strings.ToLower(parseRow.EventSource)), parseSearch) &&
				!strings.Contains(strings.TrimSpace(strings.ToLower(parseRow.EventSummary)), parseSearch) &&
				!strings.Contains(strconv.FormatInt(parseRow.InvoiceID, 10), parseSearch) {
				continue
			}
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseSortAdminBillingEventRows sorts billing-event rows by one typed sort key and direction.
func parseSortAdminBillingEventRows(parseRows []parseBillingEventRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "id":
			return parseCompareAdminInt64(parseLeftRow.ID, parseRightRow.ID, isParseSortAscending)
		case "customer_id":
			return parseCompareAdminInt64(parseLeftRow.CustomerID, parseRightRow.CustomerID, isParseSortAscending)
		case "subscription_id":
			return parseCompareAdminInt64(parseLeftRow.SubscriptionID, parseRightRow.SubscriptionID, isParseSortAscending)
		case "event_type":
			return parseCompareAdminString(parseLeftRow.EventType, parseRightRow.EventType, isParseSortAscending)
		case "event_source":
			return parseCompareAdminString(parseLeftRow.EventSource, parseRightRow.EventSource, isParseSortAscending)
		case "event_summary":
			return parseCompareAdminString(parseLeftRow.EventSummary, parseRightRow.EventSummary, isParseSortAscending)
		case "invoice_id":
			return parseCompareAdminInt64(parseLeftRow.InvoiceID, parseRightRow.InvoiceID, isParseSortAscending)
		case "actor_user_id":
			return parseCompareAdminInt64(parseLeftRow.ActorUserID, parseRightRow.ActorUserID, isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseFilterAdminBillingDunningRows filters billing-dunning rows by optional status and search term.
func parseFilterAdminBillingDunningRows(parseRows []parseBillingDunningEventRow, parseStatusFilter string, parseSearch string) []parseBillingDunningEventRow {
	parseStatusFilter = strings.TrimSpace(strings.ToLower(parseStatusFilter))
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	parseFilteredRows := make([]parseBillingDunningEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseStatusFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.Status)) != parseStatusFilter {
			continue
		}
		if parseSearch != "" {
			if !strings.Contains(strconv.FormatInt(parseRow.ID, 10), parseSearch) &&
				!strings.Contains(strconv.FormatInt(parseRow.InvoiceID, 10), parseSearch) &&
				!strings.Contains(strings.TrimSpace(strings.ToLower(parseRow.Status)), parseSearch) &&
				!strings.Contains(strings.TrimSpace(strings.ToLower(parseRow.FailureReason)), parseSearch) {
				continue
			}
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseSortAdminBillingDunningRows sorts billing-dunning rows by one typed sort key and direction.
func parseSortAdminBillingDunningRows(parseRows []parseBillingDunningEventRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "id":
			return parseCompareAdminInt64(parseLeftRow.ID, parseRightRow.ID, isParseSortAscending)
		case "invoice_id":
			return parseCompareAdminInt64(parseLeftRow.InvoiceID, parseRightRow.InvoiceID, isParseSortAscending)
		case "status":
			return parseCompareAdminString(parseLeftRow.Status, parseRightRow.Status, isParseSortAscending)
		case "attempt_count":
			return parseCompareAdminInt64(parseLeftRow.AttemptCount, parseRightRow.AttemptCount, isParseSortAscending)
		case "updated_at":
			return parseCompareAdminString(parseLeftRow.UpdatedAt, parseRightRow.UpdatedAt, isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseSortAdminBillingDunningEntryRows sorts protobuf fallback dunning-event rows by one typed sort key and direction.
func parseSortAdminBillingDunningEntryRows(parseRows []*chatpb.AdminBillingDunningEventEntry, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		if parseLeftRow == nil || parseRightRow == nil {
			return false
		}
		switch parseSortBy {
		case "id":
			return parseCompareAdminInt64(parseLeftRow.GetId(), parseRightRow.GetId(), isParseSortAscending)
		case "invoice_id":
			return parseCompareAdminInt64(parseLeftRow.GetInvoiceId(), parseRightRow.GetInvoiceId(), isParseSortAscending)
		case "status":
			return parseCompareAdminString(parseLeftRow.GetStatus(), parseRightRow.GetStatus(), isParseSortAscending)
		case "updated_at":
			return parseCompareAdminString(parseLeftRow.GetUpdatedAt(), parseRightRow.GetUpdatedAt(), isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.GetCreatedAt(), parseRightRow.GetCreatedAt(), isParseSortAscending)
		default:
			return false
		}
	})
}

// SetAdminBillingAccessOverride stores one typed admin billing access override for one target user.
func (parseS *chatServer) SetAdminBillingAccessOverride(parseCtx context.Context, parseReq *chatpb.AdminBillingAccessOverrideMutationRequest) (*chatpb.AdminBillingAccessOverrideMutationResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetAdminBillingAccessOverride"))
	var parseUserID int64
	var parseConfirm bool
	var parseOverrideKey string
	var parseOverrideValue string
	var parseReason string
	var parseIsEnabled bool
	var parseStartsAt string
	var parseEndsAt string
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseConfirm = parseReq.GetConfirm()
		parseOverrideKey = strings.TrimSpace(parseReq.GetOverrideKey())
		parseOverrideValue = strings.TrimSpace(parseReq.GetOverrideValue())
		parseReason = strings.TrimSpace(parseReq.GetReason())
		parseIsEnabled = parseReq.GetIsEnabled()
		parseStartsAt = strings.TrimSpace(parseReq.GetStartsAt())
		parseEndsAt = strings.TrimSpace(parseReq.GetEndsAt())
	}
	if parseOverrideKey == "" {
		return nil, status.Error(codes.InvalidArgument, "override key is required")
	}
	if parseErr := parseValidateUsageBasedBillingOverrideWrite(parseOverrideKey, parseOverrideValue); parseErr != nil {
		return nil, parseErr
	}
	if !parseConfirm {
		return nil, status.Error(codes.InvalidArgument, "confirmation is required")
	}
	if parseReason == "" {
		return nil, status.Error(codes.InvalidArgument, "reason is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.billing.access_override.mutation")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseOverrideRow, parseErr := parseS.store.parseSetAdminBillingAccessOverrideByUser(parseUserID, parseBillingAccessOverrideWrite{
		OverrideKey:   parseOverrideKey,
		OverrideValue: parseOverrideValue,
		Reason:        parseReason,
		IsEnabled:     parseIsEnabled,
		StartsAt:      parseStartsAt,
		EndsAt:        parseEndsAt,
		ActorUserID:   parseScope.adminUserID,
	})
	if parseErr != nil {
		if parseStatusErr, parseOk := status.FromError(parseErr); parseOk {
			return nil, status.Error(parseStatusErr.Code(), parseStatusErr.Message())
		}
		parseLogger.Error("rpc.SetAdminBillingAccessOverride: mutation failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "set admin billing access override: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.billing.override.set",
		"user",
		strconv.FormatInt(parseUserID, 10),
		"Admin billing access override updated",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseStatus := "disabled"
	if parseOverrideRow.IsEnabled {
		parseStatus = "enabled"
	}
	return &chatpb.AdminBillingAccessOverrideMutationResponse{
		UserId:      parseUserID,
		OverrideKey: parseOverrideRow.OverrideKey,
		Status:      parseStatus,
	}, nil
}

// SetAdminBillingQuotaOverride stores one typed admin billing quota override for one target user.
func (parseS *chatServer) SetAdminBillingQuotaOverride(parseCtx context.Context, parseReq *chatpb.AdminBillingQuotaOverrideMutationRequest) (*chatpb.AdminBillingQuotaOverrideMutationResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetAdminBillingQuotaOverride"))
	var parseUserID int64
	var parseConfirm bool
	var parseQuotaKey string
	var parseQuotaValue string
	var parseReason string
	var parseIsEnabled bool
	var parseStartsAt string
	var parseEndsAt string
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseConfirm = parseReq.GetConfirm()
		parseQuotaKey = strings.TrimSpace(parseReq.GetQuotaKey())
		parseQuotaValue = strings.TrimSpace(parseReq.GetQuotaValue())
		parseReason = strings.TrimSpace(parseReq.GetReason())
		parseIsEnabled = parseReq.GetIsEnabled()
		parseStartsAt = strings.TrimSpace(parseReq.GetStartsAt())
		parseEndsAt = strings.TrimSpace(parseReq.GetEndsAt())
	}
	if parseQuotaKey == "" {
		return nil, status.Error(codes.InvalidArgument, "quota key is required")
	}
	parseNormalizedQuotaKey := parseNormalizeAdminBillingQuotaOverrideKey(parseQuotaKey)
	if parseErr := parseValidateUsageBasedBillingOverrideWrite(parseNormalizedQuotaKey, parseQuotaValue); parseErr != nil {
		return nil, parseErr
	}
	if !parseConfirm {
		return nil, status.Error(codes.InvalidArgument, "confirmation is required")
	}
	if parseReason == "" {
		return nil, status.Error(codes.InvalidArgument, "reason is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.billing.quota_override.mutation")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseOverrideRow, parseErr := parseS.store.parseSetAdminBillingQuotaOverrideByUser(parseUserID, parseBillingAccessOverrideWrite{
		OverrideKey:   parseQuotaKey,
		OverrideValue: parseQuotaValue,
		Reason:        parseReason,
		IsEnabled:     parseIsEnabled,
		StartsAt:      parseStartsAt,
		EndsAt:        parseEndsAt,
		ActorUserID:   parseScope.adminUserID,
	})
	if parseErr != nil {
		if parseStatusErr, parseOk := status.FromError(parseErr); parseOk {
			return nil, status.Error(parseStatusErr.Code(), parseStatusErr.Message())
		}
		parseLogger.Error("rpc.SetAdminBillingQuotaOverride: mutation failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "set admin billing quota override: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.billing.quota_override.set",
		"user",
		strconv.FormatInt(parseUserID, 10),
		"Admin billing quota override updated",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseStatus := "disabled"
	if parseOverrideRow.IsEnabled {
		parseStatus = "enabled"
	}
	return &chatpb.AdminBillingQuotaOverrideMutationResponse{
		UserId:   parseUserID,
		QuotaKey: parseOverrideRow.OverrideKey,
		Status:   parseStatus,
	}, nil
}

// ResolveAdminBillingFailedPayment marks one failed-payment path resolved for one target user and invoice.
func (parseS *chatServer) ResolveAdminBillingFailedPayment(parseCtx context.Context, parseReq *chatpb.ResolveAdminBillingFailedPaymentRequest) (*chatpb.ResolveAdminBillingFailedPaymentResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ResolveAdminBillingFailedPayment"))
	var parseUserID int64
	var parseInvoiceID int64
	var parseReason string
	var parseConfirm bool
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseInvoiceID = parseReq.GetInvoiceId()
		parseReason = strings.TrimSpace(parseReq.GetReason())
		parseConfirm = parseReq.GetConfirm()
	}
	if parseInvoiceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invoice id is required")
	}
	if !parseConfirm {
		return nil, status.Error(codes.InvalidArgument, "confirmation is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminBillingUserScope(parseCtx, parseUserID, "dashboard.billing.failed_payment.resolve")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseResolvedInvoice, parseErr := parseS.store.parseResolveAdminBillingFailedPaymentByUser(parseUserID, parseInvoiceID, parseScope.adminUserID, parseReason)
	if parseErr != nil {
		parseLogger.Error("rpc.ResolveAdminBillingFailedPayment: mutation failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "resolve admin billing failed payment: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.billing.failed_payment.resolved",
		"user",
		strconv.FormatInt(parseUserID, 10),
		"Admin billing failed payment resolved",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	return &chatpb.ResolveAdminBillingFailedPaymentResponse{
		UserId:        parseUserID,
		InvoiceId:     parseResolvedInvoice.ID,
		InvoiceStatus: parseResolvedInvoice.Status,
	}, nil
}
