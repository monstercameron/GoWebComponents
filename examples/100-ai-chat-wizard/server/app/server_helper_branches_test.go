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

func TestAuthAndStoreGuardHelperBranches(t *testing.T) {
	store := newTestStore(t)
	auth := newAuthManager("test-secret", store, newTestLogger())
	user, err := auth.signup("helper@example.com", "password123", "Helper")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	token, err := auth.issueToken(user)
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}

	mdContext := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		authMetadataKey, "Bearer "+token,
		authMetadataKey, "   ",
	))
	if resolved, ok := auth.authenticatedUserFromContext(mdContext); !ok || resolved.ID != user.ID {
		t.Fatalf("expected authenticated user from grpc metadata, got ok=%v user=%+v", ok, resolved)
	}

	if _, ok := auth.authenticatedUserFromContext(metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer   "))); ok {
		t.Fatal("expected blank bearer metadata token to be rejected")
	}
	if _, ok := auth.authenticatedUserFromRequest(httptest.NewRequest("GET", "http://example.com/", nil)); ok {
		t.Fatal("expected missing auth cookie to be rejected")
	}

	if tokenValue, ok := authTokenFromAuthorizationValue("  bearer "+token+"  "); !ok || tokenValue != token {
		t.Fatalf("unexpected authorization token parsing: ok=%v token=%q", ok, tokenValue)
	}
	if tokenValue, ok := authTokenFromAuthorizationValue(token); !ok || tokenValue != token {
		t.Fatalf("unexpected raw authorization token parsing: ok=%v token=%q", ok, tokenValue)
	}
	if _, ok := authTokenFromAuthorizationValue("   "); ok {
		t.Fatal("expected blank authorization value to be rejected")
	}

	missingUser := authUser{ID: 999999, Email: "missing@example.com"}
	if _, ok := auth.validateActiveUser(missingUser, "test"); ok {
		t.Fatal("expected missing stored user to be rejected")
	}

	noStoreAuth := newAuthManager("test-secret", nil, newTestLogger())
	if resolved, ok := noStoreAuth.validateActiveUser(authUser{ID: user.ID, Email: user.Email}, "test"); !ok || resolved.ID != user.ID {
		t.Fatalf("expected nil-store auth manager to accept positive user id, got ok=%v user=%+v", ok, resolved)
	}
	if _, ok := noStoreAuth.validateActiveUser(authUser{}, "test"); ok {
		t.Fatal("expected zero-value user to be rejected")
	}

	if code := status.Code(statusForStoreGuard(errStoreUserMissing)); code != codes.Unauthenticated {
		t.Fatalf("expected user-missing guard to map to Unauthenticated, got %v", code)
	}
	if code := status.Code(statusForStoreGuard(errStoreConversationMissing)); code != codes.FailedPrecondition {
		t.Fatalf("expected conversation-missing guard to map to FailedPrecondition, got %v", code)
	}
	plainErr := errors.New("plain error")
	if !errors.Is(statusForStoreGuard(plainErr), plainErr) {
		t.Fatalf("expected non-guard error to pass through unchanged")
	}
	if !isStoreGuardError(errStoreUserMissing) || !isStoreGuardError(errStoreConversationMissing) {
		t.Fatal("expected store guard errors to be recognized")
	}
	if isStoreGuardError(plainErr) {
		t.Fatal("expected plain error to not be recognized as a store guard")
	}

	server := &chatServer{store: store, logger: newTestLogger()}
	if got := server.displayNameForUser(user.ID, user.Email); got != "Helper" {
		t.Fatalf("expected stored display name, got %q", got)
	}
	if got := server.displayNameForUser(0, "fallback@example.com"); got != "fallback" {
		t.Fatalf("expected fallback display name for zero user id, got %q", got)
	}
	if got := (&chatServer{logger: newTestLogger()}).displayNameForUser(user.ID, "fallback@example.com"); got != "fallback" {
		t.Fatalf("expected fallback display name without store, got %q", got)
	}

	chatReq := httptest.NewRequest("GET", "http://example.com/chat.wasm?br=true", nil)
	if rewritten := rewriteLegacyClientAssetRequest(chatReq); rewritten.URL.Path != "/app/chat.wasm" {
		t.Fatalf("expected legacy chat asset rewrite, got %q", rewritten.URL.Path)
	}
	workerReq := httptest.NewRequest("GET", "http://example.com/background-worker.wasm?br=true", nil)
	if rewritten := rewriteLegacyClientAssetRequest(workerReq); rewritten.URL.Path != "/worker/background-worker.wasm" {
		t.Fatalf("expected legacy worker asset rewrite, got %q", rewritten.URL.Path)
	}
	unchangedReq := httptest.NewRequest("GET", "http://example.com/static/app.css", nil)
	if rewritten := rewriteLegacyClientAssetRequest(unchangedReq); rewritten != unchangedReq {
		t.Fatal("expected unrelated asset request to remain unchanged")
	}
}
