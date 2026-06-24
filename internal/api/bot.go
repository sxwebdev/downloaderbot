package api

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/media"
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

func (s *grpcServer) GetMedia(ctx context.Context, req *pb.GetMediaRequest) (*pb.GetMediaResponse, error) {
	// get link info
	linkInfo, err := s.parserService.GetLinkInfo(ctx, req.GetUrl())
	if err != nil {
		return nil, fmt.Errorf("get link info error: %w", err)
	}

	// get media data from link
	data, err := s.parserService.GetMedia(ctx, linkInfo)
	if err != nil {
		return nil, err
	}

	// define response
	resp := &pb.GetMediaResponse{
		Title:   data.Title,
		Caption: data.Caption,
		Source:  string(data.Source),
		Items:   make([]*pb.MediaItem, len(data.Items)),
	}

	// set pb media items. The API returns URLs only; for items that need
	// download headers (e.g. TikTok) the raw URL is still returned but is not
	// directly fetchable by clients — see README "Known limitations".
	for index, item := range data.Items {
		url, ok := media.Default().DirectURL(item)
		if !ok {
			url = item.Url
		}
		resp.Items[index] = &pb.MediaItem{
			Url:  url,
			Type: string(item.Type),
		}
	}

	return resp, nil
}
