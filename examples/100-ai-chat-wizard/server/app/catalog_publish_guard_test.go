package app

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestCatalogMessageTemplateValidationGuards verifies placeholder syntax and parity guards for server-published catalog messages.
func TestCatalogMessageTemplateValidationGuards(parseT *testing.T) {
	parseBaseMessages := map[string]string{
		"modal.systemPromptTemplate": "Date {{date}} Time {{time}} Memories {{memories}}",
	}
	parseCandidateValid := map[string]string{
		"modal.systemPromptTemplate": "Fecha {{date}} Hora {{time}} Memorias {{memories}}",
	}
	if parseErr := parseValidateCatalogMessageTemplateSet(parseBaseMessages); parseErr != nil {
		parseT.Fatalf("parseValidateCatalogMessageTemplateSet(base) expected allow: %v", parseErr)
	}
	if parseErr := parseValidateCatalogMessageTemplateSet(parseCandidateValid); parseErr != nil {
		parseT.Fatalf("parseValidateCatalogMessageTemplateSet(candidate) expected allow: %v", parseErr)
	}
	if parseErr := parseValidateCatalogMessageTemplateParity(parseBaseMessages, parseCandidateValid); parseErr != nil {
		parseT.Fatalf("parseValidateCatalogMessageTemplateParity(valid) expected allow: %v", parseErr)
	}

	parseMalformedMessages := map[string]string{
		"modal.systemPromptTemplate": "Date {{date Time {{time}}",
	}
	if parseErr := parseValidateCatalogMessageTemplateSet(parseMalformedMessages); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("malformed placeholder status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	parseMissingTemplateMessages := map[string]string{
		"modal.systemPromptTemplate": "Fecha {{date}} Memorias {{memories}}",
	}
	if parseErr := parseValidateCatalogMessageTemplateParity(parseBaseMessages, parseMissingTemplateMessages); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing placeholder parity status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	parseExtraTemplateMessages := map[string]string{
		"modal.systemPromptTemplate": "Fecha {{date}} Hora {{time}} Memorias {{memories}} Usuario {{user_name}}",
	}
	if parseErr := parseValidateCatalogMessageTemplateParity(parseBaseMessages, parseExtraTemplateMessages); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("extra placeholder parity status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	parseUnknownMessageKey := map[string]string{
		"modal.unknownTemplate": "Date {{date}}",
	}
	if parseErr := parseValidateCatalogMessageTemplateParity(parseBaseMessages, parseUnknownMessageKey); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unknown key parity status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestCatalogOverrideWriteScopeBoundaries verifies source-layer write authorization for superuser and workspace-admin callers.
func TestCatalogOverrideWriteScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-catalog-write-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-catalog-write-bob")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-catalog-write-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseCarolAuth := parseMustCreateUser(parseT, parseStore, "catalog-write-normal@example.com")
	parseCarolCtx := parseBindAuthUser(parseServer, "peer-catalog-write-carol", parseCarolAuth.ID, parseCarolAuth.Email)

	parseBobScope, parseErr := parseServer.parseResolveAdminAccessScopeForUserID(parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseResolveAdminAccessScopeForUserID bob: %v", parseErr)
	}
	parseBobWorkspaceID := int64(0)
	for parseWorkspaceID := range parseBobScope.workspaceIDs {
		parseBobWorkspaceID = parseWorkspaceID
		break
	}
	if parseBobWorkspaceID <= 0 {
		parseT.Fatal("expected workspace-admin scope to include at least one workspace id")
	}

	parseScope, parseErr := parseServer.parseRequireCatalogOverrideWriteScope(parseAliceCtx, "marketing.home", "site_override", 0)
	if parseErr != nil {
		parseT.Fatalf("superuser site override expected allow: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseT.Fatalf("superuser site override expected platform scope, got %+v", parseScope)
	}

	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseBobCtx, "settings.profile", "site_override", parseBobWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin site override status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseBobCtx, "settings.profile", "tenant_override", parseBobWorkspaceID); parseErr != nil {
		parseT.Fatalf("workspace-admin tenant override settings expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseBobCtx, "marketing.home", "tenant_override", parseBobWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin tenant override public namespace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseBobCtx, "admin.business.pricing", "tenant_override", parseBobWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin tenant override business namespace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseBobCtx, "admin.customers.list", "tenant_override", parseBobWorkspaceID); parseErr != nil {
		parseT.Fatalf("workspace-admin tenant override customers namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseBobCtx, "settings.profile", "tenant_override", parseBobWorkspaceID+999); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin tenant override out-of-scope workspace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseCarolCtx, "settings.profile", "tenant_override", parseBobWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("normal-user tenant override status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(context.Background(), "settings.profile", "tenant_override", parseBobWorkspaceID); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("anonymous tenant override status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseAliceCtx, "settings.profile", "go_embed", 0); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("immutable source layer status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseRequireCatalogOverrideWriteScope(parseAliceCtx, "settings.profile", "unknown_layer", 0); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unknown source layer status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// BenchmarkExtractCatalogMessageTemplateSet measures placeholder parsing overhead for one representative template.
func BenchmarkExtractCatalogMessageTemplateSet(parseB *testing.B) {
	parseTemplate := "Date {{date}} Time {{time}} Memories {{memories}} User {{user.name}}"
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseTemplateSet, parseErr := parseExtractCatalogMessageTemplateSet(parseTemplate)
		if parseErr != nil {
			parseB.Fatalf("parseExtractCatalogMessageTemplateSet: %v", parseErr)
		}
		if len(parseTemplateSet) != 4 {
			parseB.Fatalf("unexpected template count: %d", len(parseTemplateSet))
		}
	}
}
