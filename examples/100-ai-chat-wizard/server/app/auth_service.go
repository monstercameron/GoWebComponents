package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
)

const authCookieName = "chat_wizard_auth"
const authMetadataKey = "authorization"
const clientMetadataKey = "x-chat-client-id"
const traceParentMetadataKey = "traceparent"
const traceStateMetadataKey = "tracestate"
const requestIDMetadataKey = "x-request-id"
const correlationIDMetadataKey = "x-correlation-id"

const authTokenTTL = 2 * time.Hour // Development default: 2 hours. Tighten to 1 hour in production.

var errInvalidCredentials = errors.New("invalid credentials")

type authUser struct {
	ID    int64
	Email string
}

type authClaims struct {
	UserID int64  `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type authManager struct {
	secret []byte
	store  *Store
	logger *slog.Logger
}

func parseNewAuthManager(parseSecret string, store *Store, parseLogger *slog.Logger) *authManager {
	if strings.TrimSpace(parseSecret) == "" {
		parseSecret = "dev-insecure-chat-auth-secret-change-me"
		parseLogger.Warn("auth: CHAT_AUTH_SECRET not set; using development fallback secret")
	}
	return &authManager{
		secret: []byte(parseSecret),
		store:  store,
		logger: parseLogger,
	}
}

func parseNormalizeAuthEmail(parseEmail string) string {
	return strings.ToLower(strings.TrimSpace(parseEmail))
}

func parseDefaultDisplayNameFromEmail(parseEmail string) string {
	parseLocalPart := strings.TrimSpace(strings.SplitN(parseEmail, "@", 2)[0])
	if parseLocalPart == "" {
		return "User"
	}
	return parseLocalPart
}

func (parseA *authManager) issueToken(parseUser authUser) (string, error) {
	parseNow := time.Now()
	parseToken := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
		UserID: parseUser.ID,
		Email:  parseUser.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user",
			IssuedAt:  jwt.NewNumericDate(parseNow),
			ExpiresAt: jwt.NewNumericDate(parseNow.Add(authTokenTTL)),
		},
	})
	return parseToken.SignedString(parseA.secret)
}

func (parseA *authManager) parseToken(parseTokenString string) (authUser, error) {
	if strings.TrimSpace(parseTokenString) == "" {
		return authUser{}, errInvalidCredentials
	}
	parseParsedToken, parseErr := jwt.ParseWithClaims(parseTokenString, &authClaims{}, func(parseToken *jwt.Token) (interface{}, error) {
		if _, parseOk := parseToken.Method.(*jwt.SigningMethodHMAC); !parseOk {
			return nil, errors.New("unexpected signing method")
		}
		return parseA.secret, nil
	})
	if parseErr != nil {
		return authUser{}, parseErr
	}
	parseClaims, parseOk2 := parseParsedToken.Claims.(*authClaims)
	if !parseOk2 || !parseParsedToken.Valid || parseClaims.UserID <= 0 {
		return authUser{}, errInvalidCredentials
	}
	return authUser{ID: parseClaims.UserID, Email: parseNormalizeAuthEmail(parseClaims.Email)}, nil
}

func (parseA *authManager) parseAuthenticatedUserFromContext(parseCtx context.Context) (authUser, bool) {
	if parseA == nil {
		return authUser{}, false
	}
	parseMd, parseOk := metadata.FromIncomingContext(parseCtx)
	if !parseOk {
		return authUser{}, false
	}
	for _, parseRawValue := range parseMd.Get(authMetadataKey) {
		parseTokenString, parseOk2 := parseAuthTokenFromAuthorizationValue(parseRawValue)
		if !parseOk2 {
			continue
		}
		parseUser, parseErr := parseA.parseToken(parseTokenString)
		if parseErr != nil {
			return authUser{}, false
		}
		return parseA.parseValidateActiveUser(parseUser, "grpc-metadata")
	}
	return authUser{}, false
}

func parseAuthTokenFromAuthorizationValue(parseValue string) (string, bool) {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return "", false
	}
	if len(parseTrimmed) >= 7 && strings.EqualFold(parseTrimmed[:7], "Bearer ") {
		parseTrimmed = strings.TrimSpace(parseTrimmed[7:])
	}
	if parseTrimmed == "" {
		return "", false
	}
	return parseTrimmed, true
}

func (parseA *authManager) parseAuthenticatedUserFromRequest(parseR *http.Request) (authUser, bool) {
	if parseA == nil {
		return authUser{}, false
	}
	parseCookie, parseErr := parseR.Cookie(authCookieName)
	if parseErr != nil {
		return authUser{}, false
	}
	parseUser, parseErr := parseA.parseToken(parseCookie.Value)
	if parseErr != nil {
		return authUser{}, false
	}
	return parseA.parseValidateActiveUser(parseUser, "http-cookie")
}

func (parseA *authManager) parseValidateActiveUser(parseUser authUser, parseSource string) (authUser, bool) {
	if parseA == nil || parseUser.ID <= 0 {
		return authUser{}, false
	}
	if parseA.store == nil {
		return parseUser, true
	}
	parseExists, parseErr := parseA.store.parseUserExists(parseUser.ID)
	if parseErr != nil {
		if parseA.logger != nil {
			parseA.logger.Warn("auth: user existence lookup failed",
				slog.Int64("user_id", parseUser.ID),
				slog.String("email", parseUser.Email),
				slog.String("source", parseSource),
				slog.String("error", parseErr.Error()),
			)
		}
		return authUser{}, false
	}
	if !parseExists {
		if parseA.logger != nil {
			parseA.logger.Warn("auth: token references missing user",
				slog.Int64("user_id", parseUser.ID),
				slog.String("email", parseUser.Email),
				slog.String("source", parseSource),
			)
		}
		return authUser{}, false
	}
	return parseUser, true
}

func (parseA *authManager) setAuthCookie(parseW http.ResponseWriter, parseR *http.Request, parseToken string) {
	http.SetCookie(parseW, &http.Cookie{
		Name:     authCookieName,
		Value:    parseToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   parseRequestUsesHTTPS(parseR),
		Expires:  time.Now().Add(authTokenTTL),
		MaxAge:   int(authTokenTTL / time.Second),
	})
}

func (parseA *authManager) clearAuthCookie(parseW http.ResponseWriter, parseR *http.Request) {
	http.SetCookie(parseW, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   parseRequestUsesHTTPS(parseR),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func parseRequestUsesHTTPS(parseR *http.Request) bool {
	if parseR == nil {
		return false
	}
	if parseR.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(parseR.Header.Get("X-Forwarded-Proto")), "https")
}

func (parseA *authManager) parseRequireAuthenticatedPage(parseNext http.Handler) http.Handler {
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if _, parseOk := parseA.parseAuthenticatedUserFromRequest(parseR); !parseOk {
			parseA.clearAuthCookie(parseW, parseR)
			http.Redirect(parseW, parseR, "/app", http.StatusSeeOther)
			return
		}
		parseNext.ServeHTTP(parseW, parseR)
	})
}

func (parseA *authManager) parseRequireAuthenticatedTunnel(parseNext http.HandlerFunc, parseOnAuthenticated func(*http.Request, authUser)) http.HandlerFunc {
	return func(parseW http.ResponseWriter, parseR *http.Request) {
		parseUser, parseOk := parseA.parseAuthenticatedUserFromRequest(parseR)
		if !parseOk {
			parseA.clearAuthCookie(parseW, parseR)
			http.Error(parseW, "authentication required", http.StatusUnauthorized)
			return
		}
		if parseOnAuthenticated != nil {
			parseOnAuthenticated(parseR, parseUser)
		}
		parseNext(parseW, parseR)
	}
}

func (parseA *authManager) parseSignup(parseEmail, parsePassword, parseDisplayName string) (authUser, error) {
	if parseA.store == nil {
		return authUser{}, errors.New("store unavailable")
	}
	parseNormalizedEmail := parseNormalizeAuthEmail(parseEmail)
	parseTrimmedPassword := strings.TrimSpace(parsePassword)
	if parseNormalizedEmail == "" || parseTrimmedPassword == "" {
		return authUser{}, errors.New("email and password are required")
	}
	if len(parseTrimmedPassword) < 8 {
		return authUser{}, errors.New("password must be at least 8 characters")
	}
	parsePasswordHash, parseErr := bcrypt.GenerateFromPassword([]byte(parseTrimmedPassword), bcrypt.DefaultCost)
	if parseErr != nil {
		return authUser{}, parseErr
	}
	if strings.TrimSpace(parseDisplayName) == "" {
		parseDisplayName = parseDefaultDisplayNameFromEmail(parseNormalizedEmail)
	}
	parseUserID, parseErr := parseA.store.parseCreateUser(parseNormalizedEmail, string(parsePasswordHash), parseDisplayName)
	if parseErr != nil {
		return authUser{}, parseErr
	}
	return authUser{ID: parseUserID, Email: parseNormalizedEmail}, nil
}

func (parseA *authManager) parseLogin(parseEmail, parsePassword string) (authUser, error) {
	if parseA.store == nil {
		return authUser{}, errors.New("store unavailable")
	}
	parseRecord, parseErr := parseA.store.getUserAuthByEmail(parseEmail)
	if parseErr != nil {
		return authUser{}, errInvalidCredentials
	}
	if parseCompareErr := bcrypt.CompareHashAndPassword([]byte(parseRecord.PasswordHash), []byte(parsePassword)); parseCompareErr != nil {
		return authUser{}, errInvalidCredentials
	}
	return authUser{ID: parseRecord.ID, Email: parseRecord.Email}, nil
}
