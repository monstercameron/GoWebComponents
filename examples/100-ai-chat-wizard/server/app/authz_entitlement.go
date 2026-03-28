package app

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const billingEntitlementChatSendEnabled = "chat.send.enabled"
const billingEntitlementUsageMonthlyTokenLimit = "usage.monthly_token_limit"
const billingEntitlementUsageSendsPerMinute = "usage.sends_per_minute"
const billingEntitlementUsageConcurrentSends = "usage.concurrent_sends"

const parseUsageBudgetDefaultSendsPerMinute int64 = 0
const parseUsageBudgetDefaultConcurrentSends int64 = 0

// parseBuildSendAccessDeniedStatus maps one send-gate error into one decision-shaped status with plan context.
func parseBuildSendAccessDeniedStatus(parseErr error, parsePlanCode string) error {
	if parseErr == nil {
		return nil
	}
	parseDecision, parseAction, parseReason := parseResolveSendAccessDecision(parseErr)
	parseDetail := parseResolveSendAccessDetail(parseErr)
	parsePlan := parseNormalizeSendAccessPlan(parsePlanCode)
	parseMessage := fmt.Sprintf(
		`send_access decision=%s plan=%s action=%s first_paid_action=chat.send reason=%s detail=%q`,
		parseDecision,
		parsePlan,
		parseAction,
		parseReason,
		parseDetail,
	)
	if parseDecision == "soft_upgrade" {
		return status.Error(codes.FailedPrecondition, parseMessage)
	}
	parseCode := status.Code(parseErr)
	if parseCode == codes.OK {
		parseCode = codes.PermissionDenied
	}
	return status.Error(parseCode, parseMessage)
}

// parseResolveSendAccessDecision classifies one send-gate failure into decision, action, and reason keys.
func parseResolveSendAccessDecision(parseErr error) (string, string, string) {
	parseMessageLower := strings.ToLower(parseResolveSendAccessDetail(parseErr))
	switch status.Code(parseErr) {
	case codes.PermissionDenied:
		if strings.Contains(parseMessageLower, "subscription entitlement missing") {
			return "soft_upgrade", "upgrade", "entitlement_missing"
		}
		return "soft_upgrade", "upgrade", "entitlement_denied"
	case codes.ResourceExhausted:
		switch {
		case strings.Contains(parseMessageLower, "monthly token quota exceeded"):
			return "soft_upgrade", "upgrade", "usage_monthly_quota_exceeded"
		case strings.Contains(parseMessageLower, "send rate limit exceeded"):
			return "hard_block", "retry_later", "usage_rate_limited"
		case strings.Contains(parseMessageLower, "concurrent send limit exceeded"):
			return "hard_block", "retry_later", "usage_concurrency_limited"
		default:
			return "hard_block", "retry_later", "usage_exhausted"
		}
	case codes.Unauthenticated:
		return "hard_block", "sign_in", "auth_required"
	case codes.FailedPrecondition:
		if strings.Contains(parseMessageLower, "subscription entitlement") {
			return "soft_upgrade", "upgrade", "entitlement_precondition"
		}
		return "hard_block", "resolve_precondition", "send_precondition"
	case codes.Internal, codes.Unavailable, codes.DeadlineExceeded:
		return "hard_block", "retry_later", "backend_unavailable"
	default:
		return "hard_block", "contact_support", "send_denied"
	}
}

// parseResolveSendAccessDetail extracts one concise detail string from one status error.
func parseResolveSendAccessDetail(parseErr error) string {
	if parseErr == nil {
		return ""
	}
	if parseStatus, parseOk := status.FromError(parseErr); parseOk {
		return strings.TrimSpace(parseStatus.Message())
	}
	return strings.TrimSpace(parseErr.Error())
}

// parseNormalizeSendAccessPlan normalizes one plan identifier for decision logging.
func parseNormalizeSendAccessPlan(parsePlanCode string) string {
	parsePlanCode = strings.TrimSpace(strings.ToLower(parsePlanCode))
	if parsePlanCode == "" {
		return "unknown"
	}
	return parsePlanCode
}

