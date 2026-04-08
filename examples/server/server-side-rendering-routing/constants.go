package main

const (
	serverBootstrapTransport = "server-json-sidecar"
	serverDeferredPanelID    = "server-streamed-route-panel"
	serverStreamModeReady    = "ready"
	serverStreamModeError    = "error"
	serverStreamModeNested   = "nested"

	serverPageHome     = "home"
	serverPageDocs     = "docs"
	serverPageSearch   = "search"
	serverPageSecure   = "secure"
	serverPageSignIn   = "signin"
	serverPageNotFound = "not-found"

	serverGuideSectionSSR       = "ssr"
	serverGuideSectionRouting   = "routing"
	serverGuideSectionTransport = "transport"

	serverTabOverview  = "overview"
	serverTabLoader    = "loader"
	serverTabTransport = "transport"

	serverSecureRoleMaintainer = "maintainer"
	serverSecureUserDefault    = "Morgan Reconciler"

	legacyRedirectPath = "/docs/routing?tab=loader"
	secureRedirectPath = "/signin?from=secure"
)
