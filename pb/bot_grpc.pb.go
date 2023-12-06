// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.1
// source: proto/bot.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	BotService_GetMediaFromInstagram_FullMethodName = "/bot.BotService/GetMediaFromInstagram"
)

// BotServiceClient is the client API for BotService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BotServiceClient interface {
	GetMediaFromInstagram(ctx context.Context, in *GetMediaFromInstagramRequest, opts ...grpc.CallOption) (*GetMediaFromInstagramResponse, error)
}

type botServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBotServiceClient(cc grpc.ClientConnInterface) BotServiceClient {
	return &botServiceClient{cc}
}

func (c *botServiceClient) GetMediaFromInstagram(ctx context.Context, in *GetMediaFromInstagramRequest, opts ...grpc.CallOption) (*GetMediaFromInstagramResponse, error) {
	out := new(GetMediaFromInstagramResponse)
	err := c.cc.Invoke(ctx, BotService_GetMediaFromInstagram_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BotServiceServer is the server API for BotService service.
// All implementations must embed UnimplementedBotServiceServer
// for forward compatibility
type BotServiceServer interface {
	GetMediaFromInstagram(context.Context, *GetMediaFromInstagramRequest) (*GetMediaFromInstagramResponse, error)
	mustEmbedUnimplementedBotServiceServer()
}

// UnimplementedBotServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBotServiceServer struct {
}

func (UnimplementedBotServiceServer) GetMediaFromInstagram(context.Context, *GetMediaFromInstagramRequest) (*GetMediaFromInstagramResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMediaFromInstagram not implemented")
}
func (UnimplementedBotServiceServer) mustEmbedUnimplementedBotServiceServer() {}

// UnsafeBotServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BotServiceServer will
// result in compilation errors.
type UnsafeBotServiceServer interface {
	mustEmbedUnimplementedBotServiceServer()
}

func RegisterBotServiceServer(s grpc.ServiceRegistrar, srv BotServiceServer) {
	s.RegisterService(&BotService_ServiceDesc, srv)
}

func _BotService_GetMediaFromInstagram_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetMediaFromInstagramRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BotServiceServer).GetMediaFromInstagram(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BotService_GetMediaFromInstagram_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BotServiceServer).GetMediaFromInstagram(ctx, req.(*GetMediaFromInstagramRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BotService_ServiceDesc is the grpc.ServiceDesc for BotService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BotService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "bot.BotService",
	HandlerType: (*BotServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetMediaFromInstagram",
			Handler:    _BotService_GetMediaFromInstagram_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/bot.proto",
}