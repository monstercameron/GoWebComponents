package pluginruntime

import (
	"fmt"
	"strings"
)

// ExecutionClass describes the expected cost profile for one plugin operation.
type ExecutionClass string

const (
	ExecutionClassHot        ExecutionClass = "hot"
	ExecutionClassWarm       ExecutionClass = "warm"
	ExecutionClassBackground ExecutionClass = "background"
)

// ActivationPolicy describes when one plugin or contribution should become active.
type ActivationPolicy string

const (
	ActivationPolicyBoot          ActivationPolicy = "boot"
	ActivationPolicyView          ActivationPolicy = "view"
	ActivationPolicySession       ActivationPolicy = "session"
	ActivationPolicyOpportunistic ActivationPolicy = "opportunistic"
)

// ServiceKey identifies one built-in kernel service family.
type ServiceKey string

const (
	ServiceKeyRuntime      ServiceKey = "runtime"
	ServiceKeyDiagnostics  ServiceKey = "diagnostics"
	ServiceKeyRoute        ServiceKey = "route"
	ServiceKeyFetch        ServiceKey = "fetch"
	ServiceKeyDOM          ServiceKey = "dom"
	ServiceKeyStyle        ServiceKey = "style"
	ServiceKeyEvents       ServiceKey = "events"
	ServiceKeyAssets       ServiceKey = "assets"
	ServiceKeySecurity     ServiceKey = "security"
	ServiceKeyCapture      ServiceKey = "capture"
	ServiceKeyRuntime2Meta ServiceKey = "runtime2-meta"
)

// ContributionKind identifies one built-in contribution family.
type ContributionKind string

const (
	ContributionKindDevtoolsSection ContributionKind = "devtools-section"
	ContributionKindDevtoolsAction  ContributionKind = "devtools-action"
	ContributionKindDevtoolsPanel   ContributionKind = "devtools-panel"
)

// Manifest describes one internal plugin registration contract.
type Manifest struct {
	ID               string
	Version          string
	Description      string
	RequiredServices []ServiceKey
	OptionalServices []ServiceKey
	ActivationPolicy ActivationPolicy
}

// Plugin describes one internal framework-owned plugin.
type Plugin interface {
	Manifest() Manifest
	StartPlugin(Context) (Handle, error)
}

// Handle describes one running plugin handle.
type Handle interface {
	StopPlugin() error
}

// PluginFactory creates one plugin instance for kernel startup.
type PluginFactory func() Plugin

// PluginRegistration stores one boot-time plugin registration.
type PluginRegistration struct {
	Manifest PluginManifestOverride
	Factory  PluginFactory
}

// PluginManifestOverride stores one optional manifest override for registration wiring.
type PluginManifestOverride struct {
	ID               string
	Version          string
	Description      string
	RequiredServices []ServiceKey
	OptionalServices []ServiceKey
	ActivationPolicy ActivationPolicy
}

// KernelInfo reports stable kernel identity metadata.
type KernelInfo struct {
	APIVersion string
}

// QueryBudget describes one bounded query budget for service reads.
type QueryBudget struct {
	MaxItems int
}

// SnapshotMeta reports metadata about one normalized snapshot.
type SnapshotMeta struct {
	BackendID string
	Truncated bool
}

// CommandOptions describes one typed command invocation.
type CommandOptions struct {
	Reason string
}

// validateManifest validates one internal plugin manifest.
func validateManifest(parseManifest Manifest) error {
	parseID := strings.TrimSpace(parseManifest.ID)
	parseVersion := strings.TrimSpace(parseManifest.Version)
	if parseID == "" {
		return fmt.Errorf("pluginruntime: manifest ID is required")
	}
	if parseVersion == "" {
		return fmt.Errorf("pluginruntime: manifest version is required for %q", parseID)
	}
	if parseErr := validateActivationPolicy(parseManifest.ActivationPolicy); parseErr != nil {
		return fmt.Errorf("pluginruntime: invalid activation policy for %q: %w", parseID, parseErr)
	}
	return nil
}

// validateActivationPolicy validates one activation policy value.
func validateActivationPolicy(parsePolicy ActivationPolicy) error {
	switch parsePolicy {
	case "", ActivationPolicyBoot, ActivationPolicyView, ActivationPolicySession, ActivationPolicyOpportunistic:
		return nil
	default:
		return fmt.Errorf("activation policy %q is unsupported", parsePolicy)
	}
}

// cloneManifest returns one cloned manifest value.
func cloneManifest(parseManifest Manifest) Manifest {
	buildClone := parseManifest
	buildClone.RequiredServices = append([]ServiceKey(nil), parseManifest.RequiredServices...)
	buildClone.OptionalServices = append([]ServiceKey(nil), parseManifest.OptionalServices...)
	return buildClone
}

// mergeRegistrationManifest resolves the effective manifest for one registration.
func mergeRegistrationManifest(parseRegistration PluginRegistration) (Manifest, error) {
	if parseRegistration.Factory == nil {
		return Manifest{}, fmt.Errorf("pluginruntime: plugin factory is required")
	}
	parsePlugin := parseRegistration.Factory()
	if parsePlugin == nil {
		return Manifest{}, fmt.Errorf("pluginruntime: plugin factory returned nil plugin")
	}
	buildManifest := cloneManifest(parsePlugin.Manifest())
	if strings.TrimSpace(parseRegistration.Manifest.ID) != "" {
		buildManifest.ID = strings.TrimSpace(parseRegistration.Manifest.ID)
	}
	if strings.TrimSpace(parseRegistration.Manifest.Version) != "" {
		buildManifest.Version = strings.TrimSpace(parseRegistration.Manifest.Version)
	}
	if strings.TrimSpace(parseRegistration.Manifest.Description) != "" {
		buildManifest.Description = strings.TrimSpace(parseRegistration.Manifest.Description)
	}
	if parseRegistration.Manifest.RequiredServices != nil {
		buildManifest.RequiredServices = append([]ServiceKey(nil), parseRegistration.Manifest.RequiredServices...)
	}
	if parseRegistration.Manifest.OptionalServices != nil {
		buildManifest.OptionalServices = append([]ServiceKey(nil), parseRegistration.Manifest.OptionalServices...)
	}
	if parseRegistration.Manifest.ActivationPolicy != "" {
		buildManifest.ActivationPolicy = parseRegistration.Manifest.ActivationPolicy
	}
	if buildManifest.ActivationPolicy == "" {
		buildManifest.ActivationPolicy = ActivationPolicyBoot
	}
	return buildManifest, validateManifest(buildManifest)
}
