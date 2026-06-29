package mockdom

// Package mockdom provides in-memory runtime adapters for unit tests that need deterministic DOM state,
// operation-log assertions, or controllable scheduler behavior without a browser.
//
// Maintenance contract:
// - keep the adapter implementations and compile-time interface assertions aligned with `internal/runtime/interfaces.go`
// - expand or update the package tests whenever runtime adapter contracts change so mockdom remains a trustworthy stand-in
// - treat nodes returned by `GetNode(...)` as read-only assertion handles and mutate DOM state through the adapter methods so the operation log stays coherent
