package app

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
)

func TestAuthManagerSignupLoginAndTokenRoundTrip(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())

	parseUser, parseErr := parseAuth.parseSignup(" Test@Example.com ", "password123", "")
	if parseErr != nil {
		parseT.Fatalf("signup: %v", parseErr)
	}
	if parseUser.Email != "test@example.com" {
		parseT.Fatalf("normalized email mismatch: %q", parseUser.Email)
	}

	parseRecord, parseErr := store.getUserAuthByEmail("test@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail: %v", parseErr)
	}
	if bcrypt.CompareHashAndPassword([]byte(parseRecord.PasswordHash), []byte("password123")) != nil {
		parseT.Fatal("stored password hash does not match original password")
	}

	parseLoggedIn, parseErr := parseAuth.parseLogin("TEST@example.com", "password123")
	if parseErr != nil {
		parseT.Fatalf("login: %v", parseErr)
	}
	if parseLoggedIn.ID != parseUser.ID {
		parseT.Fatalf("login user mismatch: got %d want %d", parseLoggedIn.ID, parseUser.ID)
	}

	parseToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}
	parseParsed, parseErr := parseAuth.parseToken(parseToken)
	if parseErr != nil {
		parseT.Fatalf("parseToken: %v", parseErr)
	}
	if parseParsed != parseUser {
		parseT.Fatalf("parsed user mismatch: got %+v want %+v", parseParsed, parseUser)
	}
}

func TestAuthManagerPersistsSessionAndRevocation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("session-user@example.com", "password123", "Session User")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}

	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-agent", "relaydesk-e2e/1.0",
		"x-forwarded-for", "203.0.113.10",
	))
	parseToken, parseErr := parseAuth.issueTokenForContext(parseCtx, parseUser, "")
	if parseErr != nil {
		parseT.Fatalf("issueTokenForContext: %v", parseErr)
	}

	parseParsedToken, parseErr := jwt.ParseWithClaims(parseToken, &authClaims{}, func(parseToken *jwt.Token) (interface{}, error) {
		return parseAuth.secret, nil
	})
	if parseErr != nil {
		parseT.Fatalf("jwt parse claims: %v", parseErr)
	}
	parseClaims, parseOk := parseParsedToken.Claims.(*authClaims)
	if !parseOk || !parseParsedToken.Valid {
		parseT.Fatalf("expected valid auth claims, got %#v", parseParsedToken.Claims)
	}
	if parseClaims.TokenVersion != 1 || parseClaims.SessionID == "" {
		parseT.Fatalf("expected session-backed claims, got %+v", parseClaims)
	}
	if parseClaims.ID != parseClaims.SessionID {
		parseT.Fatalf("expected jti to match session id, got jti=%q sid=%q", parseClaims.ID, parseClaims.SessionID)
	}

	parseSession, isParseFound, parseErr := parseStore.parseGetAuthSessionBySessionID(parseClaims.SessionID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthSessionBySessionID: %v", parseErr)
	}
	if !isParseFound {
		parseT.Fatalf("expected auth session row for %q", parseClaims.SessionID)
	}
	if parseSession.UserID != parseUser.ID || parseSession.TokenVersion != 1 {
		parseT.Fatalf("unexpected auth session row: %+v", parseSession)
	}
	if parseSession.UserAgent != "relaydesk-e2e/1.0" || parseSession.IPAddress != "203.0.113.10" {
		parseT.Fatalf("expected request metadata capture, got %+v", parseSession)
	}

	parseAuthCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseToken))
	if parseResolved, parseOk2 := parseAuth.parseAuthenticatedUserFromContext(parseAuthCtx); !parseOk2 || parseResolved.ID != parseUser.ID {
		parseT.Fatalf("expected authenticated user from context, got user=%+v ok=%v", parseResolved, parseOk2)
	}

	if parseErr2 := parseStore.parseRevokeAuthSession(parseClaims.SessionID); parseErr2 != nil {
		parseT.Fatalf("parseRevokeAuthSession: %v", parseErr2)
	}
	if _, parseOk2 := parseAuth.parseAuthenticatedUserFromContext(parseAuthCtx); parseOk2 {
		parseT.Fatal("expected revoked session to fail authentication")
	}
}

