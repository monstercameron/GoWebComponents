package app

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseCustomerSafeErrorInfoReason = "CUSTOMER_SAFE_ERROR_V1"
const parseCustomerSafeErrorInfoDomain = "relaydesk.customer_error"

const parseCustomerSafeErrorScopeAuth = "auth"
const parseCustomerSafeErrorScopeChat = "chat"
const parseCustomerSafeErrorScopeSettings = "settings"
const parseCustomerSafeErrorScopeDashboard = "dashboard"

const parseCustomerSafeErrorMetadataScope = "scope"
const parseCustomerSafeErrorMetadataMessage = "user_message"
const parseCustomerSafeErrorMetadataReferenceID = "reference_id"
const parseCustomerSafeErrorMetadataLogPath = "log_path"
const parseCustomerSafeErrorMetadataMessageKey = "message_key"
const parseCustomerSafeErrorMetadataSupportID = "support_id"
const parseCustomerSafeErrorMetadataRequestID = "request_id"
const parseCustomerSafeErrorMetadataCorrelationID = "correlation_id"

type parseCustomerSafeErrorContract struct {
	ParseScope         string
	ParseCode          codes.Code
	ParseUserMessage   string
	ParseReferenceID   string
	ParseLogPath       string
	ParseMessageKey    string
	ParseSupportID     string
	ParseRequestID     string
	ParseCorrelationID string
}

// parseBuildCustomerSafeErrorUnaryInterceptor returns one unary interceptor that applies customer-safe error contracts on scoped RPC failures.
func parseBuildCustomerSafeErrorUnaryInterceptor(parseLogger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(parseCtx context.Context, parseReq any, parseInfo *grpc.UnaryServerInfo, parseHandler grpc.UnaryHandler) (any, error) {
		parseResp, parseErr := parseHandler(parseCtx, parseReq)
		if parseErr == nil || parseInfo == nil {
			return parseResp, parseErr
		}
		return parseResp, parseWrapCustomerSafeRPCError(parseCtx, parseInfo.FullMethod, parseErr, parseLogger)
	}
}

// parseBuildCustomerSafeErrorStreamInterceptor returns one stream interceptor that applies customer-safe error contracts on scoped RPC failures.
func parseBuildCustomerSafeErrorStreamInterceptor(parseLogger *slog.Logger) grpc.StreamServerInterceptor {
	return func(parseSrv any, parseServerStream grpc.ServerStream, parseInfo *grpc.StreamServerInfo, parseHandler grpc.StreamHandler) error {
		parseErr := parseHandler(parseSrv, parseServerStream)
		if parseErr == nil || parseInfo == nil {
			return parseErr
		}
		return parseWrapCustomerSafeRPCError(parseServerStream.Context(), parseInfo.FullMethod, parseErr, parseLogger)
	}
}

// parseWrapCustomerSafeRPCError maps one RPC error into one typed customer-safe contract when the method scope is customer-visible.
func parseWrapCustomerSafeRPCError(parseCtx context.Context, parseMethod string, parseErr error, parseLogger *slog.Logger) error {
	if parseErr == nil {
		return nil
	}
	parseScope := parseResolveCustomerSafeErrorScope(parseMethod)
	if parseScope == "" {
		return parseErr
	}
	parseContract := parseBuildCustomerSafeErrorContract(parseCtx, parseScope, parseErr)
	parseWrappedErr := parseBuildCustomerSafeErrorStatus(parseContract)
	if parseLogger != nil {
		parseAttrs := []any{
			slog.String("rpc.method", strings.TrimSpace(parseMethod)),
			slog.String("error.scope", parseContract.ParseScope),
			slog.String("error.code", parseContract.ParseCode.String()),
			slog.String("error.message_key", parseContract.ParseMessageKey),
			slog.String("error.reference_id", parseContract.ParseReferenceID),
			slog.String("error.log_path", parseContract.ParseLogPath),
		}
		if parseContract.ParseSupportID != "" {
			parseAttrs = append(parseAttrs, slog.String("support.id", parseContract.ParseSupportID))
		}
		if parseContract.ParseRequestID != "" {
			parseAttrs = append(parseAttrs, slog.String("request.id", parseContract.ParseRequestID))
		}
		if parseContract.ParseCorrelationID != "" {
			parseAttrs = append(parseAttrs, slog.String("correlation.id", parseContract.ParseCorrelationID))
		}
		parseLogger.Warn("rpc failed with customer-safe contract", parseAttrs...)
	}
	return parseWrappedErr
}

