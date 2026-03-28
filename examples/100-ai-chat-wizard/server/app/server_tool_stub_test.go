package app

import (
	"context"
	"io"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeRunServerToolStream struct {
	parseCtx context.Context
}

// SetHeader satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) SetHeader(metadata.MD) error {
	return nil
}

// SendHeader satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) SendHeader(metadata.MD) error {
	return nil
}

// SetTrailer satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) SetTrailer(metadata.MD) {}

// Context returns the configured stream context for tests.
func (parseS *fakeRunServerToolStream) Context() context.Context {
	if parseS.parseCtx == nil {
		return context.Background()
	}
	return parseS.parseCtx
}

// Send satisfies the typed bidi stream contract for tests.
func (parseS *fakeRunServerToolStream) Send(*chatpb.RunServerToolEvent) error {
	return nil
}

// Recv satisfies the typed bidi stream contract for tests.
func (parseS *fakeRunServerToolStream) Recv() (*chatpb.RunServerToolRequest, error) {
	return nil, io.EOF
}

// SendMsg satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) SendMsg(any) error {
	return nil
}

// RecvMsg satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) RecvMsg(any) error {
	return nil
}

// TestGetServerToolPolicyRequiresSURole verifies the policy endpoint remains superuser-gated.
func TestGetServerToolPolicyRequiresSURole(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "policy-plain@example.com")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-policy-denied", parseUser.ID, parseUser.Email)

	_, parseErr := parseServer.GetServerToolPolicy(parseCtx, &chatpb.GetServerToolPolicyRequest{})
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected permission denied, got %v", status.Code(parseErr))
	}
}

// TestGetServerToolPolicyReturnsDefaults verifies fallback policy values when no command config is stored yet.
func TestGetServerToolPolicyReturnsDefaults(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-policy-allowed", parseOwner.ID, parseOwner.Email)

	parseResp, parseErr := parseServer.GetServerToolPolicy(parseCtx, &chatpb.GetServerToolPolicyRequest{})
	if parseErr != nil {
		parseT.Fatalf("GetServerToolPolicy: %v", parseErr)
	}
	if parseResp.GetSource() != "defaults" {
		parseT.Fatalf("expected defaults source, got %q", parseResp.GetSource())
	}
	if parseResp.GetIsEnabled() {
		parseT.Fatalf("expected disabled policy, got enabled")
	}
}

// TestSetServerToolPolicyReturnsUnimplemented verifies update wiring is intentionally stubbed.
func TestSetServerToolPolicyReturnsUnimplemented(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-policy-set", parseOwner.ID, parseOwner.Email)

	_, parseErr := parseServer.SetServerToolPolicy(parseCtx, &chatpb.SetServerToolPolicyRequest{})
	if status.Code(parseErr) != codes.Unimplemented {
		parseT.Fatalf("expected unimplemented, got %v", status.Code(parseErr))
	}
}

// TestRunServerToolReturnsUnimplemented verifies terminal execution remains explicitly stubbed.
func TestRunServerToolReturnsUnimplemented(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-tool-run", parseOwner.ID, parseOwner.Email)

	parseErr := parseServer.RunServerTool(&fakeRunServerToolStream{parseCtx: parseCtx})
	if status.Code(parseErr) != codes.Unimplemented {
		parseT.Fatalf("expected unimplemented, got %v", status.Code(parseErr))
	}
}

// TestRunServerToolRequiresStream verifies nil stream guard rails on the stub entrypoint.
func TestRunServerToolRequiresStream(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseErr := parseServer.RunServerTool(nil)
	if status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument, got %v", status.Code(parseErr))
	}
}

// TestGetServerToolPolicyReadsWhitelistFromSiteConfig verifies whitelist config reads from site_config JSON.
func TestGetServerToolPolicyReadsWhitelistFromSiteConfig(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-policy-configured", parseOwner.ID, parseOwner.Email)

	if parseErr := parseStore.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigEnabledKey,
		ConfigValue:     "true",
		ValueType:       "bool",
		Description:     "Enable server tools",
		UpdatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSiteConfig enabled: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigMaxSessionSecondsKey,
		ConfigValue:     "120",
		ValueType:       "int",
		Description:     "Session timeout",
		UpdatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSiteConfig max session: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigMaxOutputBytesKey,
		ConfigValue:     "8192",
		ValueType:       "int",
		Description:     "Output cap",
		UpdatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSiteConfig max output: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey: serverToolPolicyConfigWhitelistKey,
		ConfigValue: `{"approved_tools":[{"tool_id":"gwc-test","description":"Run gwc tests","is_enabled":true,"shell":"powershell","argv_prefix":["go","run","./tools/gwc","test"],` +
			`"allow_args_regex":["^-lane$","^(unit|wasm)$"],"deny_args_regex":["--race"],"allow_cwd_prefixes":["."]}]}`,
		ValueType:       "json",
		Description:     "Server tool whitelist",
		UpdatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSiteConfig whitelist: %v", parseErr)
	}

	parseResp, parseErr := parseServer.GetServerToolPolicy(parseCtx, &chatpb.GetServerToolPolicyRequest{})
	if parseErr != nil {
		parseT.Fatalf("GetServerToolPolicy: %v", parseErr)
	}
	if parseResp.GetSource() != "site_config" {
		parseT.Fatalf("expected site_config source, got %q", parseResp.GetSource())
	}
	if !parseResp.GetIsEnabled() {
		parseT.Fatalf("expected enabled policy, got disabled")
	}
	if parseResp.GetMaxSessionSeconds() != 120 {
		parseT.Fatalf("expected max session 120, got %d", parseResp.GetMaxSessionSeconds())
	}
	if parseResp.GetMaxOutputBytes() != 8192 {
		parseT.Fatalf("expected max output 8192, got %d", parseResp.GetMaxOutputBytes())
	}
	if len(parseResp.GetApprovedTools()) != 1 {
		parseT.Fatalf("expected one approved tool, got %d", len(parseResp.GetApprovedTools()))
	}
	parseRule := parseResp.GetApprovedTools()[0]
	if parseRule.GetToolId() != "gwc-test" {
		parseT.Fatalf("expected gwc-test tool id, got %q", parseRule.GetToolId())
	}
	if parseRule.GetShell() != "powershell" {
		parseT.Fatalf("expected powershell shell, got %q", parseRule.GetShell())
	}
}
