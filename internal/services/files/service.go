package files

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/pkg/s3"
	"github.com/tkcrm/mx/logger"
)

// IFiles files interface
type IFiles interface {
	List(ctx context.Context, bucket string) ([]s3.ListItem, error)
	Download(ctx context.Context, bucket, filePath string) ([]byte, error)
	Upload(ctx context.Context, bucket, fileName string, content []byte) (string, error)
	UploadStream(ctx context.Context, bucketName, filePath string, reader io.Reader) (string, error)
	Delete(ctx context.Context, bucket string, filePaths []string) error
	Exists(ctx context.Context, bucketName, filePath string) (bool, error)
	ListBuckets(ctx context.Context) ([]types.Bucket, error)
	CreateBucket(ctx context.Context, name string) error
	BucketExists(ctx context.Context, bucket string) (bool, error)
	DeleteBucket(ctx context.Context, bucket string) error
}

type service struct {
	cfg *config.Config
}

func New(ctx context.Context, l logger.Logger, cfg *config.Config) (IFiles, error) {
	s := &service{cfg}
	return s3.New(ctx, l, s.cfg.S3)
}