// parseRequireUserEntitlement enforces one billing entitlement gate for one authenticated user.
func (parseS *chatServer) parseRequireUserEntitlement(parseUserID int64, parseEntitlementKey string) error {
	parseEntitlementKey = strings.TrimSpace(parseEntitlementKey)
	if parseEntitlementKey == "" {
		return nil
	}
	if parseUserID <= 0 {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	if parseS == nil || parseS.store == nil {
		// TODO(authz): switch to fail-closed once all runtime paths provide billing state.
		return nil
	}
	parseAccessControl, parseErr := parseS.store.parseGetBillingAccessControlByUser(parseUserID, time.Now().UTC())
	if parseErr != nil {
		return status.Errorf(codes.Internal, "resolve billing access control: %v", parseErr)
	}
	isParseEnabled, hasParseEntry := parseResolveUserEntitlementEnabled(parseAccessControl, parseEntitlementKey)
	if !hasParseEntry {
		return status.Errorf(codes.PermissionDenied, "subscription entitlement missing: %s", parseEntitlementKey)
	}
	if !isParseEnabled {
		return status.Error(codes.PermissionDenied, "subscription entitlement denied")
	}
	return nil
}

// parseRequireUsageBudget validates monthly token, per-user rate, and concurrency quotas.
func (parseS *chatServer) parseRequireUsageBudget(parseUserID int64) (func(), error) {
	parseReleaseBudget := func() {}
	if parseUserID <= 0 {
		return parseReleaseBudget, status.Error(codes.Unauthenticated, "authentication required")
	}
	parseNow := time.Now().UTC()
	parseAccessControl := map[string]parseBillingAccessControl{}
	if parseS != nil && parseS.store != nil {
		parseResolvedAccessControl, parseErr := parseS.store.parseGetBillingAccessControlByUser(parseUserID, parseNow)
		if parseErr != nil {
			return parseReleaseBudget, status.Errorf(codes.Internal, "resolve usage budget access control: %v", parseErr)
		}
		parseAccessControl = parseResolvedAccessControl

		parseMonthlyTokenLimit, hasParseMonthlyTokenLimit, parseErr := parseResolveUsageBudgetLimit(parseAccessControl, billingEntitlementUsageMonthlyTokenLimit)
		if parseErr != nil {
			return parseReleaseBudget, status.Errorf(codes.FailedPrecondition, "usage budget config invalid: %v", parseErr)
		}
		if !hasParseMonthlyTokenLimit {
			return parseReleaseBudget, status.Errorf(codes.PermissionDenied, "subscription entitlement missing: %s", billingEntitlementUsageMonthlyTokenLimit)
		}
		if parseMonthlyTokenLimit > 0 {
			parseMonthStart := parseResolveUsageBudgetMonthStart(parseNow)
			parseUsedTokens, parseErr2 := parseS.store.parseSumUsageTokensSince(parseUserID, parseMonthStart)
			if parseErr2 != nil {
				return parseReleaseBudget, status.Errorf(codes.Internal, "resolve usage token totals: %v", parseErr2)
			}
			if parseUsedTokens >= parseMonthlyTokenLimit {
				return parseReleaseBudget, status.Errorf(
					codes.ResourceExhausted,
					"monthly token quota exceeded: used=%d limit=%d",
					parseUsedTokens,
					parseMonthlyTokenLimit,
				)
			}
		}
	}

	parseRateLimit, parseErr := parseResolveUsageBudgetOptionalLimit(parseAccessControl, billingEntitlementUsageSendsPerMinute, parseUsageBudgetDefaultSendsPerMinute)
	if parseErr != nil {
		return parseReleaseBudget, status.Errorf(codes.FailedPrecondition, "usage rate config invalid: %v", parseErr)
	}
	parseConcurrencyLimit, parseErr := parseResolveUsageBudgetOptionalLimit(parseAccessControl, billingEntitlementUsageConcurrentSends, parseUsageBudgetDefaultConcurrentSends)
	if parseErr != nil {
		return parseReleaseBudget, status.Errorf(codes.FailedPrecondition, "usage concurrency config invalid: %v", parseErr)
	}
	parseReleaseBudget, parseErr = parseS.parseAcquireUsageBudgetLease(parseUserID, parseNow, parseRateLimit, parseConcurrencyLimit)
	if parseErr != nil {
		return parseReleaseBudget, parseErr
	}
	return parseReleaseBudget, nil
}

// parseAcquireUsageBudgetLease acquires one user's in-flight usage budget lease and returns its release function.
func (parseS *chatServer) parseAcquireUsageBudgetLease(parseUserID int64, parseNow time.Time, parseRateLimit int64, parseConcurrencyLimit int64) (func(), error) {
	parseReleaseBudget := func() {}
	if parseS == nil {
		return parseReleaseBudget, nil
	}
	if parseRateLimit <= 0 && parseConcurrencyLimit <= 0 {
		return parseReleaseBudget, nil
	}

	parseS.trackUsageBudgetMutex.Lock()
	defer parseS.trackUsageBudgetMutex.Unlock()

	if parseS.trackUsageBudgetWindowByUser == nil {
		parseS.trackUsageBudgetWindowByUser = make(map[int64][]time.Time)
	}
	if parseS.trackUsageBudgetActiveByUser == nil {
		parseS.trackUsageBudgetActiveByUser = make(map[int64]int)
	}

	parseWindowCutoff := parseNow.Add(-time.Minute)
	parseUserWindow := parseS.trackUsageBudgetWindowByUser[parseUserID]
	parseWindowSize := 0
	for _, parseWindowStart := range parseUserWindow {
		if parseWindowStart.Before(parseWindowCutoff) {
			continue
		}
		parseUserWindow[parseWindowSize] = parseWindowStart
		parseWindowSize++
	}
	parseUserWindow = parseUserWindow[:parseWindowSize]
	parseS.trackUsageBudgetWindowByUser[parseUserID] = parseUserWindow
	if parseRateLimit > 0 && int64(len(parseUserWindow)) >= parseRateLimit {
		return parseReleaseBudget, status.Errorf(codes.ResourceExhausted, "send rate limit exceeded: limit_per_minute=%d", parseRateLimit)
	}

	parseUserActiveCount := parseS.trackUsageBudgetActiveByUser[parseUserID]
	if parseConcurrencyLimit > 0 && int64(parseUserActiveCount) >= parseConcurrencyLimit {
		return parseReleaseBudget, status.Errorf(codes.ResourceExhausted, "concurrent send limit exceeded: limit=%d", parseConcurrencyLimit)
	}

	parseS.trackUsageBudgetWindowByUser[parseUserID] = append(parseUserWindow, parseNow)
	parseS.trackUsageBudgetActiveByUser[parseUserID] = parseUserActiveCount + 1

	var parseReleaseOnce sync.Once
	return func() {
		parseReleaseOnce.Do(func() {
			parseS.trackUsageBudgetMutex.Lock()
			defer parseS.trackUsageBudgetMutex.Unlock()

			parseActiveCount := parseS.trackUsageBudgetActiveByUser[parseUserID]
			switch {
			case parseActiveCount <= 1:
				delete(parseS.trackUsageBudgetActiveByUser, parseUserID)
			default:
				parseS.trackUsageBudgetActiveByUser[parseUserID] = parseActiveCount - 1
			}
		})
	}, nil
}

// parseResolveUsageBudgetMonthStart resolves one UTC month-start timestamp for budget windows.
func parseResolveUsageBudgetMonthStart(parseNow time.Time) time.Time {
	parseNowUTC := parseNow.UTC()
	return time.Date(parseNowUTC.Year(), parseNowUTC.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// parseResolveUsageBudgetOptionalLimit resolves one optional quota limit with a default fallback.
func parseResolveUsageBudgetOptionalLimit(parseAccessControl map[string]parseBillingAccessControl, parseAccessKey string, parseDefaultLimit int64) (int64, error) {
	parseLimit, hasParseLimit, parseErr := parseResolveUsageBudgetLimit(parseAccessControl, parseAccessKey)
	if parseErr != nil {
		return 0, parseErr
	}
	if !hasParseLimit {
		return parseDefaultLimit, nil
	}
	return parseLimit, nil
}

// parseResolveUsageBudgetLimit resolves one parsed quota limit and whether the key exists.
func parseResolveUsageBudgetLimit(parseAccessControl map[string]parseBillingAccessControl, parseAccessKey string) (int64, bool, error) {
	parseAccessEntry, hasParseAccessEntry := parseAccessControl[parseAccessKey]
	if !hasParseAccessEntry {
		return 0, false, nil
	}
	parseLimit, parseErr := parseParseUsageBudgetLimit(parseAccessEntry.AccessValue, parseAccessKey)
	if parseErr != nil {
		return 0, true, parseErr
	}
	return parseLimit, true, nil
}

// parseParseUsageBudgetLimit parses one quota limit value from billing access control.
func parseParseUsageBudgetLimit(parseRawLimit string, parseAccessKey string) (int64, error) {
	parseLimitValue := strings.TrimSpace(parseRawLimit)
	if parseLimitValue == "" {
		return 0, fmt.Errorf("%s is empty", parseAccessKey)
	}
	if parseIsUsageBudgetUnlimited(parseLimitValue) {
		return 0, nil
	}
	parseLimit, parseErr := strconv.ParseInt(parseLimitValue, 10, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("%s must be an integer or unlimited: %w", parseAccessKey, parseErr)
	}
	if parseLimit < 0 {
		return 0, fmt.Errorf("%s must be non-negative", parseAccessKey)
	}
	return parseLimit, nil
}

// parseIsUsageBudgetUnlimited reports whether one limit value encodes unlimited quota.
func parseIsUsageBudgetUnlimited(parseRawLimit string) bool {
	switch strings.ToLower(strings.TrimSpace(parseRawLimit)) {
	case "unlimited", "inf", "infinite", "none", "off", "disabled":
		return true
	default:
		return false
	}
}

// parseResolveUserEntitlementEnabled resolves one entitlement key into enabled state and existence flags.
func parseResolveUserEntitlementEnabled(parseAccessControl map[string]parseBillingAccessControl, parseEntitlementKey string) (bool, bool) {
	parseControl, hasParseControl := parseAccessControl[parseEntitlementKey]
	if !hasParseControl {
		return false, false
	}
	return isParseEntitlementValueEnabled(parseControl.AccessValue), true
}

// isParseEntitlementValueEnabled normalizes one entitlement value into one enabled boolean.
func isParseEntitlementValueEnabled(parseRawValue string) bool {
	switch strings.ToLower(strings.TrimSpace(parseRawValue)) {
	case "1", "true", "yes", "on", "allow", "enabled":
		return true
	default:
		return false
	}
}
