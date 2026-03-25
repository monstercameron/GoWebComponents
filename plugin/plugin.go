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
func Define(parseManifest Manifest, parseSetup DefineFunc) Plugin {
	return definedPlugin{manifest: parseManifest, setup: parseSetup}
}

func (parsePlugin definedPlugin) Manifest() Manifest {
	return parsePlugin.manifest
}

func (parsePlugin definedPlugin) Setup(parseHost *Host) (CleanupFunc, error) {
	if parsePlugin.setup == nil {
		return nil, nil
	}
	return parsePlugin.setup(parseHost)
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
func NewHost(parseOptions HostOptions) *Host {
	parseCapabilities := make(map[Capability]struct{}, len(parseOptions.Capabilities))
	for _, parseCapability := range parseOptions.Capabilities {
		parseTrimmed := Capability(strings.TrimSpace(string(parseCapability)))
		if parseTrimmed == "" {
			continue
		}
		parseCapabilities[parseTrimmed] = struct{}{}
	}
	return &Host{
		capabilities: parseCapabilities,
		values:       map[string]interface{}{},
	}
}

// Register installs a plugin on the host after validating its manifest and capabilities.
func (parseHost *Host) Register(parsePlugin Plugin) error {
	if parseHost == nil {
		return errors.New("plugin: host is nil")
	}
	if parsePlugin == nil {
		return errors.New("plugin: plugin is nil")
	}

	parseManifest := parsePlugin.Manifest()
	if parseErr := validateManifest(parseManifest); parseErr != nil {
		return parseErr
	}
	if parseHost.hasPlugin(parseManifest.ID) {
		return fmt.Errorf("plugin: plugin %q is already registered", parseManifest.ID)
	}
	if parseMissing := parseHost.missingCapabilities(parseManifest.Requires); len(parseMissing) > 0 {
		return fmt.Errorf("plugin: plugin %q requires missing capabilities: %s", parseManifest.ID, strings.Join(parseMissing, ", "))
	}

	parseSnapshot := parseHost.snapshot()
	parseCleanup, parseErr2 := parsePlugin.Setup(parseHost)
	if parseErr2 != nil {
		parseHost.rollback(parseSnapshot)
		return fmt.Errorf("plugin: setup failed for %q: %w", parseManifest.ID, parseErr2)
	}

	parseHost.plugins = append(parseHost.plugins, cloneManifest(parseManifest))
	if parseCleanup != nil {
		parseHost.cleanups = append(parseHost.cleanups, parseCleanup)
	}
	return nil
}

// Close runs all registered cleanup functions in reverse registration order.
func (parseHost *Host) Close() error {
	if parseHost == nil {
		return nil
	}
	var parseJoined error
	for parseIndex := len(parseHost.cleanups) - 1; parseIndex >= 0; parseIndex-- {
		parseCleanup := parseHost.cleanups[parseIndex]
		if parseCleanup == nil {
			continue
		}
		if parseErr := parseCleanup(); parseErr != nil {
			parseJoined = errors.Join(parseJoined, parseErr)
		}
	}
	parseHost.cleanups = nil
	return parseJoined
}

// Capabilities returns the sorted list of capabilities the host was configured with.
func (parseHost *Host) Capabilities() []Capability {
	if parseHost == nil {
		return nil
	}
	parseCapabilities := make([]Capability, 0, len(parseHost.capabilities))
	for parseCapability := range parseHost.capabilities {
		parseCapabilities = append(parseCapabilities, parseCapability)
	}
	sort.Slice(parseCapabilities, func(parseI, parseJ int) bool {
		return parseCapabilities[parseI] < parseCapabilities[parseJ]
	})
	return parseCapabilities
}

// Plugins returns a snapshot of the manifests of all registered plugins.
func (parseHost *Host) Plugins() []Manifest {
	if parseHost == nil {
		return nil
	}
	parsePlugins := make([]Manifest, 0, len(parseHost.plugins))
	for _, parseManifest := range parseHost.plugins {
		parsePlugins = append(parsePlugins, cloneManifest(parseManifest))
	}
	return parsePlugins
}

// SetValue stores a named value on the host for inter-plugin communication.
func (parseHost *Host) SetValue(parseKey string, parseValue interface{}) {
	if parseHost == nil {
		return
	}
	parseTrimmed := strings.TrimSpace(parseKey)
	if parseTrimmed == "" {
		return
	}
	parseHost.values[parseTrimmed] = parseValue
}

// Value retrieves a named value stored on the host.
func (parseHost *Host) Value(parseKey string) (interface{}, bool) {
	if parseHost == nil {
		return nil, false
	}
	parseValue, parseOk := parseHost.values[strings.TrimSpace(parseKey)]
	return parseValue, parseOk
}

// AddRouteGuard registers a route guard evaluated before navigation decisions.
func (parseHost *Host) AddRouteGuard(parseGuard RouteGuard) error {
	if parseErr := parseHost.requireCapability(CapabilityRouter); parseErr != nil {
		return parseErr
	}
	if parseGuard == nil {
		return nil
	}
	parseHost.routeGuards = append(parseHost.routeGuards, parseGuard)
	return nil
}

// AddNavigationObserver registers a callback invoked after each navigation event.
func (parseHost *Host) AddNavigationObserver(parseObserver NavigationObserver) error {
	if parseErr := parseHost.requireCapability(CapabilityRouter); parseErr != nil {
		return parseErr
	}
	if parseObserver == nil {
		return nil
	}
	parseHost.navigationObservers = append(parseHost.navigationObservers, parseObserver)
	return nil
}

// EvaluateRoute runs all registered route guards and returns the first non-allow decision.
func (parseHost *Host) EvaluateRoute(parseRequest RouteRequest) GuardDecision {
	if parseHost == nil {
		return Allow("plugin host unavailable")
	}
	for _, parseGuard := range parseHost.routeGuards {
		if parseGuard == nil {
			continue
		}
		parseDecision := parseGuard(parseRequest)
		if parseDecision.Outcome == "" || parseDecision.Outcome == GuardAllow {
			continue
		}
		return parseDecision
	}
	return Allow("all registered route guards allowed navigation")
}

// NotifyNavigation delivers a navigation event to all registered observers.
func (parseHost *Host) NotifyNavigation(parseEvent NavigationEvent) {
	if parseHost == nil {
		return
	}
	for _, parseObserver := range parseHost.navigationObservers {
		if parseObserver != nil {
			parseObserver(parseEvent)
		}
	}
}

// AddCacheKeyDecorator registers a function that transforms fetch cache keys.
func (parseHost *Host) AddCacheKeyDecorator(parseDecorator CacheKeyDecorator) error {
	if parseErr := parseHost.requireCapability(CapabilityAsyncData); parseErr != nil {
		return parseErr
	}
	if parseDecorator == nil {
		return nil
	}
	parseHost.cacheDecorators = append(parseHost.cacheDecorators, parseDecorator)
	return nil
}

// AddRequestObserver registers a callback invoked for fetch request lifecycle events.
func (parseHost *Host) AddRequestObserver(parseObserver RequestObserver) error {
	if parseErr := parseHost.requireCapability(CapabilityAsyncData); parseErr != nil {
		return parseErr
	}
	if parseObserver == nil {
		return nil
	}
	parseHost.requestObservers = append(parseHost.requestObservers, parseObserver)
	return nil
}

// DecorateCacheKey applies all registered cache-key decorators in order and returns the result.
func (parseHost *Host) DecorateCacheKey(parseKey string) string {
	if parseHost == nil {
		return parseKey
	}
	parseDecorated := parseKey
	for _, parseDecorator := range parseHost.cacheDecorators {
		if parseDecorator != nil {
			parseDecorated = parseDecorator(parseDecorated)
		}
	}
	return parseDecorated
}

// NotifyRequest delivers a request event to all registered request observers.
func (parseHost *Host) NotifyRequest(parseEvent RequestEvent) {
	if parseHost == nil {
		return
	}
	for _, parseObserver := range parseHost.requestObservers {
		if parseObserver != nil {
			parseObserver(parseEvent)
		}
	}
}

// AddPanelProvider registers a devtools panel provider that returns panel metadata.
func (parseHost *Host) AddPanelProvider(parseProvider PanelProvider) error {
	if parseErr := parseHost.requireCapability(CapabilityDevtools); parseErr != nil {
		return parseErr
	}
	if parseProvider == nil {
		return nil
	}
	parseHost.panelProviders = append(parseHost.panelProviders, parseProvider)
	return nil
}

// Panels collects and returns all panels from registered panel providers.
func (parseHost *Host) Panels() []Panel {
	if parseHost == nil {
		return nil
	}
	parsePanels := make([]Panel, 0, len(parseHost.panelProviders))
	for _, parseProvider := range parseHost.panelProviders {
		if parseProvider == nil {
			continue
		}
		parsePanel := parseProvider()
		if strings.TrimSpace(parsePanel.ID) == "" || strings.TrimSpace(parsePanel.Title) == "" {
			continue
		}
		parsePanels = append(parsePanels, parsePanel)
	}
	return parsePanels
}

// AddHeadProvider registers a provider that contributes ui.Node elements to the document head.
func (parseHost *Host) AddHeadProvider(parseProvider HeadProvider) error {
	if parseErr := parseHost.requireCapability(CapabilitySSR); parseErr != nil {
		return parseErr
	}
	if parseProvider == nil {
		return nil
	}
	parseHost.headProviders = append(parseHost.headProviders, parseProvider)
	return nil
}

// HeadNodes collects and returns all head nodes from registered head providers.
func (parseHost *Host) HeadNodes() []ui.Node {
	if parseHost == nil {
		return nil
	}
	parseNodes := make([]ui.Node, 0, len(parseHost.headProviders))
	for _, parseProvider := range parseHost.headProviders {
		if parseProvider == nil {
			continue
		}
		if parseNode := parseProvider(); parseNode != nil {
			parseNodes = append(parseNodes, parseNode)
		}
	}
	return parseNodes
}

// AddBootstrapProvider registers a provider that contributes SSR bootstrap data.
func (parseHost *Host) AddBootstrapProvider(parseProvider BootstrapProvider) error {
	if parseErr := parseHost.requireCapability(CapabilitySSR); parseErr != nil {
		return parseErr
	}
	if parseProvider == nil {
		return nil
	}
	parseHost.bootstrapProviders = append(parseHost.bootstrapProviders, parseProvider)
	return nil
}

// BootstrapData collects and merges bootstrap payloads from all registered providers.
func (parseHost *Host) BootstrapData() map[string]map[string]interface{} {
	if parseHost == nil {
		return nil
	}
	parsePayloads := map[string]map[string]interface{}{}
	for _, parseProvider := range parseHost.bootstrapProviders {
		if parseProvider == nil {
			continue
		}
		parsePayload := parseProvider()
		parseNamespace := strings.TrimSpace(parsePayload.Namespace)
		if parseNamespace == "" || len(parsePayload.Data) == 0 {
			continue
		}
		parseCopyData := make(map[string]interface{}, len(parsePayload.Data))
		for parseKey, parseValue := range parsePayload.Data {
			parseCopyData[parseKey] = parseValue
		}
		parsePayloads[parseNamespace] = parseCopyData
	}
	return parsePayloads
}

// AddFormValidator registers a function that validates form submissions.
func (parseHost *Host) AddFormValidator(parseValidator FormValidator) error {
	if parseErr := parseHost.requireCapability(CapabilityForms); parseErr != nil {
		return parseErr
	}
	if parseValidator == nil {
		return nil
	}
	parseHost.formValidators = append(parseHost.formValidators, parseValidator)
	return nil
}

// AddSubmitObserver registers a callback invoked after each form submission.
func (parseHost *Host) AddSubmitObserver(parseObserver SubmitObserver) error {
	if parseErr := parseHost.requireCapability(CapabilityForms); parseErr != nil {
		return parseErr
	}
	if parseObserver == nil {
		return nil
	}
	parseHost.submitObservers = append(parseHost.submitObservers, parseObserver)
	return nil
}

// ValidateForm runs all registered form validators and returns the combined issues.
func (parseHost *Host) ValidateForm(parseSubmission FormSubmission) []ValidationIssue {
	if parseHost == nil {
		return nil
	}
	parseIssues := make([]ValidationIssue, 0)
	for _, parseValidator := range parseHost.formValidators {
		if parseValidator == nil {
			continue
		}
		parseIssues = append(parseIssues, parseValidator(parseSubmission)...)
	}
	return parseIssues
}

// NotifySubmit delivers a form submission event to all registered submit observers.
func (parseHost *Host) NotifySubmit(parseSubmission FormSubmission) {
	if parseHost == nil {
		return
	}
	for _, parseObserver := range parseHost.submitObservers {
		if parseObserver != nil {
			parseObserver(parseSubmission)
		}
	}
}

// Allow creates a GuardDecision that permits navigation with an optional reason.
func Allow(parseReason string) GuardDecision {
	return GuardDecision{Outcome: GuardAllow, Reason: strings.TrimSpace(parseReason)}
}

// Block creates a GuardDecision that blocks navigation with a reason.
func Block(parseReason string) GuardDecision {
	return GuardDecision{Outcome: GuardBlock, Reason: strings.TrimSpace(parseReason)}
}

// Redirect creates a GuardDecision that redirects navigation to the given path.
func Redirect(parsePath, parseReason string) GuardDecision {
	return GuardDecision{Outcome: GuardRedirect, Redirect: strings.TrimSpace(parsePath), Reason: strings.TrimSpace(parseReason)}
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

func (parseHost *Host) snapshot() registrySnapshot {
	parseValues := make(map[string]interface{}, len(parseHost.values))
	for parseKey, parseValue := range parseHost.values {
		parseValues[parseKey] = parseValue
	}
	return registrySnapshot{
		routeGuards:         len(parseHost.routeGuards),
		navigationObservers: len(parseHost.navigationObservers),
		cacheDecorators:     len(parseHost.cacheDecorators),
		requestObservers:    len(parseHost.requestObservers),
		panelProviders:      len(parseHost.panelProviders),
		headProviders:       len(parseHost.headProviders),
		bootstrapProviders:  len(parseHost.bootstrapProviders),
		formValidators:      len(parseHost.formValidators),
		submitObservers:     len(parseHost.submitObservers),
		cleanups:            len(parseHost.cleanups),
		plugins:             len(parseHost.plugins),
		values:              parseValues,
	}
}

func (parseHost *Host) rollback(parseSnapshot registrySnapshot) {
	parseHost.routeGuards = parseHost.routeGuards[:parseSnapshot.routeGuards]
	parseHost.navigationObservers = parseHost.navigationObservers[:parseSnapshot.navigationObservers]
	parseHost.cacheDecorators = parseHost.cacheDecorators[:parseSnapshot.cacheDecorators]
	parseHost.requestObservers = parseHost.requestObservers[:parseSnapshot.requestObservers]
	parseHost.panelProviders = parseHost.panelProviders[:parseSnapshot.panelProviders]
	parseHost.headProviders = parseHost.headProviders[:parseSnapshot.headProviders]
	parseHost.bootstrapProviders = parseHost.bootstrapProviders[:parseSnapshot.bootstrapProviders]
	parseHost.formValidators = parseHost.formValidators[:parseSnapshot.formValidators]
	parseHost.submitObservers = parseHost.submitObservers[:parseSnapshot.submitObservers]
	parseHost.cleanups = parseHost.cleanups[:parseSnapshot.cleanups]
	parseHost.plugins = parseHost.plugins[:parseSnapshot.plugins]
	parseHost.values = make(map[string]interface{}, len(parseSnapshot.values))
	for parseKey, parseValue := range parseSnapshot.values {
		parseHost.values[parseKey] = parseValue
	}
}

func (parseHost *Host) requireCapability(parseCapability Capability) error {
	if parseHost == nil {
		return errors.New("plugin: host is nil")
	}
	if _, parseOk := parseHost.capabilities[parseCapability]; !parseOk {
		return fmt.Errorf("plugin: capability %q is not enabled on this host", parseCapability)
	}
	return nil
}

func (parseHost *Host) hasPlugin(parseId string) bool {
	parseTrimmed := strings.TrimSpace(parseId)
	for _, parseManifest := range parseHost.plugins {
		if parseManifest.ID == parseTrimmed {
			return true
		}
	}
	return false
}

func (parseHost *Host) missingCapabilities(parseRequired []Capability) []string {
	parseMissing := make([]string, 0)
	for _, parseCapability := range parseRequired {
		parseTrimmed := Capability(strings.TrimSpace(string(parseCapability)))
		if parseTrimmed == "" {
			continue
		}
		if _, parseOk := parseHost.capabilities[parseTrimmed]; !parseOk {
			parseMissing = append(parseMissing, string(parseTrimmed))
		}
	}
	sort.Strings(parseMissing)
	return parseMissing
}

func validateManifest(parseManifest Manifest) error {
	parseManifest.ID = strings.TrimSpace(parseManifest.ID)
	parseManifest.Version = strings.TrimSpace(parseManifest.Version)
	parseManifest.Description = strings.TrimSpace(parseManifest.Description)
	if parseManifest.ID == "" {
		return errors.New("plugin: manifest ID is required")
	}
	if parseManifest.Version == "" {
		return fmt.Errorf("plugin: manifest version is required for %q", parseManifest.ID)
	}
	switch parseManifest.Tier {
	case TierStable, TierSupportedCompanion, TierExperimental, TierInternal:
	default:
		return fmt.Errorf("plugin: manifest tier %q is invalid for %q", parseManifest.Tier, parseManifest.ID)
	}
	return nil
}

func cloneManifest(parseManifest Manifest) Manifest {
	parseClone := parseManifest
	if parseManifest.Requires != nil {
		parseClone.Requires = append([]Capability(nil), parseManifest.Requires...)
	}
	return parseClone
}
