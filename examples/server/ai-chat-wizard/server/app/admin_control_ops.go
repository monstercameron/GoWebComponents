package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminFeatureFlagEntry maps one feature-flag row into protobuf form.
func parseBuildAdminFeatureFlagEntry(parseRow parseFeatureFlagRow) *chatpb.FeatureFlagEntry {
	return &chatpb.FeatureFlagEntry{
		FlagKey:         parseRow.FlagKey,
		Description:     parseRow.Description,
		IsEnabled:       parseRow.IsEnabled,
		RolloutPercent:  parseRow.RolloutPercent,
		AudienceJson:    parseRow.AudienceJSON,
		PayloadJson:     parseRow.PayloadJSON,
		UpdatedByUserId: parseRow.UpdatedByUserID,
		UpdatedAt:       parseRow.UpdatedAt,
	}
}

// parseBuildAdminExperimentEntry maps one experiment row into protobuf form.
func parseBuildAdminExperimentEntry(parseRow parseExperimentRow) *chatpb.ExperimentEntry {
	return &chatpb.ExperimentEntry{
		Id:            parseRow.ID,
		ExperimentKey: parseRow.ExperimentKey,
		Name:          parseRow.Name,
		Status:        parseRow.Status,
		VariantsJson:  parseRow.VariantsJSON,
		AudienceJson:  parseRow.AudienceJSON,
		StartAt:       parseRow.StartAt,
		EndAt:         parseRow.EndAt,
		UpdatedAt:     parseRow.UpdatedAt,
	}
}

// parseBuildAdminIncidentEntry maps one incident row into protobuf form.
func parseBuildAdminIncidentEntry(parseRow parseIncidentRow) *chatpb.IncidentEntry {
	return &chatpb.IncidentEntry{
		Id:            parseRow.ID,
		IncidentKey:   parseRow.IncidentKey,
		SloKey:        parseRow.SLOKey,
		Severity:      parseRow.Severity,
		Status:        parseRow.Status,
		Title:         parseRow.Title,
		Summary:       parseRow.Summary,
		StartedAt:     parseRow.StartedAt,
		ResolvedAt:    parseRow.ResolvedAt,
		PostmortemUrl: parseRow.PostmortemURL,
		UpdatedAt:     parseRow.UpdatedAt,
	}
}

// parseBuildAdminIncidentUpdateEntry maps one incident-update row into protobuf form.
func parseBuildAdminIncidentUpdateEntry(parseRow parseIncidentUpdateRow) *chatpb.IncidentUpdateEntry {
	return &chatpb.IncidentUpdateEntry{
		Id:              parseRow.ID,
		IncidentId:      parseRow.IncidentID,
		Status:          parseRow.Status,
		Message:         parseRow.Message,
		IsPublic:        parseRow.IsPublic,
		PublishedAt:     parseRow.PublishedAt,
		CreatedByUserId: parseRow.CreatedByUserID,
		CreatedAt:       parseRow.CreatedAt,
	}
}

// parseRequireAdminControlMutationConfirmation enforces explicit confirmation and one non-empty mutation reason.
func parseRequireAdminControlMutationConfirmation(isParseConfirmed bool, parseReason string) (string, error) {
	if !isParseConfirmed {
		return "", status.Error(codes.InvalidArgument, "control mutation confirmation is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return "", status.Error(codes.InvalidArgument, "control mutation reason is required")
	}
	return parseReason, nil
}

// SetAdminFeatureFlag applies one typed feature-flag toggle mutation for superusers.
func (parseS *chatServer) SetAdminFeatureFlag(parseCtx context.Context, parseReq *chatpb.SetAdminFeatureFlagRequest) (*chatpb.SetAdminFeatureFlagResponse, error) {
	var parseFlagKey string
	var isParseEnabled bool
	var parseRolloutPercent int64
	var parseAudienceJSON string
	var parsePayloadJSON string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseFlagKey = strings.TrimSpace(parseReq.GetFlagKey())
		isParseEnabled = parseReq.GetIsEnabled()
		parseRolloutPercent = parseReq.GetRolloutPercent()
		parseAudienceJSON = strings.TrimSpace(parseReq.GetAudienceJson())
		parsePayloadJSON = strings.TrimSpace(parseReq.GetPayloadJson())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	if parseFlagKey == "" {
		return nil, status.Error(codes.InvalidArgument, "flag key is required")
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationFeatureFlag, 0)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseFlagRow, parseErr := parseS.store.parseStoreAdminFeatureFlagToggle(parseFlagKey, isParseEnabled, parseRolloutPercent, parseAudienceJSON, parsePayloadJSON, parseReason, parseScope.adminUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set admin feature flag: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.control.feature_flag.toggle",
		"feature_flag",
		parseFlagKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetAdminFeatureFlagResponse{FeatureFlag: parseBuildAdminFeatureFlagEntry(parseFlagRow)}, nil
}

// RollbackAdminExperiment applies one typed experiment rollback mutation for superusers.
func (parseS *chatServer) RollbackAdminExperiment(parseCtx context.Context, parseReq *chatpb.RollbackAdminExperimentRequest) (*chatpb.RollbackAdminExperimentResponse, error) {
	var parseExperimentKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseExperimentKey = strings.TrimSpace(parseReq.GetExperimentKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	if parseExperimentKey == "" {
		return nil, status.Error(codes.InvalidArgument, "experiment key is required")
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationExperiment, 0)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseExperimentRow, parseErr := parseS.store.parseStoreAdminExperimentRollback(parseExperimentKey, parseReason)
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "experiment not found")
		}
		return nil, status.Errorf(codes.Internal, "rollback admin experiment: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.control.experiment.rollback",
		"experiment",
		parseExperimentKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.RollbackAdminExperimentResponse{Experiment: parseBuildAdminExperimentEntry(parseExperimentRow)}, nil
}

// UpdateAdminIncident applies one typed incident update and status mutation for scoped admin callers.
func (parseS *chatServer) UpdateAdminIncident(parseCtx context.Context, parseReq *chatpb.UpdateAdminIncidentRequest) (*chatpb.UpdateAdminIncidentResponse, error) {
	var parseIncidentID int64
	var parseWorkspaceID int64
	var parseStatusValue string
	var parseMessage string
	var isParsePublic bool
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseIncidentID = parseReq.GetIncidentId()
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseStatusValue = strings.TrimSpace(parseReq.GetStatus())
		parseMessage = strings.TrimSpace(parseReq.GetMessage())
		isParsePublic = parseReq.GetIsPublic()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	if parseIncidentID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "incident id is required")
	}
	if parseWorkspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationIncident, parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseIncidentRow, parseIncidentUpdateRow, parseErr := parseS.store.parseStoreAdminIncidentStatusUpdate(parseIncidentID, parseStatusValue, parseMessage, isParsePublic, parseReason, parseScope.adminUserID)
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "incident not found")
		}
		return nil, status.Errorf(codes.Internal, "update admin incident: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.control.incident.update",
		"incident",
		strconv.FormatInt(parseIncidentID, 10),
		parseReason,
		"{}",
		parseWorkspaceID,
	)
	return &chatpb.UpdateAdminIncidentResponse{
		Incident:       parseBuildAdminIncidentEntry(parseIncidentRow),
		IncidentUpdate: parseBuildAdminIncidentUpdateEntry(parseIncidentUpdateRow),
	}, nil
}

