// Package fixtures ships first-class workbench stories for the framework's boundary
// components — async/suspense, error, and hydration boundaries — the stated reason for the
// workbench (FB5/D1). They double as headless smoke tests under workbench.RunStories and as
// gallery entries, so the boundary surfaces are exercised in every state on every run.
package fixtures

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/workbench"
)

// AllStories returns every boundary fixture — async/suspense, error, suspense-boundary, and
// hydration-boundary — so a single RunStories pass exercises all four boundary surfaces (D1).
func AllStories() []workbench.Story {
	parseStories := BoundaryStories()
	parseStories = append(parseStories, SuspenseStories()...)
	parseStories = append(parseStories, HydrationBoundaryStories()...)
	return parseStories
}

// BoundaryStories returns the boundary-component stories: the async boundary in its pending,
// content, and error states, and the error boundary rendering a healthy child.
func BoundaryStories() []workbench.Story {
	return []workbench.Story{
		{
			Name: "AsyncBoundary/pending",
			Render: func() ui.Node {
				return ui.AsyncBoundary(ui.AsyncBoundaryProps{
					Pending:  true,
					Fallback: html.Text("Loading…"),
					Content:  html.Text("data"),
				})
			},
		},
		{
			Name: "AsyncBoundary/content",
			Render: func() ui.Node {
				return ui.AsyncBoundary(ui.AsyncBoundaryProps{Content: html.Text("Loaded data")})
			},
		},
		{
			Name: "AsyncBoundary/error",
			Render: func() ui.Node {
				return ui.AsyncBoundary(ui.AsyncBoundaryProps{
					Error:         errors.New("load failed"),
					ErrorFallback: func(parseErr error) ui.Node { return html.Text("Error: " + parseErr.Error()) },
					Content:       html.Text("data"),
				})
			},
		},
		{
			Name: "ErrorBoundary/healthy-child",
			Render: func() ui.Node {
				return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
					Fallback: html.Text("Something broke"),
					Child:    html.Text("Healthy child"),
				})
			},
		},
	}
}

// SuspenseStories returns suspense-boundary fixtures: a boundary that shows its fallback while a
// child is pending and swaps to the resolved content once ready — the suspense semantics
// (fallback → content) exercised explicitly, distinct from the raw async-state matrix above.
func SuspenseStories() []workbench.Story {
	return []workbench.Story{
		{
			Name: "Suspense/fallback-while-pending",
			Render: func() ui.Node {
				return ui.AsyncBoundary(ui.AsyncBoundaryProps{
					Pending:  true,
					Fallback: html.Text("Suspending…"),
					Content:  html.Text("Resolved view"),
				})
			},
		},
		{
			Name: "Suspense/resolved-content",
			Render: func() ui.Node {
				return ui.AsyncBoundary(ui.AsyncBoundaryProps{
					Pending: false,
					Content: html.Text("Resolved view"),
				})
			},
		},
	}
}

// HydrationBoundaryStories returns hydration-boundary fixtures: a progressive-hydration island in
// each resumption strategy (immediate / on-visible / on-interaction / on-idle), so the
// island-boundary wrapper and its hydration data attributes are exercised in every mode.
func HydrationBoundaryStories() []workbench.Story {
	parseStrategies := []struct {
		name     string
		strategy ui.HydrationStrategy
	}{
		{"immediate", ui.HydrateImmediately},
		{"on-visible", ui.HydrateOnVisible},
		{"on-interaction", ui.HydrateOnInteraction},
		{"on-idle", ui.HydrateOnIdle},
	}
	parseStories := make([]workbench.Story, 0, len(parseStrategies))
	for _, parseEntry := range parseStrategies {
		parseStories = append(parseStories, workbench.Story{
			Name: "HydrationBoundary/" + parseEntry.name,
			Render: func() ui.Node {
				return ui.HydrationIsland(
					ui.HydrationIslandOptions{ID: "island-" + parseEntry.name, Strategy: parseEntry.strategy},
					html.Text("Island content ("+parseEntry.name+")"),
				)
			},
		})
	}
	return parseStories
}
