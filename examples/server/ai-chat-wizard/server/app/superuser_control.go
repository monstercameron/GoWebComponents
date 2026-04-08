package app

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultSuperuserListLimit = 100
const maxSuperuserListLimit = 250
const superuserMutationSessionMaxAge = 20 * time.Minute

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

// parseBuildSuperuserListQueryShape resolves one superuser slice-query contract with superuser list-limit clamps.
func parseBuildSuperuserListQueryShape(parseLegacyLimit int32, parseQuery *chatpb.AdminListQuery) parseAdminListQueryShape {
	parseLimit := parseLegacyLimit
	if parseQuery != nil && parseQuery.GetLimit() > 0 {
		parseLimit = parseQuery.GetLimit()
	}
	parseLimit = parseClampSuperuserListLimit(parseLimit)
	parseOffset := int32(0)
	parseSearch := ""
	parseSortBy := ""
	isParseSortAscending := false
	if parseQuery != nil {
		if parseQuery.GetOffset() > 0 {
			parseOffset = parseQuery.GetOffset()
		}
		parseSearch = strings.TrimSpace(parseQuery.GetSearch())
		parseSortBy = strings.TrimSpace(strings.ToLower(parseQuery.GetSortBy()))
		isParseSortAscending = parseIsAdminSortDirectionAscending(parseQuery.GetSortDirection())
	}
	return parseAdminListQueryShape{
		parseLimit:           parseLimit,
		parseOffset:          parseOffset,
		parseSearch:          parseSearch,
		parseSortBy:          parseSortBy,
		isParseSortAscending: isParseSortAscending,
	}
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

// parseRequireSuperuserMutationUserID enforces su role plus one fresh session requirement for sensitive mutations.
func (parseS *chatServer) parseRequireSuperuserMutationUserID(parseCtx context.Context, parseMutationKey string) (int64, error) {
	parseUserID, parseErr := parseS.parseRequireSuperuserUserID(parseCtx)
	if parseErr != nil {
		return 0, parseErr
	}
	if parseS == nil || parseS.authManager == nil {
		return parseUserID, nil
	}
	parseSessionUser, parseClaims, isParseSessionAuthenticated := parseS.authManager.parseAuthenticatedSessionFromContext(parseCtx)
	if !isParseSessionAuthenticated || parseSessionUser.ID <= 0 || parseSessionUser.ID != parseUserID || parseClaims.IssuedAt == nil || parseClaims.IssuedAt.Time.IsZero() {
		return 0, status.Error(codes.Unauthenticated, "superuser re-authentication required")
	}
	parseSessionAge := time.Since(parseClaims.IssuedAt.Time.UTC())
	if parseSessionAge < 0 {
		parseSessionAge = 0
	}
	isParseFreshSession := parseSessionAge <= superuserMutationSessionMaxAge
	parsePolicyDecision, parseErr := parseAuthorizePrivilegedMutationSession(
		parseBuildDefaultPrivilegedAuthPolicy(),
		parsePrivilegedAuthAttempt{
			ParseRoleKey:    parsePrivilegedRoleSuperuser,
			ParseAuthMethod: parseClaims.AuthMethod,
		},
		isParseFreshSession,
	)
	if parseErr == nil {
		return parseUserID, nil
	}
	if parseS.logger != nil {
		parseS.logger.Warn(
			"rpc.superuser mutation requires re-authentication",
			slog.Int64("user_id", parseUserID),
			slog.String("mutation_key", strings.TrimSpace(parseMutationKey)),
			slog.String("auth_method", parseNormalizePrivilegedAuthMethod(parseClaims.AuthMethod)),
			slog.String("decision_reason", strings.TrimSpace(parsePolicyDecision.ParseReason)),
			slog.Duration("session_age", parseSessionAge),
			slog.Duration("max_session_age", superuserMutationSessionMaxAge),
		)
	}
	return 0, parseErr
}

// GetSuperuserControlPlane returns the superuser control-plane snapshot for authenticated su users.
func (parseS *chatServer) GetSuperuserControlPlane(parseCtx context.Context, parseReq *chatpb.GetSuperuserControlPlaneRequest) (*chatpb.GetSuperuserControlPlaneResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSuperuserControlPlane"))
	parseFetchStart := time.Now()
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.dashboard.slice.view",
		"slice",
		"superuser-control-plane",
		"Superuser control-plane slice viewed",
		"{}",
		0,
	)
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
	parseAuthSessions, parseErr := parseS.store.parseListAuthSessions(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list auth sessions: %v", parseErr)
	}
	parseWorkspaceInvitations, parseErr := parseS.store.parseListWorkspaceInvitations(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list workspace invitations: %v", parseErr)
	}
	parseWebhookDeliveries, parseErr := parseS.store.parseListWebhookDeliveries(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list webhook deliveries: %v", parseErr)
	}
	parseSupportTicketMessages, parseErr := parseS.store.parseListSupportTicketMessages(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list support ticket messages: %v", parseErr)
	}
	parseIncidentUpdates, parseErr := parseS.store.parseListIncidentUpdates(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list incident updates: %v", parseErr)
	}
	parseNotificationOutboxRows, parseErr := parseS.store.parseListNotificationOutbox(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list notification outbox: %v", parseErr)
	}
	parseBackgroundJobs, parseErr := parseS.store.parseListBackgroundJobs(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list background jobs: %v", parseErr)
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
		Roles:                 make([]*chatpb.SuperuserRole, 0, len(parseRoles)),
		UserRoles:             make([]*chatpb.SuperuserUserRole, 0, len(parseUserRoles)),
		SiteConfigs:           make([]*chatpb.SiteConfigEntry, 0, len(parseSiteConfigs)),
		FeatureFlags:          make([]*chatpb.FeatureFlagEntry, 0, len(parseFeatureFlags)),
		Workspaces:            make([]*chatpb.WorkspaceEntry, 0, len(parseWorkspaces)),
		Memberships:           make([]*chatpb.WorkspaceMembershipEntry, 0, len(parseMemberships)),
		ApiKeys:               make([]*chatpb.APIKeyEntry, 0, len(parseAPIKeys)),
		WebhookEndpoints:      make([]*chatpb.WebhookEndpointEntry, 0, len(parseWebhooks)),
		AuditLogs:             make([]*chatpb.AuditLogEntry, 0, len(parseAuditLogs)),
		SupportTickets:        make([]*chatpb.SupportTicketEntry, 0, len(parseSupportTickets)),
		Experiments:           make([]*chatpb.ExperimentEntry, 0, len(parseExperiments)),
		AuthSessions:          make([]*chatpb.AuthSessionEntry, 0, len(parseAuthSessions)),
		WorkspaceInvitations:  make([]*chatpb.WorkspaceInvitationEntry, 0, len(parseWorkspaceInvitations)),
		WebhookDeliveries:     make([]*chatpb.WebhookDeliveryEntry, 0, len(parseWebhookDeliveries)),
		SupportTicketMessages: make([]*chatpb.SupportTicketMessageEntry, 0, len(parseSupportTicketMessages)),
		IncidentUpdates:       make([]*chatpb.IncidentUpdateEntry, 0, len(parseIncidentUpdates)),
		NotificationOutbox:    make([]*chatpb.NotificationOutboxEntry, 0, len(parseNotificationOutboxRows)),
		BackgroundJobs:        make([]*chatpb.BackgroundJobEntry, 0, len(parseBackgroundJobs)),
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
	for _, parseSession := range parseAuthSessions {
		parseResponse.AuthSessions = append(parseResponse.AuthSessions, &chatpb.AuthSessionEntry{
			Id:           parseSession.ID,
			UserId:       parseSession.UserID,
			SessionId:    parseSession.SessionID,
			TokenVersion: parseSession.TokenVersion,
			UserAgent:    parseSession.UserAgent,
			IpAddress:    parseSession.IPAddress,
			LastSeenAt:   parseSession.LastSeenAt,
			ExpiresAt:    parseSession.ExpiresAt,
			RevokedAt:    parseSession.RevokedAt,
			CreatedAt:    parseSession.CreatedAt,
			UpdatedAt:    parseSession.UpdatedAt,
		})
	}
	for _, parseInvitation := range parseWorkspaceInvitations {
		parseResponse.WorkspaceInvitations = append(parseResponse.WorkspaceInvitations, &chatpb.WorkspaceInvitationEntry{
			Id:                  parseInvitation.ID,
			WorkspaceId:         parseInvitation.WorkspaceID,
			Email:               parseInvitation.Email,
			RoleKey:             parseInvitation.RoleKey,
			InvitationTokenHash: parseInvitation.InvitationTokenHash,
			InvitedByUserId:     parseInvitation.InvitedByUserID,
			Status:              parseInvitation.Status,
			ExpiresAt:           parseInvitation.ExpiresAt,
			AcceptedAt:          parseInvitation.AcceptedAt,
			CreatedAt:           parseInvitation.CreatedAt,
			UpdatedAt:           parseInvitation.UpdatedAt,
		})
	}
	for _, parseDelivery := range parseWebhookDeliveries {
		parseResponse.WebhookDeliveries = append(parseResponse.WebhookDeliveries, &chatpb.WebhookDeliveryEntry{
			Id:                 parseDelivery.ID,
			EndpointId:         parseDelivery.EndpointID,
			EventType:          parseDelivery.EventType,
			DeliveryKey:        parseDelivery.DeliveryKey,
			RequestHeadersJson: parseDelivery.RequestHeadersJSON,
			RequestBodyJson:    parseDelivery.RequestBodyJSON,
			ResponseStatus:     parseDelivery.ResponseStatus,
			ResponseBody:       parseDelivery.ResponseBody,
			AttemptCount:       parseDelivery.AttemptCount,
			DeliveredAt:        parseDelivery.DeliveredAt,
			FailedAt:           parseDelivery.FailedAt,
			NextRetryAt:        parseDelivery.NextRetryAt,
			CreatedAt:          parseDelivery.CreatedAt,
			UpdatedAt:          parseDelivery.UpdatedAt,
		})
	}
	for _, parseSupportTicketMessage := range parseSupportTicketMessages {
		parseResponse.SupportTicketMessages = append(parseResponse.SupportTicketMessages, &chatpb.SupportTicketMessageEntry{
			Id:           parseSupportTicketMessage.ID,
			TicketId:     parseSupportTicketMessage.TicketID,
			AuthorUserId: parseSupportTicketMessage.AuthorUserID,
			MessageType:  parseSupportTicketMessage.MessageType,
			Body:         parseSupportTicketMessage.Body,
			IsInternal:   parseSupportTicketMessage.IsInternal,
			CreatedAt:    parseSupportTicketMessage.CreatedAt,
			UpdatedAt:    parseSupportTicketMessage.UpdatedAt,
		})
	}
	for _, parseIncidentUpdate := range parseIncidentUpdates {
		parseResponse.IncidentUpdates = append(parseResponse.IncidentUpdates, &chatpb.IncidentUpdateEntry{
			Id:              parseIncidentUpdate.ID,
			IncidentId:      parseIncidentUpdate.IncidentID,
			Status:          parseIncidentUpdate.Status,
			Message:         parseIncidentUpdate.Message,
			IsPublic:        parseIncidentUpdate.IsPublic,
			PublishedAt:     parseIncidentUpdate.PublishedAt,
			CreatedByUserId: parseIncidentUpdate.CreatedByUserID,
			CreatedAt:       parseIncidentUpdate.CreatedAt,
		})
	}
	for _, parseNotificationOutbox := range parseNotificationOutboxRows {
		parseResponse.NotificationOutbox = append(parseResponse.NotificationOutbox, &chatpb.NotificationOutboxEntry{
			Id:              parseNotificationOutbox.ID,
			WorkspaceId:     parseNotificationOutbox.WorkspaceID,
			UserId:          parseNotificationOutbox.UserID,
			NotificationKey: parseNotificationOutbox.NotificationKey,
			ChannelKey:      parseNotificationOutbox.ChannelKey,
			TemplateKey:     parseNotificationOutbox.TemplateKey,
			Status:          parseNotificationOutbox.Status,
			Subject:         parseNotificationOutbox.Subject,
			BodyText:        parseNotificationOutbox.BodyText,
			PayloadJson:     parseNotificationOutbox.PayloadJSON,
			DedupeKey:       parseNotificationOutbox.DedupeKey,
			ScheduledAt:     parseNotificationOutbox.ScheduledAt,
			SentAt:          parseNotificationOutbox.SentAt,
			FailedAt:        parseNotificationOutbox.FailedAt,
			ErrorMessage:    parseNotificationOutbox.ErrorMessage,
			CreatedAt:       parseNotificationOutbox.CreatedAt,
			UpdatedAt:       parseNotificationOutbox.UpdatedAt,
		})
	}
	for _, parseBackgroundJob := range parseBackgroundJobs {
		parseResponse.BackgroundJobs = append(parseResponse.BackgroundJobs, &chatpb.BackgroundJobEntry{
			Id:           parseBackgroundJob.ID,
			JobKey:       parseBackgroundJob.JobKey,
			JobType:      parseBackgroundJob.JobType,
			QueueKey:     parseBackgroundJob.QueueKey,
			Status:       parseBackgroundJob.Status,
			AttemptCount: parseBackgroundJob.AttemptCount,
			MaxAttempts:  parseBackgroundJob.MaxAttempts,
			PayloadJson:  parseBackgroundJob.PayloadJSON,
			RunAfter:     parseBackgroundJob.RunAfter,
			StartedAt:    parseBackgroundJob.StartedAt,
			FinishedAt:   parseBackgroundJob.FinishedAt,
			ErrorMessage: parseBackgroundJob.ErrorMessage,
			CreatedAt:    parseBackgroundJob.CreatedAt,
			UpdatedAt:    parseBackgroundJob.UpdatedAt,
		})
	}
	parseFetchDuration := time.Since(parseFetchStart)
	if len(parseResponse.SiteConfigs) == 0 || len(parseResponse.FeatureFlags) == 0 {
		parseLogger.Warn(
			"rpc.GetSuperuserControlPlane: partial settings payload",
			slog.Int("site_configs", len(parseResponse.SiteConfigs)),
			slog.Int("feature_flags", len(parseResponse.FeatureFlags)),
			slog.Duration("duration", parseFetchDuration),
			slog.String("next_action", "verify settings bootstrap data and control-plane persistence"),
		)
	}
	parseLogAdminFetchOutcome(parseLogger, "rpc.GetSuperuserControlPlane", "settings", "platform", parseFetchDuration, len(parseResponse.SiteConfigs)+len(parseResponse.FeatureFlags))

	parseLogger.Info(
		"rpc.GetSuperuserControlPlane: complete",
		slog.Int("limit", int(parseLimit)),
		slog.Int("roles", len(parseResponse.Roles)),
		slog.Int("workspaces", len(parseResponse.Workspaces)),
		slog.Int("audit_logs", len(parseResponse.AuditLogs)),
		slog.Int("auth_sessions", len(parseResponse.AuthSessions)),
		slog.Int("background_jobs", len(parseResponse.BackgroundJobs)),
		slog.Duration("duration", parseFetchDuration),
	)
	return parseResponse, nil
}

// GetSuperuserSlices returns typed global superuser slices for dashboard operations.
func (parseS *chatServer) GetSuperuserSlices(parseCtx context.Context, parseReq *chatpb.GetSuperuserSlicesRequest) (*chatpb.GetSuperuserSlicesResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSuperuserSlices"))
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLimit int32
	var parseLookbackDays int32
	var parseWorkspaceListQuery *chatpb.AdminListQuery
	var parseWorkspaceStatusFilter string
	var parseSupportListQuery *chatpb.AdminListQuery
	var parseSupportStatusFilter string
	var parseIncidentListQuery *chatpb.AdminListQuery
	var parseIncidentStatusFilter string
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
		parseLookbackDays = parseReq.GetLookbackDays()
		parseWorkspaceListQuery = parseReq.GetWorkspaceListQuery()
		parseWorkspaceStatusFilter = strings.TrimSpace(parseReq.GetWorkspaceStatus())
		parseSupportListQuery = parseReq.GetSupportListQuery()
		parseSupportStatusFilter = strings.TrimSpace(parseReq.GetSupportStatus())
		parseIncidentListQuery = parseReq.GetIncidentListQuery()
		parseIncidentStatusFilter = strings.TrimSpace(parseReq.GetIncidentStatus())
	}
	parseLimit = parseClampSuperuserListLimit(parseLimit)
	parseWorkspaceListQueryShape := parseBuildSuperuserListQueryShape(parseLimit, parseWorkspaceListQuery)
	parseSupportListQueryShape := parseBuildSuperuserListQueryShape(parseLimit, parseSupportListQuery)
	parseIncidentListQueryShape := parseBuildSuperuserListQueryShape(parseLimit, parseIncidentListQuery)
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseLogger.Info(
		"rpc.GetSuperuserSlices: slice fetch",
		slog.Int("limit", int(parseLimit)),
		slog.Int("lookback_days", int(parseLookbackDays)),
	)
	parseScope := parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"superuser",
		"Superuser dashboard slices viewed",
		"{}",
		0,
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"workspace",
		"superuser",
		"Superuser drill-down access",
		"{}",
		0,
	)
	parseResponse := &chatpb.GetSuperuserSlicesResponse{
		Users:                   make([]*chatpb.AdminUserSummary, 0),
		UsageEvents:             make([]*chatpb.AdminUsageEvent, 0),
		SupportTickets:          make([]*chatpb.SupportTicketEntry, 0),
		BillingPlanOverages:     make([]*chatpb.BillingPlanOverageEntry, 0),
		BillingQuotaPolicies:    make([]*chatpb.BillingQuotaPolicyEntry, 0),
		BillingUpgradeTriggers:  make([]*chatpb.BillingUpgradeTriggerEntry, 0),
		Incidents:               make([]*chatpb.IncidentEntry, 0),
		Experiments:             make([]*chatpb.ExperimentEntry, 0),
		Workspaces:              make([]*chatpb.WorkspaceEntry, 0),
		WorkspaceCostGuardrails: make([]*chatpb.WorkspaceCostGuardrailEntry, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetSuperuserSlices: store unavailable")
		return parseResponse, nil
	}

	parseUserRows, parseErr := parseS.store.parseListAdminUsers(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser users: %v", parseErr)
	}
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser usage events: %v", parseErr)
	}
	parseSupportTicketRows, parseErr := parseS.store.parseListSupportTickets(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser support tickets: %v", parseErr)
	}
	parseOverageRows, parseErr := parseS.store.parseListBillingPlanOverages(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser billing plan overages: %v", parseErr)
	}
	parseQuotaRows, parseErr := parseS.store.parseListBillingQuotaPolicies(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser billing quota policies: %v", parseErr)
	}
	parseUpgradeTriggerRows, parseErr := parseS.store.parseListBillingUpgradeTriggers(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser billing upgrade triggers: %v", parseErr)
	}
	parseIncidentRows, parseErr := parseS.store.parseListIncidents(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser incidents: %v", parseErr)
	}
	parseExperimentRows, parseErr := parseS.store.parseListExperiments(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser experiments: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseS.store.parseListWorkspaces(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser workspaces: %v", parseErr)
	}
	parseCostGuardrailRows, parseErr := parseS.store.parseListWorkspaceCostGuardrails(int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list superuser workspace cost guardrails: %v", parseErr)
	}

	parseSupportTicketRows = parseFilterSupportTicketRows(parseSupportTicketRows, parseSupportStatusFilter, "", 0)
	parseSupportTicketRows = parseFilterSupportTicketRowsBySearch(parseSupportTicketRows, parseSupportListQueryShape.parseSearch)
	parseSortSupportTicketRows(parseSupportTicketRows, parseSupportListQueryShape.parseSortBy, parseSupportListQueryShape.isParseSortAscending)
	parseSupportTicketRows = parseApplyAdminSliceWindow(parseSupportTicketRows, parseSupportListQueryShape.parseOffset, parseSupportListQueryShape.parseLimit)

	parseIncidentRows = parseFilterSuperuserIncidentRows(parseIncidentRows, parseIncidentStatusFilter, parseIncidentListQueryShape.parseSearch)
	parseSortSuperuserIncidentRows(parseIncidentRows, parseIncidentListQueryShape.parseSortBy, parseIncidentListQueryShape.isParseSortAscending)
	parseIncidentRows = parseApplyAdminSliceWindow(parseIncidentRows, parseIncidentListQueryShape.parseOffset, parseIncidentListQueryShape.parseLimit)

	parseWorkspaceRows = parseFilterSuperuserWorkspaceRows(parseWorkspaceRows, parseWorkspaceStatusFilter, parseWorkspaceListQueryShape.parseSearch)
	parseSortSuperuserWorkspaceRows(parseWorkspaceRows, parseWorkspaceListQueryShape.parseSortBy, parseWorkspaceListQueryShape.isParseSortAscending)
	parseWorkspaceRows = parseApplyAdminSliceWindow(parseWorkspaceRows, parseWorkspaceListQueryShape.parseOffset, parseWorkspaceListQueryShape.parseLimit)

	for _, parseUserRow := range parseUserRows {
		parseResponse.Users = append(parseResponse.Users, parseBuildAdminUserSummary(parseUserRow))
	}
	for _, parseUsageRow := range parseUsageRows {
		parseResponse.UsageEvents = append(parseResponse.UsageEvents, parseBuildAdminUsageEvent(parseUsageRow))
	}
	for _, parseSupportTicketRow := range parseSupportTicketRows {
		parseResponse.SupportTickets = append(parseResponse.SupportTickets, &chatpb.SupportTicketEntry{
			Id:             parseSupportTicketRow.ID,
			TicketKey:      parseSupportTicketRow.TicketKey,
			WorkspaceId:    parseSupportTicketRow.WorkspaceID,
			UserId:         parseSupportTicketRow.UserID,
			Status:         parseSupportTicketRow.Status,
			Priority:       parseSupportTicketRow.Priority,
			Subject:        parseSupportTicketRow.Subject,
			Body:           parseSupportTicketRow.Body,
			AssigneeUserId: parseSupportTicketRow.AssigneeUserID,
			ResolutionNote: parseSupportTicketRow.ResolutionNote,
			CreatedAt:      parseSupportTicketRow.CreatedAt,
			UpdatedAt:      parseSupportTicketRow.UpdatedAt,
		})
	}
	for _, parseOverageRow := range parseOverageRows {
		parseResponse.BillingPlanOverages = append(parseResponse.BillingPlanOverages, &chatpb.BillingPlanOverageEntry{
			Id:                parseOverageRow.ID,
			PlanCode:          parseOverageRow.PlanCode,
			MeterKey:          parseOverageRow.MeterKey,
			IncludedUnits:     parseOverageRow.IncludedUnits,
			SoftLimitUnits:    parseOverageRow.SoftLimitUnits,
			HardLimitUnits:    parseOverageRow.HardLimitUnits,
			OverageUnitSize:   parseOverageRow.OverageUnitSize,
			OveragePriceCents: parseOverageRow.OveragePriceCents,
			BillingInterval:   parseOverageRow.BillingInterval,
			UpdatedAt:         parseOverageRow.UpdatedAt,
		})
	}
	for _, parseQuotaRow := range parseQuotaRows {
		parseResponse.BillingQuotaPolicies = append(parseResponse.BillingQuotaPolicies, &chatpb.BillingQuotaPolicyEntry{
			Id:              parseQuotaRow.ID,
			PlanCode:        parseQuotaRow.PlanCode,
			QuotaKey:        parseQuotaRow.QuotaKey,
			SoftLimitValue:  parseQuotaRow.SoftLimitValue,
			HardLimitValue:  parseQuotaRow.HardLimitValue,
			ResetInterval:   parseQuotaRow.ResetInterval,
			EnforcementMode: parseQuotaRow.EnforcementMode,
			UpdatedAt:       parseQuotaRow.UpdatedAt,
		})
	}
	for _, parseUpgradeTriggerRow := range parseUpgradeTriggerRows {
		parseResponse.BillingUpgradeTriggers = append(parseResponse.BillingUpgradeTriggers, &chatpb.BillingUpgradeTriggerEntry{
			Id:               parseUpgradeTriggerRow.ID,
			PlanCode:         parseUpgradeTriggerRow.PlanCode,
			TriggerKey:       parseUpgradeTriggerRow.TriggerKey,
			ThresholdPercent: parseUpgradeTriggerRow.ThresholdPercent,
			UpgradePlanCode:  parseUpgradeTriggerRow.UpgradePlanCode,
			Message:          parseUpgradeTriggerRow.Message,
			CtaLabel:         parseUpgradeTriggerRow.CTALabel,
			CtaUrl:           parseUpgradeTriggerRow.CTAURL,
			IsEnabled:        parseUpgradeTriggerRow.IsEnabled,
			UpdatedAt:        parseUpgradeTriggerRow.UpdatedAt,
		})
	}
	for _, parseIncidentRow := range parseIncidentRows {
		parseResponse.Incidents = append(parseResponse.Incidents, &chatpb.IncidentEntry{
			Id:            parseIncidentRow.ID,
			IncidentKey:   parseIncidentRow.IncidentKey,
			SloKey:        parseIncidentRow.SLOKey,
			Severity:      parseIncidentRow.Severity,
			Status:        parseIncidentRow.Status,
			Title:         parseIncidentRow.Title,
			Summary:       parseIncidentRow.Summary,
			StartedAt:     parseIncidentRow.StartedAt,
			ResolvedAt:    parseIncidentRow.ResolvedAt,
			PostmortemUrl: parseIncidentRow.PostmortemURL,
			UpdatedAt:     parseIncidentRow.UpdatedAt,
		})
	}
	for _, parseExperimentRow := range parseExperimentRows {
		parseResponse.Experiments = append(parseResponse.Experiments, &chatpb.ExperimentEntry{
			Id:            parseExperimentRow.ID,
			ExperimentKey: parseExperimentRow.ExperimentKey,
			Name:          parseExperimentRow.Name,
			Status:        parseExperimentRow.Status,
			VariantsJson:  parseExperimentRow.VariantsJSON,
			AudienceJson:  parseExperimentRow.AudienceJSON,
			StartAt:       parseExperimentRow.StartAt,
			EndAt:         parseExperimentRow.EndAt,
			UpdatedAt:     parseExperimentRow.UpdatedAt,
		})
	}
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		parseResponse.Workspaces = append(parseResponse.Workspaces, &chatpb.WorkspaceEntry{
			Id:           parseWorkspaceRow.ID,
			WorkspaceKey: parseWorkspaceRow.WorkspaceKey,
			Slug:         parseWorkspaceRow.Slug,
			Name:         parseWorkspaceRow.Name,
			PlanCode:     parseWorkspaceRow.PlanCode,
			Status:       parseWorkspaceRow.Status,
			OwnerUserId:  parseWorkspaceRow.OwnerUserID,
			SettingsJson: parseWorkspaceRow.SettingsJSON,
			CreatedAt:    parseWorkspaceRow.CreatedAt,
			UpdatedAt:    parseWorkspaceRow.UpdatedAt,
		})
	}
	for _, parseCostGuardrailRow := range parseCostGuardrailRows {
		parseResponse.WorkspaceCostGuardrails = append(parseResponse.WorkspaceCostGuardrails, &chatpb.WorkspaceCostGuardrailEntry{
			Id:                     parseCostGuardrailRow.ID,
			WorkspaceId:            parseCostGuardrailRow.WorkspaceID,
			GuardrailKey:           parseCostGuardrailRow.GuardrailKey,
			DailyBudgetCents:       parseCostGuardrailRow.DailyBudgetCents,
			MonthlyBudgetCents:     parseCostGuardrailRow.MonthlyBudgetCents,
			MaxCostPerRequestCents: parseCostGuardrailRow.MaxCostPerRequestCents,
			AlertThresholdPercent:  parseCostGuardrailRow.AlertThresholdPercent,
			ActionMode:             parseCostGuardrailRow.ActionMode,
			UpdatedAt:              parseCostGuardrailRow.UpdatedAt,
		})
	}

	parseLogger.Info(
		"rpc.GetSuperuserSlices: complete",
		slog.Int("users", len(parseResponse.Users)),
		slog.Int("usage_events", len(parseResponse.UsageEvents)),
		slog.Int("support_tickets", len(parseResponse.SupportTickets)),
		slog.Int("pricing_overages", len(parseResponse.BillingPlanOverages)),
		slog.Int("pricing_quotas", len(parseResponse.BillingQuotaPolicies)),
		slog.Int("pricing_triggers", len(parseResponse.BillingUpgradeTriggers)),
		slog.Int("incidents", len(parseResponse.Incidents)),
		slog.Int("experiments", len(parseResponse.Experiments)),
		slog.Int("workspaces", len(parseResponse.Workspaces)),
		slog.Int("cost_guardrails", len(parseResponse.WorkspaceCostGuardrails)),
	)
	return parseResponse, nil
}

