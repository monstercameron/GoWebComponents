package app

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var errStoreBillingCustomerMissing = errors.New("store billing customer missing")
var errStoreBillingInvoiceMissing = errors.New("store billing invoice missing")
var errStoreBillingScopeMissing = errors.New("store billing scope missing")

type parseBillingCustomerWrite struct {
	UserID               int64
	ProviderID           string
	ProviderCustomerID   string
	BillingEmail         string
	BillingName          string
	BillingCountry       string
	BillingRegion        string
	DefaultCurrency      string
	TaxExemptStatus      string
	ExternalMetadataJSON string
}

type parseBillingCustomerRow struct {
	ID                   int64
	UserID               int64
	ProviderID           string
	ProviderCustomerID   string
	BillingEmail         string
	BillingName          string
	BillingCountry       string
	BillingRegion        string
	DefaultCurrency      string
	TaxExemptStatus      string
	ExternalMetadataJSON string
	CreatedAt            string
	UpdatedAt            string
}

type parseBillingSubscriptionWrite struct {
	CustomerID             int64
	ProviderID             string
	ProviderSubscriptionID string
	PlanCode               string
	PriceCode              string
	Status                 string
	BillingInterval        string
	Quantity               int64
	CurrentPeriodStart     string
	CurrentPeriodEnd       string
	IsCancelAtPeriodEnd    bool
	CanceledAt             string
	TrialEndsAt            string
	AccessExpiresAt        string
}

type parseBillingSubscriptionRow struct {
	ID                     int64
	CustomerID             int64
	ProviderID             string
	ProviderSubscriptionID string
	PlanCode               string
	PriceCode              string
	Status                 string
	BillingInterval        string
	Quantity               int64
	CurrentPeriodStart     string
	CurrentPeriodEnd       string
	IsCancelAtPeriodEnd    bool
	CanceledAt             string
	TrialEndsAt            string
	AccessExpiresAt        string
	CreatedAt              string
	UpdatedAt              string
}

type parseBillingInvoiceWrite struct {
	CustomerID           int64
	SubscriptionID       int64
	ProviderID           string
	ProviderInvoiceID    string
	Status               string
	Currency             string
	SubtotalCents        int64
	TaxCents             int64
	DiscountCents        int64
	TotalCents           int64
	AmountDueCents       int64
	AmountPaidCents      int64
	PeriodStart          string
	PeriodEnd            string
	DueAt                string
	PaidAt               string
	HostedInvoiceURL     string
	ExternalMetadataJSON string
}

type parseBillingInvoiceRow struct {
	ID                   int64
	CustomerID           int64
	SubscriptionID       int64
	ProviderID           string
	ProviderInvoiceID    string
	Status               string
	Currency             string
	SubtotalCents        int64
	TaxCents             int64
	DiscountCents        int64
	TotalCents           int64
	AmountDueCents       int64
	AmountPaidCents      int64
	PeriodStart          string
	PeriodEnd            string
	DueAt                string
	PaidAt               string
	HostedInvoiceURL     string
	ExternalMetadataJSON string
	CreatedAt            string
	UpdatedAt            string
}

type parseBillingInvoiceLineItemWrite struct {
	CustomerID      int64
	InvoiceID       int64
	UsageEventID    string
	LineType        string
	Description     string
	Quantity        int64
	UnitAmountCents int64
	AmountCents     int64
	Currency        string
	PeriodStart     string
	PeriodEnd       string
}

type parseBillingInvoiceLineItemRow struct {
	ID              int64
	InvoiceID       int64
	UsageEventID    string
	LineType        string
	Description     string
	Quantity        int64
	UnitAmountCents int64
	AmountCents     int64
	Currency        string
	PeriodStart     string
	PeriodEnd       string
	CreatedAt       string
}

type parseBillingAccessOverrideWrite struct {
	CustomerID    int64
	OverrideKey   string
	OverrideValue string
	Reason        string
	IsEnabled     bool
	StartsAt      string
	EndsAt        string
	ActorUserID   int64
}

type parseBillingAccessOverrideRow struct {
	ID            int64
	CustomerID    int64
	OverrideKey   string
	OverrideValue string
	Reason        string
	IsEnabled     bool
	StartsAt      string
	EndsAt        string
	ActorUserID   int64
	CreatedAt     string
	UpdatedAt     string
}

