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

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizationAndPromptHelpers(parseT *testing.T) {
	if parseGot := parseNormalizeSelectedModelID("gpt-5.4-mini-2026-03-17"); parseGot != modelGPT54Mini {
		parseT.Fatalf("normalizeSelectedModelID mismatch: %q", parseGot)
	}
	if parseGot2 := parseNormalizeSelectedModelID("custom-model"); parseGot2 != "custom-model" {
		parseT.Fatalf("expected custom model to pass through, got %q", parseGot2)
	}
	if parseGot3 := parseNormalizeSelectedToneID("unknown"); parseGot3 != defaultToneID {
		parseT.Fatalf("expected unknown tone to fall back, got %q", parseGot3)
	}
	if parseGot4 := parseNormalizeSelectedThinkingEffort("HIGH"); parseGot4 != "high" {
		parseT.Fatalf("expected thinking effort normalization, got %q", parseGot4)
	}
	parseMemories := []userMemoryRow{{Summary: "Prefers Neovim", Detail: "Uses it daily"}}
	if !strings.Contains(buildSystemPrompt("professional", "Call me captain.", parseMemories), toneInstructionByID["professional"]) {
		parseT.Fatal("expected system prompt to include tone instruction")
	}
	if !strings.Contains(buildSystemPrompt("friendly", "", parseMemories), toneInstructionByID["friendly"]) {
		parseT.Fatal("expected system prompt to include friendly tone instruction")
	}
	if !strings.Contains(buildSystemPrompt("professional", "Call me captain.", parseMemories), "Call me captain.") {
		parseT.Fatal("expected system prompt to include custom prompt")
	}
	if !strings.Contains(buildSystemPrompt("professional", "Call me captain.", parseMemories), "Prefers Neovim") {
		parseT.Fatal("expected system prompt to include user memory context")
	}
	parseTemplatedPrompt := buildSystemPrompt("professional", "Use remembered context:\n{{memories}}", parseMemories)
	if strings.Contains(parseTemplatedPrompt, "{{memories}}") {
		parseT.Fatal("expected memories placeholder to be resolved in custom prompt")
	}
	if strings.Count(parseTemplatedPrompt, "Prefers Neovim") != 1 {
		parseT.Fatalf("expected memory to appear once when custom prompt injects memories, got prompt %q", parseTemplatedPrompt)
	}
	parseDefaultPrompt := buildSystemPrompt("balanced", "", nil)
	if !strings.Contains(parseDefaultPrompt, "Current runtime context:") {
		parseT.Fatalf("expected default system prompt template block, got %q", parseDefaultPrompt)
	}
	if strings.Contains(parseDefaultPrompt, "{{date}}") || strings.Contains(parseDefaultPrompt, "{{time}}") || strings.Contains(parseDefaultPrompt, "{{memories}}") {
		parseT.Fatalf("expected default template placeholders to resolve, got %q", parseDefaultPrompt)
	}
	if !strings.Contains(parseDefaultPrompt, "- No stored memories yet.") {
		parseT.Fatalf("expected default system prompt to include empty-memory fallback, got %q", parseDefaultPrompt)
	}
	parseDefaultWithMemories := buildSystemPrompt("balanced", "", parseMemories)
	if strings.Count(parseDefaultWithMemories, "Prefers Neovim") != 1 {
		parseT.Fatalf("expected default prompt memory injection to be deduplicated, got %q", parseDefaultWithMemories)
	}
	if buildUserMemoryPromptBlock(nil) != "" {
		parseT.Fatal("expected empty memory block for nil memories")
	}
	parseFixedNow := time.Date(2026, time.March, 25, 14, 30, 45, 0, time.FixedZone("UTC-4", -4*60*60))
	parseResolvedTemplate := parseResolveSystemPromptTemplate("Date {{date}} Time {{time}}\n{{memories}}", "- Prefers Neovim", parseFixedNow)
	if strings.Contains(parseResolvedTemplate, "{{date}}") || strings.Contains(parseResolvedTemplate, "{{time}}") || strings.Contains(parseResolvedTemplate, "{{memories}}") {
		parseT.Fatalf("expected all template placeholders to be resolved, got %q", parseResolvedTemplate)
	}
	if !strings.Contains(parseResolvedTemplate, "2026-03-25") || !strings.Contains(parseResolvedTemplate, "14:30:45 -0400") {
		parseT.Fatalf("expected date/time substitution in resolved template, got %q", parseResolvedTemplate)
	}
	parseResolvedNoMemories := parseResolveSystemPromptTemplate("Memories:\n{{memories}}", "", parseFixedNow)
	if !strings.Contains(parseResolvedNoMemories, "- No stored memories yet.") {
		parseT.Fatalf("expected empty memory fallback in resolved template, got %q", parseResolvedNoMemories)
	}

	parseLongText := strings.Repeat("a", maxTTSScriptRunes+50)
	parseSanitized := parseSanitizeTextForTTS("# Heading\n```go\nfmt.Println(1)\n```\nVisit [site](https://example.com)\n> Quote\n" + parseLongText)
	if strings.Contains(parseSanitized, "fmt.Println") || strings.Contains(parseSanitized, "https://") {
		parseT.Fatalf("sanitizeTextForTTS left markdown artifacts: %q", parseSanitized)
	}
	if len([]rune(parseSanitized)) > maxTTSScriptRunes {
		parseT.Fatalf("sanitizeTextForTTS exceeded rune limit: %d", len([]rune(parseSanitized)))
	}
	if parseSanitizeTextForTTS("   ") != "" {
		parseT.Fatal("expected whitespace-only TTS input to sanitize to empty string")
	}
}

