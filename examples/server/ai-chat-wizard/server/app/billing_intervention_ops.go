package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseBillingInterventionSnapshot struct {
	parseCustomerRow      parseBillingCustomerRow
	parseInvoiceRows      []parseBillingInvoiceRow
	parseOverrideRows     []parseBillingAccessOverrideRow
	parseEventRows        []parseBillingEventRow
	parseDunningEventRows []parseBillingEventRow
	parseQuotaPolicyRows  []parseBillingQuotaPolicyRow
	parseUpgradeRows      []parseBillingUpgradeTriggerRow
}

// parseRequireBillingInterventionCustomer resolves one scoped admin customer target for billing interventions.
func (parseS *chatServer) parseRequireBillingInterventionCustomer(parseCtx context.Context, parseTargetUserID int64, parseSliceKey string) (parseAdminAccessScope, parseBillingCustomerRow, error) {
	if parseTargetUserID <= 0 {
		return parseAdminAccessScope{}, parseBillingCustomerRow{}, status.Error(codes.InvalidArgument, "target user id is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, parseSliceKey)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseBillingCustomerRow{}, parseErr
	}
	if !parseScope.isPlatformScope {
		if _, hasParseUser := parseScope.userIDs[parseTargetUserID]; !hasParseUser {
			return parseAdminAccessScope{}, parseBillingCustomerRow{}, status.Error(codes.PermissionDenied, "target user outside workspace-admin scope")
		}
	}
	if parseS == nil || parseS.store == nil {
		return parseAdminAccessScope{}, parseBillingCustomerRow{}, status.Error(codes.Unavailable, "store unavailable")
	}
	parseCustomerRow, isParseCustomerFound, parseErr := parseS.store.parseGetBillingCustomerByUser(parseTargetUserID)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseBillingCustomerRow{}, status.Errorf(codes.Internal, "billing customer lookup failed: %v", parseErr)
	}
	if !isParseCustomerFound {
		return parseAdminAccessScope{}, parseBillingCustomerRow{}, status.Error(codes.NotFound, "billing customer not found for target user")
	}
	return parseScope, parseCustomerRow, nil
}

// parseGetBillingInterventionSnapshotByAdmin returns typed billing intervention context for one scoped customer target.
func (parseS *chatServer) parseGetBillingInterventionSnapshotByAdmin(parseCtx context.Context, parseTargetUserID int64, parseLimit int32) (parseBillingInterventionSnapshot, error) {
	parseScope, parseCustomerRow, parseErr := parseS.parseRequireBillingInterventionCustomer(parseCtx, parseTargetUserID, "dashboard.billing.snapshot")
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, parseErr
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, status.Errorf(codes.Internal, "list billing invoices: %v", parseErr)
	}
	parseOverrideRows, parseErr := parseS.store.parseListBillingAccessOverridesByCustomer(parseCustomerRow.ID)
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, status.Errorf(codes.Internal, "list billing access overrides: %v", parseErr)
	}
	parseOverrideRows = parseLimitBillingAccessOverrideRows(parseOverrideRows, parseLimit)
	parseEventRows, parseErr := parseS.store.parseListBillingEventsByCustomer(parseCustomerRow.ID, int64(parseLimit))
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, status.Errorf(codes.Internal, "list billing events: %v", parseErr)
	}
	parseDunningRows := parseFilterBillingEventRowsByDunning(parseEventRows, parseLimit)

	parsePlanCode := ""
	parseSubscriptionRows, parseErr := parseS.store.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, status.Errorf(codes.Internal, "list billing subscriptions: %v", parseErr)
	}
	if len(parseSubscriptionRows) > 0 {
		parsePlanCode = strings.TrimSpace(parseSubscriptionRows[0].PlanCode)
	}
	parseQuotaRows, parseErr := parseS.store.parseListBillingQuotaPolicies(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, status.Errorf(codes.Internal, "list billing quota policies: %v", parseErr)
	}
	parseQuotaRows = parseFilterBillingQuotaRowsByPlan(parseQuotaRows, parsePlanCode, parseLimit)
	parseUpgradeRows, parseErr := parseS.store.parseListBillingUpgradeTriggers(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseBillingInterventionSnapshot{}, status.Errorf(codes.Internal, "list billing upgrade triggers: %v", parseErr)
	}
	parseUpgradeRows = parseFilterBillingUpgradeRowsByPlan(parseUpgradeRows, parsePlanCode, parseLimit)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.billing.intervention.snapshot",
		"billing_customer",
		fmt.Sprintf("%d", parseCustomerRow.ID),
		"Billing intervention snapshot viewed",
		"{}",
		0,
	)
	return parseBillingInterventionSnapshot{
		parseCustomerRow:      parseCustomerRow,
		parseInvoiceRows:      parseInvoiceRows,
		parseOverrideRows:     parseOverrideRows,
		parseEventRows:        parseEventRows,
		parseDunningEventRows: parseDunningRows,
		parseQuotaPolicyRows:  parseQuotaRows,
		parseUpgradeRows:      parseUpgradeRows,
	}, nil
}

