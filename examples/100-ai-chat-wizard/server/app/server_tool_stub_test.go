package app

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeRunServerToolStream struct {
	parseCtx       context.Context
	parseRecvQueue []*chatpb.RunServerToolRequest
	parseRecvIndex int
	parseSent      []*chatpb.RunServerToolEvent
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
func (parseS *fakeRunServerToolStream) Send(parseEvent *chatpb.RunServerToolEvent) error {
	parseS.parseSent = append(parseS.parseSent, parseEvent)
	return nil
}

// Recv satisfies the typed bidi stream contract for tests.
func (parseS *fakeRunServerToolStream) Recv() (*chatpb.RunServerToolRequest, error) {
	if parseS.parseRecvIndex >= len(parseS.parseRecvQueue) {
		return nil, io.EOF
	}
	parseRequest := parseS.parseRecvQueue[parseS.parseRecvIndex]
	parseS.parseRecvIndex++
	return parseRequest, nil
}

// SendMsg satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) SendMsg(any) error {
	return nil
}

// RecvMsg satisfies the grpc stream contract for tests.
func (parseS *fakeRunServerToolStream) RecvMsg(any) error {
	return nil
}

// parseBuildStaleSessionToken re-signs one valid session token with one synthetic issued-at timestamp for freshness tests.
func parseBuildStaleSessionToken(parseT *testing.T, parseAuth *authManager, parseFreshToken string, parseIssuedAt time.Time) string {
	parseT.Helper()
	parseClaims, parseErr := parseAuth.parseValidateSignedTokenClaims(parseFreshToken)
	if parseErr != nil {
		parseT.Fatalf("parseValidateSignedTokenClaims: %v", parseErr)
	}
	parseRegisteredClaims := parseClaims.RegisteredClaims
	parseRegisteredClaims.IssuedAt = jwt.NewNumericDate(parseIssuedAt.UTC())
	parseToken := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
		UserID:           parseClaims.UserID,
		Email:            parseClaims.Email,
		SessionID:        parseClaims.SessionID,
		TokenVersion:     parseClaims.TokenVersion,
		AuthMethod:       parseClaims.AuthMethod,
		RegisteredClaims: parseRegisteredClaims,
	})
	if parseAuth.signingKeyID != "" {
		parseToken.Header["kid"] = parseAuth.signingKeyID
	}
	parseStaleToken, parseErr := parseToken.SignedString(parseAuth.secret)
	if parseErr != nil {
		parseT.Fatalf("SignedString stale token: %v", parseErr)
	}
	return parseStaleToken
}

// parseBuildServerToolPolicyMutationRequest builds one valid server-tool policy update request.
func parseBuildServerToolPolicyMutationRequest() *chatpb.SetServerToolPolicyRequest {
	return &chatpb.SetServerToolPolicyRequest{
		IsEnabled:         true,
		MaxSessionSeconds: 180,
		MaxOutputBytes:    8192,
		ApprovedTools: []*chatpb.ServerToolPolicyRule{
			{
				ToolId:           "go-version",
				Description:      "Run go version",
				IsEnabled:        true,
				Shell:            "powershell",
				ArgvPrefix:       []string{"go", "version"},
				AllowArgsRegex:   []string{"^$"},
				DenyArgsRegex:    []string{"^--race$"},
				AllowCwdPrefixes: []string{"."},
			},
		},
	}
}

// parseBuildServerToolStartFailurePolicyMutationRequest builds one valid policy that authorizes a command guaranteed to fail at process start.
func parseBuildServerToolStartFailurePolicyMutationRequest() *chatpb.SetServerToolPolicyRequest {
	return &chatpb.SetServerToolPolicyRequest{
		IsEnabled:         true,
		MaxSessionSeconds: 30,
		MaxOutputBytes:    4096,
		ApprovedTools: []*chatpb.ServerToolPolicyRule{
			{
				ToolId:           "missing-command",
				Description:      "Intentionally missing command for start-failure regression",
				IsEnabled:        true,
				Shell:            "powershell",
				ArgvPrefix:       []string{"definitely-not-a-real-command"},
				AllowArgsRegex:   []string{"^$"},
				DenyArgsRegex:    []string{"^--forbidden$"},
				AllowCwdPrefixes: []string{"."},
			},
		},
	}
}

