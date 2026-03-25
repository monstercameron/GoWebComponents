package plugin

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

type Tier string

const (
	TierStable             Tier = "stable"
	TierSupportedCompanion Tier = "supported-companion"
	TierExperimental       Tier = "experimental"
	TierInternal           Tier = "internal"
)

type Capability string

const (
	CapabilityRouter    Capability = "router"
	CapabilityAsyncData Capability = "async-data"
	CapabilityDevtools  Capability = "devtools"
	CapabilitySSR       Capability = "ssr"
	CapabilityForms     Capability = "forms"
)

type Manifest struct {
	ID          string
	Version     string
	Description string
	Tier        Tier
	Requires    []Capability
}

type Plugin interface {
	Manifest() Manifest
	Setup(*Host) (CleanupFunc, error)
}

type CleanupFunc func() error

type DefineFunc func(*Host) (CleanupFunc, error)

type definedPlugin struct {
	manifest Manifest
	setup    DefineFunc
}

// Define creates a Plugin from a manifest and a setup function.
func Define(manifest Manifest, setup DefineFunc) Plugin {
	return definedPlugin{manifest: manifest, setup: setup}
}

func (plugin definedPlugin) Manifest() Manifest {
	return plugin.manifest
}

func (plugin definedPlugin) Setup(host *Host) (CleanupFunc, error) {
	if plugin.setup == nil {
		return nil, nil
	}
	return plugin.setup(host)
}

type HostOptions struct {
	Capabilities []Capability
}

type Host struct {
	capabilities map[Capability]struct{}
	plugins      []Manifest
	cleanups     []CleanupFunc
	values       map[string]interface{}

	routeGuards         []RouteGuard
	navigationObservers []NavigationObserver
	cacheDecorators     []CacheKeyDecorator
	requestObservers    []RequestObserver
	panelProviders      []PanelProvider
	headProviders       []HeadProvider
	bootstrapProviders  []BootstrapProvider
	formValidators      []FormValidator
	submitObservers     []SubmitObserver
}

type RouteRequest struct {
	Path string
	Tags []string
}

type GuardOutcome string

const (
	GuardAllow    GuardOutcome = "allow"
	GuardBlock    GuardOutcome = "block"
	GuardRedirect GuardOutcome = "redirect"
)

type GuardDecision struct {
	Outcome  GuardOutcome
	Reason   string
	Redirect string
}

type RouteGuard func(RouteRequest) GuardDecision

type NavigationEvent struct {
	Path   string
	Source string
}

type NavigationObserver func(NavigationEvent)

type CacheKeyDecorator func(string) string

type RequestEvent struct {
	Key    string
	Phase  string
	Source string
}

type RequestObserver func(RequestEvent)

type Panel struct {
	ID      string
	Title   string
	Summary string
}

type PanelProvider func() Panel

type HeadProvider func() ui.Node

type BootstrapPayload struct {
	Namespace string
	Data      map[string]interface{}
}

type BootstrapProvider func() BootstrapPayload

type FormSubmission struct {
	ID     string
	Intent string
	Values map[string]string
}

type ValidationIssue struct {
	Field   string
	Message string
}

type FormValidator func(FormSubmission) []ValidationIssue

type SubmitObserver func(FormSubmission)

// NewHost creates a Host with the given capabilities configuration.
func NewHost(options HostOptions) *Host {
	capabilities := make(map[Capability]struct{}, len(options.Capabilities))
	for _, capability := range options.Capabilities {
		trimmed := Capability(strings.TrimSpace(string(capability)))
		if trimmed == "" {
			continue
		}
		capabilities[trimmed] = struct{}{}
	}
	return &Host{
		capabilities: capabilities,
		values:       map[string]interface{}{},
	}
}

// Register installs a plugin on the host after validating its manifest and capabilities.
func (host *Host) Register(plugin Plugin) error {
	if host == nil {
		return errors.New("plugin: host is nil")
	}
	if plugin == nil {
		return errors.New("plugin: plugin is nil")
	}

	manifest := plugin.Manifest()
	if err := validateManifest(manifest); err != nil {
		return err
	}
	if host.hasPlugin(manifest.ID) {
		return fmt.Errorf("plugin: plugin %q is already registered", manifest.ID)
	}
	if missing := host.missingCapabilities(manifest.Requires); len(missing) > 0 {
		return fmt.Errorf("plugin: plugin %q requires missing capabilities: %s", manifest.ID, strings.Join(missing, ", "))
	}

	snapshot := host.snapshot()
	cleanup, err := plugin.Setup(host)
	if err != nil {
		host.rollback(snapshot)
		return fmt.Errorf("plugin: setup failed for %q: %w", manifest.ID, err)
	}

	host.plugins = append(host.plugins, cloneManifest(manifest))
	if cleanup != nil {
		host.cleanups = append(host.cleanups, cleanup)
	}
	return nil
}

