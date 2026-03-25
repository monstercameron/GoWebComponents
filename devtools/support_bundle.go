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
func CaptureSupportDiagnosticBundle(label string) SupportDiagnosticBundle {
	return SanitizeBugCaptureBundleForSupport(CaptureBugBundle(label))
}

// SanitizeBugCaptureBundleForSupport converts one local debugging bundle into one support-safe artifact.
func SanitizeBugCaptureBundleForSupport(bundle BugCaptureBundle) SupportDiagnosticBundle {
	trace := cloneTraceCapture(bundle.Trace)
	trace.Label = strings.TrimSpace(trace.Label)
	trace.CapturedAt = strings.TrimSpace(trace.CapturedAt)
	trace.Snapshot = sanitizeSnapshotForSupport(trace.Snapshot)

	support := SupportDiagnosticBundle{
		Version:    currentSupportDiagnosticBundleVersion,
		Sanitized:  true,
		Label:      strings.TrimSpace(bundle.Label),
		CapturedAt: strings.TrimSpace(bundle.CapturedAt),
		Trace:      trace,
	}
	if support.Label == "" {
		support.Label = trace.Label
	}
	if support.CapturedAt == "" {
		support.CapturedAt = trace.CapturedAt
	}
	return support
}

// ExportSupportDiagnosticBundleJSON serializes one redacted support-safe bundle into stable JSON.
func ExportSupportDiagnosticBundleJSON(bundle BugCaptureBundle) ([]byte, error) {
	return json.Marshal(SanitizeBugCaptureBundleForSupport(bundle))
}

// ImportSupportDiagnosticBundleJSON deserializes one support-safe bundle from JSON.
func ImportSupportDiagnosticBundleJSON(data []byte) (SupportDiagnosticBundle, error) {
	if len(data) == 0 {
		return SupportDiagnosticBundle{}, nil
	}
	var bundle SupportDiagnosticBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return SupportDiagnosticBundle{}, err
	}
	if bundle.Version == 0 {
		bundle.Version = currentSupportDiagnosticBundleVersion
	}
	bundle.Sanitized = true
	bundle.Label = strings.TrimSpace(bundle.Label)
	bundle.CapturedAt = strings.TrimSpace(bundle.CapturedAt)
	bundle.Trace = cloneTraceCapture(bundle.Trace)
	bundle.Trace.Label = strings.TrimSpace(bundle.Trace.Label)
	bundle.Trace.CapturedAt = strings.TrimSpace(bundle.Trace.CapturedAt)
	if bundle.Label == "" {
		bundle.Label = bundle.Trace.Label
	}
	if bundle.CapturedAt == "" {
		bundle.CapturedAt = bundle.Trace.CapturedAt
	}
	return bundle, nil
}

func sanitizeSnapshotForSupport(snapshot Snapshot) Snapshot {
	snapshot.Route = sanitizeRouteForSupport(snapshot.Route)
	snapshot.Cache = sanitizeCacheEntriesForSupport(snapshot.Cache)
	snapshot.MultiClient = sanitizeMultiClientForSupport(snapshot.MultiClient)
	snapshot.Boundaries = sanitizeBoundaryInspectionForSupport(snapshot.Boundaries)
	snapshot.Coordination = sanitizeCoordinationForSupport(snapshot.Coordination)
	snapshot.Extensions = sanitizeExtensionSectionsForSupport(snapshot.Extensions)
	snapshot.Tree = sanitizeNodeForSupport(snapshot.Tree)
	snapshot.Profiling = sanitizeProfilingForSupport(snapshot.Profiling)
	snapshot.Hydration = sanitizeHydrationForSupport(snapshot.Hydration)
	snapshot.Diagnostics = sanitizeDiagnosticsForSupport(snapshot.Diagnostics)
	snapshot.Logs = sanitizeLogsForSupport(snapshot.Logs)
	return snapshot
}