// parseBuildRunServerToolStartRequest builds one run-server-tool start request.
func parseBuildRunServerToolStartRequest(parseCommand string) *chatpb.RunServerToolRequest {
	return &chatpb.RunServerToolRequest{
		Payload: &chatpb.RunServerToolRequest_Start{
			Start: &chatpb.RunServerToolStart{
				SessionId:      "session-1",
				Shell:          "powershell",
				Command:        parseCommand,
				Cwd:            ".",
				TimeoutSeconds: 30,
			},
		},
	}
}

// parseBuildDangerousConfirmContext wraps one context with explicit dangerous-change confirmation metadata.
func parseBuildDangerousConfirmContext(parseCtx context.Context) context.Context {
	return metadata.NewIncomingContext(parseCtx, metadata.Pairs(parseServerToolDangerousConfirmMetadataKey, "true"))
}

// parseHasAuditEventType reports whether one audit row set contains one expected event type.
func parseHasAuditEventType(parseRows []parseAuditLogRow, parseEventType string) bool {
	for _, parseRow := range parseRows {
		if parseRow.EventType == parseEventType {
			return true
		}
	}
	return false
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

// TestSetServerToolPolicyRequiresDangerousChangeConfirmation verifies dangerous policy changes need explicit confirmation metadata.
func TestSetServerToolPolicyRequiresDangerousChangeConfirmation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-policy-confirm", parseOwner.ID, parseOwner.Email)

	parseReq := parseBuildServerToolPolicyMutationRequest()
	if _, parseErr := parseServer.SetServerToolPolicy(parseCtx, parseReq); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument without dangerous confirmation, got %v", status.Code(parseErr))
	}

	parseConfirmCtx := parseBuildDangerousConfirmContext(parseCtx)
	parseSetResp, parseErr := parseServer.SetServerToolPolicy(parseConfirmCtx, parseReq)
	if parseErr != nil {
		parseT.Fatalf("SetServerToolPolicy confirmed: %v", parseErr)
	}
	if !parseSetResp.GetIsApplied() {
		parseT.Fatalf("expected applied response, got %+v", parseSetResp)
	}
	if parseSetResp.GetUpdatedAt() == "" {
		parseT.Fatalf("expected applied updated_at timestamp, got empty value")
	}
	parsePolicyResp, parseErr := parseServer.GetServerToolPolicy(parseCtx, &chatpb.GetServerToolPolicyRequest{})
	if parseErr != nil {
		parseT.Fatalf("GetServerToolPolicy after set: %v", parseErr)
	}
	if !parsePolicyResp.GetIsEnabled() {
		parseT.Fatalf("expected enabled policy after confirmed set")
	}
	if len(parsePolicyResp.GetApprovedTools()) != 1 {
		parseT.Fatalf("expected one approved tool after confirmed set, got %d", len(parsePolicyResp.GetApprovedTools()))
	}

	parseAuditRows, parseErr := parseStore.parseListAuditLogs(200)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs: %v", parseErr)
	}
	if !parseHasAuditEventType(parseAuditRows, "admin.superuser.server_tool_policy.denied") {
		parseT.Fatalf("expected denied policy audit event, rows=%+v", parseAuditRows)
	}
	if !parseHasAuditEventType(parseAuditRows, "admin.superuser.server_tool_policy.set") {
		parseT.Fatalf("expected set policy audit event, rows=%+v", parseAuditRows)
	}
}

// TestSetServerToolPolicyValidatesRuleArgumentPolicy verifies enabled command rules require argument policy regexes.
func TestSetServerToolPolicyValidatesRuleArgumentPolicy(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBuildDangerousConfirmContext(parseBindAuthUser(parseServer, "peer-policy-arg-policy", parseOwner.ID, parseOwner.Email))

	parseReq := &chatpb.SetServerToolPolicyRequest{
		IsEnabled: true,
		ApprovedTools: []*chatpb.ServerToolPolicyRule{
			{
				ToolId:     "gwc-test",
				IsEnabled:  true,
				Shell:      "powershell",
				ArgvPrefix: []string{"go", "run", "./tools/gwc", "test"},
			},
		},
	}
	if _, parseErr := parseServer.SetServerToolPolicy(parseCtx, parseReq); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument when argument policy regex missing, got %v", status.Code(parseErr))
	}
}