type parseBillingEventWrite struct {
	CustomerID       int64
	SubscriptionID   int64
	InvoiceID        int64
	EventType        string
	EventSource      string
	EventSummary     string
	EventPayloadJSON string
	ActorUserID      int64
}

type parseBillingEventRow struct {
	ID               int64
	CustomerID       int64
	SubscriptionID   int64
	InvoiceID        int64
	EventType        string
	EventSource      string
	EventSummary     string
	EventPayloadJSON string
	ActorUserID      int64
	CreatedAt        string
}

type parseBillingPlanEntitlementRow struct {
	EntitlementKey   string
	EntitlementValue string
	UpdatedAt        string
}

type parseBillingEffectiveAccessRow struct {
	AccessKey       string
	AccessValue     string
	SourceType      string
	SourceUpdatedAt string
	Reason          string
}

type parseBillingEffectiveModelAccessRow struct {
	ModelID   string
	IsDefault bool
	PlanCode  string
}

type parseBillingAccessControl struct {
	AccessValue     string
	SourceType      string
	SourceUpdatedAt string
	Reason          string
}

// parseUpsertBillingCustomer persists one customer identity row for one user.
func (parseS *Store) parseUpsertBillingCustomer(parseWrite parseBillingCustomerWrite) (parseBillingCustomerRow, error) {
	if parseWrite.UserID <= 0 {
		return parseBillingCustomerRow{}, errors.New("upsert billing customer: user id is required")
	}
	if strings.TrimSpace(parseWrite.ProviderID) == "" || strings.TrimSpace(parseWrite.ProviderCustomerID) == "" {
		return parseBillingCustomerRow{}, errors.New("upsert billing customer: provider id and provider customer id are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingCustomer,
		parseWrite.UserID,
		strings.TrimSpace(parseWrite.ProviderID),
		strings.TrimSpace(parseWrite.ProviderCustomerID),
		strings.TrimSpace(parseWrite.BillingEmail),
		strings.TrimSpace(parseWrite.BillingName),
		strings.TrimSpace(parseWrite.BillingCountry),
		strings.TrimSpace(parseWrite.BillingRegion),
		parseNormalizeBillingCurrency(parseWrite.DefaultCurrency),
		parseNormalizeBillingTaxExemptStatus(parseWrite.TaxExemptStatus),
		parseNormalizeBillingJSON(parseWrite.ExternalMetadataJSON),
		parseNow,
		parseNow,
		parseWrite.UserID,
	)
	if parseErr != nil {
		return parseBillingCustomerRow{}, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return parseBillingCustomerRow{}, errStoreUserMissing
	}
	parseCustomer, hasParseCustomer, parseErr := parseS.parseGetBillingCustomerByUser(parseWrite.UserID)
	if parseErr != nil {
		return parseBillingCustomerRow{}, parseErr
	}
	if !hasParseCustomer {
		return parseBillingCustomerRow{}, errStoreBillingCustomerMissing
	}
	return parseCustomer, nil
}

// parseGetBillingCustomerByUser resolves one billing customer row by user id.
func (parseS *Store) parseGetBillingCustomerByUser(parseUserID int64) (parseBillingCustomerRow, bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getBillingCustomerByUser, parseUserID)
	var parseCustomer parseBillingCustomerRow
	if parseErr := parseRow.Scan(
		&parseCustomer.ID,
		&parseCustomer.UserID,
		&parseCustomer.ProviderID,
		&parseCustomer.ProviderCustomerID,
		&parseCustomer.BillingEmail,
		&parseCustomer.BillingName,
		&parseCustomer.BillingCountry,
		&parseCustomer.BillingRegion,
		&parseCustomer.DefaultCurrency,
		&parseCustomer.TaxExemptStatus,
		&parseCustomer.ExternalMetadataJSON,
		&parseCustomer.CreatedAt,
		&parseCustomer.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseBillingCustomerRow{}, false, nil
		}
		return parseBillingCustomerRow{}, false, parseErr
	}
	return parseCustomer, true, nil
}