// parseListBillingDunningEventsByAdmin lists typed dunning events for one scoped customer target.
func (parseS *chatServer) parseListBillingDunningEventsByAdmin(parseCtx context.Context, parseTargetUserID int64, parseLimit int32) ([]parseBillingEventRow, error) {
	parseSnapshot, parseErr := parseS.parseGetBillingInterventionSnapshotByAdmin(parseCtx, parseTargetUserID, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	return parseSnapshot.parseDunningEventRows, nil
}

// parseApplyBillingAccessOverrideByAdmin applies one typed billing-access override for one scoped customer target.
func (parseS *chatServer) parseApplyBillingAccessOverrideByAdmin(parseCtx context.Context, parseTargetUserID int64, parseOverrideKey, parseOverrideValue, parseReason string, isParseEnabled bool, parseStartsAt, parseEndsAt string) error {
	parseScope, parseCustomerRow, parseErr := parseS.parseRequireBillingInterventionCustomer(parseCtx, parseTargetUserID, "dashboard.billing.override")
	if parseErr != nil {
		return parseErr
	}
	parseOverrideKey = strings.TrimSpace(parseOverrideKey)
	parseReason = strings.TrimSpace(parseReason)
	if parseOverrideKey == "" {
		return status.Error(codes.InvalidArgument, "override key is required")
	}
	if parseReason == "" {
		return status.Error(codes.InvalidArgument, "override reason is required")
	}
	if parseErr = parseS.store.parseUpsertBillingAccessOverride(parseBillingAccessOverrideWrite{
		CustomerID:    parseCustomerRow.ID,
		OverrideKey:   parseOverrideKey,
		OverrideValue: strings.TrimSpace(parseOverrideValue),
		Reason:        parseReason,
		IsEnabled:     isParseEnabled,
		StartsAt:      strings.TrimSpace(parseStartsAt),
		EndsAt:        strings.TrimSpace(parseEndsAt),
		ActorUserID:   parseScope.adminUserID,
	}); parseErr != nil {
		return status.Errorf(codes.Internal, "apply billing access override: %v", parseErr)
	}
	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "list billing invoices for override event: %v", parseErr)
	}
	if len(parseInvoiceRows) > 0 {
		if _, parseErr = parseS.store.parseCreateBillingEvent(parseBillingEventWrite{
			CustomerID:       parseCustomerRow.ID,
			SubscriptionID:   parseInvoiceRows[0].SubscriptionID,
			InvoiceID:        parseInvoiceRows[0].ID,
			EventType:        "admin.billing.override.updated",
			EventSource:      "admin",
			EventSummary:     fmt.Sprintf("override %s updated: %s", parseOverrideKey, parseReason),
			EventPayloadJSON: "{}",
			ActorUserID:      parseScope.adminUserID,
		}); parseErr != nil {
			return status.Errorf(codes.Internal, "record billing override event: %v", parseErr)
		}
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.billing.intervention.override",
		"billing_customer",
		fmt.Sprintf("%d", parseCustomerRow.ID),
		"Billing access override updated",
		"{}",
		0,
	)
	return nil
}

