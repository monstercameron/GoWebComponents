package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseFindSuperuserTestWorkspaceID resolves one seeded workspace id for superuser reliability tests.
func parseFindSuperuserTestWorkspaceID(parseT *testing.T, parseStore *Store) int64 {
	parseT.Helper()

	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	return parseWorkspaces[0].ID
}

// TestStoreSuperuserReliabilityControlFuncs verifies typed store reliability-control upsert/delete helpers.
func TestStoreSuperuserReliabilityControlFuncs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseWorkspaceID := parseFindSuperuserTestWorkspaceID(parseT, parseStore)

	parseSSORow, parseErr := parseStore.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:        parseWorkspaceID,
		ProviderKey:        "okta",
		SAMLEntrypoint:     "https://idp.example.com/saml",
		SAMLIssuer:         "relaydesk",
		SAMLCertificatePEM: "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----",
		DomainsJSON:        `["example.com"]`,
		IsEnabled:          true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserWorkspaceSSOConfig: %v", parseErr)
	}
	if parseSSORow.WorkspaceID != parseWorkspaceID || parseSSORow.ProviderKey != "okta" || parseSSORow.ProviderType != "saml" {
		parseT.Fatalf("unexpected sso row: %+v", parseSSORow)
	}
	parseOIDCSSORow, parseErr := parseStore.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:         parseWorkspaceID,
		ProviderKey:         "oidc",
		ProviderType:        "oidc",
		OIDCIssuerURL:       "https://idp.example.com",
		OIDCClientID:        "relaydesk-oidc-client",
		OIDCClientSecretRef: "vault://relaydesk/oidc/client-secret",
		OIDCScopesJSON:      `["openid","email","profile"]`,
		OIDCClaimsJSON:      `{"email":"email","subject":"sub"}`,
		DomainsJSON:         `["example.com"]`,
		IsEnabled:           true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserWorkspaceSSOConfig(oidc): %v", parseErr)
	}
	if parseOIDCSSORow.ProviderType != "oidc" || parseOIDCSSORow.OIDCIssuerURL == "" || parseOIDCSSORow.OIDCClientID == "" {
		parseT.Fatalf("unexpected oidc sso row: %+v", parseOIDCSSORow)
	}

	parseRetentionRow, parseErr := parseStore.parseUpsertSuperuserDataRetentionPolicy(parseSuperuserDataRetentionPolicyWrite{
		WorkspaceID:   parseWorkspaceID,
		ScopeKey:      "chat.messages",
		RetentionDays: 365,
		PurgeMode:     "delete",
		LegalHoldJSON: `{"enabled":false}`,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserDataRetentionPolicy: %v", parseErr)
	}
	if parseRetentionRow.ScopeKey != "chat.messages" || parseRetentionRow.RetentionDays != 365 {
		parseT.Fatalf("unexpected retention row: %+v", parseRetentionRow)
	}

	parseComplianceRow, parseErr := parseStore.parseUpsertSuperuserComplianceControl(parseSuperuserComplianceControlWrite{
		WorkspaceID:  parseWorkspaceID,
		ControlKey:   "access-review",
		FrameworkKey: "soc2",
		Status:       "implemented",
		OwnerUserID:  parseOwner.ID,
		EvidenceURL:  "https://evidence.example.com/doc/123",
		ReviewedAt:   "2026-03-27T20:00:00Z",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserComplianceControl: %v", parseErr)
	}
	if parseComplianceRow.ControlKey != "access-review" || parseComplianceRow.FrameworkKey != "soc2" {
		parseT.Fatalf("unexpected compliance row: %+v", parseComplianceRow)
	}

	parseSLORow, parseErr := parseStore.parseUpsertSuperuserServiceLevelObjective(parseSuperuserServiceLevelObjectiveWrite{
		SLOKey:             "api-latency",
		ServiceName:        "API Latency",
		ObjectivePercent:   99.9,
		WindowDays:         30,
		ErrorBudgetMinutes: 43,
		StatusPageURL:      "https://status.example.com",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserServiceLevelObjective: %v", parseErr)
	}
	if parseSLORow.SLOKey != "api-latency" || parseSLORow.ServiceName != "API Latency" {
		parseT.Fatalf("unexpected slo row: %+v", parseSLORow)
	}

	parseIncidentRow, parseErr := parseStore.parseUpsertSuperuserIncident(parseSuperuserIncidentWrite{
		IncidentKey:   "incident-20260327-1",
		SLOKey:        "api-latency",
		Severity:      "major",
		Status:        "open",
		Title:         "Regional API latency increase",
		Summary:       "Latency above objective in one region.",
		StartedAt:     "2026-03-27T19:00:00Z",
		PostmortemURL: "",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserIncident: %v", parseErr)
	}
	if parseIncidentRow.IncidentKey != "incident-20260327-1" || parseIncidentRow.SLOKey != "api-latency" {
		parseT.Fatalf("unexpected incident row: %+v", parseIncidentRow)
	}

	parseIncidentUpdateRow, parseErr := parseStore.parseUpsertSuperuserIncidentUpdate(parseSuperuserIncidentUpdateWrite{
		IncidentID:      parseIncidentRow.ID,
		Status:          "investigating",
		Message:         "Incident declared.",
		IsPublic:        true,
		CreatedByUserID: parseOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserIncidentUpdate create: %v", parseErr)
	}
	if parseIncidentUpdateRow.ID <= 0 || parseIncidentUpdateRow.IncidentID != parseIncidentRow.ID {
		parseT.Fatalf("unexpected incident update row: %+v", parseIncidentUpdateRow)
	}
	parseIncidentUpdateRow, parseErr = parseStore.parseUpsertSuperuserIncidentUpdate(parseSuperuserIncidentUpdateWrite{
		ID:              parseIncidentUpdateRow.ID,
		IncidentID:      parseIncidentRow.ID,
		Status:          "resolved",
		Message:         "Incident resolved.",
		IsPublic:        true,
		CreatedByUserID: parseOwner.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserIncidentUpdate update: %v", parseErr)
	}
	if parseIncidentUpdateRow.Status != "resolved" {
		parseT.Fatalf("expected resolved incident update status, got %+v", parseIncidentUpdateRow)
	}
	parseSSORows, parseErr := parseStore.parseListWorkspaceSSOConfigs(25)
	if parseErr != nil || len(parseSSORows) == 0 {
		parseT.Fatalf("parseListWorkspaceSSOConfigs: rows=%+v err=%v", parseSSORows, parseErr)
	}
	parseRetentionRows, parseErr := parseStore.parseListDataRetentionPolicies(25)
	if parseErr != nil || len(parseRetentionRows) == 0 {
		parseT.Fatalf("parseListDataRetentionPolicies: rows=%+v err=%v", parseRetentionRows, parseErr)
	}
	parseComplianceRows, parseErr := parseStore.parseListComplianceControls(25)
	if parseErr != nil || len(parseComplianceRows) == 0 {
		parseT.Fatalf("parseListComplianceControls: rows=%+v err=%v", parseComplianceRows, parseErr)
	}
	parseSLORows, parseErr := parseStore.parseListServiceLevelObjectives(25)
	if parseErr != nil || len(parseSLORows) == 0 {
		parseT.Fatalf("parseListServiceLevelObjectives: rows=%+v err=%v", parseSLORows, parseErr)
	}

	if parseErr = parseStore.parseDeleteSuperuserIncidentUpdate(parseIncidentUpdateRow.ID); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserIncidentUpdate: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserIncident("incident-20260327-1"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserIncident: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserServiceLevelObjective("api-latency"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserServiceLevelObjective: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserComplianceControl(parseWorkspaceID, "access-review", "soc2"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserComplianceControl: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserDataRetentionPolicy(parseWorkspaceID, "chat.messages"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserDataRetentionPolicy: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserWorkspaceSSOConfig(parseWorkspaceID, "okta"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserWorkspaceSSOConfig: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserWorkspaceSSOConfig(parseWorkspaceID, "oidc"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserWorkspaceSSOConfig(oidc): %v", parseErr)
	}

	if _, hasParseRow, parseErr := parseStore.parseGetIncidentUpdateByID(parseIncidentUpdateRow.ID); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted incident update row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
	if _, hasParseRow, parseErr := parseStore.parseGetIncidentByKey("incident-20260327-1"); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted incident row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
	if _, hasParseRow, parseErr := parseStore.parseGetServiceLevelObjectiveByKey("api-latency"); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted slo row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
	if _, hasParseRow, parseErr := parseStore.parseGetComplianceControlByScope(parseWorkspaceID, "access-review", "soc2"); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted compliance row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
	if _, hasParseRow, parseErr := parseStore.parseGetDataRetentionPolicyByScope(parseWorkspaceID, "chat.messages"); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted retention row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
	if _, hasParseRow, parseErr := parseStore.parseGetWorkspaceSSOConfigByScope(parseWorkspaceID, "okta"); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted sso row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
	if _, hasParseRow, parseErr := parseStore.parseGetWorkspaceSSOConfigByScope(parseWorkspaceID, "oidc"); parseErr != nil || hasParseRow {
		parseT.Fatalf("expected deleted oidc sso row to be absent, found=%v err=%v", hasParseRow, parseErr)
	}
}

// TestStoreSuperuserReliabilityListFuncs verifies typed reliability list helpers return seeded rows with default-limit handling.
func TestStoreSuperuserReliabilityListFuncs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseWorkspaceID := parseFindSuperuserTestWorkspaceID(parseT, parseStore)

	if _, parseErr := parseStore.parseUpsertSuperuserWorkspaceSSOConfig(parseSuperuserWorkspaceSSOConfigWrite{
		WorkspaceID:        parseWorkspaceID,
		ProviderKey:        "okta",
		SAMLEntrypoint:     "https://idp.example.com/saml",
		SAMLIssuer:         "relaydesk",
		SAMLCertificatePEM: "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----",
		DomainsJSON:        `["example.com"]`,
		IsEnabled:          true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserWorkspaceSSOConfig: %v", parseErr)
	}
	if _, parseErr := parseStore.parseUpsertSuperuserDataRetentionPolicy(parseSuperuserDataRetentionPolicyWrite{
		WorkspaceID:   parseWorkspaceID,
		ScopeKey:      "chat.messages",
		RetentionDays: 365,
		PurgeMode:     "delete",
		LegalHoldJSON: `{"enabled":false}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserDataRetentionPolicy: %v", parseErr)
	}
	if _, parseErr := parseStore.parseUpsertSuperuserComplianceControl(parseSuperuserComplianceControlWrite{
		WorkspaceID:  parseWorkspaceID,
		ControlKey:   "access-review",
		FrameworkKey: "soc2",
		Status:       "implemented",
		OwnerUserID:  parseOwner.ID,
		EvidenceURL:  "https://evidence.example.com/doc/123",
		ReviewedAt:   "2026-03-27T20:00:00Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserComplianceControl: %v", parseErr)
	}
	if _, parseErr := parseStore.parseUpsertSuperuserServiceLevelObjective(parseSuperuserServiceLevelObjectiveWrite{
		SLOKey:             "api-latency",
		ServiceName:        "API Latency",
		ObjectivePercent:   99.9,
		WindowDays:         30,
		ErrorBudgetMinutes: 43,
		StatusPageURL:      "https://status.example.com",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserServiceLevelObjective: %v", parseErr)
	}

	parseSSORows, parseErr := parseStore.parseListWorkspaceSSOConfigs(0)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaceSSOConfigs: %v", parseErr)
	}
	if len(parseSSORows) == 0 || parseSSORows[0].WorkspaceID <= 0 || parseSSORows[0].ProviderKey == "" {
		parseT.Fatalf("unexpected SSO list rows: %+v", parseSSORows)
	}
	if parseSSORows[0].ProviderType == "" {
		parseT.Fatalf("expected SSO provider type to be populated: %+v", parseSSORows[0])
	}

	parseRetentionRows, parseErr := parseStore.parseListDataRetentionPolicies(0)
	if parseErr != nil {
		parseT.Fatalf("parseListDataRetentionPolicies: %v", parseErr)
	}
	if len(parseRetentionRows) == 0 || parseRetentionRows[0].ScopeKey == "" {
		parseT.Fatalf("unexpected data-retention list rows: %+v", parseRetentionRows)
	}

	parseControlRows, parseErr := parseStore.parseListComplianceControls(0)
	if parseErr != nil {
		parseT.Fatalf("parseListComplianceControls: %v", parseErr)
	}
	if len(parseControlRows) == 0 || parseControlRows[0].ControlKey == "" || parseControlRows[0].FrameworkKey == "" {
		parseT.Fatalf("unexpected compliance-control list rows: %+v", parseControlRows)
	}

	parseSLORows, parseErr := parseStore.parseListServiceLevelObjectives(0)
	if parseErr != nil {
		parseT.Fatalf("parseListServiceLevelObjectives: %v", parseErr)
	}
	if len(parseSLORows) == 0 || parseSLORows[0].SLOKey == "" || parseSLORows[0].ServiceName == "" {
		parseT.Fatalf("unexpected SLO list rows: %+v", parseSLORows)
	}
}

// TestSuperuserReliabilityControlRPCs verifies typed superuser reliability CRUD RPC behavior and role gating.
func TestSuperuserReliabilityControlRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseWorkspaceID := parseFindSuperuserTestWorkspaceID(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-superuser-reliability-rpc", parseOwner.ID, parseOwner.Email)

	parseSSOResp, parseErr := parseServer.SetSuperuserWorkspaceSSOConfig(parseSuperuserCtx, &chatpb.SetSuperuserWorkspaceSSOConfigRequest{
		WorkspaceId:        parseWorkspaceID,
		ProviderKey:        "okta",
		ProviderType:       "saml",
		SamlEntrypoint:     "https://idp.example.com/saml",
		SamlIssuer:         "relaydesk",
		SamlCertificatePem: "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----",
		DomainsJson:        `["example.com"]`,
		IsEnabled:          true,
		Confirm:            true,
		Reason:             "sso rollout",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserWorkspaceSSOConfig: %v", parseErr)
	}
	if parseSSOResp.GetConfig().GetProviderKey() != "okta" || parseSSOResp.GetConfig().GetProviderType() != "saml" || parseSSOResp.GetStatus() != "updated" {
		parseT.Fatalf("unexpected SetSuperuserWorkspaceSSOConfig response: %+v", parseSSOResp)
	}
	parseOIDCSSOResp, parseErr := parseServer.SetSuperuserWorkspaceSSOConfig(parseSuperuserCtx, &chatpb.SetSuperuserWorkspaceSSOConfigRequest{
		WorkspaceId:         parseWorkspaceID,
		ProviderKey:         "oidc",
		ProviderType:        "oidc",
		OidcIssuerUrl:       "https://idp.example.com",
		OidcClientId:        "relaydesk-oidc-client",
		OidcClientSecretRef: "vault://relaydesk/oidc/client-secret",
		OidcScopesJson:      `["openid","email","profile"]`,
		OidcClaimsJson:      `{"email":"email","subject":"sub"}`,
		DomainsJson:         `["example.com"]`,
		IsEnabled:           true,
		Confirm:             true,
		Reason:              "oidc rollout",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserWorkspaceSSOConfig(oidc): %v", parseErr)
	}
	if parseOIDCSSOResp.GetConfig().GetProviderType() != "oidc" || parseOIDCSSOResp.GetConfig().GetOidcIssuerUrl() == "" || parseOIDCSSOResp.GetConfig().GetOidcClientId() == "" {
		parseT.Fatalf("unexpected SetSuperuserWorkspaceSSOConfig(oidc) response: %+v", parseOIDCSSOResp)
	}

	parseRetentionResp, parseErr := parseServer.SetSuperuserDataRetentionPolicy(parseSuperuserCtx, &chatpb.SetSuperuserDataRetentionPolicyRequest{
		WorkspaceId:   parseWorkspaceID,
		ScopeKey:      "chat.messages",
		RetentionDays: 365,
		PurgeMode:     "delete",
		LegalHoldJson: `{"enabled":false}`,
		Confirm:       true,
		Reason:        "retention update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserDataRetentionPolicy: %v", parseErr)
	}
	if parseRetentionResp.GetPolicy().GetScopeKey() != "chat.messages" || parseRetentionResp.GetStatus() != "updated" {
		parseT.Fatalf("unexpected SetSuperuserDataRetentionPolicy response: %+v", parseRetentionResp)
	}

	parseComplianceResp, parseErr := parseServer.SetSuperuserComplianceControl(parseSuperuserCtx, &chatpb.SetSuperuserComplianceControlRequest{
		WorkspaceId:  parseWorkspaceID,
		ControlKey:   "access-review",
		FrameworkKey: "soc2",
		Status:       "implemented",
		OwnerUserId:  parseOwner.ID,
		EvidenceUrl:  "https://evidence.example.com/doc/123",
		ReviewedAt:   "2026-03-27T20:00:00Z",
		Confirm:      true,
		Reason:       "compliance update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserComplianceControl: %v", parseErr)
	}
	if parseComplianceResp.GetControl().GetControlKey() != "access-review" || parseComplianceResp.GetStatus() != "updated" {
		parseT.Fatalf("unexpected SetSuperuserComplianceControl response: %+v", parseComplianceResp)
	}

	parseSLOResp, parseErr := parseServer.SetSuperuserServiceLevelObjective(parseSuperuserCtx, &chatpb.SetSuperuserServiceLevelObjectiveRequest{
		SloKey:             "api-latency",
		ServiceName:        "API Latency",
		ObjectivePercent:   99.9,
		WindowDays:         30,
		ErrorBudgetMinutes: 43,
		StatusPageUrl:      "https://status.example.com",
		Confirm:            true,
		Reason:             "slo update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserServiceLevelObjective: %v", parseErr)
	}
	if parseSLOResp.GetSlo().GetSloKey() != "api-latency" || parseSLOResp.GetStatus() != "updated" {
		parseT.Fatalf("unexpected SetSuperuserServiceLevelObjective response: %+v", parseSLOResp)
	}

	parseIncidentResp, parseErr := parseServer.SetSuperuserIncident(parseSuperuserCtx, &chatpb.SetSuperuserIncidentRequest{
		IncidentKey: "incident-20260327-1",
		SloKey:      "api-latency",
		Severity:    "major",
		Status:      "open",
		Title:       "Regional API latency increase",
		Summary:     "Latency above objective in one region.",
		StartedAt:   "2026-03-27T19:00:00Z",
		Confirm:     true,
		Reason:      "incident created",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserIncident: %v", parseErr)
	}
	if parseIncidentResp.GetIncident().GetIncidentKey() != "incident-20260327-1" || parseIncidentResp.GetStatus() != "updated" {
		parseT.Fatalf("unexpected SetSuperuserIncident response: %+v", parseIncidentResp)
	}

	parseIncidentUpdateResp, parseErr := parseServer.SetSuperuserIncidentUpdate(parseSuperuserCtx, &chatpb.SetSuperuserIncidentUpdateRequest{
		IncidentId:      parseIncidentResp.GetIncident().GetId(),
		Status:          "investigating",
		Message:         "Incident declared.",
		IsPublic:        true,
		CreatedByUserId: parseOwner.ID,
		Confirm:         true,
		Reason:          "first update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserIncidentUpdate create: %v", parseErr)
	}
	if parseIncidentUpdateResp.GetIncidentUpdate().GetId() <= 0 || parseIncidentUpdateResp.GetStatus() != "created" {
		parseT.Fatalf("unexpected SetSuperuserIncidentUpdate create response: %+v", parseIncidentUpdateResp)
	}
	parseIncidentUpdateResp, parseErr = parseServer.SetSuperuserIncidentUpdate(parseSuperuserCtx, &chatpb.SetSuperuserIncidentUpdateRequest{
		Id:              parseIncidentUpdateResp.GetIncidentUpdate().GetId(),
		IncidentId:      parseIncidentResp.GetIncident().GetId(),
		Status:          "resolved",
		Message:         "Incident resolved.",
		IsPublic:        true,
		CreatedByUserId: parseOwner.ID,
		Confirm:         true,
		Reason:          "resolve update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserIncidentUpdate update: %v", parseErr)
	}
	if parseIncidentUpdateResp.GetIncidentUpdate().GetStatus() != "resolved" || parseIncidentUpdateResp.GetStatus() != "updated" {
		parseT.Fatalf("unexpected SetSuperuserIncidentUpdate update response: %+v", parseIncidentUpdateResp)
	}

	if _, parseErr = parseServer.DeleteSuperuserIncidentUpdate(parseSuperuserCtx, &chatpb.DeleteSuperuserIncidentUpdateRequest{
		Id:      parseIncidentUpdateResp.GetIncidentUpdate().GetId(),
		Confirm: true,
		Reason:  "cleanup incident update",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserIncidentUpdate: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserIncident(parseSuperuserCtx, &chatpb.DeleteSuperuserIncidentRequest{
		IncidentKey: "incident-20260327-1",
		Confirm:     true,
		Reason:      "cleanup incident",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserIncident: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserServiceLevelObjective(parseSuperuserCtx, &chatpb.DeleteSuperuserServiceLevelObjectiveRequest{
		SloKey:  "api-latency",
		Confirm: true,
		Reason:  "cleanup slo",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserServiceLevelObjective: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserComplianceControl(parseSuperuserCtx, &chatpb.DeleteSuperuserComplianceControlRequest{
		WorkspaceId:  parseWorkspaceID,
		ControlKey:   "access-review",
		FrameworkKey: "soc2",
		Confirm:      true,
		Reason:       "cleanup compliance",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserComplianceControl: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserDataRetentionPolicy(parseSuperuserCtx, &chatpb.DeleteSuperuserDataRetentionPolicyRequest{
		WorkspaceId: parseWorkspaceID,
		ScopeKey:    "chat.messages",
		Confirm:     true,
		Reason:      "cleanup retention",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserDataRetentionPolicy: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserWorkspaceSSOConfig(parseSuperuserCtx, &chatpb.DeleteSuperuserWorkspaceSSOConfigRequest{
		WorkspaceId: parseWorkspaceID,
		ProviderKey: "okta",
		Confirm:     true,
		Reason:      "cleanup sso",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserWorkspaceSSOConfig: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserWorkspaceSSOConfig(parseSuperuserCtx, &chatpb.DeleteSuperuserWorkspaceSSOConfigRequest{
		WorkspaceId: parseWorkspaceID,
		ProviderKey: "oidc",
		Confirm:     true,
		Reason:      "cleanup oidc sso",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserWorkspaceSSOConfig(oidc): %v", parseErr)
	}

	parseNonSuperuser := parseMustCreateUser(parseT, parseStore, "non-su-reliability@example.com")
	parseNonSuperuserCtx := parseBindAuthUser(parseServer, "peer-non-superuser-reliability-rpc", parseNonSuperuser.ID, parseNonSuperuser.Email)
	if _, parseErr = parseServer.SetSuperuserWorkspaceSSOConfig(parseNonSuperuserCtx, &chatpb.SetSuperuserWorkspaceSSOConfigRequest{
		WorkspaceId: parseWorkspaceID,
		ProviderKey: "okta",
		Confirm:     true,
		Reason:      "should fail",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetSuperuserWorkspaceSSOConfig non-superuser status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetSuperuserWorkspaceSSOConfig(parseSuperuserCtx, &chatpb.SetSuperuserWorkspaceSSOConfigRequest{
		WorkspaceId: parseWorkspaceID,
		ProviderKey: "okta",
		Confirm:     false,
		Reason:      "missing confirm",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetSuperuserWorkspaceSSOConfig missing confirm status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// BenchmarkParseNormalizeSuperuserIncidentStatus reports micro-benchmark throughput for superuser incident-status normalization.
func BenchmarkParseNormalizeSuperuserIncidentStatus(parseB *testing.B) {
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseNormalizeSuperuserIncidentStatus("resolved")
	}
}
