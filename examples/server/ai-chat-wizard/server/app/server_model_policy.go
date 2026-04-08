package app

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type parseBillingModelPolicy struct {
	AllowedModelSet map[string]struct{}
	DefaultModelID  string
	PlanCode        string
}

// parseResolveBillingModelPolicy resolves effective model access and defaults for one user.
func (parseS *chatServer) parseResolveBillingModelPolicy(parseUserID int64, parseNow time.Time) (parseBillingModelPolicy, error) {
	if parseS == nil || parseUserID <= 0 {
		return parseBillingModelPolicy{}, nil
	}
	if parseS.isParseBypassBillingModelPolicy || parseS.store == nil {
		return parseBillingModelPolicy{}, nil
	}
	parseRows, parseErr := parseS.store.parseListBillingEffectiveModelAccessByUser(parseUserID, parseNow)
	if parseErr != nil {
		return parseBillingModelPolicy{}, parseErr
	}

	parsePolicy := parseBillingModelPolicy{
		AllowedModelSet: map[string]struct{}{},
	}
	for _, parseRow := range parseRows {
		parseModelID := parseNormalizeSelectedModelID(parseRow.ModelID)
		if parseModelID == "" {
			continue
		}
		parsePolicy.AllowedModelSet[parseModelID] = struct{}{}
		if parsePolicy.DefaultModelID == "" && parseRow.IsDefault {
			parsePolicy.DefaultModelID = parseModelID
		}
		if parsePolicy.PlanCode == "" {
			parsePolicy.PlanCode = strings.TrimSpace(parseRow.PlanCode)
		}
	}
	if parsePolicy.DefaultModelID == "" {
		parsePolicy.DefaultModelID = parseResolveFirstAllowedModelID(parsePolicy)
	}
	return parsePolicy, nil
}

// parseResolvePlanAwareModel chooses one model candidate and reports explicit-plan denial.
func parseResolvePlanAwareModel(parseRequestedModel string, parseFallbackModel string, parsePolicy parseBillingModelPolicy) (string, bool) {
	parseRequestedModel = parseNormalizeOptionalSelectedModelID(parseRequestedModel)
	parseSelectedModel := parseNormalizeSelectedModelID(parseFallbackModel)
	if parseRequestedModel != "" {
		parseSelectedModel = parseRequestedModel
	}
	if parseSelectedModel == "" {
		parseSelectedModel = parsePolicy.DefaultModelID
	}
	if parseSelectedModel == "" {
		parseSelectedModel = parseResolveFirstAllowedModelID(parsePolicy)
	}
	if parseSelectedModel == "" {
		parseSelectedModel = parseNormalizeSelectedModelID("")
	}
	if len(parsePolicy.AllowedModelSet) == 0 {
		return parseSelectedModel, false
	}
	if parseHasAllowedModel(parsePolicy, parseSelectedModel) {
		return parseSelectedModel, false
	}
	if parseRequestedModel != "" {
		return parseSelectedModel, true
	}
	if parsePolicy.DefaultModelID != "" && parseHasAllowedModel(parsePolicy, parsePolicy.DefaultModelID) {
		return parsePolicy.DefaultModelID, false
	}
	parseResolvedFirstAllowed := parseResolveFirstAllowedModelID(parsePolicy)
	if parseResolvedFirstAllowed != "" {
		return parseResolvedFirstAllowed, false
	}
	return parseSelectedModel, false
}

// parseHasAllowedModel reports whether a model is present in one policy allow set.
func parseHasAllowedModel(parsePolicy parseBillingModelPolicy, parseModelID string) bool {
	if len(parsePolicy.AllowedModelSet) == 0 {
		return true
	}
	_, hasParseModel := parsePolicy.AllowedModelSet[parseNormalizeSelectedModelID(parseModelID)]
	return hasParseModel
}

// parseResolveFirstAllowedModelID resolves one deterministic first model id from a policy allow set.
func parseResolveFirstAllowedModelID(parsePolicy parseBillingModelPolicy) string {
	if len(parsePolicy.AllowedModelSet) == 0 {
		return ""
	}
	parseModelIDs := make([]string, 0, len(parsePolicy.AllowedModelSet))
	for parseModelID := range parsePolicy.AllowedModelSet {
		parseModelIDs = append(parseModelIDs, parseModelID)
	}
	sort.Strings(parseModelIDs)
	return parseModelIDs[0]
}

// parseFormatModelDeniedByPlan builds one stable user-facing plan restriction message.
func parseFormatModelDeniedByPlan(parseModelID string, parsePolicy parseBillingModelPolicy) string {
	parseAllowedModelIDs := make([]string, 0, len(parsePolicy.AllowedModelSet))
	for parseAllowedModelID := range parsePolicy.AllowedModelSet {
		parseAllowedModelIDs = append(parseAllowedModelIDs, parseAllowedModelID)
	}
	sort.Strings(parseAllowedModelIDs)
	if strings.TrimSpace(parsePolicy.PlanCode) == "" {
		return fmt.Sprintf("model %q is not enabled for this account. Allowed models: %s", parseModelID, strings.Join(parseAllowedModelIDs, ", "))
	}
	return fmt.Sprintf("model %q is not enabled for plan %q. Allowed models: %s", parseModelID, parsePolicy.PlanCode, strings.Join(parseAllowedModelIDs, ", "))
}
