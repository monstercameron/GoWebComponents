package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const authCookieName = "chat_wizard_auth"
const authMetadataKey = "authorization"
const clientMetadataKey = "x-chat-client-id"
const traceParentMetadataKey = "traceparent"
const traceStateMetadataKey = "tracestate"
const requestIDMetadataKey = "x-request-id"
const correlationIDMetadataKey = "x-correlation-id"

const authTokenTTL = 2 * time.Hour // Development default: 2 hours. Tighten to 1 hour in production.
const emailVerificationTokenTTL = 24 * time.Hour
const passwordResetTokenTTL = 15 * time.Minute
const passwordResetRequestThrottleWindow = 15 * time.Minute
const passwordResetRequestThrottleMaxPerEmail = int64(3)
const passwordResetRequestThrottleMaxPerIP = int64(8)

const parsePasswordRecoveryAuditEventRequest = "auth.recovery.request"
const parsePasswordRecoveryAuditEventConsume = "auth.recovery.consume"

const parsePasswordRecoveryAuditOutcomeRequested = "requested"
const parsePasswordRecoveryAuditOutcomeThrottled = "throttled"
const parsePasswordRecoveryAuditOutcomeMissingUser = "missing_user"
const parsePasswordRecoveryAuditOutcomeConsumed = "consumed"
const parsePasswordRecoveryAuditOutcomeExpired = "expired"
const parsePasswordRecoveryAuditOutcomeReplayDenied = "replay_denied"
const parsePasswordRecoveryAuditOutcomeInvalidToken = "invalid_token"

var errInvalidCredentials = errors.New("invalid credentials")
var errPasswordRecoveryUnavailable = errors.New("password recovery unavailable")

type authUser struct {
	ID    int64
	Email string
}

type authClaims struct {
	UserID       int64  `json:"uid"`
	Email        string `json:"email"`
	SessionID    string `json:"sid"`
	TokenVersion int64  `json:"ver"`
	AuthMethod   string `json:"amr,omitempty"`
	jwt.RegisteredClaims
}

type authManager struct {
	secret       []byte
	signingKeyID string
	verifyKeys   map[string][]byte
	store        *Store
	logger       *slog.Logger
}

func parseNewAuthManager(parseSecret string, store *Store, parseLogger *slog.Logger) *authManager {
	parseVerifyKeys, parseSigningKeyID := parseResolveAuthSigningKeys(parseSecret, parseLogger)
	return &authManager{
		secret:       parseVerifyKeys[parseSigningKeyID],
		signingKeyID: parseSigningKeyID,
		verifyKeys:   parseVerifyKeys,
		store:        store,
		logger:       parseLogger,
	}
}

// parseResolveAuthSigningKeys resolves signing and verification keys from runtime auth configuration.
func parseResolveAuthSigningKeys(parsePrimarySecret string, parseLogger *slog.Logger) (map[string][]byte, string) {
	parsePrimarySecret = strings.TrimSpace(parsePrimarySecret)
	if parsePrimarySecret == "" {
		parsePrimarySecret = "dev-insecure-chat-auth-secret-change-me"
		if parseLogger != nil {
			parseLogger.Warn("auth: CHAT_AUTH_SECRET not set; using development fallback secret")
		}
	}
	parseVerifyKeys := map[string][]byte{
		"primary": []byte(parsePrimarySecret),
	}
	for parseKeyID, parseSecret := range parseParseAuthSigningKeys(os.Getenv("CHAT_AUTH_SIGNING_KEYS")) {
		parseVerifyKeys[parseKeyID] = []byte(parseSecret)
	}
	parseSigningKeyID := strings.TrimSpace(os.Getenv("CHAT_AUTH_ACTIVE_KID"))
	if parseSigningKeyID == "" {
		parseSigningKeyID = "primary"
	}
	if _, parseOk := parseVerifyKeys[parseSigningKeyID]; !parseOk {
		parseSigningKeyIDs := parseListAuthSigningKeyIDs(parseVerifyKeys)
		parseSigningKeyID = parseSigningKeyIDs[0]
		if parseLogger != nil {
			parseLogger.Warn("auth: CHAT_AUTH_ACTIVE_KID missing from configured keys; falling back",
				slog.String("fallback_kid", parseSigningKeyID),
			)
		}
	}
	return parseVerifyKeys, parseSigningKeyID
}

// parseParseAuthSigningKeys parses one comma-delimited key ring spec formatted as `kid=secret`.
func parseParseAuthSigningKeys(parseRawKeys string) map[string]string {
	parseRawKeys = strings.TrimSpace(parseRawKeys)
	if parseRawKeys == "" {
		return map[string]string{}
	}
	parseResolved := map[string]string{}
	for _, parseToken := range strings.Split(parseRawKeys, ",") {
		parseToken = strings.TrimSpace(parseToken)
		if parseToken == "" {
			continue
		}
		parsePair := strings.SplitN(parseToken, "=", 2)
		if len(parsePair) != 2 {
			continue
		}
		parseKeyID := strings.TrimSpace(parsePair[0])
		parseSecret := strings.TrimSpace(parsePair[1])
		if parseKeyID == "" || parseSecret == "" {
			continue
		}
		parseResolved[parseKeyID] = parseSecret
	}
	return parseResolved
}

