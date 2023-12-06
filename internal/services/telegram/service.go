package telegram

import (
	"context"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

const serviceName = "telegram-service"

type Service struct {
	logger logger.Logger
	name   string
}

func New(l logger.Logger) *Service {
	return &Service{
		logger: logger.With(l, "service", serviceName),
		name:   serviceName,
	}
}

func (s Service) Name() string { return s.name }

func (s Service) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s Service) Stop(ctx context.Context) error { return nil }

var _ service.IService = (*Service)(nil)
