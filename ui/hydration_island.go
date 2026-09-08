package ui

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// NormalizeHydrationIslandOptions returns a canonical island configuration.
func NormalizeHydrationIslandOptions(parseOptions HydrationIslandOptions) (HydrationIslandOptions, error) {
	parseOptions.ID = strings.TrimSpace(parseOptions.ID)
	parseOptions.Selector = strings.TrimSpace(parseOptions.Selector)
	if parseOptions.ID == "" && parseOptions.Selector == "" {
		return HydrationIslandOptions{}, fmt.Errorf("ui: hydration island requires an ID or selector")
	}
	if parseOptions.Selector == "" {
		parseOptions.Selector = defaultIslandSelectorPrefix + escapeIslandSelectorValue(parseOptions.ID) + "\"]"
	}
	if parseOptions.Strategy == "" {
		parseOptions.Strategy = HydrateOnVisible
	}
	switch parseOptions.Strategy {
	case HydrateImmediately, HydrateOnVisible, HydrateOnInteraction, HydrateOnIdle:
	default:
		return HydrationIslandOptions{}, fmt.Errorf("ui: unsupported hydration island strategy %q", parseOptions.Strategy)
	}
	if parseOptions.Strategy == HydrateOnInteraction {
		parseEvents := make([]string, 0, len(parseOptions.Events))
		parseSeen := map[string]struct{}{}
		for _, parseEvent := range parseOptions.Events {
			parseEvent = strings.TrimSpace(parseEvent)
			if parseEvent == "" {
				continue
			}
			if _, parseExists := parseSeen[parseEvent]; parseExists {
				continue
			}
			parseSeen[parseEvent] = struct{}{}
			parseEvents = append(parseEvents, parseEvent)
		}
		if len(parseEvents) == 0 {
			parseEvents = []string{defaultIslandEvent}
		}
		parseOptions.Events = parseEvents
	} else {
		parseOptions.Events = nil
	}
	return parseOptions, nil
}

// HydrationIsland wraps SSR markup with stable attributes used by HydrateIsland.
func HydrationIsland(parseOptions HydrationIslandOptions, parseContent Node) Node {
	parseNormalized, parseErr := NormalizeHydrationIslandOptions(parseOptions)
	if parseErr != nil {
		panic(parseErr)
	}
	parseProps := map[string]any{
		"data-gwc-hydration-island": parseNormalized.ID,
		"data-gwc-hydration":        string(parseNormalized.Strategy),
	}
	if parseNormalized.RootMargin != "" {
		parseProps["data-gwc-hydration-root-margin"] = parseNormalized.RootMargin
	}
	if len(parseNormalized.Events) > 0 {
		parseProps["data-gwc-hydration-events"] = strings.Join(parseNormalized.Events, " ")
	}
	return runtime.CreateElement("div", parseProps, parseContent)
}

// InspectHydrationIslandBudget validates a progressive hydration plan against
// the configured startup and concurrency caps.
func InspectHydrationIslandBudget(parsePlan HydrationIslandPlan) HydrationIslandBudgetReport {
	parseReport := HydrationIslandBudgetReport{}
	parseIDs := map[string]struct{}{}
	for parseIndex, parseIsland := range parsePlan.Islands {
		parseNormalized, parseErr := NormalizeHydrationIslandOptions(parseIsland)
		if parseErr != nil {
			parseReport.Violations = append(parseReport.Violations, fmt.Sprintf("island %d: %v", parseIndex, parseErr))
			continue
		}
		if parseNormalized.ID != "" {
			if _, parseExists := parseIDs[parseNormalized.ID]; parseExists {
				parseReport.Violations = append(parseReport.Violations, fmt.Sprintf("island %q is duplicated", parseNormalized.ID))
			}
			parseIDs[parseNormalized.ID] = struct{}{}
		}
		if parseNormalized.Strategy == HydrateImmediately {
			parseReport.Initial++
			continue
		}
		parseReport.Deferred++
	}
	if parsePlan.Budget.MaxInitial > 0 && parseReport.Initial > parsePlan.Budget.MaxInitial {
		parseReport.Violations = append(parseReport.Violations, fmt.Sprintf("initial hydration islands %d exceed budget %d", parseReport.Initial, parsePlan.Budget.MaxInitial))
	}
	if parsePlan.Budget.MaxConcurrent > 0 && parseReport.Deferred > parsePlan.Budget.MaxConcurrent {
		parseReport.Violations = append(parseReport.Violations, fmt.Sprintf("deferred hydration islands %d exceed concurrency budget %d", parseReport.Deferred, parsePlan.Budget.MaxConcurrent))
	}
	return parseReport
}

// escapeIslandSelectorValue escapes the narrow selector value shape generated
// from framework-owned island IDs.
func escapeIslandSelectorValue(parseValue string) string {
	parseValue = strings.ReplaceAll(parseValue, "\\", "\\\\")
	return strings.ReplaceAll(parseValue, "\"", "\\\"")
}
