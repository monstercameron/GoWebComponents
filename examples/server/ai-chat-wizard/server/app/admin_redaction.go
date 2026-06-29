package app

import (
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/protobuf/proto"
)

const parseWorkspaceScopeRedactionText = "redacted for workspace scope"

// parseShouldRedactAdminDetailForScope reports whether one response should apply workspace-scope redaction.
func parseShouldRedactAdminDetailForScope(parseScope parseAdminAccessScope) bool {
	return !parseScope.isPlatformScope
}

// parseRedactAdminUsageEventByScope applies workspace-scope redaction for sensitive provider/request diagnostics on one usage event.
func parseRedactAdminUsageEventByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.AdminUsageEvent) *chatpb.AdminUsageEvent {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.AdminUsageEvent)
	parseRedactedEntry.ProviderRequestId = ""
	parseRedactedEntry.ClientId = ""
	parseRedactedEntry.TraceId = ""
	parseRedactedEntry.SpanId = ""
	parseRedactedEntry.TraceState = ""
	if strings.TrimSpace(parseRedactedEntry.ErrorMessage) != "" {
		parseRedactedEntry.ErrorMessage = parseWorkspaceScopeRedactionText
	}
	return parseRedactedEntry
}

// parseRedactAdminConversationSummaryByScope applies workspace-scope redaction for transcript preview content on one conversation summary.
func parseRedactAdminConversationSummaryByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.AdminConversationSummary) *chatpb.AdminConversationSummary {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.AdminConversationSummary)
	if strings.TrimSpace(parseRedactedEntry.Preview) != "" {
		parseRedactedEntry.Preview = parseWorkspaceScopeRedactionText
	}
	return parseRedactedEntry
}

// parseRedactAdminAuthSessionEntryByScope applies workspace-scope redaction for session identifiers and network attributes.
func parseRedactAdminAuthSessionEntryByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.AuthSessionEntry) *chatpb.AuthSessionEntry {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.AuthSessionEntry)
	parseRedactedEntry.SessionId = ""
	parseRedactedEntry.IpAddress = ""
	return parseRedactedEntry
}

// parseRedactAdminAuditLogEntryByScope applies workspace-scope redaction for raw audit payload JSON.
func parseRedactAdminAuditLogEntryByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.AuditLogEntry) *chatpb.AuditLogEntry {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.AuditLogEntry)
	parseRedactedEntry.PayloadJson = "{}"
	return parseRedactedEntry
}

// parseRedactAdminSupportTicketEntryByScope applies workspace-scope redaction for full support-ticket bodies.
func parseRedactAdminSupportTicketEntryByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.SupportTicketEntry) *chatpb.SupportTicketEntry {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.SupportTicketEntry)
	if strings.TrimSpace(parseRedactedEntry.Body) != "" {
		parseRedactedEntry.Body = parseWorkspaceScopeRedactionText
	}
	return parseRedactedEntry
}

// parseRedactAdminSupportTicketMessageEntryByScope applies workspace-scope redaction for internal support message content.
func parseRedactAdminSupportTicketMessageEntryByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.SupportTicketMessageEntry) *chatpb.SupportTicketMessageEntry {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.SupportTicketMessageEntry)
	if parseRedactedEntry.IsInternal && strings.TrimSpace(parseRedactedEntry.Body) != "" {
		parseRedactedEntry.Body = parseWorkspaceScopeRedactionText
	}
	return parseRedactedEntry
}

// parseRedactAdminBillingEventEntryByScope applies workspace-scope redaction for raw billing-event provider payloads.
func parseRedactAdminBillingEventEntryByScope(parseScope parseAdminAccessScope, parseEntry *chatpb.AdminBillingEventEntry) *chatpb.AdminBillingEventEntry {
	if parseEntry == nil || !parseShouldRedactAdminDetailForScope(parseScope) {
		return parseEntry
	}
	parseRedactedEntry := proto.Clone(parseEntry).(*chatpb.AdminBillingEventEntry)
	parseRedactedEntry.EventPayloadJson = "{}"
	return parseRedactedEntry
}
