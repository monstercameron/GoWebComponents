// Package ui provides the preferred public component API for GoWebComponents.
//
// It exposes:
//   - CreateElement for component composition
//   - Render for browser mounting
//   - UseState, UseEffect, UseMemo, UseRef, and UseId for local stateful logic
//   - UseEvent for typed event handler wrapping
//
// The ui package is the recommended replacement for the older dom/hooks/render
// split when authoring new components.
package ui