// parseListAuthSigningKeyIDs lists configured signing key ids in stable lexical order.
func parseListAuthSigningKeyIDs(parseVerifyKeys map[string][]byte) []string {
	parseSigningKeyIDs := make([]string, 0, len(parseVerifyKeys))
	for parseKeyID := range parseVerifyKeys {
		parseSigningKeyIDs = append(parseSigningKeyIDs, parseKeyID)
	}
	sort.Strings(parseSigningKeyIDs)
	return parseSigningKeyIDs
}

type parseAuthRequestMetadata struct {
	UserAgent string
	IPAddress string
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

// parseBuildOpaqueAuthFlowToken issues one opaque token for reset/verification workflows.
func parseBuildOpaqueAuthFlowToken() string {
	return uuid.NewString() + "." + uuid.NewString()
}

// parseBuildAuthFlowTokenHash returns one stable SHA-256 digest for one raw auth-flow token.
func parseBuildAuthFlowTokenHash(parseRawToken string) string {
	parseDigest := sha256.Sum256([]byte(strings.TrimSpace(parseRawToken)))
	return hex.EncodeToString(parseDigest[:])
}

// parseBuildAuthAuditEmailHash returns one short deterministic email hash for recovery audit logs.
func parseBuildAuthAuditEmailHash(parseEmail string) string {
	parseEmail = parseNormalizeAuthEmail(parseEmail)
	if parseEmail == "" {
		return ""
	}
	parseEmailHash := parseBuildAuthFlowTokenHash(parseEmail)
	if len(parseEmailHash) > 16 {
		return parseEmailHash[:16]
	}
	return parseEmailHash
}

// issueToken issues one auth token with a new durable server-side session.
func (parseA *authManager) issueToken(parseUser authUser) (string, error) {
	return parseA.issueTokenForContextWithAuthMethod(context.Background(), parseUser, "", parseWorkspaceAuthMethodPassword)
}

// issueTokenForContext issues one auth token, persisting the backing auth session for one user.
func (parseA *authManager) issueTokenForContext(parseCtx context.Context, parseUser authUser, parseSessionID string) (string, error) {
	return parseA.issueTokenForContextWithAuthMethod(parseCtx, parseUser, parseSessionID, parseWorkspaceAuthMethodPassword)
}

// issueTokenForContextWithAuthMethod issues one auth token with one explicit login-method identity.
func (parseA *authManager) issueTokenForContextWithAuthMethod(parseCtx context.Context, parseUser authUser, parseSessionID string, parseAuthMethod string) (string, error) {
	parseNow := time.Now().UTC()
	parseSessionID = strings.TrimSpace(parseSessionID)
	if parseSessionID == "" {
		parseSessionID = uuid.NewString()
	}
	parseAuthMethod = parseNormalizePrivilegedAuthMethod(parseAuthMethod)
	parseTokenVersion := int64(1)
	if parseA != nil && parseA.store != nil && parseUser.ID > 0 {
		parseResolvedVersion, parseErr := parseA.store.parseEnsureAuthTokenVersion(parseUser.ID)
		if parseErr != nil {
			return "", parseErr
		}
		if parseResolvedVersion > 0 {
			parseTokenVersion = parseResolvedVersion
		}
		parseMetadata := parseResolveAuthMetadataFromContext(parseCtx)
		if parseErr2 := parseA.store.parseUpsertAuthSession(parseAuthSessionWrite{
			UserID:           parseUser.ID,
			SessionID:        parseSessionID,
			TokenVersion:     parseTokenVersion,
			RefreshTokenHash: "",
			UserAgent:        parseMetadata.UserAgent,
			IPAddress:        parseMetadata.IPAddress,
			LastSeenAt:       parseNow.Format(time.RFC3339),
			ExpiresAt:        parseNow.Add(authTokenTTL).Format(time.RFC3339),
			RevokedAt:        "",
		}); parseErr2 != nil {
			return "", parseErr2
		}
	}
	parseToken := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
		UserID:       parseUser.ID,
		Email:        parseUser.Email,
		SessionID:    parseSessionID,
		TokenVersion: parseTokenVersion,
		AuthMethod:   parseAuthMethod,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        parseSessionID,
			Subject:   "user",
			IssuedAt:  jwt.NewNumericDate(parseNow),
			ExpiresAt: jwt.NewNumericDate(parseNow.Add(authTokenTTL)),
		},
	})
	if strings.TrimSpace(parseA.signingKeyID) != "" {
		parseToken.Header["kid"] = parseA.signingKeyID
	}
	return parseToken.SignedString(parseA.secret)
}

