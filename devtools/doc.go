// Package devtools provides lightweight in-browser inspection for
// GoWebComponents applications.
//
// The initial surface focuses on the minimum useful debugging view:
//   - component tree visibility
//   - hook state inspection
//   - current route inspection
//   - structured runtime diagnostics
//
// Use Panel to render an embeddable development overlay inside an app, or call
// SnapshotNow to retrieve the current inspection state programmatically.
//
// Devtools composes three contribution sources:
//   - app-owned sections and overlay actions registered through Set* helpers
//   - compatibility host contributions registered through ApplyHostExtensions
//   - kernel-owned internal contributions resolved from the framework plugin kernel
package devtools
