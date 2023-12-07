package s3

import "errors"

var (
	ErrEmptyBucketName = errors.New("empty bucket name")
	ErrEmptyFilePath   = errors.New("empty file path")
)
