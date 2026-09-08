package plugin

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// PluginPanicHandler is invoked when a plugin-supplied callback panics during
// Host dispatch. Plugins are a third-party trust boundary, so a panic in a guard/
// observer/provider/validator is recovered — it must not crash the host — but it
// is NOT swallowed silently: the default handler reports it to stderr. Override
// to route recovered plugin panics to a diagnostics sink instead.
var PluginPanicHandler = func(parseWhat string, parseRecovered any) {
	fmt.Fprintf(os.Stderr, "plugin: recovered panic in %s: %v\n", parseWhat, parseRecovered)
}

// pluginCall runs a plugin-supplied callback that returns a value, converting a
// panic into parseFallback so a faulty plugin cannot crash the host.
func pluginCall[T any](parseWhat string, parseFn func() T, parseFallback T) (parseResult T) {
	defer func() {
		if parseRec := recover(); parseRec != nil {
			PluginPanicHandler(parseWhat, parseRec)
			parseResult = parseFallback
		}
	}()
	return parseFn()
}

// pluginRun runs a plugin-supplied void callback with panic isolation.
func pluginRun(parseWhat string, parseFn func()) {
	defer func() {
		if parseRec := recover(); parseRec != nil {
			PluginPanicHandler(parseWhat, parseRec)
		}
	}()
	parseFn()
}

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
	// registerMu serializes whole Register/Close operations so a plugin's
	// snapshot→Setup→finalize sequence is atomic w.r.t. other registrations.
	// stateMu guards every MUTABLE field below; it is never held across a plugin
	// callback (Setup/guard/observer/provider/cleanup) — dispatch snapshots the
	// relevant slice under RLock and invokes callbacks after releasing it, so a
	// callback that calls back into Add*/Register cannot deadlock. capabilities is
	// immutable after NewHost and needs no lock.
	registerMu sync.Mutex
	stateMu    sync.RWMutex

	capabilities map[Capability]struct{}
	plugins      []Manifest
	cleanups     []CleanupFunc
	values       map[string]any

	// activePluginRequires is the declared-capability set of the plugin currently
	// inside its Setup call, or nil when no Setup is running. registerMu ensures
	// only one plugin is ever in Setup at a time; the pointer is guarded by stateMu
	// so a concurrent direct Add* cannot race-read it. While set, requireCapability
	// additionally enforces that the capability was DECLARED in this plugin's
	// Manifest.Requires — declaring a capability is how a plugin authorizes itself
	// to the matching Add* surface (per-plugin least privilege).
	activePluginRequires map[Capability]struct{}
	activePluginID       string

	routeGuards              []RouteGuard
	navigationObservers      []NavigationObserver
	cacheDecorators          []CacheKeyDecorator
	requestObservers         []RequestObserver
	panelProviders           []PanelProvider
	devtoolsSectionProviders []DevtoolsSectionProvider
	devtoolsActionProviders  []DevtoolsActionProvider
	headProviders            []HeadProvider
	bootstrapProviders       []BootstrapProvider
	formValidators           []FormValidator
	submitObservers          []SubmitObserver
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

type DevtoolsSection struct {
	Name    string
	Summary map[string]string
	Lines   []string
}

type DevtoolsSectionProvider func() DevtoolsSection

type DevtoolsActionContext struct {
	Host          *Host
	IssueCode     string
	IssueSource   string
	IssuePath     string
	IssueMessage  string
	IssueDocs     string
	IssueTopFrame string
}

type DevtoolsAction struct {
	Label        string
	MatchCodes   []string
	MatchSources []string
	Run          func(DevtoolsActionContext)
}

type DevtoolsActionProvider func() []DevtoolsAction

type HeadProvider func() ui.Node

type BootstrapPayload struct {
	Namespace string
	Data      map[string]any
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
		values:       map[string]any{},
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
	// Serialize whole registrations so snapshot→Setup→finalize is atomic w.r.t.
	// other Register/Close calls. Held across the Setup callback; the Add* calls
	// Setup makes take stateMu (a different lock) individually, so no deadlock.
	parseHost.registerMu.Lock()
	defer parseHost.registerMu.Unlock()

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
	// Enter this plugin's Setup scope so every Add* it calls is checked against the
	// capabilities it declared, not merely those enabled host-wide. Cleared on all
	// paths (including a Setup panic) via defer.
	parseHost.beginSetup(parseManifest)
	defer parseHost.endSetup()
	parseCleanup, parseErr2 := parsePlugin.Setup(parseHost)
	if parseErr2 != nil {
		parseHost.rollback(parseSnapshot)
		return fmt.Errorf("plugin: setup failed for %q: %w", parseManifest.ID, parseErr2)
	}

	parseHost.stateMu.Lock()
	parseHost.plugins = append(parseHost.plugins, cloneManifest(parseManifest))
	if parseCleanup != nil {
		parseHost.cleanups = append(parseHost.cleanups, parseCleanup)
	}
	parseHost.stateMu.Unlock()
	return nil
}

