package daemons

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/constants"
	"github.com/sxwebdev/downloaderbot/internal/services/files"
	"github.com/tkcrm/modules/pkg/taskmanager"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

type IDaemons service.IService

type daemons struct {
	logger logger.Logger
	config *config.Config
	cron   *cron.Cron
	tm     taskmanager.ITaskmanager

	// Services
	filesService files.IFiles
}

func New(
	l logger.Logger,
	c *config.Config,
	tm taskmanager.ITaskmanager,
	fileService files.IFiles,
) IDaemons {
	return &daemons{
		logger:       l,
		config:       c,
		tm:           tm,
		filesService: fileService,
		cron:         cron.New(cron.WithSeconds()),
	}
}

func (s *daemons) initTaskManagerHandlers() error {
	if err := s.tm.RegisterWorkerHandlers(taskmanager.WorkerHandlers{
		constants.TaskDeleteTempFiles: s.deleteTempFilesTask,
	}); err != nil {
		return err
	}

	return nil
}

func (s *daemons) Name() string { return "daemons" }

func (s *daemons) Start(ctx context.Context) error {
	// Register task handlers
	if err := s.initTaskManagerHandlers(); err != nil {
		return fmt.Errorf("RegisterWorkerHandlers error: %s", err)
	}

	// Delete temp files
	if _, err := s.cron.AddFunc("@every 1m", func() {
		if err := s.tm.AddTask(constants.TaskDeleteTempFiles, nil); err != nil {
			s.logger.Error(err)
		}
	}); err != nil {
		return err
	}

	// Start cron
	s.cron.Start()

	<-ctx.Done()

	return nil
}

func (s *daemons) Stop(_ context.Context) error {
	if s.cron == nil {
		return nil
	}

	ctx := s.cron.Stop()

	// Wait for all running jobs to complete
	<-ctx.Done()

	return nil
}
