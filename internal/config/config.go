package config

import (
	"github.com/sxwebdev/downloaderbot/pkg/s3"
	"github.com/tkcrm/mx/launcher/ops"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

// Config ...
type Config struct {
	Log                 logger.Config
	Ops                 ops.Config
	Grpc                grpc_transport.Config
	S3                  s3.Config
	S3BaseUrl           string `env:"S3_BASE_URL" validate:"required" default:"http://localhost:9050"`
	TelegramBotApiToken string `validate:"required" secret:"true" usage:"use token for your telegram bot"`
}