// parseResolveCustomerSafeErrorScope maps one RPC method path to one user-facing failure scope.
func parseResolveCustomerSafeErrorScope(parseMethod string) string {
	parseMethod = strings.TrimSpace(parseMethod)
	switch parseMethod {
	case "/chat.ChatService/Send",
		"/chat.ChatService/ListConversations",
		"/chat.ChatService/ResolveConversationRoute",
		"/chat.ChatService/LoadConversation",
		"/chat.ChatService/DeleteConversation":
		return parseCustomerSafeErrorScopeChat
	case "/chat.ChatService/Login",
		"/chat.ChatService/Signup",
		"/chat.ChatService/GetSession",
		"/chat.ChatService/RefreshSession",
		"/chat.ChatService/Logout":
		return parseCustomerSafeErrorScopeAuth
	case "/chat.ChatService/SetUserName",
		"/chat.ChatService/GetUserName",
		"/chat.ChatService/ListUserMemories",
		"/chat.ChatService/UpsertUserMemory",
		"/chat.ChatService/DeleteUserMemory",
		"/chat.ChatService/SetSelectedModel",
		"/chat.ChatService/GetSelectedModel",
		"/chat.ChatService/SetSelectedTone",
		"/chat.ChatService/GetSelectedTone",
		"/chat.ChatService/SetSelectedThinkingEnabled",
		"/chat.ChatService/GetSelectedThinkingEnabled",
		"/chat.ChatService/SetSelectedThinkingEffort",
		"/chat.ChatService/GetSelectedThinkingEffort",
		"/chat.ChatService/SetCustomSystemPrompt",
		"/chat.ChatService/GetCustomSystemPrompt":
		return parseCustomerSafeErrorScopeSettings
	}
	if strings.Contains(parseMethod, "Admin") || strings.Contains(parseMethod, "Superuser") || strings.Contains(parseMethod, "WorkspaceAdmin") {
		return parseCustomerSafeErrorScopeDashboard
	}
	return ""
}

// parseBuildCustomerSafeErrorContract builds one typed customer-safe error contract from one context and one RPC error.
func parseBuildCustomerSafeErrorContract(parseCtx context.Context, parseScope string, parseErr error) parseCustomerSafeErrorContract {
	parseStatusErr, parseOk := status.FromError(parseErr)
	if !parseOk {
		parseStatusErr = status.New(codes.Internal, strings.TrimSpace(parseErr.Error()))
	}
	parseRawMessage := strings.TrimSpace(parseStatusErr.Message())
	parseSanitizedMessage := parseResolveSanitizedPublicMessage(parseStatusErr.Code(), parseRawMessage)
	parseUserMessage := parseResolveCustomerSafeErrorMessage(parseScope, parseStatusErr.Code(), parseSanitizedMessage)
	parseSupportExposure := parseResolveSupportIDExposure(parseCtx, "", "")
	parseReferenceID := parseResolveCustomerSafeErrorReferenceID(parseSupportExposure)
	parseLogPath := parseBuildCustomerSafeErrorLogPath(parseSupportExposure, parseReferenceID)
	return parseCustomerSafeErrorContract{
		ParseScope:         strings.TrimSpace(parseScope),
		ParseCode:          parseStatusErr.Code(),
		ParseUserMessage:   parseUserMessage,
		ParseReferenceID:   parseReferenceID,
		ParseLogPath:       parseLogPath,
		ParseMessageKey:    parseBuildErrorMessageKey(parseStatusErr.Code(), parseUserMessage),
		ParseSupportID:     strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID),
		ParseRequestID:     strings.TrimSpace(parseSupportExposure.ParseOperatorRequestID),
		ParseCorrelationID: strings.TrimSpace(parseSupportExposure.ParseOperatorCorrelationID),
	}
}

// parseBuildCustomerSafeErrorStatus attaches one typed error detail payload to one customer-safe gRPC status.
func parseBuildCustomerSafeErrorStatus(parseContract parseCustomerSafeErrorContract) error {
	if parseContract.ParseCode == codes.OK {
		parseContract.ParseCode = codes.Internal
	}
	if strings.TrimSpace(parseContract.ParseUserMessage) == "" {
		parseContract.ParseUserMessage = parseResolveCustomerSafeErrorFallback(parseContract.ParseScope)
	}
	parseStatusErr := status.New(parseContract.ParseCode, parseContract.ParseUserMessage)
	parseErrorInfo := &errdetails.ErrorInfo{
		Reason: parseCustomerSafeErrorInfoReason,
		Domain: parseCustomerSafeErrorInfoDomain,
		Metadata: map[string]string{
			parseCustomerSafeErrorMetadataScope:         strings.TrimSpace(parseContract.ParseScope),
			parseCustomerSafeErrorMetadataMessage:       strings.TrimSpace(parseContract.ParseUserMessage),
			parseCustomerSafeErrorMetadataReferenceID:   strings.TrimSpace(parseContract.ParseReferenceID),
			parseCustomerSafeErrorMetadataLogPath:       strings.TrimSpace(parseContract.ParseLogPath),
			parseCustomerSafeErrorMetadataMessageKey:    strings.TrimSpace(parseContract.ParseMessageKey),
			parseCustomerSafeErrorMetadataSupportID:     strings.TrimSpace(parseContract.ParseSupportID),
			parseCustomerSafeErrorMetadataRequestID:     strings.TrimSpace(parseContract.ParseRequestID),
			parseCustomerSafeErrorMetadataCorrelationID: strings.TrimSpace(parseContract.ParseCorrelationID),
		},
	}
	parseStatusWithDetails, parseErr := parseStatusErr.WithDetails(parseErrorInfo)
	if parseErr != nil {
		return parseStatusErr.Err()
	}
	return parseStatusWithDetails.Err()
}

