package api

import (
	"context"

	"github.com/sxwebdev/downloaderbot/internal/services/instagram"
	"github.com/sxwebdev/downloaderbot/pb"
	"google.golang.org/grpc"
)

type grpcServer struct {
	name string

	instagramService *instagram.Service
	pb.UnimplementedBotServiceServer
}

func NewBotGrpcServer(instagramService *instagram.Service) *grpcServer {
	return &grpcServer{name: "bot-server", instagramService: instagramService}
}

// Name of the service
func (s *grpcServer) Name() string { return s.name }

// Register service on grpc.Server
func (s *grpcServer) Register(srv *grpc.Server) {
	pb.RegisterBotServiceServer(srv, s)
}

func (s *grpcServer) GetInstagramLink(ctx context.Context, req *pb.GetInstagramLinkRequest) (*pb.GetInstagramLinkResponse, error) {
	code, err := s.instagramService.ExtractShortcodeFromLink(req.Link)
	if err != nil {
		return nil, err
	}

	resp, err := s.instagramService.GetPostWithCode(code)
	if err != nil {
		return nil, err
	}

	return &pb.GetInstagramLinkResponse{
		Link: resp.Url,
	}, nil
}
