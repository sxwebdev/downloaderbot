package instagram

import (
	"context"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

const serviceName = "instagram-service"

type Service struct {
	logger    logger.Logger
	name      string
	timeStart time.Time
}

func New(l logger.Logger) *Service {
	return &Service{
		logger:    logger.With(l, "service", serviceName),
		name:      serviceName,
		timeStart: time.Now(),
	}
}

func (s Service) Name() string { return s.name }

func (s Service) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s Service) Stop(ctx context.Context) error { return nil }

var _ service.IService = (*Service)(nil)
