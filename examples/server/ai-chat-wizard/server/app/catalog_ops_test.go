package app

import (
	"context"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
)

// TestGetCatalogBootstrapRPC verifies typed bootstrap payloads return namespace rows with cache metadata.
func TestGetCatalogBootstrapRPC(parseT *testing.T) {
	parseServer := parseNewFakeChatServer(parseNewTestStore(parseT), parseNewFakeProvider())
	parseResp, parseErr := parseServer.GetCatalogBootstrap(context.Background(), &chatpb.GetCatalogBootstrapRequest{
		Locale:     "en",
		Namespaces: []string{"marketing", "chat"},
	})
	if parseErr != nil {
		parseT.Fatalf("GetCatalogBootstrap: %v", parseErr)
	}
	if parseResp.GetBundleVersion() != parseCatalogBundleVersion || parseResp.GetBundleHash() == "" {
		parseT.Fatalf("unexpected bootstrap metadata: %+v", parseResp)
	}
	if len(parseResp.GetCatalogs()) != 2 {
		parseT.Fatalf("expected two namespace payloads, got %+v", parseResp.GetCatalogs())
	}
	parseChatPayload := parseFindCatalogNamespace(parseResp.GetCatalogs(), "chat")
	if parseChatPayload == nil || len(parseChatPayload.GetMessages()) == 0 {
		parseT.Fatalf("expected chat namespace payload rows, got %+v", parseResp.GetCatalogs())
	}
}

// TestGetCatalogBootstrapRPCNotModified verifies bootstrap requests can short-circuit by known version/hash.
func TestGetCatalogBootstrapRPCNotModified(parseT *testing.T) {
	parseServer := parseNewFakeChatServer(parseNewTestStore(parseT), parseNewFakeProvider())
	parseInitialResp, parseErr := parseServer.GetCatalogBootstrap(context.Background(), &chatpb.GetCatalogBootstrapRequest{
		Locale: "en",
	})
	if parseErr != nil {
		parseT.Fatalf("GetCatalogBootstrap initial: %v", parseErr)
	}
	parseCachedResp, parseErr := parseServer.GetCatalogBootstrap(context.Background(), &chatpb.GetCatalogBootstrapRequest{
		Locale:       "en",
		KnownVersion: parseInitialResp.GetBundleVersion(),
		KnownHash:    parseInitialResp.GetBundleHash(),
	})
	if parseErr != nil {
		parseT.Fatalf("GetCatalogBootstrap cached: %v", parseErr)
	}
	if !parseCachedResp.GetIsNotModified() || len(parseCachedResp.GetCatalogs()) != 0 {
		parseT.Fatalf("expected not-modified bootstrap response, got %+v", parseCachedResp)
	}
}

// TestGetCatalogNamespaceRPC verifies per-namespace fetch and not-modified cache behavior.
func TestGetCatalogNamespaceRPC(parseT *testing.T) {
	parseServer := parseNewFakeChatServer(parseNewTestStore(parseT), parseNewFakeProvider())
	parseResp, parseErr := parseServer.GetCatalogNamespace(context.Background(), &chatpb.GetCatalogNamespaceRequest{
		Namespace: "chat",
		Locale:    "en",
	})
	if parseErr != nil {
		parseT.Fatalf("GetCatalogNamespace: %v", parseErr)
	}
	if parseResp.GetCatalog() == nil || parseResp.GetCatalog().GetContentHash() == "" {
		parseT.Fatalf("expected namespace payload with hash, got %+v", parseResp)
	}
	parseCachedResp, parseErr := parseServer.GetCatalogNamespace(context.Background(), &chatpb.GetCatalogNamespaceRequest{
		Namespace:    "chat",
		Locale:       "en",
		KnownVersion: parseResp.GetCatalog().GetVersion(),
		KnownHash:    parseResp.GetCatalog().GetContentHash(),
	})
	if parseErr != nil {
		parseT.Fatalf("GetCatalogNamespace cached: %v", parseErr)
	}
	if !parseCachedResp.GetIsNotModified() || parseCachedResp.GetCatalog() != nil {
		parseT.Fatalf("expected not-modified namespace response, got %+v", parseCachedResp)
	}
}

// TestGetCatalogNamespaceRPCLaunchTruthGuard verifies launch-truth filtering runs on namespace RPC responses.
func TestGetCatalogNamespaceRPCLaunchTruthGuard(parseT *testing.T) {
	parseServer := parseNewFakeChatServer(parseNewTestStore(parseT), parseNewFakeProvider())
	parseResp, parseErr := parseServer.GetCatalogNamespace(context.Background(), &chatpb.GetCatalogNamespaceRequest{
		Namespace: "marketing",
		Locale:    "en",
	})
	if parseErr != nil {
		parseT.Fatalf("GetCatalogNamespace marketing: %v", parseErr)
	}
	parseMessage := parseGetCatalogPayloadMessage(parseResp.GetCatalog().GetMessages(), "product.home.card.docqa.title")
	if parseMessage == "" {
		parseT.Fatal("expected launch-safe docqa title in marketing namespace")
	}
	if !strings.Contains(parseMessage, "planned for a later release") {
		parseT.Fatalf("expected launch-safe docqa replacement, got %q", parseMessage)
	}
}

// parseFindCatalogNamespace finds one namespace payload row by namespace key.
func parseFindCatalogNamespace(parseRows []*chatpb.CatalogNamespacePayload, parseNamespace string) *chatpb.CatalogNamespacePayload {
	for _, parseRow := range parseRows {
		if parseRow != nil && parseRow.GetNamespace() == parseNamespace {
			return parseRow
		}
	}
	return nil
}
