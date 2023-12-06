package instagram

import (
	"context"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

const serviceName = "instagram-service"

type books struct {
	logger    logger.Logger
	name      string
	timeStart time.Time
}

func New(l logger.Logger) *books {
	return &books{
		logger:    logger.With(l, "service", serviceName),
		name:      serviceName,
		timeStart: time.Now(),
	}
}

func (s books) Name() string { return s.name }

func (s books) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s books) Stop(ctx context.Context) error { return nil }

var _ service.IService = (*books)(nil)