// parseToken parses and validates one token string and returns one authenticated user.
func (parseA *authManager) parseToken(parseTokenString string) (authUser, error) {
	parseUser, _, parseErr := parseA.parseTokenWithMetadata(parseTokenString, parseAuthRequestMetadata{})
	return parseUser, parseErr
}

// parseTokenWithMetadata parses one token and applies server-side revocation/session checks with request metadata.
func (parseA *authManager) parseTokenWithMetadata(parseTokenString string, parseMetadata parseAuthRequestMetadata) (authUser, authClaims, error) {
	if strings.TrimSpace(parseTokenString) == "" {
		return authUser{}, authClaims{}, errInvalidCredentials
	}
	parseClaims, parseErr := parseA.parseValidateSignedTokenClaims(parseTokenString)
	if parseErr != nil {
		return authUser{}, authClaims{}, parseErr
	}
	if parseClaims.UserID <= 0 {
		return authUser{}, authClaims{}, errInvalidCredentials
	}
	if parseA.parseIsTokenRevoked(parseClaims, parseMetadata) {
		return authUser{}, authClaims{}, errInvalidCredentials
	}
	return authUser{ID: parseClaims.UserID, Email: parseNormalizeAuthEmail(parseClaims.Email)}, *parseClaims, nil
}

// parseValidateSignedTokenClaims validates one JWT signature and claims using the configured key-rotation policy.
func (parseA *authManager) parseValidateSignedTokenClaims(parseTokenString string) (*authClaims, error) {
	parseVerificationKeys, parseErr := parseA.parseResolveVerificationKeys(parseTokenString)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLastErr error
	for _, parseVerificationKey := range parseVerificationKeys {
		parseParsedToken, parseErr := jwt.ParseWithClaims(parseTokenString, &authClaims{}, func(parseToken *jwt.Token) (interface{}, error) {
			if _, parseOk := parseToken.Method.(*jwt.SigningMethodHMAC); !parseOk {
				return nil, errors.New("unexpected signing method")
			}
			return parseVerificationKey, nil
		})
		if parseErr != nil {
			parseLastErr = parseErr
			continue
		}
		parseClaims, parseOk := parseParsedToken.Claims.(*authClaims)
		if !parseOk || !parseParsedToken.Valid {
			parseLastErr = errInvalidCredentials
			continue
		}
		return parseClaims, nil
	}
	if parseLastErr == nil {
		parseLastErr = errInvalidCredentials
	}
	return nil, parseLastErr
}

// parseResolveVerificationKeys resolves candidate verification keys for one token according to `kid` policy.
func (parseA *authManager) parseResolveVerificationKeys(parseTokenString string) ([][]byte, error) {
	if parseA == nil || len(parseA.verifyKeys) == 0 {
		return nil, errInvalidCredentials
	}
	parseKeyID, parseErr := parseExtractAuthTokenKeyID(parseTokenString)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseKeyID != "" {
		parseVerificationKey, parseOk := parseA.verifyKeys[parseKeyID]
		if !parseOk {
			return nil, errInvalidCredentials
		}
		return [][]byte{parseVerificationKey}, nil
	}
	parseSigningKeyIDs := parseListAuthSigningKeyIDs(parseA.verifyKeys)
	parseVerificationKeys := make([][]byte, 0, len(parseSigningKeyIDs))
	if parseActiveKey, parseOk := parseA.verifyKeys[parseA.signingKeyID]; parseOk {
		parseVerificationKeys = append(parseVerificationKeys, parseActiveKey)
	}
	for _, parseSigningKeyID := range parseSigningKeyIDs {
		if parseSigningKeyID == parseA.signingKeyID {
			continue
		}
		parseVerificationKeys = append(parseVerificationKeys, parseA.verifyKeys[parseSigningKeyID])
	}
	return parseVerificationKeys, nil
}

// parseExtractAuthTokenKeyID reads one token `kid` header without verifying signature.
func parseExtractAuthTokenKeyID(parseTokenString string) (string, error) {
	parseParser := jwt.Parser{}
	parseUnverifiedToken, _, parseErr := parseParser.ParseUnverified(parseTokenString, &authClaims{})
	if parseErr != nil {
		return "", parseErr
	}
	parseRawKeyID, parseOk := parseUnverifiedToken.Header["kid"]
	if !parseOk {
		return "", nil
	}
	parseKeyID, parseOk := parseRawKeyID.(string)
	if !parseOk {
		return "", errInvalidCredentials
	}
	return strings.TrimSpace(parseKeyID), nil
}