// GetAdminIncidentBlastRadius returns one scoped incident blast-radius snapshot for one workspace.
func (parseS *chatServer) GetAdminIncidentBlastRadius(parseCtx context.Context, parseReq *chatpb.GetAdminIncidentBlastRadiusRequest) (*chatpb.GetAdminIncidentBlastRadiusResponse, error) {
	var parseWorkspaceID int64
	var parseLookbackDays int32
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseLookbackDays = parseReq.GetLookbackDays()
	}
	if parseWorkspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationIncident, parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseResponse := &chatpb.GetAdminIncidentBlastRadiusResponse{
		BlastRadius: &chatpb.AdminIncidentBlastRadius{WorkspaceId: parseWorkspaceID},
	}
	if parseS.store == nil {
		return parseResponse, nil
	}
	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list workspace memberships by workspace: %v", parseErr)
	}
	parseScopedUserIDs := make(map[int64]struct{})
	for _, parseMembershipRow := range parseMembershipRows {
		if !parseIsWorkspaceMembershipActive(parseMembershipRow.Status) {
			continue
		}
		if parseMembershipRow.UserID <= 0 {
			continue
		}
		parseScopedUserIDs[parseMembershipRow.UserID] = struct{}{}
	}
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list admin usage events: %v", parseErr)
	}
	parseImpactedUserIDs := make(map[int64]struct{})
	for _, parseUsageRow := range parseUsageRows {
		if _, hasParseUser := parseScopedUserIDs[parseUsageRow.UserID]; !hasParseUser {
			continue
		}
		parseImpactedUserIDs[parseUsageRow.UserID] = struct{}{}
		parseResponse.BlastRadius.RecentUsageEventCount++
		parseResponse.BlastRadius.RecentUsageCostUsd += parseUsageRow.TotalCostUSD
	}
	parseResponse.BlastRadius.ImpactedUserCount = int64(len(parseImpactedUserIDs))

	parseTicketRows, parseErr := parseS.store.parseListSupportTickets(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list support tickets: %v", parseErr)
	}
	for _, parseTicketRow := range parseTicketRows {
		if parseTicketRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(parseTicketRow.Status)) {
		case "resolved", "closed":
			continue
		default:
			parseResponse.BlastRadius.OpenSupportTicketCount++
		}
	}

	parseQueueKey := fmt.Sprintf("workspace:%d", parseWorkspaceID)
	parseJobRows, parseErr := parseS.store.parseListBackgroundJobs(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list background jobs: %v", parseErr)
	}
	for _, parseJobRow := range parseJobRows {
		if strings.TrimSpace(parseJobRow.QueueKey) != parseQueueKey {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(parseJobRow.Status)) {
		case "pending", "running":
			parseResponse.BlastRadius.PendingBackgroundJobCount++
		}
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.control.incident.blast_radius",
		"workspace",
		strconv.FormatInt(parseWorkspaceID, 10),
		"Admin incident blast radius viewed",
		"{}",
		parseWorkspaceID,
	)
	return parseResponse, nil
}