func TestAuthManagerTokenKidRotationPolicy(parseT *testing.T) {
	parseT.Setenv("CHAT_AUTH_SIGNING_KEYS", "legacy=legacy-secret,rotated=rotated-secret")
	parseT.Setenv("CHAT_AUTH_ACTIVE_KID", "rotated")

	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("legacy-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("kid-policy@example.com", "password123", "Kid Policy")
	if parseErr != nil {
		parseT.Fatalf("parseSignup: %v", parseErr)
	}
	parseIssuedToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}

	parseParser := jwt.Parser{}
	parseUnverifiedToken, _, parseErr := parseParser.ParseUnverified(parseIssuedToken, &authClaims{})
	if parseErr != nil {
		parseT.Fatalf("ParseUnverified: %v", parseErr)
	}
	parseIssuedTokenKeyID, parseOk := parseUnverifiedToken.Header["kid"].(string)
	if !parseOk || strings.TrimSpace(parseIssuedTokenKeyID) != "rotated" {
		parseT.Fatalf("expected issued token kid=rotated, got %#v", parseUnverifiedToken.Header["kid"])
	}

	parseParsedToken, parseErr := jwt.ParseWithClaims(parseIssuedToken, &authClaims{}, func(parseToken *jwt.Token) (interface{}, error) {
		return parseAuth.secret, nil
	})
	if parseErr != nil {
		parseT.Fatalf("ParseWithClaims issued token: %v", parseErr)
	}
	parseClaims, parseOk := parseParsedToken.Claims.(*authClaims)
	if !parseOk || !parseParsedToken.Valid {
		parseT.Fatalf("expected valid issued claims, got %#v", parseParsedToken.Claims)
	}

	parseLegacyToken := jwt.NewWithClaims(jwt.SigningMethodHS256, *parseClaims)
	delete(parseLegacyToken.Header, "kid")
	parseLegacyTokenString, parseErr := parseLegacyToken.SignedString(parseAuth.verifyKeys["legacy"])
	if parseErr != nil {
		parseT.Fatalf("SignedString legacy token: %v", parseErr)
	}
	if _, parseErr2 := parseAuth.parseToken(parseLegacyTokenString); parseErr2 != nil {
		parseT.Fatalf("expected kid-less legacy token to validate during rotation window, got %v", parseErr2)
	}

	parseUnknownKidToken := jwt.NewWithClaims(jwt.SigningMethodHS256, *parseClaims)
	parseUnknownKidToken.Header["kid"] = "missing"
	parseUnknownKidTokenString, parseErr := parseUnknownKidToken.SignedString([]byte("missing-secret"))
	if parseErr != nil {
		parseT.Fatalf("SignedString unknown kid token: %v", parseErr)
	}
	if _, parseErr2 := parseAuth.parseToken(parseUnknownKidTokenString); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected unknown kid token to fail with invalid credentials, got %v", parseErr2)
	}
}

func TestAuthManagerNegativePaths(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())
	parseDevFallback := parseNewAuthManager(" ", store, parseNewTestLogger())
	parseNoStoreAuth := parseNewAuthManager("test-secret", nil, parseNewTestLogger())
	if len(parseDevFallback.secret) == 0 {
		parseT.Fatal("expected fallback auth secret to be populated")
	}
	if parseDefaultDisplayNameFromEmail("   ") != "User" {
		parseT.Fatal("expected blank email to fall back to default display name")
	}
	if _, parseErr := parseNoStoreAuth.parseSignup("nostore@example.com", "password123", ""); parseErr == nil {
		parseT.Fatal("expected signup to fail when store is unavailable")
	}
	if _, parseErr2 := parseNoStoreAuth.parseLogin("nostore@example.com", "password123"); parseErr2 == nil {
		parseT.Fatal("expected login to fail when store is unavailable")
	}

	if _, parseErr3 := parseAuth.parseSignup("", "password123", ""); parseErr3 == nil {
		parseT.Fatal("expected signup to reject missing email")
	}
	if _, parseErr4 := parseAuth.parseSignup("a@example.com", "short", ""); parseErr4 == nil {
		parseT.Fatal("expected signup to reject short password")
	}

	if _, parseErr5 := parseAuth.parseSignup("user@example.com", "password123", "User"); parseErr5 != nil {
		parseT.Fatalf("initial signup: %v", parseErr5)
	}
	if _, parseErr6 := parseAuth.parseSignup("user@example.com", "password123", "User"); !errors.Is(parseErr6, errUserAlreadyExists) {
		parseT.Fatalf("expected duplicate signup to return errUserAlreadyExists, got %v", parseErr6)
	}

	if _, parseErr7 := parseAuth.parseLogin("user@example.com", "wrong-password"); !errors.Is(parseErr7, errInvalidCredentials) {
		parseT.Fatalf("expected invalid credentials for wrong password, got %v", parseErr7)
	}
	if _, parseErr8 := parseAuth.parseToken("not-a-jwt"); parseErr8 == nil {
		parseT.Fatal("expected parseToken to reject invalid JWT")
	}
	parseNonHMACToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": 1, "email": "user@example.com"})
	parseTokenString, parseErr9 := parseNonHMACToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if parseErr9 != nil {
		parseT.Fatalf("SignedString none token: %v", parseErr9)
	}
	if _, parseErr10 := parseAuth.parseToken(parseTokenString); parseErr10 == nil {
		parseT.Fatal("expected parseToken to reject unexpected signing method")
	}
	if _, parseErr11 := parseAuth.parseToken(""); !errors.Is(parseErr11, errInvalidCredentials) {
		parseT.Fatalf("expected empty token to return invalid credentials, got %v", parseErr11)
	}
}

