package api

import (
	"github.com/sxwebdev/downloaderbot/pb"
	"google.golang.org/grpc"
)

type grpcServer struct {
	name string

	pb.UnimplementedAuthorServiceServer
}

func NewBotGrpcServer() *grpcServer {
	return &grpcServer{name: "bot-server"}
}

// Name of the service
func (s *grpcServer) Name() string { return s.name }

// Register service on grpc.Server
func (s *grpcServer) Register(srv *grpc.Server) {
	pb.RegisterAuthorServiceServer(srv, s)
}
