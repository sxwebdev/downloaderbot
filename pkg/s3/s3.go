package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/mx/logger"
)

// S3 ...
type S3 struct {
	logger     logger.Logger
	cfg        *Config
	svc        *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

// New ...
func New(ctx context.Context, l logger.Logger, cfg Config) (*S3, error) {
	s := &S3{
		logger: l,
		cfg:    &cfg,
	}

	customEndpointResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		}

		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessID, cfg.SecretKey, cfg.Token)),
		config.WithEndpointResolverWithOptions(customEndpointResolver),
		//config.WithLogger(logger.New(logger.WithConsoleColored(true), logger.WithLogFormat(logger.FORMAT_CONSOLE))),
		//config.WithClientLogMode(aws.LogRetries|aws.LogRequest),
	)
	if err != nil {
		return nil, err
	}

	s.svc = s3.NewFromConfig(awsCfg)
	s.uploader = manager.NewUploader(s.svc)
	s.downloader = manager.NewDownloader(s.svc)

	return s, nil
}

type ListItem struct {
	Path         string
	LastModified *time.Time
	Size         *int64
}

// List ...
func (s *S3) List(ctx context.Context, bucketName string) ([]ListItem, error) {
	if bucketName == "" {
		return nil, ErrEmptyBucketName
	}

	res, err := s.svc.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		MaxKeys: utils.Pointer(int32(2000)),
	})
	if err != nil {
		return nil, fmt.Errorf("ListObjects error: %w", err)
	}

	items := []ListItem{}
	for _, value := range res.Contents {
		if value.Key == nil || *value.Key == "" {
			continue
		}

		items = append(items, ListItem{
			Path:         *value.Key,
			LastModified: value.LastModified,
			Size:         value.Size,
		})
	}

	return items, nil
}

// Upload file to s3 bucket
func (s *S3) Upload(ctx context.Context, bucketName, filePath string, content []byte) (string, error) {
	if bucketName == "" {
		return "", ErrEmptyBucketName
	}

	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		return "", ErrEmptyFilePath
	}

	if len(content) == 0 {
		return "", errors.New("empty file content")
	}

	result, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filePath),
		Body:   bytes.NewBuffer(content),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if result.Key == nil {
		return "", fmt.Errorf("received empty s3 file key")
	}

	return *result.Key, nil
}

// Upload file to s3 bucket
func (s *S3) UploadStream(ctx context.Context, bucketName, filePath string, reader io.Reader) (string, error) {
	if bucketName == "" {
		return "", ErrEmptyBucketName
	}

	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		return "", ErrEmptyFilePath
	}

	result, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filePath),
		Body:   reader,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if result.Key == nil {
		return "", fmt.Errorf("received empty s3 file key")
	}

	return *result.Key, nil
}

// Download file from s3 bucket
func (s *S3) Download(ctx context.Context, bucketName, filePath string) ([]byte, error) {
	if bucketName == "" {
		return nil, ErrEmptyBucketName
	}

	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		return nil, ErrEmptyFilePath
	}

	buff := manager.NewWriteAtBuffer([]byte{})
	if _, err := s.downloader.Download(ctx, buff, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filePath),
	}); err != nil {
		return nil, fmt.Errorf("failed to download file, %v", err)
	}

	return buff.Bytes(), nil
}

// Delete - Delete directory with files from s3 bucket
func (s *S3) Delete(ctx context.Context, bucketName string, filePaths []string) error {
	if bucketName == "" {
		return ErrEmptyBucketName
	}

	if len(filePaths) == 0 {
		return errors.New("empty filePaths array")
	}

	var objectIds []types.ObjectIdentifier
	for _, filePath := range filePaths {
		filePath = strings.TrimPrefix(filePath, "/")

		if filePath == "" {
			return ErrEmptyFilePath
		}

		objectIds = append(objectIds, types.ObjectIdentifier{
			Key: aws.String(filePath),
		})
	}

	delResp, err := s.svc.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &types.Delete{Objects: objectIds},
	})
	if err != nil {
		return fmt.Errorf("DeleteObjects error: %w", err)
	}

	for _, f := range filePaths {
		var exist bool
		for _, delF := range delResp.Deleted {
			if delF.Key != nil && *delF.Key == f {
				exist = true
				break
			}
		}
		if !exist {
			return fmt.Errorf("file \"%s\" was not deleted", f)
		}
	}

	return nil
}

// Download file from s3 bucket
func (s *S3) Exists(ctx context.Context, bucketName, filePath string) (bool, error) {
	if bucketName == "" {
		return false, ErrEmptyBucketName
	}

	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		return false, ErrEmptyFilePath
	}

	_, err := s.svc.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filePath),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3) ListBuckets(ctx context.Context) ([]types.Bucket, error) {
	result, err := s.svc.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("couldn't list buckets for your account. Here's why: %v", err)
	}

	return result.Buckets, err
}

func (s *S3) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if bucketName == "" {
		return false, ErrEmptyBucketName
	}

	exists := true

	_, err := s.svc.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				exists = false
				err = nil
			}
		}
	}

	return exists, err
}

func (s *S3) CreateBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		return ErrEmptyBucketName
	}

	_, err := s.svc.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.cfg.Region),
		},
	})
	if err != nil {
		return fmt.Errorf("couldn't create bucket %v in Region %v. Here's why: %v",
			bucketName, s.cfg.Region, err,
		)
	}

	return nil
}

func (s *S3) DeleteBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		return ErrEmptyBucketName
	}

	if _, err := s.svc.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		return fmt.Errorf("couldn't delete bucket %v. Here's why: %v", bucketName, err)
	}

	return nil
}
