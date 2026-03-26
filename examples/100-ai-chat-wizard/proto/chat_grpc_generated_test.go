package chatpb

import (
	"context"
	"errors"
	"io"
	"testing"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

type fakeClientConn struct {
	invokeErr    error
	newStreamErr error
	stream       grpc.ClientStream
	invoked      []string
	streamed     []string
}

func (parseF *fakeClientConn) Invoke(parseCtx context.Context, parseMethod string, parseArgs interface{}, parseReply interface{}, parseOpts ...grpc.CallOption) error {
	parseF.invoked = append(parseF.invoked, parseMethod)
	return parseF.invokeErr
}

func (parseF *fakeClientConn) NewStream(parseCtx context.Context, parseDesc *grpc.StreamDesc, parseMethod string, parseOpts ...grpc.CallOption) (grpc.ClientStream, error) {
	parseF.streamed = append(parseF.streamed, parseMethod)
	if parseF.newStreamErr != nil {
		return nil, parseF.newStreamErr
	}
	return parseF.stream, nil
}

type fakeClientStream struct {
	sendErr    error
	closeErr   error
	recvErr    error
	sendCount  int
	recvCount  int
	lastSend   interface{}
	trailerMD  metadata.MD
	contextRef context.Context
}

func (parseF *fakeClientStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (parseF *fakeClientStream) Trailer() metadata.MD         { return parseF.trailerMD }
func (parseF *fakeClientStream) Context() context.Context {
	if parseF.contextRef == nil {
		return context.Background()
	}
	return parseF.contextRef
}
func (parseF *fakeClientStream) CloseSend() error { return parseF.closeErr }
func (parseF *fakeClientStream) SendMsg(parseM interface{}) error {
	parseF.sendCount++
	parseF.lastSend = parseM
	return parseF.sendErr
}
func (parseF *fakeClientStream) RecvMsg(parseM interface{}) error {
	parseF.recvCount++
	if parseF.recvErr != nil {
		return parseF.recvErr
	}
	if parseF.recvCount > 1 {
		return io.EOF
	}
	switch parseChunk := parseM.(type) {
	case *ChatChunk:
		parseChunk.Done = true
	case *SynthesizeSpeechChunk:
		parseChunk.Done = true
		parseChunk.AudioChunk = []byte{1}
	}
	return nil
}

type fakeRegistrar struct {
	desc grpc.ServiceDesc
	srv  interface{}
}

func (parseF *fakeRegistrar) RegisterService(parseDesc *grpc.ServiceDesc, parseSrv interface{}) {
	parseF.desc = *parseDesc
	parseF.srv = parseSrv
}

type fakeServerStream struct {
	ctx      context.Context
	recvErr  error
	recvOnce bool
}

func (parseF *fakeServerStream) SetHeader(parseMd metadata.MD) error  { return nil }
func (parseF *fakeServerStream) SendHeader(parseMd metadata.MD) error { return nil }
func (parseF *fakeServerStream) SetTrailer(parseMd metadata.MD)       {}
func (parseF *fakeServerStream) Context() context.Context {
	if parseF.ctx == nil {
		return context.Background()
	}
	return parseF.ctx
}
func (parseF *fakeServerStream) SendMsg(parseM interface{}) error { return nil }
func (parseF *fakeServerStream) RecvMsg(parseM interface{}) error {
	if parseF.recvErr != nil {
		return parseF.recvErr
	}
	if parseF.recvOnce {
		return io.EOF
	}
	parseF.recvOnce = true
	return nil
}

func TestGeneratedChatServiceClientUnaryAndStreamMethods(parseT *testing.T) {
	parseStream := &fakeClientStream{}
	parseConn := &fakeClientConn{stream: parseStream}
	parseClient := NewChatServiceClient(parseConn)
	parseCtx := context.Background()

	if _, parseErr := parseClient.Signup(parseCtx, &SignupRequest{}); parseErr != nil {
		parseT.Fatalf("Signup: %v", parseErr)
	}
	if _, parseErr2 := parseClient.Login(parseCtx, &LoginRequest{}); parseErr2 != nil {
		parseT.Fatalf("Login: %v", parseErr2)
	}
	if _, parseErr3 := parseClient.Logout(parseCtx, &emptypb.Empty{}); parseErr3 != nil {
		parseT.Fatalf("Logout: %v", parseErr3)
	}
	if _, parseErr4 := parseClient.GetSession(parseCtx, &emptypb.Empty{}); parseErr4 != nil {
		parseT.Fatalf("GetSession: %v", parseErr4)
	}
	if _, parseErr5 := parseClient.RefreshSession(parseCtx, &emptypb.Empty{}); parseErr5 != nil {
		parseT.Fatalf("RefreshSession: %v", parseErr5)
	}
	if _, parseErr6 := parseClient.ListConversations(parseCtx, &ListConversationsRequest{}); parseErr6 != nil {
		parseT.Fatalf("ListConversations: %v", parseErr6)
	}
	if _, parseErr7 := parseClient.ResolveConversationRoute(parseCtx, &ResolveConversationRouteRequest{}); parseErr7 != nil {
		parseT.Fatalf("ResolveConversationRoute: %v", parseErr7)
	}
	if _, parseErr8 := parseClient.LoadConversation(parseCtx, &LoadConversationRequest{}); parseErr8 != nil {
		parseT.Fatalf("LoadConversation: %v", parseErr8)
	}
	if _, parseErr9 := parseClient.DeleteConversation(parseCtx, &DeleteConversationRequest{}); parseErr9 != nil {
		parseT.Fatalf("DeleteConversation: %v", parseErr9)
	}
	if _, parseErr10 := parseClient.SetUserName(parseCtx, &SetUserNameRequest{}); parseErr10 != nil {
		parseT.Fatalf("SetUserName: %v", parseErr10)
	}
	if _, parseErr11 := parseClient.GetUserName(parseCtx, &GetUserNameRequest{}); parseErr11 != nil {
		parseT.Fatalf("GetUserName: %v", parseErr11)
	}
	if _, parseErr12 := parseClient.ListUserMemories(parseCtx, &ListUserMemoriesRequest{}); parseErr12 != nil {
		parseT.Fatalf("ListUserMemories: %v", parseErr12)
	}
	if _, parseErr13 := parseClient.UpsertUserMemory(parseCtx, &UpsertUserMemoryRequest{}); parseErr13 != nil {
		parseT.Fatalf("UpsertUserMemory: %v", parseErr13)
	}
	if _, parseErr14 := parseClient.DeleteUserMemory(parseCtx, &DeleteUserMemoryRequest{}); parseErr14 != nil {
		parseT.Fatalf("DeleteUserMemory: %v", parseErr14)
	}
	if _, parseErr15 := parseClient.ListModelOptions(parseCtx, &ListModelOptionsRequest{}); parseErr15 != nil {
		parseT.Fatalf("ListModelOptions: %v", parseErr15)
	}
	if _, parseErr16 := parseClient.SetSelectedModel(parseCtx, wrapperspb.String("gpt-oss-120b")); parseErr16 != nil {
		parseT.Fatalf("SetSelectedModel: %v", parseErr16)
	}
	if _, parseErr17 := parseClient.GetSelectedModel(parseCtx, &emptypb.Empty{}); parseErr17 != nil {
		parseT.Fatalf("GetSelectedModel: %v", parseErr17)
	}
	if _, parseErr18 := parseClient.SetSelectedTone(parseCtx, wrapperspb.String("concise")); parseErr18 != nil {
		parseT.Fatalf("SetSelectedTone: %v", parseErr18)
	}
	if _, parseErr19 := parseClient.GetSelectedTone(parseCtx, &emptypb.Empty{}); parseErr19 != nil {
		parseT.Fatalf("GetSelectedTone: %v", parseErr19)
	}
	if _, parseErr20 := parseClient.SetSelectedThinkingEnabled(parseCtx, wrapperspb.Bool(true)); parseErr20 != nil {
		parseT.Fatalf("SetSelectedThinkingEnabled: %v", parseErr20)
	}
	if _, parseErr21 := parseClient.GetSelectedThinkingEnabled(parseCtx, &emptypb.Empty{}); parseErr21 != nil {
		parseT.Fatalf("GetSelectedThinkingEnabled: %v", parseErr21)
	}
	if _, parseErr22 := parseClient.SetSelectedThinkingEffort(parseCtx, wrapperspb.String("medium")); parseErr22 != nil {
		parseT.Fatalf("SetSelectedThinkingEffort: %v", parseErr22)
	}
	if _, parseErr23 := parseClient.GetSelectedThinkingEffort(parseCtx, &emptypb.Empty{}); parseErr23 != nil {
		parseT.Fatalf("GetSelectedThinkingEffort: %v", parseErr23)
	}
	if _, parseErr24 := parseClient.SetCustomSystemPrompt(parseCtx, wrapperspb.String("You are practical.")); parseErr24 != nil {
		parseT.Fatalf("SetCustomSystemPrompt: %v", parseErr24)
	}
	if _, parseErr25 := parseClient.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{}); parseErr25 != nil {
		parseT.Fatalf("GetCustomSystemPrompt: %v", parseErr25)
	}

	parseSendStream, parseErr26 := parseClient.Send(parseCtx, &SendRequest{})
	if parseErr26 != nil {
		parseT.Fatalf("Send: %v", parseErr26)
	}
	if parseSendStream == nil {
		parseT.Fatal("Send stream should not be nil")
	}
	if _, parseErr27 := parseSendStream.Recv(); parseErr27 != nil && !errors.Is(parseErr27, io.EOF) {
		parseT.Fatalf("Send stream recv: %v", parseErr27)
	}

	parseSpeechStream, parseErr26 := parseClient.SynthesizeSpeech(parseCtx, &SynthesizeSpeechRequest{})
	if parseErr26 != nil {
		parseT.Fatalf("SynthesizeSpeech: %v", parseErr26)
	}
	if parseSpeechStream == nil {
		parseT.Fatal("SynthesizeSpeech stream should not be nil")
	}
	if _, parseErr28 := parseSpeechStream.Recv(); parseErr28 != nil && !errors.Is(parseErr28, io.EOF) {
		parseT.Fatalf("SynthesizeSpeech stream recv: %v", parseErr28)
	}
}

func TestGeneratedChatServiceClientErrorPaths(parseT *testing.T) {
	parseCtx := context.Background()

	parseInvokeErr := errors.New("invoke failed")
	parseClient := NewChatServiceClient(&fakeClientConn{invokeErr: parseInvokeErr, stream: &fakeClientStream{}})
	if _, parseErr := parseClient.Signup(parseCtx, &SignupRequest{}); !errors.Is(parseErr, parseInvokeErr) {
		parseT.Fatalf("expected invoke error, got %v", parseErr)
	}

	parseNewStreamErr := errors.New("new stream failed")
	parseClient = NewChatServiceClient(&fakeClientConn{newStreamErr: parseNewStreamErr})
	if _, parseErr2 := parseClient.Send(parseCtx, &SendRequest{}); !errors.Is(parseErr2, parseNewStreamErr) {
		parseT.Fatalf("expected new stream error for Send, got %v", parseErr2)
	}
	if _, parseErr3 := parseClient.SynthesizeSpeech(parseCtx, &SynthesizeSpeechRequest{}); !errors.Is(parseErr3, parseNewStreamErr) {
		parseT.Fatalf("expected new stream error for SynthesizeSpeech, got %v", parseErr3)
	}

	parseSendErr := errors.New("send failed")
	parseClient = NewChatServiceClient(&fakeClientConn{stream: &fakeClientStream{sendErr: parseSendErr}})
	if _, parseErr4 := parseClient.Send(parseCtx, &SendRequest{}); !errors.Is(parseErr4, parseSendErr) {
		parseT.Fatalf("expected send error, got %v", parseErr4)
	}

	parseCloseErr := errors.New("close failed")
	parseClient = NewChatServiceClient(&fakeClientConn{stream: &fakeClientStream{closeErr: parseCloseErr}})
	if _, parseErr5 := parseClient.SynthesizeSpeech(parseCtx, &SynthesizeSpeechRequest{}); !errors.Is(parseErr5, parseCloseErr) {
		parseT.Fatalf("expected close-send error, got %v", parseErr5)
	}
}

func TestRegisterChatServiceServer(parseT *testing.T) {
	var parseRegistrar fakeRegistrar
	RegisterChatServiceServer(&parseRegistrar, UnimplementedChatServiceServer{})
	if parseRegistrar.desc.ServiceName != "chat.v1.ChatService" {
		parseT.Fatalf("expected service name chat.v1.ChatService, got %q", parseRegistrar.desc.ServiceName)
	}
	if parseRegistrar.srv == nil {
		parseT.Fatal("expected registered server implementation")
	}
}

func TestGeneratedUnaryHandlersDecodeInterceptorAndServerPaths(parseT *testing.T) {
	parseSrv := UnimplementedChatServiceServer{}
	parseDecodeErr := errors.New("decode failed")

	for _, parseMethod := range ChatService_ServiceDesc.Methods {
		parseT.Run(parseMethod.MethodName+"_decode_error", func(parseT2 *testing.T) {
			_, parseErr := parseMethod.Handler(parseSrv, context.Background(), func(parseV interface{}) error { return parseDecodeErr }, nil)
			if !errors.Is(parseErr, parseDecodeErr) {
				parseT2.Fatalf("expected decode error, got %v", parseErr)
			}
		})

		parseT.Run(parseMethod.MethodName+"_no_interceptor", func(parseT3 *testing.T) {
			_, parseErr2 := parseMethod.Handler(parseSrv, context.Background(), func(parseV2 interface{}) error { return nil }, nil)
			if status.Code(parseErr2) != codes.Unimplemented {
				parseT3.Fatalf("expected unimplemented status, got %v", parseErr2)
			}
		})

		parseT.Run(parseMethod.MethodName+"_with_interceptor", func(parseT4 *testing.T) {
			var isSeenInfo bool
			_, parseErr3 := parseMethod.Handler(
				parseSrv,
				context.Background(),
				func(parseV3 interface{}) error { return nil },
				func(parseCtx context.Context, parseReq interface{}, parseInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					isSeenInfo = true
					if parseInfo == nil || parseInfo.FullMethod == "" {
						parseT4.Fatalf("expected unary info with full method for %s", parseMethod.MethodName)
					}
					return handler(parseCtx, parseReq)
				},
			)
			if !isSeenInfo {
				parseT4.Fatalf("expected interceptor execution for %s", parseMethod.MethodName)
			}
			if status.Code(parseErr3) != codes.Unimplemented {
				parseT4.Fatalf("expected unimplemented status, got %v", parseErr3)
			}
		})
	}
}

func TestGeneratedStreamHandlersRecvAndServerPaths(parseT *testing.T) {
	parseSrv := UnimplementedChatServiceServer{}
	parseRecvErr := errors.New("recv failed")

	for _, parseStream := range ChatService_ServiceDesc.Streams {
		parseT.Run(parseStream.StreamName+"_recv_error", func(parseT2 *testing.T) {
			parseErr := parseStream.Handler(parseSrv, &fakeServerStream{recvErr: parseRecvErr})
			if !errors.Is(parseErr, parseRecvErr) {
				parseT2.Fatalf("expected recv error, got %v", parseErr)
			}
		})

		parseT.Run(parseStream.StreamName+"_unimplemented", func(parseT3 *testing.T) {
			parseErr2 := parseStream.Handler(parseSrv, &fakeServerStream{})
			if status.Code(parseErr2) != codes.Unimplemented {
				parseT3.Fatalf("expected unimplemented status, got %v", parseErr2)
			}
		})
	}
}

func TestUnimplementedChatServiceServerMethodsReturnUnimplemented(parseT *testing.T) {
	parseSrv := UnimplementedChatServiceServer{}
	parseCtx := context.Background()

	parseUnaryChecks := []struct {
		name string
		call func() error
	}{
		{"Signup", func() error { _, parseErr := parseSrv.Signup(parseCtx, &SignupRequest{}); return parseErr }},
		{"Login", func() error { _, parseErr2 := parseSrv.Login(parseCtx, &LoginRequest{}); return parseErr2 }},
		{"Logout", func() error { _, parseErr3 := parseSrv.Logout(parseCtx, &emptypb.Empty{}); return parseErr3 }},
		{"GetSession", func() error { _, parseErr4 := parseSrv.GetSession(parseCtx, &emptypb.Empty{}); return parseErr4 }},
		{"RefreshSession", func() error {
			_, parseErr5 := parseSrv.RefreshSession(parseCtx, &emptypb.Empty{})
			return parseErr5
		}},
		{"ListConversations", func() error {
			_, parseErr6 := parseSrv.ListConversations(parseCtx, &ListConversationsRequest{})
			return parseErr6
		}},
		{"ResolveConversationRoute", func() error {
			_, parseErr7 := parseSrv.ResolveConversationRoute(parseCtx, &ResolveConversationRouteRequest{})
			return parseErr7
		}},
		{"LoadConversation", func() error {
			_, parseErr8 := parseSrv.LoadConversation(parseCtx, &LoadConversationRequest{})
			return parseErr8
		}},
		{"DeleteConversation", func() error {
			_, parseErr9 := parseSrv.DeleteConversation(parseCtx, &DeleteConversationRequest{})
			return parseErr9
		}},
		{"SetUserName", func() error {
			_, parseErr10 := parseSrv.SetUserName(parseCtx, &SetUserNameRequest{})
			return parseErr10
		}},
		{"GetUserName", func() error {
			_, parseErr11 := parseSrv.GetUserName(parseCtx, &GetUserNameRequest{})
			return parseErr11
		}},
		{"ListUserMemories", func() error {
			_, parseErr12 := parseSrv.ListUserMemories(parseCtx, &ListUserMemoriesRequest{})
			return parseErr12
		}},
		{"UpsertUserMemory", func() error {
			_, parseErr13 := parseSrv.UpsertUserMemory(parseCtx, &UpsertUserMemoryRequest{})
			return parseErr13
		}},
		{"DeleteUserMemory", func() error {
			_, parseErr14 := parseSrv.DeleteUserMemory(parseCtx, &DeleteUserMemoryRequest{})
			return parseErr14
		}},
		{"ListModelOptions", func() error {
			_, parseErr15 := parseSrv.ListModelOptions(parseCtx, &ListModelOptionsRequest{})
			return parseErr15
		}},
		{"SetSelectedModel", func() error {
			_, parseErr16 := parseSrv.SetSelectedModel(parseCtx, wrapperspb.String("x"))
			return parseErr16
		}},
		{"GetSelectedModel", func() error {
			_, parseErr17 := parseSrv.GetSelectedModel(parseCtx, &emptypb.Empty{})
			return parseErr17
		}},
		{"SetSelectedTone", func() error {
			_, parseErr18 := parseSrv.SetSelectedTone(parseCtx, wrapperspb.String("x"))
			return parseErr18
		}},
		{"GetSelectedTone", func() error { _, parseErr19 := parseSrv.GetSelectedTone(parseCtx, &emptypb.Empty{}); return parseErr19 }},
		{"SetSelectedThinkingEnabled", func() error {
			_, parseErr20 := parseSrv.SetSelectedThinkingEnabled(parseCtx, wrapperspb.Bool(true))
			return parseErr20
		}},
		{"GetSelectedThinkingEnabled", func() error {
			_, parseErr21 := parseSrv.GetSelectedThinkingEnabled(parseCtx, &emptypb.Empty{})
			return parseErr21
		}},
		{"SetSelectedThinkingEffort", func() error {
			_, parseErr22 := parseSrv.SetSelectedThinkingEffort(parseCtx, wrapperspb.String("low"))
			return parseErr22
		}},
		{"GetSelectedThinkingEffort", func() error {
			_, parseErr23 := parseSrv.GetSelectedThinkingEffort(parseCtx, &emptypb.Empty{})
			return parseErr23
		}},
		{"SetCustomSystemPrompt", func() error {
			_, parseErr24 := parseSrv.SetCustomSystemPrompt(parseCtx, wrapperspb.String("x"))
			return parseErr24
		}},
		{"GetCustomSystemPrompt", func() error {
			_, parseErr25 := parseSrv.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{})
			return parseErr25
		}},
	}

	for _, parseTc := range parseUnaryChecks {
		parseT.Run(parseTc.name, func(parseT2 *testing.T) {
			if parseCode := status.Code(parseTc.call()); parseCode != codes.Unimplemented {
				parseT2.Fatalf("expected unimplemented for %s, got %s", parseTc.name, parseCode)
			}
		})
	}

	parseStreamChecks := []struct {
		name string
		call func() error
	}{
		{"Send", func() error {
			return parseSrv.Send(&SendRequest{}, &grpc.GenericServerStream[SendRequest, ChatChunk]{ServerStream: &fakeServerStream{}})
		}},
		{"SynthesizeSpeech", func() error {
			return parseSrv.SynthesizeSpeech(&SynthesizeSpeechRequest{}, &grpc.GenericServerStream[SynthesizeSpeechRequest, SynthesizeSpeechChunk]{ServerStream: &fakeServerStream{}})
		}},
	}

	for _, parseTc2 := range parseStreamChecks {
		parseT.Run(parseTc2.name, func(parseT3 *testing.T) {
			if parseCode2 := status.Code(parseTc2.call()); parseCode2 != codes.Unimplemented {
				parseT3.Fatalf("expected unimplemented for %s, got %s", parseTc2.name, parseCode2)
			}
		})
	}
}
