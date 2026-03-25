package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func TestLoadFirstDotEnvAndRuntimeConfig(t *testing.T) {
	loaded := []string{}
	chosen := loadFirstDotEnv(func(path string) error {
		loaded = append(loaded, path)
		if path == "../.env" {
			return nil
		}
		return url.EscapeError(path)
	}, []string{".env", "../.env", "ignored.env"})
	if chosen != "../.env" {
		t.Fatalf("expected ../.env to be selected, got %q", chosen)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected loader to stop after first success, got %v", loaded)
	}
	if got := loadFirstDotEnv(func(string) error { return url.EscapeError("miss") }, []string{"a.env"}); got != "" {
		t.Fatalf("expected empty result when no env files load, got %q", got)
	}

	config := readServerRuntimeConfig(func(key string) string {
		switch key {
		case "OPENAI_API_KEY":
			return " openai-key "
		case "ANTHROPIC_API_KEY":
			return " anthropic-key "
		case "CHAT_MODEL":
			return "  "
		case "OPENAI_MODEL":
			return " gpt-5.4-mini "
		case "LISTEN_ADDR":
			return " 0.0.0.0:9999 "
		case "CHAT_DB_PATH":
			return " ./chat.db "
		case "CHAT_AUTH_SECRET":
			return " secret "
		default:
			return ""
		}
	})
	if config.openAIAPIKey != "openai-key" || config.anthropicAPIKey != "anthropic-key" {
		t.Fatalf("unexpected API key trimming: %+v", config)
	}
	if config.defaultModel != "gpt-5.4-mini" || config.addr != "0.0.0.0:9999" || config.dbPath != "./chat.db" || config.authSecret != "secret" {
		t.Fatalf("unexpected runtime config values: %+v", config)
	}

	defaultConfig := readServerRuntimeConfig(func(string) string { return "" })
	if defaultConfig.addr != "127.0.0.1:8095" || defaultConfig.dbPath != "examples/100-ai-chat-wizard/bin/runtime/chat_history.db" || defaultConfig.defaultModel != "" {
		t.Fatalf("unexpected default runtime config: %+v", defaultConfig)
	}
}

func TestChatServerAuthRPCs(t *testing.T) {
	store := newTestStore(t)
	auth := newAuthManager("test-secret", store, newTestLogger())
	server := newChatServiceServer("", "", modelGPT54Mini, store, newTestLogger())
	server.authManager = auth

	signupResp, err := server.Signup(context.Background(), &chatpb.SignupRequest{
		Email:       "startup@example.com",
		Password:    "password123",
		DisplayName: "Startup",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if signupResp.GetAuthToken() == "" || signupResp.GetEmail() != "startup@example.com" || signupResp.GetDisplayName() != "Startup" {
		t.Fatalf("unexpected signup response: %+v", signupResp)
	}

	_, err = server.Signup(context.Background(), &chatpb.SignupRequest{
		Email:    "startup@example.com",
		Password: "password123",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected duplicate signup to return AlreadyExists, got %v", status.Code(err))
	}

	_, err = server.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "startup@example.com",
		Password: "wrong",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected invalid login to return Unauthenticated, got %v", status.Code(err))
	}

	loginResp, err := server.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "startup@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loginResp.GetAuthToken() == "" || loginResp.GetEmail() != "startup@example.com" {
		t.Fatalf("unexpected login response: %+v", loginResp)
	}

	unauthSession, err := server.GetSession(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetSession unauthenticated: %v", err)
	}
	if unauthSession.GetAuthenticated() {
		t.Fatalf("expected empty session without metadata auth, got %+v", unauthSession)
	}

	authCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+loginResp.GetAuthToken()))
	session, err := server.GetSession(authCtx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetSession authenticated: %v", err)
	}
	if !session.GetAuthenticated() || session.GetEmail() != "startup@example.com" || session.GetDisplayName() != "Startup" {
		t.Fatalf("unexpected authenticated session: %+v", session)
	}

	refreshResp, err := server.RefreshSession(authCtx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("RefreshSession authenticated: %v", err)
	}
	if refreshResp.GetAuthToken() == "" || refreshResp.GetEmail() != "startup@example.com" || refreshResp.GetDisplayName() != "Startup" {
		t.Fatalf("unexpected refresh response: %+v", refreshResp)
	}

	_, err = server.RefreshSession(context.Background(), &emptypb.Empty{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected RefreshSession without auth to be unauthenticated, got %v", status.Code(err))
	}

	if _, err := server.Logout(authCtx, &emptypb.Empty{}); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

func TestProtectedShellHandler(t *testing.T) {
	fileServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("asset:" + r.URL.Path))
	})
	handler := protectedShellHandler(nil, fileServer)

	unauthResp := httptest.NewRecorder()
	handler.ServeHTTP(unauthResp, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	if unauthResp.Code != http.StatusOK || !strings.Contains(unauthResp.Body.String(), "Preparing chat runtime") {
		t.Fatalf("expected public shell content, got code=%d body=%q", unauthResp.Code, unauthResp.Body.String())
	}

	bootstrapReq := httptest.NewRequest(http.MethodGet, "http://example.com/chat-bootstrap.js", nil)
	bootstrapResp := httptest.NewRecorder()
	handler.ServeHTTP(bootstrapResp, bootstrapReq)
	if bootstrapResp.Code != http.StatusOK || !strings.Contains(bootstrapResp.Body.String(), "loadChatWasm") {
		t.Fatalf("expected bootstrap response, got code=%d body=%q", bootstrapResp.Code, bootstrapResp.Body.String())
	}

	deepLinkReq := httptest.NewRequest(http.MethodGet, "http://example.com/thread/42", nil)
	deepLinkResp := httptest.NewRecorder()
	handler.ServeHTTP(deepLinkResp, deepLinkReq)
	if deepLinkResp.Code != http.StatusOK || !strings.Contains(deepLinkResp.Body.String(), "Preparing chat runtime") {
		t.Fatalf("expected thread deep-link shell content, got code=%d body=%q", deepLinkResp.Code, deepLinkResp.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm", nil)
	assetResp := httptest.NewRecorder()
	handler.ServeHTTP(assetResp, assetReq)
	if assetResp.Code != http.StatusOK || assetResp.Body.String() != "asset:/app/chat.wasm" {
		t.Fatalf("expected file server fallback, got code=%d body=%q", assetResp.Code, assetResp.Body.String())
	}

	legacyAssetReq := httptest.NewRequest(http.MethodGet, "http://example.com/chat.wasm?br=true", nil)
	legacyAssetResp := httptest.NewRecorder()
	handler.ServeHTTP(legacyAssetResp, legacyAssetReq)
	if legacyAssetResp.Code != http.StatusOK || legacyAssetResp.Body.String() != "asset:/app/chat.wasm" {
		t.Fatalf("expected legacy wasm path rewrite, got code=%d body=%q", legacyAssetResp.Code, legacyAssetResp.Body.String())
	}
}
