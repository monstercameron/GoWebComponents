package app

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthManagerSignupLoginAndTokenRoundTrip(t *testing.T) {
	store := newTestStore(t)
	auth := newAuthManager("test-secret", store, newTestLogger())

	user, err := auth.signup(" Test@Example.com ", "password123", "")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("normalized email mismatch: %q", user.Email)
	}

	record, err := store.getUserAuthByEmail("test@example.com")
	if err != nil {
		t.Fatalf("getUserAuthByEmail: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte("password123")) != nil {
		t.Fatal("stored password hash does not match original password")
	}

	loggedIn, err := auth.login("TEST@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedIn.ID != user.ID {
		t.Fatalf("login user mismatch: got %d want %d", loggedIn.ID, user.ID)
	}

	token, err := auth.issueToken(user)
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	parsed, err := auth.parseToken(token)
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	if parsed != user {
		t.Fatalf("parsed user mismatch: got %+v want %+v", parsed, user)
	}
}

func TestAuthManagerNegativePaths(t *testing.T) {
	store := newTestStore(t)
	auth := newAuthManager("test-secret", store, newTestLogger())
	devFallback := newAuthManager(" ", store, newTestLogger())
	noStoreAuth := newAuthManager("test-secret", nil, newTestLogger())
	if len(devFallback.secret) == 0 {
		t.Fatal("expected fallback auth secret to be populated")
	}
	if defaultDisplayNameFromEmail("   ") != "User" {
		t.Fatal("expected blank email to fall back to default display name")
	}
	if _, err := noStoreAuth.signup("nostore@example.com", "password123", ""); err == nil {
		t.Fatal("expected signup to fail when store is unavailable")
	}
	if _, err := noStoreAuth.login("nostore@example.com", "password123"); err == nil {
		t.Fatal("expected login to fail when store is unavailable")
	}

	if _, err := auth.signup("", "password123", ""); err == nil {
		t.Fatal("expected signup to reject missing email")
	}
	if _, err := auth.signup("a@example.com", "short", ""); err == nil {
		t.Fatal("expected signup to reject short password")
	}

	if _, err := auth.signup("user@example.com", "password123", "User"); err != nil {
		t.Fatalf("initial signup: %v", err)
	}
	if _, err := auth.signup("user@example.com", "password123", "User"); !errors.Is(err, errUserAlreadyExists) {
		t.Fatalf("expected duplicate signup to return errUserAlreadyExists, got %v", err)
	}

	if _, err := auth.login("user@example.com", "wrong-password"); !errors.Is(err, errInvalidCredentials) {
		t.Fatalf("expected invalid credentials for wrong password, got %v", err)
	}
	if _, err := auth.parseToken("not-a-jwt"); err == nil {
		t.Fatal("expected parseToken to reject invalid JWT")
	}
	nonHMACToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": 1, "email": "user@example.com"})
	tokenString, err := nonHMACToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString none token: %v", err)
	}
	if _, err := auth.parseToken(tokenString); err == nil {
		t.Fatal("expected parseToken to reject unexpected signing method")
	}
	if _, err := auth.parseToken(""); !errors.Is(err, errInvalidCredentials) {
		t.Fatalf("expected empty token to return invalid credentials, got %v", err)
	}
}

func TestRequestUsesHTTPSAndCookieHelpers(t *testing.T) {
	auth := newAuthManager("test-secret", newTestStore(t), newTestLogger())

	plainReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if requestUsesHTTPS(nil) {
		t.Fatal("nil request should not be treated as HTTPS")
	}
	if requestUsesHTTPS(plainReq) {
		t.Fatal("plain request should not be treated as HTTPS")
	}

	forwardedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	forwardedReq.Header.Set("X-Forwarded-Proto", "https")
	if !requestUsesHTTPS(forwardedReq) {
		t.Fatal("forwarded https request should be treated as HTTPS")
	}

	tlsReq := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	if !requestUsesHTTPS(tlsReq) {
		t.Fatal("TLS request should be treated as HTTPS")
	}

	writer := httptest.NewRecorder()
	auth.setAuthCookie(writer, forwardedReq, "token-value")
	resp := writer.Result()
	if len(resp.Cookies()) != 1 {
		t.Fatalf("expected one cookie, got %d", len(resp.Cookies()))
	}
	cookie := resp.Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.Value != "token-value" {
		t.Fatalf("unexpected auth cookie: %+v", cookie)
	}

	clearWriter := httptest.NewRecorder()
	auth.clearAuthCookie(clearWriter, plainReq)
	clearedCookie := clearWriter.Result().Cookies()[0]
	if clearedCookie.MaxAge != -1 {
		t.Fatalf("expected cleared cookie MaxAge -1, got %d", clearedCookie.MaxAge)
	}
	if clearedCookie.Value != "" {
		t.Fatalf("expected cleared cookie value to be empty, got %q", clearedCookie.Value)
	}
}

func TestAuthenticatedHandlers(t *testing.T) {
	store := newTestStore(t)
	auth := newAuthManager("test-secret", store, newTestLogger())
	user, err := auth.signup("demo@example.com", "password123", "Demo")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	token, err := auth.issueToken(user)
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}

	pageHandler := auth.requireAuthenticatedPage(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	unauthorizedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	unauthorizedWriter := httptest.NewRecorder()
	pageHandler.ServeHTTP(unauthorizedWriter, unauthorizedReq)
	if unauthorizedWriter.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect for unauthenticated page request, got %d", unauthorizedWriter.Code)
	}
	if location := unauthorizedWriter.Result().Header.Get("Location"); location != "/app" {
		t.Fatalf("unexpected redirect location: %q", location)
	}

	authorizedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	authorizedReq.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	authorizedWriter := httptest.NewRecorder()
	pageHandler.ServeHTTP(authorizedWriter, authorizedReq)
	if authorizedWriter.Code != http.StatusNoContent {
		t.Fatalf("expected protected page handler to run, got %d", authorizedWriter.Code)
	}

	tunnelCalled := false
	callbackCalled := false
	tunnelHandler := auth.requireAuthenticatedTunnel(func(w http.ResponseWriter, r *http.Request) {
		tunnelCalled = true
		w.WriteHeader(http.StatusAccepted)
	}, func(r *http.Request, user authUser) {
		callbackCalled = user.ID == user.ID && r.URL.Path == "/grpc"
	})

	unauthorizedTunnelWriter := httptest.NewRecorder()
	auth.requireAuthenticatedTunnel(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)(unauthorizedTunnelWriter, httptest.NewRequest(http.MethodGet, "http://example.com/grpc", nil))
	if unauthorizedTunnelWriter.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized tunnel response, got %d", unauthorizedTunnelWriter.Code)
	}

	tunnelReq := httptest.NewRequest(http.MethodGet, "http://example.com/grpc", nil)
	tunnelReq.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	tunnelWriter := httptest.NewRecorder()
	tunnelHandler(tunnelWriter, tunnelReq)
	if !tunnelCalled {
		t.Fatal("expected authenticated tunnel handler to run")
	}
	if !callbackCalled {
		t.Fatal("expected authenticated tunnel callback to run")
	}
	if tunnelWriter.Code != http.StatusAccepted {
		t.Fatalf("expected authenticated tunnel response, got %d", tunnelWriter.Code)
	}
}
