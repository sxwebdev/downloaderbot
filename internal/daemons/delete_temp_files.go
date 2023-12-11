package daemons

import (
	"context"
	"time"

	"github.com/tkcrm/modules/pkg/taskmanager"
)

// this task delete temp files from s3 storage
func (s *daemons) deleteTempFilesTask(ctx context.Context, t *taskmanager.Task) error {
	if err := s.deleteTempFilesTaskHandler(ctx, t.Payload()); err != nil {
		s.logger.Errorf("daemon: \"%s\" error: %v", t.Type, err)
		return err
	}

	return nil
}

func (s *daemons) deleteTempFilesTaskHandler(ctx context.Context, payloadBytes []byte) error {
	files, err := s.filesService.List(ctx, s.config.S3.BucketName)
	if err != nil {
		return err
	}

	deleteLastData := time.Now().Add(-time.Minute * 10)

	filesToDelete := []string{}
	for _, file := range files {
		if file.LastModified == nil {
			continue
		}

		if deleteLastData.Before(*file.LastModified) {
			continue
		}

		filesToDelete = append(filesToDelete, file.Path)
	}

	if len(filesToDelete) > 0 {
		if err := s.filesService.Delete(ctx, s.config.S3.BucketName, filesToDelete); err != nil {
			return err
		}

		s.logger.Infof("successfully deleted %d temp files", len(filesToDelete))
	}

	return nil
}