// parseIsTokenRevoked checks whether one validated token should be rejected by revocation policy.
func (parseA *authManager) parseIsTokenRevoked(parseClaims *authClaims, parseMetadata parseAuthRequestMetadata) bool {
	if parseA == nil || parseA.store == nil {
		return false
	}
	parseSessionID := strings.TrimSpace(parseClaims.SessionID)
	if parseClaims.UserID <= 0 || parseSessionID == "" || parseClaims.TokenVersion <= 0 {
		return true
	}
	if strings.TrimSpace(parseClaims.ID) == "" || strings.TrimSpace(parseClaims.ID) != parseSessionID {
		return true
	}
	parseSessionRow, isParseFound, parseErr := parseA.store.parseGetAuthSessionBySessionID(parseSessionID)
	if parseErr != nil {
		parseA.parseLogSessionValidationFailure("session lookup failed", parseClaims, parseErr)
		return true
	}
	if !isParseFound || parseSessionRow.UserID != parseClaims.UserID {
		return true
	}
	if strings.TrimSpace(parseSessionRow.RevokedAt) != "" {
		return true
	}
	parseExpiresAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseSessionRow.ExpiresAt))
	if parseErr != nil {
		parseA.parseLogSessionValidationFailure("session expiry parse failed", parseClaims, parseErr)
		return true
	}
	if time.Now().UTC().After(parseExpiresAt) {
		return true
	}
	if parseSessionRow.TokenVersion != parseClaims.TokenVersion {
		return true
	}
	parseTokenVersion, parseErr := parseA.store.parseGetAuthTokenVersion(parseClaims.UserID)
	if parseErr != nil {
		parseA.parseLogSessionValidationFailure("token version lookup failed", parseClaims, parseErr)
		return true
	}
	if parseTokenVersion <= 0 || parseTokenVersion != parseClaims.TokenVersion {
		return true
	}
	parseTouchUserAgent := strings.TrimSpace(parseMetadata.UserAgent)
	if parseTouchUserAgent == "" {
		parseTouchUserAgent = strings.TrimSpace(parseSessionRow.UserAgent)
	}
	parseTouchIPAddress := strings.TrimSpace(parseMetadata.IPAddress)
	if parseTouchIPAddress == "" {
		parseTouchIPAddress = strings.TrimSpace(parseSessionRow.IPAddress)
	}
	if parseErr2 := parseA.store.parseTouchAuthSessionLastSeen(
		parseSessionID,
		parseTouchUserAgent,
		parseTouchIPAddress,
		time.Now().UTC().Add(authTokenTTL),
	); parseErr2 != nil {
		parseA.parseLogSessionValidationFailure("session touch failed", parseClaims, parseErr2)
		return true
	}
	return false
}

// parseLogSessionValidationFailure emits one structured auth session validation warning.
func (parseA *authManager) parseLogSessionValidationFailure(parseReason string, parseClaims *authClaims, parseErr error) {
	if parseA == nil || parseA.logger == nil {
		return
	}
	parseA.logger.Warn("auth: session validation failed",
		slog.String("reason", strings.TrimSpace(parseReason)),
		slog.Int64("user_id", parseClaims.UserID),
		slog.String("session_id", strings.TrimSpace(parseClaims.SessionID)),
		slog.String("jti", strings.TrimSpace(parseClaims.ID)),
		slog.Int64("token_version", parseClaims.TokenVersion),
		slog.String("error", parseErr.Error()),
	)
}

// parseAuthenticatedSessionFromContext authenticates one metadata token and returns both user and claims.
func (parseA *authManager) parseAuthenticatedSessionFromContext(parseCtx context.Context) (authUser, authClaims, bool) {
	if parseA == nil {
		return authUser{}, authClaims{}, false
	}
	parseTokenString, parseOk := parseResolveAuthTokenFromContext(parseCtx)
	if !parseOk {
		return authUser{}, authClaims{}, false
	}
	parseMetadata := parseResolveAuthMetadataFromContext(parseCtx)
	parseUser, parseClaims, parseErr := parseA.parseTokenWithMetadata(parseTokenString, parseMetadata)
	if parseErr != nil {
		return authUser{}, authClaims{}, false
	}
	parseValidatedUser, parseOk2 := parseA.parseValidateActiveUser(parseUser, "grpc-metadata")
	if !parseOk2 {
		return authUser{}, authClaims{}, false
	}
	return parseValidatedUser, parseClaims, true
}

// parseAuthenticatedUserFromContext authenticates one incoming gRPC request and returns one user.
func (parseA *authManager) parseAuthenticatedUserFromContext(parseCtx context.Context) (authUser, bool) {
	parseUser, _, parseOk := parseA.parseAuthenticatedSessionFromContext(parseCtx)
	return parseUser, parseOk
}

// parseAuthTokenFromAuthorizationValue extracts one bearer token from one authorization header value.
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

// parseResolveAuthTokenFromContext extracts one auth token from gRPC metadata.
func parseResolveAuthTokenFromContext(parseCtx context.Context) (string, bool) {
	parseMd, parseOk := metadata.FromIncomingContext(parseCtx)
	if !parseOk {
		return "", false
	}
	for _, parseRawValue := range parseMd.Get(authMetadataKey) {
		parseTokenString, parseOk2 := parseAuthTokenFromAuthorizationValue(parseRawValue)
		if !parseOk2 {
			continue
		}
		return parseTokenString, true
	}
	return "", false
}

