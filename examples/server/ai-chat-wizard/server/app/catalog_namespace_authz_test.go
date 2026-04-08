package app

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestCatalogNamespaceReadScopeBoundaries verifies namespace read authz across public/authenticated/admin boundaries.
func TestCatalogNamespaceReadScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-catalog-authz-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-catalog-authz-bob")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-catalog-authz-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseCarolAuth := parseMustCreateUser(parseT, parseStore, "catalog-normal@example.com")
	parseCarolCtx := parseBindAuthUser(parseServer, "peer-catalog-authz-carol", parseCarolAuth.ID, parseCarolAuth.Email)

	parseAnonymousCtx := context.Background()
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAnonymousCtx, "marketing.home"); parseErr != nil {
		parseT.Fatalf("anonymous marketing namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAnonymousCtx, "auth.login"); parseErr != nil {
		parseT.Fatalf("anonymous auth namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAnonymousCtx, "chat.composer"); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("anonymous chat namespace status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAnonymousCtx, "settings.profile"); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("anonymous settings namespace status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}

	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseCarolCtx, "chat.composer"); parseErr != nil {
		parseT.Fatalf("normal-user chat namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseCarolCtx, "settings.profile"); parseErr != nil {
		parseT.Fatalf("normal-user settings namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseCarolCtx, "admin.customers.list"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("normal-user admin namespace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseBobCtx, "admin.customers.list"); parseErr != nil {
		parseT.Fatalf("workspace-admin customers namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseBobCtx, "dashboard.chats.settings"); parseErr != nil {
		parseT.Fatalf("workspace-admin chats namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseBobCtx, "admin.business.pricing"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin business namespace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseBobCtx, "admin.providers.models"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin providers namespace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAliceCtx, "admin.business.pricing"); parseErr != nil {
		parseT.Fatalf("superuser business namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAliceCtx, "admin.providers.models"); parseErr != nil {
		parseT.Fatalf("superuser providers namespace expected allow: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAliceCtx, "dashboard.ops.incidents"); parseErr != nil {
		parseT.Fatalf("superuser ops namespace expected allow: %v", parseErr)
	}

	if _, parseErr = parseServer.parseAuthorizeCatalogNamespaceRead(parseAliceCtx, "unknown.namespace"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("unknown namespace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestResolveCatalogNamespaceClass verifies namespace classification logic.
func TestResolveCatalogNamespaceClass(parseT *testing.T) {
	parseCases := []struct {
		namespace string
		want      parseCatalogNamespaceClass
	}{
		{namespace: "marketing.home", want: parseCatalogNamespaceClassPublic},
		{namespace: "auth.login", want: parseCatalogNamespaceClassPublic},
		{namespace: "chat.composer", want: parseCatalogNamespaceClassAuthenticated},
		{namespace: "settings.profile", want: parseCatalogNamespaceClassAuthenticated},
		{namespace: "dashboard.customers.list", want: parseCatalogNamespaceClassAdmin},
		{namespace: "admin.providers.models", want: parseCatalogNamespaceClassAdmin},
		{namespace: "unknown.namespace", want: parseCatalogNamespaceClassUnknown},
	}
	for _, parseCase := range parseCases {
		if parseClass := parseResolveCatalogNamespaceClass(parseCase.namespace); parseClass != parseCase.want {
			parseT.Fatalf("namespace=%q class=%q want=%q", parseCase.namespace, parseClass, parseCase.want)
		}
	}
}
