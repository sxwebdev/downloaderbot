package s3_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/services/files"
	"github.com/sxwebdev/downloaderbot/pkg/s3"
	"github.com/tkcrm/mx/cfg"
	"github.com/tkcrm/mx/logger"
)

func getConfig() *config.Config {
	var config config.Config
	if err := cfg.Load(&config, cfg.WithEnvPath("../../../../")); err != nil {
		logger.New().Fatalf("could not load configuration: %v", err)
	}

	return &config
}

func newAWS() (files.IFiles, error) {
	c := getConfig()
	return s3.New(
		context.Background(),
		logger.New(logger.WithConsoleColored(true), logger.WithLogFormat(logger.LoggerFormatConsole)),
		c.S3,
	)
}

func Test_ListBuckets(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	result, err := s3.ListBuckets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(result)
}

func Test_List(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	result, err := s3.List(context.Background(), "backups")
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(result)
}

func Test_Upload(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.ReadFile("/Users/username/Downloads/voucher_6661575.pdf")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s3.Upload(context.Background(), "backup.new", "test/voucher_6661575.pdf", f); err != nil {
		t.Fatal(err)
	}
}

func Test_Download(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	res, err := s3.Download(context.Background(), "backup.new", "test/voucher_6661575.pdf")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(res))
}

func Test_Delete(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	filesToDelete := []string{
		"test/folder/asdf.asdf/photo_2022-12-21 19.58.37.jpeg",
		"photo_2022-12-21 19.58.37.jpeg",
	}

	if err := s3.Delete(context.Background(), "tkcrm-old", filesToDelete); err != nil {
		t.Fatal(err)
	}
}

func Test_CreateBucket(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	if err := s3.CreateBucket(context.Background(), "test-test"); err != nil {
		t.Fatal(err)
	}
}

func Test_BucketExists(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	exists, err := s3.BucketExists(context.Background(), "test-test")
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(exists)
}

func Test_DeleteBucket(t *testing.T) {
	s3, err := newAWS()
	if err != nil {
		t.Fatal(err)
	}

	if err := s3.DeleteBucket(context.Background(), "test-test"); err != nil {
		t.Fatal(err)
	}
}