func TestCapabilityStatusAndAssetHelpers(parseT *testing.T) {
	parseCapabilityErr := &provider.UnsupportedCapabilityError{Capability: provider.CapabilitySpeech, Model: modelGPT54Mini, ProviderID: "fake"}
	parseStatusErr := parseCapabilityStatusError(parseCapabilityErr)
	if status.Code(parseStatusErr) != codes.FailedPrecondition {
		parseT.Fatalf("expected failed precondition, got %v", status.Code(parseStatusErr))
	}
	if parseCapabilityStatusError(errors.New("plain")) != nil {
		parseT.Fatal("expected plain error to return nil capability status")
	}

	parseRelativePath, parseOk := parseResolveRelativeAssetPath("/app/chat.wasm")
	if !parseOk || parseRelativePath != "app/chat.wasm" {
		parseT.Fatalf("unexpected relative asset path: ok=%v path=%q", parseOk, parseRelativePath)
	}
	if _, parseOk2 := parseResolveRelativeAssetPath("/../secret"); parseOk2 {
		parseT.Fatal("expected parent traversal path to be rejected")
	}
}

func TestBrotliServingAndShellEndpoints(parseT *testing.T) {
	parseRootDir := parseT.TempDir()
	parseBrotliDir := filepath.Join(parseRootDir, "app")
	if parseErr := os.MkdirAll(parseBrotliDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRootDir, "plain.txt"), []byte("plain-file"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile plain.txt: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseBrotliDir, "chat.wasm"), []byte("wasm-bytes"), 0o644); parseErr3 != nil {
		parseT.Fatalf("WriteFile chat.wasm: %v", parseErr3)
	}
	parseBrotliFile := filepath.Join(parseBrotliDir, "chat.wasm.br")
	if parseErr4 := os.WriteFile(parseBrotliFile, []byte("brotli-data"), 0o644); parseErr4 != nil {
		parseT.Fatalf("WriteFile: %v", parseErr4)
	}

	parseRequest := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=true", nil)
	parseWriter := httptest.NewRecorder()
	if !parseTryServeBrotliWASM(parseWriter, parseRequest, parseRootDir) {
		parseT.Fatal("expected tryServeBrotliWASM to serve a Brotli sidecar")
	}
	parseResponse := parseWriter.Result()
	if parseResponse.Header.Get("Content-Encoding") != "br" {
		parseT.Fatalf("expected Brotli encoding header, got %q", parseResponse.Header.Get("Content-Encoding"))
	}
	if parseResponse.Header.Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected Brotli wasm to disable caching, got %q", parseResponse.Header.Get("Cache-Control"))
	}

	parseDefaultBrotliRequest := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm", nil)
	if !parseTryServeBrotliWASM(httptest.NewRecorder(), parseDefaultBrotliRequest, parseRootDir) {
		parseT.Fatal("expected plain wasm request to serve Brotli by default")
	}

	parseExplicitRawRequest := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=false", nil)
	if parseTryServeBrotliWASM(httptest.NewRecorder(), parseExplicitRawRequest, parseRootDir) {
		parseT.Fatal("expected ?br=false to bypass Brotli helper")
	}

	parseFileServer := parseNewPrecompressedWASMFileServer(parseRootDir)
	parsePlainWriter := httptest.NewRecorder()
	parseFileServer.ServeHTTP(parsePlainWriter, httptest.NewRequest(http.MethodGet, "http://example.com/plain.txt", nil))
	if parsePlainWriter.Code != http.StatusOK || !strings.Contains(parsePlainWriter.Body.String(), "plain-file") {
		parseT.Fatalf("expected file server fallback to serve plain file, got code=%d body=%q", parsePlainWriter.Code, parsePlainWriter.Body.String())
	}

	parseBrotliWriter := httptest.NewRecorder()
	parseFileServer.ServeHTTP(parseBrotliWriter, httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=1", nil))
	if parseBrotliWriter.Result().Header.Get("Content-Encoding") != "br" {
		parseT.Fatalf("expected precompressed wasm file server to serve br artifact, got %q", parseBrotliWriter.Result().Header.Get("Content-Encoding"))
	}
	if parseBrotliWriter.Result().Header.Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected precompressed wasm to disable caching, got %q", parseBrotliWriter.Result().Header.Get("Cache-Control"))
	}

	parseRawWASMWriter := httptest.NewRecorder()
	parseFileServer.ServeHTTP(parseRawWASMWriter, httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=false", nil))
	if parseRawWASMWriter.Result().Header.Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected raw wasm to disable caching, got %q", parseRawWASMWriter.Result().Header.Get("Cache-Control"))
	}

	parseShellWriter := httptest.NewRecorder()
	parseServeChatShell(parseShellWriter, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	if !strings.Contains(parseShellWriter.Body.String(), "chat-bootstrap.js") {
		parseT.Fatal("expected shell HTML to include bootstrap script")
	}
	if parseShellWriter.Result().Header.Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected shell HTML to disable caching, got %q", parseShellWriter.Result().Header.Get("Cache-Control"))
	}
	if !strings.Contains(parseShellWriter.Body.String(), "/static/css/tailwind.css") {
		parseT.Fatal("expected shell HTML to include local Tailwind stylesheet")
	}
	if strings.Contains(parseShellWriter.Body.String(), "cdn.tailwindcss.com") {
		parseT.Fatal("expected shell HTML to avoid Tailwind CDN script")
	}

	parseBootstrapWriter := httptest.NewRecorder()
	parseServeChatBootstrapJS(parseBootstrapWriter, httptest.NewRequest(http.MethodGet, "http://example.com/chat-bootstrap.js", nil))
	if !strings.Contains(parseBootstrapWriter.Body.String(), "loadChatWasm") {
		parseT.Fatal("expected bootstrap JS to include wasm loader")
	}
	if !strings.Contains(parseBootstrapWriter.Body.String(), "normalizeStandaloneBracketMath") {
		parseT.Fatal("expected bootstrap JS to normalize standalone bracket math blocks")
	}
	if parseBootstrapWriter.Result().Header.Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected bootstrap JS to disable caching, got %q", parseBootstrapWriter.Result().Header.Get("Cache-Control"))
	}
	if parseContentType := parseBootstrapWriter.Result().Header.Get("Content-Type"); !strings.Contains(parseContentType, "application/javascript") {
		parseT.Fatalf("unexpected bootstrap content type: %q", parseContentType)
	}
}