func sanitizeRouteForSupport(route Route) Route {
	route.Query = sanitizeStringSlicesMapForSupport(route.Query)
	route.Params = sanitizeStringMapForSupport(route.Params)
	for i := range route.Stack {
		route.Stack[i].Params = sanitizeStringMapForSupport(route.Stack[i].Params)
		route.Stack[i].Metadata = sanitizeRouteMetadataForSupport(route.Stack[i].Metadata)
	}
	for i := range route.Loaders {
		route.Loaders[i].Key = redactSupportString(route.Loaders[i].Key)
		route.Loaders[i].Path = redactSupportString(route.Loaders[i].Path)
		route.Loaders[i].Error = redactSupportString(route.Loaders[i].Error)
	}
	route.LastRedirect.Cause = redactSupportString(route.LastRedirect.Cause)
	route.LastRedirect.From = redactSupportString(route.LastRedirect.From)
	route.LastRedirect.To = redactSupportString(route.LastRedirect.To)
	route.Metadata = sanitizeRouteMetadataForSupport(route.Metadata)
	return route
}

func sanitizeRouteMetadataForSupport(metadata RouteMetadata) RouteMetadata {
	metadata.Title = redactSupportString(metadata.Title)
	metadata.Description = redactSupportString(metadata.Description)
	metadata.CanonicalURL = redactSupportString(metadata.CanonicalURL)
	return metadata
}

func sanitizeCacheEntriesForSupport(entries []CacheEntry) []CacheEntry {
	for i := range entries {
		entries[i].Key = redactSupportString(entries[i].Key)
		entries[i].LastError = redactSupportString(entries[i].LastError)
		entries[i].OwnerPaths = sanitizeStringSliceForSupport(entries[i].OwnerPaths)
		entries[i].ResumePolicy = redactSupportString(entries[i].ResumePolicy)
	}
	return entries
}

func sanitizeMultiClientForSupport(multi MultiClient) MultiClient {
	multi.AuthorityView = sanitizeStringMapForSupport(multi.AuthorityView)
	for i := range multi.Peers {
		multi.Peers[i].ID = redactSupportString(multi.Peers[i].ID)
		multi.Peers[i].App = redactSupportString(multi.Peers[i].App)
		multi.Peers[i].Surface = redactSupportString(multi.Peers[i].Surface)
		multi.Peers[i].Role = redactSupportString(multi.Peers[i].Role)
		multi.Peers[i].State = redactSupportString(multi.Peers[i].State)
		multi.Peers[i].ProtocolVersion = redactSupportString(multi.Peers[i].ProtocolVersion)
		multi.Peers[i].Encodings = sanitizeStringSliceForSupport(multi.Peers[i].Encodings)
		multi.Peers[i].Topics = sanitizeStringSliceForSupport(multi.Peers[i].Topics)
	}
	for i := range multi.RecentTraffic {
		multi.RecentTraffic[i].Kind = redactSupportString(multi.RecentTraffic[i].Kind)
		multi.RecentTraffic[i].Topic = redactSupportString(multi.RecentTraffic[i].Topic)
		multi.RecentTraffic[i].PeerID = redactSupportString(multi.RecentTraffic[i].PeerID)
		multi.RecentTraffic[i].CorrelationID = redactSupportString(multi.RecentTraffic[i].CorrelationID)
	}
	for i := range multi.FailedPublishes {
		multi.FailedPublishes[i].Topic = redactSupportString(multi.FailedPublishes[i].Topic)
		multi.FailedPublishes[i].Target = redactSupportString(multi.FailedPublishes[i].Target)
		multi.FailedPublishes[i].Code = redactSupportString(multi.FailedPublishes[i].Code)
		multi.FailedPublishes[i].Message = redactSupportString(multi.FailedPublishes[i].Message)
	}
	return multi
}

