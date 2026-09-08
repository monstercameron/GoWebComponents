package app

import (
	"context"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseRequireSuperuserBillingMutationConfirmation enforces explicit confirmation and one non-empty reason for superuser pricing mutations.
func parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed bool, parseReason string) (string, error) {
	if !isParseConfirmed {
		return "", status.Error(codes.InvalidArgument, "superuser pricing mutation confirmation is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return "", status.Error(codes.InvalidArgument, "superuser pricing mutation reason is required")
	}
	return parseReason, nil
}

// parseBuildSuperuserBillingPlanEntry maps one billing plan row into protobuf form.
func parseBuildSuperuserBillingPlanEntry(parseRow parseBillingPlanRow) *chatpb.BillingPlanEntry {
	return &chatpb.BillingPlanEntry{
		PlanCode:                parseRow.PlanCode,
		PlanName:                parseRow.PlanName,
		PlanRank:                parseRow.PlanRank,
		IsActive:                parseRow.IsActive,
		MonthlyBaseCents:        parseRow.MonthlyBaseCents,
		YearlyBaseCents:         parseRow.YearlyBaseCents,
		IncludedTokensMonthly:   parseRow.IncludedTokensMonthly,
		IncludedSeats:           parseRow.IncludedSeats,
		MaxSeats:                parseRow.MaxSeats,
		SupportsPriority:        parseRow.SupportsPriority,
		SupportsTeamWorkspace:   parseRow.SupportsTeamWorkspace,
		SupportsSso:             parseRow.SupportsSSO,
		CreatedAt:               parseRow.CreatedAt,
		UpdatedAt:               parseRow.UpdatedAt,
		MonthlyPlatformFeeCents: parseRow.MonthlyPlatformFeeCents,
		UsagePremiumBasisPoints: parseRow.UsagePremiumBasisPoints,
		WorkspaceMode:           parseRow.WorkspaceMode,
		MinSeats:                parseRow.MinSeats,
		SupportsCollaboration:   parseRow.SupportsCollaboration,
		SupportsWorkspaceAdmin:  parseRow.SupportsWorkspaceAdmin,
	}
}

// parseBuildSuperuserBillingPlanEntitlementEntry maps one billing-plan entitlement row into protobuf form.
func parseBuildSuperuserBillingPlanEntitlementEntry(parsePlanCode string, parseRow parseBillingPlanEntitlementRow) *chatpb.BillingPlanEntitlementEntry {
	return &chatpb.BillingPlanEntitlementEntry{
		PlanCode:         parsePlanCode,
		EntitlementKey:   parseRow.EntitlementKey,
		EntitlementValue: parseRow.EntitlementValue,
		UpdatedAt:        parseRow.UpdatedAt,
	}
}

// parseBuildSuperuserBillingPlanOverageEntry maps one billing overage row into protobuf form.
func parseBuildSuperuserBillingPlanOverageEntry(parseRow parseBillingPlanOverageRow) *chatpb.BillingPlanOverageEntry {
	return &chatpb.BillingPlanOverageEntry{
		Id:                parseRow.ID,
		PlanCode:          parseRow.PlanCode,
		MeterKey:          parseRow.MeterKey,
		IncludedUnits:     parseRow.IncludedUnits,
		SoftLimitUnits:    parseRow.SoftLimitUnits,
		HardLimitUnits:    parseRow.HardLimitUnits,
		OverageUnitSize:   parseRow.OverageUnitSize,
		OveragePriceCents: parseRow.OveragePriceCents,
		BillingInterval:   parseRow.BillingInterval,
		UpdatedAt:         parseRow.UpdatedAt,
	}
}

// parseBuildSuperuserBillingQuotaPolicyEntry maps one billing quota-policy row into protobuf form.
func parseBuildSuperuserBillingQuotaPolicyEntry(parseRow parseBillingQuotaPolicyRow) *chatpb.BillingQuotaPolicyEntry {
	return &chatpb.BillingQuotaPolicyEntry{
		Id:              parseRow.ID,
		PlanCode:        parseRow.PlanCode,
		QuotaKey:        parseRow.QuotaKey,
		SoftLimitValue:  parseRow.SoftLimitValue,
		HardLimitValue:  parseRow.HardLimitValue,
		ResetInterval:   parseRow.ResetInterval,
		EnforcementMode: parseRow.EnforcementMode,
		UpdatedAt:       parseRow.UpdatedAt,
	}
}

// parseBuildSuperuserBillingUpgradeTriggerEntry maps one billing upgrade-trigger row into protobuf form.
func parseBuildSuperuserBillingUpgradeTriggerEntry(parseRow parseBillingUpgradeTriggerRow) *chatpb.BillingUpgradeTriggerEntry {
	return &chatpb.BillingUpgradeTriggerEntry{
		Id:               parseRow.ID,
		PlanCode:         parseRow.PlanCode,
		TriggerKey:       parseRow.TriggerKey,
		ThresholdPercent: parseRow.ThresholdPercent,
		UpgradePlanCode:  parseRow.UpgradePlanCode,
		Message:          parseRow.Message,
		CtaLabel:         parseRow.CTALabel,
		CtaUrl:           parseRow.CTAURL,
		IsEnabled:        parseRow.IsEnabled,
		UpdatedAt:        parseRow.UpdatedAt,
	}
}

// SetSuperuserBillingPlan upserts one superuser billing plan row.
func (parseS *chatServer) SetSuperuserBillingPlan(parseCtx context.Context, parseReq *chatpb.SetSuperuserBillingPlanRequest) (*chatpb.SetSuperuserBillingPlanResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.plan.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parsePlanName string
	var parsePlanRank int64
	var isParseActive bool
	var parseMonthlyBaseCents int64
	var parseMonthlyPlatformFeeCents int64
	var parseYearlyBaseCents int64
	var parseUsagePremiumBasisPoints int64
	var parseIncludedTokensMonthly int64
	var parseIncludedSeats int64
	var parseMinSeats int64
	var parseWorkspaceMode string
	var parseMaxSeats int64
	var isParseSupportsPriority bool
	var isParseSupportsCollaboration bool
	var isParseSupportsWorkspaceAdmin bool
	var isParseSupportsTeamWorkspace bool
	var isParseSupportsSSO bool
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parsePlanName = strings.TrimSpace(parseReq.GetPlanName())
		parsePlanRank = parseReq.GetPlanRank()
		isParseActive = parseReq.GetIsActive()
		parseMonthlyBaseCents = parseReq.GetMonthlyBaseCents()
		parseMonthlyPlatformFeeCents = parseReq.GetMonthlyPlatformFeeCents()
		parseYearlyBaseCents = parseReq.GetYearlyBaseCents()
		parseUsagePremiumBasisPoints = parseReq.GetUsagePremiumBasisPoints()
		parseIncludedTokensMonthly = parseReq.GetIncludedTokensMonthly()
		parseIncludedSeats = parseReq.GetIncludedSeats()
		parseMinSeats = parseReq.GetMinSeats()
		parseWorkspaceMode = strings.TrimSpace(parseReq.GetWorkspaceMode())
		parseMaxSeats = parseReq.GetMaxSeats()
		isParseSupportsPriority = parseReq.GetSupportsPriority()
		isParseSupportsCollaboration = parseReq.GetSupportsCollaboration()
		isParseSupportsWorkspaceAdmin = parseReq.GetSupportsWorkspaceAdmin()
		isParseSupportsTeamWorkspace = parseReq.GetSupportsTeamWorkspace()
		isParseSupportsSSO = parseReq.GetSupportsSso()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserBillingPlan(parseSuperuserBillingPlanWrite{
		PlanCode:                parsePlanCode,
		PlanName:                parsePlanName,
		PlanRank:                parsePlanRank,
		IsActive:                isParseActive,
		MonthlyBaseCents:        parseMonthlyBaseCents,
		MonthlyPlatformFeeCents: parseMonthlyPlatformFeeCents,
		YearlyBaseCents:         parseYearlyBaseCents,
		UsagePremiumBasisPoints: parseUsagePremiumBasisPoints,
		IncludedTokensMonthly:   parseIncludedTokensMonthly,
		IncludedSeats:           parseIncludedSeats,
		MinSeats:                parseMinSeats,
		WorkspaceMode:           parseWorkspaceMode,
		MaxSeats:                parseMaxSeats,
		SupportsPriority:        isParseSupportsPriority,
		SupportsCollaboration:   isParseSupportsCollaboration,
		SupportsWorkspaceAdmin:  isParseSupportsWorkspaceAdmin,
		SupportsTeamWorkspace:   isParseSupportsTeamWorkspace,
		SupportsSSO:             isParseSupportsSSO,
	})
	if parseErr != nil {
		if parseStatusErr, parseOk := status.FromError(parseErr); parseOk {
			return nil, status.Error(parseStatusErr.Code(), parseStatusErr.Message())
		}
		return nil, status.Errorf(codes.Internal, "set superuser billing plan: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_plan.set",
		"billing_plan",
		parseRow.PlanCode,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserBillingPlanResponse{
		Plan:   parseBuildSuperuserBillingPlanEntry(parseRow),
		Status: "updated",
	}, nil
}

// DeleteSuperuserBillingPlan deletes one superuser billing plan row.
func (parseS *chatServer) DeleteSuperuserBillingPlan(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserBillingPlanRequest) (*chatpb.DeleteSuperuserBillingPlanResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.plan.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserBillingPlan(parsePlanCode); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "billing plan not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser billing plan: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_plan.delete",
		"billing_plan",
		parseNormalizeSuperuserBillingPlanCode(parsePlanCode),
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserBillingPlanResponse{
		PlanCode: parseNormalizeSuperuserBillingPlanCode(parsePlanCode),
		Status:   "deleted",
	}, nil
}

// SetSuperuserBillingPlanEntitlement upserts one superuser billing-plan entitlement row.
func (parseS *chatServer) SetSuperuserBillingPlanEntitlement(parseCtx context.Context, parseReq *chatpb.SetSuperuserBillingPlanEntitlementRequest) (*chatpb.SetSuperuserBillingPlanEntitlementResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.plan_entitlement.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseEntitlementKey string
	var parseEntitlementValue string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseEntitlementKey = strings.TrimSpace(parseReq.GetEntitlementKey())
		parseEntitlementValue = strings.TrimSpace(parseReq.GetEntitlementValue())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserBillingPlanEntitlement(parseSuperuserBillingPlanEntitlementWrite{
		PlanCode:         parsePlanCode,
		EntitlementKey:   parseEntitlementKey,
		EntitlementValue: parseEntitlementValue,
	})
	if parseErr != nil {
		if parseStatusErr, parseOk := status.FromError(parseErr); parseOk {
			return nil, status.Error(parseStatusErr.Code(), parseStatusErr.Message())
		}
		return nil, status.Errorf(codes.Internal, "set superuser billing plan entitlement: %v", parseErr)
	}
	parsePlanCode = parseNormalizeSuperuserBillingPlanCode(parsePlanCode)
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_plan_entitlement.set",
		"billing_plan_entitlement",
		parsePlanCode+":"+parseRow.EntitlementKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserBillingPlanEntitlementResponse{
		Entitlement: parseBuildSuperuserBillingPlanEntitlementEntry(parsePlanCode, parseRow),
		Status:      "updated",
	}, nil
}

// DeleteSuperuserBillingPlanEntitlement deletes one superuser billing-plan entitlement row.
func (parseS *chatServer) DeleteSuperuserBillingPlanEntitlement(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserBillingPlanEntitlementRequest) (*chatpb.DeleteSuperuserBillingPlanEntitlementResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.plan_entitlement.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseEntitlementKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseEntitlementKey = strings.TrimSpace(parseReq.GetEntitlementKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserBillingPlanEntitlement(parsePlanCode, parseEntitlementKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "billing plan entitlement not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser billing plan entitlement: %v", parseErr)
	}
	parsePlanCode = parseNormalizeSuperuserBillingPlanCode(parsePlanCode)
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_plan_entitlement.delete",
		"billing_plan_entitlement",
		parsePlanCode+":"+parseEntitlementKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserBillingPlanEntitlementResponse{
		PlanCode:       parsePlanCode,
		EntitlementKey: parseEntitlementKey,
		Status:         "deleted",
	}, nil
}

// SetSuperuserBillingPlanOverage upserts one superuser pricing overage control row.
func (parseS *chatServer) SetSuperuserBillingPlanOverage(parseCtx context.Context, parseReq *chatpb.SetSuperuserBillingPlanOverageRequest) (*chatpb.SetSuperuserBillingPlanOverageResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.overage.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseMeterKey string
	var parseIncludedUnits int64
	var parseSoftLimitUnits int64
	var parseHardLimitUnits int64
	var parseOverageUnitSize int64
	var parseOveragePriceCents int64
	var parseBillingInterval string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseMeterKey = strings.TrimSpace(parseReq.GetMeterKey())
		parseIncludedUnits = parseReq.GetIncludedUnits()
		parseSoftLimitUnits = parseReq.GetSoftLimitUnits()
		parseHardLimitUnits = parseReq.GetHardLimitUnits()
		parseOverageUnitSize = parseReq.GetOverageUnitSize()
		parseOveragePriceCents = parseReq.GetOveragePriceCents()
		parseBillingInterval = strings.TrimSpace(parseReq.GetBillingInterval())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserBillingPlanOverage(parseSuperuserBillingPlanOverageWrite{
		PlanCode:          parsePlanCode,
		MeterKey:          parseMeterKey,
		IncludedUnits:     parseIncludedUnits,
		SoftLimitUnits:    parseSoftLimitUnits,
		HardLimitUnits:    parseHardLimitUnits,
		OverageUnitSize:   parseOverageUnitSize,
		OveragePriceCents: parseOveragePriceCents,
		BillingInterval:   parseBillingInterval,
	})
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set superuser billing plan overage: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_plan_overage.set",
		"billing_plan_overage",
		parseRow.PlanCode+":"+parseRow.MeterKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserBillingPlanOverageResponse{
		Overage: parseBuildSuperuserBillingPlanOverageEntry(parseRow),
		Status:  "updated",
	}, nil
}

// DeleteSuperuserBillingPlanOverage deletes one superuser pricing overage control row.
func (parseS *chatServer) DeleteSuperuserBillingPlanOverage(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserBillingPlanOverageRequest) (*chatpb.DeleteSuperuserBillingPlanOverageResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.overage.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseMeterKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseMeterKey = strings.TrimSpace(parseReq.GetMeterKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserBillingPlanOverage(parsePlanCode, parseMeterKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "billing plan overage not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser billing plan overage: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_plan_overage.delete",
		"billing_plan_overage",
		parsePlanCode+":"+parseMeterKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserBillingPlanOverageResponse{
		PlanCode: parsePlanCode,
		MeterKey: parseMeterKey,
		Status:   "deleted",
	}, nil
}

// SetSuperuserBillingQuotaPolicy upserts one superuser pricing quota policy row.
func (parseS *chatServer) SetSuperuserBillingQuotaPolicy(parseCtx context.Context, parseReq *chatpb.SetSuperuserBillingQuotaPolicyRequest) (*chatpb.SetSuperuserBillingQuotaPolicyResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.quota_policy.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseQuotaKey string
	var parseSoftLimitValue int64
	var parseHardLimitValue int64
	var parseResetInterval string
	var parseEnforcementMode string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseQuotaKey = strings.TrimSpace(parseReq.GetQuotaKey())
		parseSoftLimitValue = parseReq.GetSoftLimitValue()
		parseHardLimitValue = parseReq.GetHardLimitValue()
		parseResetInterval = strings.TrimSpace(parseReq.GetResetInterval())
		parseEnforcementMode = strings.TrimSpace(parseReq.GetEnforcementMode())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserBillingQuotaPolicy(parseSuperuserBillingQuotaPolicyWrite{
		PlanCode:        parsePlanCode,
		QuotaKey:        parseQuotaKey,
		SoftLimitValue:  parseSoftLimitValue,
		HardLimitValue:  parseHardLimitValue,
		ResetInterval:   parseResetInterval,
		EnforcementMode: parseEnforcementMode,
	})
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set superuser billing quota policy: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_quota_policy.set",
		"billing_quota_policy",
		parseRow.PlanCode+":"+parseRow.QuotaKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserBillingQuotaPolicyResponse{
		Policy: parseBuildSuperuserBillingQuotaPolicyEntry(parseRow),
		Status: "updated",
	}, nil
}

// DeleteSuperuserBillingQuotaPolicy deletes one superuser pricing quota policy row.
func (parseS *chatServer) DeleteSuperuserBillingQuotaPolicy(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserBillingQuotaPolicyRequest) (*chatpb.DeleteSuperuserBillingQuotaPolicyResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.quota_policy.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseQuotaKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseQuotaKey = strings.TrimSpace(parseReq.GetQuotaKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserBillingQuotaPolicy(parsePlanCode, parseQuotaKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "billing quota policy not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser billing quota policy: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_quota_policy.delete",
		"billing_quota_policy",
		parsePlanCode+":"+parseQuotaKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserBillingQuotaPolicyResponse{
		PlanCode: parsePlanCode,
		QuotaKey: parseQuotaKey,
		Status:   "deleted",
	}, nil
}

// SetSuperuserBillingUpgradeTrigger upserts one superuser pricing upgrade-trigger row.
func (parseS *chatServer) SetSuperuserBillingUpgradeTrigger(parseCtx context.Context, parseReq *chatpb.SetSuperuserBillingUpgradeTriggerRequest) (*chatpb.SetSuperuserBillingUpgradeTriggerResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.upgrade_trigger.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseTriggerKey string
	var parseThresholdPercent int64
	var parseUpgradePlanCode string
	var parseMessage string
	var parseCTALabel string
	var parseCTAURL string
	var isParseEnabled bool
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseTriggerKey = strings.TrimSpace(parseReq.GetTriggerKey())
		parseThresholdPercent = parseReq.GetThresholdPercent()
		parseUpgradePlanCode = strings.TrimSpace(parseReq.GetUpgradePlanCode())
		parseMessage = strings.TrimSpace(parseReq.GetMessage())
		parseCTALabel = strings.TrimSpace(parseReq.GetCtaLabel())
		parseCTAURL = strings.TrimSpace(parseReq.GetCtaUrl())
		isParseEnabled = parseReq.GetIsEnabled()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserBillingUpgradeTrigger(parseSuperuserBillingUpgradeTriggerWrite{
		PlanCode:         parsePlanCode,
		TriggerKey:       parseTriggerKey,
		ThresholdPercent: parseThresholdPercent,
		UpgradePlanCode:  parseUpgradePlanCode,
		Message:          parseMessage,
		CTALabel:         parseCTALabel,
		CTAURL:           parseCTAURL,
		IsEnabled:        isParseEnabled,
	})
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set superuser billing upgrade trigger: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_upgrade_trigger.set",
		"billing_upgrade_trigger",
		parseRow.PlanCode+":"+parseRow.TriggerKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserBillingUpgradeTriggerResponse{
		Trigger: parseBuildSuperuserBillingUpgradeTriggerEntry(parseRow),
		Status:  "updated",
	}, nil
}

// DeleteSuperuserBillingUpgradeTrigger deletes one superuser pricing upgrade-trigger row.
func (parseS *chatServer) DeleteSuperuserBillingUpgradeTrigger(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserBillingUpgradeTriggerRequest) (*chatpb.DeleteSuperuserBillingUpgradeTriggerResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.upgrade_trigger.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parsePlanCode string
	var parseTriggerKey string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parsePlanCode = strings.TrimSpace(parseReq.GetPlanCode())
		parseTriggerKey = strings.TrimSpace(parseReq.GetTriggerKey())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserBillingUpgradeTrigger(parsePlanCode, parseTriggerKey); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "billing upgrade trigger not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser billing upgrade trigger: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_upgrade_trigger.delete",
		"billing_upgrade_trigger",
		parsePlanCode+":"+parseTriggerKey,
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserBillingUpgradeTriggerResponse{
		PlanCode:   parsePlanCode,
		TriggerKey: parseTriggerKey,
		Status:     "deleted",
	}, nil
}

// SetSuperuserBillingDunningEvent upserts one superuser billing dunning-event row.
func (parseS *chatServer) SetSuperuserBillingDunningEvent(parseCtx context.Context, parseReq *chatpb.SetSuperuserBillingDunningEventRequest) (*chatpb.SetSuperuserBillingDunningEventResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.dunning_event.set")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseDunningEventID int64
	var parseCustomerID int64
	var parseSubscriptionID int64
	var parseInvoiceID int64
	var parseStatusValue string
	var parseAttemptCount int64
	var parseFailureReason string
	var parseNextAttemptAt string
	var parseResolvedAt string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseDunningEventID = parseReq.GetId()
		parseCustomerID = parseReq.GetCustomerId()
		parseSubscriptionID = parseReq.GetSubscriptionId()
		parseInvoiceID = parseReq.GetInvoiceId()
		parseStatusValue = strings.TrimSpace(parseReq.GetStatus())
		parseAttemptCount = parseReq.GetAttemptCount()
		parseFailureReason = strings.TrimSpace(parseReq.GetFailureReason())
		parseNextAttemptAt = strings.TrimSpace(parseReq.GetNextAttemptAt())
		parseResolvedAt = strings.TrimSpace(parseReq.GetResolvedAt())
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseRow, parseErr := parseS.store.parseUpsertSuperuserBillingDunningEvent(parseSuperuserBillingDunningEventWrite{
		ID:             parseDunningEventID,
		CustomerID:     parseCustomerID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseInvoiceID,
		Status:         parseStatusValue,
		AttemptCount:   parseAttemptCount,
		FailureReason:  parseFailureReason,
		NextAttemptAt:  parseNextAttemptAt,
		ResolvedAt:     parseResolvedAt,
	})
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set superuser billing dunning event: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_dunning_event.set",
		"billing_dunning_event",
		strconv.FormatInt(parseRow.ID, 10),
		parseReason,
		"{}",
		0,
	)
	return &chatpb.SetSuperuserBillingDunningEventResponse{
		Event:  parseBuildAdminBillingDunningEventEntry(parseRow),
		Status: parseBuildSuperuserSetMutationStatus(parseDunningEventID > 0),
	}, nil
}

// DeleteSuperuserBillingDunningEvent deletes one superuser billing dunning-event row.
func (parseS *chatServer) DeleteSuperuserBillingDunningEvent(parseCtx context.Context, parseReq *chatpb.DeleteSuperuserBillingDunningEventRequest) (*chatpb.DeleteSuperuserBillingDunningEventResponse, error) {
	parseSuperuserUserID, parseErr := parseS.parseRequireSuperuserMutationUserID(parseCtx, "superuser.billing.dunning_event.delete")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseDunningEventID int64
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseDunningEventID = parseReq.GetId()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseReason, parseErr = parseRequireSuperuserBillingMutationConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.store.parseDeleteSuperuserBillingDunningEvent(parseDunningEventID); parseErr != nil {
		if parseErr == errStoreSuperuserScopeMissing {
			return nil, status.Error(codes.NotFound, "billing dunning event not found")
		}
		return nil, status.Errorf(codes.Internal, "delete superuser billing dunning event: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseAdminAccessScope{isPlatformScope: true, adminUserID: parseSuperuserUserID},
		"admin.superuser.billing_dunning_event.delete",
		"billing_dunning_event",
		strconv.FormatInt(parseDunningEventID, 10),
		parseReason,
		"{}",
		0,
	)
	return &chatpb.DeleteSuperuserBillingDunningEventResponse{
		Id:     parseDunningEventID,
		Status: "deleted",
	}, nil
}
