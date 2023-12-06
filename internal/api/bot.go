package api

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/pb"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
	"google.golang.org/grpc"
)

type grpcServer struct {
	name string

	pb.UnimplementedBotServiceServer
}

func NewBotGrpcServer() *grpcServer {
	return &grpcServer{name: "bot-server"}
}

// Name of the service
func (s *grpcServer) Name() string { return s.name }

// Register service on grpc.Server
func (s *grpcServer) Register(srv *grpc.Server) {
	pb.RegisterBotServiceServer(srv, s)
}

func (s *grpcServer) GetMediaFromInstagram(ctx context.Context, req *pb.GetMediaFromInstagramRequest) (*pb.GetMediaFromInstagramResponse, error) {
	if req.Url == "" {
		return nil, fmt.Errorf("empty url")
	}

	code, err := instagram.ExtractShortcodeFromLink(req.Url)
	if err != nil {
		return nil, err
	}

	data, err := instagram.GetPostWithCode(code)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetMediaFromInstagramResponse{
		Caption: data.Caption,
		Items:   make([]*pb.MediaItem, len(data.Items)),
	}

	for index, item := range data.Items {
		resp.Items[index] = &pb.MediaItem{
			Url:     item.Url,
			IsVideo: item.IsVideo,
		}
	}

	return resp, nil
}
