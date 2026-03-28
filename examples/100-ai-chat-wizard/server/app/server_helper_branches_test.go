package app

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthAndStoreGuardHelperBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("helper@example.com", "password123", "Helper")
	if parseErr != nil {
		parseT.Fatalf("signup: %v", parseErr)
	}
	parseToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}

	parseMdContext := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		authMetadataKey, "Bearer "+parseToken,
		authMetadataKey, "   ",
	))
	if parseResolved, parseOk := parseAuth.parseAuthenticatedUserFromContext(parseMdContext); !parseOk || parseResolved.ID != parseUser.ID {
		parseT.Fatalf("expected authenticated user from grpc metadata, got ok=%v user=%+v", parseOk, parseResolved)
	}

	if _, parseOk2 := parseAuth.parseAuthenticatedUserFromContext(metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer   "))); parseOk2 {
		parseT.Fatal("expected blank bearer metadata token to be rejected")
	}
	if _, parseOk3 := parseAuth.parseAuthenticatedUserFromRequest(httptest.NewRequest("GET", "http://example.com/", nil)); parseOk3 {
		parseT.Fatal("expected missing auth cookie to be rejected")
	}

	if parseTokenValue, parseOk4 := parseAuthTokenFromAuthorizationValue("  bearer " + parseToken + "  "); !parseOk4 || parseTokenValue != parseToken {
		parseT.Fatalf("unexpected authorization token parsing: ok=%v token=%q", parseOk4, parseTokenValue)
	}
	if parseTokenValue2, parseOk5 := parseAuthTokenFromAuthorizationValue(parseToken); !parseOk5 || parseTokenValue2 != parseToken {
		parseT.Fatalf("unexpected raw authorization token parsing: ok=%v token=%q", parseOk5, parseTokenValue2)
	}
	if _, parseOk6 := parseAuthTokenFromAuthorizationValue("   "); parseOk6 {
		parseT.Fatal("expected blank authorization value to be rejected")
	}

	parseMissingUser := authUser{ID: 999999, Email: "missing@example.com"}
	if _, parseOk7 := parseAuth.parseValidateActiveUser(parseMissingUser, "test"); parseOk7 {
		parseT.Fatal("expected missing stored user to be rejected")
	}

	parseNoStoreAuth := parseNewAuthManager("test-secret", nil, parseNewTestLogger())
	if parseResolved2, parseOk8 := parseNoStoreAuth.parseValidateActiveUser(authUser{ID: parseUser.ID, Email: parseUser.Email}, "test"); !parseOk8 || parseResolved2.ID != parseUser.ID {
		parseT.Fatalf("expected nil-store auth manager to accept positive user id, got ok=%v user=%+v", parseOk8, parseResolved2)
	}
	if _, parseOk9 := parseNoStoreAuth.parseValidateActiveUser(authUser{}, "test"); parseOk9 {
		parseT.Fatal("expected zero-value user to be rejected")
	}

	if parseCode := status.Code(parseStatusForStoreGuard(errStoreUserMissing)); parseCode != codes.Unauthenticated {
		parseT.Fatalf("expected user-missing guard to map to Unauthenticated, got %v", parseCode)
	}
	if parseCode2 := status.Code(parseStatusForStoreGuard(errStoreConversationMissing)); parseCode2 != codes.FailedPrecondition {
		parseT.Fatalf("expected conversation-missing guard to map to FailedPrecondition, got %v", parseCode2)
	}
	if parseCode3 := status.Code(parseStatusForStoreGuard(errStoreUserDisabled)); parseCode3 != codes.PermissionDenied {
		parseT.Fatalf("expected user-disabled guard to map to PermissionDenied, got %v", parseCode3)
	}
	if parseCode4 := status.Code(parseStatusForStoreGuard(errStoreUserAuthBlocked)); parseCode4 != codes.PermissionDenied {
		parseT.Fatalf("expected user-auth-blocked guard to map to PermissionDenied, got %v", parseCode4)
	}
	if parseCode5 := status.Code(parseStatusForStoreGuard(errStoreWorkspaceSuspended)); parseCode5 != codes.PermissionDenied {
		parseT.Fatalf("expected workspace-suspended guard to map to PermissionDenied, got %v", parseCode5)
	}
	parsePlainErr := errors.New("plain error")
	if !errors.Is(parseStatusForStoreGuard(parsePlainErr), parsePlainErr) {
		parseT.Fatalf("expected non-guard error to pass through unchanged")
	}
	if !isStoreGuardError(errStoreUserMissing) ||
		!isStoreGuardError(errStoreConversationMissing) ||
		!isStoreGuardError(errStoreUserDisabled) ||
		!isStoreGuardError(errStoreUserAuthBlocked) ||
		!isStoreGuardError(errStoreWorkspaceSuspended) {
		parseT.Fatal("expected store guard errors to be recognized")
	}
	if isStoreGuardError(parsePlainErr) {
		parseT.Fatal("expected plain error to not be recognized as a store guard")
	}

	parseServer := &chatServer{store: store, logger: parseNewTestLogger()}
	if parseGot := parseServer.parseDisplayNameForUser(parseUser.ID, parseUser.Email); parseGot != "Helper" {
		parseT.Fatalf("expected stored display name, got %q", parseGot)
	}
	if parseGot2 := parseServer.parseDisplayNameForUser(0, "fallback@example.com"); parseGot2 != "fallback" {
		parseT.Fatalf("expected fallback display name for zero user id, got %q", parseGot2)
	}
	if parseGot3 := (&chatServer{logger: parseNewTestLogger()}).parseDisplayNameForUser(parseUser.ID, "fallback@example.com"); parseGot3 != "fallback" {
		parseT.Fatalf("expected fallback display name without store, got %q", parseGot3)
	}

	parseChatReq := httptest.NewRequest("GET", "http://example.com/chat.wasm?br=true", nil)
	if parseRewritten := parseRewriteLegacyClientAssetRequest(parseChatReq); parseRewritten.URL.Path != "/app/chat.wasm" {
		parseT.Fatalf("expected legacy chat asset rewrite, got %q", parseRewritten.URL.Path)
	}
	parseWorkerReq := httptest.NewRequest("GET", "http://example.com/background-worker.wasm?br=true", nil)
	if parseRewritten2 := parseRewriteLegacyClientAssetRequest(parseWorkerReq); parseRewritten2.URL.Path != "/worker/background-worker.wasm" {
		parseT.Fatalf("expected legacy worker asset rewrite, got %q", parseRewritten2.URL.Path)
	}
	parseUnchangedReq := httptest.NewRequest("GET", "http://example.com/static/app.css", nil)
	if parseRewritten3 := parseRewriteLegacyClientAssetRequest(parseUnchangedReq); parseRewritten3 != parseUnchangedReq {
		parseT.Fatal("expected unrelated asset request to remain unchanged")
	}
}
