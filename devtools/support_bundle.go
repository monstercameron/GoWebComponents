package devtools

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

const (
	currentSupportDiagnosticBundleVersion = 1
	redactedSupportValue                  = "[redacted]"
)

var supportInlineSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+\b`),
	regexp.MustCompile(`(?i)\b(password|passwd|secret|token|authorization|cookie|session|api[_-]?key|apikey|jwt|credential)\b(\s*[:=]\s*)([^&\s,;]+)`),
}

// CaptureSupportDiagnosticBundle captures one redacted support-safe debugging bundle from the current snapshot.
func CaptureSupportDiagnosticBundle(parseLabel string) SupportDiagnosticBundle {
	return SanitizeBugCaptureBundleForSupport(CaptureBugBundle(parseLabel))
}

// SanitizeBugCaptureBundleForSupport converts one local debugging bundle into one support-safe artifact.
func SanitizeBugCaptureBundleForSupport(parseBundle BugCaptureBundle) SupportDiagnosticBundle {
	parseTrace := cloneTraceCapture(parseBundle.Trace)
	parseTrace.Label = strings.TrimSpace(parseTrace.Label)
	parseTrace.CapturedAt = strings.TrimSpace(parseTrace.CapturedAt)
	parseTrace.Snapshot = sanitizeSnapshotForSupport(parseTrace.Snapshot)

	parseSupport := SupportDiagnosticBundle{
		Version:    currentSupportDiagnosticBundleVersion,
		Sanitized:  true,
		Label:      strings.TrimSpace(parseBundle.Label),
		CapturedAt: strings.TrimSpace(parseBundle.CapturedAt),
		Trace:      parseTrace,
	}
	if parseSupport.Label == "" {
		parseSupport.Label = parseTrace.Label
	}
	if parseSupport.CapturedAt == "" {
		parseSupport.CapturedAt = parseTrace.CapturedAt
	}
	return parseSupport
}

// ExportSupportDiagnosticBundleJSON serializes one redacted support-safe bundle into stable JSON.
func ExportSupportDiagnosticBundleJSON(parseBundle BugCaptureBundle) ([]byte, error) {
	return json.Marshal(SanitizeBugCaptureBundleForSupport(parseBundle))
}

// ImportSupportDiagnosticBundleJSON deserializes one support-safe bundle from JSON.
func ImportSupportDiagnosticBundleJSON(parseData []byte) (SupportDiagnosticBundle, error) {
	if len(parseData) == 0 {
		return SupportDiagnosticBundle{}, nil
	}
	var parseBundle SupportDiagnosticBundle
	if parseErr := json.Unmarshal(parseData, &parseBundle); parseErr != nil {
		return SupportDiagnosticBundle{}, parseErr
	}
	if parseBundle.Version == 0 {
		parseBundle.Version = currentSupportDiagnosticBundleVersion
	}
	parseBundle.Sanitized = true
	parseBundle.Label = strings.TrimSpace(parseBundle.Label)
	parseBundle.CapturedAt = strings.TrimSpace(parseBundle.CapturedAt)
	parseBundle.Trace = cloneTraceCapture(parseBundle.Trace)
	parseBundle.Trace.Label = strings.TrimSpace(parseBundle.Trace.Label)
	parseBundle.Trace.CapturedAt = strings.TrimSpace(parseBundle.Trace.CapturedAt)
	if parseBundle.Label == "" {
		parseBundle.Label = parseBundle.Trace.Label
	}
	if parseBundle.CapturedAt == "" {
		parseBundle.CapturedAt = parseBundle.Trace.CapturedAt
	}
	return parseBundle, nil
}

func sanitizeSnapshotForSupport(parseSnapshot Snapshot) Snapshot {
	parseSnapshot.Route = sanitizeRouteForSupport(parseSnapshot.Route)
	parseSnapshot.Cache = sanitizeCacheEntriesForSupport(parseSnapshot.Cache)
	parseSnapshot.MultiClient = sanitizeMultiClientForSupport(parseSnapshot.MultiClient)
	parseSnapshot.Boundaries = sanitizeBoundaryInspectionForSupport(parseSnapshot.Boundaries)
	parseSnapshot.Coordination = sanitizeCoordinationForSupport(parseSnapshot.Coordination)
	parseSnapshot.Extensions = sanitizeExtensionSectionsForSupport(parseSnapshot.Extensions)
	parseSnapshot.Tree = sanitizeNodeForSupport(parseSnapshot.Tree)
	parseSnapshot.Profiling = sanitizeProfilingForSupport(parseSnapshot.Profiling)
	parseSnapshot.Hydration = sanitizeHydrationForSupport(parseSnapshot.Hydration)
	parseSnapshot.Diagnostics = sanitizeDiagnosticsForSupport(parseSnapshot.Diagnostics)
	parseSnapshot.Logs = sanitizeLogsForSupport(parseSnapshot.Logs)
	parseSnapshot.Kernel = sanitizeKernelForSupport(parseSnapshot.Kernel)
	return parseSnapshot
}

// sanitizeKernelForSupport redacts the freeform plugin-supplied strings in the
// kernel snapshot. Plugin health/diagnostic Reason/Message/Operation fields are
// populated verbatim from third-party plugin reports, so a secret or PII a
// plugin embeds would otherwise flow, un-redacted, into the ONE bundle meant to
// leave the machine (support export) — every other Snapshot section is
// sanitized; Kernel was the omission.
func sanitizeKernelForSupport(parseKernel KernelSnapshot) KernelSnapshot {
	for parseI := range parseKernel.Plugins {
		parseKernel.Plugins[parseI].Reason = redactSupportString(parseKernel.Plugins[parseI].Reason)
		parseKernel.Plugins[parseI].Message = redactSupportString(parseKernel.Plugins[parseI].Message)
	}
	for parseI := range parseKernel.Events {
		parseKernel.Events[parseI].Operation = redactSupportString(parseKernel.Events[parseI].Operation)
		parseKernel.Events[parseI].Reason = redactSupportString(parseKernel.Events[parseI].Reason)
		parseKernel.Events[parseI].Message = redactSupportString(parseKernel.Events[parseI].Message)
	}
	return parseKernel
}

func sanitizeRouteForSupport(parseRoute Route) Route {
	parseRoute.Query = sanitizeStringSlicesMapForSupport(parseRoute.Query)
	parseRoute.Params = sanitizeStringMapForSupport(parseRoute.Params)
	for parseI := range parseRoute.Stack {
		parseRoute.Stack[parseI].Params = sanitizeStringMapForSupport(parseRoute.Stack[parseI].Params)
		parseRoute.Stack[parseI].Metadata = sanitizeRouteMetadataForSupport(parseRoute.Stack[parseI].Metadata)
	}
	for parseI2 := range parseRoute.Loaders {
		parseRoute.Loaders[parseI2].Key = redactSupportString(parseRoute.Loaders[parseI2].Key)
		parseRoute.Loaders[parseI2].Path = redactSupportString(parseRoute.Loaders[parseI2].Path)
		parseRoute.Loaders[parseI2].Error = redactSupportString(parseRoute.Loaders[parseI2].Error)
	}
	parseRoute.LastRedirect.Cause = redactSupportString(parseRoute.LastRedirect.Cause)
	parseRoute.LastRedirect.From = redactSupportString(parseRoute.LastRedirect.From)
	parseRoute.LastRedirect.To = redactSupportString(parseRoute.LastRedirect.To)
	parseRoute.Metadata = sanitizeRouteMetadataForSupport(parseRoute.Metadata)
	return parseRoute
}

func sanitizeRouteMetadataForSupport(parseMetadata RouteMetadata) RouteMetadata {
	parseMetadata.Title = redactSupportString(parseMetadata.Title)
	parseMetadata.Description = redactSupportString(parseMetadata.Description)
	parseMetadata.CanonicalURL = redactSupportString(parseMetadata.CanonicalURL)
	return parseMetadata
}

func sanitizeCacheEntriesForSupport(parseEntries []CacheEntry) []CacheEntry {
	for parseI := range parseEntries {
		parseEntries[parseI].Key = redactSupportString(parseEntries[parseI].Key)
		parseEntries[parseI].LastError = redactSupportString(parseEntries[parseI].LastError)
		parseEntries[parseI].OwnerPaths = sanitizeStringSliceForSupport(parseEntries[parseI].OwnerPaths)
		parseEntries[parseI].ResumePolicy = redactSupportString(parseEntries[parseI].ResumePolicy)
	}
	return parseEntries
}

func sanitizeMultiClientForSupport(parseMulti MultiClient) MultiClient {
	parseMulti.AuthorityView = sanitizeStringMapForSupport(parseMulti.AuthorityView)
	for parseI := range parseMulti.Peers {
		parseMulti.Peers[parseI].ID = redactSupportString(parseMulti.Peers[parseI].ID)
		parseMulti.Peers[parseI].App = redactSupportString(parseMulti.Peers[parseI].App)
		parseMulti.Peers[parseI].Surface = redactSupportString(parseMulti.Peers[parseI].Surface)
		parseMulti.Peers[parseI].Role = redactSupportString(parseMulti.Peers[parseI].Role)
		parseMulti.Peers[parseI].State = redactSupportString(parseMulti.Peers[parseI].State)
		parseMulti.Peers[parseI].ProtocolVersion = redactSupportString(parseMulti.Peers[parseI].ProtocolVersion)
		parseMulti.Peers[parseI].Encodings = sanitizeStringSliceForSupport(parseMulti.Peers[parseI].Encodings)
		parseMulti.Peers[parseI].Topics = sanitizeStringSliceForSupport(parseMulti.Peers[parseI].Topics)
	}
	for parseI2 := range parseMulti.RecentTraffic {
		parseMulti.RecentTraffic[parseI2].Kind = redactSupportString(parseMulti.RecentTraffic[parseI2].Kind)
		parseMulti.RecentTraffic[parseI2].Topic = redactSupportString(parseMulti.RecentTraffic[parseI2].Topic)
		parseMulti.RecentTraffic[parseI2].PeerID = redactSupportString(parseMulti.RecentTraffic[parseI2].PeerID)
		parseMulti.RecentTraffic[parseI2].CorrelationID = redactSupportString(parseMulti.RecentTraffic[parseI2].CorrelationID)
	}
	for parseI3 := range parseMulti.FailedPublishes {
		parseMulti.FailedPublishes[parseI3].Topic = redactSupportString(parseMulti.FailedPublishes[parseI3].Topic)
		parseMulti.FailedPublishes[parseI3].Target = redactSupportString(parseMulti.FailedPublishes[parseI3].Target)
		parseMulti.FailedPublishes[parseI3].Code = redactSupportString(parseMulti.FailedPublishes[parseI3].Code)
		parseMulti.FailedPublishes[parseI3].Message = redactSupportString(parseMulti.FailedPublishes[parseI3].Message)
	}
	return parseMulti
}

func sanitizeBoundaryInspectionForSupport(parseInspection BoundaryInspection) BoundaryInspection {
	for parseI := range parseInspection.Entries {
		parseInspection.Entries[parseI].Name = redactSupportString(parseInspection.Entries[parseI].Name)
		parseInspection.Entries[parseI].Kind = redactSupportString(parseInspection.Entries[parseI].Kind)
		parseInspection.Entries[parseI].Target = redactSupportString(parseInspection.Entries[parseI].Target)
		parseInspection.Entries[parseI].CorrelationID = redactSupportString(parseInspection.Entries[parseI].CorrelationID)
		parseInspection.Entries[parseI].Notes = sanitizeStringSliceForSupport(parseInspection.Entries[parseI].Notes)
		parseInspection.Entries[parseI].Redacted = sanitizeStringSliceForSupport(parseInspection.Entries[parseI].Redacted)
		parseInspection.Entries[parseI].Downgraded = sanitizeStringSliceForSupport(parseInspection.Entries[parseI].Downgraded)
		parseInspection.Entries[parseI].Rejected = sanitizeStringSliceForSupport(parseInspection.Entries[parseI].Rejected)
	}
	return parseInspection
}

func sanitizeCoordinationForSupport(parseCoordination Coordination) Coordination {
	for parseI := range parseCoordination.Workers {
		parseCoordination.Workers[parseI].Name = redactSupportString(parseCoordination.Workers[parseI].Name)
		parseCoordination.Workers[parseI].URL = redactSupportString(parseCoordination.Workers[parseI].URL)
		parseCoordination.Workers[parseI].Kind = redactSupportString(parseCoordination.Workers[parseI].Kind)
		parseCoordination.Workers[parseI].RequestID = redactSupportString(parseCoordination.Workers[parseI].RequestID)
		parseCoordination.Workers[parseI].Progress = redactSupportString(parseCoordination.Workers[parseI].Progress)
		parseCoordination.Workers[parseI].Result = redactSupportString(parseCoordination.Workers[parseI].Result)
		parseCoordination.Workers[parseI].Error = redactSupportString(parseCoordination.Workers[parseI].Error)
		parseCoordination.Workers[parseI].Correlation = redactSupportString(parseCoordination.Workers[parseI].Correlation)
	}
	for parseI2 := range parseCoordination.SyncEvents {
		parseCoordination.SyncEvents[parseI2].Channel = redactSupportString(parseCoordination.SyncEvents[parseI2].Channel)
		parseCoordination.SyncEvents[parseI2].Topic = redactSupportString(parseCoordination.SyncEvents[parseI2].Topic)
		parseCoordination.SyncEvents[parseI2].Target = redactSupportString(parseCoordination.SyncEvents[parseI2].Target)
		parseCoordination.SyncEvents[parseI2].Error = redactSupportString(parseCoordination.SyncEvents[parseI2].Error)
		parseCoordination.SyncEvents[parseI2].Correlation = redactSupportString(parseCoordination.SyncEvents[parseI2].Correlation)
	}
	for parseI3 := range parseCoordination.Replay {
		parseCoordination.Replay[parseI3].ID = redactSupportString(parseCoordination.Replay[parseI3].ID)
		parseCoordination.Replay[parseI3].Kind = redactSupportString(parseCoordination.Replay[parseI3].Kind)
		parseCoordination.Replay[parseI3].Method = redactSupportString(parseCoordination.Replay[parseI3].Method)
		parseCoordination.Replay[parseI3].URL = redactSupportString(parseCoordination.Replay[parseI3].URL)
		parseCoordination.Replay[parseI3].Owner = redactSupportString(parseCoordination.Replay[parseI3].Owner)
		parseCoordination.Replay[parseI3].State = redactSupportString(parseCoordination.Replay[parseI3].State)
		parseCoordination.Replay[parseI3].LastError = redactSupportString(parseCoordination.Replay[parseI3].LastError)
	}
	for parseI4 := range parseCoordination.QueueEntries {
		parseCoordination.QueueEntries[parseI4].ID = redactSupportString(parseCoordination.QueueEntries[parseI4].ID)
		parseCoordination.QueueEntries[parseI4].Entity = redactSupportString(parseCoordination.QueueEntries[parseI4].Entity)
		parseCoordination.QueueEntries[parseI4].Operation = redactSupportString(parseCoordination.QueueEntries[parseI4].Operation)
		parseCoordination.QueueEntries[parseI4].Owner = redactSupportString(parseCoordination.QueueEntries[parseI4].Owner)
		parseCoordination.QueueEntries[parseI4].State = redactSupportString(parseCoordination.QueueEntries[parseI4].State)
		parseCoordination.QueueEntries[parseI4].URL = redactSupportString(parseCoordination.QueueEntries[parseI4].URL)
		parseCoordination.QueueEntries[parseI4].LastError = redactSupportString(parseCoordination.QueueEntries[parseI4].LastError)
	}
	for parseI5 := range parseCoordination.SyncHealth {
		parseCoordination.SyncHealth[parseI5].Entity = redactSupportString(parseCoordination.SyncHealth[parseI5].Entity)
		parseCoordination.SyncHealth[parseI5].Owner = redactSupportString(parseCoordination.SyncHealth[parseI5].Owner)
		parseCoordination.SyncHealth[parseI5].Status = redactSupportString(parseCoordination.SyncHealth[parseI5].Status)
		parseCoordination.SyncHealth[parseI5].Version = redactSupportString(parseCoordination.SyncHealth[parseI5].Version)
		parseCoordination.SyncHealth[parseI5].LastError = redactSupportString(parseCoordination.SyncHealth[parseI5].LastError)
	}
	parseCoordination.Reconnect.State = redactSupportString(parseCoordination.Reconnect.State)
	parseCoordination.Reconnect.Transport = redactSupportString(parseCoordination.Reconnect.Transport)
	parseCoordination.Conflict.Entity = redactSupportString(parseCoordination.Conflict.Entity)
	parseCoordination.Conflict.Owner = redactSupportString(parseCoordination.Conflict.Owner)
	parseCoordination.Conflict.Status = redactSupportString(parseCoordination.Conflict.Status)
	parseCoordination.Conflict.Strategy = redactSupportString(parseCoordination.Conflict.Strategy)
	parseCoordination.Conflict.LastError = redactSupportString(parseCoordination.Conflict.LastError)
	parseCoordination.LastReplayError = redactSupportString(parseCoordination.LastReplayError)
	return parseCoordination
}

func sanitizeExtensionSectionsForSupport(parseSections []ExtensionSection) []ExtensionSection {
	for parseI := range parseSections {
		parseSections[parseI].Name = redactSupportString(parseSections[parseI].Name)
		parseSections[parseI].Summary = sanitizeStringMapForSupport(parseSections[parseI].Summary)
		parseSections[parseI].Lines = sanitizeStringSliceForSupport(parseSections[parseI].Lines)
	}
	return parseSections
}

func sanitizeNodeForSupport(parseNode *Node) *Node {
	if parseNode == nil {
		return nil
	}
	parseCloned := *parseNode
	for parseI := range parseCloned.Hooks {
		parseCloned.Hooks[parseI].Value = redactSupportString(parseCloned.Hooks[parseI].Value)
		parseCloned.Hooks[parseI].Dependencies = redactSupportString(parseCloned.Hooks[parseI].Dependencies)
		parseCloned.Hooks[parseI].Status = redactSupportString(parseCloned.Hooks[parseI].Status)
	}
	parseCloned.ReactiveSource = redactSupportString(parseCloned.ReactiveSource)
	parseCloned.UpdateOrigin = redactSupportString(parseCloned.UpdateOrigin)
	parseCloned.Signature = redactSupportString(parseCloned.Signature)
	parseCloned.Children = make([]Node, 0, len(parseNode.Children))
	for parseI2 := range parseNode.Children {
		parseChild := sanitizeNodeForSupport(&parseNode.Children[parseI2])
		if parseChild != nil {
			parseCloned.Children = append(parseCloned.Children, *parseChild)
		}
	}
	return &parseCloned
}

func sanitizeProfilingForSupport(parseProfiling Profiling) Profiling {
	for parseI := range parseProfiling.ComponentRenders {
		parseProfiling.ComponentRenders[parseI].LastTrigger = redactSupportString(parseProfiling.ComponentRenders[parseI].LastTrigger)
	}
	for parseI2 := range parseProfiling.RecentEvents {
		parseProfiling.RecentEvents[parseI2].Target = redactSupportString(parseProfiling.RecentEvents[parseI2].Target)
		parseProfiling.RecentEvents[parseI2].CorrelationID = redactSupportString(parseProfiling.RecentEvents[parseI2].CorrelationID)
		parseProfiling.RecentEvents[parseI2].Fields = sanitizeStringMapForSupport(parseProfiling.RecentEvents[parseI2].Fields)
	}
	parseProfiling.Startup.FirstInteractionEvent = redactSupportString(parseProfiling.Startup.FirstInteractionEvent)
	for parseI3 := range parseProfiling.Startup.RouteBudgets {
		parseProfiling.Startup.RouteBudgets[parseI3].RouteFamily = redactSupportString(parseProfiling.Startup.RouteBudgets[parseI3].RouteFamily)
		parseProfiling.Startup.RouteBudgets[parseI3].LastRoutePath = redactSupportString(parseProfiling.Startup.RouteBudgets[parseI3].LastRoutePath)
	}
	return parseProfiling
}

func sanitizeHydrationForSupport(parseHydration HydrationDebug) HydrationDebug {
	parseHydration.CorrelationID = redactSupportString(parseHydration.CorrelationID)
	parseHydration.Failure = redactSupportString(parseHydration.Failure)
	parseHydration.RecentMessages = sanitizeStringSliceForSupport(parseHydration.RecentMessages)
	return parseHydration
}

func sanitizeDiagnosticsForSupport(parseDiagnostics []Diagnostic) []Diagnostic {
	for parseI := range parseDiagnostics {
		parseDiagnostics[parseI].Message = redactSupportString(parseDiagnostics[parseI].Message)
		parseDiagnostics[parseI].Path = redactSupportString(parseDiagnostics[parseI].Path)
		parseDiagnostics[parseI].TopFrame = redactSupportString(parseDiagnostics[parseI].TopFrame)
		parseDiagnostics[parseI].ComponentStack = sanitizeStringSliceForSupport(parseDiagnostics[parseI].ComponentStack)
		parseDiagnostics[parseI].Fields = sanitizeStringMapForSupport(parseDiagnostics[parseI].Fields)
	}
	return parseDiagnostics
}

func sanitizeLogsForSupport(parseLogs []Log) []Log {
	for parseI := range parseLogs {
		parseLogs[parseI].Message = redactSupportString(parseLogs[parseI].Message)
		parseLogs[parseI].TopFrame = redactSupportString(parseLogs[parseI].TopFrame)
		parseLogs[parseI].CorrelationID = redactSupportString(parseLogs[parseI].CorrelationID)
		parseLogs[parseI].Fields = sanitizeStringMapForSupport(parseLogs[parseI].Fields)
	}
	return parseLogs
}

func sanitizeStringMapForSupport(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return parseValues
	}
	parseSanitized := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		if isSupportSensitiveKey(parseKey) {
			parseSanitized[parseKey] = redactedSupportValue
			continue
		}
		parseSanitized[parseKey] = redactSupportString(parseValue)
	}
	return parseSanitized
}

func sanitizeStringSlicesMapForSupport(parseValues map[string][]string) map[string][]string {
	if len(parseValues) == 0 {
		return parseValues
	}
	parseSanitized := make(map[string][]string, len(parseValues))
	for parseKey, parseItems := range parseValues {
		if isSupportSensitiveKey(parseKey) {
			parseSanitized[parseKey] = make([]string, len(parseItems))
			for parseI := range parseItems {
				parseSanitized[parseKey][parseI] = redactedSupportValue
			}
			continue
		}
		parseSanitized[parseKey] = sanitizeStringSliceForSupport(parseItems)
	}
	return parseSanitized
}

func sanitizeStringSliceForSupport(parseValues []string) []string {
	for parseI := range parseValues {
		parseValues[parseI] = redactSupportString(parseValues[parseI])
	}
	return parseValues
}

func redactSupportString(parseValue string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return parseValue
	}
	if parseRedactedURL, parseOk := redactSupportURL(parseTrimmed); parseOk {
		return parseRedactedURL
	}
	parseRedacted := parseValue
	for _, parsePattern := range supportInlineSecretPatterns {
		parseRedacted = parsePattern.ReplaceAllStringFunc(parseRedacted, func(parseMatch string) string {
			parseLower := strings.ToLower(parseMatch)
			if strings.HasPrefix(parseLower, "bearer ") {
				return "Bearer " + redactedSupportValue
			}
			parseIndex := strings.IndexAny(parseMatch, ":=")
			if parseIndex == -1 {
				return redactedSupportValue
			}
			return parseMatch[:parseIndex+1] + " " + redactedSupportValue
		})
	}
	return parseRedacted
}

func redactSupportURL(parseRaw string) (string, bool) {
	parseParsed, parseErr := url.Parse(parseRaw)
	if parseErr != nil {
		return "", false
	}
	if parseParsed.Scheme == "" && parseParsed.Host == "" && !strings.Contains(parseRaw, "?") {
		return "", false
	}
	parseQuery := parseParsed.Query()
	if len(parseQuery) == 0 {
		return parseRaw, true
	}
	isParseChanged := false
	for parseKey, parseValues := range parseQuery {
		if !isSupportSensitiveKey(parseKey) {
			continue
		}
		for parseI := range parseValues {
			parseValues[parseI] = redactedSupportValue
		}
		parseQuery[parseKey] = parseValues
		isParseChanged = true
	}
	if !isParseChanged {
		return parseRaw, true
	}
	parseParsed.RawQuery = parseQuery.Encode()
	return parseParsed.String(), true
}

func isSupportSensitiveKey(parseKey string) bool {
	parseNormalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(parseKey)))
	switch {
	case parseNormalized == "":
		return false
	case strings.Contains(parseNormalized, "password"):
		return true
	case strings.Contains(parseNormalized, "passwd"):
		return true
	case strings.Contains(parseNormalized, "secret"):
		return true
	case strings.Contains(parseNormalized, "token"):
		return true
	case strings.Contains(parseNormalized, "auth"):
		return true
	case strings.Contains(parseNormalized, "cookie"):
		return true
	case strings.Contains(parseNormalized, "session"):
		return true
	case strings.Contains(parseNormalized, "apikey"):
		return true
	case strings.Contains(parseNormalized, "jwt"):
		return true
	case strings.Contains(parseNormalized, "credential"):
		return true
	default:
		return false
	}
}