func sanitizeBoundaryInspectionForSupport(inspection BoundaryInspection) BoundaryInspection {
	for i := range inspection.Entries {
		inspection.Entries[i].Name = redactSupportString(inspection.Entries[i].Name)
		inspection.Entries[i].Kind = redactSupportString(inspection.Entries[i].Kind)
		inspection.Entries[i].Target = redactSupportString(inspection.Entries[i].Target)
		inspection.Entries[i].CorrelationID = redactSupportString(inspection.Entries[i].CorrelationID)
		inspection.Entries[i].Notes = sanitizeStringSliceForSupport(inspection.Entries[i].Notes)
		inspection.Entries[i].Redacted = sanitizeStringSliceForSupport(inspection.Entries[i].Redacted)
		inspection.Entries[i].Downgraded = sanitizeStringSliceForSupport(inspection.Entries[i].Downgraded)
		inspection.Entries[i].Rejected = sanitizeStringSliceForSupport(inspection.Entries[i].Rejected)
	}
	return inspection
}

func sanitizeCoordinationForSupport(coordination Coordination) Coordination {
	for i := range coordination.Workers {
		coordination.Workers[i].Name = redactSupportString(coordination.Workers[i].Name)
		coordination.Workers[i].URL = redactSupportString(coordination.Workers[i].URL)
		coordination.Workers[i].Kind = redactSupportString(coordination.Workers[i].Kind)
		coordination.Workers[i].RequestID = redactSupportString(coordination.Workers[i].RequestID)
		coordination.Workers[i].Progress = redactSupportString(coordination.Workers[i].Progress)
		coordination.Workers[i].Result = redactSupportString(coordination.Workers[i].Result)
		coordination.Workers[i].Error = redactSupportString(coordination.Workers[i].Error)
		coordination.Workers[i].Correlation = redactSupportString(coordination.Workers[i].Correlation)
	}
	for i := range coordination.SyncEvents {
		coordination.SyncEvents[i].Channel = redactSupportString(coordination.SyncEvents[i].Channel)
		coordination.SyncEvents[i].Topic = redactSupportString(coordination.SyncEvents[i].Topic)
		coordination.SyncEvents[i].Target = redactSupportString(coordination.SyncEvents[i].Target)
		coordination.SyncEvents[i].Error = redactSupportString(coordination.SyncEvents[i].Error)
		coordination.SyncEvents[i].Correlation = redactSupportString(coordination.SyncEvents[i].Correlation)
	}
	for i := range coordination.Replay {
		coordination.Replay[i].ID = redactSupportString(coordination.Replay[i].ID)
		coordination.Replay[i].Kind = redactSupportString(coordination.Replay[i].Kind)
		coordination.Replay[i].Method = redactSupportString(coordination.Replay[i].Method)
		coordination.Replay[i].URL = redactSupportString(coordination.Replay[i].URL)
		coordination.Replay[i].Owner = redactSupportString(coordination.Replay[i].Owner)
		coordination.Replay[i].State = redactSupportString(coordination.Replay[i].State)
		coordination.Replay[i].LastError = redactSupportString(coordination.Replay[i].LastError)
	}
	for i := range coordination.QueueEntries {
		coordination.QueueEntries[i].ID = redactSupportString(coordination.QueueEntries[i].ID)
		coordination.QueueEntries[i].Entity = redactSupportString(coordination.QueueEntries[i].Entity)
		coordination.QueueEntries[i].Operation = redactSupportString(coordination.QueueEntries[i].Operation)
		coordination.QueueEntries[i].Owner = redactSupportString(coordination.QueueEntries[i].Owner)
		coordination.QueueEntries[i].State = redactSupportString(coordination.QueueEntries[i].State)
		coordination.QueueEntries[i].URL = redactSupportString(coordination.QueueEntries[i].URL)
		coordination.QueueEntries[i].LastError = redactSupportString(coordination.QueueEntries[i].LastError)
	}
	for i := range coordination.SyncHealth {
		coordination.SyncHealth[i].Entity = redactSupportString(coordination.SyncHealth[i].Entity)
		coordination.SyncHealth[i].Owner = redactSupportString(coordination.SyncHealth[i].Owner)
		coordination.SyncHealth[i].Status = redactSupportString(coordination.SyncHealth[i].Status)
		coordination.SyncHealth[i].Version = redactSupportString(coordination.SyncHealth[i].Version)
		coordination.SyncHealth[i].LastError = redactSupportString(coordination.SyncHealth[i].LastError)
	}
	coordination.Reconnect.State = redactSupportString(coordination.Reconnect.State)
	coordination.Reconnect.Transport = redactSupportString(coordination.Reconnect.Transport)
	coordination.Conflict.Entity = redactSupportString(coordination.Conflict.Entity)
	coordination.Conflict.Owner = redactSupportString(coordination.Conflict.Owner)
	coordination.Conflict.Status = redactSupportString(coordination.Conflict.Status)
	coordination.Conflict.Strategy = redactSupportString(coordination.Conflict.Strategy)
	coordination.Conflict.LastError = redactSupportString(coordination.Conflict.LastError)
	coordination.LastReplayError = redactSupportString(coordination.LastReplayError)
	return coordination
}

