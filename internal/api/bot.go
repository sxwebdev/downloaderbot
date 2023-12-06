package api

import (
	"context"

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
	// extract media code from url
	code, err := instagram.ExtractShortcodeFromLink(req.Url)
	if err != nil {
		return nil, err
	}

	// get media data from instagram
	data, err := instagram.GetPostWithCode(ctx, code)
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
			IsVideo: item.IsVideo,
		}
	}

	return resp, nil
}