// parseFilterSuperuserWorkspaceRows filters workspace rows by optional status and search values.
func parseFilterSuperuserWorkspaceRows(parseRows []parseWorkspaceRow, parseStatusFilter string, parseSearch string) []parseWorkspaceRow {
	parseStatusFilter = strings.TrimSpace(strings.ToLower(parseStatusFilter))
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	parseFilteredRows := make([]parseWorkspaceRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseStatusFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.Status)) != parseStatusFilter {
			continue
		}
		if parseSearch != "" {
			if !strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.WorkspaceKey)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Slug)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Name)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.PlanCode)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Status)), parseSearch) &&
				!strings.Contains(strings.TrimSpace(strings.ToLower(parseRow.SettingsJSON)), parseSearch) {
				continue
			}
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseSortSuperuserWorkspaceRows sorts workspace rows by one typed sort key and direction.
func parseSortSuperuserWorkspaceRows(parseRows []parseWorkspaceRow, parseSortBy string, isParseSortAscending bool) {
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
		case "workspace_key":
			return parseCompareAdminString(parseLeftRow.WorkspaceKey, parseRightRow.WorkspaceKey, isParseSortAscending)
		case "slug":
			return parseCompareAdminString(parseLeftRow.Slug, parseRightRow.Slug, isParseSortAscending)
		case "name":
			return parseCompareAdminString(parseLeftRow.Name, parseRightRow.Name, isParseSortAscending)
		case "plan_code":
			return parseCompareAdminString(parseLeftRow.PlanCode, parseRightRow.PlanCode, isParseSortAscending)
		case "status":
			return parseCompareAdminString(parseLeftRow.Status, parseRightRow.Status, isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		case "updated_at":
			return parseCompareAdminString(parseLeftRow.UpdatedAt, parseRightRow.UpdatedAt, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseFilterSuperuserIncidentRows filters incident rows by optional status and search values.
func parseFilterSuperuserIncidentRows(parseRows []parseIncidentRow, parseStatusFilter string, parseSearch string) []parseIncidentRow {
	parseStatusFilter = strings.TrimSpace(strings.ToLower(parseStatusFilter))
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	parseFilteredRows := make([]parseIncidentRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseStatusFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.Status)) != parseStatusFilter {
			continue
		}
		if parseSearch != "" {
			if !strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.IncidentKey)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.SLOKey)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Severity)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Status)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Title)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Summary)), parseSearch) {
				continue
			}
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseSortSuperuserIncidentRows sorts incident rows by one typed sort key and direction.
func parseSortSuperuserIncidentRows(parseRows []parseIncidentRow, parseSortBy string, isParseSortAscending bool) {
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
		case "incident_key":
			return parseCompareAdminString(parseLeftRow.IncidentKey, parseRightRow.IncidentKey, isParseSortAscending)
		case "severity":
			return parseCompareAdminString(parseLeftRow.Severity, parseRightRow.Severity, isParseSortAscending)
		case "status":
			return parseCompareAdminString(parseLeftRow.Status, parseRightRow.Status, isParseSortAscending)
		case "started_at":
			return parseCompareAdminString(parseLeftRow.StartedAt, parseRightRow.StartedAt, isParseSortAscending)
		case "resolved_at":
			return parseCompareAdminString(parseLeftRow.ResolvedAt, parseRightRow.ResolvedAt, isParseSortAscending)
		case "updated_at":
			return parseCompareAdminString(parseLeftRow.UpdatedAt, parseRightRow.UpdatedAt, isParseSortAscending)
		default:
			return false
		}
	})
}
