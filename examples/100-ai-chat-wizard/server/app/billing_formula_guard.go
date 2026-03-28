package app

import (
	"fmt"
	"math"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	parseBillingLineTypePlatformFee    = "platform_fee"
	parseBillingLineTypeUsageCost      = "usage_cost"
	parseBillingLineTypeServicePremium = "service_premium"
)

type parseUsageBasedBillingPreview struct {
	platformFeeCents    int64
	usageCostCents      int64
	servicePremiumCents int64
	totalCents          int64
}

// parseBuildUsageBasedBillingPreview computes one usage-based billing preview using platform fee plus usage plus service premium.
func parseBuildUsageBasedBillingPreview(parsePlatformFeeCents int64, parseUsageCostCents int64, parseUsagePremiumBasisPoints int64) (parseUsageBasedBillingPreview, error) {
	if parsePlatformFeeCents < 0 {
		return parseUsageBasedBillingPreview{}, status.Error(codes.InvalidArgument, "platform fee must be non-negative")
	}
	if parseUsageCostCents < 0 {
		return parseUsageBasedBillingPreview{}, status.Error(codes.InvalidArgument, "usage cost must be non-negative")
	}
	if parseUsagePremiumBasisPoints < 0 {
		return parseUsageBasedBillingPreview{}, status.Error(codes.InvalidArgument, "usage premium basis points must be non-negative")
	}
	if parseUsagePremiumBasisPoints > 100_000 {
		return parseUsageBasedBillingPreview{}, status.Error(codes.InvalidArgument, "usage premium basis points exceed maximum")
	}
	parseServicePremiumCents := int64(math.Round(float64(parseUsageCostCents*parseUsagePremiumBasisPoints) / 10_000.0))
	parseTotalCents := parsePlatformFeeCents + parseUsageCostCents + parseServicePremiumCents
	return parseUsageBasedBillingPreview{
		platformFeeCents:    parsePlatformFeeCents,
		usageCostCents:      parseUsageCostCents,
		servicePremiumCents: parseServicePremiumCents,
		totalCents:          parseTotalCents,
	}, nil
}

// parseValidateUsageBasedBillingPlanWrite validates pricing plan fields against the usage-based billing model.
func parseValidateUsageBasedBillingPlanWrite(parseWrite parseSuperuserBillingPlanWrite) error {
	if parseWrite.MonthlyBaseCents < 0 || parseWrite.YearlyBaseCents < 0 {
		return status.Error(codes.InvalidArgument, "platform fee cents must be non-negative")
	}
	if parseWrite.IncludedTokensMonthly < 0 {
		return status.Error(codes.InvalidArgument, "included tokens must be non-negative")
	}
	parsePlanCode := parseNormalizeSuperuserBillingPlanCode(parseWrite.PlanCode)
	if parseWrite.IsActive && parsePlanCode != "enterprise" && parseWrite.MonthlyBaseCents <= 0 && parseWrite.YearlyBaseCents <= 0 {
		return status.Error(codes.InvalidArgument, "active usage-based plans must include a platform fee")
	}
	return nil
}

// parseValidatePlanBoundaryPlanWrite validates canonical Pro/Team/Enterprise behavioral boundaries.
func parseValidatePlanBoundaryPlanWrite(parseWrite parseSuperuserBillingPlanWrite) error {
	parsePlanCode := parseNormalizeSuperuserBillingPlanCode(parseWrite.PlanCode)
	parseIncludedSeats := parseWrite.IncludedSeats
	if parseIncludedSeats <= 0 {
		parseIncludedSeats = 1
	}
	parseMaxSeats := parseWrite.MaxSeats
	if parseMaxSeats <= 0 {
		parseMaxSeats = parseIncludedSeats
	}
	if parseMaxSeats < parseIncludedSeats {
		parseMaxSeats = parseIncludedSeats
	}
	switch parsePlanCode {
	case "pro":
		if parseIncludedSeats != 1 || parseMaxSeats != 1 {
			return status.Error(codes.InvalidArgument, "pro plan must remain single-operator scoped")
		}
		if parseWrite.SupportsTeamWorkspace {
			return status.Error(codes.InvalidArgument, "pro plan cannot enable team workspace collaboration")
		}
		if parseWrite.SupportsSSO {
			return status.Error(codes.InvalidArgument, "pro plan cannot enable enterprise SSO path")
		}
	case "team":
		if parseIncludedSeats < 2 || parseMaxSeats < 2 {
			return status.Error(codes.InvalidArgument, "team plan must allow shared workspace seats")
		}
		if !parseWrite.SupportsTeamWorkspace {
			return status.Error(codes.InvalidArgument, "team plan must enable workspace collaboration")
		}
	case "enterprise":
		if parseIncludedSeats < 10 || parseMaxSeats < 10 {
			return status.Error(codes.InvalidArgument, "enterprise plan must remain contract-sized")
		}
		if !parseWrite.SupportsTeamWorkspace {
			return status.Error(codes.InvalidArgument, "enterprise plan must enable workspace collaboration")
		}
		if !parseWrite.SupportsSSO {
			return status.Error(codes.InvalidArgument, "enterprise plan must remain SSO-capable")
		}
	default:
		return nil
	}
	return nil
}