// parseUpsertBillingSubscription persists one subscription row for one billing customer.
func (parseS *Store) parseUpsertBillingSubscription(parseWrite parseBillingSubscriptionWrite) (parseBillingSubscriptionRow, error) {
	if parseWrite.CustomerID <= 0 {
		return parseBillingSubscriptionRow{}, errors.New("upsert billing subscription: customer id is required")
	}
	if strings.TrimSpace(parseWrite.ProviderID) == "" || strings.TrimSpace(parseWrite.ProviderSubscriptionID) == "" {
		return parseBillingSubscriptionRow{}, errors.New("upsert billing subscription: provider id and provider subscription id are required")
	}
	if strings.TrimSpace(parseWrite.PlanCode) == "" {
		return parseBillingSubscriptionRow{}, errors.New("upsert billing subscription: plan code is required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseQuantity := parseWrite.Quantity
	if parseQuantity <= 0 {
		parseQuantity = 1
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingSubscription,
		parseWrite.CustomerID,
		strings.TrimSpace(parseWrite.ProviderID),
		strings.TrimSpace(parseWrite.ProviderSubscriptionID),
		strings.TrimSpace(parseWrite.PlanCode),
		strings.TrimSpace(parseWrite.PriceCode),
		parseNormalizeBillingSubscriptionStatus(parseWrite.Status),
		parseNormalizeBillingInterval(parseWrite.BillingInterval),
		parseQuantity,
		strings.TrimSpace(parseWrite.CurrentPeriodStart),
		strings.TrimSpace(parseWrite.CurrentPeriodEnd),
		parseBuildBillingFlagValue(parseWrite.IsCancelAtPeriodEnd),
		strings.TrimSpace(parseWrite.CanceledAt),
		strings.TrimSpace(parseWrite.TrialEndsAt),
		strings.TrimSpace(parseWrite.AccessExpiresAt),
		parseNow,
		parseNow,
		parseWrite.CustomerID,
		strings.TrimSpace(parseWrite.PlanCode),
	)
	if parseErr != nil {
		return parseBillingSubscriptionRow{}, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return parseBillingSubscriptionRow{}, errStoreBillingScopeMissing
	}
	parseSubscription, hasParseSubscription, parseErr := parseS.parseGetBillingSubscriptionByProvider(strings.TrimSpace(parseWrite.ProviderSubscriptionID))
	if parseErr != nil {
		return parseBillingSubscriptionRow{}, parseErr
	}
	if !hasParseSubscription {
		return parseBillingSubscriptionRow{}, errStoreBillingScopeMissing
	}
	return parseSubscription, nil
}

// parseGetBillingSubscriptionByProvider resolves one subscription row by provider subscription id.
func (parseS *Store) parseGetBillingSubscriptionByProvider(parseProviderSubscriptionID string) (parseBillingSubscriptionRow, bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getBillingSubscriptionByProvider, strings.TrimSpace(parseProviderSubscriptionID))
	var parseSubscription parseBillingSubscriptionRow
	var parseCancelAtPeriodEnd int64
	if parseErr := parseRow.Scan(
		&parseSubscription.ID,
		&parseSubscription.CustomerID,
		&parseSubscription.ProviderID,
		&parseSubscription.ProviderSubscriptionID,
		&parseSubscription.PlanCode,
		&parseSubscription.PriceCode,
		&parseSubscription.Status,
		&parseSubscription.BillingInterval,
		&parseSubscription.Quantity,
		&parseSubscription.CurrentPeriodStart,
		&parseSubscription.CurrentPeriodEnd,
		&parseCancelAtPeriodEnd,
		&parseSubscription.CanceledAt,
		&parseSubscription.TrialEndsAt,
		&parseSubscription.AccessExpiresAt,
		&parseSubscription.CreatedAt,
		&parseSubscription.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseBillingSubscriptionRow{}, false, nil
		}
		return parseBillingSubscriptionRow{}, false, parseErr
	}
	parseSubscription.IsCancelAtPeriodEnd = parseCancelAtPeriodEnd != 0
	return parseSubscription, true, nil
}

