package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizationAndPromptHelpers(t *testing.T) {
	if got := normalizeSelectedModelID("gpt-5.4-mini-2026-03-17"); got != modelGPT54Mini {
		t.Fatalf("normalizeSelectedModelID mismatch: %q", got)
	}
	if got := normalizeSelectedModelID("custom-model"); got != "custom-model" {
		t.Fatalf("expected custom model to pass through, got %q", got)
	}
	if got := normalizeSelectedToneID("unknown"); got != defaultToneID {
		t.Fatalf("expected unknown tone to fall back, got %q", got)
	}
	if got := normalizeSelectedThinkingEffort("HIGH"); got != "high" {
		t.Fatalf("expected thinking effort normalization, got %q", got)
	}
	memories := []userMemoryRow{{Summary: "Prefers Neovim", Detail: "Uses it daily"}}
	if !strings.Contains(buildSystemPrompt("professional", "Call me captain.", memories), toneInstructionByID["professional"]) {
		t.Fatal("expected system prompt to include tone instruction")
	}
	if !strings.Contains(buildSystemPrompt("friendly", "", memories), toneInstructionByID["friendly"]) {
		t.Fatal("expected system prompt to include friendly tone instruction")
	}
	if !strings.Contains(buildSystemPrompt("professional", "Call me captain.", memories), "Call me captain.") {
		t.Fatal("expected system prompt to include custom prompt")
	}
	if !strings.Contains(buildSystemPrompt("professional", "Call me captain.", memories), "Prefers Neovim") {
		t.Fatal("expected system prompt to include user memory context")
	}
	templatedPrompt := buildSystemPrompt("professional", "Use remembered context:\n{{memories}}", memories)
	if strings.Contains(templatedPrompt, "{{memories}}") {
		t.Fatal("expected memories placeholder to be resolved in custom prompt")
	}
	if strings.Count(templatedPrompt, "Prefers Neovim") != 1 {
		t.Fatalf("expected memory to appear once when custom prompt injects memories, got prompt %q", templatedPrompt)
	}
	if buildUserMemoryPromptBlock(nil) != "" {
		t.Fatal("expected empty memory block for nil memories")
	}
	fixedNow := time.Date(2026, time.March, 25, 14, 30, 45, 0, time.FixedZone("UTC-4", -4*60*60))
	resolvedTemplate := resolveSystemPromptTemplate("Date {{date}} Time {{time}}\n{{memories}}", "- Prefers Neovim", fixedNow)
	if strings.Contains(resolvedTemplate, "{{date}}") || strings.Contains(resolvedTemplate, "{{time}}") || strings.Contains(resolvedTemplate, "{{memories}}") {
		t.Fatalf("expected all template placeholders to be resolved, got %q", resolvedTemplate)
	}
	if !strings.Contains(resolvedTemplate, "2026-03-25") || !strings.Contains(resolvedTemplate, "14:30:45 -0400") {
		t.Fatalf("expected date/time substitution in resolved template, got %q", resolvedTemplate)
	}
	resolvedNoMemories := resolveSystemPromptTemplate("Memories:\n{{memories}}", "", fixedNow)
	if !strings.Contains(resolvedNoMemories, "- No stored memories yet.") {
		t.Fatalf("expected empty memory fallback in resolved template, got %q", resolvedNoMemories)
	}

	longText := strings.Repeat("a", maxTTSScriptRunes+50)
	sanitized := sanitizeTextForTTS("# Heading\n```go\nfmt.Println(1)\n```\nVisit [site](https://example.com)\n> Quote\n" + longText)
	if strings.Contains(sanitized, "fmt.Println") || strings.Contains(sanitized, "https://") {
		t.Fatalf("sanitizeTextForTTS left markdown artifacts: %q", sanitized)
	}
	if len([]rune(sanitized)) > maxTTSScriptRunes {
		t.Fatalf("sanitizeTextForTTS exceeded rune limit: %d", len([]rune(sanitized)))
	}
	if sanitizeTextForTTS("   ") != "" {
		t.Fatal("expected whitespace-only TTS input to sanitize to empty string")
	}
}

func TestCapabilityStatusAndAssetHelpers(t *testing.T) {
	capabilityErr := &provider.UnsupportedCapabilityError{Capability: provider.CapabilitySpeech, Model: modelGPT54Mini, ProviderID: "fake"}
	statusErr := capabilityStatusError(capabilityErr)
	if status.Code(statusErr) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", status.Code(statusErr))
	}
	if capabilityStatusError(errors.New("plain")) != nil {
		t.Fatal("expected plain error to return nil capability status")
	}

	relativePath, ok := resolveRelativeAssetPath("/app/chat.wasm")
	if !ok || relativePath != "app/chat.wasm" {
		t.Fatalf("unexpected relative asset path: ok=%v path=%q", ok, relativePath)
	}
	if _, ok := resolveRelativeAssetPath("/../secret"); ok {
		t.Fatal("expected parent traversal path to be rejected")
	}
}

