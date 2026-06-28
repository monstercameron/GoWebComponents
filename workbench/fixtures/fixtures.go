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
