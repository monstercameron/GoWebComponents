package app

import "strings"

// parseBuildBillingFlagValue converts one bool into sqlite integer semantics.
func parseBuildBillingFlagValue(isEnabled bool) int64 {
	if isEnabled {
		return 1
	}
	return 0
}

// parseNormalizeBillingCurrency returns uppercase ISO-style currency or USD fallback.
func parseNormalizeBillingCurrency(parseCurrency string) string {
	parseNormalized := strings.ToUpper(strings.TrimSpace(parseCurrency))
	if parseNormalized == "" {
		return "USD"
	}
	return parseNormalized
}

// parseNormalizeBillingTaxExemptStatus normalizes one tax-exempt status value.
func parseNormalizeBillingTaxExemptStatus(parseStatus string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseStatus))
	if parseNormalized == "" {
		return "none"
	}
	return parseNormalized
}

// parseNormalizeBillingSubscriptionStatus normalizes one subscription status value.
func parseNormalizeBillingSubscriptionStatus(parseStatus string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseStatus))
	if parseNormalized == "" {
		return "active"
	}
	return parseNormalized
}

// parseNormalizeBillingInvoiceStatus normalizes one invoice status value.
func parseNormalizeBillingInvoiceStatus(parseStatus string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseStatus))
	if parseNormalized == "" {
		return "draft"
	}
	return parseNormalized
}

// parseNormalizeBillingInterval normalizes one billing interval value.
func parseNormalizeBillingInterval(parseInterval string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseInterval))
	if parseNormalized == "" {
		return "month"
	}
	return parseNormalized
}

// parseNormalizeBillingLineType normalizes one invoice line type value.
func parseNormalizeBillingLineType(parseLineType string) string {
	parseResolved, parseErr := parseResolveUsageBasedBillingLineType(parseLineType)
	if parseErr != nil {
		return parseBillingLineTypeUsageCost
	}
	return parseResolved
}

// parseNormalizeBillingEventSource normalizes one billing event source value.
func parseNormalizeBillingEventSource(parseEventSource string) string {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseEventSource))
	if parseNormalized == "" {
		return "system"
	}
	return parseNormalized
}

// parseNormalizeBillingJSON ensures optional JSON blobs always persist as object literals.
func parseNormalizeBillingJSON(parsePayload string) string {
	parseNormalized := strings.TrimSpace(parsePayload)
	if parseNormalized == "" {
		return "{}"
	}
	return parseNormalized
}