// TestRunServerToolRequiresStartPayload verifies one start frame is required before policy evaluation.
func TestRunServerToolRequiresStartPayload(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-tool-start-required", parseOwner.ID, parseOwner.Email)

	parseStream := &fakeRunServerToolStream{parseCtx: parseCtx}
	parseErr := parseServer.RunServerTool(parseStream)
	if status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for missing start payload, got %v", status.Code(parseErr))
	}
}

// TestRunServerToolEnforcesWhitelistAndArgumentPolicy verifies command/rule mismatches fail closed with one policy-denied event.
func TestRunServerToolEnforcesWhitelistAndArgumentPolicy(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-tool-policy-denied", parseOwner.ID, parseOwner.Email)
	parseConfirmCtx := parseBuildDangerousConfirmContext(parseCtx)

	if _, parseErr := parseServer.SetServerToolPolicy(parseConfirmCtx, parseBuildServerToolPolicyMutationRequest()); parseErr != nil {
		parseT.Fatalf("SetServerToolPolicy: %v", parseErr)
	}

	parseStream := &fakeRunServerToolStream{
		parseCtx: parseCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("go version --race"),
		},
	}
	parseErr := parseServer.RunServerTool(parseStream)
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected permission denied for deny-list arg, got %v", status.Code(parseErr))
	}
	if len(parseStream.parseSent) != 1 || parseStream.parseSent[0].GetError() == nil {
		parseT.Fatalf("expected one policy-denied error event, got %+v", parseStream.parseSent)
	}
	if parseStream.parseSent[0].GetError().GetCode() != "policy_denied" {
		parseT.Fatalf("expected policy_denied event code, got %+v", parseStream.parseSent[0].GetError())
	}
}

// TestRunServerToolStartFailureEmitsOneRuntimeError verifies start failures surface one operator-visible error event instead of silent stream hangs.
func TestRunServerToolStartFailureEmitsOneRuntimeError(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-tool-start-failure", parseOwner.ID, parseOwner.Email)
	parseConfirmCtx := parseBuildDangerousConfirmContext(parseCtx)

	if _, parseErr := parseServer.SetServerToolPolicy(parseConfirmCtx, parseBuildServerToolStartFailurePolicyMutationRequest()); parseErr != nil {
		parseT.Fatalf("SetServerToolPolicy: %v", parseErr)
	}

	parseStream := &fakeRunServerToolStream{
		parseCtx: parseCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("definitely-not-a-real-command"),
		},
	}
	parseErr := parseServer.RunServerTool(parseStream)
	if status.Code(parseErr) != codes.FailedPrecondition {
		parseT.Fatalf("expected failed precondition for process start failure, got %v", status.Code(parseErr))
	}
	if len(parseStream.parseSent) != 1 {
		parseT.Fatalf("expected exactly one runtime error event, got %+v", parseStream.parseSent)
	}
	parseErrorEvent := parseStream.parseSent[0].GetError()
	if parseErrorEvent == nil {
		parseT.Fatalf("expected one error event payload, got %+v", parseStream.parseSent[0])
	}
	if parseErrorEvent.GetCode() != "start_failed" {
		parseT.Fatalf("expected start_failed code, got %+v", parseErrorEvent)
	}
	if parseErrorEvent.GetMessage() == "" {
		parseT.Fatalf("expected non-empty start failure message, got %+v", parseErrorEvent)
	}
}