func TestRequestUsesHTTPSAndCookieHelpers(parseT *testing.T) {
	parseAuth := parseNewAuthManager("test-secret", parseNewTestStore(parseT), parseNewTestLogger())

	parsePlainReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if parseRequestUsesHTTPS(nil) {
		parseT.Fatal("nil request should not be treated as HTTPS")
	}
	if parseRequestUsesHTTPS(parsePlainReq) {
		parseT.Fatal("plain request should not be treated as HTTPS")
	}

	parseForwardedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseForwardedReq.Header.Set("X-Forwarded-Proto", "https")
	if !parseRequestUsesHTTPS(parseForwardedReq) {
		parseT.Fatal("forwarded https request should be treated as HTTPS")
	}

	parseTlsReq := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	parseTlsReq.TLS = &tls.ConnectionState{}
	if !parseRequestUsesHTTPS(parseTlsReq) {
		parseT.Fatal("TLS request should be treated as HTTPS")
	}

	parseWriter := httptest.NewRecorder()
	parseAuth.setAuthCookie(parseWriter, parseForwardedReq, "token-value")
	parseResp := parseWriter.Result()
	if len(parseResp.Cookies()) != 1 {
		parseT.Fatalf("expected one cookie, got %d", len(parseResp.Cookies()))
	}
	parseCookie := parseResp.Cookies()[0]
	if !parseCookie.HttpOnly || !parseCookie.Secure || parseCookie.Value != "token-value" {
		parseT.Fatalf("unexpected auth cookie: %+v", parseCookie)
	}

	clearWriter := httptest.NewRecorder()
	parseAuth.clearAuthCookie(clearWriter, parsePlainReq)
	parseClearedCookie := clearWriter.Result().Cookies()[0]
	if parseClearedCookie.MaxAge != -1 {
		parseT.Fatalf("expected cleared cookie MaxAge -1, got %d", parseClearedCookie.MaxAge)
	}
	if parseClearedCookie.Value != "" {
		parseT.Fatalf("expected cleared cookie value to be empty, got %q", parseClearedCookie.Value)
	}
}

func TestAuthenticatedHandlers(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("demo@example.com", "password123", "Demo")
	if parseErr != nil {
		parseT.Fatalf("signup: %v", parseErr)
	}
	parseToken, parseErr := parseAuth.issueToken(parseUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken: %v", parseErr)
	}

	parsePageHandler := parseAuth.parseRequireAuthenticatedPage(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusNoContent)
	}))

	parseUnauthorizedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseUnauthorizedWriter := httptest.NewRecorder()
	parsePageHandler.ServeHTTP(parseUnauthorizedWriter, parseUnauthorizedReq)
	if parseUnauthorizedWriter.Code != http.StatusSeeOther {
		parseT.Fatalf("expected redirect for unauthenticated page request, got %d", parseUnauthorizedWriter.Code)
	}
	if parseLocation := parseUnauthorizedWriter.Result().Header.Get("Location"); parseLocation != "/app" {
		parseT.Fatalf("unexpected redirect location: %q", parseLocation)
	}

	parseAuthorizedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	parseAuthorizedReq.AddCookie(&http.Cookie{Name: authCookieName, Value: parseToken})
	parseAuthorizedWriter := httptest.NewRecorder()
	parsePageHandler.ServeHTTP(parseAuthorizedWriter, parseAuthorizedReq)
	if parseAuthorizedWriter.Code != http.StatusNoContent {
		parseT.Fatalf("expected protected page handler to run, got %d", parseAuthorizedWriter.Code)
	}

	isParseTunnelCalled := false
	isParseCallbackCalled := false
	parseTunnelHandler := parseAuth.parseRequireAuthenticatedTunnel(func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
		isParseTunnelCalled = true
		parseW2.WriteHeader(http.StatusAccepted)
	}, func(parseR3 *http.Request, parseUser2 authUser) {
		isParseCallbackCalled = parseUser2.ID == parseUser.ID && parseR3.URL.Path == "/grpc"
	})

	parseUnauthorizedTunnelWriter := httptest.NewRecorder()
	parseAuth.parseRequireAuthenticatedTunnel(func(parseW3 http.ResponseWriter, parseR4 *http.Request) {
		parseW3.WriteHeader(http.StatusNoContent)
	}, nil)(parseUnauthorizedTunnelWriter, httptest.NewRequest(http.MethodGet, "http://example.com/grpc", nil))
	if parseUnauthorizedTunnelWriter.Code != http.StatusUnauthorized {
		parseT.Fatalf("expected unauthorized tunnel response, got %d", parseUnauthorizedTunnelWriter.Code)
	}

	parseTunnelReq := httptest.NewRequest(http.MethodGet, "http://example.com/grpc", nil)
	parseTunnelReq.AddCookie(&http.Cookie{Name: authCookieName, Value: parseToken})
	parseTunnelWriter := httptest.NewRecorder()
	parseTunnelHandler(parseTunnelWriter, parseTunnelReq)
	if !isParseTunnelCalled {
		parseT.Fatal("expected authenticated tunnel handler to run")
	}
	if !isParseCallbackCalled {
		parseT.Fatal("expected authenticated tunnel callback to run")
	}
	if parseTunnelWriter.Code != http.StatusAccepted {
		parseT.Fatalf("expected authenticated tunnel response, got %d", parseTunnelWriter.Code)
	}
}
