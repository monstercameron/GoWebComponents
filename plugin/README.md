# GWC | Plugin Library

# GoWebComponents (GWC)

## High-Level Overview

The `plugin` library defines the public, app-owned companion extension host for integrating application-specific behavior into GWC.

This package is not the deep framework plugin kernel. The new kernel lives under `internal/pluginruntime` and is reserved for framework-owned services and trusted internal plugins such as kernel-backed devtools contributions.

## Public APIs

### `github.com/monstercameron/GoWebComponents/plugin` (`package plugin`)
- Functions: `AddBootstrapProvider`, `AddCacheKeyDecorator`, `AddDevtoolsActionProvider`, `AddDevtoolsSectionProvider`, `AddFormValidator`, `AddHeadProvider`, `AddNavigationObserver`, `AddPanelProvider`, `AddRequestObserver`, `AddRouteGuard`, `AddSubmitObserver`, `Allow`, `Block`, `BootstrapData`, `Capabilities`, `Close`, `DecorateCacheKey`, `Define`, `DevtoolsActions`, `DevtoolsSections`, `EvaluateRoute`, `HeadNodes`, `Manifest`, `NewHost`, `NotifyNavigation`, `NotifyRequest`, `NotifySubmit`, `Panels`, `Plugins`, `Redirect`, `Register`, `SetValue`, `Setup`, `ValidateForm`, `Value`
- Types: `BootstrapPayload`, `BootstrapProvider`, `CacheKeyDecorator`, `Capability`, `CleanupFunc`, `DefineFunc`, `DevtoolsAction`, `DevtoolsActionContext`, `DevtoolsActionProvider`, `DevtoolsSection`, `DevtoolsSectionProvider`, `FormSubmission`, `FormValidator`, `GuardDecision`, `GuardOutcome`, `HeadProvider`, `Host`, `HostOptions`, `Manifest`, `NavigationEvent`, `NavigationObserver`, `Panel`, `PanelProvider`, `Plugin`, `RequestEvent`, `RequestObserver`, `RouteGuard`, `RouteRequest`, `SubmitObserver`, `Tier`, `ValidationIssue`
- Variables: _none_
- Constants: `CapabilityAsyncData`, `CapabilityDevtools`, `CapabilityForms`, `CapabilityRouter`, `CapabilitySSR`, `GuardAllow`, `GuardBlock`, `GuardRedirect`, `TierExperimental`, `TierInternal`, `TierStable`, `TierSupportedCompanion`

## Subfiles And Purpose

- `doc.go` - Package-level Go documentation
- `plugin.go` - Core implementation for plugin
- `plugin_additional_test.go` - Tests for plugin_additional behavior
- `plugin_rollback_test.go` - Tests for plugin_rollback behavior
- `plugin_test.go` - Tests for plugin behavior

## File Map

```text
plugin/
|-- doc.go
|-- plugin.go
|-- plugin_additional_test.go
|-- plugin_rollback_test.go
\-- plugin_test.go
```