// Close runs all registered cleanup functions in reverse registration order.
func (parseHost *Host) Close() error {
	if parseHost == nil {
		return nil
	}
	parseHost.registerMu.Lock()
	defer parseHost.registerMu.Unlock()

	// Snapshot the cleanups under RLock and run them OUTSIDE the lock (before the
	// reset, so a cleanup can still read host state), then clear all state.
	parseHost.stateMu.RLock()
	parseCleanups := append([]CleanupFunc(nil), parseHost.cleanups...)
	parseHost.stateMu.RUnlock()

	var parseJoined error
	for parseIndex := len(parseCleanups) - 1; parseIndex >= 0; parseIndex-- {
		parseCleanup := parseCleanups[parseIndex]
		if parseCleanup == nil {
			continue
		}
		if parseErr := parseCleanup(); parseErr != nil {
			parseJoined = errors.Join(parseJoined, parseErr)
		}
	}

	parseHost.stateMu.Lock()
	parseHost.routeGuards = nil
	parseHost.navigationObservers = nil
	parseHost.cacheDecorators = nil
	parseHost.requestObservers = nil
	parseHost.panelProviders = nil
	parseHost.devtoolsSectionProviders = nil
	parseHost.devtoolsActionProviders = nil
	parseHost.headProviders = nil
	parseHost.bootstrapProviders = nil
	parseHost.formValidators = nil
	parseHost.submitObservers = nil
	parseHost.plugins = nil
	parseHost.values = map[string]any{}
	parseHost.cleanups = nil
	parseHost.stateMu.Unlock()
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
	slices.Sort(parseCapabilities)
	return parseCapabilities
}

// Plugins returns a snapshot of the manifests of all registered plugins.
func (parseHost *Host) Plugins() []Manifest {
	if parseHost == nil {
		return nil
	}
	parseHost.stateMu.RLock()
	defer parseHost.stateMu.RUnlock()
	parsePlugins := make([]Manifest, 0, len(parseHost.plugins))
	for _, parseManifest := range parseHost.plugins {
		parsePlugins = append(parsePlugins, cloneManifest(parseManifest))
	}
	return parsePlugins
}

// SetValue stores a named value on the host for inter-plugin communication.
func (parseHost *Host) SetValue(parseKey string, parseValue any) {
	if parseHost == nil {
		return
	}
	parseTrimmed := strings.TrimSpace(parseKey)
	if parseTrimmed == "" {
		return
	}
	parseHost.stateMu.Lock()
	parseHost.values[parseTrimmed] = parseValue
	parseHost.stateMu.Unlock()
}

// Value retrieves a named value stored on the host.
func (parseHost *Host) Value(parseKey string) (any, bool) {
	if parseHost == nil {
		return nil, false
	}
	parseHost.stateMu.RLock()
	defer parseHost.stateMu.RUnlock()
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
	parseHost.stateMu.Lock()
	parseHost.routeGuards = append(parseHost.routeGuards, parseGuard)
	parseHost.stateMu.Unlock()
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
	parseHost.stateMu.Lock()
	parseHost.navigationObservers = append(parseHost.navigationObservers, parseObserver)
	parseHost.stateMu.Unlock()
	return nil
}

