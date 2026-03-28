package app

import (
	"context"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseRequireSuperuserReliabilityMutationConfirmation enforces explicit confirmation and one non-empty reason for superuser reliability mutations.
func parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed bool, parseReason string) (string, error) {
	if !isParseConfirmed {
		return "", status.Error(codes.InvalidArgument, "superuser reliability mutation confirmation is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return "", status.Error(codes.InvalidArgument, "superuser reliability mutation reason is required")
	}
	return parseReason, nil
}

// parseBuildSuperuserWorkspaceSSOConfigEntry maps one workspace SSO config row into protobuf form.
func parseBuildSuperuserWorkspaceSSOConfigEntry(parseRow parseWorkspaceSSOConfigRow) *chatpb.WorkspaceSSOConfigEntry {
	return &chatpb.WorkspaceSSOConfigEntry{
		Id:                  parseRow.ID,
		WorkspaceId:         parseRow.WorkspaceID,
		ProviderKey:         parseRow.ProviderKey,
		SamlEntrypoint:      parseRow.SAMLEntrypoint,
		SamlIssuer:          parseRow.SAMLIssuer,
		SamlCertificatePem:  parseRow.SAMLCertificatePEM,
		DomainsJson:         parseRow.DomainsJSON,
		IsEnabled:           parseRow.IsEnabled,
		UpdatedAt:           parseRow.UpdatedAt,
		ProviderType:        parseRow.ProviderType,
		OidcIssuerUrl:       parseRow.OIDCIssuerURL,
		OidcClientId:        parseRow.OIDCClientID,
		OidcClientSecretRef: parseRow.OIDCClientSecretRef,
		OidcScopesJson:      parseRow.OIDCScopesJSON,
		OidcClaimsJson:      parseRow.OIDCClaimsJSON,
	}
}

// parseBuildSuperuserDataRetentionPolicyEntry maps one data-retention policy row into protobuf form.
func parseBuildSuperuserDataRetentionPolicyEntry(parseRow parseDataRetentionPolicyRow) *chatpb.DataRetentionPolicyEntry {
	return &chatpb.DataRetentionPolicyEntry{
		Id:            parseRow.ID,
		WorkspaceId:   parseRow.WorkspaceID,
		ScopeKey:      parseRow.ScopeKey,
		RetentionDays: parseRow.RetentionDays,
		PurgeMode:     parseRow.PurgeMode,
		LegalHoldJson: parseRow.LegalHoldJSON,
		UpdatedAt:     parseRow.UpdatedAt,
	}
}

// parseBuildSuperuserComplianceControlEntry maps one compliance-control row into protobuf form.
func parseBuildSuperuserComplianceControlEntry(parseRow parseComplianceControlRow) *chatpb.ComplianceControlEntry {
	return &chatpb.ComplianceControlEntry{
		Id:           parseRow.ID,
		WorkspaceId:  parseRow.WorkspaceID,
		ControlKey:   parseRow.ControlKey,
		FrameworkKey: parseRow.FrameworkKey,
		Status:       parseRow.Status,
		OwnerUserId:  parseRow.OwnerUserID,
		EvidenceUrl:  parseRow.EvidenceURL,
		ReviewedAt:   parseRow.ReviewedAt,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseBuildSuperuserServiceLevelObjectiveEntry maps one service-level objective row into protobuf form.
func parseBuildSuperuserServiceLevelObjectiveEntry(parseRow parseServiceLevelObjectiveRow) *chatpb.ServiceLevelObjectiveEntry {
	return &chatpb.ServiceLevelObjectiveEntry{
		Id:                 parseRow.ID,
		SloKey:             parseRow.SLOKey,
		ServiceName:        parseRow.ServiceName,
		ObjectivePercent:   parseRow.ObjectivePercent,
		WindowDays:         parseRow.WindowDays,
		ErrorBudgetMinutes: parseRow.ErrorBudgetMinutes,
		StatusPageUrl:      parseRow.StatusPageURL,
		UpdatedAt:          parseRow.UpdatedAt,
	}
}

// SetSuperuserWorkspaceSSOConfig upserts one superuser workspace SSO config row.
func (parseS *chatServer) SetSuperuserWorkspaceSSOConfig(parseCtx context.Context, parseReq *chatpb.SetSuperuserWorkspaceSSOConfigRequest) (*chatpb.SetSuperuserWorkspaceSSOConfigResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.workspace_sso.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseProviderKey string
	var parseSAMLEntrypoint string
	var parseSAMLIssuer string
	var parseSAMLCertificatePEM string
	var parseProviderType string
	var parseOIDCIssuerURL string
	var parseOIDCClientID string
	var parseOIDCClientSecretRef string
	var parseOIDCScopesJSON string
	var parseOIDCClaimsJSON string
	var parseDomainsJSON string
	var isParseEnabled bool
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseProviderKey = strings.TrimSpace(parseReq.GetProviderKey())
		parseSAMLEntrypoint = strings.TrimSpace(parseReq.GetSamlEntrypoint())
		parseSAMLIssuer = strings.TrimSpace(parseReq.GetSamlIssuer())
		parseSAMLCertificatePEM = strings.TrimSpace(parseReq.GetSamlCertificatePem())
		parseProviderType = strings.TrimSpace(parseReq.GetProviderType())
		parseOIDCIssuerURL = strings.TrimSpace(parseReq.GetOidcIssuerUrl())
		parseOIDCClientID = strings.TrimSpace(parseReq.GetOidcClientId())
		parseOIDCClientSecretRef = strings.TrimSpace(parseReq.GetOidcClientSecretRef())
		parseOIDCScopesJSON = strings.TrimSpace(parseReq.GetOidcScopesJson())
		parseOIDCClaimsJSON = strings.TrimSpace(parseReq.GetOidcClaimsJson())
		parseDomainsJSON = strings.TrimSpace(parseReq.GetDomainsJson())
		isParseEnabled = parseReq.GetIsEnabled()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:         parseWorkspaceID,
		ProviderKey:         parseProviderKey,
		ProviderType:        parseProviderType,
		OIDCIssuerURL:       parseOIDCIssuerURL,
		OIDCClientID:        parseOIDCClientID,
		OIDCClientSecretRef: parseOIDCClientSecretRef,
		OIDCScopesJSON:      parseOIDCScopesJSON,
		OIDCClaimsJSON:      parseOIDCClaimsJSON,
		SAMLEntrypoint:      parseSAMLEntrypoint,
		SAMLIssuer:          parseSAMLIssuer,
		SAMLCertificatePEM:  parseSAMLCertificatePEM,
		DomainsJSON:         parseDomainsJSON,
		IsEnabled:           isParseEnabled,
	})
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "workspace sso config target not found")
		}
		return nil, status.Errorf(codes.Internal, "set superuser workspace sso config: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.workspace_sso_config.set",
		"workspace_sso_config",
		strconv.FormatInt(parseRow.WorkspaceID, 10)+":"+parseRow.ProviderKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserWorkspaceSSOConfigResponse{
		Config: parseBuildSuperuserWorkspaceSSOConfigEntry(parseRow),
		Status: "updated",
	}, nil
}

