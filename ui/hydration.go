package ui

import "time"

// HydrationOptions configures how a server-rendered tree is resumed in the browser.
type HydrationOptions struct {
	ScriptID          string
	ReferenceScriptID string
	Bootstrap         SSRBootstrap
	BootstrapRef      SSRBootstrapReference
	Strict            bool
	Observability     SSRObservabilityOptions
}

// HydrationStrategy controls when an independently rendered island resumes in
// the browser.
type HydrationStrategy string

const (
	HydrateImmediately          HydrationStrategy = "immediate"
	HydrateOnVisible            HydrationStrategy = "visible"
	HydrateOnInteraction        HydrationStrategy = "interaction"
	HydrateOnIdle               HydrationStrategy = "idle"
	defaultIslandEvent                            = "click"
	defaultIslandSelectorPrefix                   = "[data-gwc-hydration-island=\""
)

// HydrationIslandOptions describes one independently resumable SSR island.
type HydrationIslandOptions struct {
	ID         string
	Selector   string
	Strategy   HydrationStrategy
	Events     []string
	RootMargin string
	Timeout    time.Duration
	Hydration  HydrationOptions
}

// HydrationIslandBudget caps how many islands can hydrate during startup and
// how many deferred resumptions may run at once. Zero values are treated as
// unbounded so apps can opt into one budget at a time.
type HydrationIslandBudget struct {
	MaxInitial    int
	MaxConcurrent int
}

// HydrationIslandPlan groups island options with a startup budget.
type HydrationIslandPlan struct {
	Islands []HydrationIslandOptions
	Budget  HydrationIslandBudget
}

// HydrationIslandBudgetReport summarizes whether a plan fits its budget.
type HydrationIslandBudgetReport struct {
	Initial    int
	Deferred   int
	Violations []string
}

// OK reports whether the plan stayed within every configured budget cap.
func (parseR HydrationIslandBudgetReport) OK() bool {
	return len(parseR.Violations) == 0
}

// resolveHydrationOptions is a core package helper.
func resolveHydrationOptions(parseHydrationOptions []HydrationOptions) HydrationOptions {
	if len(parseHydrationOptions) == 0 {
		return HydrationOptions{}
	}
	return parseHydrationOptions[0]
}
