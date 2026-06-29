package app

import (
	"context"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// TestBuildCustomerSafeErrorContractIncludesSupportReference verifies support-id based reference and log path derivation.
func TestBuildCustomerSafeErrorContractIncludesSupportReference(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-customer-safe-123",
		correlationIDMetadataKey, "corr-customer-safe-456",
	))
	parseContract := parseBuildCustomerSafeErrorContract(parseCtx, parseCustomerSafeErrorScopeDashboard, status.Error(codes.Unavailable, "store unavailable"))
	if parseContract.ParseScope != parseCustomerSafeErrorScopeDashboard {
		parseT.Fatalf("ParseScope = %q, want %q", parseContract.ParseScope, parseCustomerSafeErrorScopeDashboard)
	}
	if parseContract.ParseReferenceID == "" || parseContract.ParseReferenceID == parseSupportIDUnknown {
		parseT.Fatalf("ParseReferenceID = %q, want derived support id", parseContract.ParseReferenceID)
	}
	if !strings.HasPrefix(parseContract.ParseLogPath, "support.id=") {
		parseT.Fatalf("ParseLogPath = %q, want support.id lookup", parseContract.ParseLogPath)
	}
	if parseContract.ParseUserMessage != "Dashboard data is temporarily unavailable. Please refresh in a moment." {
		parseT.Fatalf("ParseUserMessage = %q", parseContract.ParseUserMessage)
	}
}

// TestWrapCustomerSafeRPCErrorAttachesTypedDetails verifies scoped RPC errors include one typed ErrorInfo payload.
func TestWrapCustomerSafeRPCErrorAttachesTypedDetails(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-customer-safe-123",
		correlationIDMetadataKey, "corr-customer-safe-456",
	))
	parseWrappedErr := parseWrapCustomerSafeRPCError(parseCtx, "/chat.ChatService/Send", status.Error(codes.Unavailable, "provider busy"), parseNewTestLogger())
	parseStatusErr := status.Convert(parseWrappedErr)
	if parseStatusErr.Code() != codes.Unavailable {
		parseT.Fatalf("status code = %v, want %v", parseStatusErr.Code(), codes.Unavailable)
	}
	if parseStatusErr.Message() != "We couldn't send your message right now. Please try again." {
		parseT.Fatalf("status message = %q", parseStatusErr.Message())
	}
	parseFoundContract := false
	for _, parseDetail := range parseStatusErr.Details() {
		parseErrorInfo, isParseErrorInfo := parseDetail.(*errdetails.ErrorInfo)
		if !isParseErrorInfo {
			continue
		}
		if parseErrorInfo.GetReason() != parseCustomerSafeErrorInfoReason || parseErrorInfo.GetDomain() != parseCustomerSafeErrorInfoDomain {
			continue
		}
		parseFoundContract = true
		if parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataScope] != parseCustomerSafeErrorScopeChat {
			parseT.Fatalf("metadata scope = %q, want %q", parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataScope], parseCustomerSafeErrorScopeChat)
		}
		if parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataReferenceID] == "" {
			parseT.Fatal("metadata reference_id empty")
		}
		if !strings.HasPrefix(parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataLogPath], "support.id=") {
			parseT.Fatalf("metadata log_path = %q", parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataLogPath])
		}
	}
	if !parseFoundContract {
		parseT.Fatal("expected one customer-safe ErrorInfo detail")
	}
}

// TestBuildCustomerSafeErrorUnaryInterceptorSkipsOutOfScopeMethods verifies non-customer methods are left unchanged.
func TestBuildCustomerSafeErrorUnaryInterceptorSkipsOutOfScopeMethods(parseT *testing.T) {
	parseInterceptor := parseBuildCustomerSafeErrorUnaryInterceptor(parseNewTestLogger())
	_, parseErr := parseInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/chat.ChatService/GetCatalogNamespace"}, func(context.Context, any) (any, error) {
		return nil, status.Error(codes.Unavailable, "service unavailable")
	})
	parseStatusErr := status.Convert(parseErr)
	if parseStatusErr.Code() != codes.Unavailable {
		parseT.Fatalf("status code = %v, want %v", parseStatusErr.Code(), codes.Unavailable)
	}
	if len(parseStatusErr.Details()) != 0 {
		parseT.Fatalf("details length = %d, want 0", len(parseStatusErr.Details()))
	}
}

// TestSettingsWriteRPCsNoStoreReturnTypedUnavailable verifies launch-critical settings writes fail closed with typed customer-safe unavailable contracts when persistence is unavailable.
func TestSettingsWriteRPCsNoStoreReturnTypedUnavailable(parseT *testing.T) {
	parseServer := &chatServer{
		defaultModel: modelGPT54Mini,
		logger:       parseNewTestLogger(),
		sessions:     map[string]*sessionState{},
		authUsers:    map[string]authUser{},
	}
	parseCtx := parseBindAuthUser(parseServer, "peer-customer-safe-settings", 77, "customer-safe-settings@example.com")

	_, parseUpsertErr := parseServer.UpsertUserMemory(parseCtx, &chatpb.UpsertUserMemoryRequest{
		Memory: &chatpb.UserMemory{Summary: "Stored summary"},
	})
	parseAssertTypedUnavailableContract(parseT, parseUpsertErr, parseCustomerSafeErrorScopeSettings)

	_, parseDeleteErr := parseServer.DeleteUserMemory(parseCtx, &chatpb.DeleteUserMemoryRequest{Key: "memory-key"})
	parseAssertTypedUnavailableContract(parseT, parseDeleteErr, parseCustomerSafeErrorScopeSettings)

	_, parsePromptErr := parseServer.SetCustomSystemPrompt(parseCtx, wrapperspb.String("Keep answers concise."))
	parseAssertTypedUnavailableContract(parseT, parsePromptErr, parseCustomerSafeErrorScopeSettings)
}

// parseAssertTypedUnavailableContract verifies one unavailable status includes one customer-safe ErrorInfo contract for the expected scope.
func parseAssertTypedUnavailableContract(parseT *testing.T, parseErr error, parseScope string) {
	parseT.Helper()
	parseStatusErr := status.Convert(parseErr)
	if parseStatusErr.Code() != codes.Unavailable {
		parseT.Fatalf("status code = %v, want %v", parseStatusErr.Code(), codes.Unavailable)
	}
	parseFoundContract := false
	for _, parseDetail := range parseStatusErr.Details() {
		parseErrorInfo, isParseErrorInfo := parseDetail.(*errdetails.ErrorInfo)
		if !isParseErrorInfo {
			continue
		}
		if parseErrorInfo.GetReason() != parseCustomerSafeErrorInfoReason || parseErrorInfo.GetDomain() != parseCustomerSafeErrorInfoDomain {
			continue
		}
		parseFoundContract = true
		if parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataScope] != parseScope {
			parseT.Fatalf("metadata scope = %q, want %q", parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataScope], parseScope)
		}
		if strings.TrimSpace(parseErrorInfo.GetMetadata()[parseCustomerSafeErrorMetadataReferenceID]) == "" {
			parseT.Fatal("metadata reference_id empty")
		}
	}
	if !parseFoundContract {
		parseT.Fatal("expected one customer-safe ErrorInfo detail")
	}
}