// TestRunServerToolAllowedCommandStreamsStartedAndExit verifies allowed starts emit started/output/exit events and complete.
func TestRunServerToolAllowedCommandStreamsStartedAndExit(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-tool-runtime-unavailable", parseOwner.ID, parseOwner.Email)
	parseConfirmCtx := parseBuildDangerousConfirmContext(parseCtx)

	if _, parseErr := parseServer.SetServerToolPolicy(parseConfirmCtx, parseBuildServerToolPolicyMutationRequest()); parseErr != nil {
		parseT.Fatalf("SetServerToolPolicy: %v", parseErr)
	}

	parseStream := &fakeRunServerToolStream{
		parseCtx: parseCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("go version"),
		},
	}
	parseErr := parseServer.RunServerTool(parseStream)
	if parseErr != nil {
		parseT.Fatalf("expected allowed command execution success, got %v", parseErr)
	}
	if len(parseStream.parseSent) < 2 {
		parseT.Fatalf("expected started+exit events, got %+v", parseStream.parseSent)
	}
	if parseStream.parseSent[0].GetStarted() == nil {
		parseT.Fatalf("expected first event to be started, got %+v", parseStream.parseSent[0])
	}
	parseLastEvent := parseStream.parseSent[len(parseStream.parseSent)-1]
	if parseLastEvent.GetExit() == nil {
		parseT.Fatalf("expected terminal exit event, got %+v", parseLastEvent)
	}
	if parseLastEvent.GetExit().GetExitCode() != 0 {
		parseT.Fatalf("expected zero exit code, got %+v", parseLastEvent.GetExit())
	}
}

// TestRunServerToolStdinRequiresMatchingSessionID verifies stdin frames fail closed when session ids do not match the active session.
func TestRunServerToolStdinRequiresMatchingSessionID(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-tool-stdin-session-id", parseOwner.ID, parseOwner.Email)
	parseConfirmCtx := parseBuildDangerousConfirmContext(parseCtx)

	if _, parseErr := parseServer.SetServerToolPolicy(parseConfirmCtx, parseBuildServerToolPolicyMutationRequest()); parseErr != nil {
		parseT.Fatalf("SetServerToolPolicy: %v", parseErr)
	}
	parseStream := &fakeRunServerToolStream{
		parseCtx: parseCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("go version"),
			{
				Payload: &chatpb.RunServerToolRequest_Stdin{
					Stdin: &chatpb.RunServerToolStdin{
						SessionId: "other-session",
						Chunk:     []byte("x"),
					},
				},
			},
		},
	}
	if parseErr := parseServer.RunServerTool(parseStream); parseErr != nil {
		parseT.Fatalf("expected stream termination without rpc error, got %v", parseErr)
	}
	isParseHasStdinError := false
	for _, parseEvent := range parseStream.parseSent {
		if parseEvent.GetError() != nil && parseEvent.GetError().GetCode() == "stdin_failed" {
			isParseHasStdinError = true
		}
	}
	if !isParseHasStdinError {
		parseT.Fatalf("expected stdin_failed event for mismatched session id, got %+v", parseStream.parseSent)
	}
}