// parseResolveBillingFailedPaymentByAdmin marks one scoped invoice as resolved and records one dunning-resolution event.
func (parseS *chatServer) parseResolveBillingFailedPaymentByAdmin(parseCtx context.Context, parseTargetUserID int64, parseInvoiceID int64, parseResolutionSummary string) error {
	parseScope, parseCustomerRow, parseErr := parseS.parseRequireBillingInterventionCustomer(parseCtx, parseTargetUserID, "dashboard.billing.failed_payment.resolve")
	if parseErr != nil {
		return parseErr
	}
	parseResolutionSummary = strings.TrimSpace(parseResolutionSummary)
	if parseInvoiceID <= 0 {
		return status.Error(codes.InvalidArgument, "invoice id is required")
	}
	if parseResolutionSummary == "" {
		return status.Error(codes.InvalidArgument, "resolution summary is required")
	}
	parseInvoiceRows, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, parseAdminScopedScanLimit)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "billing invoice lookup failed: %v", parseErr)
	}
	parseInvoiceRow, isParseInvoiceFound := parseFindBillingInvoiceRowByID(parseInvoiceRows, parseInvoiceID)
	if !isParseInvoiceFound {
		return status.Error(codes.NotFound, "billing invoice not found")
	}
	parsePaidAt := time.Now().UTC().Format(time.RFC3339)
	parseAmountPaidCents := max(parseInvoiceRow.AmountPaidCents, parseInvoiceRow.TotalCents)
	_, parseErr = parseS.store.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:           parseInvoiceRow.CustomerID,
		SubscriptionID:       parseInvoiceRow.SubscriptionID,
		ProviderID:           parseInvoiceRow.ProviderID,
		ProviderInvoiceID:    parseInvoiceRow.ProviderInvoiceID,
		Status:               "paid",
		Currency:             parseInvoiceRow.Currency,
		SubtotalCents:        parseInvoiceRow.SubtotalCents,
		TaxCents:             parseInvoiceRow.TaxCents,
		DiscountCents:        parseInvoiceRow.DiscountCents,
		TotalCents:           parseInvoiceRow.TotalCents,
		AmountDueCents:       0,
		AmountPaidCents:      parseAmountPaidCents,
		PeriodStart:          parseInvoiceRow.PeriodStart,
		PeriodEnd:            parseInvoiceRow.PeriodEnd,
		DueAt:                parseInvoiceRow.DueAt,
		PaidAt:               parsePaidAt,
		HostedInvoiceURL:     parseInvoiceRow.HostedInvoiceURL,
		ExternalMetadataJSON: parseInvoiceRow.ExternalMetadataJSON,
	})
	if parseErr != nil {
		return status.Errorf(codes.Internal, "resolve failed payment invoice update: %v", parseErr)
	}
	if _, parseErr = parseS.store.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:       parseCustomerRow.ID,
		SubscriptionID:   parseInvoiceRow.SubscriptionID,
		InvoiceID:        parseInvoiceRow.ID,
		EventType:        "dunning.resolved",
		EventSource:      "admin",
		EventSummary:     parseResolutionSummary,
		EventPayloadJSON: "{}",
		ActorUserID:      parseScope.adminUserID,
	}); parseErr != nil {
		return status.Errorf(codes.Internal, "record dunning resolution event: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.billing.intervention.failed_payment_resolved",
		"billing_invoice",
		fmt.Sprintf("%d", parseInvoiceID),
		"Failed payment resolved",
		"{}",
		0,
	)
	return nil
}

// parseFindBillingInvoiceRowByID resolves one invoice row by id.
func parseFindBillingInvoiceRowByID(parseRows []parseBillingInvoiceRow, parseInvoiceID int64) (parseBillingInvoiceRow, bool) {
	for _, parseRow := range parseRows {
		if parseRow.ID != parseInvoiceID {
			continue
		}
		return parseRow, true
	}
	return parseBillingInvoiceRow{}, false
}

// parseLimitBillingAccessOverrideRows truncates billing access overrides to one limit.
func parseLimitBillingAccessOverrideRows(parseRows []parseBillingAccessOverrideRow, parseLimit int32) []parseBillingAccessOverrideRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}

// parseFilterBillingEventRowsByDunning filters billing events to dunning/failure events and applies one limit.
func parseFilterBillingEventRowsByDunning(parseRows []parseBillingEventRow, parseLimit int32) []parseBillingEventRow {
	parseFilteredRows := make([]parseBillingEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasBillingDunningEvent(parseRow.EventType) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
		if parseLimit > 0 && len(parseFilteredRows) >= int(parseLimit) {
			break
		}
	}
	return parseFilteredRows
}

// parseFilterBillingQuotaRowsByPlan filters quota policy rows to one plan and applies one limit.
func parseFilterBillingQuotaRowsByPlan(parseRows []parseBillingQuotaPolicyRow, parsePlanCode string, parseLimit int32) []parseBillingQuotaPolicyRow {
	parsePlanCode = strings.TrimSpace(parsePlanCode)
	parseFilteredRows := make([]parseBillingQuotaPolicyRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parsePlanCode != "" && !strings.EqualFold(strings.TrimSpace(parseRow.PlanCode), parsePlanCode) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
		if parseLimit > 0 && len(parseFilteredRows) >= int(parseLimit) {
			break
		}
	}
	return parseFilteredRows
}

// parseFilterBillingUpgradeRowsByPlan filters upgrade-trigger rows to one plan and applies one limit.
func parseFilterBillingUpgradeRowsByPlan(parseRows []parseBillingUpgradeTriggerRow, parsePlanCode string, parseLimit int32) []parseBillingUpgradeTriggerRow {
	parsePlanCode = strings.TrimSpace(parsePlanCode)
	parseFilteredRows := make([]parseBillingUpgradeTriggerRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parsePlanCode != "" && !strings.EqualFold(strings.TrimSpace(parseRow.PlanCode), parsePlanCode) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
		if parseLimit > 0 && len(parseFilteredRows) >= int(parseLimit) {
			break
		}
	}
	return parseFilteredRows
}