// parseResolveAuthMetadataFromContext resolves request user-agent and client IP metadata from one gRPC context.
func parseResolveAuthMetadataFromContext(parseCtx context.Context) parseAuthRequestMetadata {
	parseMetadata := parseAuthRequestMetadata{}
	if parseMd, parseOk := metadata.FromIncomingContext(parseCtx); parseOk {
		parseMetadata.UserAgent = parseFirstNonEmptyMetadataValue(parseMd, "user-agent")
		parseMetadata.IPAddress = parseExtractIPAddress(parseFirstNonEmptyMetadataValue(parseMd, "x-forwarded-for", "x-real-ip"))
	}
	if strings.TrimSpace(parseMetadata.IPAddress) == "" {
		if parsePeer, parseOk2 := peer.FromContext(parseCtx); parseOk2 && parsePeer.Addr != nil {
			parseMetadata.IPAddress = parseExtractIPAddress(parsePeer.Addr.String())
		}
	}
	return parseMetadata
}

// parseResolveAuthMetadataFromRequest resolves request user-agent and client IP from one HTTP request.
func parseResolveAuthMetadataFromRequest(parseR *http.Request) parseAuthRequestMetadata {
	if parseR == nil {
		return parseAuthRequestMetadata{}
	}
	parseMetadata := parseAuthRequestMetadata{
		UserAgent: strings.TrimSpace(parseR.UserAgent()),
		IPAddress: parseExtractIPAddress(parseR.Header.Get("X-Forwarded-For")),
	}
	if strings.TrimSpace(parseMetadata.IPAddress) == "" {
		parseMetadata.IPAddress = parseExtractIPAddress(parseR.Header.Get("X-Real-IP"))
	}
	if strings.TrimSpace(parseMetadata.IPAddress) == "" {
		parseMetadata.IPAddress = parseExtractIPAddress(parseR.RemoteAddr)
	}
	return parseMetadata
}

// parseFirstNonEmptyMetadataValue returns one trimmed metadata value from the first matching key.
func parseFirstNonEmptyMetadataValue(parseMd metadata.MD, parseKeys ...string) string {
	for _, parseKey := range parseKeys {
		for _, parseValue := range parseMd.Get(parseKey) {
			parseValue = strings.TrimSpace(parseValue)
			if parseValue == "" {
				continue
			}
			return parseValue
		}
	}
	return ""
}

// parseExtractIPAddress extracts one IP address from one forwarded-for, host:port, or raw IP input.
func parseExtractIPAddress(parseRawAddress string) string {
	parseRawAddress = strings.TrimSpace(parseRawAddress)
	if parseRawAddress == "" {
		return ""
	}
	if strings.Contains(parseRawAddress, ",") {
		parseRawAddress = strings.TrimSpace(strings.Split(parseRawAddress, ",")[0])
	}
	if parseParsedIP := net.ParseIP(parseRawAddress); parseParsedIP != nil {
		return parseParsedIP.String()
	}
	parseHost, _, parseErr := net.SplitHostPort(parseRawAddress)
	if parseErr != nil {
		return parseRawAddress
	}
	if parseParsedIP := net.ParseIP(strings.TrimSpace(parseHost)); parseParsedIP != nil {
		return parseParsedIP.String()
	}
	return strings.TrimSpace(parseHost)
}

// parseAuthenticatedUserFromRequest authenticates one HTTP request via auth cookie.
func (parseA *authManager) parseAuthenticatedUserFromRequest(parseR *http.Request) (authUser, bool) {
	if parseA == nil {
		return authUser{}, false
	}
	parseCookie, parseErr := parseR.Cookie(authCookieName)
	if parseErr != nil {
		return authUser{}, false
	}
	parseMetadata := parseResolveAuthMetadataFromRequest(parseR)
	parseUser, _, parseErr := parseA.parseTokenWithMetadata(parseCookie.Value, parseMetadata)
	if parseErr != nil {
		return authUser{}, false
	}
	return parseA.parseValidateActiveUser(parseUser, "http-cookie")
}

