package parser

import (
	"context"

	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

const serviceName = "parser-service"

type Service struct {
	logger logger.Logger
	config *config.Config
	name   string
}

func New(l logger.Logger, cfg *config.Config) *Service {
	return &Service{
		logger: logger.With(l, "service", serviceName),
		config: cfg,
		name:   serviceName,
	}
}

func (s Service) Name() string { return s.name }

func (s *Service) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s Service) Stop(ctx context.Context) error { return nil }

var _ service.IService = (*Service)(nil)