// Close runs all registered cleanup functions in reverse registration order.
func (host *Host) Close() error {
	if host == nil {
		return nil
	}
	var joined error
	for index := len(host.cleanups) - 1; index >= 0; index-- {
		cleanup := host.cleanups[index]
		if cleanup == nil {
			continue
		}
		if err := cleanup(); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	host.cleanups = nil
	return joined
}

// Capabilities returns the sorted list of capabilities the host was configured with.
func (host *Host) Capabilities() []Capability {
	if host == nil {
		return nil
	}
	capabilities := make([]Capability, 0, len(host.capabilities))
	for capability := range host.capabilities {
		capabilities = append(capabilities, capability)
	}
	sort.Slice(capabilities, func(i, j int) bool {
		return capabilities[i] < capabilities[j]
	})
	return capabilities
}

// Plugins returns a snapshot of the manifests of all registered plugins.
func (host *Host) Plugins() []Manifest {
	if host == nil {
		return nil
	}
	plugins := make([]Manifest, 0, len(host.plugins))
	for _, manifest := range host.plugins {
		plugins = append(plugins, cloneManifest(manifest))
	}
	return plugins
}

// SetValue stores a named value on the host for inter-plugin communication.
func (host *Host) SetValue(key string, value interface{}) {
	if host == nil {
		return
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return
	}
	host.values[trimmed] = value
}

// Value retrieves a named value stored on the host.
func (host *Host) Value(key string) (interface{}, bool) {
	if host == nil {
		return nil, false
	}
	value, ok := host.values[strings.TrimSpace(key)]
	return value, ok
}

// AddRouteGuard registers a route guard evaluated before navigation decisions.
func (host *Host) AddRouteGuard(guard RouteGuard) error {
	if err := host.requireCapability(CapabilityRouter); err != nil {
		return err
	}
	if guard == nil {
		return nil
	}
	host.routeGuards = append(host.routeGuards, guard)
	return nil
}

// AddNavigationObserver registers a callback invoked after each navigation event.
func (host *Host) AddNavigationObserver(observer NavigationObserver) error {
	if err := host.requireCapability(CapabilityRouter); err != nil {
		return err
	}
	if observer == nil {
		return nil
	}
	host.navigationObservers = append(host.navigationObservers, observer)
	return nil
}

// EvaluateRoute runs all registered route guards and returns the first non-allow decision.
func (host *Host) EvaluateRoute(request RouteRequest) GuardDecision {
	if host == nil {
		return Allow("plugin host unavailable")
	}
	for _, guard := range host.routeGuards {
		if guard == nil {
			continue
		}
		decision := guard(request)
		if decision.Outcome == "" || decision.Outcome == GuardAllow {
			continue
		}
		return decision
	}
	return Allow("all registered route guards allowed navigation")
}

// NotifyNavigation delivers a navigation event to all registered observers.
func (host *Host) NotifyNavigation(event NavigationEvent) {
	if host == nil {
		return
	}
	for _, observer := range host.navigationObservers {
		if observer != nil {
			observer(event)
		}
	}
}

// AddCacheKeyDecorator registers a function that transforms fetch cache keys.
func (host *Host) AddCacheKeyDecorator(decorator CacheKeyDecorator) error {
	if err := host.requireCapability(CapabilityAsyncData); err != nil {
		return err
	}
	if decorator == nil {
		return nil
	}
	host.cacheDecorators = append(host.cacheDecorators, decorator)
	return nil
}

// AddRequestObserver registers a callback invoked for fetch request lifecycle events.
func (host *Host) AddRequestObserver(observer RequestObserver) error {
	if err := host.requireCapability(CapabilityAsyncData); err != nil {
		return err
	}
	if observer == nil {
		return nil
	}
	host.requestObservers = append(host.requestObservers, observer)
	return nil
}

// DecorateCacheKey applies all registered cache-key decorators in order and returns the result.
func (host *Host) DecorateCacheKey(key string) string {
	if host == nil {
		return key
	}
	decorated := key
	for _, decorator := range host.cacheDecorators {
		if decorator != nil {
			decorated = decorator(decorated)
		}
	}
	return decorated
}

// NotifyRequest delivers a request event to all registered request observers.
func (host *Host) NotifyRequest(event RequestEvent) {
	if host == nil {
		return
	}
	for _, observer := range host.requestObservers {
		if observer != nil {
			observer(event)
		}
	}
}

// AddPanelProvider registers a devtools panel provider that returns panel metadata.
func (host *Host) AddPanelProvider(provider PanelProvider) error {
	if err := host.requireCapability(CapabilityDevtools); err != nil {
		return err
	}
	if provider == nil {
		return nil
	}
	host.panelProviders = append(host.panelProviders, provider)
	return nil
}

// Panels collects and returns all panels from registered panel providers.
func (host *Host) Panels() []Panel {
	if host == nil {
		return nil
	}
	panels := make([]Panel, 0, len(host.panelProviders))
	for _, provider := range host.panelProviders {
		if provider == nil {
			continue
		}
		panel := provider()
		if strings.TrimSpace(panel.ID) == "" || strings.TrimSpace(panel.Title) == "" {
			continue
		}
		panels = append(panels, panel)
	}
	return panels
}

// AddHeadProvider registers a provider that contributes ui.Node elements to the document head.
func (host *Host) AddHeadProvider(provider HeadProvider) error {
	if err := host.requireCapability(CapabilitySSR); err != nil {
		return err
	}
	if provider == nil {
		return nil
	}
	host.headProviders = append(host.headProviders, provider)
	return nil
}

// HeadNodes collects and returns all head nodes from registered head providers.
func (host *Host) HeadNodes() []ui.Node {
	if host == nil {
		return nil
	}
	nodes := make([]ui.Node, 0, len(host.headProviders))
	for _, provider := range host.headProviders {
		if provider == nil {
			continue
		}
		if node := provider(); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// AddBootstrapProvider registers a provider that contributes SSR bootstrap data.
func (host *Host) AddBootstrapProvider(provider BootstrapProvider) error {
	if err := host.requireCapability(CapabilitySSR); err != nil {
		return err
	}
	if provider == nil {
		return nil
	}
	host.bootstrapProviders = append(host.bootstrapProviders, provider)
	return nil
}

// BootstrapData collects and merges bootstrap payloads from all registered providers.
func (host *Host) BootstrapData() map[string]map[string]interface{} {
	if host == nil {
		return nil
	}
	payloads := map[string]map[string]interface{}{}
	for _, provider := range host.bootstrapProviders {
		if provider == nil {
			continue
		}
		payload := provider()
		namespace := strings.TrimSpace(payload.Namespace)
		if namespace == "" || len(payload.Data) == 0 {
			continue
		}
		copyData := make(map[string]interface{}, len(payload.Data))
		for key, value := range payload.Data {
			copyData[key] = value
		}
		payloads[namespace] = copyData
	}
	return payloads
}

// AddFormValidator registers a function that validates form submissions.
func (host *Host) AddFormValidator(validator FormValidator) error {
	if err := host.requireCapability(CapabilityForms); err != nil {
		return err
	}
	if validator == nil {
		return nil
	}
	host.formValidators = append(host.formValidators, validator)
	return nil
}

// AddSubmitObserver registers a callback invoked after each form submission.
func (host *Host) AddSubmitObserver(observer SubmitObserver) error {
	if err := host.requireCapability(CapabilityForms); err != nil {
		return err
	}
	if observer == nil {
		return nil
	}
	host.submitObservers = append(host.submitObservers, observer)
	return nil
}

// ValidateForm runs all registered form validators and returns the combined issues.
func (host *Host) ValidateForm(submission FormSubmission) []ValidationIssue {
	if host == nil {
		return nil
	}
	issues := make([]ValidationIssue, 0)
	for _, validator := range host.formValidators {
		if validator == nil {
			continue
		}
		issues = append(issues, validator(submission)...)
	}
	return issues
}

// NotifySubmit delivers a form submission event to all registered submit observers.
func (host *Host) NotifySubmit(submission FormSubmission) {
	if host == nil {
		return
	}
	for _, observer := range host.submitObservers {
		if observer != nil {
			observer(submission)
		}
	}
}

// Allow creates a GuardDecision that permits navigation with an optional reason.
func Allow(reason string) GuardDecision {
	return GuardDecision{Outcome: GuardAllow, Reason: strings.TrimSpace(reason)}
}

// Block creates a GuardDecision that blocks navigation with a reason.
func Block(reason string) GuardDecision {
	return GuardDecision{Outcome: GuardBlock, Reason: strings.TrimSpace(reason)}
}

// Redirect creates a GuardDecision that redirects navigation to the given path.
func Redirect(path, reason string) GuardDecision {
	return GuardDecision{Outcome: GuardRedirect, Redirect: strings.TrimSpace(path), Reason: strings.TrimSpace(reason)}
}

type registrySnapshot struct {
	routeGuards         int
	navigationObservers int
	cacheDecorators     int
	requestObservers    int
	panelProviders      int
	headProviders       int
	bootstrapProviders  int
	formValidators      int
	submitObservers     int
	cleanups            int
	plugins             int
	values              map[string]interface{}
}

func (host *Host) snapshot() registrySnapshot {
	values := make(map[string]interface{}, len(host.values))
	for key, value := range host.values {
		values[key] = value
	}
	return registrySnapshot{
		routeGuards:         len(host.routeGuards),
		navigationObservers: len(host.navigationObservers),
		cacheDecorators:     len(host.cacheDecorators),
		requestObservers:    len(host.requestObservers),
		panelProviders:      len(host.panelProviders),
		headProviders:       len(host.headProviders),
		bootstrapProviders:  len(host.bootstrapProviders),
		formValidators:      len(host.formValidators),
		submitObservers:     len(host.submitObservers),
		cleanups:            len(host.cleanups),
		plugins:             len(host.plugins),
		values:              values,
	}
}

func (host *Host) rollback(snapshot registrySnapshot) {
	host.routeGuards = host.routeGuards[:snapshot.routeGuards]
	host.navigationObservers = host.navigationObservers[:snapshot.navigationObservers]
	host.cacheDecorators = host.cacheDecorators[:snapshot.cacheDecorators]
	host.requestObservers = host.requestObservers[:snapshot.requestObservers]
	host.panelProviders = host.panelProviders[:snapshot.panelProviders]
	host.headProviders = host.headProviders[:snapshot.headProviders]
	host.bootstrapProviders = host.bootstrapProviders[:snapshot.bootstrapProviders]
	host.formValidators = host.formValidators[:snapshot.formValidators]
	host.submitObservers = host.submitObservers[:snapshot.submitObservers]
	host.cleanups = host.cleanups[:snapshot.cleanups]
	host.plugins = host.plugins[:snapshot.plugins]
	host.values = make(map[string]interface{}, len(snapshot.values))
	for key, value := range snapshot.values {
		host.values[key] = value
	}
}

func (host *Host) requireCapability(capability Capability) error {
	if host == nil {
		return errors.New("plugin: host is nil")
	}
	if _, ok := host.capabilities[capability]; !ok {
		return fmt.Errorf("plugin: capability %q is not enabled on this host", capability)
	}
	return nil
}

func (host *Host) hasPlugin(id string) bool {
	trimmed := strings.TrimSpace(id)
	for _, manifest := range host.plugins {
		if manifest.ID == trimmed {
			return true
		}
	}
	return false
}

func (host *Host) missingCapabilities(required []Capability) []string {
	missing := make([]string, 0)
	for _, capability := range required {
		trimmed := Capability(strings.TrimSpace(string(capability)))
		if trimmed == "" {
			continue
		}
		if _, ok := host.capabilities[trimmed]; !ok {
			missing = append(missing, string(trimmed))
		}
	}
	sort.Strings(missing)
	return missing
}

func validateManifest(manifest Manifest) error {
	manifest.ID = strings.TrimSpace(manifest.ID)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Description = strings.TrimSpace(manifest.Description)
	if manifest.ID == "" {
		return errors.New("plugin: manifest ID is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("plugin: manifest version is required for %q", manifest.ID)
	}
	switch manifest.Tier {
	case TierStable, TierSupportedCompanion, TierExperimental, TierInternal:
	default:
		return fmt.Errorf("plugin: manifest tier %q is invalid for %q", manifest.Tier, manifest.ID)
	}
	return nil
}

func cloneManifest(manifest Manifest) Manifest {
	clone := manifest
	if manifest.Requires != nil {
		clone.Requires = append([]Capability(nil), manifest.Requires...)
	}
	return clone
}
