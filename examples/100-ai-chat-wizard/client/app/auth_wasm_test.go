//go:build js && wasm

package app

import (
	"errors"
	"testing"

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
