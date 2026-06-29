// Package flags provides browser-visible feature flag and experiment helpers.
//
// The package intentionally evaluates only public, non-secret decisions that
// are safe to transfer through SSR bootstrap, browser storage, or client state.
// Server-side policy, entitlement checks, and secret rollout rules remain
// application-owned.
package flags
