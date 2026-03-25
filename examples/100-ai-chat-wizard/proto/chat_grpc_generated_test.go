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

func (f *fakeClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	f.invoked = append(f.invoked, method)
	return f.invokeErr
}

func (f *fakeClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	f.streamed = append(f.streamed, method)
	if f.newStreamErr != nil {
		return nil, f.newStreamErr
	}
	return f.stream, nil
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

func (f *fakeClientStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (f *fakeClientStream) Trailer() metadata.MD         { return f.trailerMD }
func (f *fakeClientStream) Context() context.Context {
	if f.contextRef == nil {
		return context.Background()
	}
	return f.contextRef
}
func (f *fakeClientStream) CloseSend() error { return f.closeErr }
func (f *fakeClientStream) SendMsg(m interface{}) error {
	f.sendCount++
	f.lastSend = m
	return f.sendErr
}
func (f *fakeClientStream) RecvMsg(m interface{}) error {
	f.recvCount++
	if f.recvErr != nil {
		return f.recvErr
	}
	if f.recvCount > 1 {
		return io.EOF
	}
	switch chunk := m.(type) {
	case *ChatChunk:
		chunk.Done = true
	case *SynthesizeSpeechChunk:
		chunk.Done = true
		chunk.AudioChunk = []byte{1}
	}
	return nil
}

type fakeRegistrar struct {
	desc grpc.ServiceDesc
	srv  interface{}
}

func (f *fakeRegistrar) RegisterService(desc *grpc.ServiceDesc, srv interface{}) {
	f.desc = *desc
	f.srv = srv
}

type fakeServerStream struct {
	ctx      context.Context
	recvErr  error
	recvOnce bool
}

func (f *fakeServerStream) SetHeader(md metadata.MD) error  { return nil }
func (f *fakeServerStream) SendHeader(md metadata.MD) error { return nil }
func (f *fakeServerStream) SetTrailer(md metadata.MD)       {}
func (f *fakeServerStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}
func (f *fakeServerStream) SendMsg(m interface{}) error { return nil }
func (f *fakeServerStream) RecvMsg(m interface{}) error {
	if f.recvErr != nil {
		return f.recvErr
	}
	if f.recvOnce {
		return io.EOF
	}
	f.recvOnce = true
	return nil
}

func TestGeneratedChatServiceClientUnaryAndStreamMethods(t *testing.T) {
	stream := &fakeClientStream{}
	conn := &fakeClientConn{stream: stream}
	client := NewChatServiceClient(conn)
	ctx := context.Background()

	if _, err := client.Signup(ctx, &SignupRequest{}); err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if _, err := client.Login(ctx, &LoginRequest{}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := client.Logout(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := client.GetSession(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if _, err := client.RefreshSession(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	if _, err := client.ListConversations(ctx, &ListConversationsRequest{}); err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if _, err := client.ResolveConversationRoute(ctx, &ResolveConversationRouteRequest{}); err != nil {
		t.Fatalf("ResolveConversationRoute: %v", err)
	}
	if _, err := client.LoadConversation(ctx, &LoadConversationRequest{}); err != nil {
		t.Fatalf("LoadConversation: %v", err)
	}
	if _, err := client.DeleteConversation(ctx, &DeleteConversationRequest{}); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	if _, err := client.SetUserName(ctx, &SetUserNameRequest{}); err != nil {
		t.Fatalf("SetUserName: %v", err)
	}
	if _, err := client.GetUserName(ctx, &GetUserNameRequest{}); err != nil {
		t.Fatalf("GetUserName: %v", err)
	}
	if _, err := client.ListUserMemories(ctx, &ListUserMemoriesRequest{}); err != nil {
		t.Fatalf("ListUserMemories: %v", err)
	}
	if _, err := client.UpsertUserMemory(ctx, &UpsertUserMemoryRequest{}); err != nil {
		t.Fatalf("UpsertUserMemory: %v", err)
	}
	if _, err := client.DeleteUserMemory(ctx, &DeleteUserMemoryRequest{}); err != nil {
		t.Fatalf("DeleteUserMemory: %v", err)
	}
	if _, err := client.ListModelOptions(ctx, &ListModelOptionsRequest{}); err != nil {
		t.Fatalf("ListModelOptions: %v", err)
	}
	if _, err := client.SetSelectedModel(ctx, wrapperspb.String("gpt-oss-120b")); err != nil {
		t.Fatalf("SetSelectedModel: %v", err)
	}
	if _, err := client.GetSelectedModel(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetSelectedModel: %v", err)
	}
	if _, err := client.SetSelectedTone(ctx, wrapperspb.String("concise")); err != nil {
		t.Fatalf("SetSelectedTone: %v", err)
	}
	if _, err := client.GetSelectedTone(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetSelectedTone: %v", err)
	}
	if _, err := client.SetSelectedThinkingEnabled(ctx, wrapperspb.Bool(true)); err != nil {
		t.Fatalf("SetSelectedThinkingEnabled: %v", err)
	}
	if _, err := client.GetSelectedThinkingEnabled(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetSelectedThinkingEnabled: %v", err)
	}
	if _, err := client.SetSelectedThinkingEffort(ctx, wrapperspb.String("medium")); err != nil {
		t.Fatalf("SetSelectedThinkingEffort: %v", err)
	}
	if _, err := client.GetSelectedThinkingEffort(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetSelectedThinkingEffort: %v", err)
	}
	if _, err := client.SetCustomSystemPrompt(ctx, wrapperspb.String("You are practical.")); err != nil {
		t.Fatalf("SetCustomSystemPrompt: %v", err)
	}
	if _, err := client.GetCustomSystemPrompt(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetCustomSystemPrompt: %v", err)
	}

	sendStream, err := client.Send(ctx, &SendRequest{})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sendStream == nil {
		t.Fatal("Send stream should not be nil")
	}
	if _, err := sendStream.Recv(); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Send stream recv: %v", err)
	}

	speechStream, err := client.SynthesizeSpeech(ctx, &SynthesizeSpeechRequest{})
	if err != nil {
		t.Fatalf("SynthesizeSpeech: %v", err)
	}
	if speechStream == nil {
		t.Fatal("SynthesizeSpeech stream should not be nil")
	}
	if _, err := speechStream.Recv(); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("SynthesizeSpeech stream recv: %v", err)
	}
}

func TestGeneratedChatServiceClientErrorPaths(t *testing.T) {
	ctx := context.Background()

	invokeErr := errors.New("invoke failed")
	client := NewChatServiceClient(&fakeClientConn{invokeErr: invokeErr, stream: &fakeClientStream{}})
	if _, err := client.Signup(ctx, &SignupRequest{}); !errors.Is(err, invokeErr) {
		t.Fatalf("expected invoke error, got %v", err)
	}

	newStreamErr := errors.New("new stream failed")
	client = NewChatServiceClient(&fakeClientConn{newStreamErr: newStreamErr})
	if _, err := client.Send(ctx, &SendRequest{}); !errors.Is(err, newStreamErr) {
		t.Fatalf("expected new stream error for Send, got %v", err)
	}
	if _, err := client.SynthesizeSpeech(ctx, &SynthesizeSpeechRequest{}); !errors.Is(err, newStreamErr) {
		t.Fatalf("expected new stream error for SynthesizeSpeech, got %v", err)
	}

	sendErr := errors.New("send failed")
	client = NewChatServiceClient(&fakeClientConn{stream: &fakeClientStream{sendErr: sendErr}})
	if _, err := client.Send(ctx, &SendRequest{}); !errors.Is(err, sendErr) {
		t.Fatalf("expected send error, got %v", err)
	}

	closeErr := errors.New("close failed")
	client = NewChatServiceClient(&fakeClientConn{stream: &fakeClientStream{closeErr: closeErr}})
	if _, err := client.SynthesizeSpeech(ctx, &SynthesizeSpeechRequest{}); !errors.Is(err, closeErr) {
		t.Fatalf("expected close-send error, got %v", err)
	}
}

func TestRegisterChatServiceServer(t *testing.T) {
	var registrar fakeRegistrar
	RegisterChatServiceServer(&registrar, UnimplementedChatServiceServer{})
	if registrar.desc.ServiceName != "chat.v1.ChatService" {
		t.Fatalf("expected service name chat.v1.ChatService, got %q", registrar.desc.ServiceName)
	}
	if registrar.srv == nil {
		t.Fatal("expected registered server implementation")
	}
}

func TestGeneratedUnaryHandlersDecodeInterceptorAndServerPaths(t *testing.T) {
	srv := UnimplementedChatServiceServer{}
	decodeErr := errors.New("decode failed")

	for _, method := range ChatService_ServiceDesc.Methods {
		t.Run(method.MethodName+"_decode_error", func(t *testing.T) {
			_, err := method.Handler(srv, context.Background(), func(v interface{}) error { return decodeErr }, nil)
			if !errors.Is(err, decodeErr) {
				t.Fatalf("expected decode error, got %v", err)
			}
		})

		t.Run(method.MethodName+"_no_interceptor", func(t *testing.T) {
			_, err := method.Handler(srv, context.Background(), func(v interface{}) error { return nil }, nil)
			if status.Code(err) != codes.Unimplemented {
				t.Fatalf("expected unimplemented status, got %v", err)
			}
		})

		t.Run(method.MethodName+"_with_interceptor", func(t *testing.T) {
			var seenInfo bool
			_, err := method.Handler(
				srv,
				context.Background(),
				func(v interface{}) error { return nil },
				func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					seenInfo = true
					if info == nil || info.FullMethod == "" {
						t.Fatalf("expected unary info with full method for %s", method.MethodName)
					}
					return handler(ctx, req)
				},
			)
			if !seenInfo {
				t.Fatalf("expected interceptor execution for %s", method.MethodName)
			}
			if status.Code(err) != codes.Unimplemented {
				t.Fatalf("expected unimplemented status, got %v", err)
			}
		})
	}
}

func TestGeneratedStreamHandlersRecvAndServerPaths(t *testing.T) {
	srv := UnimplementedChatServiceServer{}
	recvErr := errors.New("recv failed")

	for _, stream := range ChatService_ServiceDesc.Streams {
		t.Run(stream.StreamName+"_recv_error", func(t *testing.T) {
			err := stream.Handler(srv, &fakeServerStream{recvErr: recvErr})
			if !errors.Is(err, recvErr) {
				t.Fatalf("expected recv error, got %v", err)
			}
		})

		t.Run(stream.StreamName+"_unimplemented", func(t *testing.T) {
			err := stream.Handler(srv, &fakeServerStream{})
			if status.Code(err) != codes.Unimplemented {
				t.Fatalf("expected unimplemented status, got %v", err)
			}
		})
	}
}

func TestUnimplementedChatServiceServerMethodsReturnUnimplemented(t *testing.T) {
	srv := UnimplementedChatServiceServer{}
	ctx := context.Background()

	unaryChecks := []struct {
		name string
		call func() error
	}{
		{"Signup", func() error { _, err := srv.Signup(ctx, &SignupRequest{}); return err }},
		{"Login", func() error { _, err := srv.Login(ctx, &LoginRequest{}); return err }},
		{"Logout", func() error { _, err := srv.Logout(ctx, &emptypb.Empty{}); return err }},
		{"GetSession", func() error { _, err := srv.GetSession(ctx, &emptypb.Empty{}); return err }},
		{"RefreshSession", func() error { _, err := srv.RefreshSession(ctx, &emptypb.Empty{}); return err }},
		{"ListConversations", func() error { _, err := srv.ListConversations(ctx, &ListConversationsRequest{}); return err }},
		{"ResolveConversationRoute", func() error { _, err := srv.ResolveConversationRoute(ctx, &ResolveConversationRouteRequest{}); return err }},
		{"LoadConversation", func() error { _, err := srv.LoadConversation(ctx, &LoadConversationRequest{}); return err }},
		{"DeleteConversation", func() error { _, err := srv.DeleteConversation(ctx, &DeleteConversationRequest{}); return err }},
		{"SetUserName", func() error { _, err := srv.SetUserName(ctx, &SetUserNameRequest{}); return err }},
		{"GetUserName", func() error { _, err := srv.GetUserName(ctx, &GetUserNameRequest{}); return err }},
		{"ListUserMemories", func() error { _, err := srv.ListUserMemories(ctx, &ListUserMemoriesRequest{}); return err }},
		{"UpsertUserMemory", func() error { _, err := srv.UpsertUserMemory(ctx, &UpsertUserMemoryRequest{}); return err }},
		{"DeleteUserMemory", func() error { _, err := srv.DeleteUserMemory(ctx, &DeleteUserMemoryRequest{}); return err }},
		{"ListModelOptions", func() error { _, err := srv.ListModelOptions(ctx, &ListModelOptionsRequest{}); return err }},
		{"SetSelectedModel", func() error { _, err := srv.SetSelectedModel(ctx, wrapperspb.String("x")); return err }},
		{"GetSelectedModel", func() error { _, err := srv.GetSelectedModel(ctx, &emptypb.Empty{}); return err }},
		{"SetSelectedTone", func() error { _, err := srv.SetSelectedTone(ctx, wrapperspb.String("x")); return err }},
		{"GetSelectedTone", func() error { _, err := srv.GetSelectedTone(ctx, &emptypb.Empty{}); return err }},
		{"SetSelectedThinkingEnabled", func() error { _, err := srv.SetSelectedThinkingEnabled(ctx, wrapperspb.Bool(true)); return err }},
		{"GetSelectedThinkingEnabled", func() error { _, err := srv.GetSelectedThinkingEnabled(ctx, &emptypb.Empty{}); return err }},
		{"SetSelectedThinkingEffort", func() error { _, err := srv.SetSelectedThinkingEffort(ctx, wrapperspb.String("low")); return err }},
		{"GetSelectedThinkingEffort", func() error { _, err := srv.GetSelectedThinkingEffort(ctx, &emptypb.Empty{}); return err }},
		{"SetCustomSystemPrompt", func() error { _, err := srv.SetCustomSystemPrompt(ctx, wrapperspb.String("x")); return err }},
		{"GetCustomSystemPrompt", func() error { _, err := srv.GetCustomSystemPrompt(ctx, &emptypb.Empty{}); return err }},
	}

	for _, tc := range unaryChecks {
		t.Run(tc.name, func(t *testing.T) {
			if code := status.Code(tc.call()); code != codes.Unimplemented {
				t.Fatalf("expected unimplemented for %s, got %s", tc.name, code)
			}
		})
	}

	streamChecks := []struct {
		name string
		call func() error
	}{
		{"Send", func() error { return srv.Send(&SendRequest{}, &grpc.GenericServerStream[SendRequest, ChatChunk]{ServerStream: &fakeServerStream{}}) }},
		{"SynthesizeSpeech", func() error {
			return srv.SynthesizeSpeech(&SynthesizeSpeechRequest{}, &grpc.GenericServerStream[SynthesizeSpeechRequest, SynthesizeSpeechChunk]{ServerStream: &fakeServerStream{}})
		}},
	}

	for _, tc := range streamChecks {
		t.Run(tc.name, func(t *testing.T) {
			if code := status.Code(tc.call()); code != codes.Unimplemented {
				t.Fatalf("expected unimplemented for %s, got %s", tc.name, code)
			}
		})
	}
}