// parseRevokeSessionFromContext revokes one authenticated session bound to one incoming gRPC context.
func (parseA *authManager) parseRevokeSessionFromContext(parseCtx context.Context) error {
	if parseA == nil || parseA.store == nil {
		return nil
	}
	_, parseClaims, parseOk := parseA.parseAuthenticatedSessionFromContext(parseCtx)
	if !parseOk {
		return nil
	}
	return parseA.store.parseRevokeAuthSession(parseClaims.SessionID)
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
	isParseAccessDisabled, parseErr := parseA.store.parseIsUserAccessDisabled(parseUser.ID)
	if parseErr != nil {
		if parseA.logger != nil {
			parseA.logger.Warn("auth: user access-state lookup failed",
				slog.Int64("user_id", parseUser.ID),
				slog.String("email", parseUser.Email),
				slog.String("source", parseSource),
				slog.String("error", parseErr.Error()),
			)
		}
		return authUser{}, false
	}
	if isParseAccessDisabled {
		if parseA.logger != nil {
			parseA.logger.Warn("auth: user access is disabled",
				slog.Int64("user_id", parseUser.ID),
				slog.String("email", parseUser.Email),
				slog.String("source", parseSource),
			)
		}
		return authUser{}, false
	}
	parseBlockCount, parseErr := parseA.store.parseCountUserAuthBlocksByUser(parseUser.ID)
	if parseErr != nil {
		if parseA.logger != nil {
			parseA.logger.Warn("auth: user auth block lookup failed",
				slog.Int64("user_id", parseUser.ID),
				slog.String("email", parseUser.Email),
				slog.String("source", parseSource),
				slog.String("error", parseErr.Error()),
			)
		}
		return authUser{}, false
	}
	if parseBlockCount > 0 {
		if parseA.logger != nil {
			parseA.logger.Warn("auth: user authentication blocked",
				slog.Int64("user_id", parseUser.ID),
				slog.String("email", parseUser.Email),
				slog.String("source", parseSource),
				slog.Int64("block_count", parseBlockCount),
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

// parseBuildTraceabilityContextFromRequest merges request, correlation, and trace headers into one request context.
func parseBuildTraceabilityContextFromRequest(parseR *http.Request) context.Context {
	if parseR == nil {
		return context.Background()
	}
	parseIncomingMD, _ := metadata.FromIncomingContext(parseR.Context())
	parsePairs := make([]string, 0, 8)
	for _, parseKey := range []string{requestIDMetadataKey, correlationIDMetadataKey, traceParentMetadataKey, traceStateMetadataKey} {
		if parseValue := strings.TrimSpace(parseR.Header.Get(parseKey)); parseValue != "" {
			parsePairs = append(parsePairs, parseKey, parseValue)
		}
	}
	if len(parsePairs) == 0 {
		if len(parseIncomingMD) == 0 {
			return parseR.Context()
		}
		return metadata.NewIncomingContext(parseR.Context(), parseIncomingMD)
	}
	if len(parseIncomingMD) > 0 {
		parsePairsMD := metadata.Pairs(parsePairs...)
		for parseKey, parseValues := range parseIncomingMD {
			for _, parseValue := range parseValues {
				parsePairsMD.Append(parseKey, parseValue)
			}
		}
		return metadata.NewIncomingContext(parseR.Context(), parsePairsMD)
	}
	return metadata.NewIncomingContext(parseR.Context(), metadata.Pairs(parsePairs...))
}

func (parseA *authManager) parseRequireAuthenticatedPage(parseNext http.Handler) http.Handler {
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if _, parseOk := parseA.parseAuthenticatedUserFromRequest(parseR); !parseOk {
			parseA.clearAuthCookie(parseW, parseR)
			http.Redirect(parseW, parseR, "/app", http.StatusSeeOther)
			return
		}
		parseNext.ServeHTTP(parseW, parseR.Clone(parseBuildTraceabilityContextFromRequest(parseR)))
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
		parseTraceableReq := parseR.Clone(parseBuildTraceabilityContextFromRequest(parseR))
		if parseOnAuthenticated != nil {
			parseOnAuthenticated(parseTraceableReq, parseUser)
		}
		parseNext(parseW, parseTraceableReq)
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
	parseUser := authUser{ID: parseUserID, Email: parseNormalizedEmail}
	if _, parseErr2 := parseA.parseIssueEmailVerificationTokenForUser(parseUser); parseErr2 != nil {
		return authUser{}, parseErr2
	}
	return parseUser, nil
}

func (parseA *authManager) parseLogin(parseEmail, parsePassword string) (authUser, error) {
	if parseA.store == nil {
		return authUser{}, errors.New("store unavailable")
	}
	parseEmail = parseNormalizeAuthEmail(parseEmail)
	parseRecord, parseErr := parseA.store.getUserAuthByEmail(parseEmail)
	if parseErr != nil {
		return authUser{}, errInvalidCredentials
	}
	if parseCompareErr := bcrypt.CompareHashAndPassword([]byte(parseRecord.PasswordHash), []byte(parsePassword)); parseCompareErr != nil {
		return authUser{}, errInvalidCredentials
	}
	parseUser := authUser{ID: parseRecord.ID, Email: parseRecord.Email}
	isParseLoginAllowed, parseErr := parseA.parseAuthorizeLoginSignupVerificationPolicy(parseUser.Email, time.Now().UTC())
	if parseErr != nil {
		return authUser{}, errSignupVerificationUnavailable
	}
	if !isParseLoginAllowed {
		parseA.parseLogSignupVerificationAuditEvent(parseSignupVerificationAuditEventRequest, parseSignupVerificationAuditOutcomePolicyDenied, parseUser.ID, parseUser.Email)
		return authUser{}, errInvalidCredentials
	}
	if _, parseOk := parseA.parseValidateActiveUser(parseUser, "login-password"); !parseOk {
		return authUser{}, errInvalidCredentials
	}
	return parseUser, nil
}

// parseIssueEmailVerificationTokenForUser persists one pending email-verification token for one newly signed-up user.
func (parseA *authManager) parseIssueEmailVerificationTokenForUser(parseUser authUser) (string, error) {
	if parseA == nil || parseA.store == nil {
		return "", errors.New("issue email verification token: store unavailable")
	}
	parseEmail := parseNormalizeAuthEmail(parseUser.Email)
	if parseUser.ID <= 0 || parseEmail == "" {
		return "", errors.New("issue email verification token: user id and email are required")
	}
	parseRawToken := parseBuildOpaqueAuthFlowToken()
	parseTokenHash := parseBuildAuthFlowTokenHash(parseRawToken)
	parseExpiresAt := time.Now().UTC().Add(emailVerificationTokenTTL).Format(time.RFC3339)
	if _, parseErr := parseA.store.parseCreateEmailVerificationToken(parseEmailVerificationTokenWrite{
		UserID:    parseUser.ID,
		Email:     parseEmail,
		TokenHash: parseTokenHash,
		ExpiresAt: parseExpiresAt,
	}); parseErr != nil {
		return "", parseErr
	}
	return parseRawToken, nil
}

// parseLogPasswordRecoveryAuditEvent emits one structured password-recovery audit event for operator diagnostics.
func (parseA *authManager) parseLogPasswordRecoveryAuditEvent(parseEvent string, parseOutcome string, parseUserID int64, parseEmail string, parseRequestIP string) {
	if parseA == nil || parseA.logger == nil {
		return
	}
	parseA.logger.Info(
		"auth recovery audit",
		slog.String("event", strings.TrimSpace(parseEvent)),
		slog.String("outcome", strings.TrimSpace(parseOutcome)),
		slog.Int64("user_id", parseUserID),
		slog.String("email_hash", parseBuildAuthAuditEmailHash(parseEmail)),
		slog.String("request_ip", strings.TrimSpace(parseRequestIP)),
	)
}

// parseIsPasswordResetRequestThrottled reports whether a password-reset request exceeds email/IP request budgets.
func (parseA *authManager) parseIsPasswordResetRequestThrottled(parseEmail string, parseRequestIP string, parseNow time.Time) (bool, error) {
	if parseA == nil || parseA.store == nil {
		return false, errPasswordRecoveryUnavailable
	}
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseSince := parseNow.UTC().Add(-1 * passwordResetRequestThrottleWindow)
	parseEmailRequestCount, parseErr := parseA.store.parseCountPasswordResetTokenRequestsByEmailSince(parseEmail, parseSince)
	if parseErr != nil {
		return false, parseErr
	}
	if parseEmailRequestCount >= passwordResetRequestThrottleMaxPerEmail {
		return true, nil
	}
	parseRequestIP = strings.TrimSpace(parseRequestIP)
	if parseRequestIP == "" || strings.EqualFold(parseRequestIP, "self-service") {
		return false, nil
	}
	parseIPRequestCount, parseErr := parseA.store.parseCountPasswordResetTokenRequestsByIPSince(parseRequestIP, parseSince)
	if parseErr != nil {
		return false, parseErr
	}
	if parseIPRequestCount >= passwordResetRequestThrottleMaxPerIP {
		return true, nil
	}
	return false, nil
}

// parseResolvePasswordResetConsumeOutcome classifies one failed reset-token consume into one stable audit outcome.
func (parseA *authManager) parseResolvePasswordResetConsumeOutcome(parseTokenHash string, parseNow time.Time) string {
	if parseA == nil || parseA.store == nil {
		return parsePasswordRecoveryAuditOutcomeInvalidToken
	}
	parseTokenRow, isParseTokenFound, parseErr := parseA.store.parseGetPasswordResetTokenByHash(parseTokenHash)
	if parseErr != nil || !isParseTokenFound {
		return parsePasswordRecoveryAuditOutcomeInvalidToken
	}
	if !strings.EqualFold(strings.TrimSpace(parseTokenRow.Status), "pending") {
		return parsePasswordRecoveryAuditOutcomeReplayDenied
	}
	parseExpiresAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseTokenRow.ExpiresAt))
	if parseErr != nil {
		return parsePasswordRecoveryAuditOutcomeInvalidToken
	}
	if parseNow.UTC().After(parseExpiresAt.UTC()) {
		return parsePasswordRecoveryAuditOutcomeExpired
	}
	return parsePasswordRecoveryAuditOutcomeReplayDenied
}

// parseBeginPasswordResetToken persists one pending password-reset token for one account when the email exists.
func (parseA *authManager) parseBeginPasswordResetToken(parseEmail string, parseRequestIP string) (string, error) {
	if parseA == nil || parseA.store == nil {
		return "", errPasswordRecoveryUnavailable
	}
	parseEmail = parseNormalizeAuthEmail(parseEmail)
	parseRequestIP = strings.TrimSpace(parseRequestIP)
	if parseEmail == "" {
		return "", errInvalidCredentials
	}
	parseNow := time.Now().UTC()
	isParseThrottled, parseErr := parseA.parseIsPasswordResetRequestThrottled(parseEmail, parseRequestIP, parseNow)
	if parseErr != nil {
		return "", parseErr
	}
	if isParseThrottled {
		parseA.parseLogPasswordRecoveryAuditEvent(parsePasswordRecoveryAuditEventRequest, parsePasswordRecoveryAuditOutcomeThrottled, 0, parseEmail, parseRequestIP)
		return "", nil
	}
	parseRecord, parseErr := parseA.store.getUserAuthByEmail(parseEmail)
	if parseErr != nil {
		parseA.parseLogPasswordRecoveryAuditEvent(parsePasswordRecoveryAuditEventRequest, parsePasswordRecoveryAuditOutcomeMissingUser, 0, parseEmail, parseRequestIP)
		return "", nil
	}
	parseRawToken := parseBuildOpaqueAuthFlowToken()
	parseTokenHash := parseBuildAuthFlowTokenHash(parseRawToken)
	parseExpiresAt := parseNow.Add(passwordResetTokenTTL).Format(time.RFC3339)
	if _, parseErr2 := parseA.store.parseCreatePasswordResetToken(parsePasswordResetTokenWrite{
		UserID:        parseRecord.ID,
		Email:         parseRecord.Email,
		TokenHash:     parseTokenHash,
		RequestedByIP: parseRequestIP,
		ExpiresAt:     parseExpiresAt,
	}); parseErr2 != nil {
		return "", parseErr2
	}
	parseA.parseLogPasswordRecoveryAuditEvent(parsePasswordRecoveryAuditEventRequest, parsePasswordRecoveryAuditOutcomeRequested, parseRecord.ID, parseRecord.Email, parseRequestIP)
	return parseRawToken, nil
}

// parseCompletePasswordResetWithToken consumes one pending reset token, updates password hash, and revokes active sessions.
func (parseA *authManager) parseCompletePasswordResetWithToken(parseResetToken, parseNewPassword string) error {
	if parseA == nil || parseA.store == nil {
		return errPasswordRecoveryUnavailable
	}
	parseResetToken = strings.TrimSpace(parseResetToken)
	parseNewPassword = strings.TrimSpace(parseNewPassword)
	if parseResetToken == "" || parseNewPassword == "" {
		return errInvalidCredentials
	}
	if len(parseNewPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	parseTokenHash := parseBuildAuthFlowTokenHash(parseResetToken)
	parseNow := time.Now().UTC()
	parseTokenRow, parseOk, parseErr := parseA.store.parseConsumePasswordResetToken(parseTokenHash, parseNow)
	if parseErr != nil {
		return errPasswordRecoveryUnavailable
	}
	if !parseOk || parseTokenRow.UserID <= 0 {
		parseConsumeOutcome := parseA.parseResolvePasswordResetConsumeOutcome(parseTokenHash, parseNow)
		parseA.parseLogPasswordRecoveryAuditEvent(parsePasswordRecoveryAuditEventConsume, parseConsumeOutcome, 0, "", "")
		return errInvalidCredentials
	}
	parsePasswordHash, parseErr := bcrypt.GenerateFromPassword([]byte(parseNewPassword), bcrypt.DefaultCost)
	if parseErr != nil {
		return errPasswordRecoveryUnavailable
	}
	if parseErr2 := parseA.store.parseUpdateUserPasswordHash(parseTokenRow.UserID, string(parsePasswordHash)); parseErr2 != nil {
		return errPasswordRecoveryUnavailable
	}
	if _, parseErr2 := parseA.store.parseIncrementAuthTokenVersion(parseTokenRow.UserID); parseErr2 != nil {
		return errPasswordRecoveryUnavailable
	}
	if parseErr2 := parseA.store.parseRevokeAuthSessionsByUser(parseTokenRow.UserID); parseErr2 != nil {
		return errPasswordRecoveryUnavailable
	}
	parseA.parseLogPasswordRecoveryAuditEvent(parsePasswordRecoveryAuditEventConsume, parsePasswordRecoveryAuditOutcomeConsumed, parseTokenRow.UserID, parseTokenRow.Email, parseTokenRow.RequestedByIP)
	return nil
}

// parseUpdatePasswordWithCurrentPassword rotates one password using current credentials via persisted reset-token lifecycle.
func (parseA *authManager) parseUpdatePasswordWithCurrentPassword(parseEmail, parseCurrentPassword, parseNewPassword string) error {
	parseEmail = parseNormalizeAuthEmail(parseEmail)
	if parseEmail == "" || strings.TrimSpace(parseCurrentPassword) == "" {
		return errInvalidCredentials
	}
	if _, parseErr := parseA.parseLogin(parseEmail, parseCurrentPassword); parseErr != nil {
		return parseErr
	}
	parseResetToken, parseErr := parseA.parseBeginPasswordResetToken(parseEmail, "self-service")
	if parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(parseResetToken) == "" {
		return errInvalidCredentials
	}
	return parseA.parseCompletePasswordResetWithToken(parseResetToken, parseNewPassword)
}