func sanitizeExtensionSectionsForSupport(sections []ExtensionSection) []ExtensionSection {
	for i := range sections {
		sections[i].Name = redactSupportString(sections[i].Name)
		sections[i].Summary = sanitizeStringMapForSupport(sections[i].Summary)
		sections[i].Lines = sanitizeStringSliceForSupport(sections[i].Lines)
	}
	return sections
}

func sanitizeNodeForSupport(node *Node) *Node {
	if node == nil {
		return nil
	}
	cloned := *node
	for i := range cloned.Hooks {
		cloned.Hooks[i].Value = redactSupportString(cloned.Hooks[i].Value)
		cloned.Hooks[i].Dependencies = redactSupportString(cloned.Hooks[i].Dependencies)
		cloned.Hooks[i].Status = redactSupportString(cloned.Hooks[i].Status)
	}
	cloned.ReactiveSource = redactSupportString(cloned.ReactiveSource)
	cloned.UpdateOrigin = redactSupportString(cloned.UpdateOrigin)
	cloned.Signature = redactSupportString(cloned.Signature)
	cloned.Children = make([]Node, 0, len(node.Children))
	for i := range node.Children {
		child := sanitizeNodeForSupport(&node.Children[i])
		if child != nil {
			cloned.Children = append(cloned.Children, *child)
		}
	}
	return &cloned
}

func sanitizeProfilingForSupport(profiling Profiling) Profiling {
	for i := range profiling.ComponentRenders {
		profiling.ComponentRenders[i].LastTrigger = redactSupportString(profiling.ComponentRenders[i].LastTrigger)
	}
	for i := range profiling.RecentEvents {
		profiling.RecentEvents[i].Target = redactSupportString(profiling.RecentEvents[i].Target)
		profiling.RecentEvents[i].CorrelationID = redactSupportString(profiling.RecentEvents[i].CorrelationID)
		profiling.RecentEvents[i].Fields = sanitizeStringMapForSupport(profiling.RecentEvents[i].Fields)
	}
	profiling.Startup.FirstInteractionEvent = redactSupportString(profiling.Startup.FirstInteractionEvent)
	for i := range profiling.Startup.RouteBudgets {
		profiling.Startup.RouteBudgets[i].RouteFamily = redactSupportString(profiling.Startup.RouteBudgets[i].RouteFamily)
		profiling.Startup.RouteBudgets[i].LastRoutePath = redactSupportString(profiling.Startup.RouteBudgets[i].LastRoutePath)
	}
	return profiling
}

func sanitizeHydrationForSupport(hydration HydrationDebug) HydrationDebug {
	hydration.CorrelationID = redactSupportString(hydration.CorrelationID)
	hydration.Failure = redactSupportString(hydration.Failure)
	hydration.RecentMessages = sanitizeStringSliceForSupport(hydration.RecentMessages)
	return hydration
}