// EvaluateRoute runs all registered route guards and returns the first non-allow decision.
func (parseHost *Host) EvaluateRoute(parseRequest RouteRequest) GuardDecision {
	if parseHost == nil {
		return Allow("plugin host unavailable")
	}
	parseHost.stateMu.RLock()
	parseGuardsSnap := append([]RouteGuard(nil), parseHost.routeGuards...)
	parseHost.stateMu.RUnlock()
	for _, parseGuard := range parseGuardsSnap {
		if parseGuard == nil {
			continue
		}
		// A panicking guard degrades to "no decision" (treated as allow) so a
		// broken third-party plugin cannot block all navigation.
		parseDecision := pluginCall("route guard", func() GuardDecision {
			return parseGuard(parseRequest)
		}, GuardDecision{})
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
	parseHost.stateMu.RLock()
	parseNavSnap := append([]NavigationObserver(nil), parseHost.navigationObservers...)
	parseHost.stateMu.RUnlock()
	for _, parseObserver := range parseNavSnap {
		if parseObserver != nil {
			pluginRun("navigation observer", func() { parseObserver(parseEvent) })
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
	parseHost.stateMu.Lock()
	parseHost.cacheDecorators = append(parseHost.cacheDecorators, parseDecorator)
	parseHost.stateMu.Unlock()
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
	parseHost.stateMu.Lock()
	parseHost.requestObservers = append(parseHost.requestObservers, parseObserver)
	parseHost.stateMu.Unlock()
	return nil
}

// maxDecoratedCacheKeyLen bounds the byte length of a cache key produced by a
// CacheKeyDecorator. Decorators run on the fetch hot path, so a decorator that
// returns a huge (or per-call exponentially growing) string is a memory/CPU DoS
// on every request. Past the cap the decorator's output is REJECTED — the
// pre-decoration key is kept — rather than truncated: truncating two distinct
// over-long keys to the same prefix would silently collide unrelated cache
// entries. 8 KiB is far above any legitimate cache key (URLs, query signatures)
// yet bounds a runaway decorator.
const maxDecoratedCacheKeyLen = 8192

// reportCacheKeyOverflow is invoked when a decorator's output exceeds
// maxDecoratedCacheKeyLen and is therefore rejected. A test seam; the default
// surfaces the rejection to stderr so a misbehaving plugin is not hidden.
var reportCacheKeyOverflow = func(parseLen int) {
	fmt.Fprintf(os.Stderr, "plugin: cache-key decorator output of %d bytes exceeds the %d-byte cap; output rejected, key left undecorated\n", parseLen, maxDecoratedCacheKeyLen)
}

// DecorateCacheKey applies all registered cache-key decorators in order and returns the result.
func (parseHost *Host) DecorateCacheKey(parseKey string) string {
	if parseHost == nil {
		return parseKey
	}
	parseDecorated := parseKey
	parseHost.stateMu.RLock()
	parseDecSnap := append([]CacheKeyDecorator(nil), parseHost.cacheDecorators...)
	parseHost.stateMu.RUnlock()
	for _, parseDecorator := range parseDecSnap {
		if parseDecorator != nil {
			// A panicking decorator passes the key through unchanged.
			parseCurrent := parseDecorated
			parseNext := pluginCall("cache-key decorator", func() string {
				return parseDecorator(parseCurrent)
			}, parseCurrent)
			// Reject over-long decorator output so a runaway decorator cannot
			// balloon the fetch cache key; keep the pre-decoration key.
			if len(parseNext) > maxDecoratedCacheKeyLen {
				reportCacheKeyOverflow(len(parseNext))
				parseNext = parseCurrent
			}
			parseDecorated = parseNext
		}
	}
	return parseDecorated
}

// NotifyRequest delivers a request event to all registered request observers.
func (parseHost *Host) NotifyRequest(parseEvent RequestEvent) {
	if parseHost == nil {
		return
	}
	parseHost.stateMu.RLock()
	parseReqSnap := append([]RequestObserver(nil), parseHost.requestObservers...)
	parseHost.stateMu.RUnlock()
	for _, parseObserver := range parseReqSnap {
		if parseObserver != nil {
			pluginRun("request observer", func() { parseObserver(parseEvent) })
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
	parseHost.stateMu.Lock()
	parseHost.panelProviders = append(parseHost.panelProviders, parseProvider)
	parseHost.stateMu.Unlock()
	return nil
}

// Panels collects and returns all panels from registered panel providers.
func (parseHost *Host) Panels() []Panel {
	if parseHost == nil {
		return nil
	}
	parseHost.stateMu.RLock()
	parsePanelSnap := append([]PanelProvider(nil), parseHost.panelProviders...)
	parseHost.stateMu.RUnlock()
	parsePanels := make([]Panel, 0, len(parsePanelSnap))
	for _, parseProvider := range parsePanelSnap {
		if parseProvider == nil {
			continue
		}
		parsePanel := pluginCall("panel provider", parseProvider, Panel{})
		if strings.TrimSpace(parsePanel.ID) == "" || strings.TrimSpace(parsePanel.Title) == "" {
			continue
		}
		parsePanels = append(parsePanels, parsePanel)
	}
	return parsePanels
}

// AddDevtoolsSectionProvider registers a provider that contributes one devtools extension section.
func (parseHost *Host) AddDevtoolsSectionProvider(parseProvider DevtoolsSectionProvider) error {
	if parseErr := parseHost.requireCapability(CapabilityDevtools); parseErr != nil {
		return parseErr
	}
	if parseProvider == nil {
		return nil
	}
	parseHost.stateMu.Lock()
	parseHost.devtoolsSectionProviders = append(parseHost.devtoolsSectionProviders, parseProvider)
	parseHost.stateMu.Unlock()
	return nil
}

// DevtoolsSections collects and returns all valid devtools extension sections from registered providers.
func (parseHost *Host) DevtoolsSections() []DevtoolsSection {
	if parseHost == nil {
		return nil
	}
	parseHost.stateMu.RLock()
	parseSecSnap := append([]DevtoolsSectionProvider(nil), parseHost.devtoolsSectionProviders...)
	parseHost.stateMu.RUnlock()
	parseSections := make([]DevtoolsSection, 0, len(parseSecSnap))
	for _, parseProvider := range parseSecSnap {
		if parseProvider == nil {
			continue
		}
		parseSection := pluginCall("devtools section provider", parseProvider, DevtoolsSection{})
		if strings.TrimSpace(parseSection.Name) == "" {
			continue
		}
		parseSections = append(parseSections, cloneDevtoolsSection(parseSection))
	}
	return parseSections
}

// AddDevtoolsActionProvider registers a provider that contributes one or more error-overlay recovery actions.
func (parseHost *Host) AddDevtoolsActionProvider(parseProvider DevtoolsActionProvider) error {
	if parseErr := parseHost.requireCapability(CapabilityDevtools); parseErr != nil {
		return parseErr
	}
	if parseProvider == nil {
		return nil
	}
	parseHost.stateMu.Lock()
	parseHost.devtoolsActionProviders = append(parseHost.devtoolsActionProviders, parseProvider)
	parseHost.stateMu.Unlock()
	return nil
}

// DevtoolsActions collects and returns all valid devtools recovery actions from registered providers.
func (parseHost *Host) DevtoolsActions() []DevtoolsAction {
	if parseHost == nil {
		return nil
	}
	parseActions := make([]DevtoolsAction, 0)
	parseHost.stateMu.RLock()
	parseActSnap := append([]DevtoolsActionProvider(nil), parseHost.devtoolsActionProviders...)
	parseHost.stateMu.RUnlock()
	for _, parseProvider := range parseActSnap {
		if parseProvider == nil {
			continue
		}
		for _, parseAction := range pluginCall("devtools action provider", parseProvider, nil) {
			if strings.TrimSpace(parseAction.Label) == "" || parseAction.Run == nil {
				continue
			}
			parseActions = append(parseActions, cloneDevtoolsAction(parseAction))
		}
	}
	return parseActions
}

// AddHeadProvider registers a provider that contributes ui.Node elements to the document head.
func (parseHost *Host) AddHeadProvider(parseProvider HeadProvider) error {
	if parseErr := parseHost.requireCapability(CapabilitySSR); parseErr != nil {
		return parseErr
	}
	if parseProvider == nil {
		return nil
	}
	parseHost.stateMu.Lock()
	parseHost.headProviders = append(parseHost.headProviders, parseProvider)
	parseHost.stateMu.Unlock()
	return nil
}

// HeadNodes collects and returns all head nodes from registered head providers.
func (parseHost *Host) HeadNodes() []ui.Node {
	if parseHost == nil {
		return nil
	}
	parseHost.stateMu.RLock()
	parseHeadSnap := append([]HeadProvider(nil), parseHost.headProviders...)
	parseHost.stateMu.RUnlock()
	parseNodes := make([]ui.Node, 0, len(parseHeadSnap))
	for _, parseProvider := range parseHeadSnap {
		if parseProvider == nil {
			continue
		}
		if parseNode := pluginCall("head provider", parseProvider, nil); parseNode != nil {
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
	parseHost.stateMu.Lock()
	parseHost.bootstrapProviders = append(parseHost.bootstrapProviders, parseProvider)
	parseHost.stateMu.Unlock()
	return nil
}

// BootstrapData collects and merges bootstrap payloads from all registered providers.
func (parseHost *Host) BootstrapData() map[string]map[string]any {
	if parseHost == nil {
		return nil
	}
	parsePayloads := map[string]map[string]any{}
	parseHost.stateMu.RLock()
	parseBootSnap := append([]BootstrapProvider(nil), parseHost.bootstrapProviders...)
	parseHost.stateMu.RUnlock()
	for _, parseProvider := range parseBootSnap {
		if parseProvider == nil {
			continue
		}
		parsePayload := pluginCall("bootstrap provider", parseProvider, BootstrapPayload{})
		parseNamespace := strings.TrimSpace(parsePayload.Namespace)
		if parseNamespace == "" || len(parsePayload.Data) == 0 {
			continue
		}
		parseCopyData := make(map[string]any, len(parsePayload.Data))
		maps.Copy(parseCopyData, parsePayload.Data)
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
	parseHost.stateMu.Lock()
	parseHost.formValidators = append(parseHost.formValidators, parseValidator)
	parseHost.stateMu.Unlock()
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
	parseHost.stateMu.Lock()
	parseHost.submitObservers = append(parseHost.submitObservers, parseObserver)
	parseHost.stateMu.Unlock()
	return nil
}

// ValidateForm runs all registered form validators and returns the combined issues.
func (parseHost *Host) ValidateForm(parseSubmission FormSubmission) []ValidationIssue {
	if parseHost == nil {
		return nil
	}
	parseIssues := make([]ValidationIssue, 0)
	parseHost.stateMu.RLock()
	parseValSnap := append([]FormValidator(nil), parseHost.formValidators...)
	parseHost.stateMu.RUnlock()
	for _, parseValidator := range parseValSnap {
		if parseValidator == nil {
			continue
		}
		parseIssues = append(parseIssues, pluginCall("form validator", func() []ValidationIssue {
			return parseValidator(parseSubmission)
		}, nil)...)
	}
	return parseIssues
}

// NotifySubmit delivers a form submission event to all registered submit observers.
func (parseHost *Host) NotifySubmit(parseSubmission FormSubmission) {
	if parseHost == nil {
		return
	}
	parseHost.stateMu.RLock()
	parseSubSnap := append([]SubmitObserver(nil), parseHost.submitObservers...)
	parseHost.stateMu.RUnlock()
	for _, parseObserver := range parseSubSnap {
		if parseObserver != nil {
			pluginRun("submit observer", func() { parseObserver(parseSubmission) })
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
	routeGuards              int
	navigationObservers      int
	cacheDecorators          int
	requestObservers         int
	panelProviders           int
	devtoolsSectionProviders int
	devtoolsActionProviders  int
	headProviders            int
	bootstrapProviders       int
	formValidators           int
	submitObservers          int
	cleanups                 int
	plugins                  int
	values                   map[string]any
}

func (parseHost *Host) snapshot() registrySnapshot {
	parseHost.stateMu.RLock()
	defer parseHost.stateMu.RUnlock()
	parseValues := make(map[string]any, len(parseHost.values))
	maps.Copy(parseValues, parseHost.values)
	return registrySnapshot{
		routeGuards:              len(parseHost.routeGuards),
		navigationObservers:      len(parseHost.navigationObservers),
		cacheDecorators:          len(parseHost.cacheDecorators),
		requestObservers:         len(parseHost.requestObservers),
		panelProviders:           len(parseHost.panelProviders),
		devtoolsSectionProviders: len(parseHost.devtoolsSectionProviders),
		devtoolsActionProviders:  len(parseHost.devtoolsActionProviders),
		headProviders:            len(parseHost.headProviders),
		bootstrapProviders:       len(parseHost.bootstrapProviders),
		formValidators:           len(parseHost.formValidators),
		submitObservers:          len(parseHost.submitObservers),
		cleanups:                 len(parseHost.cleanups),
		plugins:                  len(parseHost.plugins),
		values:                   parseValues,
	}
}

func (parseHost *Host) rollback(parseSnapshot registrySnapshot) {
	parseHost.stateMu.Lock()
	defer parseHost.stateMu.Unlock()
	parseHost.routeGuards = parseHost.routeGuards[:parseSnapshot.routeGuards]
	parseHost.navigationObservers = parseHost.navigationObservers[:parseSnapshot.navigationObservers]
	parseHost.cacheDecorators = parseHost.cacheDecorators[:parseSnapshot.cacheDecorators]
	parseHost.requestObservers = parseHost.requestObservers[:parseSnapshot.requestObservers]
	parseHost.panelProviders = parseHost.panelProviders[:parseSnapshot.panelProviders]
	parseHost.devtoolsSectionProviders = parseHost.devtoolsSectionProviders[:parseSnapshot.devtoolsSectionProviders]
	parseHost.devtoolsActionProviders = parseHost.devtoolsActionProviders[:parseSnapshot.devtoolsActionProviders]
	parseHost.headProviders = parseHost.headProviders[:parseSnapshot.headProviders]
	parseHost.bootstrapProviders = parseHost.bootstrapProviders[:parseSnapshot.bootstrapProviders]
	parseHost.formValidators = parseHost.formValidators[:parseSnapshot.formValidators]
	parseHost.submitObservers = parseHost.submitObservers[:parseSnapshot.submitObservers]
	parseHost.cleanups = parseHost.cleanups[:parseSnapshot.cleanups]
	parseHost.plugins = parseHost.plugins[:parseSnapshot.plugins]
	parseHost.values = make(map[string]any, len(parseSnapshot.values))
	maps.Copy(parseHost.values, parseSnapshot.values)
}

// beginSetup marks the given plugin as the one currently in Setup, capturing its
// declared capability set so requireCapability can enforce per-plugin scope.
func (parseHost *Host) beginSetup(parseManifest Manifest) {
	parseRequires := make(map[Capability]struct{}, len(parseManifest.Requires))
	for _, parseCapability := range parseManifest.Requires {
		parseTrimmed := Capability(strings.TrimSpace(string(parseCapability)))
		if parseTrimmed != "" {
			parseRequires[parseTrimmed] = struct{}{}
		}
	}
	parseHost.stateMu.Lock()
	parseHost.activePluginRequires = parseRequires
	parseHost.activePluginID = parseManifest.ID
	parseHost.stateMu.Unlock()
}

// endSetup clears the active-Setup plugin scope.
func (parseHost *Host) endSetup() {
	parseHost.stateMu.Lock()
	parseHost.activePluginRequires = nil
	parseHost.activePluginID = ""
	parseHost.stateMu.Unlock()
}

func (parseHost *Host) requireCapability(parseCapability Capability) error {
	if parseHost == nil {
		return errors.New("plugin: host is nil")
	}
	if _, parseOk := parseHost.capabilities[parseCapability]; !parseOk {
		return fmt.Errorf("plugin: capability %q is not enabled on this host", parseCapability)
	}
	// If a plugin is mid-Setup, it may use only the capabilities it DECLARED in its
	// Manifest.Requires — declaring the capability is how a plugin authorizes itself
	// to the matching Add* surface (least privilege). Direct host configuration
	// outside any plugin Setup has no manifest and keeps only the host-wide check.
	parseHost.stateMu.RLock()
	parseRequires := parseHost.activePluginRequires
	parseActiveID := parseHost.activePluginID
	parseHost.stateMu.RUnlock()
	if parseRequires != nil {
		if _, parseOk := parseRequires[parseCapability]; !parseOk {
			return fmt.Errorf("plugin: plugin %q uses capability %q without declaring it in its manifest Requires", parseActiveID, parseCapability)
		}
	}
	return nil
}

func (parseHost *Host) hasPlugin(parseId string) bool {
	parseHost.stateMu.RLock()
	defer parseHost.stateMu.RUnlock()
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

// cloneDevtoolsSection returns a cloned devtools section value.
func cloneDevtoolsSection(parseSection DevtoolsSection) DevtoolsSection {
	parseClone := parseSection
	if parseSection.Summary != nil {
		parseClone.Summary = make(map[string]string, len(parseSection.Summary))
		maps.Copy(parseClone.Summary, parseSection.Summary)
	}
	if parseSection.Lines != nil {
		parseClone.Lines = append([]string(nil), parseSection.Lines...)
	}
	return parseClone
}

// cloneDevtoolsAction returns a cloned devtools action value.
func cloneDevtoolsAction(parseAction DevtoolsAction) DevtoolsAction {
	parseClone := parseAction
	if parseAction.MatchCodes != nil {
		parseClone.MatchCodes = append([]string(nil), parseAction.MatchCodes...)
	}
	if parseAction.MatchSources != nil {
		parseClone.MatchSources = append([]string(nil), parseAction.MatchSources...)
	}
	return parseClone
}
