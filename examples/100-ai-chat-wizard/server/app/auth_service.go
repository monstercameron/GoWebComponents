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

func newAuthManager(secret string, store *Store, logger *slog.Logger) *authManager {
	if strings.TrimSpace(secret) == "" {
		secret = "dev-insecure-chat-auth-secret-change-me"
		logger.Warn("auth: CHAT_AUTH_SECRET not set; using development fallback secret")
	}
	return &authManager{
		secret: []byte(secret),
		store:  store,
		logger: logger,
	}
}

func normalizeAuthEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func defaultDisplayNameFromEmail(email string) string {
	localPart := strings.TrimSpace(strings.SplitN(email, "@", 2)[0])
	if localPart == "" {
		return "User"
	}
	return localPart
}

func (a *authManager) issueToken(user authUser) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(authTokenTTL)),
		},
	})
	return token.SignedString(a.secret)
}

func (a *authManager) parseToken(tokenString string) (authUser, error) {
	if strings.TrimSpace(tokenString) == "" {
		return authUser{}, errInvalidCredentials
	}
	parsedToken, err := jwt.ParseWithClaims(tokenString, &authClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.secret, nil
	})
	if err != nil {
		return authUser{}, err
	}
	claims, ok := parsedToken.Claims.(*authClaims)
	if !ok || !parsedToken.Valid || claims.UserID <= 0 {
		return authUser{}, errInvalidCredentials
	}
	return authUser{ID: claims.UserID, Email: normalizeAuthEmail(claims.Email)}, nil
}

func (a *authManager) authenticatedUserFromContext(ctx context.Context) (authUser, bool) {
	if a == nil {
		return authUser{}, false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return authUser{}, false
	}
	for _, rawValue := range md.Get(authMetadataKey) {
		tokenString, ok := authTokenFromAuthorizationValue(rawValue)
		if !ok {
			continue
		}
		user, err := a.parseToken(tokenString)
		if err != nil {
			return authUser{}, false
		}
		return a.validateActiveUser(user, "grpc-metadata")
	}
	return authUser{}, false
}

func authTokenFromAuthorizationValue(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	if len(trimmed) >= 7 && strings.EqualFold(trimmed[:7], "Bearer ") {
		trimmed = strings.TrimSpace(trimmed[7:])
	}
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func (a *authManager) authenticatedUserFromRequest(r *http.Request) (authUser, bool) {
	if a == nil {
		return authUser{}, false
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return authUser{}, false
	}
	user, err := a.parseToken(cookie.Value)
	if err != nil {
		return authUser{}, false
	}
	return a.validateActiveUser(user, "http-cookie")
}

func (a *authManager) validateActiveUser(user authUser, source string) (authUser, bool) {
	if a == nil || user.ID <= 0 {
		return authUser{}, false
	}
	if a.store == nil {
		return user, true
	}
	exists, err := a.store.userExists(user.ID)
	if err != nil {
		if a.logger != nil {
			a.logger.Warn("auth: user existence lookup failed",
				slog.Int64("user_id", user.ID),
				slog.String("email", user.Email),
				slog.String("source", source),
				slog.String("error", err.Error()),
			)
		}
		return authUser{}, false
	}
	if !exists {
		if a.logger != nil {
			a.logger.Warn("auth: token references missing user",
				slog.Int64("user_id", user.ID),
				slog.String("email", user.Email),
				slog.String("source", source),
			)
		}
		return authUser{}, false
	}
	return user, true
}

func (a *authManager) setAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(r),
		Expires:  time.Now().Add(authTokenTTL),
		MaxAge:   int(authTokenTTL / time.Second),
	})
}

func (a *authManager) clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesHTTPS(r),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func requestUsesHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func (a *authManager) requireAuthenticatedPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.authenticatedUserFromRequest(r); !ok {
			a.clearAuthCookie(w, r)
			http.Redirect(w, r, "/app", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *authManager) requireAuthenticatedTunnel(next http.HandlerFunc, onAuthenticated func(*http.Request, authUser)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.authenticatedUserFromRequest(r)
		if !ok {
			a.clearAuthCookie(w, r)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if onAuthenticated != nil {
			onAuthenticated(r, user)
		}
		next(w, r)
	}
}

func (a *authManager) signup(email, password, displayName string) (authUser, error) {
	if a.store == nil {
		return authUser{}, errors.New("store unavailable")
	}
	normalizedEmail := normalizeAuthEmail(email)
	trimmedPassword := strings.TrimSpace(password)
	if normalizedEmail == "" || trimmedPassword == "" {
		return authUser{}, errors.New("email and password are required")
	}
	if len(trimmedPassword) < 8 {
		return authUser{}, errors.New("password must be at least 8 characters")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(trimmedPassword), bcrypt.DefaultCost)
	if err != nil {
		return authUser{}, err
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = defaultDisplayNameFromEmail(normalizedEmail)
	}
	userID, err := a.store.createUser(normalizedEmail, string(passwordHash), displayName)
	if err != nil {
		return authUser{}, err
	}
	return authUser{ID: userID, Email: normalizedEmail}, nil
}

func (a *authManager) login(email, password string) (authUser, error) {
	if a.store == nil {
		return authUser{}, errors.New("store unavailable")
	}
	record, err := a.store.getUserAuthByEmail(email)
	if err != nil {
		return authUser{}, errInvalidCredentials
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(password)); compareErr != nil {
		return authUser{}, errInvalidCredentials
	}
	return authUser{ID: record.ID, Email: record.Email}, nil
}