func sanitizeDiagnosticsForSupport(diagnostics []Diagnostic) []Diagnostic {
	for i := range diagnostics {
		diagnostics[i].Message = redactSupportString(diagnostics[i].Message)
		diagnostics[i].Path = redactSupportString(diagnostics[i].Path)
		diagnostics[i].TopFrame = redactSupportString(diagnostics[i].TopFrame)
		diagnostics[i].ComponentStack = sanitizeStringSliceForSupport(diagnostics[i].ComponentStack)
		diagnostics[i].Fields = sanitizeStringMapForSupport(diagnostics[i].Fields)
	}
	return diagnostics
}

func sanitizeLogsForSupport(logs []Log) []Log {
	for i := range logs {
		logs[i].Message = redactSupportString(logs[i].Message)
		logs[i].TopFrame = redactSupportString(logs[i].TopFrame)
		logs[i].CorrelationID = redactSupportString(logs[i].CorrelationID)
		logs[i].Fields = sanitizeStringMapForSupport(logs[i].Fields)
	}
	return logs
}

func sanitizeStringMapForSupport(values map[string]string) map[string]string {
	if len(values) == 0 {
		return values
	}
	sanitized := make(map[string]string, len(values))
	for key, value := range values {
		if isSupportSensitiveKey(key) {
			sanitized[key] = redactedSupportValue
			continue
		}
		sanitized[key] = redactSupportString(value)
	}
	return sanitized
}

func sanitizeStringSlicesMapForSupport(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return values
	}
	sanitized := make(map[string][]string, len(values))
	for key, items := range values {
		if isSupportSensitiveKey(key) {
			sanitized[key] = make([]string, len(items))
			for i := range items {
				sanitized[key][i] = redactedSupportValue
			}
			continue
		}
		sanitized[key] = sanitizeStringSliceForSupport(items)
	}
	return sanitized
}

func sanitizeStringSliceForSupport(values []string) []string {
	for i := range values {
		values[i] = redactSupportString(values[i])
	}
	return values
}

func redactSupportString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	if redactedURL, ok := redactSupportURL(trimmed); ok {
		return redactedURL
	}
	redacted := value
	for _, pattern := range supportInlineSecretPatterns {
		redacted = pattern.ReplaceAllStringFunc(redacted, func(match string) string {
			lower := strings.ToLower(match)
			if strings.HasPrefix(lower, "bearer ") {
				return "Bearer " + redactedSupportValue
			}
			index := strings.IndexAny(match, ":=")
			if index == -1 {
				return redactedSupportValue
			}
			return match[:index+1] + " " + redactedSupportValue
		})
	}
	return redacted
}

func redactSupportURL(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if parsed.Scheme == "" && parsed.Host == "" && !strings.Contains(raw, "?") {
		return "", false
	}
	query := parsed.Query()
	if len(query) == 0 {
		return raw, true
	}
	changed := false
	for key, values := range query {
		if !isSupportSensitiveKey(key) {
			continue
		}
		for i := range values {
			values[i] = redactedSupportValue
		}
		query[key] = values
		changed = true
	}
	if !changed {
		return raw, true
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), true
}

func isSupportSensitiveKey(key string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch {
	case normalized == "":
		return false
	case strings.Contains(normalized, "password"):
		return true
	case strings.Contains(normalized, "passwd"):
		return true
	case strings.Contains(normalized, "secret"):
		return true
	case strings.Contains(normalized, "token"):
		return true
	case strings.Contains(normalized, "auth"):
		return true
	case strings.Contains(normalized, "cookie"):
		return true
	case strings.Contains(normalized, "session"):
		return true
	case strings.Contains(normalized, "apikey"):
		return true
	case strings.Contains(normalized, "jwt"):
		return true
	case strings.Contains(normalized, "credential"):
		return true
	default:
		return false
	}
}