// parseBuildStoreUnavailableRPCStatus builds one typed unavailable status for one store-backed RPC method with customer-safe correlation metadata.
func parseBuildStoreUnavailableRPCStatus(parseCtx context.Context, parseMethod string, parsePublicMessage string) error {
	parsePublicMessage = strings.TrimSpace(parsePublicMessage)
	if parsePublicMessage == "" {
		parsePublicMessage = "store unavailable"
	}
	return parseWrapCustomerSafeRPCError(parseCtx, parseMethod, status.Error(codes.Unavailable, parsePublicMessage), nil)
}

// parseResolveCustomerSafeErrorMessage resolves one customer-safe message per product scope and error class.
func parseResolveCustomerSafeErrorMessage(parseScope string, parseCode codes.Code, parseSanitizedMessage string) string {
	parseSanitizedMessage = strings.TrimSpace(parseSanitizedMessage)
	isParseGenericMessage := parseSanitizedMessage == "" ||
		parseSanitizedMessage == parsePublicErrorGenericMessage ||
		parseSanitizedMessage == parsePublicErrorStorageMessage ||
		parseSanitizedMessage == parsePublicErrorProviderMessage ||
		parseSanitizedMessage == "service unavailable; retry later"
	if parseScope == parseCustomerSafeErrorScopeAuth {
		switch parseCode {
		case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unavailable, codes.DeadlineExceeded:
			return parseResolveCustomerSafeErrorFallback(parseScope)
		default:
			if !isParseGenericMessage {
				return parseSanitizedMessage
			}
			return parseResolveCustomerSafeErrorFallback(parseScope)
		}
	}
	switch parseCode {
	case codes.Unauthenticated:
		return "Your session expired. Please sign in again."
	case codes.PermissionDenied:
		if parseScope == parseCustomerSafeErrorScopeDashboard {
			return "You do not have permission to view dashboard data."
		}
		return "You do not have permission to complete that action."
	case codes.InvalidArgument, codes.FailedPrecondition, codes.NotFound, codes.AlreadyExists, codes.ResourceExhausted, codes.Canceled:
		if !isParseGenericMessage {
			return parseSanitizedMessage
		}
		return parseResolveCustomerSafeErrorFallback(parseScope)
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unavailable, codes.DeadlineExceeded:
		return parseResolveCustomerSafeErrorFallback(parseScope)
	default:
		if !isParseGenericMessage {
			return parseSanitizedMessage
		}
		return parseResolveCustomerSafeErrorFallback(parseScope)
	}
}

// parseResolveCustomerSafeErrorFallback returns one stable fallback message per customer-visible surface.
func parseResolveCustomerSafeErrorFallback(parseScope string) string {
	switch strings.TrimSpace(parseScope) {
	case parseCustomerSafeErrorScopeChat:
		return "We couldn't send your message right now. Please try again."
	case parseCustomerSafeErrorScopeSettings:
		return "We couldn't save your settings right now. Please try again."
	case parseCustomerSafeErrorScopeDashboard:
		return "Dashboard data is temporarily unavailable. Please refresh in a moment."
	case parseCustomerSafeErrorScopeAuth:
		return "Could not start your session right now."
	default:
		return "Something went wrong. Please try again."
	}
}

// parseResolveCustomerSafeErrorReferenceID picks one stable user-visible reference id from support/request/correlation identifiers.
func parseResolveCustomerSafeErrorReferenceID(parseSupportExposure parseSupportIDExposure) string {
	parseSupportID := strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID)
	if parseSupportID != "" && parseSupportID != parseSupportIDUnknown {
		return parseSupportID
	}
	if parseRequestID := strings.TrimSpace(parseSupportExposure.ParseOperatorRequestID); parseRequestID != "" {
		return parseRequestID
	}
	if parseCorrelationID := strings.TrimSpace(parseSupportExposure.ParseOperatorCorrelationID); parseCorrelationID != "" {
		return parseCorrelationID
	}
	if parseSupportID != "" {
		return parseSupportID
	}
	return parseSupportIDUnknown
}

// parseBuildCustomerSafeErrorLogPath returns one operator log lookup path that support can search with the exposed reference id.
func parseBuildCustomerSafeErrorLogPath(parseSupportExposure parseSupportIDExposure, parseReferenceID string) string {
	parseReferenceID = strings.TrimSpace(parseReferenceID)
	if parseReferenceID == "" {
		parseReferenceID = parseResolveCustomerSafeErrorReferenceID(parseSupportExposure)
	}
	parseSupportID := strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID)
	if parseSupportID != "" && parseSupportID != parseSupportIDUnknown {
		return "support.id=" + parseSupportID
	}
	if parseRequestID := strings.TrimSpace(parseSupportExposure.ParseOperatorRequestID); parseRequestID != "" {
		return "request.id=" + parseRequestID
	}
	if parseCorrelationID := strings.TrimSpace(parseSupportExposure.ParseOperatorCorrelationID); parseCorrelationID != "" {
		return "correlation.id=" + parseCorrelationID
	}
	return "support.id=" + parseReferenceID
}
