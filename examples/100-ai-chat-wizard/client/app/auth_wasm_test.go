//go:build js && wasm

package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthErrorMessageUsesFriendlyProductText(parseT *testing.T) {
	parseTests := []struct {
		name string
		mode string
		err  error
		want string
	}{
		{
			name: "login invalid credentials",
			mode: authModeLogin,
			err:  status.Error(codes.Unauthenticated, "invalid email or password"),
			want: "That email and password didn't match. Try again.",
		},
		{
			name: "signup duplicate account",
			mode: authModeSignup,
			err:  status.Error(codes.AlreadyExists, "an account with that email already exists"),
			want: "An account with that email already exists. Sign in instead or use another email.",
		},
		{
			name: "signup short password",
			mode: authModeSignup,
			err:  status.Error(codes.InvalidArgument, "password must be at least 8 characters"),
			want: "Use at least 8 characters for your password.",
		},
		{
			name: "session expired",
			mode: authModeLogin,
			err:  status.Error(codes.Unauthenticated, "authenticated user no longer exists; sign in again"),
			want: authExpiredMessage,
		},
		{
			name: "service unavailable",
			mode: authModeLogin,
			err:  status.Error(codes.Unavailable, "provider busy"),
			want: "The service is temporarily unavailable. Try again in a moment.",
		},
		{
			name: "plain rpc string gets sanitized",
			mode: authModeLogin,
			err:  errors.New("rpc error: code = Internal desc = issue auth token"),
			want: "Could not start your session right now.",
		},
		{
			name: "blank fallback uses mode specific text",
			mode: authModeSignup,
			err:  errors.New("   "),
			want: "Could not create your account right now.",
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := parseAuthErrorMessage(parseTest.mode, parseTest.err); parseGot != parseTest.want {
				parseT2.Fatalf("authErrorMessage() = %q, want %q", parseGot, parseTest.want)
			}
		})
	}
}

func TestAuthTokenExpiryHelpers(parseT *testing.T) {
	parseNow := time.Unix(1_900_000_000, 0).UTC()
	parseBuildToken := func(parseExpiry time.Time) string {
		parseHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		parsePayload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, parseExpiry.Unix())))
		return parseHeader + "." + parsePayload + ".signature"
	}

	parseFutureToken := parseBuildToken(parseNow.Add(45 * time.Minute))
	parseFutureExpiry, parseFutureOk := parseAuthTokenExpiry(parseFutureToken)
	if !parseFutureOk {
		parseT.Fatal("expected future token expiry to parse")
	}
	if parseFutureExpiry.Unix() != parseNow.Add(45*time.Minute).Unix() {
		parseT.Fatalf("future expiry unix = %d, want %d", parseFutureExpiry.Unix(), parseNow.Add(45*time.Minute).Unix())
	}
	if parseAuthTokenExpiresWithin(parseFutureToken, 15*time.Minute, parseNow) {
		parseT.Fatal("expected token outside refresh window to report false")
	}

	parseNearExpiryToken := parseBuildToken(parseNow.Add(5 * time.Minute))
	if !parseAuthTokenExpiresWithin(parseNearExpiryToken, 15*time.Minute, parseNow) {
		parseT.Fatal("expected token inside refresh window to report true")
	}

	parseMalformedToken := "not-a-jwt"
	if _, parseOk := parseAuthTokenExpiry(parseMalformedToken); parseOk {
		parseT.Fatal("expected malformed token expiry parse to fail")
	}
	if !parseAuthTokenExpiresWithin(parseMalformedToken, 15*time.Minute, parseNow) {
		parseT.Fatal("expected malformed token expiry check to fail closed")
	}
}
