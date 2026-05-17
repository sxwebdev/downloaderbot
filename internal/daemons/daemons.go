package daemons

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/services/files"
	"github.com/tkcrm/mx/logger"
)

type Daemons struct {
	logger logger.Logger
	config *config.Config
	cron   *cron.Cron

	filesService files.IFiles
}

func New(
	l logger.Logger,
	c *config.Config,
	fileService files.IFiles,
) *Daemons {
	return &Daemons{
		logger:       l,
		config:       c,
		filesService: fileService,
		cron:         cron.New(cron.WithSeconds()),
	}
}

func (s *Daemons) Name() string { return "daemons" }

func (s *Daemons) Start(ctx context.Context) error {
	// Delete temp files
	if _, err := s.cron.AddFunc("@every 1m", func() {
		if err := s.deleteTempFilesTaskHandler(ctx); err != nil {
			s.logger.Errorf("daemon: \"delete_temp_files\" error: %v", err)
		}
	}); err != nil {
		return err
	}

	// Start cron
	s.cron.Start()

	<-ctx.Done()

	return nil
}

func (s *Daemons) Stop(_ context.Context) error {
	if s.cron == nil {
		return nil
	}

	ctx := s.cron.Stop()

	// Wait for all running jobs to complete
	<-ctx.Done()

	return nil
}