// parseListBillingSubscriptionsByCustomer lists subscriptions for one customer newest-first.
func (parseS *Store) parseListBillingSubscriptionsByCustomer(parseCustomerID, parseLimit int64) ([]parseBillingSubscriptionRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingSubscriptionsByCustomer, parseCustomerID, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseSubscriptions := make([]parseBillingSubscriptionRow, 0)
	for parseRows.Next() {
		var parseSubscription parseBillingSubscriptionRow
		var parseCancelAtPeriodEnd int64
		if parseErr2 := parseRows.Scan(
			&parseSubscription.ID,
			&parseSubscription.CustomerID,
			&parseSubscription.ProviderID,
			&parseSubscription.ProviderSubscriptionID,
			&parseSubscription.PlanCode,
			&parseSubscription.PriceCode,
			&parseSubscription.Status,
			&parseSubscription.BillingInterval,
			&parseSubscription.Quantity,
			&parseSubscription.CurrentPeriodStart,
			&parseSubscription.CurrentPeriodEnd,
			&parseCancelAtPeriodEnd,
			&parseSubscription.CanceledAt,
			&parseSubscription.TrialEndsAt,
			&parseSubscription.AccessExpiresAt,
			&parseSubscription.CreatedAt,
			&parseSubscription.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseSubscription.IsCancelAtPeriodEnd = parseCancelAtPeriodEnd != 0
		parseSubscriptions = append(parseSubscriptions, parseSubscription)
	}
	return parseSubscriptions, parseRows.Err()
}

// parseUpsertBillingInvoice persists one invoice row for one billing customer.
func (parseS *Store) parseUpsertBillingInvoice(parseWrite parseBillingInvoiceWrite) (parseBillingInvoiceRow, error) {
	if parseWrite.CustomerID <= 0 {
		return parseBillingInvoiceRow{}, errors.New("upsert billing invoice: customer id is required")
	}
	if strings.TrimSpace(parseWrite.ProviderID) == "" || strings.TrimSpace(parseWrite.ProviderInvoiceID) == "" {
		return parseBillingInvoiceRow{}, errors.New("upsert billing invoice: provider id and provider invoice id are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingInvoice,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		strings.TrimSpace(parseWrite.ProviderID),
		strings.TrimSpace(parseWrite.ProviderInvoiceID),
		parseNormalizeBillingInvoiceStatus(parseWrite.Status),
		parseNormalizeBillingCurrency(parseWrite.Currency),
		parseWrite.SubtotalCents,
		parseWrite.TaxCents,
		parseWrite.DiscountCents,
		parseWrite.TotalCents,
		parseWrite.AmountDueCents,
		parseWrite.AmountPaidCents,
		strings.TrimSpace(parseWrite.PeriodStart),
		strings.TrimSpace(parseWrite.PeriodEnd),
		strings.TrimSpace(parseWrite.DueAt),
		strings.TrimSpace(parseWrite.PaidAt),
		strings.TrimSpace(parseWrite.HostedInvoiceURL),
		parseNormalizeBillingJSON(parseWrite.ExternalMetadataJSON),
		parseNow,
		parseNow,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		parseWrite.SubscriptionID,
		parseWrite.CustomerID,
	)
	if parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return parseBillingInvoiceRow{}, errStoreBillingScopeMissing
	}
	parseInvoice, hasParseInvoice, parseErr := parseS.parseGetBillingInvoiceByProvider(strings.TrimSpace(parseWrite.ProviderInvoiceID))
	if parseErr != nil {
		return parseBillingInvoiceRow{}, parseErr
	}
	if !hasParseInvoice {
		return parseBillingInvoiceRow{}, errStoreBillingScopeMissing
	}
	return parseInvoice, nil
}

// parseGetBillingInvoiceByProvider resolves one invoice row by provider invoice id.
func (parseS *Store) parseGetBillingInvoiceByProvider(parseProviderInvoiceID string) (parseBillingInvoiceRow, bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getBillingInvoiceByProvider, strings.TrimSpace(parseProviderInvoiceID))
	var parseInvoice parseBillingInvoiceRow
	if parseErr := parseRow.Scan(
		&parseInvoice.ID,
		&parseInvoice.CustomerID,
		&parseInvoice.SubscriptionID,
		&parseInvoice.ProviderID,
		&parseInvoice.ProviderInvoiceID,
		&parseInvoice.Status,
		&parseInvoice.Currency,
		&parseInvoice.SubtotalCents,
		&parseInvoice.TaxCents,
		&parseInvoice.DiscountCents,
		&parseInvoice.TotalCents,
		&parseInvoice.AmountDueCents,
		&parseInvoice.AmountPaidCents,
		&parseInvoice.PeriodStart,
		&parseInvoice.PeriodEnd,
		&parseInvoice.DueAt,
		&parseInvoice.PaidAt,
		&parseInvoice.HostedInvoiceURL,
		&parseInvoice.ExternalMetadataJSON,
		&parseInvoice.CreatedAt,
		&parseInvoice.UpdatedAt,
	); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return parseBillingInvoiceRow{}, false, nil
		}
		return parseBillingInvoiceRow{}, false, parseErr
	}
	return parseInvoice, true, nil
}