// TestSensitiveSuperuserMutationsRequireFreshSession verifies stale superuser sessions must re-authenticate before sensitive mutations.
func TestSensitiveSuperuserMutationsRequireFreshSession(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSignupResp, parseErr := parseServer.Signup(context.Background(), &chatpb.SignupRequest{
		Email:       "superuser-fresh-session@example.com",
		Password:    "password123",
		DisplayName: "Fresh Session Owner",
	})
	if parseErr != nil {
		parseT.Fatalf("Signup: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSignupResp.GetUserId())

	parseLoginResp, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "superuser-fresh-session@example.com",
		Password: "password123",
	})
	if parseErr != nil {
		parseT.Fatalf("Login: %v", parseErr)
	}
	parseFreshCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseLoginResp.GetAuthToken()))
	if _, parseErr = parseServer.SetServerToolPolicy(parseFreshCtx, &chatpb.SetServerToolPolicyRequest{}); status.Code(parseErr) == codes.Unauthenticated {
		parseT.Fatalf("expected fresh-session SetServerToolPolicy to avoid unauthenticated, got %v", status.Code(parseErr))
	}
	if parseErr = parseServer.RunServerTool(&fakeRunServerToolStream{
		parseCtx: parseFreshCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("go run ./tools/gwc test -lane unit"),
		},
	}); status.Code(parseErr) == codes.Unauthenticated {
		parseT.Fatalf("expected fresh-session RunServerTool to avoid unauthenticated, got %v", status.Code(parseErr))
	}
	parseExternalToken, parseErr := parseServer.authManager.issueTokenForContextWithAuthMethod(
		context.Background(),
		authUser{ID: parseSignupResp.GetUserId(), Email: "superuser-fresh-session@example.com"},
		"",
		parseWorkspaceAuthMethodGoogleOIDC,
	)
	if parseErr != nil {
		parseT.Fatalf("issueTokenForContextWithAuthMethod external: %v", parseErr)
	}
	parseExternalCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseExternalToken))
	if _, parseErr = parseServer.SetServerToolPolicy(parseExternalCtx, &chatpb.SetServerToolPolicyRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected external-auth SetServerToolPolicy to require password re-authentication, got %v", status.Code(parseErr))
	}
	if parseErr = parseServer.RunServerTool(&fakeRunServerToolStream{
		parseCtx: parseExternalCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("go run ./tools/gwc test -lane unit"),
		},
	}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected external-auth RunServerTool to require password re-authentication, got %v", status.Code(parseErr))
	}

	parseStaleToken := parseBuildStaleSessionToken(parseT, parseServer.authManager, parseLoginResp.GetAuthToken(), time.Now().UTC().Add(-superuserMutationSessionMaxAge-time.Minute))
	parseStaleCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseStaleToken))
	if _, parseErr = parseServer.SetServerToolPolicy(parseStaleCtx, &chatpb.SetServerToolPolicyRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected stale-session SetServerToolPolicy to require re-authentication, got %v", status.Code(parseErr))
	}
	if parseErr = parseServer.RunServerTool(&fakeRunServerToolStream{
		parseCtx: parseStaleCtx,
		parseRecvQueue: []*chatpb.RunServerToolRequest{
			parseBuildRunServerToolStartRequest("go run ./tools/gwc test -lane unit"),
		},
	}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected stale-session RunServerTool to require re-authentication, got %v", status.Code(parseErr))
	}
}

// TestRunServerToolRequiresStream verifies nil stream guard rails on the server-tool entrypoint.
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

// BenchmarkAuthorizeRunServerToolStartAllowed measures policy-authorization cost for one allowed command start.
func BenchmarkAuthorizeRunServerToolStartAllowed(parseB *testing.B) {
	parsePolicyReq := parseBuildServerToolPolicyMutationRequest()
	parsePolicy := &chatpb.GetServerToolPolicyResponse{
		IsEnabled:         true,
		MaxSessionSeconds: parsePolicyReq.GetMaxSessionSeconds(),
		MaxOutputBytes:    parsePolicyReq.GetMaxOutputBytes(),
		ApprovedTools:     parsePolicyReq.GetApprovedTools(),
	}
	parseStart := &chatpb.RunServerToolStart{
		SessionId:      "bench-allowed",
		Shell:          "powershell",
		Command:        "go run ./tools/gwc test -lane unit",
		Cwd:            ".",
		TimeoutSeconds: 30,
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if parseErr := parseAuthorizeRunServerToolStart(parsePolicy, parseStart); parseErr != nil {
			parseB.Fatalf("parseAuthorizeRunServerToolStart: %v", parseErr)
		}
	}
}

// BenchmarkAuthorizeRunServerToolStartDenied measures policy-authorization cost for one denied command start.
func BenchmarkAuthorizeRunServerToolStartDenied(parseB *testing.B) {
	parsePolicyReq := parseBuildServerToolPolicyMutationRequest()
	parsePolicy := &chatpb.GetServerToolPolicyResponse{
		IsEnabled:         true,
		MaxSessionSeconds: parsePolicyReq.GetMaxSessionSeconds(),
		MaxOutputBytes:    parsePolicyReq.GetMaxOutputBytes(),
		ApprovedTools:     parsePolicyReq.GetApprovedTools(),
	}
	parseStart := &chatpb.RunServerToolStart{
		SessionId:      "bench-denied",
		Shell:          "powershell",
		Command:        "go run ./tools/gwc test --race",
		Cwd:            ".",
		TimeoutSeconds: 30,
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if parseErr := parseAuthorizeRunServerToolStart(parsePolicy, parseStart); status.Code(parseErr) != codes.PermissionDenied {
			parseB.Fatalf("parseAuthorizeRunServerToolStart code=%v", status.Code(parseErr))
		}
	}
}