// DeleteSuperuserWorkspaceSSOConfig deletes one superuser workspace SSO config row.
func (parseS *chatServer) DeleteSuperuserWorkspaceSSOConfig(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserWorkspaceSSOConfigRequest) (*chatpb.DeleteSuperuserWorkspaceSSOConfigResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.workspace_sso.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseProviderKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseProviderKey = strings.TrimSpace(parseReq.GetProviderKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserWorkspaceSSOConfig(parseWorkspaceID, parseProviderKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "workspace sso config not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser workspace sso config: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.workspace_sso_config.delete",
		"workspace_sso_config",
		strconv.FormatInt(parseWorkspaceID, 10)+":"+parseProviderKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserWorkspaceSSOConfigResponse{
		WorkspaceId: parseWorkspaceID,
		ProviderKey: parseProviderKey,
		Status:      "deleted",
	}, nil
}

// SetSuperuserDataRetentionPolicy upserts one superuser data-retention policy row.
func (parseS *chatServer) SetSuperuserDataRetentionPolicy(parseCtx context.Context, parseReq *chatpb.SetSuperuserDataRetentionPolicyRequest) (*chatpb.SetSuperuserDataRetentionPolicyResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.retention_policy.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseScopeKey string
	var parseRetentionDays int64
	var parsePurgeMode string
	var parseLegalHoldJSON string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseScopeKey = strings.TrimSpace(parseReq.GetScopeKey())
		parseRetentionDays = parseReq.GetRetentionDays()
		parsePurgeMode = strings.TrimSpace(parseReq.GetPurgeMode())
		parseLegalHoldJSON = strings.TrimSpace(parseReq.GetLegalHoldJson())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserDataRetentionPolicy(parseSuperuserDataRetentionPolicyWrite{
		WorkspaceID:   parseWorkspaceID,
		ScopeKey:      parseScopeKey,
		RetentionDays: parseRetentionDays,
		PurgeMode:     parsePurgeMode,
		LegalHoldJSON: parseLegalHoldJSON,
	})
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "retention policy target not found")
		}
		return nil, status.Errorf(codes.Internal, "set superuser data retention policy: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.data_retention_policy.set",
		"data_retention_policy",
		strconv.FormatInt(parseRow.WorkspaceID, 10)+":"+parseRow.ScopeKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserDataRetentionPolicyResponse{
		Policy: parseBuildSuperuserDataRetentionPolicyEntry(parseRow),
		Status: "updated",
	}, nil
}

// DeleteSuperuserDataRetentionPolicy deletes one superuser data-retention policy row.
func (parseS *chatServer) DeleteSuperuserDataRetentionPolicy(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserDataRetentionPolicyRequest) (*chatpb.DeleteSuperuserDataRetentionPolicyResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.retention_policy.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseScopeKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseScopeKey = strings.TrimSpace(parseReq.GetScopeKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserDataRetentionPolicy(parseWorkspaceID, parseScopeKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "retention policy not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser data retention policy: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.data_retention_policy.delete",
		"data_retention_policy",
		strconv.FormatInt(parseWorkspaceID, 10)+":"+parseScopeKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserDataRetentionPolicyResponse{
		WorkspaceId: parseWorkspaceID,
		ScopeKey:    parseScopeKey,
		Status:      "deleted",
	}, nil
}

// SetSuperuserComplianceControl upserts one superuser compliance-control row.
func (parseS *chatServer) SetSuperuserComplianceControl(parseCtx context.Context, parseReq *chatpb.SetSuperuserComplianceControlRequest) (*chatpb.SetSuperuserComplianceControlResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.compliance_control.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseControlKey string
	var parseFrameworkKey string
	var parseStatusValue string
	var parseOwnerUserID int64
	var parseEvidenceURL string
	var parseReviewedAt string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseControlKey = strings.TrimSpace(parseReq.GetControlKey())
		parseFrameworkKey = strings.TrimSpace(parseReq.GetFrameworkKey())
		parseStatusValue = strings.TrimSpace(parseReq.GetStatus())
		parseOwnerUserID = parseReq.GetOwnerUserId()
		parseEvidenceURL = strings.TrimSpace(parseReq.GetEvidenceUrl())
		parseReviewedAt = strings.TrimSpace(parseReq.GetReviewedAt())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserComplianceControl(parseSuperuserComplianceControlWrite{
		WorkspaceID:  parseWorkspaceID,
		ControlKey:   parseControlKey,
		FrameworkKey: parseFrameworkKey,
		Status:       parseStatusValue,
		OwnerUserID:  parseOwnerUserID,
		EvidenceURL:  parseEvidenceURL,
		ReviewedAt:   parseReviewedAt,
	})
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "compliance control target not found")
		}
		return nil, status.Errorf(codes.Internal, "set superuser compliance control: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.compliance_control.set",
		"compliance_control",
		strconv.FormatInt(parseRow.WorkspaceID, 10)+":"+parseRow.ControlKey+":"+parseRow.FrameworkKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserComplianceControlResponse{
		Control: parseBuildSuperuserComplianceControlEntry(parseRow),
		Status:  "updated",
	}, nil
}

// DeleteSuperuserComplianceControl deletes one superuser compliance-control row.
func (parseS *chatServer) DeleteSuperuserComplianceControl(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserComplianceControlRequest) (*chatpb.DeleteSuperuserComplianceControlResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.compliance_control.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseControlKey string
	var parseFrameworkKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseControlKey = strings.TrimSpace(parseReq.GetControlKey())
		parseFrameworkKey = strings.TrimSpace(parseReq.GetFrameworkKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserComplianceControl(parseWorkspaceID, parseControlKey, parseFrameworkKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "compliance control not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser compliance control: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.compliance_control.delete",
		"compliance_control",
		strconv.FormatInt(parseWorkspaceID, 10)+":"+parseControlKey+":"+parseFrameworkKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserComplianceControlResponse{
		WorkspaceId:  parseWorkspaceID,
		ControlKey:   parseControlKey,
		FrameworkKey: parseFrameworkKey,
		Status:       "deleted",
	}, nil
}

// SetSuperuserServiceLevelObjective upserts one superuser SLO row.
func (parseS *chatServer) SetSuperuserServiceLevelObjective(parseCtx context.Context, parseReq *chatpb.SetSuperuserServiceLevelObjectiveRequest) (*chatpb.SetSuperuserServiceLevelObjectiveResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.service_level_objective.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseSLOKey string
	var parseServiceName string
	var parseObjectivePercent float64
	var parseWindowDays int64
	var parseErrorBudgetMinutes int64
	var parseStatusPageURL string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseSLOKey = strings.TrimSpace(parseReq.GetSloKey())
		parseServiceName = strings.TrimSpace(parseReq.GetServiceName())
		parseObjectivePercent = parseReq.GetObjectivePercent()
		parseWindowDays = parseReq.GetWindowDays()
		parseErrorBudgetMinutes = parseReq.GetErrorBudgetMinutes()
		parseStatusPageURL = strings.TrimSpace(parseReq.GetStatusPageUrl())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserServiceLevelObjective(parseSuperuserServiceLevelObjectiveWrite{
		SLOKey:             parseSLOKey,
		ServiceName:        parseServiceName,
		ObjectivePercent:   parseObjectivePercent,
		WindowDays:         parseWindowDays,
		ErrorBudgetMinutes: parseErrorBudgetMinutes,
		StatusPageURL:      parseStatusPageURL,
	})
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "service level objective target not found")
		}
		return nil, status.Errorf(codes.Internal, "set superuser service level objective: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.service_level_objective.set",
		"service_level_objective",
		parseRow.SLOKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserServiceLevelObjectiveResponse{
		Slo:    parseBuildSuperuserServiceLevelObjectiveEntry(parseRow),
		Status: "updated",
	}, nil
}

// DeleteSuperuserServiceLevelObjective deletes one superuser SLO row.
func (parseS *chatServer) DeleteSuperuserServiceLevelObjective(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserServiceLevelObjectiveRequest) (*chatpb.DeleteSuperuserServiceLevelObjectiveResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.service_level_objective.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseSLOKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseSLOKey = strings.TrimSpace(parseReq.GetSloKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserServiceLevelObjective(parseSLOKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "service level objective not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser service level objective: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.service_level_objective.delete",
		"service_level_objective",
		parseSLOKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserServiceLevelObjectiveResponse{
		SloKey: parseSLOKey,
		Status: "deleted",
	}, nil
}

// SetSuperuserIncident upserts one superuser incident row.
func (parseS *chatServer) SetSuperuserIncident(parseCtx context.Context, parseReq *chatpb.SetSuperuserIncidentRequest) (*chatpb.SetSuperuserIncidentResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.incident.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseIncidentKey string
	var parseSLOKey string
	var parseSeverity string
	var parseStatusValue string
	var parseTitle string
	var parseSummary string
	var parseStartedAt string
	var parseResolvedAt string
	var parsePostmortemURL string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseIncidentKey = strings.TrimSpace(parseReq.GetIncidentKey())
		parseSLOKey = strings.TrimSpace(parseReq.GetSloKey())
		parseSeverity = strings.TrimSpace(parseReq.GetSeverity())
		parseStatusValue = strings.TrimSpace(parseReq.GetStatus())
		parseTitle = strings.TrimSpace(parseReq.GetTitle())
		parseSummary = strings.TrimSpace(parseReq.GetSummary())
		parseStartedAt = strings.TrimSpace(parseReq.GetStartedAt())
		parseResolvedAt = strings.TrimSpace(parseReq.GetResolvedAt())
		parsePostmortemURL = strings.TrimSpace(parseReq.GetPostmortemUrl())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserIncident(parseSuperuserIncidentWrite{
		IncidentKey:   parseIncidentKey,
		SLOKey:        parseSLOKey,
		Severity:      parseSeverity,
		Status:        parseStatusValue,
		Title:         parseTitle,
		Summary:       parseSummary,
		StartedAt:     parseStartedAt,
		ResolvedAt:    parseResolvedAt,
		PostmortemURL: parsePostmortemURL,
	})
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "incident target not found")
		}
		return nil, status.Errorf(codes.Internal, "set superuser incident: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.incident.set",
		"incident",
		parseRow.IncidentKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserIncidentResponse{
		Incident: parseBuildAdminIncidentEntry(parseRow),
		Status:   "updated",
	}, nil
}

// DeleteSuperuserIncident deletes one superuser incident row.
func (parseS *chatServer) DeleteSuperuserIncident(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserIncidentRequest) (*chatpb.DeleteSuperuserIncidentResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.incident.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseIncidentKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseIncidentKey = strings.TrimSpace(parseReq.GetIncidentKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserIncident(parseIncidentKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "incident not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser incident: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.incident.delete",
		"incident",
		parseIncidentKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserIncidentResponse{
		IncidentKey: parseIncidentKey,
		Status:      "deleted",
	}, nil
}

// SetSuperuserIncidentUpdate upserts one superuser incident-update row.
func (parseS *chatServer) SetSuperuserIncidentUpdate(parseCtx context.Context, parseReq *chatpb.SetSuperuserIncidentUpdateRequest) (*chatpb.SetSuperuserIncidentUpdateResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.incident_update.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseIncidentUpdateID int64
	var parseIncidentID int64
	var parseStatusValue string
	var parseMessage string
	var isParsePublic bool
	var parsePublishedAt string
	var parseCreatedByUserID int64
	var parseCreatedAt string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseIncidentUpdateID = parseReq.GetId()
		parseIncidentID = parseReq.GetIncidentId()
		parseStatusValue = strings.TrimSpace(parseReq.GetStatus())
		parseMessage = strings.TrimSpace(parseReq.GetMessage())
		isParsePublic = parseReq.GetIsPublic()
		parsePublishedAt = strings.TrimSpace(parseReq.GetPublishedAt())
		parseCreatedByUserID = parseReq.GetCreatedByUserId()
		parseCreatedAt = strings.TrimSpace(parseReq.GetCreatedAt())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserIncidentUpdate(parseSuperuserIncidentUpdateWrite{
		ID:              parseIncidentUpdateID,
		IncidentID:      parseIncidentID,
		Status:          parseStatusValue,
		Message:         parseMessage,
		IsPublic:        isParsePublic,
		PublishedAt:     parsePublishedAt,
		CreatedByUserID: parseCreatedByUserID,
		CreatedAt:       parseCreatedAt,
	})
	if parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "incident update target not found")
		}
		return nil, status.Errorf(codes.Internal, "set superuser incident update: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.incident_update.set",
		"incident_update",
		strconv.FormatInt(parseRow.ID, 10),
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserIncidentUpdateResponse{
		IncidentUpdate: parseBuildAdminIncidentUpdateEntry(parseRow),
		Status:         parseBuildSuperuserSetMutationStatus(parseIncidentUpdateID > 0),
	}, nil
}

// DeleteSuperuserIncidentUpdate deletes one superuser incident-update row.
func (parseS *chatServer) DeleteSuperuserIncidentUpdate(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserIncidentUpdateRequest) (*chatpb.DeleteSuperuserIncidentUpdateResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.reliability.incident_update.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseIncidentUpdateID int64
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseIncidentUpdateID = parseReq.GetId()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserReliabilityMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserIncidentUpdate(parseIncidentUpdateID); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "incident update not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser incident update: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.incident_update.delete",
		"incident_update",
		strconv.FormatInt(parseIncidentUpdateID, 10),
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserIncidentUpdateResponse{
		Id:     parseIncidentUpdateID,
		Status: "deleted",
	}, nil
}