func TestBrotliServingAndShellEndpoints(t *testing.T) {
	rootDir := t.TempDir()
	brotliDir := filepath.Join(rootDir, "app")
	if err := os.MkdirAll(brotliDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "plain.txt"), []byte("plain-file"), 0o644); err != nil {
		t.Fatalf("WriteFile plain.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brotliDir, "chat.wasm"), []byte("wasm-bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile chat.wasm: %v", err)
	}
	brotliFile := filepath.Join(brotliDir, "chat.wasm.br")
	if err := os.WriteFile(brotliFile, []byte("brotli-data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=true", nil)
	writer := httptest.NewRecorder()
	if !tryServeBrotliWASM(writer, request, rootDir) {
		t.Fatal("expected tryServeBrotliWASM to serve a Brotli sidecar")
	}
	response := writer.Result()
	if response.Header.Get("Content-Encoding") != "br" {
		t.Fatalf("expected Brotli encoding header, got %q", response.Header.Get("Content-Encoding"))
	}

	defaultBrotliRequest := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm", nil)
	if !tryServeBrotliWASM(httptest.NewRecorder(), defaultBrotliRequest, rootDir) {
		t.Fatal("expected plain wasm request to serve Brotli by default")
	}

	explicitRawRequest := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=false", nil)
	if tryServeBrotliWASM(httptest.NewRecorder(), explicitRawRequest, rootDir) {
		t.Fatal("expected ?br=false to bypass Brotli helper")
	}

	fileServer := newPrecompressedWASMFileServer(rootDir)
	plainWriter := httptest.NewRecorder()
	fileServer.ServeHTTP(plainWriter, httptest.NewRequest(http.MethodGet, "http://example.com/plain.txt", nil))
	if plainWriter.Code != http.StatusOK || !strings.Contains(plainWriter.Body.String(), "plain-file") {
		t.Fatalf("expected file server fallback to serve plain file, got code=%d body=%q", plainWriter.Code, plainWriter.Body.String())
	}

	brotliWriter := httptest.NewRecorder()
	fileServer.ServeHTTP(brotliWriter, httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=1", nil))
	if brotliWriter.Result().Header.Get("Content-Encoding") != "br" {
		t.Fatalf("expected precompressed wasm file server to serve br artifact, got %q", brotliWriter.Result().Header.Get("Content-Encoding"))
	}

	shellWriter := httptest.NewRecorder()
	serveChatShell(shellWriter, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	if !strings.Contains(shellWriter.Body.String(), "chat-bootstrap.js") {
		t.Fatal("expected shell HTML to include bootstrap script")
	}
	if !strings.Contains(shellWriter.Body.String(), "/static/css/tailwind.css") {
		t.Fatal("expected shell HTML to include local Tailwind stylesheet")
	}
	if strings.Contains(shellWriter.Body.String(), "cdn.tailwindcss.com") {
		t.Fatal("expected shell HTML to avoid Tailwind CDN script")
	}

	bootstrapWriter := httptest.NewRecorder()
	serveChatBootstrapJS(bootstrapWriter, httptest.NewRequest(http.MethodGet, "http://example.com/chat-bootstrap.js", nil))
	if !strings.Contains(bootstrapWriter.Body.String(), "loadChatWasm") {
		t.Fatal("expected bootstrap JS to include wasm loader")
	}
	if !strings.Contains(bootstrapWriter.Body.String(), "normalizeStandaloneBracketMath") {
		t.Fatal("expected bootstrap JS to normalize standalone bracket math blocks")
	}
	if contentType := bootstrapWriter.Result().Header.Get("Content-Type"); !strings.Contains(contentType, "application/javascript") {
		t.Fatalf("unexpected bootstrap content type: %q", contentType)
	}
}

func TestGenerateAndSaveConversationTitle(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "title@example.com")
	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	fake := newFakeProvider()
	fake.generateTitle = func(_ context.Context, req provider.TitleRequest) (string, error) {
		return "Fresh title", nil
	}
	server := newFakeChatServer(store, fake)
	server.generateAndSaveConversationTitle(user.ID, conversationID, modelGPT54Mini, "Question", "Answer")

	deadline := time.Now().Add(2 * time.Second)
	for {
		conversations, listErr := store.listConversations(user.ID)
		if listErr != nil {
			t.Fatalf("listConversations: %v", listErr)
		}
		if len(conversations) == 1 && conversations[0].Preview == "Fresh title" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for conversation title to be saved")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