func TestBrotliServingSkipsStaleSidecar(parseT *testing.T) {
	parseRootDir := parseT.TempDir()
	parseArtifactDir := filepath.Join(parseRootDir, "app")
	if parseErr := os.MkdirAll(parseArtifactDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll: %v", parseErr)
	}
	parseRawPath := filepath.Join(parseArtifactDir, "chat.wasm")
	parseBrotliPath := parseRawPath + ".br"
	if parseErr := os.WriteFile(parseRawPath, []byte("new-raw-wasm"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile raw: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseBrotliPath, []byte("stale-brotli"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile br: %v", parseErr)
	}
	parseOlderTime := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)
	parseNewerTime := parseOlderTime.Add(2 * time.Hour)
	if parseErr := os.Chtimes(parseBrotliPath, parseOlderTime, parseOlderTime); parseErr != nil {
		parseT.Fatalf("Chtimes br: %v", parseErr)
	}
	if parseErr := os.Chtimes(parseRawPath, parseNewerTime, parseNewerTime); parseErr != nil {
		parseT.Fatalf("Chtimes raw: %v", parseErr)
	}
	if parseShouldServeFreshBrotliArtifact(parseRootDir, filepath.ToSlash(filepath.Join("app", "chat.wasm"))) {
		parseT.Fatal("expected stale brotli sidecar to be rejected when raw wasm is newer")
	}
	parseRequest := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm", nil)
	parseWriter := httptest.NewRecorder()
	if parseTryServeBrotliWASM(parseWriter, parseRequest, parseRootDir) {
		parseT.Fatal("expected stale brotli helper path to skip serving")
	}
	parseFileServer := parseNewPrecompressedWASMFileServer(parseRootDir)
	parseRawWriter := httptest.NewRecorder()
	parseFileServer.ServeHTTP(parseRawWriter, parseRequest)
	if parseRawWriter.Result().Header.Get("Content-Encoding") != "" {
		parseT.Fatalf("expected stale brotli sidecar to fall back to raw wasm, got encoding %q", parseRawWriter.Result().Header.Get("Content-Encoding"))
	}
	if parseRawWriter.Body.String() != "new-raw-wasm" {
		parseT.Fatalf("expected raw wasm body after stale brotli fallback, got %q", parseRawWriter.Body.String())
	}
}

func TestGenerateAndSaveConversationTitle(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "title@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	parseFake := parseNewFakeProvider()
	parseFake.generateTitle = func(_ context.Context, parseReq provider.TitleRequest) (string, error) {
		return "Fresh title", nil
	}
	parseServer := parseNewFakeChatServer(store, parseFake)
	parseServer.parseGenerateAndSaveConversationTitle(parseUser.ID, parseConversationID, modelGPT54Mini, "Question", "Answer")

	parseDeadline := time.Now().Add(2 * time.Second)
	for {
		parseConversations, parseListErr := store.parseListConversations(parseUser.ID)
		if parseListErr != nil {
			parseT.Fatalf("listConversations: %v", parseListErr)
		}
		if len(parseConversations) == 1 && parseConversations[0].Preview == "Fresh title" {
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatal("timed out waiting for conversation title to be saved")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
