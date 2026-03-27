package app

import (
	"context"
	"log/slog"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultSuperuserListLimit = 100
const maxSuperuserListLimit = 250

// parseClampSuperuserListLimit clamps one superuser snapshot list size into a safe range.
func parseClampSuperuserListLimit(parseLimit int32) int32 {
	if parseLimit <= 0 {
		return defaultSuperuserListLimit
	}
	if parseLimit > maxSuperuserListLimit {
		return maxSuperuserListLimit
	}
	return parseLimit
}

// parseRequireSuperuserUserID resolves the authenticated user id and enforces the su role grant.
func (parseS *chatServer) parseRequireSuperuserUserID(parseCtx context.Context) (int64, error) {
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseS.store == nil {
		return 0, status.Error(codes.Unavailable, "store unavailable")
	}
	isParseAllowed, parseErr := parseS.store.parseUserHasSURole(parseUserID)
	if parseErr != nil {
		return 0, status.Errorf(codes.Internal, "superuser role lookup failed: %v", parseErr)
	}
	if !isParseAllowed {
		return 0, status.Error(codes.PermissionDenied, "superuser role required")
	}
	return parseUserID, nil
}

// GetSuperuserControlPlane returns the superuser control-plane snapshot for authenticated su users.
func (parseS *chatServer) GetSuperuserControlPlane(parseCtx context.Context, parseReq *chatpb.GetSuperuserControlPlaneRequest) (*chatpb.GetSuperuserControlPlaneResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSuperuserControlPlane"))
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetSuperuserControlPlane: store unavailable")
		return &chatpb.GetSuperuserControlPlaneResponse{}, nil
	}
	var parseLimit int32
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
	}
	parseLimit = parseClampSuperuserListLimit(parseLimit)

	parseRoles, parseErr := parseS.store.parseListSURoles()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list su roles: %v", parseErr)
	}
	parsePermissions, parseErr := parseS.store.parseListSURolePermissions()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list su role permissions: %v", parseErr)
	}
	parseUserRoles, parseErr := parseS.store.parseListSUUserRoles()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list su user roles: %v", parseErr)
	}
	parseSiteConfigs, parseErr := parseS.store.parseListSiteConfigs()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list site configs: %v", parseErr)
	}
	parseFeatureFlags, parseErr := parseS.store.parseListFeatureFlags()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list feature flags: %v", parseErr)
	}
	parseWorkspaces, parseErr := parseS.store.parseListWorkspaces(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list workspaces: %v", parseErr)
	}
	parseMemberships, parseErr := parseS.store.parseListWorkspaceMemberships(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list workspace memberships: %v", parseErr)
	}
	parseAPIKeys, parseErr := parseS.store.parseListAPIKeys(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list api keys: %v", parseErr)
	}
	parseWebhooks, parseErr := parseS.store.parseListWebhookEndpoints(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list webhook endpoints: %v", parseErr)
	}
	parseAuditLogs, parseErr := parseS.store.parseListAuditLogs(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list audit logs: %v", parseErr)
	}
	parseSupportTickets, parseErr := parseS.store.parseListSupportTickets(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list support tickets: %v", parseErr)
	}
	parseExperiments, parseErr := parseS.store.parseListExperiments(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list experiments: %v", parseErr)
	}

	parsePermissionsByRoleKey := make(map[string][]*chatpb.SuperuserPermission, len(parseRoles))
	for _, parsePermission := range parsePermissions {
		parsePermissionsByRoleKey[parsePermission.RoleKey] = append(parsePermissionsByRoleKey[parsePermission.RoleKey], &chatpb.SuperuserPermission{
			RoleKey:         parsePermission.RoleKey,
			PermissionKey:   parsePermission.PermissionKey,
			PermissionValue: parsePermission.PermissionValue,
			UpdatedAt:       parsePermission.UpdatedAt,
		})
	}

	parseResponse := &chatpb.GetSuperuserControlPlaneResponse{
		Roles:            make([]*chatpb.SuperuserRole, 0, len(parseRoles)),
		UserRoles:        make([]*chatpb.SuperuserUserRole, 0, len(parseUserRoles)),
		SiteConfigs:      make([]*chatpb.SiteConfigEntry, 0, len(parseSiteConfigs)),
		FeatureFlags:     make([]*chatpb.FeatureFlagEntry, 0, len(parseFeatureFlags)),
		Workspaces:       make([]*chatpb.WorkspaceEntry, 0, len(parseWorkspaces)),
		Memberships:      make([]*chatpb.WorkspaceMembershipEntry, 0, len(parseMemberships)),
		ApiKeys:          make([]*chatpb.APIKeyEntry, 0, len(parseAPIKeys)),
		WebhookEndpoints: make([]*chatpb.WebhookEndpointEntry, 0, len(parseWebhooks)),
		AuditLogs:        make([]*chatpb.AuditLogEntry, 0, len(parseAuditLogs)),
		SupportTickets:   make([]*chatpb.SupportTicketEntry, 0, len(parseSupportTickets)),
		Experiments:      make([]*chatpb.ExperimentEntry, 0, len(parseExperiments)),
	}

	for _, parseRole := range parseRoles {
		parseResponse.Roles = append(parseResponse.Roles, &chatpb.SuperuserRole{
			RoleKey:     parseRole.RoleKey,
			Label:       parseRole.Label,
			Description: parseRole.Description,
			IsSystem:    parseRole.IsSystem,
			IsEnabled:   parseRole.IsEnabled,
			CreatedAt:   parseRole.CreatedAt,
			UpdatedAt:   parseRole.UpdatedAt,
			Permissions: parsePermissionsByRoleKey[parseRole.RoleKey],
		})
	}
	for _, parseUserRole := range parseUserRoles {
		parseResponse.UserRoles = append(parseResponse.UserRoles, &chatpb.SuperuserUserRole{
			UserId:           parseUserRole.UserID,
			RoleKey:          parseUserRole.RoleKey,
			AssignedByUserId: parseUserRole.AssignedByUserID,
			CreatedAt:        parseUserRole.CreatedAt,
		})
	}
	for _, parseSiteConfig := range parseSiteConfigs {
		parseResponse.SiteConfigs = append(parseResponse.SiteConfigs, &chatpb.SiteConfigEntry{
			ConfigKey:       parseSiteConfig.ConfigKey,
			ConfigValue:     parseSiteConfig.ConfigValue,
			ValueType:       parseSiteConfig.ValueType,
			Description:     parseSiteConfig.Description,
			UpdatedByUserId: parseSiteConfig.UpdatedByUserID,
			UpdatedAt:       parseSiteConfig.UpdatedAt,
		})
	}
	for _, parseFeatureFlag := range parseFeatureFlags {
		parseResponse.FeatureFlags = append(parseResponse.FeatureFlags, &chatpb.FeatureFlagEntry{
			FlagKey:         parseFeatureFlag.FlagKey,
			Description:     parseFeatureFlag.Description,
			IsEnabled:       parseFeatureFlag.IsEnabled,
			RolloutPercent:  parseFeatureFlag.RolloutPercent,
			AudienceJson:    parseFeatureFlag.AudienceJSON,
			PayloadJson:     parseFeatureFlag.PayloadJSON,
			UpdatedByUserId: parseFeatureFlag.UpdatedByUserID,
			UpdatedAt:       parseFeatureFlag.UpdatedAt,
		})
	}
	for _, parseWorkspace := range parseWorkspaces {
		parseResponse.Workspaces = append(parseResponse.Workspaces, &chatpb.WorkspaceEntry{
			Id:           parseWorkspace.ID,
			WorkspaceKey: parseWorkspace.WorkspaceKey,
			Slug:         parseWorkspace.Slug,
			Name:         parseWorkspace.Name,
			PlanCode:     parseWorkspace.PlanCode,
			Status:       parseWorkspace.Status,
			OwnerUserId:  parseWorkspace.OwnerUserID,
			SettingsJson: parseWorkspace.SettingsJSON,
			CreatedAt:    parseWorkspace.CreatedAt,
			UpdatedAt:    parseWorkspace.UpdatedAt,
		})
	}
	for _, parseMembership := range parseMemberships {
		parseResponse.Memberships = append(parseResponse.Memberships, &chatpb.WorkspaceMembershipEntry{
			Id:              parseMembership.ID,
			WorkspaceId:     parseMembership.WorkspaceID,
			UserId:          parseMembership.UserID,
			RoleKey:         parseMembership.RoleKey,
			Status:          parseMembership.Status,
			InvitedByUserId: parseMembership.InvitedByUserID,
			CreatedAt:       parseMembership.CreatedAt,
			UpdatedAt:       parseMembership.UpdatedAt,
		})
	}
	for _, parseAPIKey := range parseAPIKeys {
		parseResponse.ApiKeys = append(parseResponse.ApiKeys, &chatpb.APIKeyEntry{
			Id:          parseAPIKey.ID,
			KeyId:       parseAPIKey.KeyID,
			WorkspaceId: parseAPIKey.WorkspaceID,
			UserId:      parseAPIKey.UserID,
			Label:       parseAPIKey.Label,
			KeyPrefix:   parseAPIKey.KeyPrefix,
			ScopesJson:  parseAPIKey.ScopesJSON,
			LastUsedAt:  parseAPIKey.LastUsedAt,
			RevokedAt:   parseAPIKey.RevokedAt,
			CreatedAt:   parseAPIKey.CreatedAt,
		})
	}
	for _, parseWebhook := range parseWebhooks {
		parseResponse.WebhookEndpoints = append(parseResponse.WebhookEndpoints, &chatpb.WebhookEndpointEntry{
			Id:             parseWebhook.ID,
			WorkspaceId:    parseWebhook.WorkspaceID,
			Label:          parseWebhook.Label,
			TargetUrl:      parseWebhook.TargetURL,
			EventsJson:     parseWebhook.EventsJSON,
			IsEnabled:      parseWebhook.IsEnabled,
			LastDeliveryAt: parseWebhook.LastDeliveryAt,
			FailureCount:   parseWebhook.FailureCount,
			CreatedAt:      parseWebhook.CreatedAt,
			UpdatedAt:      parseWebhook.UpdatedAt,
		})
	}
	for _, parseAuditLog := range parseAuditLogs {
		parseResponse.AuditLogs = append(parseResponse.AuditLogs, &chatpb.AuditLogEntry{
			Id:          parseAuditLog.ID,
			ActorUserId: parseAuditLog.ActorUserID,
			WorkspaceId: parseAuditLog.WorkspaceID,
			EventType:   parseAuditLog.EventType,
			TargetType:  parseAuditLog.TargetType,
			TargetId:    parseAuditLog.TargetID,
			Summary:     parseAuditLog.Summary,
			PayloadJson: parseAuditLog.PayloadJSON,
			CreatedAt:   parseAuditLog.CreatedAt,
		})
	}
	for _, parseSupportTicket := range parseSupportTickets {
		parseResponse.SupportTickets = append(parseResponse.SupportTickets, &chatpb.SupportTicketEntry{
			Id:             parseSupportTicket.ID,
			TicketKey:      parseSupportTicket.TicketKey,
			WorkspaceId:    parseSupportTicket.WorkspaceID,
			UserId:         parseSupportTicket.UserID,
			Status:         parseSupportTicket.Status,
			Priority:       parseSupportTicket.Priority,
			Subject:        parseSupportTicket.Subject,
			Body:           parseSupportTicket.Body,
			AssigneeUserId: parseSupportTicket.AssigneeUserID,
			ResolutionNote: parseSupportTicket.ResolutionNote,
			CreatedAt:      parseSupportTicket.CreatedAt,
			UpdatedAt:      parseSupportTicket.UpdatedAt,
		})
	}
	for _, parseExperiment := range parseExperiments {
		parseResponse.Experiments = append(parseResponse.Experiments, &chatpb.ExperimentEntry{
			Id:            parseExperiment.ID,
			ExperimentKey: parseExperiment.ExperimentKey,
			Name:          parseExperiment.Name,
			Status:        parseExperiment.Status,
			VariantsJson:  parseExperiment.VariantsJSON,
			AudienceJson:  parseExperiment.AudienceJSON,
			StartAt:       parseExperiment.StartAt,
			EndAt:         parseExperiment.EndAt,
			UpdatedAt:     parseExperiment.UpdatedAt,
		})
	}

	parseLogger.Info(
		"rpc.GetSuperuserControlPlane: complete",
		slog.Int("limit", int(parseLimit)),
		slog.Int("roles", len(parseResponse.Roles)),
		slog.Int("workspaces", len(parseResponse.Workspaces)),
		slog.Int("audit_logs", len(parseResponse.AuditLogs)),
	)
	return parseResponse, nil
}
