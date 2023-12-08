package api

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/pb"
	"google.golang.org/grpc"
)

type grpcServer struct {
	name          string
	parserService *parser.Service

	pb.UnimplementedBotServiceServer
}

func NewBotGrpcServer(parserService *parser.Service) *grpcServer {
	return &grpcServer{name: "bot-server", parserService: parserService}
}

// Name of the service
func (s *grpcServer) Name() string { return s.name }

// Register service on grpc.Server
func (s *grpcServer) Register(srv *grpc.Server) {
	pb.RegisterBotServiceServer(srv, s)
}

func (s *grpcServer) GetMediaFromInstagram(ctx context.Context, req *pb.GetMediaFromInstagramRequest) (*pb.GetMediaFromInstagramResponse, error) {
	// get link info
	linkInfo, err := s.parserService.GetLinkInfo(req.GetUrl())
	if err != nil {
		return nil, fmt.Errorf("get link info error: %w", err)
	}

	// get media data from link
	data, err := s.parserService.GetMedia(ctx, linkInfo)
	if err != nil {
		return nil, err
	}

	// define response
	resp := &pb.GetMediaFromInstagramResponse{
		Caption: data.Caption,
		Items:   make([]*pb.MediaItem, len(data.Items)),
	}

	// set pb media items
	for index, item := range data.Items {
		resp.Items[index] = &pb.MediaItem{
			Url:     item.Url,
			IsVideo: item.Type == models.MediaTypeVideo,
		}
	}

	return resp, nil
}