// parseValidateUsageBasedBillingOverrideWrite validates one admin override write against usage-based billing assumptions.
func parseValidateUsageBasedBillingOverrideWrite(parseOverrideKey string, parseOverrideValue string) error {
	parseOverrideKey = strings.TrimSpace(strings.ToLower(parseOverrideKey))
	parseOverrideValue = strings.TrimSpace(strings.ToLower(parseOverrideValue))
	if parseOverrideKey == "" {
		return status.Error(codes.InvalidArgument, "override key is required")
	}
	if strings.Contains(parseOverrideKey, "flat_rate") {
		return status.Error(codes.InvalidArgument, "flat-rate billing overrides are not supported")
	}
	if parseOverrideKey == billingEntitlementUsageMonthlyTokenLimit && parseIsUsageBudgetUnlimited(parseOverrideValue) {
		return status.Error(codes.InvalidArgument, "monthly token limit overrides cannot use unlimited")
	}
	switch parseOverrideKey {
	case "billing.platform_fee_cents", "billing.usage_cost_cents", "billing.service_premium_cents", "billing.total_cents":
		return status.Error(codes.InvalidArgument, "override key cannot mutate usage-based billing formula totals directly")
	default:
		return nil
	}
}

// parseValidatePlanBoundaryEntitlementWrite validates entitlement writes against Pro/Team/Enterprise boundaries.
func parseValidatePlanBoundaryEntitlementWrite(parseWrite parseSuperuserBillingPlanEntitlementWrite) error {
	parsePlanCode := parseNormalizeSuperuserBillingPlanCode(parseWrite.PlanCode)
	parseEntitlementKey := strings.TrimSpace(strings.ToLower(parseWrite.EntitlementKey))
	parseEntitlementValue := strings.TrimSpace(strings.ToLower(parseWrite.EntitlementValue))
	switch parseEntitlementKey {
	case "workspace.multi_user.enabled":
		switch parsePlanCode {
		case "pro":
			if isParseEntitlementValueEnabled(parseEntitlementValue) {
				return status.Error(codes.InvalidArgument, "pro plan cannot enable multi-user workspace entitlement")
			}
		case "team", "enterprise":
			if !isParseEntitlementValueEnabled(parseEntitlementValue) {
				return status.Error(codes.InvalidArgument, "team and enterprise plans must keep multi-user workspace entitlement enabled")
			}
		}
	case "sso.enabled":
		switch parsePlanCode {
		case "enterprise":
			if !isParseEntitlementValueEnabled(parseEntitlementValue) {
				return status.Error(codes.InvalidArgument, "enterprise plan must keep SSO entitlement enabled")
			}
		case "pro", "team":
			if isParseEntitlementValueEnabled(parseEntitlementValue) {
				return status.Error(codes.InvalidArgument, "pro and team plans cannot enable enterprise SSO entitlement")
			}
		}
	}
	return nil
}

// parseResolveUsageBasedBillingLineType resolves one invoice line type into the usage-based billing line taxonomy.
func parseResolveUsageBasedBillingLineType(parseLineType string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseLineType)) {
	case "platform_fee", "subscription", "base_fee", "monthly_base":
		return parseBillingLineTypePlatformFee, nil
	case "usage_cost", "usage", "model_usage":
		return parseBillingLineTypeUsageCost, nil
	case "service_premium", "premium", "markup", "usage_premium":
		return parseBillingLineTypeServicePremium, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported billing line type %q", strings.TrimSpace(parseLineType))
	}
}

// parseValidateUsageBasedBillingLineItemWrite validates one invoice line item against usage-based billing line-item rules.
func parseValidateUsageBasedBillingLineItemWrite(parseWrite parseBillingInvoiceLineItemWrite) (string, error) {
	parseResolvedLineType, parseErr := parseResolveUsageBasedBillingLineType(parseWrite.LineType)
	if parseErr != nil {
		return "", parseErr
	}
	if parseWrite.UnitAmountCents < 0 || parseWrite.AmountCents < 0 {
		return "", status.Error(codes.InvalidArgument, "invoice line amounts must be non-negative")
	}
	parseQuantity := parseWrite.Quantity
	if parseQuantity <= 0 {
		parseQuantity = 1
	}
	parseExpectedAmountCents := parseQuantity * parseWrite.UnitAmountCents
	if parseWrite.AmountCents != parseExpectedAmountCents {
		return "", status.Errorf(codes.InvalidArgument, "invoice line amount must equal quantity*unit_amount (%d)", parseExpectedAmountCents)
	}
	switch parseResolvedLineType {
	case parseBillingLineTypePlatformFee, parseBillingLineTypeUsageCost, parseBillingLineTypeServicePremium:
		return parseResolvedLineType, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported billing line type %q", parseResolvedLineType)
	}
}

// parseFormatUsagePremiumBasisPoints converts one usage premium percent into basis points with stable rounding.
func parseFormatUsagePremiumBasisPoints(parseUsagePremiumPercent float64) int64 {
	if parseUsagePremiumPercent < 0 {
		return 0
	}
	return int64(math.Round(parseUsagePremiumPercent * 100))
}

// parseBuildUsageBasedBillingSummary formats one human-readable formula summary for diagnostics.
func parseBuildUsageBasedBillingSummary(parsePreview parseUsageBasedBillingPreview) string {
	return fmt.Sprintf(
		"platform_fee_cents=%d usage_cost_cents=%d service_premium_cents=%d total_cents=%d",
		parsePreview.platformFeeCents,
		parsePreview.usageCostCents,
		parsePreview.servicePremiumCents,
		parsePreview.totalCents,
	)
}