// parseListBillingInvoicesByCustomer lists invoices for one customer newest-first.
func (parseS *Store) parseListBillingInvoicesByCustomer(parseCustomerID, parseLimit int64) ([]parseBillingInvoiceRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingInvoicesByCustomer, parseCustomerID, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseInvoices := make([]parseBillingInvoiceRow, 0)
	for parseRows.Next() {
		var parseInvoice parseBillingInvoiceRow
		if parseErr2 := parseRows.Scan(
			&parseInvoice.ID,
			&parseInvoice.CustomerID,
			&parseInvoice.SubscriptionID,
			&parseInvoice.ProviderID,
			&parseInvoice.ProviderInvoiceID,
			&parseInvoice.Status,
			&parseInvoice.Currency,
			&parseInvoice.SubtotalCents,
			&parseInvoice.TaxCents,
			&parseInvoice.DiscountCents,
			&parseInvoice.TotalCents,
			&parseInvoice.AmountDueCents,
			&parseInvoice.AmountPaidCents,
			&parseInvoice.PeriodStart,
			&parseInvoice.PeriodEnd,
			&parseInvoice.DueAt,
			&parseInvoice.PaidAt,
			&parseInvoice.HostedInvoiceURL,
			&parseInvoice.ExternalMetadataJSON,
			&parseInvoice.CreatedAt,
			&parseInvoice.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseInvoices = append(parseInvoices, parseInvoice)
	}
	return parseInvoices, parseRows.Err()
}

// parseCreateBillingInvoiceLineItem inserts one line item row under one customer-owned invoice.
func (parseS *Store) parseCreateBillingInvoiceLineItem(parseWrite parseBillingInvoiceLineItemWrite) (int64, error) {
	if parseWrite.CustomerID <= 0 || parseWrite.InvoiceID <= 0 {
		return 0, errors.New("create billing invoice line item: customer id and invoice id are required")
	}
	parseQuantity := parseWrite.Quantity
	if parseQuantity <= 0 {
		parseQuantity = 1
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createBillingInvoiceLineItem,
		parseWrite.InvoiceID,
		strings.TrimSpace(parseWrite.UsageEventID),
		parseNormalizeBillingLineType(parseWrite.LineType),
		strings.TrimSpace(parseWrite.Description),
		parseQuantity,
		parseWrite.UnitAmountCents,
		parseWrite.AmountCents,
		parseNormalizeBillingCurrency(parseWrite.Currency),
		strings.TrimSpace(parseWrite.PeriodStart),
		strings.TrimSpace(parseWrite.PeriodEnd),
		parseNow,
		parseWrite.InvoiceID,
		parseWrite.CustomerID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return 0, errStoreBillingInvoiceMissing
	}
	return parseResult.LastInsertId()
}

// parseListBillingInvoiceLineItems lists line items for one customer-owned invoice.
func (parseS *Store) parseListBillingInvoiceLineItems(parseInvoiceID, parseCustomerID int64) ([]parseBillingInvoiceLineItemRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingInvoiceLineItems, parseInvoiceID, parseCustomerID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseLineItems := make([]parseBillingInvoiceLineItemRow, 0)
	for parseRows.Next() {
		var parseLineItem parseBillingInvoiceLineItemRow
		if parseErr2 := parseRows.Scan(
			&parseLineItem.ID,
			&parseLineItem.InvoiceID,
			&parseLineItem.UsageEventID,
			&parseLineItem.LineType,
			&parseLineItem.Description,
			&parseLineItem.Quantity,
			&parseLineItem.UnitAmountCents,
			&parseLineItem.AmountCents,
			&parseLineItem.Currency,
			&parseLineItem.PeriodStart,
			&parseLineItem.PeriodEnd,
			&parseLineItem.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseLineItems = append(parseLineItems, parseLineItem)
	}
	return parseLineItems, parseRows.Err()
}

// parseUpsertBillingAccessOverride persists one operator override for one customer access key.
func (parseS *Store) parseUpsertBillingAccessOverride(parseWrite parseBillingAccessOverrideWrite) error {
	if parseWrite.CustomerID <= 0 || strings.TrimSpace(parseWrite.OverrideKey) == "" {
		return errors.New("upsert billing access override: customer id and override key are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.upsertBillingAccessOverride,
		parseWrite.CustomerID,
		strings.TrimSpace(parseWrite.OverrideKey),
		strings.TrimSpace(parseWrite.OverrideValue),
		strings.TrimSpace(parseWrite.Reason),
		parseBuildBillingFlagValue(parseWrite.IsEnabled),
		strings.TrimSpace(parseWrite.StartsAt),
		strings.TrimSpace(parseWrite.EndsAt),
		parseWrite.ActorUserID,
		parseNow,
		parseNow,
		parseWrite.CustomerID,
	)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreBillingCustomerMissing
	}
	return nil
}

// parseListBillingAccessOverridesByCustomer lists configured access overrides for one customer.
func (parseS *Store) parseListBillingAccessOverridesByCustomer(parseCustomerID int64) ([]parseBillingAccessOverrideRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingAccessOverridesByCustomer, parseCustomerID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseOverrides := make([]parseBillingAccessOverrideRow, 0)
	for parseRows.Next() {
		var parseOverride parseBillingAccessOverrideRow
		var parseIsEnabled int64
		if parseErr2 := parseRows.Scan(
			&parseOverride.ID,
			&parseOverride.CustomerID,
			&parseOverride.OverrideKey,
			&parseOverride.OverrideValue,
			&parseOverride.Reason,
			&parseIsEnabled,
			&parseOverride.StartsAt,
			&parseOverride.EndsAt,
			&parseOverride.ActorUserID,
			&parseOverride.CreatedAt,
			&parseOverride.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseOverride.IsEnabled = parseIsEnabled != 0
		parseOverrides = append(parseOverrides, parseOverride)
	}
	return parseOverrides, parseRows.Err()
}

// parseCreateBillingEvent appends one immutable billing event for customer audit visibility.
func (parseS *Store) parseCreateBillingEvent(parseWrite parseBillingEventWrite) (int64, error) {
	if parseWrite.CustomerID <= 0 || strings.TrimSpace(parseWrite.EventType) == "" {
		return 0, errors.New("create billing event: customer id and event type are required")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.createBillingEvent,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		parseWrite.InvoiceID,
		strings.TrimSpace(parseWrite.EventType),
		parseNormalizeBillingEventSource(parseWrite.EventSource),
		strings.TrimSpace(parseWrite.EventSummary),
		parseNormalizeBillingJSON(parseWrite.EventPayloadJSON),
		parseWrite.ActorUserID,
		parseNow,
		parseWrite.CustomerID,
		parseWrite.SubscriptionID,
		parseWrite.SubscriptionID,
		parseWrite.CustomerID,
		parseWrite.InvoiceID,
		parseWrite.InvoiceID,
		parseWrite.CustomerID,
	)
	if parseErr != nil {
		return 0, parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return 0, errStoreBillingScopeMissing
	}
	return parseResult.LastInsertId()
}

// parseListBillingEventsByCustomer lists immutable billing events for one customer.
func (parseS *Store) parseListBillingEventsByCustomer(parseCustomerID, parseLimit int64) ([]parseBillingEventRow, error) {
	if parseLimit <= 0 {
		parseLimit = 50
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingEventsByCustomer, parseCustomerID, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseEvents := make([]parseBillingEventRow, 0)
	for parseRows.Next() {
		var parseEvent parseBillingEventRow
		if parseErr2 := parseRows.Scan(
			&parseEvent.ID,
			&parseEvent.CustomerID,
			&parseEvent.SubscriptionID,
			&parseEvent.InvoiceID,
			&parseEvent.EventType,
			&parseEvent.EventSource,
			&parseEvent.EventSummary,
			&parseEvent.EventPayloadJSON,
			&parseEvent.ActorUserID,
			&parseEvent.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseEvents = append(parseEvents, parseEvent)
	}
	return parseEvents, parseRows.Err()
}

// parseListBillingPlanEntitlements lists static entitlements for one plan code.
func (parseS *Store) parseListBillingPlanEntitlements(parsePlanCode string) ([]parseBillingPlanEntitlementRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listBillingPlanEntitlements, strings.TrimSpace(parsePlanCode))
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseEntitlements := make([]parseBillingPlanEntitlementRow, 0)
	for parseRows.Next() {
		var parseEntitlement parseBillingPlanEntitlementRow
		if parseErr2 := parseRows.Scan(
			&parseEntitlement.EntitlementKey,
			&parseEntitlement.EntitlementValue,
			&parseEntitlement.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseEntitlements = append(parseEntitlements, parseEntitlement)
	}
	return parseEntitlements, parseRows.Err()
}

// parseListBillingEffectiveAccessByUser lists active plan entitlements and enabled overrides for one user.
func (parseS *Store) parseListBillingEffectiveAccessByUser(parseUserID int64, parseNow time.Time) ([]parseBillingEffectiveAccessRow, error) {
	parseRows, parseErr := parseS.db.Query(
		parseS.queries.listBillingEffectiveAccessByUser,
		parseUserID,
		parseNow.UTC().Format(time.RFC3339),
		parseNow.UTC().Format(time.RFC3339),
	)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseAccessRows := make([]parseBillingEffectiveAccessRow, 0)
	for parseRows.Next() {
		var parseRow parseBillingEffectiveAccessRow
		if parseErr2 := parseRows.Scan(
			&parseRow.AccessKey,
			&parseRow.AccessValue,
			&parseRow.SourceType,
			&parseRow.SourceUpdatedAt,
			&parseRow.Reason,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseAccessRows = append(parseAccessRows, parseRow)
	}
	return parseAccessRows, parseRows.Err()
}

// parseListBillingEffectiveModelAccessByUser lists plan-scoped model access rows for one user.
func (parseS *Store) parseListBillingEffectiveModelAccessByUser(parseUserID int64, parseNow time.Time) ([]parseBillingEffectiveModelAccessRow, error) {
	_ = parseNow
	parseRows, parseErr := parseS.db.Query(
		parseS.queries.listBillingEffectiveModelAccessByUser,
		parseUserID,
	)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseModelRows := make([]parseBillingEffectiveModelAccessRow, 0)
	for parseRows.Next() {
		var parseRow parseBillingEffectiveModelAccessRow
		var parseIsDefault int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ModelID,
			&parseIsDefault,
			&parseRow.PlanCode,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.IsDefault = parseIsDefault != 0
		parseModelRows = append(parseModelRows, parseRow)
	}
	return parseModelRows, parseRows.Err()
}

// parseGetBillingAccessControlByUser resolves one effective access map keyed by entitlement.
func (parseS *Store) parseGetBillingAccessControlByUser(parseUserID int64, parseNow time.Time) (map[string]parseBillingAccessControl, error) {
	parseAccessRows, parseErr := parseS.parseListBillingEffectiveAccessByUser(parseUserID, parseNow)
	if parseErr != nil {
		return nil, parseErr
	}
	parseControlMap := make(map[string]parseBillingAccessControl, len(parseAccessRows))
	for _, parseAccessRow := range parseAccessRows {
		parseCurrent, hasParseCurrent := parseControlMap[parseAccessRow.AccessKey]
		if !hasParseCurrent || parseAccessRow.SourceType == "override" {
			parseControlMap[parseAccessRow.AccessKey] = parseBillingAccessControl{
				AccessValue:     parseAccessRow.AccessValue,
				SourceType:      parseAccessRow.SourceType,
				SourceUpdatedAt: parseAccessRow.SourceUpdatedAt,
				Reason:          parseAccessRow.Reason,
			}
			continue
		}
		parseControlMap[parseAccessRow.AccessKey] = parseCurrent
	}
	return parseControlMap, nil
}

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
	parseNormalized := strings.ToLower(strings.TrimSpace(parseLineType))
	if parseNormalized == "" {
		return "usage"
	}
	return parseNormalized
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
